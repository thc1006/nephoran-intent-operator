//go:build windows
// +build windows

package porch

// configureProcessGroup is a no-op on Windows.
func (gc *GracefulCommand) configureProcessGroup() {
	// Windows doesn't need special process group configuration.
}
