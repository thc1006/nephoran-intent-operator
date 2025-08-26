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

	"github.com/go-logr/logr"
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

// ConfigSyncConfig defines Config Sync configuration (unified type)
type ConfigSyncConfig struct {
	Repository  string        `json:"repository" yaml:"repository"`
	Branch      string        `json:"branch" yaml:"branch"`
	Directory   string        `json:"directory" yaml:"directory"`
	SyncPeriod  time.Duration `json:"syncPeriod" yaml:"syncPeriod"`
	Username    string        `json:"username" yaml:"username"`
	Email       string        `json:"email" yaml:"email"`
	AuthToken   string        `json:"authToken" yaml:"authToken"`
	PolicyDir   string        `json:"policyDir" yaml:"policyDir"`
	Namespaces  []string      `json:"namespaces" yaml:"namespaces"`
}

// GitConfig contains Git configuration for Config Sync
type GitConfig struct {
	URL       string
	Branch    string
	AuthToken string
	Username  string
	Email     string
}

// ConfigSyncMetrics provides Config Sync metrics
type ConfigSyncMetrics struct {
	SyncOperations     *prometheus.CounterVec
	SyncDuration       *prometheus.HistogramVec
	SyncErrors         *prometheus.CounterVec
	GitOperations      *prometheus.CounterVec
	RepositoryHealth   *prometheus.GaugeVec
	PackageDeployments *prometheus.CounterVec
}

// NewConfigSyncMetrics creates new Config Sync metrics
func NewConfigSyncMetrics() *ConfigSyncMetrics {
	return &ConfigSyncMetrics{
		SyncOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "configsync_operations_total",
				Help: "Total number of Config Sync operations",
			},
			[]string{"operation", "repository", "status"},
		),
		SyncDuration: promauto.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "configsync_operation_duration_seconds",
				Help:    "Duration of Config Sync operations",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"operation", "repository"},
		),
		SyncErrors: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "configsync_errors_total",
				Help: "Total number of Config Sync errors",
			},
			[]string{"repository", "error_type"},
		),
		GitOperations: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "configsync_git_operations_total",
				Help: "Total number of Git operations",
			},
			[]string{"operation", "repository", "status"},
		),
		RepositoryHealth: promauto.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "configsync_repository_health",
				Help: "Health status of Git repositories (1=healthy, 0=unhealthy)",
			},
			[]string{"repository", "branch"},
		),
		PackageDeployments: promauto.NewCounterVec(
			prometheus.CounterOpts{
				Name: "configsync_package_deployments_total",
				Help: "Total number of package deployments",
			},
			[]string{"package", "cluster", "status"},
		),
	}
}

// ConfigSyncClient manages Config Sync operations
type ConfigSyncClient struct {
	client    client.Client
	logger    logr.Logger
	metrics   *ConfigSyncMetrics
	tracer    trace.Tracer
	repoPath  string
	gitConfig GitConfig
}

// SyncResult represents the result of a Config Sync operation
type SyncResult struct {
	Status    string
	Resources []string
	Duration  time.Duration
	Error     error
}

// NewConfigSyncClient creates a new Config Sync client
func NewConfigSyncClient(client client.Client, config *ConfigSyncConfig) (*ConfigSyncClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	gitConfig := GitConfig{
		URL:       config.Repository,
		Branch:    config.Branch,
		AuthToken: config.AuthToken,
		Username:  config.Username,
		Email:     config.Email,
	}

	return &ConfigSyncClient{
		client:    client,
		logger:    log.Log.WithName("configsync"),
		metrics:   NewConfigSyncMetrics(),
		tracer:    otel.Tracer("configsync"),
		repoPath:  "/tmp/configsync-repo",
		gitConfig: gitConfig,
	}, nil
}

// DeployPackage deploys a package using Config Sync
func (csc *ConfigSyncClient) DeployPackage(ctx context.Context, pkg *porch.PackageRevision, cluster *WorkloadCluster) (*SyncResult, error) {
	ctx, span := csc.tracer.Start(ctx, "deploy-package")
	defer span.End()

	startTime := time.Now()
	logger := csc.logger.WithValues(
		"package", pkg.Spec.PackageName,
		"revision", pkg.Spec.Revision,
		"cluster", cluster.Name,
	)

	logger.Info("Starting package deployment via Config Sync")

	// Prepare package content for Git repository
	content, err := csc.preparePackageContent(ctx, pkg, cluster)
	if err != nil {
		csc.metrics.SyncErrors.WithLabelValues(csc.gitConfig.URL, "prepare_content").Inc()
		span.RecordError(err)
		return nil, fmt.Errorf("failed to prepare package content: %w", err)
	}

	// Clone or update Git repository
	if err := csc.setupRepository(ctx); err != nil {
		csc.metrics.SyncErrors.WithLabelValues(csc.gitConfig.URL, "setup_repo").Inc()
		span.RecordError(err)
		return nil, fmt.Errorf("failed to setup repository: %w", err)
	}

	// Write package files to repository
	packagePath := filepath.Join(csc.repoPath, "clusters", cluster.Name, pkg.Spec.PackageName)
	if err := os.MkdirAll(packagePath, 0755); err != nil {
		span.RecordError(err)
		return nil, fmt.Errorf("failed to create package directory: %w", err)
	}

	var resources []string
	for filename, data := range content {
		filePath := filepath.Join(packagePath, filename)
		if err := os.WriteFile(filePath, data, 0644); err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("failed to write file %s: %w", filename, err)
		}
		resources = append(resources, filename)
	}

	// Commit and push changes
	if err := csc.commitAndPush(ctx, pkg, cluster); err != nil {
		csc.metrics.GitOperations.WithLabelValues("push", csc.gitConfig.URL, "failed").Inc()
		span.RecordError(err)
		return nil, fmt.Errorf("failed to commit and push changes: %w", err)
	}

	csc.metrics.GitOperations.WithLabelValues("push", csc.gitConfig.URL, "success").Inc()
	csc.metrics.PackageDeployments.WithLabelValues(pkg.Spec.PackageName, cluster.Name, "success").Inc()

	syncResult := &SyncResult{
		Status:    "Success",
		Resources: resources,
		Duration:  time.Since(startTime),
	}

	csc.metrics.SyncOperations.WithLabelValues("deploy", csc.gitConfig.URL, "success").Inc()
	csc.metrics.SyncDuration.WithLabelValues("deploy", csc.gitConfig.URL).Observe(syncResult.Duration.Seconds())

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

// KrmResource represents a KRM resource structure
type KrmResource struct {
	APIVersion string                 `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind       string                 `json:"kind,omitempty" yaml:"kind,omitempty"`
	Metadata   map[string]interface{} `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec       interface{}            `json:"spec,omitempty" yaml:"spec,omitempty"`
	Status     interface{}            `json:"status,omitempty" yaml:"status,omitempty"`
	Data       interface{}            `json:"data,omitempty" yaml:"data,omitempty"`
}

// preparePackageContent prepares package content for Config Sync deployment
func (csc *ConfigSyncClient) preparePackageContent(ctx context.Context, pkg *porch.PackageRevision, cluster *WorkloadCluster) (map[string][]byte, error) {
	ctx, span := csc.tracer.Start(ctx, "prepare-package-content")
	defer span.End()

	content := make(map[string][]byte)

	// Convert KRM resources to YAML files
	for i, resourceInterface := range pkg.Spec.Resources {
		// Try to handle resource as a map first
		resourceMap, ok := resourceInterface.(map[string]interface{})
		if !ok {
			// Try to marshal and unmarshal to get a map
			resourceData, err := yaml.Marshal(resourceInterface)
			if err != nil {
				span.RecordError(err)
				return nil, fmt.Errorf("failed to marshal resource %d: %w", i, err)
			}
			
			if err := yaml.Unmarshal(resourceData, &resourceMap); err != nil {
				span.RecordError(err)
				return nil, fmt.Errorf("failed to unmarshal resource %d: %w", i, err)
			}
		}

		// Extract kind and name from the resource map
		kind, _ := resourceMap["kind"].(string)
		if kind == "" {
			kind = "Resource"
		}

		var name string
		if metadata, ok := resourceMap["metadata"].(map[string]interface{}); ok {
			name, _ = metadata["name"].(string)
		}
		if name == "" {
			name = fmt.Sprintf("resource-%d", i)
		}

		// Generate filename
		filename := fmt.Sprintf("%s-%s.yaml", strings.ToLower(kind), name)

		// Convert resource content to YAML
		yamlContent, err := yaml.Marshal(resourceMap)
		if err != nil {
			span.RecordError(err)
			return nil, fmt.Errorf("failed to marshal resource %s: %w", name, err)
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

// getResourceFileNames extracts resource file names from content map
func (csc *ConfigSyncClient) getResourceFileNames(content map[string][]byte) []string {
	var files []string
	for filename := range content {
		if filename != "kustomization.yaml" {
			files = append(files, filename)
		}
	}
	return files
}

// hasNamespaceResource checks if the content contains a namespace resource
func (csc *ConfigSyncClient) hasNamespaceResource(content map[string][]byte) bool {
	for filename := range content {
		if strings.Contains(strings.ToLower(filename), "namespace") {
			return true
		}
	}
	return false
}

// setupRepository clones or updates the Git repository
func (csc *ConfigSyncClient) setupRepository(ctx context.Context) error {
	ctx, span := csc.tracer.Start(ctx, "setup-repository")
	defer span.End()

	// Check if repository already exists
	if _, err := os.Stat(csc.repoPath); os.IsNotExist(err) {
		// Clone repository
		cmd := exec.CommandContext(ctx, "git", "clone", csc.gitConfig.URL, csc.repoPath)
		if output, err := cmd.CombinedOutput(); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to clone repository: %w, output: %s", err, output)
		}
	} else {
		// Pull latest changes
		cmd := exec.CommandContext(ctx, "git", "-C", csc.repoPath, "pull", "origin", csc.gitConfig.Branch)
		if output, err := cmd.CombinedOutput(); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to pull repository: %w, output: %s", err, output)
		}
	}

	// Configure Git user if needed
	if csc.gitConfig.Username != "" && csc.gitConfig.Email != "" {
		userCmd := exec.CommandContext(ctx, "git", "-C", csc.repoPath, "config", "user.name", csc.gitConfig.Username)
		if err := userCmd.Run(); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to configure git user: %w", err)
		}

		emailCmd := exec.CommandContext(ctx, "git", "-C", csc.repoPath, "config", "user.email", csc.gitConfig.Email)
		if err := emailCmd.Run(); err != nil {
			span.RecordError(err)
			return fmt.Errorf("failed to configure git email: %w", err)
		}
	}

	return nil
}

// commitAndPush commits and pushes changes to the Git repository
func (csc *ConfigSyncClient) commitAndPush(ctx context.Context, pkg *porch.PackageRevision, cluster *WorkloadCluster) error {
	ctx, span := csc.tracer.Start(ctx, "commit-and-push")
	defer span.End()

	// Add all changes
	addCmd := exec.CommandContext(ctx, "git", "-C", csc.repoPath, "add", ".")
	if output, err := addCmd.CombinedOutput(); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to add changes: %w, output: %s", err, output)
	}

	// Check if there are changes to commit
	statusCmd := exec.CommandContext(ctx, "git", "-C", csc.repoPath, "status", "--porcelain")
	output, err := statusCmd.CombinedOutput()
	if err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to check git status: %w", err)
	}

	if len(output) == 0 {
		csc.logger.Info("No changes to commit")
		return nil
	}

	// Commit changes
	commitMsg := fmt.Sprintf("Deploy package %s revision %s to cluster %s", pkg.Spec.PackageName, pkg.Spec.Revision, cluster.Name)
	commitCmd := exec.CommandContext(ctx, "git", "-C", csc.repoPath, "commit", "-m", commitMsg)
	if output, err := commitCmd.CombinedOutput(); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to commit changes: %w, output: %s", err, output)
	}

	// Push changes
	pushCmd := exec.CommandContext(ctx, "git", "-C", csc.repoPath, "push", "origin", csc.gitConfig.Branch)
	if output, err := pushCmd.CombinedOutput(); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to push changes: %w, output: %s", err, output)
	}

	span.SetAttributes(
		attribute.String("commit.message", commitMsg),
		attribute.String("git.branch", csc.gitConfig.Branch),
	)

	return nil
}

// ValidateRepository validates the Git repository configuration
func (csc *ConfigSyncClient) ValidateRepository(ctx context.Context) error {
	ctx, span := csc.tracer.Start(ctx, "validate-repository")
	defer span.End()

	// Test repository connectivity
	cmd := exec.CommandContext(ctx, "git", "ls-remote", csc.gitConfig.URL)
	if output, err := cmd.CombinedOutput(); err != nil {
		csc.metrics.RepositoryHealth.WithLabelValues(csc.gitConfig.URL, csc.gitConfig.Branch).Set(0)
		span.RecordError(err)
		return fmt.Errorf("repository validation failed: %w, output: %s", err, output)
	}

	csc.metrics.RepositoryHealth.WithLabelValues(csc.gitConfig.URL, csc.gitConfig.Branch).Set(1)
	return nil
}

// GetSyncStatus returns the current sync status for a package
func (csc *ConfigSyncClient) GetSyncStatus(ctx context.Context, packageName, clusterName string) (*SyncResult, error) {
	ctx, span := csc.tracer.Start(ctx, "get-sync-status")
	defer span.End()

	// This is a simplified implementation
	// In a real scenario, you would query the Config Sync API or Git repository
	return &SyncResult{
		Status:    "Unknown",
		Resources: []string{},
		Duration:  0,
	}, nil
}

// CleanupPackage removes a package from the Config Sync repository
func (csc *ConfigSyncClient) CleanupPackage(ctx context.Context, packageName, clusterName string) error {
	ctx, span := csc.tracer.Start(ctx, "cleanup-package")
	defer span.End()

	logger := csc.logger.WithValues("package", packageName, "cluster", clusterName)
	logger.Info("Cleaning up package from Config Sync repository")

	// Setup repository
	if err := csc.setupRepository(ctx); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to setup repository for cleanup: %w", err)
	}

	// Remove package directory
	packagePath := filepath.Join(csc.repoPath, "clusters", clusterName, packageName)
	if err := os.RemoveAll(packagePath); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to remove package directory: %w", err)
	}

	// Commit and push removal
	addCmd := exec.CommandContext(ctx, "git", "-C", csc.repoPath, "add", ".")
	if output, err := addCmd.CombinedOutput(); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to add removal changes: %w, output: %s", err, output)
	}

	commitMsg := fmt.Sprintf("Remove package %s from cluster %s", packageName, clusterName)
	commitCmd := exec.CommandContext(ctx, "git", "-C", csc.repoPath, "commit", "-m", commitMsg)
	if output, err := commitCmd.CombinedOutput(); err != nil {
		// Check if there were no changes to commit
		if strings.Contains(string(output), "nothing to commit") {
			logger.Info("Package directory was already removed")
			return nil
		}
		span.RecordError(err)
		return fmt.Errorf("failed to commit removal: %w, output: %s", err, output)
	}

	pushCmd := exec.CommandContext(ctx, "git", "-C", csc.repoPath, "push", "origin", csc.gitConfig.Branch)
	if output, err := pushCmd.CombinedOutput(); err != nil {
		span.RecordError(err)
		return fmt.Errorf("failed to push removal: %w, output: %s", err, output)
	}

	logger.Info("Package cleanup completed successfully")
	return nil
}