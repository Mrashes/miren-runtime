package server

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"golang.org/x/sys/unix"

	"miren.dev/runtime/lsvd"
	"miren.dev/runtime/lsvd/pkg/nbd"
	"miren.dev/runtime/lsvd/pkg/nbdnl"
)

// realMountOps implements MountOps with real OS operations
type realMountOps struct {
	log *slog.Logger
}

// NewRealMountOps creates a MountOps that performs real OS operations
func NewRealMountOps(log *slog.Logger) MountOps {
	return &realMountOps{log: log}
}

func (r *realMountOps) CreateDir(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (r *realMountOps) RemoveFile(path string) error {
	return os.Remove(path)
}

func (r *realMountOps) NBDLoopback(ctx context.Context, sizeBytes uint64) (uint32, net.Conn, *os.File, func() error, error) {
	return nbdnl.Loopback(ctx, sizeBytes, nbdnl.IndexAny)
}

func (r *realMountOps) NBDStatus(idx uint32) error {
	_, err := nbdnl.Status(idx)
	return err
}

func (r *realMountOps) NBDDisconnect(idx uint32) error {
	return nbdnl.Disconnect(idx)
}

func (r *realMountOps) CreateDeviceNode(path string, nbdIndex uint32) error {
	// Remove stale device if exists
	os.Remove(path)

	// Get NBD partition range (default: 16)
	nbdRng := uint32(16)

	devNum := int(unix.Mkdev(43, nbdIndex*nbdRng))
	if err := unix.Mknod(path, unix.S_IFBLK|0660, devNum); err != nil && !os.IsExist(err) {
		return fmt.Errorf("failed to create device node: %w", err)
	}

	return nil
}

func (r *realMountOps) Mount(device, mountPath, filesystem string, readOnly bool) error {
	var flags uintptr
	if readOnly {
		flags |= syscall.MS_RDONLY
	}

	return syscall.Mount(device, mountPath, filesystem, flags, "")
}

func (r *realMountOps) Unmount(path string) error {
	if err := syscall.Unmount(path, 0); err != nil {
		// Try lazy unmount
		return syscall.Unmount(path, syscall.MNT_DETACH)
	}
	return nil
}

func (r *realMountOps) IsMounted(path string) bool {
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return false
	}

	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 && fields[1] == path {
			return true
		}
	}

	return false
}

func (r *realMountOps) IsFormatted(device, filesystem string) (bool, error) {
	cmd := exec.Command("blkid", "-o", "value", "-s", "TYPE", device)
	output, err := cmd.Output()
	if err != nil {
		// No filesystem found
		return false, nil
	}

	fsType := strings.TrimSpace(string(output))
	return fsType == filesystem, nil
}

func (r *realMountOps) FormatDevice(ctx context.Context, device, filesystem string) error {
	var cmd *exec.Cmd

	switch filesystem {
	case "ext4":
		cmd = exec.CommandContext(ctx, "mkfs.ext4", "-F", device)
	case "xfs":
		cmd = exec.CommandContext(ctx, "mkfs.xfs", "-f", device)
	case "btrfs":
		cmd = exec.CommandContext(ctx, "mkfs.btrfs", "-f", device)
	default:
		return fmt.Errorf("unsupported filesystem: %s", filesystem)
	}

	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("mkfs failed: %w: %s", err, string(output))
	}

	return nil
}

func (r *realMountOps) OpenLSVDDisk(ctx context.Context, diskPath, volumeId string) (LSVDDisk, error) {
	sa := &lsvd.LocalFileAccess{
		Dir: diskPath,
		Log: r.log,
	}

	if err := sa.InitContainer(ctx); err != nil {
		return nil, fmt.Errorf("failed to init container: %w", err)
	}

	volInfo, err := sa.GetVolumeInfo(ctx, volumeId)
	if err != nil {
		return nil, fmt.Errorf("failed to get volume info: %w", err)
	}

	disk, err := lsvd.NewDisk(ctx, r.log, diskPath,
		lsvd.WithVolumeName(volumeId),
		lsvd.WithSegmentAccess(sa),
		lsvd.EnableAutoGC,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create disk: %w", err)
	}

	return &realLSVDDisk{
		disk: disk,
		log:  r.log,
		size: volInfo.Size.Bytes().Int64(),
	}, nil
}

// realLSVDDisk wraps an lsvd.Disk
type realLSVDDisk struct {
	disk *lsvd.Disk
	log  *slog.Logger
	size int64
}

func (d *realLSVDDisk) Close(ctx context.Context) error {
	return d.disk.Close(ctx)
}

func (d *realLSVDDisk) Size() int64 {
	return d.size
}

func (d *realLSVDDisk) HandleNBD(ctx context.Context, conn net.Conn, clientFile *os.File) error {
	nbdOpts := &nbd.Options{
		MinimumBlockSize:   4096,
		PreferredBlockSize: 4096,
		RawFile:            clientFile,
	}

	backend := lsvd.NBDWrapper(ctx, d.log, d.disk)
	return nbd.HandleTransport(d.log, conn, backend, nbdOpts)
}
