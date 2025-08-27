package a1

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestA1 bootstraps Ginkgo v2 test suite for the O-RAN A1 interface package.
// This package implements A1 policy management interface for O-RAN compliance.
// RegisterFailHandler(Fail) ensures Gomega assertions integrate with Ginkgo properly.
func TestA1(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "O-RAN A1 Interface Suite")
}
