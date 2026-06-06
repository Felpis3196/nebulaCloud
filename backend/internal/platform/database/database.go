// Package database manages the platform's PostgreSQL connection pool and
// migration runner. The package exposes:
//
//   - Connect       — opens a tuned pgxpool.Pool
//   - HealthChecker — minimal interface for /healthz integration
//   - Migrate       — runs pending migrations against the supplied DSN
//
// Modules receive *pgxpool.Pool via dependency injection rather than a
// global, so unit tests can pass txdb-style fakes when needed.
package database

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres" // pg migration driver
	_ "github.com/golang-migrate/migrate/v4/source/file"      // file source for migrations
	"github.com/jackc/pgx/v5/pgxpool"
)

// Config carries the subset of platform/config.DatabaseConfig actually
// needed by the pool builder. Keeping it local avoids a cyclic import.
type Config struct {
	DSN             string
	MaxConns        int32
	MinConns        int32
	MaxConnLifetime time.Duration
}

// Connect builds a tuned pgxpool.Pool. The supplied context bounds the
// initial dial — production callers pass a 10–30s timeout for fail-fast
// boot semantics.
func Connect(ctx context.Context, cfg Config) (*pgxpool.Pool, error) {
	if cfg.DSN == "" {
		return nil, errors.New("database: empty DSN")
	}

	poolCfg, err := pgxpool.ParseConfig(cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("database: parse DSN: %w", err)
	}

	if cfg.MaxConns > 0 {
		poolCfg.MaxConns = cfg.MaxConns
	}
	if cfg.MinConns > 0 {
		poolCfg.MinConns = cfg.MinConns
	}
	if cfg.MaxConnLifetime > 0 {
		poolCfg.MaxConnLifetime = cfg.MaxConnLifetime
	}
	poolCfg.HealthCheckPeriod = 30 * time.Second

	pool, err := pgxpool.NewWithConfig(ctx, poolCfg)
	if err != nil {
		return nil, fmt.Errorf("database: connect: %w", err)
	}

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("database: ping: %w", err)
	}
	return pool, nil
}

// Migrate runs all pending migrations in the supplied directory against the
// supplied DSN. Returns nil when no migration was needed.
func Migrate(dsn, sourcePath string) error {
	if dsn == "" {
		return errors.New("database: migrate: empty DSN")
	}
	if sourcePath == "" {
		sourcePath = "file://migrations"
	} else if len(sourcePath) > 7 && sourcePath[:7] != "file://" {
		sourcePath = "file://" + sourcePath
	}

	m, err := migrate.New(sourcePath, dsn)
	if err != nil {
		return fmt.Errorf("database: migrate init: %w", err)
	}
	defer func() { _, _ = m.Close() }()

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("database: migrate up: %w", err)
	}
	return nil
}

// MigrateDown rolls back exactly one migration version.
func MigrateDown(dsn, sourcePath string) error {
	if dsn == "" {
		return errors.New("database: migrate: empty DSN")
	}
	if sourcePath == "" {
		sourcePath = "file://migrations"
	} else if len(sourcePath) > 7 && sourcePath[:7] != "file://" {
		sourcePath = "file://" + sourcePath
	}
	m, err := migrate.New(sourcePath, dsn)
	if err != nil {
		return fmt.Errorf("database: migrate init: %w", err)
	}
	defer func() { _, _ = m.Close() }()
	if err := m.Steps(-1); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("database: migrate down: %w", err)
	}
	return nil
}

// PingHealth implements the platform/observability.HealthChecker contract
// against a live pool.
type PingHealth struct {
	Pool *pgxpool.Pool
}

// Name returns the check identifier reported in /healthz responses.
func (p PingHealth) Name() string { return "postgres" }

// Check executes a lightweight Ping bounded by ctx.
func (p PingHealth) Check(ctx context.Context) error {
	if p.Pool == nil {
		return errors.New("postgres: pool not initialised")
	}
	return p.Pool.Ping(ctx)
}

