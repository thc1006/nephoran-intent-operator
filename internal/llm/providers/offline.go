package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// OfflineProvider implements the Provider interface with deterministic, rule-based processing.
// This provider does not require external API calls and processes intents using local rules.
// It serves as a fallback option and for environments without internet connectivity.
type OfflineProvider struct {
	config *Config
}

// NewOfflineProvider creates a new offline provider instance.
func NewOfflineProvider(config *Config) (Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil: %w", ErrInvalidConfiguration)
	}

	if config.Type != ProviderTypeOffline {
		return nil, fmt.Errorf("invalid provider type %s for offline provider: %w", config.Type, ErrInvalidConfiguration)
	}

	// Offline provider doesn't need API keys or external configuration
	provider := &OfflineProvider{
		config: config,
	}

	return provider, nil
}

// ProcessIntent converts natural language input into structured NetworkIntent JSON using rule-based processing.
func (p *OfflineProvider) ProcessIntent(ctx context.Context, input string) (*IntentResponse, error) {
	// Check context cancellation early
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled: %w", ctx.Err())
	default:
	}

	// Validate input
	if input == "" {
		return nil, fmt.Errorf("input cannot be empty: %w", ErrInvalidInput)
	}

	startTime := time.Now()

	// Check context cancellation during processing
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled during processing: %w", ctx.Err())
	default:
	}

	// Process the input using rule-based logic
	intent, err := p.parseInputWithRules(input)
	if err != nil {
		return nil, fmt.Errorf("failed to parse input: %w", err)
	}

	// Final context check before returning
	select {
	case <-ctx.Done():
		return nil, fmt.Errorf("context cancelled before completion: %w", ctx.Err())
	default:
	}

	intentJSON, err := json.Marshal(intent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal intent JSON: %w", err)
	}

	processingTime := time.Since(startTime)

	return &IntentResponse{
		JSON: intentJSON,
		Metadata: ResponseMetadata{
			Provider:       "OFFLINE",
			Model:          "rule-based-v1",
			ProcessingTime: processingTime,
			TokensUsed:     0, // No tokens used in offline processing
			Confidence:     p.calculateConfidence(input, intent),
			Warnings:       p.generateWarnings(input, intent),
		},
	}, nil
}

// parseInputWithRules implements rule-based parsing of natural language input.
func (p *OfflineProvider) parseInputWithRules(input string) (map[string]interface{}, error) {
	lower := strings.ToLower(input)
	words := strings.Fields(lower)

	intent := map[string]interface{}{
		"intent_type": "scaling",
		"target":      "unknown-target",
		"namespace":   "default",
		"replicas":    1,
		"reason":      fmt.Sprintf("Rule-based processing of: %s", input),
		"source":      "test",
		"priority":    5,
		"status":      "pending",
		"created_at":  time.Now().Format(time.RFC3339),
		"updated_at":  time.Now().Format(time.RFC3339),
	}

	// Rule 1: Detect intent type
	intentType := p.detectIntentType(words)
	intent["intent_type"] = intentType

	// Rule 2: Extract target component
	target := p.extractTarget(words)
	if target != "" {
		intent["target"] = target
	}

	// Rule 3: Extract namespace
	namespace := p.extractNamespace(input, words)
	if namespace != "" {
		intent["namespace"] = namespace
	}

	// Rule 4: Extract replica count
	replicas := p.extractReplicas(words)
	if replicas > 0 {
		intent["replicas"] = replicas
	}

	// Rule 5: Set priority based on urgency keywords
	priority := p.extractPriority(words)
	intent["priority"] = priority

	// Rule 6: Generate reason with extracted information
	intent["reason"] = p.generateReason(input, intent)

	return intent, nil
}

// detectIntentType uses keyword matching to determine the intent type.
func (p *OfflineProvider) detectIntentType(words []string) string {
	scalingKeywords := []string{"scale", "scaling", "replicas", "instances", "resize", "expand", "shrink", "increase", "decrease"}
	deploymentKeywords := []string{"deploy", "deployment", "install", "create", "start", "launch"}
	configurationKeywords := []string{"configure", "config", "settings", "update", "modify", "change"}

	for _, word := range words {
		for _, keyword := range scalingKeywords {
			if strings.Contains(word, keyword) {
				return "scaling"
			}
		}
		for _, keyword := range deploymentKeywords {
			if strings.Contains(word, keyword) {
				return "deployment"
			}
		}
		for _, keyword := range configurationKeywords {
			if strings.Contains(word, keyword) {
				return "configuration"
			}
		}
	}

	return "scaling" // Default to scaling
}

// extractTarget attempts to identify network function components from the input.
func (p *OfflineProvider) extractTarget(words []string) string {
	// Common O-RAN and 5G network function patterns
	patterns := map[string][]string{
		"o-du":         {"du", "o-du", "odu", "distributed-unit"},
		"o-cu-cp":      {"cu-cp", "o-cu-cp", "cp", "control-plane"},
		"o-cu-up":      {"cu-up", "o-cu-up", "up", "user-plane"},
		"o-ru":         {"ru", "o-ru", "oru", "radio-unit"},
		"amf":          {"amf", "access-mobility"},
		"smf":          {"smf", "session-management"},
		"upf":          {"upf", "user-plane-function"},
		"nrf":          {"nrf", "network-repository"},
		"pcf":          {"pcf", "policy-control"},
		"udm":          {"udm", "unified-data-management"},
		"ausf":         {"ausf", "authentication-server"},
		"nssf":         {"nssf", "network-slice-selection"},
		"ric":          {"ric", "ran-intelligence", "xapp"},
		"e2-simulator": {"e2", "e2sim", "simulator"},
	}

	for target, keywords := range patterns {
		for _, word := range words {
			for _, keyword := range keywords {
				if strings.Contains(word, keyword) {
					return target
				}
			}
		}
	}

	// Try to find any component-like words
	for _, word := range words {
		if len(word) > 3 && (strings.Contains(word, "component") ||
			strings.Contains(word, "service") ||
			strings.Contains(word, "function") ||
			strings.Contains(word, "app")) {
			return word
		}
	}

	return "default-workload"
}

// extractNamespace looks for namespace indicators in the input.
func (p *OfflineProvider) extractNamespace(input string, words []string) string {
	// Look for explicit namespace mentions - check for "namespace" first
	inputLower := strings.ToLower(input)
	
	// Pattern 1: Look for "... in <namespace> namespace"
	if idx := strings.Index(inputLower, " namespace"); idx != -1 {
		beforeNamespace := inputLower[:idx]
		wordsBeforeNamespace := strings.Fields(beforeNamespace)
		if len(wordsBeforeNamespace) > 0 {
			candidate := wordsBeforeNamespace[len(wordsBeforeNamespace)-1]
			candidate = strings.Trim(candidate, ".,!?;:")
			if len(candidate) > 0 && isValidNamespaceName(candidate) {
				return candidate
			}
		}
	}
	
	// Pattern 2: Look for "... in <namespace>" without "namespace" keyword
	namespacePatterns := []string{
		" in ", " from ", " within ",
	}

	for _, pattern := range namespacePatterns {
		if idx := strings.Index(inputLower, pattern); idx != -1 {
			// Try to extract the word following the namespace indicator
			remaining := inputLower[idx+len(pattern):]
			words := strings.Fields(strings.TrimSpace(remaining))
			if len(words) > 0 {
				candidate := words[0]
				// Clean up the candidate (remove punctuation)
				candidate = strings.Trim(candidate, ".,!?;:")
				// Check if it's not another keyword and is valid
				if len(candidate) > 0 && candidate != "namespace" && isValidNamespaceName(candidate) {
					return candidate
				}
			}
		}
	}

	// Common O-RAN namespace patterns
	commonNamespaces := []string{
		"oran-odu", "oran-cu", "oran-ru", "oran-ric",
		"core5g", "ran", "sim", "test", "prod", "dev",
	}

	for _, ns := range commonNamespaces {
		if strings.Contains(inputLower, ns) {
			return ns
		}
	}

	return "default"
}

// extractReplicas attempts to find numeric values that represent replica counts.
func (p *OfflineProvider) extractReplicas(words []string) int {
	numberWords := map[string]int{
		"zero": 0, "one": 1, "two": 2, "three": 3, "four": 4, "five": 5,
		"six": 6, "seven": 7, "eight": 8, "nine": 9, "ten": 10,
	}

	for i, word := range words {
		// Check for numeric words
		if replicas, exists := numberWords[word]; exists {
			return replicas
		}

		// Check for digits
		if len(word) <= 3 { // Reasonable replica count
			var num int
			if _, err := fmt.Sscanf(word, "%d", &num); err == nil && num >= 0 && num <= 100 {
				return num
			}
		}

		// Look for patterns like "to 5" or "up to 3"
		if (word == "to" || word == "up") && i+1 < len(words) {
			var num int
			if _, err := fmt.Sscanf(words[i+1], "%d", &num); err == nil && num > 0 && num <= 100 {
				return num
			}
		}
	}

	return 1 // Default replica count
}

// extractPriority determines priority based on urgency keywords.
func (p *OfflineProvider) extractPriority(words []string) int {
	highPriorityKeywords := []string{"urgent", "critical", "emergency", "asap", "immediately", "high"}
	lowPriorityKeywords := []string{"low", "later", "eventual", "whenever", "background"}

	for _, word := range words {
		for _, keyword := range highPriorityKeywords {
			if strings.Contains(word, keyword) {
				return 9 // High priority
			}
		}
		for _, keyword := range lowPriorityKeywords {
			if strings.Contains(word, keyword) {
				return 2 // Low priority
			}
		}
	}

	return 5 // Default medium priority
}

// generateReason creates a human-readable reason based on extracted information.
func (p *OfflineProvider) generateReason(input string, intent map[string]interface{}) string {
	target := intent["target"].(string)
	replicas := intent["replicas"].(int)
	namespace := intent["namespace"].(string)

	return fmt.Sprintf("Rule-based intent: scale %s to %d replicas in %s namespace (from: %s)",
		target, replicas, namespace, strings.TrimSpace(input))
}

// calculateConfidence estimates confidence based on how many rules matched.
func (p *OfflineProvider) calculateConfidence(input string, intent map[string]interface{}) float64 {
	confidence := 0.5 // Base confidence

	// Increase confidence for explicit component names
	if target := intent["target"].(string); target != "default-workload" && target != "unknown-target" {
		confidence += 0.2
	}

	// Increase confidence for explicit replica counts
	if replicas := intent["replicas"].(int); replicas > 1 {
		confidence += 0.1
	}

	// Increase confidence for explicit namespace
	if namespace := intent["namespace"].(string); namespace != "default" {
		confidence += 0.1
	}

	// Increase confidence based on input length (more context)
	if len(input) > 20 {
		confidence += 0.1
	}

	if confidence > 1.0 {
		confidence = 1.0
	}

	return confidence
}

// generateWarnings creates warnings about the rule-based processing limitations.
func (p *OfflineProvider) generateWarnings(input string, intent map[string]interface{}) []string {
	var warnings []string

	if intent["target"].(string) == "default-workload" {
		warnings = append(warnings, "Could not identify specific network function component")
	}

	if intent["namespace"].(string) == "default" && strings.Contains(strings.ToLower(input), "namespace") {
		warnings = append(warnings, "Namespace mentioned but could not be extracted reliably")
	}

	if intent["replicas"].(int) == 1 && containsScalingKeywords(input) {
		warnings = append(warnings, "Scaling intent detected but replica count not clearly specified")
	}

	return warnings
}

// GetProviderInfo returns metadata about the offline provider.
func (p *OfflineProvider) GetProviderInfo() ProviderInfo {
	return ProviderInfo{
		Name:         "OFFLINE",
		Version:      "1.0.0",
		Description:  "Rule-based offline provider for NetworkIntent processing without external API dependencies",
		RequiresAuth: false,
		SupportedFeatures: []string{
			"intent_processing",
			"rule_based_parsing",
			"offline_operation",
			"deterministic_output",
			"network_function_recognition",
			"confidence_scoring",
		},
	}
}

// ValidateConfig validates the offline provider configuration.
func (p *OfflineProvider) ValidateConfig() error {
	if p.config == nil {
		return fmt.Errorf("provider config is nil: %w", ErrInvalidConfiguration)
	}

	// Offline provider has minimal configuration requirements
	if p.config.Timeout <= 0 {
		return fmt.Errorf("timeout must be positive: %w", ErrInvalidConfiguration)
	}

	// No API key or network connectivity required
	return nil
}

// Close cleans up resources used by the offline provider.
func (p *OfflineProvider) Close() error {
	// Offline provider has no resources to clean up
	return nil
}

// Helper functions

// isValidNamespaceName checks if a string is a valid Kubernetes namespace name.
func isValidNamespaceName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}

	// Basic check for valid characters (alphanumeric and hyphens)
	for _, char := range name {
		if !((char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '-') {
			return false
		}
	}

	return true
}

// containsScalingKeywords checks if the input contains scaling-related keywords.
func containsScalingKeywords(input string) bool {
	scalingKeywords := []string{"scale", "scaling", "replicas", "instances", "resize"}
	lower := strings.ToLower(input)

	for _, keyword := range scalingKeywords {
		if strings.Contains(lower, keyword) {
			return true
		}
	}

	return false
}