package loop

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	shouldFlush := p.config != nil && len(p.batch) >= p.config.BatchSize
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
	if p.validator == nil {
		return p.handleError(filename, fmt.Errorf("validator is nil"))
	}
	intent, err := p.validator.ValidateBytes(data)
	if err != nil {
		return p.handleError(filename, fmt.Errorf("validation failed: %w", err))
	}

	// Submit to porch
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if p.config == nil {
		return p.handleError(filename, fmt.Errorf("config is nil"))
	}
	if p.porchFunc == nil {
		return p.handleError(filename, fmt.Errorf("porchFunc is nil"))
	}

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
	
	// Write error file
	errorDir := "./handoff/errors"
	if p.config != nil {
		errorDir = p.config.ErrorDir
	}
	errorFile := filepath.Join(errorDir, fmt.Sprintf("%s.%s.error", basename, timestamp))
	errorContent := fmt.Sprintf("File: %s\nTime: %s\nError: %v\n", filename, time.Now().Format(time.RFC3339), err)
	
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
	handoffDir := "./handoff"
	if p.config != nil {
		handoffDir = p.config.HandoffDir
	}
	processedFile := filepath.Join(handoffDir, ".processed")
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
		batchInterval := 5 * time.Second
		if p.config != nil {
			batchInterval = p.config.BatchInterval
		}
		ticker := time.NewTicker(batchInterval)
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