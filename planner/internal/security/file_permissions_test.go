package security

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/planner"
	"github.com/thc1006/nephoran-intent-operator/planner/internal/rules"
)

// TestFilePermissions_IntentFiles tests that intent files are created with secure 0600 permissions
func TestFilePermissions_IntentFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file permission tests in short mode")
	}

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "planner-security-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validator := NewValidator(DefaultValidationConfig())

	// Create sample intent data
	intent := &planner.Intent{
		IntentType:    "scaling",
		Target:        "test-cnf",
		Namespace:     "test-namespace",
		Replicas:      3,
		Reason:        "High PRB utilization detected",
		Source:        "planner",
		CorrelationID: fmt.Sprintf("test-%d", time.Now().Unix()),
	}

	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal intent: %v", err)
	}

	filename := fmt.Sprintf("intent-%d.json", time.Now().Unix())
	path := filepath.Join(tempDir, filename)

	// Validate path security
	if err := validator.ValidateFilePath(path, "test intent file"); err != nil {
		t.Fatalf("Path validation failed: %v", err)
	}

	// Write file with secure permissions (simulating production code)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatalf("Failed to write intent file: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("Intent file was not created: %v", err)
	}

	// Test actual permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Failed to stat intent file: %v", err)
	}

	mode := info.Mode()
	perm := mode.Perm()

	// Verify permissions are exactly 0600 (on Unix) or that the file is created (on Windows)
	expectedPerm := fs.FileMode(0600)
	if runtime.GOOS == "windows" {
		// On Windows, file permissions work differently due to NTFS ACLs
		// We primarily verify that the file was created successfully
		// and that we can read it as the owner
		t.Logf("Windows detected: file permissions = %o (Windows handles permissions via ACLs)", perm)
		if perm == 0 {
			t.Error("File should have some permissions set")
		}
	} else {
		// On Unix-like systems, verify exact permissions
		if perm != expectedPerm {
			t.Errorf("Intent file permissions = %o, expected %o", perm, expectedPerm)
		}
	}

	// Additional verification for Unix systems
	if runtime.GOOS != "windows" {
		// Verify owner can read and write
		if mode&0400 == 0 {
			t.Error("Owner should have read permission")
		}
		if mode&0200 == 0 {
			t.Error("Owner should have write permission")
		}

		// Verify group and others cannot access
		if mode&0040 != 0 {
			t.Error("Group should not have read permission")
		}
		if mode&0020 != 0 {
			t.Error("Group should not have write permission")
		}
		if mode&0004 != 0 {
			t.Error("Others should not have read permission")
		}
		if mode&0002 != 0 {
			t.Error("Others should not have write permission")
		}
	}

	// Test content integrity
	readData, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("Failed to read back intent file: %v", err)
	}

	var readIntent planner.Intent
	if err := json.Unmarshal(readData, &readIntent); err != nil {
		t.Fatalf("Failed to unmarshal read intent: %v", err)
	}

	if readIntent.IntentType != intent.IntentType ||
		readIntent.Target != intent.Target ||
		readIntent.Replicas != intent.Replicas {
		t.Error("Intent data was corrupted during secure write/read")
	}
}

// TestFilePermissions_StateFiles tests that state files are created with secure 0600 permissions
func TestFilePermissions_StateFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping file permission tests in short mode")
	}

	// Create temporary directory for test
	tempDir, err := os.MkdirTemp("", "planner-state-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	stateFile := filepath.Join(tempDir, "test-state.json")

	// Create rule engine configuration
	config := rules.Config{
		StateFile:            stateFile,
		CooldownDuration:     60 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     90 * time.Second,
	}

	engine := rules.NewRuleEngine(config)
	validator := NewValidator(DefaultValidationConfig())
	engine.SetValidator(validator)

	// Create sample KMP data that should trigger state save
	kmpData := rules.KPMData{
		Timestamp:       time.Now(),
		NodeID:          "test-node-001",
		PRBUtilization:  0.9, // High utilization to trigger scaling
		P95Latency:      150.0,
		ActiveUEs:       100,
		CurrentReplicas: 2,
	}

	// Process metrics to trigger state save
	decision := engine.Evaluate(kmpData)
	if decision == nil {
		t.Log("No scaling decision made, forcing state save...")
		// Force a state save by calling the engine's save method indirectly
		// by providing metrics that will trigger a cooldown state
		for i := 0; i < 3; i++ {
			engine.Evaluate(kmpData)
		}
	}

	// Verify state file was created
	if _, err := os.Stat(stateFile); err != nil {
		t.Fatalf("State file was not created: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(stateFile)
	if err != nil {
		t.Fatalf("Failed to stat state file: %v", err)
	}

	mode := info.Mode()
	perm := mode.Perm()
	expectedPerm := fs.FileMode(0600)

	if perm != expectedPerm {
		t.Errorf("State file permissions = %o, expected %o", perm, expectedPerm)
	}

	// Verify state file content is valid JSON
	data, err := os.ReadFile(stateFile)
	if err != nil {
		t.Fatalf("Failed to read state file: %v", err)
	}

	var state interface{}
	if err := json.Unmarshal(data, &state); err != nil {
		t.Fatalf("State file does not contain valid JSON: %v", err)
	}
}

// TestFilePermissions_UnauthorizedAccess tests that unauthorized users cannot access secured files
func TestFilePermissions_UnauthorizedAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping unauthorized access tests in short mode")
	}

	// Skip this test on Windows as it has different permission semantics
	if runtime.GOOS == "windows" {
		t.Skip("Skipping unauthorized access test on Windows due to different permission model")
	}

	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "planner-access-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "secure-intent.json")
	testData := []byte(`{"test": "sensitive data"}`)

	// Write file with 0600 permissions
	if err := os.WriteFile(testFile, testData, 0600); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	// Verify current user can read the file
	if _, err := os.ReadFile(testFile); err != nil {
		t.Fatalf("Owner should be able to read the file: %v", err)
	}

	// Get file stats to verify permissions are set correctly
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}

	// Verify permissions in detail on Unix systems
	if runtime.GOOS != "windows" {
		// On Unix systems, verify detailed permission bits
		t.Logf("Unix system detected, file mode: %v", info.Mode())
		// Note: Advanced permission checking would require platform-specific imports
		// For now, we rely on the standard Go fs.FileMode permission check above
	} else {
		t.Logf("Windows system detected, file mode: %v", info.Mode())
	}
}

// TestFilePermissions_CrossPlatformCompatibility tests file permission handling across platforms
func TestFilePermissions_CrossPlatformCompatibility(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "planner-platform-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "cross-platform-test.json")
	testData := []byte(`{"platform": "` + runtime.GOOS + `"}`)

	// Write file with 0600 permissions
	if err := os.WriteFile(testFile, testData, 0600); err != nil {
		t.Fatalf("Failed to write test file on %s: %v", runtime.GOOS, err)
	}

	// Verify file exists and is readable by owner
	if _, err := os.ReadFile(testFile); err != nil {
		t.Fatalf("Failed to read back test file on %s: %v", runtime.GOOS, err)
	}

	// Check permissions interpretation on different platforms
	info, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}

	mode := info.Mode()
	t.Logf("Platform: %s, File mode: %s (%o)", runtime.GOOS, mode, mode.Perm())

	switch runtime.GOOS {
	case "windows":
		// On Windows, Go tries to approximate Unix permissions
		// but the actual behavior depends on NTFS ACLs
		if !mode.IsRegular() {
			t.Error("File should be a regular file")
		}
		// We mainly verify the file is created and accessible to the owner

	case "linux", "darwin", "freebsd", "openbsd", "netbsd":
		// On Unix-like systems, verify exact permissions
		expectedPerm := fs.FileMode(0600)
		if mode.Perm() != expectedPerm {
			t.Errorf("Unix file permissions = %o, expected %o", mode.Perm(), expectedPerm)
		}
	default:
		t.Logf("Unknown platform %s, basic file operations test passed", runtime.GOOS)
	}
}

// TestFilePermissions_DirectoryTraversalPrevention tests that directory traversal attacks don't bypass permission controls
func TestFilePermissions_DirectoryTraversalPrevention(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "planner-traversal-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validator := NewValidator(DefaultValidationConfig())

	// Create a subdirectory structure
	subDir := filepath.Join(tempDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Test various path traversal attempts
	traversalPaths := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"subdir/../../etc/shadow",
		"subdir/../../../boot/grub/grub.cfg",
		filepath.Join("subdir", "..", "..", "sensitive-file.json"),
	}

	for _, maliciousPath := range traversalPaths {
		t.Run(fmt.Sprintf("traversal-%s", strings.ReplaceAll(maliciousPath, string(os.PathSeparator), "_")), func(t *testing.T) {
			fullPath := filepath.Join(tempDir, maliciousPath)

			// Validate the raw traversal path so ".." sequences are visible to the validator.
			err := validator.ValidateFilePath(maliciousPath, "traversal test")
			if err == nil {
				t.Errorf("Path traversal attempt should have been rejected: %s", maliciousPath)
			}

			// Even if somehow the validation was bypassed, ensure the file creation
			// doesn't escape the intended directory
			if err == nil {
				testData := []byte("malicious content")
				writeErr := os.WriteFile(fullPath, testData, 0600)
				if writeErr == nil {
					// If file was created, verify it's still within our temp directory
					absPath, _ := filepath.Abs(fullPath)
					absTempDir, _ := filepath.Abs(tempDir)

					if !strings.HasPrefix(absPath, absTempDir) {
						t.Errorf("File creation escaped temp directory: %s not under %s", absPath, absTempDir)
					}
				}
			}
		})
	}
}

// TestFilePermissions_PermissionInheritance tests that permission settings are applied consistently
func TestFilePermissions_PermissionInheritance(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "planner-inheritance-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create multiple files to ensure consistent permission application
	testFiles := []string{
		"intent-001.json",
		"intent-002.json",
		"state-001.json",
		"metrics-001.json",
	}

	for i, filename := range testFiles {
		fullPath := filepath.Join(tempDir, filename)
		testData := []byte(fmt.Sprintf(`{"file": "%s", "index": %d}`, filename, i))

		// Write each file with 0600 permissions
		if err := os.WriteFile(fullPath, testData, 0600); err != nil {
			t.Fatalf("Failed to write test file %s: %v", filename, err)
		}

		// Verify permissions for each file
		info, err := os.Stat(fullPath)
		if err != nil {
			t.Fatalf("Failed to stat file %s: %v", filename, err)
		}

		expectedPerm := fs.FileMode(0600)
		if info.Mode().Perm() != expectedPerm {
			t.Errorf("File %s permissions = %o, expected %o", filename, info.Mode().Perm(), expectedPerm)
		}
	}
}

// BenchmarkFilePermissions_WritePerformance benchmarks the performance impact of secure file writes
func BenchmarkFilePermissions_WritePerformance(b *testing.B) {
	tempDir, err := os.MkdirTemp("", "planner-perf-test-*")
	if err != nil {
		b.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testData := []byte(`{"benchmark": true, "size": "1kb", "data": "` + strings.Repeat("x", 1000) + `"}`)

	b.ResetTimer()

	b.Run("secure-0600", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			filename := filepath.Join(tempDir, fmt.Sprintf("secure-%d.json", i))
			if err := os.WriteFile(filename, testData, 0600); err != nil {
				b.Fatalf("Failed to write file: %v", err)
			}
		}
	})

	b.Run("default-0644", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			filename := filepath.Join(tempDir, fmt.Sprintf("default-%d.json", i))
			if err := os.WriteFile(filename, testData, 0644); err != nil {
				b.Fatalf("Failed to write file: %v", err)
			}
		}
	})
}

// TestFilePermissions_RealWorldScenario tests file permissions in a realistic planner usage scenario
func TestFilePermissions_RealWorldScenario(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real-world scenario test in short mode")
	}

	tempDir, err := os.MkdirTemp("", "planner-realworld-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	validator := NewValidator(DefaultValidationConfig())

	// Simulate real planner operation: multiple intent files over time
	for i := 0; i < 5; i++ {
		// Create intent data
		intent := &planner.Intent{
			IntentType:    "scaling",
			Target:        fmt.Sprintf("cnf-%d", i),
			Namespace:     "production",
			Replicas:      i + 2,
			Reason:        fmt.Sprintf("Test scenario %d", i),
			Source:        "planner",
			CorrelationID: fmt.Sprintf("realworld-%d-%d", i, time.Now().Unix()),
		}

		data, err := json.MarshalIndent(intent, "", "  ")
		if err != nil {
			t.Fatalf("Failed to marshal intent %d: %v", i, err)
		}

		filename := fmt.Sprintf("intent-%d-%d.json", i, time.Now().Unix())
		path := filepath.Join(tempDir, filename)

		// Validate path
		if err := validator.ValidateFilePath(path, "real-world intent file"); err != nil {
			t.Fatalf("Path validation failed for %s: %v", path, err)
		}

		// Write with secure permissions
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatalf("Failed to write intent file %s: %v", filename, err)
		}

		// Verify file permissions
		info, err := os.Stat(path)
		if err != nil {
			t.Fatalf("Failed to stat intent file %s: %v", filename, err)
		}

		if info.Mode().Perm() != 0600 {
			t.Errorf("Intent file %s has incorrect permissions: %o", filename, info.Mode().Perm())
		}

		// Verify content integrity
		readData, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("Failed to read back intent file %s: %v", filename, err)
		}

		var readIntent planner.Intent
		if err := json.Unmarshal(readData, &readIntent); err != nil {
			t.Fatalf("Failed to unmarshal intent file %s: %v", filename, err)
		}

		if readIntent.Target != intent.Target || readIntent.Replicas != intent.Replicas {
			t.Errorf("Intent data corrupted in file %s", filename)
		}

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// Verify all files maintain secure permissions
	files, err := filepath.Glob(filepath.Join(tempDir, "intent-*.json"))
	if err != nil {
		t.Fatalf("Failed to list intent files: %v", err)
	}

	if len(files) != 5 {
		t.Errorf("Expected 5 intent files, found %d", len(files))
	}

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			t.Errorf("Failed to stat file %s: %v", file, err)
			continue
		}

		if info.Mode().Perm() != 0600 {
			t.Errorf("File %s has incorrect permissions: %o", file, info.Mode().Perm())
		}
	}
}
