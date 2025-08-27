package o2

import (
	"context"
	"fmt"
	"sync"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// ResourceTypeManager manages resource type definitions and YANG models
type ResourceTypeManager struct {
	storage    O2IMSStorage
	kubeClient client.Client
	logger     *logging.StructuredLogger
	config     *ResourceTypeConfig

	// Resource type cache
	typeCache   map[string]*models.ResourceType
	cacheMu     sync.RWMutex
	cacheExpiry time.Duration

	// YANG model management
	yangModelStore YANGModelStore
	yangValidator  YANGValidator

	// Schema registry
	schemaRegistry SchemaRegistry

	// Type discovery and validation
	discoveryEnabled  bool
	validationEnabled bool
}

// ResourceTypeConfig defines configuration for resource type management
type ResourceTypeConfig struct {
	// Cache configuration
	CacheEnabled bool          `json:"cache_enabled"`
	CacheExpiry  time.Duration `json:"cache_expiry"`
	MaxCacheSize int           `json:"max_cache_size"`

	// YANG model configuration
	YANGModelsEnabled     bool   `json:"yang_models_enabled"`
	YANGModelsPath        string `json:"yang_models_path"`
	YANGValidationEnabled bool   `json:"yang_validation_enabled"`

	// Schema validation
	SchemaValidationEnabled bool `json:"schema_validation_enabled"`
	StrictValidation        bool `json:"strict_validation"`

	// Type discovery
	AutoDiscoveryEnabled bool          `json:"auto_discovery_enabled"`
	DiscoveryInterval    time.Duration `json:"discovery_interval"`
	SupportedProviders   []string      `json:"supported_providers"`

	// Vendor extensions
	VendorExtensionsEnabled bool     `json:"vendor_extensions_enabled"`
	AllowedVendors          []string `json:"allowed_vendors"`

	// Versioning
	VersioningEnabled bool     `json:"versioning_enabled"`
	DefaultVersion    string   `json:"default_version"`
	SupportedVersions []string `json:"supported_versions"`
}

// YANGModelStore defines the interface for YANG model storage and retrieval
type YANGModelStore interface {
	// YANG model operations
	StoreYANGModel(ctx context.Context, model *YANGModel) error
	GetYANGModel(ctx context.Context, modelName, version string) (*YANGModel, error)
	ListYANGModels(ctx context.Context, filter *YANGModelFilter) ([]*YANGModel, error)
	UpdateYANGModel(ctx context.Context, modelName, version string, model *YANGModel) error
	DeleteYANGModel(ctx context.Context, modelName, version string) error

	// Module dependency resolution
	ResolveDependencies(ctx context.Context, modelName, version string) ([]*YANGModel, error)
	ValidateDependencies(ctx context.Context, model *YANGModel) error

	// Schema compilation
	CompileSchema(ctx context.Context, modelName, version string) (*CompiledSchema, error)
	CacheCompiledSchema(ctx context.Context, schema *CompiledSchema) error
}

// YANGValidator defines the interface for YANG model validation
type YANGValidator interface {
	// Model validation
	ValidateYANGModel(ctx context.Context, model *YANGModel) (*ValidationResult, error)
	ValidateYANGSyntax(ctx context.Context, yangContent string) (*ValidationResult, error)
	ValidateYANGSemantics(ctx context.Context, model *YANGModel) (*ValidationResult, error)

	// Instance validation
	ValidateInstance(ctx context.Context, data *runtime.RawExtension, schemaRef *SchemaReference) (*ValidationResult, error)
	ValidateConfiguration(ctx context.Context, config *runtime.RawExtension, resourceType *models.ResourceType) (*ValidationResult, error)

	// Constraint validation
	ValidateConstraints(ctx context.Context, data interface{}, constraints []*YANGConstraint) (*ValidationResult, error)
}

// SchemaRegistry defines the interface for schema registration and lookup
type SchemaRegistry interface {
	// Schema registration
	RegisterSchema(ctx context.Context, schema *Schema) error
	UnregisterSchema(ctx context.Context, schemaID string) error
	GetSchema(ctx context.Context, schemaID string) (*Schema, error)
	ListSchemas(ctx context.Context, filter *SchemaFilter) ([]*Schema, error)

	// Schema versioning
	RegisterSchemaVersion(ctx context.Context, schemaID, version string, schema *Schema) error
	GetSchemaVersion(ctx context.Context, schemaID, version string) (*Schema, error)
	ListSchemaVersions(ctx context.Context, schemaID string) ([]string, error)

	// Schema compatibility
	CheckCompatibility(ctx context.Context, oldSchema, newSchema *Schema) (*CompatibilityResult, error)
	ValidateBackwardCompatibility(ctx context.Context, schemaID, newVersion string, schema *Schema) (*CompatibilityResult, error)
}

// YANGModel represents a YANG model definition
type YANGModel struct {
	ModelID      string `json:"modelId"`
	Name         string `json:"name"`
	Version      string `json:"version"`
	Namespace    string `json:"namespace"`
	Prefix       string `json:"prefix"`
	Description  string `json:"description,omitempty"`
	Organization string `json:"organization,omitempty"`
	Contact      string `json:"contact,omitempty"`

	// Model content
	YANGContent    string          `json:"yangContent"`
	CompiledSchema *CompiledSchema `json:"compiledSchema,omitempty"`

	// Dependencies
	ImportedModules    []*YANGModuleRef  `json:"importedModules,omitempty"`
	IncludedSubmodules []*YANGModuleRef  `json:"includedSubmodules,omitempty"`
	Dependencies       []*YANGDependency `json:"dependencies,omitempty"`

	// Validation and metadata
	ValidationStatus *ValidationResult `json:"validationStatus,omitempty"`
	Features         []string          `json:"features,omitempty"`
	Deviations       []*YANGDeviation  `json:"deviations,omitempty"`

	// Lifecycle
	Status     string                 `json:"status"` // DRAFT, ACTIVE, DEPRECATED, OBSOLETE
	Tags       map[string]string      `json:"tags,omitempty"`
	Extensions map[string]interface{} `json:"extensions,omitempty"`
	CreatedAt  time.Time              `json:"createdAt"`
	UpdatedAt  time.Time              `json:"updatedAt"`
	CreatedBy  string                 `json:"createdBy,omitempty"`
	UpdatedBy  string                 `json:"updatedBy,omitempty"`
}

// YANGModuleRef represents a reference to a YANG module
type YANGModuleRef struct {
	Name     string `json:"name"`
	Version  string `json:"version,omitempty"`
	Prefix   string `json:"prefix,omitempty"`
	Required bool   `json:"required"`
}

// YANGDependency represents a dependency between YANG models
type YANGDependency struct {
	DependencyType string `json:"dependencyType"` // IMPORT, INCLUDE, AUGMENT, DEVIATION
	ModuleName     string `json:"moduleName"`
	Version        string `json:"version,omitempty"`
	Required       bool   `json:"required"`
	Description    string `json:"description,omitempty"`
}

// YANGDeviation represents a YANG deviation
type YANGDeviation struct {
	TargetNode    string                 `json:"targetNode"`
	DeviationType string                 `json:"deviationType"` // NOT_SUPPORTED, ADD, REPLACE, DELETE
	Description   string                 `json:"description,omitempty"`
	Reference     string                 `json:"reference,omitempty"`
	Condition     map[string]interface{} `json:"condition,omitempty"`
}

// CompiledSchema represents a compiled YANG schema
type CompiledSchema struct {
	SchemaID   string    `json:"schemaId"`
	ModelName  string    `json:"modelName"`
	Version    string    `json:"version"`
	CompiledAt time.Time `json:"compiledAt"`

	// Schema structure
	RootNodes  []*YANGNode     `json:"rootNodes"`
	DataTypes  []*YANGDataType `json:"dataTypes"`
	Identities []*YANGIdentity `json:"identities"`
	Features   []*YANGFeature  `json:"features"`

	// Constraints and validations
	Constraints []*YANGConstraint `json:"constraints,omitempty"`

	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// YANGNode represents a node in the YANG schema tree
type YANGNode struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // CONTAINER, LIST, LEAF, LEAF_LIST, CHOICE, CASE, ANYXML, ANYDATA
	Path        string `json:"path"`
	Description string `json:"description,omitempty"`

	// Node properties
	Config    *bool  `json:"config,omitempty"`
	Mandatory *bool  `json:"mandatory,omitempty"`
	Status    string `json:"status,omitempty"` // CURRENT, DEPRECATED, OBSOLETE

	// Type information
	DataType     *YANGDataType `json:"dataType,omitempty"`
	DefaultValue interface{}   `json:"defaultValue,omitempty"`

	// Constraints
	Constraints []*YANGConstraint `json:"constraints,omitempty"`

	// Hierarchy
	Parent   *YANGNode   `json:"parent,omitempty"`
	Children []*YANGNode `json:"children,omitempty"`

	// Extensions
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// YANGDataType represents a YANG data type
type YANGDataType struct {
	Name        string `json:"name"`
	BaseType    string `json:"baseType"`
	Description string `json:"description,omitempty"`

	// Type restrictions
	Restrictions []*YANGRestriction `json:"restrictions,omitempty"`
	Enumerations []*YANGEnumeration `json:"enumerations,omitempty"`
	UnionTypes   []*YANGDataType    `json:"unionTypes,omitempty"`

	// Derived type information
	DerivedFrom string `json:"derivedFrom,omitempty"`

	// Extensions
	Extensions map[string]interface{} `json:"extensions,omitempty"`
}

// YANGRestriction represents a restriction on a YANG data type
type YANGRestriction struct {
	Type         string      `json:"type"` // LENGTH, PATTERN, RANGE, FRACTION_DIGITS
	Value        interface{} `json:"value"`
	ErrorMessage string      `json:"errorMessage,omitempty"`
	Description  string      `json:"description,omitempty"`
}

// YANGEnumeration represents an enumeration value in YANG
type YANGEnumeration struct {
	Name        string `json:"name"`
	Value       *int   `json:"value,omitempty"`
	Description string `json:"description,omitempty"`
	Status      string `json:"status,omitempty"`
}

// YANGIdentity represents a YANG identity
type YANGIdentity struct {
	Name           string   `json:"name"`
	BaseIdentities []string `json:"baseIdentities,omitempty"`
	Description    string   `json:"description,omitempty"`
	Status         string   `json:"status,omitempty"`
}

// YANGFeature represents a YANG feature
type YANGFeature struct {
	Name        string   `json:"name"`
	Description string   `json:"description,omitempty"`
	Status      string   `json:"status,omitempty"`
	IfFeatures  []string `json:"ifFeatures,omitempty"`
}

// YANGConstraint represents a YANG constraint
type YANGConstraint struct {
	Type         string                 `json:"type"` // MUST, WHEN, UNIQUE, KEY
	Expression   string                 `json:"expression"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
	Description  string                 `json:"description,omitempty"`
	Context      map[string]interface{} `json:"context,omitempty"`
}

// ValidationResult represents the result of YANG validation
type ValidationResult struct {
	Valid          bool                 `json:"valid"`
	Errors         []*ValidationError   `json:"errors,omitempty"`
	Warnings       []*ValidationWarning `json:"warnings,omitempty"`
	Summary        string               `json:"summary,omitempty"`
	ValidationTime time.Duration        `json:"validationTime"`
	ValidatedAt    time.Time            `json:"validatedAt"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Code     string `json:"code"`
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
	Line     int    `json:"line,omitempty"`
	Column   int    `json:"column,omitempty"`
	Severity string `json:"severity"` // ERROR, WARNING, INFO
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Path    string `json:"path,omitempty"`
	Line    int    `json:"line,omitempty"`
	Column  int    `json:"column,omitempty"`
}

// Schema represents a generic schema definition
type Schema struct {
	SchemaID       string          `json:"schemaId"`
	Name           string          `json:"name"`
	Version        string          `json:"version"`
	Type           string          `json:"type"` // YANG, JSON_SCHEMA, XSD, AVRO
	Content        string          `json:"content"`
	CompiledSchema *CompiledSchema `json:"compiledSchema,omitempty"`

	// Metadata
	Description string                 `json:"description,omitempty"`
	Tags        map[string]string      `json:"tags,omitempty"`
	Extensions  map[string]interface{} `json:"extensions,omitempty"`

	// Lifecycle
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	CreatedBy string    `json:"createdBy,omitempty"`
}

// SchemaReference represents a reference to a schema
type SchemaReference struct {
	SchemaID string `json:"schemaId"`
	Version  string `json:"version,omitempty"`
	Type     string `json:"type,omitempty"`
}

// CompatibilityResult represents the result of schema compatibility check
type CompatibilityResult struct {
	Compatible         bool                  `json:"compatible"`
	CompatibilityLevel string                `json:"compatibilityLevel"` // FULL, BACKWARD, FORWARD, NONE
	Issues             []*CompatibilityIssue `json:"issues,omitempty"`
	Summary            string                `json:"summary,omitempty"`
}

// CompatibilityIssue represents a compatibility issue
type CompatibilityIssue struct {
	Type     string `json:"type"` // BREAKING, NON_BREAKING, WARNING
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
	Severity string `json:"severity"`
}

// Filter types for queries

// YANGModelFilter defines filters for YANG model queries
type YANGModelFilter struct {
	Names         []string          `json:"names,omitempty"`
	Versions      []string          `json:"versions,omitempty"`
	Namespaces    []string          `json:"namespaces,omitempty"`
	Organizations []string          `json:"organizations,omitempty"`
	Status        []string          `json:"status,omitempty"`
	Features      []string          `json:"features,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	CreatedAfter  *time.Time        `json:"createdAfter,omitempty"`
	CreatedBefore *time.Time        `json:"createdBefore,omitempty"`
	Limit         int               `json:"limit,omitempty"`
	Offset        int               `json:"offset,omitempty"`
	SortBy        string            `json:"sortBy,omitempty"`
	SortOrder     string            `json:"sortOrder,omitempty"`
}

// SchemaFilter defines filters for schema queries
type SchemaFilter struct {
	Names         []string          `json:"names,omitempty"`
	Versions      []string          `json:"versions,omitempty"`
	Types         []string          `json:"types,omitempty"`
	Status        []string          `json:"status,omitempty"`
	Tags          map[string]string `json:"tags,omitempty"`
	CreatedAfter  *time.Time        `json:"createdAfter,omitempty"`
	CreatedBefore *time.Time        `json:"createdBefore,omitempty"`
	Limit         int               `json:"limit,omitempty"`
	Offset        int               `json:"offset,omitempty"`
	SortBy        string            `json:"sortBy,omitempty"`
	SortOrder     string            `json:"sortOrder,omitempty"`
}

// NewResourceTypeManager creates a new resource type manager
func NewResourceTypeManager(storage O2IMSStorage, kubeClient client.Client, logger *logging.StructuredLogger) *ResourceTypeManager {
	config := &ResourceTypeConfig{
		CacheEnabled:            true,
		CacheExpiry:             30 * time.Minute,
		MaxCacheSize:            500,
		YANGModelsEnabled:       true,
		YANGModelsPath:          "/etc/yang-models",
		YANGValidationEnabled:   true,
		SchemaValidationEnabled: true,
		StrictValidation:        false,
		AutoDiscoveryEnabled:    true,
		DiscoveryInterval:       10 * time.Minute,
		SupportedProviders:      []string{"kubernetes", "openstack", "aws", "azure", "gcp"},
		VendorExtensionsEnabled: true,
		AllowedVendors:          []string{"ericsson", "nokia", "huawei", "cisco", "juniper"},
		VersioningEnabled:       true,
		DefaultVersion:          "1.0.0",
		SupportedVersions:       []string{"1.0.0", "1.1.0"},
	}

	return &ResourceTypeManager{
		storage:           storage,
		kubeClient:        kubeClient,
		logger:            logger,
		config:            config,
		typeCache:         make(map[string]*models.ResourceType),
		cacheExpiry:       config.CacheExpiry,
		discoveryEnabled:  config.AutoDiscoveryEnabled,
		validationEnabled: config.YANGValidationEnabled,
	}
}

// GetResourceTypes retrieves resource types with filtering support
func (rtm *ResourceTypeManager) GetResourceTypes(ctx context.Context, filter *models.ResourceTypeFilter) ([]*models.ResourceType, error) {
	logger := log.FromContext(ctx)
	logger.Info("retrieving resource types", "filter", filter)

	// Check cache first if enabled and no specific filter
	if rtm.config.CacheEnabled && filter == nil {
		if types := rtm.getCachedTypes(ctx); len(types) > 0 {
			return types, nil
		}
	}

	// Retrieve from storage
	types, err := rtm.storage.ListResourceTypes(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("failed to list resource types: %w", err)
	}

	// Update cache if enabled
	if rtm.config.CacheEnabled {
		rtm.updateTypeCache(types)
	}

	// Enrich with YANG model information if enabled
	if rtm.config.YANGModelsEnabled {
		if err := rtm.enrichTypesWithYANGModels(ctx, types); err != nil {
			logger.Info("failed to enrich types with YANG models", "error", err)
		}
	}

	logger.Info("retrieved resource types", "count", len(types))
	return types, nil
}

// GetResourceType retrieves a specific resource type
func (rtm *ResourceTypeManager) GetResourceType(ctx context.Context, resourceTypeID string) (*models.ResourceType, error) {
	logger := log.FromContext(ctx)
	logger.Info("retrieving resource type", "typeId", resourceTypeID)

	// Check cache first
	if rtm.config.CacheEnabled {
		if resourceType := rtm.getCachedType(resourceTypeID); resourceType != nil {
			return resourceType, nil
		}
	}

	// Retrieve from storage
	resourceType, err := rtm.storage.GetResourceType(ctx, resourceTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource type %s: %w", resourceTypeID, err)
	}

	// Update cache
	if rtm.config.CacheEnabled {
		rtm.setCachedType(resourceType)
	}

	// Enrich with YANG model information
	if rtm.config.YANGModelsEnabled {
		if err := rtm.enrichTypeWithYANGModel(ctx, resourceType); err != nil {
			logger.Info("failed to enrich type with YANG model", "error", err)
		}
	}

	return resourceType, nil
}

// CreateResourceType creates a new resource type
func (rtm *ResourceTypeManager) CreateResourceType(ctx context.Context, resourceType *models.ResourceType) (*models.ResourceType, error) {
	logger := log.FromContext(ctx)
	logger.Info("creating resource type", "name", resourceType.Name, "vendor", resourceType.Vendor)

	// Validate the resource type
	if err := rtm.validateResourceType(ctx, resourceType); err != nil {
		return nil, fmt.Errorf("resource type validation failed: %w", err)
	}

	// Set creation metadata
	resourceType.CreatedAt = time.Now()
	resourceType.UpdatedAt = time.Now()

	// Generate resource type ID if not provided
	if resourceType.ResourceTypeID == "" {
		resourceType.ResourceTypeID = rtm.generateResourceTypeID(resourceType)
	}

	// Store the resource type
	if err := rtm.storage.StoreResourceType(ctx, resourceType); err != nil {
		return nil, fmt.Errorf("failed to store resource type: %w", err)
	}

	// Update cache
	if rtm.config.CacheEnabled {
		rtm.setCachedType(resourceType)
	}

	// Process YANG model if provided
	if rtm.config.YANGModelsEnabled && resourceType.YANGModel != nil {
		if err := rtm.processYANGModel(ctx, resourceType); err != nil {
			logger.Info("failed to process YANG model", "error", err)
		}
	}

	logger.Info("resource type created successfully", "typeId", resourceType.ResourceTypeID)
	return resourceType, nil
}

// UpdateResourceType updates an existing resource type
func (rtm *ResourceTypeManager) UpdateResourceType(ctx context.Context, resourceTypeID string, resourceType *models.ResourceType) (*models.ResourceType, error) {
	logger := log.FromContext(ctx)
	logger.Info("updating resource type", "typeId", resourceTypeID)

	// Get current resource type
	existing, err := rtm.GetResourceType(ctx, resourceTypeID)
	if err != nil {
		return nil, err
	}

	// Update fields
	resourceType.ResourceTypeID = resourceTypeID
	resourceType.CreatedAt = existing.CreatedAt
	resourceType.UpdatedAt = time.Now()

	// Validate the updated resource type
	if err := rtm.validateResourceType(ctx, resourceType); err != nil {
		return nil, fmt.Errorf("resource type validation failed: %w", err)
	}

	// Check compatibility if versioning is enabled
	if rtm.config.VersioningEnabled && rtm.config.YANGModelsEnabled {
		if err := rtm.checkTypeCompatibility(ctx, existing, resourceType); err != nil {
			return nil, fmt.Errorf("compatibility check failed: %w", err)
		}
	}

	// Update in storage
	if err := rtm.storage.UpdateResourceType(ctx, resourceTypeID, resourceType); err != nil {
		return nil, fmt.Errorf("failed to update resource type: %w", err)
	}

	// Update cache
	if rtm.config.CacheEnabled {
		rtm.setCachedType(resourceType)
	}

	logger.Info("resource type updated successfully", "typeId", resourceTypeID)
	return resourceType, nil
}

// DeleteResourceType deletes a resource type
func (rtm *ResourceTypeManager) DeleteResourceType(ctx context.Context, resourceTypeID string) error {
	logger := log.FromContext(ctx)
	logger.Info("deleting resource type", "typeId", resourceTypeID)

	// Check if type exists
	_, err := rtm.GetResourceType(ctx, resourceTypeID)
	if err != nil {
		return err
	}

	// Check for dependent resources
	if err := rtm.checkResourceTypeDependencies(ctx, resourceTypeID); err != nil {
		return fmt.Errorf("cannot delete resource type with dependencies: %w", err)
	}

	// Delete from storage
	if err := rtm.storage.DeleteResourceType(ctx, resourceTypeID); err != nil {
		return fmt.Errorf("failed to delete resource type: %w", err)
	}

	// Remove from cache
	if rtm.config.CacheEnabled {
		rtm.removeCachedType(resourceTypeID)
	}

	logger.Info("resource type deleted successfully", "typeId", resourceTypeID)
	return nil
}

// DiscoverResourceTypes discovers resource types from registered providers
func (rtm *ResourceTypeManager) DiscoverResourceTypes(ctx context.Context) (map[string][]*models.ResourceType, error) {
	logger := log.FromContext(ctx)
	logger.Info("discovering resource types from providers")

	discovered := make(map[string][]*models.ResourceType)

	for _, provider := range rtm.config.SupportedProviders {
		types, err := rtm.discoverProviderResourceTypes(ctx, provider)
		if err != nil {
			logger.Info("failed to discover resource types from provider", "provider", provider, "error", err)
			continue
		}
		discovered[provider] = types
	}

	logger.Info("resource type discovery completed", "providers", len(discovered))
	return discovered, nil
}

// RegisterYANGModel registers a YANG model in the system
func (rtm *ResourceTypeManager) RegisterYANGModel(ctx context.Context, model *YANGModel) error {
	if rtm.yangModelStore == nil {
		return fmt.Errorf("YANG model store not configured")
	}

	logger := log.FromContext(ctx)
	logger.Info("registering YANG model", "name", model.Name, "version", model.Version)

	// Validate YANG model
	if rtm.yangValidator != nil {
		result, err := rtm.yangValidator.ValidateYANGModel(ctx, model)
		if err != nil {
			return fmt.Errorf("YANG model validation failed: %w", err)
		}

		if !result.Valid {
			return fmt.Errorf("YANG model is invalid: %v", result.Errors)
		}

		model.ValidationStatus = result
	}

	// Store YANG model
	if err := rtm.yangModelStore.StoreYANGModel(ctx, model); err != nil {
		return fmt.Errorf("failed to store YANG model: %w", err)
	}

	logger.Info("YANG model registered successfully", "name", model.Name, "version", model.Version)
	return nil
}

// ValidateResourceConfiguration validates resource configuration against YANG model
func (rtm *ResourceTypeManager) ValidateResourceConfiguration(ctx context.Context, resourceTypeID string, config *runtime.RawExtension) (*ValidationResult, error) {
	if rtm.yangValidator == nil {
		return &ValidationResult{Valid: true, Summary: "YANG validation not configured"}, nil
	}

	// Get resource type
	resourceType, err := rtm.GetResourceType(ctx, resourceTypeID)
	if err != nil {
		return nil, fmt.Errorf("failed to get resource type: %w", err)
	}

	// Validate configuration
	result, err := rtm.yangValidator.ValidateConfiguration(ctx, config, resourceType)
	if err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return result, nil
}

// Private helper methods

// validateResourceType validates a resource type definition
func (rtm *ResourceTypeManager) validateResourceType(ctx context.Context, resourceType *models.ResourceType) error {
	if resourceType.Name == "" {
		return fmt.Errorf("resource type name is required")
	}
	if resourceType.Vendor == "" {
		return fmt.Errorf("resource type vendor is required")
	}
	if resourceType.Version == "" {
		return fmt.Errorf("resource type version is required")
	}

	// Validate vendor if restrictions are configured
	if rtm.config.VendorExtensionsEnabled && len(rtm.config.AllowedVendors) > 0 {
		allowed := false
		for _, vendor := range rtm.config.AllowedVendors {
			if resourceType.Vendor == vendor {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("vendor %s is not in allowed vendors list", resourceType.Vendor)
		}
	}

	// Validate version format if versioning is enabled
	if rtm.config.VersioningEnabled {
		if err := rtm.validateVersion(resourceType.Version); err != nil {
			return fmt.Errorf("invalid version format: %w", err)
		}
	}

	return nil
}

// generateResourceTypeID generates a unique resource type ID
func (rtm *ResourceTypeManager) generateResourceTypeID(resourceType *models.ResourceType) string {
	return fmt.Sprintf("%s-%s-%s-%d",
		resourceType.Vendor,
		resourceType.Name,
		resourceType.Version,
		time.Now().Unix())
}

// validateVersion validates version format
func (rtm *ResourceTypeManager) validateVersion(version string) error {
	// Simple semantic versioning validation
	// In a real implementation, you might use a more sophisticated version parser
	if len(version) == 0 {
		return fmt.Errorf("version cannot be empty")
	}
	return nil
}

// checkTypeCompatibility checks compatibility between resource type versions
func (rtm *ResourceTypeManager) checkTypeCompatibility(ctx context.Context, oldType, newType *models.ResourceType) error {
	// Implement compatibility checking logic based on YANG models or schema
	// For now, just check that the version is different
	if oldType.Version == newType.Version {
		return fmt.Errorf("version must be different for updates")
	}
	return nil
}

// checkResourceTypeDependencies checks if there are resources using this type
func (rtm *ResourceTypeManager) checkResourceTypeDependencies(ctx context.Context, resourceTypeID string) error {
	// Check if there are any resources of this type
	filter := &models.ResourceFilter{
		ResourceTypeIDs: []string{resourceTypeID},
		Limit:           1,
	}

	resources, err := rtm.storage.ListResources(ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to check dependencies: %w", err)
	}

	if len(resources) > 0 {
		return fmt.Errorf("resource type has %d dependent resources", len(resources))
	}

	return nil
}

// discoverProviderResourceTypes discovers resource types from a specific provider
func (rtm *ResourceTypeManager) discoverProviderResourceTypes(ctx context.Context, provider string) ([]*models.ResourceType, error) {
	// Implement provider-specific resource type discovery
	// This would integrate with provider APIs to discover available resource types
	// For now, return empty slice
	return []*models.ResourceType{}, nil
}

// processYANGModel processes and validates a YANG model
func (rtm *ResourceTypeManager) processYANGModel(ctx context.Context, resourceType *models.ResourceType) error {
	if rtm.yangValidator == nil {
		return fmt.Errorf("YANG validator not configured")
	}

	// This would process the YANG model associated with the resource type
	// Implementation would involve parsing, validating, and compiling the YANG model
	return nil
}

// enrichTypesWithYANGModels enriches resource types with YANG model information
func (rtm *ResourceTypeManager) enrichTypesWithYANGModels(ctx context.Context, types []*models.ResourceType) error {
	for _, resourceType := range types {
		if err := rtm.enrichTypeWithYANGModel(ctx, resourceType); err != nil {
			rtm.logger.Error("failed to enrich type with YANG model", "error", err, "typeId", resourceType.ResourceTypeID)
		}
	}
	return nil
}

// enrichTypeWithYANGModel enriches a single resource type with YANG model information
func (rtm *ResourceTypeManager) enrichTypeWithYANGModel(ctx context.Context, resourceType *models.ResourceType) error {
	if rtm.yangModelStore == nil || resourceType.YANGModel == nil {
		return nil
	}

	// Retrieve and attach compiled YANG model information
	// This would involve fetching the YANG model and its compiled schema
	return nil
}

// Cache management methods

// getCachedTypes returns all cached resource types
func (rtm *ResourceTypeManager) getCachedTypes(ctx context.Context) []*models.ResourceType {
	rtm.cacheMu.RLock()
	defer rtm.cacheMu.RUnlock()

	types := make([]*models.ResourceType, 0, len(rtm.typeCache))
	for _, resourceType := range rtm.typeCache {
		types = append(types, resourceType)
	}
	return types
}

// getCachedType returns a cached resource type by ID
func (rtm *ResourceTypeManager) getCachedType(typeID string) *models.ResourceType {
	rtm.cacheMu.RLock()
	defer rtm.cacheMu.RUnlock()
	return rtm.typeCache[typeID]
}

// setCachedType adds or updates a resource type in the cache
func (rtm *ResourceTypeManager) setCachedType(resourceType *models.ResourceType) {
	rtm.cacheMu.Lock()
	defer rtm.cacheMu.Unlock()
	rtm.typeCache[resourceType.ResourceTypeID] = resourceType
}

// updateTypeCache updates multiple resource types in the cache
func (rtm *ResourceTypeManager) updateTypeCache(types []*models.ResourceType) {
	rtm.cacheMu.Lock()
	defer rtm.cacheMu.Unlock()
	for _, resourceType := range types {
		rtm.typeCache[resourceType.ResourceTypeID] = resourceType
	}
}

// removeCachedType removes a resource type from the cache
func (rtm *ResourceTypeManager) removeCachedType(typeID string) {
	rtm.cacheMu.Lock()
	defer rtm.cacheMu.Unlock()
	delete(rtm.typeCache, typeID)
}

// SetYANGModelStore sets the YANG model store for the resource type manager
func (rtm *ResourceTypeManager) SetYANGModelStore(store YANGModelStore) {
	rtm.yangModelStore = store
}

// SetYANGValidator sets the YANG validator for the resource type manager
func (rtm *ResourceTypeManager) SetYANGValidator(validator YANGValidator) {
	rtm.yangValidator = validator
}

// SetSchemaRegistry sets the schema registry for the resource type manager
func (rtm *ResourceTypeManager) SetSchemaRegistry(registry SchemaRegistry) {
	rtm.schemaRegistry = registry
}

// GetConfig returns the current configuration
func (rtm *ResourceTypeManager) GetConfig() *ResourceTypeConfig {
	return rtm.config
}

// UpdateConfig updates the configuration
func (rtm *ResourceTypeManager) UpdateConfig(config *ResourceTypeConfig) {
	rtm.config = config
	rtm.cacheExpiry = config.CacheExpiry
	rtm.discoveryEnabled = config.AutoDiscoveryEnabled
	rtm.validationEnabled = config.YANGValidationEnabled
}
