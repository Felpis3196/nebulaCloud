package routing

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// WaitContainerReady polls the app inside the container until HTTP responds or timeout.
func WaitContainerReady(ctx context.Context, containerName string, port int, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = 60 * time.Second
	}
	if port <= 0 {
		port = 8080
	}
	deadline, ok := ctx.Deadline()
	if !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
		deadline = time.Now().Add(timeout)
	}
	var lastErr error
	for time.Now().Before(deadline) {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		cmd := exec.CommandContext(ctx, "docker", "exec", containerName,
			"wget", "-qO-", "--timeout=2", fmt.Sprintf("http://127.0.0.1:%d/", port))
		if out, err := cmd.CombinedOutput(); err == nil && len(out) > 0 {
			return nil
		} else if err != nil {
			lastErr = fmt.Errorf("%w — %s", err, strings.TrimSpace(string(out)))
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
	}
	if lastErr != nil {
		return fmt.Errorf("container %s not ready on port %d: %w", containerName, port, lastErr)
	}
	return fmt.Errorf("container %s not ready on port %d before timeout", containerName, port)
}
