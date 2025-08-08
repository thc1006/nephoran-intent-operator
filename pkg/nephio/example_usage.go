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

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// This file demonstrates how to use the Nephio-native workflow patterns
// in the Nephoran Intent Operator

// ExampleUsageScenarios demonstrates various usage scenarios
type ExampleUsageScenarios struct {
	client      client.Client
	integration *NephioIntegration
}

// NewExampleUsageScenarios creates a new example usage scenarios instance
func NewExampleUsageScenarios(client client.Client, integration *NephioIntegration) *ExampleUsageScenarios {
	return &ExampleUsageScenarios{
		client:      client,
		integration: integration,
	}
}

// Scenario1_Basic5GCoreDeployment demonstrates basic 5G Core deployment
func (eus *ExampleUsageScenarios) Scenario1_Basic5GCoreDeployment(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("scenario-5g-core-basic")
	logger.Info("Starting basic 5G Core deployment scenario")

	// Step 1: Create a NetworkIntent for 5G AMF deployment
	intent := &v1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deploy-5g-amf-production",
			Namespace: "default",
			Labels: map[string]string{
				"scenario":    "basic-5g-core",
				"component":   "amf",
				"environment": "production",
			},
		},
		Spec: v1.NetworkIntentSpec{
			IntentType:      v1.NetworkIntentTypeDeployment,
			TargetComponent: "amf",
			Description:     "Deploy 5G AMF with high availability for production",
			Configuration: map[string]interface{}{
				"replicas":        3,
				"resources.cpu":   "1000m",
				"resources.memory": "2Gi",
				"highAvailability": true,
				"autoScaling": map[string]interface{}{
					"enabled":     true,
					"minReplicas": 3,
					"maxReplicas": 10,
					"targetCPU":   70,
				},
				"security": map[string]interface{}{
					"tls":            true,
					"authentication": "mutual-tls",
				},
			},
		},
	}

	logger.Info("Created NetworkIntent for 5G AMF deployment", "intent", intent.Name)

	// Step 2: Process the intent with Nephio workflow
	execution, err := eus.integration.ProcessNetworkIntent(ctx, intent)
	if err != nil {
		return fmt.Errorf("failed to process NetworkIntent: %w", err)
	}

	logger.Info("Workflow execution started",
		"executionId", execution.ID,
		"workflow", execution.WorkflowDef.Name,
		"phases", len(execution.Phases),
	)

	// Step 3: Monitor workflow execution
	return eus.monitorWorkflowExecution(ctx, execution.ID, 30*time.Minute)
}

// Scenario2_ORANRICDeployment demonstrates O-RAN RIC deployment with compliance
func (eus *ExampleUsageScenarios) Scenario2_ORANRICDeployment(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("scenario-oran-ric")
	logger.Info("Starting O-RAN RIC deployment scenario")

	// Step 1: Create NetworkIntent with O-RAN compliance requirements
	intent := &v1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deploy-oran-ric-near-rt",
			Namespace: "oran-system",
			Labels: map[string]string{
				"scenario":    "oran-ric",
				"ric-type":    "near-rt",
				"compliance":  "o-ran",
			},
		},
		Spec: v1.NetworkIntentSpec{
			IntentType:      v1.NetworkIntentTypeDeployment,
			TargetComponent: "ric",
			Description:     "Deploy Near-RT RIC with O-RAN compliance",
			Configuration: map[string]interface{}{
				"ricType":     "near-rt",
				"ricVersion":  "1.0.0",
				"xAppPlatform": map[string]interface{}{
					"enabled": true,
					"resources": map[string]interface{}{
						"cpu":    "2000m",
						"memory": "4Gi",
					},
				},
				"e2Interface": map[string]interface{}{
					"enabled": true,
					"version": "2.0",
					"endpoints": []string{
						"e2-term-alpha.oran.svc.cluster.local:38000",
						"e2-term-beta.oran.svc.cluster.local:38000",
					},
				},
				"a1Interface": map[string]interface{}{
					"enabled": true,
					"version": "2.0",
					"policyTypes": []string{
						"traffic-steering",
						"qos-management",
						"admission-control",
					},
				},
				"oranCompliance": map[string]interface{}{
					"interfaces": []map[string]interface{}{
						{
							"name":    "A1",
							"version": "2.0",
							"enabled": true,
						},
						{
							"name":    "E2",
							"version": "2.0", 
							"enabled": true,
						},
						{
							"name":    "O1",
							"version": "1.0",
							"enabled": true,
						},
					},
					"certifications": []map[string]interface{}{
						{
							"name":      "O-RAN-SC",
							"authority": "O-RAN Alliance",
							"version":   "Cherry",
						},
					},
				},
			},
		},
	}

	logger.Info("Created NetworkIntent for O-RAN RIC deployment", "intent", intent.Name)

	// Step 2: Process with Nephio workflow
	execution, err := eus.integration.ProcessNetworkIntent(ctx, intent)
	if err != nil {
		return fmt.Errorf("failed to process O-RAN NetworkIntent: %w", err)
	}

	logger.Info("O-RAN workflow execution started",
		"executionId", execution.ID,
		"workflow", execution.WorkflowDef.Name,
	)

	// Step 3: Monitor execution with extended timeout for O-RAN compliance checks
	return eus.monitorWorkflowExecution(ctx, execution.ID, 45*time.Minute)
}

// Scenario3_NetworkSliceConfiguration demonstrates network slice configuration
func (eus *ExampleUsageScenarios) Scenario3_NetworkSliceConfiguration(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("scenario-network-slice")
	logger.Info("Starting network slice configuration scenario")

	// Step 1: Create NetworkIntent for URLLC slice
	intent := &v1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configure-urllc-slice",
			Namespace: "slicing-system",
			Labels: map[string]string{
				"scenario":   "network-slice",
				"slice-type": "urllc",
				"sst":        "2",
			},
		},
		Spec: v1.NetworkIntentSpec{
			IntentType:      v1.NetworkIntentTypeConfiguration,
			TargetComponent: "nssf",
			Description:     "Configure URLLC network slice with ultra-low latency requirements",
			Configuration: map[string]interface{}{
				"networkSlice": map[string]interface{}{
					"sliceId":   "urllc-slice-001",
					"sliceType": "URLLC",
					"sst":       2,
					"sd":        "000001",
					"sla": map[string]interface{}{
						"latency":      "1ms",
						"reliability":  "99.999%",
						"throughput":   "100Mbps",
						"availability": "99.99%",
					},
					"qos": map[string]interface{}{
						"priority": 1,
						"qci":      1,
						"arp": map[string]interface{}{
							"priorityLevel":        1,
							"preemptionCapability": true,
							"preemptionVulnerability": false,
						},
						"gbr": "100Mbps",
						"mbr": "1Gbps",
					},
					"resources": map[string]interface{}{
						"cpu":     "4000m",
						"memory":  "8Gi",
						"storage": "100Gi",
						"network": map[string]interface{}{
							"bandwidth": "1Gbps",
							"isolation": "dedicated",
						},
					},
					"coverage": map[string]interface{}{
						"areas": []string{
							"industrial-zone-1",
							"factory-floor-a",
						},
						"cells": []string{
							"cell-001",
							"cell-002",
							"cell-003",
						},
					},
				},
				"networkFunctions": []map[string]interface{}{
					{
						"type": "amf",
						"config": map[string]interface{}{
							"sliceSupport": true,
							"priority":     "high",
						},
					},
					{
						"type": "smf",
						"config": map[string]interface{}{
							"sessionManagement": map[string]interface{}{
								"urllcOptimized": true,
								"fastHandover":   true,
							},
						},
					},
					{
						"type": "upf",
						"config": map[string]interface{}{
							"dataPlane": map[string]interface{}{
								"type":           "dpdk",
								"lowLatencyMode": true,
								"bufferSize":     "minimal",
							},
						},
					},
				},
			},
		},
	}

	logger.Info("Created NetworkIntent for URLLC slice", "intent", intent.Name)

	// Step 2: Process with Nephio workflow
	execution, err := eus.integration.ProcessNetworkIntent(ctx, intent)
	if err != nil {
		return fmt.Errorf("failed to process slice NetworkIntent: %w", err)
	}

	logger.Info("Network slice workflow execution started",
		"executionId", execution.ID,
		"sliceId", "urllc-slice-001",
	)

	return eus.monitorWorkflowExecution(ctx, execution.ID, 20*time.Minute)
}

// Scenario4_MultiClusterDeployment demonstrates multi-cluster deployment
func (eus *ExampleUsageScenarios) Scenario4_MultiClusterDeployment(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("scenario-multi-cluster")
	logger.Info("Starting multi-cluster deployment scenario")

	// Step 1: Register multiple workload clusters
	clusters := []*WorkloadCluster{
		{
			Name:     "edge-cluster-east",
			Endpoint: "https://edge-east.example.com:6443",
			Region:   "us-east-1",
			Zone:     "us-east-1a",
			Capabilities: []ClusterCapability{
				{
					Name:    "5g-core",
					Type:    "network-function",
					Version: "1.0",
					Status:  "ready",
				},
				{
					Name:    "edge-computing",
					Type:    "compute",
					Version: "1.0",
					Status:  "ready",
				},
			},
			Labels: map[string]string{
				"deployment-target": "edge",
				"latency-zone":      "low",
				"capacity":          "medium",
			},
		},
		{
			Name:     "edge-cluster-west",
			Endpoint: "https://edge-west.example.com:6443",
			Region:   "us-west-2",
			Zone:     "us-west-2b",
			Capabilities: []ClusterCapability{
				{
					Name:    "5g-core",
					Type:    "network-function",
					Version: "1.0",
					Status:  "ready",
				},
				{
					Name:    "edge-computing",
					Type:    "compute",
					Version: "1.0",
					Status:  "ready",
				},
			},
			Labels: map[string]string{
				"deployment-target": "edge",
				"latency-zone":      "low",
				"capacity":          "high",
			},
		},
		{
			Name:     "central-cluster",
			Endpoint: "https://central.example.com:6443",
			Region:   "us-central-1",
			Zone:     "us-central-1c",
			Capabilities: []ClusterCapability{
				{
					Name:    "5g-core",
					Type:    "network-function",
					Version: "1.0",
					Status:  "ready",
				},
				{
					Name:    "oran-ric",
					Type:    "network-function",
					Version: "1.0",
					Status:  "ready",
				},
				{
					Name:    "analytics",
					Type:    "data-processing",
					Version: "1.0",
					Status:  "ready",
				},
			},
			Labels: map[string]string{
				"deployment-target": "central",
				"latency-zone":      "medium",
				"capacity":          "very-high",
			},
		},
	}

	// Register clusters
	for _, cluster := range clusters {
		if err := eus.integration.RegisterWorkloadCluster(ctx, cluster); err != nil {
			logger.Error(err, "Failed to register cluster", "cluster", cluster.Name)
			continue
		}
		logger.Info("Registered workload cluster", "cluster", cluster.Name, "region", cluster.Region)
	}

	// Step 2: Create multi-cluster deployment intent
	intent := &v1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "deploy-5g-core-multi-cluster",
			Namespace: "multi-cluster-system",
			Labels: map[string]string{
				"scenario":        "multi-cluster",
				"deployment-type": "distributed",
			},
		},
		Spec: v1.NetworkIntentSpec{
			IntentType:      v1.NetworkIntentTypeDeployment,
			TargetComponent: "5g-core",
			Description:     "Deploy 5G Core network functions across multiple clusters",
			Configuration: map[string]interface{}{
				"deploymentStrategy": "multi-cluster",
				"clusterTargets": []map[string]interface{}{
					{
						"cluster": "edge-cluster-east",
						"components": []string{"upf", "amf"},
						"resources": map[string]interface{}{
							"upf": map[string]interface{}{
								"replicas": 2,
								"resources": map[string]interface{}{
									"cpu":    "2000m",
									"memory": "4Gi",
								},
							},
							"amf": map[string]interface{}{
								"replicas": 1,
								"resources": map[string]interface{}{
									"cpu":    "1000m",
									"memory": "2Gi",
								},
							},
						},
					},
					{
						"cluster": "edge-cluster-west",
						"components": []string{"upf", "amf"},
						"resources": map[string]interface{}{
							"upf": map[string]interface{}{
								"replicas": 3,
								"resources": map[string]interface{}{
									"cpu":    "2000m",
									"memory": "4Gi",
								},
							},
							"amf": map[string]interface{}{
								"replicas": 2,
								"resources": map[string]interface{}{
									"cpu":    "1000m",
									"memory": "2Gi",
								},
							},
						},
					},
					{
						"cluster": "central-cluster",
						"components": []string{"nrf", "nssf", "udm", "ausf"},
						"resources": map[string]interface{}{
							"nrf": map[string]interface{}{
								"replicas": 2,
								"resources": map[string]interface{}{
									"cpu":    "500m",
									"memory": "1Gi",
								},
							},
							"nssf": map[string]interface{}{
								"replicas": 1,
								"resources": map[string]interface{}{
									"cpu":    "500m",
									"memory": "1Gi",
								},
							},
						},
					},
				},
				"networkConfiguration": map[string]interface{}{
					"interconnect": map[string]interface{}{
						"type": "service-mesh",
						"security": map[string]interface{}{
							"mTLS": true,
							"encryption": "AES-256",
						},
					},
					"loadBalancing": map[string]interface{}{
						"enabled": true,
						"algorithm": "round-robin",
						"healthCheck": true,
					},
				},
				"monitoring": map[string]interface{}{
					"enabled": true,
					"centralized": true,
					"metricsCollection": map[string]interface{}{
						"prometheus": true,
						"grafana": true,
					},
				},
			},
		},
	}

	logger.Info("Created multi-cluster NetworkIntent", "intent", intent.Name)

	// Step 3: Process with Nephio workflow
	execution, err := eus.integration.ProcessNetworkIntent(ctx, intent)
	if err != nil {
		return fmt.Errorf("failed to process multi-cluster NetworkIntent: %w", err)
	}

	logger.Info("Multi-cluster workflow execution started",
		"executionId", execution.ID,
		"targetClusters", len(clusters),
	)

	return eus.monitorWorkflowExecution(ctx, execution.ID, 60*time.Minute)
}

// Scenario5_AutoScalingConfiguration demonstrates auto-scaling configuration
func (eus *ExampleUsageScenarios) Scenario5_AutoScalingConfiguration(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("scenario-auto-scaling")
	logger.Info("Starting auto-scaling configuration scenario")

	// Create NetworkIntent for auto-scaling
	intent := &v1.NetworkIntent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "configure-auto-scaling-upf",
			Namespace: "scaling-system",
			Labels: map[string]string{
				"scenario":   "auto-scaling",
				"component":  "upf",
				"scaling-type": "horizontal",
			},
		},
		Spec: v1.NetworkIntentSpec{
			IntentType:      v1.NetworkIntentTypeScaling,
			TargetComponent: "upf",
			Description:     "Configure intelligent auto-scaling for UPF based on traffic patterns",
			Configuration: map[string]interface{}{
				"scalingPolicy": map[string]interface{}{
					"type": "horizontal",
					"target": map[string]interface{}{
						"kind": "Deployment",
						"name": "upf-deployment",
					},
					"minReplicas": 2,
					"maxReplicas": 20,
					"metrics": []map[string]interface{}{
						{
							"type": "Resource",
							"resource": map[string]interface{}{
								"name": "cpu",
								"target": map[string]interface{}{
									"type":               "Utilization",
									"averageUtilization": 70,
								},
							},
						},
						{
							"type": "Resource",
							"resource": map[string]interface{}{
								"name": "memory",
								"target": map[string]interface{}{
									"type":               "Utilization",
									"averageUtilization": 80,
								},
							},
						},
						{
							"type": "External",
							"external": map[string]interface{}{
								"metric": map[string]interface{}{
									"name": "upf_session_count",
									"selector": map[string]interface{}{
										"matchLabels": map[string]interface{}{
											"component": "upf",
										},
									},
								},
								"target": map[string]interface{}{
									"type":         "AverageValue",
									"averageValue": "1000",
								},
							},
						},
						{
							"type": "External",
							"external": map[string]interface{}{
								"metric": map[string]interface{}{
									"name": "upf_throughput_mbps",
									"selector": map[string]interface{}{
										"matchLabels": map[string]interface{}{
											"component": "upf",
										},
									},
								},
								"target": map[string]interface{}{
									"type":         "AverageValue",
									"averageValue": "800",
								},
							},
						},
					},
					"behavior": map[string]interface{}{
						"scaleDown": map[string]interface{}{
							"stabilizationWindowSeconds": 300,
							"policies": []map[string]interface{}{
								{
									"type":          "Percent",
									"value":         10,
									"periodSeconds": 60,
								},
								{
									"type":          "Pods",
									"value":         2,
									"periodSeconds": 60,
								},
							},
						},
						"scaleUp": map[string]interface{}{
							"stabilizationWindowSeconds": 60,
							"policies": []map[string]interface{}{
								{
									"type":          "Percent",
									"value":         50,
									"periodSeconds": 30,
								},
								{
									"type":          "Pods",
									"value":         4,
									"periodSeconds": 30,
								},
							},
						},
					},
				},
				"predictiveScaling": map[string]interface{}{
					"enabled": true,
					"algorithm": "machine-learning",
					"trainingData": map[string]interface{}{
						"historicalPeriod": "30d",
						"samplingInterval": "5m",
						"features": []string{
							"cpu_utilization",
							"memory_utilization",
							"session_count",
							"throughput",
							"time_of_day",
							"day_of_week",
						},
					},
					"predictionHorizon": "30m",
					"scaleUpThreshold": 0.8,
				},
				"monitoring": map[string]interface{}{
					"enabled": true,
					"dashboard": true,
					"alerts": []map[string]interface{}{
						{
							"name": "HighScalingActivity",
							"condition": "scaling_events_per_hour > 10",
							"severity": "warning",
						},
						{
							"name": "ScalingAtMaxCapacity",
							"condition": "current_replicas >= max_replicas",
							"severity": "critical",
						},
					},
				},
			},
		},
	}

	logger.Info("Created NetworkIntent for auto-scaling", "intent", intent.Name)

	// Process with Nephio workflow
	execution, err := eus.integration.ProcessNetworkIntent(ctx, intent)
	if err != nil {
		return fmt.Errorf("failed to process auto-scaling NetworkIntent: %w", err)
	}

	logger.Info("Auto-scaling workflow execution started",
		"executionId", execution.ID,
		"component", "upf",
	)

	return eus.monitorWorkflowExecution(ctx, execution.ID, 15*time.Minute)
}

// monitorWorkflowExecution monitors a workflow execution until completion
func (eus *ExampleUsageScenarios) monitorWorkflowExecution(ctx context.Context, executionID string, timeout time.Duration) error {
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
			execution, err := eus.integration.GetWorkflowExecution(ctx, executionID)
			if err != nil {
				logger.Error(err, "Failed to get workflow execution")
				continue
			}

			logger.Info("Workflow execution status",
				"status", execution.Status,
				"completedPhases", eus.countCompletedPhases(execution.Phases),
				"totalPhases", len(execution.Phases),
			)

			// Log phase details
			for _, phase := range execution.Phases {
				if phase.Status != WorkflowExecutionStatusPending {
					phaseLogger := logger.WithValues("phase", phase.Name)
					phaseLogger.Info("Phase status",
						"status", phase.Status,
						"duration", phase.Duration,
						"errors", len(phase.Errors),
					)
				}
			}

			// Check if execution is complete
			switch execution.Status {
			case WorkflowExecutionStatusCompleted:
				logger.Info("Workflow execution completed successfully",
					"totalDuration", time.Since(execution.CreatedAt),
					"phases", len(execution.Phases),
					"packageVariants", len(execution.PackageVariants),
					"deployments", len(execution.Deployments),
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

// countCompletedPhases counts completed workflow phases
func (eus *ExampleUsageScenarios) countCompletedPhases(phases []WorkflowPhaseExecution) int {
	count := 0
	for _, phase := range phases {
		if phase.Status == WorkflowExecutionStatusCompleted {
			count++
		}
	}
	return count
}

// DemoCustomWorkflow demonstrates creating and registering a custom workflow
func (eus *ExampleUsageScenarios) DemoCustomWorkflow(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("demo-custom-workflow")
	logger.Info("Demonstrating custom workflow creation")

	// Create a custom workflow for edge deployment
	customWorkflow := &WorkflowDefinition{
		Name:        "edge-optimized-deployment",
		Description: "Edge-optimized deployment workflow for latency-sensitive applications",
		IntentTypes: []v1.NetworkIntentType{
			v1.NetworkIntentTypeDeployment,
		},
		Phases: []WorkflowPhase{
			{
				Name:        "edge-requirements-validation",
				Description: "Validate edge deployment requirements",
				Type:        WorkflowPhaseTypeValidation,
				Dependencies: []string{},
				Actions: []WorkflowAction{
					{
						Name:     "validate-edge-requirements",
						Type:     WorkflowActionTypeValidatePackage,
						Required: true,
						Timeout:  3 * time.Minute,
						Config: map[string]interface{}{
							"checks": []string{
								"latency-requirements",
								"edge-capabilities",
								"resource-constraints",
							},
						},
					},
				},
				Timeout:         5 * time.Minute,
				ContinueOnError: false,
			},
			{
				Name:        "edge-blueprint-selection",
				Description: "Select edge-optimized blueprint",
				Type:        WorkflowPhaseTypeBlueprintSelection,
				Dependencies: []string{"edge-requirements-validation"},
				Actions: []WorkflowAction{
					{
						Name:     "select-edge-blueprint",
						Type:     WorkflowActionTypeCreatePackageRevision,
						Required: true,
						Timeout:  5 * time.Minute,
						Config: map[string]interface{}{
							"blueprintFilter": map[string]interface{}{
								"category": "edge",
								"latencyOptimized": true,
							},
						},
					},
				},
				Timeout:         8 * time.Minute,
				ContinueOnError: false,
			},
			{
				Name:        "edge-package-optimization",
				Description: "Optimize package for edge deployment",
				Type:        WorkflowPhaseTypePackageSpecialization,
				Dependencies: []string{"edge-blueprint-selection"},
				Actions: []WorkflowAction{
					{
						Name:     "optimize-for-edge",
						Type:     WorkflowActionTypeSpecializePackage,
						Required: true,
						Timeout:  10 * time.Minute,
						Config: map[string]interface{}{
							"optimizations": []string{
								"reduce-memory-footprint",
								"enable-fast-startup",
								"optimize-network-stack",
							},
						},
					},
				},
				Timeout:         15 * time.Minute,
				ContinueOnError: false,
			},
			{
				Name:        "edge-deployment",
				Description: "Deploy to edge clusters with proximity optimization",
				Type:        WorkflowPhaseTypeDeployment,
				Dependencies: []string{"edge-package-optimization"},
				Actions: []WorkflowAction{
					{
						Name:     "deploy-to-edge-clusters",
						Type:     WorkflowActionTypeDeployToCluster,
						Required: true,
						Timeout:  15 * time.Minute,
						Config: map[string]interface{}{
							"deploymentStrategy": "proximity-based",
							"preferredZones": []string{
								"edge",
								"far-edge",
							},
						},
					},
				},
				Timeout:         20 * time.Minute,
				ContinueOnError: false,
			},
			{
				Name:        "edge-performance-validation",
				Description: "Validate edge deployment performance",
				Type:        WorkflowPhaseTypeMonitoring,
				Dependencies: []string{"edge-deployment"},
				Actions: []WorkflowAction{
					{
						Name:     "validate-edge-performance",
						Type:     WorkflowActionTypeVerifyHealth,
						Required: true,
						Timeout:  10 * time.Minute,
						Config: map[string]interface{}{
							"performanceTests": []string{
								"latency-test",
								"throughput-test",
								"resource-utilization-test",
							},
							"thresholds": map[string]interface{}{
								"maxLatency": "10ms",
								"minThroughput": "1Gbps",
								"maxCpuUsage": "70%",
							},
						},
					},
				},
				Timeout:         15 * time.Minute,
				ContinueOnError: true,
			},
		},
		Rollback: &RollbackStrategy{
			Enabled: true,
			TriggerOn: []string{
				"edge-deployment-failure",
				"performance-validation-failure",
			},
			Timeout: 10 * time.Minute,
		},
		Timeouts: &TimeoutStrategy{
			GlobalTimeout: 90 * time.Minute,
			OnTimeout:     "rollback",
		},
		Metadata: map[string]string{
			"category":     "edge",
			"optimization": "latency",
			"version":      "1.0.0",
		},
	}

	// Register the custom workflow
	if err := eus.integration.RegisterCustomWorkflow(customWorkflow); err != nil {
		return fmt.Errorf("failed to register custom workflow: %w", err)
	}

	logger.Info("Custom workflow registered successfully",
		"workflow", customWorkflow.Name,
		"phases", len(customWorkflow.Phases),
	)

	// Validate the custom workflow
	validationResult, err := eus.integration.ValidateWorkflow(ctx, customWorkflow)
	if err != nil {
		return fmt.Errorf("failed to validate custom workflow: %w", err)
	}

	logger.Info("Custom workflow validation completed",
		"valid", validationResult.Valid,
		"errors", len(validationResult.Errors),
		"warnings", len(validationResult.Warnings),
		"complexity", validationResult.Metrics.Complexity,
	)

	if !validationResult.Valid {
		for _, err := range validationResult.Errors {
			logger.Error(nil, "Validation error", "code", err.Code, "message", err.Message)
		}
		return fmt.Errorf("custom workflow validation failed")
	}

	return nil
}

// RunAllScenarios runs all example scenarios
func (eus *ExampleUsageScenarios) RunAllScenarios(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("example-scenarios")
	logger.Info("Starting all example scenarios")

	scenarios := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"Custom Workflow Demo", eus.DemoCustomWorkflow},
		{"Basic 5G Core Deployment", eus.Scenario1_Basic5GCoreDeployment},
		{"O-RAN RIC Deployment", eus.Scenario2_ORANRICDeployment},
		{"Network Slice Configuration", eus.Scenario3_NetworkSliceConfiguration},
		{"Multi-Cluster Deployment", eus.Scenario4_MultiClusterDeployment},
		{"Auto-Scaling Configuration", eus.Scenario5_AutoScalingConfiguration},
	}

	for _, scenario := range scenarios {
		logger.Info("Running scenario", "scenario", scenario.name)
		
		scenarioCtx, cancel := context.WithTimeout(ctx, 2*time.Hour)
		
		if err := scenario.fn(scenarioCtx); err != nil {
			cancel()
			logger.Error(err, "Scenario failed", "scenario", scenario.name)
			return fmt.Errorf("scenario %s failed: %w", scenario.name, err)
		}
		
		cancel()
		logger.Info("Scenario completed successfully", "scenario", scenario.name)
		
		// Brief pause between scenarios
		time.Sleep(5 * time.Second)
	}

	logger.Info("All scenarios completed successfully")
	return nil
}

// GetIntegrationStatus demonstrates how to get integration status
func (eus *ExampleUsageScenarios) GetIntegrationStatus(ctx context.Context) (*NephioIntegrationStatus, error) {
	return eus.integration.GetStatus(ctx)
}

// GetIntegrationMetrics demonstrates how to get integration metrics
func (eus *ExampleUsageScenarios) GetIntegrationMetrics(ctx context.Context) (map[string]interface{}, error) {
	return eus.integration.GetIntegrationMetrics(ctx)
}

// PerformHealthCheck demonstrates how to perform health checks
func (eus *ExampleUsageScenarios) PerformHealthCheck(ctx context.Context) error {
	return eus.integration.HealthCheck(ctx)
}