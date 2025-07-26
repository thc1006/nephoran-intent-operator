package main

import (
	"flag"
	"io/ioutil"
	"os"

	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"nephoran-intent-operator/pkg/controllers"
	"nephoran-intent-operator/pkg/git"
	nephoranv1alpha1 "nephoran-intent-operator/pkg/apis/nephoran/v1alpha1"
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(nephoranv1alpha1.AddToScheme(scheme))
}

func main() {
	// ... (flag parsing remains the same)

	sshKey, err := ioutil.ReadFile(sshKeyPath)
	if err != nil {
		setupLog.Error(err, "unable to read SSH private key")
		os.Exit(1)
	}

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		// ... (manager options)
	})
	if err != nil {
		// ... (error handling)
	}

	gitClient := git.NewClient(gitRepoURL, gitBranch, string(sshKey))

	if err = (&controllers.NetworkIntentReconciler{
		Client:    mgr.GetClient(),
		Scheme:    mgr.GetScheme(),
		GitClient: gitClient,
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "NetworkIntent")
		os.Exit(1)
	}

	// ... (health checks and manager start)
}
