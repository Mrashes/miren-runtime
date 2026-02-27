package snapshot

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/klauspost/compress/zstd"
)

// Backup reads the disk image from src, computes its SHA-256 checksum,
// and writes a complete snapshot (header + zstd-compressed data) to dst.
// It returns the checksum of the uncompressed image data.
func Backup(dst io.Writer, src io.ReadSeeker, name string, sizeBytes int64, filesystem string) (checksum string, err error) {
	// First pass: compute SHA-256 checksum
	hasher := sha256.New()
	if _, err := io.Copy(hasher, src); err != nil {
		return "", fmt.Errorf("computing checksum: %w", err)
	}
	checksum = hex.EncodeToString(hasher.Sum(nil))

	// Seek back to beginning for compression pass
	if _, err := src.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("seeking image: %w", err)
	}

	// Write snapshot header
	meta := &Meta{
		Name:       name,
		SizeBytes:  sizeBytes,
		Filesystem: filesystem,
		Timestamp:  time.Now().UTC(),
		Checksum:   checksum,
		Version:    FormatVersion,
	}

	if err := WriteHeader(dst, meta); err != nil {
		return "", fmt.Errorf("writing snapshot header: %w", err)
	}

	// Stream image data through zstd compression
	encoder, err := zstd.NewWriter(dst)
	if err != nil {
		return "", fmt.Errorf("creating zstd encoder: %w", err)
	}

	buf := make([]byte, 4*1024*1024) // 4MB buffer
	if _, err = io.CopyBuffer(encoder, src, buf); err != nil {
		encoder.Close()
		return "", fmt.Errorf("compressing image data: %w", err)
	}

	if err = encoder.Close(); err != nil {
		return "", fmt.Errorf("finalizing compression: %w", err)
	}

	return checksum, nil
}
