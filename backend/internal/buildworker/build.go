package buildworker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// RunDockerBuild runs docker build in workDir, using dockerfileRel relative to workDir.
func RunDockerBuild(ctx context.Context, workDir, imageRef, dockerfileRel string) error {
	rel := strings.TrimSpace(dockerfileRel)
	if rel == "" {
		rel = "Dockerfile"
	}
	rel = filepath.Clean(rel)
	cmd := exec.CommandContext(ctx, "docker", "build", "-t", imageRef, "-f", rel, ".")
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "DOCKER_BUILDKIT=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker build: %w — %s", err, trimOut(out))
	}
	return nil
}

// RunPackBuild runs pack build against the local Docker daemon (image loaded locally, then pushed separately).
func RunPackBuild(ctx context.Context, workDir, imageRef, builderImage string) error {
	bi := strings.TrimSpace(builderImage)
	if bi == "" {
		bi = "paketobuildpacks/builder-jammy-base:latest"
	}
	cmd := exec.CommandContext(ctx, "pack", "build", imageRef,
		"--path", workDir,
		"--builder", bi,
		"--trust-builder",
		"--pull-policy", "if-not-present",
	)
	cmd.Dir = workDir
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("pack build: %w — %s", err, trimOut(out))
	}
	return nil
}
