package rag

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
}

// DocumentLoaderConfig holds configuration for the document loader
type DocumentLoaderConfig struct {
	// Source directories and URLs
	LocalPaths       []string `json:"local_paths"`
	RemoteURLs       []string `json:"remote_urls"`
	
	// PDF processing configuration
	MaxFileSize      int64    `json:"max_file_size"`      // Maximum file size in bytes
	PDFTextExtractor string   `json:"pdf_text_extractor"` // "native" or "ocr"
	OCREnabled       bool     `json:"ocr_enabled"`
	OCRLanguage      string   `json:"ocr_language"`
	
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

	return &DocumentLoader{
		config:     config,
		logger:     slog.Default().With("component", "document-loader"),
		cache:      make(map[string]*LoadedDocument),
		metrics:    &LoaderMetrics{LastProcessedAt: time.Now()},
		httpClient: httpClient,
	}
}

// getDefaultLoaderConfig returns default configuration for the document loader
func getDefaultLoaderConfig() *DocumentLoaderConfig {
	return &DocumentLoaderConfig{
		LocalPaths:        []string{"./knowledge_base"},
		RemoteURLs:        []string{},
		MaxFileSize:       100 * 1024 * 1024, // 100MB
		PDFTextExtractor:  "native",
		OCREnabled:        false,
		OCRLanguage:       "eng",
		MinContentLength:  100,
		MaxContentLength:  1000000, // 1MB text
		LanguageFilter:    []string{"en", "eng", "english"},
		EnableCaching:     true,
		CacheDirectory:    "./cache/documents",
		CacheTTL:          24 * time.Hour,
		BatchSize:         10,
		MaxConcurrency:    5,
		ProcessingTimeout: 30 * time.Second,
		MaxRetries:        3,
		RetryDelay:        2 * time.Second,
		PreferredSources: map[string]int{
			"3GPP":  10,
			"O-RAN": 9,
			"ETSI":  8,
			"ITU":   7,
		},
		TechnicalDomains: []string{"RAN", "Core", "Transport", "Management", "O-RAN"},
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

// processPDF extracts content and metadata from a PDF file
func (dl *DocumentLoader) processPDF(ctx context.Context, filePath string) (string, string, *DocumentMetadata, error) {
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

	for i := 1; i <= pageCount; i++ {
		page := reader.Page(i)
		if page.V.IsNull() {
			continue
		}

		// Extract text from page
		text, err := page.GetPlainText()
		if err != nil {
			dl.logger.Warn("Failed to extract text from page", "page", i, "error", err)
			continue
		}

		textBuilder.WriteString(text)
		textBuilder.WriteString("\n")
		rawTextBuilder.WriteString(text)
		rawTextBuilder.WriteString("\n")
	}

	rawContent := rawTextBuilder.String()
	content := dl.cleanTextContent(textBuilder.String())

	// Extract metadata from PDF and content
	metadata := dl.extractTelecomMetadata(content, filepath.Base(filePath))
	metadata.PageCount = pageCount

	// Update metadata with additional PDF-specific information
	if reader.Trailer() != nil {
		// Try to extract PDF metadata if available
		// This would require more sophisticated PDF metadata extraction
	}

	return content, rawContent, metadata, nil
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