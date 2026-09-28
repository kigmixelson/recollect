package query

import (
	"fmt"
	"strings"
)

const (
	AggAvg = "avg"
	AggMin = "min"
	AggMax = "max"
	AggSum = "sum"
	AggDev = "dev"
)

var downsample = map[string]string{
	AggAvg: "all-avg",
	AggMin: "all-min",
	AggMax: "all-max",
	AggSum: "all-sum",
	AggDev: "all-dev",
}

func NormalizeAggregates(raw []string) ([]string, error) {
	out := make([]string, 0, len(raw))
	seen := make(map[string]struct{}, len(raw))
	for _, item := range raw {
		if strings.TrimSpace(item) == "" {
			continue
		}
		name, err := normalizeAggregate(item)
		if err != nil {
			return nil, err
		}
		if _, ok := seen[name]; ok {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}
	if len(out) == 0 {
		return []string{AggAvg}, nil
	}
	return out, nil
}

func Downsample(agg string) string {
	if v, ok := downsample[agg]; ok {
		return v
	}
	return "all-avg"
}

func normalizeAggregate(s string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "avg", "average", "mean", "среднее", "средний", "средняя":
		return AggAvg, nil
	case "min", "minimum", "минимум", "мин":
		return AggMin, nil
	case "max", "maximum", "максимум", "макс":
		return AggMax, nil
	case "sum", "total", "сумма":
		return AggSum, nil
	case "dev", "deviation", "stddev", "stdev", "std", "дивиация", "девиация", "отклонение", "стандартное отклонение", "standard deviation":
		return AggDev, nil
	case "":
		return "", fmt.Errorf("empty aggregate")
	default:
		return "", fmt.Errorf("unknown aggregate %q (expected avg, min, max, sum, dev)", s)
	}
}
