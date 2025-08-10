package main

import (
	"crypto/tls"
	"flag"
	"fmt"
	"os"
	"strconv"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	"github.com/thc1006/nephoran-intent-operator/pkg/injection"
	"github.com/thc1006/nephoran-intent-operator/pkg/security"
	"github.com/thc1006/nephoran-intent-operator/pkg/webhooks"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(nephoranv1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

// getEnvAsBool retrieves environment variable as boolean with fallback
func getEnvAsBool(key string, defaultValue bool) bool {
	if value, exists := os.LookupEnv(key); exists {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func main() {
	// Load configuration from environment variables with validation
	constants := config.LoadConstants()
	
	// Load main config from environment
	cfg, err := config.LoadFromEnv()
	if err != nil {
		setupLog.Error(err, "Failed to load configuration from environment")
		os.Exit(1)
	}

	// Validate configuration
	if err := config.ValidateConstants(constants); err != nil {
		setupLog.Error(err, "Configuration validation failed")
		os.Exit(1)
	}

	// Also validate complete configuration with all rules
	if err := config.ValidateCompleteConfiguration(constants); err != nil {
		setupLog.Error(err, "Complete configuration validation failed")
		os.Exit(1)
	}

	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool
	var enableNetworkIntent bool
	var enableLlmIntent bool
	var enableWebhooks bool
	var printConfig bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", fmt.Sprintf(":%d", constants.MetricsPort), "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", fmt.Sprintf(":%d", constants.HealthProbePort), "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", false, "If set the metrics endpoint is served securely")
	flag.BoolVar(&enableHTTP2, "enable-http2", false, "If set, HTTP/2 will be enabled for the webhook and metrics servers")
	flag.BoolVar(&enableNetworkIntent, "enable-network-intent", cfg.GetEnableNetworkIntent(), "Enable NetworkIntent controller")
	flag.BoolVar(&enableLlmIntent, "enable-llm-intent", cfg.GetEnableLLMIntent(), "Enable LLM Intent processing")
	flag.BoolVar(&enableWebhooks, "enable-webhooks", getEnvAsBool("ENABLE_WEBHOOKS", true), "Enable admission webhooks for validation")
	flag.BoolVar(&printConfig, "print-config", false, "Print current configuration and exit")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// Print configuration if requested
	if printConfig {
		config.PrintConfiguration(constants)
		return
	}

	disableHTTP2 := func(c *tls.Config) {
		setupLog.Info("disabling http/2")
		c.NextProtos = []string{"http/1.1"}
	}

	tlsOpts := []func(*tls.Config){}
	if !enableHTTP2 {
		tlsOpts = append(tlsOpts, disableHTTP2)
	}

	webhookServer := webhook.NewServer(webhook.Options{
		TLSOpts: tlsOpts,
	})

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Metrics: metricsserver.Options{
			BindAddress:   metricsAddr,
			SecureServing: secureMetrics,
			TLSOpts:       tlsOpts,
		},
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "80807133.nephoran.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Setup controllers based on feature flags
	if enableNetworkIntent {
		// Create dependency injection container
		container := injection.NewContainer(constants)

		// Set the event recorder from the manager
		container.SetEventRecorder(mgr.GetEventRecorderFor("nephoran-intent-operator"))

		// Create configuration with values from loaded constants
		controllerConfig := &controllers.Config{
			MaxRetries:      constants.MaxRetries,
			RetryDelay:      constants.RetryDelay,
			Timeout:         constants.Timeout,
			GitRepoURL:      os.Getenv("GIT_REPO_URL"),
			GitBranch:       os.Getenv("GIT_BRANCH"),
			GitDeployPath:   constants.GitDeployPath,
			LLMProcessorURL: os.Getenv("LLM_PROCESSOR_URL"),
			UseNephioPorch:  getEnvAsBool("USE_NEPHIO_PORCH", false),
			Constants:       constants,
		}

		// Create LLM sanitizer configuration from constants
		sanitizerConfig := &security.SanitizerConfig{
			MaxInputLength:  constants.MaxInputLength,
			MaxOutputLength: constants.MaxOutputLength,
			AllowedDomains:  constants.AllowedDomains,
			BlockedKeywords: constants.BlockedKeywords,
			ContextBoundary: constants.ContextBoundary,
			SystemPrompt:    constants.SystemPrompt,
		}

		// Initialize the LLM sanitizer with configuration
		llmSanitizer := security.NewLLMSanitizer(sanitizerConfig)

		// Create controller with dependencies and configuration
		controller, err := controllers.NewNetworkIntentReconciler(
			mgr.GetClient(),
			mgr.GetScheme(),
			container, // Use the container as dependencies
			controllerConfig,
		)
		if err != nil {
			setupLog.Error(err, "unable to create NetworkIntent controller")
			os.Exit(1)
		}

		if err = controller.SetupWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to setup controller", "controller", "NetworkIntent")
			os.Exit(1)
		}

		setupLog.Info("NetworkIntent controller enabled with configurable constants",
			"max_retries", constants.MaxRetries,
			"retry_delay", constants.RetryDelay,
			"timeout", constants.Timeout,
			"llm_timeout", constants.LLMTimeout,
			"git_timeout", constants.GitTimeout,
			"kubernetes_timeout", constants.KubernetesTimeout,
			"circuit_breaker_threshold", constants.CircuitBreakerFailureThreshold,
			"circuit_breaker_recovery_timeout", constants.CircuitBreakerRecoveryTimeout,
			"max_input_length", constants.MaxInputLength,
			"max_output_length", constants.MaxOutputLength,
			"allowed_domains_count", len(constants.AllowedDomains),
			"blocked_keywords_count", len(constants.BlockedKeywords),
			"sanitizer_initialized", llmSanitizer != nil)
	} else {
		setupLog.Info("NetworkIntent controller disabled")
	}

	if enableLlmIntent {
		setupLog.Info("LLM Intent processing enabled")
	} else {
		setupLog.Info("LLM Intent processing disabled")
	}

	// Setup webhooks based on feature flags
	if enableWebhooks {
		if err = webhooks.SetupNetworkIntentWebhookWithManager(mgr); err != nil {
			setupLog.Error(err, "unable to create webhook", "webhook", "NetworkIntent")
			os.Exit(1)
		}
		setupLog.Info("NetworkIntent validation webhook enabled")
	} else {
		setupLog.Info("Admission webhooks disabled")
	}

	// +kubebuilder:scaffold:builder

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	fmt.Println("Nephoran Intent Operator starting")
	setupLog.Info("starting manager",
		"networkIntentEnabled", enableNetworkIntent,
		"llmIntentEnabled", enableLlmIntent,
		"webhooksEnabled", enableWebhooks)

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
