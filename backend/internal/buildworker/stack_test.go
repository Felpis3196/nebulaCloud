package buildworker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectStack_Dockerfile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mode, det, err := DetectStack(dir, "Dockerfile")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if mode != ModeDockerfile || det != "dockerfile" {
		t.Fatalf("got mode=%v det=%s", mode, det)
	}
}

func TestDetectStack_Node(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	mode, det, err := DetectStack(dir, "Dockerfile")
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if mode != ModeBuildpack || det != "nodejs" {
		t.Fatalf("got mode=%v det=%s", mode, det)
	}
}

func TestDetectStack_DockerfileWinsOverNode(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\n"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "package.json"), []byte("{}"), 0o644)
	mode, det, err := DetectStack(dir, "Dockerfile")
	if err != nil {
		t.Fatal(err)
	}
	if mode != ModeDockerfile || det != "dockerfile" {
		t.Fatalf("got mode=%v det=%s", mode, det)
	}
}

func TestDetectStack_NoStack(t *testing.T) {
	dir := t.TempDir()
	_, _, err := DetectStack(dir, "Dockerfile")
	if err != ErrNoStack {
		t.Fatalf("expected ErrNoStack, got %v", err)
	}
}

func TestResolveDockerfilePath_escape(t *testing.T) {
	dir := t.TempDir()
	_, err := resolveDockerfilePath(dir, "..")
	if err == nil {
		t.Fatal("expected error for path escape")
	}
}
