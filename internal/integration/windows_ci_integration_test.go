//go:build windows && integration
// +build windows,integration

package integration

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nephio-project/nephoran-intent-operator/internal/loop"
	"github.com/nephio-project/nephoran-intent-operator/internal/platform"
)

// TestWindowsCIIntegration provides end-to-end integration tests that validate
// the complete Windows CI pipeline functionality and reliability.
func TestWindowsCIIntegration(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows CI integration tests only apply to Windows")
	}

	if testing.Short() {
		t.Skip("Skipping CI integration tests in short mode")
	}

	t.Run("End_To_End_Pipeline_Simulation", func(t *testing.T) {
		// Simulate a complete CI pipeline run on Windows
		testDir := t.TempDir()
		watchDir := filepath.Join(testDir, "watch")
		statusDir := filepath.Join(watchDir, "status")
		processedDir := filepath.Join(watchDir, "processed")

		// Create directory structure
		require.NoError(t, os.MkdirAll(watchDir, 0755))

		// Step 1: Create conductor loop configuration
		config := loop.Config{
			PorchPath: filepath.Join(testDir, "mock-porch"),
			Mode:      "once",
		}

		// Step 2: Create mock porch script using the fixed PowerShell generation
		mockPorchDir := filepath.Dir(config.PorchPath)
		require.NoError(t, os.MkdirAll(mockPorchDir, 0755))

		mockPorchScript := platform.GetScriptPath(mockPorchDir, "mock-porch")
		opts := platform.ScriptOptions{
			Sleep:    100 * time.Millisecond,
			Stdout:   "Mock porch execution completed successfully",
			Stderr:   "",
			ExitCode: 0,
		}

		err := platform.CreateCrossPlatformScript(mockPorchScript, platform.MockPorchScript, opts)
		require.NoError(t, err, "Should create mock porch script")

		// Verify the script doesn't have the PowerShell concatenation issue
		scriptContent, err := os.ReadFile(mockPorchScript)
		require.NoError(t, err)
		assert.NotContains(t, string(scriptContent), "100echo",
			"Mock porch script should not have PowerShell concatenation issues")

		// Step 3: Create state manager
		sm, err := loop.NewStateManager(watchDir)
		require.NoError(t, err, "State manager creation should succeed")
		defer sm.Close()

		// Step 4: Create watcher
		watcher, err := loop.NewWatcher(watchDir, config)
		require.NoError(t, err, "Watcher creation should succeed")
		defer watcher.Close()

		// Step 5: Create test intent files (simulating CI input)
		intentFiles := []struct {
			name    string
			content string
		}{
			{
				name: "scale-up-intent.json",
				content: `{
					"api_version": "intent.nephoran.io/v1",
					"kind": "ScaleIntent",
					"metadata": {"name": "ci-test-scale-up"},
					"spec": {"replicas": 5, "target": "test-workload"}
				}`,
			},
			{
				name: "config-intent.json",
				content: `{
					"api_version": "intent.nephoran.io/v1",
					"kind": "ConfigIntent",
					"metadata": {"name": "ci-test-config"},
					"spec": {"configMap": "test-config", "data": {"key": "value"}}
				}`,
			},
			{
				name: "network-intent.json",
				content: `{
					"api_version": "intent.nephoran.io/v1",
					"kind": "NetworkIntent", 
					"metadata": {"name": "ci-test-network"},
					"spec": {"policy": "allow-all", "namespace": "default"}
				}`,
			},
		}

		// Create intent files
		for _, intent := range intentFiles {
			intentPath := filepath.Join(watchDir, intent.name)
			require.NoError(t, os.WriteFile(intentPath, []byte(intent.content), 0644))
			t.Logf("Created intent file: %s", intentPath)
		}

		// Step 6: Run the processing pipeline
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Process files one by one to simulate CI behavior
		for _, intent := range intentFiles {
			t.Run("process_"+intent.name, func(t *testing.T) {
				intentPath := filepath.Join(watchDir, intent.name)

				// Verify file exists and is readable
				_, err := os.Stat(intentPath)
				require.NoError(t, err, "Intent file should exist")

				// Check if already processed
				processed, err := sm.IsProcessed(intent.name)
				require.NoError(t, err, "IsProcessed should not error")
				if processed {
					t.Logf("File %s already processed, skipping", intent.name)
					return
				}

				// Simulate the processing that would happen in CI
				t.Logf("Processing intent file: %s", intent.name)

				// Execute the mock porch script
				cmd := exec.CommandContext(ctx, "cmd.exe", "/C", mockPorchScript)
				cmd.Dir = watchDir
				output, err := cmd.CombinedOutput()

				t.Logf("Mock porch output for %s: %s", intent.name, string(output))

				// Should not have PowerShell errors
				outputStr := string(output)
				assert.NotContains(t, outputStr, "Cannot bind parameter",
					"Should not have PowerShell parameter binding errors")
				assert.NotContains(t, outputStr, "cannot convert value",
					"Should not have PowerShell conversion errors")

				if err != nil {
					t.Logf("Mock porch execution error (may be expected): %v", err)
				}

				// Mark as processed
				err = sm.MarkProcessed(intent.name)
				require.NoError(t, err, "Should mark file as processed")

				// Write status file (this tests parent directory creation)
				watcher.writeStatusFileAtomic(intentPath, "success",
					fmt.Sprintf("CI pipeline processed %s successfully", intent.name))

				// Verify status file was created
				_, err = os.Stat(statusDir)
				assert.NoError(t, err, "Status directory should be created")

				statusEntries, err := os.ReadDir(statusDir)
				require.NoError(t, err, "Should read status directory")
				assert.Greater(t, len(statusEntries), 0, "Should have status files")

				// Move to processed (simulating CI cleanup)
				require.NoError(t, os.MkdirAll(processedDir, 0755))
				processedPath := filepath.Join(processedDir, intent.name)
				err = os.Rename(intentPath, processedPath)
				require.NoError(t, err, "Should move file to processed directory")

				// Verify file was moved
				_, err = os.Stat(intentPath)
				assert.True(t, os.IsNotExist(err), "Original file should not exist")

				_, err = os.Stat(processedPath)
				assert.NoError(t, err, "Processed file should exist")
			})
		}

		// Step 7: Verify final state
		t.Run("verify_final_state", func(t *testing.T) {
			// All files should be processed
			for _, intent := range intentFiles {
				processed, err := sm.IsProcessed(intent.name)
				assert.NoError(t, err, "Final IsProcessed check should not error")
				assert.True(t, processed, "File should be marked as processed: %s", intent.name)
			}

			// Status directory should exist with files
			statusEntries, err := os.ReadDir(statusDir)
			require.NoError(t, err, "Should read final status directory")
			assert.GreaterOrEqual(t, len(statusEntries), len(intentFiles),
				"Should have status files for all processed intents")

			// Processed directory should have all files
			processedEntries, err := os.ReadDir(processedDir)
			require.NoError(t, err, "Should read processed directory")
			assert.Equal(t, len(intentFiles), len(processedEntries),
				"All intent files should be in processed directory")

			// Verify processed files have correct content
			for _, intent := range intentFiles {
				processedPath := filepath.Join(processedDir, intent.name)
				content, err := os.ReadFile(processedPath)
				assert.NoError(t, err, "Should read processed file: %s", intent.name)
				assert.Contains(t, string(content), intent.name,
					"Processed file should contain original content")
			}
		})
	})

	t.Run("CI_Build_Test_Integration", func(t *testing.T) {
		// Test integration with actual Go build/test commands that run in CI
		testDir := t.TempDir()

		// Create a minimal Go module for testing
		goModContent := `module test-ci-integration

go 1.21

require github.com/stretchr/testify v1.8.4
`
		require.NoError(t, os.WriteFile(filepath.Join(testDir, "go.mod"), []byte(goModContent), 0644))

		// Create a simple test file that uses Windows-specific functionality
		testFileContent := `package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	
	"github.com/stretchr/testify/assert"
)

func TestWindowsIntegration(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows integration test")
	}
	
	// Test path operations
	testPath := filepath.Join("C:", "test", "path")
	assert.Contains(t, testPath, "\\")
	
	// Test environment
	userProfile := os.Getenv("USERPROFILE")
	if userProfile != "" {
		assert.True(t, filepath.IsAbs(userProfile))
	}
}
`
		require.NoError(t, os.WriteFile(filepath.Join(testDir, "main_test.go"), []byte(testFileContent), 0644))

		// Create main.go file
		mainContent := `package main

import "fmt"

func main() {
	fmt.Println("CI integration test")
}
`
		require.NoError(t, os.WriteFile(filepath.Join(testDir, "main.go"), []byte(mainContent), 0644))

		// Test Go commands that would run in CI
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		t.Run("go_mod_tidy", func(t *testing.T) {
			cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
			cmd.Dir = testDir
			output, err := cmd.CombinedOutput()
			t.Logf("go mod tidy output: %s", string(output))
			assert.NoError(t, err, "go mod tidy should succeed on Windows")
		})

		t.Run("go_build", func(t *testing.T) {
			cmd := exec.CommandContext(ctx, "go", "build", ".")
			cmd.Dir = testDir
			output, err := cmd.CombinedOutput()
			t.Logf("go build output: %s", string(output))
			assert.NoError(t, err, "go build should succeed on Windows")

			// Verify executable was created
			execPath := filepath.Join(testDir, "test-ci-integration.exe")
			_, err = os.Stat(execPath)
			assert.NoError(t, err, "Executable should be created")
		})

		t.Run("go_test", func(t *testing.T) {
			cmd := exec.CommandContext(ctx, "go", "test", "-v", ".")
			cmd.Dir = testDir
			output, err := cmd.CombinedOutput()
			t.Logf("go test output: %s", string(output))
			assert.NoError(t, err, "go test should succeed on Windows")

			// Verify test output contains expected results
			outputStr := string(output)
			assert.Contains(t, outputStr, "TestWindowsIntegration", "Should run Windows integration test")
		})
	})

	t.Run("PowerShell_Script_Execution_CI", func(t *testing.T) {
		// Test PowerShell script execution similar to CI environment
		testDir := t.TempDir()

		// Create various PowerShell scripts that might be used in CI
		testScripts := []struct {
			name        string
			sleepMs     time.Duration
			expectedOut string
			description string
		}{
			{
				name:        "quick-test",
				sleepMs:     50 * time.Millisecond,
				expectedOut: "Quick CI test completed",
				description: "Fast CI operation",
			},
			{
				name:        "build-simulation",
				sleepMs:     200 * time.Millisecond,
				expectedOut: "Build simulation completed",
				description: "Build process simulation",
			},
			{
				name:        "test-execution",
				sleepMs:     500 * time.Millisecond,
				expectedOut: "Test execution completed",
				description: "Test suite execution",
			},
		}

		for _, script := range testScripts {
			t.Run("script_"+script.name, func(t *testing.T) {
				opts := platform.ScriptOptions{
					Sleep:    script.sleepMs,
					Stdout:   script.expectedOut,
					ExitCode: 0,
				}

				scriptPath := platform.GetScriptPath(testDir, script.name)
				err := platform.CreateCrossPlatformScript(scriptPath, platform.MockPorchScript, opts)
				require.NoError(t, err, "Should create script: %s", script.description)

				// Verify script doesn't have concatenation issues
				content, err := os.ReadFile(scriptPath)
				require.NoError(t, err)
				scriptContent := string(content)

				sleepMs := int(script.sleepMs.Milliseconds())
				concatenationPattern := fmt.Sprintf("%decho", sleepMs)
				assert.NotContains(t, scriptContent, concatenationPattern,
					"Script should not have concatenation issue: %s", script.description)

				// Execute script
				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()

				cmd := exec.CommandContext(ctx, "cmd.exe", "/C", scriptPath)
				output, err := cmd.CombinedOutput()

				t.Logf("Script %s output: %s", script.name, string(output))

				// Should not have PowerShell errors
				outputStr := string(output)
				assert.NotContains(t, outputStr, "Cannot bind parameter",
					"No PowerShell parameter errors: %s", script.description)
				assert.NotContains(t, outputStr, "cannot convert value",
					"No PowerShell conversion errors: %s", script.description)

				// Should contain expected output
				if script.expectedOut != "" {
					assert.Contains(t, outputStr, script.expectedOut,
						"Should contain expected output: %s", script.description)
				}
			})
		}
	})

	t.Run("Concurrent_CI_Operations", func(t *testing.T) {
		// Simulate concurrent CI operations like multiple test runners
		testDir := t.TempDir()
		baseWatchDir := filepath.Join(testDir, "concurrent-ci")

		numConcurrentJobs := 5
		intentsPerJob := 3

		var allResults []struct {
			jobID   int
			success bool
			error   string
		}

		// Create concurrent "CI jobs"
		for jobID := 0; jobID < numConcurrentJobs; jobID++ {
			t.Run(fmt.Sprintf("ci_job_%d", jobID), func(t *testing.T) {
				t.Parallel()

				// Each job has its own watch directory
				jobWatchDir := filepath.Join(baseWatchDir, fmt.Sprintf("job-%d", jobID))
				require.NoError(t, os.MkdirAll(jobWatchDir, 0755))

				// Create job-specific configuration
				config := loop.Config{
					PorchPath: filepath.Join(testDir, fmt.Sprintf("mock-porch-job-%d", jobID)),
					Mode:      "once",
				}

				// Create state manager for this job
				sm, err := loop.NewStateManager(jobWatchDir)
				require.NoError(t, err, "Job %d state manager should be created", jobID)
				defer sm.Close()

				// Create watcher for this job
				watcher, err := loop.NewWatcher(jobWatchDir, config)
				require.NoError(t, err, "Job %d watcher should be created", jobID)
				defer watcher.Close()

				// Create intent files for this job
				for intentID := 0; intentID < intentsPerJob; intentID++ {
					intentName := fmt.Sprintf("job-%d-intent-%d.json", jobID, intentID)
					intentContent := fmt.Sprintf(`{
						"api_version": "intent.nephoran.io/v1",
						"kind": "CITestIntent",
						"metadata": {"name": "%s"},
						"spec": {"job": %d, "intent": %d}
					}`, intentName, jobID, intentID)

					intentPath := filepath.Join(jobWatchDir, intentName)
					require.NoError(t, os.WriteFile(intentPath, []byte(intentContent), 0644))

					// Process the intent
					err := sm.MarkProcessed(intentName)
					assert.NoError(t, err, "Job %d should mark intent %d as processed", jobID, intentID)

					// Write status
					watcher.writeStatusFileAtomic(intentPath, "success",
						fmt.Sprintf("CI job %d processed intent %d", jobID, intentID))
				}

				// Verify job completion
				statusDir := filepath.Join(jobWatchDir, "status")
				statusEntries, err := os.ReadDir(statusDir)
				assert.NoError(t, err, "Job %d should have status directory", jobID)
				assert.GreaterOrEqual(t, len(statusEntries), intentsPerJob,
					"Job %d should have status files", jobID)

				allResults = append(allResults, struct {
					jobID   int
					success bool
					error   string
				}{
					jobID:   jobID,
					success: true,
					error:   "",
				})
			})
		}

		// Verify all concurrent jobs succeeded
		t.Run("verify_concurrent_results", func(t *testing.T) {
			successfulJobs := 0
			for _, result := range allResults {
				if result.success {
					successfulJobs++
				} else {
					t.Errorf("Job %d failed: %s", result.jobID, result.error)
				}
			}

			assert.Equal(t, numConcurrentJobs, successfulJobs,
				"All concurrent CI jobs should succeed")
		})
	})

	t.Run("Windows_Specific_CI_Environment", func(t *testing.T) {
		// Test Windows-specific CI environment characteristics
		t.Run("environment_variables", func(t *testing.T) {
			// Test common Windows CI environment variables
			windowsEnvVars := []string{
				"USERPROFILE",
				"TEMP",
				"SYSTEMROOT",
				"PROGRAMFILES",
			}

			for _, envVar := range windowsEnvVars {
				value := os.Getenv(envVar)
				if value != "" {
					t.Logf("Windows env var %s = %s", envVar, value)
					assert.True(t, len(value) > 0, "Windows env var should have value: %s", envVar)
				}
			}
		})

		t.Run("path_handling", func(t *testing.T) {
			// Test Windows path handling in CI context
			testPaths := []string{
				"C:\\Windows\\System32",
				"C:\\Program Files",
				"C:\\Users\\Default",
			}

			for _, testPath := range testPaths {
				isAbs := filepath.IsAbs(testPath)
				assert.True(t, isAbs, "Windows path should be absolute: %s", testPath)

				cleaned := filepath.Clean(testPath)
				assert.Equal(t, testPath, cleaned, "Windows path should be clean: %s", testPath)
			}
		})

		t.Run("file_operations", func(t *testing.T) {
			// Test file operations that are common in CI
			testDir := t.TempDir()

			// Test creating nested directories
			nestedDir := filepath.Join(testDir, "ci", "artifacts", "logs")
			err := os.MkdirAll(nestedDir, 0755)
			assert.NoError(t, err, "Should create nested directories")

			// Test writing files with Windows line endings
			testFile := filepath.Join(nestedDir, "ci-log.txt")
			content := "CI test started\r\nTest completed\r\nResults: SUCCESS\r\n"
			err = os.WriteFile(testFile, []byte(content), 0644)
			assert.NoError(t, err, "Should write file with Windows line endings")

			// Test reading file back
			readContent, err := os.ReadFile(testFile)
			assert.NoError(t, err, "Should read file back")
			assert.Contains(t, string(readContent), "\r\n", "Should preserve Windows line endings")

			// Test file permissions (Windows-specific behavior)
			info, err := os.Stat(testFile)
			assert.NoError(t, err, "Should stat file")
			assert.False(t, info.IsDir(), "Should be a file, not directory")
		})
	})
}

// TestWindowsCIRegressionPrevention ensures the complete CI pipeline
// doesn't regress on Windows-specific issues.
func TestWindowsCIRegressionPrevention(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows CI regression tests only apply to Windows")
	}

	t.Run("Complete_Pipeline_No_PowerShell_Errors", func(t *testing.T) {
		// Run complete pipeline and ensure no PowerShell errors occur
		testDir := t.TempDir()
		watchDir := filepath.Join(testDir, "pipeline-test")
		require.NoError(t, os.MkdirAll(watchDir, 0755))

		config := loop.Config{
			PorchPath: filepath.Join(testDir, "pipeline-porch"),
			Mode:      "once",
		}

		// Create mock porch that would trigger the original PowerShell issue
		mockPorchDir := filepath.Dir(config.PorchPath)
		require.NoError(t, os.MkdirAll(mockPorchDir, 0755))

		mockPorchScript := platform.GetScriptPath(mockPorchDir, "pipeline-porch")
		opts := platform.ScriptOptions{
			Sleep:    50 * time.Millisecond, // The problematic duration from CI failures
			Stdout:   "Pipeline porch execution completed",
			ExitCode: 0,
		}

		err := platform.CreateCrossPlatformScript(mockPorchScript, platform.MockPorchScript, opts)
		require.NoError(t, err)

		// Verify the generated script doesn't have the regression pattern
		scriptContent, err := os.ReadFile(mockPorchScript)
		require.NoError(t, err)
		assert.NotContains(t, string(scriptContent), "50echo",
			"Pipeline script must not have the regression pattern '50echo'")

		// Execute the complete pipeline
		sm, err := loop.NewStateManager(watchDir)
		require.NoError(t, err)
		defer sm.Close()

		watcher, err := loop.NewWatcher(watchDir, config)
		require.NoError(t, err)
		defer watcher.Close()

		// Create test intent
		intentFile := filepath.Join(watchDir, "pipeline-test.json")
		intentContent := `{
			"api_version": "intent.nephoran.io/v1",
			"kind": "PipelineTest",
			"metadata": {"name": "regression-test"},
			"spec": {"test": "complete-pipeline"}
		}`
		require.NoError(t, os.WriteFile(intentFile, []byte(intentContent), 0644))

		// Execute pipeline steps
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		cmd := exec.CommandContext(ctx, "cmd.exe", "/C", mockPorchScript)
		output, err := cmd.CombinedOutput()

		t.Logf("Pipeline execution output: %s", string(output))

		// The critical check: no PowerShell parameter binding errors
		outputStr := string(output)
		assert.NotContains(t, outputStr, "Cannot bind parameter 'Milliseconds'",
			"Pipeline should not have PowerShell parameter binding errors")
		assert.NotContains(t, outputStr, "cannot convert value '50echo' to type 'System.Int32'",
			"Pipeline should not have the specific regression error")
		assert.NotContains(t, outputStr, "Int32", "Pipeline should not have Int32 conversion errors")

		// Complete pipeline operations
		err = sm.MarkProcessed("pipeline-test.json")
		assert.NoError(t, err, "Pipeline should mark file as processed")

		watcher.writeStatusFileAtomic(intentFile, "success", "Pipeline completed without errors")

		// Verify no errors in the complete pipeline
		statusDir := filepath.Join(watchDir, "status")
		_, err = os.Stat(statusDir)
		assert.NoError(t, err, "Status directory should be created")

		statusEntries, err := os.ReadDir(statusDir)
		assert.NoError(t, err, "Should read status directory")
		assert.Greater(t, len(statusEntries), 0, "Should have status files")
	})

	t.Run("High_Load_CI_Simulation", func(t *testing.T) {
		// Simulate high load CI scenario that exposed the original issues
		testDir := t.TempDir()

		// Create multiple concurrent pipeline runs
		numPipelines := 5
		var pipelineErrors []error

		for i := 0; i < numPipelines; i++ {
			pipelineDir := filepath.Join(testDir, fmt.Sprintf("pipeline-%d", i))
			watchDir := filepath.Join(pipelineDir, "watch")
			require.NoError(t, os.MkdirAll(watchDir, 0755))

			config := loop.Config{
				PorchPath: filepath.Join(pipelineDir, "high-load-porch"),
				Mode:      "once",
			}

			// Create pipeline-specific mock porch
			mockPorchScript := platform.GetScriptPath(filepath.Dir(config.PorchPath), "high-load-porch")
			opts := platform.ScriptOptions{
				Sleep:    time.Duration(50+(i*10)) * time.Millisecond,
				Stdout:   fmt.Sprintf("High load pipeline %d completed", i),
				ExitCode: 0,
			}

			err := platform.CreateCrossPlatformScript(mockPorchScript, platform.MockPorchScript, opts)
			if err != nil {
				pipelineErrors = append(pipelineErrors, fmt.Errorf("pipeline %d script creation: %w", i, err))
				continue
			}

			// Verify no concatenation issues
			scriptContent, err := os.ReadFile(mockPorchScript)
			if err != nil {
				pipelineErrors = append(pipelineErrors, fmt.Errorf("pipeline %d script read: %w", i, err))
				continue
			}

			sleepMs := 50 + (i * 10)
			concatenationPattern := fmt.Sprintf("%decho", sleepMs)
			if strings.Contains(string(scriptContent), concatenationPattern) {
				pipelineErrors = append(pipelineErrors, fmt.Errorf("pipeline %d has concatenation issue", i))
				continue
			}

			// Execute pipeline
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			cmd := exec.CommandContext(ctx, "cmd.exe", "/C", mockPorchScript)
			output, err := cmd.CombinedOutput()
			cancel()

			if err != nil {
				t.Logf("Pipeline %d execution error (may be expected): %v", i, err)
			}

			// Check for PowerShell errors in output
			outputStr := string(output)
			if strings.Contains(outputStr, "Cannot bind parameter") {
				pipelineErrors = append(pipelineErrors, fmt.Errorf("pipeline %d has parameter binding error", i))
			}
			if strings.Contains(outputStr, "cannot convert value") {
				pipelineErrors = append(pipelineErrors, fmt.Errorf("pipeline %d has conversion error", i))
			}
		}

		// Report any pipeline errors
		for _, err := range pipelineErrors {
			t.Errorf("High load pipeline error: %v", err)
		}

		assert.Equal(t, 0, len(pipelineErrors), "No pipelines should have errors in high load scenario")
	})
}
