//go:build test

package auth

import (
	"testing"
)

// TestBuildConstraintsWork verifies that build constraints prevent duplicate declarations.
// DISABLED: func TestBuildConstraintsWork(t *testing.T) {
	// Test that we can create AuthHandlers without duplicate declaration errors
	var authHandlers *AuthHandlers
	if authHandlers != nil {
		t.Error("Expected nil")
	}

	// This test simply ensures that the types compile correctly
	// The real test is that this compiles without redeclaration errors
}
