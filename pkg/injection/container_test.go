package injection

import (
	"net/http"
	"os"
	"testing"
	"time"

	"k8s.io/client-go/tools/record"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
)

func TestNewContainer(t *testing.T) {
	// Test with nil constants (should load defaults)
	container := NewContainer(nil)
	if container == nil {
		t.Fatal("NewContainer returned nil")
	}
	if container.config == nil {
		t.Fatal("Container config is nil")
	}

	// Test with provided constants
	constants := &config.Constants{
		DefaultTimeout: 30 * time.Second,
		MaxRetries:     3,
	}
	container = NewContainer(constants)
	if container.config != constants {
		t.Error("Container config not set correctly")
	}
}

func TestContainerProviderRegistration(t *testing.T) {
	container := NewContainer(nil)

	// Test that providers are registered
	expectedProviders := []string{
		"http_client",
		"git_client",
		"llm_client",
		"package_generator",
		"telecom_knowledge_base",
		"metrics_collector",
		"llm_sanitizer",
	}

	for _, providerName := range expectedProviders {
		_, err := container.Get(providerName)
		if err != nil {
			t.Errorf("Provider %s not registered or failed to create: %v", providerName, err)
		}
	}
}

func TestContainerSingletonBehavior(t *testing.T) {
	container := NewContainer(nil)

	// Test that multiple calls return the same instance
	client1 := container.GetHTTPClient()
	client2 := container.GetHTTPClient()

	if client1 != client2 {
		t.Error("HTTP client should be a singleton")
	}

	// Test metrics collector singleton
	metrics1 := container.GetMetricsCollector()
	metrics2 := container.GetMetricsCollector()

	if metrics1 != metrics2 {
		t.Error("Metrics collector should be a singleton")
	}
}

func TestContainerDependencyInterfaces(t *testing.T) {
	container := NewContainer(nil)

	// Test Dependencies interface compliance
	var deps Dependencies = container

	// Test that all methods work
	httpClient := deps.GetHTTPClient()
	if httpClient == nil {
		t.Error("GetHTTPClient returned nil")
	}

	gitClient := deps.GetGitClient()
	// Git client may be nil if no config provided, which is OK

	llmClient := deps.GetLLMClient()
	// LLM client may be nil if no URL provided, which is OK

	packageGen := deps.GetPackageGenerator()
	if packageGen == nil {
		t.Error("GetPackageGenerator returned nil")
	}

	telecomKB := deps.GetTelecomKnowledgeBase()
	if telecomKB == nil {
		t.Error("GetTelecomKnowledgeBase returned nil")
	}

	metricsCollector := deps.GetMetricsCollector()
	if metricsCollector == nil {
		t.Error("GetMetricsCollector returned nil")
	}

	// Event recorder should be nil until set
	eventRecorder := deps.GetEventRecorder()
	if eventRecorder != nil {
		t.Error("GetEventRecorder should be nil before being set")
	}
}

func TestContainerEventRecorderManagement(t *testing.T) {
	container := NewContainer(nil)

	// Create a fake event recorder
	fakeRecorder := &record.FakeRecorder{}

	// Initially should be nil
	if container.GetEventRecorder() != nil {
		t.Error("Event recorder should be nil initially")
	}

	// Set the recorder
	container.SetEventRecorder(fakeRecorder)

	// Should now return the set recorder
	recorder := container.GetEventRecorder()
	if recorder != fakeRecorder {
		t.Error("Event recorder not returned correctly")
	}
}

func TestContainerConfigAccess(t *testing.T) {
	constants := &config.Constants{
		DefaultTimeout: 45 * time.Second,
		MaxRetries:     5,
	}
	container := NewContainer(constants)

	config := container.GetConfig()
	if config != constants {
		t.Error("GetConfig did not return the correct constants")
	}
}

func TestContainerConcurrentAccess(t *testing.T) {
	container := NewContainer(nil)

	// Test concurrent access to same dependency
	done := make(chan bool, 10)
	
	for i := 0; i < 10; i++ {
		go func() {
			defer func() { done <- true }()
			
			// Each goroutine gets dependencies
			client := container.GetHTTPClient()
			if client == nil {
				t.Error("Concurrent access returned nil HTTP client")
			}
			
			metrics := container.GetMetricsCollector()
			if metrics == nil {
				t.Error("Concurrent access returned nil metrics collector")
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestContainerCustomProvider(t *testing.T) {
	container := NewContainer(nil)

	// Register a custom provider
	customProviderCalled := false
	container.RegisterProvider("custom_test", func(c *Container) (interface{}, error) {
		customProviderCalled = true
		return "custom_value", nil
	})

	// Get the custom dependency
	result, err := container.Get("custom_test")
	if err != nil {
		t.Fatalf("Failed to get custom dependency: %v", err)
	}

	if result != "custom_value" {
		t.Error("Custom provider did not return expected value")
	}

	if !customProviderCalled {
		t.Error("Custom provider was not called")
	}

	// Test singleton behavior for custom provider
	result2, err := container.Get("custom_test")
	if err != nil {
		t.Fatalf("Failed to get custom dependency second time: %v", err)
	}

	if result != result2 {
		t.Error("Custom dependency should be singleton")
	}
}

func TestContainerProviderError(t *testing.T) {
	container := NewContainer(nil)

	// Register a provider that returns an error
	container.RegisterProvider("error_test", func(c *Container) (interface{}, error) {
		return nil, http.ErrBodyNotAllowed
	})

	// Should return the error
	_, err := container.Get("error_test")
	if err == nil {
		t.Error("Expected error from failing provider")
	}
}

func TestContainerUnregisteredProvider(t *testing.T) {
	container := NewContainer(nil)

	// Try to get non-existent provider
	_, err := container.Get("non_existent")
	if err == nil {
		t.Error("Expected error for non-existent provider")
	}
}

func TestContainerWithEnvironmentVariables(t *testing.T) {
	// Set up test environment variables
	oldGitURL := os.Getenv("GIT_REPO_URL")
	oldLLMURL := os.Getenv("LLM_PROCESSOR_URL")
	
	defer func() {
		// Restore original values
		if oldGitURL == "" {
			os.Unsetenv("GIT_REPO_URL")
		} else {
			os.Setenv("GIT_REPO_URL", oldGitURL)
		}
		if oldLLMURL == "" {
			os.Unsetenv("LLM_PROCESSOR_URL")
		} else {
			os.Setenv("LLM_PROCESSOR_URL", oldLLMURL)
		}
	}()

	// Set test values
	os.Setenv("GIT_REPO_URL", "https://github.com/test/test.git")
	os.Setenv("LLM_PROCESSOR_URL", "http://localhost:8080")

	container := NewContainer(nil)

	// Git client should be created with the URL
	gitClient := container.GetGitClient()
	if gitClient == nil {
		t.Error("Git client should be created when GIT_REPO_URL is set")
	}

	// LLM client should be created with the URL
	llmClient := container.GetLLMClient()
	if llmClient == nil {
		t.Error("LLM client should be created when LLM_PROCESSOR_URL is set")
	}
}

func TestContainerLLMSanitizerWithConfig(t *testing.T) {
	constants := &config.Constants{
		MaxInputLength:  1000,
		MaxOutputLength: 2000,
		AllowedDomains:  []string{"example.com"},
		BlockedKeywords: []string{"forbidden"},
	}
	
	container := NewContainer(constants)

	sanitizer, err := container.GetLLMSanitizer()
	if err != nil {
		t.Fatalf("Failed to get LLM sanitizer: %v", err)
	}

	if sanitizer == nil {
		t.Error("LLM sanitizer should not be nil")
	}
}