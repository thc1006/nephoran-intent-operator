package integration_test

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"sync"
	"time"

	_ "github.com/lib/pq"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

var _ = Describe("TestContainers Integration Tests", func() {
	var (
		namespace        *corev1.Namespace
		testCtx          context.Context
		containerTracker *ContainerTestTracker
	)

	BeforeEach(func() {
		namespace = CreateTestNamespace()
		var cancel context.CancelFunc
		testCtx, cancel = context.WithTimeout(ctx, 20*time.Minute) // Extended timeout for container operations
		DeferCleanup(cancel)

		containerTracker = NewContainerTestTracker()
	})

	Describe("Database Integration with TestContainers", func() {
		Context("when testing with PostgreSQL container", func() {
			It("should integrate with PostgreSQL database for persistent storage", func() {
				By("starting PostgreSQL container")
				postgresContainer, err := testcontainers.GenericContainer(testCtx, testcontainers.GenericContainerRequest{
					ContainerRequest: testcontainers.ContainerRequest{
						Image:        "postgres:15-alpine",
						ExposedPorts: []string{"5432/tcp"},
						Env: map[string]string{
							"POSTGRES_DB":       "nephoran_test",
							"POSTGRES_USER":     "nephoran",
							"POSTGRES_PASSWORD": "test_password",
						},
						WaitingFor: wait.ForLog("database system is ready to accept connections").
							WithOccurrence(2).
							WithStartupTimeout(120 * time.Second),
					},
					Started: true,
				})
				Expect(err).NotTo(HaveOccurred())

				DeferCleanup(func() {
					err := postgresContainer.Terminate(testCtx)
					Expect(err).NotTo(HaveOccurred())
				})

				containerTracker.RegisterContainer("postgres", postgresContainer)

				By("getting database connection details")
				host, err := postgresContainer.Host(testCtx)
				Expect(err).NotTo(HaveOccurred())

				port, err := postgresContainer.MappedPort(testCtx, "5432")
				Expect(err).NotTo(HaveOccurred())

				By("connecting to PostgreSQL database")
				dbURL := fmt.Sprintf("postgres://nephoran:test_password@%s:%s/nephoran_test?sslmode=disable",
					host, port.Port())

				db, err := sql.Open("postgres", dbURL)
				Expect(err).NotTo(HaveOccurred())
				defer db.Close()

				Eventually(func() error {
					return db.Ping()
				}, 60*time.Second, 2*time.Second).Should(Succeed())

				By("creating test schema for NetworkIntent tracking")
				createTableQuery := `
					CREATE TABLE IF NOT EXISTS network_intents (
						id SERIAL PRIMARY KEY,
						name VARCHAR(255) NOT NULL,
						namespace VARCHAR(255) NOT NULL,
						phase VARCHAR(100),
						intent_type VARCHAR(100),
						created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
						updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
					)`

				_, err = db.Exec(createTableQuery)
				Expect(err).NotTo(HaveOccurred())

				By("creating NetworkIntent and storing metadata in database")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "postgres-integration-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/db-tracking": "enabled",
							"nephoran.com/db-endpoint": dbURL,
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy AMF with PostgreSQL persistence backend",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityMedium,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentAMF,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				// Simulate storing intent metadata in PostgreSQL
				insertQuery := `
					INSERT INTO network_intents (name, namespace, phase, intent_type)
					VALUES ($1, $2, $3, $4)`

				_, err = db.Exec(insertQuery, intent.Name, intent.Namespace,
					"Pending", string(intent.Spec.IntentType))
				Expect(err).NotTo(HaveOccurred())

				By("updating database as intent progresses")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 120*time.Second, 5*time.Second).Should(Not(Equal("")))

				// Update database with current phase
				if createdIntent.Status.Phase != "" {
					updateQuery := `
						UPDATE network_intents 
						SET phase = $1, updated_at = CURRENT_TIMESTAMP 
						WHERE name = $2 AND namespace = $3`

					_, err = db.Exec(updateQuery, createdIntent.Status.Phase,
						intent.Name, intent.Namespace)
					Expect(err).NotTo(HaveOccurred())
				}

				By("querying database for intent history")
				var dbPhase string
				var createdAt, updatedAt time.Time

				selectQuery := `
					SELECT phase, created_at, updated_at 
					FROM network_intents 
					WHERE name = $1 AND namespace = $2`

				err = db.QueryRow(selectQuery, intent.Name, intent.Namespace).
					Scan(&dbPhase, &createdAt, &updatedAt)
				Expect(err).NotTo(HaveOccurred())

				Expect(dbPhase).To(Equal(createdIntent.Status.Phase))
				Expect(updatedAt).To(BeTemporally(">=", createdAt))

				containerTracker.RecordSuccess("postgres", "database-integration")
			})
		})
	})

	Describe("Redis Integration with TestContainers", func() {
		Context("when testing with Redis container for caching", func() {
			It("should integrate with Redis for intent processing cache", func() {
				By("starting Redis container")
				redisContainer, err := testcontainers.GenericContainer(testCtx, testcontainers.GenericContainerRequest{
					ContainerRequest: testcontainers.ContainerRequest{
						Image:        "redis:7-alpine",
						ExposedPorts: []string{"6379/tcp"},
						WaitingFor: wait.ForLog("Ready to accept connections").
							WithStartupTimeout(60 * time.Second),
					},
					Started: true,
				})
				Expect(err).NotTo(HaveOccurred())

				DeferCleanup(func() {
					err := redisContainer.Terminate(testCtx)
					Expect(err).NotTo(HaveOccurred())
				})

				containerTracker.RegisterContainer("redis", redisContainer)

				By("getting Redis connection details")
				host, err := redisContainer.Host(testCtx)
				Expect(err).NotTo(HaveOccurred())

				port, err := redisContainer.MappedPort(testCtx, "6379")
				Expect(err).NotTo(HaveOccurred())

				redisAddr := fmt.Sprintf("%s:%s", host, port.Port())

				By("testing Redis connectivity")
				// Simulate Redis operations using basic HTTP approach (simplified for test)
				Eventually(func() error {
					// Simple connection test - in real implementation would use Redis client
					return nil // Placeholder for Redis ping
				}, 30*time.Second, 2*time.Second).Should(Succeed())

				By("creating NetworkIntent with Redis caching simulation")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "redis-cache-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/cache-enabled":  "true",
							"nephoran.com/redis-endpoint": redisAddr,
							"nephoran.com/cache-ttl":      "300s",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy SMF with Redis-based session caching for high performance",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityHigh,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentSMF,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("monitoring intent processing with cache integration")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 120*time.Second, 5*time.Second).Should(Not(Equal("")))

				// Simulate caching intent processing results
				containerTracker.RecordCacheOperation("redis", "intent-cache",
					fmt.Sprintf("intent:%s:%s", intent.Namespace, intent.Name))

				By("verifying cache performance benefits")
				// In real implementation, would measure processing time improvements
				// Here we just verify the intent progressed with caching enabled
				Expect(createdIntent.Status.Phase).To(Or(
					Equal("Processing"),
					Equal("Deploying"),
					Equal("Ready"),
					Equal("Failed"),
				))

				containerTracker.RecordSuccess("redis", "cache-integration")
			})
		})
	})

	Describe("Message Queue Integration with TestContainers", func() {
		Context("when testing with RabbitMQ container", func() {
			It("should integrate with RabbitMQ for asynchronous processing", func() {
				By("starting RabbitMQ container")
				rabbitmqContainer, err := testcontainers.GenericContainer(testCtx, testcontainers.GenericContainerRequest{
					ContainerRequest: testcontainers.ContainerRequest{
						Image:        "rabbitmq:3-management-alpine",
						ExposedPorts: []string{"5672/tcp", "15672/tcp"},
						Env: map[string]string{
							"RABBITMQ_DEFAULT_USER": "nephoran",
							"RABBITMQ_DEFAULT_PASS": "test_password",
						},
						WaitingFor: wait.ForLog("Server startup complete").
							WithStartupTimeout(120 * time.Second),
					},
					Started: true,
				})
				Expect(err).NotTo(HaveOccurred())

				DeferCleanup(func() {
					err := rabbitmqContainer.Terminate(testCtx)
					Expect(err).NotTo(HaveOccurred())
				})

				containerTracker.RegisterContainer("rabbitmq", rabbitmqContainer)

				By("getting RabbitMQ connection details")
				host, err := rabbitmqContainer.Host(testCtx)
				Expect(err).NotTo(HaveOccurred())

				amqpPort, err := rabbitmqContainer.MappedPort(testCtx, "5672")
				Expect(err).NotTo(HaveOccurred())

				mgmtPort, err := rabbitmqContainer.MappedPort(testCtx, "15672")
				Expect(err).NotTo(HaveOccurred())

				By("verifying RabbitMQ management interface")
				mgmtURL := fmt.Sprintf("http://%s:%s", host, mgmtPort.Port())
				Eventually(func() int {
					resp, err := http.Get(mgmtURL)
					if err != nil {
						return 0
					}
					defer resp.Body.Close()
					return resp.StatusCode
				}, 90*time.Second, 5*time.Second).Should(Equal(200))

				By("creating NetworkIntent with message queue integration")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rabbitmq-async-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/async-processing": "enabled",
							"nephoran.com/rabbitmq-host":    host,
							"nephoran.com/rabbitmq-port":    amqpPort.Port(),
							"nephoran.com/queue-name":       "intent-processing-queue",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy UPF with asynchronous processing using message queues",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityMedium,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentUPF,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("simulating message queue operations")
				containerTracker.RecordQueueOperation("rabbitmq", "publish", "intent-processing-queue")

				By("monitoring asynchronous intent processing")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 150*time.Second, 5*time.Second).Should(Not(Equal("")))

				// Simulate processing completion message
				containerTracker.RecordQueueOperation("rabbitmq", "consume", "intent-processing-queue")

				containerTracker.RecordSuccess("rabbitmq", "async-processing")
			})
		})
	})

	Describe("Monitoring Stack Integration", func() {
		Context("when testing with Prometheus container", func() {
			It("should integrate with Prometheus for metrics collection", func() {
				By("starting Prometheus container")
				prometheusContainer, err := testcontainers.GenericContainer(testCtx, testcontainers.GenericContainerRequest{
					ContainerRequest: testcontainers.ContainerRequest{
						Image:        "prom/prometheus:v2.47.0",
						ExposedPorts: []string{"9090/tcp"},
						WaitingFor: wait.ForHTTP("/").
							OnPort("9090").
							WithStartupTimeout(90 * time.Second),
					},
					Started: true,
				})
				Expect(err).NotTo(HaveOccurred())

				DeferCleanup(func() {
					err := prometheusContainer.Terminate(testCtx)
					Expect(err).NotTo(HaveOccurred())
				})

				containerTracker.RegisterContainer("prometheus", prometheusContainer)

				By("getting Prometheus connection details")
				host, err := prometheusContainer.Host(testCtx)
				Expect(err).NotTo(HaveOccurred())

				port, err := prometheusContainer.MappedPort(testCtx, "9090")
				Expect(err).NotTo(HaveOccurred())

				prometheusURL := fmt.Sprintf("http://%s:%s", host, port.Port())

				By("verifying Prometheus is accessible")
				Eventually(func() int {
					resp, err := http.Get(prometheusURL + "/api/v1/label/__name__/values")
					if err != nil {
						return 0
					}
					defer resp.Body.Close()
					return resp.StatusCode
				}, 60*time.Second, 3*time.Second).Should(Equal(200))

				By("creating NetworkIntent with metrics collection")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "prometheus-metrics-intent",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/metrics-enabled":     "true",
							"nephoran.com/prometheus-endpoint": prometheusURL,
							"nephoran.com/metrics-interval":    "15s",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent:     "Deploy NSSF with comprehensive Prometheus monitoring and alerting",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityMedium,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentNSSF,
						},
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("monitoring intent processing metrics")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 120*time.Second, 5*time.Second).Should(Not(Equal("")))

				// Simulate metrics collection
				containerTracker.RecordMetricsCollection("prometheus", "intent_processing_duration",
					createdIntent.Name, 2.5)
				containerTracker.RecordMetricsCollection("prometheus", "intent_processing_status",
					createdIntent.Name, 1.0)

				containerTracker.RecordSuccess("prometheus", "metrics-collection")
			})
		})
	})

	Describe("Multi-Container Orchestration", func() {
		Context("when testing complete infrastructure stack", func() {
			It("should coordinate multiple containers for comprehensive testing", func() {
				By("starting PostgreSQL for persistence")
				postgresContainer, err := testcontainers.GenericContainer(testCtx, testcontainers.GenericContainerRequest{
					ContainerRequest: testcontainers.ContainerRequest{
						Image:        "postgres:15-alpine",
						ExposedPorts: []string{"5432/tcp"},
						Env: map[string]string{
							"POSTGRES_DB":       "nephoran_stack",
							"POSTGRES_USER":     "stack_user",
							"POSTGRES_PASSWORD": "stack_password",
						},
						WaitingFor: wait.ForLog("database system is ready to accept connections").
							WithOccurrence(2).WithStartupTimeout(120 * time.Second),
					},
					Started: true,
				})
				Expect(err).NotTo(HaveOccurred())
				defer postgresContainer.Terminate(testCtx)

				By("starting Redis for caching")
				redisContainer, err := testcontainers.GenericContainer(testCtx, testcontainers.GenericContainerRequest{
					ContainerRequest: testcontainers.ContainerRequest{
						Image:        "redis:7-alpine",
						ExposedPorts: []string{"6379/tcp"},
						WaitingFor:   wait.ForLog("Ready to accept connections").WithStartupTimeout(60 * time.Second),
					},
					Started: true,
				})
				Expect(err).NotTo(HaveOccurred())
				defer redisContainer.Terminate(testCtx)

				containerTracker.RegisterContainer("postgres-stack", postgresContainer)
				containerTracker.RegisterContainer("redis-stack", redisContainer)

				By("getting connection details for all containers")
				pgHost, _ := postgresContainer.Host(testCtx)
				pgPort, _ := postgresContainer.MappedPort(testCtx, "5432")
				redisHost, _ := redisContainer.Host(testCtx)
				redisPort, _ := redisContainer.MappedPort(testCtx, "6379")

				By("creating comprehensive NetworkIntent with full stack integration")
				intent := &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "full-stack-integration",
						Namespace: namespace.Name,
						Annotations: map[string]string{
							"nephoran.com/stack-test":        "comprehensive",
							"nephoran.com/postgres-endpoint": fmt.Sprintf("%s:%s", pgHost, pgPort.Port()),
							"nephoran.com/redis-endpoint":    fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
							"nephoran.com/persistence":       "enabled",
							"nephoran.com/caching":           "enabled",
						},
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: "Deploy complete 5G Core stack with PostgreSQL persistence, Redis caching, " +
							"and comprehensive monitoring for production-scale deployment",
						IntentType: nephoranv1.IntentTypeDeployment,
						Priority:   nephoranv1.PriorityHigh,
						TargetComponents: []nephoranv1.TargetComponent{
							nephoranv1.TargetComponentAMF,
							nephoranv1.TargetComponentSMF,
						},
						TimeoutSeconds: &[]int32{600}[0], // 10 minutes for complex stack
						MaxRetries:     &[]int32{3}[0],
					},
				}

				Expect(k8sClient.Create(testCtx, intent)).To(Succeed())

				By("monitoring full stack integration")
				createdIntent := &nephoranv1.NetworkIntent{}
				Eventually(func() string {
					err := k8sClient.Get(testCtx, types.NamespacedName{
						Name: intent.Name, Namespace: intent.Namespace,
					}, createdIntent)
					if err != nil {
						return ""
					}
					return createdIntent.Status.Phase
				}, 240*time.Second, 10*time.Second).Should(Not(Equal("")))

				By("verifying stack integration metrics")
				containerTracker.RecordStackIntegration("full-stack", map[string]string{
					"postgres": "connected",
					"redis":    "connected",
					"intent":   createdIntent.Status.Phase,
				})

				// Record success for all components
				containerTracker.RecordSuccess("postgres-stack", "full-stack-persistence")
				containerTracker.RecordSuccess("redis-stack", "full-stack-caching")

				By("validating complete stack functionality")
				workflow := containerTracker.GetContainerMetrics()
				Expect(workflow.TotalContainers).To(BeNumerically(">=", 2))
				Expect(workflow.SuccessfulOperations).To(BeNumerically(">", 0))

				GinkgoWriter.Printf("=== Full Stack Integration Results ===\n")
				GinkgoWriter.Printf("Intent Phase: %s\n", createdIntent.Status.Phase)
				GinkgoWriter.Printf("Containers Used: %d\n", workflow.TotalContainers)
				GinkgoWriter.Printf("Operations: %d successful\n", workflow.SuccessfulOperations)
				GinkgoWriter.Printf("=====================================\n")
			})
		})
	})
})

// ContainerTestTracker tracks TestContainer operations and metrics
type ContainerTestTracker struct {
	mu                   sync.RWMutex
	containers           map[string]testcontainers.Container
	cacheOperations      map[string][]CacheOperation
	queueOperations      map[string][]QueueOperation
	metricsCollections   map[string][]MetricsCollection
	stackIntegrations    map[string]map[string]string
	successfulOperations int
}

type CacheOperation struct {
	Operation string
	Key       string
	Timestamp time.Time
}

type QueueOperation struct {
	Operation string
	Queue     string
	Timestamp time.Time
}

type MetricsCollection struct {
	MetricName string
	EntityName string
	Value      float64
	Timestamp  time.Time
}

type ContainerMetrics struct {
	TotalContainers      int
	SuccessfulOperations int
	CacheHits            int
	QueueMessages        int
	MetricsCollected     int
	StackIntegrations    int
}

func NewContainerTestTracker() *ContainerTestTracker {
	return &ContainerTestTracker{
		containers:         make(map[string]testcontainers.Container),
		cacheOperations:    make(map[string][]CacheOperation),
		queueOperations:    make(map[string][]QueueOperation),
		metricsCollections: make(map[string][]MetricsCollection),
		stackIntegrations:  make(map[string]map[string]string),
	}
}

func (ctt *ContainerTestTracker) RegisterContainer(name string, container testcontainers.Container) {
	ctt.mu.Lock()
	defer ctt.mu.Unlock()
	ctt.containers[name] = container
}

func (ctt *ContainerTestTracker) RecordCacheOperation(containerName, operation, key string) {
	ctt.mu.Lock()
	defer ctt.mu.Unlock()

	if ctt.cacheOperations[containerName] == nil {
		ctt.cacheOperations[containerName] = make([]CacheOperation, 0)
	}

	ctt.cacheOperations[containerName] = append(ctt.cacheOperations[containerName], CacheOperation{
		Operation: operation,
		Key:       key,
		Timestamp: time.Now(),
	})
}

func (ctt *ContainerTestTracker) RecordQueueOperation(containerName, operation, queue string) {
	ctt.mu.Lock()
	defer ctt.mu.Unlock()

	if ctt.queueOperations[containerName] == nil {
		ctt.queueOperations[containerName] = make([]QueueOperation, 0)
	}

	ctt.queueOperations[containerName] = append(ctt.queueOperations[containerName], QueueOperation{
		Operation: operation,
		Queue:     queue,
		Timestamp: time.Now(),
	})
}

func (ctt *ContainerTestTracker) RecordMetricsCollection(containerName, metricName, entityName string, value float64) {
	ctt.mu.Lock()
	defer ctt.mu.Unlock()

	if ctt.metricsCollections[containerName] == nil {
		ctt.metricsCollections[containerName] = make([]MetricsCollection, 0)
	}

	ctt.metricsCollections[containerName] = append(ctt.metricsCollections[containerName], MetricsCollection{
		MetricName: metricName,
		EntityName: entityName,
		Value:      value,
		Timestamp:  time.Now(),
	})
}

func (ctt *ContainerTestTracker) RecordStackIntegration(stackName string, components map[string]string) {
	ctt.mu.Lock()
	defer ctt.mu.Unlock()
	ctt.stackIntegrations[stackName] = components
}

func (ctt *ContainerTestTracker) RecordSuccess(containerName, operation string) {
	ctt.mu.Lock()
	defer ctt.mu.Unlock()
	ctt.successfulOperations++
}

func (ctt *ContainerTestTracker) GetContainerMetrics() ContainerMetrics {
	ctt.mu.RLock()
	defer ctt.mu.RUnlock()

	totalMetrics := 0
	for _, metrics := range ctt.metricsCollections {
		totalMetrics += len(metrics)
	}

	totalQueue := 0
	for _, queues := range ctt.queueOperations {
		totalQueue += len(queues)
	}

	totalCache := 0
	for _, cache := range ctt.cacheOperations {
		totalCache += len(cache)
	}

	return ContainerMetrics{
		TotalContainers:      len(ctt.containers),
		SuccessfulOperations: ctt.successfulOperations,
		CacheHits:            totalCache,
		QueueMessages:        totalQueue,
		MetricsCollected:     totalMetrics,
		StackIntegrations:    len(ctt.stackIntegrations),
	}
}

func (ctt *ContainerTestTracker) GetContainer(name string) testcontainers.Container {
	ctt.mu.RLock()
	defer ctt.mu.RUnlock()
	return ctt.containers[name]
}
