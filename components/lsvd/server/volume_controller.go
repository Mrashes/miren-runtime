package server

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"time"

	"github.com/google/uuid"

	"miren.dev/runtime/api/entityserver/entityserver_v1alpha"
	"miren.dev/runtime/api/storage/storage_v1alpha"
	"miren.dev/runtime/pkg/entity"
	"miren.dev/runtime/pkg/units"
)

// VolumeController watches lsvd_volume entities and manages LSVD volumes
type VolumeController struct {
	log      *slog.Logger
	dataPath string
	nodeId   string
	eac      *entityserver_v1alpha.EntityAccessClient
	state    *State
	ops      VolumeOps
}

// NewVolumeController creates a new volume controller
func NewVolumeController(log *slog.Logger, dataPath, nodeId string, eac *entityserver_v1alpha.EntityAccessClient, state *State, ops VolumeOps) *VolumeController {
	if ops == nil {
		ops = NewRealVolumeOps()
	}
	return &VolumeController{
		log:      log.With("module", "lsvd-volume"),
		dataPath: dataPath,
		nodeId:   nodeId,
		eac:      eac,
		state:    state,
		ops:      ops,
	}
}

// Run starts the volume controller and blocks until context is cancelled
func (c *VolumeController) Run(ctx context.Context) error {
	c.log.Info("starting volume controller")

	// Use polling-based reconciliation since WatchIndex over RPC uses streams
	// which require different handling. Poll every 5 seconds.
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	c.log.Info("watching for lsvd_volume entities", "node_id", c.nodeId)

	// Initial reconciliation
	if err := c.ReconcileWithEntities(ctx); err != nil {
		c.log.Error("initial volume reconciliation failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			if err := c.ReconcileWithEntities(ctx); err != nil {
				c.log.Error("volume reconciliation failed", "error", err)
			}
		}
	}
}

// reconcileVolume reconciles a single lsvd_volume entity
func (c *VolumeController) reconcileVolume(ctx context.Context, volume *storage_v1alpha.LsvdVolume) error {
	entityId := string(volume.ID)
	c.log.Info("reconciling volume",
		"entity_id", entityId,
		"desired_state", volume.DesiredState,
		"actual_state", volume.ActualState,
	)

	switch volume.DesiredState {
	case storage_v1alpha.VOL_PRESENT:
		return c.reconcileVolumePresent(ctx, volume)
	case storage_v1alpha.VOL_ABSENT:
		return c.reconcileVolumeAbsent(ctx, volume)
	default:
		c.log.Warn("unknown desired state", "desired_state", volume.DesiredState)
		return nil
	}
}

// reconcileVolumePresent ensures the volume exists
func (c *VolumeController) reconcileVolumePresent(ctx context.Context, volume *storage_v1alpha.LsvdVolume) error {
	entityId := string(volume.ID)

	// Check if volume already exists in our state
	if existing := c.state.GetVolume(entityId); existing != nil {
		// Volume already exists, verify it's in a good state
		if volume.ActualState == storage_v1alpha.VOL_READY {
			c.log.Debug("volume already ready", "entity_id", entityId)
			return nil
		}
	}

	// Need to create the volume
	switch volume.ActualState {
	case storage_v1alpha.VOL_PENDING:
		return c.createVolume(ctx, volume)
	case storage_v1alpha.VOL_CREATING:
		// Volume is being created, wait for it
		c.log.Debug("volume is being created", "entity_id", entityId)
		return nil
	case storage_v1alpha.VOL_READY:
		// Volume is ready, nothing to do
		return nil
	case storage_v1alpha.VOL_ERROR:
		// Volume is in error state, try to recreate
		c.log.Info("volume in error state, attempting recreation", "entity_id", entityId)
		return c.createVolume(ctx, volume)
	default:
		c.log.Warn("unexpected actual state for present volume", "actual_state", volume.ActualState)
		return nil
	}
}

// reconcileVolumeAbsent ensures the volume is deleted
func (c *VolumeController) reconcileVolumeAbsent(ctx context.Context, volume *storage_v1alpha.LsvdVolume) error {
	entityId := string(volume.ID)

	switch volume.ActualState {
	case storage_v1alpha.VOL_DELETED:
		// Volume is already deleted
		c.state.DeleteVolume(entityId)
		if err := c.state.Save(); err != nil {
			c.log.Warn("failed to save state after volume deletion", "error", err)
		}
		return nil
	case storage_v1alpha.VOL_DELETING:
		// Volume is being deleted, wait for it
		return nil
	default:
		return c.deleteVolume(ctx, volume)
	}
}

// createVolume creates a new LSVD volume
func (c *VolumeController) createVolume(ctx context.Context, volume *storage_v1alpha.LsvdVolume) error {
	entityId := string(volume.ID)

	c.log.Info("creating volume",
		"entity_id", entityId,
		"size_gb", volume.SizeGb,
		"filesystem", volume.Filesystem,
		"remote_only", volume.RemoteOnly,
	)

	// Update actual state to creating
	if err := c.updateVolumeState(ctx, volume.ID, storage_v1alpha.VOL_CREATING, "", ""); err != nil {
		c.log.Warn("failed to update volume state to creating", "error", err)
	}

	// Generate volume ID
	u, err := uuid.NewV7()
	if err != nil {
		c.setVolumeError(ctx, volume.ID, fmt.Sprintf("failed to generate volume UUID: %v", err))
		return fmt.Errorf("failed to generate volume UUID: %w", err)
	}
	volumeId := u.String()

	// Create volume directory
	volumePath := c.getVolumePath(volumeId)
	if err := c.ops.CreateVolumeDir(volumePath); err != nil {
		c.setVolumeError(ctx, volume.ID, fmt.Sprintf("failed to create volume directory: %v", err))
		return fmt.Errorf("failed to create volume directory: %w", err)
	}

	// Initialize LSVD volume
	metadata := map[string]any{
		"filesystem": volume.Filesystem,
	}
	if err := c.ops.InitLSVDVolume(ctx, volumePath, volumeId, units.GigaBytes(volume.SizeGb).Bytes(), metadata); err != nil {
		c.setVolumeError(ctx, volume.ID, fmt.Sprintf("failed to init volume: %v", err))
		return fmt.Errorf("failed to init volume: %w", err)
	}

	// Update state
	c.state.SetVolume(entityId, &VolumeState{
		EntityId:   entityId,
		VolumeId:   volumeId,
		DiskPath:   volumePath,
		SizeBytes:  units.GigaBytes(volume.SizeGb).Bytes().Int64(),
		Filesystem: volume.Filesystem,
		RemoteOnly: volume.RemoteOnly,
	})

	if err := c.state.Save(); err != nil {
		c.log.Warn("failed to save state after volume creation", "error", err)
	}

	c.log.Info("volume created",
		"entity_id", entityId,
		"volume_id", volumeId,
	)

	// Update entity actual_state to VOL_READY and set volume_id
	if err := c.updateVolumeState(ctx, volume.ID, storage_v1alpha.VOL_READY, volumeId, ""); err != nil {
		c.log.Warn("failed to update volume state to ready", "error", err)
	}

	return nil
}

// deleteVolume deletes an LSVD volume
func (c *VolumeController) deleteVolume(ctx context.Context, volume *storage_v1alpha.LsvdVolume) error {
	entityId := string(volume.ID)

	c.log.Info("deleting volume", "entity_id", entityId)

	// Update actual state to deleting
	if err := c.updateVolumeState(ctx, volume.ID, storage_v1alpha.VOL_DELETING, "", ""); err != nil {
		c.log.Warn("failed to update volume state to deleting", "error", err)
	}

	// Get volume state
	volState := c.state.GetVolume(entityId)
	if volState == nil {
		c.log.Warn("volume not found in state", "entity_id", entityId)
		// Update entity to VOL_DELETED
		if err := c.updateVolumeState(ctx, volume.ID, storage_v1alpha.VOL_DELETED, "", ""); err != nil {
			c.log.Warn("failed to update volume state to deleted", "error", err)
		}
		return nil
	}

	// Delete volume directory
	if volState.DiskPath != "" {
		if err := c.ops.RemoveVolumeDir(volState.DiskPath); err != nil {
			c.log.Warn("failed to remove volume directory", "path", volState.DiskPath, "error", err)
		}
	}

	// Update state
	c.state.DeleteVolume(entityId)
	if err := c.state.Save(); err != nil {
		c.log.Warn("failed to save state after volume deletion", "error", err)
	}

	c.log.Info("volume deleted", "entity_id", entityId)

	// Update entity actual_state to VOL_DELETED
	if err := c.updateVolumeState(ctx, volume.ID, storage_v1alpha.VOL_DELETED, "", ""); err != nil {
		c.log.Warn("failed to update volume state to deleted", "error", err)
	}

	return nil
}

// getVolumePath returns the path to a volume's data directory
func (c *VolumeController) getVolumePath(volumeId string) string {
	return filepath.Join(c.dataPath, "volumes", volumeId)
}

// ReconcileWithSystem reconciles volume state with the actual system
func (c *VolumeController) ReconcileWithSystem(ctx context.Context) error {
	c.log.Info("reconciling volumes with system")

	for entityId, volState := range c.state.Volumes {
		// Verify volume directory exists
		if volState.DiskPath != "" {
			if !c.ops.VolumePathExists(volState.DiskPath) {
				c.log.Warn("volume directory missing",
					"entity_id", entityId,
					"path", volState.DiskPath,
				)
				// TODO: Mark as error in etcd
			}
		}
	}

	return nil
}

// ReconcileWithEntities reconciles local state with entity server
func (c *VolumeController) ReconcileWithEntities(ctx context.Context) error {
	c.log.Debug("reconciling volumes with entity server")

	// List all lsvd_volume entities for this node
	nodeIdRef := entity.Id(c.nodeId)
	indexAttr := entity.Ref(storage_v1alpha.LsvdVolumeNodeIdId, nodeIdRef)

	resp, err := c.eac.List(ctx, indexAttr)
	if err != nil {
		return fmt.Errorf("failed to list volume entities: %w", err)
	}

	values := resp.Values()
	c.log.Debug("found volume entities", "count", len(values))

	for _, entResp := range values {
		var volume storage_v1alpha.LsvdVolume
		volume.Decode(entResp.Entity())

		// Skip if not for this node
		if string(volume.NodeId) != c.nodeId {
			continue
		}

		// Reconcile the volume
		if err := c.reconcileVolume(ctx, &volume); err != nil {
			c.log.Error("failed to reconcile volume",
				"entity_id", volume.ID,
				"error", err,
			)
		}
	}

	return nil
}

// volumeActualStateToId maps LsvdVolumeActualState to entity.Id
func volumeActualStateToId(state storage_v1alpha.LsvdVolumeActualState) entity.Id {
	switch state {
	case storage_v1alpha.VOL_PENDING:
		return storage_v1alpha.LsvdVolumeActualStateVolPendingId
	case storage_v1alpha.VOL_CREATING:
		return storage_v1alpha.LsvdVolumeActualStateVolCreatingId
	case storage_v1alpha.VOL_READY:
		return storage_v1alpha.LsvdVolumeActualStateVolReadyId
	case storage_v1alpha.VOL_DELETING:
		return storage_v1alpha.LsvdVolumeActualStateVolDeletingId
	case storage_v1alpha.VOL_DELETED:
		return storage_v1alpha.LsvdVolumeActualStateVolDeletedId
	case storage_v1alpha.VOL_ERROR:
		return storage_v1alpha.LsvdVolumeActualStateVolErrorId
	default:
		return storage_v1alpha.LsvdVolumeActualStateVolPendingId
	}
}

// updateVolumeState updates the actual_state and optionally volume_id in the entity
func (c *VolumeController) updateVolumeState(ctx context.Context, id entity.Id, state storage_v1alpha.LsvdVolumeActualState, volumeId, errorMsg string) error {
	// Get the entity.Id for the state
	stateId := volumeActualStateToId(state)

	// Build attrs for the update - include entity ID for Patch
	attrs := []entity.Attr{
		entity.Ref(entity.DBId, id),
		entity.Ref(storage_v1alpha.LsvdVolumeActualStateId, stateId),
	}

	if volumeId != "" {
		attrs = append(attrs, entity.String(storage_v1alpha.LsvdVolumeVolumeIdId, volumeId))
	}

	if errorMsg != "" {
		attrs = append(attrs, entity.String(storage_v1alpha.LsvdVolumeErrorMessageId, errorMsg))
	}

	_, err := c.eac.Patch(ctx, attrs, 0)
	return err
}

// setVolumeError sets the volume to error state with a message
func (c *VolumeController) setVolumeError(ctx context.Context, id entity.Id, errorMsg string) {
	if err := c.updateVolumeState(ctx, id, storage_v1alpha.VOL_ERROR, "", errorMsg); err != nil {
		c.log.Warn("failed to set volume error state", "entity_id", id, "error", err)
	}
}
