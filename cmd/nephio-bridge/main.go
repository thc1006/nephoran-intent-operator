package main

import (
	"context"
	"flag"
	"os"
	"time"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
	"k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	_ "k8s.io/client-go/plugin/pkg/client/auth"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(nephoranv1.AddToScheme(scheme))
}

// waitForCRD polls the API server until a specific CRD is established.
func waitForCRD(config *rest.Config, crdName string) error {
	clientset, err := clientset.NewForConfig(config)
	if err != nil {
		return err
	}

	setupLog.Info("waiting for CRD to be established", "CRD", crdName)
	return wait.Poll(5*time.Second, 2*time.Minute, func() (bool, error) {
		_, err := clientset.ApiextensionsV1().CustomResourceDefinitions().Get(context.Background(), crdName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				setupLog.Info("CRD not found yet, retrying...", "CRD", crdName)
				return false, nil // Continue polling
			}
			return false, err // An actual error occurred
		}
		setupLog.Info("CRD has been established", "CRD", crdName)
		return true, nil // CRD found, stop polling
	})
}

func main() {
	var metricsAddr string
	var enableLeaderElection bool
	var probeAddr string
	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	flag.BoolVar(&enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")
	opts := zap.Options{
		Development: true,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		Metrics:                metricsserver.Options{BindAddress: metricsAddr},
		HealthProbeBindAddress: probeAddr,
		LeaderElection:         enableLeaderElection,
		LeaderElectionID:       "9a805727.nephoran.com",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err := waitForCRD(mgr.GetConfig(), "e2nodesets.nephoran.com"); err != nil {
		setupLog.Error(err, "failed to wait for E2NodeSet CRD")
		os.Exit(1)
	}

	gitRepoURL := os.Getenv("GIT_REPO_URL")
	gitBranch := os.Getenv("GIT_BRANCH")
	sshKey := os.Getenv("SSH_KEY")

	gitClient := git.NewClient(gitRepoURL, gitBranch, string(sshKey))

	if err := gitClient.InitRepo(); err != nil {
		setupLog.Error(err, "failed to initialize deployment repository")
		os.Exit(1)
	}

	llmProcessorURL := os.Getenv("LLM_PROCESSOR_URL")
	llmClient := llm.NewClient(llmProcessorURL)

	if err = (&controllers.NetworkIntentReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		GitClient: gitClient,
		LLMClient: llmClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NetworkIntent")
		os.Exit(1)
	}

	if err = (&controllers.E2NodeSetReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		GitClient: gitClient,
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
