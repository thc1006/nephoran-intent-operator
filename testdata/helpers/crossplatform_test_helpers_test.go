package helpers

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCrossPlatformTestEnvironment(t *testing.T) {
	env := NewCrossPlatformTestEnvironment(t)
	
	// Verify all directories exist
	assert.DirExists(t, env.TempDir)
	assert.DirExists(t, env.HandoffDir)
	assert.DirExists(t, env.OutDir)
	assert.DirExists(t, env.StatusDir)
	
	// Verify paths are correct
	assert.Equal(t, filepath.Join(env.TempDir, "handoff"), env.HandoffDir)
	assert.Equal(t, filepath.Join(env.TempDir, "out"), env.OutDir)
	assert.Equal(t, filepath.Join(env.TempDir, "status"), env.StatusDir)
}

func TestSetupMockPorch(t *testing.T) {
	env := NewCrossPlatformTestEnvironment(t)
	
	// Test simple mock setup
	env.SetupMockPorch(0, "test output", "test error")
	
	assert.NotEmpty(t, env.MockPorch)
	assert.FileExists(t, env.MockPorch)
	
	// Verify the mock has the correct extension
	if runtime.GOOS == "windows" {
		assert.Contains(t, env.MockPorch, ".bat")
	} else {
		assert.Contains(t, env.MockPorch, ".sh")
	}
}

func TestSetupSimpleMockPorch(t *testing.T) {
	env := NewCrossPlatformTestEnvironment(t)
	
	env.SetupSimpleMockPorch()
	
	assert.NotEmpty(t, env.MockPorch)
	assert.FileExists(t, env.MockPorch)
}

func TestSetupFailingMockPorch(t *testing.T) {
	env := NewCrossPlatformTestEnvironment(t)
	
	env.SetupFailingMockPorch(1, "Mock failure")
	
	assert.NotEmpty(t, env.MockPorch)
	assert.FileExists(t, env.MockPorch)
}

func TestMockPorchWithOptions(t *testing.T) {
	env := NewCrossPlatformTestEnvironment(t)
	
	// Test with sleep option
	env.SetupMockPorch(0, "slow output", "", WithSleep(100*time.Millisecond))
	
	assert.NotEmpty(t, env.MockPorch)
	assert.FileExists(t, env.MockPorch)
	
	// Test with pattern option
	env.SetupMockPorch(1, "", "pattern error", WithFailOnPattern("ERROR"))
	
	assert.NotEmpty(t, env.MockPorch)
	assert.FileExists(t, env.MockPorch)
}

func TestCreateIntentFile(t *testing.T) {
	env := NewCrossPlatformTestEnvironment(t)
	
	content := `{"action": "test", "target": {"type": "deployment", "name": "test"}, "replicas": 1}`
	intentPath := env.CreateIntentFile("test-intent.json", content)
	
	assert.FileExists(t, intentPath)
	assert.Equal(t, filepath.Join(env.HandoffDir, "test-intent.json"), intentPath)
	
	// Verify content
	data, err := os.ReadFile(intentPath)
	require.NoError(t, err)
	assert.Equal(t, content, string(data))
}

func TestGetDefaultArgs(t *testing.T) {
	env := NewCrossPlatformTestEnvironment(t)
	env.SetupSimpleMockPorch()
	
	args := env.GetDefaultArgs()
	
	expectedArgs := []string{
		"-handoff", env.HandoffDir,
		"-porch", env.MockPorch,
		"-out", env.OutDir,
		"-status", env.StatusDir,
		"-once",
		"-debounce", "10ms",
	}
	
	assert.Equal(t, expectedArgs, args)
}

func TestPlatformSpecificTestRunner(t *testing.T) {
	runner := NewPlatformSpecificTestRunner(t)
	
	windowsRan := false
	unixRan := false
	linuxRan := false
	macosRan := false
	
	runner.RunOnWindows(func() {
		windowsRan = true
	})
	
	runner.RunOnUnix(func() {
		unixRan = true
	})
	
	runner.RunOnLinux(func() {
		linuxRan = true
	})
	
	runner.RunOnMacOS(func() {
		macosRan = true
	})
	
	// Verify only the appropriate test ran for the current platform
	switch runtime.GOOS {
	case "windows":
		assert.True(t, windowsRan)
		assert.False(t, unixRan)
		assert.False(t, linuxRan)
		assert.False(t, macosRan)
	case "linux":
		assert.False(t, windowsRan)
		assert.True(t, unixRan)
		assert.True(t, linuxRan)
		assert.False(t, macosRan)
	case "darwin":
		assert.False(t, windowsRan)
		assert.True(t, unixRan)
		assert.False(t, linuxRan)
		assert.True(t, macosRan)
	}
}

func TestCrossPlatformAssertions(t *testing.T) {
	assertions := NewCrossPlatformAssertions(t)
	env := NewCrossPlatformTestEnvironment(t)
	
	// Test AssertExecutableExists with mock porch
	env.SetupSimpleMockPorch()
	assertions.AssertExecutableExists(env.MockPorch)
	
	// Test AssertScriptWorks
	assertions.AssertScriptWorks(env.MockPorch)
}

func TestCreateTestIntentFunctions(t *testing.T) {
	// Test CreateTestIntent
	intent := CreateTestIntent("scale", 3)
	assert.Contains(t, intent, `"action": "scale"`)
	assert.Contains(t, intent, `"replicas": 3`)
	assert.Contains(t, intent, `"type": "deployment"`)
	assert.Contains(t, intent, `"name": "test-app"`)
	
	// Test CreateMinimalIntent
	minimal := CreateMinimalIntent()
	assert.Contains(t, minimal, `"action": "scale"`)
	assert.Contains(t, minimal, `"replicas": 1`)
	
	// Test CreateComplexIntent
	complex := CreateComplexIntent()
	assert.Contains(t, complex, `"name": "complex-intent"`)
	assert.Contains(t, complex, `"namespace": "production"`)
	assert.Contains(t, complex, `"environment": "prod"`)
	assert.Contains(t, complex, `"cpu": "500m"`)
	assert.Contains(t, complex, `"nodeAffinity"`)
}

// Integration test for the complete test environment workflow
func TestFullWorkflow(t *testing.T) {
	// Create environment
	env := NewCrossPlatformTestEnvironment(t)
	
	// Setup mock
	env.SetupSimpleMockPorch()
	
	// Create intent file
	content := CreateMinimalIntent()
	intentPath := env.CreateIntentFile("workflow-test.json", content)
	
	// Verify setup
	assert.FileExists(t, intentPath)
	assert.FileExists(t, env.MockPorch)
	
	// Get default args
	args := env.GetDefaultArgs()
	assert.Contains(t, args, env.MockPorch)
	assert.Contains(t, args, env.HandoffDir)
	
	t.Logf("Full workflow test completed successfully on %s", runtime.GOOS)
}

// Test that verifies cross-platform path handling
func TestCrossPlatformPaths(t *testing.T) {
	env := NewCrossPlatformTestEnvironment(t)
	
	// Create nested directory structure
	nestedPath := env.CreateIntentFile("subdir/nested-intent.json", CreateMinimalIntent())
	
	// Verify the path is correct regardless of platform
	assert.FileExists(t, nestedPath)
	assert.Contains(t, nestedPath, "subdir")
	assert.Contains(t, nestedPath, "nested-intent.json")
	
	// Verify the path uses the correct separator for the platform
	expectedPath := filepath.Join(env.HandoffDir, "subdir", "nested-intent.json")
	assert.Equal(t, expectedPath, nestedPath)
}

// Performance test to ensure cross-platform utilities don't add significant overhead
func BenchmarkCrossPlatformSetup(b *testing.B) {
	for i := 0; i < b.N; i++ {
		// Create a temporary environment for benchmarking
		tempDir := b.TempDir()
		
		env := &CrossPlatformTestEnvironment{
			TempDir:    tempDir,
			HandoffDir: filepath.Join(tempDir, "handoff"),
			OutDir:     filepath.Join(tempDir, "out"),
			StatusDir:  filepath.Join(tempDir, "status"),
		}
		
		// Time the mock setup
		b.StartTimer()
		env.SetupSimpleMockPorch()
		b.StopTimer()
		
		// Verify it worked
		if env.MockPorch == "" {
			b.Fatal("Mock porch setup failed")
		}
	}
}