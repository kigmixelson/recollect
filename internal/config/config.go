package config

import (
	"os"
	"strconv"
	"time"
)

type Config struct {
	ListenAddr  string
	HTTPTimeout time.Duration
	Concurrency int
	TLSInsecure bool
}

func FromEnv() Config {
	return Config{
		ListenAddr:  env("LISTEN_ADDR", ":8080"),
		HTTPTimeout: envDuration("HTTP_TIMEOUT", 30*time.Second),
		Concurrency: envInt("SAYMON_CONCURRENCY", 8),
		TLSInsecure: envBool("SAYMON_TLS_INSECURE", false),
	}
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
