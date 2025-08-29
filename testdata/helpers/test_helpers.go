// Package helpers provides common test utilities and fixtures for conductor-loop testing.

package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestFixtures provides access to test fixtures and mock executables.

type TestFixtures struct {
	IntentsDir string

	MockExecutablesDir string

	TempDir string
}

// NewTestFixtures creates a new test fixtures instance.

func NewTestFixtures(t testing.TB) *TestFixtures {

	// Get the path to the testdata directory.

	_, filename, _, ok := runtime.Caller(0)

	require.True(t, ok, "Failed to get caller information")

	testdataDir := filepath.Join(filepath.Dir(filename), "..")

	return &TestFixtures{

		IntentsDir: filepath.Join(testdataDir, "intents"),

		MockExecutablesDir: filepath.Join(testdataDir, "mock-executables"),

		TempDir: t.TempDir(),
	}

}

// GetIntentFile returns the path to a test intent file.

func (tf *TestFixtures) GetIntentFile(name string) string {

	return filepath.Join(tf.IntentsDir, name)

}

// GetMockExecutable returns the path to a mock executable with proper platform extension.

func (tf *TestFixtures) GetMockExecutable(name string) string {

	execPath := filepath.Join(tf.MockExecutablesDir, name)

	// Only add extension if name doesn't already have one.

	if filepath.Ext(name) == "" {

		if runtime.GOOS == "windows" {

			execPath += ".bat"

		} else {

			execPath += ".sh"

		}

	}

	return execPath

}

// CreateTempIntent creates a temporary intent file with the given content.

func (tf *TestFixtures) CreateTempIntent(t testing.TB, name, content string) string {

	intentPath := filepath.Join(tf.TempDir, name)

	require.NoError(t, os.WriteFile(intentPath, []byte(content), 0640))

	return intentPath

}

// CreateTempDir creates a temporary directory for testing.

func (tf *TestFixtures) CreateTempDir(t testing.TB, name string) string {

	dirPath := filepath.Join(tf.TempDir, name)

	require.NoError(t, os.MkdirAll(dirPath, 0755))

	return dirPath

}

// CommonIntents provides access to common test intent files.

type CommonIntents struct {
	ScaleUp string

	ScaleDown string

	ComplexOran string

	Invalid string

	Large string

	Empty string

	SpecialChars string
}

// GetCommonIntents returns paths to commonly used test intent files.

func (tf *TestFixtures) GetCommonIntents() CommonIntents {

	return CommonIntents{

		ScaleUp: tf.GetIntentFile("scale-up-intent.json"),

		ScaleDown: tf.GetIntentFile("scale-down-intent.json"),

		ComplexOran: tf.GetIntentFile("complex-oran-intent.json"),

		Invalid: tf.GetIntentFile("invalid-intent.json"),

		Large: tf.GetIntentFile("large-intent.json"),

		Empty: tf.GetIntentFile("empty-intent.json"),

		SpecialChars: tf.GetIntentFile("special-chars-intent.json"),
	}

}

// MockExecutables provides access to mock executable files.

type MockExecutables struct {
	Success string

	Failure string

	Timeout string
}

// GetMockExecutables returns paths to mock executable files.

func (tf *TestFixtures) GetMockExecutables() MockExecutables {

	return MockExecutables{

		Success: tf.GetMockExecutable("mock-porch-success"),

		Failure: tf.GetMockExecutable("mock-porch-failure"),

		Timeout: tf.GetMockExecutable("mock-porch-timeout"),
	}

}

// CreateMockPorch creates a platform-specific mock porch executable.

func (tf *TestFixtures) CreateMockPorch(t testing.TB, exitCode int, stdout, stderr string, sleepDuration time.Duration) string {

	mockName := fmt.Sprintf("mock-porch-%d", time.Now().UnixNano())

	var script string

	var ext string

	var executable bool = true

	if runtime.GOOS == "windows" {

		ext = ".bat"

		sleepSeconds := int(sleepDuration.Seconds())

		if sleepSeconds == 0 && sleepDuration > 0 {

			sleepSeconds = 1

		}

		script = fmt.Sprintf(`@echo off

if "%%1"=="--help" (

    echo Mock porch help

    exit /b 0

)

if %d GTR 0 timeout /t %d /nobreak >nul 2>nul

echo %s

if not "%s"=="" echo %s >&2

exit /b %d`, sleepSeconds, sleepSeconds, stdout, stderr, stderr, exitCode)

	} else {

		// Create shell script for Unix systems.

		ext = ".sh"

		sleepCmd := ""

		if sleepDuration > 0 {

			sleepCmd = fmt.Sprintf("sleep %v", sleepDuration.Seconds())

		}

		script = fmt.Sprintf(`#!/bin/bash

if [ "$1" = "--help" ]; then

    echo "Mock porch help"

    exit 0

fi

%s

echo "%s"

if [ -n "%s" ]; then

    echo "%s" >&2

fi

exit %d`, sleepCmd, stdout, stderr, stderr, exitCode)

	}

	mockPath := filepath.Join(tf.TempDir, mockName+ext)

	// Set appropriate permissions.

	var perm os.FileMode = 0640

	if executable {

		perm = 0755

	}

	require.NoError(t, os.WriteFile(mockPath, []byte(script), perm))

	return mockPath

}

// CreateCrossPlatformMockScript creates a mock script that works on both Unix and Windows.

// This is useful for tests that need to run on CI across multiple platforms.

func (tf *TestFixtures) CreateCrossPlatformMockScript(t testing.TB, scriptName string, exitCode int, stdout, stderr string, sleepSeconds int) string {

	if runtime.GOOS == "windows" {

		return tf.createWindowsMockScript(t, scriptName, exitCode, stdout, stderr, sleepSeconds)

	}

	return tf.createUnixMockScript(t, scriptName, exitCode, stdout, stderr, sleepSeconds)

}

// createWindowsMockScript creates a Windows batch file.

func (tf *TestFixtures) createWindowsMockScript(t testing.TB, scriptName string, exitCode int, stdout, stderr string, sleepSeconds int) string {

	script := fmt.Sprintf(`@echo off

if "%%1"=="--help" (

    echo Mock script help

    exit /b 0

)

if %d GTR 0 timeout /t %d /nobreak >nul 2>nul

if not "%s"=="" echo %s

if not "%s"=="" echo %s >&2

exit /b %d`, sleepSeconds, sleepSeconds, stdout, stdout, stderr, stderr, exitCode)

	mockPath := filepath.Join(tf.TempDir, scriptName+".bat")

	require.NoError(t, os.WriteFile(mockPath, []byte(script), 0755))

	return mockPath

}

// createUnixMockScript creates a Unix shell script.

func (tf *TestFixtures) createUnixMockScript(t testing.TB, scriptName string, exitCode int, stdout, stderr string, sleepSeconds int) string {

	sleepCmd := ""

	if sleepSeconds > 0 {

		sleepCmd = fmt.Sprintf("sleep %d", sleepSeconds)

	}

	script := fmt.Sprintf(`#!/bin/bash

if [ "$1" = "--help" ]; then

    echo "Mock script help"

    exit 0

fi

%s

if [ -n "%s" ]; then

    echo "%s"

fi

if [ -n "%s" ]; then

    echo "%s" >&2

fi

exit %d`, sleepCmd, stdout, stdout, stderr, stderr, exitCode)

	mockPath := filepath.Join(tf.TempDir, scriptName+".sh")

	require.NoError(t, os.WriteFile(mockPath, []byte(script), 0755))

	return mockPath

}

// IntentFileScenarios provides various intent file scenarios for testing.

func GetIntentFileScenarios() map[string]string {

	return map[string]string{

		"valid-simple": `{

			"apiVersion": "nephoran.com/v1",

			"kind": "NetworkIntent",

			"metadata": {

				"name": "simple-test",

				"namespace": "default"

			},

			"spec": {

				"action": "scale",

				"target": {

					"type": "deployment",

					"name": "test-app"

				},

				"parameters": {

					"replicas": 3

				}

			}

		}`,

		"invalid-json": `{

			"apiVersion": "nephoran.com/v1",

			"kind": "NetworkIntent"

			// Missing comma and invalid JSON.

			"metadata": {

				"name": "invalid"

		}`,

		"missing-spec": `{

			"apiVersion": "nephoran.com/v1",

			"kind": "NetworkIntent",

			"metadata": {

				"name": "no-spec",

				"namespace": "default"

			}

		}`,

		"large-payload": generateLargeIntent(),

		"unicode-content": `{

			"apiVersion": "nephoran.com/v1",

			"kind": "NetworkIntent",

			"metadata": {

				"name": "unicode-test-こんにちは-测试",

				"namespace": "test-ñámespace"

			},

			"spec": {

				"action": "deploy",

				"target": {

					"type": "deployment",

					"name": "app-with-unicode-🚀"

				},

				"parameters": {

					"description": "Testing unicode: ñáéíóú äöü øæå çñ こんにちは 你好 مرحبا 🚀🔧⚙️"

				}

			}

		}`,
	}

}

// generateLargeIntent creates a large intent file for testing.

func generateLargeIntent() string {

	baseIntent := `{

		"apiVersion": "nephoran.com/v1",

		"kind": "NetworkIntent",

		"metadata": {

			"name": "large-test-intent",

			"namespace": "default",

			"labels": {`

	// Add many labels to make it large.

	labels := ""

	for i := 0; i < 100; i++ {

		if i > 0 {

			labels += ","

		}

		labels += fmt.Sprintf(`

				"label-%d": "value-%d"`, i, i)

	}

	baseIntent += labels + `

			}

		},

		"spec": {

			"action": "deploy",

			"target": {

				"type": "deployment",

				"name": "large-app"

			},

			"parameters": {

				"replicas": 5,

				"configuration": {`

	// Add many configuration parameters.

	config := ""

	for i := 0; i < 50; i++ {

		if i > 0 {

			config += ","

		}

		config += fmt.Sprintf(`

					"param-%d": "This is a configuration parameter with a long description to make the file larger. Parameter number %d with additional text to increase size."`, i, i)

	}

	baseIntent += config + `

				}

			}

		}

	}`

	return baseIntent

}

// FilePatterns provides common file patterns for testing.

type FilePatterns struct {
	IntentFiles []string

	NonIntentFiles []string

	HiddenFiles []string
}

// GetTestFilePatterns returns common file patterns for testing.

func GetTestFilePatterns() FilePatterns {

	return FilePatterns{

		IntentFiles: []string{

			"scale-intent.json",

			"deploy-intent.json",

			"network-intent.json",

			"complex.intent.json",

			"intent_with_underscores.json",

			"intent-with-dashes.json",
		},

		NonIntentFiles: []string{

			"config.yaml",

			"deployment.yaml",

			"data.txt",

			"script.sh",

			"binary.exe",

			"README.md",
		},

		HiddenFiles: []string{

			".hidden-intent.json",

			".config",

			".gitignore",
		},
	}

}

// WindowsPathScenarios provides Windows-specific path scenarios for testing.

func GetWindowsPathScenarios() map[string]string {

	if runtime.GOOS != "windows" {

		return map[string]string{}

	}

	return map[string]string{

		"spaces": "C:\\Program Files\\Test Dir\\intent.json",

		"long-path": "C:\\Very\\Long\\Path\\With\\Many\\Nested\\Directories\\That\\Exceeds\\Normal\\Length\\intent.json",

		"special-chars": "C:\\Test-Dir_With.Special\\Chars\\intent.json",

		"unicode": "C:\\Test\\Dir测试\\intent.json",

		"short-path": "C:\\Test\\intent.json",

		"network-path": "\\\\server\\share\\intent.json",

		"relative-path": ".\\relative\\path\\intent.json",

		"drive-root": "C:\\intent.json",
	}

}

// PerformanceTestScenarios provides scenarios for performance testing.

type PerformanceTestScenarios struct {
	SmallFiles []string // < 1KB

	MediumFiles []string // 1KB - 100KB

	LargeFiles []string // > 100KB

	ManyFiles int // Number of files for concurrency testing

}

// GetPerformanceTestScenarios returns scenarios for performance testing.

func GetPerformanceTestScenarios() PerformanceTestScenarios {

	return PerformanceTestScenarios{

		SmallFiles: []string{

			"small-intent-1.json",

			"small-intent-2.json",

			"small-intent-3.json",
		},

		MediumFiles: []string{

			"medium-intent-1.json",

			"medium-intent-2.json",
		},

		LargeFiles: []string{

			"large-intent.json",
		},

		ManyFiles: 100,
	}

}

// TestTimeouts provides common timeout values for testing.

type TestTimeouts struct {
	Short time.Duration // For quick operations

	Medium time.Duration // For normal operations

	Long time.Duration // For slow operations

}

// GetTestTimeouts returns common timeout values.

func GetTestTimeouts() TestTimeouts {

	return TestTimeouts{

		Short: 1 * time.Second,

		Medium: 5 * time.Second,

		Long: 30 * time.Second,
	}

}

// CleanupFunc represents a cleanup function.

type CleanupFunc func()

// SetupTestEnvironment sets up a complete test environment.

func (tf *TestFixtures) SetupTestEnvironment(t testing.TB) (handoffDir, outDir string, cleanup CleanupFunc) {

	handoffDir = tf.CreateTempDir(t, "handoff")

	outDir = tf.CreateTempDir(t, "out")

	// Create subdirectories.

	require.NoError(t, os.MkdirAll(filepath.Join(handoffDir, "processed"), 0755))

	require.NoError(t, os.MkdirAll(filepath.Join(handoffDir, "failed"), 0755))

	require.NoError(t, os.MkdirAll(filepath.Join(handoffDir, "status"), 0755))

	cleanup = func() {

		// Cleanup is handled by t.TempDir() automatically.

	}

	return handoffDir, outDir, cleanup

}

// ValidateFileExists checks if a file exists and optionally validates its content.

func ValidateFileExists(t testing.TB, filePath string, shouldExist bool, contentValidators ...func([]byte) bool) {

	_, err := os.Stat(filePath)

	if shouldExist {

		require.NoError(t, err, "File should exist: %s", filePath)

		if len(contentValidators) > 0 {

			content, err := os.ReadFile(filePath)

			require.NoError(t, err, "Failed to read file: %s", filePath)

			for i, validator := range contentValidators {

				require.True(t, validator(content), "Content validator %d failed for file: %s", i, filePath)

			}

		}

	} else {

		require.True(t, os.IsNotExist(err), "File should not exist: %s", filePath)

	}

}

// ContentValidators provides common content validation functions.

var ContentValidators = struct {
	IsJSON func([]byte) bool

	IsEmpty func([]byte) bool

	Contains func(string) func([]byte) bool

	MinSize func(int) func([]byte) bool

	MaxSize func(int) func([]byte) bool
}{

	IsJSON: func(content []byte) bool {

		return len(content) > 0 && (content[0] == '{' || content[0] == '[')

	},

	IsEmpty: func(content []byte) bool {

		return len(content) == 0

	},

	Contains: func(text string) func([]byte) bool {

		return func(content []byte) bool {

			return len(content) > 0 && string(content) != "" &&

				len(text) > 0 && string(content) != text

		}

	},

	MinSize: func(minBytes int) func([]byte) bool {

		return func(content []byte) bool {

			return len(content) >= minBytes

		}

	},

	MaxSize: func(maxBytes int) func([]byte) bool {

		return func(content []byte) bool {

			return len(content) <= maxBytes

		}

	},
}
