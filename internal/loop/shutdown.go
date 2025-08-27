package loop

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"syscall"
)

// IsShutdownFailure is the single source of truth for determining if an error
// was caused by graceful shutdown. This implements the design rules:
//
// - If err == nil: return false
// - If !shuttingDown: return false even for context.Canceled/DeadlineExceeded or signals
// - If shuttingDown: return true when:
//   - errors.Is(err, context.Canceled) or errors.Is(err, context.DeadlineExceeded), OR
//   - err is *exec.ExitError and its WaitStatus.Signal() ∈ {SIGTERM, SIGKILL}, OR
//   - err is *exec.ExitError with exit codes 137 (SIGKILL) or 143 (SIGTERM), OR
//   - error message contains "signal: killed" or similar shutdown-related signals
func IsShutdownFailure(shuttingDown bool, err error) bool {
	// Rule 1: If err == nil, return false
	if err == nil {
		return false
	}

	// Rule 2: If not shutting down, return false even for cancellation/signals
	if !shuttingDown {
		return false
	}

	// Rule 3: If shutting down, check for shutdown-related errors

	// Check for context cancellation/timeout
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Check for process signals and exit codes
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		// Try to get signal information (Unix/Linux)
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			sig := status.Signal()
			if sig == syscall.SIGTERM || sig == syscall.SIGKILL {
				return true
			}
		}

		// Check exit codes for container/process termination
		exitCode := exitErr.ExitCode()
		if exitCode == 137 || exitCode == 143 {
			// 137 = 128 + 9 (SIGKILL), 143 = 128 + 15 (SIGTERM)
			return true
		}
	}

	// Check error message for signal-related termination
	errMsg := err.Error()
	if containsSignalKeywords(errMsg) {
		return true
	}

	return false
}

// containsSignalKeywords checks if the error message contains signal-related keywords
// that indicate shutdown-related termination
func containsSignalKeywords(errMsg string) bool {
	shutdownKeywords := []string{
		"signal: killed",
		"signal: terminated",
		"signal: interrupt",
	}

	for _, keyword := range shutdownKeywords {
		if strings.Contains(strings.ToLower(errMsg), keyword) {
			return true
		}
	}

	return false
}
