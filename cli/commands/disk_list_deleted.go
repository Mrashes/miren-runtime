package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"miren.dev/runtime/components/diskio"
)

// DiskListDeleted lists disks that have been soft-deleted and are available
// for recovery via disk undelete.
func DiskListDeleted(ctx *Context, opts struct {
	ConfigCentric
	DataPath string `long:"data-path" description:"Path to miren data directory" default:"/var/lib/miren"`
}) error {
	diskDataPath := filepath.Join(opts.DataPath, "disk-data")
	if _, err := os.Stat(diskDataPath); err != nil {
		return fmt.Errorf("data path %s not found — this command must be run on the server", diskDataPath)
	}

	entries, err := diskio.ListDeletedVolumes(diskDataPath)
	if err != nil {
		return fmt.Errorf("listing deleted volumes: %w", err)
	}

	if len(entries) == 0 {
		ctx.Info("No deleted disks found")
		return nil
	}

	ctx.Info("Deleted disks available for recovery:")
	ctx.Info("")

	retentionDays := diskio.DefaultDeletedVolumeGCConfig().RetentionDays

	for _, e := range entries {
		meta := e.Metadata
		age := time.Since(meta.DeletedAt)
		remaining := time.Duration(retentionDays)*24*time.Hour - age

		ctx.Info("Name: %s", meta.DiskName)
		ctx.Info("  Volume ID:  %s", meta.VolumeID)
		ctx.Info("  Size:       %d GB", meta.SizeGb)
		ctx.Info("  Filesystem: %s", meta.Filesystem)
		ctx.Info("  Deleted:    %s (%s ago)", meta.DeletedAt.Format(time.RFC3339), age.Truncate(time.Minute))
		if remaining > 0 {
			ctx.Info("  Expires in: %s", remaining.Truncate(time.Minute))
		} else {
			ctx.Info("  Expires in: imminent (past retention period)")
		}
		ctx.Info("")
	}

	ctx.Info("To restore: miren disk undelete --name <disk-name>")

	return nil
}
