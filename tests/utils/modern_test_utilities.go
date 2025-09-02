// Package testutils provides modern testing utilities following 2025 testing standards
// for the Nephoran Intent Operator with comprehensive test patterns and fixtures.
package testutils

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

// TestContext provides enhanced context for modern testing patterns
type TestContext struct {
	t           *testing.T
	ctx         context.Context
	cancel      context.CancelFunc
	cleanup     []func()
	assertions  *AssertionHelper
	fixtures    *FixtureManager
	mutex       sync.RWMutex
}

// NewTestContext creates a new modern test context following 2025 standards
func NewTestContext(t *testing.T) *TestContext {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	
	tc := &TestContext{
		t:          t,
		ctx:        ctx,
		cancel:     cancel,
		cleanup:    make([]func(), 0),
		assertions: NewAssertionHelper(t),
		fixtures:   NewFixtureManager(),
		mutex:      sync.RWMutex{},
	}
	
	// Register cleanup for context cancellation
	t.Cleanup(func() {
		tc.Cleanup()
	})
	
	// Enable goroutine leak detection following 2025 standards
	if testing.Short() == false {
		goleak.VerifyNone(t)
	}
	
	return tc
}

// Context returns the test context with timeout
func (tc *TestContext) Context() context.Context {
	return tc.ctx
}

// T returns the underlying testing.T
func (tc *TestContext) T() *testing.T {
	return tc.t
}

// AddCleanup registers a cleanup function
func (tc *TestContext) AddCleanup(fn func()) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	tc.cleanup = append(tc.cleanup, fn)
}

// Cleanup executes all registered cleanup functions
func (tc *TestContext) Cleanup() {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	
	// Cancel context first
	if tc.cancel != nil {
		tc.cancel()
	}
	
	// Execute cleanup functions in reverse order
	for i := len(tc.cleanup) - 1; i >= 0; i-- {
		if tc.cleanup[i] != nil {
			tc.cleanup[i]()
		}
	}
}

// Assertions returns the assertion helper
func (tc *TestContext) Assertions() *AssertionHelper {
	return tc.assertions
}

// Fixtures returns the fixture manager
func (tc *TestContext) Fixtures() *FixtureManager {
	return tc.fixtures
}

// AssertionHelper provides enhanced assertions following 2025 testing standards
type AssertionHelper struct {
	t *testing.T
}

// NewAssertionHelper creates a new assertion helper
func NewAssertionHelper(t *testing.T) *AssertionHelper {
	return &AssertionHelper{t: t}
}

// RequireNoError asserts that error is nil with enhanced error reporting
func (a *AssertionHelper) RequireNoError(err error, msgAndArgs ...interface{}) {
	if err != nil {
		a.t.Helper()
		require.NoError(a.t, err, msgAndArgs...)
	}
}

// RequireEqual asserts equality with type-safe comparison
func (a *AssertionHelper) RequireEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	a.t.Helper()
	require.Equal(a.t, expected, actual, msgAndArgs...)
}

// RequireJSONEqual asserts JSON equality with detailed diff
func (a *AssertionHelper) RequireJSONEqual(expected, actual json.RawMessage, msgAndArgs ...interface{}) {
	a.t.Helper()
	
	var expectedObj, actualObj interface{}
	
	err := json.Unmarshal(expected, &expectedObj)
	require.NoError(a.t, err, "Failed to unmarshal expected JSON")
	
	err = json.Unmarshal(actual, &actualObj)
	require.NoError(a.t, err, "Failed to unmarshal actual JSON")
	
	require.Equal(a.t, expectedObj, actualObj, msgAndArgs...)
}

// RequireEventuallyTrue implements eventual consistency testing
func (a *AssertionHelper) RequireEventuallyTrue(condition func() bool, timeout time.Duration, msgAndArgs ...interface{}) {
	a.t.Helper()
	
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	timeoutCh := time.After(timeout)
	
	for {
		select {
		case <-ticker.C:
			if condition() {
				return
			}
		case <-timeoutCh:
			a.t.Fatalf("Condition never became true within %v: %v", timeout, fmt.Sprint(msgAndArgs...))
		}
	}
}

// AssertHTTPStatusCode asserts HTTP response status with context
func (a *AssertionHelper) AssertHTTPStatusCode(expected, actual int, url string, msgAndArgs ...interface{}) {
	a.t.Helper()
	
	if expected != actual {
		msg := fmt.Sprintf("HTTP status mismatch for URL %s: expected %d, got %d", url, expected, actual)
		if len(msgAndArgs) > 0 {
			msg += fmt.Sprintf(". %v", fmt.Sprint(msgAndArgs...))
		}
		assert.Equal(a.t, expected, actual, msg)
	}
}

// FixtureManager handles test data and fixtures following 2025 patterns
type FixtureManager struct {
	fixtures map[string]interface{}
	mutex    sync.RWMutex
}

// NewFixtureManager creates a new fixture manager
func NewFixtureManager() *FixtureManager {
	return &FixtureManager{
		fixtures: make(map[string]interface{}),
		mutex:    sync.RWMutex{},
	}
}

// LoadFixture loads a fixture by name with type safety
func (fm *FixtureManager) LoadFixture(name string, target interface{}) error {
	fm.mutex.RLock()
	defer fm.mutex.RUnlock()
	
	fixture, exists := fm.fixtures[name]
	if !exists {
		return fmt.Errorf("fixture %s not found", name)
	}
	
	// Type-safe fixture loading using JSON marshaling
	data, err := json.Marshal(fixture)
	if err != nil {
		return fmt.Errorf("failed to marshal fixture %s: %w", name, err)
	}
	
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal fixture %s: %w", name, err)
	}
	
	return nil
}

// RegisterFixture registers a fixture with the manager
func (fm *FixtureManager) RegisterFixture(name string, fixture interface{}) {
	fm.mutex.Lock()
	defer fm.mutex.Unlock()
	fm.fixtures[name] = fixture
}

// CreateNetworkIntentFixture creates a NetworkIntent test fixture
func (fm *FixtureManager) CreateNetworkIntentFixture(name, namespace, intentType string) map[string]interface{} {
	fixture := map[string]interface{}{
		"apiVersion": "nephoran.io/v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels": map[string]interface{}{
				"test.nephoran.io/fixture": "true",
				"test.nephoran.io/type":    intentType,
			},
		},
		"spec": map[string]interface{}{
			"intentType": intentType,
			"parameters": map[string]interface{}{
				"replicas": 3,
				"version":  "latest",
			},
		},
		"status": map[string]interface{}{
			"phase": "Pending",
			"conditions": []map[string]interface{}{
				{
					"type":   "Ready",
					"status": "False",
					"reason": "Initializing",
				},
			},
		},
	}
	
	fm.RegisterFixture(fmt.Sprintf("NetworkIntent/%s", name), fixture)
	return fixture
}

// CreateLLMResponseFixture creates a standardized LLM response fixture
func (fm *FixtureManager) CreateLLMResponseFixture(intentType string, confidence float64) *LLMResponse {
	response := &LLMResponse{
		IntentType:     intentType,
		Confidence:     confidence,
		Parameters:     json.RawMessage(`{"replicas": 3, "version": "latest"}`),
		Manifests:      json.RawMessage(`{"apiVersion": "apps/v1", "kind": "Deployment"}`),
		ProcessingTime: 1500,
		TokensUsed:     250,
		Model:          "gpt-4o-mini",
	}
	
	fm.RegisterFixture(fmt.Sprintf("LLMResponse/%s", intentType), response)
	return response
}

// TestDataFactory provides factory methods for creating test data
type TestDataFactory struct {
	sequence int
	mutex    sync.Mutex
}

// NewTestDataFactory creates a new test data factory
func NewTestDataFactory() *TestDataFactory {
	return &TestDataFactory{
		sequence: 0,
		mutex:    sync.Mutex{},
	}
}

// GenerateUniqueID generates a unique test ID
func (tdf *TestDataFactory) GenerateUniqueID(prefix string) string {
	tdf.mutex.Lock()
	defer tdf.mutex.Unlock()
	
	tdf.sequence++
	return fmt.Sprintf("%s-%d-%d", prefix, time.Now().Unix(), tdf.sequence)
}

// CreateTestNamespace generates a unique test namespace
func (tdf *TestDataFactory) CreateTestNamespace() string {
	return tdf.GenerateUniqueID("test-ns")
}

// TableTestCase represents a table-driven test case following 2025 patterns
type TableTestCase[T any, R any] struct {
	Name        string
	Input       T
	Expected    R
	ShouldError bool
	ErrorType   error
	Setup       func(*TestContext) error
	Teardown    func(*TestContext)
	Timeout     time.Duration
}

// RunTableTests executes table-driven tests with enhanced error handling
func RunTableTests[T any, R any](
	t *testing.T,
	testFunc func(context.Context, T) (R, error),
	testCases []TableTestCase[T, R],
) {
	for _, tc := range testCases {
		tc := tc // Capture loop variable for parallel execution
		
		t.Run(tc.Name, func(t *testing.T) {
			if testing.Short() && tc.Timeout > 10*time.Second {
				t.Skip("Skipping long-running test in short mode")
			}
			
			// Enable parallel execution for independent tests
			t.Parallel()
			
			testCtx := NewTestContext(t)
			
			// Setup phase
			if tc.Setup != nil {
				err := tc.Setup(testCtx)
				require.NoError(t, err, "Test setup failed")
			}
			
			// Teardown phase
			if tc.Teardown != nil {
				testCtx.AddCleanup(func() {
					tc.Teardown(testCtx)
				})
			}
			
			// Set custom timeout if specified
			ctx := testCtx.Context()
			if tc.Timeout > 0 {
				var cancel context.CancelFunc
				ctx, cancel = context.WithTimeout(ctx, tc.Timeout)
				defer cancel()
			}
			
			// Execute test
			result, err := testFunc(ctx, tc.Input)
			
			// Assert results
			if tc.ShouldError {
				require.Error(t, err, "Expected error but got none")
				if tc.ErrorType != nil {
					require.ErrorIs(t, err, tc.ErrorType, "Error type mismatch")
				}
			} else {
				require.NoError(t, err, "Unexpected error")
				require.Equal(t, tc.Expected, result, "Result mismatch")
			}
		})
	}
}

// BenchmarkHelper provides utilities for performance testing
type BenchmarkHelper struct {
	b *testing.B
}

// NewBenchmarkHelper creates a new benchmark helper
func NewBenchmarkHelper(b *testing.B) *BenchmarkHelper {
	return &BenchmarkHelper{b: b}
}

// MeasureOperation measures operation performance with memory tracking
func (bh *BenchmarkHelper) MeasureOperation(operation func()) {
	bh.b.Helper()
	
	bh.b.ReportAllocs()
	bh.b.ResetTimer()
	
	for i := 0; i < bh.b.N; i++ {
		operation()
	}
}

// MeasureAsyncOperation measures async operation performance
func (bh *BenchmarkHelper) MeasureAsyncOperation(operation func() <-chan struct{}) {
	bh.b.Helper()
	
	bh.b.ReportAllocs()
	bh.b.ResetTimer()
	
	for i := 0; i < bh.b.N; i++ {
		done := operation()
		<-done
	}
}