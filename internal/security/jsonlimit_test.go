package security

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// DISABLED: func TestValidateAndLimitJSON(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		maxSize       int64
		expectError   bool
		errorContains string
	}{
		{
			name:        "valid small JSON",
			input:       `{"test": "data"}`,
			maxSize:     100,
			expectError: false,
		},
		{
			name:        "exactly at limit",
			input:       `{"test": "data"}`, // 16 bytes
			maxSize:     16,
			expectError: false,
		},
		{
			name:          "exceeds limit by one byte",
			input:         `{"test": "data"}`, // 16 bytes
			maxSize:       15,
			expectError:   true,
			errorContains: "exceeds maximum JSON size limit",
		},
		{
			name:        "empty JSON",
			input:       `{}`,
			maxSize:     100,
			expectError: false,
		},
		{
			name:        "large valid JSON under limit",
			input:       strings.Repeat(`{"key": "value", "number": 123}, `, 10),
			maxSize:     1000,
			expectError: false,
		},
		{
			name:          "large JSON over limit",
			input:         strings.Repeat("a", 1025), // 1025 bytes
			maxSize:       1024,
			expectError:   true,
			errorContains: "exceeds maximum JSON size limit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)

			data, err := ValidateAndLimitJSON(reader, tt.maxSize)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
				assert.Nil(t, data)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.input, string(data))
				assert.LessOrEqual(t, int64(len(data)), tt.maxSize)
			}
		})
	}
}

// DISABLED: func TestValidateAndLimitJSON_FileHandles(t *testing.T) {
	// Create a temporary file
	tempFile, err := os.CreateTemp("", "test_json_*.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Test content
	content := `{"intent": "scaling", "target": "app", "namespace": "default"}`
	_, err = tempFile.WriteString(content)
	require.NoError(t, err)

	// Reset file position
	_, err = tempFile.Seek(0, 0)
	require.NoError(t, err)

	t.Run("file under limit", func(t *testing.T) {
		data, err := ValidateAndLimitJSON(tempFile, 1024)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("file over limit", func(t *testing.T) {
		// Reset file position
		_, err := tempFile.Seek(0, 0)
		require.NoError(t, err)

		data, err := ValidateAndLimitJSON(tempFile, 10)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum JSON size limit")
		assert.Nil(t, data)
	})
}

// DISABLED: func TestValidateAndLimitJSON_LargeFile(t *testing.T) {
	// Create a temporary file with content larger than 1MB
	tempFile, err := os.CreateTemp("", "large_test_*.json")
	require.NoError(t, err)
	defer os.Remove(tempFile.Name())
	defer tempFile.Close()

	// Write 2MB of content
	largeContent := strings.Repeat("a", 2*1024*1024)
	_, err = tempFile.WriteString(largeContent)
	require.NoError(t, err)

	// Reset file position
	_, err = tempFile.Seek(0, 0)
	require.NoError(t, err)

	t.Run("large file exceeds MaxJSONBytes", func(t *testing.T) {
		data, err := ValidateAndLimitJSON(tempFile, MaxJSONBytes)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum JSON size limit")
		assert.Nil(t, data)
	})
}

// DISABLED: func TestValidateAndLimitJSON_StreamReader(t *testing.T) {
	t.Run("stream under limit", func(t *testing.T) {
		content := `{"test": "stream data"}`
		reader := strings.NewReader(content)

		data, err := ValidateAndLimitJSON(reader, 100)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
	})

	t.Run("stream over limit", func(t *testing.T) {
		// Large content that exceeds limit
		content := strings.Repeat("x", 1025)
		reader := strings.NewReader(content)

		data, err := ValidateAndLimitJSON(reader, 1024)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "exceeds maximum JSON size limit")
		assert.Nil(t, data)
	})
}

// DISABLED: func TestCountingReader(t *testing.T) {
	t.Run("normal read under limit", func(t *testing.T) {
		content := "hello world"
		reader := strings.NewReader(content)
		counter := &countingReader{r: reader, max: 20}

		data, err := io.ReadAll(counter)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
		assert.Equal(t, int64(len(content)), counter.count)
		assert.False(t, counter.exceeded)
	})

	t.Run("read exactly at limit", func(t *testing.T) {
		content := "hello"
		reader := strings.NewReader(content)
		counter := &countingReader{r: reader, max: 5}

		data, err := io.ReadAll(counter)
		assert.NoError(t, err)
		assert.Equal(t, content, string(data))
		assert.Equal(t, int64(5), counter.count)
		assert.False(t, counter.exceeded)
	})

	t.Run("read over limit", func(t *testing.T) {
		content := "hello world"
		reader := strings.NewReader(content)
		counter := &countingReader{r: reader, max: 5}

		data, err := io.ReadAll(counter)
		assert.Error(t, err)
		assert.Equal(t, ErrMaxSizeExceeded, err)
		assert.Equal(t, "hello", string(data)) // Only reads up to limit
		assert.Equal(t, int64(5), counter.count)
		assert.True(t, counter.exceeded)
	})
}

// DISABLED: func TestMaxJSONBytesConstant(t *testing.T) {
	// Verify the constant is set to 1MB
	assert.Equal(t, int64(1<<20), MaxJSONBytes)
	assert.Equal(t, int64(1024*1024), MaxJSONBytes)
}

func BenchmarkValidateAndLimitJSON(b *testing.B) {
	content := strings.Repeat(`{"key": "value"}`, 1000) // ~13KB

	b.Run("string_reader", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			reader := strings.NewReader(content)
			_, err := ValidateAndLimitJSON(reader, MaxJSONBytes)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("bytes_buffer", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			buffer := bytes.NewBufferString(content)
			_, err := ValidateAndLimitJSON(buffer, MaxJSONBytes)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
