// Package redis wraps the platform's Redis client. It owns the connection,
// exposes a HealthChecker for /healthz, and offers a single place to add
// instrumentation hooks (tracing, metrics) later on.
package redis

import (
	"context"
	"errors"
	"fmt"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

// Config holds the subset of options needed to dial Redis.
type Config struct {
	Addr     string
	Password string
	DB       int
}

// Connect dials Redis, performs a PING bounded by ctx, and returns a ready
// client. The caller is responsible for calling Close on shutdown.
func Connect(ctx context.Context, cfg Config) (*goredis.Client, error) {
	if cfg.Addr == "" {
		return nil, errors.New("redis: empty address")
	}

	client := goredis.NewClient(&goredis.Options{
		Addr:            cfg.Addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		MinIdleConns:    1,
		PoolSize:        20,
		ConnMaxIdleTime: 5 * time.Minute,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
	})

	pingCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("redis: ping: %w", err)
	}
	return client, nil
}

// PingHealth implements observability.HealthChecker against a Redis client.
type PingHealth struct {
	Client *goredis.Client
}

// Name returns the check identifier reported in /healthz responses.
func (p PingHealth) Name() string { return "redis" }

// Check executes a lightweight Ping bounded by ctx.
func (p PingHealth) Check(ctx context.Context) error {
	if p.Client == nil {
		return errors.New("redis: client not initialised")
	}
	return p.Client.Ping(ctx).Err()
}
