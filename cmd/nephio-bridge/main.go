
package main



import (

	"context"

	"flag"

	"net/http"

	"os"

	"time"



	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"

	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"

	"github.com/thc1006/nephoran-intent-operator/pkg/git"

	"github.com/thc1006/nephoran-intent-operator/pkg/llm"

	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"

	"github.com/thc1006/nephoran-intent-operator/pkg/shared"

	"github.com/thc1006/nephoran-intent-operator/pkg/telecom"



	"k8s.io/apimachinery/pkg/runtime"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"k8s.io/client-go/tools/record"



	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/healthz"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

)



var (

	scheme   = runtime.NewScheme()

	setupLog = ctrl.Log.WithName("setup")

)



// llmClientAdapter adapts llm.Client to shared.ClientInterface.

type llmClientAdapter struct {

	client *llm.Client

}



// ProcessIntent performs processintent operation.

func (a *llmClientAdapter) ProcessIntent(ctx context.Context, prompt string) (string, error) {

	return a.client.ProcessIntent(ctx, prompt)

}



// ProcessIntentStream performs processintentstream operation.

func (a *llmClientAdapter) ProcessIntentStream(ctx context.Context, prompt string, chunks chan<- *shared.StreamingChunk) error {

	// For now, fall back to non-streaming.

	result, err := a.client.ProcessIntent(ctx, prompt)

	if err != nil {

		return err

	}



	if chunks != nil {

		chunks <- &shared.StreamingChunk{

			Content: result,

			IsLast:  true,

		}

		close(chunks)

	}

	return nil

}



// GetSupportedModels performs getsupportedmodels operation.

func (a *llmClientAdapter) GetSupportedModels() []string {

	return []string{"gpt-4o-mini", "gpt-4", "gpt-3.5-turbo"}

}



// GetModelCapabilities performs getmodelcapabilities operation.

func (a *llmClientAdapter) GetModelCapabilities(modelName string) (*shared.ModelCapabilities, error) {

	return &shared.ModelCapabilities{

		MaxTokens:         8192,

		SupportsChat:      true,

		SupportsFunction:  false,

		SupportsStreaming: false,

		CostPerToken:      0.001,

		Features:          make(map[string]interface{}),

	}, nil

}



// ValidateModel performs validatemodel operation.

func (a *llmClientAdapter) ValidateModel(modelName string) error {

	// Basic validation.

	return nil

}



// EstimateTokens performs estimatetokens operation.

func (a *llmClientAdapter) EstimateTokens(text string) int {

	// Simple estimation: roughly 4 characters per token.

	return len(text) / 4

}



// GetMaxTokens performs getmaxtokens operation.

func (a *llmClientAdapter) GetMaxTokens(modelName string) int {

	return 8192

}



// Close performs close operation.

func (a *llmClientAdapter) Close() error {

	a.client.Shutdown()

	return nil

}



// dependencyImpl implements the Dependencies interface.

type dependencyImpl struct {

	gitClient     git.ClientInterface

	llmClient     shared.ClientInterface

	packageGen    *nephio.PackageGenerator

	httpClient    *http.Client

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

		log.Fatal(1)

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

		LeaderElection:         cfg.EnableLeaderElection,

		LeaderElectionID:       "nephoran-intent-operator",

	})

	if err != nil {

		setupLog.Error(err, "unable to start manager")

		log.Fatal(1)

	}



	// Initialize clients with configuration.

	llmClient := llm.NewClient(cfg.LLMProcessorURL)



	// Create Git client with token file support.

	var gitClient *git.Client

	if cfg.GitTokenPath != "" || cfg.GitToken != "" {

		gitConfig, err := git.NewGitClientConfig(cfg.GitRepoURL, cfg.GitBranch, cfg.GitToken, cfg.GitTokenPath)

		if err != nil {

			setupLog.Error(err, "unable to create git client config")

			log.Fatal(1)

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

			log.Fatal(1)

		}

		setupLog.Info("Nephio Porch integration enabled")

	}



	// Create dependencies struct that implements Dependencies interface.

	deps := &dependencyImpl{

		gitClient:     gitClient,

		llmClient:     &llmClientAdapter{client: llmClient},

		packageGen:    packageGen,

		httpClient:    &http.Client{Timeout: 30 * time.Second},

		eventRecorder: mgr.GetEventRecorderFor("network-intent-controller"),

	}



	// Create controller configuration.

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



	// Setup NetworkIntent controller.

	networkIntentController, err := controllers.NewNetworkIntentReconciler(

		mgr.GetClient(),

		mgr.GetScheme(),

		deps,

		controllerConfig,

	)

	if err != nil {

		setupLog.Error(err, "unable to create NetworkIntent controller")

		log.Fatal(1)

	}



	if err = networkIntentController.SetupWithManager(mgr); err != nil {

		setupLog.Error(err, "unable to setup controller", "controller", "NetworkIntent")

		log.Fatal(1)

	}



	// Setup E2NodeSet controller.

	if err = (&controllers.E2NodeSetReconciler{

		Client:   mgr.GetClient(),

		Scheme:   mgr.GetScheme(),

		Recorder: mgr.GetEventRecorderFor("e2nodeset-controller"),

	}).SetupWithManager(mgr); err != nil {

		setupLog.Error(err, "unable to create controller", "controller", "E2NodeSet")

		log.Fatal(1)

	}



	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {

		setupLog.Error(err, "unable to set up health check")

		log.Fatal(1)

	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {

		setupLog.Error(err, "unable to set up ready check")

		log.Fatal(1)

	}



	setupLog.Info("starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {

		setupLog.Error(err, "problem running manager")

		log.Fatal(1)

	}

}

