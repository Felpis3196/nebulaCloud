package main

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ensureContainerSlotFree removes any container using the stable service name before docker run.
func ensureContainerSlotFree(ctx context.Context, name string, serviceID uuid.UUID) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return fmt.Errorf("empty container name")
	}
	// Remove by exact name and by service label (covers renamed/orphan instances).
	removeContainersByFilter(ctx, "name="+name)
	removeContainersByFilter(ctx, "label=nebula_service="+serviceID.String())

	var lastOut string
	for attempt := 0; attempt < 6; attempt++ {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		cmd := exec.CommandContext(ctx, "docker", "rm", "-f", name)
		out, _ := cmd.CombinedOutput()
		lastOut = strings.TrimSpace(string(out))
		removeContainersByFilter(ctx, "name="+name)
		ids, err := containerIDsByName(ctx, name)
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			return nil
		}
		for _, id := range ids {
			_ = exec.CommandContext(ctx, "docker", "rm", "-f", id).Run()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(400 * time.Millisecond):
		}
	}
	return fmt.Errorf("container name %q still in use after cleanup: %s", name, lastOut)
}

func containerIDsByName(ctx context.Context, name string) ([]string, error) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "-aq", "--filter", "name="+name)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return strings.Fields(strings.TrimSpace(string(out))), nil
}

func removeContainersByFilter(ctx context.Context, filter string) {
	cmd := exec.CommandContext(ctx, "docker", "ps", "-aq", "--filter", filter)
	out, err := cmd.Output()
	if err != nil {
		return
	}
	for _, id := range strings.Fields(strings.TrimSpace(string(out))) {
		if id == "" {
			continue
		}
		_ = exec.CommandContext(ctx, "docker", "rm", "-f", id).Run()
	}
}

func dockerRun(ctx context.Context, args []string) ([]byte, error) {
	return exec.CommandContext(ctx, "docker", args...).CombinedOutput()
}

func isNameConflict(err error, out string) bool {
	if err == nil {
		return false
	}
	s := strings.ToLower(err.Error() + " " + out)
	return strings.Contains(s, "already in use") || strings.Contains(s, "conflict")
}
