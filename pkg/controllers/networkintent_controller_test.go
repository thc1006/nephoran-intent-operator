package controllers

// MockGitClient is a mock implementation of the GitClient for testing.
type MockGitClient struct {
	CommitAndPushFunc func(files map[string]string, message string) error
}

func (m *MockGitClient) CommitAndPush(files map[string]string, message string) error {
	if m.CommitAndPushFunc != nil {
		return m.CommitAndPushFunc(files, message)
	}
	return nil
}

func (m *MockGitClient) InitRepo() error {
	return nil
}
