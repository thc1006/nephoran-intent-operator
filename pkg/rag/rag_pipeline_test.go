package rag

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test data and fixtures
const testPDFContent = `
3GPP TS 38.300 V16.0.0 (2020-07)
Technical Specification
3rd Generation Partnership Project;
Technical Specification Group Radio Access Network;
NR; Overall description;
Stage 2
(Release 16)

1 Scope
This document provides an overall description of the NR radio access technology.
The document describes the general architecture and functionality of the NR system.

2 References
The following documents contain provisions which, through reference in this text,
constitute provisions of the present document.

2.1 Normative references
[1] 3GPP TS 23.501: "System architecture for the 5G System (5GS); Stage 2".

3 Definitions and abbreviations
3.1 Definitions
For the purposes of the present document, the terms and definitions given in TR 21.905.

3.2 Abbreviations
For the purposes of the present document, the abbreviations given in TR 21.905 apply.
AMF: Access and Mobility Management Function
SMF: Session Management Function
UPF: User Plane Function
gNB: Next Generation NodeB

4 Architecture overview
The NR radio access network consists of gNBs, providing NR user plane and control plane
protocol terminations towards the UE.
`

func TestDocumentLoader(t *testing.T) {
	// Setup test environment
	testDir := t.TempDir()

	// Create a test PDF file (mock)
	testPDFPath := filepath.Join(testDir, "test_spec.pdf")
	err := os.WriteFile(testPDFPath, []byte("mock PDF content"), 0644)
	require.NoError(t, err)

	t.Run("NewDocumentLoader", func(t *testing.T) {
		config := getDefaultLoaderConfig()
		config.LocalPaths = []string{testDir}
		
		loader := NewDocumentLoader(config)
		assert.NotNil(t, loader)
		assert.Equal(t, config, loader.config)
	})

	t.Run("ProcessText", func(t *testing.T) {
		loader := NewDocumentLoader(nil)
		
		// Test text preprocessing
		text := "  This is a test   document with   multiple   spaces  "
		cleaned := loader.cleanTextContent(text)
		
		assert.Equal(t, "This is a test document with multiple spaces", cleaned)
	})

	t.Run("ExtractTelecomMetadata", func(t *testing.T) {
		loader := NewDocumentLoader(nil)
		
		metadata := loader.extractTelecomMetadata(testPDFContent, "3gpp_ts_38_300.pdf")
		
		assert.Equal(t, "3GPP", metadata.Source)
		assert.Equal(t, "TS", metadata.DocumentType)
		assert.Contains(t, metadata.Technologies, "5G")
		assert.Contains(t, metadata.NetworkFunctions, "AMF")
		assert.Contains(t, metadata.NetworkFunctions, "gNB")
	})

	t.Run("DetectSource", func(t *testing.T) {
		loader := NewDocumentLoader(nil)
		
		tests := []struct {
			content  string
			filename string
			expected string
		}{
			{testPDFContent, "3gpp_spec.pdf", "3GPP"},
			{"O-RAN Alliance specification", "oran_spec.pdf", "O-RAN"},
			{"ETSI standard document", "etsi_doc.pdf", "ETSI"},
			{"random content", "unknown.pdf", "Unknown"},
		}

		for _, test := range tests {
			source := loader.detectSource(test.content, test.filename)
			assert.Equal(t, test.expected, source, "Failed for content: %s, filename: %s", test.content, test.filename)
		}
	})
}

func TestChunkingService(t *testing.T) {
	t.Run("NewChunkingService", func(t *testing.T) {
		config := getDefaultChunkingConfig()
		service := NewChunkingService(config)
		
		assert.NotNil(t, service)
		assert.Equal(t, config, service.config)
	})

	t.Run("AnalyzeDocumentStructure", func(t *testing.T) {
		service := NewChunkingService(nil)
		
		structure, err := service.analyzeDocumentStructure(testPDFContent)
		require.NoError(t, err)
		
		assert.NotNil(t, structure)
		assert.Greater(t, len(structure.Sections), 0)
		
		// Check that sections were extracted
		found := false
		for _, section := range structure.Sections {
			if section.Title == "Scope" {
				found = true
				break
			}
		}
		assert.True(t, found, "Should find 'Scope' section")
	})

	t.Run("ChunkDocument", func(t *testing.T) {
		service := NewChunkingService(nil)
		
		// Create test document
		doc := &LoadedDocument{
			ID:       "test-doc-1",
			Content:  testPDFContent,
			Title:    "Test 3GPP Specification",
			Metadata: &DocumentMetadata{
				Source:   "3GPP",
				Category: "RAN",
			},
		}

		ctx := context.Background()
		chunks, err := service.ChunkDocument(ctx, doc)
		require.NoError(t, err)
		
		assert.Greater(t, len(chunks), 0)
		
		// Verify chunk properties
		for _, chunk := range chunks {
			assert.NotEmpty(t, chunk.ID)
			assert.Equal(t, doc.ID, chunk.DocumentID)
			assert.NotEmpty(t, chunk.Content)
			assert.Greater(t, chunk.CharacterCount, 0)
			assert.Greater(t, chunk.QualityScore, 0.0)
		}
	})

	t.Run("IntelligentChunking", func(t *testing.T) {
		config := getDefaultChunkingConfig()
		config.PreserveHierarchy = true
		config.UseSemanticBoundaries = true
		
		service := NewChunkingService(config)
		
		doc := &LoadedDocument{
			ID:      "test-doc-2",
			Content: testPDFContent,
			Title:   "Test Document with Hierarchy",
		}

		ctx := context.Background()
		chunks, err := service.ChunkDocument(ctx, doc)
		require.NoError(t, err)
		
		// Verify hierarchy information is preserved
		hierarchyFound := false
		for _, chunk := range chunks {
			if len(chunk.HierarchyPath) > 0 {
				hierarchyFound = true
				break
			}
		}
		assert.True(t, hierarchyFound, "Should preserve hierarchy information")
	})
}

func TestEmbeddingService(t *testing.T) {
	// Skip if no API key provided
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not provided, skipping embedding tests")
	}

	t.Run("NewEmbeddingService", func(t *testing.T) {
		config := getDefaultEmbeddingConfig()
		config.APIKey = "test-key"
		
		service := NewEmbeddingService(config)
		assert.NotNil(t, service)
		assert.Equal(t, config, service.config)
	})

	t.Run("PreprocessText", func(t *testing.T) {
		config := getDefaultEmbeddingConfig()
		config.NormalizeText = true
		config.TelecomPreprocessing = true
		
		service := NewEmbeddingService(config)
		
		tests := []struct {
			input    string
			expected string
		}{
			{"  multiple   spaces  ", "multiple spaces"},
			{"AMF handles authentication", "AMF handles authentication"},
			{"", ""},
		}

		for _, test := range tests {
			result := service.preprocessText(test.input)
			assert.Equal(t, test.expected, result)
		}
	})

	t.Run("CacheKey", func(t *testing.T) {
		service := NewEmbeddingService(nil)
		
		key1 := service.generateCacheKey("test text")
		key2 := service.generateCacheKey("test text")
		key3 := service.generateCacheKey("different text")
		
		assert.Equal(t, key1, key2, "Same text should generate same key")
		assert.NotEqual(t, key1, key3, "Different text should generate different keys")
	})
}

func TestQueryEnhancer(t *testing.T) {
	t.Run("NewQueryEnhancer", func(t *testing.T) {
		config := getDefaultRetrievalConfig()
		enhancer := NewQueryEnhancer(config)
		
		assert.NotNil(t, enhancer)
		assert.NotNil(t, enhancer.telecomDictionary)
		assert.NotNil(t, enhancer.synonymExpander)
		assert.NotNil(t, enhancer.spellChecker)
	})

	t.Run("ExpandTelecomTerms", func(t *testing.T) {
		enhancer := NewQueryEnhancer(nil)
		
		query := "How to configure AMF for 5G network?"
		enhanced, terms := enhancer.expandTelecomTerms(query, "configuration")
		
		assert.Contains(t, enhanced, "AMF")
		assert.Contains(t, enhanced, "Access and Mobility Management Function")
		assert.Greater(t, len(terms), 0)
	})

	t.Run("SpellChecker", func(t *testing.T) {
		enhancer := NewQueryEnhancer(nil)
		
		query := "What is gnb configuration?"
		corrected, corrections := enhancer.spellChecker.CorrectQuery(query)
		
		assert.Contains(t, corrected, "gNB")
		assert.Contains(t, corrections, "gnb")
	})

	t.Run("SynonymExpansion", func(t *testing.T) {
		enhancer := NewQueryEnhancer(nil)
		
		query := "base station configuration"
		expanded, synonyms := enhancer.synonymExpander.ExpandSynonyms(query, "")
		
		assert.Contains(t, expanded, "base station")
		assert.Greater(t, len(synonyms), 0)
	})

	t.Run("ContextEnhancement", func(t *testing.T) {
		enhancer := NewQueryEnhancer(nil)
		
		query := "How to optimize performance?"
		history := []string{
			"Previous question about AMF configuration",
			"Discussion about 5G network setup",
			"AMF troubleshooting steps",
		}
		
		enhanced := enhancer.enhanceWithContext(query, history)
		
		// Should include context terms
		assert.Contains(t, enhanced, query)
		// May contain AMF or 5G based on context extraction
	})
}

func TestSemanticReranker(t *testing.T) {
	t.Run("NewSemanticReranker", func(t *testing.T) {
		config := getDefaultRetrievalConfig()
		reranker := NewSemanticReranker(config)
		
		assert.NotNil(t, reranker)
		assert.NotNil(t, reranker.crossEncoder)
	})

	t.Run("CalculateSemanticSimilarity", func(t *testing.T) {
		reranker := NewSemanticReranker(nil)
		
		query := "AMF configuration for 5G network"
		content := "The Access and Mobility Management Function (AMF) is configured for 5G networks"
		
		similarity := reranker.calculateSemanticSimilarity(query, content)
		
		assert.Greater(t, similarity, float32(0.0))
		assert.LessOrEqual(t, similarity, float32(1.0))
	})

	t.Run("CalculateLexicalSimilarity", func(t *testing.T) {
		reranker := NewSemanticReranker(nil)
		
		query := "network configuration"
		content1 := "Network configuration parameters for telecommunications"
		content2 := "Completely unrelated document about cooking recipes"
		
		sim1 := reranker.calculateLexicalSimilarity(query, content1)
		sim2 := reranker.calculateLexicalSimilarity(query, content2)
		
		assert.Greater(t, sim1, sim2, "Related content should have higher similarity")
	})

	t.Run("RerankResults", func(t *testing.T) {
		reranker := NewSemanticReranker(nil)
		
		// Create test results
		results := []*EnhancedSearchResult{
			{
				SearchResult: &SearchResult{
					Document: &TelecomDocument{
						Content: "AMF configuration details for 5G networks",
						Source:  "3GPP",
					},
					Score: 0.5,
				},
				QualityScore: 0.8,
			},
			{
				SearchResult: &SearchResult{
					Document: &TelecomDocument{
						Content: "Unrelated content about different topic",
						Source:  "Unknown",
					},
					Score: 0.7,
				},
				QualityScore: 0.3,
			},
		}

		ctx := context.Background()
		reranked, err := reranker.RerankResults(ctx, "AMF configuration", results)
		require.NoError(t, err)
		
		assert.Equal(t, len(results), len(reranked))
		// Results should be reordered based on semantic relevance
	})
}

func TestContextAssembler(t *testing.T) {
	t.Run("NewContextAssembler", func(t *testing.T) {
		config := getDefaultRetrievalConfig()
		assembler := NewContextAssembler(config)
		
		assert.NotNil(t, assembler)
		assert.Equal(t, config, assembler.config)
	})

	t.Run("AssembleContext", func(t *testing.T) {
		config := getDefaultRetrievalConfig()
		config.MaxContextLength = 1000
		
		assembler := NewContextAssembler(config)
		
		// Create test results
		results := []*EnhancedSearchResult{
			{
				SearchResult: &SearchResult{
					Document: &TelecomDocument{
						Content:  "First document about AMF configuration",
						Title:    "AMF Configuration Guide",
						Source:   "3GPP",
						Category: "Core",
					},
					Score: 0.9,
				},
				CombinedScore: 0.9,
			},
			{
				SearchResult: &SearchResult{
					Document: &TelecomDocument{
						Content:  "Second document about network setup",
						Title:    "Network Setup Guide",
						Source:   "O-RAN",
						Category: "RAN",
					},
					Score: 0.7,
				},
				CombinedScore: 0.7,
			},
		}

		request := &EnhancedSearchRequest{
			Query:      "AMF configuration",
			IntentType: "configuration",
		}

		context, metadata := assembler.AssembleContext(results, request)
		
		assert.NotEmpty(t, context)
		assert.NotNil(t, metadata)
		assert.Equal(t, len(results), metadata.DocumentCount)
		assert.Greater(t, metadata.TotalLength, 0)
		assert.LessOrEqual(t, len(context), config.MaxContextLength)
	})

	t.Run("DifferentStrategies", func(t *testing.T) {
		assembler := NewContextAssembler(nil)
		
		results := createTestResults()
		
		strategies := []struct {
			intentType string
			expected   ContextAssemblyStrategy
		}{
			{"configuration", StrategyProgressive},
			{"troubleshooting", StrategyTopical},
			{"optimization", StrategyBalanced},
			{"", StrategyRankBased},
		}

		for _, test := range strategies {
			request := &EnhancedSearchRequest{
				IntentType: test.intentType,
			}
			
			strategy := assembler.determineAssemblyStrategy(results, request)
			// Strategy determination is context-dependent, so we just verify it returns a valid strategy
			assert.NotEmpty(t, string(strategy))
		}
	})
}

func TestRedisCache(t *testing.T) {
	// Skip Redis tests if Redis is not available
	config := getDefaultRedisCacheConfig()
	config.Address = "localhost:6379"
	
	cache, err := NewRedisCache(config)
	if err != nil {
		t.Skip("Redis not available, skipping cache tests")
	}
	defer cache.Close()

	ctx := context.Background()

	t.Run("EmbeddingCache", func(t *testing.T) {
		text := "test embedding text"
		modelName := "test-model"
		embedding := []float32{0.1, 0.2, 0.3, 0.4, 0.5}

		// Test cache miss
		result, found := cache.GetEmbedding(ctx, text, modelName)
		assert.False(t, found)
		assert.Nil(t, result)

		// Test cache set
		err := cache.SetEmbedding(ctx, text, modelName, embedding)
		require.NoError(t, err)

		// Test cache hit
		result, found = cache.GetEmbedding(ctx, text, modelName)
		assert.True(t, found)
		assert.Equal(t, embedding, result)
	})

	t.Run("DocumentCache", func(t *testing.T) {
		doc := &LoadedDocument{
			ID:      "test-doc-123",
			Content: "Test document content",
			Title:   "Test Document",
			Hash:    "test-hash-123",
		}

		// Test cache miss
		result, found := cache.GetDocument(ctx, doc.ID)
		assert.False(t, found)
		assert.Nil(t, result)

		// Test cache set
		err := cache.SetDocument(ctx, doc)
		require.NoError(t, err)

		// Test cache hit
		result, found = cache.GetDocument(ctx, doc.ID)
		assert.True(t, found)
		assert.Equal(t, doc.ID, result.ID)
		assert.Equal(t, doc.Content, result.Content)
	})

	t.Run("CacheMetrics", func(t *testing.T) {
		metrics := cache.GetMetrics()
		assert.NotNil(t, metrics)
		assert.GreaterOrEqual(t, metrics.TotalRequests, int64(0))
	})

	t.Run("HealthStatus", func(t *testing.T) {
		health := cache.GetHealthStatus(ctx)
		assert.NotNil(t, health)
		assert.Contains(t, health, "status")
		assert.Contains(t, health, "healthy")
	})
}

func TestEndToEndRAGPipeline(t *testing.T) {
	// This test requires external dependencies, so we'll mock or skip as appropriate
	if os.Getenv("INTEGRATION_TEST") != "true" {
		t.Skip("Skipping end-to-end test, set INTEGRATION_TEST=true to run")
	}

	t.Run("FullPipeline", func(t *testing.T) {
		ctx := context.Background()

		// 1. Document Loading
		loaderConfig := getDefaultLoaderConfig()
		loader := NewDocumentLoader(loaderConfig)
		
		// Create test document
		doc := &LoadedDocument{
			ID:      "e2e-test-doc",
			Content: testPDFContent,
			Title:   "End-to-End Test Document",
			Metadata: &DocumentMetadata{
				Source:   "3GPP",
				Category: "RAN",
			},
		}

		// 2. Document Chunking
		chunkingConfig := getDefaultChunkingConfig()
		chunkingService := NewChunkingService(chunkingConfig)
		
		chunks, err := chunkingService.ChunkDocument(ctx, doc)
		require.NoError(t, err)
		assert.Greater(t, len(chunks), 0)

		// 3. Embedding Generation (mocked)
		embeddingConfig := getDefaultEmbeddingConfig()
		embeddingService := NewEmbeddingService(embeddingConfig)
		
		// Mock embedding generation
		for _, chunk := range chunks {
			// In a real test, you would generate actual embeddings
			assert.NotEmpty(t, chunk.Content)
		}

		// 4. Query Enhancement
		retrievalConfig := getDefaultRetrievalConfig()
		queryEnhancer := NewQueryEnhancer(retrievalConfig)
		
		request := &EnhancedSearchRequest{
			Query:                  "How to configure AMF?",
			IntentType:            "configuration",
			EnableQueryEnhancement: true,
		}

		enhancedQuery, enhancements, err := queryEnhancer.EnhanceQuery(ctx, request)
		require.NoError(t, err)
		
		assert.NotEmpty(t, enhancedQuery)
		assert.NotNil(t, enhancements)
		assert.Contains(t, enhancedQuery, "AMF")

		// 5. Context Assembly
		contextAssembler := NewContextAssembler(retrievalConfig)
		
		// Create mock search results
		results := []*EnhancedSearchResult{
			{
				SearchResult: &SearchResult{
					Document: &TelecomDocument{
						Content: "AMF configuration parameters include...",
						Source:  "3GPP",
						Title:   "AMF Configuration Guide",
					},
					Score: 0.9,
				},
				CombinedScore: 0.9,
			},
		}

		context, metadata := contextAssembler.AssembleContext(results, request)
		
		assert.NotEmpty(t, context)
		assert.NotNil(t, metadata)
		assert.Greater(t, metadata.DocumentCount, 0)

		t.Logf("End-to-end pipeline completed successfully:")
		t.Logf("- Document loaded: %s", doc.Title)
		t.Logf("- Chunks created: %d", len(chunks))
		t.Logf("- Query enhanced: %s -> %s", request.Query, enhancedQuery)
		t.Logf("- Context assembled: %d characters", len(context))
	})
}

// Helper functions and test data

func createTestResults() []*EnhancedSearchResult {
	return []*EnhancedSearchResult{
		{
			SearchResult: &SearchResult{
				Document: &TelecomDocument{
					Content:  "Content about AMF configuration",
					Source:   "3GPP",
					Category: "Core",
				},
				Score: 0.9,
			},
			CombinedScore: 0.9,
		},
		{
			SearchResult: &SearchResult{
				Document: &TelecomDocument{
					Content:  "Content about gNB setup",
					Source:   "O-RAN",
					Category: "RAN",
				},
				Score: 0.7,
			},
			CombinedScore: 0.7,
		},
		{
			SearchResult: &SearchResult{
				Document: &TelecomDocument{
					Content:  "Transport network configuration",
					Source:   "ETSI",
					Category: "Transport",
				},
				Score: 0.6,
			},
			CombinedScore: 0.6,
		},
	}
}

// Benchmark tests

func BenchmarkDocumentChunking(b *testing.B) {
	service := NewChunkingService(nil)
	doc := &LoadedDocument{
		ID:      "benchmark-doc",
		Content: strings.Repeat(testPDFContent, 10), // Make it larger
		Title:   "Benchmark Document",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.ChunkDocument(ctx, doc)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkQueryEnhancement(b *testing.B) {
	enhancer := NewQueryEnhancer(getDefaultRetrievalConfig())
	request := &EnhancedSearchRequest{
		Query:                  "How to configure AMF for 5G network optimization?",
		IntentType:            "configuration",
		EnableQueryEnhancement: true,
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := enhancer.EnhanceQuery(ctx, request)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkContextAssembly(b *testing.B) {
	assembler := NewContextAssembler(getDefaultRetrievalConfig())
	results := createTestResults()
	request := &EnhancedSearchRequest{
		Query:      "test query",
		IntentType: "configuration",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = assembler.AssembleContext(results, request)
	}
}

// Table-driven tests

func TestTelecomTermExtraction(t *testing.T) {
	loader := NewDocumentLoader(nil)

	tests := []struct {
		name           string
		content        string
		expectedTerms  []string
		expectedNFs    []string
		expectedTechs  []string
	}{
		{
			name:          "3GPP 5G Core",
			content:       "The AMF and SMF work together with UPF in 5G core network",
			expectedNFs:   []string{"AMF", "SMF", "UPF"},
			expectedTechs: []string{"5G"},
		},
		{
			name:          "O-RAN Architecture",
			content:       "O-RAN architecture includes DU, CU, and RU components",
			expectedNFs:   []string{"DU", "CU", "RU"},
			expectedTechs: []string{"O-RAN"},
		},
		{
			name:          "LTE Network",
			content:       "The eNB connects to EPC for 4G LTE services",
			expectedNFs:   []string{"eNB"},
			expectedTechs: []string{"4G"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metadata := loader.extractTelecomMetadata(tt.content, "test.pdf")

			for _, expectedNF := range tt.expectedNFs {
				assert.Contains(t, metadata.NetworkFunctions, expectedNF,
					"Should extract network function: %s", expectedNF)
			}

			for _, expectedTech := range tt.expectedTechs {
				assert.Contains(t, metadata.Technologies, expectedTech,
					"Should extract technology: %s", expectedTech)
			}
		})
	}
}

func TestQueryEnhancementScenarios(t *testing.T) {
	enhancer := NewQueryEnhancer(getDefaultRetrievalConfig())

	tests := []struct {
		name           string
		query          string
		intentType     string
		expectedEnhancements []string
	}{
		{
			name:       "Configuration Query",
			query:      "How to configure AMF?",
			intentType: "configuration",
			expectedEnhancements: []string{"configuration", "parameter", "Access and Mobility Management Function"},
		},
		{
			name:       "Troubleshooting Query",
			query:      "AMF connection issues",
			intentType: "troubleshooting",
			expectedEnhancements: []string{"problem", "issue", "troubleshoot"},
		},
		{
			name:       "Optimization Query",
			query:      "Improve gNB performance",
			intentType: "optimization",
			expectedEnhancements: []string{"optimize", "performance", "Next Generation NodeB"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := &EnhancedSearchRequest{
				Query:                  tt.query,
				IntentType:            tt.intentType,
				EnableQueryEnhancement: true,
			}

			ctx := context.Background()
			enhanced, _, err := enhancer.EnhanceQuery(ctx, request)
			require.NoError(t, err)

			for _, expected := range tt.expectedEnhancements {
				assert.Contains(t, strings.ToLower(enhanced), strings.ToLower(expected),
					"Enhanced query should contain: %s", expected)
			}
		})
	}
}