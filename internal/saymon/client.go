package saymon

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	http *http.Client
}

func NewClient(timeout time.Duration, tlsInsecure bool) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	transport := http.DefaultTransport.(*http.Transport).Clone()
	if tlsInsecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // optional for lab SAYMON installs
	}
	return &Client{
		http: &http.Client{
			Timeout:   timeout,
			Transport: transport,
		},
	}
}

func (c *Client) Children(ctx context.Context, host, token, objectID string) ([]Object, error) {
	var objects []Object
	path := "/node/api/objects/" + url.PathEscape(objectID) + "/children"
	if err := c.get(ctx, host, token, path, nil, &objects); err != nil {
		return nil, err
	}
	return objects, nil
}

func (c *Client) History(ctx context.Context, host, token, objectID string, metrics []string, from, to time.Time, downsample string) ([]MetricHistory, error) {
	q := url.Values{}
	q.Set("from", fmt.Sprintf("%d", from.UnixMilli()))
	q.Set("to", fmt.Sprintf("%d", to.UnixMilli()))
	if downsample != "" {
		q.Set("downsample", downsample)
	}
	joined := strings.Join(metrics, ",")
	q.Set("metrics", joined)
	for _, m := range metrics {
		q.Add("metrics[]", m)
	}

	var history []MetricHistory
	path := "/node/api/objects/" + url.PathEscape(objectID) + "/history"
	if err := c.get(ctx, host, token, path, q, &history); err != nil {
		return nil, err
	}
	return history, nil
}

func (c *Client) get(ctx context.Context, host, token, path string, extra url.Values, dest any) error {
	base, err := origin(host)
	if err != nil {
		return err
	}
	u, err := url.Parse(base)
	if err != nil {
		return fmt.Errorf("saymon host: %w", err)
	}
	u.Path = path
	q := u.Query()
	for k, vs := range extra {
		for _, v := range vs {
			q.Add(k, v)
		}
	}
	q.Set("auth-token", token)
	q.Set("api-token", token)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("saymon request %s: %w", path, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
	if err != nil {
		return fmt.Errorf("saymon read %s: %w", path, err)
	}
	if resp.StatusCode >= 300 {
		msg := strings.TrimSpace(string(body))
		if len(msg) > 512 {
			msg = msg[:512] + "..."
		}
		return fmt.Errorf("saymon %s: %s: %s", path, resp.Status, msg)
	}
	if err := json.Unmarshal(body, dest); err != nil {
		return fmt.Errorf("saymon decode %s: %w", path, err)
	}
	return nil
}

func origin(host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("host is empty")
	}
	if !strings.Contains(host, "://") {
		host = "https://" + host
	}
	u, err := url.Parse(host)
	if err != nil {
		return "", fmt.Errorf("invalid host %q: %w", host, err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", fmt.Errorf("unsupported host scheme %q", u.Scheme)
	}
	if u.Host == "" {
		return "", fmt.Errorf("invalid host %q", host)
	}
	return u.Scheme + "://" + u.Host, nil
}
