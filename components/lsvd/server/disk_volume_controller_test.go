package server

import (
	"context"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"miren.dev/runtime/api/entityserver/entityserver_v1alpha"
	"miren.dev/runtime/api/storage/storage_v1alpha"
	"miren.dev/runtime/pkg/entity"
	"miren.dev/runtime/pkg/entity/testutils"
)

func newTestDiskVolumeController(log *slog.Logger, dataPath, nodeId string, eac *entityserver_v1alpha.EntityAccessClient, state *State, ops DiskVolumeOps) *DiskVolumeController {
	vc := NewDiskVolumeController(log, dataPath, nodeId, state, ops)
	vc.SetEAC(eac)
	return vc
}

func createDiskVolumeEntity(ctx context.Context, t *testing.T, es *testutils.InMemEntityServer, vol *storage_v1alpha.DiskVolume) {
	_, err := es.EAC.Create(ctx, entity.New(
		entity.DBId, vol.ID,
		vol.Encode,
	).Attrs())
	require.NoError(t, err)
}

func TestDiskVolumeControllerReconcileVolumePresent(t *testing.T) {
	ctx := t.Context()
	log := testutils.TestLogger(t)

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	dataPath := t.TempDir()
	nodeId := "test-node-1"
	state := NewState()
	ops := newMockDiskVolumeOps()

	vc := newTestDiskVolumeController(log, dataPath, nodeId, es.EAC, state, ops)

	vol := &storage_v1alpha.DiskVolume{
		ID:           "disk_volume/vol-123",
		NodeId:       entity.Id("node/" + nodeId),
		SizeGb:       10,
		Filesystem:   "ext4",
		DesiredState: storage_v1alpha.DV_PRESENT,
		ActualState:  storage_v1alpha.DV_PENDING,
	}
	createDiskVolumeEntity(ctx, t, es, vol)

	err := vc.ReconcileWithEntities(ctx)
	require.NoError(t, err)

	// Verify volume directory was created
	assert.Len(t, ops.createdDirs, 1)
	expectedVolPath := filepath.Join(dataPath, "volumes", "vol-123")
	assert.Equal(t, expectedVolPath, ops.createdDirs[0])

	// Verify sparse disk image was created
	assert.Len(t, ops.createdImages, 1)
	assert.Equal(t, filepath.Join(expectedVolPath, "disk.img"), ops.createdImages[0].path)
	assert.Equal(t, int64(10*1024*1024*1024), ops.createdImages[0].sizeBytes)

	// Verify state was updated
	volState := state.GetVolume("disk_volume/vol-123")
	require.NotNil(t, volState)
	assert.Equal(t, "disk_volume/vol-123", volState.EntityId)
	assert.Equal(t, "vol-123", volState.VolumeId)
	assert.Equal(t, int64(10*1024*1024*1024), volState.SizeBytes)
	assert.Equal(t, "ext4", volState.Filesystem)

	// Verify entity was updated to READY
	resp, err := es.EAC.Get(ctx, "disk_volume/vol-123")
	require.NoError(t, err)
	var updated storage_v1alpha.DiskVolume
	updated.Decode(resp.Entity().Entity())
	assert.Equal(t, storage_v1alpha.DV_READY, updated.ActualState)
	assert.Equal(t, "vol-123", updated.VolumeId)
	assert.Equal(t, filepath.Join(expectedVolPath, "disk.img"), updated.ImagePath)
}

func TestDiskVolumeControllerReconcileVolumeAbsent(t *testing.T) {
	ctx := t.Context()
	log := testutils.TestLogger(t)

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	dataPath := t.TempDir()
	nodeId := "test-node-1"
	state := NewState()
	ops := newMockDiskVolumeOps()

	// Pre-populate state with existing volume
	state.SetVolume("disk_volume/vol-456", &VolumeState{
		EntityId:   "disk_volume/vol-456",
		VolumeId:   "vol-456",
		DiskPath:   "/data/volumes/vol-456",
		SizeBytes:  5 * 1024 * 1024 * 1024,
		Filesystem: "xfs",
	})
	ops.existingPaths["/data/volumes/vol-456"] = true

	vc := newTestDiskVolumeController(log, dataPath, nodeId, es.EAC, state, ops)

	vol := &storage_v1alpha.DiskVolume{
		ID:           "disk_volume/vol-456",
		NodeId:       entity.Id("node/" + nodeId),
		SizeGb:       5,
		Filesystem:   "xfs",
		DesiredState: storage_v1alpha.DV_ABSENT,
		ActualState:  storage_v1alpha.DV_READY,
	}
	createDiskVolumeEntity(ctx, t, es, vol)

	err := vc.ReconcileWithEntities(ctx)
	require.NoError(t, err)

	// Verify volume directory was removed
	assert.Len(t, ops.removedDirs, 1)
	assert.Equal(t, "/data/volumes/vol-456", ops.removedDirs[0])

	// Verify state was cleaned up
	assert.Nil(t, state.GetVolume("disk_volume/vol-456"))

	// Verify entity was updated to DELETED
	resp, err := es.EAC.Get(ctx, "disk_volume/vol-456")
	require.NoError(t, err)
	var updated storage_v1alpha.DiskVolume
	updated.Decode(resp.Entity().Entity())
	assert.Equal(t, storage_v1alpha.DV_DELETED, updated.ActualState)
}

func TestDiskVolumeControllerReconcileSkipsOtherNodes(t *testing.T) {
	ctx := t.Context()
	log := testutils.TestLogger(t)

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	dataPath := t.TempDir()
	nodeId := "test-node-1"
	state := NewState()
	ops := newMockDiskVolumeOps()

	vc := newTestDiskVolumeController(log, dataPath, nodeId, es.EAC, state, ops)

	vol := &storage_v1alpha.DiskVolume{
		ID:           "disk_volume/vol-other",
		NodeId:       entity.Id("node/other-node"),
		SizeGb:       10,
		DesiredState: storage_v1alpha.DV_PRESENT,
		ActualState:  storage_v1alpha.DV_PENDING,
	}
	createDiskVolumeEntity(ctx, t, es, vol)

	err := vc.ReconcileWithEntities(ctx)
	require.NoError(t, err)

	assert.Empty(t, ops.createdDirs)
	assert.Empty(t, ops.createdImages)
}

func TestDiskVolumeControllerReconcileVolumeAlreadyReady(t *testing.T) {
	ctx := t.Context()
	log := testutils.TestLogger(t)

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	dataPath := t.TempDir()
	nodeId := "test-node-1"
	state := NewState()
	ops := newMockDiskVolumeOps()

	volPath := filepath.Join(dataPath, "volumes", "vol-ready")
	state.SetVolume("disk_volume/vol-ready", &VolumeState{
		EntityId:   "disk_volume/vol-ready",
		VolumeId:   "vol-ready",
		DiskPath:   volPath,
		SizeBytes:  10 * 1024 * 1024 * 1024,
		Filesystem: "ext4",
	})
	ops.existingPaths[volPath] = true

	vc := newTestDiskVolumeController(log, dataPath, nodeId, es.EAC, state, ops)

	vol := &storage_v1alpha.DiskVolume{
		ID:           "disk_volume/vol-ready",
		NodeId:       entity.Id("node/" + nodeId),
		SizeGb:       10,
		Filesystem:   "ext4",
		DesiredState: storage_v1alpha.DV_PRESENT,
		ActualState:  storage_v1alpha.DV_READY,
		VolumeId:     "vol-ready",
	}
	createDiskVolumeEntity(ctx, t, es, vol)

	err := vc.ReconcileWithEntities(ctx)
	require.NoError(t, err)

	assert.Empty(t, ops.createdDirs)
	assert.Empty(t, ops.createdImages)
}

func TestDiskVolumeControllerReconcileVolumeReadyButMissing(t *testing.T) {
	ctx := t.Context()
	log := testutils.TestLogger(t)

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	dataPath := t.TempDir()
	nodeId := "test-node-1"
	state := NewState()
	ops := newMockDiskVolumeOps()

	// State says volume exists, but path does NOT exist on disk
	state.SetVolume("disk_volume/vol-missing", &VolumeState{
		EntityId:   "disk_volume/vol-missing",
		VolumeId:   "vol-missing",
		DiskPath:   "/data/volumes/vol-missing",
		SizeBytes:  10 * 1024 * 1024 * 1024,
		Filesystem: "ext4",
	})
	// Do NOT mark path as existing

	vc := newTestDiskVolumeController(log, dataPath, nodeId, es.EAC, state, ops)

	vol := &storage_v1alpha.DiskVolume{
		ID:           "disk_volume/vol-missing",
		NodeId:       entity.Id("node/" + nodeId),
		SizeGb:       10,
		Filesystem:   "ext4",
		DesiredState: storage_v1alpha.DV_PRESENT,
		ActualState:  storage_v1alpha.DV_READY,
		VolumeId:     "vol-missing",
	}
	createDiskVolumeEntity(ctx, t, es, vol)

	// Reconciliation should fail and set error state
	err := vc.ReconcileWithEntities(ctx)
	require.NoError(t, err) // ReconcileWithEntities logs errors but doesn't return them

	// Verify entity was set to error state
	resp, err := es.EAC.Get(ctx, "disk_volume/vol-missing")
	require.NoError(t, err)
	var updated storage_v1alpha.DiskVolume
	updated.Decode(resp.Entity().Entity())
	assert.Equal(t, storage_v1alpha.DV_ERROR, updated.ActualState)
	assert.Contains(t, updated.ErrorMessage, "volume directory missing")
}

func TestDiskVolumeControllerReconcileVolumeErrorRetry(t *testing.T) {
	ctx := t.Context()
	log := testutils.TestLogger(t)

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	dataPath := t.TempDir()
	nodeId := "test-node-1"
	state := NewState()
	ops := newMockDiskVolumeOps()

	vc := newTestDiskVolumeController(log, dataPath, nodeId, es.EAC, state, ops)

	vol := &storage_v1alpha.DiskVolume{
		ID:           "disk_volume/vol-err",
		NodeId:       entity.Id("node/" + nodeId),
		SizeGb:       10,
		Filesystem:   "ext4",
		DesiredState: storage_v1alpha.DV_PRESENT,
		ActualState:  storage_v1alpha.DV_ERROR,
	}
	createDiskVolumeEntity(ctx, t, es, vol)

	err := vc.ReconcileWithEntities(ctx)
	require.NoError(t, err)

	// Should have recreated the volume
	assert.Len(t, ops.createdDirs, 1)
	assert.Len(t, ops.createdImages, 1)
}

func TestDiskVolumeControllerReconcileCleansUpOrphanedVolumes(t *testing.T) {
	ctx := t.Context()
	log := testutils.TestLogger(t)

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	dataPath := t.TempDir()
	nodeId := "test-node-1"
	state := NewState()
	ops := newMockDiskVolumeOps()

	// Pre-populate local state with a volume that has no corresponding entity
	state.SetVolume("disk_volume/vol-orphan", &VolumeState{
		EntityId:   "disk_volume/vol-orphan",
		VolumeId:   "vol-orphan",
		DiskPath:   "/data/volumes/vol-orphan",
		SizeBytes:  10 * 1024 * 1024 * 1024,
		Filesystem: "ext4",
	})
	ops.existingPaths["/data/volumes/vol-orphan"] = true

	vc := newTestDiskVolumeController(log, dataPath, nodeId, es.EAC, state, ops)

	err := vc.ReconcileWithEntities(ctx)
	require.NoError(t, err)

	// Verify volume directory was removed
	assert.Contains(t, ops.removedDirs, "/data/volumes/vol-orphan")

	// Verify state was cleaned up
	assert.Nil(t, state.GetVolume("disk_volume/vol-orphan"))
}

func TestDiskVolumeControllerReconcileKeepsNonOrphanedVolumes(t *testing.T) {
	ctx := t.Context()
	log := testutils.TestLogger(t)

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	dataPath := t.TempDir()
	nodeId := "test-node-1"
	state := NewState()
	ops := newMockDiskVolumeOps()

	volPath := filepath.Join(dataPath, "volumes", "vol-keep")
	state.SetVolume("disk_volume/vol-keep", &VolumeState{
		EntityId:   "disk_volume/vol-keep",
		VolumeId:   "vol-keep",
		DiskPath:   volPath,
		SizeBytes:  10 * 1024 * 1024 * 1024,
		Filesystem: "ext4",
	})
	ops.existingPaths[volPath] = true

	// Also add an orphan
	state.SetVolume("disk_volume/vol-orphan2", &VolumeState{
		EntityId:   "disk_volume/vol-orphan2",
		VolumeId:   "vol-orphan2",
		DiskPath:   "/data/volumes/vol-orphan2",
		SizeBytes:  5 * 1024 * 1024 * 1024,
		Filesystem: "ext4",
	})
	ops.existingPaths["/data/volumes/vol-orphan2"] = true

	vc := newTestDiskVolumeController(log, dataPath, nodeId, es.EAC, state, ops)

	vol := &storage_v1alpha.DiskVolume{
		ID:           "disk_volume/vol-keep",
		NodeId:       entity.Id("node/" + nodeId),
		SizeGb:       10,
		Filesystem:   "ext4",
		DesiredState: storage_v1alpha.DV_PRESENT,
		ActualState:  storage_v1alpha.DV_READY,
		VolumeId:     "vol-keep",
	}
	createDiskVolumeEntity(ctx, t, es, vol)

	err := vc.ReconcileWithEntities(ctx)
	require.NoError(t, err)

	assert.NotNil(t, state.GetVolume("disk_volume/vol-keep"))
	assert.Nil(t, state.GetVolume("disk_volume/vol-orphan2"))
	assert.Contains(t, ops.removedDirs, "/data/volumes/vol-orphan2")
	assert.NotContains(t, ops.removedDirs, volPath)
}

func TestDiskVolumeControllerOrphanCleanupSkipsLsvdVolumes(t *testing.T) {
	ctx := t.Context()
	log := testutils.TestLogger(t)

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	dataPath := t.TempDir()
	nodeId := "test-node-1"
	state := NewState()
	ops := newMockDiskVolumeOps()

	// Pre-populate state with an LSVD volume (should NOT be cleaned up by DiskVolumeController)
	state.SetVolume("lsvd_volume/vol-lsvd", &VolumeState{
		EntityId:   "lsvd_volume/vol-lsvd",
		VolumeId:   "lsvd-vol-id",
		DiskPath:   "/data/volumes/lsvd-vol-id",
		SizeBytes:  10 * 1024 * 1024 * 1024,
		Filesystem: "ext4",
	})
	ops.existingPaths["/data/volumes/lsvd-vol-id"] = true

	vc := newTestDiskVolumeController(log, dataPath, nodeId, es.EAC, state, ops)

	err := vc.ReconcileWithEntities(ctx)
	require.NoError(t, err)

	// LSVD volume should still be in state (not orphan-cleaned by disk controller)
	assert.NotNil(t, state.GetVolume("lsvd_volume/vol-lsvd"))
	assert.NotContains(t, ops.removedDirs, "/data/volumes/lsvd-vol-id")
}

func TestDiskVolumeControllerMultipleVolumes(t *testing.T) {
	ctx := t.Context()
	log := testutils.TestLogger(t)

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	dataPath := t.TempDir()
	nodeId := "test-node-1"
	state := NewState()
	ops := newMockDiskVolumeOps()

	vc := newTestDiskVolumeController(log, dataPath, nodeId, es.EAC, state, ops)

	for i := 1; i <= 3; i++ {
		vol := &storage_v1alpha.DiskVolume{
			ID:           entity.Id("disk_volume/vol-" + string(rune('0'+i))),
			NodeId:       entity.Id("node/" + nodeId),
			SizeGb:       int64(i * 10),
			Filesystem:   "ext4",
			DesiredState: storage_v1alpha.DV_PRESENT,
			ActualState:  storage_v1alpha.DV_PENDING,
		}
		createDiskVolumeEntity(ctx, t, es, vol)
	}

	err := vc.ReconcileWithEntities(ctx)
	require.NoError(t, err)

	assert.Len(t, ops.createdDirs, 3)
	assert.Len(t, ops.createdImages, 3)
	assert.Len(t, state.Volumes, 3)
}

func TestDiskVolumeControllerReconcilePersistedVolumeOnDisk(t *testing.T) {
	ctx := t.Context()
	log := testutils.TestLogger(t)

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	dataPath := t.TempDir()
	nodeId := "test-node-1"
	state := NewState()
	ops := newMockDiskVolumeOps()

	volPath := filepath.Join(dataPath, "volumes", "vol-persist")
	// State has disk path and it exists on disk
	state.SetVolume("disk_volume/vol-persist", &VolumeState{
		EntityId:   "disk_volume/vol-persist",
		VolumeId:   "vol-persist",
		DiskPath:   volPath,
		SizeBytes:  10 * 1024 * 1024 * 1024,
		Filesystem: "ext4",
	})
	ops.existingPaths[volPath] = true

	vc := newTestDiskVolumeController(log, dataPath, nodeId, es.EAC, state, ops)

	// Entity is in PENDING state but local state has the volume on disk
	vol := &storage_v1alpha.DiskVolume{
		ID:           "disk_volume/vol-persist",
		NodeId:       entity.Id("node/" + nodeId),
		SizeGb:       10,
		Filesystem:   "ext4",
		DesiredState: storage_v1alpha.DV_PRESENT,
		ActualState:  storage_v1alpha.DV_PENDING,
	}
	createDiskVolumeEntity(ctx, t, es, vol)

	err := vc.ReconcileWithEntities(ctx)
	require.NoError(t, err)

	// Should NOT recreate (state found on disk), just update entity
	assert.Empty(t, ops.createdDirs)
	assert.Empty(t, ops.createdImages)

	// Entity should now be READY
	resp, err := es.EAC.Get(ctx, "disk_volume/vol-persist")
	require.NoError(t, err)
	var updated storage_v1alpha.DiskVolume
	updated.Decode(resp.Entity().Entity())
	assert.Equal(t, storage_v1alpha.DV_READY, updated.ActualState)
}

func TestDiskVolumeControllerDeleteNotInState(t *testing.T) {
	ctx := t.Context()
	log := testutils.TestLogger(t)

	es, cleanup := testutils.NewInMemEntityServer(t)
	defer cleanup()

	dataPath := t.TempDir()
	nodeId := "test-node-1"
	state := NewState()
	ops := newMockDiskVolumeOps()

	vc := newTestDiskVolumeController(log, dataPath, nodeId, es.EAC, state, ops)

	// Volume not in local state but entity requests deletion
	vol := &storage_v1alpha.DiskVolume{
		ID:           "disk_volume/vol-gone",
		NodeId:       entity.Id("node/" + nodeId),
		SizeGb:       10,
		DesiredState: storage_v1alpha.DV_ABSENT,
		ActualState:  storage_v1alpha.DV_READY,
	}
	createDiskVolumeEntity(ctx, t, es, vol)

	err := vc.ReconcileWithEntities(ctx)
	require.NoError(t, err)

	// Should still transition to DELETED even without local state
	resp, err := es.EAC.Get(ctx, "disk_volume/vol-gone")
	require.NoError(t, err)
	var updated storage_v1alpha.DiskVolume
	updated.Decode(resp.Entity().Entity())
	assert.Equal(t, storage_v1alpha.DV_DELETED, updated.ActualState)
}
