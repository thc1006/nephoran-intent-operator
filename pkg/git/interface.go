package git

import "context"

// ClientInterface defines the interface for a Git client.
type ClientInterface interface {
	CommitAndPush(ctx context.Context, commitMessage string, modify func(repoPath string) error) (string, error)
}
