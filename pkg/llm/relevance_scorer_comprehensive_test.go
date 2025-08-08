package llm

import (
	"context"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// MockEmbeddingServiceForScorer provides a mock implementation specifically for RelevanceScorer testing
type MockEmbeddingServiceForScorer struct {
	similarity      float64
	embedding       []float64
	err             error
	callCount       int
	lastTexts       []string
	simulateLatency time.Duration
}

func (m *MockEmbeddingServiceForScorer) CalculateSimilarity(ctx context.Context, text1, text2 string) (float64, error) {
	m.callCount++
	m.lastTexts = []string{text1, text2}
	
	// Simulate latency if specified
	if m.simulateLatency > 0 {
		time.Sleep(m.simulateLatency)
	}
	
	if m.err != nil {
		return 0, m.err
	}
	
	return m.similarity, nil
}

func (m *MockEmbeddingServiceForScorer) GetEmbedding(ctx context.Context, text string) ([]float64, error) {
	m.callCount++
	
	if m.err != nil {
		return nil, m.err
	}
	
	if m.embedding != nil {
		return m.embedding, nil
	}
	
	// Generate a mock embedding based on text characteristics
	embedding := make([]float64, 384) // Typical embedding size
	for i := range embedding {
		embedding[i] = float64(len(text)%10) / 10.0 // Simple mock based on text length
	}
	
	return embedding, nil
}

func TestRelevanceScorer_CalculateRelevance(t *testing.T) {
	tests := []struct {
		name             string
		request          *RelevanceRequest
		mockSimilarity   float64
		mockEmbedding    []float64
		mockError        error
		config           *RelevanceScorerConfig
		expectedMinScore float32
		expectedMaxScore float32
		expectedError    bool
		errorContains    string
		validateScore    func(t *testing.T, score *RelevanceScore)
	}{
		{
			name: "high relevance 5G AMF deployment document",
			request: &RelevanceRequest{
				Query:      "Deploy AMF network function in 5G core",
				IntentType: "configuration",
				Document: &shared.TelecomDocument{
					ID:       "amf_deploy_guide",
					Title:    "5G AMF Deployment Guide",
					Content:  "This comprehensive guide covers Access and Mobility Management Function deployment in 5G standalone core networks, including configuration parameters, scaling policies, and integration with SMF and UPF components.",
					Source:   "3GPP TS 23.501 v17.0.0",
					Category: "configuration",
					Version:  "v17.0.0",
					Keywords: []string{"AMF", "5G", "deployment", "configuration"},
					Language: "en",
					DocumentType: "specification",
					NetworkFunction: []string{"AMF"},
					Technology: []string{"5G", "5GC"},
					UseCase: []string{"configuration", "deployment"},
					Confidence: 0.95,
					CreatedAt: time.Now().Add(-30 * 24 * time.Hour),
					UpdatedAt: time.Now().Add(-5 * 24 * time.Hour),
				},
				Position:      0,
				OriginalScore: 0.9,
			},
			mockSimilarity:   0.88,
			expectedMinScore: 0.7,
			expectedMaxScore: 1.0,
			validateScore: func(t *testing.T, score *RelevanceScore) {
				if score.SemanticScore < 0.8 {
					t.Errorf("Expected high semantic score for exact match, got %f", score.SemanticScore)
				}
				if score.AuthorityScore < 0.9 {
					t.Errorf("Expected high authority score for 3GPP source, got %f", score.AuthorityScore)
				}
				if score.IntentScore < 0.6 {
					t.Errorf("Expected good intent alignment for configuration match, got %f", score.IntentScore)
				}
				if len(score.Explanation) == 0 {
					t.Error("Expected non-empty explanation")
				}
			},
		},
		{
			name: "medium relevance O-RAN document with different intent",
			request: &RelevanceRequest{
				Query:      "Troubleshoot handover failures",
				IntentType: "troubleshooting",
				Document: &shared.TelecomDocument{
					ID:       "oran_architecture",
					Title:    "O-RAN Architecture Overview",
					Content:  "O-RAN Alliance specification for disaggregated Radio Access Network architecture including RIC, xApps, and interface specifications.",
					Source:   "O-RAN.WG1.O-RAN-Architecture-Description-v07.00",
					Category: "architecture",
					Version:  "v7.0",
					Keywords: []string{"O-RAN", "architecture", "RIC", "xApp"},
					NetworkFunction: []string{"RIC", "O-DU", "O-CU"},
					Technology: []string{"O-RAN", "5G", "RAN"},
					UseCase: []string{"architecture", "design"},
					Confidence: 0.82,
					CreatedAt: time.Now().Add(-90 * 24 * time.Hour),
					UpdatedAt: time.Now().Add(-60 * 24 * time.Hour),
				},
				Position:      2,
				OriginalScore: 0.65,
			},
			mockSimilarity:   0.35,
			expectedMinScore: 0.3,
			expectedMaxScore: 0.7,
			validateScore: func(t *testing.T, score *RelevanceScore) {
				if score.SemanticScore > 0.5 {
					t.Errorf("Expected lower semantic score for intent mismatch, got %f", score.SemanticScore)
				}
				if score.AuthorityScore < 0.8 {
					t.Errorf("Expected high authority score for O-RAN Alliance, got %f", score.AuthorityScore)
				}
				if score.IntentScore > 0.4 {
					t.Errorf("Expected low intent alignment for architecture vs troubleshooting, got %f", score.IntentScore)
				}
			},
		},
		{
			name: "low relevance outdated document",
			request: &RelevanceRequest{
				Query:      "Deploy 5G network slicing",
				IntentType: "configuration",
				Document: &shared.TelecomDocument{
					ID:       "old_2g_guide",
					Title:    "2G Network Planning Guide",
					Content:  "Legacy 2G network planning procedures and frequency allocation guidelines from early mobile networks.",
					Source:   "Internal Documentation v1.2",
					Category: "planning",
					Version:  "v1.2",
					Keywords: []string{"2G", "planning", "frequency"},
					Technology: []string{"2G", "GSM"},
					UseCase: []string{"planning"},
					Confidence: 0.45,
					CreatedAt: time.Now().Add(-5 * 365 * 24 * time.Hour), // 5 years old
					UpdatedAt: time.Now().Add(-4 * 365 * 24 * time.Hour), // 4 years old
				},
				Position:      8,
				OriginalScore: 0.25,
			},
			mockSimilarity:   0.15,
			expectedMinScore: 0.1,
			expectedMaxScore: 0.4,
			validateScore: func(t *testing.T, score *RelevanceScore) {
				if score.RecencyScore > 0.3 {
					t.Errorf("Expected low recency score for old document, got %f", score.RecencyScore)
				}
				if score.SemanticScore > 0.3 {
					t.Errorf("Expected low semantic score for 2G vs 5G mismatch, got %f", score.SemanticScore)
				}
				if score.DomainScore > 0.3 {
					t.Errorf("Expected low domain score for technology mismatch, got %f", score.DomainScore)
				}
			},
		},
		{
			name: "embedding service failure fallback",
			request: &RelevanceRequest{
				Query:      "Configure AMF parameters",
				IntentType: "configuration",
				Document: &shared.TelecomDocument{
					ID:      "amf_config",
					Title:   "AMF Configuration Parameters",
					Content: "AMF configuration includes mobility management, session management integration, and security policies.",
					Source:  "Technical Guide v2.1",
					Keywords: []string{"AMF", "configuration", "parameters"},
					NetworkFunction: []string{"AMF"},
					Technology: []string{"5G"},
					Confidence: 0.75,
					CreatedAt: time.Now().Add(-30 * 24 * time.Hour),
					UpdatedAt: time.Now().Add(-10 * 24 * time.Hour),
				},
				Position:      1,
				OriginalScore: 0.7,
			},
			mockError:        fmt.Errorf("embedding service unavailable"),
			expectedMinScore: 0.4,
			expectedMaxScore: 0.8,
			validateScore: func(t *testing.T, score *RelevanceScore) {
				// Should still have reasonable scores from other factors
				if score.AuthorityScore == 0 {
					t.Error("Expected non-zero authority score even with embedding failure")
				}
				if score.RecencyScore == 0 {
					t.Error("Expected non-zero recency score even with embedding failure")
				}
			},
		},
		{
			name: "empty query error",
			request: &RelevanceRequest{
				Query:      "",
				IntentType: "configuration",
				Document: &shared.TelecomDocument{
					ID:      "test_doc",
					Title:   "Test Document",
					Content: "Test content",
					Source:  "Test Source",
				},
				Position: 0,
			},
			expectedError: true,
			errorContains: "query cannot be empty",
		},
		{
			name: "nil document error",
			request: &RelevanceRequest{
				Query:      "Test query",
				IntentType: "configuration",
				Document:   nil,
				Position:   0,
			},
			expectedError: true,
			errorContains: "document cannot be nil",
		},
		{
			name: "nil request error",
			request:       nil,
			expectedError: true,
			errorContains: "relevance request cannot be nil",
		},
		{
			name: "technical terms boost scenario",
			request: &RelevanceRequest{
				Query:      "5G NR gNB PDCP RLC MAC PHY layer configuration",
				IntentType: "configuration",
				Document: &shared.TelecomDocument{
					ID:       "nr_protocol_stack",
					Title:    "5G NR Protocol Stack Configuration",
					Content:  "New Radio gNodeB protocol stack including PDCP, RLC, MAC, and PHY layer configuration procedures for 5G networks.",
					Source:   "3GPP TS 38.300 v16.7.0",
					Category: "configuration",
					Keywords: []string{"5G", "NR", "gNodeB", "PDCP", "RLC", "MAC", "PHY"},
					NetworkFunction: []string{"gNB"},
					Technology: []string{"5G", "NR"},
					Confidence: 0.92,
					CreatedAt: time.Now().Add(-60 * 24 * time.Hour),
					UpdatedAt: time.Now().Add(-20 * 24 * time.Hour),
				},
				Position: 0,
			},
			mockSimilarity:   0.75,
			expectedMinScore: 0.65,
			expectedMaxScore: 1.0,
			validateScore: func(t *testing.T, score *RelevanceScore) {
				// Should have high scores due to many technical term matches
				if score.SemanticScore < 0.7 {
					t.Errorf("Expected high semantic score for technical term matches, got %f", score.SemanticScore)
				}
				if score.DomainScore < 0.7 {
					t.Errorf("Expected high domain score for protocol stack match, got %f", score.DomainScore)
				}
			},
		},
		{
			name: "standards document authority boost",
			request: &RelevanceRequest{
				Query:      "Network slice selection procedures",
				IntentType: "configuration",
				Document: &shared.TelecomDocument{
					ID:       "nssf_spec",
					Title:    "Network Slice Selection Function (NSSF)",
					Content:  "3GPP specification for NSSF procedures, network slice selection, and policy enforcement in 5G networks.",
					Source:   "3GPP TS 23.501 Release 17",
					Category: "specification",
					Version:  "Rel-17",
					Keywords: []string{"NSSF", "network slicing", "5G"},
					NetworkFunction: []string{"NSSF"},
					Technology: []string{"5G", "Network Slicing"},
					Confidence: 0.98,
					CreatedAt: time.Now().Add(-15 * 24 * time.Hour),
					UpdatedAt: time.Now().Add(-1 * 24 * time.Hour),
				},
				Position: 0,
			},
			mockSimilarity:   0.82,
			expectedMinScore: 0.75,
			expectedMaxScore: 1.0,
			validateScore: func(t *testing.T, score *RelevanceScore) {
				if score.AuthorityScore < 0.95 {
					t.Errorf("Expected very high authority score for 3GPP standard, got %f", score.AuthorityScore)
				}
				if score.RecencyScore < 0.8 {
					t.Errorf("Expected high recency score for recent document, got %f", score.RecencyScore)
				}
			},
		},
		{
			name: "position penalty application",
			request: &RelevanceRequest{
				Query:      "Deploy UPF",
				IntentType: "configuration",
				Document: &shared.TelecomDocument{
					ID:      "upf_guide",
					Title:   "UPF Deployment Guide",
					Content: "User Plane Function deployment procedures for 5G core networks.",
					Source:  "Deployment Guide v1.0",
					Keywords: []string{"UPF", "deployment", "5G"},
					NetworkFunction: []string{"UPF"},
					Technology: []string{"5G"},
					Confidence: 0.88,
					CreatedAt: time.Now().Add(-20 * 24 * time.Hour),
					UpdatedAt: time.Now().Add(-5 * 24 * time.Hour),
				},
				Position:      15, // High position should apply penalty
				OriginalScore: 0.8,
			},
			mockSimilarity:   0.85,
			expectedMinScore: 0.6,
			expectedMaxScore: 0.9,
			validateScore: func(t *testing.T, score *RelevanceScore) {
				// Check that position penalty was applied
				if positionPenalty, exists := score.Factors["position_penalty"]; exists {
					if penalty, ok := positionPenalty.(float64); ok {
						if penalty >= 1.0 {
							t.Errorf("Expected position penalty < 1.0 for high position, got %f", penalty)
						}
					}
				}
			},
		},
		{
			name: "multilingual document handling",
			request: &RelevanceRequest{
				Query:      "5G network optimization",
				IntentType: "optimization",
				Document: &shared.TelecomDocument{
					ID:      "optimization_guide_de",
					Title:   "5G Netzwerkoptimierung Leitfaden",
					Content: "Dieses Dokument beschreibt Optimierungsverfahren für 5G-Netzwerke einschließlich Leistungsüberwachung und Parametereinstellung.",
					Source:  "Technical Guide v2.0",
					Language: "de",
					Keywords: []string{"5G", "Optimierung", "Netzwerk"},
					Technology: []string{"5G"},
					Confidence: 0.70,
					CreatedAt: time.Now().Add(-45 * 24 * time.Hour),
					UpdatedAt: time.Now().Add(-15 * 24 * time.Hour),
				},
				Position: 3,
			},
			mockSimilarity:   0.45, // Lower due to language mismatch
			expectedMinScore: 0.3,
			expectedMaxScore: 0.6,
			validateScore: func(t *testing.T, score *RelevanceScore) {
				// Should still have reasonable domain score due to "5G" keyword match
				if score.DomainScore == 0 {
					t.Error("Expected non-zero domain score for technology match despite language barrier")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock embedding service
			mockEmbedding := &MockEmbeddingServiceForScorer{
				similarity: tt.mockSimilarity,
				embedding:  tt.mockEmbedding,
				err:        tt.mockError,
			}

			// Use custom config if provided, otherwise default
			config := tt.config
			if config == nil {
				config = getDefaultRelevanceScorerConfig()
			}

			// Create relevance scorer
			scorer := NewRelevanceScorer(config, mockEmbedding)

			// Execute the test
			ctx := context.Background()
			result, err := scorer.CalculateRelevance(ctx, tt.request)

			// Check error expectations
			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			// Check for unexpected errors
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Validate result exists
			if result == nil {
				t.Error("Expected non-nil result")
				return
			}

			// Check overall score bounds
			if result.OverallScore < 0.0 || result.OverallScore > 1.0 {
				t.Errorf("Overall score out of bounds: %f (expected 0.0-1.0)", result.OverallScore)
			}

			// Check expected score range
			if tt.expectedMinScore > 0 && result.OverallScore < tt.expectedMinScore {
				t.Errorf("Overall score %f below expected minimum %f", result.OverallScore, tt.expectedMinScore)
			}
			if tt.expectedMaxScore > 0 && result.OverallScore > tt.expectedMaxScore {
				t.Errorf("Overall score %f above expected maximum %f", result.OverallScore, tt.expectedMaxScore)
			}

			// Check individual scores are in bounds
			scores := []struct {
				name  string
				value float32
			}{
				{"semantic", result.SemanticScore},
				{"authority", result.AuthorityScore},
				{"recency", result.RecencyScore},
				{"domain", result.DomainScore},
				{"intent", result.IntentScore},
			}

			for _, score := range scores {
				if score.value < 0.0 || score.value > 1.0 {
					t.Errorf("%s score out of bounds: %f (expected 0.0-1.0)", score.name, score.value)
				}
			}

			// Check that factors are populated
			if result.Factors == nil {
				t.Error("Expected factors to be populated")
			} else {
				requiredFactors := []string{"semantic_similarity", "source_authority", "document_recency", "domain_specificity", "intent_alignment"}
				for _, factor := range requiredFactors {
					if _, exists := result.Factors[factor]; !exists {
						t.Errorf("Missing required factor: %s", factor)
					}
				}
			}

			// Check processing time is reasonable
			if result.ProcessingTime > 10*time.Second {
				t.Errorf("Processing time too long: %v", result.ProcessingTime)
			}

			// Run custom validation if provided
			if tt.validateScore != nil {
				tt.validateScore(t, result)
			}

			t.Logf("Test '%s' - Overall: %.3f, Semantic: %.3f, Authority: %.3f, Recency: %.3f, Domain: %.3f, Intent: %.3f",
				tt.name, result.OverallScore, result.SemanticScore, result.AuthorityScore,
				result.RecencyScore, result.DomainScore, result.IntentScore)
		})
	}
}

func TestRelevanceScorer_ConfigurationValidation(t *testing.T) {
	tests := []struct {
		name          string
		config        *RelevanceScorerConfig
		expectedError bool
		errorContains string
	}{
		{
			name: "valid configuration",
			config: &RelevanceScorerConfig{
				SemanticWeight:        0.4,
				AuthorityWeight:       0.3,
				RecencyWeight:        0.15,
				DomainWeight:         0.1,
				IntentAlignmentWeight: 0.05,
			},
			expectedError: false,
		},
		{
			name: "weights sum to more than 1.0",
			config: &RelevanceScorerConfig{
				SemanticWeight:        0.5,
				AuthorityWeight:       0.4,
				RecencyWeight:        0.3,
				DomainWeight:         0.2,
				IntentAlignmentWeight: 0.1,
			},
			expectedError: true,
			errorContains: "scoring weights must sum to 1.0",
		},
		{
			name: "weights sum to less than 1.0",
			config: &RelevanceScorerConfig{
				SemanticWeight:        0.2,
				AuthorityWeight:       0.2,
				RecencyWeight:        0.1,
				DomainWeight:         0.1,
				IntentAlignmentWeight: 0.1,
			},
			expectedError: true,
			errorContains: "scoring weights must sum to 1.0",
		},
		{
			name:          "nil configuration",
			config:        nil,
			expectedError: true,
			errorContains: "configuration cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockEmbedding := &MockEmbeddingServiceForScorer{}
			scorer := NewRelevanceScorer(getDefaultRelevanceScorerConfig(), mockEmbedding)

			err := scorer.UpdateConfig(tt.config)

			if tt.expectedError {
				if err == nil {
					t.Errorf("Expected error but got none")
					return
				}
				if tt.errorContains != "" && !contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tt.errorContains, err.Error())
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
		})
	}
}

func TestRelevanceScorer_Performance(t *testing.T) {
	mockEmbedding := &MockEmbeddingServiceForScorer{
		similarity: 0.75,
	}

	scorer := NewRelevanceScorer(getDefaultRelevanceScorerConfig(), mockEmbedding)

	// Create test document
	doc := &shared.TelecomDocument{
		ID:      "perf_test_doc",
		Title:   "Performance Test Document",
		Content: "This is a test document for performance evaluation with various 5G and O-RAN related content including AMF, SMF, UPF, gNodeB, and other network function descriptions.",
		Source:  "Performance Test v1.0",
		Keywords: []string{"5G", "O-RAN", "AMF", "SMF", "UPF"},
		NetworkFunction: []string{"AMF", "SMF", "UPF"},
		Technology: []string{"5G", "O-RAN"},
		Confidence: 0.8,
		CreatedAt: time.Now().Add(-30 * 24 * time.Hour),
		UpdatedAt: time.Now().Add(-5 * 24 * time.Hour),
	}

	request := &RelevanceRequest{
		Query:      "Deploy 5G network functions",
		IntentType: "configuration",
		Document:   doc,
		Position:   0,
	}

	// Warm up
	_, _ = scorer.CalculateRelevance(context.Background(), request)

	// Performance test
	iterations := 100
	start := time.Now()

	for i := 0; i < iterations; i++ {
		_, err := scorer.CalculateRelevance(context.Background(), request)
		if err != nil {
			t.Errorf("Iteration %d failed: %v", i, err)
			return
		}
	}

	duration := time.Since(start)
	avgDuration := duration / time.Duration(iterations)

	// Performance assertions
	if avgDuration > 50*time.Millisecond {
		t.Errorf("Average processing time too high: %v (expected < 50ms)", avgDuration)
	}

	t.Logf("Performance test: %d iterations in %v (avg: %v)", iterations, duration, avgDuration)

	// Check metrics
	metrics := scorer.GetMetrics()
	if totalScores, ok := metrics.TotalScores; ok && totalScores < int64(iterations) {
		t.Errorf("Expected at least %d total scores, got %d", iterations, totalScores)
	}
}

func TestRelevanceScorer_ConcurrentAccess(t *testing.T) {
	mockEmbedding := &MockEmbeddingServiceForScorer{
		similarity: 0.7,
	}

	scorer := NewRelevanceScorer(getDefaultRelevanceScorerConfig(), mockEmbedding)

	// Create test document
	doc := &shared.TelecomDocument{
		ID:      "concurrent_test_doc",
		Title:   "Concurrent Access Test Document",
		Content: "Test document for concurrent access with 5G AMF deployment procedures",
		Source:  "Concurrent Test v1.0",
		Keywords: []string{"5G", "AMF", "deployment"},
		NetworkFunction: []string{"AMF"},
		Technology: []string{"5G"},
		Confidence: 0.85,
		CreatedAt: time.Now().Add(-20 * 24 * time.Hour),
		UpdatedAt: time.Now().Add(-3 * 24 * time.Hour),
	}

	// Run concurrent scoring operations
	concurrentRequests := 10
	results := make(chan error, concurrentRequests)

	for i := 0; i < concurrentRequests; i++ {
		go func(id int) {
			request := &RelevanceRequest{
				Query:      fmt.Sprintf("Deploy AMF network function %d", id),
				IntentType: "configuration",
				Document:   doc,
				Position:   id,
			}

			_, err := scorer.CalculateRelevance(context.Background(), request)
			results <- err
		}(i)
	}

	// Collect results
	for i := 0; i < concurrentRequests; i++ {
		err := <-results
		if err != nil {
			t.Errorf("Concurrent request %d failed: %v", i, err)
		}
	}

	// Verify metrics reflect concurrent operations
	metrics := scorer.GetMetrics()
	if totalScores := metrics.TotalScores; totalScores < int64(concurrentRequests) {
		t.Errorf("Expected at least %d total scores from concurrent requests, got %d", concurrentRequests, totalScores)
	}
}

func TestRelevanceScorer_EdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		setupScorer func() *RelevanceScorer
		request     *RelevanceRequest
		expectError bool
	}{
		{
			name: "very long document content",
			setupScorer: func() *RelevanceScorer {
				return NewRelevanceScorer(getDefaultRelevanceScorerConfig(), &MockEmbeddingServiceForScorer{similarity: 0.6})
			},
			request: &RelevanceRequest{
				Query:      "Deploy network function",
				IntentType: "configuration",
				Document: &shared.TelecomDocument{
					ID:      "long_doc",
					Title:   "Very Long Document",
					Content: generateLongContent(10000), // 10k characters
					Source:  "Long Content Test",
					Keywords: []string{"test", "long", "content"},
					Confidence: 0.5,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Position: 0,
			},
			expectError: false,
		},
		{
			name: "document with special characters",
			setupScorer: func() *RelevanceScorer {
				return NewRelevanceScorer(getDefaultRelevanceScorerConfig(), &MockEmbeddingServiceForScorer{similarity: 0.5})
			},
			request: &RelevanceRequest{
				Query:      "Deploy network function with special characters",
				IntentType: "configuration",
				Document: &shared.TelecomDocument{
					ID:      "special_chars_doc",
					Title:   "Document with Special Characters: ñáéíóú çüß 中文 한글 العربية",
					Content: "Content with special characters: €$¥£ @#%^&*()_+-=[]{}|;:,.<>? and emojis 🚀📡🌐",
					Source:  "Special Characters Test™",
					Keywords: []string{"special", "characters", "unicode"},
					Confidence: 0.7,
					CreatedAt: time.Now(),
					UpdatedAt: time.Now(),
				},
				Position: 0,
			},
			expectError: false,
		},
		{
			name: "document with zero confidence",
			setupScorer: func() *RelevanceScorer {
				return NewRelevanceScorer(getDefaultRelevanceScorerConfig(), &MockEmbeddingServiceForScorer{similarity: 0.8})
			},
			request: &RelevanceRequest{
				Query:      "Test query",
				IntentType: "test",
				Document: &shared.TelecomDocument{
					ID:         "zero_conf_doc",
					Title:      "Zero Confidence Document",
					Content:    "Document with zero confidence score",
					Source:     "Test Source",
					Confidence: 0.0, // Zero confidence
					CreatedAt:  time.Now(),
					UpdatedAt:  time.Now(),
				},
				Position: 0,
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scorer := tt.setupScorer()

			result, err := scorer.CalculateRelevance(context.Background(), tt.request)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if result == nil {
				t.Error("Expected non-nil result")
				return
			}

			// Basic sanity checks for edge cases
			if result.OverallScore < 0.0 || result.OverallScore > 1.0 {
				t.Errorf("Overall score out of bounds: %f", result.OverallScore)
			}

			t.Logf("Edge case '%s' - Overall score: %.3f", tt.name, result.OverallScore)
		})
	}
}

// Helper functions

func generateLongContent(length int) string {
	content := "This is a very long document content for testing purposes. "
	content += "It contains various telecommunications terms like 5G, AMF, SMF, UPF, gNodeB, O-RAN, RIC, xApp, and others. "

	// Repeat content to reach desired length
	for len(content) < length {
		content += content
	}

	return content[:length]
}

// Additional helper to check if string contains substring (already defined in context_builder_test.go but duplicated for clarity)
func containsString(haystack, needle string) bool {
	return len(needle) == 0 || strings.Contains(haystack, needle)
}