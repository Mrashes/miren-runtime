//go:build !linux

package commands

// ServerState is a stub for non-Linux platforms.
// The server functionality is only available on Linux.
type ServerState struct{}

// NewServerState returns a stub ServerState on non-Linux platforms.
func NewServerState() *ServerState {
	return &ServerState{}
}
