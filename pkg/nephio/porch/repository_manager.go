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

	"fmt"

	"net/url"

	"strings"

	"sync"

	"time"



	"github.com/go-logr/logr"

	"github.com/prometheus/client_golang/prometheus"



	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"



	"sigs.k8s.io/controller-runtime/pkg/log"

)



// repositoryManager implements the RepositoryManager interface.

type repositoryManager struct {

	// Parent client for API operations.

	client *Client



	// Logger for repository operations.

	logger logr.Logger



	// Metrics for repository operations.

	metrics *RepositoryMetrics



	// Repository state tracking.

	repositories map[string]*repositoryState

	stateMutex   sync.RWMutex



	// Background sync management.

	syncCancel context.CancelFunc

	syncWg     sync.WaitGroup



	// Health check management.

	healthCancel context.CancelFunc

	healthWg     sync.WaitGroup



	// Git client factory for repository operations.

	gitClientFactory GitClientFactory



	// Credential store for repository authentication.

	credentialStore CredentialStore



	// Repository validation rules.

	validationRules []RepositoryValidationRule



	// Event handlers for repository lifecycle events.

	eventHandlers []RepositoryEventHandler

}



// repositoryState tracks the runtime state of a repository.

type repositoryState struct {

	// Repository configuration.

	config *RepositoryConfig



	// Current sync status.

	lastSync       time.Time

	syncError      error

	syncInProgress bool



	// Health status.

	health          RepositoryHealth

	healthError     error

	lastHealthCheck time.Time



	// Branch information.

	branches      []string

	defaultBranch string



	// Package count and metadata.

	packageCount int32

	lastUpdated  time.Time



	// Authentication status.

	authValid   bool

	authChecked time.Time



	// Performance metrics.

	syncDuration time.Duration

	syncCount    int64

	errorCount   int64



	// Mutex for thread-safe access.

	mutex sync.RWMutex

}



// RepositoryMetrics defines Prometheus metrics for repository operations.

type RepositoryMetrics struct {

	syncOperations       *prometheus.CounterVec

	syncDuration         *prometheus.HistogramVec

	syncErrors           *prometheus.CounterVec

	repositoryHealth     *prometheus.GaugeVec

	packageCount         *prometheus.GaugeVec

	authenticationStatus *prometheus.GaugeVec

	branchCount          *prometheus.GaugeVec

}



// GitClientFactory creates Git clients for repository operations.

type GitClientFactory interface {

	CreateClient(ctx context.Context, config *RepositoryConfig) (GitClient, error)

	ValidateRepository(ctx context.Context, config *RepositoryConfig) error

}



// GitClient provides Git operations for repositories.

type GitClient interface {

	Clone(ctx context.Context, targetDir string) error

	Pull(ctx context.Context) error

	ListBranches(ctx context.Context) ([]string, error)

	CreateBranch(ctx context.Context, branchName, baseBranch string) error

	DeleteBranch(ctx context.Context, branchName string) error

	GetCommitHash(ctx context.Context) (string, error)

	GetDefaultBranch(ctx context.Context) (string, error)

	ValidateAccess(ctx context.Context) error

}



// CredentialStore manages repository credentials.

type CredentialStore interface {

	StoreCredentials(ctx context.Context, repoName string, creds *Credentials) error

	GetCredentials(ctx context.Context, repoName string) (*Credentials, error)

	DeleteCredentials(ctx context.Context, repoName string) error

	ValidateCredentials(ctx context.Context, repoName string) error

}



// RepositoryValidationRule defines a validation rule for repositories.

type RepositoryValidationRule interface {

	Validate(ctx context.Context, config *RepositoryConfig) error

	GetName() string

}



// RepositoryEventHandler handles repository lifecycle events.

type RepositoryEventHandler interface {

	OnRepositoryRegistered(ctx context.Context, repo *Repository)

	OnRepositoryUnregistered(ctx context.Context, repoName string)

	OnRepositorySynced(ctx context.Context, repoName string, result *SyncResult)

	OnRepositoryError(ctx context.Context, repoName string, err error)

}



// NewRepositoryManager creates a new repository manager.

func NewRepositoryManager(client *Client) RepositoryManager {

	return &repositoryManager{

		client:       client,

		logger:       log.Log.WithName("repository-manager"),

		repositories: make(map[string]*repositoryState),

		metrics:      initRepositoryMetrics(),

	}

}



// RegisterRepository registers a new repository for management.

func (rm *repositoryManager) RegisterRepository(ctx context.Context, config *RepositoryConfig) (*Repository, error) {

	rm.logger.Info("Registering repository", "name", config.Name, "url", config.URL)



	// Validate configuration.

	if err := rm.validateRepositoryConfig(ctx, config); err != nil {

		return nil, fmt.Errorf("repository configuration validation failed: %w", err)

	}



	// Check if repository already exists.

	rm.stateMutex.RLock()

	_, exists := rm.repositories[config.Name]

	rm.stateMutex.RUnlock()



	if exists {

		return nil, fmt.Errorf("repository %s already registered", config.Name)

	}



	// Create repository resource.

	repo := &Repository{

		ObjectMeta: metav1.ObjectMeta{

			Name: config.Name,

			Labels: map[string]string{

				LabelComponent:  "repository",

				LabelRepository: config.Name,

			},

			Annotations: map[string]string{

				AnnotationManagedBy: "nephoran-porch-client",

			},

		},

		Spec: RepositorySpec{

			Type:         config.Type,

			URL:          config.URL,

			Branch:       config.Branch,

			Directory:    config.Directory,

			Auth:         config.Auth,

			Sync:         config.Sync,

			Capabilities: config.Capabilities,

		},

		Status: RepositoryStatus{

			Conditions: []metav1.Condition{

				{

					Type:    "Registered",

					Status:  metav1.ConditionTrue,

					Reason:  "RepositoryRegistered",

					Message: "Repository successfully registered",

				},

			},

			Health: RepositoryHealthUnknown,

		},

	}



	// Create repository using Porch client.

	created, err := rm.client.CreateRepository(ctx, repo)

	if err != nil {

		return nil, fmt.Errorf("failed to create repository in Porch: %w", err)

	}



	// Initialize repository state.

	state := &repositoryState{

		config:    config,

		health:    RepositoryHealthUnknown,

		lastSync:  time.Time{},

		branches:  []string{},

		authValid: false,

	}



	rm.stateMutex.Lock()

	rm.repositories[config.Name] = state

	rm.stateMutex.Unlock()



	// Start background sync if configured.

	if config.Sync != nil && config.Sync.AutoSync {

		go rm.startRepositorySync(ctx, config.Name)

	}



	// Start health monitoring.

	go rm.startHealthMonitoring(ctx, config.Name)



	// Trigger initial validation.

	go func() {

		if err := rm.validateRepositoryAccess(ctx, config.Name); err != nil {

			rm.logger.Error(err, "Initial repository access validation failed", "name", config.Name)

		}

	}()



	// Update metrics.

	if rm.metrics != nil {

		rm.metrics.repositoryHealth.WithLabelValues(config.Name).Set(0) // Unknown

	}



	// Notify event handlers.

	for _, handler := range rm.eventHandlers {

		handler.OnRepositoryRegistered(ctx, created)

	}



	rm.logger.Info("Successfully registered repository", "name", config.Name, "uid", created.UID)

	return created, nil

}



// UnregisterRepository unregisters a repository from management.

func (rm *repositoryManager) UnregisterRepository(ctx context.Context, name string) error {

	rm.logger.Info("Unregistering repository", "name", name)



	// Check if repository exists.

	rm.stateMutex.RLock()

	_, exists := rm.repositories[name]

	rm.stateMutex.RUnlock()



	if !exists {

		return fmt.Errorf("repository %s not found", name)

	}



	// Delete repository from Porch.

	if err := rm.client.DeleteRepository(ctx, name); err != nil {

		return fmt.Errorf("failed to delete repository from Porch: %w", err)

	}



	// Stop background operations.

	rm.stopRepositoryOperations(name)



	// Clean up credentials.

	if rm.credentialStore != nil {

		if err := rm.credentialStore.DeleteCredentials(ctx, name); err != nil {

			rm.logger.Error(err, "Failed to delete repository credentials", "name", name)

		}

	}



	// Remove from state tracking.

	rm.stateMutex.Lock()

	delete(rm.repositories, name)

	rm.stateMutex.Unlock()



	// Clean up metrics.

	if rm.metrics != nil {

		rm.metrics.repositoryHealth.DeleteLabelValues(name)

		rm.metrics.packageCount.DeleteLabelValues(name)

		rm.metrics.authenticationStatus.DeleteLabelValues(name)

		rm.metrics.branchCount.DeleteLabelValues(name)

	}



	// Notify event handlers.

	for _, handler := range rm.eventHandlers {

		handler.OnRepositoryUnregistered(ctx, name)

	}



	rm.logger.Info("Successfully unregistered repository", "name", name)

	return nil

}



// SynchronizeRepository performs a manual synchronization of a repository.

func (rm *repositoryManager) SynchronizeRepository(ctx context.Context, name string) (*SyncResult, error) {

	rm.logger.V(1).Info("Synchronizing repository", "name", name)



	// Get repository state.

	rm.stateMutex.RLock()

	state, exists := rm.repositories[name]

	rm.stateMutex.RUnlock()



	if !exists {

		return nil, fmt.Errorf("repository %s not found", name)

	}



	// Check if sync is already in progress.

	state.mutex.RLock()

	if state.syncInProgress {

		state.mutex.RUnlock()

		return nil, fmt.Errorf("synchronization already in progress for repository %s", name)

	}

	state.mutex.RUnlock()



	// Perform synchronization.

	result, err := rm.performRepositorySync(ctx, name, state)

	if err != nil {

		return nil, fmt.Errorf("synchronization failed for repository %s: %w", name, err)

	}



	// Update repository status in Porch.

	if err := rm.updateRepositoryStatus(ctx, name, result); err != nil {

		rm.logger.Error(err, "Failed to update repository status", "name", name)

	}



	// Notify event handlers.

	for _, handler := range rm.eventHandlers {

		handler.OnRepositorySynced(ctx, name, result)

	}



	rm.logger.Info("Successfully synchronized repository", "name", name, "packages", result.PackageCount)

	return result, nil

}



// GetRepositoryHealth returns the current health status of a repository.

func (rm *repositoryManager) GetRepositoryHealth(ctx context.Context, name string) (*RepositoryHealth, error) {

	rm.stateMutex.RLock()

	state, exists := rm.repositories[name]

	rm.stateMutex.RUnlock()



	if !exists {

		return nil, fmt.Errorf("repository %s not found", name)

	}



	state.mutex.RLock()

	health := state.health

	state.mutex.RUnlock()



	return &health, nil

}



// CreateBranch creates a new branch in the repository.

func (rm *repositoryManager) CreateBranch(ctx context.Context, repoName, branchName, baseBranch string) error {

	rm.logger.V(1).Info("Creating branch", "repo", repoName, "branch", branchName, "base", baseBranch)



	state, exists := rm.getRepositoryState(repoName)

	if !exists {

		return fmt.Errorf("repository %s not found", repoName)

	}



	// Create Git client.

	gitClient, err := rm.createGitClient(ctx, state.config)

	if err != nil {

		return fmt.Errorf("failed to create Git client: %w", err)

	}



	// Create branch.

	if err := gitClient.CreateBranch(ctx, branchName, baseBranch); err != nil {

		return fmt.Errorf("failed to create branch %s: %w", branchName, err)

	}



	// Update branch list.

	rm.updateBranchList(ctx, repoName)



	rm.logger.Info("Successfully created branch", "repo", repoName, "branch", branchName)

	return nil

}



// DeleteBranch deletes a branch from the repository.

func (rm *repositoryManager) DeleteBranch(ctx context.Context, repoName, branchName string) error {

	rm.logger.V(1).Info("Deleting branch", "repo", repoName, "branch", branchName)



	state, exists := rm.getRepositoryState(repoName)

	if !exists {

		return fmt.Errorf("repository %s not found", repoName)

	}



	// Prevent deletion of default branch.

	state.mutex.RLock()

	defaultBranch := state.defaultBranch

	state.mutex.RUnlock()



	if branchName == defaultBranch {

		return fmt.Errorf("cannot delete default branch %s", branchName)

	}



	// Create Git client.

	gitClient, err := rm.createGitClient(ctx, state.config)

	if err != nil {

		return fmt.Errorf("failed to create Git client: %w", err)

	}



	// Delete branch.

	if err := gitClient.DeleteBranch(ctx, branchName); err != nil {

		return fmt.Errorf("failed to delete branch %s: %w", branchName, err)

	}



	// Update branch list.

	rm.updateBranchList(ctx, repoName)



	rm.logger.Info("Successfully deleted branch", "repo", repoName, "branch", branchName)

	return nil

}



// ListBranches returns the list of branches in the repository.

func (rm *repositoryManager) ListBranches(ctx context.Context, repoName string) ([]string, error) {

	state, exists := rm.getRepositoryState(repoName)

	if !exists {

		return nil, fmt.Errorf("repository %s not found", repoName)

	}



	state.mutex.RLock()

	branches := make([]string, len(state.branches))

	copy(branches, state.branches)

	state.mutex.RUnlock()



	// Refresh branch list if it's stale.

	if len(branches) == 0 || time.Since(state.lastUpdated) > 5*time.Minute {

		if err := rm.updateBranchList(ctx, repoName); err != nil {

			rm.logger.Error(err, "Failed to update branch list", "repo", repoName)

		} else {

			state.mutex.RLock()

			branches = make([]string, len(state.branches))

			copy(branches, state.branches)

			state.mutex.RUnlock()

		}

	}



	return branches, nil

}



// UpdateCredentials updates the authentication credentials for a repository.

func (rm *repositoryManager) UpdateCredentials(ctx context.Context, repoName string, creds *Credentials) error {

	rm.logger.V(1).Info("Updating credentials", "repo", repoName)



	state, exists := rm.getRepositoryState(repoName)

	if !exists {

		return fmt.Errorf("repository %s not found", repoName)

	}



	// Validate credentials format.

	if err := rm.validateCredentials(creds); err != nil {

		return fmt.Errorf("invalid credentials: %w", err)

	}



	// Store credentials.

	if rm.credentialStore != nil {

		if err := rm.credentialStore.StoreCredentials(ctx, repoName, creds); err != nil {

			return fmt.Errorf("failed to store credentials: %w", err)

		}

	}



	// Update repository configuration.

	state.mutex.Lock()

	if state.config.Auth == nil {

		state.config.Auth = &AuthConfig{}

	}

	state.config.Auth.Type = creds.Type

	state.config.Auth.Username = creds.Username

	state.config.Auth.Headers = creds.Headers

	// Don't store sensitive data in memory.

	state.authValid = false

	state.authChecked = time.Time{}

	state.mutex.Unlock()



	// Validate new credentials.

	go func() {

		if err := rm.ValidateAccess(ctx, repoName); err != nil {

			rm.logger.Error(err, "New credentials validation failed", "repo", repoName)

		}

	}()



	rm.logger.Info("Successfully updated credentials", "repo", repoName)

	return nil

}



// ValidateAccess validates that the repository can be accessed with current credentials.

func (rm *repositoryManager) ValidateAccess(ctx context.Context, repoName string) error {

	state, exists := rm.getRepositoryState(repoName)

	if !exists {

		return fmt.Errorf("repository %s not found", repoName)

	}



	// Create Git client.

	gitClient, err := rm.createGitClient(ctx, state.config)

	if err != nil {

		return fmt.Errorf("failed to create Git client: %w", err)

	}



	// Validate access.

	err = gitClient.ValidateAccess(ctx)



	// Update authentication status.

	state.mutex.Lock()

	state.authValid = (err == nil)

	state.authChecked = time.Now()

	if err != nil {

		state.syncError = err

	}

	state.mutex.Unlock()



	// Update metrics.

	if rm.metrics != nil {

		authStatus := float64(0)

		if err == nil {

			authStatus = 1

		}

		rm.metrics.authenticationStatus.WithLabelValues(repoName).Set(authStatus)

	}



	return err

}



// Private helper methods.



// validateRepositoryConfig validates repository configuration.

func (rm *repositoryManager) validateRepositoryConfig(ctx context.Context, config *RepositoryConfig) error {

	if config.Name == "" {

		return fmt.Errorf("repository name is required")

	}



	if config.URL == "" {

		return fmt.Errorf("repository URL is required")

	}



	if config.Type == "" {

		return fmt.Errorf("repository type is required")

	}



	// Validate URL format.

	parsedURL, err := url.Parse(config.URL)

	if err != nil {

		return fmt.Errorf("invalid repository URL: %w", err)

	}



	// Validate repository type and URL compatibility.

	switch config.Type {

	case "git":

		if !strings.HasPrefix(config.URL, "https://") && !strings.HasPrefix(config.URL, "git@") {

			return fmt.Errorf("git repository URL must use https:// or git@ protocol")

		}

	case "oci":

		if parsedURL.Scheme != "oci" && parsedURL.Scheme != "https" {

			return fmt.Errorf("OCI repository URL must use oci:// or https:// scheme")

		}

	default:

		return fmt.Errorf("unsupported repository type: %s", config.Type)

	}



	// Apply validation rules.

	for _, rule := range rm.validationRules {

		if err := rule.Validate(ctx, config); err != nil {

			return fmt.Errorf("validation rule %s failed: %w", rule.GetName(), err)

		}

	}



	return nil

}



// validateCredentials validates credential format and content.

func (rm *repositoryManager) validateCredentials(creds *Credentials) error {

	if creds.Type == "" {

		return fmt.Errorf("credential type is required")

	}



	switch creds.Type {

	case "basic":

		if creds.Username == "" || creds.Password == "" {

			return fmt.Errorf("username and password are required for basic auth")

		}

	case "token":

		if creds.Token == "" {

			return fmt.Errorf("token is required for token auth")

		}

	case "ssh":

		if len(creds.PrivateKey) == 0 {

			return fmt.Errorf("private key is required for SSH auth")

		}

	default:

		return fmt.Errorf("unsupported credential type: %s", creds.Type)

	}



	return nil

}



// getRepositoryState safely retrieves repository state.

func (rm *repositoryManager) getRepositoryState(name string) (*repositoryState, bool) {

	rm.stateMutex.RLock()

	defer rm.stateMutex.RUnlock()

	state, exists := rm.repositories[name]

	return state, exists

}



// createGitClient creates a Git client for the repository.

func (rm *repositoryManager) createGitClient(ctx context.Context, config *RepositoryConfig) (GitClient, error) {

	if rm.gitClientFactory != nil {

		return rm.gitClientFactory.CreateClient(ctx, config)

	}

	return nil, fmt.Errorf("git client factory not configured")

}



// performRepositorySync performs the actual repository synchronization.

func (rm *repositoryManager) performRepositorySync(ctx context.Context, name string, state *repositoryState) (*SyncResult, error) {

	start := time.Now()



	// Mark sync as in progress.

	state.mutex.Lock()

	state.syncInProgress = true

	state.mutex.Unlock()



	defer func() {

		state.mutex.Lock()

		state.syncInProgress = false

		state.lastSync = time.Now()

		state.syncDuration = time.Since(start)

		state.syncCount++

		state.mutex.Unlock()

	}()



	result := &SyncResult{

		SyncTime: time.Now(),

		Success:  false,

	}



	// Create Git client.

	gitClient, err := rm.createGitClient(ctx, state.config)

	if err != nil {

		result.Error = fmt.Sprintf("failed to create Git client: %v", err)

		state.mutex.Lock()

		state.syncError = err

		state.errorCount++

		state.mutex.Unlock()

		return result, err

	}



	// Get commit hash for comparison.

	commitHash, err := gitClient.GetCommitHash(ctx)

	if err != nil {

		result.Error = fmt.Sprintf("failed to get commit hash: %v", err)

		state.mutex.Lock()

		state.syncError = err

		state.errorCount++

		state.mutex.Unlock()

		return result, err

	}



	result.CommitHash = commitHash

	result.Success = true

	result.PackageCount = state.packageCount // This would be updated from actual package discovery



	// Update state.

	state.mutex.Lock()

	state.syncError = nil

	state.health = RepositoryHealthHealthy

	state.mutex.Unlock()



	// Update metrics.

	if rm.metrics != nil {

		rm.metrics.syncOperations.WithLabelValues(name, "success").Inc()

		rm.metrics.syncDuration.WithLabelValues(name).Observe(time.Since(start).Seconds())

		rm.metrics.repositoryHealth.WithLabelValues(name).Set(1) // Healthy

	}



	return result, nil

}



// updateRepositoryStatus updates the repository status in Porch.

func (rm *repositoryManager) updateRepositoryStatus(ctx context.Context, name string, result *SyncResult) error {

	repo, err := rm.client.GetRepository(ctx, name)

	if err != nil {

		return fmt.Errorf("failed to get repository: %w", err)

	}



	// Update status.

	repo.Status.LastSyncTime = &metav1.Time{Time: result.SyncTime}

	repo.Status.CommitHash = result.CommitHash

	repo.Status.PackageCount = result.PackageCount



	if result.Success {

		repo.Status.SyncError = ""

		repo.Status.Health = RepositoryHealthHealthy

		repo.SetCondition(metav1.Condition{

			Type:    "Synced",

			Status:  metav1.ConditionTrue,

			Reason:  "SyncSuccessful",

			Message: "Repository successfully synchronized",

		})

	} else {

		repo.Status.SyncError = result.Error

		repo.Status.Health = RepositoryHealthUnhealthy

		repo.SetCondition(metav1.Condition{

			Type:    "Synced",

			Status:  metav1.ConditionFalse,

			Reason:  "SyncFailed",

			Message: result.Error,

		})

	}



	_, err = rm.client.UpdateRepository(ctx, repo)

	return err

}



// updateBranchList updates the cached branch list for a repository.

func (rm *repositoryManager) updateBranchList(ctx context.Context, repoName string) error {

	state, exists := rm.getRepositoryState(repoName)

	if !exists {

		return fmt.Errorf("repository %s not found", repoName)

	}



	gitClient, err := rm.createGitClient(ctx, state.config)

	if err != nil {

		return fmt.Errorf("failed to create Git client: %w", err)

	}



	branches, err := gitClient.ListBranches(ctx)

	if err != nil {

		return fmt.Errorf("failed to list branches: %w", err)

	}



	defaultBranch, err := gitClient.GetDefaultBranch(ctx)

	if err != nil {

		rm.logger.V(1).Info("Failed to get default branch, using first branch", "repo", repoName, "error", err)

		if len(branches) > 0 {

			defaultBranch = branches[0]

		}

	}



	state.mutex.Lock()

	state.branches = branches

	state.defaultBranch = defaultBranch

	state.lastUpdated = time.Now()

	state.mutex.Unlock()



	// Update metrics.

	if rm.metrics != nil {

		rm.metrics.branchCount.WithLabelValues(repoName).Set(float64(len(branches)))

	}



	return nil

}



// startRepositorySync starts background synchronization for a repository.

func (rm *repositoryManager) startRepositorySync(ctx context.Context, repoName string) {

	state, exists := rm.getRepositoryState(repoName)

	if !exists {

		rm.logger.Error(nil, "Repository not found for background sync", "repo", repoName)

		return

	}



	interval := 5 * time.Minute

	if state.config.Sync != nil && state.config.Sync.Interval != nil {

		interval = state.config.Sync.Interval.Duration

	}



	ticker := time.NewTicker(interval)

	defer ticker.Stop()



	rm.syncWg.Add(1)

	defer rm.syncWg.Done()



	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			if _, err := rm.SynchronizeRepository(ctx, repoName); err != nil {

				rm.logger.Error(err, "Background sync failed", "repo", repoName)



				// Notify error handlers.

				for _, handler := range rm.eventHandlers {

					handler.OnRepositoryError(ctx, repoName, err)

				}

			}

		}

	}

}



// startHealthMonitoring starts health monitoring for a repository.

func (rm *repositoryManager) startHealthMonitoring(ctx context.Context, repoName string) {

	interval := 1 * time.Minute

	ticker := time.NewTicker(interval)

	defer ticker.Stop()



	rm.healthWg.Add(1)

	defer rm.healthWg.Done()



	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			if err := rm.checkRepositoryHealth(ctx, repoName); err != nil {

				rm.logger.V(1).Info("Health check failed", "repo", repoName, "error", err)

			}

		}

	}

}



// checkRepositoryHealth performs a health check on a repository.

func (rm *repositoryManager) checkRepositoryHealth(ctx context.Context, repoName string) error {

	state, exists := rm.getRepositoryState(repoName)

	if !exists {

		return fmt.Errorf("repository %s not found", repoName)

	}



	// Simple health check - validate access.

	err := rm.ValidateAccess(ctx, repoName)



	health := RepositoryHealthHealthy

	if err != nil {

		health = RepositoryHealthUnhealthy

	}



	state.mutex.Lock()

	state.health = health

	state.healthError = err

	state.lastHealthCheck = time.Now()

	state.mutex.Unlock()



	// Update metrics.

	if rm.metrics != nil {

		healthValue := float64(1)

		if err != nil {

			healthValue = 0

		}

		rm.metrics.repositoryHealth.WithLabelValues(repoName).Set(healthValue)

	}



	return err

}



// stopRepositoryOperations stops background operations for a repository.

func (rm *repositoryManager) stopRepositoryOperations(repoName string) {

	// This would cancel context-specific operations.

	rm.logger.V(1).Info("Stopping repository operations", "repo", repoName)

}



// validateRepositoryAccess performs initial access validation.

func (rm *repositoryManager) validateRepositoryAccess(ctx context.Context, repoName string) error {

	return rm.ValidateAccess(ctx, repoName)

}



// initRepositoryMetrics initializes Prometheus metrics for repository operations.

func initRepositoryMetrics() *RepositoryMetrics {

	return &RepositoryMetrics{

		syncOperations: prometheus.NewCounterVec(

			prometheus.CounterOpts{

				Name: "porch_repository_sync_operations_total",

				Help: "Total number of repository sync operations",

			},

			[]string{"repository", "status"},

		),

		syncDuration: prometheus.NewHistogramVec(

			prometheus.HistogramOpts{

				Name:    "porch_repository_sync_duration_seconds",

				Help:    "Duration of repository sync operations",

				Buckets: prometheus.DefBuckets,

			},

			[]string{"repository"},

		),

		syncErrors: prometheus.NewCounterVec(

			prometheus.CounterOpts{

				Name: "porch_repository_sync_errors_total",

				Help: "Total number of repository sync errors",

			},

			[]string{"repository", "error_type"},

		),

		repositoryHealth: prometheus.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "porch_repository_health",

				Help: "Repository health status (1=healthy, 0=unhealthy)",

			},

			[]string{"repository"},

		),

		packageCount: prometheus.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "porch_repository_packages_total",

				Help: "Total number of packages in repository",

			},

			[]string{"repository"},

		),

		authenticationStatus: prometheus.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "porch_repository_auth_status",

				Help: "Repository authentication status (1=valid, 0=invalid)",

			},

			[]string{"repository"},

		),

		branchCount: prometheus.NewGaugeVec(

			prometheus.GaugeOpts{

				Name: "porch_repository_branches_total",

				Help: "Total number of branches in repository",

			},

			[]string{"repository"},

		),

	}

}



// GetRepositoryMetrics returns the repository metrics.

func (rm *repositoryManager) GetRepositoryMetrics() *RepositoryMetrics {

	return rm.metrics

}



// AddValidationRule adds a validation rule for repositories.

func (rm *repositoryManager) AddValidationRule(rule RepositoryValidationRule) {

	rm.validationRules = append(rm.validationRules, rule)

}



// AddEventHandler adds an event handler for repository lifecycle events.

func (rm *repositoryManager) AddEventHandler(handler RepositoryEventHandler) {

	rm.eventHandlers = append(rm.eventHandlers, handler)

}



// SetGitClientFactory sets the Git client factory.

func (rm *repositoryManager) SetGitClientFactory(factory GitClientFactory) {

	rm.gitClientFactory = factory

}



// SetCredentialStore sets the credential store.

func (rm *repositoryManager) SetCredentialStore(store CredentialStore) {

	rm.credentialStore = store

}



// Close gracefully shuts down the repository manager.

func (rm *repositoryManager) Close() error {

	rm.logger.Info("Shutting down repository manager")



	// Cancel background operations.

	if rm.syncCancel != nil {

		rm.syncCancel()

	}

	if rm.healthCancel != nil {

		rm.healthCancel()

	}



	// Wait for background goroutines to finish.

	rm.syncWg.Wait()

	rm.healthWg.Wait()



	// Clear state.

	rm.stateMutex.Lock()

	rm.repositories = make(map[string]*repositoryState)

	rm.stateMutex.Unlock()



	rm.logger.Info("Repository manager shut down complete")

	return nil

}

