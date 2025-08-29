package platform

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestMakeExecutablePermissions verifies that MakeExecutable sets secure permissions
func TestMakeExecutablePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Unix permission test on Windows")
	}

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_executable")

	// Create a test file
	if err := os.WriteFile(testFile, []byte("#!/bin/bash\necho test"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Make it executable
	if err := MakeExecutable(testFile); err != nil {
		t.Fatalf("MakeExecutable failed: %v", err)
	}

	// Check the permissions
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat file: %v", err)
	}

	mode := info.Mode()
	
	// On Unix, should have 0750 permissions (owner: rwx, group: r-x, others: ---)
	expectedMode := os.FileMode(0750)
	if mode.Perm() != expectedMode {
		t.Errorf("Expected permissions %v, got %v", expectedMode, mode.Perm())
	}

	// Verify owner can read, write, and execute
	if mode.Perm()&0700 != 0700 {
		t.Error("Owner should have full permissions (rwx)")
	}

	// Verify group has read and execute only
	if mode.Perm()&0050 != 0050 {
		t.Error("Group should have read and execute permissions")
	}
	if mode.Perm()&0020 != 0 {
		t.Error("Group should not have write permission")
	}

	// Verify others have no permissions
	if mode.Perm()&0007 != 0 {
		t.Error("Others should have no permissions")
	}
}

// TestMakeExecutableWindowsCompatibility verifies Windows handling
func TestMakeExecutableWindowsCompatibility(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Skipping Windows test on Unix")
	}

	// Create a temporary directory for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test_file.txt")

	// Create a test file
	if err := os.WriteFile(testFile, []byte("test content"), 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Make it "executable" (on Windows, this should handle permissions gracefully)
	if err := MakeExecutable(testFile); err != nil {
		t.Fatalf("MakeExecutable failed on Windows: %v", err)
	}

	// File should still exist and be readable
	if _, err := os.Stat(testFile); err != nil {
		t.Errorf("File should still exist after MakeExecutable: %v", err)
	}
}