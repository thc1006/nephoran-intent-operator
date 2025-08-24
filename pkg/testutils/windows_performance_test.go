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

// BenchmarkWindowsFileOperations benchmarks file operations with and without optimizations
func BenchmarkWindowsFileOperations(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("Windows-specific benchmark")
	}

	b.Run("StandardFileWrite", func(b *testing.B) {
		tempDir := b.TempDir()
		content := []byte("test content for benchmarking file write operations")
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filePath := filepath.Join(tempDir, fmt.Sprintf("test-%d.txt", i))
			if err := os.WriteFile(filePath, content, 0644); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("OptimizedFileWrite", func(b *testing.B) {
		optimizer := NewWindowsTestOptimizer()
		defer optimizer.Cleanup()
		
		tempDir := optimizer.OptimizedTempDir(b, "benchmark")
		content := []byte("test content for benchmarking file write operations")
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			filePath := filepath.Join(tempDir, fmt.Sprintf("test-%d.txt", i))
			if err := optimizer.OptimizedFileWrite(filePath, content); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("BatchFileCreation", func(b *testing.B) {
		optimizer := NewWindowsTestOptimizer()
		defer optimizer.Cleanup()
		
		tempDir := optimizer.OptimizedTempDir(b, "batch")
		
		// Prepare files for batch creation
		files := make(map[string][]byte)
		for i := 0; i < 10; i++ {
			files[fmt.Sprintf("batch-%d.txt", i)] = []byte(fmt.Sprintf("batch content %d", i))
		}
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			testDir := filepath.Join(tempDir, fmt.Sprintf("run-%d", i))
			if err := optimizer.BulkFileCreation(testDir, files); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkWindowsTestRunner benchmarks the optimized test runner
func BenchmarkWindowsTestRunner(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("Windows-specific benchmark")
	}

	b.Run("StandardTestExecution", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			// Simulate standard test setup
			tempDir := b.TempDir()
			
			// Create some test files
			for j := 0; j < 5; j++ {
				filePath := filepath.Join(tempDir, fmt.Sprintf("test-%d.txt", j))
				if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
					b.Fatal(err)
				}
			}
			
			b.StartTimer()
			// Simulate test execution
			time.Sleep(time.Millisecond) // Simulate test work
		}
	})

	b.Run("OptimizedTestExecution", func(b *testing.B) {
		runner := NewOptimizedTestRunner()
		defer runner.optimizer.Cleanup()
		
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			b.StopTimer()
			// Create test context
			ctx := &TestContext{
				T:           b,
				TempDir:     runner.optimizer.OptimizedTempDir(b, "benchmark"),
				Optimizer:   runner.optimizer,
				BatchProc:   runner.batchProcessor,
				ResourceMgr: runner.resourcePool,
			}
			
			// Create test files using batch processor
			files := make(map[string][]byte)
			for j := 0; j < 5; j++ {
				files[fmt.Sprintf("test-%d.txt", j)] = []byte("test content")
			}
			
			b.StartTimer()
			_ = ctx.CreateTempFiles(files)
			// Simulate test execution
			time.Sleep(time.Millisecond) // Simulate test work
		}
	})
}

// BenchmarkWindowsConcurrency benchmarks concurrency management
func BenchmarkWindowsConcurrency(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("Windows-specific benchmark")
	}

	b.Run("UnlimitedConcurrency", func(b *testing.B) {
		tempDir := b.TempDir()
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				filePath := filepath.Join(tempDir, fmt.Sprintf("concurrent-%d.txt", time.Now().UnixNano()))
				if err := os.WriteFile(filePath, []byte("concurrent content"), 0644); err != nil {
					b.Error(err)
				}
			}
		})
	})

	b.Run("ManagedConcurrency", func(b *testing.B) {
		concurrencyMgr := NewWindowsConcurrencyManager()
		optimizer := NewWindowsTestOptimizer()
		defer optimizer.Cleanup()
		
		tempDir := optimizer.OptimizedTempDir(b, "concurrent")
		
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				testName := fmt.Sprintf("test-%d", time.Now().UnixNano())
				release := concurrencyMgr.AcquireSlot(testName)
				
				filePath := filepath.Join(tempDir, fmt.Sprintf("concurrent-%s.txt", testName))
				if err := optimizer.OptimizedFileWrite(filePath, []byte("concurrent content")); err != nil {
					b.Error(err)
				}
				
				release()
			}
		})
	})
}

// BenchmarkWindowsPathOperations benchmarks path operations
func BenchmarkWindowsPathOperations(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("Windows-specific benchmark")
	}

	pathNormalizer := NewWindowsPathNormalizer()
	longPath := filepath.Join("C:", "a very long path name that might cause issues", "with multiple levels", "of deeply nested directories", "that could exceed Windows MAX_PATH limits", "test.txt")

	b.Run("StandardPathOperations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = filepath.Clean(longPath)
			_ = filepath.Dir(longPath)
			_ = filepath.Base(longPath)
		}
	})

	b.Run("OptimizedPathOperations", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = pathNormalizer.NormalizePath(longPath)
			_ = pathNormalizer.GetShortPath(longPath)
		}
	})
}

// TestWindowsOptimizationEffectiveness tests the effectiveness of optimizations
func TestWindowsOptimizationEffectiveness(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	t.Run("FileOperationSpeed", func(t *testing.T) {
		// Test standard approach
		standardStart := time.Now()
		tempDir1 := t.TempDir()
		for i := 0; i < 100; i++ {
			filePath := filepath.Join(tempDir1, fmt.Sprintf("standard-%d.txt", i))
			if err := os.WriteFile(filePath, []byte("test content"), 0644); err != nil {
				t.Fatal(err)
			}
		}
		standardDuration := time.Since(standardStart)

		// Test optimized approach
		optimizer := NewWindowsTestOptimizer()
		defer optimizer.Cleanup()
		
		optimizedStart := time.Now()
		tempDir2 := optimizer.OptimizedTempDir(t, "effectiveness")
		
		files := make(map[string][]byte)
		for i := 0; i < 100; i++ {
			files[fmt.Sprintf("optimized-%d.txt", i)] = []byte("test content")
		}
		
		if err := optimizer.BulkFileCreation(tempDir2, files); err != nil {
			t.Fatal(err)
		}
		optimizedDuration := time.Since(optimizedStart)

		t.Logf("Standard approach: %v", standardDuration)
		t.Logf("Optimized approach: %v", optimizedDuration)
		
		if optimizedDuration > standardDuration*2 {
			t.Errorf("Optimized approach is significantly slower: %v vs %v", optimizedDuration, standardDuration)
		}
	})

	t.Run("MemoryUsage", func(t *testing.T) {
		var m1, m2 runtime.MemStats
		
		// Measure standard approach
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		tempDir1 := t.TempDir()
		for i := 0; i < 50; i++ {
			filePath := filepath.Join(tempDir1, fmt.Sprintf("mem-test-%d.txt", i))
			content := make([]byte, 1024) // 1KB per file
			if err := os.WriteFile(filePath, content, 0644); err != nil {
				t.Fatal(err)
			}
		}
		
		runtime.GC()
		runtime.ReadMemStats(&m2)
		standardMemory := m2.Alloc - m1.Alloc

		// Measure optimized approach
		optimizer := NewWindowsTestOptimizer()
		defer optimizer.Cleanup()
		
		runtime.GC()
		runtime.ReadMemStats(&m1)
		
		tempDir2 := optimizer.OptimizedTempDir(t, "memory")
		files := make(map[string][]byte)
		for i := 0; i < 50; i++ {
			files[fmt.Sprintf("mem-test-%d.txt", i)] = make([]byte, 1024)
		}
		
		if err := optimizer.BulkFileCreation(tempDir2, files); err != nil {
			t.Fatal(err)
		}
		
		runtime.GC()
		runtime.ReadMemStats(&m2)
		optimizedMemory := m2.Alloc - m1.Alloc

		t.Logf("Standard memory usage: %d bytes", standardMemory)
		t.Logf("Optimized memory usage: %d bytes", optimizedMemory)
	})

	t.Run("ConcurrencyEfficiency", func(t *testing.T) {
		concurrencyMgr := NewWindowsConcurrencyManager()
		
		// Test that concurrency manager limits active tests appropriately
		maxConcurrency := runtime.NumCPU() / 2 // Expected limit on Windows
		
		var activeTests []func()
		for i := 0; i < maxConcurrency*2; i++ {
			testName := fmt.Sprintf("concurrency-test-%d", i)
			release := concurrencyMgr.AcquireSlot(testName)
			activeTests = append(activeTests, release)
			
			if i < maxConcurrency {
				// Should be able to acquire up to max concurrency
				if concurrencyMgr.GetActiveTests() != i+1 {
					t.Errorf("Expected %d active tests, got %d", i+1, concurrencyMgr.GetActiveTests())
				}
			}
		}
		
		// Clean up
		for _, release := range activeTests {
			release()
		}
		
		if concurrencyMgr.GetActiveTests() != 0 {
			t.Errorf("Expected 0 active tests after cleanup, got %d", concurrencyMgr.GetActiveTests())
		}
	})
}

// TestWindowsTimeout tests optimized timeout handling
func TestWindowsTimeout(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	optimizer := NewWindowsTestOptimizer()
	defer optimizer.Cleanup()

	baseTimeout := 5 * time.Second
	optimizedTimeout := optimizer.OptimizedTestTimeout(baseTimeout)

	expectedTimeout := time.Duration(float64(baseTimeout) * 1.5)
	if optimizedTimeout != expectedTimeout {
		t.Errorf("Expected optimized timeout %v, got %v", expectedTimeout, optimizedTimeout)
	}

	// Test context timeout
	ctx, cancel := optimizer.OptimizedContextTimeout(context.Background(), baseTimeout)
	defer cancel()

	deadline, ok := ctx.Deadline()
	if !ok {
		t.Fatal("Context should have a deadline")
	}

	actualTimeout := time.Until(deadline)
	// Allow some variance for execution time
	if actualTimeout < expectedTimeout-time.Second || actualTimeout > expectedTimeout+time.Second {
		t.Errorf("Context timeout should be around %v, got %v", expectedTimeout, actualTimeout)
	}
}

// Example of how to use the optimized test runner
func ExampleOptimizedTestRunner() {
	// This would be used in an actual test
	runner := NewOptimizedTestRunner()
	defer runner.optimizer.Cleanup()

	// Create a mock testing.T for the example
	mockT := &testing.T{}

	runner.RunOptimizedTest(mockT, func(t *testing.T, ctx *TestContext) {
		// Create test files efficiently
		files := map[string][]byte{
			"config.json":    []byte(`{"test": true}`),
			"data.txt":       []byte("test data"),
			"subdir/file.md": []byte("# Test"),
		}

		filePaths := ctx.CreateTempFiles(files)
		
		// Use optimized context with timeout
		testCtx, cancel := ctx.GetOptimizedContext(10 * time.Second)
		defer cancel()

		// Perform test operations with the optimized context
		_ = testCtx
		_ = filePaths

		fmt.Println("Test completed with Windows optimizations")
	})

	// Output: Test completed with Windows optimizations
}