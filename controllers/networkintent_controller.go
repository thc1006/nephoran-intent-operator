/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

// Package controllers implements the core Kubernetes controllers for the Nephoran Intent Operator.

package controllers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
)

const (
	// NetworkIntentFinalizerName is the finalizer added to NetworkIntent resources
	NetworkIntentFinalizerName = "intent.nephoran.com/a1-policy-cleanup"

	// A1CleanupRetriesAnnotation tracks how many times A1 cleanup was attempted
	A1CleanupRetriesAnnotation = "intent.nephoran.com/a1-cleanup-retries"

	// maxA1CleanupRetries is the maximum number of A1 cleanup attempts before
	// removing the finalizer anyway to prevent blocking deletion indefinitely
	maxA1CleanupRetries = 3
)

// NetworkIntentReconciler reconciles a NetworkIntent object.

type NetworkIntentReconciler struct {
	client.Client

	Scheme *runtime.Scheme

	Log logr.Logger

	EnableLLMIntent bool

	LLMProcessorURL string

	// A1MediatorURL is the endpoint for the O-RAN A1 Mediator
	A1MediatorURL string

	// EnableA1Integration enables A1 policy creation
	EnableA1Integration bool
}

// LLMRequest represents the request to the LLM processor.

type LLMRequest struct {
	Intent string `json:"intent"`
}

// LLMResponse represents the response from the LLM processor.

type LLMResponse struct {
	Success bool `json:"success"`

	Message string `json:"message,omitempty"`

	Error string `json:"error,omitempty"`
}

// A1Policy represents an A1 policy conforming to policy type 100
type A1Policy struct {
	Scope         A1PolicyScope   `json:"scope"`
	QoSObjectives A1QoSObjectives `json:"qosObjectives"`
}

// A1PolicyScope defines the target scope for the policy
type A1PolicyScope struct {
	Target     string `json:"target"`
	Namespace  string `json:"namespace"`
	IntentType string `json:"intentType"`
}

// A1QoSObjectives defines QoS and scaling objectives
type A1QoSObjectives struct {
	Replicas         int32               `json:"replicas"`
	MinReplicas      int32               `json:"minReplicas,omitempty"`
	MaxReplicas      int32               `json:"maxReplicas,omitempty"`
	NetworkSliceID   string              `json:"networkSliceId,omitempty"`
	Priority         int32               `json:"priority,omitempty"`
	MaximumDataRate  string              `json:"maximumDataRate,omitempty"`
	MetricThresholds []A1MetricThreshold `json:"metricThresholds,omitempty"`
}

// A1MetricThreshold defines a metric-based scaling threshold
type A1MetricThreshold struct {
	Type  string `json:"type"`
	Value int64  `json:"value"`
}

//+kubebuilder:rbac:groups=intent.nephio.org,resources=networkintents,verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups=intent.nephio.org,resources=networkintents/status,verbs=get;update;patch

//+kubebuilder:rbac:groups=intent.nephio.org,resources=networkintents/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to.

// move the current state of the cluster closer to the desired state.

func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Fetch the NetworkIntent instance.

	networkIntent := &intentv1alpha1.NetworkIntent{}

	err := r.Get(ctx, req.NamespacedName, networkIntent)
	if err != nil {

		if errors.IsNotFound(err) {

			// Object not found, return. Created objects are automatically garbage collected.

			// For additional cleanup logic use finalizers.

			log.Info("NetworkIntent resource not found. Ignoring since object must be deleted")

			return ctrl.Result{}, nil

		}

		// Error reading the object - requeue the request.

		log.Error(err, "Failed to get NetworkIntent")

		return ctrl.Result{}, err

	}

	// Check if the object is being deleted and handle finalizer

	if !networkIntent.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(networkIntent, NetworkIntentFinalizerName) {
			// Our finalizer is present, so handle A1 policy cleanup
			log.Info("Deleting A1 policy for NetworkIntent", "name", networkIntent.Name)

			if r.EnableA1Integration {
				// Count previous cleanup attempts from annotation
				retries := 0
				if ann := networkIntent.Annotations[A1CleanupRetriesAnnotation]; ann != "" {
					retries, _ = strconv.Atoi(ann)
				}

				if retries >= maxA1CleanupRetries {
					log.Info("Max A1 cleanup retries reached, removing finalizer anyway",
						"name", networkIntent.Name, "retries", retries)
				} else if err := r.deleteA1Policy(ctx, networkIntent); err != nil {
					log.Error(err, "Failed to delete A1 policy, incrementing retry counter",
						"retry", retries+1, "maxRetries", maxA1CleanupRetries)
					// Record the retry in the annotation
					if networkIntent.Annotations == nil {
						networkIntent.Annotations = map[string]string{}
					}
					networkIntent.Annotations[A1CleanupRetriesAnnotation] = strconv.Itoa(retries + 1)
					if err := r.Update(ctx, networkIntent); err != nil {
						return ctrl.Result{}, err
					}
					return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
				} else {
					log.Info("A1 policy deleted successfully", "name", networkIntent.Name)
				}
			}

			// Remove our finalizer from the list and update it
			controllerutil.RemoveFinalizer(networkIntent, NetworkIntentFinalizerName)
			if err := r.Update(ctx, networkIntent); err != nil {
				return ctrl.Result{}, err
			}
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(networkIntent, NetworkIntentFinalizerName) {
		log.Info("Adding finalizer to NetworkIntent", "name", networkIntent.Name)
		controllerutil.AddFinalizer(networkIntent, NetworkIntentFinalizerName)
		if err := r.Update(ctx, networkIntent); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Validate the intent field.

	if len(networkIntent.Spec.Source) == 0 {
		return r.updateStatus(ctx, networkIntent, "Error", "Source field cannot be empty", networkIntent.Generation)
	}

	// Initial validation passed.

	if _, err := r.updateStatus(ctx, networkIntent, "Validated", "Intent validated successfully", networkIntent.Generation); err != nil {
		return ctrl.Result{}, err
	}

	// If A1 integration is enabled, create/update A1 policy
	if r.EnableA1Integration {
		// Check if this is an update (status already shows Deployed)
		isUpdate := networkIntent.Status.Phase == "Deployed"

		if isUpdate {
			log.Info("Updating A1 policy for modified NetworkIntent", "name", networkIntent.Name)
		} else {
			log.Info("Converting NetworkIntent to A1 policy", "name", networkIntent.Name)
		}

		// Convert NetworkIntent to A1 policy
		a1Policy, err := convertToA1Policy(networkIntent)
		if err != nil {
			log.Error(err, "Failed to convert NetworkIntent to A1 policy")
			return r.updateStatus(ctx, networkIntent, "Error",
				fmt.Sprintf("Failed to convert to A1 policy: %v", err), networkIntent.Generation)
		}

		// Create/Update A1 policy via A1 Mediator API (PUT is idempotent - creates or updates)
		if err := r.createA1Policy(ctx, networkIntent, a1Policy); err != nil {
			log.Error(err, "Failed to create/update A1 policy")
			return r.updateStatus(ctx, networkIntent, "Error",
				fmt.Sprintf("Failed to create/update A1 policy: %v", err), networkIntent.Generation)
		}

		statusMessage := "Intent deployed successfully via A1 policy"
		if isUpdate {
			statusMessage = "Intent updated successfully via A1 policy"
			log.Info("A1 policy updated successfully", "name", networkIntent.Name)
		} else {
			log.Info("A1 policy created successfully", "name", networkIntent.Name)
		}

		if _, err := r.updateStatus(ctx, networkIntent, "Deployed",
			statusMessage, networkIntent.Generation); err != nil {
			return ctrl.Result{}, err
		}

		// A1 integration complete, return
		return ctrl.Result{}, nil
	}

	// If LLM intent processing is enabled, send to LLM processor.

	if r.EnableLLMIntent {

		log.Info("Processing intent with LLM", "source", networkIntent.Spec.Source)

		// Prepare the request.

		llmReq := LLMRequest{
			Intent: networkIntent.Spec.Source,
		}

		jsonData, err := json.Marshal(llmReq)
		if err != nil {

			log.Error(err, "Failed to marshal LLM request")

			return r.updateStatus(ctx, networkIntent, "Error", fmt.Sprintf("Failed to prepare LLM request: %v", err), networkIntent.Generation)

		}

		// Create HTTP client with timeout.

		client := &http.Client{
			Timeout: 15 * time.Second,
		}

		// Determine LLM processor URL.

		llmURL := r.LLMProcessorURL

		if llmURL == "" {
			llmURL = "http://llm-processor:8080/process"
		}

		// Create the request.

		httpReq, err := http.NewRequestWithContext(ctx, "POST", llmURL, bytes.NewBuffer(jsonData))
		if err != nil {

			log.Error(err, "Failed to create HTTP request")

			return r.updateStatus(ctx, networkIntent, "Error", fmt.Sprintf("Failed to create LLM request: %v", err), networkIntent.Generation)

		}

		httpReq.Header.Set("Content-Type", "application/json")

		// Send the request.

		resp, err := client.Do(httpReq)
		if err != nil {

			log.Error(err, "Failed to send request to LLM processor")

			return r.updateStatus(ctx, networkIntent, "Error", fmt.Sprintf("Failed to process intent with LLM: %v", err), networkIntent.Generation)

		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				log.Error(err, "Failed to close response body")
			}
		}()

		// Parse the response.

		var llmResp LLMResponse

		if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {

			log.Error(err, "Failed to decode LLM response")

			return r.updateStatus(ctx, networkIntent, "Error", fmt.Sprintf("Failed to parse LLM response: %v", err), networkIntent.Generation)

		}

		// Update status based on LLM response.

		if llmResp.Success {

			log.Info("LLM processing successful", "message", llmResp.Message)

			return r.updateStatus(ctx, networkIntent, "Processed", "Intent processed successfully by LLM", networkIntent.Generation)

		} else {

			err := fmt.Errorf("LLM processing failed: %s", llmResp.Error)

			log.Error(err, "LLM processing failed")

			return r.updateStatus(ctx, networkIntent, "Error", err.Error(), networkIntent.Generation)

		}

	}

	log.Info("NetworkIntent reconciled successfully", "name", networkIntent.Name)

	return ctrl.Result{}, nil
}

// updateStatus updates the status of the NetworkIntent.

func (r *NetworkIntentReconciler) updateStatus(ctx context.Context, networkIntent *intentv1alpha1.NetworkIntent, phase, message string, generation int64) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	// Update the status fields.

	networkIntent.Status.Phase = phase

	networkIntent.Status.Message = message

	// Update the status subresource.

	if err := r.Status().Update(ctx, networkIntent); err != nil {

		log.Error(err, "Failed to update NetworkIntent status")

		return ctrl.Result{}, err

	}

	return ctrl.Result{}, nil
}

// convertToA1Policy converts a NetworkIntent to an A1 policy conforming to policy type 100
func convertToA1Policy(networkIntent *intentv1alpha1.NetworkIntent) (*A1Policy, error) {
	spec := networkIntent.Spec

	policy := &A1Policy{
		Scope: A1PolicyScope{
			Target:     spec.Target,
			Namespace:  spec.Namespace,
			IntentType: spec.IntentType,
		},
		QoSObjectives: A1QoSObjectives{
			Replicas: spec.Replicas,
		},
	}

	// Extract scaling parameters if present
	if spec.ScalingParameters != nil {
		if spec.ScalingParameters.AutoscalingPolicy != nil {
			// Copy autoscaling values
			policy.QoSObjectives.MinReplicas = spec.ScalingParameters.AutoscalingPolicy.MinReplicas
			policy.QoSObjectives.MaxReplicas = spec.ScalingParameters.AutoscalingPolicy.MaxReplicas

			// Convert metric thresholds
			for _, mt := range spec.ScalingParameters.AutoscalingPolicy.MetricThresholds {
				policy.QoSObjectives.MetricThresholds = append(policy.QoSObjectives.MetricThresholds, A1MetricThreshold{
					Type:  mt.Type,
					Value: mt.Value,
				})
			}
		}
	}

	// Extract network parameters if present
	if spec.NetworkParameters != nil {
		policy.QoSObjectives.NetworkSliceID = spec.NetworkParameters.NetworkSliceID
		if spec.NetworkParameters.QoSProfile != nil {
			policy.QoSObjectives.Priority = spec.NetworkParameters.QoSProfile.Priority
			policy.QoSObjectives.MaximumDataRate = spec.NetworkParameters.QoSProfile.MaximumDataRate
		}
	}

	return policy, nil
}

// a1PolicyRequest is the standard O-RAN Alliance A1AP-v03.01 policy request body.
// Used with PUT /v2/policies/{policyId}
type a1PolicyRequest struct {
	ID      string          `json:"id"`
	Type    string          `json:"type,omitempty"`
	RicID   string          `json:"ric,omitempty"`
	Service string          `json:"service,omitempty"`
	JSON    json.RawMessage `json:"json"`
}

// createA1Policy creates an A1 policy via the Non-RT RIC A1 Policy Management Service.
// Uses the O-RAN Alliance A1AP-v03.01 standard path: PUT /v2/policies/{policyId}
func (r *NetworkIntentReconciler) createA1Policy(ctx context.Context, networkIntent *intentv1alpha1.NetworkIntent, policy *A1Policy) error {
	log := log.FromContext(ctx)

	// Determine A1 endpoint (must be set via flag or env; no hardcoded default)
	a1URL := r.A1MediatorURL
	if a1URL == "" {
		return fmt.Errorf("A1 endpoint not configured: set --a1-endpoint flag or A1_MEDIATOR_URL env var")
	}

	// Generate policy instance ID from NetworkIntent name
	policyInstanceID := fmt.Sprintf("policy-%s", networkIntent.Name)

	// Marshal the inner policy JSON
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal A1 policy body: %w", err)
	}

	// Wrap in standard A1AP request
	req := a1PolicyRequest{
		ID:      policyInstanceID,
		Service: networkIntent.Namespace + "/" + networkIntent.Name,
		JSON:    policyJSON,
	}
	bodyJSON, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal A1 request: %w", err)
	}

	// Standard O-RAN Alliance A1AP-v03.01 path: PUT /v2/policies/{policyId}
	apiEndpoint := fmt.Sprintf("%s/v2/policies/%s", a1URL, policyInstanceID)

	log.Info("Creating A1 policy (O-RAN A1AP-v03.01)",
		"endpoint", apiEndpoint,
		"policyInstanceID", policyInstanceID)

	// Create HTTP client with timeout
	httpClient := &http.Client{Timeout: 10 * time.Second}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, apiEndpoint, bytes.NewBuffer(bodyJSON))
	if err != nil {
		return fmt.Errorf("failed to create A1 request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send A1 request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error(err, "Failed to close A1 response body")
		}
	}()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("A1 API returned error status %d for PUT %s", resp.StatusCode, apiEndpoint)
	}

	log.Info("A1 policy created successfully",
		"policyInstanceID", policyInstanceID,
		"statusCode", resp.StatusCode)

	return nil
}

// deleteA1Policy deletes an A1 policy via the Non-RT RIC A1 Policy Management Service.
// Uses the O-RAN Alliance A1AP-v03.01 standard path: DELETE /v2/policies/{policyId}
func (r *NetworkIntentReconciler) deleteA1Policy(ctx context.Context, networkIntent *intentv1alpha1.NetworkIntent) error {
	log := log.FromContext(ctx)

	// Determine A1 endpoint (must be set via flag or env)
	a1URL := r.A1MediatorURL
	if a1URL == "" {
		return fmt.Errorf("A1 endpoint not configured: set --a1-endpoint flag or A1_MEDIATOR_URL env var")
	}

	policyInstanceID := fmt.Sprintf("policy-%s", networkIntent.Name)

	// Standard O-RAN Alliance A1AP-v03.01 path: DELETE /v2/policies/{policyId}
	apiEndpoint := fmt.Sprintf("%s/v2/policies/%s", a1URL, policyInstanceID)

	log.Info("Deleting A1 policy (O-RAN A1AP-v03.01)",
		"endpoint", apiEndpoint,
		"policyInstanceID", policyInstanceID)

	httpClient := &http.Client{Timeout: 10 * time.Second}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodDelete, apiEndpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create A1 delete request: %w", err)
	}

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send A1 delete request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error(err, "Failed to close A1 response body")
		}
	}()

	// 204 No Content, 202 Accepted, and 404 Not Found are all acceptable for DELETE
	switch resp.StatusCode {
	case http.StatusNoContent, http.StatusAccepted, http.StatusNotFound:
		log.Info("A1 policy deletion completed",
			"policyInstanceID", policyInstanceID,
			"statusCode", resp.StatusCode)
		return nil
	default:
		return fmt.Errorf("A1 API DELETE %s returned unexpected status %d", apiEndpoint, resp.StatusCode)
	}
}

// SetupWithManager sets up the controller with the Manager.

func (r *NetworkIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Check if LLM intent is enabled via environment variable.

	if envVal := os.Getenv("ENABLE_LLM_INTENT"); envVal == "true" {
		r.EnableLLMIntent = true
	}

	// Get LLM processor URL from environment if set.

	if url := os.Getenv("LLM_PROCESSOR_URL"); url != "" {
		r.LLMProcessorURL = url
	}

	// Check if A1 integration is enabled (default: true)
	if envVal := os.Getenv("ENABLE_A1_INTEGRATION"); envVal == "" || envVal == "true" {
		r.EnableA1Integration = true
	}

	// Get A1 Mediator URL from environment if set
	if url := os.Getenv("A1_MEDIATOR_URL"); url != "" {
		r.A1MediatorURL = url
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&intentv1alpha1.NetworkIntent{}).
		Complete(r)
}
