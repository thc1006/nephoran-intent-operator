package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/loop"
)

// isExpectedShutdownError identifies expected errors during shutdown that don't indicate infrastructure issues
func isExpectedShutdownError(err error) bool {
	if err == nil {
		return false
	}

	errorMsg := strings.ToLower(err.Error())

	// Expected shutdown patterns - these indicate graceful cleanup is in progress
	expectedPatterns := []string{
		"stats not available - no file manager configured",
		"failed to read directory",  // Directory may be temporarily inaccessible during cleanup
		"permission denied",         // Cleanup processes may temporarily lock resources
		"file does not exist",       // Status files may be cleaned up during shutdown
		"no such file or directory", // Similar to above, for different OS error formats
		"directory not found",       // Status directories may be cleaned up
		"access is denied",          // Windows equivalent of permission denied
	}

	for _, pattern := range expectedPatterns {
		if strings.Contains(errorMsg, pattern) {
			return true
		}
	}

	return false
}

// validateHandoffDir validates that a path exists and is accessible for directory operations.
// It checks if the path exists, is a directory (not a file), and has read permissions.
// Returns clear error messages for each failure case, working consistently across Windows, Linux, and macOS.
func validateHandoffDir(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	// Clean the path to handle various path formats consistently across platforms
	cleanPath := filepath.Clean(path)

	// Check if the path exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Path doesn't exist - check if parent directory exists and is accessible
			parent := filepath.Dir(cleanPath)

			// Special case: if parent is the same as path, we've reached the root
			if parent == cleanPath {
				return fmt.Errorf("invalid path: %s (cannot validate root directory)", cleanPath)
			}

			// Recursively check if parent is valid for directory creation
			if err := validateHandoffDir(parent); err != nil {
				return fmt.Errorf("invalid parent directory for %s: %v", cleanPath, err)
			}

			// Parent exists and is valid, so we can create the directory
			return nil
		}

		// Other error (e.g., permission denied, invalid path format)
		return fmt.Errorf("cannot access path %s: %v", cleanPath, err)
	}

	// Path exists - verify it's a directory
	if !info.IsDir() {
		return fmt.Errorf("path %s exists but is not a directory", cleanPath)
	}

	// Test read permission by attempting to read the directory
	_, err = os.ReadDir(cleanPath)
	if err != nil {
		return fmt.Errorf("directory %s exists but is not readable: %v", cleanPath, err)
	}

	return nil
}

// Config holds all command-line configuration
type Config struct {
	HandoffDir  string
	PorchPath   string
	PorchURL    string
	Mode        string
	OutDir      string
	Once        bool
	DebounceDur time.Duration
	Period      time.Duration
}

func main() {
	log.SetPrefix("[conductor-loop] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Parse command-line flags
	config := parseFlags()

	// Validate and create handoff directory
	if err := validateHandoffDir(config.HandoffDir); err != nil {
		log.Fatalf("Invalid handoff directory path %s: %v", config.HandoffDir, err)
	}
	if err := os.MkdirAll(config.HandoffDir, 0755); err != nil {
		log.Fatalf("Failed to create handoff directory: %v", err)
	}

	// Convert to absolute path for consistency
	absHandoffDir, err := filepath.Abs(config.HandoffDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}
	config.HandoffDir = absHandoffDir

	// Validate and create output directory
	if err := validateHandoffDir(config.OutDir); err != nil {
		log.Fatalf("Invalid output directory path %s: %v", config.OutDir, err)
	}
	if err := os.MkdirAll(config.OutDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Convert output directory to absolute path
	absOutDir, err := filepath.Abs(config.OutDir)
	if err != nil {
		log.Fatalf("Failed to get absolute output path: %v", err)
	}
	config.OutDir = absOutDir

	log.Printf("Starting conductor-loop with config:")
	log.Printf("  Handoff directory: %s", config.HandoffDir)
	log.Printf("  Porch executable: %s", config.PorchPath)
	log.Printf("  Porch URL: %s", config.PorchURL)
	log.Printf("  Mode: %s", config.Mode)
	log.Printf("  Output directory: %s", config.OutDir)
	log.Printf("  Once mode: %t", config.Once)
	log.Printf("  Debounce duration: %v", config.DebounceDur)
	log.Printf("  Period: %v", config.Period)

	// Create and start the watcher
	watcher, err := loop.NewWatcher(config.HandoffDir, loop.Config{
		PorchPath:   config.PorchPath,
		PorchURL:    config.PorchURL,
		Mode:        config.Mode,
		OutDir:      config.OutDir,
		Once:        config.Once,
		DebounceDur: config.DebounceDur,
		Period:      config.Period,
	})
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}

	// Safe defer pattern - only register Close() after successful creation
	defer func() {
		if watcher != nil {
			if err := watcher.Close(); err != nil {
				log.Printf("Error closing watcher: %v", err)
			}
		}
	}()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start watching
	done := make(chan error, 1)
	go func() {
		if watcher != nil {
			done <- watcher.Start()
		} else {
			done <- fmt.Errorf("watcher is nil - cannot start")
		}
	}()

	// Wait for either error or interrupt signal
	var exitCode int
	select {
	case err := <-done:
		if err != nil {
			log.Printf("Watcher error: %v", err)
			exitCode = 1
		} else if config.Once {
			// In once mode, check if any files failed
			stats, statsErr := watcher.GetStats()
			if statsErr != nil {
				if isExpectedShutdownError(statsErr) {
					log.Printf("Stats unavailable due to expected shutdown conditions: %v", statsErr)
					log.Printf("Once mode completed - stats collection temporarily unavailable (expected)")
					exitCode = 3 // Stats unavailable due to expected shutdown conditions
				} else {
					log.Printf("Failed to get processing stats due to unexpected error: %v", statsErr)
					log.Printf("This indicates potential infrastructure issues with status directories or file system")
					exitCode = 2 // Unexpected stats error (infrastructure issues)
				}
			} else if stats.RealFailedCount > 0 {
				// Only real failures should affect exit code, not shutdown failures
				log.Printf("Completed with %d real failures and %d shutdown failures (total: %d failed files)",
					stats.RealFailedCount, stats.ShutdownFailedCount, stats.FailedCount)
				exitCode = 8
			} else if stats.ShutdownFailedCount > 0 {
				// Only shutdown failures - this is acceptable during graceful shutdown
				log.Printf("Completed with %d shutdown failures (no real failures)", stats.ShutdownFailedCount)
				exitCode = 0
			} else {
				log.Printf("All files processed successfully")
				exitCode = 0
			}
		}
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down gracefully", sig)
		watcher.Close()

		// Check stats after graceful shutdown to distinguish shutdown vs real failures
		stats, statsErr := watcher.GetStats()
		if statsErr != nil {
			if isExpectedShutdownError(statsErr) {
				log.Printf("Stats unavailable due to expected shutdown conditions: %v", statsErr)
				log.Printf("Graceful shutdown completed - stats collection temporarily unavailable (expected)")
				exitCode = 3 // Stats unavailable due to expected shutdown conditions
			} else {
				log.Printf("Failed to get processing stats after shutdown due to unexpected error: %v", statsErr)
				log.Printf("This indicates potential infrastructure issues with status directories or file system")
				exitCode = 2 // Unexpected stats error (infrastructure issues)
			}
		} else if stats.RealFailedCount > 0 {
			// Only real failures should affect exit code, not shutdown failures
			log.Printf("Graceful shutdown completed with %d real failures and %d shutdown failures (total: %d failed files)",
				stats.RealFailedCount, stats.ShutdownFailedCount, stats.FailedCount)
			exitCode = 8
		} else if stats.ShutdownFailedCount > 0 {
			// Only shutdown failures - this is acceptable during graceful shutdown
			log.Printf("Graceful shutdown completed with %d shutdown failures (no real failures)", stats.ShutdownFailedCount)
			exitCode = 0
		} else {
			log.Printf("Graceful shutdown completed - all files processed successfully")
			exitCode = 0
		}
	}

	log.Println("Conductor-loop stopped")
	os.Exit(exitCode)
}

// parseFlags parses and validates command-line flags
func parseFlags() Config {
	config, err := parseFlagsWithFlagSet(flag.CommandLine, os.Args[1:])
	if err != nil {
		log.Fatalf("Error parsing flags: %v", err)
	}

	// Note: directory creation is done in main() after validation
	return config
}

// parseFlagsWithFlagSet parses flags using a specific FlagSet (for testing)
func parseFlagsWithFlagSet(fs *flag.FlagSet, args []string) (Config, error) {
	var config Config

	fs.StringVar(&config.HandoffDir, "handoff", "./handoff", "Directory to watch for intent files")
	fs.StringVar(&config.PorchPath, "porch", "porch", "Path to porch executable")
	fs.StringVar(&config.PorchURL, "porch-url", "", "Porch HTTP URL (optional, for direct API calls)")
	fs.StringVar(&config.Mode, "mode", "direct", "Processing mode: direct or structured")
	fs.StringVar(&config.OutDir, "out", "./out", "Output directory for processed files")
	fs.BoolVar(&config.Once, "once", false, "Process current backlog then exit")

	var debounceDurStr string
	fs.StringVar(&debounceDurStr, "debounce", "500ms", "Debounce duration for file events (Windows optimization)")

	var periodStr string
	fs.StringVar(&periodStr, "period", "2s", "Polling period for scanning directory (default 2s)")

	if err := fs.Parse(args); err != nil {
		return config, err
	}

	// Parse debounce duration
	var err error
	config.DebounceDur, err = time.ParseDuration(debounceDurStr)
	if err != nil {
		return config, fmt.Errorf("invalid debounce duration %q: %v", debounceDurStr, err)
	}

	// Parse period duration
	config.Period, err = time.ParseDuration(periodStr)
	if err != nil {
		return config, fmt.Errorf("invalid period duration %q: %v", periodStr, err)
	}

	// Validate mode
	if config.Mode != "direct" && config.Mode != "structured" {
		return config, fmt.Errorf("invalid mode %q: must be 'direct' or 'structured'", config.Mode)
	}

	return config, nil
}
