package routing

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestFormatServiceHost_preservesBaseDomainDots(t *testing.T) {
	got := FormatServiceHost("web-1", "my-app", "nebula.localhost")
	want := "web-1.my-app.nebula.localhost"
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestSanitizeSlug(t *testing.T) {
	if SanitizeSlug("Web_Svc") != "web-svc" {
		t.Fatalf("unexpected sanitize")
	}
}

func TestDeployRoute_FileBody_port3000(t *testing.T) {
	id := uuid.MustParse("027258e7-f0fc-470f-b175-1f88e0db3ab9")
	r := NewDeployRoute("/tmp", "web-1", "app-1", "nebula.localhost", id, 3000)
	body := r.RouteFileBody()
	if !strings.Contains(body, "Host(`web-1.app-1.nebula.localhost`)") {
		t.Fatalf("host rule missing: %s", body)
	}
	if !strings.Contains(body, "http://nebula-svc-027258e7f0fc:3000") {
		t.Fatalf("backend url missing: %s", body)
	}
}

func TestWriteUserRoute_roundTrip(t *testing.T) {
	dir := t.TempDir()
	id := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	r := NewDeployRoute(dir, "svc", "proj", "nebula.localhost", id, 8080)
	if err := WriteUserRoute(dir, r); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, r.RouteID+".yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), r.Host) {
		t.Fatalf("file missing host: %s", data)
	}
}

func TestRemoveStaleUserRoutes(t *testing.T) {
	dir := t.TempDir()
	_ = os.WriteFile(filepath.Join(dir, "nebula-oldroute.yml"), []byte("x"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "nebula-keepthis.yml"), []byte("y"), 0o644)
	_ = os.WriteFile(filepath.Join(dir, "dynamic.yml"), []byte("z"), 0o644)
	RemoveStaleUserRoutes(dir, "nebula-keepthis")
	if _, err := os.Stat(filepath.Join(dir, "nebula-oldroute.yml")); !os.IsNotExist(err) {
		t.Fatal("stale route should be removed")
	}
	if _, err := os.Stat(filepath.Join(dir, "nebula-keepthis.yml")); err != nil {
		t.Fatal("keep route should remain")
	}
	if _, err := os.Stat(filepath.Join(dir, "dynamic.yml")); err != nil {
		t.Fatal("non-nebula files should remain")
	}
}
