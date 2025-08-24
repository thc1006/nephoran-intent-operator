package testutil

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestTimeout is the default timeout for test operations
const TestTimeout = 30 * time.Second

// EnvironmentGuard provides isolated environment variable management for tests
type EnvironmentGuard struct {
	t           *testing.T
	originalEnv map[string]string
	mu          sync.RWMutex
}

// NewEnvironmentGuard creates a new environment guard for test isolation
func NewEnvironmentGuard(t *testing.T) *EnvironmentGuard {
	t.Helper()
	
	guard := &EnvironmentGuard{
		t:           t,
		originalEnv: make(map[string]string),
	}
	
	t.Cleanup(guard.Restore)
	return guard
}

// Set sets an environment variable for the test and tracks it for cleanup
func (eg *EnvironmentGuard) Set(key, value string) {
	eg.mu.Lock()
	defer eg.mu.Unlock()
	
	eg.t.Helper()
	
	// Store original value if we haven't seen this key before
	if _, exists := eg.originalEnv[key]; !exists {
		if originalValue, wasSet := os.LookupEnv(key); wasSet {
			eg.originalEnv[key] = originalValue
		} else {
			eg.originalEnv[key] = "__UNSET__" // Special marker for unset vars
		}
	}
	
	eg.t.Setenv(key, value)
}

// Unset removes an environment variable for the test
func (eg *EnvironmentGuard) Unset(key string) {
	eg.mu.Lock()
	defer eg.mu.Unlock()
	
	eg.t.Helper()
	
	// Store original value if we haven't seen this key before
	if _, exists := eg.originalEnv[key]; !exists {
		if originalValue, wasSet := os.LookupEnv(key); wasSet {
			eg.originalEnv[key] = originalValue
		} else {
			eg.originalEnv[key] = "__UNSET__"
		}
	}
	
	os.Unsetenv(key)
}

// Restore restores all environment variables to their original state
func (eg *EnvironmentGuard) Restore() {
	eg.mu.Lock()
	defer eg.mu.Unlock()
	
	for key, originalValue := range eg.originalEnv {
		if originalValue == "__UNSET__" {
			os.Unsetenv(key)
		} else {
			os.Setenv(key, originalValue)
		}
	}
}

// MockHTTPServer provides a mock HTTP server for testing
type MockHTTPServer struct {
	server   *httptest.Server
	handlers map[string]http.HandlerFunc
	mu       sync.RWMutex
}

// NewMockHTTPServer creates a new mock HTTP server
func NewMockHTTPServer(t *testing.T) *MockHTTPServer {
	t.Helper()
	
	mock := &MockHTTPServer{
		handlers: make(map[string]http.HandlerFunc),
	}
	
	mux := http.NewServeMux()
	mux.HandleFunc("/", mock.defaultHandler)
	
	mock.server = httptest.NewServer(mux)
	
	t.Cleanup(func() {
		mock.server.Close()
	})
	
	return mock
}

// URL returns the mock server's URL
func (m *MockHTTPServer) URL() string {
	return m.server.URL
}

// SetHandler sets a handler for a specific path
func (m *MockHTTPServer) SetHandler(path string, handler http.HandlerFunc) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[path] = handler
}

// SetJSONResponse sets a JSON response for a specific path
func (m *MockHTTPServer) SetJSONResponse(path string, statusCode int, response string) {
	m.SetHandler(path, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		w.Write([]byte(response))
	})
}

// SetErrorResponse sets an error response for a specific path
func (m *MockHTTPServer) SetErrorResponse(path string, statusCode int, message string) {
	m.SetHandler(path, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, message, statusCode)
	})
}

// defaultHandler handles requests for paths without specific handlers
func (m *MockHTTPServer) defaultHandler(w http.ResponseWriter, r *http.Request) {
	m.mu.RLock()
	handler, exists := m.handlers[r.URL.Path]
	m.mu.RUnlock()
	
	if exists {
		handler(w, r)
		return
	}
	
	// Default success response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status": "ok", "message": "mock response"}`))
}

// ContextWithTimeout creates a context with the default test timeout
func ContextWithTimeout(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), TestTimeout)
	t.Cleanup(cancel)
	return ctx, cancel
}

// ContextWithDeadline creates a context with a specific deadline
func ContextWithDeadline(t *testing.T, timeout time.Duration) (context.Context, context.CancelFunc) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	t.Cleanup(cancel)
	return ctx, cancel
}

// RequireNotNil checks that a pointer is not nil with a helpful message
func RequireNotNil(t *testing.T, ptr interface{}, msgAndArgs ...interface{}) {
	t.Helper()
	require.NotNil(t, ptr, msgAndArgs...)
}

// RequireValidSliceIndex checks that a slice index is valid
func RequireValidSliceIndex(t *testing.T, slice interface{}, index int, msgAndArgs ...interface{}) {
	t.Helper()
	
	var length int
	switch s := slice.(type) {
	case []interface{}:
		length = len(s)
	case []string:
		length = len(s)
	case []int:
		length = len(s)
	case []byte:
		length = len(s)
	default:
		t.Errorf("unsupported slice type: %T", slice)
		return
	}
	
	if index < 0 || index >= length {
		args := append([]interface{}{
			fmt.Sprintf("slice index %d out of bounds for slice of length %d", index, length),
		}, msgAndArgs...)
		t.Error(args...)
	}
}

// SkipIfShort skips the test if running in short mode
func SkipIfShort(t *testing.T) {
	t.Helper()
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
}

// SkipIfCI skips the test if running in CI environment
func SkipIfCI(t *testing.T) {
	t.Helper()
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		t.Skip("skipping test in CI environment")
	}
}

// MockConfig provides a mock configuration for testing
type MockConfig struct {
	values map[string]interface{}
	mu     sync.RWMutex
}

// NewMockConfig creates a new mock configuration
func NewMockConfig() *MockConfig {
	return &MockConfig{
		values: make(map[string]interface{}),
	}
}

// Set sets a configuration value
func (mc *MockConfig) Set(key string, value interface{}) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.values[key] = value
}

// Get gets a configuration value
func (mc *MockConfig) Get(key string) (interface{}, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	value, exists := mc.values[key]
	return value, exists
}

// GetString gets a string configuration value
func (mc *MockConfig) GetString(key string, defaultValue string) string {
	if value, exists := mc.Get(key); exists {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return defaultValue
}

// GetInt gets an int configuration value
func (mc *MockConfig) GetInt(key string, defaultValue int) int {
	if value, exists := mc.Get(key); exists {
		if i, ok := value.(int); ok {
			return i
		}
	}
	return defaultValue
}

// GetBool gets a bool configuration value
func (mc *MockConfig) GetBool(key string, defaultValue bool) bool {
	if value, exists := mc.Get(key); exists {
		if b, ok := value.(bool); ok {
			return b
		}
	}
	return defaultValue
}

// SafeStringSliceAccess safely accesses a string slice with bounds checking
func SafeStringSliceAccess(t *testing.T, slice []string, index int, msgAndArgs ...interface{}) string {
	t.Helper()
	
	if index < 0 || index >= len(slice) {
		args := append([]interface{}{
			fmt.Sprintf("string slice index %d out of bounds for slice of length %d", index, len(slice)),
		}, msgAndArgs...)
		t.Fatal(args...)
		return ""
	}
	
	return slice[index]
}

// SafeMapAccess safely accesses a map with nil checking
func SafeMapAccess[K comparable, V any](t *testing.T, m map[K]V, key K, msgAndArgs ...interface{}) (V, bool) {
	t.Helper()
	
	var zero V
	if m == nil {
		args := append([]interface{}{
			fmt.Sprintf("attempted to access nil map with key %v", key),
		}, msgAndArgs...)
		t.Error(args...)
		return zero, false
	}
	
	value, exists := m[key]
	return value, exists
}

// CleanupCommonEnvVars cleans up commonly used environment variables in tests
func CleanupCommonEnvVars(t *testing.T) {
	t.Helper()
	
	commonEnvVars := []string{
		// Config-related
		"OPENAI_API_KEY",
		"METRICS_ADDR",
		"PROBE_ADDR",
		"ENABLE_LEADER_ELECTION",
		"LLM_PROCESSOR_URL",
		"LLM_PROCESSOR_TIMEOUT",
		"RAG_API_URL",
		"RAG_API_URL_EXTERNAL",
		"RAG_API_TIMEOUT",
		"GIT_REPO_URL",
		"GIT_TOKEN",
		"GIT_BRANCH",
		"WEAVIATE_URL",
		"WEAVIATE_INDEX",
		"OPENAI_MODEL",
		"OPENAI_EMBEDDING_MODEL",
		"NAMESPACE",
		"CRD_PATH",
		
		// Test-specific
		"ENABLE_NETWORK_INTENT",
		"ENABLE_LLM_INTENT",
		"LLM_TIMEOUT_SECS",
		"LLM_MAX_RETRIES",
		"LLM_CACHE_MAX_ENTRIES",
		"HTTP_MAX_BODY",
		"METRICS_ENABLED",
		"METRICS_ALLOWED_IPS",
		
		// CI-related
		"CI",
		"GITHUB_ACTIONS",
		"RUNNING_IN_CI",
		
		// Performance-related
		"GOMAXPROCS",
		"GOGC",
	}
	
	for _, envVar := range commonEnvVars {
		// Use t.Setenv with empty string to effectively unset for the test
		if _, exists := os.LookupEnv(envVar); exists {
			t.Setenv(envVar, "")
		}
	}
}

// WithRetry executes a function with retry logic for flaky operations
func WithRetry(t *testing.T, maxRetries int, delay time.Duration, fn func() error) error {
	t.Helper()
	
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if i < maxRetries-1 {
				time.Sleep(delay)
			}
		}
	}
	return fmt.Errorf("operation failed after %d retries: %w", maxRetries, lastErr)
}

// AssertEventuallyTrue waits for a condition to become true within a timeout
func AssertEventuallyTrue(t *testing.T, condition func() bool, timeout time.Duration, message string) {
	t.Helper()
	
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("condition never became true within %v: %s", timeout, message)
		case <-ticker.C:
			if condition() {
				return
			}
		}
	}
}

// MockFileSystem provides a mock file system for testing
type MockFileSystem struct {
	files map[string][]byte
	dirs  map[string]bool
	mu    sync.RWMutex
}

// NewMockFileSystem creates a new mock file system
func NewMockFileSystem() *MockFileSystem {
	return &MockFileSystem{
		files: make(map[string][]byte),
		dirs:  make(map[string]bool),
	}
}

// WriteFile writes a file to the mock file system
func (mfs *MockFileSystem) WriteFile(path string, data []byte) {
	mfs.mu.Lock()
	defer mfs.mu.Unlock()
	mfs.files[path] = data
	
	// Create parent directories
	parts := strings.Split(path, "/")
	for i := 1; i < len(parts); i++ {
		dir := strings.Join(parts[:i], "/")
		if dir != "" {
			mfs.dirs[dir] = true
		}
	}
}

// ReadFile reads a file from the mock file system
func (mfs *MockFileSystem) ReadFile(path string) ([]byte, bool) {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()
	data, exists := mfs.files[path]
	return data, exists
}

// FileExists checks if a file exists in the mock file system
func (mfs *MockFileSystem) FileExists(path string) bool {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()
	_, exists := mfs.files[path]
	return exists
}

// DirExists checks if a directory exists in the mock file system
func (mfs *MockFileSystem) DirExists(path string) bool {
	mfs.mu.RLock()
	defer mfs.mu.RUnlock()
	return mfs.dirs[path]
}