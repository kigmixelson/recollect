package main

import "testing"

func TestHealthcheckURL(t *testing.T) {
	t.Parallel()
	cases := map[string]string{
		":8080":          "http://127.0.0.1:8080/healthz",
		"0.0.0.0:8080":   "http://127.0.0.1:8080/healthz",
		"127.0.0.1:8080": "http://127.0.0.1:8080/healthz",
	}
	for in, want := range cases {
		if got := healthcheckURL(in); got != want {
			t.Fatalf("healthcheckURL(%q)=%q, want %q", in, got, want)
		}
	}
}
