package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Intent represents the intent JSON structure for extracting correlation_id
type Intent struct {
	CorrelationID string `json:"correlation_id,omitempty"`
}

func main() {
	var (
		handoffDir string
		outDir     string
	)

	flag.StringVar(&handoffDir, "watch", "handoff", "Directory to watch for intent-*.json files")
	flag.StringVar(&outDir, "out", "", "Output directory for scaling patches (defaults to examples/packages/scaling)")
	flag.Parse()

	// Set default output directory if not specified
	if outDir == "" {
		outDir = os.Getenv("CONDUCTOR_OUT_DIR")
		if outDir == "" {
			outDir = "examples/packages/scaling"
		}
	}

	log.Printf("Starting conductor file watcher")
	log.Printf("Watching directory: %s", handoffDir)
	log.Printf("Output directory: %s", outDir)

	// Create context for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Setup signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create file watcher
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	defer func() { _ = watcher.Close() }()

	// Add handoff directory to watcher
	err = watcher.Add(handoffDir)
	if err != nil {
		log.Fatalf("Failed to add directory to watcher: %v", err)
	}

	// Start event processing
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Create == fsnotify.Create {
					if isIntentFile(event.Name) {
						log.Printf("New intent file detected: %s", event.Name)
						// Add small delay to ensure file is fully written
						time.Sleep(100 * time.Millisecond)
						processIntentFile(event.Name, outDir)
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				log.Printf("Watcher error: %v", err)
			}
		}
	}()

	log.Printf("Conductor is running. Press Ctrl+C to stop.")

	// Wait for shutdown signal
	<-sigChan
	log.Println("Shutdown signal received, stopping conductor...")
	cancel()

	// Give goroutines time to finish
	time.Sleep(500 * time.Millisecond)
	log.Println("Conductor stopped")
}

// isIntentFile checks if the file matches the pattern intent-*.json
func isIntentFile(path string) bool {
	filename := filepath.Base(path)
	return strings.HasPrefix(filename, "intent-") && strings.HasSuffix(filename, ".json")
}

// processIntentFile triggers porch-publisher for the given intent file
func processIntentFile(intentPath, outDir string) {
	// Try to extract correlation_id from the intent file
	correlationID := extractCorrelationID(intentPath)
	if correlationID != "" {
		log.Printf("Processing intent with correlation_id: %s", correlationID)
	}

	// Build the command to run porch-publisher
	cmd := exec.Command("go", "run", "./cmd/porch-publisher", "-intent", intentPath, "-out", outDir)

	// Capture output
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Error running porch-publisher: %v", err)
		log.Printf("Output: %s", string(output))
		return
	}

	log.Printf("Successfully processed intent file: %s", intentPath)
	log.Printf("Porch-publisher output: %s", strings.TrimSpace(string(output)))
}

// extractCorrelationID reads the intent file and extracts correlation_id if present
func extractCorrelationID(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("Warning: could not read intent file for correlation_id: %v", err)
		return ""
	}

	var intent Intent
	if err := json.Unmarshal(data, &intent); err != nil {
		log.Printf("Warning: could not parse intent JSON for correlation_id: %v", err)
		return ""
	}

	return intent.CorrelationID
}
