package clients

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/config"
	"github.com/thc1006/nephoran-intent-operator/pkg/git"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// MTLSGitClient provides mTLS-enabled Git client functionality
type MTLSGitClient struct {
	httpClient *http.Client
	logger     *logging.StructuredLogger
	config     *config.Config

	// Cached git.ClientInterface implementation
	gitClient git.ClientInterface
}

// CommitFiles commits files to the Git repository using mTLS
func (c *MTLSGitClient) CommitFiles(ctx context.Context, files map[string]string, message string) (string, error) {
	if len(files) == 0 {
		return "", fmt.Errorf("no files provided for commit")
	}

	if message == "" {
		message = "Automated commit via Nephoran Intent Operator"
	}

	c.logger.Debug("committing files via mTLS Git client",
		"file_count", len(files),
		"message", message)

	// Ensure we have a Git client instance
	if err := c.ensureGitClient(); err != nil {
		return "", fmt.Errorf("failed to initialize Git client: %w", err)
	}

	// Use the underlying Git client with mTLS transport
	commitHash, err := c.gitClient.CommitFiles(ctx, files, message)
	if err != nil {
		return "", fmt.Errorf("failed to commit files: %w", err)
	}

	c.logger.Info("files committed successfully",
		"commit_hash", commitHash,
		"file_count", len(files))

	return commitHash, nil
}

// CreateBranch creates a new branch in the Git repository
func (c *MTLSGitClient) CreateBranch(ctx context.Context, branchName, baseBranch string) error {
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	c.logger.Debug("creating branch via mTLS Git client",
		"branch_name", branchName,
		"base_branch", baseBranch)

	// Ensure we have a Git client instance
	if err := c.ensureGitClient(); err != nil {
		return fmt.Errorf("failed to initialize Git client: %w", err)
	}

	if err := c.gitClient.CreateBranch(ctx, branchName, baseBranch); err != nil {
		return fmt.Errorf("failed to create branch: %w", err)
	}

	c.logger.Info("branch created successfully", "branch_name", branchName)

	return nil
}

// SwitchBranch switches to a different branch
func (c *MTLSGitClient) SwitchBranch(ctx context.Context, branchName string) error {
	if branchName == "" {
		return fmt.Errorf("branch name cannot be empty")
	}

	c.logger.Debug("switching branch via mTLS Git client",
		"branch_name", branchName)

	// Ensure we have a Git client instance
	if err := c.ensureGitClient(); err != nil {
		return fmt.Errorf("failed to initialize Git client: %w", err)
	}

	if err := c.gitClient.SwitchBranch(ctx, branchName); err != nil {
		return fmt.Errorf("failed to switch branch: %w", err)
	}

	c.logger.Info("switched to branch successfully", "branch_name", branchName)

	return nil
}

// GetCurrentBranch returns the current branch name
func (c *MTLSGitClient) GetCurrentBranch(ctx context.Context) (string, error) {
	// Ensure we have a Git client instance
	if err := c.ensureGitClient(); err != nil {
		return "", fmt.Errorf("failed to initialize Git client: %w", err)
	}

	branch, err := c.gitClient.GetCurrentBranch(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}

	return branch, nil
}

// ListBranches returns a list of branches
func (c *MTLSGitClient) ListBranches(ctx context.Context) ([]string, error) {
	// Ensure we have a Git client instance
	if err := c.ensureGitClient(); err != nil {
		return nil, fmt.Errorf("failed to initialize Git client: %w", err)
	}

	branches, err := c.gitClient.ListBranches(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list branches: %w", err)
	}

	return branches, nil
}

// GetFileContent retrieves the content of a file from the repository
func (c *MTLSGitClient) GetFileContent(ctx context.Context, filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	c.logger.Debug("retrieving file content via mTLS Git client",
		"file_path", filePath)

	// Ensure we have a Git client instance
	if err := c.ensureGitClient(); err != nil {
		return "", fmt.Errorf("failed to initialize Git client: %w", err)
	}

	content, err := c.gitClient.GetFileContent(ctx, filePath)
	if err != nil {
		return "", fmt.Errorf("failed to get file content: %w", err)
	}

	return content, nil
}

// DeleteFile deletes a file from the repository
func (c *MTLSGitClient) DeleteFile(ctx context.Context, filePath, message string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path cannot be empty")
	}

	if message == "" {
		message = fmt.Sprintf("Delete %s via Nephoran Intent Operator", filePath)
	}

	c.logger.Debug("deleting file via mTLS Git client",
		"file_path", filePath,
		"message", message)

	// Ensure we have a Git client instance
	if err := c.ensureGitClient(); err != nil {
		return "", fmt.Errorf("failed to initialize Git client: %w", err)
	}

	commitHash, err := c.gitClient.DeleteFile(ctx, filePath, message)
	if err != nil {
		return "", fmt.Errorf("failed to delete file: %w", err)
	}

	c.logger.Info("file deleted successfully",
		"file_path", filePath,
		"commit_hash", commitHash)

	return commitHash, nil
}

// GetCommitHistory returns commit history for the repository
func (c *MTLSGitClient) GetCommitHistory(ctx context.Context, limit int) ([]git.CommitInfo, error) {
	// Ensure we have a Git client instance
	if err := c.ensureGitClient(); err != nil {
		return nil, fmt.Errorf("failed to initialize Git client: %w", err)
	}

	commits, err := c.gitClient.GetCommitHistory(ctx, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get commit history: %w", err)
	}

	return commits, nil
}

// Push pushes changes to the remote repository
func (c *MTLSGitClient) Push(ctx context.Context, branch string) error {
	c.logger.Debug("pushing changes via mTLS Git client",
		"branch", branch)

	// Ensure we have a Git client instance
	if err := c.ensureGitClient(); err != nil {
		return fmt.Errorf("failed to initialize Git client: %w", err)
	}

	if err := c.gitClient.Push(ctx, branch); err != nil {
		return fmt.Errorf("failed to push changes: %w", err)
	}

	c.logger.Info("changes pushed successfully", "branch", branch)

	return nil
}

// Pull pulls changes from the remote repository
func (c *MTLSGitClient) Pull(ctx context.Context, branch string) error {
	c.logger.Debug("pulling changes via mTLS Git client",
		"branch", branch)

	// Ensure we have a Git client instance
	if err := c.ensureGitClient(); err != nil {
		return fmt.Errorf("failed to initialize Git client: %w", err)
	}

	if err := c.gitClient.Pull(ctx, branch); err != nil {
		return fmt.Errorf("failed to pull changes: %w", err)
	}

	c.logger.Info("changes pulled successfully", "branch", branch)

	return nil
}

// GetStatus returns the status of the working directory
func (c *MTLSGitClient) GetStatus(ctx context.Context) (*git.StatusInfo, error) {
	// Ensure we have a Git client instance
	if err := c.ensureGitClient(); err != nil {
		return nil, fmt.Errorf("failed to initialize Git client: %w", err)
	}

	status, err := c.gitClient.GetStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get status: %w", err)
	}

	return status, nil
}

// CreatePullRequest creates a pull request (if supported by the Git provider)
func (c *MTLSGitClient) CreatePullRequest(ctx context.Context, options *git.PullRequestOptions) (*git.PullRequestInfo, error) {
	if options == nil {
		return nil, fmt.Errorf("pull request options cannot be nil")
	}

	c.logger.Debug("creating pull request via mTLS Git client",
		"title", options.Title,
		"source_branch", options.SourceBranch,
		"target_branch", options.TargetBranch)

	// Ensure we have a Git client instance
	if err := c.ensureGitClient(); err != nil {
		return nil, fmt.Errorf("failed to initialize Git client: %w", err)
	}

	pr, err := c.gitClient.CreatePullRequest(ctx, options)
	if err != nil {
		return nil, fmt.Errorf("failed to create pull request: %w", err)
	}

	c.logger.Info("pull request created successfully",
		"title", options.Title,
		"pr_number", pr.Number,
		"pr_url", pr.URL)

	return pr, nil
}

// ensureGitClient creates a Git client instance if one doesn't exist
func (c *MTLSGitClient) ensureGitClient() error {
	if c.gitClient != nil {
		return nil
	}

	// Create Git client configuration with mTLS HTTP client
	gitConfig := &git.Config{
		RepoURL:    c.config.GitRepoURL,
		Token:      c.config.GitToken,
		Branch:     c.config.GitBranch,
		HTTPClient: c.httpClient, // Use mTLS-enabled HTTP client
		Timeout:    30 * time.Second,
	}

	// Create Git client
	var err error
	c.gitClient, err = git.NewClientWithConfig(gitConfig, c.logger)
	if err != nil {
		return fmt.Errorf("failed to create Git client: %w", err)
	}

	c.logger.Info("Git client initialized with mTLS support",
		"repo_url", gitConfig.RepoURL,
		"branch", gitConfig.Branch,
		"mtls_enabled", true)

	return nil
}

// Close closes the Git client and cleans up resources
func (c *MTLSGitClient) Close() error {
	c.logger.Debug("closing mTLS Git client")

	// Close the underlying Git client if it has a Close method
	if c.gitClient != nil {
		if closer, ok := c.gitClient.(interface{ Close() error }); ok {
			if err := closer.Close(); err != nil {
				c.logger.Warn("failed to close Git client", "error", err)
			}
		}
		c.gitClient = nil
	}

	// Close idle connections in HTTP client
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	return nil
}

// GetEndpoint returns the endpoint URL
func (c *MTLSGitClient) GetEndpoint() string {
	if c.config != nil {
		return c.config.GitRepoURL
	}
	return "git-repository"
}

// GetHealth returns the health status of the Git service
func (c *MTLSGitClient) GetHealth() (*HealthStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Try to get current branch as a health check
	if err := c.ensureGitClient(); err != nil {
		return &HealthStatus{
			Status:    "unhealthy",
			Message:   fmt.Sprintf("failed to initialize Git client: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	_, err := c.gitClient.GetCurrentBranch(ctx)
	if err != nil {
		return &HealthStatus{
			Status:    "unhealthy",
			Message:   fmt.Sprintf("failed to access Git repository: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	return &HealthStatus{
		Status:    "healthy",
		Message:   "Git client operational",
		Timestamp: time.Now(),
		Details: map[string]interface{}{
			"repo_url":     c.config.GitRepoURL,
			"branch":       c.config.GitBranch,
			"mtls_enabled": true,
		},
	}, nil
}
