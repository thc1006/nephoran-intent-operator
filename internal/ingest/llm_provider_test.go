package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRulesProvider tests the rules-based provider
func TestRulesProvider(t *testing.T) {
	provider := NewRulesProvider()
	ctx := context.Background()

	tests := []struct {
		name          string
		input         string
		expectError   bool
		expectedData  map[string]interface{}
	}{
		{
			name:        "Full Scale Pattern",
			input:       "scale odu-high-phy to 5 in ns oran-odu",
			expectError: false,
			expectedData: map[string]interface{}{
				"intent_type":    "scaling",
				"target":    "odu-high-phy",
				"replicas":  5,
				"namespace": "oran-odu",
			},
		},
		{
			name:        "Simple Scale Pattern",
			input:       "scale cu-cp to 3",
			expectError: false,
			expectedData: map[string]interface{}{
				"intent_type":    "scaling",
				"target":    "cu-cp",
				"replicas":  3,
				"namespace": "default",
			},
		},
		{
			name:        "Scale Out Pattern",
			input:       "scale out du-manager by 2 in ns oran-du",
			expectError: false,
			expectedData: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "du-manager",
				"replicas":    2, // MVP treats delta as absolute replicas
				"namespace":   "oran-du",
			},
		},
		{
			name:        "Scale In Pattern",
			input:       "scale in du-manager by 1 in ns oran-du",
			expectError: false,
			expectedData: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "du-manager",
				"replicas":    1, // MVP sets minimum replicas for scale in
				"namespace":   "oran-du",
			},
		},
		{
			name:        "Case Insensitive",
			input:       "SCALE ODU-HIGH-PHY TO 10 IN NS ORAN-ODU",
			expectError: false,
			expectedData: map[string]interface{}{
				"intent_type":    "scaling",
				"target":    "ODU-HIGH-PHY",
				"replicas":  10,
				"namespace": "ORAN-ODU",
			},
		},
		{
			name:        "Scale Out Default Namespace",
			input:       "scale out test-service by 3",
			expectError: false,
			expectedData: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-service",
				"replicas":    3, // MVP treats delta as absolute replicas
				"namespace":   "default",
			},
		},
		{
			name:        "Scale In Default Namespace",
			input:       "scale in test-service by 2",
			expectError: false,
			expectedData: map[string]interface{}{
				"intent_type": "scaling",
				"target":      "test-service",
				"replicas":    1, // MVP sets minimum replicas for scale in
				"namespace":   "default",
			},
		},
		{
			name:        "Invalid Intent Format",
			input:       "deploy new service",
			expectError: true,
		},
		{
			name:        "Empty Input",
			input:       "",
			expectError: true,
		},
		{
			name:        "Invalid Replica Count",
			input:       "scale test to abc",
			expectError: true,
		},
		{
			name:        "Zero Replicas",
			input:       "scale test to 0",
			expectError: false,
			expectedData: map[string]interface{}{
				"intent_type":    "scaling",
				"target":    "test",
				"replicas":  0,
				"namespace": "default",
			},
		},
		{
			name:        "Large Replica Count",
			input:       "scale test to 100",
			expectError: false,
			expectedData: map[string]interface{}{
				"intent_type":    "scaling",
				"target":    "test",
				"replicas":  100,
				"namespace": "default",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := provider.ParseIntent(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err, "Expected error for input: %s", tt.input)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err, "Expected no error for input: %s", tt.input)
				assert.NotNil(t, result)

				// Check expected data
				for key, expectedValue := range tt.expectedData {
					actualValue, exists := result[key]
					assert.True(t, exists, "Expected key %s to exist in result", key)
					assert.Equal(t, expectedValue, actualValue, "Key %s has incorrect value", key)
				}
			}
		})
	}
}

// TestMockLLMProvider tests the mock LLM provider
func TestMockLLMProvider(t *testing.T) {
	provider := NewMockLLMProvider()
	ctx := context.Background()

	// Test that mock provider produces same results as rules provider
	testCases := []string{
		"scale odu-high-phy to 5 in ns oran-odu",
		"scale cu-cp to 3",
		"scale out du-manager by 2 in ns oran-du",
		"scale in du-manager by 1 in ns oran-du",
	}

	rulesProvider := NewRulesProvider()

	for _, testCase := range testCases {
		t.Run("Mock_"+testCase, func(t *testing.T) {
			mockResult, mockErr := provider.ParseIntent(ctx, testCase)
			rulesResult, rulesErr := rulesProvider.ParseIntent(ctx, testCase)

			// Both should produce the same result
			assert.Equal(t, rulesErr, mockErr, "Error states should match")
			assert.Equal(t, rulesResult, mockResult, "Results should match")
		})
	}
}

// TestProviderFactory tests the provider factory
func TestProviderFactory(t *testing.T) {
	tests := []struct {
		name         string
		mode         string
		provider     string
		expectError  bool
		expectedType string
	}{
		{
			name:         "Rules Mode",
			mode:         "rules",
			provider:     "",
			expectError:  false,
			expectedType: "rules",
		},
		{
			name:         "LLM Mode with Mock Provider",
			mode:         "llm",
			provider:     "mock",
			expectError:  false,
			expectedType: "mock",
		},
		{
			name:         "Default Mode (Rules)",
			mode:         "",
			provider:     "",
			expectError:  false,
			expectedType: "rules",
		},
		{
			name:         "LLM Mode Default Provider (Mock)",
			mode:         "llm",
			provider:     "",
			expectError:  false,
			expectedType: "mock",
		},
		{
			name:        "Invalid Mode",
			mode:        "invalid",
			provider:    "",
			expectError: true,
		},
		{
			name:        "Invalid LLM Provider",
			mode:        "llm",
			provider:    "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := NewProvider(tt.mode, tt.provider)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.expectedType, result.Name())
			}
		})
	}
}

// TestProviderContext tests context handling in providers
func TestProviderContext(t *testing.T) {
	provider := NewRulesProvider()

	// Test with cancelled context
	t.Run("Cancelled Context", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		// Provider should still work (rules don't use context for cancellation)
		result, err := provider.ParseIntent(ctx, "scale test to 3")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})

	// Test with timeout context
	t.Run("Timeout Context", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
		defer cancel()

		// Let context timeout
		time.Sleep(2 * time.Millisecond)

		// Rules provider should still work (doesn't use context timeout)
		result, err := provider.ParseIntent(ctx, "scale test to 3")
		assert.NoError(t, err)
		assert.NotNil(t, result)
	})
}

// TestProviderEdgeCases tests edge cases for providers
func TestProviderEdgeCases(t *testing.T) {
	provider := NewRulesProvider()
	ctx := context.Background()

	tests := []struct {
		name        string
		input       string
		expectError bool
		description string
	}{
		{
			name:        "Whitespace Only",
			input:       "   \t\n   ",
			expectError: true,
			description: "Should fail on whitespace-only input",
		},
		{
			name:        "Leading/Trailing Whitespace",
			input:       "   scale test to 3   ",
			expectError: false,
			description: "Should handle leading/trailing whitespace",
		},
		{
			name:        "Multiple Spaces",
			input:       "scale    test    to    3",
			expectError: false, // Regex handles multiple spaces correctly
			description: "Multiple spaces between words",
		},
		{
			name:        "Hyphenated Target Names",
			input:       "scale my-service-name to 5",
			expectError: false,
			description: "Should handle hyphenated service names",
		},
		{
			name:        "Hyphenated Namespace Names",
			input:       "scale test to 3 in ns my-namespace",
			expectError: false,
			description: "Should handle hyphenated namespace names",
		},
		{
			name:        "Numeric Target Names",
			input:       "scale service123 to 3",
			expectError: false,
			description: "Should handle numeric characters in target names",
		},
		{
			name:        "Very Large Replica Count",
			input:       "scale test to 999999999",
			expectError: false,
			description: "Should handle large replica counts within int32 range",
		},
		{
			name:        "Mixed Case Patterns",
			input:       "Scale TEST-service To 3 In Ns MY-NAMESPACE",
			expectError: false,
			description: "Should handle mixed case patterns",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := provider.ParseIntent(ctx, tt.input)

			if tt.expectError {
				assert.Error(t, err, tt.description)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err, tt.description)
				assert.NotNil(t, result)
			}
		})
	}
}

// TestProviderPerformance tests performance characteristics of providers
func TestProviderPerformance(t *testing.T) {
	provider := NewRulesProvider()
	ctx := context.Background()
	testIntent := "scale odu-high-phy to 5 in ns oran-odu"

	// Test repeated parsing
	t.Run("Repeated Parsing", func(t *testing.T) {
		iterations := 1000
		start := time.Now()

		for i := 0; i < iterations; i++ {
			result, err := provider.ParseIntent(ctx, testIntent)
			require.NoError(t, err)
			require.NotNil(t, result)
		}

		duration := time.Since(start)
		avgTime := duration / time.Duration(iterations)
		
		t.Logf("Average parsing time: %v", avgTime)
		
		// Performance assertion: should parse in less than 1ms on average
		assert.Less(t, avgTime, 1*time.Millisecond, "Parsing should be fast")
	})
}

// TestProviderThreadSafety tests thread safety of providers
func TestProviderThreadSafety(t *testing.T) {
	provider := NewRulesProvider()
	ctx := context.Background()
	
	numGoroutines := 100
	numIterations := 10
	errors := make(chan error, numGoroutines*numIterations)
	
	// Launch multiple goroutines
	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < numIterations; j++ {
				testIntent := "scale test to 3"
				result, err := provider.ParseIntent(ctx, testIntent)
				if err != nil {
					errors <- err
					return
				}
				if result == nil {
					errors <- assert.AnError
					return
				}
			}
			errors <- nil
		}(i)
	}
	
	// Collect results
	for i := 0; i < numGoroutines; i++ {
		select {
		case err := <-errors:
			assert.NoError(t, err, "Goroutine should not have errors")
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for goroutines")
		}
	}
}

// BenchmarkRulesProvider benchmarks the rules provider
func BenchmarkRulesProvider(b *testing.B) {
	provider := NewRulesProvider()
	ctx := context.Background()
	testIntent := "scale odu-high-phy to 5 in ns oran-odu"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := provider.ParseIntent(ctx, testIntent)
			if err != nil {
				b.Error(err)
			}
		}
	})
}

// BenchmarkMockLLMProvider benchmarks the mock LLM provider
func BenchmarkMockLLMProvider(b *testing.B) {
	provider := NewMockLLMProvider()
	ctx := context.Background()
	testIntent := "scale odu-high-phy to 5 in ns oran-odu"

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := provider.ParseIntent(ctx, testIntent)
			if err != nil {
				b.Error(err)
			}
		}
	})
}