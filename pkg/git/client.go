package git

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

var (
	// Metrics registration guard
	metricsOnce sync.Once

	// Git push in-flight gauge metric
	gitPushInFlightGauge prometheus.Gauge
)

// InitMetrics initializes the git client metrics
// This should be called once, typically from the main application
func InitMetrics(registerer prometheus.Registerer) {
	metricsOnce.Do(func() {
		gitPushInFlightGauge = prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "nephoran_git_push_in_flight",
			Help: "Number of git push operations currently in flight",
		})

		if registerer != nil {
			registerer.MustRegister(gitPushInFlightGauge)
		}
	})
}

// StatusInfo holds git repository status information
type StatusInfo struct {
	Clean         bool              `json:"clean"`
	Modified      []string          `json:"modified"`
	Added         []string          `json:"added"`
	Deleted       []string          `json:"deleted"`
	Untracked     []string          `json:"untracked"`
	CurrentBranch string            `json:"current_branch"`
	Details       map[string]string `json:"details"`
}

// CommitInfo holds information about a git commit
type CommitInfo struct {
	Hash      string    `json:"hash"`
	Message   string    `json:"message"`
	Author    string    `json:"author"`
	Email     string    `json:"email"`
	Timestamp time.Time `json:"timestamp"`
}

// PullRequestOptions holds options for creating a pull request
type PullRequestOptions struct {
	Title        string `json:"title"`
	Description  string `json:"description"`
	SourceBranch string `json:"source_branch"`
	TargetBranch string `json:"target_branch"`
	Assignees    []string `json:"assignees,omitempty"`
	Reviewers    []string `json:"reviewers,omitempty"`
	Labels       []string `json:"labels,omitempty"`
}

// PullRequestInfo holds information about a created pull request
type PullRequestInfo struct {
	Number    int    `json:"number"`
	URL       string `json:"url"`
	Title     string `json:"title"`
	State     string `json:"state"`
	CreatedAt time.Time `json:"created_at"`
}

// ClientInterface defines the interface for a Git client.
type ClientInterface interface {
	// File and commit operations
	CommitFiles(ctx context.Context, files map[string]string, message string) (string, error)
	CommitAndPush(files map[string]string, message string) (string, error)
	CommitAndPushChanges(message string) error
	
	// Branch operations
	CreateBranch(ctx context.Context, branchName, baseBranch string) error
	SwitchBranch(ctx context.Context, branchName string) error
	GetCurrentBranch(ctx context.Context) (string, error)
	ListBranches(ctx context.Context) ([]string, error)
	
	// File operations
	GetFileContent(ctx context.Context, filePath string) (string, error)
	DeleteFile(ctx context.Context, filePath, message string) (string, error)
	
	// Repository operations
	InitRepo() error
	Push(ctx context.Context, branch string) error
	Pull(ctx context.Context, branch string) error
	GetStatus(ctx context.Context) (*StatusInfo, error)
	GetCommitHistory(ctx context.Context, limit int) ([]CommitInfo, error)
	
	// Directory operations
	RemoveDirectory(path string, commitMessage string) error
	RemoveAndPush(path string, commitMessage string) (string, error)
	
	// Pull request operations
	CreatePullRequest(ctx context.Context, options *PullRequestOptions) (*PullRequestInfo, error)
}

// Config holds configuration for creating a Git client.
type Config struct {
	RepoURL             string
	Branch              string
	Token               string // Token loaded from file or environment
	TokenPath           string // Optional path to token file
	RepoPath            string
	HTTPClient          *http.Client // Optional HTTP client for API calls
	Timeout             time.Duration // Timeout for operations
	ConcurrentPushLimit int // Maximum concurrent git operations (default 4 if <= 0)
}

// ClientConfig holds configuration for creating a Git client (for backward compatibility).
type ClientConfig struct {
	RepoURL             string
	Branch              string
	Token               string // Token loaded from file or environment
	TokenPath           string // Optional path to token file
	RepoPath            string
	Logger              *slog.Logger
	ConcurrentPushLimit int // Maximum concurrent git operations (default 4 if <= 0)
}

// Client implements the Git client.
type Client struct {
	RepoURL  string
	Branch   string
	SshKey   string
	RepoPath string
	logger   *slog.Logger
	pushSem  chan struct{} // Semaphore for concurrent git operations (buffered channel with capacity 4)

	// Test hooks - only used during testing, unexported
	// These are nil in production and only set during tests
	beforePushHook func()
	afterPushHook  func()
}

// NewGitClientConfig creates a new client configuration with token loading support.
// If tokenPath is provided, it reads the token from the file.
// If file reading fails or tokenPath is empty, it falls back to the provided token.
func NewGitClientConfig(repoURL, branch, token, tokenPath string) (*ClientConfig, error) {
	config := &ClientConfig{
		RepoURL:             repoURL,
		Branch:              branch,
		RepoPath:            "/tmp/deployment-repo",
		Logger:              slog.Default().With("component", "git-client"),
		ConcurrentPushLimit: 4, // Default value
	}

	// Override from environment variable if set
	if val := os.Getenv("GIT_CONCURRENT_PUSH_LIMIT"); val != "" {
		if limit, err := strconv.Atoi(val); err == nil && limit > 0 {
			config.ConcurrentPushLimit = limit
			config.Logger.Debug("Using custom concurrent push limit from env", "limit", limit)
		} else {
			config.Logger.Debug("Invalid GIT_CONCURRENT_PUSH_LIMIT, using default", "value", val, "default", 4)
		}
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

	// Use configured limit or default to 4
	limit := config.ConcurrentPushLimit
	if limit <= 0 {
		limit = 4
	}

	return &Client{
		RepoURL:  config.RepoURL,
		Branch:   config.Branch,
		SshKey:   config.Token,
		RepoPath: config.RepoPath,
		logger:   config.Logger,
		pushSem:  make(chan struct{}, limit), // Initialize semaphore with configurable capacity
	}
}

// NewClient creates a new Git client.
func NewClient(repoURL, branch, sshKey string) *Client {
	// Create a default logger if none provided
	logger := slog.Default().With("component", "git-client")

	// Read concurrent push limit from environment or use default
	limit := 4
	if val := os.Getenv("GIT_CONCURRENT_PUSH_LIMIT"); val != "" {
		if l, err := strconv.Atoi(val); err == nil && l > 0 {
			limit = l
			logger.Debug("Using custom concurrent push limit from env", "limit", limit)
		}
	}

	return &Client{
		RepoURL:  repoURL,
		Branch:   branch,
		SshKey:   sshKey,
		RepoPath: "/tmp/deployment-repo",
		logger:   logger,
		pushSem:  make(chan struct{}, limit), // Initialize semaphore with configurable capacity
	}
}

// NewClientWithLogger creates a new Git client with a specific logger.
func NewClientWithLogger(repoURL, branch, sshKey string, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("component", "git-client")

	// Read concurrent push limit from environment or use default
	limit := 4
	if val := os.Getenv("GIT_CONCURRENT_PUSH_LIMIT"); val != "" {
		if l, err := strconv.Atoi(val); err == nil && l > 0 {
			limit = l
			logger.Debug("Using custom concurrent push limit from env", "limit", limit)
		}
	}

	return &Client{
		RepoURL:  repoURL,
		Branch:   branch,
		SshKey:   sshKey,
		RepoPath: "/tmp/deployment-repo",
		logger:   logger,
		pushSem:  make(chan struct{}, limit), // Initialize semaphore with configurable capacity
	}
}

// acquireSemaphore acquires the semaphore for git operations with debug logging.
func (c *Client) acquireSemaphore(operation string) {
	// Handle case where client was not created with constructor (tests)
	if c.pushSem == nil {
		c.pushSem = make(chan struct{}, 4)
	}
	if c.logger == nil {
		c.logger = slog.Default().With("component", "git-client")
	}

	// Try to acquire immediately first
	select {
	case c.pushSem <- struct{}{}:
		// Successfully acquired immediately
		inFlight := len(c.pushSem)
		c.logger.Debug("git push: acquired semaphore",
			"operation", operation,
			"in_flight", inFlight,
			"limit", cap(c.pushSem),
			"acquired_immediately", true,
			"goroutine", runtime.NumGoroutine())

		// Update metrics if available
		if gitPushInFlightGauge != nil {
			gitPushInFlightGauge.Set(float64(inFlight))
		}
		return
	default:
		// Would block, log that we're waiting
		c.logger.Debug("git push: waiting on semaphore",
			"operation", operation,
			"in_flight", len(c.pushSem),
			"limit", cap(c.pushSem),
			"goroutine", runtime.NumGoroutine())
	}

	// Now block and wait for acquisition
	c.pushSem <- struct{}{}
	inFlight := len(c.pushSem)
	c.logger.Debug("git push: acquired semaphore",
		"operation", operation,
		"in_flight", inFlight,
		"limit", cap(c.pushSem),
		"acquired_immediately", false,
		"goroutine", runtime.NumGoroutine())

	// Update metrics if available
	if gitPushInFlightGauge != nil {
		gitPushInFlightGauge.Set(float64(inFlight))
	}
}

// releaseSemaphore releases the semaphore for git operations with debug logging.
func (c *Client) releaseSemaphore(operation string) {
	// Handle case where client was not created with constructor (tests)
	if c.pushSem == nil || c.logger == nil {
		return // No semaphore to release, nothing to do
	}

	defer func() {
		if r := recover(); r != nil {
			c.logger.Error("Panic during semaphore release",
				"operation", operation,
				"panic", r,
				"goroutine", runtime.NumGoroutine())
			panic(r) // Re-panic after logging
		}
	}()

	select {
	case <-c.pushSem:
		inFlight := len(c.pushSem)
		c.logger.Debug("git push: released semaphore",
			"operation", operation,
			"in_flight", inFlight,
			"limit", cap(c.pushSem),
			"goroutine", runtime.NumGoroutine())

		// Update metrics if available
		if gitPushInFlightGauge != nil {
			gitPushInFlightGauge.Set(float64(inFlight))
		}
	default:
		c.logger.Warn("git push: attempted to release semaphore when none held",
			"operation", operation,
			"in_flight", len(c.pushSem),
			"limit", cap(c.pushSem),
			"goroutine", runtime.NumGoroutine())
	}
}

// InitRepo clones the repository if it doesn't exist locally.
func (c *Client) InitRepo() error {
	c.acquireSemaphore("InitRepo")
	defer c.releaseSemaphore("InitRepo")

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
	c.acquireSemaphore("CommitAndPush")
	defer c.releaseSemaphore("CommitAndPush")

	// Test hook - before push operations
	if c.beforePushHook != nil {
		c.beforePushHook()
	}
	// Test hook - cleanup after push
	if c.afterPushHook != nil {
		defer c.afterPushHook()
	}

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
	c.acquireSemaphore("CommitAndPushChanges")
	defer c.releaseSemaphore("CommitAndPushChanges")

	// Test hook - before push operations
	if c.beforePushHook != nil {
		c.beforePushHook()
	}
	// Test hook - cleanup after push
	if c.afterPushHook != nil {
		defer c.afterPushHook()
	}

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
	c.acquireSemaphore("RemoveDirectory")
	defer c.releaseSemaphore("RemoveDirectory")

	// Test hook - before push operations
	if c.beforePushHook != nil {
		c.beforePushHook()
	}
	// Test hook - cleanup after push
	if c.afterPushHook != nil {
		defer c.afterPushHook()
	}

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

// RemoveAndPush removes a directory from the repository and commits and pushes the change
// Returns the commit hash of the created commit.
func (c *Client) RemoveAndPush(path string, commitMessage string) (string, error) {
	c.acquireSemaphore("RemoveAndPush")
	defer c.releaseSemaphore("RemoveAndPush")

	// Test hook - before push operations
	if c.beforePushHook != nil {
		c.beforePushHook()
	}
	// Test hook - cleanup after push
	if c.afterPushHook != nil {
		defer c.afterPushHook()
	}

	r, err := git.PlainOpen(c.RepoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repo: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}

	// Check if directory exists before attempting removal
	fullPath := filepath.Join(c.RepoPath, path)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		// Directory doesn't exist, nothing to remove - this is not an error
		return "", nil
	}

	// Remove directory from filesystem
	if err := os.RemoveAll(fullPath); err != nil {
		return "", fmt.Errorf("failed to remove directory %s: %w", fullPath, err)
	}

	// Get status to find all files that were deleted
	status, err := w.Status()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree status: %w", err)
	}

	// Stage all deletions within the removed directory
	filesStaged := 0
	for file := range status {
		fileStatus := status[file]
		// Stage files that are deleted and within the target path
		if fileStatus.Worktree == git.Deleted && strings.HasPrefix(file, path) {
			if _, err := w.Add(file); err != nil {
				return "", fmt.Errorf("failed to stage deletion of %s: %w", file, err)
			}
			filesStaged++
		}
	}

	// If no files were staged for deletion, the directory was already empty or didn't contain tracked files
	if filesStaged == 0 {
		// No changes to commit, which is fine
		return "", nil
	}

	// Commit the changes
	commit, err := w.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Nephio Bridge",
			Email: "nephio-bridge@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit directory removal: %w", err)
	}

	// Get commit object for hash
	commitObj, err := r.CommitObject(commit)
	if err != nil {
		return "", fmt.Errorf("failed to get commit object: %w", err)
	}

	// Push the changes
	auth, err := ssh.NewPublicKeys("git", []byte(c.SshKey), "")
	if err != nil {
		return "", fmt.Errorf("failed to create ssh auth: %w", err)
	}

	err = r.Push(&git.PushOptions{
		RemoteName: "origin",
		Auth:       auth,
	})
	if err != nil {
		return "", fmt.Errorf("failed to push directory removal: %w", err)
	}

	return commitObj.Hash.String(), nil
}

// NewGitClient creates a new Git client instance with the provided configuration
// This function provides backward compatibility for existing code
func NewGitClient(config *ClientConfig) *Client {
	if config == nil {
		return &Client{
			logger:  slog.Default().With("component", "git-client"),
			pushSem: make(chan struct{}, 4),
		}
	}
	return NewClientFromConfig(config)
}

// NewClientWithConfig creates a new Git client from configuration
func NewClientWithConfig(config *Config, logger *logging.StructuredLogger) (ClientInterface, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if logger == nil {
		logger = logging.NewStructuredLogger(logging.DefaultConfig("git-client", "1.0.0", "production"))
	}

	// Use configured limit or default to 4
	limit := config.ConcurrentPushLimit
	if limit <= 0 {
		limit = 4
	}

	return &Client{
		RepoURL:  config.RepoURL,
		Branch:   config.Branch,
		SshKey:   config.Token,
		RepoPath: config.RepoPath,
		logger:   logger.Logger, // Extract the underlying slog.Logger
		pushSem:  make(chan struct{}, limit),
	}, nil
}

// CommitFiles commits files to the repository (implements ClientInterface)
func (c *Client) CommitFiles(ctx context.Context, files map[string]string, message string) (string, error) {
	return c.CommitAndPush(files, message)
}

// CreateBranch creates a new branch (stub implementation)
func (c *Client) CreateBranch(ctx context.Context, branchName, baseBranch string) error {
	return fmt.Errorf("CreateBranch not implemented in basic git client")
}

// SwitchBranch switches to a branch (stub implementation) 
func (c *Client) SwitchBranch(ctx context.Context, branchName string) error {
	return fmt.Errorf("SwitchBranch not implemented in basic git client")
}

// GetCurrentBranch gets current branch (stub implementation)
func (c *Client) GetCurrentBranch(ctx context.Context) (string, error) {
	return c.Branch, nil
}

// ListBranches lists branches (stub implementation)
func (c *Client) ListBranches(ctx context.Context) ([]string, error) {
	return []string{c.Branch}, nil
}

// GetFileContent gets file content (stub implementation)
func (c *Client) GetFileContent(ctx context.Context, filePath string) (string, error) {
	return "", fmt.Errorf("GetFileContent not implemented in basic git client")
}

// DeleteFile deletes a file (stub implementation)
func (c *Client) DeleteFile(ctx context.Context, filePath, message string) (string, error) {
	return "", fmt.Errorf("DeleteFile not implemented in basic git client")
}

// Push pushes to remote (stub implementation)
func (c *Client) Push(ctx context.Context, branch string) error {
	return fmt.Errorf("Push not implemented in basic git client")
}

// Pull pulls from remote (stub implementation)
func (c *Client) Pull(ctx context.Context, branch string) error {
	return fmt.Errorf("Pull not implemented in basic git client")
}

// GetStatus gets repository status (stub implementation)
func (c *Client) GetStatus(ctx context.Context) (*StatusInfo, error) {
	return &StatusInfo{
		Clean:         true,
		CurrentBranch: c.Branch,
	}, nil
}

// GetCommitHistory gets commit history (stub implementation)
func (c *Client) GetCommitHistory(ctx context.Context, limit int) ([]CommitInfo, error) {
	return []CommitInfo{}, nil
}

// CreatePullRequest creates a pull request (stub implementation)
func (c *Client) CreatePullRequest(ctx context.Context, options *PullRequestOptions) (*PullRequestInfo, error) {
	return nil, fmt.Errorf("CreatePullRequest not implemented in basic git client")
}
