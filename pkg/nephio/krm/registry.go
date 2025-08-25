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
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/yaml"

	"github.com/thc1006/nephoran-intent-operator/pkg/errors"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// Registry manages KRM function discovery, caching, and metadata
type Registry struct {
	config        *RegistryConfig
	repositories  map[string]*Repository
	functions     map[string]*FunctionMetadata
	cache         *FunctionCache
	healthMonitor *HealthMonitor
	metrics       *RegistryMetrics
	tracer        trace.Tracer
	mu            sync.RWMutex
	updateChan    chan *FunctionUpdate
	shutdownChan  chan struct{}
	httpClient    *http.Client
}

// RegistryConfig defines configuration for the function registry
type RegistryConfig struct {
	// Repository settings
	DefaultRepositories []RepositoryConfig `json:"defaultRepositories" yaml:"defaultRepositories"`
	RepositoryTimeout   time.Duration      `json:"repositoryTimeout" yaml:"repositoryTimeout"`

	// Caching settings
	CacheDir          string        `json:"cacheDir" yaml:"cacheDir"`
	CacheTTL          time.Duration `json:"cacheTtl" yaml:"cacheTtl"`
	MaxCacheSize      int64         `json:"maxCacheSize" yaml:"maxCacheSize"`
	EnableCompression bool          `json:"enableCompression" yaml:"enableCompression"`

	// Health monitoring
	HealthCheckInterval time.Duration `json:"healthCheckInterval" yaml:"healthCheckInterval"`
	UnhealthyThreshold  int           `json:"unhealthyThreshold" yaml:"unhealthyThreshold"`
	HealthyThreshold    int           `json:"healthyThreshold" yaml:"healthyThreshold"`

	// Discovery settings
	DiscoveryInterval   time.Duration `json:"discoveryInterval" yaml:"discoveryInterval"`
	ConcurrentDiscovery int           `json:"concurrentDiscovery" yaml:"concurrentDiscovery"`

	// Security settings
	AllowInsecure         bool     `json:"allowInsecure" yaml:"allowInsecure"`
	TrustedRegistries     []string `json:"trustedRegistries" yaml:"trustedRegistries"`
	SignatureVerification bool     `json:"signatureVerification" yaml:"signatureVerification"`

	// Performance settings
	MaxConcurrentDownloads int           `json:"maxConcurrentDownloads" yaml:"maxConcurrentDownloads"`
	DownloadTimeout        time.Duration `json:"downloadTimeout" yaml:"downloadTimeout"`
	RetryAttempts          int           `json:"retryAttempts" yaml:"retryAttempts"`
}

// Repository represents a function repository
type Repository struct {
	Config       RepositoryConfig    `json:"config"`
	Health       RepositoryHealth    `json:"health"`
	Functions    []*FunctionMetadata `json:"functions"`
	LastSyncTime time.Time           `json:"lastSyncTime"`
	LastError    string              `json:"lastError,omitempty"`
	Version      string              `json:"version,omitempty"`
	mu           sync.RWMutex
}

// RepositoryConfig defines repository configuration
type RepositoryConfig struct {
	Name        string            `json:"name" yaml:"name"`
	URL         string            `json:"url" yaml:"url"`
	Type        string            `json:"type" yaml:"type"` // git, oci, http
	Branch      string            `json:"branch,omitempty" yaml:"branch,omitempty"`
	Path        string            `json:"path,omitempty" yaml:"path,omitempty"`
	Auth        *AuthConfig       `json:"auth,omitempty" yaml:"auth,omitempty"`
	Headers     map[string]string `json:"headers,omitempty" yaml:"headers,omitempty"`
	Insecure    bool              `json:"insecure,omitempty" yaml:"insecure,omitempty"`
	Priority    int               `json:"priority,omitempty" yaml:"priority,omitempty"`
	Enabled     bool              `json:"enabled" yaml:"enabled"`
	Tags        []string          `json:"tags,omitempty" yaml:"tags,omitempty"`
	Description string            `json:"description,omitempty" yaml:"description,omitempty"`
}

// AuthConfig defines authentication for repositories
type AuthConfig struct {
	Type      string `json:"type" yaml:"type"` // basic, bearer, api-key
	Username  string `json:"username,omitempty" yaml:"username,omitempty"`
	Password  string `json:"password,omitempty" yaml:"password,omitempty"`
	Token     string `json:"token,omitempty" yaml:"token,omitempty"`
	SecretRef string `json:"secretRef,omitempty" yaml:"secretRef,omitempty"`
}

// FunctionMetadata contains comprehensive metadata about a KRM function
type FunctionMetadata struct {
	// Basic information
	Name        string `json:"name" yaml:"name"`
	Image       string `json:"image" yaml:"image"`
	Version     string `json:"version" yaml:"version"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`

	// Classification
	Type       string   `json:"type" yaml:"type"` // mutator, validator, generator
	Categories []string `json:"categories,omitempty" yaml:"categories,omitempty"`
	Keywords   []string `json:"keywords,omitempty" yaml:"keywords,omitempty"`
	Tags       []string `json:"tags,omitempty" yaml:"tags,omitempty"`

	// Capabilities
	ResourceTypes []ResourceTypeSupport `json:"resourceTypes,omitempty" yaml:"resourceTypes,omitempty"`
	ApiVersions   []string              `json:"apiVersions,omitempty" yaml:"apiVersions,omitempty"`

	// Compatibility
	MinKubernetesVersion string   `json:"minKubernetesVersion,omitempty" yaml:"minKubernetesVersion,omitempty"`
	SupportedPlatforms   []string `json:"supportedPlatforms,omitempty" yaml:"supportedPlatforms,omitempty"`

	// Configuration
	ConfigSchema  *FunctionConfigSchema  `json:"configSchema,omitempty" yaml:"configSchema,omitempty"`
	Examples      []*FunctionExample     `json:"examples,omitempty" yaml:"examples,omitempty"`
	Documentation *FunctionDocumentation `json:"documentation,omitempty" yaml:"documentation,omitempty"`

	// Quality metrics
	Quality         *QualityMetrics  `json:"quality,omitempty" yaml:"quality,omitempty"`
	Performance     *PerformanceInfo `json:"performance,omitempty" yaml:"performance,omitempty"`
	SecurityProfile *SecurityProfile `json:"securityProfile,omitempty" yaml:"securityProfile,omitempty"`

	// Lifecycle
	Author          string    `json:"author,omitempty" yaml:"author,omitempty"`
	License         string    `json:"license,omitempty" yaml:"license,omitempty"`
	CreatedAt       time.Time `json:"createdAt,omitempty" yaml:"createdAt,omitempty"`
	UpdatedAt       time.Time `json:"updatedAt,omitempty" yaml:"updatedAt,omitempty"`
	Deprecated      bool      `json:"deprecated,omitempty" yaml:"deprecated,omitempty"`
	DeprecationNote string    `json:"deprecationNote,omitempty" yaml:"deprecationNote,omitempty"`

	// Registry metadata
	Repository   string    `json:"repository" yaml:"repository"`
	Source       string    `json:"source" yaml:"source"`
	Checksum     string    `json:"checksum,omitempty" yaml:"checksum,omitempty"`
	LastVerified time.Time `json:"lastVerified,omitempty" yaml:"lastVerified,omitempty"`
}

// ResourceTypeSupport defines support for Kubernetes resource types
type ResourceTypeSupport struct {
	APIVersion string   `json:"apiVersion" yaml:"apiVersion"`
	Kind       string   `json:"kind" yaml:"kind"`
	Operations []string `json:"operations" yaml:"operations"` // create, update, delete, patch
}

// FunctionConfigSchema defines the configuration schema for a function
type FunctionConfigSchema struct {
	Type        string                           `json:"type" yaml:"type"`
	Properties  map[string]*SchemaProperty       `json:"properties,omitempty" yaml:"properties,omitempty"`
	Required    []string                         `json:"required,omitempty" yaml:"required,omitempty"`
	Definitions map[string]*FunctionConfigSchema `json:"definitions,omitempty" yaml:"definitions,omitempty"`
}

// SchemaProperty defines a configuration property
type SchemaProperty struct {
	Type        string        `json:"type" yaml:"type"`
	Description string        `json:"description,omitempty" yaml:"description,omitempty"`
	Default     interface{}   `json:"default,omitempty" yaml:"default,omitempty"`
	Examples    []interface{} `json:"examples,omitempty" yaml:"examples,omitempty"`
	Enum        []interface{} `json:"enum,omitempty" yaml:"enum,omitempty"`
	Pattern     string        `json:"pattern,omitempty" yaml:"pattern,omitempty"`
	Minimum     *float64      `json:"minimum,omitempty" yaml:"minimum,omitempty"`
	Maximum     *float64      `json:"maximum,omitempty" yaml:"maximum,omitempty"`
}

// FunctionExample provides usage examples for a function
type FunctionExample struct {
	Name        string                 `json:"name" yaml:"name"`
	Description string                 `json:"description,omitempty" yaml:"description,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty" yaml:"config,omitempty"`
	Input       []interface{}          `json:"input,omitempty" yaml:"input,omitempty"`
	Output      []interface{}          `json:"output,omitempty" yaml:"output,omitempty"`
}

// FunctionDocumentation contains documentation links and content
type FunctionDocumentation struct {
	README     string            `json:"readme,omitempty" yaml:"readme,omitempty"`
	UsageGuide string            `json:"usageGuide,omitempty" yaml:"usageGuide,omitempty"`
	APIRef     string            `json:"apiRef,omitempty" yaml:"apiRef,omitempty"`
	Links      map[string]string `json:"links,omitempty" yaml:"links,omitempty"`
}

// QualityMetrics provides quality assessment of the function
type QualityMetrics struct {
	TestCoverage   float64 `json:"testCoverage,omitempty" yaml:"testCoverage,omitempty"`
	CodeQuality    float64 `json:"codeQuality,omitempty" yaml:"codeQuality,omitempty"`
	Documentation  float64 `json:"documentation,omitempty" yaml:"documentation,omitempty"`
	CommunityScore float64 `json:"communityScore,omitempty" yaml:"communityScore,omitempty"`
	SecurityScore  float64 `json:"securityScore,omitempty" yaml:"securityScore,omitempty"`
	OverallScore   float64 `json:"overallScore,omitempty" yaml:"overallScore,omitempty"`
}

// PerformanceInfo provides performance characteristics
type PerformanceInfo struct {
	TypicalExecutionTime       time.Duration `json:"typicalExecutionTime,omitempty" yaml:"typicalExecutionTime,omitempty"`
	MaxExecutionTime           time.Duration `json:"maxExecutionTime,omitempty" yaml:"maxExecutionTime,omitempty"`
	TypicalMemoryUsage         int64         `json:"typicalMemoryUsage,omitempty" yaml:"typicalMemoryUsage,omitempty"`
	MaxMemoryUsage             int64         `json:"maxMemoryUsage,omitempty" yaml:"maxMemoryUsage,omitempty"`
	CPUIntensive               bool          `json:"cpuIntensive,omitempty" yaml:"cpuIntensive,omitempty"`
	NetworkIO                  bool          `json:"networkIo,omitempty" yaml:"networkIo,omitempty"`
	DiskIO                     bool          `json:"diskIo,omitempty" yaml:"diskIo,omitempty"`
	ScalabilityCharacteristics string        `json:"scalabilityCharacteristics,omitempty" yaml:"scalabilityCharacteristics,omitempty"`
}

// SecurityProfile defines security characteristics
type SecurityProfile struct {
	RequiredCapabilities []string                `json:"requiredCapabilities,omitempty" yaml:"requiredCapabilities,omitempty"`
	RequiredPrivileges   []string                `json:"requiredPrivileges,omitempty" yaml:"requiredPrivileges,omitempty"`
	NetworkAccess        bool                    `json:"networkAccess,omitempty" yaml:"networkAccess,omitempty"`
	FileSystemAccess     []string                `json:"fileSystemAccess,omitempty" yaml:"fileSystemAccess,omitempty"`
	SecurityRisks        []string                `json:"securityRisks,omitempty" yaml:"securityRisks,omitempty"`
	LastSecurityScan     time.Time               `json:"lastSecurityScan,omitempty" yaml:"lastSecurityScan,omitempty"`
	Vulnerabilities      []SecurityVulnerability `json:"vulnerabilities,omitempty" yaml:"vulnerabilities,omitempty"`
}

// SecurityVulnerability represents a security vulnerability
type SecurityVulnerability struct {
	ID           string    `json:"id" yaml:"id"`
	Severity     string    `json:"severity" yaml:"severity"`
	Description  string    `json:"description" yaml:"description"`
	FixVersion   string    `json:"fixVersion,omitempty" yaml:"fixVersion,omitempty"`
	URL          string    `json:"url,omitempty" yaml:"url,omitempty"`
	DiscoveredAt time.Time `json:"discoveredAt" yaml:"discoveredAt"`
}

// FunctionCache manages function metadata and artifact caching
type FunctionCache struct {
	cacheDir string
	ttl      time.Duration
	maxSize  int64
	mu       sync.RWMutex
	items    map[string]*CacheItem
	metrics  *CacheMetrics
}

// CacheItem represents a cached function
type CacheItem struct {
	Key         string            `json:"key"`
	Metadata    *FunctionMetadata `json:"metadata"`
	Data        []byte            `json:"data,omitempty"`
	Size        int64             `json:"size"`
	ExpiresAt   time.Time         `json:"expiresAt"`
	AccessCount int64             `json:"accessCount"`
	LastAccess  time.Time         `json:"lastAccess"`
}

// HealthMonitor monitors repository and function health
type HealthMonitor struct {
	registry           *Registry
	checkInterval      time.Duration
	unhealthyThreshold int
	healthyThreshold   int
	checks             map[string]*HealthCheck
	mu                 sync.RWMutex
}

// HealthCheck tracks health of a component
type HealthCheck struct {
	Name             string    `json:"name"`
	Status           string    `json:"status"` // healthy, unhealthy, unknown
	LastCheck        time.Time `json:"lastCheck"`
	ConsecutiveFails int       `json:"consecutiveFails"`
	Error            string    `json:"error,omitempty"`
}

// FunctionUpdate represents a function update notification
type FunctionUpdate struct {
	Type     string            `json:"type"` // added, updated, removed
	Function *FunctionMetadata `json:"function"`
}

// Registry metrics
type RegistryMetrics struct {
	RepositoryCount     prometheus.Gauge
	FunctionCount       *prometheus.GaugeVec
	DiscoveryDuration   *prometheus.HistogramVec
	CacheHitRate        prometheus.Counter
	CacheMissRate       prometheus.Counter
	HealthCheckFailures *prometheus.CounterVec
}

// CacheMetrics provides cache performance metrics
type CacheMetrics struct {
	Hits      prometheus.Counter
	Misses    prometheus.Counter
	Evictions prometheus.Counter
	Size      prometheus.Gauge
	ItemCount prometheus.Gauge
}

// Default registry configuration
var DefaultRegistryConfig = &RegistryConfig{
	DefaultRepositories: []RepositoryConfig{
		{
			Name:     "nephoran-functions",
			URL:      "https://github.com/nephoran/krm-functions",
			Type:     "git",
			Branch:   "main",
			Enabled:  true,
			Priority: 100,
		},
		{
			Name:     "kpt-functions",
			URL:      "https://catalog.kpt.dev/functions",
			Type:     "http",
			Enabled:  true,
			Priority: 50,
		},
	},
	RepositoryTimeout:      30 * time.Second,
	CacheDir:               "/tmp/krm-registry",
	CacheTTL:               24 * time.Hour,
	MaxCacheSize:           1 * 1024 * 1024 * 1024, // 1GB
	EnableCompression:      true,
	HealthCheckInterval:    5 * time.Minute,
	UnhealthyThreshold:     3,
	HealthyThreshold:       2,
	DiscoveryInterval:      1 * time.Hour,
	ConcurrentDiscovery:    5,
	AllowInsecure:          false,
	TrustedRegistries:      []string{},
	SignatureVerification:  true,
	MaxConcurrentDownloads: 10,
	DownloadTimeout:        5 * time.Minute,
	RetryAttempts:          3,
}

// NewRegistry creates a new function registry with comprehensive capabilities
func NewRegistry(config *RegistryConfig) (*Registry, error) {
	if config == nil {
		config = DefaultRegistryConfig
	}

	// Validate configuration
	if err := validateRegistryConfig(config); err != nil {
		return nil, errors.WithContext(err, "invalid registry configuration")
	}

	// Initialize metrics
	registryMetrics := &RegistryMetrics{
		RepositoryCount: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "krm_registry_repositories_total",
			Help: "Total number of configured repositories",
		}),
		FunctionCount: *promauto.NewGaugeVec(prometheus.GaugeOpts{
			Name: "krm_registry_functions_total",
			Help: "Total number of functions by repository",
		}, []string{"repository"}),
		DiscoveryDuration: *promauto.NewHistogramVec(prometheus.HistogramOpts{
			Name: "krm_registry_discovery_duration_seconds",
			Help: "Duration of function discovery operations",
		}, []string{"repository"}),
		CacheHitRate: promauto.NewCounter(prometheus.CounterOpts{
			Name: "krm_registry_cache_hits_total",
			Help: "Total number of cache hits",
		}),
		CacheMissRate: promauto.NewCounter(prometheus.CounterOpts{
			Name: "krm_registry_cache_misses_total",
			Help: "Total number of cache misses",
		}),
		HealthCheckFailures: *promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "krm_registry_health_check_failures_total",
			Help: "Total number of health check failures",
		}, []string{"repository", "reason"}),
	}

	// Initialize cache
	cacheMetrics := &CacheMetrics{
		Hits: promauto.NewCounter(prometheus.CounterOpts{
			Name: "krm_registry_cache_operations_total",
			Help: "Total number of cache operations",
		}),
		Misses: promauto.NewCounter(prometheus.CounterOpts{
			Name: "krm_registry_cache_misses_total",
			Help: "Total number of cache misses",
		}),
		Evictions: promauto.NewCounter(prometheus.CounterOpts{
			Name: "krm_registry_cache_evictions_total",
			Help: "Total number of cache evictions",
		}),
		Size: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "krm_registry_cache_size_bytes",
			Help: "Current cache size in bytes",
		}),
		ItemCount: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "krm_registry_cache_items_total",
			Help: "Current number of cached items",
		}),
	}

	cache := &FunctionCache{
		cacheDir: config.CacheDir,
		ttl:      config.CacheTTL,
		maxSize:  config.MaxCacheSize,
		items:    make(map[string]*CacheItem),
		metrics:  cacheMetrics,
	}

	// Initialize health monitor
	healthMonitor := &HealthMonitor{
		checkInterval:      config.HealthCheckInterval,
		unhealthyThreshold: config.UnhealthyThreshold,
		healthyThreshold:   config.HealthyThreshold,
		checks:             make(map[string]*HealthCheck),
	}

	// Create HTTP client with timeout
	httpClient := &http.Client{
		Timeout: config.RepositoryTimeout,
	}

	// Create registry
	registry := &Registry{
		config:        config,
		repositories:  make(map[string]*Repository),
		functions:     make(map[string]*FunctionMetadata),
		cache:         cache,
		healthMonitor: healthMonitor,
		metrics:       registryMetrics,
		tracer:        otel.Tracer("krm-registry"),
		updateChan:    make(chan *FunctionUpdate, 100),
		shutdownChan:  make(chan struct{}),
		httpClient:    httpClient,
	}

	healthMonitor.registry = registry

	// Initialize cache directory
	if err := os.MkdirAll(config.CacheDir, 0755); err != nil {
		return nil, errors.WithContext(err, "failed to create cache directory")
	}

	// Initialize default repositories
	for _, repoConfig := range config.DefaultRepositories {
		if repoConfig.Enabled {
			repo := &Repository{
				Config: repoConfig,
				Health: RepositoryHealth{
					Status:      "unknown",
					LastCheck:   time.Time{},
					IsReachable: false,
				},
				Functions: []*FunctionMetadata{},
			}
			registry.repositories[repoConfig.Name] = repo
		}
	}

	// Start background processes
	go registry.discoveryWorker()
	go registry.healthMonitorWorker()
	go registry.cacheCleanupWorker()

	return registry, nil
}

// DiscoverFunctions discovers functions from all configured repositories
func (r *Registry) DiscoverFunctions(ctx context.Context) error {
	ctx, span := r.tracer.Start(ctx, "krm-registry-discover-functions")
	defer span.End()

	logger := log.FromContext(ctx).WithName("krm-registry")

	r.mu.RLock()
	repositories := make([]*Repository, 0, len(r.repositories))
	for _, repo := range r.repositories {
		if repo.Config.Enabled {
			repositories = append(repositories, repo)
		}
	}
	r.mu.RUnlock()

	span.SetAttributes(attribute.Int("repositories.count", len(repositories)))
	logger.Info("Starting function discovery", "repositories", len(repositories))

	// Create semaphore for concurrent discovery
	sem := make(chan struct{}, r.config.ConcurrentDiscovery)
	var wg sync.WaitGroup
	var discoverErr error
	var errMu sync.Mutex

	for _, repo := range repositories {
		wg.Add(1)
		go func(repository *Repository) {
			defer wg.Done()

			// Acquire semaphore
			sem <- struct{}{}
			defer func() { <-sem }()

			if err := r.discoverRepository(ctx, repository); err != nil {
				errMu.Lock()
				if discoverErr == nil {
					discoverErr = fmt.Errorf("discovery failed")
				}
				discoverErr = fmt.Errorf("%w; %s: %v", discoverErr, repository.Config.Name, err)
				errMu.Unlock()
			}
		}(repo)
	}

	wg.Wait()

	// Update function index
	r.updateFunctionIndex()

	if discoverErr != nil {
		span.RecordError(discoverErr)
		span.SetStatus(codes.Error, "discovery partially failed")
		logger.Error(discoverErr, "Function discovery completed with errors")
		return discoverErr
	}

	span.SetStatus(codes.Ok, "discovery completed")
	logger.Info("Function discovery completed successfully")
	return nil
}

// GetFunction retrieves function metadata by name
func (r *Registry) GetFunction(ctx context.Context, name string) (*FunctionMetadata, error) {
	ctx, span := r.tracer.Start(ctx, "krm-registry-get-function")
	defer span.End()

	span.SetAttributes(attribute.String("function.name", name))

	r.mu.RLock()
	function, exists := r.functions[name]
	r.mu.RUnlock()

	if !exists {
		span.SetStatus(codes.Error, "function not found")
		return nil, fmt.Errorf("function %s not found", name)
	}

	span.SetStatus(codes.Ok, "function found")
	return function.deepCopy(), nil
}

// ListFunctions returns all available functions with optional filtering
func (r *Registry) ListFunctions(ctx context.Context, filter *FunctionFilter) ([]*FunctionMetadata, error) {
	ctx, span := r.tracer.Start(ctx, "krm-registry-list-functions")
	defer span.End()

	r.mu.RLock()
	allFunctions := make([]*FunctionMetadata, 0, len(r.functions))
	for _, function := range r.functions {
		allFunctions = append(allFunctions, function)
	}
	r.mu.RUnlock()

	// Apply filters
	filteredFunctions := r.applyFilter(allFunctions, filter)

	span.SetAttributes(
		attribute.Int("functions.total", len(allFunctions)),
		attribute.Int("functions.filtered", len(filteredFunctions)),
	)
	span.SetStatus(codes.Ok, "functions listed")

	return filteredFunctions, nil
}

// SearchFunctions searches for functions using various criteria
func (r *Registry) SearchFunctions(ctx context.Context, query *SearchQuery) (*SearchResult, error) {
	ctx, span := r.tracer.Start(ctx, "krm-registry-search-functions")
	defer span.End()

	span.SetAttributes(attribute.String("search.query", query.Terms))

	functions, err := r.ListFunctions(ctx, nil)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "failed to list functions")
		return nil, err
	}

	// Perform search
	matches := r.performSearch(functions, query)

	result := &SearchResult{
		Query:      query,
		Matches:    matches,
		TotalCount: len(functions),
		MatchCount: len(matches),
		SearchTime: time.Now(),
	}

	span.SetAttributes(
		attribute.Int("search.total", result.TotalCount),
		attribute.Int("search.matches", result.MatchCount),
	)
	span.SetStatus(codes.Ok, "search completed")

	return result, nil
}

// AddRepository adds a new repository to the registry
func (r *Registry) AddRepository(ctx context.Context, config RepositoryConfig) error {
	ctx, span := r.tracer.Start(ctx, "krm-registry-add-repository")
	defer span.End()

	span.SetAttributes(attribute.String("repository.name", config.Name))

	if err := r.validateRepositoryConfig(&config); err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "invalid repository config")
		return errors.WithContext(err, "invalid repository configuration")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.repositories[config.Name]; exists {
		err := fmt.Errorf("repository %s already exists", config.Name)
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository exists")
		return err
	}

	repo := &Repository{
		Config: config,
		Health: RepositoryHealth{
			Status:      "unknown",
			LastCheck:   time.Time{},
			IsReachable: false,
		},
		Functions: []*FunctionMetadata{},
	}

	r.repositories[config.Name] = repo
	r.metrics.RepositoryCount.Inc()

	span.SetStatus(codes.Ok, "repository added")
	return nil
}

// RemoveRepository removes a repository from the registry
func (r *Registry) RemoveRepository(ctx context.Context, name string) error {
	ctx, span := r.tracer.Start(ctx, "krm-registry-remove-repository")
	defer span.End()

	span.SetAttributes(attribute.String("repository.name", name))

	r.mu.Lock()
	defer r.mu.Unlock()

	repo, exists := r.repositories[name]
	if !exists {
		err := fmt.Errorf("repository %s not found", name)
		span.RecordError(err)
		span.SetStatus(codes.Error, "repository not found")
		return err
	}

	// Remove functions from this repository
	for _, function := range repo.Functions {
		delete(r.functions, function.Name)
		r.metrics.FunctionCount.WithLabelValues(name).Dec()
	}

	delete(r.repositories, name)
	r.metrics.RepositoryCount.Dec()

	span.SetStatus(codes.Ok, "repository removed")
	return nil
}

// GetRepositoryHealth returns health status of a repository
func (r *Registry) GetRepositoryHealth(ctx context.Context, name string) (*RepositoryHealth, error) {
	r.mu.RLock()
	repo, exists := r.repositories[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("repository %s not found", name)
	}

	repo.mu.RLock()
	health := repo.Health
	repo.mu.RUnlock()

	return &health, nil
}

// GetHealth returns overall registry health
func (r *Registry) GetHealth() *RegistryHealth {
	r.mu.RLock()
	defer r.mu.RUnlock()

	health := &RegistryHealth{
		Status:        "healthy",
		Repositories:  len(r.repositories),
		Functions:     len(r.functions),
		LastDiscovery: time.Time{},
		CacheStatus:   r.cache.getStatus(),
	}

	// Check repository health
	unhealthyRepos := 0
	for _, repo := range r.repositories {
		repo.mu.RLock()
		if repo.Health.Status != "healthy" {
			unhealthyRepos++
		}
		if repo.LastSyncTime.After(health.LastDiscovery) {
			health.LastDiscovery = repo.LastSyncTime
		}
		repo.mu.RUnlock()
	}

	if unhealthyRepos > 0 {
		health.Status = "degraded"
		if unhealthyRepos >= len(r.repositories) {
			health.Status = "unhealthy"
		}
	}

	return health
}

// Shutdown gracefully shuts down the registry
func (r *Registry) Shutdown(ctx context.Context) error {
	logger := log.FromContext(ctx).WithName("krm-registry")
	logger.Info("Shutting down KRM registry")

	// Signal shutdown
	close(r.shutdownChan)

	// Save cache
	if err := r.cache.save(); err != nil {
		logger.Error(err, "Failed to save cache during shutdown")
	}

	logger.Info("KRM registry shutdown complete")
	return nil
}

// Private helper methods

func (r *Registry) discoverRepository(ctx context.Context, repo *Repository) error {
	startTime := time.Now()
	defer func() {
		duration := time.Since(startTime)
		r.metrics.DiscoveryDuration.WithLabelValues(repo.Config.Name).Observe(duration.Seconds())
	}()

	logger := log.FromContext(ctx).WithName("krm-registry").WithValues("repository", repo.Config.Name)

	switch repo.Config.Type {
	case "git":
		return r.discoverGitRepository(ctx, repo)
	case "oci":
		return r.discoverOCIRepository(ctx, repo)
	case "http":
		return r.discoverHTTPRepository(ctx, repo)
	default:
		err := fmt.Errorf("unsupported repository type: %s", repo.Config.Type)
		logger.Error(err, "Repository discovery failed")
		return err
	}
}

func (r *Registry) discoverHTTPRepository(ctx context.Context, repo *Repository) error {
	req, err := http.NewRequestWithContext(ctx, "GET", repo.Config.URL, nil)
	if err != nil {
		return err
	}

	// Add authentication if configured
	if repo.Config.Auth != nil {
		r.addAuthentication(req, repo.Config.Auth)
	}

	// Add custom headers
	for key, value := range repo.Config.Headers {
		req.Header.Set(key, value)
	}

	resp, err := r.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	var catalog FunctionCatalog
	if err := json.NewDecoder(resp.Body).Decode(&catalog); err != nil {
		return err
	}

	// Update repository with discovered functions
	repo.mu.Lock()
	repo.Functions = catalog.Functions
	repo.LastSyncTime = time.Now()
	repo.LastError = ""
	repo.Version = catalog.Version
	repo.mu.Unlock()

	return nil
}

func (r *Registry) discoverGitRepository(ctx context.Context, repo *Repository) error {
	// Git repository discovery would be implemented here
	// For now, return a placeholder implementation
	return fmt.Errorf("git repository discovery not yet implemented")
}

func (r *Registry) discoverOCIRepository(ctx context.Context, repo *Repository) error {
	// OCI repository discovery would be implemented here
	// For now, return a placeholder implementation
	return fmt.Errorf("OCI repository discovery not yet implemented")
}

func (r *Registry) addAuthentication(req *http.Request, auth *AuthConfig) {
	switch auth.Type {
	case "basic":
		req.SetBasicAuth(auth.Username, auth.Password)
	case "bearer":
		req.Header.Set("Authorization", "Bearer "+auth.Token)
	case "api-key":
		req.Header.Set("X-API-Key", auth.Token)
	}
}

func (r *Registry) updateFunctionIndex() {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Clear existing index
	r.functions = make(map[string]*FunctionMetadata)

	// Rebuild index from all repositories
	for repoName, repo := range r.repositories {
		repo.mu.RLock()
		for _, function := range repo.Functions {
			function.Repository = repoName
			r.functions[function.Name] = function
		}
		repo.mu.RUnlock()
		r.metrics.FunctionCount.WithLabelValues(repoName).Set(float64(len(repo.Functions)))
	}
}

func (r *Registry) discoveryWorker() {
	ticker := time.NewTicker(r.config.DiscoveryInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			ctx := context.Background()
			if err := r.DiscoverFunctions(ctx); err != nil {
				log.Log.Error(err, "Background function discovery failed")
			}
		case <-r.shutdownChan:
			return
		}
	}
}

func (r *Registry) healthMonitorWorker() {
	ticker := time.NewTicker(r.healthMonitor.checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.healthMonitor.checkHealth()
		case <-r.shutdownChan:
			return
		}
	}
}

func (r *Registry) cacheCleanupWorker() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.cache.cleanup()
		case <-r.shutdownChan:
			return
		}
	}
}

// Supporting types and helper functions

type RepositoryHealth struct {
	Status      string    `json:"status"`
	LastCheck   time.Time `json:"lastCheck"`
	IsReachable bool      `json:"isReachable"`
	Error       string    `json:"error,omitempty"`
}

type FunctionFilter struct {
	Type       string   `json:"type,omitempty"`
	Categories []string `json:"categories,omitempty"`
	Keywords   []string `json:"keywords,omitempty"`
	Repository string   `json:"repository,omitempty"`
	MinVersion string   `json:"minVersion,omitempty"`
	Deprecated *bool    `json:"deprecated,omitempty"`
}

type SearchQuery struct {
	Terms      string   `json:"terms"`
	Categories []string `json:"categories,omitempty"`
	Limit      int      `json:"limit,omitempty"`
	Offset     int      `json:"offset,omitempty"`
}

type SearchResult struct {
	Query      *SearchQuery        `json:"query"`
	Matches    []*FunctionMetadata `json:"matches"`
	TotalCount int                 `json:"totalCount"`
	MatchCount int                 `json:"matchCount"`
	SearchTime time.Time           `json:"searchTime"`
}

type FunctionCatalog struct {
	Version   string              `json:"version"`
	Functions []*FunctionMetadata `json:"functions"`
}

type RegistryHealth struct {
	Status        string       `json:"status"`
	Repositories  int          `json:"repositories"`
	Functions     int          `json:"functions"`
	LastDiscovery time.Time    `json:"lastDiscovery"`
	CacheStatus   *CacheStatus `json:"cacheStatus"`
}

type CacheStatus struct {
	Size      int64     `json:"size"`
	Items     int       `json:"items"`
	HitRate   float64   `json:"hitRate"`
	LastClean time.Time `json:"lastClean"`
}

// Helper methods

func (f *FunctionMetadata) deepCopy() *FunctionMetadata {
	data, _ := json.Marshal(f)
	var copy FunctionMetadata
	json.Unmarshal(data, &copy)
	return &copy
}

func (r *Registry) applyFilter(functions []*FunctionMetadata, filter *FunctionFilter) []*FunctionMetadata {
	if filter == nil {
		return functions
	}

	var filtered []*FunctionMetadata
	for _, function := range functions {
		if r.matchesFilter(function, filter) {
			filtered = append(filtered, function)
		}
	}
	return filtered
}

func (r *Registry) matchesFilter(function *FunctionMetadata, filter *FunctionFilter) bool {
	if filter.Type != "" && function.Type != filter.Type {
		return false
	}
	if filter.Repository != "" && function.Repository != filter.Repository {
		return false
	}
	if filter.Deprecated != nil && function.Deprecated != *filter.Deprecated {
		return false
	}
	// Add more filter logic as needed
	return true
}

func (r *Registry) performSearch(functions []*FunctionMetadata, query *SearchQuery) []*FunctionMetadata {
	if query.Terms == "" {
		return functions
	}

	terms := strings.ToLower(query.Terms)
	var matches []*FunctionMetadata

	for _, function := range functions {
		score := r.calculateSearchScore(function, terms)
		if score > 0 {
			matches = append(matches, function)
		}
	}

	// Apply limit and offset
	if query.Limit > 0 && len(matches) > query.Limit {
		start := query.Offset
		if start > len(matches) {
			start = len(matches)
		}
		end := start + query.Limit
		if end > len(matches) {
			end = len(matches)
		}
		matches = matches[start:end]
	}

	return matches
}

func (r *Registry) calculateSearchScore(function *FunctionMetadata, terms string) float64 {
	score := 0.0

	// Check name (highest weight)
	if strings.Contains(strings.ToLower(function.Name), terms) {
		score += 10.0
	}

	// Check description
	if strings.Contains(strings.ToLower(function.Description), terms) {
		score += 5.0
	}

	// Check keywords
	for _, keyword := range function.Keywords {
		if strings.Contains(strings.ToLower(keyword), terms) {
			score += 3.0
		}
	}

	// Check categories
	for _, category := range function.Categories {
		if strings.Contains(strings.ToLower(category), terms) {
			score += 2.0
		}
	}

	return score
}

func (c *FunctionCache) getStatus() *CacheStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	totalSize := int64(0)
	for _, item := range c.items {
		totalSize += item.Size
	}

	return &CacheStatus{
		Size:      totalSize,
		Items:     len(c.items),
		HitRate:   0.0, // Would calculate from metrics
		LastClean: time.Now(),
	}
}

func (c *FunctionCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	for key, item := range c.items {
		if now.After(item.ExpiresAt) {
			delete(c.items, key)
			c.metrics.Evictions.Inc()
		}
	}
}

func (c *FunctionCache) save() error {
	// Implementation to save cache to disk
	return nil
}

func (hm *HealthMonitor) checkHealth() {
	hm.mu.Lock()
	defer hm.mu.Unlock()

	for name, repo := range hm.registry.repositories {
		check := hm.checks[name]
		if check == nil {
			check = &HealthCheck{
				Name:   name,
				Status: "unknown",
			}
			hm.checks[name] = check
		}

		// Perform health check
		healthy := hm.checkRepositoryHealth(repo)

		check.LastCheck = time.Now()
		if healthy {
			check.ConsecutiveFails = 0
			if check.Status != "healthy" && check.ConsecutiveFails <= -hm.healthyThreshold {
				check.Status = "healthy"
			}
		} else {
			check.ConsecutiveFails++
			if check.ConsecutiveFails >= hm.unhealthyThreshold {
				check.Status = "unhealthy"
			}
		}
	}
}

func (hm *HealthMonitor) checkRepositoryHealth(repo *Repository) bool {
	// Simple health check - could be more sophisticated
	repo.mu.RLock()
	lastSync := repo.LastSyncTime
	lastError := repo.LastError
	repo.mu.RUnlock()

	// Consider healthy if synced recently and no errors
	return time.Since(lastSync) < 2*hm.checkInterval && lastError == ""
}

func validateRegistryConfig(config *RegistryConfig) error {
	if config.CacheDir == "" {
		return fmt.Errorf("cacheDir is required")
	}
	if config.DiscoveryInterval <= 0 {
		return fmt.Errorf("discoveryInterval must be positive")
	}
	if config.HealthCheckInterval <= 0 {
		return fmt.Errorf("healthCheckInterval must be positive")
	}
	return nil
}

func (r *Registry) validateRepositoryConfig(config *RepositoryConfig) error {
	if config.Name == "" {
		return fmt.Errorf("repository name is required")
	}
	if config.URL == "" {
		return fmt.Errorf("repository URL is required")
	}
	if config.Type == "" {
		return fmt.Errorf("repository type is required")
	}

	// Validate URL
	if _, err := url.Parse(config.URL); err != nil {
		return fmt.Errorf("invalid repository URL: %v", err)
	}

	return nil
}
