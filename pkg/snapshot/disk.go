package snapshot

import (
	"context"
	"fmt"
	"path/filepath"
)

const (
	StatusDeleting = "DELETING"
	StatusAttached = "ATTACHED"

	LeaseStatusBound = "BOUND"
)

// DiskState holds the state of a disk entity as returned by a DiskResolver.
type DiskState struct {
	ID         string
	Name       string
	Status     string
	Filesystem string
}

// VolumeState holds the state of a disk volume entity.
type VolumeState struct {
	VolumeID  string
	ImagePath string
}

// LeaseState holds the state of a disk lease entity.
type LeaseState struct {
	ID     string
	Status string
}

// DiskResolver resolves disk-related entities from the entity store.
type DiskResolver interface {
	FindDisk(ctx context.Context, name string) (*DiskState, error)
	FindVolume(ctx context.Context, diskID string) (*VolumeState, error)
	FindLeases(ctx context.Context, diskID string) ([]LeaseState, error)
}

// BackupTarget contains resolved and validated information needed to
// perform a disk backup.
type BackupTarget struct {
	Name       string
	Filesystem string
	ImagePath  string
	IsAttached bool
}

// RestoreTarget contains resolved and validated information needed to
// perform a disk restore.
type RestoreTarget struct {
	Name      string
	ImagePath string
}

// PrepareBackup resolves disk entities and validates the disk is in a
// state suitable for backup.
func PrepareBackup(ctx context.Context, resolver DiskResolver, name string, dataPath string) (*BackupTarget, error) {
	disk, err := resolver.FindDisk(ctx, name)
	if err != nil {
		return nil, err
	}

	if disk.Status == StatusDeleting {
		return nil, fmt.Errorf("disk %q is being deleted, cannot backup", name)
	}

	vol, err := resolver.FindVolume(ctx, disk.ID)
	if err != nil {
		return nil, err
	}

	return &BackupTarget{
		Name:       name,
		Filesystem: disk.Filesystem,
		ImagePath:  resolveImagePath(vol, dataPath),
		IsAttached: disk.Status == StatusAttached,
	}, nil
}

// PrepareRestore resolves disk entities and validates the disk is in a
// state suitable for restore (not deleting, no bound leases).
func PrepareRestore(ctx context.Context, resolver DiskResolver, name string, dataPath string) (*RestoreTarget, error) {
	disk, err := resolver.FindDisk(ctx, name)
	if err != nil {
		return nil, err
	}

	if disk.Status == StatusDeleting {
		return nil, fmt.Errorf("disk %q is being deleted, cannot restore", name)
	}

	leases, err := resolver.FindLeases(ctx, disk.ID)
	if err != nil {
		return nil, err
	}

	for _, lease := range leases {
		if lease.Status == LeaseStatusBound {
			return nil, fmt.Errorf("disk %q has an active lease (ID: %s), cannot restore while mounted", name, lease.ID)
		}
	}

	vol, err := resolver.FindVolume(ctx, disk.ID)
	if err != nil {
		return nil, err
	}

	return &RestoreTarget{
		Name:      name,
		ImagePath: resolveImagePath(vol, dataPath),
	}, nil
}

func resolveImagePath(vol *VolumeState, dataPath string) string {
	if vol.ImagePath != "" {
		return vol.ImagePath
	}
	return filepath.Join(dataPath, "disk-data", "volumes", vol.VolumeID, "disk.img")
}
