package loop

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/thc1006/nephoran-intent-operator/internal/ingest"
	"github.com/thc1006/nephoran-intent-operator/internal/porch"
)

// ProcessorConfig holds configuration for the intent processor
type ProcessorConfig struct {
	HandoffDir    string
	ErrorDir      string
	PorchMode     string // "direct" or "structured"
	BatchSize     int
	BatchInterval time.Duration
	MaxRetries    int
}

// DefaultConfig returns default processor configuration
func DefaultConfig() *ProcessorConfig {
	return &ProcessorConfig{
		HandoffDir:    "./handoff",
		ErrorDir:      "./handoff/errors",
		PorchMode:     "structured",
		BatchSize:     10,
		BatchInterval: 5 * time.Second,
		MaxRetries:    3,
	}
}

// Validator interface for intent validation
type Validator interface {
	ValidateBytes([]byte) (*ingest.Intent, error)
}

// IntentProcessor handles validation and submission of intents
type IntentProcessor struct {
	config     *ProcessorConfig
	validator  Validator
	porchFunc  PorchSubmitFunc
	batch      []string
	batchMu    sync.Mutex
	processed  *SafeSet // for idempotency
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	
	// Stats tracking
	processedCount      int64
	failedCount         int64
	realFailedCount     int64
	shutdownFailedCount int64
	statsMu            sync.RWMutex
	
	// Graceful shutdown tracking
	gracefulShutdown    atomic.Bool
	shutdownStartTime   time.Time
	shutdownMutex      sync.RWMutex
}

// PorchSubmitFunc is a function type for submitting to porch
type PorchSubmitFunc func(ctx context.Context, intent *ingest.Intent, mode string) error

// NewProcessor creates a new intent processor
func NewProcessor(config *ProcessorConfig, validator Validator, porchFunc PorchSubmitFunc) (*IntentProcessor, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure error directory exists
	if err := os.MkdirAll(config.ErrorDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create error directory: %w", err)
	}

	// Load processed intents for idempotency
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

	return &IntentProcessor{
		config:    config,
		validator: validator,
		porchFunc: porchFunc,
		batch:     make([]string, 0, config.BatchSize),
		processed: processed,
		ctx:       ctx,
		cancel:    cancel,
	}, nil
}

// ProcessFile processes a single intent file
func (p *IntentProcessor) ProcessFile(filename string) error {
	// Check if already processed (idempotency)
	basename := filepath.Base(filename)
	if p.processed.Has(basename) {
		log.Printf("File already processed (idempotent): %s", filename)
		return nil
	}

	// Add to batch
	p.batchMu.Lock()
	p.batch = append(p.batch, filename)
	shouldFlush := len(p.batch) >= p.config.BatchSize
	p.batchMu.Unlock()

	if shouldFlush {
		return p.FlushBatch()
	}

	return nil
}

// FlushBatch processes all files in the current batch
func (p *IntentProcessor) FlushBatch() error {
	p.batchMu.Lock()
	if len(p.batch) == 0 {
		p.batchMu.Unlock()
		return nil
	}
	
	files := make([]string, len(p.batch))
	copy(files, p.batch)
	p.batch = p.batch[:0]
	p.batchMu.Unlock()

	log.Printf("Processing batch of %d files", len(files))
	
	for _, file := range files {
		if err := p.processSingleFile(file); err != nil {
			log.Printf("Error processing %s: %v", file, err)
			// Continue processing other files
		}
	}

	return nil
}

// processSingleFile handles the actual processing of one file
func (p *IntentProcessor) processSingleFile(filename string) error {
	// Read file
	data, err := os.ReadFile(filename)
	if err != nil {
		return p.handleError(filename, fmt.Errorf("failed to read file: %w", err))
	}

	// Validate using the same validation as admission webhook
	intent, err := p.validator.ValidateBytes(data)
	if err != nil {
		return p.handleError(filename, fmt.Errorf("validation failed: %w", err))
	}

	// Submit to porch
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var submitErr error
	for retry := 0; retry < p.config.MaxRetries; retry++ {
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

	// Mark as processed
	p.markProcessed(filename)
	log.Printf("Successfully processed: %s", filename)
	
	return nil
}

// handleError writes error details to error directory
func (p *IntentProcessor) handleError(filename string, err error) error {
	basename := filepath.Base(filename)
	timestamp := time.Now().Format("20060102T150405")
	
	// Check if this is a shutdown failure
	isShutdownErr := p.IsShutdownFailure(err)
	
	// Prepare error content with shutdown marker if needed
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
	
	// Write error file
	errorFile := filepath.Join(p.config.ErrorDir, fmt.Sprintf("%s.%s.error", basename, timestamp))
	if writeErr := os.WriteFile(errorFile, []byte(errorContent), 0644); writeErr != nil {
		log.Printf("Failed to write error file %s: %v", errorFile, writeErr)
	}

	// Copy original file to error directory
	origData, _ := os.ReadFile(filename)
	origCopy := filepath.Join(p.config.ErrorDir, fmt.Sprintf("%s.%s.json", basename, timestamp))
	if writeErr := os.WriteFile(origCopy, origData, 0644); writeErr != nil {
		log.Printf("Failed to copy original file to error dir: %v", writeErr)
	}

	return err
}

// markProcessed marks a file as processed for idempotency
func (p *IntentProcessor) markProcessed(filename string) {
	basename := filepath.Base(filename)
	p.processed.Add(basename)

	// Persist to file
	processedFile := filepath.Join(p.config.HandoffDir, ".processed")
	f, err := os.OpenFile(processedFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open processed file: %v", err)
		return
	}
	defer f.Close()
	
	if _, err := f.WriteString(basename + "\n"); err != nil {
		log.Printf("Failed to write to processed file: %v", err)
	}
}

// StartBatchProcessor starts the background batch processor
func (p *IntentProcessor) StartBatchProcessor() {
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		ticker := time.NewTicker(p.config.BatchInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				if err := p.FlushBatch(); err != nil {
					log.Printf("Error flushing batch: %v", err)
				}
			case <-p.ctx.Done():
				// Flush remaining batch before stopping
				p.FlushBatch()
				return
			}
		}
	}()
}

// Stop stops the processor and waits for all goroutines to finish
func (p *IntentProcessor) Stop() {
	p.MarkGracefulShutdown()
	p.cancel()
	p.wg.Wait()
}

// DefaultPorchSubmit is the default porch submission function
func DefaultPorchSubmit(ctx context.Context, intent *ingest.Intent, mode string) error {
	// Determine output directory based on mode
	outDir := "./output"
	format := "full"
	if mode == "direct" {
		format = "smp"
	}

	// Use the existing porch writer
	return porch.WriteIntent(intent, outDir, format)
}

// GetStats returns processing statistics compatible with FileManager.GetStats()
func (p *IntentProcessor) GetStats() (ProcessingStats, error) {
	// Get stats from the error directory using similar logic to FileManager
	processedFiles, err := p.getProcessedFiles()
	if err != nil {
		return ProcessingStats{}, fmt.Errorf("failed to get processed files: %w", err)
	}
	
	failedFiles, err := p.getFailedFiles()
	if err != nil {
		return ProcessingStats{}, fmt.Errorf("failed to get failed files: %w", err)
	}
	
	// Categorize failed files into shutdown failures vs real failures
	var shutdownFailedFiles []string
	var realFailedFiles []string
	
	for _, failedFile := range failedFiles {
		// Check error log to determine failure type
		if p.isShutdownFailure(failedFile) {
			shutdownFailedFiles = append(shutdownFailedFiles, failedFile)
		} else {
			realFailedFiles = append(realFailedFiles, failedFile)
		}
	}
	
	return ProcessingStats{
		ProcessedCount:       len(processedFiles),
		FailedCount:          len(failedFiles),
		ShutdownFailedCount:  len(shutdownFailedFiles),
		RealFailedCount:      len(realFailedFiles),
		ProcessedFiles:       processedFiles,
		FailedFiles:          failedFiles,
		ShutdownFailedFiles:  shutdownFailedFiles,
		RealFailedFiles:      realFailedFiles,
	}, nil
}

// getProcessedFiles returns list of processed files from .processed file
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

// getFailedFiles returns list of failed files from error directory
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
		// Look for .json files (original files copied to error dir)
		if strings.HasSuffix(name, ".json") {
			// Extract base name (remove timestamp suffix)
			parts := strings.Split(name, ".")
			if len(parts) >= 3 {
				// Format: basename.timestamp.json
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

// isShutdownFailure checks if a failed file was caused by graceful shutdown
func (p *IntentProcessor) isShutdownFailure(failedFilePath string) bool {
	// Look for corresponding error file
	basename := filepath.Base(failedFilePath)
	
	// Remove .json suffix and find .error file
	if strings.HasSuffix(basename, ".json") {
		baseWithoutExt := strings.TrimSuffix(basename, ".json")
		
		// Find the corresponding .error file
		entries, err := os.ReadDir(p.config.ErrorDir)
		if err != nil {
			return false
		}
		
		for _, entry := range entries {
			name := entry.Name()
			if strings.HasPrefix(name, baseWithoutExt) && strings.HasSuffix(name, ".error") {
				errorPath := filepath.Join(p.config.ErrorDir, name)
				errorContent, err := os.ReadFile(errorPath)
				if err != nil {
					continue
				}
				
				errorMsg := string(errorContent)
				
				// Check for shutdown failure patterns
				return strings.Contains(errorMsg, "SHUTDOWN_FAILURE:") ||
					   strings.Contains(strings.ToLower(errorMsg), "context canceled") ||
					   strings.Contains(strings.ToLower(errorMsg), "context cancelled") ||
					   strings.Contains(strings.ToLower(errorMsg), "signal: killed") ||
					   strings.Contains(strings.ToLower(errorMsg), "signal: interrupt") ||
					   strings.Contains(strings.ToLower(errorMsg), "signal: terminated")
			}
		}
	}
	
	return false
}

// MarkGracefulShutdown marks that graceful shutdown has started
func (p *IntentProcessor) MarkGracefulShutdown() {
	p.shutdownMutex.Lock()
	p.gracefulShutdown.Store(true)
	p.shutdownStartTime = time.Now()
	p.shutdownMutex.Unlock()
	
	log.Printf("Processor graceful shutdown initiated at %s", p.shutdownStartTime.Format(time.RFC3339))
}

// IsShutdownFailure determines if a processing failure was caused by graceful shutdown
func (p *IntentProcessor) IsShutdownFailure(err error) bool {
	// Check if graceful shutdown is active
	if !p.gracefulShutdown.Load() {
		return false
	}
	
	// If graceful shutdown is active, any failure is likely due to shutdown
	errorMsg := strings.ToLower(err.Error())
	shutdownPatterns := []string{
		"context canceled",
		"context cancelled",
		"signal: killed",
		"signal: interrupt",
		"signal: terminated",
	}
	
	for _, pattern := range shutdownPatterns {
		if strings.Contains(errorMsg, pattern) {
			return true
		}
	}
	
	// Check timing - if failure occurs after shutdown was initiated, 
	// and it's not a clear validation error, consider it a shutdown failure
	p.shutdownMutex.RLock()
	shutdownTime := p.shutdownStartTime
	p.shutdownMutex.RUnlock()
	
	if !shutdownTime.IsZero() {
		// If error occurred after shutdown and is not a validation error, 
		// treat as shutdown failure
		isValidationError := strings.Contains(errorMsg, "validation") ||
			strings.Contains(errorMsg, "invalid") ||
			strings.Contains(errorMsg, "schema") ||
			strings.Contains(errorMsg, "format")
		
		// If it's not a validation error and shutdown is active, treat as shutdown failure
		if !isValidationError {
			return true
		}
	}
	
	return false
}