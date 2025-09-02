package platform

import (
	"os"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBatCommandSeparation validates that BAT script commands are properly separated
// to prevent concatenation issues like '50echo' instead of proper command separation.
// DISABLED: func TestBatCommandSeparation(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("BAT script test only applies to Windows")
	}

	t.Run("sleep_and_echo_commands_separated", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "bat_command_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a script with both sleep and echo commands
		opts := ScriptOptions{
			Sleep:    50 * time.Millisecond,
			Stdout:   "Mock porch completed",
			ExitCode: 0,
		}

		scriptPath := GetScriptPath(tempDir, "test-mock")
		err = CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
		require.NoError(t, err)

		// Read the generated script content
		content, err := os.ReadFile(scriptPath)
		require.NoError(t, err)
		scriptContent := string(content)

		// Validate that commands are properly separated
		t.Logf("Generated script content:\n%s", scriptContent)

		// Check that Start-Sleep command exists and is properly terminated
		assert.Contains(t, scriptContent, "Start-Sleep -Milliseconds 50",
			"Should contain the PowerShell Start-Sleep command")

		// Verify that commands don't get concatenated like '50echo'
		assert.NotContains(t, scriptContent, "50echo",
			"Commands should not be concatenated without proper separation")
		assert.NotContains(t, scriptContent, "Milliseconds 50echo",
			"Sleep and echo commands should be properly separated")

		// Check that each command is on its own line or properly terminated
		lines := strings.Split(scriptContent, "\n")
		var sleepLineFound, echoLineFound bool
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if strings.Contains(trimmed, "Start-Sleep -Milliseconds 50") {
				sleepLineFound = true
				// Verify the line doesn't continue with other commands
				assert.Equal(t, "powershell -NoProfile -Command \"Start-Sleep -Milliseconds 50\"", trimmed,
					"Start-Sleep command should be on its own line")
			}
			if strings.Contains(trimmed, "echo Mock porch completed") {
				echoLineFound = true
			}
		}

		assert.True(t, sleepLineFound, "Start-Sleep command line should be found")
		assert.True(t, echoLineFound, "Echo command line should be found")
	})

	t.Run("multiple_commands_with_stderr", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "bat_multi_command_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a script with sleep, stdout, and stderr
		opts := ScriptOptions{
			Sleep:    100 * time.Millisecond,
			Stdout:   "Success message",
			Stderr:   "Error message",
			ExitCode: 1,
		}

		scriptPath := GetScriptPath(tempDir, "test-multi")
		err = CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
		require.NoError(t, err)

		content, err := os.ReadFile(scriptPath)
		require.NoError(t, err)
		scriptContent := string(content)

		t.Logf("Multi-command script:\n%s", scriptContent)

		// Validate no command concatenation
		assert.NotContains(t, scriptContent, "100echo", "No concatenation between sleep and echo")
		assert.NotContains(t, scriptContent, "message>&2", "No concatenation in stderr redirection")

		// Ensure each command type is present
		assert.Contains(t, scriptContent, "Start-Sleep -Milliseconds 100")
		assert.Contains(t, scriptContent, "echo Success message")
		assert.Contains(t, scriptContent, "echo Error message >&2")
	})

	t.Run("generic_script_command_separation", func(t *testing.T) {
		tempDir, err := os.MkdirTemp("", "generic_script_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Test generic script (uses timeout instead of PowerShell)
		opts := ScriptOptions{
			Sleep:    2 * time.Second,
			Stdout:   "Generic output",
			ExitCode: 0,
		}

		scriptPath := GetScriptPath(tempDir, "test-generic")
		err = CreateCrossPlatformScript(scriptPath, GenericScript, opts)
		require.NoError(t, err)

		content, err := os.ReadFile(scriptPath)
		require.NoError(t, err)
		scriptContent := string(content)

		t.Logf("Generic script content:\n%s", scriptContent)

		// For generic scripts, it uses timeout instead of PowerShell
		assert.Contains(t, scriptContent, "timeout /t 2")
		assert.Contains(t, scriptContent, "echo Generic output")

		// Ensure no command concatenation
		assert.NotContains(t, scriptContent, "2echo", "Timeout and echo should be separated")
	})

	t.Run("regression_test_powershell_parameter_parsing", func(t *testing.T) {
		// This test specifically validates the original issue:
		// "Start-Sleep : Cannot bind parameter 'Milliseconds' ... cannot convert value '50echo' to type 'Int32'."

		tempDir, err := os.MkdirTemp("", "regression_test")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		opts := ScriptOptions{
			Sleep:    50 * time.Millisecond,
			Stdout:   "Regression test output",
			ExitCode: 0,
		}

		scriptPath := GetScriptPath(tempDir, "regression-test")
		err = CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
		require.NoError(t, err)

		content, err := os.ReadFile(scriptPath)
		require.NoError(t, err)
		scriptContent := string(content)

		// The exact pattern that was causing the issue
		problematicPattern := "50echo"
		assert.NotContains(t, scriptContent, problematicPattern,
			"The '50echo' concatenation that caused PowerShell parameter binding error should not exist")

		// Verify correct structure instead
		assert.Contains(t, scriptContent, "Start-Sleep -Milliseconds 50")
		assert.Contains(t, scriptContent, "echo Regression test output")

		// Ensure the PowerShell command is properly quoted and terminated
		lines := strings.Split(scriptContent, "\n")
		for _, line := range lines {
			if strings.Contains(line, "Start-Sleep") {
				// Should be properly terminated and not continue with other commands
				trimmed := strings.TrimSpace(line)
				assert.True(t, strings.HasPrefix(trimmed, "powershell -NoProfile -Command \"Start-Sleep"),
					"PowerShell command should be properly formatted")
				assert.True(t, strings.HasSuffix(trimmed, "\""),
					"PowerShell command should be properly closed")
			}
		}
	})
}
