package loop

import (
	"fmt"
	"log"
)

// Example patterns for safe defer usage with Watcher

// Pattern 1: Defer after successful creation (RECOMMENDED)
func ExampleSafeDeferPattern1() error {
	watcher, err := NewWatcher("./handoff", Config{})
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Only register defer AFTER successful creation
	defer func() {
		if err := watcher.Close(); err != nil {
			log.Printf("Error closing watcher: %v", err)
		}
	}()

	// Use watcher...
	return watcher.Start()
}

// Pattern 2: Conditional defer with nil check (DEFENSIVE)
func ExampleSafeDeferPattern2() error {
	var watcher *Watcher
	var err error

	// Register defer early with nil safety
	defer func() {
		if watcher != nil {
			if err := watcher.Close(); err != nil {
				log.Printf("Error closing watcher: %v", err)
			}
		}
	}()

	// Create watcher (might fail)
	watcher, err = NewWatcher("./handoff", Config{})
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Use watcher...
	return watcher.Start()
}

// Pattern 3: Using helper function (CLEAN)
func ExampleSafeDeferPattern3() error {
	watcher, err := NewWatcher("./handoff", Config{})
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Use helper function for clean defer
	defer SafeCloserFunc(watcher)()

	// Use watcher...
	return watcher.Start()
}

// Pattern 4: Manual resource management (EXPLICIT)
func ExampleSafeDeferPattern4() error {
	watcher, err := NewWatcher("./handoff", Config{})
	if err != nil {
		return fmt.Errorf("failed to create watcher: %w", err)
	}

	// Explicit cleanup without defer
	watcherErr := watcher.Start()
	closeErr := watcher.Close()

	// Handle both errors
	if watcherErr != nil && closeErr != nil {
		return fmt.Errorf("watcher error: %w, close error: %v", watcherErr, closeErr)
	}
	if watcherErr != nil {
		return watcherErr
	}
	if closeErr != nil {
		return closeErr
	}

	return nil
}

// BAD PATTERN - DON'T DO THIS (causes nil pointer panic)
func ExampleBadDeferPattern() error {
	var watcher *Watcher
	defer watcher.Close() // ❌ DANGEROUS: Will panic if NewWatcher fails

	var err error
	watcher, err = NewWatcher("./handoff", Config{})
	if err != nil {
		// log.Fatalf() will call defer, which calls Close() on nil watcher -> PANIC!
		log.Fatalf("failed to create watcher: %v", err)
	}

	return watcher.Start()
}

// FIXED PATTERN - Safe version of the above
func ExampleFixedDeferPattern() error {
	var watcher *Watcher
	defer func() {
		// ✅ SAFE: Check for nil before calling Close()
		if watcher != nil {
			if err := watcher.Close(); err != nil {
				log.Printf("Error closing watcher: %v", err)
			}
		}
	}()

	var err error
	watcher, err = NewWatcher("./handoff", Config{})
	if err != nil {
		// Now safe to use log.Fatalf() - defer won't panic
		log.Fatalf("failed to create watcher: %v", err)
	}

	return watcher.Start()
}
