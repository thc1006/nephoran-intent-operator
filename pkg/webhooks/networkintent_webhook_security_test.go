/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package webhooks

import (
	"context"
	"strings"
	"testing"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestSecurityValidation tests the enhanced security validation for NetworkIntent
func TestSecurityValidation(t *testing.T) {
	validator := NewNetworkIntentValidator()
	ctx := context.Background()

	tests := []struct {
		name        string
		intent      string
		shouldFail  bool
		errorMsg    string
	}{
		// Valid intents - should pass
		{
			name:       "Valid AMF deployment intent",
			intent:     "Deploy a high-availability AMF instance for production with auto-scaling",
			shouldFail: false,
		},
		{
			name:       "Valid network slice intent",
			intent:     "Create a network slice for URLLC with 1ms latency requirements",
			shouldFail: false,
		},
		{
			name:       "Valid QoS configuration",
			intent:     "Configure QoS policies for enhanced mobile broadband services",
			shouldFail: false,
		},
		{
			name:       "Valid with allowed punctuation",
			intent:     "Deploy AMF, SMF, and UPF components: high-availability (production)",
			shouldFail: false,
		},
		{
			name:       "Valid with brackets",
			intent:     "Configure network functions [AMF] with parameters [auto-scaling]",
			shouldFail: false,
		},

		// Security violations - should fail
		{
			name:       "SQL injection attempt with quotes",
			intent:     "Deploy AMF'; DROP TABLE users; --",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Command injection with backticks",
			intent:     "Deploy AMF `rm -rf /`",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Script injection with angle brackets",
			intent:     "Deploy <script>alert('xss')</script> AMF",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Path traversal attempt",
			intent:     "Deploy AMF ../../etc/passwd",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Command injection with dollar sign",
			intent:     "Deploy AMF $(whoami)",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "URL injection with @",
			intent:     "Deploy AMF @evil.com",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Hash injection",
			intent:     "Deploy AMF #../../bin/sh",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Percent encoding attempt",
			intent:     "Deploy AMF %2e%2e%2f",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Ampersand for command chaining",
			intent:     "Deploy AMF & cat /etc/passwd",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Asterisk glob pattern",
			intent:     "Deploy AMF * /etc/*",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Equals sign for SQL",
			intent:     "Deploy AMF where id=1",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Question mark injection",
			intent:     "Deploy AMF?file=/etc/passwd",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Exclamation for history expansion",
			intent:     "Deploy AMF !!/bin/sh",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Pipe character",
			intent:     "Deploy AMF | cat sensitive",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Curly braces",
			intent:     "Deploy AMF {malicious}",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Plus sign",
			intent:     "Deploy AMF+union+select",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Tilde for home directory",
			intent:     "Deploy AMF ~/sensitive",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},
		{
			name:       "Caret character",
			intent:     "Deploy AMF^command",
			shouldFail: true,
			errorMsg:   "disallowed character",
		},

		// Pattern-based attacks (even with allowed chars)
		{
			name:       "SQL pattern with allowed chars",
			intent:     "Deploy AMF drop table users",
			shouldFail: true,
			errorMsg:   "suspicious pattern",
		},
		{
			name:       "Command pattern with allowed chars",
			intent:     "Deploy AMF rm -rf root",
			shouldFail: true,
			errorMsg:   "suspicious pattern",
		},
		{
			name:       "Spaced attack pattern",
			intent:     "d r o p table AMF",
			shouldFail: true,
			errorMsg:   "spaced pattern",
		},
		{
			name:       "Long alphanumeric encoding",
			intent:     "Deploy " + strings.Repeat("A", 45) + " AMF",
			shouldFail: true,
			errorMsg:   "long unbroken alphanumeric",
		},
		{
			name:       "Excessive punctuation",
			intent:     "Deploy AMF" + strings.Repeat(".", 15),
			shouldFail: true,
			errorMsg:   "excessive repeated punctuation",
		},
		{
			name:       "Repeated punctuation pattern",
			intent:     "Deploy AMF.......",
			shouldFail: true,
			errorMsg:   "excessive repeated punctuation",
		},

		// Length validation
		{
			name:       "Too long intent",
			intent:     "Deploy AMF " + strings.Repeat("a", 1000),
			shouldFail: true,
			errorMsg:   "exceeds maximum length",
		},
		{
			name:       "Empty intent",
			intent:     "",
			shouldFail: true,
			errorMsg:   "empty",
		},
		{
			name:       "Whitespace only",
			intent:     "   \t\n  ",
			shouldFail: true,
			errorMsg:   "empty or only whitespace",
		},

		// Telecom relevance
		{
			name:       "Non-telecom intent",
			intent:     "Order pizza with extra cheese",
			shouldFail: true,
			errorMsg:   "telecommunications-related",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ni := &nephoranv1.NetworkIntent{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-intent",
					Namespace: "test-namespace",
				},
				Spec: nephoranv1.NetworkIntentSpec{
					Intent: tt.intent,
				},
			}

			err := validator.validateNetworkIntent(ctx, ni)

			if tt.shouldFail {
				if err == nil {
					t.Errorf("Expected validation to fail for %s, but it passed", tt.name)
				} else if tt.errorMsg != "" && !strings.Contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMsg, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected validation to pass for %s, but got error: %v", tt.name, err)
				}
			}
		})
	}
}

// TestCharacterAllowlist tests the specific character allowlist implementation
func TestCharacterAllowlist(t *testing.T) {
	tests := []struct {
		char    rune
		allowed bool
	}{
		// Allowed characters
		{'a', true}, {'z', true}, {'A', true}, {'Z', true},
		{'0', true}, {'9', true},
		{' ', true}, {'-', true}, {'_', true}, {'.', true},
		{',', true}, {';', true}, {':', true},
		{'(', true}, {')', true}, {'[', true}, {']', true},
		
		// Disallowed characters
		{'<', false}, {'>', false}, {'/', false}, {'\\', false},
		{'@', false}, {'#', false}, {'$', false}, {'%', false},
		{'&', false}, {'*', false}, {'=', false}, {'?', false},
		{'!', false}, {'|', false}, {'+', false}, {'~', false},
		{'^', false}, {'`', false}, {'"', false}, {'\'', false},
		{'{', false}, {'}', false},
	}

	for _, tt := range tests {
		t.Run(string(tt.char), func(t *testing.T) {
			result := isAllowedChar(tt.char)
			if result != tt.allowed {
				t.Errorf("isAllowedChar('%c') = %v, want %v", tt.char, result, tt.allowed)
			}
		})
	}
}

// BenchmarkSecurityValidation benchmarks the security validation performance
func BenchmarkSecurityValidation(b *testing.B) {
	validator := NewNetworkIntentValidator()
	ctx := context.Background()
	
	ni := &nephoranv1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-intent",
			Namespace: "benchmark-namespace",
		},
		Spec: nephoranv1.NetworkIntentSpec{
			Intent: "Deploy a high-availability AMF instance for production with auto-scaling enabled and QoS policies configured for enhanced mobile broadband services",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = validator.validateNetworkIntent(ctx, ni)
	}
}