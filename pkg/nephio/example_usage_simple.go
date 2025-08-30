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

package nephio

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
)

// SimpleExampleUsage demonstrates basic usage with correct API structure.

type SimpleExampleUsage struct {
	client client.Client

	integration *NephioIntegration
}

// NewSimpleExampleUsage creates a new simple example usage instance.

func NewSimpleExampleUsage(client client.Client, integration *NephioIntegration) *SimpleExampleUsage {

	return &SimpleExampleUsage{

		client: client,

		integration: integration,
	}

}

// BasicDeployment demonstrates basic 5G Core deployment.

func (seu *SimpleExampleUsage) BasicDeployment(ctx context.Context) error {

	logger := log.FromContext(ctx).WithName("simple-deployment")

	logger.Info("Starting basic deployment scenario")

	// Create a NetworkIntent with correct API structure.

	intent := &v1.NetworkIntent{

		ObjectMeta: metav1.ObjectMeta{

			Name: "deploy-5g-amf-production",

			Namespace: "default",

			Labels: map[string]string{

				"scenario": "basic-5g-core",

				"component": "amf",

				"environment": "production",
			},
		},

		Spec: v1.NetworkIntentSpec{

			Intent: "Deploy 5G AMF with high availability for production",

			IntentType: v1.IntentTypeDeployment,

			Priority: v1.PriorityHigh,

			TargetComponents: []v1.ORANComponent{v1.ORANComponentAMF},
		},
	}

	logger.Info("Created NetworkIntent for 5G AMF deployment", "intent", intent.Name)

	// Process the intent with Nephio workflow.

	execution, err := seu.integration.ProcessNetworkIntent(ctx, intent)

	if err != nil {

		return fmt.Errorf("failed to process NetworkIntent: %w", err)

	}

	logger.Info("Workflow execution started",

		"executionId", execution.ID,

		"workflow", execution.WorkflowDef.Name,

		"phases", len(execution.Phases),
	)

	return seu.monitorWorkflowExecution(ctx, execution.ID, 30*time.Minute)

}

// ORANRICDeployment demonstrates O-RAN RIC deployment.

func (seu *SimpleExampleUsage) ORANRICDeployment(ctx context.Context) error {

	logger := log.FromContext(ctx).WithName("oran-ric-deployment")

	logger.Info("Starting O-RAN RIC deployment scenario")

	intent := &v1.NetworkIntent{

		ObjectMeta: metav1.ObjectMeta{

			Name: "deploy-oran-ric-near-rt",

			Namespace: "oran-system",

			Labels: map[string]string{

				"scenario": "oran-ric",

				"ric-type": "near-rt",

				"compliance": "o-ran",
			},
		},

		Spec: v1.NetworkIntentSpec{

			Intent: "Deploy Near-RT RIC with O-RAN compliance and E2 interface support",

			IntentType: v1.IntentTypeDeployment,

			Priority: v1.PriorityHigh,

			TargetComponents: []v1.ORANComponent{v1.ORANComponentNearRTRIC, v1.ORANComponentE2, v1.ORANComponentA1},
		},
	}

	logger.Info("Created NetworkIntent for O-RAN RIC deployment", "intent", intent.Name)

	execution, err := seu.integration.ProcessNetworkIntent(ctx, intent)

	if err != nil {

		return fmt.Errorf("failed to process O-RAN NetworkIntent: %w", err)

	}

	logger.Info("O-RAN workflow execution started",

		"executionId", execution.ID,

		"workflow", execution.WorkflowDef.Name,
	)

	return seu.monitorWorkflowExecution(ctx, execution.ID, 45*time.Minute)

}

// NetworkSliceConfiguration demonstrates network slice configuration.

func (seu *SimpleExampleUsage) NetworkSliceConfiguration(ctx context.Context) error {

	logger := log.FromContext(ctx).WithName("network-slice-config")

	logger.Info("Starting network slice configuration scenario")

	intent := &v1.NetworkIntent{

		ObjectMeta: metav1.ObjectMeta{

			Name: "configure-urllc-slice",

			Namespace: "slicing-system",

			Labels: map[string]string{

				"scenario": "network-slice",

				"slice-type": "urllc",

				"sst": "2",
			},
		},

		Spec: v1.NetworkIntentSpec{

			Intent: "Configure URLLC network slice with ultra-low latency requirements and QoS policies",

			IntentType: v1.IntentTypeOptimization,

			Priority: v1.PriorityHigh,

			TargetComponents: []v1.ORANComponent{v1.ORANComponentAMF, v1.ORANComponentSMF, v1.ORANComponentUPF},
		},
	}

	logger.Info("Created NetworkIntent for URLLC slice", "intent", intent.Name)

	execution, err := seu.integration.ProcessNetworkIntent(ctx, intent)

	if err != nil {

		return fmt.Errorf("failed to process slice NetworkIntent: %w", err)

	}

	logger.Info("Network slice workflow execution started",

		"executionId", execution.ID,

		"sliceType", "urllc",
	)

	return seu.monitorWorkflowExecution(ctx, execution.ID, 20*time.Minute)

}

// AutoScalingConfiguration demonstrates auto-scaling configuration.

func (seu *SimpleExampleUsage) AutoScalingConfiguration(ctx context.Context) error {

	logger := log.FromContext(ctx).WithName("auto-scaling-config")

	logger.Info("Starting auto-scaling configuration scenario")

	intent := &v1.NetworkIntent{

		ObjectMeta: metav1.ObjectMeta{

			Name: "configure-auto-scaling-upf",

			Namespace: "scaling-system",

			Labels: map[string]string{

				"scenario": "auto-scaling",

				"component": "upf",

				"scaling-type": "horizontal",
			},
		},

		Spec: v1.NetworkIntentSpec{

			Intent: "Configure intelligent auto-scaling for UPF based on traffic patterns with predictive capabilities",

			IntentType: v1.IntentTypeScaling,

			Priority: v1.PriorityMedium,

			TargetComponents: []v1.ORANComponent{v1.ORANComponentUPF},
		},
	}

	logger.Info("Created NetworkIntent for auto-scaling", "intent", intent.Name)

	execution, err := seu.integration.ProcessNetworkIntent(ctx, intent)

	if err != nil {

		return fmt.Errorf("failed to process auto-scaling NetworkIntent: %w", err)

	}

	logger.Info("Auto-scaling workflow execution started",

		"executionId", execution.ID,

		"component", "upf",
	)

	return seu.monitorWorkflowExecution(ctx, execution.ID, 15*time.Minute)

}

// monitorWorkflowExecution monitors a workflow execution until completion.

func (seu *SimpleExampleUsage) monitorWorkflowExecution(ctx context.Context, executionID string, timeout time.Duration) error {

	logger := log.FromContext(ctx).WithName("workflow-monitor").WithValues("executionId", executionID)

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)

	defer cancel()

	ticker := time.NewTicker(10 * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-timeoutCtx.Done():

			return fmt.Errorf("workflow execution monitoring timed out after %v", timeout)

		case <-ticker.C:

			execution, err := seu.integration.GetWorkflowExecution(ctx, executionID)

			if err != nil {

				logger.Error(err, "Failed to get workflow execution")

				continue

			}

			logger.Info("Workflow execution status",

				"status", execution.Status,

				"completedPhases", seu.countCompletedPhases(execution.Phases),

				"totalPhases", len(execution.Phases),
			)

			// Check if execution is complete.

			switch execution.Status {

			case WorkflowExecutionStatusCompleted:

				logger.Info("Workflow execution completed successfully",

					"totalDuration", time.Since(execution.CreatedAt),

					"phases", len(execution.Phases),
				)

				return nil

			case WorkflowExecutionStatusFailed:

				logger.Error(nil, "Workflow execution failed",

					"totalDuration", time.Since(execution.CreatedAt),
				)

				return fmt.Errorf("workflow execution failed")

			case WorkflowExecutionStatusCancelled:

				logger.Info("Workflow execution was cancelled")

				return fmt.Errorf("workflow execution was cancelled")

			}

		}

	}

}

// countCompletedPhases counts completed workflow phases.

func (seu *SimpleExampleUsage) countCompletedPhases(phases []WorkflowPhaseExecution) int {

	count := 0

	for _, phase := range phases {

		if phase.Status == WorkflowExecutionStatusCompleted {

			count++

		}

	}

	return count

}

// RunAllScenarios runs all simple example scenarios.

func (seu *SimpleExampleUsage) RunAllScenarios(ctx context.Context) error {

	logger := log.FromContext(ctx).WithName("simple-scenarios")

	logger.Info("Starting all simple example scenarios")

	scenarios := []struct {
		name string

		fn func(context.Context) error
	}{

		{"Basic 5G Core Deployment", seu.BasicDeployment},

		{"O-RAN RIC Deployment", seu.ORANRICDeployment},

		{"Network Slice Configuration", seu.NetworkSliceConfiguration},

		{"Auto-Scaling Configuration", seu.AutoScalingConfiguration},
	}

	for _, scenario := range scenarios {

		logger.Info("Running scenario", "scenario", scenario.name)

		scenarioCtx, cancel := context.WithTimeout(ctx, 1*time.Hour)

		if err := scenario.fn(scenarioCtx); err != nil {

			cancel()

			logger.Error(err, "Scenario failed", "scenario", scenario.name)

			return fmt.Errorf("scenario %s failed: %w", scenario.name, err)

		}

		cancel()

		logger.Info("Scenario completed successfully", "scenario", scenario.name)

		// Brief pause between scenarios.

		time.Sleep(5 * time.Second)

	}

	logger.Info("All simple scenarios completed successfully")

	return nil

}
