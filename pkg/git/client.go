package git

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
	"github.com/prometheus/client_golang/prometheus"
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

// CommitInfo represents information about a Git commit
type CommitInfo struct {
	Hash      string
	Message   string
	Author    string
	Email     string
	Timestamp time.Time
}

// StatusInfo represents the status of a file in the Git repository
type StatusInfo struct {
	Path     string
	Status   string // Modified, Added, Deleted, Untracked, etc.
	Staging  string // Status in staging area
	Worktree string // Status in working tree
}

// PullRequestOptions contains options for creating a pull request
type PullRequestOptions struct {
	Title       string
	Description string
	SourceBranch string
	TargetBranch string
	Labels      []string
	Assignees   []string
}

// PullRequestInfo represents information about a pull request
type PullRequestInfo struct {
	ID          int
	Number      int
	Title       string
	Description string
	State       string // open, closed, merged
	SourceBranch string
	TargetBranch string
	Author      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ClientInterface defines the interface for a Git client.
type ClientInterface interface {
	CommitAndPush(files map[string]string, message string) (string, error)
	CommitAndPushChanges(message string) error
	InitRepo() error
	RemoveDirectory(path string, commitMessage string) error
	
	// New methods
	CommitFiles(files []string, msg string) error
	CreateBranch(name string) error
	SwitchBranch(name string) error
	GetCurrentBranch() (string, error)
	ListBranches() ([]string, error)
	GetFileContent(path string) ([]byte, error)
}

// ClientConfig holds configuration for creating a Git client.
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

// CommitFiles commits specified files with a message without pushing
func (c *Client) CommitFiles(files []string, msg string) error {
	c.acquireSemaphore("CommitFiles")
	defer c.releaseSemaphore("CommitFiles")

	r, err := git.PlainOpen(c.RepoPath)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Stage specified files
	for _, file := range files {
		if _, err := w.Add(file); err != nil {
			return fmt.Errorf("failed to add file %s: %w", file, err)
		}
	}

	// Commit the changes
	_, err = w.Commit(msg, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Nephio Bridge",
			Email: "nephio-bridge@example.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return fmt.Errorf("failed to commit: %w", err)
	}

	return nil
}

// CreateBranch creates a new branch from the current HEAD
func (c *Client) CreateBranch(name string) error {
	c.acquireSemaphore("CreateBranch")
	defer c.releaseSemaphore("CreateBranch")

	r, err := git.PlainOpen(c.RepoPath)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Get HEAD reference
	headRef, err := r.Head()
	if err != nil {
		return fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Create new branch reference
	branchRef := plumbing.NewBranchReferenceName(name)
	ref := plumbing.NewHashReference(branchRef, headRef.Hash())
	
	err = r.Storer.SetReference(ref)
	if err != nil {
		return fmt.Errorf("failed to create branch %s: %w", name, err)
	}

	return nil
}

// SwitchBranch switches to the specified branch
func (c *Client) SwitchBranch(name string) error {
	c.acquireSemaphore("SwitchBranch")
	defer c.releaseSemaphore("SwitchBranch")

	r, err := git.PlainOpen(c.RepoPath)
	if err != nil {
		return fmt.Errorf("failed to open repo: %w", err)
	}

	w, err := r.Worktree()
	if err != nil {
		return fmt.Errorf("failed to get worktree: %w", err)
	}

	// Checkout the branch
	err = w.Checkout(&git.CheckoutOptions{
		Branch: plumbing.NewBranchReferenceName(name),
	})
	if err != nil {
		return fmt.Errorf("failed to switch to branch %s: %w", name, err)
	}

	// Update the client's branch field
	c.Branch = name

	return nil
}

// GetCurrentBranch returns the name of the current branch
func (c *Client) GetCurrentBranch() (string, error) {
	c.acquireSemaphore("GetCurrentBranch")
	defer c.releaseSemaphore("GetCurrentBranch")

	r, err := git.PlainOpen(c.RepoPath)
	if err != nil {
		return "", fmt.Errorf("failed to open repo: %w", err)
	}

	headRef, err := r.Head()
	if err != nil {
		return "", fmt.Errorf("failed to get HEAD: %w", err)
	}

	// Extract branch name from reference
	if headRef.Name().IsBranch() {
		return headRef.Name().Short(), nil
	}

	// If we're in detached HEAD state, return the hash
	return headRef.Hash().String()[:7], nil
}

// ListBranches returns a list of all local branches
func (c *Client) ListBranches() ([]string, error) {
	c.acquireSemaphore("ListBranches")
	defer c.releaseSemaphore("ListBranches")

	r, err := git.PlainOpen(c.RepoPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open repo: %w", err)
	}

	refs, err := r.References()
	if err != nil {
		return nil, fmt.Errorf("failed to get references: %w", err)
	}

	var branches []string
	err = refs.ForEach(func(ref *plumbing.Reference) error {
		if ref.Name().IsBranch() {
			branches = append(branches, ref.Name().Short())
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to iterate references: %w", err)
	}

	return branches, nil
}

// GetFileContent reads and returns the content of a file from the repository
func (c *Client) GetFileContent(path string) ([]byte, error) {
	c.acquireSemaphore("GetFileContent")
	defer c.releaseSemaphore("GetFileContent")

	fullPath := filepath.Join(c.RepoPath, path)
	
	// Check if file exists
	if _, err := os.Stat(fullPath); err != nil {
		if os.IsNotExist(err) {
			return nil, &fs.PathError{
				Op:   "open",
				Path: path,
				Err:  fs.ErrNotExist,
			}
		}
		return nil, fmt.Errorf("failed to stat file %s: %w", path, err)
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file %s: %w", path, err)
	}

	return content, nil
}
