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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
)

// NetworkIntentReconciler reconciles a NetworkIntent object.

type NetworkIntentReconciler struct {
	client.Client

	Scheme *runtime.Scheme

	Log logr.Logger

	EnableLLMIntent bool

	LLMProcessorURL string
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

//+kubebuilder:rbac:groups=nephoran.io,resources=networkintents,verbs=get;list;watch;create;update;patch;delete

//+kubebuilder:rbac:groups=nephoran.io,resources=networkintents/status,verbs=get;update;patch

//+kubebuilder:rbac:groups=nephoran.io,resources=networkintents/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to.

// move the current state of the cluster closer to the desired state.

func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log := log.FromContext(ctx)

	// Fetch the NetworkIntent instance.

	networkIntent := &nephoranv1.NetworkIntent{}

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

	if networkIntent.Spec.Intent == "" {

		return r.updateStatus(ctx, networkIntent, "Error", "Intent field cannot be empty", networkIntent.Generation)

	}

	// Initial validation passed.

	if _, err := r.updateStatus(ctx, networkIntent, "Validated", "Intent validated successfully", networkIntent.Generation); err != nil {

		return ctrl.Result{}, err

	}

	// If LLM intent processing is enabled, send to LLM processor.

	if r.EnableLLMIntent {

		log.Info("Processing intent with LLM", "intent", networkIntent.Spec.Intent)

		// Prepare the request.

		llmReq := LLMRequest{

			Intent: networkIntent.Spec.Intent,
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

func (r *NetworkIntentReconciler) updateStatus(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, phase, message string, generation int64) (ctrl.Result, error) {

	log := log.FromContext(ctx)

	// Update the status fields.

	networkIntent.Status.Phase = nephoranv1.NetworkIntentPhase(phase)

	networkIntent.Status.LastMessage = message

	networkIntent.Status.ObservedGeneration = generation

	networkIntent.Status.LastUpdateTime = metav1.Now()

	// Update the status subresource.

	if err := r.Status().Update(ctx, networkIntent); err != nil {

		log.Error(err, "Failed to update NetworkIntent status")

		return ctrl.Result{}, err

	}

	return ctrl.Result{}, nil

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

	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		Complete(r)

}
