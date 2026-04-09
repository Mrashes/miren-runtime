package harness

import (
	"bytes"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Miren wraps CLI execution against a target cluster.
type Miren struct {
	t       *testing.T
	cluster *Cluster
}

// NewMiren creates a CLI runner bound to the given cluster.
func NewMiren(t *testing.T, cluster *Cluster) *Miren {
	t.Helper()
	return &Miren{t: t, cluster: cluster}
}

// Run executes a miren CLI command and returns the result.
// In dev mode the command is dispatched through hack/dev-exec.
func (m *Miren) Run(args ...string) *Result {
	m.t.Helper()

	var cmd *exec.Cmd

	switch m.cluster.Mode {
	case ModeDev:
		// hack/dev-exec m <args>
		devExec := filepath.Join(m.cluster.RepoRoot, "hack", "dev-exec")
		execArgs := append([]string{"m"}, args...)
		cmd = exec.Command(devExec, execArgs...)
		cmd.Dir = m.cluster.RepoRoot
	case ModeLocal:
		cmd = exec.Command(m.cluster.MirenBin, args...)
	case ModePeers:
		// iso peers exec coordinator -- m <args>
		execArgs := append([]string{"peers", "exec", "coordinator", "--", "m"}, args...)
		cmd = exec.Command("iso", execArgs...)
		cmd.Dir = m.cluster.RepoRoot
	default:
		m.t.Fatalf("unknown mode: %s", m.cluster.Mode)
		return nil
	}

	// Suppress interactive prompts
	cmd.Env = append(cmd.Environ(), "TERM=dumb")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			m.t.Fatalf("failed to execute command: %v", err)
		}
	}

	r := &Result{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}

	m.t.Logf("miren %s -> exit %d", strings.Join(args, " "), exitCode)
	if r.Stdout != "" {
		m.t.Logf("stdout: %s", r.Stdout)
	}
	if r.Stderr != "" {
		m.t.Logf("stderr: %s", r.Stderr)
	}

	return r
}

// MustRun executes a miren CLI command and fails the test on non-zero exit.
func (m *Miren) MustRun(args ...string) *Result {
	m.t.Helper()
	r := m.Run(args...)
	r.RequireSuccess(m.t)
	return r
}

// RunCmd executes an arbitrary command (not miren CLI) in the dev container.
// In local mode it runs the command directly on the host.
func (m *Miren) RunCmd(args ...string) *Result {
	m.t.Helper()
	if len(args) == 0 {
		m.t.Fatalf("RunCmd requires at least one argument")
		return nil
	}

	var cmd *exec.Cmd

	switch m.cluster.Mode {
	case ModeDev:
		devExec := filepath.Join(m.cluster.RepoRoot, "hack", "dev-exec")
		cmd = exec.Command(devExec, args...)
		cmd.Dir = m.cluster.RepoRoot
	case ModeLocal:
		cmd = exec.Command(args[0], args[1:]...)
	case ModePeers:
		execArgs := append([]string{"peers", "exec", "coordinator", "--"}, args...)
		cmd = exec.Command("iso", execArgs...)
		cmd.Dir = m.cluster.RepoRoot
	default:
		m.t.Fatalf("unknown mode: %s", m.cluster.Mode)
		return nil
	}

	cmd.Env = append(cmd.Environ(), "TERM=dumb")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			m.t.Fatalf("failed to execute command: %v", err)
		}
	}

	r := &Result{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}

	m.t.Logf("cmd %s -> exit %d", strings.Join(args, " "), exitCode)
	if r.Stdout != "" {
		m.t.Logf("stdout: %s", r.Stdout)
	}
	if r.Stderr != "" {
		m.t.Logf("stderr: %s", r.Stderr)
	}

	return r
}

// PeerExec runs a command on a specific iso peer container. This is only
// meaningful in ModePeers and is used for verification tasks like querying
// runner-side state or hitting internal HTTP endpoints.
func (m *Miren) PeerExec(peer string, args ...string) *Result {
	m.t.Helper()
	if m.cluster.Mode != ModePeers {
		m.t.Fatalf("PeerExec requires ModePeers (current: %s)", m.cluster.Mode)
		return nil
	}

	execArgs := append([]string{"peers", "exec", peer, "--"}, args...)
	cmd := exec.Command("iso", execArgs...)
	cmd.Dir = m.cluster.RepoRoot
	cmd.Env = append(cmd.Environ(), "TERM=dumb")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			m.t.Fatalf("PeerExec(%s) failed: %v", peer, err)
		}
	}

	r := &Result{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}

	m.t.Logf("peer(%s) %s -> exit %d", peer, strings.Join(args, " "), exitCode)
	if r.Stdout != "" {
		m.t.Logf("stdout: %s", r.Stdout)
	}
	if r.Stderr != "" {
		m.t.Logf("stderr: %s", r.Stderr)
	}

	return r
}

// ContainerPath translates a host-side path to a container-internal path.
// In dev mode, the repo is mounted at /src inside the iso container.
func (m *Miren) ContainerPath(hostPath string) string {
	if m.cluster.Mode != ModeDev && m.cluster.Mode != ModePeers {
		return hostPath
	}
	rel, err := filepath.Rel(m.cluster.RepoRoot, hostPath)
	if err != nil {
		m.t.Fatalf("path %q is not under repo root %q: %v", hostPath, m.cluster.RepoRoot, err)
	}
	rel = filepath.Clean(rel)
	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		m.t.Fatalf("path %q is outside repo root %q", hostPath, m.cluster.RepoRoot)
	}
	return filepath.Join("/src", rel)
}
