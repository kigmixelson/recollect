package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	ListenAddr  string
	HTTPTimeout time.Duration
	Concurrency int
	TLSInsecure bool
	HTTPPrefix  string
}

func FromEnv() Config {
	return Config{
		ListenAddr:  env("LISTEN_ADDR", ":8080"),
		HTTPTimeout: envDuration("HTTP_TIMEOUT", 30*time.Second),
		Concurrency: envInt("SAYMON_CONCURRENCY", 8),
		TLSInsecure: envBool("SAYMON_TLS_INSECURE", false),
		HTTPPrefix:  env("HTTP_PREFIX", "/recollect"),
	}
}

// HTTPPrefixes is the list of URL prefixes the HTTP API is mounted on.
// The empty prefix (root) is always included so healthcheck and stripped nginx paths keep working.
func (c Config) HTTPPrefixes() []string {
	return ParseHTTPPrefixes(c.HTTPPrefix)
}

func ParseHTTPPrefixes(raw string) []string {
	raw = strings.TrimSpace(raw)
	extras := []string{"/recollect"}
	switch {
	case raw == "-" || strings.EqualFold(raw, "none"):
		extras = nil
	case raw != "":
		extras = nil
		for _, part := range strings.Split(raw, ",") {
			if p := normalizePrefix(part); p != "" {
				extras = append(extras, p)
			}
		}
	}
	out := []string{""}
	seen := map[string]struct{}{"": {}}
	for _, p := range extras {
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		out = append(out, p)
	}
	return out
}

func normalizePrefix(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, "/")
	if p == "" {
		return ""
	}
	return "/" + p
}

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func envBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func envDuration(key string, fallback time.Duration) time.Duration {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil || d <= 0 {
		return fallback
	}
	return d
}
