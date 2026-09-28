package config

import (
	"reflect"
	"testing"
)

func TestParseHTTPPrefixes(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in   string
		want []string
	}{
		{"/recollect", []string{"", "/recollect"}},
		{"recollect", []string{"", "/recollect"}},
		{"/recollect/", []string{"", "/recollect"}},
		{"-", []string{""}},
		{"none", []string{""}},
		{"/recollect,/collect", []string{"", "/recollect", "/collect"}},
	}
	for _, tc := range cases {
		got := ParseHTTPPrefixes(tc.in)
		if !reflect.DeepEqual(got, tc.want) {
			t.Fatalf("ParseHTTPPrefixes(%q)=%q, want %q", tc.in, got, tc.want)
		}
	}
}
