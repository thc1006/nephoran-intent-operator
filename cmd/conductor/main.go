package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/bep/debounce"
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
	defer watcher.Close()

	// Add handoff directory to watcher
	err = watcher.Add(handoffDir)
	if err != nil {
		log.Fatalf("Failed to add directory to watcher: %v", err)
	}

	// Create debounced file processors to handle rapid file changes
	// Map from file path to debounced processor
	processors := make(map[string]func())
	processorsMutex := &sync.RWMutex{}

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
						log.Printf("New intent file detected: %s (setting up debounced processing)", event.Name)
						
						// Create or get existing debounced processor for this file
						processorsMutex.Lock()
						processor, exists := processors[event.Name]
						if !exists {
							// Create new debounced processor with 500ms window
							filePath := event.Name // Capture for closure
							debouncedFunc := debounce.New(500 * time.Millisecond)
							processor = func() {
								debouncedFunc(func() {
									log.Printf("Processing debounced intent file: %s", filePath)
									processIntentFile(filePath, outDir)
									
									// Clean up processor after execution to prevent memory leaks
									processorsMutex.Lock()
									delete(processors, filePath)
									processorsMutex.Unlock()
								})
							}
							processors[event.Name] = processor
						}
						processorsMutex.Unlock()
						
						// Trigger debounced processing
						processor()
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

	// Give goroutines and debounced operations time to finish
	// Wait longer to ensure any pending debounced operations complete
	time.Sleep(1 * time.Second)
	
	// Log any remaining processors (should be cleaned up by now)
	processorsMutex.RLock()
	remainingProcessors := len(processors)
	processorsMutex.RUnlock()
	
	if remainingProcessors > 0 {
		log.Printf("Warning: %d debounced processors still pending during shutdown", remainingProcessors)
	}
	
	log.Println("Conductor stopped")
}

// isIntentFile checks if the file matches the pattern intent-*.json
func isIntentFile(path string) bool {
	filename := filepath.Base(path)
	return strings.HasPrefix(filename, "intent-") && strings.HasSuffix(filename, ".json")
}

// processIntentFile triggers porch-publisher for the given intent file
func processIntentFile(intentPath string, outDir string) {
	// SECURITY FIX: Validate and sanitize input paths to prevent command injection
	if !isValidPath(intentPath) {
		log.Printf("SECURITY VIOLATION: Invalid intent path rejected: %s", intentPath)
		return
	}
	if !isValidPath(outDir) {
		log.Printf("SECURITY VIOLATION: Invalid output directory rejected: %s", outDir)
		return
	}

	// Try to extract correlation_id from the intent file
	correlationID := extractCorrelationID(intentPath)
	if correlationID != "" {
		log.Printf("Processing intent with correlation_id: %s", correlationID)
	}

	// Build the command to run porch-publisher with sanitized paths
	// Use absolute paths to prevent directory traversal attacks
	absIntentPath, err := filepath.Abs(intentPath)
	if err != nil {
		log.Printf("Error resolving absolute path for intent: %v", err)
		return
	}
	absOutDir, err := filepath.Abs(outDir)
	if err != nil {
		log.Printf("Error resolving absolute path for output directory: %v", err)
		return
	}

	// Find porch-publisher binary path
	publisherBinary := findPorchPublisherBinary()
	if publisherBinary == "" {
		log.Printf("Error: porch-publisher binary not found. Please run 'make build-porch-publisher' first.")
		return
	}

	cmd := exec.Command(publisherBinary, "-intent", absIntentPath, "-out", absOutDir)

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

// isValidPath validates file paths to prevent command injection and directory traversal
func isValidPath(path string) bool {
	// Reject paths containing dangerous characters or patterns
	dangerousPatterns := []string{
		";", "|", "&", "$", "`", "$(", "${", // Command injection
		"../", "..\\", // Directory traversal
		"\x00", // Null byte
	}

	for _, pattern := range dangerousPatterns {
		if strings.Contains(path, pattern) {
			return false
		}
	}

	// Only allow alphanumeric, dash, underscore, dot, slash, and backslash
	validPathRegex := regexp.MustCompile(`^[a-zA-Z0-9._/\-\\:]+$`)
	if !validPathRegex.MatchString(path) {
		return false
	}

	// Reject paths that are too long (potential buffer overflow)
	if len(path) > 4096 {
		return false
	}

	return true
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

// findPorchPublisherBinary locates the porch-publisher binary with fallback options
func findPorchPublisherBinary() string {
	// Define potential binary paths in order of preference
	candidates := []string{
		"bin/porch-publisher",         // Relative to project root (most common)
		"./bin/porch-publisher",       // Explicit relative path
		"porch-publisher",             // In PATH
	}

	// On Windows, also check .exe variants
	if runtime.GOOS == "windows" {
		windowsCandidates := make([]string, 0, len(candidates)*2)
		for _, candidate := range candidates {
			windowsCandidates = append(windowsCandidates, candidate)
			windowsCandidates = append(windowsCandidates, candidate+".exe")
		}
		candidates = windowsCandidates
	}

	// Try each candidate path
	for _, candidate := range candidates {
		// Convert relative paths to absolute for consistency
		absPath, err := filepath.Abs(candidate)
		if err != nil {
			continue
		}

		// Check if the file exists and is executable
		if fileInfo, err := os.Stat(absPath); err == nil && !fileInfo.IsDir() {
			// On Windows, any file is considered executable
			// On Unix-like systems, check executable bit
			if runtime.GOOS == "windows" || fileInfo.Mode()&0111 != 0 {
				log.Printf("Found porch-publisher binary at: %s", absPath)
				// For relative paths, use the original candidate to avoid Windows path issues
				if !filepath.IsAbs(candidate) {
					return candidate
				}
				return absPath
			}
		}

		// Also try the candidate as-is (for PATH lookups)
		if candidate == "porch-publisher" || (runtime.GOOS == "windows" && candidate == "porch-publisher.exe") {
			if _, err := exec.LookPath(candidate); err == nil {
				log.Printf("Found porch-publisher binary in PATH: %s", candidate)
				return candidate
			}
		}
	}

	// If not found, try to build it automatically
	log.Printf("porch-publisher binary not found, attempting to build...")
	if err := buildPorchPublisher(); err != nil {
		log.Printf("Failed to build porch-publisher: %v", err)
		return ""
	}

	// Try again after building
	primaryPath := "bin/porch-publisher"
	if runtime.GOOS == "windows" {
		primaryPath += ".exe"
	}

	if absPath, err := filepath.Abs(primaryPath); err == nil {
		if _, err := os.Stat(absPath); err == nil {
			log.Printf("Successfully built and found porch-publisher at: %s", absPath)
			return absPath
		}
	}

	return ""
}

// buildPorchPublisher attempts to build the porch-publisher binary using go build
func buildPorchPublisher() error {
	log.Printf("Building porch-publisher binary...")

	// Create bin directory if it doesn't exist
	if err := os.MkdirAll("bin", 0755); err != nil {
		return err
	}

	// Determine output binary name
	outputPath := "bin/porch-publisher"
	if runtime.GOOS == "windows" {
		outputPath += ".exe"
	}

	// Build command
	cmd := exec.Command("go", "build", "-o", outputPath, "./cmd/porch-publisher")
	cmd.Env = os.Environ()

	// Run the build
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("build failed: %v\nOutput: %s", err, string(output))
	}

	log.Printf("Successfully built porch-publisher: %s", outputPath)
	return nil
}
