package webhooks

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// Test helper functions
func createTestValidator() (*NetworkIntentValidator, error) {
	scheme := runtime.NewScheme()
	err := nephoranv1.AddToScheme(scheme)
	if err != nil {
		return nil, err
	}

	decoder := admission.NewDecoder(scheme)

	validator := NewNetworkIntentValidator()
	err = validator.InjectDecoder(decoder)
	if err != nil {
		return nil, err
	}

	return validator, nil
}

func createAdmissionRequest(ni *nephoranv1.NetworkIntent, operation admissionv1.Operation) admission.Request {
	raw, _ := json.Marshal(ni)
	return admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			UID:       "test-uid",
			Operation: operation,
			Object: runtime.RawExtension{
				Raw: raw,
			},
			Namespace: ni.Namespace,
			Name:      ni.Name,
		},
	}
}

func createTestNetworkIntent(name, namespace, intent string) *nephoranv1.NetworkIntent {
	return &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: intent,
		},
	}
}

// Comprehensive unit tests for NetworkIntent webhook validation
func TestNetworkIntentValidator_Handle(t *testing.T) {
	validator, err := createTestValidator()
	require.NoError(t, err)

	tests := []struct {
		name            string
		networkIntent   *nephoranv1.NetworkIntent
		operation       admissionv1.Operation
		expectedAllowed bool
		expectedReason  string
		containsMessage string
	}{
		{
			name:            "valid telecommunications intent - AMF deployment",
			networkIntent:   createTestNetworkIntent("test-amf", "default", "Deploy AMF network function for 5G core"),
			operation:       admissionv1.Create,
			expectedAllowed: true,
			expectedReason:  "",
		},
		{
			name:            "valid telecommunications intent - SMF configuration",
			networkIntent:   createTestNetworkIntent("test-smf", "default", "Configure SMF with UPF integration for session management"),
			operation:       admissionv1.Create,
			expectedAllowed: true,
			expectedReason:  "",
		},
		{
			name:            "valid O-RAN intent",
			networkIntent:   createTestNetworkIntent("test-oran", "default", "Setup O-CU and O-DU components for RAN deployment"),
			operation:       admissionv1.Create,
			expectedAllowed: true,
			expectedReason:  "",
		},
		{
			name:            "valid network slice intent",
			networkIntent:   createTestNetworkIntent("test-slice", "default", "Deploy eMBB slice for high bandwidth applications"),
			operation:       admissionv1.Create,
			expectedAllowed: true,
			expectedReason:  "",
		},
		{
			name:            "empty intent",
			networkIntent:   createTestNetworkIntent("test-empty", "default", ""),
			operation:       admissionv1.Create,
			expectedAllowed: false,
			containsMessage: "empty",
		},
		{
			name:            "whitespace only intent",
			networkIntent:   createTestNetworkIntent("test-whitespace", "default", "   \t\n  "),
			operation:       admissionv1.Create,
			expectedAllowed: false,
			containsMessage: "empty",
		},
		{
			name:            "non-telecom intent",
			networkIntent:   createTestNetworkIntent("test-generic", "default", "Deploy a generic web application with database"),
			operation:       admissionv1.Create,
			expectedAllowed: false,
			containsMessage: "telecommunications",
		},
		{
			name:            "malicious script injection",
			networkIntent:   createTestNetworkIntent("test-script", "default", "Deploy AMF <script>alert('xss')</script>"),
			operation:       admissionv1.Create,
			expectedAllowed: false,
			containsMessage: "malicious",
		},
		{
			name:            "SQL injection attempt",
			networkIntent:   createTestNetworkIntent("test-sql", "default", "Deploy SMF'; DROP TABLE users; --"),
			operation:       admissionv1.Create,
			expectedAllowed: false,
			containsMessage: "malicious",
		},
		{
			name:            "shell command injection",
			networkIntent:   createTestNetworkIntent("test-shell", "default", "Deploy UPF && rm -rf /"),
			operation:       admissionv1.Create,
			expectedAllowed: false,
			containsMessage: "malicious",
		},
		{
			name:            "intent too long",
			networkIntent:   createTestNetworkIntent("test-long", "default", strings.Repeat("Deploy AMF with many configurations ", 100)),
			operation:       admissionv1.Create,
			expectedAllowed: false,
			containsMessage: "complex",
		},
		{
			name:            "intent with encoded characters",
			networkIntent:   createTestNetworkIntent("test-encoded", "default", "Deploy AMF \\x41\\x42\\x43"),
			operation:       admissionv1.Create,
			expectedAllowed: false,
			containsMessage: "encoded",
		},
		{
			name:            "delete operation",
			networkIntent:   createTestNetworkIntent("test-delete", "default", "Deploy AMF"),
			operation:       admissionv1.Delete,
			expectedAllowed: true,
			expectedReason:  "Delete operations are allowed",
		},
		{
			name:            "update operation with valid intent",
			networkIntent:   createTestNetworkIntent("test-update", "default", "Update SMF configuration for better performance"),
			operation:       admissionv1.Update,
			expectedAllowed: true,
		},
		{
			name:            "intent with minimal telecom keywords",
			networkIntent:   createTestNetworkIntent("test-minimal", "default", "Configure network slice"),
			operation:       admissionv1.Create,
			expectedAllowed: true,
		},
		{
			name:            "very short name",
			networkIntent:   createTestNetworkIntent("ab", "default", "Deploy AMF network function"),
			operation:       admissionv1.Create,
			expectedAllowed: false,
			containsMessage: "too short",
		},
		{
			name:            "very long name",
			networkIntent:   createTestNetworkIntent(strings.Repeat("a", 65), "default", "Deploy AMF network function"),
			operation:       admissionv1.Create,
			expectedAllowed: false,
			containsMessage: "too long",
		},
		{
			name:            "name with reserved prefix",
			networkIntent:   createTestNetworkIntent("system-intent", "default", "Deploy AMF network function"),
			operation:       admissionv1.Create,
			expectedAllowed: false,
			containsMessage: "reserved",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := createAdmissionRequest(tt.networkIntent, tt.operation)
			ctx := context.Background()

			response := validator.Handle(ctx, req)

			assert.Equal(t, tt.expectedAllowed, response.Allowed)

			if tt.expectedReason != "" {
				assert.Contains(t, response.Result.Message, tt.expectedReason)
			}

			if tt.containsMessage != "" {
				assert.Contains(t, strings.ToLower(response.Result.Message), strings.ToLower(tt.containsMessage))
			}
		})
	}
}

func TestValidateIntentContent(t *testing.T) {
	validator, err := createTestValidator()
	require.NoError(t, err)

	tests := []struct {
		name          string
		intent        string
		expectedError bool
		errorContains string
	}{
		{
			name:          "valid intent",
			intent:        "Deploy AMF network function",
			expectedError: false,
		},
		{
			name:          "empty intent",
			intent:        "",
			expectedError: true,
			errorContains: "empty",
		},
		{
			name:          "whitespace only",
			intent:        "   \t\n  ",
			expectedError: true,
			errorContains: "empty",
		},
		{
			name:          "intent with newlines",
			intent:        "Deploy AMF\nwith configuration\nfor 5G core",
			expectedError: false,
		},
		{
			name:          "intent with control characters",
			intent:        "Deploy AMF\u0001with control chars",
			expectedError: true,
			errorContains: "invalid character",
		},
		{
			name:          "intent with unicode",
			intent:        "Deploy AMF with π characters",
			expectedError: false,
		},
		{
			name:          "very long intent",
			intent:        strings.Repeat("Deploy AMF with configuration ", 200),
			expectedError: true,
			errorContains: "too long",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateIntentContent(tt.intent)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSecurity(t *testing.T) {
	validator, err := createTestValidator()
	require.NoError(t, err)

	tests := []struct {
		name          string
		intent        string
		expectedError bool
		errorContains string
	}{
		{
			name:          "safe intent",
			intent:        "Deploy AMF network function with configuration",
			expectedError: false,
		},
		{
			name:          "script tag injection",
			intent:        "Deploy AMF <script>alert('xss')</script>",
			expectedError: true,
			errorContains: "malicious",
		},
		{
			name:          "SQL injection",
			intent:        "Deploy SMF'; DROP TABLE users; --",
			expectedError: true,
			errorContains: "malicious",
		},
		{
			name:          "shell command injection",
			intent:        "Deploy UPF && rm -rf /",
			expectedError: true,
			errorContains: "malicious",
		},
		{
			name:          "encoded characters",
			intent:        "Deploy AMF \\x41\\x42",
			expectedError: true,
			errorContains: "encoded",
		},
		{
			name:          "unicode encoded",
			intent:        "Deploy AMF \\u0041\\u0042",
			expectedError: true,
			errorContains: "encoded",
		},
		{
			name:          "path traversal attempt",
			intent:        "Deploy AMF with config from ../../../etc/passwd",
			expectedError: true,
			errorContains: "malicious",
		},
		{
			name:          "LDAP injection",
			intent:        "Deploy AMF *)(cn=*",
			expectedError: true,
			errorContains: "malicious",
		},
		{
			name:          "safe special characters",
			intent:        "Deploy AMF with port 8080 and memory 512Mi",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateSecurity(tt.intent)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTelecomRelevance(t *testing.T) {
	validator, err := createTestValidator()
	require.NoError(t, err)

	tests := []struct {
		name          string
		intent        string
		expectedError bool
		errorContains string
	}{
		{
			name:          "5G core AMF intent",
			intent:        "Deploy AMF network function for 5G core",
			expectedError: false,
		},
		{
			name:          "O-RAN intent",
			intent:        "Configure O-RAN components with O-CU and O-DU",
			expectedError: false,
		},
		{
			name:          "network slice intent",
			intent:        "Deploy eMBB slice for enhanced mobile broadband",
			expectedError: false,
		},
		{
			name:          "multiple telecom keywords",
			intent:        "Configure SMF and UPF for 5G SA deployment with network slicing",
			expectedError: false,
		},
		{
			name:          "minimal telecom relevance",
			intent:        "Configure network slice",
			expectedError: false,
		},
		{
			name:          "non-telecom intent",
			intent:        "Deploy web application with database",
			expectedError: true,
			errorContains: "telecommunications",
		},
		{
			name:          "generic deployment",
			intent:        "Deploy microservice with load balancer",
			expectedError: true,
			errorContains: "telecommunications",
		},
		{
			name:          "cloud infrastructure only",
			intent:        "Setup Kubernetes cluster with monitoring",
			expectedError: true,
			errorContains: "telecommunications",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateTelecomRelevance(tt.intent)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateComplexity(t *testing.T) {
	validator, err := createTestValidator()
	require.NoError(t, err)

	tests := []struct {
		name          string
		intent        string
		expectedError bool
		errorContains string
	}{
		{
			name:          "normal complexity",
			intent:        "Deploy AMF network function with configuration",
			expectedError: false,
		},
		{
			name:          "no words",
			intent:        "!@#$%^&*()",
			expectedError: true,
			errorContains: "no recognizable words",
		},
		{
			name:          "too many words",
			intent:        strings.Repeat("word ", 200),
			expectedError: true,
			errorContains: "too complex",
		},
		{
			name:          "very long words",
			intent:        "Deploy " + strings.Repeat("a", 50) + " function",
			expectedError: true,
			errorContains: "suspiciously long words",
		},
		{
			name:          "excessive repetition",
			intent:        "Deploy Deploy Deploy Deploy Deploy AMF AMF AMF AMF function",
			expectedError: true,
			errorContains: "excessive word repetition",
		},
		{
			name:          "reasonable repetition",
			intent:        "Deploy AMF and SMF network functions",
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateComplexity(tt.intent)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateResourceNaming(t *testing.T) {
	validator, err := createTestValidator()
	require.NoError(t, err)

	tests := []struct {
		name          string
		networkIntent *nephoranv1.NetworkIntent
		expectedError bool
		errorContains string
	}{
		{
			name:          "valid name",
			networkIntent: createTestNetworkIntent("valid-intent-name", "default", "Deploy AMF"),
			expectedError: false,
		},
		{
			name:          "too short name",
			networkIntent: createTestNetworkIntent("ab", "default", "Deploy AMF"),
			expectedError: true,
			errorContains: "too short",
		},
		{
			name:          "too long name",
			networkIntent: createTestNetworkIntent(strings.Repeat("a", 65), "default", "Deploy AMF"),
			expectedError: true,
			errorContains: "too long",
		},
		{
			name:          "system reserved prefix",
			networkIntent: createTestNetworkIntent("system-intent", "default", "Deploy AMF"),
			expectedError: true,
			errorContains: "reserved",
		},
		{
			name:          "admin reserved prefix",
			networkIntent: createTestNetworkIntent("admin-config", "default", "Deploy AMF"),
			expectedError: true,
			errorContains: "reserved",
		},
		{
			name:          "temp prefix (allowed)",
			networkIntent: createTestNetworkIntent("temp-test-intent", "default", "Deploy AMF"),
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateResourceNaming(tt.networkIntent)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateIntentCoherence(t *testing.T) {
	validator, err := createTestValidator()
	require.NoError(t, err)

	tests := []struct {
		name          string
		intent        string
		expectedError bool
		errorContains string
	}{
		{
			name:          "actionable intent with deploy",
			intent:        "Deploy AMF network function",
			expectedError: false,
		},
		{
			name:          "actionable intent with configure",
			intent:        "Configure SMF with UPF integration",
			expectedError: false,
		},
		{
			name:          "actionable intent with setup",
			intent:        "Setup O-RAN components",
			expectedError: false,
		},
		{
			name:          "actionable intent with scale",
			intent:        "Scale UPF instances to handle increased traffic",
			expectedError: false,
		},
		{
			name:          "non-actionable intent",
			intent:        "AMF is a network function",
			expectedError: true,
			errorContains: "actionable",
		},
		{
			name:          "question instead of action",
			intent:        "What is SMF configuration?",
			expectedError: true,
			errorContains: "actionable",
		},
		{
			name:          "vague intent",
			intent:        "Something with network functions",
			expectedError: true,
			errorContains: "actionable",
		},
		{
			name:          "conflicting actions",
			intent:        "Deploy and delete AMF function simultaneously",
			expectedError: true,
			errorContains: "conflicting",
		},
		{
			name:          "enable and disable conflict",
			intent:        "Enable and disable network slicing",
			expectedError: true,
			errorContains: "conflicting",
		},
		{
			name:          "start and stop conflict",
			intent:        "Start and stop UPF service",
			expectedError: true,
			errorContains: "conflicting",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.validateIntentCoherence(tt.intent)

			if tt.expectedError {
				assert.Error(t, err)
				if tt.errorContains != "" {
					assert.Contains(t, strings.ToLower(err.Error()), strings.ToLower(tt.errorContains))
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Edge case tests
func TestValidatorEdgeCases(t *testing.T) {
	validator, err := createTestValidator()
	require.NoError(t, err)

	tests := []struct {
		name            string
		setupRequest    func() admission.Request
		expectedAllowed bool
		expectedMessage string
	}{
		{
			name: "invalid JSON in request",
			setupRequest: func() admission.Request {
				return admission.Request{
					AdmissionRequest: admissionv1.AdmissionRequest{
						UID:       "test-uid",
						Operation: admissionv1.Create,
						Object: runtime.RawExtension{
							Raw: []byte(`{invalid json`),
						},
					},
				}
			},
			expectedAllowed: false,
			expectedMessage: "decode",
		},
		{
			name: "empty request object",
			setupRequest: func() admission.Request {
				return admission.Request{
					AdmissionRequest: admissionv1.AdmissionRequest{
						UID:       "test-uid",
						Operation: admissionv1.Create,
						Object:    runtime.RawExtension{},
					},
				}
			},
			expectedAllowed: false,
			expectedMessage: "decode",
		},
		{
			name: "update operation",
			setupRequest: func() admission.Request {
				ni := createTestNetworkIntent("test-update", "default", "Update AMF configuration")
				return createAdmissionRequest(ni, admissionv1.Update)
			},
			expectedAllowed: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := tt.setupRequest()
			ctx := context.Background()

			response := validator.Handle(ctx, req)

			assert.Equal(t, tt.expectedAllowed, response.Allowed)

			if tt.expectedMessage != "" {
				assert.Contains(t, strings.ToLower(response.Result.Message), strings.ToLower(tt.expectedMessage))
			}
		})
	}
}

// Benchmark tests for webhook performance
func BenchmarkValidateIntentContent(b *testing.B) {
	validator, _ := createTestValidator()
	intent := "Deploy AMF network function with comprehensive configuration for 5G core deployment"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.validateIntentContent(intent)
	}
}

func BenchmarkValidateSecurity(b *testing.B) {
	validator, _ := createTestValidator()
	intent := "Deploy AMF network function with port 8080 and memory 512Mi for production environment"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.validateSecurity(intent)
	}
}

func BenchmarkValidateTelecomRelevance(b *testing.B) {
	validator, _ := createTestValidator()
	intent := "Deploy AMF and SMF network functions with UPF integration for 5G SA deployment"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.validateTelecomRelevance(intent)
	}
}

func BenchmarkValidateComplexity(b *testing.B) {
	validator, _ := createTestValidator()
	intent := "Deploy comprehensive 5G core network with AMF SMF UPF NSSF components and network slicing support"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		validator.validateComplexity(intent)
	}
}

func BenchmarkFullValidation(b *testing.B) {
	validator, _ := createTestValidator()
	ni := createTestNetworkIntent("benchmark-intent", "default", "Deploy AMF network function for 5G core")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req := createAdmissionRequest(ni, admissionv1.Create)
		validator.Handle(context.Background(), req)
	}
}
