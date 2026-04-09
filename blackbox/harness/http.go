package harness

import (
	"fmt"
	"strconv"
	"strings"
)

// HTTPGetViaPOP makes an HTTPS GET request through a POP's TLS listener.
// It uses curl --resolve to route the hostname to 127.0.0.1 on the given port.
func HTTPGetViaPOP(m *Miren, popListenPort, hostname, path string) (statusCode int, body string, err error) {
	r := m.RunCmd("curl", "-sk", "-w", "\n%{http_code}",
		"--resolve", fmt.Sprintf("%s:%s:127.0.0.1", hostname, popListenPort),
		"--max-time", "10",
		fmt.Sprintf("https://%s:%s%s", hostname, popListenPort, path))

	if !r.Success() {
		return 0, "", fmt.Errorf("curl failed (exit %d): %s", r.ExitCode, strings.TrimSpace(r.Stderr))
	}

	lines := strings.Split(strings.TrimRight(r.Stdout, "\n"), "\n")
	if len(lines) < 1 {
		return 0, "", fmt.Errorf("unexpected curl output: %q", r.Stdout)
	}

	statusStr := lines[len(lines)-1]
	code, parseErr := strconv.Atoi(strings.TrimSpace(statusStr))
	if parseErr != nil {
		return 0, "", fmt.Errorf("failed to parse status code %q: %v", statusStr, parseErr)
	}

	bodyStr := strings.Join(lines[:len(lines)-1], "\n")
	return code, bodyStr, nil
}

// HTTPGet makes an HTTP GET request from inside the dev container via curl.
// The host header is set to the given hostname so ingress routing works,
// while the actual request goes to localhost:443 over HTTPS (with -k to
// skip certificate verification). Port 80 redirects to HTTPS, so we
// connect directly to avoid redirect resolution issues.
func HTTPGet(m *Miren, hostname, path string) (statusCode int, body string, err error) {
	r := m.RunCmd("curl", "-sk", "-w", "\n%{http_code}",
		"-H", fmt.Sprintf("Host: %s", hostname),
		"--max-time", "10",
		fmt.Sprintf("https://localhost:443%s", path))

	if !r.Success() {
		return 0, "", fmt.Errorf("curl failed (exit %d): %s", r.ExitCode, strings.TrimSpace(r.Stderr))
	}

	// Output format: body\nstatus_code
	lines := strings.Split(strings.TrimRight(r.Stdout, "\n"), "\n")
	if len(lines) < 1 {
		return 0, "", fmt.Errorf("unexpected curl output: %q", r.Stdout)
	}

	statusStr := lines[len(lines)-1]
	code, parseErr := strconv.Atoi(strings.TrimSpace(statusStr))
	if parseErr != nil {
		return 0, "", fmt.Errorf("failed to parse status code %q: %v", statusStr, parseErr)
	}

	bodyStr := strings.Join(lines[:len(lines)-1], "\n")
	return code, bodyStr, nil
}
