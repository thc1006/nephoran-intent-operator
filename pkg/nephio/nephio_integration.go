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

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// NephioIntegration provides the main integration point for Nephio-native operations
// This is the primary interface for the Nephoran Intent Operator to interact with Nephio
type NephioIntegration struct {
	client               client.Client
	porchClient          porch.PorchClient
	workflowOrchestrator *NephioWorkflowOrchestrator
	packageCatalog       *NephioPackageCatalog
	workloadRegistry     *WorkloadClusterRegistry
	configSync           *ConfigSyncClient
	workflowEngine       *NephioWorkflowEngine
	config               *NephioIntegrationConfig
}

// NephioIntegrationConfig defines configuration for Nephio integration
type NephioIntegrationConfig struct {
	// Core Nephio settings
	NephioNamespace   string `json:"nephioNamespace" yaml:"nephioNamespace"`
	PorchAPI          string `json:"porchAPI" yaml:"porchAPI"`
	ConfigSyncEnabled bool   `json:"configSyncEnabled" yaml:"configSyncEnabled"`

	// Repository settings
	UpstreamRepository   string `json:"upstreamRepository" yaml:"upstreamRepository"`
	DownstreamRepository string `json:"downstreamRepository" yaml:"downstreamRepository"`
	CatalogRepository    string `json:"catalogRepository" yaml:"catalogRepository"`

	// Workflow settings
	WorkflowOrchestrator *WorkflowOrchestratorConfig `json:"workflowOrchestrator,omitempty" yaml:"workflowOrchestrator,omitempty"`
	PackageCatalog       *PackageCatalogConfig       `json:"packageCatalog,omitempty" yaml:"packageCatalog,omitempty"`
	WorkloadRegistry     *WorkloadClusterConfig      `json:"workloadRegistry,omitempty" yaml:"workloadRegistry,omitempty"`
	ConfigSync           *ConfigSyncConfig           `json:"configSync,omitempty" yaml:"configSync,omitempty"`
	WorkflowEngine       *WorkflowEngineConfig       `json:"workflowEngine,omitempty" yaml:"workflowEngine,omitempty"`

	// Integration settings
	EnableMetrics       bool          `json:"enableMetrics" yaml:"enableMetrics"`
	EnableTracing       bool          `json:"enableTracing" yaml:"enableTracing"`
	HealthCheckInterval time.Duration `json:"healthCheckInterval" yaml:"healthCheckInterval"`
	InitTimeout         time.Duration `json:"initTimeout" yaml:"initTimeout"`
}

// NephioIntegrationStatus represents the status of Nephio integration
type NephioIntegrationStatus struct {
	Ready                bool      `json:"ready"`
	InitializedAt        time.Time `json:"initializedAt"`
	LastHealthCheck      time.Time `json:"lastHealthCheck"`
	WorkflowOrchestrator string    `json:"workflowOrchestrator"`
	PackageCatalog       string    `json:"packageCatalog"`
	WorkloadRegistry     string    `json:"workloadRegistry"`
	ConfigSync           string    `json:"configSync"`
	WorkflowEngine       string    `json:"workflowEngine"`
	RegisteredWorkflows  int       `json:"registeredWorkflows"`
	RegisteredBlueprints int       `json:"registeredBlueprints"`
	RegisteredClusters   int       `json:"registeredClusters"`
}

// Default configuration
var DefaultNephioIntegrationConfig = &NephioIntegrationConfig{
	NephioNamespace:      "nephio-system",
	PorchAPI:             "https://porch.nephio-system.svc.cluster.local:8080",
	ConfigSyncEnabled:    true,
	UpstreamRepository:   "nephoran-blueprints",
	DownstreamRepository: "nephoran-deployments",
	CatalogRepository:    "nephoran-catalog",
	EnableMetrics:        true,
	EnableTracing:        true,
	HealthCheckInterval:  1 * time.Minute,
	InitTimeout:          10 * time.Minute,
}

// NewNephioIntegration creates a new Nephio integration instance
func NewNephioIntegration(
	client client.Client,
	porchClient porch.PorchClient,
	config *NephioIntegrationConfig,
) (*NephioIntegration, error) {
	if config == nil {
		config = DefaultNephioIntegrationConfig
	}

	logger := log.Log.WithName("nephio-integration")
	logger.Info("Initializing Nephio integration",
		"namespace", config.NephioNamespace,
		"porchAPI", config.PorchAPI,
		"configSyncEnabled", config.ConfigSyncEnabled,
	)

	// Initialize Config Sync client
	var configSync *ConfigSyncClient
	if config.ConfigSyncEnabled {
		configSyncConfig := config.ConfigSync
		if configSyncConfig == nil {
			configSyncConfig = &ConfigSyncConfig{
				Repository: config.DownstreamRepository,
				Branch:     "main",
				Directory:  "clusters",
				SyncPeriod: 30 * time.Second,
			}
		}

		var err error
		configSync, err = NewConfigSyncClient(client, configSyncConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize Config Sync client: %w", err)
		}
		logger.Info("Config Sync client initialized successfully")
	}

	// Initialize workload cluster registry
	workloadRegistryConfig := config.WorkloadRegistry
	if workloadRegistryConfig == nil {
		workloadRegistryConfig = DefaultWorkloadClusterConfig
	}

	workloadRegistry, err := NewWorkloadClusterRegistry(client, workloadRegistryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize workload cluster registry: %w", err)
	}
	logger.Info("Workload cluster registry initialized successfully")

	// Initialize package catalog
	packageCatalogConfig := config.PackageCatalog
	if packageCatalogConfig == nil {
		packageCatalogConfig = DefaultPackageCatalogConfig
		packageCatalogConfig.CatalogRepository = config.CatalogRepository
	}

	packageCatalog, err := NewNephioPackageCatalog(client, porchClient, packageCatalogConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize package catalog: %w", err)
	}
	logger.Info("Package catalog initialized successfully")

	// Initialize workflow engine
	workflowEngineConfig := config.WorkflowEngine
	if workflowEngineConfig == nil {
		workflowEngineConfig = DefaultWorkflowEngineConfig
	}

	workflowEngine, err := NewNephioWorkflowEngine(workflowEngineConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize workflow engine: %w", err)
	}
	logger.Info("Workflow engine initialized successfully")

	// Initialize workflow orchestrator
	workflowOrchestratorConfig := config.WorkflowOrchestrator
	if workflowOrchestratorConfig == nil {
		workflowOrchestratorConfig = DefaultWorkflowOrchestratorConfig
		workflowOrchestratorConfig.UpstreamRepository = config.UpstreamRepository
		workflowOrchestratorConfig.DownstreamRepository = config.DownstreamRepository
		workflowOrchestratorConfig.CatalogRepository = config.CatalogRepository
	}

	workflowOrchestrator, err := NewNephioWorkflowOrchestrator(
		client,
		porchClient,
		workflowOrchestratorConfig,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize workflow orchestrator: %w", err)
	}

	// Wire up dependencies
	workflowOrchestrator.configSync = configSync
	workflowOrchestrator.workloadRegistry = workloadRegistry
	workflowOrchestrator.packageCatalog = packageCatalog
	workflowOrchestrator.workflowEngine = workflowEngine

	logger.Info("Workflow orchestrator initialized successfully")

	integration := &NephioIntegration{
		client:               client,
		porchClient:          porchClient,
		workflowOrchestrator: workflowOrchestrator,
		packageCatalog:       packageCatalog,
		workloadRegistry:     workloadRegistry,
		configSync:           configSync,
		workflowEngine:       workflowEngine,
		config:               config,
	}

	logger.Info("Nephio integration initialized successfully")
	return integration, nil
}

// ProcessNetworkIntent processes a NetworkIntent using Nephio-native workflows
func (ni *NephioIntegration) ProcessNetworkIntent(ctx context.Context, intent *v1.NetworkIntent) (*WorkflowExecution, error) {
	logger := log.FromContext(ctx).WithName("nephio-integration").WithValues(
		"intent", intent.Name,
		"namespace", intent.Namespace,
		"type", string(intent.Spec.IntentType),
	)

	logger.Info("Processing NetworkIntent with Nephio workflow")

	// Validate intent for Nephio processing
	if err := ni.validateIntentForNephio(ctx, intent); err != nil {
		return nil, fmt.Errorf("%s: %w", "intent validation failed for Nephio processing", err)
	}

	// Execute Nephio workflow
	execution, err := ni.workflowOrchestrator.ExecuteNephioWorkflow(ctx, intent)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", "failed to execute Nephio workflow", err)
	}

	logger.Info("Nephio workflow execution initiated",
		"executionId", execution.ID,
		"workflow", execution.WorkflowDef.Name,
	)

	return execution, nil
}

// GetWorkflowExecution retrieves a workflow execution by ID
func (ni *NephioIntegration) GetWorkflowExecution(ctx context.Context, executionID string) (*WorkflowExecution, error) {
	if value, exists := ni.workflowOrchestrator.executionCache.Load(executionID); exists {
		if execution, ok := value.(*WorkflowExecution); ok {
			return execution, nil
		}
	}
	return nil, fmt.Errorf("workflow execution not found: %s", executionID)
}

// ListWorkflowExecutions lists all workflow executions
func (ni *NephioIntegration) ListWorkflowExecutions(ctx context.Context) ([]*WorkflowExecution, error) {
	executions := make([]*WorkflowExecution, 0)

	ni.workflowOrchestrator.executionCache.Range(func(key, value interface{}) bool {
		if execution, ok := value.(*WorkflowExecution); ok {
			executions = append(executions, execution)
		}
		return true
	})

	return executions, nil
}

// RegisterWorkloadCluster registers a new workload cluster
func (ni *NephioIntegration) RegisterWorkloadCluster(ctx context.Context, cluster *WorkloadCluster) error {
	return ni.workloadRegistry.RegisterWorkloadCluster(ctx, cluster)
}

// GetWorkloadClusters retrieves all registered workload clusters
func (ni *NephioIntegration) GetWorkloadClusters(ctx context.Context) ([]*WorkloadCluster, error) {
	return ni.workloadRegistry.ListWorkloadClusters(ctx)
}

// GetPackageCatalog provides access to the package catalog
func (ni *NephioIntegration) GetPackageCatalog() *NephioPackageCatalog {
	return ni.packageCatalog
}

// GetWorkflowEngine provides access to the workflow engine
func (ni *NephioIntegration) GetWorkflowEngine() *NephioWorkflowEngine {
	return ni.workflowEngine
}

// GetConfigSyncClient provides access to the Config Sync client
func (ni *NephioIntegration) GetConfigSyncClient() *ConfigSyncClient {
	return ni.configSync
}

// GetStatus returns the current status of Nephio integration
func (ni *NephioIntegration) GetStatus(ctx context.Context) (*NephioIntegrationStatus, error) {
	// Count registered workflows
	workflows, err := ni.workflowEngine.ListWorkflows(ctx)
	if err != nil {
		return nil, err
	}

	// Count registered clusters
	clusters, err := ni.workloadRegistry.ListWorkloadClusters(ctx)
	if err != nil {
		return nil, err
	}

	// Count blueprints (simulated)
	blueprintCount := 0
	ni.packageCatalog.blueprints.Range(func(key, value interface{}) bool {
		blueprintCount++
		return true
	})

	return &NephioIntegrationStatus{
		Ready:                true,
		InitializedAt:        time.Now(), // Would store actual initialization time
		LastHealthCheck:      time.Now(),
		WorkflowOrchestrator: "active",
		PackageCatalog:       "active",
		WorkloadRegistry:     "active",
		ConfigSync:           ni.getConfigSyncStatus(),
		WorkflowEngine:       "active",
		RegisteredWorkflows:  len(workflows),
		RegisteredBlueprints: blueprintCount,
		RegisteredClusters:   len(clusters),
	}, nil
}

// HealthCheck performs a comprehensive health check of Nephio integration
func (ni *NephioIntegration) HealthCheck(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("nephio-health-check")

	// Check workflow orchestrator
	if ni.workflowOrchestrator == nil {
		return fmt.Errorf("workflow orchestrator not initialized")
	}

	// Check package catalog
	if ni.packageCatalog == nil {
		return fmt.Errorf("package catalog not initialized")
	}

	// Check workload registry
	if ni.workloadRegistry == nil {
		return fmt.Errorf("workload registry not initialized")
	}

	// Check workflow engine
	if ni.workflowEngine == nil {
		return fmt.Errorf("workflow engine not initialized")
	}

	// Check Config Sync (if enabled)
	if ni.config.ConfigSyncEnabled && ni.configSync == nil {
		return fmt.Errorf("Config Sync not initialized but required")
	}

	// Verify workflow registrations
	workflows, err := ni.workflowEngine.ListWorkflows(ctx)
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	if len(workflows) == 0 {
		return fmt.Errorf("no workflows registered")
	}

	logger.Info("Nephio integration health check passed",
		"workflows", len(workflows),
		"configSyncEnabled", ni.config.ConfigSyncEnabled,
	)

	return nil
}

// validateIntentForNephio validates that an intent can be processed with Nephio
func (ni *NephioIntegration) validateIntentForNephio(ctx context.Context, intent *v1.NetworkIntent) error {
	// Check if intent type is supported by any registered workflow
	workflows, err := ni.workflowEngine.ListWorkflows(ctx)
	if err != nil {
		return fmt.Errorf("failed to list workflows: %w", err)
	}

	for _, workflow := range workflows {
		for _, intentType := range workflow.IntentTypes {
			if intentType == intent.Spec.IntentType {
				return nil // Found supporting workflow
			}
		}
	}

	return fmt.Errorf("no workflow found for intent type: %s", intent.Spec.IntentType)
}

// getConfigSyncStatus returns the Config Sync status
func (ni *NephioIntegration) getConfigSyncStatus() string {
	if ni.configSync == nil {
		return "disabled"
	}
	return "active"
}

// Cleanup performs cleanup of Nephio integration resources
func (ni *NephioIntegration) Cleanup(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("nephio-cleanup")

	// Stop health monitoring
	if ni.workloadRegistry != nil && ni.workloadRegistry.healthMonitor != nil {
		ni.workloadRegistry.healthMonitor.Stop()
	}

	// Cleanup workflows (if any active executions need to be stopped)
	executions, err := ni.ListWorkflowExecutions(ctx)
	if err == nil {
		for _, execution := range executions {
			if execution.Status == WorkflowExecutionStatusRunning {
				logger.Info("Found running workflow execution during cleanup", "executionId", execution.ID)
				// In a real implementation, would gracefully stop the execution
			}
		}
	}

	logger.Info("Nephio integration cleanup completed")
	return nil
}

// GetWorkflowForIntent finds the appropriate workflow for a given intent
func (ni *NephioIntegration) GetWorkflowForIntent(ctx context.Context, intent *v1.NetworkIntent) (*WorkflowDefinition, error) {
	return ni.workflowOrchestrator.selectWorkflow(ctx, intent)
}

// CreatePackageVariant creates a package variant for a specific cluster
func (ni *NephioIntegration) CreatePackageVariant(ctx context.Context, blueprint *BlueprintPackage, specialization *SpecializationRequest) (*PackageVariant, error) {
	return ni.packageCatalog.CreatePackageVariant(ctx, blueprint, specialization)
}

// FindBlueprint finds a blueprint for a given intent
func (ni *NephioIntegration) FindBlueprint(ctx context.Context, intent *v1.NetworkIntent) (*BlueprintPackage, error) {
	return ni.packageCatalog.FindBlueprintForIntent(ctx, intent)
}

// DeployPackageToCluster deploys a package to a specific cluster via Config Sync
func (ni *NephioIntegration) DeployPackageToCluster(ctx context.Context, pkg *porch.PackageRevision, cluster *WorkloadCluster) (*SyncResult, error) {
	if ni.configSync == nil {
		return nil, fmt.Errorf("Config Sync not enabled")
	}
	return ni.configSync.DeployPackage(ctx, pkg, cluster)
}

// GetClusterHealth checks the health of a specific cluster
func (ni *NephioIntegration) GetClusterHealth(ctx context.Context, clusterName string) (*ClusterHealth, error) {
	return ni.workloadRegistry.CheckClusterHealth(ctx, clusterName)
}

// RegisterCustomWorkflow allows registration of custom workflow definitions
func (ni *NephioIntegration) RegisterCustomWorkflow(workflow *WorkflowDefinition) error {
	return ni.workflowEngine.RegisterWorkflow(workflow)
}

// GetIntegrationMetrics returns integration metrics (placeholder for metrics collection)
func (ni *NephioIntegration) GetIntegrationMetrics(ctx context.Context) (map[string]interface{}, error) {
	metrics := make(map[string]interface{})

	// Workflow metrics
	workflows, _ := ni.workflowEngine.ListWorkflows(ctx)
	metrics["registered_workflows"] = len(workflows)

	// Cluster metrics
	clusters, _ := ni.workloadRegistry.ListWorkloadClusters(ctx)
	metrics["registered_clusters"] = len(clusters)

	// Execution metrics
	executions, _ := ni.ListWorkflowExecutions(ctx)
	metrics["total_executions"] = len(executions)

	runningExecutions := 0
	for _, execution := range executions {
		if execution.Status == WorkflowExecutionStatusRunning {
			runningExecutions++
		}
	}
	metrics["running_executions"] = runningExecutions

	// Blueprint metrics
	blueprintCount := 0
	ni.packageCatalog.blueprints.Range(func(key, value interface{}) bool {
		blueprintCount++
		return true
	})
	metrics["available_blueprints"] = blueprintCount

	return metrics, nil
}

// ValidateWorkflow validates a workflow definition
func (ni *NephioIntegration) ValidateWorkflow(ctx context.Context, workflow *WorkflowDefinition) (*ValidationResult, error) {
	return ni.workflowEngine.validator.ValidateWorkflow(ctx, workflow), nil
}
