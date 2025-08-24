package testutils

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

// TestOptimizedVsStandard demonstrates the performance difference between standard and optimized test approaches
func TestOptimizedVsStandard(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance comparison test in short mode")
	}

	// Only run detailed comparison on Windows
	if runtime.GOOS == "windows" {
		t.Run("PerformanceComparison", func(t *testing.T) {
			runPerformanceComparison(t)
		})
	}

	t.Run("StandardApproach", func(t *testing.T) {
		runStandardTestExample(t)
	})

	t.Run("OptimizedApproach", func(t *testing.T) {
		runOptimizedTestExample(t)
	})
}

// runPerformanceComparison measures the performance difference
func runPerformanceComparison(t *testing.T) {
	const numFiles = 50
	const numIterations = 5

	// Measure standard approach
	standardTimes := make([]time.Duration, numIterations)
	for i := 0; i < numIterations; i++ {
		start := time.Now()
		runStandardFileOperations(t, numFiles)
		standardTimes[i] = time.Since(start)
	}

	// Measure optimized approach
	optimizedTimes := make([]time.Duration, numIterations)
	for i := 0; i < numIterations; i++ {
		start := time.Now()
		runOptimizedFileOperations(t, numFiles)
		optimizedTimes[i] = time.Since(start)
	}

	// Calculate averages
	var standardTotal, optimizedTotal time.Duration
	for i := 0; i < numIterations; i++ {
		standardTotal += standardTimes[i]
		optimizedTotal += optimizedTimes[i]
	}

	standardAvg := standardTotal / time.Duration(numIterations)
	optimizedAvg := optimizedTotal / time.Duration(numIterations)

	t.Logf("Performance comparison for %d files:", numFiles)
	t.Logf("Standard approach average: %v", standardAvg)
	t.Logf("Optimized approach average: %v", optimizedAvg)

	if optimizedAvg > 0 {
		speedup := float64(standardAvg) / float64(optimizedAvg)
		t.Logf("Speedup factor: %.2fx", speedup)

		// On Windows, we expect at least some improvement
		if runtime.GOOS == "windows" && speedup < 1.1 {
			t.Logf("Warning: Expected better speedup on Windows, got %.2fx", speedup)
		}
	}
}

// runStandardTestExample demonstrates standard test patterns
func runStandardTestExample(t *testing.T) {
	// Standard approach: create temp dir and files individually
	tempDir := t.TempDir()

	// Create test configuration files one by one
	configFiles := map[string]string{
		"config.json":        `{"debug": true, "port": 8080}`,
		"database.yaml":      "host: localhost\nport: 5432\nname: testdb",
		"logging.properties": "log.level=DEBUG\nlog.file=/tmp/app.log",
		"secrets.env":        "API_KEY=test123\nDB_PASSWORD=secret",
	}

	for filename, content := range configFiles {
		filePath := filepath.Join(tempDir, filename)
		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create %s: %v", filename, err)
		}
	}

	// Standard timeout (might be too short on Windows)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Simulate test operations
	if err := simulateFileProcessing(ctx, tempDir); err != nil {
		t.Fatalf("File processing failed: %v", err)
	}

	// Manual verification of created files
	for filename := range configFiles {
		filePath := filepath.Join(tempDir, filename)
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			t.Errorf("Expected file %s was not created", filename)
		}
	}
}

// runOptimizedTestExample demonstrates optimized test patterns
func runOptimizedTestExample(t *testing.T) {
	// Use optimized test runner
	WithOptimizedTest(t, func(t *testing.T, ctx *TestContext) {
		// Create test configuration files efficiently using batch creation
		configFiles := map[string][]byte{
			"config.json":        []byte(`{"debug": true, "port": 8080}`),
			"database.yaml":      []byte("host: localhost\nport: 5432\nname: testdb"),
			"logging.properties": []byte("log.level=DEBUG\nlog.file=/tmp/app.log"),
			"secrets.env":        []byte("API_KEY=test123\nDB_PASSWORD=secret"),
		}

		// Batch file creation is much faster
		filePaths := ctx.CreateTempFiles(configFiles)

		// Platform-optimized timeout (automatically adjusted for Windows)
		testCtx, cancel := ctx.GetOptimizedContext(2 * time.Second)
		defer cancel()

		// Simulate test operations with optimized context
		if err := simulateFileProcessing(testCtx, ctx.TempDir); err != nil {
			t.Fatalf("File processing failed: %v", err)
		}

		// Verification is built into the optimized context
		for filename := range configFiles {
			if _, exists := filePaths[filename]; !exists {
				t.Errorf("Expected file %s was not created", filename)
			}
		}
	})
}

// runStandardFileOperations simulates standard file operations
func runStandardFileOperations(t testing.TB, numFiles int) {
	tempDir := t.TempDir()

	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("file_%03d.txt", i)
		content := fmt.Sprintf("Content for file %d\nWith multiple lines\nAnd some data", i)
		filePath := filepath.Join(tempDir, filename)

		if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %d: %v", i, err)
		}

		// Simulate some processing
		if _, err := os.ReadFile(filePath); err != nil {
			t.Fatalf("Failed to read file %d: %v", i, err)
		}
	}
}

// runOptimizedFileOperations simulates optimized file operations
func runOptimizedFileOperations(t testing.TB, numFiles int) {
	optimizer := NewWindowsTestOptimizer()
	defer optimizer.Cleanup()

	tempDir := optimizer.OptimizedTempDir(t, "perf-test")

	// Prepare all files for batch creation
	files := make(map[string][]byte)
	for i := 0; i < numFiles; i++ {
		filename := fmt.Sprintf("file_%03d.txt", i)
		content := fmt.Sprintf("Content for file %d\nWith multiple lines\nAnd some data", i)
		files[filename] = []byte(content)
	}

	// Batch create all files
	if err := optimizer.BulkFileCreation(tempDir, files); err != nil {
		t.Fatalf("Failed to create files: %v", err)
	}

	// Batch read operations with caching
	for filename := range files {
		filePath := filepath.Join(tempDir, filename)
		if _, err := optimizer.OptimizedFileRead(filePath); err != nil {
			t.Fatalf("Failed to read file %s: %v", filename, err)
		}
	}
}

// simulateFileProcessing simulates processing of files (placeholder for real logic)
func simulateFileProcessing(ctx context.Context, dir string) error {
	// Simulate some processing time
	select {
	case <-time.After(100 * time.Millisecond):
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// BenchmarkFileOperationComparison benchmarks the difference in file operations
func BenchmarkFileOperationComparison(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("Benchmark focused on Windows performance")
	}

	b.Run("Standard_10_Files", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runStandardFileOperations(b, 10)
		}
	})

	b.Run("Optimized_10_Files", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runOptimizedFileOperations(b, 10)
		}
	})

	b.Run("Standard_50_Files", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runStandardFileOperations(b, 50)
		}
	})

	b.Run("Optimized_50_Files", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			runOptimizedFileOperations(b, 50)
		}
	})
}

// TestConcurrencyOptimization demonstrates concurrency management benefits
func TestConcurrencyOptimization(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Concurrency optimization test focused on Windows")
	}

	t.Run("WithoutConcurrencyManagement", func(t *testing.T) {
		// This test demonstrates the problems without concurrency management
		start := time.Now()

		// Create many temporary files concurrently (can overwhelm Windows)
		const numGoroutines = 20
		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				tempDir := t.TempDir()
				for j := 0; j < 5; j++ {
					filename := fmt.Sprintf("concurrent_%d_%d.txt", id, j)
					filePath := filepath.Join(tempDir, filename)
					err := os.WriteFile(filePath, []byte("concurrent content"), 0644)
					results <- err
				}
			}(i)
		}

		// Wait for all operations
		for i := 0; i < numGoroutines; i++ {
			if err := <-results; err != nil {
				t.Errorf("Concurrent operation failed: %v", err)
			}
		}

		unmanaged := time.Since(start)
		t.Logf("Unmanaged concurrency took: %v", unmanaged)
	})

	t.Run("WithConcurrencyManagement", func(t *testing.T) {
		// This test shows the benefits of concurrency management
		start := time.Now()

		concurrencyMgr := NewWindowsConcurrencyManager()
		optimizer := NewWindowsTestOptimizer()
		defer optimizer.Cleanup()

		const numGoroutines = 20
		results := make(chan error, numGoroutines)

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				// Acquire concurrency slot
				testName := fmt.Sprintf("managed-test-%d", id)
				release := concurrencyMgr.AcquireSlot(testName)
				defer release()

				tempDir := optimizer.OptimizedTempDir(t, fmt.Sprintf("managed-%d", id))

				// Create files using batch operations
				files := make(map[string][]byte)
				for j := 0; j < 5; j++ {
					filename := fmt.Sprintf("concurrent_%d_%d.txt", id, j)
					files[filename] = []byte("concurrent content")
				}

				err := optimizer.BulkFileCreation(tempDir, files)
				results <- err
			}(i)
		}

		// Wait for all operations
		for i := 0; i < numGoroutines; i++ {
			if err := <-results; err != nil {
				t.Errorf("Managed concurrent operation failed: %v", err)
			}
		}

		managed := time.Since(start)
		t.Logf("Managed concurrency took: %v", managed)
	})
}

// TestResourceCaching demonstrates the benefits of resource caching
func TestResourceCaching(t *testing.T) {
	t.Run("WithoutCaching", func(t *testing.T) {
		start := time.Now()

		// Repeatedly create the same test data
		for i := 0; i < 10; i++ {
			tempDir := t.TempDir()
			
			// Create the same files repeatedly
			commonFiles := []string{"config.json", "data.txt", "schema.yaml"}
			for _, filename := range commonFiles {
				filePath := filepath.Join(tempDir, filename)
				content := fmt.Sprintf("Content for %s iteration %d", filename, i)
				if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
					t.Fatalf("Failed to create %s: %v", filename, err)
				}
			}
		}

		withoutCaching := time.Since(start)
		t.Logf("Without caching took: %v", withoutCaching)
	})

	t.Run("WithCaching", func(t *testing.T) {
		start := time.Now()

		optimizer := NewWindowsTestOptimizer()
		defer optimizer.Cleanup()

		// Prepare common test data once
		commonFiles := map[string][]byte{
			"config.json": []byte(`{"common": "config"}`),
			"data.txt":    []byte("common test data"),
			"schema.yaml": []byte("common: schema"),
		}

		// Use cached data across iterations
		for i := 0; i < 10; i++ {
			tempDir := optimizer.OptimizedTempDir(t, fmt.Sprintf("cached-%d", i))
			
			// Efficiently create files using cached data
			if err := optimizer.BulkFileCreation(tempDir, commonFiles); err != nil {
				t.Fatalf("Failed to create cached files: %v", err)
			}
		}

		withCaching := time.Since(start)
		t.Logf("With caching took: %v", withCaching)
	})
}

// TestTimeoutOptimization demonstrates platform-aware timeout handling
func TestTimeoutOptimization(t *testing.T) {
	optimizer := NewWindowsTestOptimizer()
	defer optimizer.Cleanup()

	baseTimeout := 1 * time.Second

	t.Run("StandardTimeout", func(t *testing.T) {
		// Standard timeout might be too short on Windows
		ctx, cancel := context.WithTimeout(context.Background(), baseTimeout)
		defer cancel()

		// Simulate operation that might take longer on Windows
		select {
		case <-time.After(time.Duration(float64(baseTimeout) * 1.2)): // 20% longer
			t.Log("Operation completed")
		case <-ctx.Done():
			if runtime.GOOS == "windows" {
				t.Log("Expected timeout on Windows with standard timeout")
			} else {
				t.Error("Unexpected timeout")
			}
		}
	})

	t.Run("OptimizedTimeout", func(t *testing.T) {
		// Optimized timeout accounts for platform differences
		ctx, cancel := optimizer.OptimizedContextTimeout(context.Background(), baseTimeout)
		defer cancel()

		// Same operation with platform-optimized timeout
		select {
		case <-time.After(time.Duration(float64(baseTimeout) * 1.2)): // 20% longer
			t.Log("Operation completed with optimized timeout")
		case <-ctx.Done():
			t.Error("Unexpected timeout with optimized timeout")
		}
	})
}