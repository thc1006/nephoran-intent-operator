//go:build ml && !test

package ml

import (
	"testing"
)

func TestNewOptimizationEngine(t *testing.T) {
	tests := []struct {
		name    string
		config  *OptimizationConfig
		promURL string
		wantErr bool
	}{
		{
			name: "empty prometheus URL",
			config: &OptimizationConfig{
				MLModelConfig: &MLModelConfig{
					TrafficPrediction:    &ModelConfig{Enabled: false},
					ResourceOptimization: &ModelConfig{Enabled: false},
					AnomalyDetection:     &ModelConfig{Enabled: false},
					IntentClassification: &ModelConfig{Enabled: false},
				},
			},
			promURL: "",
			wantErr: true,
		},
		{
			name: "valid configuration",
			config: &OptimizationConfig{
				PrometheusURL: "http://prometheus:9090",
				MLModelConfig: &MLModelConfig{
					TrafficPrediction:    &ModelConfig{Enabled: true, Algorithm: "linear"},
					ResourceOptimization: &ModelConfig{Enabled: true},
					AnomalyDetection:     &ModelConfig{Enabled: true},
					IntentClassification: &ModelConfig{Enabled: true},
				},
			},
			promURL: "http://prometheus:9090",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewOptimizationEngine(tt.config, tt.promURL)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewOptimizationEngine() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
