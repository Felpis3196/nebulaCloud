package application

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestComposeImageRef_preservesRegistryPort(t *testing.T) {
	id := uuid.MustParse("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee")
	ref := composeImageRef("registry.nebula.localhost:5000", "Personal", "app", "web", id)
	if strings.Contains(ref, "localhost-5000") || strings.Contains(ref, ".localhost-5000") {
		t.Fatalf("port colon mangled: %q", ref)
	}
	if !strings.HasPrefix(ref, "registry.nebula.localhost:5000/") {
		t.Fatalf("unexpected ref: %q", ref)
	}
}

func TestComposeImageRef_localhostRegistry(t *testing.T) {
	id := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	ref := composeImageRef("localhost:5000", "org-1", "my-app", "web", id)
	wantPrefix := "localhost:5000/org-1/my-app/web:"
	if !strings.HasPrefix(ref, wantPrefix) {
		t.Fatalf("got %q want prefix %q", ref, wantPrefix)
	}
}
