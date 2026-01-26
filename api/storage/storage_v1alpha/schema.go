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
)

type Disk struct {
	ID           entity.Id      `json:"id"`
	CreatedBy    entity.Id      `cbor:"created_by,omitempty" json:"created_by,omitempty"`
	Filesystem   DiskFilesystem `cbor:"filesystem,omitempty" json:"filesystem,omitempty"`
	LsvdVolumeId string         `cbor:"lsvd_volume_id,omitempty" json:"lsvd_volume_id,omitempty"`
	Name         string         `cbor:"name" json:"name"`
	RemoteOnly   bool           `cbor:"remote_only,omitempty" json:"remote_only,omitempty"`
	SizeGb       int64          `cbor:"size_gb" json:"size_gb"`
	Status       DiskStatus     `cbor:"status,omitempty" json:"status,omitempty"`
}

type DiskFilesystem string

const (
	EXT4  DiskFilesystem = "filesystem.ext4"
	XFS   DiskFilesystem = "filesystem.xfs"
	BTRFS DiskFilesystem = "filesystem.btrfs"
)

var diskfilesystemFromId = map[entity.Id]DiskFilesystem{DiskFilesystemExt4Id: EXT4, DiskFilesystemXfsId: XFS, DiskFilesystemBtrfsId: BTRFS}
var diskfilesystemToId = map[DiskFilesystem]entity.Id{EXT4: DiskFilesystemExt4Id, XFS: DiskFilesystemXfsId, BTRFS: DiskFilesystemBtrfsId}

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
	if !entity.Empty(o.Name) {
		attrs = append(attrs, entity.String(DiskNameId, o.Name))
	}
	attrs = append(attrs, entity.Bool(DiskRemoteOnlyId, o.RemoteOnly))
	attrs = append(attrs, entity.Int64(DiskSizeGbId, o.SizeGb))
	if a, ok := diskstatusToId[o.Status]; ok {
		attrs = append(attrs, entity.Ref(DiskStatusId, a))
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
	return true
}

func (o *Disk) InitSchema(sb *schema.SchemaBuilder) {
	sb.Ref("created_by", "dev.miren.storage/disk.created_by", schema.Doc("Application that created this disk (for tracking purposes)"), schema.Indexed, schema.Tags("dev.miren.app_ref"))
	sb.Singleton("dev.miren.storage/filesystem.ext4")
	sb.Singleton("dev.miren.storage/filesystem.xfs")
	sb.Singleton("dev.miren.storage/filesystem.btrfs")
	sb.Ref("filesystem", "dev.miren.storage/disk.filesystem", schema.Doc("Filesystem type for the disk"), schema.Choices(DiskFilesystemExt4Id, DiskFilesystemXfsId, DiskFilesystemBtrfsId))
	sb.String("lsvd_volume_id", "dev.miren.storage/disk.lsvd_volume_id", schema.Doc("LSVD backend volume identifier"), schema.Indexed)
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
	sb.Ref("node_id", "dev.miren.storage/lsvd_volume.node_id", schema.Doc("Node where this volume should be provisioned"), schema.Required, schema.Indexed)
	sb.Bool("remote_only", "dev.miren.storage/lsvd_volume.remote_only", schema.Doc("If true, use only remote storage"))
	sb.Int64("size_gb", "dev.miren.storage/lsvd_volume.size_gb", schema.Doc("Volume size in gigabytes"), schema.Required)
	sb.String("volume_id", "dev.miren.storage/lsvd_volume.volume_id", schema.Doc("The LSVD volume identifier (generated by lsvd-server)"), schema.Indexed)
}

var (
	KindDisk       = entity.Id("dev.miren.storage/kind.disk")
	KindDiskLease  = entity.Id("dev.miren.storage/kind.disk_lease")
	KindLsvdMount  = entity.Id("dev.miren.storage/kind.lsvd_mount")
	KindLsvdVolume = entity.Id("dev.miren.storage/kind.lsvd_volume")
	Schema         = entity.Id("dev.miren.storage/schema.v1alpha")
)

func init() {
	schema.Register("dev.miren.storage", "v1alpha", func(sb *schema.SchemaBuilder) {
		(&Disk{}).InitSchema(sb)
		(&DiskLease{}).InitSchema(sb)
		(&LsvdMount{}).InitSchema(sb)
		(&LsvdVolume{}).InitSchema(sb)
	})
	schema.RegisterEncodedSchema("dev.miren.storage", "v1alpha", []byte("\x1f\x8b\b\x00\x00\x00\x00\x00\x00\xff\xacX۲\xac&\x10\xfd\x90\xdc\xefws\xa9\xfc\x8f\x85C\xeb0\"x\x80\x998y\xcdC\x92\xca_\xe4L\xed\xaa\xfc`\x9eS4\xa0\xe8 rN\xe5\xc5\x02\\k\xd9аZ}PA\x06xE\xe1V\rL\x81\xa8\xb4\x91\x8at\x00=\x13T?\xa6w\x9e\xee|o\xefT\x94\xe9\xfe\x05\xb9\xb7g\x84\xbd\xe9\x04\xfem\xa9\x1c\b\x13\xcf\x0fh[\x06\x9c\xea\xdf_7\x8cN\x1f\xa55\xaa\x93\x02b\x80\xd6\xcd\x1d\x1fu\x89\xfa\xe6>B\xc3\xe8#Go\x19\a}\xd7\x06\x06G\x8f\xfa\x96NA\\\x87\xde^\xea\x1b\xe1WЯOS\xab\xa7\x0f\x9f\xd5\x16b5\xb5\x9a\xc2d~N=4\x82Y\b4F\xb5z\xfa8\vD\f.\xc2g;\xb3\xe0\xfaF\xeb\x9b\xe4\xd7\x01jFq&b3fg\xd3j\xa3\x98\xe8P*\x915\x94\xb2\\\xba\\\xb6\xb4D\xa4HS0H\x03\xb5\x14\xdc塏\ap%\x1b)9J\xbc\xbf#\xa1ٯPw\rһб\xd4\x13\x13\x06\x93\xf8\xde\x1e\xd3\x10s\xd5Hl};\x99\xbc\x17\x00\xa5\xa4JE\xe0h\x15\xde?\x13c\xc8\xe9\f\xc9]\xe3\x81\x01r\xa6\xc0\xc10\xd1e\xb0\x01r\xa6p\xa8\x1b \xfd\xa8\xe4\x8di&\x05\xd0\xe9\xd3]x\x84\xe2s\xdbF\x93\xd8)[\n\x13]w\x03e\x9b\xdd\xedG\xc2\xc73\xe1\xa3b\x03Q\xf7ڞLj\xd76\x15\xea|\xbak\x0eD\x83;\xe3ӻ\xe9\xe48L\xe1Q\xff\x037ȗ9\xa5\x8a\x9c^]\x99\x02Z\x13\xe3vZ<\x80i7l\x00\x14\xfa$/4\x8eᰴ\xbe\xed\x1d\x03ɉE\x8f\xc8\xd8\xf4\xec.tb\xfa\xd7Y:\xee\xb3z\x00\xadI\xe7Nڰ\x1e\x8a\xce\xdd#s\xee\xbc\xdc \xaf\u00ad\x06\xb8\xa6\xa5\xb3\x93\x1cF)@\x98\xa5\xe5s\xf5\xacVm\xd5\n3\xf6\x1bN\xf6\x83\xe7\xe8P\xa4\x92\xa3aR\xb8\xa3م\xce\xd6S\x12;ǱGb\xceΆ\xb0\xb5\xe5%\xb6\xa6\xe3) t\xb1\"\xb6tg#\xcan|\xb7\x86\x05\x9b@H:\xfbm\x17:\xf1&\xf8\"K\xd7D\xd0FNA\xe1\x12\xf5\xe3ҕ\xdfť\xde\xf7\x80F^E\xd2}\xbd1\xe0\xfd\xb6%\x8cC2\xa3\x1e\xe6\x00\xdd\b\x82Z\xa3I\x94\xc2`4\x0eqV\x80\x91\xe6\\/@\xb2i\xb9,\xb3\xdeu%,z\x98\xbe}WZ0\x85{\xfc/L\xc3W9\xa5\x8a\x9c̕\xf0\xda\xceǝg\xbe\x1aI\xa6\xe4\x1f6\bS\xbb\x92\x94\xf0\xbcX\xa0\x9a\x91\xbdm\xb9\xe8\x93&\xf3\xc4\xf2X䅤\x95\xf0<\x96\xdb\xf6\\\x10\xbf) \x0602\xe7\x92W\xc2\f`>\xc7m\x83-a\x06\xf0\xb0\x04`\xa9\xdf\x16\x87\x1b\xb8.\x84R\xee\x8c\x16\xb6w\x15s\xcc\xdf\x15\x90\x17\xf8c\xa7`D\xfb\x8b\x82\xc6\x1a\xb7l\xb0a=\x94\xdca\x7f\x8f\xf6A\xbf\x90h\xcb\xfc\x90p\x93X\xa8\xda2\xd4<\xe0\x03\x06:\xfdT,2s\xf6*\xfbj\x8e7v\x82zv\xfd>\x1eؚ\xff\xc1r\xcdV\x11\xdcuX\x0f\x1d\x94\xeaH\xeaMJu\xc1$]\x04B\x8a\x93\x13\xebねT\xa2~DRx]\x96\xeb\x12\xf5\xb7B\x9fg\x85DCk[\xf7&W/\x97nx\xfbޫ\x85\xb1\xc6Q-̇p\\\xb2\vD\xd6_@l\xfd\xf1Ӱ\x83\xf2\xb2\b\xa5\u07b6\x96\xf2\xe2d}}I|\x92D\xa0\xc2\x02\xf3g\xd6\x00\x9c\xd4[U\x98\x17\xbb\x06\x85\x15fF\"\xc7&\xe0^\xc4Ado[\xf8\x9dSP\x95\",\xf2J\xabR\x84嶍\x9f\xfa%\x15\"\x06\xf3\xf9\xe9\xa5\xcc\x00\xc6$%\bq\x92\xdeҦ/\xf69\xa4\xd1 L\xf2\xc3a\xe5\xad\v֭\x9e\x02\xe4%\"{\xe6y\xf0\xdeτ\xd5T\x8e\xbep\x0e\x96\xe2\x7f\xf3M\xafw\xf0\xa7&V:\x98١U\xed\xbd\xeay\xfe\x1b\xfc\xeb8\x88$\xfb\xcb#[\x01\xbc@\xd6\xf1\xfc\x92d]\xaf\x8fԶ\xc0^\x9f\xa52\xb5\xfb\xcd\xe7\xfe\x06\xe4\xfe\xf5\x95\xbc\x9f/\x90\xd8n\x8f\xdf\xe6\xe30K\xdc\xf9?\x00\x00\x00\xff\xff\x01\x00\x00\xff\xff\x19\xfd\xd8c\xb4\x14\x00\x00"))
}
