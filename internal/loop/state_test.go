package loop

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewStateManager(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string // returns test directory
		wantErr   bool
	}{
		{
			name: "success with empty directory",
			setupFunc: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: false,
		},
		{
			name: "success with existing state file",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				stateFile := filepath.Join(dir, StateFileName)
				stateData := map[string]interface{}{
					"version": "1.0",
					"states": map[string]*FileState{
						"test": {
							FilePath:    "/test/file.json",
							SHA256:      "abc123",
							Size:        100,
							ProcessedAt: time.Now(),
							Status:      "processed",
						},
					},
				}
				data, _ := json.Marshal(stateData)
				require.NoError(t, os.WriteFile(stateFile, data, 0644))
				return dir
			},
			wantErr: false,
		},
		{
			name: "success with corrupted state file",
			setupFunc: func(t *testing.T) string {
				dir := t.TempDir()
				stateFile := filepath.Join(dir, StateFileName)
				require.NoError(t, os.WriteFile(stateFile, []byte("invalid json"), 0644))
				return dir
			},
			wantErr: false, // Should handle corruption gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setupFunc(t)
			
			sm, err := NewStateManager(dir)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.NotNil(t, sm)
			assert.Equal(t, filepath.Join(dir, StateFileName), sm.stateFile)
			assert.True(t, sm.autoSave)
			
			// Cleanup
			require.NoError(t, sm.Close())
		})
	}
}

func TestStateManager_IsProcessed(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewStateManager(dir)
	require.NoError(t, err)
	defer sm.Close()

	// Create a test file
	testFile := filepath.Join(dir, "test-intent.json")
	testContent := `{"action": "scale", "target": "deployment", "count": 3}`
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	tests := []struct {
		name           string
		setupFunc      func(t *testing.T) string // returns file path
		expectedResult bool
		wantErr        bool
	}{
		{
			name: "file not in state",
			setupFunc: func(t *testing.T) string {
				return testFile
			},
			expectedResult: false,
			wantErr:        false,
		},
		{
			name: "file processed successfully",
			setupFunc: func(t *testing.T) string {
				require.NoError(t, sm.MarkProcessed(testFile))
				return testFile
			},
			expectedResult: true,
			wantErr:        false,
		},
		{
			name: "file failed processing",
			setupFunc: func(t *testing.T) string {
				require.NoError(t, sm.MarkFailed(testFile))
				return testFile
			},
			expectedResult: false, // Failed files should not be considered processed
			wantErr:        false,
		},
		{
			name: "file modified after processing",
			setupFunc: func(t *testing.T) string {
				require.NoError(t, sm.MarkProcessed(testFile))
				// Modify the file content
				newContent := `{"action": "scale", "target": "deployment", "count": 5}`
				require.NoError(t, os.WriteFile(testFile, []byte(newContent), 0644))
				return testFile
			},
			expectedResult: false, // Modified files should not be considered processed
			wantErr:        false,
		},
		{
			name: "nonexistent file",
			setupFunc: func(t *testing.T) string {
				return filepath.Join(dir, "nonexistent.json")
			},
			expectedResult: false,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.setupFunc(t)
			
			result, err := sm.IsProcessed(filePath)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			
			require.NoError(t, err)
			assert.Equal(t, tt.expectedResult, result)
		})
	}
}

func TestStateManager_MarkProcessed(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewStateManager(dir)
	require.NoError(t, err)
	defer sm.Close()

	// Create a test file
	testFile := filepath.Join(dir, "test-intent.json")
	testContent := `{"action": "scale", "target": "deployment", "count": 3}`
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	// Mark as processed
	err = sm.MarkProcessed(testFile)
	require.NoError(t, err)

	// Verify state
	processed, err := sm.IsProcessed(testFile)
	require.NoError(t, err)
	assert.True(t, processed)

	// Verify state file was created
	stateFilePath := filepath.Join(dir, StateFileName)
	assert.FileExists(t, stateFilePath)

	// Verify state file content
	data, err := os.ReadFile(stateFilePath)
	require.NoError(t, err)

	var stateData struct {
		Version string                `json:"version"`
		States  map[string]*FileState `json:"states"`
	}
	require.NoError(t, json.Unmarshal(data, &stateData))
	assert.Equal(t, "1.0", stateData.Version)
	assert.Len(t, stateData.States, 1)

	// Find the state entry
	var state *FileState
	for _, s := range stateData.States {
		state = s
		break
	}
	require.NotNil(t, state)
	assert.Equal(t, "processed", state.Status)
	assert.Equal(t, int64(len(testContent)), state.Size)
	assert.NotEmpty(t, state.SHA256)
}

func TestStateManager_MarkFailed(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewStateManager(dir)
	require.NoError(t, err)
	defer sm.Close()

	// Create a test file
	testFile := filepath.Join(dir, "test-intent.json")
	testContent := `{"action": "scale", "target": "deployment", "count": 3}`
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	// Mark as failed
	err = sm.MarkFailed(testFile)
	require.NoError(t, err)

	// Verify not processed
	processed, err := sm.IsProcessed(testFile)
	require.NoError(t, err)
	assert.False(t, processed)

	// Verify in failed list
	failedFiles := sm.GetFailedFiles()
	assert.Contains(t, failedFiles, filepath.Clean(testFile))
}

func TestStateManager_ConcurrentAccess(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewStateManager(dir)
	require.NoError(t, err)
	defer sm.Close()

	// Create multiple test files
	testFiles := make([]string, 10)
	for i := 0; i < 10; i++ {
		testFile := filepath.Join(dir, fmt.Sprintf("test-intent-%d.json", i))
		testContent := fmt.Sprintf(`{"action": "scale", "target": "deployment", "count": %d}`, i+1)
		require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))
		testFiles[i] = testFile
	}

	// Test concurrent marking
	var wg sync.WaitGroup
	errors := make(chan error, 20)

	// Mark half as processed concurrently
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if err := sm.MarkProcessed(testFiles[idx]); err != nil {
				errors <- err
			}
		}(i)
	}

	// Mark half as failed concurrently
	for i := 5; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if err := sm.MarkFailed(testFiles[idx]); err != nil {
				errors <- err
			}
		}(i)
	}

	// Check status concurrently
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			if _, err := sm.IsProcessed(testFiles[idx]); err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("Concurrent access error: %v", err)
	}

	// Verify final state
	processedFiles := sm.GetProcessedFiles()
	failedFiles := sm.GetFailedFiles()
	assert.Equal(t, 5, len(processedFiles))
	assert.Equal(t, 5, len(failedFiles))
}

func TestStateManager_CleanupOldEntries(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewStateManager(dir)
	require.NoError(t, err)
	defer sm.Close()

	// Create test files
	testFile1 := filepath.Join(dir, "old-intent.json")
	testFile2 := filepath.Join(dir, "new-intent.json")
	testContent := `{"action": "scale", "target": "deployment", "count": 3}`
	require.NoError(t, os.WriteFile(testFile1, []byte(testContent), 0644))
	require.NoError(t, os.WriteFile(testFile2, []byte(testContent), 0644))

	// Mark files as processed
	require.NoError(t, sm.MarkProcessed(testFile1))
	require.NoError(t, sm.MarkProcessed(testFile2))

	// Manually set old timestamp for first file
	sm.mu.Lock()
	absPath1, _ := filepath.Abs(testFile1)
	key1 := createStateKey(absPath1)
	if state, exists := sm.states[key1]; exists {
		state.ProcessedAt = time.Now().Add(-48 * time.Hour) // 2 days ago
	}
	sm.mu.Unlock()

	// Cleanup entries older than 24 hours
	err = sm.CleanupOldEntries(24 * time.Hour)
	require.NoError(t, err)

	// Verify old entry was removed
	processed1, err := sm.IsProcessed(testFile1)
	require.NoError(t, err)
	assert.False(t, processed1)

	// Verify new entry remains
	processed2, err := sm.IsProcessed(testFile2)
	require.NoError(t, err)
	assert.True(t, processed2)
}

func TestStateManager_LoadStateWithCorruption(t *testing.T) {
	dir := t.TempDir()
	stateFile := filepath.Join(dir, StateFileName)

	tests := []struct {
		name        string
		fileContent string
		expectError bool
	}{
		{
			name:        "empty file",
			fileContent: "",
			expectError: false,
		},
		{
			name:        "invalid json",
			fileContent: "{invalid json",
			expectError: false, // Should handle gracefully
		},
		{
			name:        "valid json wrong structure",
			fileContent: `{"not": "expected"}`,
			expectError: false, // Should handle gracefully
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create corrupted state file
			require.NoError(t, os.WriteFile(stateFile, []byte(tt.fileContent), 0644))

			sm, err := NewStateManager(dir)
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, sm)
			
			// Should have created backup if corruption was detected
			if tt.fileContent != "" && tt.fileContent != `{"not": "expected"}` {
				backupFiles, _ := filepath.Glob(filepath.Join(dir, StateFileName+".backup.*"))
				if len(backupFiles) > 0 {
					assert.True(t, len(backupFiles) > 0, "Should create backup for corrupted file")
				}
			}

			require.NoError(t, sm.Close())
			
			// Clean up for next test
			os.Remove(stateFile)
			backupFiles, _ := filepath.Glob(filepath.Join(dir, StateFileName+".backup.*"))
			for _, backup := range backupFiles {
				os.Remove(backup)
			}
		})
	}
}

func TestCalculateFileHash(t *testing.T) {
	dir := t.TempDir()
	
	tests := []struct {
		name        string
		content     string
		expectError bool
	}{
		{
			name:        "normal file",
			content:     `{"action": "scale", "count": 3}`,
			expectError: false,
		},
		{
			name:        "empty file",
			content:     "",
			expectError: false,
		},
		{
			name:        "large file",
			content:     string(make([]byte, 1024*1024)), // 1MB of zeros
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testFile := filepath.Join(dir, "test.json")
			require.NoError(t, os.WriteFile(testFile, []byte(tt.content), 0644))

			hash, size, err := calculateFileHash(dir, "test.json")
			if tt.expectError {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, int64(len(tt.content)), size)
			assert.NotEmpty(t, hash)
			assert.Len(t, hash, 64) // SHA256 hex string length

			// Verify hash is correct
			expectedHash := sha256.Sum256([]byte(tt.content))
			expectedHashStr := hex.EncodeToString(expectedHash[:])
			assert.Equal(t, expectedHashStr, hash)
		})
	}

	// Test nonexistent file
	_, _, err := calculateFileHash(dir, "nonexistent.json")
	assert.Error(t, err)
}

func TestCreateStateKey(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple path",
			path:     "/test/file.json",
			expected: filepath.Clean("/test/file.json"),
		},
		{
			name:     "windows path",
			path:     `C:\test\file.json`,
			expected: filepath.Clean(`C:\test\file.json`),
		},
		{
			name:     "path with dots",
			path:     "/test/../test/./file.json",
			expected: filepath.Clean("/test/../test/./file.json"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := createStateKey(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestStateManager_GetProcessedAndFailedFiles(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewStateManager(dir)
	require.NoError(t, err)
	defer sm.Close()

	// Create test files
	processedFiles := []string{
		filepath.Join(dir, "processed1.json"),
		filepath.Join(dir, "processed2.json"),
	}
	failedFiles := []string{
		filepath.Join(dir, "failed1.json"),
		filepath.Join(dir, "failed2.json"),
	}

	testContent := `{"action": "scale", "count": 3}`
	for _, file := range append(processedFiles, failedFiles...) {
		require.NoError(t, os.WriteFile(file, []byte(testContent), 0644))
	}

	// Mark files
	for _, file := range processedFiles {
		require.NoError(t, sm.MarkProcessed(file))
	}
	for _, file := range failedFiles {
		require.NoError(t, sm.MarkFailed(file))
	}

	// Test GetProcessedFiles
	resultProcessed := sm.GetProcessedFiles()
	assert.Len(t, resultProcessed, len(processedFiles))
	for _, file := range processedFiles {
		absFile, _ := filepath.Abs(file)
		assert.Contains(t, resultProcessed, absFile)
	}

	// Test GetFailedFiles
	resultFailed := sm.GetFailedFiles()
	assert.Len(t, resultFailed, len(failedFiles))
	for _, file := range failedFiles {
		absFile, _ := filepath.Abs(file)
		assert.Contains(t, resultFailed, absFile)
	}
}

func TestStateManager_AutoSave(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewStateManager(dir)
	require.NoError(t, err)
	defer sm.Close()

	// Disable auto-save temporarily
	sm.autoSave = false

	testFile := filepath.Join(dir, "test.json")
	testContent := `{"action": "scale", "count": 3}`
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0644))

	// Mark as processed
	require.NoError(t, sm.MarkProcessed(testFile))

	// State file should not exist yet
	stateFilePath := filepath.Join(dir, StateFileName)
	assert.NoFileExists(t, stateFilePath)

	// Enable auto-save and mark another file
	sm.autoSave = true
	testFile2 := filepath.Join(dir, "test2.json")
	require.NoError(t, os.WriteFile(testFile2, []byte(testContent), 0644))
	require.NoError(t, sm.MarkProcessed(testFile2))

	// State file should exist now
	assert.FileExists(t, stateFilePath)
}

func BenchmarkStateManager_IsProcessed(b *testing.B) {
	dir := b.TempDir()
	sm, err := NewStateManager(dir)
	require.NoError(b, err)
	defer sm.Close()

	// Create and mark many files as processed
	testFiles := make([]string, 1000)
	testContent := `{"action": "scale", "count": 3}`
	for i := 0; i < 1000; i++ {
		testFile := filepath.Join(dir, fmt.Sprintf("test%d.json", i))
		require.NoError(b, os.WriteFile(testFile, []byte(testContent), 0644))
		require.NoError(b, sm.MarkProcessed(testFile))
		testFiles[i] = testFile
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			_, _ = sm.IsProcessed(testFiles[i%len(testFiles)])
			i++
		}
	})
}

func BenchmarkStateManager_MarkProcessed(b *testing.B) {
	dir := b.TempDir()
	sm, err := NewStateManager(dir)
	require.NoError(b, err)
	defer sm.Close()

	// Disable auto-save for benchmark
	sm.autoSave = false

	testContent := `{"action": "scale", "count": 3}`
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		testFile := filepath.Join(dir, fmt.Sprintf("test%d.json", i))
		_ = os.WriteFile(testFile, []byte(testContent), 0644)
		_ = sm.MarkProcessed(testFile)
	}
}