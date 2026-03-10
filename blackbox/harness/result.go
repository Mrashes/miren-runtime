package harness

import (
	"strings"
	"testing"
)

// Result captures the output of a CLI invocation.
type Result struct {
	ExitCode int
	Stdout   string
	Stderr   string
}

// Success returns true if the command exited with code 0.
func (r *Result) Success() bool {
	return r.ExitCode == 0
}

// OutputContains checks whether stdout or stderr contains the given substring.
func (r *Result) OutputContains(s string) bool {
	return strings.Contains(r.Stdout, s) || strings.Contains(r.Stderr, s)
}

// RequireSuccess fails the test if the command did not exit with code 0.
func (r *Result) RequireSuccess(t *testing.T) {
	t.Helper()
	if r.ExitCode != 0 {
		t.Fatalf("expected exit code 0, got %d\nstdout: %s\nstderr: %s", r.ExitCode, r.Stdout, r.Stderr)
	}
}

// RequireContains fails the test if neither stdout nor stderr contains substr.
func (r *Result) RequireContains(t *testing.T, substr string) {
	t.Helper()
	if !r.OutputContains(substr) {
		t.Fatalf("expected output to contain %q\nstdout: %s\nstderr: %s", substr, r.Stdout, r.Stderr)
	}
}

// RequireExitCode fails the test if the exit code doesn't match.
func (r *Result) RequireExitCode(t *testing.T, code int) {
	t.Helper()
	if r.ExitCode != code {
		t.Fatalf("expected exit code %d, got %d\nstdout: %s\nstderr: %s", code, r.ExitCode, r.Stdout, r.Stderr)
	}
}
