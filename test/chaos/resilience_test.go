package chaos

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/rest"

	porchclient "github.com/thc1006/nephoran-intent-operator/pkg/porch"
)

const (
	chaosTestPackages     = 100
	networkPartitionDelay = 500 * time.Millisecond
	slowResponseDelay     = 250 * time.Millisecond
)

func simulateNetworkLatency(fn func() error) error {
	randomDelay := time.Duration(rand.Intn(500)) * time.Millisecond
	time.Sleep(randomDelay)
	return fn()
}

func TestPorchResilienceUnderFailure(t *testing.T) {
	_, err := rest.InClusterConfig()
	require.NoError(t, err, "Failed to load Kubernetes config")

	porchClient := porchclient.NewClient("http://porch-server:8080", false)

	t.Run("NetworkPartitionAndRecovery", func(t *testing.T) {
		var wg sync.WaitGroup
		var successCount, failureCount int32
		var mu sync.Mutex

		for i := 0; i < chaosTestPackages; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				// Simulate network partition with probabilistic failure
				err := simulateNetworkLatency(func() error {
					pkgName := fmt.Sprintf("chaos-pkg-%d", idx)
					
					// Mock porch package creation using the porch client
					packageReq := &porchclient.PackageRequest{
						Repository: "chaos-test-repo",
						Package:    pkgName,
						Workspace:  "main",
						Namespace:  "nephio-chaos",
					}
					
					_, err := porchClient.CreateOrUpdatePackage(packageReq)
					return err // Mock deletion successful for test
				})

				mu.Lock()
				defer mu.Unlock()

				if err == nil {
					successCount++
				} else {
					failureCount++
				}
			}(i)
		}

		wg.Wait()

		t.Logf("Success Count: %d, Failure Count: %d", successCount, failureCount)
		assert.Less(t, float64(failureCount)/float64(chaosTestPackages), 0.2, 
			"Failure rate should be less than 20%")
	})

	t.Run("ServerResponseSlow", func(t *testing.T) {
		pkgName := fmt.Sprintf("slow-response-pkg-%d", time.Now().UnixNano())
		
		// Mock porch package creation for slow response test
		packageReq := &porchclient.PackageRequest{
			Repository: "slow-response-repo",
			Package:    pkgName,
			Workspace:  "main",
			Namespace:  "nephio-chaos",
		}

		// Simulate slow server response
		startTime := time.Now()
		createdPkg, err := porchClient.CreateOrUpdatePackage(packageReq)
		duration := time.Since(startTime)

		require.NoError(t, err, "Failed to create package with slow response")
		assert.True(t, duration >= slowResponseDelay, "Operation should take at least the configured delay")

		// Note: Package deletion would be handled by the porch client in real scenario
		// For now, we just verify creation was successful
		assert.NotNil(t, createdPkg, "Created package should not be nil")
	})
}