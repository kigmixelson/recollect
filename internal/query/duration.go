package query

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

var (
	humanRe = regexp.MustCompile(`(?i)^\s*(\d+(?:\.\d+)?)\s*([a-zA-Zа-яА-ЯёЁ]+)\s*$`)
	dayRe   = regexp.MustCompile(`(?i)(\d+(?:\.\d+)?)d`)
)

// ParseDepth converts a human-readable window into a duration counted back from now.
// Accepts Go durations (12h, 3m, 90s), day fragments (1d, 2d12h),
// and English/Russian phrases (12 hours, 12 часов, 3 минуты).
func ParseDepth(s string) (time.Duration, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, fmt.Errorf("depth is required")
	}

	if d, err := time.ParseDuration(strings.ToLower(s)); err == nil {
		return positive(d)
	}
	if d, err := parseWithDays(s); err == nil {
		return positive(d)
	}

	m := humanRe.FindStringSubmatch(s)
	if m == nil {
		return 0, fmt.Errorf("invalid depth %q", s)
	}
	n, err := strconv.ParseFloat(m[1], 64)
	if err != nil {
		return 0, fmt.Errorf("invalid depth %q", s)
	}
	unit, err := unitDuration(m[2])
	if err != nil {
		return 0, fmt.Errorf("invalid depth %q: %w", s, err)
	}
	return positive(time.Duration(n * float64(unit)))
}

func parseWithDays(s string) (time.Duration, error) {
	if !dayRe.MatchString(s) {
		return 0, fmt.Errorf("no day unit")
	}
	converted := dayRe.ReplaceAllStringFunc(strings.ToLower(s), func(part string) string {
		num := strings.TrimSuffix(strings.ToLower(part), "d")
		n, err := strconv.ParseFloat(num, 64)
		if err != nil {
			return part
		}
		return strconv.FormatFloat(n*24, 'f', -1, 64) + "h"
	})
	return time.ParseDuration(converted)
}

func unitDuration(raw string) (time.Duration, error) {
	u := strings.ToLower(strings.TrimSpace(raw))
	u = strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, u)

	switch u {
	case "ms", "millisecond", "milliseconds":
		return time.Millisecond, nil
	case "s", "sec", "secs", "second", "seconds", "сек", "секунда", "секунды", "секунд":
		return time.Second, nil
	case "m", "min", "mins", "minute", "minutes", "мин", "минута", "минуты", "минут":
		return time.Minute, nil
	case "h", "hr", "hrs", "hour", "hours", "ч", "час", "часа", "часов":
		return time.Hour, nil
	case "d", "day", "days", "д", "день", "дня", "дней":
		return 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown unit %q", raw)
	}
}

func positive(d time.Duration) (time.Duration, error) {
	if d <= 0 {
		return 0, fmt.Errorf("depth must be positive")
	}
	return d, nil
}
