package orchestration

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestOrchestration bootstraps Ginkgo v2 test suite for the orchestration package.
// This package handles complex intent orchestration workflows for O-RAN/Nephio deployments.
// RegisterFailHandler(Fail) wires Gomega assertions to Ginkgo's failure reporting.
func TestOrchestration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Orchestration Suite")
}