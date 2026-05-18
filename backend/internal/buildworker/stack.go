// Package buildworker contains logic shared by cmd/build-worker (clone, stack
// detection, buildpack vs Dockerfile).
package buildworker

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ErrNoStack is returned when the repository has neither a usable Dockerfile
// nor any supported buildpack entry signal.
var ErrNoStack = fmt.Errorf("no supported stack detected: add a Dockerfile, or package.json (Node), requirements.txt/pyproject.toml (Python), go.mod (Go), or a .csproj (.NET)")

// BuildMode selects how the worker produces the OCI image.
type BuildMode int

const (
	ModeDockerfile BuildMode = iota
	ModeBuildpack
)

// DetectStack applies ARCHITECTURE.md ordering: Dockerfile first, then
// buildpack heuristics. detected is stored on builds.detected_stack (dockerfile,
// nodejs, python, go, dotnet).
func DetectStack(workDir, dockerfileRel string) (mode BuildMode, detected string, err error) {
	dfPath, err := resolveDockerfilePath(workDir, dockerfileRel)
	if err != nil {
		return 0, "", err
	}
	if _, statErr := os.Stat(dfPath); statErr == nil {
		return ModeDockerfile, "dockerfile", nil
	}

	if _, e := os.Stat(filepath.Join(workDir, "package.json")); e == nil {
		return ModeBuildpack, "nodejs", nil
	}
	if _, e := os.Stat(filepath.Join(workDir, "requirements.txt")); e == nil {
		return ModeBuildpack, "python", nil
	}
	if _, e := os.Stat(filepath.Join(workDir, "pyproject.toml")); e == nil {
		return ModeBuildpack, "python", nil
	}
	if _, e := os.Stat(filepath.Join(workDir, "go.mod")); e == nil {
		return ModeBuildpack, "go", nil
	}
	matches, _ := filepath.Glob(filepath.Join(workDir, "*.csproj"))
	if len(matches) > 0 {
		return ModeBuildpack, "dotnet", nil
	}

	return 0, "", ErrNoStack
}

func resolveDockerfilePath(workDir, rel string) (string, error) {
	rel = strings.TrimSpace(rel)
	if rel == "" {
		rel = "Dockerfile"
	}
	rel = filepath.Clean(rel)
	if rel == "." || rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return "", fmt.Errorf("invalid dockerfile_path")
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("dockerfile_path must be relative")
	}
	full := filepath.Join(workDir, rel)
	cwd := filepath.Clean(workDir)
	cp := filepath.Clean(full)
	if cp == cwd {
		return "", fmt.Errorf("dockerfile_path resolves to workspace root")
	}
	if !strings.HasPrefix(cp, cwd+string(os.PathSeparator)) && cp != cwd {
		return "", fmt.Errorf("dockerfile_path escapes workspace")
	}
	return full, nil
}
