package rag

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"
)

// CostAwareEmbeddingService extends EmbeddingService with advanced cost optimization
type CostAwareEmbeddingService struct {
	*EmbeddingService
	costOptimizer    *CostOptimizer
	providerMonitor  *ProviderMonitor
	fallbackManager  *FallbackManager
	budgetManager    *BudgetManager
	circuitBreakers  map[string]*CircuitBreaker
	mu               sync.RWMutex
}

// CostOptimizer manages cost optimization strategies
type CostOptimizer struct {
	config           *CostOptimizerConfig
	logger           *slog.Logger
	costHistory      *CostHistory
	providerMetrics  map[string]*ProviderMetrics
	mu               sync.RWMutex
}

// CostOptimizerConfig holds configuration for cost optimization
type CostOptimizerConfig struct {
	// Cost optimization strategies
	OptimizationStrategy string  `json:"optimization_strategy"` // "aggressive", "balanced", "quality_first"
	CostWeight           float64 `json:"cost_weight"`           // Weight for cost in provider selection (0-1)
	PerformanceWeight    float64 `json:"performance_weight"`    // Weight for performance (0-1)
	QualityWeight        float64 `json:"quality_weight"`        // Weight for quality (0-1)
	
	// Budget management
	EnableBudgetTracking bool    `json:"enable_budget_tracking"`
	HourlyBudget         float64 `json:"hourly_budget"`
	DailyBudget          float64 `json:"daily_budget"`
	MonthlyBudget        float64 `json:"monthly_budget"`
	BudgetAlertThreshold float64 `json:"budget_alert_threshold"` // Alert when X% of budget used
	
	// Provider selection
	MinProviderScore     float64       `json:"min_provider_score"`      // Minimum score for provider selection
	ProviderTimeout      time.Duration `json:"provider_timeout"`        // Timeout for provider health checks
	RebalanceInterval    time.Duration `json:"rebalance_interval"`      // How often to rebalance provider selection
	
	// Circuit breaker configuration
	CircuitBreakerThreshold int           `json:"circuit_breaker_threshold"` // Failures before opening circuit
	CircuitBreakerTimeout   time.Duration `json:"circuit_breaker_timeout"`   // Time before attempting to close circuit
	CircuitBreakerMaxRetries int          `json:"circuit_breaker_max_retries"` // Max retries in half-open state
}

// ProviderMonitor monitors provider performance and health
type ProviderMonitor struct {
	metrics          map[string]*ProviderMetrics
	healthChecks     map[string]*HealthCheckResult
	performanceStats map[string]*PerformanceStats
	mu               sync.RWMutex
	logger           *slog.Logger
}

// ProviderMetrics tracks detailed metrics for each provider
type ProviderMetrics struct {
	// Cost metrics
	TotalCost           float64   `json:"total_cost"`
	AverageCostPerToken float64   `json:"average_cost_per_token"`
	CostTrend           []float64 `json:"cost_trend"` // Historical cost data
	
	// Performance metrics
	AverageLatency      time.Duration `json:"average_latency"`
	P95Latency          time.Duration `json:"p95_latency"`
	P99Latency          time.Duration `json:"p99_latency"`
	SuccessRate         float64       `json:"success_rate"`
	ErrorRate           float64       `json:"error_rate"`
	
	// Quality metrics
	AverageQuality      float64   `json:"average_quality"`
	QualityVariance     float64   `json:"quality_variance"`
	QualityTrend        []float64 `json:"quality_trend"`
	
	// Usage metrics
	RequestCount        int64     `json:"request_count"`
	TokensProcessed     int64     `json:"tokens_processed"`
	LastUsed            time.Time `json:"last_used"`
	
	// Health metrics
	ConsecutiveFailures int       `json:"consecutive_failures"`
	LastHealthCheck     time.Time `json:"last_health_check"`
	HealthScore         float64   `json:"health_score"`
}

// HealthCheckResult represents the result of a provider health check
type HealthCheckResult struct {
	Provider    string        `json:"provider"`
	Healthy     bool          `json:"healthy"`
	Latency     time.Duration `json:"latency"`
	Error       error         `json:"error,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	Diagnostics map[string]interface{} `json:"diagnostics,omitempty"`
}

// PerformanceStats tracks performance statistics
type PerformanceStats struct {
	LatencyHistogram []time.Duration `json:"latency_histogram"`
	ErrorTypes       map[string]int  `json:"error_types"`
	ThroughputRate   float64         `json:"throughput_rate"` // Tokens per second
}

// FallbackManager manages fallback strategies
type FallbackManager struct {
	config          *FallbackConfig
	fallbackChains  map[string][]string // Primary provider -> ordered fallback list
	fallbackHistory map[string]*FallbackHistory
	mu              sync.RWMutex
	logger          *slog.Logger
}

// FallbackConfig holds fallback configuration
type FallbackConfig struct {
	EnableAutoFallback   bool          `json:"enable_auto_fallback"`
	FallbackStrategy     string        `json:"fallback_strategy"`      // "cost_optimized", "performance_optimized", "quality_optimized"
	MaxFallbackAttempts  int           `json:"max_fallback_attempts"`
	FallbackTimeout      time.Duration `json:"fallback_timeout"`
	PreserveBudget       bool          `json:"preserve_budget"`        // Don't use expensive fallbacks if budget is tight
}

// FallbackHistory tracks fallback usage
type FallbackHistory struct {
	FallbackCount    int                    `json:"fallback_count"`
	SuccessfulFallbacks int                 `json:"successful_fallbacks"`
	FallbackReasons  map[string]int         `json:"fallback_reasons"`
	LastFallback     time.Time              `json:"last_fallback"`
}

// BudgetManager manages budget constraints
type BudgetManager struct {
	config         *BudgetConfig
	currentSpend   *BudgetSpend
	budgetAlerts   []BudgetAlert
	spendHistory   map[string]float64 // date/hour -> spend amount
	mu             sync.RWMutex
	logger         *slog.Logger
}

// BudgetConfig holds budget configuration
type BudgetConfig struct {
	HourlyLimit    float64 `json:"hourly_limit"`
	DailyLimit     float64 `json:"daily_limit"`
	MonthlyLimit   float64 `json:"monthly_limit"`
	ReservePercent float64 `json:"reserve_percent"` // Reserve X% of budget for critical operations
	AutoThrottle   bool    `json:"auto_throttle"`   // Automatically throttle when approaching limits
}

// BudgetSpend tracks current spending
type BudgetSpend struct {
	HourlySpend  float64   `json:"hourly_spend"`
	DailySpend   float64   `json:"daily_spend"`
	MonthlySpend float64   `json:"monthly_spend"`
	LastReset    time.Time `json:"last_reset"`
}

// BudgetAlert represents a budget alert
type BudgetAlert struct {
	Type      string    `json:"type"` // "hourly", "daily", "monthly"
	Threshold float64   `json:"threshold"`
	Current   float64   `json:"current"`
	Limit     float64   `json:"limit"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

// CircuitBreaker implements circuit breaker pattern for providers
type CircuitBreaker struct {
	provider         string
	state            CircuitState
	failures         int
	successCount     int
	lastFailureTime  time.Time
	lastAttemptTime  time.Time
	config           *CircuitBreakerConfig
	mu               sync.RWMutex
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// CircuitBreakerConfig holds circuit breaker configuration
type CircuitBreakerConfig struct {
	FailureThreshold int           `json:"failure_threshold"`
	SuccessThreshold int           `json:"success_threshold"`
	Timeout          time.Duration `json:"timeout"`
	MaxRetries       int           `json:"max_retries"`
}

// CostHistory tracks historical cost data
type CostHistory struct {
	hourlyHistory  map[string][]CostDataPoint
	dailyHistory   map[string][]CostDataPoint
	monthlyHistory map[string][]CostDataPoint
	mu             sync.RWMutex
}

// CostDataPoint represents a cost data point
type CostDataPoint struct {
	Timestamp    time.Time `json:"timestamp"`
	Provider     string    `json:"provider"`
	Cost         float64   `json:"cost"`
	TokenCount   int       `json:"token_count"`
	RequestCount int       `json:"request_count"`
}

// NewCostAwareEmbeddingService creates a new cost-aware embedding service
func NewCostAwareEmbeddingService(baseService *EmbeddingService, config *CostOptimizerConfig) *CostAwareEmbeddingService {
	service := &CostAwareEmbeddingService{
		EmbeddingService: baseService,
		costOptimizer:    NewCostOptimizer(config),
		providerMonitor:  NewProviderMonitor(),
		fallbackManager:  NewFallbackManager(&FallbackConfig{
			EnableAutoFallback:  true,
			FallbackStrategy:    "cost_optimized",
			MaxFallbackAttempts: 3,
			FallbackTimeout:     30 * time.Second,
			PreserveBudget:      true,
		}),
		budgetManager:   NewBudgetManager(&BudgetConfig{
			HourlyLimit:    config.HourlyBudget,
			DailyLimit:     config.DailyBudget,
			MonthlyLimit:   config.MonthlyBudget,
			ReservePercent: 10.0,
			AutoThrottle:   true,
		}),
		circuitBreakers: make(map[string]*CircuitBreaker),
	}
	
	// Initialize circuit breakers for each provider
	for name := range baseService.providers {
		service.circuitBreakers[name] = NewCircuitBreaker(name, &CircuitBreakerConfig{
			FailureThreshold: config.CircuitBreakerThreshold,
			SuccessThreshold: 3,
			Timeout:          config.CircuitBreakerTimeout,
			MaxRetries:       config.CircuitBreakerMaxRetries,
		})
	}
	
	// Start monitoring and optimization routines
	go service.startCostOptimizationLoop()
	go service.startProviderMonitoring()
	go service.startBudgetMonitoring()
	
	return service
}

// GenerateEmbeddingsOptimized generates embeddings with cost optimization
func (caes *CostAwareEmbeddingService) GenerateEmbeddingsOptimized(ctx context.Context, request *EmbeddingRequest) (*EmbeddingResponse, error) {
	startTime := time.Now()
	
	// Check budget constraints
	if !caes.budgetManager.CanProceed(request) {
		return nil, fmt.Errorf("budget constraints exceeded")
	}
	
	// Select optimal provider based on current conditions
	provider, err := caes.selectOptimalProvider(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("failed to select provider: %w", err)
	}
	
	// Check circuit breaker
	cb := caes.circuitBreakers[provider.GetName()]
	if !cb.CanAttempt() {
		caes.logger.Warn("Circuit breaker open for provider", "provider", provider.GetName())
		// Try fallback
		provider, err = caes.fallbackManager.GetFallbackProvider(provider.GetName(), caes.providers)
		if err != nil {
			return nil, fmt.Errorf("no available providers: %w", err)
		}
	}
	
	// Generate embeddings with selected provider
	response, err := caes.generateWithProviderAndFallback(ctx, request, provider)
	if err != nil {
		cb.RecordFailure()
		return nil, err
	}
	
	cb.RecordSuccess()
	
	// Update metrics and budget
	caes.updateCostMetrics(provider.GetName(), response.TokenUsage)
	caes.budgetManager.RecordSpend(response.TokenUsage.EstimatedCost)
	
	// Monitor quality if enabled
	if caes.config.EnableQualityCheck {
		go caes.monitorEmbeddingQuality(response.Embeddings, request.Texts)
	}
	
	response.ProcessingTime = time.Since(startTime)
	return response, nil
}

// selectOptimalProvider selects the best provider based on current conditions
func (caes *CostAwareEmbeddingService) selectOptimalProvider(ctx context.Context, request *EmbeddingRequest) (EmbeddingProvider, error) {
	caes.mu.RLock()
	defer caes.mu.RUnlock()
	
	var candidates []ProviderScore
	
	for name, provider := range caes.providers {
		// Skip unhealthy providers
		if !caes.isProviderHealthy(name) {
			continue
		}
		
		// Skip providers with open circuit breakers
		if cb, exists := caes.circuitBreakers[name]; exists && !cb.CanAttempt() {
			continue
		}
		
		// Calculate provider score
		score := caes.calculateProviderScore(name, provider, request)
		if score.TotalScore >= caes.costOptimizer.config.MinProviderScore {
			candidates = append(candidates, score)
		}
	}
	
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no suitable providers available")
	}
	
	// Sort by score (descending)
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].TotalScore > candidates[j].TotalScore
	})
	
	// Select best provider
	selectedProvider := caes.providers[candidates[0].ProviderName]
	
	caes.logger.Info("Selected provider",
		"provider", candidates[0].ProviderName,
		"score", candidates[0].TotalScore,
		"cost_score", candidates[0].CostScore,
		"performance_score", candidates[0].PerformanceScore,
		"quality_score", candidates[0].QualityScore)
	
	return selectedProvider, nil
}

// ProviderScore represents a provider's score for selection
type ProviderScore struct {
	ProviderName     string
	TotalScore       float64
	CostScore        float64
	PerformanceScore float64
	QualityScore     float64
	HealthScore      float64
}

// calculateProviderScore calculates a composite score for provider selection
func (caes *CostAwareEmbeddingService) calculateProviderScore(name string, provider EmbeddingProvider, request *EmbeddingRequest) ProviderScore {
	metrics := caes.providerMonitor.GetMetrics(name)
	config := caes.costOptimizer.config
	
	// Calculate individual scores (0-1 scale)
	costScore := caes.calculateCostScore(provider, metrics, request)
	performanceScore := caes.calculatePerformanceScore(metrics)
	qualityScore := caes.calculateQualityScore(metrics)
	healthScore := caes.calculateHealthScore(name, metrics)
	
	// Apply weights based on optimization strategy
	var totalScore float64
	switch config.OptimizationStrategy {
	case "aggressive":
		// Heavily favor cost
		totalScore = costScore*0.7 + performanceScore*0.2 + qualityScore*0.1
	case "balanced":
		// Balance all factors
		totalScore = costScore*config.CostWeight + 
			performanceScore*config.PerformanceWeight + 
			qualityScore*config.QualityWeight
	case "quality_first":
		// Prioritize quality and performance
		totalScore = costScore*0.2 + performanceScore*0.3 + qualityScore*0.5
	default:
		// Default balanced approach
		totalScore = costScore*0.4 + performanceScore*0.3 + qualityScore*0.3
	}
	
	// Apply health score as a multiplier
	totalScore *= healthScore
	
	return ProviderScore{
		ProviderName:     name,
		TotalScore:       totalScore,
		CostScore:        costScore,
		PerformanceScore: performanceScore,
		QualityScore:     qualityScore,
		HealthScore:      healthScore,
	}
}

// calculateCostScore calculates cost efficiency score
func (caes *CostAwareEmbeddingService) calculateCostScore(provider EmbeddingProvider, metrics *ProviderMetrics, request *EmbeddingRequest) float64 {
	// Estimate cost for this request
	estimatedTokens := caes.estimateTokenCount(request.Texts)
	estimatedCost := provider.GetCostEstimate(estimatedTokens)
	
	// Get budget remaining
	budgetRemaining := caes.budgetManager.GetRemainingBudget("daily")
	
	// Calculate cost efficiency relative to budget
	if budgetRemaining <= 0 {
		return 0.0 // No budget left
	}
	
	costRatio := estimatedCost / budgetRemaining
	if costRatio > 0.1 { // More than 10% of remaining budget
		return 0.0
	}
	
	// Calculate score based on relative cost
	// Lower cost = higher score
	maxAcceptableCost := budgetRemaining * 0.05 // 5% of remaining budget
	if estimatedCost >= maxAcceptableCost {
		return 0.0
	}
	
	score := 1.0 - (estimatedCost / maxAcceptableCost)
	
	// Adjust based on historical cost trend
	if len(metrics.CostTrend) > 0 {
		avgCost := average(metrics.CostTrend)
		if metrics.AverageCostPerToken < avgCost {
			score *= 1.1 // Bonus for improving cost efficiency
		}
	}
	
	return math.Min(score, 1.0)
}

// calculatePerformanceScore calculates performance score
func (caes *CostAwareEmbeddingService) calculatePerformanceScore(metrics *ProviderMetrics) float64 {
	if metrics.RequestCount == 0 {
		return 0.5 // Neutral score for new providers
	}
	
	// Base score on latency
	targetLatency := 500 * time.Millisecond
	latencyScore := 1.0
	if metrics.AverageLatency > targetLatency {
		latencyScore = float64(targetLatency) / float64(metrics.AverageLatency)
	}
	
	// Factor in success rate
	successScore := metrics.SuccessRate
	
	// Combine scores
	score := latencyScore*0.6 + successScore*0.4
	
	// Penalty for high error rate
	if metrics.ErrorRate > 0.05 { // More than 5% errors
		score *= (1.0 - metrics.ErrorRate)
	}
	
	return math.Min(math.Max(score, 0.0), 1.0)
}

// calculateQualityScore calculates embedding quality score
func (caes *CostAwareEmbeddingService) calculateQualityScore(metrics *ProviderMetrics) float64 {
	if metrics.AverageQuality == 0 {
		return 0.5 // Neutral score if no quality data
	}
	
	// Base quality score
	qualityScore := metrics.AverageQuality
	
	// Adjust for consistency (lower variance is better)
	if metrics.QualityVariance > 0 {
		consistencyScore := 1.0 / (1.0 + metrics.QualityVariance)
		qualityScore = qualityScore*0.7 + consistencyScore*0.3
	}
	
	// Consider quality trend
	if len(metrics.QualityTrend) > 2 {
		trend := calculateTrend(metrics.QualityTrend)
		if trend > 0 {
			qualityScore *= 1.1 // Bonus for improving quality
		} else if trend < -0.1 {
			qualityScore *= 0.9 // Penalty for declining quality
		}
	}
	
	return math.Min(math.Max(qualityScore, 0.0), 1.0)
}

// calculateHealthScore calculates provider health score
func (caes *CostAwareEmbeddingService) calculateHealthScore(name string, metrics *ProviderMetrics) float64 {
	// Check circuit breaker state
	if cb, exists := caes.circuitBreakers[name]; exists {
		switch cb.GetState() {
		case CircuitOpen:
			return 0.0
		case CircuitHalfOpen:
			return 0.5
		}
	}
	
	// Base health on recent performance
	healthScore := 1.0
	
	// Penalty for consecutive failures
	if metrics.ConsecutiveFailures > 0 {
		healthScore *= math.Pow(0.8, float64(metrics.ConsecutiveFailures))
	}
	
	// Consider time since last successful use
	timeSinceLastUse := time.Since(metrics.LastUsed)
	if timeSinceLastUse > 1*time.Hour {
		// Decay score for unused providers
		hoursUnused := timeSinceLastUse.Hours()
		healthScore *= math.Exp(-hoursUnused / 24.0) // Exponential decay over 24 hours
	}
	
	return math.Max(healthScore, 0.0)
}

// generateWithProviderAndFallback generates embeddings with automatic fallback
func (caes *CostAwareEmbeddingService) generateWithProviderAndFallback(ctx context.Context, request *EmbeddingRequest, primaryProvider EmbeddingProvider) (*EmbeddingResponse, error) {
	// Try primary provider
	response, err := caes.tryProvider(ctx, request, primaryProvider)
	if err == nil {
		return response, nil
	}
	
	caes.logger.Warn("Primary provider failed, attempting fallback",
		"provider", primaryProvider.GetName(),
		"error", err)
	
	// Record fallback attempt
	caes.fallbackManager.RecordFallback(primaryProvider.GetName(), err.Error())
	
	// Get fallback chain
	fallbackChain := caes.fallbackManager.GetFallbackChain(primaryProvider.GetName())
	
	// Try each fallback provider
	for _, fallbackName := range fallbackChain {
		fallbackProvider, exists := caes.providers[fallbackName]
		if !exists {
			continue
		}
		
		// Check if fallback is healthy
		if !caes.isProviderHealthy(fallbackName) {
			continue
		}
		
		// Check budget constraints for fallback
		if caes.budgetManager.ShouldPreserveBudget() {
			estimatedCost := fallbackProvider.GetCostEstimate(caes.estimateTokenCount(request.Texts))
			if estimatedCost > caes.budgetManager.GetMaxAllowableSpend() {
				caes.logger.Info("Skipping expensive fallback due to budget constraints",
					"provider", fallbackName,
					"estimated_cost", estimatedCost)
				continue
			}
		}
		
		response, err = caes.tryProvider(ctx, request, fallbackProvider)
		if err == nil {
			caes.logger.Info("Fallback provider succeeded", "provider", fallbackName)
			return response, nil
		}
		
		caes.logger.Warn("Fallback provider failed",
			"provider", fallbackName,
			"error", err)
	}
	
	return nil, fmt.Errorf("all providers failed, last error: %w", err)
}

// tryProvider attempts to generate embeddings with a specific provider
func (caes *CostAwareEmbeddingService) tryProvider(ctx context.Context, request *EmbeddingRequest, provider EmbeddingProvider) (*EmbeddingResponse, error) {
	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, caes.costOptimizer.config.ProviderTimeout)
	defer cancel()
	
	// Generate embeddings
	embeddings, usage, err := caes.generateEmbeddingsWithSpecificProvider(timeoutCtx, request.Texts, provider)
	if err != nil {
		return nil, err
	}
	
	// Create response
	response := &EmbeddingResponse{
		Embeddings:   embeddings,
		TokenUsage:   usage,
		ModelUsed:    provider.GetName(),
		RequestID:    request.RequestID,
		Metadata:     request.Metadata,
	}
	
	return response, nil
}

// isProviderHealthy checks if a provider is healthy
func (caes *CostAwareEmbeddingService) isProviderHealthy(providerName string) bool {
	health := caes.providerMonitor.GetHealthStatus(providerName)
	return health != nil && health.Healthy
}

// updateCostMetrics updates cost-related metrics
func (caes *CostAwareEmbeddingService) updateCostMetrics(provider string, usage TokenUsage) {
	caes.costOptimizer.RecordCost(provider, usage.EstimatedCost, usage.TotalTokens)
	caes.providerMonitor.UpdateMetrics(provider, usage)
}

// monitorEmbeddingQuality monitors the quality of generated embeddings
func (caes *CostAwareEmbeddingService) monitorEmbeddingQuality(embeddings [][]float32, texts []string) {
	// This runs asynchronously to not block the main request
	quality := caes.qualityAssess.AssessQuality(embeddings, texts)
	
	// Update provider quality metrics
	// Note: This assumes we can determine which provider was used
	// In practice, this would need to be passed through
}

// Background routines

func (caes *CostAwareEmbeddingService) startCostOptimizationLoop() {
	ticker := time.NewTicker(caes.costOptimizer.config.RebalanceInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		caes.rebalanceProviders()
	}
}

func (caes *CostAwareEmbeddingService) startProviderMonitoring() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		caes.performProviderHealthChecks()
	}
}

func (caes *CostAwareEmbeddingService) startBudgetMonitoring() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		caes.checkBudgetAlerts()
	}
}

func (caes *CostAwareEmbeddingService) rebalanceProviders() {
	// Analyze current provider performance and costs
	// Adjust fallback chains and preferences
	caes.logger.Info("Rebalancing providers based on performance and cost")
	
	// Implementation would analyze metrics and update provider preferences
}

func (caes *CostAwareEmbeddingService) performProviderHealthChecks() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	for name, provider := range caes.providers {
		go func(providerName string, p EmbeddingProvider) {
			health := caes.providerMonitor.CheckProviderHealth(ctx, providerName, p)
			if !health.Healthy {
				caes.logger.Warn("Provider health check failed",
					"provider", providerName,
					"error", health.Error)
			}
		}(name, provider)
	}
}

func (caes *CostAwareEmbeddingService) checkBudgetAlerts() {
	alerts := caes.budgetManager.CheckAlerts()
	for _, alert := range alerts {
		caes.logger.Warn("Budget alert",
			"type", alert.Type,
			"message", alert.Message,
			"current", alert.Current,
			"limit", alert.Limit)
	}
}

// Helper functions

func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateTrend(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	// Simple linear trend
	firstHalf := average(values[:len(values)/2])
	secondHalf := average(values[len(values)/2:])
	return (secondHalf - firstHalf) / firstHalf
}

// Component implementations

func NewCostOptimizer(config *CostOptimizerConfig) *CostOptimizer {
	return &CostOptimizer{
		config:          config,
		logger:          slog.Default().With("component", "cost-optimizer"),
		costHistory:     NewCostHistory(),
		providerMetrics: make(map[string]*ProviderMetrics),
	}
}

func NewProviderMonitor() *ProviderMonitor {
	return &ProviderMonitor{
		metrics:          make(map[string]*ProviderMetrics),
		healthChecks:     make(map[string]*HealthCheckResult),
		performanceStats: make(map[string]*PerformanceStats),
		logger:           slog.Default().With("component", "provider-monitor"),
	}
}

func NewFallbackManager(config *FallbackConfig) *FallbackManager {
	return &FallbackManager{
		config:          config,
		fallbackChains:  make(map[string][]string),
		fallbackHistory: make(map[string]*FallbackHistory),
		logger:          slog.Default().With("component", "fallback-manager"),
	}
}

func NewBudgetManager(config *BudgetConfig) *BudgetManager {
	return &BudgetManager{
		config:       config,
		currentSpend: &BudgetSpend{LastReset: time.Now()},
		spendHistory: make(map[string]float64),
		logger:       slog.Default().With("component", "budget-manager"),
	}
}

func NewCircuitBreaker(provider string, config *CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		provider: provider,
		state:    CircuitClosed,
		config:   config,
	}
}

func NewCostHistory() *CostHistory {
	return &CostHistory{
		hourlyHistory:  make(map[string][]CostDataPoint),
		dailyHistory:   make(map[string][]CostDataPoint),
		monthlyHistory: make(map[string][]CostDataPoint),
	}
}

// ProviderMonitor methods

func (pm *ProviderMonitor) GetMetrics(provider string) *ProviderMetrics {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	metrics, exists := pm.metrics[provider]
	if !exists {
		return &ProviderMetrics{
			SuccessRate:    1.0,
			AverageQuality: 0.8,
		}
	}
	return metrics
}

func (pm *ProviderMonitor) GetHealthStatus(provider string) *HealthCheckResult {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	return pm.healthChecks[provider]
}

func (pm *ProviderMonitor) CheckProviderHealth(ctx context.Context, provider string, p EmbeddingProvider) *HealthCheckResult {
	startTime := time.Now()
	
	err := p.HealthCheck(ctx)
	latency := time.Since(startTime)
	
	result := &HealthCheckResult{
		Provider:  provider,
		Healthy:   err == nil,
		Latency:   latency,
		Error:     err,
		Timestamp: time.Now(),
	}
	
	pm.mu.Lock()
	pm.healthChecks[provider] = result
	pm.mu.Unlock()
	
	return result
}

func (pm *ProviderMonitor) UpdateMetrics(provider string, usage TokenUsage) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	metrics, exists := pm.metrics[provider]
	if !exists {
		metrics = &ProviderMetrics{}
		pm.metrics[provider] = metrics
	}
	
	metrics.RequestCount++
	metrics.TokensProcessed += int64(usage.TotalTokens)
	metrics.TotalCost += usage.EstimatedCost
	metrics.LastUsed = time.Now()
	
	// Update cost trend
	metrics.CostTrend = append(metrics.CostTrend, usage.EstimatedCost)
	if len(metrics.CostTrend) > 100 {
		metrics.CostTrend = metrics.CostTrend[1:]
	}
}

// FallbackManager methods

func (fm *FallbackManager) GetFallbackChain(provider string) []string {
	fm.mu.RLock()
	defer fm.mu.RUnlock()
	
	chain, exists := fm.fallbackChains[provider]
	if !exists {
		// Return default fallback order
		return []string{"openai", "azure", "huggingface", "local"}
	}
	return chain
}

func (fm *FallbackManager) GetFallbackProvider(failedProvider string, availableProviders map[string]EmbeddingProvider) (EmbeddingProvider, error) {
	chain := fm.GetFallbackChain(failedProvider)
	
	for _, providerName := range chain {
		if providerName == failedProvider {
			continue
		}
		
		if provider, exists := availableProviders[providerName]; exists {
			return provider, nil
		}
	}
	
	return nil, fmt.Errorf("no fallback providers available")
}

func (fm *FallbackManager) RecordFallback(provider string, reason string) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	
	history, exists := fm.fallbackHistory[provider]
	if !exists {
		history = &FallbackHistory{
			FallbackReasons: make(map[string]int),
		}
		fm.fallbackHistory[provider] = history
	}
	
	history.FallbackCount++
	history.FallbackReasons[reason]++
	history.LastFallback = time.Now()
}

// BudgetManager methods

func (bm *BudgetManager) CanProceed(request *EmbeddingRequest) bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	
	// Check if any limits are exceeded
	if bm.currentSpend.HourlySpend >= bm.config.HourlyLimit ||
		bm.currentSpend.DailySpend >= bm.config.DailyLimit ||
		bm.currentSpend.MonthlySpend >= bm.config.MonthlyLimit {
		return false
	}
	
	return true
}

func (bm *BudgetManager) RecordSpend(amount float64) {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	
	now := time.Now()
	
	// Reset counters if needed
	if now.Hour() != bm.currentSpend.LastReset.Hour() {
		bm.currentSpend.HourlySpend = 0
	}
	if now.Day() != bm.currentSpend.LastReset.Day() {
		bm.currentSpend.DailySpend = 0
	}
	if now.Month() != bm.currentSpend.LastReset.Month() {
		bm.currentSpend.MonthlySpend = 0
	}
	
	bm.currentSpend.HourlySpend += amount
	bm.currentSpend.DailySpend += amount
	bm.currentSpend.MonthlySpend += amount
	bm.currentSpend.LastReset = now
	
	// Record in history
	hourKey := now.Format("2006-01-02-15")
	dayKey := now.Format("2006-01-02")
	bm.spendHistory[hourKey] += amount
	bm.spendHistory[dayKey] += amount
}

func (bm *BudgetManager) GetRemainingBudget(period string) float64 {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	
	switch period {
	case "hourly":
		return bm.config.HourlyLimit - bm.currentSpend.HourlySpend
	case "daily":
		return bm.config.DailyLimit - bm.currentSpend.DailySpend
	case "monthly":
		return bm.config.MonthlyLimit - bm.currentSpend.MonthlySpend
	default:
		return 0
	}
}

func (bm *BudgetManager) ShouldPreserveBudget() bool {
	bm.mu.RLock()
	defer bm.mu.RUnlock()
	
	// Check if we're close to any limit
	hourlyPercent := bm.currentSpend.HourlySpend / bm.config.HourlyLimit
	dailyPercent := bm.currentSpend.DailySpend / bm.config.DailyLimit
	monthlyPercent := bm.currentSpend.MonthlySpend / bm.config.MonthlyLimit
	
	threshold := (100.0 - bm.config.ReservePercent) / 100.0
	
	return hourlyPercent > threshold || dailyPercent > threshold || monthlyPercent > threshold
}

func (bm *BudgetManager) GetMaxAllowableSpend() float64 {
	remaining := bm.GetRemainingBudget("daily")
	reserve := bm.config.DailyLimit * (bm.config.ReservePercent / 100.0)
	
	allowable := remaining - reserve
	if allowable < 0 {
		return 0
	}
	return allowable
}

func (bm *BudgetManager) CheckAlerts() []BudgetAlert {
	bm.mu.Lock()
	defer bm.mu.Unlock()
	
	var alerts []BudgetAlert
	
	// Check hourly
	if bm.currentSpend.HourlySpend > bm.config.HourlyLimit*0.8 {
		alerts = append(alerts, BudgetAlert{
			Type:      "hourly",
			Threshold: 0.8,
			Current:   bm.currentSpend.HourlySpend,
			Limit:     bm.config.HourlyLimit,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Hourly spend at %.1f%% of limit", (bm.currentSpend.HourlySpend/bm.config.HourlyLimit)*100),
		})
	}
	
	// Check daily
	if bm.currentSpend.DailySpend > bm.config.DailyLimit*0.8 {
		alerts = append(alerts, BudgetAlert{
			Type:      "daily",
			Threshold: 0.8,
			Current:   bm.currentSpend.DailySpend,
			Limit:     bm.config.DailyLimit,
			Timestamp: time.Now(),
			Message:   fmt.Sprintf("Daily spend at %.1f%% of limit", (bm.currentSpend.DailySpend/bm.config.DailyLimit)*100),
		})
	}
	
	return alerts
}

// CircuitBreaker methods

func (cb *CircuitBreaker) CanAttempt() bool {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	
	switch cb.state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if timeout has passed
		if time.Since(cb.lastFailureTime) > cb.config.Timeout {
			cb.mu.RUnlock()
			cb.mu.Lock()
			cb.state = CircuitHalfOpen
			cb.successCount = 0
			cb.mu.Unlock()
			cb.mu.RLock()
			return true
		}
		return false
	case CircuitHalfOpen:
		return cb.successCount < cb.config.MaxRetries
	}
	
	return false
}

func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.lastAttemptTime = time.Now()
	
	switch cb.state {
	case CircuitHalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = CircuitClosed
			cb.failures = 0
			cb.successCount = 0
		}
	case CircuitClosed:
		cb.failures = 0
	}
}

func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	
	cb.lastAttemptTime = time.Now()
	cb.lastFailureTime = time.Now()
	cb.failures++
	
	switch cb.state {
	case CircuitClosed:
		if cb.failures >= cb.config.FailureThreshold {
			cb.state = CircuitOpen
		}
	case CircuitHalfOpen:
		cb.state = CircuitOpen
		cb.successCount = 0
	}
}

func (cb *CircuitBreaker) GetState() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// CostOptimizer methods

func (co *CostOptimizer) RecordCost(provider string, cost float64, tokens int) {
	co.mu.Lock()
	defer co.mu.Unlock()
	
	dataPoint := CostDataPoint{
		Timestamp:    time.Now(),
		Provider:     provider,
		Cost:         cost,
		TokenCount:   tokens,
		RequestCount: 1,
	}
	
	// Add to history
	hourKey := dataPoint.Timestamp.Format("2006-01-02-15")
	dayKey := dataPoint.Timestamp.Format("2006-01-02")
	monthKey := dataPoint.Timestamp.Format("2006-01")
	
	co.costHistory.mu.Lock()
	co.costHistory.hourlyHistory[hourKey] = append(co.costHistory.hourlyHistory[hourKey], dataPoint)
	co.costHistory.dailyHistory[dayKey] = append(co.costHistory.dailyHistory[dayKey], dataPoint)
	co.costHistory.monthlyHistory[monthKey] = append(co.costHistory.monthlyHistory[monthKey], dataPoint)
	co.costHistory.mu.Unlock()
	
	// Update provider metrics
	metrics, exists := co.providerMetrics[provider]
	if !exists {
		metrics = &ProviderMetrics{}
		co.providerMetrics[provider] = metrics
	}
	
	metrics.TotalCost += cost
	metrics.TokensProcessed += int64(tokens)
	if tokens > 0 {
		metrics.AverageCostPerToken = metrics.TotalCost / float64(metrics.TokensProcessed)
	}
}