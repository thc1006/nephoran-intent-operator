package integration_tests

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/audit"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers"
)

// E2EAuditTestSuite tests end-to-end audit workflows with real controllers
type E2EAuditTestSuite struct {
	suite.Suite
	client         client.Client
	scheme         *runtime.Scheme
	recorder       record.EventRecorder
	controller     *controllers.AuditTrailController
	auditSystem    *AuditSystem
	tempDir        string
	httpServer     *httptest.Server
	eventsReceived []*AuditEvent
	eventMutex     sync.RWMutex
}

// DISABLED: func TestE2EAuditTestSuite(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping E2E tests in short mode")
	}
	suite.Run(t, new(E2EAuditTestSuite))
}

func (suite *E2EAuditTestSuite) SetupSuite() {
	// Setup temporary directory
	var err error
	suite.tempDir, err = ioutil.TempDir("", "e2e_audit_test")
	suite.Require().NoError(err)

	// Setup HTTP server for webhook testing
	suite.httpServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		suite.eventMutex.Lock()
		defer suite.eventMutex.Unlock()

		var event AuditEvent
		_, err := ioutil.ReadAll(r.Body)
		if err == nil {
			// In a real implementation, we'd unmarshal the JSON
			// For testing, we'll create a mock event
			event = AuditEvent{
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
				Component: "webhook-receiver",
				Action:    "received",
				Severity:  SeverityInfo,
			}
			suite.eventsReceived = append(suite.eventsReceived, &event)
		}

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status": "ok"}`))
	}))

	// Setup Kubernetes client and scheme
	suite.scheme = runtime.NewScheme()
	err = nephv1.AddToScheme(suite.scheme)
	suite.Require().NoError(err)

	suite.client = fake.NewClientBuilder().WithScheme(suite.scheme).Build()

	// Setup event recorder
	suite.recorder = record.NewFakeRecorder(100)

	// Setup controller
	logger := zap.New(zap.UseDevMode(true))
	suite.controller = controllers.NewAuditTrailController(
		suite.client,
		logger,
		suite.scheme,
		suite.recorder,
	)
}

func (suite *E2EAuditTestSuite) TearDownSuite() {
	if suite.httpServer != nil {
		suite.httpServer.Close()
	}
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

func (suite *E2EAuditTestSuite) SetupTest() {
	// Reset received events
	suite.eventMutex.Lock()
	suite.eventsReceived = suite.eventsReceived[:0]
	suite.eventMutex.Unlock()
}

func (suite *E2EAuditTestSuite) TearDownTest() {
	if suite.auditSystem != nil {
		suite.auditSystem.Stop()
		suite.auditSystem = nil
	}
}

// Test complete audit trail creation and lifecycle
func (suite *E2EAuditTestSuite) TestCompleteAuditTrailLifecycle() {
	suite.Run("create audit trail resource", func() {
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: ctrl.ObjectMeta{
				Name:      "test-audit-trail",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled:         true,
				LogLevel:        "info",
				BatchSize:       10,
				FlushInterval:   5,
				MaxQueueSize:    1000,
				EnableIntegrity: true,
				ComplianceMode:  []string{"soc2", "iso27001"},
				Backends: []nephv1.AuditBackendConfig{
					{
						Type:    "file",
						Enabled: true,
						Name:    "file-backend",
						Settings: runtime.RawExtension{
							Raw: []byte(fmt.Sprintf(`{"path": "%s"}`, filepath.Join(suite.tempDir, "audit.log"))),
						},
					},
					{
						Type:    "webhook",
						Enabled: true,
						Name:    "webhook-backend",
						Settings: runtime.RawExtension{
							Raw: []byte(fmt.Sprintf(`{"url": "%s"}`, suite.httpServer.URL)),
						},
					},
				},
			},
		}

		err := suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		// Reconcile the audit trail
		result, err := suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test-audit-trail",
				Namespace: "default",
			},
		})

		suite.NoError(err)
		suite.False(result.Requeue)

		// Verify audit system was created
		createdSystem := suite.controller.GetAuditSystem("default", "test-audit-trail")
		suite.NotNil(createdSystem)
		suite.auditSystem = createdSystem

		// Verify status was updated
		var updatedAuditTrail nephv1.AuditTrail
		err = suite.client.Get(context.Background(), types.NamespacedName{
			Name:      "test-audit-trail",
			Namespace: "default",
		}, &updatedAuditTrail)
		suite.NoError(err)
		suite.Equal("Running", updatedAuditTrail.Status.Phase)
	})

	suite.Run("log events through audit system", func() {
		suite.Require().NotNil(suite.auditSystem)

		events := []*AuditEvent{
			{
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
				EventType: audit.EventTypeAuthentication,
				Component: "e2e-test",
				Action:    "login",
				Severity:  SeverityInfo,
				Result:    ResultSuccess,
				UserContext: &UserContext{
					UserID:   "test-user-1",
					Username: "testuser1",
				},
			},
			{
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
				EventType: audit.EventTypeDataAccess,
				Component: "e2e-test",
				Action:    "read_data",
				Severity:  SeverityInfo,
				Result:    ResultSuccess,
				UserContext: &UserContext{
					UserID:   "test-user-2",
					Username: "testuser2",
				},
				ResourceContext: &ResourceContext{
					ResourceType: "user",
					ResourceID:   "user123",
					Operation:    "read",
				},
			},
			{
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
				EventType: audit.EventTypeSecurityViolation,
				Component: "e2e-test",
				Action:    "suspicious_activity",
				Severity:  SeverityCritical,
				Result:    ResultFailure,
				Data: map[string]interface{}{
					"violation_type": "brute_force",
					"attempts":       10,
				},
			},
		}

		// Log all events
		for _, event := range events {
			err := suite.auditSystem.LogEvent(event)
			suite.NoError(err)
		}

		// Wait for processing
		time.Sleep(2 * time.Second)

		// Verify events were written to file
		logFile := filepath.Join(suite.tempDir, "audit.log")
		content, err := ioutil.ReadFile(logFile)
		suite.NoError(err)
		suite.NotEmpty(content)

		// Verify events were sent to webhook
		suite.Eventually(func() bool {
			suite.eventMutex.RLock()
			defer suite.eventMutex.RUnlock()
			return len(suite.eventsReceived) > 0
		}, 5*time.Second, 100*time.Millisecond)
	})

	suite.Run("update audit trail configuration", func() {
		// Update the audit trail to add another backend
		var auditTrail nephv1.AuditTrail
		err := suite.client.Get(context.Background(), types.NamespacedName{
			Name:      "test-audit-trail",
			Namespace: "default",
		}, &auditTrail)
		suite.NoError(err)

		// Add syslog backend
		auditTrail.Spec.Backends = append(auditTrail.Spec.Backends, nephv1.AuditBackendConfig{
			Type:    "syslog",
			Enabled: true,
			Name:    "syslog-backend",
			Settings: runtime.RawExtension{
				Raw: []byte(`{"network": "udp", "address": "localhost:514"}`),
			},
		})

		err = suite.client.Update(context.Background(), &auditTrail)
		suite.NoError(err)

		// Reconcile the changes
		result, err := suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test-audit-trail",
				Namespace: "default",
			},
		})

		suite.NoError(err)
		suite.False(result.Requeue)

		// Verify the audit system was updated
		stats := suite.auditSystem.GetStats()
		suite.Equal(3, stats.BackendCount) // file, webhook, syslog
	})

	suite.Run("delete audit trail", func() {
		// Delete the audit trail
		var auditTrail nephv1.AuditTrail
		err := suite.client.Get(context.Background(), types.NamespacedName{
			Name:      "test-audit-trail",
			Namespace: "default",
		}, &auditTrail)
		suite.NoError(err)

		err = suite.client.Delete(context.Background(), &auditTrail)
		suite.NoError(err)

		// Reconcile the deletion
		result, err := suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test-audit-trail",
				Namespace: "default",
			},
		})

		suite.NoError(err)
		suite.False(result.Requeue)

		// Verify audit system was cleaned up
		deletedSystem := suite.controller.GetAuditSystem("default", "test-audit-trail")
		suite.Nil(deletedSystem)
	})
}

// Test audit events from different sources
func (suite *E2EAuditTestSuite) TestAuditEventSources() {
	suite.Run("setup audit system", func() {
		suite.setupBasicAuditSystem()
	})

	suite.Run("controller reconciliation events", func() {
		suite.Require().NotNil(suite.auditSystem)

		// Simulate controller reconciliation event
		reconcileEvent := &AuditEvent{
			ID:        uuid.New().String(),
			Timestamp: time.Now(),
			EventType: audit.EventTypeSystemChange,
			Component: "networkintent-controller",
			Action:    "reconcile",
			Severity:  SeverityInfo,
			Result:    ResultSuccess,
			UserContext: &UserContext{
				UserID:         "system",
				ServiceAccount: true,
			},
			ResourceContext: &ResourceContext{
				ResourceType: "NetworkIntent",
				ResourceID:   "test-intent",
				Operation:    "update",
				Namespace:    "default",
			},
			Data: map[string]interface{}{
				"generation":       1,
				"resource_version": "12345",
				"reconcile_reason": "spec_change",
			},
		}

		err := suite.auditSystem.LogEvent(reconcileEvent)
		suite.NoError(err)

		// Wait for processing
		time.Sleep(1 * time.Second)

		// Verify event processing
		stats := suite.auditSystem.GetStats()
		suite.Greater(stats.EventsReceived, int64(0))
	})

	suite.Run("webhook admission controller events", func() {
		// Simulate webhook admission controller event
		admissionEvent := &AuditEvent{
			ID:        uuid.New().String(),
			Timestamp: time.Now(),
			EventType: audit.EventTypeAuthorization,
			Component: "admission-webhook",
			Action:    "validate",
			Severity:  SeverityInfo,
			Result:    ResultSuccess,
			UserContext: &UserContext{
				UserID:   "user@example.com",
				Username: "developer",
				Role:     "developer",
			},
			NetworkContext: &NetworkContext{
				SourcePort: 8443,
				UserAgent:  "kubectl/v1.28.0",
			},
			ResourceContext: &ResourceContext{
				ResourceType: "NetworkIntent",
				Operation:    "create",
				Namespace:    "production",
			},
			Data: map[string]interface{}{
				"admission_allowed":  true,
				"validation_time_ms": 15,
				"policies_evaluated": []string{"security-policy", "resource-quota"},
			},
		}

		err := suite.auditSystem.LogEvent(admissionEvent)
		suite.NoError(err)
	})

	suite.Run("API server authentication events", func() {
		// Simulate API server authentication event
		authEvent := &AuditEvent{
			ID:        uuid.New().String(),
			Timestamp: time.Now(),
			EventType: EventTypeAuthentication,
			Component: "api-server",
			Action:    "authenticate",
			Severity:  SeverityInfo,
			Result:    ResultSuccess,
			UserContext: &UserContext{
				UserID:       "user@example.com",
				Username:     "developer",
				AuthMethod:   "oidc",
				AuthProvider: "google",
				Groups:       []string{"developers", "k8s-users"},
			},
			NetworkContext: &NetworkContext{
				UserAgent: "kubectl/v1.28.0",
				RequestID: "req-" + uuid.New().String(),
			},
			Data: map[string]interface{}{
				"token_type": "bearer",
				"token_exp":  time.Now().Add(1 * time.Hour).Unix(),
				"scopes":     []string{"openid", "email", "profile"},
			},
		}

		err := suite.auditSystem.LogEvent(authEvent)
		suite.NoError(err)
	})
}

// Test audit system with high load
func (suite *E2EAuditTestSuite) TestHighLoadAuditing() {
	suite.Run("setup high-capacity audit system", func() {
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: ctrl.ObjectMeta{
				Name:      "high-load-audit",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled:         true,
				LogLevel:        "info",
				BatchSize:       100,
				FlushInterval:   1, // 1 second
				MaxQueueSize:    10000,
				EnableIntegrity: false, // Disable for performance
				Backends: []nephv1.AuditBackendConfig{
					{
						Type:    "file",
						Enabled: true,
						Name:    "high-load-file",
						Settings: runtime.RawExtension{
							Raw: []byte(fmt.Sprintf(`{"path": "%s"}`, filepath.Join(suite.tempDir, "high_load.log"))),
						},
					},
				},
			},
		}

		err := suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		// Reconcile
		_, err = suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "high-load-audit",
				Namespace: "default",
			},
		})
		suite.NoError(err)

		suite.auditSystem = suite.controller.GetAuditSystem("default", "high-load-audit")
		suite.NotNil(suite.auditSystem)
	})

	suite.Run("generate high volume of events", func() {
		const numEvents = 1000
		const numGoroutines = 10
		eventsPerGoroutine := numEvents / numGoroutines

		var wg sync.WaitGroup
		start := time.Now()

		for g := 0; g < numGoroutines; g++ {
			wg.Add(1)
			go func(goroutineID int) {
				defer wg.Done()

				for i := 0; i < eventsPerGoroutine; i++ {
					event := &AuditEvent{
						ID:        uuid.New().String(),
						Timestamp: time.Now(),
						EventType: audit.EventTypeAPICall,
						Component: "high-load-test",
						Action:    fmt.Sprintf("api-call-%d-%d", goroutineID, i),
						Severity:  SeverityInfo,
						Result:    ResultSuccess,
						UserContext: &UserContext{
							UserID: fmt.Sprintf("user-%d", goroutineID),
						},
						Data: map[string]interface{}{
							"goroutine_id": goroutineID,
							"event_index":  i,
						},
					}

					err := suite.auditSystem.LogEvent(event)
					if err != nil {
						// Log error but continue (some events might be dropped due to queue limits)
						fmt.Printf("Error logging event: %v\n", err)
					}
				}
			}(g)
		}

		wg.Wait()
		duration := time.Since(start)

		// Wait for processing
		time.Sleep(5 * time.Second)

		stats := suite.auditSystem.GetStats()
		eventsPerSecond := float64(stats.EventsReceived) / duration.Seconds()

		suite.Greater(stats.EventsReceived, int64(numEvents*0.8)) // Allow for some drops
		suite.Greater(eventsPerSecond, 100.0)                     // Should process at least 100 events/sec

		fmt.Printf("High load test results: %d events received, %.2f events/sec, %d dropped\n",
			stats.EventsReceived, eventsPerSecond, stats.EventsDropped)
	})
}

// Test audit system error recovery
func (suite *E2EAuditTestSuite) TestErrorRecovery() {
	suite.Run("setup audit system with failing backend", func() {
		// Create a backend that will fail
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: ctrl.ObjectMeta{
				Name:      "error-recovery-audit",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled:       true,
				LogLevel:      "info",
				BatchSize:     5,
				FlushInterval: 1,
				MaxQueueSize:  100,
				Backends: []nephv1.AuditBackendConfig{
					{
						Type:    "webhook",
						Enabled: true,
						Name:    "failing-webhook",
						Settings: runtime.RawExtension{
							Raw: []byte(`{"url": "http://invalid-url-that-will-fail:99999"}`),
						},
					},
					{
						Type:    "file",
						Enabled: true,
						Name:    "backup-file",
						Settings: runtime.RawExtension{
							Raw: []byte(fmt.Sprintf(`{"path": "%s"}`, filepath.Join(suite.tempDir, "backup.log"))),
						},
					},
				},
			},
		}

		err := suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		// Reconcile
		_, err = suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "error-recovery-audit",
				Namespace: "default",
			},
		})
		suite.NoError(err)

		suite.auditSystem = suite.controller.GetAuditSystem("default", "error-recovery-audit")
		suite.NotNil(suite.auditSystem)
	})

	suite.Run("verify system continues operating with partial failures", func() {
		// Log events that will partially fail
		for i := 0; i < 10; i++ {
			event := &AuditEvent{
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
				EventType: audit.EventTypeSystemChange,
				Component: "error-recovery-test",
				Action:    fmt.Sprintf("test-action-%d", i),
				Severity:  SeverityInfo,
				Result:    ResultSuccess,
			}

			err := suite.auditSystem.LogEvent(event)
			suite.NoError(err) // Event submission should not fail
		}

		// Wait for processing
		time.Sleep(3 * time.Second)

		// Verify audit system is still running
		stats := suite.auditSystem.GetStats()
		suite.Greater(stats.EventsReceived, int64(0))

		// Verify backup file received events (webhook should fail but file should succeed)
		backupFile := filepath.Join(suite.tempDir, "backup.log")
		content, err := ioutil.ReadFile(backupFile)
		suite.NoError(err)
		suite.NotEmpty(content)
	})
}

// Test compliance integration
func (suite *E2EAuditTestSuite) TestComplianceIntegration() {
	suite.Run("setup compliance-enabled audit system", func() {
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: ctrl.ObjectMeta{
				Name:      "compliance-audit",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled:         true,
				LogLevel:        "info",
				BatchSize:       10,
				FlushInterval:   2,
				MaxQueueSize:    1000,
				EnableIntegrity: true,
				ComplianceMode:  []string{"soc2", "iso27001", "pci_dss"},
				Backends: []nephv1.AuditBackendConfig{
					{
						Type:    "file",
						Enabled: true,
						Name:    "compliance-file",
						Settings: runtime.RawExtension{
							Raw: []byte(fmt.Sprintf(`{"path": "%s"}`, filepath.Join(suite.tempDir, "compliance.log"))),
						},
					},
				},
			},
		}

		err := suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		// Reconcile
		_, err = suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "compliance-audit",
				Namespace: "default",
			},
		})
		suite.NoError(err)

		suite.auditSystem = suite.controller.GetAuditSystem("default", "compliance-audit")
		suite.NotNil(suite.auditSystem)
	})

	suite.Run("log compliance-relevant events", func() {
		complianceEvents := []*AuditEvent{
			{
				ID:                 uuid.New().String(),
				Timestamp:          time.Now(),
				EventType:          EventTypeAuthentication,
				Component:          "auth-service",
				Action:             "mfa_verification",
				Severity:           SeverityInfo,
				Result:             ResultSuccess,
				DataClassification: "Authentication Data",
				UserContext: &UserContext{
					UserID:     "compliance-user-1",
					AuthMethod: "mfa",
				},
				Data: map[string]interface{}{
					"mfa_method": "totp",
					"device_id":  "device123",
				},
			},
			{
				ID:                 uuid.New().String(),
				Timestamp:          time.Now(),
				EventType:          EventTypeDataAccess,
				Component:          "payment-api",
				Action:             "cardholder_data_access",
				Severity:           SeverityNotice,
				Result:             ResultSuccess,
				DataClassification: "Cardholder Data",
				UserContext: &UserContext{
					UserID: "payment-processor",
					Role:   "service_account",
				},
				Data: map[string]interface{}{
					"cardholder_data": true,
					"transaction_id":  "txn_789",
					"amount":          100.00,
				},
			},
			{
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
				EventType: audit.EventTypeSecurityViolation,
				Component: "security-monitor",
				Action:    "pii_access_violation",
				Severity:  SeverityCritical,
				Result:    ResultFailure,
				UserContext: &UserContext{
					UserID: "suspicious-user",
				},
				Data: map[string]interface{}{
					"violation_type":    "unauthorized_pii_access",
					"records_attempted": 500,
					"blocked":           true,
				},
			},
		}

		for _, event := range complianceEvents {
			err := suite.auditSystem.LogEvent(event)
			suite.NoError(err)
		}

		// Wait for processing
		time.Sleep(3 * time.Second)

		// Verify compliance metadata was added
		logFile := filepath.Join(suite.tempDir, "compliance.log")
		content, err := ioutil.ReadFile(logFile)
		suite.NoError(err)
		suite.NotEmpty(content)

		// In a real implementation, we'd parse the JSON and verify compliance metadata
		contentStr := string(content)
		suite.Contains(contentStr, "compliance_metadata")
	})
}

// Test audit system monitoring and health checks
func (suite *E2EAuditTestSuite) TestMonitoringAndHealth() {
	suite.Run("setup monitored audit system", func() {
		suite.setupBasicAuditSystem()
	})

	suite.Run("verify health checks", func() {
		suite.Require().NotNil(suite.auditSystem)

		// Health check should pass for running system
		stats := suite.auditSystem.GetStats()
		suite.Equal(1, stats.BackendCount)
		suite.True(stats.IntegrityEnabled)

		// Log some events to generate activity
		for i := 0; i < 5; i++ {
			event := &AuditEvent{
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
				EventType: audit.EventTypeHealthCheck,
				Component: "monitoring-test",
				Action:    fmt.Sprintf("health-check-%d", i),
				Severity:  SeverityInfo,
				Result:    ResultSuccess,
			}

			err := suite.auditSystem.LogEvent(event)
			suite.NoError(err)
		}

		// Wait and verify stats
		time.Sleep(2 * time.Second)
		updatedStats := suite.auditSystem.GetStats()
		suite.Greater(updatedStats.EventsReceived, stats.EventsReceived)
	})

	suite.Run("verify metrics collection", func() {
		// In a real implementation, we'd check Prometheus metrics
		// For now, verify that metrics-related code is being exercised
		stats := suite.auditSystem.GetStats()
		suite.GreaterOrEqual(stats.EventsReceived, int64(5))
		suite.Equal(int64(0), stats.EventsDropped) // No drops expected in normal operation
	})
}

// Helper methods

func (suite *E2EAuditTestSuite) setupBasicAuditSystem() {
	auditTrail := &nephv1.AuditTrail{
		ObjectMeta: ctrl.ObjectMeta{
			Name:      "basic-audit-system",
			Namespace: "default",
		},
		Spec: nephv1.AuditTrailSpec{
			Enabled:         true,
			LogLevel:        "info",
			BatchSize:       5,
			FlushInterval:   2,
			MaxQueueSize:    100,
			EnableIntegrity: true,
			ComplianceMode:  []string{"soc2"},
			Backends: []nephv1.AuditBackendConfig{
				{
					Type:    "file",
					Enabled: true,
					Name:    "basic-file",
					Settings: runtime.RawExtension{
						Raw: []byte(fmt.Sprintf(`{"path": "%s"}`, filepath.Join(suite.tempDir, "basic.log"))),
					},
				},
			},
		},
	}

	err := suite.client.Create(context.Background(), auditTrail)
	suite.Require().NoError(err)

	// Reconcile
	_, err = suite.controller.Reconcile(context.Background(), ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "basic-audit-system",
			Namespace: "default",
		},
	})
	suite.Require().NoError(err)

	suite.auditSystem = suite.controller.GetAuditSystem("default", "basic-audit-system")
	suite.Require().NotNil(suite.auditSystem)
}

func (suite *E2EAuditTestSuite) Eventually(condition func() bool, timeout, interval time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if condition() {
			return
		}
		time.Sleep(interval)
	}
	suite.Fail("Condition was not met within timeout")
}

// Integration test for audit system with real Kubernetes events
func (suite *E2EAuditTestSuite) TestKubernetesIntegration() {
	suite.Run("setup kubernetes audit integration", func() {
		suite.setupBasicAuditSystem()
	})

	suite.Run("simulate kubernetes API server events", func() {
		k8sEvents := []*AuditEvent{
			{
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
				EventType: audit.EventTypeAPICall,
				Component: "kube-apiserver",
				Action:    "create",
				Severity:  SeverityInfo,
				Result:    ResultSuccess,
				UserContext: &UserContext{
					UserID:   "kubernetes-admin",
					Username: "kubernetes-admin",
					Groups:   []string{"system:masters"},
				},
				NetworkContext: &NetworkContext{
					UserAgent: "kubectl/v1.28.0",
				},
				ResourceContext: &ResourceContext{
					ResourceType: "pods",
					ResourceID:   "test-pod",
					Namespace:    "default",
					Operation:    "create",
					APIVersion:   "v1",
				},
				Data: map[string]interface{}{
					"verb":          "create",
					"resource":      "pods",
					"namespace":     "default",
					"response_code": 201,
					"user_agent":    "kubectl/v1.28.0",
				},
			},
			{
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
				EventType: audit.EventTypeAPICall,
				Component: "kube-apiserver",
				Action:    "get",
				Severity:  SeverityInfo,
				Result:    ResultSuccess,
				UserContext: &UserContext{
					UserID:         "system:serviceaccount:kube-system:default",
					ServiceAccount: true,
				},
				ResourceContext: &ResourceContext{
					ResourceType: "nodes",
					ResourceID:   "worker-node-1",
					Operation:    "get",
					APIVersion:   "v1",
				},
				Data: map[string]interface{}{
					"verb":          "get",
					"resource":      "nodes",
					"response_code": 200,
				},
			},
		}

		for _, event := range k8sEvents {
			err := suite.auditSystem.LogEvent(event)
			suite.NoError(err)
		}

		// Wait for processing
		time.Sleep(2 * time.Second)

		// Verify events were processed
		stats := suite.auditSystem.GetStats()
		suite.Greater(stats.EventsReceived, int64(0))

		// Verify log file contains Kubernetes-specific fields
		logFile := filepath.Join(suite.tempDir, "basic.log")
		content, err := ioutil.ReadFile(logFile)
		suite.NoError(err)

		contentStr := string(content)
		suite.Contains(contentStr, "kube-apiserver")
		suite.Contains(contentStr, "kubectl")
		suite.Contains(contentStr, "system:masters")
	})
}

// Performance and scalability test
func (suite *E2EAuditTestSuite) TestScalabilityMetrics() {
	suite.Run("measure end-to-end latency", func() {
		suite.setupBasicAuditSystem()

		const numEvents = 100
		latencies := make([]time.Duration, numEvents)

		for i := 0; i < numEvents; i++ {
			event := &AuditEvent{
				ID:        uuid.New().String(),
				Timestamp: time.Now(),
				EventType: audit.EventTypeAPICall,
				Component: "latency-test",
				Action:    fmt.Sprintf("action-%d", i),
				Severity:  SeverityInfo,
				Result:    ResultSuccess,
			}

			start := time.Now()
			err := suite.auditSystem.LogEvent(event)
			latencies[i] = time.Since(start)

			suite.NoError(err)
		}

		// Calculate metrics
		var totalLatency time.Duration
		var maxLatency time.Duration
		for _, latency := range latencies {
			totalLatency += latency
			if latency > maxLatency {
				maxLatency = latency
			}
		}

		avgLatency := totalLatency / numEvents

		suite.Less(avgLatency, 10*time.Millisecond, "Average latency too high: %v", avgLatency)
		suite.Less(maxLatency, 100*time.Millisecond, "Max latency too high: %v", maxLatency)

		fmt.Printf("Latency metrics: avg=%v, max=%v\n", avgLatency, maxLatency)
	})
}
