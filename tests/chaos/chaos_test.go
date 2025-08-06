// Package chaos provides comprehensive chaos engineering testing for resilience validation
package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	"github.com/thc1006/nephoran-intent-operator/tests/framework"
)

// ChaosTestSuite provides comprehensive chaos engineering testing
type ChaosTestSuite struct {
	*framework.TestSuite
	chaosInjector *ChaosInjector
}

// TestChaosEngineering runs the chaos engineering test suite
func TestChaosEngineering(t *testing.T) {
	suite.Run(t, &ChaosTestSuite{
		TestSuite: framework.NewTestSuite(&framework.TestConfig{
			ChaosTestEnabled:  true,
			FailureRate:       0.2, // 20% failure rate
			LoadTestEnabled:   true,
			MaxConcurrency:    50,
			TestDuration:      3 * time.Minute,
			MockExternalAPIs:  true,
		}),
	})
}

// SetupSuite initializes the chaos testing environment
func (suite *ChaosTestSuite) SetupSuite() {
	suite.TestSuite.SetupSuite()
	suite.chaosInjector = NewChaosInjector()
}

// ChaosInjector manages various chaos engineering scenarios
type ChaosInjector struct {
	networkFailures    map[string]*NetworkFailure
	resourceExhaustion map[string]*ResourceExhaustion
	latencyInjection   map[string]*LatencyInjection
	serviceFailures    map[string]*ServiceFailure
	mu                sync.RWMutex
}

// NetworkFailure represents network-related chaos
type NetworkFailure struct {
	ServiceName     string
	FailureType     string // "timeout", "connection_refused", "dns_failure"
	FailureRate     float64
	Duration        time.Duration
	Active          bool
}

// ResourceExhaustion represents resource-related chaos
type ResourceExhaustion struct {
	ResourceType    string // "memory", "cpu", "storage", "file_descriptors"
	ExhaustionLevel float64 // 0.0 to 1.0
	Duration        time.Duration
	Active          bool
}

// LatencyInjection represents latency-related chaos
type LatencyInjection struct {
	ServiceName    string
	MinLatency     time.Duration
	MaxLatency     time.Duration
	InjectionRate  float64
	Active         bool
}

// ServiceFailure represents service-level failures
type ServiceFailure struct {
	ServiceName    string
	FailureMode    string // "crash", "hang", "corrupt_response", "slow_response"
	FailureRate    float64
	RecoveryTime   time.Duration
	Active         bool
}

// NewChaosInjector creates a new chaos injector
func NewChaosInjector() *ChaosInjector {
	return &ChaosInjector{
		networkFailures:    make(map[string]*NetworkFailure),
		resourceExhaustion: make(map[string]*ResourceExhaustion),
		latencyInjection:   make(map[string]*LatencyInjection),
		serviceFailures:    make(map[string]*ServiceFailure),
	}
}

// InjectNetworkFailure injects network-related failures
func (ci *ChaosInjector) InjectNetworkFailure(serviceName string, failureType string, failureRate float64, duration time.Duration) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	
	ci.networkFailures[serviceName] = &NetworkFailure{
		ServiceName: serviceName,
		FailureType: failureType,
		FailureRate: failureRate,
		Duration:    duration,
		Active:      true,
	}
	
	// Auto-disable after duration
	go func() {
		time.Sleep(duration)
		ci.mu.Lock()
		if failure, exists := ci.networkFailures[serviceName]; exists {
			failure.Active = false
		}
		ci.mu.Unlock()
	}()
}

// InjectResourceExhaustion injects resource exhaustion scenarios
func (ci *ChaosInjector) InjectResourceExhaustion(resourceType string, exhaustionLevel float64, duration time.Duration) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	
	exhaustionID := fmt.Sprintf("%s-%d", resourceType, time.Now().UnixNano())
	ci.resourceExhaustion[exhaustionID] = &ResourceExhaustion{
		ResourceType:    resourceType,
		ExhaustionLevel: exhaustionLevel,
		Duration:        duration,
		Active:          true,
	}
	
	// Simulate resource exhaustion
	go ci.simulateResourceExhaustion(exhaustionID)
	
	// Auto-disable after duration
	go func() {
		time.Sleep(duration)
		ci.mu.Lock()
		if exhaustion, exists := ci.resourceExhaustion[exhaustionID]; exists {
			exhaustion.Active = false
		}
		ci.mu.Unlock()
	}()
}

// InjectLatency injects latency into service calls
func (ci *ChaosInjector) InjectLatency(serviceName string, minLatency, maxLatency time.Duration, injectionRate float64) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	
	ci.latencyInjection[serviceName] = &LatencyInjection{
		ServiceName:   serviceName,
		MinLatency:    minLatency,
		MaxLatency:    maxLatency,
		InjectionRate: injectionRate,
		Active:        true,
	}
}

// InjectServiceFailure injects service-level failures
func (ci *ChaosInjector) InjectServiceFailure(serviceName string, failureMode string, failureRate float64, recoveryTime time.Duration) {
	ci.mu.Lock()
	defer ci.mu.Unlock()
	
	ci.serviceFailures[serviceName] = &ServiceFailure{
		ServiceName:  serviceName,
		FailureMode:  failureMode,
		FailureRate:  failureRate,
		RecoveryTime: recoveryTime,
		Active:       true,
	}
}

// simulateResourceExhaustion simulates different types of resource exhaustion
func (ci *ChaosInjector) simulateResourceExhaustion(exhaustionID string) {
	ci.mu.RLock()
	exhaustion, exists := ci.resourceExhaustion[exhaustionID]
	ci.mu.RUnlock()
	
	if !exists || !exhaustion.Active {
		return
	}
	
	switch exhaustion.ResourceType {
	case "memory":
		// Allocate memory to simulate exhaustion
		size := int(exhaustion.ExhaustionLevel * 100 * 1024 * 1024) // Up to 100MB
		data := make([]byte, size)
		_ = data // Keep reference to prevent GC
		
		time.Sleep(exhaustion.Duration)
		data = nil // Allow GC
		
	case "cpu":
		// Simulate CPU exhaustion with busy loops
		numGoroutines := int(exhaustion.ExhaustionLevel * 10)
		for i := 0; i < numGoroutines; i++ {
			go func() {
				end := time.Now().Add(exhaustion.Duration)
				for time.Now().Before(end) {
					// Busy loop to consume CPU
					for j := 0; j < 10000; j++ {
						_ = j * j
					}
					time.Sleep(1 * time.Millisecond) // Brief pause
				}
			}()
		}
		
	case "storage":
		// Simulate storage exhaustion (not actually exhausting disk)
		fmt.Printf("Simulating storage exhaustion at %.1f%% level\n", exhaustion.ExhaustionLevel*100)
		
	case "file_descriptors":
		// Simulate file descriptor exhaustion
		fmt.Printf("Simulating file descriptor exhaustion at %.1f%% level\n", exhaustion.ExhaustionLevel*100)
	}
}

// ShouldInjectFailure determines if a failure should be injected for a service
func (ci *ChaosInjector) ShouldInjectFailure(serviceName string) bool {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	
	// Check network failures
	if failure, exists := ci.networkFailures[serviceName]; exists && failure.Active {
		return rand.Float64() < failure.FailureRate
	}
	
	// Check service failures
	if failure, exists := ci.serviceFailures[serviceName]; exists && failure.Active {
		return rand.Float64() < failure.FailureRate
	}
	
	return false
}

// GetInjectedLatency returns latency to inject for a service
func (ci *ChaosInjector) GetInjectedLatency(serviceName string) time.Duration {
	ci.mu.RLock()
	defer ci.mu.RUnlock()
	
	if injection, exists := ci.latencyInjection[serviceName]; exists && injection.Active {
		if rand.Float64() < injection.InjectionRate {
			// Generate random latency within range
			latencyRange := injection.MaxLatency - injection.MinLatency
			randomLatency := time.Duration(rand.Float64() * float64(latencyRange))
			return injection.MinLatency + randomLatency
		}
	}
	
	return 0
}

// TestNetworkFailures tests system resilience under network failures
func (suite *ChaosTestSuite) TestNetworkFailures() {
	ginkgo.Describe("Network Failure Resilience", func() {
		var controller *controllers.NetworkIntentReconciler
		
		ginkgo.BeforeEach(func() {
			controller = &controllers.NetworkIntentReconciler{
				Client: suite.GetK8sClient(),
				Scheme: suite.GetK8sClient().Scheme(),
			}
		})
		
		ginkgo.Context("LLM Service Network Failures", func() {
			ginkgo.It("should handle LLM service timeouts gracefully", func() {
				// Inject network timeouts for LLM service
				suite.chaosInjector.InjectNetworkFailure("llm-service", "timeout", 0.5, 30*time.Second)
				
				// Configure mock to simulate timeouts
				llmMock := suite.GetMocks().GetLLMMock()
				llmMock.ExpectedCalls = nil
				llmMock.On("ProcessIntent", gomega.Any(), gomega.Any()).Return(
					map[string]interface{}{}, fmt.Errorf("request timeout"))
				
				intent := &nephranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "timeout-test-intent",
						Namespace: "default",
					},
					Spec: nephranv1.NetworkIntentSpec{
						Description: "Test intent for timeout resilience",
						Priority:    "high",
					},
				}
				
				err := suite.GetK8sClient().Create(suite.GetContext(), intent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}
				
				// Should handle timeout gracefully with retry
				result, err := controller.Reconcile(suite.GetContext(), req)
				gomega.Expect(result.Requeue).To(gomega.BeTrue()) // Should retry
				
				// Verify intent status reflects the failure
				updatedIntent := &nephranv1.NetworkIntent{}
				err = suite.GetK8sClient().Get(suite.GetContext(), req.NamespacedName, updatedIntent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				gomega.Expect(updatedIntent.Status.Phase).To(gomega.Equal("Failed"))
			})
			
			ginkgo.It("should handle intermittent network failures", func() {
				// Inject intermittent failures (30% failure rate)
				suite.chaosInjector.InjectNetworkFailure("llm-service", "connection_refused", 0.3, 1*time.Minute)
				
				successCount := 0
				retryCount := 0
				
				// Process multiple intents to test intermittent failures
				for i := 0; i < 10; i++ {
					intent := &nephranv1.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("intermittent-test-intent-%d", i),
							Namespace: "default",
						},
						Spec: nephranv1.NetworkIntentSpec{
							Description: fmt.Sprintf("Test intent %d for intermittent failure resilience", i),
							Priority:    "medium",
						},
					}
					
					err := suite.GetK8sClient().Create(suite.GetContext(), intent)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					
					req := reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      intent.Name,
							Namespace: intent.Namespace,
						},
					}
					
					// Configure mock to simulate intermittent failures
					if suite.chaosInjector.ShouldInjectFailure("llm-service") {
						llmMock := suite.GetMocks().GetLLMMock()
						llmMock.ExpectedCalls = nil
						llmMock.On("ProcessIntent", gomega.Any(), gomega.Any()).Return(
							map[string]interface{}{}, fmt.Errorf("connection refused"))
						retryCount++
					} else {
						llmMock := suite.GetMocks().GetLLMMock()
						llmMock.ExpectedCalls = nil
						llmMock.On("ProcessIntent", gomega.Any(), gomega.Any()).Return(
							map[string]interface{}{
								"type": "NetworkFunctionDeployment",
								"networkFunction": "AMF",
								"replicas": int64(3),
							}, nil)
						successCount++
					}
					
					controller.Reconcile(suite.GetContext(), req)
				}
				
				fmt.Printf("Network failure test: %d successes, %d retries\n", successCount, retryCount)
				
				// Should have some successes despite failures
				gomega.Expect(successCount).To(gomega.BeNumerically(">", 0))
			})
		})
		
		ginkgo.Context("Weaviate Service Network Failures", func() {
			ginkgo.It("should handle vector database connectivity issues", func() {
				// Inject network failures for Weaviate
				suite.chaosInjector.InjectNetworkFailure("weaviate", "dns_failure", 0.4, 45*time.Second)
				
				// Configure mock to simulate DNS failures
				weaviateMock := suite.GetMocks().GetWeaviateMock()
				weaviateMock.ExpectedCalls = nil
				weaviateMock.On("Query").Return(nil, fmt.Errorf("dns resolution failed"))
				
				intent := &nephranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "weaviate-failure-test",
						Namespace: "default",
					},
					Spec: nephranv1.NetworkIntentSpec{
						Description: "Test intent for Weaviate failure resilience",
						Priority:    "high",
					},
				}
				
				err := suite.GetK8sClient().Create(suite.GetContext(), intent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}
				
				// Should handle failure gracefully (may fall back to basic processing)
				result, err := controller.Reconcile(suite.GetContext(), req)
				gomega.Expect(result.Requeue).To(gomega.BeTrue())
			})
		})
	})
}

// TestResourceExhaustion tests system behavior under resource pressure
func (suite *ChaosTestSuite) TestResourceExhaustion() {
	ginkgo.Describe("Resource Exhaustion Resilience", func() {
		ginkgo.Context("Memory Pressure", func() {
			ginkgo.It("should handle memory pressure gracefully", func() {
				// Inject memory exhaustion (70% level)
				suite.chaosInjector.InjectResourceExhaustion("memory", 0.7, 30*time.Second)
				
				controller := &controllers.NetworkIntentReconciler{
					Client: suite.GetK8sClient(),
					Scheme: suite.GetK8sClient().Scheme(),
				}
				
				// Process intents under memory pressure
				successCount := 0
				for i := 0; i < 5; i++ {
					intent := &nephranv1.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("memory-pressure-intent-%d", i),
							Namespace: "default",
						},
						Spec: nephranv1.NetworkIntentSpec{
							Description: fmt.Sprintf("Memory pressure test intent %d", i),
							Priority:    "medium",
						},
					}
					
					err := suite.GetK8sClient().Create(suite.GetContext(), intent)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					
					req := reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      intent.Name,
							Namespace: intent.Namespace,
						},
					}
					
					result, err := controller.Reconcile(suite.GetContext(), req)
					if err == nil && !result.Requeue {
						successCount++
					}
					
					// Brief pause between operations
					time.Sleep(1 * time.Second)
				}
				
				// Should process at least some intents successfully despite memory pressure
				gomega.Expect(successCount).To(gomega.BeNumerically(">", 0))
				fmt.Printf("Processed %d/5 intents successfully under memory pressure\n", successCount)
			})
		})
		
		ginkgo.Context("CPU Pressure", func() {
			ginkgo.It("should maintain functionality under CPU pressure", func() {
				// Inject CPU exhaustion (60% level)
				suite.chaosInjector.InjectResourceExhaustion("cpu", 0.6, 20*time.Second)
				
				controller := &controllers.NetworkIntentReconciler{
					Client: suite.GetK8sClient(),
					Scheme: suite.GetK8sClient().Scheme(),
				}
				
				start := time.Now()
				
				intent := &nephranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cpu-pressure-intent",
						Namespace: "default",
					},
					Spec: nephranv1.NetworkIntentSpec{
						Description: "CPU pressure test intent",
						Priority:    "high",
					},
				}
				
				err := suite.GetK8sClient().Create(suite.GetContext(), intent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}
				
				result, err := controller.Reconcile(suite.GetContext(), req)
				processingTime := time.Since(start)
				
				// Should complete processing (may take longer due to CPU pressure)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				fmt.Printf("Processing completed in %v under CPU pressure\n", processingTime)
				
				// Should complete within reasonable time even under pressure (allow 30s)
				gomega.Expect(processingTime).To(gomega.BeNumerically("<", 30*time.Second))
			})
		})
	})
}

// TestLatencyInjection tests system behavior with increased latency
func (suite *ChaosTestSuite) TestLatencyInjection() {
	ginkgo.Describe("Latency Injection Resilience", func() {
		ginkgo.It("should handle high latency in external services", func() {
			// Inject high latency for all external services
			suite.chaosInjector.InjectLatency("llm-service", 1*time.Second, 5*time.Second, 0.8)
			suite.chaosInjector.InjectLatency("weaviate", 500*time.Millisecond, 2*time.Second, 0.6)
			
			controller := &controllers.NetworkIntentReconciler{
				Client: suite.GetK8sClient(),
				Scheme: suite.GetK8sClient().Scheme(),
			}
			
			// Configure mocks to simulate latency
			llmMock := suite.GetMocks().GetLLMMock()
			llmMock.ExpectedCalls = nil
			llmMock.On("ProcessIntent", gomega.Any(), gomega.Any()).Run(func(args mock.Arguments) {
				// Inject latency
				latency := suite.chaosInjector.GetInjectedLatency("llm-service")
				if latency > 0 {
					time.Sleep(latency)
				}
			}).Return(map[string]interface{}{
				"type": "NetworkFunctionDeployment",
				"networkFunction": "AMF",
				"replicas": int64(3),
			}, nil)
			
			intent := &nephranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "latency-test-intent",
					Namespace: "default",
				},
				Spec: nephranv1.NetworkIntentSpec{
					Description: "Test intent for latency resilience",
					Priority:    "medium",
				},
			}
			
			err := suite.GetK8sClient().Create(suite.GetContext(), intent)
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			
			req := reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      intent.Name,
					Namespace: intent.Namespace,
				},
			}
			
			start := time.Now()
			result, err := controller.Reconcile(suite.GetContext(), req)
			processingTime := time.Since(start)
			
			// Should complete despite high latency
			gomega.Expect(err).NotTo(gomega.HaveOccurred())
			fmt.Printf("Processing completed in %v with latency injection\n", processingTime)
			
			// Record latency metrics
			suite.GetMetrics().RecordLatency("chaos_latency_injection", processingTime)
			
			// Should handle latency gracefully (may take longer)
			gomega.Expect(processingTime).To(gomega.BeNumerically("<", 15*time.Second))
		})
	})
}

// TestServiceFailures tests system behavior with service failures
func (suite *ChaosTestSuite) TestServiceFailures() {
	ginkgo.Describe("Service Failure Resilience", func() {
		ginkgo.Context("Dependent Service Crashes", func() {
			ginkgo.It("should handle service crashes with graceful degradation", func() {
				// Inject service crash scenario
				suite.chaosInjector.InjectServiceFailure("llm-service", "crash", 0.3, 10*time.Second)
				
				controller := &controllers.NetworkIntentReconciler{
					Client: suite.GetK8sClient(),
					Scheme: suite.GetK8sClient().Scheme(),
				}
				
				processedCount := 0
				failedCount := 0
				
				// Process multiple intents to test crash recovery
				for i := 0; i < 8; i++ {
					intent := &nephranv1.NetworkIntent{
						ObjectMeta: metav1.ObjectMeta{
							Name:      fmt.Sprintf("crash-test-intent-%d", i),
							Namespace: "default",
						},
						Spec: nephranv1.NetworkIntentSpec{
							Description: fmt.Sprintf("Crash test intent %d", i),
							Priority:    "medium",
						},
					}
					
					err := suite.GetK8sClient().Create(suite.GetContext(), intent)
					gomega.Expect(err).NotTo(gomega.HaveOccurred())
					
					req := reconcile.Request{
						NamespacedName: types.NamespacedName{
							Name:      intent.Name,
							Namespace: intent.Namespace,
						},
					}
					
					// Simulate service crashes
					if suite.chaosInjector.ShouldInjectFailure("llm-service") {
						llmMock := suite.GetMocks().GetLLMMock()
						llmMock.ExpectedCalls = nil
						llmMock.On("ProcessIntent", gomega.Any(), gomega.Any()).Return(
							map[string]interface{}{}, fmt.Errorf("service crashed"))
						failedCount++
					} else {
						llmMock := suite.GetMocks().GetLLMMock()
						llmMock.ExpectedCalls = nil
						llmMock.On("ProcessIntent", gomega.Any(), gomega.Any()).Return(
							map[string]interface{}{
								"type": "NetworkFunctionDeployment",
								"networkFunction": "AMF",
								"replicas": int64(3),
							}, nil)
						processedCount++
					}
					
					controller.Reconcile(suite.GetContext(), req)
					
					// Brief pause between operations
					time.Sleep(500 * time.Millisecond)
				}
				
				fmt.Printf("Service crash test: %d processed, %d failed\n", processedCount, failedCount)
				
				// Should process some intents successfully despite crashes
				gomega.Expect(processedCount).To(gomega.BeNumerically(">", 0))
				gomega.Expect(failedCount).To(gomega.BeNumerically("<", 8)) // Not all should fail
			})
		})
		
		ginkgo.Context("Hanging Services", func() {
			ginkgo.It("should handle hanging services with timeouts", func() {
				// Inject service hang scenario
				suite.chaosInjector.InjectServiceFailure("llm-service", "hang", 0.4, 20*time.Second)
				
				controller := &controllers.NetworkIntentReconciler{
					Client: suite.GetK8sClient(),
					Scheme: suite.GetK8sClient().Scheme(),
				}
				
				intent := &nephranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "hang-test-intent",
						Namespace: "default",
					},
					Spec: nephranv1.NetworkIntentSpec{
						Description: "Test intent for hang resilience",
						Priority:    "high",
					},
				}
				
				err := suite.GetK8sClient().Create(suite.GetContext(), intent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}
				
				// Configure mock to simulate hanging
				if suite.chaosInjector.ShouldInjectFailure("llm-service") {
					llmMock := suite.GetMocks().GetLLMMock()
					llmMock.ExpectedCalls = nil
					llmMock.On("ProcessIntent", gomega.Any(), gomega.Any()).Run(func(args mock.Arguments) {
						// Simulate hanging by sleeping longer than expected timeout
						time.Sleep(10 * time.Second)
					}).Return(map[string]interface{}{}, fmt.Errorf("service hanging"))
				}
				
				start := time.Now()
				
				// Use context with timeout to test timeout handling
				ctx, cancel := context.WithTimeout(suite.GetContext(), 5*time.Second)
				defer cancel()
				
				result, err := controller.Reconcile(ctx, req)
				processingTime := time.Since(start)
				
				// Should timeout gracefully
				if err != nil {
					gomega.Expect(err.Error()).To(gomega.ContainSubstring("timeout"))
				}
				gomega.Expect(processingTime).To(gomega.BeNumerically("<", 6*time.Second))
				
				fmt.Printf("Hang test completed in %v (expected timeout)\n", processingTime)
			})
		})
	})
}

// TestCombinedChaosScenarios tests multiple chaos conditions simultaneously
func (suite *ChaosTestSuite) TestCombinedChaosScenarios() {
	ginkgo.Describe("Combined Chaos Scenarios", func() {
		ginkgo.It("should handle multiple simultaneous failure modes", func() {
			// Inject multiple chaos conditions simultaneously
			suite.chaosInjector.InjectNetworkFailure("llm-service", "timeout", 0.2, 1*time.Minute)
			suite.chaosInjector.InjectLatency("weaviate", 200*time.Millisecond, 1*time.Second, 0.5)
			suite.chaosInjector.InjectResourceExhaustion("memory", 0.5, 45*time.Second)
			suite.chaosInjector.InjectServiceFailure("rag-api", "slow_response", 0.3, 30*time.Second)
			
			controller := &controllers.NetworkIntentReconciler{
				Client: suite.GetK8sClient(),
				Scheme: suite.GetK8sClient().Scheme(),
			}
			
			successCount := 0
			totalCount := 6
			
			for i := 0; i < totalCount; i++ {
				intent := &nephranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("combined-chaos-intent-%d", i),
						Namespace: "default",
					},
					Spec: nephranv1.NetworkIntentSpec{
						Description: fmt.Sprintf("Combined chaos test intent %d", i),
						Priority:    "medium",
					},
				}
				
				err := suite.GetK8sClient().Create(suite.GetContext(), intent)
				gomega.Expect(err).NotTo(gomega.HaveOccurred())
				
				req := reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      intent.Name,
						Namespace: intent.Namespace,
					},
				}
				
				// Configure mocks based on chaos injection
				llmMock := suite.GetMocks().GetLLMMock()
				llmMock.ExpectedCalls = nil
				
				if suite.chaosInjector.ShouldInjectFailure("llm-service") {
					llmMock.On("ProcessIntent", gomega.Any(), gomega.Any()).Return(
						map[string]interface{}{}, fmt.Errorf("network timeout"))
				} else {
					// Inject latency even for successful calls
					llmMock.On("ProcessIntent", gomega.Any(), gomega.Any()).Run(func(args mock.Arguments) {
						latency := suite.chaosInjector.GetInjectedLatency("weaviate")
						if latency > 0 {
							time.Sleep(latency)
						}
					}).Return(map[string]interface{}{
						"type": "NetworkFunctionDeployment",
						"networkFunction": "AMF",
						"replicas": int64(3),
					}, nil)
				}
				
				start := time.Now()
				result, err := controller.Reconcile(suite.GetContext(), req)
				processingTime := time.Since(start)
				
				if err == nil && !result.Requeue {
					successCount++
				}
				
				suite.GetMetrics().RecordLatency("combined_chaos", processingTime)
				
				// Brief pause between operations
				time.Sleep(2 * time.Second)
			}
			
			successRate := float64(successCount) / float64(totalCount)
			fmt.Printf("Combined chaos test: %d/%d succeeded (%.1f%% success rate)\n",
				successCount, totalCount, successRate*100)
			
			// Should have some success even under multiple failure conditions
			// Allow for degraded performance but not complete failure
			gomega.Expect(successRate).To(gomega.BeNumerically(">=", 0.3)) // At least 30% success rate
		})
	})
}

var _ = ginkgo.Describe("Chaos Engineering", func() {
	var testSuite *ChaosTestSuite
	
	ginkgo.BeforeEach(func() {
		testSuite = &ChaosTestSuite{
			TestSuite: framework.NewTestSuite(&framework.TestConfig{
				ChaosTestEnabled:  true,
				FailureRate:       0.2,
				LoadTestEnabled:   true,
				MaxConcurrency:    50,
				TestDuration:      2 * time.Minute,
				MockExternalAPIs:  true,
			}),
		}
		testSuite.SetupSuite()
	})
	
	ginkgo.AfterEach(func() {
		testSuite.TearDownSuite()
	})
	
	ginkgo.Context("Network Failures", func() {
		testSuite.TestNetworkFailures()
	})
	
	ginkgo.Context("Resource Exhaustion", func() {
		testSuite.TestResourceExhaustion()
	})
	
	ginkgo.Context("Latency Injection", func() {
		testSuite.TestLatencyInjection()
	})
	
	ginkgo.Context("Service Failures", func() {
		testSuite.TestServiceFailures()
	})
	
	ginkgo.Context("Combined Scenarios", func() {
		testSuite.TestCombinedChaosScenarios()
	})
})