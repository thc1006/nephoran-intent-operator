package oran

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSMOManager(t *testing.T) {
	tests := []struct {
		name          string
		config        *SMOConfig
		expectedError bool
	}{
		{
			name: "valid configuration",
			config: &SMOConfig{
				Endpoint: "https://smo.example.com",
				Username: "testuser",
				Password: "testpass",
				Timeout:  30 * time.Second,
			},
			expectedError: false,
		},
		{
			name:          "nil configuration",
			config:        nil,
			expectedError: true,
		},
		{
			name: "configuration with defaults",
			config: &SMOConfig{
				Endpoint: "https://smo.example.com",
				Username: "testuser",
				Password: "testpass",
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewSMOManager(tt.config)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Nil(t, manager)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, manager)
				assert.NotNil(t, manager.policyManager)
				assert.NotNil(t, manager.serviceRegistry)
				assert.NotNil(t, manager.orchestrator)

				// Verify defaults are set
				if tt.config.APIVersion == "" {
					assert.Equal(t, "v1", manager.config.APIVersion)
				}
				if tt.config.RetryCount == 0 {
					assert.Equal(t, 3, manager.config.RetryCount)
				}
			}
		})
	}
}

func TestSMOManager_Start(t *testing.T) {
	// Mock SMO server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	tests := []struct {
		name          string
		config        *SMOConfig
		expectedError bool
	}{
		{
			name: "successful start",
			config: &SMOConfig{
				Endpoint:        server.URL,
				Username:        "testuser",
				Password:        "testpass",
				HealthCheckPath: "/health",
			},
			expectedError: false,
		},
		{
			name: "connection failure",
			config: &SMOConfig{
				Endpoint:        "http://nonexistent.example.com",
				Username:        "testuser",
				Password:        "testpass",
				HealthCheckPath: "/health",
				Timeout:         1 * time.Second,
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager, err := NewSMOManager(tt.config)
			require.NoError(t, err)

			ctx := context.Background()
			err = manager.Start(ctx)

			if tt.expectedError {
				assert.Error(t, err)
				assert.False(t, manager.IsConnected())
			} else {
				assert.NoError(t, err)
				assert.True(t, manager.IsConnected())
			}

			manager.Stop()
		})
	}
}

func TestPolicyManager_CreatePolicy(t *testing.T) {
	// Mock SMO server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/policies":
			if r.Method == "POST" {
				var policy A1Policy
				err := json.NewDecoder(r.Body).Decode(&policy)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(policy)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &SMOConfig{
		Endpoint:        server.URL,
		Username:        "testuser",
		Password:        "testpass",
		HealthCheckPath: "/health",
	}

	manager, err := NewSMOManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop()

	// Add a policy type for testing
	policyType := &A1PolicyType{
		ID:          "qos-policy-type",
		Name:        "QoS Policy Type",
		Version:     "1.0.0",
		Description: "Policy type for QoS management",
		Schema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"qci": map[string]interface{}{
					"type": "integer",
				},
			},
		},
		CreatedAt: time.Now(),
	}
	manager.policyManager.policyTypes[policyType.ID] = policyType

	tests := []struct {
		name          string
		policy        *A1Policy
		expectedError bool
	}{
		{
			name: "valid policy creation",
			policy: &A1Policy{
				ID:          "test-policy-1",
				TypeID:      "qos-policy-type",
				Version:     "1.0.0",
				Description: "Test QoS policy",
				Data: map[string]interface{}{
					"qci": 7,
				},
				TargetRICs: []string{"ric-1", "ric-2"},
				Metadata: map[string]string{
					"priority": "high",
				},
			},
			expectedError: false,
		},
		{
			name: "policy with unknown type",
			policy: &A1Policy{
				ID:          "test-policy-2",
				TypeID:      "unknown-policy-type",
				Version:     "1.0.0",
				Description: "Test policy with unknown type",
				Data: map[string]interface{}{
					"param": "value",
				},
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.policyManager.CreatePolicy(ctx, tt.policy)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify policy was stored
				stored, err := manager.policyManager.GetPolicy(tt.policy.ID)
				assert.NoError(t, err)
				assert.Equal(t, tt.policy.ID, stored.ID)
				assert.Equal(t, tt.policy.TypeID, stored.TypeID)
				assert.Equal(t, "ACTIVE", stored.Status)
				assert.NotZero(t, stored.CreatedAt)
			}
		})
	}
}

func TestPolicyManager_UpdatePolicy(t *testing.T) {
	// Mock SMO server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/policies/test-policy-1":
			if r.Method == "PUT" {
				w.WriteHeader(http.StatusOK)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &SMOConfig{
		Endpoint:        server.URL,
		Username:        "testuser",
		Password:        "testpass",
		HealthCheckPath: "/health",
	}

	manager, err := NewSMOManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop()

	// Create a test policy
	policy := &A1Policy{
		ID:          "test-policy-1",
		TypeID:      "qos-policy-type",
		Version:     "1.0.0",
		Description: "Original description",
		Status:      "ACTIVE",
		Data: map[string]interface{}{
			"qci": 7,
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	manager.policyManager.policies[policy.ID] = policy

	tests := []struct {
		name          string
		policyID      string
		updates       map[string]interface{}
		expectedError bool
		validateFunc  func(*testing.T, *A1Policy)
	}{
		{
			name:     "update description",
			policyID: "test-policy-1",
			updates: map[string]interface{}{
				"description": "Updated description",
			},
			expectedError: false,
			validateFunc: func(t *testing.T, p *A1Policy) {
				assert.Equal(t, "Updated description", p.Description)
			},
		},
		{
			name:     "update data",
			policyID: "test-policy-1",
			updates: map[string]interface{}{
				"data": map[string]interface{}{
					"qci": 9,
				},
			},
			expectedError: false,
			validateFunc: func(t *testing.T, p *A1Policy) {
				data := p.Data
				assert.Equal(t, float64(9), data["qci"]) // JSON unmarshals numbers as float64
			},
		},
		{
			name:     "update status",
			policyID: "test-policy-1",
			updates: map[string]interface{}{
				"status": "INACTIVE",
			},
			expectedError: false,
			validateFunc: func(t *testing.T, p *A1Policy) {
				assert.Equal(t, "INACTIVE", p.Status)
			},
		},
		{
			name:     "update nonexistent policy",
			policyID: "nonexistent-policy",
			updates: map[string]interface{}{
				"description": "This should fail",
			},
			expectedError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.policyManager.UpdatePolicy(ctx, tt.policyID, tt.updates)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				if tt.validateFunc != nil {
					updated, err := manager.policyManager.GetPolicy(tt.policyID)
					require.NoError(t, err)
					tt.validateFunc(t, updated)
				}
			}
		})
	}
}

func TestPolicyManager_SubscribeToPolicyEvents(t *testing.T) {
	config := &SMOConfig{
		Endpoint: "https://mock-smo.example.com",
		Username: "testuser",
		Password: "testpass",
	}

	manager, err := NewSMOManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	// Test event subscription
	eventReceived := false
	subscription, err := manager.policyManager.SubscribeToPolicyEvents(
		ctx,
		"test-policy-1",
		[]string{"CREATED", "UPDATED"},
		func(event *PolicyEvent) {
			eventReceived = true
			assert.Equal(t, "test-policy-1", event.PolicyID)
		},
	)

	assert.NoError(t, err)
	assert.NotNil(t, subscription)
	assert.Equal(t, "test-policy-1", subscription.PolicyID)
	assert.Equal(t, "nephoran-intent-operator", subscription.Subscriber)
	assert.Contains(t, subscription.Events, "CREATED")
	assert.Contains(t, subscription.Events, "UPDATED")

	// Simulate receiving an event
	event := &PolicyEvent{
		ID:        "event-1",
		Type:      "CREATED",
		PolicyID:  "test-policy-1",
		RIC:       "ric-1",
		Timestamp: time.Now(),
		Data:      map[string]interface{}{},
		Severity:  "INFO",
	}

	// Manually trigger callback
	if callback := manager.policyManager.eventCallbacks[subscription.ID]; callback != nil {
		callback(event)
	}

	assert.True(t, eventReceived)
}

func TestServiceRegistry_RegisterService(t *testing.T) {
	// Mock SMO server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/services":
			if r.Method == "POST" {
				var service ServiceInstance
				err := json.NewDecoder(r.Body).Decode(&service)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(service)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &SMOConfig{
		Endpoint:        server.URL,
		Username:        "testuser",
		Password:        "testpass",
		HealthCheckPath: "/health",
	}

	manager, err := NewSMOManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop()

	tests := []struct {
		name          string
		service       *ServiceInstance
		expectedError bool
	}{
		{
			name: "register RIC service",
			service: &ServiceInstance{
				ID:       "ric-1",
				Name:     "Near-RT RIC",
				Type:     "RIC",
				Version:  "1.0.0",
				Endpoint: "http://ric-1.example.com:8080",
				Capabilities: []string{
					"policy-management",
					"xapp-management",
				},
				Metadata: map[string]string{
					"region": "us-west-1",
				},
				HealthCheck: &HealthCheckConfig{
					Enabled:  true,
					Path:     "/health",
					Interval: 30 * time.Second,
					Timeout:  5 * time.Second,
				},
			},
			expectedError: false,
		},
		{
			name: "register xApp service",
			service: &ServiceInstance{
				ID:       "xapp-kpi-monitor",
				Name:     "KPI Monitor xApp",
				Type:     "xApp",
				Version:  "2.1.0",
				Endpoint: "http://xapp-kpi.example.com:8000",
				Capabilities: []string{
					"kpi-monitoring",
					"analytics",
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.serviceRegistry.RegisterService(ctx, tt.service)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify service was registered
				assert.Equal(t, "REGISTERED", tt.service.Status)
				assert.NotZero(t, tt.service.RegisteredAt)
				assert.NotZero(t, tt.service.LastHeartbeat)

				// Check if service is in registry
				manager.serviceRegistry.mu.RLock()
				registered := manager.serviceRegistry.registeredServices[tt.service.ID]
				manager.serviceRegistry.mu.RUnlock()

				assert.NotNil(t, registered)
				assert.Equal(t, tt.service.ID, registered.ID)
			}
		})
	}
}

func TestServiceOrchestrator_DeployRApp(t *testing.T) {
	// Mock SMO server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.WriteHeader(http.StatusOK)
		case "/api/v1/rapps":
			if r.Method == "POST" {
				var rApp RAppInstance
				err := json.NewDecoder(r.Body).Decode(&rApp)
				if err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(rApp)
			}
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &SMOConfig{
		Endpoint:        server.URL,
		Username:        "testuser",
		Password:        "testpass",
		HealthCheckPath: "/health",
	}

	manager, err := NewSMOManager(config)
	require.NoError(t, err)

	ctx := context.Background()
	err = manager.Start(ctx)
	require.NoError(t, err)
	defer manager.Stop()

	tests := []struct {
		name          string
		rApp          *RAppInstance
		expectedError bool
	}{
		{
			name: "deploy analytics rApp",
			rApp: &RAppInstance{
				ID:      "analytics-rapp-1",
				Name:    "Analytics rApp",
				Type:    "analytics",
				Version: "1.0.0",
				Image:   "registry.example.com/analytics-rapp:1.0.0",
				Configuration: map[string]interface{}{
					"data_sources":  []string{"ric-1", "ric-2"},
					"output_format": "json",
				},
				Resources: &ResourceRequirements{
					CPU:     "1000m",
					Memory:  "2Gi",
					Storage: "10Gi",
				},
				Dependencies: []string{"data-collector"},
				Lifecycle: &LifecycleConfig{
					PreStart: []LifecycleHook{
						{
							Name:   "validate-config",
							Type:   "HTTP",
							Action: "validate",
							Params: map[string]interface{}{
								"endpoint": "http://config-validator:8080/validate",
							},
							Timeout: 30 * time.Second,
						},
					},
					PostStart: []LifecycleHook{
						{
							Name:   "register-with-discovery",
							Type:   "HTTP",
							Action: "register",
							Params: map[string]interface{}{
								"endpoint": "http://service-discovery:8080/register",
							},
							Timeout: 10 * time.Second,
						},
					},
				},
			},
			expectedError: false,
		},
		{
			name: "deploy optimization rApp",
			rApp: &RAppInstance{
				ID:      "optimization-rapp-1",
				Name:    "Optimization rApp",
				Type:    "optimization",
				Version: "2.0.0",
				Image:   "registry.example.com/optimization-rapp:2.0.0",
				Configuration: map[string]interface{}{
					"algorithm":  "genetic",
					"iterations": 1000,
				},
				Resources: &ResourceRequirements{
					CPU:     "2000m",
					Memory:  "4Gi",
					Storage: "20Gi",
				},
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.orchestrator.DeployRApp(ctx, tt.rApp)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Verify rApp was deployed
				assert.Equal(t, "RUNNING", tt.rApp.Status)
				assert.NotZero(t, tt.rApp.CreatedAt)
				assert.NotNil(t, tt.rApp.StartedAt)

				// Check if rApp is in orchestrator
				manager.orchestrator.mu.RLock()
				deployed := manager.orchestrator.rApps[tt.rApp.ID]
				manager.orchestrator.mu.RUnlock()

				assert.NotNil(t, deployed)
				assert.Equal(t, tt.rApp.ID, deployed.ID)
				assert.Equal(t, "RUNNING", deployed.Status)
			}
		})
	}
}

func TestSMOClient_HTTPMethods(t *testing.T) {
	// Mock server for testing HTTP methods
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check authentication
		username, password, ok := r.BasicAuth()
		if !ok || username != "testuser" || password != "testpass" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Check custom headers
		if r.Header.Get("X-Custom-Header") != "custom-value" {
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch r.Method {
		case "GET":
			response := map[string]interface{}{
				"method": "GET",
				"path":   r.URL.Path,
			}
			json.NewEncoder(w).Encode(response)
		case "POST":
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			response := map[string]interface{}{
				"method": "POST",
				"body":   body,
			}
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(response)
		case "PUT":
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			response := map[string]interface{}{
				"method": "PUT",
				"body":   body,
			}
			json.NewEncoder(w).Encode(response)
		case "DELETE":
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
	defer server.Close()

	client := &SMOClient{
		baseURL: server.URL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		auth: &AuthConfig{
			Type:     "basic",
			Username: "testuser",
			Password: "testpass",
		},
		headers: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}

	ctx := context.Background()

	t.Run("GET request", func(t *testing.T) {
		var result map[string]interface{}
		err := client.get(ctx, server.URL+"/test", &result)

		assert.NoError(t, err)
		assert.Equal(t, "GET", result["method"])
		assert.Equal(t, "/test", result["path"])
	})

	t.Run("POST request", func(t *testing.T) {
		body := map[string]interface{}{
			"key": "value",
		}
		var result map[string]interface{}
		err := client.post(ctx, server.URL+"/test", body, &result)

		assert.NoError(t, err)
		assert.Equal(t, "POST", result["method"])
		assert.NotNil(t, result["body"])
	})

	t.Run("PUT request", func(t *testing.T) {
		body := map[string]interface{}{
			"updated": "value",
		}
		var result map[string]interface{}
		err := client.put(ctx, server.URL+"/test", body, &result)

		assert.NoError(t, err)
		assert.Equal(t, "PUT", result["method"])
		assert.NotNil(t, result["body"])
	})

	t.Run("DELETE request", func(t *testing.T) {
		err := client.delete(ctx, server.URL+"/test")
		assert.NoError(t, err)
	})
}

// Integration tests

func TestSMOManager_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	// This would be an integration test with a real SMO instance
	// For now, we'll simulate the complete workflow

	// Mock SMO server with complete API
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/health":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]string{"status": "healthy"})

		case r.URL.Path == "/api/v1/policies" && r.Method == "POST":
			var policy A1Policy
			json.NewDecoder(r.Body).Decode(&policy)
			policy.Status = "ACTIVE"
			policy.CreatedAt = time.Now()
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(policy)

		case r.URL.Path == "/api/v1/services" && r.Method == "POST":
			var service ServiceInstance
			json.NewDecoder(r.Body).Decode(&service)
			service.Status = "REGISTERED"
			service.RegisteredAt = time.Now()
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(service)

		case r.URL.Path == "/api/v1/rapps" && r.Method == "POST":
			var rApp RAppInstance
			json.NewDecoder(r.Body).Decode(&rApp)
			rApp.Status = "RUNNING"
			rApp.CreatedAt = time.Now()
			now := time.Now()
			rApp.StartedAt = &now
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(rApp)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	config := &SMOConfig{
		Endpoint:        server.URL,
		Username:        "integrationuser",
		Password:        "integrationpass",
		HealthCheckPath: "/health",
	}

	manager, err := NewSMOManager(config)
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("complete SMO integration workflow", func(t *testing.T) {
		// 1. Start SMO manager
		err := manager.Start(ctx)
		assert.NoError(t, err)
		assert.True(t, manager.IsConnected())

		// 2. Register a policy type
		policyType := &A1PolicyType{
			ID:          "integration-policy-type",
			Name:        "Integration Test Policy",
			Version:     "1.0.0",
			Description: "Policy type for integration testing",
			Schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"priority": map[string]interface{}{
						"type": "integer",
					},
				},
			},
			CreatedAt: time.Now(),
		}
		manager.policyManager.policyTypes[policyType.ID] = policyType

		// 3. Create a policy
		policy := &A1Policy{
			ID:          "integration-policy-1",
			TypeID:      "integration-policy-type",
			Version:     "1.0.0",
			Description: "Integration test policy",
			Data: map[string]interface{}{
				"priority": 1,
			},
			TargetRICs: []string{"integration-ric"},
		}

		err = manager.policyManager.CreatePolicy(ctx, policy)
		assert.NoError(t, err)

		// 4. Register a service
		service := &ServiceInstance{
			ID:       "integration-service",
			Name:     "Integration Test Service",
			Type:     "RIC",
			Version:  "1.0.0",
			Endpoint: "http://integration-ric:8080",
			Capabilities: []string{
				"policy-management",
			},
		}

		err = manager.serviceRegistry.RegisterService(ctx, service)
		assert.NoError(t, err)

		// 5. Deploy an rApp
		rApp := &RAppInstance{
			ID:      "integration-rapp",
			Name:    "Integration Test rApp",
			Type:    "analytics",
			Version: "1.0.0",
			Image:   "integration/rapp:latest",
			Configuration: map[string]interface{}{
				"mode": "test",
			},
			Resources: &ResourceRequirements{
				CPU:     "500m",
				Memory:  "1Gi",
				Storage: "5Gi",
			},
		}

		err = manager.orchestrator.DeployRApp(ctx, rApp)
		assert.NoError(t, err)

		// 6. Verify all components are working
		policies := manager.policyManager.ListPolicies()
		assert.Len(t, policies, 1)
		assert.Equal(t, "integration-policy-1", policies[0].ID)

		// 7. Stop manager
		manager.Stop()
		assert.False(t, manager.IsConnected())
	})
}

// Benchmark tests

func BenchmarkPolicyManager_CreatePolicy(b *testing.B) {
	config := &SMOConfig{
		Endpoint: "https://benchmark-smo.example.com",
		Username: "benchuser",
		Password: "benchpass",
	}

	manager, err := NewSMOManager(config)
	if err != nil {
		b.Fatal(err)
	}

	// Add policy type
	policyType := &A1PolicyType{
		ID:        "bench-policy-type",
		CreatedAt: time.Now(),
	}
	manager.policyManager.policyTypes[policyType.ID] = policyType

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		policy := &A1Policy{
			ID:          fmt.Sprintf("bench-policy-%d", i),
			TypeID:      "bench-policy-type",
			Version:     "1.0.0",
			Description: "Benchmark policy",
			Data: map[string]interface{}{
				"value": i,
			},
		}

		// Note: This will fail due to no actual SMO server, but we're benchmarking
		// the manager logic, not the network call
		manager.policyManager.CreatePolicy(ctx, policy)
	}
}

func BenchmarkServiceRegistry_RegisterService(b *testing.B) {
	config := &SMOConfig{
		Endpoint: "https://benchmark-smo.example.com",
		Username: "benchuser",
		Password: "benchpass",
	}

	manager, err := NewSMOManager(config)
	if err != nil {
		b.Fatal(err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service := &ServiceInstance{
			ID:       fmt.Sprintf("bench-service-%d", i),
			Name:     fmt.Sprintf("Benchmark Service %d", i),
			Type:     "RIC",
			Version:  "1.0.0",
			Endpoint: fmt.Sprintf("http://bench-service-%d:8080", i),
		}

		// Note: This will fail due to no actual SMO server
		manager.serviceRegistry.RegisterService(ctx, service)
	}
}
