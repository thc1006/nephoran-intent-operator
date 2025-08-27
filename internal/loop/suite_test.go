package loop

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestLoop bootstraps Ginkgo v2 test suite for the conductor loop package.
// This package implements the core polling-based intent processing loop with
// cross-platform file system operations and performance optimizations.
// RegisterFailHandler(Fail) connects Gomega assertions to Ginkgo's failure system.
func TestLoop(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Conductor Loop Suite")
}
