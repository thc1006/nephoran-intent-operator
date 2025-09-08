package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

func TestProviderIntegrationWithSchemaValidation(t *testing.T) {
	// Setup schema validator
	schemaPath := getTestSchemaPath(t)
	validator, err := ingest.NewValidator(schemaPath)
	require.NoError(t, err, "Failed to create validator")

	tests := []struct {
		name           string
		providerType   ProviderType
		config         *Config
		input          string
		expectValid    bool
		expectMinReqs  bool // Check minimum required fields
		setupEnv       map[string]string
		cleanupEnv     []string
	}{
		{
			name:         "OFFLINE Provider with Simple Scaling Intent",
			providerType: ProviderTypeOffline,
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:         "Scale O-DU to 3 replicas in oran-odu namespace",
			expectValid:   true,
			expectMinReqs: true,
		},
		{
			name:         "OFFLINE Provider with Complex Network Function Scaling",
			providerType: ProviderTypeOffline,
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:         "Urgent: increase AMF instances to 5 in core5g namespace for high priority traffic",
			expectValid:   true,
			expectMinReqs: true,
		},
		{
			name:         "OFFLINE Provider with Minimal Input",
			providerType: ProviderTypeOffline,
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:         "scale up",
			expectValid:   true,
			expectMinReqs: true,
		},
		{
			name:         "OFFLINE Provider with E2 Simulator Request",
			providerType: ProviderTypeOffline,
			config: &Config{
				Type:       ProviderTypeOffline,
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:         "Deploy 2 E2 simulators in test namespace",
			expectValid:   true,
			expectMinReqs: true,
		},
		{
			name:         "OpenAI Provider with Mock Configuration",
			providerType: ProviderTypeOpenAI,
			config: &Config{
				Type:       ProviderTypeOpenAI,
				APIKey:     "sk-test-api-key-12345678901234567890",
				Model:      "gpt-4",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:         "Scale RIC xApp to 4 instances",
			expectValid:   true,
			expectMinReqs: true,
			setupEnv:      map[string]string{"LLM_PROVIDER": "OPENAI"},
			cleanupEnv:    []string{"LLM_PROVIDER"},
		},
		{
			name:         "Anthropic Provider with Mock Configuration",
			providerType: ProviderTypeAnthropic,
			config: &Config{
				Type:       ProviderTypeAnthropic,
				APIKey:     "sk-ant-test-api-key-12345678901234567890123456789012345678901234567890",
				Model:      "claude-3-haiku-20240307",
				Timeout:    30 * time.Second,
				MaxRetries: 3,
			},
			input:         "Configure SMF with 6 replicas in core5g",
			expectValid:   true,
			expectMinReqs: true,
			setupEnv:      map[string]string{"LLM_PROVIDER": "ANTHROPIC"},
			cleanupEnv:    []string{"LLM_PROVIDER"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup environment variables
			for key, value := range tt.setupEnv {
				os.Setenv(key, value)
			}
			defer func() {
				for _, key := range tt.cleanupEnv {
					os.Unsetenv(key)
				}
			}()

			// Create provider through factory
			factory := NewFactory()
			provider, err := factory.CreateProvider(tt.config)
			require.NoError(t, err, "Failed to create provider")
			defer provider.Close()

			// Process intent
			ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
			defer cancel()

			response, err := provider.ProcessIntent(ctx, tt.input)
			require.NoError(t, err, "Failed to process intent")
			require.NotNil(t, response, "Response should not be nil")
			require.NotNil(t, response.JSON, "Response JSON should not be nil")

			// Validate against schema
			intent, err := validator.ValidateBytes(response.JSON)
			if tt.expectValid {
				assert.NoError(t, err, "Intent should validate against schema")
				assert.NotNil(t, intent, "Validated intent should not be nil")

				// Check minimum required fields
				if tt.expectMinReqs {
					assert.NotEmpty(t, intent.IntentType, "intent_type should not be empty")
					assert.NotEmpty(t, intent.Target, "target should not be empty")
					assert.NotEmpty(t, intent.Namespace, "namespace should not be empty")
					assert.Greater(t, intent.Replicas, 0, "replicas should be greater than 0")
				}
			} else {
				assert.Error(t, err, "Intent should not validate against schema")
			}

			// Verify response metadata
			assert.NotEmpty(t, response.Metadata.Provider, "Provider name should not be empty")
			assert.True(t, response.Metadata.ProcessingTime >= 0, "Processing time should be non-negative")

			// Verify JSON structure (should be parseable)
			var jsonData map[string]interface{}
			err = json.Unmarshal(response.JSON, &jsonData)
			assert.NoError(t, err, "Response JSON should be valid JSON")
		})
	}
}

func TestProviderFactoryWithEnvironmentVariables(t *testing.T) {
	tests := []struct {
		name         string
		envVars      map[string]string
		expectedType ProviderType
		shouldError  bool
	}{
		{
			name:         "Default to OFFLINE when no environment set",
			envVars:      map[string]string{},
			expectedType: ProviderTypeOffline,
			shouldError:  false,
		},
		{
			name:         "OFFLINE provider from environment",
			envVars:      map[string]string{"LLM_PROVIDER": "OFFLINE"},
			expectedType: ProviderTypeOffline,
			shouldError:  false,
		},
		{
			name:         "OpenAI provider from environment with API key",
			envVars:      map[string]string{"LLM_PROVIDER": "OPENAI", "LLM_API_KEY": "sk-test-key-12345678901234567890"},
			expectedType: ProviderTypeOpenAI,
			shouldError:  false,
		},
		{
			name:         "Anthropic provider from environment with API key",
			envVars:      map[string]string{"LLM_PROVIDER": "ANTHROPIC", "LLM_API_KEY": "sk-ant-test-key-12345678901234567890123456789012345678901234567890"},
			expectedType: ProviderTypeAnthropic,
			shouldError:  false,
		},
		{
			name:         "Invalid provider type",
			envVars:      map[string]string{"LLM_PROVIDER": "INVALID"},
			expectedType: ProviderTypeOffline, // Should default
			shouldError:  false,
		},
		{
			name:         "OpenAI without API key should error",
			envVars:      map[string]string{"LLM_PROVIDER": "OPENAI"},
			expectedType: ProviderTypeOpenAI,
			shouldError:  true,
		},
		{
			name:         "Anthropic without API key should error",
			envVars:      map[string]string{"LLM_PROVIDER": "ANTHROPIC"},
			expectedType: ProviderTypeAnthropic,
			shouldError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original environment
			originalEnv := make(map[string]string)
			for key := range tt.envVars {
				originalEnv[key] = os.Getenv(key)
			}

			// Set test environment
			for key, value := range tt.envVars {
				os.Setenv(key, value)
			}

			// Clean up original values
			defer func() {
				for key, originalValue := range originalEnv {
					if originalValue == "" {
						os.Unsetenv(key)
					} else {
						os.Setenv(key, originalValue)
					}
				}
			}()

			// Test ConfigFromEnvironment
			config, err := ConfigFromEnvironment()
			if tt.shouldError {
				// If we expect an error, it might come from ConfigFromEnvironment or CreateProvider
				if err == nil {
					// Try creating the provider
					factory := NewFactory()
					_, err = factory.CreateProvider(config)
				}
				assert.Error(t, err, "Should error for configuration: %+v", tt.envVars)
			} else {
				assert.NoError(t, err, "Should not error for configuration: %+v", tt.envVars)
				assert.Equal(t, tt.expectedType, config.Type, "Provider type should match expected")

				// Test creating the provider
				factory := NewFactory()
				provider, err := factory.CreateProvider(config)
				assert.NoError(t, err, "Should be able to create provider")
				if provider != nil {
					provider.Close()
				}
			}
		})
	}
}

func TestOfflineProviderDetailedParsing(t *testing.T) {
	// Setup schema validator
	schemaPath := getTestSchemaPath(t)
	validator, err := ingest.NewValidator(schemaPath)
	require.NoError(t, err, "Failed to create validator")

	config := &Config{
		Type:       ProviderTypeOffline,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}

	provider, err := NewOfflineProvider(config)
	require.NoError(t, err, "Failed to create offline provider")
	defer provider.Close()

	tests := []struct {
		name           string
		input          string
		expectedTarget string
		expectedNS     string
		expectedReps   int
		expectedType   string
		shouldValidate bool
	}{
		{
			name:           "O-RAN DU Scaling",
			input:          "Scale distributed unit to 4 replicas in oran-odu namespace",
			expectedTarget: "o-du",
			expectedNS:     "oran-odu",
			expectedReps:   4,
			expectedType:   "scaling",
			shouldValidate: true,
		},
		{
			name:           "AMF Core Function",
			input:          "Increase AMF instances to 8 in core5g",
			expectedTarget: "amf",
			expectedNS:     "core5g",
			expectedReps:   8,
			expectedType:   "scaling",
			shouldValidate: true,
		},
		{
			name:           "RIC xApp Deployment",
			input:          "Deploy RIC application with 2 instances",
			expectedTarget: "ric",
			expectedNS:     "default",
			expectedReps:   2,
			expectedType:   "deployment",
			shouldValidate: true,
		},
		{
			name:           "E2 Simulator",
			input:          "Start 3 E2SIM simulators in test namespace",
			expectedTarget: "e2-simulator",
			expectedNS:     "test",
			expectedReps:   3,
			expectedType:   "deployment",
			shouldValidate: true,
		},
		{
			name:           "Number Words",
			input:          "Scale SMF to five replicas",
			expectedTarget: "smf",
			expectedNS:     "default",
			expectedReps:   5,
			expectedType:   "scaling",
			shouldValidate: true,
		},
		{
			name:           "High Priority Request",
			input:          "Urgent: scale UPF to 10 instances immediately",
			expectedTarget: "upf",
			expectedNS:     "default",
			expectedReps:   10,
			expectedType:   "scaling",
			shouldValidate: true,
		},
		{
			name:           "Minimal Input",
			input:          "scale workload",
			expectedTarget: "default-workload",
			expectedNS:     "default",
			expectedReps:   1,
			expectedType:   "scaling",
			shouldValidate: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			
			response, err := provider.ProcessIntent(ctx, tt.input)
			require.NoError(t, err, "Failed to process intent")
			require.NotNil(t, response, "Response should not be nil")
			
			// Validate against schema
			intent, err := validator.ValidateBytes(response.JSON)
			if tt.shouldValidate {
				assert.NoError(t, err, "Intent should validate against schema")
				assert.NotNil(t, intent, "Validated intent should not be nil")
				
				// Check specific fields
				assert.Equal(t, tt.expectedType, intent.IntentType, "Intent type should match")
				assert.Equal(t, tt.expectedTarget, intent.Target, "Target should match")
				assert.Equal(t, tt.expectedNS, intent.Namespace, "Namespace should match")
				assert.Equal(t, tt.expectedReps, intent.Replicas, "Replicas should match")
			} else {
				assert.Error(t, err, "Intent should not validate against schema")
			}
			
			// Verify provider metadata
			assert.Equal(t, "OFFLINE", response.Metadata.Provider, "Provider should be OFFLINE")
			assert.Equal(t, "rule-based-v1", response.Metadata.Model, "Model should be rule-based-v1")
			assert.Greater(t, response.Metadata.Confidence, 0.0, "Confidence should be greater than 0")
			assert.LessOrEqual(t, response.Metadata.Confidence, 1.0, "Confidence should be less than or equal to 1")
		})
	}
}

func TestSchemaValidationErrorHandling(t *testing.T) {
	// Setup schema validator
	schemaPath := getTestSchemaPath(t)
	validator, err := ingest.NewValidator(schemaPath)
	require.NoError(t, err, "Failed to create validator")

	tests := []struct {
		name        string
		jsonData    string
		shouldError bool
		errorMsg    string
	}{
		{
			name: "Valid Intent",
			jsonData: `{
				"intent_type": "scaling",
				"target": "test-workload",
				"namespace": "default",
				"replicas": 3
			}`,
			shouldError: false,
		},
		{
			name: "Missing Required Field - intent_type",
			jsonData: `{
				"target": "test-workload",
				"namespace": "default",
				"replicas": 3
			}`,
			shouldError: true,
			errorMsg:    "intent_type",
		},
		{
			name: "Missing Required Field - target",
			jsonData: `{
				"intent_type": "scaling",
				"namespace": "default",
				"replicas": 3
			}`,
			shouldError: true,
			errorMsg:    "target",
		},
		{
			name: "Invalid Intent Type",
			jsonData: `{
				"intent_type": "invalid_type",
				"target": "test-workload",
				"namespace": "default",
				"replicas": 3
			}`,
			shouldError: true,
			errorMsg:    "value must be one of",
		},
		{
			name: "Invalid Replica Count - Too Low",
			jsonData: `{
				"intent_type": "scaling",
				"target": "test-workload",
				"namespace": "default",
				"replicas": 0
			}`,
			shouldError: true,
			errorMsg:    "minimum",
		},
		{
			name: "Invalid Replica Count - Too High",
			jsonData: `{
				"intent_type": "scaling",
				"target": "test-workload",
				"namespace": "default",
				"replicas": 101
			}`,
			shouldError: true,
			errorMsg:    "maximum",
		},
		{
			name: "Empty Target",
			jsonData: `{
				"intent_type": "scaling",
				"target": "",
				"namespace": "default",
				"replicas": 3
			}`,
			shouldError: true,
			errorMsg:    "minLength",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := validator.ValidateBytes([]byte(tt.jsonData))
			if tt.shouldError {
				assert.Error(t, err, "Should error for invalid JSON")
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg, "Error message should contain expected text")
				}
			} else {
				assert.NoError(t, err, "Should not error for valid JSON")
			}
		})
	}
}

func TestProviderCleanup(t *testing.T) {
	config := &Config{
		Type:       ProviderTypeOffline,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}

	factory := NewFactory()
	provider, err := factory.CreateProvider(config)
	require.NoError(t, err, "Failed to create provider")

	// Test provider info
	info := provider.GetProviderInfo()
	assert.Equal(t, "OFFLINE", info.Name)
	assert.False(t, info.RequiresAuth)
	assert.NotEmpty(t, info.SupportedFeatures)

	// Test config validation
	err = provider.ValidateConfig()
	assert.NoError(t, err, "Config validation should pass")

	// Test cleanup
	err = provider.Close()
	assert.NoError(t, err, "Provider cleanup should succeed")
}

func TestConcurrentProviderAccess(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping concurrent test in short mode")
	}

	// Setup schema validator
	schemaPath := getTestSchemaPath(t)
	validator, err := ingest.NewValidator(schemaPath)
	require.NoError(t, err, "Failed to create validator")

	config := &Config{
		Type:       ProviderTypeOffline,
		Timeout:    30 * time.Second,
		MaxRetries: 3,
	}

	factory := NewFactory()
	provider, err := factory.CreateProvider(config)
	require.NoError(t, err, "Failed to create provider")
	defer provider.Close()

	// Test concurrent access
	const numGoroutines = 10
	const numRequests = 5

	results := make(chan error, numGoroutines*numRequests)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			for j := 0; j < numRequests; j++ {
				input := fmt.Sprintf("Scale workload-%d-%d to 2 replicas", goroutineID, j)
				
				ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
				
				response, err := provider.ProcessIntent(ctx, input)
				cancel()
				
				if err != nil {
					results <- fmt.Errorf("goroutine %d, request %d: %w", goroutineID, j, err)
					continue
				}

				// Validate the response
				_, err = validator.ValidateBytes(response.JSON)
				if err != nil {
					results <- fmt.Errorf("goroutine %d, request %d validation: %w", goroutineID, j, err)
					continue
				}

				results <- nil
			}
		}(i)
	}

	// Collect results
	for i := 0; i < numGoroutines*numRequests; i++ {
		err := <-results
		assert.NoError(t, err, "Concurrent request should succeed")
	}
}

// Helper function to find the test schema path
func getTestSchemaPath(t *testing.T) string {
	// Try multiple possible paths
	paths := []string{
		"../../../docs/contracts/intent.schema.json",
		"../../docs/contracts/intent.schema.json", 
		"../docs/contracts/intent.schema.json",
		"docs/contracts/intent.schema.json",
	}

	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			absPath, err := filepath.Abs(path)
			require.NoError(t, err, "Failed to get absolute path")
			return absPath
		}
	}

	// If not found, try to find from current working directory
	cwd, err := os.Getwd()
	require.NoError(t, err, "Failed to get current working directory")

	// Navigate up to find the project root
	for dir := cwd; dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		schemaPath := filepath.Join(dir, "docs", "contracts", "intent.schema.json")
		if _, err := os.Stat(schemaPath); err == nil {
			return schemaPath
		}
	}

	t.Fatal("Could not find intent.schema.json file")
	return ""
}