package main

import (
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/kigmixelson/recollect/internal/collect"
	"github.com/kigmixelson/recollect/internal/config"
	"github.com/kigmixelson/recollect/internal/httpapi"
	"github.com/kigmixelson/recollect/internal/saymon"
)

func main() {
	cfg := config.FromEnv()
	if len(os.Args) > 1 && os.Args[1] == "healthcheck" {
		os.Exit(runHealthcheck(cfg.ListenAddr))
	}
	log := slog.New(slog.NewTextHandler(os.Stdout, nil))

	client := saymon.NewClient(cfg.HTTPTimeout, cfg.TLSInsecure)
	svc := collect.New(client, cfg.Concurrency)
	handler := httpapi.New(svc, log, cfg.HTTPPrefixes()...)

	srv := &http.Server{
		Addr:              cfg.ListenAddr,
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      2 * time.Minute,
		IdleTimeout:       60 * time.Second,
	}

	log.Info("listening",
		"addr", cfg.ListenAddr,
		"timeout", cfg.HTTPTimeout.String(),
		"concurrency", cfg.Concurrency,
		"tls_insecure", cfg.TLSInsecure,
		"prefixes", cfg.HTTPPrefixes(),
	)
	if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Error("server stopped", "err", err)
		os.Exit(1)
	}
}
