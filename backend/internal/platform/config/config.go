// Package config loads and validates the platform configuration from
// environment variables. All NebulaCloud binaries (api, build-worker,
// runtime-agent) call Load to obtain a strongly-typed Config value.
//
// Configuration sources, in order of precedence:
//  1. real process environment
//  2. variables loaded from a .env file (auto-discovered, optional)
//  3. struct-tag defaults (env:"...,default=..." on Config fields)
package config

import (
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/caarlos0/env/v11"
	"github.com/joho/godotenv"
)

// Environment identifies the runtime context the binary is executing in.
type Environment string

const (
	EnvDevelopment Environment = "development"
	EnvStaging     Environment = "staging"
	EnvProduction  Environment = "production"
)

// IsProduction returns true when running in the production profile.
func (e Environment) IsProduction() bool { return e == EnvProduction }

// IsDevelopment returns true when running locally / in development.
func (e Environment) IsDevelopment() bool { return e == EnvDevelopment }

// Config is the root configuration aggregate consumed by every binary.
type Config struct {
	Env         Environment `env:"NEBULA_ENV"          envDefault:"development"`
	ServiceName string      `env:"NEBULA_SERVICE_NAME" envDefault:"nebula-api"`
	LogLevel    string      `env:"NEBULA_LOG_LEVEL"    envDefault:"info"`

	HTTP          HTTPConfig
	Database      DatabaseConfig
	Redis         RedisConfig
	Auth          AuthConfig
	Secrets       SecretsConfig
	GitHub        GitHubConfig
	Build         BuildConfig
	Runtime       RuntimeConfig
	Metrics       MetricsConfig
	Tracing       TracingConfig
	RateLimit     RateLimitConfig
	CORS          CORSConfig
	App           AppConfig
	Observability ObservabilityConfig
	AutoMigrate   bool `env:"NEBULA_AUTO_MIGRATE" envDefault:"false"`
	Terminal      TerminalConfig
}

// AppConfig holds dashboard URLs used for OAuth redirects.
type AppConfig struct {
	URL string `env:"NEBULA_APP_URL" envDefault:"http://localhost:3000"`
}

// TerminalConfig gates the web terminal (requires Docker socket on API).
type TerminalConfig struct {
	Enabled bool `env:"NEBULA_TERMINAL_ENABLED" envDefault:"false"`
}

// ObservabilityConfig points the API at Loki/Prometheus for dashboard reads.
type ObservabilityConfig struct {
	LokiURL       string `env:"NEBULA_LOKI_URL"       envDefault:"http://localhost:3100"`
	PrometheusURL string `env:"NEBULA_PROMETHEUS_URL" envDefault:"http://localhost:9090"`
}

// HTTPConfig governs the public-facing API gateway.
type HTTPConfig struct {
	Host            string        `env:"NEBULA_API_HOST"             envDefault:"0.0.0.0"`
	Port            int           `env:"NEBULA_API_PORT"             envDefault:"8080"`
	ReadTimeout     time.Duration `env:"NEBULA_API_READ_TIMEOUT"     envDefault:"15s"`
	WriteTimeout    time.Duration `env:"NEBULA_API_WRITE_TIMEOUT"    envDefault:"15s"`
	IdleTimeout     time.Duration `env:"NEBULA_API_IDLE_TIMEOUT"     envDefault:"60s"`
	ShutdownTimeout time.Duration `env:"NEBULA_API_SHUTDOWN_TIMEOUT" envDefault:"20s"`
	PublicURL       string        `env:"NEBULA_API_PUBLIC_URL"       envDefault:"http://api.nebula.localhost"`
	DashboardURL    string        `env:"NEBULA_DASHBOARD_URL"        envDefault:"http://app.nebula.localhost"`
}

// Address returns the host:port the HTTP server should listen on.
func (h HTTPConfig) Address() string { return fmt.Sprintf("%s:%d", h.Host, h.Port) }

// DatabaseConfig encapsulates all Postgres connectivity options.
type DatabaseConfig struct {
	URL             string        `env:"NEBULA_DATABASE_URL"`
	Host            string        `env:"POSTGRES_HOST"            envDefault:"localhost"`
	Port            int           `env:"POSTGRES_PORT"            envDefault:"5432"`
	User            string        `env:"POSTGRES_USER"            envDefault:"nebula"`
	Password        string        `env:"POSTGRES_PASSWORD"        envDefault:"nebula"`
	Database        string        `env:"POSTGRES_DB"              envDefault:"nebula"`
	SSLMode         string        `env:"POSTGRES_SSLMODE"         envDefault:"disable"`
	MaxConns        int32         `env:"POSTGRES_MAX_CONNS"       envDefault:"20"`
	MinConns        int32         `env:"POSTGRES_MIN_CONNS"       envDefault:"2"`
	MaxConnLifetime time.Duration `env:"POSTGRES_MAX_CONN_LIFETIME" envDefault:"1h"`
}

// DSN returns a libpq-style connection string.
func (d DatabaseConfig) DSN() string {
	if d.URL != "" {
		return d.URL
	}
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s",
		url.QueryEscape(d.User), url.QueryEscape(d.Password),
		d.Host, d.Port, d.Database, d.SSLMode,
	)
}

// RedisConfig encapsulates Redis connectivity.
type RedisConfig struct {
	URL      string `env:"NEBULA_REDIS_URL"`
	Host     string `env:"REDIS_HOST"     envDefault:"localhost"`
	Port     int    `env:"REDIS_PORT"     envDefault:"6379"`
	Password string `env:"REDIS_PASSWORD"`
	DB       int    `env:"REDIS_DB"       envDefault:"0"`
}

// Address returns Redis address.
func (r RedisConfig) Address() string {
	if r.URL != "" {
		return r.URL
	}
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// AuthConfig governs JWT issuing and password hashing peppers.
type AuthConfig struct {
	JWTSecret      string        `env:"NEBULA_JWT_SECRET,required"`
	JWTIssuer      string        `env:"NEBULA_JWT_ISSUER"      envDefault:"nebula-cloud"`
	AccessTTL      time.Duration `env:"NEBULA_JWT_ACCESS_TTL"  envDefault:"15m"`
	RefreshTTL     time.Duration `env:"NEBULA_JWT_REFRESH_TTL" envDefault:"720h"`
	PasswordPepper string        `env:"NEBULA_PASSWORD_PEPPER,required"`
}

// SecretsConfig governs the env-var encryption key used to seal user secrets.
type SecretsConfig struct {
	Key string `env:"NEBULA_SECRETS_KEY,required"` // base64-encoded 32 bytes
}

// GitHubConfig contains GitHub App credentials.
type GitHubConfig struct {
	AppID             string `env:"NEBULA_GITHUB_APP_ID"`
	ClientID          string `env:"NEBULA_GITHUB_APP_CLIENT_ID"`
	ClientSecret      string `env:"NEBULA_GITHUB_APP_CLIENT_SECRET"`
	WebhookSecret     string `env:"NEBULA_GITHUB_APP_WEBHOOK_SECRET"`
	PrivateKeyPath    string `env:"NEBULA_GITHUB_APP_PRIVATE_KEY_PATH"`
	OAuthRedirectURL  string `env:"NEBULA_GITHUB_OAUTH_REDIRECT_URL"`
}

// BuildConfig configures the build worker.
type BuildConfig struct {
	RegistryURL      string `env:"NEBULA_REGISTRY_URL"      envDefault:"registry.nebula.localhost:5000"`
	RegistryInsecure bool   `env:"NEBULA_REGISTRY_INSECURE" envDefault:"true"`
	BuilderImage     string `env:"NEBULA_BUILDER_IMAGE"     envDefault:"paketobuildpacks/builder-jammy-base:latest"`
}

// RuntimeConfig configures the runtime agent.
type RuntimeConfig struct {
	DockerHost           string `env:"NEBULA_DOCKER_HOST"              envDefault:"unix:///var/run/docker.sock"`
	Network              string `env:"NEBULA_RUNTIME_NETWORK"          envDefault:"nebula_platform"`
	BaseDomain           string `env:"NEBULA_BASE_DOMAIN"              envDefault:"nebula.localhost"`
	TraefikUserRoutesDir string `env:"NEBULA_TRAEFIK_USER_ROUTES_DIR"  envDefault:"/etc/traefik/dynamic"`
	TraefikAPIURL        string `env:"NEBULA_TRAEFIK_API_URL"          envDefault:"http://traefik:8080"`
}

// MetricsConfig exposes Prometheus scrape options.
type MetricsConfig struct {
	Enabled bool `env:"NEBULA_METRICS_ENABLED" envDefault:"true"`
	Port    int  `env:"NEBULA_METRICS_PORT"    envDefault:"9100"`
}

// Address returns host:port for the metrics listener.
func (m MetricsConfig) Address() string { return fmt.Sprintf("0.0.0.0:%d", m.Port) }

// TracingConfig governs OpenTelemetry exporters.
type TracingConfig struct {
	Enabled      bool   `env:"NEBULA_TRACING_ENABLED" envDefault:"false"`
	OTLPEndpoint string `env:"NEBULA_OTLP_ENDPOINT"`
}

// RateLimitConfig controls the per-IP / per-user token-bucket limiter.
type RateLimitConfig struct {
	RPS   float64 `env:"NEBULA_RATE_LIMIT_RPS"   envDefault:"20"`
	Burst int     `env:"NEBULA_RATE_LIMIT_BURST" envDefault:"40"`
}

// CORSConfig controls allowed origins for browser clients.
type CORSConfig struct {
	AllowedOrigins []string `env:"NEBULA_CORS_ALLOWED_ORIGINS" envSeparator:"," envDefault:"http://app.nebula.localhost,http://localhost:3000"`
}

// Load reads configuration from the process environment.
func Load() (Config, error) {
	_ = godotenv.Overload(".env", "../.env", "../../.env")

	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return Config{}, fmt.Errorf("config: parse: %w", err)
	}
	if err := cfg.validate(); err != nil {
		return Config{}, fmt.Errorf("config: validate: %w", err)
	}
	return cfg, nil
}

// MustLoad panics if Load fails.
func MustLoad() Config {
	cfg, err := Load()
	if err != nil {
		panic(err)
	}
	return cfg
}

func (c *Config) validate() error {
	switch c.Env {
	case EnvDevelopment, EnvStaging, EnvProduction:
	default:
		return fmt.Errorf("invalid NEBULA_ENV %q", c.Env)
	}

	if c.Env.IsProduction() {
		if strings.HasPrefix(c.Auth.JWTSecret, "change-me") {
			return errors.New("NEBULA_JWT_SECRET must be rotated in production")
		}
		if strings.HasPrefix(c.Auth.PasswordPepper, "change-me") {
			return errors.New("NEBULA_PASSWORD_PEPPER must be rotated in production")
		}
		if strings.Contains(c.Secrets.Key, "fakekey") {
			return errors.New("NEBULA_SECRETS_KEY must be rotated in production")
		}
	}

	if c.HTTP.Port <= 0 || c.HTTP.Port > 65535 {
		return fmt.Errorf("invalid NEBULA_API_PORT %d", c.HTTP.Port)
	}
	if c.Metrics.Enabled && (c.Metrics.Port <= 0 || c.Metrics.Port > 65535) {
		return fmt.Errorf("invalid NEBULA_METRICS_PORT %d", c.Metrics.Port)
	}
	return nil
}
