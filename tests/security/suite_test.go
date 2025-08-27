package security

import (
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	testutils "github.com/thc1006/nephoran-intent-operator/tests/utils"
)

// TestSecuritySuite runs the complete security test suite
func TestSecuritySuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Nephoran Intent Operator Security Test Suite")
}

var _ = BeforeSuite(func() {
	By("Setting up test environment")

	// Setup test environment
	err := testutils.SetupTestEnvironment()
	Expect(err).NotTo(HaveOccurred())

	// Wait for environment to be ready
	time.Sleep(5 * time.Second)

	By("Test environment setup completed")
})

var _ = AfterSuite(func() {
	By("Tearing down test environment")

	err := testutils.TeardownTestEnvironment()
	Expect(err).NotTo(HaveOccurred())

	By("Test environment cleanup completed")
})
