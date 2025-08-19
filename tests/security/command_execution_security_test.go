package security

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestBinaryPathValidation ensures only whitelisted executables can be used
func TestBinaryPathValidation(t *testing.T) {
	whitelistedBinaries := []string{
		"/usr/bin/kubectl",
		"/usr/local/bin/kpt",
		"/usr/bin/git",
	}

	maliciousBinaries := []string{
		"/bin/bash",
		"/usr/bin/curl",
		"/bin/sh",
		"/usr/bin/wget",
	}

	for _, binary := range whitelistedBinaries {
		t.Run(fmt.Sprintf("Validate Whitelisted Binary: %s", binary), func(t *testing.T) {
			err := validateBinaryPath(binary)
			assert.NoError(t, err, "Whitelisted binary should be allowed")
		})
	}

	for _, binary := range maliciousBinaries {
		t.Run(fmt.Sprintf("Reject Malicious Binary: %s", binary), func(t *testing.T) {
			err := validateBinaryPath(binary)
			assert.Error(t, err, "Non-whitelisted binary should be rejected")
		})
	}
}

// TestSecureCommandExecution validates command execution with security controls
func TestSecureCommandExecution(t *testing.T) {
	testCases := []struct {
		name                string
		command             string
		args                []string
		expectedError       bool
		expectedTimeoutSecs int
	}{
		{
			name:                "Safe Kubectl Command",
			command:             "/usr/bin/kubectl",
			args:                []string{"version", "--client"},
			expectedError:       false,
			expectedTimeoutSecs: 10,
		},
		{
			name:                "Shell Injection Attempt",
			command:             "/bin/bash",
			args:                []string{"-c", "rm -rf /"},
			expectedError:       true,
			expectedTimeoutSecs: 5,
		},
		{
			name:                "Command with Excessive Arguments",
			command:             "/usr/bin/git",
			args:                generateLongArgumentList(100),
			expectedError:       true,
			expectedTimeoutSecs: 15,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 
				time.Duration(tc.expectedTimeoutSecs)*time.Second)
			defer cancel()

			result, err := executeSecureCommand(ctx, tc.command, tc.args)

			if tc.expectedError {
				assert.Error(t, err, "Expected command to be rejected")
			} else {
				assert.NoError(t, err, "Expected safe command to execute")
				assert.NotEmpty(t, result, "Expected command output")
			}
		})
	}
}

// Simulate implementation of secure execution functions
func validateBinaryPath(binaryPath string) error {
	allowedPrefixes := []string{
		"/usr/bin/kubectl",
		"/usr/local/bin/kpt",
		"/usr/bin/git",
	}

	for _, prefix := range allowedPrefixes {
		if strings.HasPrefix(binaryPath, prefix) {
			return nil
		}
	}

	return fmt.Errorf("binary path not in whitelist: %s", binaryPath)
}

func executeSecureCommand(ctx context.Context, command string, args []string) (string, error) {
	// Validate binary path
	err := validateBinaryPath(command)
	if err != nil {
		return "", err
	}

	// Prevent excessive arguments
	if len(args) > 50 {
		return "", fmt.Errorf("too many arguments: %d", len(args))
	}

	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()

	if ctx.Err() == context.DeadlineExceeded {
		return "", fmt.Errorf("command execution timed out")
	}

	if err != nil {
		return "", fmt.Errorf("command execution failed: %v", err)
	}

	return string(output), nil
}

// Helper function to generate long argument lists
func generateLongArgumentList(length int) []string {
	args := make([]string, length)
	for i := 0; i < length; i++ {
		args[i] = fmt.Sprintf("arg_%d", i)
	}
	return args
}