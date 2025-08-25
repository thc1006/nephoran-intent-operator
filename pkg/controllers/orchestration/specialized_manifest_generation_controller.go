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
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
)

// SpecializedManifestGenerationController handles Kubernetes manifest generation
type SpecializedManifestGenerationController struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	Logger   logr.Logger

	// Manifest generation services
	TemplateEngine       *KubernetesTemplateEngine
	ManifestValidator    *ManifestValidator
	PolicyEnforcer       *ManifestPolicyEnforcer
	ManifestOptimizer    *ManifestOptimizer
	HelmIntegration      *HelmManifestIntegration
	KustomizeIntegration *KustomizeManifestIntegration

	// Configuration
	Config      ManifestGenerationConfig
	Templates   map[string]*ManifestTemplate
	PolicyRules []*ManifestPolicyRule

	// Internal state
	activeGeneration sync.Map // map[string]*GenerationSession
	manifestCache    *ManifestCache
	metrics          *ManifestGenerationMetrics

	// Health and lifecycle
	started      bool
	stopChan     chan struct{}
	healthStatus interfaces.HealthStatus
	mutex        sync.RWMutex
}

// ManifestGenerationConfig holds configuration for manifest generation
type ManifestGenerationConfig struct {
	// Template settings
	TemplateDirectory string `json:"templateDirectory"`
	DefaultNamespace  string `json:"defaultNamespace"`
	EnableHelm        bool   `json:"enableHelm"`
	EnableKustomize   bool   `json:"enableKustomize"`

	// Validation settings
	ValidateManifests bool `json:"validateManifests"`
	DryRunValidation  bool `json:"dryRunValidation"`
	SchemaValidation  bool `json:"schemaValidation"`

	// Optimization settings
	OptimizeManifests bool `json:"optimizeManifests"`
	MinifyManifests   bool `json:"minifyManifests"`
	RemoveDuplicates  bool `json:"removeDuplicates"`

	// Policy enforcement
	EnforcePolicies  bool `json:"enforcePolicies"`
	SecurityPolicies bool `json:"securityPolicies"`
	ResourcePolicies bool `json:"resourcePolicies"`

	// Cache configuration
	CacheEnabled    bool          `json:"cacheEnabled"`
	CacheTTL        time.Duration `json:"cacheTtl"`
	MaxCacheEntries int           `json:"maxCacheEntries"`

	// Generation parameters
	MaxGenerationTime   time.Duration `json:"maxGenerationTime"`
	ParallelGeneration  bool          `json:"parallelGeneration"`
	ConcurrentTemplates int           `json:"concurrentTemplates"`
}

// KubernetesTemplateEngine handles template processing
type KubernetesTemplateEngine struct {
	logger          logr.Logger
	templates       map[string]*template.Template
	functions       template.FuncMap
	globalVariables map[string]interface{}
}

// ManifestTemplate defines a template for generating manifests
type ManifestTemplate struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`     // deployment, service, configmap, etc.
	Category     string                 `json:"category"` // 5GC, RAN, monitoring, etc.
	Template     string                 `json:"template"`
	Variables    []string               `json:"variables"`
	Dependencies []string               `json:"dependencies"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// ManifestPolicyRule defines policy enforcement rules
type ManifestPolicyRule struct {
	ID              string                 `json:"id"`
	Name            string                 `json:"name"`
	Type            string                 `json:"type"`     // security, resource, compliance
	Severity        string                 `json:"severity"` // info, warning, error
	Rule            ManifestPolicyCheck    `json:"rule"`
	ViolationAction string                 `json:"violationAction"` // reject, modify, warn
	Metadata        map[string]interface{} `json:"metadata"`
}

// ManifestPolicyCheck defines policy checking logic
type ManifestPolicyCheck func(manifest map[string]interface{}) error

// GenerationSession tracks active manifest generation
type GenerationSession struct {
	IntentID      string    `json:"intentId"`
	CorrelationID string    `json:"correlationId"`
	StartTime     time.Time `json:"startTime"`
	Status        string    `json:"status"`
	Progress      float64   `json:"progress"`
	CurrentStep   string    `json:"currentStep"`

	// Input data
	ResourcePlan     *interfaces.ResourcePlan            `json:"resourcePlan"`
	NetworkFunctions []interfaces.PlannedNetworkFunction `json:"networkFunctions"`

	// Generation results
	GeneratedManifests map[string]string      `json:"generatedManifests"`
	ManifestMetadata   map[string]interface{} `json:"manifestMetadata"`
	ValidationResults  []ValidationResult     `json:"validationResults"`
	PolicyResults      []PolicyResult         `json:"policyResults"`

	// Error handling
	Errors    []string `json:"errors"`
	Warnings  []string `json:"warnings"`
	LastError string   `json:"lastError"`

	// Performance tracking
	Metrics GenerationSessionMetrics `json:"metrics"`

	mutex sync.RWMutex
}

// GenerationSessionMetrics tracks metrics for generation session
type GenerationSessionMetrics struct {
	TemplateProcessingTime time.Duration `json:"templateProcessingTime"`
	ValidationTime         time.Duration `json:"validationTime"`
	PolicyCheckTime        time.Duration `json:"policyCheckTime"`
	OptimizationTime       time.Duration `json:"optimizationTime"`
	TotalTime              time.Duration `json:"totalTime"`
	ManifestsGenerated     int           `json:"manifestsGenerated"`
	TemplatesProcessed     int           `json:"templatesProcessed"`
	ValidationErrors       int           `json:"validationErrors"`
	PolicyViolations       int           `json:"policyViolations"`
	CacheHit               bool          `json:"cacheHit"`
}

// ManifestGenerationMetrics tracks overall controller metrics
type ManifestGenerationMetrics struct {
	TotalGenerated             int64         `json:"totalGenerated"`
	SuccessfulGenerated        int64         `json:"successfulGenerated"`
	FailedGenerated            int64         `json:"failedGenerated"`
	AverageGenerationTime      time.Duration `json:"averageGenerationTime"`
	AverageManifestsPerRequest int64         `json:"averageManifestsPerRequest"`

	// Per template metrics
	TemplateMetrics map[string]*TemplateGenerationMetrics `json:"templateMetrics"`

	// Validation and policy metrics
	ValidationSuccessRate float64 `json:"validationSuccessRate"`
	PolicyComplianceRate  float64 `json:"policyComplianceRate"`

	// Cache metrics
	CacheHitRate          float64 `json:"cacheHitRate"`
	TemplateCompileErrors int64   `json:"templateCompileErrors"`

	LastUpdated time.Time `json:"lastUpdated"`
	mutex       sync.RWMutex
}

// TemplateGenerationMetrics tracks metrics per template
type TemplateGenerationMetrics struct {
	TemplateName          string        `json:"templateName"`
	UsageCount            int64         `json:"usageCount"`
	SuccessRate           float64       `json:"successRate"`
	AverageProcessingTime time.Duration `json:"averageProcessingTime"`
	LastUsed              time.Time     `json:"lastUsed"`
}

// ManifestCache provides caching for generated manifests
type ManifestCache struct {
	entries    map[string]*ManifestCacheEntry
	mutex      sync.RWMutex
	ttl        time.Duration
	maxEntries int
}

// ManifestCacheEntry represents a cached manifest
type ManifestCacheEntry struct {
	Manifests map[string]string      `json:"manifests"`
	Metadata  map[string]interface{} `json:"metadata"`
	Timestamp time.Time              `json:"timestamp"`
	HitCount  int64                  `json:"hitCount"`
	PlanHash  string                 `json:"planHash"`
}

// ValidationResult represents manifest validation result
type ValidationResult struct {
	ManifestName string   `json:"manifestName"`
	Valid        bool     `json:"valid"`
	Errors       []string `json:"errors"`
	Warnings     []string `json:"warnings"`
}

// PolicyResult represents policy enforcement result
type PolicyResult struct {
	PolicyID   string   `json:"policyId"`
	PolicyName string   `json:"policyName"`
	Compliant  bool     `json:"compliant"`
	Violations []string `json:"violations"`
	Action     string   `json:"action"`
}

// NewSpecializedManifestGenerationController creates a new manifest generation controller
func NewSpecializedManifestGenerationController(mgr ctrl.Manager, config ManifestGenerationConfig) (*SpecializedManifestGenerationController, error) {
	logger := log.FromContext(context.Background()).WithName("specialized-manifest-generator")

	// Initialize template engine
	templateEngine := NewKubernetesTemplateEngine(logger, config)

	// Initialize manifest validator
	manifestValidator := NewManifestValidator(logger, config)

	// Initialize policy enforcer
	policyEnforcer := NewManifestPolicyEnforcer(logger, config)

	// Initialize manifest optimizer
	manifestOptimizer := NewManifestOptimizer(logger, config)

	// Initialize integrations
	var helmIntegration *HelmManifestIntegration
	var kustomizeIntegration *KustomizeManifestIntegration

	if config.EnableHelm {
		helmIntegration = NewHelmManifestIntegration(logger)
	}

	if config.EnableKustomize {
		kustomizeIntegration = NewKustomizeManifestIntegration(logger)
	}

	controller := &SpecializedManifestGenerationController{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("specialized-manifest-generator"),
		Logger:   logger,

		TemplateEngine:       templateEngine,
		ManifestValidator:    manifestValidator,
		PolicyEnforcer:       policyEnforcer,
		ManifestOptimizer:    manifestOptimizer,
		HelmIntegration:      helmIntegration,
		KustomizeIntegration: kustomizeIntegration,

		Config:      config,
		Templates:   initializeManifestTemplates(),
		PolicyRules: initializeManifestPolicyRules(),

		metrics:  NewManifestGenerationMetrics(),
		stopChan: make(chan struct{}),

		healthStatus: interfaces.HealthStatus{
			Status:      "Healthy",
			Message:     "Manifest generation controller initialized",
			LastChecked: time.Now(),
		},
	}

	// Initialize cache if enabled
	if config.CacheEnabled {
		controller.manifestCache = &ManifestCache{
			entries:    make(map[string]*ManifestCacheEntry),
			ttl:        config.CacheTTL,
			maxEntries: config.MaxCacheEntries,
		}
	}

	return controller, nil
}

// ProcessPhase implements the PhaseController interface
func (c *SpecializedManifestGenerationController) ProcessPhase(ctx context.Context, intent *nephoranv1.NetworkIntent, phase interfaces.ProcessingPhase) (interfaces.ProcessingResult, error) {
	if phase != interfaces.PhaseManifestGeneration {
		return interfaces.ProcessingResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("unsupported phase: %s", phase),
		}, nil
	}

	// Extract resource plan from intent status
	resourcePlanInterface, ok := intent.Status.ResourcePlan.(map[string]interface{})
	if !ok {
		return interfaces.ProcessingResult{
			Success:      false,
			ErrorMessage: "invalid or missing resource plan data",
			ErrorCode:    "INVALID_INPUT",
		}, nil
	}

	// Convert to ResourcePlan struct
	resourcePlan, err := c.convertToResourcePlan(resourcePlanInterface)
	if err != nil {
		return interfaces.ProcessingResult{
			Success:      false,
			ErrorMessage: fmt.Sprintf("failed to convert resource plan: %v", err),
			ErrorCode:    "CONVERSION_ERROR",
		}, nil
	}

	return c.generateManifestsFromResourcePlan(ctx, intent, resourcePlan)
}

// GenerateManifests implements the ManifestGenerator interface
func (c *SpecializedManifestGenerationController) GenerateManifests(ctx context.Context, resourcePlan *interfaces.ResourcePlan) (map[string]string, error) {
	result, err := c.generateManifestsFromResourcePlan(ctx, nil, resourcePlan)
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, fmt.Errorf("manifest generation failed: %s", result.ErrorMessage)
	}

	if manifests, ok := result.Data["generatedManifests"].(map[string]string); ok {
		return manifests, nil
	}

	return nil, fmt.Errorf("invalid manifests in result")
}

// generateManifestsFromResourcePlan performs manifest generation based on resource plan
func (c *SpecializedManifestGenerationController) generateManifestsFromResourcePlan(ctx context.Context, intent *nephoranv1.NetworkIntent, resourcePlan *interfaces.ResourcePlan) (interfaces.ProcessingResult, error) {
	startTime := time.Now()

	intentID := "unknown"
	if intent != nil {
		intentID = intent.Name
	}

	c.Logger.Info("Generating manifests from resource plan", "intentId", intentID)

	// Create generation session
	session := &GenerationSession{
		IntentID:         intentID,
		CorrelationID:    fmt.Sprintf("manifest-%s-%d", intentID, startTime.Unix()),
		StartTime:        startTime,
		Status:           "generating",
		Progress:         0.0,
		CurrentStep:      "initialization",
		ResourcePlan:     resourcePlan,
		NetworkFunctions: resourcePlan.NetworkFunctions,
	}

	// Store session for tracking
	c.activeGeneration.Store(intentID, session)
	defer c.activeGeneration.Delete(intentID)

	// Check cache first
	if c.manifestCache != nil {
		if cached := c.getCachedManifests(resourcePlan); cached != nil {
			c.Logger.Info("Cache hit for manifest generation", "intentId", intentID)
			session.Metrics.CacheHit = true
			c.updateMetrics(session, true, time.Since(startTime))

			return interfaces.ProcessingResult{
				Success:   true,
				NextPhase: interfaces.PhaseGitOpsCommit,
				Data: map[string]interface{}{
					"generatedManifests": cached.Manifests,
					"manifestMetadata":   cached.Metadata,
					"correlationId":      session.CorrelationID,
				},
				Metrics: map[string]float64{
					"generation_time_ms": float64(time.Since(startTime).Milliseconds()),
					"cache_hit":          1,
				},
			}, nil
		}
	}

	// Generate manifests for each network function
	session.updateProgress(0.2, "generating_manifests")
	generatedManifests := make(map[string]string)
	manifestMetadata := make(map[string]interface{})

	templateStartTime := time.Now()

	if c.Config.ParallelGeneration {
		manifests, metadata, err := c.generateManifestsParallel(ctx, session)
		if err != nil {
			session.addError(fmt.Sprintf("parallel manifest generation failed: %v", err))
			c.updateMetrics(session, false, time.Since(startTime))
			return interfaces.ProcessingResult{
				Success:      false,
				ErrorMessage: err.Error(),
				ErrorCode:    "GENERATION_ERROR",
			}, nil
		}
		generatedManifests = manifests
		manifestMetadata = metadata
	} else {
		manifests, metadata, err := c.generateManifestsSequential(ctx, session)
		if err != nil {
			session.addError(fmt.Sprintf("sequential manifest generation failed: %v", err))
			c.updateMetrics(session, false, time.Since(startTime))
			return interfaces.ProcessingResult{
				Success:      false,
				ErrorMessage: err.Error(),
				ErrorCode:    "GENERATION_ERROR",
			}, nil
		}
		generatedManifests = manifests
		manifestMetadata = metadata
	}

	session.Metrics.TemplateProcessingTime = time.Since(templateStartTime)
	session.GeneratedManifests = generatedManifests
	session.ManifestMetadata = manifestMetadata
	session.Metrics.ManifestsGenerated = len(generatedManifests)

	// Validate manifests if enabled
	if c.Config.ValidateManifests {
		session.updateProgress(0.6, "validating_manifests")
		validationStartTime := time.Now()

		validationResults, err := c.ValidateManifests(ctx, generatedManifests)
		if err != nil {
			session.addWarning(fmt.Sprintf("manifest validation failed: %v", err))
			// Continue with warnings
		} else {
			session.ValidationResults = validationResults
			// Check for validation errors
			for _, result := range validationResults {
				if !result.Valid {
					session.Metrics.ValidationErrors++
					session.addWarning(fmt.Sprintf("manifest %s validation failed: %v", result.ManifestName, result.Errors))
				}
			}
		}

		session.Metrics.ValidationTime = time.Since(validationStartTime)
	}

	// Enforce policies if enabled
	if c.Config.EnforcePolicies {
		session.updateProgress(0.7, "enforcing_policies")
		policyStartTime := time.Now()

		policyResults, modifiedManifests, err := c.enforcePolicies(ctx, generatedManifests)
		if err != nil {
			session.addWarning(fmt.Sprintf("policy enforcement failed: %v", err))
		} else {
			session.PolicyResults = policyResults
			// Use modified manifests if policies made changes
			if len(modifiedManifests) > 0 {
				generatedManifests = modifiedManifests
				session.GeneratedManifests = generatedManifests
			}

			// Check for policy violations
			for _, result := range policyResults {
				if !result.Compliant {
					session.Metrics.PolicyViolations++
					session.addWarning(fmt.Sprintf("policy %s violated: %v", result.PolicyName, result.Violations))
				}
			}
		}

		session.Metrics.PolicyCheckTime = time.Since(policyStartTime)
	}

	// Optimize manifests if enabled
	if c.Config.OptimizeManifests {
		session.updateProgress(0.8, "optimizing_manifests")
		optimizationStartTime := time.Now()

		optimizedManifests, err := c.OptimizeManifests(ctx, generatedManifests)
		if err != nil {
			session.addWarning(fmt.Sprintf("manifest optimization failed: %v", err))
			// Continue without optimization
		} else {
			generatedManifests = optimizedManifests
			session.GeneratedManifests = generatedManifests
		}

		session.Metrics.OptimizationTime = time.Since(optimizationStartTime)
	}

	// Finalize
	session.updateProgress(1.0, "completed")
	session.Metrics.TotalTime = time.Since(startTime)

	// Create result
	resultData := map[string]interface{}{
		"generatedManifests": generatedManifests,
		"manifestMetadata":   manifestMetadata,
		"correlationId":      session.CorrelationID,
		"networkFunctions":   len(session.NetworkFunctions),
	}

	if len(session.ValidationResults) > 0 {
		resultData["validationResults"] = session.ValidationResults
	}
	if len(session.PolicyResults) > 0 {
		resultData["policyResults"] = session.PolicyResults
	}

	result := interfaces.ProcessingResult{
		Success:   true,
		NextPhase: interfaces.PhaseGitOpsCommit,
		Data:      resultData,
		Metrics: map[string]float64{
			"generation_time_ms":          float64(session.Metrics.TotalTime.Milliseconds()),
			"template_processing_time_ms": float64(session.Metrics.TemplateProcessingTime.Milliseconds()),
			"validation_time_ms":          float64(session.Metrics.ValidationTime.Milliseconds()),
			"policy_check_time_ms":        float64(session.Metrics.PolicyCheckTime.Milliseconds()),
			"optimization_time_ms":        float64(session.Metrics.OptimizationTime.Milliseconds()),
			"manifests_generated":         float64(session.Metrics.ManifestsGenerated),
			"validation_errors":           float64(session.Metrics.ValidationErrors),
			"policy_violations":           float64(session.Metrics.PolicyViolations),
		},
		Events: []interfaces.ProcessingEvent{
			{
				Timestamp:     time.Now(),
				EventType:     "ManifestsGenerated",
				Message:       fmt.Sprintf("Generated %d manifests for %d network functions", len(generatedManifests), len(session.NetworkFunctions)),
				CorrelationID: session.CorrelationID,
				Data: map[string]interface{}{
					"intentId":           intentID,
					"manifestsGenerated": len(generatedManifests),
					"generationTimeMs":   session.Metrics.TotalTime.Milliseconds(),
				},
			},
		},
	}

	// Cache result if enabled
	if c.manifestCache != nil {
		c.cacheManifests(resourcePlan, generatedManifests, manifestMetadata)
	}

	// Update metrics
	c.updateMetrics(session, true, time.Since(startTime))

	c.Logger.Info("Manifest generation completed successfully",
		"intentId", intentID,
		"manifestCount", len(generatedManifests),
		"duration", time.Since(startTime))

	return result, nil
}

// generateManifestsSequential generates manifests sequentially
func (c *SpecializedManifestGenerationController) generateManifestsSequential(ctx context.Context, session *GenerationSession) (map[string]string, map[string]interface{}, error) {
	generatedManifests := make(map[string]string)
	manifestMetadata := make(map[string]interface{})

	for i, nf := range session.NetworkFunctions {
		progress := 0.2 + (float64(i)/float64(len(session.NetworkFunctions)))*0.4 // 20% to 60%
		session.updateProgress(progress, fmt.Sprintf("generating_manifest_%s", nf.Name))

		manifests, metadata, err := c.generateNetworkFunctionManifests(ctx, &nf, session.ResourcePlan)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to generate manifests for %s: %w", nf.Name, err)
		}

		// Merge manifests
		for name, manifest := range manifests {
			generatedManifests[name] = manifest
		}

		// Merge metadata
		for key, value := range metadata {
			manifestMetadata[key] = value
		}

		session.Metrics.TemplatesProcessed++
	}

	return generatedManifests, manifestMetadata, nil
}

// generateManifestsParallel generates manifests in parallel
func (c *SpecializedManifestGenerationController) generateManifestsParallel(ctx context.Context, session *GenerationSession) (map[string]string, map[string]interface{}, error) {
	// Create buffered channel for results
	resultChan := make(chan *nfManifestResult, len(session.NetworkFunctions))
	errorChan := make(chan error, len(session.NetworkFunctions))

	// Limit concurrency
	semaphore := make(chan struct{}, c.Config.ConcurrentTemplates)

	// Generate manifests concurrently
	for _, nf := range session.NetworkFunctions {
		go func(networkFunction interfaces.PlannedNetworkFunction) {
			semaphore <- struct{}{}        // Acquire semaphore
			defer func() { <-semaphore }() // Release semaphore

			manifests, metadata, err := c.generateNetworkFunctionManifests(ctx, &networkFunction, session.ResourcePlan)
			if err != nil {
				errorChan <- fmt.Errorf("failed to generate manifests for %s: %w", networkFunction.Name, err)
				return
			}

			resultChan <- &nfManifestResult{
				NFName:    networkFunction.Name,
				Manifests: manifests,
				Metadata:  metadata,
			}
		}(nf)
	}

	// Collect results
	generatedManifests := make(map[string]string)
	manifestMetadata := make(map[string]interface{})
	completedCount := 0

	for completedCount < len(session.NetworkFunctions) {
		select {
		case result := <-resultChan:
			// Merge manifests
			for name, manifest := range result.Manifests {
				generatedManifests[name] = manifest
			}

			// Merge metadata
			for key, value := range result.Metadata {
				manifestMetadata[key] = value
			}

			completedCount++
			session.Metrics.TemplatesProcessed++

			// Update progress
			progress := 0.2 + (float64(completedCount)/float64(len(session.NetworkFunctions)))*0.4
			session.updateProgress(progress, fmt.Sprintf("completed_%d_of_%d_manifests", completedCount, len(session.NetworkFunctions)))

		case err := <-errorChan:
			return nil, nil, err

		case <-ctx.Done():
			return nil, nil, ctx.Err()
		}
	}

	return generatedManifests, manifestMetadata, nil
}

// nfManifestResult holds result of network function manifest generation
type nfManifestResult struct {
	NFName    string                 `json:"nfName"`
	Manifests map[string]string      `json:"manifests"`
	Metadata  map[string]interface{} `json:"metadata"`
}

// generateNetworkFunctionManifests generates manifests for a single network function
func (c *SpecializedManifestGenerationController) generateNetworkFunctionManifests(ctx context.Context, nf *interfaces.PlannedNetworkFunction, resourcePlan *interfaces.ResourcePlan) (map[string]string, map[string]interface{}, error) {
	manifests := make(map[string]string)
	metadata := make(map[string]interface{})

	// Prepare template variables
	templateVars := c.prepareTemplateVariables(nf, resourcePlan)

	// Generate deployment manifest
	if deploymentManifest, err := c.TemplateEngine.ProcessTemplate("deployment", nf.Type, templateVars); err == nil {
		manifestName := fmt.Sprintf("%s-deployment.yaml", nf.Name)
		manifests[manifestName] = deploymentManifest
		metadata[manifestName] = map[string]interface{}{
			"type": "deployment",
			"nf":   nf.Name,
		}
	} else {
		return nil, nil, fmt.Errorf("failed to generate deployment manifest: %w", err)
	}

	// Generate service manifest if ports are defined
	if len(nf.Ports) > 0 {
		if serviceManifest, err := c.TemplateEngine.ProcessTemplate("service", nf.Type, templateVars); err == nil {
			manifestName := fmt.Sprintf("%s-service.yaml", nf.Name)
			manifests[manifestName] = serviceManifest
			metadata[manifestName] = map[string]interface{}{
				"type": "service",
				"nf":   nf.Name,
			}
		} else {
			c.Logger.Error(err, "Failed to generate service manifest", "nf", nf.Name)
		}
	}

	// Generate configmap manifest if configuration is present
	if len(nf.Configuration) > 0 {
		if configMapManifest, err := c.TemplateEngine.ProcessTemplate("configmap", nf.Type, templateVars); err == nil {
			manifestName := fmt.Sprintf("%s-configmap.yaml", nf.Name)
			manifests[manifestName] = configMapManifest
			metadata[manifestName] = map[string]interface{}{
				"type": "configmap",
				"nf":   nf.Name,
			}
		} else {
			c.Logger.Error(err, "Failed to generate configmap manifest", "nf", nf.Name)
		}
	}

	// Generate RBAC manifests for certain network functions
	if c.requiresRBAC(nf.Type) {
		if rbacManifests, err := c.generateRBACManifests(nf, templateVars); err == nil {
			for name, manifest := range rbacManifests {
				manifests[name] = manifest
				metadata[name] = map[string]interface{}{
					"type": "rbac",
					"nf":   nf.Name,
				}
			}
		} else {
			c.Logger.Error(err, "Failed to generate RBAC manifests", "nf", nf.Name)
		}
	}

	return manifests, metadata, nil
}

// prepareTemplateVariables prepares variables for template processing
func (c *SpecializedManifestGenerationController) prepareTemplateVariables(nf *interfaces.PlannedNetworkFunction, resourcePlan *interfaces.ResourcePlan) map[string]interface{} {
	return map[string]interface{}{
		"NetworkFunction":   nf,
		"ResourcePlan":      resourcePlan,
		"Namespace":         c.Config.DefaultNamespace,
		"Name":              nf.Name,
		"Type":              nf.Type,
		"Image":             nf.Image,
		"Version":           nf.Version,
		"Replicas":          nf.Replicas,
		"Resources":         nf.Resources,
		"Ports":             nf.Ports,
		"Environment":       nf.Environment,
		"Configuration":     nf.Configuration,
		"DeploymentPattern": resourcePlan.DeploymentPattern,
		"Labels": map[string]string{
			"app.kubernetes.io/name":       nf.Name,
			"app.kubernetes.io/component":  nf.Type,
			"app.kubernetes.io/managed-by": "nephoran-intent-operator",
		},
	}
}

// requiresRBAC checks if network function requires RBAC
func (c *SpecializedManifestGenerationController) requiresRBAC(nfType string) bool {
	rbacRequiredTypes := []string{"amf", "smf", "nrf", "ausf", "udm", "udr", "pcf"}
	for _, requiredType := range rbacRequiredTypes {
		if strings.Contains(strings.ToLower(nfType), requiredType) {
			return true
		}
	}
	return false
}

// generateRBACManifests generates RBAC manifests
func (c *SpecializedManifestGenerationController) generateRBACManifests(nf *interfaces.PlannedNetworkFunction, templateVars map[string]interface{}) (map[string]string, error) {
	rbacManifests := make(map[string]string)

	// Generate service account
	if saManifest, err := c.TemplateEngine.ProcessTemplate("serviceaccount", nf.Type, templateVars); err == nil {
		rbacManifests[fmt.Sprintf("%s-serviceaccount.yaml", nf.Name)] = saManifest
	}

	// Generate cluster role
	if crManifest, err := c.TemplateEngine.ProcessTemplate("clusterrole", nf.Type, templateVars); err == nil {
		rbacManifests[fmt.Sprintf("%s-clusterrole.yaml", nf.Name)] = crManifest
	}

	// Generate cluster role binding
	if crbManifest, err := c.TemplateEngine.ProcessTemplate("clusterrolebinding", nf.Type, templateVars); err == nil {
		rbacManifests[fmt.Sprintf("%s-clusterrolebinding.yaml", nf.Name)] = crbManifest
	}

	return rbacManifests, nil
}

// convertToResourcePlan converts interface to ResourcePlan struct
func (c *SpecializedManifestGenerationController) convertToResourcePlan(data map[string]interface{}) (*interfaces.ResourcePlan, error) {
	// This is a simplified conversion - in practice, you'd use proper serialization
	plan := &interfaces.ResourcePlan{}

	// Convert network functions
	if nfsInterface, exists := data["networkFunctions"]; exists {
		if nfsArray, ok := nfsInterface.([]interface{}); ok {
			for _, nfInterface := range nfsArray {
				if nfMap, ok := nfInterface.(map[string]interface{}); ok {
					nf := interfaces.PlannedNetworkFunction{}
					if name, exists := nfMap["name"]; exists {
						nf.Name, _ = name.(string)
					}
					if nfType, exists := nfMap["type"]; exists {
						nf.Type, _ = nfType.(string)
					}
					// Add more field conversions as needed
					plan.NetworkFunctions = append(plan.NetworkFunctions, nf)
				}
			}
		}
	}

	// Convert resource requirements
	if resourcesInterface, exists := data["resourceRequirements"]; exists {
		if resourcesMap, ok := resourcesInterface.(map[string]interface{}); ok {
			if cpu, exists := resourcesMap["cpu"]; exists {
				plan.ResourceRequirements.CPU, _ = cpu.(string)
			}
			if memory, exists := resourcesMap["memory"]; exists {
				plan.ResourceRequirements.Memory, _ = memory.(string)
			}
			if storage, exists := resourcesMap["storage"]; exists {
				plan.ResourceRequirements.Storage, _ = storage.(string)
			}
		}
	}

	// Convert deployment pattern
	if pattern, exists := data["deploymentPattern"]; exists {
		plan.DeploymentPattern, _ = pattern.(string)
	}

	return plan, nil
}

// ValidateManifests implements the ManifestGenerator interface
func (c *SpecializedManifestGenerationController) ValidateManifests(ctx context.Context, manifests map[string]string) error {
	if !c.Config.ValidateManifests {
		return nil
	}

	return c.ManifestValidator.ValidateManifests(ctx, manifests)
}

// OptimizeManifests implements the ManifestGenerator interface
func (c *SpecializedManifestGenerationController) OptimizeManifests(ctx context.Context, manifests map[string]string) (map[string]string, error) {
	if !c.Config.OptimizeManifests {
		return manifests, nil
	}

	return c.ManifestOptimizer.OptimizeManifests(ctx, manifests)
}

// GetSupportedTemplates implements the ManifestGenerator interface
func (c *SpecializedManifestGenerationController) GetSupportedTemplates() []string {
	templates := make([]string, 0, len(c.Templates))
	for name := range c.Templates {
		templates = append(templates, name)
	}
	return templates
}

// enforcePolicies enforces manifest policies
func (c *SpecializedManifestGenerationController) enforcePolicies(ctx context.Context, manifests map[string]string) ([]PolicyResult, map[string]string, error) {
	if !c.Config.EnforcePolicies {
		return nil, nil, nil
	}

	return c.PolicyEnforcer.EnforcePolicies(ctx, manifests, c.PolicyRules)
}

// Helper methods for session management

// updateProgress updates session progress
func (s *GenerationSession) updateProgress(progress float64, step string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Progress = progress
	s.CurrentStep = step
}

// addError adds error to session
func (s *GenerationSession) addError(errorMsg string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Errors = append(s.Errors, errorMsg)
	s.LastError = errorMsg
}

// addWarning adds warning to session
func (s *GenerationSession) addWarning(warningMsg string) {
	s.mutex.Lock()
	defer s.mutex.Unlock()
	s.Warnings = append(s.Warnings, warningMsg)
}

// Cache management methods

// getCachedManifests retrieves cached manifests
func (c *SpecializedManifestGenerationController) getCachedManifests(resourcePlan *interfaces.ResourcePlan) *ManifestCacheEntry {
	if c.manifestCache == nil {
		return nil
	}

	c.manifestCache.mutex.RLock()
	defer c.manifestCache.mutex.RUnlock()

	planHash := c.hashResourcePlan(resourcePlan)
	if entry, exists := c.manifestCache.entries[planHash]; exists {
		// Check if entry is still valid
		if time.Since(entry.Timestamp) < c.manifestCache.ttl {
			entry.HitCount++
			return entry
		}
		// Entry expired, remove it
		delete(c.manifestCache.entries, planHash)
	}

	return nil
}

// cacheManifests stores manifests in cache
func (c *SpecializedManifestGenerationController) cacheManifests(resourcePlan *interfaces.ResourcePlan, manifests map[string]string, metadata map[string]interface{}) {
	if c.manifestCache == nil {
		return
	}

	c.manifestCache.mutex.Lock()
	defer c.manifestCache.mutex.Unlock()

	planHash := c.hashResourcePlan(resourcePlan)

	// Check cache size limit
	if len(c.manifestCache.entries) >= c.manifestCache.maxEntries {
		// Remove oldest entry (simple LRU)
		var oldestKey string
		var oldestTime time.Time
		for key, entry := range c.manifestCache.entries {
			if oldestKey == "" || entry.Timestamp.Before(oldestTime) {
				oldestKey = key
				oldestTime = entry.Timestamp
			}
		}
		if oldestKey != "" {
			delete(c.manifestCache.entries, oldestKey)
		}
	}

	c.manifestCache.entries[planHash] = &ManifestCacheEntry{
		Manifests: manifests,
		Metadata:  metadata,
		Timestamp: time.Now(),
		HitCount:  0,
		PlanHash:  planHash,
	}
}

// hashResourcePlan creates a hash for resource plan
func (c *SpecializedManifestGenerationController) hashResourcePlan(resourcePlan *interfaces.ResourcePlan) string {
	// Simple hash based on key characteristics - in production, use proper hashing
	var hashComponents []string

	for _, nf := range resourcePlan.NetworkFunctions {
		hashComponents = append(hashComponents, fmt.Sprintf("%s:%s", nf.Name, nf.Type))
	}
	hashComponents = append(hashComponents, resourcePlan.DeploymentPattern)
	hashComponents = append(hashComponents, resourcePlan.ResourceRequirements.CPU)
	hashComponents = append(hashComponents, resourcePlan.ResourceRequirements.Memory)

	combined := strings.Join(hashComponents, "|")
	return fmt.Sprintf("manifest_%x", len(combined)+int(combined[0]))
}

// updateMetrics updates controller metrics
func (c *SpecializedManifestGenerationController) updateMetrics(session *GenerationSession, success bool, totalDuration time.Duration) {
	c.metrics.mutex.Lock()
	defer c.metrics.mutex.Unlock()

	c.metrics.TotalGenerated++
	if success {
		c.metrics.SuccessfulGenerated++
	} else {
		c.metrics.FailedGenerated++
	}

	// Update average generation time
	if c.metrics.TotalGenerated > 0 {
		totalTime := time.Duration(c.metrics.TotalGenerated-1) * c.metrics.AverageGenerationTime
		totalTime += totalDuration
		c.metrics.AverageGenerationTime = totalTime / time.Duration(c.metrics.TotalGenerated)
	} else {
		c.metrics.AverageGenerationTime = totalDuration
	}

	// Update average manifests per request
	if c.metrics.TotalGenerated > 0 {
		totalManifests := (c.metrics.TotalGenerated-1)*c.metrics.AverageManifestsPerRequest + int64(session.Metrics.ManifestsGenerated)
		c.metrics.AverageManifestsPerRequest = totalManifests / c.metrics.TotalGenerated
	} else {
		c.metrics.AverageManifestsPerRequest = int64(session.Metrics.ManifestsGenerated)
	}

	// Update validation success rate
	if c.Config.ValidateManifests && len(session.ValidationResults) > 0 {
		validationSuccesses := 0
		for _, result := range session.ValidationResults {
			if result.Valid {
				validationSuccesses++
			}
		}
		// Simple moving average for validation success rate
		c.metrics.ValidationSuccessRate = 0.9*c.metrics.ValidationSuccessRate + 0.1*float64(validationSuccesses)/float64(len(session.ValidationResults))
	}

	// Update policy compliance rate
	if c.Config.EnforcePolicies && len(session.PolicyResults) > 0 {
		policyCompliance := 0
		for _, result := range session.PolicyResults {
			if result.Compliant {
				policyCompliance++
			}
		}
		// Simple moving average for policy compliance rate
		c.metrics.PolicyComplianceRate = 0.9*c.metrics.PolicyComplianceRate + 0.1*float64(policyCompliance)/float64(len(session.PolicyResults))
	}

	// Update cache hit rate
	if c.manifestCache != nil {
		totalRequests := c.metrics.TotalGenerated
		cacheHits := int64(0)
		c.manifestCache.mutex.RLock()
		for _, entry := range c.manifestCache.entries {
			cacheHits += entry.HitCount
		}
		c.manifestCache.mutex.RUnlock()

		if totalRequests > 0 {
			c.metrics.CacheHitRate = float64(cacheHits) / float64(totalRequests)
		}
	}

	c.metrics.LastUpdated = time.Now()
}

// NewManifestGenerationMetrics creates new metrics instance
func NewManifestGenerationMetrics() *ManifestGenerationMetrics {
	return &ManifestGenerationMetrics{
		TemplateMetrics: make(map[string]*TemplateGenerationMetrics),
	}
}

// Interface implementation methods

// GetPhaseStatus returns the status of a processing phase
func (c *SpecializedManifestGenerationController) GetPhaseStatus(ctx context.Context, intentID string) (*interfaces.PhaseStatus, error) {
	if session, exists := c.activeGeneration.Load(intentID); exists {
		s := session.(*GenerationSession)
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
			Phase:     interfaces.PhaseManifestGeneration,
			Status:    status,
			StartTime: &metav1.Time{Time: s.StartTime},
			LastError: s.LastError,
			Metrics: map[string]float64{
				"progress":                    s.Progress,
				"template_processing_time_ms": float64(s.Metrics.TemplateProcessingTime.Milliseconds()),
				"validation_time_ms":          float64(s.Metrics.ValidationTime.Milliseconds()),
				"policy_check_time_ms":        float64(s.Metrics.PolicyCheckTime.Milliseconds()),
				"manifests_generated":         float64(s.Metrics.ManifestsGenerated),
				"validation_errors":           float64(s.Metrics.ValidationErrors),
				"policy_violations":           float64(s.Metrics.PolicyViolations),
			},
		}, nil
	}

	return &interfaces.PhaseStatus{
		Phase:  interfaces.PhaseManifestGeneration,
		Status: "Pending",
	}, nil
}

// HandlePhaseError handles errors during phase processing
func (c *SpecializedManifestGenerationController) HandlePhaseError(ctx context.Context, intentID string, err error) error {
	c.Logger.Error(err, "Manifest generation error", "intentId", intentID)

	if session, exists := c.activeGeneration.Load(intentID); exists {
		s := session.(*GenerationSession)
		s.addError(err.Error())
	}

	return err
}

// GetDependencies returns phase dependencies
func (c *SpecializedManifestGenerationController) GetDependencies() []interfaces.ProcessingPhase {
	return []interfaces.ProcessingPhase{interfaces.PhaseResourcePlanning}
}

// GetBlockedPhases returns phases blocked by this controller
func (c *SpecializedManifestGenerationController) GetBlockedPhases() []interfaces.ProcessingPhase {
	return []interfaces.ProcessingPhase{
		interfaces.PhaseGitOpsCommit,
		interfaces.PhaseDeploymentVerification,
	}
}

// SetupWithManager sets up the controller with the Manager
func (c *SpecializedManifestGenerationController) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&nephoranv1.NetworkIntent{}).
		Complete(c)
}

// Start starts the controller
func (c *SpecializedManifestGenerationController) Start(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if c.started {
		return fmt.Errorf("controller already started")
	}

	c.Logger.Info("Starting specialized manifest generation controller")

	// Start background goroutines
	go c.backgroundCleanup(ctx)
	go c.healthMonitoring(ctx)

	c.started = true
	c.healthStatus = interfaces.HealthStatus{
		Status:      "Healthy",
		Message:     "Manifest generation controller started successfully",
		LastChecked: time.Now(),
	}

	return nil
}

// Stop stops the controller
func (c *SpecializedManifestGenerationController) Stop(ctx context.Context) error {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	if !c.started {
		return nil
	}

	c.Logger.Info("Stopping specialized manifest generation controller")

	close(c.stopChan)
	c.started = false
	c.healthStatus = interfaces.HealthStatus{
		Status:      "Stopped",
		Message:     "Controller stopped",
		LastChecked: time.Now(),
	}

	return nil
}

// GetHealthStatus returns the health status of the controller
func (c *SpecializedManifestGenerationController) GetHealthStatus(ctx context.Context) (interfaces.HealthStatus, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	c.healthStatus.Metrics = map[string]interface{}{
		"totalGenerated":        c.metrics.TotalGenerated,
		"successRate":           c.getSuccessRate(),
		"averageGenerationTime": c.metrics.AverageGenerationTime.Milliseconds(),
		"validationSuccessRate": c.metrics.ValidationSuccessRate,
		"policyComplianceRate":  c.metrics.PolicyComplianceRate,
		"cacheHitRate":          c.metrics.CacheHitRate,
		"activeGeneration":      c.getActiveGenerationCount(),
	}
	c.healthStatus.LastChecked = time.Now()

	return c.healthStatus, nil
}

// GetMetrics returns controller metrics
func (c *SpecializedManifestGenerationController) GetMetrics(ctx context.Context) (map[string]float64, error) {
	c.metrics.mutex.RLock()
	defer c.metrics.mutex.RUnlock()

	return map[string]float64{
		"total_generated":               float64(c.metrics.TotalGenerated),
		"successful_generated":          float64(c.metrics.SuccessfulGenerated),
		"failed_generated":              float64(c.metrics.FailedGenerated),
		"success_rate":                  c.getSuccessRate(),
		"average_generation_time_ms":    float64(c.metrics.AverageGenerationTime.Milliseconds()),
		"average_manifests_per_request": float64(c.metrics.AverageManifestsPerRequest),
		"validation_success_rate":       c.metrics.ValidationSuccessRate,
		"policy_compliance_rate":        c.metrics.PolicyComplianceRate,
		"cache_hit_rate":                c.metrics.CacheHitRate,
		"template_compile_errors":       float64(c.metrics.TemplateCompileErrors),
		"active_generation":             float64(c.getActiveGenerationCount()),
	}, nil
}

// Reconcile implements the controller reconciliation logic
func (c *SpecializedManifestGenerationController) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// Get the NetworkIntent
	var intent nephoranv1.NetworkIntent
	if err := c.Get(ctx, req.NamespacedName, &intent); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("NetworkIntent resource not found, ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		logger.Error(err, "Failed to get NetworkIntent")
		return ctrl.Result{}, err
	}

	// Check if this intent should be processed by this controller
	if intent.Status.ProcessingPhase != interfaces.PhaseManifestGeneration {
		return ctrl.Result{}, nil
	}

	// Process the intent
	result, err := c.ProcessPhase(ctx, &intent, interfaces.PhaseManifestGeneration)
	if err != nil {
		logger.Error(err, "Failed to process manifest generation phase")
		return ctrl.Result{RequeueAfter: time.Minute * 1}, err
	}

	// Update intent status based on result
	if result.Success {
		intent.Status.ProcessingPhase = result.NextPhase
		intent.Status.GeneratedManifests = result.Data
		intent.Status.LastUpdated = metav1.Now()
	} else {
		intent.Status.ProcessingPhase = interfaces.PhaseFailed
		intent.Status.ErrorMessage = result.ErrorMessage
		intent.Status.LastUpdated = metav1.Now()
	}

	// Update the intent status
	if err := c.Status().Update(ctx, &intent); err != nil {
		logger.Error(err, "Failed to update NetworkIntent status")
		return ctrl.Result{}, err
	}

	logger.Info("Manifest generation completed", "intentId", intent.Name, "success", result.Success)

	return ctrl.Result{}, nil
}

// Helper methods

// backgroundCleanup performs background cleanup tasks
func (c *SpecializedManifestGenerationController) backgroundCleanup(ctx context.Context) {
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

// healthMonitoring performs periodic health monitoring
func (c *SpecializedManifestGenerationController) healthMonitoring(ctx context.Context) {
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

// cleanupExpiredSessions removes expired generation sessions
func (c *SpecializedManifestGenerationController) cleanupExpiredSessions() {
	expiredSessions := make([]string, 0)

	c.activeGeneration.Range(func(key, value interface{}) bool {
		session := value.(*GenerationSession)
		if time.Since(session.StartTime) > time.Hour { // 1 hour expiration
			expiredSessions = append(expiredSessions, key.(string))
		}
		return true
	})

	for _, sessionID := range expiredSessions {
		c.activeGeneration.Delete(sessionID)
		c.Logger.Info("Cleaned up expired generation session", "sessionId", sessionID)
	}
}

// cleanupExpiredCache removes expired cache entries
func (c *SpecializedManifestGenerationController) cleanupExpiredCache() {
	if c.manifestCache == nil {
		return
	}

	c.manifestCache.mutex.Lock()
	defer c.manifestCache.mutex.Unlock()

	expiredKeys := make([]string, 0)
	for key, entry := range c.manifestCache.entries {
		if time.Since(entry.Timestamp) > c.manifestCache.ttl {
			expiredKeys = append(expiredKeys, key)
		}
	}

	for _, key := range expiredKeys {
		delete(c.manifestCache.entries, key)
	}

	if len(expiredKeys) > 0 {
		c.Logger.Info("Cleaned up expired cache entries", "count", len(expiredKeys))
	}
}

// performHealthCheck performs controller health checking
func (c *SpecializedManifestGenerationController) performHealthCheck() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	successRate := c.getSuccessRate()
	activeCount := c.getActiveGenerationCount()

	status := "Healthy"
	message := "Manifest generation controller operating normally"

	if successRate < 0.8 && c.metrics.TotalGenerated > 10 {
		status = "Degraded"
		message = fmt.Sprintf("Low success rate: %.2f", successRate)
	}

	if activeCount > 20 { // Threshold for too many active sessions
		status = "Degraded"
		message = fmt.Sprintf("High active generation count: %d", activeCount)
	}

	if c.metrics.ValidationSuccessRate < 0.9 && c.Config.ValidateManifests {
		status = "Degraded"
		message = fmt.Sprintf("Low validation success rate: %.2f", c.metrics.ValidationSuccessRate)
	}

	c.healthStatus = interfaces.HealthStatus{
		Status:      status,
		Message:     message,
		LastChecked: time.Now(),
		Metrics: map[string]interface{}{
			"successRate":           successRate,
			"activeGeneration":      activeCount,
			"totalGenerated":        c.metrics.TotalGenerated,
			"validationSuccessRate": c.metrics.ValidationSuccessRate,
		},
	}
}

// getSuccessRate calculates success rate
func (c *SpecializedManifestGenerationController) getSuccessRate() float64 {
	if c.metrics.TotalGenerated == 0 {
		return 1.0
	}
	return float64(c.metrics.SuccessfulGenerated) / float64(c.metrics.TotalGenerated)
}

// getActiveGenerationCount returns count of active generation sessions
func (c *SpecializedManifestGenerationController) getActiveGenerationCount() int {
	count := 0
	c.activeGeneration.Range(func(key, value interface{}) bool {
		count++
		return true
	})
	return count
}

// Helper service implementations

// NewKubernetesTemplateEngine creates a new template engine
func NewKubernetesTemplateEngine(logger logr.Logger, config ManifestGenerationConfig) *KubernetesTemplateEngine {
	engine := &KubernetesTemplateEngine{
		logger:          logger,
		templates:       make(map[string]*template.Template),
		globalVariables: make(map[string]interface{}),
		functions: template.FuncMap{
			"upper": strings.ToUpper,
			"lower": strings.ToLower,
			"join":  strings.Join,
		},
	}

	// Load and compile templates
	engine.loadTemplates()

	return engine
}

// ProcessTemplate processes a template with given variables
func (e *KubernetesTemplateEngine) ProcessTemplate(templateType, nfType string, variables map[string]interface{}) (string, error) {
	templateName := fmt.Sprintf("%s-%s", templateType, nfType)

	// Try specific template first
	tmpl, exists := e.templates[templateName]
	if !exists {
		// Fall back to generic template
		genericName := fmt.Sprintf("%s-generic", templateType)
		tmpl, exists = e.templates[genericName]
		if !exists {
			return "", fmt.Errorf("template not found: %s", templateName)
		}
	}

	// Merge global variables with provided variables
	mergedVars := make(map[string]interface{})
	for k, v := range e.globalVariables {
		mergedVars[k] = v
	}
	for k, v := range variables {
		mergedVars[k] = v
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, mergedVars); err != nil {
		return "", fmt.Errorf("template execution failed: %w", err)
	}

	return buf.String(), nil
}

// loadTemplates loads all templates
func (e *KubernetesTemplateEngine) loadTemplates() {
	// Define basic templates inline for simplicity
	// In production, these would be loaded from files

	// Deployment template
	deploymentTemplate := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ .Name }}
  namespace: {{ .Namespace }}
  labels:
    {{- range $key, $value := .Labels }}
    {{ $key }}: {{ $value }}
    {{- end }}
spec:
  replicas: {{ .Replicas }}
  selector:
    matchLabels:
      app: {{ .Name }}
  template:
    metadata:
      labels:
        app: {{ .Name }}
    spec:
      containers:
      - name: {{ .Name }}
        image: {{ .Image }}:{{ .Version }}
        {{- if .Resources }}
        resources:
          {{- if .Resources.Requests }}
          requests:
            cpu: "{{ .Resources.Requests.CPU }}"
            memory: "{{ .Resources.Requests.Memory }}"
          {{- end }}
          {{- if .Resources.Limits }}
          limits:
            cpu: "{{ .Resources.Limits.CPU }}"
            memory: "{{ .Resources.Limits.Memory }}"
          {{- end }}
        {{- end }}
        {{- if .Ports }}
        ports:
        {{- range .Ports }}
        - containerPort: {{ .Port }}
          name: {{ .Name }}
        {{- end }}
        {{- end }}
        {{- if .Environment }}
        env:
        {{- range .Environment }}
        - name: {{ .Name }}
          value: "{{ .Value }}"
        {{- end }}
        {{- end }}`

	// Service template
	serviceTemplate := `apiVersion: v1
kind: Service
metadata:
  name: {{ .Name }}-service
  namespace: {{ .Namespace }}
  labels:
    {{- range $key, $value := .Labels }}
    {{ $key }}: {{ $value }}
    {{- end }}
spec:
  selector:
    app: {{ .Name }}
  {{- if .Ports }}
  ports:
  {{- range .Ports }}
  - port: {{ .Port }}
    targetPort: {{ .TargetPort }}
    name: {{ .Name }}
  {{- end }}
  {{- end }}`

	// ConfigMap template
	configMapTemplate := `apiVersion: v1
kind: ConfigMap
metadata:
  name: {{ .Name }}-config
  namespace: {{ .Namespace }}
  labels:
    {{- range $key, $value := .Labels }}
    {{ $key }}: {{ $value }}
    {{- end }}
data:
  {{- range $key, $value := .Configuration }}
  {{ $key }}: "{{ $value }}"
  {{- end }}`

	// Parse and store templates
	e.parseAndStoreTemplate("deployment-generic", deploymentTemplate)
	e.parseAndStoreTemplate("service-generic", serviceTemplate)
	e.parseAndStoreTemplate("configmap-generic", configMapTemplate)
}

// parseAndStoreTemplate parses and stores a template
func (e *KubernetesTemplateEngine) parseAndStoreTemplate(name, templateStr string) {
	tmpl, err := template.New(name).Funcs(e.functions).Parse(templateStr)
	if err != nil {
		e.logger.Error(err, "Failed to parse template", "templateName", name)
		return
	}
	e.templates[name] = tmpl
}

// ManifestValidator handles manifest validation
type ManifestValidator struct {
	logger logr.Logger
	config ManifestGenerationConfig
}

// NewManifestValidator creates a new manifest validator
func NewManifestValidator(logger logr.Logger, config ManifestGenerationConfig) *ManifestValidator {
	return &ManifestValidator{
		logger: logger,
		config: config,
	}
}

// ValidateManifests validates a set of manifests
func (v *ManifestValidator) ValidateManifests(ctx context.Context, manifests map[string]string) error {
	for name, manifest := range manifests {
		if err := v.validateSingleManifest(name, manifest); err != nil {
			return fmt.Errorf("manifest %s validation failed: %w", name, err)
		}
	}
	return nil
}

// validateSingleManifest validates a single manifest
func (v *ManifestValidator) validateSingleManifest(name, manifest string) error {
	// Basic YAML validation
	var obj map[string]interface{}
	if err := yaml.Unmarshal([]byte(manifest), &obj); err != nil {
		return fmt.Errorf("invalid YAML: %w", err)
	}

	// Check required fields
	if _, exists := obj["apiVersion"]; !exists {
		return fmt.Errorf("missing required field: apiVersion")
	}
	if _, exists := obj["kind"]; !exists {
		return fmt.Errorf("missing required field: kind")
	}
	if _, exists := obj["metadata"]; !exists {
		return fmt.Errorf("missing required field: metadata")
	}

	return nil
}

// ManifestPolicyEnforcer handles policy enforcement
type ManifestPolicyEnforcer struct {
	logger logr.Logger
	config ManifestGenerationConfig
}

// NewManifestPolicyEnforcer creates a new policy enforcer
func NewManifestPolicyEnforcer(logger logr.Logger, config ManifestGenerationConfig) *ManifestPolicyEnforcer {
	return &ManifestPolicyEnforcer{
		logger: logger,
		config: config,
	}
}

// EnforcePolicies enforces policies on manifests
func (e *ManifestPolicyEnforcer) EnforcePolicies(ctx context.Context, manifests map[string]string, rules []*ManifestPolicyRule) ([]PolicyResult, map[string]string, error) {
	var results []PolicyResult
	modifiedManifests := make(map[string]string)

	for manifestName, manifestContent := range manifests {
		// Parse manifest
		var manifestObj map[string]interface{}
		if err := yaml.Unmarshal([]byte(manifestContent), &manifestObj); err != nil {
			continue // Skip invalid manifests
		}

		// Check each policy rule
		for _, rule := range rules {
			result := PolicyResult{
				PolicyID:   rule.ID,
				PolicyName: rule.Name,
				Compliant:  true,
			}

			if err := rule.Rule(manifestObj); err != nil {
				result.Compliant = false
				result.Violations = []string{err.Error()}
				result.Action = rule.ViolationAction
			}

			results = append(results, result)
		}
	}

	return results, modifiedManifests, nil
}

// ManifestOptimizer handles manifest optimization
type ManifestOptimizer struct {
	logger logr.Logger
	config ManifestGenerationConfig
}

// NewManifestOptimizer creates a new manifest optimizer
func NewManifestOptimizer(logger logr.Logger, config ManifestGenerationConfig) *ManifestOptimizer {
	return &ManifestOptimizer{
		logger: logger,
		config: config,
	}
}

// OptimizeManifests optimizes a set of manifests
func (o *ManifestOptimizer) OptimizeManifests(ctx context.Context, manifests map[string]string) (map[string]string, error) {
	optimized := make(map[string]string)

	for name, manifest := range manifests {
		if o.config.MinifyManifests {
			manifest = o.minifyManifest(manifest)
		}
		optimized[name] = manifest
	}

	return optimized, nil
}

// minifyManifest removes unnecessary whitespace
func (o *ManifestOptimizer) minifyManifest(manifest string) string {
	// Simple minification - remove extra spaces and empty lines
	lines := strings.Split(manifest, "\n")
	var minified []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			minified = append(minified, trimmed)
		}
	}

	return strings.Join(minified, "\n")
}

// Helm and Kustomize integration stubs
type HelmManifestIntegration struct {
	logger logr.Logger
}

func NewHelmManifestIntegration(logger logr.Logger) *HelmManifestIntegration {
	return &HelmManifestIntegration{logger: logger}
}

type KustomizeManifestIntegration struct {
	logger logr.Logger
}

func NewKustomizeManifestIntegration(logger logr.Logger) *KustomizeManifestIntegration {
	return &KustomizeManifestIntegration{logger: logger}
}

// Initialize helper functions

// initializeManifestTemplates initializes manifest templates
func initializeManifestTemplates() map[string]*ManifestTemplate {
	templates := make(map[string]*ManifestTemplate)

	templates["deployment-generic"] = &ManifestTemplate{
		Name:      "deployment-generic",
		Type:      "deployment",
		Category:  "kubernetes",
		Variables: []string{"Name", "Image", "Version", "Replicas", "Resources", "Ports", "Environment"},
	}

	templates["service-generic"] = &ManifestTemplate{
		Name:      "service-generic",
		Type:      "service",
		Category:  "kubernetes",
		Variables: []string{"Name", "Ports"},
	}

	templates["configmap-generic"] = &ManifestTemplate{
		Name:      "configmap-generic",
		Type:      "configmap",
		Category:  "kubernetes",
		Variables: []string{"Name", "Configuration"},
	}

	return templates
}

// initializeManifestPolicyRules initializes policy rules
func initializeManifestPolicyRules() []*ManifestPolicyRule {
	var rules []*ManifestPolicyRule

	// Security policy - require resource limits
	rules = append(rules, &ManifestPolicyRule{
		ID:       "resource-limits-required",
		Name:     "Resource Limits Required",
		Type:     "security",
		Severity: "warning",
		Rule: func(manifest map[string]interface{}) error {
			kind, _ := manifest["kind"].(string)
			if kind == "Deployment" {
				// Check for resource limits in containers
				spec, ok := manifest["spec"].(map[string]interface{})
				if !ok {
					return fmt.Errorf("missing spec section")
				}

				template, ok := spec["template"].(map[string]interface{})
				if !ok {
					return nil // Skip non-deployment resources
				}

				podSpec, ok := template["spec"].(map[string]interface{})
				if !ok {
					return nil
				}

				containers, ok := podSpec["containers"].([]interface{})
				if !ok {
					return nil
				}

				for _, containerInterface := range containers {
					container, ok := containerInterface.(map[string]interface{})
					if !ok {
						continue
					}

					resources, ok := container["resources"].(map[string]interface{})
					if !ok {
						return fmt.Errorf("container missing resource limits")
					}

					if _, exists := resources["limits"]; !exists {
						return fmt.Errorf("container missing resource limits")
					}
				}
			}
			return nil
		},
		ViolationAction: "warn",
	})

	return rules
}
