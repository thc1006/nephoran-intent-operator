//go:build test_imports
// +build test_imports

// Package main provides import testing for dependency verification.
// This file ensures all critical dependencies can be imported without conflicts.
//
// Usage: go build -tags=test_imports
package main

import (
<<<<<<< HEAD
	_ "github.com/onsi/ginkgo/v2"
	_ "github.com/onsi/gomega"
	_ "k8s.io/client-go/kubernetes/scheme"
	_ "sigs.k8s.io/controller-runtime/pkg/client"
	_ "sigs.k8s.io/controller-runtime/pkg/envtest"
	
	_ "github.com/thc1006/nephoran-intent-operator/api/v1alpha1"
=======
	"encoding/json"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/porch"
	_ "github.com/thc1006/nephoran-intent-operator/pkg/knowledge" 
	_ "github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch/porchtest"
	_ "github.com/thc1006/nephoran-intent-operator/internal/patch"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
)

func main() {
	// This is a test-only file for import verification
}
