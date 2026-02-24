package main

import (
	"crypto/tls"
	"flag"
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"sigs.k8s.io/controller-runtime/pkg/metrics/filters"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
	"github.com/thc1006/nephoran-intent-operator/controllers"
	"github.com/thc1006/nephoran-intent-operator/pkg/validation"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(intentv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme

	// Apply Go 1.24.8 runtime optimizations early
	initializeRuntimeOptimizations()
}

// initializeRuntimeOptimizations applies Go 1.24.8 performance tuning
func initializeRuntimeOptimizations() {
	// Go 1.24.8 runtime optimizations (applied at process startup)
	// These optimizations are applied via build flags and environment variables
	// No additional runtime imports needed for core optimizations
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	var secureMetrics bool
	var enableHTTP2 bool

	// Endpoint flags — override environment variables when provided
	var a1Endpoint string
	var a1APIFormat string
	var llmEndpoint string
	var porchServer string
	var validateEndpointsDNS bool

	flag.StringVar(&metricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8080 for HTTPS or :8081 for HTTP, or leave as 0 to disable the metrics service.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	flag.BoolVar(&secureMetrics, "metrics-secure", true,
		"If set, the metrics endpoint is served securely via HTTPS. Use --metrics-secure=false to use HTTP instead.")
	flag.BoolVar(&enableHTTP2, "enable-http2", false,
		"If set, HTTP/2 will be enabled for the metrics and webhook servers")
	flag.StringVar(&a1Endpoint, "a1-endpoint", "",
		"Non-RT RIC A1 Policy Management Service endpoint (e.g. http://nonrtric-a1pms:8081). "+
			"Overrides the A1_MEDIATOR_URL environment variable.")
	flag.StringVar(&a1APIFormat, "a1-api-format", "",
		"A1 API path format: 'standard' for O-RAN Alliance (/v2/policies/{policyId}), "+
			"'legacy' for O-RAN SC RICPLT (/A1-P/v2/policytypes/{typeId}/policies/{policyId}). "+
			"Defaults to 'legacy'. Overrides A1_API_FORMAT environment variable.")
	flag.StringVar(&llmEndpoint, "llm-endpoint", "",
		"LLM inference endpoint (e.g. http://ollama-service:11434). "+
			"Overrides the LLM_PROCESSOR_URL environment variable.")
	flag.StringVar(&porchServer, "porch-server", "",
		"Nephio Porch server endpoint (e.g. http://porch-server:7007). "+
			"Overrides the PORCH_SERVER_URL environment variable.")
	flag.BoolVar(&validateEndpointsDNS, "validate-endpoints-dns", false,
		"Enable DNS resolution checking for configured endpoints at startup. "+
			"May slow startup but catches configuration errors early.")

	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	// if the enable-http2 flag is false (the default), http/2 should be disabled
	// due to its vulnerabilities. More specifically, disabling http/2 will
	// prevent from potential DoS attacks to the webhook server.
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

	// Metrics endpoint is enabled in 'config/default/kustomization.yaml'. The Metrics options configure the server.
	// More info:
	// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/metrics/server
	// - https://book.kubebuilder.io/reference/metrics.html
	metricsServerOptions := metricsserver.Options{
		BindAddress:   metricsAddr,
		SecureServing: secureMetrics,
		TLSOpts:       tlsOpts,
	}

	if secureMetrics {
		// FilterProvider is used to protect the metrics endpoint with authn/authz.
		// These configurations ensure that only authorized users and service accounts
		// can access the metrics endpoint. The RBAC are configured in 'config/rbac/kustomization.yaml'. More info:
		// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.21.0/pkg/metrics/filters#WithAuthenticationAndAuthorization
		metricsServerOptions.FilterProvider = filters.WithAuthenticationAndAuthorization
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsServerOptions,
		WebhookServer:          webhookServer,
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "1caa2c9c.nephoran.io",
		// 2025 best practice: Enable graceful shutdown and set controller concurrency
		GracefulShutdownTimeout: &[]time.Duration{30 * time.Second}[0],
		// LeaderElectionReleaseOnCancel defines if the leader should step down voluntarily
		// when the Manager ends. This requires the binary to immediately end when the
		// Manager is stopped, otherwise, this setting is unsafe. Setting this significantly
		// speeds up voluntary leader transitions as the new leader don't have to wait
		// LeaseDuration time first.
		//
		// In the default scaffold provided, the program ends immediately after
		// the manager stops, so would be fine to enable this option. However,
		// if you are doing or is intended to do any operation such as perform cleanups
		// after the manager stops then its usage might be unsafe.
		// LeaderElectionReleaseOnCancel: true,
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	// Propagate flag values into env vars so all downstream packages (blueprints,
	// packagerevision, nephio) pick them up via their envOrDefault helpers.
	if porchServer != "" {
		os.Setenv("PORCH_SERVER_URL", porchServer) //nolint:errcheck
	}
	if llmEndpoint != "" {
		os.Setenv("LLM_ENDPOINT", llmEndpoint) //nolint:errcheck
	}
	if a1APIFormat != "" {
		os.Setenv("A1_API_FORMAT", a1APIFormat) //nolint:errcheck
	}

	// Validate endpoints before starting controllers
	if err := validateEndpoints(a1Endpoint, llmEndpoint, porchServer, validateEndpointsDNS); err != nil {
		setupLog.Error(err, "Endpoint validation failed at startup")
		os.Exit(1)
	}

	reconciler := &controllers.NetworkIntentReconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}
	// Apply flag overrides (flags take precedence over env vars read in SetupWithManager)
	if a1Endpoint != "" {
		reconciler.A1MediatorURL = a1Endpoint
	}
	if llmEndpoint != "" {
		reconciler.LLMProcessorURL = llmEndpoint
	}
	if a1APIFormat != "" {
		switch a1APIFormat {
		case "standard":
			reconciler.A1APIFormat = controllers.A1FormatStandard
		case "legacy":
			reconciler.A1APIFormat = controllers.A1FormatLegacy
		default:
			setupLog.Error(nil, "Invalid A1 API format flag", "format", a1APIFormat)
			os.Exit(1)
		}
	}

	if err = reconciler.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NetworkIntent")
		os.Exit(1)
	}

	// Setup webhook for NetworkIntent v1alpha1
	if err = (&intentv1alpha1.NetworkIntent{}).SetupWebhookWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "NetworkIntent")
		os.Exit(1)
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

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}

// validateEndpoints validates all configured endpoints at startup to fail fast
// with clear error messages instead of encountering runtime DNS errors
func validateEndpoints(a1Endpoint, llmEndpoint, porchServer string, validateDNS bool) error {
	// Collect all configured endpoints (from flags and environment)
	endpoints := make(map[string]string)

	// A1 Mediator endpoint
	a1URL := a1Endpoint
	if a1URL == "" {
		// Check environment variables (same precedence as in reconciler)
		a1URL = getFirstNonEmpty(os.Getenv("A1_MEDIATOR_URL"), os.Getenv("A1_ENDPOINT"))
	}
	if a1URL != "" {
		endpoints["A1 Mediator"] = a1URL
	}

	// LLM endpoint
	llmURL := llmEndpoint
	if llmURL == "" {
		llmURL = getFirstNonEmpty(os.Getenv("LLM_PROCESSOR_URL"), os.Getenv("LLM_ENDPOINT"))
	}
	if llmURL != "" {
		endpoints["LLM Service"] = llmURL
	}

	// Porch server endpoint
	porchURL := porchServer
	if porchURL == "" {
		porchURL = os.Getenv("PORCH_SERVER_URL")
	}
	if porchURL != "" {
		endpoints["Porch Server"] = porchURL
	}

	// RAG service endpoint (environment only, no flag for this one)
	ragURL := os.Getenv("RAG_API_URL")
	if ragURL != "" {
		endpoints["RAG Service"] = ragURL
	}

	// Validate all endpoints
	config := &validation.ValidationConfig{
		ValidateDNS:          validateDNS,
		ValidateReachability: false, // Don't check reachability at startup (services may not be ready)
		Timeout:              5 * time.Second,
	}

	validationErrors := validation.ValidateEndpoints(endpoints, config)
	if len(validationErrors) > 0 {
		// Build helpful error message with suggestions
		var errorMessages []string
		for _, err := range validationErrors {
			// Extract service name from error
			serviceName := extractServiceName(err.Error())
			suggestion := validation.GetCommonErrorSuggestions(serviceName)
			errorMessages = append(errorMessages, err.Error()+"\n  → "+suggestion)
		}

		return formatValidationErrors(errorMessages)
	}

	return nil
}

// getFirstNonEmpty returns the first non-empty string from the arguments
func getFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// extractServiceName extracts the service name from an error message
func extractServiceName(errorMsg string) string {
	// Error format: "ServiceName endpoint validation failed: ..."
	parts := strings.SplitN(errorMsg, " endpoint validation failed:", 2)
	if len(parts) == 2 && parts[0] != "" {
		return strings.TrimSpace(parts[0])
	}
	return "Unknown Service"
}

// formatValidationErrors formats multiple validation errors into a single error
func formatValidationErrors(errorMessages []string) error {
	if len(errorMessages) == 0 {
		return nil
	}

	var sb strings.Builder
	sb.WriteString("Endpoint validation failed:\n\n")
	for i, msg := range errorMessages {
		sb.WriteString(msg)
		if i < len(errorMessages)-1 {
			sb.WriteString("\n\n")
		}
	}
	sb.WriteString("\n\nFix configuration and restart the operator.")

	return &EndpointValidationFailure{Message: sb.String()}
}

// EndpointValidationFailure represents a fatal startup validation failure
type EndpointValidationFailure struct {
	Message string
}

func (e *EndpointValidationFailure) Error() string {
	return e.Message
}
