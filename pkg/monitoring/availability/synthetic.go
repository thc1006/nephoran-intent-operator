package availability

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// SyntheticCheckType represents the type of synthetic check
type SyntheticCheckType string

const (
	CheckTypeHTTP       SyntheticCheckType = "http"
	CheckTypeIntentFlow SyntheticCheckType = "intent_flow"
	CheckTypeDatabase   SyntheticCheckType = "database"
	CheckTypeExternal   SyntheticCheckType = "external"
	CheckTypeChaos      SyntheticCheckType = "chaos"
)

// SyntheticCheckStatus represents the status of a synthetic check
type SyntheticCheckStatus string

const (
	CheckStatusPass    SyntheticCheckStatus = "pass"
	CheckStatusFail    SyntheticCheckStatus = "fail"
	CheckStatusTimeout SyntheticCheckStatus = "timeout"
	CheckStatusError   SyntheticCheckStatus = "error"
)

// SyntheticCheck defines a synthetic monitoring check
type SyntheticCheck struct {
	ID              string             `json:"id"`
	Name            string             `json:"name"`
	Type            SyntheticCheckType `json:"type"`
	Enabled         bool               `json:"enabled"`
	Interval        time.Duration      `json:"interval"`
	Timeout         time.Duration      `json:"timeout"`
	RetryCount      int                `json:"retry_count"`
	RetryDelay      time.Duration      `json:"retry_delay"`
	BusinessImpact  BusinessImpact     `json:"business_impact"`
	Region          string             `json:"region"`
	Tags            map[string]string  `json:"tags"`
	Config          CheckConfig        `json:"config"`
	AlertThresholds AlertThresholds    `json:"alert_thresholds"`
}

// CheckConfig holds configuration specific to check types
type CheckConfig struct {
	// HTTP check configuration
	URL             string            `json:"url,omitempty"`
	Method          string            `json:"method,omitempty"`
	Headers         map[string]string `json:"headers,omitempty"`
	Body            string            `json:"body,omitempty"`
	ExpectedStatus  int               `json:"expected_status,omitempty"`
	ExpectedBody    string            `json:"expected_body,omitempty"`
	FollowRedirects bool              `json:"follow_redirects,omitempty"`
	SkipTLS         bool              `json:"skip_tls,omitempty"`

	// Intent flow configuration
	IntentPayload    map[string]interface{} `json:"intent_payload,omitempty"`
	ExpectedResponse map[string]interface{} `json:"expected_response,omitempty"`
	FlowSteps        []IntentFlowStep       `json:"flow_steps,omitempty"`

	// Database configuration
	ConnectionString string `json:"connection_string,omitempty"`
	Query            string `json:"query,omitempty"`
	ExpectedRows     int    `json:"expected_rows,omitempty"`

	// External service configuration
	ServiceName     string `json:"service_name,omitempty"`
	ServiceEndpoint string `json:"service_endpoint,omitempty"`

	// Chaos testing configuration
	ChaosType      string        `json:"chaos_type,omitempty"`
	ChaosDuration  time.Duration `json:"chaos_duration,omitempty"`
	ChaosIntensity float64       `json:"chaos_intensity,omitempty"`
}

// IntentFlowStep represents a step in an intent processing flow
type IntentFlowStep struct {
	Name            string           `json:"name"`
	Action          string           `json:"action"` // create_intent, check_status, validate_deployment
	Payload         interface{}      `json:"payload"`
	ExpectedStatus  string           `json:"expected_status"`
	MaxWaitTime     time.Duration    `json:"max_wait_time"`
	ValidationRules []ValidationRule `json:"validation_rules"`
}

// ValidationRule defines validation criteria for synthetic checks
type ValidationRule struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"` // equals, contains, greater_than, less_than
	Value    interface{} `json:"value"`
}

// AlertThresholds defines thresholds for alerting
type AlertThresholds struct {
	ResponseTime     time.Duration `json:"response_time"`     // Alert if response time exceeds this
	ErrorRate        float64       `json:"error_rate"`        // Alert if error rate exceeds this (0-1)
	Availability     float64       `json:"availability"`      // Alert if availability drops below this (0-1)
	ConsecutiveFails int           `json:"consecutive_fails"` // Alert after this many consecutive failures
}

// SyntheticResult represents the result of a synthetic check execution
type SyntheticResult struct {
	CheckID      string                 `json:"check_id"`
	CheckName    string                 `json:"check_name"`
	Timestamp    time.Time              `json:"timestamp"`
	Status       SyntheticCheckStatus   `json:"status"`
	ResponseTime time.Duration          `json:"response_time"`
	Error        string                 `json:"error,omitempty"`
	Region       string                 `json:"region"`
	HTTPStatus   int                    `json:"http_status,omitempty"`
	StepResults  []StepResult           `json:"step_results,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// StepResult represents the result of a single step in a multi-step check
type StepResult struct {
	StepName     string               `json:"step_name"`
	Status       SyntheticCheckStatus `json:"status"`
	ResponseTime time.Duration        `json:"response_time"`
	Error        string               `json:"error,omitempty"`
	Output       interface{}          `json:"output,omitempty"`
}

// SyntheticMonitor manages synthetic monitoring checks
type SyntheticMonitor struct {
	checks       map[string]*SyntheticCheck
	checksMutex  sync.RWMutex
	results      []SyntheticResult
	resultsMutex sync.RWMutex

	// Configuration
	config *SyntheticMonitorConfig

	// Clients
	httpClient *http.Client
	promClient v1.API

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	stopCh chan struct{}

	// Observability
	tracer trace.Tracer

	// Check execution
	executors map[SyntheticCheckType]CheckExecutor

	// Alerting
	alertManager AlertManager
}

// SyntheticMonitorConfig holds configuration for synthetic monitoring
type SyntheticMonitorConfig struct {
	MaxConcurrentChecks int           `json:"max_concurrent_checks"`
	DefaultTimeout      time.Duration `json:"default_timeout"`
	DefaultRetryCount   int           `json:"default_retry_count"`
	DefaultRetryDelay   time.Duration `json:"default_retry_delay"`
	ResultRetention     time.Duration `json:"result_retention"`
	RegionID            string        `json:"region_id"`
	EnableChaosTests    bool          `json:"enable_chaos_tests"`
	ChaosTestInterval   time.Duration `json:"chaos_test_interval"`

	// HTTP client configuration
	HTTPTimeout         time.Duration `json:"http_timeout"`
	HTTPMaxIdleConns    int           `json:"http_max_idle_conns"`
	HTTPMaxConnsPerHost int           `json:"http_max_conns_per_host"`
	HTTPSkipTLS         bool          `json:"http_skip_tls"`

	// Intent flow configuration
	IntentAPIEndpoint string `json:"intent_api_endpoint"`
	IntentAPIToken    string `json:"intent_api_token"`

	// Alerting configuration
	AlertingEnabled bool          `json:"alerting_enabled"`
	AlertWebhookURL string        `json:"alert_webhook_url"`
	AlertRetention  time.Duration `json:"alert_retention"`
}

// CheckExecutor defines the interface for executing different types of checks
type CheckExecutor interface {
	Execute(ctx context.Context, check *SyntheticCheck) (*SyntheticResult, error)
	Type() SyntheticCheckType
}

// AlertManager handles alerting for synthetic check failures
type AlertManager interface {
	SendAlert(ctx context.Context, check *SyntheticCheck, result *SyntheticResult) error
	EvaluateThresholds(ctx context.Context, check *SyntheticCheck, results []SyntheticResult) (bool, error)
}

// NewSyntheticMonitor creates a new synthetic monitor
func NewSyntheticMonitor(config *SyntheticMonitorConfig, promClient api.Client, alertManager AlertManager) (*SyntheticMonitor, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Configure HTTP client with sensible defaults
	httpClient := &http.Client{
		Timeout: config.HTTPTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        config.HTTPMaxIdleConns,
			MaxIdleConnsPerHost: config.HTTPMaxConnsPerHost,
			IdleConnTimeout:     90 * time.Second,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.HTTPSkipTLS,
			},
		},
	}

	var promAPI v1.API
	if promClient != nil {
		promAPI = v1.NewAPI(promClient)
	}

	monitor := &SyntheticMonitor{
		checks:       make(map[string]*SyntheticCheck),
		results:      make([]SyntheticResult, 0, 10000),
		config:       config,
		httpClient:   httpClient,
		promClient:   promAPI,
		ctx:          ctx,
		cancel:       cancel,
		stopCh:       make(chan struct{}),
		tracer:       otel.Tracer("synthetic-monitor"),
		executors:    make(map[SyntheticCheckType]CheckExecutor),
		alertManager: alertManager,
	}

	// Initialize executors
	monitor.initializeExecutors()

	return monitor, nil
}

// initializeExecutors initializes check executors
func (sm *SyntheticMonitor) initializeExecutors() {
	sm.executors[CheckTypeHTTP] = NewHTTPCheckExecutor(sm.httpClient)
	sm.executors[CheckTypeIntentFlow] = NewIntentFlowExecutor(sm.httpClient, sm.config.IntentAPIEndpoint, sm.config.IntentAPIToken)
	sm.executors[CheckTypeDatabase] = NewDatabaseCheckExecutor()
	sm.executors[CheckTypeExternal] = NewExternalServiceExecutor(sm.httpClient)
	if sm.config.EnableChaosTests {
		sm.executors[CheckTypeChaos] = NewChaosCheckExecutor()
	}
}

// Start begins synthetic monitoring
func (sm *SyntheticMonitor) Start() error {
	ctx, span := sm.tracer.Start(sm.ctx, "synthetic-monitor-start")
	defer span.End()

	span.AddEvent("Starting synthetic monitor")

	// Start check execution goroutines
	go sm.runCheckExecution(ctx)

	// Start result cleanup routine
	go sm.runResultCleanup(ctx)

	// Start chaos testing if enabled
	if sm.config.EnableChaosTests {
		go sm.runChaosTests(ctx)
	}

	return nil
}

// Stop stops synthetic monitoring
func (sm *SyntheticMonitor) Stop() error {
	sm.cancel()
	close(sm.stopCh)
	return nil
}

// AddCheck adds a new synthetic check
func (sm *SyntheticMonitor) AddCheck(check *SyntheticCheck) error {
	if check == nil {
		return fmt.Errorf("check cannot be nil")
	}

	if check.ID == "" {
		return fmt.Errorf("check ID cannot be empty")
	}

	// Apply defaults
	if check.Interval == 0 {
		check.Interval = 30 * time.Second
	}
	if check.Timeout == 0 {
		check.Timeout = sm.config.DefaultTimeout
	}
	if check.RetryCount == 0 {
		check.RetryCount = sm.config.DefaultRetryCount
	}
	if check.RetryDelay == 0 {
		check.RetryDelay = sm.config.DefaultRetryDelay
	}
	if check.Region == "" {
		check.Region = sm.config.RegionID
	}

	sm.checksMutex.Lock()
	defer sm.checksMutex.Unlock()

	sm.checks[check.ID] = check

	return nil
}

// RemoveCheck removes a synthetic check
func (sm *SyntheticMonitor) RemoveCheck(checkID string) error {
	sm.checksMutex.Lock()
	defer sm.checksMutex.Unlock()

	delete(sm.checks, checkID)
	return nil
}

// GetCheck retrieves a synthetic check by ID
func (sm *SyntheticMonitor) GetCheck(checkID string) (*SyntheticCheck, bool) {
	sm.checksMutex.RLock()
	defer sm.checksMutex.RUnlock()

	check, exists := sm.checks[checkID]
	return check, exists
}

// ListChecks returns all synthetic checks
func (sm *SyntheticMonitor) ListChecks() []*SyntheticCheck {
	sm.checksMutex.RLock()
	defer sm.checksMutex.RUnlock()

	checks := make([]*SyntheticCheck, 0, len(sm.checks))
	for _, check := range sm.checks {
		checks = append(checks, check)
	}

	return checks
}

// runCheckExecution runs the main check execution loop
func (sm *SyntheticMonitor) runCheckExecution(ctx context.Context) {
	// Create a semaphore to limit concurrent check executions
	semaphore := make(chan struct{}, sm.config.MaxConcurrentChecks)

	// Schedule checks based on their intervals
	checkScheduler := make(map[string]*time.Ticker)

	sm.checksMutex.RLock()
	for _, check := range sm.checks {
		if check.Enabled {
			ticker := time.NewTicker(check.Interval)
			checkScheduler[check.ID] = ticker

			go sm.scheduleCheck(ctx, check, ticker, semaphore)
		}
	}
	sm.checksMutex.RUnlock()

	<-ctx.Done()

	// Cleanup tickers
	for _, ticker := range checkScheduler {
		ticker.Stop()
	}
}

// scheduleCheck schedules execution of a single check
func (sm *SyntheticMonitor) scheduleCheck(ctx context.Context, check *SyntheticCheck, ticker *time.Ticker, semaphore chan struct{}) {
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Acquire semaphore
			select {
			case semaphore <- struct{}{}:
				go func() {
					defer func() { <-semaphore }()
					sm.executeCheck(ctx, check)
				}()
			default:
				// Skip execution if too many checks are running
				continue
			}
		}
	}
}

// executeCheck executes a single synthetic check
func (sm *SyntheticMonitor) executeCheck(ctx context.Context, check *SyntheticCheck) {
	ctx, span := sm.tracer.Start(ctx, "execute-check",
		trace.WithAttributes(
			attribute.String("check_id", check.ID),
			attribute.String("check_type", string(check.Type)),
			attribute.String("region", check.Region),
		),
	)
	defer span.End()

	executor, exists := sm.executors[check.Type]
	if !exists {
		span.RecordError(fmt.Errorf("no executor found for check type: %s", check.Type))
		return
	}

	var result *SyntheticResult
	var err error

	// Execute check with retries
	for attempt := 0; attempt <= check.RetryCount; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(check.RetryDelay):
			}
		}

		// Create timeout context for this attempt
		attemptCtx, cancel := context.WithTimeout(ctx, check.Timeout)
		result, err = executor.Execute(attemptCtx, check)
		cancel()

		if err == nil && result.Status == CheckStatusPass {
			break
		}

		span.AddEvent(fmt.Sprintf("Attempt %d failed", attempt+1),
			trace.WithAttributes(
				attribute.String("error", fmt.Sprintf("%v", err)),
			),
		)
	}

	if result == nil {
		result = &SyntheticResult{
			CheckID:      check.ID,
			CheckName:    check.Name,
			Timestamp:    time.Now(),
			Status:       CheckStatusError,
			ResponseTime: 0,
			Error:        fmt.Sprintf("execution failed: %v", err),
			Region:       check.Region,
		}
	}

	// Store result
	sm.storeResult(result)

	// Check alerting thresholds
	if sm.config.AlertingEnabled && sm.alertManager != nil {
		sm.evaluateAlerts(ctx, check, result)
	}

	span.AddEvent("Check executed",
		trace.WithAttributes(
			attribute.String("status", string(result.Status)),
			attribute.Int64("response_time_ms", result.ResponseTime.Milliseconds()),
		),
	)
}

// storeResult stores a check result
func (sm *SyntheticMonitor) storeResult(result *SyntheticResult) {
	sm.resultsMutex.Lock()
	defer sm.resultsMutex.Unlock()

	sm.results = append(sm.results, *result)
}

// evaluateAlerts evaluates alerting thresholds for a check
func (sm *SyntheticMonitor) evaluateAlerts(ctx context.Context, check *SyntheticCheck, result *SyntheticResult) {
	// Get recent results for this check to evaluate trends
	recentResults := sm.getRecentResults(check.ID, time.Hour) // Look at last hour

	shouldAlert, err := sm.alertManager.EvaluateThresholds(ctx, check, recentResults)
	if err != nil {
		// Log error but don't fail
		return
	}

	if shouldAlert {
		if err := sm.alertManager.SendAlert(ctx, check, result); err != nil {
			// Log error but don't fail
		}
	}
}

// getRecentResults gets recent results for a specific check
func (sm *SyntheticMonitor) getRecentResults(checkID string, duration time.Duration) []SyntheticResult {
	sm.resultsMutex.RLock()
	defer sm.resultsMutex.RUnlock()

	cutoff := time.Now().Add(-duration)
	results := make([]SyntheticResult, 0)

	for _, result := range sm.results {
		if result.CheckID == checkID && result.Timestamp.After(cutoff) {
			results = append(results, result)
		}
	}

	return results
}

// runResultCleanup cleans up old results
func (sm *SyntheticMonitor) runResultCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.cleanupResults()
		}
	}
}

// cleanupResults removes old results based on retention policy
func (sm *SyntheticMonitor) cleanupResults() {
	sm.resultsMutex.Lock()
	defer sm.resultsMutex.Unlock()

	cutoff := time.Now().Add(-sm.config.ResultRetention)
	validResults := make([]SyntheticResult, 0, len(sm.results))

	for _, result := range sm.results {
		if result.Timestamp.After(cutoff) {
			validResults = append(validResults, result)
		}
	}

	sm.results = validResults
}

// runChaosTests runs periodic chaos testing
func (sm *SyntheticMonitor) runChaosTests(ctx context.Context) {
	ticker := time.NewTicker(sm.config.ChaosTestInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sm.executeChaosTest(ctx)
		}
	}
}

// executeChaosTest executes a random chaos test
func (sm *SyntheticMonitor) executeChaosTest(ctx context.Context) {
	ctx, span := sm.tracer.Start(ctx, "execute-chaos-test")
	defer span.End()

	// Select a random critical service for chaos testing
	sm.checksMutex.RLock()
	var criticalChecks []*SyntheticCheck
	for _, check := range sm.checks {
		if check.BusinessImpact >= ImpactHigh {
			criticalChecks = append(criticalChecks, check)
		}
	}
	sm.checksMutex.RUnlock()

	if len(criticalChecks) == 0 {
		return
	}

	// Select a random critical check
	selectedCheck := criticalChecks[rand.Intn(len(criticalChecks))]

	// Create a chaos test version
	chaosCheck := &SyntheticCheck{
		ID:             fmt.Sprintf("chaos-%s-%d", selectedCheck.ID, time.Now().Unix()),
		Name:           fmt.Sprintf("Chaos Test - %s", selectedCheck.Name),
		Type:           CheckTypeChaos,
		Enabled:        true,
		Interval:       time.Hour, // One-time execution
		Timeout:        5 * time.Minute,
		RetryCount:     0, // No retries for chaos tests
		BusinessImpact: selectedCheck.BusinessImpact,
		Region:         selectedCheck.Region,
		Config: CheckConfig{
			ChaosType:      "network_latency", // Could be expanded to other types
			ChaosDuration:  2 * time.Minute,
			ChaosIntensity: 0.5, // 50% intensity
		},
	}

	// Execute the chaos test
	if executor, exists := sm.executors[CheckTypeChaos]; exists {
		result, err := executor.Execute(ctx, chaosCheck)
		if err != nil {
			span.RecordError(err)
			return
		}

		sm.storeResult(result)

		span.AddEvent("Chaos test completed",
			trace.WithAttributes(
				attribute.String("target_service", selectedCheck.Name),
				attribute.String("chaos_type", chaosCheck.Config.ChaosType),
				attribute.String("status", string(result.Status)),
			),
		)
	}
}

// GetResults returns synthetic check results within a time window
func (sm *SyntheticMonitor) GetResults(since time.Time, until time.Time) []SyntheticResult {
	sm.resultsMutex.RLock()
	defer sm.resultsMutex.RUnlock()

	results := make([]SyntheticResult, 0)
	for _, result := range sm.results {
		if result.Timestamp.After(since) && result.Timestamp.Before(until) {
			results = append(results, result)
		}
	}

	return results
}

// GetResultsByCheck returns results for a specific check
func (sm *SyntheticMonitor) GetResultsByCheck(checkID string, since time.Time, until time.Time) []SyntheticResult {
	sm.resultsMutex.RLock()
	defer sm.resultsMutex.RUnlock()

	results := make([]SyntheticResult, 0)
	for _, result := range sm.results {
		if result.CheckID == checkID &&
			result.Timestamp.After(since) &&
			result.Timestamp.Before(until) {
			results = append(results, result)
		}
	}

	return results
}

// GetAvailabilityMetrics converts synthetic results to availability metrics
func (sm *SyntheticMonitor) GetAvailabilityMetrics(checkID string, since time.Time, until time.Time) (*AvailabilityMetric, error) {
	results := sm.GetResultsByCheck(checkID, since, until)
	if len(results) == 0 {
		return nil, fmt.Errorf("no results found for check %s", checkID)
	}

	check, exists := sm.GetCheck(checkID)
	if !exists {
		return nil, fmt.Errorf("check %s not found", checkID)
	}

	// Calculate metrics
	var totalResponseTime time.Duration
	var successCount int
	var errorCount int

	latestResult := results[len(results)-1]

	for _, result := range results {
		totalResponseTime += result.ResponseTime
		if result.Status == CheckStatusPass {
			successCount++
		} else {
			errorCount++
		}
	}

	avgResponseTime := totalResponseTime / time.Duration(len(results))
	errorRate := float64(errorCount) / float64(len(results))

	// Determine health status
	var status HealthStatus
	if errorRate == 0 && avgResponseTime < check.AlertThresholds.ResponseTime {
		status = HealthHealthy
	} else if errorRate < check.AlertThresholds.ErrorRate {
		status = HealthDegraded
	} else {
		status = HealthUnhealthy
	}

	metric := &AvailabilityMetric{
		Timestamp:      latestResult.Timestamp,
		Dimension:      DimensionService, // Synthetic checks are service-level
		EntityID:       checkID,
		EntityType:     string(check.Type),
		Status:         status,
		ResponseTime:   avgResponseTime,
		ErrorRate:      errorRate,
		BusinessImpact: check.BusinessImpact,
		Layer:          LayerAPI, // Most synthetic checks are API-level
		Metadata: map[string]interface{}{
			"region":        check.Region,
			"total_checks":  len(results),
			"success_count": successCount,
			"error_count":   errorCount,
			"check_type":    string(check.Type),
		},
	}

	return metric, nil
}
