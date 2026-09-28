package query

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type CollectParams struct {
	Host       string
	Token      string
	ObjectID   string
	Metrics    []string
	Aggregates []string
	Depth      time.Duration
	DepthRaw   string
	From       time.Time
	To         time.Time
	Scheme     string
	Debug      bool
}

type collectBody struct {
	Host       string      `json:"host"`
	Hostname   string      `json:"hostname"`
	SaymonHost string      `json:"saymon_host"`
	Token      string      `json:"token"`
	ObjectID   string      `json:"object_id"`
	ObjectId   string      `json:"objectId"`
	Object     string      `json:"object"`
	ID         string      `json:"id"`
	Metrics    flexStrings `json:"metrics"`
	Aggregates flexStrings `json:"aggregates"`
	Depth      string      `json:"depth"`
	Scheme     string      `json:"scheme"`
	Debug      *flexBool   `json:"debug"`
}

type flexStrings []string

func (f *flexStrings) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || string(b) == "null" {
		*f = nil
		return nil
	}
	if b[0] == '"' {
		var one string
		if err := json.Unmarshal(b, &one); err != nil {
			return err
		}
		*f = UniqueNonEmpty([]string{one})
		return nil
	}
	var many []string
	if err := json.Unmarshal(b, &many); err != nil {
		return fmt.Errorf("metrics/aggregates must be a string or array of strings")
	}
	*f = UniqueNonEmpty(many)
	return nil
}

func ParseCollect(r *http.Request) (CollectParams, error) {
	q := r.URL.Query()
	now := time.Now()

	p := CollectParams{
		Host:     first(q, r.Header.Get("X-Saymon-Host"), "host", "hostname", "saymon_host"),
		Token:    first(q, headerToken(r), "token", "auth-token", "auth_token", "api-token", "api_token"),
		ObjectID: first(q, "", "object_id", "objectId", "object", "id"),
		Metrics:  UniqueNonEmpty(values(q, "metrics", "metric", "metrics[]")),
		DepthRaw: first(q, "", "depth", "window", "period"),
		Scheme:   first(q, "", "scheme", "proto", "protocol"),
	}

	bodyAggs := []string(nil)
	if body, err := readCollectBody(r); err != nil {
		return CollectParams{}, err
	} else if body != nil {
		if p.Host == "" {
			p.Host = firstNonEmpty(body.Host, body.Hostname, body.SaymonHost)
		}
		if p.Token == "" {
			p.Token = body.Token
		}
		if p.ObjectID == "" {
			p.ObjectID = firstNonEmpty(body.ObjectID, body.ObjectId, body.Object, body.ID)
		}
		if len(p.Metrics) == 0 {
			p.Metrics = UniqueNonEmpty([]string(body.Metrics))
		}
		if p.DepthRaw == "" {
			p.DepthRaw = body.Depth
		}
		if p.Scheme == "" {
			p.Scheme = body.Scheme
		}
		if !q.Has("debug") && body.Debug != nil {
			p.Debug = bool(*body.Debug)
		}
		bodyAggs = []string(body.Aggregates)
	}
	if q.Has("debug") {
		p.Debug = truthyFlag(q.Get("debug"))
	}

	aggSrc := values(q, "aggregates", "aggregate", "aggs", "aggregates[]")
	if len(UniqueNonEmpty(aggSrc)) == 0 {
		aggSrc = bodyAggs
	}
	aggs, err := NormalizeAggregates(aggSrc)
	if err != nil {
		return CollectParams{}, err
	}
	p.Aggregates = aggs

	if strings.TrimSpace(p.Host) == "" {
		return CollectParams{}, fmt.Errorf("host is required")
	}
	normalized, err := NormalizeHost(p.Host, p.Scheme)
	if err != nil {
		return CollectParams{}, err
	}
	p.Host = normalized
	if strings.TrimSpace(p.Token) == "" {
		return CollectParams{}, fmt.Errorf("token is required")
	}
	if strings.TrimSpace(p.ObjectID) == "" {
		return CollectParams{}, fmt.Errorf("object_id is required")
	}
	if len(p.Metrics) == 0 {
		return CollectParams{}, fmt.Errorf("metrics is required")
	}

	depth, err := ParseDepth(p.DepthRaw)
	if err != nil {
		return CollectParams{}, err
	}
	p.Depth = depth
	p.To = now
	p.From = now.Add(-depth)
	return p, nil
}

func readCollectBody(r *http.Request) (*collectBody, error) {
	if r.Body == nil || r.Method == http.MethodGet || r.Method == http.MethodHead {
		return nil, nil
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}
	raw = bytes.TrimSpace(raw)
	if len(raw) == 0 {
		return nil, nil
	}
	if raw[0] != '{' {
		return nil, fmt.Errorf("body must be a JSON object")
	}
	var body collectBody
	if err := json.Unmarshal(raw, &body); err != nil {
		return nil, fmt.Errorf("invalid JSON body: %w", err)
	}
	return &body, nil
}

func headerToken(r *http.Request) string {
	if v := r.Header.Get("X-Saymon-Token"); v != "" {
		return v
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
		return strings.TrimSpace(auth[7:])
	}
	return ""
}

func values(q url.Values, keys ...string) []string {
	var out []string
	for _, key := range keys {
		out = append(out, q[key]...)
	}
	return out
}

func first(q url.Values, header string, keys ...string) string {
	if strings.TrimSpace(header) != "" {
		return strings.TrimSpace(header)
	}
	for _, key := range keys {
		if v := strings.TrimSpace(q.Get(key)); v != "" {
			return v
		}
	}
	return ""
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

type flexBool bool

func (f *flexBool) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*f = false
		return nil
	}
	switch b[0] {
	case 't', 'f':
		var v bool
		if err := json.Unmarshal(b, &v); err != nil {
			return err
		}
		*f = flexBool(v)
		return nil
	case '"':
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*f = flexBool(truthyFlag(s))
		return nil
	default:
		var n json.Number
		if err := json.Unmarshal(b, &n); err != nil {
			return fmt.Errorf("debug must be a boolean")
		}
		*f = flexBool(n != "0")
		return nil
	}
}

func truthyFlag(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "", "1", "true", "yes", "y", "on", "debug":
		return true
	case "0", "false", "no", "n", "off":
		return false
	default:
		return true
	}
}
