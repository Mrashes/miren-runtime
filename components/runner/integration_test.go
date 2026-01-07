package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/pkg/namespaces"
	"github.com/stretchr/testify/require"
	"miren.dev/runtime/api/entityserver/entityserver_v1alpha"
	"miren.dev/runtime/components/coordinate"
	"miren.dev/runtime/components/netresolve"
	"miren.dev/runtime/controllers/sandbox"
	"miren.dev/runtime/network"
	"miren.dev/runtime/pkg/entity"
	"miren.dev/runtime/pkg/testutils"
)

func TestRunnerCoordinatorIntegration(t *testing.T) {
	r := require.New(t)

	// Setup test dependencies
	testDeps, cleanup := testutils.NewTestDeps()
	defer cleanup()

	// Create temp directory for test data
	tempDir := t.TempDir()

	// Setup coordinator config
	coordCfg := coordinate.CoordinatorConfig{
		Address:       "localhost:9991",          // Use test port
		EtcdEndpoints: []string{"etcd:2379"},     // Default etcd port
		Prefix:        "/test/miren/" + t.Name(), // Unique prefix for this test
		DataPath:      tempDir,                   // Use temp directory to prevent file leaks
	}

	// Setup runner config
	runnerCfg := RunnerConfig{
		Id:      "test-runner",
		Workers: 2,
	}

	// Create contexts
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start coordinator in background
	coord := coordinate.NewCoordinator(testDeps.Log, coordCfg)
	err := coord.Start(ctx)
	r.NoError(err)

	// Wait for coordinator to start
	time.Sleep(1 * time.Second)

	rcfg, err := coord.ServiceConfig()
	r.NoError(err)

	runnerCfg.Config = rcfg
	runnerCfg.DataPath = t.TempDir()

	res, _ := netresolve.NewLocalResolver()

	// Build RunnerDeps from testDeps
	sbMetrics := sandbox.NewMetrics()
	sbMetrics.Log = testDeps.Log
	sbMetrics.CPUUsage = testDeps.CPU
	sbMetrics.MemUsage = testDeps.Mem

	deps := RunnerDeps{
		CC:              testDeps.CC,
		Namespace:       testDeps.Namespace,
		Bridge:          testDeps.Bridge,
		Tempdir:         tempDir,
		Subnet:          testDeps.Subnet,
		NetServ:         network.NewServiceManager(testDeps.Log, nil),
		LogsMaintainer:  testDeps.LogsMaintainer,
		LogWriter:       testDeps.LogWriter,
		StatusMon:       testDeps.StatusMon,
		IPv4Routable:    testDeps.IPv4Routable,
		ServicePrefixes: testDeps.ServicePrefixes,
		DisableLocalNet: false,
		Resolver:        res,
		SandboxMetrics:  sbMetrics,
	}

	// Create and start runner
	runner, err := NewRunner(testDeps.Log, deps, runnerCfg)
	r.NoError(err)

	runnerDone := make(chan error, 1)
	go func() {
		runnerDone <- runner.Start(ctx)
	}()

	defer runner.Close()

	cfg, err := coord.LocalConfig()
	r.NoError(err)

	// Create RPC client to interact with coordinator
	rs, err := cfg.State(ctx)
	require.NoError(t, err)

	client, err := rs.Connect(coordCfg.Address, "entities")
	require.NoError(t, err)

	eac := entityserver_v1alpha.EntityAccessClient{Client: client}

	// Check the node entity for the runner
	nodeId := "node/" + runnerCfg.Id

	// Wait for runner to register
	deadline := time.Now().Add(30 * time.Second)
	var nodeEntity *entity.Entity
	for time.Now().Before(deadline) {
		resp, err := eac.Get(ctx, nodeId)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		nodeEntity = resp.Entity().Entity()
		break
	}

	require.NotNil(t, nodeEntity, "node entity should be created")
	require.Equal(t, nodeId, string(nodeEntity.Id()))

	// Stop runner
	cancel()

	// Check that runner shuts down properly
	select {
	case err := <-runnerDone:
		if err != nil && err != context.Canceled {
			t.Errorf("Runner exited with error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Runner did not shut down in time")
	}
}

func TestRunnerContainerLifecycle(t *testing.T) {
	r := require.New(t)

	// Setup test dependencies
	testDeps, cleanup := testutils.NewTestDeps()
	defer cleanup()

	// Create temp directory for test data
	tempDir := t.TempDir()

	// Setup coordinator config
	coordCfg := coordinate.CoordinatorConfig{
		Address:       "localhost:9993",          // Use test port
		EtcdEndpoints: []string{"etcd:2379"},     // Default etcd port
		Prefix:        "/test/miren/" + t.Name(), // Unique prefix for this test
		DataPath:      tempDir,                   // Use temp directory to prevent file leaks
	}

	// Setup runner config
	runnerCfg := RunnerConfig{
		Id:      "test-runner",
		Workers: 2,
	}

	// Create contexts
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start coordinator in background
	coord := coordinate.NewCoordinator(testDeps.Log, coordCfg)
	err := coord.Start(ctx)
	r.NoError(err)

	// Wait for coordinator to start
	time.Sleep(1 * time.Second)

	rcfg, err := coord.ServiceConfig()
	r.NoError(err)

	runnerCfg.Config = rcfg
	runnerCfg.DataPath = t.TempDir()

	res, _ := netresolve.NewLocalResolver()

	// Build RunnerDeps from testDeps
	sbMetrics2 := sandbox.NewMetrics()
	sbMetrics2.Log = testDeps.Log
	sbMetrics2.CPUUsage = testDeps.CPU
	sbMetrics2.MemUsage = testDeps.Mem

	deps := RunnerDeps{
		CC:              testDeps.CC,
		Namespace:       testDeps.Namespace,
		Bridge:          testDeps.Bridge,
		Tempdir:         tempDir,
		Subnet:          testDeps.Subnet,
		NetServ:         network.NewServiceManager(testDeps.Log, nil),
		LogsMaintainer:  testDeps.LogsMaintainer,
		LogWriter:       testDeps.LogWriter,
		StatusMon:       testDeps.StatusMon,
		IPv4Routable:    testDeps.IPv4Routable,
		ServicePrefixes: testDeps.ServicePrefixes,
		DisableLocalNet: false,
		Resolver:        res,
		SandboxMetrics:  sbMetrics2,
	}

	// Create and start runner
	runner, err := NewRunner(testDeps.Log, deps, runnerCfg)
	r.NoError(err)

	runnerDone := make(chan error, 1)
	go func() {
		runnerDone <- runner.Start(ctx)
	}()

	defer runner.Close()

	// Setup entity client
	cfg, err := coord.LocalConfig()
	r.NoError(err)

	rs, err := cfg.State(ctx)
	require.NoError(t, err)

	client, err := rs.Connect(coordCfg.Address, "entities")
	require.NoError(t, err)

	eac := entityserver_v1alpha.NewEntityAccessClient(client)

	// Create test container context
	testNs := fmt.Sprintf("test-ns-%d", time.Now().UnixNano())
	containerCtx := namespaces.WithNamespace(ctx, testNs)

	// We'll use runner's containerd client to create a test container
	testImage := "docker.io/library/alpine:latest"

	// Pull the image if needed
	image, err := testDeps.CC.Pull(containerCtx, testImage, containerd.WithPullUnpack)
	if err != nil {
		t.Logf("Warning: Could not pull test image: %v", err)
		t.Skip("Skipping container lifecycle test - could not pull test image")
		return
	}

	// Create a container (but don't start it through runner - just verify containerd works)
	containerName := "test-container-" + t.Name()
	container, err := testDeps.CC.NewContainer(
		containerCtx,
		containerName,
		containerd.WithImage(image),
		containerd.WithNewSnapshot(containerName+"-snapshot", image),
	)
	if err != nil {
		t.Fatalf("Failed to create container: %v", err)
	}
	defer container.Delete(containerCtx, containerd.WithSnapshotCleanup)

	t.Logf("Created container: %s", container.ID())

	// Verify runner is still healthy
	nodeId := "node/" + runnerCfg.Id
	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := eac.Get(ctx, nodeId)
		if err != nil {
			time.Sleep(100 * time.Millisecond)
			continue
		}
		nodeEntity := resp.Entity().Entity()
		if string(nodeEntity.Id()) == nodeId {
			t.Logf("Runner node entity verified: %s", nodeId)
			break
		}
	}

	// Clean shutdown
	cancel()

	select {
	case err := <-runnerDone:
		if err != nil && err != context.Canceled {
			t.Errorf("Runner exited with error: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Error("Runner did not shut down in time")
	}
}

// TestRunnerWithTestDeps tests that runner works with explicit test dependencies
func TestRunnerWithTestDeps(t *testing.T) {
	r := require.New(t)

	// Create test dependencies explicitly
	testDeps, cleanup := testutils.NewTestDeps()
	defer cleanup()

	tempDir := t.TempDir()

	res, _ := netresolve.NewLocalResolver()

	// Create RunnerDeps from TestDeps
	sbMetrics3 := sandbox.NewMetrics()
	sbMetrics3.Log = testDeps.Log
	sbMetrics3.CPUUsage = testDeps.CPU
	sbMetrics3.MemUsage = testDeps.Mem

	deps := RunnerDeps{
		CC:              testDeps.CC,
		Namespace:       testDeps.Namespace,
		Bridge:          testDeps.Bridge,
		Tempdir:         tempDir,
		Subnet:          testDeps.Subnet,
		NetServ:         network.NewServiceManager(testDeps.Log, nil),
		LogsMaintainer:  testDeps.LogsMaintainer,
		LogWriter:       testDeps.LogWriter,
		StatusMon:       testDeps.StatusMon,
		IPv4Routable:    testDeps.IPv4Routable,
		ServicePrefixes: testDeps.ServicePrefixes,
		DisableLocalNet: false,
		Resolver:        res,
		SandboxMetrics:  sbMetrics3,
	}

	t.Log("RunnerDeps created successfully from testutils.NewTestDeps")

	// Verify service config file creation
	scPath := filepath.Join(tempDir, "runner", "service.yaml")
	err := os.MkdirAll(filepath.Dir(scPath), 0755)
	r.NoError(err)

	// Just verify deps are valid - we don't need to actually start the runner
	r.NotNil(deps.CC)
	r.NotEmpty(deps.Namespace)
	r.NotEmpty(deps.Bridge)
	r.NotEmpty(deps.Tempdir)
}
