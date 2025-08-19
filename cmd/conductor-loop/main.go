package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/loop"
)

// Config holds all command-line configuration
type Config struct {
	HandoffDir     string
	PorchPath      string
	PorchURL       string
	Mode          string
	OutDir        string
	Once          bool
	DebounceDur   time.Duration
	Period        time.Duration
}

func main() {
	log.SetPrefix("[conductor-loop] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Parse command-line flags
	config := parseFlags()

	// Ensure handoff directory exists
	if err := os.MkdirAll(config.HandoffDir, 0755); err != nil {
		log.Fatalf("Failed to create handoff directory: %v", err)
	}

	// Convert to absolute path for consistency
	absHandoffDir, err := filepath.Abs(config.HandoffDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path: %v", err)
	}
	config.HandoffDir = absHandoffDir

	// Ensure output directory exists
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
		PorchPath:     config.PorchPath,
		PorchURL:      config.PorchURL,
		Mode:         config.Mode,
		OutDir:       config.OutDir,
		Once:         config.Once,
		DebounceDur:  config.DebounceDur,
		Period:       config.Period,
	})
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start watching
	done := make(chan error, 1)
	go func() {
		done <- watcher.Start()
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
				log.Printf("Failed to get processing stats: %v", statsErr)
				exitCode = 1
			} else if stats.FailedCount > 0 {
				log.Printf("Completed with %d failed files", stats.FailedCount)
				exitCode = 8
			} else {
				log.Printf("All files processed successfully")
				exitCode = 0
			}
		}
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down gracefully", sig)
		watcher.Close()
		exitCode = 0
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
	
	// Create directories if they don't exist (not done in parseFlagsWithFlagSet for testing)
	if err := os.MkdirAll(config.HandoffDir, 0755); err != nil {
		log.Fatalf("Failed to create handoff directory: %v", err)
	}
	if err := os.MkdirAll(config.OutDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}
	
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