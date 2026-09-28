package query

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestIntersect(t *testing.T) {
	t.Parallel()
	got := Intersect(
		[]string{"message.I", "missing", "message.Temp", "message.I"},
		[]string{"topic", "message.I", "message.Temp", "message.E"},
	)
	want := []string{"message.I", "message.Temp"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestNormalizeAggregates(t *testing.T) {
	t.Parallel()
	got, err := NormalizeAggregates([]string{"average", "MIN", "максимум", "avg", "сумма", "дивиация"})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{AggAvg, AggMin, AggMax, AggSum, AggDev}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}

func TestParseCollectJSONBody(t *testing.T) {
	t.Parallel()
	body := `{"host":"saymon.local","object_id":"obj1","metrics":["message.I","message.Temp"],"aggregates":"avg,min","depth":"12h"}`
	req := httptest.NewRequest(http.MethodPost, "/api/collect", strings.NewReader(body))
	req.Header.Set("X-Saymon-Token", "abc")
	p, err := ParseCollect(req)
	if err != nil {
		t.Fatal(err)
	}
	if p.Host != "saymon.local" || p.Token != "abc" || p.ObjectID != "obj1" {
		t.Fatalf("unexpected identity: %+v", p)
	}
	if len(p.Metrics) != 2 || p.Aggregates[0] != AggAvg || p.Depth != 12*time.Hour {
		t.Fatalf("parsed: %+v", p)
	}
}

func TestParseCollect(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodGet, "/api/collect?host=saymon.local&token=abc&object_id=obj1&metrics=message.I&metrics=message.Temp&aggregates=avg&aggregates=min&depth=12h", nil)
	p, err := ParseCollect(req)
	if err != nil {
		t.Fatal(err)
	}
	if p.Host != "saymon.local" || p.Token != "abc" || p.ObjectID != "obj1" {
		t.Fatalf("unexpected identity: %+v", p)
	}
	if len(p.Metrics) != 2 || p.Metrics[0] != "message.I" {
		t.Fatalf("metrics: %v", p.Metrics)
	}
	if len(p.Aggregates) != 2 || p.Aggregates[0] != AggAvg || p.Aggregates[1] != AggMin {
		t.Fatalf("aggregates: %v", p.Aggregates)
	}
	if p.Depth != 12*time.Hour {
		t.Fatalf("depth: %s", p.Depth)
	}
	if !p.To.After(p.From) {
		t.Fatalf("from/to window is invalid: %v %v", p.From, p.To)
	}
}
