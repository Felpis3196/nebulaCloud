package config_test

import (
	"os"
	"testing"

	"github.com/nebulacloud/nebula/internal/platform/config"
)

func TestLoadDevelopmentDefaults(t *testing.T) {
	t.Setenv("NEBULA_ENV", "development")
	t.Setenv("NEBULA_JWT_SECRET", "test-jwt-secret-at-least-32-chars-long")
	t.Setenv("NEBULA_PASSWORD_PEPPER", "test-pepper")
	t.Setenv("NEBULA_SECRETS_KEY", "bmVidWxhLWRldi1zZWNyZXRzLWtleS0zMmJ5dGVzISE=")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Runtime.Network != "nebula_platform" {
		t.Fatalf("expected nebula_platform network, got %q", cfg.Runtime.Network)
	}
	if cfg.Terminal.Enabled {
		t.Fatal("terminal should default to disabled")
	}
}

func TestLoadRequiresSecrets(t *testing.T) {
	_ = os.Unsetenv("NEBULA_JWT_SECRET")
	_, err := config.Load()
	if err == nil {
		t.Fatal("expected error without JWT secret")
	}
}
