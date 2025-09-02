package throughput

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// ErrorClassifier provides comprehensive error tracking and classification.

type ErrorClassifier struct {
	mu sync.RWMutex

	// Error tracking.

	errorCounts map[ErrorCategory]int

	errorHistory []ErrorEvent

	categoryCounts map[ErrorCategory]map[ErrorType]int

	// Prometheus metrics.

	errorRateMetric *prometheus.CounterVec

	errorTypeMetric *prometheus.CounterVec

	slaImpactMetric *prometheus.GaugeVec
}

// ErrorCategory represents broad error categories.

type ErrorCategory string

const (

	// CategorySystem holds categorysystem value.

	CategorySystem ErrorCategory = "system"

	// CategoryUser holds categoryuser value.

	CategoryUser ErrorCategory = "user"

	// CategoryExternal holds categoryexternal value.

	CategoryExternal ErrorCategory = "external"

	// CategoryTransient holds categorytransient value.

	CategoryTransient ErrorCategory = "transient"
)

// ErrorType represents specific error types within categories.

type ErrorType string

const (

	// System Errors.

	ErrorTypeInternalServerError ErrorType = "internal_server_error"

	// ErrorTypeResourceExhaustion holds errortyperesourceexhaustion value.

	ErrorTypeResourceExhaustion ErrorType = "resource_exhaustion"

	// ErrorTypeConfigurationError holds errortypeconfigurationerror value.

	ErrorTypeConfigurationError ErrorType = "configuration_error"

	// User Errors.

	ErrorTypeInvalidIntent ErrorType = "invalid_intent"

	// ErrorTypeAuthorizationFailed holds errortypeauthorizationfailed value.

	ErrorTypeAuthorizationFailed ErrorType = "authorization_failed"

	// ErrorTypeValidationError holds errortypevalidationerror value.

	ErrorTypeValidationError ErrorType = "validation_error"

	// External Errors.

	ErrorTypeLLMServiceFailure ErrorType = "llm_service_failure"

	// ErrorTypeNetworkConnectivity holds errortypenetworkconnectivity value.

	ErrorTypeNetworkConnectivity ErrorType = "network_connectivity"

	// ErrorTypeExternalAPIError holds errortypeexternalapierror value.

	ErrorTypeExternalAPIError ErrorType = "external_api_error"

	// Transient Errors.

	ErrorTypeTimeout ErrorType = "timeout"

	// ErrorTypeRateLimitExceeded holds errortyperatelimitexceeded value.

	ErrorTypeRateLimitExceeded ErrorType = "rate_limit_exceeded"

	// ErrorTypeTemporaryFailure holds errortypetemporaryfailure value.

	ErrorTypeTemporaryFailure ErrorType = "temporary_failure"
)

// ErrorEvent represents a detailed error occurrence.

type ErrorEvent struct {
	Timestamp time.Time

	Category ErrorCategory

	Type ErrorType

	Message string

	BusinessImpact float64
}

// NewErrorClassifier creates a new error classifier.

func NewErrorClassifier() *ErrorClassifier {
	return &ErrorClassifier{
		errorCounts: make(map[ErrorCategory]int),

		errorHistory: make([]ErrorEvent, 0, 1000),

		categoryCounts: make(map[ErrorCategory]map[ErrorType]int),

		errorRateMetric: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_error_rate",

			Help: "Total number of errors by category",
		}, []string{"category", "type"}),

		errorTypeMetric: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_error_type_count",

			Help: "Detailed error type counts",
		}, []string{"category", "type"}),

		slaImpactMetric: promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "nephoran_sla_impact",

			Help: "Business impact of errors",
		}, []string{"category", "type"}),
	}
}

// RecordError logs and classifies an error.

func (ec *ErrorClassifier) RecordError(
	category ErrorCategory,

	errorType ErrorType,

	message string,
) ErrorEvent {
	ec.mu.Lock()

	defer ec.mu.Unlock()

	// Calculate business impact.

	businessImpact := ec.calculateBusinessImpact(category, errorType)

	// Create error event.

	errorEvent := ErrorEvent{
		Timestamp: time.Now(),

		Category: category,

		Type: errorType,

		Message: message,

		BusinessImpact: businessImpact,
	}

	// Update counts.

	ec.errorCounts[category]++

	if ec.categoryCounts[category] == nil {
		ec.categoryCounts[category] = make(map[ErrorType]int)
	}

	ec.categoryCounts[category][errorType]++

	// Add to error history.

	ec.errorHistory = append(ec.errorHistory, errorEvent)

	if len(ec.errorHistory) > 1000 {
		ec.errorHistory = ec.errorHistory[1:]
	}

	// Update Prometheus metrics.

	ec.errorRateMetric.WithLabelValues(string(category), string(errorType)).Inc()

	ec.errorTypeMetric.WithLabelValues(string(category), string(errorType)).Inc()

	ec.slaImpactMetric.WithLabelValues(string(category), string(errorType)).Set(businessImpact)

	return errorEvent
}

// calculateBusinessImpact determines the severity of an error.

func (ec *ErrorClassifier) calculateBusinessImpact(
	category ErrorCategory,

	errorType ErrorType,
) float64 {
	// Impact scoring matrix.

	impactScores := map[ErrorCategory]map[ErrorType]float64{
		CategorySystem: {
			ErrorTypeInternalServerError: 0.9,

			ErrorTypeResourceExhaustion: 0.8,

			ErrorTypeConfigurationError: 0.7,
		},

		CategoryUser: {
			ErrorTypeInvalidIntent: 0.3,

			ErrorTypeAuthorizationFailed: 0.6,

			ErrorTypeValidationError: 0.4,
		},

		CategoryExternal: {
			ErrorTypeLLMServiceFailure: 0.7,

			ErrorTypeNetworkConnectivity: 0.8,

			ErrorTypeExternalAPIError: 0.6,
		},

		CategoryTransient: {
			ErrorTypeTimeout: 0.5,

			ErrorTypeRateLimitExceeded: 0.4,

			ErrorTypeTemporaryFailure: 0.3,
		},
	}

	// Default to low impact if not found.

	if categoryScores, exists := impactScores[category]; exists {
		if score, typeExists := categoryScores[errorType]; typeExists {
			return score
		}
	}

	return 0.1 // Minimal default impact
}

// GetErrorSummary provides an overview of error occurrences.

func (ec *ErrorClassifier) GetErrorSummary() map[ErrorCategory]int {
	ec.mu.RLock()

	defer ec.mu.RUnlock()

	return ec.errorCounts
}

// GetDetailedErrorBreakdown provides granular error type distribution.

func (ec *ErrorClassifier) GetDetailedErrorBreakdown() map[ErrorCategory]map[ErrorType]int {
	ec.mu.RLock()

	defer ec.mu.RUnlock()

	// Create a deep copy to prevent modification.

	breakdown := make(map[ErrorCategory]map[ErrorType]int)

	for category, typeMap := range ec.categoryCounts {

		breakdown[category] = make(map[ErrorType]int)

		for errorType, count := range typeMap {
			breakdown[category][errorType] = count
		}

	}

	return breakdown
}

// AnalyzeErrorTrends identifies emerging error patterns.

func (ec *ErrorClassifier) AnalyzeErrorTrends() map[string]interface{} {
	ec.mu.RLock()

	defer ec.mu.RUnlock()

	trends := make(map[string]interface{})

	// Calculate total error rate.

	totalErrors := 0

	for _, count := range ec.errorCounts {
		totalErrors += count
	}

	// Identify dominant error categories.

	dominantCategories := make([]string, 0)

	for category, count := range ec.errorCounts {
		if float64(count)/float64(totalErrors) > 0.2 {
			dominantCategories = append(dominantCategories, string(category))
		}
	}

	// Analyze business impact.

	totalBusinessImpact := 0.0

	for _, event := range ec.errorHistory {
		totalBusinessImpact += event.BusinessImpact
	}

	trends["total_errors"] = totalErrors

	trends["dominant_categories"] = dominantCategories

	trends["total_business_impact"] = totalBusinessImpact

	trends["avg_business_impact"] = totalBusinessImpact / float64(len(ec.errorHistory))

	return trends
}

// RetryStrategy provides intelligent retry recommendations.

type RetryStrategy struct {
	mu sync.Mutex

	// Retry tracking.

	retryAttempts map[ErrorType]int

	successfulRetries map[ErrorType]int

	// Retry metrics.

	retryMetric *prometheus.CounterVec
}

// NewRetryStrategy creates a new retry strategy.

func NewRetryStrategy() *RetryStrategy {
	return &RetryStrategy{
		retryAttempts: make(map[ErrorType]int),

		successfulRetries: make(map[ErrorType]int),

		retryMetric: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "nephoran_retry_attempts",

			Help: "Number of retry attempts by error type",
		}, []string{"error_type", "status"}),
	}
}

// ShouldRetry determines if an error should be retried.

func (rs *RetryStrategy) ShouldRetry(
	errorType ErrorType,

	currentAttempt int,
) bool {
	rs.mu.Lock()

	defer rs.mu.Unlock()

	// Retry configuration based on error type.

	retryConfig := map[ErrorType]int{
		ErrorTypeTimeout: 3,

		ErrorTypeRateLimitExceeded: 2,

		ErrorTypeTemporaryFailure: 2,

		ErrorTypeNetworkConnectivity: 3,

		ErrorTypeLLMServiceFailure: 2,
	}

	// Check if retry is configured for this error type.

	maxRetries, configured := retryConfig[errorType]

	if !configured {
		return false
	}

	// Track retry attempts.

	rs.retryAttempts[errorType]++

	rs.retryMetric.WithLabelValues(string(errorType), "attempt").Inc()

	return currentAttempt < maxRetries
}

// RecordRetryOutcome logs the result of a retry attempt.

func (rs *RetryStrategy) RecordRetryOutcome(
	errorType ErrorType,

	successful bool,
) {
	rs.mu.Lock()

	defer rs.mu.Unlock()

	if successful {

		rs.successfulRetries[errorType]++

		rs.retryMetric.WithLabelValues(string(errorType), "success").Inc()

	} else {
		rs.retryMetric.WithLabelValues(string(errorType), "failure").Inc()
	}
}

// GetRetryEffectiveness calculates retry success rates.

func (rs *RetryStrategy) GetRetryEffectiveness() map[ErrorType]float64 {
	rs.mu.Lock()

	defer rs.mu.Unlock()

	effectiveness := make(map[ErrorType]float64)

	for errorType, attempts := range rs.retryAttempts {

		successCount := rs.successfulRetries[errorType]

		effectiveness[errorType] = float64(successCount) / float64(attempts)

	}

	return effectiveness
}
