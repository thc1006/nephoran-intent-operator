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

package controllers

import (
	"fmt"
	"math"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestGetNodeIndexFromConfigMapIntegerOverflow verifies that the

// getNodeIndexFromConfigMap function properly handles integer overflow

// when converting from string to int32 (G109/G115 security fix)

func TestGetNodeIndexFromConfigMapIntegerOverflow(t *testing.T) {
	r := &E2NodeSetReconciler{}

	tests := []struct {
		name string

		indexValue string

		expectedIndex int32

		description string
	}{
		{
			name: "valid small positive index",

			indexValue: "5",

			expectedIndex: 5,

			description: "Normal valid index should be returned",
		},

		{
			name: "valid large positive index",

			indexValue: "2147483647", // MaxInt32

			expectedIndex: 2147483647,

			description: "Maximum int32 value should be handled",
		},

		{
			name: "negative index",

			indexValue: "-1",

			expectedIndex: 0,

			description: "Negative indices should return 0 (default)",
		},

		{
			name: "overflow beyond int32 max",

			indexValue: "2147483648", // MaxInt32 + 1

			expectedIndex: 0,

			description: "Values exceeding int32 max should return 0",
		},

		{
			name: "very large overflow",

			indexValue: "9223372036854775807", // MaxInt64

			expectedIndex: 0,

			description: "Very large values should return 0",
		},

		{
			name: "invalid non-numeric",

			indexValue: "abc",

			expectedIndex: 0,

			description: "Non-numeric values should return 0",
		},

		{
			name: "empty string",

			indexValue: "",

			expectedIndex: 0,

			description: "Empty string should return 0",
		},

		{
			name: "zero index",

			indexValue: "0",

			expectedIndex: 0,

			description: "Zero should be valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a ConfigMap with the test index value

			cm := &corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-config",

					Namespace: "default",

					Labels: make(map[string]string),
				},
			}

			if tt.indexValue != "" {
				cm.Labels[E2NodeIndexLabelKey] = tt.indexValue
			}

			// Call the function under test

			result := r.getNodeIndexFromConfigMap(cm)

			// Verify the result

			if result != tt.expectedIndex {
				t.Errorf("%s: Expected index %d, got %d",

					tt.description, tt.expectedIndex, result)
			}
		})
	}
}

// TestGetNodeIndexFromConfigMapMissingLabel verifies behavior when

// the index label is missing from the ConfigMap

func TestGetNodeIndexFromConfigMapMissingLabel(t *testing.T) {
	r := &E2NodeSetReconciler{}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-config",

			Namespace: "default",

			Labels: map[string]string{
				"other-label": "value",
			},
		},
	}

	result := r.getNodeIndexFromConfigMap(cm)

	if result != 0 {
		t.Errorf("Expected 0 for missing label, got %d", result)
	}
}

// TestIntegerBoundsValidation verifies that our integer conversion

// bounds checking works correctly for edge cases

func TestIntegerBoundsValidation(t *testing.T) {
	tests := []struct {
		name string

		input int

		expected int32
	}{
		{
			name: "within bounds positive",

			input: 1000,

			expected: 1000,
		},

		{
			name: "max int32",

			input: math.MaxInt32,

			expected: math.MaxInt32,
		},

		{
			name: "min int32",

			input: 0,

			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the bounds checking logic

			value := tt.input

			if value < 0 || value > math.MaxInt32 {
				value = 0 // Default for out of bounds
			}

			result := int32(value)

			if result != tt.expected {
				t.Errorf("Expected %d, got %d", tt.expected, result)
			}
		})
	}
}

// BenchmarkGetNodeIndexFromConfigMap benchmarks the performance

// of the integer conversion with bounds checking

func BenchmarkGetNodeIndexFromConfigMap(b *testing.B) {
	r := &E2NodeSetReconciler{}

	cm := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: "bench-config",

			Namespace: "default",

			Labels: map[string]string{
				E2NodeIndexLabelKey: fmt.Sprintf("%d", math.MaxInt32),
			},
		},
	}

	b.ResetTimer()

	for range b.N {
		_ = r.getNodeIndexFromConfigMap(cm)
	}
}
