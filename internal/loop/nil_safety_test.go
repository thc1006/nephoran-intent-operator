package loop

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestNilWatcherSafety validates Fix 1: Nil pointer dereference protection
func TestNilWatcherSafety(t *testing.T) {
	t.Run("nil_watcher_close", func(t *testing.T) {
		// Test that Close() on nil watcher does not panic or error
		var nilWatcher *Watcher
		err := nilWatcher.Close()
		assert.NoError(t, err, "Close() on nil Watcher should not panic or error")
	})
	
	t.Run("nil_watcher_methods", func(t *testing.T) {
		var nilWatcher *Watcher
		
		// Test other methods that should be safe with nil receiver
		metrics := nilWatcher.GetMetrics()
		assert.Nil(t, metrics, "GetMetrics() on nil Watcher should return nil")
		
		// Close should be idempotent and safe
		err1 := nilWatcher.Close()
		err2 := nilWatcher.Close()
		assert.NoError(t, err1, "First Close() should not error")
		assert.NoError(t, err2, "Second Close() should not error")
	})
}

// TestIsIntentFileFunction tests the intent file detection function
func TestIsIntentFileFunction(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"valid_intent_file", "intent-test.json", true},
		{"valid_intent_with_numbers", "intent-123.json", true},
		{"invalid_prefix", "test-intent.json", false},
		{"invalid_extension", "intent-test.txt", false},
		{"no_extension", "intent-test", false},
		{"empty_filename", "", false},
		{"just_prefix", "intent-", false},
		{"just_extension", ".json", false},
		{"valid_complex", "intent-my-test-file.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsIntentFile(tt.filename)
			assert.Equal(t, tt.expected, result, 
				"IsIntentFile(%s) should return %t", tt.filename, tt.expected)
		})
	}
}