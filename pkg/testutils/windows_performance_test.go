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

// BenchmarkContext provides optimized test context for benchmarks
type BenchmarkContext struct {
	B           *testing.B
	TempDir     string
	Optimizer   *WindowsTestOptimizer
	BatchProc   *BatchFileProcessor
	ResourceMgr *TestResourcePool
	cleanupFns  []func()
}

// CreateTempFile creates a temporary file with optimizations for benchmarks
func (bc *BenchmarkContext) CreateTempFile(name string, content []byte) string {
	filePath := filepath.Join(bc.TempDir, name)
	if err := bc.Optimizer.OptimizedFileWrite(filePath, content); err != nil {
		bc.B.Fatalf("Failed to create temp file %s: %v", name, err)
	}
	return filePath
}

// CreateTempFiles creates multiple files efficiently for benchmarks
func (bc *BenchmarkContext) CreateTempFiles(files map[string][]byte) map[string]string {
	results := make(map[string]string)
	if err := bc.BatchProc.CreateFiles(bc.TempDir, files); err != nil {
		bc.B.Fatalf("Failed to create temp files: %v", err)
	}

	for name := range files {
		results[name] = filepath.Join(bc.TempDir, name)
	}
	return results
}

// GetOptimizedContext creates a context with optimized timeout for benchmarks
func (bc *BenchmarkContext) GetOptimizedContext(baseTimeout time.Duration) (context.Context, context.CancelFunc) {
	return bc.Optimizer.OptimizedContextTimeout(context.Background(), baseTimeout)
}

// AddCleanup adds a cleanup function for benchmarks
func (bc *BenchmarkContext) AddCleanup(fn func()) {
	bc.cleanupFns = append(bc.cleanupFns, fn)
}

// Cleanup performs all cleanup operations for benchmarks
func (bc *BenchmarkContext) Cleanup() {
	// Execute cleanup functions in reverse order
	for i := len(bc.cleanupFns) - 1; i >= 0; i-- {
		if fn := bc.cleanupFns[i]; fn != nil {
			fn()
		}
	}
	bc.cleanupFns = nil

	// Cleanup optimizer
	bc.Optimizer.Cleanup()
}

// BenchmarkWindowsFileOperations benchmarks file operations with and without optimizations
func BenchmarkWindowsFileOperations(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("Windows-specific benchmarks")
		return
	}

	// Skip if running in CI or short tests
	if testing.Short() || os.Getenv("CI") != "" {
		b.Skip("Skipping performance benchmarks in CI/short mode")
		return
	}

	tests := []struct {
		name             string
		fileCount        int
		fileSize         int
		useOptimizations bool
	}{
		{"Small_Files_Unoptimized", 10, 1024, false},
		{"Small_Files_Optimized", 10, 1024, true},
		{"Medium_Files_Unoptimized", 5, 10240, false},
		{"Medium_Files_Optimized", 5, 10240, true},
		{"Large_Files_Unoptimized", 2, 102400, false},
		{"Large_Files_Optimized", 2, 102400, true},
	}

	for _, tt := range tests {
		b.Run(tt.name, func(b *testing.B) {
			benchmarkFileOperations(b, tt.fileCount, tt.fileSize, tt.useOptimizations)
		})
	}
}

func benchmarkFileOperations(b *testing.B, fileCount, fileSize int, useOptimizations bool) {
	// Create optimizer
	optimizer := NewWindowsTestOptimizer()
	batchProcessor := &BatchFileProcessor{
		concurrencyLimit: make(chan struct{}, 4),
	}
	resourcePool := NewTestResourcePool()

	runner := &OptimizedTestRunner{
		optimizer:      optimizer,
		batchProcessor: batchProcessor,
		resourcePool:   resourcePool,
	}

	// Reset timer before the benchmark loop
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		// Create benchmark context - b implements testing.TB interface
		ctx := &BenchmarkContext{
			B:           b,
			TempDir:     runner.optimizer.OptimizedTempDir(b, "benchmark"),
			Optimizer:   runner.optimizer,
			BatchProc:   runner.batchProcessor,
			ResourceMgr: runner.resourcePool,
		}

		// Create test files using batch processor
		files := make(map[string][]byte)
		for j := 0; j < fileCount; j++ {
			content := make([]byte, fileSize)
			for k := range content {
				content[k] = byte(k % 256)
			}
			files[fmt.Sprintf("testfile_%d.txt", j)] = content
		}

		b.StartTimer()

		if useOptimizations {
			// Use optimized file operations
			_ = ctx.CreateTempFiles(files)
		} else {
			// Use standard file operations
			for name, content := range files {
				tempFile := filepath.Join(ctx.TempDir, name)
				if err := os.WriteFile(tempFile, content, 0644); err != nil {
					b.Fatalf("Failed to write file: %v", err)
				}
			}
		}

		b.StopTimer()
		ctx.Cleanup()
	}
}

// BenchmarkWindowsContextOperations benchmarks context operations
func BenchmarkWindowsContextOperations(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("Windows-specific benchmarks")
		return
	}

	if testing.Short() {
		b.Skip("Skipping performance benchmarks in short mode")
		return
	}

	optimizer := NewWindowsTestOptimizer()
	defer optimizer.Cleanup()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx, cancel := optimizer.OptimizedContextTimeout(context.Background(), time.Second*5)

		// Simulate some work
		select {
		case <-time.After(time.Millisecond * 10):
		case <-ctx.Done():
			b.Fatalf("Context cancelled unexpectedly")
		}

		cancel()
	}
}

// BenchmarkWindowsResourcePoolOperations benchmarks resource pool operations
func BenchmarkWindowsResourcePoolOperations(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("Windows-specific benchmarks")
		return
	}

	if testing.Short() {
		b.Skip("Skipping performance benchmarks in short mode")
		return
	}

	pool := NewTestResourcePool()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test cached directory operations
		key := fmt.Sprintf("test-dir-%d", i%10)
		if cachedDir, exists := pool.GetCachedDirectory(key); exists {
			// Use cached directory
			_ = cachedDir
		} else {
			// Create and cache new directory
			tempDir, err := os.MkdirTemp("", "bench-test-*")
			if err != nil {
				b.Fatalf("Failed to create temp dir: %v", err)
			}
			pool.CacheDirectory(key, tempDir)
		}

		// Test cached file operations
		testContent := []byte("benchmark test content")
		filePath := fmt.Sprintf("test-file-%d.txt", i%10)
		if cachedContent, exists := pool.GetCachedFile(filePath); exists {
			// Use cached content
			_ = cachedContent
		} else {
			// Cache new content
			pool.CacheFile(filePath, testContent)
		}
	}
}
