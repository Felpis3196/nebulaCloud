package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

// VerifyOptions configures post-write route checks.
type VerifyOptions struct {
	TraefikAPIURL string
	Retries       int
	RetryDelay    time.Duration
	InitialDelay  time.Duration
}

// VerifyRoute checks the route is reachable through Traefik (HTTP probe is authoritative on Docker Desktop).
func VerifyRoute(ctx context.Context, route DeployRoute, opts VerifyOptions) error {
	if opts.Retries <= 0 {
		opts.Retries = 15
	}
	if opts.RetryDelay <= 0 {
		opts.RetryDelay = 3 * time.Second
	}
	if opts.InitialDelay > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(opts.InitialDelay):
		}
	}
	api := strings.TrimSpace(opts.TraefikAPIURL)
	if api == "" {
		api = "http://traefik:8080"
	}

	var lastProbe error
	for attempt := 0; attempt < opts.Retries; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if attempt > 0 && attempt%5 == 0 {
			_ = bumpRouteFile(route.FilePath)
		}
		if err := probeHTTP(ctx, route.Host); err == nil {
			return nil
		} else {
			lastProbe = err
		}
		// API check is informational only (file provider may lag on Windows).
		_ = verifyTraefikRouter(ctx, api, route.Host, route.RouteID)
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(opts.RetryDelay):
		}
	}
	if lastProbe != nil {
		return fmt.Errorf("traefik route verify failed for host %q after %d attempts: %w (try: docker compose restart traefik)", route.Host, opts.Retries, lastProbe)
	}
	return fmt.Errorf("traefik route verify failed for host %q", route.Host)
}

func verifyTraefikRouter(ctx context.Context, apiURL, host, routeID string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(apiURL, "/")+"/api/http/routers", nil)
	if err != nil {
		return err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("traefik api: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("traefik api status %d", res.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if err != nil {
		return err
	}
	var routers []struct {
		Name   string `json:"name"`
		Rule   string `json:"rule"`
		Status string `json:"status"`
	}
	if err := json.Unmarshal(body, &routers); err != nil {
		return err
	}
	wantHost := "Host(`" + host + "`)"
	for _, r := range routers {
		if !strings.Contains(r.Name, routeID) {
			continue
		}
		if strings.Contains(r.Rule, wantHost) && r.Status == "enabled" {
			return nil
		}
	}
	return fmt.Errorf("router %q with rule containing %q not found in traefik api", routeID, wantHost)
}

func probeHTTP(ctx context.Context, host string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://traefik/", nil)
	if err != nil {
		return err
	}
	req.Host = host
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("http probe: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode == http.StatusNotFound {
		return fmt.Errorf("http probe returned 404 for Host %q", host)
	}
	if res.StatusCode >= 500 {
		return fmt.Errorf("http probe returned %d for Host %q", res.StatusCode, host)
	}
	return nil
}

func bumpRouteFile(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return err
	}
	now := time.Now()
	return os.Chtimes(path, now, now)
}

// SyncDeployRoute writes the route file, removes stale nebula-*.yml, and verifies via HTTP through Traefik.
func SyncDeployRoute(ctx context.Context, dir string, route DeployRoute, verify VerifyOptions) error {
	if err := WriteUserRoute(dir, route); err != nil {
		return err
	}
	RemoveStaleUserRoutes(dir, route.RouteID)
	_ = bumpRouteFile(route.FilePath)
	if verify.InitialDelay <= 0 {
		verify.InitialDelay = 2 * time.Second
	}
	if err := VerifyRoute(ctx, route, verify); err != nil {
		RemoveUserRoute(dir, route.RouteID)
		return err
	}
	return nil
}
