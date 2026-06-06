// Package terminal attaches to running user containers via the Docker API.
package terminal

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// Session streams stdin/stdout for an interactive shell in a service container.
type Session struct {
	execID      string
	containerID string
	cli         *client.Client
	resizeMu    sync.Mutex
}

// Open finds the container labeled nebula_service=<serviceID> and starts /bin/sh.
func Open(ctx context.Context, serviceID string) (*Session, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	cid, err := findContainer(ctx, cli, serviceID)
	if err != nil {
		_ = cli.Close()
		return nil, err
	}
	execResp, err := cli.ContainerExecCreate(ctx, cid, container.ExecOptions{
		AttachStdin:  true,
		AttachStdout: true,
		AttachStderr: true,
		Tty:          true,
		Cmd:          []string{"/bin/sh"},
	})
	if err != nil {
		_ = cli.Close()
		return nil, fmt.Errorf("exec create: %w", err)
	}
	return &Session{execID: execResp.ID, containerID: cid, cli: cli}, nil
}

func findContainer(ctx context.Context, cli *client.Client, serviceID string) (string, error) {
	serviceID = strings.TrimSpace(serviceID)
	args := filters.NewArgs(filters.Arg("label", "nebula_service="+serviceID))
	list, err := cli.ContainerList(ctx, container.ListOptions{Filters: args, Limit: 1})
	if err != nil {
		return "", err
	}
	if len(list) == 0 {
		return "", fmt.Errorf("no running container for service %s", serviceID)
	}
	return list[0].ID, nil
}

// Attach returns hijacked streams for the exec session.
func (s *Session) Attach(ctx context.Context) (types.HijackedResponse, error) {
	return s.cli.ContainerExecAttach(ctx, s.execID, container.ExecAttachOptions{Tty: true})
}

// ResizePTY updates terminal geometry.
func (s *Session) Resize(ctx context.Context, cols, rows uint) error {
	s.resizeMu.Lock()
	defer s.resizeMu.Unlock()
	return s.cli.ContainerExecResize(ctx, s.execID, container.ResizeOptions{
		Height: rows,
		Width:  cols,
	})
}

// Close releases the Docker client.
func (s *Session) Close() error {
	if s.cli == nil {
		return nil
	}
	return s.cli.Close()
}

// Copy bridges reader/writer until ctx is done or io ends.
func Copy(ctx context.Context, dst io.Writer, src io.Reader) {
	buf := make([]byte, 32*1024)
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}
		n, err := src.Read(buf)
		if n > 0 {
			_, _ = dst.Write(buf[:n])
		}
		if err != nil {
			return
		}
	}
}
