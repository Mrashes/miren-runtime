package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"strings"
	"time"

	"miren.dev/runtime/api/entityserver/entityserver_v1alpha"
	"miren.dev/runtime/api/storage/storage_v1alpha"
	"miren.dev/runtime/pkg/entity"
)

// MountController watches lsvd_mount entities and manages NBD devices and mounts
type MountController struct {
	log      *slog.Logger
	dataPath string
	nodeId   string
	eac      *entityserver_v1alpha.EntityAccessClient
	state    *State
	ops      MountOps

	// Active NBD handlers, keyed by entity ID
	handlers map[string]*nbdHandler
}

type nbdHandler struct {
	conn       net.Conn
	clientFile *os.File
	cancel     context.CancelFunc
	disk       LSVDDisk
}

// NewMountController creates a new mount controller
func NewMountController(log *slog.Logger, dataPath, nodeId string, eac *entityserver_v1alpha.EntityAccessClient, state *State, ops MountOps) *MountController {
	if ops == nil {
		ops = NewRealMountOps(log)
	}
	return &MountController{
		log:      log.With("module", "lsvd-mount"),
		dataPath: dataPath,
		nodeId:   nodeId,
		eac:      eac,
		state:    state,
		ops:      ops,
		handlers: make(map[string]*nbdHandler),
	}
}

// Run starts the mount controller and blocks until context is cancelled
func (c *MountController) Run(ctx context.Context) error {
	c.log.Info("starting mount controller")

	// Use polling-based reconciliation since WatchIndex over RPC uses streams
	// which require different handling. Poll every 5 seconds.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	c.log.Info("watching for lsvd_mount entities", "node_id", c.nodeId)

	// Initial reconciliation
	if err := c.ReconcileWithEntities(ctx); err != nil {
		c.log.Error("initial mount reconciliation failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			// Cleanup handlers on shutdown
			for entityId, h := range c.handlers {
				c.log.Info("cleaning up handler on shutdown", "entity_id", entityId)
				if h.cancel != nil {
					h.cancel()
				}
			}
			return nil
		case <-ticker.C:
			if err := c.ReconcileWithEntities(ctx); err != nil {
				c.log.Error("mount reconciliation failed", "error", err)
			}
		}
	}
}

// reconcileMount reconciles a single lsvd_mount entity
func (c *MountController) reconcileMount(ctx context.Context, mount *storage_v1alpha.LsvdMount) error {
	entityId := string(mount.ID)
	c.log.Info("reconciling mount",
		"entity_id", entityId,
		"desired_state", mount.DesiredState,
		"actual_state", mount.ActualState,
	)

	switch mount.DesiredState {
	case storage_v1alpha.MNT_WANT_MOUNTED:
		return c.reconcileMountMounted(ctx, mount)
	case storage_v1alpha.MNT_WANT_UNMOUNTED:
		return c.reconcileMountUnmounted(ctx, mount)
	default:
		c.log.Warn("unknown desired state", "desired_state", mount.DesiredState)
		return nil
	}
}

// reconcileMountMounted ensures the volume is mounted
func (c *MountController) reconcileMountMounted(ctx context.Context, mount *storage_v1alpha.LsvdMount) error {
	entityId := string(mount.ID)

	switch mount.ActualState {
	case storage_v1alpha.MNT_PENDING:
		return c.attachAndMount(ctx, mount)
	case storage_v1alpha.MNT_ATTACHING:
		// Already attaching, wait
		return nil
	case storage_v1alpha.MNT_ATTACHED:
		// Attached but not mounted, mount it
		return c.mountVolume(ctx, mount)
	case storage_v1alpha.MNT_MOUNTING:
		// Already mounting, wait
		return nil
	case storage_v1alpha.MNT_MOUNTED:
		// Already mounted, nothing to do
		return nil
	case storage_v1alpha.MNT_ERROR:
		// Error state, try to recover
		c.log.Info("mount in error state, attempting recovery", "entity_id", entityId)
		return c.attachAndMount(ctx, mount)
	default:
		c.log.Warn("unexpected actual state for mounted", "actual_state", mount.ActualState)
		return nil
	}
}

// reconcileMountUnmounted ensures the volume is unmounted
func (c *MountController) reconcileMountUnmounted(ctx context.Context, mount *storage_v1alpha.LsvdMount) error {
	entityId := string(mount.ID)

	switch mount.ActualState {
	case storage_v1alpha.MNT_DETACHED:
		// Already detached
		c.state.DeleteMount(entityId)
		if err := c.state.Save(); err != nil {
			c.log.Warn("failed to save state after mount cleanup", "error", err)
		}
		return nil
	case storage_v1alpha.MNT_UNMOUNTING, storage_v1alpha.MNT_DETACHING:
		// Already in progress
		return nil
	default:
		return c.unmountAndDetach(ctx, mount)
	}
}

// attachAndMount attaches NBD device and mounts the filesystem
func (c *MountController) attachAndMount(ctx context.Context, mount *storage_v1alpha.LsvdMount) error {
	entityId := string(mount.ID)
	volumeId := string(mount.VolumeId)

	c.log.Info("attaching and mounting volume",
		"entity_id", entityId,
		"volume_id", volumeId,
		"mount_path", mount.MountPath,
	)

	// Update actual_state to MNT_ATTACHING
	if err := c.updateMountState(ctx, mount.ID, storage_v1alpha.MNT_ATTACHING, 0, "", ""); err != nil {
		c.log.Warn("failed to update mount state to attaching", "error", err)
	}

	// Get volume state
	volState := c.state.GetVolume(volumeId)
	if volState == nil {
		c.setMountError(ctx, mount.ID, fmt.Sprintf("volume %s not found in state", volumeId))
		return fmt.Errorf("volume %s not found in state", volumeId)
	}

	// Open LSVD disk
	disk, err := c.ops.OpenLSVDDisk(ctx, volState.DiskPath, volState.VolumeId)
	if err != nil {
		c.setMountError(ctx, mount.ID, fmt.Sprintf("failed to open disk: %v", err))
		return fmt.Errorf("failed to open disk: %w", err)
	}

	// Attach NBD device
	sizeBytes := uint64(disk.Size())
	idx, conn, clientFile, cleanup, err := c.ops.NBDLoopback(ctx, sizeBytes)
	if err != nil {
		disk.Close(ctx)
		return fmt.Errorf("failed to setup NBD loopback: %w", err)
	}

	// Create device node
	devicePath := c.getDevicePath(volState.VolumeId)
	dir := filepath.Dir(devicePath)
	if err := c.ops.CreateDir(dir, 0755); err != nil {
		cleanup()
		disk.Close(ctx)
		return fmt.Errorf("failed to create device directory: %w", err)
	}

	if err := c.ops.CreateDeviceNode(devicePath, idx); err != nil {
		cleanup()
		disk.Close(ctx)
		return fmt.Errorf("failed to create device node: %w", err)
	}

	// Start NBD handler
	handlerCtx, handlerCancel := context.WithCancel(ctx)
	go c.runNBDHandler(handlerCtx, entityId, disk, conn, clientFile)

	// Wait for NBD device to be ready
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if err := c.ops.NBDStatus(idx); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			handlerCancel()
			cleanup()
			disk.Close(ctx)
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Update state
	c.state.SetMount(entityId, &MountState{
		EntityId:   entityId,
		VolumeId:   volumeId,
		NbdIndex:   idx,
		DevicePath: devicePath,
		MountPath:  mount.MountPath,
		Mounted:    false,
		ReadOnly:   mount.ReadOnly,
		LeaseNonce: mount.LeaseNonce,
	})

	c.handlers[entityId] = &nbdHandler{
		conn:       conn,
		clientFile: clientFile,
		cancel:     handlerCancel,
		disk:       disk,
	}

	if err := c.state.Save(); err != nil {
		c.log.Warn("failed to save state after NBD attach", "error", err)
	}

	// Update actual_state to MNT_ATTACHED
	if err := c.updateMountState(ctx, mount.ID, storage_v1alpha.MNT_ATTACHED, int64(idx), devicePath, ""); err != nil {
		c.log.Warn("failed to update mount state to attached", "error", err)
	}

	// Now mount the volume
	return c.mountVolume(ctx, mount)
}

// mountVolume mounts the filesystem
func (c *MountController) mountVolume(ctx context.Context, mount *storage_v1alpha.LsvdMount) error {
	entityId := string(mount.ID)

	mountState := c.state.GetMount(entityId)
	if mountState == nil {
		return fmt.Errorf("mount state not found for %s", entityId)
	}

	c.log.Info("mounting filesystem",
		"entity_id", entityId,
		"device", mountState.DevicePath,
		"mount_path", mount.MountPath,
	)

	// Update actual_state to MNT_MOUNTING
	if err := c.updateMountState(ctx, mount.ID, storage_v1alpha.MNT_MOUNTING, 0, "", ""); err != nil {
		c.log.Warn("failed to update mount state to mounting", "error", err)
	}

	// Create mount point
	if err := c.ops.CreateDir(mount.MountPath, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	// Get volume state for filesystem info
	volState := c.state.GetVolume(mountState.VolumeId)
	filesystem := "ext4"
	if volState != nil && volState.Filesystem != "" {
		filesystem = volState.Filesystem
	}

	// Format if needed (check for existing filesystem)
	formatted, err := c.ops.IsFormatted(mountState.DevicePath, filesystem)
	if err != nil {
		c.log.Warn("failed to check if formatted", "error", err)
	}

	if !formatted {
		c.log.Info("formatting device", "device", mountState.DevicePath, "filesystem", filesystem)
		if err := c.ops.FormatDevice(ctx, mountState.DevicePath, filesystem); err != nil {
			return fmt.Errorf("failed to format device: %w", err)
		}
	}

	// Mount the filesystem
	if err := c.ops.Mount(mountState.DevicePath, mount.MountPath, filesystem, mount.ReadOnly); err != nil {
		return fmt.Errorf("failed to mount: %w", err)
	}

	// Update state
	mountState.Mounted = true
	c.state.SetMount(entityId, mountState)
	if err := c.state.Save(); err != nil {
		c.log.Warn("failed to save state after mount", "error", err)
	}

	c.log.Info("filesystem mounted",
		"entity_id", entityId,
		"mount_path", mount.MountPath,
	)

	// Update actual_state to MNT_MOUNTED
	if err := c.updateMountState(ctx, mount.ID, storage_v1alpha.MNT_MOUNTED, 0, "", ""); err != nil {
		c.log.Warn("failed to update mount state to mounted", "error", err)
	}

	return nil
}

// unmountAndDetach unmounts the filesystem and detaches the NBD device
func (c *MountController) unmountAndDetach(ctx context.Context, mount *storage_v1alpha.LsvdMount) error {
	entityId := string(mount.ID)

	c.log.Info("unmounting and detaching",
		"entity_id", entityId,
		"mount_path", mount.MountPath,
	)

	// Update actual_state to MNT_UNMOUNTING
	if err := c.updateMountState(ctx, mount.ID, storage_v1alpha.MNT_UNMOUNTING, 0, "", ""); err != nil {
		c.log.Warn("failed to update mount state to unmounting", "error", err)
	}

	mountState := c.state.GetMount(entityId)
	if mountState == nil {
		c.log.Warn("mount state not found", "entity_id", entityId)
		// Update actual_state to MNT_DETACHED
		if err := c.updateMountState(ctx, mount.ID, storage_v1alpha.MNT_DETACHED, 0, "", ""); err != nil {
			c.log.Warn("failed to update mount state to detached", "error", err)
		}
		return nil
	}

	// Unmount if mounted
	if mountState.Mounted {
		if err := c.ops.Unmount(mountState.MountPath); err != nil {
			c.log.Warn("failed to unmount", "error", err)
		}
		mountState.Mounted = false
		c.state.SetMount(entityId, mountState)
	}

	// Update actual_state to MNT_DETACHING
	if err := c.updateMountState(ctx, mount.ID, storage_v1alpha.MNT_DETACHING, 0, "", ""); err != nil {
		c.log.Warn("failed to update mount state to detaching", "error", err)
	}

	// Stop NBD handler
	if h, ok := c.handlers[entityId]; ok {
		if h.cancel != nil {
			h.cancel()
		}
		if h.disk != nil {
			h.disk.Close(ctx)
		}
		delete(c.handlers, entityId)
	}

	// Disconnect NBD device
	if mountState.NbdIndex > 0 {
		if err := c.ops.NBDDisconnect(mountState.NbdIndex); err != nil {
			c.log.Warn("failed to disconnect NBD", "error", err)
		}
	}

	// Remove device node
	if mountState.DevicePath != "" {
		c.ops.RemoveFile(mountState.DevicePath)
	}

	// Update state
	c.state.DeleteMount(entityId)
	if err := c.state.Save(); err != nil {
		c.log.Warn("failed to save state after unmount", "error", err)
	}

	c.log.Info("volume unmounted and detached", "entity_id", entityId)

	// Update actual_state to MNT_DETACHED
	if err := c.updateMountState(ctx, mount.ID, storage_v1alpha.MNT_DETACHED, 0, "", ""); err != nil {
		c.log.Warn("failed to update mount state to detached", "error", err)
	}

	return nil
}

// runNBDHandler runs the NBD handler for a mounted volume
func (c *MountController) runNBDHandler(ctx context.Context, entityId string, disk LSVDDisk, conn net.Conn, clientFile *os.File) {
	c.log.Info("starting NBD handler", "entity_id", entityId)
	defer c.log.Info("NBD handler stopped", "entity_id", entityId)
	defer clientFile.Close()
	defer conn.Close()

	if err := disk.HandleNBD(ctx, conn, clientFile); err != nil {
		c.log.Warn("NBD handler error", "entity_id", entityId, "error", err)
	}
}

// getDevicePath returns the path to a device node
func (c *MountController) getDevicePath(volumeId string) string {
	return filepath.Join(c.dataPath, "devices", strings.ReplaceAll(volumeId, "/", "-"))
}

// ReconcileWithSystem reconciles mount state with the actual system
func (c *MountController) ReconcileWithSystem(ctx context.Context) error {
	c.log.Info("reconciling mounts with system")

	for entityId, mountState := range c.state.Mounts {
		// Check if we have an active handler for this mount
		_, hasHandler := c.handlers[entityId]

		// Check if NBD device is still connected
		nbdConnected := false
		if mountState.NbdIndex > 0 {
			err := c.ops.NBDStatus(mountState.NbdIndex)
			nbdConnected = (err == nil)
		}

		if !hasHandler || !nbdConnected {
			c.log.Info("NBD handler needs reconnection",
				"entity_id", entityId,
				"has_handler", hasHandler,
				"nbd_connected", nbdConnected,
				"nbd_index", mountState.NbdIndex,
			)

			// Try to reconnect the NBD device
			if err := c.reconnectNBD(ctx, entityId, mountState); err != nil {
				c.log.Error("failed to reconnect NBD",
					"entity_id", entityId,
					"error", err,
				)
				// Mark mount state as needing recovery
				mountState.Mounted = false
				c.state.SetMount(entityId, mountState)
				continue
			}
		}

		// Check if still mounted
		if mountState.Mounted {
			if !c.ops.IsMounted(mountState.MountPath) {
				c.log.Warn("volume not mounted but should be",
					"entity_id", entityId,
					"mount_path", mountState.MountPath,
				)
				mountState.Mounted = false
				c.state.SetMount(entityId, mountState)

				// Try to remount
				if err := c.remountFilesystem(ctx, entityId, mountState); err != nil {
					c.log.Error("failed to remount filesystem",
						"entity_id", entityId,
						"error", err,
					)
				}
			}
		}
	}

	if err := c.state.Save(); err != nil {
		c.log.Warn("failed to save state after system reconciliation", "error", err)
	}

	return nil
}

// reconnectNBD reconnects the NBD device for a mount after process restart
func (c *MountController) reconnectNBD(ctx context.Context, entityId string, mountState *MountState) error {
	// Get volume state
	volState := c.state.GetVolume(mountState.VolumeId)
	if volState == nil {
		return fmt.Errorf("volume %s not found in state", mountState.VolumeId)
	}

	c.log.Info("reconnecting NBD device",
		"entity_id", entityId,
		"volume_id", mountState.VolumeId,
		"disk_path", volState.DiskPath,
	)

	// Clean up old handler if exists
	if h, ok := c.handlers[entityId]; ok {
		if h.cancel != nil {
			h.cancel()
		}
		if h.disk != nil {
			h.disk.Close(ctx)
		}
		delete(c.handlers, entityId)
	}

	// Disconnect old NBD device if still partially connected
	if mountState.NbdIndex > 0 {
		_ = c.ops.NBDDisconnect(mountState.NbdIndex)
	}

	// Open LSVD disk
	disk, err := c.ops.OpenLSVDDisk(ctx, volState.DiskPath, volState.VolumeId)
	if err != nil {
		return fmt.Errorf("failed to open disk: %w", err)
	}

	// Attach NBD device
	sizeBytes := uint64(disk.Size())
	idx, conn, clientFile, cleanup, err := c.ops.NBDLoopback(ctx, sizeBytes)
	if err != nil {
		disk.Close(ctx)
		return fmt.Errorf("failed to setup NBD loopback: %w", err)
	}

	// Create device node
	devicePath := c.getDevicePath(volState.VolumeId)
	dir := filepath.Dir(devicePath)
	if err := c.ops.CreateDir(dir, 0755); err != nil {
		cleanup()
		disk.Close(ctx)
		return fmt.Errorf("failed to create device directory: %w", err)
	}

	if err := c.ops.CreateDeviceNode(devicePath, idx); err != nil {
		cleanup()
		disk.Close(ctx)
		return fmt.Errorf("failed to create device node: %w", err)
	}

	// Start NBD handler
	handlerCtx, handlerCancel := context.WithCancel(ctx)
	go c.runNBDHandler(handlerCtx, entityId, disk, conn, clientFile)

	// Wait for NBD device to be ready
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if err := c.ops.NBDStatus(idx); err == nil {
			break
		}
		select {
		case <-ctx.Done():
			handlerCancel()
			cleanup()
			disk.Close(ctx)
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}

	// Update state with new NBD index
	mountState.NbdIndex = idx
	mountState.DevicePath = devicePath
	c.state.SetMount(entityId, mountState)

	c.handlers[entityId] = &nbdHandler{
		conn:       conn,
		clientFile: clientFile,
		cancel:     handlerCancel,
		disk:       disk,
	}

	c.log.Info("NBD device reconnected",
		"entity_id", entityId,
		"nbd_index", idx,
		"device_path", devicePath,
	)

	return nil
}

// remountFilesystem remounts the filesystem after recovery
func (c *MountController) remountFilesystem(ctx context.Context, entityId string, mountState *MountState) error {
	// Get volume state for filesystem info
	volState := c.state.GetVolume(mountState.VolumeId)
	filesystem := "ext4"
	if volState != nil && volState.Filesystem != "" {
		filesystem = volState.Filesystem
	}

	c.log.Info("remounting filesystem",
		"entity_id", entityId,
		"device", mountState.DevicePath,
		"mount_path", mountState.MountPath,
		"filesystem", filesystem,
	)

	// Create mount point if needed
	if err := c.ops.CreateDir(mountState.MountPath, 0755); err != nil {
		return fmt.Errorf("failed to create mount point: %w", err)
	}

	// Mount the filesystem
	if err := c.ops.Mount(mountState.DevicePath, mountState.MountPath, filesystem, mountState.ReadOnly); err != nil {
		return fmt.Errorf("failed to mount: %w", err)
	}

	// Update state
	mountState.Mounted = true
	c.state.SetMount(entityId, mountState)

	c.log.Info("filesystem remounted",
		"entity_id", entityId,
		"mount_path", mountState.MountPath,
	)

	return nil
}

// ReconcileWithEntities reconciles local state with entity server
func (c *MountController) ReconcileWithEntities(ctx context.Context) error {
	c.log.Debug("reconciling mounts with entity server")

	// List all lsvd_mount entities for this node
	nodeIdRef := entity.Id(c.nodeId)
	indexAttr := entity.Ref(storage_v1alpha.LsvdMountNodeIdId, nodeIdRef)

	resp, err := c.eac.List(ctx, indexAttr)
	if err != nil {
		return fmt.Errorf("failed to list mount entities: %w", err)
	}

	values := resp.Values()
	c.log.Debug("found mount entities", "count", len(values))

	for _, entResp := range values {
		var mount storage_v1alpha.LsvdMount
		mount.Decode(entResp.Entity())

		// Skip if not for this node
		if string(mount.NodeId) != c.nodeId {
			continue
		}

		// Reconcile the mount
		if err := c.reconcileMount(ctx, &mount); err != nil {
			c.log.Error("failed to reconcile mount",
				"entity_id", mount.ID,
				"error", err,
			)
		}
	}

	return nil
}

// mountActualStateToId maps LsvdMountActualState to entity.Id
func mountActualStateToId(state storage_v1alpha.LsvdMountActualState) entity.Id {
	switch state {
	case storage_v1alpha.MNT_PENDING:
		return storage_v1alpha.LsvdMountActualStateMntPendingId
	case storage_v1alpha.MNT_ATTACHING:
		return storage_v1alpha.LsvdMountActualStateMntAttachingId
	case storage_v1alpha.MNT_ATTACHED:
		return storage_v1alpha.LsvdMountActualStateMntAttachedId
	case storage_v1alpha.MNT_MOUNTING:
		return storage_v1alpha.LsvdMountActualStateMntMountingId
	case storage_v1alpha.MNT_MOUNTED:
		return storage_v1alpha.LsvdMountActualStateMntMountedId
	case storage_v1alpha.MNT_UNMOUNTING:
		return storage_v1alpha.LsvdMountActualStateMntUnmountingId
	case storage_v1alpha.MNT_DETACHING:
		return storage_v1alpha.LsvdMountActualStateMntDetachingId
	case storage_v1alpha.MNT_DETACHED:
		return storage_v1alpha.LsvdMountActualStateMntDetachedId
	case storage_v1alpha.MNT_ERROR:
		return storage_v1alpha.LsvdMountActualStateMntErrorId
	default:
		return storage_v1alpha.LsvdMountActualStateMntPendingId
	}
}

// updateMountState updates the actual_state and optionally other fields in the entity
func (c *MountController) updateMountState(ctx context.Context, id entity.Id, state storage_v1alpha.LsvdMountActualState, nbdIndex int64, devicePath, errorMsg string) error {
	// Get the entity.Id for the state
	stateId := mountActualStateToId(state)

	// Build attrs for the update - include entity ID for Patch
	attrs := []entity.Attr{
		entity.Ref(entity.DBId, id),
		entity.Ref(storage_v1alpha.LsvdMountActualStateId, stateId),
	}

	if nbdIndex > 0 {
		attrs = append(attrs, entity.Int64(storage_v1alpha.LsvdMountNbdIndexId, nbdIndex))
	}

	if devicePath != "" {
		attrs = append(attrs, entity.String(storage_v1alpha.LsvdMountDevicePathId, devicePath))
	}

	if errorMsg != "" {
		attrs = append(attrs, entity.String(storage_v1alpha.LsvdMountErrorMessageId, errorMsg))
	}

	_, err := c.eac.Patch(ctx, attrs, 0)
	return err
}

// setMountError sets the mount to error state with a message
func (c *MountController) setMountError(ctx context.Context, id entity.Id, errorMsg string) {
	if err := c.updateMountState(ctx, id, storage_v1alpha.MNT_ERROR, 0, "", errorMsg); err != nil {
		c.log.Warn("failed to set mount error state", "entity_id", id, "error", err)
	}
}
