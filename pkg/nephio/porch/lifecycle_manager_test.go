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
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.com/thc1006/nephoran-intent-operator/pkg/nephio/porch/testutil"
)

// MockLifecycleManager implements lifecycle management for testing
type MockLifecycleManager struct {
	client            PorchClient
	logger            *zap.Logger
	mutex             sync.RWMutex
	transitions       map[string][]LifecycleTransition
	validators        map[PackageRevisionLifecycle]ValidationFunc
	hooks             map[string][]LifecycleHook
	transitionHistory []TransitionEvent
	enabled           bool
}

// LifecycleTransition represents a state transition
type LifecycleTransition struct {
	From      PackageRevisionLifecycle
	To        PackageRevisionLifecycle
	Condition TransitionCondition
	Action    TransitionAction
}

// TransitionCondition checks if transition is allowed
type TransitionCondition func(ctx context.Context, pkg *PackageRevision) (bool, error)

// TransitionAction performs action during transition
type TransitionAction func(ctx context.Context, pkg *PackageRevision) error

// ValidationFunc validates package in specific lifecycle state
type ValidationFunc func(ctx context.Context, pkg *PackageRevision) error

// LifecycleHook executes during lifecycle events
type LifecycleHook func(ctx context.Context, event LifecycleEvent) error

// LifecycleEvent represents a lifecycle event
type LifecycleEvent struct {
	Package   *PackageRevision
	From      PackageRevisionLifecycle
	To        PackageRevisionLifecycle
	Timestamp time.Time
	Metadata  map[string]interface{}
}

// TransitionEvent records transition history
type TransitionEvent struct {
	PackageKey string
	From       PackageRevisionLifecycle
	To         PackageRevisionLifecycle
	Success    bool
	Error      error
	Timestamp  time.Time
	Duration   time.Duration
}

// NewMockLifecycleManager creates a new mock lifecycle manager
func NewMockLifecycleManager(client PorchClient) *MockLifecycleManager {
	manager := &MockLifecycleManager{
		client:            client,
		logger:            zap.New(zap.UseDevMode(true)),
		transitions:       make(map[string][]LifecycleTransition),
		validators:        make(map[PackageRevisionLifecycle]ValidationFunc),
		hooks:             make(map[string][]LifecycleHook),
		transitionHistory: []TransitionEvent{},
		enabled:           true,
	}

	// Setup default transitions
	manager.setupDefaultTransitions()
	manager.setupDefaultValidators()
	manager.setupDefaultHooks()

	return manager
}

// setupDefaultTransitions configures standard lifecycle transitions
func (m *MockLifecycleManager) setupDefaultTransitions() {
	// Draft -> Proposed
	m.RegisterTransition(LifecycleTransition{
		From: PackageRevisionLifecycleDraft,
		To:   PackageRevisionLifecycleProposed,
		Condition: func(ctx context.Context, pkg *PackageRevision) (bool, error) {
			// Check if package content exists and is valid
			contents, err := m.client.GetPackageContents(ctx, pkg.Spec.PackageName, pkg.Spec.Revision)
			if err != nil {
				return false, err
			}
			return len(contents) > 0, nil
		},
		Action: func(ctx context.Context, pkg *PackageRevision) error {
			// Run validation before proposing
			validation, err := m.client.ValidatePackage(ctx, pkg.Spec.PackageName, pkg.Spec.Revision)
			if err != nil {
				return err
			}
			if !validation.Valid {
				return fmt.Errorf("validation failed: %v", validation.Errors)
			}
			return nil
		},
	})

	// Proposed -> Published
	m.RegisterTransition(LifecycleTransition{
		From: PackageRevisionLifecycleProposed,
		To:   PackageRevisionLifecyclePublished,
		Condition: func(ctx context.Context, pkg *PackageRevision) (bool, error) {
			// Check approval status
			return m.isPackageApproved(pkg), nil
		},
		Action: func(ctx context.Context, pkg *PackageRevision) error {
			// Set publish timestamp
			now := metav1.Now()
			pkg.Status.PublishTime = &now
			return nil
		},
	})

	// Proposed -> Draft (rejection)
	m.RegisterTransition(LifecycleTransition{
		From: PackageRevisionLifecycleProposed,
		To:   PackageRevisionLifecycleDraft,
		Condition: func(ctx context.Context, pkg *PackageRevision) (bool, error) {
			// Always allow rejection
			return true, nil
		},
		Action: func(ctx context.Context, pkg *PackageRevision) error {
			// Add rejection condition
			pkg.Status.Conditions = append(pkg.Status.Conditions, metav1.Condition{
				Type:    "Rejected",
				Status:  metav1.ConditionTrue,
				Reason:  "PackageRejected",
				Message: "Package was rejected during review",
			})
			return nil
		},
	})

	// Published -> Deletable
	m.RegisterTransition(LifecycleTransition{
		From: PackageRevisionLifecyclePublished,
		To:   PackageRevisionLifecycleDeletable,
		Condition: func(ctx context.Context, pkg *PackageRevision) (bool, error) {
			// Check if package has downstream dependencies
			return !m.hasDownstreamDependencies(pkg), nil
		},
		Action: func(ctx context.Context, pkg *PackageRevision) error {
			// Clean up resources
			return m.cleanupPackageResources(ctx, pkg)
		},
	})
}

// setupDefaultValidators configures validation functions
func (m *MockLifecycleManager) setupDefaultValidators() {
	m.validators[PackageRevisionLifecycleDraft] = func(ctx context.Context, pkg *PackageRevision) error {
		// Basic validation for draft packages
		if pkg.Spec.PackageName == "" {
			return fmt.Errorf("package name is required")
		}
		return nil
	}

	m.validators[PackageRevisionLifecycleProposed] = func(ctx context.Context, pkg *PackageRevision) error {
		// Enhanced validation for proposed packages
		if err := m.validators[PackageRevisionLifecycleDraft](ctx, pkg); err != nil {
			return err
		}

		// Check for required metadata
		if pkg.Spec.PackageMetadata == nil {
			return fmt.Errorf("package metadata is required for proposed packages")
		}

		return nil
	}

	m.validators[PackageRevisionLifecyclePublished] = func(ctx context.Context, pkg *PackageRevision) error {
		// Comprehensive validation for published packages
		if err := m.validators[PackageRevisionLifecycleProposed](ctx, pkg); err != nil {
			return err
		}

		// Check publish conditions
		if pkg.Status.PublishTime == nil {
			return fmt.Errorf("publish time must be set for published packages")
		}

		return nil
	}
}

// setupDefaultHooks configures lifecycle hooks
func (m *MockLifecycleManager) setupDefaultHooks() {
	// Pre-transition hooks
	m.RegisterHook("pre-transition", func(ctx context.Context, event LifecycleEvent) error {
		m.logger.Info("Pre-transition hook",
			zap.String("package", event.Package.Spec.PackageName),
			zap.String("from", string(event.From)),
			zap.String("to", string(event.To)))
		return nil
	})

	// Post-transition hooks
	m.RegisterHook("post-transition", func(ctx context.Context, event LifecycleEvent) error {
		m.logger.Info("Post-transition hook",
			zap.String("package", event.Package.Spec.PackageName),
			zap.String("from", string(event.From)),
			zap.String("to", string(event.To)))

		// Update package metrics
		m.updatePackageMetrics(event)
		return nil
	})

	// Notification hooks
	m.RegisterHook("notification", func(ctx context.Context, event LifecycleEvent) error {
		// Simulate sending notifications
		m.logger.Info("Notification sent",
			zap.String("package", event.Package.Spec.PackageName),
			zap.String("transition", fmt.Sprintf("%s->%s", event.From, event.To)))
		return nil
	})
}

// RegisterTransition adds a new lifecycle transition
func (m *MockLifecycleManager) RegisterTransition(transition LifecycleTransition) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	key := fmt.Sprintf("%s->%s", transition.From, transition.To)
	m.transitions[key] = append(m.transitions[key], transition)
}

// RegisterHook adds a lifecycle hook
func (m *MockLifecycleManager) RegisterHook(event string, hook LifecycleHook) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	m.hooks[event] = append(m.hooks[event], hook)
}

// TransitionPackage transitions a package to a new lifecycle state
func (m *MockLifecycleManager) TransitionPackage(ctx context.Context, pkg *PackageRevision, targetState PackageRevisionLifecycle) error {
	if !m.enabled {
		return fmt.Errorf("lifecycle manager is disabled")
	}

	start := time.Now()
	currentState := pkg.Spec.Lifecycle

	// Record transition attempt
	defer func() {
		duration := time.Since(start)
		event := TransitionEvent{
			PackageKey: fmt.Sprintf("%s@%s", pkg.Spec.PackageName, pkg.Spec.Revision),
			From:       currentState,
			To:         targetState,
			Timestamp:  start,
			Duration:   duration,
		}

		m.mutex.Lock()
		m.transitionHistory = append(m.transitionHistory, event)
		m.mutex.Unlock()
	}()

	// Check if transition is valid
	if !currentState.CanTransitionTo(targetState) {
		return fmt.Errorf("cannot transition from %s to %s", currentState, targetState)
	}

	// Find applicable transitions
	key := fmt.Sprintf("%s->%s", currentState, targetState)
	m.mutex.RLock()
	transitions, exists := m.transitions[key]
	m.mutex.RUnlock()

	if !exists || len(transitions) == 0 {
		return fmt.Errorf("no transitions defined for %s", key)
	}

	// Execute pre-transition hooks
	event := LifecycleEvent{
		Package:   pkg,
		From:      currentState,
		To:        targetState,
		Timestamp: time.Now(),
	}

	if err := m.executeHooks(ctx, "pre-transition", event); err != nil {
		return fmt.Errorf("pre-transition hook failed: %w", err)
	}

	// Try each transition until one succeeds
	var lastError error
	for _, transition := range transitions {
		// Check condition
		canTransition, err := transition.Condition(ctx, pkg)
		if err != nil {
			lastError = err
			continue
		}
		if !canTransition {
			lastError = fmt.Errorf("transition condition not met")
			continue
		}

		// Validate current state
		if validator, exists := m.validators[currentState]; exists {
			if err := validator(ctx, pkg); err != nil {
				lastError = fmt.Errorf("current state validation failed: %w", err)
				continue
			}
		}

		// Execute transition action
		if err := transition.Action(ctx, pkg); err != nil {
			lastError = fmt.Errorf("transition action failed: %w", err)
			continue
		}

		// Update package state
		pkg.Spec.Lifecycle = targetState

		// Validate new state
		if validator, exists := m.validators[targetState]; exists {
			if err := validator(ctx, pkg); err != nil {
				// Rollback state change
				pkg.Spec.Lifecycle = currentState
				lastError = fmt.Errorf("target state validation failed: %w", err)
				continue
			}
		}

		// Execute post-transition hooks
		if err := m.executeHooks(ctx, "post-transition", event); err != nil {
			m.logger.Warn("Post-transition hook failed", zap.Error(err))
		}

		// Execute notification hooks
		go func() {
			if err := m.executeHooks(context.Background(), "notification", event); err != nil {
				m.logger.Warn("Notification hook failed", zap.Error(err))
			}
		}()

		return nil
	}

	return lastError
}

// GetTransitionHistory returns the history of transitions
func (m *MockLifecycleManager) GetTransitionHistory() []TransitionEvent {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	history := make([]TransitionEvent, len(m.transitionHistory))
	copy(history, m.transitionHistory)
	return history
}

// GetValidTransitions returns valid transitions for a package
func (m *MockLifecycleManager) GetValidTransitions(pkg *PackageRevision) []PackageRevisionLifecycle {
	current := pkg.Spec.Lifecycle
	var validTransitions []PackageRevisionLifecycle

	allStates := []PackageRevisionLifecycle{
		PackageRevisionLifecycleDraft,
		PackageRevisionLifecycleProposed,
		PackageRevisionLifecyclePublished,
		PackageRevisionLifecycleDeletable,
	}

	for _, state := range allStates {
		if current.CanTransitionTo(state) {
			validTransitions = append(validTransitions, state)
		}
	}

	return validTransitions
}

// Helper methods

func (m *MockLifecycleManager) executeHooks(ctx context.Context, event string, lifecycleEvent LifecycleEvent) error {
	m.mutex.RLock()
	hooks, exists := m.hooks[event]
	m.mutex.RUnlock()

	if !exists {
		return nil
	}

	for _, hook := range hooks {
		if err := hook(ctx, lifecycleEvent); err != nil {
			return err
		}
	}

	return nil
}

func (m *MockLifecycleManager) isPackageApproved(pkg *PackageRevision) bool {
	// Simulate approval logic
	for _, condition := range pkg.Status.Conditions {
		if condition.Type == "Approved" && condition.Status == metav1.ConditionTrue {
			return true
		}
	}
	return false
}

func (m *MockLifecycleManager) hasDownstreamDependencies(pkg *PackageRevision) bool {
	// Simulate dependency check
	return len(pkg.Status.Downstream) > 0
}

func (m *MockLifecycleManager) cleanupPackageResources(ctx context.Context, pkg *PackageRevision) error {
	// Simulate resource cleanup
	m.logger.Info("Cleaning up package resources",
		zap.String("package", pkg.Spec.PackageName),
		zap.String("revision", pkg.Spec.Revision))
	return nil
}

func (m *MockLifecycleManager) updatePackageMetrics(event LifecycleEvent) {
	// Simulate metrics update
	m.logger.Info("Updated package metrics",
		zap.String("package", event.Package.Spec.PackageName),
		zap.String("transition", fmt.Sprintf("%s->%s", event.From, event.To)))
}

// Enable/Disable lifecycle management
func (m *MockLifecycleManager) Enable() {
	m.enabled = true
}

func (m *MockLifecycleManager) Disable() {
	m.enabled = false
}

// Test Cases

// TestLifecycleManagerCreation tests lifecycle manager creation
func TestLifecycleManagerCreation(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()
	manager := NewMockLifecycleManager(mockClient)

	assert.NotNil(t, manager)
	assert.NotNil(t, manager.client)
	assert.True(t, manager.enabled)
	assert.True(t, len(manager.transitions) > 0)
	assert.True(t, len(manager.validators) > 0)
	assert.True(t, len(manager.hooks) > 0)
}

// TestBasicLifecycleTransitions tests basic state transitions
func TestBasicLifecycleTransitions(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()
	manager := NewMockLifecycleManager(mockClient)

	// Create a test package
	pkg := fixture.CreateTestPackageRevision("lifecycle-test", "v1.0.0")
	mockClient.AddPackageRevision(pkg)

	// Add some content to the package
	contents := testutil.GeneratePackageContents()
	err := mockClient.UpdatePackageContents(fixture.Context, "lifecycle-test", "v1.0.0", contents)
	require.NoError(t, err)

	testCases := []struct {
		name         string
		fromState    PackageRevisionLifecycle
		toState      PackageRevisionLifecycle
		expectError  bool
		setupPackage func(*PackageRevision)
	}{
		{
			name:        "draft_to_proposed",
			fromState:   PackageRevisionLifecycleDraft,
			toState:     PackageRevisionLifecycleProposed,
			expectError: false,
		},
		{
			name:        "proposed_to_published",
			fromState:   PackageRevisionLifecycleProposed,
			toState:     PackageRevisionLifecyclePublished,
			expectError: false,
			setupPackage: func(pkg *PackageRevision) {
				// Add approval condition
				pkg.Status.Conditions = append(pkg.Status.Conditions, metav1.Condition{
					Type:   "Approved",
					Status: metav1.ConditionTrue,
					Reason: "PackageApproved",
				})
			},
		},
		{
			name:        "published_to_deletable",
			fromState:   PackageRevisionLifecyclePublished,
			toState:     PackageRevisionLifecycleDeletable,
			expectError: false,
		},
		{
			name:        "invalid_draft_to_published",
			fromState:   PackageRevisionLifecycleDraft,
			toState:     PackageRevisionLifecyclePublished,
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Reset package state
			pkg.Spec.Lifecycle = tc.fromState
			pkg.Status.Conditions = []metav1.Condition{}
			pkg.Status.PublishTime = nil

			// Setup package if needed
			if tc.setupPackage != nil {
				tc.setupPackage(pkg)
			}

			// Attempt transition
			err := manager.TransitionPackage(fixture.Context, pkg, tc.toState)

			if tc.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tc.toState, pkg.Spec.Lifecycle)
			}
		})
	}
}

// TestLifecycleValidation tests state validation
func TestLifecycleValidation(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()
	manager := NewMockLifecycleManager(mockClient)

	testCases := []struct {
		name        string
		pkg         *PackageRevision
		state       PackageRevisionLifecycle
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid_draft_package",
			pkg:         fixture.CreateTestPackageRevision("valid-draft", "v1.0.0"),
			state:       PackageRevisionLifecycleDraft,
			expectError: false,
		},
		{
			name: "invalid_draft_no_name",
			pkg: &PackageRevision{
				Spec: PackageRevisionSpec{
					PackageName: "",
					Repository:  "test-repo",
					Revision:    "v1.0.0",
					Lifecycle:   PackageRevisionLifecycleDraft,
				},
			},
			state:       PackageRevisionLifecycleDraft,
			expectError: true,
			errorMsg:    "package name is required",
		},
		{
			name:        "proposed_without_metadata",
			pkg:         fixture.CreateTestPackageRevision("no-metadata", "v1.0.0"),
			state:       PackageRevisionLifecycleProposed,
			expectError: true,
			errorMsg:    "package metadata is required",
		},
		{
			name: "valid_proposed_with_metadata",
			pkg: fixture.CreateTestPackageRevision("with-metadata", "v1.0.0", func(pkg *PackageRevision) {
				pkg.Spec.PackageMetadata = &PackageMetadata{
					Name:        "with-metadata",
					Description: "Test package with metadata",
				}
			}),
			state:       PackageRevisionLifecycleProposed,
			expectError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validator, exists := manager.validators[tc.state]
			require.True(t, exists, "Validator should exist for state %s", tc.state)

			err := validator(fixture.Context, tc.pkg)

			if tc.expectError {
				assert.Error(t, err)
				if tc.errorMsg != "" {
					assert.Contains(t, err.Error(), tc.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestLifecycleHooks tests lifecycle hooks execution
func TestLifecycleHooks(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()
	manager := NewMockLifecycleManager(mockClient)

	// Track hook execution
	hookCalls := make(map[string]int)
	var hookMutex sync.Mutex

	// Register test hooks
	manager.RegisterHook("test-pre", func(ctx context.Context, event LifecycleEvent) error {
		hookMutex.Lock()
		hookCalls["test-pre"]++
		hookMutex.Unlock()
		return nil
	})

	manager.RegisterHook("test-post", func(ctx context.Context, event LifecycleEvent) error {
		hookMutex.Lock()
		hookCalls["test-post"]++
		hookMutex.Unlock()
		return nil
	})

	// Create and setup package
	pkg := fixture.CreateTestPackageRevision("hook-test", "v1.0.0")
	mockClient.AddPackageRevision(pkg)
	contents := testutil.GeneratePackageContents()
	err := mockClient.UpdatePackageContents(fixture.Context, "hook-test", "v1.0.0", contents)
	require.NoError(t, err)

	// Perform transition
	err = manager.TransitionPackage(fixture.Context, pkg, PackageRevisionLifecycleProposed)
	require.NoError(t, err)

	// Give notification hook time to execute
	time.Sleep(100 * time.Millisecond)

	// Verify hook execution
	hookMutex.Lock()
	assert.Equal(t, 1, hookCalls["test-pre"])
	assert.Equal(t, 1, hookCalls["test-post"])
	hookMutex.Unlock()
}

// TestLifecycleHookErrors tests hook error handling
func TestLifecycleHookErrors(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()
	manager := NewMockLifecycleManager(mockClient)

	// Register failing hook
	manager.RegisterHook("pre-transition", func(ctx context.Context, event LifecycleEvent) error {
		return fmt.Errorf("hook failed")
	})

	// Create and setup package
	pkg := fixture.CreateTestPackageRevision("hook-error-test", "v1.0.0")
	mockClient.AddPackageRevision(pkg)
	contents := testutil.GeneratePackageContents()
	err := mockClient.UpdatePackageContents(fixture.Context, "hook-error-test", "v1.0.0", contents)
	require.NoError(t, err)

	// Attempt transition - should fail due to hook error
	err = manager.TransitionPackage(fixture.Context, pkg, PackageRevisionLifecycleProposed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pre-transition hook failed")
	assert.Equal(t, PackageRevisionLifecycleDraft, pkg.Spec.Lifecycle)
}

// TestTransitionHistory tests transition history tracking
func TestTransitionHistory(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()
	manager := NewMockLifecycleManager(mockClient)

	// Create multiple packages and perform transitions
	packages := []struct {
		name     string
		revision string
	}{
		{"pkg-1", "v1.0.0"},
		{"pkg-2", "v1.0.0"},
		{"pkg-3", "v1.0.0"},
	}

	for _, pkgInfo := range packages {
		pkg := fixture.CreateTestPackageRevision(pkgInfo.name, pkgInfo.revision)
		mockClient.AddPackageRevision(pkg)

		// Add content
		contents := testutil.GeneratePackageContents()
		err := mockClient.UpdatePackageContents(fixture.Context, pkgInfo.name, pkgInfo.revision, contents)
		require.NoError(t, err)

		// Perform transition
		err = manager.TransitionPackage(fixture.Context, pkg, PackageRevisionLifecycleProposed)
		assert.NoError(t, err)
	}

	// Check transition history
	history := manager.GetTransitionHistory()
	assert.Equal(t, len(packages), len(history))

	for i, event := range history {
		assert.Equal(t, PackageRevisionLifecycleDraft, event.From)
		assert.Equal(t, PackageRevisionLifecycleProposed, event.To)
		assert.True(t, event.Success)
		assert.NoError(t, event.Error)
		assert.True(t, event.Duration > 0)
		assert.Equal(t, fmt.Sprintf("%s@%s", packages[i].name, packages[i].revision), event.PackageKey)
	}
}

// TestConcurrentTransitions tests concurrent lifecycle transitions
func TestConcurrentTransitions(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()
	manager := NewMockLifecycleManager(mockClient)

	const numPackages = 10
	const numGoroutines = 5

	// Create packages
	packages := make([]*PackageRevision, numPackages)
	for i := 0; i < numPackages; i++ {
		pkg := fixture.CreateTestPackageRevision(fmt.Sprintf("concurrent-pkg-%d", i), "v1.0.0")
		packages[i] = pkg
		mockClient.AddPackageRevision(pkg)

		// Add content
		contents := testutil.GeneratePackageContents()
		err := mockClient.UpdatePackageContents(fixture.Context, pkg.Spec.PackageName, pkg.Spec.Revision, contents)
		require.NoError(t, err)
	}

	// Perform concurrent transitions
	var wg sync.WaitGroup
	errors := make(chan error, numPackages)

	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(startIdx int) {
			defer wg.Done()

			for j := startIdx; j < numPackages; j += numGoroutines {
				err := manager.TransitionPackage(fixture.Context, packages[j], PackageRevisionLifecycleProposed)
				if err != nil {
					errors <- err
				}
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		if err != nil {
			t.Errorf("Concurrent transition failed: %v", err)
			errorCount++
		}
	}

	assert.Equal(t, 0, errorCount, "Expected no errors in concurrent transitions")

	// Verify all packages transitioned
	for _, pkg := range packages {
		assert.Equal(t, PackageRevisionLifecycleProposed, pkg.Spec.Lifecycle)
	}

	// Check history count
	history := manager.GetTransitionHistory()
	assert.Equal(t, numPackages, len(history))
}

// TestLifecycleManagerDisabled tests behavior when lifecycle management is disabled
func TestLifecycleManagerDisabled(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()
	manager := NewMockLifecycleManager(mockClient)

	// Create package
	pkg := fixture.CreateTestPackageRevision("disabled-test", "v1.0.0")
	mockClient.AddPackageRevision(pkg)

	// Disable lifecycle management
	manager.Disable()

	// Attempt transition - should fail
	err := manager.TransitionPackage(fixture.Context, pkg, PackageRevisionLifecycleProposed)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "lifecycle manager is disabled")
	assert.Equal(t, PackageRevisionLifecycleDraft, pkg.Spec.Lifecycle)

	// Re-enable and try again
	manager.Enable()
	contents := testutil.GeneratePackageContents()
	err = mockClient.UpdatePackageContents(fixture.Context, "disabled-test", "v1.0.0", contents)
	require.NoError(t, err)

	err = manager.TransitionPackage(fixture.Context, pkg, PackageRevisionLifecycleProposed)
	assert.NoError(t, err)
	assert.Equal(t, PackageRevisionLifecycleProposed, pkg.Spec.Lifecycle)
}

// TestValidTransitions tests getting valid transitions for a package
func TestValidTransitions(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()
	manager := NewMockLifecycleManager(mockClient)

	testCases := []struct {
		name                string
		currentState        PackageRevisionLifecycle
		expectedTransitions []PackageRevisionLifecycle
	}{
		{
			name:         "draft_transitions",
			currentState: PackageRevisionLifecycleDraft,
			expectedTransitions: []PackageRevisionLifecycle{
				PackageRevisionLifecycleProposed,
				PackageRevisionLifecycleDeletable,
			},
		},
		{
			name:         "proposed_transitions",
			currentState: PackageRevisionLifecycleProposed,
			expectedTransitions: []PackageRevisionLifecycle{
				PackageRevisionLifecyclePublished,
				PackageRevisionLifecycleDraft,
				PackageRevisionLifecycleDeletable,
			},
		},
		{
			name:         "published_transitions",
			currentState: PackageRevisionLifecyclePublished,
			expectedTransitions: []PackageRevisionLifecycle{
				PackageRevisionLifecycleDeletable,
			},
		},
		{
			name:                "deletable_transitions",
			currentState:        PackageRevisionLifecycleDeletable,
			expectedTransitions: []PackageRevisionLifecycle{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			pkg := fixture.CreateTestPackageRevision("transition-test", "v1.0.0",
				testutil.WithPackageLifecycle(tc.currentState))

			validTransitions := manager.GetValidTransitions(pkg)
			assert.Equal(t, len(tc.expectedTransitions), len(validTransitions))

			for _, expected := range tc.expectedTransitions {
				assert.Contains(t, validTransitions, expected)
			}
		})
	}
}

// TestComplexLifecycleScenarios tests complex real-world scenarios
func TestComplexLifecycleScenarios(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()
	manager := NewMockLifecycleManager(mockClient)

	t.Run("complete_approval_workflow", func(t *testing.T) {
		// Create package
		pkg := fixture.CreateTestPackageRevision("approval-workflow", "v1.0.0")
		mockClient.AddPackageRevision(pkg)

		// Add content
		contents := testutil.GeneratePackageContents()
		err := mockClient.UpdatePackageContents(fixture.Context, "approval-workflow", "v1.0.0", contents)
		require.NoError(t, err)

		// Propose package
		err = manager.TransitionPackage(fixture.Context, pkg, PackageRevisionLifecycleProposed)
		require.NoError(t, err)
		assert.Equal(t, PackageRevisionLifecycleProposed, pkg.Spec.Lifecycle)

		// Simulate approval
		pkg.Status.Conditions = append(pkg.Status.Conditions, metav1.Condition{
			Type:   "Approved",
			Status: metav1.ConditionTrue,
			Reason: "PackageApproved",
		})

		// Publish package
		err = manager.TransitionPackage(fixture.Context, pkg, PackageRevisionLifecyclePublished)
		require.NoError(t, err)
		assert.Equal(t, PackageRevisionLifecyclePublished, pkg.Spec.Lifecycle)
		assert.NotNil(t, pkg.Status.PublishTime)

		// Mark as deletable
		err = manager.TransitionPackage(fixture.Context, pkg, PackageRevisionLifecycleDeletable)
		require.NoError(t, err)
		assert.Equal(t, PackageRevisionLifecycleDeletable, pkg.Spec.Lifecycle)
	})

	t.Run("rejection_and_resubmission", func(t *testing.T) {
		// Create package
		pkg := fixture.CreateTestPackageRevision("rejection-test", "v1.0.0")
		mockClient.AddPackageRevision(pkg)

		// Add content
		contents := testutil.GeneratePackageContents()
		err := mockClient.UpdatePackageContents(fixture.Context, "rejection-test", "v1.0.0", contents)
		require.NoError(t, err)

		// Propose package
		err = manager.TransitionPackage(fixture.Context, pkg, PackageRevisionLifecycleProposed)
		require.NoError(t, err)
		assert.Equal(t, PackageRevisionLifecycleProposed, pkg.Spec.Lifecycle)

		// Reject package (transition back to draft)
		err = manager.TransitionPackage(fixture.Context, pkg, PackageRevisionLifecycleDraft)
		require.NoError(t, err)
		assert.Equal(t, PackageRevisionLifecycleDraft, pkg.Spec.Lifecycle)

		// Check rejection condition was added
		hasRejection := false
		for _, condition := range pkg.Status.Conditions {
			if condition.Type == "Rejected" && condition.Status == metav1.ConditionTrue {
				hasRejection = true
				break
			}
		}
		assert.True(t, hasRejection, "Rejection condition should be present")

		// Resubmit after fixes
		err = manager.TransitionPackage(fixture.Context, pkg, PackageRevisionLifecycleProposed)
		require.NoError(t, err)
		assert.Equal(t, PackageRevisionLifecycleProposed, pkg.Spec.Lifecycle)
	})
}

// BenchmarkLifecycleTransitions benchmarks transition performance
func BenchmarkLifecycleTransitions(b *testing.B) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()
	manager := NewMockLifecycleManager(mockClient)

	// Setup packages
	packages := make([]*PackageRevision, b.N)
	for i := 0; i < b.N; i++ {
		pkg := fixture.CreateTestPackageRevision(fmt.Sprintf("bench-pkg-%d", i), "v1.0.0")
		packages[i] = pkg
		mockClient.AddPackageRevision(pkg)

		contents := testutil.GeneratePackageContents()
		err := mockClient.UpdatePackageContents(fixture.Context, pkg.Spec.PackageName, pkg.Spec.Revision, contents)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i >= len(packages) {
				i = 0
			}
			err := manager.TransitionPackage(fixture.Context, packages[i], PackageRevisionLifecycleProposed)
			if err != nil {
				b.Fatal(err)
			}
			i++
		}
	})
}

// TestLifecycleMetrics tests metrics collection during transitions
func TestLifecycleMetrics(t *testing.T) {
	fixture := testutil.NewTestFixture(context.Background())
	defer fixture.Cleanup()

	mockClient := testutil.NewMockPorchClient()
	manager := NewMockLifecycleManager(mockClient)

	// Track metrics calls
	metricsUpdated := false

	// Replace the metrics update function with a test version
	originalUpdateMetrics := manager.updatePackageMetrics
	manager.updatePackageMetrics = func(event LifecycleEvent) {
		metricsUpdated = true
		originalUpdateMetrics(event)
	}

	// Create and transition package
	pkg := fixture.CreateTestPackageRevision("metrics-test", "v1.0.0")
	mockClient.AddPackageRevision(pkg)

	contents := testutil.GeneratePackageContents()
	err := mockClient.UpdatePackageContents(fixture.Context, "metrics-test", "v1.0.0", contents)
	require.NoError(t, err)

	err = manager.TransitionPackage(fixture.Context, pkg, PackageRevisionLifecycleProposed)
	require.NoError(t, err)

	// Verify metrics were updated
	assert.True(t, metricsUpdated, "Metrics should have been updated")

	// Verify transition was recorded in history
	history := manager.GetTransitionHistory()
	assert.Equal(t, 1, len(history))
	assert.True(t, history[0].Duration > 0)
	assert.Equal(t, "metrics-test@v1.0.0", history[0].PackageKey)
}
