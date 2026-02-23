// Package integration provides integration tests for Porch functionality.
//
// Environment Variables:
//   - PORCH_SERVER_URL: Primary environment variable for Porch endpoint (highest priority)
//   - PORCH_ENDPOINT: Alternative environment variable for Porch endpoint
//
// Default Porch URL (if no env vars set):
//   - http://porch-server.porch-system.svc.cluster.local:7007
//
// Usage:
//   # Run tests with default Porch URL
//   go test ./test/integration/...
//
//   # Run tests with custom Porch URL
//   PORCH_SERVER_URL=http://localhost:7007 go test ./test/integration/...
//
//   # Run tests and skip if Porch not available (default behavior)
//   go test ./test/integration/... -v
//
// Tests will automatically skip if Porch is not reachable at the configured URL.
package integration

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	porchclient "github.com/thc1006/nephoran-intent-operator/pkg/porch"
)

// getPorchURL returns the Porch server URL from environment variables or default.
//
// Priority order:
//  1. PORCH_SERVER_URL environment variable
//  2. PORCH_ENDPOINT environment variable
//  3. Default: http://porch-server.porch-system.svc.cluster.local:7007
func getPorchURL() string {
	if url := os.Getenv("PORCH_SERVER_URL"); url != "" {
		return url
	}
	if url := os.Getenv("PORCH_ENDPOINT"); url != "" {
		return url
	}
	return "http://porch-server.porch-system.svc.cluster.local:7007"
}

// checkPorchAvailability probes the Porch server to see if it is reachable.
func checkPorchAvailability(porchURL string) error {
	client := &http.Client{Timeout: 3 * time.Second}
	healthURL := fmt.Sprintf("%s/healthz", porchURL)
	resp, err := client.Get(healthURL)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// Porch v1alpha1 API types - defining locally for integration test compatibility
var porchv1alpha1 = struct {
	Package            func() *Package
	PackageSpec        func() PackageSpec
	PackageStatus      func() PackageStatus
	Workspacev1Package func() Workspacev1Package
}{
	Package:            func() *Package { return &Package{} },
	PackageSpec:        func() PackageSpec { return PackageSpec{} },
	PackageStatus:      func() PackageStatus { return PackageStatus{} },
	Workspacev1Package: func() Workspacev1Package { return Workspacev1Package{} },
}

// Package represents a Porch package resource
type Package struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              PackageSpec   `json:"spec,omitempty"`
	Status            PackageStatus `json:"status,omitempty"`
}

// PackageSpec defines the desired state of Package
type PackageSpec struct {
	Repository         string             `json:"repository,omitempty"`
	Workspacev1Package Workspacev1Package `json:"workspacev1Package,omitempty"`
}

// PackageStatus defines the observed state of Package
type PackageStatus struct {
	Phase      string             `json:"phase,omitempty"`
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Workspacev1Package represents workspace package configuration
type Workspacev1Package struct {
	Description string                 `json:"description,omitempty"`
	Keywords    []string               `json:"keywords,omitempty"`
	Data        map[string]interface{} `json:"data,omitempty"`
}

const (
	testNamespace       = "nephio-test"
	porchServerTimeout  = 2 * time.Minute
	concurrentIntentNum = 50
)

func setupTestEnvironment(t *testing.T, porchURL string) (*rest.Config, *kubernetes.Clientset, *porchclient.Client) {
	// Load Kubernetes configuration (in-cluster or kubeconfig fallback)
	config, err := rest.InClusterConfig()
	if err != nil {
		config, err = clientcmd.BuildConfigFromFlags("", clientcmd.RecommendedHomeFile)
	}
	require.NoError(t, err, "Failed to load Kubernetes config")

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	require.NoError(t, err, "Failed to create Kubernetes clientset")

	// Create Porch client using configurable URL
	porchClient, err := porchclient.NewClient(porchURL, false)
	require.NoError(t, err, "Failed to create Porch client (SSRF validation)")

	// Create test namespace
	ns := &v1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: testNamespace,
		},
	}
	_, err = clientset.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		t.Fatalf("Failed to create test namespace: %v", err)
	}

	return config, clientset, porchClient
}

func TestPorchIntegration(t *testing.T) {
	// Get Porch URL from environment or use default
	porchURL := getPorchURL()
	t.Logf("Using Porch URL: %s", porchURL)

	// Skip if Porch is not deployed
	if err := checkPorchAvailability(porchURL); err != nil {
		t.Skipf("Skipping: Porch server not available at %s - %v\n"+
			"Set PORCH_SERVER_URL or PORCH_ENDPOINT environment variable to override", porchURL, err)
	}

	// Setup test environment
	_, _, porchClient := setupTestEnvironment(t, porchURL)

	// Test Package Creation
	t.Run("CreatePackage", func(t *testing.T) {
		pkgName := fmt.Sprintf("test-package-%d", time.Now().UnixNano())
		pkg := &porchclient.Package{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: testNamespace,
			},
			Spec: porchclient.ClientPackageSpec{
				Repository: "test-repo",
				Workspacev1Package: porchclient.Workspacev1Package{
					Description: "Test integration package",
				},
			},
		}

		createdPkg, err := porchClient.Create(context.Background(), pkg)
		require.NoError(t, err, "Failed to create package")
		assert.NotNil(t, createdPkg, "Created package should not be nil")
		assert.Equal(t, pkgName, createdPkg.Name, "Package name should match")
	})

	// Test Concurrent Package Creation
	t.Run("ConcurrentPackageCreation", func(t *testing.T) {
		var wg sync.WaitGroup
		var mu sync.Mutex
		packages := make([]*porchclient.Package, concurrentIntentNum)

		for i := 0; i < concurrentIntentNum; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				pkgName := fmt.Sprintf("concurrent-pkg-%d", idx)
				pkg := &porchclient.Package{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pkgName,
						Namespace: testNamespace,
					},
					Spec: porchclient.ClientPackageSpec{
						Repository: "test-concurrent-repo",
					},
				}

				createdPkg, err := porchClient.Create(context.Background(), pkg)
				mu.Lock()
				defer mu.Unlock()
				assert.NoError(t, err, "Failed to create concurrent package")
				packages[idx] = createdPkg
			}(i)
		}

		wg.Wait()
		for _, pkg := range packages {
			assert.NotNil(t, pkg, "Concurrent package should be created")
		}
	})

	// Test Package Update
	t.Run("UpdatePackage", func(t *testing.T) {
		pkgName := fmt.Sprintf("update-package-%d", time.Now().UnixNano())
		pkg := &porchclient.Package{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: testNamespace,
			},
			Spec: porchclient.ClientPackageSpec{
				Repository: "test-update-repo",
			},
		}

		createdPkg, err := porchClient.Create(context.Background(), pkg)
		require.NoError(t, err, "Failed to create package for update")

		// Update package
		createdPkg.Spec.Workspacev1Package.Description = "Updated package description"
		updatedPkg, err := porchClient.Update(context.Background(), createdPkg)
		require.NoError(t, err, "Failed to update package")
		assert.Equal(t, "Updated package description", updatedPkg.Spec.Workspacev1Package.Description)
	})

	// Test Package Deletion
	t.Run("DeletePackage", func(t *testing.T) {
		pkgName := fmt.Sprintf("delete-package-%d", time.Now().UnixNano())
		pkg := &porchclient.Package{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: testNamespace,
			},
			Spec: porchclient.ClientPackageSpec{
				Repository: "test-delete-repo",
			},
		}

		createdPkg, err := porchClient.Create(context.Background(), pkg)
		require.NoError(t, err, "Failed to create package for deletion")

		err = porchClient.Delete(context.Background(), createdPkg)
		require.NoError(t, err, "Failed to delete package")

		// Verify deletion
		_, err = porchClient.Get(context.Background(), createdPkg.Name, createdPkg.Namespace)
		assert.Error(t, err, "Package should be deleted")
		assert.True(t, errors.IsNotFound(err), "Error should be a not found error")
	})
}

func TestPorchRollbackScenarios(t *testing.T) {
	// Get Porch URL from environment or use default
	porchURL := getPorchURL()
	t.Logf("Using Porch URL: %s", porchURL)

	// Skip if Porch is not deployed
	if err := checkPorchAvailability(porchURL); err != nil {
		t.Skipf("Skipping: Porch server not available at %s - %v\n"+
			"Set PORCH_SERVER_URL or PORCH_ENDPOINT environment variable to override", porchURL, err)
	}

	_, _, porchClient := setupTestEnvironment(t, porchURL)

	t.Run("RollbackPackageVersion", func(t *testing.T) {
		pkgName := fmt.Sprintf("rollback-package-%d", time.Now().UnixNano())
		pkg := &porchclient.Package{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: testNamespace,
			},
			Spec: porchclient.ClientPackageSpec{
				Repository: "test-rollback-repo",
			},
		}

		// Create initial package
		createdPkg, err := porchClient.Create(context.Background(), pkg)
		require.NoError(t, err, "Failed to create package for rollback")

		// Create multiple package versions
		versions := make([]*porchclient.Package, 3)
		versions[0] = createdPkg

		for i := 1; i < 3; i++ {
			createdPkg.Spec.Workspacev1Package.Description = fmt.Sprintf("Version %d", i)
			updatedPkg, err := porchClient.Update(context.Background(), createdPkg)
			require.NoError(t, err, "Failed to update package version")
			versions[i] = updatedPkg
		}

		// Rollback to a previous version
		rolledBackPkg, err := porchClient.Rollback(context.Background(), versions[0])
		require.NoError(t, err, "Failed to rollback package")
		assert.Equal(t, versions[0].Spec, rolledBackPkg.Spec, "Rolled back package spec should match original")
	})
}
