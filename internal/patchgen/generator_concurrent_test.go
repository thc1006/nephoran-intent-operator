package patchgen

import (
	"fmt"
<<<<<<< HEAD
=======
	"os"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestConcurrentPackageNameGeneration(t *testing.T) {
	const numGoRoutines = 100
	packageNames := make([]string, numGoRoutines)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	tempDir := t.TempDir()

	// Concurrent package name generation
	for i := 0; i < numGoRoutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			intent := &Intent{
				IntentType: "scaling",
				Target:     fmt.Sprintf("app-%d", id),
				Namespace:  "default",
				Replicas:   3,
			}

			patchPackage := NewPatchPackage(intent, tempDir)

			mutex.Lock()
			packageNames[id] = patchPackage.Kptfile.Metadata.Name
			mutex.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify uniqueness of package names
	packageNameSet := make(map[string]bool)
	for _, name := range packageNames {
		assert.False(t, packageNameSet[name], "Package name should be unique")
		packageNameSet[name] = true
	}
}

func TestPackageGenerationStressTest(t *testing.T) {
	const numPackages = 100 // Reduced for Windows performance
	tempDir := t.TempDir()
	var mutex sync.Mutex
	packagesCreated := 0
	var wg sync.WaitGroup
<<<<<<< HEAD
=======
	
	// Use sync.Once to ensure base directory structure is created safely
	var dirCreationOnce sync.Once
	dirCreationError := make(chan error, 1)
	
	// Pre-create all output directories to avoid race conditions
	for i := 0; i < numPackages; i++ {
		outputDir := filepath.Join(tempDir, fmt.Sprintf("stress-output-%d", i))
		if err := os.MkdirAll(outputDir, 0o755); err != nil {
			t.Fatalf("Failed to create output directory %s: %v", outputDir, err)
		}
	}
>>>>>>> 6835433495e87288b95961af7173d866977175ff

	for i := 0; i < numPackages; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
<<<<<<< HEAD
=======
			
			// Ensure base directory creation happens only once
			dirCreationOnce.Do(func() {
				// This ensures any shared directory creation is thread-safe
				if err := os.MkdirAll(tempDir, 0o755); err != nil {
					select {
					case dirCreationError <- err:
					default:
					}
				}
			})
			
			// Check for directory creation errors
			select {
			case err := <-dirCreationError:
				assert.NoError(t, err, "Directory creation should not fail")
				return
			default:
			}
			
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			intent := &Intent{
				IntentType: "scaling",
				Target:     fmt.Sprintf("stress-app-%d", id),
				Namespace:  "stress-test",
				Replicas:   5,
				Reason:     "Performance testing",
				Source:     "StressTest",
			}

			outputDir := filepath.Join(tempDir, fmt.Sprintf("stress-output-%d", id))
<<<<<<< HEAD
			patchPackage := NewPatchPackage(intent, outputDir)

			err := patchPackage.Generate()
=======
			
			// Create the output directory before generating the package
			err := os.MkdirAll(outputDir, 0o755)
			if err != nil {
				t.Errorf("Failed to create output directory %s: %v", outputDir, err)
				return
			}
			
			patchPackage := NewPatchPackage(intent, outputDir)

			err = patchPackage.Generate()
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			assert.NoError(t, err, "Package generation should not fail")

			// Atomic increment of successful package creation
			mutex.Lock()
			if err == nil {
				packagesCreated++
			}
			mutex.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify all packages were created
	assert.Equal(t, numPackages, packagesCreated, "All packages should be created")
}

func TestUniqueTimestampGeneration(t *testing.T) {
	const numTimestamps = 100 // Reduced for Windows performance
	timestamps := make([]string, numTimestamps)
	var mutex sync.Mutex
	var wg sync.WaitGroup

	for i := 0; i < numTimestamps; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			intent := &Intent{
				IntentType: "scaling",
				Target:     fmt.Sprintf("timestamp-test-%d", id),
				Namespace:  "default",
				Replicas:   1,
			}

			patchPackage := NewPatchPackage(intent, "/tmp")
			timestamp := patchPackage.PatchFile.Metadata.Annotations["nephoran.io/generated-at"]

			mutex.Lock()
			timestamps[id] = timestamp
			mutex.Unlock()
		}(i)
	}

	wg.Wait()

	// Verify timestamps are unique
	timestampSet := make(map[string]bool)
	for _, ts := range timestamps {
		_, err := time.Parse(time.RFC3339, ts)
		assert.NoError(t, err, "Timestamp should be valid RFC3339")

		// If the timestamp already exists, the test will fail
		assert.False(t, timestampSet[ts], "Timestamps should be unique")
		timestampSet[ts] = true
	}
}

func TestInvalidIntentHandling(t *testing.T) {
	testCases := []struct {
		name   string
		intent *Intent
	}{
		{
			name: "Empty Target",
			intent: &Intent{
				IntentType: "scaling",
				Target:     "",
				Namespace:  "default",
				Replicas:   1,
			},
		},
		{
			name: "Invalid Namespace",
			intent: &Intent{
				IntentType: "scaling",
				Target:     "app",
				Namespace:  "invalid namespace",
				Replicas:   1,
			},
		},
		{
			name: "Negative Replicas",
			intent: &Intent{
				IntentType: "scaling",
				Target:     "app",
				Namespace:  "default",
				Replicas:   -1,
			},
		},
		{
			name: "Very Large Replica Count",
			intent: &Intent{
				IntentType: "scaling",
				Target:     "app",
				Namespace:  "default",
				Replicas:   10001, // Beyond reasonable limits
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tempDir := t.TempDir()
			patchPackage := NewPatchPackage(tc.intent, tempDir)

			err := patchPackage.Generate()

			assert.Error(t, err, "Invalid intent should generate an error")
		})
	}
}
