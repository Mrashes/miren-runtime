package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"miren.dev/runtime/api/entityserver/entityserver_v1alpha"
	"miren.dev/runtime/pkg/snapshot"
)

// DiskRestore restores a disk from a compressed snapshot file.
func DiskRestore(ctx *Context, opts struct {
	ConfigCentric
	Snapshot string `short:"s" long:"snapshot" description:"Path to snapshot file" required:"true"`
	Name     string `short:"n" long:"name" description:"Disk name to restore to (default: original name from snapshot)"`
	DataPath string `long:"data-path" description:"Path to miren data directory" default:"/var/lib/miren"`
	Force    bool   `short:"f" long:"force" description:"Overwrite existing disk image without confirmation"`
}) error {
	snapFile, err := os.Open(opts.Snapshot)
	if err != nil {
		return fmt.Errorf("opening snapshot file: %w", err)
	}
	defer snapFile.Close()

	meta, err := snapshot.ReadHeader(snapFile)
	if err != nil {
		return fmt.Errorf("reading snapshot header: %w", err)
	}

	diskName := opts.Name
	if diskName == "" {
		diskName = meta.Name
	}

	ctx.Info("Restoring from snapshot: %s", opts.Snapshot)
	ctx.Info("  Original disk:  %s", meta.Name)
	ctx.Info("  Size:           %s", formatBytes(meta.SizeBytes))
	ctx.Info("  Filesystem:     %s", meta.Filesystem)
	ctx.Info("  Created:        %s", meta.Timestamp.Format(time.RFC3339))
	ctx.Info("  Checksum:       %s", meta.Checksum)
	ctx.Info("  Target disk:    %s", diskName)

	client, err := ctx.RPCClient("entities")
	if err != nil {
		return err
	}

	eac := entityserver_v1alpha.NewEntityAccessClient(client)
	resolver := newEntityDiskResolver(eac)

	target, err := snapshot.PrepareRestore(ctx, resolver, diskName, opts.DataPath)
	if err != nil {
		return err
	}

	if _, err := os.Stat(target.ImagePath); err == nil {
		if !opts.Force {
			return fmt.Errorf("disk image already exists at %s — use --force to overwrite", target.ImagePath)
		}
		ctx.Warn("Overwriting existing disk image at %s", target.ImagePath)
	}

	ctx.Info("Restoring to: %s", target.ImagePath)

	start := time.Now()

	if err := os.MkdirAll(filepath.Dir(target.ImagePath), 0o755); err != nil {
		return fmt.Errorf("creating image directory: %w", err)
	}

	outFile, err := os.Create(target.ImagePath)
	if err != nil {
		return fmt.Errorf("creating image file: %w", err)
	}
	defer outFile.Close()

	if err := outFile.Truncate(meta.SizeBytes); err != nil {
		return fmt.Errorf("truncating image file: %w", err)
	}

	ctx.Info("Decompressing...")

	if err := snapshot.RestoreImage(outFile, snapFile, meta); err != nil {
		return err
	}

	duration := time.Since(start)

	ctx.Info("Restore complete")
	ctx.Info("  Disk:      %s", diskName)
	ctx.Info("  Image:     %s", target.ImagePath)
	ctx.Info("  Size:      %s", formatBytes(meta.SizeBytes))
	ctx.Info("  Checksum:  verified")
	ctx.Info("  Duration:  %s", duration.Truncate(time.Millisecond))

	return nil
}
