//go:build !linux

package commands

import (
	"miren.dev/mflags"
)

func addCommands(d *mflags.Dispatcher) {
	// Server management commands - provide helpful errors directing to Docker
	d.Dispatch("server install", Infer("server install", "Install miren server (Linux only)", ServerInstall))

	d.Dispatch("server uninstall", Infer("server uninstall", "Uninstall miren server (Linux only)", ServerUninstall))

	d.Dispatch("server status", Infer("server status", "Show miren service status (Linux only)", ServerStatus))
}
