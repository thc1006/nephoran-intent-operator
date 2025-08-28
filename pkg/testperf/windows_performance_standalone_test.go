package testperf

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

// WindowsTestOptimizer provides Windows-specific test optimizations (standalone version)
type WindowsTestOptimizer struct {
	tempDirCache   sync.Map
	filePoolCache  sync.Map
	cleanupQueue   []func()
	cleanupMutex   sync.Mutex
	concurrencyLim chan struct{}
}

// NewWindowsTestOptimizer creates a new Windows test optimizer
func NewWindowsTestOptimizer() *WindowsTestOptimizer {
	maxConcurrency := runtime.NumCPU()
	if runtime.GOOS == "windows" {
		// Reduce concurrency on Windows to avoid file system contention
		maxConcurrency = max(1, maxConcurrency/2)
	}

	return &WindowsTestOptimizer{
		concurrencyLim: make(chan struct{}, maxConcurrency),
	}
}

// OptimizedTempDir creates temporary directories with Windows-specific optimizations
func (w *WindowsTestOptimizer) OptimizedTempDir(t testing.TB, prefix string) string {
	// Create optimized temp directory
	var tempDir string
	if runtime.GOOS == "windows" {
		// Use shorter paths on Windows to avoid MAX_PATH issues
		tempBase := os.TempDir()
		shortPrefix := prefix
		if len(prefix) > 8 {
			shortPrefix = prefix[:8]
		}
		tempDir = filepath.Join(tempBase, fmt.Sprintf("%s-%d", shortPrefix, time.Now().UnixNano()%1000000))
	} else {
		tempDir = t.TempDir()
	}

	// Create directory with appropriate permissions
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Schedule cleanup
	w.scheduleCleanup(func() {
		os.RemoveAll(tempDir)
	})

	return tempDir
}

// OptimizedFileWrite performs Windows-optimized file writes
func (w *WindowsTestOptimizer) OptimizedFileWrite(filePath string, content []byte) error {
	// Acquire concurrency limit to prevent file system overload
	w.concurrencyLim <- struct{}{}
	defer func() { <-w.concurrencyLim }()

	// Ensure directory exists
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Write with optimizations for Windows
	if runtime.GOOS == "windows" {
		// Use temporary file and atomic rename for reliability
		tempFile := filePath + ".tmp"
		if err := os.WriteFile(tempFile, content, 0644); err != nil {
			return fmt.Errorf("failed to write temp file: %w", err)
		}
		if err := os.Rename(tempFile, filePath); err != nil {
			os.Remove(tempFile) // Cleanup on failure
			return fmt.Errorf("failed to rename temp file: %w", err)
		}
	} else {
		// Direct write on Unix systems
		if err := os.WriteFile(filePath, content, 0644); err != nil {
			return fmt.Errorf("failed to write file: %w", err)
		}
	}

	return nil
}

// BulkFileCreation creates multiple files efficiently
func (w *WindowsTestOptimizer) BulkFileCreation(baseDir string, files map[string][]byte) error {
	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return fmt.Errorf("failed to create base directory: %w", err)
	}

	// Create files with optimized concurrency
	errChan := make(chan error, len(files))
	var wg sync.WaitGroup

	for filename, content := range files {
		wg.Add(1)
		go func(fname string, data []byte) {
			defer wg.Done()
			filePath := filepath.Join(baseDir, fname)
			if err := w.OptimizedFileWrite(filePath, data); err != nil {
				errChan <- fmt.Errorf("failed to create file %s: %w", fname, err)
			}
		}(filename, content)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		if err != nil {
			return err
		}
	}

	return nil
}

// OptimizedTestTimeout returns platform-optimized test timeouts
func (w *WindowsTestOptimizer) OptimizedTestTimeout(baseTimeout time.Duration) time.Duration {
	if runtime.GOOS == "windows" {
		// Increase timeouts on Windows by 50% due to slower file I/O
		return time.Duration(float64(baseTimeout) * 1.5)
	}
	return baseTimeout
}

// scheduleCleanup adds a cleanup function to be executed later
func (w *WindowsTestOptimizer) scheduleCleanup(fn func()) {
	w.cleanupMutex.Lock()
	defer w.cleanupMutex.Unlock()
	w.cleanupQueue = append(w.cleanupQueue, fn)
}

// Cleanup performs all scheduled cleanup operations
func (w *WindowsTestOptimizer) Cleanup() {
	w.cleanupMutex.Lock()
	defer w.cleanupMutex.Unlock()

	// Execute cleanup functions in reverse order (LIFO)
	for i := len(w.cleanupQueue) - 1; i >= 0; i-- {
		if fn := w.cleanupQueue[i]; fn != nil {
			fn()
		}
	}
	w.cleanupQueue = nil
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// BenchmarkWindowsFileOperations benchmarks file operations with and without optimizations
func BenchmarkWindowsFileOperations(b *testing.B) {
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

// TestWindowsOptimizationEffectiveness tests the effectiveness of optimizations
func TestWindowsOptimizationEffectiveness(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping effectiveness test in short mode")
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
		
		speedup := float64(standardDuration) / float64(optimizedDuration)
		t.Logf("Speedup factor: %.2fx", speedup)
		
		if optimizedDuration > standardDuration*2 {
			t.Errorf("Optimized approach is significantly slower: %v vs %v", optimizedDuration, standardDuration)
		}
	})

	t.Run("TimeoutOptimization", func(t *testing.T) {
		optimizer := NewWindowsTestOptimizer()
		defer optimizer.Cleanup()

		baseTimeout := 5 * time.Second
		optimizedTimeout := optimizer.OptimizedTestTimeout(baseTimeout)

		if runtime.GOOS == "windows" {
			expectedTimeout := time.Duration(float64(baseTimeout) * 1.5)
			if optimizedTimeout != expectedTimeout {
				t.Errorf("Expected optimized timeout %v, got %v", expectedTimeout, optimizedTimeout)
			}
		} else {
			if optimizedTimeout != baseTimeout {
				t.Errorf("Expected timeout unchanged on non-Windows, got %v instead of %v", optimizedTimeout, baseTimeout)
			}
		}
	})
}

// TestConcurrencyLimiting tests that concurrency is properly limited on Windows
func TestConcurrencyLimiting(t *testing.T) {
	optimizer := NewWindowsTestOptimizer()
	defer optimizer.Cleanup()
	
	// Test that the optimizer limits concurrency properly
	expectedLimit := runtime.NumCPU()
	if runtime.GOOS == "windows" {
		expectedLimit = max(1, expectedLimit/2)
	}
	
	actualLimit := cap(optimizer.concurrencyLim)
	if actualLimit != expectedLimit {
		t.Errorf("Expected concurrency limit %d, got %d", expectedLimit, actualLimit)
	}
	
	t.Logf("Platform: %s, CPUs: %d, Concurrency limit: %d", 
		runtime.GOOS, runtime.NumCPU(), actualLimit)
}

// Example function showing how to use optimizations
func ExampleWindowsTestOptimizer() {
	// Create optimizer
	optimizer := NewWindowsTestOptimizer()
	defer optimizer.Cleanup()
	
	// Create optimized temp directory
	tempDir := optimizer.OptimizedTempDir(&testing.T{}, "example")
	
	// Prepare files for batch creation
	files := map[string][]byte{
		"config.json": []byte(`{"test": true}`),
		"data.txt":    []byte("example data"),
		"readme.md":   []byte("# Example"),
	}
	
	// Create files efficiently
	if err := optimizer.BulkFileCreation(tempDir, files); err != nil {
		fmt.Printf("Error creating files: %v\n", err)
		return
	}
	
	// Get optimized timeout
	baseTimeout := 5 * time.Second
	optimizedTimeout := optimizer.OptimizedTestTimeout(baseTimeout)
	
	fmt.Printf("Base timeout: %v\n", baseTimeout)
	fmt.Printf("Optimized timeout: %v\n", optimizedTimeout)
	fmt.Printf("Files created in: %s\n", tempDir)
	
	// Output varies by platform:
	// On Windows: Optimized timeout will be 1.5x longer
	// On Unix: Optimized timeout will be the same
}