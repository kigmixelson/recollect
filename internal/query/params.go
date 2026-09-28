package query

import (
	"fmt"
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
	}

	aggs, err := NormalizeAggregates(values(q, "aggregates", "aggregate", "aggs", "aggregates[]"))
	if err != nil {
		return CollectParams{}, err
	}
	p.Aggregates = aggs

	if strings.TrimSpace(p.Host) == "" {
		return CollectParams{}, fmt.Errorf("host is required")
	}
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
