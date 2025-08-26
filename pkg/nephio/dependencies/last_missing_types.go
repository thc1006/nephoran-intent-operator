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

package dependencies

import (
	"context"
	"time"
)

// Last batch of missing types

// ResolutionCache represents a cache for resolution results
type ResolutionCache struct {
	MaxSize     int                              `json:"maxSize"`
	TTL         time.Duration                    `json:"ttl"`
	Entries     map[string]*ResolutionCacheEntry `json:"entries"`
	Stats       *CacheStats                      `json:"stats"`
	LastCleanup time.Time                        `json:"lastCleanup"`
}

// ResolutionCacheEntry represents a cached resolution result
type ResolutionCacheEntry struct {
	Key       string            `json:"key"`
	Result    *ResolutionResult `json:"result"`
	CreatedAt time.Time         `json:"createdAt"`
	ExpiresAt time.Time         `json:"expiresAt"`
	HitCount  int64             `json:"hitCount"`
}

// WorkerPool represents a pool of worker goroutines
type WorkerPool struct {
	Size        int               `json:"size"`
	QueueSize   int               `json:"queueSize"`
	Workers     []*Worker         `json:"workers"`
	JobQueue    chan Job          `json:"-"`
	ResultQueue chan JobResult    `json:"-"`
	Stats       *WorkerPoolStats  `json:"stats"`
}

// Worker represents a single worker in the pool
type Worker struct {
	ID       int    `json:"id"`
	Status   string `json:"status"`
	JobsProcessed int64 `json:"jobsProcessed"`
	LastJobAt *time.Time `json:"lastJobAt,omitempty"`
}

// Job represents a work item for the worker pool
type Job interface {
	Execute() JobResult
	ID() string
	Priority() int
}

// JobResult represents the result of a job execution
type JobResult interface {
	Success() bool
	Error() error
	Data() interface{}
	ExecutionTime() time.Duration
}

// WorkerPoolStats represents statistics for the worker pool
type WorkerPoolStats struct {
	TotalJobs       int64         `json:"totalJobs"`
	CompletedJobs   int64         `json:"completedJobs"`
	FailedJobs      int64         `json:"failedJobs"`
	QueuedJobs      int64         `json:"queuedJobs"`
	AvgJobTime      time.Duration `json:"avgJobTime"`
	MaxJobTime      time.Duration `json:"maxJobTime"`
	ActiveWorkers   int           `json:"activeWorkers"`
	IdleWorkers     int           `json:"idleWorkers"`
}

// RateLimiter represents a rate limiting interface
type RateLimiter interface {
	Allow() bool
	Wait(ctx context.Context) error
	Limit() float64
	SetLimit(limit float64)
	Tokens() int
}

// ResolverConfig represents configuration for the dependency resolver
type ResolverConfig struct {
	// Resolution options
	MaxDepth            int               `json:"maxDepth"`
	Timeout             time.Duration     `json:"timeout"`
	EnableCaching       bool              `json:"enableCaching"`
	EnableParallel      bool              `json:"enableParallel"`
	WorkerCount         int               `json:"workerCount"`
	
	// Cache configuration
	CacheSize           int               `json:"cacheSize"`
	CacheTTL            time.Duration     `json:"cacheTTL"`
	
	// Rate limiting
	RateLimit           float64           `json:"rateLimit"`
	RateBurst           int               `json:"rateBurst"`
	
	// Version handling
	PrereleaseStrategy  PrereleaseStrategy  `json:"prereleaseStrategy"`
	BuildMetadataStrategy BuildMetadataStrategy `json:"buildMetadataStrategy"`
	
	// Conflict resolution
	ConflictStrategy    ConflictStrategy  `json:"conflictStrategy"`
	AutoResolveConflicts bool             `json:"autoResolveConflicts"`
	
	// Retry configuration
	MaxRetries          int               `json:"maxRetries"`
	RetryDelay          time.Duration     `json:"retryDelay"`
	
	// Logging and monitoring
	LogLevel            string            `json:"logLevel"`
	EnableMetrics       bool              `json:"enableMetrics"`
	MetricsPort         int               `json:"metricsPort"`
}

// DependencyProvider represents an interface for providing dependency information
type DependencyProvider interface {
	GetPackage(ctx context.Context, ref *PackageReference) (*PackageInfo, error)
	GetVersions(ctx context.Context, packageName string) ([]*VersionInfo, error)
	GetDependencies(ctx context.Context, ref *PackageReference) ([]*PackageReference, error)
	SearchPackages(ctx context.Context, query string) ([]*PackageInfo, error)
	SupportsPackage(packageName string) bool
	GetMetadata(ctx context.Context, ref *PackageReference) (map[string]interface{}, error)
}

// PackageInfo represents information about a package
type PackageInfo struct {
	Name           string                 `json:"name"`
	Repository     string                 `json:"repository"`
	Description    string                 `json:"description"`
	Homepage       string                 `json:"homepage,omitempty"`
	License        string                 `json:"license,omitempty"`
	Author         string                 `json:"author,omitempty"`
	Keywords       []string               `json:"keywords,omitempty"`
	Versions       []*VersionInfo         `json:"versions,omitempty"`
	LatestVersion  string                 `json:"latestVersion"`
	DownloadCount  int64                  `json:"downloadCount,omitempty"`
	Stars          int                    `json:"stars,omitempty"`
	Forks          int                    `json:"forks,omitempty"`
	Issues         int                    `json:"issues,omitempty"`
	LastUpdated    time.Time              `json:"lastUpdated"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

// ConflictDetector represents an interface for detecting conflicts
type ConflictDetector interface {
	DetectConflicts(ctx context.Context, packages []*PackageReference) ([]*DependencyConflict, error)
	Name() string
	CanHandle(conflictType ConflictType) bool
	Priority() int
}

// ConflictPredictor represents an interface for predicting potential conflicts
type ConflictPredictor interface {
	PredictConflicts(ctx context.Context, packages []*PackageReference) ([]*ConflictPrediction, error)
	GetPredictionAccuracy() float64
	UpdateModel(conflicts []*DependencyConflict) error
}

// ConflictPrediction represents a predicted conflict
type ConflictPrediction struct {
	ID          string              `json:"id"`
	Type        ConflictType        `json:"type"`
	Packages    []*PackageReference `json:"packages"`
	Probability float64             `json:"probability"`
	Confidence  float64             `json:"confidence"`
	Description string              `json:"description"`
	Prevention  []string            `json:"prevention,omitempty"`
	PredictedAt time.Time           `json:"predictedAt"`
}

// UpdateConstraints represents constraints for dependency updates
type UpdateConstraints struct {
	AllowMajorUpdates    bool              `json:"allowMajorUpdates"`
	AllowMinorUpdates    bool              `json:"allowMinorUpdates"`
	AllowPatchUpdates    bool              `json:"allowPatchUpdates"`
	AllowPrerelease      bool              `json:"allowPrerelease"`
	MaxVersionAge        time.Duration     `json:"maxVersionAge,omitempty"`
	RequiredPackages     []string          `json:"requiredPackages,omitempty"`
	BannedPackages       []string          `json:"bannedPackages,omitempty"`
	VersionOverrides     map[string]string `json:"versionOverrides,omitempty"`
	SecurityUpdatesOnly  bool              `json:"securityUpdatesOnly"`
	StableVersionsOnly   bool              `json:"stableVersionsOnly"`
}

// CompatibilityRules represents rules for compatibility checking
type CompatibilityRules struct {
	SemanticVersioning   bool                        `json:"semanticVersioning"`
	BreakingChangePolicy string                      `json:"breakingChangePolicy"`
	PlatformCompatibility map[string][]string        `json:"platformCompatibility"`
	APICompatibility     map[string]*APICompatRule   `json:"apiCompatibility"`
	LicenseCompatibility map[string][]string         `json:"licenseCompatibility"`
	CustomRules          []*CustomCompatibilityRule  `json:"customRules,omitempty"`
}

// APICompatRule represents API compatibility rules
type APICompatRule struct {
	APIVersion    string   `json:"apiVersion"`
	Compatible    []string `json:"compatible"`
	Incompatible  []string `json:"incompatible"`
	DeprecatedIn  string   `json:"deprecatedIn,omitempty"`
	RemovedIn     string   `json:"removedIn,omitempty"`
}

// CustomCompatibilityRule represents a custom compatibility rule
type CustomCompatibilityRule struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Pattern     string                 `json:"pattern"`
	Action      string                 `json:"action"`
	Parameters  map[string]interface{} `json:"parameters,omitempty"`
}

// RolloutConfig represents configuration for rolling out updates
type RolloutConfig struct {
	Strategy        string        `json:"strategy"`        // "immediate", "gradual", "canary", "blue-green"
	BatchSize       int           `json:"batchSize"`
	BatchDelay      time.Duration `json:"batchDelay"`
	HealthCheck     *HealthCheckConfig `json:"healthCheck,omitempty"`
	Rollback        *RollbackConfig    `json:"rollback,omitempty"`
	Notifications   []string      `json:"notifications,omitempty"`
	MaxFailures     int           `json:"maxFailures,omitempty"`
	SuccessThreshold float64      `json:"successThreshold,omitempty"`
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Enabled         bool          `json:"enabled"`
	Endpoint        string        `json:"endpoint,omitempty"`
	Timeout         time.Duration `json:"timeout"`
	Interval        time.Duration `json:"interval"`
	HealthyThreshold int          `json:"healthyThreshold"`
	UnhealthyThreshold int        `json:"unhealthyThreshold"`
	ExpectedStatus  int           `json:"expectedStatus,omitempty"`
	ExpectedBody    string        `json:"expectedBody,omitempty"`
}

// RollbackConfig represents rollback configuration
type RollbackConfig struct {
	Enabled       bool          `json:"enabled"`
	AutoRollback  bool          `json:"autoRollback"`
	Timeout       time.Duration `json:"timeout"`
	FailureThreshold int        `json:"failureThreshold"`
	CheckInterval time.Duration `json:"checkInterval"`
}