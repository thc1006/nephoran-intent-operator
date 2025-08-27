package e2

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestE2 bootstraps Ginkgo v2 test suite for the O-RAN E2 interface package.
// This package implements E2 interface protocols for RAN management and monitoring.
// RegisterFailHandler(Fail) wires Gomega to Ginkgo for comprehensive test reporting.
func TestE2(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "O-RAN E2 Interface Suite")
}
