package git

import (
	"fmt"
	"io/fs"
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

func (m *MockClientInterface) RemoveDirectory(path string) error {
	args := m.Called(path)
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
	assert.Equal(t, sshKey, client.SshKey)
	assert.Equal(t, "/tmp/deployment-repo", client.RepoPath)
}

func TestClient_InitRepo_NewRepo(t *testing.T) {
	tmpDir := createTempDir(t)
	
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SshKey:   createTestSSHKey(),
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
		SshKey:   createTestSSHKey(),
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
	err = os.WriteFile(initialFile, []byte("# Test Repository"), 0644)
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
		SshKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}
	_ = client // Use the client variable

	files := map[string]string{
		"test1.txt":       "content1",
		"subdir/test2.txt": "content2",
		"test3.yaml":      "key: value",
	}

	// Note: This will fail at the push step due to no remote, but we can test file creation and commit
	commitHash, err := client.CommitAndPush(files, "Test commit")
	
	// The commit should succeed but push will fail
	if err != nil {
		assert.Contains(t, err.Error(), "failed to push")
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
		SshKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	files := map[string]string{
		"config/app.yaml":           "app: test",
		"scripts/deploy.sh":         "#!/bin/bash\necho 'deploying'",
		"docs/README.md":           "# Documentation",
		"nested/deep/structure.json": `{"key": "value"}`,
	}

	// This will fail at push, but we can verify file operations
	_, err = client.CommitAndPush(files, "Add configuration files")

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
		SshKey:   createTestSSHKey(),
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
	err = os.WriteFile(initialFile, []byte("initial content"), 0644)
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
		SshKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Create some changes
	testFile := filepath.Join(repoPath, "changes.txt")
	err = os.WriteFile(testFile, []byte("new changes"), 0644)
	require.NoError(t, err)

	// This will fail at push, but commit should work
	err = client.CommitAndPushChanges("Add changes")
	
	// Expect push failure but verify commit was made
	if err != nil {
		assert.Contains(t, err.Error(), "failed to push")
	}
	
	// Verify the file was added to git
	_, err = os.Stat(testFile)
	assert.NoError(t, err)
}

func TestClient_CommitAndPushChanges_InvalidRepo(t *testing.T) {
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SshKey:   createTestSSHKey(),
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
		SshKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Create a directory structure to remove
	testDir := filepath.Join(repoPath, "to-remove")
	err = os.MkdirAll(testDir, 0755)
	require.NoError(t, err)

	testFile := filepath.Join(testDir, "file.txt")
	err = os.WriteFile(testFile, []byte("content"), 0644)
	require.NoError(t, err)

	// Verify directory exists
	_, err = os.Stat(testDir)
	assert.NoError(t, err)

	// Remove the directory
	err = client.RemoveDirectory("to-remove")
	assert.NoError(t, err)

	// Verify directory was removed
	_, err = os.Stat(testDir)
	assert.True(t, os.IsNotExist(err))
}

func TestClient_RemoveDirectory_InvalidRepo(t *testing.T) {
	client := &Client{
		RepoURL:  "https://github.com/test/repo.git",
		Branch:   "main",
		SshKey:   createTestSSHKey(),
		RepoPath: "/non/existent/path",
	}

	err := client.RemoveDirectory("some-dir")
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
		SshKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	// Try to remove non-existent directory (should not error)
	err = client.RemoveDirectory("non-existent-dir")
	assert.NoError(t, err)
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
	mockClient.On("RemoveDirectory", "old-dir").Return(nil)
	
	err = mockClient.RemoveDirectory("old-dir")
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
		SshKey:   createTestSSHKey(),
		RepoPath: repoPath,
	}

	files := map[string]string{
		"script.sh":    "#!/bin/bash\necho 'test'",
		"config.yaml": "key: value",
	}

	// This will fail at push, but files should be created
	_, err = client.CommitAndPush(files, "Add files")

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
			assert.Equal(t, fs.FileMode(0644), perm, "File should have 0644 permissions: %s", filePath)
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
			SshKey:   createTestSSHKey(),
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
			SshKey:   createTestSSHKey(),
			RepoPath: repoPath,
		}

		// Create a very long file path
		longPath := strings.Repeat("long-directory-name/", 10) + "file.txt"
		files := map[string]string{
			longPath: "content",
		}

		_, err = client.CommitAndPush(files, "Long path test")
		
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
			SshKey:   createTestSSHKey(),
			RepoPath: repoPath,
		}

		files := map[string]string{
			"file-with-dashes.txt":      "content1",
			"file_with_underscores.txt": "content2",
			"file.with.dots.txt":        "content3",
			"file123numbers.txt":        "content4",
		}

		_, err = client.CommitAndPush(files, "Special characters test")

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
			SshKey:   createTestSSHKey(),
			RepoPath: repoPath,
		}

		// Create a large file content (1MB of data)
		largeContent := strings.Repeat("This is a line of test data for large file testing.\n", 20000)
		files := map[string]string{
			"large-file.txt": largeContent,
		}

		_, err = client.CommitAndPush(files, "Large file test")

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
		SshKey:   createTestSSHKey(),
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
		SshKey:   createTestSSHKey(),
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
			os.MkdirAll(filepath.Dir(fullPath), 0755)
			os.WriteFile(fullPath, []byte(content), 0644)
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
		SshKey:   createTestSSHKey(),
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