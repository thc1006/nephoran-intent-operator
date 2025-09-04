package loop

import (
	"strings"
	"testing"
)

// assertErrorContainsAny checks if an error contains any of the provided substrings.

// This helper is useful for cross-platform compatibility where error messages.

// may vary slightly between operating systems while maintaining the same security guarantees.

func assertErrorContainsAny(t *testing.T, err error, candidates ...string) {
	t.Helper()

	if err == nil {

		t.Errorf("Expected error containing one of %v, but got nil", candidates)

		return

	}

	errStr := err.Error()

	for _, candidate := range candidates {
		if strings.Contains(errStr, candidate) {
			// Found a match, test passes.

			return
		}
	}

	// No match found.

	t.Errorf("Expected error to contain one of %v, but got: %s", candidates, errStr)
}

// assertErrorContainsAnyWithDescription is like assertErrorContainsAny but includes a description.

func assertErrorContainsAnyWithDescription(t *testing.T, err error, description string, candidates ...string) {
	t.Helper()

	if err == nil {

		t.Errorf("%s: Expected error containing one of %v, but got nil", description, candidates)

		return

	}

	errStr := err.Error()

	for _, candidate := range candidates {
		if strings.Contains(errStr, candidate) {
			// Found a match, test passes.

			return
		}
	}

	// No match found.

	t.Errorf("%s: Expected error to contain one of %v, but got: %s", description, candidates, errStr)
}
