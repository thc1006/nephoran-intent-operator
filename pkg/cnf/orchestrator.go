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

package cnf

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran"
	"github.com/thc1006/nephoran-intent-operator/pkg/shared"
)

const (
	CNFOrchestratorFinalizer = "cnfdeployment.nephoran.com/finalizer"
	
	// Default timeout values
	DefaultCNFDeployTimeout = 10 * time.Minute
	DefaultCNFDeleteTimeout = 5 * time.Minute
	DefaultHealthCheckTimeout = 2 * time.Minute
	
	// Chart repository defaults
	Default5GCoreChartsRepo = "https://charts.5g-core.io"
	DefaultORANChartsRepo   = "https://charts.o-ran.io"
	DefaultEdgeChartsRepo   = "https://charts.edge.io"
	
	// Events
	EventCNFDeploymentStarted   = "CNFDeploymentStarted"
	EventCNFDeploymentCompleted = "CNFDeploymentCompleted"
	EventCNFDeploymentFailed    = "CNFDeploymentFailed"
	EventCNFScalingStarted      = "CNFScalingStarted"
	EventCNFScalingCompleted    = "CNFScalingCompleted"
	EventCNFHealthCheckFailed   = "CNFHealthCheckFailed"
)

// CNFOrchestrator manages the lifecycle of Cloud Native Functions
type CNFOrchestrator struct {
	client.Client
	Scheme             *runtime.Scheme
	Recorder           record.EventRecorder
	PackageGenerator   *nephio.PackageGenerator
	GitClient          git.ClientInterface
	MetricsCollector   *monitoring.MetricsCollector
	ORANClient         *oran.Client
	HelmSettings       *cli.EnvSettings
	
	// Configuration
	Config *CNFOrchestratorConfig
	
	// Chart repositories
	ChartRepositories map[nephoranv1.CNFType]string
	
	// Template registry
	TemplateRegistry *CNFTemplateRegistry
}

// CNFOrchestratorConfig holds configuration for the CNF orchestrator
type CNFOrchestratorConfig struct {
	DefaultNamespace     string
	EnableServiceMesh    bool
	ServiceMeshType      string
	EnableMonitoring     bool
	MonitoringNamespace  string
	EnableBackup         bool
	BackupStorage        string
	GitRepoURL           string
	GitBranch            string
	GitDeployPath        string
	HelmTimeout          time.Duration
	MaxConcurrentDeploys int
}

// CNFTemplateRegistry manages CNF deployment templates
type CNFTemplateRegistry struct {
	Templates map[nephoranv1.CNFFunction]*CNFTemplate
}

// CNFTemplate defines a deployment template for a specific CNF function
type CNFTemplate struct {
	Function        nephoranv1.CNFFunction
	ChartReference  ChartReference
	DefaultValues   map[string]interface{}
	RequiredConfigs []string
	Dependencies    []nephoranv1.CNFFunction
	Interfaces      []InterfaceSpec
	HealthChecks    []HealthCheckSpec
	MonitoringSpecs []MonitoringSpec
}

// ChartReference specifies Helm chart information
type ChartReference struct {
	Repository   string
	ChartName    string
	ChartVersion string
	Values       map[string]interface{}
}

// InterfaceSpec defines network interface specifications
type InterfaceSpec struct {
	Name         string
	Type         string // N1, N2, N3, N4, N6, SBI, etc.
	Protocol     []string
	Port         int32
	Mandatory    bool
	Dependencies []string
}

// HealthCheckSpec defines health check specifications
type HealthCheckSpec struct {
	Name        string
	Type        string // HTTP, TCP, gRPC
	Path        string
	Port        int32
	Interval    time.Duration
	Timeout     time.Duration
	Retries     int32
	InitialDelay time.Duration
}

// MonitoringSpec defines monitoring specifications
type MonitoringSpec struct {
	MetricName   string
	MetricType   string
	Path         string
	Port         int32
	AlertRules   []AlertRule
}

// AlertRule defines alerting rule
type AlertRule struct {
	Name        string
	Expression  string
	Duration    time.Duration
	Severity    string
	Description string
}

// DeploymentResult represents the result of a CNF deployment
type DeploymentResult struct {
	Success          bool
	ReleaseName      string
	Namespace        string
	ServiceEndpoints []nephoranv1.ServiceEndpoint
	ResourceStatus   map[string]string
	Errors           []string
	Duration         time.Duration
}

// NewCNFOrchestrator creates a new CNF orchestrator instance
func NewCNFOrchestrator(client client.Client, scheme *runtime.Scheme, recorder record.EventRecorder) *CNFOrchestrator {
	orchestrator := &CNFOrchestrator{
		Client:   client,
		Scheme:   scheme,
		Recorder: recorder,
		Config: &CNFOrchestratorConfig{
			DefaultNamespace:     "default",
			EnableServiceMesh:    true,
			ServiceMeshType:      "istio",
			EnableMonitoring:     true,
			MonitoringNamespace:  "monitoring",
			EnableBackup:         true,
			BackupStorage:        "s3",
			HelmTimeout:          DefaultCNFDeployTimeout,
			MaxConcurrentDeploys: 5,
		},
		ChartRepositories: map[nephoranv1.CNFType]string{
			nephoranv1.CNF5GCore: Default5GCoreChartsRepo,
			nephoranv1.CNFORAN:   DefaultORANChartsRepo,
			nephoranv1.CNFEdge:   DefaultEdgeChartsRepo,
		},
		HelmSettings: cli.New(),
	}
	
	orchestrator.initializeTemplateRegistry()
	return orchestrator
}

// DeployRequest represents a CNF deployment request
type DeployRequest struct {
	CNFDeployment   *nephoranv1.CNFDeployment
	NetworkIntent   *nephoranv1.NetworkIntent
	Context         context.Context
	ProcessingPhase string
}

// Deploy orchestrates the deployment of a CNF
func (c *CNFOrchestrator) Deploy(ctx context.Context, req *DeployRequest) (*DeploymentResult, error) {
	logger := log.FromContext(ctx)
	logger.Info("Starting CNF deployment", "cnf", req.CNFDeployment.Name, "function", req.CNFDeployment.Spec.Function)
	
	// Record deployment start event
	c.Recorder.Event(req.CNFDeployment, "Normal", EventCNFDeploymentStarted, 
		fmt.Sprintf("Starting deployment of %s CNF", req.CNFDeployment.Spec.Function))
	
	startTime := time.Now()
	result := &DeploymentResult{}
	
	// Validate deployment request
	if err := c.validateDeploymentRequest(req); err != nil {
		return nil, fmt.Errorf("deployment validation failed: %w", err)
	}
	
	// Get CNF template
	template, err := c.getCNFTemplate(req.CNFDeployment.Spec.Function)
	if err != nil {
		return nil, fmt.Errorf("failed to get CNF template: %w", err)
	}
	
	// Check dependencies
	if err := c.checkDependencies(ctx, req.CNFDeployment, template); err != nil {
		return nil, fmt.Errorf("dependency check failed: %w", err)
	}
	
	// Prepare deployment configuration
	config, err := c.prepareDeploymentConfig(req.CNFDeployment, template)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare deployment config: %w", err)
	}
	
	// Deploy based on strategy
	switch req.CNFDeployment.Spec.DeploymentStrategy {
	case nephoranv1.DeploymentStrategyHelm:
		result, err = c.deployViaHelm(ctx, req.CNFDeployment, config)
	case nephoranv1.DeploymentStrategyOperator:
		result, err = c.deployViaOperator(ctx, req.CNFDeployment, config)
	case nephoranv1.DeploymentStrategyGitOps:
		result, err = c.deployViaGitOps(ctx, req.CNFDeployment, config)
	case nephoranv1.DeploymentStrategyDirect:
		result, err = c.deployDirect(ctx, req.CNFDeployment, config)
	default:
		return nil, fmt.Errorf("unsupported deployment strategy: %s", req.CNFDeployment.Spec.DeploymentStrategy)
	}
	
	if err != nil {
		c.Recorder.Event(req.CNFDeployment, "Warning", EventCNFDeploymentFailed,
			fmt.Sprintf("CNF deployment failed: %v", err))
		return result, err
	}
	
	// Configure service mesh if enabled
	if req.CNFDeployment.Spec.ServiceMesh != nil && req.CNFDeployment.Spec.ServiceMesh.Enabled {
		if err := c.configureServiceMesh(ctx, req.CNFDeployment, result); err != nil {
			logger.Error(err, "Failed to configure service mesh")
			// Don't fail deployment, just log the error
		}
	}
	
	// Setup monitoring
	if req.CNFDeployment.Spec.Monitoring != nil && req.CNFDeployment.Spec.Monitoring.Enabled {
		if err := c.setupMonitoring(ctx, req.CNFDeployment, template); err != nil {
			logger.Error(err, "Failed to setup monitoring")
			// Don't fail deployment, just log the error
		}
	}
	
	// Configure auto-scaling
	if req.CNFDeployment.Spec.AutoScaling != nil && req.CNFDeployment.Spec.AutoScaling.Enabled {
		if err := c.configureAutoScaling(ctx, req.CNFDeployment); err != nil {
			logger.Error(err, "Failed to configure auto-scaling")
			// Don't fail deployment, just log the error
		}
	}
	
	// Perform health checks
	if err := c.performHealthChecks(ctx, req.CNFDeployment, template); err != nil {
		logger.Error(err, "Health checks failed")
		c.Recorder.Event(req.CNFDeployment, "Warning", EventCNFHealthCheckFailed,
			fmt.Sprintf("Health checks failed: %v", err))
	}
	
	result.Duration = time.Since(startTime)
	result.Success = true
	
	// Record successful deployment
	c.Recorder.Event(req.CNFDeployment, "Normal", EventCNFDeploymentCompleted,
		fmt.Sprintf("CNF deployment completed successfully in %v", result.Duration))
	
	// Update metrics
	if c.MetricsCollector != nil {
		c.MetricsCollector.RecordCNFDeployment(req.CNFDeployment.Spec.Function, result.Duration)
	}
	
	logger.Info("CNF deployment completed successfully", 
		"cnf", req.CNFDeployment.Name, 
		"function", req.CNFDeployment.Spec.Function,
		"duration", result.Duration)
	
	return result, nil
}

// validateDeploymentRequest validates the deployment request
func (c *CNFOrchestrator) validateDeploymentRequest(req *DeployRequest) error {
	if req.CNFDeployment == nil {
		return fmt.Errorf("CNF deployment is required")
	}
	
	if err := req.CNFDeployment.ValidateCNFDeployment(); err != nil {
		return fmt.Errorf("CNF deployment validation failed: %w", err)
	}
	
	// Validate strategy-specific requirements
	switch req.CNFDeployment.Spec.DeploymentStrategy {
	case nephoranv1.DeploymentStrategyHelm:
		if req.CNFDeployment.Spec.Helm == nil {
			return fmt.Errorf("Helm configuration is required for Helm deployment strategy")
		}
	case nephoranv1.DeploymentStrategyOperator:
		if req.CNFDeployment.Spec.Operator == nil {
			return fmt.Errorf("Operator configuration is required for Operator deployment strategy")
		}
	}
	
	return nil
}

// getCNFTemplate retrieves the deployment template for a CNF function
func (c *CNFOrchestrator) getCNFTemplate(function nephoranv1.CNFFunction) (*CNFTemplate, error) {
	if c.TemplateRegistry == nil {
		return nil, fmt.Errorf("template registry not initialized")
	}
	
	template, exists := c.TemplateRegistry.Templates[function]
	if !exists {
		return nil, fmt.Errorf("no template found for CNF function: %s", function)
	}
	
	return template, nil
}

// checkDependencies checks if required dependencies are deployed
func (c *CNFOrchestrator) checkDependencies(ctx context.Context, cnf *nephoranv1.CNFDeployment, template *CNFTemplate) error {
	if len(template.Dependencies) == 0 {
		return nil
	}
	
	logger := log.FromContext(ctx)
	logger.Info("Checking CNF dependencies", "dependencies", template.Dependencies)
	
	for _, dep := range template.Dependencies {
		// Check if dependency is deployed in the same namespace
		dependencyList := &nephoranv1.CNFDeploymentList{}
		listOpts := []client.ListOption{
			client.InNamespace(cnf.Namespace),
			client.MatchingFields{"spec.function": string(dep)},
		}
		
		if err := c.List(ctx, dependencyList, listOpts...); err != nil {
			return fmt.Errorf("failed to check dependency %s: %w", dep, err)
		}
		
		found := false
		for _, depCNF := range dependencyList.Items {
			if depCNF.Status.Phase == "Running" {
				found = true
				break
			}
		}
		
		if !found {
			return fmt.Errorf("required dependency %s is not deployed or not running", dep)
		}
	}
	
	return nil
}

// prepareDeploymentConfig prepares the deployment configuration
func (c *CNFOrchestrator) prepareDeploymentConfig(cnf *nephoranv1.CNFDeployment, template *CNFTemplate) (map[string]interface{}, error) {
	config := make(map[string]interface{})
	
	// Start with template default values
	for k, v := range template.DefaultValues {
		config[k] = v
	}
	
	// Apply CNF-specific configuration
	if cnf.Spec.Configuration.Raw != nil {
		var cnfConfig map[string]interface{}
		if err := json.Unmarshal(cnf.Spec.Configuration.Raw, &cnfConfig); err != nil {
			return nil, fmt.Errorf("failed to unmarshal CNF configuration: %w", err)
		}
		
		for k, v := range cnfConfig {
			config[k] = v
		}
	}
	
	// Set resource requirements
	config["resources"] = map[string]interface{}{
		"requests": map[string]interface{}{
			"cpu":    cnf.Spec.Resources.CPU.String(),
			"memory": cnf.Spec.Resources.Memory.String(),
		},
		"limits": map[string]interface{}{
			"cpu":    cnf.Spec.Resources.CPU.String(),
			"memory": cnf.Spec.Resources.Memory.String(),
		},
	}
	
	// Override limits if specified
	if cnf.Spec.Resources.MaxCPU != nil {
		limits := config["resources"].(map[string]interface{})["limits"].(map[string]interface{})
		limits["cpu"] = cnf.Spec.Resources.MaxCPU.String()
	}
	
	if cnf.Spec.Resources.MaxMemory != nil {
		limits := config["resources"].(map[string]interface{})["limits"].(map[string]interface{})
		limits["memory"] = cnf.Spec.Resources.MaxMemory.String()
	}
	
	// Set replica count
	config["replicaCount"] = cnf.Spec.Replicas
	
	// Configure storage if specified
	if cnf.Spec.Resources.Storage != nil {
		config["persistence"] = map[string]interface{}{
			"enabled": true,
			"size":    cnf.Spec.Resources.Storage.String(),
		}
	}
	
	// Configure DPDK if specified
	if cnf.Spec.Resources.DPDK != nil && cnf.Spec.Resources.DPDK.Enabled {
		config["dpdk"] = map[string]interface{}{
			"enabled": true,
			"cores":   cnf.Spec.Resources.DPDK.Cores,
			"memory":  cnf.Spec.Resources.DPDK.Memory,
			"driver":  cnf.Spec.Resources.DPDK.Driver,
		}
	}
	
	// Configure hugepages if specified
	if cnf.Spec.Resources.Hugepages != nil {
		config["hugepages"] = cnf.Spec.Resources.Hugepages
	}
	
	return config, nil
}

// deployViaHelm deploys CNF using Helm charts
func (c *CNFOrchestrator) deployViaHelm(ctx context.Context, cnf *nephoranv1.CNFDeployment, config map[string]interface{}) (*DeploymentResult, error) {
	logger := log.FromContext(ctx)
	logger.Info("Deploying CNF via Helm", "cnf", cnf.Name, "chart", cnf.Spec.Helm.ChartName)
	
	// Prepare Helm configuration
	actionConfig := new(action.Configuration)
	if err := actionConfig.Init(c.HelmSettings.RESTClientGetter(), cnf.Namespace, "memory", func(format string, v ...interface{}) {
		logger.Info(fmt.Sprintf(format, v...))
	}); err != nil {
		return nil, fmt.Errorf("failed to initialize Helm action config: %w", err)
	}
	
	// Determine release name
	releaseName := cnf.Spec.Helm.ReleaseName
	if releaseName == "" {
		releaseName = fmt.Sprintf("%s-%s", strings.ToLower(string(cnf.Spec.Function)), cnf.Name)
	}
	
	// Install or upgrade the chart
	installAction := action.NewInstall(actionConfig)
	installAction.ReleaseName = releaseName
	installAction.Namespace = cnf.Namespace
	installAction.CreateNamespace = true
	installAction.Timeout = c.Config.HelmTimeout
	
	// Load chart
	chart, err := loader.LoadDir(cnf.Spec.Helm.ChartName) // This would need proper chart loading
	if err != nil {
		return nil, fmt.Errorf("failed to load Helm chart: %w", err)
	}
	
	// Merge values
	values := config
	if cnf.Spec.Helm.Values.Raw != nil {
		var helmValues map[string]interface{}
		if err := json.Unmarshal(cnf.Spec.Helm.Values.Raw, &helmValues); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Helm values: %w", err)
		}
		for k, v := range helmValues {
			values[k] = v
		}
	}
	
	// Install the release
	release, err := installAction.Run(chart, values)
	if err != nil {
		return nil, fmt.Errorf("failed to install Helm chart: %w", err)
	}
	
	result := &DeploymentResult{
		Success:     true,
		ReleaseName: release.Name,
		Namespace:   release.Namespace,
	}
	
	return result, nil
}

// deployViaOperator deploys CNF using Kubernetes operators
func (c *CNFOrchestrator) deployViaOperator(ctx context.Context, cnf *nephoranv1.CNFDeployment, config map[string]interface{}) (*DeploymentResult, error) {
	logger := log.FromContext(ctx)
	logger.Info("Deploying CNF via Operator", "cnf", cnf.Name, "operator", cnf.Spec.Operator.Name)
	
	// Create the custom resource for the operator
	var cr runtime.Object
	if err := json.Unmarshal(cnf.Spec.Operator.CustomResource.Raw, &cr); err != nil {
		return nil, fmt.Errorf("failed to unmarshal custom resource: %w", err)
	}
	
	// Apply the custom resource
	if err := c.Create(ctx, cr); err != nil && !errors.IsAlreadyExists(err) {
		return nil, fmt.Errorf("failed to create custom resource: %w", err)
	}
	
	result := &DeploymentResult{
		Success:   true,
		Namespace: cnf.Namespace,
	}
	
	return result, nil
}

// deployViaGitOps deploys CNF using GitOps workflow
func (c *CNFOrchestrator) deployViaGitOps(ctx context.Context, cnf *nephoranv1.CNFDeployment, config map[string]interface{}) (*DeploymentResult, error) {
	logger := log.FromContext(ctx)
	logger.Info("Deploying CNF via GitOps", "cnf", cnf.Name)
	
	if c.PackageGenerator == nil {
		return nil, fmt.Errorf("package generator not configured")
	}
	
	if c.GitClient == nil {
		return nil, fmt.Errorf("git client not configured")
	}
	
	// Generate Nephio package
	packageData, err := c.PackageGenerator.GenerateCNFPackage(cnf, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate CNF package: %w", err)
	}
	
	// Commit to Git repository
	commitMsg := fmt.Sprintf("Deploy %s CNF: %s", cnf.Spec.Function, cnf.Name)
	commitHash, err := c.GitClient.CommitPackage(ctx, packageData, commitMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to commit CNF package: %w", err)
	}
	
	result := &DeploymentResult{
		Success:   true,
		Namespace: cnf.Namespace,
		ResourceStatus: map[string]string{
			"gitCommit": commitHash,
		},
	}
	
	return result, nil
}

// deployDirect deploys CNF using direct Kubernetes manifests
func (c *CNFOrchestrator) deployDirect(ctx context.Context, cnf *nephoranv1.CNFDeployment, config map[string]interface{}) (*DeploymentResult, error) {
	logger := log.FromContext(ctx)
	logger.Info("Deploying CNF directly", "cnf", cnf.Name)
	
	// Generate Kubernetes manifests
	manifests, err := c.generateDirectManifests(cnf, config)
	if err != nil {
		return nil, fmt.Errorf("failed to generate manifests: %w", err)
	}
	
	// Apply manifests
	for _, manifest := range manifests {
		if err := c.Apply(ctx, manifest); err != nil {
			return nil, fmt.Errorf("failed to apply manifest: %w", err)
		}
	}
	
	result := &DeploymentResult{
		Success:   true,
		Namespace: cnf.Namespace,
	}
	
	return result, nil
}

// generateDirectManifests generates Kubernetes manifests for direct deployment
func (c *CNFOrchestrator) generateDirectManifests(cnf *nephoranv1.CNFDeployment, config map[string]interface{}) ([]client.Object, error) {
	// This would contain logic to generate appropriate Kubernetes resources
	// based on the CNF function type and configuration
	
	// For now, return empty slice - this would be implemented with specific
	// manifest generation logic for each CNF type
	return []client.Object{}, nil
}

// configureServiceMesh configures service mesh integration
func (c *CNFOrchestrator) configureServiceMesh(ctx context.Context, cnf *nephoranv1.CNFDeployment, result *DeploymentResult) error {
	logger := log.FromContext(ctx)
	logger.Info("Configuring service mesh", "cnf", cnf.Name, "mesh", cnf.Spec.ServiceMesh.Type)
	
	// Service mesh configuration would be implemented here
	// This is a placeholder for the actual implementation
	
	return nil
}

// setupMonitoring sets up monitoring for the CNF
func (c *CNFOrchestrator) setupMonitoring(ctx context.Context, cnf *nephoranv1.CNFDeployment, template *CNFTemplate) error {
	logger := log.FromContext(ctx)
	logger.Info("Setting up monitoring", "cnf", cnf.Name)
	
	// Monitoring setup would be implemented here
	// This is a placeholder for the actual implementation
	
	return nil
}

// configureAutoScaling configures auto-scaling for the CNF
func (c *CNFOrchestrator) configureAutoScaling(ctx context.Context, cnf *nephoranv1.CNFDeployment) error {
	logger := log.FromContext(ctx)
	logger.Info("Configuring auto-scaling", "cnf", cnf.Name)
	
	// Auto-scaling configuration would be implemented here
	// This is a placeholder for the actual implementation
	
	return nil
}

// performHealthChecks performs health checks on the deployed CNF
func (c *CNFOrchestrator) performHealthChecks(ctx context.Context, cnf *nephoranv1.CNFDeployment, template *CNFTemplate) error {
	logger := log.FromContext(ctx)
	logger.Info("Performing health checks", "cnf", cnf.Name)
	
	// Health check implementation would be here
	// This is a placeholder for the actual implementation
	
	return nil
}

// initializeTemplateRegistry initializes the CNF template registry with predefined templates
func (c *CNFOrchestrator) initializeTemplateRegistry() {
	c.TemplateRegistry = &CNFTemplateRegistry{
		Templates: make(map[nephoranv1.CNFFunction]*CNFTemplate),
	}
	
	// Initialize 5G Core function templates
	c.init5GCoreTemplates()
	
	// Initialize O-RAN function templates
	c.initORANTemplates()
	
	// Initialize edge function templates
	c.initEdgeTemplates()
}

// init5GCoreTemplates initializes templates for 5G Core functions
func (c *CNFOrchestrator) init5GCoreTemplates() {
	// AMF template
	c.TemplateRegistry.Templates[nephoranv1.CNFFunctionAMF] = &CNFTemplate{
		Function: nephoranv1.CNFFunctionAMF,
		ChartReference: ChartReference{
			Repository:   c.ChartRepositories[nephoranv1.CNF5GCore],
			ChartName:    "amf",
			ChartVersion: "1.0.0",
		},
		DefaultValues: map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "5gc/amf",
				"tag":        "latest",
			},
			"service": map[string]interface{}{
				"type": "ClusterIP",
				"ports": map[string]interface{}{
					"sbi": 8080,
					"sctp": 38412,
				},
			},
		},
		RequiredConfigs: []string{"plmnId", "amfId", "guami"},
		Interfaces: []InterfaceSpec{
			{
				Name:      "N1",
				Type:      "NAS",
				Protocol:  []string{"NAS"},
				Port:      38412,
				Mandatory: true,
			},
			{
				Name:      "N2",
				Type:      "NGAP",
				Protocol:  []string{"SCTP"},
				Port:      38412,
				Mandatory: true,
			},
			{
				Name:      "SBI",
				Type:      "HTTP",
				Protocol:  []string{"HTTP2"},
				Port:      8080,
				Mandatory: true,
			},
		},
		HealthChecks: []HealthCheckSpec{
			{
				Name:         "http-health",
				Type:         "HTTP",
				Path:         "/health",
				Port:         8080,
				Interval:     30 * time.Second,
				Timeout:      10 * time.Second,
				Retries:      3,
				InitialDelay: 30 * time.Second,
			},
		},
	}
	
	// SMF template
	c.TemplateRegistry.Templates[nephoranv1.CNFFunctionSMF] = &CNFTemplate{
		Function: nephoranv1.CNFFunctionSMF,
		ChartReference: ChartReference{
			Repository:   c.ChartRepositories[nephoranv1.CNF5GCore],
			ChartName:    "smf",
			ChartVersion: "1.0.0",
		},
		DefaultValues: map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "5gc/smf",
				"tag":        "latest",
			},
			"service": map[string]interface{}{
				"type": "ClusterIP",
				"ports": map[string]interface{}{
					"sbi": 8080,
				},
			},
		},
		RequiredConfigs: []string{"plmnId", "nfInstanceId"},
		Dependencies:    []nephoranv1.CNFFunction{nephoranv1.CNFFunctionNRF, nephoranv1.CNFFunctionUDM},
		Interfaces: []InterfaceSpec{
			{
				Name:      "N4",
				Type:      "PFCP",
				Protocol:  []string{"UDP"},
				Port:      8805,
				Mandatory: true,
			},
			{
				Name:      "SBI",
				Type:      "HTTP",
				Protocol:  []string{"HTTP2"},
				Port:      8080,
				Mandatory: true,
			},
		},
		HealthChecks: []HealthCheckSpec{
			{
				Name:         "http-health",
				Type:         "HTTP",
				Path:         "/health",
				Port:         8080,
				Interval:     30 * time.Second,
				Timeout:      10 * time.Second,
				Retries:      3,
				InitialDelay: 30 * time.Second,
			},
		},
	}
	
	// UPF template
	c.TemplateRegistry.Templates[nephoranv1.CNFFunctionUPF] = &CNFTemplate{
		Function: nephoranv1.CNFFunctionUPF,
		ChartReference: ChartReference{
			Repository:   c.ChartRepositories[nephoranv1.CNF5GCore],
			ChartName:    "upf",
			ChartVersion: "1.0.0",
		},
		DefaultValues: map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "5gc/upf",
				"tag":        "latest",
			},
			"service": map[string]interface{}{
				"type": "ClusterIP",
				"ports": map[string]interface{}{
					"pfcp": 8805,
					"gtpu": 2152,
				},
			},
			"dpdk": map[string]interface{}{
				"enabled": true,
				"cores":   4,
				"memory":  2048,
			},
		},
		RequiredConfigs: []string{"dnn", "pfcpAddress", "gtpuAddress"},
		Interfaces: []InterfaceSpec{
			{
				Name:      "N3",
				Type:      "GTP-U",
				Protocol:  []string{"UDP"},
				Port:      2152,
				Mandatory: true,
			},
			{
				Name:      "N4",
				Type:      "PFCP",
				Protocol:  []string{"UDP"},
				Port:      8805,
				Mandatory: true,
			},
			{
				Name:      "N6",
				Type:      "Data",
				Protocol:  []string{"IP"},
				Port:      0,
				Mandatory: true,
			},
		},
		HealthChecks: []HealthCheckSpec{
			{
				Name:         "pfcp-health",
				Type:         "UDP",
				Port:         8805,
				Interval:     30 * time.Second,
				Timeout:      10 * time.Second,
				Retries:      3,
				InitialDelay: 30 * time.Second,
			},
		},
	}
}

// initORANTemplates initializes templates for O-RAN functions
func (c *CNFOrchestrator) initORANTemplates() {
	// Near-RT RIC template
	c.TemplateRegistry.Templates[nephoranv1.CNFFunctionNearRTRIC] = &CNFTemplate{
		Function: nephoranv1.CNFFunctionNearRTRIC,
		ChartReference: ChartReference{
			Repository:   c.ChartRepositories[nephoranv1.CNFORAN],
			ChartName:    "near-rt-ric",
			ChartVersion: "1.0.0",
		},
		DefaultValues: map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "oran/near-rt-ric",
				"tag":        "latest",
			},
			"service": map[string]interface{}{
				"type": "ClusterIP",
				"ports": map[string]interface{}{
					"a1": 10000,
					"e2": 36421,
				},
			},
		},
		RequiredConfigs: []string{"ricId", "plmnId"},
		Interfaces: []InterfaceSpec{
			{
				Name:      "A1",
				Type:      "REST",
				Protocol:  []string{"HTTP"},
				Port:      10000,
				Mandatory: true,
			},
			{
				Name:      "E2",
				Type:      "SCTP",
				Protocol:  []string{"SCTP"},
				Port:      36421,
				Mandatory: true,
			},
		},
		HealthChecks: []HealthCheckSpec{
			{
				Name:         "http-health",
				Type:         "HTTP",
				Path:         "/a1-p/healthcheck",
				Port:         10000,
				Interval:     30 * time.Second,
				Timeout:      10 * time.Second,
				Retries:      3,
				InitialDelay: 30 * time.Second,
			},
		},
	}
	
	// O-DU template
	c.TemplateRegistry.Templates[nephoranv1.CNFFunctionODU] = &CNFTemplate{
		Function: nephoranv1.CNFFunctionODU,
		ChartReference: ChartReference{
			Repository:   c.ChartRepositories[nephoranv1.CNFORAN],
			ChartName:    "o-du",
			ChartVersion: "1.0.0",
		},
		DefaultValues: map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "oran/o-du",
				"tag":        "latest",
			},
			"service": map[string]interface{}{
				"type": "ClusterIP",
				"ports": map[string]interface{}{
					"f1": 38472,
					"e2": 36421,
				},
			},
		},
		RequiredConfigs: []string{"duId", "cellId"},
		Dependencies:    []nephoranv1.CNFFunction{nephoranv1.CNFFunctionNearRTRIC},
		Interfaces: []InterfaceSpec{
			{
				Name:      "F1-C",
				Type:      "F1AP",
				Protocol:  []string{"SCTP"},
				Port:      38472,
				Mandatory: true,
			},
			{
				Name:      "F1-U",
				Type:      "GTP-U",
				Protocol:  []string{"UDP"},
				Port:      2152,
				Mandatory: true,
			},
			{
				Name:      "E2",
				Type:      "SCTP",
				Protocol:  []string{"SCTP"},
				Port:      36421,
				Mandatory: true,
			},
		},
		HealthChecks: []HealthCheckSpec{
			{
				Name:         "f1-health",
				Type:         "TCP",
				Port:         38472,
				Interval:     30 * time.Second,
				Timeout:      10 * time.Second,
				Retries:      3,
				InitialDelay: 30 * time.Second,
			},
		},
	}
}

// initEdgeTemplates initializes templates for edge functions
func (c *CNFOrchestrator) initEdgeTemplates() {
	// UE Simulator template
	c.TemplateRegistry.Templates[nephoranv1.CNFFunctionUESimulator] = &CNFTemplate{
		Function: nephoranv1.CNFFunctionUESimulator,
		ChartReference: ChartReference{
			Repository:   c.ChartRepositories[nephoranv1.CNFEdge],
			ChartName:    "ue-simulator",
			ChartVersion: "1.0.0",
		},
		DefaultValues: map[string]interface{}{
			"image": map[string]interface{}{
				"repository": "edge/ue-simulator",
				"tag":        "latest",
			},
			"ues": map[string]interface{}{
				"count": 100,
				"imsiStart": "001010000000001",
			},
		},
		RequiredConfigs: []string{"amfAddress", "gnbAddress"},
	}
}