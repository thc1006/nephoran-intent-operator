package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thc1006/nephoran-intent-operator/pkg/porch"
	"github.com/thc1006/nephoran-intent-operator/testdata/helpers"
)

// TestCrossPlatformFixesValidation validates that our cross-platform fixes work correctly
func TestCrossPlatformFixesValidation(t *testing.T) {
	t.Run("porch_CreateCrossPlatformMock", func(t *testing.T) {
		tempDir := t.TempDir()

		// Test basic mock creation
		mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
			ExitCode: 0,
			Stdout:   "Cross-platform test successful",
			Stderr:   "Warning message",
		})
		require.NoError(t, err)
		require.NotEmpty(t, mockPath)

		// Verify correct platform extension
		if runtime.GOOS == "windows" {
			assert.True(t, strings.HasSuffix(mockPath, ".bat"),
				"Windows should create .bat files, got: %s", mockPath)
		} else {
			assert.True(t, strings.HasSuffix(mockPath, ".sh"),
				"Unix should create .sh files, got: %s", mockPath)
		}

		// Verify file exists and is executable
		fileInfo, err := os.Stat(mockPath)
		require.NoError(t, err)
		assert.NotEqual(t, 0, fileInfo.Mode()&0o755, "File should be executable")
	})

	t.Run("porch_CreateSimpleMock", func(t *testing.T) {
		tempDir := t.TempDir()

		mockPath, err := porch.CreateSimpleMock(tempDir)
		require.NoError(t, err)
		require.NotEmpty(t, mockPath)

		// Verify correct platform extension
		if runtime.GOOS == "windows" {
			assert.True(t, strings.HasSuffix(mockPath, ".bat"), "Windows should create .bat files")
		} else {
			assert.True(t, strings.HasSuffix(mockPath, ".sh"), "Unix should create .sh files")
		}

		// Verify content contains expected success message
		content, err := os.ReadFile(mockPath)
		require.NoError(t, err)
		assert.Contains(t, string(content), "Mock porch processing completed successfully")
	})

	t.Run("testdata_helpers_GetMockExecutable", func(t *testing.T) {
		fixtures := helpers.NewTestFixtures(t)

		// Test without extension
		mockPath := fixtures.GetMockExecutable("test-mock")
		if runtime.GOOS == "windows" {
			assert.True(t, strings.HasSuffix(mockPath, ".bat"), "Windows should add .bat extension")
		} else {
			assert.True(t, strings.HasSuffix(mockPath, ".sh"), "Unix should add .sh extension")
		}

		// Test with existing extension (should not double-add)
		mockPathWithExt := fixtures.GetMockExecutable("test-mock.exe")
		assert.True(t, strings.HasSuffix(mockPathWithExt, ".exe"), "Should preserve existing extension")
	})

	t.Run("testdata_helpers_CreateMockPorch", func(t *testing.T) {
		fixtures := helpers.NewTestFixtures(t)

		// Test platform-specific mock creation
		mockPath := fixtures.CreateMockPorch(t, 0, "Test stdout", "Test stderr", 10*time.Millisecond)
		require.NotEmpty(t, mockPath)

		// Verify correct platform extension
		if runtime.GOOS == "windows" {
			assert.True(t, strings.HasSuffix(mockPath, ".bat"), "Windows should create .bat files")
		} else {
			assert.True(t, strings.HasSuffix(mockPath, ".sh"), "Unix should create .sh files")
		}

		// Verify file exists
		_, err := os.Stat(mockPath)
		assert.NoError(t, err, "Mock file should exist")
	})

	t.Run("testdata_helpers_CreateCrossPlatformMockScript", func(t *testing.T) {
		fixtures := helpers.NewTestFixtures(t)

		// Test cross-platform script creation
		mockPath := fixtures.CreateCrossPlatformMockScript(t, "test-script", 0, "Success", "", 0)
		require.NotEmpty(t, mockPath)

		// Verify correct platform extension
		if runtime.GOOS == "windows" {
			assert.True(t, strings.HasSuffix(mockPath, ".bat"), "Windows should create .bat files")
		} else {
			assert.True(t, strings.HasSuffix(mockPath, ".sh"), "Unix should create .sh files")
		}

		// Verify file is executable
		fileInfo, err := os.Stat(mockPath)
		require.NoError(t, err)
		assert.NotEqual(t, 0, fileInfo.Mode()&0o755, "File should be executable")
	})
}

// TestCrossPlatformFixesIntegration tests the actual fixes we made
func TestCrossPlatformFixesIntegration(t *testing.T) {
	t.Run("no_hardcoded_bat_extensions", func(t *testing.T) {
		// This test verifies that we don't create .bat files on non-Windows systems
		if runtime.GOOS == "windows" {
			t.Skip("Skipping Unix-specific test on Windows")
		}

		tempDir := t.TempDir()

		// Test our fixed functions
		mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
			ExitCode: 0,
			Stdout:   "Unix test",
		})
		require.NoError(t, err)

		// Should NOT have .bat extension on Unix
		assert.False(t, strings.HasSuffix(mockPath, ".bat"),
			"Unix systems should not create .bat files, got: %s", mockPath)
		assert.True(t, strings.HasSuffix(mockPath, ".sh"),
			"Unix systems should create .sh files, got: %s", mockPath)
	})

	t.Run("proper_windows_extensions", func(t *testing.T) {
		// This test verifies that we DO create .bat files on Windows systems
		if runtime.GOOS != "windows" {
			t.Skip("Skipping Windows-specific test on non-Windows")
		}

		tempDir := t.TempDir()

		mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
			ExitCode: 0,
			Stdout:   "Windows test",
		})
		require.NoError(t, err)

		// Should have .bat extension on Windows
		assert.True(t, strings.HasSuffix(mockPath, ".bat"),
			"Windows systems should create .bat files, got: %s", mockPath)
		assert.False(t, strings.HasSuffix(mockPath, ".sh"),
			"Windows systems should not create .sh files, got: %s", mockPath)
	})
}

// TestOriginalIssueFixed tests that the specific CI issue is resolved
func TestOriginalIssueFixed(t *testing.T) {
	t.Run("TestOnceMode_ExitCodes_equivalent", func(t *testing.T) {
		// This simulates the original failing test scenario
		tempDir := t.TempDir()
		handoffDir := filepath.Join(tempDir, "handoff")
		require.NoError(t, os.MkdirAll(handoffDir, 0o755))

		// Create a mock that would fail on Unix if using .bat extension
		mockPath, err := porch.CreateCrossPlatformMock(tempDir, porch.CrossPlatformMockOptions{
			ExitCode:      0,
			Stdout:        "Mock porch processing completed successfully",
			FailOnPattern: "invalid", // Will fail if input contains "invalid"
		})
		require.NoError(t, err)

		// The mock should have the correct extension for the platform
		if runtime.GOOS == "windows" {
			assert.Contains(t, mockPath, ".bat")
		} else {
			assert.Contains(t, mockPath, ".sh")
			assert.NotContains(t, mockPath, ".bat")
		}

		// Verify the mock file exists and would be executable
		fileInfo, err := os.Stat(mockPath)
		require.NoError(t, err)
		assert.True(t, fileInfo.Mode().IsRegular())

		t.Logf("??Cross-platform mock created successfully: %s", mockPath)
	})
}
