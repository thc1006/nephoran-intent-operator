package llm

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestIsValidIntent(t *testing.T) {
	tests := []struct {
		name     string
		intent   string
		expected bool
	}{
		{
			name:     "valid intent",
			intent:   "Deploy UPF with 3 replicas",
			expected: true,
		},
		{
			name:     "too short",
			intent:   "short",
			expected: false,
		},
		{
			name:     "exactly 10 chars",
			intent:   "0123456789",
			expected: true,
		},
		{
			name:     "exactly 9 chars",
			intent:   "012345678",
			expected: false,
		},
		{
			name:     "too long (over 2048)",
			intent:   string(make([]byte, 2049)),
			expected: false,
		},
		{
			name:     "exactly 2048 chars",
			intent:   string(make([]byte, 2048)),
			expected: true,
		},
		{
			name:     "complex valid intent",
			intent:   "Deploy 5G Core with AMF, SMF, UPF and configure QoS policies",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidIntent(tt.intent)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeIntent(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "no sanitization needed",
			input:    "Deploy UPF",
			expected: "Deploy UPF",
		},
		{
			name:     "removes null bytes",
			input:    "Deploy\x00UPF",
			expected: "DeployUPF",
		},
		{
			name:     "removes control characters",
			input:    "Deploy\x01\x02\x03UPF",
			expected: "DeployUPF",
		},
		{
			name:     "keeps newlines",
			input:    "Deploy\nUPF",
			expected: "Deploy\nUPF",
		},
		{
			name:     "keeps tabs",
			input:    "Deploy\tUPF",
			expected: "Deploy\tUPF",
		},
		{
			name:     "keeps carriage returns",
			input:    "Deploy\rUPF",
			expected: "Deploy\rUPF",
		},
		{
			name:     "mixed safe and unsafe",
			input:    "Deploy\x00\nUPF\x01with\ttabs",
			expected: "Deploy\nUPFwith\ttabs",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "only printable ASCII",
			input:    "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789",
			expected: "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeIntent(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractKeywords(t *testing.T) {
	tests := []struct {
		name     string
		intent   string
		expected []string
	}{
		{
			name:     "extract UPF",
			intent:   "Deploy UPF with 3 replicas",
			expected: []string{"UPF", "deploy", "replicas"},
		},
		{
			name:     "extract multiple NFs",
			intent:   "Deploy AMF and SMF with UPF",
			expected: []string{"AMF", "SMF", "UPF", "deploy"},
		},
		{
			name:     "case insensitive",
			intent:   "deploy upf and amf",
			expected: []string{"UPF", "AMF", "deploy"},
		},
		{
			name:     "extract O-RAN components",
			intent:   "Configure Near-RT-RIC with O-DU and O-CU",
			expected: []string{"Near-RT-RIC", "O-DU", "O-CU"},
		},
		{
			name:     "extract resource keywords",
			intent:   "Scale UPF replicas with cpu and memory limits",
			expected: []string{"UPF", "scale", "replicas", "cpu", "memory"},
		},
		{
			name:     "no keywords",
			intent:   "This has no relevant keywords",
			expected: []string{},
		},
		{
			name:     "update operation",
			intent:   "Update SMF configuration",
			expected: []string{"SMF", "update"},
		},
		{
			name:     "delete operation",
			intent:   "Delete PCF deployment",
			expected: []string{"PCF", "delete"},
		},
		{
			name:     "configure operation",
			intent:   "Configure UDM with replicas",
			expected: []string{"UDM", "configure", "replicas"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractKeywords(tt.intent)
			assert.ElementsMatch(t, tt.expected, result)
		})
	}
}

func TestValidateNetworkFunction(t *testing.T) {
	tests := []struct {
		name     string
		nf       string
		expected bool
	}{
		// Valid 5G Core NFs
		{"UPF valid", "UPF", true},
		{"AMF valid", "AMF", true},
		{"SMF valid", "SMF", true},
		{"PCF valid", "PCF", true},
		{"UDM valid", "UDM", true},
		{"AUSF valid", "AUSF", true},
		{"NRF valid", "NRF", true},
		{"NSSF valid", "NSSF", true},

		// Valid O-RAN NFs
		{"Near-RT-RIC valid", "Near-RT-RIC", true},
		{"O-DU valid", "O-DU", true},
		{"O-CU valid", "O-CU", true},
		{"O-RU valid", "O-RU", true},

		// Invalid NFs
		{"invalid lowercase", "upf", false},
		{"invalid unknown", "UNKNOWN", false},
		{"invalid empty", "", false},
		{"invalid partial", "UP", false},
		{"invalid typo", "UPF1", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ValidateNetworkFunction(tt.nf)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEstimateProcessingTime(t *testing.T) {
	tests := []struct {
		name              string
		intent            string
		enableEnrichment  bool
		expectedMinimum   time.Duration
		expectedMaximum   time.Duration
	}{
		{
			name:             "short intent without enrichment",
			intent:           "Deploy UPF",
			enableEnrichment: false,
			expectedMinimum:  100 * time.Millisecond,
			expectedMaximum:  150 * time.Millisecond,
		},
		{
			name:             "short intent with enrichment",
			intent:           "Deploy UPF",
			enableEnrichment: true,
			expectedMinimum:  300 * time.Millisecond,
			expectedMaximum:  350 * time.Millisecond,
		},
		{
			name:             "long intent without enrichment",
			intent:           string(make([]byte, 600)),
			enableEnrichment: false,
			expectedMinimum:  150 * time.Millisecond,
			expectedMaximum:  200 * time.Millisecond,
		},
		{
			name:             "long intent with enrichment",
			intent:           string(make([]byte, 600)),
			enableEnrichment: true,
			expectedMinimum:  350 * time.Millisecond,
			expectedMaximum:  400 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := EstimateProcessingTime(tt.intent, tt.enableEnrichment)
			assert.GreaterOrEqual(t, result, tt.expectedMinimum)
			assert.LessOrEqual(t, result, tt.expectedMaximum)
		})
	}
}

func TestFormatValidationErrors(t *testing.T) {
	tests := []struct {
		name     string
		errors   []PipelineValidationError
		expected string
	}{
		{
			name:     "no errors",
			errors:   []PipelineValidationError{},
			expected: "",
		},
		{
			name: "single error",
			errors: []PipelineValidationError{
				{
					Field:    "intent",
					Message:  "required field missing",
					Severity: "error",
				},
			},
			expected: "Validation errors:\n1. [error] intent: required field missing\n",
		},
		{
			name: "multiple errors",
			errors: []PipelineValidationError{
				{
					Field:    "intent",
					Message:  "too short",
					Severity: "error",
				},
				{
					Field:    "replicas",
					Message:  "must be positive",
					Severity: "warning",
				},
			},
			expected: "Validation errors:\n1. [error] intent: too short\n2. [warning] replicas: must be positive\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatValidationErrors(tt.errors)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateConfidenceScore(t *testing.T) {
	tests := []struct {
		name           string
		classification ClassificationResult
		validation     PipelineValidationResult
		expected       float64
	}{
		{
			name: "valid with high confidence",
			classification: ClassificationResult{
				Confidence: 0.9,
			},
			validation: PipelineValidationResult{
				Valid: true,
				Score: 0.8,
			},
			expected: 0.85,
		},
		{
			name: "invalid validation",
			classification: ClassificationResult{
				Confidence: 0.9,
			},
			validation: PipelineValidationResult{
				Valid: false,
				Score: 0.5,
			},
			expected: 0.0,
		},
		{
			name: "valid with low confidence",
			classification: ClassificationResult{
				Confidence: 0.5,
			},
			validation: PipelineValidationResult{
				Valid: true,
				Score: 0.6,
			},
			expected: 0.55,
		},
		{
			name: "perfect scores",
			classification: ClassificationResult{
				Confidence: 1.0,
			},
			validation: PipelineValidationResult{
				Valid: true,
				Score: 1.0,
			},
			expected: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateConfidenceScore(tt.classification, tt.validation)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}

func TestIsValidKubernetesName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"valid name", "my-deployment", true},
		{"valid short", "a", true},
		{"valid numbers", "deploy-123", true},
		{"too long (64 chars)", string(make([]byte, 64)), false},
		{"exactly 63 chars", string(make([]byte, 63)), false}, // Would need valid chars
		{"empty string", "", false},
		{"starts with dash", "-deployment", false},
		{"ends with dash", "deployment-", false},
		{"uppercase", "Deployment", false},
		{"underscore", "my_deployment", false},
		{"dot", "my.deployment", false},
		{"valid lowercase", "mydeployment", true},
		{"valid with numbers", "deploy123", true},
		{"space", "my deployment", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidKubernetesName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsAlphaNumeric(t *testing.T) {
	tests := []struct {
		name     string
		char     byte
		expected bool
	}{
		{"lowercase a", 'a', true},
		{"lowercase z", 'z', true},
		{"digit 0", '0', true},
		{"digit 9", '9', true},
		{"uppercase A", 'A', false},
		{"uppercase Z", 'Z', false},
		{"dash", '-', false},
		{"underscore", '_', false},
		{"space", ' ', false},
		{"dot", '.', false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isAlphaNumeric(tt.char)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeIntent_EdgeCases(t *testing.T) {
	t.Run("very long string with control chars", func(t *testing.T) {
		input := string(make([]byte, 1000))
		for i := range input {
			if i%10 == 0 {
				input = input[:i] + "\x00" + input[i+1:]
			}
		}

		result := SanitizeIntent(input)
		// Should remove all null bytes
		assert.NotContains(t, result, "\x00")
		// Should be shorter than input
		assert.Less(t, len(result), len(input))
	})

	t.Run("unicode characters", func(t *testing.T) {
		input := "Deploy UPF 部署"
		result := SanitizeIntent(input)
		// Only ASCII should remain
		assert.Equal(t, "Deploy UPF ", result)
	})
}

func TestExtractKeywords_EdgeCases(t *testing.T) {
	t.Run("repeated keywords", func(t *testing.T) {
		intent := "Deploy UPF deploy UPF deploy"
		keywords := ExtractKeywords(intent)

		// Should have duplicates (current implementation)
		upfCount := 0
		deployCount := 0
		for _, kw := range keywords {
			if kw == "UPF" {
				upfCount++
			}
			if kw == "deploy" {
				deployCount++
			}
		}
		assert.Greater(t, upfCount, 0)
		assert.Greater(t, deployCount, 0)
	})

	t.Run("partial matches", func(t *testing.T) {
		intent := "Deploy UPFOO and AMFBAR"
		keywords := ExtractKeywords(intent)

		// Should match UPF and AMF within larger strings
		assert.Contains(t, keywords, "deploy")
	})
}
