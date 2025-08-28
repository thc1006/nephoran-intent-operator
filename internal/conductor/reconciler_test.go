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

package conductor

import (
	"testing"

	"github.com/go-logr/logr"
)

func TestParseIntentString(t *testing.T) {
	reconciler := &NetworkIntentReconciler{
		Log: logr.Discard(),
	}

	tests := []struct {
		name        string
		intent      string
		namespace   string
		expectedTarget string
		expectedReplicas int
		expectError bool
	}{
		{
			name:             "simple scaling intent",
			intent:           "scale my-app to 5 replicas",
			namespace:        "default",
			expectedTarget:   "my-app",
			expectedReplicas: 5,
			expectError:      false,
		},
		{
			name:             "deployment with instances",
			intent:           "scale deployment web-server to 3 instances",
			namespace:        "production",
			expectedTarget:   "web-server",
			expectedReplicas: 3,
			expectError:      false,
		},
		{
			name:             "app with replicas keyword",
			intent:           "scale app nginx replicas: 2",
			namespace:        "test",
			expectedTarget:   "nginx",
			expectedReplicas: 2,
			expectError:      false,
		},
		{
			name:             "service scale up",
			intent:           "scale service api-gateway 4 replicas",
			namespace:        "api",
			expectedTarget:   "api-gateway",
			expectedReplicas: 4,
			expectError:      false,
		},
		{
			name:             "default replicas when not specified",
			intent:           "scale deployment frontend",
			namespace:        "web",
			expectedTarget:   "frontend",
			expectedReplicas: 1,
			expectError:      false,
		},
		{
			name:        "missing target",
			intent:      "scale to 3 replicas",
			namespace:   "default",
			expectError: true,
		},
		{
			name:        "excessive replica count",
			intent:      "scale my-app to 150 replicas",
			namespace:   "default",
			expectError: true,
		},
		{
			name:             "mixed case and extra spaces",
			intent:           "  Scale  DEPLOYMENT  my-service   to  7   Replicas  ",
			namespace:        "default",
			expectedTarget:   "my-service",
			expectedReplicas: 7,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := reconciler.parseIntentString(tt.intent, tt.namespace)

			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if result.IntentType != "scaling" {
				t.Errorf("expected intent_type 'scaling', got '%s'", result.IntentType)
			}

			if result.Target != tt.expectedTarget {
				t.Errorf("expected target '%s', got '%s'", tt.expectedTarget, result.Target)
			}

			if result.Replicas != tt.expectedReplicas {
				t.Errorf("expected replicas %d, got %d", tt.expectedReplicas, result.Replicas)
			}

			if result.Namespace != tt.namespace {
				t.Errorf("expected namespace '%s', got '%s'", tt.namespace, result.Namespace)
			}
		})
	}
}

func TestCreateIntentFile(t *testing.T) {
	reconciler := &NetworkIntentReconciler{
		Log:       logr.Discard(),
		OutputDir: t.TempDir(),
	}

	intentData := &IntentJSON{
		IntentType: "scaling",
		Target:     "test-app",
		Namespace:  "default",
		Replicas:   3,
		Source:     "user",
		CorrelationID: "test-123",
		Reason:     "Test scaling",
	}

	filePath, err := reconciler.createIntentFile(intentData, "test-intent")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if filePath == "" {
		t.Error("expected file path but got empty string")
	}

	// Verify the file exists and can be read
	// In a more comprehensive test, we would also validate the JSON content
}