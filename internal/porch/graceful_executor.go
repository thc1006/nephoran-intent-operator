package porch

import (
	"context"
	"log"
	"os/exec"
	"runtime"
	"syscall"
	"time"
)

// GracefulCommand wraps exec.Cmd with graceful termination support
type GracefulCommand struct {
	*exec.Cmd
	gracePeriod time.Duration
}

// NewGracefulCommand creates a command with graceful shutdown support
func NewGracefulCommand(ctx context.Context, name string, args ...string) *GracefulCommand {
	cmd := exec.CommandContext(ctx, name, args...)

	// Set graceful termination behavior
	gc := &GracefulCommand{
		Cmd:         cmd,
		gracePeriod: 5 * time.Second, // Default grace period
	}

	// Configure graceful process termination
	gc.setupGracefulTermination()

	return gc
}

// setupGracefulTermination configures the command for graceful shutdown
func (gc *GracefulCommand) setupGracefulTermination() {
	// Set Cancel function to send SIGTERM first
	gc.Cancel = func() error {
		if gc.Process == nil {
			return nil
		}

		log.Printf("Sending graceful termination signal to process %d", gc.Process.Pid)

		// Send SIGTERM first for graceful shutdown
		if runtime.GOOS == "windows" {
			// On Windows, we can only Kill(), but the process will handle it as best it can
			// Note: Windows doesn't have SIGTERM, so this is the equivalent
			return gc.Process.Kill()
		} else {
			// On Unix systems, send SIGTERM first for graceful shutdown
			if err := gc.Process.Signal(syscall.SIGTERM); err != nil {
				log.Printf("Failed to send SIGTERM to process %d: %v", gc.Process.Pid, err)
				return err
			}
			log.Printf("Sent SIGTERM to process %d, waiting %v before SIGKILL", gc.Process.Pid, gc.gracePeriod)
			return nil
		}
	}

	// Set WaitDelay for escalation to SIGKILL
	gc.WaitDelay = gc.gracePeriod

	// Set process group for better process management on Unix systems
	if runtime.GOOS != "windows" {
		gc.configurePlatformSpecific()
	}
}

// SetGracePeriod configures how long to wait before escalating to SIGKILL
func (gc *GracefulCommand) SetGracePeriod(d time.Duration) {
	gc.gracePeriod = d
	gc.WaitDelay = d
}

// RunWithGracefulShutdown runs the command with enhanced error handling
func (gc *GracefulCommand) RunWithGracefulShutdown() error {
	log.Printf("Starting graceful command: %s", gc.String())

	if err := gc.Start(); err != nil {
		return err
	}

	log.Printf("Process %d started, waiting for completion...", gc.Process.Pid)

	err := gc.Wait()
	if err != nil {
		// Check if the error is due to graceful termination
		if exitErr, ok := err.(*exec.ExitError); ok {
			if exitErr.ExitCode() == -1 || exitErr.ExitCode() == 130 || exitErr.ExitCode() == 143 {
				log.Printf("Process %d terminated gracefully (exit code: %d)", gc.Process.Pid, exitErr.ExitCode())
				return nil // Consider graceful termination as success
			}
		}
	}

	return err
}

// configurePlatformSpecific configures platform-specific process attributes
func (gc *GracefulCommand) configurePlatformSpecific() {
	// On Windows, we don't need to set process groups
	// This method can be extended with platform-specific build tags if needed
}
