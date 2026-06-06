package buildworker

import (
	"context"
	"encoding/json"
	"testing"
)

func TestResolveListenPort_respectsExplicit8080(t *testing.T) {
	rt, _ := json.Marshal(map[string]any{
		"listen_port":      8080,
		"listen_port_auto": false,
	})
	port, src := ResolveListenPort(rt, "nodejs", "nonexistent:tag", context.Background())
	if port != 8080 || src != "config" {
		t.Fatalf("got port=%d src=%s want 8080 config", port, src)
	}
}

func TestResolveListenPort_autoUsesStack(t *testing.T) {
	rt, _ := json.Marshal(map[string]any{
		"listen_port":      8080,
		"listen_port_auto": true,
	})
	port, _ := ResolveListenPort(rt, "nodejs", "nonexistent:tag", context.Background())
	if port != 3000 {
		t.Fatalf("got %d want 3000 from nodejs default", port)
	}
}
