package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLLMIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LLM Integration Suite")
}

var _ = Describe("LLM Client Integration Tests", func() {
	var (
		client     *Client
		mockServer *httptest.Server
		responses  map[string]interface{}
	)

	BeforeEach(func() {
		responses = make(map[string]interface{})

		// Create mock LLM processor server
		mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/process" && r.Method == "POST" {
				handleProcessRequest(w, r, responses)
			} else if r.URL.Path == "/healthz" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "ok"}`))
			} else if r.URL.Path == "/readyz" {
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"status": "ready"}`))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))

		client = NewClient(mockServer.URL)
	})

	AfterEach(func() {
		mockServer.Close()
	})

	Context("When processing network function deployment intents", func() {
		It("should successfully process UPF deployment intent", func() {
			// Setup mock response
			responses["upf"] = map[string]interface{}{
				"type":      "NetworkFunctionDeployment",
				"name":      "upf-core-network",
				"namespace": "5g-core",
				"spec": map[string]interface{}{
					"replicas": float64(3),
					"image":    "registry.5g.local/upf:v2.1.0",
					"resources": map[string]interface{}{
						"requests": map[string]string{"cpu": "2000m", "memory": "4Gi"},
						"limits":   map[string]string{"cpu": "4000m", "memory": "8Gi"},
					},
					"ports": []map[string]interface{}{
						{"containerPort": float64(8805), "protocol": "UDP"},
						{"containerPort": float64(2152), "protocol": "UDP"},
					},
				},
				"o1_config": "<?xml version=\"1.0\"?><config xmlns=\"urn:o-ran:nf:1.0\"><upf><interfaces><n3>192.168.1.0/24</n3></interfaces></upf></config>",
				"a1_policy": map[string]interface{}{
					"policy_type_id": "upf-qos-policy-v1",
					"policy_data": map[string]interface{}{
						"scope": "nsi-core-001",
					},
				},
			}

			ctx := context.Background()
			intent := "Deploy a UPF network function with 3 replicas for high availability"

			result, err := client.ProcessIntent(ctx, intent)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeEmpty())

			var response map[string]interface{}
			err = json.Unmarshal([]byte(result), &response)
			Expect(err).ToNot(HaveOccurred())

			Expect(response["type"]).To(Equal("NetworkFunctionDeployment"))
			Expect(response["name"]).To(Equal("upf-core-network"))
			Expect(response["namespace"]).To(Equal("5g-core"))

			spec := response["spec"].(map[string]interface{})
			Expect(spec["replicas"]).To(Equal(float64(3)))
			Expect(spec["image"]).To(Equal("registry.5g.local/upf:v2.1.0"))
		})

		It("should successfully process Near-RT RIC deployment intent", func() {
			responses["ric"] = map[string]interface{}{
				"type":      "NetworkFunctionDeployment",
				"name":      "near-rt-ric-xapp-platform",
				"namespace": "o-ran",
				"spec": map[string]interface{}{
					"replicas": float64(2),
					"image":    "registry.oran.local/near-rt-ric:v3.0.0",
					"resources": map[string]interface{}{
						"requests": map[string]string{"cpu": "1000m", "memory": "2Gi"},
						"limits":   map[string]string{"cpu": "2000m", "memory": "4Gi"},
					},
				},
			}

			ctx := context.Background()
			intent := "Set up Near-RT RIC with xApp support for intelligent traffic management"

			result, err := client.ProcessIntent(ctx, intent)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeEmpty())

			var response map[string]interface{}
			err = json.Unmarshal([]byte(result), &response)
			Expect(err).ToNot(HaveOccurred())

			Expect(response["type"]).To(Equal("NetworkFunctionDeployment"))
			Expect(response["name"]).To(Equal("near-rt-ric-xapp-platform"))
			Expect(response["namespace"]).To(Equal("o-ran"))
		})
	})

	Context("When processing network function scaling intents", func() {
		It("should successfully process AMF horizontal scaling intent", func() {
			responses["scale"] = map[string]interface{}{
				"type":      "NetworkFunctionScale",
				"name":      "amf-core",
				"namespace": "5g-core",
				"spec": map[string]interface{}{
					"scaling": map[string]interface{}{
						"horizontal": map[string]interface{}{
							"replicas":               float64(5),
							"min_replicas":           float64(3),
							"max_replicas":           float64(10),
							"target_cpu_utilization": float64(70),
						},
					},
				},
			}

			ctx := context.Background()
			intent := "Scale AMF instances to 5 replicas to handle increased signaling load"

			result, err := client.ProcessIntent(ctx, intent)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeEmpty())

			var response map[string]interface{}
			err = json.Unmarshal([]byte(result), &response)
			Expect(err).ToNot(HaveOccurred())

			Expect(response["type"]).To(Equal("NetworkFunctionScale"))
			Expect(response["name"]).To(Equal("amf-core"))

			spec := response["spec"].(map[string]interface{})
			scaling := spec["scaling"].(map[string]interface{})
			horizontal := scaling["horizontal"].(map[string]interface{})
			Expect(horizontal["replicas"]).To(Equal(float64(5)))
		})

		It("should successfully process UPF vertical scaling intent", func() {
			responses["scale"] = map[string]interface{}{
				"type":      "NetworkFunctionScale",
				"name":      "upf-edge",
				"namespace": "5g-core",
				"spec": map[string]interface{}{
					"scaling": map[string]interface{}{
						"vertical": map[string]interface{}{
							"cpu":    "4000m",
							"memory": "8Gi",
						},
					},
				},
			}

			ctx := context.Background()
			intent := "Increase UPF resources to 4 CPU cores and 8GB memory"

			result, err := client.ProcessIntent(ctx, intent)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeEmpty())

			var response map[string]interface{}
			err = json.Unmarshal([]byte(result), &response)
			Expect(err).ToNot(HaveOccurred())

			Expect(response["type"]).To(Equal("NetworkFunctionScale"))
			spec := response["spec"].(map[string]interface{})
			scaling := spec["scaling"].(map[string]interface{})
			vertical := scaling["vertical"].(map[string]interface{})
			Expect(vertical["cpu"]).To(Equal("4000m"))
			Expect(vertical["memory"]).To(Equal("8Gi"))
		})
	})

	Context("When handling errors", func() {
		It("should retry on retriable errors", func() {
			// First request fails, second succeeds
			attemptCount := 0
			mockServer.Close()
			mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				attemptCount++
				if attemptCount == 1 {
					w.WriteHeader(http.StatusInternalServerError)
					w.Write([]byte(`{"error": "internal server error"}`))
					return
				}
				// Second attempt succeeds
				response := map[string]interface{}{
					"type":      "NetworkFunctionDeployment",
					"name":      "test-nf",
					"namespace": "default",
					"spec": map[string]interface{}{
						"replicas": float64(1),
						"image":    "test:latest",
					},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(response)
			}))

			client = NewClient(mockServer.URL)
			ctx := context.Background()
			intent := "Deploy test network function"

			result, err := client.ProcessIntent(ctx, intent)

			Expect(err).ToNot(HaveOccurred())
			Expect(result).ToNot(BeEmpty())
			Expect(attemptCount).To(Equal(2)) // Should have retried once
		})

		It("should fail on non-retriable errors", func() {
			mockServer.Close()
			mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusBadRequest)
				w.Write([]byte(`{"error": "invalid request format"}`))
			}))

			client = NewClient(mockServer.URL)
			ctx := context.Background()
			intent := "Deploy test network function"

			_, err := client.ProcessIntent(ctx, intent)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("invalid request format"))
		})

		It("should validate response format", func() {
			mockServer.Close()
			mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Return invalid response (missing required fields)
				invalidResponse := map[string]interface{}{
					"type": "NetworkFunctionDeployment",
					// Missing name, namespace, spec fields
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(invalidResponse)
			}))

			client = NewClient(mockServer.URL)
			ctx := context.Background()
			intent := "Deploy test network function"

			_, err := client.ProcessIntent(ctx, intent)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("response validation failed"))
		})
	})

	Context("When using timeout", func() {
		It("should timeout on slow responses", func() {
			mockServer.Close()
			mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(2 * time.Second) // Longer than client timeout
				w.WriteHeader(http.StatusOK)
			}))

			// Create client with short timeout
			client = NewClient(mockServer.URL)
			client.httpClient.Timeout = 500 * time.Millisecond

			ctx := context.Background()
			intent := "Deploy test network function"

			_, err := client.ProcessIntent(ctx, intent)

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("deadline exceeded"))
		})
	})

	Context("When testing parameter extraction", func() {
		It("should extract replicas from intent", func() {
			params := client.promptEngine.ExtractParameters("Deploy UPF with 5 replicas")
			Expect(params["replicas"]).To(Equal("5"))
		})

		It("should extract CPU cores from intent", func() {
			params := client.promptEngine.ExtractParameters("Scale to 4 CPU cores")
			Expect(params["cpu"]).To(Equal("4000m"))
		})

		It("should extract memory from intent", func() {
			params := client.promptEngine.ExtractParameters("Allocate 8GB memory")
			Expect(params["memory"]).To(Equal("8Gi"))
		})

		It("should extract network function type", func() {
			params := client.promptEngine.ExtractParameters("Deploy UPF network function")
			Expect(params["network_function"]).To(Equal("upf"))
		})

		It("should extract namespace hints", func() {
			params := client.promptEngine.ExtractParameters("Deploy in 5g-core namespace")
			Expect(params["namespace"]).To(Equal("5g-core"))
		})
	})
})

// Helper function to handle process requests in mock server
func handleProcessRequest(w http.ResponseWriter, r *http.Request, responses map[string]interface{}) {
	var req map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid request body"}`))
		return
	}

	spec, ok := req["spec"].(map[string]interface{})
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "missing spec field"}`))
		return
	}

	intent, ok := spec["intent"].(string)
	if !ok {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "missing intent field"}`))
		return
	}

	// Determine response based on intent content
	var response interface{}
	lowerIntent := fmt.Sprintf("%s", intent)

	// Check for scaling intents first (more specific)
	if containsAny(lowerIntent, []string{"scale", "Scale", "increase", "decrease", "resources", "CPU", "memory"}) {
		response = responses["scale"]
	} else if containsAny(lowerIntent, []string{"upf", "UPF"}) {
		response = responses["upf"]
	} else if containsAny(lowerIntent, []string{"ric", "RIC", "near-rt", "Near-RT"}) {
		response = responses["ric"]
	} else {
		// Default deployment response
		response = map[string]interface{}{
			"type":      "NetworkFunctionDeployment",
			"name":      "test-nf",
			"namespace": "default",
			"spec": map[string]interface{}{
				"replicas": float64(1),
				"image":    "test:latest",
			},
		}
	}

	if response == nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "no mock response configured"}`))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error": "failed to encode response"}`))
	}
}

// Helper function to check if string contains any of the given substrings
func containsAny(str string, substrings []string) bool {
	for _, substr := range substrings {
		// Manual substring check
		for i := 0; i <= len(str)-len(substr); i++ {
			if str[i:i+len(substr)] == substr {
				return true
			}
		}
	}
	return false
}

// Benchmark tests for performance validation
var _ = Describe("LLM Client Performance Tests", func() {
	var (
		client     *Client
		mockServer *httptest.Server
	)

	BeforeEach(func() {
		mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			response := map[string]interface{}{
				"type":      "NetworkFunctionDeployment",
				"name":      "perf-test-nf",
				"namespace": "default",
				"spec": map[string]interface{}{
					"replicas": float64(1),
					"image":    "test:latest",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		}))

		client = NewClient(mockServer.URL)
	})

	AfterEach(func() {
		mockServer.Close()
	})

	It("should process intents within performance targets", func() {
		ctx := context.Background()
		intent := "Deploy UPF network function with 3 replicas"

		start := time.Now()
		_, err := client.ProcessIntent(ctx, intent)
		duration := time.Since(start)

		Expect(err).ToNot(HaveOccurred())
		Expect(duration.Seconds()).To(BeNumerically("<", 2.0), "Processing should complete within 2 seconds")
	})
})
