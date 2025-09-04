package optimized

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
)

// BenchmarkOptimizedNetworkIntentController benchmarks the optimized NetworkIntent controller
func BenchmarkOptimizedNetworkIntentController(b *testing.B) {
	// Setup test environment
	s := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(s)
	_ = scheme.AddToScheme(s)

	client := fake.NewClientBuilder().WithScheme(s).Build()
	recorder := record.NewFakeRecorder(100)

	// Create mock dependencies
	deps := &mockDependencies{}
	config := controllers.Config{
		MaxRetries:      3,
		RetryDelay:      1 * time.Second,
		Timeout:         30 * time.Second,
		GitRepoURL:      "https://github.com/test/repo",
		GitBranch:       "main",
		GitDeployPath:   "deployments",
		LLMProcessorURL: "http://localhost:8080",
		UseNephioPorch:  false,
	}

	reconciler := NewOptimizedNetworkIntentReconciler(client, s, recorder, config, deps)
	defer reconciler.Shutdown()

	// Create test NetworkIntent
	networkIntent := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-intent",
			Namespace: "default",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent:      "Deploy high-availability AMF for production",
			Description: "Production AMF deployment with auto-scaling",
		},
	}

	// Create the NetworkIntent in the fake client
	if err := client.Create(context.TODO(), networkIntent); err != nil {
		b.Fatalf("Failed to create test NetworkIntent: %v", err)
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      networkIntent.Name,
			Namespace: networkIntent.Namespace,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark single reconcile operations
	b.Run("SingleReconcile", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_, err := reconciler.Reconcile(ctx, req)
			cancel()
			if err != nil && !isRequeueError(err) {
				b.Errorf("Reconcile failed: %v", err)
			}
		}
	})

	// Benchmark concurrent reconciles
	concurrencyLevels := []int{1, 5, 10, 20}
	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("ConcurrentReconcile-%d", concurrency), func(b *testing.B) {
			var wg sync.WaitGroup
			semaphore := make(chan struct{}, concurrency)

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					semaphore <- struct{}{}
					wg.Add(1)

					go func() {
						defer func() {
							<-semaphore
							wg.Done()
						}()

						ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
						defer cancel()

						_, err := reconciler.Reconcile(ctx, req)
						if err != nil && !isRequeueError(err) {
							b.Errorf("Concurrent reconcile failed: %v", err)
						}
					}()
				}
			})

			wg.Wait()
		})
	}
}

// BenchmarkOptimizedE2NodeSetController benchmarks the optimized E2NodeSet controller
func BenchmarkOptimizedE2NodeSetController(b *testing.B) {
	// Setup test environment
	s := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(s)
	_ = scheme.AddToScheme(s)

	client := fake.NewClientBuilder().WithScheme(s).Build()
	recorder := record.NewFakeRecorder(100)

	reconciler := NewOptimizedE2NodeSetReconciler(client, s, recorder)
	defer reconciler.Shutdown()

	// Create test E2NodeSet
	e2nodeSet := &nephoranv1.E2NodeSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-e2nodeset",
			Namespace: "default",
		},
		Spec: nephoranv1.E2NodeSetSpec{
			Replicas:    5,
			RicEndpoint: "http://ric:8080",
		},
	}

	// Create the E2NodeSet in the fake client
	if err := client.Create(context.TODO(), e2nodeSet); err != nil {
		b.Fatalf("Failed to create test E2NodeSet: %v", err)
	}

	req := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      e2nodeSet.Name,
			Namespace: e2nodeSet.Namespace,
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Benchmark single reconcile operations
	b.Run("SingleReconcile", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			_, err := reconciler.Reconcile(ctx, req)
			cancel()
			if err != nil && !isRequeueError(err) {
				b.Errorf("Reconcile failed: %v", err)
			}
		}
	})

	// Benchmark concurrent reconciles
	concurrencyLevels := []int{1, 3, 5, 10}
	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("ConcurrentReconcile-%d", concurrency), func(b *testing.B) {
			var wg sync.WaitGroup
			semaphore := make(chan struct{}, concurrency)

			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					semaphore <- struct{}{}
					wg.Add(1)

					go func() {
						defer func() {
							<-semaphore
							wg.Done()
						}()

						ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
						defer cancel()

						_, err := reconciler.Reconcile(ctx, req)
						if err != nil && !isRequeueError(err) {
							b.Errorf("Concurrent reconcile failed: %v", err)
						}
					}()
				}
			})

			wg.Wait()
		})
	}

	// Benchmark scaling operations
	scalingTests := []struct {
		name     string
		replicas int32
	}{
		{"Scale-5", 5},
		{"Scale-10", 10},
		{"Scale-20", 20},
		{"Scale-50", 50},
	}

	for _, test := range scalingTests {
		b.Run(test.name, func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				// Update replica count
				e2nodeSet.Spec.Replicas = test.replicas
				client.Update(context.TODO(), e2nodeSet)

				// Reconcile
				ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
				_, err := reconciler.Reconcile(ctx, req)
				cancel()

				if err != nil && !isRequeueError(err) {
					b.Errorf("Scaling reconcile failed: %v", err)
				}
			}
		})
	}
}

// BenchmarkBackoffManager benchmarks the backoff manager
func BenchmarkBackoffManager(b *testing.B) {
	manager := NewBackoffManager()

	errors := []error{
		fmt.Errorf("connection timeout"),
		fmt.Errorf("unauthorized access"),
		fmt.Errorf("resource conflict"),
		fmt.Errorf("rate limit exceeded"),
		fmt.Errorf("validation failed"),
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.Run("GetNextDelay", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			resourceKey := fmt.Sprintf("resource-%d", i%100)
			err := errors[i%len(errors)]
			errorType := manager.ClassifyError(err)

			manager.GetNextDelay(resourceKey, errorType, err)
		}
	})

	b.Run("ConcurrentGetNextDelay", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				resourceKey := fmt.Sprintf("resource-%d", i%100)
				err := errors[i%len(errors)]
				errorType := manager.ClassifyError(err)

				manager.GetNextDelay(resourceKey, errorType, err)
				i++
			}
		})
	})

	b.Run("RecordSuccess", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			resourceKey := fmt.Sprintf("resource-%d", i%100)
			manager.RecordSuccess(resourceKey)
		}
	})
}

// BenchmarkStatusBatcher benchmarks the status batcher
func BenchmarkStatusBatcher(b *testing.B) {
	s := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(s)

	client := fake.NewClientBuilder().WithScheme(s).Build()
	config := DefaultBatchConfig
	config.MaxBatchSize = 50 // Larger batch size for benchmarking
	config.BatchTimeout = 100 * time.Millisecond

	batcher := NewStatusBatcher(client, config)
	defer batcher.Stop()

	// Create some test objects
	for i := 0; i < 100; i++ {
		networkIntent := &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-intent-%d", i),
				Namespace: "default",
			},
		}
		client.Create(context.TODO(), networkIntent)
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.Run("QueueUpdate", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			namespacedName := types.NamespacedName{
				Name:      fmt.Sprintf("test-intent-%d", i%100),
				Namespace: "default",
			}

			condition := metav1.Condition{
				Type:               "Ready",
				Status:             metav1.ConditionTrue,
				Reason:             "TestUpdate",
				Message:            fmt.Sprintf("Test update %d", i),
				LastTransitionTime: metav1.Now(),
			}

			batcher.QueueNetworkIntentUpdate(namespacedName, []metav1.Condition{condition}, "Testing", MediumPriority)
		}
	})

	b.Run("ConcurrentQueueUpdate", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				namespacedName := types.NamespacedName{
					Name:      fmt.Sprintf("test-intent-%d", i%100),
					Namespace: "default",
				}

				condition := metav1.Condition{
					Type:               "Ready",
					Status:             metav1.ConditionTrue,
					Reason:             "ConcurrentTestUpdate",
					Message:            fmt.Sprintf("Concurrent test update %d", i),
					LastTransitionTime: metav1.Now(),
				}

				batcher.QueueNetworkIntentUpdate(namespacedName, []metav1.Condition{condition}, "Testing", MediumPriority)
				i++
			}
		})
	})
}

// BenchmarkControllerMetrics benchmarks the metrics collection
func BenchmarkControllerMetrics(b *testing.B) {
	metrics := NewControllerMetrics()

	controllers := []string{"networkintent", "e2nodeset", "oran"}
	errorTypes := []ErrorType{TransientError, PermanentError, ResourceError, ThrottlingError, ValidationError}
	strategies := []BackoffStrategy{ExponentialBackoff, LinearBackoff, ConstantBackoff}

	b.ResetTimer()
	b.ReportAllocs()

	b.Run("RecordReconcileResult", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			controller := controllers[i%len(controllers)]
			result := "success"
			if i%10 == 0 {
				result = "error"
			}

			metrics.RecordReconcileResult(controller, result)
		}
	})

	b.Run("RecordBackoffDelay", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			controller := controllers[i%len(controllers)]
			errorType := errorTypes[i%len(errorTypes)]
			strategy := strategies[i%len(strategies)]
			delay := time.Duration(i%1000) * time.Millisecond

			metrics.RecordBackoffDelay(controller, errorType, strategy, delay)
		}
	})

	b.Run("NewReconcileTimer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			controller := controllers[i%len(controllers)]
			timer := metrics.NewReconcileTimer(controller, "default", fmt.Sprintf("resource-%d", i), "main")
			timer.Finish()
		}
	})

	b.Run("ConcurrentMetrics", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				controller := controllers[i%len(controllers)]

				// Mix different metric operations
				switch i % 4 {
				case 0:
					metrics.RecordReconcileResult(controller, "success")
				case 1:
					errorType := errorTypes[i%len(errorTypes)]
					strategy := strategies[i%len(strategies)]
					delay := time.Duration(i%1000) * time.Millisecond
					metrics.RecordBackoffDelay(controller, errorType, strategy, delay)
				case 2:
					timer := metrics.NewReconcileTimer(controller, "default", fmt.Sprintf("resource-%d", i), "main")
					timer.Finish()
				case 3:
					metrics.UpdateActiveReconcilers(controller, i%20)
				}

				i++
			}
		})
	})
}

// BenchmarkComparison compares optimized vs original controller performance
func BenchmarkComparison(b *testing.B) {
	// This would compare the optimized controller against the original
	// For now, we'll focus on measuring the optimized controller's performance characteristics

	b.Run("OptimizedController", func(b *testing.B) {
		// Run optimized controller benchmark
		BenchmarkOptimizedNetworkIntentController(b)
	})

	// Memory allocation benchmarks
	b.Run("MemoryAllocations", func(b *testing.B) {
		s := runtime.NewScheme()
		_ = nephoranv1.AddToScheme(s)

		client := fake.NewClientBuilder().WithScheme(s).Build()
		recorder := record.NewFakeRecorder(100)
		deps := &mockDependencies{}
		config := controllers.Config{
			MaxRetries: 3,
			RetryDelay: 1 * time.Second,
		}

		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			reconciler := NewOptimizedNetworkIntentReconciler(client, s, recorder, config, deps)
			reconciler.Shutdown()
		}
	})
}

// Mock dependencies for testing
type mockDependencies struct{}

func (m *mockDependencies) GetGitClient() git.ClientInterface                      { return nil }
func (m *mockDependencies) GetLLMClient() shared.ClientInterface                   { return nil }
func (m *mockDependencies) GetPackageGenerator() *nephio.PackageGenerator          { return nil }
func (m *mockDependencies) GetHTTPClient() *http.Client                            { return nil }
func (m *mockDependencies) GetEventRecorder() record.EventRecorder                 { return nil }
func (m *mockDependencies) GetTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase { return nil }
func (m *mockDependencies) GetMetricsCollector() monitoring.MetricsCollector       { return nil }

// Helper function to check if error is a requeue error (acceptable in benchmarks)
func isRequeueError(err error) bool {
	// In real scenarios, you might check for specific error types that indicate requeuing
	return false // For benchmarking, we'll consider all errors as actual errors
}

// Additional benchmark for controller startup and shutdown
func BenchmarkControllerLifecycle(b *testing.B) {
	s := runtime.NewScheme()
	_ = nephoranv1.AddToScheme(s)

	client := fake.NewClientBuilder().WithScheme(s).Build()
	recorder := record.NewFakeRecorder(100)
	deps := &mockDependencies{}
	config := controllers.Config{
		MaxRetries: 3,
		RetryDelay: 1 * time.Second,
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.Run("ControllerCreation", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			reconciler := NewOptimizedNetworkIntentReconciler(client, s, recorder, config, deps)
			reconciler.Shutdown()
		}
	})

	b.Run("ControllerShutdown", func(b *testing.B) {
		controllers := make([]*OptimizedNetworkIntentReconciler, b.N)

		// Create controllers
		for i := 0; i < b.N; i++ {
			controllers[i] = NewOptimizedNetworkIntentReconciler(client, s, recorder, config, deps)
		}

		b.ResetTimer()

		// Benchmark shutdown
		for i := 0; i < b.N; i++ {
			controllers[i].Shutdown()
		}
	})
}
