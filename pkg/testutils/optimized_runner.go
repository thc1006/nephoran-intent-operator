
package testutils



import (

	"context"

	"fmt"

	"os"

	"path/filepath"

	"runtime"

	"sync"

	"testing"

	"time"

)



// OptimizedTestRunner provides an optimized test execution environment.

type OptimizedTestRunner struct {

	optimizer        *WindowsTestOptimizer

	concurrencyMgr   *WindowsConcurrencyManager

	batchProcessor   *BatchFileProcessor

	resourcePool     *TestResourcePool

	metricsCollector *TestMetricsCollector

}



// NewOptimizedTestRunner creates a new optimized test runner.

func NewOptimizedTestRunner() *OptimizedTestRunner {

	return &OptimizedTestRunner{

		optimizer:        NewWindowsTestOptimizer(),

		concurrencyMgr:   NewWindowsConcurrencyManager(),

		batchProcessor:   NewBatchFileProcessor(),

		resourcePool:     NewTestResourcePool(),

		metricsCollector: NewTestMetricsCollector(),

	}

}



// RunOptimizedTest runs a test with optimizations.

func (r *OptimizedTestRunner) RunOptimizedTest(t *testing.T, testFunc func(*testing.T, *TestContext)) {

	startTime := time.Now()



	// Acquire concurrency slot.

	releaseSlot := r.concurrencyMgr.AcquireSlot(t.Name())

	defer releaseSlot()



	// Create optimized test context.

	ctx := &TestContext{

		T:           t,

		TempDir:     r.optimizer.OptimizedTempDir(t, "test"),

		Optimizer:   r.optimizer,

		BatchProc:   r.batchProcessor,

		ResourceMgr: r.resourcePool,

	}



	// Set up cleanup.

	defer func() {

		r.metricsCollector.RecordTestCompletion(t.Name(), time.Since(startTime))

		ctx.Cleanup()

	}()



	// Run the test.

	testFunc(t, ctx)

}



// TestContext provides optimized test context.

type TestContext struct {

	T           *testing.T

	TempDir     string

	Optimizer   *WindowsTestOptimizer

	BatchProc   *BatchFileProcessor

	ResourceMgr *TestResourcePool

	cleanupFns  []func()

	mu          sync.Mutex

}



// CreateTempFile creates a temporary file with optimizations.

func (tc *TestContext) CreateTempFile(name string, content []byte) string {

	filePath := filepath.Join(tc.TempDir, name)

	if err := tc.Optimizer.OptimizedFileWrite(filePath, content); err != nil {

		tc.T.Fatalf("Failed to create temp file %s: %v", name, err)

	}

	return filePath

}



// CreateTempFiles creates multiple files efficiently.

func (tc *TestContext) CreateTempFiles(files map[string][]byte) map[string]string {

	results := make(map[string]string)

	if err := tc.BatchProc.CreateFiles(tc.TempDir, files); err != nil {

		tc.T.Fatalf("Failed to create temp files: %v", err)

	}



	for name := range files {

		results[name] = filepath.Join(tc.TempDir, name)

	}

	return results

}



// GetOptimizedContext creates a context with optimized timeout.

func (tc *TestContext) GetOptimizedContext(baseTimeout time.Duration) (context.Context, context.CancelFunc) {

	return tc.Optimizer.OptimizedContextTimeout(context.Background(), baseTimeout)

}



// AddCleanup adds a cleanup function.

func (tc *TestContext) AddCleanup(fn func()) {

	tc.mu.Lock()

	defer tc.mu.Unlock()

	tc.cleanupFns = append(tc.cleanupFns, fn)

}



// Cleanup performs all cleanup operations.

func (tc *TestContext) Cleanup() {

	tc.mu.Lock()

	defer tc.mu.Unlock()



	// Execute cleanup functions in reverse order.

	for i := len(tc.cleanupFns) - 1; i >= 0; i-- {

		if fn := tc.cleanupFns[i]; fn != nil {

			fn()

		}

	}

	tc.cleanupFns = nil



	// Cleanup optimizer.

	tc.Optimizer.Cleanup()

}



// BatchFileProcessor handles efficient batch file operations.

type BatchFileProcessor struct {

	concurrencyLimit chan struct{}

}



// NewBatchFileProcessor creates a new batch file processor.

func NewBatchFileProcessor() *BatchFileProcessor {

	limit := runtime.NumCPU()

	if runtime.GOOS == "windows" {

		limit = max(1, limit/2)

	}



	return &BatchFileProcessor{

		concurrencyLimit: make(chan struct{}, limit),

	}

}



// CreateFiles creates multiple files concurrently with optimizations.

func (b *BatchFileProcessor) CreateFiles(baseDir string, files map[string][]byte) error {

	if err := os.MkdirAll(baseDir, 0o755); err != nil {

		return fmt.Errorf("failed to create base directory: %w", err)

	}



	// Use a bounded goroutine pool.

	var wg sync.WaitGroup

	errChan := make(chan error, len(files))



	for filename, content := range files {

		wg.Add(1)

		go func(fname string, data []byte) {

			defer wg.Done()



			// Acquire concurrency slot.

			b.concurrencyLimit <- struct{}{}

			defer func() { <-b.concurrencyLimit }()



			filePath := filepath.Join(baseDir, fname)



			// Ensure subdirectory exists.

			dir := filepath.Dir(filePath)

			if err := os.MkdirAll(dir, 0o755); err != nil {

				errChan <- fmt.Errorf("failed to create directory for %s: %w", fname, err)

				return

			}



			// Write file with platform-specific optimizations.

			if err := writeFileOptimized(filePath, data); err != nil {

				errChan <- fmt.Errorf("failed to write file %s: %w", fname, err)

			}

		}(filename, content)

	}



	wg.Wait()

	close(errChan)



	// Check for errors.

	for err := range errChan {

		if err != nil {

			return err

		}

	}



	return nil

}



// writeFileOptimized writes a file with platform-specific optimizations.

func writeFileOptimized(filePath string, content []byte) error {

	if runtime.GOOS == "windows" {

		// Use atomic write on Windows for better reliability.

		tempFile := filePath + ".tmp"

		if err := os.WriteFile(tempFile, content, 0o640); err != nil {

			return err

		}

		return os.Rename(tempFile, filePath)

	}

	return os.WriteFile(filePath, content, 0o640)

}



// TestResourcePool manages reusable test resources.

type TestResourcePool struct {

	directories sync.Map

	files       sync.Map

	mu          sync.RWMutex

}



// NewTestResourcePool creates a new resource pool.

func NewTestResourcePool() *TestResourcePool {

	return &TestResourcePool{}

}



// GetCachedDirectory returns a cached directory or creates a new one.

func (p *TestResourcePool) GetCachedDirectory(key string) (string, bool) {

	if val, ok := p.directories.Load(key); ok {

		return val.(string), true

	}

	return "", false

}



// CacheDirectory caches a directory for reuse.

func (p *TestResourcePool) CacheDirectory(key, path string) {

	p.directories.Store(key, path)

}



// GetCachedFile returns cached file content.

func (p *TestResourcePool) GetCachedFile(path string) ([]byte, bool) {

	if val, ok := p.files.Load(path); ok {

		return val.([]byte), true

	}

	return nil, false

}



// CacheFile caches file content for reuse.

func (p *TestResourcePool) CacheFile(path string, content []byte) {

	// Only cache files up to 1MB to avoid memory issues.

	if len(content) <= 1024*1024 {

		p.files.Store(path, content)

	}

}



// Clear clears all cached resources.

func (p *TestResourcePool) Clear() {

	p.directories = sync.Map{}

	p.files = sync.Map{}

}



// TestMetricsCollector collects test performance metrics.

type TestMetricsCollector struct {

	testDurations sync.Map

	mu            sync.RWMutex

}



// NewTestMetricsCollector creates a new metrics collector.

func NewTestMetricsCollector() *TestMetricsCollector {

	return &TestMetricsCollector{}

}



// RecordTestCompletion records test completion time.

func (m *TestMetricsCollector) RecordTestCompletion(testName string, duration time.Duration) {

	m.testDurations.Store(testName, duration)

}



// GetAverageDuration gets average duration for tests with given prefix.

func (m *TestMetricsCollector) GetAverageDuration(prefix string) time.Duration {

	var total time.Duration

	var count int



	m.testDurations.Range(func(key, value interface{}) bool {

		if testName, ok := key.(string); ok {

			if len(testName) >= len(prefix) && testName[:len(prefix)] == prefix {

				if duration, ok := value.(time.Duration); ok {

					total += duration

					count++

				}

			}

		}

		return true

	})



	if count == 0 {

		return 0

	}

	return total / time.Duration(count)

}



// PrintSummary prints a summary of test performance.

func (m *TestMetricsCollector) PrintSummary() {

	fmt.Printf("\n=== Test Performance Summary ===\n")

	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	fmt.Printf("CPUs: %d\n", runtime.NumCPU())



	var slowestTest string

	var slowestDuration time.Duration

	var totalDuration time.Duration

	var testCount int



	m.testDurations.Range(func(key, value interface{}) bool {

		if testName, ok := key.(string); ok {

			if duration, ok := value.(time.Duration); ok {

				totalDuration += duration

				testCount++



				if duration > slowestDuration {

					slowestDuration = duration

					slowestTest = testName

				}

			}

		}

		return true

	})



	if testCount > 0 {

		avgDuration := totalDuration / time.Duration(testCount)

		fmt.Printf("Total tests: %d\n", testCount)

		fmt.Printf("Total duration: %v\n", totalDuration)

		fmt.Printf("Average duration: %v\n", avgDuration)

		fmt.Printf("Slowest test: %s (%v)\n", slowestTest, slowestDuration)

	}

	fmt.Printf("================================\n\n")

}



// Optimized test helper functions.



// WithOptimizedTest runs a test with optimizations.

func WithOptimizedTest(t *testing.T, testFunc func(*testing.T, *TestContext)) {

	runner := NewOptimizedTestRunner()

	runner.RunOptimizedTest(t, testFunc)

}



// WithOptimizedTimeout creates a context with optimized timeout for the platform.

func WithOptimizedTimeout(ctx context.Context, baseTimeout time.Duration) (context.Context, context.CancelFunc) {

	optimizedTimeout := baseTimeout

	if runtime.GOOS == "windows" {

		// Increase timeout by 50% on Windows.

		optimizedTimeout = time.Duration(float64(baseTimeout) * 1.5)

	}

	return context.WithTimeout(ctx, optimizedTimeout)

}



// CreateOptimizedTempDir creates a temporary directory optimized for the platform.

func CreateOptimizedTempDir(t testing.TB, prefix string) string {

	if runtime.GOOS == "windows" {

		// Use shorter paths on Windows.

		tempBase := os.TempDir()

		shortPrefix := prefix

		if len(prefix) > 8 {

			shortPrefix = prefix[:8]

		}

		tempDir := filepath.Join(tempBase, fmt.Sprintf("%s-%d", shortPrefix, time.Now().UnixNano()%1000000))



		if err := os.MkdirAll(tempDir, 0o755); err != nil {

			t.Fatalf("Failed to create temp dir: %v", err)

		}



		// Clean up on test completion.

		t.Cleanup(func() {

			os.RemoveAll(tempDir)

		})



		return tempDir

	}



	return t.TempDir()

}



// BatchCreateFiles creates multiple files efficiently.

func BatchCreateFiles(baseDir string, files map[string][]byte) error {

	processor := NewBatchFileProcessor()

	return processor.CreateFiles(baseDir, files)

}

