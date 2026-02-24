package loop

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

// ProcessorConfig holds configuration for the intent processor.

type ProcessorConfig struct {
	HandoffDir string

	ErrorDir string

	PorchMode string // "direct" or "structured"

	BatchSize int

	BatchInterval time.Duration

	MaxRetries int

	SendTimeout time.Duration // Timeout for sending to batch coordinator

	WorkerCount int // Number of concurrent workers
}

// DefaultConfig returns default processor configuration.

func DefaultConfig() *ProcessorConfig {
	sendTimeout := 5 * time.Second

	if runtime.GOOS == "windows" {
		sendTimeout = 10 * time.Second // Longer timeout on Windows
	}

	return &ProcessorConfig{
		HandoffDir: "./handoff",

		ErrorDir: "./handoff/errors",

		PorchMode: "structured",

		BatchSize: 10,

		BatchInterval: 5 * time.Second,

		MaxRetries: 3,

		SendTimeout: sendTimeout,

		WorkerCount: runtime.NumCPU(),
	}
}

// Validator interface for intent validation.

type Validator interface {
	ValidateBytes([]byte) (*ingest.Intent, error)
}

// IntentProcessor handles validation and submission of intents.

type IntentProcessor struct {
	config *ProcessorConfig

	validator Validator

	porchFunc PorchSubmitFunc

	processed *SafeSet // for idempotency

	ctx context.Context

	cancel context.CancelFunc

	wg sync.WaitGroup

	// Batch coordinator channels.

	inCh chan string // Input channel for file paths

	stopCh chan struct{} // Stop signal for coordinator

	coordReady chan struct{} // Signals coordinator is ready

	// Task tracking for graceful shutdown.

	taskWg sync.WaitGroup // Tracks in-flight tasks

	tasksQueued atomic.Int64 // Number of tasks queued

	// Stats tracking (atomic for thread safety).

	processedCount int64

	failedCount int64

	realFailedCount int64

	shutdownFailedCount int64

	// Graceful shutdown tracking.

	gracefulShutdown atomic.Bool

	shutdownStartTime time.Time

	shutdownMutex sync.RWMutex
}

// NewProcessor creates a new intent processor.

func NewProcessor(config *ProcessorConfig, validator Validator, porchFunc PorchSubmitFunc) (*IntentProcessor, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure error directory exists.

	if err := os.MkdirAll(config.ErrorDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create error directory: %w", err)
	}

	// Load processed intents for idempotency.

	processed := NewSafeSet()

	processedFile := filepath.Join(config.HandoffDir, ".processed")

	if data, err := os.ReadFile(processedFile); err == nil {
		lines := strings.Split(string(data), "\n")

		for _, line := range lines {
			if line != "" {
				processed.Add(line)
			}
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	// Size channel buffer to prevent coordinator send timeouts during normal operation.

	// Formula: BatchSize * 4 + WorkerCount * 4 ensures adequate buffering.

	// This prevents the "timeout sending file to batch coordinator" errors.

	channelBuffer := config.BatchSize * 4

	if config.WorkerCount > 0 {
		channelBuffer = max(channelBuffer, config.WorkerCount*4)
	}

	// Ensure minimum buffer size to handle burst load.

	if channelBuffer < 50 {
		channelBuffer = 50
	}

	return &IntentProcessor{
		config: config,

		validator: validator,

		porchFunc: porchFunc,

		processed: processed,

		ctx: ctx,

		cancel: cancel,

		inCh: make(chan string, channelBuffer), // Larger buffer to prevent blocking

		stopCh: make(chan struct{}),

		coordReady: make(chan struct{}),
	}, nil
}

// ProcessFile processes a single intent file.

func (p *IntentProcessor) ProcessFile(filename string) error {
	// During shutdown, don't attempt to queue new work.

	if p.gracefulShutdown.Load() {
		return fmt.Errorf("processor is shutting down")
	}

	// Check if already processed (idempotency).

	basename := filepath.Base(filename)

	if p.processed.Has(basename) {
		log.Printf("File already processed (idempotent): %s", filename)

		return nil
	}

	// Track this task AFTER shutdown check to avoid phantom tasks.

	p.taskWg.Add(1)

	p.tasksQueued.Add(1)

	// Double-check shutdown after task tracking to handle race condition.

	if p.gracefulShutdown.Load() {
		p.taskWg.Done()

		p.tasksQueued.Add(-1)

		return fmt.Errorf("processor is shutting down")
	}

	// Wait for coordinator to be ready to accept work.

	select {
	case <-p.coordReady:

		// Coordinator is ready.

	case <-p.ctx.Done():

		p.taskWg.Done()

		p.tasksQueued.Add(-1)

		return fmt.Errorf("processor context cancelled")

	case <-time.After(5 * time.Second):

		p.taskWg.Done()

		p.tasksQueued.Add(-1)

		return fmt.Errorf("timeout waiting for batch coordinator to start")
	}

	// For normal operation, use non-blocking send with minimal retry.

	// Only use timeout during shutdown to drain remaining work.

	sendTimeout := p.config.SendTimeout

	if sendTimeout == 0 {
		sendTimeout = 5 * time.Second

		// Reduce timeout on Windows during normal operation.

		if runtime.GOOS == "windows" && !p.gracefulShutdown.Load() {
			sendTimeout = 2 * time.Second
		}

		if runtime.GOOS == "windows" && p.gracefulShutdown.Load() {
			sendTimeout = 10 * time.Second
		}
	}

	// Try to send with exponential backoff.

	backoff := time.Millisecond * 100

	maxBackoff := sendTimeout / 2

	// During shutdown, use shorter timeout; during normal ops, try immediate send first.

	if !p.gracefulShutdown.Load() {
		// Normal operation: try immediate send, then short backoff.

		select {
		case p.inCh <- filename:

			return nil

		case <-p.ctx.Done():

			p.taskWg.Done()

			p.tasksQueued.Add(-1)

			return fmt.Errorf("processor context cancelled")

		default:

			// Channel full, but during normal ops this should be rare.
		}
	}

	deadline := time.Now().Add(sendTimeout)

	for time.Now().Before(deadline) {
		select {
		case p.inCh <- filename:

			return nil

		case <-p.ctx.Done():

			p.taskWg.Done()

			p.tasksQueued.Add(-1)

			return fmt.Errorf("processor context cancelled")

		default:

			time.Sleep(backoff)

			backoff *= 2

			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		}
	}

	p.taskWg.Done()

	p.tasksQueued.Add(-1)

	return fmt.Errorf("timeout sending file to batch coordinator (buffer full after %v)", sendTimeout)
}

// processBatch processes a batch of files.

func (p *IntentProcessor) processBatch(files []string) {
	if len(files) == 0 {
		return
	}

	log.Printf("Processing batch of %d files", len(files))

	for _, file := range files {
		if err := p.processSingleFile(file); err != nil {
			log.Printf("Error processing %s: %v", file, err)

			// Continue processing other files.
		}

		// Mark task as complete.

		p.taskWg.Done()

		p.tasksQueued.Add(-1)
	}
}

// processSingleFile handles the actual processing of one file.

func (p *IntentProcessor) processSingleFile(filename string) error {
	// Read file with retry logic for Windows race conditions.

	data, err := readFileWithRetry(filename)
	if err != nil {
		// If file disappeared, it was likely processed by another worker.

		if errors.Is(err, ErrFileGone) {
			log.Printf("File %s disappeared (likely processed by another worker)", filename)

			return nil
		}

		return p.handleError(filename, fmt.Errorf("failed to read file: %w", err))
	}

	// Validate using the same validation as admission webhook.

	if p.validator == nil {
		return p.handleError(filename, fmt.Errorf("validator is nil"))
	}

	intent, err := p.validator.ValidateBytes(data)
	if err != nil {
		return p.handleError(filename, fmt.Errorf("validation failed: %w", err))
	}

	// Submit to porch.

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	defer cancel()

	if p.config == nil {
		return p.handleError(filename, fmt.Errorf("config is nil"))
	}

	if p.porchFunc == nil {
		return p.handleError(filename, fmt.Errorf("porchFunc is nil"))
	}

	var submitErr error

	for retry := range p.config.MaxRetries {
		if retry > 0 {
			time.Sleep(time.Duration(retry) * time.Second)
		}

		submitErr = p.porchFunc(ctx, intent, p.config.PorchMode)

		if submitErr == nil {
			break
		}

		log.Printf("Retry %d/%d for %s: %v", retry+1, p.config.MaxRetries, filename, submitErr)
	}

	if submitErr != nil {
		return p.handleError(filename, fmt.Errorf("porch submission failed after %d retries: %w", p.config.MaxRetries, submitErr))
	}

	// Mark as processed.

	p.markProcessed(filename)

	log.Printf("Successfully processed: %s", filename)

	return nil
}

// handleError writes error details to error directory.

func (p *IntentProcessor) handleError(filename string, err error) error {
	basename := filepath.Base(filename)

	timestamp := time.Now().Format("20060102T150405")

	// Check if this is a shutdown failure.

	isShutdownErr := p.IsShutdownFailure(err)

	// Increment appropriate counter atomically.

	if isShutdownErr {
		atomic.AddInt64(&p.shutdownFailedCount, 1)
	} else {
		atomic.AddInt64(&p.realFailedCount, 1)
	}

	atomic.AddInt64(&p.failedCount, 1)

	// Prepare error content with shutdown marker if needed.

	var errorContent string

	if isShutdownErr {
		errorContent = fmt.Sprintf("SHUTDOWN_FAILURE: %v\nFile: %s\nTime: %s\nError: %v\n",

			err, filename, time.Now().Format(time.RFC3339), err)

		log.Printf("Processor: File %s failed due to graceful shutdown (expected): %s",

			filename, err.Error())
	} else {
		errorContent = fmt.Sprintf("File: %s\nTime: %s\nError: %v\n", filename, time.Now().Format(time.RFC3339), err)

		log.Printf("Processor: File %s failed with real error: %s", filename, err.Error())
	}

	// Ensure error directory exists before writing files.

	if err := os.MkdirAll(p.config.ErrorDir, 0o755); err != nil {
		log.Printf("Failed to create error directory %s: %v", p.config.ErrorDir, err)

		return err
	}

	// Write error file with atomic operation.

	errorFile := filepath.Join(p.config.ErrorDir, fmt.Sprintf("%s.%s.error", basename, timestamp))

	if writeErr := atomicWriteFile(errorFile, []byte(errorContent), 0o644); writeErr != nil {
		log.Printf("Failed to write error file %s: %v", errorFile, writeErr)
	}

	// Copy original file to error directory with retry.

	origData, _ := readFileWithRetry(filename)

	if origData != nil {
		origCopy := filepath.Join(p.config.ErrorDir, fmt.Sprintf("%s.%s.json", basename, timestamp))

		if writeErr := atomicWriteFile(origCopy, origData, 0o644); writeErr != nil {
			log.Printf("Failed to copy original file to error dir: %v", writeErr)
		}
	}

	return err
}

// markProcessed marks a file as processed for idempotency.

func (p *IntentProcessor) markProcessed(filename string) {
	basename := filepath.Base(filename)

	p.processed.Add(basename)

	// Increment processed count atomically.

	atomic.AddInt64(&p.processedCount, 1)

	// Persist to file.

	handoffDir := "./handoff"

	if p.config != nil {
		handoffDir = p.config.HandoffDir
	}

	processedFile := filepath.Join(handoffDir, ".processed")

	f, err := os.OpenFile(processedFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		log.Printf("Failed to open processed file: %v", err)

		return
	}

	defer f.Close() // #nosec G307 - Error handled in defer

	if _, err := f.WriteString(basename + "\n"); err != nil {
		log.Printf("Failed to write to processed file: %v", err)
	}
}

// StartBatchProcessor starts the background batch coordinator.

func (p *IntentProcessor) StartBatchProcessor() {
	p.wg.Add(1)

	go func() {
		defer p.wg.Done()

		// Single owner of the batch slice.

		batch := make([]string, 0, p.config.BatchSize)

		ticker := time.NewTicker(p.config.BatchInterval)

		defer ticker.Stop()

		// Signal that coordinator is ready.

		close(p.coordReady)

		for {
			select {
			case file := <-p.inCh:

				// Append to batch (only this goroutine touches the batch).

				batch = append(batch, file)

				// Check if batch is full.

				if len(batch) >= p.config.BatchSize {
					p.processBatch(batch)

					batch = batch[:0] // Reset batch
				}

			case <-ticker.C:

				// Flush batch on interval.

				if len(batch) > 0 {
					p.processBatch(batch)

					batch = batch[:0] // Reset batch
				}

			case <-p.stopCh:

				// Flush remaining batch before stopping.

				if len(batch) > 0 {
					p.processBatch(batch)
				}

				return

			case <-p.ctx.Done():

				// Context cancelled, flush and exit.

				if len(batch) > 0 {
					p.processBatch(batch)
				}

				return
			}
		}
	}()
}

// Stop implements graceful shutdown with proper drain sequencing:.

// 1. Allow current tasks to start processing (brief delay).

// 2. Stop accepting new files (mark shutdown).

// 3. Wait for queued tasks to complete (drain with timeout).

// 4. Stop coordinator and cancel context.

// 5. Wait for all goroutines to finish.

func (p *IntentProcessor) Stop() {
	log.Printf("Processor shutdown initiated")

	// Phase 0: Allow queued tasks to start processing before blocking new ones.

	// This prevents the race where tasks are queued but shutdown begins immediately.

	queuedBeforeShutdown := p.tasksQueued.Load()

	if queuedBeforeShutdown > 0 {
		log.Printf("Allowing %d queued tasks to start processing before shutdown", queuedBeforeShutdown)

		// Give workers a moment to pick up queued work.

		time.Sleep(100 * time.Millisecond)
	}

	// Phase 1: Stop accepting new files (mark shutdown).

	log.Printf("Stopping new file acceptance")

	p.MarkGracefulShutdown()

	// Phase 2: Wait for all queued tasks to drain with timeout.

	queuedCount := p.tasksQueued.Load()

	log.Printf("Processor drain phase: waiting for %d queued tasks to complete", queuedCount)

	// Use a timeout channel to prevent indefinite blocking.

	drainTimeout := 8 * time.Second

	drainDone := make(chan struct{})

	go func() {
		p.taskWg.Wait()

		close(drainDone)
	}()

	select {
	case <-drainDone:

		log.Printf("Processor drain completed - all tasks processed")

	case <-time.After(drainTimeout):

		remaining := p.tasksQueued.Load()

		log.Printf("Processor drain timeout after %v - %d tasks may still be processing", drainTimeout, remaining)
	}

	// Phase 3: Stop coordinator and cancel context.

	log.Printf("Processor shutdown phase: stopping coordinator and cancelling context")

	select {
	case <-p.stopCh:

		// Already closed.

	default:

		close(p.stopCh)
	}

	p.cancel()

	// Phase 4: Wait for all background goroutines.

	p.wg.Wait()

	log.Printf("Processor stopped gracefully")
}

// DefaultPorchSubmit is the default porch submission function.

func DefaultPorchSubmit(ctx context.Context, intent *ingest.Intent, mode string) error {
	// TODO: Fix WriteIntent signature mismatch - needs logger parameter
	// This function is a stub until the proper porch writer implementation is available
	_ = mode // suppress unused variable warning
	return fmt.Errorf("WriteIntent not implemented - signature mismatch")
}

// GetStats returns processing statistics using atomic counters.

func (p *IntentProcessor) GetStats() (ProcessingStats, error) {
	// Use atomic counters for accurate real-time stats.

	processedCount := atomic.LoadInt64(&p.processedCount)

	failedCount := atomic.LoadInt64(&p.failedCount)

	realFailedCount := atomic.LoadInt64(&p.realFailedCount)

	shutdownFailedCount := atomic.LoadInt64(&p.shutdownFailedCount)

	return ProcessingStats{
		ProcessedCount: int(processedCount),

		FailedCount: int(failedCount),

		ShutdownFailedCount: int(shutdownFailedCount),

		RealFailedCount: int(realFailedCount),

		ProcessedFiles: []string{}, // Can be populated if needed

		FailedFiles: []string{}, // Can be populated if needed

		ShutdownFailedFiles: []string{}, // Can be populated if needed

		RealFailedFiles: []string{}, // Can be populated if needed

	}, nil
}

// getProcessedFiles returns list of processed files from .processed file.

func (p *IntentProcessor) getProcessedFiles() ([]string, error) {
	processedFile := filepath.Join(p.config.HandoffDir, ".processed")

	data, err := os.ReadFile(processedFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}

		return nil, err
	}

	lines := strings.Split(string(data), "\n")

	var files []string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if line != "" {
			files = append(files, line)
		}
	}

	return files, nil
}

// getFailedFiles returns list of failed files from error directory.

func (p *IntentProcessor) getFailedFiles() ([]string, error) {
	entries, err := os.ReadDir(p.config.ErrorDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}

		return nil, err
	}

	var files []string

	seen := make(map[string]bool)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Look for .json files (original files copied to error dir).

		if strings.HasSuffix(name, ".json") {
			// Extract base name (remove timestamp suffix).

			parts := strings.Split(name, ".")

			if len(parts) >= 3 {
				// Format: basename.timestamp.json.

				baseName := strings.Join(parts[:len(parts)-2], ".")

				if !seen[baseName] {
					files = append(files, filepath.Join(p.config.ErrorDir, name))

					seen[baseName] = true
				}
			}
		}
	}

	return files, nil
}

// isShutdownFailure checks if a failed file was caused by graceful shutdown.

func (p *IntentProcessor) isShutdownFailure(failedFilePath string) bool {
	// Look for corresponding error file.

	basename := filepath.Base(failedFilePath)

	// Remove .json suffix and find .error.log file.

	if strings.HasSuffix(basename, ".json") {
		baseWithoutExt := strings.TrimSuffix(basename, ".json")

		// Look for the specific error log file (consistent with FileManager)
		errorLogFile := baseWithoutExt + ".json.error.log"
		errorLogPath := filepath.Join(p.config.ErrorDir, errorLogFile)
		
		// Read the error log content directly
		errorContent, err := os.ReadFile(errorLogPath)
		if err != nil {
			return false
		}

		errorMsg := string(errorContent)

		// Check for shutdown failure patterns.
		return strings.Contains(errorMsg, "SHUTDOWN_FAILURE:") ||
			strings.Contains(strings.ToLower(errorMsg), "context canceled") ||
			strings.Contains(strings.ToLower(errorMsg), "context cancelled") ||
			strings.Contains(strings.ToLower(errorMsg), "signal: killed") ||
			strings.Contains(strings.ToLower(errorMsg), "signal: interrupt") ||
			strings.Contains(strings.ToLower(errorMsg), "signal: terminated")
	}

	return false
}


// MarkGracefulShutdown marks that graceful shutdown has started.

func (p *IntentProcessor) MarkGracefulShutdown() {
	p.shutdownMutex.Lock()

	p.gracefulShutdown.Store(true)

	p.shutdownStartTime = time.Now()

	p.shutdownMutex.Unlock()

	log.Printf("Processor graceful shutdown initiated at %s", p.shutdownStartTime.Format(time.RFC3339))
}

// IsShutdownFailure determines if a processing failure was caused by graceful shutdown.

// This delegates to the centralized IsShutdownFailure function.

func (p *IntentProcessor) IsShutdownFailure(err error) bool {
	return IsShutdownFailure(p.gracefulShutdown.Load(), err)
}
