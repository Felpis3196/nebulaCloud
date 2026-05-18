// Command api is the NebulaCloud control-plane entrypoint. It owns the HTTP
// surface for every authenticated client (dashboard + automation) and
// composes the modular monolith from its constituent modules.
//
// Subcommands:
//
//	api                — run the HTTP server (default)
//	api healthcheck    — exit 0 iff /healthz returns 200 (used by Docker)
//	api migrate up     — apply pending DB migrations
//	api version        — print the build version and exit
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime/debug"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"

	auditmod "github.com/nebulacloud/nebula/internal/modules/audit"
	identityapp "github.com/nebulacloud/nebula/internal/modules/identity/application"
	identityinfra "github.com/nebulacloud/nebula/internal/modules/identity/infrastructure"
	identityif "github.com/nebulacloud/nebula/internal/modules/identity/interfaces"
	projectsapp "github.com/nebulacloud/nebula/internal/modules/projects/application"
	projectsinfra "github.com/nebulacloud/nebula/internal/modules/projects/infrastructure"
	projectsif "github.com/nebulacloud/nebula/internal/modules/projects/interfaces"
	"github.com/nebulacloud/nebula/internal/platform/config"
	"github.com/nebulacloud/nebula/internal/platform/database"
	platformerrors "github.com/nebulacloud/nebula/internal/platform/errors"
	"github.com/nebulacloud/nebula/internal/platform/httpx"
	"github.com/nebulacloud/nebula/internal/platform/logger"
	"github.com/nebulacloud/nebula/internal/platform/observability"
	platformqueue "github.com/nebulacloud/nebula/internal/platform/queue"
	platformredis "github.com/nebulacloud/nebula/internal/platform/redis"
	"github.com/nebulacloud/nebula/internal/platform/secrets"
)

// version is overridden at build time:
//
//	-ldflags "-X main.version=$(git describe --tags --always)"
var version = "dev"

func main() {
	if err := run(os.Args[1:]); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "fatal: %v\n", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) > 0 {
		switch args[0] {
		case "version":
			fmt.Println(buildVersion())
			return nil
		case "healthcheck":
			return runHealthcheck()
		case "migrate":
			cfg, err := config.Load()
			if err != nil {
				return err
			}
			return runMigrate(cfg, args[1:])
		}
	}
	return runServer()
}

// ----------------------------------------------------------------------------
// HTTP server
// ----------------------------------------------------------------------------

func runServer() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	log := logger.New(logger.Options{
		Level:       cfg.LogLevel,
		ServiceName: cfg.ServiceName,
		Environment: string(cfg.Env),
	})
	logger.SetDefault(log)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	ctx = logger.WithLogger(ctx, log)

	log.Info("api.start",
		slog.String("version", buildVersion()),
		slog.String("env", string(cfg.Env)),
		slog.String("address", cfg.HTTP.Address()),
	)

	// ---- datastores ------------------------------------------------------
	dbCtx, dbCancel := context.WithTimeout(ctx, 30*time.Second)
	defer dbCancel()
	pool, err := database.Connect(dbCtx, database.Config{
		DSN:             cfg.Database.DSN(),
		MaxConns:        cfg.Database.MaxConns,
		MinConns:        cfg.Database.MinConns,
		MaxConnLifetime: cfg.Database.MaxConnLifetime,
	})
	if err != nil {
		return fmt.Errorf("database connect: %w", err)
	}
	defer pool.Close()

	redisClient, err := platformredis.Connect(ctx, platformredis.Config{
		Addr:     cfg.Redis.Address(),
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	if err != nil {
		return fmt.Errorf("redis connect: %w", err)
	}
	defer func() { _ = redisClient.Close() }()

	// ---- observability ---------------------------------------------------
	metrics := observability.NewMetricsRegistry(cfg.ServiceName)
	health := observability.NewHealthRegistry(buildVersion())
	health.Register(
		database.PingHealth{Pool: pool},
		platformredis.PingHealth{Client: redisClient},
	)

	// ---- modules ---------------------------------------------------------
	auditRecorder := auditmod.NewRecorder(pool)

	identityHasher := identityinfra.NewArgon2idHasher(cfg.Auth.PasswordPepper)
	identityIssuer, err := identityinfra.NewJWTIssuer(cfg.Auth.JWTSecret, cfg.Auth.JWTIssuer)
	if err != nil {
		return fmt.Errorf("identity: jwt issuer: %w", err)
	}
	identitySvc, err := identityapp.New(identityapp.Config{
		Users:       identityinfra.NewPostgresUserRepo(pool),
		Sessions:    identityinfra.NewPostgresSessionRepo(pool),
		Memberships: identityinfra.NewPostgresMembershipRepo(pool),
		Hasher:      identityHasher,
		Tokens:      identityIssuer,
		Refresh:     identityinfra.NewRefreshGenerator(),
		Audit:       auditRecorder,
		AccessTTL:   cfg.Auth.AccessTTL,
		RefreshTTL:  cfg.Auth.RefreshTTL,
	})
	if err != nil {
		return fmt.Errorf("identity: %w", err)
	}
	identityHandler := identityif.NewHandler(identitySvc)

	sealer, err := secrets.NewAESGCMSealerFromBase64(cfg.Secrets.Key)
	if err != nil {
		return fmt.Errorf("secrets sealer: %w", err)
	}
	projectRepo := projectsinfra.NewRepository(pool)
	queueProd := platformqueue.NewAsynqProducer(cfg.Redis.Address(), cfg.Redis.Password, cfg.Redis.DB)
	defer func() { _ = queueProd.Close() }()
	projectsSvc := projectsapp.New(projectRepo, sealer, queueProd, auditRecorder, cfg)
	projectsHandler := projectsif.NewHandler(projectsSvc, sealer, cfg.Runtime.BaseDomain)

	// ---- HTTP router -----------------------------------------------------
	router := newRouter(cfg, log, metrics, health, identitySvc, identityHandler, projectsHandler)

	server := httpx.NewServer(router, httpx.ServerOptions{
		Address:         cfg.HTTP.Address(),
		ReadTimeout:     cfg.HTTP.ReadTimeout,
		WriteTimeout:    cfg.HTTP.WriteTimeout,
		IdleTimeout:     cfg.HTTP.IdleTimeout,
		ShutdownTimeout: cfg.HTTP.ShutdownTimeout,
	})

	// ---- supervised goroutines ------------------------------------------
	var wg sync.WaitGroup
	errCh := make(chan error, 2)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := server.Run(ctx); err != nil {
			errCh <- fmt.Errorf("api server: %w", err)
		}
	}()

	if cfg.Metrics.Enabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := metrics.RunMetricsServer(ctx, observability.MetricsServerOptions{
				Address: cfg.Metrics.Address(),
			}); err != nil {
				errCh <- fmt.Errorf("metrics server: %w", err)
			}
		}()
	}

	go func() { wg.Wait(); close(errCh) }()

	// First non-nil error from any goroutine wins.
	var firstErr error
	for err := range errCh {
		if err != nil && firstErr == nil {
			firstErr = err
			stop() // unwind every other goroutine
		}
	}

	log.Info("api.shutdown.complete")
	return firstErr
}

// ----------------------------------------------------------------------------
// Router
// ----------------------------------------------------------------------------

func newRouter(
	cfg config.Config,
	log *slog.Logger,
	metrics *observability.MetricsRegistry,
	health *observability.HealthRegistry,
	identitySvc *identityapp.Service,
	identityHandler *identityif.Handler,
	projectsHandler *projectsif.Handler,
) http.Handler {
	r := chi.NewRouter()

	// Standard middleware chain. Order matters: correlation id must be first
	// so every other middleware can attach it to logs / responses.
	r.Use(middleware.RealIP)
	r.Use(middleware.RequestSize(1 << 20))
	r.Use(middleware.StripSlashes)
	r.Use(middleware.Compress(5))
	r.Use(httpx.CorrelationID())
	r.Use(httpx.SecureHeaders())
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   cfg.CORS.AllowedOrigins,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Authorization", "Content-Type", "X-Request-Id"},
		ExposedHeaders:   []string{"X-Request-Id"},
		AllowCredentials: true,
		MaxAge:           300,
	}))
	r.Use(httpx.RequestLogger(log))
	r.Use(httpx.Recoverer())
	r.Use(httpx.PrometheusMetrics(metrics.Registerer()))

	// Liveness / readiness — public, no auth.
	r.Get("/healthz", health.LivenessHandler())
	r.Get("/readyz", health.ReadinessHandler())

	// API surface lives under /api/v1. Modules mount their sub-routers here
	// as they come online.
	r.Route("/api/v1", func(api chi.Router) {
		api.Get("/", func(w http.ResponseWriter, _ *http.Request) {
			httpx.OK(w, map[string]any{
				"name":    "NebulaCloud API",
				"version": buildVersion(),
				"phase":   "3 — workspace + GitHub webhooks",
				"docs":    "/api/v1/openapi.yaml",
			})
		})
		api.Get("/openapi.yaml", serveOpenAPI())

		api.Post("/webhooks/github", projectsHandler.GithubWebhook)

		// Public identity routes (register, login, refresh, logout).
		identityHandler.Mount(api)

		// Authenticated routes — install the bearer-token guard.
		api.Group(func(authed chi.Router) {
			authed.Use(identityif.Authenticator(identitySvc))
			identityHandler.MountAuthenticated(authed)
			projectsHandler.Mount(authed)
		})
	})

	// Catch-all — keep envelope shape consistent.
	r.NotFound(func(w http.ResponseWriter, _ *http.Request) {
		httpx.Error(w, platformerrors.NotFound("route not found"))
	})

	return r
}

// ----------------------------------------------------------------------------
// Subcommands
// ----------------------------------------------------------------------------

func runHealthcheck() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	url := fmt.Sprintf("http://127.0.0.1:%d/healthz", cfg.HTTP.Port)
	client := http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("healthcheck: status %d", resp.StatusCode)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

func runMigrate(cfg config.Config, args []string) error {
	direction := "up"
	if len(args) > 0 {
		direction = strings.ToLower(args[0])
	}
	switch direction {
	case "up":
		return database.Migrate(cfg.Database.DSN(), "migrations")
	case "down":
		return errors.New("migrate down: not yet exposed (phase 0); use docker exec for now")
	default:
		return fmt.Errorf("unknown migrate direction: %s", direction)
	}
}

// ----------------------------------------------------------------------------
// helpers
// ----------------------------------------------------------------------------

// serveOpenAPI returns a handler that serves the embedded OpenAPI document.
// Phase 0 keeps it as a simple file read; Phase 1+ may switch to embed.FS.
func serveOpenAPI() http.HandlerFunc {
	return func(w http.ResponseWriter, _ *http.Request) {
		path := "api/openapi.yaml"
		f, err := os.Open(path)
		if err != nil {
			httpx.Error(w, platformerrors.NotFound("openapi spec missing"))
			return
		}
		defer f.Close()
		w.Header().Set("Content-Type", "application/yaml")
		_, _ = io.Copy(w, f)
	}
}

func buildVersion() string {
	if version != "dev" {
		return version
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				if len(s.Value) > 7 {
					return s.Value[:7]
				}
				return s.Value
			}
		}
	}
	return "dev"
}
