package server

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"strings"

	"miren.dev/runtime/api/entityserver/entityserver_v1alpha"
	"miren.dev/runtime/api/storage/storage_v1alpha"
	"miren.dev/runtime/pkg/entity"
	"miren.dev/runtime/pkg/units"
)

// DiskVolumeController watches disk_volume entities and manages sparse disk images.
// This replaces VolumeController for universal-mode disks, using loop devices
// instead of LSVD/NBD.
type DiskVolumeController struct {
	log      *slog.Logger
	dataPath string
	nodeId   string
	eac      *entityserver_v1alpha.EntityAccessClient
	state    *State
	ops      DiskVolumeOps
}

func NewDiskVolumeController(log *slog.Logger, dataPath, nodeId string, state *State, ops DiskVolumeOps) *DiskVolumeController {
	return &DiskVolumeController{
		log:      log.With("module", "disk-volume"),
		dataPath: dataPath,
		nodeId:   nodeId,
		state:    state,
		ops:      ops,
	}
}

func (c *DiskVolumeController) SetEAC(eac *entityserver_v1alpha.EntityAccessClient) {
	c.eac = eac
}

func (c *DiskVolumeController) Init(ctx context.Context) error {
	return nil
}

func (c *DiskVolumeController) Reconcile(ctx context.Context, volume *storage_v1alpha.DiskVolume, meta *entity.Meta) error {
	return c.reconcileVolume(ctx, volume)
}

func (c *DiskVolumeController) Index() entity.Attr {
	fullNodeId := "node/" + c.nodeId
	return entity.Ref(storage_v1alpha.DiskVolumeNodeIdId, entity.Id(fullNodeId))
}

func (c *DiskVolumeController) reconcileVolume(ctx context.Context, volume *storage_v1alpha.DiskVolume) error {
	entityId := string(volume.ID)
	c.log.Info("reconciling disk volume",
		"entity_id", entityId,
		"desired_state", volume.DesiredState,
		"actual_state", volume.ActualState,
	)

	switch volume.DesiredState {
	case storage_v1alpha.DV_PRESENT:
		return c.reconcileVolumePresent(ctx, volume)
	case storage_v1alpha.DV_ABSENT:
		return c.reconcileVolumeAbsent(ctx, volume)
	default:
		c.log.Warn("unknown desired state", "desired_state", volume.DesiredState)
		return nil
	}
}

func (c *DiskVolumeController) reconcileVolumePresent(ctx context.Context, volume *storage_v1alpha.DiskVolume) error {
	entityId := string(volume.ID)

	// Check if volume already exists in our state
	if existing := c.state.GetVolume(entityId); existing != nil {
		if volume.ActualState == storage_v1alpha.DV_READY {
			if existing.DiskPath != "" && !c.ops.VolumePathExists(existing.DiskPath) {
				c.log.Warn("volume directory missing, setting error state",
					"entity_id", entityId,
					"disk_path", existing.DiskPath,
				)
				c.setVolumeError(ctx, volume.ID, "volume directory missing")
				return fmt.Errorf("volume directory missing: %s", existing.DiskPath)
			}
			c.log.Debug("volume already ready", "entity_id", entityId)
			return nil
		}
		// Persisted state has a disk path and it exists on disk — reconcile entity
		if existing.DiskPath != "" && c.ops.VolumePathExists(existing.DiskPath) {
			c.log.Info("found persisted volume on disk, reconciling entity state",
				"entity_id", entityId,
				"disk_path", existing.DiskPath,
			)
			if err := c.updateVolumeState(ctx, volume.ID, storage_v1alpha.DV_READY, existing.VolumeId, ""); err != nil {
				c.log.Warn("failed to update volume state from persisted volume", "entity_id", entityId, "error", err)
			}
			return nil
		}
	}

	switch volume.ActualState {
	case storage_v1alpha.DV_PENDING:
		return c.createVolume(ctx, volume)
	case storage_v1alpha.DV_CREATING:
		c.log.Debug("volume is being created", "entity_id", entityId)
		return nil
	case storage_v1alpha.DV_READY:
		c.log.Warn("entity says DV_READY but no local state found", "entity_id", entityId)
		return nil
	case storage_v1alpha.DV_ERROR:
		c.log.Info("volume in error state, attempting recreation", "entity_id", entityId)
		return c.createVolume(ctx, volume)
	default:
		c.log.Warn("unexpected actual state for present volume", "actual_state", volume.ActualState)
		return nil
	}
}

func (c *DiskVolumeController) reconcileVolumeAbsent(ctx context.Context, volume *storage_v1alpha.DiskVolume) error {
	entityId := string(volume.ID)

	switch volume.ActualState {
	case storage_v1alpha.DV_DELETED:
		volState := c.state.GetVolume(entityId)
		if volState != nil && volState.DiskPath != "" && c.ops.VolumePathExists(volState.DiskPath) {
			c.log.Info("cleaning up local volume data", "entity_id", entityId, "disk_path", volState.DiskPath)
			if err := c.ops.RemoveVolumeDir(volState.DiskPath); err != nil {
				c.log.Warn("failed to remove volume directory", "entity_id", entityId, "error", err)
			}
		}
		c.state.DeleteVolume(entityId)
		if err := c.state.Save(); err != nil {
			c.log.Warn("failed to save state after volume deletion", "error", err)
		}
		return nil
	case storage_v1alpha.DV_DELETING:
		return nil
	default:
		return c.deleteVolume(ctx, volume)
	}
}

func (c *DiskVolumeController) createVolume(ctx context.Context, volume *storage_v1alpha.DiskVolume) error {
	entityId := string(volume.ID)

	c.log.Info("creating disk volume",
		"entity_id", entityId,
		"size_gb", volume.SizeGb,
		"filesystem", volume.Filesystem,
	)

	if err := c.updateVolumeState(ctx, volume.ID, storage_v1alpha.DV_CREATING, "", ""); err != nil {
		c.log.Warn("failed to update volume state to creating", "error", err)
	}

	// Create volume directory
	volumePath := c.getVolumePath(entityId)
	if err := c.ops.CreateVolumeDir(volumePath); err != nil {
		c.setVolumeError(ctx, volume.ID, fmt.Sprintf("failed to create volume directory: %v", err))
		return fmt.Errorf("failed to create volume directory: %w", err)
	}

	// Create sparse disk image
	imagePath := filepath.Join(volumePath, "disk.img")
	sizeBytes := units.GigaBytes(volume.SizeGb).Bytes().Int64()
	if err := c.ops.CreateDiskImage(imagePath, sizeBytes); err != nil {
		c.setVolumeError(ctx, volume.ID, fmt.Sprintf("failed to create disk image: %v", err))
		return fmt.Errorf("failed to create disk image: %w", err)
	}

	// Use the entity ID suffix as the volume ID
	volumeId := entityId
	if idx := strings.LastIndex(entityId, "/"); idx != -1 {
		volumeId = entityId[idx+1:]
	}

	// Update state
	c.state.SetVolume(entityId, &VolumeState{
		EntityId:   entityId,
		VolumeId:   volumeId,
		Name:       volume.Name,
		DiskPath:   volumePath,
		SizeBytes:  sizeBytes,
		Filesystem: volume.Filesystem,
	})

	if err := c.state.Save(); err != nil {
		c.log.Warn("failed to save state after volume creation", "error", err)
	}

	c.log.Info("disk volume created",
		"entity_id", entityId,
		"volume_id", volumeId,
		"image_path", imagePath,
	)

	if err := c.updateVolumeState(ctx, volume.ID, storage_v1alpha.DV_READY, volumeId, ""); err != nil {
		c.log.Warn("failed to update volume state to ready", "error", err)
	}

	// Also update the image_path in the entity
	if c.eac != nil {
		attrs := []entity.Attr{
			entity.Ref(entity.DBId, volume.ID),
			entity.String(storage_v1alpha.DiskVolumeImagePathId, imagePath),
		}
		if _, err := c.eac.Patch(ctx, attrs, 0); err != nil {
			c.log.Warn("failed to update image_path in entity", "error", err)
		}
	}

	return nil
}

func (c *DiskVolumeController) deleteVolume(ctx context.Context, volume *storage_v1alpha.DiskVolume) error {
	entityId := string(volume.ID)

	c.log.Info("deleting disk volume", "entity_id", entityId)

	if err := c.updateVolumeState(ctx, volume.ID, storage_v1alpha.DV_DELETING, "", ""); err != nil {
		c.log.Warn("failed to update volume state to deleting", "error", err)
	}

	volState := c.state.GetVolume(entityId)
	if volState == nil {
		c.log.Warn("volume not found in state", "entity_id", entityId)
		if err := c.updateVolumeState(ctx, volume.ID, storage_v1alpha.DV_DELETED, "", ""); err != nil {
			c.log.Warn("failed to update volume state to deleted", "error", err)
		}
		return nil
	}

	if volState.DiskPath != "" {
		if err := c.ops.RemoveVolumeDir(volState.DiskPath); err != nil {
			c.log.Warn("failed to remove volume directory", "path", volState.DiskPath, "error", err)
		}
	}

	c.state.DeleteVolume(entityId)
	if err := c.state.Save(); err != nil {
		c.log.Warn("failed to save state after volume deletion", "error", err)
	}

	c.log.Info("disk volume deleted", "entity_id", entityId)

	if err := c.updateVolumeState(ctx, volume.ID, storage_v1alpha.DV_DELETED, "", ""); err != nil {
		c.log.Warn("failed to update volume state to deleted", "error", err)
	}

	return nil
}

func (c *DiskVolumeController) getVolumePath(volumeEntityId string) string {
	dirName := volumeEntityId
	if idx := strings.LastIndex(volumeEntityId, "/"); idx != -1 {
		dirName = volumeEntityId[idx+1:]
	}
	return filepath.Join(c.dataPath, "volumes", dirName)
}

func diskVolumeActualStateToId(state storage_v1alpha.DiskVolumeActualState) entity.Id {
	switch state {
	case storage_v1alpha.DV_PENDING:
		return storage_v1alpha.DiskVolumeActualStateDvPendingId
	case storage_v1alpha.DV_CREATING:
		return storage_v1alpha.DiskVolumeActualStateDvCreatingId
	case storage_v1alpha.DV_READY:
		return storage_v1alpha.DiskVolumeActualStateDvReadyId
	case storage_v1alpha.DV_DELETING:
		return storage_v1alpha.DiskVolumeActualStateDvDeletingId
	case storage_v1alpha.DV_DELETED:
		return storage_v1alpha.DiskVolumeActualStateDvDeletedId
	case storage_v1alpha.DV_ERROR:
		return storage_v1alpha.DiskVolumeActualStateDvErrorId
	default:
		return storage_v1alpha.DiskVolumeActualStateDvPendingId
	}
}

func (c *DiskVolumeController) updateVolumeState(ctx context.Context, id entity.Id, state storage_v1alpha.DiskVolumeActualState, volumeId, errorMsg string) error {
	if c.eac == nil {
		return nil
	}

	stateId := diskVolumeActualStateToId(state)

	attrs := []entity.Attr{
		entity.Ref(entity.DBId, id),
		entity.Ref(storage_v1alpha.DiskVolumeActualStateId, stateId),
	}

	if volumeId != "" {
		attrs = append(attrs, entity.String(storage_v1alpha.DiskVolumeVolumeIdId, volumeId))
	}

	attrs = append(attrs, entity.String(storage_v1alpha.DiskVolumeErrorMessageId, errorMsg))

	_, err := c.eac.Patch(ctx, attrs, 0)
	return err
}

func (c *DiskVolumeController) setVolumeError(ctx context.Context, id entity.Id, errorMsg string) {
	if err := c.updateVolumeState(ctx, id, storage_v1alpha.DV_ERROR, "", errorMsg); err != nil {
		c.log.Warn("failed to set volume error state", "entity_id", id, "error", err)
	}
}

// ReconcileWithEntities reconciles local state with entity server
func (c *DiskVolumeController) ReconcileWithEntities(ctx context.Context) error {
	fullNodeId := "node/" + c.nodeId
	nodeIdRef := entity.Id(fullNodeId)
	indexAttr := entity.Ref(storage_v1alpha.DiskVolumeNodeIdId, nodeIdRef)

	resp, err := c.eac.List(ctx, indexAttr)
	if err != nil {
		return fmt.Errorf("failed to list disk_volume entities: %w", err)
	}

	values := resp.Values()

	entityIds := make(map[string]struct{}, len(values))

	for _, entResp := range values {
		var volume storage_v1alpha.DiskVolume
		volume.Decode(entResp.Entity())

		entityIds[string(volume.ID)] = struct{}{}

		if string(volume.NodeId) != fullNodeId {
			continue
		}

		if err := c.reconcileVolume(ctx, &volume); err != nil {
			c.log.Error("failed to reconcile disk volume",
				"entity_id", volume.ID,
				"error", err,
			)
		}
	}

	// Clean up orphaned volumes
	orphanCleaned := false
	for _, volState := range c.state.ListVolumes() {
		// Only clean up disk_volume entries (not lsvd_volume entries)
		if !strings.HasPrefix(volState.EntityId, "disk_volume/") {
			continue
		}
		if _, exists := entityIds[volState.EntityId]; exists {
			continue
		}

		c.log.Info("cleaning up orphaned disk volume", "entity_id", volState.EntityId)

		if volState.DiskPath != "" {
			if err := c.ops.RemoveVolumeDir(volState.DiskPath); err != nil {
				c.log.Warn("failed to remove orphaned volume directory", "entity_id", volState.EntityId, "error", err)
			}
		}

		c.state.DeleteVolume(volState.EntityId)
		orphanCleaned = true
	}

	if orphanCleaned {
		if err := c.state.Save(); err != nil {
			c.log.Warn("failed to save state after orphan cleanup", "error", err)
		}
	}

	return nil
}
