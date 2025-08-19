package loop

import (
	"errors"
	"log"
)

// SafeClose safely closes a Watcher instance with proper nil checking
// This is a helper function for proper resource cleanup patterns
func SafeClose(watcher *Watcher) error {
	if watcher == nil {
		log.Printf("SafeClose: watcher is nil - nothing to close")
		return nil
	}
	
	return watcher.Close()
}

// SafeCloserFunc returns a closure that safely closes a Watcher
// This is useful for defer statements to prevent nil pointer panics
func SafeCloserFunc(watcher *Watcher) func() {
	return func() {
		if err := SafeClose(watcher); err != nil {
			log.Printf("Error during safe close: %v", err)
		}
	}
}

// MustCreateWatcher creates a Watcher and panics on failure
// This is for cases where failure to create a watcher is unrecoverable
func MustCreateWatcher(dir string, config Config) *Watcher {
	watcher, err := NewWatcher(dir, config)
	if err != nil {
		log.Fatalf("Failed to create watcher: %v", err)
	}
	return watcher
}

// MustCreateWatcherWithProcessor creates a Watcher with processor and panics on failure
func MustCreateWatcherWithProcessor(dir string, processor *IntentProcessor) *Watcher {
	watcher, err := NewWatcherWithProcessor(dir, processor)
	if err != nil {
		log.Fatalf("Failed to create watcher with processor: %v", err)
	}
	return watcher
}

// SafeWatcherOperation executes an operation on a watcher with nil safety
func SafeWatcherOperation(watcher *Watcher, operation func(*Watcher) error) error {
	if watcher == nil {
		return ErrNilWatcher
	}
	return operation(watcher)
}

// Common errors
var (
	ErrNilWatcher = errors.New("watcher is nil")
)