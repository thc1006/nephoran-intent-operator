/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUTHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1alpha1 "github.com/thc1006/nephoran-intent-operator/pkg/apis/nephoran/v1alpha1"
)

// E2NodeSetReconciler reconciles a E2NodeSet object
type E2NodeSetReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=nephoran.org,resources=e2nodesets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nephoran.org,resources=e2nodesets/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nephoran.org,resources=e2nodesets/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *E2NodeSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var e2NodeSet nephoranv1alpha1.E2NodeSet
	if err := r.Get(ctx, req.NamespacedName, &e2NodeSet); err != nil {
		logger.Error(err, "unable to fetch E2NodeSet")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	logger.Info("Reconciling E2NodeSet", "replicas", e2NodeSet.Spec.Replicas)

	// In a future step, we will add the logic here to:
	// 1. Load the ric-sim Kpt package.
	// 2. Generate 'replicas' number of Deployment manifests.
	// 3. Commit the manifests to the deployment Git repository.

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *E2NodeSetReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1alpha1.E2NodeSet{}).
		Complete(r)
}
