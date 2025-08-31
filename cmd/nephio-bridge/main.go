package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
	"github.com/nephio-project/nephoran-intent-operator/pkg/config"
	"github.com/nephio-project/nephoran-intent-operator/pkg/controllers"
	"github.com/nephio-project/nephoran-intent-operator/pkg/git"
	"github.com/nephio-project/nephoran-intent-operator/pkg/llm"
	"github.com/nephio-project/nephoran-intent-operator/pkg/monitoring"
	"github.com/nephio-project/nephoran-intent-operator/pkg/nephio"
	"github.com/nephio-project/nephoran-intent-operator/pkg/shared"
	"github.com/nephio-project/nephoran-intent-operator/pkg/telecom"
)

var (
	scheme = runtime.NewScheme()

	setupLog = ctrl.Log.WithName("setup")
)

// llmClientAdapter adapts llm.Client to shared.ClientInterface.
// This adapter implements the modern shared.ClientInterface while bridging
// to the legacy llm.Client implementation.
type llmClientAdapter struct {
	client *llm.Client
}

// Compile-time interface compliance check
var _ shared.ClientInterface = (*llmClientAdapter)(nil)

// ProcessRequest implements shared.ClientInterface.ProcessRequest
func (a *llmClientAdapter) ProcessRequest(ctx context.Context, request *shared.LLMRequest) (*shared.LLMResponse, error) {
	// Convert modern LLMRequest to legacy prompt
	prompt := a.convertRequestToPrompt(request)
	
	// Call legacy method
	result, err := a.client.ProcessIntent(ctx, prompt)
	if err != nil {
		return nil, err
	}
	
	// Convert legacy response to modern LLMResponse
	return &shared.LLMResponse{
		ID:      "legacy-" + fmt.Sprintf("%d", time.Now().UnixNano()),
		Content: result,
		Model:   request.Model,
		Usage: shared.TokenUsage{
			PromptTokens:     a.EstimateTokens(prompt),
			CompletionTokens: a.EstimateTokens(result),
			TotalTokens:      a.EstimateTokens(prompt) + a.EstimateTokens(result),
		},
		Created: time.Now(),
	}, nil
}

// ProcessStreamingRequest implements shared.ClientInterface.ProcessStreamingRequest
func (a *llmClientAdapter) ProcessStreamingRequest(ctx context.Context, request *shared.LLMRequest) (<-chan *shared.StreamingChunk, error) {
	chunks := make(chan *shared.StreamingChunk, 1)
	
	go func() {
		defer close(chunks)
		
		// Convert to legacy format and process
		prompt := a.convertRequestToPrompt(request)
		result, err := a.client.ProcessIntent(ctx, prompt)
		
		if err != nil {
			chunks <- &shared.StreamingChunk{
				Error: &shared.LLMError{
					Code:    "processing_error",
					Message: err.Error(),
					Type:    "client_error",
				},
			}
			return
		}
		
		// Send result as single chunk (simulated streaming)
		chunks <- &shared.StreamingChunk{
			ID:        "legacy-chunk-" + fmt.Sprintf("%d", time.Now().UnixNano()),
			Content:   result,
			Delta:     result,
			Done:      true,
			IsLast:    true,
			Timestamp: time.Now(),
		}
	}()
	
	return chunks, nil
}

// HealthCheck implements shared.ClientInterface.HealthCheck
func (a *llmClientAdapter) HealthCheck(ctx context.Context) error {
	// Perform a simple health check by making a minimal request
	_, err := a.client.ProcessIntent(ctx, "health check")
	return err
}

// GetStatus implements shared.ClientInterface.GetStatus
func (a *llmClientAdapter) GetStatus() shared.ClientStatus {
	// Simple status check - assume healthy if client exists
	if a.client != nil {
		return shared.ClientStatusHealthy
	}
	return shared.ClientStatusUnavailable
}

// GetModelCapabilities implements shared.ClientInterface.GetModelCapabilities
// Note: This returns ModelCapabilities directly, not a pointer with error
func (a *llmClientAdapter) GetModelCapabilities() shared.ModelCapabilities {
	return shared.ModelCapabilities{
		SupportsStreaming:    false, // Legacy client doesn't support true streaming
		SupportsSystemPrompt: true,
		SupportsChatFormat:   true,
		SupportsChat:         true,
		SupportsFunction:     false,
		MaxTokens:            8192,
		CostPerToken:         0.001,
		SupportedMimeTypes:   []string{"text/plain", "application/json"},
		ModelVersion:         "legacy-1.0",
		Features:             make(map[string]interface{}),
	}
}

// GetEndpoint implements shared.ClientInterface.GetEndpoint
func (a *llmClientAdapter) GetEndpoint() string {
	return a.client.url
}

// Close implements shared.ClientInterface.Close
func (a *llmClientAdapter) Close() error {
	a.client.Shutdown()
	return nil
}

// Legacy compatibility methods (kept for backward compatibility)

// ProcessIntent provides backward compatibility with legacy interface
func (a *llmClientAdapter) ProcessIntent(ctx context.Context, prompt string) (string, error) {
	return a.client.ProcessIntent(ctx, prompt)
}

// ProcessIntentStream provides backward compatibility with legacy interface
func (a *llmClientAdapter) ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *shared.StreamingChunk) error {
	// For now, fall back to non-streaming
	result, err := a.client.ProcessIntent(ctx, prompt)
	if err != nil {
		return err
	}
	
	if chunks != nil {
		chunks <- &shared.StreamingChunk{
			Content:   result,
			IsLast:    true,
			Timestamp: time.Now(),
		}
		close(chunks)
	}
	return nil
}

// Helper methods

// convertRequestToPrompt converts modern LLMRequest to legacy prompt string
func (a *llmClientAdapter) convertRequestToPrompt(request *shared.LLMRequest) string {
	if len(request.Messages) == 0 {
		return ""
	}
	
	// Simple conversion: concatenate all message content
	var prompt strings.Builder
	for _, msg := range request.Messages {
		if msg.Role == "system" {
			prompt.WriteString("System: ")
		} else if msg.Role == "user" {
			prompt.WriteString("User: ")
		} else if msg.Role == "assistant" {
			prompt.WriteString("Assistant: ")
		}
		prompt.WriteString(msg.Content)
		prompt.WriteString("\n")
	}
	return strings.TrimSpace(prompt.String())
}

// EstimateTokens estimates token count for a given text
func (a *llmClientAdapter) EstimateTokens(text string) int {
	// Simple estimation: roughly 4 characters per token
	return len(text) / 4
}

// dependencyImpl implements the Dependencies interface.

type dependencyImpl struct {
	gitClient git.ClientInterface

	llmClient shared.ClientInterface

	packageGen *nephio.PackageGenerator

	httpClient *http.Client

	eventRecorder record.EventRecorder
}

// GetGitClient performs getgitclient operation.

func (d *dependencyImpl) GetGitClient() git.ClientInterface {

	return d.gitClient

}

// GetLLMClient performs getllmclient operation.

func (d *dependencyImpl) GetLLMClient() shared.ClientInterface {

	return d.llmClient

}

// GetPackageGenerator performs getpackagegenerator operation.

func (d *dependencyImpl) GetPackageGenerator() *nephio.PackageGenerator {

	return d.packageGen

}

// GetHTTPClient performs gethttpclient operation.

func (d *dependencyImpl) GetHTTPClient() *http.Client {

	return d.httpClient

}

// GetEventRecorder performs geteventrecorder operation.

func (d *dependencyImpl) GetEventRecorder() record.EventRecorder {

	return d.eventRecorder

}

// GetMetricsCollector returns the metrics collector (placeholder implementation).

func (d *dependencyImpl) GetMetricsCollector() *monitoring.MetricsCollector {

	return nil // TODO: Implement metrics collector

}

// GetTelecomKnowledgeBase returns the telecom knowledge base (placeholder implementation).

func (d *dependencyImpl) GetTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase {

	return nil // TODO: Implement telecom knowledge base

}

func init() {

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(nephoranv1.AddToScheme(scheme))

}

func main() {

	// Load configuration from environment variables.

	cfg, err := config.LoadFromEnv()

	if err != nil {

		setupLog.Error(err, "failed to load configuration")

		os.Exit(1)

	}

	// Set up flags with configuration defaults.

	flag.StringVar(&cfg.MetricsAddr, "metrics-bind-address", cfg.MetricsAddr, "The address the metric endpoint binds to.")

	flag.StringVar(&cfg.ProbeAddr, "health-probe-bind-address", cfg.ProbeAddr, "The address the probe endpoint binds to.")

	flag.BoolVar(&cfg.EnableLeaderElection, "leader-elect", cfg.EnableLeaderElection,

		"Enable leader election for controller manager. "+

			"Enabling this will ensure there is only one active controller manager.")

	opts := zap.Options{

		Development: true,
	}

	opts.BindFlags(flag.CommandLine)

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	setupLog.Info("Starting with configuration",

		"llm-processor-url", cfg.LLMProcessorURL,

		"rag-api-url", cfg.RAGAPIURLInternal,

		"git-repo-url", cfg.GitRepoURL,

		"namespace", cfg.Namespace)

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{

		Scheme: scheme,

		Metrics: metricsserver.Options{

			BindAddress: cfg.MetricsAddr,
		},

		HealthProbeBindAddress: cfg.ProbeAddr,

		LeaderElection: cfg.EnableLeaderElection,

		LeaderElectionID: "nephoran-intent-operator",
	})

	if err != nil {

		setupLog.Error(err, "unable to start manager")

		os.Exit(1)

	}

	// Initialize clients with configuration.

	llmClient := llm.NewClient(cfg.LLMProcessorURL)

	// Create Git client with token file support.

	var gitClient *git.Client

	if cfg.GitTokenPath != "" || cfg.GitToken != "" {

		gitConfig, err := git.NewGitClientConfig(cfg.GitRepoURL, cfg.GitBranch, cfg.GitToken, cfg.GitTokenPath)

		if err != nil {

			setupLog.Error(err, "unable to create git client config")

			os.Exit(1)

		}

		// Apply concurrent push limit from config if set.

		if cfg.GitConcurrentPushLimit > 0 {

			gitConfig.ConcurrentPushLimit = cfg.GitConcurrentPushLimit

		}

		gitClient = git.NewClientFromConfig(gitConfig)

	} else {

		// Fallback to default constructor for backward compatibility.

		gitClient = git.NewClient(cfg.GitRepoURL, cfg.GitBranch, cfg.GitToken)

	}

	// Initialize Nephio package generator if enabled.

	var packageGen *nephio.PackageGenerator

	useNephioPorch := os.Getenv("USE_NEPHIO_PORCH") == "true"

	if useNephioPorch {

		var err error

		packageGen, err = nephio.NewPackageGenerator()

		if err != nil {

			setupLog.Error(err, "unable to create package generator")

			os.Exit(1)

		}

		setupLog.Info("Nephio Porch integration enabled")

	}

	// Create dependencies struct that implements Dependencies interface.

	deps := &dependencyImpl{

		gitClient: gitClient,

		llmClient: &llmClientAdapter{client: llmClient},

		packageGen: packageGen,

		httpClient: &http.Client{Timeout: 30 * time.Second},

		eventRecorder: mgr.GetEventRecorderFor("network-intent-controller"),
	}

	// Create controller configuration.

	controllerConfig := &controllers.Config{

		MaxRetries: 3,

		RetryDelay: time.Minute * 2,

		Timeout: time.Minute * 10,

		GitRepoURL: cfg.GitRepoURL,

		GitBranch: cfg.GitBranch,

		GitDeployPath: "deployments",

		LLMProcessorURL: cfg.LLMProcessorURL,

		UseNephioPorch: useNephioPorch,
	}

	// Setup NetworkIntent controller.

	networkIntentController, err := controllers.NewNetworkIntentReconciler(

		mgr.GetClient(),

		mgr.GetScheme(),

		deps,

		controllerConfig,
	)

	if err != nil {

		setupLog.Error(err, "unable to create NetworkIntent controller")

		os.Exit(1)

	}

	if err = networkIntentController.SetupWithManager(mgr); err != nil {

		setupLog.Error(err, "unable to setup controller", "controller", "NetworkIntent")

		os.Exit(1)

	}

	// Setup E2NodeSet controller.

	if err = (&controllers.E2NodeSetReconciler{

		Client: mgr.GetClient(),

		Scheme: mgr.GetScheme(),

		Recorder: mgr.GetEventRecorderFor("e2nodeset-controller"),
	}).SetupWithManager(mgr); err != nil {

		setupLog.Error(err, "unable to create controller", "controller", "E2NodeSet")

		os.Exit(1)

	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {

		setupLog.Error(err, "unable to set up health check")

		os.Exit(1)

	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {

		setupLog.Error(err, "unable to set up ready check")

		os.Exit(1)

	}

	setupLog.Info("starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {

		setupLog.Error(err, "problem running manager")

		os.Exit(1)

	}

}
