package httpx

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	"github.com/prometheus/client_golang/prometheus"

	"github.com/nebulacloud/nebula/internal/platform/logger"
)

// HeaderRequestID is the canonical correlation-id header (case-insensitive).
const HeaderRequestID = "X-Request-Id"

type ctxKey string

const ctxKeyCorrelationID ctxKey = "correlation_id"

// CorrelationIDFromContext returns the correlation id stored on ctx.
func CorrelationIDFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(ctxKeyCorrelationID).(string); ok {
		return v
	}
	return ""
}

// CorrelationID middleware ensures every request has an X-Request-Id header.
// If the client supplied one we trust it (sized capped); otherwise we mint
// a fresh ULID. The id is stored on the context and echoed in the response.
func CorrelationID() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			id := strings.TrimSpace(r.Header.Get(HeaderRequestID))
			if id == "" || len(id) > 128 {
				id = ulid.Make().String()
			}
			w.Header().Set(HeaderRequestID, id)
			ctx := context.WithValue(r.Context(), ctxKeyCorrelationID, id)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequestLogger emits a structured access log per request and attaches a
// per-request slog.Logger to the context (so handlers can pick it up via
// logger.FromContext and inherit the correlation id).
func RequestLogger(base *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			id := CorrelationIDFromContext(r.Context())
			scoped := logger.WithCorrelationID(base, id)

			ctx := logger.WithLogger(r.Context(), scoped)
			ww := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

			defer func() {
				dur := time.Since(start)
				attrs := []any{
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", ww.status),
					slog.Int("bytes", ww.bytes),
					slog.Duration("duration", dur),
					slog.String("remote_ip", clientIP(r)),
					slog.String("user_agent", r.UserAgent()),
				}
				switch {
				case ww.status >= 500:
					scoped.LogAttrs(ctx, slog.LevelError, "http.request", toAttrs(attrs)...)
				case ww.status >= 400:
					scoped.LogAttrs(ctx, slog.LevelWarn, "http.request", toAttrs(attrs)...)
				default:
					scoped.LogAttrs(ctx, slog.LevelInfo, "http.request", toAttrs(attrs)...)
				}
			}()

			next.ServeHTTP(ww, r.WithContext(ctx))
		})
	}
}

// Recoverer catches panics, logs them, and returns 500.
func Recoverer() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.FromContext(r.Context()).Error("http.panic", slog.Any("panic", rec))
					http.Error(w, `{"error":{"kind":"internal","message":"internal error"}}`, http.StatusInternalServerError)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// PrometheusMetrics records RED metrics for every request.
//
//	nebula_http_requests_total{method,route,status}
//	nebula_http_request_duration_seconds{method,route} (histogram)
//	nebula_http_in_flight (gauge)
//
// Route resolution: chi exposes the matched pattern via RouteContext; the
// concrete value is read in the deferred block.
func PrometheusMetrics(reg prometheus.Registerer) func(http.Handler) http.Handler {
	requests := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "nebula",
			Subsystem: "http",
			Name:      "requests_total",
			Help:      "Total number of HTTP requests handled.",
		},
		[]string{"method", "route", "status"},
	)
	duration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "nebula",
			Subsystem: "http",
			Name:      "request_duration_seconds",
			Help:      "Duration of HTTP requests.",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "route"},
	)
	inflight := prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "nebula",
		Subsystem: "http",
		Name:      "in_flight",
		Help:      "Number of in-flight HTTP requests.",
	})

	if reg != nil {
		reg.MustRegister(requests, duration, inflight)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			inflight.Inc()
			defer inflight.Dec()

			next.ServeHTTP(ww, r)

			route := routePattern(r)
			requests.WithLabelValues(r.Method, route, statusBucket(ww.status)).Inc()
			duration.WithLabelValues(r.Method, route).Observe(time.Since(start).Seconds())
		})
	}
}

// SecureHeaders applies a conservative set of response headers. Traefik may
// also enforce these at the edge — keeping them at the app layer is a
// defence-in-depth choice.
func SecureHeaders() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			h := w.Header()
			h.Set("X-Content-Type-Options", "nosniff")
			h.Set("X-Frame-Options", "DENY")
			h.Set("Referrer-Policy", "strict-origin-when-cross-origin")
			h.Set("Cross-Origin-Opener-Policy", "same-origin")
			h.Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")
			next.ServeHTTP(w, r)
		})
	}
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

type statusRecorder struct {
	http.ResponseWriter
	status int
	bytes  int
}

func (s *statusRecorder) WriteHeader(code int) {
	s.status = code
	s.ResponseWriter.WriteHeader(code)
}

func (s *statusRecorder) Write(b []byte) (int, error) {
	n, err := s.ResponseWriter.Write(b)
	s.bytes += n
	return n, err
}

func clientIP(r *http.Request) string {
	if v := r.Header.Get("X-Forwarded-For"); v != "" {
		if i := strings.IndexByte(v, ','); i > 0 {
			return strings.TrimSpace(v[:i])
		}
		return strings.TrimSpace(v)
	}
	if v := r.Header.Get("X-Real-Ip"); v != "" {
		return v
	}
	return r.RemoteAddr
}

func statusBucket(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	case code >= 200:
		return "2xx"
	default:
		return "1xx"
	}
}

func toAttrs(values []any) []slog.Attr {
	out := make([]slog.Attr, 0, len(values))
	for _, v := range values {
		if a, ok := v.(slog.Attr); ok {
			out = append(out, a)
		}
	}
	return out
}
