package integration

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/rand"
	"strings"
	"sync"
	"time"

	"github.com/nephio-project/nephoran-intent-operator/pkg/llm"
)

// RetrievedDocument represents a document retrieved from a vector store (stub).

type RetrievedDocument struct {
	ID string

	Content string

	Score float32

	Source string

	Metadata map[string]interface{}
}

// MockLLMService simulates LLM service behavior.

type MockLLMService struct {
	mu sync.RWMutex

	responses map[string]string

	errors map[string]error

	latencySimulation time.Duration

	callCount int

	failureRate float64
}

// NewMockLLMService creates a new mock LLM service.

func NewMockLLMService() *MockLLMService {

	return &MockLLMService{

		responses: map[string]string{

			"network_slice_creation": "To create a network slice, you need to configure the slice parameters including bandwidth, latency, and service type...",

			"configuration_request": "Network slicing configuration involves setting up dedicated virtual network segments with specific QoS parameters...",

			"knowledge_request": "The O-RAN E2 interface is a standardized interface that connects the Near-RT RIC to E2 nodes...",

			"creation_intent": "Creating a network slice for IoT devices requires specific configuration for low-power, wide-area connectivity...",
		},

		latencySimulation: 100 * time.Millisecond,

		failureRate: 0.0,
	}

}

// ProcessRequest simulates LLM request processing.

func (m *MockLLMService) ProcessRequest(ctx context.Context, request *llm.IntentRequest) (*llm.IntentResponse, error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.callCount++

	// Simulate latency.

	time.Sleep(m.latencySimulation)

	// Simulate random failures.

	if rand.Float64() < m.failureRate {

		return nil, errors.New("simulated LLM service failure")

	}

	// Check for configured errors.

	if err, exists := m.errors[request.Intent]; exists {

		return nil, err

	}

	// Generate response.

	responseText, exists := m.responses[request.Intent]

	if !exists {

		responseText = fmt.Sprintf("Generated response for: %s", request.Intent)

	}

	return &llm.IntentResponse{

		Response: responseText,

		Confidence: float32(0.8 + rand.Float64()*0.2), // 0.8-1.0 (cast to float32)

		TokensUsed: 50 + rand.Intn(200), // 50-250 tokens

		ProcessingTime: m.latencySimulation,

		ModelUsed: "mock-llm-model",

		CacheHit: false,

		Metadata: map[string]interface{}{

			"mock_call_count": m.callCount,

			"model_name": "mock-llm-model",
		},
	}, nil

}

// SetResponse configures a response for a specific intent type.

func (m *MockLLMService) SetResponse(intentType, response string) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.responses[intentType] = response

}

// SetError configures an error for a specific intent type.

func (m *MockLLMService) SetError(intentType string, err error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.errors[intentType] = err

}

// SetLatency configures response latency simulation.

func (m *MockLLMService) SetLatency(latency time.Duration) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.latencySimulation = latency

}

// SetFailureRate configures random failure rate (0.0-1.0).

func (m *MockLLMService) SetFailureRate(rate float64) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.failureRate = rate

}

// GetCallCount returns the number of calls made.

func (m *MockLLMService) GetCallCount() int {

	m.mu.RLock()

	defer m.mu.RUnlock()

	return m.callCount

}

// MockWeaviateService simulates Weaviate vector database.

type MockWeaviateService struct {
	mu sync.RWMutex

	documents map[string]llm.Document

	vectors map[string][]float32

	searchResults map[string][]RetrievedDocument

	errors map[string]error

	latency time.Duration

	queryCount int

	failureRate float64
}

// NewMockWeaviateService creates a new mock Weaviate service.

func NewMockWeaviateService() *MockWeaviateService {

	return &MockWeaviateService{

		documents: make(map[string]llm.Document),

		vectors: make(map[string][]float32),

		searchResults: make(map[string][]RetrievedDocument),

		errors: make(map[string]error),

		latency: 50 * time.Millisecond,

		failureRate: 0.0,
	}

}

// StoreDocument simulates document storage.

func (m *MockWeaviateService) StoreDocument(ctx context.Context, doc llm.Document) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	// Simulate latency.

	time.Sleep(m.latency)

	// Check for configured errors.

	if err, exists := m.errors["store"]; exists {

		return err

	}

	// Simulate random failures.

	if rand.Float64() < m.failureRate {

		return errors.New("simulated Weaviate storage failure")

	}

	// Store document.

	m.documents[doc.ID] = doc

	// Generate mock vector.

	vector := make([]float32, 384) // Typical sentence transformer dimension

	for i := range vector {

		vector[i] = rand.Float32()*2 - 1 // Random values between -1 and 1

	}

	m.vectors[doc.ID] = vector

	return nil

}

// GetDocument simulates document retrieval.

func (m *MockWeaviateService) GetDocument(ctx context.Context, docID string) (*llm.Document, error) {

	m.mu.RLock()

	defer m.mu.RUnlock()

	// Simulate latency.

	time.Sleep(m.latency / 2)

	// Check for configured errors.

	if err, exists := m.errors["get"]; exists {

		return nil, err

	}

	doc, exists := m.documents[docID]

	if !exists {

		return nil, errors.New("document not found")

	}

	return &doc, nil

}

// SearchSimilar simulates vector similarity search.

func (m *MockWeaviateService) SearchSimilar(ctx context.Context, query string, limit int) ([]RetrievedDocument, error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.queryCount++

	// Simulate latency.

	time.Sleep(m.latency)

	// Check for configured errors.

	if err, exists := m.errors["search"]; exists {

		return nil, err

	}

	// Simulate random failures.

	if rand.Float64() < m.failureRate {

		return nil, errors.New("simulated Weaviate search failure")

	}

	// Check for pre-configured results.

	if results, exists := m.searchResults[query]; exists {

		return results, nil

	}

	// Generate mock search results.

	var results []RetrievedDocument

	// Return a subset of stored documents with random scores.

	count := 0

	for _, doc := range m.documents {

		if count >= limit {

			break

		}

		// Simple keyword matching for more realistic results.

		score := m.calculateSimilarityScore(query, doc.Content)

		if score > 0.3 { // Minimum relevance threshold

			results = append(results, RetrievedDocument{

				ID: doc.ID,

				Content: doc.Content,

				Score: float32(score),

				Source: doc.Source,

				Metadata: doc.Metadata,
			})

			count++

		}

	}

	// Sort by score (descending).

	for i := 0; i < len(results)-1; i++ {

		for j := i + 1; j < len(results); j++ {

			if results[i].Score < results[j].Score {

				results[i], results[j] = results[j], results[i]

			}

		}

	}

	return results, nil

}

// calculateSimilarityScore provides simple keyword-based similarity scoring.

func (m *MockWeaviateService) calculateSimilarityScore(query, content string) float64 {

	// Simple keyword matching.

	queryWords := strings.Fields(strings.ToLower(query))

	contentWords := strings.Fields(strings.ToLower(content))

	matches := 0

	for _, qword := range queryWords {

		for _, cword := range contentWords {

			if strings.Contains(cword, qword) || strings.Contains(qword, cword) {

				matches++

				break

			}

		}

	}

	if len(queryWords) == 0 {

		return 0.0

	}

	baseScore := float64(matches) / float64(len(queryWords))

	// Add some randomness to make it more realistic.

	return baseScore + (rand.Float64()-0.5)*0.2

}

// SetSearchResults configures search results for a specific query.

func (m *MockWeaviateService) SetSearchResults(query string, results []RetrievedDocument) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.searchResults[query] = results

}

// SetError configures an error for a specific operation.

func (m *MockWeaviateService) SetError(operation string, err error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	if err != nil {

		m.errors[operation] = err

	} else {

		delete(m.errors, operation)

	}

}

// ClearError removes all configured errors.

func (m *MockWeaviateService) ClearError() {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.errors = make(map[string]error)

}

// SetLatency configures operation latency simulation.

func (m *MockWeaviateService) SetLatency(latency time.Duration) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.latency = latency

}

// GetQueryCount returns the number of search queries made.

func (m *MockWeaviateService) GetQueryCount() int {

	m.mu.RLock()

	defer m.mu.RUnlock()

	return m.queryCount

}

// MockRedisService simulates Redis cache behavior.

type MockRedisService struct {
	mu sync.RWMutex

	cache map[string]interface{}

	hitCount int

	missCount int

	errors map[string]error

	latency time.Duration

	failureRate float64
}

// NewMockRedisService creates a new mock Redis service.

func NewMockRedisService() *MockRedisService {

	return &MockRedisService{

		cache: make(map[string]interface{}),

		errors: make(map[string]error),

		latency: 5 * time.Millisecond,

		failureRate: 0.0,
	}

}

// Get simulates cache retrieval.

func (m *MockRedisService) Get(ctx context.Context, key string) (interface{}, bool, error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	// Simulate latency.

	time.Sleep(m.latency)

	// Check for configured errors.

	if err, exists := m.errors["get"]; exists {

		return nil, false, err

	}

	// Simulate random failures.

	if rand.Float64() < m.failureRate {

		return nil, false, errors.New("simulated Redis get failure")

	}

	value, exists := m.cache[key]

	if exists {

		m.hitCount++

		return value, true, nil

	}

	m.missCount++

	return nil, false, nil

}

// Set simulates cache storage.

func (m *MockRedisService) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	// Simulate latency.

	time.Sleep(m.latency)

	// Check for configured errors.

	if err, exists := m.errors["set"]; exists {

		return err

	}

	// Simulate random failures.

	if rand.Float64() < m.failureRate {

		return errors.New("simulated Redis set failure")

	}

	m.cache[key] = value

	// Simulate expiration (simplified).

	if expiration > 0 {

		go func() {

			time.Sleep(expiration)

			m.mu.Lock()

			defer m.mu.Unlock()

			delete(m.cache, key)

		}()

	}

	return nil

}

// Delete simulates cache deletion.

func (m *MockRedisService) Delete(ctx context.Context, key string) error {

	m.mu.Lock()

	defer m.mu.Unlock()

	// Simulate latency.

	time.Sleep(m.latency)

	// Check for configured errors.

	if err, exists := m.errors["delete"]; exists {

		return err

	}

	delete(m.cache, key)

	return nil

}

// GetHitRate returns cache hit rate.

func (m *MockRedisService) GetHitRate() float64 {

	m.mu.RLock()

	defer m.mu.RUnlock()

	total := m.hitCount + m.missCount

	if total == 0 {

		return 0.0

	}

	return float64(m.hitCount) / float64(total)

}

// GetStats returns cache statistics.

func (m *MockRedisService) GetStats() map[string]interface{} {

	m.mu.RLock()

	defer m.mu.RUnlock()

	return map[string]interface{}{

		"hits": m.hitCount,

		"misses": m.missCount,

		"hit_rate": m.GetHitRate(),

		"cache_size": len(m.cache),
	}

}

// SetError configures an error for a specific operation.

func (m *MockRedisService) SetError(operation string, err error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	if err != nil {

		m.errors[operation] = err

	} else {

		delete(m.errors, operation)

	}

}

// ClearErrors removes all configured errors.

func (m *MockRedisService) ClearErrors() {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.errors = make(map[string]error)

}

// SetLatency configures operation latency simulation.

func (m *MockRedisService) SetLatency(latency time.Duration) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.latency = latency

}

// SetFailureRate configures random failure rate (0.0-1.0).

func (m *MockRedisService) SetFailureRate(rate float64) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.failureRate = rate

}

// Clear removes all cached data.

func (m *MockRedisService) Clear() {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.cache = make(map[string]interface{})

	m.hitCount = 0

	m.missCount = 0

}

// MockEmbeddingService simulates embedding generation.

type MockEmbeddingService struct {
	mu sync.RWMutex

	embeddings map[string][]float32

	errors map[string]error

	latency time.Duration

	failureRate float64

	callCount int
}

// NewMockEmbeddingService creates a new mock embedding service.

func NewMockEmbeddingService() *MockEmbeddingService {

	return &MockEmbeddingService{

		embeddings: make(map[string][]float32),

		errors: make(map[string]error),

		latency: 200 * time.Millisecond,

		failureRate: 0.0,
	}

}

// GenerateEmbeddings simulates embedding generation.

func (m *MockEmbeddingService) GenerateEmbeddings(ctx context.Context, texts []string) ([][]float32, error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.callCount++

	// Simulate latency proportional to input size.

	processingTime := time.Duration(len(texts)) * m.latency

	time.Sleep(processingTime)

	// Check for configured errors.

	if err, exists := m.errors["generate"]; exists {

		return nil, err

	}

	// Simulate random failures.

	if rand.Float64() < m.failureRate {

		return nil, errors.New("simulated embedding generation failure")

	}

	var results [][]float32

	for _, text := range texts {

		// Check cache first.

		if embedding, exists := m.embeddings[text]; exists {

			results = append(results, embedding)

			continue

		}

		// Generate new embedding.

		embedding := make([]float32, 384) // Standard sentence transformer size

		// Simple deterministic generation based on text content.

		hash := simpleHash(text)

		rng := rand.New(rand.NewSource(int64(hash)))

		for i := range embedding {

			embedding[i] = rng.Float32()*2 - 1 // Values between -1 and 1

		}

		// Normalize to unit vector.

		norm := float32(0)

		for _, val := range embedding {

			norm += val * val

		}

		norm = float32(math.Sqrt(float64(norm)))

		if norm > 0 {

			for i := range embedding {

				embedding[i] /= norm

			}

		}

		// Cache the embedding.

		m.embeddings[text] = embedding

		results = append(results, embedding)

	}

	return results, nil

}

// simpleHash provides a simple hash function for deterministic randomization.

func simpleHash(s string) uint32 {

	hash := uint32(2166136261)

	for _, b := range []byte(s) {

		hash ^= uint32(b)

		hash *= 16777619

	}

	return hash

}

// SetError configures an error for operations.

func (m *MockEmbeddingService) SetError(operation string, err error) {

	m.mu.Lock()

	defer m.mu.Unlock()

	if err != nil {

		m.errors[operation] = err

	} else {

		delete(m.errors, operation)

	}

}

// GetCallCount returns the number of generation calls made.

func (m *MockEmbeddingService) GetCallCount() int {

	m.mu.RLock()

	defer m.mu.RUnlock()

	return m.callCount

}

// SetLatency configures embedding generation latency per text.

func (m *MockEmbeddingService) SetLatency(latency time.Duration) {

	m.mu.Lock()

	defer m.mu.Unlock()

	m.latency = latency

}
