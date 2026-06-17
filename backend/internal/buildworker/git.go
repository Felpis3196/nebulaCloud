package buildworker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// CloneSource checks out repository contents into workDir (must exist and be empty).
// When commitSHA is non-empty and not an all-zero placeholder, performs a shallow
// fetch of that object; otherwise shallow-clones the given git ref (branch or tag).
// If the configured branch does not exist, retries once with the remote HEAD branch.
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
	branch := normalizeGitRef(gitRef)
	if err := shallowCloneBranch(ctx, workDir, repoURL, branch); err != nil {
		if !isRemoteBranchMissing(err) {
			return err
		}
		resolved, rerr := resolveRemoteDefaultBranch(ctx, repoURL)
		if rerr != nil || resolved == "" || strings.EqualFold(resolved, branch) {
			return fmt.Errorf("%w — configured branch %q not found on remote; update default branch in project settings (e.g. master)", err, branch)
		}
		if err := resetWorkDir(workDir); err != nil {
			return err
		}
		if err2 := shallowCloneBranch(ctx, workDir, repoURL, resolved); err2 != nil {
			return fmt.Errorf("git clone branch %q failed: %w; remote default %q also failed: %w", branch, err, resolved, err2)
		}
		return nil
	}
	return nil
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

func isRemoteBranchMissing(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "remote branch") && strings.Contains(msg, "not found") ||
		strings.Contains(msg, "could not find remote branch")
}

// resolveRemoteDefaultBranch returns the branch name HEAD points to (e.g. master).
func resolveRemoteDefaultBranch(ctx context.Context, repoURL string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "ls-remote", "--symref", repoURL, "HEAD")
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("git ls-remote: %w — %s", err, trimOut(out))
	}
	return parseSymrefHEAD(string(out)), nil
}

func parseSymrefHEAD(output string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "ref: refs/heads/") {
			continue
		}
		rest := strings.TrimPrefix(line, "ref: refs/heads/")
		if i := strings.IndexAny(rest, " \t"); i >= 0 {
			rest = rest[:i]
		}
		rest = strings.TrimSpace(rest)
		if rest != "" {
			return rest
		}
	}
	return ""
}

func resetWorkDir(workDir string) error {
	entries, err := os.ReadDir(workDir)
	if err != nil {
		return err
	}
	for _, e := range entries {
		if err := os.RemoveAll(filepath.Join(workDir, e.Name())); err != nil {
			return err
		}
	}
	return nil
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
	return Excerpt(string(b), DefaultExcerptLimit)
}
