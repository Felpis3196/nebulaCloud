package buildworker

import (
	"strings"
	"testing"
)

func TestExcerpt_shortInputUnchanged(t *testing.T) {
	in := "short error"
	if got := Excerpt(in, DefaultExcerptLimit); got != in {
		t.Fatalf("got %q want %q", got, in)
	}
}

func TestExcerpt_preservesTailError(t *testing.T) {
	head := strings.Repeat("pulling layer ", 500)
	tail := "ERROR: process \"/bin/sh -c pip install\" did not complete successfully: exit code: 1"
	in := head + tail
	got := Excerpt(in, 200)
	if !strings.Contains(got, tail) {
		t.Fatalf("tail error missing from excerpt:\n%s", got)
	}
	if !strings.Contains(got, "chars omitted") {
		t.Fatalf("omitted marker missing:\n%s", got)
	}
	if strings.Contains(got, strings.Repeat("pulling layer ", 200)) {
		t.Fatal("excerpt should not keep entire head noise")
	}
}

func TestExcerpt_runeSafe(t *testing.T) {
	in := strings.Repeat("é", 100) + "TAIL_MARKER"
	got := Excerpt(in, 30)
	if !strings.Contains(got, "TAIL_MARKER") {
		t.Fatalf("tail marker missing: %q", got)
	}
	if strings.Contains(got, "\uFFFD") {
		t.Fatalf("replacement char in excerpt: %q", got)
	}
}

func TestTrimOut_delegatesToExcerpt(t *testing.T) {
	in := []byte(strings.Repeat("x", 100) + "REAL_ERROR")
	got := trimOut(in)
	if !strings.Contains(got, "REAL_ERROR") {
		t.Fatalf("trimOut should preserve tail: %q", got)
	}
}
