package security

import (
	"errors"
	"fmt"
	"io"
	"os"
)

const (
	// MaxJSONBytes defines the maximum allowed JSON size (1MB).
	// This limit prevents memory exhaustion attacks from maliciously large JSON payloads
	// while still allowing reasonable intent files for O-RAN network configurations.
	MaxJSONBytes int64 = 1 * 1024 * 1024 // 1MB
)

// ErrMaxSizeExceeded is returned when JSON content exceeds the maximum allowed size.

var ErrMaxSizeExceeded = errors.New("exceeds maximum JSON size limit")

// ValidateAndLimitJSON validates and reads JSON content with size limits.
// This function provides defense against JSON bombs and memory exhaustion attacks.
//
// Security features:
// - For file handles: Uses Stat() to pre-check size before reading (fast rejection)
// - For streams: Uses counting reader that stops at limit with proper error
// - Returns raw bytes for subsequent JSON parsing with size guarantees
//
// Parameters:
//   - r: The input reader (file, stream, etc.)
//   - max: Maximum allowed size in bytes
//
// Returns:
//   - []byte: The validated JSON content (guaranteed to be <= max bytes)
//   - error: ErrMaxSizeExceeded if size limit exceeded, other errors for I/O issues

func ValidateAndLimitJSON(r io.Reader, max int64) ([]byte, error) {
	// If reader is a file, use Stat() for fast size check before reading
	if file, ok := r.(*os.File); ok {
		stat, err := file.Stat()
		if err != nil {
			return nil, fmt.Errorf("failed to stat file: %w", err)
		}

		// Fast rejection for oversized files
		if stat.Size() > max {
			return nil, fmt.Errorf("file size %d bytes %w (%d bytes)",
				stat.Size(), ErrMaxSizeExceeded, max)
		}
	}

	// Use a counting reader to track bytes read and enforce limit
	counter := &countingReader{r: r, max: max}

	// Read all content with size limit enforcement
	data, err := io.ReadAll(counter)
	if err != nil {
		// Check if we hit the size limit
		if counter.exceeded {
			return nil, fmt.Errorf("stream %w (%d bytes)", ErrMaxSizeExceeded, max)
		}
		return nil, fmt.Errorf("failed to read JSON content: %w", err)
	}

	return data, nil
}

// countingReader wraps an io.Reader and tracks bytes read, enforcing a maximum limit.
// When the limit is exceeded, it returns ErrMaxSizeExceeded instead of EOF.
type countingReader struct {
	r        io.Reader
	max      int64
	count    int64
	exceeded bool
	eof      bool // Track if we've hit EOF naturally
}

// Read implements io.Reader interface with size limit enforcement.
func (c *countingReader) Read(p []byte) (int, error) {
	// If we've already hit EOF naturally, return EOF
	if c.eof {
		return 0, io.EOF
	}

	// Check if we would exceed the limit with this read
	remaining := c.max - c.count

	if remaining <= 0 {
		// We've already read the maximum allowed bytes
		// Check if there would be more data by attempting a read
		var dummy [1]byte
		n, err := c.r.Read(dummy[:])
		if err == io.EOF {
			// Natural EOF, we're done
			c.eof = true
			return 0, io.EOF
		}
		if n > 0 {
			// There's more data, which would exceed the limit
			c.exceeded = true
			return 0, ErrMaxSizeExceeded
		}
		// Some other error
		return 0, err
	}

	// Limit the read size to remaining capacity
	if int64(len(p)) > remaining {
		p = p[:remaining]
	}

	n, err := c.r.Read(p)
	c.count += int64(n)

	// Handle EOF case
	if err == io.EOF {
		c.eof = true
		return n, err
	}

	// Check if we've exceeded the limit after this read
	if c.count > c.max {
		c.exceeded = true
		return n, ErrMaxSizeExceeded
	}

	return n, err
}
