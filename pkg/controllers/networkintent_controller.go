package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1alpha1 "nephoran-intent-operator/pkg/apis/nephoran/v1alpha1"
	"nephoran-intent-operator/pkg/git"
)

// NetworkIntentReconciler reconciles a NetworkIntent object
type NetworkIntentReconciler struct {
	client.Client
	Scheme          *runtime.Scheme
	GitClient       git.ClientInterface
	LLMProcessorURL string
	HTTPClient      *http.Client
}

//+kubebuilder:rbac:groups=nephoran.nephoran.io,resources=networkintents,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=nephoran.nephoran.io,resources=networkintents/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=nephoran.nephoran.io,resources=networkintents/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling NetworkIntent")

	var networkIntent nephoranv1alpha1.NetworkIntent
	if err := r.Get(ctx, req.NamespacedName, &networkIntent); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Check if the intent has already been processed by checking the status condition
	if meta.IsStatusConditionTrue(networkIntent.Status.Conditions, "Processed") {
		logger.Info("Intent already processed")
		return ctrl.Result{}, nil
	}

	// 1. Call LLM Processor to get the structured intent
	structuredIntent, err := r.callLLMProcessor(ctx, &networkIntent)
	if err != nil {
		logger.Error(err, "Failed to call LLM Processor")
		meta.SetStatusCondition(&networkIntent.Status.Conditions, metav1.Condition{
			Type:    "LLMProcessingFailed",
			Status:  metav1.ConditionTrue,
			Reason:  "ErrorCallingLLMProcessor",
			Message: err.Error(),
		})
		if updateErr := r.Status().Update(ctx, &networkIntent); updateErr != nil {
			logger.Error(updateErr, "Failed to update NetworkIntent status")
		}
		return ctrl.Result{}, err // Requeue to retry
	}

	logger.Info("Received structured intent from LLM", "intent", structuredIntent)

	// 2. Dispatch to the appropriate handler based on the structured intent type
	var commitHash string
	var dispatchErr error

	intentBytes, err := json.Marshal(structuredIntent)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to re-marshal structured intent: %w", err)
	}

	intentType, ok := structuredIntent["type"].(string)
	if !ok {
		return ctrl.Result{}, fmt.Errorf("structured intent missing 'type' field")
	}

	switch intentType {
	case "NetworkFunctionDeployment":
		var deployIntent NetworkFunctionDeploymentIntent
		if err := json.Unmarshal(intentBytes, &deployIntent); err != nil {
			dispatchErr = fmt.Errorf("failed to unmarshal deployment intent: %w", err)
		} else {
			commitHash, dispatchErr = r.handleDeploymentIntent(ctx, &deployIntent)
		}
	case "NetworkFunctionScale":
		var scaleIntent NetworkFunctionScaleIntent
		if err := json.Unmarshal(intentBytes, &scaleIntent); err != nil {
			dispatchErr = fmt.Errorf("failed to unmarshal scale intent: %w", err)
		} else {
			commitHash, dispatchErr = r.handleScaleIntent(ctx, &scaleIntent)
		}
	default:
		dispatchErr = fmt.Errorf("unknown structured intent type: %s", intentType)
	}

	if dispatchErr != nil {
		logger.Error(dispatchErr, "Failed to handle intent")
		meta.SetStatusCondition(&networkIntent.Status.Conditions, metav1.Condition{
			Type:    "GitProcessingFailed",
			Status:  metav1.ConditionTrue,
			Reason:  "ErrorHandlingIntent",
			Message: dispatchErr.Error(),
		})
		if updateErr := r.Status().Update(ctx, &networkIntent); updateErr != nil {
			logger.Error(updateErr, "Failed to update NetworkIntent status")
		}
		return ctrl.Result{}, dispatchErr
	}

	// 3. Update Status to mark as processed
	meta.SetStatusCondition(&networkIntent.Status.Conditions, metav1.Condition{
		Type:    "Processed",
		Status:  metav1.ConditionTrue,
		Reason:  "IntentProcessedSuccessfully",
		Message: fmt.Sprintf("Intent processed and committed with hash: %s", commitHash),
	})
	if err := r.Status().Update(ctx, &networkIntent); err != nil {
		logger.Error(err, "Failed to update NetworkIntent status")
		return ctrl.Result{}, err
	}

	logger.Info("Successfully processed and reconciled NetworkIntent", "commitHash", commitHash)
	return ctrl.Result{}, nil
}

func (r *NetworkIntentReconciler) callLLMProcessor(ctx context.Context, intent *nephoranv1alpha1.NetworkIntent) (map[string]interface{}, error) {
	requestBody, err := json.Marshal(intent)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", r.LLMProcessorURL, bytes.NewBuffer(requestBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create http request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := r.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request to llm-processor: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("llm-processor returned non-200 status code: %d", resp.StatusCode)
	}

	var structuredIntent map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&structuredIntent); err != nil {
		return nil, fmt.Errorf("failed to decode response from llm-processor: %w", err)
	}

	return structuredIntent, nil
}

func (r *NetworkIntentReconciler) handleDeploymentIntent(ctx context.Context, intent *NetworkFunctionDeploymentIntent) (string, error) {
	commitMessage := fmt.Sprintf("feat(%s): create network function %s", intent.Namespace, intent.Name)
	modifyFunc := func(repoPath string) error {
		// Git logic to write manifests...
		return nil
	}
	return r.GitClient.CommitAndPush(ctx, commitMessage, modifyFunc)
}

func (r *NetworkIntentReconciler) handleScaleIntent(ctx context.Context, intent *NetworkFunctionScaleIntent) (string, error) {
	commitMessage := fmt.Sprintf("feat(%s): scale network function %s to %d replicas", intent.Namespace, intent.Name, intent.Replicas)
	modifyFunc := func(repoPath string) error {
		// Git logic to update manifests...
		return nil
	}
	return r.GitClient.CommitAndPush(ctx, commitMessage, modifyFunc)
}

func (r *NetworkIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	r.HTTPClient = &http.Client{Timeout: 30 * time.Second}
	r.LLMProcessorURL = os.Getenv("LLM_PROCESSOR_URL")
	if r.LLMProcessorURL == "" {
		r.LLMProcessorURL = "http://llm-processor.default.svc.cluster.local:8080/process"
	}
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1alpha1.NetworkIntent{}).
		Complete(r)
}
