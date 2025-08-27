package loop

import (
	"testing"
)

// TestNilWatcherClose verifies that calling Close() on a nil watcher doesn't panic
func TestNilWatcherClose(t *testing.T) {
	// This test ensures the nil pointer panic is fixed
	var watcher *Watcher = nil

	// This should not panic after our fix
	err := watcher.Close()
	if err != nil {
		t.Errorf("Expected nil error when closing nil watcher, got: %v", err)
	}
}

// TestSafeClose verifies the SafeClose helper function
func TestSafeClose(t *testing.T) {
	tests := []struct {
		name    string
		watcher *Watcher
		wantErr bool
	}{
		{
			name:    "nil watcher",
			watcher: nil,
			wantErr: false,
		},
		{
			name:    "valid watcher",
			watcher: &Watcher{}, // minimal instance for testing
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := SafeClose(tt.watcher)
			if (err != nil) != tt.wantErr {
				t.Errorf("SafeClose() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

// TestSafeCloserFunc verifies the SafeCloserFunc helper
func TestSafeCloserFunc(t *testing.T) {
	// Test with nil watcher
	closer := SafeCloserFunc(nil)

	// This should not panic
	closer()

	// Test with valid watcher
	watcher := &Watcher{}
	closer2 := SafeCloserFunc(watcher)
	closer2()
}

// TestSafeWatcherOperation verifies the SafeWatcherOperation helper
func TestSafeWatcherOperation(t *testing.T) {
	// Test with nil watcher
	err := SafeWatcherOperation(nil, func(w *Watcher) error {
		return nil
	})

	if err != ErrNilWatcher {
		t.Errorf("Expected ErrNilWatcher, got: %v", err)
	}

	// Test with valid watcher
	watcher := &Watcher{}
	called := false

	err = SafeWatcherOperation(watcher, func(w *Watcher) error {
		called = true
		return nil
	})

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if !called {
		t.Error("Operation was not called")
	}
}

// TestDeferPatterns demonstrates safe defer patterns
func TestDeferPatterns(t *testing.T) {
	// Pattern 1: Immediate defer with nil check (what we implemented)
	func() {
		var watcher *Watcher = nil
		defer func() {
			if watcher != nil {
				if err := watcher.Close(); err != nil {
					t.Logf("Error closing watcher: %v", err)
				}
			}
		}()

		// Simulate initialization failure
		// watcher remains nil, defer should handle it gracefully
	}()

	// Pattern 2: Using SafeCloserFunc
	func() {
		var watcher *Watcher = nil
		defer SafeCloserFunc(watcher)()

		// Simulate initialization failure
		// defer should handle nil gracefully
	}()

	// Pattern 3: Defer after successful creation
	func() {
		watcher := &Watcher{} // Simulate successful creation
		defer func() {
			if err := watcher.Close(); err != nil {
				t.Logf("Error closing watcher: %v", err)
			}
		}()

		// Normal operation...
	}()
}
