package o1

import (
	"context"
	"time"
)

// StreamFilter provides filtering capabilities for O1 streams
type StreamFilter interface {
	// Filter applies a predicate to filter stream elements
	Filter(predicate func(interface{}) bool) StreamFilter
	
	// Apply executes the filter on the given stream
	Apply(stream interface{}) interface{}
}

// StreamO1SecurityManager handles security-related operations for O1 streams
type StreamO1SecurityManager interface {
	// Encrypt secures the stream data
	Encrypt(data []byte) ([]byte, error)
	
	// Decrypt retrieves original stream data
	Decrypt(encryptedData []byte) ([]byte, error)
	
	// GenerateAccessToken creates a security token for stream access
	GenerateAccessToken(streamID string) (string, error)
	
	// ValidateAccessToken checks the validity of a stream access token
	ValidateAccessToken(token string) bool
}

// O1StreamingManager manages O1 stream lifecycle and operations
type O1StreamingManager interface {
	// CreateStream initializes a new streaming context
	CreateStream(ctx context.Context, streamConfig StreamConfig) (Stream, error)
	
	// CloseStream terminates an active stream
	CloseStream(streamID string) error
	
	// ListActiveStreams returns all currently active streams
	ListActiveStreams() []Stream
}

// StreamMultiplexer handles multiple concurrent stream processing
type StreamMultiplexer interface {
	// AddStream registers a new stream for multiplexing
	AddStream(stream Stream) error
	
	// RemoveStream stops tracking a specific stream
	RemoveStream(streamID string) error
	
	// Route distributes incoming data to appropriate streams
	Route(data interface{}) error
}

// StreamEncoder provides encoding/decoding capabilities for stream data
type StreamEncoder interface {
	// Encode transforms data into a standard format
	Encode(data interface{}) ([]byte, error)
	
	// Decode converts encoded data back to original format
	Decode(encodedData []byte, target interface{}) error
}

// StreamCompressor handles data compression for streams
type StreamCompressor interface {
	// Compress reduces the size of stream data
	Compress(data []byte) ([]byte, error)
	
	// Decompress restores compressed data to original size
	Decompress(compressedData []byte) ([]byte, error)
}

// StreamConfig extends the existing configuration with retry interval
type StreamConfig struct {
	// Existing fields from O1Config
	Endpoint       string
	AuthToken     string
	TLSConfig     TLSConfig
	
	// New RetryInterval field
	RetryInterval time.Duration
	
	// Additional stream-specific configuration
	BufferSize    int
	MaxRetries    int
}

// Stream represents a generic O1 communication stream
type Stream interface {
	// ID returns unique stream identifier
	ID() string
	
	// Send transmits data through the stream
	Send(data interface{}) error
	
	// Receive obtains incoming stream data
	Receive() (interface{}, error)
	
	// Close terminates the stream
	Close() error
	
	// Status provides current stream health
	Status() StreamStatus
}

// StreamStatus represents the current state of a stream
type StreamStatus string

const (
	StreamStatusActive   StreamStatus = "ACTIVE"
	StreamStatusClosed   StreamStatus = "CLOSED"
	StreamStatusError    StreamStatus = "ERROR"
	StreamStatusPaused   StreamStatus = "PAUSED"
)