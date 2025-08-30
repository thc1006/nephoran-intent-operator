//go:build windows
// +build windows

package platform

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWindowsPowerShellComprehensive provides comprehensive test coverage for Windows PowerShell
// command separation, execution, and edge cases that were fixed to resolve CI failures.
func TestWindowsPowerShellComprehensive(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows PowerShell comprehensive tests only apply to Windows")
	}

	t.Run("PowerShell_Command_Separation_Edge_Cases", func(t *testing.T) {
		testCases := []struct {
			name     string
			sleep    time.Duration
			stdout   string
			stderr   string
			expected struct {
				sleepMs     string
				stdoutCmd   string
				stderrCmd   string
				noConcat    []string // Patterns that should NOT exist
				mustContain []string // Patterns that MUST exist
			}
		}{
			{
				name:   "very_short_sleep",
				sleep:  1 * time.Millisecond,
				stdout: "Quick test",
				stderr: "Quick error",
				expected: struct {
					sleepMs     string
					stdoutCmd   string
					stderrCmd   string
					noConcat    []string
					mustContain []string
				}{
					sleepMs:     "1",
					stdoutCmd:   "echo Quick test",
					stderrCmd:   "echo Quick error >&2",
					noConcat:    []string{"1echo", "1Quick", "testecho"},
					mustContain: []string{"Start-Sleep -Milliseconds 1", "-NoProfile"},
				},
			},
			{
				name:   "millisecond_boundary_values",
				sleep:  999 * time.Millisecond,
				stdout: "Boundary test output",
				stderr: "",
				expected: struct {
					sleepMs     string
					stdoutCmd   string
					stderrCmd   string
					noConcat    []string
					mustContain []string
				}{
					sleepMs:     "999",
					stdoutCmd:   "echo Boundary test output",
					stderrCmd:   "",
					noConcat:    []string{"999echo", "999Boundary", "outputecho"},
					mustContain: []string{"Start-Sleep -Milliseconds 999"},
				},
			},
			{
				name:   "second_to_millisecond_conversion",
				sleep:  3 * time.Second,
				stdout: "Long operation complete",
				stderr: "Warning: Long operation",
				expected: struct {
					sleepMs     string
					stdoutCmd   string
					stderrCmd   string
					noConcat    []string
					mustContain []string
				}{
					sleepMs:     "3000",
					stdoutCmd:   "echo Long operation complete",
					stderrCmd:   "echo Warning: Long operation >&2",
					noConcat:    []string{"3000echo", "3000Long", "completeecho", "operationecho"},
					mustContain: []string{"Start-Sleep -Milliseconds 3000"},
				},
			},
			{
				name:   "special_characters_in_output",
				sleep:  25 * time.Millisecond,
				stdout: "Output with & special < > characters",
				stderr: "Error with | pipe && double",
				expected: struct {
					sleepMs     string
					stdoutCmd   string
					stderrCmd   string
					noConcat    []string
					mustContain []string
				}{
					sleepMs:     "25",
					stdoutCmd:   "echo Output with & special < > characters",
					stderrCmd:   "echo Error with | pipe && double >&2",
					noConcat:    []string{"25echo", "25Output", "charactersecho", "doubleecho"},
					mustContain: []string{"Start-Sleep -Milliseconds 25"},
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				tempDir := t.TempDir()
				opts := ScriptOptions{
					Sleep:    tc.sleep,
					Stdout:   tc.stdout,
					Stderr:   tc.stderr,
					ExitCode: 0,
				}

				scriptPath := GetScriptPath(tempDir, "test-"+tc.name)
				err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
				require.NoError(t, err)

				content, err := os.ReadFile(scriptPath)
				require.NoError(t, err)
				scriptContent := string(content)

				t.Logf("Test case %s script:\n%s", tc.name, scriptContent)

				// Verify required patterns exist
				for _, pattern := range tc.expected.mustContain {
					assert.Contains(t, scriptContent, pattern,
						"Script must contain pattern: %s", pattern)
				}

				// Verify concatenation patterns don't exist
				for _, badPattern := range tc.expected.noConcat {
					assert.NotContains(t, scriptContent, badPattern,
						"Script must NOT contain concatenated pattern: %s", badPattern)
				}

				// Verify specific command format
				expectedPSCmd := fmt.Sprintf("powershell -NoProfile -Command \"Start-Sleep -Milliseconds %s\"", tc.expected.sleepMs)
				assert.Contains(t, scriptContent, expectedPSCmd,
					"PowerShell command must be properly formatted")

				if tc.expected.stdoutCmd != "" {
					assert.Contains(t, scriptContent, tc.expected.stdoutCmd)
				}
				if tc.expected.stderrCmd != "" {
					assert.Contains(t, scriptContent, tc.expected.stderrCmd)
				}
			})
		}
	})

	t.Run("PowerShell_Execution_Validation", func(t *testing.T) {
		// Test that generated scripts actually execute without PowerShell errors
		tempDir := t.TempDir()

		testCases := []struct {
			name        string
			sleep       time.Duration
			expectedOut string
			expectedErr string
		}{
			{
				name:        "minimal_execution",
				sleep:       10 * time.Millisecond,
				expectedOut: "PowerShell execution test",
				expectedErr: "",
			},
			{
				name:        "with_stderr",
				sleep:       15 * time.Millisecond,
				expectedOut: "Success output",
				expectedErr: "Warning output",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				opts := ScriptOptions{
					Sleep:    tc.sleep,
					Stdout:   tc.expectedOut,
					Stderr:   tc.expectedErr,
					ExitCode: 0,
				}

				scriptPath := GetScriptPath(tempDir, "exec-"+tc.name)
				err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
				require.NoError(t, err)

				// Execute the script with timeout to prevent hanging
				ctx := make(chan bool, 1)
				var output []byte
				var execErr error

				go func() {
					cmd := exec.Command("cmd.exe", "/C", scriptPath)
					output, execErr = cmd.CombinedOutput()
					ctx <- true
				}()

				select {
				case <-ctx:
					// Script completed
				case <-time.After(5 * time.Second):
					t.Fatal("Script execution timed out")
				}

				outputStr := string(output)
				t.Logf("Execution output for %s:\n%s", tc.name, outputStr)
				if execErr != nil {
					t.Logf("Execution error: %v", execErr)
				}

				// Verify no PowerShell parameter binding errors
				assert.NotContains(t, outputStr, "Cannot bind parameter",
					"No PowerShell parameter binding errors")
				assert.NotContains(t, outputStr, "cannot convert value",
					"No PowerShell type conversion errors")
				assert.NotContains(t, outputStr, "Int32",
					"No Int32 conversion errors")

				// The specific regression case
				assert.NotContains(t, outputStr, "50echo",
					"No command concatenation in output")

				// If stdout is expected, check for it
				if tc.expectedOut != "" {
					assert.Contains(t, outputStr, tc.expectedOut,
						"Expected stdout should be present")
				}
			})
		}
	})

	t.Run("PowerShell_Stress_Testing", func(t *testing.T) {
		// Stress test with multiple concurrent PowerShell executions
		tempDir := t.TempDir()
		numConcurrent := 5
		var wg sync.WaitGroup

		errors := make(chan error, numConcurrent)
		outputs := make(chan string, numConcurrent)

		for i := 0; i < numConcurrent; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				opts := ScriptOptions{
					Sleep:    time.Duration(10+id*5) * time.Millisecond,
					Stdout:   fmt.Sprintf("Concurrent test %d", id),
					Stderr:   fmt.Sprintf("Concurrent error %d", id),
					ExitCode: 0,
				}

				scriptPath := GetScriptPath(tempDir, fmt.Sprintf("stress-%d", id))
				err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
				if err != nil {
					errors <- fmt.Errorf("script creation failed for %d: %w", id, err)
					return
				}

				cmd := exec.Command("cmd.exe", "/C", scriptPath)
				output, err := cmd.CombinedOutput()
				if err != nil {
					errors <- fmt.Errorf("execution failed for %d: %w", id, err)
					return
				}

				outputs <- string(output)
			}(i)
		}

		wg.Wait()
		close(errors)
		close(outputs)

		// Check for errors
		var errCount int
		for err := range errors {
			t.Errorf("Concurrent execution error: %v", err)
			errCount++
		}

		assert.Equal(t, 0, errCount, "No concurrent execution errors expected")

		// Verify outputs
		outputCount := 0
		for output := range outputs {
			outputCount++
			t.Logf("Concurrent output %d: %s", outputCount, output)

			// Verify no concatenation errors in any output
			assert.NotContains(t, output, "echo", "No command concatenation")
			assert.NotContains(t, output, "Cannot bind parameter", "No parameter binding errors")
		}

		assert.Equal(t, numConcurrent, outputCount, "All concurrent executions should produce output")
	})

	t.Run("PowerShell_Command_Line_Validation", func(t *testing.T) {
		// Validate that generated PowerShell commands are syntactically correct
		tempDir := t.TempDir()

		// Test various sleep values that caused issues
		problematicSleeps := []time.Duration{
			50 * time.Millisecond, // Original failing case
			100 * time.Millisecond,
			1 * time.Second,
			5 * time.Second,
		}

		for _, sleep := range problematicSleeps {
			t.Run(fmt.Sprintf("sleep_%dms", sleep.Milliseconds()), func(t *testing.T) {
				opts := ScriptOptions{
					Sleep:    sleep,
					Stdout:   "Command validation test",
					ExitCode: 0,
				}

				scriptPath := GetScriptPath(tempDir, "validate")
				err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
				require.NoError(t, err)

				content, err := os.ReadFile(scriptPath)
				require.NoError(t, err)

				// Parse the script line by line to validate PowerShell commands
				scanner := bufio.NewScanner(strings.NewReader(string(content)))
				for scanner.Scan() {
					line := strings.TrimSpace(scanner.Text())
					if strings.Contains(line, "powershell") && strings.Contains(line, "Start-Sleep") {
						// Extract the PowerShell command
						if strings.Contains(line, "-NoProfile -Command") {
							// Find the command within quotes
							startIdx := strings.Index(line, "\"")
							endIdx := strings.LastIndex(line, "\"")
							if startIdx != -1 && endIdx != -1 && startIdx < endIdx {
								psCommand := line[startIdx+1 : endIdx]
								t.Logf("PowerShell command: %s", psCommand)

								// Validate command structure
								assert.True(t, strings.HasPrefix(psCommand, "Start-Sleep"),
									"PowerShell command should start with Start-Sleep")
								assert.Contains(t, psCommand, "-Milliseconds",
									"Should contain -Milliseconds parameter")

								// Extract milliseconds value
								parts := strings.Split(psCommand, " ")
								var msValue string
								for i, part := range parts {
									if part == "-Milliseconds" && i+1 < len(parts) {
										msValue = parts[i+1]
										break
									}
								}

								// Validate milliseconds value is numeric
								if msValue != "" {
									_, err := strconv.Atoi(msValue)
									assert.NoError(t, err, "Milliseconds value should be numeric: %s", msValue)

									// Ensure it matches expected value
									expectedMs := int(sleep.Milliseconds())
									actualMs, _ := strconv.Atoi(msValue)
									assert.Equal(t, expectedMs, actualMs,
										"Milliseconds value should match expected: %d vs %d", expectedMs, actualMs)
								}
							}
						}
					}
				}
			})
		}
	})

	t.Run("PowerShell_Error_Handling", func(t *testing.T) {
		// Test error scenarios and ensure graceful handling
		tempDir := t.TempDir()

		t.Run("invalid_sleep_duration", func(t *testing.T) {
			// Test edge case: zero duration
			opts := ScriptOptions{
				Sleep:    0,
				Stdout:   "Zero sleep test",
				ExitCode: 0,
			}

			scriptPath := GetScriptPath(tempDir, "zero-sleep")
			err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
			require.NoError(t, err)

			content, err := os.ReadFile(scriptPath)
			require.NoError(t, err)

			// Should handle zero duration gracefully (no sleep command generated)
			scriptContent := string(content)
			assert.NotContains(t, scriptContent, "Start-Sleep") // No sleep command for 0 duration
			assert.NotContains(t, scriptContent, "0echo")
		})

		t.Run("very_large_sleep_duration", func(t *testing.T) {
			// Test with large duration
			opts := ScriptOptions{
				Sleep:    10 * time.Minute, // 600000 ms
				Stdout:   "Long sleep test",
				ExitCode: 0,
			}

			scriptPath := GetScriptPath(tempDir, "long-sleep")
			err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
			require.NoError(t, err)

			content, err := os.ReadFile(scriptPath)
			require.NoError(t, err)

			scriptContent := string(content)
			assert.Contains(t, scriptContent, "Start-Sleep -Milliseconds 600000")
			assert.NotContains(t, scriptContent, "600000echo")
			assert.NotContains(t, scriptContent, "600000Long")
		})
	})

	t.Run("PowerShell_Backwards_Compatibility", func(t *testing.T) {
		// Ensure the fix doesn't break existing functionality
		tempDir := t.TempDir()

		// Test the exact scenario from the original bug report
		opts := ScriptOptions{
			Sleep:    50 * time.Millisecond, // The problematic value
			Stdout:   "Mock porch completed successfully",
			ExitCode: 0,
		}

		scriptPath := GetScriptPath(tempDir, "backwards-compat")
		err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
		require.NoError(t, err)

		content, err := os.ReadFile(scriptPath)
		require.NoError(t, err)
		scriptContent := string(content)

		// Verify the exact fix for the reported issue
		assert.NotContains(t, scriptContent, "50echo",
			"The exact issue '50echo' should be fixed")
		assert.NotContains(t, scriptContent, "Milliseconds 50Mock",
			"No concatenation with stdout message")

		// Verify correct structure
		assert.Contains(t, scriptContent, "powershell -NoProfile -Command \"Start-Sleep -Milliseconds 50\"")
		assert.Contains(t, scriptContent, "echo Mock porch completed successfully")

		// Test execution to ensure it works
		cmd := exec.Command("cmd.exe", "/C", scriptPath)
		output, err := cmd.CombinedOutput()

		if err != nil {
			t.Logf("Execution error (may be expected): %v", err)
		}

		outputStr := string(output)
		t.Logf("Backwards compatibility test output: %s", outputStr)

		// The key assertion: no PowerShell parameter binding error
		assert.NotContains(t, outputStr, "Cannot bind parameter 'Milliseconds'",
			"Should not have the original parameter binding error")
		assert.NotContains(t, outputStr, "cannot convert value '50echo'",
			"Should not have the original conversion error")
	})
}

// TestPowerShellRegressionPrevention ensures that specific regression patterns cannot occur
func TestPowerShellRegressionPrevention(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell regression tests only apply to Windows")
	}

	// List of known problematic patterns from CI failures
	problematicPatterns := []struct {
		name        string
		sleep       time.Duration
		stdout      string
		badPatterns []string
	}{
		{
			name:   "original_50echo_issue",
			sleep:  50 * time.Millisecond,
			stdout: "echo test",
			badPatterns: []string{
				"50echo",
				"Milliseconds 50echo",
				"50\"echo",
			},
		},
		{
			name:   "other_numeric_concatenation",
			sleep:  100 * time.Millisecond,
			stdout: "output message",
			badPatterns: []string{
				"100output",
				"100\"output",
				"Milliseconds 100output",
			},
		},
		{
			name:   "second_conversion_issues",
			sleep:  2 * time.Second,
			stdout: "completion message",
			badPatterns: []string{
				"2000completion",
				"2000\"completion",
				"Milliseconds 2000completion",
			},
		},
	}

	tempDir := t.TempDir()

	for _, tc := range problematicPatterns {
		t.Run(tc.name, func(t *testing.T) {
			opts := ScriptOptions{
				Sleep:    tc.sleep,
				Stdout:   tc.stdout,
				ExitCode: 0,
			}

			scriptPath := GetScriptPath(tempDir, "regression-"+tc.name)
			err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
			require.NoError(t, err)

			content, err := os.ReadFile(scriptPath)
			require.NoError(t, err)
			scriptContent := string(content)

			// Verify none of the problematic patterns exist
			for _, badPattern := range tc.badPatterns {
				assert.NotContains(t, scriptContent, badPattern,
					"Regression pattern should not exist: %s", badPattern)
			}

			// Verify correct patterns do exist
			expectedMs := int(tc.sleep.Milliseconds())
			expectedPSCmd := fmt.Sprintf("Start-Sleep -Milliseconds %d", expectedMs)
			assert.Contains(t, scriptContent, expectedPSCmd,
				"Correct PowerShell command should exist")
		})
	}
}
