
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



// WindowsTestOptimizer provides Windows-specific test optimizations.

type WindowsTestOptimizer struct {

	tempDirCache   sync.Map

	filePoolCache  sync.Map

	cleanupQueue   []func()

	cleanupMutex   sync.Mutex

	concurrencyLim chan struct{}

}



// NewWindowsTestOptimizer creates a new Windows test optimizer.

func NewWindowsTestOptimizer() *WindowsTestOptimizer {

	maxConcurrency := runtime.NumCPU()

	if runtime.GOOS == "windows" {

		// Reduce concurrency on Windows to avoid file system contention.

		maxConcurrency = max(1, maxConcurrency/2)

	}



	return &WindowsTestOptimizer{

		concurrencyLim: make(chan struct{}, maxConcurrency),

	}

}



// OptimizedTempDir creates temporary directories with Windows-specific optimizations.

func (w *WindowsTestOptimizer) OptimizedTempDir(t testing.TB, prefix string) string {

	// Use cached temp dir if available for the same test.

	testName := t.Name()

	if cached, ok := w.tempDirCache.Load(testName); ok {

		return cached.(string)

	}



	// Create optimized temp directory.

	var tempDir string

	if runtime.GOOS == "windows" {

		// Use shorter paths on Windows to avoid MAX_PATH issues.

		tempBase := os.TempDir()

		shortPrefix := prefix

		if len(prefix) > 8 {

			shortPrefix = prefix[:8]

		}

		tempDir = filepath.Join(tempBase, fmt.Sprintf("%s-%d", shortPrefix, time.Now().UnixNano()%1000000))

	} else {

		tempDir = t.TempDir()

	}



	// Create directory with appropriate permissions.

	if err := os.MkdirAll(tempDir, 0o755); err != nil {

		t.Fatalf("Failed to create temp dir: %v", err)

	}



	// Cache for reuse within the same test.

	w.tempDirCache.Store(testName, tempDir)



	// Schedule cleanup.

	w.scheduleCleanup(func() {

		os.RemoveAll(tempDir)

		w.tempDirCache.Delete(testName)

	})



	return tempDir

}



// OptimizedFileWrite performs Windows-optimized file writes.

func (w *WindowsTestOptimizer) OptimizedFileWrite(filePath string, content []byte) error {

	// Acquire concurrency limit to prevent file system overload.

	w.concurrencyLim <- struct{}{}

	defer func() { <-w.concurrencyLim }()



	// Ensure directory exists.

	dir := filepath.Dir(filePath)

	if err := os.MkdirAll(dir, 0o755); err != nil {

		return fmt.Errorf("failed to create directory %s: %w", dir, err)

	}



	// Write with optimizations for Windows.

	if runtime.GOOS == "windows" {

		// Use temporary file and atomic rename for reliability.

		tempFile := filePath + ".tmp"

		if err := os.WriteFile(tempFile, content, 0o640); err != nil {

			return fmt.Errorf("failed to write temp file: %w", err)

		}

		if err := os.Rename(tempFile, filePath); err != nil {

			os.Remove(tempFile) // Cleanup on failure

			return fmt.Errorf("failed to rename temp file: %w", err)

		}

	} else {

		// Direct write on Unix systems.

		if err := os.WriteFile(filePath, content, 0o640); err != nil {

			return fmt.Errorf("failed to write file: %w", err)

		}

	}



	return nil

}



// OptimizedFileRead performs Windows-optimized file reads with caching.

func (w *WindowsTestOptimizer) OptimizedFileRead(filePath string) ([]byte, error) {

	// Check file pool cache first.

	if cached, ok := w.filePoolCache.Load(filePath); ok {

		return cached.([]byte), nil

	}



	// Acquire concurrency limit.

	w.concurrencyLim <- struct{}{}

	defer func() { <-w.concurrencyLim }()



	content, err := os.ReadFile(filePath)

	if err != nil {

		return nil, fmt.Errorf("failed to read file %s: %w", filePath, err)

	}



	// Cache for subsequent reads.

	w.filePoolCache.Store(filePath, content)



	return content, nil

}



// BulkFileCreation creates multiple files efficiently.

func (w *WindowsTestOptimizer) BulkFileCreation(baseDir string, files map[string][]byte) error {

	// Ensure base directory exists.

	if err := os.MkdirAll(baseDir, 0o755); err != nil {

		return fmt.Errorf("failed to create base directory: %w", err)

	}



	// Create files with optimized concurrency.

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



	// Check for errors.

	for err := range errChan {

		if err != nil {

			return err

		}

	}



	return nil

}



// OptimizedTestTimeout returns platform-optimized test timeouts.

func (w *WindowsTestOptimizer) OptimizedTestTimeout(baseTimeout time.Duration) time.Duration {

	if runtime.GOOS == "windows" {

		// Increase timeouts on Windows by 50% due to slower file I/O.

		return time.Duration(float64(baseTimeout) * 1.5)

	}

	return baseTimeout

}



// OptimizedContextTimeout creates contexts with platform-optimized timeouts.

func (w *WindowsTestOptimizer) OptimizedContextTimeout(ctx context.Context, baseTimeout time.Duration) (context.Context, context.CancelFunc) {

	optimizedTimeout := w.OptimizedTestTimeout(baseTimeout)

	return context.WithTimeout(ctx, optimizedTimeout)

}



// scheduleCleanup adds a cleanup function to be executed later.

func (w *WindowsTestOptimizer) scheduleCleanup(fn func()) {

	w.cleanupMutex.Lock()

	defer w.cleanupMutex.Unlock()

	w.cleanupQueue = append(w.cleanupQueue, fn)

}



// Cleanup performs all scheduled cleanup operations.

func (w *WindowsTestOptimizer) Cleanup() {

	w.cleanupMutex.Lock()

	defer w.cleanupMutex.Unlock()



	// Execute cleanup functions in reverse order (LIFO).

	for i := len(w.cleanupQueue) - 1; i >= 0; i-- {

		if fn := w.cleanupQueue[i]; fn != nil {

			fn()

		}

	}

	w.cleanupQueue = nil



	// Clear caches.

	w.tempDirCache = sync.Map{}

	w.filePoolCache = sync.Map{}

}



// WindowsFileSystemOptimizer provides Windows-specific file system optimizations.

type WindowsFileSystemOptimizer struct {

	pathNormalizer *WindowsPathNormalizer

	ioLimiter      chan struct{}

}



// NewWindowsFileSystemOptimizer creates a new file system optimizer.

func NewWindowsFileSystemOptimizer() *WindowsFileSystemOptimizer {

	// Limit concurrent I/O operations on Windows.

	ioLimit := runtime.NumCPU()

	if runtime.GOOS == "windows" {

		ioLimit = max(1, ioLimit/2)

	}



	return &WindowsFileSystemOptimizer{

		pathNormalizer: NewWindowsPathNormalizer(),

		ioLimiter:      make(chan struct{}, ioLimit),

	}

}



// WindowsPathNormalizer handles Windows path normalization.

type WindowsPathNormalizer struct {

	driveLetterCache sync.Map

}



// NewWindowsPathNormalizer creates a new path normalizer.

func NewWindowsPathNormalizer() *WindowsPathNormalizer {

	return &WindowsPathNormalizer{}

}



// NormalizePath normalizes paths for Windows compatibility.

func (w *WindowsPathNormalizer) NormalizePath(path string) string {

	if runtime.GOOS != "windows" {

		return filepath.Clean(path)

	}



	// Handle Windows-specific path issues.

	normalized := filepath.Clean(path)



	// Convert forward slashes to backslashes.

	normalized = filepath.FromSlash(normalized)



	// Handle long path prefix if needed.

	if len(normalized) > 260 && !filepath.IsAbs(normalized) {

		if abs, err := filepath.Abs(normalized); err == nil {

			normalized = abs

		}

	}



	return normalized

}



// GetShortPath returns a Windows 8.3 short path if possible.

func (w *WindowsPathNormalizer) GetShortPath(path string) string {

	if runtime.GOOS != "windows" {

		return path

	}



	// Try to get short path to avoid MAX_PATH issues.

	// This is a simplified version - in production, you might use Windows APIs.

	normalized := w.NormalizePath(path)



	// If path is too long, try to use temp directory with shorter name.

	if len(normalized) > 200 {

		tempDir := os.TempDir()

		baseName := filepath.Base(normalized)

		if len(baseName) > 50 {

			baseName = baseName[:50]

		}

		return filepath.Join(tempDir, baseName)

	}



	return normalized

}



// WindowsConcurrencyManager manages test concurrency for Windows.

type WindowsConcurrencyManager struct {

	semaphore chan struct{}

	active    sync.Map

}



// NewWindowsConcurrencyManager creates a new concurrency manager.

func NewWindowsConcurrencyManager() *WindowsConcurrencyManager {

	// Reduce concurrency on Windows to prevent resource exhaustion.

	maxConcurrency := runtime.NumCPU()

	if runtime.GOOS == "windows" {

		maxConcurrency = max(1, maxConcurrency/2)

	}



	return &WindowsConcurrencyManager{

		semaphore: make(chan struct{}, maxConcurrency),

	}

}



// AcquireSlot acquires a concurrency slot for the given test.

func (w *WindowsConcurrencyManager) AcquireSlot(testName string) func() {

	w.semaphore <- struct{}{}

	w.active.Store(testName, time.Now())



	return func() {

		w.active.Delete(testName)

		<-w.semaphore

	}

}



// GetActiveTests returns the number of currently active tests.

func (w *WindowsConcurrencyManager) GetActiveTests() int {

	count := 0

	w.active.Range(func(key, value interface{}) bool {

		count++

		return true

	})

	return count

}



// Global instances for easy access.

var (

	globalTestOptimizer       = NewWindowsTestOptimizer()

	globalFileSystemOptimizer = NewWindowsFileSystemOptimizer()

	globalConcurrencyManager  = NewWindowsConcurrencyManager()

)



// Convenience functions for global access.



// GetOptimizedTempDir creates an optimized temporary directory.

func GetOptimizedTempDir(t testing.TB, prefix string) string {

	return globalTestOptimizer.OptimizedTempDir(t, prefix)

}



// WriteFileOptimized writes a file with Windows optimizations.

func WriteFileOptimized(filePath string, content []byte) error {

	return globalTestOptimizer.OptimizedFileWrite(filePath, content)

}



// ReadFileOptimized reads a file with Windows optimizations and caching.

func ReadFileOptimized(filePath string) ([]byte, error) {

	return globalTestOptimizer.OptimizedFileRead(filePath)

}



// GetOptimizedTimeout returns a platform-optimized timeout.

func GetOptimizedTimeout(baseTimeout time.Duration) time.Duration {

	return globalTestOptimizer.OptimizedTestTimeout(baseTimeout)

}



// GetOptimizedContext creates a context with platform-optimized timeout.

func GetOptimizedContext(ctx context.Context, baseTimeout time.Duration) (context.Context, context.CancelFunc) {

	return globalTestOptimizer.OptimizedContextTimeout(ctx, baseTimeout)

}



// NormalizePath normalizes a path for the current platform.

func NormalizePath(path string) string {

	return globalFileSystemOptimizer.pathNormalizer.NormalizePath(path)

}



// AcquireConcurrencySlot acquires a concurrency slot for better resource management.

func AcquireConcurrencySlot(testName string) func() {

	return globalConcurrencyManager.AcquireSlot(testName)

}



// CleanupGlobalOptimizers cleans up all global optimizers.

func CleanupGlobalOptimizers() {

	globalTestOptimizer.Cleanup()

}



// max returns the maximum of two integers.

func max(a, b int) int {

	if a > b {

		return a

	}

	return b

}

