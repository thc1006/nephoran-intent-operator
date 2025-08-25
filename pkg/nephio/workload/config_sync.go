package workload

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

// ConfigSyncManager manages GitOps-based configuration synchronization
type ConfigSyncManager struct {
	client            client.Client
	registry          *ClusterRegistry
	repositories      map[string]*GitRepository
	syncStatus        map[string]*SyncStatus
	policyManager     *PolicyManager
	driftDetector     *DriftDetector
	complianceTracker *ComplianceTracker
	logger            logr.Logger
	metrics           *configSyncMetrics
	mu                sync.RWMutex
	stopCh            chan struct{}
}

// GitRepository represents a Git repository configuration
type GitRepository struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	URL        string            `json:"url"`
	Branch     string            `json:"branch"`
	Path       string            `json:"path"`
	Auth       GitAuth           `json:"auth"`
	SyncPolicy SyncPolicy        `json:"sync_policy"`
	Clusters   []string          `json:"clusters"`
	LastSync   time.Time         `json:"last_sync"`
	LastCommit string            `json:"last_commit"`
	Status     RepoStatus        `json:"status"`
	Metadata   map[string]string `json:"metadata"`
}

// GitAuth contains Git authentication information
type GitAuth struct {
	Type       string `json:"type"`
	Username   string `json:"username"`
	Password   string `json:"-"`
	Token      string `json:"-"`
	PrivateKey string `json:"-"`
	SecretRef  string `json:"secret_ref,omitempty"`
}

// SyncPolicy defines how repositories are synchronized
type SyncPolicy struct {
	AutoSync          bool          `json:"auto_sync"`
	SyncInterval      time.Duration `json:"sync_interval"`
	RetryLimit        int           `json:"retry_limit"`
	RetryBackoff      time.Duration `json:"retry_backoff"`
	PruneOrphans      bool          `json:"prune_orphans"`
	ValidateManifests bool          `json:"validate_manifests"`
	DryRun            bool          `json:"dry_run"`
	Timeout           time.Duration `json:"timeout"`
}

// RepoStatus represents the status of a repository
type RepoStatus string

const (
	RepoStatusHealthy   RepoStatus = "healthy"
	RepoStatusSyncing   RepoStatus = "syncing"
	RepoStatusFailed    RepoStatus = "failed"
	RepoStatusOutOfSync RepoStatus = "out-of-sync"
	RepoStatusUnknown   RepoStatus = "unknown"
)

// SyncStatus tracks the sync status for a cluster-repository pair
type SyncStatus struct {
	ClusterID       string           `json:"cluster_id"`
	RepositoryID    string           `json:"repository_id"`
	Status          SyncState        `json:"status"`
	LastSyncTime    time.Time        `json:"last_sync_time"`
	LastSuccessTime time.Time        `json:"last_success_time"`
	CommitHash      string           `json:"commit_hash"`
	SyncedResources []SyncedResource `json:"synced_resources"`
	Errors          []SyncError      `json:"errors"`
	Metrics         SyncMetrics      `json:"metrics"`
}

// SyncState represents the state of synchronization
type SyncState string

const (
	SyncStateInSync    SyncState = "in-sync"
	SyncStateOutOfSync SyncState = "out-of-sync"
	SyncStateSyncing   SyncState = "syncing"
	SyncStateFailed    SyncState = "failed"
	SyncStateUnknown   SyncState = "unknown"
)

// SyncedResource represents a resource that has been synced
type SyncedResource struct {
	Group       string    `json:"group"`
	Version     string    `json:"version"`
	Kind        string    `json:"kind"`
	Namespace   string    `json:"namespace"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	LastUpdated time.Time `json:"last_updated"`
	Hash        string    `json:"hash"`
}

// SyncError represents an error during synchronization
type SyncError struct {
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
	Resource  string    `json:"resource"`
	Type      string    `json:"type"`
}

// SyncMetrics contains sync operation metrics
type SyncMetrics struct {
	Duration         time.Duration `json:"duration"`
	ResourcesTotal   int           `json:"resources_total"`
	ResourcesApplied int           `json:"resources_applied"`
	ResourcesFailed  int           `json:"resources_failed"`
	RetryCount       int           `json:"retry_count"`
}

// PolicyManager manages policies as code
type PolicyManager struct {
	policies   map[string]*Policy
	violations map[string][]*PolicyViolation
	templates  map[string]*PolicyTemplate
	logger     logr.Logger
	mu         sync.RWMutex
}

// Policy represents a policy configuration
type Policy struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"`
	Scope       string            `json:"scope"`
	Rules       []PolicyRule      `json:"rules"`
	Enforcement string            `json:"enforcement"`
	Metadata    map[string]string `json:"metadata"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// PolicyRule represents a single policy rule
type PolicyRule struct {
	Name       string      `json:"name"`
	Condition  string      `json:"condition"`
	Action     string      `json:"action"`
	Parameters interface{} `json:"parameters"`
	Severity   string      `json:"severity"`
}

// PolicyViolation represents a policy violation
type PolicyViolation struct {
	PolicyID   string    `json:"policy_id"`
	ClusterID  string    `json:"cluster_id"`
	Resource   string    `json:"resource"`
	Violation  string    `json:"violation"`
	Severity   string    `json:"severity"`
	DetectedAt time.Time `json:"detected_at"`
	ResolvedAt time.Time `json:"resolved_at,omitempty"`
	Status     string    `json:"status"`
}

// PolicyTemplate represents a policy template
type PolicyTemplate struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Category    string            `json:"category"`
	Template    string            `json:"template"`
	Parameters  map[string]string `json:"parameters"`
}

// DriftDetector detects configuration drift
type DriftDetector struct {
	detectionRules map[string]*DriftRule
	drift          map[string][]*DriftDetection
	logger         logr.Logger
	mu             sync.RWMutex
}

// DriftRule defines a rule for drift detection
type DriftRule struct {
	Name      string        `json:"name"`
	Resource  string        `json:"resource"`
	Fields    []string      `json:"fields"`
	Threshold time.Duration `json:"threshold"`
	Severity  string        `json:"severity"`
	Action    string        `json:"action"`
}

// DriftDetection represents detected configuration drift
type DriftDetection struct {
	ClusterID     string      `json:"cluster_id"`
	Resource      string      `json:"resource"`
	Field         string      `json:"field"`
	ExpectedValue interface{} `json:"expected_value"`
	ActualValue   interface{} `json:"actual_value"`
	DetectedAt    time.Time   `json:"detected_at"`
	Severity      string      `json:"severity"`
	Status        string      `json:"status"`
}

// ComplianceTracker tracks compliance status
type ComplianceTracker struct {
	standards   map[string]*ComplianceStandard
	assessments map[string]*ComplianceAssessment
	reports     map[string]*ComplianceReport
	logger      logr.Logger
	mu          sync.RWMutex
}

// ComplianceStandard represents a compliance standard
type ComplianceStandard struct {
	Name         string              `json:"name"`
	Version      string              `json:"version"`
	Description  string              `json:"description"`
	Controls     []ComplianceControl `json:"controls"`
	Requirements []string            `json:"requirements"`
}

// ComplianceControl represents a compliance control
type ComplianceControl struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Checks      []string `json:"checks"`
	Severity    string   `json:"severity"`
}

// ComplianceAssessment represents a compliance assessment
type ComplianceAssessment struct {
	ID        string             `json:"id"`
	ClusterID string             `json:"cluster_id"`
	Standard  string             `json:"standard"`
	Results   []ComplianceResult `json:"results"`
	Score     float64            `json:"score"`
	Status    string             `json:"status"`
	CreatedAt time.Time          `json:"created_at"`
}

// ComplianceResult represents a compliance check result
type ComplianceResult struct {
	ControlID string    `json:"control_id"`
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Evidence  string    `json:"evidence"`
	Timestamp time.Time `json:"timestamp"`
}

// ComplianceReport represents a compliance report
type ComplianceReport struct {
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Period      string                 `json:"period"`
	Clusters    []string               `json:"clusters"`
	Standards   []string               `json:"standards"`
	Summary     ComplianceSummary      `json:"summary"`
	Details     []ComplianceAssessment `json:"details"`
	GeneratedAt time.Time              `json:"generated_at"`
}

// ComplianceSummary provides a summary of compliance status
type ComplianceSummary struct {
	TotalClusters     int     `json:"total_clusters"`
	CompliantClusters int     `json:"compliant_clusters"`
	OverallScore      float64 `json:"overall_score"`
	CriticalFindings  int     `json:"critical_findings"`
	HighFindings      int     `json:"high_findings"`
	MediumFindings    int     `json:"medium_findings"`
	LowFindings       int     `json:"low_findings"`
}

// configSyncMetrics contains Prometheus metrics
type configSyncMetrics struct {
	syncOperations   *prometheus.CounterVec
	syncDuration     *prometheus.HistogramVec
	syncStatus       *prometheus.GaugeVec
	resourcesTotal   *prometheus.GaugeVec
	driftDetections  *prometheus.CounterVec
	policyViolations *prometheus.CounterVec
	complianceScore  *prometheus.GaugeVec
}

// NewConfigSyncManager creates a new ConfigSync manager
func NewConfigSyncManager(client client.Client, registry *ClusterRegistry, logger logr.Logger) *ConfigSyncManager {
	metrics := &configSyncMetrics{
		syncOperations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_config_sync_operations_total",
				Help: "Total number of sync operations",
			},
			[]string{"cluster", "repository", "result"},
		),
		syncDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "nephio_config_sync_duration_seconds",
				Help:    "Duration of sync operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"cluster", "repository"},
		),
		syncStatus: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nephio_config_sync_status",
				Help: "Current sync status (0=failed, 1=success)",
			},
			[]string{"cluster", "repository"},
		),
		resourcesTotal: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nephio_config_sync_resources_total",
				Help: "Total number of resources managed by sync",
			},
			[]string{"cluster", "repository", "status"},
		),
		driftDetections: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_config_drift_detections_total",
				Help: "Total number of configuration drift detections",
			},
			[]string{"cluster", "resource", "severity"},
		),
		policyViolations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "nephio_policy_violations_total",
				Help: "Total number of policy violations",
			},
			[]string{"cluster", "policy", "severity"},
		),
		complianceScore: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "nephio_compliance_score",
				Help: "Compliance score for clusters",
			},
			[]string{"cluster", "standard"},
		),
	}

	// Register metrics
	prometheus.MustRegister(
		metrics.syncOperations,
		metrics.syncDuration,
		metrics.syncStatus,
		metrics.resourcesTotal,
		metrics.driftDetections,
		metrics.policyViolations,
		metrics.complianceScore,
	)

	return &ConfigSyncManager{
		client:       client,
		registry:     registry,
		repositories: make(map[string]*GitRepository),
		syncStatus:   make(map[string]*SyncStatus),
		policyManager: &PolicyManager{
			policies:   make(map[string]*Policy),
			violations: make(map[string][]*PolicyViolation),
			templates:  make(map[string]*PolicyTemplate),
			logger:     logger.WithName("policy-manager"),
		},
		driftDetector: &DriftDetector{
			detectionRules: make(map[string]*DriftRule),
			drift:          make(map[string][]*DriftDetection),
			logger:         logger.WithName("drift-detector"),
		},
		complianceTracker: &ComplianceTracker{
			standards:   make(map[string]*ComplianceStandard),
			assessments: make(map[string]*ComplianceAssessment),
			reports:     make(map[string]*ComplianceReport),
			logger:      logger.WithName("compliance-tracker"),
		},
		logger:  logger.WithName("config-sync"),
		metrics: metrics,
		stopCh:  make(chan struct{}),
	}
}

// Start starts the ConfigSync manager
func (csm *ConfigSyncManager) Start(ctx context.Context) error {
	csm.logger.Info("Starting ConfigSync manager")

	// Start sync loops
	go csm.runSyncLoop(ctx)
	go csm.runDriftDetection(ctx)
	go csm.runPolicyValidation(ctx)
	go csm.runComplianceAssessment(ctx)

	return nil
}

// Stop stops the ConfigSync manager
func (csm *ConfigSyncManager) Stop() {
	csm.logger.Info("Stopping ConfigSync manager")
	close(csm.stopCh)
}

// AddRepository adds a Git repository for synchronization
func (csm *ConfigSyncManager) AddRepository(ctx context.Context, repo *GitRepository) error {
	csm.mu.Lock()
	defer csm.mu.Unlock()

	if _, exists := csm.repositories[repo.ID]; exists {
		return fmt.Errorf("repository %s already exists", repo.ID)
	}

	csm.logger.Info("Adding repository", "id", repo.ID, "url", repo.URL)

	// Validate repository
	if err := csm.validateRepository(ctx, repo); err != nil {
		return fmt.Errorf("repository validation failed: %w", err)
	}

	// Clone repository to check accessibility
	if err := csm.testRepositoryAccess(ctx, repo); err != nil {
		return fmt.Errorf("repository access test failed: %w", err)
	}

	csm.repositories[repo.ID] = repo
	repo.Status = RepoStatusHealthy
	repo.LastSync = time.Now()

	csm.logger.Info("Successfully added repository", "id", repo.ID)
	return nil
}

// RemoveRepository removes a Git repository
func (csm *ConfigSyncManager) RemoveRepository(ctx context.Context, repoID string) error {
	csm.mu.Lock()
	defer csm.mu.Unlock()

	_, exists := csm.repositories[repoID]
	if !exists {
		return fmt.Errorf("repository %s not found", repoID)
	}

	csm.logger.Info("Removing repository", "id", repoID)

	// Clean up sync status for this repository
	for key := range csm.syncStatus {
		if strings.Contains(key, repoID) {
			delete(csm.syncStatus, key)
		}
	}

	delete(csm.repositories, repoID)

	csm.logger.Info("Successfully removed repository", "id", repoID)
	return nil
}

// SyncRepository manually triggers synchronization for a repository
func (csm *ConfigSyncManager) SyncRepository(ctx context.Context, repoID string, clusterIDs []string) error {
	csm.mu.RLock()
	repo, exists := csm.repositories[repoID]
	csm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("repository %s not found", repoID)
	}

	csm.logger.Info("Manually syncing repository", "repo", repoID, "clusters", len(clusterIDs))

	// If no clusters specified, sync to all configured clusters
	if len(clusterIDs) == 0 {
		clusterIDs = repo.Clusters
	}

	for _, clusterID := range clusterIDs {
		if err := csm.syncToCluster(ctx, repo, clusterID); err != nil {
			csm.logger.Error(err, "Failed to sync to cluster", "repo", repoID, "cluster", clusterID)
		}
	}

	return nil
}

// GetSyncStatus retrieves sync status for a cluster-repository pair
func (csm *ConfigSyncManager) GetSyncStatus(clusterID, repoID string) (*SyncStatus, error) {
	csm.mu.RLock()
	defer csm.mu.RUnlock()

	key := fmt.Sprintf("%s:%s", clusterID, repoID)
	status, exists := csm.syncStatus[key]
	if !exists {
		return nil, fmt.Errorf("sync status not found for cluster %s and repository %s", clusterID, repoID)
	}

	return status, nil
}

// ListRepositories lists all configured repositories
func (csm *ConfigSyncManager) ListRepositories() []*GitRepository {
	csm.mu.RLock()
	defer csm.mu.RUnlock()

	repos := make([]*GitRepository, 0, len(csm.repositories))
	for _, repo := range csm.repositories {
		repos = append(repos, repo)
	}

	return repos
}

// RollbackCluster rolls back a cluster to a previous commit
func (csm *ConfigSyncManager) RollbackCluster(ctx context.Context, clusterID, repoID, commitHash string) error {
	csm.logger.Info("Rolling back cluster", "cluster", clusterID, "repo", repoID, "commit", commitHash)

	// Implementation would:
	// 1. Checkout the specified commit
	// 2. Apply the configurations from that commit
	// 3. Update sync status

	return nil
}

// CreatePolicy creates a new policy
func (csm *ConfigSyncManager) CreatePolicy(ctx context.Context, policy *Policy) error {
	return csm.policyManager.CreatePolicy(policy)
}

// ValidatePolicy validates a policy against a cluster
func (csm *ConfigSyncManager) ValidatePolicy(ctx context.Context, policyID, clusterID string) ([]*PolicyViolation, error) {
	return csm.policyManager.ValidatePolicy(ctx, policyID, clusterID)
}

// DetectDrift detects configuration drift for a cluster
func (csm *ConfigSyncManager) DetectDrift(ctx context.Context, clusterID string) ([]*DriftDetection, error) {
	return csm.driftDetector.DetectDrift(ctx, clusterID)
}

// AssessCompliance assesses compliance for a cluster
func (csm *ConfigSyncManager) AssessCompliance(ctx context.Context, clusterID, standard string) (*ComplianceAssessment, error) {
	return csm.complianceTracker.AssessCompliance(ctx, clusterID, standard)
}

// Private methods

func (csm *ConfigSyncManager) runSyncLoop(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-csm.stopCh:
			return
		case <-ticker.C:
			csm.performScheduledSync(ctx)
		}
	}
}

func (csm *ConfigSyncManager) performScheduledSync(ctx context.Context) {
	csm.mu.RLock()
	repos := make([]*GitRepository, 0, len(csm.repositories))
	for _, repo := range csm.repositories {
		if repo.SyncPolicy.AutoSync {
			repos = append(repos, repo)
		}
	}
	csm.mu.RUnlock()

	for _, repo := range repos {
		// Check if sync is due
		if time.Since(repo.LastSync) >= repo.SyncPolicy.SyncInterval {
			for _, clusterID := range repo.Clusters {
				go csm.syncToCluster(ctx, repo, clusterID)
			}
		}
	}
}

func (csm *ConfigSyncManager) syncToCluster(ctx context.Context, repo *GitRepository, clusterID string) error {
	timer := prometheus.NewTimer(csm.metrics.syncDuration.WithLabelValues(clusterID, repo.ID))
	defer timer.ObserveDuration()

	csm.logger.Info("Syncing repository to cluster", "repo", repo.ID, "cluster", clusterID)

	// Get cluster
	cluster, err := csm.registry.GetCluster(clusterID)
	if err != nil {
		csm.metrics.syncOperations.WithLabelValues(clusterID, repo.ID, "failed").Inc()
		csm.metrics.syncStatus.WithLabelValues(clusterID, repo.ID).Set(0)
		return fmt.Errorf("failed to get cluster: %w", err)
	}

	// Create sync status
	key := fmt.Sprintf("%s:%s", clusterID, repo.ID)
	status := &SyncStatus{
		ClusterID:    clusterID,
		RepositoryID: repo.ID,
		Status:       SyncStateSyncing,
		LastSyncTime: time.Now(),
	}

	csm.mu.Lock()
	csm.syncStatus[key] = status
	csm.mu.Unlock()

	// Clone repository
	repoDir, err := csm.cloneRepository(ctx, repo)
	if err != nil {
		status.Status = SyncStateFailed
		status.Errors = append(status.Errors, SyncError{
			Timestamp: time.Now(),
			Message:   err.Error(),
			Type:      "clone-error",
		})
		csm.metrics.syncOperations.WithLabelValues(clusterID, repo.ID, "failed").Inc()
		csm.metrics.syncStatus.WithLabelValues(clusterID, repo.ID).Set(0)
		return err
	}
	defer os.RemoveAll(repoDir)

	// Read manifests
	manifests, err := csm.readManifests(repoDir, repo.Path)
	if err != nil {
		status.Status = SyncStateFailed
		status.Errors = append(status.Errors, SyncError{
			Timestamp: time.Now(),
			Message:   err.Error(),
			Type:      "manifest-error",
		})
		csm.metrics.syncOperations.WithLabelValues(clusterID, repo.ID, "failed").Inc()
		csm.metrics.syncStatus.WithLabelValues(clusterID, repo.ID).Set(0)
		return err
	}

	// Apply manifests
	syncedResources, appliedCount, failedCount, err := csm.applyManifests(ctx, cluster.Client, manifests, repo.SyncPolicy)
	if err != nil {
		status.Status = SyncStateFailed
		status.Errors = append(status.Errors, SyncError{
			Timestamp: time.Now(),
			Message:   err.Error(),
			Type:      "apply-error",
		})
		csm.metrics.syncOperations.WithLabelValues(clusterID, repo.ID, "failed").Inc()
		csm.metrics.syncStatus.WithLabelValues(clusterID, repo.ID).Set(0)
		return err
	}

	// Update status
	status.Status = SyncStateInSync
	status.LastSuccessTime = time.Now()
	status.SyncedResources = syncedResources
	status.Metrics = SyncMetrics{
		Duration:         time.Since(status.LastSyncTime),
		ResourcesTotal:   len(manifests),
		ResourcesApplied: appliedCount,
		ResourcesFailed:  failedCount,
	}

	// Update metrics
	csm.metrics.syncOperations.WithLabelValues(clusterID, repo.ID, "success").Inc()
	csm.metrics.syncStatus.WithLabelValues(clusterID, repo.ID).Set(1)
	csm.metrics.resourcesTotal.WithLabelValues(clusterID, repo.ID, "applied").Set(float64(appliedCount))
	csm.metrics.resourcesTotal.WithLabelValues(clusterID, repo.ID, "failed").Set(float64(failedCount))

	// Update repository last sync
	csm.mu.Lock()
	repo.LastSync = time.Now()
	csm.mu.Unlock()

	csm.logger.Info("Successfully synced repository to cluster",
		"repo", repo.ID,
		"cluster", clusterID,
		"applied", appliedCount,
		"failed", failedCount)

	return nil
}

func (csm *ConfigSyncManager) validateRepository(ctx context.Context, repo *GitRepository) error {
	// Validate URL
	if _, err := url.Parse(repo.URL); err != nil {
		return fmt.Errorf("invalid repository URL: %w", err)
	}

	// Validate branch
	if repo.Branch == "" {
		repo.Branch = "main"
	}

	// Validate sync policy
	if repo.SyncPolicy.SyncInterval <= 0 {
		repo.SyncPolicy.SyncInterval = 5 * time.Minute
	}

	if repo.SyncPolicy.RetryLimit <= 0 {
		repo.SyncPolicy.RetryLimit = 3
	}

	if repo.SyncPolicy.Timeout <= 0 {
		repo.SyncPolicy.Timeout = 10 * time.Minute
	}

	return nil
}

func (csm *ConfigSyncManager) testRepositoryAccess(ctx context.Context, repo *GitRepository) error {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", "repo-test-")
	if err != nil {
		return fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Clone repository
	auth, err := csm.getGitAuth(ctx, repo.Auth)
	if err != nil {
		return fmt.Errorf("failed to get git auth: %w", err)
	}

	_, err = git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:           repo.URL,
		ReferenceName: plumbing.NewBranchReferenceName(repo.Branch),
		SingleBranch:  true,
		Depth:         1,
		Auth:          auth,
	})

	if err != nil {
		return fmt.Errorf("failed to clone repository: %w", err)
	}

	return nil
}

func (csm *ConfigSyncManager) cloneRepository(ctx context.Context, repo *GitRepository) (string, error) {
	// Create temporary directory
	tempDir, err := os.MkdirTemp("", fmt.Sprintf("repo-%s-", repo.ID))
	if err != nil {
		return "", fmt.Errorf("failed to create temp directory: %w", err)
	}

	// Get authentication
	auth, err := csm.getGitAuth(ctx, repo.Auth)
	if err != nil {
		return "", fmt.Errorf("failed to get git auth: %w", err)
	}

	// Clone repository
	r, err := git.PlainClone(tempDir, false, &git.CloneOptions{
		URL:           repo.URL,
		ReferenceName: plumbing.NewBranchReferenceName(repo.Branch),
		SingleBranch:  true,
		Auth:          auth,
	})

	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to clone repository: %w", err)
	}

	// Get latest commit hash
	ref, err := r.Head()
	if err != nil {
		os.RemoveAll(tempDir)
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	repo.LastCommit = ref.Hash().String()

	return tempDir, nil
}

func (csm *ConfigSyncManager) getGitAuth(ctx context.Context, auth GitAuth) (interface{}, error) {
	switch auth.Type {
	case "token":
		return &http.BasicAuth{
			Username: "token",
			Password: auth.Token,
		}, nil
	case "basic":
		return &http.BasicAuth{
			Username: auth.Username,
			Password: auth.Password,
		}, nil
	case "secret":
		// Get auth from Kubernetes secret
		secret := &corev1.Secret{}
		if err := csm.client.Get(ctx, client.ObjectKey{
			Namespace: "nephio-system",
			Name:      auth.SecretRef,
		}, secret); err != nil {
			return nil, fmt.Errorf("failed to get auth secret: %w", err)
		}

		username := string(secret.Data["username"])
		password := string(secret.Data["password"])
		token := string(secret.Data["token"])

		if token != "" {
			return &http.BasicAuth{
				Username: "token",
				Password: token,
			}, nil
		}

		return &http.BasicAuth{
			Username: username,
			Password: password,
		}, nil
	default:
		return nil, nil
	}
}

func (csm *ConfigSyncManager) readManifests(repoDir, path string) ([]*unstructured.Unstructured, error) {
	manifestPath := filepath.Join(repoDir, path)
	manifests := []*unstructured.Unstructured{}

	err := filepath.Walk(manifestPath, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		if !strings.HasSuffix(filePath, ".yaml") && !strings.HasSuffix(filePath, ".yml") {
			return nil
		}

		data, err := os.ReadFile(filePath)
		if err != nil {
			return fmt.Errorf("failed to read file %s: %w", filePath, err)
		}

		// Split multi-document YAML
		docs := strings.Split(string(data), "---")
		for _, doc := range docs {
			doc = strings.TrimSpace(doc)
			if doc == "" {
				continue
			}

			var obj unstructured.Unstructured
			if err := yaml.Unmarshal([]byte(doc), &obj); err != nil {
				csm.logger.Error(err, "Failed to unmarshal YAML", "file", filePath)
				continue
			}

			if obj.GetKind() != "" {
				manifests = append(manifests, &obj)
			}
		}

		return nil
	})

	return manifests, err
}

func (csm *ConfigSyncManager) applyManifests(ctx context.Context, clusterClient client.Client, manifests []*unstructured.Unstructured, policy SyncPolicy) ([]SyncedResource, int, int, error) {
	syncedResources := []SyncedResource{}
	appliedCount := 0
	failedCount := 0

	for _, manifest := range manifests {
		// Validate manifest if enabled
		if policy.ValidateManifests {
			if err := csm.validateManifest(manifest); err != nil {
				csm.logger.Error(err, "Manifest validation failed", "kind", manifest.GetKind(), "name", manifest.GetName())
				failedCount++
				continue
			}
		}

		// Calculate resource hash
		hash := csm.calculateResourceHash(manifest)

		if policy.DryRun {
			// Dry run - just log what would be applied
			csm.logger.Info("DRY RUN: Would apply resource",
				"kind", manifest.GetKind(),
				"name", manifest.GetName(),
				"namespace", manifest.GetNamespace())
		} else {
			// Apply manifest
			if err := clusterClient.Patch(ctx, manifest, client.Apply, client.ForceOwnership, client.FieldOwner("nephio-config-sync")); err != nil {
				csm.logger.Error(err, "Failed to apply manifest", "kind", manifest.GetKind(), "name", manifest.GetName())
				failedCount++
				continue
			}
		}

		syncedResource := SyncedResource{
			Group:       manifest.GetObjectKind().GroupVersionKind().Group,
			Version:     manifest.GetObjectKind().GroupVersionKind().Version,
			Kind:        manifest.GetKind(),
			Namespace:   manifest.GetNamespace(),
			Name:        manifest.GetName(),
			Status:      "synced",
			LastUpdated: time.Now(),
			Hash:        hash,
		}

		syncedResources = append(syncedResources, syncedResource)
		appliedCount++
	}

	return syncedResources, appliedCount, failedCount, nil
}

func (csm *ConfigSyncManager) validateManifest(manifest *unstructured.Unstructured) error {
	// Basic validation
	if manifest.GetKind() == "" {
		return fmt.Errorf("manifest missing kind")
	}
	if manifest.GetName() == "" {
		return fmt.Errorf("manifest missing name")
	}

	// Additional validations can be added here
	return nil
}

func (csm *ConfigSyncManager) calculateResourceHash(manifest *unstructured.Unstructured) string {
	data, _ := json.Marshal(manifest.Object)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (csm *ConfigSyncManager) runDriftDetection(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-csm.stopCh:
			return
		case <-ticker.C:
			csm.performDriftDetection(ctx)
		}
	}
}

func (csm *ConfigSyncManager) performDriftDetection(ctx context.Context) {
	clusters := csm.registry.ListClusters()

	for _, cluster := range clusters {
		drifts, err := csm.driftDetector.DetectDrift(ctx, cluster.Metadata.ID)
		if err != nil {
			csm.logger.Error(err, "Drift detection failed", "cluster", cluster.Metadata.ID)
			continue
		}

		for _, drift := range drifts {
			csm.metrics.driftDetections.WithLabelValues(
				cluster.Metadata.ID,
				drift.Resource,
				drift.Severity,
			).Inc()
		}
	}
}

func (csm *ConfigSyncManager) runPolicyValidation(ctx context.Context) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-csm.stopCh:
			return
		case <-ticker.C:
			csm.performPolicyValidation(ctx)
		}
	}
}

func (csm *ConfigSyncManager) performPolicyValidation(ctx context.Context) {
	clusters := csm.registry.ListClusters()
	policies := csm.policyManager.ListPolicies()

	for _, cluster := range clusters {
		for _, policy := range policies {
			violations, err := csm.policyManager.ValidatePolicy(ctx, policy.ID, cluster.Metadata.ID)
			if err != nil {
				csm.logger.Error(err, "Policy validation failed", "cluster", cluster.Metadata.ID, "policy", policy.ID)
				continue
			}

			for _, violation := range violations {
				csm.metrics.policyViolations.WithLabelValues(
					cluster.Metadata.ID,
					policy.ID,
					violation.Severity,
				).Inc()
			}
		}
	}
}

func (csm *ConfigSyncManager) runComplianceAssessment(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-csm.stopCh:
			return
		case <-ticker.C:
			csm.performComplianceAssessment(ctx)
		}
	}
}

func (csm *ConfigSyncManager) performComplianceAssessment(ctx context.Context) {
	clusters := csm.registry.ListClusters()
	standards := csm.complianceTracker.ListStandards()

	for _, cluster := range clusters {
		for _, standard := range standards {
			assessment, err := csm.complianceTracker.AssessCompliance(ctx, cluster.Metadata.ID, standard.Name)
			if err != nil {
				csm.logger.Error(err, "Compliance assessment failed", "cluster", cluster.Metadata.ID, "standard", standard.Name)
				continue
			}

			csm.metrics.complianceScore.WithLabelValues(
				cluster.Metadata.ID,
				standard.Name,
			).Set(assessment.Score)
		}
	}
}

// PolicyManager methods

// CreatePolicy creates a new policy
func (pm *PolicyManager) CreatePolicy(policy *Policy) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if _, exists := pm.policies[policy.ID]; exists {
		return fmt.Errorf("policy %s already exists", policy.ID)
	}

	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()

	pm.policies[policy.ID] = policy
	pm.logger.Info("Created policy", "id", policy.ID, "name", policy.Name)

	return nil
}

// ValidatePolicy validates a policy against a cluster
func (pm *PolicyManager) ValidatePolicy(ctx context.Context, policyID, clusterID string) ([]*PolicyViolation, error) {
	pm.mu.RLock()
	policy, exists := pm.policies[policyID]
	pm.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("policy %s not found", policyID)
	}

	violations := []*PolicyViolation{}

	// Implementation would evaluate policy rules against cluster resources
	// This is a placeholder
	for _, rule := range policy.Rules {
		// Evaluate rule condition
		violated := pm.evaluateRule(ctx, rule, clusterID)
		if violated {
			violation := &PolicyViolation{
				PolicyID:   policyID,
				ClusterID:  clusterID,
				Violation:  fmt.Sprintf("Rule %s violated", rule.Name),
				Severity:   rule.Severity,
				DetectedAt: time.Now(),
				Status:     "open",
			}
			violations = append(violations, violation)
		}
	}

	pm.mu.Lock()
	pm.violations[clusterID] = append(pm.violations[clusterID], violations...)
	pm.mu.Unlock()

	return violations, nil
}

// ListPolicies lists all policies
func (pm *PolicyManager) ListPolicies() []*Policy {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	policies := make([]*Policy, 0, len(pm.policies))
	for _, policy := range pm.policies {
		policies = append(policies, policy)
	}

	return policies
}

func (pm *PolicyManager) evaluateRule(ctx context.Context, rule PolicyRule, clusterID string) bool {
	// Implementation would evaluate the rule condition
	// This is a placeholder that randomly returns violations for demonstration
	return false
}

// DriftDetector methods

// DetectDrift detects configuration drift for a cluster
func (dd *DriftDetector) DetectDrift(ctx context.Context, clusterID string) ([]*DriftDetection, error) {
	dd.mu.RLock()
	rules := make([]*DriftRule, 0, len(dd.detectionRules))
	for _, rule := range dd.detectionRules {
		rules = append(rules, rule)
	}
	dd.mu.RUnlock()

	detections := []*DriftDetection{}

	// Implementation would compare expected vs actual configuration
	// This is a placeholder

	dd.mu.Lock()
	dd.drift[clusterID] = detections
	dd.mu.Unlock()

	return detections, nil
}

// ComplianceTracker methods

// AssessCompliance assesses compliance for a cluster
func (ct *ComplianceTracker) AssessCompliance(ctx context.Context, clusterID, standard string) (*ComplianceAssessment, error) {
	ct.mu.RLock()
	std, exists := ct.standards[standard]
	ct.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("compliance standard %s not found", standard)
	}

	assessment := &ComplianceAssessment{
		ID:        fmt.Sprintf("%s-%s-%d", clusterID, standard, time.Now().Unix()),
		ClusterID: clusterID,
		Standard:  standard,
		Results:   []ComplianceResult{},
		CreatedAt: time.Now(),
	}

	// Evaluate each control
	passedControls := 0
	for _, control := range std.Controls {
		result := ComplianceResult{
			ControlID: control.ID,
			Timestamp: time.Now(),
		}

		// Implementation would evaluate the control
		// This is a placeholder
		if ct.evaluateControl(ctx, control, clusterID) {
			result.Status = "pass"
			result.Message = "Control satisfied"
			passedControls++
		} else {
			result.Status = "fail"
			result.Message = "Control not satisfied"
		}

		assessment.Results = append(assessment.Results, result)
	}

	// Calculate score
	assessment.Score = float64(passedControls) / float64(len(std.Controls)) * 100

	if assessment.Score >= 90 {
		assessment.Status = "compliant"
	} else if assessment.Score >= 70 {
		assessment.Status = "partially-compliant"
	} else {
		assessment.Status = "non-compliant"
	}

	ct.mu.Lock()
	ct.assessments[assessment.ID] = assessment
	ct.mu.Unlock()

	return assessment, nil
}

// ListStandards lists all compliance standards
func (ct *ComplianceTracker) ListStandards() []*ComplianceStandard {
	ct.mu.RLock()
	defer ct.mu.RUnlock()

	standards := make([]*ComplianceStandard, 0, len(ct.standards))
	for _, standard := range ct.standards {
		standards = append(standards, standard)
	}

	return standards
}

func (ct *ComplianceTracker) evaluateControl(ctx context.Context, control ComplianceControl, clusterID string) bool {
	// Implementation would evaluate the control checks
	// This is a placeholder that randomly returns compliance status
	return true
}
