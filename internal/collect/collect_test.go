package collect

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/kigmixelson/recollect/internal/query"
	"github.com/kigmixelson/recollect/internal/saymon"
)

type fakeClient struct {
	children []saymon.Object
	history  map[string][]saymon.MetricHistory
	childErr error
	histErr  error
}

func (f *fakeClient) Children(context.Context, string, string, string) ([]saymon.Object, error) {
	return f.children, f.childErr
}

func (f *fakeClient) History(_ context.Context, _, _, objectID string, _ []string, _, _ time.Time, _ string) ([]saymon.MetricHistory, error) {
	if f.histErr != nil {
		return nil, f.histErr
	}
	return f.history[objectID], nil
}

func TestServiceCollectMatchesAndAggregates(t *testing.T) {
	t.Parallel()
	now := time.Now()
	cli := &fakeClient{
		children: []saymon.Object{
			{ID: "c1", Name: "ИКЗ 1", MetricsCache: []string{"topic", "message.I", "message.Temp"}},
			{ID: "c2", Name: "ИКЗ 2", MetricsCache: []string{"topic", "message.I"}},
			{ID: "c3", Name: "ИКЗ 3", MetricsCache: []string{"topic", "message.I"}},
			{ID: "c4", Name: "ИКЗ 4", MetricsCache: []string{"topic", "message.E"}},
		},
		history: map[string][]saymon.MetricHistory{
			"c1": {{Metric: "message.I", Dps: saymon.DataPoints{{Timestamp: 1, Value: 10}, {Timestamp: 9, Value: 52.625}}}},
			"c2": {{Metric: "message.I", Dps: saymon.DataPoints{{Timestamp: 5, Value: 42.25}}}},
			"c3": {{Metric: "message.I", Dps: saymon.DataPoints{{Timestamp: 4, Value: 37}}}},
		},
	}

	svc := New(cli, 2)
	res, err := svc.Collect(context.Background(), query.CollectParams{
		Host:       "https://saymon.local",
		Token:      "tok",
		ObjectID:   "parent",
		Metrics:    []string{"message.I", "message.Temp"},
		Aggregates: []string{query.AggAvg, query.AggMin, query.AggMax},
		DepthRaw:   "12h",
		From:       now.Add(-12 * time.Hour),
		To:         now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].ID != "c4" {
		t.Fatalf("skipped: %+v", res.Skipped)
	}
	if len(res.Samples) != 3 {
		t.Fatalf("samples: %+v", res.Samples)
	}
	assertFloat(t, res.Values["message.I"]["avg"], (52.625+42.25+37)/3)
	assertFloat(t, res.Values["message.I"]["min"], 37)
	assertFloat(t, res.Values["message.I"]["max"], 52.625)
	if len(res.Values["message.Temp"]) != 0 {
		t.Fatalf("empty metric should be {}, got %v", res.Values["message.Temp"])
	}
}

func TestServiceCollectEmptyWindow(t *testing.T) {
	t.Parallel()
	now := time.Now()
	cli := &fakeClient{
		children: []saymon.Object{
			{ID: "c1", Name: "ИКЗ 1", MetricsCache: []string{"message.I"}},
		},
		history: map[string][]saymon.MetricHistory{
			"c1": {{Metric: "message.I", Dps: nil}},
		},
	}
	res, err := New(cli, 1).Collect(context.Background(), query.CollectParams{
		Host: "https://saymon.local", Token: "tok", ObjectID: "parent",
		Metrics: []string{"message.I"}, Aggregates: []string{query.AggAvg},
		From: now.Add(-12 * time.Hour), To: now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Samples) != 0 {
		t.Fatalf("samples: %+v", res.Samples)
	}
	if len(res.Values["message.I"]) != 0 {
		t.Fatalf("empty window should be {}, got %v", res.Values["message.I"])
	}
}

func TestServiceCollectChildrenError(t *testing.T) {
	t.Parallel()
	svc := New(&fakeClient{childErr: fmt.Errorf("boom")}, 1)
	_, err := svc.Collect(context.Background(), query.CollectParams{
		Host: "h", Token: "t", ObjectID: "o",
		Metrics: []string{"m"}, Aggregates: []string{query.AggAvg},
		From: time.Now().Add(-time.Hour), To: time.Now(),
	})
	if err == nil {
		t.Fatal("expected error")
	}
}

func assertFloat(t *testing.T, got *float64, want float64) {
	t.Helper()
	if got == nil || *got != want {
		t.Fatalf("got %v want %v", got, want)
	}
}
