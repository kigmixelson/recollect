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
	Host       string         `json:"host"`
	ObjectID   string         `json:"object_id"`
	Depth      string         `json:"depth"`
	From       int64          `json:"from"`
	To         int64          `json:"to"`
	Metrics    []string       `json:"metrics"`
	Aggregates []string       `json:"aggregates"`
	Results    []ChildResult  `json:"results"`
	Skipped    []SkippedChild `json:"skipped,omitempty"`
}

type ChildResult struct {
	ID             string                         `json:"id"`
	Name           string                         `json:"name"`
	MatchedMetrics []string                       `json:"matched_metrics"`
	Values         map[string]map[string]*float64 `json:"values"`
	Error          string                         `json:"error,omitempty"`
}

type SkippedChild struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Reason string `json:"reason"`
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
		Results:    make([]ChildResult, 0, len(children)),
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

	collected := make([]ChildResult, len(jobs))
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
				collected[idx] = ChildResult{
					ID:             item.child.ID,
					Name:           item.child.Name,
					MatchedMetrics: item.matched,
					Error:          ctx.Err().Error(),
				}
				return
			}
			collected[idx] = s.collectChild(ctx, p, item.child, item.matched)
		}(i, j)
	}
	wg.Wait()

	out.Results = collected
	return out, ctx.Err()
}

func (s *Service) collectChild(ctx context.Context, p query.CollectParams, child saymon.Object, matched []string) ChildResult {
	res := ChildResult{
		ID:             child.ID,
		Name:           child.Name,
		MatchedMetrics: matched,
		Values:         make(map[string]map[string]*float64, len(matched)),
	}
	for _, metric := range matched {
		res.Values[metric] = make(map[string]*float64, len(p.Aggregates))
	}

	var errs []string
	for _, agg := range p.Aggregates {
		history, err := s.client.History(ctx, p.Host, p.Token, child.ID, matched, p.From, p.To, query.Downsample(agg))
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", agg, err))
			continue
		}
		byMetric := make(map[string]saymon.MetricHistory, len(history))
		for _, h := range history {
			byMetric[h.Metric] = h
		}
		for _, metric := range matched {
			h, ok := byMetric[metric]
			if !ok {
				res.Values[metric][agg] = nil
				continue
			}
			res.Values[metric][agg] = saymon.Reduce(h.Dps, agg)
		}
	}
	if len(errs) > 0 {
		res.Error = fmt.Sprintf("history: %s", joinErrors(errs))
	}
	return res
}

func joinErrors(errs []string) string {
	if len(errs) == 1 {
		return errs[0]
	}
	out := errs[0]
	for i := 1; i < len(errs); i++ {
		out += "; " + errs[i]
	}
	return out
}
