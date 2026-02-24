package disk

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"miren.dev/runtime/api/entityserver/entityserver_v1alpha"
	"miren.dev/runtime/api/storage/storage_v1alpha"
	"miren.dev/runtime/pkg/controller"
	"miren.dev/runtime/pkg/entity"
	"miren.dev/runtime/pkg/idgen"
)

// detectDiskMode determines which disk I/O mode to use.
func detectDiskMode() storage_v1alpha.DiskMode {
	if mode := os.Getenv("MIREN_DISK_MODE"); mode != "" {
		switch mode {
		case "universal":
			return storage_v1alpha.UNIVERSAL
		case "accelerator":
			return storage_v1alpha.ACCELERATOR
		}
	}

	// Loop devices are available on all Linux systems
	return storage_v1alpha.UNIVERSAL
}

// DiskController manages disk entities and their lifecycle.
// It uses lsvd_volume or disk_volume entities to coordinate with lsvd-server for volume operations.
type DiskController struct {
	Log *slog.Logger
	EAC *entityserver_v1alpha.EntityAccessClient

	// NodeId is the ID of this node, used for creating volume entities
	NodeId string

	// Base path for disk mounts (e.g., /var/lib/miren/disks)
	mountBasePath string

	// diskMode determines how disks are provisioned (universal or accelerator)
	diskMode storage_v1alpha.DiskMode
}

// NewDiskController creates a disk controller that uses lsvd_volume entities.
// The lsvd-server process watches these entities and performs the actual volume operations.
func NewDiskController(log *slog.Logger, eac *entityserver_v1alpha.EntityAccessClient, nodeId string) *DiskController {
	return &DiskController{
		Log:           log.With("module", "disk"),
		EAC:           eac,
		NodeId:        nodeId,
		mountBasePath: "/var/lib/miren/disks",
	}
}

// ForceLSVDMode forces the controller to use lsvd_volume entities.
// This is used by integration tests where the LSVD volume/mount ops are mocked.
func (d *DiskController) ForceLSVDMode() {
	d.diskMode = ""
}

// ForceUniversalMode forces the controller to use disk_volume entities with
// loop devices. This is used by integration tests.
func (d *DiskController) ForceUniversalMode() {
	d.diskMode = storage_v1alpha.UNIVERSAL
}

// Init initializes the disk controller
func (d *DiskController) Init(ctx context.Context) error {
	d.diskMode = detectDiskMode()
	d.Log.Info("disk controller initialized", "mode", d.diskMode)
	return nil
}

// Create handles creation of a new disk entity
func (d *DiskController) Create(ctx context.Context, disk *storage_v1alpha.Disk, meta *entity.Meta) error {
	d.Log.Info("Processing disk creation",
		"disk", disk.ID,
		"status", disk.Status)

	return d.reconcileDisk(ctx, disk, meta)
}

// Update handles updates to an existing disk entity
func (d *DiskController) Update(ctx context.Context, disk *storage_v1alpha.Disk, meta *entity.Meta) error {
	d.Log.Info("Processing disk update",
		"disk", disk.ID,
		"status", disk.Status)

	return d.reconcileDisk(ctx, disk, meta)
}

// Delete handles deletion of a disk entity
func (d *DiskController) Delete(ctx context.Context, id entity.Id, obj *storage_v1alpha.Disk) error {
	d.Log.Info("Processing disk deletion", "disk", id)
	// Deletion is handled through the DELETING status in reconcileDisk
	return nil
}

// reconcileDisk reconciles the disk state
func (d *DiskController) reconcileDisk(ctx context.Context, disk *storage_v1alpha.Disk, meta *entity.Meta) error {
	var err error

	switch disk.Status {
	case storage_v1alpha.PROVISIONED:
		// Verify the disk is actually provisioned
		err = d.handleProvisioned(ctx, disk)
	case storage_v1alpha.PROVISIONING:
		err = d.handleProvisioning(ctx, disk)
	case storage_v1alpha.DELETING:
		err = d.handleDeletion(ctx, disk)
	case storage_v1alpha.ATTACHED, storage_v1alpha.DETACHED:
		// These states are managed by disk lease controller
		return nil
	case storage_v1alpha.ERROR:
		// Error state is terminal, no action needed
		return nil
	default:
		// Unknown status, log warning
		d.Log.Warn("Unknown disk status", "disk", disk.ID, "status", disk.Status)
		return nil
	}

	if err != nil {
		return err
	}

	// Update entity attributes if any changes
	if meta != nil {
		// Ensure meta.Entity is initialized
		if meta.Entity == nil {
			meta.Entity = entity.New(disk.Encode())
		} else {
			// Caller does a diff so we can always send it back
			meta.Entity.Update(disk.Encode())
		}
	}

	return nil
}

// handleProvisioning provisions a new disk volume based on the disk mode
func (d *DiskController) handleProvisioning(ctx context.Context, disk *storage_v1alpha.Disk) error {
	// For universal mode, use disk_volume entities
	if d.diskMode == storage_v1alpha.UNIVERSAL {
		return d.handleProvisioningUniversal(ctx, disk)
	}

	// Default: use LSVD path (legacy or accelerator mode)
	return d.handleProvisioningLSVD(ctx, disk)
}

// handleProvisioningUniversal provisions a disk volume using disk_volume entities
// and loop devices (Universal Mode)
func (d *DiskController) handleProvisioningUniversal(ctx context.Context, disk *storage_v1alpha.Disk) error {
	// Check if a disk_volume entity already exists for this disk
	existingVolume, err := d.getDiskVolumeForDisk(ctx, disk.ID)
	if err != nil {
		return fmt.Errorf("error looking up existing disk_volume for disk %s: %w", disk.ID, err)
	}

	if existingVolume != nil {
		d.Log.Debug("found existing disk_volume for disk",
			"disk", disk.ID,
			"disk_volume", existingVolume.ID,
			"actual_state", existingVolume.ActualState,
			"volume_id", existingVolume.VolumeId)

		switch existingVolume.ActualState {
		case storage_v1alpha.DV_READY:
			disk.Status = storage_v1alpha.PROVISIONED
			disk.VolumeId = existingVolume.VolumeId
			disk.Mode = storage_v1alpha.UNIVERSAL
			d.Log.Info("disk provisioned via disk_volume entity",
				"disk", disk.ID,
				"volume_id", existingVolume.VolumeId)
			return nil

		case storage_v1alpha.DV_ERROR:
			d.Log.Warn("disk_volume in error state",
				"disk", disk.ID,
				"disk_volume", existingVolume.ID,
				"error", existingVolume.ErrorMessage)
			return nil

		default:
			d.Log.Debug("disk_volume still provisioning",
				"disk", disk.ID,
				"disk_volume", existingVolume.ID,
				"actual_state", existingVolume.ActualState)
			return nil
		}
	}

	// Create new disk_volume entity
	filesystem := strings.TrimPrefix(string(disk.Filesystem), "filesystem.")

	diskVolume := &storage_v1alpha.DiskVolume{
		Name:         disk.Name,
		DiskId:       disk.ID,
		SizeGb:       disk.SizeGb,
		Filesystem:   filesystem,
		Mode:         "universal",
		DesiredState: storage_v1alpha.DV_PRESENT,
		ActualState:  storage_v1alpha.DV_PENDING,
		NodeId:       entity.Id("node/" + strings.TrimPrefix(d.NodeId, "node/")),
	}

	d.Log.Info("creating disk_volume entity",
		"disk", disk.ID,
		"size_gb", disk.SizeGb,
		"filesystem", filesystem,
		"node_id", d.NodeId)

	volumeId := idgen.GenNS("disk-vol")
	createAttrs := entity.New(
		entity.DBId, entity.Id("disk_volume/"+volumeId),
		diskVolume.Encode,
	).Attrs()

	_, err = d.EAC.Create(ctx, createAttrs)
	if err != nil {
		return fmt.Errorf("failed to create disk_volume entity: %w", err)
	}

	// Set mode on disk entity so other controllers know this is universal mode
	disk.Mode = storage_v1alpha.UNIVERSAL

	d.Log.Info("created disk_volume entity, waiting for provisioning",
		"disk", disk.ID)

	return nil
}

// handleProvisioningLSVD provisions a disk volume using lsvd_volume entities (legacy path)
func (d *DiskController) handleProvisioningLSVD(ctx context.Context, disk *storage_v1alpha.Disk) error {
	// Check if an lsvd_volume entity already exists for this disk
	existingVolume, err := d.getLsvdVolumeForDisk(ctx, disk.ID)
	if err != nil {
		return fmt.Errorf("error looking up existing lsvd_volume for disk %s: %w", disk.ID, err)
	}

	if existingVolume != nil {
		d.Log.Debug("found existing lsvd_volume for disk",
			"disk", disk.ID,
			"lsvd_volume", existingVolume.ID,
			"actual_state", existingVolume.ActualState,
			"volume_id", existingVolume.VolumeId)

		switch existingVolume.ActualState {
		case storage_v1alpha.VOL_READY:
			disk.Status = storage_v1alpha.PROVISIONED
			disk.LsvdVolumeId = existingVolume.VolumeId
			d.Log.Info("disk provisioned via lsvd_volume entity",
				"disk", disk.ID,
				"volume", existingVolume.VolumeId)
			return nil

		case storage_v1alpha.VOL_ERROR:
			d.Log.Warn("lsvd_volume in error state",
				"disk", disk.ID,
				"lsvd_volume", existingVolume.ID,
				"error", existingVolume.ErrorMessage)
			return nil

		default:
			d.Log.Debug("lsvd_volume still provisioning",
				"disk", disk.ID,
				"lsvd_volume", existingVolume.ID,
				"actual_state", existingVolume.ActualState)
			return nil
		}
	}

	// Create new lsvd_volume entity
	filesystem := strings.TrimPrefix(string(disk.Filesystem), "filesystem.")

	lsvdVolume := &storage_v1alpha.LsvdVolume{
		Name:         disk.Name,
		DiskId:       disk.ID,
		SizeGb:       disk.SizeGb,
		Filesystem:   filesystem,
		RemoteOnly:   disk.RemoteOnly,
		DesiredState: storage_v1alpha.VOL_PRESENT,
		ActualState:  storage_v1alpha.VOL_PENDING,
		NodeId:       entity.Id("node/" + strings.TrimPrefix(d.NodeId, "node/")),
	}

	d.Log.Info("creating lsvd_volume entity",
		"disk", disk.ID,
		"size_gb", disk.SizeGb,
		"filesystem", filesystem,
		"remote_only", disk.RemoteOnly,
		"node_id", d.NodeId)

	volumeId := idgen.GenNS("lsvd-vol")
	createAttrs := entity.New(
		entity.DBId, entity.Id("lsvd_volume/"+volumeId),
		lsvdVolume.Encode,
	).Attrs()

	_, err = d.EAC.Create(ctx, createAttrs)
	if err != nil {
		return fmt.Errorf("failed to create lsvd_volume entity: %w", err)
	}

	d.Log.Info("created lsvd_volume entity, waiting for lsvd-server to provision",
		"disk", disk.ID)

	return nil
}

// handleProvisioned verifies a provisioned disk has a ready volume entity
func (d *DiskController) handleProvisioned(ctx context.Context, disk *storage_v1alpha.Disk) error {
	// Determine which volume system this disk uses
	if disk.VolumeId != "" || disk.Mode == storage_v1alpha.UNIVERSAL {
		return d.handleProvisionedUniversal(ctx, disk)
	}

	// Fall back to LSVD path for backward compatibility
	return d.handleProvisionedLSVD(ctx, disk)
}

func (d *DiskController) handleProvisionedUniversal(ctx context.Context, disk *storage_v1alpha.Disk) error {
	if disk.VolumeId == "" {
		d.Log.Warn("provisioned universal disk has no volume ID, re-provisioning", "disk", disk.ID)
		disk.Status = storage_v1alpha.PROVISIONING
		return d.handleProvisioning(ctx, disk)
	}

	if d.EAC == nil {
		return nil
	}

	volume, err := d.getDiskVolumeForDisk(ctx, disk.ID)
	if err != nil {
		return fmt.Errorf("error looking up disk_volume for provisioned disk %s: %w", disk.ID, err)
	}

	if volume == nil {
		d.Log.Warn("provisioned disk has no disk_volume entity, clearing volume ID",
			"disk", disk.ID,
			"volume_id", disk.VolumeId)
		disk.VolumeId = ""
		disk.Status = storage_v1alpha.PROVISIONING
		return nil
	}

	if volume.ActualState != storage_v1alpha.DV_READY {
		d.Log.Warn("disk_volume not ready for provisioned disk",
			"disk", disk.ID,
			"disk_volume", volume.ID,
			"actual_state", volume.ActualState)
		disk.Status = storage_v1alpha.PROVISIONING
		disk.VolumeId = ""
		return nil
	}

	return nil
}

func (d *DiskController) handleProvisionedLSVD(ctx context.Context, disk *storage_v1alpha.Disk) error {
	if disk.LsvdVolumeId == "" {
		d.Log.Warn("provisioned disk has no volume ID, re-provisioning", "disk", disk.ID)
		disk.Status = storage_v1alpha.PROVISIONING
		return d.handleProvisioning(ctx, disk)
	}

	// Verify via lsvd_volume entity
	volume, err := d.getLsvdVolumeForDisk(ctx, disk.ID)
	if err != nil {
		return fmt.Errorf("error looking up lsvd_volume for provisioned disk %s: %w", disk.ID, err)
	}

	if volume == nil {
		d.Log.Warn("provisioned disk has no lsvd_volume entity, clearing volume ID",
			"disk", disk.ID,
			"volume", disk.LsvdVolumeId)
		disk.LsvdVolumeId = ""
		disk.Status = storage_v1alpha.PROVISIONING
		return nil
	}

	if volume.ActualState != storage_v1alpha.VOL_READY {
		d.Log.Warn("lsvd_volume not ready for provisioned disk",
			"disk", disk.ID,
			"lsvd_volume", volume.ID,
			"actual_state", volume.ActualState)
		disk.Status = storage_v1alpha.PROVISIONING
		disk.LsvdVolumeId = ""
		return nil
	}

	return nil
}

// handleDeletion sets desired_state=absent on the volume entity
func (d *DiskController) handleDeletion(ctx context.Context, disk *storage_v1alpha.Disk) error {
	// Try universal mode first if disk has VolumeId or Mode set
	if disk.VolumeId != "" || disk.Mode == storage_v1alpha.UNIVERSAL {
		return d.handleDeletionUniversal(ctx, disk)
	}

	// Fall back to LSVD path
	return d.handleDeletionLSVD(ctx, disk)
}

func (d *DiskController) handleDeletionUniversal(ctx context.Context, disk *storage_v1alpha.Disk) error {
	volume, err := d.getDiskVolumeForDisk(ctx, disk.ID)
	if err != nil {
		d.Log.Warn("error looking up disk_volume for deletion",
			"disk", disk.ID,
			"error", err)
		return err
	}

	if volume != nil {
		if volume.ActualState == storage_v1alpha.DV_DELETED {
			d.Log.Info("disk_volume already deleted, cleaning up disk",
				"disk", disk.ID,
				"disk_volume", volume.ID)

			if _, err := d.EAC.Delete(ctx, volume.ID.String()); err != nil {
				d.Log.Warn("failed to delete disk_volume entity",
					"disk_volume", volume.ID,
					"error", err)
				return err
			}
		} else if volume.DesiredState != storage_v1alpha.DV_ABSENT {
			d.Log.Info("setting disk_volume desired_state to absent",
				"disk", disk.ID,
				"disk_volume", volume.ID)

			updateAttrs := []entity.Attr{
				entity.Ref(entity.DBId, volume.ID),
				entity.Ref(storage_v1alpha.DiskVolumeDesiredStateId, storage_v1alpha.DiskVolumeDesiredStateDvAbsentId),
			}
			if _, err := d.EAC.Patch(ctx, updateAttrs, 0); err != nil {
				d.Log.Error("failed to update disk_volume desired_state",
					"disk_volume", volume.ID,
					"error", err)
				return err
			}

			return nil
		} else {
			d.Log.Debug("disk_volume already marked for deletion",
				"disk", disk.ID,
				"disk_volume", volume.ID,
				"actual_state", volume.ActualState)
			return nil
		}
	}

	// No disk_volume or it's been deleted - delete the disk entity
	if d.EAC != nil {
		if _, err := d.EAC.Delete(ctx, disk.ID.String()); err != nil {
			d.Log.Error("failed to delete disk entity", "disk", disk.ID, "error", err)
			return err
		}
	}

	return nil
}

func (d *DiskController) handleDeletionLSVD(ctx context.Context, disk *storage_v1alpha.Disk) error {
	volume, err := d.getLsvdVolumeForDisk(ctx, disk.ID)
	if err != nil {
		d.Log.Warn("error looking up lsvd_volume for deletion",
			"disk", disk.ID,
			"error", err)
		return err
	}

	if volume != nil {
		if volume.ActualState == storage_v1alpha.VOL_DELETED {
			d.Log.Info("lsvd_volume already deleted, cleaning up disk",
				"disk", disk.ID,
				"lsvd_volume", volume.ID)

			if _, err := d.EAC.Delete(ctx, volume.ID.String()); err != nil {
				d.Log.Warn("failed to delete lsvd_volume entity",
					"lsvd_volume", volume.ID,
					"error", err)
				return err
			}
		} else if volume.DesiredState != storage_v1alpha.VOL_ABSENT {
			d.Log.Info("setting lsvd_volume desired_state to absent",
				"disk", disk.ID,
				"lsvd_volume", volume.ID)

			updateAttrs := []entity.Attr{
				entity.Ref(entity.DBId, volume.ID),
				entity.Ref(storage_v1alpha.LsvdVolumeDesiredStateId, storage_v1alpha.LsvdVolumeDesiredStateVolAbsentId),
			}
			if _, err := d.EAC.Patch(ctx, updateAttrs, 0); err != nil {
				d.Log.Error("failed to update lsvd_volume desired_state",
					"lsvd_volume", volume.ID,
					"error", err)
				return err
			}

			return nil
		} else {
			d.Log.Debug("lsvd_volume already marked for deletion",
				"disk", disk.ID,
				"lsvd_volume", volume.ID,
				"actual_state", volume.ActualState)
			return nil
		}
	}

	// No lsvd_volume or it's been deleted - delete the disk entity
	if d.EAC != nil {
		if _, err := d.EAC.Delete(ctx, disk.ID.String()); err != nil {
			d.Log.Error("failed to delete disk entity", "disk", disk.ID, "error", err)
			return err
		}
	}

	return nil
}

// getDiskVolumeForDisk finds the disk_volume entity for a disk
func (d *DiskController) getDiskVolumeForDisk(ctx context.Context, diskId entity.Id) (*storage_v1alpha.DiskVolume, error) {
	if d.EAC == nil {
		return nil, nil
	}

	indexAttr := entity.Ref(storage_v1alpha.DiskVolumeDiskIdId, diskId)

	resp, err := d.EAC.List(ctx, indexAttr)
	if err != nil {
		return nil, fmt.Errorf("failed to list disk_volume entities: %w", err)
	}

	values := resp.Values()
	if len(values) == 0 {
		return nil, nil
	}

	var volume storage_v1alpha.DiskVolume
	volume.Decode(values[0].Entity())

	return &volume, nil
}

// getLsvdVolumeForDisk finds the lsvd_volume entity for a disk
func (d *DiskController) getLsvdVolumeForDisk(ctx context.Context, diskId entity.Id) (*storage_v1alpha.LsvdVolume, error) {
	// No EAC in test mode
	if d.EAC == nil {
		return nil, nil
	}

	// Query by disk_id index
	indexAttr := entity.Ref(storage_v1alpha.LsvdVolumeDiskIdId, diskId)

	resp, err := d.EAC.List(ctx, indexAttr)
	if err != nil {
		return nil, fmt.Errorf("failed to list lsvd_volume entities: %w", err)
	}

	values := resp.Values()
	if len(values) == 0 {
		return nil, nil
	}

	// Return the first matching entity
	var volume storage_v1alpha.LsvdVolume
	volume.Decode(values[0].Entity())

	return &volume, nil
}

// Close gracefully shuts down the disk controller
func (d *DiskController) Close() error {
	d.Log.Info("Shutting down disk controller")
	return nil
}

// Start starts the disk controller
func (d *DiskController) Start(ctx context.Context) error {
	// Create reconcile controller using AdaptController
	rc := controller.NewReconcileController(
		"disk",
		d.Log,
		entity.Ref(entity.EntityKind, storage_v1alpha.KindDisk),
		d.EAC,
		controller.AdaptController(d),
		0, // No resync period
		1, // Single worker for now
	)

	return rc.Start(ctx)
}
