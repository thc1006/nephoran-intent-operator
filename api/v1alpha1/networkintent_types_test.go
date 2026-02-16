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
			APIVersion: "nephoran.io/v1alpha1",
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
			Source:     "user",
			IntentType: "scaling",
			Target:     "gnb-simulator",
			Namespace:  "ran",
			Replicas:   5,
			ScalingParameters: &ScalingParameters{
				Replicas: 5,
				AutoscalingPolicy: &AutoscalingPolicy{
					MinReplicas: 2,
					MaxReplicas: 10,
					MetricThresholds: []MetricThreshold{
						{
							Type:  "CPU",
							Value: 80,
						},
					},
				},
			},
			NetworkParameters: &NetworkParameters{
				NetworkSliceID: "slice-1",
				QoSProfile: &QoSProfile{
					Priority:        1,
					MaximumDataRate: "100Mbps",
				},
			},
		},
		Status: NetworkIntentStatus{
			Phase: "pending",
			Conditions: []metav1.Condition{
				{
					Type:               "Ready",
					Status:             metav1.ConditionTrue,
					LastTransitionTime: metav1.Now(),
					Reason:             "Initialized",
					Message:            "NetworkIntent has been initialized",
				},
			},
			ObservedReplicas: 5,
			ReadyReplicas:    5,
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := json.Marshal(intent)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkNetworkIntentJSONUnmarshal tests JSON unmarshaling performance for NetworkIntent
func BenchmarkNetworkIntentJSONUnmarshal(b *testing.B) {
	// Create JSON data that represents a typical NetworkIntent
	jsonData := []byte(`{
		"apiVersion": "nephoran.io/v1alpha1",
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
			"source": "user",
			"intentType": "scaling",
			"target": "gnb-simulator",
			"namespace": "ran",
			"replicas": 5,
			"scalingParameters": {
				"replicas": 5,
				"autoscalingPolicy": {
					"minReplicas": 2,
					"maxReplicas": 10,
					"metricThresholds": [
						{
							"type": "CPU",
							"value": 80
						}
					]
				}
			},
			"networkParameters": {
				"networkSliceId": "slice-1",
				"qosProfile": {
					"priority": 1,
					"maximumDataRate": "100Mbps"
				}
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
			"observedReplicas": 5,
			"readyReplicas": 5
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

// BenchmarkScalingParametersJSONRoundTrip tests complete JSON round-trip for ScalingParameters
func BenchmarkScalingParametersJSONRoundTrip(b *testing.B) {
	params := ScalingParameters{
		Replicas: 5,
		AutoscalingPolicy: &AutoscalingPolicy{
			MinReplicas: 2,
			MaxReplicas: 10,
			MetricThresholds: []MetricThreshold{
				{
					Type:  "CPU",
					Value: 80,
				},
				{
					Type:  "Memory",
					Value: 70,
				},
			},
		},
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Marshal
		data, err := json.Marshal(params)
		if err != nil {
			b.Fatal(err)
		}

		// Unmarshal
		var result ScalingParameters
		err = json.Unmarshal(data, &result)
		if err != nil {
			b.Fatal(err)
		}
	}
}