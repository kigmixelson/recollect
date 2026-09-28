package query

import (
	"testing"
	"time"
)

func TestParseDepth(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want time.Duration
	}{
		{"12h", 12 * time.Hour},
		{"24h", 24 * time.Hour},
		{"3m", 3 * time.Minute},
		{"90s", 90 * time.Second},
		{"1d", 24 * time.Hour},
		{"2d12h", 60 * time.Hour},
		{"12 hours", 12 * time.Hour},
		{"3 minutes", 3 * time.Minute},
		{"12 часов", 12 * time.Hour},
		{"24 часа", 24 * time.Hour},
		{"3 минуты", 3 * time.Minute},
		{"1 день", 24 * time.Hour},
		{"2 дня", 48 * time.Hour},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.in, func(t *testing.T) {
			t.Parallel()
			got, err := ParseDepth(tc.in)
			if err != nil {
				t.Fatalf("ParseDepth(%q): %v", tc.in, err)
			}
			if got != tc.want {
				t.Fatalf("ParseDepth(%q)=%s, want %s", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseDepthRejects(t *testing.T) {
	t.Parallel()
	for _, in := range []string{"", "0h", "-1h", "foo", "12 weeks"} {
		if _, err := ParseDepth(in); err == nil {
			t.Fatalf("ParseDepth(%q) expected error", in)
		}
	}
}
