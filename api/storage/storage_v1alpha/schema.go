package storage_v1alpha

import (
	"time"

	entity "miren.dev/runtime/pkg/entity"
	schema "miren.dev/runtime/pkg/entity/schema"
)

const (
	DiskCreatedById          = entity.Id("dev.miren.storage/disk.created_by")
	DiskFilesystemId         = entity.Id("dev.miren.storage/disk.filesystem")
	DiskFilesystemExt4Id     = entity.Id("dev.miren.storage/filesystem.ext4")
	DiskFilesystemXfsId      = entity.Id("dev.miren.storage/filesystem.xfs")
	DiskFilesystemBtrfsId    = entity.Id("dev.miren.storage/filesystem.btrfs")
	DiskLsvdVolumeIdId       = entity.Id("dev.miren.storage/disk.lsvd_volume_id")
	DiskModeId               = entity.Id("dev.miren.storage/disk.mode")
	DiskModeUniversalId      = entity.Id("dev.miren.storage/mode.universal")
	DiskModeAcceleratorId    = entity.Id("dev.miren.storage/mode.accelerator")
	DiskNameId               = entity.Id("dev.miren.storage/disk.name")
	DiskRemoteOnlyId         = entity.Id("dev.miren.storage/disk.remote_only")
	DiskSizeGbId             = entity.Id("dev.miren.storage/disk.size_gb")
	DiskStatusId             = entity.Id("dev.miren.storage/disk.status")
	DiskStatusProvisioningId = entity.Id("dev.miren.storage/status.provisioning")
	DiskStatusProvisionedId  = entity.Id("dev.miren.storage/status.provisioned")
	DiskStatusAttachedId     = entity.Id("dev.miren.storage/status.attached")
	DiskStatusDetachedId     = entity.Id("dev.miren.storage/status.detached")
	DiskStatusDeletingId     = entity.Id("dev.miren.storage/status.deleting")
	DiskStatusErrorId        = entity.Id("dev.miren.storage/status.error")
	DiskVolumeIdId           = entity.Id("dev.miren.storage/disk.volume_id")
)

type Disk struct {
	ID           entity.Id      `json:"id"`
	CreatedBy    entity.Id      `cbor:"created_by,omitempty" json:"created_by,omitempty"`
	Filesystem   DiskFilesystem `cbor:"filesystem,omitempty" json:"filesystem,omitempty"`
	LsvdVolumeId string         `cbor:"lsvd_volume_id,omitempty" json:"lsvd_volume_id,omitempty"`
	Mode         DiskMode       `cbor:"mode,omitempty" json:"mode,omitempty"`
	Name         string         `cbor:"name" json:"name"`
	RemoteOnly   bool           `cbor:"remote_only,omitempty" json:"remote_only,omitempty"`
	SizeGb       int64          `cbor:"size_gb" json:"size_gb"`
	Status       DiskStatus     `cbor:"status,omitempty" json:"status,omitempty"`
	VolumeId     string         `cbor:"volume_id,omitempty" json:"volume_id,omitempty"`
}

type DiskFilesystem string

const (
	EXT4  DiskFilesystem = "filesystem.ext4"
	XFS   DiskFilesystem = "filesystem.xfs"
	BTRFS DiskFilesystem = "filesystem.btrfs"
)

var diskfilesystemFromId = map[entity.Id]DiskFilesystem{DiskFilesystemExt4Id: EXT4, DiskFilesystemXfsId: XFS, DiskFilesystemBtrfsId: BTRFS}
var diskfilesystemToId = map[DiskFilesystem]entity.Id{EXT4: DiskFilesystemExt4Id, XFS: DiskFilesystemXfsId, BTRFS: DiskFilesystemBtrfsId}

type DiskMode string

const (
	UNIVERSAL   DiskMode = "mode.universal"
	ACCELERATOR DiskMode = "mode.accelerator"
)

var diskmodeFromId = map[entity.Id]DiskMode{DiskModeUniversalId: UNIVERSAL, DiskModeAcceleratorId: ACCELERATOR}
var diskmodeToId = map[DiskMode]entity.Id{UNIVERSAL: DiskModeUniversalId, ACCELERATOR: DiskModeAcceleratorId}

type DiskStatus string

const (
	PROVISIONING DiskStatus = "status.provisioning"
	PROVISIONED  DiskStatus = "status.provisioned"
	ATTACHED     DiskStatus = "status.attached"
	DETACHED     DiskStatus = "status.detached"
	DELETING     DiskStatus = "status.deleting"
	ERROR        DiskStatus = "status.error"
)

var diskstatusFromId = map[entity.Id]DiskStatus{DiskStatusProvisioningId: PROVISIONING, DiskStatusProvisionedId: PROVISIONED, DiskStatusAttachedId: ATTACHED, DiskStatusDetachedId: DETACHED, DiskStatusDeletingId: DELETING, DiskStatusErrorId: ERROR}
var diskstatusToId = map[DiskStatus]entity.Id{PROVISIONING: DiskStatusProvisioningId, PROVISIONED: DiskStatusProvisionedId, ATTACHED: DiskStatusAttachedId, DETACHED: DiskStatusDetachedId, DELETING: DiskStatusDeletingId, ERROR: DiskStatusErrorId}

func (o *Disk) Decode(e entity.AttrGetter) {
	o.ID = entity.MustGet(e, entity.DBId).Value.Id()
	if a, ok := e.Get(DiskCreatedById); ok && a.Value.Kind() == entity.KindId {
		o.CreatedBy = a.Value.Id()
	}
	if a, ok := e.Get(DiskFilesystemId); ok && a.Value.Kind() == entity.KindId {
		o.Filesystem = diskfilesystemFromId[a.Value.Id()]
	}
	if a, ok := e.Get(DiskLsvdVolumeIdId); ok && a.Value.Kind() == entity.KindString {
		o.LsvdVolumeId = a.Value.String()
	}
	if a, ok := e.Get(DiskModeId); ok && a.Value.Kind() == entity.KindId {
		o.Mode = diskmodeFromId[a.Value.Id()]
	}
	if a, ok := e.Get(DiskNameId); ok && a.Value.Kind() == entity.KindString {
		o.Name = a.Value.String()
	}
	if a, ok := e.Get(DiskRemoteOnlyId); ok && a.Value.Kind() == entity.KindBool {
		o.RemoteOnly = a.Value.Bool()
	}
	if a, ok := e.Get(DiskSizeGbId); ok && a.Value.Kind() == entity.KindInt64 {
		o.SizeGb = a.Value.Int64()
	}
	if a, ok := e.Get(DiskStatusId); ok && a.Value.Kind() == entity.KindId {
		o.Status = diskstatusFromId[a.Value.Id()]
	}
	if a, ok := e.Get(DiskVolumeIdId); ok && a.Value.Kind() == entity.KindString {
		o.VolumeId = a.Value.String()
	}
}

func (o *Disk) Is(e entity.AttrGetter) bool {
	return entity.Is(e, KindDisk)
}

func (o *Disk) ShortKind() string {
	return "disk"
}

func (o *Disk) Kind() entity.Id {
	return KindDisk
}

func (o *Disk) EntityId() entity.Id {
	return o.ID
}

func (o *Disk) Encode() (attrs []entity.Attr) {
	if !entity.Empty(o.CreatedBy) {
		attrs = append(attrs, entity.Ref(DiskCreatedById, o.CreatedBy))
	}
	if a, ok := diskfilesystemToId[o.Filesystem]; ok {
		attrs = append(attrs, entity.Ref(DiskFilesystemId, a))
	}
	if !entity.Empty(o.LsvdVolumeId) {
		attrs = append(attrs, entity.String(DiskLsvdVolumeIdId, o.LsvdVolumeId))
	}
	if a, ok := diskmodeToId[o.Mode]; ok {
		attrs = append(attrs, entity.Ref(DiskModeId, a))
	}
	if !entity.Empty(o.Name) {
		attrs = append(attrs, entity.String(DiskNameId, o.Name))
	}
	attrs = append(attrs, entity.Bool(DiskRemoteOnlyId, o.RemoteOnly))
	attrs = append(attrs, entity.Int64(DiskSizeGbId, o.SizeGb))
	if a, ok := diskstatusToId[o.Status]; ok {
		attrs = append(attrs, entity.Ref(DiskStatusId, a))
	}
	if !entity.Empty(o.VolumeId) {
		attrs = append(attrs, entity.String(DiskVolumeIdId, o.VolumeId))
	}
	attrs = append(attrs, entity.Ref(entity.EntityKind, KindDisk))
	return
}

func (o *Disk) Empty() bool {
	if !entity.Empty(o.CreatedBy) {
		return false
	}
	if o.Filesystem != "" {
		return false
	}
	if !entity.Empty(o.LsvdVolumeId) {
		return false
	}
	if o.Mode != "" {
		return false
	}
	if !entity.Empty(o.Name) {
		return false
	}
	if !entity.Empty(o.RemoteOnly) {
		return false
	}
	if !entity.Empty(o.SizeGb) {
		return false
	}
	if o.Status != "" {
		return false
	}
	if !entity.Empty(o.VolumeId) {
		return false
	}
	return true
}

func (o *Disk) InitSchema(sb *schema.SchemaBuilder) {
	sb.Ref("created_by", "dev.miren.storage/disk.created_by", schema.Doc("Application that created this disk (for tracking purposes)"), schema.Indexed, schema.Tags("dev.miren.app_ref"))
	sb.Singleton("dev.miren.storage/filesystem.ext4")
	sb.Singleton("dev.miren.storage/filesystem.xfs")
	sb.Singleton("dev.miren.storage/filesystem.btrfs")
	sb.Ref("filesystem", "dev.miren.storage/disk.filesystem", schema.Doc("Filesystem type for the disk"), schema.Choices(DiskFilesystemExt4Id, DiskFilesystemXfsId, DiskFilesystemBtrfsId))
	sb.String("lsvd_volume_id", "dev.miren.storage/disk.lsvd_volume_id", schema.Doc("LSVD backend volume identifier"), schema.Indexed)
	sb.Singleton("dev.miren.storage/mode.universal")
	sb.Singleton("dev.miren.storage/mode.accelerator")
	sb.Ref("mode", "dev.miren.storage/disk.mode", schema.Doc("Disk I/O mode"), schema.Indexed, schema.Choices(DiskModeUniversalId, DiskModeAcceleratorId))
	sb.String("name", "dev.miren.storage/disk.name", schema.Doc("Human-readable name for the disk"), schema.Required, schema.Indexed)
	sb.Bool("remote_only", "dev.miren.storage/disk.remote_only", schema.Doc("If true, disk is stored only remotely without local replica"))
	sb.Int64("size_gb", "dev.miren.storage/disk.size_gb", schema.Doc("Storage capacity in gigabytes"), schema.Required)
	sb.Singleton("dev.miren.storage/status.provisioning")
	sb.Singleton("dev.miren.storage/status.provisioned")
	sb.Singleton("dev.miren.storage/status.attached")
	sb.Singleton("dev.miren.storage/status.detached")
	sb.Singleton("dev.miren.storage/status.deleting")
	sb.Singleton("dev.miren.storage/status.error")
	sb.Ref("status", "dev.miren.storage/disk.status", schema.Doc("Current state of the disk"), schema.Indexed, schema.Choices(DiskStatusProvisioningId, DiskStatusProvisionedId, DiskStatusAttachedId, DiskStatusDetachedId, DiskStatusDeletingId, DiskStatusErrorId))
	sb.String("volume_id", "dev.miren.storage/disk.volume_id", schema.Doc("Volume identifier for universal/accelerator mode disks"), schema.Indexed)
}

const (
	DiskLeaseAcquiredAtId     = entity.Id("dev.miren.storage/disk_lease.acquired_at")
	DiskLeaseAppIdId          = entity.Id("dev.miren.storage/disk_lease.app_id")
	DiskLeaseDiskIdId         = entity.Id("dev.miren.storage/disk_lease.disk_id")
	DiskLeaseErrorMessageId   = entity.Id("dev.miren.storage/disk_lease.error_message")
	DiskLeaseMountId          = entity.Id("dev.miren.storage/disk_lease.mount")
	DiskLeaseNodeIdId         = entity.Id("dev.miren.storage/disk_lease.node_id")
	DiskLeaseSandboxIdId      = entity.Id("dev.miren.storage/disk_lease.sandbox_id")
	DiskLeaseStatusId         = entity.Id("dev.miren.storage/disk_lease.status")
	DiskLeaseStatusPendingId  = entity.Id("dev.miren.storage/status.pending")
	DiskLeaseStatusBoundId    = entity.Id("dev.miren.storage/status.bound")
	DiskLeaseStatusFailedId   = entity.Id("dev.miren.storage/status.failed")
	DiskLeaseStatusReleasedId = entity.Id("dev.miren.storage/status.released")
)

type DiskLease struct {
	ID           entity.Id       `json:"id"`
	AcquiredAt   time.Time       `cbor:"acquired_at,omitempty" json:"acquired_at,omitempty"`
	AppId        entity.Id       `cbor:"app_id,omitempty" json:"app_id,omitempty"`
	DiskId       entity.Id       `cbor:"disk_id" json:"disk_id"`
	ErrorMessage string          `cbor:"error_message,omitempty" json:"error_message,omitempty"`
	Mount        Mount           `cbor:"mount,omitempty" json:"mount,omitempty"`
	NodeId       entity.Id       `cbor:"node_id" json:"node_id"`
	SandboxId    entity.Id       `cbor:"sandbox_id,omitempty" json:"sandbox_id,omitempty"`
	Status       DiskLeaseStatus `cbor:"status,omitempty" json:"status,omitempty"`
}

type DiskLeaseStatus string

const (
	PENDING  DiskLeaseStatus = "status.pending"
	BOUND    DiskLeaseStatus = "status.bound"
	FAILED   DiskLeaseStatus = "status.failed"
	RELEASED DiskLeaseStatus = "status.released"
)

var disk_leasestatusFromId = map[entity.Id]DiskLeaseStatus{DiskLeaseStatusPendingId: PENDING, DiskLeaseStatusBoundId: BOUND, DiskLeaseStatusFailedId: FAILED, DiskLeaseStatusReleasedId: RELEASED}
var disk_leasestatusToId = map[DiskLeaseStatus]entity.Id{PENDING: DiskLeaseStatusPendingId, BOUND: DiskLeaseStatusBoundId, FAILED: DiskLeaseStatusFailedId, RELEASED: DiskLeaseStatusReleasedId}

func (o *DiskLease) Decode(e entity.AttrGetter) {
	o.ID = entity.MustGet(e, entity.DBId).Value.Id()
	if a, ok := e.Get(DiskLeaseAcquiredAtId); ok && a.Value.Kind() == entity.KindTime {
		o.AcquiredAt = a.Value.Time()
	}
	if a, ok := e.Get(DiskLeaseAppIdId); ok && a.Value.Kind() == entity.KindId {
		o.AppId = a.Value.Id()
	}
	if a, ok := e.Get(DiskLeaseDiskIdId); ok && a.Value.Kind() == entity.KindId {
		o.DiskId = a.Value.Id()
	}
	if a, ok := e.Get(DiskLeaseErrorMessageId); ok && a.Value.Kind() == entity.KindString {
		o.ErrorMessage = a.Value.String()
	}
	if a, ok := e.Get(DiskLeaseMountId); ok && a.Value.Kind() == entity.KindComponent {
		o.Mount.Decode(a.Value.Component())
	}
	if a, ok := e.Get(DiskLeaseNodeIdId); ok && a.Value.Kind() == entity.KindId {
		o.NodeId = a.Value.Id()
	}
	if a, ok := e.Get(DiskLeaseSandboxIdId); ok && a.Value.Kind() == entity.KindId {
		o.SandboxId = a.Value.Id()
	}
	if a, ok := e.Get(DiskLeaseStatusId); ok && a.Value.Kind() == entity.KindId {
		o.Status = disk_leasestatusFromId[a.Value.Id()]
	}
}

func (o *DiskLease) Is(e entity.AttrGetter) bool {
	return entity.Is(e, KindDiskLease)
}

func (o *DiskLease) ShortKind() string {
	return "disk_lease"
}

func (o *DiskLease) Kind() entity.Id {
	return KindDiskLease
}

func (o *DiskLease) EntityId() entity.Id {
	return o.ID
}

func (o *DiskLease) Encode() (attrs []entity.Attr) {
	if !entity.Empty(o.AcquiredAt) {
		attrs = append(attrs, entity.Time(DiskLeaseAcquiredAtId, o.AcquiredAt))
	}
	if !entity.Empty(o.AppId) {
		attrs = append(attrs, entity.Ref(DiskLeaseAppIdId, o.AppId))
	}
	if !entity.Empty(o.DiskId) {
		attrs = append(attrs, entity.Ref(DiskLeaseDiskIdId, o.DiskId))
	}
	if !entity.Empty(o.ErrorMessage) {
		attrs = append(attrs, entity.String(DiskLeaseErrorMessageId, o.ErrorMessage))
	}
	if !o.Mount.Empty() {
		attrs = append(attrs, entity.Component(DiskLeaseMountId, o.Mount.Encode()))
	}
	if !entity.Empty(o.NodeId) {
		attrs = append(attrs, entity.Ref(DiskLeaseNodeIdId, o.NodeId))
	}
	if !entity.Empty(o.SandboxId) {
		attrs = append(attrs, entity.Ref(DiskLeaseSandboxIdId, o.SandboxId))
	}
	if a, ok := disk_leasestatusToId[o.Status]; ok {
		attrs = append(attrs, entity.Ref(DiskLeaseStatusId, a))
	}
	attrs = append(attrs, entity.Ref(entity.EntityKind, KindDiskLease))
	return
}

func (o *DiskLease) Empty() bool {
	if !entity.Empty(o.AcquiredAt) {
		return false
	}
	if !entity.Empty(o.AppId) {
		return false
	}
	if !entity.Empty(o.DiskId) {
		return false
	}
	if !entity.Empty(o.ErrorMessage) {
		return false
	}
	if !o.Mount.Empty() {
		return false
	}
	if !entity.Empty(o.NodeId) {
		return false
	}
	if !entity.Empty(o.SandboxId) {
		return false
	}
	if o.Status != "" {
		return false
	}
	return true
}

func (o *DiskLease) InitSchema(sb *schema.SchemaBuilder) {
	sb.Time("acquired_at", "dev.miren.storage/disk_lease.acquired_at", schema.Doc("When the lease was acquired"))
	sb.Ref("app_id", "dev.miren.storage/disk_lease.app_id", schema.Doc("Reference to the application (for debugging)"), schema.Indexed, schema.Tags("dev.miren.app_ref"))
	sb.Ref("disk_id", "dev.miren.storage/disk_lease.disk_id", schema.Doc("Reference to the leased disk"), schema.Required, schema.Indexed)
	sb.String("error_message", "dev.miren.storage/disk_lease.error_message", schema.Doc("Error details if lease binding failed"))
	sb.Component("mount", "dev.miren.storage/disk_lease.mount", schema.Doc("Mount configuration for the disk"))
	(&Mount{}).InitSchema(sb.Builder("disk_lease.mount"))
	sb.Ref("node_id", "dev.miren.storage/disk_lease.node_id", schema.Doc("Node where the disk is mounted"), schema.Required)
	sb.Ref("sandbox_id", "dev.miren.storage/disk_lease.sandbox_id", schema.Doc("Reference to the sandbox using the disk"), schema.Indexed)
	sb.Singleton("dev.miren.storage/status.pending")
	sb.Singleton("dev.miren.storage/status.bound")
	sb.Singleton("dev.miren.storage/status.failed")
	sb.Singleton("dev.miren.storage/status.released")
	sb.Ref("status", "dev.miren.storage/disk_lease.status", schema.Doc("Current state of the lease"), schema.Indexed, schema.Choices(DiskLeaseStatusPendingId, DiskLeaseStatusBoundId, DiskLeaseStatusFailedId, DiskLeaseStatusReleasedId))
}

const (
	MountOptionsId  = entity.Id("dev.miren.storage/mount.options")
	MountPathId     = entity.Id("dev.miren.storage/mount.path")
	MountReadOnlyId = entity.Id("dev.miren.storage/mount.read_only")
)

type Mount struct {
	Options  string `cbor:"options,omitempty" json:"options,omitempty"`
	Path     string `cbor:"path" json:"path"`
	ReadOnly bool   `cbor:"read_only,omitempty" json:"read_only,omitempty"`
}

func (o *Mount) Decode(e entity.AttrGetter) {
	if a, ok := e.Get(MountOptionsId); ok && a.Value.Kind() == entity.KindString {
		o.Options = a.Value.String()
	}
	if a, ok := e.Get(MountPathId); ok && a.Value.Kind() == entity.KindString {
		o.Path = a.Value.String()
	}
	if a, ok := e.Get(MountReadOnlyId); ok && a.Value.Kind() == entity.KindBool {
		o.ReadOnly = a.Value.Bool()
	}
}

func (o *Mount) Encode() (attrs []entity.Attr) {
	if !entity.Empty(o.Options) {
		attrs = append(attrs, entity.String(MountOptionsId, o.Options))
	}
	if !entity.Empty(o.Path) {
		attrs = append(attrs, entity.String(MountPathId, o.Path))
	}
	attrs = append(attrs, entity.Bool(MountReadOnlyId, o.ReadOnly))
	return
}

func (o *Mount) Empty() bool {
	if !entity.Empty(o.Options) {
		return false
	}
	if !entity.Empty(o.Path) {
		return false
	}
	if !entity.Empty(o.ReadOnly) {
		return false
	}
	return true
}

func (o *Mount) InitSchema(sb *schema.SchemaBuilder) {
	sb.String("options", "dev.miren.storage/mount.options", schema.Doc("Mount options (e.g., \"rw,noatime\")"))
	sb.String("path", "dev.miren.storage/mount.path", schema.Doc("Mount path in the container"), schema.Required)
	sb.Bool("read_only", "dev.miren.storage/mount.read_only", schema.Doc("Whether the mount is read-only"))
}

const (
	DiskMountActualStateId                 = entity.Id("dev.miren.storage/disk_mount.actual_state")
	DiskMountActualStateDmPendingId        = entity.Id("dev.miren.storage/actual_state.dm_pending")
	DiskMountActualStateDmAttachingId      = entity.Id("dev.miren.storage/actual_state.dm_attaching")
	DiskMountActualStateDmAttachedId       = entity.Id("dev.miren.storage/actual_state.dm_attached")
	DiskMountActualStateDmMountingId       = entity.Id("dev.miren.storage/actual_state.dm_mounting")
	DiskMountActualStateDmMountedId        = entity.Id("dev.miren.storage/actual_state.dm_mounted")
	DiskMountActualStateDmUnmountingId     = entity.Id("dev.miren.storage/actual_state.dm_unmounting")
	DiskMountActualStateDmDetachingId      = entity.Id("dev.miren.storage/actual_state.dm_detaching")
	DiskMountActualStateDmDetachedId       = entity.Id("dev.miren.storage/actual_state.dm_detached")
	DiskMountActualStateDmErrorId          = entity.Id("dev.miren.storage/actual_state.dm_error")
	DiskMountDesiredStateId                = entity.Id("dev.miren.storage/disk_mount.desired_state")
	DiskMountDesiredStateDmWantMountedId   = entity.Id("dev.miren.storage/desired_state.dm_want_mounted")
	DiskMountDesiredStateDmWantUnmountedId = entity.Id("dev.miren.storage/desired_state.dm_want_unmounted")
	DiskMountDevicePathId                  = entity.Id("dev.miren.storage/disk_mount.device_path")
	DiskMountDiskLeaseIdId                 = entity.Id("dev.miren.storage/disk_mount.disk_lease_id")
	DiskMountErrorMessageId                = entity.Id("dev.miren.storage/disk_mount.error_message")
	DiskMountLoopDeviceId                  = entity.Id("dev.miren.storage/disk_mount.loop_device")
	DiskMountMountPathId                   = entity.Id("dev.miren.storage/disk_mount.mount_path")
	DiskMountNodeIdId                      = entity.Id("dev.miren.storage/disk_mount.node_id")
	DiskMountReadOnlyId                    = entity.Id("dev.miren.storage/disk_mount.read_only")
	DiskMountVolumeIdId                    = entity.Id("dev.miren.storage/disk_mount.volume_id")
)

type DiskMount struct {
	ID           entity.Id             `json:"id"`
	ActualState  DiskMountActualState  `cbor:"actual_state,omitempty" json:"actual_state,omitempty"`
	DesiredState DiskMountDesiredState `cbor:"desired_state,omitempty" json:"desired_state,omitempty"`
	DevicePath   string                `cbor:"device_path,omitempty" json:"device_path,omitempty"`
	DiskLeaseId  entity.Id             `cbor:"disk_lease_id,omitempty" json:"disk_lease_id,omitempty"`
	ErrorMessage string                `cbor:"error_message,omitempty" json:"error_message,omitempty"`
	LoopDevice   string                `cbor:"loop_device,omitempty" json:"loop_device,omitempty"`
	MountPath    string                `cbor:"mount_path" json:"mount_path"`
	NodeId       entity.Id             `cbor:"node_id" json:"node_id"`
	ReadOnly     bool                  `cbor:"read_only,omitempty" json:"read_only,omitempty"`
	VolumeId     entity.Id             `cbor:"volume_id" json:"volume_id"`
}

type DiskMountActualState string

const (
	DM_PENDING    DiskMountActualState = "actual_state.dm_pending"
	DM_ATTACHING  DiskMountActualState = "actual_state.dm_attaching"
	DM_ATTACHED   DiskMountActualState = "actual_state.dm_attached"
	DM_MOUNTING   DiskMountActualState = "actual_state.dm_mounting"
	DM_MOUNTED    DiskMountActualState = "actual_state.dm_mounted"
	DM_UNMOUNTING DiskMountActualState = "actual_state.dm_unmounting"
	DM_DETACHING  DiskMountActualState = "actual_state.dm_detaching"
	DM_DETACHED   DiskMountActualState = "actual_state.dm_detached"
	DM_ERROR      DiskMountActualState = "actual_state.dm_error"
)

var disk_mountactual_stateFromId = map[entity.Id]DiskMountActualState{DiskMountActualStateDmPendingId: DM_PENDING, DiskMountActualStateDmAttachingId: DM_ATTACHING, DiskMountActualStateDmAttachedId: DM_ATTACHED, DiskMountActualStateDmMountingId: DM_MOUNTING, DiskMountActualStateDmMountedId: DM_MOUNTED, DiskMountActualStateDmUnmountingId: DM_UNMOUNTING, DiskMountActualStateDmDetachingId: DM_DETACHING, DiskMountActualStateDmDetachedId: DM_DETACHED, DiskMountActualStateDmErrorId: DM_ERROR}
var disk_mountactual_stateToId = map[DiskMountActualState]entity.Id{DM_PENDING: DiskMountActualStateDmPendingId, DM_ATTACHING: DiskMountActualStateDmAttachingId, DM_ATTACHED: DiskMountActualStateDmAttachedId, DM_MOUNTING: DiskMountActualStateDmMountingId, DM_MOUNTED: DiskMountActualStateDmMountedId, DM_UNMOUNTING: DiskMountActualStateDmUnmountingId, DM_DETACHING: DiskMountActualStateDmDetachingId, DM_DETACHED: DiskMountActualStateDmDetachedId, DM_ERROR: DiskMountActualStateDmErrorId}

type DiskMountDesiredState string

const (
	DM_WANT_MOUNTED   DiskMountDesiredState = "desired_state.dm_want_mounted"
	DM_WANT_UNMOUNTED DiskMountDesiredState = "desired_state.dm_want_unmounted"
)

var disk_mountdesired_stateFromId = map[entity.Id]DiskMountDesiredState{DiskMountDesiredStateDmWantMountedId: DM_WANT_MOUNTED, DiskMountDesiredStateDmWantUnmountedId: DM_WANT_UNMOUNTED}
var disk_mountdesired_stateToId = map[DiskMountDesiredState]entity.Id{DM_WANT_MOUNTED: DiskMountDesiredStateDmWantMountedId, DM_WANT_UNMOUNTED: DiskMountDesiredStateDmWantUnmountedId}

func (o *DiskMount) Decode(e entity.AttrGetter) {
	o.ID = entity.MustGet(e, entity.DBId).Value.Id()
	if a, ok := e.Get(DiskMountActualStateId); ok && a.Value.Kind() == entity.KindId {
		o.ActualState = disk_mountactual_stateFromId[a.Value.Id()]
	}
	if a, ok := e.Get(DiskMountDesiredStateId); ok && a.Value.Kind() == entity.KindId {
		o.DesiredState = disk_mountdesired_stateFromId[a.Value.Id()]
	}
	if a, ok := e.Get(DiskMountDevicePathId); ok && a.Value.Kind() == entity.KindString {
		o.DevicePath = a.Value.String()
	}
	if a, ok := e.Get(DiskMountDiskLeaseIdId); ok && a.Value.Kind() == entity.KindId {
		o.DiskLeaseId = a.Value.Id()
	}
	if a, ok := e.Get(DiskMountErrorMessageId); ok && a.Value.Kind() == entity.KindString {
		o.ErrorMessage = a.Value.String()
	}
	if a, ok := e.Get(DiskMountLoopDeviceId); ok && a.Value.Kind() == entity.KindString {
		o.LoopDevice = a.Value.String()
	}
	if a, ok := e.Get(DiskMountMountPathId); ok && a.Value.Kind() == entity.KindString {
		o.MountPath = a.Value.String()
	}
	if a, ok := e.Get(DiskMountNodeIdId); ok && a.Value.Kind() == entity.KindId {
		o.NodeId = a.Value.Id()
	}
	if a, ok := e.Get(DiskMountReadOnlyId); ok && a.Value.Kind() == entity.KindBool {
		o.ReadOnly = a.Value.Bool()
	}
	if a, ok := e.Get(DiskMountVolumeIdId); ok && a.Value.Kind() == entity.KindId {
		o.VolumeId = a.Value.Id()
	}
}

func (o *DiskMount) Is(e entity.AttrGetter) bool {
	return entity.Is(e, KindDiskMount)
}

func (o *DiskMount) ShortKind() string {
	return "disk_mount"
}

func (o *DiskMount) Kind() entity.Id {
	return KindDiskMount
}

func (o *DiskMount) EntityId() entity.Id {
	return o.ID
}

func (o *DiskMount) Encode() (attrs []entity.Attr) {
	if a, ok := disk_mountactual_stateToId[o.ActualState]; ok {
		attrs = append(attrs, entity.Ref(DiskMountActualStateId, a))
	}
	if a, ok := disk_mountdesired_stateToId[o.DesiredState]; ok {
		attrs = append(attrs, entity.Ref(DiskMountDesiredStateId, a))
	}
	if !entity.Empty(o.DevicePath) {
		attrs = append(attrs, entity.String(DiskMountDevicePathId, o.DevicePath))
	}
	if !entity.Empty(o.DiskLeaseId) {
		attrs = append(attrs, entity.Ref(DiskMountDiskLeaseIdId, o.DiskLeaseId))
	}
	if !entity.Empty(o.ErrorMessage) {
		attrs = append(attrs, entity.String(DiskMountErrorMessageId, o.ErrorMessage))
	}
	if !entity.Empty(o.LoopDevice) {
		attrs = append(attrs, entity.String(DiskMountLoopDeviceId, o.LoopDevice))
	}
	if !entity.Empty(o.MountPath) {
		attrs = append(attrs, entity.String(DiskMountMountPathId, o.MountPath))
	}
	if !entity.Empty(o.NodeId) {
		attrs = append(attrs, entity.Ref(DiskMountNodeIdId, o.NodeId))
	}
	attrs = append(attrs, entity.Bool(DiskMountReadOnlyId, o.ReadOnly))
	if !entity.Empty(o.VolumeId) {
		attrs = append(attrs, entity.Ref(DiskMountVolumeIdId, o.VolumeId))
	}
	attrs = append(attrs, entity.Ref(entity.EntityKind, KindDiskMount))
	return
}

func (o *DiskMount) Empty() bool {
	if o.ActualState != "" {
		return false
	}
	if o.DesiredState != "" {
		return false
	}
	if !entity.Empty(o.DevicePath) {
		return false
	}
	if !entity.Empty(o.DiskLeaseId) {
		return false
	}
	if !entity.Empty(o.ErrorMessage) {
		return false
	}
	if !entity.Empty(o.LoopDevice) {
		return false
	}
	if !entity.Empty(o.MountPath) {
		return false
	}
	if !entity.Empty(o.NodeId) {
		return false
	}
	if !entity.Empty(o.ReadOnly) {
		return false
	}
	if !entity.Empty(o.VolumeId) {
		return false
	}
	return true
}

func (o *DiskMount) InitSchema(sb *schema.SchemaBuilder) {
	sb.Singleton("dev.miren.storage/actual_state.dm_pending")
	sb.Singleton("dev.miren.storage/actual_state.dm_attaching")
	sb.Singleton("dev.miren.storage/actual_state.dm_attached")
	sb.Singleton("dev.miren.storage/actual_state.dm_mounting")
	sb.Singleton("dev.miren.storage/actual_state.dm_mounted")
	sb.Singleton("dev.miren.storage/actual_state.dm_unmounting")
	sb.Singleton("dev.miren.storage/actual_state.dm_detaching")
	sb.Singleton("dev.miren.storage/actual_state.dm_detached")
	sb.Singleton("dev.miren.storage/actual_state.dm_error")
	sb.Ref("actual_state", "dev.miren.storage/disk_mount.actual_state", schema.Doc("Current state of the mount"), schema.Indexed, schema.Choices(DiskMountActualStateDmPendingId, DiskMountActualStateDmAttachingId, DiskMountActualStateDmAttachedId, DiskMountActualStateDmMountingId, DiskMountActualStateDmMountedId, DiskMountActualStateDmUnmountingId, DiskMountActualStateDmDetachingId, DiskMountActualStateDmDetachedId, DiskMountActualStateDmErrorId))
	sb.Singleton("dev.miren.storage/desired_state.dm_want_mounted")
	sb.Singleton("dev.miren.storage/desired_state.dm_want_unmounted")
	sb.Ref("desired_state", "dev.miren.storage/disk_mount.desired_state", schema.Doc("What state should this mount be in"), schema.Indexed, schema.Choices(DiskMountDesiredStateDmWantMountedId, DiskMountDesiredStateDmWantUnmountedId))
	sb.String("device_path", "dev.miren.storage/disk_mount.device_path", schema.Doc("Full path to the device node (e.g., /dev/loopN)"))
	sb.Ref("disk_lease_id", "dev.miren.storage/disk_mount.disk_lease_id", schema.Doc("Reference to the parent DiskLease entity"), schema.Indexed)
	sb.String("error_message", "dev.miren.storage/disk_mount.error_message", schema.Doc("Error details if actual_state is error"))
	sb.String("loop_device", "dev.miren.storage/disk_mount.loop_device", schema.Doc("Loop device name for universal mode"))
	sb.String("mount_path", "dev.miren.storage/disk_mount.mount_path", schema.Doc("Path where the volume should be mounted"), schema.Required)
	sb.Ref("node_id", "dev.miren.storage/disk_mount.node_id", schema.Doc("Node where this mount exists"), schema.Required, schema.Indexed)
	sb.Bool("read_only", "dev.miren.storage/disk_mount.read_only", schema.Doc("Whether the mount is read-only"))
	sb.Ref("volume_id", "dev.miren.storage/disk_mount.volume_id", schema.Doc("Reference to the disk_volume entity"), schema.Required, schema.Indexed)
}

const (
	DiskVolumeActualStateId           = entity.Id("dev.miren.storage/disk_volume.actual_state")
	DiskVolumeActualStateDvPendingId  = entity.Id("dev.miren.storage/actual_state.dv_pending")
	DiskVolumeActualStateDvCreatingId = entity.Id("dev.miren.storage/actual_state.dv_creating")
	DiskVolumeActualStateDvReadyId    = entity.Id("dev.miren.storage/actual_state.dv_ready")
	DiskVolumeActualStateDvDeletingId = entity.Id("dev.miren.storage/actual_state.dv_deleting")
	DiskVolumeActualStateDvDeletedId  = entity.Id("dev.miren.storage/actual_state.dv_deleted")
	DiskVolumeActualStateDvErrorId    = entity.Id("dev.miren.storage/actual_state.dv_error")
	DiskVolumeDesiredStateId          = entity.Id("dev.miren.storage/disk_volume.desired_state")
	DiskVolumeDesiredStateDvPresentId = entity.Id("dev.miren.storage/desired_state.dv_present")
	DiskVolumeDesiredStateDvAbsentId  = entity.Id("dev.miren.storage/desired_state.dv_absent")
	DiskVolumeDiskIdId                = entity.Id("dev.miren.storage/disk_volume.disk_id")
	DiskVolumeErrorMessageId          = entity.Id("dev.miren.storage/disk_volume.error_message")
	DiskVolumeFilesystemId            = entity.Id("dev.miren.storage/disk_volume.filesystem")
	DiskVolumeImagePathId             = entity.Id("dev.miren.storage/disk_volume.image_path")
	DiskVolumeModeId                  = entity.Id("dev.miren.storage/disk_volume.mode")
	DiskVolumeNameId                  = entity.Id("dev.miren.storage/disk_volume.name")
	DiskVolumeNodeIdId                = entity.Id("dev.miren.storage/disk_volume.node_id")
	DiskVolumeSizeGbId                = entity.Id("dev.miren.storage/disk_volume.size_gb")
	DiskVolumeVolumeIdId              = entity.Id("dev.miren.storage/disk_volume.volume_id")
)

type DiskVolume struct {
	ID           entity.Id              `json:"id"`
	ActualState  DiskVolumeActualState  `cbor:"actual_state,omitempty" json:"actual_state,omitempty"`
	DesiredState DiskVolumeDesiredState `cbor:"desired_state,omitempty" json:"desired_state,omitempty"`
	DiskId       entity.Id              `cbor:"disk_id" json:"disk_id"`
	ErrorMessage string                 `cbor:"error_message,omitempty" json:"error_message,omitempty"`
	Filesystem   string                 `cbor:"filesystem,omitempty" json:"filesystem,omitempty"`
	ImagePath    string                 `cbor:"image_path,omitempty" json:"image_path,omitempty"`
	Mode         string                 `cbor:"mode,omitempty" json:"mode,omitempty"`
	Name         string                 `cbor:"name,omitempty" json:"name,omitempty"`
	NodeId       entity.Id              `cbor:"node_id" json:"node_id"`
	SizeGb       int64                  `cbor:"size_gb" json:"size_gb"`
	VolumeId     string                 `cbor:"volume_id,omitempty" json:"volume_id,omitempty"`
}

type DiskVolumeActualState string

const (
	DV_PENDING  DiskVolumeActualState = "actual_state.dv_pending"
	DV_CREATING DiskVolumeActualState = "actual_state.dv_creating"
	DV_READY    DiskVolumeActualState = "actual_state.dv_ready"
	DV_DELETING DiskVolumeActualState = "actual_state.dv_deleting"
	DV_DELETED  DiskVolumeActualState = "actual_state.dv_deleted"
	DV_ERROR    DiskVolumeActualState = "actual_state.dv_error"
)

var disk_volumeactual_stateFromId = map[entity.Id]DiskVolumeActualState{DiskVolumeActualStateDvPendingId: DV_PENDING, DiskVolumeActualStateDvCreatingId: DV_CREATING, DiskVolumeActualStateDvReadyId: DV_READY, DiskVolumeActualStateDvDeletingId: DV_DELETING, DiskVolumeActualStateDvDeletedId: DV_DELETED, DiskVolumeActualStateDvErrorId: DV_ERROR}
var disk_volumeactual_stateToId = map[DiskVolumeActualState]entity.Id{DV_PENDING: DiskVolumeActualStateDvPendingId, DV_CREATING: DiskVolumeActualStateDvCreatingId, DV_READY: DiskVolumeActualStateDvReadyId, DV_DELETING: DiskVolumeActualStateDvDeletingId, DV_DELETED: DiskVolumeActualStateDvDeletedId, DV_ERROR: DiskVolumeActualStateDvErrorId}

type DiskVolumeDesiredState string

const (
	DV_PRESENT DiskVolumeDesiredState = "desired_state.dv_present"
	DV_ABSENT  DiskVolumeDesiredState = "desired_state.dv_absent"
)

var disk_volumedesired_stateFromId = map[entity.Id]DiskVolumeDesiredState{DiskVolumeDesiredStateDvPresentId: DV_PRESENT, DiskVolumeDesiredStateDvAbsentId: DV_ABSENT}
var disk_volumedesired_stateToId = map[DiskVolumeDesiredState]entity.Id{DV_PRESENT: DiskVolumeDesiredStateDvPresentId, DV_ABSENT: DiskVolumeDesiredStateDvAbsentId}

func (o *DiskVolume) Decode(e entity.AttrGetter) {
	o.ID = entity.MustGet(e, entity.DBId).Value.Id()
	if a, ok := e.Get(DiskVolumeActualStateId); ok && a.Value.Kind() == entity.KindId {
		o.ActualState = disk_volumeactual_stateFromId[a.Value.Id()]
	}
	if a, ok := e.Get(DiskVolumeDesiredStateId); ok && a.Value.Kind() == entity.KindId {
		o.DesiredState = disk_volumedesired_stateFromId[a.Value.Id()]
	}
	if a, ok := e.Get(DiskVolumeDiskIdId); ok && a.Value.Kind() == entity.KindId {
		o.DiskId = a.Value.Id()
	}
	if a, ok := e.Get(DiskVolumeErrorMessageId); ok && a.Value.Kind() == entity.KindString {
		o.ErrorMessage = a.Value.String()
	}
	if a, ok := e.Get(DiskVolumeFilesystemId); ok && a.Value.Kind() == entity.KindString {
		o.Filesystem = a.Value.String()
	}
	if a, ok := e.Get(DiskVolumeImagePathId); ok && a.Value.Kind() == entity.KindString {
		o.ImagePath = a.Value.String()
	}
	if a, ok := e.Get(DiskVolumeModeId); ok && a.Value.Kind() == entity.KindString {
		o.Mode = a.Value.String()
	}
	if a, ok := e.Get(DiskVolumeNameId); ok && a.Value.Kind() == entity.KindString {
		o.Name = a.Value.String()
	}
	if a, ok := e.Get(DiskVolumeNodeIdId); ok && a.Value.Kind() == entity.KindId {
		o.NodeId = a.Value.Id()
	}
	if a, ok := e.Get(DiskVolumeSizeGbId); ok && a.Value.Kind() == entity.KindInt64 {
		o.SizeGb = a.Value.Int64()
	}
	if a, ok := e.Get(DiskVolumeVolumeIdId); ok && a.Value.Kind() == entity.KindString {
		o.VolumeId = a.Value.String()
	}
}

func (o *DiskVolume) Is(e entity.AttrGetter) bool {
	return entity.Is(e, KindDiskVolume)
}

func (o *DiskVolume) ShortKind() string {
	return "disk_volume"
}

func (o *DiskVolume) Kind() entity.Id {
	return KindDiskVolume
}

func (o *DiskVolume) EntityId() entity.Id {
	return o.ID
}

func (o *DiskVolume) Encode() (attrs []entity.Attr) {
	if a, ok := disk_volumeactual_stateToId[o.ActualState]; ok {
		attrs = append(attrs, entity.Ref(DiskVolumeActualStateId, a))
	}
	if a, ok := disk_volumedesired_stateToId[o.DesiredState]; ok {
		attrs = append(attrs, entity.Ref(DiskVolumeDesiredStateId, a))
	}
	if !entity.Empty(o.DiskId) {
		attrs = append(attrs, entity.Ref(DiskVolumeDiskIdId, o.DiskId))
	}
	if !entity.Empty(o.ErrorMessage) {
		attrs = append(attrs, entity.String(DiskVolumeErrorMessageId, o.ErrorMessage))
	}
	if !entity.Empty(o.Filesystem) {
		attrs = append(attrs, entity.String(DiskVolumeFilesystemId, o.Filesystem))
	}
	if !entity.Empty(o.ImagePath) {
		attrs = append(attrs, entity.String(DiskVolumeImagePathId, o.ImagePath))
	}
	if !entity.Empty(o.Mode) {
		attrs = append(attrs, entity.String(DiskVolumeModeId, o.Mode))
	}
	if !entity.Empty(o.Name) {
		attrs = append(attrs, entity.String(DiskVolumeNameId, o.Name))
	}
	if !entity.Empty(o.NodeId) {
		attrs = append(attrs, entity.Ref(DiskVolumeNodeIdId, o.NodeId))
	}
	attrs = append(attrs, entity.Int64(DiskVolumeSizeGbId, o.SizeGb))
	if !entity.Empty(o.VolumeId) {
		attrs = append(attrs, entity.String(DiskVolumeVolumeIdId, o.VolumeId))
	}
	attrs = append(attrs, entity.Ref(entity.EntityKind, KindDiskVolume))
	return
}

func (o *DiskVolume) Empty() bool {
	if o.ActualState != "" {
		return false
	}
	if o.DesiredState != "" {
		return false
	}
	if !entity.Empty(o.DiskId) {
		return false
	}
	if !entity.Empty(o.ErrorMessage) {
		return false
	}
	if !entity.Empty(o.Filesystem) {
		return false
	}
	if !entity.Empty(o.ImagePath) {
		return false
	}
	if !entity.Empty(o.Mode) {
		return false
	}
	if !entity.Empty(o.Name) {
		return false
	}
	if !entity.Empty(o.NodeId) {
		return false
	}
	if !entity.Empty(o.SizeGb) {
		return false
	}
	if !entity.Empty(o.VolumeId) {
		return false
	}
	return true
}

func (o *DiskVolume) InitSchema(sb *schema.SchemaBuilder) {
	sb.Singleton("dev.miren.storage/actual_state.dv_pending")
	sb.Singleton("dev.miren.storage/actual_state.dv_creating")
	sb.Singleton("dev.miren.storage/actual_state.dv_ready")
	sb.Singleton("dev.miren.storage/actual_state.dv_deleting")
	sb.Singleton("dev.miren.storage/actual_state.dv_deleted")
	sb.Singleton("dev.miren.storage/actual_state.dv_error")
	sb.Ref("actual_state", "dev.miren.storage/disk_volume.actual_state", schema.Doc("Current state of the volume"), schema.Indexed, schema.Choices(DiskVolumeActualStateDvPendingId, DiskVolumeActualStateDvCreatingId, DiskVolumeActualStateDvReadyId, DiskVolumeActualStateDvDeletingId, DiskVolumeActualStateDvDeletedId, DiskVolumeActualStateDvErrorId))
	sb.Singleton("dev.miren.storage/desired_state.dv_present")
	sb.Singleton("dev.miren.storage/desired_state.dv_absent")
	sb.Ref("desired_state", "dev.miren.storage/disk_volume.desired_state", schema.Doc("What state should this volume be in"), schema.Indexed, schema.Choices(DiskVolumeDesiredStateDvPresentId, DiskVolumeDesiredStateDvAbsentId))
	sb.Ref("disk_id", "dev.miren.storage/disk_volume.disk_id", schema.Doc("Reference to the parent Disk entity"), schema.Required, schema.Indexed)
	sb.String("error_message", "dev.miren.storage/disk_volume.error_message", schema.Doc("Error details if actual_state is error"))
	sb.String("filesystem", "dev.miren.storage/disk_volume.filesystem", schema.Doc("Filesystem type (ext4, xfs, btrfs)"))
	sb.String("image_path", "dev.miren.storage/disk_volume.image_path", schema.Doc("Path to backing image file"))
	sb.String("mode", "dev.miren.storage/disk_volume.mode", schema.Doc("Disk I/O mode (universal or accelerator)"))
	sb.String("name", "dev.miren.storage/disk_volume.name", schema.Doc("Human-readable name for the volume (from parent disk)"))
	sb.Ref("node_id", "dev.miren.storage/disk_volume.node_id", schema.Doc("Node where this volume should be provisioned"), schema.Required, schema.Indexed)
	sb.Int64("size_gb", "dev.miren.storage/disk_volume.size_gb", schema.Doc("Volume size in gigabytes"), schema.Required)
	sb.String("volume_id", "dev.miren.storage/disk_volume.volume_id", schema.Doc("Volume identifier (generated during creation)"), schema.Indexed)
}

const (
	LsvdMountActualStateId                  = entity.Id("dev.miren.storage/lsvd_mount.actual_state")
	LsvdMountActualStateMntPendingId        = entity.Id("dev.miren.storage/actual_state.mnt_pending")
	LsvdMountActualStateMntAttachingId      = entity.Id("dev.miren.storage/actual_state.mnt_attaching")
	LsvdMountActualStateMntAttachedId       = entity.Id("dev.miren.storage/actual_state.mnt_attached")
	LsvdMountActualStateMntMountingId       = entity.Id("dev.miren.storage/actual_state.mnt_mounting")
	LsvdMountActualStateMntMountedId        = entity.Id("dev.miren.storage/actual_state.mnt_mounted")
	LsvdMountActualStateMntUnmountingId     = entity.Id("dev.miren.storage/actual_state.mnt_unmounting")
	LsvdMountActualStateMntDetachingId      = entity.Id("dev.miren.storage/actual_state.mnt_detaching")
	LsvdMountActualStateMntDetachedId       = entity.Id("dev.miren.storage/actual_state.mnt_detached")
	LsvdMountActualStateMntErrorId          = entity.Id("dev.miren.storage/actual_state.mnt_error")
	LsvdMountDesiredStateId                 = entity.Id("dev.miren.storage/lsvd_mount.desired_state")
	LsvdMountDesiredStateMntWantMountedId   = entity.Id("dev.miren.storage/desired_state.mnt_want_mounted")
	LsvdMountDesiredStateMntWantUnmountedId = entity.Id("dev.miren.storage/desired_state.mnt_want_unmounted")
	LsvdMountDevicePathId                   = entity.Id("dev.miren.storage/lsvd_mount.device_path")
	LsvdMountDiskLeaseIdId                  = entity.Id("dev.miren.storage/lsvd_mount.disk_lease_id")
	LsvdMountErrorMessageId                 = entity.Id("dev.miren.storage/lsvd_mount.error_message")
	LsvdMountLeaseNonceId                   = entity.Id("dev.miren.storage/lsvd_mount.lease_nonce")
	LsvdMountMountPathId                    = entity.Id("dev.miren.storage/lsvd_mount.mount_path")
	LsvdMountNbdIndexId                     = entity.Id("dev.miren.storage/lsvd_mount.nbd_index")
	LsvdMountNodeIdId                       = entity.Id("dev.miren.storage/lsvd_mount.node_id")
	LsvdMountReadOnlyId                     = entity.Id("dev.miren.storage/lsvd_mount.read_only")
	LsvdMountVolumeIdId                     = entity.Id("dev.miren.storage/lsvd_mount.volume_id")
)

type LsvdMount struct {
	ID           entity.Id             `json:"id"`
	ActualState  LsvdMountActualState  `cbor:"actual_state,omitempty" json:"actual_state,omitempty"`
	DesiredState LsvdMountDesiredState `cbor:"desired_state,omitempty" json:"desired_state,omitempty"`
	DevicePath   string                `cbor:"device_path,omitempty" json:"device_path,omitempty"`
	DiskLeaseId  entity.Id             `cbor:"disk_lease_id,omitempty" json:"disk_lease_id,omitempty"`
	ErrorMessage string                `cbor:"error_message,omitempty" json:"error_message,omitempty"`
	LeaseNonce   string                `cbor:"lease_nonce,omitempty" json:"lease_nonce,omitempty"`
	MountPath    string                `cbor:"mount_path" json:"mount_path"`
	NbdIndex     int64                 `cbor:"nbd_index,omitempty" json:"nbd_index,omitempty"`
	NodeId       entity.Id             `cbor:"node_id" json:"node_id"`
	ReadOnly     bool                  `cbor:"read_only,omitempty" json:"read_only,omitempty"`
	VolumeId     entity.Id             `cbor:"volume_id" json:"volume_id"`
}

type LsvdMountActualState string

const (
	MNT_PENDING    LsvdMountActualState = "actual_state.mnt_pending"
	MNT_ATTACHING  LsvdMountActualState = "actual_state.mnt_attaching"
	MNT_ATTACHED   LsvdMountActualState = "actual_state.mnt_attached"
	MNT_MOUNTING   LsvdMountActualState = "actual_state.mnt_mounting"
	MNT_MOUNTED    LsvdMountActualState = "actual_state.mnt_mounted"
	MNT_UNMOUNTING LsvdMountActualState = "actual_state.mnt_unmounting"
	MNT_DETACHING  LsvdMountActualState = "actual_state.mnt_detaching"
	MNT_DETACHED   LsvdMountActualState = "actual_state.mnt_detached"
	MNT_ERROR      LsvdMountActualState = "actual_state.mnt_error"
)

var lsvd_mountactual_stateFromId = map[entity.Id]LsvdMountActualState{LsvdMountActualStateMntPendingId: MNT_PENDING, LsvdMountActualStateMntAttachingId: MNT_ATTACHING, LsvdMountActualStateMntAttachedId: MNT_ATTACHED, LsvdMountActualStateMntMountingId: MNT_MOUNTING, LsvdMountActualStateMntMountedId: MNT_MOUNTED, LsvdMountActualStateMntUnmountingId: MNT_UNMOUNTING, LsvdMountActualStateMntDetachingId: MNT_DETACHING, LsvdMountActualStateMntDetachedId: MNT_DETACHED, LsvdMountActualStateMntErrorId: MNT_ERROR}
var lsvd_mountactual_stateToId = map[LsvdMountActualState]entity.Id{MNT_PENDING: LsvdMountActualStateMntPendingId, MNT_ATTACHING: LsvdMountActualStateMntAttachingId, MNT_ATTACHED: LsvdMountActualStateMntAttachedId, MNT_MOUNTING: LsvdMountActualStateMntMountingId, MNT_MOUNTED: LsvdMountActualStateMntMountedId, MNT_UNMOUNTING: LsvdMountActualStateMntUnmountingId, MNT_DETACHING: LsvdMountActualStateMntDetachingId, MNT_DETACHED: LsvdMountActualStateMntDetachedId, MNT_ERROR: LsvdMountActualStateMntErrorId}

type LsvdMountDesiredState string

const (
	MNT_WANT_MOUNTED   LsvdMountDesiredState = "desired_state.mnt_want_mounted"
	MNT_WANT_UNMOUNTED LsvdMountDesiredState = "desired_state.mnt_want_unmounted"
)

var lsvd_mountdesired_stateFromId = map[entity.Id]LsvdMountDesiredState{LsvdMountDesiredStateMntWantMountedId: MNT_WANT_MOUNTED, LsvdMountDesiredStateMntWantUnmountedId: MNT_WANT_UNMOUNTED}
var lsvd_mountdesired_stateToId = map[LsvdMountDesiredState]entity.Id{MNT_WANT_MOUNTED: LsvdMountDesiredStateMntWantMountedId, MNT_WANT_UNMOUNTED: LsvdMountDesiredStateMntWantUnmountedId}

func (o *LsvdMount) Decode(e entity.AttrGetter) {
	o.ID = entity.MustGet(e, entity.DBId).Value.Id()
	if a, ok := e.Get(LsvdMountActualStateId); ok && a.Value.Kind() == entity.KindId {
		o.ActualState = lsvd_mountactual_stateFromId[a.Value.Id()]
	}
	if a, ok := e.Get(LsvdMountDesiredStateId); ok && a.Value.Kind() == entity.KindId {
		o.DesiredState = lsvd_mountdesired_stateFromId[a.Value.Id()]
	}
	if a, ok := e.Get(LsvdMountDevicePathId); ok && a.Value.Kind() == entity.KindString {
		o.DevicePath = a.Value.String()
	}
	if a, ok := e.Get(LsvdMountDiskLeaseIdId); ok && a.Value.Kind() == entity.KindId {
		o.DiskLeaseId = a.Value.Id()
	}
	if a, ok := e.Get(LsvdMountErrorMessageId); ok && a.Value.Kind() == entity.KindString {
		o.ErrorMessage = a.Value.String()
	}
	if a, ok := e.Get(LsvdMountLeaseNonceId); ok && a.Value.Kind() == entity.KindString {
		o.LeaseNonce = a.Value.String()
	}
	if a, ok := e.Get(LsvdMountMountPathId); ok && a.Value.Kind() == entity.KindString {
		o.MountPath = a.Value.String()
	}
	if a, ok := e.Get(LsvdMountNbdIndexId); ok && a.Value.Kind() == entity.KindInt64 {
		o.NbdIndex = a.Value.Int64()
	}
	if a, ok := e.Get(LsvdMountNodeIdId); ok && a.Value.Kind() == entity.KindId {
		o.NodeId = a.Value.Id()
	}
	if a, ok := e.Get(LsvdMountReadOnlyId); ok && a.Value.Kind() == entity.KindBool {
		o.ReadOnly = a.Value.Bool()
	}
	if a, ok := e.Get(LsvdMountVolumeIdId); ok && a.Value.Kind() == entity.KindId {
		o.VolumeId = a.Value.Id()
	}
}

func (o *LsvdMount) Is(e entity.AttrGetter) bool {
	return entity.Is(e, KindLsvdMount)
}

func (o *LsvdMount) ShortKind() string {
	return "lsvd_mount"
}

func (o *LsvdMount) Kind() entity.Id {
	return KindLsvdMount
}

func (o *LsvdMount) EntityId() entity.Id {
	return o.ID
}

func (o *LsvdMount) Encode() (attrs []entity.Attr) {
	if a, ok := lsvd_mountactual_stateToId[o.ActualState]; ok {
		attrs = append(attrs, entity.Ref(LsvdMountActualStateId, a))
	}
	if a, ok := lsvd_mountdesired_stateToId[o.DesiredState]; ok {
		attrs = append(attrs, entity.Ref(LsvdMountDesiredStateId, a))
	}
	if !entity.Empty(o.DevicePath) {
		attrs = append(attrs, entity.String(LsvdMountDevicePathId, o.DevicePath))
	}
	if !entity.Empty(o.DiskLeaseId) {
		attrs = append(attrs, entity.Ref(LsvdMountDiskLeaseIdId, o.DiskLeaseId))
	}
	if !entity.Empty(o.ErrorMessage) {
		attrs = append(attrs, entity.String(LsvdMountErrorMessageId, o.ErrorMessage))
	}
	if !entity.Empty(o.LeaseNonce) {
		attrs = append(attrs, entity.String(LsvdMountLeaseNonceId, o.LeaseNonce))
	}
	if !entity.Empty(o.MountPath) {
		attrs = append(attrs, entity.String(LsvdMountMountPathId, o.MountPath))
	}
	if !entity.Empty(o.NbdIndex) {
		attrs = append(attrs, entity.Int64(LsvdMountNbdIndexId, o.NbdIndex))
	}
	if !entity.Empty(o.NodeId) {
		attrs = append(attrs, entity.Ref(LsvdMountNodeIdId, o.NodeId))
	}
	attrs = append(attrs, entity.Bool(LsvdMountReadOnlyId, o.ReadOnly))
	if !entity.Empty(o.VolumeId) {
		attrs = append(attrs, entity.Ref(LsvdMountVolumeIdId, o.VolumeId))
	}
	attrs = append(attrs, entity.Ref(entity.EntityKind, KindLsvdMount))
	return
}

func (o *LsvdMount) Empty() bool {
	if o.ActualState != "" {
		return false
	}
	if o.DesiredState != "" {
		return false
	}
	if !entity.Empty(o.DevicePath) {
		return false
	}
	if !entity.Empty(o.DiskLeaseId) {
		return false
	}
	if !entity.Empty(o.ErrorMessage) {
		return false
	}
	if !entity.Empty(o.LeaseNonce) {
		return false
	}
	if !entity.Empty(o.MountPath) {
		return false
	}
	if !entity.Empty(o.NbdIndex) {
		return false
	}
	if !entity.Empty(o.NodeId) {
		return false
	}
	if !entity.Empty(o.ReadOnly) {
		return false
	}
	if !entity.Empty(o.VolumeId) {
		return false
	}
	return true
}

func (o *LsvdMount) InitSchema(sb *schema.SchemaBuilder) {
	sb.Singleton("dev.miren.storage/actual_state.mnt_pending")
	sb.Singleton("dev.miren.storage/actual_state.mnt_attaching")
	sb.Singleton("dev.miren.storage/actual_state.mnt_attached")
	sb.Singleton("dev.miren.storage/actual_state.mnt_mounting")
	sb.Singleton("dev.miren.storage/actual_state.mnt_mounted")
	sb.Singleton("dev.miren.storage/actual_state.mnt_unmounting")
	sb.Singleton("dev.miren.storage/actual_state.mnt_detaching")
	sb.Singleton("dev.miren.storage/actual_state.mnt_detached")
	sb.Singleton("dev.miren.storage/actual_state.mnt_error")
	sb.Ref("actual_state", "dev.miren.storage/lsvd_mount.actual_state", schema.Doc("Current state of the mount (set by lsvd-server)"), schema.Indexed, schema.Choices(LsvdMountActualStateMntPendingId, LsvdMountActualStateMntAttachingId, LsvdMountActualStateMntAttachedId, LsvdMountActualStateMntMountingId, LsvdMountActualStateMntMountedId, LsvdMountActualStateMntUnmountingId, LsvdMountActualStateMntDetachingId, LsvdMountActualStateMntDetachedId, LsvdMountActualStateMntErrorId))
	sb.Singleton("dev.miren.storage/desired_state.mnt_want_mounted")
	sb.Singleton("dev.miren.storage/desired_state.mnt_want_unmounted")
	sb.Ref("desired_state", "dev.miren.storage/lsvd_mount.desired_state", schema.Doc("What state should this mount be in"), schema.Indexed, schema.Choices(LsvdMountDesiredStateMntWantMountedId, LsvdMountDesiredStateMntWantUnmountedId))
	sb.String("device_path", "dev.miren.storage/lsvd_mount.device_path", schema.Doc("Full path to the device node (set by lsvd-server)"))
	sb.Ref("disk_lease_id", "dev.miren.storage/lsvd_mount.disk_lease_id", schema.Doc("Reference to the parent DiskLease entity"), schema.Indexed)
	sb.String("error_message", "dev.miren.storage/lsvd_mount.error_message", schema.Doc("Error details if actual_state is error"))
	sb.String("lease_nonce", "dev.miren.storage/lsvd_mount.lease_nonce", schema.Doc("Volume lease nonce from remote Disk API"))
	sb.String("mount_path", "dev.miren.storage/lsvd_mount.mount_path", schema.Doc("Path where the volume should be mounted"), schema.Required)
	sb.Int64("nbd_index", "dev.miren.storage/lsvd_mount.nbd_index", schema.Doc("NBD device index (set by lsvd-server)"))
	sb.Ref("node_id", "dev.miren.storage/lsvd_mount.node_id", schema.Doc("Node where this mount exists"), schema.Required, schema.Indexed)
	sb.Bool("read_only", "dev.miren.storage/lsvd_mount.read_only", schema.Doc("Whether the mount is read-only"))
	sb.Ref("volume_id", "dev.miren.storage/lsvd_mount.volume_id", schema.Doc("Reference to the lsvd_volume entity"), schema.Required, schema.Indexed)
}

const (
	LsvdVolumeActualStateId            = entity.Id("dev.miren.storage/lsvd_volume.actual_state")
	LsvdVolumeActualStateVolPendingId  = entity.Id("dev.miren.storage/actual_state.vol_pending")
	LsvdVolumeActualStateVolCreatingId = entity.Id("dev.miren.storage/actual_state.vol_creating")
	LsvdVolumeActualStateVolReadyId    = entity.Id("dev.miren.storage/actual_state.vol_ready")
	LsvdVolumeActualStateVolDeletingId = entity.Id("dev.miren.storage/actual_state.vol_deleting")
	LsvdVolumeActualStateVolDeletedId  = entity.Id("dev.miren.storage/actual_state.vol_deleted")
	LsvdVolumeActualStateVolErrorId    = entity.Id("dev.miren.storage/actual_state.vol_error")
	LsvdVolumeDesiredStateId           = entity.Id("dev.miren.storage/lsvd_volume.desired_state")
	LsvdVolumeDesiredStateVolPresentId = entity.Id("dev.miren.storage/desired_state.vol_present")
	LsvdVolumeDesiredStateVolAbsentId  = entity.Id("dev.miren.storage/desired_state.vol_absent")
	LsvdVolumeDiskIdId                 = entity.Id("dev.miren.storage/lsvd_volume.disk_id")
	LsvdVolumeErrorMessageId           = entity.Id("dev.miren.storage/lsvd_volume.error_message")
	LsvdVolumeFilesystemId             = entity.Id("dev.miren.storage/lsvd_volume.filesystem")
	LsvdVolumeNameId                   = entity.Id("dev.miren.storage/lsvd_volume.name")
	LsvdVolumeNodeIdId                 = entity.Id("dev.miren.storage/lsvd_volume.node_id")
	LsvdVolumeRemoteOnlyId             = entity.Id("dev.miren.storage/lsvd_volume.remote_only")
	LsvdVolumeSizeGbId                 = entity.Id("dev.miren.storage/lsvd_volume.size_gb")
	LsvdVolumeVolumeIdId               = entity.Id("dev.miren.storage/lsvd_volume.volume_id")
)

type LsvdVolume struct {
	ID           entity.Id              `json:"id"`
	ActualState  LsvdVolumeActualState  `cbor:"actual_state,omitempty" json:"actual_state,omitempty"`
	DesiredState LsvdVolumeDesiredState `cbor:"desired_state,omitempty" json:"desired_state,omitempty"`
	DiskId       entity.Id              `cbor:"disk_id" json:"disk_id"`
	ErrorMessage string                 `cbor:"error_message,omitempty" json:"error_message,omitempty"`
	Filesystem   string                 `cbor:"filesystem,omitempty" json:"filesystem,omitempty"`
	Name         string                 `cbor:"name,omitempty" json:"name,omitempty"`
	NodeId       entity.Id              `cbor:"node_id" json:"node_id"`
	RemoteOnly   bool                   `cbor:"remote_only,omitempty" json:"remote_only,omitempty"`
	SizeGb       int64                  `cbor:"size_gb" json:"size_gb"`
	VolumeId     string                 `cbor:"volume_id,omitempty" json:"volume_id,omitempty"`
}

type LsvdVolumeActualState string

const (
	VOL_PENDING  LsvdVolumeActualState = "actual_state.vol_pending"
	VOL_CREATING LsvdVolumeActualState = "actual_state.vol_creating"
	VOL_READY    LsvdVolumeActualState = "actual_state.vol_ready"
	VOL_DELETING LsvdVolumeActualState = "actual_state.vol_deleting"
	VOL_DELETED  LsvdVolumeActualState = "actual_state.vol_deleted"
	VOL_ERROR    LsvdVolumeActualState = "actual_state.vol_error"
)

var lsvd_volumeactual_stateFromId = map[entity.Id]LsvdVolumeActualState{LsvdVolumeActualStateVolPendingId: VOL_PENDING, LsvdVolumeActualStateVolCreatingId: VOL_CREATING, LsvdVolumeActualStateVolReadyId: VOL_READY, LsvdVolumeActualStateVolDeletingId: VOL_DELETING, LsvdVolumeActualStateVolDeletedId: VOL_DELETED, LsvdVolumeActualStateVolErrorId: VOL_ERROR}
var lsvd_volumeactual_stateToId = map[LsvdVolumeActualState]entity.Id{VOL_PENDING: LsvdVolumeActualStateVolPendingId, VOL_CREATING: LsvdVolumeActualStateVolCreatingId, VOL_READY: LsvdVolumeActualStateVolReadyId, VOL_DELETING: LsvdVolumeActualStateVolDeletingId, VOL_DELETED: LsvdVolumeActualStateVolDeletedId, VOL_ERROR: LsvdVolumeActualStateVolErrorId}

type LsvdVolumeDesiredState string

const (
	VOL_PRESENT LsvdVolumeDesiredState = "desired_state.vol_present"
	VOL_ABSENT  LsvdVolumeDesiredState = "desired_state.vol_absent"
)

var lsvd_volumedesired_stateFromId = map[entity.Id]LsvdVolumeDesiredState{LsvdVolumeDesiredStateVolPresentId: VOL_PRESENT, LsvdVolumeDesiredStateVolAbsentId: VOL_ABSENT}
var lsvd_volumedesired_stateToId = map[LsvdVolumeDesiredState]entity.Id{VOL_PRESENT: LsvdVolumeDesiredStateVolPresentId, VOL_ABSENT: LsvdVolumeDesiredStateVolAbsentId}

func (o *LsvdVolume) Decode(e entity.AttrGetter) {
	o.ID = entity.MustGet(e, entity.DBId).Value.Id()
	if a, ok := e.Get(LsvdVolumeActualStateId); ok && a.Value.Kind() == entity.KindId {
		o.ActualState = lsvd_volumeactual_stateFromId[a.Value.Id()]
	}
	if a, ok := e.Get(LsvdVolumeDesiredStateId); ok && a.Value.Kind() == entity.KindId {
		o.DesiredState = lsvd_volumedesired_stateFromId[a.Value.Id()]
	}
	if a, ok := e.Get(LsvdVolumeDiskIdId); ok && a.Value.Kind() == entity.KindId {
		o.DiskId = a.Value.Id()
	}
	if a, ok := e.Get(LsvdVolumeErrorMessageId); ok && a.Value.Kind() == entity.KindString {
		o.ErrorMessage = a.Value.String()
	}
	if a, ok := e.Get(LsvdVolumeFilesystemId); ok && a.Value.Kind() == entity.KindString {
		o.Filesystem = a.Value.String()
	}
	if a, ok := e.Get(LsvdVolumeNameId); ok && a.Value.Kind() == entity.KindString {
		o.Name = a.Value.String()
	}
	if a, ok := e.Get(LsvdVolumeNodeIdId); ok && a.Value.Kind() == entity.KindId {
		o.NodeId = a.Value.Id()
	}
	if a, ok := e.Get(LsvdVolumeRemoteOnlyId); ok && a.Value.Kind() == entity.KindBool {
		o.RemoteOnly = a.Value.Bool()
	}
	if a, ok := e.Get(LsvdVolumeSizeGbId); ok && a.Value.Kind() == entity.KindInt64 {
		o.SizeGb = a.Value.Int64()
	}
	if a, ok := e.Get(LsvdVolumeVolumeIdId); ok && a.Value.Kind() == entity.KindString {
		o.VolumeId = a.Value.String()
	}
}

func (o *LsvdVolume) Is(e entity.AttrGetter) bool {
	return entity.Is(e, KindLsvdVolume)
}

func (o *LsvdVolume) ShortKind() string {
	return "lsvd_volume"
}

func (o *LsvdVolume) Kind() entity.Id {
	return KindLsvdVolume
}

func (o *LsvdVolume) EntityId() entity.Id {
	return o.ID
}

func (o *LsvdVolume) Encode() (attrs []entity.Attr) {
	if a, ok := lsvd_volumeactual_stateToId[o.ActualState]; ok {
		attrs = append(attrs, entity.Ref(LsvdVolumeActualStateId, a))
	}
	if a, ok := lsvd_volumedesired_stateToId[o.DesiredState]; ok {
		attrs = append(attrs, entity.Ref(LsvdVolumeDesiredStateId, a))
	}
	if !entity.Empty(o.DiskId) {
		attrs = append(attrs, entity.Ref(LsvdVolumeDiskIdId, o.DiskId))
	}
	if !entity.Empty(o.ErrorMessage) {
		attrs = append(attrs, entity.String(LsvdVolumeErrorMessageId, o.ErrorMessage))
	}
	if !entity.Empty(o.Filesystem) {
		attrs = append(attrs, entity.String(LsvdVolumeFilesystemId, o.Filesystem))
	}
	if !entity.Empty(o.Name) {
		attrs = append(attrs, entity.String(LsvdVolumeNameId, o.Name))
	}
	if !entity.Empty(o.NodeId) {
		attrs = append(attrs, entity.Ref(LsvdVolumeNodeIdId, o.NodeId))
	}
	attrs = append(attrs, entity.Bool(LsvdVolumeRemoteOnlyId, o.RemoteOnly))
	attrs = append(attrs, entity.Int64(LsvdVolumeSizeGbId, o.SizeGb))
	if !entity.Empty(o.VolumeId) {
		attrs = append(attrs, entity.String(LsvdVolumeVolumeIdId, o.VolumeId))
	}
	attrs = append(attrs, entity.Ref(entity.EntityKind, KindLsvdVolume))
	return
}

func (o *LsvdVolume) Empty() bool {
	if o.ActualState != "" {
		return false
	}
	if o.DesiredState != "" {
		return false
	}
	if !entity.Empty(o.DiskId) {
		return false
	}
	if !entity.Empty(o.ErrorMessage) {
		return false
	}
	if !entity.Empty(o.Filesystem) {
		return false
	}
	if !entity.Empty(o.Name) {
		return false
	}
	if !entity.Empty(o.NodeId) {
		return false
	}
	if !entity.Empty(o.RemoteOnly) {
		return false
	}
	if !entity.Empty(o.SizeGb) {
		return false
	}
	if !entity.Empty(o.VolumeId) {
		return false
	}
	return true
}

func (o *LsvdVolume) InitSchema(sb *schema.SchemaBuilder) {
	sb.Singleton("dev.miren.storage/actual_state.vol_pending")
	sb.Singleton("dev.miren.storage/actual_state.vol_creating")
	sb.Singleton("dev.miren.storage/actual_state.vol_ready")
	sb.Singleton("dev.miren.storage/actual_state.vol_deleting")
	sb.Singleton("dev.miren.storage/actual_state.vol_deleted")
	sb.Singleton("dev.miren.storage/actual_state.vol_error")
	sb.Ref("actual_state", "dev.miren.storage/lsvd_volume.actual_state", schema.Doc("Current state of the volume (set by lsvd-server)"), schema.Indexed, schema.Choices(LsvdVolumeActualStateVolPendingId, LsvdVolumeActualStateVolCreatingId, LsvdVolumeActualStateVolReadyId, LsvdVolumeActualStateVolDeletingId, LsvdVolumeActualStateVolDeletedId, LsvdVolumeActualStateVolErrorId))
	sb.Singleton("dev.miren.storage/desired_state.vol_present")
	sb.Singleton("dev.miren.storage/desired_state.vol_absent")
	sb.Ref("desired_state", "dev.miren.storage/lsvd_volume.desired_state", schema.Doc("What state should this volume be in"), schema.Indexed, schema.Choices(LsvdVolumeDesiredStateVolPresentId, LsvdVolumeDesiredStateVolAbsentId))
	sb.Ref("disk_id", "dev.miren.storage/lsvd_volume.disk_id", schema.Doc("Reference to the parent Disk entity"), schema.Required, schema.Indexed)
	sb.String("error_message", "dev.miren.storage/lsvd_volume.error_message", schema.Doc("Error details if actual_state is error"))
	sb.String("filesystem", "dev.miren.storage/lsvd_volume.filesystem", schema.Doc("Filesystem type (ext4, xfs, btrfs)"))
	sb.String("name", "dev.miren.storage/lsvd_volume.name", schema.Doc("Human-readable name for the volume (from parent disk)"))
	sb.Ref("node_id", "dev.miren.storage/lsvd_volume.node_id", schema.Doc("Node where this volume should be provisioned"), schema.Required, schema.Indexed)
	sb.Bool("remote_only", "dev.miren.storage/lsvd_volume.remote_only", schema.Doc("If true, use only remote storage"))
	sb.Int64("size_gb", "dev.miren.storage/lsvd_volume.size_gb", schema.Doc("Volume size in gigabytes"), schema.Required)
	sb.String("volume_id", "dev.miren.storage/lsvd_volume.volume_id", schema.Doc("The LSVD volume identifier (generated by lsvd-server)"), schema.Indexed)
}

var (
	KindDisk       = entity.Id("dev.miren.storage/kind.disk")
	KindDiskLease  = entity.Id("dev.miren.storage/kind.disk_lease")
	KindDiskMount  = entity.Id("dev.miren.storage/kind.disk_mount")
	KindDiskVolume = entity.Id("dev.miren.storage/kind.disk_volume")
	KindLsvdMount  = entity.Id("dev.miren.storage/kind.lsvd_mount")
	KindLsvdVolume = entity.Id("dev.miren.storage/kind.lsvd_volume")
	Schema         = entity.Id("dev.miren.storage/schema.v1alpha")
)

func init() {
	schema.Register("dev.miren.storage", "v1alpha", func(sb *schema.SchemaBuilder) {
		(&Disk{}).InitSchema(sb)
		(&DiskLease{}).InitSchema(sb)
		(&DiskMount{}).InitSchema(sb)
		(&DiskVolume{}).InitSchema(sb)
		(&LsvdMount{}).InitSchema(sb)
		(&LsvdVolume{}).InitSchema(sb)
	})
	schema.RegisterEncodedSchema("dev.miren.storage", "v1alpha", []byte("\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\xacZْ\xab6\x1b|\x90\xffϾo\x9c\x9cTއ\x92-\x81e\x90\xc4\x01\x990\xb9\xcdM\x96\xb7ș\x9a\xaa\xbc`\xaeS\xfa\xb4\xf0\x81\x05\x92\xed\xb9\x99B\xb8\xbb\xd1\xf2\xb9\x1b\xc9\xf3L%\x11\xec\x1dec!x\xcfd1hՓ\x9a\xb1\x86K:\xbcL\xff\xbb\xfa\xe4\x8d\xf9\xa4\xa0|h^\x80;^#̇V\xe0ߊ*A\xb8\xbc~@Uq\xd6\xd2\xe1\x8f\xf7\aN\xa7O\xe2\x1aűgD3Z\x1e\x9e\xe0Qg\xd4\xd6O\x1d;p\xfa\xbcG\xafxˆ\xa7A3a\xe9\xa8m\xe8\x94ɋh̟r$\xed\x85\r\xef\x8fS5L\x1f_\xab\xcd\xc4b\xaa\x06\xca&\xfds\xec\xa1\bf \xec\xa0\xfbj\x98>\xdd\x05\x02\x06&ውQ\xb4\xc3H\xcbQ\xb5\x17\xc1JNa$ruό\xa6\x1at\xcfe\r\x13\x12Y5\x90\x12\x8a2\x10\xa0p\x15\x9d\x84\xbf\xf9E\xf2\x91\xf5\x03icSa\x88E@4\xe4xd-\xeb\x89V}l\xa0\x80F\x98\xf7{\xbd\x83\x8e\xcd\x7fР\x80\x16\x91\aZτҬT\xb2\xb5U\xd2\xe0\x1b0ăR-H|\xb8!1\xf0_YY\x1f\x80^\xfb\x86\xa1\x1e\xb9\xd40\xa3\x1fl15ї\x01\x88\x95\xbb\x8e\xce\xea\vc}\xaf\xfaX\x0f,\xad\x80\xcfODkr<\xb1hM;\xa0\x87\x9c(k\x99\xe6\xb2\xde\xc1zȉ\xb2\xa4\xae\x874]\xafF>p%\x19\x9d>߄#T\x1b\xaeMo\"u\xbc\xa6\xf8%\x8d\xd4\x17\xcc\xea\xb2\xday\xb4\xd0kS\x81\\\xc9z|K\xda\xeeDڮ\xe7\x82\xf4O\xa51\x1ejdbc\r\xe6U\xb6\x8c\f\xccZ\xd8\xf4\xffx?,&\xd3\xc9~\x87\x11}\xbd\xa7T\x90\xe3\xbb\v\xef\x19-\x89\xb6\xa5\x8ao@\xddh.\x18\b}\xb6/\xd4u~v*w\xed\f\x11ȑUCd\xb8t\xec\xda70\xfd\xdb]:\x14j)\xd80\x90\xda~U\xc5\xf2\xd6ڍ6\xbe\xb8NN\xa8\x8b\xb4\xb3\xc1쥡\xf3\xa3\x12\x9d\x92L\xea\xf9ʭյZ\xb1V\xcb\\\xb1\xdf`\xb0\x1f\xc5\\\xeb\"u\xa1:͕\xb4\xdf\xed\xda7֦\x14\xa9\x1c\xcb\xee\x88>Y\x1f\x83\xab5/R\x9a\x96\xd73Bg/\xe3s38\xd9n\xe1\xdb9\xcc(\x02\xa9h\xf8\x82վ\x81\x8b\xe0\xab]\xfa@$=\xa8\xc9+\x9cQ\x1b'\xf3~\x15\xe7\x9a\xe73;\xa8\x8b\x8cڷs\x16\xf8\xbc\xaa\boYtE\x1d\xcc\x02\xea\x8eIj\x9c*b?ީ,\xe2\xd43\xe8\xe9\x9emz\xc8\uec9c\xe7Q\xef\xbb\x12,_\u0095n\xa9\xf1?a\x19\xbe\xd9S*\xc8Q_H[\x9a\xf1\xd8\xefs\xbb\xb8\x13]\x92\x7fNT\x946\xd2\"\x85\x82\xf9\x85\a\x9e\xa9p]\x8fvh\xcdqP\xc3\xf2\xeb\x95\xc1rІ\x8a2Di\xc4\xce\xd64\x8f5\xbc\x10\x95\x19\xbc\x90\x99\xbeæ\x9b\x19<\x8fmó\r\xf1\xbb\u070e:\xa6}z&3\x80\x05\x15\xe5E\x86\xde~\x9f\xa6\xce\xe8\xe7\xbdx\xb0\xd5D\xd9\x00\x896\x97\x93Xފ\xbfu**\xca_\x88ԡD\xdeD\x9e\x82u\x8a\x15\xe1\x9do\xbb\xde2:\xbd͕\b\x94\xdd\f\xf7\xe3\x1b\xf9\x91\x95\xc1\xdf\x1b|cm\xf3\x89\xa9\n\xa6\xe0}T,o儲\x95\xba%\x943\x06\xd9*Օv`v\x90\xf8\xc6Zj+)\xac\x14\xfc\x9d\xa7\xeb\x8c\xdak\xa1\xadĲB\xc9\xc4\xfar\x97\x9e\x0e\xd6\f\x91\xdd\x17\xd3\x03\xcf\t\x01\x10\xda\f\x01\xd8\xd8%B`\xc6d\x86\xc0_[!0+\xdd\x17\x02\\H\xedR RO\v\x1b\t\xc8F\xa0/y\xca*\x11\x16x>\trx\x0eۚ\xeb\x90\x05)\xa3\xc4``\x864\xc8azp\x1b\xfa\x9dc\xce\x18,\xe6\x0e\xe4\x98\xf3\x02-\xe6.\xe4r\x03Z\x8a\xd9\x06\r\xf9\x87\fr:\x16P}\xdd\x19\v\x9dy\xd0\"\x17~L\x99\xfa\x9aч\x1bs2\xfc\x94-\x92\x8c\x86\xc5\x18\x1f\x8b\x06,\xf5`4 \xa9\u05c8\x06$g{ \x95\fрndD\x03\x92\xba!\x1a\"\xb6\x8c\x84䁖f\xf33Y[\x9e\x9b\xfe\fg+^\xb0\xc6\x1d\xf1\x82\xe8\xf7\xc7\v\x12y,^f\xa1ؖ{\xdecXY\x97/\x1b\a[\x0etS\xc0l\xbd\xa1X\xa9\xbb\x12\xe6\xe5D\xc7\xdcm\x86\x03\x1a\x86\x99\xfd\xa7\x1c\x06\x00\xcft,\xe1\xa0,gc\x12\xa0\x86\x95\xbd1\x19\xe7\x8d\xc9X\xc2)v\xd6Fa\xc66\xfe\xc1\x99<\x8f\x85\x85\x89\x04\x10^\x98;\xad\x99ӱ$\x87\x81I\x1d\xdd_._\xb4=\x14f\xadg\xc0\x8a\xd5˚\xe5\xb0{\a\xe4a\x18\xa9c\xad\xc44\xbc\xda+\xb4\xd3K\xfc\xfap\x83\x12\x17\xa4F\xb1rF\xed\xccCr\xaf\x14\xf9\x01\xe0\x06v\xe2\x80>\xb1>I\x83M\xf0w\x0f\xe9w7\"N\xe0\xe1S\xe5\x06\xa9mZ,\xfaif\xdbb\x11覃\x9c\xad\xa4\x7f\xc4b\xcdDd\xbe\xc4\a$p\xac\xcb\xe6p\x00٘+\xef\xb3)\vCX\xe0\xe5\xbe\xf8#lk\xae\x83צ^\xc21\xb8\rO\xcfe\xa6\xec\x16/ҝv{6\xcfq~\x9bt\xce\x19kg\xcf9n\xcc\x01\xafx\t\xcb]\f\xe5\x0e\xcb\xc5\xfcW{5\xbd\xddr#__\xact\xbb\xd5-\xd8)\xab\xdbڋ;\xfe\r?i&zr\x97ib\x81\xc7M\x13\xa9\xad\x81\xcdpR\xbd.\xed\xff\x1a\xd8\xdf\xec\xf6\xfe\xe1 \xfb\x14\x1d 9\xc7-3\x04\xbf=\xa7\x0fg\xb2\x92\x00a\xf0$\xe4$\xc7\x7f\x00\x00\x00\xff\xff\x01\x00\x00\xff\xff߳ ٗ!\x00\x00"))
}
