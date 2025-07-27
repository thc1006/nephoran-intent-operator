package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/llm"
)

type NetworkIntentReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	GitClient       git.ClientInterface
	LLMClient       llm.ClientInterface
	LLMProcessorURL string
	HTTPClient      *http.Client
}

//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nephoran.com,resources=networkintents/finalizers,verbs=update

func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Fetch the NetworkIntent instance
	var networkIntent nephoranv1.NetworkIntent
	if err := r.Get(ctx, req.NamespacedName, &networkIntent); err != nil {
		logger.Error(err, "unable to fetch NetworkIntent")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Process the intent using LLM client
	if r.LLMClient != nil {
		processedResult, err := r.LLMClient.ProcessIntent(ctx, networkIntent.Spec.Intent)
		if err != nil {
			logger.Error(err, "failed to process intent with LLM")
			// Update status to reflect the error
			networkIntent.Status.Conditions = []metav1.Condition{{
				Type:               "Processed",
				Status:             metav1.ConditionFalse,
				Reason:             "LLMProcessingFailed",
				Message:            fmt.Sprintf("Failed to process intent: %v", err),
				LastTransitionTime: metav1.Now(),
			}}
			if updateErr := r.Status().Update(ctx, &networkIntent); updateErr != nil {
				logger.Error(updateErr, "failed to update NetworkIntent status")
			}
			return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
		}

		// Parse the processed result and update the Parameters field
		var parameters map[string]interface{}
		if err := json.Unmarshal([]byte(processedResult), &parameters); err != nil {
			logger.Error(err, "failed to parse LLM response")
			return ctrl.Result{RequeueAfter: time.Minute * 5}, nil
		}

		// Update NetworkIntent with processed parameters
		parametersRaw, _ := json.Marshal(parameters)
		networkIntent.Spec.Parameters = runtime.RawExtension{Raw: parametersRaw}

		if err := r.Update(ctx, &networkIntent); err != nil {
			logger.Error(err, "failed to update NetworkIntent with parameters")
			return ctrl.Result{}, err
		}

		// Update status to reflect successful processing
		networkIntent.Status.Conditions = []metav1.Condition{{
			Type:               "Processed",
			Status:             metav1.ConditionTrue,
			Reason:             "LLMProcessingSucceeded",
			Message:            "Intent successfully processed by LLM",
			LastTransitionTime: metav1.Now(),
		}}
		
		if err := r.Status().Update(ctx, &networkIntent); err != nil {
			logger.Error(err, "failed to update NetworkIntent status")
			return ctrl.Result{}, err
		}

		logger.Info("NetworkIntent processed successfully", "intent", networkIntent.Spec.Intent)
	}

	return ctrl.Result{}, nil
}

func (r *NetworkIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		Complete(r)
}
