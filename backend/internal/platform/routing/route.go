package routing

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

// DeployRoute is the single source of truth for Traefik file routes and public URLs.
type DeployRoute struct {
	RouteID       string
	Host          string
	ContainerName string
	Port          int
	FilePath      string
	PublicURL     string
}

// NewDeployRoute builds route metadata for a web service deployment.
func NewDeployRoute(dir, serviceSlug, projectSlug, baseDomain string, serviceID uuid.UUID, port int) DeployRoute {
	host := FormatServiceHost(serviceSlug, projectSlug, baseDomain)
	routeID := RouteIDForService(serviceID)
	cname := ContainerNameForService(serviceID)
	if port <= 0 {
		port = 8080
	}
	return DeployRoute{
		RouteID:       routeID,
		Host:          host,
		ContainerName: cname,
		Port:          port,
		FilePath:      filepath.Join(strings.TrimSpace(dir), routeID+".yml"),
		PublicURL:     "http://" + host,
	}
}

// RouteIDForService returns a stable Traefik router/file prefix for the service.
func RouteIDForService(serviceID uuid.UUID) string {
	return "nebula-" + shortHex(serviceID, 8)
}

// ContainerNameForService returns the stable Docker container name for a service.
func ContainerNameForService(serviceID uuid.UUID) string {
	return "nebula-svc-" + shortHex(serviceID, 12)
}

func shortHex(id uuid.UUID, n int) string {
	s := strings.ReplaceAll(strings.ToLower(id.String()), "-", "")
	if len(s) < n {
		return s + strings.Repeat("0", n-len(s))
	}
	return s[:n]
}

// RouteFileBody returns the Traefik dynamic YAML for this route.
func (r DeployRoute) RouteFileBody() string {
	backend := fmt.Sprintf("http://%s:%d", r.ContainerName, r.Port)
	return fmt.Sprintf(`http:
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
          - url: %q
`, r.RouteID, yamlQuote(r.Host), r.RouteID, r.RouteID, backend)
}
