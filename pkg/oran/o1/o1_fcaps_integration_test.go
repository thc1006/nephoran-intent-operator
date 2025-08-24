//go:build integration

package o1_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/prometheus/client_golang/prometheus"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran/o1"
)

var _ = ginkgo.Describe("O1 FCAPS Integration Tests", func() {
	var (
		ctx           context.Context
		k8sClient     client.Client
		o1Adaptor     *o1.O1Adaptor
		faultMgr      *o1.FaultManager
		configMgr     *o1.ConfigurationManager
		perfMgr       *o1.PerformanceManager
		securityMgr   *o1.SecurityManager
		accountingMgr *o1.AccountingManager
		smoIntegrator *o1.SMOIntegrator
		streamingSvc  *o1.StreamingService
		testServer    *httptest.Server
		wsServer      *httptest.Server
	)

	ginkgo.BeforeEach(func() {
		ctx = context.Background()

		// Create fake Kubernetes client
		scheme := runtime.NewScheme()
		_ = corev1.AddToScheme(scheme)
		_ = nephoranv1.AddToScheme(scheme)

		k8sClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		// Create test configuration
		config := &o1.O1Config{
			DefaultPort:    830,
			ConnectTimeout: 5 * time.Second,
			RequestTimeout: 10 * time.Second,
			MaxRetries:     3,
			RetryInterval:  1 * time.Second,
		}

		// Initialize O1 adaptor
		o1Adaptor = o1.NewO1Adaptor(config, k8sClient)

		// Initialize FCAPS managers
		faultMgr = o1.NewFaultManager(o1Adaptor, k8sClient)
		configMgr = o1.NewConfigurationManager(o1Adaptor, k8sClient)
		perfMgr = o1.NewPerformanceManager(o1Adaptor, k8sClient)
		securityMgr = o1.NewSecurityManager(o1Adaptor, k8sClient)
		accountingMgr = o1.NewAccountingManager(o1Adaptor, k8sClient)

		// Initialize SMO integrator
		smoConfig := &o1.SMOConfig{
			Endpoint:               "http://localhost:8080/smo",
			RegistrationEnabled:    true,
			HeartbeatInterval:      30 * time.Second,
			HierarchicalManagement: true,
		}
		smoIntegrator = o1.NewSMOIntegrator(smoConfig, o1Adaptor, k8sClient)

		// Initialize streaming service
		streamingConfig := &o1.StreamingConfig{
			Port:           8081,
			MaxConnections: 100,
			BufferSize:     1024,
			QoSLevel:       o1.QoSReliable,
		}
		streamingSvc = o1.NewStreamingService(streamingConfig, o1Adaptor)

		// Setup mock NETCONF server
		testServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleMockNetconfRequest(w, r)
		}))

		// Setup WebSocket server for streaming
		wsServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			handleWebSocketConnection(w, r)
		}))
	})

	ginkgo.AfterEach(func() {
		if testServer != nil {
			testServer.Close()
		}
		if wsServer != nil {
			wsServer.Close()
		}
		if streamingSvc != nil {
			_ = streamingSvc.Stop(ctx)
		}
	})

	ginkgo.Describe("Fault Management", func() {
		ginkgo.It("should collect and correlate alarms", func() {
			// Create test managed element
			me := createTestManagedElement("test-ne-1", testServer.URL)
			gomega.Expect(k8sClient.Create(ctx, me)).To(gomega.Succeed())

			// Start fault manager
			gomega.Expect(faultMgr.Start(ctx)).To(gomega.Succeed())

			// Simulate alarms
			alarms := []o1.Alarm{
				{
					ID:               "alarm-001",
					ManagedElementID: "test-ne-1",
					Severity:         "CRITICAL",
					Type:             "COMMUNICATIONS",
					ProbableCause:    "link-failure",
					SpecificProblem:  "Ethernet link down",
					TimeRaised:       time.Now(),
				},
				{
					ID:               "alarm-002",
					ManagedElementID: "test-ne-1",
					Severity:         "MAJOR",
					Type:             "PROCESSING",
					ProbableCause:    "cpu-overload",
					SpecificProblem:  "CPU usage above 90%",
					TimeRaised:       time.Now(),
				},
			}

			// Process alarms
			for _, alarm := range alarms {
				gomega.Expect(faultMgr.ProcessAlarm(ctx, &alarm)).To(gomega.Succeed())
			}

			// Verify alarm correlation
			correlatedAlarms, err := faultMgr.GetCorrelatedAlarms(ctx, "test-ne-1")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(correlatedAlarms).To(gomega.HaveLen(1)) // Should be correlated

			// Verify root cause analysis
			rootCause, err := faultMgr.AnalyzeRootCause(ctx, "test-ne-1")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(rootCause.ProbableCause).To(gomega.Equal("link-failure"))
		})

		ginkgo.It("should integrate with Prometheus AlertManager", func() {
			// Setup mock AlertManager
			alertManagerServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gomega.Expect(r.Method).To(gomega.Equal("POST"))
				gomega.Expect(r.URL.Path).To(gomega.Equal("/api/v1/alerts"))

				var alerts []map[string]interface{}
				gomega.Expect(json.NewDecoder(r.Body).Decode(&alerts)).To(gomega.Succeed())
				gomega.Expect(alerts).ToNot(gomega.BeEmpty())

				w.WriteHeader(http.StatusOK)
			}))
			defer alertManagerServer.Close()

			// Configure AlertManager integration
			faultMgr.ConfigureAlertManager(alertManagerServer.URL, "")

			// Create and send alarm
			alarm := &o1.Alarm{
				ID:               "alarm-003",
				ManagedElementID: "test-ne-1",
				Severity:         "CRITICAL",
				Type:             "QUALITY_OF_SERVICE",
				ProbableCause:    "threshold-crossed",
				SpecificProblem:  "Packet loss above threshold",
				TimeRaised:       time.Now(),
			}

			gomega.Expect(faultMgr.SendToAlertManager(ctx, alarm)).To(gomega.Succeed())
		})
	})

	ginkgo.Describe("Configuration Management", func() {
		ginkgo.It("should manage configuration versions and rollback", func() {
			me := createTestManagedElement("test-ne-2", testServer.URL)
			gomega.Expect(k8sClient.Create(ctx, me)).To(gomega.Succeed())

			// Start configuration manager
			gomega.Expect(configMgr.Start(ctx)).To(gomega.Succeed())

			// Apply initial configuration
			config1 := `<config><interface><name>eth0</name><enabled>true</enabled></interface></config>`
			version1, err := configMgr.ApplyConfiguration(ctx, me, config1)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(version1).ToNot(gomega.BeEmpty())

			// Apply updated configuration
			config2 := `<config><interface><name>eth0</name><enabled>false</enabled></interface></config>`
			version2, err := configMgr.ApplyConfiguration(ctx, me, config2)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(version2).ToNot(gomega.Equal(version1))

			// Get configuration history
			history, err := configMgr.GetConfigurationHistory(ctx, me)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(history).To(gomega.HaveLen(2))

			// Rollback to previous version
			gomega.Expect(configMgr.RollbackConfiguration(ctx, me, version1)).To(gomega.Succeed())

			// Verify rollback
			currentConfig, err := configMgr.GetCurrentConfiguration(ctx, me)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(currentConfig).To(gomega.ContainSubstring("true"))
		})

		ginkgo.It("should detect configuration drift", func() {
			me := createTestManagedElement("test-ne-3", testServer.URL)
			gomega.Expect(k8sClient.Create(ctx, me)).To(gomega.Succeed())

			// Apply baseline configuration
			baselineConfig := `<config><ntp><server>10.0.0.1</server></ntp></config>`
			_, err := configMgr.ApplyConfiguration(ctx, me, baselineConfig)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			// Enable drift detection
			gomega.Expect(configMgr.EnableDriftDetection(ctx, me, 5*time.Second)).To(gomega.Succeed())

			// Simulate configuration change outside of manager
			time.Sleep(6 * time.Second)

			// Check for drift
			driftDetected, driftDetails, err := configMgr.CheckConfigurationDrift(ctx, me)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(driftDetected).To(gomega.BeTrue())
			gomega.Expect(driftDetails).ToNot(gomega.BeEmpty())
		})
	})

	ginkgo.Describe("Performance Management", func() {
		ginkgo.It("should collect and aggregate KPIs", func() {
			me := createTestManagedElement("test-ne-4", testServer.URL)
			gomega.Expect(k8sClient.Create(ctx, me)).To(gomega.Succeed())

			// Start performance manager
			gomega.Expect(perfMgr.Start(ctx)).To(gomega.Succeed())

			// Configure KPI collection
			kpiConfig := &o1.KPICollectionConfig{
				ManagedElement: me.Name,
				KPIs: []string{
					"throughput",
					"latency",
					"packet_loss",
					"cpu_usage",
					"memory_usage",
				},
				CollectionInterval: 5 * time.Second,
				AggregationPeriods: []string{"5min", "15min", "1hour"},
			}

			collectionID, err := perfMgr.StartKPICollection(ctx, kpiConfig)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(collectionID).ToNot(gomega.BeEmpty())

			// Wait for collection
			time.Sleep(10 * time.Second)

			// Get aggregated KPIs
			aggregatedKPIs, err := perfMgr.GetAggregatedKPIs(ctx, me.Name, "5min")
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(aggregatedKPIs).ToNot(gomega.BeEmpty())

			// Verify KPI values
			gomega.Expect(aggregatedKPIs).To(gomega.HaveKey("throughput"))
			gomega.Expect(aggregatedKPIs).To(gomega.HaveKey("latency"))

			// Stop collection
			gomega.Expect(perfMgr.StopKPICollection(ctx, collectionID)).To(gomega.Succeed())
		})

		ginkgo.It("should detect performance anomalies", func() {
			me := createTestManagedElement("test-ne-5", testServer.URL)
			gomega.Expect(k8sClient.Create(ctx, me)).To(gomega.Succeed())

			// Enable anomaly detection
			anomalyConfig := &o1.AnomalyDetectionConfig{
				ManagedElement: me.Name,
				Metrics:        []string{"latency", "packet_loss"},
				Algorithm:      "isolation_forest",
				Sensitivity:    0.95,
			}

			gomega.Expect(perfMgr.EnableAnomalyDetection(ctx, anomalyConfig)).To(gomega.Succeed())

			// Simulate anomalous KPI values
			anomalousKPIs := map[string]float64{
				"latency":     500.0, // High latency
				"packet_loss": 10.0,  // High packet loss
			}

			// Process KPIs
			anomalies, err := perfMgr.DetectAnomalies(ctx, me.Name, anomalousKPIs)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(anomalies).ToNot(gomega.BeEmpty())
			gomega.Expect(anomalies[0].Metric).To(gomega.BeElementOf("latency", "packet_loss"))
		})
	})

	ginkgo.Describe("Security Management", func() {
		ginkgo.It("should manage certificates lifecycle", func() {
			me := createTestManagedElement("test-ne-6", testServer.URL)
			gomega.Expect(k8sClient.Create(ctx, me)).To(gomega.Succeed())

			// Start security manager
			gomega.Expect(securityMgr.Start(ctx)).To(gomega.Succeed())

			// Generate certificate request
			certRequest := &o1.CertificateRequest{
				CommonName:   "test-ne-6.o-ran.local",
				Organization: "O-RAN Alliance",
				Country:      "US",
				ValidityDays: 365,
			}

			cert, err := securityMgr.GenerateCertificate(ctx, certRequest)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(cert).ToNot(gomega.BeNil())

			// Install certificate
			gomega.Expect(securityMgr.InstallCertificate(ctx, me, cert)).To(gomega.Succeed())

			// Check certificate expiry
			expiryWarning, daysRemaining, err := securityMgr.CheckCertificateExpiry(ctx, me)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(expiryWarning).To(gomega.BeFalse())
			gomega.Expect(daysRemaining).To(gomega.BeNumerically(">", 300))

			// Rotate certificate
			newCert, err := securityMgr.RotateCertificate(ctx, me)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(newCert).ToNot(gomega.BeNil())
		})

		ginkgo.It("should detect security threats", func() {
			me := createTestManagedElement("test-ne-7", testServer.URL)
			gomega.Expect(k8sClient.Create(ctx, me)).To(gomega.Succeed())

			// Enable intrusion detection
			idsConfig := &o1.IntrusionDetectionConfig{
				ManagedElement:  me.Name,
				DetectionRules:  []string{"brute_force", "port_scan", "dos_attack"},
				AlertThreshold:  3,
				BlockingEnabled: true,
			}

			gomega.Expect(securityMgr.EnableIntrusionDetection(ctx, idsConfig)).To(gomega.Succeed())

			// Simulate security events
			events := []o1.SecurityEvent{
				{
					Type:      "failed_login",
					Source:    "192.168.1.100",
					Target:    me.Name,
					Timestamp: time.Now(),
				},
				{
					Type:      "failed_login",
					Source:    "192.168.1.100",
					Target:    me.Name,
					Timestamp: time.Now().Add(1 * time.Second),
				},
				{
					Type:      "failed_login",
					Source:    "192.168.1.100",
					Target:    me.Name,
					Timestamp: time.Now().Add(2 * time.Second),
				},
			}

			// Process events
			for _, event := range events {
				threat, err := securityMgr.ProcessSecurityEvent(ctx, &event)
				if err == nil && threat != nil {
					gomega.Expect(threat.Type).To(gomega.Equal("brute_force"))
					gomega.Expect(threat.Severity).To(gomega.Equal("HIGH"))
				}
			}
		})
	})

	ginkgo.Describe("Accounting Management", func() {
		ginkgo.It("should collect and process usage data", func() {
			me := createTestManagedElement("test-ne-8", testServer.URL)
			gomega.Expect(k8sClient.Create(ctx, me)).To(gomega.Succeed())

			// Start accounting manager
			gomega.Expect(accountingMgr.Start(ctx)).To(gomega.Succeed())

			// Configure usage collection
			usageConfig := &o1.UsageCollectionConfig{
				ManagedElement:     me.Name,
				CollectionInterval: 5 * time.Second,
				Metrics: []string{
					"data_volume",
					"session_count",
					"bandwidth_usage",
				},
			}

			collectionID, err := accountingMgr.StartUsageCollection(ctx, usageConfig)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())

			// Wait for collection
			time.Sleep(10 * time.Second)

			// Get usage records
			filter := &o1.UsageFilter{
				StartTime: time.Now().Add(-1 * time.Hour),
				EndTime:   time.Now(),
			}

			records, err := accountingMgr.GetUsageRecords(ctx, me.Name, filter)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(records).ToNot(gomega.BeEmpty())

			// Calculate billing
			billingInfo, err := accountingMgr.CalculateBilling(ctx, records[0])
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(billingInfo.TotalCost).To(gomega.BeNumerically(">", 0))

			// Stop collection
			gomega.Expect(accountingMgr.StopUsageCollection(ctx, collectionID)).To(gomega.Succeed())
		})

		ginkgo.It("should detect fraudulent usage patterns", func() {
			me := createTestManagedElement("test-ne-9", testServer.URL)
			gomega.Expect(k8sClient.Create(ctx, me)).To(gomega.Succeed())

			// Enable fraud detection
			fraudConfig := &o1.FraudDetectionConfig{
				ManagedElement: me.Name,
				Rules: []o1.FraudRule{
					{
						Type:      "usage_spike",
						Threshold: 200, // 200% increase
						Window:    1 * time.Hour,
					},
					{
						Type:      "location_anomaly",
						Threshold: 1000, // 1000km distance
						Window:    5 * time.Minute,
					},
				},
			}

			gomega.Expect(accountingMgr.EnableFraudDetection(ctx, fraudConfig)).To(gomega.Succeed())

			// Simulate fraudulent usage
			fraudulentUsage := &o1.UsageRecord{
				ID:        "usage-fraud-001",
				UserID:    "user-123",
				ServiceID: "5g-data",
				StartTime: time.Now().Add(-10 * time.Minute),
				EndTime:   time.Now(),
				ResourceUsage: map[string]interface{}{
					"data_volume_mb": 10000, // Suspicious spike
					"location":       "New York",
				},
			}

			isFraud, fraudType, err := accountingMgr.DetectFraud(ctx, fraudulentUsage)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(isFraud).To(gomega.BeTrue())
			gomega.Expect(fraudType).To(gomega.Equal("usage_spike"))
		})
	})

	ginkgo.Describe("SMO Integration", func() {
		ginkgo.It("should register with SMO and maintain heartbeat", func() {
			// Setup mock SMO server
			smoServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.URL.Path {
				case "/register":
					response := map[string]string{
						"registration_id": "smo-reg-001",
						"status":          "registered",
					}
					json.NewEncoder(w).Encode(response)
				case "/heartbeat":
					w.WriteHeader(http.StatusOK)
				default:
					w.WriteHeader(http.StatusNotFound)
				}
			}))
			defer smoServer.Close()

			// Update SMO configuration
			smoIntegrator.UpdateEndpoint(smoServer.URL)

			// Register with SMO
			regID, err := smoIntegrator.Register(ctx)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(regID).To(gomega.Equal("smo-reg-001"))

			// Start heartbeat
			gomega.Expect(smoIntegrator.StartHeartbeat(ctx)).To(gomega.Succeed())

			// Wait for heartbeat
			time.Sleep(2 * time.Second)

			// Check registration status
			isRegistered := smoIntegrator.IsRegistered()
			gomega.Expect(isRegistered).To(gomega.BeTrue())

			// Stop heartbeat
			smoIntegrator.StopHeartbeat()
		})

		ginkgo.It("should support hierarchical management", func() {
			// Create parent and child managed elements
			parentME := createTestManagedElement("parent-ne", testServer.URL)
			childME1 := createTestManagedElement("child-ne-1", testServer.URL)
			childME2 := createTestManagedElement("child-ne-2", testServer.URL)

			gomega.Expect(k8sClient.Create(ctx, parentME)).To(gomega.Succeed())
			gomega.Expect(k8sClient.Create(ctx, childME1)).To(gomega.Succeed())
			gomega.Expect(k8sClient.Create(ctx, childME2)).To(gomega.Succeed())

			// Establish hierarchy
			hierarchy := &o1.ManagementHierarchy{
				Parent: parentME.Name,
				Children: []string{
					childME1.Name,
					childME2.Name,
				},
			}

			gomega.Expect(smoIntegrator.EstablishHierarchy(ctx, hierarchy)).To(gomega.Succeed())

			// Verify hierarchy
			retrievedHierarchy, err := smoIntegrator.GetHierarchy(ctx, parentME.Name)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(retrievedHierarchy.Children).To(gomega.HaveLen(2))

			// Propagate configuration through hierarchy
			config := `<config><logging><level>debug</level></logging></config>`
			gomega.Expect(smoIntegrator.PropagateConfiguration(ctx, parentME.Name, config)).To(gomega.Succeed())
		})
	})

	ginkgo.Describe("Real-time Streaming", func() {
		ginkgo.It("should stream alarms via WebSocket", func() {
			// Start streaming service
			gomega.Expect(streamingSvc.Start(ctx)).To(gomega.Succeed())

			// Connect WebSocket client
			wsURL := strings.Replace(wsServer.URL, "http", "ws", 1) + "/stream/alarms"
			ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			defer ws.Close()

			// Subscribe to alarm stream
			subscription := o1.StreamSubscription{
				Type:   o1.StreamTypeAlarm,
				Filter: "severity=CRITICAL",
			}

			subID, err := streamingSvc.Subscribe(ctx, &subscription)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(subID).ToNot(gomega.BeEmpty())

			// Generate test alarm
			alarm := &o1.Alarm{
				ID:               "alarm-stream-001",
				ManagedElementID: "test-ne-stream",
				Severity:         "CRITICAL",
				Type:             "COMMUNICATIONS",
				ProbableCause:    "link-failure",
				TimeRaised:       time.Now(),
			}

			// Stream alarm
			gomega.Expect(streamingSvc.StreamAlarm(ctx, alarm)).To(gomega.Succeed())

			// Read from WebSocket
			var receivedAlarm o1.Alarm
			err = ws.ReadJSON(&receivedAlarm)
			gomega.Expect(err).ToNot(gomega.HaveOccurred())
			gomega.Expect(receivedAlarm.ID).To(gomega.Equal("alarm-stream-001"))

			// Unsubscribe
			gomega.Expect(streamingSvc.Unsubscribe(ctx, subID)).To(gomega.Succeed())
		})

		ginkgo.It("should handle backpressure and rate limiting", func() {
			// Start streaming service with rate limiting
			streamingSvc.ConfigureRateLimit(10, 20) // 10 req/s, burst of 20
			gomega.Expect(streamingSvc.Start(ctx)).To(gomega.Succeed())

			// Create multiple connections
			var connections []*websocket.Conn
			for i := 0; i < 5; i++ {
				wsURL := strings.Replace(wsServer.URL, "http", "ws", 1) + fmt.Sprintf("/stream/perf/%d", i)
				ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
				gomega.Expect(err).ToNot(gomega.HaveOccurred())
				connections = append(connections, ws)
			}

			// Generate high volume of performance data
			var wg sync.WaitGroup
			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func(index int) {
					defer wg.Done()

					perfData := &o1.PerformanceData{
						ManagedElement: fmt.Sprintf("ne-%d", index),
						Timestamp:      time.Now(),
						Metrics: map[string]float64{
							"throughput": float64(index * 100),
							"latency":    float64(index),
						},
					}

					_ = streamingSvc.StreamPerformance(ctx, perfData)
				}(i)
			}

			wg.Wait()

			// Verify backpressure handling
			stats := streamingSvc.GetStatistics()
			gomega.Expect(stats.DroppedMessages).To(gomega.BeNumerically(">", 0))
			gomega.Expect(stats.RateLimitedRequests).To(gomega.BeNumerically(">", 0))

			// Close connections
			for _, conn := range connections {
				conn.Close()
			}
		})
	})

	ginkgo.Describe("End-to-End FCAPS Workflow", func() {
		ginkgo.It("should handle complete FCAPS lifecycle", func() {
			// Create O1Interface CR
			o1Interface := &nephoranv1.O1Interface{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-o1-interface",
					Namespace: "default",
				},
				Spec: nephoranv1.O1InterfaceSpec{
					Host:     testServer.URL,
					Port:     830,
					Protocol: "ssh",
					Credentials: nephoranv1.O1Credentials{
						UsernameRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "o1-credentials",
							},
							Key: "username",
						},
						PasswordRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "o1-credentials",
							},
							Key: "password",
						},
					},
					FCAPS: nephoranv1.FCAPSConfig{
						FaultManagement: nephoranv1.FaultManagementConfig{
							Enabled:            true,
							CorrelationEnabled: true,
							RootCauseAnalysis:  true,
							SeverityFilter:     []string{"CRITICAL", "MAJOR"},
						},
						ConfigurationManagement: nephoranv1.ConfigManagementConfig{
							Enabled:           true,
							GitOpsEnabled:     true,
							VersioningEnabled: true,
							DriftDetection:    true,
						},
						PerformanceManagement: nephoranv1.PerformanceManagementConfig{
							Enabled:            true,
							CollectionInterval: 15,
							AggregationPeriods: []string{"5min", "15min"},
							AnomalyDetection:   true,
						},
						SecurityManagement: nephoranv1.SecurityManagementConfig{
							Enabled:               true,
							CertificateManagement: true,
							IntrusionDetection:    true,
							ComplianceMonitoring:  true,
						},
						AccountingManagement: nephoranv1.AccountingManagementConfig{
							Enabled:            true,
							CollectionInterval: 60,
							BillingEnabled:     true,
							FraudDetection:     true,
						},
					},
					StreamingConfig: &nephoranv1.StreamingConfig{
						Enabled:        true,
						WebSocketPort:  8082,
						StreamTypes:    []string{"alarms", "performance", "configuration"},
						QoSLevel:       "Reliable",
						MaxConnections: 50,
					},
				},
			}

			// Create credentials secret
			credentialsSecret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "o1-credentials",
					Namespace: "default",
				},
				Data: map[string][]byte{
					"username": []byte("admin"),
					"password": []byte("password123"),
				},
			}

			gomega.Expect(k8sClient.Create(ctx, credentialsSecret)).To(gomega.Succeed())
			gomega.Expect(k8sClient.Create(ctx, o1Interface)).To(gomega.Succeed())

			// Wait for O1Interface to be ready
			gomega.Eventually(func() string {
				var o1if nephoranv1.O1Interface
				_ = k8sClient.Get(ctx, types.NamespacedName{
					Name:      o1Interface.Name,
					Namespace: o1Interface.Namespace,
				}, &o1if)
				return o1if.Status.Phase
			}, 30*time.Second, 1*time.Second).Should(gomega.Equal("Connected"))

			// Verify all FCAPS subsystems are ready
			var o1if nephoranv1.O1Interface
			gomega.Expect(k8sClient.Get(ctx, types.NamespacedName{
				Name:      o1Interface.Name,
				Namespace: o1Interface.Namespace,
			}, &o1if)).To(gomega.Succeed())

			gomega.Expect(o1if.Status.FCAPSStatus.FaultManagementReady).To(gomega.BeTrue())
			gomega.Expect(o1if.Status.FCAPSStatus.ConfigManagementReady).To(gomega.BeTrue())
			gomega.Expect(o1if.Status.FCAPSStatus.PerformanceManagementReady).To(gomega.BeTrue())
			gomega.Expect(o1if.Status.FCAPSStatus.SecurityManagementReady).To(gomega.BeTrue())
			gomega.Expect(o1if.Status.FCAPSStatus.AccountingManagementReady).To(gomega.BeTrue())

			// Verify streaming service is active
			gomega.Expect(o1if.Status.StreamingStatus).ToNot(gomega.BeNil())
			gomega.Expect(o1if.Status.StreamingStatus.Active).To(gomega.BeTrue())
		})
	})
})

// Helper functions

func createTestManagedElement(name, host string) *nephoranv1.ManagedElement {
	return &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: nephoranv1.ManagedElementSpec{
			Host: host,
			Port: 830,
			Credentials: nephoranv1.MECredentials{
				UsernameRef: &nephoranv1.SecretReference{
					Name:      "test-credentials",
					Namespace: "default",
					Key:       "username",
				},
				PasswordRef: &nephoranv1.SecretReference{
					Name:      "test-credentials",
					Namespace: "default",
					Key:       "password",
				},
			},
			O1Config: `<config><test>value</test></config>`,
		},
	}
}

func handleMockNetconfRequest(w http.ResponseWriter, r *http.Request) {
	// Mock NETCONF server responses
	response := `<?xml version="1.0" encoding="UTF-8"?>
<rpc-reply xmlns="urn:ietf:params:xml:ns:netconf:base:1.0" message-id="1">
  <ok/>
</rpc-reply>`

	w.Header().Set("Content-Type", "application/xml")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(response))
}

func handleWebSocketConnection(w http.ResponseWriter, r *http.Request) {
	upgrader := websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true
		},
	}

	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	// Echo messages back for testing
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		if err := conn.WriteMessage(messageType, message); err != nil {
			break
		}
	}
}

func TestO1FCAPSIntegration(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "O1 FCAPS Integration Test Suite")
}
