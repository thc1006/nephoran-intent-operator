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
	"sync"
	"time"

	"github.com/stretchr/testify/mock"

	"github.com/thc1006/nephoran-intent-operator/pkg/rag"
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
				"content":    "AMF (Access and Mobility Management Function) is a key component of 5G Core Network",
				"metadata":   map[string]interface{}{"source": "3GPP TS 23.501", "section": "6.2.2"},
				"similarity": 0.9,
			},
			{
				"content":    "SMF (Session Management Function) handles PDU sessions in 5G networks",
				"metadata":   map[string]interface{}{"source": "3GPP TS 23.502", "section": "4.3.2"},
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
		Documents:   documents,
		Query:       request.Query,
		TotalFound:  len(documents),
		ProcessTime: 50 * time.Millisecond,
		Context:     make(map[string]interface{}),
		Metadata:    make(map[string]interface{}),
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
		return nil, mock.AnythingOfType("error")
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

	return &rag.QueryResponse{
		Documents:     filteredDocs,
		MaxSimilarity: m.maxSimilarity,
		AvgSimilarity: avgSimilarity,
		Metadata: map[string]interface{}{
			"queryTime": m.queryDelay,
			"totalDocs": len(m.documents),
		},
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