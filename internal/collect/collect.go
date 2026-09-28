package collect

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/kigmixelson/recollect/internal/query"
	"github.com/kigmixelson/recollect/internal/saymon"
)

type Client interface {
	Children(ctx context.Context, host, token, objectID string) ([]saymon.Object, error)
	History(ctx context.Context, host, token, objectID string, metrics []string, from, to time.Time, downsample string) ([]saymon.MetricHistory, error)
}

type Service struct {
	client      Client
	concurrency int
}

func New(client Client, concurrency int) *Service {
	if concurrency <= 0 {
		concurrency = 8
	}
	return &Service{client: client, concurrency: concurrency}
}

type Result struct {
	Host       string                         `json:"host"`
	ObjectID   string                         `json:"object_id"`
	Depth      string                         `json:"depth"`
	From       int64                          `json:"from"`
	To         int64                          `json:"to"`
	Metrics    []string                       `json:"metrics"`
	Aggregates []string                       `json:"aggregates"`
	Values     map[string]map[string]*float64 `json:"values"`
	Samples    []Sample                       `json:"samples,omitempty"`
	Skipped    []SkippedChild                 `json:"skipped,omitempty"`
}

type Sample struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Metric    string  `json:"metric"`
	Value     float64 `json:"value"`
	Timestamp int64   `json:"timestamp"`
}

type SkippedChild struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
}

type childPoints struct {
	child  saymon.Object
	latest map[string]saymon.DataPoint
	err    string
}

func (s *Service) Collect(ctx context.Context, p query.CollectParams) (*Result, error) {
	children, err := s.client.Children(ctx, p.Host, p.Token, p.ObjectID)
	if err != nil {
		return nil, fmt.Errorf("list children: %w", err)
	}

	out := &Result{
		Host:       p.Host,
		ObjectID:   p.ObjectID,
		Depth:      p.DepthRaw,
		From:       p.From.UnixMilli(),
		To:         p.To.UnixMilli(),
		Metrics:    p.Metrics,
		Aggregates: p.Aggregates,
		Values:     make(map[string]map[string]*float64, len(p.Metrics)),
	}
	for _, metric := range p.Metrics {
		out.Values[metric] = map[string]*float64{}
	}

	type job struct {
		child   saymon.Object
		matched []string
	}
	jobs := make([]job, 0, len(children))
	for _, child := range children {
		matched := query.Intersect(p.Metrics, child.MetricsCache)
		if len(matched) == 0 {
			out.Skipped = append(out.Skipped, SkippedChild{
				ID:     child.ID,
				Name:   child.Name,
				Reason: "no matching metrics in metrics_cache",
			})
			continue
		}
		jobs = append(jobs, job{child: child, matched: matched})
	}

	collected := make([]childPoints, len(jobs))
	sem := make(chan struct{}, s.concurrency)
	var wg sync.WaitGroup

	for i, j := range jobs {
		wg.Add(1)
		go func(idx int, item job) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
				defer func() { <-sem }()
			case <-ctx.Done():
				collected[idx] = childPoints{child: item.child, err: ctx.Err().Error()}
				return
			}
			collected[idx] = s.latestFromChild(ctx, p, item.child, item.matched)
		}(i, j)
	}
	wg.Wait()

	buckets := make(map[string]saymon.DataPoints, len(p.Metrics))
	for _, item := range collected {
		if item.err != "" {
			out.Skipped = append(out.Skipped, SkippedChild{
				ID:     item.child.ID,
				Name:   item.child.Name,
				Reason: item.err,
			})
			continue
		}
		for metric, point := range item.latest {
			out.Samples = append(out.Samples, Sample{
				ID:        item.child.ID,
				Name:      item.child.Name,
				Metric:    metric,
				Value:     point.Value,
				Timestamp: point.Timestamp,
			})
			buckets[metric] = append(buckets[metric], point)
		}
	}

	for _, metric := range p.Metrics {
		points := buckets[metric]
		if len(points) == 0 {
			continue
		}
		for _, agg := range p.Aggregates {
			out.Values[metric][agg] = saymon.Reduce(points, agg)
		}
	}

	return out, ctx.Err()
}

func (s *Service) latestFromChild(ctx context.Context, p query.CollectParams, child saymon.Object, matched []string) childPoints {
	out := childPoints{
		child:  child,
		latest: make(map[string]saymon.DataPoint, len(matched)),
	}
	history, err := s.client.History(ctx, p.Host, p.Token, child.ID, matched, p.From, p.To, "")
	if err != nil {
		out.err = "history: " + err.Error()
		return out
	}
	byMetric := make(map[string]saymon.MetricHistory, len(history))
	for _, h := range history {
		byMetric[h.Metric] = h
	}
	for _, metric := range matched {
		h, ok := byMetric[metric]
		if !ok {
			continue
		}
		last := saymon.Latest(h.Dps)
		if last == nil {
			continue
		}
		out.latest[metric] = *last
	}
	return out
}
