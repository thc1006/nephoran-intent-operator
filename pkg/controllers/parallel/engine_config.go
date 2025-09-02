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

package parallel

import "time"

// ProcessingEngineConfig holds configuration for the parallel processing engine.
type ProcessingEngineConfig struct {
	// Worker pool configurations.
	IntentWorkers     int `json:"intentWorkers"`
	LLMWorkers        int `json:"llmWorkers"`
	RAGWorkers        int `json:"ragWorkers"`
	ResourceWorkers   int `json:"resourceWorkers"`
	ManifestWorkers   int `json:"manifestWorkers"`
	GitOpsWorkers     int `json:"gitopsWorkers"`
	DeploymentWorkers int `json:"deploymentWorkers"`

	// Queue configurations.
	MaxQueueSize int           `json:"maxQueueSize"`
	QueueTimeout time.Duration `json:"queueTimeout"`

	// Performance tuning.
	MaxConcurrentIntents int           `json:"maxConcurrentIntents"`
	ProcessingTimeout    time.Duration `json:"processingTimeout"`
	HealthCheckInterval  time.Duration `json:"healthCheckInterval"`

	// Resource limits.
	MaxMemoryPerWorker int64   `json:"maxMemoryPerWorker"`
	MaxCPUPerWorker    float64 `json:"maxCpuPerWorker"`

	// Backpressure settings.
	BackpressureEnabled   bool    `json:"backpressureEnabled"`
	BackpressureThreshold float64 `json:"backpressureThreshold"`

	// Load balancing.
	LoadBalancingStrategy string `json:"loadBalancingStrategy"`

	// Retry and circuit breaker.
	MaxRetries            int  `json:"maxRetries"`
	CircuitBreakerEnabled bool `json:"circuitBreakerEnabled"`

	// Monitoring.
	MetricsEnabled  bool `json:"metricsEnabled"`
	DetailedLogging bool `json:"detailedLogging"`
}

// getDefaultProcessingEngineConfig returns default configuration.
func getDefaultProcessingEngineConfig() *ProcessingEngineConfig {
	return &ProcessingEngineConfig{
		IntentWorkers:         5,
		LLMWorkers:            3,
		RAGWorkers:            3,
		ResourceWorkers:       4,
		ManifestWorkers:       4,
		GitOpsWorkers:         2,
		DeploymentWorkers:     3,
		MaxQueueSize:          1000,
		QueueTimeout:          5 * time.Minute,
		MaxConcurrentIntents:  100,
		ProcessingTimeout:     10 * time.Minute,
		HealthCheckInterval:   30 * time.Second,
		MaxMemoryPerWorker:    512 * 1024 * 1024, // 512MB
		MaxCPUPerWorker:       0.5,               // 50%
		BackpressureEnabled:   true,
		BackpressureThreshold: 0.8, // 80%
		LoadBalancingStrategy: "round_robin",
		MaxRetries:            3,
		CircuitBreakerEnabled: true,
		MetricsEnabled:        true,
		DetailedLogging:       false,
	}
}
