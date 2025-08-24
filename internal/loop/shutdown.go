package loop

import (
	"context"
	"errors"
	"os/exec"
	"syscall"
)

// IsShutdownFailure is the single source of truth for determining if an error
// was caused by graceful shutdown. This implements the design rules:
//
// - If err == nil: return false
// - If !shuttingDown: return false even for context.Canceled/DeadlineExceeded or signals
// - If shuttingDown: return true when:
//   - errors.Is(err, context.Canceled) or errors.Is(err, context.DeadlineExceeded), OR
//   - err is *exec.ExitError and its WaitStatus.Signal() ∈ {SIGTERM, SIGKILL}
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
	
	// Check for process signals (SIGTERM/SIGKILL)
	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			sig := status.Signal()
			if sig == syscall.SIGTERM || sig == syscall.SIGKILL {
				return true
			}
		}
	}
	
	return false
}