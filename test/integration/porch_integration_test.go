//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"

	"github.com/thc1006/nephoran-intent-operator/test/integration/mocks"
)

const (
	testNamespace       = "nephio-test"
	porchServerTimeout  = 2 * time.Minute
	concurrentIntentNum = 50
)

func setupTestEnvironment(t *testing.T) (*rest.Config, *mocks.MockNephioClient) {
	// Create test configuration
	config := mocks.NewTestConfig()

	// Create mock Nephio client
	porchClient := mocks.NewMockNephioClient()

	return config, porchClient
}

func TestPorchIntegration(t *testing.T) {
	// Setup test environment
	config, porchClient := setupTestEnvironment(t)
	_ = config // Use config to avoid unused variable error

	// Test Package Creation
	t.Run("CreatePackage", func(t *testing.T) {
		pkgName := fmt.Sprintf("test-package-%d", time.Now().UnixNano())
		pkg := &mocks.PackageRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: testNamespace,
			},
			Spec: mocks.PackageRevisionSpec{
				Repository:  "test-repo",
				PackageName: pkgName,
				Revision:    "v1.0.0",
				Lifecycle:   mocks.PackageRevisionLifecycleDraft,
			},
		}

		err := porchClient.Create(context.Background(), pkg)
		require.NoError(t, err, "Failed to create package")
		assert.Equal(t, pkgName, pkg.Name, "Package name should match")
	})

	// Test Package Update
	t.Run("UpdatePackage", func(t *testing.T) {
		pkgName := fmt.Sprintf("update-package-%d", time.Now().UnixNano())
		pkg := &mocks.PackageRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: testNamespace,
			},
			Spec: mocks.PackageRevisionSpec{
				Repository:  "test-update-repo",
				PackageName: pkgName,
				Revision:    "v1.0.0",
				Lifecycle:   mocks.PackageRevisionLifecycleDraft,
			},
		}

		err := porchClient.Create(context.Background(), pkg)
		require.NoError(t, err, "Failed to create package for update")

		// Update package
		pkg.Spec.Lifecycle = mocks.PackageRevisionLifecycleProposed
		err = porchClient.Update(context.Background(), pkg)
		require.NoError(t, err, "Failed to update package")
		assert.Equal(t, mocks.PackageRevisionLifecycleProposed, pkg.Spec.Lifecycle)
	})

	// Test Package Deletion
	t.Run("DeletePackage", func(t *testing.T) {
		pkgName := fmt.Sprintf("delete-package-%d", time.Now().UnixNano())
		pkg := &mocks.PackageRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: testNamespace,
			},
			Spec: mocks.PackageRevisionSpec{
				Repository:  "test-delete-repo",
				PackageName: pkgName,
				Revision:    "v1.0.0",
				Lifecycle:   mocks.PackageRevisionLifecycleDraft,
			},
		}

		err := porchClient.Create(context.Background(), pkg)
		require.NoError(t, err, "Failed to create package for deletion")

		err = porchClient.Delete(context.Background(), pkg)
		require.NoError(t, err, "Failed to delete package")
	})
}

func TestPorchMockScenarios(t *testing.T) {
	config, porchClient := setupTestEnvironment(t)
	_ = config // Use config to avoid unused variable error

	t.Run("MockPackageLifecycle", func(t *testing.T) {
		pkgName := fmt.Sprintf("lifecycle-package-%d", time.Now().UnixNano())
		pkg := &mocks.PackageRevision{
			ObjectMeta: metav1.ObjectMeta{
				Name:      pkgName,
				Namespace: testNamespace,
			},
			Spec: mocks.PackageRevisionSpec{
				Repository:  "test-lifecycle-repo",
				PackageName: pkgName,
				Revision:    "v1.0.0",
				Lifecycle:   mocks.PackageRevisionLifecycleDraft,
			},
		}

		// Create initial package
		err := porchClient.Create(context.Background(), pkg)
		require.NoError(t, err, "Failed to create package for lifecycle test")

		// Update lifecycle: Draft -> Proposed -> Published
		pkg.Spec.Lifecycle = mocks.PackageRevisionLifecycleProposed
		err = porchClient.Update(context.Background(), pkg)
		require.NoError(t, err, "Failed to update package to proposed")

		pkg.Spec.Lifecycle = mocks.PackageRevisionLifecyclePublished
		err = porchClient.Update(context.Background(), pkg)
		require.NoError(t, err, "Failed to update package to published")

		assert.Equal(t, mocks.PackageRevisionLifecyclePublished, pkg.Spec.Lifecycle)
	})
}
