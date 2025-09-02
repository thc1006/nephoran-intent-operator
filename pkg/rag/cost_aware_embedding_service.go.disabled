//go:build !disable_rag && !test

package rag

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// CostAwareEmbeddingService provides intelligent embedding provider selection.

type CostAwareEmbeddingService struct {
	logger *zap.Logger

	providers map[string]EmbeddingProvider

	providerConfigs map[string]*CostAwareProviderConfig

	costTracker *CostAwareCostTracker

	healthMonitor *HealthMonitor

	circuitBreakers map[string]*CircuitBreaker

	fallbackChains map[string][]string

	metrics *costAwareMetrics

	mu sync.RWMutex
}

// CostAwareProviderConfig holds configuration for each embedding provider.

type CostAwareProviderConfig struct {
	Name string

	CostPerToken float64

	CostPerRequest float64

	MaxTokensPerReq int

	Priority int

	QualityScore float64

	LatencyTarget time.Duration

	Enabled bool
}

// CostTracker tracks costs and budgets.

type CostAwareCostTracker struct {
	hourlyBudget float64

	dailyBudget float64

	monthlyBudget float64

	currentCosts *CostAccumulator

	historicalData map[string]*ProviderStats

	mu sync.RWMutex
}

// CostAccumulator tracks accumulated costs.

type CostAccumulator struct {
	hourly atomic.Value // float64

	daily atomic.Value // float64

	monthly atomic.Value // float64

	lastReset time.Time
}

// ProviderStats tracks provider performance statistics.

type ProviderStats struct {
	TotalRequests atomic.Int64

	SuccessfulReqs atomic.Int64

	FailedReqs atomic.Int64

	TotalCost atomic.Value // float64

	TotalLatency atomic.Int64 // nanoseconds

	AverageQuality atomic.Value // float64

	LastUsed atomic.Int64 // unix timestamp
}

// HealthMonitor monitors provider health.

type HealthMonitor struct {
	healthChecks map[string]*CostAwareHealthStatus

	checkInterval time.Duration

	mu sync.RWMutex
}

// HealthStatus represents provider health.

type CostAwareHealthStatus struct {
	Healthy atomic.Bool

	LastCheck atomic.Int64 // unix timestamp

	ConsecutiveFails atomic.Int32

	ResponseTime atomic.Int64 // milliseconds
}

// CircuitBreaker implements circuit breaker pattern.

type CircuitBreaker struct {
	state atomic.Int32 // 0=closed, 1=open, 2=half-open

	failures atomic.Int32

	lastFailure atomic.Int64 // unix timestamp

	successCount atomic.Int32

	config CircuitBreakerConfig
}

// CircuitBreakerConfig holds circuit breaker configuration.

type CircuitBreakerConfig struct {
	FailureThreshold int32

	RecoveryTimeout time.Duration

	SuccessThreshold int32
}

// CostAwareEmbeddingRequest represents a request for cost-aware embeddings.

type CostAwareEmbeddingRequest struct {
	Text string

	MaxBudget float64

	QualityRequired float64

	LatencyBudget time.Duration

	PreferredProvider string
}

// EmbeddingResponse represents the embedding result.

type CostAwareEmbeddingResponse struct {
	Embeddings []float64

	Provider string

	Cost float64

	Latency time.Duration

	Quality float64
}

// costAwareMetrics tracks performance metrics.

type costAwareMetrics struct {
	requestsTotal prometheus.Counter

	requestsPerProvider *prometheus.CounterVec

	costTotal prometheus.Counter

	costPerProvider *prometheus.CounterVec

	providerLatency *prometheus.HistogramVec

	providerQuality *prometheus.GaugeVec

	budgetExceeded prometheus.Counter

	fallbacksUsed prometheus.Counter

	circuitBreakerTrips prometheus.Counter
}

// NewCostAwareEmbeddingService creates a new cost-aware embedding service.

func NewCostAwareEmbeddingService(logger *zap.Logger) *CostAwareEmbeddingService {
	metrics := &costAwareMetrics{
		requestsTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "embedding_requests_total",

			Help: "Total number of embedding requests",
		}),

		requestsPerProvider: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "embedding_requests_per_provider_total",

			Help: "Embedding requests per provider",
		}, []string{"provider"}),

		costTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "embedding_cost_total",

			Help: "Total cost of embeddings",
		}),

		costPerProvider: prometheus.NewCounterVec(prometheus.CounterOpts{
			Name: "embedding_cost_per_provider_total",

			Help: "Cost per embedding provider",
		}, []string{"provider"}),

		providerLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Name: "embedding_provider_latency_seconds",

			Help: "Latency of embedding providers",

			Buckets: prometheus.DefBuckets,
		}, []string{"provider"}),

		providerQuality: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "embedding_provider_quality",

			Help: "Quality score of embedding providers",
		}, []string{"provider"}),

		budgetExceeded: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "embedding_budget_exceeded_total",

			Help: "Number of times budget was exceeded",
		}),

		fallbacksUsed: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "embedding_fallbacks_used_total",

			Help: "Number of times fallback providers were used",
		}),

		circuitBreakerTrips: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "embedding_circuit_breaker_trips_total",

			Help: "Number of circuit breaker trips",
		}),
	}

	// Register metrics.

	prometheus.MustRegister(

		metrics.requestsTotal,

		metrics.requestsPerProvider,

		metrics.costTotal,

		metrics.costPerProvider,

		metrics.providerLatency,

		metrics.providerQuality,

		metrics.budgetExceeded,

		metrics.fallbacksUsed,

		metrics.circuitBreakerTrips,
	)

	service := &CostAwareEmbeddingService{
		logger: logger,

		providers: make(map[string]EmbeddingProvider),

		providerConfigs: make(map[string]*CostAwareProviderConfig),

		circuitBreakers: make(map[string]*CircuitBreaker),

		fallbackChains: make(map[string][]string),

		metrics: metrics,

		costTracker: &CostAwareCostTracker{
			currentCosts: &CostAccumulator{
				lastReset: time.Now(),
			},

			historicalData: make(map[string]*ProviderStats),
		},

		healthMonitor: &HealthMonitor{
			healthChecks: make(map[string]*CostAwareHealthStatus),

			checkInterval: 30 * time.Second,
		},
	}

	// Initialize atomic values.

	service.costTracker.currentCosts.hourly.Store(0.0)

	service.costTracker.currentCosts.daily.Store(0.0)

	service.costTracker.currentCosts.monthly.Store(0.0)

	// Start background tasks.

	go service.monitorHealth()

	go service.resetCostTracking()

	return service
}

// ConfigureProvider adds or updates a provider configuration.

func (s *CostAwareEmbeddingService) ConfigureProvider(config CostAwareProviderConfig) {
	s.mu.Lock()

	defer s.mu.Unlock()

	s.providerConfigs[config.Name] = &config

	// Initialize circuit breaker.

	if _, exists := s.circuitBreakers[config.Name]; !exists {
		s.circuitBreakers[config.Name] = &CircuitBreaker{
			config: CircuitBreakerConfig{
				FailureThreshold: 5,

				RecoveryTimeout: 30 * time.Second,

				SuccessThreshold: 3,
			},
		}
	}

	// Initialize health status.

	if _, exists := s.healthMonitor.healthChecks[config.Name]; !exists {

		status := &CostAwareHealthStatus{}

		status.Healthy.Store(true)

		s.healthMonitor.healthChecks[config.Name] = status

	}

	// Initialize provider stats.

	if _, exists := s.costTracker.historicalData[config.Name]; !exists {

		stats := &ProviderStats{}

		stats.TotalCost.Store(0.0)

		stats.AverageQuality.Store(config.QualityScore)

		s.costTracker.historicalData[config.Name] = stats

	}

	s.logger.Info("Configured embedding provider",

		zap.String("provider", config.Name),

		zap.Float64("cost_per_token", config.CostPerToken),

		zap.Float64("quality_score", config.QualityScore))
}

// SetBudgets sets cost budgets.

func (s *CostAwareEmbeddingService) SetBudgets(hourly, daily, monthly float64) {
	s.costTracker.mu.Lock()

	defer s.costTracker.mu.Unlock()

	s.costTracker.hourlyBudget = hourly

	s.costTracker.dailyBudget = daily

	s.costTracker.monthlyBudget = monthly
}

// SetFallbackChain sets fallback providers for a primary provider.

func (s *CostAwareEmbeddingService) SetFallbackChain(primary string, fallbacks []string) {
	s.mu.Lock()

	defer s.mu.Unlock()

	s.fallbackChains[primary] = fallbacks
}

// GetEmbeddings generates embeddings with cost optimization.

func (s *CostAwareEmbeddingService) GetEmbeddings(
	ctx context.Context,

	request CostAwareEmbeddingRequest,
) (*CostAwareEmbeddingResponse, error) {
	s.metrics.requestsTotal.Inc()

	// Select optimal provider.

	provider, providerName := s.selectProvider(request)

	if provider == nil {
		return nil, fmt.Errorf("no available provider found")
	}

	// Try primary provider.

	response, err := s.tryProvider(ctx, provider, providerName, request)

	if err == nil {
		return response, nil
	}

	// Try fallback providers.

	s.metrics.fallbacksUsed.Inc()

	fallbacks := s.getFallbackChain(providerName)

	for _, fallbackName := range fallbacks {

		fallbackProvider := s.providers[fallbackName]

		if fallbackProvider == nil {
			continue
		}

		response, err = s.tryProvider(ctx, fallbackProvider, fallbackName, request)

		if err == nil {

			s.logger.Info("Used fallback provider",

				zap.String("primary", providerName),

				zap.String("fallback", fallbackName))

			return response, nil

		}

	}

	return nil, fmt.Errorf("all providers failed for request")
}

// selectProvider selects the optimal provider based on constraints.

func (s *CostAwareEmbeddingService) selectProvider(request CostAwareEmbeddingRequest) (EmbeddingProvider, string) {
	s.mu.RLock()

	defer s.mu.RUnlock()

	// If preferred provider is specified and available, use it.

	if request.PreferredProvider != "" {
		if provider, exists := s.providers[request.PreferredProvider]; exists {
			if s.isProviderAvailable(request.PreferredProvider) {
				return provider, request.PreferredProvider
			}
		}
	}

	// Score and rank providers.

	type providerScore struct {
		name string

		provider EmbeddingProvider

		score float64
	}

	var candidates []providerScore

	for name, provider := range s.providers {

		if !s.isProviderAvailable(name) {
			continue
		}

		config := s.providerConfigs[name]

		if config == nil || !config.Enabled {
			continue
		}

		// Calculate provider score.

		score := s.calculateProviderScore(name, config, request)

		candidates = append(candidates, providerScore{
			name: name,

			provider: provider,

			score: score,
		})

	}

	// Sort by score (higher is better).

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].score > candidates[j].score
	})

	if len(candidates) > 0 {
		return candidates[0].provider, candidates[0].name
	}

	return nil, ""
}

// calculateProviderScore calculates a provider's suitability score.

func (s *CostAwareEmbeddingService) calculateProviderScore(
	name string,

	config *CostAwareProviderConfig,

	request CostAwareEmbeddingRequest,
) float64 {
	score := 100.0

	// Cost factor (40% weight).

	estimatedCost := s.estimateCost(config, request.Text)

	if request.MaxBudget > 0 && estimatedCost > request.MaxBudget {
		return 0 // Provider exceeds budget
	}

	costScore := math.Max(0, 1-estimatedCost/request.MaxBudget) * 40

	score = score * (1 + costScore/100)

	// Quality factor (30% weight).

	if request.QualityRequired > 0 {

		if config.QualityScore < request.QualityRequired {
			return 0 // Provider doesn't meet quality requirements
		}

		qualityScore := config.QualityScore / request.QualityRequired * 30

		score = score * (1 + qualityScore/100)

	}

	// Performance factor (20% weight).

	stats := s.costTracker.historicalData[name]

	if stats != nil {

		successRate := float64(stats.SuccessfulReqs.Load()) / float64(stats.TotalRequests.Load()+1)

		perfScore := successRate * 20

		score = score * (1 + perfScore/100)

	}

	// Latency factor (10% weight).

	if request.LatencyBudget > 0 && config.LatencyTarget > request.LatencyBudget {

		latencyPenalty := float64(config.LatencyTarget-request.LatencyBudget) / float64(request.LatencyBudget)

		score = score * (1 - latencyPenalty*0.1)

	}

	// Apply priority boost.

	score = score * (1 + float64(config.Priority)/100)

	return score
}

// tryProvider attempts to get embeddings from a specific provider.

func (s *CostAwareEmbeddingService) tryProvider(
	ctx context.Context,

	provider EmbeddingProvider,

	providerName string,

	request CostAwareEmbeddingRequest,
) (*CostAwareEmbeddingResponse, error) {
	// Check circuit breaker.

	if !s.checkCircuitBreaker(providerName) {
		return nil, fmt.Errorf("circuit breaker open for provider %s", providerName)
	}

	start := time.Now()

	// Get embeddings.

	embeddings, err := provider.GenerateBatchEmbeddings(ctx, []string{request.Text})

	duration := time.Since(start)

	// Update metrics.

	s.metrics.requestsPerProvider.WithLabelValues(providerName).Inc()

	s.metrics.providerLatency.WithLabelValues(providerName).Observe(duration.Seconds())

	if err != nil {

		s.recordFailure(providerName)

		return nil, err

	}

	// Calculate cost.

	config := s.providerConfigs[providerName]

	cost := s.estimateCost(config, request.Text)

	// Update cost tracking.

	s.updateCostTracking(providerName, cost)

	// Record success.

	s.recordSuccess(providerName, duration)

	// Convert embeddings for quality assessment (float32 to float64).

	embedding64 := make([]float64, len(embeddings[0]))

	for i, v := range embeddings[0] {
		embedding64[i] = float64(v)
	}

	// Assess quality.

	quality := s.assessQuality(embedding64)

	response := &CostAwareEmbeddingResponse{
		Embeddings: embedding64,

		Provider: providerName,

		Cost: cost,

		Latency: duration,

		Quality: quality,
	}

	return response, nil
}

// estimateCost estimates the cost for a request.

func (s *CostAwareEmbeddingService) estimateCost(config *CostAwareProviderConfig, text string) float64 {
	// Simple token estimation (rough approximation).

	tokenCount := len(text) / 4

	return config.CostPerRequest + (float64(tokenCount) * config.CostPerToken)
}

// assessQuality assesses the quality of embeddings.

func (s *CostAwareEmbeddingService) assessQuality(embeddings []float64) float64 {
	// Simple quality assessment based on embedding properties.

	// In practice, this would use more sophisticated methods.

	// Check embedding magnitude.

	magnitude := 0.0

	for _, val := range embeddings {
		magnitude += val * val
	}

	magnitude = math.Sqrt(magnitude)

	// Check for reasonable values.

	if magnitude < 0.1 || magnitude > 10.0 {
		return 0.5 // Low quality
	}

	// Check dimension.

	if len(embeddings) < 384 {
		return 0.7 // Medium quality
	}

	return 0.9 // High quality
}

// isProviderAvailable checks if a provider is available.

func (s *CostAwareEmbeddingService) isProviderAvailable(name string) bool {
	// Check if provider exists.

	if _, exists := s.providers[name]; !exists {
		return false
	}

	// Check health status.

	if health, exists := s.healthMonitor.healthChecks[name]; exists {
		if !health.Healthy.Load() {
			return false
		}
	}

	// Check circuit breaker.

	if !s.checkCircuitBreaker(name) {
		return false
	}

	// Check budget.

	if !s.checkBudget() {
		return false
	}

	return true
}

// checkCircuitBreaker checks if circuit breaker allows requests.

func (s *CostAwareEmbeddingService) checkCircuitBreaker(name string) bool {
	breaker, exists := s.circuitBreakers[name]

	if !exists {
		return true
	}

	state := breaker.state.Load()

	switch state {

	case 0: // Closed

		return true

	case 1: // Open

		// Check if recovery timeout has passed.

		lastFailure := time.Unix(breaker.lastFailure.Load(), 0)

		if time.Since(lastFailure) > breaker.config.RecoveryTimeout {

			breaker.state.Store(2) // Move to half-open

			return true

		}

		return false

	case 2: // Half-open

		return true

	default:

		return false

	}
}

// checkBudget checks if we're within budget limits.

func (s *CostAwareEmbeddingService) checkBudget() bool {
	s.costTracker.mu.RLock()

	defer s.costTracker.mu.RUnlock()

	hourly := s.costTracker.currentCosts.hourly.Load().(float64)

	daily := s.costTracker.currentCosts.daily.Load().(float64)

	monthly := s.costTracker.currentCosts.monthly.Load().(float64)

	if s.costTracker.hourlyBudget > 0 && hourly >= s.costTracker.hourlyBudget {

		s.metrics.budgetExceeded.Inc()

		return false

	}

	if s.costTracker.dailyBudget > 0 && daily >= s.costTracker.dailyBudget {

		s.metrics.budgetExceeded.Inc()

		return false

	}

	if s.costTracker.monthlyBudget > 0 && monthly >= s.costTracker.monthlyBudget {

		s.metrics.budgetExceeded.Inc()

		return false

	}

	return true
}

// updateCostTracking updates cost tracking.

func (s *CostAwareEmbeddingService) updateCostTracking(provider string, cost float64) {
	// Update current costs.

	for {

		oldHourly := s.costTracker.currentCosts.hourly.Load().(float64)

		if s.costTracker.currentCosts.hourly.CompareAndSwap(oldHourly, oldHourly+cost) {
			break
		}

	}

	for {

		oldDaily := s.costTracker.currentCosts.daily.Load().(float64)

		if s.costTracker.currentCosts.daily.CompareAndSwap(oldDaily, oldDaily+cost) {
			break
		}

	}

	for {

		oldMonthly := s.costTracker.currentCosts.monthly.Load().(float64)

		if s.costTracker.currentCosts.monthly.CompareAndSwap(oldMonthly, oldMonthly+cost) {
			break
		}

	}

	// Update provider stats.

	if stats, exists := s.costTracker.historicalData[provider]; exists {
		for {

			oldCost := stats.TotalCost.Load().(float64)

			if stats.TotalCost.CompareAndSwap(oldCost, oldCost+cost) {
				break
			}

		}
	}

	// Update metrics.

	s.metrics.costTotal.Add(cost)

	s.metrics.costPerProvider.WithLabelValues(provider).Add(cost)
}

// recordSuccess records a successful request.

func (s *CostAwareEmbeddingService) recordSuccess(provider string, duration time.Duration) {
	if stats, exists := s.costTracker.historicalData[provider]; exists {

		stats.TotalRequests.Add(1)

		stats.SuccessfulReqs.Add(1)

		stats.TotalLatency.Add(int64(duration))

		stats.LastUsed.Store(time.Now().Unix())

	}

	// Update circuit breaker.

	if breaker, exists := s.circuitBreakers[provider]; exists {

		breaker.failures.Store(0)

		if breaker.state.Load() == 2 { // Half-open

			breaker.successCount.Add(1)

			if breaker.successCount.Load() >= breaker.config.SuccessThreshold {

				breaker.state.Store(0) // Close circuit

				breaker.successCount.Store(0)

			}

		}

	}
}

// recordFailure records a failed request.

func (s *CostAwareEmbeddingService) recordFailure(provider string) {
	if stats, exists := s.costTracker.historicalData[provider]; exists {

		stats.TotalRequests.Add(1)

		stats.FailedReqs.Add(1)

	}

	// Update circuit breaker.

	if breaker, exists := s.circuitBreakers[provider]; exists {

		failures := breaker.failures.Add(1)

		breaker.lastFailure.Store(time.Now().Unix())

		if failures >= breaker.config.FailureThreshold {
			if breaker.state.CompareAndSwap(0, 1) { // Close to Open

				s.metrics.circuitBreakerTrips.Inc()

				s.logger.Warn("Circuit breaker opened",

					zap.String("provider", provider),

					zap.Int32("failures", failures))

			}
		}

		if breaker.state.Load() == 2 { // Half-open

			breaker.state.Store(1) // Back to open

			breaker.successCount.Store(0)

		}

	}
}

// getFallbackChain returns the fallback chain for a provider.

func (s *CostAwareEmbeddingService) getFallbackChain(provider string) []string {
	s.mu.RLock()

	defer s.mu.RUnlock()

	return s.fallbackChains[provider]
}

// monitorHealth periodically checks provider health.

func (s *CostAwareEmbeddingService) monitorHealth() {
	ticker := time.NewTicker(s.healthMonitor.checkInterval)

	defer ticker.Stop()

	for range ticker.C {

		s.mu.RLock()

		providers := make(map[string]EmbeddingProvider)

		for k, v := range s.providers {
			providers[k] = v
		}

		s.mu.RUnlock()

		for name, provider := range providers {
			go s.checkProviderHealth(name, provider)
		}

	}
}

// checkProviderHealth checks the health of a single provider.

func (s *CostAwareEmbeddingService) checkProviderHealth(name string, provider EmbeddingProvider) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	start := time.Now()

	// Simple health check - try to get embeddings for a small text.

	_, err := provider.GenerateBatchEmbeddings(ctx, []string{"health check"})

	duration := time.Since(start)

	health, exists := s.healthMonitor.healthChecks[name]

	if !exists {
		return
	}

	health.LastCheck.Store(time.Now().Unix())

	health.ResponseTime.Store(duration.Milliseconds())

	if err != nil {

		health.ConsecutiveFails.Add(1)

		if health.ConsecutiveFails.Load() >= 3 {

			health.Healthy.Store(false)

			s.logger.Warn("Provider marked unhealthy",

				zap.String("provider", name),

				zap.Error(err))

		}

	} else {

		health.ConsecutiveFails.Store(0)

		health.Healthy.Store(true)

	}
}

// resetCostTracking periodically resets cost accumulators.

func (s *CostAwareEmbeddingService) resetCostTracking() {
	ticker := time.NewTicker(time.Minute)

	defer ticker.Stop()

	for range ticker.C {

		now := time.Now()

		// Reset hourly costs.

		if now.Hour() != s.costTracker.currentCosts.lastReset.Hour() {
			s.costTracker.currentCosts.hourly.Store(0.0)
		}

		// Reset daily costs.

		if now.Day() != s.costTracker.currentCosts.lastReset.Day() {
			s.costTracker.currentCosts.daily.Store(0.0)
		}

		// Reset monthly costs.

		if now.Month() != s.costTracker.currentCosts.lastReset.Month() {
			s.costTracker.currentCosts.monthly.Store(0.0)
		}

		s.costTracker.currentCosts.lastReset = now

	}
}
