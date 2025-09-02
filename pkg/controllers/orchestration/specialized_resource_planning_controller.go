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

package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
)

// SpecializedResourcePlanningController handles resource planning and optimization for telecom workloads.

type SpecializedResourcePlanningController struct {
	client.Client

	Scheme *runtime.Scheme

	Recorder record.EventRecorder

	Logger logr.Logger

	// Resource planning services.

	ResourceCalculator *TelecomResourceCalculator

	OptimizationEngine *ResourceOptimizationEngine

	ConstraintSolver *ResourceConstraintSolver

	CostEstimator *TelecomCostEstimator

	// Configuration.

	Config ResourcePlanningConfig

	ResourceTemplates map[string]*NetworkFunctionTemplate

	ConstraintRules []*ResourceConstraintRule

	// Internal state.

	activePlanning sync.Map // map[string]*PlanningSession

	planningCache *ResourcePlanCache

	metrics *ResourcePlanningMetrics

	// Health and lifecycle.

	started bool

	stopChan chan struct{}

	healthStatus interfaces.HealthStatus

	mutex sync.RWMutex
}

// Note: ResourcePlanningConfig is defined in resource_planning_controller.go to avoid duplication.

// TelecomResourceCalculator calculates resource requirements for telecom NFs.

type TelecomResourceCalculator struct {
	logger logr.Logger

	baselineProfiles map[string]*ResourceProfile

	scalingFactors map[string]*ScalingProfile

	performanceTargets map[string]*PerformanceTarget
}

// ResourceProfile defines baseline resource requirements.

type ResourceProfile struct {
	NetworkFunction string `json:"networkFunction"`

	BaselineCPU resource.Quantity `json:"baselineCpu"`

	BaselineMemory resource.Quantity `json:"baselineMemory"`

	BaselineStorage resource.Quantity `json:"baselineStorage"`

	NetworkBandwidth string `json:"networkBandwidth"`

	IOPSRequirement int64 `json:"iopsRequirement"`

	Metadata json.RawMessage `json:"metadata"`
}

// ScalingProfile defines how resources scale with load.

type ScalingProfile struct {
	NetworkFunction string `json:"networkFunction"`

	CPUScalingFactor float64 `json:"cpuScalingFactor"`

	MemoryScalingFactor float64 `json:"memoryScalingFactor"`

	StorageScalingFactor float64 `json:"storageScalingFactor"`

	MaxReplicas int32 `json:"maxReplicas"`

	MinReplicas int32 `json:"minReplicas"`

	ScalingThreshold float64 `json:"scalingThreshold"`
}

// PerformanceTarget defines performance requirements.

type PerformanceTarget struct {
	NetworkFunction string `json:"networkFunction"`

	MaxLatency time.Duration `json:"maxLatency"`

	MinThroughput float64 `json:"minThroughput"`

	MaxErrorRate float64 `json:"maxErrorRate"`

	AvailabilityTarget float64 `json:"availabilityTarget"`
}

// NetworkFunctionTemplate defines templates for common NF deployments.

type NetworkFunctionTemplate struct {
	Name string `json:"name"`

	Type string `json:"type"`

	Category string `json:"category"` // 5GC, RAN, Edge

	DefaultResources *interfaces.ResourceSpec `json:"defaultResources"`

	Dependencies []string `json:"dependencies"`

	AntiAffinity []string `json:"antiAffinity"`

	PreferredNodes []string `json:"preferredNodes"`

	Metadata json.RawMessage `json:"metadata"`
}

// ResourceConstraintRule defines resource constraints.

type ResourceConstraintRule struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Type string `json:"type"` // hard, soft

	Scope string `json:"scope"` // global, per-nf, per-cluster

	Rule ResourceConstraintCheck `json:"rule"`

	ViolationAction string `json:"violationAction"` // reject, warn, optimize

	Metadata json.RawMessage `json:"metadata"`
}

// ResourceConstraintCheck defines constraint checking logic.

type ResourceConstraintCheck func(plan *interfaces.ResourcePlan) error

// PlanningSession tracks active resource planning.

type PlanningSession struct {
	IntentID string `json:"intentId"`

	CorrelationID string `json:"correlationId"`

	StartTime time.Time `json:"startTime"`

	Status string `json:"status"`

	Progress float64 `json:"progress"`

	CurrentStep string `json:"currentStep"`

	// Input data.

	LLMResponse json.RawMessage `json:"llmResponse"`

	RequiredNFs []string `json:"requiredNFs"`

	DeploymentType string `json:"deploymentType"`

	// Planning results.

	ResourcePlan *interfaces.ResourcePlan `json:"resourcePlan"`

	OptimizedPlan *interfaces.OptimizedPlan `json:"optimizedPlan"`

	CostEstimate *interfaces.CostEstimate `json:"costEstimate"`

	// Error handling.

	Errors []string `json:"errors"`

	Warnings []string `json:"warnings"`

	LastError string `json:"lastError"`

	// Performance tracking.

	Metrics PlanningSessionMetrics `json:"metrics"`

	mutex sync.RWMutex
}

// PlanningSessionMetrics tracks metrics for planning session.

type PlanningSessionMetrics struct {
	CalculationTime time.Duration `json:"calculationTime"`

	OptimizationTime time.Duration `json:"optimizationTime"`

	ConstraintCheckTime time.Duration `json:"constraintCheckTime"`

	TotalTime time.Duration `json:"totalTime"`

	NFsPlanned int `json:"nfsPlanned"`

	ConstraintsChecked int `json:"constraintsChecked"`

	OptimizationsApplied int `json:"optimizationsApplied"`

	CacheHit bool `json:"cacheHit"`
}

// ResourcePlanningMetrics tracks overall controller metrics.

type ResourcePlanningMetrics struct {
	TotalPlanned int64 `json:"totalPlanned"`

	SuccessfulPlanned int64 `json:"successfulPlanned"`

	FailedPlanned int64 `json:"failedPlanned"`

	AveragePlanningTime time.Duration `json:"averagePlanningTime"`

	AverageCostSavings float64 `json:"averageCostSavings"`

	// Per NF type metrics.

	NFTypeMetrics map[string]*NFPlanningMetrics `json:"nfTypeMetrics"`

	// Resource utilization.

	TotalCPUPlanned resource.Quantity `json:"totalCpuPlanned"`

	TotalMemoryPlanned resource.Quantity `json:"totalMemoryPlanned"`

	TotalStoragePlanned resource.Quantity `json:"totalStoragePlanned"`

	// Cache metrics.

	CacheHitRate float64 `json:"cacheHitRate"`

	ConstraintViolations int64 `json:"constraintViolations"`

	LastUpdated time.Time `json:"lastUpdated"`

	mutex sync.RWMutex
}

// NFPlanningMetrics tracks metrics per NF type.

type NFPlanningMetrics struct {
	Count int64 `json:"count"`

	AverageCPU resource.Quantity `json:"averageCpu"`

	AverageMemory resource.Quantity `json:"averageMemory"`

	AverageStorage resource.Quantity `json:"averageStorage"`

	SuccessRate float64 `json:"successRate"`

	AveragePlanningTime time.Duration `json:"averagePlanningTime"`
}

// ResourcePlanCache provides caching for resource plans.

type ResourcePlanCache struct {
	entries map[string]*PlanCacheEntry

	mutex sync.RWMutex

	ttl time.Duration

	maxEntries int
}

// PlanCacheEntry represents a cached resource plan.

type PlanCacheEntry struct {
	Plan *interfaces.ResourcePlan `json:"plan"`

	Timestamp time.Time `json:"timestamp"`

	HitCount int64 `json:"hitCount"`

	PlanHash string `json:"planHash"`
}

// NewSpecializedResourcePlanningController creates a new resource planning controller.

func NewSpecializedResourcePlanningController(mgr ctrl.Manager, config ResourcePlanningConfig) (*SpecializedResourcePlanningController, error) {
	logger := log.FromContext(context.Background()).WithName("specialized-resource-planner")

	// Initialize resource calculator.

	resourceCalculator := NewTelecomResourceCalculator(logger)

	// Initialize optimization engine.

	optimizationEngine := NewResourceOptimizationEngine(logger, config)

	// Initialize constraint solver.

	constraintSolver := NewResourceConstraintSolver(logger)

	// Initialize cost estimator.

	costEstimator := NewTelecomCostEstimator(logger)

	controller := &SpecializedResourcePlanningController{
		Client: mgr.GetClient(),

		Scheme: mgr.GetScheme(),

		Recorder: mgr.GetEventRecorderFor("specialized-resource-planner"),

		Logger: logger,

		ResourceCalculator: resourceCalculator,

		OptimizationEngine: optimizationEngine,

		ConstraintSolver: constraintSolver,

		CostEstimator: costEstimator,

		Config: config,

		ResourceTemplates: initializeResourceTemplates(),

		ConstraintRules: initializeConstraintRules(),

		metrics: NewResourcePlanningMetrics(),

		stopChan: make(chan struct{}),

		healthStatus: interfaces.HealthStatus{
			Status: "Healthy",

			Message: "Resource planning controller initialized",

			LastChecked: time.Now(),
		},
	}

	// Initialize cache if enabled.

	if config.CacheEnabled {
		controller.planningCache = &ResourcePlanCache{
			entries: make(map[string]*PlanCacheEntry),

			ttl: config.CacheTTL,

			maxEntries: config.MaxCacheEntries,
		}
	}

	return controller, nil
}

// ProcessPhase implements the PhaseController interface.

func (c *SpecializedResourcePlanningController) ProcessPhase(ctx context.Context, intent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase) (interfaces.ProcessingResult, error) {
	if phase != interfaces.PhaseResourcePlanning {
		return interfaces.ProcessingResult{
			Success: false,

			ErrorMessage: fmt.Sprintf("unsupported phase: %s", phase),
		}, nil
	}

	// Extract LLM response from intent status.

	var llmResponse map[string]interface{}

	if len(intent.Status.LLMResponse.Raw) > 0 {
		if err := json.Unmarshal(intent.Status.LLMResponse.Raw, &llmResponse); err != nil {
			return interfaces.ProcessingResult{
				Success: false,

				ErrorMessage: fmt.Sprintf("failed to unmarshal LLM response: %v", err),

				ErrorCode: "INVALID_INPUT",
			}, nil
		}
	} else {
		return interfaces.ProcessingResult{
			Success: false,

			ErrorMessage: "invalid or missing LLM response data",

			ErrorCode: "INVALID_INPUT",
		}, nil
	}

	return c.planResourcesFromLLM(ctx, intent, llmResponse)
}

// PlanResources implements the ResourcePlanner interface.

func (c *SpecializedResourcePlanningController) PlanResources(ctx context.Context, llmResponse map[string]interface{}) (*interfaces.ResourcePlan, error) {
	result, err := c.planResourcesFromLLM(ctx, nil, llmResponse)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("resource planning failed: %s", result.ErrorMessage)
	}

	if plan, ok := result.Data["resourcePlan"].(*interfaces.ResourcePlan); ok {
		return plan, nil
	}

	return nil, fmt.Errorf("invalid resource plan in result")
}

// planResourcesFromLLM performs resource planning based on LLM response.

func (c *SpecializedResourcePlanningController) planResourcesFromLLM(ctx context.Context, intent *nephoranv1.NetworkIntent, llmResponse map[string]interface{}) (interfaces.ProcessingResult, error) {
	startTime := time.Now()

	intentID := "unknown"

	if intent != nil {
		intentID = intent.Name
	}

	c.Logger.Info("Planning resources from LLM response", "intentId", intentID)

	// Create planning session.

	session := &PlanningSession{
		IntentID: intentID,

		CorrelationID: fmt.Sprintf("plan-%s-%d", intentID, startTime.Unix()),

		StartTime: startTime,

		Status: "planning",

		Progress: 0.0,

		CurrentStep: "initialization",

		LLMResponse: llmResponse,
	}

	// Store session for tracking.

	c.activePlanning.Store(intentID, session)

	defer c.activePlanning.Delete(intentID)

	// Check cache first.

	if c.planningCache != nil {
		if cached := c.getCachedPlan(llmResponse); cached != nil {

			c.Logger.Info("Cache hit for resource plan", "intentId", intentID)

			session.Metrics.CacheHit = true

			c.updateMetrics(session, true, time.Since(startTime))

			return interfaces.ProcessingResult{
				Success: true,

				NextPhase: interfaces.PhaseManifestGeneration,

				Data: json.RawMessage(`{}`),

				Metrics: map[string]float64{
					"planning_time_ms": float64(time.Since(startTime).Milliseconds()),

					"cache_hit": 1,
				},
			}, nil

		}
	}

	// Parse network functions from LLM response.

	session.updateProgress(0.2, "parsing_network_functions")

	networkFunctions, err := c.parseNetworkFunctions(llmResponse)
	if err != nil {

		session.addError(fmt.Sprintf("failed to parse network functions: %v", err))

		c.updateMetrics(session, false, time.Since(startTime))

		return interfaces.ProcessingResult{
			Success: false,

			ErrorMessage: err.Error(),

			ErrorCode: "PARSING_ERROR",
		}, nil

	}

	session.RequiredNFs = c.extractNFNames(networkFunctions)

	// Calculate resource requirements.

	session.updateProgress(0.4, "calculating_resources")

	calcStartTime := time.Now()

	resourcePlan, err := c.calculateResourceRequirements(ctx, networkFunctions, llmResponse)
	if err != nil {

		session.addError(fmt.Sprintf("resource calculation failed: %v", err))

		c.updateMetrics(session, false, time.Since(startTime))

		return interfaces.ProcessingResult{
			Success: false,

			ErrorMessage: err.Error(),

			ErrorCode: "CALCULATION_ERROR",
		}, nil

	}

	session.Metrics.CalculationTime = time.Since(calcStartTime)

	session.ResourcePlan = resourcePlan

	// Validate constraints.

	session.updateProgress(0.6, "validating_constraints")

	constraintStartTime := time.Now()

	if err := c.ValidateResourceConstraints(ctx, resourcePlan); err != nil {
		session.addWarning(fmt.Sprintf("constraint validation warning: %v", err))

		// Continue with warnings, don't fail.
	}

	session.Metrics.ConstraintCheckTime = time.Since(constraintStartTime)

	// Optimize resource allocation if enabled.

	var optimizedPlan *interfaces.OptimizedPlan

	if c.Config.OptimizationEnabled {

		session.updateProgress(0.8, "optimizing_resources")

		optStartTime := time.Now()

		requirements := &interfaces.ResourceRequirements{
			CPU: c.Config.DefaultCPURequest,

			Memory: c.Config.DefaultMemoryRequest,

			Storage: c.Config.DefaultStorageRequest,
		}

		optimizedPlan, err = c.OptimizeResourceAllocation(ctx, requirements)

		if err != nil {
			session.addWarning(fmt.Sprintf("optimization failed: %v", err))

			// Continue without optimization.
		} else {

			session.OptimizedPlan = optimizedPlan

			// Use optimized plan if available.

			if optimizedPlan != nil && optimizedPlan.OptimizedPlan != nil {
				resourcePlan = optimizedPlan.OptimizedPlan
			}

		}

		session.Metrics.OptimizationTime = time.Since(optStartTime)

	}

	// Estimate costs.

	session.updateProgress(0.9, "estimating_costs")

	costEstimate, err := c.EstimateResourceCosts(ctx, resourcePlan)

	if err != nil {
		session.addWarning(fmt.Sprintf("cost estimation failed: %v", err))

		// Continue without cost estimate.
	} else {
		session.CostEstimate = costEstimate
	}

	// Finalize.

	session.updateProgress(1.0, "completed")

	session.Metrics.TotalTime = time.Since(startTime)

	// Create result.

	resultData := json.RawMessage(`{}`)

	if optimizedPlan != nil {
		resultData["optimizedPlan"] = optimizedPlan
	}

	if costEstimate != nil {
		resultData["costEstimate"] = costEstimate
	}

	result := interfaces.ProcessingResult{
		Success: true,

		NextPhase: interfaces.PhaseManifestGeneration,

		Data: resultData,

		Metrics: map[string]float64{
			"planning_time_ms": float64(session.Metrics.TotalTime.Milliseconds()),

			"calculation_time_ms": float64(session.Metrics.CalculationTime.Milliseconds()),

			"optimization_time_ms": float64(session.Metrics.OptimizationTime.Milliseconds()),

			"constraint_check_time_ms": float64(session.Metrics.ConstraintCheckTime.Milliseconds()),

			"nfs_planned": float64(len(networkFunctions)),
		},

		Events: []interfaces.ProcessingEvent{
			{
				Timestamp: time.Now(),

				EventType: "ResourcesPlanned",

				Message: fmt.Sprintf("Planned resources for %d network functions", len(networkFunctions)),

				CorrelationID: session.CorrelationID,

				Data: json.RawMessage(`{}`),
			},
		},
	}

	// Cache result if enabled.

	if c.planningCache != nil {
		c.cachePlan(llmResponse, resourcePlan)
	}

	// Update metrics.

	c.updateMetrics(session, true, time.Since(startTime))

	c.Logger.Info("Resource planning completed successfully",

		"intentId", intentID,

		"networkFunctions", len(networkFunctions),

		"duration", time.Since(startTime))

	return result, nil
}

// parseNetworkFunctions parses network functions from LLM response.

func (c *SpecializedResourcePlanningController) parseNetworkFunctions(llmResponse map[string]interface{}) ([]interfaces.PlannedNetworkFunction, error) {
	var networkFunctions []interfaces.PlannedNetworkFunction

	// Try to extract network functions from various possible fields.

	nfFields := []string{"network_functions", "networkFunctions", "nfs", "components", "services"}

	var nfData interface{}

	var found bool

	for _, field := range nfFields {
		if data, exists := llmResponse[field]; exists {

			nfData = data

			found = true

			break

		}
	}

	if !found {
		return nil, fmt.Errorf("no network functions found in LLM response")
	}

	// Parse based on data type.

	switch nfs := nfData.(type) {

	case []interface{}:

		for i, nfInterface := range nfs {

			nf, err := c.parseNetworkFunction(nfInterface, i)
			if err != nil {

				c.Logger.Info("Failed to parse network function", "index", i, "error", err)

				continue

			}

			networkFunctions = append(networkFunctions, *nf)

		}

	case map[string]interface{}:

		// Single network function.

		nf, err := c.parseNetworkFunction(nfs, 0)
		if err != nil {
			return nil, fmt.Errorf("failed to parse network function: %w", err)
		}

		networkFunctions = append(networkFunctions, *nf)

	default:

		return nil, fmt.Errorf("unexpected network functions data type: %T", nfData)

	}

	if len(networkFunctions) == 0 {
		return nil, fmt.Errorf("no valid network functions parsed")
	}

	return networkFunctions, nil
}

// parseNetworkFunction parses a single network function from data.

func (c *SpecializedResourcePlanningController) parseNetworkFunction(data interface{}, index int) (*interfaces.PlannedNetworkFunction, error) {
	nfMap, ok := data.(map[string]interface{})

	if !ok {
		return nil, fmt.Errorf("network function data must be an object")
	}

	// Extract basic information.

	name, _ := nfMap["name"].(string)

	if name == "" {
		name = fmt.Sprintf("nf-%d", index)
	}

	nfType, _ := nfMap["type"].(string)

	if nfType == "" {
		nfType = "generic"
	}

	image, _ := nfMap["image"].(string)

	version, _ := nfMap["version"].(string)

	// Parse replicas.

	replicas := int32(1)

	if replicasInterface, exists := nfMap["replicas"]; exists {
		switch r := replicasInterface.(type) {

		case float64:

			replicas = int32(r)

		case int:

			replicas = int32(r)

		case string:

			if parsed, err := strconv.ParseInt(r, 10, 32); err == nil {
				replicas = int32(parsed)
			}

		}
	}

	// Parse resources.

	resources := c.parseResourceSpec(nfMap)

	// Parse ports.

	ports := c.parsePorts(nfMap)

	// Parse environment variables.

	environment := c.parseEnvironment(nfMap)

	// Parse configuration.

	configuration := make(map[string]interface{})

	if configInterface, exists := nfMap["configuration"]; exists {
		if configMap, ok := configInterface.(map[string]interface{}); ok {
			configuration = configMap
		}
	}

	nf := &interfaces.PlannedNetworkFunction{
		Name: name,

		Type: nfType,

		Image: image,

		Version: version,

		Replicas: replicas,

		Resources: resources,

		Ports: ports,

		Environment: environment,

		Configuration: configuration,
	}

	return nf, nil
}

// parseResourceSpec parses resource specifications.

func (c *SpecializedResourcePlanningController) parseResourceSpec(nfMap map[string]interface{}) interfaces.ResourceSpec {
	resourceSpec := interfaces.ResourceSpec{
		Requests: interfaces.ResourceRequirements{
			CPU: c.Config.DefaultCPURequest,

			Memory: c.Config.DefaultMemoryRequest,

			Storage: c.Config.DefaultStorageRequest,
		},

		Limits: interfaces.ResourceRequirements{
			CPU: "",

			Memory: "",

			Storage: "",
		},
	}

	// Parse resources from various possible locations.

	if resourcesInterface, exists := nfMap["resources"]; exists {
		if resourcesMap, ok := resourcesInterface.(map[string]interface{}); ok {

			// Parse requests.

			if requestsInterface, exists := resourcesMap["requests"]; exists {
				if requestsMap, ok := requestsInterface.(map[string]interface{}); ok {

					if cpu, exists := requestsMap["cpu"]; exists {
						if cpuStr, ok := cpu.(string); ok {
							resourceSpec.Requests.CPU = cpuStr
						}
					}

					if memory, exists := requestsMap["memory"]; exists {
						if memoryStr, ok := memory.(string); ok {
							resourceSpec.Requests.Memory = memoryStr
						}
					}

					if storage, exists := requestsMap["storage"]; exists {
						if storageStr, ok := storage.(string); ok {
							resourceSpec.Requests.Storage = storageStr
						}
					}

				}
			}

			// Parse limits.

			if limitsInterface, exists := resourcesMap["limits"]; exists {
				if limitsMap, ok := limitsInterface.(map[string]interface{}); ok {

					if cpu, exists := limitsMap["cpu"]; exists {
						if cpuStr, ok := cpu.(string); ok {
							resourceSpec.Limits.CPU = cpuStr
						}
					}

					if memory, exists := limitsMap["memory"]; exists {
						if memoryStr, ok := memory.(string); ok {
							resourceSpec.Limits.Memory = memoryStr
						}
					}

					if storage, exists := limitsMap["storage"]; exists {
						if storageStr, ok := storage.(string); ok {
							resourceSpec.Limits.Storage = storageStr
						}
					}

				}
			}

		}
	}

	return resourceSpec
}

// parsePorts parses port specifications.

func (c *SpecializedResourcePlanningController) parsePorts(nfMap map[string]interface{}) []interfaces.PortSpec {
	var ports []interfaces.PortSpec

	if portsInterface, exists := nfMap["ports"]; exists {
		if portsArray, ok := portsInterface.([]interface{}); ok {
			for _, portInterface := range portsArray {
				if portMap, ok := portInterface.(map[string]interface{}); ok {

					port := interfaces.PortSpec{}

					if name, exists := portMap["name"]; exists {
						port.Name, _ = name.(string)
					}

					if portNum, exists := portMap["port"]; exists {
						if portFloat, ok := portNum.(float64); ok {
							port.Port = int32(portFloat)
						}
					}

					if targetPort, exists := portMap["targetPort"]; exists {
						if targetFloat, ok := targetPort.(float64); ok {
							port.TargetPort = int32(targetFloat)
						}
					}

					if protocol, exists := portMap["protocol"]; exists {
						port.Protocol, _ = protocol.(string)
					}

					if serviceType, exists := portMap["serviceType"]; exists {
						port.ServiceType, _ = serviceType.(string)
					}

					ports = append(ports, port)

				}
			}
		}
	}

	return ports
}

// parseEnvironment parses environment variables.

func (c *SpecializedResourcePlanningController) parseEnvironment(nfMap map[string]interface{}) []interfaces.EnvVar {
	var environment []interfaces.EnvVar

	if envInterface, exists := nfMap["environment"]; exists {
		switch env := envInterface.(type) {

		case []interface{}:

			for _, envVarInterface := range env {
				if envVarMap, ok := envVarInterface.(map[string]interface{}); ok {

					envVar := interfaces.EnvVar{}

					if name, exists := envVarMap["name"]; exists {
						envVar.Name, _ = name.(string)
					}

					if value, exists := envVarMap["value"]; exists {
						envVar.Value, _ = value.(string)
					}

					environment = append(environment, envVar)

				}
			}

		case map[string]interface{}:

			for key, value := range env {
				if valueStr, ok := value.(string); ok {
					environment = append(environment, interfaces.EnvVar{
						Name: key,

						Value: valueStr,
					})
				}
			}

		}
	}

	return environment
}

// calculateResourceRequirements calculates total resource requirements.

func (c *SpecializedResourcePlanningController) calculateResourceRequirements(ctx context.Context, networkFunctions []interfaces.PlannedNetworkFunction, llmResponse map[string]interface{}) (*interfaces.ResourcePlan, error) {
	// Extract deployment pattern.

	deploymentPattern, _ := llmResponse["deployment_pattern"].(string)

	if deploymentPattern == "" {
		deploymentPattern = "standard"
	}

	// Calculate total resource requirements.

	totalCPU := resource.NewQuantity(0, resource.DecimalSI)

	totalMemory := resource.NewQuantity(0, resource.BinarySI)

	totalStorage := resource.NewQuantity(0, resource.BinarySI)

	// Calculate resources for each NF.

	for i := range networkFunctions {

		nf := &networkFunctions[i]

		// Apply resource templates if available.

		if template, exists := c.ResourceTemplates[nf.Type]; exists {
			c.applyResourceTemplate(nf, template)
		}

		// Calculate per-replica resources.

		cpuRequest, err := resource.ParseQuantity(nf.Resources.Requests.CPU)
		if err != nil {
			cpuRequest, _ = resource.ParseQuantity(c.Config.DefaultCPURequest)
		}

		memoryRequest, err := resource.ParseQuantity(nf.Resources.Requests.Memory)
		if err != nil {
			memoryRequest, _ = resource.ParseQuantity(c.Config.DefaultMemoryRequest)
		}

		storageRequest, err := resource.ParseQuantity(nf.Resources.Requests.Storage)
		if err != nil {
			storageRequest, _ = resource.ParseQuantity(c.Config.DefaultStorageRequest)
		}

		// Multiply by replicas.

		replicaMultiplier := int64(nf.Replicas)

		cpuRequest.Set(cpuRequest.Value() * replicaMultiplier)

		memoryRequest.Set(memoryRequest.Value() * replicaMultiplier)

		storageRequest.Set(storageRequest.Value() * replicaMultiplier)

		// Add to totals.

		totalCPU.Add(cpuRequest)

		totalMemory.Add(memoryRequest)

		totalStorage.Add(storageRequest)

	}

	// Calculate constraints.

	constraints := c.calculateConstraints(networkFunctions, deploymentPattern)

	// Calculate dependencies.

	dependencies := c.calculateDependencies(networkFunctions)

	resourcePlan := &interfaces.ResourcePlan{
		NetworkFunctions: networkFunctions,

		ResourceRequirements: interfaces.ResourceRequirements{
			CPU: totalCPU.String(),

			Memory: totalMemory.String(),

			Storage: totalStorage.String(),
		},

		DeploymentPattern: deploymentPattern,

		Constraints: constraints,

		Dependencies: dependencies,
	}

	return resourcePlan, nil
}

// applyResourceTemplate applies resource template to network function.

func (c *SpecializedResourcePlanningController) applyResourceTemplate(nf *interfaces.PlannedNetworkFunction, template *NetworkFunctionTemplate) {
	// Apply default resources if not specified.

	if nf.Resources.Requests.CPU == "" || nf.Resources.Requests.CPU == c.Config.DefaultCPURequest {
		nf.Resources.Requests.CPU = template.DefaultResources.Requests.CPU
	}

	if nf.Resources.Requests.Memory == "" || nf.Resources.Requests.Memory == c.Config.DefaultMemoryRequest {
		nf.Resources.Requests.Memory = template.DefaultResources.Requests.Memory
	}

	if nf.Resources.Requests.Storage == "" || nf.Resources.Requests.Storage == c.Config.DefaultStorageRequest {
		nf.Resources.Requests.Storage = template.DefaultResources.Requests.Storage
	}

	// Apply limits if not specified.

	if nf.Resources.Limits.CPU == "" {
		nf.Resources.Limits.CPU = template.DefaultResources.Limits.CPU
	}

	if nf.Resources.Limits.Memory == "" {
		nf.Resources.Limits.Memory = template.DefaultResources.Limits.Memory
	}

	if nf.Resources.Limits.Storage == "" {
		nf.Resources.Limits.Storage = template.DefaultResources.Limits.Storage
	}

	// Apply metadata.

	if nf.Configuration == nil {
		nf.Configuration = make(map[string]interface{})
	}

	for key, value := range template.Metadata {
		if _, exists := nf.Configuration[key]; !exists {
			nf.Configuration[key] = value
		}
	}
}

// calculateConstraints calculates resource constraints.

func (c *SpecializedResourcePlanningController) calculateConstraints(networkFunctions []interfaces.PlannedNetworkFunction, deploymentPattern string) []interfaces.ResourceConstraint {
	var constraints []interfaces.ResourceConstraint

	// Anti-affinity constraints for critical NFs.

	criticalNFs := []string{"amf", "smf", "upf", "ausf", "nrf"}

	for _, nf := range networkFunctions {
		for _, criticalType := range criticalNFs {
			if strings.Contains(strings.ToLower(nf.Type), criticalType) {

				constraints = append(constraints, interfaces.ResourceConstraint{
					Type: "anti-affinity",

					Resource: nf.Name,

					Operator: "NotIn",

					Value: []string{nf.Type},

					Message: fmt.Sprintf("Ensure %s instances are distributed across nodes", nf.Type),
				})

				break

			}
		}
	}

	// Resource limit constraints.

	constraints = append(constraints, interfaces.ResourceConstraint{
		Type: "resource-limit",

		Resource: "cpu",

		Operator: "LessThan",

		Value: "80%", // Don't use more than 80% of node CPU

		Message: "Ensure sufficient CPU headroom for system processes",
	})

	constraints = append(constraints, interfaces.ResourceConstraint{
		Type: "resource-limit",

		Resource: "memory",

		Operator: "LessThan",

		Value: "85%", // Don't use more than 85% of node memory

		Message: "Ensure sufficient memory headroom for system processes",
	})

	// Deployment pattern specific constraints.

	switch deploymentPattern {

	case "high-availability":

		constraints = append(constraints, interfaces.ResourceConstraint{
			Type: "replica-minimum",

			Resource: "all",

			Operator: "GreaterThanOrEqual",

			Value: 3,

			Message: "High availability requires minimum 3 replicas",
		})

	case "edge":

		constraints = append(constraints, interfaces.ResourceConstraint{
			Type: "resource-limit",

			Resource: "cpu",

			Operator: "LessThan",

			Value: "4",

			Message: "Edge deployments have limited CPU resources",
		})

	}

	return constraints
}

// calculateDependencies calculates NF dependencies.

func (c *SpecializedResourcePlanningController) calculateDependencies(networkFunctions []interfaces.PlannedNetworkFunction) []interfaces.Dependency {
	var dependencies []interfaces.Dependency

	// Define common 5G dependencies.

	dependencyMap := map[string][]string{
		"amf": {"nrf", "ausf", "udm"},

		"smf": {"nrf", "udm", "upf"},

		"upf": {"pfcp"},

		"nssf": {"nrf"},

		"ausf": {"nrf", "udm"},

		"udm": {"nrf"},

		"udr": {"nrf"},

		"pcf": {"nrf"},

		"nef": {"nrf"},
	}

	// Create dependency map for existing NFs.

	existingNFs := make(map[string]bool)

	for _, nf := range networkFunctions {
		existingNFs[strings.ToLower(nf.Type)] = true
	}

	// Calculate dependencies.

	for _, nf := range networkFunctions {

		nfType := strings.ToLower(nf.Type)

		if deps, exists := dependencyMap[nfType]; exists {
			for _, dep := range deps {
				if existingNFs[dep] {
					dependencies = append(dependencies, interfaces.Dependency{
						Name: fmt.Sprintf("%s-depends-on-%s", nf.Name, dep),

						Type: "service",

						Required: true,

						Metadata: map[string]string{
							"source": nf.Name,

							"target": dep,

							"type": "5g-core",
						},
					})
				}
			}
		}

	}

	return dependencies
}

// extractNFNames extracts network function names.

func (c *SpecializedResourcePlanningController) extractNFNames(networkFunctions []interfaces.PlannedNetworkFunction) []string {
	names := make([]string, len(networkFunctions))

	for i, nf := range networkFunctions {
		names[i] = nf.Name
	}

	return names
}

// Helper methods for session management.

// updateProgress updates session progress.

func (s *PlanningSession) updateProgress(progress float64, step string) {
	s.mutex.Lock()

	defer s.mutex.Unlock()

	s.Progress = progress

	s.CurrentStep = step
}

// addError adds error to session.

func (s *PlanningSession) addError(errorMsg string) {
	s.mutex.Lock()

	defer s.mutex.Unlock()

	s.Errors = append(s.Errors, errorMsg)

	s.LastError = errorMsg
}

// addWarning adds warning to session.

func (s *PlanningSession) addWarning(warningMsg string) {
	s.mutex.Lock()

	defer s.mutex.Unlock()

	s.Warnings = append(s.Warnings, warningMsg)
}

// Cache management methods.

// getCachedPlan retrieves cached plan.

func (c *SpecializedResourcePlanningController) getCachedPlan(llmResponse map[string]interface{}) *PlanCacheEntry {
	if c.planningCache == nil {
		return nil
	}

	c.planningCache.mutex.RLock()

	defer c.planningCache.mutex.RUnlock()

	planHash := c.hashLLMResponse(llmResponse)

	if entry, exists := c.planningCache.entries[planHash]; exists {

		// Check if entry is still valid.

		if time.Since(entry.Timestamp) < c.planningCache.ttl {

			entry.HitCount++

			return entry

		}

		// Entry expired, remove it.

		delete(c.planningCache.entries, planHash)

	}

	return nil
}

// cachePlan stores plan in cache.

func (c *SpecializedResourcePlanningController) cachePlan(llmResponse map[string]interface{}, plan *interfaces.ResourcePlan) {
	if c.planningCache == nil {
		return
	}

	c.planningCache.mutex.Lock()

	defer c.planningCache.mutex.Unlock()

	planHash := c.hashLLMResponse(llmResponse)

	// Check cache size limit.

	if len(c.planningCache.entries) >= c.planningCache.maxEntries {

		// Remove oldest entry (simple LRU).

		var oldestKey string

		var oldestTime time.Time

		for key, entry := range c.planningCache.entries {
			if oldestKey == "" || entry.Timestamp.Before(oldestTime) {

				oldestKey = key

				oldestTime = entry.Timestamp

			}
		}

		if oldestKey != "" {
			delete(c.planningCache.entries, oldestKey)
		}

	}

	c.planningCache.entries[planHash] = &PlanCacheEntry{
		Plan: plan,

		Timestamp: time.Now(),

		HitCount: 0,

		PlanHash: planHash,
	}
}

// hashLLMResponse creates a hash for LLM response.

func (c *SpecializedResourcePlanningController) hashLLMResponse(llmResponse map[string]interface{}) string {
	// Simple hash based on key characteristics - in production, use proper hashing.

	var hashComponents []string

	if nfs, exists := llmResponse["network_functions"]; exists {
		hashComponents = append(hashComponents, fmt.Sprintf("nfs:%v", nfs))
	}

	if pattern, exists := llmResponse["deployment_pattern"]; exists {
		hashComponents = append(hashComponents, fmt.Sprintf("pattern:%v", pattern))
	}

	if resources, exists := llmResponse["resources"]; exists {
		hashComponents = append(hashComponents, fmt.Sprintf("resources:%v", resources))
	}

	combined := strings.Join(hashComponents, "|")

	return fmt.Sprintf("plan_%x", len(combined)+int(combined[0]))
}

// updateMetrics updates controller metrics.

func (c *SpecializedResourcePlanningController) updateMetrics(session *PlanningSession, success bool, totalDuration time.Duration) {
	c.metrics.mutex.Lock()

	defer c.metrics.mutex.Unlock()

	c.metrics.TotalPlanned++

	if success {
		c.metrics.SuccessfulPlanned++
	} else {
		c.metrics.FailedPlanned++
	}

	// Update average planning time.

	if c.metrics.TotalPlanned > 0 {

		totalTime := time.Duration(c.metrics.TotalPlanned-1) * c.metrics.AveragePlanningTime

		totalTime += totalDuration

		c.metrics.AveragePlanningTime = totalTime / time.Duration(c.metrics.TotalPlanned)

	} else {
		c.metrics.AveragePlanningTime = totalDuration
	}

	// Update per NF type metrics.

	for _, nfName := range session.RequiredNFs {

		if _, exists := c.metrics.NFTypeMetrics[nfName]; !exists {
			c.metrics.NFTypeMetrics[nfName] = &NFPlanningMetrics{}
		}

		nfMetrics := c.metrics.NFTypeMetrics[nfName]

		nfMetrics.Count++

		if success {
			// Update success rate.

			nfMetrics.SuccessRate = float64(nfMetrics.Count-1)/float64(nfMetrics.Count)*nfMetrics.SuccessRate + 1.0/float64(nfMetrics.Count)
		}

		// Update average planning time.

		totalNFTime := time.Duration(nfMetrics.Count-1) * nfMetrics.AveragePlanningTime

		totalNFTime += totalDuration

		nfMetrics.AveragePlanningTime = totalNFTime / time.Duration(nfMetrics.Count)

	}

	// Update cache hit rate.

	if c.planningCache != nil {

		totalRequests := c.metrics.TotalPlanned

		cacheHits := int64(0)

		c.planningCache.mutex.RLock()

		for _, entry := range c.planningCache.entries {
			cacheHits += entry.HitCount
		}

		c.planningCache.mutex.RUnlock()

		if totalRequests > 0 {
			c.metrics.CacheHitRate = float64(cacheHits) / float64(totalRequests)
		}

	}

	c.metrics.LastUpdated = time.Now()
}

// NewResourcePlanningMetrics creates new metrics instance.

func NewResourcePlanningMetrics() *ResourcePlanningMetrics {
	return &ResourcePlanningMetrics{
		NFTypeMetrics: make(map[string]*NFPlanningMetrics),
	}
}

// OptimizeResourceAllocation implements the ResourcePlanner interface.

func (c *SpecializedResourcePlanningController) OptimizeResourceAllocation(ctx context.Context, requirements *interfaces.ResourceRequirements) (*interfaces.OptimizedPlan, error) {
	if !c.Config.OptimizationEnabled {
		return nil, fmt.Errorf("optimization not enabled")
	}

	return c.OptimizationEngine.OptimizeAllocation(ctx, requirements, c.Config)
}

// ValidateResourceConstraints implements the ResourcePlanner interface.

func (c *SpecializedResourcePlanningController) ValidateResourceConstraints(ctx context.Context, plan *interfaces.ResourcePlan) error {
	if !c.Config.ConstraintCheckEnabled {
		return nil
	}

	return c.ConstraintSolver.ValidateConstraints(ctx, plan, c.ConstraintRules)
}

// EstimateResourceCosts implements the ResourcePlanner interface.

func (c *SpecializedResourcePlanningController) EstimateResourceCosts(ctx context.Context, plan *interfaces.ResourcePlan) (*interfaces.CostEstimate, error) {
	return c.CostEstimator.EstimateCosts(ctx, plan)
}

// GetPhaseStatus returns the status of a processing phase.

func (c *SpecializedResourcePlanningController) GetPhaseStatus(ctx context.Context, intentID string) (*interfaces.PhaseStatus, error) {
	if session, exists := c.activePlanning.Load(intentID); exists {

		s := session.(*PlanningSession)

		s.mutex.RLock()

		defer s.mutex.RUnlock()

		status := "Pending"

		if s.Progress > 0 && s.Progress < 1.0 {
			status = "InProgress"
		} else if s.Progress >= 1.0 {
			status = "Completed"
		}

		if len(s.Errors) > 0 {
			status = "Failed"
		}

		return &interfaces.PhaseStatus{
			Phase: interfaces.PhaseResourcePlanning,

			Status: status,

			StartTime: &metav1.Time{Time: s.StartTime},

			LastError: s.LastError,

			Metrics: map[string]float64{
				"progress": s.Progress,

				"calculation_time_ms": float64(s.Metrics.CalculationTime.Milliseconds()),

				"optimization_time_ms": float64(s.Metrics.OptimizationTime.Milliseconds()),

				"nfs_planned": float64(s.Metrics.NFsPlanned),
			},
		}, nil

	}

	return &interfaces.PhaseStatus{
		Phase: interfaces.PhaseResourcePlanning,

		Status: "Pending",
	}, nil
}

// HandlePhaseError handles errors during phase processing.

func (c *SpecializedResourcePlanningController) HandlePhaseError(ctx context.Context, intentID string, err error) error {
	c.Logger.Error(err, "Resource planning error", "intentId", intentID)

	if session, exists := c.activePlanning.Load(intentID); exists {

		s := session.(*PlanningSession)

		s.addError(err.Error())

	}

	return err
}

// GetDependencies returns phase dependencies.

func (c *SpecializedResourcePlanningController) GetDependencies() []interfaces.ProcessingPhase {
	return []interfaces.ProcessingPhase{interfaces.PhaseLLMProcessing}
}

// GetBlockedPhases returns phases blocked by this controller.

func (c *SpecializedResourcePlanningController) GetBlockedPhases() []interfaces.ProcessingPhase {
	return []interfaces.ProcessingPhase{
		interfaces.PhaseManifestGeneration,

		interfaces.PhaseGitOpsCommit,

		interfaces.PhaseDeploymentVerification,
	}
}

// SetupWithManager sets up the controller with the Manager.

func (c *SpecializedResourcePlanningController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		Complete(c)
}

// Start starts the controller.

func (c *SpecializedResourcePlanningController) Start(ctx context.Context) error {
	c.mutex.Lock()

	defer c.mutex.Unlock()

	if c.started {
		return fmt.Errorf("controller already started")
	}

	c.Logger.Info("Starting specialized resource planning controller")

	// Start background goroutines.

	go c.backgroundCleanup(ctx)

	go c.healthMonitoring(ctx)

	c.started = true

	c.healthStatus = interfaces.HealthStatus{
		Status: "Healthy",

		Message: "Resource planning controller started successfully",

		LastChecked: time.Now(),
	}

	return nil
}

// Stop stops the controller.

func (c *SpecializedResourcePlanningController) Stop(ctx context.Context) error {
	c.mutex.Lock()

	defer c.mutex.Unlock()

	if !c.started {
		return nil
	}

	c.Logger.Info("Stopping specialized resource planning controller")

	close(c.stopChan)

	c.started = false

	c.healthStatus = interfaces.HealthStatus{
		Status: "Stopped",

		Message: "Controller stopped",

		LastChecked: time.Now(),
	}

	return nil
}

// GetHealthStatus returns the health status of the controller.

func (c *SpecializedResourcePlanningController) GetHealthStatus(ctx context.Context) (interfaces.HealthStatus, error) {
	c.mutex.RLock()

	defer c.mutex.RUnlock()

	c.healthStatus.Metrics = json.RawMessage(`{}`)

	c.healthStatus.LastChecked = time.Now()

	return c.healthStatus, nil
}

// GetMetrics returns controller metrics.

func (c *SpecializedResourcePlanningController) GetMetrics(ctx context.Context) (map[string]float64, error) {
	c.metrics.mutex.RLock()

	defer c.metrics.mutex.RUnlock()

	return map[string]float64{
		"total_planned": float64(c.metrics.TotalPlanned),

		"successful_planned": float64(c.metrics.SuccessfulPlanned),

		"failed_planned": float64(c.metrics.FailedPlanned),

		"success_rate": c.getSuccessRate(),

		"average_planning_time_ms": float64(c.metrics.AveragePlanningTime.Milliseconds()),

		"average_cost_savings": c.metrics.AverageCostSavings,

		"cache_hit_rate": c.metrics.CacheHitRate,

		"constraint_violations": float64(c.metrics.ConstraintViolations),

		"active_planning": float64(c.getActivePlanningCount()),
	}, nil
}

// Reconcile implements the controller reconciliation logic.

func (c *SpecializedResourcePlanningController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get the NetworkIntent.

	var intent nephoranv1.NetworkIntent

	if err := c.Get(ctx, req.NamespacedName, &intent); err != nil {

		if apierrors.IsNotFound(err) {

			logger.Info("NetworkIntent resource not found, ignoring since object must be deleted")

			return ctrl.Result{}, nil

		}

		logger.Error(err, "Failed to get NetworkIntent")

		return ctrl.Result{}, err

	}

	// Check if this intent should be processed by this controller.

	if intent.Status.ProcessingPhase != string(interfaces.PhaseResourcePlanning) {
		return ctrl.Result{}, nil
	}

	// Process the intent.

	result, err := c.ProcessPhase(ctx, &intent, interfaces.PhaseResourcePlanning)
	if err != nil {

		logger.Error(err, "Failed to process resource planning phase")

		return ctrl.Result{RequeueAfter: time.Minute * 1}, err

	}

	// Update intent status based on result.

	if result.Success {

		intent.Status.ProcessingPhase = string(result.NextPhase)

		// Marshal result.Data to JSON and create RawExtension.

		if resultBytes, err := json.Marshal(result.Data); err != nil {

			logger.Error(err, "Failed to marshal result data")

			return ctrl.Result{}, err

		} else {
			intent.Status.ResourcePlan = runtime.RawExtension{Raw: resultBytes}
		}

		intent.Status.LastUpdated = metav1.Now()

	} else {

		intent.Status.ProcessingPhase = string(interfaces.PhaseFailed)

		intent.Status.ErrorMessage = result.ErrorMessage

		intent.Status.LastUpdated = metav1.Now()

	}

	// Update the intent status.

	if err := c.Status().Update(ctx, &intent); err != nil {

		logger.Error(err, "Failed to update NetworkIntent status")

		return ctrl.Result{}, err

	}

	logger.Info("Resource planning completed", "intentId", intent.Name, "success", result.Success)

	return ctrl.Result{}, nil
}

// Helper methods.

// backgroundCleanup performs background cleanup tasks.

func (c *SpecializedResourcePlanningController) backgroundCleanup(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			c.cleanupExpiredSessions()

			c.cleanupExpiredCache()

		case <-c.stopChan:

			return

		case <-ctx.Done():

			return

		}
	}
}

// healthMonitoring performs periodic health monitoring.

func (c *SpecializedResourcePlanningController) healthMonitoring(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			c.performHealthCheck()

		case <-c.stopChan:

			return

		case <-ctx.Done():

			return

		}
	}
}

// cleanupExpiredSessions removes expired planning sessions.

func (c *SpecializedResourcePlanningController) cleanupExpiredSessions() {
	expiredSessions := make([]string, 0)

	c.activePlanning.Range(func(key, value interface{}) bool {
		session := value.(*PlanningSession)

		if time.Since(session.StartTime) > time.Hour { // 1 hour expiration

			expiredSessions = append(expiredSessions, key.(string))
		}

		return true
	})

	for _, sessionID := range expiredSessions {

		c.activePlanning.Delete(sessionID)

		c.Logger.Info("Cleaned up expired planning session", "sessionId", sessionID)

	}
}

// cleanupExpiredCache removes expired cache entries.

func (c *SpecializedResourcePlanningController) cleanupExpiredCache() {
	if c.planningCache == nil {
		return
	}

	c.planningCache.mutex.Lock()

	defer c.planningCache.mutex.Unlock()

	expiredKeys := make([]string, 0)

	for key, entry := range c.planningCache.entries {
		if time.Since(entry.Timestamp) > c.planningCache.ttl {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(c.planningCache.entries, key)
	}

	if len(expiredKeys) > 0 {
		c.Logger.Info("Cleaned up expired cache entries", "count", len(expiredKeys))
	}
}

// performHealthCheck performs controller health checking.

func (c *SpecializedResourcePlanningController) performHealthCheck() {
	c.mutex.Lock()

	defer c.mutex.Unlock()

	successRate := c.getSuccessRate()

	activeCount := c.getActivePlanningCount()

	status := "Healthy"

	message := "Resource planning controller operating normally"

	if successRate < 0.8 && c.metrics.TotalPlanned > 10 {

		status = "Degraded"

		message = fmt.Sprintf("Low success rate: %.2f", successRate)

	}

	if activeCount > 50 { // Threshold for too many active sessions

		status = "Degraded"

		message = fmt.Sprintf("High active planning count: %d", activeCount)

	}

	c.healthStatus = interfaces.HealthStatus{
		Status: status,

		Message: message,

		LastChecked: time.Now(),

		Metrics: json.RawMessage(`{}`),
	}
}

// getSuccessRate calculates success rate.

func (c *SpecializedResourcePlanningController) getSuccessRate() float64 {
	if c.metrics.TotalPlanned == 0 {
		return 1.0
	}

	return float64(c.metrics.SuccessfulPlanned) / float64(c.metrics.TotalPlanned)
}

// getActivePlanningCount returns count of active planning sessions.

func (c *SpecializedResourcePlanningController) getActivePlanningCount() int {
	count := 0

	c.activePlanning.Range(func(key, value interface{}) bool {
		count++

		return true
	})

	return count
}

// Helper service implementations.

// NewTelecomResourceCalculator creates a new resource calculator.

func NewTelecomResourceCalculator(logger logr.Logger) *TelecomResourceCalculator {
	return &TelecomResourceCalculator{
		logger: logger,

		baselineProfiles: initializeBaselineProfiles(),

		scalingFactors: initializeScalingFactors(),

		performanceTargets: initializePerformanceTargets(),
	}
}

// ResourceOptimizationEngine handles resource optimization.

type ResourceOptimizationEngine struct {
	logger logr.Logger

	config ResourcePlanningConfig
}

// NewResourceOptimizationEngine creates a new optimization engine.

func NewResourceOptimizationEngine(logger logr.Logger, config ResourcePlanningConfig) *ResourceOptimizationEngine {
	return &ResourceOptimizationEngine{
		logger: logger,

		config: config,
	}
}

// OptimizeAllocation optimizes resource allocation.

func (e *ResourceOptimizationEngine) OptimizeAllocation(ctx context.Context, requirements *interfaces.ResourceRequirements, config ResourcePlanningConfig) (*interfaces.OptimizedPlan, error) {
	// Simple optimization: apply overcommit ratios.

	originalPlan := &interfaces.ResourcePlan{
		ResourceRequirements: *requirements,
	}

	optimizedRequirements := *requirements

	// Apply CPU overcommit.

	if cpuQuantity, err := resource.ParseQuantity(requirements.CPU); err == nil {

		optimizedCPU := cpuQuantity.DeepCopy()

		optimizedValue := float64(cpuQuantity.MilliValue()) / config.CPUOvercommitRatio

		optimizedCPU.SetMilli(int64(optimizedValue))

		optimizedRequirements.CPU = optimizedCPU.String()

	}

	// Apply memory overcommit.

	if memQuantity, err := resource.ParseQuantity(requirements.Memory); err == nil {

		optimizedMem := memQuantity.DeepCopy()

		optimizedValue := float64(memQuantity.Value()) / config.MemoryOvercommitRatio

		optimizedMem.Set(int64(optimizedValue))

		optimizedRequirements.Memory = optimizedMem.String()

	}

	optimizedPlan := &interfaces.ResourcePlan{
		ResourceRequirements: optimizedRequirements,
	}

	return &interfaces.OptimizedPlan{
		OriginalPlan: originalPlan,

		OptimizedPlan: optimizedPlan,

		Optimizations: []interfaces.Optimization{
			{
				Type: "cpu_overcommit",

				Description: fmt.Sprintf("Applied CPU overcommit ratio of %.2f", config.CPUOvercommitRatio),

				Impact: "Reduced CPU allocation while maintaining performance",
			},

			{
				Type: "memory_overcommit",

				Description: fmt.Sprintf("Applied memory overcommit ratio of %.2f", config.MemoryOvercommitRatio),

				Impact: "Reduced memory allocation while maintaining performance",
			},
		},
	}, nil
}

// ResourceConstraintSolver handles constraint validation.

type ResourceConstraintSolver struct {
	logger logr.Logger
}

// NewResourceConstraintSolver creates a new constraint solver.

func NewResourceConstraintSolver(logger logr.Logger) *ResourceConstraintSolver {
	return &ResourceConstraintSolver{
		logger: logger,
	}
}

// ValidateConstraints validates resource constraints.

func (s *ResourceConstraintSolver) ValidateConstraints(ctx context.Context, plan *interfaces.ResourcePlan, rules []*ResourceConstraintRule) error {
	var violations []string

	for _, rule := range rules {
		if err := rule.Rule(plan); err != nil {

			violations = append(violations, fmt.Sprintf("Constraint %s violated: %v", rule.Name, err))

			if rule.Type == "hard" {
				return fmt.Errorf("hard constraint violated: %s", rule.Name)
			}

		}
	}

	if len(violations) > 0 {
		s.logger.Info("Soft constraints violated", "violations", violations)
	}

	return nil
}

// TelecomCostEstimator estimates costs for telecom deployments.

type TelecomCostEstimator struct {
	logger logr.Logger

	pricingModel map[string]float64
}

// NewTelecomCostEstimator creates a new cost estimator.

func NewTelecomCostEstimator(logger logr.Logger) *TelecomCostEstimator {
	return &TelecomCostEstimator{
		logger: logger,

		pricingModel: map[string]float64{
			"cpu_core_hour": 0.05, // $0.05 per CPU core hour

			"memory_gb_hour": 0.01, // $0.01 per GB memory hour

			"storage_gb_month": 0.10, // $0.10 per GB storage per month

		},
	}
}

// EstimateCosts estimates costs for a resource plan.

func (e *TelecomCostEstimator) EstimateCosts(ctx context.Context, plan *interfaces.ResourcePlan) (*interfaces.CostEstimate, error) {
	costBreakdown := make(map[string]float64)

	// Parse CPU requirements.

	if cpuQuantity, err := resource.ParseQuantity(plan.ResourceRequirements.CPU); err == nil {

		cpuCores := float64(cpuQuantity.MilliValue()) / 1000.0

		hourlyCPUCost := cpuCores * e.pricingModel["cpu_core_hour"]

		costBreakdown["cpu"] = hourlyCPUCost * 24 * 30 // Monthly cost

	}

	// Parse memory requirements.

	if memQuantity, err := resource.ParseQuantity(plan.ResourceRequirements.Memory); err == nil {

		memoryGB := float64(memQuantity.Value()) / (1024 * 1024 * 1024)

		hourlyMemCost := memoryGB * e.pricingModel["memory_gb_hour"]

		costBreakdown["memory"] = hourlyMemCost * 24 * 30 // Monthly cost

	}

	// Parse storage requirements.

	if storageQuantity, err := resource.ParseQuantity(plan.ResourceRequirements.Storage); err == nil {

		storageGB := float64(storageQuantity.Value()) / (1024 * 1024 * 1024)

		monthlyStoageCost := storageGB * e.pricingModel["storage_gb_month"]

		costBreakdown["storage"] = monthlyStoageCost

	}

	// Calculate total cost.

	totalCost := 0.0

	for _, cost := range costBreakdown {
		totalCost += cost
	}

	return &interfaces.CostEstimate{
		TotalCost: totalCost,

		Currency: "USD",

		BillingPeriod: "monthly",

		CostBreakdown: costBreakdown,

		EstimationDate: time.Now(),
	}, nil
}

// Initialize helper functions.

// initializeResourceTemplates initializes resource templates for common NFs.

func initializeResourceTemplates() map[string]*NetworkFunctionTemplate {
	templates := make(map[string]*NetworkFunctionTemplate)

	// AMF Template.

	templates["amf"] = &NetworkFunctionTemplate{
		Name: "amf",

		Type: "amf",

		Category: "5GC",

		DefaultResources: &interfaces.ResourceSpec{
			Requests: interfaces.ResourceRequirements{
				CPU: "500m",

				Memory: "1Gi",

				Storage: "10Gi",
			},

			Limits: interfaces.ResourceRequirements{
				CPU: "2",

				Memory: "4Gi",

				Storage: "20Gi",
			},
		},

		Dependencies: []string{"nrf", "ausf"},

		AntiAffinity: []string{"amf"},
	}

	// SMF Template.

	templates["smf"] = &NetworkFunctionTemplate{
		Name: "smf",

		Type: "smf",

		Category: "5GC",

		DefaultResources: &interfaces.ResourceSpec{
			Requests: interfaces.ResourceRequirements{
				CPU: "500m",

				Memory: "1Gi",

				Storage: "10Gi",
			},

			Limits: interfaces.ResourceRequirements{
				CPU: "2",

				Memory: "4Gi",

				Storage: "20Gi",
			},
		},

		Dependencies: []string{"nrf", "udm", "upf"},

		AntiAffinity: []string{"smf"},
	}

	// UPF Template.

	templates["upf"] = &NetworkFunctionTemplate{
		Name: "upf",

		Type: "upf",

		Category: "5GC",

		DefaultResources: &interfaces.ResourceSpec{
			Requests: interfaces.ResourceRequirements{
				CPU: "1",

				Memory: "2Gi",

				Storage: "20Gi",
			},

			Limits: interfaces.ResourceRequirements{
				CPU: "4",

				Memory: "8Gi",

				Storage: "100Gi",
			},
		},

		PreferredNodes: []string{"high-performance"},
	}

	return templates
}

// initializeConstraintRules initializes constraint rules.

func initializeConstraintRules() []*ResourceConstraintRule {
	var rules []*ResourceConstraintRule

	// CPU limit constraint.

	rules = append(rules, &ResourceConstraintRule{
		ID: "cpu-limit",

		Name: "CPU Limit Constraint",

		Type: "soft",

		Scope: "global",

		Rule: func(plan *interfaces.ResourcePlan) error {
			if cpuQuantity, err := resource.ParseQuantity(plan.ResourceRequirements.CPU); err == nil {
				if cpuQuantity.MilliValue() > 100000 { // 100 CPU cores

					return fmt.Errorf("CPU requirement exceeds limit: %s", plan.ResourceRequirements.CPU)
				}
			}

			return nil
		},

		ViolationAction: "warn",
	})

	// Memory limit constraint.

	rules = append(rules, &ResourceConstraintRule{
		ID: "memory-limit",

		Name: "Memory Limit Constraint",

		Type: "soft",

		Scope: "global",

		Rule: func(plan *interfaces.ResourcePlan) error {
			if memQuantity, err := resource.ParseQuantity(plan.ResourceRequirements.Memory); err == nil {
				if memQuantity.Value() > 500*1024*1024*1024 { // 500 GB

					return fmt.Errorf("Memory requirement exceeds limit: %s", plan.ResourceRequirements.Memory)
				}
			}

			return nil
		},

		ViolationAction: "warn",
	})

	return rules
}

// initializeBaselineProfiles initializes baseline resource profiles.

func initializeBaselineProfiles() map[string]*ResourceProfile {
	profiles := make(map[string]*ResourceProfile)

	profiles["amf"] = &ResourceProfile{
		NetworkFunction: "amf",

		BaselineCPU: resource.MustParse("500m"),

		BaselineMemory: resource.MustParse("1Gi"),

		BaselineStorage: resource.MustParse("10Gi"),

		NetworkBandwidth: "1Gbps",

		IOPSRequirement: 1000,
	}

	profiles["smf"] = &ResourceProfile{
		NetworkFunction: "smf",

		BaselineCPU: resource.MustParse("500m"),

		BaselineMemory: resource.MustParse("1Gi"),

		BaselineStorage: resource.MustParse("10Gi"),

		NetworkBandwidth: "1Gbps",

		IOPSRequirement: 1000,
	}

	profiles["upf"] = &ResourceProfile{
		NetworkFunction: "upf",

		BaselineCPU: resource.MustParse("1"),

		BaselineMemory: resource.MustParse("2Gi"),

		BaselineStorage: resource.MustParse("20Gi"),

		NetworkBandwidth: "10Gbps",

		IOPSRequirement: 5000,
	}

	return profiles
}

// initializeScalingFactors initializes scaling factors.

func initializeScalingFactors() map[string]*ScalingProfile {
	profiles := make(map[string]*ScalingProfile)

	profiles["amf"] = &ScalingProfile{
		NetworkFunction: "amf",

		CPUScalingFactor: 1.2,

		MemoryScalingFactor: 1.1,

		StorageScalingFactor: 1.0,

		MaxReplicas: 10,

		MinReplicas: 2,

		ScalingThreshold: 0.8,
	}

	profiles["smf"] = &ScalingProfile{
		NetworkFunction: "smf",

		CPUScalingFactor: 1.3,

		MemoryScalingFactor: 1.2,

		StorageScalingFactor: 1.0,

		MaxReplicas: 10,

		MinReplicas: 2,

		ScalingThreshold: 0.8,
	}

	profiles["upf"] = &ScalingProfile{
		NetworkFunction: "upf",

		CPUScalingFactor: 1.5,

		MemoryScalingFactor: 1.3,

		StorageScalingFactor: 1.1,

		MaxReplicas: 5,

		MinReplicas: 1,

		ScalingThreshold: 0.9,
	}

	return profiles
}

// initializePerformanceTargets initializes performance targets.

func initializePerformanceTargets() map[string]*PerformanceTarget {
	targets := make(map[string]*PerformanceTarget)

	targets["amf"] = &PerformanceTarget{
		NetworkFunction: "amf",

		MaxLatency: time.Millisecond * 10,

		MinThroughput: 1000.0,

		MaxErrorRate: 0.01,

		AvailabilityTarget: 0.999,
	}

	targets["smf"] = &PerformanceTarget{
		NetworkFunction: "smf",

		MaxLatency: time.Millisecond * 20,

		MinThroughput: 500.0,

		MaxErrorRate: 0.01,

		AvailabilityTarget: 0.999,
	}

	targets["upf"] = &PerformanceTarget{
		NetworkFunction: "upf",

		MaxLatency: time.Millisecond * 5,

		MinThroughput: 10000.0,

		MaxErrorRate: 0.001,

		AvailabilityTarget: 0.9999,
	}

	return targets
}

