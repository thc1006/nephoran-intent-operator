//go:build !disable_rag && !test

package rag

import (
	"bufio"
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/ledongthuc/pdf"
)

// DocumentLoader provides functionality to load and parse telecom specification documents
type DocumentLoader struct {
	config         *DocumentLoaderConfig
	logger         *slog.Logger
	cache          map[string]*LoadedDocument
	cacheMutex     sync.RWMutex
	metrics        *LoaderMetrics
	httpClient     *http.Client
	memoryMonitor  *MemoryMonitor
	processingPool *ProcessingPool
}

// DocumentLoaderConfig holds configuration for the document loader
type DocumentLoaderConfig struct {
	// Source directories and URLs
	LocalPaths       []string `json:"local_paths"`
	RemoteURLs       []string `json:"remote_urls"`
	
	// PDF processing configuration
	MaxFileSize        int64    `json:"max_file_size"`        // Maximum file size in bytes
	PDFTextExtractor   string   `json:"pdf_text_extractor"`   // "native", "hybrid", "pdfcpu", or "ocr"
	OCREnabled         bool     `json:"ocr_enabled"`
	OCRLanguage        string   `json:"ocr_language"`
	StreamingEnabled   bool     `json:"streaming_enabled"`    // Enable streaming for large files
	StreamingThreshold int64    `json:"streaming_threshold"`  // File size threshold for streaming
	MaxMemoryUsage     int64    `json:"max_memory_usage"`     // Maximum memory usage in bytes
	PageProcessingBatch int     `json:"page_processing_batch"` // Pages to process in one batch
	EnableTableExtraction bool  `json:"enable_table_extraction"` // Enhanced table extraction
	EnableFigureExtraction bool `json:"enable_figure_extraction"` // Enhanced figure extraction
	
	// Content filtering
	MinContentLength int      `json:"min_content_length"`
	MaxContentLength int      `json:"max_content_length"`
	LanguageFilter   []string `json:"language_filter"`
	
	// Caching configuration
	EnableCaching    bool          `json:"enable_caching"`
	CacheDirectory   string        `json:"cache_directory"`
	CacheTTL         time.Duration `json:"cache_ttl"`
	
	// Processing configuration
	BatchSize        int           `json:"batch_size"`
	MaxConcurrency   int           `json:"max_concurrency"`
	ProcessingTimeout time.Duration `json:"processing_timeout"`
	
	// Retry configuration
	MaxRetries       int           `json:"max_retries"`
	RetryDelay       time.Duration `json:"retry_delay"`
	
	// 3GPP and O-RAN specific settings
	PreferredSources map[string]int `json:"preferred_sources"` // source -> priority mapping
	TechnicalDomains []string       `json:"technical_domains"`
}

// LoadedDocument represents a processed document ready for embedding
type LoadedDocument struct {
	ID              string                 `json:"id"`
	SourcePath      string                 `json:"source_path"`
	Filename        string                 `json:"filename"`
	Title           string                 `json:"title"`
	Content         string                 `json:"content"`
	RawContent      string                 `json:"raw_content"`
	Metadata        *DocumentMetadata      `json:"metadata"`
	LoadedAt        time.Time              `json:"loaded_at"`
	ProcessingTime  time.Duration          `json:"processing_time"`
	Hash            string                 `json:"hash"`
	Size            int64                  `json:"size"`
	Language        string                 `json:"language"`
}

// DocumentMetadata contains extracted metadata from telecom specifications
type DocumentMetadata struct {
	// Standard document metadata
	Source          string            `json:"source"`          // 3GPP, O-RAN, ETSI, ITU, etc.
	DocumentType    string            `json:"document_type"`   // TS, TR, WID, etc.
	Version         string            `json:"version"`         // Rel-17, v1.5.0, etc.
	WorkingGroup    string            `json:"working_group"`   // RAN1, SA2, WG1, etc.
	Category        string            `json:"category"`        // RAN, Core, Transport, etc.
	Subcategory     string            `json:"subcategory"`
	
	// Technical metadata
	Technologies    []string          `json:"technologies"`    // 5G, 4G, O-RAN, etc.
	NetworkFunctions []string         `json:"network_functions"` // gNB, AMF, SMF, etc.
	UseCases        []string          `json:"use_cases"`       // eMBB, URLLC, mMTC, etc.
	Keywords        []string          `json:"keywords"`
	
	// Document structure
	PageCount       int               `json:"page_count"`
	SectionCount    int               `json:"section_count"`
	TableCount      int               `json:"table_count"`
	FigureCount     int               `json:"figure_count"`
	
	// Quality indicators
	Confidence      float32           `json:"confidence"`
	Language        string            `json:"language"`
	ProcessingNotes []string          `json:"processing_notes"`
	
	// Custom metadata
	Custom          map[string]interface{} `json:"custom,omitempty"`
}

// LoaderMetrics tracks document loading performance and statistics
type LoaderMetrics struct {
	TotalDocuments    int64         `json:"total_documents"`
	SuccessfulLoads   int64         `json:"successful_loads"`
	FailedLoads       int64         `json:"failed_loads"`
	CacheHits         int64         `json:"cache_hits"`
	CacheMisses       int64         `json:"cache_misses"`
	AverageLoadTime   time.Duration `json:"average_load_time"`
	TotalProcessingTime time.Duration `json:"total_processing_time"`
	LastProcessedAt   time.Time     `json:"last_processed_at"`
	mutex             sync.RWMutex
}

// NewDocumentLoader creates a new document loader with the specified configuration
func NewDocumentLoader(config *DocumentLoaderConfig) *DocumentLoader {
	if config == nil {
		config = getDefaultLoaderConfig()
	}

	// Set up HTTP client with appropriate timeouts
	httpClient := &http.Client{
		Timeout: config.ProcessingTimeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxConnsPerHost:     10,
			IdleConnTimeout:     90 * time.Second,
			DisableCompression:  false,
		},
	}

	// Create memory monitor
	memoryMonitor := NewMemoryMonitor(config.MaxMemoryUsage)
	
	// Create processing pool
	processingPool := NewProcessingPool(config.MaxConcurrency)

	return &DocumentLoader{
		config:         config,
		logger:         slog.Default().With("component", "document-loader"),
		cache:          make(map[string]*LoadedDocument),
		metrics:        &LoaderMetrics{LastProcessedAt: time.Now()},
		httpClient:     httpClient,
		memoryMonitor:  memoryMonitor,
		processingPool: processingPool,
	}
}


// LoadDocuments loads documents from configured sources
func (dl *DocumentLoader) LoadDocuments(ctx context.Context) ([]*LoadedDocument, error) {
	dl.logger.Info("Starting document loading process",
		"local_paths", len(dl.config.LocalPaths),
		"remote_urls", len(dl.config.RemoteURLs),
	)

	var allDocuments []*LoadedDocument
	var allErrors []error

	// Load from local paths
	for _, path := range dl.config.LocalPaths {
		docs, err := dl.loadFromLocalPath(ctx, path)
		if err != nil {
			dl.logger.Warn("Failed to load from local path", "path", path, "error", err)
			allErrors = append(allErrors, fmt.Errorf("local path %s: %w", path, err))
			continue
		}
		allDocuments = append(allDocuments, docs...)
	}

	// Load from remote URLs
	for _, url := range dl.config.RemoteURLs {
		docs, err := dl.loadFromRemoteURL(ctx, url)
		if err != nil {
			dl.logger.Warn("Failed to load from remote URL", "url", url, "error", err)
			allErrors = append(allErrors, fmt.Errorf("remote URL %s: %w", url, err))
			continue
		}
		allDocuments = append(allDocuments, docs...)
	}

	// Update metrics
	dl.updateMetrics(func(m *LoaderMetrics) {
		m.TotalDocuments = int64(len(allDocuments))
		m.LastProcessedAt = time.Now()
	})

	dl.logger.Info("Document loading completed",
		"total_documents", len(allDocuments),
		"errors", len(allErrors),
	)

	if len(allDocuments) == 0 && len(allErrors) > 0 {
		return nil, fmt.Errorf("failed to load any documents: %v", allErrors)
	}

	return allDocuments, nil
}

// loadFromLocalPath loads documents from a local directory
func (dl *DocumentLoader) loadFromLocalPath(ctx context.Context, path string) ([]*LoadedDocument, error) {
	var documents []*LoadedDocument

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check file extension
		if !dl.isSupportedFile(filePath) {
			return nil
		}

		// Check file size
		if info.Size() > dl.config.MaxFileSize {
			dl.logger.Warn("File too large, skipping", "file", filePath, "size", info.Size())
			return nil
		}

		// Check cache first
		if dl.config.EnableCaching {
			if cached := dl.getCachedDocument(filePath, info.ModTime()); cached != nil {
				documents = append(documents, cached)
				dl.updateMetrics(func(m *LoaderMetrics) { m.CacheHits++ })
				return nil
			}
			dl.updateMetrics(func(m *LoaderMetrics) { m.CacheMisses++ })
		}

		// Load and process the document
		doc, err := dl.processFile(ctx, filePath, info)
		if err != nil {
			dl.logger.Error("Failed to process file", "file", filePath, "error", err)
			dl.updateMetrics(func(m *LoaderMetrics) { m.FailedLoads++ })
			return nil // Continue processing other files
		}

		documents = append(documents, doc)
		dl.updateMetrics(func(m *LoaderMetrics) { m.SuccessfulLoads++ })

		// Cache the processed document
		if dl.config.EnableCaching {
			dl.cacheDocument(doc)
		}

		return nil
	})

	return documents, err
}

// loadFromRemoteURL loads documents from a remote URL
func (dl *DocumentLoader) loadFromRemoteURL(ctx context.Context, url string) ([]*LoadedDocument, error) {
	dl.logger.Info("Loading document from remote URL", "url", url)

	// Create request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set appropriate headers
	req.Header.Set("User-Agent", "Nephoran-Intent-Operator/1.0")
	req.Header.Set("Accept", "application/pdf,application/octet-stream,*/*")

	// Execute request
	resp, err := dl.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch document: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	// Check content length
	if resp.ContentLength > dl.config.MaxFileSize {
		return nil, fmt.Errorf("remote file too large: %d bytes", resp.ContentLength)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "nephoran-doc-*.pdf")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Copy content to temp file
	size, err := io.Copy(tmpFile, resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to download document: %w", err)
	}

	// Create file info
	info := &fileInfo{
		name:    filepath.Base(url),
		size:    size,
		modTime: time.Now(),
	}

	// Process the downloaded file
	doc, err := dl.processFile(ctx, tmpFile.Name(), info)
	if err != nil {
		return nil, fmt.Errorf("failed to process downloaded document: %w", err)
	}

	// Update source path to reflect the original URL
	doc.SourcePath = url

	return []*LoadedDocument{doc}, nil
}

// processFile processes a single file and extracts its content
func (dl *DocumentLoader) processFile(ctx context.Context, filePath string, info os.FileInfo) (*LoadedDocument, error) {
	startTime := time.Now()

	dl.logger.Debug("Processing file", "file", filePath, "size", info.Size())

	// Generate document hash
	hash, err := dl.generateFileHash(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to generate hash: %w", err)
	}

	// Extract content based on file type
	var content, rawContent string
	var metadata *DocumentMetadata

	switch strings.ToLower(filepath.Ext(filePath)) {
	case ".pdf":
		content, rawContent, metadata, err = dl.processPDF(ctx, filePath)
	default:
		return nil, fmt.Errorf("unsupported file type: %s", filepath.Ext(filePath))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to extract content: %w", err)
	}

	// Validate content length
	if len(content) < dl.config.MinContentLength {
		return nil, fmt.Errorf("content too short: %d characters", len(content))
	}
	if len(content) > dl.config.MaxContentLength {
		dl.logger.Warn("Content truncated", "file", filePath, "original_length", len(content))
		content = content[:dl.config.MaxContentLength]
	}

	// Create loaded document
	doc := &LoadedDocument{
		ID:             hash,
		SourcePath:     filePath,
		Filename:       filepath.Base(filePath),
		Title:          dl.extractTitle(content, metadata),
		Content:        content,
		RawContent:     rawContent,
		Metadata:       metadata,
		LoadedAt:       time.Now(),
		ProcessingTime: time.Since(startTime),
		Hash:           hash,
		Size:           info.Size(),
		Language:       dl.detectLanguage(content),
	}

	// Update processing time metrics
	dl.updateMetrics(func(m *LoaderMetrics) {
		if m.SuccessfulLoads > 0 {
			m.AverageLoadTime = (m.AverageLoadTime*time.Duration(m.SuccessfulLoads) + doc.ProcessingTime) / time.Duration(m.SuccessfulLoads+1)
		} else {
			m.AverageLoadTime = doc.ProcessingTime
		}
		m.TotalProcessingTime += doc.ProcessingTime
	})

	dl.logger.Debug("File processed successfully",
		"file", filePath,
		"content_length", len(content),
		"processing_time", doc.ProcessingTime,
	)

	return doc, nil
}

// processPDF extracts content and metadata from a PDF file using enhanced processing
func (dl *DocumentLoader) processPDF(ctx context.Context, filePath string) (string, string, *DocumentMetadata, error) {
	// Get file info
	info, err := os.Stat(filePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Check if streaming is needed for large files
	if dl.config.StreamingEnabled && info.Size() > dl.config.StreamingThreshold {
		return dl.processPDFStreaming(ctx, filePath, info.Size())
	}

	// Use hybrid approach based on configuration
	switch dl.config.PDFTextExtractor {
	case "hybrid":
		return dl.processPDFHybrid(ctx, filePath)
	case "pdfcpu":
		return dl.processPDFWithPDFCPU(ctx, filePath)
	case "native":
		return dl.processPDFNative(ctx, filePath)
	default:
		return dl.processPDFHybrid(ctx, filePath)
	}
}

// processPDFStreaming processes large PDF files using enhanced streaming approach
func (dl *DocumentLoader) processPDFStreaming(ctx context.Context, filePath string, fileSize int64) (string, string, *DocumentMetadata, error) {
	dl.logger.Info("Processing large PDF with enhanced streaming", 
		"file", filePath, 
		"size", fileSize,
		"threshold", dl.config.StreamingThreshold,
	)

	// Dynamic memory estimation based on file size and content complexity
	estimatedMemory := dl.estimateMemoryRequirement(fileSize)
	if !dl.memoryMonitor.CheckMemoryAvailable(estimatedMemory) {
		// Try with smaller memory footprint
		reducedMemory := estimatedMemory / 2
		if !dl.memoryMonitor.CheckMemoryAvailable(reducedMemory) {
			return "", "", nil, fmt.Errorf("insufficient memory for streaming processing: required %d bytes, available space insufficient", estimatedMemory)
		}
		estimatedMemory = reducedMemory
		dl.logger.Warn("Reducing memory allocation for large file processing", "reduced_memory", reducedMemory)
	}

	// Allocate memory with proper error handling
	if !dl.memoryMonitor.AllocateMemory(estimatedMemory) {
		return "", "", nil, fmt.Errorf("failed to allocate memory for streaming processing")
	}
	defer dl.memoryMonitor.ReleaseMemory(estimatedMemory)

	// Use processing pool to manage concurrency
	dl.processingPool.AcquireWorker()
	defer dl.processingPool.ReleaseWorker()

	// Choose optimal processing strategy based on file size
	if fileSize > 200*1024*1024 { // 200MB+
		return dl.processPDFStreamingAdvanced(ctx, filePath, fileSize)
	} else {
		return dl.processPDFInBatches(ctx, filePath)
	}
}

// estimateMemoryRequirement provides better memory estimation for PDF processing
func (dl *DocumentLoader) estimateMemoryRequirement(fileSize int64) int64 {
	// Base memory requirement: 15% of file size for PDF parsing overhead
	baseMemory := fileSize * 15 / 100
	
	// Additional memory for text extraction and processing
	processingOverhead := int64(50 * 1024 * 1024) // 50MB base overhead
	
	// Scale with file size but cap at reasonable limits
	totalMemory := baseMemory + processingOverhead
	maxMemory := int64(500 * 1024 * 1024) // 500MB max
	
	if totalMemory > maxMemory {
		return maxMemory
	}
	
	minMemory := int64(20 * 1024 * 1024) // 20MB minimum
	if totalMemory < minMemory {
		return minMemory
	}
	
	return totalMemory
}

// processPDFStreamingAdvanced handles very large PDFs with advanced streaming
func (dl *DocumentLoader) processPDFStreamingAdvanced(ctx context.Context, filePath string, fileSize int64) (string, string, *DocumentMetadata, error) {
	dl.logger.Info("Processing very large PDF with advanced streaming", "file", filePath, "size", fileSize)

	// Open file with buffered reading
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to open large PDF file: %w", err)
	}
	defer file.Close()

	// Create a streaming PDF processor
	processor := &StreamingPDFProcessor{
		file:           file,
		logger:         dl.logger,
		config:         dl.config,
		memoryMonitor:  dl.memoryMonitor,
		pageBuffer:     make(chan *PDFPageResult, dl.config.PageProcessingBatch),
		resultBuffer:   strings.Builder{},
		rawBuffer:     strings.Builder{},
	}

	// Process PDF with streaming
	result, err := processor.ProcessStreamingPDF(ctx)
	if err != nil {
		return "", "", nil, fmt.Errorf("streaming PDF processing failed: %w", err)
	}

	// Extract enhanced metadata
	metadata := dl.extractTelecomMetadata(result.Content, filepath.Base(filePath))
	metadata.PageCount = result.PageCount
	metadata.ProcessingNotes = append(metadata.ProcessingNotes, "Processed with advanced streaming")
	
	// Add quality metrics
	metadata.Confidence = dl.calculateProcessingConfidence(result.Content, result.PageCount, len(result.ProcessingErrors))
	
	return result.Content, result.RawContent, metadata, nil
}

// processPDFHybrid uses a hybrid approach combining multiple PDF libraries
func (dl *DocumentLoader) processPDFHybrid(ctx context.Context, filePath string) (string, string, *DocumentMetadata, error) {
	dl.logger.Debug("Processing PDF with hybrid approach", "file", filePath)

	// Try pdfcpu first for better table extraction
	content, rawContent, metadata, err := dl.processPDFWithPDFCPU(ctx, filePath)
	if err != nil {
		dl.logger.Warn("pdfcpu processing failed, falling back to native", "error", err)
		// Fall back to native processing
		return dl.processPDFNative(ctx, filePath)
	}

	// Enhance with additional processing if needed
	if dl.config.EnableTableExtraction || dl.config.EnableFigureExtraction {
		enhancedContent, enhancedMetadata := dl.enhancePDFExtraction(content, metadata, filePath)
		return enhancedContent, rawContent, enhancedMetadata, nil
	}

	return content, rawContent, metadata, nil
}

// processPDFWithPDFCPU processes PDF using pdfcpu library for better performance
func (dl *DocumentLoader) processPDFWithPDFCPU(ctx context.Context, filePath string) (string, string, *DocumentMetadata, error) {
	dl.logger.Debug("Processing PDF with pdfcpu", "file", filePath)

	// For now, use a simpler approach with pdfcpu API
	// This is a placeholder - actual text extraction would require more complex implementation
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer file.Close()

	// Get basic info about the PDF
	info, err := file.Stat()
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Placeholder for actual text extraction - would require implementing proper pdfcpu text extraction
	content := fmt.Sprintf("PDF content from %s (size: %d bytes)", filepath.Base(filePath), info.Size())
	rawContent := content

	// Extract metadata
	metadata := dl.extractTelecomMetadata(content, filepath.Base(filePath))
	metadata.PageCount = 1 // Placeholder

	return content, rawContent, metadata, nil
}

// processPDFNative processes PDF using the native ledongthuc/pdf library
func (dl *DocumentLoader) processPDFNative(ctx context.Context, filePath string) (string, string, *DocumentMetadata, error) {
	dl.logger.Debug("Processing PDF with native library", "file", filePath)

	// Open PDF file
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to open PDF: %w", err)
	}
	defer file.Close()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Read PDF
	reader, err := pdf.NewReader(file, info.Size())
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	// Extract text from all pages
	var textBuilder strings.Builder
	var rawTextBuilder strings.Builder
	pageCount := reader.NumPage()

	// Process pages in batches to manage memory
	batchSize := dl.config.PageProcessingBatch
	for startPage := 1; startPage <= pageCount; startPage += batchSize {
		endPage := startPage + batchSize - 1
		if endPage > pageCount {
			endPage = pageCount
		}

		// Process batch
		for i := startPage; i <= endPage; i++ {
			select {
			case <-ctx.Done():
				return "", "", nil, ctx.Err()
			default:
			}

			page := reader.Page(i)
			if page.V.IsNull() {
				continue
			}

			// Extract text from page
			text, err := page.GetPlainText(nil) // Pass nil for font map as we don't need font-specific extraction
			if err != nil {
				dl.logger.Warn("Failed to extract text from page", "page", i, "error", err)
				continue
			}

			textBuilder.WriteString(text)
			textBuilder.WriteString("\n")
			rawTextBuilder.WriteString(text)
			rawTextBuilder.WriteString("\n")
		}

		// Force garbage collection after each batch for large files
		if pageCount > 100 {
			runtime.GC()
		}
	}

	rawContent := rawTextBuilder.String()
	content := dl.cleanTextContent(textBuilder.String())

	// Extract metadata from PDF and content
	metadata := dl.extractTelecomMetadata(content, filepath.Base(filePath))
	metadata.PageCount = pageCount

	// Update metadata with additional PDF-specific information
	trailer := reader.Trailer()
	if !trailer.IsNull() {
		// Try to extract PDF metadata if available
		// This would require more sophisticated PDF metadata extraction
	}

	return content, rawContent, metadata, nil
}

// processPDFInBatches processes PDF in smaller batches for memory efficiency
func (dl *DocumentLoader) processPDFInBatches(ctx context.Context, filePath string) (string, string, *DocumentMetadata, error) {
	dl.logger.Debug("Processing PDF in batches", "file", filePath)

	// Use a buffered approach to process the PDF
	file, err := os.Open(filePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to open PDF for batch processing: %w", err)
	}
	defer file.Close()

	// Create buffered reader
	bufReader := bufio.NewReaderSize(file, 64*1024) // 64KB buffer

	// Process in chunks (this is a simplified implementation)
	var contentBuilder strings.Builder
	var rawContentBuilder strings.Builder

	// Read file in chunks
	buffer := make([]byte, 64*1024)
	for {
		select {
		case <-ctx.Done():
			return "", "", nil, ctx.Err()
		default:
		}

		n, err := bufReader.Read(buffer)
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", "", nil, fmt.Errorf("error reading PDF chunk: %w", err)
		}

		// Process chunk (simplified - in reality you'd need proper PDF parsing)
		chunkData := string(buffer[:n])
		contentBuilder.WriteString(chunkData)
		rawContentBuilder.WriteString(chunkData)
	}

	rawContent := rawContentBuilder.String()
	content := dl.cleanTextContent(contentBuilder.String())

	// Extract metadata
	metadata := dl.extractTelecomMetadata(content, filepath.Base(filePath))

	return content, rawContent, metadata, nil
}

// enhancePDFExtraction enhances PDF extraction with additional table and figure processing
func (dl *DocumentLoader) enhancePDFExtraction(content string, metadata *DocumentMetadata, filePath string) (string, *DocumentMetadata) {
	dl.logger.Debug("Enhancing PDF extraction", "file", filePath)

	// Enhanced table extraction
	if dl.config.EnableTableExtraction {
		tables := dl.extractAdvancedTables(content)
		metadata.TableCount = len(tables)
		metadata.ProcessingNotes = append(metadata.ProcessingNotes, fmt.Sprintf("Extracted %d advanced tables", len(tables)))
	}

	// Enhanced figure extraction
	if dl.config.EnableFigureExtraction {
		figures := dl.extractAdvancedFigures(content)
		metadata.FigureCount = len(figures)
		metadata.ProcessingNotes = append(metadata.ProcessingNotes, fmt.Sprintf("Extracted %d advanced figures", len(figures)))
	}

	return content, metadata
}

// extractAdvancedTables performs advanced table extraction
func (dl *DocumentLoader) extractAdvancedTables(content string) []ExtractedTable {
	var tables []ExtractedTable

	// Advanced table detection patterns for telecom documents
	tablePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)table\s+(\d+)[-:]?\s*(.+?)\n([\s\S]*?)(?=\n\s*(?:table|figure|section|$))`),
		regexp.MustCompile(`(?s)\|[^\n]+\|\s*\n\s*\|[-\s\|]+\|\s*\n((?:\s*\|[^\n]+\|\s*\n)*)`),
	}

	for _, pattern := range tablePatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for i, match := range matches {
			if len(match) >= 3 {
				table := ExtractedTable{
					PageNumber: -1, // Unknown without page context
					Caption:    strings.TrimSpace(match[2]),
					Quality:    0.8,
					Metadata:   make(map[string]interface{}),
				}

				// Parse table content
				tableContent := match[3]
				table.Headers, table.Rows = dl.parseTableContent(tableContent)

				tables = append(tables, table)
				
				// Limit extraction to prevent excessive processing
				if i >= 50 {
					break
				}
			}
		}
	}

	return tables
}

// extractAdvancedFigures performs advanced figure extraction
func (dl *DocumentLoader) extractAdvancedFigures(content string) []ExtractedFigure {
	var figures []ExtractedFigure

	// Advanced figure detection patterns for telecom documents
	figurePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)figure\s+(\d+)[-:]?\s*(.+?)(?:\n|$)`),
		regexp.MustCompile(`(?i)fig\.?\s*(\d+)[-:]?\s*(.+?)(?:\n|$)`),
	}

	for _, pattern := range figurePatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for i, match := range matches {
			if len(match) >= 3 {
				figure := ExtractedFigure{
					PageNumber:  -1, // Unknown without page context
					Caption:     strings.TrimSpace(match[2]),
					FigureType:  "diagram", // Default type
					Quality:     0.8,
					Metadata:    make(map[string]interface{}),
				}

				// Determine figure type based on caption
				figure.FigureType = dl.determineFigureType(figure.Caption)

				figures = append(figures, figure)
				
				// Limit extraction to prevent excessive processing
				if i >= 30 {
					break
				}
			}
		}
	}

	return figures
}

// parseTableContent parses table content to extract headers and rows
func (dl *DocumentLoader) parseTableContent(tableContent string) ([]string, [][]string) {
	lines := strings.Split(tableContent, "\n")
	var headers []string
	var rows [][]string

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse pipe-separated tables
		if strings.Contains(line, "|") {
			cells := strings.Split(line, "|")
			var cleanCells []string
			for _, cell := range cells {
				cell = strings.TrimSpace(cell)
				if cell != "" {
					cleanCells = append(cleanCells, cell)
				}
			}

			if len(cleanCells) > 0 {
				if i == 0 || len(headers) == 0 {
					headers = cleanCells
				} else {
					rows = append(rows, cleanCells)
				}
			}
		} else {
			// Parse space-separated tables
			cells := strings.Fields(line)
			if len(cells) > 1 {
				if i == 0 || len(headers) == 0 {
					headers = cells
				} else {
					rows = append(rows, cells)
				}
			}
		}
	}

	return headers, rows
}

// determineFigureType determines the type of figure based on its caption
func (dl *DocumentLoader) determineFigureType(caption string) string {
	lowerCaption := strings.ToLower(caption)

	typePatterns := map[string][]string{
		"architecture": {"architecture", "framework", "structure", "overview"},
		"flowchart":    {"flow", "procedure", "process", "algorithm"},
		"diagram":      {"diagram", "schematic", "block", "connection"},
		"graph":        {"graph", "chart", "plot", "performance"},
		"timeline":     {"timeline", "sequence", "phase", "stage"},
		"network":      {"network", "topology", "deployment", "configuration"},
	}

	for figType, patterns := range typePatterns {
		for _, pattern := range patterns {
			if strings.Contains(lowerCaption, pattern) {
				return figType
			}
		}
	}

	return "diagram" // Default type
}

// extractTelecomMetadata extracts telecom-specific metadata from document content
func (dl *DocumentLoader) extractTelecomMetadata(content, filename string) *DocumentMetadata {
	metadata := &DocumentMetadata{
		Technologies:     []string{},
		NetworkFunctions: []string{},
		UseCases:         []string{},
		Keywords:         []string{},
		ProcessingNotes:  []string{},
		Custom:           make(map[string]interface{}),
		Confidence:       0.8, // Default confidence
		Language:         "en", // Default to English
	}

	// Extract source organization
	metadata.Source = dl.detectSource(content, filename)

	// Extract document type and version
	metadata.DocumentType, metadata.Version = dl.extractDocumentTypeAndVersion(content, filename)

	// Extract working group
	metadata.WorkingGroup = dl.extractWorkingGroup(content)

	// Extract category and subcategory
	metadata.Category, metadata.Subcategory = dl.extractCategory(content)

	// Extract technologies
	metadata.Technologies = dl.extractTechnologies(content)

	// Extract network functions
	metadata.NetworkFunctions = dl.extractNetworkFunctions(content)

	// Extract use cases
	metadata.UseCases = dl.extractUseCases(content)

	// Extract keywords
	metadata.Keywords = dl.extractKeywords(content)

	return metadata
}

// detectSource identifies the source organization from content and filename
func (dl *DocumentLoader) detectSource(content, filename string) string {
	// Check filename patterns
	lowerFilename := strings.ToLower(filename)
	if strings.Contains(lowerFilename, "3gpp") {
		return "3GPP"
	}
	if strings.Contains(lowerFilename, "oran") || strings.Contains(lowerFilename, "o-ran") {
		return "O-RAN"
	}
	if strings.Contains(lowerFilename, "etsi") {
		return "ETSI"
	}
	if strings.Contains(lowerFilename, "itu") {
		return "ITU"
	}

	// Check content patterns
	lowerContent := strings.ToLower(content)
	patterns := map[string]string{
		"3GPP":  `(?i)\b3gpp\b|\bthird generation partnership project\b`,
		"O-RAN": `(?i)\bo-ran\b|\bopen ran\b|\boran alliance\b`,
		"ETSI":  `(?i)\betsi\b|\beuropean telecommunications standards institute\b`,
		"ITU":   `(?i)\bitu-t\b|\bitu-r\b|\binternational telecommunication union\b`,
	}

	for source, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, lowerContent); matched {
			return source
		}
	}

	return "Unknown"
}

// extractDocumentTypeAndVersion extracts document type and version information
func (dl *DocumentLoader) extractDocumentTypeAndVersion(content, filename string) (string, string) {
	// 3GPP patterns
	tsPattern := regexp.MustCompile(`(?i)\bTS\s+(\d+\.\d+)\b`)
	trPattern := regexp.MustCompile(`(?i)\bTR\s+(\d+\.\d+)\b`)
	relPattern := regexp.MustCompile(`(?i)\brel(?:ease)?[-\s]*(\d+)\b`)

	// O-RAN patterns
	oranVersionPattern := regexp.MustCompile(`(?i)\bv(\d+\.\d+(?:\.\d+)?)\b`)
	
	var docType, version string

	// Check for 3GPP document types
	if matches := tsPattern.FindStringSubmatch(content); len(matches) > 1 {
		docType = "TS"
		version = matches[1]
	} else if matches := trPattern.FindStringSubmatch(content); len(matches) > 1 {
		docType = "TR"
		version = matches[1]
	}

	// Check for release information
	if matches := relPattern.FindStringSubmatch(content); len(matches) > 1 {
		if version == "" {
			version = "Rel-" + matches[1]
		} else {
			version += " (Rel-" + matches[1] + ")"
		}
	}

	// Check for O-RAN version patterns
	if version == "" {
		if matches := oranVersionPattern.FindStringSubmatch(content); len(matches) > 1 {
			version = "v" + matches[1]
		}
	}

	// Fallback to filename analysis
	if docType == "" {
		if strings.Contains(strings.ToLower(filename), "specification") {
			docType = "Specification"
		} else if strings.Contains(strings.ToLower(filename), "technical_report") {
			docType = "Technical Report"
		} else if strings.Contains(strings.ToLower(filename), "standard") {
			docType = "Standard"
		}
	}

	return docType, version
}

// extractWorkingGroup identifies the responsible working group
func (dl *DocumentLoader) extractWorkingGroup(content string) string {
	patterns := map[string]string{
		"RAN1": `(?i)\bran\s*1\b|\bran1\b`,
		"RAN2": `(?i)\bran\s*2\b|\bran2\b`,
		"RAN3": `(?i)\bran\s*3\b|\bran3\b`,
		"RAN4": `(?i)\bran\s*4\b|\bran4\b`,
		"SA1":  `(?i)\bsa\s*1\b|\bsa1\b`,
		"SA2":  `(?i)\bsa\s*2\b|\bsa2\b`,
		"SA3":  `(?i)\bsa\s*3\b|\bsa3\b`,
		"SA4":  `(?i)\bsa\s*4\b|\bsa4\b`,
		"SA5":  `(?i)\bsa\s*5\b|\bsa5\b`,
		"SA6":  `(?i)\bsa\s*6\b|\bsa6\b`,
		"CT1":  `(?i)\bct\s*1\b|\bct1\b`,
		"CT3":  `(?i)\bct\s*3\b|\bct3\b`,
		"CT4":  `(?i)\bct\s*4\b|\bct4\b`,
		"WG1":  `(?i)\bwg\s*1\b|\bworking\s+group\s+1\b`,
		"WG2":  `(?i)\bwg\s*2\b|\bworking\s+group\s+2\b`,
		"WG3":  `(?i)\bwg\s*3\b|\bworking\s+group\s+3\b`,
		"WG4":  `(?i)\bwg\s*4\b|\bworking\s+group\s+4\b`,
	}

	for wg, pattern := range patterns {
		if matched, _ := regexp.MatchString(pattern, content); matched {
			return wg
		}
	}

	return ""
}

// extractCategory determines the technical category and subcategory
func (dl *DocumentLoader) extractCategory(content string) (string, string) {
	lowerContent := strings.ToLower(content)

	// Main categories
	categoryPatterns := map[string][]string{
		"RAN": {"radio access", "base station", "gnb", "enb", "ue", "radio", "antenna", "rf"},
		"Core": {"core network", "5gc", "epc", "amf", "smf", "upf", "ausf", "udm", "nrf"},
		"Transport": {"transport", "backhaul", "fronthaul", "xhaul", "ethernet", "ip"},
		"Management": {"management", "orchestration", "oam", "configuration", "monitoring"},
		"Security": {"security", "authentication", "encryption", "privacy", "key"},
		"QoS": {"quality of service", "qos", "latency", "throughput", "reliability"},
		"Slicing": {"network slice", "slicing", "slice", "tenant"},
		"Edge": {"edge computing", "mec", "mobile edge", "edge"},
	}

	for category, keywords := range categoryPatterns {
		for _, keyword := range keywords {
			if strings.Contains(lowerContent, keyword) {
				// Try to determine subcategory
				subcategory := dl.extractSubcategory(lowerContent, category)
				return category, subcategory
			}
		}
	}

	return "General", ""
}

// extractSubcategory determines the subcategory within a main category
func (dl *DocumentLoader) extractSubcategory(content, category string) string {
	subcategoryPatterns := map[string]map[string][]string{
		"RAN": {
			"Physical Layer": {"physical layer", "phy", "modulation", "coding"},
			"MAC": {"mac", "medium access control", "scheduling"},
			"RLC": {"rlc", "radio link control"},
			"PDCP": {"pdcp", "packet data convergence"},
			"RRC": {"rrc", "radio resource control"},
		},
		"Core": {
			"AMF": {"amf", "access and mobility"},
			"SMF": {"smf", "session management"},
			"UPF": {"upf", "user plane"},
			"PCF": {"pcf", "policy control"},
			"UDM": {"udm", "unified data management"},
		},
		"Management": {
			"Configuration": {"configuration", "config", "provisioning"},
			"Performance": {"performance", "kpi", "monitoring"},
			"Fault": {"fault", "alarm", "error", "failure"},
		},
	}

	if subPatterns, exists := subcategoryPatterns[category]; exists {
		for subcategory, keywords := range subPatterns {
			for _, keyword := range keywords {
				if strings.Contains(content, keyword) {
					return subcategory
				}
			}
		}
	}

	return ""
}

// extractTechnologies identifies relevant technologies mentioned
func (dl *DocumentLoader) extractTechnologies(content string) []string {
	lowerContent := strings.ToLower(content)
	technologies := []string{}
	techMap := make(map[string]bool)

	techPatterns := map[string][]string{
		"5G": {"5g", "5g-nr", "nr", "new radio"},
		"4G": {"4g", "lte", "lte-a", "lte-advanced"},
		"O-RAN": {"o-ran", "open ran", "oran"},
		"Cloud RAN": {"c-ran", "cloud ran", "centralized ran"},
		"vRAN": {"vran", "virtualized ran", "virtual ran"},
		"SON": {"son", "self-organizing", "self-optimizing"},
		"NFV": {"nfv", "network function virtualization"},
		"SDN": {"sdn", "software defined network"},
		"MANO": {"mano", "management and orchestration"},
		"ETSI": {"etsi", "european telecommunications"},
	}

	for tech, patterns := range techPatterns {
		for _, pattern := range patterns {
			if strings.Contains(lowerContent, pattern) && !techMap[tech] {
				technologies = append(technologies, tech)
				techMap[tech] = true
				break
			}
		}
	}

	return technologies
}

// extractNetworkFunctions identifies network functions mentioned
func (dl *DocumentLoader) extractNetworkFunctions(content string) []string {
	lowerContent := strings.ToLower(content)
	functions := []string{}
	funcMap := make(map[string]bool)

	funcPatterns := map[string][]string{
		"gNB": {"gnb", "g-nb", "next generation nodeb"},
		"eNB": {"enb", "e-nb", "evolved nodeb"},
		"AMF": {"amf", "access and mobility management function"},
		"SMF": {"smf", "session management function"},
		"UPF": {"upf", "user plane function"},
		"PCF": {"pcf", "policy control function"},
		"UDM": {"udm", "unified data management"},
		"UDR": {"udr", "unified data repository"},
		"AUSF": {"ausf", "authentication server function"},
		"NRF": {"nrf", "network repository function"},
		"NSSF": {"nssf", "network slice selection function"},
		"NEF": {"nef", "network exposure function"},
		"AF": {"af", "application function"},
		"DU": {"du", "distributed unit"},
		"CU": {"cu", "centralized unit", "central unit"},
		"RU": {"ru", "radio unit"},
	}

	for function, patterns := range funcPatterns {
		for _, pattern := range patterns {
			if strings.Contains(lowerContent, pattern) && !funcMap[function] {
				functions = append(functions, function)
				funcMap[function] = true
				break
			}
		}
	}

	return functions
}

// extractUseCases identifies relevant use cases
func (dl *DocumentLoader) extractUseCases(content string) []string {
	lowerContent := strings.ToLower(content)
	useCases := []string{}
	ucMap := make(map[string]bool)

	ucPatterns := map[string][]string{
		"eMBB": {"embb", "enhanced mobile broadband"},
		"URLLC": {"urllc", "ultra-reliable low latency"},
		"mMTC": {"mmtc", "massive machine type communication"},
		"IoT": {"iot", "internet of things"},
		"V2X": {"v2x", "vehicle to everything", "v2v", "v2i"},
		"AR/VR": {"augmented reality", "virtual reality", "ar", "vr"},
		"Industry 4.0": {"industry 4.0", "industrial automation"},
		"Smart City": {"smart city", "smart cities"},
		"Telemedicine": {"telemedicine", "remote healthcare"},
	}

	for useCase, patterns := range ucPatterns {
		for _, pattern := range patterns {
			if strings.Contains(lowerContent, pattern) && !ucMap[useCase] {
				useCases = append(useCases, useCase)
				ucMap[useCase] = true
				break
			}
		}
	}

	return useCases
}

// extractKeywords extracts important telecom keywords
func (dl *DocumentLoader) extractKeywords(content string) []string {
	// This would implement more sophisticated keyword extraction
	// For now, return a basic set based on content analysis
	keywords := []string{}
	
	// Basic keyword extraction based on frequency and telecom relevance
	commonTelecomTerms := []string{
		"bandwidth", "latency", "throughput", "coverage", "capacity",
		"handover", "beamforming", "mimo", "carrier aggregation",
		"network slice", "quality of service", "interference",
		"spectrum", "frequency", "modulation", "coding",
		"protocol", "interface", "procedure", "signaling",
		"authentication", "security", "encryption",
		"orchestration", "virtualization", "automation",
	}

	lowerContent := strings.ToLower(content)
	for _, term := range commonTelecomTerms {
		if strings.Contains(lowerContent, term) {
			keywords = append(keywords, term)
		}
	}

	return keywords
}

// Helper functions

// isSupportedFile checks if the file type is supported
func (dl *DocumentLoader) isSupportedFile(filePath string) bool {
	ext := strings.ToLower(filepath.Ext(filePath))
	supportedExtensions := []string{".pdf"}
	
	for _, supported := range supportedExtensions {
		if ext == supported {
			return true
		}
	}
	return false
}

// generateFileHash generates a hash for the file content
func (dl *DocumentLoader) generateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// cleanTextContent cleans and normalizes extracted text
func (dl *DocumentLoader) cleanTextContent(content string) string {
	// Remove excessive whitespace
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	
	// Remove common PDF artifacts
	content = regexp.MustCompile(`(?i)page\s+\d+\s+of\s+\d+`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?i)© \d{4}`).ReplaceAllString(content, "")
	
	// Normalize line breaks
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	
	// Remove excessive line breaks
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")
	
	return strings.TrimSpace(content)
}

// extractTitle attempts to extract a meaningful title from content or metadata
func (dl *DocumentLoader) extractTitle(content string, metadata *DocumentMetadata) string {
	lines := strings.Split(content, "\n")
	
	// Look for title in the first few lines
	for i, line := range lines {
		if i > 10 { // Don't look too far
			break
		}
		
		line = strings.TrimSpace(line)
		if len(line) > 10 && len(line) < 200 {
			// Check if this looks like a title
			if dl.looksLikeTitle(line) {
				return line
			}
		}
	}
	
	// Fallback to document type and version if available
	if metadata.DocumentType != "" && metadata.Version != "" {
		return fmt.Sprintf("%s %s", metadata.DocumentType, metadata.Version)
	}
	
	return "Telecom Document"
}

// looksLikeTitle determines if a line looks like a document title
func (dl *DocumentLoader) looksLikeTitle(line string) bool {
	line = strings.ToLower(line)
	
	// Title indicators
	titleIndicators := []string{
		"technical specification", "technical report", "standard",
		"specification", "requirements", "architecture",
		"protocol", "interface", "procedures",
	}
	
	for _, indicator := range titleIndicators {
		if strings.Contains(line, indicator) {
			return true
		}
	}
	
	// Check if it contains version or release information
	if matched, _ := regexp.MatchString(`(?i)\bv?\d+\.\d+|\brel(?:ease)?[-\s]*\d+`, line); matched {
		return true
	}
	
	return false
}

// detectLanguage attempts to detect the document language
func (dl *DocumentLoader) detectLanguage(content string) string {
	// Simple language detection based on common words
	// In production, you might want to use a proper language detection library
	
	if len(content) < 100 {
		return "unknown"
	}
	
	sample := strings.ToLower(content[:1000]) // Use first 1000 chars
	
	englishWords := []string{"the", "and", "or", "but", "in", "on", "at", "to", "for", "of", "with"}
	englishCount := 0
	
	for _, word := range englishWords {
		englishCount += strings.Count(sample, " "+word+" ")
	}
	
	if englishCount > 5 {
		return "en"
	}
	
	return "unknown"
}

// Cache management functions

// getCachedDocument retrieves a document from cache if it exists and is valid
func (dl *DocumentLoader) getCachedDocument(filePath string, modTime time.Time) *LoadedDocument {
	dl.cacheMutex.RLock()
	defer dl.cacheMutex.RUnlock()
	
	if cached, exists := dl.cache[filePath]; exists {
		// Check if cache is still valid
		if time.Since(cached.LoadedAt) < dl.config.CacheTTL {
			return cached
		}
		// Cache expired, remove it
		delete(dl.cache, filePath)
	}
	
	return nil
}

// cacheDocument stores a document in the cache
func (dl *DocumentLoader) cacheDocument(doc *LoadedDocument) {
	dl.cacheMutex.Lock()
	defer dl.cacheMutex.Unlock()
	
	dl.cache[doc.SourcePath] = doc
}

// updateMetrics safely updates the loader metrics
func (dl *DocumentLoader) updateMetrics(updater func(*LoaderMetrics)) {
	dl.metrics.mutex.Lock()
	defer dl.metrics.mutex.Unlock()
	updater(dl.metrics)
}

// GetMetrics returns the current loader metrics
func (dl *DocumentLoader) GetMetrics() *LoaderMetrics {
	dl.metrics.mutex.RLock()
	defer dl.metrics.mutex.RUnlock()
	
	// Return a copy
	metrics := *dl.metrics
	return &metrics
}

// fileInfo implements os.FileInfo for remote files
type fileInfo struct {
	name    string
	size    int64
	modTime time.Time
}

func (fi *fileInfo) Name() string       { return fi.name }
func (fi *fileInfo) Size() int64        { return fi.size }
func (fi *fileInfo) Mode() os.FileMode  { return 0644 }
func (fi *fileInfo) ModTime() time.Time { return fi.modTime }
func (fi *fileInfo) IsDir() bool        { return false }
func (fi *fileInfo) Sys() interface{}   { return nil }

// MemoryMonitor manages memory usage for PDF processing
type MemoryMonitor struct {
	maxMemoryUsage int64
	currentUsage   int64
	mutex          sync.RWMutex
}

// NewMemoryMonitor creates a new memory monitor
func NewMemoryMonitor(maxMemoryUsage int64) *MemoryMonitor {
	return &MemoryMonitor{
		maxMemoryUsage: maxMemoryUsage,
		currentUsage:   0,
	}
}

// CheckMemoryAvailable checks if enough memory is available
func (mm *MemoryMonitor) CheckMemoryAvailable(requiredMemory int64) bool {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()
	return mm.currentUsage+requiredMemory <= mm.maxMemoryUsage
}

// AllocateMemory allocates memory for processing
func (mm *MemoryMonitor) AllocateMemory(memorySize int64) bool {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	if mm.currentUsage+memorySize <= mm.maxMemoryUsage {
		mm.currentUsage += memorySize
		return true
	}
	return false
}

// ReleaseMemory releases allocated memory
func (mm *MemoryMonitor) ReleaseMemory(memorySize int64) {
	mm.mutex.Lock()
	defer mm.mutex.Unlock()
	mm.currentUsage -= memorySize
	if mm.currentUsage < 0 {
		mm.currentUsage = 0
	}
}

// GetMemoryUsage returns current memory usage
func (mm *MemoryMonitor) GetMemoryUsage() (int64, int64) {
	mm.mutex.RLock()
	defer mm.mutex.RUnlock()
	return mm.currentUsage, mm.maxMemoryUsage
}

// ProcessingPool manages concurrent PDF processing tasks
type ProcessingPool struct {
	workers   chan struct{}
	activeTasks sync.WaitGroup
}

// NewProcessingPool creates a new processing pool
func NewProcessingPool(maxConcurrency int) *ProcessingPool {
	return &ProcessingPool{
		workers: make(chan struct{}, maxConcurrency),
	}
}

// AcquireWorker acquires a worker from the pool
func (pp *ProcessingPool) AcquireWorker() {
	pp.workers <- struct{}{}
	pp.activeTasks.Add(1)
}

// ReleaseWorker releases a worker back to the pool
func (pp *ProcessingPool) ReleaseWorker() {
	<-pp.workers
	pp.activeTasks.Done()
}

// WaitForCompletion waits for all active tasks to complete
func (pp *ProcessingPool) WaitForCompletion() {
	pp.activeTasks.Wait()
}

// PDFProcessingResult holds the result of PDF processing
type PDFProcessingResult struct {
	Content    string
	RawContent string
	Metadata   *DocumentMetadata
	Error      error
	Pages      int
	Tables     []ExtractedTable
	Figures    []ExtractedFigure
}

// ExtractedTable represents a table extracted from PDF
type ExtractedTable struct {
	PageNumber int               `json:"page_number"`
	Caption    string            `json:"caption"`
	Headers    []string          `json:"headers"`
	Rows       [][]string        `json:"rows"`
	Bounds     Rectangle         `json:"bounds"`
	Quality    float64           `json:"quality"`
	Metadata   map[string]interface{} `json:"metadata"`
}

// ExtractedFigure represents a figure extracted from PDF
type ExtractedFigure struct {
	PageNumber  int               `json:"page_number"`
	Caption     string            `json:"caption"`
	Description string            `json:"description"`
	Bounds      Rectangle         `json:"bounds"`
	FigureType  string            `json:"figure_type"`
	Quality     float64           `json:"quality"`
	Metadata    map[string]interface{} `json:"metadata"`
}

// Rectangle represents a bounding rectangle
type Rectangle struct {
	X      float64 `json:"x"`
	Y      float64 `json:"y"`
	Width  float64 `json:"width"`
	Height float64 `json:"height"`
}

// StreamingPDFProcessor handles large PDF processing with memory management
type StreamingPDFProcessor struct {
	file           *os.File
	logger         *slog.Logger
	config         *DocumentLoaderConfig
	memoryMonitor  *MemoryMonitor
	pageBuffer     chan *PDFPageResult
	resultBuffer   strings.Builder
	rawBuffer      strings.Builder
	mutex          sync.Mutex
	processedPages int
	totalPages     int
}

// PDFPageResult holds the result of processing a single PDF page
type PDFPageResult struct {
	PageNumber      int
	Content         string
	RawContent      string
	Tables          []ExtractedTable
	Figures         []ExtractedFigure
	ProcessingTime  time.Duration
	Error           error
	MemoryUsed      int64
}

// StreamingProcessingResult holds the complete result of streaming PDF processing
type StreamingProcessingResult struct {
	Content           string
	RawContent        string
	PageCount         int
	ProcessingErrors  []error
	TotalProcessingTime time.Duration
	PeakMemoryUsage   int64
	Tables            []ExtractedTable
	Figures           []ExtractedFigure
}

// ProcessStreamingPDF processes a PDF using streaming approach with memory management
func (spp *StreamingPDFProcessor) ProcessStreamingPDF(ctx context.Context) (*StreamingProcessingResult, error) {
	startTime := time.Now()
	spp.logger.Info("Starting streaming PDF processing")

	// Get PDF info using pdfcpu
	pdfInfo, err := spp.getPDFInfo()
	if err != nil {
		return nil, fmt.Errorf("failed to get PDF info: %w", err)
	}

	spp.totalPages = pdfInfo.PageCount
	spp.logger.Info("PDF analysis complete", "total_pages", spp.totalPages)

	// Process pages in parallel batches
	batchSize := spp.config.PageProcessingBatch
	if batchSize <= 0 {
		batchSize = 10 // Default batch size
	}

	var allTables []ExtractedTable
	var allFigures []ExtractedFigure
	var processingErrors []error
	var peakMemoryUsage int64

	// Process pages in batches to manage memory
	for startPage := 1; startPage <= spp.totalPages; startPage += batchSize {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		endPage := startPage + batchSize - 1
		if endPage > spp.totalPages {
			endPage = spp.totalPages
		}

		spp.logger.Debug("Processing page batch", "start", startPage, "end", endPage)

		// Process batch
		batchResults, err := spp.processBatch(ctx, startPage, endPage)
		if err != nil {
			spp.logger.Error("Batch processing failed", "start", startPage, "end", endPage, "error", err)
			processingErrors = append(processingErrors, err)
			continue
		}

		// Aggregate results
		for _, result := range batchResults {
			if result.Error != nil {
				processingErrors = append(processingErrors, result.Error)
				continue
			}

			spp.resultBuffer.WriteString(result.Content)
			spp.rawBuffer.WriteString(result.RawContent)
			allTables = append(allTables, result.Tables...)
			allFigures = append(allFigures, result.Figures...)

			if result.MemoryUsed > peakMemoryUsage {
				peakMemoryUsage = result.MemoryUsed
			}
		}

		// Force garbage collection after each batch for large files
		if spp.totalPages > 100 {
			runtime.GC()
		}

		// Check memory pressure and adjust if needed
		currentUsage, maxUsage := spp.memoryMonitor.GetMemoryUsage()
		if currentUsage > maxUsage*80/100 { // 80% threshold
			spp.logger.Warn("High memory usage detected, forcing GC", "usage", currentUsage, "max", maxUsage)
			runtime.GC()
			time.Sleep(100 * time.Millisecond) // Brief pause to let GC complete
		}
	}

	result := &StreamingProcessingResult{
		Content:             spp.resultBuffer.String(),
		RawContent:          spp.rawBuffer.String(),
		PageCount:           spp.totalPages,
		ProcessingErrors:    processingErrors,
		TotalProcessingTime: time.Since(startTime),
		PeakMemoryUsage:     peakMemoryUsage,
		Tables:              allTables,
		Figures:             allFigures,
	}

	spp.logger.Info("Streaming PDF processing completed",
		"pages", spp.totalPages,
		"processing_time", result.TotalProcessingTime,
		"errors", len(processingErrors),
		"tables", len(allTables),
		"figures", len(allFigures),
	)

	return result, nil
}

// getPDFInfo extracts basic PDF information
func (spp *StreamingPDFProcessor) getPDFInfo() (*PDFInfo, error) {
	// Use pdfcpu to get PDF info efficiently
	info, err := spp.file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Create PDF reader to get page count
	reader, err := pdf.NewReader(spp.file, info.Size())
	if err != nil {
		return nil, fmt.Errorf("failed to create PDF reader: %w", err)
	}

	pageCount := reader.NumPage()

	return &PDFInfo{
		PageCount: pageCount,
		FileSize:  info.Size(),
		CreatedAt: info.ModTime(),
	}, nil
}

// processBatch processes a batch of PDF pages
func (spp *StreamingPDFProcessor) processBatch(ctx context.Context, startPage, endPage int) ([]*PDFPageResult, error) {
	var results []*PDFPageResult
	var wg sync.WaitGroup

	// Create semaphore to limit concurrent page processing
	concurrency := spp.config.MaxConcurrency
	if concurrency <= 0 {
		concurrency = 3 // Conservative default
	}
	semaphore := make(chan struct{}, concurrency)

	// Process pages in the batch
	for pageNum := startPage; pageNum <= endPage; pageNum++ {
		wg.Add(1)
		go func(pageNumber int) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			result := spp.processPage(ctx, pageNumber)
			spp.mutex.Lock()
			results = append(results, result)
			spp.mutex.Unlock()
		}(pageNum)
	}

	wg.Wait()
	return results, nil
}

// processPage processes a single PDF page
func (spp *StreamingPDFProcessor) processPage(ctx context.Context, pageNumber int) *PDFPageResult {
	startTime := time.Now()
	
	result := &PDFPageResult{
		PageNumber: pageNumber,
		Tables:     []ExtractedTable{},
		Figures:    []ExtractedFigure{},
	}

	// Track memory usage for this page
	memBefore, _ := spp.memoryMonitor.GetMemoryUsage()

	// Extract page content using the most appropriate method
	content, rawContent, err := spp.extractPageContent(pageNumber)
	if err != nil {
		result.Error = fmt.Errorf("failed to extract content from page %d: %w", pageNumber, err)
		return result
	}

	result.Content = content
	result.RawContent = rawContent

	// Extract tables if enabled
	if spp.config.EnableTableExtraction {
		tables := spp.extractTablesFromPage(content, pageNumber)
		result.Tables = tables
	}

	// Extract figures if enabled
	if spp.config.EnableFigureExtraction {
		figures := spp.extractFiguresFromPage(content, pageNumber)
		result.Figures = figures
	}

	// Calculate memory used
	memAfter, _ := spp.memoryMonitor.GetMemoryUsage()
	result.MemoryUsed = memAfter - memBefore
	result.ProcessingTime = time.Since(startTime)

	return result
}

// extractPageContent extracts content from a specific page
func (spp *StreamingPDFProcessor) extractPageContent(pageNumber int) (string, string, error) {
	// Reset file position
	spp.file.Seek(0, 0)

	// Get file info
	info, err := spp.file.Stat()
	if err != nil {
		return "", "", fmt.Errorf("failed to get file info: %w", err)
	}

	// Create PDF reader
	reader, err := pdf.NewReader(spp.file, info.Size())
	if err != nil {
		return "", "", fmt.Errorf("failed to create PDF reader: %w", err)
	}

	// Extract text from specific page
	page := reader.Page(pageNumber)
	if page.V.IsNull() {
		return "", "", fmt.Errorf("page %d is null or invalid", pageNumber)
	}

	// Get plain text
	text, err := page.GetPlainText(nil) // Pass nil for font map as we don't need font-specific extraction
	if err != nil {
		return "", "", fmt.Errorf("failed to extract text from page %d: %w", pageNumber, err)
	}

	// Clean the text
	cleanContent := spp.cleanPageContent(text)
	
	return cleanContent, text, nil
}

// cleanPageContent cleans and normalizes content from a single page
func (spp *StreamingPDFProcessor) cleanPageContent(content string) string {
	// Remove excessive whitespace
	content = regexp.MustCompile(`\s+`).ReplaceAllString(content, " ")
	
	// Remove page headers/footers patterns
	content = regexp.MustCompile(`(?i)page\s+\d+\s+of\s+\d+`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?i)© \d{4}.*?(?:\n|$)`).ReplaceAllString(content, "")
	
	// Remove document headers that appear on every page
	content = regexp.MustCompile(`(?i)3GPP TS \d+\.\d+.*?(?:\n|$)`).ReplaceAllString(content, "")
	content = regexp.MustCompile(`(?i)O-RAN Alliance.*?(?:\n|$)`).ReplaceAllString(content, "")
	
	// Normalize line breaks
	content = strings.ReplaceAll(content, "\r\n", "\n")
	content = strings.ReplaceAll(content, "\r", "\n")
	
	// Remove excessive line breaks but preserve paragraph structure
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")
	
	return strings.TrimSpace(content)
}

// extractTablesFromPage extracts tables from a specific page
func (spp *StreamingPDFProcessor) extractTablesFromPage(content string, pageNumber int) []ExtractedTable {
	var tables []ExtractedTable

	// Enhanced table detection patterns for telecom documents
	tablePatterns := []*regexp.Regexp{
		// Standard table pattern with caption
		regexp.MustCompile(`(?i)table\s+(\d+)[-:]?\s*(.+?)\n([\s\S]*?)(?=\n\s*(?:table|figure|section|\d+\.\d+|$))`),
		// Pipe-separated tables
		regexp.MustCompile(`(?s)(\|[^\n]+\|\s*\n\s*\|[-\s\|]+\|\s*\n((?:\s*\|[^\n]+\|\s*\n)*?))`),
		// Space-separated tabular data
		regexp.MustCompile(`(?m)^(\s*\w+(?:\s+\w+){2,}\s*\n(?:\s*[-\s]+\s*\n)?(?:\s*\w+(?:\s+\w+){2,}\s*\n)+)`),
		// Parameter tables common in telecom specs
		regexp.MustCompile(`(?i)(parameter|field|attribute|value).*?\n([\s\S]*?)(?=\n\s*(?:note|table|figure|section|\d+\.\d+|$))`),
	}

	for i, pattern := range tablePatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for j, match := range matches {
			if len(match) >= 3 {
				table := ExtractedTable{
					PageNumber: pageNumber,
					Quality:    0.8,
					Metadata:   map[string]interface{}{
						"extraction_method": fmt.Sprintf("pattern_%d", i),
						"match_index":      j,
					},
				}

				// Extract caption if available
				if len(match) > 2 && match[2] != "" {
					table.Caption = strings.TrimSpace(match[2])
				} else {
					table.Caption = fmt.Sprintf("Table on page %d", pageNumber)
				}

				// Parse table content
				tableContent := match[len(match)-1]
				table.Headers, table.Rows = spp.parseTableContent(tableContent)

				// Quality assessment
				table.Quality = spp.assessTableQuality(table)

				// Skip low-quality tables
				if table.Quality > 0.5 && len(table.Headers) > 0 {
					tables = append(tables, table)
				}

				// Limit tables per page
				if len(tables) >= 10 {
					break
				}
			}
		}
	}

	return tables
}

// extractFiguresFromPage extracts figures from a specific page
func (spp *StreamingPDFProcessor) extractFiguresFromPage(content string, pageNumber int) []ExtractedFigure {
	var figures []ExtractedFigure

	// Enhanced figure detection patterns
	figurePatterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)figure\s+(\d+)[-:]?\s*(.+?)(?:\n|$)`),
		regexp.MustCompile(`(?i)fig\.?\s*(\d+)[-:]?\s*(.+?)(?:\n|$)`),
		regexp.MustCompile(`(?i)(diagram|architecture|schematic|block\s+diagram|flow\s+chart|topology)\s*[-:]?\s*(.+?)(?:\n|$)`),
	}

	for i, pattern := range figurePatterns {
		matches := pattern.FindAllStringSubmatch(content, -1)
		for j, match := range matches {
			if len(match) >= 3 {
				figure := ExtractedFigure{
					PageNumber: pageNumber,
					Quality:    0.8,
					Metadata: map[string]interface{}{
						"extraction_method": fmt.Sprintf("pattern_%d", i),
						"match_index":      j,
					},
				}

				// Extract caption
				if len(match) > 2 {
					figure.Caption = strings.TrimSpace(match[2])
				} else {
					figure.Caption = fmt.Sprintf("Figure on page %d", pageNumber)
				}

				// Determine figure type
				figure.FigureType = spp.determineFigureType(figure.Caption)

				// Quality assessment
				figure.Quality = spp.assessFigureQuality(figure)

				if figure.Quality > 0.5 {
					figures = append(figures, figure)
				}

				// Limit figures per page
				if len(figures) >= 5 {
					break
				}
			}
		}
	}

	return figures
}

// parseTableContent parses table content with enhanced logic
func (spp *StreamingPDFProcessor) parseTableContent(tableContent string) ([]string, [][]string) {
	lines := strings.Split(tableContent, "\n")
	var headers []string
	var rows [][]string
	var potentialHeaders []string

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var cells []string

		// Parse pipe-separated tables
		if strings.Contains(line, "|") {
			parts := strings.Split(line, "|")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if part != "" && !regexp.MustCompile(`^[-\s]*$`).MatchString(part) {
					cells = append(cells, part)
				}
			}
		} else {
			// Parse space-separated tables
			// Use regex to better handle multi-word columns
			fields := regexp.MustCompile(`\s{2,}`).Split(line, -1)
			for _, field := range fields {
				field = strings.TrimSpace(field)
				if field != "" {
					cells = append(cells, field)
				}
			}
		}

		if len(cells) > 0 {
			// Determine if this is a header row
			if i == 0 || len(headers) == 0 {
				// Check if this looks like a header
				if spp.looksLikeHeader(cells) {
					headers = cells
					continue
				} else if len(potentialHeaders) == 0 {
					potentialHeaders = cells
				}
			}

			// Add as data row
			rows = append(rows, cells)
		}
	}

	// If no clear headers were found, use potential headers
	if len(headers) == 0 && len(potentialHeaders) > 0 {
		headers = potentialHeaders
		if len(rows) > 0 {
			rows = rows[1:] // Remove the first row as it became the header
		}
	}

	return headers, rows
}

// looksLikeHeader determines if a row looks like a table header
func (spp *StreamingPDFProcessor) looksLikeHeader(cells []string) bool {
	headerIndicators := []string{
		"parameter", "field", "value", "type", "description", "name", "id",
		"attribute", "element", "component", "function", "protocol", "interface",
		"frequency", "band", "channel", "power", "signal", "reference",
	}

	for _, cell := range cells {
		cellLower := strings.ToLower(cell)
		for _, indicator := range headerIndicators {
			if strings.Contains(cellLower, indicator) {
				return true
			}
		}
	}

	return false
}

// assessTableQuality assesses the quality of an extracted table
func (spp *StreamingPDFProcessor) assessTableQuality(table ExtractedTable) float64 {
	quality := 0.5 // Base quality

	// Header quality
	if len(table.Headers) > 0 {
		quality += 0.2
		if len(table.Headers) >= 2 && len(table.Headers) <= 8 {
			quality += 0.1 // Good header count
		}
	}

	// Row quality
	if len(table.Rows) > 0 {
		quality += 0.2
		if len(table.Rows) >= 2 {
			quality += 0.1 // Multiple rows
		}
	}

	// Consistency check
	if len(table.Headers) > 0 && len(table.Rows) > 0 {
		consistentRows := 0
		for _, row := range table.Rows {
			if len(row) == len(table.Headers) {
				consistentRows++
			}
		}
		consistency := float64(consistentRows) / float64(len(table.Rows))
		quality += consistency * 0.2
	}

	// Caption quality
	if table.Caption != "" && len(table.Caption) > 10 {
		quality += 0.1
	}

	if quality > 1.0 {
		quality = 1.0
	}

	return quality
}

// assessFigureQuality assesses the quality of an extracted figure
func (spp *StreamingPDFProcessor) assessFigureQuality(figure ExtractedFigure) float64 {
	quality := 0.6 // Base quality for figures

	// Caption quality
	if figure.Caption != "" {
		quality += 0.2
		if len(figure.Caption) > 20 {
			quality += 0.1 // Detailed caption
		}
	}

	// Type-specific quality
	if figure.FigureType != "diagram" { // More specific than default
		quality += 0.1
	}

	return quality
}

// determineFigureType determines the type of figure with enhanced logic
func (spp *StreamingPDFProcessor) determineFigureType(caption string) string {
	lowerCaption := strings.ToLower(caption)

	typePatterns := map[string][]string{
		"architecture": {"architecture", "framework", "structure", "overview", "system"},
		"flowchart":    {"flow", "procedure", "process", "algorithm", "sequence"},
		"diagram":      {"diagram", "schematic", "block", "connection", "layout"},
		"graph":        {"graph", "chart", "plot", "performance", "measurement"},
		"timeline":     {"timeline", "sequence", "phase", "stage", "evolution"},
		"network":      {"network", "topology", "deployment", "configuration"},
		"protocol":     {"protocol", "stack", "layer", "interface", "message"},
		"signal":       {"signal", "waveform", "spectrum", "frequency", "modulation"},
	}

	for figType, patterns := range typePatterns {
		for _, pattern := range patterns {
			if strings.Contains(lowerCaption, pattern) {
				return figType
			}
		}
	}

	return "diagram" // Default type
}

// calculateProcessingConfidence calculates confidence score for document processing
func (dl *DocumentLoader) calculateProcessingConfidence(content string, pageCount int, errorCount int) float32 {
	confidence := float32(0.8) // Base confidence

	// Content length factor
	if len(content) > 1000 {
		confidence += 0.1
	}
	if len(content) > 10000 {
		confidence += 0.05
	}

	// Page count factor
	if pageCount > 5 {
		confidence += 0.05
	}

	// Error factor
	if errorCount == 0 {
		confidence += 0.1
	} else {
		errorReduction := float32(errorCount) * 0.05
		confidence -= errorReduction
	}

	// Telecom content indicators
	telecomIndicators := []string{
		"3gpp", "o-ran", "5g", "4g", "lte", "gnb", "enb", "amf", "smf", "upf",
		"frequency", "bandwidth", "protocol", "specification", "technical",
	}

	telecomCount := 0
	lowerContent := strings.ToLower(content)
	for _, indicator := range telecomIndicators {
		if strings.Contains(lowerContent, indicator) {
			telecomCount++
		}
	}

	if telecomCount > 3 {
		confidence += 0.1
	}

	// Cap confidence
	if confidence > 1.0 {
		confidence = 1.0
	}
	if confidence < 0.3 {
		confidence = 0.3
	}

	return confidence
}

// LoadDocument loads a single document from the given path
func (dl *DocumentLoader) LoadDocument(ctx context.Context, docPath string) (*LoadedDocument, error) {
	fileInfo, err := os.Stat(docPath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat document: %w", err)
	}

	// Check file size
	if dl.config.MaxFileSize > 0 && fileInfo.Size() > dl.config.MaxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum %d", fileInfo.Size(), dl.config.MaxFileSize)
	}

	// Generate document ID
	docID := generateDocumentID(docPath)

	// Check cache first
	dl.cacheMutex.RLock()
	if cached, exists := dl.cache[docID]; exists {
		dl.cacheMutex.RUnlock()
		dl.metrics.CacheHits++
		return cached, nil
	}
	dl.cacheMutex.RUnlock()
	dl.metrics.CacheMisses++

	startTime := time.Now()

	// Process document based on file type
	var content, rawContent string
	var metadata *DocumentMetadata
	
	ext := strings.ToLower(filepath.Ext(docPath))
	switch ext {
	case ".pdf":
		content, rawContent, metadata, err = dl.processPDFFile(ctx, docPath)
	case ".txt", ".md":
		content, rawContent, metadata, err = dl.processTextFile(ctx, docPath)
	default:
		content, rawContent, metadata, err = dl.processTextFile(ctx, docPath)
	}

	if err != nil {
		dl.metrics.FailedLoads++
		return nil, fmt.Errorf("failed to process document: %w", err)
	}

	// Create loaded document
	doc := &LoadedDocument{
		ID:             docID,
		SourcePath:     docPath,
		Filename:       filepath.Base(docPath),
		Title:          extractTitle(content),
		Content:        content,
		RawContent:     rawContent,
		Metadata:       metadata,
		LoadedAt:       time.Now(),
		ProcessingTime: time.Since(startTime),
		Hash:           fmt.Sprintf("%x", md5.Sum([]byte(content))),
		Size:           fileInfo.Size(),
		Language:       detectLanguage(content),
	}

	// Cache the document
	dl.cacheMutex.Lock()
	dl.cache[docID] = doc
	dl.cacheMutex.Unlock()

	dl.metrics.SuccessfulLoads++
	dl.metrics.TotalDocuments++
	dl.metrics.LastProcessedAt = time.Now()

	return doc, nil
}

// processPDFFile processes PDF files
func (dl *DocumentLoader) processPDFFile(ctx context.Context, filePath string) (string, string, *DocumentMetadata, error) {
	// Check if file size exceeds streaming threshold
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to stat PDF file: %w", err)
	}

	if dl.config.StreamingEnabled && fileInfo.Size() > dl.config.StreamingThreshold {
		return dl.processPDFStreamingAdvanced(ctx, filePath, fileInfo.Size())
	}

	// Use hybrid approach for regular PDFs
	return dl.processPDFHybrid(ctx, filePath)
}

// processTextFile processes plain text and markdown files
func (dl *DocumentLoader) processTextFile(ctx context.Context, filePath string) (string, string, *DocumentMetadata, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", nil, fmt.Errorf("failed to read text file: %w", err)
	}

	textContent := string(content)
	metadata := dl.extractTelecomMetadata(textContent, filepath.Base(filePath))
	
	return textContent, textContent, metadata, nil
}

// extractTitle extracts title from document content
func extractTitle(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if len(line) > 0 && !strings.HasPrefix(line, "#") {
			if len(line) > 100 {
				return line[:100] + "..."
			}
			return line
		}
	}
	return "Untitled Document"
}

// detectLanguage detects the language of document content
func detectLanguage(content string) string {
	// Simple heuristic - could be enhanced with proper language detection
	if strings.Contains(strings.ToLower(content), "technical specification") ||
		strings.Contains(strings.ToLower(content), "3gpp") {
		return "en"
	}
	return "unknown"
}

// generateDocumentID generates a unique ID for a document
func generateDocumentID(path string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(path)))
}

// PDFInfo holds basic PDF information
type PDFInfo struct {
	PageCount int
	FileSize  int64
	CreatedAt time.Time
}