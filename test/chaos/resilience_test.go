package chaos

import (
	"context"
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	porchv1alpha1 "github.com/nephio-project/nephio/api/porch/v1alpha1"
	porchclient "github.com/nephio-project/nephio/pkg/client/porch"
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
	config, err := rest.InClusterConfig()
	require.NoError(t, err, "Failed to load Kubernetes config")

	porchClient, err := porchclient.NewClient(config)
	require.NoError(t, err, "Failed to create Porch client")

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
					pkg := &porchv1alpha1.Package{
						ObjectMeta: metav1.ObjectMeta{
							Name:      pkgName,
							Namespace: "nephio-chaos",
						},
						Spec: porchv1alpha1.PackageSpec{
							Repository: "chaos-test-repo",
						},
					}

					createdPkg, err := porchClient.Create(context.Background(), pkg)
					if err != nil {
						return err
					}
					return porchClient.Delete(context.Background(), createdPkg)
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
		pkg := &porchv1alpha1.Package{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: "nephio-chaos",
			},
			Spec: porchv1alpha1.PackageSpec{
				Repository: "slow-response-repo",
			},
		}

		// Simulate slow server response
		startTime := time.Now()
		createdPkg, err := porchClient.CreateWithTimeout(context.Background(), pkg, slowResponseDelay)
		duration := time.Since(startTime)

		require.NoError(t, err, "Failed to create package with slow response")
		assert.True(t, duration >= slowResponseDelay, "Operation should take at least the configured delay")

		// Delete the package
		err = porchClient.Delete(context.Background(), createdPkg)
		require.NoError(t, err, "Failed to delete package after slow response")
	})
}