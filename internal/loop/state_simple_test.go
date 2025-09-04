package loop

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStateManager_BasicOperations(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewStateManager(dir)
	require.NoError(t, err)
	defer sm.Close() // #nosec G307 - Error handled in defer

	// Create a test file
	testFile := filepath.Join(dir, "test-intent.json")
	testContent := `{"action": "scale", "target": "deployment", "count": 3}`
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0o644))

	// Initially should not be processed
	processed, err := sm.IsProcessed(testFile)
	require.NoError(t, err)
	assert.False(t, processed)

	// Mark as processed
	err = sm.MarkProcessed(testFile)
	require.NoError(t, err)

	// Now should be processed
	processed, err = sm.IsProcessed(testFile)
	require.NoError(t, err)
	assert.True(t, processed)

	// Mark as failed
	err = sm.MarkFailed(testFile)
	require.NoError(t, err)

	// Should not be considered processed after marking as failed
	processed, err = sm.IsProcessed(testFile)
	require.NoError(t, err)
	assert.False(t, processed)
}

func TestStateManager_FileModification(t *testing.T) {
	dir := t.TempDir()
	sm, err := NewStateManager(dir)
	require.NoError(t, err)
	defer sm.Close() // #nosec G307 - Error handled in defer

	// Create and mark file as processed
	testFile := filepath.Join(dir, "test-intent.json")
	testContent := `{"action": "scale", "target": "deployment", "count": 3}`
	require.NoError(t, os.WriteFile(testFile, []byte(testContent), 0o644))

	err = sm.MarkProcessed(testFile)
	require.NoError(t, err)

	// Should be processed
	processed, err := sm.IsProcessed(testFile)
	require.NoError(t, err)
	assert.True(t, processed)

	// Modify file content
	newContent := `{"action": "scale", "target": "deployment", "count": 5}`
	require.NoError(t, os.WriteFile(testFile, []byte(newContent), 0o644))

	// Should no longer be considered processed
	processed, err = sm.IsProcessed(testFile)
	require.NoError(t, err)
	assert.False(t, processed)
}
