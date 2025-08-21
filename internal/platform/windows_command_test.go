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

// TestWindowsCommandSeparation tests that Windows PowerShell commands are properly separated
// This prevents the "Start-Sleep -Milliseconds 50echo" concatenation error
func TestWindowsCommandSeparation(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	tempDir := t.TempDir()

	// Test case that previously caused "50echo" concatenation
	opts := ScriptOptions{
		Sleep:    50 * time.Millisecond,
		Stdout:   "Mock porch completed",
		ExitCode: 0,
	}

	scriptPath := GetScriptPath(tempDir, "test-separation")
	err := CreateCrossPlatformScript(scriptPath, MockPorchScript, opts)
	require.NoError(t, err)

	// Read the generated script content
	content, err := os.ReadFile(scriptPath)
	require.NoError(t, err)
	
	scriptText := string(content)
	t.Logf("Generated script content:\n%s", scriptText)

	// CRITICAL: Verify that "50echo" concatenation does not occur
	assert.NotContains(t, scriptText, "50echo", "Script should not contain '50echo' concatenation")
	assert.NotContains(t, scriptText, "50Mock", "Script should not contain command concatenation")

	// Verify proper command separation (each command on its own line)
	lines := strings.Split(scriptText, "\n")
	var powershellLine, echoLine string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "powershell -command") {
			powershellLine = trimmed
		}
		if strings.HasPrefix(trimmed, "echo Mock porch completed") {
			echoLine = trimmed
		}
	}

	assert.NotEmpty(t, powershellLine, "Should have PowerShell Start-Sleep command")
	assert.NotEmpty(t, echoLine, "Should have echo command")
	assert.Contains(t, powershellLine, "Start-Sleep -Milliseconds 50", "PowerShell command should be properly formatted")

	// Functional test: Execute the script and verify it works
	cmd := exec.Command("cmd.exe", "/C", scriptPath)
	output, err := cmd.CombinedOutput()
	
	// The script should execute without PowerShell parameter binding errors
	if err != nil {
		t.Logf("Script execution error: %v", err)
		t.Logf("Script output: %s", string(output))
		// Check for the specific error that was occurring
		assert.NotContains(t, string(output), "cannot convert value '50echo'", 
			"Should not have PowerShell parameter binding errors")
	}
}