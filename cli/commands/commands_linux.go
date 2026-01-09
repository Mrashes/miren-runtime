package commands

import (
	"miren.dev/mflags"
)

func addCommands(d *mflags.Dispatcher) {
	// Server command is now defined in commands.go (renamed from dev)

	// Cloud registration commands
	d.Dispatch("server register", Infer("server register", "Register this cluster with miren.cloud", Register))

	d.Dispatch("server register status", Infer("server register status", "Show cluster registration status", RegisterStatus))

	// Server management commands
	d.Dispatch("server install", Infer("server install", "Install systemd service for miren server", ServerInstall))

	d.Dispatch("server uninstall", Infer("server uninstall", "Remove systemd service for miren server", ServerUninstall))

	d.Dispatch("server status", Infer("server status", "Show miren service status", ServerStatus))
}

// setupServerComponents is deprecated and will be removed.
// All server components are now initialized explicitly via ServerState.
