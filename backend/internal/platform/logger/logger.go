// Package logger provides a single source of truth for structured logging
// across NebulaCloud. It exposes a thin wrapper over the standard-library
// slog package so we can:
//
//   - emit JSON logs in production and human-friendly text in development
//   - attach correlation, user, org, and trace IDs to every log entry
//   - swap implementations (e.g. zap, zerolog) without rewriting call sites
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
)

// Field keys reused by every emitter to keep logs queryable in Loki.
const (
	FieldService       = "service"
	FieldEnv           = "env"
	FieldCorrelationID = "correlation_id"
	FieldUserID        = "user_id"
	FieldOrgID         = "org_id"
	FieldTraceID       = "trace_id"
	FieldComponent     = "component"
)

// Options controls how the logger is constructed.
type Options struct {
	Level       string // debug | info | warn | error
	Format      string // json | text (auto-detect when empty: json in prod, text in dev)
	ServiceName string
	Environment string
	Output      io.Writer // defaults to os.Stdout
}

// New builds a structured logger configured according to Options.
func New(opts Options) *slog.Logger {
	if opts.Output == nil {
		opts.Output = os.Stdout
	}

	level := parseLevel(opts.Level)
	handlerOpts := &slog.HandlerOptions{
		Level:     level,
		AddSource: level == slog.LevelDebug,
	}

	var handler slog.Handler
	switch strings.ToLower(opts.Format) {
	case "text":
		handler = slog.NewTextHandler(opts.Output, handlerOpts)
	case "json":
		handler = slog.NewJSONHandler(opts.Output, handlerOpts)
	default:
		// Auto: text in dev, JSON otherwise.
		if strings.EqualFold(opts.Environment, "development") {
			handler = slog.NewTextHandler(opts.Output, handlerOpts)
		} else {
			handler = slog.NewJSONHandler(opts.Output, handlerOpts)
		}
	}

	logger := slog.New(handler).With(
		slog.String(FieldService, opts.ServiceName),
		slog.String(FieldEnv, opts.Environment),
	)
	return logger
}

// SetDefault installs the provided logger as the slog default. Tests may
// override it freely; production code should retrieve via FromContext.
func SetDefault(l *slog.Logger) { slog.SetDefault(l) }

// ctxKey is unexported so downstream packages cannot accidentally collide.
type ctxKey struct{}

// WithLogger returns a context carrying the supplied logger.
func WithLogger(ctx context.Context, l *slog.Logger) context.Context {
	return context.WithValue(ctx, ctxKey{}, l)
}

// FromContext returns the logger attached to ctx, or slog.Default() if none.
func FromContext(ctx context.Context) *slog.Logger {
	if v, ok := ctx.Value(ctxKey{}).(*slog.Logger); ok && v != nil {
		return v
	}
	return slog.Default()
}

// WithCorrelationID returns a derived logger annotated with the provided id.
// Callers typically use the http correlation middleware which does this.
func WithCorrelationID(l *slog.Logger, id string) *slog.Logger {
	if l == nil {
		l = slog.Default()
	}
	if id == "" {
		return l
	}
	return l.With(slog.String(FieldCorrelationID, id))
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	case "info":
		fallthrough
	default:
		return slog.LevelInfo
	}
}
