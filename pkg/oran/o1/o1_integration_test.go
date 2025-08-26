package o1

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// TestO1AdaptorIntegrationWithManagedElement tests the integration between O1Adaptor and ManagedElement CRD
func TestO1AdaptorIntegrationWithManagedElement(t *testing.T) {
	// Setup test environment
	ctx := context.Background()
	logger := log.FromContext(ctx)

	// Create fake Kubernetes client with ManagedElement CRD
	s := runtime.NewScheme()
	err := nephoranv1.AddToScheme(s)
	require.NoError(t, err)

	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	// Create O1 adaptor
	adaptor := NewO1Adaptor(nil, fakeClient)

	tests := []struct {
		name           string
		managedElement *nephoranv1.ManagedElement
		testFunction   func(*testing.T, *O1Adaptor, *nephoranv1.ManagedElement)
	}{
		{
			name: "ManagedElement with complete O1 configuration",
			managedElement: &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-o-ran-du",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.com/component": "o-ran-du",
						"nephoran.com/type":      "network-function",
					},
				},
				Spec: nephoranv1.ManagedElementSpec{
					Host: "192.168.1.10",
					Port: 830,
					Credentials: nephoranv1.ManagedElementCredentials{
						UsernameRef: &nephoranv1.SecretReference{
							Name: "test-secret",
							Key:  "username",
						},
						PasswordRef: &nephoranv1.SecretReference{
							Name: "test-secret",
							Key:  "password",
						},
					},
					O1Config: `<hardware>
						<component>
							<name>radio-unit-1</name>
							<class>radio</class>
							<state>
								<name>radio-unit-1</name>
								<type>active</type>
								<oper-state>enabled</oper-state>
							</state>
						</component>
					</hardware>`,
				},
				Status: nephoranv1.ManagedElementStatus{
					Conditions: []metav1.Condition{
						{
							Type:   "Ready",
							Status: metav1.ConditionFalse,
							Reason: "NotConfigured",
						},
					},
				},
			},
			testFunction: testO1ConfigurationApplication,
		},
		{
			name: "ManagedElement with JSON O1 configuration",
			managedElement: &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-o-ran-cu",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.com/component": "o-ran-cu",
						"nephoran.com/type":      "network-function",
					},
				},
				Spec: nephoranv1.ManagedElementSpec{
					Host: "192.168.1.11",
					Port: 830,
					Credentials: nephoranv1.ManagedElementCredentials{
						UsernameRef: &nephoranv1.SecretReference{
							Name: "test-secret",
							Key:  "username",
						},
						PasswordRef: &nephoranv1.SecretReference{
							Name: "test-secret",
							Key:  "password",
						},
					},
					O1Config: `{
						"interfaces": {
							"interface": [
								{
									"name": "eth0",
									"type": "ethernet",
									"enabled": true,
									"ipv4": {
										"address": "192.168.1.11",
										"prefix-length": 24
									}
								}
							]
						}
					}`,
				},
				Status: nephoranv1.ManagedElementStatus{
					Conditions: []metav1.Condition{
						{
							Type:   "Ready",
							Status: metav1.ConditionFalse,
							Reason: "NotConfigured",
						},
					},
				},
			},
			testFunction: testO1JSONConfiguration,
		},
		{
			name: "ManagedElement with performance monitoring configuration",
			managedElement: &nephoranv1.ManagedElement{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-performance-me",
					Namespace: "default",
					Labels: map[string]string{
						"nephoran.com/component": "performance-monitor",
						"nephoran.com/type":      "monitoring",
					},
				},
				Spec: nephoranv1.ManagedElementSpec{
					Host: "192.168.1.12",
					Port: 830,
					Credentials: nephoranv1.ManagedElementCredentials{
						UsernameRef: &nephoranv1.SecretReference{
							Name: "test-secret",
							Key:  "username",
						},
						PasswordRef: &nephoranv1.SecretReference{
							Name: "test-secret",
							Key:  "password",
						},
					},
					O1Config: `<performance-measurement>
						<measurement-group>
							<name>cpu-metrics</name>
							<measurement-interval>30</measurement-interval>
							<measurements>
								<measurement>
									<name>cpu-usage</name>
									<enabled>true</enabled>
								</measurement>
							</measurements>
						</measurement-group>
					</performance-measurement>`,
				},
			},
			testFunction: testO1PerformanceMonitoring,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create the ManagedElement in the fake client
			err := fakeClient.Create(ctx, tt.managedElement)
			require.NoError(t, err)

			// Run the specific test function
			tt.testFunction(t, adaptor, tt.managedElement)

			// Cleanup
			err = fakeClient.Delete(ctx, tt.managedElement)
			if err != nil {
				logger.Info("cleanup failed", "error", err)
			}
		})
	}
}

// testO1ConfigurationApplication tests applying O1 configuration to a managed element
func testO1ConfigurationApplication(t *testing.T, adaptor *O1Adaptor, me *nephoranv1.ManagedElement) {
	ctx := context.Background()

	// Test configuration validation
	t.Run("validate_configuration", func(t *testing.T) {
		err := adaptor.ValidateConfiguration(ctx, me.Spec.O1Config)
		assert.NoError(t, err, "O1 configuration should be valid")
	})

	// Test connection status
	t.Run("connection_status", func(t *testing.T) {
		connected := adaptor.IsConnected(me)
		assert.False(t, connected, "Should not be connected initially")
	})

	// Test YANG model validation
	t.Run("yang_validation", func(t *testing.T) {
		// Test that YANG models are loaded
		models := adaptor.yangRegistry.ListModels()
		assert.Greater(t, len(models), 0, "YANG models should be loaded")

		// Verify specific O-RAN models are present
		modelNames := make([]string, len(models))
		for i, model := range models {
			modelNames[i] = model.Name
		}
		assert.Contains(t, modelNames, "o-ran-hardware")
	})

	// Note: Real connection tests would require a mock NETCONF server
	// For integration testing, we focus on the interface contracts
}

// testO1JSONConfiguration tests JSON-based O1 configuration
func testO1JSONConfiguration(t *testing.T, adaptor *O1Adaptor, me *nephoranv1.ManagedElement) {
	ctx := context.Background()

	// Test JSON configuration validation
	t.Run("validate_json_configuration", func(t *testing.T) {
		err := adaptor.ValidateConfiguration(ctx, me.Spec.O1Config)
		assert.NoError(t, err, "JSON O1 configuration should be valid")
	})

	// Test YANG validation against ietf-interfaces model
	t.Run("yang_interfaces_validation", func(t *testing.T) {
		model, err := adaptor.yangRegistry.GetModel("ietf-interfaces")
		assert.NoError(t, err, "Should retrieve ietf-interfaces model")
		assert.NotNil(t, model, "ietf-interfaces model should exist")
	})
}

// testO1PerformanceMonitoring tests performance monitoring configuration
func testO1PerformanceMonitoring(t *testing.T, adaptor *O1Adaptor, me *nephoranv1.ManagedElement) {
	ctx := context.Background()

	// Test performance configuration validation
	t.Run("validate_performance_config", func(t *testing.T) {
		err := adaptor.ValidateConfiguration(ctx, me.Spec.O1Config)
		assert.NoError(t, err, "Performance measurement configuration should be valid")
	})

	// Test metric collection configuration
	t.Run("metric_collection_setup", func(t *testing.T) {
		metricConfig := &MetricConfig{
			MetricNames:      []string{"cpu_usage", "memory_usage", "temperature"},
			CollectionPeriod: 30 * time.Second,
			ReportingPeriod:  60 * time.Second,
			Aggregation:      "AVG",
		}

		// This would typically fail due to no connection, but we test the interface
		err := adaptor.StartMetricCollection(ctx, me, metricConfig)
		assert.Error(t, err, "Should fail without connection, but interface should work")
		assert.Contains(t, err.Error(), "connect", "Error should mention connection issue")
	})
}

// TestO1AdaptorControllerCompatibility tests compatibility with Kubernetes controllers
func TestO1AdaptorControllerCompatibility(t *testing.T) {
	ctx := context.Background()

	// Create fake Kubernetes client
	s := runtime.NewScheme()
	err := nephoranv1.AddToScheme(s)
	require.NoError(t, err)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	// Create O1 adaptor with config
	adaptor := NewO1Adaptor(&O1Config{
		DefaultPort:    830,
		ConnectTimeout: 10 * time.Second,
		RequestTimeout: 30 * time.Second,
		MaxRetries:     2,
		RetryInterval:  2 * time.Second,
	}, fakeClient)

	// Test ManagedElement specification validation
	t.Run("managed_element_spec_validation", func(t *testing.T) {
		validME := &nephoranv1.ManagedElement{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "valid-element",
				Namespace: "default",
			},
			Spec: nephoranv1.ManagedElementSpec{
				Host: "192.168.1.100",
				Port: 830,
				Credentials: nephoranv1.ManagedElementCredentials{
					UsernameRef: &nephoranv1.SecretReference{
						Name: "test-secret",
						Key:  "username",
					},
					PasswordRef: &nephoranv1.SecretReference{
						Name: "test-secret",
						Key:  "password",
					},
				},
				O1Config: `<hardware><component><name>test</name></component></hardware>`,
			},
		}

		// Test that all required fields are accessible
		assert.NotEmpty(t, validME.Spec.Host)
		assert.NotZero(t, validME.Spec.Port)
		assert.NotNil(t, validME.Spec.Credentials.UsernameRef)
		assert.NotEmpty(t, validME.Spec.O1Config)

		// Test configuration validation
		err := adaptor.ValidateConfiguration(ctx, validME.Spec.O1Config)
		assert.NoError(t, err)
	})

	// Test status field compatibility
	t.Run("status_field_compatibility", func(t *testing.T) {
		me := &nephoranv1.ManagedElement{}

		// Test that status fields can be set (as a controller would do)
		me.Status.Conditions = []metav1.Condition{
			{
				Type:   "ConfigurationApplied",
				Status: metav1.ConditionTrue,
				Reason: "ConfigurationValid",
				LastTransitionTime: metav1.Now(),
			},
		}

		assert.Len(t, me.Status.Conditions, 1)
		assert.Equal(t, "ConfigurationApplied", me.Status.Conditions[0].Type)
	})
}

// TestO1AdaptorWithMockNetconfServer simulates NETCONF server interactions
func TestO1AdaptorWithMockNetconfServer(t *testing.T) {
	ctx := context.Background()

	// Note: This would ideally use a mock NETCONF server
	// For now, we test the error handling and interface behavior

	// Create fake Kubernetes client
	s := runtime.NewScheme()
	err := nephoranv1.AddToScheme(s)
	require.NoError(t, err)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	adaptor := NewO1Adaptor(nil, fakeClient)

	mockME := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name: "mock-element",
		},
		Spec: nephoranv1.ManagedElementSpec{
			Host: "localhost", // Non-existent NETCONF server
			Port: 8300,        // Non-standard port to ensure failure
			Credentials: nephoranv1.ManagedElementCredentials{
				UsernameRef: &nephoranv1.SecretReference{
					Name: "test-secret",
					Key:  "username",
				},
				PasswordRef: &nephoranv1.SecretReference{
					Name: "test-secret",
					Key:  "password",
				},
			},
			O1Config: `<test>configuration</test>`,
		},
	}

	t.Run("connection_failure_handling", func(t *testing.T) {
		err := adaptor.Connect(ctx, mockME)
		assert.Error(t, err, "Should fail to connect to non-existent server")
		assert.Contains(t, err.Error(), "failed to establish NETCONF connection")
	})

	t.Run("graceful_disconnect", func(t *testing.T) {
		// Even without connection, disconnect should not panic
		err := adaptor.Disconnect(ctx, mockME)
		assert.NoError(t, err, "Disconnect should handle non-connected state gracefully")
	})

	t.Run("operations_without_connection", func(t *testing.T) {
		// Test that operations fail gracefully without connection
		_, err := adaptor.GetConfiguration(ctx, mockME)
		assert.Error(t, err, "GetConfiguration should fail without connection")

		_, err = adaptor.GetAlarms(ctx, mockME)
		assert.Error(t, err, "GetAlarms should fail without connection")

		_, err = adaptor.GetMetrics(ctx, mockME, []string{"cpu_usage"})
		assert.Error(t, err, "GetMetrics should fail without connection")
	})
}

// TestO1AdaptorFCAPSOperations tests comprehensive FCAPS operations
func TestO1AdaptorFCAPSOperations(t *testing.T) {
	ctx := context.Background()

	// Create fake Kubernetes client
	s := runtime.NewScheme()
	err := nephoranv1.AddToScheme(s)
	require.NoError(t, err)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	adaptor := NewO1Adaptor(nil, fakeClient)

	me := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name: "fcaps-test-element",
		},
		Spec: nephoranv1.ManagedElementSpec{
			Host: "192.168.1.200",
			Port: 830,
			Credentials: nephoranv1.ManagedElementCredentials{
				UsernameRef: &nephoranv1.SecretReference{
					Name: "test-secret",
					Key:  "username",
				},
				PasswordRef: &nephoranv1.SecretReference{
					Name: "test-secret",
					Key:  "password",
				},
			},
		},
	}

	// Test Fault Management (FM)
	t.Run("fault_management", func(t *testing.T) {
		// Test alarm subscription interface
		alarmReceived := false
		callback := func(alarm *Alarm) {
			alarmReceived = true
			assert.NotNil(t, alarm)
			assert.NotEmpty(t, alarm.ID)
			assert.NotEmpty(t, alarm.Severity)
		}

		err := adaptor.SubscribeToAlarms(ctx, me, callback)
		assert.Error(t, err, "Should fail without connection")

		// Test alarm clearing interface
		err = adaptor.ClearAlarm(ctx, me, "test-alarm-123")
		assert.Error(t, err, "Should fail without connection")
	})

	// Test Configuration Management (CM)
	t.Run("configuration_management", func(t *testing.T) {
		testConfig := `<interfaces>
			<interface>
				<name>eth0</name>
				<enabled>true</enabled>
			</interface>
		</interfaces>`

		// Test configuration validation
		err := adaptor.ValidateConfiguration(ctx, testConfig)
		assert.NoError(t, err, "Valid XML configuration should pass validation")

		// Test configuration application
		me.Spec.O1Config = testConfig
		err = adaptor.ApplyConfiguration(ctx, me)
		assert.Error(t, err, "Should fail without connection")
	})

	// Test Performance Management (PM)
	t.Run("performance_management", func(t *testing.T) {
		metricConfig := &MetricConfig{
			MetricNames:      []string{"cpu_usage", "memory_usage", "throughput_mbps"},
			CollectionPeriod: 30 * time.Second,
			ReportingPeriod:  300 * time.Second,
			Aggregation:      "AVG",
		}

		// Test metric collection startup
		err := adaptor.StartMetricCollection(ctx, me, metricConfig)
		assert.Error(t, err, "Should fail without connection")

		// Test metric collection stopping
		err = adaptor.StopMetricCollection(ctx, me, "test-collector-123")
		assert.NoError(t, err, "Should handle non-existent collector gracefully")
	})

	// Test Accounting Management (AM)
	t.Run("accounting_management", func(t *testing.T) {
		filter := &UsageFilter{
			StartTime: time.Now().Add(-24 * time.Hour),
			EndTime:   time.Now(),
			UserID:    "test-user",
			ServiceID: "5g-data-service",
		}

		records, err := adaptor.GetUsageRecords(ctx, me, filter)
		assert.Error(t, err, "Should fail without connection")
		assert.Nil(t, records)
	})

	// Test Security Management (SM)
	t.Run("security_management", func(t *testing.T) {
		policy := &SecurityPolicy{
			PolicyID:    "security-policy-001",
			PolicyType:  "access-control",
			Enforcement: "strict",
			Rules: []SecurityRule{
				{
					RuleID: "rule-001",
					Action: "ALLOW",
					Conditions: map[string]interface{}{
						"source_ip": "192.168.1.0/24",
						"protocol":  "https",
					},
				},
			},
		}

		// Test security policy update
		err := adaptor.UpdateSecurityPolicy(ctx, me, policy)
		assert.Error(t, err, "Should fail without connection")

		// Test security status retrieval
		status, err := adaptor.GetSecurityStatus(ctx, me)
		assert.Error(t, err, "Should fail without connection")
		assert.Nil(t, status)
	})
}

// TestO1AdaptorResourceManagement tests resource cleanup and management
func TestO1AdaptorResourceManagement(t *testing.T) {
	ctx := context.Background()

	// Create fake Kubernetes client
	s := runtime.NewScheme()
	err := nephoranv1.AddToScheme(s)
	require.NoError(t, err)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	adaptor := NewO1Adaptor(&O1Config{
		DefaultPort:    830,
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 10 * time.Second,
		MaxRetries:     1,
		RetryInterval:  1 * time.Second,
	}, fakeClient)

	// Test that adaptor properly initializes internal resources
	t.Run("initialization", func(t *testing.T) {
		assert.NotNil(t, adaptor.clients)
		assert.NotNil(t, adaptor.yangRegistry)
		assert.NotNil(t, adaptor.subscriptions)
		assert.NotNil(t, adaptor.metricCollectors)
		assert.NotNil(t, adaptor.config)
	})

	// Test concurrent access safety
	t.Run("concurrent_access", func(t *testing.T) {
		me1 := &nephoranv1.ManagedElement{
			ObjectMeta: metav1.ObjectMeta{Name: "concurrent-test-1"},
			Spec: nephoranv1.ManagedElementSpec{
				Host: "192.168.1.201",
				Port: 830,
				Credentials: nephoranv1.ManagedElementCredentials{
					UsernameRef: &nephoranv1.SecretReference{
						Name: "test-secret", Key: "username",
					},
					PasswordRef: &nephoranv1.SecretReference{
						Name: "test-secret", Key: "password",
					},
				},
			},
		}

		me2 := &nephoranv1.ManagedElement{
			ObjectMeta: metav1.ObjectMeta{Name: "concurrent-test-2"},
			Spec: nephoranv1.ManagedElementSpec{
				Host: "192.168.1.202",
				Port: 830,
				Credentials: nephoranv1.ManagedElementCredentials{
					UsernameRef: &nephoranv1.SecretReference{
						Name: "test-secret", Key: "username",
					},
					PasswordRef: &nephoranv1.SecretReference{
						Name: "test-secret", Key: "password",
					},
				},
			},
		}

		// Test concurrent connection attempts (will fail, but should not panic)
		done1 := make(chan bool)
		done2 := make(chan bool)

		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Concurrent access caused panic: %v", r)
				}
				done1 <- true
			}()
			adaptor.Connect(ctx, me1)
			adaptor.IsConnected(me1)
			adaptor.Disconnect(ctx, me1)
		}()

		go func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("Concurrent access caused panic: %v", r)
				}
				done2 <- true
			}()
			adaptor.Connect(ctx, me2)
			adaptor.IsConnected(me2)
			adaptor.Disconnect(ctx, me2)
		}()

		<-done1
		<-done2
	})
}

// Benchmark tests for performance verification
func BenchmarkO1AdaptorValidateConfiguration(b *testing.B) {
	ctx := context.Background()

	// Create fake Kubernetes client
	s := runtime.NewScheme()
	nephoranv1.AddToScheme(s)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	adaptor := NewO1Adaptor(nil, fakeClient)

	config := `<hardware>
		<component>
			<name>test-component</name>
			<class>cpu</class>
			<state>
				<name>test-component</name>
				<oper-state>enabled</oper-state>
			</state>
		</component>
	</hardware>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = adaptor.ValidateConfiguration(ctx, config)
	}
}

func BenchmarkO1AdaptorParseAlarmData(b *testing.B) {
	// Create fake Kubernetes client
	s := runtime.NewScheme()
	nephoranv1.AddToScheme(s)
	fakeClient := fake.NewClientBuilder().WithScheme(s).Build()

	adaptor := NewO1Adaptor(nil, fakeClient)

	xmlData := `<data>
		<active-alarm-list>
			<active-alarms>
				<fault-id>12345</fault-id>
				<fault-source>radio-unit-1</fault-source>
				<fault-severity>major</fault-severity>
				<is-cleared>false</is-cleared>
				<fault-text>Radio unit communication failure</fault-text>
				<event-time>2024-01-15T10:30:00Z</event-time>
			</active-alarms>
		</active-alarm-list>
	</data>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = adaptor.parseAlarmData(xmlData, "test-element")
	}
}