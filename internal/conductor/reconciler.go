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

// Package conductor implements the core reconciliation logic for network intent processing.
package conductor

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// PorchExecutor defines the interface for executing porch CLI commands.

// This allows for dependency injection during testing.

type PorchExecutor interface {
	ExecutePorch(ctx context.Context, porchPath string, args []string, outputDir, intentFile, mode string) error
}

// defaultPorchExecutor provides the real porch CLI execution.

type defaultPorchExecutor struct{}

// ExecutePorch performs executeporch operation.

func (dpe *defaultPorchExecutor) ExecutePorch(ctx context.Context, porchPath string, args []string, outputDir, intentFile, mode string) error {
	cmd := exec.CommandContext(ctx, porchPath, args...)

	cmd.Dir = outputDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("porch CLI failed: %w, output: %s", err, string(output))
	}

	return nil
}

// NetworkIntentReconciler reconciles a NetworkIntent object.

type NetworkIntentReconciler struct {
	client.Client

	Scheme *runtime.Scheme

	Log logr.Logger

	PorchPath string

	Mode string

	OutputDir string

	porchExecutor PorchExecutor // Injected for testing
}

// IntentJSON represents the intent JSON structure matching docs/contracts/intent.schema.json.

type IntentJSON struct {
	IntentType string `json:"intent_type"`

	Target string `json:"target"`

	Namespace string `json:"namespace"`

	Replicas int `json:"replicas"`

	Reason string `json:"reason,omitempty"`

	Source string `json:"source,omitempty"`

	CorrelationID string `json:"correlation_id,omitempty"`
}

// SetupWithManager sets up the controller with the Manager.

func (r *NetworkIntentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		Complete(r)
}

// Reconcile handles NetworkIntent resource changes.

func (r *NetworkIntentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("networkintent", req.NamespacedName)

	logger.Info("Reconciling NetworkIntent")

	// Fetch the NetworkIntent instance.

	var networkIntent nephoranv1.NetworkIntent

	if err := r.Get(ctx, req.NamespacedName, &networkIntent); err != nil {

		if client.IgnoreNotFound(err) == nil {

			logger.Info("NetworkIntent not found, likely deleted")

			return ctrl.Result{}, nil

		}

		logger.Error(err, "Failed to get NetworkIntent")

		return ctrl.Result{}, err

	}

	// Parse the intent string to extract scaling information.

	intentData, err := r.parseIntentString(networkIntent.Spec.Intent, networkIntent.Namespace)
	if err != nil {

		logger.Error(err, "Failed to parse intent string", "intent", networkIntent.Spec.Intent)

		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil // Retry later

	}

	// Add metadata.

	intentData.Source = "user"

	intentData.CorrelationID = fmt.Sprintf("%s-%s-%d",

		networkIntent.Name,

		networkIntent.Namespace,

		time.Now().Unix())

	intentData.Reason = fmt.Sprintf("Generated from NetworkIntent %s/%s",

		networkIntent.Namespace, networkIntent.Name)

	// Create intent JSON file.

	intentFile, err := r.createIntentFile(intentData, networkIntent.Name)
	if err != nil {

		logger.Error(err, "Failed to create intent JSON file")

		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil

	}

	logger.Info("Created intent file", "file", intentFile, "target", intentData.Target, "replicas", intentData.Replicas)

	// Call porch CLI to generate KRM files.

	if err := r.callPorchCLI(ctx, intentFile, logger); err != nil {

		logger.Error(err, "Failed to call porch CLI")

		return ctrl.Result{RequeueAfter: time.Minute * 5}, nil

	}

	logger.Info("Successfully processed NetworkIntent")

	return ctrl.Result{}, nil
}

// parseIntentString parses the natural language intent string to extract scaling parameters.

// This is an MVP implementation using simple regex patterns.

func (r *NetworkIntentReconciler) parseIntentString(intent, namespace string) (*IntentJSON, error) {
	intentData := &IntentJSON{
		IntentType: "scaling",

		Namespace: namespace,
	}

	// Normalize the intent string.

	intent = strings.ToLower(strings.TrimSpace(intent))

	// Pattern to match deployment/target names.

	// Look for patterns like "scale <deployment-name>" or "deployment <name>".

	var matches []string

	// Try different patterns to capture the target name.

	patterns := []*regexp.Regexp{
		regexp.MustCompile(`scale\s+(?:deployment|app|service)\s+([a-z0-9\-]+)`), // "scale deployment web-server"

		regexp.MustCompile(`(?:deployment|app|service)\s+([a-z0-9\-]+)`), // "deployment web-server"

	}

	// First try patterns with explicit keywords.

	for _, pattern := range patterns {

		matches = pattern.FindStringSubmatch(intent)

		if len(matches) > 1 {

			candidate := matches[1]

			// Make sure we didn't capture a keyword itself.

			if candidate != "deployment" && candidate != "app" && candidate != "service" && candidate != "to" {

				intentData.Target = candidate

				break

			}

		}

	}

	// If no match yet, try simpler pattern but exclude common keywords.

	if intentData.Target == "" {

		simplePattern := regexp.MustCompile(`scale\s+([a-z0-9\-]+)`)

		matches = simplePattern.FindStringSubmatch(intent)

		if len(matches) > 1 {

			candidate := matches[1]

			if candidate != "deployment" && candidate != "app" && candidate != "service" && candidate != "to" {
				intentData.Target = candidate
			}

		}

	}

	// Pattern to match replica counts.

	// Look for patterns like "to 5 replicas", "5 instances", "replicas: 3".

	// Order matters - more specific patterns first to avoid false matches.

	replicaPatterns := []*regexp.Regexp{
		regexp.MustCompile(`to\s+(\d+)\s+(?:replicas?|instances?)`),

		regexp.MustCompile(`(\d+)\s+(?:replicas?|instances?)(?:\s|$)`), // Added word boundary

		regexp.MustCompile(`replicas?:?\s*(\d+)`),

		// Remove the greedy pattern that was causing issues.

	}

	for _, pattern := range replicaPatterns {

		matches := pattern.FindStringSubmatch(intent)

		if len(matches) > 1 {
			if count, err := strconv.Atoi(matches[1]); err == nil && count > 0 {
				// Security fix (G115): Validate bounds for replica count
				if count > math.MaxInt32 {
					count = math.MaxInt32
				}
				intentData.Replicas = count

				break

			}
		}

	}

	// Validation.

	if intentData.Target == "" {
		return nil, fmt.Errorf("could not extract deployment target from intent: %s", intent)
	}

	if intentData.Replicas == 0 {
		// Default to 1 if no replica count is specified.

		intentData.Replicas = 1
	}

	if intentData.Replicas > 100 {
		return nil, fmt.Errorf("replica count %d exceeds maximum of 100", intentData.Replicas)
	}

	return intentData, nil
}

// createIntentFile creates an intent JSON file in the output directory.

func (r *NetworkIntentReconciler) createIntentFile(intentData *IntentJSON, name string) (string, error) {
	// Ensure output directory exists.

	if err := os.MkdirAll(r.OutputDir, 0o755); err != nil {
		return "", fmt.Errorf("failed to create output directory: %w", err)
	}

	// Generate filename with timestamp.

	filename := fmt.Sprintf("intent-%s-%d.json", name, time.Now().Unix())

	filepath := filepath.Join(r.OutputDir, filename)

	// Marshal intent to JSON.

	jsonData, err := json.MarshalIndent(intentData, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal intent JSON: %w", err)
	}

	// Write to file.

	if err := os.WriteFile(filepath, jsonData, 0o644); err != nil {
		return "", fmt.Errorf("failed to write intent file: %w", err)
	}

	return filepath, nil
}

// callPorchCLI executes the porch CLI command to generate KRM files.

func (r *NetworkIntentReconciler) callPorchCLI(ctx context.Context, intentFile string, logger logr.Logger) error {
	args := []string{
		"fn", "render",

		"--input", intentFile,

		"--output", r.OutputDir,
	}

	// Add mode-specific arguments.

	if r.Mode == "dry-run" {
		args = append(args, "--dry-run")
	}

	logger.Info("Executing porch CLI", "command", r.PorchPath, "args", args)

	// Use injected executor for testing, otherwise use default.

	executor := r.porchExecutor

	if executor == nil {
		executor = &defaultPorchExecutor{}
	}

	err := executor.ExecutePorch(ctx, r.PorchPath, args, r.OutputDir, intentFile, r.Mode)
	if err != nil {

		logger.Error(err, "Porch CLI execution failed")

		return err

	}

	logger.Info("Porch CLI executed successfully")

	return nil
}
