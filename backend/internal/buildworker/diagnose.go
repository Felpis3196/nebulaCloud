package buildworker

import "strings"

type failureRule struct {
	match func(lower string) bool
	hint  string
}

var failureRules = []failureRule{
	{
		match: func(s string) bool {
			return strings.Contains(s, "no module named 'cgi'") ||
				(strings.Contains(s, "htmlmin") && strings.Contains(s, "mkdocs"))
		},
		hint: "App Dockerfile uses Python 3.13+ but docs dependencies need an older Python. Pin python:3.12-alpine in the Dockerfile, or update mkdocs/htmlmin.",
	},
	{
		match: func(s string) bool {
			return strings.Contains(s, "remote branch") && strings.Contains(s, "not found")
		},
		hint: "Git branch mismatch. Set the project default branch to match the repository (e.g. main vs master).",
	},
	{
		match: func(s string) bool {
			return strings.Contains(s, "pull access denied") ||
				(strings.Contains(s, "repository does not exist") && strings.Contains(s, "registry"))
		},
		hint: "Registry auth or image name issue. Check NEBULA_REGISTRY_URL and Docker Desktop insecure-registries for localhost:5000.",
	},
	{
		match: func(s string) bool {
			return strings.Contains(s, "no such host") &&
				(strings.Contains(s, "5000") || strings.Contains(s, "registry"))
		},
		hint: "Registry hostname not reachable from host Docker. Use localhost:5000 in dev.",
	},
	{
		match: func(s string) bool {
			return strings.Contains(s, "no supported stack detected") ||
				strings.Contains(s, "no usable dockerfile")
		},
		hint: "No Dockerfile and no supported buildpack markers in the repo root. Add a Dockerfile or a recognized stack file (package.json, go.mod, etc.).",
	},
	{
		match: func(s string) bool {
			return strings.Contains(s, "failed to solve") &&
				(strings.Contains(s, "dockerfile") || strings.Contains(s, "docker build"))
		},
		hint: "Docker build failed in the repository Dockerfile. Scroll the build logs for the failing RUN step.",
	},
}

// DiagnoseFailure returns a short human-readable hint for common build failures, or "".
func DiagnoseFailure(output string) string {
	lower := strings.ToLower(output)
	for _, rule := range failureRules {
		if rule.match(lower) {
			return rule.hint
		}
	}
	return ""
}
