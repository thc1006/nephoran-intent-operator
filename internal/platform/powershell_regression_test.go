package platform

import (
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPowerShellCommandRegression validates the fix for PowerShell command separation issues
// This test specifically targets the issue: "Start-Sleep : Cannot bind parameter 'Milliseconds'. 
// Cannot convert value '50echo' to type 'System.Int32'"
func TestPowerShellCommandRegression(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("PowerShell regression test only applies to Windows")
	}

	t.Run("powershell_noprofile_flag", func(t *testing.T) {
		tempDir := t.TempDir()

		opts := ScriptOptions{
			Sleep:    50 * time.Millisecond,
			Stdout:   "PowerShell test output",
			ExitCode: 0,
		}

		scriptPath := GetScriptPath(tempDir, "powershell-test")
		err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
		require.NoError(t, err)

		content, err := os.ReadFile(scriptPath)
		require.NoError(t, err)
		scriptContent := string(content)

		// Verify the -NoProfile flag is present for better PowerShell performance
		assert.Contains(t, scriptContent, "powershell -NoProfile -Command", 
			"PowerShell should use -NoProfile flag for consistency and performance")

		// Verify the Start-Sleep command is properly formatted
		assert.Contains(t, scriptContent, "Start-Sleep -Milliseconds 50",
			"Start-Sleep command should be properly formatted")

		// Ensure no command concatenation occurs
		assert.NotContains(t, scriptContent, "50PowerShell",
			"Should not contain command concatenation with PowerShell")
		assert.NotContains(t, scriptContent, "50echo",
			"Should not contain the problematic '50echo' concatenation")
	})

	t.Run("powershell_parameter_binding_fix", func(t *testing.T) {
		tempDir := t.TempDir()

		// Test various sleep durations to ensure robustness
		testCases := []struct {
			name     string
			sleep    time.Duration
			expected string
		}{
			{"short_sleep", 10 * time.Millisecond, "10"},
			{"medium_sleep", 500 * time.Millisecond, "500"},
			{"long_sleep", 2 * time.Second, "2000"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				opts := ScriptOptions{
					Sleep:    tc.sleep,
					Stdout:   "Test output",
					ExitCode: 0,
				}

				scriptPath := GetScriptPath(tempDir, "param-test-"+tc.name)
				err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
				require.NoError(t, err)

				content, err := os.ReadFile(scriptPath)
				require.NoError(t, err)
				scriptContent := string(content)

				// Verify the exact PowerShell command format
				expectedCmd := "powershell -NoProfile -Command \"Start-Sleep -Milliseconds " + tc.expected + "\""
				assert.Contains(t, scriptContent, expectedCmd,
					"PowerShell command should be properly formatted with correct milliseconds")

				// Ensure proper command termination and separation
				lines := strings.Split(scriptContent, "\n")
				for _, line := range lines {
					if strings.Contains(line, "Start-Sleep") {
						trimmed := strings.TrimSpace(line)
						// Verify line doesn't contain any concatenated commands
						assert.False(t, strings.Contains(trimmed, "echo") && strings.Contains(trimmed, "Start-Sleep"),
							"Start-Sleep and echo should not be on the same line")
					}
				}
			})
		}
	})

	t.Run("powershell_execution_validation", func(t *testing.T) {
		// This test validates that the generated PowerShell commands can actually execute
		// without parameter binding errors
		tempDir := t.TempDir()

		opts := ScriptOptions{
			Sleep:    25 * time.Millisecond,
			Stdout:   "Execution validation test",
			ExitCode: 0,
		}

		scriptPath := GetScriptPath(tempDir, "exec-validation")
		err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
		require.NoError(t, err)

		// Execute the script to ensure no PowerShell errors occur
		cmd := exec.Command("cmd.exe", "/C", scriptPath)
		output, err := cmd.CombinedOutput()
		
		// Log output for debugging
		t.Logf("Script execution output: %s", string(output))
		
		if err != nil {
			t.Logf("Script execution error: %v", err)
		}

		// Check that the specific PowerShell parameter binding error doesn't occur
		outputStr := string(output)
		assert.NotContains(t, outputStr, "cannot convert value", 
			"Should not have PowerShell parameter conversion errors")
		assert.NotContains(t, outputStr, "Cannot bind parameter", 
			"Should not have PowerShell parameter binding errors")
		assert.NotContains(t, outputStr, "Int32", 
			"Should not have Int32 type conversion errors")
		
		// Script should complete successfully
		if err != nil {
			// If there's an error, it shouldn't be related to PowerShell parameter issues
			assert.NotContains(t, err.Error(), "50echo", 
				"Error should not be related to command concatenation")
		}
	})

	t.Run("multiple_powershell_commands_robustness", func(t *testing.T) {
		tempDir := t.TempDir()

		// Test script with multiple potential command interactions
		opts := ScriptOptions{
			Sleep:  75 * time.Millisecond,
			Stdout: "Multiple commands test",
			Stderr: "Error output test",
			CustomCommands: struct {
				Windows []string
				Unix    []string
			}{
				Windows: []string{
					"echo Custom command 1",
					"echo Custom command 2",
				},
			},
			ExitCode: 0,
		}

		scriptPath := GetScriptPath(tempDir, "multi-cmd-test")
		err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
		require.NoError(t, err)

		content, err := os.ReadFile(scriptPath)
		require.NoError(t, err)
		scriptContent := string(content)

		// Verify all commands are properly separated
		assert.Contains(t, scriptContent, "powershell -NoProfile -Command \"Start-Sleep -Milliseconds 75\"")
		assert.Contains(t, scriptContent, "echo Multiple commands test")
		assert.Contains(t, scriptContent, "echo Error output test >&2")
		assert.Contains(t, scriptContent, "echo Custom command 1")
		assert.Contains(t, scriptContent, "echo Custom command 2")

		// Ensure no command concatenation anywhere in the script
		assert.NotContains(t, scriptContent, "75echo", "No concatenation between sleep and echo")
		assert.NotContains(t, scriptContent, "75Custom", "No concatenation between sleep and custom commands")
		assert.NotContains(t, scriptContent, "testecho", "No concatenation between strings and echo")

		// Verify that each major command type is on separate lines
		lines := strings.Split(scriptContent, "\n")
		var powershellLines, echoLines int
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "powershell -NoProfile") {
				powershellLines++
				// PowerShell line should not contain other commands
				assert.False(t, strings.Contains(trimmed, "echo"),
					"PowerShell line should not contain echo commands")
			}
			if strings.Contains(trimmed, "echo") && !strings.Contains(trimmed, "REM") {
				echoLines++
			}
		}

		assert.Equal(t, 1, powershellLines, "Should have exactly one PowerShell command line")
		assert.GreaterOrEqual(t, echoLines, 4, "Should have multiple echo command lines")
	})
}