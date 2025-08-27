//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

var _ = Describe("LLM RAG Integration Tests", func() {
	var (
		namespace      *corev1.Namespace
		testCtx        context.Context
		fakeWeaviate   *FakeWeaviateServer
		llmServer      *httptest.Server
		ragServer      *httptest.Server
		requestTracker *RequestTracker
	)

	BeforeEach(func() {
		namespace = CreateTestNamespace()
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, 5*time.Minute)
		DeferCleanup(cancel)

		// Initialize fake Weaviate server
		fakeWeaviate = NewFakeWeaviateServer()

		// Initialize request tracker
		requestTracker = NewRequestTracker()

		// Setup mock LLM server
		llmServer = setupMockLLMServer(requestTracker)
		DeferCleanup(llmServer.Close)

		// Setup mock RAG server
		ragServer = setupMockRAGServer(fakeWeaviate, requestTracker)
		DeferCleanup(ragServer.Close)
	})

	Describe("RAG Pipeline Integration", func() {
		Context("when processing intents with knowledge retrieval", func() {
			It("should successfully retrieve relevant telecommunications knowledge", func() {
				By("querying fake Weaviate for AMF-related knowledge")
				query := "Deploy AMF with high availability"
				vector := fakeWeaviate.generateMockEmbedding(query)

				results, err := fakeWeaviate.VectorSearch("TelecomDocument", vector, 5, 0.7)
				Expect(err).NotTo(HaveOccurred())
				Expect(results).NotTo(BeNil())
				Expect(results.Objects).To(HaveLen(BeNumerically(">", 0)))

				By("verifying search results contain relevant AMF information")
				foundAMFDoc := false
				for _, obj := range results.Objects {
					title, ok := obj.Properties["title"].(string)
					if ok && strings.Contains(strings.ToLower(title), "amf") {
						foundAMFDoc = true
						Expect(obj.Score).To(BeNumerically(">", 0.7))
						break
					}
				}
				Expect(foundAMFDoc).To(BeTrue(), "Should find AMF-related documentation")
			})

			It("should perform hybrid search combining vector and keyword matching", func() {
				By("executing hybrid search for O-RAN components")
				query := "O-RAN Near-RT RIC deployment"
				vector := fakeWeaviate.generateMockEmbedding(query)

				results, err := fakeWeaviate.HybridSearch("TelecomDocument", query, vector, 10)
				Expect(err).NotTo(HaveOccurred())
				Expect(results.Objects).To(HaveLen(BeNumerically(">", 0)))

				By("verifying hybrid search combines vector and keyword scores")
				for _, obj := range results.Objects {
					Expect(obj.Score).To(BeNumerically(">", 0))
					Expect(obj.Certainty).To(Equal(obj.Score))

					// Check if result contains O-RAN related content
					content, ok := obj.Properties["content"].(string)
					if ok {
						contentLower := strings.ToLower(content)
						hasORANKeywords := strings.Contains(contentLower, "o-ran") ||
							strings.Contains(contentLower, "ric") ||
							strings.Contains(contentLower, "near-rt")
						if hasORANKeywords {
							Expect(obj.Score).To(BeNumerically(">", 0.5))
						}
					}
				}
			})

			It("should retrieve network function deployment knowledge", func() {
				By("searching for network function deployment patterns")
				results, err := fakeWeaviate.VectorSearch("NetworkFunction",
					fakeWeaviate.generateMockEmbedding("deployment requirements scaling"), 5, 0.6)
				Expect(err).NotTo(HaveOccurred())
				Expect(results.Objects).To(HaveLen(BeNumerically(">", 0)))

				By("verifying network function results contain deployment information")
				for _, obj := range results.Objects {
					deploymentReqs, ok := obj.Properties["deploymentRequirements"].(string)
					if ok {
						Expect(deploymentReqs).NotTo(BeEmpty())
						Expect(strings.ToLower(deploymentReqs)).To(ContainSubstring("deployment"))
					}

					scalingPolicy, ok := obj.Properties["scalingPolicy"].(string)
					if ok {
						Expect(scalingPolicy).NotTo(BeEmpty())
					}
				}
			})
		})
	})

	Describe("LLM Processing Integration", func() {
		Context("when processing NetworkIntent with LLM", func() {
			It("should successfully process intent through mock LLM endpoint", func() {
				By("creating NetworkIntent that triggers LLM processing")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "llm-processing-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/llm-endpoint": llmServer.URL,
							"nephoran.com/rag-endpoint": ragServer.URL,
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy a production-ready 5G AMF with auto-scaling, monitoring, and security policies",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityHigh,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentAMF,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for controller to process the intent")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 60*time.Second, 2*time.Second).Should(Or(
					Equal("Processing"),
					Equal("Deploying"),
					Equal("Ready"),
					Equal("Failed"),
				))

				By("verifying LLM endpoint was called")
				Eventually(func() int {
					return requestTracker.GetRequestCount("/llm/process")
				}, 30*time.Second, 1*time.Second).Should(BeNumerically(">", 0))

				By("verifying RAG endpoint was called for knowledge retrieval")
				Eventually(func() int {
					return requestTracker.GetRequestCount("/rag/search")
				}, 30*time.Second, 1*time.Second).Should(BeNumerically(">", 0))

				By("checking processed parameters in intent status")
				if createdIntent.Status.Phase == "Ready" || createdIntent.Status.Phase == "Deploying" {
					// In a real implementation, processed parameters would be stored in status
					// For now, verify that processing was attempted
					Expect(createdIntent.Status.ProcessingStartTime).NotTo(BeNil())
				}
			})

			It("should handle different intent types with appropriate LLM responses", func() {
				testCases := []struct {
					name             string
					intent           string
					intentType       nephoranv1.IntentType
					targetComponent  nephoranv1.TargetComponent
					expectedKeywords []string
				}{
					{
						name:             "SMF deployment",
						intent:           "Deploy Session Management Function with PDU session management",
						intentType:       nephoranv1.IntentTypeDeployment,
						targetComponent:  nephoranv1.TargetComponentSMF,
						expectedKeywords: []string{"smf", "session", "pdu"},
					},
					{
						name:             "UPF scaling",
						intent:           "Scale User Plane Function for increased throughput",
						intentType:       nephoranv1.IntentTypeScaling,
						targetComponent:  nephoranv1.TargetComponentUPF,
						expectedKeywords: []string{"upf", "scale", "throughput"},
					},
					{
						name:             "Near-RT RIC optimization",
						intent:           "Optimize Near-RT RIC performance for ML workloads",
						intentType:       nephoranv1.IntentTypeOptimization,
						targetComponent:  nephoranv1.TargetComponentNearRTRIC,
						expectedKeywords: []string{"ric", "optimize", "ml"},
					},
				}

				for i, tc := range testCases {
					By(fmt.Sprintf("processing %s intent", tc.name))
					intent := &nephoranv1.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("multi-type-intent-%d", i),
							Namespace: namespace.Name,
							Annotations: map[string]string{
								"nephoran.com/llm-endpoint": llmServer.URL,
								"nephoran.com/rag-endpoint": ragServer.URL,
							},
						},
						Spec: nephoranv1.NetworkIntentSpec{
							Intent:     tc.intent,
							IntentType: tc.intentType,
							Priority:   nephoranv1.PriorityMedium,
							TargetComponents: []nephoranv1.TargetComponent{
								tc.targetComponent,
							},
						},
					}

					Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

					// Wait for processing to begin
					createdIntent := &nephoranv1.NetworkIntent{}
					Eventually(func() string {
						err := k8sClient.Get(testCtx, types.NamespacedName{
							Name: intent.Name, Namespace: intent.Namespace,
						}, createdIntent)
						if err != nil {
							return ""
						}
						return createdIntent.Status.Phase
					}, 45*time.Second, 2*time.Second).Should(Not(Equal("Pending")))
				}

				By("verifying all intent types were processed by LLM")
				Eventually(func() int {
					return requestTracker.GetRequestCount("/llm/process")
				}, 60*time.Second, 2*time.Second).Should(BeNumerically(">=", len(testCases)))
			})

			It("should handle LLM processing failures gracefully", func() {
				By("configuring LLM server to return errors")
				requestTracker.SetErrorMode("/llm/process", true)

				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "error-handling-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/llm-endpoint": llmServer.URL,
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy AMF that should trigger LLM error",
						IntentType: nephoranv1.IntentTypeDeployment,
						MaxRetries: &[]int32{2}[0],
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("waiting for controller to handle the error")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 90*time.Second, 3*time.Second).Should(Or(
					Equal("Failed"),
					Equal("Processing"), // May still be retrying
				))

				By("verifying retry attempts were made")
				Eventually(func() int {
					return requestTracker.GetRequestCount("/llm/process")
				}, 30*time.Second, 2*time.Second).Should(BeNumerically(">=", 1))

				By("resetting error mode and verifying recovery")
				requestTracker.SetErrorMode("/llm/process", false)

				// Trigger reconciliation by updating intent
				createdIntent.Annotations["nephoran.com/retry"] = "true"
				Expect(k8sClient.Update(testCtx, createdIntent)).To(Succeed())

				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 60*time.Second, 2*time.Second).Should(Or(
					Equal("Processing"),
					Equal("Deploying"),
					Equal("Ready"),
				))
			})
		})
	})

	Describe("RAG Knowledge Base Integration", func() {
		Context("when managing telecommunications knowledge", func() {
			It("should successfully query network function specifications", func() {
				By("adding custom network function to knowledge base")
				customNF := &WeaviateObject{
					Class: "NetworkFunction",
					Properties: map[string]interface{}{
						"name":                   "Custom-AMF",
						"description":            "Custom Access and Mobility Management Function with enhanced features",
						"type":                   "5GC",
						"interfaces":             []string{"N1", "N2", "N8", "N11", "N12", "N14", "N15"},
						"deploymentRequirements": "Requires persistent storage, high availability setup with 5 replicas minimum",
						"scalingPolicy":          "Auto-scale based on N1/N2 interface load, maximum 20 replicas",
					},
				}

				err := fakeWeaviate.AddObject("NetworkFunction", customNF)
				Expect(err).NotTo(HaveOccurred())

				By("verifying custom network function can be retrieved")
				results, err := fakeWeaviate.VectorSearch("NetworkFunction",
					fakeWeaviate.generateMockEmbedding("custom AMF enhanced features"), 5, 0.5)
				Expect(err).NotTo(HaveOccurred())
				Expect(results.Objects).To(HaveLen(BeNumerically(">", 0)))

				foundCustomNF := false
				for _, obj := range results.Objects {
					if name, ok := obj.Properties["name"].(string); ok && name == "Custom-AMF" {
						foundCustomNF = true
						Expect(obj.Properties["description"]).To(ContainSubstring("enhanced features"))
						break
					}
				}
				Expect(foundCustomNF).To(BeTrue())
			})

			It("should handle knowledge base updates during processing", func() {
				By("creating initial document in knowledge base")
				initialDoc := &WeaviateObject{
					Class: "TelecomDocument",
					Properties: map[string]interface{}{
						"title":        "AMF Deployment Guide v1.0",
						"content":      "Basic AMF deployment with standard configuration",
						"documentType": "guide",
						"category":     "implementation",
						"version":      "1.0",
					},
				}

				err := fakeWeaviate.AddObject("TelecomDocument", initialDoc)
				Expect(err).NotTo(HaveOccurred())

				By("searching for the initial document")
				results1, err := fakeWeaviate.VectorSearch("TelecomDocument",
					fakeWeaviate.generateMockEmbedding("AMF deployment guide"), 5, 0.5)
				Expect(err).NotTo(HaveOccurred())
				initialCount := len(results1.Objects)

				By("adding updated document to knowledge base")
				updatedDoc := &WeaviateObject{
					Class: "TelecomDocument",
					Properties: map[string]interface{}{
						"title":        "AMF Deployment Guide v2.0",
						"content":      "Advanced AMF deployment with high availability, monitoring, and auto-scaling features",
						"documentType": "guide",
						"category":     "implementation",
						"version":      "2.0",
					},
				}

				err = fakeWeaviate.AddObject("TelecomDocument", updatedDoc)
				Expect(err).NotTo(HaveOccurred())

				By("verifying updated knowledge is available")
				results2, err := fakeWeaviate.VectorSearch("TelecomDocument",
					fakeWeaviate.generateMockEmbedding("AMF deployment guide"), 5, 0.5)
				Expect(err).NotTo(HaveOccurred())
				Expect(len(results2.Objects)).To(BeNumerically(">", initialCount))

				foundV2 := false
				for _, obj := range results2.Objects {
					if title, ok := obj.Properties["title"].(string); ok {
						if strings.Contains(title, "v2.0") {
							foundV2 = true
							content := obj.Properties["content"].(string)
							Expect(content).To(ContainSubstring("high availability"))
							break
						}
					}
				}
				Expect(foundV2).To(BeTrue())
			})

			It("should measure search performance and accuracy", func() {
				By("performing multiple search queries and measuring performance")
				queries := []string{
					"5G core network function deployment",
					"O-RAN interface specifications",
					"AMF auto-scaling configuration",
					"SMF PDU session management",
					"UPF performance optimization",
				}

				searchResults := make([]SearchPerformanceMetric, len(queries))

				for i, query := range queries {
					start := time.Now()
					vector := fakeWeaviate.generateMockEmbedding(query)
					results, err := fakeWeaviate.VectorSearch("TelecomDocument", vector, 10, 0.6)
					duration := time.Since(start)

					Expect(err).NotTo(HaveOccurred())

					searchResults[i] = SearchPerformanceMetric{
						Query:       query,
						Duration:    duration,
						ResultCount: len(results.Objects),
						AvgScore:    calculateAverageScore(results.Objects),
					}

					// Verify search performance
					Expect(duration).To(BeNumerically("<", 100*time.Millisecond))
					Expect(len(results.Objects)).To(BeNumerically(">", 0))
				}

				By("verifying overall search performance metrics")
				totalDuration := time.Duration(0)
				totalResults := 0
				for _, metric := range searchResults {
					totalDuration += metric.Duration
					totalResults += metric.ResultCount
					GinkgoWriter.Printf("Query: %s, Duration: %v, Results: %d, Avg Score: %.2f\n",
						metric.Query, metric.Duration, metric.ResultCount, metric.AvgScore)
				}

				avgDuration := totalDuration / time.Duration(len(queries))
				avgResults := float64(totalResults) / float64(len(queries))

				Expect(avgDuration).To(BeNumerically("<", 50*time.Millisecond))
				Expect(avgResults).To(BeNumerically(">", 2.0))
			})
		})
	})
})

// RequestTracker tracks HTTP requests for testing
type RequestTracker struct {
	requestCounts map[string]int
	errorModes    map[string]bool
	mu            sync.RWMutex
}

func NewRequestTracker() *RequestTracker {
	return &RequestTracker{
		requestCounts: make(map[string]int),
		errorModes:    make(map[string]bool),
	}
}

func (rt *RequestTracker) IncrementRequest(path string) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.requestCounts[path]++
}

func (rt *RequestTracker) GetRequestCount(path string) int {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.requestCounts[path]
}

func (rt *RequestTracker) SetErrorMode(path string, enabled bool) {
	rt.mu.Lock()
	defer rt.mu.Unlock()
	rt.errorModes[path] = enabled
}

func (rt *RequestTracker) ShouldReturnError(path string) bool {
	rt.mu.RLock()
	defer rt.mu.RUnlock()
	return rt.errorModes[path]
}

// SearchPerformanceMetric tracks search performance metrics
type SearchPerformanceMetric struct {
	Query       string
	Duration    time.Duration
	ResultCount int
	AvgScore    float32
}

// setupMockLLMServer creates a mock LLM server for testing
func setupMockLLMServer(tracker *RequestTracker) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracker.IncrementRequest(r.URL.Path)

		if tracker.ShouldReturnError(r.URL.Path) {
			http.Error(w, "Simulated LLM error", http.StatusInternalServerError)
			return
		}

		switch r.URL.Path {
		case "/llm/process":
			handleLLMProcessRequest(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
}

// setupMockRAGServer creates a mock RAG server for testing
func setupMockRAGServer(weaviate *FakeWeaviateServer, tracker *RequestTracker) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		tracker.IncrementRequest(r.URL.Path)

		if tracker.ShouldReturnError(r.URL.Path) {
			http.Error(w, "Simulated RAG error", http.StatusInternalServerError)
			return
		}

		switch r.URL.Path {
		case "/rag/search":
			handleRAGSearchRequest(w, r, weaviate)
		default:
			http.NotFound(w, r)
		}
	}))
}

// handleLLMProcessRequest handles LLM processing requests
func handleLLMProcessRequest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Intent   string                 `json:"intent"`
		Metadata map[string]interface{} `json:"metadata"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Generate mock LLM response based on intent content
	response := generateMockLLMResponse(request.Intent)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleRAGSearchRequest handles RAG search requests
func handleRAGSearchRequest(w http.ResponseWriter, r *http.Request, weaviate *FakeWeaviateServer) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var request struct {
		Query     string    `json:"query"`
		Vector    []float32 `json:"vector"`
		ClassName string    `json:"className"`
		Limit     int       `json:"limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Default values
	if request.ClassName == "" {
		request.ClassName = "TelecomDocument"
	}
	if request.Limit == 0 {
		request.Limit = 5
	}
	if len(request.Vector) == 0 {
		request.Vector = weaviate.generateMockEmbedding(request.Query)
	}

	results, err := weaviate.HybridSearch(request.ClassName, request.Query, request.Vector, request.Limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Search failed: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(results)
}

// generateMockLLMResponse creates a realistic LLM response based on intent
func generateMockLLMResponse(intent string) *shared.LLMResponse {
	intentLower := strings.ToLower(intent)

	// Determine intent type and parameters based on content
	var intentType, networkFunction string
	var parameters map[string]interface{}
	var manifests map[string]interface{}

	if strings.Contains(intentLower, "amf") {
		intentType = "5G-Core-AMF"
		networkFunction = "AMF"
		parameters = map[string]interface{}{
			"replicas":          3,
			"scaling":           true,
			"ha_enabled":        true,
			"cpu_request":       "500m",
			"memory_request":    "1Gi",
			"cpu_limit":         "2000m",
			"memory_limit":      "4Gi",
			"monitoring":        true,
			"security_policies": true,
		}
		manifests = generateAMFManifests()
	} else if strings.Contains(intentLower, "smf") {
		intentType = "5G-Core-SMF"
		networkFunction = "SMF"
		parameters = map[string]interface{}{
			"replicas":       2,
			"scaling":        true,
			"pdu_sessions":   5000,
			"cpu_request":    "300m",
			"memory_request": "512Mi",
		}
		manifests = generateSMFManifests()
	} else if strings.Contains(intentLower, "upf") {
		intentType = "5G-Core-UPF"
		networkFunction = "UPF"
		parameters = map[string]interface{}{
			"replicas":        2,
			"dpdk_enabled":    true,
			"edge_deployment": true,
			"cpu_request":     "1000m",
			"memory_request":  "2Gi",
		}
		manifests = generateUPFManifests()
	} else if strings.Contains(intentLower, "ric") {
		intentType = "O-RAN-Near-RT-RIC"
		networkFunction = "Near-RT-RIC"
		parameters = map[string]interface{}{
			"replicas":       1,
			"xapp_runtime":   true,
			"ml_inference":   true,
			"cpu_request":    "2000m",
			"memory_request": "4Gi",
		}
		manifests = generateRICManifests()
	} else {
		// Default to generic 5G core function
		intentType = "5G-Core-Generic"
		networkFunction = "Generic"
		parameters = map[string]interface{}{
			"replicas":       2,
			"scaling":        false,
			"cpu_request":    "200m",
			"memory_request": "256Mi",
		}
		manifests = generateGenericManifests()
	}

	return &shared.LLMResponse{
		IntentType:      intentType,
		NetworkFunction: networkFunction,
		Confidence:      0.92,
		Parameters:      parameters,
		Manifests:       manifests,
		ProcessingTime:  1200, // 1.2 seconds
		TokensUsed:      180,
		Model:           "gpt-4o-mini",
	}
}

// generateAMFManifests generates AMF deployment manifests
func generateAMFManifests() map[string]interface{} {
	return map[string]interface{}{
		"deployment": map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": "amf-deployment",
				"labels": map[string]interface{}{
					"app":       "amf",
					"component": "5g-core",
				},
			},
			"spec": map[string]interface{}{
				"replicas": 3,
				"selector": map[string]interface{}{
					"matchLabels": map[string]interface{}{
						"app": "amf",
					},
				},
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app": "amf",
						},
					},
					"spec": map[string]interface{}{
						"containers": []interface{}{
							map[string]interface{}{
								"name":  "amf",
								"image": "amf:latest",
								"resources": map[string]interface{}{
									"requests": map[string]interface{}{
										"cpu":    "500m",
										"memory": "1Gi",
									},
									"limits": map[string]interface{}{
										"cpu":    "2000m",
										"memory": "4Gi",
									},
								},
							},
						},
					},
				},
			},
		},
		"service": map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Service",
			"metadata": map[string]interface{}{
				"name": "amf-service",
			},
			"spec": map[string]interface{}{
				"selector": map[string]interface{}{
					"app": "amf",
				},
				"ports": []interface{}{
					map[string]interface{}{
						"port":       80,
						"targetPort": 8080,
					},
				},
			},
		},
	}
}

// generateSMFManifests generates SMF deployment manifests
func generateSMFManifests() map[string]interface{} {
	return map[string]interface{}{
		"deployment": map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": "smf-deployment",
			},
			"spec": map[string]interface{}{
				"replicas": 2,
			},
		},
	}
}

// generateUPFManifests generates UPF deployment manifests
func generateUPFManifests() map[string]interface{} {
	return map[string]interface{}{
		"deployment": map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": "upf-deployment",
			},
			"spec": map[string]interface{}{
				"replicas": 2,
			},
		},
	}
}

// generateRICManifests generates RIC deployment manifests
func generateRICManifests() map[string]interface{} {
	return map[string]interface{}{
		"deployment": map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": "near-rt-ric-deployment",
			},
			"spec": map[string]interface{}{
				"replicas": 1,
			},
		},
	}
}

// generateGenericManifests generates generic deployment manifests
func generateGenericManifests() map[string]interface{} {
	return map[string]interface{}{
		"deployment": map[string]interface{}{
			"apiVersion": "apps/v1",
			"kind":       "Deployment",
			"metadata": map[string]interface{}{
				"name": "generic-deployment",
			},
			"spec": map[string]interface{}{
				"replicas": 2,
			},
		},
	}
}

// calculateAverageScore calculates average score from search results
func calculateAverageScore(objects []SearchObject) float32 {
	if len(objects) == 0 {
		return 0
	}

	var total float32
	for _, obj := range objects {
		total += obj.Score
	}
	return total / float32(len(objects))
}
