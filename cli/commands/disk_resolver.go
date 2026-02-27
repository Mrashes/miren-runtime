package commands

import (
	"context"
	"fmt"

	"miren.dev/runtime/api/entityserver/entityserver_v1alpha"
	"miren.dev/runtime/api/storage/storage_v1alpha"
	"miren.dev/runtime/pkg/entity"
	"miren.dev/runtime/pkg/snapshot"
)

// entityDiskResolver implements snapshot.DiskResolver using the entity
// access RPC client.
type entityDiskResolver struct {
	eac *entityserver_v1alpha.EntityAccessClient
}

func newEntityDiskResolver(eac *entityserver_v1alpha.EntityAccessClient) *entityDiskResolver {
	return &entityDiskResolver{eac: eac}
}

func (r *entityDiskResolver) FindDisk(ctx context.Context, name string) (*snapshot.DiskState, error) {
	ref := entity.Ref(entity.EntityKind, storage_v1alpha.KindDisk)
	results, err := r.eac.List(ctx, ref)
	if err != nil {
		return nil, fmt.Errorf("listing disks: %w", err)
	}

	for _, e := range results.Values() {
		var disk storage_v1alpha.Disk
		disk.Decode(e.Entity())
		if disk.Name == name {
			return &snapshot.DiskState{
				ID:         string(disk.ID),
				Name:       disk.Name,
				Status:     string(disk.Status),
				Filesystem: string(disk.Filesystem),
			}, nil
		}
	}

	return nil, fmt.Errorf("disk %q not found", name)
}

func (r *entityDiskResolver) FindVolume(ctx context.Context, diskID string) (*snapshot.VolumeState, error) {
	resp, err := r.eac.List(ctx, entity.Ref(storage_v1alpha.DiskVolumeDiskIdId, entity.Id(diskID)))
	if err != nil {
		return nil, fmt.Errorf("listing disk volumes: %w", err)
	}

	values := resp.Values()
	if len(values) == 0 {
		return nil, fmt.Errorf("no disk volume found for disk %s", diskID)
	}

	var vol storage_v1alpha.DiskVolume
	vol.Decode(values[0].Entity())
	return &snapshot.VolumeState{
		VolumeID:  vol.VolumeId,
		ImagePath: vol.ImagePath,
	}, nil
}

func (r *entityDiskResolver) FindLeases(ctx context.Context, diskID string) ([]snapshot.LeaseState, error) {
	resp, err := r.eac.List(ctx, entity.Ref(storage_v1alpha.DiskLeaseDiskIdId, entity.Id(diskID)))
	if err != nil {
		return nil, fmt.Errorf("listing disk leases: %w", err)
	}

	var leases []snapshot.LeaseState
	for _, e := range resp.Values() {
		var lease storage_v1alpha.DiskLease
		lease.Decode(e.Entity())
		leases = append(leases, snapshot.LeaseState{
			ID:     string(lease.ID),
			Status: string(lease.Status),
		})
	}

	return leases, nil
}
