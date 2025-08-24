package conductor

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestConductor bootstraps Ginkgo v2 test suite for the conductor package.
// RegisterFailHandler(Fail) wires Gomega to Ginkgo's failure reporting system.
func TestConductor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Conductor Suite")
}