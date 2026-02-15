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
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	intentv1alpha1 "github.com/thc1006/nephoran-intent-operator/api/intent/v1alpha1"
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

	// Check if the object is being deleted.

	if !networkIntent.ObjectMeta.DeletionTimestamp.IsZero() {

		log.Info("NetworkIntent is being deleted", "name", networkIntent.Name)

		return ctrl.Result{}, nil

	}

	// Validate the intent field.

	if len(networkIntent.Spec.Source) == 0 {
		return r.updateStatus(ctx, networkIntent, "Error", "Source field cannot be empty", networkIntent.Generation)
	}

	// Initial validation passed.

	if _, err := r.updateStatus(ctx, networkIntent, "Validated", "Intent validated successfully", networkIntent.Generation); err != nil {
		return ctrl.Result{}, err
	}

	// If A1 integration is enabled, create A1 policy
	if r.EnableA1Integration {
		log.Info("Converting NetworkIntent to A1 policy", "name", networkIntent.Name)

		// Convert NetworkIntent to A1 policy
		a1Policy, err := convertToA1Policy(networkIntent)
		if err != nil {
			log.Error(err, "Failed to convert NetworkIntent to A1 policy")
			return r.updateStatus(ctx, networkIntent, "Error",
				fmt.Sprintf("Failed to convert to A1 policy: %v", err), networkIntent.Generation)
		}

		// Create A1 policy via A1 Mediator API
		if err := r.createA1Policy(ctx, networkIntent, a1Policy); err != nil {
			log.Error(err, "Failed to create A1 policy")
			return r.updateStatus(ctx, networkIntent, "Error",
				fmt.Sprintf("Failed to create A1 policy: %v", err), networkIntent.Generation)
		}

		log.Info("A1 policy created successfully", "name", networkIntent.Name)
		if _, err := r.updateStatus(ctx, networkIntent, "Deployed",
			"Intent deployed successfully via A1 policy", networkIntent.Generation); err != nil {
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
	policy := &A1Policy{
		Scope: A1PolicyScope{
			Target:     getStringField(networkIntent, "target"),
			Namespace:  getStringField(networkIntent, "namespace"),
			IntentType: getStringField(networkIntent, "intentType"),
		},
		QoSObjectives: A1QoSObjectives{
			Replicas: getInt32Field(networkIntent, "replicas"),
		},
	}

	// Extract scaling parameters if present
	if scalingParams := getObjectField(networkIntent, "scalingParameters"); scalingParams != nil {
		if autoscalingPolicy := scalingParams["autoscalingPolicy"]; autoscalingPolicy != nil {
			if asMap, ok := autoscalingPolicy.(map[string]interface{}); ok {
				if minReplicas, ok := asMap["minReplicas"].(float64); ok {
					policy.QoSObjectives.MinReplicas = int32(minReplicas)
				}
				if maxReplicas, ok := asMap["maxReplicas"].(float64); ok {
					policy.QoSObjectives.MaxReplicas = int32(maxReplicas)
				}
				if metricThresholds, ok := asMap["metricThresholds"].([]interface{}); ok {
					for _, mt := range metricThresholds {
						if mtMap, ok := mt.(map[string]interface{}); ok {
							threshold := A1MetricThreshold{}
							if metricType, ok := mtMap["type"].(string); ok {
								threshold.Type = metricType
							}
							if value, ok := mtMap["value"].(float64); ok {
								threshold.Value = int64(value)
							}
							policy.QoSObjectives.MetricThresholds = append(policy.QoSObjectives.MetricThresholds, threshold)
						}
					}
				}
			}
		}
	}

	// Extract network parameters if present
	if networkParams := getObjectField(networkIntent, "networkParameters"); networkParams != nil {
		if networkSliceID, ok := networkParams["networkSliceId"].(string); ok {
			policy.QoSObjectives.NetworkSliceID = networkSliceID
		}
		if qosProfile := networkParams["qosProfile"]; qosProfile != nil {
			if qosMap, ok := qosProfile.(map[string]interface{}); ok {
				if priority, ok := qosMap["priority"].(float64); ok {
					policy.QoSObjectives.Priority = int32(priority)
				}
				if maxDataRate, ok := qosMap["maximumDataRate"].(string); ok {
					policy.QoSObjectives.MaximumDataRate = maxDataRate
				}
			}
		}
	}

	return policy, nil
}

// Helper functions to extract fields from NetworkIntent spec (using runtime.RawExtension)
func getStringField(networkIntent *intentv1alpha1.NetworkIntent, fieldName string) string {
	spec := networkIntent.Spec
	// Access spec fields through reflection or type assertion
	// For now, we'll use a simpler approach assuming the spec is a map
	if specBytes, err := json.Marshal(spec); err == nil {
		var specMap map[string]interface{}
		if err := json.Unmarshal(specBytes, &specMap); err == nil {
			if val, ok := specMap[fieldName].(string); ok {
				return val
			}
		}
	}
	return ""
}

func getInt32Field(networkIntent *intentv1alpha1.NetworkIntent, fieldName string) int32 {
	if specBytes, err := json.Marshal(networkIntent.Spec); err == nil {
		var specMap map[string]interface{}
		if err := json.Unmarshal(specBytes, &specMap); err == nil {
			if val, ok := specMap[fieldName].(float64); ok {
				return int32(val)
			}
		}
	}
	return 0
}

func getObjectField(networkIntent *intentv1alpha1.NetworkIntent, fieldName string) map[string]interface{} {
	if specBytes, err := json.Marshal(networkIntent.Spec); err == nil {
		var specMap map[string]interface{}
		if err := json.Unmarshal(specBytes, &specMap); err == nil {
			if val, ok := specMap[fieldName].(map[string]interface{}); ok {
				return val
			}
		}
	}
	return nil
}

// createA1Policy creates an A1 policy via the A1 Mediator API
func (r *NetworkIntentReconciler) createA1Policy(ctx context.Context, networkIntent *intentv1alpha1.NetworkIntent, policy *A1Policy) error {
	log := log.FromContext(ctx)

	// Convert policy to JSON
	policyJSON, err := json.Marshal(policy)
	if err != nil {
		return fmt.Errorf("failed to marshal A1 policy: %w", err)
	}

	// Determine A1 Mediator URL
	a1URL := r.A1MediatorURL
	if a1URL == "" {
		// Default to in-cluster service
		a1URL = "http://service-ricplt-a1mediator-http.ricplt:10000"
	}

	// Policy type ID (using 100 as established in testing)
	policyTypeID := "100"
	// Generate policy instance ID from NetworkIntent name
	policyInstanceID := fmt.Sprintf("policy-%s", networkIntent.Name)

	// Construct A1 v2 API endpoint
	apiEndpoint := fmt.Sprintf("%s/A1-P/v2/policytypes/%s/policies/%s",
		a1URL, policyTypeID, policyInstanceID)

	log.Info("Creating A1 policy",
		"endpoint", apiEndpoint,
		"policyTypeID", policyTypeID,
		"policyInstanceID", policyInstanceID)

	// Create HTTP client with timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	// Create PUT request (A1 v2 uses PUT for policy creation)
	httpReq, err := http.NewRequestWithContext(ctx, "PUT", apiEndpoint, bytes.NewBuffer(policyJSON))
	if err != nil {
		return fmt.Errorf("failed to create A1 request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")

	// Send the request
	resp, err := client.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to send A1 request: %w", err)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.Error(err, "Failed to close A1 response body")
		}
	}()

	// Check response status
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := json.Marshal(resp.Body)
		return fmt.Errorf("A1 API returned error status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	log.Info("A1 policy created successfully",
		"policyInstanceID", policyInstanceID,
		"statusCode", resp.StatusCode)

	return nil
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
