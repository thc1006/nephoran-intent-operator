package llm

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestLLM bootstraps Ginkgo v2 test suite for the llm package.
// RegisterFailHandler(Fail) ensures Gomega assertions work with Ginkgo's failure handler.
func TestLLM(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LLM Suite")
}
