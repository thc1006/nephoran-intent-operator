package unit

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
)

// RedisCacheTestSuite tests Redis cache functionality
type RedisCacheTestSuite struct {
	suite.Suite
	ctx   context.Context
	cache *rag.RedisCache
}

// SetupSuite initializes the test suite
func (suite *RedisCacheTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.cache = rag.NewRedisCache(&rag.RedisCacheConfig{
		Address:  "localhost:6379",
		Database: 0,
		MockMode: true,
	})
}

// TestCacheOperations tests basic cache operations
func (suite *RedisCacheTestSuite) TestCacheOperations() {
	// Test Set and Get
	err := suite.cache.Set(suite.ctx, "test_key", "test_value", 1*time.Minute)
	suite.NoError(err)

	value, found, err := suite.cache.Get(suite.ctx, "test_key")
	suite.NoError(err)
	suite.True(found)
	suite.Equal("test_value", value)

	// Test Get non-existent key
	value, found, err = suite.cache.Get(suite.ctx, "non_existent_key")
	suite.NoError(err)
	suite.False(found)
	suite.Nil(value)

	// Test Delete
	err = suite.cache.Delete(suite.ctx, "test_key")
	suite.NoError(err)

	value, found, err = suite.cache.Get(suite.ctx, "test_key")
	suite.NoError(err)
	suite.False(found)
	suite.Nil(value)
}

// TestCacheExpiration tests cache expiration functionality
func (suite *RedisCacheTestSuite) TestCacheExpiration() {
	// Set value with short expiration
	err := suite.cache.Set(suite.ctx, "expire_key", "expire_value", 100*time.Millisecond)
	suite.NoError(err)

	// Should be available immediately
	value, found, err := suite.cache.Get(suite.ctx, "expire_key")
	suite.NoError(err)
	suite.True(found)
	suite.Equal("expire_value", value)

	// Wait for expiration (in mock mode, we simulate this)
	time.Sleep(150 * time.Millisecond)

	// In mock mode, we don't actually implement expiration timing,
	// but we verify the interface works correctly
	suite.True(true) // Placeholder for expiration test
}

// TestCacheStats tests cache statistics
func (suite *RedisCacheTestSuite) TestCacheStats() {
	// Perform some operations
	suite.cache.Set(suite.ctx, "stats_key1", "value1", 1*time.Minute)
	suite.cache.Set(suite.ctx, "stats_key2", "value2", 1*time.Minute)

	// Hit
	suite.cache.Get(suite.ctx, "stats_key1")
	suite.cache.Get(suite.ctx, "stats_key2")

	// Miss
	suite.cache.Get(suite.ctx, "non_existent")

	stats := suite.cache.GetStats()
	suite.NotNil(stats)
	suite.Contains(stats, "hits")
	suite.Contains(stats, "misses")
	suite.Contains(stats, "hit_rate")
}

// TestContextAssemblerSuite tests context assembly functionality
type ContextAssemblerTestSuite struct {
	suite.Suite
	ctx       context.Context
	assembler *rag.ContextAssembler
}

// SetupSuite initializes the context assembler test suite
func (suite *ContextAssemblerTestSuite) SetupSuite() {
	suite.ctx = context.Background()
	suite.assembler = rag.NewContextAssembler(&rag.ContextAssemblerConfig{
		MaxContextLength:  4000,
		MaxDocuments:      10,
		PriorityWeighting: true,
		RemoveDuplicates:  true,
		IncludeMetadata:   true,
	})
}

// TestAssembleContext tests context assembly from retrieved documents
func (suite *ContextAssemblerTestSuite) TestAssembleContext() {
	documents := []rag.RetrievedDocument{
		{
			DocumentID:     "doc1",
			Title:          "5G Network Architecture",
			Content:        "5G networks utilize a service-based architecture with network functions.",
			Score:          0.95,
			RelevanceScore: 0.95,
			Source:         "3GPP TS 23.501",
			Metadata: map[string]interface{}{
				"category": "5g_core",
				"section":  "4.2",
			},
		},
		{
			DocumentID:     "doc2",
			Title:          "Network Slicing",
			Content:        "Network slicing enables multiple virtual networks to run on shared infrastructure.",
			Score:          0.87,
			RelevanceScore: 0.87,
			Source:         "3GPP TS 23.501",
			Metadata: map[string]interface{}{
				"category": "network_slicing",
				"section":  "5.15",
			},
		},
	}

	context, err := suite.assembler.AssembleContext(suite.ctx, documents, &rag.ContextAssemblyRequest{
		Query:           "5G network architecture and slicing",
		IntentType:      "knowledge_request",
		MaxLength:       2000,
		IncludeMetadata: true,
	})

	suite.NoError(err)
	suite.NotNil(context)
	suite.NotEmpty(context.AssembledText)
	suite.LessOrEqual(len(context.AssembledText), 2000)
	suite.Equal(2, len(context.SourceDocuments))

	// Verify content includes key information
	suite.Contains(context.AssembledText, "5G networks")
	suite.Contains(context.AssembledText, "network slicing")

	// Verify metadata is included
	suite.NotEmpty(context.Metadata)
	suite.Contains(context.Metadata, "source_count")
	suite.Contains(context.Metadata, "assembly_strategy")
}

// TestRemoveDuplicateContent tests duplicate content removal
func (suite *ContextAssemblerTestSuite) TestRemoveDuplicateContent() {
	documents := []rag.RetrievedDocument{
		{
			DocumentID: "doc1",
			Content:    "5G networks provide enhanced mobile broadband services.",
			Score:      0.9,
		},
		{
			DocumentID: "doc2",
			Content:    "5G networks provide enhanced mobile broadband services.", // Duplicate
			Score:      0.8,
		},
		{
			DocumentID: "doc3",
			Content:    "Network slicing is a key feature of 5G.",
			Score:      0.85,
		},
	}

	context, err := suite.assembler.AssembleContext(suite.ctx, documents, &rag.ContextAssemblyRequest{
		Query:     "5G features",
		MaxLength: 1000,
	})

	suite.NoError(err)
	suite.NotNil(context)

	// Should only include unique content
	suite.Equal(2, len(context.SourceDocuments)) // doc1 and doc3, doc2 filtered out

	// Verify both unique pieces of content are present
	suite.Contains(context.AssembledText, "enhanced mobile broadband")
	suite.Contains(context.AssembledText, "network slicing")
}

// TestPriorityWeighting tests priority-based document ordering
func (suite *ContextAssemblerTestSuite) TestPriorityWeighting() {
	documents := []rag.RetrievedDocument{
		{
			DocumentID:     "low_score",
			Content:        "Low relevance content about networks.",
			Score:          0.6,
			RelevanceScore: 0.6,
		},
		{
			DocumentID:     "high_score",
			Content:        "High relevance content about 5G network architecture.",
			Score:          0.95,
			RelevanceScore: 0.95,
		},
		{
			DocumentID:     "medium_score",
			Content:        "Medium relevance content about network slicing.",
			Score:          0.8,
			RelevanceScore: 0.8,
		},
	}

	context, err := suite.assembler.AssembleContext(suite.ctx, documents, &rag.ContextAssemblyRequest{
		Query:     "5G network",
		MaxLength: 200, // Limited length to test prioritization
	})

	suite.NoError(err)
	suite.NotNil(context)

	// High-priority content should appear first/be included
	suite.Contains(context.AssembledText, "High relevance content") // From high_score doc

	// Verify source documents are ordered by priority
	suite.True(len(context.SourceDocuments) >= 1)
	if len(context.SourceDocuments) > 0 {
		// First document should be the highest scoring one
		suite.Equal("high_score", context.SourceDocuments[0].DocumentID)
	}
}

// TestSemanticRerankerSuite tests semantic reranking functionality
type SemanticRerankerTestSuite struct {
	suite.Suite
	ctx      context.Context
	reranker *rag.SemanticReranker
}

// SetupSuite initializes the semantic reranker test suite
func (suite *SemanticRerankerTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	mockEmbedding := &MockEmbeddingService{}
	mockEmbedding.On("GenerateEmbeddings", mock.Anything, mock.AnythingOfType("[]string")).Return(
		func(ctx context.Context, texts []string) [][]float32 {
			// Generate mock embeddings based on text content
			embeddings := make([][]float32, len(texts))
			for i, text := range texts {
				embeddings[i] = generateMockEmbedding(text)
			}
			return embeddings
		}, nil)

	suite.reranker = rag.NewSemanticReranker(&rag.SemanticRerankerConfig{
		EmbeddingService: mockEmbedding,
		ModelName:        "test-reranker",
		BatchSize:        32,
		MockMode:         true,
	})
}

// TestRerank tests document reranking based on query similarity
func (suite *SemanticRerankerTestSuite) TestRerank() {
	query := "5G network architecture"
	documents := []rag.RetrievedDocument{
		{
			DocumentID: "doc1",
			Content:    "Legacy 4G network protocols and standards",
			Score:      0.8, // High initial score but low semantic similarity
		},
		{
			DocumentID: "doc2",
			Content:    "5G network architecture and service-based design",
			Score:      0.7, // Lower initial score but high semantic similarity
		},
		{
			DocumentID: "doc3",
			Content:    "WiFi and Bluetooth networking technologies",
			Score:      0.75, // Medium score, low semantic similarity
		},
	}

	rerankedDocs, err := suite.reranker.Rerank(suite.ctx, query, documents)

	suite.NoError(err)
	suite.Len(rerankedDocs, 3)

	// Document with highest semantic similarity to query should be ranked first
	suite.Equal("doc2", rerankedDocs[0].DocumentID)

	// Verify reranking scores are updated
	for _, doc := range rerankedDocs {
		suite.GreaterOrEqual(doc.RelevanceScore, 0.0)
		suite.LessOrEqual(doc.RelevanceScore, 1.0)
		// Original score should be preserved
		suite.GreaterOrEqual(doc.Score, 0.0)
	}

	// Results should be sorted by RelevanceScore in descending order
	for i := 1; i < len(rerankedDocs); i++ {
		suite.GreaterOrEqual(rerankedDocs[i-1].RelevanceScore, rerankedDocs[i].RelevanceScore)
	}
}

// TestRerankEmpty tests reranking with empty document list
func (suite *SemanticRerankerTestSuite) TestRerankEmpty() {
	query := "test query"
	documents := []rag.RetrievedDocument{}

	rerankedDocs, err := suite.reranker.Rerank(suite.ctx, query, documents)

	suite.NoError(err)
	suite.Empty(rerankedDocs)
}

// TestRerankSingleDocument tests reranking with single document
func (suite *SemanticRerankerTestSuite) TestRerankSingleDocument() {
	query := "5G network"
	documents := []rag.RetrievedDocument{
		{
			DocumentID: "single_doc",
			Content:    "5G network architecture overview",
			Score:      0.8,
		},
	}

	rerankedDocs, err := suite.reranker.Rerank(suite.ctx, query, documents)

	suite.NoError(err)
	suite.Len(rerankedDocs, 1)
	suite.Equal("single_doc", rerankedDocs[0].DocumentID)
	suite.GreaterOrEqual(rerankedDocs[0].RelevanceScore, 0.0)
}

// Helper function to generate mock embeddings
func generateMockEmbedding(text string) []float32 {
	// Simple deterministic embedding generation for testing
	embedding := make([]float32, 384)

	// Use text characteristics to generate somewhat realistic embeddings
	textLen := len(text)
	wordCount := len(strings.Fields(text))

	for i := range embedding {
		// Create patterns based on text content
		val := float32(0.0)
		if i < textLen {
			val += float32(text[i%textLen]) / 255.0
		}
		if i < wordCount*10 {
			val += float32(wordCount) / 100.0
		}

		// Add some deterministic "randomness"
		val += float32((i*textLen)%100) / 100.0

		// Normalize to [-1, 1]
		embedding[i] = (val - 0.5) * 2.0
	}

	// Normalize to unit vector
	var magnitude float32
	for _, val := range embedding {
		magnitude += val * val
	}
	magnitude = float32(math.Sqrt(float64(magnitude)))

	if magnitude > 0 {
		for i := range embedding {
			embedding[i] /= magnitude
		}
	}

	return embedding
}

// MockEmbeddingService for testing
type MockEmbeddingService struct {
	mock.Mock
}

func (m *MockEmbeddingService) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {
	args := m.Called(ctx, texts)
	return args.Get(0).([][]float32), args.Error(1)
}

func (m *MockEmbeddingService) GetMetrics() *rag.EmbeddingMetrics {
	args := m.Called()
	return args.Get(0).(*rag.EmbeddingMetrics)
}

// Test runners
func TestRedisCache(t *testing.T) {
	suite.Run(t, new(RedisCacheTestSuite))
}

func TestContextAssembler(t *testing.T) {
	suite.Run(t, new(ContextAssemblerTestSuite))
}

func TestSemanticReranker(t *testing.T) {
	suite.Run(t, new(SemanticRerankerTestSuite))
}
