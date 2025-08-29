//go:build !windows
// +build !windows

package porch

import (
	"syscall"
)

// configureProcessGroup configures Unix-specific process group settings.
func (gc *GracefulCommand) configureProcessGroup() {
	gc.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true, // Create new process group for easier cleanup
	}
}
