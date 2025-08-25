package loop

import (
	"fmt"
	"log"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
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
	// For MVP, just log the detection
	operation := "UNKNOWN"
	if event.Op&fsnotify.Create != 0 {
		operation = "CREATE"
	} else if event.Op&fsnotify.Write != 0 {
		operation = "WRITE"
	}

	log.Printf("[%s] Intent file detected: %s", operation, event.Name)

	// TODO: In future, trigger porch-publisher here
	// For now, this is just a placeholder comment
	// Example: publisherClient.PublishIntent(event.Name)
}

// Close stops the watcher and releases resources
func (w *Watcher) Close() error {
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}
