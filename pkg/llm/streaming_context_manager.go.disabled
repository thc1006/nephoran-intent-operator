package llm

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// StreamingContextManager manages context preservation during streaming with TTL-based cleanup
type StreamingContextManager struct {
	tokenManager      *TokenManager
	contexts          map[string]*StreamingContext
	config            *StreamingContextConfig
	logger            *slog.Logger
	metrics           *ContextManagerMetrics
	cleanupTicker     *time.Ticker
	mutex             sync.RWMutex
	closed            bool
}

// StreamingContextConfig holds configuration for context management
type StreamingContextConfig struct {
	// TTL settings
	DefaultTTL        time.Duration `json:"default_ttl"`
	MaxTTL            time.Duration `json:"max_ttl"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`
	
	// Context preservation
	MaxContextSize    int           `json:"max_context_size"`
	ContextOverhead   time.Duration `json:"context_overhead"`
	PreservationRatio float64       `json:"preservation_ratio"`
	
	// Performance settings
	MaxConcurrentContexts int `json:"max_concurrent_contexts"`
	EnableCompression     bool `json:"enable_compression"`
	
	// Memory management
	MaxMemoryUsage    int64 `json:"max_memory_usage"` // bytes
	MemoryCleanupThreshold float64 `json:"memory_cleanup_threshold"`
}

// ContextManagerMetrics tracks context manager performance
type ContextManagerMetrics struct {
	ActiveContexts        int64     `json:"active_contexts"`
	TotalContextsCreated  int64     `json:"total_contexts_created"`
	TotalContextsExpired  int64     `json:"total_contexts_expired"`
	TotalContextsEvicted  int64     `json:"total_contexts_evicted"`
	AverageContextSize    int       `json:"average_context_size"`
	MemoryUsage          int64     `json:"memory_usage"`
	CleanupOperations    int64     `json:"cleanup_operations"`
	ContextHits          int64     `json:"context_hits"`
	ContextMisses        int64     `json:"context_misses"`
	LastCleanup          time.Time `json:"last_cleanup"`
	LastUpdated          time.Time `json:"last_updated"`
	mutex                sync.RWMutex
}

// StreamingContext represents a context maintained during streaming
type StreamingContext struct {
	ID            string                 `json:"id"`
	Content       string                 `json:"content"`
	TokenCount    int                    `json:"token_count"`
	CreatedAt     time.Time              `json:"created_at"`
	LastAccessed  time.Time              `json:"last_accessed"`
	TTL           time.Duration          `json:"ttl"`
	AccessCount   int64                  `json:"access_count"`
	Size          int64                  `json:"size"` // bytes
	Metadata      map[string]interface{} `json:"metadata"`
	IsCompressed  bool                   `json:"is_compressed"`
	mutex         sync.RWMutex
}

// ContextRequest represents a request for context management
type ContextRequest struct {
	ID          string        `json:"id"`
	Content     string        `json:"content"`
	ModelName   string        `json:"model_name"`
	TTL         time.Duration `json:"ttl,omitempty"`
	Priority    int           `json:"priority,omitempty"`
	Compress    bool          `json:"compress,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ContextResponse represents the response from context operations
type ContextResponse struct {
	Context       *StreamingContext `json:"context"`
	Found         bool              `json:"found"`
	Created       bool              `json:"created"`
	Updated       bool              `json:"updated"`
	TokenCount    int               `json:"token_count"`
	ProcessingTime time.Duration    `json:"processing_time"`
	FromCache     bool              `json:"from_cache"`
}

// NewStreamingContextManager creates a new streaming context manager
func NewStreamingContextManager(tokenManager *TokenManager, contextOverhead time.Duration) *StreamingContextManager {
	config := &StreamingContextConfig{
		DefaultTTL:                5 * time.Minute,
		MaxTTL:                   30 * time.Minute,
		CleanupInterval:          time.Minute,
		MaxContextSize:           10000,
		ContextOverhead:          contextOverhead,
		PreservationRatio:        0.8,
		MaxConcurrentContexts:    1000,
		EnableCompression:        true,
		MaxMemoryUsage:          100 * 1024 * 1024, // 100MB
		MemoryCleanupThreshold:  0.8,
	}
	
	scm := &StreamingContextManager{
		tokenManager: tokenManager,
		contexts:     make(map[string]*StreamingContext),
		config:       config,
		logger:       slog.Default().With("component", "streaming-context-manager"),
		metrics:      &ContextManagerMetrics{LastUpdated: time.Now()},
	}
	
	// Start cleanup routine
	scm.startCleanupRoutine()
	
	return scm
}

// startCleanupRoutine starts the background cleanup routine
func (scm *StreamingContextManager) startCleanupRoutine() {
	scm.cleanupTicker = time.NewTicker(scm.config.CleanupInterval)
	
	go func() {
		for {
			select {
			case <-scm.cleanupTicker.C:
				if scm.closed {
					return
				}
				scm.performCleanup()
			}
		}
	}()
}

// StoreContext stores a context with TTL-based expiration
func (scm *StreamingContextManager) StoreContext(ctx context.Context, request *ContextRequest) (*ContextResponse, error) {
	startTime := time.Now()
	
	// Validate request
	if err := scm.validateContextRequest(request); err != nil {
		return nil, fmt.Errorf("invalid context request: %w", err)
	}
	
	// Check concurrent context limit
	if scm.getActiveContextCount() >= int64(scm.config.MaxConcurrentContexts) {
		// Try to evict least recently used context
		if err := scm.evictLRUContext(); err != nil {
			return nil, fmt.Errorf("context limit exceeded and eviction failed: %w", err)
		}
	}
	
	// Calculate token count
	tokenCount := scm.tokenManager.EstimateTokensForModel(request.Content, request.ModelName)
	
	// Compress content if requested and beneficial
	content := request.Content
	isCompressed := false
	if request.Compress && scm.config.EnableCompression && len(content) > 1000 {
		// In a real implementation, you would use actual compression
		// For now, we'll just mark it as compressed
		isCompressed = true
	}
	
	// Determine TTL
	ttl := request.TTL
	if ttl == 0 {
		ttl = scm.config.DefaultTTL
	}
	if ttl > scm.config.MaxTTL {
		ttl = scm.config.MaxTTL
	}
	
	// Create context
	now := time.Now()
	streamingContext := &StreamingContext{
		ID:           request.ID,
		Content:      content,
		TokenCount:   tokenCount,
		CreatedAt:    now,
		LastAccessed: now,
		TTL:          ttl,
		AccessCount:  0,
		Size:         int64(len(content)),
		Metadata:     request.Metadata,
		IsCompressed: isCompressed,
	}
	
	// Store context
	scm.mutex.Lock()
	existingContext, exists := scm.contexts[request.ID]
	scm.contexts[request.ID] = streamingContext
	scm.mutex.Unlock()
	
	// Update metrics
	scm.updateMetrics(func(m *ContextManagerMetrics) {
		if !exists {
			m.TotalContextsCreated++
			m.ActiveContexts++
		}
		m.MemoryUsage += streamingContext.Size
		if exists && existingContext != nil {
			m.MemoryUsage -= existingContext.Size
		}
		m.AverageContextSize = int((int64(m.AverageContextSize)*m.TotalContextsCreated + int64(len(content))) / (m.TotalContextsCreated + 1))
		m.LastUpdated = time.Now()
	})
	
	scm.logger.Debug("Context stored",
		"context_id", request.ID,
		"token_count", tokenCount,
		"size_bytes", streamingContext.Size,
		"ttl", ttl,
		"compressed", isCompressed,
	)
	
	response := &ContextResponse{
		Context:        streamingContext,
		Found:          exists,
		Created:        !exists,
		Updated:        exists,
		TokenCount:     tokenCount,
		ProcessingTime: time.Since(startTime),
		FromCache:      false,
	}
	
	return response, nil
}

// RetrieveContext retrieves a context by ID
func (scm *StreamingContextManager) RetrieveContext(ctx context.Context, contextID string) (*ContextResponse, error) {
	startTime := time.Now()
	
	scm.mutex.RLock()
	streamingContext, exists := scm.contexts[contextID]
	scm.mutex.RUnlock()
	
	if !exists {
		scm.updateMetrics(func(m *ContextManagerMetrics) {
			m.ContextMisses++
		})
		
		return &ContextResponse{
			Found:          false,
			ProcessingTime: time.Since(startTime),
		}, nil
	}
	
	// Check if context has expired
	streamingContext.mutex.RLock()
	expired := time.Since(streamingContext.CreatedAt) > streamingContext.TTL
	streamingContext.mutex.RUnlock()
	
	if expired {
		// Remove expired context
		scm.removeContext(contextID)
		
		scm.updateMetrics(func(m *ContextManagerMetrics) {
			m.ContextMisses++
			m.TotalContextsExpired++
		})
		
		return &ContextResponse{
			Found:          false,
			ProcessingTime: time.Since(startTime),
		}, nil
	}
	
	// Update access information
	streamingContext.mutex.Lock()
	streamingContext.LastAccessed = time.Now()
	streamingContext.AccessCount++
	streamingContext.mutex.Unlock()
	
	scm.updateMetrics(func(m *ContextManagerMetrics) {
		m.ContextHits++
	})
	
	scm.logger.Debug("Context retrieved",
		"context_id", contextID,
		"access_count", streamingContext.AccessCount,
		"age", time.Since(streamingContext.CreatedAt),
	)
	
	response := &ContextResponse{
		Context:        streamingContext,
		Found:          true,
		TokenCount:     streamingContext.TokenCount,
		ProcessingTime: time.Since(startTime),
		FromCache:      true,
	}
	
	return response, nil
}

// UpdateContextTTL updates the TTL of an existing context
func (scm *StreamingContextManager) UpdateContextTTL(contextID string, newTTL time.Duration) error {
	scm.mutex.RLock()
	streamingContext, exists := scm.contexts[contextID]
	scm.mutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("context not found: %s", contextID)
	}
	
	// Validate TTL
	if newTTL > scm.config.MaxTTL {
		newTTL = scm.config.MaxTTL
	}
	
	streamingContext.mutex.Lock()
	oldTTL := streamingContext.TTL
	streamingContext.TTL = newTTL
	streamingContext.LastAccessed = time.Now()
	streamingContext.mutex.Unlock()
	
	scm.logger.Debug("Context TTL updated",
		"context_id", contextID,
		"old_ttl", oldTTL,
		"new_ttl", newTTL,
	)
	
	return nil
}

// RemoveContext removes a context by ID
func (scm *StreamingContextManager) RemoveContext(contextID string) error {
	scm.removeContext(contextID)
	return nil
}

// removeContext internal method to remove context
func (scm *StreamingContextManager) removeContext(contextID string) {
	scm.mutex.Lock()
	streamingContext, exists := scm.contexts[contextID]
	if exists {
		delete(scm.contexts, contextID)
	}
	scm.mutex.Unlock()
	
	if exists {
		scm.updateMetrics(func(m *ContextManagerMetrics) {
			m.ActiveContexts--
			m.MemoryUsage -= streamingContext.Size
		})
		
		scm.logger.Debug("Context removed", "context_id", contextID)
	}
}

// performCleanup performs TTL-based cleanup of expired contexts
func (scm *StreamingContextManager) performCleanup() {
	startTime := time.Now()
	
	scm.mutex.Lock()
	defer scm.mutex.Unlock()
	
	var expiredContexts []string
	now := time.Now()
	
	// Find expired contexts
	for contextID, streamingContext := range scm.contexts {
		streamingContext.mutex.RLock()
		expired := now.Sub(streamingContext.CreatedAt) > streamingContext.TTL
		streamingContext.mutex.RUnlock()
		
		if expired {
			expiredContexts = append(expiredContexts, contextID)
		}
	}
	
	// Remove expired contexts
	var totalSizeFreed int64
	for _, contextID := range expiredContexts {
		if streamingContext, exists := scm.contexts[contextID]; exists {
			totalSizeFreed += streamingContext.Size
			delete(scm.contexts, contextID)
		}
	}
	
	// Check memory usage and perform additional cleanup if needed
	currentMemoryUsage := scm.getCurrentMemoryUsage()
	if float64(currentMemoryUsage) > float64(scm.config.MaxMemoryUsage)*scm.config.MemoryCleanupThreshold {
		additionalFreed := scm.performMemoryCleanup()
		totalSizeFreed += additionalFreed
	}
	
	// Update metrics
	scm.updateMetrics(func(m *ContextManagerMetrics) {
		m.TotalContextsExpired += int64(len(expiredContexts))
		m.ActiveContexts -= int64(len(expiredContexts))
		m.MemoryUsage -= totalSizeFreed
		m.CleanupOperations++
		m.LastCleanup = now
		m.LastUpdated = time.Now()
	})
	
	if len(expiredContexts) > 0 || totalSizeFreed > 0 {
		scm.logger.Debug("Cleanup completed",
			"expired_contexts", len(expiredContexts),
			"memory_freed_bytes", totalSizeFreed,
			"cleanup_time", time.Since(startTime),
		)
	}
}

// performMemoryCleanup performs additional cleanup based on memory usage
func (scm *StreamingContextManager) performMemoryCleanup() int64 {
	// Find least recently used contexts for eviction
	type contextInfo struct {
		id           string
		lastAccessed time.Time
		size         int64
	}
	
	var contexts []contextInfo
	for contextID, streamingContext := range scm.contexts {
		streamingContext.mutex.RLock()
		contexts = append(contexts, contextInfo{
			id:           contextID,
			lastAccessed: streamingContext.LastAccessed,
			size:         streamingContext.Size,
		})
		streamingContext.mutex.RUnlock()
	}
	
	// Sort by last accessed time (oldest first)
	for i := 0; i < len(contexts)-1; i++ {
		for j := i + 1; j < len(contexts); j++ {
			if contexts[i].lastAccessed.After(contexts[j].lastAccessed) {
				contexts[i], contexts[j] = contexts[j], contexts[i]
			}
		}
	}
	
	// Evict contexts until we're under the memory threshold
	var totalFreed int64
	targetMemory := int64(float64(scm.config.MaxMemoryUsage) * 0.7) // Target 70% of max
	currentMemory := scm.getCurrentMemoryUsage()
	
	for _, ctx := range contexts {
		if currentMemory-totalFreed <= targetMemory {
			break
		}
		
		delete(scm.contexts, ctx.id)
		totalFreed += ctx.size
		
		scm.updateMetrics(func(m *ContextManagerMetrics) {
			m.TotalContextsEvicted++
			m.ActiveContexts--
		})
	}
	
	return totalFreed
}

// evictLRUContext evicts the least recently used context
func (scm *StreamingContextManager) evictLRUContext() error {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()
	
	if len(scm.contexts) == 0 {
		return fmt.Errorf("no contexts to evict")
	}
	
	// Find LRU context
	var oldestID string
	var oldestTime time.Time = time.Now()
	
	for contextID, streamingContext := range scm.contexts {
		streamingContext.mutex.RLock()
		if streamingContext.LastAccessed.Before(oldestTime) {
			oldestTime = streamingContext.LastAccessed
			oldestID = contextID
		}
		streamingContext.mutex.RUnlock()
	}
	
	if oldestID != "" {
		streamingContext := scm.contexts[oldestID]
		delete(scm.contexts, oldestID)
		
		scm.updateMetrics(func(m *ContextManagerMetrics) {
			m.TotalContextsEvicted++
			m.ActiveContexts--
			m.MemoryUsage -= streamingContext.Size
		})
		
		scm.logger.Debug("LRU context evicted", "context_id", oldestID)
	}
	
	return nil
}

// validateContextRequest validates a context request
func (scm *StreamingContextManager) validateContextRequest(request *ContextRequest) error {
	if request == nil {
		return fmt.Errorf("context request cannot be nil")
	}
	if request.ID == "" {
		return fmt.Errorf("context ID cannot be empty")
	}
	if request.Content == "" {
		return fmt.Errorf("context content cannot be empty")
	}
	if len(request.Content) > scm.config.MaxContextSize {
		return fmt.Errorf("context content exceeds maximum size: %d > %d", 
			len(request.Content), scm.config.MaxContextSize)
	}
	return nil
}

// getActiveContextCount returns the current number of active contexts
func (scm *StreamingContextManager) getActiveContextCount() int64 {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()
	return int64(len(scm.contexts))
}

// getCurrentMemoryUsage calculates current memory usage
func (scm *StreamingContextManager) getCurrentMemoryUsage() int64 {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()
	
	var totalSize int64
	for _, streamingContext := range scm.contexts {
		streamingContext.mutex.RLock()
		totalSize += streamingContext.Size
		streamingContext.mutex.RUnlock()
	}
	
	return totalSize
}

// GetContextStats returns statistics about stored contexts
func (scm *StreamingContextManager) GetContextStats() map[string]interface{} {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()
	
	stats := map[string]interface{}{
		"active_contexts":    len(scm.contexts),
		"memory_usage_bytes": scm.getCurrentMemoryUsage(),
		"memory_limit_bytes": scm.config.MaxMemoryUsage,
	}
	
	if len(scm.contexts) > 0 {
		var totalAge time.Duration
		var totalAccess int64
		oldestTime := time.Now()
		newestTime := time.Time{}
		
		for _, streamingContext := range scm.contexts {
			streamingContext.mutex.RLock()
			age := time.Since(streamingContext.CreatedAt)
			totalAge += age
			totalAccess += streamingContext.AccessCount
			
			if streamingContext.CreatedAt.Before(oldestTime) {
				oldestTime = streamingContext.CreatedAt
			}
			if streamingContext.CreatedAt.After(newestTime) {
				newestTime = streamingContext.CreatedAt
			}
			streamingContext.mutex.RUnlock()
		}
		
		stats["average_age_seconds"] = float64(totalAge) / float64(len(scm.contexts)) / float64(time.Second)
		stats["average_access_count"] = float64(totalAccess) / float64(len(scm.contexts))
		stats["oldest_context_age_seconds"] = time.Since(oldestTime).Seconds()
		stats["newest_context_age_seconds"] = time.Since(newestTime).Seconds()
	}
	
	return stats
}

// updateMetrics safely updates metrics
func (scm *StreamingContextManager) updateMetrics(updater func(*ContextManagerMetrics)) {
	scm.metrics.mutex.Lock()
	defer scm.metrics.mutex.Unlock()
	updater(scm.metrics)
}

// GetMetrics returns current context manager metrics
func (scm *StreamingContextManager) GetMetrics() *ContextManagerMetrics {
	scm.metrics.mutex.RLock()
	defer scm.metrics.mutex.RUnlock()
	
	metrics := *scm.metrics
	return &metrics
}

// GetConfig returns the current configuration
func (scm *StreamingContextManager) GetConfig() *StreamingContextConfig {
	scm.mutex.RLock()
	defer scm.mutex.RUnlock()
	
	config := *scm.config
	return &config
}

// UpdateConfig updates the configuration
func (scm *StreamingContextManager) UpdateConfig(config *StreamingContextConfig) error {
	if config == nil {
		return fmt.Errorf("configuration cannot be nil")
	}
	
	scm.mutex.Lock()
	defer scm.mutex.Unlock()
	
	oldInterval := scm.config.CleanupInterval
	scm.config = config
	
	// Restart cleanup routine if interval changed
	if oldInterval != config.CleanupInterval {
		scm.cleanupTicker.Stop()
		scm.startCleanupRoutine()
	}
	
	scm.logger.Info("Streaming context manager configuration updated")
	return nil
}

// Close gracefully shuts down the context manager
func (scm *StreamingContextManager) Close() error {
	scm.mutex.Lock()
	defer scm.mutex.Unlock()
	
	scm.closed = true
	
	if scm.cleanupTicker != nil {
		scm.cleanupTicker.Stop()
	}
	
	// Clear all contexts
	contextCount := len(scm.contexts)
	scm.contexts = make(map[string]*StreamingContext)
	
	scm.updateMetrics(func(m *ContextManagerMetrics) {
		m.ActiveContexts = 0
		m.MemoryUsage = 0
	})
	
	scm.logger.Info("Streaming context manager closed",
		"cleared_contexts", contextCount,
	)
	
	return nil
}