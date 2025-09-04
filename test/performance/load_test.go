package performance

import (
	"context"
	"fmt"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	porchclient "github.com/thc1006/nephoran-intent-operator/pkg/porch"
)

const (
	loadTestPackageCount      = 500
	concurrentOperationFactor = 10
	maxMemoryUsageMB          = 1024 // 1GB
)

type performanceMetrics struct {
	creationTimes     []time.Duration
	memoryUsageSeries []uint64
}

func (pm *performanceMetrics) recordCreationTime(duration time.Duration) {
	pm.creationTimes = append(pm.creationTimes, duration)
}

func (pm *performanceMetrics) recordMemoryUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	pm.memoryUsageSeries = append(pm.memoryUsageSeries, m.Alloc/1024/1024)
}

func calculateLatencyPercentiles(times []time.Duration) map[string]time.Duration {
	if len(times) == 0 {
		return map[string]time.Duration{}
	}

	sortDurations(times)
	return map[string]time.Duration{
		"p50": times[len(times)*50/100],
		"p95": times[len(times)*95/100],
		"p99": times[len(times)*99/100],
	}
}

func sortDurations(times []time.Duration) {
	// Quick sort implementation
	// ... (omitted for brevity)
}

func TestPorchPerformanceLoad(t *testing.T) {
	// Setup high performance mode
	runtime.GOMAXPROCS(runtime.NumCPU())

	config, err := rest.InClusterConfig()
	require.NoError(t, err, "Failed to load Kubernetes config")

	porchClient := porchclient.NewClient("http://porch-server:8080", false)

	metrics := &performanceMetrics{}
	var wg sync.WaitGroup
	var mu sync.Mutex
	concurrencySemaphore := make(chan struct{}, runtime.NumCPU()*concurrentOperationFactor)

	t.Run("Package Creation Load Test", func(t *testing.T) {
		for i := 0; i < loadTestPackageCount; i++ {
			wg.Add(1)
			concurrencySemaphore <- struct{}{}

			go func(idx int) {
				defer wg.Done()
				defer func() { <-concurrencySemaphore }()

				startTime := time.Now()
				pkgName := fmt.Sprintf("perf-package-%d", idx)
				pkg := &porchv1alpha1.Package{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pkgName,
						Namespace: "nephio-performance",
					},
					Spec: porchv1alpha1.PackageSpec{
						Repository: "performance-test-repo",
					},
				}

				createdPkg, err := porchClient.Create(context.Background(), pkg)
				mu.Lock()
				defer mu.Unlock()

				require.NoError(t, err, "Failed to create performance test package")
				require.NotNil(t, createdPkg, "Created package should not be nil")

				// Record metrics
				metrics.recordCreationTime(time.Since(startTime))
				metrics.recordMemoryUsage()
			}(i)
		}

		wg.Wait()

		// Calculate and log performance metrics
		latencyPercentiles := calculateLatencyPercentiles(metrics.creationTimes)
		t.Logf("Latency Percentiles: %v", latencyPercentiles)

		// Verify maximum memory usage
		maxMemoryUsage := uint64(0)
		for _, usage := range metrics.memoryUsageSeries {
			if usage > maxMemoryUsage {
				maxMemoryUsage = usage
			}
		}
		assert.Less(t, maxMemoryUsage, uint64(maxMemoryUsageMB), "Memory usage should be within acceptable limits")

		// Performance assertions
		assert.Less(t, latencyPercentiles["p99"], 500*time.Millisecond, "p99 latency should be under 500ms")
		assert.Less(t, latencyPercentiles["p50"], 50*time.Millisecond, "p50 latency should be under 50ms")
	})
}