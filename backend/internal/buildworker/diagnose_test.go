package buildworker

import (
	"strings"
	"testing"
)

func TestDiagnoseFailure_pythonCgiHtmlmin(t *testing.T) {
	raw := `ModuleNotFoundError: No module named 'cgi'
ERROR: Failed to build 'htmlmin' when getting requirements to build wheel`
	hint := DiagnoseFailure(raw)
	if hint == "" {
		t.Fatal("expected hint for cgi/htmlmin failure")
	}
	if !strings.Contains(hint, "python:3.12-alpine") || !strings.Contains(hint, "mkdocs") {
		t.Fatalf("unexpected hint: %q", hint)
	}
}

func TestDiagnoseFailure_gitBranch(t *testing.T) {
	raw := `git clone: exit status 128 — fatal: Remote branch main not found`
	hint := DiagnoseFailure(raw)
	if hint == "" || !strings.Contains(strings.ToLower(hint), "branch") {
		t.Fatalf("unexpected hint: %q", hint)
	}
}

func TestDiagnoseFailure_registryHost(t *testing.T) {
	raw := `docker push: lookup registry.nebula.localhost-5000: no such host`
	hint := DiagnoseFailure(raw)
	if hint == "" {
		t.Fatal("expected registry host hint")
	}
}

func TestDiagnoseFailure_noStack(t *testing.T) {
	hint := DiagnoseFailure(ErrNoStack.Error())
	if hint == "" {
		t.Fatal("expected no-stack hint")
	}
}

func TestDiagnoseFailure_unknownReturnsEmpty(t *testing.T) {
	if got := DiagnoseFailure("something completely unrelated"); got != "" {
		t.Fatalf("got %q want empty", got)
	}
}
