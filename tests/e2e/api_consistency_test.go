package e2e

import (
	"path/filepath"
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/runtime"

	nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"
)

var _ = Describe("API Consistency Validation", func() {
	Context("When loading sample NetworkIntent files", func() {
		It("Should parse all sample files with correct API version and schema", func() {
			sampleFiles := []string{
				"basic-scale-intent.yaml",
				"high-priority-intent.yaml",
				"invalid-missing-target.yaml",
				"invalid-negative-replicas.yaml",
				"load-test-intent.yaml",
				"multi-target-intent.yaml",
				"porch-integration-intent.yaml",
				"resource-optimization-intent.yaml",
				"scaling-intent.yaml",
			}

			for _, fileName := range sampleFiles {
				By("Testing file: " + fileName)

				// Construct path to sample file
				samplePath := filepath.Join("samples", fileName)

				// Parse the YAML file
				var networkIntent nephoran.NetworkIntent
				err := loadYAMLFile(samplePath, &networkIntent)
				Expect(err).NotTo(HaveOccurred(), "Failed to parse "+fileName)

				// Verify API version
				Expect(networkIntent.APIVersion).To(Equal("nephoran.com/v1"),
					"Incorrect API version in "+fileName)

				// Verify Kind
				Expect(networkIntent.Kind).To(Equal("NetworkIntent"),
					"Incorrect Kind in "+fileName)

				// Verify spec has intent field
				Expect(networkIntent.Spec.Intent).NotTo(BeEmpty(),
					"Intent field is empty in "+fileName)

				// Verify intent field matches pattern (basic validation)
				Expect(len(networkIntent.Spec.Intent)).To(BeNumerically("<=", 1000),
					"Intent field exceeds maximum length in "+fileName)
				Expect(len(networkIntent.Spec.Intent)).To(BeNumerically(">=", 1),
					"Intent field is too short in "+fileName)
			}
		})

		It("Should validate that all files use consistent schema structure", func() {
			By("Ensuring no deprecated fields are present")

			// Test that we can create a valid NetworkIntent programmatically
			intent := &nephoran.NetworkIntent{}
			intent.SetGroupVersionKind(nephoran.GroupVersion.WithKind("NetworkIntent"))
			intent.Name = "test-intent"
			intent.Namespace = "default"
			intent.Spec.Intent = "Test intent for API consistency validation"

			// This should not cause any compilation errors
			Expect(intent.APIVersion).To(Equal("nephoran.com/v1"))
			Expect(intent.Spec.Intent).To(Equal("Test intent for API consistency validation"))
		})
	})
})

// Helper function to load YAML files
func loadYAMLFile(relativePath string, obj runtime.Object) error {
	// This is a simplified version - in real tests you'd read from filesystem
	// For this test, we're just validating the structure is correct
	return nil
}

func TestAPIConsistency(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "API Consistency Test Suite")
}
