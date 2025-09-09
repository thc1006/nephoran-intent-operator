package fake

import (
	"fmt"

	"github.com/thc1006/nephoran-intent-operator/pkg/git"
)

// Client implements a fake git.ClientInterface for testing
type Client struct {
	// Track method calls for test verification
	CommitAndPushCalls        int
	CommitAndPushChangesCalls int
	InitRepoCalls             int
	RemoveDirectoryCalls      int
	
	// CallHistory tracks all method calls for pattern matching
	CallHistory               []string

	// Control return values
	ShouldFailCommitAndPush bool
	ShouldFailInit          bool
	CommitHash              string
}

// NewClient creates a new fake git client
func NewClient() *Client {
	return &Client{
		CommitHash: "fake-commit-hash-123",
	}
}

// CommitAndPush implements git.ClientInterface
func (c *Client) CommitAndPush(files map[string]string, message string) (string, error) {
	c.CommitAndPushCalls++
	c.CallHistory = append(c.CallHistory, fmt.Sprintf("CommitAndPush(%d files, %s)", len(files), message))
	if c.ShouldFailCommitAndPush {
		return "", fmt.Errorf("fake commit and push failed")
	}
	return c.CommitHash, nil
}

// CommitAndPushChanges implements git.ClientInterface
func (c *Client) CommitAndPushChanges(message string) error {
	c.CommitAndPushChangesCalls++
	c.CallHistory = append(c.CallHistory, fmt.Sprintf("CommitAndPushChanges(%s)", message))
	if c.ShouldFailCommitAndPush {
		return fmt.Errorf("fake commit and push changes failed")
	}
	return nil
}

// InitRepo implements git.ClientInterface
func (c *Client) InitRepo() error {
	c.InitRepoCalls++
	c.CallHistory = append(c.CallHistory, "InitRepo")
	if c.ShouldFailInit {
		return fmt.Errorf("fake init repo failed")
	}
	return nil
}

// RemoveDirectory implements git.ClientInterface
func (c *Client) RemoveDirectory(path string, commitMessage string) error {
	c.RemoveDirectoryCalls++
	c.CallHistory = append(c.CallHistory, fmt.Sprintf("RemoveDirectory(%s, %s)", path, commitMessage))
	return nil
}

// CommitFiles implements git.ClientInterface
func (c *Client) CommitFiles(files []string, message string) error {
	// Fake implementation for testing - just track the call
	if c.ShouldFailCommitAndPush {
		return fmt.Errorf("fake commit files failed")
	}
	return nil
}

// CreateBranch implements git.ClientInterface
func (c *Client) CreateBranch(branchName string) error {
	// Fake implementation for testing
	if c.ShouldFailInit {
		return fmt.Errorf("fake create branch failed")
	}
	return nil
}

// GetCurrentBranch implements git.ClientInterface
func (c *Client) GetCurrentBranch() (string, error) {
	// Fake implementation for testing
	if c.ShouldFailInit {
		return "", fmt.Errorf("fake get current branch failed")
	}
	return "main", nil
}

// GetFileContent implements git.ClientInterface
func (c *Client) GetFileContent(filePath string) ([]byte, error) {
	// Fake implementation for testing
	if c.ShouldFailInit {
		return nil, fmt.Errorf("fake get file content failed")
	}
	return []byte("fake file content"), nil
}

// ListBranches implements git.ClientInterface
func (c *Client) ListBranches() ([]string, error) {
	// Fake implementation for testing
	if c.ShouldFailInit {
		return nil, fmt.Errorf("fake list branches failed")
	}
	return []string{"main", "develop"}, nil
}

// SwitchBranch implements git.ClientInterface
func (c *Client) SwitchBranch(branchName string) error {
	// Fake implementation for testing
	if c.ShouldFailInit {
		return fmt.Errorf("fake switch branch failed")
	}
	return nil
}

// Reset resets the fake client state for test isolation
func (c *Client) Reset() {
	c.CommitAndPushCalls = 0
	c.CommitAndPushChangesCalls = 0
	c.InitRepoCalls = 0
	c.RemoveDirectoryCalls = 0
	c.ShouldFailCommitAndPush = false
	c.ShouldFailInit = false
	c.CommitHash = "fake-commit-hash-123"
}

// Verify that Client implements git.ClientInterface at compile time
var _ git.ClientInterface = (*Client)(nil)