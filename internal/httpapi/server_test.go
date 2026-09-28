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
			"c1|all-max": {{Metric: "message.I", Dps: saymon.DataPoints{{Value: 9}}}},
		},
	}
	handler := New(collect.New(cli, 1), nil)

	req := httptest.NewRequest(http.MethodGet, "/api/collect?host=saymon.local&token=abc&object_id=parent&metrics=message.I&aggregates=max&depth=3m", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d body %s", rec.Code, rec.Body.String())
	}

	var payload collect.Result
	if err := json.Unmarshal(rec.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if len(payload.Results) != 1 || payload.Results[0].Values["message.I"]["max"] == nil {
		t.Fatalf("payload: %+v", payload)
	}
	if *payload.Results[0].Values["message.I"]["max"] != 9 {
		t.Fatalf("value: %+v", payload.Results[0].Values)
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
			"c1|all-avg": {{Metric: "message.I", Dps: saymon.DataPoints{{Value: 3}}}},
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
			"c1|all-avg": {{Metric: "message.I", Dps: saymon.DataPoints{{Value: 3}}}},
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

func (f *fakeClient) History(_ context.Context, _, _, objectID string, _ []string, _, _ time.Time, downsample string) ([]saymon.MetricHistory, error) {
	return f.history[objectID+"|"+downsample], nil
}
