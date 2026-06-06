package buildworker

import (
	"fmt"
	"testing"
)

func TestParseSymrefHEAD(t *testing.T) {
	out := `ref: refs/heads/master	HEAD
a1b2c3d4	HEAD
`
	if got := parseSymrefHEAD(out); got != "master" {
		t.Fatalf("got %q want master", got)
	}
}

func TestIsRemoteBranchMissing(t *testing.T) {
	err := fmt.Errorf("git clone: exit status 128 — fatal: Remote branch main not found")
	if !isRemoteBranchMissing(err) {
		t.Fatal("expected branch missing")
	}
}
