package v1alpha1

import (
	"encoding/json"
	"testing"

<<<<<<< HEAD
=======
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
>>>>>>> 6835433495e87288b95961af7173d866977175ff
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
<<<<<<< HEAD
		},
		Status: NetworkIntentStatus{
=======
			ScalingIntent: &apiextensionsv1.JSON{
				Raw: []byte(`{"intent_type":"scaling","target":"gnb-simulator","namespace":"ran","replicas":5}`),
			},
			Deployment: DeploymentSpec{
				ClusterSelector: map[string]string{
					"region": "us-west",
					"type":   "edge",
				},
				NetworkFunctions: []NetworkFunction{
					{
						Name:    "gnb-cu",
						Type:    "CNF",
						Version: "1.0.0",
						Config: &apiextensionsv1.JSON{
							Raw: []byte(`{"maxConnections":1000,"timeout":30}`),
						},
						Resources: NetworkFunctionResources{
							CPU:    "2",
							Memory: "4Gi",
						},
					},
				},
				Replicas: 3,
			},
		},
		Status: NetworkIntentStatus{
			Phase: "pending",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
			Conditions: []metav1.Condition{
				{
					Type:               "Ready",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
<<<<<<< HEAD
					Reason:             "NetworkIntentReady",
					Message:            "NetworkIntent is ready for processing",
=======
					Reason:             "Initialized",
					Message:            "NetworkIntent has been initialized",
>>>>>>> 6835433495e87288b95961af7173d866977175ff
				},
			},
			ObservedGeneration: 1,
		},
	}

<<<<<<< HEAD
	// Run the benchmark
	b.ResetTimer()
=======
	b.ReportAllocs()
	b.ResetTimer()

>>>>>>> 6835433495e87288b95961af7173d866977175ff
	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(intent)
		if err != nil {
			b.Fatal(err)
		}
	}
}

<<<<<<< HEAD
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
=======
// BenchmarkNetworkIntentJSONUnmarshal tests JSON unmarshaling performance for NetworkIntent
func BenchmarkNetworkIntentJSONUnmarshal(b *testing.B) {
	// Create JSON data that represents a typical NetworkIntent
	jsonData := []byte(`{
		"apiVersion": "nephio.io/v1alpha1",
		"kind": "NetworkIntent",
		"metadata": {
			"name": "test-intent",
			"namespace": "default",
			"labels": {
				"app": "test",
				"version": "v1"
			}
		},
		"spec": {
			"scalingPriority": "high",
			"targetClusters": ["cluster-1", "cluster-2", "cluster-3"],
			"scalingIntent": {
				"intent_type": "scaling",
				"target": "gnb-simulator",
				"namespace": "ran",
				"replicas": 5
			},
			"deployment": {
				"clusterSelector": {
					"region": "us-west",
					"type": "edge"
				},
				"networkFunctions": [
					{
						"name": "gnb-cu",
						"type": "CNF",
						"version": "1.0.0",
						"config": {
							"maxConnections": 1000,
							"timeout": 30
						},
						"resources": {
							"cpu": "2",
							"memory": "4Gi"
						}
					}
				],
				"replicas": 3
			}
		},
		"status": {
			"phase": "pending",
			"conditions": [
				{
					"type": "Ready",
					"status": "True",
					"lastTransitionTime": "2025-01-01T00:00:00Z",
					"reason": "Initialized",
					"message": "NetworkIntent has been initialized"
				}
			],
			"observedGeneration": 1
		}
	}`)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		var intent NetworkIntent
		err := json.Unmarshal(jsonData, &intent)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkDeploymentSpecJSONRoundTrip tests complete JSON round-trip for DeploymentSpec
func BenchmarkDeploymentSpecJSONRoundTrip(b *testing.B) {
	spec := DeploymentSpec{
		ClusterSelector: map[string]string{
			"region":      "us-west",
			"type":        "edge",
			"environment": "production",
		},
		NetworkFunctions: []NetworkFunction{
			{
				Name:    "gnb-cu",
				Type:    "CNF",
				Version: "1.0.0",
				Resources: NetworkFunctionResources{
					CPU:     "4",
					Memory:  "8Gi",
					Storage: "100Gi",
				},
			},
			{
				Name:    "gnb-du",
				Type:    "CNF",
				Version: "1.0.0",
				Resources: NetworkFunctionResources{
					CPU:    "8",
					Memory: "16Gi",
				},
			},
		},
		Replicas: 5,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Marshal
		data, err := json.Marshal(spec)
		if err != nil {
			b.Fatal(err)
		}

		// Unmarshal
		var result DeploymentSpec
		err = json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
>>>>>>> 6835433495e87288b95961af7173d866977175ff
	}
}