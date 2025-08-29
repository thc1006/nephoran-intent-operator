
package injection



import (

	"fmt"

	"net/http"

	"sync"

	"time"



	"github.com/thc1006/nephoran-intent-operator/pkg/config"

	"github.com/thc1006/nephoran-intent-operator/pkg/git"

	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"

	"github.com/thc1006/nephoran-intent-operator/pkg/security"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"

	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"



	"k8s.io/client-go/tools/record"

)



// Container manages dependency injection for the Nephoran Intent Operator.

type Container struct {

	mu         sync.RWMutex

	singletons map[string]interface{}

	providers  map[string]Provider

	config     *config.Constants

}



// Provider defines a function that creates a dependency.

type Provider func(c *Container) (interface{}, error)



// NewContainer creates a new dependency injection container.

func NewContainer(constants *config.Constants) *Container {

	if constants == nil {

		constants = config.LoadConstants()

	}



	c := &Container{

		singletons: make(map[string]interface{}),

		providers:  make(map[string]Provider),

		config:     constants,

	}



	// Register all providers.

	c.registerProviders()

	return c

}



// RegisterProvider registers a provider function for a dependency type.

func (c *Container) RegisterProvider(name string, provider Provider) {

	c.mu.Lock()

	defer c.mu.Unlock()

	c.providers[name] = provider

}



// RegisterSingleton registers a singleton instance.

func (c *Container) RegisterSingleton(name string, instance interface{}) {

	c.mu.Lock()

	defer c.mu.Unlock()

	c.singletons[name] = instance

}



// Get retrieves a dependency by name, creating it if necessary.

func (c *Container) Get(name string) (interface{}, error) {

	// Check singletons first.

	c.mu.RLock()

	if instance, exists := c.singletons[name]; exists {

		c.mu.RUnlock()

		return instance, nil

	}

	provider, exists := c.providers[name]

	c.mu.RUnlock()



	if !exists {

		return nil, fmt.Errorf("no provider registered for dependency: %s", name)

	}



	// Create the instance.

	instance, err := provider(c)

	if err != nil {

		return nil, fmt.Errorf("failed to create dependency %s: %w", name, err)

	}



	// Store as singleton if it doesn't already exist.

	c.mu.Lock()

	if _, exists := c.singletons[name]; !exists {

		c.singletons[name] = instance

	} else {

		// Return the existing singleton if it was created concurrently.

		instance = c.singletons[name]

	}

	c.mu.Unlock()



	return instance, nil

}



// Factory methods with error handling for internal use.

func (c *Container) getHTTPClient() (*http.Client, error) {

	dep, err := c.Get("http_client")

	if err != nil {

		return nil, err

	}

	return dep.(*http.Client), nil

}



func (c *Container) getGitClient() (git.ClientInterface, error) {

	dep, err := c.Get("git_client")

	if err != nil {

		return nil, err

	}

	return dep.(git.ClientInterface), nil

}



func (c *Container) getLLMClient() (shared.ClientInterface, error) {

	dep, err := c.Get("llm_client")

	if err != nil {

		return nil, err

	}

	return dep.(shared.ClientInterface), nil

}



func (c *Container) getPackageGenerator() (*nephio.PackageGenerator, error) {

	dep, err := c.Get("package_generator")

	if err != nil {

		return nil, err

	}

	return dep.(*nephio.PackageGenerator), nil

}



func (c *Container) getTelecomKnowledgeBase() (*telecom.TelecomKnowledgeBase, error) {

	dep, err := c.Get("telecom_knowledge_base")

	if err != nil {

		return nil, err

	}

	return dep.(*telecom.TelecomKnowledgeBase), nil

}



func (c *Container) getMetricsCollector() (*monitoring.MetricsCollector, error) {

	dep, err := c.Get("metrics_collector")

	if err != nil {

		return nil, err

	}

	return dep.(*monitoring.MetricsCollector), nil

}



// GetLLMSanitizer returns the LLM sanitizer dependency.

func (c *Container) GetLLMSanitizer() (*security.LLMSanitizer, error) {

	dep, err := c.Get("llm_sanitizer")

	if err != nil {

		return nil, err

	}

	return dep.(*security.LLMSanitizer), nil

}



// Dependencies interface methods - these match the interface expected by controllers.

func (c *Container) GetHTTPClient() *http.Client {

	client, err := c.getHTTPClient()

	if err != nil {

		return &http.Client{Timeout: 30 * time.Second}

	}

	return client

}



// GetGitClient performs getgitclient operation.

func (c *Container) GetGitClient() git.ClientInterface {

	client, err := c.getGitClient()

	if err != nil {

		return nil

	}

	return client

}



// GetLLMClient performs getllmclient operation.

func (c *Container) GetLLMClient() shared.ClientInterface {

	client, err := c.getLLMClient()

	if err != nil {

		return nil

	}

	return client

}



// GetPackageGenerator performs getpackagegenerator operation.

func (c *Container) GetPackageGenerator() *nephio.PackageGenerator {

	generator, err := c.getPackageGenerator()

	if err != nil {

		return nil

	}

	return generator

}



// GetTelecomKnowledgeBase performs gettelecomknowledgebase operation.

func (c *Container) GetTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase {

	kb, err := c.getTelecomKnowledgeBase()

	if err != nil {

		return nil

	}

	return kb

}



// GetMetricsCollector performs getmetricscollector operation.

func (c *Container) GetMetricsCollector() *monitoring.MetricsCollector {

	collector, err := c.getMetricsCollector()

	if err != nil {

		return nil

	}

	return collector

}



// GetEventRecorder performs geteventrecorder operation.

func (c *Container) GetEventRecorder() record.EventRecorder {

	c.mu.RLock()

	defer c.mu.RUnlock()

	if recorder, exists := c.singletons["event_recorder"]; exists {

		return recorder.(record.EventRecorder)

	}

	return nil

}



// SetEventRecorder sets the event recorder instance.

func (c *Container) SetEventRecorder(recorder record.EventRecorder) {

	c.RegisterSingleton("event_recorder", recorder)

}



// GetConfig returns the configuration constants.

func (c *Container) GetConfig() *config.Constants {

	return c.config

}



// registerProviders registers all the default providers.

func (c *Container) registerProviders() {

	// Register all providers from providers.go.

	c.RegisterProvider("http_client", HTTPClientProvider)

	c.RegisterProvider("git_client", GitClientProvider)

	c.RegisterProvider("llm_client", LLMClientProvider)

	c.RegisterProvider("package_generator", PackageGeneratorProvider)

	c.RegisterProvider("telecom_knowledge_base", TelecomKnowledgeBaseProvider)

	c.RegisterProvider("metrics_collector", MetricsCollectorProvider)

	c.RegisterProvider("llm_sanitizer", LLMSanitizerProvider)

}

