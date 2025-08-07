package excellence_test

import (
	"os"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestExcellenceValidation(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Excellence Validation Suite")
}

var _ = BeforeSuite(func() {
	// Setup test environment for excellence validation
	GinkgoWriter.Println("Setting up excellence validation test environment...")

	// Ensure test directories exist
	testDirs := []string{
		"../../.excellence-reports",
		"../../.test-artifacts",
	}

	for _, dir := range testDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			GinkgoWriter.Printf("Warning: Could not create directory %s: %v\n", dir, err)
		}
	}

	GinkgoWriter.Println("Excellence validation test environment ready")
})

var _ = AfterSuite(func() {
	GinkgoWriter.Println("Cleaning up excellence validation test environment...")
	// Cleanup is handled by individual tests to preserve artifacts for analysis
})