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

package nephiomulticluster

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/go-logr/logr/testr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	fakeclient "sigs.k8s.io/controller-runtime/pkg/client/fake"
	// 	porchv1alpha1 "github.com/GoogleContainerTools/kpt/porch/api/porchapi/v1alpha1" // DISABLED: external dependency not available
)

// Performance test utilities
type PerformanceMetrics struct {
	OperationsPerSecond float64
	AverageLatency      time.Duration
	P95Latency          time.Duration
	P99Latency          time.Duration
	MemoryUsageMB       float64
	GoroutineCount      int
}

type LatencyMeasurement struct {
	Duration  time.Duration
	Timestamp time.Time
}

func measureLatencies(measurements []LatencyMeasurement) (avg, p95, p99 time.Duration) {
	if len(measurements) == 0 {
		return 0, 0, 0
	}

	// Sort measurements by duration
	durations := make([]time.Duration, len(measurements))
	var sum time.Duration
	for i, m := range measurements {
		durations[i] = m.Duration
		sum += m.Duration
	}

	// Sort durations for percentile calculations
	for i := 0; i < len(durations)-1; i++ {
		for j := i + 1; j < len(durations); j++ {
			if durations[i] > durations[j] {
				durations[i], durations[j] = durations[j], durations[i]
			}
		}
	}

	avg = sum / time.Duration(len(measurements))
	p95Index := int(float64(len(durations)) * 0.95)
	p99Index := int(float64(len(durations)) * 0.99)

	if p95Index >= len(durations) {
		p95Index = len(durations) - 1
	}
	if p99Index >= len(durations) {
		p99Index = len(durations) - 1
	}

	p95 = durations[p95Index]
	p99 = durations[p99Index]

	return avg, p95, p99
}

func getMemoryUsage() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / 1024 / 1024 // Convert to MB
}

func setupPerformanceTestEnvironment(t *testing.T, numClusters int) *ClusterManager {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	client := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	logger := testr.New(t)

	clusterMgr := NewClusterManager(client, logger)
	config := &rest.Config{Host: "https://test-cluster.example.com"}

	// Create test clusters
	for i := 1; i <= numClusters; i++ {
		clusterName := types.NamespacedName{
			Name:      fmt.Sprintf("perf-cluster-%d", i),
			Namespace: "default",
		}

		clusterInfo := &ClusterInfo{
			Name:       clusterName,
			Kubeconfig: config,
			ResourceUtilization: ResourceUtilization{
				CPUTotal:    float64(4 + (i % 8)),
				MemoryTotal: int64((8 + (i % 16)) * 1024 * 1024 * 1024),
			},
			HealthStatus: ClusterHealthStatus{Available: true},
		}

		clusterMgr.clusters[clusterName] = clusterInfo
	}

	return clusterMgr
}

// Benchmark Tests
func BenchmarkClusterManager_RegisterCluster(b *testing.B) {
	scheme := runtime.NewScheme()
	require.NoError(b, corev1.AddToScheme(scheme))

	client := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	logger := testr.New(b)
	clusterMgr := NewClusterManager(client, logger)

	config := &rest.Config{Host: "https://benchmark-cluster.example.com"}
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		clusterName := types.NamespacedName{
			Name:      fmt.Sprintf("bench-cluster-%d", i),
			Namespace: "default",
		}

		_, err := clusterMgr.RegisterCluster(ctx, config, clusterName)
		if err != nil {
			b.Fatalf("Failed to register cluster: %v", err)
		}
	}
}

func BenchmarkClusterManager_SelectTargetClusters_10_Clusters(b *testing.B) {
	clusterMgr := setupPerformanceTestEnvironment(b, 10)
	packageRevision := createTestPackageRevision("bench-package", "v1.0.0")

	candidates := make([]types.NamespacedName, 10)
	for i := 0; i < 10; i++ {
		candidates[i] = types.NamespacedName{
			Name:      fmt.Sprintf("perf-cluster-%d", i+1),
			Namespace: "default",
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := clusterMgr.SelectTargetClusters(ctx, candidates, packageRevision)
		if err != nil {
			b.Fatalf("Failed to select clusters: %v", err)
		}
	}
}

func BenchmarkClusterManager_SelectTargetClusters_100_Clusters(b *testing.B) {
	clusterMgr := setupPerformanceTestEnvironment(b, 100)
	packageRevision := createTestPackageRevision("bench-package", "v1.0.0")

	candidates := make([]types.NamespacedName, 100)
	for i := 0; i < 100; i++ {
		candidates[i] = types.NamespacedName{
			Name:      fmt.Sprintf("perf-cluster-%d", i+1),
			Namespace: "default",
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := clusterMgr.SelectTargetClusters(ctx, candidates, packageRevision)
		if err != nil {
			b.Fatalf("Failed to select clusters: %v", err)
		}
	}
}

func BenchmarkClusterManager_SelectTargetClusters_1000_Clusters(b *testing.B) {
	clusterMgr := setupPerformanceTestEnvironment(b, 1000)
	packageRevision := createTestPackageRevision("bench-package", "v1.0.0")

	candidates := make([]types.NamespacedName, 1000)
	for i := 0; i < 1000; i++ {
		candidates[i] = types.NamespacedName{
			Name:      fmt.Sprintf("perf-cluster-%d", i+1),
			Namespace: "default",
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := clusterMgr.SelectTargetClusters(ctx, candidates, packageRevision)
		if err != nil {
			b.Fatalf("Failed to select clusters: %v", err)
		}
	}
}

func BenchmarkHealthMonitor_ProcessAlerts(b *testing.B) {
	scheme := runtime.NewScheme()
	require.NoError(b, corev1.AddToScheme(scheme))

	client := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	logger := testr.New(b)

	healthMonitor := NewHealthMonitor(client, logger)
	mockHandler := &MockAlertHandler{}
	healthMonitor.RegisterAlertHandler(mockHandler)

	alerts := []Alert{
		{
			Severity:  SeverityWarning,
			Type:      AlertTypeResourcePressure,
			Message:   "Benchmark alert",
			Timestamp: time.Now(),
		},
		{
			Severity:  SeverityCritical,
			Type:      AlertTypeComponentFailure,
			Message:   "Critical benchmark alert",
			Timestamp: time.Now(),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		healthMonitor.processAlerts(alerts)
	}
}

func BenchmarkSyncEngine_SyncPackageToCluster(b *testing.B) {
	scheme := runtime.NewScheme()
	require.NoError(b, corev1.AddToScheme(scheme))

	client := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	logger := testr.New(b)

	syncEngine := NewSyncEngine(client, logger)
	packageRevision := createTestPackageRevision("bench-sync-package", "v1.0.0")
	targetCluster := types.NamespacedName{Name: "bench-target-cluster", Namespace: "default"}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := syncEngine.SyncPackageToCluster(ctx, packageRevision, targetCluster)
		if err != nil {
			b.Fatalf("Failed to sync package: %v", err)
		}
	}
}

// Concurrency Performance Tests
func TestPerformance_ConcurrentClusterSelection(t *testing.T) {
	clusterMgr := setupPerformanceTestEnvironment(t, 50)
	packageRevision := createTestPackageRevision("concurrent-test", "v1.0.0")

	candidates := make([]types.NamespacedName, 50)
	for i := 0; i < 50; i++ {
		candidates[i] = types.NamespacedName{
			Name:      fmt.Sprintf("perf-cluster-%d", i+1),
			Namespace: "default",
		}
	}

	ctx := context.Background()
	numGoroutines := 20
	operationsPerGoroutine := 10

	var wg sync.WaitGroup
	var measurements []LatencyMeasurement
	var measurementsMutex sync.Mutex

	startTime := time.Now()

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			for j := 0; j < operationsPerGoroutine; j++ {
				operationStart := time.Now()

				_, err := clusterMgr.SelectTargetClusters(ctx, candidates, packageRevision)
				operationDuration := time.Since(operationStart)

				assert.NoError(t, err)

				measurementsMutex.Lock()
				measurements = append(measurements, LatencyMeasurement{
					Duration:  operationDuration,
					Timestamp: operationStart,
				})
				measurementsMutex.Unlock()
			}
		}()
	}

	wg.Wait()
	totalDuration := time.Since(startTime)

	// Calculate performance metrics
	totalOperations := numGoroutines * operationsPerGoroutine
	operationsPerSecond := float64(totalOperations) / totalDuration.Seconds()
	avgLatency, p95Latency, p99Latency := measureLatencies(measurements)
	memoryUsage := getMemoryUsage()
	goroutineCount := runtime.NumGoroutine()

	metrics := PerformanceMetrics{
		OperationsPerSecond: operationsPerSecond,
		AverageLatency:      avgLatency,
		P95Latency:          p95Latency,
		P99Latency:          p99Latency,
		MemoryUsageMB:       memoryUsage,
		GoroutineCount:      goroutineCount,
	}

	// Performance assertions
	assert.Greater(t, metrics.OperationsPerSecond, 50.0,
		"Should achieve at least 50 operations per second")
	assert.Less(t, metrics.AverageLatency, 100*time.Millisecond,
		"Average latency should be under 100ms")
	assert.Less(t, metrics.P95Latency, 200*time.Millisecond,
		"P95 latency should be under 200ms")
	assert.Less(t, metrics.MemoryUsageMB, 100.0,
		"Memory usage should be reasonable")

	t.Logf("Performance Metrics:")
	t.Logf("  Operations/sec: %.2f", metrics.OperationsPerSecond)
	t.Logf("  Average Latency: %v", metrics.AverageLatency)
	t.Logf("  P95 Latency: %v", metrics.P95Latency)
	t.Logf("  P99 Latency: %v", metrics.P99Latency)
	t.Logf("  Memory Usage: %.2f MB", metrics.MemoryUsageMB)
	t.Logf("  Goroutines: %d", metrics.GoroutineCount)
}

func TestPerformance_HealthMonitoringScalability(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	client := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	logger := testr.New(t)

	healthMonitor := NewHealthMonitor(client, logger)

	// Setup multiple clusters for monitoring
	numClusters := 100
	for i := 1; i <= numClusters; i++ {
		clusterName := types.NamespacedName{
			Name:      fmt.Sprintf("health-perf-cluster-%d", i),
			Namespace: "default",
		}

		healthMonitor.clusters[clusterName] = &ClusterHealthState{
			Name:          clusterName,
			OverallStatus: HealthStatusHealthy,
		}
	}

	// Register alert handlers
	numHandlers := 5
	handlers := make([]*MockAlertHandler, numHandlers)
	for i := 0; i < numHandlers; i++ {
		handlers[i] = &MockAlertHandler{}
		healthMonitor.RegisterAlertHandler(handlers[i])
	}

	// Start monitoring with high frequency
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	startTime := time.Now()
	healthMonitor.StartHealthMonitoring(ctx, 50*time.Millisecond)

	// Let monitoring run
	time.Sleep(2 * time.Second)

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// Verify performance under load
	assert.Less(t, duration, 6*time.Second,
		"Health monitoring should not significantly delay operations")

	// Verify monitoring is working
	var totalCycles int
	for _, cluster := range healthMonitor.clusters {
		if cluster.LastHealthCheck.After(startTime) {
			totalCycles++
		}
	}

	expectedMinCycles := numClusters / 2 // At least half the clusters should be checked
	assert.Greater(t, totalCycles, expectedMinCycles,
		"Health monitoring should check significant number of clusters")

	t.Logf("Health Monitoring Performance:")
	t.Logf("  Monitored Clusters: %d", numClusters)
	t.Logf("  Monitoring Duration: %v", duration)
	t.Logf("  Health Check Cycles: %d", totalCycles)
}

func TestPerformance_AlertProcessingThroughput(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	client := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	logger := testr.New(t)

	healthMonitor := NewHealthMonitor(client, logger)
	mockHandler := &MockAlertHandler{}
	healthMonitor.RegisterAlertHandler(mockHandler)

	// Generate large number of alerts
	numAlerts := 10000
	alerts := make([]Alert, numAlerts)
	for i := 0; i < numAlerts; i++ {
		alerts[i] = Alert{
			Severity:     SeverityWarning,
			Type:         AlertTypeResourcePressure,
			Message:      fmt.Sprintf("Performance test alert #%d", i),
			ResourceName: fmt.Sprintf("resource-%d", i),
			Timestamp:    time.Now(),
		}
	}

	// Measure alert processing performance
	startTime := time.Now()
	healthMonitor.processAlerts(alerts)
	processingTime := time.Since(startTime)

	alertsPerSecond := float64(numAlerts) / processingTime.Seconds()

	// Performance assertions
	assert.Greater(t, alertsPerSecond, 1000.0,
		"Should process at least 1000 alerts per second")
	assert.Less(t, processingTime, 5*time.Second,
		"Should process 10k alerts within 5 seconds")

	// Verify all alerts were processed
	processedCount := mockHandler.GetAlertsCount()
	assert.Equal(t, numAlerts, processedCount,
		"All alerts should be processed")

	t.Logf("Alert Processing Performance:")
	t.Logf("  Alerts Processed: %d", numAlerts)
	t.Logf("  Processing Time: %v", processingTime)
	t.Logf("  Alerts/Second: %.2f", alertsPerSecond)
}

func TestPerformance_MemoryLeaks(t *testing.T) {
	// Monitor memory usage during prolonged operations
	runtime.GC()
	initialMemory := getMemoryUsage()

	clusterMgr := setupPerformanceTestEnvironment(t, 10)
	packageRevision := createTestPackageRevision("memory-test", "v1.0.0")

	candidates := make([]types.NamespacedName, 10)
	for i := 0; i < 10; i++ {
		candidates[i] = types.NamespacedName{
			Name:      fmt.Sprintf("perf-cluster-%d", i+1),
			Namespace: "default",
		}
	}

	ctx := context.Background()

	// Perform many operations to detect memory leaks
	numIterations := 1000
	for i := 0; i < numIterations; i++ {
		_, err := clusterMgr.SelectTargetClusters(ctx, candidates, packageRevision)
		assert.NoError(t, err)

		// Force GC periodically
		if i%100 == 0 {
			runtime.GC()
		}
	}

	// Final GC and memory check
	runtime.GC()
	time.Sleep(100 * time.Millisecond) // Allow GC to complete
	runtime.GC()

	finalMemory := getMemoryUsage()
	memoryGrowth := finalMemory - initialMemory

	// Memory growth assertions
	assert.Less(t, memoryGrowth, 50.0,
		"Memory growth should be reasonable (under 50MB)")

	t.Logf("Memory Usage:")
	t.Logf("  Initial: %.2f MB", initialMemory)
	t.Logf("  Final: %.2f MB", finalMemory)
	t.Logf("  Growth: %.2f MB", memoryGrowth)
	t.Logf("  Iterations: %d", numIterations)
}

func TestPerformance_ConcurrentChannelOperations(t *testing.T) {
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))

	client := fakeclient.NewClientBuilder().WithScheme(scheme).Build()
	logger := testr.New(t)

	healthMonitor := NewHealthMonitor(client, logger)

	numClusters := 50
	numUpdatesPerCluster := 100

	var wg sync.WaitGroup
	startTime := time.Now()

	// Create channels and start concurrent operations
	for i := 1; i <= numClusters; i++ {
		clusterName := types.NamespacedName{
			Name:      fmt.Sprintf("channel-perf-cluster-%d", i),
			Namespace: "default",
		}

		// Register health channel
		healthChan := healthMonitor.RegisterHealthChannel(clusterName)

		// Start goroutine to consume health updates
		wg.Add(1)
		go func(cluster types.NamespacedName, ch <-chan HealthUpdate) {
			defer wg.Done()

			receivedCount := 0
			for update := range ch {
				assert.Equal(t, cluster, update.ClusterName)
				receivedCount++
				if receivedCount >= numUpdatesPerCluster {
					break
				}
			}
		}(clusterName, healthChan)

		// Start goroutine to send health updates
		wg.Add(1)
		go func(cluster types.NamespacedName) {
			defer wg.Done()

			for j := 0; j < numUpdatesPerCluster; j++ {
				update := HealthUpdate{
					ClusterName: cluster,
					Status:      HealthStatusHealthy,
					Timestamp:   time.Now(),
					Alerts:      []Alert{},
				}

				healthMonitor.notifyHealthChannels(update)
			}
		}(clusterName)
	}

	wg.Wait()
	processingTime := time.Since(startTime)

	// Clean up channels
	for i := 1; i <= numClusters; i++ {
		clusterName := types.NamespacedName{
			Name:      fmt.Sprintf("channel-perf-cluster-%d", i),
			Namespace: "default",
		}
		healthMonitor.UnregisterHealthChannel(clusterName)
	}

	totalUpdates := numClusters * numUpdatesPerCluster
	updatesPerSecond := float64(totalUpdates) / processingTime.Seconds()

	// Performance assertions
	assert.Greater(t, updatesPerSecond, 1000.0,
		"Should process at least 1000 health updates per second")
	assert.Less(t, processingTime, 10*time.Second,
		"Should complete all updates within reasonable time")

	t.Logf("Channel Performance:")
	t.Logf("  Clusters: %d", numClusters)
	t.Logf("  Updates per Cluster: %d", numUpdatesPerCluster)
	t.Logf("  Total Updates: %d", totalUpdates)
	t.Logf("  Processing Time: %v", processingTime)
	t.Logf("  Updates/Second: %.2f", updatesPerSecond)
}
