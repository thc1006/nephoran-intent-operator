package loop

import (
	"testing"
	"time"
)

// WaitFor polls a condition function until it returns true or timeout occurs
func WaitFor(t *testing.T, condition func() bool, timeout time.Duration, description string) bool {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return true
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				t.Logf("Timeout waiting for: %s", description)
				return false
			}
		}
	}
}

// WaitForValue waits for a function to return an expected value
func WaitForValue[T comparable](t *testing.T, getter func() T, expected T, timeout time.Duration, description string) bool {
	t.Helper()
	return WaitFor(t, func() bool {
		return getter() == expected
	}, timeout, description)
}
