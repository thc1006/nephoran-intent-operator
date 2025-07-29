package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// Service structures matching the LLM processor
type ProcessIntentRequest struct {
	Intent   string            `json:"intent"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

type ProcessIntentResponse struct {
	Result          string                 `json:"result"`
	ProcessingTime  string                 `json:"processing_time"`
	RequestID       string                 `json:"request_id"`
	ServiceVersion  string                 `json:"service_version"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
	Status          string                 `json:"status"`
	Error           string                 `json:"error,omitempty"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Version   string `json:"version"`
	Uptime    string `json:"uptime"`
	Timestamp string `json:"timestamp"`
}

// LLMProcessorService represents the LLM processor service for testing
type LLMProcessorService struct {
	llmClient      llm.ClientInterface
	serviceVersion string
	startTime      time.Time
	healthy        bool
	ready          bool
}

// NewLLMProcessorService creates a new LLM processor service for testing
func NewLLMProcessorService(client llm.ClientInterface) *LLMProcessorService {
	return &LLMProcessorService{
		llmClient:      client,
		serviceVersion: "test-v1.0.0",
		startTime:      time.Now(),
		healthy:        true,
		ready:          true,
	}
}

// SetupHandler creates HTTP handlers for the LLM processor service
func (s *LLMProcessorService) SetupHandler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", s.healthzHandler)
	mux.HandleFunc("/readyz", s.readyzHandler)
	mux.HandleFunc("/process", s.processIntentHandler)
	mux.HandleFunc("/status", s.statusHandler)
	return mux
}

func (s *LLMProcessorService) healthzHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "healthy",
		Version:   s.serviceVersion,
		Uptime:    time.Since(s.startTime).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if !s.healthy {
		response.Status = "unhealthy"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *LLMProcessorService) readyzHandler(w http.ResponseWriter, r *http.Request) {
	response := HealthResponse{
		Status:    "ready",
		Version:   s.serviceVersion,
		Uptime:    time.Since(s.startTime).String(),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	if !s.ready {
		response.Status = "not ready"
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *LLMProcessorService) processIntentHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	startTime := time.Now()
	reqID := fmt.Sprintf("test-%d", time.Now().UnixNano())

	var req ProcessIntentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Intent == "" {
		http.Error(w, "Intent is required", http.StatusBadRequest)
		return
	}

	// Process intent using the LLM client
	result, err := s.llmClient.ProcessIntent(r.Context(), req.Intent)
	if err != nil {
		response := ProcessIntentResponse{
			Status:         "error",
			Error:          err.Error(),
			RequestID:      reqID,
			ServiceVersion: s.serviceVersion,
			ProcessingTime: time.Since(startTime).String(),
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(response)
		return
	}

	response := ProcessIntentResponse{
		Result:         result,
		Status:         "success",
		ProcessingTime: time.Since(startTime).String(),
		RequestID:      reqID,
		ServiceVersion: s.serviceVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *LLMProcessorService) statusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"service":     "llm-processor",
		"version":     s.serviceVersion,
		"uptime":      time.Since(s.startTime).String(),
		"healthy":     s.healthy,
		"ready":       s.ready,
		"timestamp":   time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (s *LLMProcessorService) SetHealthy(healthy bool) {
	s.healthy = healthy
}

func (s *LLMProcessorService) SetReady(ready bool) {
	s.ready = ready
}

var _ = Describe("LLM Processor Service Tests", func() {
	var (
		service    *LLMProcessorService
		testServer *httptest.Server
	)

	BeforeEach(func() {
		By("Setting up test LLM processor service")
		// Create mock LLM client for service testing
		mockClient := &MockLLMClient{
			Response: `{"action": "test", "result": "success"}`,
			Error:    nil,
		}
		service = NewLLMProcessorService(mockClient)
		testServer = httptest.NewServer(service.SetupHandler())
	})

	AfterEach(func() {
		By("Cleaning up test server")
		if testServer != nil {
			testServer.Close()
		}
	})

	Context("Health Check Endpoints", func() {
		It("Should return healthy status on /healthz", func() {
			By("Making request to health endpoint")
			resp, err := http.Get(testServer.URL + "/healthz")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By("Parsing health response")
			var healthResp HealthResponse
			Expect(json.NewDecoder(resp.Body).Decode(&healthResp)).To(Succeed())

			Expect(healthResp.Status).To(Equal("healthy"))
			Expect(healthResp.Version).To(Equal("test-v1.0.0"))
			Expect(healthResp.Uptime).NotTo(BeEmpty())
			Expect(healthResp.Timestamp).NotTo(BeEmpty())
		})

		It("Should return unhealthy status when service is unhealthy", func() {
			By("Setting service to unhealthy state")
			service.SetHealthy(false)

			By("Making request to health endpoint")
			resp, err := http.Get(testServer.URL + "/healthz")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusServiceUnavailable))

			By("Parsing health response")
			var healthResp HealthResponse
			Expect(json.NewDecoder(resp.Body).Decode(&healthResp)).To(Succeed())

			Expect(healthResp.Status).To(Equal("unhealthy"))
		})

		It("Should return ready status on /readyz", func() {
			By("Making request to readiness endpoint")
			resp, err := http.Get(testServer.URL + "/readyz")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By("Parsing readiness response")
			var readyResp HealthResponse
			Expect(json.NewDecoder(resp.Body).Decode(&readyResp)).To(Succeed())

			Expect(readyResp.Status).To(Equal("ready"))
			Expect(readyResp.Version).To(Equal("test-v1.0.0"))
		})

		It("Should return not ready status when service is not ready", func() {
			By("Setting service to not ready state")
			service.SetReady(false)

			By("Making request to readiness endpoint")
			resp, err := http.Get(testServer.URL + "/readyz")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusServiceUnavailable))

			By("Parsing readiness response")
			var readyResp HealthResponse
			Expect(json.NewDecoder(resp.Body).Decode(&readyResp)).To(Succeed())

			Expect(readyResp.Status).To(Equal("not ready"))
		})
	})

	Context("Status Endpoint", func() {
		It("Should return service status information", func() {
			By("Making request to status endpoint")
			resp, err := http.Get(testServer.URL + "/status")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By("Parsing status response")
			var statusResp map[string]interface{}
			Expect(json.NewDecoder(resp.Body).Decode(&statusResp)).To(Succeed())

			Expect(statusResp["service"]).To(Equal("llm-processor"))
			Expect(statusResp["version"]).To(Equal("test-v1.0.0"))
			Expect(statusResp["healthy"]).To(Equal(true))
			Expect(statusResp["ready"]).To(Equal(true))
			Expect(statusResp["uptime"]).NotTo(BeEmpty())
			Expect(statusResp["timestamp"]).NotTo(BeEmpty())
		})
	})

	Context("Intent Processing Endpoint", func() {
		It("Should successfully process valid intent requests", func() {
			By("Creating a valid intent request")
			request := ProcessIntentRequest{
				Intent: "Scale E2 nodes to 5 replicas",
				Metadata: map[string]string{
					"source":    "test",
					"namespace": "default",
				},
			}

			requestBody, err := json.Marshal(request)
			Expect(err).NotTo(HaveOccurred())

			By("Making POST request to process endpoint")
			resp, err := http.Post(
				testServer.URL+"/process",
				"application/json",
				bytes.NewBuffer(requestBody),
			)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By("Parsing processing response")
			var processResp ProcessIntentResponse
			Expect(json.NewDecoder(resp.Body).Decode(&processResp)).To(Succeed())

			Expect(processResp.Status).To(Equal("success"))
			Expect(processResp.Result).To(Equal(`{"action": "test", "result": "success"}`))
			Expect(processResp.RequestID).NotTo(BeEmpty())
			Expect(processResp.ServiceVersion).To(Equal("test-v1.0.0"))
			Expect(processResp.ProcessingTime).NotTo(BeEmpty())
			Expect(processResp.Error).To(BeEmpty())
		})

		It("Should reject requests with empty intent", func() {
			By("Creating request with empty intent")
			request := ProcessIntentRequest{
				Intent: "",
			}

			requestBody, err := json.Marshal(request)
			Expect(err).NotTo(HaveOccurred())

			By("Making POST request with empty intent")
			resp, err := http.Post(
				testServer.URL+"/process",
				"application/json",
				bytes.NewBuffer(requestBody),
			)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))

			By("Verifying error response")
			body, err := io.ReadAll(resp.Body)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(body)).To(ContainSubstring("Intent is required"))
		})

		It("Should reject non-POST requests", func() {
			By("Making GET request to process endpoint")
			resp, err := http.Get(testServer.URL + "/process")
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusMethodNotAllowed))
		})

		It("Should handle invalid JSON requests", func() {
			By("Making request with invalid JSON")
			invalidJSON := `{"intent": "test", "invalid": json}`

			resp, err := http.Post(
				testServer.URL+"/process",
				"application/json",
				strings.NewReader(invalidJSON),
			)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusBadRequest))
		})

		It("Should handle LLM processing failures", func() {
			By("Setting up service with failing LLM client")
			failingClient := &MockLLMClient{
				Response: "",
				Error:    fmt.Errorf("LLM service unavailable"),
			}
			failingService := NewLLMProcessorService(failingClient)
			failingServer := httptest.NewServer(failingService.SetupHandler())
			defer failingServer.Close()

			By("Creating intent request")
			request := ProcessIntentRequest{
				Intent: "Test intent for failure",
			}

			requestBody, err := json.Marshal(request)
			Expect(err).NotTo(HaveOccurred())

			By("Making request to failing service")
			resp, err := http.Post(
				failingServer.URL+"/process",
				"application/json",
				bytes.NewBuffer(requestBody),
			)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusInternalServerError))

			By("Parsing error response")
			var errorResp ProcessIntentResponse
			Expect(json.NewDecoder(resp.Body).Decode(&errorResp)).To(Succeed())

			Expect(errorResp.Status).To(Equal("error"))
			Expect(errorResp.Error).To(ContainSubstring("LLM service unavailable"))
			Expect(errorResp.RequestID).NotTo(BeEmpty())
			Expect(errorResp.ProcessingTime).NotTo(BeEmpty())
		})

		It("Should handle requests with metadata", func() {
			By("Creating request with metadata")
			request := ProcessIntentRequest{
				Intent: "Deploy network function with metadata",
				Metadata: map[string]string{
					"priority":  "high",
					"namespace": "production",
					"source":    "operator-dashboard",
					"user":      "admin",
				},
			}

			requestBody, err := json.Marshal(request)
			Expect(err).NotTo(HaveOccurred())

			By("Making request with metadata")
			resp, err := http.Post(
				testServer.URL+"/process",
				"application/json",
				bytes.NewBuffer(requestBody),
			)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By("Verifying successful processing")
			var processResp ProcessIntentResponse
			Expect(json.NewDecoder(resp.Body).Decode(&processResp)).To(Succeed())

			Expect(processResp.Status).To(Equal("success"))
			Expect(processResp.Result).NotTo(BeEmpty())
		})
	})

	Context("Service Integration with Different LLM Responses", func() {
		It("Should handle complex JSON responses from LLM", func() {
			By("Setting up service with complex response")
			complexResponse := map[string]interface{}{
				"action":     "deploy",
				"component":  "5g-core",
				"replicas":   3,
				"resources": map[string]interface{}{
					"cpu":    "2000m",
					"memory": "4Gi",
				},
				"networking": map[string]interface{}{
					"ports": []int{8080, 9090},
					"ingress": map[string]interface{}{
						"enabled":  true,
						"hostname": "core.example.com",
					},
				},
			}
			complexResponseBytes, _ := json.Marshal(complexResponse)

			complexClient := &MockLLMClient{
				Response: string(complexResponseBytes),
				Error:    nil,
			}
			complexService := NewLLMProcessorService(complexClient)
			complexServer := httptest.NewServer(complexService.SetupHandler())
			defer complexServer.Close()

			By("Making request for complex processing")
			request := ProcessIntentRequest{
				Intent: "Deploy 5G core with high availability and ingress",
			}

			requestBody, err := json.Marshal(request)
			Expect(err).NotTo(HaveOccurred())

			resp, err := http.Post(
				complexServer.URL+"/process",
				"application/json",
				bytes.NewBuffer(requestBody),
			)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By("Verifying complex response handling")
			var processResp ProcessIntentResponse
			Expect(json.NewDecoder(resp.Body).Decode(&processResp)).To(Succeed())

			Expect(processResp.Status).To(Equal("success"))
			Expect(processResp.Result).To(Equal(string(complexResponseBytes)))

			// Verify the result can be parsed back to JSON
			var resultJSON map[string]interface{}
			Expect(json.Unmarshal([]byte(processResp.Result), &resultJSON)).To(Succeed())
			Expect(resultJSON["action"]).To(Equal("deploy"))
			Expect(resultJSON["component"]).To(Equal("5g-core"))
		})

		It("Should handle retry scenarios with LLM client", func() {
			By("Setting up service with retry-capable LLM client")
			retryClient := &MockLLMClient{
				Response:  `{"action": "retry_success", "attempt": 3}`,
				Error:     fmt.Errorf("temporary failure"),
				FailCount: 2, // Fail first 2 attempts, succeed on 3rd
				CallCount: 0,
			}
			retryService := NewLLMProcessorService(retryClient)
			retryServer := httptest.NewServer(retryService.SetupHandler())
			defer retryServer.Close()

			By("Making request that will initially fail")
			request := ProcessIntentRequest{
				Intent: "Test retry behavior",
			}

			requestBody, err := json.Marshal(request)
			Expect(err).NotTo(HaveOccurred())

			// First request should fail
			resp1, err := http.Post(
				retryServer.URL+"/process",
				"application/json",
				bytes.NewBuffer(requestBody),
			)
			Expect(err).NotTo(HaveOccurred())
			defer resp1.Body.Close()

			Expect(resp1.StatusCode).To(Equal(http.StatusInternalServerError))

			// Second request should also fail
			resp2, err := http.Post(
				retryServer.URL+"/process",
				"application/json",
				bytes.NewBuffer(requestBody),
			)
			Expect(err).NotTo(HaveOccurred())
			defer resp2.Body.Close()

			Expect(resp2.StatusCode).To(Equal(http.StatusInternalServerError))

			// Third request should succeed
			retryClient.Error = nil // Clear error for success
			resp3, err := http.Post(
				retryServer.URL+"/process",
				"application/json",
				bytes.NewBuffer(requestBody),
			)
			Expect(err).NotTo(HaveOccurred())
			defer resp3.Body.Close()

			Expect(resp3.StatusCode).To(Equal(http.StatusOK))

			By("Verifying successful retry response")
			var processResp ProcessIntentResponse
			Expect(json.NewDecoder(resp3.Body).Decode(&processResp)).To(Succeed())

			Expect(processResp.Status).To(Equal("success"))
			Expect(processResp.Result).To(ContainSubstring("retry_success"))
		})
	})

	Context("Service Performance and Timing", func() {
		It("Should include processing time in responses", func() {
			By("Making request and measuring timing")
			request := ProcessIntentRequest{
				Intent: "Test processing time measurement",
			}

			requestBody, err := json.Marshal(request)
			Expect(err).NotTo(HaveOccurred())

			startTime := time.Now()
			resp, err := http.Post(
				testServer.URL+"/process",
				"application/json",
				bytes.NewBuffer(requestBody),
			)
			requestDuration := time.Since(startTime)
			Expect(err).NotTo(HaveOccurred())
			defer resp.Body.Close()

			Expect(resp.StatusCode).To(Equal(http.StatusOK))

			By("Verifying processing time is included")
			var processResp ProcessIntentResponse
			Expect(json.NewDecoder(resp.Body).Decode(&processResp)).To(Succeed())

			Expect(processResp.ProcessingTime).NotTo(BeEmpty())

			// Parse the processing time and verify it's reasonable
			processingTime, err := time.ParseDuration(processResp.ProcessingTime)
			Expect(err).NotTo(HaveOccurred())
			Expect(processingTime).To(BeNumerically("<=", requestDuration))
			Expect(processingTime).To(BeNumerically(">", 0))
		})

		It("Should generate unique request IDs", func() {
			By("Making multiple requests")
			request := ProcessIntentRequest{
				Intent: "Test unique request IDs",
			}

			requestBody, err := json.Marshal(request)
			Expect(err).NotTo(HaveOccurred())

			requestIDs := make(map[string]bool)

			for i := 0; i < 5; i++ {
				resp, err := http.Post(
					testServer.URL+"/process",
					"application/json",
					bytes.NewBuffer(requestBody),
				)
				Expect(err).NotTo(HaveOccurred())

				var processResp ProcessIntentResponse
				Expect(json.NewDecoder(resp.Body).Decode(&processResp)).To(Succeed())
				resp.Body.Close()

				Expect(processResp.RequestID).NotTo(BeEmpty())
				Expect(requestIDs[processResp.RequestID]).To(BeFalse(), "Request ID should be unique")
				requestIDs[processResp.RequestID] = true
			}

			By("Verifying all request IDs are unique")
			Expect(len(requestIDs)).To(Equal(5))
		})
	})
})