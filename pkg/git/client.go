package git

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport/ssh"
)

// ClientInterface defines the interface for a Git client.
type ClientInterface interface {
	CommitAndPush(files map[string]string, message string) (string, error)
	InitRepo() error
}

// Client implements the Git client.
type Client struct {
	RepoURL  string
	Branch   string
	SshKey   string
	RepoPath string
}

// NewClient creates a new Git client.
func NewClient(repoURL, branch, sshKey string) *Client {
	return &Client{
		RepoURL:  repoURL,
		Branch:   branch,
		SshKey:   sshKey,
		RepoPath: "/tmp/deployment-repo",
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
			return "", fmt.Errorf("failed to create directory for %s: %w", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			return "", fmt.Errorf("failed to write file %s: %w", path, err)
		}
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
