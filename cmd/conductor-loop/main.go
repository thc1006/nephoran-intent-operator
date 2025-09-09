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

	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
	"github.com/thc1006/nephoran-intent-operator/internal/loop"
)

<<<<<<< HEAD
=======
// hasOnceMode checks if the watcher is configured for once mode (processor approach)
func hasOnceMode(watcher *loop.Watcher) bool {
	// For now, assume processor approach uses once mode in tests
	// This could be extended to check actual watcher configuration if needed
	return true
}

>>>>>>> 6835433495e87288b95961af7173d866977175ff
// isExpectedShutdownError identifies expected errors during shutdown that don't indicate infrastructure issues.

func isExpectedShutdownError(err error) bool {
	if err == nil {
		return false
	}

	errorMsg := strings.ToLower(err.Error())

	// Expected shutdown patterns - these indicate graceful cleanup is in progress.

	expectedPatterns := []string{
		"stats not available - no file manager configured",

		"failed to read directory", // Directory may be temporarily inaccessible during cleanup

		"permission denied", // Cleanup processes may temporarily lock resources

		"file does not exist", // Status files may be cleaned up during shutdown

		"no such file or directory", // Similar to above, for different OS error formats

		"directory not found", // Status directories may be cleaned up

		"access is denied", // Windows equivalent of permission denied

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

	// Clean the path to handle various path formats consistently across platforms.

	cleanPath := filepath.Clean(path)

	// Check if the path exists.

	info, err := os.Stat(cleanPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Path doesn't exist - check if parent directory exists and is accessible.

			parent := filepath.Dir(cleanPath)

			// Special case: if parent is the same as path, we've reached the root.
<<<<<<< HEAD

			if parent == cleanPath {
=======
			// This includes drive roots like "Z:\" or "C:\" on Windows and "/" on Unix

			if parent == cleanPath {
				// For Windows, check if it's a drive root that doesn't exist
				if len(cleanPath) == 3 && cleanPath[1] == ':' && (cleanPath[2] == '\\' || cleanPath[2] == '/') {
					// Test if the drive actually exists by trying to access it
					if _, statErr := os.Stat(cleanPath); statErr != nil {
						return fmt.Errorf("invalid path: drive %s does not exist or is not accessible", cleanPath)
					}
				}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
				return fmt.Errorf("invalid path: %s (cannot validate root directory)", cleanPath)
			}

			// Recursively check if parent is valid for directory creation.

			if err := validateHandoffDir(parent); err != nil {
				return fmt.Errorf("invalid parent directory for %s: %w", cleanPath, err)
			}

			// Parent exists and is valid, so we can create the directory.

			return nil
		}

		// Other error (e.g., permission denied, invalid path format).

		return fmt.Errorf("cannot access path %s: %w", cleanPath, err)
	}

	// Path exists - verify it's a directory.

	if !info.IsDir() {
		return fmt.Errorf("path %s exists but is not a directory", cleanPath)
	}

	// Test read permission by attempting to read the directory.

	_, err = os.ReadDir(cleanPath)
	if err != nil {
		return fmt.Errorf("directory %s exists but is not readable: %w", cleanPath, err)
	}

	return nil
}

// Config holds all command-line configuration.

type Config struct {
	HandoffDir string

	PorchPath string

	PorchURL string

	Mode string

	OutDir string

	Once bool

	DebounceDur time.Duration

	Period time.Duration
}

func main() {
	exitCode := runMain()
	if exitCode != 0 {
		os.Exit(exitCode)
	}
}

func runMain() int {
	log.SetPrefix("[conductor-loop] ")

	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Parse command-line flags - support both legacy and new approach.

	useProcessor := flag.Bool("use-processor", false, "Use IntentProcessor pattern (new approach) instead of Config-based (legacy)")

	// New IntentProcessor approach flags.

	handoffDir := flag.String("handoff-dir", "./handoff", "Directory to watch for intent files")

	errorDir := flag.String("error-dir", "./handoff/errors", "Directory for error files")

	porchMode := flag.String("porch-mode", "structured", "Porch submission mode: 'direct' or 'structured'")

	batchSize := flag.Int("batch-size", 10, "Maximum batch size for processing")

	batchInterval := flag.Duration("batch-interval", 5*time.Second, "Interval for batch processing")

	schemaPath := flag.String("schema", "", "Path to intent schema file (defaults to docs/contracts/intent.schema.json)")

	// Legacy Config approach flags (parsed within parseFlags).

	config := parseFlags()

	var absHandoffDir string

	var err error

	if *useProcessor {
		// New IntentProcessor approach.

		log.Printf("Using IntentProcessor pattern (new approach)")

		// Validate porch mode.

		if *porchMode != "direct" && *porchMode != "structured" {
			log.Printf("Invalid porch-mode: %s. Must be 'direct' or 'structured'", *porchMode)
			return 1
		}

		// Validate directories before creating.

		if err := validateHandoffDir(*handoffDir); err != nil {
			log.Printf("Invalid handoff directory path %s: %v", *handoffDir, err)
			return 1
		}

		// Ensure directories exist.

		for _, dir := range []string{*handoffDir, *errorDir} {
			if err := os.MkdirAll(dir, 0o750); err != nil {
				log.Printf("Failed to create directory %s: %v", dir, err)
				return 1
			}
		}

		// Convert to absolute paths for consistency.

		absHandoffDir, err = filepath.Abs(*handoffDir)
		if err != nil {
			log.Printf("Failed to get absolute path for handoff dir: %v", err)
			return 1
		}
	} else {
		// Legacy Config-based approach.

		log.Printf("Using Config-based pattern (legacy approach)")

		// Validate and create handoff directory.

		if err := validateHandoffDir(config.HandoffDir); err != nil {
			log.Printf("Invalid handoff directory path %s: %v", config.HandoffDir, err)
			return 1
		}

		if err := os.MkdirAll(config.HandoffDir, 0o750); err != nil {
			log.Printf("Failed to create handoff directory: %v", err)
			return 1
		}

		// Convert to absolute path for consistency.

		absHandoffDir, err = filepath.Abs(config.HandoffDir)
		if err != nil {
			log.Printf("Failed to get absolute path for handoff dir: %v", err)
			return 1
		}

		config.HandoffDir = absHandoffDir
	}

	var watcher *loop.Watcher

	if *useProcessor {
		// New IntentProcessor approach setup.

		absErrorDir, err := filepath.Abs(*errorDir)
		if err != nil {
			log.Printf("Failed to get absolute path for error dir: %v", err)
			return 1
		}

		// Determine schema path.

		if *schemaPath == "" {
			// Default to standard location.

			*schemaPath = filepath.Join(".", "docs", "contracts", "intent.schema.json")
		}

		// Create validator (using ingest package).

		validator, err := ingest.NewValidator(*schemaPath)
		if err != nil {
			log.Printf("Failed to create validator: %v", err)
			return 1
		}

		// Create processor configuration.

		processorConfig := &loop.ProcessorConfig{
			HandoffDir: absHandoffDir,

			ErrorDir: absErrorDir,

			PorchMode: *porchMode,

			BatchSize: *batchSize,

			BatchInterval: *batchInterval,

			MaxRetries: 3,
		}

		// Create processor.

		processor, err := loop.NewProcessor(processorConfig, validator, loop.DefaultPorchSubmit)
		if err != nil {
			log.Printf("Failed to create processor: %v", err)
			return 1
		}

		// Start batch processor.

		processor.StartBatchProcessor()

		defer processor.Stop()

		log.Printf("Starting conductor-loop:")

		log.Printf("  Watching: %s", absHandoffDir)

		log.Printf("  Errors: %s", absErrorDir)

		log.Printf("  Mode: %s", *porchMode)

		log.Printf("  Batch: size=%d, interval=%v", *batchSize, *batchInterval)

		// Create and start the watcher with processor.

		watcher, err = loop.NewWatcherWithProcessor(absHandoffDir, processor)
		if err != nil {
			log.Printf("Failed to create watcher: %v", err)
			log.Printf("Cannot continue without watcher: %v", err)
			return 1
		}
	} else {
		// Legacy Config-based approach setup.

		// Validate and create output directory.

		if err := validateHandoffDir(config.OutDir); err != nil {
			log.Printf("Invalid output directory path %s: %v", config.OutDir, err)
			return 1
		}

		if err := os.MkdirAll(config.OutDir, 0o750); err != nil {
			log.Printf("Failed to create output directory: %v", err)
			return 1
		}

		// Convert output directory to absolute path.

		absOutDir, err := filepath.Abs(config.OutDir)
		if err != nil {
			log.Printf("Failed to get absolute output path: %v", err)
			return 1
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

		// Create and start the watcher.

		watcher, err = loop.NewWatcher(config.HandoffDir, loop.Config{
			PorchPath: config.PorchPath,

			PorchURL: config.PorchURL,

			Mode: config.Mode,

			OutDir: config.OutDir,

			Once: config.Once,

			DebounceDur: config.DebounceDur,

			Period: config.Period,
		})
		if err != nil {
			log.Printf("Failed to create watcher: %v", err)
			return 1
		}
	}

	// Safe defer func() { _ = pattern - only register Close() after successful creation.

	defer func() {
		if watcher != nil {
			if err := watcher.Close(); err != nil {
				log.Printf("Error closing watcher: %v", err)
			}
		}
	}()

	// Process existing files on startup (for idempotency) - only for new processor approach.

	if *useProcessor && watcher != nil {
		if err := watcher.ProcessExistingFiles(); err != nil {
			log.Printf("Warning: Failed to process existing files: %v", err)
		}
	}

	// Setup signal handling for graceful shutdown.

	sigChan := make(chan os.Signal, 1)

	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start watching.

	done := make(chan error, 1)

	go func() {
		if watcher != nil {
			done <- watcher.Start()
		} else {
			done <- fmt.Errorf("watcher is nil - cannot start")
		}
	}()

	// Wait for either error or interrupt signal.

	var exitCode int

	select {
	case err := <-done:

		if err != nil {
			log.Printf("Watcher error: %v", err)

			exitCode = 1
<<<<<<< HEAD
		} else if !*useProcessor && config.Once {
			// In once mode, check if any files failed (only for legacy approach).
=======
		} else if config.Once || (*useProcessor && hasOnceMode(watcher)) {
			// In once mode, check if any files failed (both legacy and processor approaches).
>>>>>>> 6835433495e87288b95961af7173d866977175ff

			stats, statsErr := watcher.GetStats()

			switch {
			case statsErr != nil && isExpectedShutdownError(statsErr):
				log.Printf("Stats unavailable due to expected shutdown conditions: %v", statsErr)
				log.Printf("Once mode completed - stats collection temporarily unavailable (expected)")
				exitCode = 3 // Stats unavailable due to expected shutdown conditions

			case statsErr != nil:
				log.Printf("Failed to get processing stats due to unexpected error: %v", statsErr)
				log.Printf("This indicates potential infrastructure issues with status directories or file system")
				exitCode = 2 // Unexpected stats error (infrastructure issues)

			case stats.RealFailedCount > 0:
				// Only real failures should affect exit code, not shutdown failures.
				log.Printf("Completed with %d real failures and %d shutdown failures (total: %d failed files)",
					stats.RealFailedCount, stats.ShutdownFailedCount, stats.FailedCount)
				exitCode = 8

			case stats.ShutdownFailedCount > 0:
				// Only shutdown failures - this is acceptable during graceful shutdown.
				log.Printf("Completed with %d shutdown failures (no real failures)", stats.ShutdownFailedCount)
				exitCode = 0

			default:
				log.Printf("All files processed successfully")
				exitCode = 0
			}
		}

	case sig := <-sigChan:

		log.Printf("Received signal %v, shutting down gracefully", sig)

		if err := watcher.Close(); err != nil {
			log.Printf("Error closing watcher: %v", err)
		}

		// Check stats after graceful shutdown to distinguish shutdown vs real failures.

<<<<<<< HEAD
		if !*useProcessor {
			// Only check stats for legacy approach.
=======
		if !*useProcessor || (*useProcessor && hasOnceMode(watcher)) {
			// Check stats for both legacy approach and processor approach in once mode.
>>>>>>> 6835433495e87288b95961af7173d866977175ff

			stats, statsErr := watcher.GetStats()

			switch {
			case statsErr != nil && isExpectedShutdownError(statsErr):
				log.Printf("Stats unavailable due to expected shutdown conditions: %v", statsErr)
				log.Printf("Graceful shutdown completed - stats collection temporarily unavailable (expected)")
				exitCode = 3 // Stats unavailable due to expected shutdown conditions

			case statsErr != nil:
				log.Printf("Failed to get processing stats after shutdown due to unexpected error: %v", statsErr)
				log.Printf("This indicates potential infrastructure issues with status directories or file system")
				exitCode = 2 // Unexpected stats error (infrastructure issues)

			case stats.RealFailedCount > 0:
				// Only real failures should affect exit code, not shutdown failures.
				log.Printf("Graceful shutdown completed with %d real failures and %d shutdown failures (total: %d failed files)",
					stats.RealFailedCount, stats.ShutdownFailedCount, stats.FailedCount)
				exitCode = 8

			case stats.ShutdownFailedCount > 0:
				// Only shutdown failures - this is acceptable during graceful shutdown.
				log.Printf("Graceful shutdown completed with %d shutdown failures (no real failures)", stats.ShutdownFailedCount)
				exitCode = 0

			default:
				log.Printf("Graceful shutdown completed - all files processed successfully")
				exitCode = 0
			}
		} else {
			// For new processor approach, graceful shutdown is always successful.

			exitCode = 0

			log.Printf("Graceful shutdown completed")
		}
	}

	log.Println("Conductor-loop stopped")

	return exitCode
}

// parseFlags parses and validates command-line flags.

func parseFlags() Config {
	config, err := parseFlagsWithFlagSet(flag.CommandLine, os.Args[1:])
	if err != nil {
		log.Printf("Error parsing flags: %v", err)
		os.Exit(1)
	}

	// Note: directory creation is done in main() after validation.

	return config
}

// parseFlagsWithFlagSet parses flags using a specific FlagSet (for testing).

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

	// Parse debounce duration.

	var err error

	config.DebounceDur, err = time.ParseDuration(debounceDurStr)
	if err != nil {
		return config, fmt.Errorf("invalid debounce duration %q: %w", debounceDurStr, err)
	}

	// Parse period duration.

	config.Period, err = time.ParseDuration(periodStr)
	if err != nil {
		return config, fmt.Errorf("invalid period duration %q: %w", periodStr, err)
	}

	// Validate mode.

	if config.Mode != "direct" && config.Mode != "structured" {
		return config, fmt.Errorf("invalid mode %q: must be 'direct' or 'structured'", config.Mode)
	}

	return config, nil
}
