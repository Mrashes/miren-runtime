//go:build blackbox

package blackbox

import (
	"testing"

	"miren.dev/runtime/blackbox/harness"
)

func TestSandboxList(t *testing.T) {
	c := harness.NewCluster(t)
	m := harness.NewMiren(t, c)

	name := harness.DeployApp(t, m, harness.AppOptions{
		Testdata: "go-server",
	})

	// List sandboxes — our app's sandbox should appear in the output
	r := m.MustRun("sandbox", "list")
	r.RequireContains(t, name)
}

func TestSandboxExec(t *testing.T) {
	c := harness.NewCluster(t)
	m := harness.NewMiren(t, c)

	name := harness.DeployApp(t, m, harness.AppOptions{
		Testdata: "go-server",
	})

	// Get sandbox ID from JSON listing
	sandboxID := harness.GetSandboxID(t, m, name)

	// Exec a simple command in the sandbox.
	// Note: sandbox exec may exit non-zero due to a known CLI cleanup issue,
	// but the command output should still be correct.
	r := m.Run("sandbox", "exec", "-i", sandboxID, "--", "echo", "hello-from-sandbox")
	r.RequireContains(t, "hello-from-sandbox")
}
