package buildworker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CloneSource checks out repository contents into workDir (must exist and be empty).
// When commitSHA is non-empty and not an all-zero placeholder, performs a shallow
// fetch of that object; otherwise shallow-clones the given git ref (branch or tag).
func CloneSource(ctx context.Context, workDir, repoURL, gitRef string, commitSHA *string) error {
	repoURL = strings.TrimSpace(repoURL)
	if repoURL == "" {
		return fmt.Errorf("empty repository URL")
	}
	sha := ""
	if commitSHA != nil {
		sha = strings.TrimSpace(*commitSHA)
	}
	if sha != "" && !isAllZeroSHA(sha) {
		return shallowFetchCommit(ctx, workDir, repoURL, sha)
	}
	return shallowCloneBranch(ctx, workDir, repoURL, normalizeGitRef(gitRef))
}

func normalizeGitRef(ref string) string {
	ref = strings.TrimSpace(ref)
	ref = strings.TrimPrefix(ref, "refs/heads/")
	ref = strings.TrimPrefix(ref, "refs/tags/")
	if ref == "" {
		return "main"
	}
	return ref
}

func isAllZeroSHA(s string) bool {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return true
	}
	for _, c := range s {
		if c != '0' {
			return false
		}
	}
	return true
}

func shallowCloneBranch(ctx context.Context, workDir, repoURL, branch string) error {
	cmd := exec.CommandContext(ctx, "git", "clone",
		"--depth", "1", "--branch", branch, "--single-branch", repoURL, ".")
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git clone: %w — %s", err, trimOut(out))
	}
	return nil
}

func shallowFetchCommit(ctx context.Context, workDir, repoURL, sha string) error {
	init := exec.CommandContext(ctx, "git", "init")
	init.Dir = workDir
	init.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := init.CombinedOutput(); err != nil {
		return fmt.Errorf("git init: %w — %s", err, trimOut(out))
	}
	remote := exec.CommandContext(ctx, "git", "remote", "add", "origin", repoURL)
	remote.Dir = workDir
	remote.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := remote.CombinedOutput(); err != nil {
		return fmt.Errorf("git remote add: %w — %s", err, trimOut(out))
	}
	fetch := exec.CommandContext(ctx, "git", "fetch", "--depth", "1", "origin", sha)
	fetch.Dir = workDir
	fetch.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := fetch.CombinedOutput(); err != nil {
		return fmt.Errorf("git fetch %s: %w — %s", sha, err, trimOut(out))
	}
	co := exec.CommandContext(ctx, "git", "checkout", "-q", "FETCH_HEAD")
	co.Dir = workDir
	co.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if out, err := co.CombinedOutput(); err != nil {
		return fmt.Errorf("git checkout: %w — %s", err, trimOut(out))
	}
	return nil
}

func trimOut(b []byte) string {
	s := strings.TrimSpace(string(b))
	if len(s) > 4000 {
		return s[:4000] + "…"
	}
	return s
}
