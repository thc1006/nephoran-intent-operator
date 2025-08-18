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

func main() {
	log.SetPrefix("[conductor-loop] ")
	log.SetFlags(log.Ldate | log.Ltime | log.Lmicroseconds)

	// Parse command-line flags
	handoffDir := flag.String("handoff-dir", "./handoff", "Directory to watch for intent files")
	errorDir := flag.String("error-dir", "./handoff/errors", "Directory for error files")
	porchMode := flag.String("porch-mode", "structured", "Porch submission mode: 'direct' or 'structured'")
	batchSize := flag.Int("batch-size", 10, "Maximum batch size for processing")
	batchInterval := flag.Duration("batch-interval", 5*time.Second, "Interval for batch processing")
	schemaPath := flag.String("schema", "", "Path to intent schema file (defaults to docs/contracts/intent.schema.json)")
	flag.Parse()

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
	absHandoffDir, err := filepath.Abs(*handoffDir)
	if err != nil {
		log.Fatalf("Failed to get absolute path for handoff dir: %v", err)
	}

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
	watcher, err := loop.NewWatcherWithProcessor(absHandoffDir, processor)
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
	select {
	case err := <-done:
		if err != nil {
			log.Fatalf("Watcher error: %v", err)
		}
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down gracefully", sig)
		watcher.Close()
		processor.Stop()
	}

	log.Println("Conductor-loop stopped")
}