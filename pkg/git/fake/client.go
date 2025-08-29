// FIXME: Adding package comment per revive linter.

// Package fake provides mock implementations of Git client interfaces for testing.


package fake



import (

	"fmt"



	"github.com/thc1006/nephoran-intent-operator/pkg/git"

)



// Client is a fake implementation of git.ClientInterface for testing.

type Client struct {

	// ShouldFailCommitAndPush controls whether CommitAndPush should fail.

	ShouldFailCommitAndPush bool

	// ShouldFailInitRepo controls whether InitRepo should fail.

	ShouldFailInitRepo bool

	// ShouldFailRemoveDirectory controls whether RemoveDirectory should fail.

	ShouldFailRemoveDirectory bool

	// ShouldFailCommitAndPushChanges controls whether CommitAndPushChanges should fail.

	ShouldFailCommitAndPushChanges bool

	// ShouldFailWithPushError controls whether to simulate a Git push failure.

	ShouldFailWithPushError bool



	// New failure flags for the missing methods.

	ShouldFailCommitFiles      bool

	ShouldFailCreateBranch     bool

	ShouldFailSwitchBranch     bool

	ShouldFailGetCurrentBranch bool

	ShouldFailListBranches     bool

	ShouldFailGetFileContent   bool



	// CallHistory tracks method calls for verification in tests.

	CallHistory []string



	// CommitHash is the hash that will be returned by CommitAndPush.

	CommitHash string



	// Mock data for testing.

	CurrentBranch string

	Branches      []string

	FileContents  map[string][]byte // path -> content mapping

}



// NewClient creates a new fake Git client.

func NewClient() *Client {

	return &Client{

		CallHistory:   make([]string, 0),

		CommitHash:    "fake-commit-hash-12345678",

		CurrentBranch: "main",

		Branches:      []string{"main", "dev", "feature-branch"},

		FileContents:  make(map[string][]byte),

	}

}



// Ensure Client implements the GitClientInterface.

var _ git.ClientInterface = (*Client)(nil)



// InitRepo fake implementation.

func (c *Client) InitRepo() error {

	c.CallHistory = append(c.CallHistory, "InitRepo")

	if c.ShouldFailInitRepo {

		return fmt.Errorf("fake InitRepo error")

	}

	return nil

}



// CommitAndPush fake implementation.

func (c *Client) CommitAndPush(files map[string]string, message string) (string, error) {

	c.CallHistory = append(c.CallHistory, fmt.Sprintf("CommitAndPush(files=%d, message=%s)", len(files), message))

	if c.ShouldFailCommitAndPush {

		return "", fmt.Errorf("fake CommitAndPush error")

	}

	return c.CommitHash, nil

}



// CommitAndPushChanges fake implementation.

func (c *Client) CommitAndPushChanges(message string) error {

	c.CallHistory = append(c.CallHistory, fmt.Sprintf("CommitAndPushChanges(message=%s)", message))

	if c.ShouldFailCommitAndPushChanges {

		return fmt.Errorf("fake CommitAndPushChanges error")

	}

	return nil

}



// RemoveDirectory fake implementation.

func (c *Client) RemoveDirectory(path string, commitMessage string) error {

	c.CallHistory = append(c.CallHistory, fmt.Sprintf("RemoveDirectory(path=%s, commitMessage=%s)", path, commitMessage))

	if c.ShouldFailRemoveDirectory {

		// Simulate different types of failures.

		if c.ShouldFailWithPushError {

			return fmt.Errorf("failed to push directory removal: remote rejected push")

		}

		return fmt.Errorf("fake RemoveDirectory error")

	}

	return nil

}



// Reset clears the call history and resets failure flags.

func (c *Client) Reset() {

	c.CallHistory = make([]string, 0)

	c.ShouldFailCommitAndPush = false

	c.ShouldFailInitRepo = false

	c.ShouldFailRemoveDirectory = false

	c.ShouldFailCommitAndPushChanges = false

	c.ShouldFailWithPushError = false

	c.ShouldFailCommitFiles = false

	c.ShouldFailCreateBranch = false

	c.ShouldFailSwitchBranch = false

	c.ShouldFailGetCurrentBranch = false

	c.ShouldFailListBranches = false

	c.ShouldFailGetFileContent = false

}



// SetCommitHash sets the commit hash returned by CommitAndPush.

func (c *Client) SetCommitHash(hash string) {

	c.CommitHash = hash

}



// GetCallHistory returns the history of method calls.

func (c *Client) GetCallHistory() []string {

	return c.CallHistory

}



// CommitFiles fake implementation.

func (c *Client) CommitFiles(files []string, msg string) error {

	c.CallHistory = append(c.CallHistory, fmt.Sprintf("CommitFiles(files=%v, msg=%s)", files, msg))

	if c.ShouldFailCommitFiles {

		return fmt.Errorf("fake CommitFiles error")

	}

	return nil

}



// CreateBranch fake implementation.

func (c *Client) CreateBranch(name string) error {

	c.CallHistory = append(c.CallHistory, fmt.Sprintf("CreateBranch(name=%s)", name))

	if c.ShouldFailCreateBranch {

		return fmt.Errorf("fake CreateBranch error")

	}

	// Add the branch to our mock branches list if not already present.

	for _, branch := range c.Branches {

		if branch == name {

			return nil // Branch already exists

		}

	}

	c.Branches = append(c.Branches, name)

	return nil

}



// SwitchBranch fake implementation.

func (c *Client) SwitchBranch(name string) error {

	c.CallHistory = append(c.CallHistory, fmt.Sprintf("SwitchBranch(name=%s)", name))

	if c.ShouldFailSwitchBranch {

		return fmt.Errorf("fake SwitchBranch error")

	}

	// Check if branch exists.

	branchExists := false

	for _, branch := range c.Branches {

		if branch == name {

			branchExists = true

			break

		}

	}

	if !branchExists {

		return fmt.Errorf("branch %s does not exist", name)

	}

	c.CurrentBranch = name

	return nil

}



// GetCurrentBranch fake implementation.

func (c *Client) GetCurrentBranch() (string, error) {

	c.CallHistory = append(c.CallHistory, "GetCurrentBranch()")

	if c.ShouldFailGetCurrentBranch {

		return "", fmt.Errorf("fake GetCurrentBranch error")

	}

	return c.CurrentBranch, nil

}



// ListBranches fake implementation.

func (c *Client) ListBranches() ([]string, error) {

	c.CallHistory = append(c.CallHistory, "ListBranches()")

	if c.ShouldFailListBranches {

		return nil, fmt.Errorf("fake ListBranches error")

	}

	// Return a copy to prevent external modifications.

	branches := make([]string, len(c.Branches))

	copy(branches, c.Branches)

	return branches, nil

}



// GetFileContent fake implementation.

func (c *Client) GetFileContent(path string) ([]byte, error) {

	c.CallHistory = append(c.CallHistory, fmt.Sprintf("GetFileContent(path=%s)", path))

	if c.ShouldFailGetFileContent {

		return nil, fmt.Errorf("fake GetFileContent error")

	}

	content, exists := c.FileContents[path]

	if !exists {

		return nil, fmt.Errorf("file %s not found", path)

	}

	// Return a copy to prevent external modifications.

	result := make([]byte, len(content))

	copy(result, content)

	return result, nil

}



// SetFileContent sets the content for a file (for testing purposes).

func (c *Client) SetFileContent(path string, content []byte) {

	if c.FileContents == nil {

		c.FileContents = make(map[string][]byte)

	}

	c.FileContents[path] = content

}



// SetCurrentBranch sets the current branch (for testing purposes).

func (c *Client) SetCurrentBranch(branch string) {

	c.CurrentBranch = branch

}



// SetBranches sets the list of branches (for testing purposes).

func (c *Client) SetBranches(branches []string) {

	c.Branches = branches

}

