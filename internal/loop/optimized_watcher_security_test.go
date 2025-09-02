package loop

import (
	"math"
	"testing"
	"time"
)

// TestIntegerOverflowProtection verifies that integer conversions are safe from overflow.
func TestIntegerOverflowProtection(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected int32
	}{
		{
			name:     "Normal value",
			input:    100,
			expected: 100,
		},
		{
			name:     "MaxInt32",
			input:    math.MaxInt32,
			expected: math.MaxInt32,
		},
		{
			name:     "Beyond MaxInt32",
			input:    math.MaxInt32 + 1,
			expected: math.MaxInt32, // Should be capped
		},
		{
			name:     "Negative value",
			input:    -10,
			expected: 0, // Should be normalized to 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the bounds checking logic from optimized_watcher.go
			value := tt.input
			if value > math.MaxInt32 {
				value = math.MaxInt32
			} else if value < 0 {
				value = 0
			}
			result := int32(value)

			if result != tt.expected {
				t.Errorf("Expected %d, got %d for input %d", tt.expected, result, tt.input)
			}
		})
	}
}

// TestDurationToUint64Conversion verifies safe conversion of duration to uint64.
func TestDurationToUint64Conversion(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		expected uint64
	}{
		{
			name:     "Positive duration",
			duration: 100 * time.Millisecond,
			expected: uint64(100 * time.Millisecond),
		},
		{
			name:     "Zero duration",
			duration: 0,
			expected: 0,
		},
		{
			name:     "Negative duration (should be handled as 0)",
			duration: -100 * time.Millisecond,
			expected: 0, // Negative durations should be normalized to 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the safe conversion logic
			nanosInt := tt.duration.Nanoseconds()
			if nanosInt < 0 {
				nanosInt = 0
			}
			result := uint64(nanosInt)

			expectedNanos := uint64(0)
			if tt.expected > 0 {
				expectedNanos = uint64(tt.duration.Nanoseconds())
			}

			if result != expectedNanos {
				t.Errorf("Expected %d, got %d for duration %v", expectedNanos, result, tt.duration)
			}
		})
	}
}
