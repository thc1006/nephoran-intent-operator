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

package blueprint

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	v1 "github.com/nephio-project/nephoran-intent-operator/api/v1"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

// Template represents a blueprint template.

type Template struct {
	ID string `json:"id" yaml:"id"`

	Name string `json:"name" yaml:"name"`

	Description string `json:"description" yaml:"description"`

	Version string `json:"version" yaml:"version"`

	Type TemplateType `json:"type" yaml:"type"`

	Category TemplateCategory `json:"category" yaml:"category"`

	Tags []string `json:"tags" yaml:"tags"`

	Author string `json:"author" yaml:"author"`

	License string `json:"license" yaml:"license"`

	// Component targeting.

	TargetComponents []v1.ORANComponent `json:"targetComponents" yaml:"targetComponents"`

	IntentTypes []v1.IntentType `json:"intentTypes" yaml:"intentTypes"`

	// Dependencies.

	Dependencies []TemplateDependency `json:"dependencies" yaml:"dependencies"`

	Prerequisites []string `json:"prerequisites" yaml:"prerequisites"`

	// Template content and metadata.

	Files map[string]string `json:"files" yaml:"files"`

	Parameters []TemplateParameter `json:"parameters" yaml:"parameters"`

	Outputs []TemplateOutput `json:"outputs" yaml:"outputs"`

	// Validation and compliance.

	ORANCompliant bool `json:"oranCompliant" yaml:"oranCompliant"`

	Validated bool `json:"validated" yaml:"validated"`

	ValidationErrors []string `json:"validationErrors,omitempty" yaml:"validationErrors,omitempty"`

	// Metadata.

	CreatedAt time.Time `json:"createdAt" yaml:"createdAt"`

	UpdatedAt time.Time `json:"updatedAt" yaml:"updatedAt"`

	UsageCount int64 `json:"usageCount" yaml:"usageCount"`

	Rating float64 `json:"rating" yaml:"rating"`

	Checksum string `json:"checksum" yaml:"checksum"`

	// Repository information.

	Repository string `json:"repository" yaml:"repository"`

	Path string `json:"path" yaml:"path"`

	Branch string `json:"branch" yaml:"branch"`

	CommitHash string `json:"commitHash" yaml:"commitHash"`
}

// TemplateType defines the type of blueprint template.

type TemplateType string

const (

	// TemplateTypeDeployment holds templatetypedeployment value.

	TemplateTypeDeployment TemplateType = "deployment"

	// TemplateTypeService holds templatetypeservice value.

	TemplateTypeService TemplateType = "service"

	// TemplateTypeConfiguration holds templatetypeconfiguration value.

	TemplateTypeConfiguration TemplateType = "configuration"

	// TemplateTypeNetworking holds templatetypenetworking value.

	TemplateTypeNetworking TemplateType = "networking"

	// TemplateTypeMonitoring holds templatetypemonitoring value.

	TemplateTypeMonitoring TemplateType = "monitoring"

	// TemplateTypeSecurity holds templatetypesecurity value.

	TemplateTypeSecurity TemplateType = "security"

	// TemplateTypeComplete holds templatetypecomplete value.

	TemplateTypeComplete TemplateType = "complete"
)

// TemplateCategory defines the category of blueprint template.

type TemplateCategory string

const (

	// TemplateCategoryCore holds templatecategorycore value.

	TemplateCategoryCore TemplateCategory = "5g-core"

	// TemplateCategoryRAN holds templatecategoryran value.

	TemplateCategoryRAN TemplateCategory = "ran"

	// TemplateCategoryORAN holds templatecategoryoran value.

	TemplateCategoryORAN TemplateCategory = "oran"

	// TemplateCategoryEdge holds templatecategoryedge value.

	TemplateCategoryEdge TemplateCategory = "edge"

	// TemplateCategorySlicing holds templatecategoryslicing value.

	TemplateCategorySlicing TemplateCategory = "slicing"

	// TemplateCategoryCloud holds templatecategorycloud value.

	TemplateCategoryCloud TemplateCategory = "cloud"

	// TemplateCategoryNetwork holds templatecategorynetwork value.

	TemplateCategoryNetwork TemplateCategory = "network"

	// TemplateCategoryGeneric holds templatecategorygeneric value.

	TemplateCategoryGeneric TemplateCategory = "generic"
)

// TemplateDependency represents a template dependency.

type TemplateDependency struct {
	TemplateID string `json:"templateId" yaml:"templateId"`

	Version string `json:"version" yaml:"version"`

	Required bool `json:"required" yaml:"required"`

	Reason string `json:"reason" yaml:"reason"`

	Compatibility []string `json:"compatibility" yaml:"compatibility"`
}

// TemplateParameter represents a configurable parameter in a template.

type TemplateParameter struct {
	Name string `json:"name" yaml:"name"`

	Type string `json:"type" yaml:"type"`

	Description string `json:"description" yaml:"description"`

	Default interface{} `json:"default,omitempty" yaml:"default,omitempty"`

	Required bool `json:"required" yaml:"required"`

	Constraints []string `json:"constraints,omitempty" yaml:"constraints,omitempty"`

	Options []string `json:"options,omitempty" yaml:"options,omitempty"`

	Pattern string `json:"pattern,omitempty" yaml:"pattern,omitempty"`
}

// TemplateOutput represents an output from a template.

type TemplateOutput struct {
	Name string `json:"name" yaml:"name"`

	Type string `json:"type" yaml:"type"`

	Description string `json:"description" yaml:"description"`

	Export bool `json:"export" yaml:"export"`
}

// SearchCriteria defines criteria for template search and filtering.

type SearchCriteria struct {

	// Basic filters.

	Query string `json:"query,omitempty"`

	Type TemplateType `json:"type,omitempty"`

	Category TemplateCategory `json:"category,omitempty"`

	Tags []string `json:"tags,omitempty"`

	// Component filters.

	TargetComponents []v1.ORANComponent `json:"targetComponents,omitempty"`

	IntentTypes []v1.IntentType `json:"intentTypes,omitempty"`

	// Quality filters.

	ORANCompliant *bool `json:"oranCompliant,omitempty"`

	MinRating *float64 `json:"minRating,omitempty"`

	Validated *bool `json:"validated,omitempty"`

	// Version and dependency filters.

	Version string `json:"version,omitempty"`

	MinVersion string `json:"minVersion,omitempty"`

	MaxVersion string `json:"maxVersion,omitempty"`

	// Sorting and pagination.

	SortBy string `json:"sortBy,omitempty"`

	SortOrder string `json:"sortOrder,omitempty"`

	Limit int `json:"limit,omitempty"`

	Offset int `json:"offset,omitempty"`
}

// Catalog manages blueprint template repository and provides discovery capabilities.

type Catalog struct {
	config *BlueprintConfig

	logger *zap.Logger

	// Template storage.

	templates sync.Map // map[string]*Template

	templatesByType map[TemplateType][]*Template

	templatesByCategory map[TemplateCategory][]*Template

	templatesByComponent map[string][]*Template

	// Repository management.

	repositories []TemplateRepository

	repoCache sync.Map

	// Indexing and search.

	searchIndex *SearchIndex

	dependencyGraph *DependencyGraph

	// Cache and performance.

	cacheHits int64

	cacheMisses int64

	lastSync time.Time

	syncMutex sync.RWMutex

	// Background operations.

	ctx context.Context

	cancel context.CancelFunc

	wg sync.WaitGroup
}

// TemplateRepository represents a template repository configuration.

type TemplateRepository struct {
	Name string `json:"name" yaml:"name"`

	URL string `json:"url" yaml:"url"`

	Branch string `json:"branch" yaml:"branch"`

	Path string `json:"path" yaml:"path"`

	Credentials string `json:"credentials,omitempty" yaml:"credentials,omitempty"`

	Priority int `json:"priority" yaml:"priority"`

	Enabled bool `json:"enabled" yaml:"enabled"`
}

// SearchIndex provides fast template search capabilities.

type SearchIndex struct {
	nameIndex map[string][]*Template

	tagIndex map[string][]*Template

	componentIndex map[string][]*Template

	typeIndex map[v1.IntentType][]*Template

	textIndex map[string][]*Template

	mutex sync.RWMutex
}

// DependencyGraph manages template dependencies and compatibility.

type DependencyGraph struct {
	dependencies map[string][]string

	dependents map[string][]string

	conflicts map[string][]string

	compatibility map[string][]string

	mutex sync.RWMutex
}

// NewCatalog creates a new blueprint catalog.

func NewCatalog(config *BlueprintConfig, logger *zap.Logger) (*Catalog, error) {

	if config == nil {

		config = DefaultBlueprintConfig()

	}

	if logger == nil {

		logger = zap.NewNop()

	}

	ctx, cancel := context.WithCancel(context.Background())

	catalog := &Catalog{

		config: config,

		logger: logger,

		templatesByType: make(map[TemplateType][]*Template),

		templatesByCategory: make(map[TemplateCategory][]*Template),

		templatesByComponent: make(map[string][]*Template),

		repositories: []TemplateRepository{},

		searchIndex: NewSearchIndex(),

		dependencyGraph: NewDependencyGraph(),

		ctx: ctx,

		cancel: cancel,
	}

	// Initialize default repositories.

	if err := catalog.initializeDefaultRepositories(); err != nil {

		cancel()

		return nil, fmt.Errorf("failed to initialize repositories: %w", err)

	}

	// Start background operations.

	catalog.startBackgroundOperations()

	logger.Info("Blueprint catalog initialized",

		zap.Int("repositories", len(catalog.repositories)))

	return catalog, nil

}

// initializeDefaultRepositories sets up default template repositories.

func (c *Catalog) initializeDefaultRepositories() error {

	defaultRepos := []TemplateRepository{

		{

			Name: "nephio-official",

			URL: "https://github.com/nephio-project/catalog.git",

			Branch: "main",

			Path: "templates",

			Priority: 10,

			Enabled: true,
		},

		{

			Name: "oran-alliance",

			URL: "https://github.com/o-ran-sc/scp-oam-modeling.git",

			Branch: "master",

			Path: "yang-models",

			Priority: 8,

			Enabled: true,
		},

		{

			Name: "free5gc-templates",

			URL: "https://github.com/free5gc/free5gc-k8s.git",

			Branch: "main",

			Path: "templates",

			Priority: 6,

			Enabled: true,
		},
	}

	c.repositories = append(c.repositories, defaultRepos...)

	return nil

}

// startBackgroundOperations starts background worker goroutines.

func (c *Catalog) startBackgroundOperations() {

	// Template synchronization worker.

	c.wg.Add(1)

	go c.syncWorker()

	// Index maintenance worker.

	c.wg.Add(1)

	go c.indexMaintenanceWorker()

	// Cache cleanup worker.

	c.wg.Add(1)

	go c.cacheCleanupWorker()

}

// FindTemplates searches for templates matching the given criteria.

func (c *Catalog) FindTemplates(ctx context.Context, criteria *SearchCriteria) ([]*Template, error) {

	startTime := time.Now()

	c.logger.Debug("Searching templates",

		zap.Any("criteria", criteria))

	// Try cache first.

	cacheKey := c.buildCacheKey(criteria)

	if cached, ok := c.repoCache.Load(cacheKey); ok {

		if cacheEntry, ok := cached.(map[string]interface{}); ok {

			if expiry, ok := cacheEntry["expiry"].(time.Time); ok && time.Now().Before(expiry) {

				if templates, ok := cacheEntry["templates"].([]*Template); ok {

					c.cacheHits++

					c.logger.Debug("Template search cache hit",

						zap.String("cache_key", cacheKey),

						zap.Int("results", len(templates)))

					return templates, nil

				}

			}

		}

	}

	c.cacheMisses++

	// Perform search.

	results := c.performSearch(criteria)

	// Apply additional filters.

	filtered := c.applyFilters(results, criteria)

	// Sort results.

	sorted := c.sortTemplates(filtered, criteria.SortBy, criteria.SortOrder)

	// Apply pagination.

	paginated := c.paginateResults(sorted, criteria.Limit, criteria.Offset)

	// Cache results.

	c.cacheSearchResults(cacheKey, paginated)

	duration := time.Since(startTime)

	c.logger.Debug("Template search completed",

		zap.Duration("duration", duration),

		zap.Int("total_results", len(results)),

		zap.Int("filtered_results", len(filtered)),

		zap.Int("paginated_results", len(paginated)))

	return paginated, nil

}

// GetTemplate retrieves a specific template by ID.

func (c *Catalog) GetTemplate(ctx context.Context, templateID string) (*Template, error) {

	if template, ok := c.templates.Load(templateID); ok {

		if tmpl, ok := template.(*Template); ok {

			c.cacheHits++

			return tmpl, nil

		}

	}

	c.cacheMisses++

	return nil, fmt.Errorf("template not found: %s", templateID)

}

// GetTemplatesByComponent returns templates for specific components.

func (c *Catalog) GetTemplatesByComponent(ctx context.Context, components []v1.ORANComponent) ([]*Template, error) {

	var results []*Template

	for _, component := range components {

		if templates, ok := c.templatesByComponent[string(component)]; ok {

			results = append(results, templates...)

		}

	}

	// Remove duplicates.

	uniqueResults := c.removeDuplicateTemplates(results)

	return uniqueResults, nil

}

// GetRecommendedTemplates returns recommended templates for a NetworkIntent.

func (c *Catalog) GetRecommendedTemplates(ctx context.Context, intent *v1.NetworkIntent) ([]*Template, error) {

	criteria := &SearchCriteria{

		TargetComponents: intent.Spec.TargetComponents,

		IntentTypes: []v1.IntentType{intent.Spec.IntentType},

		ORANCompliant: &[]bool{true}[0],

		Validated: &[]bool{true}[0],

		MinRating: &[]float64{3.0}[0],

		SortBy: "rating",

		SortOrder: "desc",

		Limit: 10,
	}

	return c.FindTemplates(ctx, criteria)

}

// ValidateTemplate validates a template against O-RAN compliance and other criteria.

func (c *Catalog) ValidateTemplate(ctx context.Context, template *Template) error {

	var errors []string

	// Validate required fields.

	if template.ID == "" {

		errors = append(errors, "template ID is required")

	}

	if template.Name == "" {

		errors = append(errors, "template name is required")

	}

	if template.Version == "" {

		errors = append(errors, "template version is required")

	}

	// Validate files exist.

	if len(template.Files) == 0 {

		errors = append(errors, "template must contain at least one file")

	}

	// Validate O-RAN compliance if claimed.

	if template.ORANCompliant {

		if err := c.validateORANCompliance(template); err != nil {

			errors = append(errors, fmt.Sprintf("O-RAN compliance validation failed: %v", err))

		}

	}

	// Validate dependencies.

	if err := c.validateDependencies(template); err != nil {

		errors = append(errors, fmt.Sprintf("dependency validation failed: %v", err))

	}

	// Validate template syntax.

	if err := c.validateTemplateSyntax(template); err != nil {

		errors = append(errors, fmt.Sprintf("template syntax validation failed: %v", err))

	}

	template.ValidationErrors = errors

	template.Validated = len(errors) == 0

	if len(errors) > 0 {

		return fmt.Errorf("template validation failed: %v", errors)

	}

	return nil

}

// RegisterTemplate registers a new template in the catalog.

func (c *Catalog) RegisterTemplate(ctx context.Context, template *Template) error {

	// Validate template.

	if err := c.ValidateTemplate(ctx, template); err != nil {

		return fmt.Errorf("template validation failed: %w", err)

	}

	// Set metadata.

	now := time.Now()

	if template.CreatedAt.IsZero() {

		template.CreatedAt = now

	}

	template.UpdatedAt = now

	template.Checksum = c.calculateChecksum(template)

	// Store template.

	c.templates.Store(template.ID, template)

	// Update indexes.

	c.updateIndexes(template)

	c.logger.Info("Template registered",

		zap.String("template_id", template.ID),

		zap.String("template_name", template.Name),

		zap.String("version", template.Version))

	return nil

}

// UpdateTemplate updates an existing template.

func (c *Catalog) UpdateTemplate(ctx context.Context, template *Template) error {

	existing, err := c.GetTemplate(ctx, template.ID)

	if err != nil {

		return fmt.Errorf("template not found for update: %w", err)

	}

	// Preserve creation time and usage stats.

	template.CreatedAt = existing.CreatedAt

	template.UsageCount = existing.UsageCount

	// Update modification time and checksum.

	template.UpdatedAt = time.Now()

	template.Checksum = c.calculateChecksum(template)

	// Validate updated template.

	if err := c.ValidateTemplate(ctx, template); err != nil {

		return fmt.Errorf("template validation failed: %w", err)

	}

	// Update storage and indexes.

	c.templates.Store(template.ID, template)

	c.updateIndexes(template)

	c.logger.Info("Template updated",

		zap.String("template_id", template.ID),

		zap.String("template_name", template.Name),

		zap.String("version", template.Version))

	return nil

}

// RemoveTemplate removes a template from the catalog.

func (c *Catalog) RemoveTemplate(ctx context.Context, templateID string) error {

	template, err := c.GetTemplate(ctx, templateID)

	if err != nil {

		return err

	}

	// Check for dependencies.

	if dependents := c.dependencyGraph.GetDependents(templateID); len(dependents) > 0 {

		return fmt.Errorf("cannot remove template %s: has dependents %v", templateID, dependents)

	}

	// Remove from storage and indexes.

	c.templates.Delete(templateID)

	c.removeFromIndexes(template)

	c.logger.Info("Template removed",

		zap.String("template_id", templateID),

		zap.String("template_name", template.Name))

	return nil

}

// SyncRepositories synchronizes templates from all configured repositories.

func (c *Catalog) SyncRepositories(ctx context.Context) error {

	c.syncMutex.Lock()

	defer c.syncMutex.Unlock()

	c.logger.Info("Starting repository synchronization",

		zap.Int("repositories", len(c.repositories)))

	var allErrors []string

	for _, repo := range c.repositories {

		if !repo.Enabled {

			continue

		}

		if err := c.syncRepository(ctx, &repo); err != nil {

			allErrors = append(allErrors, fmt.Sprintf("repository %s: %v", repo.Name, err))

			c.logger.Error("Repository sync failed",

				zap.String("repository", repo.Name),

				zap.Error(err))

		}

	}

	c.lastSync = time.Now()

	if len(allErrors) > 0 {

		return fmt.Errorf("repository sync errors: %v", allErrors)

	}

	c.logger.Info("Repository synchronization completed",

		zap.Time("last_sync", c.lastSync))

	return nil

}

// syncRepository synchronizes templates from a single repository.

func (c *Catalog) syncRepository(ctx context.Context, repo *TemplateRepository) error {

	c.logger.Debug("Syncing repository",

		zap.String("repository", repo.Name),

		zap.String("url", repo.URL))

	// Clone or pull repository.

	repoPath, err := c.cloneRepository(ctx, repo)

	if err != nil {

		return fmt.Errorf("failed to clone repository: %w", err)

	}

	// Discover templates in repository.

	templates, err := c.discoverTemplates(repoPath, repo)

	if err != nil {

		return fmt.Errorf("failed to discover templates: %w", err)

	}

	// Register discovered templates.

	for _, template := range templates {

		template.Repository = repo.Name

		if err := c.RegisterTemplate(ctx, template); err != nil {

			c.logger.Warn("Failed to register template",

				zap.String("template_id", template.ID),

				zap.Error(err))

		}

	}

	c.logger.Debug("Repository sync completed",

		zap.String("repository", repo.Name),

		zap.Int("templates", len(templates)))

	return nil

}

// Implementation of search, indexing, and caching methods.

func (c *Catalog) performSearch(criteria *SearchCriteria) []*Template {

	var results []*Template

	if criteria.Query != "" {

		results = c.searchIndex.SearchByText(criteria.Query)

	} else {

		// Return all templates if no specific query.

		c.templates.Range(func(key, value interface{}) bool {

			if template, ok := value.(*Template); ok {

				results = append(results, template)

			}

			return true

		})

	}

	return results

}

func (c *Catalog) applyFilters(templates []*Template, criteria *SearchCriteria) []*Template {

	var filtered []*Template

	for _, template := range templates {

		if c.matchesFilters(template, criteria) {

			filtered = append(filtered, template)

		}

	}

	return filtered

}

func (c *Catalog) matchesFilters(template *Template, criteria *SearchCriteria) bool {

	// Type filter.

	if criteria.Type != "" && template.Type != criteria.Type {

		return false

	}

	// Category filter.

	if criteria.Category != "" && template.Category != criteria.Category {

		return false

	}

	// Component filter.

	if len(criteria.TargetComponents) > 0 {

		if !c.hasMatchingComponent(template.TargetComponents, criteria.TargetComponents) {

			return false

		}

	}

	// Intent type filter.

	if len(criteria.IntentTypes) > 0 {

		if !c.hasMatchingIntentType(template.IntentTypes, criteria.IntentTypes) {

			return false

		}

	}

	// O-RAN compliance filter.

	if criteria.ORANCompliant != nil && template.ORANCompliant != *criteria.ORANCompliant {

		return false

	}

	// Rating filter.

	if criteria.MinRating != nil && template.Rating < *criteria.MinRating {

		return false

	}

	// Validation filter.

	if criteria.Validated != nil && template.Validated != *criteria.Validated {

		return false

	}

	return true

}

func (c *Catalog) sortTemplates(templates []*Template, sortBy, sortOrder string) []*Template {

	if sortBy == "" {

		sortBy = "name"

	}

	if sortOrder == "" {

		sortOrder = "asc"

	}

	sort.Slice(templates, func(i, j int) bool {

		var less bool

		switch sortBy {

		case "name":

			less = templates[i].Name < templates[j].Name

		case "rating":

			less = templates[i].Rating < templates[j].Rating

		case "usage":

			less = templates[i].UsageCount < templates[j].UsageCount

		case "updated":

			less = templates[i].UpdatedAt.Before(templates[j].UpdatedAt)

		default:

			less = templates[i].Name < templates[j].Name

		}

		if sortOrder == "desc" {

			return !less

		}

		return less

	})

	return templates

}

func (c *Catalog) paginateResults(templates []*Template, limit, offset int) []*Template {

	if limit <= 0 {

		return templates

	}

	start := offset

	if start >= len(templates) {

		return []*Template{}

	}

	end := start + limit

	if end > len(templates) {

		end = len(templates)

	}

	return templates[start:end]

}

// Helper and utility methods.

func (c *Catalog) buildCacheKey(criteria *SearchCriteria) string {

	data, _ := json.Marshal(criteria)

	hash := sha256.Sum256(data)

	return fmt.Sprintf("search_%x", hash[:8])

}

func (c *Catalog) cacheSearchResults(key string, results []*Template) {

	c.repoCache.Store(key, map[string]interface{}{

		"templates": results,

		"expiry": time.Now().Add(c.config.CacheTTL),
	})

}

func (c *Catalog) calculateChecksum(template *Template) string {

	data, _ := json.Marshal(template.Files)

	hash := sha256.Sum256(data)

	return fmt.Sprintf("%x", hash)

}

func (c *Catalog) updateIndexes(template *Template) {

	c.searchIndex.UpdateTemplate(template)

	c.dependencyGraph.UpdateTemplate(template)

	// Update type index.

	c.templatesByType[template.Type] = append(c.templatesByType[template.Type], template)

	// Update category index.

	c.templatesByCategory[template.Category] = append(c.templatesByCategory[template.Category], template)

	// Update component index.

	for _, component := range template.TargetComponents {

		c.templatesByComponent[string(component)] = append(c.templatesByComponent[string(component)], template)

	}

}

// Background worker methods.

func (c *Catalog) syncWorker() {

	defer c.wg.Done()

	ticker := time.NewTicker(30 * time.Minute) // Sync every 30 minutes

	defer ticker.Stop()

	for {

		select {

		case <-c.ctx.Done():

			return

		case <-ticker.C:

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)

			if err := c.SyncRepositories(ctx); err != nil {

				c.logger.Error("Background repository sync failed", zap.Error(err))

			}

			cancel()

		}

	}

}

func (c *Catalog) indexMaintenanceWorker() {

	defer c.wg.Done()

	ticker := time.NewTicker(10 * time.Minute)

	defer ticker.Stop()

	for {

		select {

		case <-c.ctx.Done():

			return

		case <-ticker.C:

			c.maintainIndexes()

		}

	}

}

func (c *Catalog) cacheCleanupWorker() {

	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CacheTTL / 2)

	defer ticker.Stop()

	for {

		select {

		case <-c.ctx.Done():

			return

		case <-ticker.C:

			c.cleanupCache()

		}

	}

}

// GetCacheHits returns the number of cache hits.

func (c *Catalog) GetCacheHits() int64 {

	return c.cacheHits

}

// GetCacheMisses returns the number of cache misses.

func (c *Catalog) GetCacheMisses() int64 {

	return c.cacheMisses

}

// HealthCheck performs a health check on the catalog.

func (c *Catalog) HealthCheck(ctx context.Context) bool {

	// Check if we have templates loaded.

	templateCount := 0

	c.templates.Range(func(_, _ interface{}) bool {

		templateCount++

		return templateCount < 100 // Just count a few to avoid long iteration

	})

	if templateCount == 0 {

		c.logger.Warn("No templates loaded in catalog")

		return false

	}

	// Check if last sync was successful and recent.

	if time.Since(c.lastSync) > 2*time.Hour {

		c.logger.Warn("Template synchronization is outdated",

			zap.Time("last_sync", c.lastSync))

		return false

	}

	return true

}

// Placeholder implementations for complex methods that would need full implementation.

func NewSearchIndex() *SearchIndex {

	return &SearchIndex{

		nameIndex: make(map[string][]*Template),

		tagIndex: make(map[string][]*Template),

		componentIndex: make(map[string][]*Template),

		typeIndex: make(map[v1.IntentType][]*Template),

		textIndex: make(map[string][]*Template),
	}

}

// NewDependencyGraph performs newdependencygraph operation.

func NewDependencyGraph() *DependencyGraph {

	return &DependencyGraph{

		dependencies: make(map[string][]string),

		dependents: make(map[string][]string),

		conflicts: make(map[string][]string),

		compatibility: make(map[string][]string),
	}

}

// removeDuplicateTemplates removes duplicate templates from a slice.

func (c *Catalog) removeDuplicateTemplates(templates []*Template) []*Template {

	seen := make(map[string]bool)

	var result []*Template

	for _, template := range templates {

		if template == nil {

			continue

		}

		if !seen[template.ID] {

			seen[template.ID] = true

			result = append(result, template)

		}

	}

	return result

}

// validateORANCompliance validates a template against O-RAN compliance rules.

func (c *Catalog) validateORANCompliance(template *Template) error {

	if !template.ORANCompliant {

		return nil // Skip validation if not claiming O-RAN compliance

	}

	// Check for required O-RAN interfaces.

	requiredInterfaces := []string{"A1", "O1", "O2", "E2"}

	foundInterfaces := make(map[string]bool)

	// Scan template files for interface implementations.

	for filename, content := range template.Files {

		contentLower := strings.ToLower(content)

		for _, iface := range requiredInterfaces {

			patterns := c.getInterfacePatterns(iface)

			for _, pattern := range patterns {

				if strings.Contains(contentLower, pattern) {

					foundInterfaces[iface] = true

					break

				}

			}

		}

		_ = filename // Avoid unused variable warning

	}

	// Check if at least one O-RAN interface is implemented.

	if len(foundInterfaces) == 0 {

		return fmt.Errorf("template claims O-RAN compliance but no O-RAN interfaces found")

	}

	return nil

}

// validateDependencies validates template dependencies.

func (c *Catalog) validateDependencies(template *Template) error {

	for _, dep := range template.Dependencies {

		if dep.Required {

			// Check if dependency template exists.

			if _, err := c.GetTemplate(context.Background(), dep.TemplateID); err != nil {

				return fmt.Errorf("required dependency %s not found: %w", dep.TemplateID, err)

			}

		}

	}

	return nil

}

// validateTemplateSyntax validates template file syntax.

func (c *Catalog) validateTemplateSyntax(template *Template) error {

	for filename, content := range template.Files {

		if strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") {

			var obj interface{}

			if err := yaml.Unmarshal([]byte(content), &obj); err != nil {

				return fmt.Errorf("invalid YAML syntax in %s: %w", filename, err)

			}

		}

		// Add more syntax validation for other file types as needed.

	}

	return nil

}

// getInterfacePatterns returns search patterns for O-RAN interfaces.

func (c *Catalog) getInterfacePatterns(iface string) []string {

	patterns := map[string][]string{

		"A1": {"a1", "policy", "near-rt-ric", "non-rt-ric"},

		"O1": {"o1", "netconf", "yang", "fcaps", "fault", "configuration", "accounting", "performance", "security"},

		"O2": {"o2", "infrastructure", "cloud", "deployment"},

		"E2": {"e2", "subscription", "indication", "control", "report"},
	}

	if p, ok := patterns[iface]; ok {

		return p

	}

	return []string{}

}

// hasMatchingComponent checks if template components match search criteria.

func (c *Catalog) hasMatchingComponent(templateComponents, searchComponents []v1.ORANComponent) bool {

	for _, searchComp := range searchComponents {

		for _, templateComp := range templateComponents {

			if templateComp == searchComp {

				return true

			}

		}

	}

	return false

}

// hasMatchingIntentType checks if template intent types match search criteria.

func (c *Catalog) hasMatchingIntentType(templateTypes, searchTypes []v1.IntentType) bool {

	for _, searchType := range searchTypes {

		for _, templateType := range templateTypes {

			if templateType == searchType {

				return true

			}

		}

	}

	return false

}

// removeFromIndexes removes template from all indexes.

func (c *Catalog) removeFromIndexes(template *Template) {

	c.searchIndex.RemoveTemplate(template)

	c.dependencyGraph.RemoveTemplate(template)

	// Remove from type index.

	if templates, ok := c.templatesByType[template.Type]; ok {

		for i, t := range templates {

			if t.ID == template.ID {

				c.templatesByType[template.Type] = append(templates[:i], templates[i+1:]...)

				break

			}

		}

	}

	// Remove from category index.

	if templates, ok := c.templatesByCategory[template.Category]; ok {

		for i, t := range templates {

			if t.ID == template.ID {

				c.templatesByCategory[template.Category] = append(templates[:i], templates[i+1:]...)

				break

			}

		}

	}

	// Remove from component index.

	for _, component := range template.TargetComponents {

		if templates, ok := c.templatesByComponent[string(component)]; ok {

			for i, t := range templates {

				if t.ID == template.ID {

					c.templatesByComponent[string(component)] = append(templates[:i], templates[i+1:]...)

					break

				}

			}

		}

	}

}

// maintainIndexes performs index maintenance.

func (c *Catalog) maintainIndexes() {

	c.logger.Debug("Performing index maintenance")

	// Cleanup empty index entries, rebuild if needed, etc.

	// This is a placeholder for more sophisticated index maintenance.

}

// cleanupCache removes expired cache entries.

func (c *Catalog) cleanupCache() {

	now := time.Now()

	c.repoCache.Range(func(key, value interface{}) bool {

		if cacheEntry, ok := value.(map[string]interface{}); ok {

			if expiry, ok := cacheEntry["expiry"].(time.Time); ok {

				if now.After(expiry) {

					c.repoCache.Delete(key)

				}

			}

		}

		return true

	})

}

// cloneRepository clones or updates a repository.

func (c *Catalog) cloneRepository(ctx context.Context, repo *TemplateRepository) (string, error) {

	// This is a stub implementation.

	// In a real implementation, this would use git-go to clone/update repositories.

	c.logger.Debug("Cloning repository (stub)", zap.String("repo", repo.Name))

	return "/tmp/repo-" + repo.Name, nil

}

// discoverTemplates discovers templates in a repository path.

func (c *Catalog) discoverTemplates(repoPath string, repo *TemplateRepository) ([]*Template, error) {

	// This is a stub implementation.

	// In a real implementation, this would scan the repository for template files.

	c.logger.Debug("Discovering templates (stub)", zap.String("path", repoPath))

	return []*Template{}, nil

}

// SearchIndex methods.

// SearchByText searches templates by text.

func (si *SearchIndex) SearchByText(query string) []*Template {

	si.mutex.RLock()

	defer si.mutex.RUnlock()

	// This is a simple implementation - would be more sophisticated in practice.

	queryLower := strings.ToLower(query)

	if templates, ok := si.textIndex[queryLower]; ok {

		return templates

	}

	return []*Template{}

}

// UpdateTemplate updates template in search index.

func (si *SearchIndex) UpdateTemplate(template *Template) {

	si.mutex.Lock()

	defer si.mutex.Unlock()

	// Update name index.

	nameLower := strings.ToLower(template.Name)

	si.nameIndex[nameLower] = append(si.nameIndex[nameLower], template)

	// Update tag index.

	for _, tag := range template.Tags {

		tagLower := strings.ToLower(tag)

		si.tagIndex[tagLower] = append(si.tagIndex[tagLower], template)

	}

	// Update component index.

	for _, component := range template.TargetComponents {

		si.componentIndex[string(component)] = append(si.componentIndex[string(component)], template)

	}

	// Update type index.

	for _, intentType := range template.IntentTypes {

		si.typeIndex[intentType] = append(si.typeIndex[intentType], template)

	}

}

// RemoveTemplate removes template from search index.

func (si *SearchIndex) RemoveTemplate(template *Template) {

	si.mutex.Lock()

	defer si.mutex.Unlock()

	// Remove from name index.

	nameLower := strings.ToLower(template.Name)

	if templates, ok := si.nameIndex[nameLower]; ok {

		si.nameIndex[nameLower] = si.removeTemplateFromSlice(templates, template.ID)

	}

	// Remove from tag index.

	for _, tag := range template.Tags {

		tagLower := strings.ToLower(tag)

		if templates, ok := si.tagIndex[tagLower]; ok {

			si.tagIndex[tagLower] = si.removeTemplateFromSlice(templates, template.ID)

		}

	}

	// Remove from component index.

	for _, component := range template.TargetComponents {

		if templates, ok := si.componentIndex[string(component)]; ok {

			si.componentIndex[string(component)] = si.removeTemplateFromSlice(templates, template.ID)

		}

	}

	// Remove from type index.

	for _, intentType := range template.IntentTypes {

		if templates, ok := si.typeIndex[intentType]; ok {

			si.typeIndex[intentType] = si.removeTemplateFromSlice(templates, template.ID)

		}

	}

}

func (si *SearchIndex) removeTemplateFromSlice(templates []*Template, templateID string) []*Template {

	for i, template := range templates {

		if template.ID == templateID {

			return append(templates[:i], templates[i+1:]...)

		}

	}

	return templates

}

// DependencyGraph methods.

// GetDependents returns templates that depend on the given template.

func (dg *DependencyGraph) GetDependents(templateID string) []string {

	dg.mutex.RLock()

	defer dg.mutex.RUnlock()

	if dependents, ok := dg.dependents[templateID]; ok {

		return dependents

	}

	return []string{}

}

// UpdateTemplate updates template dependencies in the graph.

func (dg *DependencyGraph) UpdateTemplate(template *Template) {

	dg.mutex.Lock()

	defer dg.mutex.Unlock()

	// Clear existing dependencies for this template.

	delete(dg.dependencies, template.ID)

	// Add new dependencies.

	var deps []string

	for _, dep := range template.Dependencies {

		deps = append(deps, dep.TemplateID)

		// Update dependents mapping.

		if dg.dependents[dep.TemplateID] == nil {

			dg.dependents[dep.TemplateID] = []string{}

		}

		dg.dependents[dep.TemplateID] = append(dg.dependents[dep.TemplateID], template.ID)

	}

	dg.dependencies[template.ID] = deps

}

// RemoveTemplate removes template from dependency graph.

func (dg *DependencyGraph) RemoveTemplate(template *Template) {

	dg.mutex.Lock()

	defer dg.mutex.Unlock()

	// Remove dependencies.

	delete(dg.dependencies, template.ID)

	// Remove from dependents.

	delete(dg.dependents, template.ID)

	// Remove from other templates' dependents lists.

	for templateID, dependents := range dg.dependents {

		for i, dependent := range dependents {

			if dependent == template.ID {

				dg.dependents[templateID] = append(dependents[:i], dependents[i+1:]...)

				break

			}

		}

	}

}
