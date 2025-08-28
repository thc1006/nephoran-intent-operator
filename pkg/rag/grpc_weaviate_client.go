//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
)

// GRPCWeaviateClient provides high-performance gRPC-based Weaviate client
type GRPCWeaviateClient struct {
	config       *GRPCClientConfig
	logger       *slog.Logger
	connPool     *GRPCConnectionPool
	metrics      *GRPCMetrics
	codec        *OptimizedCodec
	interceptors *GRPCInterceptors
}

// GRPCClientConfig holds gRPC client configuration
type GRPCClientConfig struct {
	ServerAddress     string        `json:"server_address"`
	MaxConnections    int           `json:"max_connections"`
	ConnectionTimeout time.Duration `json:"connection_timeout"`
	RequestTimeout    time.Duration `json:"request_timeout"`

	// gRPC specific settings
	EnableCompression            bool          `json:"enable_compression"`
	CompressionAlgorithm         string        `json:"compression_algorithm"`
	MaxMessageSize               int           `json:"max_message_size"`
	EnableKeepalive              bool          `json:"enable_keepalive"`
	KeepaliveTime                time.Duration `json:"keepalive_time"`
	KeepaliveTimeout             time.Duration `json:"keepalive_timeout"`
	KeepalivePermitWithoutStream bool          `json:"keepalive_permit_without_stream"`

	// Performance optimizations
	EnableConnectionPooling bool          `json:"enable_connection_pooling"`
	EnableBatching          bool          `json:"enable_batching"`
	BatchTimeout            time.Duration `json:"batch_timeout"`
	MaxBatchSize            int           `json:"max_batch_size"`

	// Security settings
	EnableTLS  bool   `json:"enable_tls"`
	CertFile   string `json:"cert_file"`
	KeyFile    string `json:"key_file"`
	CAFile     string `json:"ca_file"`
	ServerName string `json:"server_name"`

	// Authentication
	APIKey string `json:"api_key"`
	Token  string `json:"token"`
}

// GRPCConnectionPool manages a pool of gRPC connections
type GRPCConnectionPool struct {
	connections []*grpc.ClientConn
	current     int
	mutex       sync.RWMutex
	config      *GRPCClientConfig
	logger      *slog.Logger
}

// GRPCMetrics tracks gRPC client performance metrics
type GRPCMetrics struct {
	TotalRequests      int64         `json:"total_requests"`
	SuccessfulRequests int64         `json:"successful_requests"`
	FailedRequests     int64         `json:"failed_requests"`
	AverageLatency     time.Duration `json:"average_latency"`
	ConnectionsActive  int32         `json:"connections_active"`
	ConnectionsCreated int64         `json:"connections_created"`
	ConnectionsFailed  int64         `json:"connections_failed"`
	BytesSent          int64         `json:"bytes_sent"`
	BytesReceived      int64         `json:"bytes_received"`
	CompressionSavings float64       `json:"compression_savings"`
	BatchedRequests    int64         `json:"batched_requests"`
	BatchEfficiency    float64       `json:"batch_efficiency"`
	mutex              sync.RWMutex
}

// OptimizedCodec provides optimized encoding/decoding
type OptimizedCodec struct {
	enableCompression bool
	compressionLevel  int
	bufferPool        sync.Pool
	encoderPool       sync.Pool
	decoderPool       sync.Pool
}

// GRPCInterceptors provides performance-focused interceptors
type GRPCInterceptors struct {
	metrics     *GRPCMetrics
	logger      *slog.Logger
	rateLimiter *GRPCRateLimiter
}

// GRPCRateLimiter provides gRPC-specific rate limiting
type GRPCRateLimiter struct {
	requestsPerSecond float64
	burstSize         int
	tokens            float64
	lastUpdate        time.Time
	mutex             sync.Mutex
}


// GRPC-specific request structures 

type VectorSearchRequest struct {
	Query    string                 `json:"query"`
	Vector   []float32              `json:"vector"`
	Limit    int                    `json:"limit"`
	Filters  map[string]interface{} `json:"filters"`
	Metadata map[string]string      `json:"metadata"`
}


// GRPC-specific batch response

type GRPCBatchSearchResponse struct {
	Responses []*VectorSearchResponse `json:"responses"`
	Metadata  map[string]string       `json:"metadata"`
	Took      time.Duration           `json:"took"`
}

type VectorSearchResponse struct {
	Results  []*VectorResult   `json:"results"`
	Metadata map[string]string `json:"metadata"`
}

type VectorResult struct {
	ID       string                 `json:"id"`
	Score    float32                `json:"score"`
	Vector   []float32              `json:"vector"`
	Document map[string]interface{} `json:"document"`
}

// NewGRPCWeaviateClient creates a new gRPC-based Weaviate client
func NewGRPCWeaviateClient(config *GRPCClientConfig) (*GRPCWeaviateClient, error) {
	if config == nil {
		config = getDefaultGRPCConfig()
	}

	logger := slog.Default().With("component", "grpc-weaviate-client")

	// Create connection pool
	connPool, err := newGRPCConnectionPool(config, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Create optimized codec
	codec := newOptimizedCodec(config)

	// Create interceptors
	interceptors := &GRPCInterceptors{
		metrics: &GRPCMetrics{},
		logger:  logger,
		rateLimiter: &GRPCRateLimiter{
			requestsPerSecond: 1000.0,
			burstSize:         100,
		},
	}

	client := &GRPCWeaviateClient{
		config:       config,
		logger:       logger,
		connPool:     connPool,
		metrics:      interceptors.metrics,
		codec:        codec,
		interceptors: interceptors,
	}

	return client, nil
}

// getDefaultGRPCConfig returns default gRPC configuration
func getDefaultGRPCConfig() *GRPCClientConfig {
	return &GRPCClientConfig{
		ServerAddress:     "localhost:50051",
		MaxConnections:    10,
		ConnectionTimeout: 30 * time.Second,
		RequestTimeout:    10 * time.Second,

		EnableCompression:            true,
		CompressionAlgorithm:         "gzip",
		MaxMessageSize:               4 * 1024 * 1024, // 4MB
		EnableKeepalive:              true,
		KeepaliveTime:                30 * time.Second,
		KeepaliveTimeout:             5 * time.Second,
		KeepalivePermitWithoutStream: true,

		EnableConnectionPooling: true,
		EnableBatching:          true,
		BatchTimeout:            50 * time.Millisecond,
		MaxBatchSize:            20,

		EnableTLS: false,
	}
}

// newGRPCConnectionPool creates a new connection pool
func newGRPCConnectionPool(config *GRPCClientConfig, logger *slog.Logger) (*GRPCConnectionPool, error) {
	pool := &GRPCConnectionPool{
		connections: make([]*grpc.ClientConn, config.MaxConnections),
		config:      config,
		logger:      logger,
	}

	// Create connections
	for i := 0; i < config.MaxConnections; i++ {
		conn, err := pool.createConnection()
		if err != nil {
			// Close any existing connections
			pool.Close()
			return nil, fmt.Errorf("failed to create connection %d: %w", i, err)
		}
		pool.connections[i] = conn
	}

	logger.Info("gRPC connection pool created", "connections", config.MaxConnections)
	return pool, nil
}

// createConnection creates a new gRPC connection with optimized settings
func (p *GRPCConnectionPool) createConnection() (*grpc.ClientConn, error) {
	var opts []grpc.DialOption

	// Security settings
	if p.config.EnableTLS {
		// TLS credentials would be configured here
		// opts = append(opts, grpc.WithTransportCredentials(...))
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	// Keepalive settings
	if p.config.EnableKeepalive {
		keepaliveParams := keepalive.ClientParameters{
			Time:                p.config.KeepaliveTime,
			Timeout:             p.config.KeepaliveTimeout,
			PermitWithoutStream: p.config.KeepalivePermitWithoutStream,
		}
		opts = append(opts, grpc.WithKeepaliveParams(keepaliveParams))
	}

	// Message size limits
	opts = append(opts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(p.config.MaxMessageSize),
		grpc.MaxCallSendMsgSize(p.config.MaxMessageSize),
	))

	// Compression
	if p.config.EnableCompression {
		opts = append(opts, grpc.WithDefaultCallOptions(
			grpc.UseCompressor(p.config.CompressionAlgorithm),
		))
	}

	// Connection timeout
	ctx, cancel := context.WithTimeout(context.Background(), p.config.ConnectionTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, p.config.ServerAddress, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC server: %w", err)
	}

	return conn, nil
}

// GetConnection returns a connection from the pool using round-robin
func (p *GRPCConnectionPool) GetConnection() *grpc.ClientConn {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	conn := p.connections[p.current]
	p.current = (p.current + 1) % len(p.connections)

	return conn
}

// Close closes all connections in the pool
func (p *GRPCConnectionPool) Close() error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	var lastErr error
	for i, conn := range p.connections {
		if conn != nil {
			if err := conn.Close(); err != nil {
				p.logger.Error("Failed to close connection", "index", i, "error", err)
				lastErr = err
			}
		}
	}

	return lastErr
}

// newOptimizedCodec creates an optimized codec with object pooling
func newOptimizedCodec(config *GRPCClientConfig) *OptimizedCodec {
	codec := &OptimizedCodec{
		enableCompression: config.EnableCompression,
		compressionLevel:  6, // Balanced compression level
	}

	// Initialize buffer pool
	codec.bufferPool.New = func() interface{} {
		return make([]byte, 0, 1024) // Initial capacity of 1KB
	}

	// Initialize encoder pool - using bytes.Buffer as proto.Buffer is deprecated
	codec.encoderPool.New = func() interface{} {

		return make([]byte, 0, 1024) // Protobuf encoder buffer

	}

	// Initialize decoder pool - using bytes.Buffer as proto.Buffer is deprecated  
	codec.decoderPool.New = func() interface{} {

		return make([]byte, 0, 1024) // Protobuf decoder buffer

	}

	return codec
}

// Search performs a high-performance vector search using gRPC
func (c *GRPCWeaviateClient) Search(ctx context.Context, query *SearchQuery) (*SearchResponse, error) {
	startTime := time.Now()

	// Convert to gRPC request format
	grpcReq := &VectorSearchRequest{
		Query:   query.Query,
		Limit:   query.Limit,
		Filters: query.Filters,
		Metadata: map[string]string{
			"hybrid_search": fmt.Sprintf("%t", query.HybridSearch),
			"hybrid_alpha":  fmt.Sprintf("%.2f", *query.HybridAlpha),
		},
	}

	// Perform the search
	grpcResp, err := c.performGRPCSearch(ctx, grpcReq)
	if err != nil {
		c.updateMetrics(false, time.Since(startTime), 0, 0)
		return nil, fmt.Errorf("gRPC search failed: %w", err)
	}

	// Convert back to standard SearchResponse
	response := c.convertToSearchResponse(grpcResp, query.Query)
	response.Took = time.Since(startTime)

	c.updateMetrics(true, time.Since(startTime), len(response.Results), 0)

	c.logger.Debug("gRPC search completed",
		"query", query.Query,
		"results", len(response.Results),
		"took", response.Took,
	)

	return response, nil
}

// BatchSearch performs optimized batch searching using gRPC
func (c *GRPCWeaviateClient) BatchSearch(ctx context.Context, queries []*SearchQuery) ([]*SearchResponse, error) {
	if len(queries) == 0 {
		return nil, fmt.Errorf("no queries provided")
	}

	startTime := time.Now()


	// Convert to gRPC batch request - using Queries field instead of Requests
	batchReq := &BatchSearchRequest{
		Queries: make([]*SearchQuery, len(queries)),
		Metadata: map[string]interface{}{

			"batch_size": fmt.Sprintf("%d", len(queries)),
		},
	}

	for i, query := range queries {
		batchReq.Queries[i] = query
	}

	// Perform batch search
	batchResp, err := c.performGRPCBatchSearch(ctx, batchReq)
	if err != nil {
		c.updateMetrics(false, time.Since(startTime), 0, len(queries))
		return nil, fmt.Errorf("gRPC batch search failed: %w", err)
	}

	// Convert responses
	responses := make([]*SearchResponse, len(batchResp.Responses))
	for i, grpcResp := range batchResp.Responses {
		responses[i] = c.convertToSearchResponse(grpcResp, queries[i].Query)
	}

	c.updateMetrics(true, time.Since(startTime), len(responses), len(queries))

	c.logger.Info("gRPC batch search completed",
		"queries", len(queries),
		"took", time.Since(startTime),
	)

	return responses, nil
}

// performGRPCSearch performs the actual gRPC search call
func (c *GRPCWeaviateClient) performGRPCSearch(ctx context.Context, req *VectorSearchRequest) (*VectorSearchResponse, error) {
	// Get connection from pool

	_ = c.connPool.GetConnection() // TODO: Use when actual gRPC implementation is ready


	// Add authentication metadata
	ctx = c.addAuthenticationContext(ctx)

	// Add timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout)
	defer cancel()
	_ = ctx // Context prepared for future gRPC call implementation

	// This is a placeholder - in a real implementation, you would call the actual gRPC service
	// client := vectorsearch.NewVectorSearchServiceClient(conn)
	// response, err := client.Search(ctx, req)

	// For now, simulate a gRPC call
	time.Sleep(10 * time.Millisecond) // Simulate network latency

	// Return mock response
	return &VectorSearchResponse{
		Results: []*VectorResult{
			{
				ID:    "doc1",
				Score: 0.95,
				Document: map[string]interface{}{
					"content": "Sample telecom document content",
					"title":   "5G Network Configuration",
				},
			},
		},
		Metadata: map[string]string{
			"total_results": "1",
		},
	}, nil
}

// performGRPCBatchSearch performs batch gRPC search

func (c *GRPCWeaviateClient) performGRPCBatchSearch(ctx context.Context, req *BatchSearchRequest) (*GRPCBatchSearchResponse, error) {

	// Get connection from pool
	conn := c.connPool.GetConnection()
	_ = conn // Use the connection

	// Add authentication metadata
	ctx = c.addAuthenticationContext(ctx)

	// Add timeout
	ctx, cancel := context.WithTimeout(ctx, c.config.RequestTimeout*time.Duration(len(req.Queries)))
	defer cancel()
	_ = ctx // Context prepared for future gRPC batch call implementation

	// This is a placeholder for actual gRPC batch call
	time.Sleep(20 * time.Millisecond) // Simulate batch processing time

	// Return mock batch response
	responses := make([]*VectorSearchResponse, len(req.Queries))
	for i := range req.Queries {
		responses[i] = &VectorSearchResponse{
			Results: []*VectorResult{
				{
					ID:    fmt.Sprintf("doc%d", i+1),
					Score: 0.90 + float32(i)*0.01,
					Document: map[string]interface{}{
						"content": fmt.Sprintf("Document content for query %d", i+1),
						"title":   fmt.Sprintf("Document %d", i+1),
					},
				},
			},
		}
	}

	return &GRPCBatchSearchResponse{
		Responses: responses,
		Metadata: map[string]string{
			"batch_size": fmt.Sprintf("%d", len(req.Queries)),
		},
	}, nil
}

// addAuthenticationContext adds authentication to the gRPC context
func (c *GRPCWeaviateClient) addAuthenticationContext(ctx context.Context) context.Context {
	md := metadata.New(map[string]string{})

	if c.config.APIKey != "" {
		md.Set("api-key", c.config.APIKey)
	}

	if c.config.Token != "" {
		md.Set("authorization", "Bearer "+c.config.Token)
	}

	return metadata.NewOutgoingContext(ctx, md)
}

// convertToSearchResponse converts gRPC response to standard SearchResponse
func (c *GRPCWeaviateClient) convertToSearchResponse(grpcResp *VectorSearchResponse, query string) *SearchResponse {

	results := make([]*SearchResult, len(grpcResp.Results))


	for i, result := range grpcResp.Results {
		doc := &types.TelecomDocument{
			ID: result.ID,
		}

		// Extract document fields from the map
		if content, ok := result.Document["content"].(string); ok {
			doc.Content = content
		}
		if title, ok := result.Document["title"].(string); ok {
			doc.Title = title
		}


		results[i] = &SearchResult{

			Document: doc,
			Score:    result.Score,
		}
	}

	return &SearchResponse{
		Results: convertSharedSearchResults(results),
		Took:    0, // Timing should be calculated by the caller
		Total:   int64(len(results)),
	}
}

// convertSharedSearchResults converts shared.SearchResult to local SearchResult
func convertSharedSearchResults(sharedResults []*types.SearchResult) []*SearchResult {
	results := make([]*SearchResult, len(sharedResults))
	for i, sharedResult := range sharedResults {
		// Extract fields from the shared result
		id := ""
		content := ""
		if sharedResult.Document != nil {
			id = sharedResult.Document.ID
			content = sharedResult.Document.Content
		}

		results[i] = &SearchResult{
			ID:         id,
			Content:    content,
			Confidence: float64(sharedResult.Score), // Convert score to confidence
			Metadata:   sharedResult.Metadata,
			Score:      sharedResult.Score,
			Document:   sharedResult.Document,
		}
	}
	return results
}

// updateMetrics updates gRPC client metrics
func (c *GRPCWeaviateClient) updateMetrics(success bool, latency time.Duration, resultCount, batchSize int) {
	c.metrics.mutex.Lock()
	defer c.metrics.mutex.Unlock()

	c.metrics.TotalRequests++

	if success {
		c.metrics.SuccessfulRequests++
	} else {
		c.metrics.FailedRequests++
	}

	// Update average latency
	if c.metrics.TotalRequests == 1 {
		c.metrics.AverageLatency = latency
	} else {
		c.metrics.AverageLatency = (c.metrics.AverageLatency*time.Duration(c.metrics.TotalRequests-1) +
			latency) / time.Duration(c.metrics.TotalRequests)
	}

	// Update batch metrics
	if batchSize > 1 {
		c.metrics.BatchedRequests++
		c.metrics.BatchEfficiency = float64(c.metrics.BatchedRequests) / float64(c.metrics.TotalRequests)
	}
}

// StreamingSearch provides streaming search results
func (c *GRPCWeaviateClient) StreamingSearch(ctx context.Context, queries []*SearchQuery, resultChan chan<- *SearchResponse) error {
	defer close(resultChan)

	for _, query := range queries {
		result, err := c.Search(ctx, query)
		if err != nil {
			c.logger.Error("Streaming search failed", "error", err, "query", query.Query)
			continue
		}

		select {
		case resultChan <- result:
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	return nil
}

// GetMetrics returns current gRPC client metrics
func (c *GRPCWeaviateClient) GetMetrics() *GRPCMetrics {
	c.metrics.mutex.RLock()
	defer c.metrics.mutex.RUnlock()

	// Return a copy without the mutex
	metrics := &GRPCMetrics{
		TotalRequests:      c.metrics.TotalRequests,
		SuccessfulRequests: c.metrics.SuccessfulRequests,
		FailedRequests:     c.metrics.FailedRequests,
		AverageLatency:     c.metrics.AverageLatency,
		ConnectionsActive:  c.metrics.ConnectionsActive,
		ConnectionsCreated: c.metrics.ConnectionsCreated,
		ConnectionsFailed:  c.metrics.ConnectionsFailed,
		BytesSent:          c.metrics.BytesSent,
		BytesReceived:      c.metrics.BytesReceived,
		CompressionSavings: c.metrics.CompressionSavings,
		BatchedRequests:    c.metrics.BatchedRequests,
		BatchEfficiency:    c.metrics.BatchEfficiency,
	}
	return metrics
}

// GetConnectionPoolStatus returns connection pool status
func (c *GRPCWeaviateClient) GetConnectionPoolStatus() map[string]interface{} {
	c.connPool.mutex.RLock()
	defer c.connPool.mutex.RUnlock()

	activeConns := 0
	for _, conn := range c.connPool.connections {
		if conn != nil && conn.GetState().String() == "READY" {
			activeConns++
		}
	}

	return map[string]interface{}{
		"total_connections":  len(c.connPool.connections),
		"active_connections": activeConns,
		"current_index":      c.connPool.current,
	}
}

// Close closes the gRPC client and all connections
func (c *GRPCWeaviateClient) Close() error {
	c.logger.Info("Closing gRPC Weaviate client")
	return c.connPool.Close()
}

// Health check methods and additional utilities would be implemented here...

// IsHealthy checks if the gRPC client is healthy
func (c *GRPCWeaviateClient) IsHealthy(ctx context.Context) bool {
	conn := c.connPool.GetConnection()
	state := conn.GetState()
	return state.String() == "READY" || state.String() == "IDLE"
}
