//go:build go1.24

package generics

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// Example: Complete Network Intent Processing Pipeline
func ExampleNetworkIntentPipeline() {
	// Define network intent data structure
	type NetworkIntent struct {
		ID       string
		Type     string
		Region   string
		Priority int
		Config   map[string]any
	}

	// Define processing result
	type ProcessingResult struct {
		IntentID string
		Status   string
		Message  string
		Duration time.Duration
	}

	// 1. Configuration Management
	configManager := NewConfigManager(map[string]any{
		"max_concurrency": 10,
		"timeout":         30,
	})

	// 2. Validation Pipeline
	validator := NewValidationBuilder[NetworkIntent]().
		Required("id", func(ni NetworkIntent) any { return ni.ID }).
		OneOf("type", []string{"5G-Core", "RAN", "Edge"},
			func(ni NetworkIntent) string { return ni.Type }).
		OneOf("region", []string{"us-east", "us-west", "eu-central"},
			func(ni NetworkIntent) string { return ni.Region }).
		Range("priority", 1, 10,
			func(ni NetworkIntent) int64 { return int64(ni.Priority) }).
		Build()

	// 3. Event Bus for Processing Events
	eventBus := NewEventBus[ProcessingResult](EventBusConfig{
		BufferSize:  1000,
		WorkerCount: 5,
	})

	// Subscribe to processing events
	eventBus.Subscribe(func(ctx context.Context, event Event[ProcessingResult]) Result[bool, error] {
		fmt.Printf("Processing event: %s - %s\n", event.Data.IntentID, event.Data.Status)
		return Ok[bool, error](true)
	})

	// 4. Processing Pipeline with Middleware
	type ProcessingRequest struct {
		Intent NetworkIntent
		Config map[string]any
	}

	chain := NewMiddlewareChain[ProcessingRequest, ProcessingResult]().
		Add(LoggingMiddleware[ProcessingRequest, ProcessingResult](
			NewJSONLogger[ProcessingRequest, ProcessingResult](
				func(log string) { fmt.Println("LOG:", log) },
			),
		)).
		Add(TimeoutMiddleware[ProcessingRequest, ProcessingResult](30 * time.Second))

	// Final processing handler
	processor := func(ctx context.Context, req ProcessingRequest) Result[ProcessingResult, error] {
		start := time.Now()

		// Simulate processing
		time.Sleep(10 * time.Millisecond)

		result := ProcessingResult{
			IntentID: req.Intent.ID,
			Status:   "completed",
			Message:  fmt.Sprintf("Processed %s intent for %s", req.Intent.Type, req.Intent.Region),
			Duration: time.Since(start),
		}

		return Ok[ProcessingResult, error](result)
	}

	// 5. Process intents
	intents := []NetworkIntent{
		{ID: "intent-1", Type: "5G-Core", Region: "us-east", Priority: 5},
		{ID: "intent-2", Type: "RAN", Region: "us-west", Priority: 3},
		{ID: "intent-3", Type: "Edge", Region: "eu-central", Priority: 7},
	}

	ctx := context.Background()

	for _, intent := range intents {
		// Validate intent
		validationResult := validator(intent)
		if !validationResult.Valid {
			fmt.Printf("Validation failed for %s: %s\n", intent.ID, validationResult.Error())
			continue
		}

		// Load configuration
		config := configManager.Load(ctx, "processing")
		if config.IsErr() {
			fmt.Printf("Config load failed: %v\n", config.Error())
			continue
		}

		// Process with middleware
		request := ProcessingRequest{
			Intent: intent,
			Config: config.Value().(map[string]any),
		}

		result := chain.Execute(ctx, request, processor)
		if result.IsOk() {
			// Publish processing event
			event := NewEvent(intent.ID, "processing.completed", result.Value())
			eventBus.Publish(event)
		} else {
			fmt.Printf("Processing failed for %s: %v\n", intent.ID, result.Error())
		}
	}

	// Clean up
	eventBus.Close()
	configManager.Close()

	// Output:
	// Processing event: intent-1 - completed
	// Processing event: intent-2 - completed
	// Processing event: intent-3 - completed
}

// Example: Client Pool with Circuit Breaker
func ExampleClientPoolWithCircuitBreaker() {
	type APIRequest struct {
		Action string
		Data   map[string]any
	}

	type APIResponse struct {
		Success bool
		Result  any
		Error   string
	}

	// Create HTTP clients
	config1 := HTTPClientConfig[APIRequest, APIResponse]{
		BaseURL:  "https://api1.example.com",
		Endpoint: "/process",
		Method:   "POST",
		Timeout:  10 * time.Second,
	}

	config2 := HTTPClientConfig[APIRequest, APIResponse]{
		BaseURL:  "https://api2.example.com",
		Endpoint: "/process",
		Method:   "POST",
		Timeout:  10 * time.Second,
	}

	client1 := NewHTTPClient(config1)
	client2 := NewHTTPClient(config2)

	// Create client pool
	pool := NewClientPool[APIRequest, APIResponse](client1, client2)

	// Process requests with automatic failover
	requests := []APIRequest{
		{Action: "deploy", Data: map[string]any{"type": "5G-Core"}},
		{Action: "scale", Data: map[string]any{"replicas": 3}},
		{Action: "monitor", Data: map[string]any{"interval": "30s"}},
	}

	ctx := context.Background()

	for _, request := range requests {
		// Try with failover
		result := pool.ExecuteWithFailover(ctx, request)

		if result.IsOk() {
			response := result.Value()
			fmt.Printf("Request %s succeeded: %v\n", request.Action, response.Success)
		} else {
			fmt.Printf("Request %s failed: %v\n", request.Action, result.Error())
		}
	}

	// Health check all clients
	healthResults := pool.HealthCheck(ctx)
	if healthResults.IsOk() {
		healths := healthResults.Value()
		for i, healthy := range healths {
			fmt.Printf("Client %d healthy: %v\n", i+1, healthy)
		}
	}

	pool.Close()
}

// Example: Event-Driven Microservices with Aggregation
func ExampleEventDrivenMicroservices() {
	// Define event types
	type DeploymentEvent struct {
		ServiceID string
		Action    string
		Status    string
		Timestamp time.Time
		Details   map[string]any
	}

	type ServiceMetrics struct {
		TotalDeployments int
		SuccessRate      float64
		AverageLatency   time.Duration
		FailureReasons   map[string]int
	}

	// Create event buses
	deploymentBus := NewEventBus[DeploymentEvent](EventBusConfig{
		BufferSize:  5000,
		WorkerCount: 10,
	})

	metricsBus := NewEventBus[ServiceMetrics](EventBusConfig{
		BufferSize:  1000,
		WorkerCount: 3,
	})

	// Set up event routing
	router := NewEventRouter[DeploymentEvent](func(event Event[DeploymentEvent]) string {
		return event.Data.Action // Route by action type
	})

	startBus := NewEventBus[DeploymentEvent](EventBusConfig{BufferSize: 1000})
	completeBus := NewEventBus[DeploymentEvent](EventBusConfig{BufferSize: 1000})
	failBus := NewEventBus[DeploymentEvent](EventBusConfig{BufferSize: 1000})

	router.AddRoute("start", startBus)
	router.AddRoute("complete", completeBus)
	router.AddRoute("fail", failBus)

	// Subscribe to specific event types
	startBus.Subscribe(func(ctx context.Context, event Event[DeploymentEvent]) Result[bool, error] {
		fmt.Printf("Deployment started: %s\n", event.Data.ServiceID)
		return Ok[bool, error](true)
	})

	completeBus.Subscribe(func(ctx context.Context, event Event[DeploymentEvent]) Result[bool, error] {
		fmt.Printf("Deployment completed: %s\n", event.Data.ServiceID)
		return Ok[bool, error](true)
	})

	failBus.Subscribe(func(ctx context.Context, event Event[DeploymentEvent]) Result[bool, error] {
		fmt.Printf("Deployment failed: %s - %v\n", event.Data.ServiceID, event.Data.Details)
		return Ok[bool, error](true)
	})

	// Set up metrics aggregation
	aggregator := NewEventAggregator(
		deploymentBus,
		metricsBus,
		func(events []Event[DeploymentEvent]) Result[Event[ServiceMetrics], error] {
			if len(events) == 0 {
				return Err[Event[ServiceMetrics], error](fmt.Errorf("no events to aggregate"))
			}

			total := len(events)
			successful := 0
			failures := make(map[string]int)
			var totalLatency time.Duration

			for _, event := range events {
				if event.Data.Status == "success" {
					successful++
				} else {
					reason := "unknown"
					if r, ok := event.Data.Details["reason"]; ok {
						reason = r.(string)
					}
					failures[reason]++
				}

				if latency, ok := event.Data.Details["latency"]; ok {
					if dur, ok := latency.(time.Duration); ok {
						totalLatency += dur
					}
				}
			}

			metrics := ServiceMetrics{
				TotalDeployments: total,
				SuccessRate:      float64(successful) / float64(total),
				AverageLatency:   totalLatency / time.Duration(total),
				FailureReasons:   failures,
			}

			return Ok[Event[ServiceMetrics], error](
				NewEvent("metrics", "aggregation", metrics),
			)
		},
		30*time.Second, // Aggregate every 30 seconds
	)

	// Subscribe to metrics
	metricsBus.Subscribe(func(ctx context.Context, event Event[ServiceMetrics]) Result[bool, error] {
		metrics := event.Data
		fmt.Printf("Metrics - Total: %d, Success Rate: %.2f%%, Avg Latency: %v\n",
			metrics.TotalDeployments, metrics.SuccessRate*100, metrics.AverageLatency)
		return Ok[bool, error](true)
	})

	// Generate sample events
	events := []DeploymentEvent{
		{ServiceID: "service-1", Action: "start", Status: "pending", Timestamp: time.Now()},
		{ServiceID: "service-1", Action: "complete", Status: "success", Timestamp: time.Now(), Details: map[string]any{"latency": 2 * time.Second}},
		{ServiceID: "service-2", Action: "start", Status: "pending", Timestamp: time.Now()},
		{ServiceID: "service-2", Action: "fail", Status: "error", Timestamp: time.Now(), Details: map[string]any{"reason": "timeout", "latency": 30 * time.Second}},
		{ServiceID: "service-3", Action: "start", Status: "pending", Timestamp: time.Now()},
		{ServiceID: "service-3", Action: "complete", Status: "success", Timestamp: time.Now(), Details: map[string]any{"latency": 1 * time.Second}},
	}

	// Publish events
	for _, eventData := range events {
		event := NewEvent(eventData.ServiceID, eventData.Action, eventData)

		// Route to appropriate bus
		router.Route(event)

		// Also publish to main bus for aggregation
		deploymentBus.Publish(event)

		time.Sleep(100 * time.Millisecond) // Small delay between events
	}

	// Wait for aggregation
	time.Sleep(1 * time.Second)

	// Clean up
	deploymentBus.Close()
	metricsBus.Close()
	startBus.Close()
	completeBus.Close()
	failBus.Close()
	aggregator.Close()
}

// Example: Advanced Validation with Cross-Field Dependencies
func ExampleAdvancedValidation() {
	type NetworkConfig struct {
		Environment string // "dev", "staging", "prod"
		Region      string
		Resources   struct {
			CPU     int
			Memory  int
			Storage int
		}
		HighAvailability bool
		BackupConfig     *struct {
			Enabled   bool
			Frequency string
			Retention int
		}
	}

	// Build complex validator
	validator := NewValidationBuilder[NetworkConfig]().
		Required("environment", func(nc NetworkConfig) any { return nc.Environment }).
		OneOf("environment", []string{"dev", "staging", "prod"},
			func(nc NetworkConfig) string { return nc.Environment }).
		Required("region", func(nc NetworkConfig) any { return nc.Region }).
		Range("cpu", 1, 64, func(nc NetworkConfig) int64 { return int64(nc.Resources.CPU) }).
		Range("memory", 1, 256, func(nc NetworkConfig) int64 { return int64(nc.Resources.Memory) }).
		Range("storage", 10, 10000, func(nc NetworkConfig) int64 { return int64(nc.Resources.Storage) }).
		// Cross-field validation
		Custom(func(nc NetworkConfig) ValidationResult {
			result := NewValidationResult()

			// Production environments require more resources
			if nc.Environment == "prod" {
				if nc.Resources.CPU < 4 {
					result.AddError("resources.cpu", "production requires minimum 4 CPU cores", "prod_resources", nc.Resources.CPU)
				}
				if nc.Resources.Memory < 16 {
					result.AddError("resources.memory", "production requires minimum 16GB memory", "prod_resources", nc.Resources.Memory)
				}
				if !nc.HighAvailability {
					result.AddError("high_availability", "production must have high availability enabled", "prod_ha", nc.HighAvailability)
				}
			}

			// High availability requires backup configuration
			if nc.HighAvailability && (nc.BackupConfig == nil || !nc.BackupConfig.Enabled) {
				result.AddError("backup_config", "high availability requires backup configuration", "ha_backup", nc.BackupConfig)
			}

			// Backup retention validation
			if nc.BackupConfig != nil && nc.BackupConfig.Enabled {
				if nc.BackupConfig.Retention < 7 {
					result.AddError("backup_config.retention", "backup retention must be at least 7 days", "backup_retention", nc.BackupConfig.Retention)
				}
			}

			return result
		}).
		Build()

	// Test configurations
	configs := []NetworkConfig{
		{
			Environment: "dev",
			Region:      "us-east",
			Resources:   struct{ CPU, Memory, Storage int }{CPU: 2, Memory: 4, Storage: 50},
		},
		{
			Environment:      "prod",
			Region:           "us-west",
			Resources:        struct{ CPU, Memory, Storage int }{CPU: 8, Memory: 32, Storage: 500},
			HighAvailability: true,
			BackupConfig: &struct {
				Enabled   bool
				Frequency string
				Retention int
			}{Enabled: true, Frequency: "daily", Retention: 30},
		},
		{
			Environment:      "prod",
			Region:           "eu-central",
			Resources:        struct{ CPU, Memory, Storage int }{CPU: 2, Memory: 8, Storage: 100}, // Insufficient for prod
			HighAvailability: false,                                                               // Missing HA for prod
		},
	}

	for i, config := range configs {
		result := validator(config)
		fmt.Printf("Config %d (%s): Valid=%v\n", i+1, config.Environment, result.Valid)

		if !result.Valid {
			for _, err := range result.Errors {
				fmt.Printf("  Error - %s: %s\n", err.Field, err.Message)
			}
		}
	}
}

// Integration test demonstrating all components working together
func TestFullIntegration(t *testing.T) {
	// This would be a comprehensive integration test
	// combining all the generic components in a realistic scenario

	t.Run("NetworkIntentPipeline", func(t *testing.T) {
		ExampleNetworkIntentPipeline()
	})

	t.Run("ClientPoolWithCircuitBreaker", func(t *testing.T) {
		// Note: This would normally require mock HTTP servers
		// ExampleClientPoolWithCircuitBreaker()
	})

	t.Run("EventDrivenMicroservices", func(t *testing.T) {
		ExampleEventDrivenMicroservices()
	})

	t.Run("AdvancedValidation", func(t *testing.T) {
		ExampleAdvancedValidation()
	})
}

// Performance test demonstrating the zero-overhead nature of generics
func BenchmarkGenericVsInterface(b *testing.B) {
	// Generic version
	b.Run("Generic", func(b *testing.B) {
		result := Ok[int, error](42)
		transform := func(x int) int { return x * 2 }

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = Map(result, transform)
		}
	})

	// Interface version (for comparison)
	b.Run("Interface", func(b *testing.B) {
		var result interface{} = 42
		transform := func(x interface{}) interface{} {
			return x.(int) * 2
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = transform(result)
		}
	})
}
