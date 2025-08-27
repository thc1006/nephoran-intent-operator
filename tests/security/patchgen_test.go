package security

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/patchgen"
)

// TestPatchGenConcurrencySafety tests the concurrent package generation
func TestPatchGenConcurrencySafety(t *testing.T) {
	// Set a high number of concurrent generations
	const (
		generationCount = 1000
		timeout         = 10 * time.Second
	)

	// Track unique generated packages
	generatedPackages := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Context with timeout to prevent infinite loops
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Concurrent package generation
	for i := 0; i < generationCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			// Generate a random prefix to simulate different inputs
			randBytes := make([]byte, 16)
			_, err := rand.Read(randBytes)
			require.NoError(t, err)

			// Use cryptographically secure random number as input
			randNum, err := rand.Int(rand.Reader, big.NewInt(1000000))
			require.NoError(t, err)

			// Generate package with unique options
			opts := &patchgen.PackageOptions{
				Name:      fmt.Sprintf("test-pkg-%d-%s", idx, randNum.String()),
				Namespace: "default",
			}

			pkg, err := patchgen.GeneratePackage(ctx, opts)

			// Handle context cancellation gracefully
			if err != nil && strings.Contains(err.Error(), "context") {
				return
			}

			require.NoError(t, err)
			require.NotNil(t, pkg)

			// Thread-safe package tracking
			mu.Lock()
			defer mu.Unlock()

			// Ensure no name collisions
			assert.False(t, generatedPackages[pkg.Name],
				"Package name collision detected: %s", pkg.Name)
			generatedPackages[pkg.Name] = true
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Verify all packages were generated uniquely
	mu.Lock()
	defer mu.Unlock()

	t.Logf("Successfully generated %d unique packages", len(generatedPackages))
	assert.True(t, len(generatedPackages) > 0, "No packages were generated")
}

// TestResourceConstraintValidation tests resource constraint validation
func TestResourceConstraintValidation(t *testing.T) {
	ctx := context.Background()

	testCases := []struct {
		name        string
		limits      patchgen.ResourceLimits
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid constraints",
			limits: patchgen.ResourceLimits{
				MaxCPU:    4,
				MaxMemory: 16 * 1024, // 16GB in MB
				Timeout:   5 * time.Minute,
			},
			expectError: false,
		},
		{
			name: "CPU exceeds limit",
			limits: patchgen.ResourceLimits{
				MaxCPU:    16, // Exceeds max of 8
				MaxMemory: 8 * 1024,
				Timeout:   5 * time.Minute,
			},
			expectError: true,
			errorMsg:    "CPU allocation exceeds limit",
		},
		{
			name: "Memory exceeds limit",
			limits: patchgen.ResourceLimits{
				MaxCPU:    4,
				MaxMemory: 64 * 1024, // 64GB exceeds max of 32GB
				Timeout:   5 * time.Minute,
			},
			expectError: true,
			errorMsg:    "memory allocation exceeds limit",
		},
		{
			name: "Timeout exceeds limit",
			limits: patchgen.ResourceLimits{
				MaxCPU:    4,
				MaxMemory: 8 * 1024,
				Timeout:   45 * time.Minute, // Exceeds max of 30 minutes
			},
			expectError: true,
			errorMsg:    "timeout exceeds maximum allowed duration",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pkg, err := patchgen.GeneratePackageWithConstraints(ctx, tc.limits)

			if tc.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errorMsg)
				assert.Nil(t, pkg)
			} else {
				require.NoError(t, err)
				require.NotNil(t, pkg)
				assert.NotEmpty(t, pkg.Name)
				assert.Equal(t, "default", pkg.Namespace)
			}
		})
	}
}

// TestPackageNameUniqueness tests that package names are cryptographically unique
func TestPackageNameUniqueness(t *testing.T) {
	ctx := context.Background()
	const numPackages = 100

	names := make(map[string]bool)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < numPackages; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			opts := &patchgen.PackageOptions{
				Name:      "test-package",
				Namespace: "default",
			}

			pkg, err := patchgen.GeneratePackage(ctx, opts)
			require.NoError(t, err)
			require.NotNil(t, pkg)

			mu.Lock()
			defer mu.Unlock()

			// Ensure name is unique
			assert.False(t, names[pkg.Name],
				"Duplicate package name generated: %s", pkg.Name)
			names[pkg.Name] = true
		}(i)
	}

	wg.Wait()

	// Verify all names are unique
	assert.Equal(t, numPackages, len(names),
		"Expected %d unique names, got %d", numPackages, len(names))
}

// TestPackageGenerationTimeout tests timeout handling
func TestPackageGenerationTimeout(t *testing.T) {
	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for context to timeout
	time.Sleep(1 * time.Millisecond)

	opts := &patchgen.PackageOptions{
		Name:      "timeout-test",
		Namespace: "default",
	}

	pkg, err := patchgen.GeneratePackage(ctx, opts)

	// Should handle timeout gracefully
	if err != nil {
		assert.Contains(t, err.Error(), "context")
	}

	// Package should be nil on timeout
	assert.Nil(t, pkg)
}

// TestSecureRandomGeneration tests that random generation is cryptographically secure
func TestSecureRandomGeneration(t *testing.T) {
	ctx := context.Background()
	const iterations = 50

	// Generate multiple packages and verify uniqueness
	names := make([]string, 0, iterations)

	for i := 0; i < iterations; i++ {
		opts := &patchgen.PackageOptions{
			Name:      "random-test",
			Namespace: "default",
		}

		pkg, err := patchgen.GeneratePackage(ctx, opts)
		require.NoError(t, err)
		require.NotNil(t, pkg)

		names = append(names, pkg.Name)
	}

	// Verify all names are unique
	uniqueNames := make(map[string]bool)
	for _, name := range names {
		assert.False(t, uniqueNames[name],
			"Duplicate name detected: %s", name)
		uniqueNames[name] = true
	}

	assert.Equal(t, iterations, len(uniqueNames))
}

// TestPackageStructureValidation tests package structure validation
func TestPackageStructureValidation(t *testing.T) {
	ctx := context.Background()

	opts := &patchgen.PackageOptions{
		Name:      "structure-test",
		Namespace: "test-namespace",
	}

	pkg, err := patchgen.GeneratePackage(ctx, opts)
	require.NoError(t, err)
	require.NotNil(t, pkg)

	// Validate package structure
	assert.NotEmpty(t, pkg.Name)
	assert.Contains(t, pkg.Name, "structure-test")
	assert.Equal(t, "test-namespace", pkg.Namespace)
}

// TestHighVolumeGeneration tests generation under high load
func TestHighVolumeGeneration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high volume test in short mode")
	}

	ctx := context.Background()
	const volumeSize = 5000

	var wg sync.WaitGroup
	var successCount int64
	var mu sync.Mutex

	start := time.Now()

	for i := 0; i < volumeSize; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			opts := &patchgen.PackageOptions{
				Name:      fmt.Sprintf("volume-test-%d", idx),
				Namespace: "default",
			}

			pkg, err := patchgen.GeneratePackage(ctx, opts)
			if err == nil && pkg != nil {
				mu.Lock()
				successCount++
				mu.Unlock()
			}
		}(i)
	}

	wg.Wait()

	duration := time.Since(start)

	t.Logf("Generated %d packages in %v", successCount, duration)
	t.Logf("Rate: %.2f packages/second", float64(successCount)/duration.Seconds())

	// Should generate most packages successfully
	assert.Greater(t, successCount, int64(volumeSize*0.95),
		"Expected at least 95%% success rate")
}
