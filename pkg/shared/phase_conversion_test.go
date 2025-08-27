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

package shared

import (
	"testing"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
)

func TestProcessingPhaseToNetworkIntentPhase(t *testing.T) {
	tests := []struct {
		name     string
		input    interfaces.ProcessingPhase
		expected nephoranv1.NetworkIntentPhase
	}{
		{
			name:     "IntentReceived maps to Pending",
			input:    interfaces.PhaseIntentReceived,
			expected: nephoranv1.NetworkIntentPhasePending,
		},
		{
			name:     "LLMProcessing maps to Processing",
			input:    interfaces.PhaseLLMProcessing,
			expected: nephoranv1.NetworkIntentPhaseProcessing,
		},
		{
			name:     "ResourcePlanning maps to Deploying",
			input:    interfaces.PhaseResourcePlanning,
			expected: nephoranv1.NetworkIntentPhaseDeploying,
		},
		{
			name:     "Completed maps to Deployed",
			input:    interfaces.PhaseCompleted,
			expected: nephoranv1.NetworkIntentPhaseDeployed,
		},
		{
			name:     "Failed maps to Failed",
			input:    interfaces.PhaseFailed,
			expected: nephoranv1.NetworkIntentPhaseFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ProcessingPhaseToNetworkIntentPhase(tt.input)
			if result != tt.expected {
				t.Errorf("ProcessingPhaseToNetworkIntentPhase(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestStringToNetworkIntentPhase(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected nephoranv1.NetworkIntentPhase
	}{
		{
			name:     "Pending string maps to Pending phase",
			input:    "Pending",
			expected: nephoranv1.NetworkIntentPhasePending,
		},
		{
			name:     "Processing string maps to Processing phase",
			input:    "Processing",
			expected: nephoranv1.NetworkIntentPhaseProcessing,
		},
		{
			name:     "Invalid string maps to Pending",
			input:    "InvalidPhase",
			expected: nephoranv1.NetworkIntentPhasePending,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := StringToNetworkIntentPhase(tt.input)
			if result != tt.expected {
				t.Errorf("StringToNetworkIntentPhase(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestNetworkIntentPhaseToString(t *testing.T) {
	tests := []struct {
		name     string
		input    nephoranv1.NetworkIntentPhase
		expected string
	}{
		{
			name:     "Pending phase to string",
			input:    nephoranv1.NetworkIntentPhasePending,
			expected: "Pending",
		},
		{
			name:     "Deployed phase to string",
			input:    nephoranv1.NetworkIntentPhaseDeployed,
			expected: "Deployed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NetworkIntentPhaseToString(tt.input)
			if result != tt.expected {
				t.Errorf("NetworkIntentPhaseToString(%s) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}
