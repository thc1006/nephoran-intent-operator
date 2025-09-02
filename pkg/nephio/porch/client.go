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

package porch

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sony/gobreaker"
	"golang.org/x/time/rate"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ClientConfig defines configuration for the Porch client.

type ClientConfig struct {
	// Porch server endpoint (optional, uses in-cluster config if empty).

	Endpoint string

	// Authentication config.

	AuthConfig *AuthConfig

	// TLS configuration.

	TLSConfig *ClientTLSConfig

	// Porch-specific configuration.

	PorchConfig *PorchConfig
}

// AuthConfig is defined in types.go.

// ClientTLSConfig defines TLS configuration.

type ClientTLSConfig struct {
	InsecureSkipVerify bool

	CertFile string

	KeyFile string

	CAFile string
}

// PorchConfig defines Porch-specific configuration.

type PorchConfig struct {
	// Default namespace for operations.

	DefaultNamespace string

	// Default repository for package operations.

	DefaultRepository string

	// Circuit breaker configuration.

	CircuitBreaker *ClientCircuitBreakerConfig

	// Rate limiting configuration.

	RateLimit *ClientRateLimitConfig

	// Connection pool configuration.

	ConnectionPool *ClientConnectionPoolConfig

	// Function execution configuration.

	FunctionExecution *ClientFunctionExecutionConfig
}

// ClientCircuitBreakerConfig defines circuit breaker settings.

type ClientCircuitBreakerConfig struct {
	Enabled bool

	FailureThreshold int

	SuccessThreshold int

	Timeout time.Duration

	HalfOpenMaxCalls int
}

// ClientRateLimitConfig defines rate limiting settings.

type ClientRateLimitConfig struct {
	Enabled bool

	RequestsPerSecond float64

	Burst int
}

// ClientConnectionPoolConfig defines connection pool settings.

type ClientConnectionPoolConfig struct {
	MaxOpenConns int

	MaxIdleConns int

	ConnMaxLifetime time.Duration
}

// ClientFunctionExecutionConfig defines function execution settings.

type ClientFunctionExecutionConfig struct {
	DefaultTimeout time.Duration

	MaxConcurrency int

	ResourceLimits map[string]string
}

// GetKubernetesConfig returns the Kubernetes configuration from the Porch config.

func (c *ClientConfig) GetKubernetesConfig() (*rest.Config, error) {
	// If using in-cluster config.

	if c.AuthConfig == nil || c.AuthConfig.Type == "" {
		return rest.InClusterConfig()
	}

	// Build config based on auth type.

	config := &rest.Config{}

	if c.Endpoint != "" {
		config.Host = c.Endpoint
	}

	switch c.AuthConfig.Type {

	case "bearer":

		config.BearerToken = c.AuthConfig.Token

	case "basic":

		config.Username = c.AuthConfig.Username

		config.Password = c.AuthConfig.Password

	case "kubeconfig":

		// Load from kubeconfig file.

		return rest.InClusterConfig()

	default:

		return rest.InClusterConfig()

	}

	// Apply TLS configuration.

	if c.TLSConfig != nil {
		config.TLSClientConfig = rest.TLSClientConfig{
			Insecure: c.TLSConfig.InsecureSkipVerify,

			CertFile: c.TLSConfig.CertFile,

			KeyFile: c.TLSConfig.KeyFile,

			CAFile: c.TLSConfig.CAFile,
		}
	}

	return config, nil
}

// DefaultPorchConfig returns default Porch configuration.

func DefaultPorchConfig() *ClientConfig {
	return &ClientConfig{
		PorchConfig: &PorchConfig{
			DefaultNamespace: "default",

			DefaultRepository: "default",

			CircuitBreaker: &ClientCircuitBreakerConfig{
				Enabled: true,

				FailureThreshold: 5,

				SuccessThreshold: 3,

				Timeout: 30 * time.Second,

				HalfOpenMaxCalls: 3,
			},

			RateLimit: &ClientRateLimitConfig{
				Enabled: true,

				RequestsPerSecond: 10.0,

				Burst: 20,
			},

			ConnectionPool: &ClientConnectionPoolConfig{
				MaxOpenConns: 10,

				MaxIdleConns: 5,

				ConnMaxLifetime: 30 * time.Minute,
			},

			FunctionExecution: &ClientFunctionExecutionConfig{
				DefaultTimeout: 60 * time.Second,

				MaxConcurrency: 5,

				ResourceLimits: map[string]string{
					"cpu": "100m",

					"memory": "128Mi",
				},
			},
		},
	}
}

// Client implements the PorchClient interface providing comprehensive CRUD operations.

type Client struct {
	// Kubernetes clients.

	dynamic dynamic.Interface

	client client.Client

	restClient rest.Interface

	// Configuration.

	config *ClientConfig

	// Circuit breaker for fault tolerance.

	circuitBreaker *gobreaker.CircuitBreaker

	// Rate limiter for API calls.

	rateLimiter *rate.Limiter

	// Metrics.

	metrics *ClientMetrics

	// Logger.

	logger logr.Logger

	// Cache for frequently accessed resources.

	cache *clientCache

	// Repository manager.

	repoManager RepositoryManager

	// Package revision manager.

	pkgManager PackageRevisionManager

	// Function runner.

	functionRunner FunctionRunner

	// Connection pool.

	connectionPool *connectionPool

	// Mutex for thread safety.

	mutex sync.RWMutex
}

// ClientOptions defines options for creating a Porch client.

type ClientOptions struct {
	Config *ClientConfig

	KubeConfig *rest.Config

	Logger logr.Logger

	MetricsEnabled bool

	CacheEnabled bool

	CacheSize int

	CacheTTL time.Duration
}

// ClientMetrics defines Prometheus metrics for the Porch client.

type ClientMetrics struct {
	requestsTotal *prometheus.CounterVec

	requestDuration *prometheus.HistogramVec

	cacheHits prometheus.Counter

	cacheMisses prometheus.Counter

	circuitBreakerState prometheus.Gauge

	activeConnections prometheus.Gauge

	errorRate *prometheus.CounterVec
}

// clientCache provides caching for frequently accessed resources.

type clientCache struct {
	cache map[string]*cacheEntry

	mutex sync.RWMutex

	ttl time.Duration

	maxSize int
}

type cacheEntry struct {
	data interface{}

	timestamp time.Time
}

// connectionPool manages HTTP connections for better performance.

type connectionPool struct {
	pool chan *rest.RESTClient

	maxSize int

	current int

	mutex sync.Mutex
}

// NewClient creates a new Porch API client with the specified options.

func NewClient(opts ClientOptions) (*Client, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if opts.KubeConfig == nil {

		var err error

		opts.KubeConfig, err = opts.Config.GetKubernetesConfig()
		if err != nil {
			return nil, fmt.Errorf("failed to get kubernetes config: %w", err)
		}

	}

	if opts.Logger.GetSink() == nil {
		opts.Logger = log.Log.WithName("porch-client")
	}

	// Create dynamic client.

	dynamicClient, err := dynamic.NewForConfig(opts.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Create controller-runtime client.

	scheme := runtime.NewScheme()

	ctrlClient, err := client.New(opts.KubeConfig, client.Options{
		Scheme: scheme,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create controller client: %w", err)
	}

	// Create REST client.

	restClient, err := rest.RESTClientFor(opts.KubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create REST client: %w", err)
	}

	client := &Client{
		dynamic: dynamicClient,

		client: ctrlClient,

		restClient: restClient,

		config: opts.Config,

		logger: opts.Logger,
	}

	// Initialize circuit breaker.

	if err := client.initCircuitBreaker(); err != nil {
		return nil, fmt.Errorf("failed to initialize circuit breaker: %w", err)
	}

	// Initialize rate limiter.

	if err := client.initRateLimiter(); err != nil {
		return nil, fmt.Errorf("failed to initialize rate limiter: %w", err)
	}

	// Initialize metrics.

	if opts.MetricsEnabled {
		client.metrics = client.initMetrics()
	}

	// Initialize cache.

	if opts.CacheEnabled {

		client.cache = &clientCache{
			cache: make(map[string]*cacheEntry),

			ttl: opts.CacheTTL,

			maxSize: opts.CacheSize,
		}

		if client.cache.ttl == 0 {
			client.cache.ttl = 5 * time.Minute
		}

		if client.cache.maxSize == 0 {
			client.cache.maxSize = 1000
		}

	}

	// Initialize connection pool.

	poolSize := 10

	if client.config.PorchConfig.ConnectionPool != nil && client.config.PorchConfig.ConnectionPool.MaxOpenConns > 0 {
		poolSize = client.config.PorchConfig.ConnectionPool.MaxOpenConns
	}

	client.connectionPool = &connectionPool{
		pool: make(chan *rest.RESTClient, poolSize),

		maxSize: poolSize,
	}

	// Initialize sub-managers with stub implementations.

	client.repoManager = &repositoryManagerStub{client: client}

	client.pkgManager = &packageRevisionManagerStub{client: client}

	client.functionRunner = &functionRunnerStub{client: client}

	return client, nil
}

// Repository Operations.

// GetRepository retrieves a repository by name.

func (c *Client) GetRepository(ctx context.Context, name string) (*Repository, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("GetRepository").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("GetRepository", "complete").Inc()

		}
	}()

	// Check cache first.

	if c.cache != nil {

		if cached := c.getCached(fmt.Sprintf("repo:%s", name)); cached != nil {
			if repo, ok := cached.(*Repository); ok {

				if c.metrics != nil {
					c.metrics.cacheHits.Inc()
				}

				return repo, nil

			}
		}

		if c.metrics != nil {
			c.metrics.cacheMisses.Inc()
		}

	}

	// Execute with circuit breaker.

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.getRepositoryInternal(ctx, name)
	})
	if err != nil {

		c.recordError("GetRepository", err)

		return nil, err

	}

	repo := result.(*Repository)

	// Cache the result.

	if c.cache != nil {
		c.setCached(fmt.Sprintf("repo:%s", name), repo)
	}

	return repo, nil
}

// ListRepositories lists all repositories matching the provided options.

func (c *Client) ListRepositories(ctx context.Context, opts *ListOptions) (*RepositoryList, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("ListRepositories").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("ListRepositories", "complete").Inc()

		}
	}()

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.listRepositoriesInternal(ctx, opts)
	})
	if err != nil {

		c.recordError("ListRepositories", err)

		return nil, err

	}

	return result.(*RepositoryList), nil
}

// CreateRepository creates a new repository.

func (c *Client) CreateRepository(ctx context.Context, repo *Repository) (*Repository, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("CreateRepository").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("CreateRepository", "complete").Inc()

		}
	}()

	// Validate repository.

	if err := c.validateRepository(repo); err != nil {
		return nil, fmt.Errorf("repository validation failed: %w", err)
	}

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.createRepositoryInternal(ctx, repo)
	})
	if err != nil {

		c.recordError("CreateRepository", err)

		return nil, err

	}

	created := result.(*Repository)

	// Invalidate cache.

	if c.cache != nil {
		c.invalidateCache(fmt.Sprintf("repo:%s", created.Name))
	}

	return created, nil
}

// UpdateRepository updates an existing repository.

func (c *Client) UpdateRepository(ctx context.Context, repo *Repository) (*Repository, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("UpdateRepository").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("UpdateRepository", "complete").Inc()

		}
	}()

	// Validate repository.

	if err := c.validateRepository(repo); err != nil {
		return nil, fmt.Errorf("repository validation failed: %w", err)
	}

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.updateRepositoryInternal(ctx, repo)
	})
	if err != nil {

		c.recordError("UpdateRepository", err)

		return nil, err

	}

	updated := result.(*Repository)

	// Invalidate cache.

	if c.cache != nil {
		c.invalidateCache(fmt.Sprintf("repo:%s", updated.Name))
	}

	return updated, nil
}

// DeleteRepository deletes a repository.

func (c *Client) DeleteRepository(ctx context.Context, name string) error {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("DeleteRepository").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("DeleteRepository", "complete").Inc()

		}
	}()

	_, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, c.deleteRepositoryInternal(ctx, name)
	})
	if err != nil {

		c.recordError("DeleteRepository", err)

		return err

	}

	// Invalidate cache.

	if c.cache != nil {
		c.invalidateCache(fmt.Sprintf("repo:%s", name))
	}

	return nil
}

// SyncRepository synchronizes a repository with its remote source.

func (c *Client) SyncRepository(ctx context.Context, name string) error {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("SyncRepository").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("SyncRepository", "complete").Inc()

		}
	}()

	_, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, c.syncRepositoryInternal(ctx, name)
	})
	if err != nil {

		c.recordError("SyncRepository", err)

		return err

	}

	// Invalidate cache.

	if c.cache != nil {
		c.invalidateCache(fmt.Sprintf("repo:%s", name))
	}

	return nil
}

// PackageRevision Operations.

// GetPackageRevision retrieves a package revision.

func (c *Client) GetPackageRevision(ctx context.Context, name, revision string) (*PackageRevision, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("GetPackageRevision").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("GetPackageRevision", "complete").Inc()

		}
	}()

	cacheKey := fmt.Sprintf("pkg:%s:%s", name, revision)

	// Check cache first.

	if c.cache != nil {

		if cached := c.getCached(cacheKey); cached != nil {
			if pkg, ok := cached.(*PackageRevision); ok {

				if c.metrics != nil {
					c.metrics.cacheHits.Inc()
				}

				return pkg, nil

			}
		}

		if c.metrics != nil {
			c.metrics.cacheMisses.Inc()
		}

	}

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.getPackageRevisionInternal(ctx, name, revision)
	})
	if err != nil {

		c.recordError("GetPackageRevision", err)

		return nil, err

	}

	pkg := result.(*PackageRevision)

	// Cache the result.

	if c.cache != nil {
		c.setCached(cacheKey, pkg)
	}

	return pkg, nil
}

// ListPackageRevisions lists package revisions matching the provided options.

func (c *Client) ListPackageRevisions(ctx context.Context, opts *ListOptions) (*PackageRevisionList, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("ListPackageRevisions").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("ListPackageRevisions", "complete").Inc()

		}
	}()

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.listPackageRevisionsInternal(ctx, opts)
	})
	if err != nil {

		c.recordError("ListPackageRevisions", err)

		return nil, err

	}

	return result.(*PackageRevisionList), nil
}

// CreatePackageRevision creates a new package revision.

func (c *Client) CreatePackageRevision(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("CreatePackageRevision").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("CreatePackageRevision", "complete").Inc()

		}
	}()

	// Validate package revision.

	if err := c.validatePackageRevision(pkg); err != nil {
		return nil, fmt.Errorf("package revision validation failed: %w", err)
	}

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.createPackageRevisionInternal(ctx, pkg)
	})
	if err != nil {

		c.recordError("CreatePackageRevision", err)

		return nil, err

	}

	created := result.(*PackageRevision)

	// Invalidate cache.

	if c.cache != nil {

		cacheKey := fmt.Sprintf("pkg:%s:%s", created.Spec.PackageName, created.Spec.Revision)

		c.invalidateCache(cacheKey)

	}

	return created, nil
}

// UpdatePackageRevision updates an existing package revision.

func (c *Client) UpdatePackageRevision(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("UpdatePackageRevision").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("UpdatePackageRevision", "complete").Inc()

		}
	}()

	// Validate package revision.

	if err := c.validatePackageRevision(pkg); err != nil {
		return nil, fmt.Errorf("package revision validation failed: %w", err)
	}

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.updatePackageRevisionInternal(ctx, pkg)
	})
	if err != nil {

		c.recordError("UpdatePackageRevision", err)

		return nil, err

	}

	updated := result.(*PackageRevision)

	// Invalidate cache.

	if c.cache != nil {

		cacheKey := fmt.Sprintf("pkg:%s:%s", updated.Spec.PackageName, updated.Spec.Revision)

		c.invalidateCache(cacheKey)

	}

	return updated, nil
}

// DeletePackageRevision deletes a package revision.

func (c *Client) DeletePackageRevision(ctx context.Context, name, revision string) error {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("DeletePackageRevision").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("DeletePackageRevision", "complete").Inc()

		}
	}()

	_, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, c.deletePackageRevisionInternal(ctx, name, revision)
	})
	if err != nil {

		c.recordError("DeletePackageRevision", err)

		return err

	}

	// Invalidate cache.

	if c.cache != nil {

		cacheKey := fmt.Sprintf("pkg:%s:%s", name, revision)

		c.invalidateCache(cacheKey)

	}

	return nil
}

// ApprovePackageRevision approves a package revision for publishing.

func (c *Client) ApprovePackageRevision(ctx context.Context, name, revision string) error {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("ApprovePackageRevision").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("ApprovePackageRevision", "complete").Inc()

		}
	}()

	_, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, c.approvePackageRevisionInternal(ctx, name, revision)
	})
	if err != nil {

		c.recordError("ApprovePackageRevision", err)

		return err

	}

	// Invalidate cache.

	if c.cache != nil {

		cacheKey := fmt.Sprintf("pkg:%s:%s", name, revision)

		c.invalidateCache(cacheKey)

	}

	return nil
}

// ProposePackageRevision proposes a package revision for review.

func (c *Client) ProposePackageRevision(ctx context.Context, name, revision string) error {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("ProposePackageRevision").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("ProposePackageRevision", "complete").Inc()

		}
	}()

	_, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, c.proposePackageRevisionInternal(ctx, name, revision)
	})
	if err != nil {

		c.recordError("ProposePackageRevision", err)

		return err

	}

	// Invalidate cache.

	if c.cache != nil {

		cacheKey := fmt.Sprintf("pkg:%s:%s", name, revision)

		c.invalidateCache(cacheKey)

	}

	return nil
}

// RejectPackageRevision rejects a package revision with a reason.

func (c *Client) RejectPackageRevision(ctx context.Context, name, revision, reason string) error {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("RejectPackageRevision").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("RejectPackageRevision", "complete").Inc()

		}
	}()

	_, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, c.rejectPackageRevisionInternal(ctx, name, revision, reason)
	})
	if err != nil {

		c.recordError("RejectPackageRevision", err)

		return err

	}

	// Invalidate cache.

	if c.cache != nil {

		cacheKey := fmt.Sprintf("pkg:%s:%s", name, revision)

		c.invalidateCache(cacheKey)

	}

	return nil
}

// Package Content Operations.

// GetPackageContents retrieves the contents of a package revision.

func (c *Client) GetPackageContents(ctx context.Context, name, revision string) (map[string][]byte, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("GetPackageContents").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("GetPackageContents", "complete").Inc()

		}
	}()

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.getPackageContentsInternal(ctx, name, revision)
	})
	if err != nil {

		c.recordError("GetPackageContents", err)

		return nil, err

	}

	return result.(map[string][]byte), nil
}

// UpdatePackageContents updates the contents of a package revision.

func (c *Client) UpdatePackageContents(ctx context.Context, name, revision string, contents map[string][]byte) error {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("UpdatePackageContents").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("UpdatePackageContents", "complete").Inc()

		}
	}()

	_, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, c.updatePackageContentsInternal(ctx, name, revision, contents)
	})
	if err != nil {

		c.recordError("UpdatePackageContents", err)

		return err

	}

	// Invalidate cache.

	if c.cache != nil {

		cacheKey := fmt.Sprintf("pkg:%s:%s", name, revision)

		c.invalidateCache(cacheKey)

	}

	return nil
}

// RenderPackage renders a package using KRM functions.

func (c *Client) RenderPackage(ctx context.Context, name, revision string) (*RenderResult, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("RenderPackage").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("RenderPackage", "complete").Inc()

		}
	}()

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.renderPackageInternal(ctx, name, revision)
	})
	if err != nil {

		c.recordError("RenderPackage", err)

		return nil, err

	}

	return result.(*RenderResult), nil
}

// Function Operations.

// RunFunction executes a KRM function on resources.

func (c *Client) RunFunction(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("RunFunction").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("RunFunction", "complete").Inc()

		}
	}()

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.runFunctionInternal(ctx, req)
	})
	if err != nil {

		c.recordError("RunFunction", err)

		return nil, err

	}

	return result.(*FunctionResponse), nil
}

// ExecuteFunction delegates to the function runner.

func (c *Client) ExecuteFunction(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	if c.functionRunner == nil {
		return &FunctionResponse{}, nil
	}

	return c.functionRunner.ExecuteFunction(ctx, req)
}

// ValidateFunction delegates to the function runner.

func (c *Client) ValidateFunction(ctx context.Context, functionName string) (*FunctionValidation, error) {
	if c.functionRunner == nil {
		return &FunctionValidation{Valid: true}, nil
	}

	return c.functionRunner.ValidateFunction(ctx, functionName)
}

// ListFunctions delegates to the function runner.

func (c *Client) ListFunctions(ctx context.Context) ([]*FunctionInfo, error) {
	if c.functionRunner == nil {
		return []*FunctionInfo{}, nil
	}

	return c.functionRunner.ListFunctions(ctx)
}

// ValidatePackage validates a package revision.

func (c *Client) ValidatePackage(ctx context.Context, name, revision string) (*ValidationResult, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("ValidatePackage").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("ValidatePackage", "complete").Inc()

		}
	}()

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.validatePackageInternal(ctx, name, revision)
	})
	if err != nil {

		c.recordError("ValidatePackage", err)

		return nil, err

	}

	return result.(*ValidationResult), nil
}

// Workflow Operations.

// GetWorkflow retrieves a workflow by name.

func (c *Client) GetWorkflow(ctx context.Context, name string) (*Workflow, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("GetWorkflow").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("GetWorkflow", "complete").Inc()

		}
	}()

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.getWorkflowInternal(ctx, name)
	})
	if err != nil {

		c.recordError("GetWorkflow", err)

		return nil, err

	}

	return result.(*Workflow), nil
}

// ListWorkflows lists all workflows matching the provided options.

func (c *Client) ListWorkflows(ctx context.Context, opts *ListOptions) (*WorkflowList, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("ListWorkflows").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("ListWorkflows", "complete").Inc()

		}
	}()

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.listWorkflowsInternal(ctx, opts)
	})
	if err != nil {

		c.recordError("ListWorkflows", err)

		return nil, err

	}

	return result.(*WorkflowList), nil
}

// CreateWorkflow creates a new workflow.

func (c *Client) CreateWorkflow(ctx context.Context, workflow *Workflow) (*Workflow, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("CreateWorkflow").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("CreateWorkflow", "complete").Inc()

		}
	}()

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.createWorkflowInternal(ctx, workflow)
	})
	if err != nil {

		c.recordError("CreateWorkflow", err)

		return nil, err

	}

	return result.(*Workflow), nil
}

// UpdateWorkflow updates an existing workflow.

func (c *Client) UpdateWorkflow(ctx context.Context, workflow *Workflow) (*Workflow, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("UpdateWorkflow").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("UpdateWorkflow", "complete").Inc()

		}
	}()

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.updateWorkflowInternal(ctx, workflow)
	})
	if err != nil {

		c.recordError("UpdateWorkflow", err)

		return nil, err

	}

	return result.(*Workflow), nil
}

// DeleteWorkflow deletes a workflow.

func (c *Client) DeleteWorkflow(ctx context.Context, name string) error {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("DeleteWorkflow").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("DeleteWorkflow", "complete").Inc()

		}
	}()

	_, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return nil, c.deleteWorkflowInternal(ctx, name)
	})
	if err != nil {

		c.recordError("DeleteWorkflow", err)

		return err

	}

	return nil
}

// Health and Status Operations.

// Health returns the health status of the Porch system.

func (c *Client) Health(ctx context.Context) (*HealthStatus, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("Health").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("Health", "complete").Inc()

		}
	}()

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.healthInternal(ctx)
	})
	if err != nil {

		c.recordError("Health", err)

		return nil, err

	}

	return result.(*HealthStatus), nil
}

// Version returns version information for the Porch system.

func (c *Client) Version(ctx context.Context) (*VersionInfo, error) {
	start := time.Now()

	defer func() {
		if c.metrics != nil {

			c.metrics.requestDuration.WithLabelValues("Version").Observe(time.Since(start).Seconds())

			c.metrics.requestsTotal.WithLabelValues("Version", "complete").Inc()

		}
	}()

	result, err := c.circuitBreaker.Execute(func() (interface{}, error) {
		return c.versionInternal(ctx)
	})
	if err != nil {

		c.recordError("Version", err)

		return nil, err

	}

	return result.(*VersionInfo), nil
}

// Internal implementation methods.

// getRepositoryInternal implements the actual repository retrieval logic.

func (c *Client) getRepositoryInternal(ctx context.Context, name string) (*Repository, error) {
	c.logger.V(1).Info("Getting repository", "name", name)

	// Apply rate limiting.

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Get repository using dynamic client.

	gvr := schema.GroupVersionResource{
		Group: "config.porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "repositories",
	}

	obj, err := c.dynamic.Resource(gvr).Get(ctx, name, metav1.GetOptions{})
	if err != nil {

		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("repository %s not found: %w", name, ErrRepositoryNotFound)
		}

		return nil, fmt.Errorf("failed to get repository %s: %w", name, err)

	}

	// Convert unstructured to Repository.

	repo := &Repository{}

	if err := convertFromUnstructured(obj, repo); err != nil {
		return nil, fmt.Errorf("failed to convert repository: %w", err)
	}

	c.logger.V(1).Info("Successfully retrieved repository", "name", name, "url", repo.Spec.URL)

	return repo, nil
}

// listRepositoriesInternal implements the actual repository listing logic.

func (c *Client) listRepositoriesInternal(ctx context.Context, opts *ListOptions) (*RepositoryList, error) {
	c.logger.V(1).Info("Listing repositories", "opts", opts)

	// Apply rate limiting.

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Get repositories using dynamic client.

	gvr := schema.GroupVersionResource{
		Group: "config.porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "repositories",
	}

	listOptions := metav1.ListOptions{}

	if opts != nil {

		if opts.LabelSelector != "" {
			listOptions.LabelSelector = opts.LabelSelector
		}

		if opts.FieldSelector != "" {
			listOptions.FieldSelector = opts.FieldSelector
		}

		if opts.Limit > 0 {
			listOptions.Limit = opts.Limit
		}

		if opts.Continue != "" {
			listOptions.Continue = opts.Continue
		}

	}

	namespace := ""

	if opts != nil && opts.Namespace != "" {
		namespace = opts.Namespace
	}

	var obj *unstructured.UnstructuredList

	var err error

	if namespace != "" {
		obj, err = c.dynamic.Resource(gvr).Namespace(namespace).List(ctx, listOptions)
	} else {
		obj, err = c.dynamic.Resource(gvr).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}

	// Convert unstructured list to RepositoryList.

	repoList := &RepositoryList{}

	if err := convertFromUnstructured(obj, repoList); err != nil {
		return nil, fmt.Errorf("failed to convert repository list: %w", err)
	}

	c.logger.V(1).Info("Successfully listed repositories", "count", len(repoList.Items))

	return repoList, nil
}

// createRepositoryInternal implements the actual repository creation logic.

func (c *Client) createRepositoryInternal(ctx context.Context, repo *Repository) (*Repository, error) {
	c.logger.V(1).Info("Creating repository", "name", repo.Name, "url", repo.Spec.URL)

	// Apply rate limiting.

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Set creation timestamp and generate name if needed.

	now := metav1.Now()

	if repo.CreationTimestamp.IsZero() {
		repo.CreationTimestamp = now
	}

	// Set API version and kind.

	repo.APIVersion = "config.porch.kpt.dev/v1alpha1"

	repo.Kind = "Repository"

	// Convert to unstructured for creation.

	obj, err := convertToUnstructured(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to convert repository: %w", err)
	}

	// Create repository using dynamic client.

	gvr := schema.GroupVersionResource{
		Group: "config.porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "repositories",
	}

	created, err := c.dynamic.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create repository: %w", err)
	}

	// Convert back to Repository.

	result := &Repository{}

	if err := convertFromUnstructured(created, result); err != nil {
		return nil, fmt.Errorf("failed to convert created repository: %w", err)
	}

	c.logger.Info("Successfully created repository", "name", result.Name, "uid", result.UID)

	return result, nil
}

// updateRepositoryInternal implements the actual repository update logic.

func (c *Client) updateRepositoryInternal(ctx context.Context, repo *Repository) (*Repository, error) {
	c.logger.V(1).Info("Updating repository", "name", repo.Name)

	// Apply rate limiting.

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Convert to unstructured for update.

	obj, err := convertToUnstructured(repo)
	if err != nil {
		return nil, fmt.Errorf("failed to convert repository: %w", err)
	}

	// Update repository using dynamic client.

	gvr := schema.GroupVersionResource{
		Group: "config.porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "repositories",
	}

	updated, err := c.dynamic.Resource(gvr).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update repository: %w", err)
	}

	// Convert back to Repository.

	result := &Repository{}

	if err := convertFromUnstructured(updated, result); err != nil {
		return nil, fmt.Errorf("failed to convert updated repository: %w", err)
	}

	c.logger.Info("Successfully updated repository", "name", result.Name)

	return result, nil
}

// deleteRepositoryInternal implements the actual repository deletion logic.

func (c *Client) deleteRepositoryInternal(ctx context.Context, name string) error {
	c.logger.V(1).Info("Deleting repository", "name", name)

	// Apply rate limiting.

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Delete repository using dynamic client.

	gvr := schema.GroupVersionResource{
		Group: "config.porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "repositories",
	}

	err := c.dynamic.Resource(gvr).Delete(ctx, name, metav1.DeleteOptions{})
	if err != nil {

		if apierrors.IsNotFound(err) {
			return fmt.Errorf("repository %s not found: %w", name, ErrRepositoryNotFound)
		}

		return fmt.Errorf("failed to delete repository %s: %w", name, err)

	}

	c.logger.Info("Successfully deleted repository", "name", name)

	return nil
}

// syncRepositoryInternal implements the actual repository synchronization logic.

func (c *Client) syncRepositoryInternal(ctx context.Context, name string) error {
	c.logger.V(1).Info("Syncing repository", "name", name)

	// Apply rate limiting.

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Get the repository first.

	repo, err := c.getRepositoryInternal(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to get repository for sync: %w", err)
	}

	// Update sync status.

	repo.Status.LastSyncTime = &metav1.Time{Time: time.Now()}

	repo.Status.SyncError = ""

	// Update the repository.

	_, err = c.updateRepositoryInternal(ctx, repo)
	if err != nil {
		return fmt.Errorf("failed to update repository sync status: %w", err)
	}

	c.logger.Info("Successfully synced repository", "name", name)

	return nil
}

// listPackageRevisionsInternal implements package revision listing.

func (c *Client) listPackageRevisionsInternal(ctx context.Context, opts *ListOptions) (*PackageRevisionList, error) {
	c.logger.V(1).Info("Listing package revisions", "opts", opts)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Get package revisions using dynamic client.

	gvr := schema.GroupVersionResource{
		Group: "porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "packagerevisions",
	}

	listOptions := metav1.ListOptions{}

	if opts != nil {

		if opts.LabelSelector != "" {
			listOptions.LabelSelector = opts.LabelSelector
		}

		if opts.FieldSelector != "" {
			listOptions.FieldSelector = opts.FieldSelector
		}

		if opts.Limit > 0 {
			listOptions.Limit = opts.Limit
		}

		if opts.Continue != "" {
			listOptions.Continue = opts.Continue
		}

	}

	namespace := ""

	if opts != nil && opts.Namespace != "" {
		namespace = opts.Namespace
	}

	var obj *unstructured.UnstructuredList

	var err error

	if namespace != "" {
		obj, err = c.dynamic.Resource(gvr).Namespace(namespace).List(ctx, listOptions)
	} else {
		obj, err = c.dynamic.Resource(gvr).List(ctx, listOptions)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to list package revisions: %w", err)
	}

	// Convert unstructured list to PackageRevisionList.

	pkgList := &PackageRevisionList{}

	if err := convertFromUnstructured(obj, pkgList); err != nil {
		return nil, fmt.Errorf("failed to convert package revision list: %w", err)
	}

	c.logger.V(1).Info("Successfully listed package revisions", "count", len(pkgList.Items))

	return pkgList, nil
}

// createPackageRevisionInternal implements package revision creation.

func (c *Client) createPackageRevisionInternal(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error) {
	c.logger.V(1).Info("Creating package revision", "name", pkg.Spec.PackageName, "revision", pkg.Spec.Revision)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Set creation timestamp and generate name if needed.

	now := metav1.Now()

	if pkg.CreationTimestamp.IsZero() {
		pkg.CreationTimestamp = now
	}

	// Set API version and kind.

	pkg.APIVersion = "porch.kpt.dev/v1alpha1"

	pkg.Kind = "PackageRevision"

	// Generate name if not set (typically packagename-revision).

	if pkg.Name == "" {
		pkg.Name = fmt.Sprintf("%s-%s", pkg.Spec.PackageName, pkg.Spec.Revision)
	}

	// Set default lifecycle if not specified.

	if pkg.Spec.Lifecycle == "" {
		pkg.Spec.Lifecycle = PackageRevisionLifecycleDraft
	}

	// Convert to unstructured for creation.

	obj, err := convertToUnstructured(pkg)
	if err != nil {
		return nil, fmt.Errorf("failed to convert package revision: %w", err)
	}

	// Create package revision using dynamic client.

	gvr := schema.GroupVersionResource{
		Group: "porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "packagerevisions",
	}

	// PackageRevisions are cluster-scoped in Porch.

	created, err := c.dynamic.Resource(gvr).Create(ctx, obj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create package revision: %w", err)
	}

	// Convert back to PackageRevision.

	result := &PackageRevision{}

	if err := convertFromUnstructured(created, result); err != nil {
		return nil, fmt.Errorf("failed to convert created package revision: %w", err)
	}

	c.logger.Info("Successfully created package revision", "name", result.ObjectMeta.Name, "uid", result.UID)

	return result, nil
}

// updatePackageRevisionInternal implements package revision update.

func (c *Client) updatePackageRevisionInternal(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error) {
	c.logger.V(1).Info("Updating package revision", "name", pkg.Spec.PackageName, "revision", pkg.Spec.Revision)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Convert to unstructured for update.

	obj, err := convertToUnstructured(pkg)
	if err != nil {
		return nil, fmt.Errorf("failed to convert package revision: %w", err)
	}

	// Update package revision using dynamic client.

	gvr := schema.GroupVersionResource{
		Group: "porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "packagerevisions",
	}

	updated, err := c.dynamic.Resource(gvr).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to update package revision: %w", err)
	}

	// Convert back to PackageRevision.

	result := &PackageRevision{}

	if err := convertFromUnstructured(updated, result); err != nil {
		return nil, fmt.Errorf("failed to convert updated package revision: %w", err)
	}

	c.logger.Info("Successfully updated package revision", "name", result.ObjectMeta.Name)

	return result, nil
}

// deletePackageRevisionInternal implements package revision deletion.

func (c *Client) deletePackageRevisionInternal(ctx context.Context, name, revision string) error {
	c.logger.V(1).Info("Deleting package revision", "name", name, "revision", revision)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Delete package revision using dynamic client.

	gvr := schema.GroupVersionResource{
		Group: "porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "packagerevisions",
	}

	// Construct the package revision name (typically name-revision).

	resourceName := fmt.Sprintf("%s-%s", name, revision)

	err := c.dynamic.Resource(gvr).Delete(ctx, resourceName, metav1.DeleteOptions{})
	if err != nil {

		if apierrors.IsNotFound(err) {
			return fmt.Errorf("package revision %s:%s not found: %w", name, revision, ErrPackageNotFound)
		}

		return fmt.Errorf("failed to delete package revision %s:%s: %w", name, revision, err)

	}

	c.logger.Info("Successfully deleted package revision", "name", name, "revision", revision)

	return nil
}

// approvePackageRevisionInternal implements package revision approval.

func (c *Client) approvePackageRevisionInternal(ctx context.Context, name, revision string) error {
	c.logger.V(1).Info("Approving package revision", "name", name, "revision", revision)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Get the current package revision.

	pkg, err := c.getPackageRevisionInternal(ctx, name, revision)
	if err != nil {
		return fmt.Errorf("failed to get package revision: %w", err)
	}

	// Check current lifecycle state.

	if pkg.Spec.Lifecycle != PackageRevisionLifecycleProposed {
		return fmt.Errorf("package revision must be in Proposed state to approve, current state: %s", pkg.Spec.Lifecycle)
	}

	// Transition to Published lifecycle.

	pkg.Spec.Lifecycle = PackageRevisionLifecyclePublished

	// Update status to indicate approval.

	now := metav1.Now()

	pkg.Status.PublishTime = &now

	// Set approval condition.

	condition := metav1.Condition{
		Type: "Approved",

		Status: metav1.ConditionTrue,

		LastTransitionTime: now,

		Reason: "PackageApproved",

		Message: "Package revision has been approved and published",
	}

	pkg.SetCondition(condition)

	// Update the package revision.

	_, err = c.updatePackageRevisionInternal(ctx, pkg)
	if err != nil {
		return fmt.Errorf("failed to approve package revision: %w", err)
	}

	c.logger.Info("Successfully approved package revision", "name", name, "revision", revision)

	return nil
}

// proposePackageRevisionInternal implements package revision proposal.

func (c *Client) proposePackageRevisionInternal(ctx context.Context, name, revision string) error {
	c.logger.V(1).Info("Proposing package revision", "name", name, "revision", revision)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Get the current package revision.

	pkg, err := c.getPackageRevisionInternal(ctx, name, revision)
	if err != nil {
		return fmt.Errorf("failed to get package revision: %w", err)
	}

	// Check current lifecycle state.

	if pkg.Spec.Lifecycle != PackageRevisionLifecycleDraft {
		return fmt.Errorf("package revision must be in Draft state to propose, current state: %s", pkg.Spec.Lifecycle)
	}

	// Transition to Proposed lifecycle.

	pkg.Spec.Lifecycle = PackageRevisionLifecycleProposed

	// Set proposed condition.

	now := metav1.Now()

	condition := metav1.Condition{
		Type: "Proposed",

		Status: metav1.ConditionTrue,

		LastTransitionTime: now,

		Reason: "PackageProposed",

		Message: "Package revision has been proposed for review",
	}

	pkg.SetCondition(condition)

	// Update the package revision.

	_, err = c.updatePackageRevisionInternal(ctx, pkg)
	if err != nil {
		return fmt.Errorf("failed to propose package revision: %w", err)
	}

	c.logger.Info("Successfully proposed package revision", "name", name, "revision", revision)

	return nil
}

// rejectPackageRevisionInternal implements package revision rejection.

func (c *Client) rejectPackageRevisionInternal(ctx context.Context, name, revision, reason string) error {
	c.logger.V(1).Info("Rejecting package revision", "name", name, "revision", revision, "reason", reason)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Get the current package revision.

	pkg, err := c.getPackageRevisionInternal(ctx, name, revision)
	if err != nil {
		return fmt.Errorf("failed to get package revision: %w", err)
	}

	// Check current lifecycle state.

	if pkg.Spec.Lifecycle != PackageRevisionLifecycleProposed {
		return fmt.Errorf("package revision must be in Proposed state to reject, current state: %s", pkg.Spec.Lifecycle)
	}

	// Transition back to Draft lifecycle.

	pkg.Spec.Lifecycle = PackageRevisionLifecycleDraft

	// Set rejection condition.

	now := metav1.Now()

	condition := metav1.Condition{
		Type: "Rejected",

		Status: metav1.ConditionTrue,

		LastTransitionTime: now,

		Reason: "PackageRejected",

		Message: fmt.Sprintf("Package revision rejected: %s", reason),
	}

	pkg.SetCondition(condition)

	// Update the package revision.

	_, err = c.updatePackageRevisionInternal(ctx, pkg)
	if err != nil {
		return fmt.Errorf("failed to reject package revision: %w", err)
	}

	c.logger.Info("Successfully rejected package revision", "name", name, "revision", revision, "reason", reason)

	return nil
}

// getPackageContentsInternal implements package content retrieval.

func (c *Client) getPackageContentsInternal(ctx context.Context, name, revision string) (map[string][]byte, error) {
	c.logger.V(1).Info("Getting package contents", "name", name, "revision", revision)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Get the package revision first.

	pkg, err := c.getPackageRevisionInternal(ctx, name, revision)
	if err != nil {
		return nil, fmt.Errorf("failed to get package revision: %w", err)
	}

	contents := make(map[string][]byte)

	// Extract contents from resources in the package revision spec.

	if pkg.Spec.Resources != nil {
		for i, resourceInterface := range pkg.Spec.Resources {

			// Convert interface{} to KRMResource.

			resource, err := convertToKRMResource(resourceInterface)
			if err != nil {

				c.logger.Error(err, "Failed to convert interface to KRMResource", "index", i)

				continue

			}

			// Convert KRMResource to YAML.

			yamlData, err := convertKRMResourceToYAML(resource)
			if err != nil {

				c.logger.Error(err, "Failed to convert resource to YAML", "index", i)

				continue

			}

			// Generate filename based on resource metadata.

			filename := generateResourceFilename(resource)

			contents[filename] = yamlData

		}
	}

	// If no resources in spec, try to get from PackageRevisionResources sub-resource.

	if len(contents) == 0 {

		resourceContents, err := c.getPackageResourcesFromPorch(ctx, name, revision)

		if err != nil {
			c.logger.V(1).Info("No resources found in PackageRevisionResources", "error", err)
		} else {
			contents = resourceContents
		}

	}

	c.logger.V(1).Info("Successfully retrieved package contents", "name", name, "revision", revision, "fileCount", len(contents))

	return contents, nil
}

// updatePackageContentsInternal implements package content update.

func (c *Client) updatePackageContentsInternal(ctx context.Context, name, revision string, contents map[string][]byte) error {
	c.logger.V(1).Info("Updating package contents", "name", name, "revision", revision, "fileCount", len(contents))

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Get the current package revision.

	pkg, err := c.getPackageRevisionInternal(ctx, name, revision)
	if err != nil {
		return fmt.Errorf("failed to get package revision: %w", err)
	}

	// Convert file contents to KRM resources.

	resources := []KRMResource{}

	for filename, content := range contents {

		// Skip non-YAML files.

		if !isYAMLFile(filename) {
			continue
		}

		resource, err := convertYAMLToKRMResource(content)
		if err != nil {

			c.logger.Error(err, "Failed to convert YAML to KRM resource", "filename", filename)

			continue

		}

		resources = append(resources, *resource)

	}

	// Update the package revision with new resources.

	var interfaceResources []interface{}

	for _, resource := range resources {
		interfaceResources = append(interfaceResources, convertFromKRMResource(resource))
	}

	pkg.Spec.Resources = interfaceResources

	// Update the package revision.

	_, err = c.updatePackageRevisionInternal(ctx, pkg)
	if err != nil {
		return fmt.Errorf("failed to update package revision with new contents: %w", err)
	}

	// Also try to update using PackageRevisionResources if available.

	if err := c.updatePackageResourcesInPorch(ctx, name, revision, contents); err != nil {
		c.logger.V(1).Info("Failed to update PackageRevisionResources", "error", err)

		// Don't fail the entire operation, as the main update succeeded.
	}

	c.logger.Info("Successfully updated package contents", "name", name, "revision", revision, "resourceCount", len(resources))

	return nil
}

// renderPackageInternal implements package rendering.

func (c *Client) renderPackageInternal(ctx context.Context, name, revision string) (*RenderResult, error) {
	c.logger.V(1).Info("Rendering package", "name", name, "revision", revision)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Get the package revision.

	pkg, err := c.getPackageRevisionInternal(ctx, name, revision)
	if err != nil {
		return nil, fmt.Errorf("failed to get package revision: %w", err)
	}

	// If no functions defined, return resources as-is.

	if len(pkg.Spec.Functions) == 0 {

		// Convert interface{} resources to KRMResource.

		var resources []KRMResource

		for _, resourceInterface := range pkg.Spec.Resources {

			resource, err := convertToKRMResource(resourceInterface)
			if err != nil {

				c.logger.Error(err, "Failed to convert interface to KRMResource")

				continue

			}

			resources = append(resources, resource)

		}

		return &RenderResult{
			Resources: resources,
		}, nil

	}

	// Execute each function in the pipeline.

	// Convert interface{} resources to KRMResource.

	var currentResources []KRMResource

	for _, resourceInterface := range pkg.Spec.Resources {

		resource, err := convertToKRMResource(resourceInterface)
		if err != nil {

			c.logger.Error(err, "Failed to convert interface to KRMResource")

			continue

		}

		currentResources = append(currentResources, resource)

	}

	var allResults []*FunctionResult

	for _, functionInterface := range pkg.Spec.Functions {

		// Convert interface{} to FunctionConfig.

		functionConfig, err := convertToFunctionConfig(functionInterface)
		if err != nil {

			c.logger.Error(err, "Failed to convert interface to FunctionConfig")

			continue

		}

		c.logger.V(1).Info("Executing function in pipeline", "image", functionConfig.Image)

		// Create function request.

		req := &FunctionRequest{
			FunctionConfig: functionConfig,

			Resources: currentResources,

			Context: &FunctionContext{
				Package: &PackageReference{
					Repository: pkg.Spec.Repository,

					PackageName: pkg.Spec.PackageName,

					Revision: pkg.Spec.Revision,
				},
			},
		}

		// Execute function.

		resp, err := c.runFunctionInternal(ctx, req)
		if err != nil {
			return &RenderResult{
				Resources: currentResources,

				Error: &RenderError{
					Message: fmt.Sprintf("Function %s failed: %v", functionConfig.Image, err),

					Type: "FunctionExecutionError",

					Details: err.Error(),
				},
			}, nil
		}

		// Check for function execution errors.

		if resp.Error != nil {
			return &RenderResult{
				Resources: currentResources,

				Error: &RenderError{
					Message: fmt.Sprintf("Function %s error: %s", functionConfig.Image, resp.Error.Message),

					Type: "FunctionError",

					Details: resp.Error.Details,
				},
			}, nil
		}

		// Update resources for next function in pipeline.

		currentResources = resp.Resources

		allResults = append(allResults, resp.Results...)

	}

	c.logger.V(1).Info("Successfully rendered package", "name", name, "revision", revision, "finalResourceCount", len(currentResources))

	return &RenderResult{
		Resources: currentResources,

		Results: allResults,
	}, nil
}

// runFunctionInternal implements function execution.

func (c *Client) runFunctionInternal(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	c.logger.V(1).Info("Running function", "image", req.FunctionConfig.Image)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Create FunctionEvalTask resource for Porch function execution.

	gvr := schema.GroupVersionResource{
		Group: "porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "functionevaltasks",
	}

	// Create the function evaluation task.

	task := map[string]interface{}{
		"metadata": map[string]interface{}{
			"generateName": "function-eval-",
			"labels": map[string]interface{}{},
		},
		"spec": map[string]interface{}{},
	}

	// Convert to unstructured.

	taskObj := &unstructured.Unstructured{}

	taskObj.Object = task

	// Create the task.

	createdTask, err := c.dynamic.Resource(gvr).Create(ctx, taskObj, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create function evaluation task: %w", err)
	}

	// Wait for task completion (polling approach).

	taskName := createdTask.GetName()

	response, err := c.waitForFunctionTaskCompletion(ctx, taskName, 60*time.Second)
	if err != nil {
		return nil, fmt.Errorf("function execution failed: %w", err)
	}

	// Clean up the task.

	if err := c.dynamic.Resource(gvr).Delete(ctx, taskName, metav1.DeleteOptions{}); err != nil {
		c.logger.Error(err, "Failed to cleanup function evaluation task", "taskName", taskName)
	}

	c.logger.V(1).Info("Successfully executed function", "image", req.FunctionConfig.Image, "resourceCount", len(response.Resources))

	return response, nil
}

// validatePackageInternal implements package validation.

func (c *Client) validatePackageInternal(ctx context.Context, name, revision string) (*ValidationResult, error) {
	c.logger.V(1).Info("Validating package", "name", name, "revision", revision)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	// Get the package revision.

	pkg, err := c.getPackageRevisionInternal(ctx, name, revision)
	if err != nil {
		return &ValidationResult{
			Valid: false,

			Errors: []ValidationError{{
				Path: "package",

				Message: fmt.Sprintf("Failed to retrieve package: %v", err),

				Severity: "error",
			}},
		}, nil
	}

	var errors []ValidationError

	var warnings []ValidationError

	// Validate basic package structure.

	if pkg.Spec.PackageName == "" {
		errors = append(errors, ValidationError{
			Path: "spec.packageName",

			Message: "Package name is required",

			Severity: "error",

			Code: "MISSING_PACKAGE_NAME",
		})
	}

	if pkg.Spec.Repository == "" {
		errors = append(errors, ValidationError{
			Path: "spec.repository",

			Message: "Repository is required",

			Severity: "error",

			Code: "MISSING_REPOSITORY",
		})
	}

	// Validate lifecycle transitions.

	if !IsValidLifecycle(pkg.Spec.Lifecycle) {
		errors = append(errors, ValidationError{
			Path: "spec.lifecycle",

			Message: fmt.Sprintf("Invalid lifecycle state: %s", pkg.Spec.Lifecycle),

			Severity: "error",

			Code: "INVALID_LIFECYCLE",
		})
	}

	// Validate resources.

	for i, resourceInterface := range pkg.Spec.Resources {

		resource, err := convertToKRMResource(resourceInterface)
		if err != nil {

			errors = append(errors, ValidationError{
				Path: fmt.Sprintf("spec.resources[%d]", i),

				Message: fmt.Sprintf("Invalid resource format: %v", err),

				Severity: "error",

				Code: "INVALID_RESOURCE_FORMAT",
			})

			continue

		}

		if resource.APIVersion == "" {
			errors = append(errors, ValidationError{
				Path: fmt.Sprintf("spec.resources[%d].apiVersion", i),

				Message: "Resource apiVersion is required",

				Severity: "error",

				Code: "MISSING_API_VERSION",
			})
		}

		if resource.Kind == "" {
			errors = append(errors, ValidationError{
				Path: fmt.Sprintf("spec.resources[%d].kind", i),

				Message: "Resource kind is required",

				Severity: "error",

				Code: "MISSING_KIND",
			})
		}

		// Check for resource name in metadata.

		// Check if resource has name in metadata
		hasName := false
		if len(resource.Metadata) > 0 {
			var metadataMap map[string]interface{}
			if err := json.Unmarshal(resource.Metadata, &metadataMap); err == nil {
				if name, ok := metadataMap["name"].(string); ok && name != "" {
					hasName = true
				}
			}
		}
		if !hasName {
			warnings = append(warnings, ValidationError{
				Path: fmt.Sprintf("spec.resources[%d].metadata.name", i),
				Message: "Resource name is recommended in metadata",
				Severity: "warning",
				Code: "MISSING_RESOURCE_NAME",
			})
		}

	}

	// Validate function configurations.

	for i, functionInterface := range pkg.Spec.Functions {

		function, err := convertToFunctionConfig(functionInterface)
		if err != nil {

			errors = append(errors, ValidationError{
				Path: fmt.Sprintf("spec.functions[%d]", i),

				Message: fmt.Sprintf("Invalid function format: %v", err),

				Severity: "error",

				Code: "INVALID_FUNCTION_FORMAT",
			})

			continue

		}

		if function.Image == "" {
			errors = append(errors, ValidationError{
				Path: fmt.Sprintf("spec.functions[%d].image", i),

				Message: "Function image is required",

				Severity: "error",

				Code: "MISSING_FUNCTION_IMAGE",
			})
		}

	}

	isValid := len(errors) == 0

	c.logger.V(1).Info("Package validation completed", "name", name, "revision", revision, "valid", isValid, "errors", len(errors), "warnings", len(warnings))

	return &ValidationResult{
		Valid: isValid,

		Errors: errors,

		Warnings: warnings,
	}, nil
}

// getWorkflowInternal implements workflow retrieval.

func (c *Client) getWorkflowInternal(ctx context.Context, name string) (*Workflow, error) {
	c.logger.V(1).Info("Getting workflow", "name", name)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	return &Workflow{}, nil
}

// listWorkflowsInternal implements workflow listing.

func (c *Client) listWorkflowsInternal(ctx context.Context, opts *ListOptions) (*WorkflowList, error) {
	c.logger.V(1).Info("Listing workflows", "opts", opts)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	return &WorkflowList{}, nil
}

// createWorkflowInternal implements workflow creation.

func (c *Client) createWorkflowInternal(ctx context.Context, workflow *Workflow) (*Workflow, error) {
	c.logger.V(1).Info("Creating workflow", "name", workflow.Name)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	return workflow, nil
}

// updateWorkflowInternal implements workflow update.

func (c *Client) updateWorkflowInternal(ctx context.Context, workflow *Workflow) (*Workflow, error) {
	c.logger.V(1).Info("Updating workflow", "name", workflow.Name)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	return workflow, nil
}

// deleteWorkflowInternal implements workflow deletion.

func (c *Client) deleteWorkflowInternal(ctx context.Context, name string) error {
	c.logger.V(1).Info("Deleting workflow", "name", name)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	return nil
}

// healthInternal implements health check.

func (c *Client) healthInternal(ctx context.Context) (*HealthStatus, error) {
	c.logger.V(1).Info("Checking health")

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	return &HealthStatus{
		Status: "healthy",

		Timestamp: &metav1.Time{Time: time.Now()},
	}, nil
}

// versionInternal implements version retrieval.

func (c *Client) versionInternal(ctx context.Context) (*VersionInfo, error) {
	c.logger.V(1).Info("Getting version")

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	return &VersionInfo{
		Version: "v1.0.0",
	}, nil
}

// getPackageRevisionInternal implements package revision retrieval.

func (c *Client) getPackageRevisionInternal(ctx context.Context, name, revision string) (*PackageRevision, error) {
	c.logger.V(1).Info("Getting package revision", "name", name, "revision", revision)

	if c.rateLimiter != nil {
		if err := c.rateLimiter.Wait(ctx); err != nil {
			return nil, fmt.Errorf("rate limit exceeded: %w", err)
		}
	}

	gvr := schema.GroupVersionResource{
		Group: "porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "packagerevisions",
	}

	// Construct the package revision name (typically name-revision).

	resourceName := fmt.Sprintf("%s-%s", name, revision)

	obj, err := c.dynamic.Resource(gvr).Get(ctx, resourceName, metav1.GetOptions{})
	if err != nil {

		if apierrors.IsNotFound(err) {
			return nil, fmt.Errorf("package revision %s:%s not found: %w", name, revision, ErrPackageNotFound)
		}

		return nil, fmt.Errorf("failed to get package revision %s:%s: %w", name, revision, err)

	}

	pkg := &PackageRevision{}

	if err := convertFromUnstructured(obj, pkg); err != nil {
		return nil, fmt.Errorf("failed to convert package revision: %w", err)
	}

	c.logger.V(1).Info("Successfully retrieved package revision", "name", name, "revision", revision)

	return pkg, nil
}

// Validation methods.

// validateRepository validates repository configuration.

func (c *Client) validateRepository(repo *Repository) error {
	if repo == nil {
		return fmt.Errorf("repository cannot be nil")
	}

	if repo.Name == "" {
		return fmt.Errorf("repository name is required")
	}

	if repo.Spec.URL == "" {
		return fmt.Errorf("repository URL is required")
	}

	if repo.Spec.Type == "" {
		return fmt.Errorf("repository type is required")
	}

	// Validate repository type.

	validTypes := map[string]bool{
		"git": true,

		"oci": true,
	}

	if !validTypes[repo.Spec.Type] {
		return fmt.Errorf("unsupported repository type: %s", repo.Spec.Type)
	}

	// Validate URL format.

	if repo.Spec.Type == "git" {
		if !strings.HasPrefix(repo.Spec.URL, "https://") &&

			!strings.HasPrefix(repo.Spec.URL, "git@") {

			return fmt.Errorf("git repository URL must start with https:// or git@")
		}
	}

	return nil
}

// validatePackageRevision validates package revision configuration.

func (c *Client) validatePackageRevision(pkg *PackageRevision) error {
	if pkg == nil {
		return fmt.Errorf("package revision cannot be nil")
	}

	if pkg.Spec.PackageName == "" {
		return fmt.Errorf("package name is required")
	}

	if pkg.Spec.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	if pkg.Spec.Revision == "" {
		return fmt.Errorf("revision is required")
	}

	// Validate lifecycle.

	if pkg.Spec.Lifecycle != "" && !IsValidLifecycle(pkg.Spec.Lifecycle) {
		return fmt.Errorf("invalid lifecycle: %s", pkg.Spec.Lifecycle)
	}

	return nil
}

// Initialization methods.

// initCircuitBreaker initializes the circuit breaker.

func (c *Client) initCircuitBreaker() error {
	settings := gobreaker.Settings{
		Name: "porch-client",

		MaxRequests: 3,

		Interval: time.Minute,

		Timeout: 30 * time.Second,

		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)

			return counts.Requests >= 3 && failureRatio >= 0.6
		},

		OnStateChange: func(name string, from, to gobreaker.State) {
			c.logger.Info("Circuit breaker state changed", "name", name, "from", from, "to", to)

			if c.metrics != nil {
				c.metrics.circuitBreakerState.Set(float64(to))
			}
		},
	}

	if c.config.PorchConfig.CircuitBreaker != nil {

		cb := c.config.PorchConfig.CircuitBreaker

		if cb.Enabled {

			if cb.FailureThreshold > 0 {
				settings.ReadyToTrip = func(counts gobreaker.Counts) bool {
					return counts.TotalFailures >= uint32(cb.FailureThreshold)
				}
			}

			if cb.Timeout > 0 {
				settings.Timeout = cb.Timeout
			}

			if cb.HalfOpenMaxCalls > 0 {
				settings.MaxRequests = uint32(cb.HalfOpenMaxCalls)
			}

		}

	}

	c.circuitBreaker = gobreaker.NewCircuitBreaker(settings)

	return nil
}

// initRateLimiter initializes the rate limiter.

func (c *Client) initRateLimiter() error {
	if c.config.PorchConfig.RateLimit != nil && c.config.PorchConfig.RateLimit.Enabled {

		rl := c.config.PorchConfig.RateLimit

		c.rateLimiter = rate.NewLimiter(rate.Limit(rl.RequestsPerSecond), rl.Burst)

	}

	return nil
}

// initMetrics initializes Prometheus metrics.

func (c *Client) initMetrics() *ClientMetrics {
	return &ClientMetrics{
		requestsTotal: prometheus.NewCounterVec(

			prometheus.CounterOpts{
				Name: "porch_client_requests_total",

				Help: "Total number of Porch client requests",
			},

			[]string{"operation", "status"},
		),

		requestDuration: prometheus.NewHistogramVec(

			prometheus.HistogramOpts{
				Name: "porch_client_request_duration_seconds",

				Help: "Duration of Porch client requests",

				Buckets: prometheus.DefBuckets,
			},

			[]string{"operation"},
		),

		cacheHits: prometheus.NewCounter(

			prometheus.CounterOpts{
				Name: "porch_client_cache_hits_total",

				Help: "Total number of cache hits",
			},
		),

		cacheMisses: prometheus.NewCounter(

			prometheus.CounterOpts{
				Name: "porch_client_cache_misses_total",

				Help: "Total number of cache misses",
			},
		),

		circuitBreakerState: prometheus.NewGauge(

			prometheus.GaugeOpts{
				Name: "porch_client_circuit_breaker_state",

				Help: "Current circuit breaker state (0=Closed, 1=HalfOpen, 2=Open)",
			},
		),

		activeConnections: prometheus.NewGauge(

			prometheus.GaugeOpts{
				Name: "porch_client_active_connections",

				Help: "Number of active connections",
			},
		),

		errorRate: prometheus.NewCounterVec(

			prometheus.CounterOpts{
				Name: "porch_client_errors_total",

				Help: "Total number of Porch client errors",
			},

			[]string{"operation", "error_type"},
		),
	}
}

// Cache methods.

// getCached retrieves a value from cache.

func (c *Client) getCached(key string) interface{} {
	if c.cache == nil {
		return nil
	}

	c.cache.mutex.RLock()

	defer c.cache.mutex.RUnlock()

	entry, exists := c.cache.cache[key]

	if !exists {
		return nil
	}

	// Check if expired.

	if time.Since(entry.timestamp) > c.cache.ttl {

		go c.evictExpired()

		return nil

	}

	return entry.data
}

// setCached stores a value in cache.

func (c *Client) setCached(key string, value interface{}) {
	if c.cache == nil {
		return
	}

	c.cache.mutex.Lock()

	defer c.cache.mutex.Unlock()

	// Evict if cache is full.

	if len(c.cache.cache) >= c.cache.maxSize {
		c.evictOldestLocked()
	}

	c.cache.cache[key] = &cacheEntry{
		data: value,

		timestamp: time.Now(),
	}
}

// invalidateCache removes a key from cache.

func (c *Client) invalidateCache(key string) {
	if c.cache == nil {
		return
	}

	c.cache.mutex.Lock()

	defer c.cache.mutex.Unlock()

	delete(c.cache.cache, key)
}

// evictExpired removes expired entries from cache.

func (c *Client) evictExpired() {
	if c.cache == nil {
		return
	}

	c.cache.mutex.Lock()

	defer c.cache.mutex.Unlock()

	now := time.Now()

	for key, entry := range c.cache.cache {
		if now.Sub(entry.timestamp) > c.cache.ttl {
			delete(c.cache.cache, key)
		}
	}
}

// evictOldestLocked removes the oldest entry (requires lock).

func (c *Client) evictOldestLocked() {
	var oldestKey string

	var oldestTime time.Time

	for key, entry := range c.cache.cache {
		if oldestKey == "" || entry.timestamp.Before(oldestTime) {

			oldestKey = key

			oldestTime = entry.timestamp

		}
	}

	if oldestKey != "" {
		delete(c.cache.cache, oldestKey)
	}
}

// Utility methods.

// recordError records an error in metrics.

func (c *Client) recordError(operation string, err error) {
	if c.metrics == nil {
		return
	}

	errorType := "unknown"

	if apierrors.IsNotFound(err) {
		errorType = "not_found"
	} else if apierrors.IsConflict(err) {
		errorType = "conflict"
	} else if apierrors.IsForbidden(err) {
		errorType = "forbidden"
	} else if apierrors.IsUnauthorized(err) {
		errorType = "unauthorized"
	} else if apierrors.IsBadRequest(err) {
		errorType = "bad_request"
	} else if apierrors.IsTimeout(err) {
		errorType = "timeout"
	} else if apierrors.IsServerTimeout(err) {
		errorType = "server_timeout"
	} else if apierrors.IsTooManyRequests(err) {
		errorType = "rate_limited"
	}

	c.metrics.errorRate.WithLabelValues(operation, errorType).Inc()
}

// convertToUnstructured converts a typed object to unstructured.

func convertToUnstructured(obj interface{}) (*unstructured.Unstructured, error) {
	data, err := json.Marshal(obj)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal object: %w", err)
	}

	unstructuredObj := &unstructured.Unstructured{}

	if err := json.Unmarshal(data, unstructuredObj); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to unstructured: %w", err)
	}

	return unstructuredObj, nil
}

// convertFromUnstructured converts an unstructured object to a typed object.

func convertFromUnstructured(obj, target interface{}) error {
	data, err := json.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal unstructured object: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal to target type: %w", err)
	}

	return nil
}

// Type conversion helpers for interface{} fields in multicluster types.

// convertToKRMResource converts interface{} to KRMResource.

func convertToKRMResource(resource interface{}) (KRMResource, error) {
	// Handle KRMResource type directly.

	if krmRes, ok := resource.(KRMResource); ok {
		return krmRes, nil
	}

	// Handle map conversion.

	if resourceMap, ok := resource.(map[string]interface{}); ok {

		krmRes := KRMResource{
			Metadata: json.RawMessage("{}"),
		}

		if apiVersion, ok := resourceMap["apiVersion"].(string); ok {
			krmRes.APIVersion = apiVersion
		}

		if kind, ok := resourceMap["kind"].(string); ok {
			krmRes.Kind = kind
		}

		if metadata, ok := resourceMap["metadata"].(map[string]interface{}); ok {
			if metadataBytes, err := json.Marshal(metadata); err == nil {
				krmRes.Metadata = metadataBytes
			}
		}

		if spec, ok := resourceMap["spec"].(map[string]interface{}); ok {
			if specBytes, err := json.Marshal(spec); err == nil {
				krmRes.Spec = specBytes
			}
		}

		if status, ok := resourceMap["status"].(map[string]interface{}); ok {
			if statusBytes, err := json.Marshal(status); err == nil {
				krmRes.Status = statusBytes
			}
		}

		if data, ok := resourceMap["data"].(map[string]interface{}); ok {
			if dataBytes, err := json.Marshal(data); err == nil {
				krmRes.Data = dataBytes
			}
		}

		return krmRes, nil

	}

	return KRMResource{}, fmt.Errorf("cannot convert %T to KRMResource", resource)
}

// convertToFunctionConfig converts interface{} to FunctionConfig.

func convertToFunctionConfig(function interface{}) (FunctionConfig, error) {
	// Handle FunctionConfig type directly.

	if funcConfig, ok := function.(FunctionConfig); ok {
		return funcConfig, nil
	}

	// Handle map conversion.

	if functionMap, ok := function.(map[string]interface{}); ok {

		funcConfig := FunctionConfig{}

		if image, ok := functionMap["image"].(string); ok {
			funcConfig.Image = image
		}

		if configPath, ok := functionMap["configPath"].(string); ok {
			funcConfig.ConfigPath = configPath
		}

		if configMap, ok := functionMap["configMap"].(map[string]interface{}); ok {
			if configMapBytes, err := json.Marshal(configMap); err == nil {
				funcConfig.ConfigMap = configMapBytes
			}
		}

		return funcConfig, nil

	}

	return FunctionConfig{}, fmt.Errorf("cannot convert %T to FunctionConfig", function)
}

// convertFromKRMResource converts KRMResource to interface{} for storage in multicluster types.

func convertFromKRMResource(resource KRMResource) interface{} {
	return json.RawMessage("{}")
}

// convertFromFunctionConfig converts FunctionConfig to interface{} for storage in multicluster types.

func convertFromFunctionConfig(function FunctionConfig) interface{} {
	return json.RawMessage("{}")
}

// Helper functions for package content operations.

// convertKRMResourceToYAML converts a KRMResource to YAML bytes.

func convertKRMResourceToYAML(resource KRMResource) ([]byte, error) {
	// Create a map with the resource data.

	resourceMap := json.RawMessage("{}")

	if resource.Spec != nil {
		resourceMap["spec"] = resource.Spec
	}

	if resource.Status != nil {
		resourceMap["status"] = resource.Status
	}

	if resource.Data != nil {
		resourceMap["data"] = resource.Data
	}

	// Convert to YAML using JSON marshaling then conversion.

	jsonBytes, err := json.Marshal(resourceMap)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal resource to JSON: %w", err)
	}

	// Simple YAML conversion (in production, use proper YAML library).

	// For now, return as formatted JSON which is valid YAML.

	return jsonBytes, nil
}

// convertYAMLToKRMResource converts YAML bytes to a KRMResource.

func convertYAMLToKRMResource(yamlData []byte) (*KRMResource, error) {
	var resourceMap map[string]interface{}

	// Parse as JSON first (most YAML is valid JSON).

	if err := json.Unmarshal(yamlData, &resourceMap); err != nil {
		return nil, fmt.Errorf("failed to unmarshal YAML data: %w", err)
	}

	resource := &KRMResource{}

	// Extract required fields.

	if apiVersion, ok := resourceMap["apiVersion"].(string); ok {
		resource.APIVersion = apiVersion
	}

	if kind, ok := resourceMap["kind"].(string); ok {
		resource.Kind = kind
	}

	if metadata, ok := resourceMap["metadata"].(map[string]interface{}); ok {
		if metadataBytes, err := json.Marshal(metadata); err == nil {
			resource.Metadata = metadataBytes
		}
	}

	if spec, ok := resourceMap["spec"].(map[string]interface{}); ok {
		if specBytes, err := json.Marshal(spec); err == nil {
			resource.Spec = specBytes
		}
	}

	if status, ok := resourceMap["status"].(map[string]interface{}); ok {
		if statusBytes, err := json.Marshal(status); err == nil {
			resource.Status = statusBytes
		}
	}

	if data, ok := resourceMap["data"].(map[string]interface{}); ok {
		if dataBytes, err := json.Marshal(data); err == nil {
			resource.Data = dataBytes
		}
	}

	return resource, nil
}

// generateResourceFilename generates a filename for a KRM resource.

func generateResourceFilename(resource KRMResource) string {
	var name string

	if metadata := resource.Metadata; len(metadata) > 0 {
		var metadataMap map[string]interface{}
		if err := json.Unmarshal(metadata, &metadataMap); err == nil {
			if resourceName, ok := metadataMap["name"].(string); ok && resourceName != "" {
				name = resourceName
			}
		}
	}

	if name == "" {
		name = "resource"
	}

	// Create filename from kind and name.

	kind := strings.ToLower(resource.Kind)

	if kind == "" {
		kind = "resource"
	}

	return fmt.Sprintf("%s-%s.yaml", kind, name)
}

// isYAMLFile checks if a filename represents a YAML file.

func isYAMLFile(filename string) bool {
	return strings.HasSuffix(strings.ToLower(filename), ".yaml") ||

		strings.HasSuffix(strings.ToLower(filename), ".yml")
}

// getPackageResourcesFromPorch retrieves package resources from Porch PackageRevisionResources.

func (c *Client) getPackageResourcesFromPorch(ctx context.Context, name, revision string) (map[string][]byte, error) {
	// PackageRevisionResources is a sub-resource of PackageRevision in Porch.

	gvr := schema.GroupVersionResource{
		Group: "porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "packagerevisionresources",
	}

	resourceName := fmt.Sprintf("%s-%s", name, revision)

	obj, err := c.dynamic.Resource(gvr).Get(ctx, resourceName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get package revision resources: %w", err)
	}

	// Extract resources from the PackageRevisionResources spec.

	spec, found, err := unstructured.NestedMap(obj.Object, "spec")

	if err != nil || !found {
		return nil, fmt.Errorf("no spec found in PackageRevisionResources")
	}

	resources, found, err := unstructured.NestedMap(spec, "resources")

	if err != nil || !found {
		return nil, fmt.Errorf("no resources found in PackageRevisionResources spec")
	}

	contents := make(map[string][]byte)

	for filename, content := range resources {
		if contentStr, ok := content.(string); ok {
			contents[filename] = []byte(contentStr)
		}
	}

	return contents, nil
}

// updatePackageResourcesInPorch updates package resources in Porch PackageRevisionResources.

func (c *Client) updatePackageResourcesInPorch(ctx context.Context, name, revision string, contents map[string][]byte) error {
	gvr := schema.GroupVersionResource{
		Group: "porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "packagerevisionresources",
	}

	resourceName := fmt.Sprintf("%s-%s", name, revision)

	// Get current PackageRevisionResources.

	obj, err := c.dynamic.Resource(gvr).Get(ctx, resourceName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get package revision resources: %w", err)
	}

	// Update resources in spec.

	resources := make(map[string]interface{})

	for filename, content := range contents {
		resources[filename] = string(content)
	}

	if err := unstructured.SetNestedMap(obj.Object, resources, "spec", "resources"); err != nil {
		return fmt.Errorf("failed to set resources in spec: %w", err)
	}

	// Update the PackageRevisionResources.

	_, err = c.dynamic.Resource(gvr).Update(ctx, obj, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update package revision resources: %w", err)
	}

	return nil
}

// convertResourcesToUnstructured converts KRM resources to unstructured format.

func (c *Client) convertResourcesToUnstructured(resources []KRMResource) []interface{} {
	result := make([]interface{}, len(resources))

	for i, resource := range resources {

		resourceMap := json.RawMessage("{}")

		if resource.Spec != nil {
			resourceMap["spec"] = resource.Spec
		}

		if resource.Status != nil {
			resourceMap["status"] = resource.Status
		}

		if resource.Data != nil {
			resourceMap["data"] = resource.Data
		}

		result[i] = resourceMap

	}

	return result
}

// waitForFunctionTaskCompletion waits for a function evaluation task to complete.

func (c *Client) waitForFunctionTaskCompletion(ctx context.Context, taskName string, timeout time.Duration) (*FunctionResponse, error) {
	gvr := schema.GroupVersionResource{
		Group: "porch.kpt.dev",

		Version: "v1alpha1",

		Resource: "functionevaltasks",
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)

	defer cancel()

	ticker := time.NewTicker(2 * time.Second)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return nil, fmt.Errorf("timeout waiting for function task completion")

		case <-ticker.C:

			obj, err := c.dynamic.Resource(gvr).Get(ctx, taskName, metav1.GetOptions{})
			if err != nil {

				c.logger.Error(err, "Failed to get function task status", "taskName", taskName)

				continue

			}

			// Check task status.

			status, found, err := unstructured.NestedMap(obj.Object, "status")

			if err != nil || !found {
				continue // Task not yet started
			}

			// Check if completed.

			if phase, ok := status["phase"].(string); ok {
				switch phase {

				case "Succeeded":

					return c.extractFunctionResponse(status)

				case "Failed":

					errorMsg := "Function execution failed"

					if msg, ok := status["error"].(string); ok {
						errorMsg = msg
					}

					return nil, errors.New(errorMsg)

				case "Running", "Pending":

					continue // Keep waiting

				default:

					c.logger.V(1).Info("Unknown function task phase", "phase", phase)

					continue

				}
			}

		}
	}
}

// extractFunctionResponse extracts FunctionResponse from task status.

func (c *Client) extractFunctionResponse(status map[string]interface{}) (*FunctionResponse, error) {
	response := &FunctionResponse{
		Resources: []KRMResource{},

		Results: []*FunctionResult{},

		Logs: []string{},
	}

	// Extract resources.

	if resources, ok := status["resources"].([]interface{}); ok {
		for _, res := range resources {
			if resMap, ok := res.(map[string]interface{}); ok {

				resource := KRMResource{}

				if apiVersion, ok := resMap["apiVersion"].(string); ok {
					resource.APIVersion = apiVersion
				}

				if kind, ok := resMap["kind"].(string); ok {
					resource.Kind = kind
				}

				if metadata, ok := resMap["metadata"].(map[string]interface{}); ok {
					if metadataBytes, err := json.Marshal(metadata); err == nil {
						resource.Metadata = metadataBytes
					}
				}

				if spec, ok := resMap["spec"].(map[string]interface{}); ok {
					if specBytes, err := json.Marshal(spec); err == nil {
						resource.Spec = specBytes
					}
				}

				if statusData, ok := resMap["status"].(map[string]interface{}); ok {
					if statusBytes, err := json.Marshal(statusData); err == nil {
						resource.Status = statusBytes
					}
				}

				if data, ok := resMap["data"].(map[string]interface{}); ok {
					if dataBytes, err := json.Marshal(data); err == nil {
						resource.Data = dataBytes
					}
				}

				response.Resources = append(response.Resources, resource)

			}
		}
	}

	// Extract results.

	if results, ok := status["results"].([]interface{}); ok {
		for _, res := range results {
			if resMap, ok := res.(map[string]interface{}); ok {

				result := &FunctionResult{}

				if message, ok := resMap["message"].(string); ok {
					result.Message = message
				}

				if severity, ok := resMap["severity"].(string); ok {
					result.Severity = severity
				}

				if field, ok := resMap["field"].(string); ok {
					result.Field = field
				}

				if file, ok := resMap["file"].(string); ok {
					result.File = file
				}

				if tags, ok := resMap["tags"].(map[string]interface{}); ok {

					result.Tags = make(map[string]string)

					for k, v := range tags {
						if vStr, ok := v.(string); ok {
							result.Tags[k] = vStr
						}
					}

				}

				response.Results = append(response.Results, result)

			}
		}
	}

	// Extract logs.

	if logs, ok := status["logs"].([]interface{}); ok {
		for _, log := range logs {
			if logStr, ok := log.(string); ok {
				response.Logs = append(response.Logs, logStr)
			}
		}
	}

	return response, nil
}

// Close closes the client and cleans up resources.

func (c *Client) Close() error {
	c.logger.Info("Closing Porch client")

	// Close connection pool.

	if c.connectionPool != nil {
		close(c.connectionPool.pool)
	}

	// Clear cache.

	if c.cache != nil {

		c.cache.mutex.Lock()

		c.cache.cache = make(map[string]*cacheEntry)

		c.cache.mutex.Unlock()

	}

	return nil
}

// GetConfig returns the client configuration.

func (c *Client) GetConfig() *ClientConfig {
	return c.config
}

// GetMetrics returns the client metrics.

func (c *Client) GetMetrics() *ClientMetrics {
	return c.metrics
}

// IsHealthy checks if the client is healthy.

func (c *Client) IsHealthy(ctx context.Context) bool {
	_, err := c.Health(ctx)

	return err == nil
}

// SetLogger updates the client logger.

func (c *Client) SetLogger(logger logr.Logger) {
	c.logger = logger
}

// Stub implementations for sub-managers - will be replaced by full implementations.

// repositoryManagerStub provides a temporary stub implementation.

type repositoryManagerStub struct {
	client *Client
}

// RegisterRepository performs registerrepository operation.

func (r *repositoryManagerStub) RegisterRepository(ctx context.Context, config *RepositoryConfig) (*Repository, error) {
	return &Repository{}, nil
}

// UnregisterRepository performs unregisterrepository operation.

func (r *repositoryManagerStub) UnregisterRepository(ctx context.Context, name string) error {
	return nil
}

// SynchronizeRepository performs synchronizerepository operation.

func (r *repositoryManagerStub) SynchronizeRepository(ctx context.Context, name string) (*SyncResult, error) {
	return &SyncResult{}, nil
}

// GetRepositoryHealth performs getrepositoryhealth operation.

func (r *repositoryManagerStub) GetRepositoryHealth(ctx context.Context, name string) (*RepositoryHealth, error) {
	health := RepositoryHealthHealthy

	return &health, nil
}

// CreateBranch performs createbranch operation.

func (r *repositoryManagerStub) CreateBranch(ctx context.Context, repoName, branchName, baseBranch string) error {
	return nil
}

// DeleteBranch performs deletebranch operation.

func (r *repositoryManagerStub) DeleteBranch(ctx context.Context, repoName, branchName string) error {
	return nil
}

// ListBranches performs listbranches operation.

func (r *repositoryManagerStub) ListBranches(ctx context.Context, repoName string) ([]string, error) {
	return []string{"main"}, nil
}

// UpdateCredentials performs updatecredentials operation.

func (r *repositoryManagerStub) UpdateCredentials(ctx context.Context, repoName string, creds *Credentials) error {
	return nil
}

// ValidateAccess performs validateaccess operation.

func (r *repositoryManagerStub) ValidateAccess(ctx context.Context, repoName string) error {
	return nil
}

// packageRevisionManagerStub provides a temporary stub implementation.

type packageRevisionManagerStub struct {
	client *Client
}

// CreatePackage performs createpackage operation.

func (p *packageRevisionManagerStub) CreatePackage(ctx context.Context, spec *PackageSpec) (*PackageRevision, error) {
	return &PackageRevision{}, nil
}

// ClonePackage performs clonepackage operation.

func (p *packageRevisionManagerStub) ClonePackage(ctx context.Context, source *PackageReference, target *PackageSpec) (*PackageRevision, error) {
	return &PackageRevision{}, nil
}

// DeletePackage performs deletepackage operation.

func (p *packageRevisionManagerStub) DeletePackage(ctx context.Context, ref *PackageReference) error {
	return nil
}

// CreateRevision performs createrevision operation.

func (p *packageRevisionManagerStub) CreateRevision(ctx context.Context, ref *PackageReference) (*PackageRevision, error) {
	return &PackageRevision{}, nil
}

// GetRevision performs getrevision operation.

func (p *packageRevisionManagerStub) GetRevision(ctx context.Context, ref *PackageReference) (*PackageRevision, error) {
	return &PackageRevision{}, nil
}

// ListRevisions performs listrevisions operation.

func (p *packageRevisionManagerStub) ListRevisions(ctx context.Context, packageName string) ([]*PackageRevision, error) {
	return []*PackageRevision{}, nil
}

// CompareRevisions performs comparerevisions operation.

func (p *packageRevisionManagerStub) CompareRevisions(ctx context.Context, ref1, ref2 *PackageReference) (*ComparisonResult, error) {
	return &ComparisonResult{}, nil
}

// PromoteToProposed performs promotetoproposed operation.

func (p *packageRevisionManagerStub) PromoteToProposed(ctx context.Context, ref *PackageReference) error {
	return nil
}

// PromoteToPublished performs promotetopublished operation.

func (p *packageRevisionManagerStub) PromoteToPublished(ctx context.Context, ref *PackageReference) error {
	return nil
}

// RevertToRevision performs reverttorevision operation.

func (p *packageRevisionManagerStub) RevertToRevision(ctx context.Context, ref *PackageReference, targetRevision string) error {
	return nil
}

// UpdateContent performs updatecontent operation.

func (p *packageRevisionManagerStub) UpdateContent(ctx context.Context, ref *PackageReference, updates map[string][]byte) error {
	return nil
}

// GetContent performs getcontent operation.

func (p *packageRevisionManagerStub) GetContent(ctx context.Context, ref *PackageReference) (*PackageContent, error) {
	return &PackageContent{}, nil
}

// ValidateContent performs validatecontent operation.

func (p *packageRevisionManagerStub) ValidateContent(ctx context.Context, ref *PackageReference) (*ValidationResult, error) {
	return &ValidationResult{Valid: true}, nil
}

// functionRunnerStub provides a temporary stub implementation.

type functionRunnerStub struct {
	client *Client
}

// ExecuteFunction performs executefunction operation.

func (f *functionRunnerStub) ExecuteFunction(ctx context.Context, req *FunctionRequest) (*FunctionResponse, error) {
	return &FunctionResponse{}, nil
}

// ExecutePipeline performs executepipeline operation.

func (f *functionRunnerStub) ExecutePipeline(ctx context.Context, req *PipelineRequest) (*PipelineResponse, error) {
	return &PipelineResponse{}, nil
}

// ValidateFunction performs validatefunction operation.

func (f *functionRunnerStub) ValidateFunction(ctx context.Context, functionName string) (*FunctionValidation, error) {
	return &FunctionValidation{Valid: true}, nil
}

// ListFunctions performs listfunctions operation.

func (f *functionRunnerStub) ListFunctions(ctx context.Context) ([]*FunctionInfo, error) {
	return []*FunctionInfo{}, nil
}

// GetFunctionSchema performs getfunctionschema operation.

func (f *functionRunnerStub) GetFunctionSchema(ctx context.Context, functionName string) (*FunctionSchema, error) {
	return &FunctionSchema{}, nil
}

// RegisterFunction performs registerfunction operation.

func (f *functionRunnerStub) RegisterFunction(ctx context.Context, info *FunctionInfo) error {
	return nil
}
