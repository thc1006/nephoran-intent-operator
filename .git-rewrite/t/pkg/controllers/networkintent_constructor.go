package controllers

import (
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

// NetworkIntentReconcilerConfig holds the configuration for creating a NetworkIntentReconciler
type NetworkIntentReconcilerConfig struct {
	Client        client.Client
	Scheme        *runtime.Scheme
	EventRecorder record.EventRecorder
	Config        *config.Config
}

// NewNetworkIntentReconciler creates a new NetworkIntentReconciler with proper initialization
func NewNetworkIntentReconciler(config NetworkIntentReconcilerConfig) *NetworkIntentReconciler {
	// Initialize Git client if configuration is available
	var gitClient git.ClientInterface
	if config.Config.GitRepoURL != "" {
		gitClient = git.NewClient(
			config.Config.GitRepoURL,
			config.Config.GitBranch,
			config.Config.GitToken,
		)
	}

	// Initialize LLM client
	var llmClient llm.ClientInterface
	if config.Config.LLMProcessorURL != "" {
		llmClient = llm.NewClient(config.Config.LLMProcessorURL)
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.Config.LLMProcessorTimeout,
	}

	return &NetworkIntentReconciler{
		Client:          config.Client,
		Scheme:          config.Scheme,
		GitClient:       gitClient,
		LLMClient:       llmClient,
		LLMProcessorURL: config.Config.LLMProcessorURL,
		HTTPClient:      httpClient,
		EventRecorder:   config.EventRecorder,

		// Retry configuration
		MaxRetries: 3,
		RetryDelay: time.Second * 30,

		// GitOps configuration
		GitRepoURL:    config.Config.GitRepoURL,
		GitBranch:     config.Config.GitBranch,
		GitDeployPath: "networkintents", // Default deployment path
	}
}
