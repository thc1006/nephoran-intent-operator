package integration

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestIntegration bootstraps Ginkgo v2 test suite for integration tests.
// This package contains end-to-end integration tests for the full O-RAN/Nephio workflow.
// RegisterFailHandler(Fail) ensures proper failure handling for complex integration scenarios.
func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Test Suite")
}