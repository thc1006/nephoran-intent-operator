package main

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestLLMProcessor bootstraps Ginkgo v2 test suite for the llm-processor package.
// This function wires Gomega to Ginkgo's Fail handler and runs all specs in the package.
// RegisterFailHandler(Fail) connects Gomega assertions to Ginkgo's failure reporting.
func TestLLMProcessor(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LLM Processor Suite")
}
