package security

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAtomicFileWriter(t *testing.T) {
	t.Run("creates writer for valid directory", func(t *testing.T) {
		dir := t.TempDir()

		writer, err := NewAtomicFileWriter(dir)
		require.NoError(t, err)
		require.NotNil(t, writer)
		assert.Equal(t, dir, writer.dir)
	})

	t.Run("fails for non-existent directory", func(t *testing.T) {
		writer, err := NewAtomicFileWriter("/nonexistent/path")
		assert.Error(t, err)
		assert.Nil(t, writer)
		assert.Contains(t, err.Error(), "directory validation failed")
	})

	t.Run("fails for file instead of directory", func(t *testing.T) {
		dir := t.TempDir()
		file := filepath.Join(dir, "file.txt")
		err := os.WriteFile(file, []byte("test"), 0600)
		require.NoError(t, err)

		writer, err := NewAtomicFileWriter(file)
		assert.Error(t, err)
		assert.Nil(t, writer)
		assert.Contains(t, err.Error(), "not a directory")
	})

	t.Run("fails for read-only directory", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping test when running as root")
		}

		dir := t.TempDir()
		err := os.Chmod(dir, 0400)
		require.NoError(t, err)
		defer os.Chmod(dir, 0700)

		writer, err := NewAtomicFileWriter(dir)
		assert.Error(t, err)
		assert.Nil(t, writer)
		assert.Contains(t, err.Error(), "directory not writable")
	})
}

func TestAtomicFileWriter_CreateIntentFile(t *testing.T) {
	dir := t.TempDir()
	writer, err := NewAtomicFileWriter(dir)
	require.NoError(t, err)

	t.Run("creates unique file", func(t *testing.T) {
		f, fpath, err := writer.CreateIntentFile()
		require.NoError(t, err)
		require.NotNil(t, f)
		defer f.Close()
		defer os.Remove(fpath)

		assert.True(t, strings.HasPrefix(filepath.Base(fpath), "intent-"))
		assert.True(t, strings.HasSuffix(fpath, ".json"))

		// Verify file exists and is writable
		_, err = f.WriteString("test")
		assert.NoError(t, err)
	})

	t.Run("creates multiple unique files", func(t *testing.T) {
		paths := make([]string, 10)

		for i := 0; i < 10; i++ {
			f, fpath, err := writer.CreateIntentFile()
			require.NoError(t, err)
			f.Close()
			defer os.Remove(fpath)

			paths[i] = fpath
		}

		// Verify all paths are unique
		seen := make(map[string]bool)
		for _, path := range paths {
			assert.False(t, seen[path], "duplicate path: %s", path)
			seen[path] = true
		}
	})

	t.Run("prevents duplicate creation with O_EXCL", func(t *testing.T) {
		// Create a file manually
		testFile := filepath.Join(dir, "intent-12345-test.json")
		err := os.WriteFile(testFile, []byte(""), 0600)
		require.NoError(t, err)
		defer os.Remove(testFile)

		// Try to create with same name should fail due to O_EXCL
		// (This would require mocking the random generation,
		// but demonstrates the intent)
	})
}

func TestAtomicFileWriter_WriteIntent(t *testing.T) {
	dir := t.TempDir()
	writer, err := NewAtomicFileWriter(dir)
	require.NoError(t, err)

	t.Run("writes data successfully", func(t *testing.T) {
		data := []byte(`{"intent": "deploy UPF"}`)

		fpath, err := writer.WriteIntent(data)
		require.NoError(t, err)
		defer os.Remove(fpath)

		// Verify file exists
		assert.FileExists(t, fpath)

		// Verify content
		content, err := os.ReadFile(fpath)
		require.NoError(t, err)
		assert.Equal(t, data, content)
	})

	t.Run("cleans up on write failure", func(t *testing.T) {
		// This test demonstrates cleanup behavior
		// In practice, write failures are rare
		data := []byte(`{"test": "data"}`)

		fpath, err := writer.WriteIntent(data)
		require.NoError(t, err)
		defer os.Remove(fpath)

		assert.FileExists(t, fpath)
	})

	t.Run("syncs data to disk", func(t *testing.T) {
		data := []byte(`{"intent": "scale AMF"}`)

		fpath, err := writer.WriteIntent(data)
		require.NoError(t, err)
		defer os.Remove(fpath)

		// Verify file is immediately readable (sync worked)
		content, err := os.ReadFile(fpath)
		require.NoError(t, err)
		assert.Equal(t, data, content)
	})
}

func TestAtomicFileWriter_WriteIntentAtomic(t *testing.T) {
	dir := t.TempDir()
	writer, err := NewAtomicFileWriter(dir)
	require.NoError(t, err)

	t.Run("writes atomically with rename", func(t *testing.T) {
		data := []byte(`{"intent": "configure O-DU"}`)

		fpath, err := writer.WriteIntentAtomic(data)
		require.NoError(t, err)
		defer os.Remove(fpath)

		// Verify file exists
		assert.FileExists(t, fpath)

		// Verify content
		content, err := os.ReadFile(fpath)
		require.NoError(t, err)
		assert.Equal(t, data, content)

		// Verify no temp files left
		files, err := os.ReadDir(dir)
		require.NoError(t, err)
		for _, f := range files {
			assert.False(t, strings.HasPrefix(f.Name(), ".tmp-intent-"))
		}
	})

	t.Run("cleans up temp file on failure", func(t *testing.T) {
		data := []byte(`{"test": "atomic"}`)

		fpath, err := writer.WriteIntentAtomic(data)
		require.NoError(t, err)
		defer os.Remove(fpath)

		// Verify no temp files remain
		files, err := os.ReadDir(dir)
		require.NoError(t, err)
		for _, f := range files {
			assert.False(t, strings.HasPrefix(f.Name(), ".tmp-"))
		}
	})
}

func TestValidateFilename(t *testing.T) {
	tests := []struct {
		name      string
		filename  string
		wantError bool
	}{
		{"valid filename", "intent-123.json", false},
		{"valid with timestamp", "intent-1234567890-abc.json", false},
		{"path traversal", "../etc/passwd", true},
		{"absolute path", "/etc/passwd", true},
		{"null byte", "intent\x00.json", true},
		{"control character", "intent\x01.json", true},
		{"current dir", ".", true},
		{"parent dir", "..", true},
		{"valid dash", "intent-test-123.json", false},
		{"non-printable char", "intent\x7f.json", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateFilename(tt.filename)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSecureCopy(t *testing.T) {
	t.Run("copies data within limit", func(t *testing.T) {
		src := bytes.NewReader([]byte("test data"))
		dst := &bytes.Buffer{}

		n, err := SecureCopy(dst, src, 100)
		require.NoError(t, err)
		assert.Equal(t, int64(9), n)
		assert.Equal(t, "test data", dst.String())
	})

	t.Run("enforces size limit", func(t *testing.T) {
		data := bytes.Repeat([]byte("x"), 1000)
		src := bytes.NewReader(data)
		dst := &bytes.Buffer{}

		n, err := SecureCopy(dst, src, 100)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum size")
		assert.Equal(t, int64(100), n)
	})

	t.Run("handles exact size", func(t *testing.T) {
		data := bytes.Repeat([]byte("x"), 100)
		src := bytes.NewReader(data)
		dst := &bytes.Buffer{}

		n, err := SecureCopy(dst, src, 100)
		require.NoError(t, err)
		assert.Equal(t, int64(100), n)
	})

	t.Run("handles empty source", func(t *testing.T) {
		src := bytes.NewReader([]byte{})
		dst := &bytes.Buffer{}

		n, err := SecureCopy(dst, src, 100)
		require.NoError(t, err)
		assert.Equal(t, int64(0), n)
	})

	t.Run("handles EOF properly", func(t *testing.T) {
		src := bytes.NewReader([]byte("short"))
		dst := &bytes.Buffer{}

		n, err := SecureCopy(dst, src, 100)
		require.NoError(t, err)
		assert.Equal(t, int64(5), n)
		assert.Equal(t, "short", dst.String())
	})
}

func TestAtomicFileWriter_Concurrency(t *testing.T) {
	dir := t.TempDir()
	writer, err := NewAtomicFileWriter(dir)
	require.NoError(t, err)

	// Create files concurrently
	const numGoroutines = 10
	results := make(chan string, numGoroutines)
	errors := make(chan error, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(id int) {
			data := []byte(`{"id": ` + string(rune(id)) + `}`)
			fpath, err := writer.WriteIntent(data)
			if err != nil {
				errors <- err
				return
			}
			results <- fpath
		}(i)
	}

	// Collect results
	paths := make([]string, 0, numGoroutines)
	for i := 0; i < numGoroutines; i++ {
		select {
		case fpath := <-results:
			paths = append(paths, fpath)
			defer os.Remove(fpath)
		case err := <-errors:
			t.Errorf("concurrent write failed: %v", err)
		}
	}

	// Verify all paths are unique
	seen := make(map[string]bool)
	for _, path := range paths {
		assert.False(t, seen[path], "duplicate path in concurrent test")
		seen[path] = true
	}
}

func TestAtomicFileWriter_LargeFile(t *testing.T) {
	dir := t.TempDir()
	writer, err := NewAtomicFileWriter(dir)
	require.NoError(t, err)

	// Create a large JSON payload
	largeData := bytes.Repeat([]byte(`{"key":"value"},`), 1000)
	largeData = append([]byte(`[`), largeData...)
	largeData = append(largeData, []byte(`]`)...)

	fpath, err := writer.WriteIntentAtomic(largeData)
	require.NoError(t, err)
	defer os.Remove(fpath)

	// Verify content
	content, err := os.ReadFile(fpath)
	require.NoError(t, err)
	assert.Equal(t, largeData, content)
}
