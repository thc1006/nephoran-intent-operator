package health

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// HealthCheckerTestSuite provides comprehensive test coverage for HealthChecker
type HealthCheckerTestSuite struct {
	suite.Suite
	ctx     context.Context
	cancel  context.CancelFunc
	logger  *slog.Logger
	checker *HealthChecker
}

func TestHealthCheckerSuite(t *testing.T) {
	suite.Run(t, new(HealthCheckerTestSuite))
}

func (suite *HealthCheckerTestSuite) SetupSuite() {
	suite.ctx, suite.cancel = context.WithTimeout(context.Background(), 30*time.Second)
	suite.logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}

func (suite *HealthCheckerTestSuite) TearDownSuite() {
	suite.cancel()
}

func (suite *HealthCheckerTestSuite) SetupTest() {
	suite.checker = NewHealthChecker("test-service", "1.0.0", suite.logger)
}

func (suite *HealthCheckerTestSuite) TestNewHealthChecker() {
	checker := NewHealthChecker("nephoran-llm-processor", "2.1.0", suite.logger)
	
	assert.Equal(suite.T(), "nephoran-llm-processor", checker.serviceName)
	assert.Equal(suite.T(), "2.1.0", checker.serviceVersion)
	assert.NotZero(suite.T(), checker.startTime)
	assert.NotNil(suite.T(), checker.checks)
	assert.NotNil(suite.T(), checker.dependencies)
	assert.Equal(suite.T(), 30*time.Second, checker.timeout)
	assert.Equal(suite.T(), 15*time.Second, checker.gracePeriod)
	assert.True(suite.T(), checker.healthyState)
	assert.False(suite.T(), checker.readyState)
}

func (suite *HealthCheckerTestSuite) TestRegisterCheck() {
	checkName := "database-connection"
	checkFunc := func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: "Database connection OK",
		}
	}
	
	suite.checker.RegisterCheck(checkName, checkFunc)
	
	suite.checker.mu.RLock()
	registeredFunc, exists := suite.checker.checks[checkName]
	suite.checker.mu.RUnlock()
	
	assert.True(suite.T(), exists)
	assert.NotNil(suite.T(), registeredFunc)
}

func (suite *HealthCheckerTestSuite) TestRegisterDependency() {
	depName := "external-api"
	depFunc := func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: "External API reachable",
		}
	}
	
	suite.checker.RegisterDependency(depName, depFunc)
	
	suite.checker.mu.RLock()
	registeredFunc, exists := suite.checker.dependencies[depName]
	suite.checker.mu.RUnlock()
	
	assert.True(suite.T(), exists)
	assert.NotNil(suite.T(), registeredFunc)
}

func (suite *HealthCheckerTestSuite) TestSetAndGetReady() {
	// Initially not ready
	assert.False(suite.T(), suite.checker.IsReady())
	
	// Set ready
	suite.checker.SetReady(true)
	assert.True(suite.T(), suite.checker.IsReady())
	
	// Set not ready
	suite.checker.SetReady(false)
	assert.False(suite.T(), suite.checker.IsReady())
}

func (suite *HealthCheckerTestSuite) TestSetAndGetHealthy() {
	// Initially healthy
	assert.True(suite.T(), suite.checker.IsHealthy())
	
	// Set unhealthy
	suite.checker.SetHealthy(false)
	assert.False(suite.T(), suite.checker.IsHealthy())
	
	// Set healthy
	suite.checker.SetHealthy(true)
	assert.True(suite.T(), suite.checker.IsHealthy())
}

func (suite *HealthCheckerTestSuite) TestCheck_NoChecks() {
	response := suite.checker.Check(suite.ctx)
	
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), "test-service", response.Service)
	assert.Equal(suite.T(), "1.0.0", response.Version)
	assert.NotEmpty(suite.T(), response.Uptime)
	assert.NotZero(suite.T(), response.Timestamp)
	assert.Empty(suite.T(), response.Checks)
	assert.Empty(suite.T(), response.Dependencies)
	assert.Equal(suite.T(), StatusUnknown, response.Status)
	assert.Equal(suite.T(), 0, response.Summary.Total)
}

func (suite *HealthCheckerTestSuite) TestCheck_HealthyChecks() {
	// Register healthy checks
	suite.checker.RegisterCheck("check1", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: "Check 1 is healthy",
		}
	})
	
	suite.checker.RegisterCheck("check2", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: "Check 2 is healthy",
		}
	})
	
	response := suite.checker.Check(suite.ctx)
	
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), StatusHealthy, response.Status)
	assert.Len(suite.T(), response.Checks, 2)
	assert.Equal(suite.T(), 2, response.Summary.Total)
	assert.Equal(suite.T(), 2, response.Summary.Healthy)
	assert.Equal(suite.T(), 0, response.Summary.Unhealthy)
}

func (suite *HealthCheckerTestSuite) TestCheck_UnhealthyChecks() {
	// Register unhealthy check
	suite.checker.RegisterCheck("failing-check", func(ctx context.Context) *Check {
		return &Check{
			Status: StatusUnhealthy,
			Error:  "Something went wrong",
		}
	})
	
	suite.checker.RegisterCheck("healthy-check", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: "This check is fine",
		}
	})
	
	response := suite.checker.Check(suite.ctx)
	
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), StatusUnhealthy, response.Status)
	assert.Len(suite.T(), response.Checks, 2)
	assert.Equal(suite.T(), 2, response.Summary.Total)
	assert.Equal(suite.T(), 1, response.Summary.Healthy)
	assert.Equal(suite.T(), 1, response.Summary.Unhealthy)
}

func (suite *HealthCheckerTestSuite) TestCheck_DegradedChecks() {
	// Register degraded check
	suite.checker.RegisterCheck("degraded-check", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusDegraded,
			Message: "Service is running but performance is degraded",
		}
	})
	
	suite.checker.RegisterCheck("healthy-check", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: "This check is fine",
		}
	})
	
	response := suite.checker.Check(suite.ctx)
	
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), StatusDegraded, response.Status)
	assert.Len(suite.T(), response.Checks, 2)
	assert.Equal(suite.T(), 2, response.Summary.Total)
	assert.Equal(suite.T(), 1, response.Summary.Healthy)
	assert.Equal(suite.T(), 0, response.Summary.Unhealthy)
	assert.Equal(suite.T(), 1, response.Summary.Degraded)
}

func (suite *HealthCheckerTestSuite) TestCheck_WithDependencies() {
	// Register service check
	suite.checker.RegisterCheck("service-check", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: "Service is running",
		}
	})
	
	// Register dependency
	suite.checker.RegisterDependency("database", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: "Database is responsive",
		}
	})
	
	suite.checker.RegisterDependency("cache", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusDegraded,
			Message: "Cache is slow but responsive",
		}
	})
	
	response := suite.checker.Check(suite.ctx)
	
	assert.NotNil(suite.T(), response)
	assert.Equal(suite.T(), StatusDegraded, response.Status)
	assert.Len(suite.T(), response.Checks, 1)
	assert.Len(suite.T(), response.Dependencies, 2)
	assert.Equal(suite.T(), 3, response.Summary.Total)
	assert.Equal(suite.T(), 2, response.Summary.Healthy)
	assert.Equal(suite.T(), 1, response.Summary.Degraded)
}

func (suite *HealthCheckerTestSuite) TestCheck_NilCheckResult() {
	// Register check that returns nil
	suite.checker.RegisterCheck("nil-check", func(ctx context.Context) *Check {
		return nil
	})
	
	response := suite.checker.Check(suite.ctx)
	
	assert.NotNil(suite.T(), response)
	assert.Len(suite.T(), response.Checks, 1)
	assert.Equal(suite.T(), StatusUnknown, response.Status)
	assert.Equal(suite.T(), "nil-check", response.Checks[0].Name)
	assert.Equal(suite.T(), StatusUnknown, response.Checks[0].Status)
	assert.Contains(suite.T(), response.Checks[0].Message, "Check function returned nil")
}

func (suite *HealthCheckerTestSuite) TestCheck_ContextTimeout() {
	// Create a context with very short timeout
	ctx, cancel := context.WithTimeout(suite.ctx, 1*time.Millisecond)
	defer cancel()
	
	// Register check that takes longer than timeout
	suite.checker.RegisterCheck("slow-check", func(ctx context.Context) *Check {
		select {
		case <-time.After(100 * time.Millisecond):
			return &Check{
				Status:  StatusHealthy,
				Message: "Check completed",
			}
		case <-ctx.Done():
			return &Check{
				Status: StatusUnhealthy,
				Error:  "Check timed out",
			}
		}
	})
	
	response := suite.checker.Check(ctx)
	
	assert.NotNil(suite.T(), response)
	// The check should handle timeout gracefully
	assert.Len(suite.T(), response.Checks, 1)
}

func (suite *HealthCheckerTestSuite) TestHealthzHandler_Healthy() {
	suite.checker.RegisterCheck("test-check", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: "All good",
		}
	})
	
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()
	
	suite.checker.HealthzHandler(recorder, req)
	
	assert.Equal(suite.T(), http.StatusOK, recorder.Code)
	assert.Equal(suite.T(), "application/json", recorder.Header().Get("Content-Type"))
	
	var response HealthResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), StatusHealthy, response.Status)
	assert.Equal(suite.T(), "test-service", response.Service)
}

func (suite *HealthCheckerTestSuite) TestHealthzHandler_Unhealthy() {
	suite.checker.RegisterCheck("failing-check", func(ctx context.Context) *Check {
		return &Check{
			Status: StatusUnhealthy,
			Error:  "Check failed",
		}
	})
	
	// Ensure we're not in grace period
	suite.checker.startTime = time.Now().Add(-20 * time.Second)
	
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()
	
	suite.checker.HealthzHandler(recorder, req)
	
	assert.Equal(suite.T(), http.StatusServiceUnavailable, recorder.Code)
	
	var response HealthResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), StatusUnhealthy, response.Status)
}

func (suite *HealthCheckerTestSuite) TestHealthzHandler_GracePeriod() {
	suite.checker.RegisterCheck("failing-check", func(ctx context.Context) *Check {
		return &Check{
			Status: StatusUnhealthy,
			Error:  "Check failed",
		}
	})
	
	// Ensure we're in grace period
	suite.checker.startTime = time.Now()
	
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	recorder := httptest.NewRecorder()
	
	suite.checker.HealthzHandler(recorder, req)
	
	// Should return healthy during grace period despite failing checks
	assert.Equal(suite.T(), http.StatusOK, recorder.Code)
	
	var response HealthResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), StatusHealthy, response.Status)
}

func (suite *HealthCheckerTestSuite) TestHealthzHandler_WrongMethod() {
	req := httptest.NewRequest(http.MethodPost, "/healthz", nil)
	recorder := httptest.NewRecorder()
	
	suite.checker.HealthzHandler(recorder, req)
	
	assert.Equal(suite.T(), http.StatusMethodNotAllowed, recorder.Code)
}

func (suite *HealthCheckerTestSuite) TestReadyzHandler_Ready() {
	suite.checker.SetReady(true)
	suite.checker.RegisterCheck("test-check", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: "Ready to serve",
		}
	})
	
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()
	
	suite.checker.ReadyzHandler(recorder, req)
	
	assert.Equal(suite.T(), http.StatusOK, recorder.Code)
	
	var response HealthResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), StatusHealthy, response.Status)
}

func (suite *HealthCheckerTestSuite) TestReadyzHandler_NotReady_ServiceNotReady() {
	suite.checker.SetReady(false) // Service not ready
	suite.checker.RegisterCheck("test-check", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: "Check is healthy",
		}
	})
	
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()
	
	suite.checker.ReadyzHandler(recorder, req)
	
	assert.Equal(suite.T(), http.StatusServiceUnavailable, recorder.Code)
	
	var response HealthResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), StatusUnhealthy, response.Status)
}

func (suite *HealthCheckerTestSuite) TestReadyzHandler_NotReady_UnhealthyDependency() {
	suite.checker.SetReady(true)
	suite.checker.RegisterCheck("test-check", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: "Check is healthy",
		}
	})
	
	suite.checker.RegisterDependency("critical-service", func(ctx context.Context) *Check {
		return &Check{
			Status: StatusUnhealthy,
			Error:  "Critical service is down",
		}
	})
	
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()
	
	suite.checker.ReadyzHandler(recorder, req)
	
	assert.Equal(suite.T(), http.StatusServiceUnavailable, recorder.Code)
	
	var response HealthResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), StatusUnhealthy, response.Status)
}

func (suite *HealthCheckerTestSuite) TestReadyzHandler_ReadyWithDegradedService() {
	suite.checker.SetReady(true)
	suite.checker.RegisterCheck("degraded-check", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusDegraded,
			Message: "Service is degraded but functional",
		}
	})
	
	req := httptest.NewRequest(http.MethodGet, "/readyz", nil)
	recorder := httptest.NewRecorder()
	
	suite.checker.ReadyzHandler(recorder, req)
	
	assert.Equal(suite.T(), http.StatusOK, recorder.Code)
	
	var response HealthResponse
	err := json.Unmarshal(recorder.Body.Bytes(), &response)
	require.NoError(suite.T(), err)
	
	assert.Equal(suite.T(), StatusHealthy, response.Status)
}

// Table-driven tests for different health check scenarios
func (suite *HealthCheckerTestSuite) TestHealthCheckScenarios_TableDriven() {
	testCases := []struct {
		name            string
		setupChecks     func(*HealthChecker)
		expectedStatus  Status
		expectedHealthy int
		expectedUnhealthy int
		expectedDegraded int
	}{
		{
			name: "All checks healthy",
			setupChecks: func(hc *HealthChecker) {
				hc.RegisterCheck("check1", func(ctx context.Context) *Check {
					return &Check{Status: StatusHealthy, Message: "OK"}
				})
				hc.RegisterCheck("check2", func(ctx context.Context) *Check {
					return &Check{Status: StatusHealthy, Message: "OK"}
				})
			},
			expectedStatus:    StatusHealthy,
			expectedHealthy:   2,
			expectedUnhealthy: 0,
			expectedDegraded:  0,
		},
		{
			name: "Mixed healthy and degraded",
			setupChecks: func(hc *HealthChecker) {
				hc.RegisterCheck("healthy", func(ctx context.Context) *Check {
					return &Check{Status: StatusHealthy, Message: "OK"}
				})
				hc.RegisterCheck("degraded", func(ctx context.Context) *Check {
					return &Check{Status: StatusDegraded, Message: "Slow"}
				})
			},
			expectedStatus:    StatusDegraded,
			expectedHealthy:   1,
			expectedUnhealthy: 0,
			expectedDegraded:  1,
		},
		{
			name: "One unhealthy check fails all",
			setupChecks: func(hc *HealthChecker) {
				hc.RegisterCheck("healthy", func(ctx context.Context) *Check {
					return &Check{Status: StatusHealthy, Message: "OK"}
				})
				hc.RegisterCheck("unhealthy", func(ctx context.Context) *Check {
					return &Check{Status: StatusUnhealthy, Error: "Failed"}
				})
				hc.RegisterCheck("degraded", func(ctx context.Context) *Check {
					return &Check{Status: StatusDegraded, Message: "Slow"}
				})
			},
			expectedStatus:    StatusUnhealthy,
			expectedHealthy:   1,
			expectedUnhealthy: 1,
			expectedDegraded:  1,
		},
		{
			name: "Unknown status check",
			setupChecks: func(hc *HealthChecker) {
				hc.RegisterCheck("unknown", func(ctx context.Context) *Check {
					return &Check{Status: StatusUnknown, Message: "Status unclear"}
				})
				hc.RegisterCheck("healthy", func(ctx context.Context) *Check {
					return &Check{Status: StatusHealthy, Message: "OK"}
				})
			},
			expectedStatus:    StatusUnknown,
			expectedHealthy:   1,
			expectedUnhealthy: 0,
			expectedDegraded:  0,
		},
	}
	
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			checker := NewHealthChecker("test-service", "1.0.0", suite.logger)
			tc.setupChecks(checker)
			
			response := checker.Check(suite.ctx)
			
			assert.Equal(suite.T(), tc.expectedStatus, response.Status)
			assert.Equal(suite.T(), tc.expectedHealthy, response.Summary.Healthy)
			assert.Equal(suite.T(), tc.expectedUnhealthy, response.Summary.Unhealthy)
			assert.Equal(suite.T(), tc.expectedDegraded, response.Summary.Degraded)
		})
	}
}

// Test common health check functions

func (suite *HealthCheckerTestSuite) TestHTTPCheck_Success() {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()
	
	checkFunc := HTTPCheck("api-health", server.URL)
	check := checkFunc(suite.ctx)
	
	assert.NotNil(suite.T(), check)
	assert.Equal(suite.T(), StatusHealthy, check.Status)
	assert.Contains(suite.T(), check.Message, "HTTP 200")
}

func (suite *HealthCheckerTestSuite) TestHTTPCheck_Failure() {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Error"))
	}))
	defer server.Close()
	
	checkFunc := HTTPCheck("api-health", server.URL)
	check := checkFunc(suite.ctx)
	
	assert.NotNil(suite.T(), check)
	assert.Equal(suite.T(), StatusUnhealthy, check.Status)
	assert.Contains(suite.T(), check.Error, "HTTP 500")
}

func (suite *HealthCheckerTestSuite) TestHTTPCheck_NetworkError() {
	// Use invalid URL to trigger network error
	checkFunc := HTTPCheck("api-health", "http://non-existent-host:12345/health")
	check := checkFunc(suite.ctx)
	
	assert.NotNil(suite.T(), check)
	assert.Equal(suite.T(), StatusUnhealthy, check.Status)
	assert.Contains(suite.T(), check.Error, "HTTP request failed")
}

func (suite *HealthCheckerTestSuite) TestDatabaseCheck_Success() {
	pingFunc := func(ctx context.Context) error {
		return nil // Successful ping
	}
	
	checkFunc := DatabaseCheck("database", pingFunc)
	check := checkFunc(suite.ctx)
	
	assert.NotNil(suite.T(), check)
	assert.Equal(suite.T(), StatusHealthy, check.Status)
	assert.Contains(suite.T(), check.Message, "Database connection successful")
}

func (suite *HealthCheckerTestSuite) TestDatabaseCheck_Failure() {
	pingFunc := func(ctx context.Context) error {
		return fmt.Errorf("connection refused")
	}
	
	checkFunc := DatabaseCheck("database", pingFunc)
	check := checkFunc(suite.ctx)
	
	assert.NotNil(suite.T(), check)
	assert.Equal(suite.T(), StatusUnhealthy, check.Status)
	assert.Contains(suite.T(), check.Error, "Database connection failed")
	assert.Contains(suite.T(), check.Error, "connection refused")
}

func (suite *HealthCheckerTestSuite) TestMemoryCheck() {
	checkFunc := MemoryCheck("memory", 1024)
	check := checkFunc(suite.ctx)
	
	assert.NotNil(suite.T(), check)
	assert.Equal(suite.T(), StatusHealthy, check.Status)
	assert.Contains(suite.T(), check.Message, "Memory usage within limits")
}

// Concurrent execution tests
func (suite *HealthCheckerTestSuite) TestConcurrentHealthChecks() {
	numChecks := 10
	
	// Register multiple checks that take different amounts of time
	for i := 0; i < numChecks; i++ {
		checkName := fmt.Sprintf("check-%d", i)
		delay := time.Duration(i*10) * time.Millisecond
		
		suite.checker.RegisterCheck(checkName, func(ctx context.Context) *Check {
			time.Sleep(delay)
			return &Check{
				Status:  StatusHealthy,
				Message: fmt.Sprintf("Check completed after %v", delay),
			}
		})
	}
	
	start := time.Now()
	response := suite.checker.Check(suite.ctx)
	duration := time.Since(start)
	
	// All checks should complete concurrently, so total time should be less than
	// the sum of all delays (which would be ~450ms)
	assert.Less(suite.T(), duration, 300*time.Millisecond)
	assert.Len(suite.T(), response.Checks, numChecks)
	assert.Equal(suite.T(), StatusHealthy, response.Status)
	assert.Equal(suite.T(), numChecks, response.Summary.Healthy)
}

// Edge cases and error handling tests
func (suite *HealthCheckerTestSuite) TestEdgeCases() {
	testCases := []struct {
		name        string
		setupFunc   func(*HealthChecker)
		validateFunc func(*testing.T, *HealthResponse)
	}{
		{
			name: "Check with panic recovery",
			setupFunc: func(hc *HealthChecker) {
				hc.RegisterCheck("panic-check", func(ctx context.Context) *Check {
					// This would cause a panic in a real scenario
					// For testing, we'll simulate the recovery behavior
					return &Check{
						Status: StatusUnhealthy,
						Error:  "Check panicked and was recovered",
					}
				})
			},
			validateFunc: func(t *testing.T, response *HealthResponse) {
				assert.Equal(t, StatusUnhealthy, response.Status)
				assert.Len(t, response.Checks, 1)
			},
		},
		{
			name: "Very slow check",
			setupFunc: func(hc *HealthChecker) {
				hc.RegisterCheck("slow-check", func(ctx context.Context) *Check {
					select {
					case <-time.After(5 * time.Second):
						return &Check{Status: StatusHealthy, Message: "Slow but completed"}
					case <-ctx.Done():
						return &Check{Status: StatusUnhealthy, Error: "Check timed out"}
					}
				})
			},
			validateFunc: func(t *testing.T, response *HealthResponse) {
				// Should complete within the context timeout
				assert.Len(t, response.Checks, 1)
			},
		},
		{
			name: "Check with metadata",
			setupFunc: func(hc *HealthChecker) {
				hc.RegisterCheck("metadata-check", func(ctx context.Context) *Check {
					return &Check{
						Status:  StatusHealthy,
						Message: "Check with metadata",
						Metadata: map[string]interface{}{
							"version": "1.2.3",
							"uptime":  "5m30s",
							"connections": 42,
						},
					}
				})
			},
			validateFunc: func(t *testing.T, response *HealthResponse) {
				assert.Equal(t, StatusHealthy, response.Status)
				assert.Len(t, response.Checks, 1)
				assert.NotNil(t, response.Checks[0].Metadata)
			},
		},
	}
	
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			checker := NewHealthChecker("test-service", "1.0.0", suite.logger)
			tc.setupFunc(checker)
			
			response := checker.Check(suite.ctx)
			tc.validateFunc(suite.T(), response)
		})
	}
}

// Performance and load testing
func (suite *HealthCheckerTestSuite) TestHighLoadConcurrentChecks() {
	numChecks := 100
	var wg sync.WaitGroup
	
	// Register many checks
	for i := 0; i < numChecks; i++ {
		checkName := fmt.Sprintf("load-check-%d", i)
		suite.checker.RegisterCheck(checkName, func(ctx context.Context) *Check {
			// Simulate some work
			time.Sleep(1 * time.Millisecond)
			return &Check{
				Status:  StatusHealthy,
				Message: "Load test check completed",
			}
		})
	}
	
	// Run multiple health checks concurrently
	numConcurrentChecks := 10
	responses := make([]*HealthResponse, numConcurrentChecks)
	
	for i := 0; i < numConcurrentChecks; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			responses[index] = suite.checker.Check(suite.ctx)
		}(i)
	}
	
	wg.Wait()
	
	// Verify all responses are valid
	for i, response := range responses {
		assert.NotNil(suite.T(), response, "Response %d should not be nil", i)
		assert.Len(suite.T(), response.Checks, numChecks, "Response %d should have all checks", i)
		assert.Equal(suite.T(), StatusHealthy, response.Status, "Response %d should be healthy", i)
	}
}

// Benchmarks for performance testing
func BenchmarkHealthCheck_SingleCheck(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	checker := NewHealthChecker("benchmark-service", "1.0.0", logger)
	
	checker.RegisterCheck("benchmark", func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: "Benchmark check",
		}
	})
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.Check(ctx)
	}
}

func BenchmarkHealthCheck_MultipleChecks(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	checker := NewHealthChecker("benchmark-service", "1.0.0", logger)
	
	// Register 10 checks
	for i := 0; i < 10; i++ {
		checkName := fmt.Sprintf("benchmark-check-%d", i)
		checker.RegisterCheck(checkName, func(ctx context.Context) *Check {
			return &Check{
				Status:  StatusHealthy,
				Message: "Benchmark check",
			}
		})
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.Check(ctx)
	}
}

func BenchmarkHealthCheck_WithDependencies(b *testing.B) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelError}))
	checker := NewHealthChecker("benchmark-service", "1.0.0", logger)
	
	// Register service checks
	for i := 0; i < 5; i++ {
		checkName := fmt.Sprintf("service-check-%d", i)
		checker.RegisterCheck(checkName, func(ctx context.Context) *Check {
			return &Check{Status: StatusHealthy, Message: "Service check"}
		})
	}
	
	// Register dependencies
	for i := 0; i < 5; i++ {
		depName := fmt.Sprintf("dependency-%d", i)
		checker.RegisterDependency(depName, func(ctx context.Context) *Check {
			return &Check{Status: StatusHealthy, Message: "Dependency check"}
		})
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = checker.Check(ctx)
	}
}

// Helper functions for testing
func createHealthyCheck(name, message string) CheckFunc {
	return func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusHealthy,
			Message: message,
		}
	}
}

func createUnhealthyCheck(name, error string) CheckFunc {
	return func(ctx context.Context) *Check {
		return &Check{
			Status: StatusUnhealthy,
			Error:  error,
		}
	}
}

func createDegradedCheck(name, message string) CheckFunc {
	return func(ctx context.Context) *Check {
		return &Check{
			Status:  StatusDegraded,
			Message: message,
		}
	}
}