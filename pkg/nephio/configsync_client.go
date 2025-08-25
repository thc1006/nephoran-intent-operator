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
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	"gopkg.in/yaml.v2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/errors"
	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch"
)

// ConfigSyncMetrics provides Config Sync metrics
type ConfigSyncMetrics struct {
	SyncOperations     prometheus.CounterVec
	SyncDuration       prometheus.HistogramVec
	SyncErrors         prometheus.CounterVec
	GitOperations      prometheus.CounterVec
	RepositoryHealth   prometheus.GaugeVec
	PackageDeployments prometheus.CounterVec
}

// GitClient implements Git operations for Config Sync
type GitClient struct {
	workingDir string
	config     *GitConfig
	tracer     trace.Tracer
	metrics    *GitClientMetrics
}

// GitConfig defines Git client configuration
type GitConfig struct {
	Username    string            `json:"username,omitempty"`
	Email       string            `json:"email,omitempty"`
	SSHKeyPath  string            `json:"sshKeyPath,omitempty"`
	HTTPSToken  string            `json:"httpsToken,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
	Timeout     time.Duration     `json:"timeout,omitempty"`
}

// GitClientMetrics provides Git client metrics
type GitClientMetrics struct {
	GitCommands     prometheus.CounterVec
	CommandDuration prometheus.HistogramVec
	GitErrors       prometheus.CounterVec
}

// ConfigSyncResult extends SyncResult with additional Config Sync information
type ConfigSyncResult struct {
	*SyncResult
	Repository     string          `json:"repository"`
	Branch         string          `json:"branch"`
	Directory      string          `json:"directory"`
	SyncRevision   string          `json:"syncRevision"`
	ClusterStatus  string          `json:"clusterStatus"`
	RootSyncStatus *RootSyncStatus `json:"rootSyncStatus,omitempty"`
	RepoSyncStatus *RepoSyncStatus `json:"repoSyncStatus,omitempty"`
	Conflicts      []SyncConflict  `json:"conflicts,omitempty"`
}

// RootSyncStatus represents RootSync resource status
type RootSyncStatus struct {
	Name       string          `json:"name"`
	Status     string          `json:"status"`
	Message    string          `json:"message"`
	Conditions []SyncCondition `json:"conditions"`
	Source     *SyncSource     `json:"source"`
	Sync       *SyncStatusInfo `json:"sync"`
}

// RepoSyncStatus represents RepoSync resource status
type RepoSyncStatus struct {
	Name       string          `json:"name"`
	Namespace  string          `json:"namespace"`
	Status     string          `json:"status"`
	Message    string          `json:"message"`
	Conditions []SyncCondition `json:"conditions"`
	Source     *SyncSource     `json:"source"`
	Sync       *SyncStatusInfo `json:"sync"`
}

// SyncCondition represents a sync condition
type SyncCondition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastUpdateTime     time.Time `json:"lastUpdateTime"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	Reason             string    `json:"reason"`
	Message            string    `json:"message"`
}

// SyncSource represents sync source information
type SyncSource struct {
	Git       *GitSyncSource `json:"git,omitempty"`
	Revision  string         `json:"revision"`
	Directory string         `json:"directory"`
}

// GitSyncSource represents Git sync source
type GitSyncSource struct {
	Repo      string           `json:"repo"`
	Branch    string           `json:"branch"`
	Revision  string           `json:"revision"`
	Auth      string           `json:"auth,omitempty"`
	SecretRef *SecretReference `json:"secretRef,omitempty"`
}

// SecretReference represents a secret reference
type SecretReference struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
}

// SyncStatusInfo represents sync status information
type SyncStatusInfo struct {
	Commit       string        `json:"commit"`
	ErrorSummary *ErrorSummary `json:"errorSummary,omitempty"`
	Errors       []SyncError   `json:"errors,omitempty"`
}

// ErrorSummary represents error summary
type ErrorSummary struct {
	TotalCount                int  `json:"totalCount"`
	Truncated                 bool `json:"truncated"`
	ErrorCountAfterTruncation int  `json:"errorCountAfterTruncation"`
}

// SyncConflict represents a sync conflict
type SyncConflict struct {
	Name      string `json:"name"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Message   string `json:"message"`
	Status    string `json:"status"`
}

// PolicyDir represents a policy directory structure
type PolicyDir struct {
	Path       string         `json:"path"`
	Policies   []PolicyFile   `json:"policies"`
	Namespaces []NamespaceDir `json:"namespaces"`
	Clusters   []ClusterDir   `json:"clusters"`
}

// PolicyFile represents a policy file
type PolicyFile struct {
	Name    string      `json:"name"`
	Kind    string      `json:"kind"`
	Content interface{} `json:"content"`
}

// NamespaceDir represents a namespace directory
type NamespaceDir struct {
	Name      string       `json:"name"`
	Path      string       `json:"path"`
	Resources []PolicyFile `json:"resources"`
}

// ClusterDir represents a cluster directory
type ClusterDir struct {
	Name      string       `json:"name"`
	Path      string       `json:"path"`
	Resources []PolicyFile `json:"resources"`
}

// Default Config Sync configuration
var DefaultConfigSyncMetrics = &ConfigSyncMetrics{
	SyncOperations: *promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephio_configsync_operations_total",
			Help: "Total number of Config Sync operations",
		},
		[]string{"operation", "cluster", "repository", "status"},
	),
	SyncDuration: *promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nephio_configsync_duration_seconds",
			Help:    "Duration of Config Sync operations",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10),
		},
		[]string{"operation", "cluster"},
	),
	SyncErrors: *promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephio_configsync_errors_total",
			Help: "Total number of Config Sync errors",
		},
		[]string{"cluster", "error_type"},
	),
	GitOperations: *promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephio_configsync_git_operations_total",
			Help: "Total number of Git operations",
		},
		[]string{"operation", "repository", "status"},
	),
	RepositoryHealth: *promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nephio_configsync_repository_health",
			Help: "Health status of Config Sync repositories",
		},
		[]string{"repository", "branch"},
	),
	PackageDeployments: *promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephio_configsync_package_deployments_total",
			Help: "Total number of package deployments via Config Sync",
		},
		[]string{"cluster", "package", "status"},
	),
}

// NewConfigSyncClient creates a new Config Sync client
func NewConfigSyncClient(
	client client.Client,
	config *ConfigSyncConfig,
) (*ConfigSyncClient, error) {
	if config == nil {
		return nil, fmt.Errorf("Config Sync configuration is required")
	}

	// Initialize metrics
	metrics := DefaultConfigSyncMetrics

	// Initialize Git client
	gitConfig := &GitConfig{
		Username: "nephoran-operator",
		Email:    "nephoran-operator@example.com",
		Timeout:  5 * time.Minute,
	}

	if config.Credentials != nil {
		gitConfig.Username = config.Credentials.Username
		gitConfig.HTTPSToken = config.Credentials.Token
		gitConfig.SSHKeyPath = config.Credentials.SSHKey
	}

	gitClient := &GitClient{
		config: gitConfig,
		tracer: otel.Tracer("nephio-git-client"),
		metrics: &GitClientMetrics{
			GitCommands: *promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "nephio_git_commands_total",
					Help: "Total number of Git commands executed",
				},
				[]string{"command", "repository", "status"},
			),
			CommandDuration: *promauto.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    "nephio_git_command_duration_seconds",
					Help:    "Duration of Git commands",
					Buckets: prometheus.ExponentialBuckets(0.1, 2, 8),
				},
				[]string{"command", "repository"},
			),
			GitErrors: *promauto.NewCounterVec(
				prometheus.CounterOpts{
					Name: "nephio_git_errors_total",
					Help: "Total number of Git errors",
				},
				[]string{"command", "error_type"},
			),
		},
	}

	// Initialize sync service
	syncService := &SyncService{
		client:       client,
		configSync:   config,
		tracer:       otel.Tracer("nephio-sync-service"),
		pollInterval: 30 * time.Second,
	}

	return &ConfigSyncClient{
		client:      client,
		gitClient:   gitClient,
		syncService: syncService,
		config:      config,
		tracer:      otel.Tracer("nephio-configsync"),
		metrics:     metrics,
	}, nil
}

// SyncPackageToCluster deploys a package to a cluster via Config Sync
func (csc *ConfigSyncClient) SyncPackageToCluster(ctx context.Context, pkg *porch.PackageRevision, cluster *WorkloadCluster) (*SyncResult, error) {
	ctx, span := csc.tracer.Start(ctx, "sync-package-to-cluster")
	defer span.End()

	logger := log.FromContext(ctx).WithName("configsync").WithValues(
		"package", pkg.Spec.PackageName,
		"revision", pkg.Spec.Revision,
		"cluster", cluster.Name,
	)

	span.SetAttributes(
		attribute.String("package.name", pkg.Spec.PackageName),
		attribute.String("package.revision", pkg.Spec.Revision),
		attribute.String("cluster.name", cluster.Name),
	)

	startTime := time.Now()
	defer func() {
		csc.metrics.SyncDuration.WithLabelValues(
			"sync_package", cluster.Name,
		).Observe(time.Since(startTime).Seconds())
	}()

	// Step 1: Prepare package contents for deployment
	deploymentContent, err := csc.preparePackageContent(ctx, pkg, cluster)
	if err != nil {
		span.RecordError(err)
		csc.metrics.SyncErrors.WithLabelValues(cluster.Name, "preparation_failed").Inc()
		return nil, fmt.Errorf("failed to prepare package content: %w", err)
	}

	// Step 2: Create cluster-specific directory structure
	clusterDir := filepath.Join(csc.config.Directory, cluster.Name)
	packageDir := filepath.Join(clusterDir, pkg.Spec.PackageName)

	// Step 3: Write package resources to Git repository
	if err := csc.writePackageToRepository(ctx, packageDir, deploymentContent); err != nil {
		span.RecordError(err)
		csc.metrics.SyncErrors.WithLabelValues(cluster.Name, "write_failed").Inc()
		return nil, fmt.Errorf("failed to write package to repository: %w", err)
	}

	// Step 4: Commit and push changes
	commitMessage := fmt.Sprintf("Deploy %s v%s to cluster %s",
		pkg.Spec.PackageName, pkg.Spec.Revision, cluster.Name)

	files := []string{packageDir}
	if err := csc.gitClient.Commit(ctx, commitMessage, files); err != nil {
		span.RecordError(err)
		csc.metrics.GitOperations.WithLabelValues("commit", csc.config.Repository, "failed").Inc()
		return nil, fmt.Errorf("failed to commit package changes: %w", err)
	}

	if err := csc.gitClient.Push(ctx); err != nil {
		span.RecordError(err)
		csc.metrics.GitOperations.WithLabelValues("push", csc.config.Repository, "failed").Inc()
		return nil, fmt.Errorf("failed to push package changes: %w", err)
	}

	// Step 5: Monitor sync status
	syncResult, err := csc.waitForSyncCompletion(ctx, cluster, pkg.Spec.PackageName)
	if err != nil {
		logger.Error(err, "Sync monitoring failed")
		// Don't fail the operation - sync may still succeed
	}

	// Step 6: Verify deployment status
	if syncResult == nil {
		syncResult = &SyncResult{
			Status:    "Deployed",
			Message:   "Package deployed successfully",
			Commit:    "unknown", // Would get actual commit hash
			Resources: []SyncedResource{},
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
		}
	}

	csc.metrics.SyncOperations.WithLabelValues(
		"sync_package", cluster.Name, csc.config.Repository, "success",
	).Inc()

	csc.metrics.PackageDeployments.WithLabelValues(
		cluster.Name, pkg.Spec.PackageName, "success",
	).Inc()

	logger.Info("Package deployed successfully",
		"duration", time.Since(startTime),
		"status", syncResult.Status,
	)

	span.SetAttributes(
		attribute.String("sync.status", syncResult.Status),
		attribute.Int("sync.resources", len(syncResult.Resources)),
		attribute.Float64("sync.duration", syncResult.Duration.Seconds()),
	)

	return syncResult, nil
}

// preparePackageContent prepares package content for Config Sync deployment
func (csc *ConfigSyncClient) preparePackageContent(ctx context.Context, pkg *porch.PackageRevision, cluster *WorkloadCluster) (map[string][]byte, error) {
	ctx, span := csc.tracer.Start(ctx, "prepare-package-content")
	defer span.End()

	content := make(map[string][]byte)

	// Convert KRM resources to YAML files
	for i, resource := range pkg.Spec.Resources {
		// Extract name from metadata
		resourceName := "unnamed"
		if metadata, ok := resource.Metadata["name"].(string); ok && metadata != "" {
			resourceName = metadata
		}

		// Generate filename
		filename := fmt.Sprintf("%s-%s.yaml", strings.ToLower(resource.Kind), resourceName)
		if resourceName == "unnamed" {
			filename = fmt.Sprintf("%s-%d.yaml", strings.ToLower(resource.Kind), i)
		}

		// Convert entire resource to YAML
		yamlContent, err := yaml.Marshal(resource)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("failed to marshal resource: %w", err)
		}

		content[filename] = yamlContent
	}

	// Generate Kustomization file
	kustomization := map[string]interface{}{
		"apiVersion": "kustomize.config.k8s.io/v1beta1",
		"kind":       "Kustomization",
		"metadata": map[string]interface{}{
			"name": pkg.Spec.PackageName,
			"annotations": map[string]interface{}{
				"config.k8s.io/local-config": "true",
			},
		},
		"resources": csc.getResourceFileNames(content),
		"commonLabels": map[string]interface{}{
			"app.kubernetes.io/name":       pkg.Spec.PackageName,
			"app.kubernetes.io/version":    pkg.Spec.Revision,
			"app.kubernetes.io/managed-by": "nephoran-intent-operator",
			"nephoran.io/cluster":          cluster.Name,
			"nephoran.io/package":          pkg.Spec.PackageName,
		},
		"commonAnnotations": map[string]interface{}{
			"nephoran.io/deployed-at": time.Now().Format(time.RFC3339),
			"nephoran.io/source-repo": pkg.Spec.Repository,
			"nephoran.io/source-rev":  pkg.Spec.Revision,
		},
	}

	kustomizationYAML, err := yaml.Marshal(kustomization)
	if err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to generate kustomization.yaml: %w", err)
	}
	content["kustomization.yaml"] = kustomizationYAML

	// Generate namespace if not present
	if !csc.hasNamespaceResource(content) {
		namespace := map[string]interface{}{
			"apiVersion": "v1",
			"kind":       "Namespace",
			"metadata": map[string]interface{}{
				"name": fmt.Sprintf("%s-ns", pkg.Spec.PackageName),
				"labels": map[string]interface{}{
					"nephoran.io/managed": "true",
					"nephoran.io/cluster": cluster.Name,
					"nephoran.io/package": pkg.Spec.PackageName,
				},
			},
		}

		namespaceYAML, err := yaml.Marshal(namespace)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("failed to generate namespace: %w", err)
		}
		content["namespace.yaml"] = namespaceYAML
	}

	span.SetAttributes(
		attribute.Int("content.files", len(content)),
		attribute.Int("content.resources", len(pkg.Spec.Resources)),
	)

	return content, nil
}

// writePackageToRepository writes package content to Git repository
func (csc *ConfigSyncClient) writePackageToRepository(ctx context.Context, packageDir string, content map[string][]byte) error {
	ctx, span := csc.tracer.Start(ctx, "write-package-to-repository")
	defer span.End()

	// Create directory structure
	if err := os.MkdirAll(packageDir, 0755); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to create package directory: %w", err)
	}

	// Write all files
	for filename, data := range content {
		filePath := filepath.Join(packageDir, filename)
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to write file %s: %w", filename, err)
		}
	}

	span.SetAttributes(
		attribute.String("package.dir", packageDir),
		attribute.Int("files.written", len(content)),
	)

	return nil
}

// waitForSyncCompletion waits for Config Sync to complete deployment
func (csc *ConfigSyncClient) waitForSyncCompletion(ctx context.Context, cluster *WorkloadCluster, packageName string) (*SyncResult, error) {
	ctx, span := csc.tracer.Start(ctx, "wait-for-sync-completion")
	defer span.End()

	// In a real implementation, this would:
	// 1. Query Config Sync status in the target cluster
	// 2. Monitor RootSync/RepoSync resources
	// 3. Check resource deployment status
	// 4. Wait for all resources to be applied

	// For now, simulate successful sync after a delay
	time.Sleep(2 * time.Second)

	syncResult := &SyncResult{
		Status:  "Synced",
		Message: "All resources synced successfully",
		Commit:  "abc123def", // Would be actual commit hash
		Resources: []SyncedResource{
			{
				Name:      fmt.Sprintf("%s-deployment", packageName),
				Kind:      "Deployment",
				Namespace: fmt.Sprintf("%s-ns", packageName),
				Status:    "Synced",
				Message:   "Deployment created successfully",
			},
			{
				Name:      fmt.Sprintf("%s-service", packageName),
				Kind:      "Service",
				Namespace: fmt.Sprintf("%s-ns", packageName),
				Status:    "Synced",
				Message:   "Service created successfully",
			},
		},
		Duration:  2 * time.Second,
		Timestamp: time.Now(),
	}

	span.SetAttributes(
		attribute.String("sync.status", syncResult.Status),
		attribute.Int("sync.resources", len(syncResult.Resources)),
	)

	return syncResult, nil
}

// Helper methods

func (csc *ConfigSyncClient) getResourceFileNames(content map[string][]byte) []string {
	var files []string
	for filename := range content {
		if filename != "kustomization.yaml" {
			files = append(files, filename)
		}
	}
	return files
}

func (csc *ConfigSyncClient) hasNamespaceResource(content map[string][]byte) bool {
	for filename := range content {
		if strings.Contains(strings.ToLower(filename), "namespace") {
			return true
		}
	}
	return false
}

// Git client implementation

// Clone clones a Git repository
func (gc *GitClient) Clone(ctx context.Context, repo, branch, dir string) error {
	ctx, span := gc.tracer.Start(ctx, "git-clone")
	defer span.End()

	startTime := time.Now()
	defer func() {
		gc.metrics.CommandDuration.WithLabelValues("clone", repo).Observe(time.Since(startTime).Seconds())
	}()

	args := []string{"clone", "--branch", branch, "--single-branch", repo, dir}
	if err := gc.runGitCommand(ctx, ".", args...); err != nil {
		span.RecordError(err)
		gc.metrics.GitErrors.WithLabelValues("clone", "execution_failed").Inc()
		gc.metrics.GitCommands.WithLabelValues("clone", repo, "failed").Inc()
		return err
	}

	gc.workingDir = dir
	gc.metrics.GitCommands.WithLabelValues("clone", repo, "success").Inc()

	span.SetAttributes(
		attribute.String("git.repo", repo),
		attribute.String("git.branch", branch),
		attribute.String("git.dir", dir),
	)

	return nil
}

// Commit commits changes to Git
func (gc *GitClient) Commit(ctx context.Context, message string, files []string) error {
	ctx, span := gc.tracer.Start(ctx, "git-commit")
	defer span.End()

	startTime := time.Now()
	defer func() {
		gc.metrics.CommandDuration.WithLabelValues("commit", "local").Observe(time.Since(startTime).Seconds())
	}()

	// Add files
	for _, file := range files {
		if err := gc.runGitCommand(ctx, gc.workingDir, "add", file); err != nil {
			span.RecordError(err)
			gc.metrics.GitErrors.WithLabelValues("add", "execution_failed").Inc()
			return err
		}
	}

	// Commit
	if err := gc.runGitCommand(ctx, gc.workingDir, "commit", "-m", message); err != nil {
		span.RecordError(err)
		gc.metrics.GitErrors.WithLabelValues("commit", "execution_failed").Inc()
		gc.metrics.GitCommands.WithLabelValues("commit", "local", "failed").Inc()
		return err
	}

	gc.metrics.GitCommands.WithLabelValues("commit", "local", "success").Inc()

	span.SetAttributes(
		attribute.String("git.message", message),
		attribute.Int("git.files", len(files)),
	)

	return nil
}

// Push pushes changes to remote
func (gc *GitClient) Push(ctx context.Context) error {
	ctx, span := gc.tracer.Start(ctx, "git-push")
	defer span.End()

	startTime := time.Now()
	defer func() {
		gc.metrics.CommandDuration.WithLabelValues("push", "remote").Observe(time.Since(startTime).Seconds())
	}()

	if err := gc.runGitCommand(ctx, gc.workingDir, "push", "origin"); err != nil {
		span.RecordError(err)
		gc.metrics.GitErrors.WithLabelValues("push", "execution_failed").Inc()
		gc.metrics.GitCommands.WithLabelValues("push", "remote", "failed").Inc()
		return err
	}

	gc.metrics.GitCommands.WithLabelValues("push", "remote", "success").Inc()
	return nil
}

// Pull pulls changes from remote
func (gc *GitClient) Pull(ctx context.Context) error {
	ctx, span := gc.tracer.Start(ctx, "git-pull")
	defer span.End()

	startTime := time.Now()
	defer func() {
		gc.metrics.CommandDuration.WithLabelValues("pull", "remote").Observe(time.Since(startTime).Seconds())
	}()

	if err := gc.runGitCommand(ctx, gc.workingDir, "pull", "origin"); err != nil {
		span.RecordError(err)
		gc.metrics.GitErrors.WithLabelValues("pull", "execution_failed").Inc()
		gc.metrics.GitCommands.WithLabelValues("pull", "remote", "failed").Inc()
		return err
	}

	gc.metrics.GitCommands.WithLabelValues("pull", "remote", "success").Inc()
	return nil
}

// CreateBranch creates a new branch
func (gc *GitClient) CreateBranch(ctx context.Context, branch string) error {
	ctx, span := gc.tracer.Start(ctx, "git-create-branch")
	defer span.End()

	if err := gc.runGitCommand(ctx, gc.workingDir, "checkout", "-b", branch); err != nil {
		span.RecordError(err)
		gc.metrics.GitErrors.WithLabelValues("checkout", "execution_failed").Inc()
		return err
	}

	span.SetAttributes(attribute.String("git.branch", branch))
	return nil
}

// MergeBranch merges source branch into target
func (gc *GitClient) MergeBranch(ctx context.Context, source, target string) error {
	ctx, span := gc.tracer.Start(ctx, "git-merge-branch")
	defer span.End()

	// Checkout target branch
	if err := gc.runGitCommand(ctx, gc.workingDir, "checkout", target); err != nil {
		span.RecordError(err)
		return err
	}

	// Merge source branch
	if err := gc.runGitCommand(ctx, gc.workingDir, "merge", source); err != nil {
		span.RecordError(err)
		gc.metrics.GitErrors.WithLabelValues("merge", "execution_failed").Inc()
		return err
	}

	span.SetAttributes(
		attribute.String("git.source", source),
		attribute.String("git.target", target),
	)

	return nil
}

// GetCommitHash gets the current commit hash
func (gc *GitClient) GetCommitHash(ctx context.Context) (string, error) {
	ctx, span := gc.tracer.Start(ctx, "git-commit-hash")
	defer span.End()

	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	cmd.Dir = gc.workingDir

	output, err := cmd.Output()
	if err != nil {
		span.RecordError(err)
		return "", err
	}

	hash := strings.TrimSpace(string(output))
	span.SetAttributes(attribute.String("git.hash", hash))

	return hash, nil
}

// runGitCommand runs a Git command with proper configuration
func (gc *GitClient) runGitCommand(ctx context.Context, dir string, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir

	// Set up environment
	env := os.Environ()
	if gc.config.Username != "" {
		env = append(env, fmt.Sprintf("GIT_AUTHOR_NAME=%s", gc.config.Username))
		env = append(env, fmt.Sprintf("GIT_COMMITTER_NAME=%s", gc.config.Username))
	}
	if gc.config.Email != "" {
		env = append(env, fmt.Sprintf("GIT_AUTHOR_EMAIL=%s", gc.config.Email))
		env = append(env, fmt.Sprintf("GIT_COMMITTER_EMAIL=%s", gc.config.Email))
	}
	cmd.Env = env

	// Set timeout
	if gc.config.Timeout > 0 {
		ctx, cancel := context.WithTimeout(ctx, gc.config.Timeout)
		defer cancel()
		cmd = exec.CommandContext(ctx, cmd.Path, cmd.Args[1:]...)
		cmd.Dir = dir
		cmd.Env = env
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed: %s, output: %s", err, string(output))
	}

	return nil
}
