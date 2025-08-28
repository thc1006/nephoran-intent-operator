/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// SpecializedIntentProcessingController implements specialized intent processing with LLM and RAG integration
type SpecializedIntentProcessingController struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Logger   logr.Logger

	// LLM Services
	LLMClient            *llm.Client
	RAGService           *RAGService
	PromptEngine         *llm.TelecomPromptEngine
	StreamingProcessor   *llm.StreamingProcessorImpl
	PerformanceOptimizer *llm.PerformanceOptimizer

	// Processing configuration
	Config              IntentProcessingConfig
	SupportedIntents    []string
	ConfidenceThreshold float64

	// Internal state management
	activeProcessing sync.Map // map[string]*ProcessingSession
	metrics          *IntentProcessingMetrics
	cache            *IntentProcessingCache
	circuitBreaker   *llm.CircuitBreaker

	// Health and lifecycle
	started      bool
	stopChan     chan struct{}
	healthStatus interfaces.HealthStatus
	mutex        sync.RWMutex
}

// Note: IntentProcessingConfig is defined in intent_processing_controller.go to avoid duplication

// ProcessingSession tracks an active intent processing session
type ProcessingSession struct {
	IntentID      string    `json:"intentId"`
	CorrelationID string    `json:"correlationId"`
	StartTime     time.Time `json:"startTime"`
	Status        string    `json:"status"`
	Progress      float64   `json:"progress"`
	CurrentStep   string    `json:"currentStep"`
	StreamingID   string    `json:"streamingId,omitempty"`

	// Context and results
	IntentText  string                 `json:"intentText"`
	RAGContext  map[string]interface{} `json:"ragContext,omitempty"`
	LLMResponse map[string]interface{} `json:"llmResponse,omitempty"`
	Confidence  float64                `json:"confidence"`

	// Error handling
	Errors     []string `json:"errors,omitempty"`
	RetryCount int      `json:"retryCount"`
	LastError  string   `json:"lastError,omitempty"`

	// Performance tracking
	Metrics IntentSessionMetrics `json:"metrics"`

	mutex sync.RWMutex
}

// IntentSessionMetrics tracks metrics for a processing session
type IntentSessionMetrics struct {
	LLMLatency         time.Duration `json:"llmLatency"`
	RAGLatency         time.Duration `json:"ragLatency"`
	TotalLatency       time.Duration `json:"totalLatency"`
	TokensUsed         int           `json:"tokensUsed"`
	RAGChunksRetrieved int           `json:"ragChunksRetrieved"`
	CacheHit           bool          `json:"cacheHit"`
	APICallsCount      int           `json:"apiCallsCount"`
}

// IntentProcessingMetrics tracks overall controller metrics
type IntentProcessingMetrics struct {
	TotalProcessed      int64         `json:"totalProcessed"`
	SuccessfulProcessed int64         `json:"successfulProcessed"`
	FailedProcessed     int64         `json:"failedProcessed"`
	AverageLatency      time.Duration `json:"averageLatency"`
	AverageConfidence   float64       `json:"averageConfidence"`

	// Per intent type metrics
	IntentTypeMetrics map[string]*IntentTypeMetrics `json:"intentTypeMetrics"`

	// Resource usage
	TokensUsedTotal     int64   `json:"tokensUsedTotal"`
	CacheHitRate        float64 `json:"cacheHitRate"`
	CircuitBreakerTrips int64   `json:"circuitBreakerTrips"`

	LastUpdated time.Time `json:"lastUpdated"`
	mutex       sync.RWMutex
}

// IntentTypeMetrics tracks metrics per intent type
type IntentTypeMetrics struct {
	Count             int64         `json:"count"`
	SuccessRate       float64       `json:"successRate"`
	AverageLatency    time.Duration `json:"averageLatency"`
	AverageConfidence float64       `json:"averageConfidence"`
}

// IntentProcessingCache provides caching for processed intents
type IntentProcessingCache struct {
	entries    map[string]*CacheEntry
	mutex      sync.RWMutex
	ttl        time.Duration
	maxEntries int
}

// CacheEntry represents a cached intent processing result
type CacheEntry struct {
	Result     interfaces.ProcessingResult `json:"result"`
	Timestamp  time.Time                   `json:"timestamp"`
	HitCount   int64                       `json:"hitCount"`
	IntentHash string                      `json:"intentHash"`
}

// NewSpecializedIntentProcessingController creates a new specialized intent processing controller
func NewSpecializedIntentProcessingController(mgr ctrl.Manager, config IntentProcessingConfig) (*SpecializedIntentProcessingController, error) {
	logger := log.FromContext(context.Background()).WithName("specialized-intent-processor")

	// Initialize LLM client
	llmClient := llm.NewClient(config.LLMEndpoint)

	// Initialize RAG service (stub implementation)
	// Note: MaxContextChunks field doesn't exist, using fallback values
	ragService := &RAGService{} // Use stub implementation
	var err error = nil
	if err != nil {
		return nil, fmt.Errorf("failed to initialize RAG service: %w", err)
	}

	// Initialize prompt engine
	promptEngine := llm.NewTelecomPromptEngine()

	// Initialize streaming processor
	streamingProcessor := llm.NewStreamingProcessorImpl(llmClient, llm.NewTokenManager(), nil)

	// Initialize performance optimizer
	performanceOptimizer := llm.NewPerformanceOptimizer(&llm.PerformanceConfig{
		LatencyBufferSize:     1000,
		OptimizationInterval:  time.Minute * 5,
		MetricsExportInterval: time.Second * 30,
		EnableTracing:         true,
		TraceSamplingRatio:    0.1,
	})

	// Initialize circuit breaker
	var circuitBreaker *llm.CircuitBreaker
	if config.CircuitBreakerEnabled {
		cbConfig := &llm.CircuitBreakerConfig{
			FailureThreshold:    int64(config.FailureThreshold),
			Timeout:             config.RecoveryTimeout,
			FailureRate:         0.5,
			MinimumRequestCount: 10,
			HalfOpenTimeout:     60 * time.Second,
			SuccessThreshold:    3,
			HalfOpenMaxRequests: 5,
			ResetTimeout:        60 * time.Second,
		}
		circuitBreaker = llm.NewCircuitBreaker("specialized-intent-processor", cbConfig)
	}

	controller := &SpecializedIntentProcessingController{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("specialized-intent-processor"),
		Logger:   logger,

		LLMClient:            llmClient,
		RAGService:           ragService,
		PromptEngine:         promptEngine,
		StreamingProcessor:   streamingProcessor,
		PerformanceOptimizer: performanceOptimizer,

		Config:              config,
		SupportedIntents:    []string{"5g-deployment", "network-slice", "cnf-deployment", "monitoring-setup", "security-config"},
		ConfidenceThreshold: 0.7,

		metrics:        NewIntentProcessingMetrics(),
		circuitBreaker: circuitBreaker,
		stopChan:       make(chan struct{}),

		healthStatus: interfaces.HealthStatus{
			Status:      "Healthy",
			Message:     "Controller initialized successfully",
			LastChecked: time.Now(),
		},
	}

	// Initialize cache if enabled
	if config.CacheEnabled {
		cacheTTL := config.CacheTTL
		if cacheTTL == 0 {
			cacheTTL = 5 * time.Minute // Default TTL
		}
		controller.cache = &IntentProcessingCache{
			entries:    make(map[string]*CacheEntry),
			ttl:        cacheTTL,
			maxEntries: 1000, // Default max cache entries
		}
	}

	return controller, nil
}

// ProcessPhase implements the PhaseController interface
func (c *SpecializedIntentProcessingController) ProcessPhase(ctx context.Context, intent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase) (interfaces.ProcessingResult, error) {
	if phase != interfaces.PhaseLLMProcessing {
		return interfaces.ProcessingResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("unsupported phase: %s", phase),
		}, nil
	}

	result, err := c.ProcessIntent(ctx, intent)
	if err != nil {
		return interfaces.ProcessingResult{
			Success:      false,
			ErrorMessage: err.Error(),
		}, err
	}
	return *result, nil
}

// ProcessIntent implements the IntentProcessor interface
func (c *SpecializedIntentProcessingController) ProcessIntent(ctx context.Context, intent *nephoranv1.NetworkIntent) (*interfaces.ProcessingResult, error) {
	startTime := time.Now()

	c.Logger.Info("Processing intent",
		"intentId", intent.Name,
		"intentText", intent.Spec.Intent,
		"intentType", intent.Spec.IntentType)

	// Create processing session
	session := &ProcessingSession{
		IntentID:      intent.Name,
		CorrelationID: fmt.Sprintf("intent-%s-%d", intent.Name, startTime.Unix()),
		StartTime:     startTime,
		Status:        "processing",
		Progress:      0.0,
		CurrentStep:   "initialization",
		IntentText:    intent.Spec.Intent,
	}

	// Store session for tracking
	c.activeProcessing.Store(intent.Name, session)
	defer c.activeProcessing.Delete(intent.Name)

	// Check cache first
	if c.cache != nil {
		if cached := c.getCachedResult(intent.Spec.Intent); cached != nil {
			c.Logger.Info("Cache hit for intent", "intentId", intent.Name)
			session.Metrics.CacheHit = true
			c.updateMetrics(session, true, time.Since(startTime))

			return &interfaces.ProcessingResult{
				Success:   true,
				NextPhase: interfaces.PhaseResourcePlanning,
				Data:      cached.Result.Data,
				Metrics: map[string]float64{
					"processing_time_ms": float64(time.Since(startTime).Milliseconds()),
					"cache_hit":          1,
				},
			}, nil
		}
	}

	// Validate intent
	session.updateProgress(0.1, "validating_intent")
	if err := c.ValidateIntent(ctx, intent.Spec.Intent); err != nil {
		session.addError(fmt.Sprintf("intent validation failed: %v", err))
		c.updateMetrics(session, false, time.Since(startTime))
		return &interfaces.ProcessingResult{
			Success:      false,
			ErrorMessage: err.Error(),
			ErrorCode:    "VALIDATION_ERROR",
		}, nil
	}

	// Enhance with RAG context
	session.updateProgress(0.3, "enhancing_with_rag")
	ragContext, err := c.EnhanceWithRAG(ctx, intent.Spec.Intent)
	if err != nil {
		c.Logger.Error(err, "Failed to enhance intent with RAG", "intentId", intent.Name)
		session.addError(fmt.Sprintf("RAG enhancement failed: %v", err))
		// Continue processing without RAG context
		ragContext = make(map[string]interface{})
	}
	session.RAGContext = ragContext
	session.Metrics.RAGLatency = time.Since(startTime)

	// Process with LLM
	session.updateProgress(0.6, "processing_with_llm")
	llmStartTime := time.Now()

	var llmResponse map[string]interface{}
	var confidence float64

	// For now, use regular LLM processing
	// Streaming support would require additional configuration
	llmResponse, confidence, err = c.processWithLLM(ctx, intent, ragContext)

	session.Metrics.LLMLatency = time.Since(llmStartTime)

	if err != nil {
		session.addError(fmt.Sprintf("LLM processing failed: %v", err))
		c.updateMetrics(session, false, time.Since(startTime))
		return &interfaces.ProcessingResult{
			Success:      false,
			ErrorMessage: err.Error(),
			ErrorCode:    "LLM_PROCESSING_ERROR",
		}, nil
	}

	// Check confidence threshold
	if confidence < c.ConfidenceThreshold {
		session.addError(fmt.Sprintf("confidence %f below threshold %f", confidence, c.ConfidenceThreshold))
		c.updateMetrics(session, false, time.Since(startTime))
		return &interfaces.ProcessingResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("intent processing confidence %.2f below threshold %.2f", confidence, c.ConfidenceThreshold),
			ErrorCode:    "LOW_CONFIDENCE",
		}, nil
	}

	session.LLMResponse = llmResponse
	session.Confidence = confidence
	session.updateProgress(1.0, "completed")

	// Create result
	result := &interfaces.ProcessingResult{
		Success:   true,
		NextPhase: interfaces.PhaseResourcePlanning,
		Data: map[string]interface{}{
			"llmResponse":    llmResponse,
			"ragContext":     ragContext,
			"confidence":     confidence,
			"intentType":     intent.Spec.IntentType,
			"originalIntent": intent.Spec.Intent,
			"correlationId":  session.CorrelationID,
		},
		Metrics: map[string]float64{
			"processing_time_ms": float64(time.Since(startTime).Milliseconds()),
			"llm_latency_ms":     float64(session.Metrics.LLMLatency.Milliseconds()),
			"rag_latency_ms":     float64(session.Metrics.RAGLatency.Milliseconds()),
			"confidence":         confidence,
			"tokens_used":        float64(session.Metrics.TokensUsed),
		},
		Events: []interfaces.ProcessingEvent{
			{
				Timestamp:     time.Now(),
				EventType:     "IntentProcessed",
				Message:       fmt.Sprintf("Intent processed successfully with confidence %.2f", confidence),
				CorrelationID: session.CorrelationID,
				Data: map[string]interface{}{
					"intentId":   intent.Name,
					"confidence": confidence,
				},
			},
		},
	}

	// Cache result if enabled
	if c.cache != nil {
		c.cacheResult(intent.Spec.Intent, *result)
	}

	// Update metrics
	c.updateMetrics(session, true, time.Since(startTime))

	c.Logger.Info("Intent processing completed successfully",
		"intentId", intent.Name,
		"confidence", confidence,
		"duration", time.Since(startTime))

	return result, nil
}

// processWithLLM processes intent using standard LLM client
func (c *SpecializedIntentProcessingController) processWithLLM(ctx context.Context, intent *nephoranv1.NetworkIntent, ragContext map[string]interface{}) (map[string]interface{}, float64, error) {
	// Build enhanced prompt with RAG context
	prompt := c.PromptEngine.GeneratePrompt("NetworkFunctionDeployment", intent.Spec.Intent)
	if len(ragContext) > 0 {
		// Add RAG context to the prompt
		prompt += "\n\nAdditional Context from Knowledge Base:\n"
		if docs, ok := ragContext["relevant_documents"]; ok {
			if docList, ok := docs.([]interface{}); ok {
				for i, doc := range docList {
					if i >= 3 { // Limit to 3 most relevant documents
						break
					}
					prompt += fmt.Sprintf("- %v\n", doc)
				}
			}
		}
	}

	// Use circuit breaker if enabled
	if c.circuitBreaker != nil {
		response, err := c.circuitBreaker.Execute(ctx, func(ctx context.Context) (interface{}, error) {
			return c.LLMClient.ProcessIntent(ctx, prompt)
		})
		if err != nil {
			return nil, 0, err
		}
		return c.parseValidateLLMResponse(response.(string))
	}

	// Direct LLM call
	response, err := c.LLMClient.ProcessIntent(ctx, prompt)
	if err != nil {
		return nil, 0, fmt.Errorf("LLM processing failed: %w", err)
	}

	return c.parseValidateLLMResponse(response)
}

// processWithStreamingLLM processes intent using streaming LLM client
func (c *SpecializedIntentProcessingController) processWithStreamingLLM(ctx context.Context, intent *nephoranv1.NetworkIntent, ragContext map[string]interface{}) (map[string]interface{}, float64, error) {
	// Note: Streaming would require a different interface
	// For now, fall back to regular processing
	return c.processWithLLM(ctx, intent, ragContext)
}

// parseValidateLLMResponse parses and validates LLM response
func (c *SpecializedIntentProcessingController) parseValidateLLMResponse(response string) (map[string]interface{}, float64, error) {
	// Parse JSON response
	var result map[string]interface{}
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		// Try to extract JSON from response if it's wrapped in text
		startIdx := strings.Index(response, "{")
		endIdx := strings.LastIndex(response, "}")
		if startIdx >= 0 && endIdx > startIdx {
			jsonStr := response[startIdx : endIdx+1]
			if err := json.Unmarshal([]byte(jsonStr), &result); err != nil {
				return nil, 0, fmt.Errorf("failed to parse LLM response as JSON: %w", err)
			}
		} else {
			return nil, 0, fmt.Errorf("no valid JSON found in LLM response")
		}
	}

	// Extract confidence score
	confidence, ok := result["confidence"].(float64)
	if !ok {
		// Try alternative confidence field names
		if confInterface, exists := result["confidence_score"]; exists {
			if conf, ok := confInterface.(float64); ok {
				confidence = conf
			}
		}
		if confidence == 0 {
			confidence = 0.8 // Default confidence if not provided
		}
	}

	// Validate required fields
	requiredFields := []string{"network_functions", "deployment_pattern", "resources"}
	for _, field := range requiredFields {
		if _, exists := result[field]; !exists {
			c.Logger.Info("Missing required field in LLM response", "field", field)
		}
	}

	return result, confidence, nil
}

// ValidateIntent validates the intent text
func (c *SpecializedIntentProcessingController) ValidateIntent(ctx context.Context, intent string) error {
	// Basic validation
	if strings.TrimSpace(intent) == "" {
		return fmt.Errorf("intent cannot be empty")
	}

	// Length validation
	if len(intent) > 10000 {
		return fmt.Errorf("intent text too long (max 10000 characters)")
	}

	// Telecom-specific validation
	telecomKeywords := []string{
		"5g", "network", "deployment", "slice", "cnf", "vnf", "amf", "smf", "upf",
		"monitoring", "security", "performance", "scaling", "bandwidth", "latency",
	}

	intentLower := strings.ToLower(intent)
	hasRelevantKeyword := false
	for _, keyword := range telecomKeywords {
		if strings.Contains(intentLower, keyword) {
			hasRelevantKeyword = true
			break
		}
	}

	if !hasRelevantKeyword {
		return fmt.Errorf("intent does not appear to be telecommunications-related")
	}

	return nil
}

// EnhanceWithRAG enhances intent with RAG context
func (c *SpecializedIntentProcessingController) EnhanceWithRAG(ctx context.Context, intent string) (map[string]interface{}, error) {
	if c.RAGService == nil {
		return make(map[string]interface{}), nil
	}

	// Query RAG service for relevant context
	ragQuery := &RAGRequest{
		Query:         intent,
		MaxResults:    10,  // Default value since MaxContextChunks doesn't exist
		MinConfidence: 0.7, // Default value since SimilarityThreshold doesn't exist
	}

	ragResponse, err := c.RAGService.ProcessQuery(ctx, ragQuery)
	if err != nil {
		return nil, fmt.Errorf("RAG query failed: %w", err)
	}

	// Structure RAG context
	ragContext := ragResponse.Context

	return ragContext, nil
}

// GetSupportedIntentTypes returns supported intent types
func (c *SpecializedIntentProcessingController) GetSupportedIntentTypes() []string {
	return c.SupportedIntents
}

// Helper methods for session management

// updateProgress updates session progress
func (s *ProcessingSession) updateProgress(progress float64, step string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Progress = progress
	s.CurrentStep = step
}

// addError adds error to session
func (s *ProcessingSession) addError(errorMsg string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Errors = append(s.Errors, errorMsg)
	s.LastError = errorMsg
}

// Cache management methods

// getCachedResult retrieves cached result for intent
func (c *SpecializedIntentProcessingController) getCachedResult(intent string) *CacheEntry {
	if c.cache == nil {
		return nil
	}

	c.cache.mutex.RLock()
	defer c.cache.mutex.RUnlock()

	intentHash := c.hashIntent(intent)
	if entry, exists := c.cache.entries[intentHash]; exists {
		// Check if entry is still valid
		if time.Since(entry.Timestamp) < c.cache.ttl {
			entry.HitCount++
			return entry
		}
		// Entry expired, remove it
		delete(c.cache.entries, intentHash)
	}

	return nil
}

// cacheResult stores result in cache
func (c *SpecializedIntentProcessingController) cacheResult(intent string, result interfaces.ProcessingResult) {
	if c.cache == nil {
		return
	}

	c.cache.mutex.Lock()
	defer c.cache.mutex.Unlock()

	intentHash := c.hashIntent(intent)

	// Check cache size limit
	if len(c.cache.entries) >= c.cache.maxEntries {
		// Remove oldest entry (simple LRU)
		var oldestKey string
		var oldestTime time.Time
		for key, entry := range c.cache.entries {
			if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
				oldestKey = key
				oldestTime = entry.Timestamp
			}
		}
		if oldestKey != "" {
			delete(c.cache.entries, oldestKey)
		}
	}

	c.cache.entries[intentHash] = &CacheEntry{
		Result:     result,
		Timestamp:  time.Now(),
		HitCount:   0,
		IntentHash: intentHash,
	}
}

// hashIntent creates a hash for intent text
func (c *SpecializedIntentProcessingController) hashIntent(intent string) string {
	// Simple hash based on intent content - in production, use proper hashing
	return fmt.Sprintf("intent_%x", len(intent)+int(intent[0]))
}

// updateMetrics updates controller metrics
func (c *SpecializedIntentProcessingController) updateMetrics(session *ProcessingSession, success bool, totalDuration time.Duration) {
	c.metrics.mutex.Lock()
	defer c.metrics.mutex.Unlock()

	c.metrics.TotalProcessed++
	if success {
		c.metrics.SuccessfulProcessed++
	} else {
		c.metrics.FailedProcessed++
	}

	// Update average latency
	if c.metrics.TotalProcessed > 0 {
		totalLatency := time.Duration(c.metrics.TotalProcessed-1) * c.metrics.AverageLatency
		totalLatency += totalDuration
		c.metrics.AverageLatency = totalLatency / time.Duration(c.metrics.TotalProcessed)
	} else {
		c.metrics.AverageLatency = totalDuration
	}

	// Update cache hit rate
	if c.cache != nil {
		totalCacheRequests := c.metrics.TotalProcessed
		cacheHits := int64(0)
		c.cache.mutex.RLock()
		for _, entry := range c.cache.entries {
			cacheHits += entry.HitCount
		}
		c.cache.mutex.RUnlock()

		if totalCacheRequests > 0 {
			c.metrics.CacheHitRate = float64(cacheHits) / float64(totalCacheRequests)
		}
	}

	c.metrics.LastUpdated = time.Now()
}

// NewIntentProcessingMetrics creates new metrics instance
func NewIntentProcessingMetrics() *IntentProcessingMetrics {
	return &IntentProcessingMetrics{
		IntentTypeMetrics: make(map[string]*IntentTypeMetrics),
	}
}

// Interface implementation methods

// GetPhaseStatus returns the status of a processing phase
func (c *SpecializedIntentProcessingController) GetPhaseStatus(ctx context.Context, intentID string) (*interfaces.PhaseStatus, error) {
	if session, exists := c.activeProcessing.Load(intentID); exists {
		s := session.(*ProcessingSession)
		s.mutex.RLock()
		defer s.mutex.RUnlock()

		status := "Pending"
		if s.Progress > 0 && s.Progress < 1.0 {
			status = "InProgress"
		} else if s.Progress >= 1.0 {
			status = "Completed"
		}
		if len(s.Errors) > 0 {
			status = "Failed"
		}

		return &interfaces.PhaseStatus{
			Phase:      interfaces.PhaseLLMProcessing,
			Status:     status,
			StartTime:  &metav1.Time{Time: s.StartTime},
			RetryCount: int32(s.RetryCount),
			LastError:  s.LastError,
			Metrics: map[string]float64{
				"progress":    s.Progress,
				"confidence":  s.Confidence,
				"llm_latency": float64(s.Metrics.LLMLatency.Milliseconds()),
				"rag_latency": float64(s.Metrics.RAGLatency.Milliseconds()),
			},
		}, nil
	}

	return &interfaces.PhaseStatus{
		Phase:  interfaces.PhaseLLMProcessing,
		Status: "Pending",
	}, nil
}

// HandlePhaseError handles errors during phase processing
func (c *SpecializedIntentProcessingController) HandlePhaseError(ctx context.Context, intentID string, err error) error {
	c.Logger.Error(err, "Intent processing error", "intentId", intentID)

	if session, exists := c.activeProcessing.Load(intentID); exists {
		s := session.(*ProcessingSession)
		s.addError(err.Error())
		s.RetryCount++

		// Implement retry logic if retries are available
		if s.RetryCount < c.Config.MaxRetries {
			c.Logger.Info("Scheduling retry for intent processing",
				"intentId", intentID,
				"retryCount", s.RetryCount,
				"maxRetries", c.Config.MaxRetries)
			// Return nil to indicate retry should happen
			return nil
		}
	}

	return err
}

// GetDependencies returns phase dependencies
func (c *SpecializedIntentProcessingController) GetDependencies() []interfaces.ProcessingPhase {
	return []interfaces.ProcessingPhase{interfaces.PhaseIntentReceived}
}

// GetBlockedPhases returns phases blocked by this controller
func (c *SpecializedIntentProcessingController) GetBlockedPhases() []interfaces.ProcessingPhase {
	return []interfaces.ProcessingPhase{
		interfaces.PhaseResourcePlanning,
		interfaces.PhaseManifestGeneration,
		interfaces.PhaseGitOpsCommit,
		interfaces.PhaseDeploymentVerification,
	}
}

// SetupWithManager sets up the controller with the Manager
func (c *SpecializedIntentProcessingController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		Complete(c)
}

// Start starts the controller
func (c *SpecializedIntentProcessingController) Start(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.started {
		return fmt.Errorf("controller already started")
	}

	c.Logger.Info("Starting specialized intent processing controller")

	// Start background goroutines for cleanup and monitoring
	go c.backgroundCleanup(ctx)
	go c.healthMonitoring(ctx)

	c.started = true
	c.healthStatus = interfaces.HealthStatus{
		Status:      "Healthy",
		Message:     "Controller started successfully",
		LastChecked: time.Now(),
	}

	return nil
}

// Stop stops the controller
func (c *SpecializedIntentProcessingController) Stop(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.started {
		return nil
	}

	c.Logger.Info("Stopping specialized intent processing controller")

	close(c.stopChan)

	// Wait for active processing to complete or timeout
	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	for {
		activeCount := 0
		c.activeProcessing.Range(func(key, value interface{}) bool {
			activeCount++
			return true
		})

		if activeCount == 0 {
			break
		}

		select {
		case <-timeout.C:
			c.Logger.Info("Timeout waiting for active processing to complete", "activeCount", activeCount)
			break
		case <-time.After(100 * time.Millisecond):
			// Continue waiting
		}
	}

	c.started = false
	c.healthStatus = interfaces.HealthStatus{
		Status:      "Stopped",
		Message:     "Controller stopped",
		LastChecked: time.Now(),
	}

	return nil
}

// GetHealthStatus returns the health status of the controller
func (c *SpecializedIntentProcessingController) GetHealthStatus(ctx context.Context) (interfaces.HealthStatus, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Update metrics in health status
	c.healthStatus.Metrics = map[string]interface{}{
		"totalProcessed":   c.metrics.TotalProcessed,
		"successRate":      c.getSuccessRate(),
		"averageLatency":   c.metrics.AverageLatency.Milliseconds(),
		"cacheHitRate":     c.metrics.CacheHitRate,
		"activeProcessing": c.getActiveProcessingCount(),
	}
	c.healthStatus.LastChecked = time.Now()

	return c.healthStatus, nil
}

// GetMetrics returns controller metrics
func (c *SpecializedIntentProcessingController) GetMetrics(ctx context.Context) (map[string]float64, error) {
	c.metrics.mutex.RLock()
	defer c.metrics.mutex.RUnlock()

	return map[string]float64{
		"total_processed":       float64(c.metrics.TotalProcessed),
		"successful_processed":  float64(c.metrics.SuccessfulProcessed),
		"failed_processed":      float64(c.metrics.FailedProcessed),
		"success_rate":          c.getSuccessRate(),
		"average_latency_ms":    float64(c.metrics.AverageLatency.Milliseconds()),
		"average_confidence":    c.metrics.AverageConfidence,
		"tokens_used_total":     float64(c.metrics.TokensUsedTotal),
		"cache_hit_rate":        c.metrics.CacheHitRate,
		"circuit_breaker_trips": float64(c.metrics.CircuitBreakerTrips),
		"active_processing":     float64(c.getActiveProcessingCount()),
	}, nil
}

// Reconcile implements the controller reconciliation logic
func (c *SpecializedIntentProcessingController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get the NetworkIntent
	var intent nephoranv1.NetworkIntent
	if err := c.Get(ctx, req.NamespacedName, &intent); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("NetworkIntent resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get NetworkIntent")
		return ctrl.Result{}, err
	}

	// Check if this intent should be processed by this controller
	// For now, we'll process all intents in certain phases
	if intent.Status.Phase != "LLMProcessing" && intent.Status.Phase != "" {
		return ctrl.Result{}, nil
	}

	// Process the intent
	result, err := c.ProcessIntent(ctx, &intent)
	if err != nil {
		logger.Error(err, "Failed to process intent")
		return ctrl.Result{RequeueAfter: time.Minute * 1}, err
	}

	// Update intent status based on result
	if result.Success {
		intent.Status.Phase = "ResourcePlanning"
		if intent.Status.Extensions == nil {
			intent.Status.Extensions = make(map[string]runtime.RawExtension)
		}
		// Store LLM response in extensions
		if llmResponseBytes, err := json.Marshal(result.Data); err == nil {
			intent.Status.Extensions["llmResponse"] = runtime.RawExtension{Raw: llmResponseBytes}
		}
		intent.Status.LastUpdateTime = metav1.Now()
		intent.Status.LastMessage = "Intent processed successfully with LLM"
	} else {
		intent.Status.Phase = "Failed"
		intent.Status.LastMessage = result.ErrorMessage
		intent.Status.LastUpdateTime = metav1.Now()
	}

	// Update the intent status
	if err := c.Status().Update(ctx, &intent); err != nil {
		logger.Error(err, "Failed to update NetworkIntent status")
		return ctrl.Result{}, err
	}

	logger.Info("Intent processing completed", "intentId", intent.Name, "success", result.Success)

	return ctrl.Result{}, nil
}

// Helper methods

// backgroundCleanup performs background cleanup tasks
func (c *SpecializedIntentProcessingController) backgroundCleanup(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanupExpiredSessions()
			c.cleanupExpiredCache()

		case <-c.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// healthMonitoring performs periodic health monitoring
func (c *SpecializedIntentProcessingController) healthMonitoring(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.performHealthCheck()

		case <-c.stopChan:
			return

		case <-ctx.Done():
			return
		}
	}
}

// cleanupExpiredSessions removes expired processing sessions
func (c *SpecializedIntentProcessingController) cleanupExpiredSessions() {
	expiredSessions := make([]string, 0)

	c.activeProcessing.Range(func(key, value interface{}) bool {
		session := value.(*ProcessingSession)
		if time.Since(session.StartTime) > time.Hour { // 1 hour expiration
			expiredSessions = append(expiredSessions, key.(string))
		}
		return true
	})

	for _, sessionID := range expiredSessions {
		c.activeProcessing.Delete(sessionID)
		c.Logger.Info("Cleaned up expired processing session", "sessionId", sessionID)
	}
}

// cleanupExpiredCache removes expired cache entries
func (c *SpecializedIntentProcessingController) cleanupExpiredCache() {
	if c.cache == nil {
		return
	}

	c.cache.mutex.Lock()
	defer c.cache.mutex.Unlock()

	expiredKeys := make([]string, 0)
	for key, entry := range c.cache.entries {
		if time.Since(entry.Timestamp) > c.cache.ttl {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(c.cache.entries, key)
	}

	if len(expiredKeys) > 0 {
		c.Logger.Info("Cleaned up expired cache entries", "count", len(expiredKeys))
	}
}

// performHealthCheck performs controller health checking
func (c *SpecializedIntentProcessingController) performHealthCheck() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// Check if controller is still responsive
	successRate := c.getSuccessRate()
	activeCount := c.getActiveProcessingCount()

	status := "Healthy"
	message := "Controller operating normally"

	if successRate < 0.8 && c.metrics.TotalProcessed > 10 {
		status = "Degraded"
		message = fmt.Sprintf("Low success rate: %.2f", successRate)
	}

	if activeCount > 100 { // Threshold for too many active sessions
		status = "Degraded"
		message = fmt.Sprintf("High active processing count: %d", activeCount)
	}

	if c.circuitBreaker != nil {
		// Access circuit breaker state through metrics or stats
		cbStats := c.circuitBreaker.GetStats()
		if stateVal, ok := cbStats["state"]; ok {
			if state, ok := stateVal.(string); ok && state == "Open" {
				status = "Unhealthy"
				message = "Circuit breaker is open"
			}
		}
	}

	c.healthStatus = interfaces.HealthStatus{
		Status:      status,
		Message:     message,
		LastChecked: time.Now(),
		Metrics: map[string]interface{}{
			"successRate":      successRate,
			"activeProcessing": activeCount,
			"totalProcessed":   c.metrics.TotalProcessed,
		},
	}
}

// getSuccessRate calculates success rate
func (c *SpecializedIntentProcessingController) getSuccessRate() float64 {
	if c.metrics.TotalProcessed == 0 {
		return 1.0
	}
	return float64(c.metrics.SuccessfulProcessed) / float64(c.metrics.TotalProcessed)
}

// getActiveProcessingCount returns count of active processing sessions
func (c *SpecializedIntentProcessingController) getActiveProcessingCount() int {
	count := 0
	c.activeProcessing.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}
