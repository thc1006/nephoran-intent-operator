package porch

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestPorch bootstraps Ginkgo v2 test suite for the porch package.
// This package manages Nephio Porch integration for GitOps package lifecycle.
// RegisterFailHandler(Fail) connects Gomega to Ginkgo's failure handling system.
func TestPorch(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Porch Suite")
}
