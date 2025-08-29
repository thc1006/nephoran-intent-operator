
package helpers



import (

	"context"

	"fmt"

	"os"

	"os/exec"

	"path/filepath"

	"runtime"

	"testing"

	"time"



	"github.com/stretchr/testify/require"



	"github.com/thc1006/nephoran-intent-operator/internal/porch"

)



// CrossPlatformTestEnvironment provides a consistent testing environment across platforms.

type CrossPlatformTestEnvironment struct {

	TempDir    string

	HandoffDir string

	OutDir     string

	StatusDir  string

	MockPorch  string

	BinaryPath string

	t          *testing.T

}



// NewCrossPlatformTestEnvironment creates a new test environment.

func NewCrossPlatformTestEnvironment(t *testing.T) *CrossPlatformTestEnvironment {

	tempDir := t.TempDir()



	env := &CrossPlatformTestEnvironment{

		TempDir:    tempDir,

		HandoffDir: filepath.Join(tempDir, "handoff"),

		OutDir:     filepath.Join(tempDir, "out"),

		StatusDir:  filepath.Join(tempDir, "status"),

		t:          t,

	}



	// Create required directories.

	require.NoError(t, os.MkdirAll(env.HandoffDir, 0755))

	require.NoError(t, os.MkdirAll(env.OutDir, 0755))

	require.NoError(t, os.MkdirAll(env.StatusDir, 0755))



	return env

}



// SetupMockPorch creates a mock porch executable with the specified behavior.

func (env *CrossPlatformTestEnvironment) SetupMockPorch(exitCode int, stdout, stderr string, opts ...MockPorchOption) {

	mockOpts := porch.CrossPlatformMockOptions{

		ExitCode: exitCode,

		Stdout:   stdout,

		Stderr:   stderr,

	}



	// Apply options.

	for _, opt := range opts {

		opt(&mockOpts)

	}



	mockPath, err := porch.CreateCrossPlatformMock(env.TempDir, mockOpts)

	require.NoError(env.t, err)



	env.MockPorch = mockPath

}



// SetupSimpleMockPorch creates a basic working mock porch.

func (env *CrossPlatformTestEnvironment) SetupSimpleMockPorch() {

	env.SetupMockPorch(0, "Mock porch processing completed successfully", "")

}



// SetupFailingMockPorch creates a mock porch that fails.

func (env *CrossPlatformTestEnvironment) SetupFailingMockPorch(exitCode int, errorMsg string) {

	env.SetupMockPorch(exitCode, "", errorMsg)

}



// BuildConductorLoop builds the conductor-loop binary for testing.

func (env *CrossPlatformTestEnvironment) BuildConductorLoop() {

	binaryName := "conductor-loop"

	if runtime.GOOS == "windows" {

		binaryName += ".exe"

	}



	env.BinaryPath = filepath.Join(env.TempDir, binaryName)



	// Find the main package directory.

	mainDir := filepath.Join("..", "..", "cmd", "conductor-loop")

	if _, err := os.Stat(mainDir); os.IsNotExist(err) {

		// Try alternative path.

		mainDir = "../../cmd/conductor-loop"

	}



	cmd := exec.Command("go", "build", "-o", env.BinaryPath, ".")

	cmd.Dir = mainDir



	require.NoError(env.t, cmd.Run())

	require.FileExists(env.t, env.BinaryPath)

}



// CreateIntentFile creates a test intent file.

func (env *CrossPlatformTestEnvironment) CreateIntentFile(filename, content string) string {

	intentPath := filepath.Join(env.HandoffDir, filename)

	require.NoError(env.t, os.WriteFile(intentPath, []byte(content), 0640))

	return intentPath

}



// RunConductorLoop executes the conductor-loop with the given arguments.

func (env *CrossPlatformTestEnvironment) RunConductorLoop(ctx context.Context, args ...string) (*exec.Cmd, error) {

	if env.BinaryPath == "" {

		return nil, fmt.Errorf("conductor-loop binary not built, call BuildConductorLoop() first")

	}



	cmd := exec.CommandContext(ctx, env.BinaryPath, args...)

	cmd.Dir = env.TempDir



	return cmd, cmd.Run()

}



// RunConductorLoopWithOutput executes conductor-loop and captures output.

func (env *CrossPlatformTestEnvironment) RunConductorLoopWithOutput(ctx context.Context, args ...string) ([]byte, []byte, error) {

	if env.BinaryPath == "" {

		return nil, nil, fmt.Errorf("conductor-loop binary not built, call BuildConductorLoop() first")

	}



	cmd := exec.CommandContext(ctx, env.BinaryPath, args...)

	cmd.Dir = env.TempDir



	stdout, err := cmd.Output()

	var stderr []byte

	if exitError, ok := err.(*exec.ExitError); ok {

		stderr = exitError.Stderr

	}



	return stdout, stderr, err

}



// GetDefaultArgs returns the default arguments for conductor-loop.

func (env *CrossPlatformTestEnvironment) GetDefaultArgs() []string {

	return []string{

		"-handoff", env.HandoffDir,

		"-porch", env.MockPorch,

		"-out", env.OutDir,

		"-status", env.StatusDir,

		"-once",

		"-debounce", "10ms",

	}

}



// MockPorchOption configures mock porch behavior.

type MockPorchOption func(*porch.CrossPlatformMockOptions)



// WithSleep adds a sleep delay to the mock porch.

func WithSleep(duration time.Duration) MockPorchOption {

	return func(opts *porch.CrossPlatformMockOptions) {

		opts.Sleep = duration

	}

}



// WithFailOnPattern makes the mock fail when a pattern is found.

func WithFailOnPattern(pattern string) MockPorchOption {

	return func(opts *porch.CrossPlatformMockOptions) {

		opts.FailOnPattern = pattern

	}

}



// WithCustomScript provides custom script content.

func WithCustomScript(windows, unix string) MockPorchOption {

	return func(opts *porch.CrossPlatformMockOptions) {

		opts.CustomScript.Windows = windows

		opts.CustomScript.Unix = unix

	}

}



// PlatformSpecificTestRunner runs tests with platform-specific behavior.

type PlatformSpecificTestRunner struct {

	t *testing.T

}



// NewPlatformSpecificTestRunner creates a new platform-specific test runner.

func NewPlatformSpecificTestRunner(t *testing.T) *PlatformSpecificTestRunner {

	return &PlatformSpecificTestRunner{t: t}

}



// RunOnWindows runs the test function only on Windows.

func (r *PlatformSpecificTestRunner) RunOnWindows(testFunc func()) {

	if runtime.GOOS == "windows" {

		testFunc()

	} else {

		r.t.Skip("Skipping Windows-specific test on " + runtime.GOOS)

	}

}



// RunOnUnix runs the test function only on Unix-like systems.

func (r *PlatformSpecificTestRunner) RunOnUnix(testFunc func()) {

	if runtime.GOOS != "windows" {

		testFunc()

	} else {

		r.t.Skip("Skipping Unix-specific test on Windows")

	}

}



// RunOnLinux runs the test function only on Linux.

func (r *PlatformSpecificTestRunner) RunOnLinux(testFunc func()) {

	if runtime.GOOS == "linux" {

		testFunc()

	} else {

		r.t.Skip("Skipping Linux-specific test on " + runtime.GOOS)

	}

}



// RunOnMacOS runs the test function only on macOS.

func (r *PlatformSpecificTestRunner) RunOnMacOS(testFunc func()) {

	if runtime.GOOS == "darwin" {

		testFunc()

	} else {

		r.t.Skip("Skipping macOS-specific test on " + runtime.GOOS)

	}

}



// CrossPlatformAssertions provides platform-aware assertions.

type CrossPlatformAssertions struct {

	t *testing.T

}



// NewCrossPlatformAssertions creates a new cross-platform assertion helper.

func NewCrossPlatformAssertions(t *testing.T) *CrossPlatformAssertions {

	return &CrossPlatformAssertions{t: t}

}



// AssertExecutableExists verifies an executable exists with the correct extension.

func (a *CrossPlatformAssertions) AssertExecutableExists(path string) {

	if runtime.GOOS == "windows" {

		// Check for various Windows executable extensions.

		found := false

		for _, ext := range []string{"", ".exe", ".bat", ".cmd"} {

			testPath := path + ext

			if _, err := os.Stat(testPath); err == nil {

				found = true

				break

			}

		}

		require.True(a.t, found, "Executable not found with any valid Windows extension: %s", path)

	} else {

		require.FileExists(a.t, path)



		// On Unix, also check if it's executable.

		info, err := os.Stat(path)

		require.NoError(a.t, err)

		require.True(a.t, info.Mode()&0111 != 0, "File is not executable: %s", path)

	}

}



// AssertScriptWorks verifies a script can be executed.

func (a *CrossPlatformAssertions) AssertScriptWorks(scriptPath string) {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()



	cmd := exec.CommandContext(ctx, scriptPath, "--help")

	err := cmd.Run()

	require.NoError(a.t, err, "Script failed to execute: %s", scriptPath)

}



// CreateTestIntent returns a sample intent for testing.

func CreateTestIntent(action string, replicas int) string {

	return fmt.Sprintf(`{

	"metadata": {

		"name": "test-intent",

		"namespace": "default"

	},

	"spec": {

		"action": "%s",

		"target": "deployment/test-app",

		"replicas": %d

	}

}`, action, replicas)

}



// CreateMinimalIntent returns a minimal intent for testing.

func CreateMinimalIntent() string {

	return `{"action": "scale", "target": "deployment/test", "replicas": 1}`

}



// CreateComplexIntent returns a complex intent for testing.

func CreateComplexIntent() string {

	return `{

	"metadata": {

		"name": "complex-intent",

		"namespace": "production",

		"labels": {

			"environment": "prod",

			"version": "v1.0.0"

		}

	},

	"spec": {

		"action": "scale",

		"target": "deployment/web-app",

		"replicas": 5,

		"resources": {

			"cpu": "500m",

			"memory": "1Gi"

		},

		"affinity": {

			"nodeAffinity": {

				"requiredDuringSchedulingIgnoredDuringExecution": {

					"nodeSelectorTerms": [{

						"matchExpressions": [{

							"key": "kubernetes.io/os",

							"operator": "In",

							"values": ["linux"]

						}]

					}]

				}

			}

		}

	}

}`

}

