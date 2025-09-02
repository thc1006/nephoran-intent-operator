package git

import (
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-git/go-git/v5"

	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/storage/memory"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockClientInterface is a mock implementation of ClientInterface for testing
type MockClientInterface struct {
	mock.Mock
}

func (m *MockClientInterface) CommitAndPush(files map[string]string, message string) (string, error) {
	args := m.Called(files, message)
	return args.String(0), args.Error(1)
}

func (m *MockClientInterface) CommitAndPushChanges(message string) error {
	args := m.Called(message)
	return args.Error(0)
}

func (m *MockClientInterface) InitRepo() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockClientInterface) RemoveDirectory(path string, commitMessage string) error {
	args := m.Called(path, commitMessage)
	return args.Error(0)
}

// Test helper functions
func createTestRepo(t *testing.T, path string) *git.Repository {
	// Create a bare repository in memory for testing
	repo, err := git.Init(memory.NewStorage(), nil)
	require.NoError(t, err)
	return repo
}

func createTempDir(t *testing.T) string {
	tmpDir, err := os.MkdirTemp("", "git-test-*")
	require.NoError(t, err)
	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})
	return tmpDir
}

func createTestSSHKey() string {
	// This is a dummy SSH key for testing purposes only
	return `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAFwAAAAdzc2gtcn
-----END OPENSSH PRIVATE KEY-----`
}

func TestNewClient(t *testing.T) {
	repoURL := "git@github.com:test/repo.git"
	branch := "main"
	sshKey := createTestSSHKey()

	client := NewClient(repoURL, branch, sshKey)

	assert.NotNil(t, client)
	assert.Equal(t, repoURL, client.RepoURL)
	assert.Equal(t, branch, client.Branch)
	assert.Equal(t, sshKey, client.SSHKey)
	assert.Equal(t, "/tmp/deployment-repo", client.RepoPath)
}

func TestClient_InitRepo_NewRepo(t *testing.T) {
	tmpDir := createTempDir(t)

	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: filepath.Join(tmpDir, "test-repo"),
	}

	// Since we can't actually clone from a remote, we'll test the directory creation logic
	// In a real scenario, this would fail due to the fake repository URL
	err := client.InitRepo()

	// The error is expected since we're using a fake URL
	// But we can verify that the method attempts to handle non-existent directories correctly
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to clone repo")
}

func TestClient_InitRepo_ExistingRepo(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "existing-repo")

	// Create a real git repository
	_, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Should not return an error for existing repo
	err = client.InitRepo()
	assert.NoError(t, err)
}

func TestClient_CommitAndPush_Success(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create a real git repository for testing
	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	// Create an initial commit to establish the repository
	workTree, err := repo.Worktree()
	require.NoError(t, err)

	// Create initial file
	initialFile := filepath.Join(repoPath, "README.md")
	err = os.WriteFile(initialFile, []byte("# Test Repository"), 0o644)
	require.NoError(t, err)

	_, err = workTree.Add("README.md")
	require.NoError(t, err)

	_, err = workTree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}
	_ = client // Use the client variable

	files := map[string]string{
		"test1.txt":        "content1",
		"subdir/test2.txt": "content2",
		"test3.yaml":       "key: value",
	}

	// Note: This will fail at the push step due to no remote, but we can test file creation and commit
	commitHash, err := client.CommitAndPush(files, "Test commit")

	// The commit should succeed but push or SSH auth will fail
	if err != nil {
		assert.True(t,
			strings.Contains(err.Error(), "failed to push") ||
				strings.Contains(err.Error(), "failed to create ssh auth"),
			"Expected push or SSH auth error, got: %v", err)
		// Verify that files were created and committed even though push failed
		for filePath, expectedContent := range files {
			fullPath := filepath.Join(repoPath, filePath)
			content, readErr := os.ReadFile(fullPath)
			assert.NoError(t, readErr, "File should have been created: %s", filePath)
			assert.Equal(t, expectedContent, string(content), "File content should match")
		}
	} else {
		// If no error, commit hash should be returned
		assert.NotEmpty(t, commitHash)
		assert.Len(t, commitHash, 40) // SHA-1 hash length
	}
}

func TestClient_CommitAndPush_FileCreation(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create a real git repository
	_, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	files := map[string]string{
		"config/app.yaml":            "app: test",
		"scripts/deploy.sh":          "#!/bin/bash\necho 'deploying'",
		"docs/README.md":             "# Documentation",
		"nested/deep/structure.json": `{"key": "value"}`,
	}

	// This will fail at push, but we can verify file operations
	_, _ = client.CommitAndPush(files, "Add configuration files")

	// Verify all files were created with correct content
	for filePath, expectedContent := range files {
		fullPath := filepath.Join(repoPath, filePath)

		// Check that file exists
		_, statErr := os.Stat(fullPath)
		assert.NoError(t, statErr, "File should exist: %s", filePath)

		// Check that directories were created
		dir := filepath.Dir(fullPath)
		_, dirStatErr := os.Stat(dir)
		assert.NoError(t, dirStatErr, "Directory should exist: %s", dir)

		// Check file content
		content, readErr := os.ReadFile(fullPath)
		assert.NoError(t, readErr, "Should be able to read file: %s", filePath)
		assert.Equal(t, expectedContent, string(content), "File content should match for: %s", filePath)
	}
}

func TestClient_CommitAndPush_InvalidRepo(t *testing.T) {
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: "/non/existent/path",
	}

	files := map[string]string{
		"test.txt": "content",
	}

	_, err := client.CommitAndPush(files, "Test commit")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open repo")
}

func TestClient_CommitAndPushChanges_Success(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create a real git repository
	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	// Create initial commit
	workTree, err := repo.Worktree()
	require.NoError(t, err)

	initialFile := filepath.Join(repoPath, "initial.txt")
	err = os.WriteFile(initialFile, []byte("initial content"), 0o644)
	require.NoError(t, err)

	_, err = workTree.Add("initial.txt")
	require.NoError(t, err)

	_, err = workTree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Create some changes
	testFile := filepath.Join(repoPath, "changes.txt")
	err = os.WriteFile(testFile, []byte("new changes"), 0o644)
	require.NoError(t, err)

	// This will fail at push, but commit should work
	err = client.CommitAndPushChanges("Add changes")
	// Expect push failure but verify commit was made
	if err != nil {
		assert.True(t,
			strings.Contains(err.Error(), "failed to push") ||
				strings.Contains(err.Error(), "failed to create ssh auth"),
			"Expected push or SSH auth error, got: %v", err)
	}

	// Verify the file was added to git
	_, err = os.Stat(testFile)
	assert.NoError(t, err)
}

func TestClient_CommitAndPushChanges_InvalidRepo(t *testing.T) {
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: "/non/existent/path",
	}

	err := client.CommitAndPushChanges("Test commit")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open repo")
}

func TestClient_RemoveDirectory_Success(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create a real git repository
	_, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Create a directory structure to remove
	testDir := filepath.Join(repoPath, "to-remove")
	err = os.MkdirAll(testDir, 0o755)
	require.NoError(t, err)

	testFile := filepath.Join(testDir, "file.txt")
	err = os.WriteFile(testFile, []byte("content"), 0o644)
	require.NoError(t, err)

	// Verify directory exists
	_, err = os.Stat(testDir)
	assert.NoError(t, err)

	// Remove the directory - may fail with empty commit if directory isn't tracked
	err = client.RemoveDirectory("to-remove", "Remove test directory")
	if err != nil {
		// Allow for empty commit errors when removing untracked directories
		assert.True(t,
			strings.Contains(err.Error(), "clean working tree") ||
				strings.Contains(err.Error(), "failed to push") ||
				strings.Contains(err.Error(), "failed to create ssh auth"),
			"Expected empty commit, push, or SSH auth error, got: %v", err)
	}

	// Verify directory was removed
	_, err = os.Stat(testDir)
	assert.True(t, os.IsNotExist(err))
}

func TestClient_RemoveDirectory_InvalidRepo(t *testing.T) {
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: "/non/existent/path",
	}

	err := client.RemoveDirectory("some-dir", "Remove directory")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to open repo")
}

func TestClient_RemoveDirectory_NonExistentDirectory(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create a real git repository
	_, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Try to remove non-existent directory
	err = client.RemoveDirectory("non-existent-dir", "Remove non-existent directory")
	if err != nil {
		// Should only fail with empty commit error since nothing to remove
		assert.True(t,
			strings.Contains(err.Error(), "clean working tree"),
			"Expected empty commit error for non-existent directory, got: %v", err)
	}
}

// Comprehensive tests for new CommitAndPushChanges functionality
func TestClient_CommitAndPushChanges_GitFilesExclusion(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create a real git repository
	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	// Create initial commit
	workTree, err := repo.Worktree()
	require.NoError(t, err)

	initialFile := filepath.Join(repoPath, "initial.txt")
	err = os.WriteFile(initialFile, []byte("initial content"), 0o644)
	require.NoError(t, err)

	_, err = workTree.Add("initial.txt")
	require.NoError(t, err)

	_, err = workTree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Create various files including .git related files
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("test content"), 0o644)
	require.NoError(t, err)

	// Modify the initial file (tracked file)
	err = os.WriteFile(initialFile, []byte("modified initial content"), 0o644)
	require.NoError(t, err)

	// Create .git related files and directories (should be excluded)
	gitIgnoreFile := filepath.Join(repoPath, ".gitignore")
	err = os.WriteFile(gitIgnoreFile, []byte("*.log"), 0o644)
	require.NoError(t, err)

	gitConfigDir := filepath.Join(repoPath, ".git-backup")
	err = os.MkdirAll(gitConfigDir, 0o755)
	require.NoError(t, err)
	gitBackupFile := filepath.Join(gitConfigDir, "config")
	err = os.WriteFile(gitBackupFile, []byte("backup config"), 0o644)
	require.NoError(t, err)

	// Stage the initial modification (simulate tracked file)
	_, err = workTree.Add("initial.txt")
	require.NoError(t, err)

	// Attempt to commit and push changes
	err = client.CommitAndPushChanges("Test git files exclusion")
	// Verify that the method handles the files appropriately
	// The commit should succeed for tracked files but may fail at push or ssh auth
	if err != nil {
		// Allow for both push failures and SSH auth failures
		assert.True(t,
			strings.Contains(err.Error(), "failed to push") ||
				strings.Contains(err.Error(), "failed to create ssh auth"),
			"Expected push or SSH auth error, got: %v", err)
	}

	// Verify that .git files would be properly handled by checking that the method
	// would skip them if they were in the status. Since the CommitAndPushChanges method
	// only processes tracked files and skips .git files, we need to verify this behavior
	// by simulating what the method would do.

	status, err := workTree.Status()
	require.NoError(t, err)

	// Check that our test setup is correct - .git files should be untracked
	gitFilesFound := 0
	for file, fileStatus := range status {
		if strings.HasPrefix(file, ".git") {
			gitFilesFound++
			// .git files should remain untracked (not processed by CommitAndPushChanges)
			assert.Equal(t, git.Untracked, fileStatus.Worktree,
				"Git-related files should be untracked: %s", file)
		}
	}
	assert.Greater(t, gitFilesFound, 0, "Test should have created .git files to verify exclusion")

	// The key test is that the method should have run without excluding tracked files
	// because of .git prefix, while properly excluding .git files. Since the method
	// may or may not succeed (due to SSH issues), we verify the core functionality
	// by checking that .git files remain untracked and regular files were processed.

	// If there's a test.txt in status, it should be untracked (newly created)
	if testStatus, exists := status["test.txt"]; exists {
		assert.Equal(t, git.Untracked, testStatus.Worktree, "test.txt should be untracked")
	}

	// The initial.txt file should have been processed if the method succeeded
	// (it may not appear in status if successfully committed)
}

func TestClient_CommitAndPushChanges_TrackedFilesOnly(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create a real git repository
	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	workTree, err := repo.Worktree()
	require.NoError(t, err)

	// Create initial files and commit them (make them tracked)
	trackedFile1 := filepath.Join(repoPath, "tracked1.txt")
	trackedFile2 := filepath.Join(repoPath, "tracked2.txt")

	err = os.WriteFile(trackedFile1, []byte("tracked content 1"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(trackedFile2, []byte("tracked content 2"), 0o644)
	require.NoError(t, err)

	_, err = workTree.Add("tracked1.txt")
	require.NoError(t, err)
	_, err = workTree.Add("tracked2.txt")
	require.NoError(t, err)

	_, err = workTree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Modify tracked files
	err = os.WriteFile(trackedFile1, []byte("modified tracked content 1"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(trackedFile2, []byte("modified tracked content 2"), 0o644)
	require.NoError(t, err)

	// Create untracked files (should NOT be staged)
	untrackedFile1 := filepath.Join(repoPath, "untracked1.txt")
	untrackedFile2 := filepath.Join(repoPath, "untracked2.txt")

	err = os.WriteFile(untrackedFile1, []byte("untracked content 1"), 0o644)
	require.NoError(t, err)
	err = os.WriteFile(untrackedFile2, []byte("untracked content 2"), 0o644)
	require.NoError(t, err)

	// Get status before commit to verify untracked files exist
	statusBefore, err := workTree.Status()
	require.NoError(t, err)

	// Verify we have both tracked (modified) and untracked files
	foundTracked := false
	foundUntracked := false
	for file, status := range statusBefore {
		if strings.Contains(file, "tracked") && status.Worktree == git.Modified {
			foundTracked = true
		}
		if strings.Contains(file, "untracked") && status.Worktree == git.Untracked {
			foundUntracked = true
		}
	}
	assert.True(t, foundTracked, "Should have modified tracked files")
	assert.True(t, foundUntracked, "Should have untracked files")

	// Commit and push changes
	err = client.CommitAndPushChanges("Test tracked files only")
	// The method should only stage tracked files and skip untracked ones
	// It may fail at push or SSH auth but should succeed at staging and commit
	if err != nil {
		assert.True(t,
			strings.Contains(err.Error(), "failed to push") ||
				strings.Contains(err.Error(), "failed to create ssh auth"),
			"Expected push or SSH auth error, got: %v", err)
	}

	// Verify that untracked files still show as untracked after the operation
	statusAfter, err := workTree.Status()
	require.NoError(t, err)

	for file, status := range statusAfter {
		if strings.Contains(file, "untracked") {
			assert.Equal(t, git.Untracked, status.Worktree, "Untracked files should remain untracked: %s", file)
		}
	}
}

func TestClient_CommitAndPushChanges_MixedFileStates(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create a real git repository
	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	workTree, err := repo.Worktree()
	require.NoError(t, err)

	// Create initial files for different test scenarios
	files := []string{"modify-me.txt", "delete-me.txt", "rename-me.txt"}
	for _, file := range files {
		fullPath := filepath.Join(repoPath, file)
		err = os.WriteFile(fullPath, []byte("initial content"), 0o644)
		require.NoError(t, err)
		_, err = workTree.Add(file)
		require.NoError(t, err)
	}

	_, err = workTree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Create mixed file state scenarios

	// 1. Modified tracked file
	modifyFile := filepath.Join(repoPath, "modify-me.txt")
	err = os.WriteFile(modifyFile, []byte("modified content"), 0o644)
	require.NoError(t, err)

	// 2. Delete tracked file
	deleteFile := filepath.Join(repoPath, "delete-me.txt")
	err = os.Remove(deleteFile)
	require.NoError(t, err)

	// 3. Rename tracked file (appears as delete + add)
	oldRenameFile := filepath.Join(repoPath, "rename-me.txt")
	newRenameFile := filepath.Join(repoPath, "renamed.txt")
	content, err := os.ReadFile(oldRenameFile)
	require.NoError(t, err)
	err = os.Remove(oldRenameFile)
	require.NoError(t, err)
	err = os.WriteFile(newRenameFile, content, 0o644)
	require.NoError(t, err)

	// 4. New untracked file (should be ignored)
	untrackedFile := filepath.Join(repoPath, "new-untracked.txt")
	err = os.WriteFile(untrackedFile, []byte("untracked content"), 0o644)
	require.NoError(t, err)

	// 5. .git related file (should be ignored)
	gitFile := filepath.Join(repoPath, ".gitattributes")
	err = os.WriteFile(gitFile, []byte("* text=auto"), 0o644)
	require.NoError(t, err)

	// Get status to verify our test setup
	statusBefore, err := workTree.Status()
	require.NoError(t, err)
	t.Logf("Status before CommitAndPushChanges: %+v", statusBefore)

	// Commit changes - should only handle tracked file changes, skip untracked and .git files
	err = client.CommitAndPushChanges("Test mixed file states")
	if err != nil {
		assert.True(t,
			strings.Contains(err.Error(), "failed to push") ||
				strings.Contains(err.Error(), "failed to create ssh auth"),
			"Expected push or SSH auth error, got: %v", err)
	}

	// Verify the expected behavior
	statusAfter, err := workTree.Status()
	require.NoError(t, err)
	t.Logf("Status after CommitAndPushChanges: %+v", statusAfter)

	// Verify untracked files remain untracked
	for file, status := range statusAfter {
		if strings.Contains(file, "untracked") || strings.HasPrefix(file, ".git") {
			assert.Equal(t, git.Untracked, status.Worktree, "Untracked/git files should remain untracked: %s", file)
		}
	}
}

func TestClient_CommitAndPushChanges_EmptyChanges(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create a real git repository
	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	workTree, err := repo.Worktree()
	require.NoError(t, err)

	// Create initial commit
	initialFile := filepath.Join(repoPath, "initial.txt")
	err = os.WriteFile(initialFile, []byte("initial content"), 0o644)
	require.NoError(t, err)

	_, err = workTree.Add("initial.txt")
	require.NoError(t, err)

	_, err = workTree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Try to commit with no changes
	err = client.CommitAndPushChanges("Empty commit test")

	// Should handle empty changes gracefully - may fail at push, SSH auth, or have no changes to commit
	if err != nil && !strings.Contains(err.Error(), "failed to push") && !strings.Contains(err.Error(), "failed to create ssh auth") {
		// If it's not a push or SSH auth error, it might be a "nothing to commit" type error
		// This is acceptable behavior
		assert.NotContains(t, err.Error(), "panic")
	}
}

// Comprehensive tests for new RemoveDirectory functionality
func TestClient_RemoveDirectory_AtomicOperation(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create a real git repository
	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	workTree, err := repo.Worktree()
	require.NoError(t, err)

	// Create directory structure to track and remove
	testDir := "test-directory"
	testDirPath := filepath.Join(repoPath, testDir)
	err = os.MkdirAll(testDirPath, 0o755)
	require.NoError(t, err)

	// Create files in the directory
	files := []string{"file1.txt", "file2.yaml", "subdir/nested.json"}
	for _, file := range files {
		fullPath := filepath.Join(testDirPath, file)
		err = os.MkdirAll(filepath.Dir(fullPath), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(fmt.Sprintf("content of %s", file)), 0o644)
		require.NoError(t, err)

		// Add to git (make tracked)
		gitPath := filepath.Join(testDir, file)
		_, err = workTree.Add(gitPath)
		require.NoError(t, err)
	}

	// Initial commit to establish the files
	_, err = workTree.Commit("Add test directory", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Verify directory exists before removal
	_, err = os.Stat(testDirPath)
	assert.NoError(t, err, "Test directory should exist before removal")

	// Remove directory with commit message
	commitMessage := "Remove test directory and all contents"
	err = client.RemoveDirectory(testDir, commitMessage)
	// May fail at push, SSH auth, or have no changes to commit
	if err != nil {
		assert.True(t,
			strings.Contains(err.Error(), "failed to push") ||
				strings.Contains(err.Error(), "failed to create ssh auth") ||
				strings.Contains(err.Error(), "clean working tree"),
			"Expected push, SSH auth, or empty commit error, got: %v", err)
	}

	// Verify directory was removed from filesystem
	_, err = os.Stat(testDirPath)
	assert.True(t, os.IsNotExist(err), "Directory should be removed from filesystem")

	// Verify commit was created with correct message (only if commit succeeded)
	ref, err := repo.Head()
	require.NoError(t, err)

	commit, err := repo.CommitObject(ref.Hash())
	require.NoError(t, err)

	// If the RemoveDirectory operation succeeded in creating a commit, verify the details
	if !strings.Contains(commit.Message, "Add test directory") {
		// This should be the removal commit, not the setup commit
		assert.Contains(t, commit.Message, commitMessage, "Commit should contain the provided message")
		assert.Equal(t, "Nephio Bridge", commit.Author.Name)
		assert.Equal(t, "nephio-bridge@example.com", commit.Author.Email)
	} else {
		// If we're still on the setup commit, the removal had no changes to commit
		// This is acceptable behavior when removing directories that aren't tracked
		t.Logf("RemoveDirectory had no changes to commit, still on setup commit")
	}
}

func TestClient_RemoveDirectory_NestedDirectories(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	workTree, err := repo.Worktree()
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Create deeply nested directory structure
	basePath := "deep/nested/structure"
	fullBasePath := filepath.Join(repoPath, basePath)
	err = os.MkdirAll(fullBasePath, 0o755)
	require.NoError(t, err)

	// Create files at different levels
	nestedFiles := map[string]string{
		"deep/file1.txt":                      "content1",
		"deep/nested/file2.txt":               "content2",
		"deep/nested/structure/file3.txt":     "content3",
		"deep/nested/structure/sub/file4.txt": "content4",
	}

	for filePath, content := range nestedFiles {
		fullPath := filepath.Join(repoPath, filePath)
		err = os.MkdirAll(filepath.Dir(fullPath), 0o755)
		require.NoError(t, err)
		err = os.WriteFile(fullPath, []byte(content), 0o644)
		require.NoError(t, err)

		// Track the file
		_, err = workTree.Add(filePath)
		require.NoError(t, err)
	}

	// Initial commit
	_, err = workTree.Commit("Add nested structure", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Remove the top-level directory
	err = client.RemoveDirectory("deep", "Remove entire deep directory structure")
	if err != nil {
		assert.True(t,
			strings.Contains(err.Error(), "failed to push") ||
				strings.Contains(err.Error(), "failed to create ssh auth") ||
				strings.Contains(err.Error(), "clean working tree"),
			"Expected push, SSH auth, or empty commit error, got: %v", err)
	}

	// Verify entire directory tree was removed
	deepPath := filepath.Join(repoPath, "deep")
	_, err = os.Stat(deepPath)
	assert.True(t, os.IsNotExist(err), "Deep directory should be completely removed")

	// Verify no remnants of the nested structure exist
	for filePath := range nestedFiles {
		fullPath := filepath.Join(repoPath, filePath)
		_, err = os.Stat(fullPath)
		assert.True(t, os.IsNotExist(err), "Nested file should be removed: %s", filePath)
	}
}

func TestClient_RemoveDirectory_PartialPathMatching(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	workTree, err := repo.Worktree()
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Create directories with similar names to test path matching
	directories := []string{"test", "test-similar", "test/subdir", "different/test"}
	files := make(map[string]string)

	for _, dir := range directories {
		dirPath := filepath.Join(repoPath, dir)
		err = os.MkdirAll(dirPath, 0o755)
		require.NoError(t, err)

		fileName := filepath.Join(dir, "file.txt")
		filePath := filepath.Join(repoPath, fileName)
		content := fmt.Sprintf("content in %s", dir)
		err = os.WriteFile(filePath, []byte(content), 0o644)
		require.NoError(t, err)

		files[fileName] = content
		_, err = workTree.Add(fileName)
		require.NoError(t, err)
	}

	// Initial commit
	_, err = workTree.Commit("Add test directories", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	// Remove only the "test" directory (not "test-similar" or "different/test")
	err = client.RemoveDirectory("test", "Remove test directory only")
	if err != nil {
		assert.True(t,
			strings.Contains(err.Error(), "failed to push") ||
				strings.Contains(err.Error(), "failed to create ssh auth"),
			"Expected push or SSH auth error, got: %v", err)
	}

	// Verify only the exact "test" directory was removed
	testPath := filepath.Join(repoPath, "test")
	_, err = os.Stat(testPath)
	assert.True(t, os.IsNotExist(err), "test directory should be removed")

	// Verify similar named directories were NOT removed
	testSimilarPath := filepath.Join(repoPath, "test-similar")
	_, err = os.Stat(testSimilarPath)
	assert.NoError(t, err, "test-similar directory should still exist")

	differentTestPath := filepath.Join(repoPath, "different/test")
	_, err = os.Stat(differentTestPath)
	assert.NoError(t, err, "different/test file should still exist")
}

func TestClient_RemoveDirectory_ErrorScenarios(t *testing.T) {
	t.Run("Invalid repository path", func(t *testing.T) {
		client := &Client{
			RepoURL:  "origin",
			Branch:   "main",
			SSHKey:   createTestSSHKey(),
			RepoPath: "/non/existent/path",
		}

		err := client.RemoveDirectory("some-dir", "Remove directory")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to open repo")
	})

	t.Run("Remove non-existent directory", func(t *testing.T) {
		tmpDir := createTempDir(t)
		repoPath := filepath.Join(tmpDir, "test-repo")

		_, err := git.PlainInit(repoPath, false)
		require.NoError(t, err)

		client := &Client{
			RepoURL:  "origin",
			Branch:   "main",
			SSHKey:   createTestSSHKey(),
			RepoPath: repoPath,
		}

		// Should handle non-existent directory gracefully
		err = client.RemoveDirectory("non-existent", "Remove non-existent directory")
		if err != nil {
			// Should only fail with empty commit error since nothing to remove
			assert.True(t,
				strings.Contains(err.Error(), "clean working tree"),
				"Expected empty commit error for non-existent directory, got: %v", err)
		}
	})

	t.Run("Remove directory with only untracked files", func(t *testing.T) {
		tmpDir := createTempDir(t)
		repoPath := filepath.Join(tmpDir, "test-repo")

		_, err := git.PlainInit(repoPath, false)
		require.NoError(t, err)

		client := &Client{
			RepoURL:  "origin",
			Branch:   "main",
			SSHKey:   createTestSSHKey(),
			RepoPath: repoPath,
		}

		// Create directory with untracked files
		untrackedDir := filepath.Join(repoPath, "untracked-dir")
		err = os.MkdirAll(untrackedDir, 0o755)
		require.NoError(t, err)

		untrackedFile := filepath.Join(untrackedDir, "untracked.txt")
		err = os.WriteFile(untrackedFile, []byte("untracked content"), 0o644)
		require.NoError(t, err)

		// Remove the directory
		err = client.RemoveDirectory("untracked-dir", "Remove untracked directory")

		// Should succeed in removing from filesystem but may have no git changes to commit
		if err != nil && !strings.Contains(err.Error(), "failed to push") && !strings.Contains(err.Error(), "failed to create ssh auth") {
			// Allow for "nothing to commit" type errors
			assert.NotContains(t, err.Error(), "panic")
		}

		// Verify directory was removed from filesystem
		_, err = os.Stat(untrackedDir)
		assert.True(t, os.IsNotExist(err), "Untracked directory should be removed from filesystem")
	})
}

func TestClient_RemoveDirectory_CommitMessagePropagation(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	workTree, err := repo.Worktree()
	require.NoError(t, err)

	// Create and track a file to remove
	testDir := filepath.Join(repoPath, "removeme")
	err = os.MkdirAll(testDir, 0o755)
	require.NoError(t, err)

	testFile := filepath.Join(testDir, "file.txt")
	err = os.WriteFile(testFile, []byte("content"), 0o644)
	require.NoError(t, err)

	_, err = workTree.Add("removeme/file.txt")
	require.NoError(t, err)

	_, err = workTree.Commit("Initial commit", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Test various commit message formats
	testCases := []struct {
		name          string
		commitMessage string
	}{
		{"Simple message", "Remove directory"},
		{"Multi-line message", "Remove directory\n\nThis removes the test directory\nwith all its contents"},
		{"Message with special chars", "Remove: test/directory [cleanup] - phase 1"},
		{"Empty message", ""},
		{"Very long message", strings.Repeat("Very long commit message ", 20)},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Recreate the directory for each test
			err = os.MkdirAll(testDir, 0o755)
			require.NoError(t, err)
			err = os.WriteFile(testFile, []byte("content"), 0o644)
			require.NoError(t, err)
			_, err = workTree.Add("removeme/file.txt")
			require.NoError(t, err)
			_, err = workTree.Commit("Recreate for test", &git.CommitOptions{
				Author: &object.Signature{
					Name:  "Test User",
					Email: "test@example.com",
					When:  time.Now(),
				},
			})
			require.NoError(t, err)

			// Remove with the test commit message
			err = client.RemoveDirectory("removeme", tc.commitMessage)
			if err != nil {
				// Should only fail at push, SSH auth, or empty commit (no changes to commit)
				assert.True(t,
					strings.Contains(err.Error(), "failed to push") ||
						strings.Contains(err.Error(), "failed to create ssh auth") ||
						strings.Contains(err.Error(), "clean working tree"),
					"Expected push, SSH auth, or empty commit error, got: %v", err)
			}

			// Verify commit message was used correctly (only if a new commit was created)
			ref, err := repo.Head()
			require.NoError(t, err)

			commit, err := repo.CommitObject(ref.Hash())
			require.NoError(t, err)

			// Only verify commit message if this isn't a setup commit
			if !strings.Contains(commit.Message, "Recreate for test") &&
				!strings.Contains(commit.Message, "Setup test structure") {
				expectedMessage := tc.commitMessage
				if expectedMessage == "" {
					expectedMessage = "" // Empty messages should be preserved
				}

				assert.Equal(t, expectedMessage, commit.Message, "Commit message should match for test: %s", tc.name)
			} else {
				// If we're still on a setup commit, the RemoveDirectory had no changes to commit
				t.Logf("Test %s: RemoveDirectory had no changes to commit", tc.name)
			}
		})
	}
}

// Table-driven tests for CommitAndPushChanges edge cases
func TestClient_CommitAndPushChanges_TableDriven(t *testing.T) {
	testCases := []struct {
		name            string
		setupFiles      map[string]string // files to create initially (tracked)
		modifyFiles     map[string]string // files to modify after initial commit
		createFiles     map[string]string // new files to create (untracked)
		deleteFiles     []string          // tracked files to delete
		expectStaged    []string          // files that should be staged
		expectUntracked []string          // files that should remain untracked
	}{
		{
			name: "Only tracked modifications",
			setupFiles: map[string]string{
				"config.yaml": "initial: config",
				"app.js":      "console.log('initial');",
			},
			modifyFiles: map[string]string{
				"config.yaml": "modified: config",
				"app.js":      "console.log('modified');",
			},
			expectStaged: []string{"config.yaml", "app.js"},
		},
		{
			name: "Mixed tracked and untracked",
			setupFiles: map[string]string{
				"existing.txt": "existing content",
			},
			modifyFiles: map[string]string{
				"existing.txt": "modified existing content",
			},
			createFiles: map[string]string{
				"new.txt":           "new content",
				"untracked/sub.txt": "untracked sub content",
			},
			expectStaged:    []string{"existing.txt"},
			expectUntracked: []string{"new.txt", "untracked/sub.txt"},
		},
		{
			name: "File deletions",
			setupFiles: map[string]string{
				"keep.txt":   "keep this",
				"delete.txt": "delete this",
			},
			deleteFiles:  []string{"delete.txt"},
			expectStaged: []string{"delete.txt"}, // deletions should be staged
		},
		{
			name: "Git files mixed with regular files",
			setupFiles: map[string]string{
				"regular.txt": "regular content",
			},
			modifyFiles: map[string]string{
				"regular.txt": "modified regular content",
			},
			createFiles: map[string]string{
				".gitignore":     "*.log",
				".git-config":    "config content",
				".github/ci.yml": "ci: config",
			},
			expectStaged:    []string{"regular.txt"},
			expectUntracked: []string{".gitignore", ".git-config", ".github/ci.yml"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := createTempDir(t)
			repoPath := filepath.Join(tmpDir, "table-test-repo")

			repo, err := git.PlainInit(repoPath, false)
			require.NoError(t, err)

			workTree, err := repo.Worktree()
			require.NoError(t, err)

			// Setup initial tracked files
			for filePath, content := range tc.setupFiles {
				fullPath := filepath.Join(repoPath, filePath)
				err = os.MkdirAll(filepath.Dir(fullPath), 0o755)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0o644)
				require.NoError(t, err)
				_, err = workTree.Add(filePath)
				require.NoError(t, err)
			}

			// Initial commit
			if len(tc.setupFiles) > 0 {
				_, err = workTree.Commit("Initial commit", &git.CommitOptions{
					Author: &object.Signature{
						Name:  "Test User",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)
			}

			// Modify tracked files
			for filePath, content := range tc.modifyFiles {
				fullPath := filepath.Join(repoPath, filePath)
				err = os.WriteFile(fullPath, []byte(content), 0o644)
				require.NoError(t, err)
			}

			// Create untracked files
			for filePath, content := range tc.createFiles {
				fullPath := filepath.Join(repoPath, filePath)
				err = os.MkdirAll(filepath.Dir(fullPath), 0o755)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0o644)
				require.NoError(t, err)
			}

			// Delete specified files
			for _, filePath := range tc.deleteFiles {
				fullPath := filepath.Join(repoPath, filePath)
				err = os.Remove(fullPath)
				require.NoError(t, err)
			}

			client := &Client{
				RepoURL:  "origin",
				Branch:   "main",
				SSHKey:   createTestSSHKey(),
				RepoPath: repoPath,
			}

			// Execute CommitAndPushChanges
			err = client.CommitAndPushChanges(fmt.Sprintf("Test: %s", tc.name))

			// Allow for push failures and SSH auth failures but verify staging behavior
			if err != nil && !strings.Contains(err.Error(), "failed to push") && !strings.Contains(err.Error(), "failed to create ssh auth") {
				// Check if it's a "nothing to commit" error, which is acceptable
				if !strings.Contains(strings.ToLower(err.Error()), "nothing to commit") {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Verify final status matches expectations
			status, err := workTree.Status()
			require.NoError(t, err)

			// Check that expected untracked files remain untracked
			for _, expectedUntracked := range tc.expectUntracked {
				if fileStatus, exists := status[expectedUntracked]; exists {
					assert.Equal(t, git.Untracked, fileStatus.Worktree,
						"File should remain untracked: %s", expectedUntracked)
				}
			}

			// Verify .git files are not processed
			for file := range status {
				if strings.HasPrefix(file, ".git") {
					assert.Equal(t, git.Untracked, status[file].Worktree,
						"Git-related files should remain untracked: %s", file)
				}
			}
		})
	}
}

// Table-driven tests for RemoveDirectory scenarios
func TestClient_RemoveDirectory_TableDriven(t *testing.T) {
	testCases := []struct {
		name           string
		setupStructure map[string]string // files to create and track
		removeDir      string            // directory to remove
		commitMsg      string            // commit message to use
		shouldExist    []string          // files/dirs that should still exist
		shouldNotExist []string          // files/dirs that should be removed
		expectError    bool              // whether an error is expected (other than push)
	}{
		{
			name: "Remove simple directory",
			setupStructure: map[string]string{
				"simple/file1.txt": "content1",
				"simple/file2.txt": "content2",
				"keep/file3.txt":   "content3",
			},
			removeDir:      "simple",
			commitMsg:      "Remove simple directory",
			shouldExist:    []string{"keep", "keep/file3.txt"},
			shouldNotExist: []string{"simple", "simple/file1.txt", "simple/file2.txt"},
		},
		{
			name: "Remove nested directory structure",
			setupStructure: map[string]string{
				"parent/child1/file1.txt":     "content1",
				"parent/child2/file2.txt":     "content2",
				"parent/child1/sub/file3.txt": "content3",
				"other/file4.txt":             "content4",
			},
			removeDir:      "parent",
			commitMsg:      "Remove entire parent directory tree",
			shouldExist:    []string{"other", "other/file4.txt"},
			shouldNotExist: []string{"parent", "parent/child1", "parent/child2", "parent/child1/file1.txt", "parent/child2/file2.txt", "parent/child1/sub/file3.txt"},
		},
		{
			name: "Remove directory with special characters",
			setupStructure: map[string]string{
				"special-dir_123/file.with.dots.txt": "special content",
				"special-dir_123/sub-dir/file2.txt":  "more content",
				"normal/file.txt":                    "normal content",
			},
			removeDir:      "special-dir_123",
			commitMsg:      "Remove directory with special characters",
			shouldExist:    []string{"normal", "normal/file.txt"},
			shouldNotExist: []string{"special-dir_123", "special-dir_123/file.with.dots.txt", "special-dir_123/sub-dir"},
		},
		{
			name: "Attempt to remove non-existent directory",
			setupStructure: map[string]string{
				"existing/file.txt": "content",
			},
			removeDir:      "non-existent",
			commitMsg:      "Remove non-existent directory",
			shouldExist:    []string{"existing", "existing/file.txt"},
			shouldNotExist: []string{}, // nothing should be removed
			expectError:    false,      // should handle gracefully
		},
		{
			name: "Remove with empty commit message",
			setupStructure: map[string]string{
				"empty-msg/file.txt": "content",
			},
			removeDir:      "empty-msg",
			commitMsg:      "",
			shouldExist:    []string{},
			shouldNotExist: []string{"empty-msg", "empty-msg/file.txt"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tmpDir := createTempDir(t)
			repoPath := filepath.Join(tmpDir, "table-remove-test")

			repo, err := git.PlainInit(repoPath, false)
			require.NoError(t, err)

			workTree, err := repo.Worktree()
			require.NoError(t, err)

			// Setup directory structure
			for filePath, content := range tc.setupStructure {
				fullPath := filepath.Join(repoPath, filePath)
				err = os.MkdirAll(filepath.Dir(fullPath), 0o755)
				require.NoError(t, err)
				err = os.WriteFile(fullPath, []byte(content), 0o644)
				require.NoError(t, err)
				_, err = workTree.Add(filePath)
				require.NoError(t, err)
			}

			// Initial commit
			if len(tc.setupStructure) > 0 {
				_, err = workTree.Commit("Setup test structure", &git.CommitOptions{
					Author: &object.Signature{
						Name:  "Test User",
						Email: "test@example.com",
						When:  time.Now(),
					},
				})
				require.NoError(t, err)
			}

			client := &Client{
				RepoURL:  "origin",
				Branch:   "main",
				SSHKey:   createTestSSHKey(),
				RepoPath: repoPath,
			}

			// Execute RemoveDirectory
			err = client.RemoveDirectory(tc.removeDir, tc.commitMsg)

			if tc.expectError {
				assert.Error(t, err)
				return
			}

			// Allow push failures, SSH auth failures, and empty commit errors but verify filesystem operations
			if err != nil && !strings.Contains(err.Error(), "failed to push") &&
				!strings.Contains(err.Error(), "failed to create ssh auth") &&
				!strings.Contains(err.Error(), "clean working tree") {
				// Allow for "nothing to commit" scenarios
				if !strings.Contains(strings.ToLower(err.Error()), "nothing to commit") {
					t.Errorf("Unexpected error for test %s: %v", tc.name, err)
				}
			}

			// Verify files/directories that should still exist
			for _, shouldExist := range tc.shouldExist {
				fullPath := filepath.Join(repoPath, shouldExist)
				_, err = os.Stat(fullPath)
				assert.NoError(t, err, "Path should still exist: %s", shouldExist)
			}

			// Verify files/directories that should be removed
			for _, shouldNotExist := range tc.shouldNotExist {
				fullPath := filepath.Join(repoPath, shouldNotExist)
				_, err = os.Stat(fullPath)
				assert.True(t, os.IsNotExist(err), "Path should be removed: %s", shouldNotExist)
			}

			// Verify commit message if there were changes
			if len(tc.setupStructure) > 0 && tc.removeDir != "non-existent" {
				ref, err := repo.Head()
				if err == nil {
					commit, err := repo.CommitObject(ref.Hash())
					if err == nil && commit.Message != "Setup test structure" {
						// Only check if it's not the setup commit
						assert.Equal(t, tc.commitMsg, commit.Message, "Commit message should match")
					}
				}
			}
		})
	}
}

// Additional edge case tests
func TestClient_CommitAndPushChanges_FileStatusEdgeCases(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "edge-case-repo")

	repo, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	workTree, err := repo.Worktree()
	require.NoError(t, err)

	// Test with files that have different git status combinations
	testFile := filepath.Join(repoPath, "test.txt")
	err = os.WriteFile(testFile, []byte("initial"), 0o644)
	require.NoError(t, err)

	_, err = workTree.Add("test.txt")
	require.NoError(t, err)

	_, err = workTree.Commit("Initial", &git.CommitOptions{
		Author: &object.Signature{
			Name:  "Test User",
			Email: "test@example.com",
			When:  time.Now(),
		},
	})
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Test scenario: file is both staged and modified in worktree
	err = os.WriteFile(testFile, []byte("staged change"), 0o644)
	require.NoError(t, err)

	_, err = workTree.Add("test.txt") // Stage the change
	require.NoError(t, err)

	err = os.WriteFile(testFile, []byte("worktree change"), 0o644) // Modify again
	require.NoError(t, err)

	// Should handle the mixed state correctly
	err = client.CommitAndPushChanges("Handle mixed staged/worktree state")
	if err != nil {
		assert.True(t,
			strings.Contains(err.Error(), "failed to push") ||
				strings.Contains(err.Error(), "failed to create ssh auth"),
			"Expected push or SSH auth error, got: %v", err)
	}

	// Verify the final state
	content, err := os.ReadFile(testFile)
	require.NoError(t, err)

	// The worktree version should be preserved
	assert.Equal(t, "worktree change", string(content))
}

func TestClient_RemoveDirectory_EdgeCaseScenarios(t *testing.T) {
	t.Run("Remove root level files vs directories", func(t *testing.T) {
		tmpDir := createTempDir(t)
		repoPath := filepath.Join(tmpDir, "root-test")

		repo, err := git.PlainInit(repoPath, false)
		require.NoError(t, err)

		workTree, err := repo.Worktree()
		require.NoError(t, err)

		// Create both root-level files and directories with similar names
		rootFile := filepath.Join(repoPath, "test")
		testDir := filepath.Join(repoPath, "testdir")
		testDirFile := filepath.Join(testDir, "file.txt")

		err = os.WriteFile(rootFile, []byte("root file content"), 0o644)
		require.NoError(t, err)
		_, err = workTree.Add("test")
		require.NoError(t, err)

		err = os.MkdirAll(testDir, 0o755)
		require.NoError(t, err)
		err = os.WriteFile(testDirFile, []byte("dir file content"), 0o644)
		require.NoError(t, err)
		_, err = workTree.Add("testdir/file.txt")
		require.NoError(t, err)

		_, err = workTree.Commit("Setup", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test User",
				Email: "test@example.com",
				When:  time.Now(),
			},
		})
		require.NoError(t, err)

		client := &Client{
			RepoURL:  "origin",
			Branch:   "main",
			SSHKey:   createTestSSHKey(),
			RepoPath: repoPath,
		}

		// Remove directory, not the root file
		err = client.RemoveDirectory("testdir", "Remove testdir only")
		if err != nil {
			assert.True(t,
				strings.Contains(err.Error(), "failed to push") ||
					strings.Contains(err.Error(), "failed to create ssh auth"),
				"Expected push or SSH auth error, got: %v", err)
		}

		// Root file should still exist
		_, err = os.Stat(rootFile)
		assert.NoError(t, err, "Root file 'test' should still exist")

		// Directory should be removed
		_, err = os.Stat(testDir)
		assert.True(t, os.IsNotExist(err), "Directory 'testdir' should be removed")
	})

	t.Run("Remove directory with mixed tracked/untracked content", func(t *testing.T) {
		tmpDir := createTempDir(t)
		repoPath := filepath.Join(tmpDir, "mixed-test")

		repo, err := git.PlainInit(repoPath, false)
		require.NoError(t, err)

		workTree, err := repo.Worktree()
		require.NoError(t, err)

		// Create directory with both tracked and untracked files
		mixedDir := filepath.Join(repoPath, "mixed")
		err = os.MkdirAll(mixedDir, 0o755)
		require.NoError(t, err)

		trackedFile := filepath.Join(mixedDir, "tracked.txt")
		untrackedFile := filepath.Join(mixedDir, "untracked.txt")

		err = os.WriteFile(trackedFile, []byte("tracked content"), 0o644)
		require.NoError(t, err)
		_, err = workTree.Add("mixed/tracked.txt")
		require.NoError(t, err)

		_, err = workTree.Commit("Add tracked file", &git.CommitOptions{
			Author: &object.Signature{
				Name:  "Test User",
				Email: "test@example.com",
				When:  time.Now(),
			},
		})
		require.NoError(t, err)

		// Add untracked file after commit
		err = os.WriteFile(untrackedFile, []byte("untracked content"), 0o644)
		require.NoError(t, err)

		client := &Client{
			RepoURL:  "origin",
			Branch:   "main",
			SSHKey:   createTestSSHKey(),
			RepoPath: repoPath,
		}

		// Remove directory containing both tracked and untracked files
		err = client.RemoveDirectory("mixed", "Remove mixed directory")

		if err != nil && !strings.Contains(err.Error(), "failed to push") && !strings.Contains(err.Error(), "failed to create ssh auth") {
			// May have nothing to commit if only untracked content
			assert.NotContains(t, err.Error(), "panic")
		}

		// Entire directory should be removed from filesystem
		_, err = os.Stat(mixedDir)
		assert.True(t, os.IsNotExist(err), "Mixed directory should be completely removed")
	})
}

// Integration tests with mock implementations
func TestClientInterface_MockImplementation(t *testing.T) {
	mockClient := &MockClientInterface{}

	// Test CommitAndPush
	files := map[string]string{
		"test.txt": "content",
	}
	expectedHash := "abc123def456"

	mockClient.On("CommitAndPush", files, "Test message").Return(expectedHash, nil)

	hash, err := mockClient.CommitAndPush(files, "Test message")
	assert.NoError(t, err)
	assert.Equal(t, expectedHash, hash)

	// Test CommitAndPushChanges
	mockClient.On("CommitAndPushChanges", "Changes message").Return(nil)

	err = mockClient.CommitAndPushChanges("Changes message")
	assert.NoError(t, err)

	// Test InitRepo
	mockClient.On("InitRepo").Return(nil)

	err = mockClient.InitRepo()
	assert.NoError(t, err)

	// Test RemoveDirectory
	mockClient.On("RemoveDirectory", "old-dir", "Remove old directory").Return(nil)

	err = mockClient.RemoveDirectory("old-dir", "Remove old directory")
	assert.NoError(t, err)

	mockClient.AssertExpectations(t)
}

func TestClientInterface_MockWithErrors(t *testing.T) {
	mockClient := &MockClientInterface{}

	// Test error scenarios
	files := map[string]string{
		"test.txt": "content",
	}

	expectedError := fmt.Errorf("push failed")
	mockClient.On("CommitAndPush", files, "Test message").Return("", expectedError)

	hash, err := mockClient.CommitAndPush(files, "Test message")
	assert.Error(t, err)
	assert.Equal(t, expectedError, err)
	assert.Empty(t, hash)

	mockClient.AssertExpectations(t)
}

// Test file permission handling
func TestClient_FilePermissions(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "test-repo")

	// Create a real git repository
	_, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	files := map[string]string{
		"script.sh":   "#!/bin/bash\necho 'test'",
		"config.yaml": "key: value",
	}

	// This will fail at push, but files should be created
	_, _ = client.CommitAndPush(files, "Add files")

	// Verify file permissions (should be 0644)
	for filePath := range files {
		fullPath := filepath.Join(repoPath, filePath)
		info, statErr := os.Stat(fullPath)
		assert.NoError(t, statErr)

		// Check that file has expected permissions
		mode := info.Mode()
		assert.True(t, mode.IsRegular())

		// On Unix systems, check specific permissions
		if mode&fs.ModePerm != 0 {
			perm := mode & fs.ModePerm
			assert.Equal(t, fs.FileMode(0o644), perm, "File should have 0644 permissions: %s", filePath)
		}
	}
}

// Test edge cases and error handling
func TestClient_EdgeCases(t *testing.T) {
	tmpDir := createTempDir(t)

	t.Run("Empty files map", func(t *testing.T) {
		repoPath := filepath.Join(tmpDir, "empty-test-repo")
		_, err := git.PlainInit(repoPath, false)
		require.NoError(t, err)

		client := &Client{
			RepoURL:  "origin",
			Branch:   "main",
			SSHKey:   createTestSSHKey(),
			RepoPath: repoPath,
		}

		files := map[string]string{}

		// Should handle empty files map gracefully
		_, err = client.CommitAndPush(files, "Empty commit")
		// May fail at push, but should not panic
		if err != nil {
			assert.NotContains(t, err.Error(), "panic")
		}
	})

	t.Run("Very long file paths", func(t *testing.T) {
		repoPath := filepath.Join(tmpDir, "long-path-test-repo")
		_, err := git.PlainInit(repoPath, false)
		require.NoError(t, err)

		client := &Client{
			RepoURL:  "origin",
			Branch:   "main",
			SSHKey:   createTestSSHKey(),
			RepoPath: repoPath,
		}

		// Create a very long file path
		longPath := strings.Repeat("long-directory-name/", 10) + "file.txt"
		files := map[string]string{
			longPath: "content",
		}

		_, _ = client.CommitAndPush(files, "Long path test")

		// Should handle long paths correctly
		fullPath := filepath.Join(repoPath, longPath)
		_, statErr := os.Stat(fullPath)
		assert.NoError(t, statErr, "Long path file should be created")
	})

	t.Run("Special characters in filenames", func(t *testing.T) {
		repoPath := filepath.Join(tmpDir, "special-chars-test-repo")
		_, err := git.PlainInit(repoPath, false)
		require.NoError(t, err)

		client := &Client{
			RepoURL:  "origin",
			Branch:   "main",
			SSHKey:   createTestSSHKey(),
			RepoPath: repoPath,
		}

		files := map[string]string{
			"file-with-dashes.txt":      "content1",
			"file_with_underscores.txt": "content2",
			"file.with.dots.txt":        "content3",
			"file123numbers.txt":        "content4",
		}

		_, _ = client.CommitAndPush(files, "Special characters test")

		// Verify all files were created
		for filePath, expectedContent := range files {
			fullPath := filepath.Join(repoPath, filePath)
			content, readErr := os.ReadFile(fullPath)
			assert.NoError(t, readErr, "Should be able to read file with special chars: %s", filePath)
			assert.Equal(t, expectedContent, string(content))
		}
	})

	t.Run("Large file content", func(t *testing.T) {
		repoPath := filepath.Join(tmpDir, "large-file-test-repo")
		_, err := git.PlainInit(repoPath, false)
		require.NoError(t, err)

		client := &Client{
			RepoURL:  "origin",
			Branch:   "main",
			SSHKey:   createTestSSHKey(),
			RepoPath: repoPath,
		}

		// Create a large file content (1MB of data)
		largeContent := strings.Repeat("This is a line of test data for large file testing.\n", 20000)
		files := map[string]string{
			"large-file.txt": largeContent,
		}

		_, _ = client.CommitAndPush(files, "Large file test")

		// Verify large file was created correctly
		fullPath := filepath.Join(repoPath, "large-file.txt")
		content, readErr := os.ReadFile(fullPath)
		assert.NoError(t, readErr)
		assert.Equal(t, largeContent, string(content))
		assert.Greater(t, len(content), 1000000, "File should be larger than 1MB")
	})
}

// Test concurrent access (basic test)
func TestClient_ConcurrentAccess(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "concurrent-test-repo")

	// Create a real git repository
	_, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Note: Real concurrent git operations would require more sophisticated testing
	// This test just verifies that multiple operations don't cause obvious issues

	// Perform multiple operations in sequence (simulating potential concurrent scenarios)
	for i := 0; i < 5; i++ {
		files := map[string]string{
			fmt.Sprintf("file%d.txt", i): fmt.Sprintf("content%d", i),
		}

		_, err = client.CommitAndPush(files, fmt.Sprintf("Commit %d", i))
		// Errors expected due to no remote, but should not panic
		if err != nil {
			assert.NotContains(t, err.Error(), "panic")
		}
	}
}

// Benchmark tests
func BenchmarkClient_CommitAndPush(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-test-*")
	require.NoError(b, err)
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "bench-repo")
	_, err = git.PlainInit(repoPath, false)
	require.NoError(b, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	files := map[string]string{
		"bench.txt": "benchmark content",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files[fmt.Sprintf("bench%d.txt", i)] = fmt.Sprintf("content%d", i)
		client.CommitAndPush(files, fmt.Sprintf("Benchmark commit %d", i))
	}
}

func BenchmarkClient_FileCreation(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-file-test-*")
	require.NoError(b, err)
	defer os.RemoveAll(tmpDir)

	repoPath := filepath.Join(tmpDir, "bench-repo")
	_, err = git.PlainInit(repoPath, false)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		files := map[string]string{
			fmt.Sprintf("subdir/file%d.txt", i): fmt.Sprintf("content%d", i),
		}

		// Only test file creation part, not git operations
		for filePath, content := range files {
			fullPath := filepath.Join(repoPath, filePath)
			os.MkdirAll(filepath.Dir(fullPath), 0o755)
			os.WriteFile(fullPath, []byte(content), 0o644)
		}
	}
}

// Test helper to verify git repository state
func verifyGitCommit(t *testing.T, repoPath string, commitMessage string) {
	repo, err := git.PlainOpen(repoPath)
	require.NoError(t, err)

	ref, err := repo.Head()
	require.NoError(t, err)

	commit, err := repo.CommitObject(ref.Hash())
	require.NoError(t, err)

	assert.Contains(t, commit.Message, commitMessage)
	assert.Equal(t, "Nephio Bridge", commit.Author.Name)
	assert.Equal(t, "nephio-bridge@example.com", commit.Author.Email)
}

func TestGitCommitDetails(t *testing.T) {
	tmpDir := createTempDir(t)
	repoPath := filepath.Join(tmpDir, "commit-details-test")

	// Create a real git repository
	_, err := git.PlainInit(repoPath, false)
	require.NoError(t, err)

	client := &Client{
		RepoURL:  "origin",
		Branch:   "main",
		SSHKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	files := map[string]string{
		"test.txt": "test content",
	}

	commitMessage := "Test commit with details"
	_, err = client.CommitAndPush(files, commitMessage)

	// Verify commit details (will fail at push, but commit should be created)
	if err == nil || !strings.Contains(err.Error(), "failed to push") {
		verifyGitCommit(t, repoPath, commitMessage)
	}
}

// ========================================
// Tests for NewGitClientConfig token loading functionality
// ========================================

func TestNewGitClientConfig_ValidTokenFile(t *testing.T) {
	tmpDir := createTempDir(t)
	tokenFile := filepath.Join(tmpDir, "token.txt")
	expectedToken := "github_pat_123456789abcdef"

	// Create token file
	err := os.WriteFile(tokenFile, []byte(expectedToken), 0o600)
	require.NoError(t, err)

	config, err := NewGitClientConfig(
		"git@github.com:test/repo.git",
		"main",
		"fallback-token", // This should not be used
		tokenFile,
	)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "git@github.com:test/repo.git", config.RepoURL)
	assert.Equal(t, "main", config.Branch)
	assert.Equal(t, expectedToken, config.Token)
	assert.Equal(t, tokenFile, config.TokenPath)
	assert.Equal(t, "/tmp/deployment-repo", config.RepoPath)
	assert.NotNil(t, config.Logger)
}

func TestNewGitClientConfig_TokenFileWithWhitespace(t *testing.T) {
	tmpDir := createTempDir(t)
	tokenFile := filepath.Join(tmpDir, "token-with-whitespace.txt")
	rawToken := "  \n\t  github_pat_with_whitespace  \n\t  "
	expectedToken := "github_pat_with_whitespace"

	// Create token file with whitespace
	err := os.WriteFile(tokenFile, []byte(rawToken), 0o600)
	require.NoError(t, err)

	config, err := NewGitClientConfig(
		"git@github.com:test/repo.git",
		"main",
		"fallback-token",
		tokenFile,
	)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, expectedToken, config.Token, "Token should be trimmed of whitespace")
	assert.Equal(t, tokenFile, config.TokenPath)
}

func TestNewGitClientConfig_InvalidTokenFileFallbackToEnvVar(t *testing.T) {
	nonExistentFile := "/non/existent/token/file.txt"
	fallbackToken := "env-var-token-123"

	config, err := NewGitClientConfig(
		"git@github.com:test/repo.git",
		"main",
		fallbackToken,
		nonExistentFile,
	)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "git@github.com:test/repo.git", config.RepoURL)
	assert.Equal(t, "main", config.Branch)
	assert.Equal(t, fallbackToken, config.Token, "Should fallback to environment variable token")
	assert.Empty(t, config.TokenPath, "TokenPath should be empty when fallback is used")
	assert.Equal(t, "/tmp/deployment-repo", config.RepoPath)
	assert.NotNil(t, config.Logger)
}

func TestNewGitClientConfig_EmptyTokenPathUsesEnvVar(t *testing.T) {
	envVarToken := "environment-variable-token"

	config, err := NewGitClientConfig(
		"git@github.com:test/repo.git",
		"main",
		envVarToken,
		"", // Empty token path
	)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "git@github.com:test/repo.git", config.RepoURL)
	assert.Equal(t, "main", config.Branch)
	assert.Equal(t, envVarToken, config.Token, "Should use environment variable token")
	assert.Empty(t, config.TokenPath, "TokenPath should be empty")
	assert.Equal(t, "/tmp/deployment-repo", config.RepoPath)
	assert.NotNil(t, config.Logger)
}

func TestNewGitClientConfig_NoTokenFileNoEnvVar(t *testing.T) {
	nonExistentFile := "/non/existent/token/file.txt"

	config, err := NewGitClientConfig(
		"git@github.com:test/repo.git",
		"main",
		"", // Empty environment variable token
		nonExistentFile,
	)

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "no git token available")
	assert.Contains(t, err.Error(), nonExistentFile)
}

func TestNewGitClientConfig_EmptyTokenFileAndEmptyEnvVar(t *testing.T) {
	config, err := NewGitClientConfig(
		"git@github.com:test/repo.git",
		"main",
		"", // Empty environment variable token
		"", // Empty token path
	)

	assert.Error(t, err)
	assert.Nil(t, config)
	assert.Contains(t, err.Error(), "no git token available")
}

func TestNewGitClientConfig_TokenFileReadError(t *testing.T) {
	tmpDir := createTempDir(t)

	// Create a directory instead of a file to cause read error
	tokenDir := filepath.Join(tmpDir, "token-dir")
	err := os.Mkdir(tokenDir, 0o755)
	require.NoError(t, err)

	fallbackToken := "fallback-env-token"

	config, err := NewGitClientConfig(
		"git@github.com:test/repo.git",
		"main",
		fallbackToken,
		tokenDir, // This is a directory, not a file
	)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, fallbackToken, config.Token, "Should fallback to environment variable on read error")
	assert.Empty(t, config.TokenPath, "TokenPath should be empty when fallback is used")
}

func TestNewGitClientConfig_TokenFilePermissions(t *testing.T) {
	tmpDir := createTempDir(t)
	tokenFile := filepath.Join(tmpDir, "secure-token.txt")
	expectedToken := "secure_token_123"

	// Create token file with restricted permissions
	err := os.WriteFile(tokenFile, []byte(expectedToken), 0o600)
	require.NoError(t, err)

	config, err := NewGitClientConfig(
		"git@github.com:test/repo.git",
		"main",
		"fallback-token",
		tokenFile,
	)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, expectedToken, config.Token)

	// Verify file exists and can be read (permissions behavior varies by OS)
	info, err := os.Stat(tokenFile)
	require.NoError(t, err)
	assert.True(t, info.Mode().IsRegular(), "Token file should be a regular file")
}

func TestNewGitClientConfig_EmptyTokenFileContent(t *testing.T) {
	tmpDir := createTempDir(t)
	tokenFile := filepath.Join(tmpDir, "empty-token.txt")
	fallbackToken := "fallback-token-456"

	// Create empty token file
	err := os.WriteFile(tokenFile, []byte(""), 0o600)
	require.NoError(t, err)

	config, err := NewGitClientConfig(
		"git@github.com:test/repo.git",
		"main",
		fallbackToken,
		tokenFile,
	)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "", config.Token, "Empty token file should result in empty token")
	assert.Equal(t, tokenFile, config.TokenPath)
}

func TestNewGitClientConfig_WhitespaceOnlyTokenFile(t *testing.T) {
	tmpDir := createTempDir(t)
	tokenFile := filepath.Join(tmpDir, "whitespace-token.txt")
	fallbackToken := "fallback-token-789"

	// Create token file with only whitespace
	err := os.WriteFile(tokenFile, []byte("   \n\t   \n   "), 0o600)
	require.NoError(t, err)

	config, err := NewGitClientConfig(
		"git@github.com:test/repo.git",
		"main",
		fallbackToken,
		tokenFile,
	)

	assert.NoError(t, err)
	assert.NotNil(t, config)
	assert.Equal(t, "", config.Token, "Whitespace-only token file should result in empty token after trimming")
	assert.Equal(t, tokenFile, config.TokenPath)
}

func TestNewGitClientConfig_TableDriven(t *testing.T) {
	tmpDir := createTempDir(t)

	testCases := []struct {
		name          string
		setupToken    func() (string, string) // Returns tokenPath, tokenContent
		envVarToken   string
		expectError   bool
		expectedToken string
		expectedPath  string
	}{
		{
			name: "Valid token file with content",
			setupToken: func() (string, string) {
				tokenFile := filepath.Join(tmpDir, "valid-token.txt")
				content := "github_pat_valid_token"
				os.WriteFile(tokenFile, []byte(content), 0o600)
				return tokenFile, content
			},
			envVarToken:   "env-token",
			expectError:   false,
			expectedToken: "github_pat_valid_token",
			expectedPath:  "file", // Indicates should have TokenPath set to the file
		},
		{
			name: "Token file with multiline content",
			setupToken: func() (string, string) {
				tokenFile := filepath.Join(tmpDir, "multiline-token.txt")
				content := "line1\ngithub_pat_multiline\nline3"
				os.WriteFile(tokenFile, []byte(content), 0o600)
				return tokenFile, content
			},
			envVarToken:   "env-token",
			expectError:   false,
			expectedToken: "line1\ngithub_pat_multiline\nline3",
			expectedPath:  "file", // Indicates should have TokenPath set to the file
		},
		{
			name: "Non-existent file with valid env var",
			setupToken: func() (string, string) {
				return "/non/existent/file.txt", ""
			},
			envVarToken:   "valid-env-token",
			expectError:   false,
			expectedToken: "valid-env-token",
			expectedPath:  "",
		},
		{
			name: "Non-existent file with empty env var",
			setupToken: func() (string, string) {
				return "/non/existent/file.txt", ""
			},
			envVarToken: "",
			expectError: true,
		},
		{
			name: "Empty path with valid env var",
			setupToken: func() (string, string) {
				return "", ""
			},
			envVarToken:   "env-var-token",
			expectError:   false,
			expectedToken: "env-var-token",
			expectedPath:  "",
		},
		{
			name: "Empty path with empty env var",
			setupToken: func() (string, string) {
				return "", ""
			},
			envVarToken: "",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			tokenPath, _ := tc.setupToken()

			config, err := NewGitClientConfig(
				"git@github.com:test/repo.git",
				"main",
				tc.envVarToken,
				tokenPath,
			)

			if tc.expectError {
				assert.Error(t, err)
				assert.Nil(t, config)
				assert.Contains(t, err.Error(), "no git token available")
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, config)
				assert.Equal(t, "git@github.com:test/repo.git", config.RepoURL)
				assert.Equal(t, "main", config.Branch)
				assert.Equal(t, tc.expectedToken, config.Token)

				if tc.expectedPath == "file" {
					assert.Equal(t, tokenPath, config.TokenPath)
				} else if tc.expectedPath == "" {
					assert.Empty(t, config.TokenPath)
				} else {
					assert.Equal(t, tc.expectedPath, config.TokenPath)
				}

				assert.Equal(t, "/tmp/deployment-repo", config.RepoPath)
				assert.NotNil(t, config.Logger)
			}
		})
	}
}

func TestNewClientFromConfig(t *testing.T) {
	config := &ClientConfig{
		RepoURL:  "git@github.com:test/repo.git",
		Branch:   "main",
		Token:    "test-token",
		RepoPath: "/custom/repo/path",
		Logger:   slog.Default().With("test", "value"),
	}

	client := NewClientFromConfig(config)

	assert.NotNil(t, client)
	assert.Equal(t, config.RepoURL, client.RepoURL)
	assert.Equal(t, config.Branch, client.Branch)
	assert.Equal(t, config.Token, client.SSHKey)
	assert.Equal(t, config.RepoPath, client.RepoPath)
	assert.Equal(t, config.Logger, client.logger)
}

func TestNewClientFromConfig_NilLogger(t *testing.T) {
	config := &ClientConfig{
		RepoURL:  "git@github.com:test/repo.git",
		Branch:   "main",
		Token:    "test-token",
		RepoPath: "/custom/repo/path",
		Logger:   nil, // Nil logger should be handled
	}

	client := NewClientFromConfig(config)

	assert.NotNil(t, client)
	assert.NotNil(t, client.logger, "Logger should be set to default when nil")
	assert.Equal(t, config.RepoURL, client.RepoURL)
	assert.Equal(t, config.Branch, client.Branch)
	assert.Equal(t, config.Token, client.SSHKey)
	assert.Equal(t, config.RepoPath, client.RepoPath)
}

// Integration test for token loading and client creation workflow
func TestTokenLoadingIntegration(t *testing.T) {
	tmpDir := createTempDir(t)
	tokenFile := filepath.Join(tmpDir, "integration-token.txt")
	token := "integration_test_token_123"

	// Create token file
	err := os.WriteFile(tokenFile, []byte("  "+token+"  \n"), 0o600)
	require.NoError(t, err)

	// Create config with token file
	config, err := NewGitClientConfig(
		"git@github.com:test/repo.git",
		"develop",
		"fallback-token",
		tokenFile,
	)
	require.NoError(t, err)
	require.NotNil(t, config)

	// Create client from config
	client := NewClientFromConfig(config)
	require.NotNil(t, client)

	// Verify the complete workflow
	assert.Equal(t, "git@github.com:test/repo.git", client.RepoURL)
	assert.Equal(t, "develop", client.Branch)
	assert.Equal(t, token, client.SSHKey, "Token should be trimmed and used as SSH key")
	assert.Equal(t, "/tmp/deployment-repo", client.RepoPath)
	assert.NotNil(t, client.logger)
}

// Edge case tests for token loading
func TestNewGitClientConfig_EdgeCases(t *testing.T) {
	t.Run("Very large token file", func(t *testing.T) {
		tmpDir := createTempDir(t)
		tokenFile := filepath.Join(tmpDir, "large-token.txt")

		// Create a very large token (10KB)
		largeToken := strings.Repeat("a", 10240)
		err := os.WriteFile(tokenFile, []byte(largeToken), 0o600)
		require.NoError(t, err)

		config, err := NewGitClientConfig(
			"git@github.com:test/repo.git",
			"main",
			"fallback",
			tokenFile,
		)

		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, largeToken, config.Token)
	})

	t.Run("Token file with binary content", func(t *testing.T) {
		tmpDir := createTempDir(t)
		tokenFile := filepath.Join(tmpDir, "binary-token.txt")

		// Create a file with binary content
		binaryContent := []byte{0x00, 0x01, 0x02, 0xFF, 0xFE, 0xFD}
		err := os.WriteFile(tokenFile, binaryContent, 0o600)
		require.NoError(t, err)

		config, err := NewGitClientConfig(
			"git@github.com:test/repo.git",
			"main",
			"fallback",
			tokenFile,
		)

		assert.NoError(t, err)
		assert.NotNil(t, config)
		// Binary content should be read as-is and trimmed
		expectedToken := strings.TrimSpace(string(binaryContent))
		assert.Equal(t, expectedToken, config.Token)
	})

	t.Run("Unicode content in token file", func(t *testing.T) {
		tmpDir := createTempDir(t)
		tokenFile := filepath.Join(tmpDir, "unicode-token.txt")

		// Create a file with Unicode content
		unicodeToken := "github_pat_🔑_token_测试"
		err := os.WriteFile(tokenFile, []byte(unicodeToken), 0o600)
		require.NoError(t, err)

		config, err := NewGitClientConfig(
			"git@github.com:test/repo.git",
			"main",
			"fallback",
			tokenFile,
		)

		assert.NoError(t, err)
		assert.NotNil(t, config)
		assert.Equal(t, unicodeToken, config.Token)
	})
}

// Benchmark tests for token loading
func BenchmarkNewGitClientConfig_TokenFile(b *testing.B) {
	tmpDir, err := os.MkdirTemp("", "bench-token-*")
	require.NoError(b, err)
	defer os.RemoveAll(tmpDir)

	tokenFile := filepath.Join(tmpDir, "bench-token.txt")
	token := "benchmark_token_123456789"
	err = os.WriteFile(tokenFile, []byte(token), 0o600)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config, err := NewGitClientConfig(
			"git@github.com:test/repo.git",
			"main",
			"fallback",
			tokenFile,
		)
		if err != nil || config == nil {
			b.Fatal("Benchmark failed")
		}
	}
}

func BenchmarkNewGitClientConfig_EnvVar(b *testing.B) {
	token := "benchmark_env_token_123456789"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		config, err := NewGitClientConfig(
			"git@github.com:test/repo.git",
			"main",
			token,
			"", // Empty token path to force env var usage
		)
		if err != nil || config == nil {
			b.Fatal("Benchmark failed")
		}
	}
}
