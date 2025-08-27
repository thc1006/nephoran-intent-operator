package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

// llmClientAdapter adapts llm.Client to shared.ClientInterface
type llmClientAdapter struct {
	client *llm.Client
}

func (a *llmClientAdapter) ProcessRequest(ctx context.Context, request *shared.LLMRequest) (*shared.LLMResponse, error) {
	// Convert LLMRequest to a simple prompt for the underlying client
	prompt := ""
	for _, msg := range request.Messages {
		prompt += msg.Role + ": " + msg.Content + "\n"
	}
	
	result, err := a.client.ProcessIntent(ctx, prompt)
	if err != nil {
		return nil, err
	}
	
	return &shared.LLMResponse{
		ID:      "adapter-" + time.Now().Format("20060102150405"),
		Content: result,
		Model:   request.Model,
		Usage: shared.TokenUsage{
			PromptTokens:     len(prompt) / 4,
			CompletionTokens: len(result) / 4,
			TotalTokens:      (len(prompt) + len(result)) / 4,
		},
		Created: time.Now(),
	}, nil
}

func (a *llmClientAdapter) ProcessStreamingRequest(ctx context.Context, request *shared.LLMRequest) (<-chan *shared.StreamingChunk, error) {
	// For now, fall back to non-streaming
	response, err := a.ProcessRequest(ctx, request)
	if err != nil {
		return nil, err
	}
	
	chan_result := make(chan *shared.StreamingChunk, 1)
	chan_result <- &shared.StreamingChunk{
		ID:        response.ID,
		Content:   response.Content,
		Done:      true,
		IsLast:    true,
		Timestamp: time.Now(),
	}
	close(chan_result)
	return chan_result, nil
}

func (a *llmClientAdapter) HealthCheck(ctx context.Context) error {
	// Basic health check - assume healthy if client exists
	if a.client == nil {
		return fmt.Errorf("llm client is nil")
	}
	return nil
}

func (a *llmClientAdapter) GetStatus() shared.ClientStatus {
	if a.client == nil {
		return shared.ClientStatusUnhealthy
	}
	return shared.ClientStatusHealthy
}

func (a *llmClientAdapter) GetModelCapabilities() shared.ModelCapabilities {
	return shared.ModelCapabilities{
		MaxTokens:            8192,
		SupportsChat:         true,
		SupportsFunction:     false,
		SupportsStreaming:    false,
		SupportsChatFormat:   true,
		SupportsSystemPrompt: true,
		CostPerToken:         0.001,
		SupportedMimeTypes:   []string{"text/plain"},
		ModelVersion:         "1.0",
		Features:             make(map[string]interface{}),
	}
}

func (a *llmClientAdapter) GetEndpoint() string {
	// Return a default endpoint - in real implementation this would be configurable
	return "http://localhost:8080"
}

func (a *llmClientAdapter) Close() error {
	a.client.Shutdown()
	return nil
}

// dependencyImpl implements the Dependencies interface
type dependencyImpl struct {
	gitClient              git.ClientInterface
	llmClient              shared.ClientInterface
	packageGen             *nephio.PackageGenerator
	httpClient             *http.Client
	eventRecorder          record.EventRecorder
	telecomKnowledgeBase   *telecom.TelecomKnowledgeBase
	metricsCollector       *monitoring.MetricsCollector
}

func (d *dependencyImpl) GetGitClient() git.ClientInterface {
	return d.gitClient
}

func (d *dependencyImpl) GetLLMClient() shared.ClientInterface {
	return d.llmClient
}

func (d *dependencyImpl) GetPackageGenerator() *nephio.PackageGenerator {
	return d.packageGen
}

func (d *dependencyImpl) GetHTTPClient() *http.Client {
	return d.httpClient
}

func (d *dependencyImpl) GetEventRecorder() record.EventRecorder {
	return d.eventRecorder
}

func (d *dependencyImpl) GetTelecomKnowledgeBase() *telecom.TelecomKnowledgeBase {
	return d.telecomKnowledgeBase
}

func (d *dependencyImpl) GetMetricsCollector() *monitoring.MetricsCollector {
	return d.metricsCollector
}

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(nephoranv1.AddToScheme(scheme))
}

func main() {
	// Load configuration from environment variables
	cfg, err := config.LoadFromEnv()
	if err != nil {
		setupLog.Error(err, "failed to load configuration")
		os.Exit(1)
	}

	// Set up flags with configuration defaults
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
		LeaderElection:         cfg.EnableLeaderElection,
		LeaderElectionID:       "nephoran-intent-operator",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Initialize clients with configuration
	llmClient := llm.NewClient(cfg.LLMProcessorURL)

	// Create Git client with token file support
	var gitClient *git.Client
	if cfg.GitTokenPath != "" || cfg.GitToken != "" {
		gitConfig, err := git.NewGitClientConfig(cfg.GitRepoURL, cfg.GitBranch, cfg.GitToken, cfg.GitTokenPath)
		if err != nil {
			setupLog.Error(err, "unable to create git client config")
			os.Exit(1)
		}
		// Apply concurrent push limit from config if set
		if cfg.GitConcurrentPushLimit > 0 {
			gitConfig.ConcurrentPushLimit = cfg.GitConcurrentPushLimit
		}
		gitClient = git.NewClientFromConfig(gitConfig)
	} else {
		// Fallback to default constructor for backward compatibility
		gitClient = git.NewClient(cfg.GitRepoURL, cfg.GitBranch, cfg.GitToken)
	}

	// Initialize Nephio package generator if enabled
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

	// Initialize telecom knowledge base and metrics collector
	telecomKB := &telecom.TelecomKnowledgeBase{
		NetworkFunctions: make(map[string]*telecom.NetworkFunctionSpec),
		Interfaces:       make(map[string]*telecom.InterfaceSpec),
		QosProfiles:      make(map[string]*telecom.QosProfile),
		SliceTypes:       make(map[string]*telecom.SliceTypeSpec),
		PerformanceKPIs:  make(map[string]*telecom.KPISpec),
	}
	
	metricsCollector := &monitoring.MetricsCollector{}
	
	// Create dependencies struct that implements Dependencies interface
	deps := &dependencyImpl{
		gitClient:              gitClient,
		llmClient:              &llmClientAdapter{client: llmClient},
		packageGen:             packageGen,
		httpClient:             &http.Client{Timeout: 30 * time.Second},
		eventRecorder:          mgr.GetEventRecorderFor("network-intent-controller"),
		telecomKnowledgeBase:   telecomKB,
		metricsCollector:       metricsCollector,
	}

	// Create controller configuration
	controllerConfig := &controllers.Config{
		MaxRetries:      3,
		RetryDelay:      time.Minute * 2,
		Timeout:         time.Minute * 10,
		GitRepoURL:      cfg.GitRepoURL,
		GitBranch:       cfg.GitBranch,
		GitDeployPath:   "deployments",
		LLMProcessorURL: cfg.LLMProcessorURL,
		UseNephioPorch:  useNephioPorch,
	}

	// Setup NetworkIntent controller
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

	// Setup E2NodeSet controller
	if err = (&controllers.E2NodeSetReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
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
