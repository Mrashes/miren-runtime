//go:build blackbox

package blackbox

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"miren.dev/runtime/blackbox/harness"
)

// TestPOP exercises the global router end-to-end. The expensive cloud+POP
// setup (binary builds, migrations, server restart) happens once in the
// parent test; each subtest only deploys its own app, binds a hostname, and
// sends traffic.
func TestPOP(t *testing.T) {
	c := harness.NewCluster(t)
	m := harness.NewMiren(t, c)
	env := harness.NewCloudEnv(t, m)

	// TestHTTPForwarding: full HTTP path through POP to go-server app.
	//
	//	client → POP TLS → cloud coordination → QUIC H3 → cluster → app → response
	t.Run("HTTPForwarding", func(t *testing.T) {
		appName := harness.DeployApp(t, m, harness.AppOptions{Testdata: "go-server"})
		hostname := appName + ".test.pop"
		m.MustRun("route", "set", hostname, appName)
		env.BindAppHostname(t, hostname)

		// Assert on the response body to confirm the request actually
		// traversed the full tunnel (not just a 200 from a proxy error page).
		harness.Poll(t, "HTTP via POP", 90*time.Second, 3*time.Second, func() (bool, string) {
			code, body, err := harness.HTTPGetViaPOP(m, env.PopListenPort, hostname, "/")
			if err != nil {
				return false, fmt.Sprintf("curl error: %v", err)
			}
			if code != 200 {
				return false, fmt.Sprintf("status %d, body: %s", code, body)
			}
			if !strings.Contains(body, "Hello from Go!") {
				return false, fmt.Sprintf("unexpected body (not from go-server): %q", body)
			}
			return true, ""
		})
	})

	// HTTPHealthCheckWebSocketApp: verifies a WS-capable app's HTTP /health
	// endpoint is reachable via POP. This only exercises the HTTP path
	// through the POP (not the WS tunnel frame path). Kept separate from
	// HTTPForwarding because it confirms POP routing works even for apps
	// that primarily serve WebSockets.
	// TODO: exercise the actual WS tunnel frame path once we have a WS
	// client available in the test environment.
	t.Run("HTTPHealthCheckWebSocketApp", func(t *testing.T) {
		appName := harness.DeployApp(t, m, harness.AppOptions{Testdata: "websocket-echo"})
		hostname := appName + ".test.pop"
		m.MustRun("route", "set", hostname, appName)
		env.BindAppHostname(t, hostname)

		harness.Poll(t, "health via POP", 90*time.Second, 3*time.Second, func() (bool, string) {
			code, _, err := harness.HTTPGetViaPOP(m, env.PopListenPort, hostname, "/health")
			if err != nil {
				return false, fmt.Sprintf("health check error: %v", err)
			}
			if code != 200 {
				return false, fmt.Sprintf("health check status %d", code)
			}
			return true, ""
		})
	})
}
