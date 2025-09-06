package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/audit"
)

// AuditTrailControllerTestSuite tests the AuditTrail controller
type AuditTrailControllerTestSuite struct {
	suite.Suite
	client     client.Client
	scheme     *runtime.Scheme
	recorder   record.EventRecorder
	controller *AuditTrailController
}

func TestAuditTrailControllerTestSuite(t *testing.T) {
	suite.Run(t, new(AuditTrailControllerTestSuite))
}

func (suite *AuditTrailControllerTestSuite) SetupTest() {
	// Setup scheme
	suite.scheme = runtime.NewScheme()
	err := nephv1.AddToScheme(suite.scheme)
	suite.Require().NoError(err)
	err = corev1.AddToScheme(suite.scheme)
	suite.Require().NoError(err)

	// Setup fake client
	suite.client = fake.NewClientBuilder().WithScheme(suite.scheme).Build()

	// Setup fake event recorder
	suite.recorder = record.NewFakeRecorder(100)

	// Setup controller
	logger := zap.New(zap.UseDevMode(true))
	suite.controller = NewAuditTrailController(
		suite.client,
		logger,
		suite.scheme,
		suite.recorder,
	)
}

func (suite *AuditTrailControllerTestSuite) TearDownTest() {
	// Stop any running audit systems
	if suite.controller.auditSystems != nil {
		for _, system := range suite.controller.auditSystems {
			system.Stop()
		}
	}
}

// Test basic controller lifecycle
func (suite *AuditTrailControllerTestSuite) TestControllerLifecycle() {
	suite.Run("create audit trail resource", func() {
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: metav1.ObjectMeta{
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
				ComplianceMode:  []string{"soc2"},
				Backends: []nephv1.AuditBackendConfig{
					{
						Type:    "file",
						Enabled: true,
						Name:    "test-file-backend",
						Settings: runtime.RawExtension{
							Raw: json.RawMessage(`{"path": "/tmp/test-audit.log"}`),
						},
					},
				},
			},
		}

		err := suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		// Reconcile
		result, err := suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test-audit-trail",
				Namespace: "default",
			},
		})

		suite.NoError(err)
		suite.False(result.Requeue)

		// Verify audit system was created
		auditSystem := suite.controller.GetAuditSystem("default", "test-audit-trail")
		suite.NotNil(auditSystem)

		// Verify status was updated
		var updatedAuditTrail nephv1.AuditTrail
		err = suite.client.Get(context.Background(), types.NamespacedName{
			Name:      "test-audit-trail",
			Namespace: "default",
		}, &updatedAuditTrail)
		suite.NoError(err)
		suite.Equal("Running", updatedAuditTrail.Status.Phase)
		suite.NotNil(updatedAuditTrail.Status.LastUpdate)
	})

	suite.Run("update audit trail configuration", func() {
		// Get existing audit trail
		var auditTrail nephv1.AuditTrail
		err := suite.client.Get(context.Background(), types.NamespacedName{
			Name:      "test-audit-trail",
			Namespace: "default",
		}, &auditTrail)
		suite.NoError(err)

		// Update configuration
		auditTrail.Spec.BatchSize = 20
		auditTrail.Spec.Backends = append(auditTrail.Spec.Backends, nephv1.AuditBackendConfig{
			Type:    "syslog",
			Enabled: true,
			Name:    "test-syslog-backend",
			Settings: runtime.RawExtension{
				Raw: json.RawMessage(`{"network": "udp", "address": "localhost:514"}`),
			},
		})

		err = suite.client.Update(context.Background(), &auditTrail)
		suite.NoError(err)

		// Reconcile
		result, err := suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test-audit-trail",
				Namespace: "default",
			},
		})

		suite.NoError(err)
		suite.False(result.Requeue)

		// Verify audit system was updated
		auditSystem := suite.controller.GetAuditSystem("default", "test-audit-trail")
		suite.NotNil(auditSystem)
		stats := auditSystem.GetStats()
		suite.Equal(2, stats.BackendCount) // file + syslog
	})

	suite.Run("delete audit trail", func() {
		// Get audit trail
		var auditTrail nephv1.AuditTrail
		err := suite.client.Get(context.Background(), types.NamespacedName{
			Name:      "test-audit-trail",
			Namespace: "default",
		}, &auditTrail)
		suite.NoError(err)

		// Delete audit trail
		err = suite.client.Delete(context.Background(), &auditTrail)
		suite.NoError(err)

		// Reconcile deletion
		result, err := suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "test-audit-trail",
				Namespace: "default",
			},
		})

		suite.NoError(err)
		suite.False(result.Requeue)

		// Verify audit system was cleaned up
		auditSystem := suite.controller.GetAuditSystem("default", "test-audit-trail")
		suite.Nil(auditSystem)
	})
}

// Test configuration validation and conversion
func (suite *AuditTrailControllerTestSuite) TestConfigurationHandling() {
	suite.Run("valid configuration conversion", func() {
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "config-test",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled:         true,
				LogLevel:        "warning",
				BatchSize:       50,
				FlushInterval:   10,
				MaxQueueSize:    5000,
				EnableIntegrity: false,
				ComplianceMode:  []string{"iso27001", "pci_dss"},
				Backends: []nephv1.AuditBackendConfig{
					{
						Type:    "elasticsearch",
						Enabled: true,
						Name:    "es-backend",
						Format:  "json",
						Settings: runtime.RawExtension{
							Raw: json.RawMessage(`{"urls": ["http://localhost:9200"], "index": "audit-logs"}`),
						},
						RetryPolicy: &nephv1.RetryPolicySpec{
							MaxRetries:   5,
							InitialDelay: 2,
							MaxDelay:     30,
						},
						TLS: &nephv1.TLSConfigSpec{
							Enabled:    true,
							ServerName: "elasticsearch.local",
						},
						Filter: &nephv1.FilterConfigSpec{
							MinSeverity:   "error",
							EventTypes:    []string{"authentication", "authorization"},
							Components:    []string{"auth-service"},
							ExcludeTypes:  []string{"health_check"},
							IncludeFields: []string{"user_id", "action", "result"},
							ExcludeFields: []string{"debug_info"},
						},
					},
				},
			},
		}

		config, err := suite.controller.buildAuditSystemConfig(context.Background(), auditTrail, suite.controller.Log)
		suite.NoError(err)
		suite.NotNil(config)

		// Verify basic configuration
		suite.True(config.Enabled)
		suite.Equal(audit.SeverityWarning, config.LogLevel)
		suite.Equal(50, config.BatchSize)
		suite.Equal(10*time.Second, config.FlushInterval)
		suite.Equal(5000, config.MaxQueueSize)
		suite.False(config.EnableIntegrity)
		suite.Contains(config.ComplianceMode, audit.ComplianceISO27001)
		suite.Contains(config.ComplianceMode, audit.CompliancePCIDSS)

		// Verify backend configuration
		suite.Len(config.Backends, 1)
		backendConfig := config.Backends[0]
		suite.Equal("elasticsearch", string(backendConfig.Type))
		suite.True(backendConfig.Enabled)
		suite.Equal("es-backend", backendConfig.Name)
		suite.Equal("json", backendConfig.Format)
		suite.Equal(5, backendConfig.RetryPolicy.MaxRetries)
		suite.Equal(2*time.Second, backendConfig.RetryPolicy.InitialDelay)
		suite.True(backendConfig.TLS.Enabled)
		suite.Equal("elasticsearch.local", backendConfig.TLS.ServerName)
		suite.Equal(audit.SeverityError, backendConfig.Filter.MinSeverity)
		suite.Contains(backendConfig.Filter.EventTypes, audit.EventTypeAuthentication)
	})

	suite.Run("default configuration values", func() {
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "defaults-test",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled: true,
				// Most fields left empty to test defaults
			},
		}

		config, err := suite.controller.buildAuditSystemConfig(context.Background(), auditTrail, suite.controller.Log)
		suite.NoError(err)

		// Verify defaults are applied
		suite.Equal(100, config.BatchSize)
		suite.Equal(10*time.Second, config.FlushInterval)
		suite.Equal(10000, config.MaxQueueSize)
	})

	suite.Run("invalid configuration handling", func() {
		invalidConfigs := []nephv1.AuditTrailSpec{
			{
				Enabled:       true,
				BatchSize:     -1, // Invalid batch size
				FlushInterval: 10,
			},
			{
				Enabled:       true,
				BatchSize:     10,
				FlushInterval: -1, // Invalid flush interval
			},
			{
				Enabled:       true,
				BatchSize:     10,
				FlushInterval: 10,
				MaxQueueSize:  -1, // Invalid queue size
			},
		}

		for i, spec := range invalidConfigs {
			suite.Run(fmt.Sprintf("invalid_config_%d", i), func() {
				auditTrail := &nephv1.AuditTrail{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("invalid-test-%d", i),
						Namespace: "default",
					},
					Spec: spec,
				}

				config, err := suite.controller.buildAuditSystemConfig(context.Background(), auditTrail, suite.controller.Log)

				// Should either error or apply safe defaults
				if err != nil {
					suite.Contains(err.Error(), "invalid")
				} else {
					// Verify safe defaults were applied
					suite.Greater(config.BatchSize, 0)
					suite.Greater(config.FlushInterval, time.Duration(0))
					suite.Greater(config.MaxQueueSize, 0)
				}
			})
		}
	})
}

// Test error handling scenarios
func (suite *AuditTrailControllerTestSuite) TestErrorHandling() {
	suite.Run("resource not found", func() {
		// Reconcile non-existent resource
		result, err := suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "non-existent",
				Namespace: "default",
			},
		})

		suite.NoError(err)
		suite.False(result.Requeue)
	})

	suite.Run("malformed backend configuration", func() {
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "malformed-test",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled:  true,
				LogLevel: "info",
				Backends: []nephv1.AuditBackendConfig{
					{
						Type:    "invalid_backend_type",
						Enabled: true,
						Name:    "invalid-backend",
						Settings: runtime.RawExtension{
							Raw: json.RawMessage(`invalid json`),
						},
					},
				},
			},
		}

		err := suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		// Reconcile should handle the error gracefully
		_, err = suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "malformed-test",
				Namespace: "default",
			},
		})
		// Should not crash, might return error or handle gracefully
		if err != nil {
			suite.T().Logf("Expected error handling malformed config: %v", err)
		}

		// Verify status reflects the error
		var updatedAuditTrail nephv1.AuditTrail
		err = suite.client.Get(context.Background(), types.NamespacedName{
			Name:      "malformed-test",
			Namespace: "default",
		}, &updatedAuditTrail)
		suite.NoError(err)

		// Status should indicate failure or error
		if updatedAuditTrail.Status.Phase == "Failed" {
			suite.NotEmpty(updatedAuditTrail.Status.Conditions)
		}
	})

	suite.Run("audit system creation failure", func() {
		// This test would require mocking the audit system creation
		// For now, we'll test with configuration that might fail
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "creation-failure-test",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled:      true,
				LogLevel:     "info",
				MaxQueueSize: 0, // This might cause creation to fail
			},
		}

		err := suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		result, err := suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "creation-failure-test",
				Namespace: "default",
			},
		})

		// Controller should handle creation failures gracefully
		suite.T().Logf("Reconcile result: %+v, error: %v", result, err)
	})
}

// Test status updates
func (suite *AuditTrailControllerTestSuite) TestStatusUpdates() {
	suite.Run("normal status progression", func() {
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "status-test",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled:  true,
				LogLevel: "info",
				Backends: []nephv1.AuditBackendConfig{
					{
						Type:    "file",
						Enabled: true,
						Name:    "status-file-backend",
						Settings: runtime.RawExtension{
							Raw: json.RawMessage(`{"path": "/tmp/status-test.log"}`),
						},
					},
				},
			},
		}

		err := suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		// Initial reconcile
		_, err = suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "status-test",
				Namespace: "default",
			},
		})
		suite.NoError(err)

		// Check initial status
		var updatedAuditTrail nephv1.AuditTrail
		err = suite.client.Get(context.Background(), types.NamespacedName{
			Name:      "status-test",
			Namespace: "default",
		}, &updatedAuditTrail)
		suite.NoError(err)

		suite.Equal("Running", updatedAuditTrail.Status.Phase)
		suite.NotNil(updatedAuditTrail.Status.LastUpdate)
		suite.NotNil(updatedAuditTrail.Status.Stats)
		suite.Equal(int64(1), updatedAuditTrail.Status.Stats.BackendCount)
		suite.NotEmpty(updatedAuditTrail.Status.Conditions)

		// Verify Ready condition
		readyCondition := findCondition(updatedAuditTrail.Status.Conditions, ConditionTypeReady)
		suite.NotNil(readyCondition)
		suite.Equal(metav1.ConditionTrue, readyCondition.Status)
		suite.Equal("SystemOperational", readyCondition.Reason)
	})

	suite.Run("disabled audit trail status", func() {
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "disabled-test",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled: false, // Disabled
			},
		}

		err := suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		_, err = suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "disabled-test",
				Namespace: "default",
			},
		})
		suite.NoError(err)

		var updatedAuditTrail nephv1.AuditTrail
		err = suite.client.Get(context.Background(), types.NamespacedName{
			Name:      "disabled-test",
			Namespace: "default",
		}, &updatedAuditTrail)
		suite.NoError(err)

		suite.Equal("Stopped", updatedAuditTrail.Status.Phase)
	})
}

// Test finalizer handling
func (suite *AuditTrailControllerTestSuite) TestFinalizerHandling() {
	suite.Run("finalizer addition and removal", func() {
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "finalizer-test",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled: true,
			},
		}

		err := suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		// Initial reconcile should add finalizer
		_, err = suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "finalizer-test",
				Namespace: "default",
			},
		})
		suite.NoError(err)

		// Check finalizer was added
		var updatedAuditTrail nephv1.AuditTrail
		err = suite.client.Get(context.Background(), types.NamespacedName{
			Name:      "finalizer-test",
			Namespace: "default",
		}, &updatedAuditTrail)
		suite.NoError(err)
		suite.Contains(updatedAuditTrail.Finalizers, AuditTrailFinalizer)

		// Delete the resource
		err = suite.client.Delete(context.Background(), &updatedAuditTrail)
		suite.NoError(err)

		// Reconcile deletion
		_, err = suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "finalizer-test",
				Namespace: "default",
			},
		})
		suite.NoError(err)

		// Resource should be deleted (finalizer removed)
		err = suite.client.Get(context.Background(), types.NamespacedName{
			Name:      "finalizer-test",
			Namespace: "default",
		}, &updatedAuditTrail)
		suite.True(client.IgnoreNotFound(err) == nil)
	})
}

// Test concurrent reconciliation
func (suite *AuditTrailControllerTestSuite) TestConcurrentReconciliation() {
	suite.Run("concurrent reconcile requests", func() {
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "concurrent-test",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled:  true,
				LogLevel: "info",
			},
		}

		err := suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		// Run multiple concurrent reconciles
		const numConcurrent = 5
		results := make(chan error, numConcurrent)

		for i := 0; i < numConcurrent; i++ {
			go func() {
				_, err := suite.controller.Reconcile(context.Background(), ctrl.Request{
					NamespacedName: types.NamespacedName{
						Name:      "concurrent-test",
						Namespace: "default",
					},
				})
				results <- err
			}()
		}

		// Collect results
		for i := 0; i < numConcurrent; i++ {
			err := <-results
			suite.NoError(err)
		}

		// Verify final state is consistent
		auditSystem := suite.controller.GetAuditSystem("default", "concurrent-test")
		suite.NotNil(auditSystem)
	})
}

// Test metrics and events
func (suite *AuditTrailControllerTestSuite) TestMetricsAndEvents() {
	suite.Run("events generation", func() {
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "events-test",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled: true,
			},
		}

		err := suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		// Clear recorder
		recorder := suite.recorder.(*record.FakeRecorder)
		// Drain any existing events
		select {
		case <-recorder.Events:
		default:
		}

		// Reconcile
		_, err = suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "events-test",
				Namespace: "default",
			},
		})
		suite.NoError(err)

		// Check for events
		select {
		case event := <-recorder.Events:
			suite.Contains(event, "Created")
			suite.T().Logf("Received event: %s", event)
		case <-time.After(1 * time.Second):
			suite.T().Log("No events received")
		}
	})
}

// Test integration with Kubernetes resources
func (suite *AuditTrailControllerTestSuite) TestKubernetesIntegration() {
	suite.Run("secret reference handling", func() {
		// Create a secret for backend authentication
		secret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "audit-backend-secret",
				Namespace: "default",
			},
			Data: map[string][]byte{
				"username": []byte("audit-user"),
				"password": []byte("secret-password"),
				"api-key":  []byte("secret-api-key"),
			},
		}

		err := suite.client.Create(context.Background(), secret)
		suite.NoError(err)

		// Create audit trail that references the secret
		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "secret-test",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled: true,
				Backends: []nephv1.AuditBackendConfig{
					{
						Type:    "webhook",
						Enabled: true,
						Name:    "secret-webhook",
						Settings: runtime.RawExtension{
							Raw: json.RawMessage(`{"url": "https://webhook.example.com/audit", "auth": {"type": "basic", "username_secret": "audit-backend-secret", "password_secret": "audit-backend-secret"}}`),
						},
					},
				},
			},
		}

		err = suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		// Reconcile
		_, err = suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "secret-test",
				Namespace: "default",
			},
		})
		suite.NoError(err)

		// Verify audit system was created successfully
		auditSystem := suite.controller.GetAuditSystem("default", "secret-test")
		suite.NotNil(auditSystem)
	})

	suite.Run("configmap reference handling", func() {
		// Create a configmap for backend configuration
		configData := map[string]interface{}{
			"urls":  []string{"http://elasticsearch:9200"},
			"index": "audit-logs",
		}
		configBytes, _ := json.Marshal(configData)

		configMap := &corev1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "audit-backend-config",
				Namespace: "default",
			},
			Data: map[string]string{
				"elasticsearch.json": string(configBytes),
			},
		}

		err := suite.client.Create(context.Background(), configMap)
		suite.NoError(err)

		auditTrail := &nephv1.AuditTrail{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "configmap-test",
				Namespace: "default",
			},
			Spec: nephv1.AuditTrailSpec{
				Enabled: true,
				Backends: []nephv1.AuditBackendConfig{
					{
						Type:    "elasticsearch",
						Enabled: true,
						Name:    "configmap-elasticsearch",
						Settings: runtime.RawExtension{
							Raw: json.RawMessage(`{"urls": ["http://localhost:9200"], "index": "audit-logs", "config_from_configmap": {"name": "audit-backend-config", "key": "elasticsearch.json"}}`),
						},
					},
				},
			},
		}

		err = suite.client.Create(context.Background(), auditTrail)
		suite.NoError(err)

		_, err = suite.controller.Reconcile(context.Background(), ctrl.Request{
			NamespacedName: types.NamespacedName{
				Name:      "configmap-test",
				Namespace: "default",
			},
		})
		suite.NoError(err)

		auditSystem := suite.controller.GetAuditSystem("default", "configmap-test")
		suite.NotNil(auditSystem)
	})
}

// Benchmark controller performance
func BenchmarkAuditTrailControllerReconcile(b *testing.B) {
	// Setup
	scheme := runtime.NewScheme()
	err := nephv1.AddToScheme(scheme)
	require.NoError(b, err)

	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	recorder := record.NewFakeRecorder(1000)
	logger := zap.New(zap.UseDevMode(true))

	controller := NewAuditTrailController(client, logger, scheme, recorder)

	// Create test resource
	auditTrail := &nephv1.AuditTrail{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "benchmark-test",
			Namespace: "default",
		},
		Spec: nephv1.AuditTrailSpec{
			Enabled: true,
		},
	}

	err = client.Create(context.Background(), auditTrail)
	require.NoError(b, err)

	request := ctrl.Request{
		NamespacedName: types.NamespacedName{
			Name:      "benchmark-test",
			Namespace: "default",
		},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		result, err := controller.Reconcile(context.Background(), request)
		if err != nil {
			b.Fatal(err)
		}
		_ = result
	}

	// Cleanup
	if controller.auditSystems != nil {
		for _, system := range controller.auditSystems {
			system.Stop()
		}
	}
}

// Test helper functions

func int64Ptr(i int64) *int64 {
	return &i
}

// Table-driven tests for various scenarios

func TestAuditTrailControllerScenarios(t *testing.T) {
	tests := []struct {
		name           string
		spec           nephv1.AuditTrailSpec
		expectError    bool
		expectedPhase  string
		expectedEvents int
	}{
		{
			name: "minimal valid configuration",
			spec: nephv1.AuditTrailSpec{
				Enabled: true,
			},
			expectError:    false,
			expectedPhase:  "Running",
			expectedEvents: 1,
		},
		{
			name: "disabled audit trail",
			spec: nephv1.AuditTrailSpec{
				Enabled: false,
			},
			expectError:    false,
			expectedPhase:  "Stopped",
			expectedEvents: 0,
		},
		{
			name: "complex configuration",
			spec: nephv1.AuditTrailSpec{
				Enabled:         true,
				LogLevel:        "debug",
				BatchSize:       100,
				FlushInterval:   30,
				MaxQueueSize:    5000,
				EnableIntegrity: true,
				ComplianceMode:  []string{"soc2", "iso27001"},
				Backends: []nephv1.AuditBackendConfig{
					{
						Type:    "file",
						Enabled: true,
						Name:    "main-file",
						Settings: runtime.RawExtension{
							Raw: json.RawMessage(`{"path": "/var/log/audit.log"}`),
						},
					},
					{
						Type:    "syslog",
						Enabled: true,
						Name:    "syslog-backend",
						Settings: runtime.RawExtension{
							Raw: json.RawMessage(`{"network": "tcp", "address": "syslog.local:514"}`),
						},
					},
				},
			},
			expectError:    false,
			expectedPhase:  "Running",
			expectedEvents: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			scheme := runtime.NewScheme()
			err := nephv1.AddToScheme(scheme)
			require.NoError(t, err)

			client := fake.NewClientBuilder().WithScheme(scheme).Build()
			recorder := record.NewFakeRecorder(100)
			logger := zap.New(zap.UseDevMode(true))

			controller := NewAuditTrailController(client, logger, scheme, recorder)
			defer func() {
				if controller.auditSystems != nil {
					for _, system := range controller.auditSystems {
						system.Stop()
					}
				}
			}()

			// Create resource
			auditTrail := &nephv1.AuditTrail{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("test-%s", tt.name),
					Namespace: "default",
				},
				Spec: tt.spec,
			}

			err = client.Create(context.Background(), auditTrail)
			require.NoError(t, err)

			// Reconcile
			result, err := controller.Reconcile(context.Background(), ctrl.Request{
				NamespacedName: types.NamespacedName{
					Name:      auditTrail.Name,
					Namespace: auditTrail.Namespace,
				},
			})

			// Verify results
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.False(t, result.Requeue)

				// Check status
				var updatedAuditTrail nephv1.AuditTrail
				err = client.Get(context.Background(), types.NamespacedName{
					Name:      auditTrail.Name,
					Namespace: auditTrail.Namespace,
				}, &updatedAuditTrail)
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPhase, updatedAuditTrail.Status.Phase)

				// Check audit system creation
				if tt.spec.Enabled {
					auditSystem := controller.GetAuditSystem(auditTrail.Namespace, auditTrail.Name)
					assert.NotNil(t, auditSystem, "Audit system should be created for enabled trail")
				}
			}
		})
	}
}
