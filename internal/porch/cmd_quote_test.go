package porch

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCmdSafeQuoteCrossPlatform ensures cmdSafeQuote compiles and behaves consistently across platforms
func TestCmdSafeQuoteCrossPlatform(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "simple_command",
			args: []string{"echo", "hello"},
		},
		{
			name: "command_with_spaces",
			args: []string{"echo", "hello world"},
		},
		{
			name: "empty_args",
			args: []string{},
		},
		{
			name: "single_arg",
			args: []string{"ls"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test verifies that cmdSafeQuote compiles and runs on all platforms
			result := cmdSafeQuote(tt.args)
			
			// Basic sanity checks
			if len(tt.args) == 0 {
				assert.Equal(t, "", result, "Empty args should return empty string")
			} else {
				assert.NotEmpty(t, result, "Non-empty args should produce output")
			}
			
			// Log the platform-specific behavior
			t.Logf("Platform: %s, Args: %v, Result: %q", runtime.GOOS, tt.args, result)
		})
	}
}

// TestCmdSafeQuoteWindowsBehavior tests Windows-specific quoting behavior
func TestCmdSafeQuoteWindowsBehavior(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}
	
	// Test that special characters are properly handled on Windows
	args := []string{"echo", "test&echo"}
	result := cmdSafeQuote(args)
	
	// On Windows, this should produce escaped output
	assert.Contains(t, result, "test", "Should contain the test command")
	assert.NotContains(t, result, "test&echo", "Ampersand should be escaped")
}

// TestCmdSafeQuoteUnixBehavior tests Unix-specific behavior
func TestCmdSafeQuoteUnixBehavior(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix-specific test")
	}
	
	// Test that on Unix, the function is essentially a passthrough
	args := []string{"echo", "test&echo"}
	result := cmdSafeQuote(args)
	
	// On Unix, this should just join with spaces
	assert.Equal(t, "echo test&echo", result, "Unix should just join args")
}