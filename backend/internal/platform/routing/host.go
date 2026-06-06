package routing

import (
	"strings"
)

// FormatServiceHost builds {service}.{project}.{baseDomain} for Traefik Host() rules.
// Only service/project slugs are normalized; baseDomain keeps dots (e.g. nebula.localhost).
func FormatServiceHost(serviceSlug, projectSlug, baseDomain string) string {
	base := strings.TrimSpace(baseDomain)
	base = strings.TrimPrefix(strings.ToLower(base), ".")
	return SanitizeSlug(serviceSlug) + "." + SanitizeSlug(projectSlug) + "." + base
}

// ServiceURL returns the public HTTP URL for a web service.
func ServiceURL(serviceSlug, projectSlug, baseDomain string) string {
	return "http://" + FormatServiceHost(serviceSlug, projectSlug, baseDomain)
}

// SanitizeSlug lowercases and maps slug characters to [a-z0-9-].
func SanitizeSlug(s string) string {
	var b strings.Builder
	s = strings.ToLower(strings.TrimSpace(s))
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
			continue
		}
		if r == '_' {
			b.WriteRune('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "svc"
	}
	return out
}
