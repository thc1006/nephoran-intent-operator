// Package framework provides comprehensive mocking infrastructure for testing
package framework

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/mock"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// MockManager manages all mock services and dependencies
type MockManager struct {
	// HTTP mock servers
	weaviateServer   *httptest.Server
	llmServer        *httptest.Server
	oranServer       *httptest.Server
	prometheusServer *httptest.Server

	// Service mocks
	weaviateMock *MockWeaviateClient
	llmMock      *MockLLMClient
	redisMock    *MockRedisClient
	k8sMock      *MockK8sClient

	// Chaos injection
	chaosEnabled     bool
	failureRate      float64
	latencyInjection time.Duration

	// Request tracking
	requestCounts   map[string]int
	responseLatency map[string][]time.Duration

	// Synchronization
	mu sync.RWMutex

	// Configuration
	config *TestConfig
}

// NewMockManager creates a new mock manager
func NewMockManager() *MockManager {
	return &MockManager{
		requestCounts:   make(map[string]int),
		responseLatency: make(map[string][]time.Duration),
	}
}

// Initialize sets up all mock services
func (mm *MockManager) Initialize(config *TestConfig) {
	mm.config = config

	if config.MockExternalAPIs {
		mm.setupWeaviateMock()
		mm.setupLLMMock()
		mm.setupORANMock()
		mm.setupPrometheusMock()
	}

	mm.setupServiceMocks()
}

// Reset resets all mocks to their initial state
func (mm *MockManager) Reset() {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	// Reset request tracking
	mm.requestCounts = make(map[string]int)
	mm.responseLatency = make(map[string][]time.Duration)

	// Reset service mocks
	if mm.weaviateMock != nil {
		mm.weaviateMock.Reset()
	}
	if mm.llmMock != nil {
		mm.llmMock.Reset()
	}
	if mm.redisMock != nil {
		mm.redisMock.Reset()
	}
	if mm.k8sMock != nil {
		mm.k8sMock.Reset()
	}
}

// Cleanup stops all mock servers and cleans up resources
func (mm *MockManager) Cleanup() {
	if mm.weaviateServer != nil {
		mm.weaviateServer.Close()
	}
	if mm.llmServer != nil {
		mm.llmServer.Close()
	}
	if mm.oranServer != nil {
		mm.oranServer.Close()
	}
	if mm.prometheusServer != nil {
		mm.prometheusServer.Close()
	}
}

// setupWeaviateMock creates a mock Weaviate server
func (mm *MockManager) setupWeaviateMock() {
	router := mux.NewRouter()

	// Health check endpoint
	router.HandleFunc("/v1/.well-known/ready", func(w http.ResponseWriter, r *http.Request) {
		mm.trackRequest("weaviate_health")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]bool{"ready": true})
	}).Methods("GET")

	// GraphQL endpoint for queries
	router.HandleFunc("/v1/graphql", func(w http.ResponseWriter, r *http.Request) {
		mm.trackRequest("weaviate_query")

		if mm.shouldInjectFailure() {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		// Mock response for semantic search
		response := map[string]interface{}{
			"data": map[string]interface{}{
				"Get": map[string]interface{}{
					"TelecomKnowledge": []map[string]interface{}{
						{
							"title":   "AMF Configuration Guide",
							"content": "Access and Mobility Management Function configuration for 5G networks...",
							"_additional": map[string]interface{}{
								"certainty": 0.95,
							},
						},
						{
							"title":   "SMF Deployment Procedures",
							"content": "Session Management Function deployment in containerized environments...",
							"_additional": map[string]interface{}{
								"certainty": 0.87,
							},
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}).Methods("POST")

	// Object creation endpoint
	router.HandleFunc("/v1/objects", func(w http.ResponseWriter, r *http.Request) {
		mm.trackRequest("weaviate_create")

		if mm.shouldInjectFailure() {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"id": "mock-object-id",
		})
	}).Methods("POST")

	mm.weaviateServer = httptest.NewServer(router)
	mm.weaviateMock = &MockWeaviateClient{}
}

// setupLLMMock creates a mock LLM provider server
func (mm *MockManager) setupLLMMock() {
	router := mux.NewRouter()

	// Chat completions endpoint (OpenAI-compatible)
	router.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		mm.trackRequest("llm_completion")

		if mm.shouldInjectFailure() {
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{
				"error": "Rate limit exceeded",
			})
			return
		}

		// Mock structured response for network intent processing
		response := map[string]interface{}{
			"id":      "mock-completion-id",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   "gpt-4o-mini",
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]interface{}{
						"role": "assistant",
						"content": `{
							"type": "NetworkFunctionDeployment",
							"networkFunction": "AMF",
							"replicas": 3,
							"namespace": "telecom-core",
							"resources": {
								"requests": {"cpu": "1000m", "memory": "2Gi"},
								"limits": {"cpu": "2000m", "memory": "4Gi"}
							},
							"ports": [
								{"name": "sbi", "port": 8080, "protocol": "TCP"}
							],
							"config": {
								"plmn": {"mcc": "001", "mnc": "01"},
								"slice_support": ["eMBB", "URLLC"]
							}
						}`,
					},
					"finish_reason": "stop",
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     150,
				"completion_tokens": 200,
				"total_tokens":      350,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}).Methods("POST")

	// Embeddings endpoint
	router.HandleFunc("/v1/embeddings", func(w http.ResponseWriter, r *http.Request) {
		mm.trackRequest("llm_embedding")

		if mm.shouldInjectFailure() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		// Mock embedding response
		response := map[string]interface{}{
			"object": "list",
			"data": []map[string]interface{}{
				{
					"object":    "embedding",
					"embedding": mm.generateMockEmbedding(1536), // Standard embedding size
					"index":     0,
				},
			},
			"model": "text-embedding-3-large",
			"usage": map[string]int{
				"prompt_tokens": 50,
				"total_tokens":  50,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}).Methods("POST")

	mm.llmServer = httptest.NewServer(router)
	mm.llmMock = &MockLLMClient{}
}

// setupORANMock creates mock O-RAN interface servers
func (mm *MockManager) setupORANMock() {
	router := mux.NewRouter()

	// A1 Policy Management Interface
	router.HandleFunc("/a1-p/v2/policytypes", func(w http.ResponseWriter, r *http.Request) {
		mm.trackRequest("oran_a1_policy_types")

		response := []map[string]interface{}{
			{
				"policy_type_id": 1000,
				"name":           "QoS Policy",
				"description":    "Quality of Service policy for network slices",
			},
			{
				"policy_type_id": 2000,
				"name":           "Traffic Steering Policy",
				"description":    "Traffic steering policy for load balancing",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	// O1 Interface (NETCONF/RESTCONF)
	router.HandleFunc("/restconf/data/ietf-interfaces:interfaces", func(w http.ResponseWriter, r *http.Request) {
		mm.trackRequest("oran_o1_interfaces")

		response := map[string]interface{}{
			"ietf-interfaces:interfaces": map[string]interface{}{
				"interface": []map[string]interface{}{
					{
						"name":    "eth0",
						"type":    "iana-if-type:ethernetCsmacd",
						"enabled": true,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	// O2 Interface (Cloud Infrastructure)
	router.HandleFunc("/o2/v1/deployments", func(w http.ResponseWriter, r *http.Request) {
		mm.trackRequest("oran_o2_deployments")

		if r.Method == "POST" {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"deployment_id": "mock-deployment-123",
				"status":        "creating",
			})
		} else {
			response := []map[string]interface{}{
				{
					"deployment_id": "mock-deployment-123",
					"name":          "amf-deployment",
					"status":        "running",
					"replicas":      3,
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}
	}).Methods("GET", "POST")

	mm.oranServer = httptest.NewServer(router)
}

// setupPrometheusMock creates a mock Prometheus server
func (mm *MockManager) setupPrometheusMock() {
	router := mux.NewRouter()

	// Query endpoint
	router.HandleFunc("/api/v1/query", func(w http.ResponseWriter, r *http.Request) {
		mm.trackRequest("prometheus_query")

		response := map[string]interface{}{
			"status": "success",
			"data": map[string]interface{}{
				"resultType": "vector",
				"result": []map[string]interface{}{
					{
						"metric": map[string]string{
							"__name__": "nephran_intent_processing_duration_seconds",
							"instance": "localhost:8080",
						},
						"value": []interface{}{
							time.Now().Unix(),
							"1.234",
						},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	mm.prometheusServer = httptest.NewServer(router)
}

// setupServiceMocks initializes service-level mocks
func (mm *MockManager) setupServiceMocks() {
	mm.weaviateMock = &MockWeaviateClient{}
	mm.llmMock = &MockLLMClient{}
	mm.redisMock = &MockRedisClient{}
	mm.k8sMock = &MockK8sClient{}
}

// Mock service implementations

// MockWeaviateClient mocks the Weaviate client
type MockWeaviateClient struct {
	mock.Mock
}

func (m *MockWeaviateClient) Query() interface{} {
	args := m.Called()
	return args.Get(0)
}

func (m *MockWeaviateClient) Reset() {
	m.ExpectedCalls = nil
	m.Calls = nil
}

// MockLLMClient mocks LLM service calls
type MockLLMClient struct {
	mock.Mock
}

func (m *MockLLMClient) ProcessIntent(ctx context.Context, intent string) (map[string]interface{}, error) {
	args := m.Called(ctx, intent)
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockLLMClient) Reset() {
	m.ExpectedCalls = nil
	m.Calls = nil
}

// MockRedisClient mocks Redis operations
type MockRedisClient struct {
	mock.Mock
}

func (m *MockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	args := m.Called(ctx, key)
	return args.Get(0).(*redis.StringCmd)
}

func (m *MockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	args := m.Called(ctx, key, value, expiration)
	return args.Get(0).(*redis.StatusCmd)
}

func (m *MockRedisClient) Reset() {
	m.ExpectedCalls = nil
	m.Calls = nil
}

// MockK8sClient mocks Kubernetes client operations
type MockK8sClient struct {
	mock.Mock
}

func (m *MockK8sClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	args := m.Called(ctx, key, obj, opts)
	return args.Error(0)
}

func (m *MockK8sClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockK8sClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockK8sClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
	args := m.Called(ctx, obj, opts)
	return args.Error(0)
}

func (m *MockK8sClient) Reset() {
	m.ExpectedCalls = nil
	m.Calls = nil
}

// Chaos engineering methods

// InjectChaos enables chaos injection with specified failure rate
func (mm *MockManager) InjectChaos(failureRate float64, testFunc func() error) error {
	mm.mu.Lock()
	mm.chaosEnabled = true
	mm.failureRate = failureRate
	mm.latencyInjection = time.Duration(rand.Intn(1000)) * time.Millisecond
	mm.mu.Unlock()

	defer func() {
		mm.mu.Lock()
		mm.chaosEnabled = false
		mm.failureRate = 0
		mm.latencyInjection = 0
		mm.mu.Unlock()
	}()

	return testFunc()
}

// shouldInjectFailure determines if a failure should be injected
func (mm *MockManager) shouldInjectFailure() bool {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	if !mm.chaosEnabled {
		return false
	}

	// Inject latency
	if mm.latencyInjection > 0 {
		time.Sleep(mm.latencyInjection)
	}

	// Inject failure based on rate
	return rand.Float64() < mm.failureRate
}

// trackRequest tracks mock service requests for analysis
func (mm *MockManager) trackRequest(service string) {
	mm.mu.Lock()
	defer mm.mu.Unlock()

	mm.requestCounts[service]++

	// Track latency (simulated)
	latency := time.Duration(rand.Intn(100)) * time.Millisecond
	mm.responseLatency[service] = append(mm.responseLatency[service], latency)
}

// generateMockEmbedding creates a mock embedding vector
func (mm *MockManager) generateMockEmbedding(dimensions int) []float64 {
	embedding := make([]float64, dimensions)
	for i := range embedding {
		embedding[i] = rand.Float64()*2 - 1 // Random values between -1 and 1
	}
	return embedding
}

// GetMockServerURLs returns URLs for mock servers
func (mm *MockManager) GetMockServerURLs() map[string]string {
	urls := make(map[string]string)

	if mm.weaviateServer != nil {
		urls["weaviate"] = mm.weaviateServer.URL
	}
	if mm.llmServer != nil {
		urls["llm"] = mm.llmServer.URL
	}
	if mm.oranServer != nil {
		urls["oran"] = mm.oranServer.URL
	}
	if mm.prometheusServer != nil {
		urls["prometheus"] = mm.prometheusServer.URL
	}

	return urls
}

// GenerateReport creates a comprehensive mock interaction report
func (mm *MockManager) GenerateReport() {
	mm.mu.RLock()
	defer mm.mu.RUnlock()

	fmt.Println("=== Mock Service Interaction Report ===")

	for service, count := range mm.requestCounts {
		fmt.Printf("Service: %s, Requests: %d\n", service, count)

		if latencies, exists := mm.responseLatency[service]; exists && len(latencies) > 0 {
			var total time.Duration
			for _, lat := range latencies {
				total += lat
			}
			avg := total / time.Duration(len(latencies))
			fmt.Printf("  Average Latency: %v\n", avg)
		}
	}
}

// GetWeaviateMock returns the Weaviate mock client
func (mm *MockManager) GetWeaviateMock() *MockWeaviateClient {
	return mm.weaviateMock
}

// GetLLMMock returns the LLM mock client
func (mm *MockManager) GetLLMMock() *MockLLMClient {
	return mm.llmMock
}

// GetRedisMock returns the Redis mock client
func (mm *MockManager) GetRedisMock() *MockRedisClient {
	return mm.redisMock
}

// GetK8sMock returns the Kubernetes mock client
func (mm *MockManager) GetK8sMock() *MockK8sClient {
	return mm.k8sMock
}
