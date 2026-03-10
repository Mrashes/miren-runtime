package harness

import (
	"testing"
	"time"
)

// Poll repeatedly calls condition until it returns true or the timeout expires.
// On timeout, it fails the test with the last message from the condition.
func Poll(t *testing.T, description string, timeout, interval time.Duration, condition func() (done bool, msg string)) {
	t.Helper()

	if timeout <= 0 {
		t.Fatalf("Poll: timeout must be positive, got %s", timeout)
	}
	if interval <= 0 {
		t.Fatalf("Poll: interval must be positive, got %s", interval)
	}

	deadline := time.Now().Add(timeout)
	var lastMsg string

	for time.Now().Before(deadline) {
		done, msg := condition()
		if done {
			return
		}
		lastMsg = msg
		time.Sleep(interval)
	}

	t.Fatalf("timed out waiting for %s (after %s): %s", description, timeout, lastMsg)
}
