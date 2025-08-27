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

	"github.com/ghodss/yaml"
	"go.uber.org/zap"

	v1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// Template represents a blueprint template
type Template struct {
	ID          string           `json:"id" yaml:"id"`
	Name        string           `json:"name" yaml:"name"`
	Description string           `json:"description" yaml:"description"`
	Version     string           `json:"version" yaml:"version"`
	Type        TemplateType     `json:"type" yaml:"type"`
	Category    TemplateCategory `json:"category" yaml:"category"`
	Tags        []string         `json:"tags" yaml:"tags"`
	Author      string           `json:"author" yaml:"author"`
	License     string           `json:"license" yaml:"license"`

	// Component targeting
	TargetComponents []v1.TargetComponent `json:"targetComponents" yaml:"targetComponents"`
	IntentTypes      []v1.IntentType      `json:"intentTypes" yaml:"intentTypes"`

	// Dependencies
	Dependencies  []TemplateDependency `json:"dependencies" yaml:"dependencies"`
	Prerequisites []string             `json:"prerequisites" yaml:"prerequisites"`

	// Template content and metadata
	Files      map[string]string   `json:"files" yaml:"files"`
	Parameters []TemplateParameter `json:"parameters" yaml:"parameters"`
	Outputs    []TemplateOutput    `json:"outputs" yaml:"outputs"`

	// Validation and compliance
	ORANCompliant    bool     `json:"oranCompliant" yaml:"oranCompliant"`
	Validated        bool     `json:"validated" yaml:"validated"`
	ValidationErrors []string `json:"validationErrors,omitempty" yaml:"validationErrors,omitempty"`

	// Metadata
	CreatedAt  time.Time `json:"createdAt" yaml:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt" yaml:"updatedAt"`
	UsageCount int64     `json:"usageCount" yaml:"usageCount"`
	Rating     float64   `json:"rating" yaml:"rating"`
	Checksum   string    `json:"checksum" yaml:"checksum"`

	// Repository information
	Repository string `json:"repository" yaml:"repository"`
	Path       string `json:"path" yaml:"path"`
	Branch     string `json:"branch" yaml:"branch"`
	CommitHash string `json:"commitHash" yaml:"commitHash"`
}

// TemplateType defines the type of blueprint template
type TemplateType string

const (
	TemplateTypeDeployment    TemplateType = "deployment"
	TemplateTypeService       TemplateType = "service"
	TemplateTypeConfiguration TemplateType = "configuration"
	TemplateTypeNetworking    TemplateType = "networking"
	TemplateTypeMonitoring    TemplateType = "monitoring"
	TemplateTypeSecurity      TemplateType = "security"
	TemplateTypeComplete      TemplateType = "complete"
)

// TemplateCategory defines the category of blueprint template
type TemplateCategory string

const (
	TemplateCategoryCore    TemplateCategory = "5g-core"
	TemplateCategoryRAN     TemplateCategory = "ran"
	TemplateCategoryORAN    TemplateCategory = "oran"
	TemplateCategoryEdge    TemplateCategory = "edge"
	TemplateCategorySlicing TemplateCategory = "slicing"
	TemplateCategoryCloud   TemplateCategory = "cloud"
	TemplateCategoryNetwork TemplateCategory = "network"
	TemplateCategoryGeneric TemplateCategory = "generic"
)

// TemplateDependency represents a template dependency
type TemplateDependency struct {
	TemplateID    string   `json:"templateId" yaml:"templateId"`
	Version       string   `json:"version" yaml:"version"`
	Required      bool     `json:"required" yaml:"required"`
	Reason        string   `json:"reason" yaml:"reason"`
	Compatibility []string `json:"compatibility" yaml:"compatibility"`
}

// TemplateParameter represents a configurable parameter in a template
type TemplateParameter struct {
	Name        string      `json:"name" yaml:"name"`
	Type        string      `json:"type" yaml:"type"`
	Description string      `json:"description" yaml:"description"`
	Default     interface{} `json:"default,omitempty" yaml:"default,omitempty"`
	Required    bool        `json:"required" yaml:"required"`
	Constraints []string    `json:"constraints,omitempty" yaml:"constraints,omitempty"`
	Options     []string    `json:"options,omitempty" yaml:"options,omitempty"`
	Pattern     string      `json:"pattern,omitempty" yaml:"pattern,omitempty"`
}

// TemplateOutput represents an output from a template
type TemplateOutput struct {
	Name        string `json:"name" yaml:"name"`
	Type        string `json:"type" yaml:"type"`
	Description string `json:"description" yaml:"description"`
	Export      bool   `json:"export" yaml:"export"`
}

// SearchCriteria defines criteria for template search and filtering
type SearchCriteria struct {
	// Basic filters
	Query    string           `json:"query,omitempty"`
	Type     TemplateType     `json:"type,omitempty"`
	Category TemplateCategory `json:"category,omitempty"`
	Tags     []string         `json:"tags,omitempty"`

	// Component filters
	TargetComponents []v1.TargetComponent `json:"targetComponents,omitempty"`
	IntentTypes      []v1.IntentType      `json:"intentTypes,omitempty"`

	// Quality filters
	ORANCompliant *bool    `json:"oranCompliant,omitempty"`
	MinRating     *float64 `json:"minRating,omitempty"`
	Validated     *bool    `json:"validated,omitempty"`

	// Version and dependency filters
	Version    string `json:"version,omitempty"`
	MinVersion string `json:"minVersion,omitempty"`
	MaxVersion string `json:"maxVersion,omitempty"`

	// Sorting and pagination
	SortBy    string `json:"sortBy,omitempty"`
	SortOrder string `json:"sortOrder,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

// Catalog manages blueprint template repository and provides discovery capabilities
type Catalog struct {
	config *BlueprintConfig
	logger *zap.Logger

	// Template storage
	templates            sync.Map // map[string]*Template
	templatesByType      map[TemplateType][]*Template
	templatesByCategory  map[TemplateCategory][]*Template
	templatesByComponent map[string][]*Template  // Use string instead of v1.TargetComponent struct

	// Repository management
	repositories []TemplateRepository
	repoCache    sync.Map

	// Indexing and search
	searchIndex     *SearchIndex
	dependencyGraph *DependencyGraph

	// Cache and performance
	cacheHits   int64
	cacheMisses int64
	lastSync    time.Time
	syncMutex   sync.RWMutex

	// Background operations
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// TemplateRepository represents a template repository configuration
type TemplateRepository struct {
	Name        string `json:"name" yaml:"name"`
	URL         string `json:"url" yaml:"url"`
	Branch      string `json:"branch" yaml:"branch"`
	Path        string `json:"path" yaml:"path"`
	Credentials string `json:"credentials,omitempty" yaml:"credentials,omitempty"`
	Priority    int    `json:"priority" yaml:"priority"`
	Enabled     bool   `json:"enabled" yaml:"enabled"`
}

// SearchIndex provides fast template search capabilities
type SearchIndex struct {
	nameIndex      map[string][]*Template
	tagIndex       map[string][]*Template
	componentIndex map[string][]*Template  // Use string instead of v1.TargetComponent struct
	typeIndex      map[v1.IntentType][]*Template
	textIndex      map[string][]*Template
	mutex          sync.RWMutex
}

// DependencyGraph manages template dependencies and compatibility
type DependencyGraph struct {
	dependencies  map[string][]string
	dependents    map[string][]string
	conflicts     map[string][]string
	compatibility map[string][]string
	mutex         sync.RWMutex
}

// NewCatalog creates a new blueprint catalog
func NewCatalog(config *BlueprintConfig, logger *zap.Logger) (*Catalog, error) {
	if config == nil {
		config = DefaultBlueprintConfig()
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	ctx, cancel := context.WithCancel(context.Background())

	catalog := &Catalog{
		config:               config,
		logger:               logger,
		templatesByType:      make(map[TemplateType][]*Template),
		templatesByCategory:  make(map[TemplateCategory][]*Template),
		templatesByComponent: make(map[string][]*Template),
		repositories:         []TemplateRepository{},
		searchIndex:          NewSearchIndex(),
		dependencyGraph:      NewDependencyGraph(),
		ctx:                  ctx,
		cancel:               cancel,
	}

	// Initialize default repositories
	if err := catalog.initializeDefaultRepositories(); err != nil {
		cancel()
		return nil, fmt.Errorf("failed to initialize repositories: %w", err)
	}

	// Start background operations
	catalog.startBackgroundOperations()

	logger.Info("Blueprint catalog initialized",
		zap.Int("repositories", len(catalog.repositories)))

	return catalog, nil
}

// initializeDefaultRepositories sets up default template repositories
func (c *Catalog) initializeDefaultRepositories() error {
	defaultRepos := []TemplateRepository{
		{
			Name:     "nephio-official",
			URL:      "https://github.com/nephio-project/catalog.git",
			Branch:   "main",
			Path:     "templates",
			Priority: 10,
			Enabled:  true,
		},
		{
			Name:     "oran-alliance",
			URL:      "https://github.com/o-ran-sc/scp-oam-modeling.git",
			Branch:   "master",
			Path:     "yang-models",
			Priority: 8,
			Enabled:  true,
		},
		{
			Name:     "free5gc-templates",
			URL:      "https://github.com/free5gc/free5gc-k8s.git",
			Branch:   "main",
			Path:     "templates",
			Priority: 6,
			Enabled:  true,
		},
	}

	c.repositories = append(c.repositories, defaultRepos...)
	return nil
}

// startBackgroundOperations starts background worker goroutines
func (c *Catalog) startBackgroundOperations() {
	// Template synchronization worker
	c.wg.Add(1)
	go c.syncWorker()

	// Index maintenance worker
	c.wg.Add(1)
	go c.indexMaintenanceWorker()

	// Cache cleanup worker
	c.wg.Add(1)
	go c.cacheCleanupWorker()
}

// FindTemplates searches for templates matching the given criteria
func (c *Catalog) FindTemplates(ctx context.Context, criteria *SearchCriteria) ([]*Template, error) {
	startTime := time.Now()

	c.logger.Debug("Searching templates",
		zap.Any("criteria", criteria))

	// Try cache first
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

	// Perform search
	results := c.performSearch(criteria)

	// Apply additional filters
	filtered := c.applyFilters(results, criteria)

	// Sort results
	sorted := c.sortTemplates(filtered, criteria.SortBy, criteria.SortOrder)

	// Apply pagination
	paginated := c.paginateResults(sorted, criteria.Limit, criteria.Offset)

	// Cache results
	c.cacheSearchResults(cacheKey, paginated)

	duration := time.Since(startTime)
	c.logger.Debug("Template search completed",
		zap.Duration("duration", duration),
		zap.Int("total_results", len(results)),
		zap.Int("filtered_results", len(filtered)),
		zap.Int("paginated_results", len(paginated)))

	return paginated, nil
}

// GetTemplate retrieves a specific template by ID
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

// GetTemplatesByComponent returns templates for specific components
func (c *Catalog) GetTemplatesByComponent(ctx context.Context, components []string) ([]*Template, error) {
	var results []*Template

	for _, component := range components {
		if templates, ok := c.templatesByComponent[component]; ok {
			results = append(results, templates...)
		}
	}

	// Remove duplicates
	uniqueResults := c.removeDuplicateTemplates(results)

	return uniqueResults, nil
}

// GetRecommendedTemplates returns recommended templates for a NetworkIntent
func (c *Catalog) GetRecommendedTemplates(ctx context.Context, intent *v1.NetworkIntent) ([]*Template, error) {
	// Convert NetworkTargetComponent to TargetComponent
	targetComponents := make([]v1.TargetComponent, len(intent.Spec.TargetComponents))
	for i, component := range intent.Spec.TargetComponents {
		targetComponents[i] = v1.TargetComponent{
			Name:      string(component),
			Type:      "deployment", // Default component type
			Namespace: intent.Namespace,
		}
	}

	criteria := &SearchCriteria{
		TargetComponents: targetComponents,
		IntentTypes:      []v1.IntentType{intent.Spec.IntentType},
		ORANCompliant:    &[]bool{true}[0],
		Validated:        &[]bool{true}[0],
		MinRating:        &[]float64{3.0}[0],
		SortBy:           "rating",
		SortOrder:        "desc",
		Limit:            10,
	}

	return c.FindTemplates(ctx, criteria)
}

// ValidateTemplate validates a template against O-RAN compliance and other criteria
func (c *Catalog) ValidateTemplate(ctx context.Context, template *Template) error {
	var errors []string

	// Validate required fields
	if template.ID == "" {
		errors = append(errors, "template ID is required")
	}
	if template.Name == "" {
		errors = append(errors, "template name is required")
	}
	if template.Version == "" {
		errors = append(errors, "template version is required")
	}

	// Validate files exist
	if len(template.Files) == 0 {
		errors = append(errors, "template must contain at least one file")
	}

	// Validate O-RAN compliance if claimed
	if template.ORANCompliant {
		if err := c.validateORANCompliance(template); err != nil {
			errors = append(errors, fmt.Sprintf("O-RAN compliance validation failed: %v", err))
		}
	}

	// Validate dependencies
	if err := c.validateDependencies(template); err != nil {
		errors = append(errors, fmt.Sprintf("dependency validation failed: %v", err))
	}

	// Validate template syntax
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

// RegisterTemplate registers a new template in the catalog
func (c *Catalog) RegisterTemplate(ctx context.Context, template *Template) error {
	// Validate template
	if err := c.ValidateTemplate(ctx, template); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Set metadata
	now := time.Now()
	if template.CreatedAt.IsZero() {
		template.CreatedAt = now
	}
	template.UpdatedAt = now
	template.Checksum = c.calculateChecksum(template)

	// Store template
	c.templates.Store(template.ID, template)

	// Update indexes
	c.updateIndexes(template)

	c.logger.Info("Template registered",
		zap.String("template_id", template.ID),
		zap.String("template_name", template.Name),
		zap.String("version", template.Version))

	return nil
}

// UpdateTemplate updates an existing template
func (c *Catalog) UpdateTemplate(ctx context.Context, template *Template) error {
	existing, err := c.GetTemplate(ctx, template.ID)
	if err != nil {
		return fmt.Errorf("template not found for update: %w", err)
	}

	// Preserve creation time and usage stats
	template.CreatedAt = existing.CreatedAt
	template.UsageCount = existing.UsageCount

	// Update modification time and checksum
	template.UpdatedAt = time.Now()
	template.Checksum = c.calculateChecksum(template)

	// Validate updated template
	if err := c.ValidateTemplate(ctx, template); err != nil {
		return fmt.Errorf("template validation failed: %w", err)
	}

	// Update storage and indexes
	c.templates.Store(template.ID, template)
	c.updateIndexes(template)

	c.logger.Info("Template updated",
		zap.String("template_id", template.ID),
		zap.String("template_name", template.Name),
		zap.String("version", template.Version))

	return nil
}

// RemoveTemplate removes a template from the catalog
func (c *Catalog) RemoveTemplate(ctx context.Context, templateID string) error {
	template, err := c.GetTemplate(ctx, templateID)
	if err != nil {
		return err
	}

	// Check for dependencies
	if dependents := c.dependencyGraph.GetDependents(templateID); len(dependents) > 0 {
		return fmt.Errorf("cannot remove template %s: has dependents %v", templateID, dependents)
	}

	// Remove from storage and indexes
	c.templates.Delete(templateID)
	c.removeFromIndexes(template)

	c.logger.Info("Template removed",
		zap.String("template_id", templateID),
		zap.String("template_name", template.Name))

	return nil
}

// SyncRepositories synchronizes templates from all configured repositories
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

// syncRepository synchronizes templates from a single repository
func (c *Catalog) syncRepository(ctx context.Context, repo *TemplateRepository) error {
	c.logger.Debug("Syncing repository",
		zap.String("repository", repo.Name),
		zap.String("url", repo.URL))

	// Clone or pull repository
	repoPath, err := c.cloneRepository(ctx, repo.URL)
	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	// Discover templates in repository
	templates, err := c.discoverTemplates(ctx, repoPath)
	if err != nil {
		return fmt.Errorf("failed to discover templates: %w", err)
	}

	// Register discovered templates
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

// Implementation of search, indexing, and caching methods

func (c *Catalog) performSearch(criteria *SearchCriteria) []*Template {
	var results []*Template

	if criteria.Query != "" {
		searchResults, err := c.searchIndex.SearchByText(context.Background(), criteria.Query)
		if err == nil {
			results = searchResults
		}
	} else {
		// Return all templates if no specific query
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
	// Type filter
	if criteria.Type != "" && template.Type != criteria.Type {
		return false
	}

	// Category filter
	if criteria.Category != "" && template.Category != criteria.Category {
		return false
	}

	// Component filter
	if len(criteria.TargetComponents) > 0 {
		if !c.hasMatchingComponent(template.TargetComponents, criteria.TargetComponents) {
			return false
		}
	}

	// Intent type filter
	if len(criteria.IntentTypes) > 0 {
		if !c.hasMatchingIntentType(template.IntentTypes, criteria.IntentTypes) {
			return false
		}
	}

	// O-RAN compliance filter
	if criteria.ORANCompliant != nil && template.ORANCompliant != *criteria.ORANCompliant {
		return false
	}

	// Rating filter
	if criteria.MinRating != nil && template.Rating < *criteria.MinRating {
		return false
	}

	// Validation filter
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

// removeDuplicateTemplates removes duplicate templates from a slice based on ID
func (c *Catalog) removeDuplicateTemplates(templates []*Template) []*Template {
	if len(templates) == 0 {
		return templates
	}

	seen := make(map[string]bool)
	result := make([]*Template, 0, len(templates))

	for _, template := range templates {
		if template != nil && !seen[template.ID] {
			seen[template.ID] = true
			result = append(result, template)
		}
	}

	return result
}

// Helper and utility methods

func (c *Catalog) buildCacheKey(criteria *SearchCriteria) string {
	data, _ := json.Marshal(criteria)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("search_%x", hash[:8])
}

func (c *Catalog) cacheSearchResults(key string, results []*Template) {
	c.repoCache.Store(key, map[string]interface{}{
		"templates": results,
		"expiry":    time.Now().Add(c.config.CacheTTL),
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

	// Update type index
	c.templatesByType[template.Type] = append(c.templatesByType[template.Type], template)

	// Update category index
	c.templatesByCategory[template.Category] = append(c.templatesByCategory[template.Category], template)

	// Update component index
	for _, component := range template.TargetComponents {
		componentKey := component.Name + ":" + component.Type
		c.templatesByComponent[componentKey] = append(c.templatesByComponent[componentKey], template)
	}
}

// Background worker methods
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

// GetCacheHits returns the number of cache hits
func (c *Catalog) GetCacheHits() int64 {
	return c.cacheHits
}

// GetCacheMisses returns the number of cache misses
func (c *Catalog) GetCacheMisses() int64 {
	return c.cacheMisses
}

// HealthCheck performs a health check on the catalog
func (c *Catalog) HealthCheck(ctx context.Context) bool {
	// Check if we have templates loaded
	templateCount := 0
	c.templates.Range(func(_, _ interface{}) bool {
		templateCount++
		return templateCount < 100 // Just count a few to avoid long iteration
	})

	if templateCount == 0 {
		c.logger.Warn("No templates loaded in catalog")
		return false
	}

	// Check if last sync was successful and recent
	if time.Since(c.lastSync) > 2*time.Hour {
		c.logger.Warn("Template synchronization is outdated",
			zap.Time("last_sync", c.lastSync))
		return false
	}

	return true
}

// Placeholder implementations for complex methods that would need full implementation
func NewSearchIndex() *SearchIndex {
	return &SearchIndex{
		nameIndex:      make(map[string][]*Template),
		tagIndex:       make(map[string][]*Template),
		componentIndex: make(map[string][]*Template),
		typeIndex:      make(map[v1.IntentType][]*Template),
		textIndex:      make(map[string][]*Template),
	}
}

func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		dependencies:  make(map[string][]string),
		dependents:    make(map[string][]string),
		conflicts:     make(map[string][]string),
		compatibility: make(map[string][]string),
	}
}

// validateORANCompliance validates O-RAN compliance for a template
func (c *Catalog) validateORANCompliance(template *Template) error {
	if !template.ORANCompliant {
		return nil
	}

	// Check for O-RAN interface definitions
	oranInterfaces := []string{"A1", "O1", "O2", "E2"}
	foundInterfaces := make(map[string]bool)

	for filename, content := range template.Files {
		contentLower := strings.ToLower(content)
		for _, iface := range oranInterfaces {
			patterns := map[string][]string{
				"A1": {"a1", "policy", "near-rt-ric"},
				"O1": {"o1", "netconf", "yang", "fcaps"},
				"O2": {"o2", "infrastructure", "cloud"},
				"E2": {"e2", "subscription", "indication"},
			}

			if interfacePatterns, ok := patterns[iface]; ok {
				for _, pattern := range interfacePatterns {
					if strings.Contains(contentLower, pattern) {
						foundInterfaces[iface] = true
						c.logger.Debug("Found O-RAN interface pattern",
							zap.String("interface", iface),
							zap.String("pattern", pattern),
							zap.String("file", filename))
						break
					}
				}
			}
		}
	}

	if len(foundInterfaces) == 0 {
		return fmt.Errorf("template claims O-RAN compliance but no O-RAN interface patterns found")
	}

	return nil
}

// validateDependencies validates template dependencies
func (c *Catalog) validateDependencies(template *Template) error {
	for _, dep := range template.Dependencies {
		// Check if dependency exists in catalog
		found := false
		c.templates.Range(func(key, value interface{}) bool {
			if existingTemplate, ok := value.(*Template); ok {
				if existingTemplate.ID == dep.TemplateID && 
				   existingTemplate.Version == dep.Version {
					found = true
					return false // Stop iteration
				}
			}
			return true // Continue iteration
		})

		if !found && dep.Required {
			return fmt.Errorf("required dependency not found: %s:%s", dep.TemplateID, dep.Version)
		}
	}

	// Check for circular dependencies
	if err := c.checkCircularDependencies(template); err != nil {
		return fmt.Errorf("circular dependency detected: %w", err)
	}

	return nil
}

// validateTemplateSyntax validates template file syntax
func (c *Catalog) validateTemplateSyntax(template *Template) error {
	for filename, content := range template.Files {
		// Skip non-YAML files
		if !isYAMLFile(filename, content) {
			continue
		}

		// Try to parse YAML
		var obj interface{}
		if err := yaml.Unmarshal([]byte(content), &obj); err != nil {
			return fmt.Errorf("invalid YAML syntax in file %s: %w", filename, err)
		}

		// For Kubernetes manifests, validate basic structure
		if isKubernetesManifest(content) {
			if err := c.validateKubernetesManifest(filename, content); err != nil {
				return fmt.Errorf("invalid Kubernetes manifest %s: %w", filename, err)
			}
		}
	}

	return nil
}

// checkCircularDependencies checks for circular dependencies
func (c *Catalog) checkCircularDependencies(template *Template) error {
	visited := make(map[string]bool)
	stack := make(map[string]bool)

	var dfs func(templateName string) error
	dfs = func(templateName string) error {
		if stack[templateName] {
			return fmt.Errorf("circular dependency involving %s", templateName)
		}
		if visited[templateName] {
			return nil
		}

		visited[templateName] = true
		stack[templateName] = true

		// Find template by name
		var currentTemplate *Template
		c.templates.Range(func(key, value interface{}) bool {
			if template, ok := value.(*Template); ok && template.Name == templateName {
				currentTemplate = template
				return false // Stop iteration
			}
			return true // Continue iteration
		})

		if currentTemplate != nil {
			for _, dep := range currentTemplate.Dependencies {
				if err := dfs(dep.TemplateID); err != nil {
					return err
				}
			}
		}

		stack[templateName] = false
		return nil
	}

	return dfs(template.Name)
}

// validateKubernetesManifest validates basic Kubernetes manifest structure
func (c *Catalog) validateKubernetesManifest(filename, content string) error {
	var manifest map[string]interface{}
	if err := yaml.Unmarshal([]byte(content), &manifest); err != nil {
		return err
	}

	// Check required fields
	if _, ok := manifest["apiVersion"]; !ok {
		return fmt.Errorf("missing apiVersion field")
	}
	if _, ok := manifest["kind"]; !ok {
		return fmt.Errorf("missing kind field")
	}
	if _, ok := manifest["metadata"]; !ok {
		return fmt.Errorf("missing metadata field")
	}

	return nil
}

// Helper functions
func isYAMLFile(filename, content string) bool {
	return strings.HasSuffix(filename, ".yaml") || strings.HasSuffix(filename, ".yml") ||
		(strings.Contains(content, "apiVersion:") && strings.Contains(content, "kind:"))
}

func isKubernetesManifest(content string) bool {
	return strings.Contains(content, "apiVersion:") && strings.Contains(content, "kind:")
}

// hasMatchingComponent checks if template has matching target components
func (c *Catalog) hasMatchingComponent(templateComponents, criteriaComponents []v1.TargetComponent) bool {
	if len(criteriaComponents) == 0 {
		return true
	}

	for _, criteriaComp := range criteriaComponents {
		for _, templateComp := range templateComponents {
			if templateComp.Name == criteriaComp.Name || templateComp.Type == criteriaComp.Type {
				return true
			}
		}
	}

	return false
}

// hasMatchingIntentType checks if template has matching intent type
func (c *Catalog) hasMatchingIntentType(templateTypes, criteriaTypes []v1.IntentType) bool {
	if len(criteriaTypes) == 0 {
		return true
	}

	for _, criteriaType := range criteriaTypes {
		for _, templateType := range templateTypes {
			if templateType == criteriaType {
				return true
			}
		}
	}

	return false
}

// removeFromIndexes removes template from search indexes
func (c *Catalog) removeFromIndexes(template *Template) {
	// Remove from search index
	c.searchIndex.mutex.Lock()
	for keyword := range c.searchIndex.textIndex {
		templates := c.searchIndex.textIndex[keyword]
		for i, t := range templates {
			if t.ID == template.ID {
				c.searchIndex.textIndex[keyword] = append(templates[:i], templates[i+1:]...)
				break
			}
		}
	}
	c.searchIndex.mutex.Unlock()

	// Remove from component index  
	for componentKey := range c.templatesByComponent {
		templates := c.templatesByComponent[componentKey]
		for i, t := range templates {
			if t.ID == template.ID {
				c.templatesByComponent[componentKey] = append(templates[:i], templates[i+1:]...)
				break
			}
		}
	}

	// Remove from dependency graph
	c.dependencyGraph.mutex.Lock()
	delete(c.dependencyGraph.dependencies, template.Name)
	delete(c.dependencyGraph.dependents, template.Name)
	
	// Remove from other templates' dependencies
	for name, deps := range c.dependencyGraph.dependencies {
		for i, dep := range deps {
			if dep == template.Name {
				c.dependencyGraph.dependencies[name] = append(deps[:i], deps[i+1:]...)
				break
			}
		}
	}
	c.dependencyGraph.mutex.Unlock()
}

// cloneRepository clones a Git repository
func (c *Catalog) cloneRepository(ctx context.Context, repoURL string) (string, error) {
	// Stub implementation - would clone repo and return path
	c.logger.Info("Cloning repository", zap.String("url", repoURL))
	return "/tmp/cloned-repo", nil
}

// discoverTemplates discovers templates in a directory
func (c *Catalog) discoverTemplates(ctx context.Context, dir string) ([]*Template, error) {
	// Stub implementation - would scan directory for template files
	c.logger.Info("Discovering templates", zap.String("dir", dir))
	return []*Template{}, nil
}

// SearchIndex method implementations
func (si *SearchIndex) SearchByText(ctx context.Context, query string) ([]*Template, error) {
	si.mutex.RLock()
	defer si.mutex.RUnlock()

	var results []*Template
	queryWords := strings.Fields(strings.ToLower(query))

	for _, word := range queryWords {
		if templates, exists := si.textIndex[word]; exists {
			results = append(results, templates...)
		}
	}

	// Remove duplicates
	seen := make(map[string]bool)
	uniqueResults := make([]*Template, 0)
	for _, template := range results {
		if !seen[template.ID] {
			seen[template.ID] = true
			uniqueResults = append(uniqueResults, template)
		}
	}

	return uniqueResults, nil
}

func (si *SearchIndex) UpdateTemplate(template *Template) {
	si.mutex.Lock()
	defer si.mutex.Unlock()

	// Remove old entries
	for keyword := range si.textIndex {
		templates := si.textIndex[keyword]
		for i, t := range templates {
			if t.ID == template.ID {
				si.textIndex[keyword] = append(templates[:i], templates[i+1:]...)
				break
			}
		}
	}

	// Add new entries
	keywords := strings.Fields(strings.ToLower(template.Name + " " + template.Description))
	for _, keyword := range keywords {
		si.textIndex[keyword] = append(si.textIndex[keyword], template)
	}
}

// DependencyGraph method implementations
func (dg *DependencyGraph) GetDependents(templateName string) []string {
	dg.mutex.RLock()
	defer dg.mutex.RUnlock()

	if dependents, exists := dg.dependents[templateName]; exists {
		// Return a copy to avoid concurrent modification
		result := make([]string, len(dependents))
		copy(result, dependents)
		return result
	}

	return []string{}
}

func (dg *DependencyGraph) UpdateTemplate(template *Template) {
	dg.mutex.Lock()
	defer dg.mutex.Unlock()

	templateName := template.Name

	// Clear existing dependencies
	delete(dg.dependencies, templateName)

	// Remove from dependents of other templates
	for name, dependents := range dg.dependents {
		for i, dependent := range dependents {
			if dependent == templateName {
				dg.dependents[name] = append(dependents[:i], dependents[i+1:]...)
				break
			}
		}
	}

	// Add new dependencies
	deps := make([]string, len(template.Dependencies))
	for i, dep := range template.Dependencies {
		deps[i] = dep.TemplateID
		
		// Add to dependents map
		if dg.dependents[dep.TemplateID] == nil {
			dg.dependents[dep.TemplateID] = []string{}
		}
		dg.dependents[dep.TemplateID] = append(dg.dependents[dep.TemplateID], templateName)
	}
	dg.dependencies[templateName] = deps
}

// maintainIndexes maintains search and component indexes
func (c *Catalog) maintainIndexes() {
	c.logger.Debug("Maintaining indexes")
	
	// Rebuild search index if needed
	totalTemplates := 0
	c.templates.Range(func(key, value interface{}) bool {
		totalTemplates++
		return true
	})

	// Trigger index rebuild if template count significantly changed
	if totalTemplates > 0 {
		c.logger.Debug("Index maintenance completed", zap.Int("templates", totalTemplates))
	}
}

// cleanupCache cleans up expired cache entries
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
