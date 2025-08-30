package controllers

import (
	"fmt"
	"math"
	"math/rand"
	"strconv"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
	configPkg "github.com/nephio-project/nephoran-intent-operator/pkg/config"
)

// BackoffConfig holds configuration for exponential backoff
type BackoffConfig struct {
	BaseDelay    time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
	JitterFactor float64
}

// DefaultBackoffConfig returns the default backoff configuration
func DefaultBackoffConfig() *BackoffConfig {
	return &BackoffConfig{
		BaseDelay:    1 * time.Second,
		MaxDelay:     5 * time.Minute,
		Multiplier:   2.0,
		JitterFactor: 0.1,
	}
}

// CalculateExponentialBackoff calculates the exponential backoff delay with jitter
// retryCount: current retry attempt (0-based)
// config: backoff configuration parameters
func CalculateExponentialBackoff(retryCount int, config *BackoffConfig) time.Duration {
	if config == nil {
		config = DefaultBackoffConfig()
	}

	// Calculate exponential backoff: baseDelay * (multiplier^retryCount)
	backoffDelay := float64(config.BaseDelay) * math.Pow(config.Multiplier, float64(retryCount))

	// Cap at maximum delay
	if backoffDelay > float64(config.MaxDelay) {
		backoffDelay = float64(config.MaxDelay)
	}

	// Add jitter to prevent thundering herd
	jitterRange := backoffDelay * config.JitterFactor
	jitter := (rand.Float64() - 0.5) * 2 * jitterRange
	finalDelay := backoffDelay + jitter

	// Ensure minimum delay
	if finalDelay < float64(config.BaseDelay) {
		finalDelay = float64(config.BaseDelay)
	}

	return time.Duration(finalDelay)
}

// CalculateExponentialBackoffWithConstants calculates backoff using configPkg.Constants
// for backward compatibility with NetworkIntent controller
func CalculateExponentialBackoffWithConstants(retryCount int, baseDelay, maxDelay time.Duration, constants *configPkg.Constants) time.Duration {
	if baseDelay <= 0 {
		baseDelay = constants.BaseBackoffDelay
	}
	if maxDelay <= 0 {
		maxDelay = constants.MaxBackoffDelay
	}

	config := &BackoffConfig{
		BaseDelay:    baseDelay,
		MaxDelay:     maxDelay,
		Multiplier:   constants.BackoffMultiplier,
		JitterFactor: constants.JitterFactor,
	}

	return CalculateExponentialBackoff(retryCount, config)
}

// CalculateExponentialBackoffForNetworkIntentOperation calculates backoff for NetworkIntent operations
func CalculateExponentialBackoffForNetworkIntentOperation(retryCount int, operation string, constants *configPkg.Constants) time.Duration {
	var baseDelay, maxDelay time.Duration

	switch operation {
	case "llm-processing":
		baseDelay = constants.LLMProcessingBaseDelay
		maxDelay = constants.LLMProcessingMaxDelay
	case "git-operations":
		baseDelay = constants.GitOperationsBaseDelay
		maxDelay = constants.GitOperationsMaxDelay
	case "resource-planning":
		baseDelay = constants.ResourcePlanningBaseDelay
		maxDelay = constants.ResourcePlanningMaxDelay
	default:
		baseDelay = constants.BaseBackoffDelay
		maxDelay = constants.MaxBackoffDelay
	}

	return CalculateExponentialBackoffWithConstants(retryCount, baseDelay, maxDelay, constants)
}

// CalculateExponentialBackoffForE2NodeSetOperation calculates backoff for E2NodeSet operations
func CalculateExponentialBackoffForE2NodeSetOperation(retryCount int, operation string) time.Duration {
	var config *BackoffConfig

	switch operation {
	case "configmap-operations":
		// ConfigMap operations: moderate backoff for Kubernetes API
		config = &BackoffConfig{
			BaseDelay:    2 * time.Second,
			MaxDelay:     2 * time.Minute,
			Multiplier:   2.0,
			JitterFactor: 0.1,
		}
	case "e2-provisioning":
		// E2 node provisioning: longer backoff for complex operations
		config = &BackoffConfig{
			BaseDelay:    5 * time.Second,
			MaxDelay:     5 * time.Minute,
			Multiplier:   2.0,
			JitterFactor: 0.1,
		}
	case "cleanup":
		// Cleanup operations: existing configuration
		config = &BackoffConfig{
			BaseDelay:    10 * time.Second,
			MaxDelay:     5 * time.Minute,
			Multiplier:   2.0,
			JitterFactor: 0.1,
		}
	default:
		// Default configuration using E2NodeSet constants
		config = &BackoffConfig{
			BaseDelay:    1 * time.Second, // BaseBackoffDelay from e2nodeset_controller.go
			MaxDelay:     5 * time.Minute, // MaxBackoffDelay from e2nodeset_controller.go
			Multiplier:   2.0,             // BackoffMultiplier from e2nodeset_controller.go
			JitterFactor: 0.1,             // JitterFactor from e2nodeset_controller.go
		}
	}

	return CalculateExponentialBackoff(retryCount, config)
}

// GetNetworkIntentRetryCount retrieves the retry count for a NetworkIntent operation from annotations
func GetNetworkIntentRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string) int {
	if networkIntent.Annotations == nil {
		return 0
	}

	key := fmt.Sprintf("nephoran.com/retry-count-%s", operation)
	if countStr, exists := networkIntent.Annotations[key]; exists {
		if count, err := strconv.Atoi(countStr); err == nil {
			return count
		}
	}
	return 0
}

// SetNetworkIntentRetryCount sets the retry count for a NetworkIntent operation in annotations
func SetNetworkIntentRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string, count int) {
	if networkIntent.Annotations == nil {
		networkIntent.Annotations = make(map[string]string)
	}
	key := fmt.Sprintf("nephoran.com/retry-count-%s", operation)
	networkIntent.Annotations[key] = strconv.Itoa(count)
}

// ClearNetworkIntentRetryCount removes the retry count for a NetworkIntent operation from annotations
func ClearNetworkIntentRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string) {
	if networkIntent.Annotations == nil {
		return
	}
	key := fmt.Sprintf("nephoran.com/retry-count-%s", operation)
	delete(networkIntent.Annotations, key)
}

// GetE2NodeSetRetryCount retrieves the retry count for an E2NodeSet operation from annotations
func GetE2NodeSetRetryCount(e2nodeSet *nephoranv1.E2NodeSet, operation string) int {
	if e2nodeSet.Annotations == nil {
		return 0
	}

	key := fmt.Sprintf("nephoran.com/%s-retry-count", operation)
	if countStr, exists := e2nodeSet.Annotations[key]; exists {
		if count, err := fmt.Sscanf(countStr, "%d", new(int)); err == nil && count == 1 {
			var result int
			fmt.Sscanf(countStr, "%d", &result)
			return result
		}
	}
	return 0
}

// SetE2NodeSetRetryCount sets the retry count for an E2NodeSet operation in annotations
func SetE2NodeSetRetryCount(e2nodeSet *nephoranv1.E2NodeSet, operation string, count int) {
	if e2nodeSet.Annotations == nil {
		e2nodeSet.Annotations = make(map[string]string)
	}

	key := fmt.Sprintf("nephoran.com/%s-retry-count", operation)
	e2nodeSet.Annotations[key] = fmt.Sprintf("%d", count)
}

// ClearE2NodeSetRetryCount removes the retry count for an E2NodeSet operation from annotations
func ClearE2NodeSetRetryCount(e2nodeSet *nephoranv1.E2NodeSet, operation string) {
	if e2nodeSet.Annotations == nil {
		return
	}

	key := fmt.Sprintf("nephoran.com/%s-retry-count", operation)
	delete(e2nodeSet.Annotations, key)
}

// UpdateCondition updates or adds a condition to a condition slice
func UpdateCondition(conditions *[]metav1.Condition, newCondition metav1.Condition) {
	if conditions == nil {
		return
	}

	for i, condition := range *conditions {
		if condition.Type == newCondition.Type {
			// Update existing condition
			(*conditions)[i] = newCondition
			return
		}
	}
	// Add new condition
	*conditions = append(*conditions, newCondition)
}

// Helper functions for GitOps handler - these provide convenience wrappers
// around the existing NetworkIntent retry count functions

// getRetryCount retrieves the retry count for a NetworkIntent operation
func getRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string) int {
	return GetNetworkIntentRetryCount(networkIntent, operation)
}

// setRetryCount sets the retry count for a NetworkIntent operation
func setRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string, count int) {
	SetNetworkIntentRetryCount(networkIntent, operation, count)
}

// clearRetryCount removes the retry count for a NetworkIntent operation
func clearRetryCount(networkIntent *nephoranv1.NetworkIntent, operation string) {
	ClearNetworkIntentRetryCount(networkIntent, operation)
}

// updateCondition updates or adds a condition to a NetworkIntent's condition slice
func updateCondition(conditions *[]metav1.Condition, newCondition metav1.Condition) {
	UpdateCondition(conditions, newCondition)
}

// calculateExponentialBackoffForOperation calculates exponential backoff for operations
// Used by GitOps handler for consistent backoff behavior
func calculateExponentialBackoffForOperation(retryCount int, operation string) time.Duration {
	// Use default constants if not provided
	constants := &configPkg.Constants{
		BaseBackoffDelay:          1 * time.Second,
		MaxBackoffDelay:           5 * time.Minute,
		BackoffMultiplier:         2.0,
		JitterFactor:              0.1,
		LLMProcessingBaseDelay:    2 * time.Second,
		LLMProcessingMaxDelay:     3 * time.Minute,
		GitOperationsBaseDelay:    1 * time.Second,
		GitOperationsMaxDelay:     2 * time.Minute,
		ResourcePlanningBaseDelay: 3 * time.Second,
		ResourcePlanningMaxDelay:  4 * time.Minute,
	}

	return CalculateExponentialBackoffForNetworkIntentOperation(retryCount, operation, constants)
}

// isConditionTrue checks if a condition is present and true
func isConditionTrue(conditions []metav1.Condition, conditionType string) bool {
	for _, condition := range conditions {
		if condition.Type == conditionType {
			return condition.Status == metav1.ConditionTrue
		}
	}
	return false
}
