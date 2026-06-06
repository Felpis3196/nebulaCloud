package buildworker

import (
	"context"
	"encoding/json"
	"os/exec"
	"strconv"
	"strings"
)

var exposedPortPreference = []int{80, 8080, 3000, 8000, 5000}

// DefaultListenPort returns a heuristic listen port for a detected stack.
func DefaultListenPort(detected string) int {
	switch strings.ToLower(strings.TrimSpace(detected)) {
	case "nodejs":
		return 3000
	case "python":
		return 8000
	case "go", "dotnet":
		return 8080
	case "dockerfile":
		return 80
	default:
		return 8080
	}
}

// InspectImageListenPort reads EXPOSE from a built local image tag.
func InspectImageListenPort(ctx context.Context, image string) int {
	image = strings.TrimSpace(image)
	if image == "" {
		return 0
	}
	cmd := exec.CommandContext(ctx, "docker", "image", "inspect", image, "--format", "{{json .Config.ExposedPorts}}")
	out, err := cmd.Output()
	if err != nil {
		return 0
	}
	var ports map[string]struct{}
	if json.Unmarshal(out, &ports) != nil || len(ports) == 0 {
		return 0
	}
	found := map[int]bool{}
	var ordered []int
	for key := range ports {
		p, _, ok := strings.Cut(key, "/")
		if !ok {
			continue
		}
		n, err := strconv.Atoi(p)
		if err != nil || n <= 0 || found[n] {
			continue
		}
		found[n] = true
		ordered = append(ordered, n)
	}
	for _, pref := range exposedPortPreference {
		if found[pref] {
			return pref
		}
	}
	if len(ordered) > 0 {
		return ordered[0]
	}
	return 0
}

// ResolveListenPort picks the container port for Traefik.
// Returns port and a short source label for logs (config, image, stack).
func ResolveListenPort(runtimeJSON []byte, detected, image string, ctx context.Context) (int, string) {
	cfg := parseRuntimeJSON(runtimeJSON)
	if !cfg.ListenPortAuto && cfg.ListenPort > 0 {
		return cfg.ListenPort, "config"
	}
	if p := InspectImageListenPort(ctx, image); p > 0 {
		return p, "image"
	}
	if !cfg.ListenPortAuto && cfg.ListenPort > 0 {
		return cfg.ListenPort, "config-default"
	}
	if p := DefaultListenPort(detected); p > 0 {
		return p, "stack"
	}
	return 8080, "fallback"
}

type runtimeCfg struct {
	ListenPort     int
	ListenPortAuto bool
}

func parseRuntimeJSON(raw []byte) runtimeCfg {
	var m map[string]any
	if json.Unmarshal(raw, &m) != nil {
		return runtimeCfg{ListenPortAuto: true}
	}
	var c runtimeCfg
	if v, ok := m["listen_port"].(float64); ok && int(v) > 0 {
		c.ListenPort = int(v)
	}
	if v, ok := m["port"].(float64); ok && int(v) > 0 && c.ListenPort == 0 {
		c.ListenPort = int(v)
	}
	if v, ok := m["listen_port_auto"].(bool); ok {
		c.ListenPortAuto = v
	} else {
		c.ListenPortAuto = true
	}
	return c
}

// RuntimeConfigJSON builds runtime_config bytes after auto-detecting listen port.
func RuntimeConfigJSON(listenPort int, detectedStack string) json.RawMessage {
	b, _ := json.Marshal(map[string]any{
		"listen_port":      listenPort,
		"listen_port_auto": true,
		"detected_stack":   detectedStack,
		"replicas":         1,
	})
	return b
}
