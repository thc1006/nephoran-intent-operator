package v1alpha1

import (
	"encoding/json"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// BenchmarkNetworkIntentJSONMarshal tests JSON marshaling performance for NetworkIntent
func BenchmarkNetworkIntentJSONMarshal(b *testing.B) {
	// Create a realistic NetworkIntent object with typical data
	intent := &NetworkIntent{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "nephio.io/v1alpha1",
			Kind:       "NetworkIntent",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-intent",
			Namespace: "default",
			Labels: map[string]string{
				"app":     "test",
				"version": "v1",
			},
		},
		Spec: NetworkIntentSpec{
			ScalingPriority: "high",
			TargetClusters:  []string{"cluster-1", "cluster-2", "cluster-3"},
		},
		Status: NetworkIntentStatus{
			Conditions: []metav1.Condition{
				{
					Type:               "Ready",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "NetworkIntentReady",
					Message:            "NetworkIntent is ready for processing",
				},
			},
			ObservedGeneration: 1,
		},
	}

	// Run the benchmark
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(intent)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// TestNetworkIntentValidation tests validation of NetworkIntent fields
func TestNetworkIntentValidation(b *testing.T) {
	tests := []struct {
		name    string
		intent  NetworkIntent
		wantErr bool
	}{
		{
			name: "valid_intent_with_high_priority",
			intent: NetworkIntent{
				Spec: NetworkIntentSpec{
					ScalingPriority: "high",
					TargetClusters:  []string{"cluster-1"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_intent_with_medium_priority",
			intent: NetworkIntent{
				Spec: NetworkIntentSpec{
					ScalingPriority: "medium",
					TargetClusters:  []string{"cluster-1", "cluster-2"},
				},
			},
			wantErr: false,
		},
		{
			name: "valid_intent_with_low_priority",
			intent: NetworkIntent{
				Spec: NetworkIntentSpec{
					ScalingPriority: "low",
					TargetClusters:  []string{"cluster-1", "cluster-2", "cluster-3"},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(t *testing.T) {
			// Test JSON marshaling
			_, err := json.Marshal(tt.intent)
			if (err != nil) != tt.wantErr {
				t.Errorf("JSON Marshal error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			// Test basic field validation
			if tt.intent.Spec.ScalingPriority != "" {
				validPriorities := []string{"low", "medium", "high"}
				valid := false
				for _, p := range validPriorities {
					if tt.intent.Spec.ScalingPriority == p {
						valid = true
						break
					}
				}
				if !valid {
					t.Errorf("Invalid ScalingPriority: %s", tt.intent.Spec.ScalingPriority)
				}
			}
		})
	}
}