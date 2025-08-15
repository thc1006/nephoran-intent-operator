package loop

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/thc1006/nephoran-intent-operator/internal/porch"
)

// Watcher monitors a directory for intent file changes
type Watcher struct {
	watcher *fsnotify.Watcher
	dir     string
}

// NewWatcher creates a new file system watcher for the specified directory
func NewWatcher(dir string) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	// Add the directory to watch
	if err := watcher.Add(dir); err != nil {
		watcher.Close()
		return nil, fmt.Errorf("failed to add directory %s to watcher: %w", dir, err)
	}

	return &Watcher{
		watcher: watcher,
		dir:     dir,
	}, nil
}

// Start begins watching for file events
func (w *Watcher) Start() error {
	log.Printf("Watching directory: %s", w.dir)
	
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return fmt.Errorf("watcher events channel closed")
			}
			
			// Process only Create and Write events
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				// Check if this is an intent file
				if IsIntentFile(filepath.Base(event.Name)) {
					w.handleIntentFile(event)
				}
			}
			
		case err, ok := <-w.watcher.Errors:
			if !ok {
				return fmt.Errorf("watcher errors channel closed")
			}
			if err != nil {
				log.Printf("Watcher error: %v", err)
			}
		}
	}
}

// handleIntentFile processes detected intent files
func (w *Watcher) handleIntentFile(event fsnotify.Event) {
	// Log the detection
	operation := "UNKNOWN"
	if event.Op&fsnotify.Create != 0 {
		operation = "CREATE"
	} else if event.Op&fsnotify.Write != 0 {
		operation = "WRITE"
	}
	
	log.Printf("[%s] Intent file detected: %s", operation, event.Name)
	
	// Process the intent file
	if err := w.processIntent(event.Name); err != nil {
		log.Printf("Error processing intent file %s: %v", event.Name, err)
		w.writeStatusFile(event.Name, "failed", err.Error())
	} else {
		log.Printf("Successfully processed intent file: %s", event.Name)
		w.writeStatusFile(event.Name, "success", "Intent processed and sent to Porch")
	}
}

// processIntent reads the intent file and triggers Porch
func (w *Watcher) processIntent(filePath string) error {
	// Read the intent file
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read intent file: %w", err)
	}
	
	// Parse JSON intent
	var intent map[string]interface{}
	if err := json.Unmarshal(data, &intent); err != nil {
		return fmt.Errorf("failed to parse intent JSON: %w", err)
	}
	
	// Create output directory for Porch
	porchDir := filepath.Join(w.dir, "porch-output")
	if err := os.MkdirAll(porchDir, 0755); err != nil {
		return fmt.Errorf("failed to create porch output directory: %w", err)
	}
	
	// Write intent to Porch format (using the existing writer)
	if err := porch.WriteIntent(intent, porchDir, "full"); err != nil {
		return fmt.Errorf("failed to write Porch package: %w", err)
	}
	
	log.Printf("Intent written to Porch output directory: %s", porchDir)
	return nil
}

// writeStatusFile writes a status file after processing
func (w *Watcher) writeStatusFile(intentFile, status, message string) {
	statusData := map[string]interface{}{
		"intent_file": filepath.Base(intentFile),
		"status":      status,
		"message":     message,
		"timestamp":   time.Now().Format(time.RFC3339),
		"processed_by": "conductor-loop",
	}
	
	data, err := json.MarshalIndent(statusData, "", "  ")
	if err != nil {
		log.Printf("Failed to marshal status data: %v", err)
		return
	}
	
	// Create status filename based on intent filename
	baseName := filepath.Base(intentFile)
	statusFile := filepath.Join(w.dir, "status", baseName+".status")
	
	// Ensure status directory exists
	statusDir := filepath.Dir(statusFile)
	if err := os.MkdirAll(statusDir, 0755); err != nil {
		log.Printf("Failed to create status directory: %v", err)
		return
	}
	
	// Write status file
	if err := os.WriteFile(statusFile, data, 0644); err != nil {
		log.Printf("Failed to write status file: %v", err)
		return
	}
	
	log.Printf("Status written to: %s", statusFile)
}

// Close stops the watcher and releases resources
func (w *Watcher) Close() error {
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}