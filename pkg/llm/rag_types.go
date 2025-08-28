//go:build !disable_rag
// +build !disable_rag

package llm

// This file provides type aliases and re-exports for RAG types to resolve compilation issues
// It acts as a bridge between the pkg/rag and pkg/llm packages

import (
	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// Type aliases for rag package types - all types actually exist in pkg/rag

// WeaviateConnectionPool is re-exported from rag package
type WeaviateConnectionPool = rag.WeaviateConnectionPool

// EmbeddingServiceInterface is re-exported from rag package  
type EmbeddingServiceInterface = rag.EmbeddingServiceInterface

// EmbeddingService is re-exported from rag package
type EmbeddingService = rag.EmbeddingService

// WeaviateClient is re-exported from rag package
type WeaviateClient = rag.WeaviateClient

// TelecomDocument is re-exported from shared package (it's defined there)
type TelecomDocument = shared.TelecomDocument

// SearchResponse is re-exported from shared package (it's defined there)
type SearchResponse = shared.SearchResponse

// SearchResult is re-exported from shared package
type SearchResult = shared.SearchResult

// RAGService is re-exported from rag package
type RAGService = rag.RAGService

// RAGRequest is re-exported from rag package
type RAGRequest = rag.RAGRequest

// RAGResponse is re-exported from rag package
type RAGResponse = rag.RAGResponse

// SearchQuery is re-exported from shared package
type SearchQuery = shared.SearchQuery

// Type aliases are working correctly - no additional type assertions needed since
// all types exist and are correctly defined in their respective packages