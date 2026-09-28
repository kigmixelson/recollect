package httpapi

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kigmixelson/recollect/internal/collect"
	"github.com/kigmixelson/recollect/internal/saymon"
)

func TestHealthz(t *testing.T) {
	t.Parallel()
	handler := New(collect.New(&fakeClient{}, 1), nil)
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
}

func TestCollectHTTP(t *testing.T) {
	t.Parallel()
	cli := &fakeClient{
		children: []saymon.Object{{
			ID:           "c1",
			Name:         "child",
			MetricsCache: []string{"message.I"},
		}},
		history: map[string][]saymon.MetricHistory{
			"c1": {{Metric: "message.I", Dps: saymon.DataPoints{{Timestamp: 1, Value: 9}}}},
		},
	}
	handler := New(collect.New(cli, 1), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/collect?host=saymon.local&token=abc&object_id=parent&metrics=message.I&aggregates=max&depth=3m", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}

	var payload map[string]map[string]*float64
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["message.I"]["max"] == nil || *payload["message.I"]["max"] != 9 {
		t.Fatalf("payload: %s", rec.Body.String())
	}
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	if _, ok := raw["host"]; ok {
		t.Fatalf("compact body should not include metadata: %s", rec.Body.String())
	}
}

func TestCollectHTTPPostJSON(t *testing.T) {
	t.Parallel()
	cli := &fakeClient{
		children: []saymon.Object{{
			ID:           "c1",
			Name:         "child",
			MetricsCache: []string{"message.I"},
		}},
		history: map[string][]saymon.MetricHistory{
			"c1": {{Metric: "message.I", Dps: saymon.DataPoints{{Timestamp: 1, Value: 3}}}},
		},
	}
	handler := New(collect.New(cli, 1), nil)
	body := `{"host":"saymon.local","object_id":"parent","metrics":["message.I"],"aggregates":["avg"],"depth":"3m"}`
	req := httptest.NewRequest(http.MethodPost, "/api/collect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Saymon-Token", "abc")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var payload map[string]map[string]*float64
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["message.I"]["avg"] == nil || *payload["message.I"]["avg"] != 3 {
		t.Fatalf("payload: %s", rec.Body.String())
	}
}

func TestCollectHTTPPrefixed(t *testing.T) {
	t.Parallel()
	cli := &fakeClient{
		children: []saymon.Object{{
			ID:           "c1",
			Name:         "child",
			MetricsCache: []string{"message.I"},
		}},
		history: map[string][]saymon.MetricHistory{
			"c1": {{Metric: "message.I", Dps: saymon.DataPoints{{Timestamp: 1, Value: 3}}}},
		},
	}
	handler := New(collect.New(cli, 1), nil)
	req := httptest.NewRequest(http.MethodGet, "/recollect/healthz", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("healthz prefix status %d", rec.Code)
	}

	body := `{"host":"saymon.local","token":"abc","object_id":"parent","metrics":["message.I"],"aggregates":["avg"],"depth":"3m"}`
	req = httptest.NewRequest(http.MethodPost, "/recollect/api/collect", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("collect prefix status %d body %s", rec.Code, rec.Body.String())
	}
}

func TestCollectHTTPDebug(t *testing.T) {
	t.Parallel()
	cli := &fakeClient{
		children: []saymon.Object{{
			ID:           "c1",
			Name:         "child",
			MetricsCache: []string{"message.I"},
		}},
		history: map[string][]saymon.MetricHistory{
			"c1": {{Metric: "message.I", Dps: saymon.DataPoints{{Timestamp: 1, Value: 9}}}},
		},
	}
	handler := New(collect.New(cli, 1), nil)
	req := httptest.NewRequest(http.MethodGet, "/api/collect?host=saymon.local&token=abc&object_id=parent&metrics=message.I&aggregates=avg&depth=3m&debug=1", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}
	var payload collect.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.ObjectID != "parent" || len(payload.Samples) != 1 {
		t.Fatalf("debug payload: %+v", payload)
	}
	if payload.Values["message.I"]["avg"] == nil || *payload.Values["message.I"]["avg"] != 9 {
		t.Fatalf("debug values: %+v", payload.Values)
	}
}

func TestCollectHTTPBadRequest(t *testing.T) {
	t.Parallel()
	handler := New(collect.New(&fakeClient{}, 1), nil)
	req := httptest.NewRequest(http.MethodGet, "/api/collect", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("status %d", rec.Code)
	}
}

type fakeClient struct {
	children []saymon.Object
	history  map[string][]saymon.MetricHistory
}

func (f *fakeClient) Children(context.Context, string, string, string) ([]saymon.Object, error) {
	return f.children, nil
}

func (f *fakeClient) History(_ context.Context, _, _, objectID string, _ []string, _, _ time.Time, _ string) ([]saymon.MetricHistory, error) {
	return f.history[objectID], nil
}
