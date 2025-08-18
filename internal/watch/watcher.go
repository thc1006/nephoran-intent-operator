package watch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// IntentFile represents the structure of an intent JSON file
type IntentFile struct {
	IntentType    string  `json:"intent_type"`
	Target        string  `json:"target"`
	Namespace     string  `json:"namespace"`
	Replicas      int     `json:"replicas"`
	Reason        *string `json:"reason,omitempty"`
	Source        *string `json:"source,omitempty"`
	CorrelationID *string `json:"correlation_id,omitempty"`
}

// Config holds watcher configuration
type Config struct {
	HandoffDir    string
	SchemaPath    string
	PostURL       string
	DebounceDelay time.Duration
}

// Watcher watches for intent files with debouncing and validation
type Watcher struct {
	config    *Config
	validator *Validator
	watcher   *fsnotify.Watcher
	
	// Debouncing
	mu         sync.Mutex
	pending    map[string]*time.Timer
	httpClient *http.Client
}

// NewWatcher creates a new file watcher
func NewWatcher(config *Config) (*Watcher, error) {
	// Set default debounce delay
	if config.DebounceDelay == 0 {
		config.DebounceDelay = 300 * time.Millisecond
	}

	// Create validator
	validator, err := NewValidator(config.SchemaPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}

	// Create fsnotify watcher
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	// Add directory to watch
	if err := fsWatcher.Add(config.HandoffDir); err != nil {
		fsWatcher.Close()
		return nil, fmt.Errorf("failed to add directory to watcher: %w", err)
	}

	return &Watcher{
		config:    config,
		validator: validator,
		watcher:   fsWatcher,
		pending:   make(map[string]*time.Timer),
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}, nil
}

// Start begins watching for file changes
func (w *Watcher) Start() error {
	// Process existing files first
	if err := w.processExistingFiles(); err != nil {
		log.Printf("Warning: Failed to process existing files: %v", err)
	}

	// Main event loop
	for {
		select {
		case event, ok := <-w.watcher.Events:
			if !ok {
				return fmt.Errorf("watcher events channel closed")
			}

			// Only process Create and Write events for intent files
			if event.Op&(fsnotify.Create|fsnotify.Write) != 0 {
				if w.isIntentFile(filepath.Base(event.Name)) {
					w.handleFileEvent(event)
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

// Stop stops the watcher
func (w *Watcher) Stop() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Cancel all pending timers
	for _, timer := range w.pending {
		timer.Stop()
	}

	return w.watcher.Close()
}

// handleFileEvent handles a file event with debouncing
func (w *Watcher) handleFileEvent(event fsnotify.Event) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Cancel existing timer for this file
	if timer, exists := w.pending[event.Name]; exists {
		timer.Stop()
	}

	// Create new debounced timer
	w.pending[event.Name] = time.AfterFunc(w.config.DebounceDelay, func() {
		w.processFile(event.Name, event.Op&fsnotify.Create != 0)
		
		// Clean up timer
		w.mu.Lock()
		delete(w.pending, event.Name)
		w.mu.Unlock()
	})
}

// processExistingFiles processes any existing intent files in the directory
func (w *Watcher) processExistingFiles() error {
	entries, err := os.ReadDir(w.config.HandoffDir)
	if err != nil {
		return err
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if w.isIntentFile(entry.Name()) {
			fullPath := filepath.Join(w.config.HandoffDir, entry.Name())
			w.processFile(fullPath, false)
			count++
		}
	}

	if count > 0 {
		log.Printf("Processed %d existing intent files on startup", count)
	}

	return nil
}

// isIntentFile checks if a filename matches the intent file pattern
func (w *Watcher) isIntentFile(filename string) bool {
	return strings.HasPrefix(filename, "intent-") && strings.HasSuffix(filename, ".json")
}

// processFile validates and optionally posts an intent file
func (w *Watcher) processFile(filePath string, isNew bool) {
	filename := filepath.Base(filePath)
	
	// Log new file detection
	if isNew {
		log.Printf("WATCH:NEW %s", filename)
	}

	// Read file content with retry for Windows file lock issues
	var data []byte
	var err error
	for attempts := 0; attempts < 3; attempts++ {
		data, err = os.ReadFile(filePath)
		if err == nil {
			break
		}
		if attempts < 2 {
			time.Sleep(time.Duration(50*(attempts+1)) * time.Millisecond) // 50ms, 100ms
		}
	}
	if err != nil {
		log.Printf("WATCH:ERROR Failed to read %s after 3 attempts: %v", filename, err)
		return
	}

	// Validate against schema
	if err := w.validator.Validate(data); err != nil {
		log.Printf("WATCH:INVALID %s - %v", filename, err)
		return
	}

	// Parse into struct for logging
	var intent IntentFile
	if err := json.Unmarshal(data, &intent); err != nil {
		log.Printf("WATCH:ERROR Failed to parse intent structure %s: %v", filename, err)
		return
	}

	// Log successful validation with structured summary
	log.Printf("WATCH:OK %s - type=%s target=%s namespace=%s replicas=%d",
		filename, intent.IntentType, intent.Target, intent.Namespace, intent.Replicas)

	// Optionally POST to HTTP endpoint
	if w.config.PostURL != "" {
		go w.postIntent(filePath, data)
	}
}

// postIntent sends the validated intent to an HTTP endpoint
func (w *Watcher) postIntent(filePath string, data []byte) {
	filename := filepath.Base(filePath)
	
	// Create HTTP request with context
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", w.config.PostURL, bytes.NewReader(data))
	if err != nil {
		log.Printf("WATCH:POST_ERROR %s - Failed to create request: %v", filename, err)
		return
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Intent-File", filename)
	req.Header.Set("X-Timestamp", time.Now().UTC().Format(time.RFC3339))

	// Send request
	resp, err := w.httpClient.Do(req)
	if err != nil {
		log.Printf("WATCH:POST_ERROR %s - Failed to POST: %v", filename, err)
		return
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("WATCH:POST_ERROR %s - Failed to read response: %v", filename, err)
		return
	}

	// Log result
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		log.Printf("WATCH:POST_OK %s - Status=%d Response=%s", filename, resp.StatusCode, string(body))
	} else {
		log.Printf("WATCH:POST_FAILED %s - Status=%d Response=%s", filename, resp.StatusCode, string(body))
	}
}