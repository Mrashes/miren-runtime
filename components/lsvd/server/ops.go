package server

import (
	"context"
	"net"
	"os"

	"miren.dev/runtime/lsvd"
	"miren.dev/runtime/pkg/units"
)

// VolumeOps abstracts OS and LSVD operations for volume management.
// This interface enables testing without requiring actual filesystem or LSVD operations.
type VolumeOps interface {
	// CreateVolumeDir creates the directory for a volume
	CreateVolumeDir(path string) error

	// RemoveVolumeDir removes a volume directory and all its contents
	RemoveVolumeDir(path string) error

	// VolumePathExists checks if a volume path exists
	VolumePathExists(path string) bool

	// InitLSVDVolume initializes an LSVD volume at the given path
	InitLSVDVolume(ctx context.Context, path, volumeId string, size units.Bytes, metadata map[string]any) error
}

// MountOps abstracts OS operations for mount management.
// This interface enables testing without requiring actual NBD, device, or mount operations.
type MountOps interface {
	// CreateDir creates a directory with the specified permissions
	CreateDir(path string, perm os.FileMode) error

	// RemoveFile removes a file
	RemoveFile(path string) error

	// NBDLoopback sets up an NBD loopback device
	// Returns: index, conn, clientFile, cleanup function, error
	NBDLoopback(ctx context.Context, sizeBytes uint64) (uint32, net.Conn, *os.File, func() error, error)

	// NBDStatus checks the status of an NBD device
	NBDStatus(idx uint32) error

	// NBDDisconnect disconnects an NBD device
	NBDDisconnect(idx uint32) error

	// CreateDeviceNode creates a block device node
	CreateDeviceNode(path string, nbdIndex uint32) error

	// Mount mounts a device at the specified path
	Mount(device, mountPath, filesystem string, readOnly bool) error

	// Unmount unmounts a path
	Unmount(path string) error

	// IsMounted checks if a path is a mount point
	IsMounted(path string) bool

	// IsFormatted checks if a device has a filesystem
	IsFormatted(device, filesystem string) (bool, error)

	// FormatDevice formats a device with the specified filesystem
	FormatDevice(ctx context.Context, device, filesystem string) error

	// OpenLSVDDisk opens an LSVD disk for the given volume
	OpenLSVDDisk(ctx context.Context, diskPath, volumeId string) (LSVDDisk, error)
}

// LSVDDisk abstracts LSVD disk operations for NBD handling
type LSVDDisk interface {
	// Close closes the disk
	Close(ctx context.Context) error

	// Size returns the disk size in bytes
	Size() int64

	// HandleNBD handles NBD protocol on the given connection
	HandleNBD(ctx context.Context, conn net.Conn, clientFile *os.File) error
}

// realVolumeOps implements VolumeOps with real OS/LSVD operations
type realVolumeOps struct{}

// NewRealVolumeOps creates a VolumeOps that performs real OS operations
func NewRealVolumeOps() VolumeOps {
	return &realVolumeOps{}
}

func (r *realVolumeOps) CreateVolumeDir(path string) error {
	return os.MkdirAll(path, 0755)
}

func (r *realVolumeOps) RemoveVolumeDir(path string) error {
	return os.RemoveAll(path)
}

func (r *realVolumeOps) VolumePathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func (r *realVolumeOps) InitLSVDVolume(ctx context.Context, path, volumeId string, size units.Bytes, metadata map[string]any) error {
	sa := &lsvd.LocalFileAccess{Dir: path}

	if err := sa.InitContainer(ctx); err != nil {
		return err
	}

	volInfo := &lsvd.VolumeInfo{
		Name:     volumeId,
		Size:     size,
		UUID:     volumeId,
		Metadata: metadata,
	}

	return sa.InitVolume(ctx, volInfo)
}
