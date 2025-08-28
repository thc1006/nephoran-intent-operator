package security

import (
	"context"
	"crypto/rand"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/patchgen"
	"github.com/thc1006/nephoran-intent-operator/internal/patchgen/generator"
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
			randPrefix := fmt.Sprintf("test-pkg-%x", randBytes)

			// Generate package
			pkg, err := patchgen.GeneratePackage(ctx, &patchgen.PackageOptions{
				Name:      randPrefix,
				Namespace: "test-ns",
				// Add other relevant generation parameters
			})
			require.NoError(t, err)

			// Ensure unique package names
			mu.Lock()
			defer mu.Unlock()

			// Check for name collision
			pkgName := pkg.GetName()
			assert.False(t, generatedPackages[pkgName], 
				"Package name collision detected: %s", pkgName)
			
			generatedPackages[pkgName] = true
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Validate results
	assert.Equal(t, generationCount, len(generatedPackages), 
		"Not all package generations were unique")
}

// TestPatchGenPathTraversalPrevention tests path traversal prevention
func TestPatchGenPathTraversalPrevention(t *testing.T) {
	testCases := []struct {
		name             string
		inputPath        string
		shouldPreventNow bool
	}{
		{
			name:             "Simple Valid Path",
			inputPath:        "valid/path/package",
			shouldPreventNow: false,
		},
		{
			name:             "Path Traversal Attempt 1",
			inputPath:        "../../../etc/passwd",
			shouldPreventNow: true,
		},
		{
			name:             "Path Traversal Attempt 2",
			inputPath:        "/etc/sensitive/config",
			shouldPreventNow: true,
		},
		{
			name:             "Encoded Path Traversal",
			inputPath:        "%2e%2e%2f%2e%2e%2fetc%2fpasswd",
			shouldPreventNow: true,
		},
		{
			name:             "Nested Traversal",
			inputPath:        "valid/path/../../../sensitive",
			shouldPreventNow: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Sanitize path
			sanitizedPath := generator.SanitizePath(tc.inputPath)
			
			// Validate path sanitization
			if tc.shouldPreventNow {
				assert.NotEqual(t, tc.inputPath, sanitizedPath, 
					"Path traversal should have been prevented")
				assert.False(t, strings.Contains(sanitizedPath, ".."), 
					"Path still contains traversal characters")
			} else {
				assert.Equal(t, tc.inputPath, sanitizedPath, 
					"Valid path should remain unchanged")
			}

			// Ensure absolute path cannot escape base directory
			baseDir := "/tmp/nephio/packages"
			fullPath := filepath.Join(baseDir, sanitizedPath)
			assert.True(t, strings.HasPrefix(fullPath, baseDir), 
				"Path cannot escape base directory")
		})
	}
}

// TestPatchGenCommandInjectionPrevention tests command injection prevention
func TestPatchGenCommandInjectionPrevention(t *testing.T) {
	testCases := []struct {
		name                   string
		inputCommand           string
		shouldPreventInjection bool
	}{
		{
			name:                   "Simple Valid Command",
			inputCommand:           "kubectl apply",
			shouldPreventInjection: false,
		},
		{
			name:                   "Shell Injection Attempt 1",
			inputCommand:           "kubectl apply; rm -rf /",
			shouldPreventInjection: true,
		},
		{
			name:                   "Shell Injection Attempt 2",
			inputCommand:           "`/bin/bash -c 'some malicious command'`",
			shouldPreventInjection: true,
		},
		{
			name:                   "Encoded Command Injection",
			inputCommand:           "kubectl%20apply%3B%20cat%20%2Fetc%2Fpasswd",
			shouldPreventInjection: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Sanitize command
			sanitizedCommand := generator.SanitizeCommand(tc.inputCommand)
			
			// Validate command sanitization
			if tc.shouldPreventInjection {
				assert.NotEqual(t, tc.inputCommand, sanitizedCommand, 
					"Command injection should have been prevented")
				assert.False(t, 
					strings.ContainsAny(sanitizedCommand, ";|&`"), 
					"Sanitized command still contains dangerous characters")
			} else {
				assert.Equal(t, tc.inputCommand, sanitizedCommand, 
					"Valid command should remain unchanged")
			}
		})
	}
}

// TestPatchGenTimeoutAndResourceLimits tests timeout and resource constraints
func TestPatchGenTimeoutAndResourceLimits(t *testing.T) {
	// Test configuration
	testCases := []struct {
		name                string
		resourceConstraints config.ResourceLimits
		expectedResult      bool
	}{
		{
			name: "Normal Resource Allocation",
			resourceConstraints: config.ResourceLimits{
				MaxCPU:    2,
				MaxMemory: 4096, // 4 GB
				Timeout:   5 * time.Minute,
			},
			expectedResult: true,
		},
		{
			name: "Excessive CPU Request",
			resourceConstraints: config.ResourceLimits{
				MaxCPU:    100, // Unreasonable
				MaxMemory: 4096,
				Timeout:   5 * time.Minute,
			},
			expectedResult: false,
		},
		{
			name: "Excessive Memory Request",
			resourceConstraints: config.ResourceLimits{
				MaxCPU:    2,
				MaxMemory: 1024 * 1024, // 1 TB
				Timeout:   5 * time.Minute,
			},
			expectedResult: false,
		},
		{
			name: "Extremely Long Timeout",
			resourceConstraints: config.ResourceLimits{
				MaxCPU:    2,
				MaxMemory: 4096,
				Timeout:   24 * time.Hour, // 1 day
			},
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tc.resourceConstraints.Timeout)
			defer cancel()

			// Simulate package generation with resource constraints
			result, err := patchgen.GeneratePackageWithConstraints(ctx, tc.resourceConstraints)
			
			if tc.expectedResult {
				assert.NoError(t, err, "Package generation should succeed")
				assert.NotNil(t, result, "Package should be generated")
			} else {
				assert.Error(t, err, "Package generation should fail")
				assert.Nil(t, result, "Package should not be generated")
			}
		})
	}
}

// TestPatchGenBinaryValidation tests binary and content validation
func TestPatchGenBinaryValidation(t *testing.T) {
	testCases := []struct {
		name           string
		inputContent   []byte
		expectedResult bool
	}{
		{
			name:           "Valid YAML Content",
			inputContent:   []byte("apiVersion: v1\nkind: ConfigMap\n"),
			expectedResult: true,
		},
		{
			name:           "Invalid Binary Content",
			inputContent:   []byte{0x00, 0x01, 0x02, 0x03}, // Raw binary
			expectedResult: false,
		},
		{
			name:           "Malformed YAML",
			inputContent:   []byte("invalid: yaml:\n  nested: {"),
			expectedResult: false,
		},
		{
			name:           "Large Content",
			inputContent:   make([]byte, 10*1024*1024), // 10 MB
			expectedResult: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate binary content
			isValid := generator.ValidateBinaryContent(tc.inputContent)
			assert.Equal(t, tc.expectedResult, isValid, 
				"Binary content validation failed")
		})
	}
}