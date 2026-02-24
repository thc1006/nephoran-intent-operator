package porch

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/platform"
)

// CrossPlatformMockOptions configures the behavior of the cross-platform mock script.

type CrossPlatformMockOptions struct {
	// ExitCode is the exit code the mock script should return.

	ExitCode int

	// Stdout is the output the mock should write to stdout.

	Stdout string

	// Stderr is the output the mock should write to stderr.

	Stderr string

	// Sleep is the duration the mock should sleep before executing.

	Sleep time.Duration

	// FailOnPattern will cause the mock to exit with code 1 if the input contains this pattern.

	FailOnPattern string

	// CustomScript allows providing completely custom script content (overrides other options).

	CustomScript struct {
		Windows string // Windows batch content

		Unix string // Unix shell content
	}
}

// CreateCrossPlatformMock creates a mock porch script that works on both Windows and Unix systems.

// It returns the path to the created mock script.

func CreateCrossPlatformMock(tempDir string, opts CrossPlatformMockOptions) (string, error) {
	var mockPath string

	var script string

	if runtime.GOOS == "windows" {

		// Create Windows batch file.

		mockPath = filepath.Join(tempDir, "mock-porch.bat")

		// Use custom script if provided.

		if opts.CustomScript.Windows != "" {
			script = opts.CustomScript.Windows
		} else {

			sleepSeconds := int(opts.Sleep.Seconds())

			if sleepSeconds == 0 && opts.Sleep > 0 {
				sleepSeconds = 1
			}

			sleepCmd := ""

			if sleepSeconds > 0 {
				// Use powershell Start-Sleep for more precise timing on Windows.

				sleepCmd = fmt.Sprintf("powershell -command \"Start-Sleep -Milliseconds %d\"", int(opts.Sleep.Milliseconds()))
			}

			stdoutCmd := ""

			if opts.Stdout != "" {
				stdoutCmd = fmt.Sprintf("echo %s", opts.Stdout)
			}

			stderrCmd := ""

			if opts.Stderr != "" {
				stderrCmd = fmt.Sprintf("echo %s >&2", opts.Stderr)
			}

			failOnPatternCmd := ""

			if opts.FailOnPattern != "" {
				failOnPatternCmd = fmt.Sprintf(`findstr /C:"%s" "%%2" >nul 2>nul

if %%errorlevel%% equ 0 (

    echo Error: Pattern '%s' found in input file

    exit /b 1

)`, opts.FailOnPattern, opts.FailOnPattern)
			}

			script = fmt.Sprintf(`@echo off

if "%%1"=="--help" (

    echo Mock porch help

    exit /b 0

)

%s

%s

%s

%s

exit /b %d`, failOnPatternCmd, sleepCmd, stdoutCmd, stderrCmd, opts.ExitCode)

		}

	} else {

		// Create Unix shell script.

		mockPath = filepath.Join(tempDir, "mock-porch.sh")

		// Use custom script if provided.

		if opts.CustomScript.Unix != "" {
			script = opts.CustomScript.Unix
		} else {

			sleepCmd := ""

			if opts.Sleep > 0 {
				sleepCmd = fmt.Sprintf("sleep %v", opts.Sleep.Seconds())
			}

			stdoutCmd := ""

			if opts.Stdout != "" {
				stdoutCmd = fmt.Sprintf("echo %q", opts.Stdout)
			}

			stderrCmd := ""

			if opts.Stderr != "" {
				stderrCmd = fmt.Sprintf("echo %q >&2", opts.Stderr)
			}

			failOnPatternCmd := ""

			if opts.FailOnPattern != "" {
				failOnPatternCmd = fmt.Sprintf(`if grep -q "%s" "$2" 2>/dev/null; then

    echo "Error: Pattern '%s' found in input file" >&2

    exit 1

fi`, opts.FailOnPattern, opts.FailOnPattern)
			}

			script = fmt.Sprintf(`#!/bin/bash

if [ "$1" = "--help" ]; then

    echo "Mock porch help"

    exit 0

fi

%s

%s

%s

%s

exit %d`, failOnPatternCmd, sleepCmd, stdoutCmd, stderrCmd, opts.ExitCode)

		}

	}

	// Write the script file with restrictive permissions first.
	// Security: Use 0600 for initial write to satisfy G306
	if err := os.WriteFile(mockPath, []byte(script), 0o600); err != nil {
		return "", fmt.Errorf("failed to write mock script: %w", err)
	}

	// Then set appropriate permissions for the platform
	// Security: Use 0755 for executable scripts on both Unix and Windows
	// This two-step approach ensures secure initial write with restrictive permissions
	chmod := 0o755 // Both Unix and Windows need executable permissions
	if runtime.GOOS == "windows" {
		chmod = 0o755 // Windows needs executable permissions
	}
	if err := os.Chmod(mockPath, os.FileMode(chmod)); err != nil {
		return "", fmt.Errorf("failed to make mock script executable: %w", err)
	}

	return mockPath, nil
}

// CreateSimpleMock creates a basic mock script that echoes arguments and exits with code 0.

func CreateSimpleMock(tempDir string) (string, error) {
	return CreateCrossPlatformMock(tempDir, CrossPlatformMockOptions{
		ExitCode: 0,

		Stdout: "Mock porch processing completed successfully",
	})
}

// CreateAdvancedMock creates a mock script using the new platform utilities.

func CreateAdvancedMock(tempDir string, opts CrossPlatformMockOptions) (string, error) {
	// Convert to platform.ScriptOptions.

	scriptOpts := platform.ScriptOptions{
		ExitCode: opts.ExitCode,

		Stdout: opts.Stdout,

		Stderr: opts.Stderr,

		Sleep: opts.Sleep,

		FailOnPattern: opts.FailOnPattern,
	}

	// Set custom commands if provided.

	if opts.CustomScript.Windows != "" || opts.CustomScript.Unix != "" {

		if opts.CustomScript.Windows != "" {
			scriptOpts.CustomCommands.Windows = []string{opts.CustomScript.Windows}
		}

		if opts.CustomScript.Unix != "" {
			scriptOpts.CustomCommands.Unix = []string{opts.CustomScript.Unix}
		}

	}

	// Create the script using platform utilities.

	mockPath := platform.GetScriptPath(tempDir, "mock-porch")

	err := platform.CreateCrossPlatformScript(mockPath, platform.MockPorchScript, scriptOpts)
	if err != nil {
		return "", fmt.Errorf("failed to create advanced mock script: %w", err)
	}

	return mockPath, nil
}

// GetMockPorchPath returns the correct mock porch path for the current platform.

func GetMockPorchPath(tempDir string) string {
	return platform.GetScriptPath(tempDir, "mock-porch")
}

// ValidateMockExecutable ensures the mock script is executable on the current platform.

func ValidateMockExecutable(mockPath string) error {
	if !platform.IsExecutable(mockPath) {
		if err := platform.MakeExecutable(mockPath); err != nil {
			return fmt.Errorf("failed to make mock executable: %w", err)
		}
	}

	return nil
}

// CreateFailingMock creates a mock that simulates porch failures.

func CreateFailingMock(tempDir string, exitCode int, errorMessage string) (string, error) {
	return CreateCrossPlatformMock(tempDir, CrossPlatformMockOptions{
		ExitCode: exitCode,

		Stderr: errorMessage,
	})
}

// CreateSlowMock creates a mock that simulates slow porch execution.

func CreateSlowMock(tempDir string, delay time.Duration) (string, error) {
	return CreateCrossPlatformMock(tempDir, CrossPlatformMockOptions{
		ExitCode: 0,

		Sleep: delay,

		Stdout: "Slow mock porch processing completed",
	})
}

// CreatePatternFailingMock creates a mock that fails when specific patterns are found.

func CreatePatternFailingMock(tempDir, failPattern string) (string, error) {
	return CreateCrossPlatformMock(tempDir, CrossPlatformMockOptions{
		ExitCode: 1,

		FailOnPattern: failPattern,

		Stderr: "Pattern-based failure triggered",
	})
}
