/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package porch

import (
	"context"
	"fmt"
	"sync"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MockPorchClient implements the PorchClient interface for testing
type MockPorchClient struct {
	repositories    map[string]*Repository
	packages        map[string]*PackageRevision
	workflows       map[string]*Workflow
	packageContents map[string]map[string][]byte
	mutex           sync.RWMutex

	// Behavior control flags
	simulateErrors     bool
	simulateLatency    time.Duration
	circuitBreakerOpen bool
	rateLimited        bool
	healthCheckFails   bool

	// Call tracking for assertions
	calls map[string]int
}

// NewMockPorchClient creates a new mock Porch client
func NewMockPorchClient() *MockPorchClient {
	return &MockPorchClient{
		repositories:    make(map[string]*Repository),
		packages:        make(map[string]*PackageRevision),
		workflows:       make(map[string]*Workflow),
		packageContents: make(map[string]map[string][]byte),
		calls:           make(map[string]int),
	}
}

// Repository Operations

func (m *MockPorchClient) GetRepository(ctx context.Context, name string) (*Repository, error) {
	m.trackCall("GetRepository")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	if repo, exists := m.repositories[name]; exists {
		return m.cloneRepository(repo), nil
	}

	return nil, fmt.Errorf("repository %s not found", name)
}

func (m *MockPorchClient) ListRepositories(ctx context.Context, opts *ListOptions) (*RepositoryList, error) {
	m.trackCall("ListRepositories")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.RLock()
	defer m.mutex.RUnlock()

	items := make([]Repository, 0, len(m.repositories))
	for _, repo := range m.repositories {
		if m.matchesListOptions(repo.ObjectMeta, opts) {
			items = append(items, *m.cloneRepository(repo))
		}
	}

	return &RepositoryList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "config.porch.kpt.dev/v1alpha1",
			Kind:       "RepositoryList",
		},
		Items: items,
	}, nil
}

func (m *MockPorchClient) CreateRepository(ctx context.Context, repo *Repository) (*Repository, error) {
	m.trackCall("CreateRepository")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.repositories[repo.Name]; exists {
		return nil, fmt.Errorf("repository %s already exists", repo.Name)
	}

	created := m.cloneRepository(repo)
	created.Status.Conditions = []metav1.Condition{
		{
			Type:    "Ready",
			Status:  metav1.ConditionTrue,
			Reason:  "RepositoryCreated",
			Message: "Repository created successfully",
		},
	}
	created.Status.Health = RepositoryHealthHealthy
	now := metav1.Now()
	created.Status.LastSyncTime = &now

	m.repositories[repo.Name] = created
	return m.cloneRepository(created), nil
}

func (m *MockPorchClient) UpdateRepository(ctx context.Context, repo *Repository) (*Repository, error) {
	m.trackCall("UpdateRepository")

	if err := m.simulateConditions(); err != nil {
		return nil, err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.repositories[repo.Name]; !exists {
		return nil, fmt.Errorf("repository %s not found", repo.Name)
	}

	updated := m.cloneRepository(repo)
	m.repositories[repo.Name] = updated
	return m.cloneRepository(updated), nil
}

func (m *MockPorchClient) DeleteRepository(ctx context.Context, name string) error {
	m.trackCall("DeleteRepository")

	if err := m.simulateConditions(); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.repositories[name]; !exists {
		return fmt.Errorf("repository %s not found", name)
	}

	delete(m.repositories, name)
	return nil
}

func (m *MockPorchClient) SyncRepository(ctx context.Context, name string) error {
	m.trackCall("SyncRepository")

	if err := m.simulateConditions(); err != nil {
		return err
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	repo, exists := m.repositories[name]
	if !exists {
		return fmt.Errorf("repository %s not found", name)
	}

	now := metav1.Now()
	repo.Status.LastSyncTime = &now
	repo.Status.SyncError = ""

	return nil
}

// Test utility methods

// SetSimulateErrors enables or disables error simulation
func (m *MockPorchClient) SetSimulateErrors(enable bool) {
	m.simulateErrors = enable
}

// SetSimulateLatency sets artificial latency for operations
func (m *MockPorchClient) SetSimulateLatency(duration time.Duration) {
	m.simulateLatency = duration
}

// SetCircuitBreakerOpen simulates circuit breaker being open
func (m *MockPorchClient) SetCircuitBreakerOpen(open bool) {
	m.circuitBreakerOpen = open
}

// SetRateLimited simulates rate limiting
func (m *MockPorchClient) SetRateLimited(limited bool) {
	m.rateLimited = limited
}

// SetHealthCheckFails simulates health check failures
func (m *MockPorchClient) SetHealthCheckFails(fails bool) {
	m.healthCheckFails = fails
}

// GetCallCount returns the number of times a method was called
func (m *MockPorchClient) GetCallCount(method string) int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return m.calls[method]
}

// ResetCallCounts resets all call counters
func (m *MockPorchClient) ResetCallCounts() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.calls = make(map[string]int)
}

// AddRepository adds a repository to the mock data
func (m *MockPorchClient) AddRepository(repo *Repository) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.repositories[repo.Name] = m.cloneRepository(repo)
}

// Clear removes all mock data
func (m *MockPorchClient) Clear() {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.repositories = make(map[string]*Repository)
	m.packages = make(map[string]*PackageRevision)
	m.workflows = make(map[string]*Workflow)
	m.packageContents = make(map[string]map[string][]byte)
	m.calls = make(map[string]int)
}

// Helper methods

func (m *MockPorchClient) trackCall(method string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.calls[method]++
}

func (m *MockPorchClient) simulateConditions() error {
	if m.simulateLatency > 0 {
		time.Sleep(m.simulateLatency)
	}

	if m.circuitBreakerOpen {
		return fmt.Errorf("circuit breaker is open")
	}

	if m.rateLimited {
		return fmt.Errorf("rate limit exceeded")
	}

	if m.simulateErrors {
		return fmt.Errorf("simulated error")
	}

	return nil
}

func (m *MockPorchClient) matchesListOptions(meta metav1.ObjectMeta, opts *ListOptions) bool {
	if opts == nil {
		return true
	}

	// Simple label selector matching
	if opts.LabelSelector != "" {
		// This is a simplified implementation
		// In real scenarios, you'd parse the selector properly
		return true
	}

	// Simple namespace filtering
	if opts.Namespace != "" && meta.Namespace != opts.Namespace {
		return false
	}

	return true
}

// Deep copy methods

func (m *MockPorchClient) cloneRepository(repo *Repository) *Repository {
	if repo == nil {
		return nil
	}
	// Manual deep copy for repository
	clone := *repo

	// Deep copy conditions
	if repo.Status.Conditions != nil {
		clone.Status.Conditions = make([]metav1.Condition, len(repo.Status.Conditions))
		copy(clone.Status.Conditions, repo.Status.Conditions)
	}

	return &clone
}