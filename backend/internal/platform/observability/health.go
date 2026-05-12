// Package observability assembles the platform's runtime visibility surface:
// health checks, Prometheus exposure, and (later) OpenTelemetry tracing.
package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// HealthChecker is implemented by any subsystem that wants to be probed by
// the platform's /healthz and /readyz endpoints.
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) error
}

// HealthRegistry aggregates health checkers and exposes HTTP handlers.
//
// /healthz       — liveness, never fails (the process is up)
// /readyz        — readiness, runs all registered checks
type HealthRegistry struct {
	mu       sync.RWMutex
	checkers []HealthChecker
	timeout  time.Duration
	version  string
}

// NewHealthRegistry returns an empty registry.
func NewHealthRegistry(version string) *HealthRegistry {
	return &HealthRegistry{timeout: 3 * time.Second, version: version}
}

// Register adds one or more checkers.
func (h *HealthRegistry) Register(checkers ...HealthChecker) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.checkers = append(h.checkers, checkers...)
}

// LivenessHandler answers cheaply with the build version.
func (h *HealthRegistry) LivenessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"version": h.version,
			"time":    time.Now().UTC().Format(time.RFC3339Nano),
		})
	}
}

// ReadinessHandler runs every registered check in parallel and returns 200
// only when all pass. Otherwise it returns 503 with a per-check breakdown.
func (h *HealthRegistry) ReadinessHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		h.mu.RLock()
		checkers := make([]HealthChecker, len(h.checkers))
		copy(checkers, h.checkers)
		h.mu.RUnlock()

		ctx, cancel := context.WithTimeout(r.Context(), h.timeout)
		defer cancel()

		type result struct {
			Name   string `json:"name"`
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}

		results := make([]result, len(checkers))
		var wg sync.WaitGroup
		for i, c := range checkers {
			wg.Add(1)
			go func(i int, c HealthChecker) {
				defer wg.Done()
				if err := c.Check(ctx); err != nil {
					results[i] = result{Name: c.Name(), Status: "down", Error: err.Error()}
					return
				}
				results[i] = result{Name: c.Name(), Status: "ok"}
			}(i, c)
		}
		wg.Wait()

		status := http.StatusOK
		overall := "ok"
		for _, r := range results {
			if r.Status != "ok" {
				status = http.StatusServiceUnavailable
				overall = "degraded"
				break
			}
		}

		writeJSON(w, status, map[string]any{
			"status":  overall,
			"version": h.version,
			"time":    time.Now().UTC().Format(time.RFC3339Nano),
			"checks":  results,
		})
	}
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
