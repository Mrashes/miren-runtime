package server

import (
	"context"
	"errors"
	"log/slog"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"miren.dev/runtime/api/storage/storage_v1alpha"
	"miren.dev/runtime/pkg/entity"
	"miren.dev/runtime/pkg/entity/testutils"
)

func TestServerRecordVolumeReconcileSuccess(t *testing.T) {
	server := &Server{
		log:      slog.Default(),
		dataPath: t.TempDir(),
	}

	// Record a successful reconciliation
	before := time.Now()
	server.recordVolumeReconcile(100*time.Millisecond, nil)

	assert.Equal(t, int64(1), server.volumeReconcileCount)
	assert.Equal(t, int64(0), server.volumeErrorCount)
	assert.Equal(t, 100*time.Millisecond, server.lastVolumeDuration)
	assert.True(t, server.lastVolumeReconcile.After(before) || server.lastVolumeReconcile.Equal(before))
	assert.Empty(t, server.lastError)
}

func TestServerRecordVolumeReconcileError(t *testing.T) {
	server := &Server{
		log:      slog.Default(),
		dataPath: t.TempDir(),
	}

	// Record a failed reconciliation
	testErr := errors.New("volume reconcile failed")
	server.recordVolumeReconcile(200*time.Millisecond, testErr)

	assert.Equal(t, int64(1), server.volumeReconcileCount)
	assert.Equal(t, int64(1), server.volumeErrorCount)
	assert.Equal(t, 200*time.Millisecond, server.lastVolumeDuration)
	assert.Equal(t, "volume reconcile failed", server.lastError)
}

func TestServerRecordVolumeReconcileMultiple(t *testing.T) {
	server := &Server{
		log:      slog.Default(),
		dataPath: t.TempDir(),
	}

	// Record multiple reconciliations
	server.recordVolumeReconcile(100*time.Millisecond, nil)
	server.recordVolumeReconcile(150*time.Millisecond, nil)
	server.recordVolumeReconcile(200*time.Millisecond, errors.New("error 1"))
	server.recordVolumeReconcile(50*time.Millisecond, nil)
	server.recordVolumeReconcile(75*time.Millisecond, errors.New("error 2"))

	assert.Equal(t, int64(5), server.volumeReconcileCount)
	assert.Equal(t, int64(2), server.volumeErrorCount)
	assert.Equal(t, 75*time.Millisecond, server.lastVolumeDuration) // Last duration
	assert.Equal(t, "error 2", server.lastError)                    // Last error
}

func TestServerRecordMountReconcileSuccess(t *testing.T) {
	server := &Server{
		log:      slog.Default(),
		dataPath: t.TempDir(),
	}

	// Record a successful reconciliation
	before := time.Now()
	server.recordMountReconcile(50*time.Millisecond, nil)

	assert.Equal(t, int64(1), server.mountReconcileCount)
	assert.Equal(t, int64(0), server.mountErrorCount)
	assert.Equal(t, 50*time.Millisecond, server.lastMountDuration)
	assert.True(t, server.lastMountReconcile.After(before) || server.lastMountReconcile.Equal(before))
	assert.Empty(t, server.lastError)
}

func TestServerRecordMountReconcileError(t *testing.T) {
	server := &Server{
		log:      slog.Default(),
		dataPath: t.TempDir(),
	}

	// Record a failed reconciliation
	testErr := errors.New("mount reconcile failed")
	server.recordMountReconcile(300*time.Millisecond, testErr)

	assert.Equal(t, int64(1), server.mountReconcileCount)
	assert.Equal(t, int64(1), server.mountErrorCount)
	assert.Equal(t, 300*time.Millisecond, server.lastMountDuration)
	assert.Equal(t, "mount reconcile failed", server.lastError)
}

func TestServerRecordMountReconcileMultiple(t *testing.T) {
	server := &Server{
		log:      slog.Default(),
		dataPath: t.TempDir(),
	}

	// Record multiple reconciliations
	server.recordMountReconcile(100*time.Millisecond, nil)
	server.recordMountReconcile(150*time.Millisecond, errors.New("error 1"))
	server.recordMountReconcile(200*time.Millisecond, nil)

	assert.Equal(t, int64(3), server.mountReconcileCount)
	assert.Equal(t, int64(1), server.mountErrorCount)
	assert.Equal(t, 200*time.Millisecond, server.lastMountDuration)
}

func TestServerSetLastVolumeReconcile(t *testing.T) {
	server := &Server{
		log:      slog.Default(),
		dataPath: t.TempDir(),
	}

	before := time.Now()
	server.setLastVolumeReconcile()

	assert.True(t, server.lastVolumeReconcile.After(before) || server.lastVolumeReconcile.Equal(before))
	assert.True(t, server.lastVolumeReconcile.Before(time.Now().Add(time.Second)))
}

func TestServerSetLastMountReconcile(t *testing.T) {
	server := &Server{
		log:      slog.Default(),
		dataPath: t.TempDir(),
	}

	before := time.Now()
	server.setLastMountReconcile()

	assert.True(t, server.lastMountReconcile.After(before) || server.lastMountReconcile.Equal(before))
	assert.True(t, server.lastMountReconcile.Before(time.Now().Add(time.Second)))
}

func TestServerSetLastError(t *testing.T) {
	server := &Server{
		log:      slog.Default(),
		dataPath: t.TempDir(),
	}

	server.setLastError("first error")
	assert.Equal(t, "first error", server.lastError)

	server.setLastError("second error")
	assert.Equal(t, "second error", server.lastError)

	server.setLastError("")
	assert.Equal(t, "", server.lastError)
}

func TestServerMetricsConcurrency(t *testing.T) {
	server := &Server{
		log:      slog.Default(),
		dataPath: t.TempDir(),
	}

	// Run concurrent reconcile recordings
	done := make(chan bool)
	iterations := 100

	// Concurrent volume reconciles
	go func() {
		for i := 0; i < iterations; i++ {
			if i%10 == 0 {
				server.recordVolumeReconcile(time.Duration(i)*time.Millisecond, errors.New("error"))
			} else {
				server.recordVolumeReconcile(time.Duration(i)*time.Millisecond, nil)
			}
		}
		done <- true
	}()

	// Concurrent mount reconciles
	go func() {
		for i := 0; i < iterations; i++ {
			if i%5 == 0 {
				server.recordMountReconcile(time.Duration(i)*time.Millisecond, errors.New("error"))
			} else {
				server.recordMountReconcile(time.Duration(i)*time.Millisecond, nil)
			}
		}
		done <- true
	}()

	// Wait for both goroutines
	<-done
	<-done

	// Verify counts are correct
	assert.Equal(t, int64(iterations), server.volumeReconcileCount)
	assert.Equal(t, int64(iterations), server.mountReconcileCount)
	assert.Equal(t, int64(iterations/10), server.volumeErrorCount) // Every 10th is an error
	assert.Equal(t, int64(iterations/5), server.mountErrorCount)   // Every 5th is an error
}

func TestAddrFileName(t *testing.T) {
	assert.Equal(t, "lsvd-server.addr", AddrFileName)
}

func TestReadyFileName(t *testing.T) {
	assert.Equal(t, "lsvd-server.ready", ReadyFileName)
}

func TestNewServer(t *testing.T) {
	log := slog.Default()
	dataPath := t.TempDir()
	nodeId := "test-node"
	entityServerAddr := "localhost:9000"
	debugAddr := "localhost:0"

	srv, err := NewServer(log, dataPath, nodeId, entityServerAddr, debugAddr)

	assert.NoError(t, err)
	assert.NotNil(t, srv)
	assert.Equal(t, dataPath, srv.dataPath)
	assert.Equal(t, nodeId, srv.nodeId)
	assert.Equal(t, entityServerAddr, srv.entityServerAddr)
	assert.Equal(t, debugAddr, srv.debugAddr)
}

func TestServerReconcileWithSystem(t *testing.T) {
	ctx := t.Context()
	log := slog.Default()
	dataPath := t.TempDir()
	nodeId := "test-node"
	state := NewState()

	// Add a volume to state
	state.SetVolume("vol-test", &VolumeState{
		EntityId: "vol-test",
		VolumeId: "test-vol-id",
		DiskPath: dataPath + "/volumes/test-vol-id",
	})

	// Create server with mock controllers
	srv := &Server{
		log:      log,
		dataPath: dataPath,
		nodeId:   nodeId,
		state:    state,
	}

	// Create mock volume ops that reports path exists
	mockVolOps := newMockVolumeOps()
	mockVolOps.existingPaths[dataPath+"/volumes/test-vol-id"] = true

	// Create controllers with mock ops
	srv.volumeController = NewVolumeController(log, dataPath, nodeId, nil, state, mockVolOps)
	srv.mountController = NewMountController(log, dataPath, nodeId, nil, state, newMockMountOps())

	// Run reconcileWithSystem
	err := srv.reconcileWithSystem(ctx)

	assert.NoError(t, err)
}

func TestServerReconcileWithEntities(t *testing.T) {
	ctx := t.Context()
	log := slog.Default()
	dataPath := t.TempDir()
	nodeId := "test-node"
	state := NewState()

	// Create server
	srv := &Server{
		log:      log,
		dataPath: dataPath,
		nodeId:   nodeId,
		state:    state,
	}

	// Create in-memory entity server
	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	// Create mock ops
	mockVolOps := newMockVolumeOps()
	mockMntOps := newMockMountOps()

	// Create controllers with mock ops and entity access client
	srv.volumeController = NewVolumeController(log, dataPath, nodeId, es.EAC, state, mockVolOps)
	srv.mountController = NewMountController(log, dataPath, nodeId, es.EAC, state, mockMntOps)

	// Create a volume entity
	vol := &storage_v1alpha.LsvdVolume{
		ID:           "lsvd_volume/vol-ent",
		NodeId:       entity.Id(nodeId),
		SizeGb:       10,
		Filesystem:   "ext4",
		DesiredState: storage_v1alpha.VOL_PRESENT,
		ActualState:  storage_v1alpha.VOL_PENDING,
	}
	createLsvdVolumeEntity(ctx, t, es, vol)

	// Run reconcileWithEntities
	err := srv.reconcileWithEntities(ctx)

	assert.NoError(t, err)

	// Verify volume was created
	assert.Len(t, mockVolOps.createdDirs, 1)
	assert.Len(t, mockVolOps.initedVolumes, 1)
}

func TestServerWatchVolumes(t *testing.T) {
	ctx := t.Context()
	log := slog.Default()
	dataPath := t.TempDir()
	nodeId := "test-node"
	state := NewState()

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	mockVolOps := newMockVolumeOps()

	// Create server with volume controller
	srv := &Server{
		log:      log,
		dataPath: dataPath,
		nodeId:   nodeId,
		state:    state,
	}
	srv.volumeController = NewVolumeController(log, dataPath, nodeId, es.EAC, state, mockVolOps)

	// Create a volume entity to reconcile
	vol := &storage_v1alpha.LsvdVolume{
		ID:           "lsvd_volume/vol-watch",
		NodeId:       entity.Id(nodeId),
		SizeGb:       10,
		Filesystem:   "ext4",
		DesiredState: storage_v1alpha.VOL_PRESENT,
		ActualState:  storage_v1alpha.VOL_PENDING,
	}
	createLsvdVolumeEntity(ctx, t, es, vol)

	// Start watchVolumes with a short-lived context
	runCtx, cancel := context.WithCancel(ctx)

	done := make(chan error, 1)
	go func() {
		done <- srv.watchVolumes(runCtx)
	}()

	// Wait for initial reconciliation
	time.Sleep(100 * time.Millisecond)

	// Cancel and wait for exit
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("watchVolumes did not exit after context cancellation")
	}

	// Verify volume was reconciled
	assert.Len(t, mockVolOps.createdDirs, 1)
}

func TestServerWatchMounts(t *testing.T) {
	ctx := t.Context()
	log := slog.Default()
	dataPath := t.TempDir()
	nodeId := "test-node"
	state := NewState()

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	// Pre-populate volume state (mount requires a volume)
	state.SetVolume("lsvd_volume/vol-watch", &VolumeState{
		EntityId:   "lsvd_volume/vol-watch",
		VolumeId:   "watch-vol-id",
		DiskPath:   "/data/volumes/watch-vol-id",
		SizeBytes:  10 * 1024 * 1024 * 1024,
		Filesystem: "ext4",
	})

	mockMntOps := newMockMountOps()

	// Create server with mount controller
	srv := &Server{
		log:      log,
		dataPath: dataPath,
		nodeId:   nodeId,
		state:    state,
	}
	srv.mountController = NewMountController(log, dataPath, nodeId, es.EAC, state, mockMntOps)

	// Create a mount entity to reconcile
	mount := &storage_v1alpha.LsvdMount{
		ID:           "lsvd_mount/mnt-watch",
		NodeId:       entity.Id(nodeId),
		VolumeId:     "lsvd_volume/vol-watch",
		MountPath:    "/mnt/watch",
		DesiredState: storage_v1alpha.MNT_WANT_MOUNTED,
		ActualState:  storage_v1alpha.MNT_PENDING,
	}
	createLsvdMountEntity(ctx, t, es, mount)

	// Start watchMounts with a short-lived context
	runCtx, cancel := context.WithCancel(ctx)

	done := make(chan error, 1)
	go func() {
		done <- srv.watchMounts(runCtx)
	}()

	// Wait for initial reconciliation
	time.Sleep(100 * time.Millisecond)

	// Cancel and wait for exit
	cancel()

	select {
	case err := <-done:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("watchMounts did not exit after context cancellation")
	}

	// Verify mount was reconciled
	assert.Len(t, mockMntOps.nbdLoopbackCalls, 1)
}
