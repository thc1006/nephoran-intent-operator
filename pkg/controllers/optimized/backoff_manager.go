
package optimized



import (

	"context"

	"math"

	"math/rand"

	"sync"

	"time"

)



// BackoffStrategy defines different backoff strategies.

type BackoffStrategy string



const (

	// ExponentialBackoff holds exponentialbackoff value.

	ExponentialBackoff BackoffStrategy = "exponential"

	// LinearBackoff holds linearbackoff value.

	LinearBackoff BackoffStrategy = "linear"

	// ConstantBackoff holds constantbackoff value.

	ConstantBackoff BackoffStrategy = "constant"

)



// ErrorType categorizes errors for appropriate backoff strategies.

type ErrorType string



const (

	// TransientError holds transienterror value.

	TransientError ErrorType = "transient" // Network timeouts, temporary unavailability

	// PermanentError holds permanenterror value.

	PermanentError ErrorType = "permanent" // Invalid configuration, auth failures

	// ResourceError holds resourceerror value.

	ResourceError ErrorType = "resource" // Resource conflicts, quota exceeded

	// ThrottlingError holds throttlingerror value.

	ThrottlingError ErrorType = "throttling" // API rate limiting

	// ValidationError holds validationerror value.

	ValidationError ErrorType = "validation" // Schema validation failures

)



// BackoffConfig holds configuration for backoff behavior.

type BackoffConfig struct {

	Strategy      BackoffStrategy

	BaseDelay     time.Duration

	MaxDelay      time.Duration

	Multiplier    float64

	JitterEnabled bool

	MaxRetries    int

}



// DefaultBackoffConfigs provides sensible defaults for different error types.

var DefaultBackoffConfigs = map[ErrorType]BackoffConfig{

	TransientError: {

		Strategy:      ExponentialBackoff,

		BaseDelay:     2 * time.Second,

		MaxDelay:      5 * time.Minute,

		Multiplier:    2.0,

		JitterEnabled: true,

		MaxRetries:    5,

	},

	PermanentError: {

		Strategy:      ConstantBackoff,

		BaseDelay:     30 * time.Second,

		MaxDelay:      30 * time.Second,

		Multiplier:    1.0,

		JitterEnabled: false,

		MaxRetries:    2,

	},

	ResourceError: {

		Strategy:      LinearBackoff,

		BaseDelay:     5 * time.Second,

		MaxDelay:      2 * time.Minute,

		Multiplier:    1.5,

		JitterEnabled: true,

		MaxRetries:    4,

	},

	ThrottlingError: {

		Strategy:      ExponentialBackoff,

		BaseDelay:     10 * time.Second,

		MaxDelay:      10 * time.Minute,

		Multiplier:    2.5,

		JitterEnabled: true,

		MaxRetries:    3,

	},

	ValidationError: {

		Strategy:      ConstantBackoff,

		BaseDelay:     15 * time.Second,

		MaxDelay:      15 * time.Second,

		Multiplier:    1.0,

		JitterEnabled: false,

		MaxRetries:    1,

	},

}



// BackoffEntry tracks backoff state for a specific resource.

type BackoffEntry struct {

	RetryCount       int

	LastAttempt      time.Time

	CurrentDelay     time.Duration

	ErrorType        ErrorType

	ConsecutiveFails int

}



// BackoffManager manages backoff state for multiple resources.

type BackoffManager struct {

	mu      sync.RWMutex

	entries map[string]*BackoffEntry

	configs map[ErrorType]BackoffConfig

	rand    *rand.Rand

}



// NewBackoffManager creates a new BackoffManager with default configurations.

func NewBackoffManager() *BackoffManager {

	return &BackoffManager{

		entries: make(map[string]*BackoffEntry),

		configs: DefaultBackoffConfigs,

		rand:    rand.New(rand.NewSource(time.Now().UnixNano())),

	}

}



// SetConfig updates the backoff configuration for a specific error type.

func (bm *BackoffManager) SetConfig(errorType ErrorType, config BackoffConfig) {

	bm.mu.Lock()

	defer bm.mu.Unlock()

	bm.configs[errorType] = config

}



// GetNextDelay calculates the next delay duration for a resource and error type.

func (bm *BackoffManager) GetNextDelay(resourceKey string, errorType ErrorType, err error) time.Duration {

	bm.mu.Lock()

	defer bm.mu.Unlock()



	config := bm.configs[errorType]

	entry := bm.getOrCreateEntry(resourceKey)



	// Update entry state.

	entry.RetryCount++

	entry.LastAttempt = time.Now()

	entry.ErrorType = errorType



	// Check if we've exceeded max retries.

	if entry.RetryCount > config.MaxRetries {

		// Reset for fresh attempt after cooldown.

		entry.RetryCount = 1

		entry.ConsecutiveFails++

	}



	// Calculate base delay based on strategy.

	var delay time.Duration

	switch config.Strategy {

	case ExponentialBackoff:

		delay = time.Duration(float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(entry.RetryCount-1)))

	case LinearBackoff:

		delay = time.Duration(float64(config.BaseDelay) * (1 + config.Multiplier*float64(entry.RetryCount-1)))

	case ConstantBackoff:

		delay = config.BaseDelay

	default:

		delay = config.BaseDelay

	}



	// Apply maximum delay cap.

	if delay > config.MaxDelay {

		delay = config.MaxDelay

	}



	// Apply consecutive failure penalty.

	if entry.ConsecutiveFails > 0 {

		penalty := time.Duration(float64(delay) * (1 + 0.5*float64(entry.ConsecutiveFails)))

		if penalty <= config.MaxDelay {

			delay = penalty

		}

	}



	// Apply jitter if enabled.

	if config.JitterEnabled {

		jitter := time.Duration(bm.rand.Float64() * float64(delay) * 0.1) // 10% jitter

		delay = delay + jitter

	}



	entry.CurrentDelay = delay

	return delay

}



// ShouldRetry determines if a retry should be attempted based on current state.

func (bm *BackoffManager) ShouldRetry(resourceKey string, errorType ErrorType) bool {

	bm.mu.RLock()

	defer bm.mu.RUnlock()



	config := bm.configs[errorType]

	entry, exists := bm.entries[resourceKey]



	if !exists {

		return true // First attempt

	}



	return entry.RetryCount <= config.MaxRetries

}



// RecordSuccess resets the backoff state for a successful operation.

func (bm *BackoffManager) RecordSuccess(resourceKey string) {

	bm.mu.Lock()

	defer bm.mu.Unlock()



	if entry, exists := bm.entries[resourceKey]; exists {

		entry.RetryCount = 0

		entry.CurrentDelay = 0

		entry.ConsecutiveFails = 0

	}

}



// GetRetryCount returns the current retry count for a resource.

func (bm *BackoffManager) GetRetryCount(resourceKey string) int {

	bm.mu.RLock()

	defer bm.mu.RUnlock()



	if entry, exists := bm.entries[resourceKey]; exists {

		return entry.RetryCount

	}

	return 0

}



// CleanupStaleEntries removes entries older than the specified duration.

func (bm *BackoffManager) CleanupStaleEntries(ctx context.Context, maxAge time.Duration) {

	ticker := time.NewTicker(maxAge / 2) // Cleanup at half the max age

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			bm.mu.Lock()

			cutoff := time.Now().Add(-maxAge)

			for key, entry := range bm.entries {

				if entry.LastAttempt.Before(cutoff) {

					delete(bm.entries, key)

				}

			}

			bm.mu.Unlock()

		}

	}

}



// ClassifyError attempts to classify an error into an appropriate ErrorType.

func (bm *BackoffManager) ClassifyError(err error) ErrorType {

	if err == nil {

		return TransientError

	}



	errStr := err.Error()



	// Classification rules based on error content.

	switch {

	case containsAny(errStr, []string{"timeout", "connection refused", "network", "temporary"}):

		return TransientError

	case containsAny(errStr, []string{"unauthorized", "forbidden", "authentication", "permission"}):

		return PermanentError

	case containsAny(errStr, []string{"conflict", "quota", "resource", "capacity"}):

		return ResourceError

	case containsAny(errStr, []string{"rate limit", "throttle", "too many requests"}):

		return ThrottlingError

	case containsAny(errStr, []string{"validation", "invalid", "schema", "format"}):

		return ValidationError

	default:

		return TransientError // Default to transient for unknown errors

	}

}



// GetBackoffStats returns statistics for monitoring.

func (bm *BackoffManager) GetBackoffStats() BackoffStats {

	bm.mu.RLock()

	defer bm.mu.RUnlock()



	stats := BackoffStats{

		TotalEntries: len(bm.entries),

		ErrorTypes:   make(map[ErrorType]int),

		RetryRanges:  make(map[string]int),

	}



	for _, entry := range bm.entries {

		stats.ErrorTypes[entry.ErrorType]++



		switch {

		case entry.RetryCount == 0:

			stats.RetryRanges["0"]++

		case entry.RetryCount <= 2:

			stats.RetryRanges["1-2"]++

		case entry.RetryCount <= 5:

			stats.RetryRanges["3-5"]++

		default:

			stats.RetryRanges["6+"]++

		}

	}



	return stats

}



// BackoffStats contains statistics about backoff manager state.

type BackoffStats struct {

	TotalEntries int               `json:"total_entries"`

	ErrorTypes   map[ErrorType]int `json:"error_types"`

	RetryRanges  map[string]int    `json:"retry_ranges"`

}



// getOrCreateEntry gets or creates a backoff entry for a resource.

func (bm *BackoffManager) getOrCreateEntry(resourceKey string) *BackoffEntry {

	if entry, exists := bm.entries[resourceKey]; exists {

		return entry

	}



	entry := &BackoffEntry{

		RetryCount:       0,

		LastAttempt:      time.Now(),

		CurrentDelay:     0,

		ConsecutiveFails: 0,

	}

	bm.entries[resourceKey] = entry

	return entry

}



// containsAny checks if a string contains any of the provided substrings.

func containsAny(s string, substrings []string) bool {

	for _, substr := range substrings {

		if len(substr) > 0 && len(s) >= len(substr) {

			for i := 0; i <= len(s)-len(substr); i++ {

				if s[i:i+len(substr)] == substr {

					return true

				}

			}

		}

	}

	return false

}

