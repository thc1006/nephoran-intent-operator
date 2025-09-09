//go:build test_imports
// +build test_imports

// Package main provides import testing for dependency verification.
// This file ensures all critical dependencies can be imported without conflicts.
//
// Usage: go build -tags=test_imports
package main

import (
	"encoding/json"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/porch"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/knowledge" 
	_ "github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch/porchtest"
	_ "github.com/thc1006/nephoran-intent-operator/internal/patch"
)

// Test imports main function - only compiled with test_imports build tag
func main() {
	// Verify json import works
	var _ json.RawMessage
}
