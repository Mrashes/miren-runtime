// Package subsystem provides infrastructure for building composable server subsystems.
//
// Subsystems group related components that are built together with explicit dependencies.
// Each subsystem follows a pattern:
//   - Config struct with required fields
//   - Option functions for optional/override dependencies
//   - New() constructor that validates config and builds components
//   - Close() method for cleanup
package subsystem

import "io"

// Subsystem is implemented by all subsystems that require cleanup.
type Subsystem interface {
	io.Closer
}
