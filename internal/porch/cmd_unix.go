//go:build !windows
// +build !windows

package porch

import (
	"strings"
)

// cmdSafeQuote is a no-op on Unix systems since this function.
// is only needed for Windows cmd.exe. On Unix systems, exec.Command.
// handles argument escaping properly by default.
//
// This function exists to satisfy the cross-platform compilation.
// requirement when building on Linux/Unix systems.
func cmdSafeQuote(args []string) string {
	// On Unix systems, we shouldn't be calling Windows-specific functions,.
	// but if we do reach here, just join the arguments with spaces.
	// This should never actually be used since pathutil.IsWindowsBatchFile.
	// should return false on Unix systems.
	return strings.Join(args, " ")
}
