package fake

import (
	"fmt"

	"github.com/thc1006/nephoran-intent-operator/pkg/git"
)

// Client is a fake implementation of git.ClientInterface for testing
type Client struct {
	// ShouldFailCommitAndPush controls whether CommitAndPush should fail
	ShouldFailCommitAndPush bool
	// ShouldFailInitRepo controls whether InitRepo should fail
	ShouldFailInitRepo bool
	// ShouldFailRemoveDirectory controls whether RemoveDirectory should fail
	ShouldFailRemoveDirectory bool
	// ShouldFailCommitAndPushChanges controls whether CommitAndPushChanges should fail
	ShouldFailCommitAndPushChanges bool
	// ShouldFailRemoveAndPush controls whether RemoveAndPush should fail
	ShouldFailRemoveAndPush bool
	// ShouldFailWithPushError controls whether to simulate a Git push failure
	ShouldFailWithPushError bool

	// CallHistory tracks method calls for verification in tests
	CallHistory []string

	// CommitHash is the hash that will be returned by CommitAndPush
	CommitHash string
}

// NewClient creates a new fake Git client
func NewClient() *Client {
	return &Client{
		CallHistory: make([]string, 0),
		CommitHash:  "fake-commit-hash-12345678",
	}
}

// Ensure Client implements the GitClientInterface
var _ git.ClientInterface = (*Client)(nil)

// InitRepo fake implementation
func (c *Client) InitRepo() error {
	c.CallHistory = append(c.CallHistory, "InitRepo")
	if c.ShouldFailInitRepo {
		return fmt.Errorf("fake InitRepo error")
	}
	return nil
}

// CommitAndPush fake implementation
func (c *Client) CommitAndPush(files map[string]string, message string) (string, error) {
	c.CallHistory = append(c.CallHistory, fmt.Sprintf("CommitAndPush(files=%d, message=%s)", len(files), message))
	if c.ShouldFailCommitAndPush {
		return "", fmt.Errorf("fake CommitAndPush error")
	}
	return c.CommitHash, nil
}

// CommitAndPushChanges fake implementation
func (c *Client) CommitAndPushChanges(message string) error {
	c.CallHistory = append(c.CallHistory, fmt.Sprintf("CommitAndPushChanges(message=%s)", message))
	if c.ShouldFailCommitAndPushChanges {
		return fmt.Errorf("fake CommitAndPushChanges error")
	}
	return nil
}

// RemoveDirectory fake implementation
func (c *Client) RemoveDirectory(path string, commitMessage string) error {
	c.CallHistory = append(c.CallHistory, fmt.Sprintf("RemoveDirectory(path=%s, commitMessage=%s)", path, commitMessage))
	if c.ShouldFailRemoveDirectory {
		// Simulate different types of failures
		if c.ShouldFailWithPushError {
			return fmt.Errorf("failed to push directory removal: remote rejected push")
		}
		return fmt.Errorf("fake RemoveDirectory error")
	}
	return nil
}

// RemoveAndPush fake implementation
func (c *Client) RemoveAndPush(path string, commitMessage string) (string, error) {
	c.CallHistory = append(c.CallHistory, fmt.Sprintf("RemoveAndPush(path=%s, commitMessage=%s)", path, commitMessage))
	if c.ShouldFailRemoveAndPush {
		// Simulate different types of failures
		if c.ShouldFailWithPushError {
			return "", fmt.Errorf("failed to push directory removal: remote rejected push")
		}
		return "", fmt.Errorf("fake RemoveAndPush error")
	}
	return c.CommitHash, nil
}

// Reset clears the call history and resets failure flags
func (c *Client) Reset() {
	c.CallHistory = make([]string, 0)
	c.ShouldFailCommitAndPush = false
	c.ShouldFailInitRepo = false
	c.ShouldFailRemoveDirectory = false
	c.ShouldFailCommitAndPushChanges = false
	c.ShouldFailRemoveAndPush = false
	c.ShouldFailWithPushError = false
}

// SetCommitHash sets the commit hash returned by CommitAndPush
func (c *Client) SetCommitHash(hash string) {
	c.CommitHash = hash
}

// GetCallHistory returns the history of method calls
func (c *Client) GetCallHistory() []string {
	return c.CallHistory
}
