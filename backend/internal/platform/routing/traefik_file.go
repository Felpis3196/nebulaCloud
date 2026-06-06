package routing

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// WriteUserRoute registers an HTTP route via Traefik file provider (Docker Desktop dev).
func WriteUserRoute(dir string, route DeployRoute) error {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return fmt.Errorf("routing: empty traefik user routes dir")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("routing: mkdir: %w", err)
	}
	path := filepath.Join(dir, route.RouteID+".yml")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(route.RouteFileBody()), 0o644); err != nil {
		return fmt.Errorf("routing: write: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("routing: rename: %w", err)
	}
	return nil
}

// WriteUserRouteLegacy writes a route from discrete fields (tests / compatibility).
func WriteUserRouteLegacy(dir, routeID, host, containerName string, port int) error {
	if port <= 0 {
		port = 8080
	}
	body := fmt.Sprintf(`http:
  routers:
    %s:
      rule: Host(%s)
      entryPoints:
        - web
      service: %s
  services:
    %s:
      loadBalancer:
        servers:
          - url: %s
`, routeID, yamlQuote(host), routeID, routeID, yamlQuote(fmt.Sprintf("http://%s:%d", containerName, port)))
	path := filepath.Join(strings.TrimSpace(dir), routeID+".yml")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, []byte(body), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// RemoveUserRoute deletes a user route file (best-effort).
func RemoveUserRoute(dir, routeID string) {
	if dir == "" || routeID == "" {
		return
	}
	_ = os.Remove(filepath.Join(dir, routeID+".yml"))
}

// RemoveStaleUserRoutes removes other nebula-*.yml files in dir (orphan routes after redeploy).
func RemoveStaleUserRoutes(dir, keepRouteID string) {
	dir = strings.TrimSpace(dir)
	if dir == "" {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	keep := keepRouteID + ".yml"
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasPrefix(name, "nebula-") || !strings.HasSuffix(name, ".yml") {
			continue
		}
		if name == keep {
			continue
		}
		_ = os.Remove(filepath.Join(dir, name))
	}
}

func yamlQuote(s string) string {
	return fmt.Sprintf("`%s`", strings.ReplaceAll(s, "`", ""))
}
