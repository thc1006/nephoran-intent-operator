package git

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// ClientInterface defines the interface for a Git client.
type ClientInterface interface {
	CommitAndPush(files map[string]string, message string) (string, error)
	CommitAndPushChanges(message string) error
	InitRepo() error
	RemoveDirectory(path string, commitMessage string) error
}

// ClientConfig holds configuration for creating a Git client.
type ClientConfig struct {
	RepoURL   string
	Branch    string
	Token     string   // Token loaded from file or environment
	TokenPath string   // Optional path to token file
	RepoPath  string
	Logger    *slog.Logger
}

// Client implements the Git client.
type Client struct {
	RepoURL  string
	Branch   string
	SshKey   string
	RepoPath string
	logger   *slog.Logger
}

// NewGitClientConfig creates a new client configuration with token loading support.
// If tokenPath is provided, it reads the token from the file.
// If file reading fails or tokenPath is empty, it falls back to the provided token.
func NewGitClientConfig(repoURL, branch, token, tokenPath string) (*ClientConfig, error) {
	config := &ClientConfig{
		RepoURL:  repoURL,
		Branch:   branch,
		RepoPath: "/tmp/deployment-repo",
		Logger:   slog.Default().With("component", "git-client"),
	}

	// Try to read token from file first
	if tokenPath != "" {
		tokenData, err := os.ReadFile(tokenPath)
		if err == nil {
			config.Token = strings.TrimSpace(string(tokenData))
			config.TokenPath = tokenPath
			return config, nil
		}
		// Log warning but continue with fallback
		config.Logger.Warn("Failed to read token from file, falling back to environment variable",
			"path", tokenPath,
			"error", err)
	}

	// Fallback to provided token (from environment variable)
	if token != "" {
		config.Token = token
		return config, nil
	}

	return nil, fmt.Errorf("no git token available: neither file at %s nor environment variable", tokenPath)
}

// NewClientFromConfig creates a new Git client from configuration.
func NewClientFromConfig(config *ClientConfig) *Client {
	if config.Logger == nil {
		config.Logger = slog.Default().With("component", "git-client")
	}

	return &Client{
		RepoURL:  config.RepoURL,
		Branch:   config.Branch,
		SshKey:   config.Token,
		RepoPath: config.RepoPath,
		logger:   config.Logger,
	}
}

// NewClient creates a new Git client.
func NewClient(repoURL, branch, sshKey string) *Client {
	// Create a default logger if none provided
	logger := slog.Default().With("component", "git-client")
	
	return &Client{
		RepoURL:  repoURL,
		Branch:   branch,
		SshKey:   sshKey,
		RepoPath: "/tmp/deployment-repo",
		logger:   logger,
	}
}

// NewClientWithLogger creates a new Git client with a specific logger.
func NewClientWithLogger(repoURL, branch, sshKey string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("component", "git-client")
	
	return &Client{
		RepoURL:  repoURL,
		Branch:   branch,
		SshKey:   sshKey,
		RepoPath: "/tmp/deployment-repo",
		logger:   logger,
	}
}

// InitRepo clones the repository if it doesn't exist locally.
func (c *Client) InitRepo() error {
	if _, err := os.Stat(c.RepoPath); os.IsNotExist(err) {
		_, err := git.PlainClone(c.RepoPath, false, &git.CloneOptions{
			URL:      c.RepoURL,
			Progress: os.Stdout,
		})
		if err != nil && err != git.ErrRepositoryAlreadyExists {
			return fmt.Errorf("failed to clone repo: %w", err)
		}
	}
	return nil
}

// CommitAndPush writes files, commits them, and pushes to the remote repository.
// Returns the commit hash of the created commit.
func (c *Client) CommitAndPush(files map[string]string, message string) (string, error) {
	r, err := git.PlainOpen(c.RepoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repo: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	for path, content := range files {
		fullPath := filepath.Join(c.RepoPath, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			c.logger.Error("Failed to create directory", 
				"directory", filepath.Dir(fullPath),
				"file_path", path,
				"error", err,
				"operation", "CommitAndPush")
			return "", fmt.Errorf("failed to create directory for %s: %w", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			c.logger.Error("Failed to write file", 
				"filename", fullPath,
				"relative_path", path,
				"error", err,
				"operation", "CommitAndPush")
			return "", fmt.Errorf("failed to write file %s: %w", path, err)
		}
		c.logger.Debug("Successfully wrote file", 
			"filename", fullPath,
			"relative_path", path,
			"size_bytes", len(content),
			"operation", "CommitAndPush")
		if _, err := w.Add(path); err != nil {
			return "", fmt.Errorf("failed to add file %s: %w", path, err)
		}
	}

	commit, err := w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Nephio Bridge",
			Email: "nephio-bridge@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit: %w", err)
	}

	commitObj, err := r.CommitObject(commit)
	if err != nil {
		return "", fmt.Errorf("failed to get commit object: %w", err)
	}

	auth, err := ssh.NewPublicKeys("git", []byte(c.SshKey), "")
	if err != nil {
		return "", fmt.Errorf("failed to create ssh auth: %w", err)
	}

	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil {
		return "", fmt.Errorf("failed to push: %w", err)
	}

	return commitObj.Hash.String(), nil
}

// CommitAndPushChanges commits and pushes any changes without specifying files
func (c *Client) CommitAndPushChanges(message string) error {
	r, err := git.PlainOpen(c.RepoPath)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get status and stage only tracked files, skip untracked and .git files
	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("failed to get worktree status: %w", err)
	}

	for file := range status {
		// Skip files/directories starting with .git
		if strings.HasPrefix(file, ".git") {
			continue
		}

		// Stage only tracked files (modified, deleted, renamed)
		fileStatus := status[file]
		if fileStatus.Staging != git.Untracked && fileStatus.Worktree != git.Untracked {
			if _, err := w.Add(file); err != nil {
				return fmt.Errorf("failed to add file %s: %w", file, err)
			}
		}
	}

	_, err = w.Commit(message, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Nephio Bridge",
			Email: "nephio-bridge@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	auth, err := ssh.NewPublicKeys("git", []byte(c.SshKey), "")
	if err != nil {
		return fmt.Errorf("failed to create ssh auth: %w", err)
	}

	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil {
		return fmt.Errorf("failed to push: %w", err)
	}

	return nil
}

// RemoveDirectory removes a directory from the repository and commits the change
func (c *Client) RemoveDirectory(path string, commitMessage string) error {
	r, err := git.PlainOpen(c.RepoPath)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Check if directory exists before attempting removal
	fullPath := filepath.Join(c.RepoPath, path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		// Directory doesn't exist, nothing to remove - this is not an error
		return nil
	}

	// Remove directory from filesystem
	if err := os.RemoveAll(fullPath); err != nil {
		return fmt.Errorf("failed to remove directory %s: %w", fullPath, err)
	}

	// Get status to find all files that were deleted
	status, err := w.Status()
	if err != nil {
		return fmt.Errorf("failed to get worktree status: %w", err)
	}

	// Stage all deletions within the removed directory
	filesStaged := 0
	for file := range status {
		fileStatus := status[file]
		// Stage files that are deleted and within the target path
		if fileStatus.Worktree == git.Deleted && strings.HasPrefix(file, path) {
			if _, err := w.Add(file); err != nil {
				return fmt.Errorf("failed to stage deletion of %s: %w", file, err)
			}
			filesStaged++
		}
	}

	// If no files were staged for deletion, the directory was already empty or didn't contain tracked files
	if filesStaged == 0 {
		// No changes to commit, which is fine
		return nil
	}

	// Commit the changes
	_, err = w.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Nephio Bridge",
			Email: "nephio-bridge@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit directory removal: %w", err)
	}

	// Push the changes
	auth, err := ssh.NewPublicKeys("git", []byte(c.SshKey), "")
	if err != nil {
		return fmt.Errorf("failed to create ssh auth: %w", err)
	}

	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil {
		return fmt.Errorf("failed to push directory removal: %w", err)
	}

	return nil
}
