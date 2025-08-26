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
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/errors"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// PackageCatalogConfig defines configuration for package catalog
type PackageCatalogConfig struct {
	CatalogRepository  string        `json:"catalogRepository" yaml:"catalogRepository"`
	BlueprintDirectory string        `json:"blueprintDirectory" yaml:"blueprintDirectory"`
	VariantDirectory   string        `json:"variantDirectory" yaml:"variantDirectory"`
	TemplateDirectory  string        `json:"templateDirectory" yaml:"templateDirectory"`
	CacheTimeout       time.Duration `json:"cacheTimeout" yaml:"cacheTimeout"`
	MaxCachedItems     int           `json:"maxCachedItems" yaml:"maxCachedItems"`
	EnableVersioning   bool          `json:"enableVersioning" yaml:"enableVersioning"`
	EnableMetrics      bool          `json:"enableMetrics" yaml:"enableMetrics"`
	EnableTracing      bool          `json:"enableTracing" yaml:"enableTracing"`
}

// PackageCatalogMetrics provides catalog metrics
type PackageCatalogMetrics struct {
	BlueprintQueries    *prometheus.CounterVec
	VariantCreations    *prometheus.CounterVec
	CatalogOperations   *prometheus.CounterVec
	CacheHitRate        prometheus.Counter
	CacheMissRate       prometheus.Counter
	BlueprintLoadTime   *prometheus.HistogramVec
	VariantCreationTime *prometheus.HistogramVec
	CatalogSize         *prometheus.GaugeVec
}

// PackageDependency represents a package dependency
type PackageDependency struct {
	Name       string            `json:"name"`
	Repository string            `json:"repository"`
	Version    string            `json:"version"`
	Optional   bool              `json:"optional"`
	Condition  string            `json:"condition,omitempty"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// ParameterDefinition defines a blueprint parameter
type ParameterDefinition struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Default     interface{} `json:"default,omitempty"`
	Required    bool        `json:"required"`
	Validation  string      `json:"validation,omitempty"`
	Constraints []string    `json:"constraints,omitempty"`
}

// ValidationRule defines validation rules for blueprints
type ValidationRule struct {
	Name       string            `json:"name"`
	Type       string            `json:"type"`
	Expression string            `json:"expression"`
	Message    string            `json:"message"`
	Severity   string            `json:"severity"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// ResourceTemplate defines a resource template in a blueprint
type ResourceTemplate struct {
	Name       string                 `json:"name"`
	Kind       string                 `json:"kind"`
	APIVersion string                 `json:"apiVersion"`
	Template   map[string]interface{} `json:"template"`
	Conditions []string               `json:"conditions,omitempty"`
}

// FunctionDefinition defines a KRM function in a blueprint
type FunctionDefinition struct {
	Name     string                 `json:"name"`
	Image    string                 `json:"image"`
	Config   map[string]interface{} `json:"config,omitempty"`
	Stage    string                 `json:"stage"`
	Required bool                   `json:"required"`
}

// SpecializationSpec defines how a blueprint can be specialized
type SpecializationSpec struct {
	Parameters  []ParameterDefinition `json:"parameters"`
	Templates   []TemplateDefinition  `json:"templates"`
	Functions   []FunctionDefinition  `json:"functions"`
	Validations []ValidationRule      `json:"validations"`
	Constraints []string              `json:"constraints,omitempty"`
}

// TemplateDefinition defines a specialization template
type TemplateDefinition struct {
	Name       string                 `json:"name"`
	Type       string                 `json:"type"`
	Template   map[string]interface{} `json:"template"`
	Conditions []string               `json:"conditions,omitempty"`
}

// ClusterContext provides cluster-specific context for specialization
type ClusterContext struct {
	Name         string              `json:"name"`
	Region       string              `json:"region"`
	Zone         string              `json:"zone"`
	Capabilities []ClusterCapability `json:"capabilities"`
	Resources    *ClusterResources   `json:"resources,omitempty"`
	Labels       map[string]string   `json:"labels,omitempty"`
	Annotations  map[string]string   `json:"annotations,omitempty"`
}

// ClusterResources represents available cluster resources
type ClusterResources struct {
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Storage string `json:"storage"`
	GPU     string `json:"gpu,omitempty"`
	Network string `json:"network,omitempty"`
}

// NetworkSliceSpec defines network slice specifications
type NetworkSliceSpec struct {
	SliceID   string                 `json:"sliceId"`
	SliceType string                 `json:"sliceType"`
	SLA       *SLASpecification      `json:"sla,omitempty"`
	QoS       *QoSSpecification      `json:"qos,omitempty"`
	Resources *ResourceRequirements  `json:"resources,omitempty"`
	Security  *SecuritySpecification `json:"security,omitempty"`
}

// SLASpecification defines SLA requirements
type SLASpecification struct {
	Latency      string `json:"latency"`
	Throughput   string `json:"throughput"`
	Reliability  string `json:"reliability"`
	Availability string `json:"availability"`
}

// QoSSpecification defines QoS requirements
type QoSSpecification struct {
	Priority int    `json:"priority"`
	QCI      int    `json:"qci"`
	ARP      int    `json:"arp"`
	GBR      string `json:"gbr,omitempty"`
	MBR      string `json:"mbr,omitempty"`
}

// ResourceRequirements defines resource requirements
type ResourceRequirements struct {
	CPU     string `json:"cpu"`
	Memory  string `json:"memory"`
	Storage string `json:"storage"`
	Network string `json:"network"`
}

// SecuritySpecification defines security requirements
type SecuritySpecification struct {
	Encryption     bool     `json:"encryption"`
	Authentication string   `json:"authentication"`
	Authorization  string   `json:"authorization"`
	Policies       []string `json:"policies,omitempty"`
}

// ORANComplianceSpec defines O-RAN compliance requirements
type ORANComplianceSpec struct {
	Interfaces     []ORANInterface     `json:"interfaces"`
	Validations    []ComplianceRule    `json:"validations"`
	Certifications []Certification     `json:"certifications,omitempty"`
	Standards      []StandardReference `json:"standards,omitempty"`
}

// ORANInterface defines an O-RAN interface specification
type ORANInterface struct {
	Name    string                 `json:"name"`
	Type    string                 `json:"type"`
	Version string                 `json:"version"`
	Config  map[string]interface{} `json:"config,omitempty"`
	Enabled bool                   `json:"enabled"`
}

// ComplianceRule defines a compliance validation rule
type ComplianceRule struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Standard   string `json:"standard"`
	Expression string `json:"expression"`
	Required   bool   `json:"required"`
}

// Certification defines certification requirements
type Certification struct {
	Name      string    `json:"name"`
	Authority string    `json:"authority"`
	Version   string    `json:"version"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
}

// StandardReference defines standard compliance references
type StandardReference struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Section string `json:"section,omitempty"`
	URL     string `json:"url,omitempty"`
}

// SecurityPolicySpec defines security policy requirements
type SecurityPolicySpec struct {
	Authentication *AuthenticationPolicy `json:"authentication,omitempty"`
	Authorization  *AuthorizationPolicy  `json:"authorization,omitempty"`
	NetworkPolicy  *NetworkPolicySpec    `json:"networkPolicy,omitempty"`
	PodSecurity    *PodSecurityPolicy    `json:"podSecurity,omitempty"`
}

// AuthenticationPolicy defines authentication requirements
type AuthenticationPolicy struct {
	Method   string            `json:"method"`
	Config   map[string]string `json:"config,omitempty"`
	Required bool              `json:"required"`
}

// AuthorizationPolicy defines authorization requirements
type AuthorizationPolicy struct {
	RBAC     *RBACPolicy  `json:"rbac,omitempty"`
	Policies []PolicyRule `json:"policies,omitempty"`
}

// RBACPolicy defines RBAC requirements
type RBACPolicy struct {
	Roles        []string `json:"roles,omitempty"`
	RoleBindings []string `json:"roleBindings,omitempty"`
	Subjects     []string `json:"subjects,omitempty"`
}

// PolicyRule defines a policy rule
type PolicyRule struct {
	Name      string   `json:"name"`
	Resources []string `json:"resources"`
	Verbs     []string `json:"verbs"`
	Subjects  []string `json:"subjects"`
}

// NetworkPolicySpec defines network policy requirements
type NetworkPolicySpec struct {
	Ingress []NetworkRule `json:"ingress,omitempty"`
	Egress  []NetworkRule `json:"egress,omitempty"`
}

// NetworkRule defines a network policy rule
type NetworkRule struct {
	From  []NetworkPeer `json:"from,omitempty"`
	To    []NetworkPeer `json:"to,omitempty"`
	Ports []NetworkPort `json:"ports,omitempty"`
}

// NetworkPeer defines a network peer
type NetworkPeer struct {
	PodSelector       map[string]string `json:"podSelector,omitempty"`
	NamespaceSelector map[string]string `json:"namespaceSelector,omitempty"`
	IPBlock           *IPBlock          `json:"ipBlock,omitempty"`
}

// IPBlock defines an IP block
type IPBlock struct {
	CIDR   string   `json:"cidr"`
	Except []string `json:"except,omitempty"`
}

// NetworkPort defines a network port
type NetworkPort struct {
	Protocol string `json:"protocol"`
	Port     int32  `json:"port"`
}

// PodSecurityPolicy defines pod security requirements
type PodSecurityPolicy struct {
	RunAsUser    *int64   `json:"runAsUser,omitempty"`
	RunAsGroup   *int64   `json:"runAsGroup,omitempty"`
	ReadOnlyRoot bool     `json:"readOnlyRoot"`
	Privileged   bool     `json:"privileged"`
	Capabilities []string `json:"capabilities,omitempty"`
}

// PlacementPolicySpec defines placement policy requirements
type PlacementPolicySpec struct {
	NodeSelector    map[string]string `json:"nodeSelector,omitempty"`
	NodeAffinity    *NodeAffinity     `json:"nodeAffinity,omitempty"`
	PodAffinity     *PodAffinity      `json:"podAffinity,omitempty"`
	PodAntiAffinity *PodAffinity      `json:"podAntiAffinity,omitempty"`
	Tolerations     []Toleration      `json:"tolerations,omitempty"`
	TopologySpread  []TopologySpread  `json:"topologySpread,omitempty"`
}

// NodeAffinity defines node affinity requirements
type NodeAffinity struct {
	RequiredTerms  []NodeSelectorTerm `json:"requiredTerms,omitempty"`
	PreferredTerms []NodeSelectorTerm `json:"preferredTerms,omitempty"`
}

// NodeSelectorTerm defines a node selector term
type NodeSelectorTerm struct {
	MatchExpressions []NodeSelectorRequirement `json:"matchExpressions,omitempty"`
	MatchFields      []NodeSelectorRequirement `json:"matchFields,omitempty"`
}

// NodeSelectorRequirement defines a node selector requirement
type NodeSelectorRequirement struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values,omitempty"`
}

// PodAffinity defines pod affinity requirements
type PodAffinity struct {
	RequiredTerms  []PodAffinityTerm `json:"requiredTerms,omitempty"`
	PreferredTerms []PodAffinityTerm `json:"preferredTerms,omitempty"`
}

// PodAffinityTerm defines a pod affinity term
type PodAffinityTerm struct {
	LabelSelector     map[string]string `json:"labelSelector,omitempty"`
	NamespaceSelector map[string]string `json:"namespaceSelector,omitempty"`
	TopologyKey       string            `json:"topologyKey"`
}

// Toleration defines a toleration
type Toleration struct {
	Key      string `json:"key"`
	Operator string `json:"operator"`
	Value    string `json:"value,omitempty"`
	Effect   string `json:"effect"`
}

// TopologySpread defines topology spread constraints
type TopologySpread struct {
	MaxSkew           int32             `json:"maxSkew"`
	TopologyKey       string            `json:"topologyKey"`
	WhenUnsatisfiable string            `json:"whenUnsatisfiable"`
	LabelSelector     map[string]string `json:"labelSelector,omitempty"`
}

// Default configuration
var DefaultPackageCatalogConfig = &PackageCatalogConfig{
	CatalogRepository:  "nephoran-catalog",
	BlueprintDirectory: "blueprints",
	VariantDirectory:   "variants",
	TemplateDirectory:  "templates",
	CacheTimeout:       30 * time.Minute,
	MaxCachedItems:     1000,
	EnableVersioning:   true,
	EnableMetrics:      true,
	EnableTracing:      true,
}

// NewNephioPackageCatalog creates a new package catalog
func NewNephioPackageCatalog(
	client client.Client,
	porchClient porch.PorchClient,
	config *PackageCatalogConfig,
) (*NephioPackageCatalog, error) {
	if config == nil {
		config = DefaultPackageCatalogConfig
	}

	// Initialize metrics
	metrics := &PackageCatalogMetrics{
		BlueprintQueries: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_catalog_blueprint_queries_total",
				Help: "Total number of blueprint queries",
			},
			[]string{"intent_type", "status"},
		),
		VariantCreations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_catalog_variant_creations_total",
				Help: "Total number of package variant creations",
			},
			[]string{"blueprint", "cluster", "status"},
		),
		CatalogOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_catalog_operations_total",
				Help: "Total number of catalog operations",
			},
			[]string{"operation", "status"},
		),
		CacheHitRate: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "nephio_catalog_cache_hits_total",
				Help: "Total number of cache hits",
			},
		),
		CacheMissRate: promauto.NewCounter(
			prometheus.CounterOpts{
				Name: "nephio_catalog_cache_misses_total",
				Help: "Total number of cache misses",
			},
		),
		BlueprintLoadTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nephio_catalog_blueprint_load_duration_seconds",
				Help:    "Time taken to load blueprints",
				Buckets: prometheus.ExponentialBuckets(0.1, 2, 8),
			},
			[]string{"blueprint", "repository"},
		),
		VariantCreationTime: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nephio_catalog_variant_creation_duration_seconds",
				Help:    "Time taken to create package variants",
				Buckets: prometheus.ExponentialBuckets(1, 2, 8),
			},
			[]string{"blueprint", "cluster"},
		),
		CatalogSize: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nephio_catalog_items",
				Help: "Number of items in catalog",
			},
			[]string{"type"},
		),
	}

	catalog := &NephioPackageCatalog{
		client:  client,
		config:  config,
		metrics: metrics,
		tracer:  otel.Tracer("nephio-package-catalog"),
	}

	// Initialize catalog with standard blueprints
	if err := catalog.initializeStandardBlueprints(); err != nil {
		return nil, fmt.Errorf("failed to initialize standard blueprints: %w", err)
	}

	return catalog, nil
}

// FindBlueprintForIntent finds the best blueprint for a given intent
func (npc *NephioPackageCatalog) FindBlueprintForIntent(ctx context.Context, intent *v1.NetworkIntent) (*BlueprintPackage, error) {
	ctx, span := npc.tracer.Start(ctx, "find-blueprint-for-intent")
	defer span.End()

	logger := log.FromContext(ctx).WithName("package-catalog").WithValues(
		"intentType", string(intent.Spec.IntentType),
		"targetComponents", intent.Spec.TargetComponents,
	)

	startTime := time.Now()
	defer func() {
		npc.metrics.BlueprintLoadTime.WithLabelValues(
			"query", npc.config.CatalogRepository,
		).Observe(time.Since(startTime).Seconds())
	}()

	// Check cache first
	cacheKey := fmt.Sprintf("blueprint:%s:%v", intent.Spec.IntentType, intent.Spec.TargetComponents)
	if cached, found := npc.blueprints.Load(cacheKey); found {
		npc.metrics.CacheHitRate.Inc()
		if blueprint, ok := cached.(*BlueprintPackage); ok {
			span.SetAttributes(
				attribute.String("blueprint.name", blueprint.Name),
				attribute.String("blueprint.source", "cache"),
			)
			return blueprint, nil
		}
	}

	npc.metrics.CacheMissRate.Inc()

	// Find matching blueprints
	var candidates []*BlueprintPackage

	npc.blueprints.Range(func(key, value interface{}) bool {
		if blueprint, ok := value.(*BlueprintPackage); ok {
			// Check if blueprint supports this intent type
			for _, intentType := range blueprint.IntentTypes {
				if intentType == intent.Spec.IntentType {
					candidates = append(candidates, blueprint)
					break
				}
			}
		}
		return true
	})

	if len(candidates) == 0 {
		npc.metrics.BlueprintQueries.WithLabelValues(
			string(intent.Spec.IntentType), "not_found",
		).Inc()
		return nil, fmt.Errorf("no blueprint found for intent type: %s", intent.Spec.IntentType)
	}

	// Select best candidate (for now, just pick the first one)
	// In a real implementation, this would use scoring based on intent requirements
	bestBlueprint := candidates[0]

	// Cache the result
	npc.blueprints.Store(cacheKey, bestBlueprint)

	npc.metrics.BlueprintQueries.WithLabelValues(
		string(intent.Spec.IntentType), "found",
	).Inc()

	logger.Info("Found blueprint for intent",
		"blueprint", bestBlueprint.Name,
		"version", bestBlueprint.Version,
	)

	span.SetAttributes(
		attribute.String("blueprint.name", bestBlueprint.Name),
		attribute.String("blueprint.version", bestBlueprint.Version),
		attribute.String("blueprint.source", "query"),
		attribute.Int("candidates.count", len(candidates)),
	)

	return bestBlueprint, nil
}

// CreatePackageVariant creates a specialized package variant for a target cluster
func (npc *NephioPackageCatalog) CreatePackageVariant(ctx context.Context, blueprint *BlueprintPackage, specialization *SpecializationRequest) (*PackageVariant, error) {
	ctx, span := npc.tracer.Start(ctx, "create-package-variant")
	defer span.End()

	logger := log.FromContext(ctx).WithName("package-variant").WithValues(
		"blueprint", blueprint.Name,
		"cluster", specialization.ClusterContext.Name,
	)

	startTime := time.Now()
	defer func() {
		npc.metrics.VariantCreationTime.WithLabelValues(
			blueprint.Name, specialization.ClusterContext.Name,
		).Observe(time.Since(startTime).Seconds())
	}()

	// Generate variant name
	variantName := fmt.Sprintf("%s-%s-%s",
		blueprint.Name,
		specialization.ClusterContext.Name,
		time.Now().Format("20060102-150405"))

	// Create package variant
	variant := &PackageVariant{
		Name:           variantName,
		Blueprint:      blueprint,
		Specialization: specialization,
		Status:         PackageVariantStatusSpecializing,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Find target cluster
	cluster, err := npc.findWorkloadCluster(ctx, specialization.ClusterContext.Name)
	if err != nil {
		span.RecordError(err)
		npc.metrics.VariantCreations.WithLabelValues(
			blueprint.Name, specialization.ClusterContext.Name, "failed",
		).Inc()
		return nil, fmt.Errorf("failed to find target cluster: %w", err)
	}
	variant.TargetCluster = cluster

	// Create specialized package revision
	packageRevision, err := npc.createSpecializedPackageRevision(ctx, variant)
	if err != nil {
		span.RecordError(err)
		npc.metrics.VariantCreations.WithLabelValues(
			blueprint.Name, specialization.ClusterContext.Name, "failed",
		).Inc()
		return nil, fmt.Errorf("failed to create specialized package revision: %w", err)
	}

	variant.PackageRevision = packageRevision
	variant.Status = PackageVariantStatusReady
	variant.UpdatedAt = time.Now()

	// Store in cache
	npc.variants.Store(variantName, variant)

	npc.metrics.VariantCreations.WithLabelValues(
		blueprint.Name, specialization.ClusterContext.Name, "success",
	).Inc()

	logger.Info("Created package variant",
		"variant", variantName,
		"packageRevision", packageRevision.Name,
	)

	span.SetAttributes(
		attribute.String("variant.name", variantName),
		attribute.String("variant.status", string(variant.Status)),
		attribute.String("package.revision", packageRevision.Name),
	)

	return variant, nil
}

// createSpecializedPackageRevision creates a specialized package revision
func (npc *NephioPackageCatalog) createSpecializedPackageRevision(ctx context.Context, variant *PackageVariant) (*porch.PackageRevision, error) {
	ctx, span := npc.tracer.Start(ctx, "create-specialized-package-revision")
	defer span.End()

	// Create package spec based on blueprint and specialization
	packageSpec := &porch.PackageSpec{
		Repository:  npc.config.CatalogRepository,
		PackageName: variant.Name,
		Revision:    "v1",
		Lifecycle:   porch.PackageRevisionLifecycleDraft,
		Labels: map[string]string{
			"blueprint":      variant.Blueprint.Name,
			"cluster":        variant.TargetCluster.Name,
			"specialized":    "true",
			"intent-managed": "true",
		},
		Annotations: map[string]string{
			"blueprint.version":   variant.Blueprint.Version,
			"cluster.region":      variant.TargetCluster.Region,
			"cluster.zone":        variant.TargetCluster.Zone,
			"specialization.time": time.Now().Format(time.RFC3339),
		},
	}

	// Create package revision
	packageRevision := &porch.PackageRevision{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "porch.nephoran.com/v1",
			Kind:       "PackageRevision",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        fmt.Sprintf("%s-%s", packageSpec.PackageName, packageSpec.Revision),
			Namespace:   "nephoran-system",
			Labels:      packageSpec.Labels,
			Annotations: packageSpec.Annotations,
		},
		Spec: porch.PackageRevisionSpec{
			PackageName: packageSpec.PackageName,
			Repository:  packageSpec.Repository,
			Revision:    packageSpec.Revision,
			Lifecycle:   packageSpec.Lifecycle,
			Resources:   npc.convertResourcesForSpec(npc.generateSpecializedResources(ctx, variant)),
			Functions:   npc.convertFunctionsForSpec(npc.generateSpecializedFunctions(ctx, variant)),
		},
	}

	// Create the package revision in Porch (simulated)
	// In a real implementation, this would use the Porch client
	createdPackage := &porch.PackageRevision{
		TypeMeta:   packageRevision.TypeMeta,
		ObjectMeta: packageRevision.ObjectMeta,
		Spec:       packageRevision.Spec,
		Status: porch.PackageRevisionStatus{
			UpstreamLock: nil,
			PublishedBy:  "nephoran-intent-operator",
			PublishedAt:  &metav1.Time{Time: time.Now()},
		},
	}

	span.SetAttributes(
		attribute.String("package.name", createdPackage.Spec.PackageName),
		attribute.String("package.revision", createdPackage.Spec.Revision),
		attribute.Int("resources.count", len(createdPackage.Spec.Resources)),
		attribute.Int("functions.count", len(createdPackage.Spec.Functions)),
	)

	return createdPackage, nil
}

// generateSpecializedResources generates specialized Kubernetes resources
func (npc *NephioPackageCatalog) generateSpecializedResources(ctx context.Context, variant *PackageVariant) []porch.KRMResource {
	resources := make([]porch.KRMResource, 0)

	// Generate resources based on blueprint templates and specialization
	for _, template := range variant.Blueprint.Resources {
		// Apply specialization parameters to template
		specializedResource := npc.applySpecialization(template, variant.Specialization)

		resource := porch.KRMResource{
			APIVersion: template.APIVersion,
			Kind:       template.Kind,
			Metadata: map[string]interface{}{
				"name": template.Name,
			},
			Spec: specializedResource,
		}

		resources = append(resources, resource)
	}

	// Add cluster-specific resources
	if variant.TargetCluster != nil {
		clusterResources := npc.generateClusterSpecificResources(ctx, variant)
		resources = append(resources, clusterResources...)
	}

	return resources
}

// generateSpecializedFunctions generates specialized KRM functions
func (npc *NephioPackageCatalog) generateSpecializedFunctions(ctx context.Context, variant *PackageVariant) []porch.FunctionConfig {
	functions := make([]porch.FunctionConfig, 0)

	// Add blueprint functions with specialization
	for _, funcDef := range variant.Blueprint.Functions {
		functionConfig := porch.FunctionConfig{
			Image: funcDef.Image,
			ConfigMap: npc.mergeConfigs(
				funcDef.Config,
				variant.Specialization.Parameters,
			),
		}
		functions = append(functions, functionConfig)
	}

	// Add specialization-specific functions
	if variant.Specialization.ORANCompliance != nil {
		oranFunction := porch.FunctionConfig{
			Image: "krm/oran-validator:latest",
			ConfigMap: map[string]interface{}{
				"interfaces":     variant.Specialization.ORANCompliance.Interfaces,
				"validations":    variant.Specialization.ORANCompliance.Validations,
				"certifications": variant.Specialization.ORANCompliance.Certifications,
			},
		}
		functions = append(functions, oranFunction)
	}

	if variant.Specialization.NetworkSlice != nil {
		sliceFunction := porch.FunctionConfig{
			Image: "krm/network-slice-optimizer:latest",
			ConfigMap: map[string]interface{}{
				"sliceId":   variant.Specialization.NetworkSlice.SliceID,
				"sliceType": variant.Specialization.NetworkSlice.SliceType,
				"sla":       variant.Specialization.NetworkSlice.SLA,
				"qos":       variant.Specialization.NetworkSlice.QoS,
			},
		}
		functions = append(functions, sliceFunction)
	}

	return functions
}

// applySpecialization applies specialization parameters to a resource template
func (npc *NephioPackageCatalog) applySpecialization(template ResourceTemplate, spec *SpecializationRequest) map[string]interface{} {
	// Deep copy the template
	specialized := npc.deepCopyMap(template.Template)

	// Apply cluster context
	if spec.ClusterContext != nil {
		npc.injectClusterContext(specialized, spec.ClusterContext)
	}

	// Apply parameters
	for key, value := range spec.Parameters {
		npc.injectParameter(specialized, key, value)
	}

	// Apply resource overrides
	for key, value := range spec.ResourceOverrides {
		npc.injectResourceOverride(specialized, key, value)
	}

	return specialized
}

// generateClusterSpecificResources generates cluster-specific resources
func (npc *NephioPackageCatalog) generateClusterSpecificResources(ctx context.Context, variant *PackageVariant) []porch.KRMResource {
	resources := make([]porch.KRMResource, 0)

	cluster := variant.TargetCluster

	// Namespace resource
	namespaceResource := porch.KRMResource{
		Kind:       "Namespace",
		APIVersion: "v1",
		Metadata: map[string]interface{}{
			"name": fmt.Sprintf("%s-ns", variant.Name),
			"labels": map[string]interface{}{
				"cluster":    cluster.Name,
				"region":     cluster.Region,
				"zone":       cluster.Zone,
				"managed-by": "nephoran-intent-operator",
			},
		},
	}
	resources = append(resources, namespaceResource)

	// ConfigMap with cluster info
	configMapResource := porch.KRMResource{
		Kind:       "ConfigMap",
		APIVersion: "v1",
		Metadata: map[string]interface{}{
			"name":      fmt.Sprintf("%s-cluster-info", variant.Name),
			"namespace": fmt.Sprintf("%s-ns", variant.Name),
		},
		Data: map[string]interface{}{
			"cluster.name":   cluster.Name,
			"cluster.region": cluster.Region,
			"cluster.zone":   cluster.Zone,
			"variant.name":   variant.Name,
		},
	}
	resources = append(resources, configMapResource)

	return resources
}

// Helper methods

func (npc *NephioPackageCatalog) findWorkloadCluster(ctx context.Context, clusterName string) (*WorkloadCluster, error) {
	// This would typically query the workload registry
	// For now, return a mock cluster
	return &WorkloadCluster{
		Name:   clusterName,
		Status: WorkloadClusterStatusActive,
		Region: "us-east-1",
		Zone:   "us-east-1a",
		Capabilities: []ClusterCapability{
			{
				Name:    "5g-core",
				Type:    "network-function",
				Version: "1.0",
				Status:  "ready",
			},
		},
		CreatedAt: time.Now(),
	}, nil
}

func (npc *NephioPackageCatalog) deepCopyMap(source map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	for k, v := range source {
		switch val := v.(type) {
		case map[string]interface{}:
			result[k] = npc.deepCopyMap(val)
		case []interface{}:
			result[k] = npc.deepCopySlice(val)
		default:
			result[k] = val
		}
	}
	return result
}

func (npc *NephioPackageCatalog) deepCopySlice(source []interface{}) []interface{} {
	result := make([]interface{}, len(source))
	for i, v := range source {
		switch val := v.(type) {
		case map[string]interface{}:
			result[i] = npc.deepCopyMap(val)
		case []interface{}:
			result[i] = npc.deepCopySlice(val)
		default:
			result[i] = val
		}
	}
	return result
}

func (npc *NephioPackageCatalog) injectClusterContext(resource map[string]interface{}, cluster *ClusterContext) {
	// Inject cluster-specific information into resource
	if metadata, ok := resource["metadata"].(map[string]interface{}); ok {
		if labels, ok := metadata["labels"].(map[string]interface{}); ok {
			labels["cluster.name"] = cluster.Name
			labels["cluster.region"] = cluster.Region
			labels["cluster.zone"] = cluster.Zone
		}
	}
}

func (npc *NephioPackageCatalog) injectParameter(resource map[string]interface{}, key string, value interface{}) {
	// Simple parameter injection - in practice this would be more sophisticated
	npc.replaceInMap(resource, fmt.Sprintf("{{.%s}}", key), value)
}

func (npc *NephioPackageCatalog) injectResourceOverride(resource map[string]interface{}, path string, value interface{}) {
	// Apply resource override at specified path
	// This would use JSONPath or similar for complex path resolution
	resource[path] = value
}

func (npc *NephioPackageCatalog) replaceInMap(m map[string]interface{}, placeholder string, value interface{}) {
	for k, v := range m {
		switch val := v.(type) {
		case string:
			if val == placeholder {
				m[k] = value
			}
		case map[string]interface{}:
			npc.replaceInMap(val, placeholder, value)
		case []interface{}:
			npc.replaceInSlice(val, placeholder, value)
		}
	}
}

func (npc *NephioPackageCatalog) replaceInSlice(s []interface{}, placeholder string, value interface{}) {
	for i, v := range s {
		switch val := v.(type) {
		case string:
			if val == placeholder {
				s[i] = value
			}
		case map[string]interface{}:
			npc.replaceInMap(val, placeholder, value)
		case []interface{}:
			npc.replaceInSlice(val, placeholder, value)
		}
	}
}

func (npc *NephioPackageCatalog) mergeConfigs(base map[string]interface{}, overlay map[string]interface{}) map[string]interface{} {
	result := npc.deepCopyMap(base)
	for k, v := range overlay {
		result[k] = v
	}
	return result
}

// initializeStandardBlueprints initializes the catalog with standard blueprints
func (npc *NephioPackageCatalog) initializeStandardBlueprints() error {
	blueprints := []*BlueprintPackage{
		{
			Name:        "5g-amf-blueprint",
			Repository:  npc.config.CatalogRepository,
			Version:     "1.0.0",
			Description: "5G Access and Mobility Management Function",
			Category:    "5g-core",
			IntentTypes: []v1.IntentType{
				v1.IntentTypeDeployment,
				v1.IntentTypeOptimization,
				v1.IntentTypeScaling,
			},
			Dependencies: []PackageDependency{
				{
					Name:       "5g-common",
					Repository: npc.config.CatalogRepository,
					Version:    "1.0.0",
					Optional:   false,
				},
			},
			Parameters: []ParameterDefinition{
				{
					Name:        "replicas",
					Type:        "int",
					Description: "Number of AMF replicas",
					Default:     1,
					Required:    false,
				},
				{
					Name:        "resources.cpu",
					Type:        "string",
					Description: "CPU resource requirements",
					Default:     "500m",
					Required:    false,
				},
				{
					Name:        "resources.memory",
					Type:        "string",
					Description: "Memory resource requirements",
					Default:     "512Mi",
					Required:    false,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:        "5g-upf-blueprint",
			Repository:  npc.config.CatalogRepository,
			Version:     "1.0.0",
			Description: "5G User Plane Function",
			Category:    "5g-core",
			IntentTypes: []v1.IntentType{
				v1.IntentTypeDeployment,
				v1.IntentTypeOptimization,
				v1.IntentTypeScaling,
			},
			Dependencies: []PackageDependency{
				{
					Name:       "5g-common",
					Repository: npc.config.CatalogRepository,
					Version:    "1.0.0",
					Optional:   false,
				},
			},
			Parameters: []ParameterDefinition{
				{
					Name:        "replicas",
					Type:        "int",
					Description: "Number of UPF replicas",
					Default:     1,
					Required:    false,
				},
				{
					Name:        "dataPlane.type",
					Type:        "string",
					Description: "Data plane implementation",
					Default:     "dpdk",
					Required:    false,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			Name:        "oran-ric-blueprint",
			Repository:  npc.config.CatalogRepository,
			Version:     "1.0.0",
			Description: "O-RAN RIC Platform",
			Category:    "oran",
			IntentTypes: []v1.IntentType{
				v1.IntentTypeDeployment,
				v1.IntentTypeOptimization,
			},
			Dependencies: []PackageDependency{
				{
					Name:       "oran-common",
					Repository: npc.config.CatalogRepository,
					Version:    "1.0.0",
					Optional:   false,
				},
			},
			Parameters: []ParameterDefinition{
				{
					Name:        "ric.type",
					Type:        "string",
					Description: "RIC type (near-rt or non-rt)",
					Default:     "near-rt",
					Required:    true,
				},
			},
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	// Store blueprints in catalog
	for _, blueprint := range blueprints {
		npc.blueprints.Store(blueprint.Name, blueprint)
		npc.metrics.CatalogSize.WithLabelValues("blueprint").Inc()
	}

	return nil
}

// convertResourcesForSpec converts []porch.KRMResource to []interface{} for PackageRevisionSpec
func (npc *NephioPackageCatalog) convertResourcesForSpec(resources []porch.KRMResource) []interface{} {
	result := make([]interface{}, len(resources))
	for i, resource := range resources {
		result[i] = resource
	}
	return result
}

// convertFunctionsForSpec converts []porch.FunctionConfig to []interface{} for PackageRevisionSpec
func (npc *NephioPackageCatalog) convertFunctionsForSpec(functions []porch.FunctionConfig) []interface{} {
	result := make([]interface{}, len(functions))
	for i, function := range functions {
		result[i] = function
	}
	return result
}
