package unit

import (
	"context"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

// RAGComponentsTestSuite provides comprehensive unit testing for RAG components
type RAGComponentsTestSuite struct {
	suite.Suite
	ctx context.Context
}

// SetupSuite initializes the test suite
func (suite *RAGComponentsTestSuite) SetupSuite() {
	suite.ctx = context.Background()
}

// TestDocumentLoaderSuite tests document loader functionality
func (suite *RAGComponentsTestSuite) TestDocumentLoaderSuite() {
	suite.Run("LoadDocument", suite.testLoadDocument)
	suite.Run("LoadDocumentInvalidFormat", suite.testLoadDocumentInvalidFormat)
	suite.Run("LoadDocumentWithMetadata", suite.testLoadDocumentWithMetadata)
	suite.Run("LoadLargeDocument", suite.testLoadLargeDocument)
}

func (suite *RAGComponentsTestSuite) testLoadDocument() {
	loader := rag.NewDocumentLoader(&rag.DocumentLoaderConfig{
		SupportedFormats: []string{"txt", "md", "json"},
		MaxFileSize:      10 * 1024 * 1024, // 10MB
	})

	// Test valid document
	doc, err := loader.LoadDocument(suite.ctx, &rag.DocumentSource{
		ID:       "test-doc-1",
		Content:  "This is a test document content for loading verification.",
		Type:     "txt",
		Metadata: json.RawMessage(`{"author":"test"}`),
	})

	suite.NoError(err)
	suite.NotNil(doc)
	suite.Equal("test-doc-1", doc.ID)
	suite.Equal("This is a test document content for loading verification.", doc.Content)
	suite.Equal("txt", doc.Type)
	suite.Equal("test", doc.Metadata["author"])
}

func (suite *RAGComponentsTestSuite) testLoadDocumentInvalidFormat() {
	loader := rag.NewDocumentLoader(&rag.DocumentLoaderConfig{
		SupportedFormats: []string{"txt", "md"},
		MaxFileSize:      10 * 1024 * 1024,
	})

	// Test unsupported format
	_, err := loader.LoadDocument(suite.ctx, &rag.DocumentSource{
		ID:      "test-doc-invalid",
		Content: "content",
		Type:    "unsupported",
	})

	suite.Error(err)
	suite.Contains(err.Error(), "unsupported format")
}

func (suite *RAGComponentsTestSuite) testLoadDocumentWithMetadata() {
	loader := rag.NewDocumentLoader(&rag.DocumentLoaderConfig{
		SupportedFormats: []string{"json"},
		MaxFileSize:      10 * 1024 * 1024,
	})

	metadata := json.RawMessage(`{}`),
	}

	doc, err := loader.LoadDocument(suite.ctx, &rag.DocumentSource{
		ID:       "test-doc-meta",
		Content:  "Content with rich metadata",
		Type:     "json",
		Metadata: metadata,
	})

	suite.NoError(err)
	suite.Equal("technical", doc.Metadata["category"])
	suite.Equal("1.0", doc.Metadata["version"])
	tags, ok := doc.Metadata["tags"].([]string)
	suite.True(ok)
	suite.Contains(tags, "network")
	suite.Contains(tags, "5g")
}

func (suite *RAGComponentsTestSuite) testLoadLargeDocument() {
	loader := rag.NewDocumentLoader(&rag.DocumentLoaderConfig{
		SupportedFormats: []string{"txt"},
		MaxFileSize:      1024, // 1KB limit
	})

	// Test document exceeding size limit
	largeContent := make([]byte, 2048) // 2KB
	for i := range largeContent {
		largeContent[i] = 'A'
	}

	_, err := loader.LoadDocument(suite.ctx, &rag.DocumentSource{
		ID:      "large-doc",
		Content: string(largeContent),
		Type:    "txt",
	})

	suite.Error(err)
	suite.Contains(err.Error(), "exceeds maximum size")
}

// TestChunkingServiceSuite tests document chunking functionality
func (suite *RAGComponentsTestSuite) TestChunkingServiceSuite() {
	suite.Run("ChunkBySentence", suite.testChunkBySentence)
	suite.Run("ChunkByFixedSize", suite.testChunkByFixedSize)
	suite.Run("ChunkWithOverlap", suite.testChunkWithOverlap)
	suite.Run("ChunkEmptyDocument", suite.testChunkEmptyDocument)
	suite.Run("ChunkTelecomContent", suite.testChunkTelecomContent)
}

func (suite *RAGComponentsTestSuite) testChunkBySentence() {
	chunker := rag.NewChunkingService(&rag.ChunkingConfig{
		Strategy:     "sentence",
		MaxChunkSize: 200,
		Overlap:      20,
	})

	doc := rag.Document{
		ID:      "test-chunk-1",
		Content: "This is the first sentence. This is the second sentence with more content. This is the third sentence that concludes the test.",
		Type:    "txt",
	}

	chunks, err := chunker.ChunkDocument(suite.ctx, doc)

	suite.NoError(err)
	suite.GreaterOrEqual(len(chunks), 1)

	// Verify chunk properties
	for _, chunk := range chunks {
		suite.Equal(doc.ID, chunk.DocumentID)
		suite.LessOrEqual(len(chunk.Content), 200)
		suite.NotEmpty(chunk.Content)
		suite.GreaterOrEqual(chunk.ChunkIndex, 0)
	}

	// Verify sentence boundaries are respected
	for _, chunk := range chunks {
		// Should end with sentence punctuation or be the last chunk
		content := chunk.Content
		if chunk.ChunkIndex < len(chunks)-1 {
			suite.True(
				content[len(content)-1] == '.' ||
					content[len(content)-1] == '!' ||
					content[len(content)-1] == '?',
				"Chunk should end with sentence punctuation: %s", content,
			)
		}
	}
}

func (suite *RAGComponentsTestSuite) testChunkByFixedSize() {
	chunker := rag.NewChunkingService(&rag.ChunkingConfig{
		Strategy:     "fixed_size",
		MaxChunkSize: 50,
		Overlap:      10,
	})

	doc := rag.Document{
		ID:      "test-fixed-chunk",
		Content: "This is a longer document that will be split into fixed-size chunks regardless of sentence boundaries for testing purposes.",
		Type:    "txt",
	}

	chunks, err := chunker.ChunkDocument(suite.ctx, doc)

	suite.NoError(err)
	suite.GreaterOrEqual(len(chunks), 2)

	// Verify fixed size constraint
	for i, chunk := range chunks {
		if i < len(chunks)-1 { // Not the last chunk
			suite.Equal(50, len(chunk.Content))
		} else { // Last chunk can be smaller
			suite.LessOrEqual(len(chunk.Content), 50)
		}
	}
}

func (suite *RAGComponentsTestSuite) testChunkWithOverlap() {
	chunker := rag.NewChunkingService(&rag.ChunkingConfig{
		Strategy:     "fixed_size",
		MaxChunkSize: 30,
		Overlap:      10,
	})

	doc := rag.Document{
		ID:      "test-overlap",
		Content: "123456789012345678901234567890123456789012345678901234567890", // 60 chars
		Type:    "txt",
	}

	chunks, err := chunker.ChunkDocument(suite.ctx, doc)

	suite.NoError(err)
	suite.GreaterOrEqual(len(chunks), 2)

	// Verify overlap
	if len(chunks) >= 2 {
		chunk1 := chunks[0].Content
		chunk2 := chunks[1].Content

		// Extract overlap regions
		overlap1 := chunk1[len(chunk1)-10:]
		overlap2 := chunk2[:10]

		suite.Equal(overlap1, overlap2, "Chunks should have proper overlap")
	}
}

func (suite *RAGComponentsTestSuite) testChunkEmptyDocument() {
	chunker := rag.NewChunkingService(&rag.ChunkingConfig{
		Strategy:     "sentence",
		MaxChunkSize: 100,
		Overlap:      10,
	})

	doc := rag.Document{
		ID:      "empty-doc",
		Content: "",
		Type:    "txt",
	}

	chunks, err := chunker.ChunkDocument(suite.ctx, doc)

	suite.NoError(err)
	suite.Empty(chunks)
}

func (suite *RAGComponentsTestSuite) testChunkTelecomContent() {
	chunker := rag.NewChunkingService(&rag.ChunkingConfig{
		Strategy:     "sentence",
		MaxChunkSize: 300,
		Overlap:      50,
	})

	doc := rag.Document{
		ID: "telecom-doc",
		Content: `5G New Radio (NR) supports both Frequency Division Duplex (FDD) and Time Division Duplex (TDD) modes. 
		The gNodeB (gNB) is the base station in 5G networks, providing radio access functionality. 
		Network slicing allows multiple virtual networks to run on a single physical infrastructure. 
		The Core Network (5GC) includes functions like AMF, SMF, UPF, and PCF for different network operations.`,
		Type: "technical",
		Metadata: json.RawMessage(`{}`),
	}

	chunks, err := chunker.ChunkDocument(suite.ctx, doc)

	suite.NoError(err)
	suite.GreaterOrEqual(len(chunks), 1)

	// Verify telecom-specific content preservation
	allContent := ""
	for _, chunk := range chunks {
		allContent += chunk.Content
		suite.Equal(doc.Metadata, chunk.Metadata)
	}

	// Check that key telecom terms are preserved
	suite.Contains(allContent, "5G")
	suite.Contains(allContent, "gNodeB")
	suite.Contains(allContent, "network slicing")
	suite.Contains(allContent, "Core Network")
}

// TestEmbeddingServiceSuite tests embedding generation functionality
func (suite *RAGComponentsTestSuite) TestEmbeddingServiceSuite() {
	suite.Run("GenerateSingleEmbedding", suite.testGenerateSingleEmbedding)
	suite.Run("GenerateBatchEmbeddings", suite.testGenerateBatchEmbeddings)
	suite.Run("EmbeddingCaching", suite.testEmbeddingCaching)
	suite.Run("EmbeddingConsistency", suite.testEmbeddingConsistency)
	suite.Run("HandleEmptyInput", suite.testHandleEmptyInput)
}

func (suite *RAGComponentsTestSuite) testGenerateSingleEmbedding() {
	service := rag.NewEmbeddingService(&rag.EmbeddingConfig{
		ModelName:    "test-model",
		ModelPath:    "test-path",
		BatchSize:    32,
		CacheEnabled: false,
		MockMode:     true,
	})

	text := "This is a test sentence for embedding generation."
	embeddings, err := service.GenerateEmbeddings(suite.ctx, []string{text})

	suite.NoError(err)
	suite.Len(embeddings, 1)
	suite.Greater(len(embeddings[0]), 0)

	// Validate embedding properties
	embedding := embeddings[0]
	suite.True(len(embedding) > 100, "Embedding should have reasonable dimensionality")

	// Check if embedding is normalized (for typical models)
	var magnitude float32
	for _, val := range embedding {
		magnitude += val * val
	}
	magnitude = float32(math.Sqrt(float64(magnitude)))
	suite.InDelta(1.0, magnitude, 0.1, "Embedding should be approximately normalized")
}

func (suite *RAGComponentsTestSuite) testGenerateBatchEmbeddings() {
	service := rag.NewEmbeddingService(&rag.EmbeddingConfig{
		ModelName:    "test-model",
		BatchSize:    2,
		CacheEnabled: false,
		MockMode:     true,
	})

	texts := []string{
		"First text for batch processing.",
		"Second text for batch processing.",
		"Third text for batch processing.",
		"Fourth text for batch processing.",
	}

	embeddings, err := service.GenerateEmbeddings(suite.ctx, texts)

	suite.NoError(err)
	suite.Len(embeddings, len(texts))

	// Verify all embeddings have same dimensionality
	if len(embeddings) > 1 {
		expectedDim := len(embeddings[0])
		for i, emb := range embeddings {
			suite.Equal(expectedDim, len(emb), "Embedding %d has different dimensionality", i)
		}
	}

	// Verify embeddings are different
	if len(embeddings) >= 2 {
		suite.NotEqual(embeddings[0], embeddings[1], "Different texts should produce different embeddings")
	}
}

func (suite *RAGComponentsTestSuite) testEmbeddingCaching() {
	service := rag.NewEmbeddingService(&rag.EmbeddingConfig{
		ModelName:    "test-model",
		CacheEnabled: true,
		MockMode:     true,
	})

	text := "Cached embedding test text"

	// First generation (cache miss)
	start1 := time.Now()
	embeddings1, err1 := service.GenerateEmbeddings(suite.ctx, []string{text})
	duration1 := time.Since(start1)

	suite.NoError(err1)
	suite.Len(embeddings1, 1)

	// Second generation (cache hit)
	start2 := time.Now()
	embeddings2, err2 := service.GenerateEmbeddings(suite.ctx, []string{text})
	duration2 := time.Since(start2)

	suite.NoError(err2)
	suite.Len(embeddings2, 1)

	// Cache hit should be faster and return same result
	suite.True(duration2 < duration1/2, "Cached embedding should be significantly faster")
	suite.Equal(embeddings1[0], embeddings2[0], "Cached embedding should be identical")
}

func (suite *RAGComponentsTestSuite) testEmbeddingConsistency() {
	service := rag.NewEmbeddingService(&rag.EmbeddingConfig{
		ModelName:    "test-model",
		CacheEnabled: false,
		MockMode:     true,
	})

	text := "Consistency test text"

	// Generate multiple times
	var allEmbeddings [][]float32
	for i := 0; i < 3; i++ {
		embeddings, err := service.GenerateEmbeddings(suite.ctx, []string{text})
		suite.NoError(err)
		suite.Len(embeddings, 1)
		allEmbeddings = append(allEmbeddings, embeddings[0])
	}

	// In mock mode with deterministic generation, embeddings should be consistent
	for i := 1; i < len(allEmbeddings); i++ {
		suite.Equal(allEmbeddings[0], allEmbeddings[i], "Embeddings should be consistent for same input")
	}
}

func (suite *RAGComponentsTestSuite) testHandleEmptyInput() {
	service := rag.NewEmbeddingService(&rag.EmbeddingConfig{
		ModelName: "test-model",
		MockMode:  true,
	})

	// Test empty slice
	embeddings1, err1 := service.GenerateEmbeddings(suite.ctx, []string{})
	suite.NoError(err1)
	suite.Empty(embeddings1)

	// Test slice with empty string
	embeddings2, err2 := service.GenerateEmbeddings(suite.ctx, []string{""})
	suite.NoError(err2)
	suite.Len(embeddings2, 1)
	suite.Greater(len(embeddings2[0]), 0) // Should still generate embedding
}

// TestRetrievalServiceSuite tests document retrieval functionality
func (suite *RAGComponentsTestSuite) TestRetrievalServiceSuite() {
	suite.Run("SemanticSearch", suite.testSemanticSearch)
	suite.Run("HybridSearch", suite.testHybridSearch)
	suite.Run("FilterByScore", suite.testFilterByScore)
	suite.Run("ReRanking", suite.testReRanking)
	suite.Run("SearchWithMetadata", suite.testSearchWithMetadata)
}

func (suite *RAGComponentsTestSuite) testSemanticSearch() {
	// Mock Weaviate client
	mockWeaviate := &MockWeaviateClient{}
	mockWeaviate.On("SearchSimilar", mock.Anything, mock.Anything, mock.Anything).Return(
		[]rag.RetrievedDocument{
			{
				DocumentID:     "doc1",
				Title:          "5G Network Architecture",
				Content:        "5G networks use a service-based architecture...",
				Score:          0.95,
				RelevanceScore: 0.95,
			},
			{
				DocumentID:     "doc2",
				Title:          "Network Slicing",
				Content:        "Network slicing enables multiple virtual networks...",
				Score:          0.87,
				RelevanceScore: 0.87,
			},
		}, nil)

	service := rag.NewEnhancedRetrievalService(&rag.RetrievalConfig{
		MaxResults:       10,
		MinScore:         0.5,
		SemanticSearch:   true,
		RerankingEnabled: false,
		MockMode:         true,
	})
	service.SetWeaviateClient(mockWeaviate)

	results, err := service.RetrieveDocuments(suite.ctx, &rag.RetrievalRequest{
		Query:      "5G network architecture",
		MaxResults: 5,
		SearchType: "semantic",
		MinScore:   0.8,
	})

	suite.NoError(err)
	suite.Len(results, 2)
	suite.Equal("doc1", results[0].DocumentID)
	suite.GreaterOrEqual(results[0].Score, 0.8)

	// Results should be sorted by score
	for i := 1; i < len(results); i++ {
		suite.GreaterOrEqual(results[i-1].Score, results[i].Score)
	}

	mockWeaviate.AssertExpectations(suite.T())
}

func (suite *RAGComponentsTestSuite) testHybridSearch() {
	mockWeaviate := &MockWeaviateClient{}
	mockWeaviate.On("SearchSimilar", mock.Anything, mock.Anything, mock.Anything).Return(
		[]rag.RetrievedDocument{
			{DocumentID: "doc1", Score: 0.9},
			{DocumentID: "doc2", Score: 0.8},
		}, nil)

	service := rag.NewEnhancedRetrievalService(&rag.RetrievalConfig{
		HybridSearch:   true,
		SemanticSearch: true,
		KeywordWeight:  0.3,
		SemanticWeight: 0.7,
		MockMode:       true,
	})
	service.SetWeaviateClient(mockWeaviate)

	results, err := service.RetrieveDocuments(suite.ctx, &rag.RetrievalRequest{
		Query:      "network slicing configuration",
		SearchType: "hybrid",
		MaxResults: 10,
	})

	suite.NoError(err)
	suite.GreaterOrEqual(len(results), 1)

	// Hybrid search should combine semantic and keyword scores
	for _, result := range results {
		suite.GreaterOrEqual(result.Score, 0.0)
		suite.LessOrEqual(result.Score, 1.0)
	}

	mockWeaviate.AssertExpectations(suite.T())
}

func (suite *RAGComponentsTestSuite) testFilterByScore() {
	mockWeaviate := &MockWeaviateClient{}
	mockWeaviate.On("SearchSimilar", mock.Anything, mock.Anything, mock.Anything).Return(
		[]rag.RetrievedDocument{
			{DocumentID: "doc1", Score: 0.95},
			{DocumentID: "doc2", Score: 0.75},
			{DocumentID: "doc3", Score: 0.45}, // Below threshold
			{DocumentID: "doc4", Score: 0.35}, // Below threshold
		}, nil)

	service := rag.NewEnhancedRetrievalService(&rag.RetrievalConfig{
		MinScore: 0.5,
		MockMode: true,
	})
	service.SetWeaviateClient(mockWeaviate)

	results, err := service.RetrieveDocuments(suite.ctx, &rag.RetrievalRequest{
		Query:      "test query",
		MinScore:   0.5,
		MaxResults: 10,
	})

	suite.NoError(err)
	suite.Len(results, 2) // Only doc1 and doc2 should pass the filter

	for _, result := range results {
		suite.GreaterOrEqual(result.Score, 0.5)
	}

	mockWeaviate.AssertExpectations(suite.T())
}

func (suite *RAGComponentsTestSuite) testReRanking() {
	mockWeaviate := &MockWeaviateClient{}
	mockWeaviate.On("SearchSimilar", mock.Anything, mock.Anything, mock.Anything).Return(
		[]rag.RetrievedDocument{
			{DocumentID: "doc1", Content: "5G network configuration guide", Score: 0.8},
			{DocumentID: "doc2", Content: "Network slicing implementation", Score: 0.9},
			{DocumentID: "doc3", Content: "Legacy network protocols", Score: 0.7},
		}, nil)

	service := rag.NewEnhancedRetrievalService(&rag.RetrievalConfig{
		RerankingEnabled: true,
		MockMode:         true,
	})
	service.SetWeaviateClient(mockWeaviate)

	results, err := service.RetrieveDocuments(suite.ctx, &rag.RetrievalRequest{
		Query:      "5G configuration",
		SearchType: "semantic",
		MaxResults: 3,
	})

	suite.NoError(err)
	suite.Len(results, 3)

	// Re-ranking should potentially change order based on query relevance
	// Verify that re-ranking scores are set
	for _, result := range results {
		suite.GreaterOrEqual(result.RelevanceScore, 0.0)
		suite.LessOrEqual(result.RelevanceScore, 1.0)
	}

	mockWeaviate.AssertExpectations(suite.T())
}

func (suite *RAGComponentsTestSuite) testSearchWithMetadata() {
	mockWeaviate := &MockWeaviateClient{}
	mockWeaviate.On("SearchSimilar", mock.Anything, mock.Anything, mock.Anything).Return(
		[]rag.RetrievedDocument{
			{
				DocumentID: "doc1",
				Score:      0.9,
				Metadata: json.RawMessage(`{}`),
			},
		}, nil)

	service := rag.NewEnhancedRetrievalService(&rag.RetrievalConfig{
		MockMode: true,
	})
	service.SetWeaviateClient(mockWeaviate)

	results, err := service.RetrieveDocuments(suite.ctx, &rag.RetrievalRequest{
		Query: "5G core network",
		Filters: json.RawMessage(`{}`),
		MaxResults: 5,
	})

	suite.NoError(err)
	suite.Len(results, 1)

	result := results[0]
	suite.Equal("5g_core", result.Metadata["category"])
	suite.Equal("3gpp", result.Metadata["standard"])
	suite.Equal("17.0", result.Metadata["version"])

	mockWeaviate.AssertExpectations(suite.T())
}

// Test runner
func TestRAGComponents(t *testing.T) {
	suite.Run(t, new(RAGComponentsTestSuite))
}

// MockWeaviateClient for testing
type MockWeaviateClient struct {
	mock.Mock
}

func (m *MockWeaviateClient) SearchSimilar(ctx context.Context, query string, limit int) ([]rag.RetrievedDocument, error) {
	args := m.Called(ctx, query, limit)
	return args.Get(0).([]rag.RetrievedDocument), args.Error(1)
}

func (m *MockWeaviateClient) StoreDocument(ctx context.Context, doc rag.Document) error {
	args := m.Called(ctx, doc)
	return args.Error(0)
}

func (m *MockWeaviateClient) GetDocument(ctx context.Context, docID string) (*rag.Document, error) {
	args := m.Called(ctx, docID)
	return args.Get(0).(*rag.Document), args.Error(1)
}

func (m *MockWeaviateClient) DeleteDocument(ctx context.Context, docID string) error {
	args := m.Called(ctx, docID)
	return args.Error(0)
}

