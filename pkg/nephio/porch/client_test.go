package porch

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRepositoryManager is a mock implementation of RepositoryManager
type MockRepositoryManager struct {
	mock.Mock
}

// MockPackageRevisionManager is a mock implementation of PackageRevisionManager
type MockPackageRevisionManager struct {
	mock.Mock
}

func TestMockPackageRevisionManager(t *testing.T) {
	// Test that our mock implementations compile and work
	mockPkgManager := &MockPackageRevisionManager{}
	mockRepoManager := &MockRepositoryManager{}

	// Test that the interfaces are satisfied
	var _ PackageRevisionManager = mockPkgManager
	var _ RepositoryManager = mockRepoManager

	// Basic mock setup test
	ctx := context.Background()
	expectedPackageRevision := &PackageRevision{
		Spec: PackageRevisionSpec{
			PackageName: "test-package",
			Revision:    "v1",
		},
	}

	// Test mock method calls
	mockPkgManager.On("GetRevision", ctx, mock.Anything).Return(expectedPackageRevision, nil)
	result, err := mockPkgManager.GetRevision(ctx, &PackageReference{PackageName: "test", Revision: "v1"})

	assert.NoError(t, err)
	assert.NotNil(t, result)
	mockPkgManager.AssertExpectations(t)
}

// Implement the mock methods for testing
func (m *MockPackageRevisionManager) GetPackageRevision(ctx context.Context, name, revision string) (*PackageRevision, error) {
	args := m.Called(ctx, name, revision)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPackageRevisionManager) CreatePackageRevision(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error) {
	args := m.Called(ctx, pkg)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPackageRevisionManager) UpdatePackageRevision(ctx context.Context, pkg *PackageRevision) (*PackageRevision, error) {
	args := m.Called(ctx, pkg)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPackageRevisionManager) DeletePackageRevision(ctx context.Context, name, revision string) error {
	args := m.Called(ctx, name, revision)
	return args.Error(0)
}

func (m *MockPackageRevisionManager) ClonePackage(ctx context.Context, ref *PackageReference, spec *PackageSpec) (*PackageRevision, error) {
	args := m.Called(ctx, ref, spec)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

// Methods to satisfy the full RepositoryManager interface
func (m *MockRepositoryManager) CreateBranch(ctx context.Context, repo, branch, source string) error {
	args := m.Called(ctx, repo, branch, source)
	return args.Error(0)
}

// Additional mock methods for repository management
func (m *MockRepositoryManager) GetRepository(ctx context.Context, name string) (*Repository, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*Repository), args.Error(1)
}

// DeleteBranch deletes a repository branch
func (m *MockRepositoryManager) DeleteBranch(ctx context.Context, repoName, branchName string) error {
	args := m.Called(ctx, repoName, branchName)
	return args.Error(0)
}

// Additional MockRepositoryManager methods to satisfy the interface
func (m *MockRepositoryManager) RegisterRepository(ctx context.Context, config *RepositoryConfig) (*Repository, error) {
	args := m.Called(ctx, config)
	return args.Get(0).(*Repository), args.Error(1)
}

func (m *MockRepositoryManager) UnregisterRepository(ctx context.Context, name string) error {
	args := m.Called(ctx, name)
	return args.Error(0)
}

func (m *MockRepositoryManager) SynchronizeRepository(ctx context.Context, name string) (*SyncResult, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*SyncResult), args.Error(1)
}

func (m *MockRepositoryManager) GetRepositoryHealth(ctx context.Context, name string) (*RepositoryHealth, error) {
	args := m.Called(ctx, name)
	return args.Get(0).(*RepositoryHealth), args.Error(1)
}

func (m *MockRepositoryManager) ListBranches(ctx context.Context, repoName string) ([]string, error) {
	args := m.Called(ctx, repoName)
	return args.Get(0).([]string), args.Error(1)
}

func (m *MockRepositoryManager) UpdateCredentials(ctx context.Context, repoName string, creds *Credentials) error {
	args := m.Called(ctx, repoName, creds)
	return args.Error(0)
}

func (m *MockRepositoryManager) ValidateAccess(ctx context.Context, repoName string) error {
	args := m.Called(ctx, repoName)
	return args.Error(0)
}

func (m *MockPackageRevisionManager) ProposePackageRevision(ctx context.Context, name, revision string) error {
	args := m.Called(ctx, name, revision)
	return args.Error(0)
}

func (m *MockPackageRevisionManager) ApprovePackageRevision(ctx context.Context, name, revision string) error {
	args := m.Called(ctx, name, revision)
	return args.Error(0)
}

func (m *MockPackageRevisionManager) RejectPackageRevision(ctx context.Context, name, revision, reason string) error {
	args := m.Called(ctx, name, revision, reason)
	return args.Error(0)
}

// CompareRevisions compares two package revisions
func (m *MockPackageRevisionManager) CompareRevisions(ctx context.Context, ref1, ref2 *PackageReference) (*ComparisonResult, error) {
	args := m.Called(ctx, ref1, ref2)
	return args.Get(0).(*ComparisonResult), args.Error(1)
}

// Additional MockPackageRevisionManager methods to satisfy the interface
func (m *MockPackageRevisionManager) CreatePackage(ctx context.Context, spec *PackageSpec) (*PackageRevision, error) {
	args := m.Called(ctx, spec)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPackageRevisionManager) DeletePackage(ctx context.Context, ref *PackageReference) error {
	args := m.Called(ctx, ref)
	return args.Error(0)
}

func (m *MockPackageRevisionManager) CreateRevision(ctx context.Context, ref *PackageReference) (*PackageRevision, error) {
	args := m.Called(ctx, ref)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPackageRevisionManager) GetRevision(ctx context.Context, ref *PackageReference) (*PackageRevision, error) {
	args := m.Called(ctx, ref)
	return args.Get(0).(*PackageRevision), args.Error(1)
}

func (m *MockPackageRevisionManager) ListRevisions(ctx context.Context, packageName string) ([]*PackageRevision, error) {
	args := m.Called(ctx, packageName)
	return args.Get(0).([]*PackageRevision), args.Error(1)
}

func (m *MockPackageRevisionManager) PromoteToProposed(ctx context.Context, ref *PackageReference) error {
	args := m.Called(ctx, ref)
	return args.Error(0)
}

func (m *MockPackageRevisionManager) PromoteToPublished(ctx context.Context, ref *PackageReference) error {
	args := m.Called(ctx, ref)
	return args.Error(0)
}

func (m *MockPackageRevisionManager) RevertToRevision(ctx context.Context, ref *PackageReference, targetRevision string) error {
	args := m.Called(ctx, ref, targetRevision)
	return args.Error(0)
}

func (m *MockPackageRevisionManager) UpdateContent(ctx context.Context, ref *PackageReference, updates map[string][]byte) error {
	args := m.Called(ctx, ref, updates)
	return args.Error(0)
}

func (m *MockPackageRevisionManager) GetContent(ctx context.Context, ref *PackageReference) (*PackageContent, error) {
	args := m.Called(ctx, ref)
	return args.Get(0).(*PackageContent), args.Error(1)
}

func (m *MockPackageRevisionManager) ValidateContent(ctx context.Context, ref *PackageReference) (*ValidationResult, error) {
	args := m.Called(ctx, ref)
	return args.Get(0).(*ValidationResult), args.Error(1)
}
