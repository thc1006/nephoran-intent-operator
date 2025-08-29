package main

import (
	"flag"
	"os"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"

	"github.com/nephio-project/nephoran-intent-operator/pkg/controllers"

	"k8s.io/apimachinery/pkg/runtime"

	utilruntime "k8s.io/apimachinery/pkg/util/runtime"

	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	ctrl "sigs.k8s.io/controller-runtime"

	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

var (
	scheme = runtime.NewScheme()

	setupLog = ctrl.Log.WithName("setup")
)

func init() {

	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(nephoranv1.AddToScheme(scheme))

}

func main() {

	var metricsAddr string

	var enableLeaderElection bool

	var probeAddr string

	flag.StringVar(&metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")

	flag.StringVar(&probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")

	flag.BoolVar(&enableLeaderElection, "leader-elect", false, "Enable leader election for controller manager.")

	opts := zap.Options{

		Development: true,
	}

	opts.BindFlags(flag.CommandLine)

	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{

		Scheme: scheme,

		HealthProbeBindAddress: probeAddr,

		LeaderElection: enableLeaderElection,

		LeaderElectionID: "oran-adaptor.nephoran.io",
	})

	if err != nil {

		setupLog.Error(err, "unable to start manager")

		os.Exit(1)

	}

	if err = (&controllers.OranAdaptorReconciler{

		Client: mgr.GetClient(),

		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {

		setupLog.Error(err, "unable to create controller", "controller", "OranAdaptor")

		os.Exit(1)

	}

	setupLog.Info("starting manager")

	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {

		setupLog.Error(err, "problem running manager")

		os.Exit(1)

	}

}
