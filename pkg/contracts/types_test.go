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

package contracts

import (
	"testing"
	"time"
)

func TestNetworkIntentValidation(t *testing.T) {
	tests := []struct {
		name    string
		intent  NetworkIntent
		wantErr bool
	}{
		{
			name: "valid network intent",
			intent: NetworkIntent{
				ID:              "test-intent-1",
				Type:            "scaling",
				Priority:        5,
				Description:     "Test scaling intent for demo",
				Parameters:      json.RawMessage(`{}`),
				TargetResources: []string{"deployment/test"},
				Status:          "pending",
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing ID",
			intent: NetworkIntent{
				Type:            "scaling",
				Priority:        5,
				Description:     "Test scaling intent for demo",
				Parameters:      json.RawMessage(`{}`),
				TargetResources: []string{"deployment/test"},
				Status:          "pending",
			},
			wantErr: true,
		},
		{
			name: "empty target resources",
			intent: NetworkIntent{
				ID:              "test-intent-2",
				Type:            "scaling",
				Priority:        5,
				Description:     "Test scaling intent for demo",
				Parameters:      json.RawMessage(`{}`),
				TargetResources: []string{},
				Status:          "pending",
			},
			wantErr: true,
		},
		{
			name: "invalid priority",
			intent: NetworkIntent{
				ID:              "test-intent-3",
				Type:            "scaling",
				Priority:        15, // Invalid: > 10
				Description:     "Test scaling intent for demo",
				Parameters:      json.RawMessage(`{}`),
				TargetResources: []string{"deployment/test"},
				Status:          "pending",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.intent.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("NetworkIntent.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestScalingIntentValidation(t *testing.T) {
	tests := []struct {
		name    string
		intent  ScalingIntent
		wantErr bool
	}{
		{
			name: "valid scaling intent",
			intent: ScalingIntent{
				ID:             "scale-intent-1",
				ResourceType:   "deployment",
				ResourceName:   "test-deployment",
				Namespace:      "default",
				TargetScale:    5,
				CurrentScale:   3,
				ScaleDirection: "up",
				Reason:         "Increased load detected",
				Status:         "pending",
				CreatedAt:      time.Now(),
			},
			wantErr: false,
		},
		{
			name: "missing namespace",
			intent: ScalingIntent{
				ID:             "scale-intent-2",
				ResourceType:   "deployment",
				ResourceName:   "test-deployment",
				TargetScale:    5,
				CurrentScale:   3,
				ScaleDirection: "up",
				Reason:         "Increased load detected",
				Status:         "pending",
			},
			wantErr: true,
		},
		{
			name: "negative target scale",
			intent: ScalingIntent{
				ID:             "scale-intent-3",
				ResourceType:   "deployment",
				ResourceName:   "test-deployment",
				Namespace:      "default",
				TargetScale:    -1, // Invalid: negative
				CurrentScale:   3,
				ScaleDirection: "up",
				Reason:         "Test negative scale",
				Status:         "pending",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.intent.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("ScalingIntent.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestIntentContractInterface(t *testing.T) {
	now := time.Now()

	// Test NetworkIntent implements IntentContract
	var ni IntentContract = &NetworkIntent{
		ID:        "test-ni",
		Type:      "scaling",
		Status:    "pending",
		CreatedAt: now,
		UpdatedAt: now,
	}

	if ni.GetID() != "test-ni" {
		t.Errorf("NetworkIntent.GetID() = %v, want %v", ni.GetID(), "test-ni")
	}

	if ni.GetType() != "scaling" {
		t.Errorf("NetworkIntent.GetType() = %v, want %v", ni.GetType(), "scaling")
	}

	ni.SetStatus("processing")
	if ni.GetStatus() != "processing" {
		t.Errorf("NetworkIntent.GetStatus() = %v, want %v", ni.GetStatus(), "processing")
	}

	// Test ScalingIntent implements IntentContract
	var si IntentContract = &ScalingIntent{
		ID:        "test-si",
		Status:    "pending",
		CreatedAt: now,
	}

	if si.GetID() != "test-si" {
		t.Errorf("ScalingIntent.GetID() = %v, want %v", si.GetID(), "test-si")
	}

	if si.GetType() != "scaling" {
		t.Errorf("ScalingIntent.GetType() = %v, want %v", si.GetType(), "scaling")
	}

	si.SetStatus("completed")
	if si.GetStatus() != "completed" {
		t.Errorf("ScalingIntent.GetStatus() = %v, want %v", si.GetStatus(), "completed")
	}
}

