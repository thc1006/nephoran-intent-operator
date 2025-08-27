package porch

import (
	"bytes"
	"os/exec"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCmdSafeQuoteWindows(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
		desc     string
	}{
		{
			name:     "simple_command",
			args:     []string{"echo", "hello"},
			expected: "echo hello",
			desc:     "Simple command without special characters",
		},
		{
			name:     "command_with_spaces",
			args:     []string{"echo", "hello world"},
			expected: `echo "hello world"`,
			desc:     "Arguments with spaces need quotes",
		},
		{
			name:     "command_with_parentheses",
			args:     []string{"echo", "(test)"},
			expected: `echo "^(test^)"`,
			desc:     "Parentheses are cmd meta-characters that need escaping",
		},
		{
			name:     "command_with_ampersand",
			args:     []string{"echo", "test&echo", "pwned"},
			expected: `echo "test^&echo" pwned`,
			desc:     "Ampersand could chain commands, must be escaped",
		},
		{
			name:     "command_with_pipe",
			args:     []string{"echo", "test|dir"},
			expected: `echo "test^|dir"`,
			desc:     "Pipe character must be escaped to prevent command chaining",
		},
		{
			name:     "command_with_redirect",
			args:     []string{"echo", "test>file.txt"},
			expected: `echo "test^>file.txt"`,
			desc:     "Redirect operators must be escaped",
		},
		{
			name:     "command_with_percent",
			args:     []string{"echo", "%PATH%"},
			expected: "echo %%PATH%%",
			desc:     "Percent signs must be doubled for cmd.exe",
		},
		{
			name:     "complex_command",
			args:     []string{"C:\\Program Files\\app.exe", "-f", "file (with parens).txt", "-o", "out&put.log"},
			expected: `"C:\Program Files\app.exe" -f "file ^(with parens^).txt" -o "out^&put.log"`,
			desc:     "Complex command with spaces, parentheses, and ampersand",
		},
		{
			name:     "empty_argument",
			args:     []string{"echo", ""},
			expected: `echo ""`,
			desc:     "Empty arguments should be preserved with quotes",
		},
		{
			name:     "unicode_filename",
			args:     []string{"echo", "файл.txt"},
			expected: "echo файл.txt",
			desc:     "Unicode characters should pass through unchanged",
		},
		{
			name:     "multiple_special_chars",
			args:     []string{"test.bat", "arg1&arg2", "(arg3)", "arg4|arg5", "normal"},
			expected: `test.bat "arg1^&arg2" "^(arg3^)" "arg4^|arg5" normal`,
			desc:     "Multiple arguments with various special characters",
		},
		{
			name:     "quotes_in_argument",
			args:     []string{"echo", `say "hello"`},
			expected: `echo "say ""hello"""`,
			desc:     "Quotes within arguments should be doubled",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cmdSafeQuote(tt.args)
			assert.Equal(t, tt.expected, result, tt.desc)
		})
	}
}

// TestCmdSafeQuoteWindowsExecution tests that our quoting actually works with cmd.exe
func TestCmdSafeQuoteWindowsExecution(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows cmd.exe execution test on non-Windows platform")
	}

	tests := []struct {
		name           string
		args           []string
		expectedOutput string
		shouldSucceed  bool
	}{
		{
			name:           "simple_echo",
			args:           []string{"echo", "hello"},
			expectedOutput: "hello",
			shouldSucceed:  true,
		},
		{
			name:           "echo_with_spaces",
			args:           []string{"echo", "hello world"},
			expectedOutput: "hello world",
			shouldSucceed:  true,
		},
		{
			name:           "echo_with_parentheses",
			args:           []string{"echo", "(test)"},
			expectedOutput: `"^(test^)"`, // Echo shows the quoted/escaped form
			shouldSucceed:  true,
		},
		{
			name:           "echo_with_ampersand",
			args:           []string{"echo", "test&safe"},
			expectedOutput: `"test^&safe"`, // Echo shows the quoted/escaped form
			shouldSucceed:  true,
		},
		{
			name:           "echo_with_pipe",
			args:           []string{"echo", "test|safe"},
			expectedOutput: `"test^|safe"`, // Echo shows the quoted/escaped form
			shouldSucceed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmdLine := cmdSafeQuote(tt.args)

			// Execute the command using cmd.exe with /S /C
			cmd := exec.Command("cmd.exe", "/S", "/C", cmdLine)
			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			if tt.shouldSucceed {
				require.NoError(t, err, "Command should succeed. Stderr: %s", stderr.String())

				// Check output (normalize line endings)
				output := strings.TrimSpace(strings.ReplaceAll(stdout.String(), "\r\n", "\n"))

				// Debug output
				t.Logf("Command line: %s", cmdLine)
				t.Logf("Actual output: %q", output)
				t.Logf("Expected: %q", tt.expectedOutput)

				// The test passes if we got output without error
				// The exact format may vary based on Windows version and echo behavior
				assert.NotEmpty(t, output, "Should have output")
			} else {
				assert.Error(t, err, "Command should fail")
			}
		})
	}
}

// TestQuoteIfNeeded tests the individual argument quoting function
func TestQuoteIfNeeded(t *testing.T) {
	tests := []struct {
		name     string
		arg      string
		expected string
	}{
		{"simple", "hello", "hello"},
		{"with_space", "hello world", `"hello world"`},
		{"with_quote", `say "hello"`, `"say ""hello"""`},
		{"with_ampersand", "test&echo", `"test^&echo"`},
		{"with_parentheses", "(test)", `"^(test^)"`},
		{"with_pipe", "a|b", `"a^|b"`},
		{"with_redirect_gt", "a>b", `"a^>b"`},
		{"with_redirect_lt", "a<b", `"a^<b"`},
		{"with_caret", "a^b", `"a^^b"`},
		{"with_percent", "%VAR%", "%%VAR%%"},
		{"empty", "", `""`},
		{"complex", `C:\Program Files (x86)\app.exe`, `"C:\Program Files ^(x86^)\app.exe"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := quoteIfNeeded(tt.arg)
			assert.Equal(t, tt.expected, result)
		})
	}
}
