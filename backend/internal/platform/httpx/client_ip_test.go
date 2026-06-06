package httpx

import (
	"net/http/httptest"
	"testing"
)

func TestClientIP_stripsPortFromRemoteAddr(t *testing.T) {
	r := httptest.NewRequest("POST", "/api/v1/auth/login", nil)
	r.RemoteAddr = "172.19.0.1:54321"
	if got := ClientIP(r); got != "172.19.0.1" {
		t.Fatalf("got %q want 172.19.0.1", got)
	}
}

func TestClientIP_ipv6Bracket(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "[::1]:8080"
	if got := ClientIP(r); got != "::1" {
		t.Fatalf("got %q want ::1", got)
	}
}

func TestClientIP_xForwardedFor(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.Header.Set("X-Forwarded-For", "203.0.113.1, 10.0.0.1")
	if got := ClientIP(r); got != "203.0.113.1" {
		t.Fatalf("got %q", got)
	}
}
