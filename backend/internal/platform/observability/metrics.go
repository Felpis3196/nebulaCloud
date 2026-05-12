package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/nebulacloud/nebula/internal/platform/logger"
)

// MetricsRegistry packages a fresh prometheus.Registry seeded with the
// canonical Go and process collectors. Modules register their own metrics
// on it via the Registerer accessor.
type MetricsRegistry struct {
	reg *prometheus.Registry
}

// NewMetricsRegistry builds a clean registry with sane defaults.
//
// We avoid prometheus.DefaultRegisterer to keep test isolation strong and
// to allow the build worker / runtime agent to coexist in the same process
// during integration tests.
func NewMetricsRegistry(serviceName string) *MetricsRegistry {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{Namespace: "nebula"}),
		collectors.NewBuildInfoCollector(),
	)

	prometheus.NewGaugeFunc(prometheus.GaugeOpts{
		Namespace:   "nebula",
		Name:        "service_info",
		Help:        "1 with the service name and start time as labels.",
		ConstLabels: prometheus.Labels{"service": serviceName},
	}, func() float64 { return 1 })

	return &MetricsRegistry{reg: reg}
}

// Registerer returns the underlying prometheus.Registerer so modules can
// register their own counters/histograms.
func (m *MetricsRegistry) Registerer() prometheus.Registerer { return m.reg }

// Gatherer returns the underlying prometheus.Gatherer for the HTTP exporter.
func (m *MetricsRegistry) Gatherer() prometheus.Gatherer { return m.reg }

// Handler returns an http.Handler exposing /metrics in the Prometheus format.
func (m *MetricsRegistry) Handler() http.Handler {
	return promhttp.HandlerFor(m.reg, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
	})
}

// MetricsServerOptions configures the standalone /metrics endpoint.
type MetricsServerOptions struct {
	Address string
	Path    string // defaults to /metrics
}

// RunMetricsServer starts a dedicated HTTP listener for the /metrics endpoint
// and blocks until ctx is cancelled. Operators typically isolate this on a
// separate port (NEBULA_METRICS_PORT) so it can stay private.
func (m *MetricsRegistry) RunMetricsServer(ctx context.Context, opts MetricsServerOptions) error {
	if opts.Path == "" {
		opts.Path = "/metrics"
	}

	mux := http.NewServeMux()
	mux.Handle(opts.Path, m.Handler())
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	srv := &http.Server{
		Addr:              opts.Address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log := logger.FromContext(ctx)
	errCh := make(chan error, 1)
	go func() {
		log.Info("metrics.server.start", slog.String("address", opts.Address), slog.String("path", opts.Path))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("metrics server: %w", err)
			return
		}
		errCh <- nil
	}()

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
		return nil
	case err := <-errCh:
		return err
	}
}
