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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// ManifestGenerationPhase represents the phase of manifest generation
type ManifestGenerationPhase string

const (
	// ManifestGenerationPhasePending indicates generation is pending
	ManifestGenerationPhasePending ManifestGenerationPhase = "Pending"
	// ManifestGenerationPhaseTemplating indicates template processing is in progress
	ManifestGenerationPhaseTemplating ManifestGenerationPhase = "Templating"
	// ManifestGenerationPhaseGenerating indicates manifest generation is in progress
	ManifestGenerationPhaseGenerating ManifestGenerationPhase = "Generating"
	// ManifestGenerationPhaseValidating indicates validation is in progress
	ManifestGenerationPhaseValidating ManifestGenerationPhase = "Validating"
	// ManifestGenerationPhaseOptimizing indicates optimization is in progress
	ManifestGenerationPhaseOptimizing ManifestGenerationPhase = "Optimizing"
	// ManifestGenerationPhaseCompleted indicates generation is completed
	ManifestGenerationPhaseCompleted ManifestGenerationPhase = "Completed"
	// ManifestGenerationPhaseFailed indicates generation has failed
	ManifestGenerationPhaseFailed ManifestGenerationPhase = "Failed"
)

// TemplateEngine represents the template engine used for generation
type TemplateEngine string

const (
	// TemplateEngineHelm represents Helm templates
	TemplateEngineHelm TemplateEngine = "helm"
	// TemplateEngineKustomize represents Kustomize overlays
	TemplateEngineKustomize TemplateEngine = "kustomize"
	// TemplateEngineKsonnet represents Jsonnet templates
	TemplateEngineKsonnet TemplateEngine = "jsonnet"
	// TemplateEngineGo represents Go templates
	TemplateEngineGo TemplateEngine = "go"
	// TemplateEngineJinja represents Jinja2 templates
	TemplateEngineJinja TemplateEngine = "jinja"
)

// ManifestType represents the type of Kubernetes manifest
type ManifestType string

const (
	// ManifestTypeDeployment represents Deployment manifests
	ManifestTypeDeployment ManifestType = "Deployment"
	// ManifestTypeService represents Service manifests
	ManifestTypeService ManifestType = "Service"
	// ManifestTypeConfigMap represents ConfigMap manifests
	ManifestTypeConfigMap ManifestType = "ConfigMap"
	// ManifestTypeSecret represents Secret manifests
	ManifestTypeSecret ManifestType = "Secret"
	// ManifestTypeIngress represents Ingress manifests
	ManifestTypeIngress ManifestType = "Ingress"
	// ManifestTypePersistentVolumeClaim represents PVC manifests
	ManifestTypePersistentVolumeClaim ManifestType = "PersistentVolumeClaim"
	// ManifestTypeStatefulSet represents StatefulSet manifests
	ManifestTypeStatefulSet ManifestType = "StatefulSet"
	// ManifestTypeDaemonSet represents DaemonSet manifests
	ManifestTypeDaemonSet ManifestType = "DaemonSet"
	// ManifestTypeCustomResource represents Custom Resource manifests
	ManifestTypeCustomResource ManifestType = "CustomResource"
)

// ManifestGenerationSpec defines the desired state of ManifestGeneration
type ManifestGenerationSpec struct {
	// ParentIntentRef references the parent NetworkIntent
	// +kubebuilder:validation:Required
	ParentIntentRef ObjectReference `json:"parentIntentRef"`

	// ResourcePlanRef references the ResourcePlan resource
	// +optional
	ResourcePlanRef *ObjectReference `json:"resourcePlanRef,omitempty"`

	// ResourcePlanInput contains the resource plan for generation
	// +kubebuilder:validation:Required
	// +kubebuilder:pruning:PreserveUnknownFields
	ResourcePlanInput runtime.RawExtension `json:"resourcePlanInput"`

	// TemplateEngine specifies the template engine to use
	// +optional
	// +kubebuilder:default="helm"
	TemplateEngine TemplateEngine `json:"templateEngine,omitempty"`

	// TemplateSource specifies the source of templates
	// +optional
	TemplateSource *TemplateSource `json:"templateSource,omitempty"`

	// GenerationOptions contains options for manifest generation
	// +optional
	GenerationOptions *GenerationOptions `json:"generationOptions,omitempty"`

	// OutputFormat specifies the output format for manifests
	// +optional
	// +kubebuilder:default="yaml"
	// +kubebuilder:validation:Enum=yaml;json
	OutputFormat string `json:"outputFormat,omitempty"`

	// Priority defines generation priority
	// +optional
	// +kubebuilder:default="medium"
	Priority Priority `json:"priority,omitempty"`

	// ValidateManifests enables manifest validation
	// +optional
	// +kubebuilder:default=true
	ValidateManifests *bool `json:"validateManifests,omitempty"`

	// OptimizeManifests enables manifest optimization
	// +optional
	// +kubebuilder:default=true
	OptimizeManifests *bool `json:"optimizeManifests,omitempty"`

	// NamespaceTemplate defines template for namespace creation
	// +optional
	NamespaceTemplate *NamespaceTemplate `json:"namespaceTemplate,omitempty"`

	// SecurityContext defines security context for manifests
	// +optional
	SecurityContext *ManifestSecurityContext `json:"securityContext,omitempty"`
}

// TemplateSource defines the source of templates
type TemplateSource struct {
	// Type specifies the source type
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:Enum=git;helm;oci;configmap;builtin
	Type string `json:"type"`

	// GitSource for Git-based templates
	// +optional
	GitSource *GitTemplateSource `json:"gitSource,omitempty"`

	// HelmSource for Helm chart templates
	// +optional
	HelmSource *HelmTemplateSource `json:"helmSource,omitempty"`

	// OCISource for OCI registry templates
	// +optional
	OCISource *OCITemplateSource `json:"ociSource,omitempty"`

	// ConfigMapSource for ConfigMap-based templates
	// +optional
	ConfigMapSource *ConfigMapTemplateSource `json:"configMapSource,omitempty"`

	// BuiltinTemplates for built-in templates
	// +optional
	BuiltinTemplates []string `json:"builtinTemplates,omitempty"`
}

// GitTemplateSource defines Git-based template source
type GitTemplateSource struct {
	// URL of the Git repository
	// +kubebuilder:validation:Required
	URL string `json:"url"`

	// Branch to use
	// +optional
	// +kubebuilder:default="main"
	Branch string `json:"branch,omitempty"`

	// Tag to use (takes precedence over branch)
	// +optional
	Tag string `json:"tag,omitempty"`

	// Path within the repository
	// +optional
	// +kubebuilder:default="."
	Path string `json:"path,omitempty"`

	// SecretRef for authentication
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// HelmTemplateSource defines Helm chart template source
type HelmTemplateSource struct {
	// Repository URL
	// +kubebuilder:validation:Required
	Repository string `json:"repository"`

	// Chart name
	// +kubebuilder:validation:Required
	Chart string `json:"chart"`

	// Version of the chart
	// +optional
	Version string `json:"version,omitempty"`

	// Values for the chart
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Values runtime.RawExtension `json:"values,omitempty"`

	// SecretRef for authentication
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// OCITemplateSource defines OCI registry template source
type OCITemplateSource struct {
	// Registry URL
	// +kubebuilder:validation:Required
	Registry string `json:"registry"`

	// Repository name
	// +kubebuilder:validation:Required
	Repository string `json:"repository"`

	// Tag or digest
	// +optional
	// +kubebuilder:default="latest"
	Tag string `json:"tag,omitempty"`

	// SecretRef for authentication
	// +optional
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// ConfigMapTemplateSource defines ConfigMap-based template source
type ConfigMapTemplateSource struct {
	// Name of the ConfigMap
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// Namespace of the ConfigMap
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Keys to use from the ConfigMap
	// +optional
	Keys []string `json:"keys,omitempty"`
}

// SecretReference defines a reference to a Secret
// SecretReference is defined in audittrail_types.go to avoid duplication

// GenerationOptions contains options for manifest generation
type GenerationOptions struct {
	// IncludeNamespaces generates namespace manifests
	// +optional
	// +kubebuilder:default=true
	IncludeNamespaces *bool `json:"includeNamespaces,omitempty"`

	// IncludeRBAC generates RBAC manifests
	// +optional
	// +kubebuilder:default=true
	IncludeRBAC *bool `json:"includeRBAC,omitempty"`

	// IncludeNetworkPolicies generates NetworkPolicy manifests
	// +optional
	// +kubebuilder:default=false
	IncludeNetworkPolicies *bool `json:"includeNetworkPolicies,omitempty"`

	// IncludePodSecurityPolicies generates PodSecurityPolicy manifests
	// +optional
	// +kubebuilder:default=false
	IncludePodSecurityPolicies *bool `json:"includePodSecurityPolicies,omitempty"`

	// GenerateMonitoring includes monitoring-related manifests
	// +optional
	// +kubebuilder:default=true
	GenerateMonitoring *bool `json:"generateMonitoring,omitempty"`

	// GenerateHealthChecks includes health check configurations
	// +optional
	// +kubebuilder:default=true
	GenerateHealthChecks *bool `json:"generateHealthChecks,omitempty"`

	// ResourceNaming defines naming conventions
	// +optional
	ResourceNaming *ResourceNamingOptions `json:"resourceNaming,omitempty"`

	// Labels to apply to all generated manifests
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to apply to all generated manifests
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// ResourceNamingOptions defines naming conventions for resources
type ResourceNamingOptions struct {
	// Prefix for all resource names
	// +optional
	Prefix string `json:"prefix,omitempty"`

	// Suffix for all resource names
	// +optional
	Suffix string `json:"suffix,omitempty"`

	// IncludeIntentName includes the intent name in resource names
	// +optional
	// +kubebuilder:default=true
	IncludeIntentName *bool `json:"includeIntentName,omitempty"`

	// IncludeComponent includes the component name in resource names
	// +optional
	// +kubebuilder:default=true
	IncludeComponent *bool `json:"includeComponent,omitempty"`

	// MaxLength limits the maximum name length
	// +optional
	// +kubebuilder:validation:Minimum=1
	// +kubebuilder:validation:Maximum=253
	// +kubebuilder:default=63
	MaxLength *int `json:"maxLength,omitempty"`
}

// NamespaceTemplate defines template for namespace creation
type NamespaceTemplate struct {
	// Name template for the namespace
	// +optional
	NameTemplate string `json:"nameTemplate,omitempty"`

	// Labels to apply to the namespace
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Annotations to apply to the namespace
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// ResourceQuota for the namespace
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	ResourceQuota runtime.RawExtension `json:"resourceQuota,omitempty"`

	// NetworkPolicy for the namespace
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	NetworkPolicy runtime.RawExtension `json:"networkPolicy,omitempty"`
}

// ManifestSecurityContext defines security context for manifests
type ManifestSecurityContext struct {
	// RunAsNonRoot ensures containers run as non-root
	// +optional
	// +kubebuilder:default=true
	RunAsNonRoot *bool `json:"runAsNonRoot,omitempty"`

	// ReadOnlyRootFilesystem makes root filesystem read-only
	// +optional
	// +kubebuilder:default=true
	ReadOnlyRootFilesystem *bool `json:"readOnlyRootFilesystem,omitempty"`

	// AllowPrivilegeEscalation controls privilege escalation
	// +optional
	// +kubebuilder:default=false
	AllowPrivilegeEscalation *bool `json:"allowPrivilegeEscalation,omitempty"`

	// DropCapabilities specifies capabilities to drop
	// +optional
	DropCapabilities []string `json:"dropCapabilities,omitempty"`

	// AddCapabilities specifies capabilities to add
	// +optional
	AddCapabilities []string `json:"addCapabilities,omitempty"`

	// SeccompProfile specifies seccomp profile
	// +optional
	SeccompProfile string `json:"seccompProfile,omitempty"`

	// SeLinuxOptions specifies SELinux options
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	SeLinuxOptions runtime.RawExtension `json:"seLinuxOptions,omitempty"`
}

// ManifestGenerationStatus defines the observed state of ManifestGeneration
type ManifestGenerationStatus struct {
	// Phase represents the current generation phase
	// +optional
	Phase ManifestGenerationPhase `json:"phase,omitempty"`

	// Conditions represent the latest available observations
	// +optional
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// GenerationStartTime indicates when generation started
	// +optional
	GenerationStartTime *metav1.Time `json:"generationStartTime,omitempty"`

	// GenerationCompletionTime indicates when generation completed
	// +optional
	GenerationCompletionTime *metav1.Time `json:"generationCompletionTime,omitempty"`

	// GeneratedManifests contains the generated Kubernetes manifests
	// +optional
	GeneratedManifests map[string]string `json:"generatedManifests,omitempty"`

	// ManifestSummary provides a summary of generated manifests
	// +optional
	ManifestSummary *ManifestSummary `json:"manifestSummary,omitempty"`

	// ValidationResults contains validation results
	// +optional
	ValidationResults []ManifestValidationResult `json:"validationResults,omitempty"`

	// OptimizationResults contains optimization results
	// +optional
	OptimizationResults []ManifestOptimizationResult `json:"optimizationResults,omitempty"`

	// SecurityAnalysis contains security analysis results
	// +optional
	SecurityAnalysis *SecurityAnalysisResult `json:"securityAnalysis,omitempty"`

	// ResourceReferences contains references to generated resources
	// +optional
	ResourceReferences []GeneratedResourceReference `json:"resourceReferences,omitempty"`

	// GenerationDuration represents total generation time
	// +optional
	GenerationDuration *metav1.Duration `json:"generationDuration,omitempty"`

	// TemplateInfo contains information about used templates
	// +optional
	TemplateInfo *TemplateInfo `json:"templateInfo,omitempty"`

	// RetryCount tracks retry attempts
	// +optional
	RetryCount int32 `json:"retryCount,omitempty"`

	// QualityScore represents the quality of generated manifests
	// +optional
	QualityScore *float64 `json:"qualityScore,omitempty"`

	// ObservedGeneration reflects the generation observed
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`
}

// ManifestSummary provides a summary of generated manifests
type ManifestSummary struct {
	// TotalManifests is the total number of generated manifests
	TotalManifests int32 `json:"totalManifests"`

	// ManifestsByType breaks down manifests by type
	ManifestsByType map[ManifestType]int32 `json:"manifestsByType"`

	// ManifestsByNamespace breaks down manifests by namespace
	// +optional
	ManifestsByNamespace map[string]int32 `json:"manifestsByNamespace,omitempty"`

	// ManifestsByComponent breaks down manifests by component
	// +optional
	ManifestsByComponent map[string]int32 `json:"manifestsByComponent,omitempty"`

	// TotalSize is the total size of all manifests in bytes
	// +optional
	TotalSize *int64 `json:"totalSize,omitempty"`

	// GeneratedNamespaces lists generated namespaces
	// +optional
	GeneratedNamespaces []string `json:"generatedNamespaces,omitempty"`
}

// ManifestValidationResult represents the result of manifest validation
type ManifestValidationResult struct {
	// ManifestName identifies the validated manifest
	ManifestName string `json:"manifestName"`

	// ManifestType specifies the type of manifest
	ManifestType ManifestType `json:"manifestType"`

	// Valid indicates if the manifest is valid
	Valid bool `json:"valid"`

	// Errors contains validation errors
	// +optional
	Errors []string `json:"errors,omitempty"`

	// Warnings contains validation warnings
	// +optional
	Warnings []string `json:"warnings,omitempty"`

	// ValidationRules lists which validation rules were applied
	// +optional
	ValidationRules []string `json:"validationRules,omitempty"`

	// ValidatedAt timestamp
	ValidatedAt metav1.Time `json:"validatedAt"`
}

// ManifestOptimizationResult represents the result of manifest optimization
type ManifestOptimizationResult struct {
	// ManifestName identifies the optimized manifest
	ManifestName string `json:"manifestName"`

	// OptimizationType specifies the type of optimization
	OptimizationType string `json:"optimizationType"`

	// Applied indicates if optimization was applied
	Applied bool `json:"applied"`

	// Description describes the optimization
	// +optional
	Description string `json:"description,omitempty"`

	// ImprovementPercent quantifies the improvement
	// +optional
	ImprovementPercent *float64 `json:"improvementPercent,omitempty"`

	// Changes lists the changes made
	// +optional
	Changes []string `json:"changes,omitempty"`

	// OptimizedAt timestamp
	OptimizedAt metav1.Time `json:"optimizedAt"`
}

// SecurityAnalysisResult contains security analysis results
type SecurityAnalysisResult struct {
	// OverallScore is the overall security score (0.0-1.0)
	OverallScore float64 `json:"overallScore"`

	// SecurityIssues lists identified security issues
	// +optional
	SecurityIssues []SecurityIssue `json:"securityIssues,omitempty"`

	// ComplianceResults contains compliance check results
	// +optional
	ComplianceResults []SecurityComplianceResult `json:"complianceResults,omitempty"`

	// Recommendations provides security recommendations
	// +optional
	Recommendations []string `json:"recommendations,omitempty"`

	// AnalyzedAt timestamp
	AnalyzedAt metav1.Time `json:"analyzedAt"`
}

// SecurityIssue represents a security issue in manifests
type SecurityIssue struct {
	// Type of security issue
	Type string `json:"type"`

	// Severity level (Critical, High, Medium, Low)
	Severity string `json:"severity"`

	// Description of the issue
	Description string `json:"description"`

	// AffectedManifests lists affected manifests
	// +optional
	AffectedManifests []string `json:"affectedManifests,omitempty"`

	// Remediation suggests how to fix the issue
	// +optional
	Remediation string `json:"remediation,omitempty"`

	// CVSS score if applicable
	// +optional
	CVSSScore *float64 `json:"cvssScore,omitempty"`
}

// SecurityComplianceResult represents compliance check results
type SecurityComplianceResult struct {
	// Standard being checked (PCI-DSS, SOC2, etc.)
	Standard string `json:"standard"`

	// Version of the standard
	// +optional
	Version string `json:"version,omitempty"`

	// Compliant indicates compliance status
	Compliant bool `json:"compliant"`

	// Violations lists compliance violations
	// +optional
	Violations []string `json:"violations,omitempty"`

	// Score represents compliance score (0.0-1.0)
	// +optional
	Score *float64 `json:"score,omitempty"`
}

// GeneratedResourceReference represents a reference to a generated resource
type GeneratedResourceReference struct {
	// Name of the resource
	Name string `json:"name"`

	// Kind of the resource
	Kind string `json:"kind"`

	// APIVersion of the resource
	APIVersion string `json:"apiVersion"`

	// Namespace of the resource
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Component this resource belongs to
	// +optional
	Component string `json:"component,omitempty"`

	// ManifestFile where this resource is defined
	// +optional
	ManifestFile string `json:"manifestFile,omitempty"`

	// Dependencies lists resource dependencies
	// +optional
	Dependencies []string `json:"dependencies,omitempty"`
}

// TemplateInfo contains information about used templates
type TemplateInfo struct {
	// Engine used for templating
	Engine TemplateEngine `json:"engine"`

	// Source information
	Source TemplateSource `json:"source"`

	// TemplatesUsed lists the templates that were used
	// +optional
	TemplatesUsed []string `json:"templatesUsed,omitempty"`

	// TemplateVersion indicates the version of templates used
	// +optional
	TemplateVersion string `json:"templateVersion,omitempty"`

	// Variables contains the template variables used
	// +optional
	// +kubebuilder:pruning:PreserveUnknownFields
	Variables runtime.RawExtension `json:"variables,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:printcolumn:name="Parent Intent",type=string,JSONPath=`.spec.parentIntentRef.name`
//+kubebuilder:printcolumn:name="Phase",type=string,JSONPath=`.status.phase`
//+kubebuilder:printcolumn:name="Engine",type=string,JSONPath=`.spec.templateEngine`
//+kubebuilder:printcolumn:name="Manifests",type=integer,JSONPath=`.status.manifestSummary.totalManifests`
//+kubebuilder:printcolumn:name="Quality",type=string,JSONPath=`.status.qualityScore`
//+kubebuilder:printcolumn:name="Security",type=string,JSONPath=`.status.securityAnalysis.overallScore`
//+kubebuilder:printcolumn:name="Age",type=date,JSONPath=`.metadata.creationTimestamp`
//+kubebuilder:resource:shortName=mg;manigen
//+kubebuilder:storageversion

// ManifestGeneration is the Schema for the manifestgenerations API
type ManifestGeneration struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ManifestGenerationSpec   `json:"spec,omitempty"`
	Status ManifestGenerationStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// ManifestGenerationList contains a list of ManifestGeneration
type ManifestGenerationList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ManifestGeneration `json:"items"`
}

// GetParentIntentName returns the name of the parent NetworkIntent
func (mg *ManifestGeneration) GetParentIntentName() string {
	return mg.Spec.ParentIntentRef.Name
}

// GetParentIntentNamespace returns the namespace of the parent NetworkIntent
func (mg *ManifestGeneration) GetParentIntentNamespace() string {
	if mg.Spec.ParentIntentRef.Namespace != "" {
		return mg.Spec.ParentIntentRef.Namespace
	}
	return mg.Namespace
}

// IsGenerationComplete returns true if generation is complete
func (mg *ManifestGeneration) IsGenerationComplete() bool {
	return mg.Status.Phase == ManifestGenerationPhaseCompleted
}

// IsGenerationFailed returns true if generation has failed
func (mg *ManifestGeneration) IsGenerationFailed() bool {
	return mg.Status.Phase == ManifestGenerationPhaseFailed
}

// GetGeneratedManifestCount returns the number of generated manifests
func (mg *ManifestGeneration) GetGeneratedManifestCount() int32 {
	if mg.Status.ManifestSummary != nil {
		return mg.Status.ManifestSummary.TotalManifests
	}
	return 0
}

// HasValidationErrors returns true if there are validation errors
func (mg *ManifestGeneration) HasValidationErrors() bool {
	for _, result := range mg.Status.ValidationResults {
		if !result.Valid {
			return true
		}
	}
	return false
}

// GetSecurityScore returns the overall security score
func (mg *ManifestGeneration) GetSecurityScore() float64 {
	if mg.Status.SecurityAnalysis != nil {
		return mg.Status.SecurityAnalysis.OverallScore
	}
	return 0.0
}

// ShouldValidateManifests returns true if manifest validation is enabled
func (mg *ManifestGeneration) ShouldValidateManifests() bool {
	if mg.Spec.ValidateManifests == nil {
		return true
	}
	return *mg.Spec.ValidateManifests
}

// ShouldOptimizeManifests returns true if manifest optimization is enabled
func (mg *ManifestGeneration) ShouldOptimizeManifests() bool {
	if mg.Spec.OptimizeManifests == nil {
		return true
	}
	return *mg.Spec.OptimizeManifests
}

func init() {
	SchemeBuilder.Register(&ManifestGeneration{}, &ManifestGenerationList{})
}
