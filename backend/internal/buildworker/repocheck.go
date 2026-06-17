package buildworker

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// RepoAnalysis holds advisory results from repository checks (never blocks deploy).
type RepoAnalysis struct {
	OK       bool     `json:"ok"`
	Warnings []string `json:"warnings,omitempty"`
	Hints    []string `json:"hints,omitempty"`
	Stack    string   `json:"stack,omitempty"`
}

type knownRepoRule struct {
	match func(norm string) bool
	warn  string
	hint  string
}

var knownRepoRules = []knownRepoRule{
	{
		match: func(n string) bool { return strings.Contains(n, "github.com/docker/getting-started") },
		warn:  "This repo builds docs with mkdocs on python:alpine (Python 3.13+). The build often fails unless you pin python:3.12-alpine in the Dockerfile.",
		hint:  "Try https://github.com/docker/welcome-to-docker for a quick smoke test.",
	},
	{
		match: func(n string) bool { return strings.Contains(n, "github.com/octocat/hello-world") },
		warn:  "Hello-World has no Dockerfile or supported buildpack markers in the repository root.",
		hint:  "Add a Dockerfile or connect a repo with package.json, go.mod, requirements.txt, or pyproject.toml at the root.",
	},
}

var recommendedRepos = []string{
	"https://github.com/docker/welcome-to-docker",
	"https://github.com/tiangolo/uvicorn-gunicorn-fastapi-docker",
}

// AnalyzeRepoURL performs fast URL-level checks without cloning.
func AnalyzeRepoURL(repoURL, branch string) RepoAnalysis {
	out := RepoAnalysis{OK: true}
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return RepoAnalysis{OK: false, Warnings: []string{"Repository URL is required."}}
	}
	norm := normalizeRepoURL(repoURL)
	owner, repo, err := parseGitHubOwnerRepo(norm)
	if err != nil {
		out.Warnings = append(out.Warnings, err.Error())
		out.Hints = append(out.Hints, "Use a GitHub HTTPS URL like https://github.com/owner/repo")
		return out
	}
	_ = owner
	_ = repo

	for _, rule := range knownRepoRules {
		if rule.match(norm) {
			if rule.warn != "" {
				out.Warnings = append(out.Warnings, rule.warn)
			}
			if rule.hint != "" {
				out.Hints = append(out.Hints, rule.hint)
			}
		}
	}

	branch = normalizeGitRef(branch)
	if branch == "" {
		branch = "main"
	}
	out.Hints = append(out.Hints, fmt.Sprintf("Default branch configured as %q — verify it exists on the remote.", branch))
	return out
}

// ProbeGitHubRepo calls the GitHub REST API for public repo metadata.
func ProbeGitHubRepo(ctx context.Context, owner, repo, branch string) ([]string, []string, error) {
	if owner == "" || repo == "" {
		return nil, nil, fmt.Errorf("invalid github owner/repo")
	}
	reqCtx, cancel := context.WithTimeout(ctx, 12*time.Second)
	defer cancel()

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "nebula-cloud")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return []string{"Could not reach GitHub API to verify the repository."}, nil, nil
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 64*1024))

	if resp.StatusCode == http.StatusNotFound {
		return []string{
			"Repository not found or is private. Private repos need a linked GitHub App installation.",
		}, []string{"Connect via GitHub OAuth and set the installation ID in project settings."}, nil
	}
	if resp.StatusCode == http.StatusForbidden {
		return []string{"GitHub API rate limit or access denied — analysis is partial."}, nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("github api status %d", resp.StatusCode)
	}

	var meta struct {
		Private       bool   `json:"private"`
		DefaultBranch string `json:"default_branch"`
	}
	if err := json.Unmarshal(body, &meta); err != nil {
		return nil, nil, err
	}

	var warnings, hints []string
	if meta.Private {
		warnings = append(warnings, "This is a private repository. Ensure GitHub App installation is linked before deploying.")
	}
	branch = normalizeGitRef(branch)
	if branch == "" {
		branch = "main"
	}
	if meta.DefaultBranch != "" && !strings.EqualFold(meta.DefaultBranch, branch) {
		warnings = append(warnings, fmt.Sprintf(
			"Configured branch %q differs from GitHub default branch %q.",
			branch, meta.DefaultBranch,
		))
		hints = append(hints, fmt.Sprintf("Consider setting default branch to %q in project settings.", meta.DefaultBranch))
	}
	return warnings, hints, nil
}

// MergeAnalysis combines two RepoAnalysis values.
func MergeAnalysis(base, extra RepoAnalysis) RepoAnalysis {
	out := base
	if !extra.OK {
		out.OK = false
	}
	out.Warnings = append(out.Warnings, extra.Warnings...)
	out.Hints = append(out.Hints, extra.Hints...)
	if out.Stack == "" && extra.Stack != "" {
		out.Stack = extra.Stack
	}
	return out
}

// AnalyzeWorkspace inspects a cloned workspace before build.
func AnalyzeWorkspace(workDir, dockerfileRel string) RepoAnalysis {
	out := RepoAnalysis{OK: true}
	mode, detected, err := DetectStack(workDir, dockerfileRel)
	if err != nil {
		out.Warnings = append(out.Warnings, err.Error())
		out.Hints = append(out.Hints,
			"Supported stacks: Dockerfile, package.json (Node), requirements.txt/pyproject.toml (Python), go.mod (Go), *.csproj (.NET).",
		)
		return out
	}
	out.Stack = detected

	if mode == ModeDockerfile {
		dfPath, pathErr := resolveDockerfilePath(workDir, dockerfileRel)
		if pathErr == nil {
			if content, readErr := os.ReadFile(dfPath); readErr == nil {
				lower := strings.ToLower(string(content))
				if strings.Contains(lower, "from python:alpine") ||
					strings.Contains(lower, "from python:3-alpine") {
					if strings.Contains(lower, "mkdocs") || strings.Contains(lower, "pip install") {
						out.Warnings = append(out.Warnings,
							"Dockerfile uses python:alpine which may resolve to Python 3.13+. Old mkdocs/htmlmin builds can fail.",
						)
						out.Hints = append(out.Hints, "Pin FROM python:3.12-alpine if docs dependencies are outdated.")
					}
				}
			}
		}
	}
	return out
}

// FullRepoAnalysis runs URL checks plus optional GitHub probe.
func FullRepoAnalysis(ctx context.Context, repoURL, branch string) RepoAnalysis {
	base := AnalyzeRepoURL(repoURL, branch)
	norm := normalizeRepoURL(repoURL)
	owner, repo, err := parseGitHubOwnerRepo(norm)
	if err != nil {
		return base
	}
	warnings, hints, probeErr := ProbeGitHubRepo(ctx, owner, repo, branch)
	if probeErr != nil {
		base.Hints = append(base.Hints, "Could not complete GitHub metadata check.")
		return base
	}
	extra := RepoAnalysis{OK: true, Warnings: warnings, Hints: hints}
	return MergeAnalysis(base, extra)
}

func normalizeRepoURL(raw string) string {
	u := strings.TrimSpace(strings.ToLower(raw))
	u = strings.TrimSuffix(u, ".git")
	u = strings.TrimSuffix(u, "/")
	if strings.HasPrefix(u, "git@github.com:") {
		u = "https://github.com/" + strings.TrimPrefix(u, "git@github.com:")
	}
	if parsed, err := url.Parse(u); err == nil && parsed.Host != "" {
		path := strings.TrimSuffix(parsed.Path, ".git")
		return parsed.Scheme + "://" + parsed.Host + path
	}
	return u
}

func parseGitHubOwnerRepo(norm string) (owner, repo string, err error) {
	parsed, parseErr := url.Parse(norm)
	if parseErr != nil || parsed.Host == "" {
		return "", "", fmt.Errorf("invalid repository URL")
	}
	host := strings.ToLower(parsed.Host)
	if host != "github.com" && host != "www.github.com" {
		return "", "", fmt.Errorf("only github.com HTTPS URLs are supported for automated analysis (other hosts may still work at build time)")
	}
	parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
	if len(parts) < 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("URL must look like https://github.com/owner/repo")
	}
	return parts[0], strings.TrimSuffix(parts[1], ".git"), nil
}

// RecommendedRepos returns copy-paste URLs for smoke tests.
func RecommendedRepos() []string {
	out := make([]string, len(recommendedRepos))
	copy(out, recommendedRepos)
	return out
}

// RepoWarningsFromURL is a convenience for computed project DTO fields.
func RepoWarningsFromURL(repoURL, branch string) []string {
	a := AnalyzeRepoURL(repoURL, branch)
	return a.Warnings
}
