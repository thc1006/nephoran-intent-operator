package integration

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	porchclient "github.com/thc1006/nephoran-intent-operator/pkg/porch"
	"github.com/thc1006/nephoran-intent-operator/internal/intent"
)

const (
	testNamespace       = "nephio-test"
	porchServerTimeout  = 2 * time.Minute
	concurrentIntentNum = 50
)

func setupTestEnvironment(t *testing.T) (*rest.Config, *kubernetes.Clientset, *porchclient.Client) {
	// Load Kubernetes configuration
	config, err := rest.InClusterConfig()
	require.NoError(t, err, "Failed to load Kubernetes config")

	// Create Kubernetes clientset
	clientset, err := kubernetes.NewForConfig(config)
	require.NoError(t, err, "Failed to create Kubernetes clientset")

	// Create Porch client
	porchClient := porchclient.NewClient("http://porch-server:8080", false)

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
	// Setup test environment
	config, clientset, porchClient := setupTestEnvironment(t)

	// Test Package Creation
	t.Run("CreatePackage", func(t *testing.T) {
		pkgName := fmt.Sprintf("test-package-%d", time.Now().UnixNano())
		pkg := &porchv1alpha1.Package{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: testNamespace,
			},
			Spec: porchv1alpha1.PackageSpec{
				Repository: "test-repo",
				Workspacev1Package: porchv1alpha1.Workspacev1Package{
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
		packages := make([]*porchv1alpha1.Package, concurrentIntentNum)

		for i := 0; i < concurrentIntentNum; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()
				pkgName := fmt.Sprintf("concurrent-pkg-%d", idx)
				pkg := &porchv1alpha1.Package{
					ObjectMeta: metav1.ObjectMeta{
						Name:      pkgName,
						Namespace: testNamespace,
					},
					Spec: porchv1alpha1.PackageSpec{
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
		pkg := &porchv1alpha1.Package{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: testNamespace,
			},
			Spec: porchv1alpha1.PackageSpec{
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
		pkg := &porchv1alpha1.Package{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: testNamespace,
			},
			Spec: porchv1alpha1.PackageSpec{
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
	config, _, porchClient := setupTestEnvironment(t)

	t.Run("RollbackPackageVersion", func(t *testing.T) {
		pkgName := fmt.Sprintf("rollback-package-%d", time.Now().UnixNano())
		pkg := &porchv1alpha1.Package{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: testNamespace,
			},
			Spec: porchv1alpha1.PackageSpec{
				Repository: "test-rollback-repo",
			},
		}

		// Create initial package
		createdPkg, err := porchClient.Create(context.Background(), pkg)
		require.NoError(t, err, "Failed to create package for rollback")

		// Create multiple package versions
		versions := make([]*porchv1alpha1.Package, 3)
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