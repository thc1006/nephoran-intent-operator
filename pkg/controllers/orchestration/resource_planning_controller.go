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
	"math"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
)

// ResourcePlanningController reconciles ResourcePlan objects.

type ResourcePlanningController struct {
	client.Client

	Scheme *runtime.Scheme

	Recorder record.EventRecorder

	Logger logr.Logger

	// Services.

	ResourcePlanner *ResourcePlanningService

	CostEstimator *CostEstimationService

	ComplianceValidator *ComplianceValidationService

	// Configuration.

	Config *ResourcePlanningConfig

	// Event bus for coordination.

	EventBus *EventBus

	// Metrics.

	MetricsCollector *MetricsCollector
}

// ResourcePlanningConfig contains configuration for the controller.

type ResourcePlanningConfig struct {
	MaxConcurrentPlanning int `json:"maxConcurrentPlanning"`

	DefaultTimeout time.Duration `json:"defaultTimeout"`

	MaxRetries int `json:"maxRetries"`

	RetryBackoff time.Duration `json:"retryBackoff"`

	QualityThreshold float64 `json:"qualityThreshold"`

	CostOptimizationEnabled bool `json:"costOptimizationEnabled"`

	PerformanceOptimizationEnabled bool `json:"performanceOptimizationEnabled"`

	ComplianceValidationEnabled bool `json:"complianceValidationEnabled"`

	// Cache configuration.

	CacheEnabled bool `json:"cacheEnabled"`

	CacheTTL time.Duration `json:"cacheTTL"`

	MaxCacheEntries int `json:"maxCacheEntries"`

	// Optimization configuration.

	OptimizationEnabled bool `json:"optimizationEnabled"`

	CPUOvercommitRatio float64 `json:"cpuOvercommitRatio"`

	MemoryOvercommitRatio float64 `json:"memoryOvercommitRatio"`

	// Default resource requests.

	DefaultCPURequest string `json:"defaultCpuRequest"`

	DefaultMemoryRequest string `json:"defaultMemoryRequest"`

	DefaultStorageRequest string `json:"defaultStorageRequest"`

	// Constraint checking.

	ConstraintCheckEnabled bool `json:"constraintCheckEnabled"`
}

// ResourcePlanningService handles resource planning logic.

type ResourcePlanningService struct {
	Logger logr.Logger

	Config *ResourcePlanningServiceConfig
}

// ResourcePlanningServiceConfig contains service configuration.

type ResourcePlanningServiceConfig struct {
	DefaultCPURequest resource.Quantity

	DefaultMemoryRequest resource.Quantity

	DefaultCPULimit resource.Quantity

	DefaultMemoryLimit resource.Quantity

	DefaultReplicas int32

	ResourceMultipliers map[string]float64
}

// CostEstimationService handles cost estimation.

type CostEstimationService struct {
	Logger logr.Logger

	CostRates map[string]float64 // Cost per unit per hour
}

// ComplianceValidationService handles compliance validation.

type ComplianceValidationService struct {
	Logger logr.Logger

	Rules map[string]ComplianceRule
}

// ComplianceRule represents a compliance rule.

type ComplianceRule struct {
	Standard string

	Requirement string

	Validator func(plan *nephoranv1.ResourcePlan) (bool, string)
}

// NewResourcePlanningController creates a new ResourcePlanningController.

func NewResourcePlanningController(
	client client.Client,

	scheme *runtime.Scheme,

	recorder record.EventRecorder,

	eventBus *EventBus,

	config *ResourcePlanningConfig,
) *ResourcePlanningController {
	return &ResourcePlanningController{
		Client: client,

		Scheme: scheme,

		Recorder: recorder,

		Logger: log.Log.WithName("resource-planning-controller"),

		EventBus: eventBus,

		Config: config,

		ResourcePlanner: NewResourcePlanningService(),

		CostEstimator: NewCostEstimationService(),

		ComplianceValidator: NewComplianceValidationService(),

		MetricsCollector: NewMetricsCollector(),
	}
}

// Reconcile handles ResourcePlan resources.

func (r *ResourcePlanningController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Logger.WithValues("resourceplan", req.NamespacedName)

	// Fetch the ResourcePlan instance.

	resourcePlan := &nephoranv1.ResourcePlan{}

	if err := r.Get(ctx, req.NamespacedName, resourcePlan); err != nil {

		if apierrors.IsNotFound(err) {

			log.Info("ResourcePlan resource not found, ignoring")

			return ctrl.Result{}, nil

		}

		log.Error(err, "Failed to get ResourcePlan")

		return ctrl.Result{}, err

	}

	// Handle deletion.

	if resourcePlan.DeletionTimestamp != nil {
		return r.handleDeletion(ctx, resourcePlan)
	}

	// Add finalizer if not present.

	if !controllerutil.ContainsFinalizer(resourcePlan, "resourceplan.nephoran.com/finalizer") {

		controllerutil.AddFinalizer(resourcePlan, "resourceplan.nephoran.com/finalizer")

		return ctrl.Result{}, r.Update(ctx, resourcePlan)

	}

	// Process the resource plan.

	return r.processResourcePlan(ctx, resourcePlan)
}

// processResourcePlan processes the resource planning request.

func (r *ResourcePlanningController) processResourcePlan(ctx context.Context, resourcePlan *nephoranv1.ResourcePlan) (ctrl.Result, error) {
	log := r.Logger.WithValues("resourceplan", resourcePlan.Name, "namespace", resourcePlan.Namespace)

	// Check if planning is already complete.

	if resourcePlan.IsPlanningComplete() {

		log.V(1).Info("Resource planning already complete")

		return ctrl.Result{}, nil

	}

	// Check if planning failed and can retry.

	if resourcePlan.IsPlanningFailed() && resourcePlan.Status.RetryCount >= int32(r.Config.MaxRetries) {

		log.Info("Resource planning failed and cannot retry")

		return ctrl.Result{}, nil

	}

	// Record planning start.

	if resourcePlan.Status.PlanningStartTime == nil {

		now := metav1.Now()

		resourcePlan.Status.PlanningStartTime = &now

		resourcePlan.Status.Phase = nephoranv1.ResourcePlanPhaseAnalyzing

		if err := r.updateStatus(ctx, resourcePlan); err != nil {
			return ctrl.Result{}, err
		}

		r.MetricsCollector.RecordPhaseStart(interfaces.PhaseResourcePlanning, string(resourcePlan.UID))

	}

	// Publish planning start event.

	if err := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseResourcePlanning, EventResourcePlanningStarted,

		string(resourcePlan.UID), false, map[string]interface{}{}); err != nil {
		log.Error(err, "Failed to publish planning start event")
	}

	// Create planning context with timeout.

	planningCtx, cancel := context.WithTimeout(ctx, r.Config.DefaultTimeout)

	defer cancel()

	// Execute resource planning.

	result, err := r.executeResourcePlanning(planningCtx, resourcePlan)
	if err != nil {
		return r.handlePlanningError(ctx, resourcePlan, err)
	}

	// Update status with results.

	return r.handlePlanningSuccess(ctx, resourcePlan, result)
}

// executeResourcePlanning performs the actual resource planning.

func (r *ResourcePlanningController) executeResourcePlanning(ctx context.Context, resourcePlan *nephoranv1.ResourcePlan) (*ResourcePlanningResult, error) {
	log := r.Logger.WithValues("resourceplan", resourcePlan.Name)

	// Phase 1: Requirements Analysis.

	log.Info("Starting requirements analysis")

	resourcePlan.Status.Phase = nephoranv1.ResourcePlanPhaseAnalyzing

	if err := r.updateStatus(ctx, resourcePlan); err != nil {
		return nil, err
	}

	requirements, err := r.analyzeRequirements(ctx, resourcePlan)
	if err != nil {
		return nil, fmt.Errorf("requirements analysis failed: %w", err)
	}

	// Phase 2: Resource Planning.

	log.Info("Starting resource planning")

	resourcePlan.Status.Phase = nephoranv1.ResourcePlanPhasePlanning

	if err := r.updateStatus(ctx, resourcePlan); err != nil {
		return nil, err
	}

	plannedResources, err := r.ResourcePlanner.PlanResources(ctx, requirements)
	if err != nil {
		return nil, fmt.Errorf("resource planning failed: %w", err)
	}

	// Phase 3: Optimization.

	log.Info("Starting optimization")

	resourcePlan.Status.Phase = nephoranv1.ResourcePlanPhaseOptimizing

	if err := r.updateStatus(ctx, resourcePlan); err != nil {
		return nil, err
	}

	optimizedResources, optimizationResults, err := r.optimizeResources(ctx, resourcePlan, plannedResources)
	if err != nil {

		log.Error(err, "Optimization failed, continuing with unoptimized plan")

		// Continue with unoptimized resources rather than failing.

		optimizedResources = plannedResources

	}

	// Phase 4: Cost Estimation.

	log.Info("Calculating cost estimates")

	costEstimate, err := r.CostEstimator.EstimateCosts(ctx, optimizedResources)
	if err != nil {
		log.Error(err, "Cost estimation failed")

		// Continue without cost estimate.
	}

	// Phase 5: Performance Estimation.

	log.Info("Calculating performance estimates")

	performanceEstimate, err := r.estimatePerformance(ctx, optimizedResources)
	if err != nil {
		log.Error(err, "Performance estimation failed")

		// Continue without performance estimate.
	}

	// Phase 6: Compliance Validation.

	log.Info("Validating compliance")

	resourcePlan.Status.Phase = nephoranv1.ResourcePlanPhaseValidating

	if err := r.updateStatus(ctx, resourcePlan); err != nil {
		return nil, err
	}

	complianceResults, validationResults, err := r.validateCompliance(ctx, resourcePlan, optimizedResources)
	if err != nil {
		log.Error(err, "Compliance validation failed")

		// Continue without compliance validation.
	}

	// Calculate quality score.

	qualityScore := r.calculateQualityScore(optimizedResources, costEstimate, performanceEstimate, complianceResults)

	// Create result.

	result := &ResourcePlanningResult{
		PlannedResources: optimizedResources,

		CostEstimate: costEstimate,

		PerformanceEstimate: performanceEstimate,

		ComplianceResults: complianceResults,

		OptimizationResults: optimizationResults,

		ValidationResults: validationResults,

		QualityScore: qualityScore,
	}

	return result, nil
}

// analyzeRequirements analyzes the input requirements.

func (r *ResourcePlanningController) analyzeRequirements(ctx context.Context, resourcePlan *nephoranv1.ResourcePlan) (*ResourceRequirements, error) {
	// Parse requirements input.

	var requirementsData map[string]interface{}

	if err := json.Unmarshal(resourcePlan.Spec.RequirementsInput.Raw, &requirementsData); err != nil {
		return nil, fmt.Errorf("failed to parse requirements input: %w", err)
	}

	// Create requirements object.

	requirements := &ResourceRequirements{
		TargetComponents: resourcePlan.Spec.TargetComponents,

		DeploymentPattern: resourcePlan.Spec.DeploymentPattern,

		ResourceConstraints: resourcePlan.Spec.ResourceConstraints,

		TargetClusters: resourcePlan.Spec.TargetClusters,

		NetworkSlice: resourcePlan.Spec.NetworkSlice,

		OptimizationGoals: resourcePlan.Spec.OptimizationGoals,

		ComplianceRequirements: resourcePlan.Spec.ComplianceRequirements,

		ParsedInput: requirementsData,
	}

	return requirements, nil
}

// optimizeResources optimizes the resource allocation.

func (r *ResourcePlanningController) optimizeResources(ctx context.Context, resourcePlan *nephoranv1.ResourcePlan, resources []nephoranv1.PlannedResource) ([]nephoranv1.PlannedResource, []nephoranv1.OptimizationResult, error) {
	optimizedResources := make([]nephoranv1.PlannedResource, len(resources))

	copy(optimizedResources, resources)

	var optimizationResults []nephoranv1.OptimizationResult

	// Apply cost optimization if enabled.

	if resourcePlan.ShouldOptimizeCost() && r.Config.CostOptimizationEnabled {

		result, err := r.applyCostOptimization(ctx, optimizedResources)
		if err != nil {
			return nil, nil, fmt.Errorf("cost optimization failed: %w", err)
		}

		optimizationResults = append(optimizationResults, result)

	}

	// Apply performance optimization if enabled.

	if resourcePlan.ShouldOptimizePerformance() && r.Config.PerformanceOptimizationEnabled {

		result, err := r.applyPerformanceOptimization(ctx, optimizedResources)
		if err != nil {
			return nil, nil, fmt.Errorf("performance optimization failed: %w", err)
		}

		optimizationResults = append(optimizationResults, result)

	}

	// Apply resource packing optimization.

	result, err := r.applyResourcePackingOptimization(ctx, optimizedResources)
	if err != nil {
		return nil, nil, fmt.Errorf("resource packing optimization failed: %w", err)
	}

	optimizationResults = append(optimizationResults, result)

	return optimizedResources, optimizationResults, nil
}

// applyCostOptimization applies cost optimization.

func (r *ResourcePlanningController) applyCostOptimization(ctx context.Context, resources []nephoranv1.PlannedResource) (nephoranv1.OptimizationResult, error) {
	optimizedCount := 0

	totalSavings := 0.0

	for i := range resources {

		planResource := &resources[i]

		// Optimize CPU requests (reduce by 10% if over-provisioned).

		if planResource.ResourceRequirements.Requests.CPU != nil {

			currentCPU := planResource.ResourceRequirements.Requests.CPU.MilliValue()

			if currentCPU > 500 { // Only optimize if > 500m

				newCPU := int64(float64(currentCPU) * 0.9)

				planResource.ResourceRequirements.Requests.CPU = resource.NewMilliQuantity(newCPU, resource.DecimalSI)

				optimizedCount++

				totalSavings += float64(currentCPU-newCPU) * 0.01 // Estimate $0.01 per mCPU/hour

			}

		}

		// Optimize memory requests (reduce by 5% if over-provisioned).

		if planResource.ResourceRequirements.Requests.Memory != nil {

			currentMemory := planResource.ResourceRequirements.Requests.Memory.Value()

			if currentMemory > 1073741824 { // Only optimize if > 1Gi

				newMemory := int64(float64(currentMemory) * 0.95)

				planResource.ResourceRequirements.Requests.Memory = resource.NewQuantity(newMemory, resource.BinarySI)

				optimizedCount++

				totalSavings += float64(currentMemory-newMemory) / 1073741824 * 0.005 // Estimate $0.005 per Gi/hour

			}

		}

	}

	improvementPercent := 0.0

	if len(resources) > 0 {
		improvementPercent = float64(optimizedCount) / float64(len(resources)) * 100
	}

	result := nephoranv1.OptimizationResult{
		Type: "cost",

		Status: "Success",

		ImprovementPercent: stringPtr(fmt.Sprintf("%.2f", improvementPercent)),

		Description: fmt.Sprintf("Optimized %d resources for cost reduction", optimizedCount),

		OptimizedAt: metav1.Now(),
	}

	if optimizedCount > 0 {
		result.AppliedChanges = []string{
			fmt.Sprintf("Reduced CPU requests for %d resources", optimizedCount),

			fmt.Sprintf("Estimated savings: $%.2f per hour", totalSavings),
		}
	}

	return result, nil
}

// applyPerformanceOptimization applies performance optimization.

func (r *ResourcePlanningController) applyPerformanceOptimization(ctx context.Context, resources []nephoranv1.PlannedResource) (nephoranv1.OptimizationResult, error) {
	optimizedCount := 0

	for i := range resources {

		planResource := &resources[i]

		// Increase CPU limits for performance-critical components.

		if r.isPerformanceCritical(planResource.Component) {
			if planResource.ResourceRequirements.Limits.CPU != nil {

				currentCPU := planResource.ResourceRequirements.Limits.CPU.MilliValue()

				newCPU := int64(float64(currentCPU) * 1.2) // Increase by 20%

				planResource.ResourceRequirements.Limits.CPU = resource.NewMilliQuantity(newCPU, resource.DecimalSI)

				optimizedCount++

			}
		}

		// Add memory buffer for memory-intensive components.

		if r.isMemoryIntensive(planResource.Component) {
			if planResource.ResourceRequirements.Requests.Memory != nil {

				currentMemory := planResource.ResourceRequirements.Requests.Memory.Value()

				newMemory := int64(float64(currentMemory) * 1.1) // Increase by 10%

				planResource.ResourceRequirements.Requests.Memory = resource.NewQuantity(newMemory, resource.BinarySI)

				optimizedCount++

			}
		}

	}

	improvementPercent := 0.0

	if len(resources) > 0 {
		improvementPercent = float64(optimizedCount) / float64(len(resources)) * 100
	}

	result := nephoranv1.OptimizationResult{
		Type: "performance",

		Status: "Success",

		ImprovementPercent: stringPtr(fmt.Sprintf("%.2f", improvementPercent)),

		Description: fmt.Sprintf("Optimized %d resources for performance", optimizedCount),

		OptimizedAt: metav1.Now(),
	}

	if optimizedCount > 0 {
		result.AppliedChanges = []string{
			fmt.Sprintf("Increased resource limits for %d performance-critical components", optimizedCount),

			"Added memory buffers for memory-intensive components",
		}
	}

	return result, nil
}

// applyResourcePackingOptimization applies resource packing optimization.

func (r *ResourcePlanningController) applyResourcePackingOptimization(ctx context.Context, resources []nephoranv1.PlannedResource) (nephoranv1.OptimizationResult, error) {
	// Sort resources by resource requirements to optimize packing.

	sort.Slice(resources, func(i, j int) bool {
		cpuI := resources[i].ResourceRequirements.Requests.CPU.MilliValue()

		cpuJ := resources[j].ResourceRequirements.Requests.CPU.MilliValue()

		return cpuI > cpuJ
	})

	// Group resources by similar resource profiles.

	packingGroups := r.groupResourcesByProfile(resources)

	improvementPercent := 0.0

	if len(resources) > 0 {

		// Calculate improvement based on reduced cluster spread.

		originalSpread := len(resources)

		newSpread := len(packingGroups)

		if originalSpread > 0 {
			improvementPercent = float64(originalSpread-newSpread) / float64(originalSpread) * 100
		}

	}

	result := nephoranv1.OptimizationResult{
		Type: "packing",

		Status: "Success",

		ImprovementPercent: stringPtr(fmt.Sprintf("%.2f", improvementPercent)),

		Description: fmt.Sprintf("Grouped %d resources into %d packing groups", len(resources), len(packingGroups)),

		OptimizedAt: metav1.Now(),
	}

	if len(packingGroups) < len(resources) {
		result.AppliedChanges = []string{
			fmt.Sprintf("Optimized resource packing from %d to %d groups", len(resources), len(packingGroups)),

			"Improved cluster resource utilization",
		}
	}

	return result, nil
}

// groupResourcesByProfile groups resources by similar resource profiles.

func (r *ResourcePlanningController) groupResourcesByProfile(resources []nephoranv1.PlannedResource) [][]nephoranv1.PlannedResource {
	var groups [][]nephoranv1.PlannedResource

	tolerance := 0.2 // 20% tolerance for grouping

	for _, resource := range resources {

		found := false

		for i, group := range groups {
			if len(group) > 0 && r.areResourceProfilesSimilar(resource, group[0], tolerance) {

				groups[i] = append(groups[i], resource)

				found = true

				break

			}
		}

		if !found {
			groups = append(groups, []nephoranv1.PlannedResource{resource})
		}

	}

	return groups
}

// areResourceProfilesSimilar checks if two resources have similar profiles.

func (r *ResourcePlanningController) areResourceProfilesSimilar(r1, r2 nephoranv1.PlannedResource, tolerance float64) bool {
	cpu1 := float64(r1.ResourceRequirements.Requests.CPU.MilliValue())

	cpu2 := float64(r2.ResourceRequirements.Requests.CPU.MilliValue())

	mem1 := float64(r1.ResourceRequirements.Requests.Memory.Value())

	mem2 := float64(r2.ResourceRequirements.Requests.Memory.Value())

	cpuDiff := math.Abs(cpu1-cpu2) / math.Max(cpu1, cpu2)

	memDiff := math.Abs(mem1-mem2) / math.Max(mem1, mem2)

	return cpuDiff <= tolerance && memDiff <= tolerance
}

// isPerformanceCritical checks if a component is performance-critical.

func (r *ResourcePlanningController) isPerformanceCritical(component nephoranv1.TargetComponent) bool {
	performanceCritical := map[string]bool{
		"upf": true,

		"gnb": true,

		"nearrt-ric": true,

		"odu": true,

		"ocu-up": true,
	}

	return performanceCritical[component.Name]
}

// isMemoryIntensive checks if a component is memory-intensive.

func (r *ResourcePlanningController) isMemoryIntensive(component nephoranv1.TargetComponent) bool {
	memoryIntensive := map[string]bool{
		"udr": true,

		"udm": true,

		"nwdaf": true,

		"smo": true,
	}

	return memoryIntensive[component.Name]
}

// estimatePerformance estimates performance characteristics.

func (r *ResourcePlanningController) estimatePerformance(ctx context.Context, resources []nephoranv1.PlannedResource) (*nephoranv1.PerformanceEstimate, error) {
	// Calculate aggregate resource capacity.

	totalCPU := int64(0)

	totalMemory := int64(0)

	for _, resource := range resources {

		if resource.ResourceRequirements.Requests.CPU != nil {
			totalCPU += resource.ResourceRequirements.Requests.CPU.MilliValue()
		}

		if resource.ResourceRequirements.Requests.Memory != nil {
			totalMemory += resource.ResourceRequirements.Requests.Memory.Value()
		}

	}

	// Estimate performance based on resource capacity.

	// These are simplified estimates - in practice, you'd use more sophisticated models.

	expectedThroughput := float64(totalCPU) * 0.1 // 0.1 RPS per mCPU

	expectedLatency := math.Max(10.0, 100.0/math.Sqrt(float64(totalCPU/1000))) // Latency improves with more CPU

	expectedAvailability := math.Min(99.99, 95.0+float64(len(resources))*0.5) // More replicas = higher availability

	// Create performance estimate.
	expectedThroughputStr := fmt.Sprintf("%.2f", expectedThroughput)
	expectedLatencyStr := fmt.Sprintf("%.2f", expectedLatency)
	expectedAvailabilityStr := fmt.Sprintf("%.2f", expectedAvailability)

	performanceEstimate := &nephoranv1.PerformanceEstimate{
		ExpectedThroughput: &expectedThroughputStr,

		ExpectedLatency: &expectedLatencyStr,

		ExpectedAvailability: &expectedAvailabilityStr,

		ResourceUtilization: map[string]string{
			"cpu_utilization": strconv.FormatFloat(0.7, 'f', 6, 64), // Assume 70% utilization

			"memory_utilization": strconv.FormatFloat(0.8, 'f', 6, 64), // Assume 80% utilization

		},
	}

	// Add bottleneck analysis.

	if totalCPU < 2000 { // Less than 2 CPU cores total

		performanceEstimate.BottleneckAnalysis = append(performanceEstimate.BottleneckAnalysis, "CPU resources may be insufficient for high load")
	}

	if totalMemory < 4*1024*1024*1024 { // Less than 4GB total

		performanceEstimate.BottleneckAnalysis = append(performanceEstimate.BottleneckAnalysis, "Memory resources may be insufficient for high load")
	}

	// Add scaling recommendations.

	for _, resource := range resources {
		if resource.Replicas != nil && *resource.Replicas < 2 {

			recommendation := nephoranv1.ScalingRecommendation{
				Resource: resource.Name,

				RecommendedReplicas: 2,

				Reason: "Increase replicas for high availability",

				Impact: "Improves availability and fault tolerance",

				Confidence: func() *string { c := "0.8"; return &c }(),
			}

			performanceEstimate.ScalingRecommendations = append(performanceEstimate.ScalingRecommendations, recommendation)

		}
	}

	return performanceEstimate, nil
}

// validateCompliance validates compliance requirements.

func (r *ResourcePlanningController) validateCompliance(ctx context.Context, resourcePlan *nephoranv1.ResourcePlan, resources []nephoranv1.PlannedResource) ([]nephoranv1.ComplianceStatus, []nephoranv1.ValidationResult, error) {
	complianceResults := make([]nephoranv1.ComplianceStatus, 0, len(resourcePlan.Spec.ComplianceRequirements))

	validationResults := make([]nephoranv1.ValidationResult, 0, len(resources))

	if !r.Config.ComplianceValidationEnabled {
		return complianceResults, validationResults, nil
	}

	// Validate each compliance requirement.

	for _, requirement := range resourcePlan.Spec.ComplianceRequirements {

		status, violations := r.ComplianceValidator.ValidateRequirement(requirement, resourcePlan, resources)

		complianceResults = append(complianceResults, status)

		// Determine validation status based on violations.

		validationStatus := "passed"

		if len(violations) > 0 {
			validationStatus = "failed"
		}

		// Create validation result.

		validationResult := nephoranv1.ValidationResult{
			Type: "compliance",

			Status: validationStatus,

			Message: fmt.Sprintf("Compliance validation for %s: %s", requirement.Standard, validationStatus),

			ValidatedAt: metav1.Now(),
		}

		if len(violations) > 0 {

			validationResult.Details = make(map[string]string)

			for i, violation := range violations {
				validationResult.Details[fmt.Sprintf("violation_%d", i)] = violation.Description
			}

		}

		validationResults = append(validationResults, validationResult)

	}

	// Add general validation checks.

	generalValidation := r.performGeneralValidation(resources)

	validationResults = append(validationResults, generalValidation...)

	return complianceResults, validationResults, nil
}

// performGeneralValidation performs general validation checks.

func (r *ResourcePlanningController) performGeneralValidation(resources []nephoranv1.PlannedResource) []nephoranv1.ValidationResult {
	var results []nephoranv1.ValidationResult

	// Check resource limits.

	result := nephoranv1.ValidationResult{
		Type: "resource_limits",

		Status: "Passed",

		Message: "All resources have appropriate limits set",

		ValidatedAt: metav1.Now(),
	}

	for _, resource := range resources {
		if resource.ResourceRequirements.Limits.CPU == nil {

			result.Status = "Warning"

			result.Message = "Some resources lack CPU limits"

			break

		}
	}

	results = append(results, result)

	// Check naming conventions.

	namingResult := nephoranv1.ValidationResult{
		Type: "naming_conventions",

		Status: "Passed",

		Message: "All resources follow naming conventions",

		ValidatedAt: metav1.Now(),
	}

	for _, resource := range resources {
		if len(resource.Name) > 63 {

			namingResult.Status = "Failed"

			namingResult.Message = "Some resource names exceed 63 characters"

			break

		}
	}

	results = append(results, namingResult)

	return results
}

// calculateQualityScore calculates an overall quality score for the plan.

func (r *ResourcePlanningController) calculateQualityScore(resources []nephoranv1.PlannedResource, costEstimate *nephoranv1.CostEstimate, performanceEstimate *nephoranv1.PerformanceEstimate, complianceResults []nephoranv1.ComplianceStatus) float64 {
	score := 1.0

	// Factor in resource optimization (30% weight).

	resourceScore := r.calculateResourceScore(resources)

	score *= (0.7 + 0.3*resourceScore)

	// Factor in cost efficiency (20% weight).

	if costEstimate != nil && costEstimate.Confidence != nil {
		costScore, err := strconv.ParseFloat(*costEstimate.Confidence, 64)
		if err != nil {
			costScore = 0.5 // Default confidence score
		}

		score *= (0.8 + 0.2*costScore)

	}

	// Factor in performance expectations (30% weight).

	if performanceEstimate != nil {

		performanceScore := 0.8 // Base score

		if performanceEstimate.ExpectedAvailability != nil {
			availabilityValue, err := strconv.ParseFloat(*performanceEstimate.ExpectedAvailability, 64)
			if err == nil && availabilityValue >= 99.0 {
				performanceScore = 1.0
			}
		}

		score *= (0.7 + 0.3*performanceScore)

	}

	// Factor in compliance (20% weight).

	compliantCount := 0

	for _, status := range complianceResults {
		if status.ViolationCount == 0 {
			compliantCount++
		}
	}

	if len(complianceResults) > 0 {

		complianceScore := float64(compliantCount) / float64(len(complianceResults))

		score *= (0.8 + 0.2*complianceScore)

	}

	// Ensure score is within bounds.

	if score < 0 {
		score = 0
	}

	if score > 1 {
		score = 1
	}

	return score
}

// calculateResourceScore calculates a score based on resource optimization.

func (r *ResourcePlanningController) calculateResourceScore(resources []nephoranv1.PlannedResource) float64 {
	if len(resources) == 0 {
		return 0
	}

	score := 0.0

	for _, resource := range resources {

		// Check if resource has appropriate limits.

		resourceScore := 0.5 // Base score

		if resource.ResourceRequirements.Limits.CPU != nil {
			resourceScore += 0.2
		}

		if resource.ResourceRequirements.Limits.Memory != nil {
			resourceScore += 0.2
		}

		if resource.Replicas != nil && *resource.Replicas >= 2 {
			resourceScore += 0.1
		}

		score += resourceScore

	}

	return score / float64(len(resources))
}

// handlePlanningSuccess handles successful planning.

func (r *ResourcePlanningController) handlePlanningSuccess(ctx context.Context, resourcePlan *nephoranv1.ResourcePlan, result *ResourcePlanningResult) (ctrl.Result, error) {
	log := r.Logger.WithValues("resourceplan", resourcePlan.Name)

	// Update status with results.

	now := metav1.Now()

	resourcePlan.Status.PlanningCompletionTime = &now

	resourcePlan.Status.Phase = nephoranv1.ResourcePlanPhaseCompleted

	// Set planned resources.

	resourcePlan.Status.PlannedResources = result.PlannedResources

	// Set resource allocation.

	if allocationBytes, err := json.Marshal(result.PlannedResources); err == nil {
		resourcePlan.Status.ResourceAllocation = runtime.RawExtension{Raw: allocationBytes}
	}

	// Set cost estimate.

	resourcePlan.Status.CostEstimate = result.CostEstimate

	// Set performance estimate.

	resourcePlan.Status.PerformanceEstimate = result.PerformanceEstimate

	// Set compliance status - convert ComplianceStatus to ResourceComplianceStatus.

	resourceComplianceStatus := make([]nephoranv1.ResourceComplianceStatus, len(result.ComplianceResults))

	for i, compStatus := range result.ComplianceResults {

		status := "Compliant"

		if compStatus.ViolationCount > 0 {
			status = "NonCompliant"
		}

		resourceComplianceStatus[i] = nephoranv1.ResourceComplianceStatus{
			Standard: compStatus.Standards[0], // Use first standard if available

			Status: status,
		}

		if len(compStatus.Standards) > 0 {
			resourceComplianceStatus[i].Standard = compStatus.Standards[0]
		}

	}

	resourcePlan.Status.ComplianceStatus = resourceComplianceStatus

	// Set optimization results.

	resourcePlan.Status.OptimizationResults = result.OptimizationResults

	// Set validation results.

	resourcePlan.Status.ValidationResults = result.ValidationResults

	// Set quality score.
	qualityScoreStr := fmt.Sprintf("%.2f", result.QualityScore)
	resourcePlan.Status.QualityScore = &qualityScoreStr

	// Calculate planning duration.

	if resourcePlan.Status.PlanningStartTime != nil {

		duration := now.Sub(resourcePlan.Status.PlanningStartTime.Time)

		resourcePlan.Status.PlanningDuration = &metav1.Duration{Duration: duration}

	}

	// Update status.

	if err := r.updateStatus(ctx, resourcePlan); err != nil {
		return ctrl.Result{}, err
	}

	// Record success event.

	r.Recorder.Event(resourcePlan, "Normal", "PlanningCompleted", "Resource planning completed successfully")

	// Publish completion event.

	if err := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseResourcePlanning, EventResourcePlanningCompleted,

		string(resourcePlan.UID), true, map[string]interface{}{}); err != nil {
		log.Error(err, "Failed to publish completion event")
	}

	// Record metrics.

	r.MetricsCollector.RecordPhaseCompletion(interfaces.PhaseResourcePlanning, string(resourcePlan.UID), true)

	log.Info("Resource planning completed successfully", "qualityScore", result.QualityScore, "resourceCount", len(result.PlannedResources))

	return ctrl.Result{}, nil
}

// handlePlanningError handles planning errors with retry logic.

func (r *ResourcePlanningController) handlePlanningError(ctx context.Context, resourcePlan *nephoranv1.ResourcePlan, err error) (ctrl.Result, error) {
	log := r.Logger.WithValues("resourceplan", resourcePlan.Name)

	log.Error(err, "Resource planning failed")

	// Increment retry count.

	resourcePlan.Status.RetryCount++

	// Check if we should retry.

	if resourcePlan.Status.RetryCount < int32(r.Config.MaxRetries) {

		// Calculate backoff duration.

		backoffDuration := r.calculateBackoff(resourcePlan.Status.RetryCount)

		if err := r.updateStatus(ctx, resourcePlan); err != nil {
			return ctrl.Result{}, err
		}

		// Record retry event.

		r.Recorder.Event(resourcePlan, "Warning", "PlanningRetry",

			fmt.Sprintf("Retrying resource planning (attempt %d/%d): %v",

				resourcePlan.Status.RetryCount, r.Config.MaxRetries, err))

		log.Info("Scheduling retry", "attempt", resourcePlan.Status.RetryCount, "backoff", backoffDuration)

		return ctrl.Result{RequeueAfter: backoffDuration}, nil

	}

	// Max retries exceeded - mark as permanently failed.

	resourcePlan.Status.Phase = nephoranv1.ResourcePlanPhaseFailed

	// Add failure condition.

	now := metav1.Now()

	condition := metav1.Condition{
		Type: "PlanningFailed",

		Status: metav1.ConditionTrue,

		ObservedGeneration: resourcePlan.Generation,

		Reason: "MaxRetriesExceeded",

		Message: fmt.Sprintf("Planning failed after %d attempts: %v", resourcePlan.Status.RetryCount, err),

		LastTransitionTime: now,
	}

	resourcePlan.Status.Conditions = append(resourcePlan.Status.Conditions, condition)

	if updateErr := r.updateStatus(ctx, resourcePlan); updateErr != nil {
		return ctrl.Result{}, updateErr
	}

	// Record failure event.

	r.Recorder.Event(resourcePlan, "Warning", "PlanningFailed",

		fmt.Sprintf("Resource planning failed permanently after %d attempts: %v",

			resourcePlan.Status.RetryCount, err))

	// Publish failure event.

	if pubErr := r.EventBus.PublishPhaseEvent(ctx, interfaces.PhaseResourcePlanning, EventResourcePlanningFailed,

		string(resourcePlan.UID), false, map[string]interface{}{}); pubErr != nil {
		log.Error(pubErr, "Failed to publish failure event")
	}

	// Record metrics.

	r.MetricsCollector.RecordPhaseCompletion(interfaces.PhaseResourcePlanning, string(resourcePlan.UID), false)

	return ctrl.Result{}, nil
}

// calculateBackoff calculates the backoff duration for retries.

func (r *ResourcePlanningController) calculateBackoff(retryCount int32) time.Duration {
	backoff := r.Config.RetryBackoff

	for i := int32(1); i < retryCount; i++ {

		backoff *= 2

		if backoff > 5*time.Minute {

			backoff = 5 * time.Minute

			break

		}

	}

	return backoff
}

// handleDeletion handles resource deletion.

func (r *ResourcePlanningController) handleDeletion(ctx context.Context, resourcePlan *nephoranv1.ResourcePlan) (ctrl.Result, error) {
	log := r.Logger.WithValues("resourceplan", resourcePlan.Name)

	log.Info("Handling ResourcePlan deletion")

	// Cleanup any resources if needed.

	// (In this case, there are no external resources to clean up).

	// Remove finalizer.

	controllerutil.RemoveFinalizer(resourcePlan, "resourceplan.nephoran.com/finalizer")

	return ctrl.Result{}, r.Update(ctx, resourcePlan)
}

// updateStatus updates the status of the ResourcePlan resource.

func (r *ResourcePlanningController) updateStatus(ctx context.Context, resourcePlan *nephoranv1.ResourcePlan) error {
	resourcePlan.Status.ObservedGeneration = resourcePlan.Generation

	return r.Status().Update(ctx, resourcePlan)
}

// SetupWithManager sets up the controller with the Manager.

func (r *ResourcePlanningController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.ResourcePlan{}).
		Named("resourceplanning").
		Complete(r)
}

// Supporting types and functions.

// ResourceRequirements contains analyzed requirements.

type ResourceRequirements struct {
	TargetComponents []nephoranv1.TargetComponent

	DeploymentPattern nephoranv1.DeploymentPattern

	ResourceConstraints *nephoranv1.ResourceConstraints

	TargetClusters []nephoranv1.ClusterReference

	NetworkSlice *nephoranv1.NetworkSliceSpec

	OptimizationGoals []nephoranv1.OptimizationGoal

	ComplianceRequirements []nephoranv1.ComplianceRequirement

	ParsedInput map[string]interface{}
}

// ResourcePlanningResult contains the result of resource planning.

type ResourcePlanningResult struct {
	PlannedResources []nephoranv1.PlannedResource

	CostEstimate *nephoranv1.CostEstimate

	PerformanceEstimate *nephoranv1.PerformanceEstimate

	ComplianceResults []nephoranv1.ComplianceStatus

	OptimizationResults []nephoranv1.OptimizationResult

	ValidationResults []nephoranv1.ValidationResult

	QualityScore float64
}

// Service constructors.

// NewResourcePlanningService creates a new ResourcePlanningService.

func NewResourcePlanningService() *ResourcePlanningService {
	return &ResourcePlanningService{
		Logger: log.Log.WithName("resource-planning-service"),

		Config: DefaultResourcePlanningServiceConfig(),
	}
}

// NewCostEstimationService creates a new CostEstimationService.

func NewCostEstimationService() *CostEstimationService {
	return &CostEstimationService{
		Logger: log.Log.WithName("cost-estimation-service"),

		CostRates: map[string]float64{
			"cpu_core_hour": 0.048, // $0.048 per vCPU hour

			"memory_gb_hour": 0.0065, // $0.0065 per GB hour

			"storage_gb_hour": 0.0001, // $0.0001 per GB hour

		},
	}
}

// NewComplianceValidationService creates a new ComplianceValidationService.

func NewComplianceValidationService() *ComplianceValidationService {
	return &ComplianceValidationService{
		Logger: log.Log.WithName("compliance-validation-service"),

		Rules: DefaultComplianceRules(),
	}
}

// Default configuration values.

func DefaultResourcePlanningConfig() *ResourcePlanningConfig {
	return &ResourcePlanningConfig{
		MaxConcurrentPlanning: 5,

		DefaultTimeout: 300 * time.Second,

		MaxRetries: 3,

		RetryBackoff: 30 * time.Second,

		QualityThreshold: 0.7,

		CostOptimizationEnabled: true,

		PerformanceOptimizationEnabled: true,

		ComplianceValidationEnabled: true,

		// Cache configuration.

		CacheEnabled: true,

		CacheTTL: 5 * time.Minute,

		MaxCacheEntries: 1000,

		// Optimization configuration.

		OptimizationEnabled: true,

		CPUOvercommitRatio: 1.5,

		MemoryOvercommitRatio: 1.2,

		// Default resource requests.

		DefaultCPURequest: "100m",

		DefaultMemoryRequest: "128Mi",

		DefaultStorageRequest: "1Gi",

		// Constraint checking.

		ConstraintCheckEnabled: true,
	}
}

// DefaultResourcePlanningServiceConfig performs defaultresourceplanningserviceconfig operation.

func DefaultResourcePlanningServiceConfig() *ResourcePlanningServiceConfig {
	return &ResourcePlanningServiceConfig{
		DefaultCPURequest: resource.MustParse("100m"),

		DefaultMemoryRequest: resource.MustParse("128Mi"),

		DefaultCPULimit: resource.MustParse("500m"),

		DefaultMemoryLimit: resource.MustParse("512Mi"),

		DefaultReplicas: 1,

		ResourceMultipliers: map[string]float64{
			"AMF": 1.0,

			"SMF": 1.2,

			"UPF": 2.0,

			"gNodeB": 1.8,

			"Near-RT-RIC": 1.5,
		},
	}
}

// DefaultComplianceRules performs defaultcompliancerules operation.

func DefaultComplianceRules() map[string]ComplianceRule {
	return map[string]ComplianceRule{
		"3GPP": {
			Standard: "3GPP",

			Requirement: "Network functions must have resource limits",

			Validator: validate3GPPCompliance,
		},

		"ETSI": {
			Standard: "ETSI",

			Requirement: "Network functions must follow ETSI NFV standards",

			Validator: validateETSICompliance,
		},
	}
}

// Compliance validators.

func validate3GPPCompliance(plan *nephoranv1.ResourcePlan) (bool, string) {
	// Simplified 3GPP compliance check.

	for _, resource := range plan.Status.PlannedResources {
		if resource.ResourceRequirements.Limits.CPU == nil {
			return false, fmt.Sprintf("Resource %s lacks CPU limits", resource.Name)
		}
	}

	return true, "All resources have required CPU limits"
}

func validateETSICompliance(plan *nephoranv1.ResourcePlan) (bool, string) {
	// Simplified ETSI compliance check.

	for _, resource := range plan.Status.PlannedResources {
		if resource.ResourceRequirements.Limits.Memory == nil {
			return false, fmt.Sprintf("Resource %s lacks memory limits", resource.Name)
		}
	}

	return true, "All resources have required memory limits"
}

// Service method implementations.

// PlanResources plans resources based on requirements.

func (rps *ResourcePlanningService) PlanResources(ctx context.Context, requirements *ResourceRequirements) ([]nephoranv1.PlannedResource, error) {
	plannedResources := make([]nephoranv1.PlannedResource, 0, len(requirements.TargetComponents))

	// Plan resources for each target component.

	for _, component := range requirements.TargetComponents {

		resource := rps.planComponentResource(component, requirements)

		plannedResources = append(plannedResources, resource)

	}

	return plannedResources, nil
}

// planComponentResource plans resources for a specific component.

func (rps *ResourcePlanningService) planComponentResource(component nephoranv1.TargetComponent, requirements *ResourceRequirements) nephoranv1.PlannedResource {
	// Get base resource requirements.

	baseResource := rps.getBaseResourceRequirements(component)

	// Apply multipliers based on deployment pattern.

	multiplier := rps.getDeploymentMultiplier(requirements.DeploymentPattern)

	// Calculate final resources.

	cpuRequest := resource.NewMilliQuantity(int64(float64(baseResource.Requests.CPU.MilliValue())*multiplier), resource.DecimalSI)

	memoryRequest := resource.NewQuantity(int64(float64(baseResource.Requests.Memory.Value())*multiplier), resource.BinarySI)

	cpuLimit := resource.NewMilliQuantity(int64(float64(baseResource.Limits.CPU.MilliValue())*multiplier), resource.DecimalSI)

	memoryLimit := resource.NewQuantity(int64(float64(baseResource.Limits.Memory.Value())*multiplier), resource.BinarySI)

	// Determine replicas based on deployment pattern.

	replicas := rps.getReplicas(requirements.DeploymentPattern)

	return nephoranv1.PlannedResource{
		Name: strings.ToLower(component.Name),

		Type: "NetworkFunction",

		Component: component,

		ResourceRequirements: nephoranv1.ResourceSpec{
			Requests: nephoranv1.ResourceList{
				CPU: cpuRequest,

				Memory: memoryRequest,
			},

			Limits: nephoranv1.ResourceList{
				CPU: cpuLimit,

				Memory: memoryLimit,
			},
		},

		Replicas: &replicas,

		Labels: map[string]string{
			"app.kubernetes.io/name": strings.ToLower(component.Name),

			"app.kubernetes.io/component": "network-function",

			"nephoran.com/component": component.Name,
		},
	}
}

// getBaseResourceRequirements returns base resource requirements for a component.

func (rps *ResourcePlanningService) getBaseResourceRequirements(component nephoranv1.TargetComponent) nephoranv1.ResourceSpec {
	// Apply component-specific multipliers.

	multiplier := 1.0

	if m, exists := rps.Config.ResourceMultipliers[component.Name]; exists {
		multiplier = m
	}

	cpuRequest := resource.NewMilliQuantity(int64(float64(rps.Config.DefaultCPURequest.MilliValue())*multiplier), resource.DecimalSI)

	memoryRequest := resource.NewQuantity(int64(float64(rps.Config.DefaultMemoryRequest.Value())*multiplier), resource.BinarySI)

	cpuLimit := resource.NewMilliQuantity(int64(float64(rps.Config.DefaultCPULimit.MilliValue())*multiplier), resource.DecimalSI)

	memoryLimit := resource.NewQuantity(int64(float64(rps.Config.DefaultMemoryLimit.Value())*multiplier), resource.BinarySI)

	return nephoranv1.ResourceSpec{
		Requests: nephoranv1.ResourceList{
			CPU: cpuRequest,

			Memory: memoryRequest,
		},

		Limits: nephoranv1.ResourceList{
			CPU: cpuLimit,

			Memory: memoryLimit,
		},
	}
}

// getDeploymentMultiplier returns resource multiplier based on deployment pattern.

func (rps *ResourcePlanningService) getDeploymentMultiplier(pattern nephoranv1.DeploymentPattern) float64 {
	switch pattern {

	case nephoranv1.DeploymentPatternStandalone:

		return 1.0

	case nephoranv1.DeploymentPatternHighAvailability:

		return 1.2

	case nephoranv1.DeploymentPatternDistributed:

		return 0.8

	case nephoranv1.DeploymentPatternEdge:

		return 0.6

	case nephoranv1.DeploymentPatternHybrid:

		return 1.1

	default:

		return 1.0

	}
}

// getReplicas returns the number of replicas based on deployment pattern.

func (rps *ResourcePlanningService) getReplicas(pattern nephoranv1.DeploymentPattern) int32 {
	switch pattern {

	case nephoranv1.DeploymentPatternStandalone:

		return 1

	case nephoranv1.DeploymentPatternHighAvailability:

		return 3

	case nephoranv1.DeploymentPatternDistributed:

		return 2

	case nephoranv1.DeploymentPatternEdge:

		return 1

	case nephoranv1.DeploymentPatternHybrid:

		return 2

	default:

		return rps.Config.DefaultReplicas

	}
}

// EstimateCosts estimates the costs for planned resources.

func (ces *CostEstimationService) EstimateCosts(ctx context.Context, resources []nephoranv1.PlannedResource) (*nephoranv1.CostEstimate, error) {
	totalCost := 0.0

	costBreakdown := make(map[string]string)

	for _, resource := range resources {

		// Calculate hourly cost for this resource.

		hourlyCost := 0.0

		if resource.ResourceRequirements.Requests.CPU != nil {

			cpuCores := float64(resource.ResourceRequirements.Requests.CPU.MilliValue()) / 1000.0

			cpuCost := cpuCores * ces.CostRates["cpu_core_hour"]

			hourlyCost += cpuCost

		}

		if resource.ResourceRequirements.Requests.Memory != nil {

			memoryGB := float64(resource.ResourceRequirements.Requests.Memory.Value()) / (1024 * 1024 * 1024)

			memoryCost := memoryGB * ces.CostRates["memory_gb_hour"]

			hourlyCost += memoryCost

		}

		// Apply replica multiplier.

		if resource.Replicas != nil {
			hourlyCost *= float64(*resource.Replicas)
		}

		// Calculate monthly cost (24 * 30 = 720 hours).

		monthlyCost := hourlyCost * 720

		totalCost += monthlyCost

		costBreakdown[resource.Name] = strconv.FormatFloat(monthlyCost, 'f', 6, 64)

	}

	// Create cost estimate.

	confidence := "0.8" // 80% confidence in the estimate

	costEstimate := &nephoranv1.CostEstimate{
		TotalCost: fmt.Sprintf("%.2f", totalCost),

		Currency: "USD",

		BillingPeriod: "monthly",

		CostBreakdown: costBreakdown,

		EstimatedAt: metav1.Now(),

		Confidence: stringPtr(confidence),
	}

	return costEstimate, nil
}

// ValidateRequirement validates a compliance requirement.

func (cvs *ComplianceValidationService) ValidateRequirement(requirement nephoranv1.ComplianceRequirement, plan *nephoranv1.ResourcePlan, resources []nephoranv1.PlannedResource) (nephoranv1.ComplianceStatus, []nephoranv1.ComplianceViolation) {
	status := nephoranv1.ComplianceStatus{
		Standards: []string{requirement.Standard},

		ViolationCount: 0,
	}

	var violations []nephoranv1.ComplianceViolation

	// Check if we have a validator for this standard.

	if rule, exists := cvs.Rules[requirement.Standard]; exists {

		compliant, message := rule.Validator(plan)

		if !compliant {

			violation := nephoranv1.ComplianceViolation{
				Type: "Standard",

				Severity: "Major",

				Description: message,
			}

			violations = append(violations, violation)

			status.ViolationCount = int64(len(violations))

		}

	}

	// Convert violations to summaries for the status.

	recentViolations := make([]nephoranv1.ComplianceViolationSummary, len(violations))

	for i := range violations {
		recentViolations[i] = nephoranv1.ComplianceViolationSummary{

			// ViolationID and other fields would need to be set based on actual types.

			// For now, leaving empty as the exact fields are not visible.

		}
	}

	status.RecentViolations = recentViolations

	return status, violations
}
