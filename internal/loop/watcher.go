package loop

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
)

// Watcher monitors a directory for intent file changes
type Watcher struct {
	watcher   *fsnotify.Watcher
	dir       string
	processor *IntentProcessor
}

// NewWatcher creates a new file system watcher for the specified directory (backward compatibility)
func NewWatcher(dir string) (*Watcher, error) {
	return NewWatcherWithProcessor(dir, nil)
}

// NewWatcherWithProcessor creates a new file system watcher with a processor
func NewWatcherWithProcessor(dir string, processor *IntentProcessor) (*Watcher, error) {
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
		watcher:   watcher,
		dir:       dir,
		processor: processor,
	}, nil
}

// ProcessExistingFiles processes any existing intent files in the directory (for restart idempotency)
func (w *Watcher) ProcessExistingFiles() error {
	if w.processor == nil {
		return nil // No processor, nothing to do
	}

	entries, err := os.ReadDir(w.dir)
	if err != nil {
		return fmt.Errorf("failed to read directory: %w", err)
	}

	count := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if IsIntentFile(entry.Name()) {
			fullPath := filepath.Join(w.dir, entry.Name())
			if w.processor != nil {
				if err := w.processor.ProcessFile(fullPath); err != nil {
					log.Printf("Error processing existing file %s: %v", fullPath, err)
					// Continue processing other files
				}
			}
			count++
		}
	}

	if count > 0 {
		log.Printf("Processed %d existing intent files", count)
		// Flush any batched files
		if w.processor != nil {
			if err := w.processor.FlushBatch(); err != nil {
				log.Printf("Error flushing batch after processing existing files: %v", err)
			}
		}
	}

	return nil
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
	operation := "UNKNOWN"
	if event.Op&fsnotify.Create != 0 {
		operation = "CREATE"
	} else if event.Op&fsnotify.Write != 0 {
		operation = "WRITE"
	}

	log.Printf("[%s] Intent file detected: %s", operation, event.Name)

	// If we have a processor, use it
	if w.processor != nil {
		if err := w.processor.ProcessFile(event.Name); err != nil {
			log.Printf("Error processing file %s: %v", event.Name, err)
		}
	}
}

// Close stops the watcher and releases resources
func (w *Watcher) Close() error {
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}

// IsIntentFile checks if a filename matches the intent file pattern
func IsIntentFile(filename string) bool {
	// Check if file matches intent-*.json pattern
	return strings.HasPrefix(filename, "intent-") && strings.HasSuffix(filename, ".json")
}
