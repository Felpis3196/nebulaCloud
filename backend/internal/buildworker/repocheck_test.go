package buildworker

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAnalyzeRepoURL_knownWarnings(t *testing.T) {
	a := AnalyzeRepoURL("https://github.com/docker/getting-started", "main")
	if a.OK != true {
		t.Fatalf("expected ok=true")
	}
	if len(a.Warnings) == 0 {
		t.Fatal("expected warning for getting-started")
	}
	if !strings.Contains(strings.Join(a.Warnings, " "), "mkdocs") {
		t.Fatalf("warnings=%v", a.Warnings)
	}
}

func TestAnalyzeRepoURL_invalidURL(t *testing.T) {
	a := AnalyzeRepoURL("", "main")
	if a.OK {
		t.Fatal("expected not ok for empty url")
	}
	a2 := AnalyzeRepoURL("https://gitlab.com/foo/bar", "main")
	if len(a2.Warnings) == 0 {
		t.Fatal("expected non-github warning")
	}
}

func TestParseGitHubOwnerRepo(t *testing.T) {
	o, r, err := parseGitHubOwnerRepo("https://github.com/docker/welcome-to-docker")
	if err != nil || o != "docker" || r != "welcome-to-docker" {
		t.Fatalf("owner=%q repo=%q err=%v", o, r, err)
	}
}

func TestNormalizeRepoURL(t *testing.T) {
	got := normalizeRepoURL("https://github.com/Docker/Welcome-To-Docker.git/")
	want := "https://github.com/docker/welcome-to-docker"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestAnalyzeWorkspace_dockerfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM node:20\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	a := AnalyzeWorkspace(dir, "")
	if a.Stack != "dockerfile" {
		t.Fatalf("stack=%q", a.Stack)
	}
}

func TestAnalyzeWorkspace_noStack(t *testing.T) {
	dir := t.TempDir()
	a := AnalyzeWorkspace(dir, "")
	if len(a.Warnings) == 0 {
		t.Fatal("expected warning for empty repo")
	}
}

func TestMergeAnalysis(t *testing.T) {
	base := RepoAnalysis{OK: true, Warnings: []string{"a"}}
	extra := RepoAnalysis{Warnings: []string{"b"}, Hints: []string{"hint"}}
	m := MergeAnalysis(base, extra)
	if len(m.Warnings) != 2 || len(m.Hints) != 1 {
		t.Fatalf("%+v", m)
	}
}

func TestProbeGitHubRepo_public(t *testing.T) {
	if testing.Short() {
		t.Skip("network")
	}
	ctx := context.Background()
	warnings, hints, err := ProbeGitHubRepo(ctx, "docker", "welcome-to-docker", "main")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	_ = warnings
	_ = hints
}

func TestProbeGitHubRepo_notFound(t *testing.T) {
	if testing.Short() {
		t.Skip("network")
	}
	ctx := context.Background()
	warnings, _, err := ProbeGitHubRepo(ctx, "nebula-cloud-nonexistent-org-xyz", "no-such-repo-abc", "main")
	if err != nil {
		t.Fatalf("err=%v", err)
	}
	if len(warnings) == 0 {
		t.Fatal("expected not-found warning")
	}
}

func TestRecommendedRepos(t *testing.T) {
	repos := RecommendedRepos()
	if len(repos) < 2 {
		t.Fatalf("repos=%v", repos)
	}
}
