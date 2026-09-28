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

func (f *fakeClient) History(_ context.Context, _, _, objectID string, _ []string, _, _ time.Time, downsample string) ([]saymon.MetricHistory, error) {
	if f.histErr != nil {
		return nil, f.histErr
	}
	key := objectID + "|" + downsample
	return f.history[key], nil
}

func TestServiceCollectMatchesAndAggregates(t *testing.T) {
	t.Parallel()
	now := time.Now()
	cli := &fakeClient{
		children: []saymon.Object{
			{ID: "c1", Name: "ИКЗ 1", MetricsCache: []string{"topic", "message.I", "message.Temp"}},
			{ID: "c2", Name: "ИКЗ 2", MetricsCache: []string{"topic", "message.E"}},
		},
		history: map[string][]saymon.MetricHistory{
			"c1|all-avg": {{Metric: "message.I", Dps: saymon.DataPoints{{Value: 10}, {Value: 20}}}},
			"c1|all-min": {{Metric: "message.I", Dps: saymon.DataPoints{{Value: 10}, {Value: 20}}}},
			"c1|all-sum": {{Metric: "message.I", Dps: saymon.DataPoints{{Value: 10}, {Value: 20}}}},
			"c1|all-dev": {{Metric: "message.I", Dps: saymon.DataPoints{{Value: 10}, {Value: 20}}}},
		},
	}

	svc := New(cli, 2)
	res, err := svc.Collect(context.Background(), query.CollectParams{
		Host:       "https://saymon.local",
		Token:      "tok",
		ObjectID:   "parent",
		Metrics:    []string{"message.I", "message.Temp"},
		Aggregates: []string{query.AggAvg, query.AggMin, query.AggSum, query.AggDev},
		DepthRaw:   "12h",
		From:       now.Add(-12 * time.Hour),
		To:         now,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Results) != 1 {
		t.Fatalf("results: %+v", res.Results)
	}
	if len(res.Skipped) != 1 || res.Skipped[0].ID != "c2" {
		t.Fatalf("skipped: %+v", res.Skipped)
	}
	vals := res.Results[0].Values["message.I"]
	assertFloat(t, vals["avg"], 15)
	assertFloat(t, vals["min"], 10)
	assertFloat(t, vals["sum"], 30)
	assertFloat(t, vals["dev"], 5)
	if res.Results[0].Values["message.Temp"]["avg"] != nil {
		t.Fatalf("missing metric should stay nil, got %v", res.Results[0].Values["message.Temp"]["avg"])
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
