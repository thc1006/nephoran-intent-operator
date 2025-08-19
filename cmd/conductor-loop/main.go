package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
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

	// Parse command-line flags - support both legacy and new approach
	useProcessor := flag.Bool("use-processor", false, "Use IntentProcessor pattern (new approach) instead of Config-based (legacy)")
	
	// New IntentProcessor approach flags
	handoffDir := flag.String("handoff-dir", "./handoff", "Directory to watch for intent files")
	errorDir := flag.String("error-dir", "./handoff/errors", "Directory for error files")
	porchMode := flag.String("porch-mode", "structured", "Porch submission mode: 'direct' or 'structured'")
	batchSize := flag.Int("batch-size", 10, "Maximum batch size for processing")
	batchInterval := flag.Duration("batch-interval", 5*time.Second, "Interval for batch processing")
	schemaPath := flag.String("schema", "", "Path to intent schema file (defaults to docs/contracts/intent.schema.json)")
	
	// Legacy Config approach flags (parsed within parseFlags)
	config := parseFlags()
	
	var absHandoffDir string
	var err error
	
	if *useProcessor {
		// New IntentProcessor approach
		log.Printf("Using IntentProcessor pattern (new approach)")
		
		// Validate porch mode
		if *porchMode != "direct" && *porchMode != "structured" {
			log.Fatalf("Invalid porch-mode: %s. Must be 'direct' or 'structured'", *porchMode)
		}
		
		// Ensure directories exist
		for _, dir := range []string{*handoffDir, *errorDir} {
			if err := os.MkdirAll(dir, 0755); err != nil {
				log.Fatalf("Failed to create directory %s: %v", dir, err)
			}
		}
		
		// Convert to absolute paths for consistency
		absHandoffDir, err = filepath.Abs(*handoffDir)
		if err != nil {
			log.Fatalf("Failed to get absolute path for handoff dir: %v", err)
		}
	} else {
		// Legacy Config-based approach
		log.Printf("Using Config-based pattern (legacy approach)")
		
		// Ensure handoff directory exists
		if err := os.MkdirAll(config.HandoffDir, 0755); err != nil {
			log.Fatalf("Failed to create handoff directory: %v", err)
		}
		
		// Convert to absolute path for consistency
		absHandoffDir, err = filepath.Abs(config.HandoffDir)
		if err != nil {
			log.Fatalf("Failed to get absolute path for handoff dir: %v", err)
		}
		config.HandoffDir = absHandoffDir
	}
	
	var watcher *loop.Watcher
	
	if *useProcessor {
		// New IntentProcessor approach setup
		absErrorDir, err := filepath.Abs(*errorDir)
		if err != nil {
			log.Fatalf("Failed to get absolute path for error dir: %v", err)
		}

		// Determine schema path
		if *schemaPath == "" {
			// Default to standard location
			*schemaPath = filepath.Join(".", "docs", "contracts", "intent.schema.json")
		}

		// Create validator
		validator, err := ingest.NewValidator(*schemaPath)
		if err != nil {
			log.Fatalf("Failed to create validator: %v", err)
		}

		// Create processor configuration
		processorConfig := &loop.ProcessorConfig{
			HandoffDir:    absHandoffDir,
			ErrorDir:      absErrorDir,
			PorchMode:     *porchMode,
			BatchSize:     *batchSize,
			BatchInterval: *batchInterval,
			MaxRetries:    3,
		}

		// Create processor
		processor, err := loop.NewProcessor(processorConfig, validator, loop.DefaultPorchSubmit)
		if err != nil {
			log.Fatalf("Failed to create processor: %v", err)
		}

		// Start batch processor
		processor.StartBatchProcessor()
		defer processor.Stop()

		log.Printf("Starting conductor-loop:")
		log.Printf("  Watching: %s", absHandoffDir)
		log.Printf("  Errors: %s", absErrorDir)
		log.Printf("  Mode: %s", *porchMode)
		log.Printf("  Batch: size=%d, interval=%v", *batchSize, *batchInterval)

		// Create and start the watcher with processor
		watcher, err = loop.NewWatcherWithProcessor(absHandoffDir, processor)
	} else {
		// Legacy Config-based approach setup
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
		watcher, err = loop.NewWatcher(absHandoffDir, loop.Config{
			PorchPath:     config.PorchPath,
			PorchURL:      config.PorchURL,
			Mode:         config.Mode,
			OutDir:       config.OutDir,
			Once:         config.Once,
			DebounceDur:  config.DebounceDur,
			Period:       config.Period,
		})
	}
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	defer watcher.Close()

	// Process existing files on startup (for idempotency)
	if err := watcher.ProcessExistingFiles(); err != nil {
		log.Printf("Warning: Failed to process existing files: %v", err)
	}

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
		} else if (!*useProcessor && config.Once) {
			// In once mode, check if any files failed (only for legacy approach)
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
	var config Config

	flag.StringVar(&config.HandoffDir, "handoff", "./handoff", "Directory to watch for intent files")
	flag.StringVar(&config.PorchPath, "porch", "porch", "Path to porch executable")
	flag.StringVar(&config.PorchURL, "porch-url", "", "Porch HTTP URL (optional, for direct API calls)")
	flag.StringVar(&config.Mode, "mode", "direct", "Processing mode: direct or structured")
	flag.StringVar(&config.OutDir, "out", "./out", "Output directory for processed files")
	flag.BoolVar(&config.Once, "once", false, "Process current backlog then exit")
	
	var debounceDurStr string
	flag.StringVar(&debounceDurStr, "debounce", "500ms", "Debounce duration for file events (Windows optimization)")
	
	var periodStr string
	flag.StringVar(&periodStr, "period", "2s", "Polling period for scanning directory (default 2s)")

	flag.Parse()

	// Parse debounce duration
	var err error
	config.DebounceDur, err = time.ParseDuration(debounceDurStr)
	if err != nil {
		log.Fatalf("Invalid debounce duration %q: %v", debounceDurStr, err)
	}

	// Parse period duration
	config.Period, err = time.ParseDuration(periodStr)
	if err != nil {
		log.Fatalf("Invalid period duration %q: %v", periodStr, err)
	}

	// Validate mode
	if config.Mode != "direct" && config.Mode != "structured" {
		log.Fatalf("Invalid mode %q: must be 'direct' or 'structured'", config.Mode)
	}

	return config
}