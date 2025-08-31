package main

import (
	"flag"
	"os"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
)

var (
	scheme = runtime.NewScheme()

	setupLog = ctrl.Log.WithName("setup")
)

func init() {

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(intentv1alpha1.AddToScheme(scheme))

}

func main() {

	var (
		metricsAddr string

		probeAddr string

		webhookPort int

		certDir string
	)

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")

	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")

	flag.IntVar(&webhookPort, "webhook-port", 9443, "Webhook server port.")

	flag.StringVar(&certDir, "cert-dir", "", "Directory that contains the webhook serving certs (tls.crt, tls.key). If empty, use controller-runtime defaults.")

	opts := zap.Options{Development: true}

	opts.BindFlags(flag.CommandLine)

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	cfg := ctrl.GetConfigOrDie()

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{

		Scheme: scheme,

		// New-style metrics server options (replaces MetricsBindAddress).

		Metrics: metricsserver.Options{BindAddress: metricsAddr},

		HealthProbeBindAddress: probeAddr,

		// LeaderElection disabled for webhook manager

		LeaderElection: false,
	})

	if err != nil {

		setupLog.Error(err, "unable to start manager")

		os.Exit(1)

	}

	// Create and register webhook server (new API; Port/CertDir set here)

	hookServer := webhook.NewServer(webhook.Options{

		Port: webhookPort,

		CertDir: certDir, // If empty, controller-runtime uses default location

	})

	if err := mgr.Add(hookServer); err != nil {

		setupLog.Error(err, "unable to add webhook server to manager")

		os.Exit(1)

	}

	// Register CRD webhook with manager (no longer register with mgr.GetWebhookServer()

	if err := (&intentv1alpha1.NetworkIntent{}).SetupWebhookWithManager(mgr); err != nil {

		setupLog.Error(err, "unable to create webhook", "webhook", "NetworkIntent")

		os.Exit(1)

	}

	// Health checks and readiness checks

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {

		setupLog.Error(err, "unable to set up health check")

		os.Exit(1)

	}

	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {

		setupLog.Error(err, "unable to set up ready check")

		os.Exit(1)

	}

	setupLog.Info("starting manager (webhook-mode)")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {

		setupLog.Error(err, "problem running manager")

		os.Exit(1)

	}

}
