package llm

import (
	"fmt"
	"strings"
	"time"
)

// generateRequestID is defined in types.go

// Helper functions for the LLM package

// IsValidIntent checks if an intent string is valid
func IsValidIntent(intent string) bool {
	if len(intent) < 10 {
		return false
	}
	if len(intent) > 2048 {
		return false
	}
	return true
}

// SanitizeIntent removes potentially harmful content from intent strings
func SanitizeIntent(intent string) string {
	// Remove null bytes and control characters
	sanitized := ""
	for _, char := range intent {
		if char >= 32 && char <= 126 || char == '\n' || char == '\r' || char == '\t' {
			sanitized += string(char)
		}
	}
	return sanitized
}

// ExtractKeywords extracts important keywords from an intent
func ExtractKeywords(intent string) []string {
	// This is a simplified keyword extraction
	// In a real implementation, you might use NLP libraries
	keywords := []string{}
	
	// Common network function keywords
	nfKeywords := []string{
		"UPF", "AMF", "SMF", "PCF", "UDM", "AUSF", "NRF", "NSSF",
		"Near-RT-RIC", "O-DU", "O-CU", "O-RU",
		"deploy", "scale", "update", "delete", "configure",
		"replicas", "resources", "cpu", "memory",
	}
	
	intentLower := strings.ToLower(intent)
	for _, keyword := range nfKeywords {
		if strings.Contains(intentLower, strings.ToLower(keyword)) {
			keywords = append(keywords, keyword)
		}
	}
	
	return keywords
}

// ValidateNetworkFunction checks if a network function name is valid
func ValidateNetworkFunction(nf string) bool {
	validNFs := map[string]bool{
		"UPF":        true,
		"AMF":        true,
		"SMF":        true,
		"PCF":        true,
		"UDM":        true,
		"AUSF":       true,
		"NRF":        true,
		"NSSF":       true,
		"Near-RT-RIC": true,
		"O-DU":       true,
		"O-CU":       true,
		"O-RU":       true,
	}
	
	return validNFs[nf]
}

// EstimateProcessingTime estimates how long processing will take
func EstimateProcessingTime(intent string, enableEnrichment bool) time.Duration {
	baseTime := 100 * time.Millisecond
	
	// Add time based on intent complexity
	if len(intent) > 500 {
		baseTime += 50 * time.Millisecond
	}
	
	// Add time for enrichment
	if enableEnrichment {
		baseTime += 200 * time.Millisecond
	}
	
	return baseTime
}

// FormatValidationErrors formats validation errors for display
func FormatValidationErrors(errors []PipelineValidationError) string {
	if len(errors) == 0 {
		return ""
	}
	
	result := "Validation errors:\n"
	for i, err := range errors {
		result += fmt.Sprintf("%d. [%s] %s: %s\n", 
			i+1, err.Severity, err.Field, err.Message)
	}
	
	return result
}

// CalculateConfidenceScore calculates an overall confidence score
func CalculateConfidenceScore(classification ClassificationResult, validation PipelineValidationResult) float64 {
	if !validation.Valid {
		return 0.0
	}
	
	// Combine classification confidence with validation score
	return (classification.Confidence + validation.Score) / 2.0
}

// isValidKubernetesName checks if a name is valid for Kubernetes resources
func isValidKubernetesName(name string) bool {
	if name == "" || len(name) > 63 {
		return false
	}

	// Must start and end with alphanumeric
	if !isAlphaNumeric(name[0]) || !isAlphaNumeric(name[len(name)-1]) {
		return false
	}

	// Check each character
	for _, char := range name {
		if !isAlphaNumeric(byte(char)) && char != '-' {
			return false
		}
	}

	return true
}

// isAlphaNumeric checks if a character is alphanumeric
func isAlphaNumeric(char byte) bool {
	return (char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9')
}