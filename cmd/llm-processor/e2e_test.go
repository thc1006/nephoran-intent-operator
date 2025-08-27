//go:build integration

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/handlers"
	"github.com/thc1006/nephoran-intent-operator/pkg/services"
)

// Type definitions for test requests and responses
type NetworkIntentRequest struct {
	Spec struct {
		Intent string `json:"intent"`
	} `json:"spec"`
	Metadata struct {
		Name       string `json:"name,omitempty"`
		Namespace  string `json:"namespace,omitempty"`
		UID        string `json:"uid,omitempty"`
		Generation int64  `json:"generation,omitempty"`
	} `json:"metadata"`
}

type NetworkIntentResponse struct {
	Type               string      `json:"type"`
	Name               string      `json:"name"`
	Namespace          string      `json:"namespace"`
	OriginalIntent     string      `json:"originalIntent"`
	Spec               interface{} `json:"spec"`
	ProcessingMetadata struct {
		ModelUsed       string  `json:"modelUsed"`
		ConfidenceScore float64 `json:"confidenceScore"`
	} `json:"processingMetadata"`
}

type HealthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

type ReadinessResponse struct {
	Status string `json:"status"`
}

type ErrorResponse struct {
	ErrorCode string `json:"errorCode"`
	Message   string `json:"message,omitempty"`
}

func TestE2E(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LLM Processor E2E Suite")
}

var _ = Describe("End-to-End LLM Processor Tests", func() {
	var (
		server      *httptest.Server
		mockLLM     *httptest.Server
		testConfig  *config.LLMProcessorConfig
		testLogger  *slog.Logger
		testService *services.LLMProcessorService
		testHandler *handlers.LLMProcessorHandler
	)

	BeforeEach(func() {
		// Create mock LLM server
		mockLLM = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var requestBody map[string]interface{}
			err := json.NewDecoder(r.Body).Decode(&requestBody)
			if err != nil {
				http.Error(w, "Invalid JSON", http.StatusBadRequest)
				return
			}

			// Extract intent from messages
			messages := requestBody["messages"].([]interface{})
			userMessage := messages[len(messages)-1].(map[string]interface{})
			intent := userMessage["content"].(string)

			// Generate response based on intent
			response := generateMockLLMResponse(intent)

			llmResponse := map[string]interface{}{
				"choices": []map[string]interface{}{
					{
						"message": map[string]interface{}{
							"content": response,
						},
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			if err := json.NewEncoder(w).Encode(llmResponse); err != nil {
				log.Printf("Error encoding JSON: %v", err)
				return
			}
		}))

		// Set up test configuration
		testConfig = &config.LLMProcessorConfig{
			Port:             "0", // Let the system choose a port
			LogLevel:         "info",
			ServiceVersion:   "test-v1.0.0",
			GracefulShutdown: 5 * time.Second,

			LLMBackendType: "openai",
			LLMAPIKey:      "test-key",
			LLMModelName:   "test-model",
			LLMTimeout:     10 * time.Second,
			LLMMaxTokens:   1000,

			RAGEnabled: false,
			RAGAPIURL:  mockLLM.URL,
			RAGTimeout: 10 * time.Second,

			CircuitBreakerEnabled:   true,
			CircuitBreakerThreshold: 5,
			CircuitBreakerTimeout:   30 * time.Second,

			MaxRetries:   2,
			RetryDelay:   100 * time.Millisecond,
			RetryBackoff: "exponential",

			CORSEnabled: true,
			AuthEnabled: false,
			RequireAuth: false,

			RequestTimeout: 30 * time.Second,
		}

		// Initialize test logger
		testLogger = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))

		// Initialize service components
		testService = services.NewLLMProcessorService(testConfig, testLogger)
		ctx := context.Background()
		err := testService.Initialize(ctx)
		Expect(err).ToNot(HaveOccurred())

		// Get initialized components
		processor, streamingProcessor, circuitBreakerMgr, tokenManager, contextBuilder, relevanceScorer, promptBuilder, healthChecker := testService.GetComponents()

		// Create handler with initialized components
		testHandler = handlers.NewLLMProcessorHandler(
			testConfig, processor, streamingProcessor, circuitBreakerMgr,
			tokenManager, contextBuilder, relevanceScorer, promptBuilder,
			testLogger, healthChecker, time.Now(),
		)

		// Create test server with handler routes
		mux := http.NewServeMux()
		mux.HandleFunc("/process", testHandler.ProcessIntentHandler)
		mux.HandleFunc("/healthz", healthChecker.HealthzHandler)
		mux.HandleFunc("/readyz", healthChecker.ReadyzHandler)
		mux.HandleFunc("/status", testHandler.StatusHandler)
		mux.HandleFunc("/metrics", testHandler.MetricsHandler)

		server = httptest.NewServer(mux)
	})

	AfterEach(func() {
		server.Close()
		mockLLM.Close()
		if testService != nil {
			ctx := context.Background()
			testService.Shutdown(ctx)
		}
	})

	Context("Service Health and Readiness", func() {
		It("should respond to health checks", func() {
			resp, err := http.Get(server.URL + "/healthz")
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var healthResp HealthResponse
			err = json.NewDecoder(resp.Body).Decode(&healthResp)
			Expect(err).ToNot(HaveOccurred())
			Expect(healthResp.Status).To(Equal("ok"))
			Expect(healthResp.Version).To(Equal("test-v1.0.0"))
		})

		It("should respond to readiness checks", func() {
			resp, err := http.Get(server.URL + "/readyz")
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var readyResp ReadinessResponse
			err = json.NewDecoder(resp.Body).Decode(&readyResp)
			Expect(err).ToNot(HaveOccurred())
			Expect(readyResp.Status).To(Equal("ready"))
		})

		It("should provide service information", func() {
			resp, err := http.Get(server.URL + "/info")
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var info map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&info)
			Expect(err).ToNot(HaveOccurred())
			Expect(info).To(HaveKey("service"))
			Expect(info).To(HaveKey("configuration"))
			Expect(info).To(HaveKey("runtime"))
		})

		It("should provide version information", func() {
			resp, err := http.Get(server.URL + "/version")
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var version map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&version)
			Expect(err).ToNot(HaveOccurred())
			Expect(version["version"]).To(Equal("test-v1.0.0"))
		})
	})

	Context("Intent Processing", func() {
		It("should process UPF deployment intent", func() {
			intent := NetworkIntentRequest{
				Spec: struct {
					Intent string `json:"intent"`
				}{
					Intent: "Deploy UPF network function with 3 replicas for high availability",
				},
				Metadata: struct {
					Name       string `json:"name,omitempty"`
					Namespace  string `json:"namespace,omitempty"`
					UID        string `json:"uid,omitempty"`
					Generation int64  `json:"generation,omitempty"`
				}{
					Name:      "test-upf",
					Namespace: "5g-core",
				},
			}

			reqBody, err := json.Marshal(intent)
			Expect(err).ToNot(HaveOccurred())

			resp, err := http.Post(server.URL+"/process", "application/json", bytes.NewBuffer(reqBody))
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var response NetworkIntentResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			Expect(err).ToNot(HaveOccurred())

			Expect(response.Type).To(Equal("NetworkFunctionDeployment"))
			Expect(response.Name).ToNot(BeEmpty())
			Expect(response.Namespace).ToNot(BeEmpty())
			Expect(response.OriginalIntent).To(Equal(intent.Spec.Intent))
			Expect(response.ProcessingMetadata.ModelUsed).To(Equal("test-model"))
			Expect(response.ProcessingMetadata.ConfidenceScore).To(BeNumerically(">", 0))
		})

		It("should process AMF scaling intent", func() {
			intent := NetworkIntentRequest{
				Spec: struct {
					Intent string `json:"intent"`
				}{
					Intent: "Scale AMF instances to 5 replicas to handle increased signaling load",
				},
			}

			reqBody, err := json.Marshal(intent)
			Expect(err).ToNot(HaveOccurred())

			resp, err := http.Post(server.URL+"/process", "application/json", bytes.NewBuffer(reqBody))
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var response NetworkIntentResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			Expect(err).ToNot(HaveOccurred())

			Expect(response.Type).To(Equal("NetworkFunctionScale"))
			Expect(response.Name).ToNot(BeEmpty())
		})

		It("should process Near-RT RIC deployment intent", func() {
			intent := NetworkIntentRequest{
				Spec: struct {
					Intent string `json:"intent"`
				}{
					Intent: "Setup Near-RT RIC with xApp support for intelligent traffic management",
				},
			}

			reqBody, err := json.Marshal(intent)
			Expect(err).ToNot(HaveOccurred())

			resp, err := http.Post(server.URL+"/process", "application/json", bytes.NewBuffer(reqBody))
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			var response NetworkIntentResponse
			err = json.NewDecoder(resp.Body).Decode(&response)
			Expect(err).ToNot(HaveOccurred())

			Expect(response.Type).To(Equal("NetworkFunctionDeployment"))
			Expect(response.Name).To(ContainSubstring("ric"))
		})
	})

	Context("Error Handling", func() {
		It("should handle invalid request format", func() {
			invalidJSON := `{"invalid": "json"}`

			resp, err := http.Post(server.URL+"/process", "application/json", strings.NewReader(invalidJSON))
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

			var errorResp ErrorResponse
			err = json.NewDecoder(resp.Body).Decode(&errorResp)
			Expect(err).ToNot(HaveOccurred())
			Expect(errorResp.ErrorCode).To(Equal("INVALID_REQUEST"))
		})

		It("should handle method not allowed", func() {
			resp, err := http.Get(server.URL + "/process")
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
		})

		It("should handle empty intent", func() {
			intent := NetworkIntentRequest{
				Spec: struct {
					Intent string `json:"intent"`
				}{
					Intent: "",
				},
			}

			reqBody, err := json.Marshal(intent)
			Expect(err).ToNot(HaveOccurred())

			resp, err := http.Post(server.URL+"/process", "application/json", bytes.NewBuffer(reqBody))
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))
		})

		It("should handle LLM service failures", func() {
			// Stop the mock LLM server to simulate failure
			mockLLM.Close()

			intent := NetworkIntentRequest{
				Spec: struct {
					Intent string `json:"intent"`
				}{
					Intent: "Deploy test network function",
				},
			}

			reqBody, err := json.Marshal(intent)
			Expect(err).ToNot(HaveOccurred())

			resp, err := http.Post(server.URL+"/process", "application/json", bytes.NewBuffer(reqBody))
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))

			var errorResp ErrorResponse
			err = json.NewDecoder(resp.Body).Decode(&errorResp)
			Expect(err).ToNot(HaveOccurred())
			Expect(errorResp.ErrorCode).To(Equal("PROCESSING_FAILED"))
		})
	})

	Context("CORS Support", func() {
		It("should handle CORS preflight requests", func() {
			req, err := http.NewRequest("OPTIONS", server.URL+"/process", nil)
			Expect(err).ToNot(HaveOccurred())

			req.Header.Set("Origin", "https://example.com")
			req.Header.Set("Access-Control-Request-Method", "POST")

			client := &http.Client{}
			resp, err := client.Do(req)
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).To(Equal(http.StatusNoContent))
			Expect(resp.Header.Get("Access-Control-Allow-Origin")).ToNot(BeEmpty())
		})

		It("should include CORS headers in responses", func() {
			intent := NetworkIntentRequest{
				Spec: struct {
					Intent string `json:"intent"`
				}{
					Intent: "Deploy test network function with 1 replica",
				},
			}

			reqBody, err := json.Marshal(intent)
			Expect(err).ToNot(HaveOccurred())

			req, err := http.NewRequest("POST", server.URL+"/process", bytes.NewBuffer(reqBody))
			Expect(err).ToNot(HaveOccurred())

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Origin", "https://example.com")

			client := &http.Client{}
			resp, err := client.Do(req)
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.Header.Get("Access-Control-Allow-Origin")).ToNot(BeEmpty())
		})
	})

	Context("Request Tracking", func() {
		It("should include request ID in responses", func() {
			intent := NetworkIntentRequest{
				Spec: struct {
					Intent string `json:"intent"`
				}{
					Intent: "Deploy UPF with request tracking",
				},
			}

			reqBody, err := json.Marshal(intent)
			Expect(err).ToNot(HaveOccurred())

			resp, err := http.Post(server.URL+"/process", "application/json", bytes.NewBuffer(reqBody))
			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.Header.Get("X-Request-ID")).ToNot(BeEmpty())
		})
	})

	Context("Concurrent Processing", func() {
		It("should handle multiple concurrent requests", func() {
			const numRequests = 20
			const numWorkers = 5

			var wg sync.WaitGroup
			results := make(chan error, numRequests)
			requests := make(chan int, numRequests)

			// Fill request queue
			for i := 0; i < numRequests; i++ {
				requests <- i
			}
			close(requests)

			// Start workers
			for i := 0; i < numWorkers; i++ {
				wg.Add(1)
				go func(workerID int) {
					defer wg.Done()
					client := &http.Client{Timeout: 30 * time.Second}

					for reqNum := range requests {
						intent := NetworkIntentRequest{
							Spec: struct {
								Intent string `json:"intent"`
							}{
								Intent: fmt.Sprintf("Deploy UPF-%d-%d with 2 replicas", workerID, reqNum),
							},
						}

						reqBody, err := json.Marshal(intent)
						if err != nil {
							results <- err
							continue
						}

						resp, err := client.Post(server.URL+"/process", "application/json", bytes.NewBuffer(reqBody))
						if err != nil {
							results <- err
							continue
						}

						if resp.StatusCode != http.StatusOK {
							results <- fmt.Errorf("unexpected status code: %d", resp.StatusCode)
						} else {
							results <- nil
						}
						resp.Body.Close()
					}
				}(i)
			}

			wg.Wait()
			close(results)

			// Check results
			successCount := 0
			errorCount := 0
			for err := range results {
				if err != nil {
					errorCount++
				} else {
					successCount++
				}
			}

			Expect(successCount).To(Equal(numRequests))
			Expect(errorCount).To(Equal(0))
		})
	})

	Context("Performance and Reliability", func() {
		It("should process requests within acceptable time limits", func() {
			intent := NetworkIntentRequest{
				Spec: struct {
					Intent string `json:"intent"`
				}{
					Intent: "Deploy high-performance UPF with GPU acceleration and SR-IOV networking",
				},
			}

			reqBody, err := json.Marshal(intent)
			Expect(err).ToNot(HaveOccurred())

			start := time.Now()
			resp, err := http.Post(server.URL+"/process", "application/json", bytes.NewBuffer(reqBody))
			duration := time.Since(start)

			Expect(err).ToNot(HaveOccurred())
			defer func() { _ = resp.Body.Close() }()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))
			Expect(duration).To(BeNumerically("<", 10*time.Second), "Request should complete within 10 seconds")
		})

		It("should handle parameter extraction correctly", func() {
			testCases := []struct {
				intent           string
				expectedReplicas interface{}
				expectedCPU      string
				expectedMemory   string
			}{
				{
					intent:           "Deploy UPF with 5 replicas",
					expectedReplicas: float64(5),
				},
				{
					intent:      "Scale AMF to 4 CPU cores",
					expectedCPU: "4000m",
				},
				{
					intent:         "Allocate 8GB memory for SMF",
					expectedMemory: "8Gi",
				},
			}

			for _, tc := range testCases {
				intent := NetworkIntentRequest{
					Spec: struct {
						Intent string `json:"intent"`
					}{
						Intent: tc.intent,
					},
				}

				reqBody, err := json.Marshal(intent)
				Expect(err).ToNot(HaveOccurred())

				resp, err := http.Post(server.URL+"/process", "application/json", bytes.NewBuffer(reqBody))
				Expect(err).ToNot(HaveOccurred())
				defer func() { _ = resp.Body.Close() }()

				Expect(resp.StatusCode).To(Equal(http.StatusOK))

				var response NetworkIntentResponse
				err = json.NewDecoder(resp.Body).Decode(&response)
				Expect(err).ToNot(HaveOccurred())

				spec, ok := response.Spec.(map[string]interface{})
				Expect(ok).To(BeTrue())

				if tc.expectedReplicas != nil {
					Expect(spec["replicas"]).To(Equal(tc.expectedReplicas))
				}

				if tc.expectedCPU != "" {
					resources, ok := spec["resources"].(map[string]interface{})
					Expect(ok).To(BeTrue())
					requests, ok := resources["requests"].(map[string]interface{})
					Expect(ok).To(BeTrue())
					Expect(requests["cpu"]).To(Equal(tc.expectedCPU))
				}

				if tc.expectedMemory != "" {
					resources, ok := spec["resources"].(map[string]interface{})
					Expect(ok).To(BeTrue())
					requests, ok := resources["requests"].(map[string]interface{})
					Expect(ok).To(BeTrue())
					Expect(requests["memory"]).To(Equal(tc.expectedMemory))
				}
			}
		})
	})
})

// generateMockLLMResponse generates appropriate mock responses based on intent
func generateMockLLMResponse(intent string) string {
	lowerIntent := strings.ToLower(intent)

	if strings.Contains(lowerIntent, "scale") {
		return `{
			"type": "NetworkFunctionScale",
			"name": "amf-core",
			"namespace": "5g-core",
			"spec": {
				"scaling": {
					"horizontal": {
						"replicas": 5,
						"min_replicas": 2,
						"max_replicas": 10,
						"target_cpu_utilization": 70
					}
				}
			}
		}`
	}

	if strings.Contains(lowerIntent, "ric") {
		return `{
			"type": "NetworkFunctionDeployment",
			"name": "near-rt-ric-xapp-platform",
			"namespace": "o-ran",
			"spec": {
				"replicas": 2,
				"image": "registry.oran.local/near-rt-ric:v3.0.0",
				"resources": {
					"requests": {"cpu": "1000m", "memory": "2Gi"},
					"limits": {"cpu": "2000m", "memory": "4Gi"}
				},
				"ports": [
					{"containerPort": 8080, "protocol": "TCP"},
					{"containerPort": 36421, "protocol": "SCTP"}
				],
				"env": [
					{"name": "RIC_MODE", "value": "near-rt"},
					{"name": "XAPP_REGISTRY", "value": "http://xapp-registry:8080"}
				]
			}
		}`
	}

	// Default deployment response
	var replicas float64 = 1
	var cpu string = "500m"
	var memory string = "1Gi"
	var name string = "network-function"
	var image string = "registry.5g.local/nf:latest"

	// Extract parameters from intent
	if strings.Contains(lowerIntent, "upf") {
		name = "upf-deployment"
		image = "registry.5g.local/upf:v2.1.0"
		cpu = "2000m"
		memory = "4Gi"
	} else if strings.Contains(lowerIntent, "amf") {
		name = "amf-deployment"
		image = "registry.5g.local/amf:v2.1.0"
	} else if strings.Contains(lowerIntent, "smf") {
		name = "smf-deployment"
		image = "registry.5g.local/smf:v2.1.0"
	}

	// Extract replicas
	if strings.Contains(lowerIntent, "3 replicas") {
		replicas = 3
	} else if strings.Contains(lowerIntent, "5 replicas") {
		replicas = 5
	} else if strings.Contains(lowerIntent, "2 replicas") {
		replicas = 2
	}

	// Extract CPU
	if strings.Contains(lowerIntent, "4 cpu") || strings.Contains(lowerIntent, "4 cores") {
		cpu = "4000m"
	}

	// Extract memory
	if strings.Contains(lowerIntent, "8gb") || strings.Contains(lowerIntent, "8 gb") {
		memory = "8Gi"
	} else if strings.Contains(lowerIntent, "4gb") || strings.Contains(lowerIntent, "4 gb") {
		memory = "4Gi"
	}

	response := map[string]interface{}{
		"type":      "NetworkFunctionDeployment",
		"name":      name,
		"namespace": "5g-core",
		"spec": map[string]interface{}{
			"replicas": replicas,
			"image":    image,
			"resources": map[string]interface{}{
				"requests": map[string]string{"cpu": cpu, "memory": memory},
				"limits":   map[string]string{"cpu": cpu, "memory": memory},
			},
			"ports": []map[string]interface{}{
				{"containerPort": float64(8080), "protocol": "TCP"},
			},
			"env": []map[string]interface{}{
				{"name": "LOG_LEVEL", "value": "info"},
			},
		},
	}

	responseJSON, _ := json.Marshal(response)
	return string(responseJSON)
}
