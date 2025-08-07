package ims

import (
	"context"
	"fmt"
	"sync"
	"time"
	
	"sigs.k8s.io/controller-runtime/pkg/log"
	
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o2/models"
)

// CatalogService manages the O2 IMS resource catalog and deployment templates
// This service provides centralized management of resource types and deployment templates
// following O-RAN.WG6.O2ims-Interface-v01.01 specification
type CatalogService struct {
	// Resource type management
	resourceTypes map[string]*models.ResourceType
	rtMutex       sync.RWMutex
	
	// Deployment template management
	deploymentTemplates map[string]*models.DeploymentTemplate
	dtMutex            sync.RWMutex
	
	// Template validation and processing
	templateValidator   TemplateValidator
	templateProcessor   TemplateProcessor
	
	// Background services
	backgroundCtx    context.Context
	backgroundCancel context.CancelFunc
	
	// Statistics and metrics
	stats *CatalogStatistics
}

// TemplateValidator interface for validating deployment templates
type TemplateValidator interface {
	ValidateTemplate(ctx context.Context, template *models.DeploymentTemplate) error
	ValidateTemplateContent(ctx context.Context, content string, templateType string) error
	ValidateInputSchema(ctx context.Context, schema string) error
	ValidateOutputSchema(ctx context.Context, schema string) error
}

// TemplateProcessor interface for processing deployment templates
type TemplateProcessor interface {
	ProcessTemplate(ctx context.Context, template *models.DeploymentTemplate, parameters map[string]interface{}) (*ProcessedTemplate, error)
	ExtractParameters(ctx context.Context, template *models.DeploymentTemplate) ([]TemplateParameter, error)
	GenerateDocumentation(ctx context.Context, template *models.DeploymentTemplate) (*TemplateDocumentation, error)
}

// ProcessedTemplate represents a processed deployment template ready for deployment
type ProcessedTemplate struct {
	RenderedContent   string                 `json:"renderedContent"`
	Resources        []TemplateResource     `json:"resources"`
	Dependencies     []string               `json:"dependencies"`
	Parameters       map[string]interface{} `json:"parameters"`
	Outputs          map[string]interface{} `json:"outputs"`
	Metadata         map[string]string      `json:"metadata"`
}

// TemplateResource represents a resource defined in a template
type TemplateResource struct {
	Name         string                 `json:"name"`
	Type         string                 `json:"type"`
	APIVersion   string                 `json:"apiVersion"`
	Kind         string                 `json:"kind"`
	Namespace    string                 `json:"namespace,omitempty"`
	Specification map[string]interface{} `json:"specification"`
	Dependencies []string               `json:"dependencies"`
}

// TemplateParameter represents a parameter in a deployment template
type TemplateParameter struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"`
	Description  string      `json:"description"`
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"defaultValue,omitempty"`
	AllowedValues []interface{} `json:"allowedValues,omitempty"`
	Validation   *ParameterValidation `json:"validation,omitempty"`
}

// ParameterValidation represents validation rules for template parameters
type ParameterValidation struct {
	MinLength *int     `json:"minLength,omitempty"`
	MaxLength *int     `json:"maxLength,omitempty"`
	Pattern   string   `json:"pattern,omitempty"`
	MinValue  *float64 `json:"minValue,omitempty"`
	MaxValue  *float64 `json:"maxValue,omitempty"`
}

// TemplateDocumentation represents generated documentation for a template
type TemplateDocumentation struct {
	Overview     string                `json:"overview"`
	Description  string                `json:"description"`
	Parameters   []TemplateParameter   `json:"parameters"`
	Resources    []TemplateResource    `json:"resources"`
	Examples     []TemplateExample     `json:"examples"`
	Changelog    []TemplateChange      `json:"changelog"`
	GeneratedAt  time.Time             `json:"generatedAt"`
}

// TemplateExample represents an example usage of a template
type TemplateExample struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
	Expected    map[string]interface{} `json:"expected"`
}

// TemplateChange represents a change in template version
type TemplateChange struct {
	Version     string    `json:"version"`
	Date        time.Time `json:"date"`
	Author      string    `json:"author"`
	Description string    `json:"description"`
	Changes     []string  `json:"changes"`
	Breaking    bool      `json:"breaking"`
}

// CatalogStatistics represents catalog service statistics
type CatalogStatistics struct {
	ResourceTypeCount       int       `json:"resourceTypeCount"`
	DeploymentTemplateCount int       `json:"deploymentTemplateCount"`
	TemplatesByCategory     map[string]int `json:"templatesByCategory"`
	TemplatesByType         map[string]int `json:"templatesByType"`
	ValidationCount         int64     `json:"validationCount"`
	ProcessingCount         int64     `json:"processingCount"`
	LastUpdated             time.Time `json:"lastUpdated"`
}

// NewCatalogService creates a new catalog service
func NewCatalogService() *CatalogService {
	backgroundCtx, backgroundCancel := context.WithCancel(context.Background())
	
	service := &CatalogService{
		resourceTypes:       make(map[string]*models.ResourceType),
		deploymentTemplates: make(map[string]*models.DeploymentTemplate),
		templateValidator:   NewDefaultTemplateValidator(),
		templateProcessor:   NewDefaultTemplateProcessor(),
		backgroundCtx:       backgroundCtx,
		backgroundCancel:    backgroundCancel,
		stats: &CatalogStatistics{
			TemplatesByCategory: make(map[string]int),
			TemplatesByType:     make(map[string]int),
			LastUpdated:         time.Now(),
		},
	}
	
	// Initialize default resource types
	service.initializeDefaultResourceTypes()
	
	// Start background maintenance
	go service.startBackgroundMaintenance()
	
	return service
}

// Resource Type Management

// RegisterResourceType registers a new resource type
func (c *CatalogService) RegisterResourceType(ctx context.Context, resourceType *models.ResourceType) error {
	logger := log.FromContext(ctx)
	logger.Info("registering resource type", "typeID", resourceType.ResourceTypeID, "name", resourceType.Name)
	
	// Validate resource type
	if err := c.validateResourceType(resourceType); err != nil {
		return fmt.Errorf("resource type validation failed: %w", err)
	}
	
	// Set timestamps
	resourceType.CreatedAt = time.Now()
	resourceType.UpdatedAt = time.Now()
	
	// Store resource type
	c.rtMutex.Lock()
	c.resourceTypes[resourceType.ResourceTypeID] = resourceType
	c.rtMutex.Unlock()
	
	// Update statistics
	c.updateResourceTypeStats()
	
	logger.Info("resource type registered successfully", "typeID", resourceType.ResourceTypeID)
	return nil
}

// GetResourceType retrieves a resource type by ID
func (c *CatalogService) GetResourceType(ctx context.Context, resourceTypeID string) (*models.ResourceType, error) {
	c.rtMutex.RLock()
	defer c.rtMutex.RUnlock()
	
	resourceType, exists := c.resourceTypes[resourceTypeID]
	if !exists {
		return nil, fmt.Errorf("resource type not found: %s", resourceTypeID)
	}
	
	// Return a copy to prevent modification
	return c.copyResourceType(resourceType), nil
}

// ListResourceTypes lists resource types with optional filtering
func (c *CatalogService) ListResourceTypes(ctx context.Context, filter *models.ResourceTypeFilter) ([]*models.ResourceType, error) {
	c.rtMutex.RLock()
	defer c.rtMutex.RUnlock()
	
	var resourceTypes []*models.ResourceType
	
	for _, rt := range c.resourceTypes {
		if c.matchesResourceTypeFilter(rt, filter) {
			resourceTypes = append(resourceTypes, c.copyResourceType(rt))
		}
	}
	
	// Apply sorting and pagination
	resourceTypes = c.sortAndPaginateResourceTypes(resourceTypes, filter)
	
	return resourceTypes, nil
}

// UpdateResourceType updates an existing resource type
func (c *CatalogService) UpdateResourceType(ctx context.Context, resourceTypeID string, updates map[string]interface{}) error {
	logger := log.FromContext(ctx)
	logger.Info("updating resource type", "typeID", resourceTypeID)
	
	c.rtMutex.Lock()
	defer c.rtMutex.Unlock()
	
	resourceType, exists := c.resourceTypes[resourceTypeID]
	if !exists {
		return fmt.Errorf("resource type not found: %s", resourceTypeID)
	}
	
	// Apply updates
	if err := c.applyResourceTypeUpdates(resourceType, updates); err != nil {
		return fmt.Errorf("failed to apply updates: %w", err)
	}
	
	// Validate updated resource type
	if err := c.validateResourceType(resourceType); err != nil {
		return fmt.Errorf("updated resource type validation failed: %w", err)
	}
	
	// Update timestamp
	resourceType.UpdatedAt = time.Now()
	
	logger.Info("resource type updated successfully", "typeID", resourceTypeID)
	return nil
}

// UnregisterResourceType removes a resource type from the catalog
func (c *CatalogService) UnregisterResourceType(ctx context.Context, resourceTypeID string) error {
	logger := log.FromContext(ctx)
	logger.Info("unregistering resource type", "typeID", resourceTypeID)
	
	c.rtMutex.Lock()
	defer c.rtMutex.Unlock()
	
	if _, exists := c.resourceTypes[resourceTypeID]; !exists {
		return fmt.Errorf("resource type not found: %s", resourceTypeID)
	}
	
	// TODO: Check if resource type is in use by any resources
	// For now, allow removal
	
	delete(c.resourceTypes, resourceTypeID)
	
	// Update statistics
	c.updateResourceTypeStats()
	
	logger.Info("resource type unregistered successfully", "typeID", resourceTypeID)
	return nil
}

// Deployment Template Management

// RegisterDeploymentTemplate registers a new deployment template
func (c *CatalogService) RegisterDeploymentTemplate(ctx context.Context, template *models.DeploymentTemplate) error {
	logger := log.FromContext(ctx)
	logger.Info("registering deployment template", "templateID", template.DeploymentTemplateID, "name", template.Name)
	
	// Validate template
	if err := c.templateValidator.ValidateTemplate(ctx, template); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}
	
	// Set timestamps
	template.CreatedAt = time.Now()
	template.UpdatedAt = time.Now()
	
	// Store template
	c.dtMutex.Lock()
	c.deploymentTemplates[template.DeploymentTemplateID] = template
	c.dtMutex.Unlock()
	
	// Update statistics
	c.updateTemplateStats()
	
	logger.Info("deployment template registered successfully", "templateID", template.DeploymentTemplateID)
	return nil
}

// GetDeploymentTemplate retrieves a deployment template by ID
func (c *CatalogService) GetDeploymentTemplate(ctx context.Context, templateID string) (*models.DeploymentTemplate, error) {
	c.dtMutex.RLock()
	defer c.dtMutex.RUnlock()
	
	template, exists := c.deploymentTemplates[templateID]
	if !exists {
		return nil, fmt.Errorf("deployment template not found: %s", templateID)
	}
	
	// Return a copy to prevent modification
	return c.copyDeploymentTemplate(template), nil
}

// ListDeploymentTemplates lists deployment templates with optional filtering
func (c *CatalogService) ListDeploymentTemplates(ctx context.Context, filter *models.DeploymentTemplateFilter) ([]*models.DeploymentTemplate, error) {
	c.dtMutex.RLock()
	defer c.dtMutex.RUnlock()
	
	var templates []*models.DeploymentTemplate
	
	for _, template := range c.deploymentTemplates {
		if c.matchesTemplateFilter(template, filter) {
			templates = append(templates, c.copyDeploymentTemplate(template))
		}
	}
	
	// Apply sorting and pagination
	templates = c.sortAndPaginateTemplates(templates, filter)
	
	return templates, nil
}

// UpdateDeploymentTemplate updates an existing deployment template
func (c *CatalogService) UpdateDeploymentTemplate(ctx context.Context, templateID string, updates map[string]interface{}) error {
	logger := log.FromContext(ctx)
	logger.Info("updating deployment template", "templateID", templateID)
	
	c.dtMutex.Lock()
	defer c.dtMutex.Unlock()
	
	template, exists := c.deploymentTemplates[templateID]
	if !exists {
		return fmt.Errorf("deployment template not found: %s", templateID)
	}
	
	// Apply updates
	if err := c.applyTemplateUpdates(template, updates); err != nil {
		return fmt.Errorf("failed to apply updates: %w", err)
	}
	
	// Validate updated template
	if err := c.templateValidator.ValidateTemplate(context.Background(), template); err != nil {
		return fmt.Errorf("updated template validation failed: %w", err)
	}
	
	// Update timestamp
	template.UpdatedAt = time.Now()
	
	logger.Info("deployment template updated successfully", "templateID", templateID)
	return nil
}

// UnregisterDeploymentTemplate removes a deployment template from the catalog
func (c *CatalogService) UnregisterDeploymentTemplate(ctx context.Context, templateID string) error {
	logger := log.FromContext(ctx)
	logger.Info("unregistering deployment template", "templateID", templateID)
	
	c.dtMutex.Lock()
	defer c.dtMutex.Unlock()
	
	if _, exists := c.deploymentTemplates[templateID]; !exists {
		return fmt.Errorf("deployment template not found: %s", templateID)
	}
	
	// TODO: Check if template is in use by any deployments
	// For now, allow removal
	
	delete(c.deploymentTemplates, templateID)
	
	// Update statistics
	c.updateTemplateStats()
	
	logger.Info("deployment template unregistered successfully", "templateID", templateID)
	return nil
}

// Template Processing and Validation

// ValidateTemplate validates a deployment template
func (c *CatalogService) ValidateTemplate(ctx context.Context, template *models.DeploymentTemplate) error {
	return c.templateValidator.ValidateTemplate(ctx, template)
}

// ProcessTemplate processes a deployment template with parameters
func (c *CatalogService) ProcessTemplate(ctx context.Context, templateID string, parameters map[string]interface{}) (*ProcessedTemplate, error) {
	logger := log.FromContext(ctx)
	logger.Info("processing deployment template", "templateID", templateID)
	
	// Get template
	template, err := c.GetDeploymentTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	
	// Process template
	processed, err := c.templateProcessor.ProcessTemplate(ctx, template, parameters)
	if err != nil {
		return nil, fmt.Errorf("template processing failed: %w", err)
	}
	
	// Update statistics
	c.stats.ProcessingCount++
	
	logger.Info("deployment template processed successfully", "templateID", templateID)
	return processed, nil
}

// GetTemplateParameters extracts parameters from a deployment template
func (c *CatalogService) GetTemplateParameters(ctx context.Context, templateID string) ([]TemplateParameter, error) {
	template, err := c.GetDeploymentTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	
	return c.templateProcessor.ExtractParameters(ctx, template)
}

// GenerateTemplateDocumentation generates documentation for a deployment template
func (c *CatalogService) GenerateTemplateDocumentation(ctx context.Context, templateID string) (*TemplateDocumentation, error) {
	template, err := c.GetDeploymentTemplate(ctx, templateID)
	if err != nil {
		return nil, fmt.Errorf("failed to get template: %w", err)
	}
	
	return c.templateProcessor.GenerateDocumentation(ctx, template)
}

// Statistics and Metrics

// GetStatistics returns catalog service statistics
func (c *CatalogService) GetStatistics(ctx context.Context) *CatalogStatistics {
	// Update statistics before returning
	c.updateStats()
	
	// Return a copy to prevent modification
	statsCopy := *c.stats
	return &statsCopy
}

// Shutdown gracefully shuts down the catalog service
func (c *CatalogService) Shutdown(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("shutting down catalog service")
	
	// Cancel background services
	c.backgroundCancel()
	
	logger.Info("catalog service shutdown completed")
	return nil
}

// Private helper methods

func (c *CatalogService) initializeDefaultResourceTypes() {
	// Initialize common O-RAN resource types
	defaultTypes := []*models.ResourceType{
		{
			ResourceTypeID: "compute-deployment",
			Name:           "Kubernetes Deployment",
			Description:    "Kubernetes Deployment for compute workloads",
			Vendor:         "Kubernetes",
			Model:          "Deployment",
			Version:        "apps/v1",
			Specifications: &models.ResourceTypeSpec{
				Category: models.ResourceCategoryCompute,
				MinResources: map[string]string{
					"cpu":    "100m",
					"memory": "128Mi",
				},
				DefaultResources: map[string]string{
					"cpu":    "500m",
					"memory": "512Mi",
				},
			},
			SupportedActions: []string{"create", "update", "delete", "scale"},
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ResourceTypeID: "network-service",
			Name:           "Kubernetes Service",
			Description:    "Kubernetes Service for network connectivity",
			Vendor:         "Kubernetes",
			Model:          "Service",
			Version:        "v1",
			Specifications: &models.ResourceTypeSpec{
				Category: models.ResourceCategoryNetwork,
				Properties: map[string]interface{}{
					"serviceTypes": []string{"ClusterIP", "NodePort", "LoadBalancer"},
					"protocols":    []string{"TCP", "UDP"},
				},
			},
			SupportedActions: []string{"create", "update", "delete"},
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
		{
			ResourceTypeID: "storage-pvc",
			Name:           "Persistent Volume Claim",
			Description:    "Kubernetes Persistent Volume Claim for storage",
			Vendor:         "Kubernetes",
			Model:          "PersistentVolumeClaim",
			Version:        "v1",
			Specifications: &models.ResourceTypeSpec{
				Category: models.ResourceCategoryStorage,
				Properties: map[string]interface{}{
					"accessModes": []string{"ReadWriteOnce", "ReadOnlyMany", "ReadWriteMany"},
					"volumeModes": []string{"Filesystem", "Block"},
				},
			},
			SupportedActions: []string{"create", "delete"},
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
		},
	}
	
	// Register default types
	for _, resourceType := range defaultTypes {
		c.resourceTypes[resourceType.ResourceTypeID] = resourceType
	}
}

func (c *CatalogService) validateResourceType(resourceType *models.ResourceType) error {
	if resourceType.ResourceTypeID == "" {
		return fmt.Errorf("resource type ID is required")
	}
	if resourceType.Name == "" {
		return fmt.Errorf("resource type name is required")
	}
	if resourceType.Specifications == nil {
		return fmt.Errorf("resource type specifications are required")
	}
	if resourceType.Specifications.Category == "" {
		return fmt.Errorf("resource type category is required")
	}
	
	// Validate category
	validCategories := []string{
		models.ResourceCategoryCompute,
		models.ResourceCategoryStorage,
		models.ResourceCategoryNetwork,
		models.ResourceCategoryAccelerator,
	}
	
	validCategory := false
	for _, category := range validCategories {
		if resourceType.Specifications.Category == category {
			validCategory = true
			break
		}
	}
	
	if !validCategory {
		return fmt.Errorf("invalid resource type category: %s", resourceType.Specifications.Category)
	}
	
	return nil
}

func (c *CatalogService) matchesResourceTypeFilter(rt *models.ResourceType, filter *models.ResourceTypeFilter) bool {
	if filter == nil {
		return true
	}
	
	// Check names filter
	if len(filter.Names) > 0 {
		found := false
		for _, name := range filter.Names {
			if rt.Name == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check categories filter
	if len(filter.Categories) > 0 && rt.Specifications != nil {
		found := false
		for _, category := range filter.Categories {
			if rt.Specifications.Category == category {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check vendors filter
	if len(filter.Vendors) > 0 {
		found := false
		for _, vendor := range filter.Vendors {
			if rt.Vendor == vendor {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check models filter
	if len(filter.Models) > 0 {
		found := false
		for _, model := range filter.Models {
			if rt.Model == model {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check versions filter
	if len(filter.Versions) > 0 {
		found := false
		for _, version := range filter.Versions {
			if rt.Version == version {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	return true
}

func (c *CatalogService) matchesTemplateFilter(template *models.DeploymentTemplate, filter *models.DeploymentTemplateFilter) bool {
	if filter == nil {
		return true
	}
	
	// Check names filter
	if len(filter.Names) > 0 {
		found := false
		for _, name := range filter.Names {
			if template.Name == name {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check categories filter
	if len(filter.Categories) > 0 {
		found := false
		for _, category := range filter.Categories {
			if template.Category == category {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check types filter
	if len(filter.Types) > 0 {
		found := false
		for _, templateType := range filter.Types {
			if template.Type == templateType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check versions filter
	if len(filter.Versions) > 0 {
		found := false
		for _, version := range filter.Versions {
			if template.Version == version {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	// Check authors filter
	if len(filter.Authors) > 0 {
		found := false
		for _, author := range filter.Authors {
			if template.Author == author {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	
	return true
}

func (c *CatalogService) copyResourceType(rt *models.ResourceType) *models.ResourceType {
	// Deep copy the resource type
	copy := *rt
	if rt.Specifications != nil {
		specCopy := *rt.Specifications
		copy.Specifications = &specCopy
	}
	if rt.AlarmDictionary != nil {
		alarmCopy := *rt.AlarmDictionary
		copy.AlarmDictionary = &alarmCopy
	}
	return &copy
}

func (c *CatalogService) copyDeploymentTemplate(template *models.DeploymentTemplate) *models.DeploymentTemplate {
	// Deep copy the deployment template
	copy := *template
	// Note: For production, implement proper deep copy for nested structures
	return &copy
}

func (c *CatalogService) sortAndPaginateResourceTypes(resourceTypes []*models.ResourceType, filter *models.ResourceTypeFilter) []*models.ResourceType {
	// Apply sorting if specified
	// Apply pagination if specified
	if filter != nil && filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit
		if start >= len(resourceTypes) {
			return []*models.ResourceType{}
		}
		if end > len(resourceTypes) {
			end = len(resourceTypes)
		}
		return resourceTypes[start:end]
	}
	return resourceTypes
}

func (c *CatalogService) sortAndPaginateTemplates(templates []*models.DeploymentTemplate, filter *models.DeploymentTemplateFilter) []*models.DeploymentTemplate {
	// Apply sorting if specified
	// Apply pagination if specified
	if filter != nil && filter.Limit > 0 {
		start := filter.Offset
		end := start + filter.Limit
		if start >= len(templates) {
			return []*models.DeploymentTemplate{}
		}
		if end > len(templates) {
			end = len(templates)
		}
		return templates[start:end]
	}
	return templates
}

func (c *CatalogService) applyResourceTypeUpdates(rt *models.ResourceType, updates map[string]interface{}) error {
	// Apply updates to resource type fields
	// This would be more comprehensive in a full implementation
	if name, ok := updates["name"].(string); ok {
		rt.Name = name
	}
	if description, ok := updates["description"].(string); ok {
		rt.Description = description
	}
	return nil
}

func (c *CatalogService) applyTemplateUpdates(template *models.DeploymentTemplate, updates map[string]interface{}) error {
	// Apply updates to template fields
	// This would be more comprehensive in a full implementation
	if name, ok := updates["name"].(string); ok {
		template.Name = name
	}
	if description, ok := updates["description"].(string); ok {
		template.Description = description
	}
	return nil
}

func (c *CatalogService) updateStats() {
	c.rtMutex.RLock()
	resourceTypeCount := len(c.resourceTypes)
	c.rtMutex.RUnlock()
	
	c.dtMutex.RLock()
	templateCount := len(c.deploymentTemplates)
	templatesByCategory := make(map[string]int)
	templatesByType := make(map[string]int)
	
	for _, template := range c.deploymentTemplates {
		templatesByCategory[template.Category]++
		templatesByType[template.Type]++
	}
	c.dtMutex.RUnlock()
	
	c.stats.ResourceTypeCount = resourceTypeCount
	c.stats.DeploymentTemplateCount = templateCount
	c.stats.TemplatesByCategory = templatesByCategory
	c.stats.TemplatesByType = templatesByType
	c.stats.LastUpdated = time.Now()
}

func (c *CatalogService) updateResourceTypeStats() {
	c.updateStats()
}

func (c *CatalogService) updateTemplateStats() {
	c.updateStats()
}

func (c *CatalogService) startBackgroundMaintenance() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-c.backgroundCtx.Done():
			return
		case <-ticker.C:
			c.performMaintenance()
		}
	}
}

func (c *CatalogService) performMaintenance() {
	// Update statistics
	c.updateStats()
	
	// Perform any cleanup operations
	// This could include removing expired templates, validating integrity, etc.
}

// Default implementations

// DefaultTemplateValidator provides basic template validation
type DefaultTemplateValidator struct{}

func NewDefaultTemplateValidator() TemplateValidator {
	return &DefaultTemplateValidator{}
}

func (v *DefaultTemplateValidator) ValidateTemplate(ctx context.Context, template *models.DeploymentTemplate) error {
	if template.DeploymentTemplateID == "" {
		return fmt.Errorf("deployment template ID is required")
	}
	if template.Name == "" {
		return fmt.Errorf("template name is required")
	}
	if template.Version == "" {
		return fmt.Errorf("template version is required")
	}
	if template.Category == "" {
		return fmt.Errorf("template category is required")
	}
	if template.Type == "" {
		return fmt.Errorf("template type is required")
	}
	if template.Content == nil {
		return fmt.Errorf("template content is required")
	}
	
	// Validate template type
	validTypes := []string{
		models.TemplateTypeHelm,
		models.TemplateTypeKubernetes,
		models.TemplateTypeTerraform,
		models.TemplateTypeAnsible,
	}
	
	validType := false
	for _, validT := range validTypes {
		if template.Type == validT {
			validType = true
			break
		}
	}
	
	if !validType {
		return fmt.Errorf("invalid template type: %s", template.Type)
	}
	
	return nil
}

func (v *DefaultTemplateValidator) ValidateTemplateContent(ctx context.Context, content string, templateType string) error {
	// Basic content validation based on template type
	if content == "" {
		return fmt.Errorf("template content cannot be empty")
	}
	
	// Type-specific validation would be implemented here
	return nil
}

func (v *DefaultTemplateValidator) ValidateInputSchema(ctx context.Context, schema string) error {
	// JSON schema validation would be implemented here
	return nil
}

func (v *DefaultTemplateValidator) ValidateOutputSchema(ctx context.Context, schema string) error {
	// JSON schema validation would be implemented here
	return nil
}

// DefaultTemplateProcessor provides basic template processing
type DefaultTemplateProcessor struct{}

func NewDefaultTemplateProcessor() TemplateProcessor {
	return &DefaultTemplateProcessor{}
}

func (p *DefaultTemplateProcessor) ProcessTemplate(ctx context.Context, template *models.DeploymentTemplate, parameters map[string]interface{}) (*ProcessedTemplate, error) {
	// Basic template processing
	// In a full implementation, this would render templates with actual templating engines
	processed := &ProcessedTemplate{
		RenderedContent: string(template.Content.Raw),
		Parameters:      parameters,
		Metadata: map[string]string{
			"template_id": template.DeploymentTemplateID,
			"template_type": template.Type,
			"processed_at": time.Now().Format(time.RFC3339),
		},
	}
	
	return processed, nil
}

func (p *DefaultTemplateProcessor) ExtractParameters(ctx context.Context, template *models.DeploymentTemplate) ([]TemplateParameter, error) {
	// Extract parameters from template schema
	// This would parse the input schema and extract parameter definitions
	var parameters []TemplateParameter
	
	// Basic implementation - would be enhanced for production
	if template.InputSchema != nil {
		// Parse JSON schema and extract parameters
		// For now, return empty list
	}
	
	return parameters, nil
}

func (p *DefaultTemplateProcessor) GenerateDocumentation(ctx context.Context, template *models.DeploymentTemplate) (*TemplateDocumentation, error) {
	// Generate documentation for template
	doc := &TemplateDocumentation{
		Overview:    fmt.Sprintf("Documentation for template: %s", template.Name),
		Description: template.Description,
		GeneratedAt: time.Now(),
	}
	
	// Extract parameters
	parameters, err := p.ExtractParameters(ctx, template)
	if err != nil {
		return nil, fmt.Errorf("failed to extract parameters: %w", err)
	}
	doc.Parameters = parameters
	
	return doc, nil
}