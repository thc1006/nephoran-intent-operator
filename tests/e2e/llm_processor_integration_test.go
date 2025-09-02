//go:build integration

package e2e

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nephoran "github.com/thc1006/nephoran-intent-operator/api/v1"
)

var _ = Describe("LLM Processor Integration Tests", func() {
	var (
		ctx             context.Context
		namespace       string
		llmProcessorURL string
		httpClient      *http.Client
	)

	BeforeEach(func() {
		ctx = context.Background()
		namespace = "default"
		// In real environment, this would be the actual LLM processor service URL
		llmProcessorURL = "http://localhost:8080" // Default LLM processor port
		httpClient = &http.Client{
			Timeout: 30 * time.Second,
		}
	})

	Context("LLM Processor Service Health", func() {
		It("should respond to health check requests", func() {
			Skip("Skipping until LLM processor service is running in test environment")

			By("Making health check request")
			resp, err := httpClient.Get(llmProcessorURL + "/health")
			if err != nil {
				// Service might not be running in test environment
				Skip(fmt.Sprintf("LLM processor service not available: %v", err))
			}
			defer resp.Body.Close()

			Expect(resp.StatusCode).Should(Equal(http.StatusOK))

			body, err := io.ReadAll(resp.Body)
			Expect(err).ShouldNot(HaveOccurred())

			var healthResponse map[string]interface{}
			err = json.Unmarshal(body, &healthResponse)
			Expect(err).ShouldNot(HaveOccurred())
			Expect(healthResponse["status"]).Should(Equal("healthy"))
		})

		It("should respond to metrics endpoint", func() {
			Skip("Skipping until LLM processor service is running in test environment")

			By("Making metrics request")
			resp, err := httpClient.Get(llmProcessorURL + "/metrics")
			if err != nil {
				Skip(fmt.Sprintf("LLM processor service not available: %v", err))
			}
			defer resp.Body.Close()

			Expect(resp.StatusCode).Should(Equal(http.StatusOK))
		})
	})

	Context("LLM Processing Requests", func() {
		It("should process NetworkIntent and return structured output", func() {
			Skip("Skipping until LLM processor service is running in test environment")

			// Create a NetworkIntent first
			intentName := "llm-processing-test-intent"
			networkIntent := &nephoran.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      intentName,
					Namespace: namespace,
				},
				Spec: nephoran.NetworkIntentSpec{
					Intent:     "Scale up UPF instances to handle 10x increased traffic during peak hours",
					IntentType: nephoran.IntentTypeScaling,
					Priority:   nephoran.NetworkPriorityHigh,
					TargetComponents: []nephoran.NetworkTargetComponent{
						nephoran.NetworkTargetComponentUPF,
					},
				},
			}

			By("Creating the NetworkIntent")
			Expect(k8sClient.Create(ctx, networkIntent)).Should(Succeed())

			lookupKey := types.NamespacedName{Name: intentName, Namespace: namespace}
			createdIntent := &nephoran.NetworkIntent{}

			Eventually(func() bool {
				err := k8sClient.Get(ctx, lookupKey, createdIntent)
				return err == nil
			}, 10*time.Second, 1*time.Second).Should(BeTrue())

			By("Sending processing request to LLM processor")
			requestPayload := json.RawMessage("{}")

			jsonPayload, err := json.Marshal(requestPayload)
			Expect(err).ShouldNot(HaveOccurred())

			resp, err := httpClient.Post(
				llmProcessorURL+"/api/v1/process-intent",
				"application/json",
				bytes.NewBuffer(jsonPayload),
			)
			if err != nil {
				Skip(fmt.Sprintf("LLM processor service not available: %v", err))
			}
			defer resp.Body.Close()

			Expect(resp.StatusCode).Should(Equal(http.StatusOK))

			body, err := io.ReadAll(resp.Body)
			Expect(err).ShouldNot(HaveOccurred())

			var response map[string]interface{}
			err = json.Unmarshal(body, &response)
			Expect(err).ShouldNot(HaveOccurred())

			By("Verifying LLM response structure")
			Expect(response["processed_intent"]).ShouldNot(BeEmpty())
			Expect(response["confidence_score"]).Should(BeNumerically(">=", 0))
			Expect(response["recommendations"]).Should(BeAssignableToTypeOf([]interface{}{}))

			By("Cleaning up")
			Expect(k8sClient.Delete(ctx, createdIntent)).Should(Succeed())
		})

		It("should handle streaming responses for complex intents", func() {
			Skip("Skipping until LLM processor service is running in test environment")

			By("Creating complex intent for streaming test")
			complexIntent := json.RawMessage("{}"),
				"priority":          1,
				"streaming":         true,
			}

			jsonPayload, err := json.Marshal(complexIntent)
			Expect(err).ShouldNot(HaveOccurred())

			req, err := http.NewRequest("POST", llmProcessorURL+"/api/v1/process-intent", bytes.NewBuffer(jsonPayload))
			Expect(err).ShouldNot(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "text/event-stream")

			resp, err := httpClient.Do(req)
			if err != nil {
				Skip(fmt.Sprintf("LLM processor service not available: %v", err))
			}
			defer resp.Body.Close()

			Expect(resp.StatusCode).Should(Equal(http.StatusOK))
			Expect(resp.Header.Get("Content-Type")).Should(ContainSubstring("text/event-stream"))

			By("Reading streaming response")
			buf := make([]byte, 1024)
			n, err := resp.Body.Read(buf)
			if err != nil && err != io.EOF {
				Fail(fmt.Sprintf("Failed to read streaming response: %v", err))
			}
			Expect(n).Should(BeNumerically(">", 0))
		})

		It("should validate input and return appropriate errors", func() {
			Skip("Skipping until LLM processor service is running in test environment")

			testCases := []struct {
				name           string
				payload        map[string]interface{}
				expectedStatus int
			}{
				{
					name: "empty intent",
					payload: json.RawMessage("{}"),
					},
					expectedStatus: http.StatusBadRequest,
				},
				{
					name: "invalid intent type",
					payload: json.RawMessage("{}"),
					},
					expectedStatus: http.StatusBadRequest,
				},
				{
					name: "missing required fields",
					payload: json.RawMessage("{}"),
					expectedStatus: http.StatusBadRequest,
				},
			}

			for _, tc := range testCases {
				By(fmt.Sprintf("Testing %s", tc.name))

				jsonPayload, err := json.Marshal(tc.payload)
				Expect(err).ShouldNot(HaveOccurred())

				resp, err := httpClient.Post(
					llmProcessorURL+"/api/v1/process-intent",
					"application/json",
					bytes.NewBuffer(jsonPayload),
				)
				if err != nil {
					Skip(fmt.Sprintf("LLM processor service not available: %v", err))
				}
				defer resp.Body.Close()

				Expect(resp.StatusCode).Should(Equal(tc.expectedStatus))
			}
		})

		It("should handle timeout scenarios gracefully", func() {
			Skip("Skipping until LLM processor service is running in test environment")

			By("Creating a very complex intent that might timeout")
			timeoutIntent := json.RawMessage("{}"),
				"priority":          1,
				"timeout":           1, // 1 second timeout to force timeout
			}

			jsonPayload, err := json.Marshal(timeoutIntent)
			Expect(err).ShouldNot(HaveOccurred())

			// Create client with very short timeout
			shortTimeoutClient := &http.Client{
				Timeout: 2 * time.Second,
			}

			resp, err := shortTimeoutClient.Post(
				llmProcessorURL+"/api/v1/process-intent",
				"application/json",
				bytes.NewBuffer(jsonPayload),
			)
			if err != nil {
				if urlErr, ok := err.(*url.Error); ok && urlErr.Timeout() {
					By("Request timed out as expected")
					return
				}
				Skip(fmt.Sprintf("LLM processor service not available: %v", err))
			}
			defer resp.Body.Close()

			// If no timeout, service should return 408 or 504
			Expect(resp.StatusCode).Should(BeElementOf([]int{http.StatusRequestTimeout, http.StatusGatewayTimeout}))
		})
	})

	Context("Circuit Breaker Integration", func() {
		It("should activate circuit breaker under high error rate", func() {
			Skip("Skipping until LLM processor service is running in test environment")

			By("Sending multiple invalid requests to trigger circuit breaker")
			invalidPayload := json.RawMessage("{}")
			jsonPayload, err := json.Marshal(invalidPayload)
			Expect(err).ShouldNot(HaveOccurred())

			// Send multiple invalid requests
			errorCount := 0
			for i := 0; i < 10; i++ {
				resp, err := httpClient.Post(
					llmProcessorURL+"/api/v1/process-intent",
					"application/json",
					bytes.NewBuffer(jsonPayload),
				)
				if err != nil {
					Skip(fmt.Sprintf("LLM processor service not available: %v", err))
				}
				resp.Body.Close()

				if resp.StatusCode >= 400 {
					errorCount++
				}

				time.Sleep(100 * time.Millisecond)
			}

			By("Verifying errors occurred")
			Expect(errorCount).Should(BeNumerically(">=", 5))

			By("Checking if circuit breaker is activated via health endpoint")
			resp, err := httpClient.Get(llmProcessorURL + "/health")
			if err != nil {
				Skip(fmt.Sprintf("Health endpoint not available: %v", err))
			}
			defer resp.Body.Close()

			body, err := io.ReadAll(resp.Body)
			Expect(err).ShouldNot(HaveOccurred())

			var healthResponse map[string]interface{}
			err = json.Unmarshal(body, &healthResponse)
			Expect(err).ShouldNot(HaveOccurred())

			// Circuit breaker status might be included in health response
			if cbStatus, exists := healthResponse["circuit_breaker"]; exists {
				Expect(cbStatus).Should(ContainSubstring("open"))
			}
		})
	})

	Context("Rate Limiting", func() {
		It("should enforce rate limits under high load", func() {
			Skip("Skipping until LLM processor service is running in test environment")

			By("Sending rapid requests to test rate limiting")
			validPayload := json.RawMessage("{}"),
				"priority":          1,
			}
			jsonPayload, err := json.Marshal(validPayload)
			Expect(err).ShouldNot(HaveOccurred())

			rateLimitedCount := 0
			totalRequests := 20

			for i := 0; i < totalRequests; i++ {
				resp, err := httpClient.Post(
					llmProcessorURL+"/api/v1/process-intent",
					"application/json",
					bytes.NewBuffer(jsonPayload),
				)
				if err != nil {
					Skip(fmt.Sprintf("LLM processor service not available: %v", err))
				}
				resp.Body.Close()

				if resp.StatusCode == http.StatusTooManyRequests {
					rateLimitedCount++
				}
			}

			By("Verifying rate limiting was activated")
			Expect(rateLimitedCount).Should(BeNumerically(">", 0))
		})
	})

	Context("Authentication and Authorization", func() {
		It("should require valid authentication tokens", func() {
			Skip("Skipping until LLM processor service is running with auth enabled")

			By("Making request without authentication token")
			payload := json.RawMessage("{}"),
			}
			jsonPayload, err := json.Marshal(payload)
			Expect(err).ShouldNot(HaveOccurred())

			resp, err := httpClient.Post(
				llmProcessorURL+"/api/v1/process-intent",
				"application/json",
				bytes.NewBuffer(jsonPayload),
			)
			if err != nil {
				Skip(fmt.Sprintf("LLM processor service not available: %v", err))
			}
			defer resp.Body.Close()

			// Should return 401 Unauthorized if auth is enabled
			Expect(resp.StatusCode).Should(Equal(http.StatusUnauthorized))
		})

		It("should accept valid authentication tokens", func() {
			Skip("Skipping until LLM processor service is running with auth enabled")

			By("Making request with valid authentication token")
			payload := json.RawMessage("{}"),
			}
			jsonPayload, err := json.Marshal(payload)
			Expect(err).ShouldNot(HaveOccurred())

			req, err := http.NewRequest("POST", llmProcessorURL+"/api/v1/process-intent", bytes.NewBuffer(jsonPayload))
			Expect(err).ShouldNot(HaveOccurred())
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test-valid-token")

			resp, err := httpClient.Do(req)
			if err != nil {
				Skip(fmt.Sprintf("LLM processor service not available: %v", err))
			}
			defer resp.Body.Close()

			// Should return 200 OK with valid token
			Expect(resp.StatusCode).Should(Equal(http.StatusOK))
		})
	})
})
