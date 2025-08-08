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

package krm

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/errors"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// PorchIntegrationManager manages the integration between KRM functions and Porch
type PorchIntegrationManager struct {
	client            client.Client
	porchClient       porch.PorchClient
	functionManager   *FunctionManager
	pipelineOrch      *PipelineOrchestrator
	runtime           *Runtime
	config            *PorchIntegrationConfig
	metrics           *PorchIntegrationMetrics
	tracer            trace.Tracer
	intentCache       sync.Map
	packageCache      sync.Map
	mu                sync.RWMutex
}

// PorchIntegrationConfig defines configuration for Porch integration
type PorchIntegrationConfig struct {
	// Package creation settings
	DefaultRepository     string        `json:"defaultRepository" yaml:"defaultRepository"`
	DefaultNamespace      string        `json:"defaultNamespace" yaml:"defaultNamespace"`
	PackageTimeout        time.Duration `json:"packageTimeout" yaml:"packageTimeout"`
	RenderTimeout         time.Duration `json:"renderTimeout" yaml:"renderTimeout"`
	
	// Function execution settings
	EnablePipeline        bool          `json:"enablePipeline" yaml:"enablePipeline"`
	PipelineTimeout       time.Duration `json:"pipelineTimeout" yaml:"pipelineTimeout"`
	MaxConcurrentPackages int           `json:"maxConcurrentPackages" yaml:"maxConcurrentPackages"`
	
	// Cache settings
	EnableCaching         bool          `json:"enableCaching" yaml:"enableCaching"`
	CacheTTL             time.Duration `json:"cacheTtl" yaml:"cacheTtl"`
	MaxCacheSize         int           `json:"maxCacheSize" yaml:"maxCacheSize"`
	
	// Retry settings
	MaxRetries           int           `json:"maxRetries" yaml:"maxRetries"`
	RetryBackoff         time.Duration `json:"retryBackoff" yaml:"retryBackoff"`
	
	// Monitoring settings
	EnableMetrics        bool          `json:"enableMetrics" yaml:"enableMetrics"`
	EnableTracing        bool          `json:"enableTracing" yaml:"enableTracing"`
}

// PorchIntegrationMetrics provides comprehensive metrics for Porch integration
type PorchIntegrationMetrics struct {
	PackageCreations     prometheus.CounterVec
	PackageRevisions     prometheus.CounterVec
	FunctionExecutions   prometheus.CounterVec
	PipelineExecutions   prometheus.CounterVec
	ExecutionDuration    prometheus.HistogramVec
	PackageSize          prometheus.HistogramVec
	CacheHitRate         prometheus.Counter
	CacheMissRate        prometheus.Counter
	ErrorRate            prometheus.CounterVec
}

// FunctionEvalTask represents a function evaluation task for Porch
type FunctionEvalTask struct {
	ID               string                    `json:"id"`
	IntentID         string                    `json:"intentId"`
	PackageRef       *porch.PackageReference   `json:"packageRef"`
	FunctionPipeline *PipelineDefinition       `json:"functionPipeline"`
	Context          *FunctionEvalContext      `json:"context"`
	Status           FunctionEvalTaskStatus    `json:"status"`
	Results          *FunctionEvalResults      `json:"results,omitempty"`
	CreatedAt        time.Time                 `json:"createdAt"`
	UpdatedAt        time.Time                 `json:"updatedAt"`
	CompletedAt      *time.Time                `json:"completedAt,omitempty"`
}

// FunctionEvalContext provides context for function evaluation
type FunctionEvalContext struct {
	IntentSpec       *v1.NetworkIntentSpec     `json:"intentSpec"`
	TargetClusters   []*porch.ClusterTarget    `json:"targetClusters,omitempty"`
	ORANCompliance   *porch.ORANComplianceSpec `json:"oranCompliance,omitempty"`
	NetworkSlice     *porch.NetworkSliceSpec   `json:"networkSlice,omitempty"`
	Environment      map[string]string         `json:"environment,omitempty"`
	User             string                    `json:"user,omitempty"`
	Namespace        string                    `json:"namespace"`
}

// FunctionEvalResults contains the results of function evaluation
type FunctionEvalResults struct {
	PackageRevision    *porch.PackageRevision   `json:"packageRevision"`
	ValidationResults  []*porch.ValidationResult `json:"validationResults,omitempty"`
	RenderResults      *porch.RenderResult       `json:"renderResults,omitempty"`
	PipelineResults    *PipelineExecution        `json:"pipelineResults,omitempty"`
	DeploymentTargets  []*porch.DeploymentTarget `json:"deploymentTargets,omitempty"`
	GeneratedResources []porch.KRMResource       `json:"generatedResources"`
	AppliedFunctions   []string                  `json:"appliedFunctions"`
	Errors             []string                  `json:"errors,omitempty"`
}

// FunctionEvalTaskStatus represents the status of a function evaluation task
type FunctionEvalTaskStatus string

const (
	FunctionEvalTaskStatusPending    FunctionEvalTaskStatus = "Pending"
	FunctionEvalTaskStatusRunning    FunctionEvalTaskStatus = "Running"
	FunctionEvalTaskStatusCompleted  FunctionEvalTaskStatus = "Completed"
	FunctionEvalTaskStatusFailed     FunctionEvalTaskStatus = "Failed"
	FunctionEvalTaskStatusTimeout    FunctionEvalTaskStatus = "Timeout"
	FunctionEvalTaskStatusCancelled  FunctionEvalTaskStatus = "Cancelled"
)

// PackageRevisionLifecycleManager manages package revision lifecycle with KRM functions
type PackageRevisionLifecycleManager struct {
	integration *PorchIntegrationManager
	logger      logr.Logger
}

// IntentToPackageConverter converts NetworkIntent to Porch package structure
type IntentToPackageConverter struct {
	config        *PorchIntegrationConfig
	funcManager   *FunctionManager
	templateCache sync.Map
	tracer        trace.Tracer
}

// Default configuration
var DefaultPorchIntegrationConfig = &PorchIntegrationConfig{
	DefaultRepository:     "nephoran-packages",
	DefaultNamespace:      "nephoran-system",
	PackageTimeout:        10 * time.Minute,
	RenderTimeout:         5 * time.Minute,
	EnablePipeline:        true,
	PipelineTimeout:       15 * time.Minute,
	MaxConcurrentPackages: 20,
	EnableCaching:         true,
	CacheTTL:             1 * time.Hour,
	MaxCacheSize:         1000,
	MaxRetries:           3,
	RetryBackoff:         5 * time.Second,
	EnableMetrics:        true,
	EnableTracing:        true,
}

// NewPorchIntegrationManager creates a new Porch integration manager
func NewPorchIntegrationManager(
	client client.Client,
	porchClient porch.PorchClient,
	functionManager *FunctionManager,
	pipelineOrch *PipelineOrchestrator,
	runtime *Runtime,
	config *PorchIntegrationConfig,
) (*PorchIntegrationManager, error) {
	if config == nil {
		config = DefaultPorchIntegrationConfig
	}

	// Initialize metrics
	metrics := &PorchIntegrationMetrics{
		PackageCreations: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_porch_package_creations_total",
				Help: "Total number of package creations",
			},
			[]string{"repository", "intent_type", "status"},
		),
		PackageRevisions: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_porch_package_revisions_total",
				Help: "Total number of package revisions created",
			},
			[]string{"repository", "package", "lifecycle", "status"},
		),
		FunctionExecutions: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_porch_function_executions_total",
				Help: "Total number of KRM function executions in Porch context",
			},
			[]string{"function", "package", "status"},
		),
		PipelineExecutions: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_porch_pipeline_executions_total",
				Help: "Total number of KRM pipeline executions",
			},
			[]string{"pipeline", "package", "status"},
		),
		ExecutionDuration: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "krm_porch_execution_duration_seconds",
				Help:    "Duration of Porch integration operations",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
			},
			[]string{"operation", "intent_type"},
		),
		PackageSize: *promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "krm_porch_package_size_bytes",
				Help:    "Size of generated packages in bytes",
				Buckets: prometheus.ExponentialBuckets(1024, 2, 15), // 1KB to 32MB
			},
			[]string{"repository", "package"},
		),
		CacheHitRate: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "krm_porch_cache_hits_total",
				Help: "Total number of cache hits",
			},
		),
		CacheMissRate: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "krm_porch_cache_misses_total",
				Help: "Total number of cache misses",
			},
		),
		ErrorRate: *promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "krm_porch_errors_total",
				Help: "Total number of errors in Porch integration",
			},
			[]string{"operation", "error_type"},
		),
	}

	return &PorchIntegrationManager{
		client:          client,
		porchClient:     porchClient,
		functionManager: functionManager,
		pipelineOrch:    pipelineOrch,
		runtime:         runtime,
		config:          config,
		metrics:         metrics,
		tracer:          otel.Tracer("krm-porch-integration"),
	}, nil
}

// ProcessNetworkIntent processes a NetworkIntent and creates corresponding Porch packages
func (pim *PorchIntegrationManager) ProcessNetworkIntent(ctx context.Context, intent *v1.NetworkIntent) (*FunctionEvalTask, error) {
	ctx, span := pim.tracer.Start(ctx, "process-network-intent")
	defer span.End()

	logger := log.FromContext(ctx).WithName("porch-integration").WithValues(
		"intent", intent.Name,
		"namespace", intent.Namespace,
	)

	span.SetAttributes(
		attribute.String("intent.name", intent.Name),
		attribute.String("intent.namespace", intent.Namespace),
		attribute.String("intent.type", string(intent.Spec.IntentType)),
	)

	startTime := time.Now()

	// Create function evaluation task
	task := &FunctionEvalTask{
		ID:       generateTaskID(),
		IntentID: string(intent.UID),
		Context: &FunctionEvalContext{
			IntentSpec:   &intent.Spec,
			Environment:  make(map[string]string),
			User:         extractUserFromContext(ctx),
			Namespace:    intent.Namespace,
		},
		Status:    FunctionEvalTaskStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	// Extract O-RAN compliance and network slice from intent extensions
	if intent.Spec.Extensions != nil {
		if extensions, ok := intent.Spec.Extensions.(*porch.NetworkIntentExtensions); ok {
			task.Context.ORANCompliance = extensions.ORANCompliance
			task.Context.NetworkSlice = extensions.NetworkSlice
			task.Context.TargetClusters = extensions.ClusterTargets
		}
	}

	// Store in cache
	pim.intentCache.Store(task.ID, task)

	logger.Info("Created function evaluation task", "taskId", task.ID)

	// Process intent asynchronously
	go func() {
		defer func() {
			duration := time.Since(startTime)
			pim.metrics.ExecutionDuration.WithLabelValues(
				"process_intent", string(intent.Spec.IntentType),
			).Observe(duration.Seconds())
		}()

		if err := pim.processIntentTask(ctx, task, intent); err != nil {
			logger.Error(err, "Failed to process intent task", "taskId", task.ID)
			task.Status = FunctionEvalTaskStatusFailed
			if task.Results == nil {
				task.Results = &FunctionEvalResults{}
			}
			task.Results.Errors = append(task.Results.Errors, err.Error())
			pim.metrics.ErrorRate.WithLabelValues("process_intent", "task_failed").Inc()
		}

		task.UpdatedAt = time.Now()
		if task.Status == FunctionEvalTaskStatusCompleted || task.Status == FunctionEvalTaskStatusFailed {
			completedAt := time.Now()
			task.CompletedAt = &completedAt
		}

		// Update cache
		pim.intentCache.Store(task.ID, task)
	}()

	span.SetStatus(codes.Ok, "task created successfully")
	return task, nil
}

// processIntentTask processes an intent task end-to-end
func (pim *PorchIntegrationManager) processIntentTask(ctx context.Context, task *FunctionEvalTask, intent *v1.NetworkIntent) error {
	ctx, span := pim.tracer.Start(ctx, "process-intent-task")
	defer span.End()

	logger := log.FromContext(ctx).WithName("porch-integration").WithValues("taskId", task.ID)

	task.Status = FunctionEvalTaskStatusRunning
	task.UpdatedAt = time.Now()

	// Step 1: Convert intent to package specification
	packageSpec, err := pim.convertIntentToPackageSpec(ctx, intent)
	if err != nil {
		span.RecordError(err)
		return errors.WithContext(err, "failed to convert intent to package spec")
	}

	// Step 2: Create initial package revision
	packageRevision, err := pim.createPackageRevision(ctx, packageSpec)
	if err != nil {
		span.RecordError(err)
		return errors.WithContext(err, "failed to create package revision")
	}

	task.PackageRef = &porch.PackageReference{
		Repository:  packageRevision.Spec.Repository,
		PackageName: packageRevision.Spec.PackageName,
		Revision:    packageRevision.Spec.Revision,
	}

	logger.Info("Created package revision", 
		"repository", packageRevision.Spec.Repository,
		"package", packageRevision.Spec.PackageName,
		"revision", packageRevision.Spec.Revision,
	)

	// Step 3: Build function pipeline based on intent type
	pipeline, err := pim.buildFunctionPipeline(ctx, intent, packageRevision)
	if err != nil {
		span.RecordError(err)
		return errors.WithContext(err, "failed to build function pipeline")
	}

	task.FunctionPipeline = pipeline

	// Step 4: Execute function pipeline
	if pim.config.EnablePipeline && pipeline != nil {
		pipelineCtx, cancel := context.WithTimeout(ctx, pim.config.PipelineTimeout)
		defer cancel()

		pipelineExecution, err := pim.executeFunctionPipeline(pipelineCtx, task, packageRevision)
		if err != nil {
			span.RecordError(err)
			return errors.WithContext(err, "failed to execute function pipeline")
		}

		if task.Results == nil {
			task.Results = &FunctionEvalResults{}
		}
		task.Results.PipelineResults = pipelineExecution
		task.Results.GeneratedResources = pipelineExecution.FinalResources
		task.Results.AppliedFunctions = pipelineExecution.ExecutedStages

		// Update package revision with pipeline results
		packageRevision, err = pim.updatePackageWithResults(ctx, packageRevision, pipelineExecution)
		if err != nil {
			span.RecordError(err)
			return errors.WithContext(err, "failed to update package with pipeline results")
		}
	}

	// Step 5: Validate package
	validationResults, err := pim.validatePackage(ctx, packageRevision)
	if err != nil {
		logger.Error(err, "Package validation failed", "package", packageRevision.Name)
		// Continue processing even if validation fails
	}

	// Step 6: Render package
	renderResults, err := pim.renderPackage(ctx, packageRevision)
	if err != nil {
		span.RecordError(err)
		return errors.WithContext(err, "failed to render package")
	}

	// Step 7: Update task results
	if task.Results == nil {
		task.Results = &FunctionEvalResults{}
	}

	task.Results.PackageRevision = packageRevision
	task.Results.ValidationResults = validationResults
	task.Results.RenderResults = renderResults

	// Step 8: Determine deployment targets
	if task.Context.TargetClusters != nil {
		deploymentTargets := make([]*porch.DeploymentTarget, 0, len(task.Context.TargetClusters))
		for _, cluster := range task.Context.TargetClusters {
			deploymentTargets = append(deploymentTargets, &porch.DeploymentTarget{
				Cluster:   cluster.Name,
				Namespace: cluster.Namespace,
				Status:    "Pending",
			})
		}
		task.Results.DeploymentTargets = deploymentTargets
	}

	// Step 9: Update NetworkIntent status
	if err := pim.updateIntentStatus(ctx, intent, task); err != nil {
		logger.Error(err, "Failed to update intent status", "intent", intent.Name)
		// Don't fail the task for status update errors
	}

	task.Status = FunctionEvalTaskStatusCompleted
	task.UpdatedAt = time.Now()

	pim.metrics.PackageCreations.WithLabelValues(
		packageRevision.Spec.Repository,
		string(intent.Spec.IntentType),
		"success",
	).Inc()

	logger.Info("Successfully completed intent processing", "taskId", task.ID)
	span.SetStatus(codes.Ok, "intent task completed successfully")
	
	return nil
}

// convertIntentToPackageSpec converts a NetworkIntent to a PackageSpec
func (pim *PorchIntegrationManager) convertIntentToPackageSpec(ctx context.Context, intent *v1.NetworkIntent) (*porch.PackageSpec, error) {
	ctx, span := pim.tracer.Start(ctx, "convert-intent-to-package-spec")
	defer span.End()

	// Generate package name based on intent
	packageName := fmt.Sprintf("%s-%s", intent.Name, string(intent.Spec.IntentType))
	if len(packageName) > 63 {
		packageName = packageName[:63] // Kubernetes name limit
	}

	// Determine repository
	repository := pim.config.DefaultRepository
	if intent.Spec.Extensions != nil {
		if extensions, ok := intent.Spec.Extensions.(*porch.NetworkIntentExtensions); ok {
			if extensions.PackageSpec != nil && extensions.PackageSpec.Repository != "" {
				repository = extensions.PackageSpec.Repository
			}
		}
	}

	// Create labels and annotations
	labels := map[string]string{
		porch.LabelComponent:       "nephoran-intent-operator",
		porch.LabelRepository:      repository,
		porch.LabelPackageName:     packageName,
		porch.LabelIntentType:      string(intent.Spec.IntentType),
		porch.LabelTargetComponent: intent.Spec.TargetComponent,
	}

	annotations := map[string]string{
		porch.AnnotationManagedBy:  "nephoran-intent-operator",
		porch.AnnotationIntentID:   string(intent.UID),
		porch.AnnotationRepository: repository,
		porch.AnnotationPackageName: packageName,
	}

	// Add network slice information if available
	if intent.Spec.Extensions != nil {
		if extensions, ok := intent.Spec.Extensions.(*porch.NetworkIntentExtensions); ok {
			if extensions.NetworkSlice != nil {
				labels[porch.LabelNetworkSlice] = extensions.NetworkSlice.SliceID
			}
			if extensions.ORANCompliance != nil {
				annotations[porch.AnnotationORANCompliance] = "enabled"
				// Add O-RAN interface labels
				for _, iface := range extensions.ORANCompliance.Interfaces {
					if iface.Enabled {
						labels[fmt.Sprintf("%s/%s", porch.LabelORANInterface, iface.Name)] = iface.Type
					}
				}
			}
		}
	}

	packageSpec := &porch.PackageSpec{
		Repository:  repository,
		PackageName: packageName,
		Revision:    "v1", // Start with v1, increment for updates
		Lifecycle:   porch.PackageRevisionLifecycleDraft,
		Labels:      labels,
		Annotations: annotations,
	}

	span.SetAttributes(
		attribute.String("package.name", packageName),
		attribute.String("package.repository", repository),
	)

	return packageSpec, nil
}

// createPackageRevision creates a new package revision in Porch
func (pim *PorchIntegrationManager) createPackageRevision(ctx context.Context, spec *porch.PackageSpec) (*porch.PackageRevision, error) {
	ctx, span := pim.tracer.Start(ctx, "create-package-revision")
	defer span.End()

	// Create package revision structure
	packageRevision := &porch.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "porch.nephoran.com/v1",
			Kind:       "PackageRevision",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s", spec.PackageName, spec.Revision),
			Namespace:   pim.config.DefaultNamespace,
			Labels:      spec.Labels,
			Annotations: spec.Annotations,
		},
		Spec: porch.PackageRevisionSpec{
			PackageName: spec.PackageName,
			Repository:  spec.Repository,
			Revision:    spec.Revision,
			Lifecycle:   spec.Lifecycle,
			Resources:   []porch.KRMResource{}, // Will be populated by functions
			Functions:   []porch.FunctionConfig{}, // Will be populated by pipeline
		},
	}

	// Create in Porch
	createdPackage, err := pim.porchClient.CreatePackageRevision(ctx, packageRevision)
	if err != nil {
		span.RecordError(err)
		pim.metrics.PackageRevisions.WithLabelValues(
			spec.Repository, spec.PackageName, string(spec.Lifecycle), "failed",
		).Inc()
		return nil, errors.WithContext(err, "failed to create package revision in Porch")
	}

	pim.metrics.PackageRevisions.WithLabelValues(
		spec.Repository, spec.PackageName, string(spec.Lifecycle), "success",
	).Inc()

	span.SetAttributes(
		attribute.String("package.id", createdPackage.Name),
		attribute.String("package.revision", createdPackage.Spec.Revision),
	)

	return createdPackage, nil
}

// buildFunctionPipeline builds a function pipeline based on intent type and requirements
func (pim *PorchIntegrationManager) buildFunctionPipeline(ctx context.Context, intent *v1.NetworkIntent, packageRevision *porch.PackageRevision) (*PipelineDefinition, error) {
	ctx, span := pim.tracer.Start(ctx, "build-function-pipeline")
	defer span.End()

	pipeline := &PipelineDefinition{
		Name:        fmt.Sprintf("%s-pipeline", packageRevision.Spec.PackageName),
		Description: fmt.Sprintf("Function pipeline for %s intent", intent.Spec.IntentType),
		Stages:      make([]PipelineStageDefinition, 0),
		Execution:   PipelineExecutionModeDAG,
	}

	// Stage 1: O-RAN Compliance Validation (if required)
	if intent.Spec.Extensions != nil {
		if extensions, ok := intent.Spec.Extensions.(*porch.NetworkIntentExtensions); ok {
			if extensions.ORANCompliance != nil {
				oranStage := PipelineStageDefinition{
					Name:        "oran-compliance-validation",
					Description: "Validate O-RAN compliance requirements",
					Functions: []PipelineFunctionDefinition{
						{
							Name:     "oran-compliance-validator",
							Function: "oran-compliance-validator", // References our O-RAN validator function
							Config: map[string]interface{}{
								"interfaces":     extensions.ORANCompliance.Interfaces,
								"validations":    extensions.ORANCompliance.Validations,
								"certifications": extensions.ORANCompliance.Certifications,
								"standards":      extensions.ORANCompliance.Standards,
							},
							Required: true,
						},
					},
					Dependencies: []string{}, // First stage
					Parallel:     false,
					Timeout:      5 * time.Minute,
				}
				pipeline.Stages = append(pipeline.Stages, oranStage)
			}
		}
	}

	// Stage 2: Intent Type Specific Processing
	intentStage := pim.buildIntentSpecificStage(ctx, intent)
	if intentStage != nil {
		pipeline.Stages = append(pipeline.Stages, *intentStage)
	}

	// Stage 3: Network Slice Optimization (if applicable)
	if intent.Spec.Extensions != nil {
		if extensions, ok := intent.Spec.Extensions.(*porch.NetworkIntentExtensions); ok {
			if extensions.NetworkSlice != nil {
				sliceStage := PipelineStageDefinition{
					Name:        "network-slice-optimization",
					Description: "Optimize network slice configuration",
					Functions: []PipelineFunctionDefinition{
						{
							Name:     "network-slice-optimizer",
							Function: "network-slice-optimizer",
							Config: map[string]interface{}{
								"sliceId":   extensions.NetworkSlice.SliceID,
								"sliceType": extensions.NetworkSlice.SliceType,
								"sla":       extensions.NetworkSlice.SLA,
								"qos":       extensions.NetworkSlice.QoS,
								"resources": extensions.NetworkSlice.Resources,
							},
							Required: true,
						},
					},
					Dependencies: []string{"oran-compliance-validation"},
					Parallel:     false,
					Timeout:      10 * time.Minute,
				}
				pipeline.Stages = append(pipeline.Stages, sliceStage)
			}
		}
	}

	// Stage 4: Multi-Vendor Configuration Normalization
	normalizationStage := PipelineStageDefinition{
		Name:        "multi-vendor-normalization",
		Description: "Normalize configurations for multi-vendor compatibility",
		Functions: []PipelineFunctionDefinition{
			{
				Name:     "multi-vendor-normalizer",
				Function: "multi-vendor-normalizer",
				Config: map[string]interface{}{
					"intentType":      string(intent.Spec.IntentType),
					"targetComponent": intent.Spec.TargetComponent,
				},
				Required: false, // Optional optimization
			},
		},
		Dependencies: []string{"network-slice-optimization"},
		Parallel:     false,
		Timeout:      5 * time.Minute,
	}
	pipeline.Stages = append(pipeline.Stages, normalizationStage)

	// Final Stage: 5G Core Validation
	if pim.is5GCoreIntent(intent) {
		coreStage := PipelineStageDefinition{
			Name:        "5g-core-validation",
			Description: "Validate 5G Core network function configurations",
			Functions: []PipelineFunctionDefinition{
				{
					Name:     "5g-core-validator",
					Function: "5g-core-validator",
					Config: map[string]interface{}{
						"intentType":      string(intent.Spec.IntentType),
						"targetComponent": intent.Spec.TargetComponent,
						"strictMode":      true,
					},
					Required: true,
				},
			},
			Dependencies: []string{"multi-vendor-normalization"},
			Parallel:     false,
			Timeout:      5 * time.Minute,
		}
		pipeline.Stages = append(pipeline.Stages, coreStage)
	}

	span.SetAttributes(
		attribute.Int("pipeline.stages", len(pipeline.Stages)),
		attribute.String("pipeline.execution", string(pipeline.Execution)),
	)

	return pipeline, nil
}

// buildIntentSpecificStage builds a pipeline stage specific to the intent type
func (pim *PorchIntegrationManager) buildIntentSpecificStage(ctx context.Context, intent *v1.NetworkIntent) *PipelineStageDefinition {
	switch intent.Spec.IntentType {
	case v1.NetworkIntentTypeDeployment:
		return &PipelineStageDefinition{
			Name:        "deployment-configuration",
			Description: "Configure deployment-specific parameters",
			Functions: []PipelineFunctionDefinition{
				{
					Name:     "deployment-config-generator",
					Function: "5g-core-optimizer", // Use 5G optimizer for deployments
					Config: map[string]interface{}{
						"deploymentType":  "production",
						"scalingPolicy":   "auto",
						"resourceLimits":  intent.Spec.Configuration,
						"targetComponent": intent.Spec.TargetComponent,
					},
					Required: true,
				},
			},
			Dependencies: []string{"oran-compliance-validation"},
			Parallel:     false,
			Timeout:      10 * time.Minute,
		}

	case v1.NetworkIntentTypeConfiguration:
		return &PipelineStageDefinition{
			Name:        "configuration-validation",
			Description: "Validate and optimize configuration parameters",
			Functions: []PipelineFunctionDefinition{
				{
					Name:     "config-validator",
					Function: "5g-core-validator", // Use 5G validator for configuration
					Config: map[string]interface{}{
						"configType":      "network-function",
						"targetComponent": intent.Spec.TargetComponent,
						"parameters":      intent.Spec.Configuration,
					},
					Required: true,
				},
			},
			Dependencies: []string{"oran-compliance-validation"},
			Parallel:     false,
			Timeout:      5 * time.Minute,
		}

	case v1.NetworkIntentTypeScaling:
		return &PipelineStageDefinition{
			Name:        "scaling-optimization",
			Description: "Optimize scaling configuration and resource allocation",
			Functions: []PipelineFunctionDefinition{
				{
					Name:     "scaling-optimizer",
					Function: "5g-core-optimizer", // Use 5G optimizer for scaling
					Config: map[string]interface{}{
						"scalingType":     "horizontal",
						"targetComponent": intent.Spec.TargetComponent,
						"metrics":         intent.Spec.Configuration,
					},
					Required: true,
				},
			},
			Dependencies: []string{"oran-compliance-validation"},
			Parallel:     false,
			Timeout:      10 * time.Minute,
		}

	default:
		// Generic configuration stage for unknown intent types
		return &PipelineStageDefinition{
			Name:        "generic-configuration",
			Description: "Generic configuration processing",
			Functions: []PipelineFunctionDefinition{
				{
					Name:     "generic-processor",
					Function: "5g-core-validator", // Use validator as default
					Config: map[string]interface{}{
						"intentType":      string(intent.Spec.IntentType),
						"targetComponent": intent.Spec.TargetComponent,
						"configuration":   intent.Spec.Configuration,
					},
					Required: false,
				},
			},
			Dependencies: []string{"oran-compliance-validation"},
			Parallel:     false,
			Timeout:      5 * time.Minute,
		}
	}
}

// executeFunctionPipeline executes the function pipeline using the pipeline orchestrator
func (pim *PorchIntegrationManager) executeFunctionPipeline(ctx context.Context, task *FunctionEvalTask, packageRevision *porch.PackageRevision) (*PipelineExecution, error) {
	ctx, span := pim.tracer.Start(ctx, "execute-function-pipeline")
	defer span.End()

	logger := log.FromContext(ctx).WithName("pipeline-executor").WithValues("taskId", task.ID)

	// Convert Porch resources to KRM resources for pipeline execution
	resources := make([]*porch.KRMResource, 0)
	for _, resource := range packageRevision.Spec.Resources {
		resources = append(resources, &resource)
	}

	// Execute pipeline
	execution, err := pim.pipelineOrch.ExecutePipeline(ctx, task.FunctionPipeline, resources)
	if err != nil {
		span.RecordError(err)
		pim.metrics.PipelineExecutions.WithLabelValues(
			task.FunctionPipeline.Name,
			packageRevision.Spec.PackageName,
			"failed",
		).Inc()
		return nil, errors.WithContext(err, "pipeline execution failed")
	}

	// Record metrics for each function execution
	for _, stage := range execution.ExecutedStages {
		pim.metrics.FunctionExecutions.WithLabelValues(
			stage, packageRevision.Spec.PackageName, "success",
		).Inc()
	}

	pim.metrics.PipelineExecutions.WithLabelValues(
		task.FunctionPipeline.Name,
		packageRevision.Spec.PackageName,
		"success",
	).Inc()

	logger.Info("Pipeline execution completed",
		"stages", len(execution.ExecutedStages),
		"resources", len(execution.FinalResources),
		"duration", execution.Duration,
	)

	span.SetAttributes(
		attribute.Int("pipeline.stages", len(execution.ExecutedStages)),
		attribute.Int("pipeline.resources", len(execution.FinalResources)),
		attribute.String("pipeline.status", string(execution.Status)),
	)

	return execution, nil
}

// updatePackageWithResults updates the package revision with pipeline results
func (pim *PorchIntegrationManager) updatePackageWithResults(ctx context.Context, packageRevision *porch.PackageRevision, execution *PipelineExecution) (*porch.PackageRevision, error) {
	ctx, span := pim.tracer.Start(ctx, "update-package-with-results")
	defer span.End()

	// Convert pipeline resources back to package resources
	updatedResources := make([]porch.KRMResource, 0)
	for _, resource := range execution.FinalResources {
		updatedResources = append(updatedResources, *resource)
	}

	// Update package revision spec
	packageRevision.Spec.Resources = updatedResources

	// Add function configurations from pipeline
	for _, stage := range execution.ExecutedStages {
		functionConfig := porch.FunctionConfig{
			Image: fmt.Sprintf("krm/%s:latest", stage), // Assuming standard naming
			ConfigMap: map[string]interface{}{
				"executedAt": time.Now().Format(time.RFC3339),
				"status":     "completed",
			},
		}
		packageRevision.Spec.Functions = append(packageRevision.Spec.Functions, functionConfig)
	}

	// Update package revision in Porch
	updatedPackage, err := pim.porchClient.UpdatePackageRevision(ctx, packageRevision)
	if err != nil {
		span.RecordError(err)
		return nil, errors.WithContext(err, "failed to update package revision")
	}

	// Record package size metric
	packageSize := pim.calculatePackageSize(updatedPackage)
	pim.metrics.PackageSize.WithLabelValues(
		updatedPackage.Spec.Repository,
		updatedPackage.Spec.PackageName,
	).Observe(float64(packageSize))

	return updatedPackage, nil
}

// validatePackage validates the package using Porch validation
func (pim *PorchIntegrationManager) validatePackage(ctx context.Context, packageRevision *porch.PackageRevision) ([]*porch.ValidationResult, error) {
	ctx, span := pim.tracer.Start(ctx, "validate-package")
	defer span.End()

	result, err := pim.porchClient.ValidatePackage(ctx, packageRevision.Spec.PackageName, packageRevision.Spec.Revision)
	if err != nil {
		span.RecordError(err)
		return nil, errors.WithContext(err, "package validation failed")
	}

	// Convert single result to slice for consistency
	results := []*porch.ValidationResult{result}

	span.SetAttributes(
		attribute.Bool("validation.valid", result.Valid),
		attribute.Int("validation.errors", len(result.Errors)),
		attribute.Int("validation.warnings", len(result.Warnings)),
	)

	return results, nil
}

// renderPackage renders the package using Porch rendering
func (pim *PorchIntegrationManager) renderPackage(ctx context.Context, packageRevision *porch.PackageRevision) (*porch.RenderResult, error) {
	ctx, span := pim.tracer.Start(ctx, "render-package")
	defer span.End()

	result, err := pim.porchClient.RenderPackage(ctx, packageRevision.Spec.PackageName, packageRevision.Spec.Revision)
	if err != nil {
		span.RecordError(err)
		return nil, errors.WithContext(err, "package rendering failed")
	}

	span.SetAttributes(
		attribute.Int("render.resources", len(result.Resources)),
		attribute.Int("render.results", len(result.Results)),
	)

	return result, nil
}

// updateIntentStatus updates the NetworkIntent status with processing results
func (pim *PorchIntegrationManager) updateIntentStatus(ctx context.Context, intent *v1.NetworkIntent, task *FunctionEvalTask) error {
	ctx, span := pim.tracer.Start(ctx, "update-intent-status")
	defer span.End()

	// Update intent status based on task results
	intent.Status.Phase = v1.NetworkIntentPhaseProcessing

	if task.Results != nil && task.Results.PackageRevision != nil {
		// Set package reference in status
		intent.Status.PackageRevision = &v1.PackageRevisionReference{
			Repository:  task.Results.PackageRevision.Spec.Repository,
			PackageName: task.Results.PackageRevision.Spec.PackageName,
			Revision:    task.Results.PackageRevision.Spec.Revision,
		}

		// Update deployment status
		if task.Results.DeploymentTargets != nil {
			intent.Status.DeploymentStatus = &v1.DeploymentStatus{
				Phase: "Pending",
				Targets: make([]v1.DeploymentTargetStatus, 0, len(task.Results.DeploymentTargets)),
			}

			for _, target := range task.Results.DeploymentTargets {
				intent.Status.DeploymentStatus.Targets = append(intent.Status.DeploymentStatus.Targets, v1.DeploymentTargetStatus{
					Cluster:   target.Cluster,
					Namespace: target.Namespace,
					Status:    target.Status,
				})
			}
		}

		// Set completion status
		if task.Status == FunctionEvalTaskStatusCompleted {
			intent.Status.Phase = v1.NetworkIntentPhaseReady
		} else if task.Status == FunctionEvalTaskStatusFailed {
			intent.Status.Phase = v1.NetworkIntentPhaseFailed
			if len(task.Results.Errors) > 0 {
				intent.Status.Message = task.Results.Errors[0] // First error as message
			}
		}
	}

	// Update last processed timestamp
	now := metav1.NewTime(time.Now())
	intent.Status.LastProcessed = &now

	// Update intent in cluster
	if err := pim.client.Status().Update(ctx, intent); err != nil {
		span.RecordError(err)
		return errors.WithContext(err, "failed to update intent status")
	}

	return nil
}

// GetFunctionEvalTask retrieves a function evaluation task by ID
func (pim *PorchIntegrationManager) GetFunctionEvalTask(ctx context.Context, taskID string) (*FunctionEvalTask, error) {
	if value, ok := pim.intentCache.Load(taskID); ok {
		if task, ok := value.(*FunctionEvalTask); ok {
			return task, nil
		}
	}
	return nil, fmt.Errorf("task not found: %s", taskID)
}

// ListFunctionEvalTasks lists all function evaluation tasks
func (pim *PorchIntegrationManager) ListFunctionEvalTasks(ctx context.Context) ([]*FunctionEvalTask, error) {
	tasks := make([]*FunctionEvalTask, 0)
	pim.intentCache.Range(func(key, value interface{}) bool {
		if task, ok := value.(*FunctionEvalTask); ok {
			tasks = append(tasks, task)
		}
		return true
	})
	return tasks, nil
}

// CancelFunctionEvalTask cancels a running function evaluation task
func (pim *PorchIntegrationManager) CancelFunctionEvalTask(ctx context.Context, taskID string) error {
	if value, ok := pim.intentCache.Load(taskID); ok {
		if task, ok := value.(*FunctionEvalTask); ok {
			if task.Status == FunctionEvalTaskStatusRunning || task.Status == FunctionEvalTaskStatusPending {
				task.Status = FunctionEvalTaskStatusCancelled
				task.UpdatedAt = time.Now()
				completedAt := time.Now()
				task.CompletedAt = &completedAt
				pim.intentCache.Store(taskID, task)
				return nil
			}
		}
	}
	return fmt.Errorf("task not found or cannot be cancelled: %s", taskID)
}

// Cleanup performs cleanup of expired cache entries
func (pim *PorchIntegrationManager) Cleanup(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("porch-integration-cleanup")
	
	cleaned := 0
	cutoff := time.Now().Add(-pim.config.CacheTTL)

	// Clean intent cache
	pim.intentCache.Range(func(key, value interface{}) bool {
		if task, ok := value.(*FunctionEvalTask); ok {
			if task.CompletedAt != nil && task.CompletedAt.Before(cutoff) {
				pim.intentCache.Delete(key)
				cleaned++
			}
		}
		return true
	})

	// Clean package cache
	pim.packageCache.Range(func(key, value interface{}) bool {
		if entry, ok := value.(map[string]interface{}); ok {
			if timestamp, ok := entry["timestamp"].(time.Time); ok {
				if timestamp.Before(cutoff) {
					pim.packageCache.Delete(key)
					cleaned++
				}
			}
		}
		return true
	})

	logger.Info("Cleanup completed", "cleaned", cleaned)
	return nil
}

// Helper functions

func generateTaskID() string {
	return fmt.Sprintf("task-%d", time.Now().UnixNano())
}

func extractUserFromContext(ctx context.Context) string {
	// Extract user from context (implementation depends on auth setup)
	// This is a placeholder - actual implementation would extract from JWT or similar
	return "system"
}

func (pim *PorchIntegrationManager) is5GCoreIntent(intent *v1.NetworkIntent) bool {
	// Check if intent targets 5G Core components
	coreComponents := []string{"amf", "smf", "upf", "nssf", "nrf", "udm", "ausf", "pcf"}
	for _, component := range coreComponents {
		if intent.Spec.TargetComponent == component {
			return true
		}
	}
	return false
}

func (pim *PorchIntegrationManager) calculatePackageSize(packageRevision *porch.PackageRevision) int64 {
	// Calculate package size in bytes (simplified calculation)
	size := int64(0)
	for _, resource := range packageRevision.Spec.Resources {
		resourceJSON, _ := json.Marshal(resource)
		size += int64(len(resourceJSON))
	}
	return size
}