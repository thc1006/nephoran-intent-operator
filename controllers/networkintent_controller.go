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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/security"
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

// A1APIFormat defines the API path format for A1 interface
type A1APIFormat string

const (
	// A1FormatStandard uses O-RAN Alliance standard paths: /v2/policies/{policyId}
	A1FormatStandard A1APIFormat = "standard"
	// A1FormatLegacy uses O-RAN SC RICPLT paths: /A1-P/v2/policytypes/{typeId}/policies/{policyId}
	A1FormatLegacy A1APIFormat = "legacy"
)

// NetworkIntentReconciler reconciles a NetworkIntent object.

type NetworkIntentReconciler struct {
	client.Client

	Scheme *runtime.Scheme

	Logger logging.Logger

	EnableLLMIntent bool

	LLMProcessorURL string

	// A1MediatorURL is the endpoint for the O-RAN A1 Mediator
	A1MediatorURL string

	// A1APIFormat specifies which API path format to use (standard or legacy)
	// Defaults to "legacy" for backward compatibility with O-RAN SC RICPLT
	A1APIFormat A1APIFormat

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
	start := time.Now()
	logger := r.Logger.ReconcileStart(req.Namespace, req.Name)

	// Track reconciliation duration and outcome
	defer func() {
		duration := time.Since(start).Seconds()
		logger.InfoEvent("Reconciliation completed",
			"durationSeconds", duration,
		)
	}()

	// Fetch the NetworkIntent instance.

	networkIntent := &intentv1alpha1.NetworkIntent{}

	err := r.Get(ctx, req.NamespacedName, networkIntent)
	if err != nil {

		if errors.IsNotFound(err) {

			// Object not found, return. Created objects are automatically garbage collected.

			// For additional cleanup logic use finalizers.

			logger.DebugEvent("NetworkIntent resource not found, ignoring since object must be deleted")

			return ctrl.Result{}, nil

		}

		// Error reading the object - requeue the request.

		duration := time.Since(start).Seconds()
		logger.ReconcileError(req.Namespace, req.Name, err, duration)
		logger.ErrorEvent(err, "Failed to get NetworkIntent from API server")

		return ctrl.Result{}, err

	}

	// Check if the object is being deleted and handle finalizer

	if !networkIntent.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is being deleted
		if controllerutil.ContainsFinalizer(networkIntent, NetworkIntentFinalizerName) {
			// Our finalizer is present, so handle A1 policy cleanup
			logger.InfoEvent("Deleting A1 policy for NetworkIntent during finalizer cleanup")

			if r.EnableA1Integration {
				// Count previous cleanup attempts from annotation
				retries := 0
				if ann := networkIntent.Annotations[A1CleanupRetriesAnnotation]; ann != "" {
					retries, _ = strconv.Atoi(ann)
				}

				if retries >= maxA1CleanupRetries {
					logger.WarnEvent("Max A1 cleanup retries reached, removing finalizer anyway",
						"retries", retries,
						"maxRetries", maxA1CleanupRetries)
				} else if err := r.deleteA1Policy(ctx, networkIntent, logger); err != nil {
					logger.ErrorEvent(err, "Failed to delete A1 policy, incrementing retry counter",
						"retry", retries+1,
						"maxRetries", maxA1CleanupRetries)
					// Record the retry in the annotation
					if networkIntent.Annotations == nil {
						networkIntent.Annotations = map[string]string{}
					}
					networkIntent.Annotations[A1CleanupRetriesAnnotation] = strconv.Itoa(retries + 1)
					if err := r.Update(ctx, networkIntent); err != nil {
						logger.ErrorEvent(err, "Failed to update NetworkIntent with retry annotation")
						return ctrl.Result{}, err
					}
					return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
				} else {
					policyID := fmt.Sprintf("policy-%s", networkIntent.Name)
					logger.A1PolicyDeleted(policyID)
				}
			}

			// Remove our finalizer from the list and update it
			controllerutil.RemoveFinalizer(networkIntent, NetworkIntentFinalizerName)
			if err := r.Update(ctx, networkIntent); err != nil {
				logger.ErrorEvent(err, "Failed to remove finalizer from NetworkIntent")
				return ctrl.Result{}, err
			}
			logger.InfoEvent("Finalizer removed, NetworkIntent deletion complete")
		}

		// Stop reconciliation as the item is being deleted
		return ctrl.Result{}, nil
	}

	// Add finalizer if not present
	if !controllerutil.ContainsFinalizer(networkIntent, NetworkIntentFinalizerName) {
		logger.InfoEvent("Adding finalizer to NetworkIntent")
		controllerutil.AddFinalizer(networkIntent, NetworkIntentFinalizerName)
		if err := r.Update(ctx, networkIntent); err != nil {
			logger.ErrorEvent(err, "Failed to add finalizer to NetworkIntent")
			return ctrl.Result{}, err
		}
	}

	// Validate the intent field.

	logger.DebugEvent("Validating NetworkIntent spec")
	if len(networkIntent.Spec.Source) == 0 {
		logger.ErrorEvent(fmt.Errorf("source field empty"), "NetworkIntent validation failed")
		return r.updateStatus(ctx, networkIntent, "Error", "Source field cannot be empty", networkIntent.Generation, logger)
	}

	// Initial validation passed.

	logger.InfoEvent("NetworkIntent validation successful")
	if _, err := r.updateStatus(ctx, networkIntent, "Validated", "Intent validated successfully", networkIntent.Generation, logger); err != nil {
		return ctrl.Result{}, err
	}

	// If A1 integration is enabled, create/update A1 policy
	if r.EnableA1Integration {
		// Check if this is an update (status already shows Deployed)
		isUpdate := networkIntent.Status.Phase == "Deployed"

		if isUpdate {
			logger.InfoEvent("Updating A1 policy for modified NetworkIntent")
		} else {
			logger.InfoEvent("Converting NetworkIntent to A1 policy")
		}

		// Convert NetworkIntent to A1 policy
		logger.DebugEvent("Converting NetworkIntent spec to A1 policy format",
			"intentType", networkIntent.Spec.IntentType,
			"target", networkIntent.Spec.Target,
			"replicas", networkIntent.Spec.Replicas)
		a1Policy, err := convertToA1Policy(networkIntent)
		if err != nil {
			logger.ErrorEvent(err, "Failed to convert NetworkIntent to A1 policy")
			return r.updateStatus(ctx, networkIntent, "Error",
				fmt.Sprintf("Failed to convert to A1 policy: %v", err), networkIntent.Generation, logger)
		}

		// Create/Update A1 policy via A1 Mediator API (PUT is idempotent - creates or updates)
		if err := r.createA1Policy(ctx, networkIntent, a1Policy, logger); err != nil {
			logger.ErrorEvent(err, "Failed to create/update A1 policy")
			return r.updateStatus(ctx, networkIntent, "Error",
				fmt.Sprintf("Failed to create/update A1 policy: %v", err), networkIntent.Generation, logger)
		}

		// Record the A1 endpoint we successfully contacted
		r.recordObservedEndpoint(networkIntent, "a1", r.A1MediatorURL)

		statusMessage := "Intent deployed successfully via A1 policy"
		if isUpdate {
			statusMessage = "Intent updated successfully via A1 policy"
			logger.InfoEvent("A1 policy updated successfully")
		} else {
			policyID := fmt.Sprintf("policy-%s", networkIntent.Name)
			logger.A1PolicyCreated(policyID, networkIntent.Spec.IntentType)
		}

		if _, err := r.updateStatus(ctx, networkIntent, "Deployed",
			statusMessage, networkIntent.Generation, logger); err != nil {
			return ctrl.Result{}, err
		}

		// A1 integration complete, return success
		duration := time.Since(start).Seconds()
		logger.ReconcileSuccess(req.Namespace, req.Name, duration)
		return ctrl.Result{}, nil
	}

	// If LLM intent processing is enabled, send to LLM processor.

	if r.EnableLLMIntent {

		logger.InfoEvent("Processing intent with LLM",
			"source", networkIntent.Spec.Source,
			"sourceLength", len(networkIntent.Spec.Source))

		// Prepare the request.

		llmReq := LLMRequest{
			Intent: networkIntent.Spec.Source,
		}

		jsonData, err := json.Marshal(llmReq)
		if err != nil {

			logger.ErrorEvent(err, "Failed to marshal LLM request")

			return r.updateStatus(ctx, networkIntent, "Error", fmt.Sprintf("Failed to prepare LLM request: %v", err), networkIntent.Generation, logger)

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

		logger.DebugEvent("Sending request to LLM processor",
			"url", llmURL,
			"timeout", "15s")

		// Create the request.

		httpReq, err := http.NewRequestWithContext(ctx, "POST", llmURL, bytes.NewBuffer(jsonData))
		if err != nil {

			logger.ErrorEvent(err, "Failed to create HTTP request for LLM processor")

			return r.updateStatus(ctx, networkIntent, "Error", fmt.Sprintf("Failed to create LLM request: %v", err), networkIntent.Generation, logger)

		}

		httpReq.Header.Set("Content-Type", "application/json")

		// Send the request.

		llmStart := time.Now()
		resp, err := client.Do(httpReq)
		llmDuration := time.Since(llmStart).Seconds()

		if err != nil {

			logger.HTTPError("POST", llmURL, 0, err, llmDuration)
			logger.ErrorEvent(err, "Failed to send request to LLM processor")

			return r.updateStatus(ctx, networkIntent, "Error", fmt.Sprintf("Failed to process intent with LLM: %v", err), networkIntent.Generation, logger)

		}

		defer func() {
			if err := resp.Body.Close(); err != nil {
				logger.ErrorEvent(err, "Failed to close LLM response body")
			}
		}()

		logger.HTTPRequest("POST", llmURL, resp.StatusCode, llmDuration)

		// Parse the response.

		var llmResp LLMResponse

		if err := json.NewDecoder(resp.Body).Decode(&llmResp); err != nil {

			logger.ErrorEvent(err, "Failed to decode LLM response",
				"statusCode", resp.StatusCode)

			return r.updateStatus(ctx, networkIntent, "Error", fmt.Sprintf("Failed to parse LLM response: %v", err), networkIntent.Generation, logger)

		}

		// Update status based on LLM response.

		if llmResp.Success {

			logger.InfoEvent("LLM processing successful",
				"message", llmResp.Message,
				"durationSeconds", llmDuration)

			// Record the LLM endpoint we successfully contacted
			r.recordObservedEndpoint(networkIntent, "llm", llmURL)

			duration := time.Since(start).Seconds()
			logger.ReconcileSuccess(req.Namespace, req.Name, duration)
			return r.updateStatus(ctx, networkIntent, "Processed", "Intent processed successfully by LLM", networkIntent.Generation, logger)

		} else {

			err := fmt.Errorf("LLM processing failed: %s", llmResp.Error)

			logger.ErrorEvent(err, "LLM processing failed",
				"llmError", llmResp.Error)

			return r.updateStatus(ctx, networkIntent, "Error", err.Error(), networkIntent.Generation, logger)

		}

	}

	// Neither A1 nor LLM enabled, just mark as validated
	duration := time.Since(start).Seconds()
	logger.ReconcileSuccess(req.Namespace, req.Name, duration)
	logger.InfoEvent("NetworkIntent reconciled successfully (no A1/LLM integration enabled)")

	return ctrl.Result{}, nil
}

// recordObservedEndpoint adds or updates an endpoint in the NetworkIntent status.
// This provides observability into which external services the operator actually contacted.
func (r *NetworkIntentReconciler) recordObservedEndpoint(networkIntent *intentv1alpha1.NetworkIntent, name, url string) {
	now := metav1.Now()

	// Find existing endpoint or add new one
	found := false
	for i := range networkIntent.Status.ObservedEndpoints {
		if networkIntent.Status.ObservedEndpoints[i].Name == name {
			networkIntent.Status.ObservedEndpoints[i].URL = url
			networkIntent.Status.ObservedEndpoints[i].LastContactedAt = &now
			found = true
			break
		}
	}

	if !found {
		networkIntent.Status.ObservedEndpoints = append(networkIntent.Status.ObservedEndpoints,
			intentv1alpha1.ObservedEndpoint{
				Name:            name,
				URL:             url,
				LastContactedAt: &now,
			})
	}
}

// updateStatus updates the status of the NetworkIntent.

func (r *NetworkIntentReconciler) updateStatus(ctx context.Context, networkIntent *intentv1alpha1.NetworkIntent, phase, message string, generation int64, logger logging.Logger) (ctrl.Result, error) {

	logger.DebugEvent("Updating NetworkIntent status",
		"phase", phase,
		"generation", generation)

	// Update the status fields.

	networkIntent.Status.Phase = phase

	networkIntent.Status.Message = message

	// Update the status subresource.

	if err := r.Status().Update(ctx, networkIntent); err != nil {

		logger.ErrorEvent(err, "Failed to update NetworkIntent status",
			"phase", phase,
			"message", message)

		return ctrl.Result{}, err

	}

	logger.InfoEvent("NetworkIntent status updated successfully",
		"phase", phase)

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

// createA1Policy creates an A1 policy via the A1 interface.
// Supports both O-RAN Alliance standard format and O-RAN SC RICPLT legacy format.
// Format is controlled by r.A1APIFormat (standard or legacy).
func (r *NetworkIntentReconciler) createA1Policy(ctx context.Context, networkIntent *intentv1alpha1.NetworkIntent, policy *A1Policy, logger logging.Logger) error {

	// Determine A1 endpoint (must be set via flag or env; no hardcoded default)
	a1URL := r.A1MediatorURL
	if a1URL == "" {
		err := fmt.Errorf("A1 endpoint not configured: set --a1-endpoint flag or A1_MEDIATOR_URL env var")
		logger.ErrorEvent(err, "A1 endpoint configuration missing")
		return err
	}

	// Generate policy instance ID from NetworkIntent name
	policyInstanceID := fmt.Sprintf("policy-%s", networkIntent.Name)

	// Use policy type 100 (test policy type registered in O-RAN SC RIC)
	const policyTypeID = 100

	// Marshal the policy JSON (direct payload, no wrapper for O-RAN SC)
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		logger.ErrorEvent(err, "Failed to marshal A1 policy body")
		return fmt.Errorf("failed to marshal A1 policy body: %w", err)
	}

	// Build API endpoint based on configured format
	var apiEndpoint string
	apiFormat := r.A1APIFormat
	if apiFormat == "" {
		apiFormat = A1FormatLegacy // default to legacy for backward compatibility
	}

	switch apiFormat {
	case A1FormatStandard:
		// O-RAN Alliance A1AP-v03.01 standard: PUT /v2/policies/{policyId}
		apiEndpoint = fmt.Sprintf("%s/v2/policies/%s", a1URL, policyInstanceID)
	case A1FormatLegacy:
		// O-RAN SC RICPLT legacy: PUT /A1-P/v2/policytypes/{typeId}/policies/{policyId}
		apiEndpoint = fmt.Sprintf("%s/A1-P/v2/policytypes/%d/policies/%s", a1URL, policyTypeID, policyInstanceID)
	default:
		return fmt.Errorf("unknown A1 API format: %s (valid: standard, legacy)", apiFormat)
	}

	logger.InfoEvent("Creating A1 policy",
		"endpoint", apiEndpoint,
		"apiFormat", apiFormat,
		"policyTypeID", policyTypeID,
		"policyInstanceID", policyInstanceID,
		"target", policy.Scope.Target,
		"replicas", policy.QoSObjectives.Replicas)

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
		logger.ErrorEvent(err, "Failed to create A1 HTTP request")
		return fmt.Errorf("failed to create A1 request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	a1Start := time.Now()
	resp, err := httpClient.Do(httpReq)
	a1Duration := time.Since(a1Start).Seconds()

	if err != nil {
		logger.HTTPError(http.MethodPut, apiEndpoint, 0, err, a1Duration)
		logger.ErrorEvent(err, "Failed to send A1 request")
		return fmt.Errorf("failed to send A1 request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.ErrorEvent(err, "Failed to close A1 response body")
		}
	}()

	logger.HTTPRequest(http.MethodPut, apiEndpoint, resp.StatusCode, a1Duration)

	// O-RAN SC A1 Mediator returns 200 OK/201 Created/202 Accepted for successful PUT
	// 202 Accepted indicates async processing, which is valid for policy creation
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.ErrorEvent(fmt.Errorf("A1 API error"), "A1 API returned error status",
			"statusCode", resp.StatusCode,
			"endpoint", apiEndpoint,
			"responseBody", string(bodyBytes))
		return fmt.Errorf("A1 API returned error status %d for PUT %s: %s", resp.StatusCode, apiEndpoint, string(bodyBytes))
	}

	logger.InfoEvent("A1 policy created successfully",
		"policyInstanceID", policyInstanceID,
		"policyTypeID", policyTypeID,
		"statusCode", resp.StatusCode,
		"durationSeconds", a1Duration)

	return nil
}

// deleteA1Policy deletes an A1 policy via the A1 interface.
// Supports both O-RAN Alliance standard format and O-RAN SC RICPLT legacy format.
func (r *NetworkIntentReconciler) deleteA1Policy(ctx context.Context, networkIntent *intentv1alpha1.NetworkIntent, logger logging.Logger) error {

	// Determine A1 endpoint (must be set via flag or env)
	a1URL := r.A1MediatorURL
	if a1URL == "" {
		err := fmt.Errorf("A1 endpoint not configured: set --a1-endpoint flag or A1_MEDIATOR_URL env var")
		logger.ErrorEvent(err, "A1 endpoint configuration missing for deletion")
		return err
	}

	policyInstanceID := fmt.Sprintf("policy-%s", networkIntent.Name)
	const policyTypeID = 100

	// Build API endpoint based on configured format
	var apiEndpoint string
	apiFormat := r.A1APIFormat
	if apiFormat == "" {
		apiFormat = A1FormatLegacy // default to legacy for backward compatibility
	}

	switch apiFormat {
	case A1FormatStandard:
		// O-RAN Alliance A1AP-v03.01 standard: DELETE /v2/policies/{policyId}
		apiEndpoint = fmt.Sprintf("%s/v2/policies/%s", a1URL, policyInstanceID)
	case A1FormatLegacy:
		// O-RAN SC RICPLT legacy: DELETE /A1-P/v2/policytypes/{typeId}/policies/{policyId}
		apiEndpoint = fmt.Sprintf("%s/A1-P/v2/policytypes/%d/policies/%s", a1URL, policyTypeID, policyInstanceID)
	default:
		return fmt.Errorf("unknown A1 API format: %s (valid: standard, legacy)", apiFormat)
	}

	logger.InfoEvent("Deleting A1 policy",
		"endpoint", apiEndpoint,
		"apiFormat", apiFormat,
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
		logger.ErrorEvent(err, "Failed to create A1 DELETE request")
		return fmt.Errorf("failed to create A1 delete request: %w", err)
	}

	deleteStart := time.Now()
	resp, err := httpClient.Do(httpReq)
	deleteDuration := time.Since(deleteStart).Seconds()

	if err != nil {
		logger.HTTPError(http.MethodDelete, apiEndpoint, 0, err, deleteDuration)
		logger.ErrorEvent(err, "Failed to send A1 delete request")
		return fmt.Errorf("failed to send A1 delete request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			logger.ErrorEvent(err, "Failed to close A1 DELETE response body")
		}
	}()

	logger.HTTPRequest(http.MethodDelete, apiEndpoint, resp.StatusCode, deleteDuration)

	// O-RAN SC returns 200 OK for successful DELETE, 404 if not found (both acceptable)
	switch resp.StatusCode {
	case http.StatusOK, http.StatusNoContent, http.StatusAccepted, http.StatusNotFound:
		logger.InfoEvent("A1 policy deletion completed",
			"policyInstanceID", policyInstanceID,
			"policyTypeID", policyTypeID,
			"statusCode", resp.StatusCode,
			"durationSeconds", deleteDuration)
		return nil
	default:
		bodyBytes, _ := io.ReadAll(resp.Body)
		logger.ErrorEvent(fmt.Errorf("A1 DELETE failed"), "A1 API DELETE returned unexpected status",
			"statusCode", resp.StatusCode,
			"endpoint", apiEndpoint,
			"responseBody", string(bodyBytes))
		return fmt.Errorf("A1 API DELETE %s returned unexpected status %d: %s", apiEndpoint, resp.StatusCode, string(bodyBytes))
	}
}

// SetupWithManager sets up the controller with the Manager.

func (r *NetworkIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Initialize structured logger
	r.Logger = logging.NewLogger(logging.ComponentController)

	r.Logger.InfoEvent("Setting up NetworkIntent controller",
		"component", logging.ComponentController)

	// Check if LLM intent is enabled via environment variable.

	if envVal := os.Getenv("ENABLE_LLM_INTENT"); envVal == "true" {
		r.EnableLLMIntent = true
		r.Logger.InfoEvent("LLM intent processing enabled")
	}

	// Get LLM processor URL from environment if set.
	// SSRF protection: validate user-provided endpoint URLs at startup.
	ssrfValidator := security.NewSSRFValidator(security.SSRFValidatorConfig{
		AllowPrivateIPs: true, // In-cluster services use private IPs
	})

	if url := os.Getenv("LLM_PROCESSOR_URL"); url != "" {
		if err := ssrfValidator.ValidateEndpointURL(url); err != nil {
			r.Logger.ErrorEvent(err, "LLM_PROCESSOR_URL failed SSRF validation",
				"url", url)
			return fmt.Errorf("LLM_PROCESSOR_URL failed SSRF validation: %w", err)
		}
		r.LLMProcessorURL = url
		r.Logger.InfoEvent("LLM processor URL configured",
			"url", url)
	}

	// Check if A1 integration is enabled (default: true)
	if envVal := os.Getenv("ENABLE_A1_INTEGRATION"); envVal == "" || envVal == "true" {
		r.EnableA1Integration = true
		r.Logger.InfoEvent("A1 integration enabled")
	}

	// Get A1 Mediator URL from environment if set
	if url := os.Getenv("A1_MEDIATOR_URL"); url != "" {
		if err := ssrfValidator.ValidateEndpointURL(url); err != nil {
			r.Logger.ErrorEvent(err, "A1_MEDIATOR_URL failed SSRF validation",
				"url", url)
			return fmt.Errorf("A1_MEDIATOR_URL failed SSRF validation: %w", err)
		}
		r.A1MediatorURL = url
		r.Logger.InfoEvent("A1 Mediator URL configured",
			"url", url)
	}

	// Get A1 API format from environment (default: legacy for backward compatibility)
	apiFormat := os.Getenv("A1_API_FORMAT")
	if apiFormat == "" {
		apiFormat = string(A1FormatLegacy)
	}
	switch A1APIFormat(apiFormat) {
	case A1FormatStandard:
		r.A1APIFormat = A1FormatStandard
		r.Logger.InfoEvent("A1 API format configured",
			"format", "standard",
			"paths", "/v2/policies/{policyId}")
	case A1FormatLegacy:
		r.A1APIFormat = A1FormatLegacy
		r.Logger.InfoEvent("A1 API format configured",
			"format", "legacy",
			"paths", "/A1-P/v2/policytypes/{typeId}/policies/{policyId}")
	default:
		return fmt.Errorf("invalid A1_API_FORMAT: %s (valid: standard, legacy)", apiFormat)
	}

	r.Logger.InfoEvent("NetworkIntent controller setup complete",
		"enableLLM", r.EnableLLMIntent,
		"enableA1", r.EnableA1Integration)

	return ctrl.NewControllerManagedBy(mgr).
		For(&intentv1alpha1.NetworkIntent{}).
		Complete(r)
}
