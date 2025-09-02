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

package benchmarks

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/controllers"
	"github.com/thc1006/nephoran-intent-operator/tests/fixtures"
)

// BenchmarkNetworkIntentCreation benchmarks NetworkIntent creation performance
func BenchmarkNetworkIntentCreation(b *testing.B) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	if err != nil {
		b.Fatalf("Failed to add scheme: %v", err)
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		intent := fixtures.SimpleNetworkIntent()
		intent.Name = fmt.Sprintf("benchmark-intent-%d", i)
		intent.Namespace = "default"

		err := k8sClient.Create(ctx, intent)
		if err != nil {
			b.Fatalf("Failed to create intent %d: %v", i, err)
		}
	}
}

// BenchmarkNetworkIntentRetrieval benchmarks NetworkIntent retrieval performance
func BenchmarkNetworkIntentRetrieval(b *testing.B) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	if err != nil {
		b.Fatalf("Failed to add scheme: %v", err)
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	// Pre-populate with test data
	const numObjects = 1000
	for i := 0; i < numObjects; i++ {
		intent := fixtures.SimpleNetworkIntent()
		intent.Name = fmt.Sprintf("benchmark-intent-%d", i)
		intent.Namespace = "default"
		err := k8sClient.Create(ctx, intent)
		if err != nil {
			b.Fatalf("Failed to create intent %d: %v", i, err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		intent := &nephoranv1.NetworkIntent{}
		err := k8sClient.Get(ctx, types.NamespacedName{
			Name:      fmt.Sprintf("benchmark-intent-%d", i%numObjects),
			Namespace: "default",
		}, intent)
		if err != nil {
			b.Fatalf("Failed to get intent %d: %v", i, err)
		}
	}
}

// BenchmarkNetworkIntentUpdate benchmarks NetworkIntent update performance
func BenchmarkNetworkIntentUpdate(b *testing.B) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	if err != nil {
		b.Fatalf("Failed to add scheme: %v", err)
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	// Pre-create test object
	intent := fixtures.SimpleNetworkIntent()
	intent.Name = "benchmark-update-intent"
	intent.Namespace = "default"
	err = k8sClient.Create(ctx, intent)
	if err != nil {
		b.Fatalf("Failed to create intent: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Get latest version
		key := types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace}
		err := k8sClient.Get(ctx, key, intent)
		if err != nil {
			b.Fatalf("Failed to get intent: %v", err)
		}

		// Update intent
		intent.Spec.Intent = fmt.Sprintf("updated intent %d", i)
		err = k8sClient.Update(ctx, intent)
		if err != nil {
			b.Fatalf("Failed to update intent %d: %v", i, err)
		}
	}
}

// BenchmarkNetworkIntentList benchmarks NetworkIntent list performance
func BenchmarkNetworkIntentList(b *testing.B) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	if err != nil {
		b.Fatalf("Failed to add scheme: %v", err)
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	// Pre-populate with test data
	const numObjects = 100
	for i := 0; i < numObjects; i++ {
		intent := fixtures.SimpleNetworkIntent()
		intent.Name = fmt.Sprintf("benchmark-list-%d", i)
		intent.Namespace = "default"
		err := k8sClient.Create(ctx, intent)
		if err != nil {
			b.Fatalf("Failed to create intent %d: %v", i, err)
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		intentList := &nephoranv1.NetworkIntentList{}
		err := k8sClient.List(ctx, intentList, &client.ListOptions{Namespace: "default"})
		if err != nil {
			b.Fatalf("Failed to list intents: %v", err)
		}
		if len(intentList.Items) != numObjects {
			b.Fatalf("Expected %d intents, got %d", numObjects, len(intentList.Items))
		}
	}
}

// BenchmarkControllerReconcile benchmarks controller reconciliation performance
func BenchmarkControllerReconcile(b *testing.B) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	if err != nil {
		b.Fatalf("Failed to add scheme: %v", err)
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	reconciler := &controllers.NetworkIntentReconciler{
		Client:          k8sClient,
		Scheme:          scheme,
		Log:             zap.New(zap.UseDevMode(false)), // Use production logger for benchmarks
		EnableLLMIntent: false,                          // Disable LLM for pure reconcile performance
	}

	// Pre-create test object
	intent := fixtures.SimpleNetworkIntent()
	intent.Name = "benchmark-reconcile-intent"
	intent.Namespace = "default"
	err = k8sClient.Create(ctx, intent)
	if err != nil {
		b.Fatalf("Failed to create intent: %v", err)
	}

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := reconciler.Reconcile(ctx, request)
		if err != nil {
			b.Fatalf("Reconcile failed: %v", err)
		}
	}
}

// BenchmarkConcurrentNetworkIntentOperations benchmarks concurrent operations
func BenchmarkConcurrentNetworkIntentOperations(b *testing.B) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	if err != nil {
		b.Fatalf("Failed to add scheme: %v", err)
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			intent := fixtures.SimpleNetworkIntent()
			intent.Name = fmt.Sprintf("concurrent-intent-%d", i)
			intent.Namespace = "default"

			err := k8sClient.Create(ctx, intent)
			if err != nil {
				b.Fatalf("Failed to create intent: %v", err)
			}
			i++
		}
	})
}

// BenchmarkFixtureGeneration benchmarks test fixture generation performance
func BenchmarkFixtureGeneration(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = fixtures.SimpleNetworkIntent()
	}
}

// BenchmarkFixtureGenerationComplex benchmarks complex fixture generation
func BenchmarkFixtureGenerationComplex(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = fixtures.ComplexIntent()
	}
}

// BenchmarkFixtureGenerationWithMetadata benchmarks fixture with metadata generation
func BenchmarkFixtureGenerationWithMetadata(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = fixtures.IntentWithLabelsAndAnnotations()
	}
}

// BenchmarkMultipleIntentsGeneration benchmarks bulk fixture generation
func BenchmarkMultipleIntentsGeneration(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = fixtures.MultipleIntents(10)
	}
}

// BenchmarkJSONMarshaling benchmarks JSON marshaling performance
func BenchmarkJSONMarshaling(b *testing.B) {
	intent := fixtures.ComplexIntent()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(intent)
		if err != nil {
			b.Fatalf("Failed to marshal JSON: %v", err)
		}
	}
}

// BenchmarkJSONUnmarshaling benchmarks JSON unmarshaling performance
func BenchmarkJSONUnmarshaling(b *testing.B) {
	intent := fixtures.ComplexIntent()
	data, err := json.Marshal(intent)
	if err != nil {
		b.Fatalf("Failed to marshal test data: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		var unmarshaled nephoranv1.NetworkIntent
		err := json.Unmarshal(data, &unmarshaled)
		if err != nil {
			b.Fatalf("Failed to unmarshal JSON: %v", err)
		}
	}
}

// BenchmarkDeepCopy benchmarks deep copy performance
func BenchmarkDeepCopy(b *testing.B) {
	intent := fixtures.ComplexIntent()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = intent.DeepCopy()
	}
}

// BenchmarkMemoryUsage benchmarks memory usage patterns
func BenchmarkMemoryUsage(b *testing.B) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	if err != nil {
		b.Fatalf("Failed to add scheme: %v", err)
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Create multiple objects to simulate memory pressure
		intents := fixtures.MultipleIntents(10)
		for j, intent := range intents {
			intent.Name = fmt.Sprintf("memory-test-%d-%d", i, j)
			intent.Namespace = "default"

			err := k8sClient.Create(ctx, intent)
			if err != nil {
				b.Fatalf("Failed to create intent: %v", err)
			}
		}

		// Force some GC pressure
		_ = make([]byte, 1024) // 1KB allocation
	}
}

// BenchmarkHighConcurrency benchmarks high concurrency scenarios
func BenchmarkHighConcurrency(b *testing.B) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	if err != nil {
		b.Fatalf("Failed to add scheme: %v", err)
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	const numWorkers = 100
	var wg sync.WaitGroup

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		wg.Add(numWorkers)

		for w := 0; w < numWorkers; w++ {
			go func(worker, iteration int) {
				defer wg.Done()

				intent := fixtures.SimpleNetworkIntent()
				intent.Name = fmt.Sprintf("high-concurrency-%d-%d", iteration, worker)
				intent.Namespace = "default"

				err := k8sClient.Create(ctx, intent)
				if err != nil {
					b.Errorf("Worker %d failed to create intent: %v", worker, err)
				}
			}(w, i)
		}

		wg.Wait()
	}
}

// BenchmarkControllerConcurrentReconcile benchmarks concurrent reconciliation
func BenchmarkControllerConcurrentReconcile(b *testing.B) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	if err != nil {
		b.Fatalf("Failed to add scheme: %v", err)
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	reconciler := &controllers.NetworkIntentReconciler{
		Client:          k8sClient,
		Scheme:          scheme,
		Log:             zap.New(zap.UseDevMode(false)),
		EnableLLMIntent: false,
	}

	// Pre-create test objects
	const numIntents = 10
	requests := make([]ctrl.Request, numIntents)
	for i := 0; i < numIntents; i++ {
		intent := fixtures.SimpleNetworkIntent()
		intent.Name = fmt.Sprintf("concurrent-reconcile-%d", i)
		intent.Namespace = "default"

		err := k8sClient.Create(ctx, intent)
		if err != nil {
			b.Fatalf("Failed to create intent %d: %v", i, err)
		}

		requests[i] = ctrl.Request{
			NamespacedName: types.NamespacedName{Name: intent.Name, Namespace: intent.Namespace},
		}
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			request := requests[i%numIntents]
			_, err := reconciler.Reconcile(ctx, request)
			if err != nil {
				b.Fatalf("Reconcile failed: %v", err)
			}
			i++
		}
	})
}

// BenchmarkScalabilityTest benchmarks scalability with increasing load
func BenchmarkScalabilityTest(b *testing.B) {
	sizes := []int{10, 100, 1000, 5000}

	for _, size := range sizes {
		b.Run(fmt.Sprintf("Size-%d", size), func(b *testing.B) {
			scheme := runtime.NewScheme()
			err := nephoranv1.AddToScheme(scheme)
			if err != nil {
				b.Fatalf("Failed to add scheme: %v", err)
			}

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
			ctx := context.Background()

			// Pre-populate with data
			for i := 0; i < size; i++ {
				intent := fixtures.SimpleNetworkIntent()
				intent.Name = fmt.Sprintf("scalability-test-%d", i)
				intent.Namespace = "default"
				err := k8sClient.Create(ctx, intent)
				if err != nil {
					b.Fatalf("Failed to create intent %d: %v", i, err)
				}
			}

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				intentList := &nephoranv1.NetworkIntentList{}
				err := k8sClient.List(ctx, intentList, &client.ListOptions{Namespace: "default"})
				if err != nil {
					b.Fatalf("Failed to list intents: %v", err)
				}
				if len(intentList.Items) != size {
					b.Fatalf("Expected %d intents, got %d", size, len(intentList.Items))
				}
			}
		})
	}
}

// BenchmarkResourceCleanup benchmarks resource cleanup performance
func BenchmarkResourceCleanup(b *testing.B) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	if err != nil {
		b.Fatalf("Failed to add scheme: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
		ctx := context.Background()

		// Create resources
		const numResources = 50
		intents := make([]*nephoranv1.NetworkIntent, numResources)
		for j := 0; j < numResources; j++ {
			intent := fixtures.SimpleNetworkIntent()
			intent.Name = fmt.Sprintf("cleanup-test-%d-%d", i, j)
			intent.Namespace = "default"

			err := k8sClient.Create(ctx, intent)
			if err != nil {
				b.Fatalf("Failed to create intent: %v", err)
			}
			intents[j] = intent
		}

		// Benchmark cleanup
		b.StartTimer()
		for _, intent := range intents {
			err := k8sClient.Delete(ctx, intent)
			if err != nil {
				b.Fatalf("Failed to delete intent: %v", err)
			}
		}
		b.StopTimer()
	}
}

// BenchmarkControllerSetup benchmarks controller setup time
func BenchmarkControllerSetup(b *testing.B) {
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		scheme := runtime.NewScheme()
		err := nephoranv1.AddToScheme(scheme)
		if err != nil {
			b.Fatalf("Failed to add scheme: %v", err)
		}

		k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		_ = &controllers.NetworkIntentReconciler{
			Client:          k8sClient,
			Scheme:          scheme,
			Log:             zap.New(zap.UseDevMode(false)),
			EnableLLMIntent: false,
		}
	}
}

// BenchmarkLargeIntentProcessing benchmarks processing of large intent strings
func BenchmarkLargeIntentProcessing(b *testing.B) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	if err != nil {
		b.Fatalf("Failed to add scheme: %v", err)
	}

	k8sClient := fake.NewClientBuilder().WithScheme(scheme).Build()
	ctx := context.Background()

	// Create intent with large content
	intent := fixtures.LongIntent()
	intent.Namespace = "default"

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		intent.Name = fmt.Sprintf("large-intent-%d", i)
		err := k8sClient.Create(ctx, intent)
		if err != nil {
			b.Fatalf("Failed to create large intent: %v", err)
		}
	}
}

// BenchmarkAnalysisTips provides guidance on interpreting benchmark results
func BenchmarkAnalysisTips(b *testing.B) {
	// This function demonstrates how to analyze benchmark results
	// Run benchmarks with: go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof
	// Guidance:
	// 1. Look for ns/op (nanoseconds per operation) - lower is better
	// 2. Check B/op (bytes per operation) - indicates memory usage
	// 3. Monitor allocs/op (allocations per operation) - fewer is better
	// 4. Use -benchtime to run longer benchmarks for more stable results
	// 5. Run multiple times and compare results to identify trends
}
