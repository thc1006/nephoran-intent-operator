package rag

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// TestRAG bootstraps Ginkgo v2 test suite for the RAG (Retrieval-Augmented Generation) package.
// This package implements vector database operations, document processing, and embedding services
// for AI-enhanced network intent processing with Weaviate integration.
// RegisterFailHandler(Fail) connects Gomega to Ginkgo for comprehensive test reporting.
func TestRAG(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "RAG Suite")
}