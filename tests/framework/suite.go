// Package framework provides comprehensive testing infrastructure for Nephoran Intent Operator.

// This includes unit testing, integration testing, load testing, and chaos engineering capabilities.

package framework

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/suite"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	crzap "sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// TestSuite provides comprehensive testing infrastructure.

type TestSuite struct {
	suite.Suite

	// Environment setup.

	testEnv *envtest.Environment

	cfg *rest.Config

	k8sClient client.Client

	// Test context and cancellation.

	ctx context.Context

	cancel context.CancelFunc

	// Test configuration.

	config *TestConfig

	// Mocking infrastructure.

	mocks *MockManager

	// Performance metrics.

	metrics *TestMetrics

	// Synchronization.

	mu sync.RWMutex
}

// TestConfig holds configuration for test execution.

type TestConfig struct {

	// Environment settings.

	UseExistingCluster bool

	CRDPath string

	BinaryAssetsPath string

	// Test execution settings.

	Timeout time.Duration

	CleanupTimeout time.Duration

	ParallelNodes int

	// Coverage settings.

	CoverageEnabled bool

	CoverageThreshold float64

	// Load testing settings.

	LoadTestEnabled bool

	MaxConcurrency int

	TestDuration time.Duration

	// Chaos testing settings.

	ChaosTestEnabled bool

	FailureRate float64

	// External services.

	WeaviateURL string

	LLMProviderURL string

	MockExternalAPIs bool
}

// DefaultTestConfig returns a default test configuration.

func DefaultTestConfig() *TestConfig {

	return &TestConfig{

		UseExistingCluster: false,

		CRDPath: filepath.Join("..", "..", "deployments", "crds"),

		BinaryAssetsPath: "",

		Timeout: 30 * time.Second,

		CleanupTimeout: 10 * time.Second,

		ParallelNodes: 4,

		CoverageEnabled: true,

		CoverageThreshold: 95.0,

		LoadTestEnabled: false,

		MaxConcurrency: 100,

		TestDuration: 5 * time.Minute,

		ChaosTestEnabled: false,

		FailureRate: 0.1,

		WeaviateURL: "http://localhost:8080",

		LLMProviderURL: "http://localhost:8081",

		MockExternalAPIs: true,
	}

}

// NewTestSuite creates a new comprehensive test suite.

func NewTestSuite(config *TestConfig) *TestSuite {

	if config == nil {

		config = DefaultTestConfig()

	}

	ctx, cancel := context.WithCancel(context.Background())

	return &TestSuite{

		config: config,

		ctx: ctx,

		cancel: cancel,

		mocks: NewMockManager(),

		metrics: NewTestMetrics(),
	}

}

// SetupSuite initializes the test environment.

func (ts *TestSuite) SetupSuite() {

	logf.SetLogger(crzap.New(crzap.UseDevMode(true)))

	ginkgo.By("Bootstrapping test environment")

	// Setup test environment.

	ts.testEnv = &envtest.Environment{

		CRDDirectoryPaths: []string{ts.config.CRDPath},

		ErrorIfCRDPathMissing: true,

		BinaryAssetsDirectory: ts.config.BinaryAssetsPath,

		UseExistingCluster: &ts.config.UseExistingCluster,
	}

	var err error

	ts.cfg, err = ts.testEnv.Start()

	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	gomega.Expect(ts.cfg).NotTo(gomega.BeNil())

	// Register our API types.

	err = nephranv1.AddToScheme(scheme.Scheme)

	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	// Create Kubernetes client.

	ts.k8sClient, err = client.New(ts.cfg, client.Options{Scheme: scheme.Scheme})

	gomega.Expect(err).NotTo(gomega.HaveOccurred())

	gomega.Expect(ts.k8sClient).NotTo(gomega.BeNil())

	// Initialize mocking infrastructure.

	ts.mocks.Initialize(ts.config)

	// Setup performance metrics collection.

	ts.metrics.Initialize()

	ginkgo.By("Test environment ready")

}

// TearDownSuite cleans up the test environment.

func (ts *TestSuite) TearDownSuite() {

	ginkgo.By("Tearing down test environment")

	// Cancel context.

	ts.cancel()

	// Generate test reports.

	ts.generateTestReports()

	// Cleanup mocks.

	ts.mocks.Cleanup()

	// Stop test environment.

	err := ts.testEnv.Stop()

	gomega.Expect(err).NotTo(gomega.HaveOccurred())

}

// SetupTest initializes each test case.

func (ts *TestSuite) SetupTest() {

	// Reset metrics for each test.

	ts.metrics.Reset()

	// Reset mocks.

	ts.mocks.Reset()

	// Create test-specific context.

	ts.ctx, ts.cancel = context.WithTimeout(context.Background(), ts.config.Timeout)

}

// TearDownTest cleans up after each test case.

func (ts *TestSuite) TearDownTest() {

	// Collect test metrics.

	ts.metrics.CollectTestMetrics()

	// Cancel test context.

	if ts.cancel != nil {

		ts.cancel()

	}

}

// GetK8sClient returns the Kubernetes client for testing.

func (ts *TestSuite) GetK8sClient() client.Client {

	return ts.k8sClient

}

// GetConfig returns the REST config for testing.

func (ts *TestSuite) GetConfig() *rest.Config {

	return ts.cfg

}

// GetTestConfig returns the test configuration.

func (ts *TestSuite) GetTestConfig() *TestConfig {

	return ts.config

}

// GetContext returns the test context.

func (ts *TestSuite) GetContext() context.Context {

	return ts.ctx

}

// GetMocks returns the mock manager.

func (ts *TestSuite) GetMocks() *MockManager {

	return ts.mocks

}

// GetMetrics returns the test metrics collector.

func (ts *TestSuite) GetMetrics() *TestMetrics {

	return ts.metrics

}

// RunLoadTest executes load testing scenarios.

func (ts *TestSuite) RunLoadTest(testFunc func() error) error {

	if !ts.config.LoadTestEnabled {

		return nil

	}

	ginkgo.By(fmt.Sprintf("Running load test with %d concurrent operations", ts.config.MaxConcurrency))

	return ts.metrics.ExecuteLoadTest(ts.config.MaxConcurrency, ts.config.TestDuration, testFunc)

}

// RunChaosTest executes chaos engineering scenarios.

func (ts *TestSuite) RunChaosTest(testFunc func() error) error {

	if !ts.config.ChaosTestEnabled {

		return nil

	}

	ginkgo.By(fmt.Sprintf("Running chaos test with %.1f%% failure rate", ts.config.FailureRate*100))

	return ts.mocks.InjectChaos(ts.config.FailureRate, testFunc)

}

// generateTestReports creates comprehensive test reports.

func (ts *TestSuite) generateTestReports() {

	ginkgo.By("Generating test reports")

	// Generate coverage report.

	if ts.config.CoverageEnabled {

		ts.generateCoverageReport()

	}

	// Generate performance report.

	ts.metrics.GenerateReport()

	// Generate mock interaction report.

	ts.mocks.GenerateReport()

}

// generateCoverageReport creates a code coverage report.

func (ts *TestSuite) generateCoverageReport() {

	// Implementation for coverage reporting.

	// This would integrate with Go's coverage tools.

	coverage := ts.metrics.GetCoveragePercentage()

	if coverage < ts.config.CoverageThreshold {

		ginkgo.Fail(fmt.Sprintf("Coverage %.2f%% is below threshold %.2f%%", coverage, ts.config.CoverageThreshold))

	}

	fmt.Printf("Code coverage: %.2f%%\n", coverage)

}

// RunIntegrationTests executes comprehensive integration test scenarios.

func RunIntegrationTests(t *testing.T, config *TestConfig) {

	if config == nil {

		config = DefaultTestConfig()

	}

	// Register Ginkgo fail handler.

	gomega.RegisterFailHandler(ginkgo.Fail)

	// Create test suite.

	testSuite := NewTestSuite(config)

	_ = testSuite // Use the test suite variable

	// Run Ginkgo tests.

	ginkgo.RunSpecs(t, "Nephoran Intent Operator Integration Test Suite")

}

// ValidateTestEnvironment ensures the test environment is properly configured.

func (ts *TestSuite) ValidateTestEnvironment() error {

	// Check if required environment variables are set.

	requiredEnvVars := []string{

		"KUBEBUILDER_ASSETS",
	}

	for _, envVar := range requiredEnvVars {

		if os.Getenv(envVar) == "" {

			return fmt.Errorf("required environment variable %s is not set", envVar)

		}

	}

	// Validate CRD paths exist.

	if _, err := os.Stat(ts.config.CRDPath); os.IsNotExist(err) {

		return fmt.Errorf("CRD path does not exist: %s", ts.config.CRDPath)

	}

	return nil

}

// GetTestNamespace returns a unique namespace for testing.

func (ts *TestSuite) GetTestNamespace() string {

	return fmt.Sprintf("nephran-test-%d", time.Now().Unix())

}
