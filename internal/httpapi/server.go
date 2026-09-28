package httpapi

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/kigmixelson/recollect/internal/collect"
	"github.com/kigmixelson/recollect/internal/query"
)

type Server struct {
	service *collect.Service
	log     *slog.Logger
}

func New(service *collect.Service, log *slog.Logger, prefixes ...string) http.Handler {
	if log == nil {
		log = slog.Default()
	}
	if len(prefixes) == 0 {
		prefixes = []string{"", "/recollect"}
	}
	s := &Server{service: service, log: log}

	mux := http.NewServeMux()
	seen := map[string]struct{}{}
	for _, prefix := range prefixes {
		p := normalizePrefix(prefix)
		if _, ok := seen[p]; ok {
			continue
		}
		seen[p] = struct{}{}
		mux.HandleFunc("GET "+p+"/healthz", s.health)
		mux.HandleFunc("GET "+p+"/api/collect", s.collect)
		mux.HandleFunc("POST "+p+"/api/collect", s.collect)
	}
	return withRecover(log, mux)
}

func normalizePrefix(p string) string {
	p = strings.TrimSpace(p)
	p = strings.Trim(p, "/")
	if p == "" {
		return ""
	}
	return "/" + p
}

func (s *Server) health(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) collect(w http.ResponseWriter, r *http.Request) {
	params, err := query.ParseCollect(r)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
		return
	}

	started := time.Now()
	result, err := s.service.Collect(r.Context(), params)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, r.Context().Err()) {
			status = http.StatusGatewayTimeout
		}
		s.log.Error("collect failed", "err", err, "object_id", params.ObjectID, "host", params.Host)
		writeJSON(w, status, map[string]string{"error": err.Error()})
		return
	}

	s.log.Info("collect ok",
		"object_id", params.ObjectID,
		"samples", len(result.Samples),
		"skipped", len(result.Skipped),
		"took", time.Since(started).String(),
	)
	writeJSON(w, http.StatusOK, result)
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func withRecover(log *slog.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Error("panic", "err", rec, "path", r.URL.Path)
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "internal server error"})
			}
		}()
		next.ServeHTTP(w, r)
	})
}
