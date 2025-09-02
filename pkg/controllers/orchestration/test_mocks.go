/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package orchestration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

// SharedMockRAGService provides a centralized mock RAG service for all tests
type SharedMockRAGService struct {
	mock.Mock
	responses         map[string]*rag.RetrievalResponse
	documents         []map[string]interface{}
	shouldReturnError bool
	queryDelay        time.Duration
	maxSimilarity     float64
	mutex             sync.RWMutex
}

func NewSharedMockRAGService() *SharedMockRAGService {
	return &SharedMockRAGService{
		responses: make(map[string]*rag.RetrievalResponse),
		documents: []map[string]interface{}{
			{
				"content":    "AMF (Access and Mobility Management Function) manages access and mobility for UE",
				"metadata":   map[string]interface{}{"source": "3GPP TS 23.501", "section": "6.2.2"},
				"similarity": 0.9,
			},
			{
				"content":    "SMF (Session Management Function) handles PDU sessions in 5G networks",
				"metadata":   map[string]interface{}{},
				"similarity": 0.85,
			},
		},
		maxSimilarity: 0.95,
	}
}

func (m *SharedMockRAGService) RetrieveContext(ctx context.Context, request *rag.RetrievalRequest) (*rag.RetrievalResponse, error) {
	args := m.Called(ctx, request)

	// Simulate delay if configured
	if m.queryDelay > 0 {
		time.Sleep(m.queryDelay)
	}

	// Return error if configured
	if m.shouldReturnError {
		return nil, args.Error(1)
	}

	// Return predefined response if available
	m.mutex.RLock()
	if response, exists := m.responses[request.Query]; exists {
		m.mutex.RUnlock()
		return response, nil
	}
	m.mutex.RUnlock()

	// Create default response
	documents := make([]rag.Doc, 0, len(m.documents))
	for _, doc := range m.documents {
		documents = append(documents, rag.Doc{
			ID:         "doc-1",
			Content:    doc["content"].(string),
			Confidence: doc["similarity"].(float64),
			Metadata:   doc["metadata"].(map[string]interface{}),
		})
	}

	response := &rag.RetrievalResponse{
		Documents:             make([]map[string]interface{}, len(documents)),
		Duration:              50 * time.Millisecond,
		AverageRelevanceScore: 0.8,
		TopRelevanceScore:     0.9,
		QueryWasEnhanced:      false,
		Metadata:              make(map[string]interface{}),
	}

	// Convert rag.Doc to map[string]interface{}
	for i, doc := range documents {
		response.Documents[i] = json.RawMessage(`{}`)
	}

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	return response, nil
}

func (m *SharedMockRAGService) Query(ctx context.Context, req *rag.QueryRequest) (*rag.QueryResponse, error) {
	// Simulate delay if configured
	if m.queryDelay > 0 {
		time.Sleep(m.queryDelay)
	}

	// Return error if configured
	if m.shouldReturnError {
		return nil, fmt.Errorf("mock RAG service error")
	}

	// Filter documents based on query
	var filteredDocs []map[string]interface{}
	avgSimilarity := 0.0

	for _, doc := range m.documents {
		if similarity, ok := doc["similarity"].(float64); ok && similarity >= 0.7 {
			filteredDocs = append(filteredDocs, doc)
			avgSimilarity += similarity
		}
	}

	if len(filteredDocs) > 0 {
		avgSimilarity /= float64(len(filteredDocs))
	}

	// Convert filteredDocs to SearchResult format for QueryResponse
	results := make([]*shared.SearchResult, len(filteredDocs))
	for i, doc := range filteredDocs {
		results[i] = &shared.SearchResult{
			ID:       fmt.Sprintf("doc-%d", i),
			Content:  doc["content"].(string),
			Score:    float32(doc["similarity"].(float64)),
			Metadata: doc["metadata"].(map[string]interface{}),
		}
	}

	return &rag.QueryResponse{
		Query:          req.Query,
		Results:        results,
		ProcessingTime: m.queryDelay,
		EmbeddingCost:  0.001,
		ProviderUsed:   "mock-provider",
	}, nil
}

func (m *SharedMockRAGService) SetResponse(query string, response *rag.RetrievalResponse) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.responses == nil {
		m.responses = make(map[string]*rag.RetrievalResponse)
	}
	m.responses[query] = response
}

func (m *SharedMockRAGService) SetQueryDelay(delay time.Duration) {
	m.queryDelay = delay
}

func (m *SharedMockRAGService) SetShouldReturnError(shouldError bool) {
	m.shouldReturnError = shouldError
}

func (m *SharedMockRAGService) SetMaxSimilarity(similarity float64) {
	m.maxSimilarity = similarity
}

func (m *SharedMockRAGService) SetDocuments(docs []map[string]interface{}) {
	m.documents = docs
}

// Backward compatibility aliases for existing tests
type MockRAGService = SharedMockRAGService

// NewMockRAGService creates a new mock RAG service (backward compatible)
func NewMockRAGService() *MockRAGService {
	return NewSharedMockRAGService()
}

// MockTelecomResourceCalculator provides mock telecom resource calculation functionality
type MockTelecomResourceCalculator struct {
	mock.Mock
	calculationDelay  time.Duration
	shouldReturnError bool
	mutex             sync.RWMutex
}

// NewMockTelecomResourceCalculator creates a new mock telecom resource calculator
func NewMockTelecomResourceCalculator() *MockTelecomResourceCalculator {
	return &MockTelecomResourceCalculator{}
}

// CalculateResources performs mock resource calculation
func (m *MockTelecomResourceCalculator) CalculateResources(ctx context.Context, request interface{}) (interface{}, error) {
	args := m.Called(ctx, request)

	// Simulate delay if configured
	m.mutex.RLock()
	delay := m.calculationDelay
	shouldError := m.shouldReturnError
	m.mutex.RUnlock()

	if delay > 0 {
		time.Sleep(delay)
	}

	// Return error if configured
	if shouldError {
		return nil, args.Error(1)
	}

	// Return mock calculation result
	result := json.RawMessage(`{}`)

	if args.Error(1) != nil {
		return nil, args.Error(1)
	}

	return result, nil
}

// SetCalculationDelay sets the delay for mock calculations
func (m *MockTelecomResourceCalculator) SetCalculationDelay(delay time.Duration) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.calculationDelay = delay
}

// SetShouldReturnError configures whether the mock should return errors
func (m *MockTelecomResourceCalculator) SetShouldReturnError(shouldError bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.shouldReturnError = shouldError
}

// Additional mock types needed for compilation (non-duplicates only)

// MockResourceOptimizationEngine provides mock resource optimization functionality
type MockResourceOptimizationEngine struct {
	mock.Mock
}

// NewMockResourceOptimizationEngine creates a new mock resource optimization engine
func NewMockResourceOptimizationEngine() *MockResourceOptimizationEngine {
	return &MockResourceOptimizationEngine{}
}

// MockResourceConstraintSolver provides mock resource constraint solving functionality
type MockResourceConstraintSolver struct {
	mock.Mock
}

// NewMockResourceConstraintSolver creates a new mock resource constraint solver
func NewMockResourceConstraintSolver() *MockResourceConstraintSolver {
	return &MockResourceConstraintSolver{}
}

// MockTelecomCostEstimator provides mock telecom cost estimation functionality
type MockTelecomCostEstimator struct {
	mock.Mock
}

// NewMockTelecomCostEstimator creates a new mock telecom cost estimator
func NewMockTelecomCostEstimator() *MockTelecomCostEstimator {
	return &MockTelecomCostEstimator{}
}

