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

package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/cnf"
)

const (
	// Labels for CNF integration
	CNFIntentLabel           = "nephoran.com/cnf-intent"
	CNFDeploymentSourceLabel = "nephoran.com/source"
	CNFDeploymentSourceValue = "network-intent"
	CNFDeploymentIntentLabel = "nephoran.com/network-intent"

	// Annotations for CNF tracking
	CNFProcessingResultAnnotation = "nephoran.com/cnf-processing-result"
	CNFLastProcessedAnnotation    = "nephoran.com/cnf-last-processed"

	// Events for CNF processing
	EventCNFIntentDetected      = "CNFIntentDetected"
	EventCNFProcessingStarted   = "CNFProcessingStarted"
	EventCNFProcessingCompleted = "CNFProcessingCompleted"
	EventCNFProcessingFailed    = "CNFProcessingFailed"
	EventCNFDeploymentCreated   = "CNFDeploymentCreated"
	EventCNFDeploymentFailed    = "CNFDeploymentFailed"
)

// CNFIntegrationManager manages CNF deployment integration with NetworkIntent
type CNFIntegrationManager struct {
	Client             client.Client
	Scheme             *runtime.Scheme
	Recorder           record.EventRecorder
	CNFIntentProcessor *cnf.CNFIntentProcessor
	CNFOrchestrator    *cnf.CNFOrchestrator
	Config             *CNFIntegrationConfig
}

// CNFIntegrationConfig holds configuration for CNF integration
type CNFIntegrationConfig struct {
	EnableCNFIntegration    bool
	CNFProcessingTimeout    time.Duration
	CNFDeploymentTimeout    time.Duration
	MaxConcurrentCNFDeploys int
	EnableAutoDeployment    bool
	RequireExplicitApproval bool
	DefaultTargetNamespace  string
	CNFLabelSelector        map[string]string
	RetryPolicy             CNFRetryPolicy
}

// CNFRetryPolicy defines retry behavior for CNF operations
type CNFRetryPolicy struct {
	MaxRetries        int
	InitialDelay      time.Duration
	MaxDelay          time.Duration
	BackoffMultiplier float64
}

// CNFDeploymentContext holds context for CNF deployment operations
type CNFDeploymentContext struct {
	NetworkIntent    *nephoranv1.NetworkIntent
	ProcessingResult *nephoranv1.CNFIntentProcessingResult
	CNFDeployments   []*nephoranv1.CNFDeployment
	StartTime        time.Time
	RequestID        string
	ProcessingPhase  string
	Errors           []error
	Warnings         []string
}

// NewCNFIntegrationManager creates a new CNF integration manager
func NewCNFIntegrationManager(
	client client.Client,
	scheme *runtime.Scheme,
	recorder record.EventRecorder,
	cnfProcessor *cnf.CNFIntentProcessor,
	cnfOrchestrator *cnf.CNFOrchestrator,
) *CNFIntegrationManager {

	return &CNFIntegrationManager{
		Client:             client,
		Scheme:             scheme,
		Recorder:           recorder,
		CNFIntentProcessor: cnfProcessor,
		CNFOrchestrator:    cnfOrchestrator,
		Config: &CNFIntegrationConfig{
			EnableCNFIntegration:    true,
			CNFProcessingTimeout:    5 * time.Minute,
			CNFDeploymentTimeout:    15 * time.Minute,
			MaxConcurrentCNFDeploys: 5,
			EnableAutoDeployment:    true,
			RequireExplicitApproval: false,
			DefaultTargetNamespace:  "cnf-deployments",
			RetryPolicy: CNFRetryPolicy{
				MaxRetries:        3,
				InitialDelay:      30 * time.Second,
				MaxDelay:          5 * time.Minute,
				BackoffMultiplier: 2.0,
			},
		},
	}
}

// ProcessCNFIntent processes a NetworkIntent for CNF deployment requirements
func (m *CNFIntegrationManager) ProcessCNFIntent(ctx context.Context, networkIntent *nephoranv1.NetworkIntent) error {
	logger := log.FromContext(ctx)
	logger.Info("Processing NetworkIntent for CNF deployment", "intent", networkIntent.Name)

	if !m.Config.EnableCNFIntegration {
		logger.Info("CNF integration is disabled, skipping CNF processing")
		return nil
	}

	// Check if this intent contains CNF-related requirements
	if !m.isCNFIntent(networkIntent) {
		logger.Info("NetworkIntent does not contain CNF deployment requirements")
		return nil
	}

	// Create processing context
	deploymentContext := &CNFDeploymentContext{
		NetworkIntent:   networkIntent,
		StartTime:       time.Now(),
		RequestID:       string(networkIntent.UID),
		ProcessingPhase: "Detection",
	}

	// Record CNF intent detection
	m.Recorder.Event(networkIntent, "Normal", EventCNFIntentDetected,
		"CNF deployment intent detected in NetworkIntent")

	// Check for existing processing results
	if m.hasExistingProcessingResult(networkIntent) && !m.shouldReprocess(networkIntent) {
		logger.Info("CNF intent already processed, using existing results")
		return m.deployExistingCNFs(ctx, deploymentContext)
	}

	// Process the intent for CNF deployment specifications
	deploymentContext.ProcessingPhase = "Processing"
	m.Recorder.Event(networkIntent, "Normal", EventCNFProcessingStarted,
		"Starting CNF intent processing")

	processingResult, err := m.CNFIntentProcessor.ProcessCNFIntent(ctx, networkIntent)
	if err != nil {
		logger.Error(err, "CNF intent processing failed")
		m.Recorder.Event(networkIntent, "Warning", EventCNFProcessingFailed,
			fmt.Sprintf("CNF intent processing failed: %v", err))
		return m.handleProcessingError(ctx, networkIntent, deploymentContext, err)
	}

	deploymentContext.ProcessingResult = processingResult

	// Validate processing results
	if err := m.validateProcessingResults(processingResult); err != nil {
		logger.Error(err, "CNF processing results validation failed")
		deploymentContext.Errors = append(deploymentContext.Errors, err)
		return err
	}

	// Store processing results in NetworkIntent annotations
	if err := m.storeProcessingResults(ctx, networkIntent, processingResult); err != nil {
		logger.Error(err, "Failed to store CNF processing results")
		deploymentContext.Warnings = append(deploymentContext.Warnings, err.Error())
	}

	m.Recorder.Event(networkIntent, "Normal", EventCNFProcessingCompleted,
		fmt.Sprintf("CNF intent processing completed. Detected %d CNF functions", len(processingResult.DetectedFunctions)))

	// Create CNF deployments
	deploymentContext.ProcessingPhase = "Deployment"
	if err := m.createCNFDeployments(ctx, deploymentContext); err != nil {
		logger.Error(err, "Failed to create CNF deployments")
		return err
	}

	// Update NetworkIntent status with CNF deployment information
	if err := m.updateNetworkIntentWithCNFStatus(ctx, networkIntent, deploymentContext); err != nil {
		logger.Error(err, "Failed to update NetworkIntent status")
	}

	logger.Info("CNF intent processing completed successfully",
		"intent", networkIntent.Name,
		"cnfDeployments", len(deploymentContext.CNFDeployments),
		"duration", time.Since(deploymentContext.StartTime))

	return nil
}

// isCNFIntent determines if a NetworkIntent contains CNF deployment requirements
func (m *CNFIntegrationManager) isCNFIntent(networkIntent *nephoranv1.NetworkIntent) bool {
	intent := strings.ToLower(networkIntent.Spec.Intent)

	cnfKeywords := []string{
		"cnf", "cloud native", "containerized",
		"deploy", "deployment", "helm", "kubernetes",
		"microservice", "scale", "scaling",
		"high availability", "ha", "auto-scale",
		"service mesh", "istio", "monitoring",
		"amf", "smf", "upf", "nrf", "ausf", "udm",
		"o-du", "o-cu", "ric", "near-rt", "non-rt",
	}

	for _, keyword := range cnfKeywords {
		if strings.Contains(intent, keyword) {
			return true
		}
	}

	// Check for CNF-related target components
	for _, component := range networkIntent.Spec.TargetComponents {
		cnfComponents := []nephoranv1.NetworkTargetComponent{
			nephoranv1.NetworkTargetComponentAMF, nephoranv1.NetworkTargetComponentSMF, nephoranv1.NetworkTargetComponentUPF,
			nephoranv1.NetworkTargetComponentNRF, nephoranv1.NetworkTargetComponentAUSF, nephoranv1.NetworkTargetComponentUDM,
			nephoranv1.NetworkTargetComponentDU, nephoranv1.NetworkTargetComponentCUCP, nephoranv1.NetworkTargetComponentCUUP,
		}

		for _, cnfComponent := range cnfComponents {
			if component == cnfComponent {
				return true
			}
		}
	}

	// Check processed parameters for CNF-related content
	if networkIntent.Spec.ProcessedParameters != nil && networkIntent.Spec.ProcessedParameters.Raw != nil {
		// Try to parse the raw extension as a map to look for network function info
		var params map[string]interface{}
		if err := json.Unmarshal(networkIntent.Spec.ProcessedParameters.Raw, &params); err == nil {
			if networkFunction, exists := params["networkFunction"]; exists {
				if networkFunctionStr, ok := networkFunction.(string); ok && networkFunctionStr != "" {
					cnfFunctions := []string{"amf", "smf", "upf", "nrf", "ausf", "udm", "o-du", "o-cu", "ric"}
					networkFunctionLower := strings.ToLower(networkFunctionStr)
					for _, cnfFunc := range cnfFunctions {
						if strings.Contains(networkFunctionLower, cnfFunc) {
							return true
						}
					}
				}
			}
		}
	}

	return false
}

// hasExistingProcessingResult checks if there are existing CNF processing results
func (m *CNFIntegrationManager) hasExistingProcessingResult(networkIntent *nephoranv1.NetworkIntent) bool {
	if networkIntent.Annotations == nil {
		return false
	}

	_, exists := networkIntent.Annotations[CNFProcessingResultAnnotation]
	return exists
}

// shouldReprocess determines if CNF processing should be redone
func (m *CNFIntegrationManager) shouldReprocess(networkIntent *nephoranv1.NetworkIntent) bool {
	if networkIntent.Annotations == nil {
		return true
	}

	lastProcessedStr, exists := networkIntent.Annotations[CNFLastProcessedAnnotation]
	if !exists {
		return true
	}

	lastProcessed, err := time.Parse(time.RFC3339, lastProcessedStr)
	if err != nil {
		return true
	}

	// Reprocess if the intent was modified after last processing
	return networkIntent.GetGeneration() != networkIntent.Status.ObservedGeneration ||
		networkIntent.GetCreationTimestamp().After(lastProcessed)
}

// deployExistingCNFs deploys CNFs based on existing processing results
func (m *CNFIntegrationManager) deployExistingCNFs(ctx context.Context, deploymentContext *CNFDeploymentContext) error {
	logger := log.FromContext(ctx)

	// Retrieve existing processing results
	resultStr, exists := deploymentContext.NetworkIntent.Annotations[CNFProcessingResultAnnotation]
	if !exists {
		return fmt.Errorf("no existing CNF processing results found")
	}

	var processingResult nephoranv1.CNFIntentProcessingResult
	if err := json.Unmarshal([]byte(resultStr), &processingResult); err != nil {
		logger.Error(err, "Failed to unmarshal existing CNF processing results")
		return err
	}

	deploymentContext.ProcessingResult = &processingResult
	return m.createCNFDeployments(ctx, deploymentContext)
}

// validateProcessingResults validates the CNF processing results
func (m *CNFIntegrationManager) validateProcessingResults(result *nephoranv1.CNFIntentProcessingResult) error {
	if result == nil {
		return fmt.Errorf("processing result is nil")
	}

	if len(result.DetectedFunctions) == 0 {
		return fmt.Errorf("no CNF functions detected")
	}

	if len(result.CNFDeployments) == 0 {
		return fmt.Errorf("no CNF deployment specifications generated")
	}

	if result.ConfidenceScore < 0.5 {
		return fmt.Errorf("confidence score too low: %f", result.ConfidenceScore)
	}

	if len(result.Errors) > 0 {
		return fmt.Errorf("processing errors detected: %v", result.Errors)
	}

	return nil
}

// storeProcessingResults stores CNF processing results in NetworkIntent annotations
func (m *CNFIntegrationManager) storeProcessingResults(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, result *nephoranv1.CNFIntentProcessingResult) error {
	resultBytes, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal processing results: %w", err)
	}

	if networkIntent.Annotations == nil {
		networkIntent.Annotations = make(map[string]string)
	}

	networkIntent.Annotations[CNFProcessingResultAnnotation] = string(resultBytes)
	networkIntent.Annotations[CNFLastProcessedAnnotation] = time.Now().Format(time.RFC3339)

	return m.Client.Update(ctx, networkIntent)
}

// createCNFDeployments creates CNFDeployment resources based on processing results
func (m *CNFIntegrationManager) createCNFDeployments(ctx context.Context, deploymentContext *CNFDeploymentContext) error {
	logger := log.FromContext(ctx)
	logger.Info("Creating CNF deployments", "count", len(deploymentContext.ProcessingResult.CNFDeployments))

	for i, cnfSpec := range deploymentContext.ProcessingResult.CNFDeployments {
		cnfDeployment, err := m.createCNFDeploymentResource(deploymentContext.NetworkIntent, &cnfSpec, i)
		if err != nil {
			deploymentContext.Errors = append(deploymentContext.Errors, err)
			continue
		}

		// Create the CNFDeployment resource
		if err := m.Client.Create(ctx, cnfDeployment); err != nil {
			if !errors.IsAlreadyExists(err) {
				logger.Error(err, "Failed to create CNF deployment", "cnf", cnfDeployment.Name)
				deploymentContext.Errors = append(deploymentContext.Errors, err)
				m.Recorder.Event(deploymentContext.NetworkIntent, "Warning", EventCNFDeploymentFailed,
					fmt.Sprintf("Failed to create CNF deployment %s: %v", cnfDeployment.Name, err))
				continue
			}
			// If already exists, try to update
			existingCNF := &nephoranv1.CNFDeployment{}
			if err := m.Client.Get(ctx, types.NamespacedName{Name: cnfDeployment.Name, Namespace: cnfDeployment.Namespace}, existingCNF); err != nil {
				logger.Error(err, "Failed to get existing CNF deployment")
				continue
			}

			// Update the existing deployment
			existingCNF.Spec = cnfDeployment.Spec
			if err := m.Client.Update(ctx, existingCNF); err != nil {
				logger.Error(err, "Failed to update existing CNF deployment")
				deploymentContext.Errors = append(deploymentContext.Errors, err)
				continue
			}
			cnfDeployment = existingCNF
		}

		deploymentContext.CNFDeployments = append(deploymentContext.CNFDeployments, cnfDeployment)

		m.Recorder.Event(deploymentContext.NetworkIntent, "Normal", EventCNFDeploymentCreated,
			fmt.Sprintf("Created CNF deployment: %s (%s)", cnfDeployment.Name, cnfSpec.Function))

		logger.Info("Successfully created CNF deployment",
			"deployment", cnfDeployment.Name,
			"function", cnfSpec.Function,
			"strategy", cnfSpec.DeploymentStrategy)
	}

	if len(deploymentContext.Errors) > 0 {
		return fmt.Errorf("failed to create %d out of %d CNF deployments",
			len(deploymentContext.Errors), len(deploymentContext.ProcessingResult.CNFDeployments))
	}

	return nil
}

// createCNFDeploymentResource creates a CNFDeployment resource from the intent specification
func (m *CNFIntegrationManager) createCNFDeploymentResource(networkIntent *nephoranv1.NetworkIntent, cnfSpec *nephoranv1.CNFDeploymentIntent, index int) (*nephoranv1.CNFDeployment, error) {
	// Generate CNF deployment name
	cnfName := fmt.Sprintf("%s-%s-%d", networkIntent.Name, strings.ToLower(string(cnfSpec.Function)), index)

	// Determine target namespace
	targetNamespace := m.Config.DefaultTargetNamespace
	if networkIntent.Spec.TargetNamespace != "" {
		targetNamespace = networkIntent.Spec.TargetNamespace
	}
	if cnfSpec.Function != "" {
		targetNamespace = fmt.Sprintf("cnf-%s", strings.ToLower(string(cnfSpec.Function)))
	}

	cnfDeployment := &nephoranv1.CNFDeployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cnfName,
			Namespace: targetNamespace,
			Labels: map[string]string{
				CNFDeploymentSourceLabel:    CNFDeploymentSourceValue,
				CNFDeploymentIntentLabel:    networkIntent.Name,
				CNFIntentLabel:              "true",
				"nephoran.com/cnf-type":     string(cnfSpec.CNFType),
				"nephoran.com/cnf-function": string(cnfSpec.Function),
			},
			Annotations: map[string]string{
				"nephoran.com/source-intent-uid":       string(networkIntent.UID),
				"nephoran.com/source-intent-namespace": networkIntent.Namespace,
				"nephoran.com/processing-timestamp":    time.Now().Format(time.RFC3339),
			},
		},
		Spec: nephoranv1.CNFDeploymentSpec{
			CNFType:            cnfSpec.CNFType,
			Function:           cnfSpec.Function,
			DeploymentStrategy: nephoranv1.CNFDeploymentStrategy(cnfSpec.DeploymentStrategy),
			Replicas:           1, // Default replicas
			Resources:          m.convertResourceIntent(cnfSpec.Resources),
			TargetNamespace:    targetNamespace,
			TargetCluster:      networkIntent.Spec.TargetCluster,
			NetworkSlice:       networkIntent.Spec.NetworkSlice,
		},
	}

	// Set replicas if specified
	if cnfSpec.Replicas != nil {
		cnfDeployment.Spec.Replicas = *cnfSpec.Replicas
	}

	// Convert auto-scaling intent
	if cnfSpec.AutoScaling != nil && cnfSpec.AutoScaling.Enabled {
		cnfDeployment.Spec.AutoScaling = m.convertAutoScalingIntent(cnfSpec.AutoScaling)
	}

	// Convert service mesh intent
	if cnfSpec.ServiceMesh != nil && cnfSpec.ServiceMesh.Enabled {
		cnfDeployment.Spec.ServiceMesh = m.convertServiceMeshIntent(cnfSpec.ServiceMesh)
	}

	// Convert monitoring intent
	if cnfSpec.Monitoring != nil && cnfSpec.Monitoring.Enabled {
		cnfDeployment.Spec.Monitoring = m.convertMonitoringIntent(cnfSpec.Monitoring)
	}

	// Set owner reference to NetworkIntent
	if err := controllerutil.SetControllerReference(networkIntent, cnfDeployment, m.Scheme); err != nil {
		return nil, fmt.Errorf("failed to set owner reference: %w", err)
	}

	// Set default Helm configuration if strategy is Helm
	if cnfDeployment.Spec.DeploymentStrategy == nephoranv1.DeploymentStrategyHelm {
		cnfDeployment.Spec.Helm = m.getDefaultHelmConfig(cnfSpec.Function)
	}

	return cnfDeployment, nil
}

// Helper methods for converting intent specifications to deployment specifications

func (m *CNFIntegrationManager) convertResourceIntent(resourceIntent *nephoranv1.CNFResourceIntent) nephoranv1.CNFResources {
	if resourceIntent == nil {
		// Return minimal default resources
		return nephoranv1.CNFResources{
			CPU:    mustParseQuantity("500m"),
			Memory: mustParseQuantity("1Gi"),
		}
	}

	resources := nephoranv1.CNFResources{}

	if resourceIntent.CPU != nil {
		resources.CPU = *resourceIntent.CPU
	} else {
		resources.CPU = mustParseQuantity("500m")
	}

	if resourceIntent.Memory != nil {
		resources.Memory = *resourceIntent.Memory
	} else {
		resources.Memory = mustParseQuantity("1Gi")
	}

	if resourceIntent.Storage != nil {
		resources.Storage = resourceIntent.Storage
	}

	if resourceIntent.GPU != nil {
		resources.GPU = resourceIntent.GPU
	}

	if resourceIntent.DPDK != nil && resourceIntent.DPDK.Enabled {
		resources.DPDK = &nephoranv1.DPDKConfig{
			Enabled: true,
			Cores:   resourceIntent.DPDK.Cores,
			Memory:  resourceIntent.DPDK.Memory,
			Driver:  resourceIntent.DPDK.Driver,
		}
	}

	if resourceIntent.Hugepages != nil {
		resources.Hugepages = resourceIntent.Hugepages
	}

	return resources
}

func (m *CNFIntegrationManager) convertAutoScalingIntent(autoScalingIntent *nephoranv1.AutoScalingIntent) *nephoranv1.AutoScaling {
	if autoScalingIntent == nil || !autoScalingIntent.Enabled {
		return nil
	}

	autoScaling := &nephoranv1.AutoScaling{
		Enabled:     true,
		MinReplicas: 1,
		MaxReplicas: 10,
	}

	if autoScalingIntent.MinReplicas != nil {
		autoScaling.MinReplicas = *autoScalingIntent.MinReplicas
	}

	if autoScalingIntent.MaxReplicas != nil {
		autoScaling.MaxReplicas = *autoScalingIntent.MaxReplicas
	}

	if autoScalingIntent.TargetCPUUtilization != nil {
		autoScaling.CPUUtilization = autoScalingIntent.TargetCPUUtilization
	}

	if autoScalingIntent.TargetMemoryUtilization != nil {
		autoScaling.MemoryUtilization = autoScalingIntent.TargetMemoryUtilization
	}

	// Convert custom metrics
	for _, metricName := range autoScalingIntent.CustomMetrics {
		customMetric := nephoranv1.CustomMetric{
			Name:        metricName,
			Type:        "pods", // Default type
			TargetValue: "10",   // Default target value
		}
		autoScaling.CustomMetrics = append(autoScaling.CustomMetrics, customMetric)
	}

	return autoScaling
}

func (m *CNFIntegrationManager) convertServiceMeshIntent(serviceMeshIntent *nephoranv1.ServiceMeshIntent) *nephoranv1.ServiceMeshConfig {
	if serviceMeshIntent == nil || !serviceMeshIntent.Enabled {
		return nil
	}

	serviceMesh := &nephoranv1.ServiceMeshConfig{
		Enabled: true,
		Type:    "istio", // Default to Istio
	}

	if serviceMeshIntent.Type != "" {
		serviceMesh.Type = serviceMeshIntent.Type
	}

	if serviceMeshIntent.MTLS != nil {
		serviceMesh.MTLS = &nephoranv1.MTLSConfig{
			Enabled: serviceMeshIntent.MTLS.Enabled,
			Mode:    serviceMeshIntent.MTLS.Mode,
		}
	}

	// Convert traffic management preferences to policies
	for _, trafficMgmt := range serviceMeshIntent.TrafficManagement {
		policy := nephoranv1.TrafficPolicy{
			Name:        fmt.Sprintf("policy-%s", trafficMgmt),
			Source:      "*",
			Destination: "*",
			LoadBalancing: &nephoranv1.LoadBalancingConfig{
				Algorithm: "round_robin",
			},
		}
		serviceMesh.TrafficPolicies = append(serviceMesh.TrafficPolicies, policy)
	}

	return serviceMesh
}

func (m *CNFIntegrationManager) convertMonitoringIntent(monitoringIntent *nephoranv1.MonitoringIntent) *nephoranv1.MonitoringConfig {
	if monitoringIntent == nil || !monitoringIntent.Enabled {
		return nil
	}

	monitoring := &nephoranv1.MonitoringConfig{
		Enabled: true,
		Prometheus: &nephoranv1.PrometheusConfig{
			Enabled:  true,
			Path:     "/metrics",
			Port:     9090,
			Interval: "30s",
		},
	}

	monitoring.CustomMetrics = monitoringIntent.Metrics
	monitoring.AlertingRules = monitoringIntent.Alerts

	return monitoring
}

func (m *CNFIntegrationManager) getDefaultHelmConfig(function nephoranv1.CNFFunction) *nephoranv1.HelmConfig {
	// Map CNF functions to their default Helm charts
	functionChartMap := map[nephoranv1.CNFFunction]nephoranv1.HelmConfig{
		nephoranv1.CNFFunctionAMF: {
			Repository:   "https://charts.5g-core.io",
			ChartName:    "amf",
			ChartVersion: "1.0.0",
		},
		nephoranv1.CNFFunctionSMF: {
			Repository:   "https://charts.5g-core.io",
			ChartName:    "smf",
			ChartVersion: "1.0.0",
		},
		nephoranv1.CNFFunctionUPF: {
			Repository:   "https://charts.5g-core.io",
			ChartName:    "upf",
			ChartVersion: "1.0.0",
		},
		nephoranv1.CNFFunctionNearRTRIC: {
			Repository:   "https://charts.o-ran.io",
			ChartName:    "near-rt-ric",
			ChartVersion: "1.0.0",
		},
		nephoranv1.CNFFunctionODU: {
			Repository:   "https://charts.o-ran.io",
			ChartName:    "o-du",
			ChartVersion: "1.0.0",
		},
	}

	if helmConfig, exists := functionChartMap[function]; exists {
		return &helmConfig
	}

	// Default Helm configuration
	return &nephoranv1.HelmConfig{
		Repository:   "https://charts.nephoran.io",
		ChartName:    strings.ToLower(string(function)),
		ChartVersion: "latest",
	}
}

// handleProcessingError handles errors during CNF processing
func (m *CNFIntegrationManager) handleProcessingError(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, deploymentContext *CNFDeploymentContext, processingErr error) error {
	logger := log.FromContext(ctx)

	// Update NetworkIntent status with error
	networkIntent.Status.ValidationErrors = append(networkIntent.Status.ValidationErrors,
		fmt.Sprintf("CNF processing failed: %v", processingErr))

	if err := m.Client.Status().Update(ctx, networkIntent); err != nil {
		logger.Error(err, "Failed to update NetworkIntent status with CNF processing error")
	}

	deploymentContext.Errors = append(deploymentContext.Errors, processingErr)
	return processingErr
}

// updateNetworkIntentWithCNFStatus updates NetworkIntent status with CNF deployment information
func (m *CNFIntegrationManager) updateNetworkIntentWithCNFStatus(ctx context.Context, networkIntent *nephoranv1.NetworkIntent, deploymentContext *CNFDeploymentContext) error {
	// Add CNF deployment information to NetworkIntent status
	if networkIntent.Status.DeployedComponents == nil {
		networkIntent.Status.DeployedComponents = []nephoranv1.NetworkTargetComponent{}
	}

	// Map CNF functions to target components
	for _, cnfDeployment := range deploymentContext.CNFDeployments {
		targetComponent := m.mapCNFFunctionToNetworkTargetComponent(cnfDeployment.Spec.Function)
		if targetComponent != "" {
			// Check if component is already in the list
			found := false
			for _, existing := range networkIntent.Status.DeployedComponents {
				if existing == targetComponent {
					found = true
					break
				}
			}
			if !found {
				networkIntent.Status.DeployedComponents = append(networkIntent.Status.DeployedComponents, targetComponent)
			}
		}
	}

	// Update processing duration - Note: ProcessingDuration field not defined in NetworkIntentStatus
	// if deploymentContext.ProcessingResult != nil {
	//	duration := time.Since(deploymentContext.StartTime)
	//	networkIntent.Status.ProcessingDuration = &metav1.Duration{Duration: duration}
	// }

	return m.Client.Status().Update(ctx, networkIntent)
}

// mapCNFFunctionToNetworkTargetComponent maps CNF functions to NetworkTargetComponent enum
func (m *CNFIntegrationManager) mapCNFFunctionToNetworkTargetComponent(function nephoranv1.CNFFunction) nephoranv1.NetworkTargetComponent {
	mapping := map[nephoranv1.CNFFunction]nephoranv1.NetworkTargetComponent{
		nephoranv1.CNFFunctionAMF:       nephoranv1.NetworkTargetComponentAMF,
		nephoranv1.CNFFunctionSMF:       nephoranv1.NetworkTargetComponentSMF,
		nephoranv1.CNFFunctionUPF:       nephoranv1.NetworkTargetComponentUPF,
		nephoranv1.CNFFunctionNRF:       nephoranv1.NetworkTargetComponentNRF,
		nephoranv1.CNFFunctionAUSF:      nephoranv1.NetworkTargetComponentAUSF,
		nephoranv1.CNFFunctionUDM:       nephoranv1.NetworkTargetComponentUDM,
		nephoranv1.CNFFunctionPCF:       nephoranv1.NetworkTargetComponentPCF,
		nephoranv1.CNFFunctionNSSF:      nephoranv1.NetworkTargetComponentNSSF,
		nephoranv1.CNFFunctionODU:       nephoranv1.NetworkTargetComponentDU,
		nephoranv1.CNFFunctionOCUCP:     nephoranv1.NetworkTargetComponentCUCP,
		nephoranv1.CNFFunctionOCUUP:     nephoranv1.NetworkTargetComponentCUUP,
		// Note: These constants don't exist in NetworkTargetComponent, using generic component
		// nephoranv1.CNFFunctionNearRTRIC: nephoranv1.NetworkTargetComponentNearRTRIC,
		// nephoranv1.CNFFunctionNonRTRIC:  nephoranv1.NetworkTargetComponentNonRTRIC,
		// nephoranv1.CNFFunctionOENB:      nephoranv1.NetworkTargetComponentENB,
		// nephoranv1.CNFFunctionSMO:       nephoranv1.NetworkTargetComponentSMO,
		// nephoranv1.CNFFunctionRApp:      nephoranv1.NetworkTargetComponentRAPP,
		// nephoranv1.CNFFunctionXApp:      nephoranv1.NetworkTargetComponentXAPP,
	}

	return mapping[function]
}

// mustParseQuantity parses a quantity string and panics if it fails (for constants)
func mustParseQuantity(s string) resource.Quantity {
	q, err := resource.ParseQuantity(s)
	if err != nil {
		panic(fmt.Sprintf("failed to parse quantity %s: %v", s, err))
	}
	return q
}
