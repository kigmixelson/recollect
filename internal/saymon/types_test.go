package saymon

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestReduce(t *testing.T) {
	t.Parallel()
	points := DataPoints{
		{Timestamp: 1, Value: 10},
		{Timestamp: 2, Value: 2},
		{Timestamp: 3, Value: 6},
	}
	assertFloat(t, Reduce(points, "min"), 2)
	assertFloat(t, Reduce(points, "max"), 10)
	assertFloat(t, Reduce(points, "avg"), 6)
	assertFloat(t, Reduce(points, "sum"), 18)
	assertFloat(t, Reduce(points, "dev"), math.Sqrt(32.0/3.0))
	assertFloat(t, Reduce(DataPoints{{Value: 4.2}}, "dev"), 4.2)
	if Reduce(nil, "avg") != nil {
		t.Fatal("expected nil for empty series")
	}
}

func TestDataPointsUnmarshal(t *testing.T) {
	t.Parallel()

	var array MetricHistory
	if err := json.Unmarshal([]byte(`{"metric":"message.I","dps":[[100,1.5],[200,2.5]]}`), &array); err != nil {
		t.Fatal(err)
	}
	if len(array.Dps) != 2 || array.Dps[1].Value != 2.5 {
		t.Fatalf("array dps: %+v", array.Dps)
	}

	var obj MetricHistory
	if err := json.Unmarshal([]byte(`{"metric":"message.I","dps":{"200":3.5,"100":1.5}}`), &obj); err != nil {
		t.Fatal(err)
	}
	if len(obj.Dps) != 2 || obj.Dps[0].Timestamp != 100 || obj.Dps[1].Value != 3.5 {
		t.Fatalf("object dps: %+v", obj.Dps)
	}
}

func TestClientChildrenAndHistory(t *testing.T) {
	t.Parallel()
	mux := http.NewServeMux()
	mux.HandleFunc("/node/api/objects/parent/children", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("auth-token") != "tok" {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
		_ = json.NewEncoder(w).Encode([]Object{{
			ID:           "child-1",
			Name:         "ИКЗ",
			MetricsCache: []string{"message.I", "topic"},
		}})
	})
	mux.HandleFunc("/node/api/objects/child-1/history", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if q.Get("metrics") != "message.I" || q.Get("downsample") != "all-avg" {
			http.Error(w, "bad query", http.StatusBadRequest)
			return
		}
		_ = json.NewEncoder(w).Encode([]MetricHistory{{
			Metric: "message.I",
			Dps:    DataPoints{{Timestamp: 1, Value: 4.2}},
		}})
	})
	srv := httptest.NewServer(mux)
	t.Cleanup(srv.Close)

	c := NewClient(2*time.Second, false)
	children, err := c.Children(context.Background(), srv.URL, "tok", "parent")
	if err != nil {
		t.Fatal(err)
	}
	if len(children) != 1 || children[0].ID != "child-1" {
		t.Fatalf("children: %+v", children)
	}

	hist, err := c.History(context.Background(), srv.URL, "tok", "child-1", []string{"message.I"}, time.Unix(0, 0), time.Unix(10, 0), "all-avg")
	if err != nil {
		t.Fatal(err)
	}
	if len(hist) != 1 || hist[0].Dps[0].Value != 4.2 {
		t.Fatalf("history: %+v", hist)
	}
}

func assertFloat(t *testing.T, got *float64, want float64) {
	t.Helper()
	if got == nil || math.Abs(*got-want) > 1e-9 {
		t.Fatalf("got %v, want %v", got, want)
	}
}
