//go:build examples
// +build examples

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

package main

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
)

// This file demonstrates how to use the Nephio-native workflow patterns
// in the Nephoran Intent Operator

// ExampleUsageScenarios demonstrates various usage scenarios
type ExampleUsageScenarios struct {
	client client.Client
}

// NewExampleUsageScenarios creates a new example usage scenarios instance
func NewExampleUsageScenarios(client client.Client) *ExampleUsageScenarios {
	return &ExampleUsageScenarios{
		client: client,
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
			Intent: "Deploy 5G AMF with high availability for production and auto-scaling",
		},
	}

	logger.Info("Created NetworkIntent for 5G AMF deployment", "intent", intent.Name)

	// Create the intent in Kubernetes
	if err := eus.client.Create(ctx, intent); err != nil {
		return fmt.Errorf("failed to create NetworkIntent: %w", err)
	}

	// Monitor the intent processing
	return eus.monitorIntentProcessing(ctx, intent.Name, intent.Namespace, 30*time.Minute)
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
				"scenario":   "oran-ric",
				"ric-type":   "near-rt",
				"compliance": "o-ran",
			},
		},
		Spec: v1.NetworkIntentSpec{
			Intent: "Deploy Near-RT RIC with O-RAN compliance including A1 E2 and O1 interfaces with xApp platform support",
		},
	}

	logger.Info("Created NetworkIntent for O-RAN RIC deployment", "intent", intent.Name)

	// Create the intent in Kubernetes
	if err := eus.client.Create(ctx, intent); err != nil {
		return fmt.Errorf("failed to create O-RAN NetworkIntent: %w", err)
	}

	// Monitor with extended timeout for O-RAN compliance checks
	return eus.monitorIntentProcessing(ctx, intent.Name, intent.Namespace, 45*time.Minute)
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
			Intent: "Configure URLLC network slice with 1ms latency and dedicated resources for industrial automation",
		},
	}

	logger.Info("Created NetworkIntent for URLLC slice", "intent", intent.Name)

	// Create the intent in Kubernetes
	if err := eus.client.Create(ctx, intent); err != nil {
		return fmt.Errorf("failed to create slice NetworkIntent: %w", err)
	}

	return eus.monitorIntentProcessing(ctx, intent.Name, intent.Namespace, 20*time.Minute)
}

// Scenario4_MultiClusterDeployment demonstrates multi-cluster deployment
func (eus *ExampleUsageScenarios) Scenario4_MultiClusterDeployment(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("scenario-multi-cluster")
	logger.Info("Starting multi-cluster deployment scenario")

	// Step 1: Create multi-cluster deployment intent
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
			Intent: "Deploy 5G Core across multiple clusters with UPF and AMF at edge and central functions like NRF NSSF at core with service mesh interconnect",
		},
	}

	logger.Info("Created multi-cluster NetworkIntent", "intent", intent.Name)

	// Create the intent in Kubernetes
	if err := eus.client.Create(ctx, intent); err != nil {
		return fmt.Errorf("failed to create multi-cluster NetworkIntent: %w", err)
	}

	return eus.monitorIntentProcessing(ctx, intent.Name, intent.Namespace, 60*time.Minute)
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
				"scenario":     "auto-scaling",
				"component":    "upf",
				"scaling-type": "horizontal",
			},
		},
		Spec: v1.NetworkIntentSpec{
			Intent: "Configure intelligent auto-scaling for UPF with predictive ML-based horizontal scaling from 2 to 20 replicas based on CPU memory and traffic metrics",
		},
	}

	logger.Info("Created NetworkIntent for auto-scaling", "intent", intent.Name)

	// Create the intent in Kubernetes
	if err := eus.client.Create(ctx, intent); err != nil {
		return fmt.Errorf("failed to create auto-scaling NetworkIntent: %w", err)
	}

	return eus.monitorIntentProcessing(ctx, intent.Name, intent.Namespace, 15*time.Minute)
}

// monitorIntentProcessing monitors a NetworkIntent until completion
func (eus *ExampleUsageScenarios) monitorIntentProcessing(ctx context.Context, name, namespace string, timeout time.Duration) error {
	logger := log.FromContext(ctx).WithName("intent-monitor").WithValues("name", name, "namespace", namespace)

	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-timeoutCtx.Done():
			return fmt.Errorf("intent processing monitoring timed out after %v", timeout)

		case <-ticker.C:
			intent := &v1.NetworkIntent{}
			err := eus.client.Get(ctx, client.ObjectKey{Name: name, Namespace: namespace}, intent)
			if err != nil {
				logger.Error(err, "Failed to get NetworkIntent")
				continue
			}

			logger.Info("NetworkIntent status",
				"phase", intent.Status.Phase,
				"message", intent.Status.LastMessage,
			)

			// Check if processing is complete
			switch intent.Status.Phase {
			case "Ready", "Completed":
				logger.Info("NetworkIntent processing completed successfully",
					"phase", intent.Status.Phase,
				)
				return nil

			case "Failed":
				logger.Error(nil, "NetworkIntent processing failed",
					"message", intent.Status.LastMessage,
				)
				return fmt.Errorf("intent processing failed: %s", intent.Status.LastMessage)
			}
		}
	}
}

// RunAllScenarios runs all example scenarios
func (eus *ExampleUsageScenarios) RunAllScenarios(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("example-scenarios")
	logger.Info("Starting all example scenarios")

	scenarios := []struct {
		name string
		fn   func(context.Context) error
	}{
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