package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

// ScriptType represents the type of script to create
type ScriptType int

const (
	// MockPorchScript represents a mock porch executable script
	MockPorchScript ScriptType = iota
	// GenericScript represents a generic executable script
	GenericScript
)

// ScriptOptions configures script generation behavior
type ScriptOptions struct {
	// ExitCode is the exit code the script should return
	ExitCode int
	// Stdout is the output the script should write to stdout
	Stdout string
	// Stderr is the output the script should write to stderr
	Stderr string
	// Sleep is the duration the script should sleep before executing
	Sleep time.Duration
	// CustomCommands allows providing platform-specific custom commands
	CustomCommands struct {
		Windows []string // Windows batch commands
		Unix    []string // Unix shell commands
	}
	// FailOnPattern will cause the script to exit with code 1 if input contains this pattern
	FailOnPattern string
}

// GetScriptExtension returns the appropriate script file extension for the current OS
func GetScriptExtension() string {
	if runtime.GOOS == "windows" {
		return ".bat"
	}
	return ".sh"
}

// GetScriptPath returns the appropriate script path for the current OS
func GetScriptPath(dir, baseName string) string {
	return filepath.Join(dir, baseName+GetScriptExtension())
}

// GetExecutablePath returns the path to an executable with proper extension handling
func GetExecutablePath(dir, baseName string) string {
	if runtime.GOOS == "windows" {
		// Check for .exe, .bat, .cmd extensions
		for _, ext := range []string{".exe", ".bat", ".cmd"} {
			path := filepath.Join(dir, baseName+ext)
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
		// Default to .exe if none found
		return filepath.Join(dir, baseName+".exe")
	}
	// Unix systems - no extension needed
	return filepath.Join(dir, baseName)
}

// CreateCrossPlatformScript creates a script that works on both Windows and Unix systems
func CreateCrossPlatformScript(scriptPath string, scriptType ScriptType, opts ScriptOptions) error {
	var script string
	var fileMode os.FileMode = 0755

	if runtime.GOOS == "windows" {
		script = generateWindowsScript(scriptType, opts)
		fileMode = 0644 // Windows doesn't use Unix permissions
	} else {
		script = generateUnixScript(scriptType, opts)
	}

	return os.WriteFile(scriptPath, []byte(script), fileMode)
}

// generateWindowsScript creates Windows batch script content
func generateWindowsScript(scriptType ScriptType, opts ScriptOptions) string {
	var script string

	switch scriptType {
	case MockPorchScript:
		script = generateWindowsMockPorchScript(opts)
	case GenericScript:
		script = generateWindowsGenericScript(opts)
	}

	return script
}

// generateUnixScript creates Unix shell script content
func generateUnixScript(scriptType ScriptType, opts ScriptOptions) string {
	var script string

	switch scriptType {
	case MockPorchScript:
		script = generateUnixMockPorchScript(opts)
	case GenericScript:
		script = generateUnixGenericScript(opts)
	}

	return script
}

// generateWindowsMockPorchScript creates a Windows batch file that mimics porch behavior
func generateWindowsMockPorchScript(opts ScriptOptions) string {
	sleepCmd := ""
	if opts.Sleep > 0 {
		// Use powershell Start-Sleep for more precise timing on Windows
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
    echo Error: Pattern '%s' found in input file >&2
    exit /b 1
)`, opts.FailOnPattern, opts.FailOnPattern)
	}

	customCmds := ""
	for _, cmd := range opts.CustomCommands.Windows {
		customCmds += cmd + "\n"
	}

	return fmt.Sprintf(`@echo off
setlocal enabledelayedexpansion

REM Mock Porch Script - Windows Version
REM Arguments: %%1=command %%2=intent-file %%3=flag %%4=output-dir

if "%%1"=="--help" (
    echo Mock porch help - Windows version
    echo Usage: mock-porch [command] [intent-file] [flags] [output-dir]
    exit /b 0
)

REM Check for pattern failure condition
%s

REM Sleep if requested
%s

REM Custom commands
%s

REM Output messages
%s
%s

REM Mock successful processing
if not "%%2"=="" (
    echo Mock porch processing intent file: %%2
)
if not "%%4"=="" (
    echo Output directory: %%4
    REM Create a mock output file
    if not exist "%%4" mkdir "%%4"
    echo apiVersion: v1 > "%%4\mock-output.yaml"
    echo kind: ConfigMap >> "%%4\mock-output.yaml"
    echo metadata: >> "%%4\mock-output.yaml"
    echo   name: mock-porch-output >> "%%4\mock-output.yaml"
    echo Processing completed successfully
)

%s
%s
%s
%s
%s
exit /b %d`, failOnPatternCmd, sleepCmd, customCmds, stdoutCmd, stderrCmd, opts.ExitCode)
}

// generateUnixMockPorchScript creates a Unix shell script that mimics porch behavior
func generateUnixMockPorchScript(opts ScriptOptions) string {
	sleepCmd := ""
	if opts.Sleep > 0 {
		sleepCmd = fmt.Sprintf("sleep %v", opts.Sleep.Seconds())
	}

	stdoutCmd := ""
	if opts.Stdout != "" {
		stdoutCmd = fmt.Sprintf("echo \"%s\"", opts.Stdout)
	}

	stderrCmd := ""
	if opts.Stderr != "" {
		stderrCmd = fmt.Sprintf("echo \"%s\" >&2", opts.Stderr)
	}

	failOnPatternCmd := ""
	if opts.FailOnPattern != "" {
		failOnPatternCmd = fmt.Sprintf(`if grep -q "%s" "$2" 2>/dev/null; then
    echo "Error: Pattern '%s' found in input file" >&2
    exit 1
fi`, opts.FailOnPattern, opts.FailOnPattern)
	}

	customCmds := ""
	for _, cmd := range opts.CustomCommands.Unix {
		customCmds += cmd + "\n"
	}

	return fmt.Sprintf(`#!/bin/bash
set -euo pipefail

# Mock Porch Script - Unix Version
# Arguments: $1=command $2=intent-file $3=flag $4=output-dir

if [ "${1:-}" = "--help" ]; then
    echo "Mock porch help - Unix version"
    echo "Usage: mock-porch [command] [intent-file] [flags] [output-dir]"
    exit 0
fi

# Check for pattern failure condition
%s

# Sleep if requested
%s

# Custom commands
%s

# Output messages
%s
%s

# Mock successful processing
if [ -n "${2:-}" ]; then
    echo "Mock porch processing intent file: $2"
fi

if [ -n "${4:-}" ]; then
    echo "Output directory: $4"
    # Create a mock output file
    mkdir -p "$4"
    cat > "$4/mock-output.yaml" << EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: mock-porch-output
EOF
    echo "Processing completed successfully"
fi

exit %d`, failOnPatternCmd, sleepCmd, customCmds, stdoutCmd, stderrCmd, opts.ExitCode)
}

// generateWindowsGenericScript creates a generic Windows batch script
func generateWindowsGenericScript(opts ScriptOptions) string {
	sleepCmd := ""
	if opts.Sleep > 0 {
		sleepCmd = fmt.Sprintf("timeout /t %d /nobreak >nul 2>nul", int(opts.Sleep.Seconds()))
	}

	stdoutCmd := ""
	if opts.Stdout != "" {
		stdoutCmd = fmt.Sprintf("echo %s", opts.Stdout)
	}

	stderrCmd := ""
	if opts.Stderr != "" {
		stderrCmd = fmt.Sprintf("echo %s >&2", opts.Stderr)
	}

	customCmds := ""
	for _, cmd := range opts.CustomCommands.Windows {
		customCmds += cmd + "\n"
	}

	return fmt.Sprintf(`@echo off
%s
%s
%s
%s
exit /b %d`, sleepCmd, customCmds, stdoutCmd, stderrCmd, opts.ExitCode)
}

// generateUnixGenericScript creates a generic Unix shell script
func generateUnixGenericScript(opts ScriptOptions) string {
	sleepCmd := ""
	if opts.Sleep > 0 {
		sleepCmd = fmt.Sprintf("sleep %v", opts.Sleep.Seconds())
	}

	stdoutCmd := ""
	if opts.Stdout != "" {
		stdoutCmd = fmt.Sprintf("echo \"%s\"", opts.Stdout)
	}

	stderrCmd := ""
	if opts.Stderr != "" {
		stderrCmd = fmt.Sprintf("echo \"%s\" >&2", opts.Stderr)
	}

	customCmds := ""
	for _, cmd := range opts.CustomCommands.Unix {
		customCmds += cmd + "\n"
	}

	return fmt.Sprintf(`#!/bin/bash
%s
%s
%s
%s
exit %d`, sleepCmd, customCmds, stdoutCmd, stderrCmd, opts.ExitCode)
}

// IsExecutable checks if a file is executable on the current platform
func IsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		// On Windows, check file extension
		ext := filepath.Ext(path)
		return ext == ".exe" || ext == ".bat" || ext == ".cmd"
	}

	// On Unix, check executable permission
	return info.Mode()&0111 != 0
}

// MakeExecutable ensures a file is executable on the current platform
func MakeExecutable(path string) error {
	if runtime.GOOS == "windows" {
		// On Windows, permissions are handled differently
		// Ensure file exists and is readable
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Mode()&0200 == 0 {
			return os.Chmod(path, 0644)
		}
		return nil
	}

	// On Unix, set executable permission
	return os.Chmod(path, 0755)
}