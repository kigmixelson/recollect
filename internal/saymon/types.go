package saymon

import (
	"bytes"
	"encoding/json"
	"math"
	"sort"
	"strconv"
)

type Object struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	MetricsCache []string `json:"metrics_cache"`
}

type MetricHistory struct {
	Metric string     `json:"metric"`
	Dps    DataPoints `json:"dps"`
}

type DataPoint struct {
	Timestamp int64
	Value     float64
}

type DataPoints []DataPoint

func (d DataPoints) MarshalJSON() ([]byte, error) {
	out := make([][2]float64, len(d))
	for i, p := range d {
		out[i] = [2]float64{float64(p.Timestamp), p.Value}
	}
	return json.Marshal(out)
}

func (d *DataPoints) UnmarshalJSON(b []byte) error {
	b = bytes.TrimSpace(b)
	if len(b) == 0 || bytes.Equal(b, []byte("null")) {
		*d = nil
		return nil
	}

	switch b[0] {
	case '[':
		var items []json.RawMessage
		if err := json.Unmarshal(b, &items); err != nil {
			return err
		}
		out := make(DataPoints, 0, len(items))
		for _, item := range items {
			item = bytes.TrimSpace(item)
			if len(item) == 0 {
				continue
			}
			if item[0] == '[' {
				var pair []float64
				if err := json.Unmarshal(item, &pair); err != nil || len(pair) < 2 {
					continue
				}
				out = append(out, DataPoint{Timestamp: int64(pair[0]), Value: pair[1]})
				continue
			}
			var obj struct {
				Timestamp int64   `json:"Timestamp"`
				Value     float64 `json:"Value"`
			}
			if err := json.Unmarshal(item, &obj); err == nil {
				out = append(out, DataPoint{Timestamp: obj.Timestamp, Value: obj.Value})
			}
		}
		*d = out
		return nil
	case '{':
		var raw map[string]float64
		if err := json.Unmarshal(b, &raw); err != nil {
			return err
		}
		out := make(DataPoints, 0, len(raw))
		for k, v := range raw {
			ts, err := strconv.ParseInt(k, 10, 64)
			if err != nil {
				f, ferr := strconv.ParseFloat(k, 64)
				if ferr != nil {
					continue
				}
				ts = int64(f)
			}
			out = append(out, DataPoint{Timestamp: ts, Value: v})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Timestamp < out[j].Timestamp })
		*d = out
		return nil
	default:
		return json.Unmarshal(b, (*[]DataPoint)(d))
	}
}

func Reduce(points DataPoints, agg string) *float64 {
	if len(points) == 0 {
		return nil
	}
	// A single point from SAYMON all-* downsample is already the aggregate.
	if len(points) == 1 {
		v := points[0].Value
		return &v
	}
	switch agg {
	case "min":
		v := points[0].Value
		for _, p := range points[1:] {
			if p.Value < v {
				v = p.Value
			}
		}
		return &v
	case "max":
		v := points[0].Value
		for _, p := range points[1:] {
			if p.Value > v {
				v = p.Value
			}
		}
		return &v
	case "sum":
		sum := 0.0
		for _, p := range points {
			sum += p.Value
		}
		return &sum
	case "dev":
		return stddev(points)
	default:
		sum := 0.0
		for _, p := range points {
			sum += p.Value
		}
		v := sum / float64(len(points))
		return &v
	}
}

func stddev(points DataPoints) *float64 {
	n := float64(len(points))
	mean := 0.0
	for _, p := range points {
		mean += p.Value
	}
	mean /= n
	ss := 0.0
	for _, p := range points {
		d := p.Value - mean
		ss += d * d
	}
	v := math.Sqrt(ss / n)
	return &v
}
