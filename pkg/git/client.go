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

// Client is a simple client for performing Git operations.
type Client struct {
	RepoURL       string
	Branch        string
	SSHPrivateKey string
}

// NewClient creates a new Git client.
func NewClient(repoURL, branch, sshPrivateKey string) *Client {
	return &Client{
		RepoURL:       repoURL,
		Branch:        branch,
		SSHPrivateKey: sshPrivateKey,
	}
}

// CommitAndPush clones the repository, applies a set of modifications, and pushes the changes.
// The modify function is responsible for making the actual file changes on disk.
func (c *Client) CommitAndPush(ctx context.Context, commitMessage string, modify func(repoPath string) error) (string, error) {
	dir, err := os.MkdirTemp("", "nephio-git")
	if err != nil {
		return "", fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(dir)

	publicKeys, err := ssh.NewPublicKeys("git", []byte(c.SSHPrivateKey), "")
	if err != nil {
		return "", fmt.Errorf("failed to create public keys: %w", err)
	}

	repo, err := git.PlainClone(dir, false, &git.CloneOptions{
		URL:  c.RepoURL,
		Auth: publicKeys,
	})
	if err != nil {
		return "", fmt.Errorf("failed to clone repo: %w", err)
	}

	// Call the user-provided function to make file changes
	if err := modify(dir); err != nil {
		return "", fmt.Errorf("modification function failed: %w", err)
	}

	w, err := repo.Worktree()
	if err != nil {
		return "", fmt.Errorf("failed to get worktree: %w", err)
	}
	_, err = w.Add(".")
	if err != nil {
		return "", fmt.Errorf("failed to add files to git: %w", err)
	}

	commit, err := w.Commit(commitMessage, &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Nephio Bridge",
			Email: "nephio-bridge@nephoran.com",
			When:  time.Now(),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit changes: %w", err)
	}

	err = repo.Push(&git.PushOptions{
		Auth: publicKeys,
	})
	if err != nil {
		return "", fmt.Errorf("failed to push changes: %w", err)
	}

	return commit.String(), nil
}
