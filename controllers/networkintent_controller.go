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
	"io"
	"net"
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

		// Create HTTP client with timeout and connection pool.

		client := &http.Client{
			Timeout: 15 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 50,
				IdleConnTimeout:     90 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
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

// createA1Policy creates an A1 policy via the O-RAN SC A1 Mediator.
// Uses the O-RAN SC specific path: PUT /A1-P/v2/policytypes/{typeId}/policies/{policyId}
// Note: O-RAN SC A1 Mediator requires capital "A1-P" prefix (not lowercase "a1-p")
func (r *NetworkIntentReconciler) createA1Policy(ctx context.Context, networkIntent *intentv1alpha1.NetworkIntent, policy *A1Policy) error {
	log := log.FromContext(ctx)

	// Determine A1 endpoint (must be set via flag or env; no hardcoded default)
	a1URL := r.A1MediatorURL
	if a1URL == "" {
		return fmt.Errorf("A1 endpoint not configured: set --a1-endpoint flag or A1_MEDIATOR_URL env var")
	}

	// Generate policy instance ID from NetworkIntent name
	policyInstanceID := fmt.Sprintf("policy-%s", networkIntent.Name)

	// Use policy type 100 (test policy type registered in O-RAN SC RIC)
	const policyTypeID = 100

	// Marshal the policy JSON (direct payload, no wrapper for O-RAN SC)
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal A1 policy body: %w", err)
	}

	// O-RAN SC A1 Mediator path: PUT /A1-P/v2/policytypes/{typeId}/policies/{policyId}
	// IMPORTANT: Use capital "A1-P" (not "a1-p" or "/v2/policies/")
	apiEndpoint := fmt.Sprintf("%s/A1-P/v2/policytypes/%d/policies/%s", a1URL, policyTypeID, policyInstanceID)

	log.Info("Creating A1 policy (O-RAN SC A1 Mediator)",
		"endpoint", apiEndpoint,
		"policyTypeID", policyTypeID,
		"policyInstanceID", policyInstanceID)

	// Create HTTP client with timeout and connection pool
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 50,
			IdleConnTimeout:     90 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, apiEndpoint, bytes.NewBuffer(policyJSON))
	if err != nil {
		return fmt.Errorf("failed to create A1 request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send A1 request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error(err, "Failed to close A1 response body")
		}
	}()

	// O-RAN SC A1 Mediator returns 200 OK/201 Created/202 Accepted for successful PUT
	// 202 Accepted indicates async processing, which is valid for policy creation
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("A1 API returned error status %d for PUT %s: %s", resp.StatusCode, apiEndpoint, string(bodyBytes))
	}

	log.Info("A1 policy created successfully",
		"policyInstanceID", policyInstanceID,
		"policyTypeID", policyTypeID,
		"statusCode", resp.StatusCode)

	return nil
}

// deleteA1Policy deletes an A1 policy via the O-RAN SC A1 Mediator.
// Uses the O-RAN SC specific path: DELETE /A1-P/v2/policytypes/{typeId}/policies/{policyId}
func (r *NetworkIntentReconciler) deleteA1Policy(ctx context.Context, networkIntent *intentv1alpha1.NetworkIntent) error {
	log := log.FromContext(ctx)

	// Determine A1 endpoint (must be set via flag or env)
	a1URL := r.A1MediatorURL
	if a1URL == "" {
		return fmt.Errorf("A1 endpoint not configured: set --a1-endpoint flag or A1_MEDIATOR_URL env var")
	}

	policyInstanceID := fmt.Sprintf("policy-%s", networkIntent.Name)
	const policyTypeID = 100

	// O-RAN SC A1 Mediator path: DELETE /A1-P/v2/policytypes/{typeId}/policies/{policyId}
	apiEndpoint := fmt.Sprintf("%s/A1-P/v2/policytypes/%d/policies/%s", a1URL, policyTypeID, policyInstanceID)

	log.Info("Deleting A1 policy (O-RAN SC A1 Mediator)",
		"endpoint", apiEndpoint,
		"policyTypeID", policyTypeID,
		"policyInstanceID", policyInstanceID)

	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 50,
			IdleConnTimeout:     90 * time.Second,
			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
		},
	}

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

	// O-RAN SC returns 200 OK for successful DELETE, 404 if not found (both acceptable)
	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent, http.StatusAccepted, http.StatusNotFound:
		log.Info("A1 policy deletion completed",
			"policyInstanceID", policyInstanceID,
			"policyTypeID", policyTypeID,
			"statusCode", resp.StatusCode)
		return nil
	default:
		bodyBytes, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("A1 API DELETE %s returned unexpected status %d: %s", apiEndpoint, resp.StatusCode, string(bodyBytes))
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
