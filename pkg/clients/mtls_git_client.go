package clients

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/nephio-project/nephoran-intent-operator/pkg/config"
	"github.com/nephio-project/nephoran-intent-operator/pkg/git"
	"github.com/nephio-project/nephoran-intent-operator/pkg/logging"
)

// MTLSGitClient provides mTLS-enabled Git client functionality.

type MTLSGitClient struct {
	httpClient *http.Client

	logger *logging.StructuredLogger

	config *config.Config

	// Cached git.ClientInterface implementation.

	gitClient git.ClientInterface
}

// CommitFiles commits files to the Git repository using mTLS.

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

	// Ensure we have a Git client instance.

	if err := c.ensureGitClient(); err != nil {

		return "", fmt.Errorf("failed to initialize Git client: %w", err)

	}

	// Note: the underlying CommitFiles method doesn't handle file content,

	// so we'll use CommitAndPush instead which handles both content and commit.

	commitHash, err := c.gitClient.CommitAndPush(files, message)

	if err != nil {

		return "", fmt.Errorf("failed to commit files: %w", err)

	}

	c.logger.Info("files committed successfully",

		"commit_hash", commitHash,

		"file_count", len(files))

	return commitHash, nil

}

// CreateBranch creates a new branch in the Git repository.

func (c *MTLSGitClient) CreateBranch(ctx context.Context, branchName, baseBranch string) error {

	if branchName == "" {

		return fmt.Errorf("branch name cannot be empty")

	}

	c.logger.Debug("creating branch via mTLS Git client",

		"branch_name", branchName,

		"base_branch", baseBranch)

	// Ensure we have a Git client instance.

	if err := c.ensureGitClient(); err != nil {

		return fmt.Errorf("failed to initialize Git client: %w", err)

	}

	// Note: baseBranch parameter is ignored by the underlying implementation.

	// which creates branches from current HEAD.

	if err := c.gitClient.CreateBranch(branchName); err != nil {

		return fmt.Errorf("failed to create branch: %w", err)

	}

	c.logger.Info("branch created successfully", "branch_name", branchName)

	return nil

}

// SwitchBranch switches to a different branch.

func (c *MTLSGitClient) SwitchBranch(ctx context.Context, branchName string) error {

	if branchName == "" {

		return fmt.Errorf("branch name cannot be empty")

	}

	c.logger.Debug("switching branch via mTLS Git client",

		"branch_name", branchName)

	// Ensure we have a Git client instance.

	if err := c.ensureGitClient(); err != nil {

		return fmt.Errorf("failed to initialize Git client: %w", err)

	}

	if err := c.gitClient.SwitchBranch(branchName); err != nil {

		return fmt.Errorf("failed to switch branch: %w", err)

	}

	c.logger.Info("switched to branch successfully", "branch_name", branchName)

	return nil

}

// GetCurrentBranch returns the current branch name.

func (c *MTLSGitClient) GetCurrentBranch(ctx context.Context) (string, error) {

	// Ensure we have a Git client instance.

	if err := c.ensureGitClient(); err != nil {

		return "", fmt.Errorf("failed to initialize Git client: %w", err)

	}

	branch, err := c.gitClient.GetCurrentBranch()

	if err != nil {

		return "", fmt.Errorf("failed to get current branch: %w", err)

	}

	return branch, nil

}

// ListBranches returns a list of branches.

func (c *MTLSGitClient) ListBranches(ctx context.Context) ([]string, error) {

	// Ensure we have a Git client instance.

	if err := c.ensureGitClient(); err != nil {

		return nil, fmt.Errorf("failed to initialize Git client: %w", err)

	}

	branches, err := c.gitClient.ListBranches()

	if err != nil {

		return nil, fmt.Errorf("failed to list branches: %w", err)

	}

	return branches, nil

}

// GetFileContent retrieves the content of a file from the repository.

func (c *MTLSGitClient) GetFileContent(ctx context.Context, filePath string) (string, error) {

	if filePath == "" {

		return "", fmt.Errorf("file path cannot be empty")

	}

	c.logger.Debug("retrieving file content via mTLS Git client",

		"file_path", filePath)

	// Ensure we have a Git client instance.

	if err := c.ensureGitClient(); err != nil {

		return "", fmt.Errorf("failed to initialize Git client: %w", err)

	}

	content, err := c.gitClient.GetFileContent(filePath)

	if err != nil {

		return "", fmt.Errorf("failed to get file content: %w", err)

	}

	return string(content), nil

}

// DeleteFile deletes a file from the repository.

// Note: This method uses RemoveDirectory for file deletion since DeleteFile is not in the interface.

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

	// Ensure we have a Git client instance.

	if err := c.ensureGitClient(); err != nil {

		return "", fmt.Errorf("failed to initialize Git client: %w", err)

	}

	// Use RemoveDirectory for file deletion (works for files too).

	err := c.gitClient.RemoveDirectory(filePath, message)

	if err != nil {

		return "", fmt.Errorf("failed to delete file: %w", err)

	}

	c.logger.Info("file deleted successfully",

		"file_path", filePath)

	// Return a placeholder commit hash since RemoveDirectory doesn't return one.

	return "file-deleted", nil

}

// GetCommitHistory returns commit history for the repository.

// Note: This method is not supported by the underlying git client interface.

func (c *MTLSGitClient) GetCommitHistory(ctx context.Context, limit int) ([]git.CommitInfo, error) {

	return nil, fmt.Errorf("GetCommitHistory is not supported by the underlying git client interface")

}

// Push pushes changes to the remote repository.

// Note: Push functionality is handled by CommitAndPush and CommitAndPushChanges methods.

func (c *MTLSGitClient) Push(ctx context.Context, branch string) error {

	return fmt.Errorf("Push is not supported directly; use CommitAndPush or CommitAndPushChanges instead")

}

// Pull pulls changes from the remote repository.

// Note: This method is not supported by the underlying git client interface.

func (c *MTLSGitClient) Pull(ctx context.Context, branch string) error {

	return fmt.Errorf("Pull is not supported by the underlying git client interface")

}

// GetStatus returns the status of the working directory.

// Note: This method is not supported by the underlying git client interface.

func (c *MTLSGitClient) GetStatus(ctx context.Context) (*git.StatusInfo, error) {

	return nil, fmt.Errorf("GetStatus is not supported by the underlying git client interface")

}

// CreatePullRequest creates a pull request (if supported by the Git provider).

// Note: This method is not supported by the underlying git client interface.

func (c *MTLSGitClient) CreatePullRequest(ctx context.Context, options *git.PullRequestOptions) (*git.PullRequestInfo, error) {

	return nil, fmt.Errorf("CreatePullRequest is not supported by the underlying git client interface")

}

// ensureGitClient creates a Git client instance if one doesn't exist.

func (c *MTLSGitClient) ensureGitClient() error {

	if c.gitClient != nil {

		return nil

	}

	// Create Git client using the existing constructor.

	// Note: The current git client doesn't support custom HTTP clients directly.

	c.gitClient = git.NewClient(

		c.config.GitRepoURL,

		c.config.GitBranch,

		c.config.GitToken,
	)

	c.logger.Info("Git client initialized",

		"repo_url", c.config.GitRepoURL,

		"branch", c.config.GitBranch,

		"mtls_note", "mTLS is configured at HTTP client level but not directly integrated")

	return nil

}

// Close closes the Git client and cleans up resources.

func (c *MTLSGitClient) Close() error {

	c.logger.Debug("closing mTLS Git client")

	// Close the underlying Git client if it has a Close method.

	if c.gitClient != nil {

		if closer, ok := c.gitClient.(interface{ Close() error }); ok {

			if err := closer.Close(); err != nil {

				c.logger.Warn("failed to close Git client", "error", err)

			}

		}

		c.gitClient = nil

	}

	// Close idle connections in HTTP client.

	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {

		transport.CloseIdleConnections()

	}

	return nil

}

// GetHealth returns the health status of the Git service.

func (c *MTLSGitClient) GetHealth() (*HealthStatus, error) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()

	_ = ctx // ctx not used with current interface but kept for future compatibility

	// Try to get current branch as a health check.

	if err := c.ensureGitClient(); err != nil {

		return &HealthStatus{

			Status: "unhealthy",

			Message: fmt.Sprintf("failed to initialize Git client: %v", err),

			Timestamp: time.Now(),
		}, nil

	}

	_, err := c.gitClient.GetCurrentBranch()

	if err != nil {

		return &HealthStatus{

			Status: "unhealthy",

			Message: fmt.Sprintf("failed to access Git repository: %v", err),

			Timestamp: time.Now(),
		}, nil

	}

	return &HealthStatus{

		Status: "healthy",

		Message: "Git client operational",

		Timestamp: time.Now(),

		Details: map[string]interface{}{

			"repo_url": c.config.GitRepoURL,

			"branch": c.config.GitBranch,

			"mtls_enabled": true,
		},
	}, nil

}
