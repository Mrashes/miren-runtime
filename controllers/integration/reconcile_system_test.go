package integration

import (
	"context"
	"testing"

	storage "miren.dev/runtime/api/storage/storage_v1alpha"
	lsvdserver "miren.dev/runtime/components/lsvd/server"
	"miren.dev/runtime/pkg/entity"
)

// TestReconcileWithSystemSkipsUnmountingMounts verifies that ReconcileWithSystem
// does NOT attempt to reconnect NBD devices for mounts whose entity has
// desired_state=MNT_WANT_UNMOUNTED. This prevents a race condition where:
//
//  1. unmountAndDetach() deletes the handler from the map but hasn't yet cleaned
//     up local state
//  2. ReconcileWithSystem fires on its 30s timer, finds the mount state with no
//     handler, and wastes time reconnecting a mount that's being torn down
func TestReconcileWithSystemSkipsUnmountingMounts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	h := NewTestHarness(t)

	// Set up volume state in LSVD state so reconnectNBD can find the volume
	volumeEntityId := "lsvd_volume/test-vol-1"
	h.LsvdState.SetVolume(volumeEntityId, &lsvdserver.VolumeState{
		EntityId:   volumeEntityId,
		VolumeId:   "cloud-vol-1",
		DiskPath:   "/tmp/test-vol",
		SizeBytes:  10 * 1024 * 1024 * 1024,
		Filesystem: "ext4",
	})

	// Create lsvd_mount entity with desired_state=MNT_WANT_UNMOUNTED.
	// This simulates the state during unmount: the disk lease controller has
	// already set desired_state to MNT_WANT_UNMOUNTED but the mount controller
	// is partway through unmountAndDetach.
	mountEntityId := entity.Id("lsvd_mount/test-mnt-1")
	mount := &storage.LsvdMount{
		DesiredState: storage.MNT_WANT_UNMOUNTED,
		ActualState:  storage.MNT_MOUNTED,
		NodeId:       entity.Id("node/" + testNodeId),
		VolumeId:     entity.Id(volumeEntityId),
		MountPath:    "/mnt/test",
	}
	_, err := h.EAC.Create(ctx, entity.New(
		entity.DBId, mountEntityId,
		mount.Encode,
	).Attrs())
	if err != nil {
		t.Fatalf("failed to create lsvd_mount entity: %v", err)
	}

	// Set mount state in LSVD local state with NO handler in the handlers map.
	// This simulates the race window: unmountAndDetach has cancelled the handler
	// and deleted it from the map, but hasn't yet cleaned up local state.
	h.LsvdState.SetMount(string(mountEntityId), &lsvdserver.MountState{
		EntityId:   string(mountEntityId),
		VolumeId:   volumeEntityId,
		NbdIndex:   5,
		DevicePath: "/dev/nbd5",
		MountPath:  "/mnt/test",
		Mounted:    true,
		LeaseNonce: "old-nonce",
	})

	callsBefore := h.MockMountOps.openDiskCalls

	// ReconcileWithSystem should see the mount state with no handler but
	// skip reconnection because the entity's desired_state is MNT_WANT_UNMOUNTED.
	err = h.LsvdMountCtrl.ReconcileWithSystem(ctx)
	if err != nil {
		t.Fatalf("ReconcileWithSystem failed: %v", err)
	}

	newCalls := h.MockMountOps.openDiskCalls - callsBefore
	if newCalls != 0 {
		t.Errorf("ReconcileWithSystem reconnected a mount being unmounted: OpenLSVDDisk called %d time(s), want 0", newCalls)
	}
}

// TestReconcileWithSystemReconnectsWantedMounts verifies that ReconcileWithSystem
// still reconnects mounts whose entity has desired_state=MNT_WANT_MOUNTED
// (the normal process-restart recovery path).
func TestReconcileWithSystemReconnectsWantedMounts(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	h := NewTestHarness(t)

	volumeEntityId := "lsvd_volume/test-vol-2"
	h.LsvdState.SetVolume(volumeEntityId, &lsvdserver.VolumeState{
		EntityId:   volumeEntityId,
		VolumeId:   "cloud-vol-2",
		DiskPath:   "/tmp/test-vol-2",
		SizeBytes:  10 * 1024 * 1024 * 1024,
		Filesystem: "ext4",
	})

	// Create lsvd_mount entity with desired_state=MNT_WANT_MOUNTED.
	// This simulates a process restart where the mount should be reconnected.
	mountEntityId := entity.Id("lsvd_mount/test-mnt-2")
	mount := &storage.LsvdMount{
		DesiredState: storage.MNT_WANT_MOUNTED,
		ActualState:  storage.MNT_MOUNTED,
		NodeId:       entity.Id("node/" + testNodeId),
		VolumeId:     entity.Id(volumeEntityId),
		MountPath:    "/mnt/test-2",
	}
	_, err := h.EAC.Create(ctx, entity.New(
		entity.DBId, mountEntityId,
		mount.Encode,
	).Attrs())
	if err != nil {
		t.Fatalf("failed to create lsvd_mount entity: %v", err)
	}

	h.LsvdState.SetMount(string(mountEntityId), &lsvdserver.MountState{
		EntityId:   string(mountEntityId),
		VolumeId:   volumeEntityId,
		NbdIndex:   5,
		DevicePath: "/dev/nbd5",
		MountPath:  "/mnt/test-2",
		Mounted:    true,
		LeaseNonce: "old-nonce",
	})

	callsBefore := h.MockMountOps.openDiskCalls

	err = h.LsvdMountCtrl.ReconcileWithSystem(ctx)
	if err != nil {
		t.Fatalf("ReconcileWithSystem failed: %v", err)
	}

	newCalls := h.MockMountOps.openDiskCalls - callsBefore
	if newCalls == 0 {
		t.Errorf("ReconcileWithSystem should reconnect mounts with desired_state=MNT_WANT_MOUNTED, but OpenLSVDDisk was not called")
	}
}

// TestReconcileWithSystemSkipsDeletedMountEntity verifies that ReconcileWithSystem
// does NOT attempt to reconnect mounts whose entity no longer exists in the
// entity store. This handles the case where the mount entity was already cleaned
// up but local state lingers.
func TestReconcileWithSystemSkipsDeletedMountEntity(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	h := NewTestHarness(t)

	volumeEntityId := "lsvd_volume/test-vol-3"
	h.LsvdState.SetVolume(volumeEntityId, &lsvdserver.VolumeState{
		EntityId:   volumeEntityId,
		VolumeId:   "cloud-vol-3",
		DiskPath:   "/tmp/test-vol-3",
		SizeBytes:  10 * 1024 * 1024 * 1024,
		Filesystem: "ext4",
	})

	// Set mount state in LSVD local state but do NOT create the entity.
	// This simulates a case where the entity was deleted but local state wasn't
	// cleaned up yet.
	mountEntityId := "lsvd_mount/test-mnt-3"
	h.LsvdState.SetMount(mountEntityId, &lsvdserver.MountState{
		EntityId:   mountEntityId,
		VolumeId:   volumeEntityId,
		NbdIndex:   5,
		DevicePath: "/dev/nbd5",
		MountPath:  "/mnt/test-3",
		Mounted:    true,
		LeaseNonce: "old-nonce",
	})

	callsBefore := h.MockMountOps.openDiskCalls

	err := h.LsvdMountCtrl.ReconcileWithSystem(ctx)
	if err != nil {
		t.Fatalf("ReconcileWithSystem failed: %v", err)
	}

	newCalls := h.MockMountOps.openDiskCalls - callsBefore
	if newCalls != 0 {
		t.Errorf("ReconcileWithSystem reconnected a mount with deleted entity: OpenLSVDDisk called %d time(s), want 0", newCalls)
	}
}
