
package ca



import (

	"bytes"

	"context"

	"crypto"

	"crypto/sha256"

	"crypto/x509"

	"crypto/x509/pkix"

	"encoding/hex"

	"fmt"

	"io"

	"net/http"

	"sync"

	"sync/atomic"

	"time"



	"golang.org/x/crypto/ocsp"



	"github.com/nephio-project/nephoran-intent-operator/pkg/logging"

)



// RevocationStatus represents the revocation status of a certificate.

type RevocationStatus string



const (

	// RevocationStatusUnknown holds revocationstatusunknown value.

	RevocationStatusUnknown RevocationStatus = "unknown"

	// RevocationStatusValid holds revocationstatusvalid value.

	RevocationStatusValid RevocationStatus = "valid"

	// RevocationStatusRevoked holds revocationstatusrevoked value.

	RevocationStatusRevoked RevocationStatus = "revoked"

	// RevocationStatusGood holds revocationstatusgood value.

	RevocationStatusGood RevocationStatus = "good" // Alias for valid, used in OCSP responses

)



// EnhancedRevocationSystem provides comprehensive revocation checking with performance optimization.

type EnhancedRevocationSystem struct {

	config            *EnhancedRevocationConfig

	logger            *logging.StructuredLogger

	crlManager        *CRLManager

	ocspManager       *OCSPManager

	cacheManager      *RevocationCacheManager

	automaticReplacer *AutomaticCertificateReplacer

	metricsCollector  *RevocationMetricsCollector



	// Performance optimization.

	batchProcessor *BatchRevocationProcessor

	asyncValidator *AsyncRevocationValidator

	circuitBreaker *RevocationCircuitBreaker



	// Control.

	ctx    context.Context

	cancel context.CancelFunc

	wg     sync.WaitGroup

}



// EnhancedRevocationConfig configures the enhanced revocation system.

type EnhancedRevocationConfig struct {

	// CRL Configuration.

	CRLEnabled            bool          `yaml:"crl_enabled"`

	CRLDistributionPoints []string      `yaml:"crl_distribution_points"`

	CRLUpdateInterval     time.Duration `yaml:"crl_update_interval"`

	CRLCacheSize          int           `yaml:"crl_cache_size"`

	CRLMaxSize            int64         `yaml:"crl_max_size"`

	CRLDeltaEnabled       bool          `yaml:"crl_delta_enabled"`



	// OCSP Configuration.

	OCSPEnabled         bool          `yaml:"ocsp_enabled"`

	OCSPResponders      []string      `yaml:"ocsp_responders"`

	OCSPTimeout         time.Duration `yaml:"ocsp_timeout"`

	OCSPStaplingEnabled bool          `yaml:"ocsp_stapling_enabled"`

	OCSPNonceEnabled    bool          `yaml:"ocsp_nonce_enabled"`

	OCSPCacheSize       int           `yaml:"ocsp_cache_size"`

	OCSPCacheTTL        time.Duration `yaml:"ocsp_cache_ttl"`



	// Performance Settings.

	BatchProcessingEnabled bool          `yaml:"batch_processing_enabled"`

	BatchSize              int           `yaml:"batch_size"`

	BatchTimeout           time.Duration `yaml:"batch_timeout"`

	AsyncValidation        bool          `yaml:"async_validation"`

	MaxConcurrentChecks    int           `yaml:"max_concurrent_checks"`

	ConnectionPoolSize     int           `yaml:"connection_pool_size"`



	// Cache Configuration.

	CacheEnabled   bool          `yaml:"cache_enabled"`

	L1CacheSize    int           `yaml:"l1_cache_size"`

	L1CacheTTL     time.Duration `yaml:"l1_cache_ttl"`

	L2CacheEnabled bool          `yaml:"l2_cache_enabled"`

	L2CacheSize    int           `yaml:"l2_cache_size"`

	L2CacheTTL     time.Duration `yaml:"l2_cache_ttl"`

	PreloadCache   bool          `yaml:"preload_cache"`



	// Fail Policy.

	SoftFailEnabled       bool `yaml:"soft_fail_enabled"`

	HardFailEnabled       bool `yaml:"hard_fail_enabled"`

	FailureThreshold      int  `yaml:"failure_threshold"`

	CircuitBreakerEnabled bool `yaml:"circuit_breaker_enabled"`



	// Automatic Replacement.

	AutoReplaceRevoked bool          `yaml:"auto_replace_revoked"`

	ReplacementTimeout time.Duration `yaml:"replacement_timeout"`

	ReplacementRetries int           `yaml:"replacement_retries"`

}



// CRLManager manages Certificate Revocation Lists.

type CRLManager struct {

	distributionPoints map[string]*CRLDistributionPoint

	deltaProcessor     *DeltaCRLProcessor

	httpClient         *http.Client

	logger             *logging.StructuredLogger

	mu                 sync.RWMutex

}



// CRLDistributionPoint represents a CRL distribution point.

type CRLDistributionPoint struct {

	URL              string

	CRL              *pkix.CertificateList

	DeltaCRL         *pkix.CertificateList

	LastUpdate       time.Time

	NextUpdate       time.Time

	UpdateInProgress int32  // 0=idle, 1=updating (replaced atomic.Bool)

	ErrorCount       uint32 // atomic counter (replaced atomic.Uint32)

	mu               sync.RWMutex

}



// DeltaCRLProcessor processes delta CRLs for efficient updates.

type DeltaCRLProcessor struct {

	baseCRLs  map[string]*pkix.CertificateList

	deltaCRLs map[string]*pkix.CertificateList

	mu        sync.RWMutex

}



// processDelta processes a delta CRL update.

func (d *DeltaCRLProcessor) processDelta(url string, crl *pkix.CertificateList) {

	d.mu.Lock()

	defer d.mu.Unlock()

	d.baseCRLs[url] = crl

}



// OCSPManager manages OCSP operations.

type OCSPManager struct {

	responders     map[string]*OCSPResponder

	staplingCache  *StaplingCache

	nonceGenerator *NonceGenerator

	connectionPool *OCSPConnectionPool

	logger         *logging.StructuredLogger

	mu             sync.RWMutex

}



// OCSPResponder represents an OCSP responder.

type OCSPResponder struct {

	URL          string

	Client       *http.Client

	LastCheck    time.Time

	ResponseTime time.Duration

	SuccessRate  float64

	ErrorCount   uint32 // atomic counter (replaced atomic.Uint32)

	RequestCount uint64 // atomic counter (replaced atomic.Uint64)

}



// StaplingCache caches OCSP stapling responses.

type StaplingCache struct {

	responses map[string]*StapledResponse

	maxSize   int

	ttl       time.Duration

	mu        sync.RWMutex

}



// StapledResponse represents a cached OCSP stapled response.

type StapledResponse struct {

	Response   []byte

	ProducedAt time.Time

	NextUpdate time.Time

	CachedAt   time.Time

}



// NonceGenerator generates OCSP nonces for replay protection.

type NonceGenerator struct {

	nonces map[string]time.Time

	mu     sync.RWMutex

}



// RevocationCacheManager manages multi-level caching.

type RevocationCacheManager struct {

	l1Cache      *L1RevocationCache

	l2Cache      *L2RevocationCache

	preloadCache *PreloadRevocationCache

	stats        *CacheStatistics

	mu           sync.RWMutex

}



// L1RevocationCache is a fast in-memory cache.

type L1RevocationCache struct {

	entries map[string]*RevocationCacheEntry

	lru     *LRUEvictionPolicy

	maxSize int

	ttl     time.Duration

	mu      sync.RWMutex

}



// RevocationCacheEntry represents a cached revocation check result.

type RevocationCacheEntry struct {

	SerialNumber string

	Status       RevocationStatus

	CheckTime    time.Time

	NextUpdate   time.Time

	Source       string

	HitCount     uint32 // atomic counter (replaced atomic.Uint32)

	LastAccessed time.Time

}



// BatchRevocationProcessor processes revocation checks in batches.

type BatchRevocationProcessor struct {

	batchQueue   chan *RevocationBatch

	batchSize    int

	batchTimeout time.Duration

	processor    RevocationProcessor

	logger       *logging.StructuredLogger

	wg           sync.WaitGroup

}



// RevocationBatch represents a batch of revocation checks.

type RevocationBatch struct {

	ID           string

	Certificates []*x509.Certificate

	Results      map[string]*RevocationCheckResult

	CompleteChan chan struct{}

	CreatedAt    time.Time

}



// AsyncRevocationValidator performs asynchronous revocation validation.

type AsyncRevocationValidator struct {

	workers         int

	workQueue       chan *AsyncRevocationRequest

	resultCollector *ResultCollector

	logger          *logging.StructuredLogger

	wg              sync.WaitGroup

}



// AsyncRevocationRequest represents an async revocation check request.

type AsyncRevocationRequest struct {

	ID          string

	Certificate *x509.Certificate

	Priority    int

	Timeout     time.Duration

	ResultChan  chan *RevocationCheckResult

	CreatedAt   time.Time

}



// RevocationCircuitBreaker implements circuit breaker for revocation checks.

type RevocationCircuitBreaker struct {

	crlBreaker  *CircuitBreaker

	ocspBreaker *CircuitBreaker

	state       atomic.Value

	mu          sync.RWMutex

}



// CircuitBreaker represents a single circuit breaker.

type CircuitBreaker struct {

	name            string

	state           atomic.Value

	failureCount    uint32 // atomic counter (replaced atomic.Uint32)

	successCount    uint32 // atomic counter (replaced atomic.Uint32)

	lastFailureTime atomic.Value

	config          *CircuitBreakerConfig

}



// AutomaticCertificateReplacer automatically replaces revoked certificates.

type AutomaticCertificateReplacer struct {

	caManager        *CAManager

	kubeClient       interface{} // kubernetes.Interface

	replacementQueue chan *ReplacementRequest

	logger           *logging.StructuredLogger

	wg               sync.WaitGroup

}



// ReplacementRequest represents a certificate replacement request.

type ReplacementRequest struct {

	SerialNumber   string

	ServiceName    string

	Namespace      string

	RevokedCert    *x509.Certificate

	RevocationTime time.Time

	Reason         string

	Priority       int

}



// NewEnhancedRevocationSystem creates a new enhanced revocation system.

func NewEnhancedRevocationSystem(

	config *EnhancedRevocationConfig,

	logger *logging.StructuredLogger,

	caManager *CAManager,

) (*EnhancedRevocationSystem, error) {

	if config == nil {

		return nil, fmt.Errorf("revocation config is required")

	}



	ctx, cancel := context.WithCancel(context.Background())



	system := &EnhancedRevocationSystem{

		config: config,

		logger: logger,

		ctx:    ctx,

		cancel: cancel,

	}



	// Initialize CRL manager.

	if config.CRLEnabled {

		system.crlManager = &CRLManager{

			distributionPoints: make(map[string]*CRLDistributionPoint),

			httpClient: &http.Client{

				Timeout: 30 * time.Second,

			},

			logger: logger,

		}



		if config.CRLDeltaEnabled {

			system.crlManager.deltaProcessor = &DeltaCRLProcessor{

				baseCRLs:  make(map[string]*pkix.CertificateList),

				deltaCRLs: make(map[string]*pkix.CertificateList),

			}

		}

	}



	// Initialize OCSP manager.

	if config.OCSPEnabled {

		system.ocspManager = &OCSPManager{

			responders: make(map[string]*OCSPResponder),

			logger:     logger,

		}



		if config.OCSPStaplingEnabled {

			system.ocspManager.staplingCache = &StaplingCache{

				responses: make(map[string]*StapledResponse),

				maxSize:   config.OCSPCacheSize,

				ttl:       config.OCSPCacheTTL,

			}

		}



		if config.OCSPNonceEnabled {

			system.ocspManager.nonceGenerator = &NonceGenerator{

				nonces: make(map[string]time.Time),

			}

		}



		system.ocspManager.connectionPool = &OCSPConnectionPool{

			connections: make(map[string]*PooledConnection),

			maxSize:     config.ConnectionPoolSize,

			timeout:     config.OCSPTimeout,

		}

	}



	// Initialize cache manager.

	if config.CacheEnabled {

		system.cacheManager = &RevocationCacheManager{

			l1Cache: &L1RevocationCache{

				entries: make(map[string]*RevocationCacheEntry),

				maxSize: config.L1CacheSize,

				ttl:     config.L1CacheTTL,

			},

			stats: &CacheStatistics{},

		}



		if config.L2CacheEnabled {

			// Initialize L2 cache (Redis or similar).

		}



		if config.PreloadCache {

			// Initialize preload cache.

		}

	}



	// Initialize batch processor.

	if config.BatchProcessingEnabled {

		system.batchProcessor = &BatchRevocationProcessor{

			batchQueue:   make(chan *RevocationBatch, 100),

			batchSize:    config.BatchSize,

			batchTimeout: config.BatchTimeout,

			logger:       logger,

		}

	}



	// Initialize async validator.

	if config.AsyncValidation {

		system.asyncValidator = &AsyncRevocationValidator{

			workers:   config.MaxConcurrentChecks,

			workQueue: make(chan *AsyncRevocationRequest, 1000),

			logger:    logger,

		}

	}



	// Initialize circuit breaker.

	if config.CircuitBreakerEnabled {

		system.circuitBreaker = &RevocationCircuitBreaker{

			crlBreaker: &CircuitBreaker{

				name: "CRL",

				config: &CircuitBreakerConfig{

					FailureThreshold: uint32(config.FailureThreshold),

					SuccessThreshold: 2,

					Timeout:          30 * time.Second,

				},

			},

			ocspBreaker: &CircuitBreaker{

				name: "OCSP",

				config: &CircuitBreakerConfig{

					FailureThreshold: uint32(config.FailureThreshold),

					SuccessThreshold: 2,

					Timeout:          30 * time.Second,

				},

			},

		}

		system.circuitBreaker.crlBreaker.state.Store(CircuitClosed)

		system.circuitBreaker.ocspBreaker.state.Store(CircuitClosed)

	}



	// Initialize automatic replacer.

	if config.AutoReplaceRevoked && caManager != nil {

		system.automaticReplacer = &AutomaticCertificateReplacer{

			caManager:        caManager,

			replacementQueue: make(chan *ReplacementRequest, 100),

			logger:           logger,

		}

	}



	// Initialize metrics collector.

	system.metricsCollector = NewRevocationMetricsCollector()



	return system, nil

}



// CheckRevocationEnhanced performs enhanced revocation checking.

func (s *EnhancedRevocationSystem) CheckRevocationEnhanced(ctx context.Context, cert *x509.Certificate) (*RevocationCheckResult, error) {

	start := time.Now()



	// Check cache first.

	if cached := s.checkCache(cert); cached != nil {

		s.metricsCollector.RecordCacheHit()

		return cached, nil

	}

	s.metricsCollector.RecordCacheMiss()



	// Check circuit breakers.

	if s.circuitBreaker != nil {

		if !s.circuitBreaker.AllowCRLCheck() && !s.circuitBreaker.AllowOCSPCheck() {

			if s.config.SoftFailEnabled {

				return &RevocationCheckResult{

					Status:   RevocationStatusUnknown,

					Warnings: []string{"all revocation check methods unavailable"},

				}, nil

			}

			return nil, fmt.Errorf("revocation checking unavailable due to circuit breaker")

		}

	}



	// Perform revocation check based on configuration.

	var result *RevocationCheckResult

	var err error



	if s.config.AsyncValidation && s.asyncValidator != nil {

		result, err = s.performAsyncCheck(ctx, cert)

	} else if s.config.BatchProcessingEnabled && s.batchProcessor != nil {

		result, err = s.performBatchCheck(ctx, cert)

	} else {

		result, err = s.performStandardCheck(ctx, cert)

	}



	if err != nil {

		s.handleCheckError(err)

		if s.config.SoftFailEnabled {

			return &RevocationCheckResult{

				Status:   RevocationStatusUnknown,

				Errors:   []string{err.Error()},

				Warnings: []string{"soft-fail mode: treating as unknown"},

			}, nil

		}

		return nil, err

	}



	// Cache the result.

	s.cacheResult(cert, result)



	// Handle revoked certificates.

	if result.Status == RevocationStatusRevoked && s.automaticReplacer != nil {

		s.scheduleReplacement(cert, result)

	}



	// Record metrics.

	duration := time.Since(start)

	s.metricsCollector.RecordCheckDuration(duration)

	s.metricsCollector.RecordCheckResult(result.Status)



	return result, nil

}



// performStandardCheck performs standard revocation checking.

func (s *EnhancedRevocationSystem) performStandardCheck(ctx context.Context, cert *x509.Certificate) (*RevocationCheckResult, error) {

	result := &RevocationCheckResult{

		SerialNumber: cert.SerialNumber.String(),

		Status:       RevocationStatusUnknown,

		CheckTime:    time.Now(),

		Details:      &RevocationCheckDetails{},

	}



	// Try OCSP first (usually faster).

	if s.config.OCSPEnabled && s.ocspManager != nil {

		ocspResult := s.checkOCSPEnhanced(ctx, cert)

		if ocspResult.Status != RevocationStatusUnknown {

			return ocspResult, nil

		}

		result.Errors = append(result.Errors, ocspResult.Errors...)

		result.Warnings = append(result.Warnings, ocspResult.Warnings...)

	}



	// Fall back to CRL.

	if s.config.CRLEnabled && s.crlManager != nil {

		crlResult := s.checkCRLEnhanced(ctx, cert)

		if crlResult.Status != RevocationStatusUnknown {

			return crlResult, nil

		}

		result.Errors = append(result.Errors, crlResult.Errors...)

		result.Warnings = append(result.Warnings, crlResult.Warnings...)

	}



	result.CheckDuration = time.Since(result.CheckTime)

	return result, nil

}



// checkOCSPEnhanced performs enhanced OCSP checking.

func (s *EnhancedRevocationSystem) checkOCSPEnhanced(ctx context.Context, cert *x509.Certificate) *RevocationCheckResult {

	result := &RevocationCheckResult{

		SerialNumber: cert.SerialNumber.String(),

		Status:       RevocationStatusUnknown,

		Method:       RevocationMethodOCSP,

		CheckTime:    time.Now(),

	}



	// Check OCSP stapling cache first.

	if s.config.OCSPStaplingEnabled {

		if stapled := s.ocspManager.staplingCache.Get(cert.SerialNumber.String()); stapled != nil {

			return s.parseStapledResponse(stapled, result)

		}

	}



	// Get best OCSP responder.

	responder := s.ocspManager.GetBestResponder(cert.OCSPServer)

	if responder == nil {

		result.Errors = append(result.Errors, "no available OCSP responders")

		return result

	}



	// Create OCSP request.

	ocspReq, err := s.createOCSPRequest(cert)

	if err != nil {

		result.Errors = append(result.Errors, fmt.Sprintf("failed to create OCSP request: %v", err))

		return result

	}



	// Add nonce if enabled.

	var nonce []byte

	if s.config.OCSPNonceEnabled {

		nonce = s.ocspManager.nonceGenerator.GenerateNonce()

		ocspReq = s.addNonceToRequest(ocspReq, nonce)

	}



	// Send OCSP request.

	ocspResp, err := s.sendOCSPRequest(ctx, responder, ocspReq, cert)

	if err != nil {

		atomic.AddUint32(&responder.ErrorCount, 1)

		s.circuitBreaker.ocspBreaker.RecordFailure()

		result.Errors = append(result.Errors, fmt.Sprintf("OCSP request failed: %v", err))

		return result

	}



	atomic.AddUint64(&responder.RequestCount, 1)

	s.circuitBreaker.ocspBreaker.RecordSuccess()



	// Parse OCSP response.

	return s.parseOCSPResponse(ocspResp, nonce, result)

}



// checkCRLEnhanced performs enhanced CRL checking.

func (s *EnhancedRevocationSystem) checkCRLEnhanced(ctx context.Context, cert *x509.Certificate) *RevocationCheckResult {

	result := &RevocationCheckResult{

		SerialNumber: cert.SerialNumber.String(),

		Status:       RevocationStatusUnknown,

		Method:       RevocationMethodCRL,

		CheckTime:    time.Now(),

	}



	// Get CRL distribution points.

	if len(cert.CRLDistributionPoints) == 0 {

		result.Errors = append(result.Errors, "no CRL distribution points in certificate")

		return result

	}



	// Check each distribution point.

	for _, dpURL := range cert.CRLDistributionPoints {

		dp := s.crlManager.GetOrCreateDistributionPoint(dpURL)



		// Update CRL if needed.

		if s.shouldUpdateCRL(dp) {

			if err := s.updateCRL(ctx, dp); err != nil {

				atomic.AddUint32(&dp.ErrorCount, 1)

				s.circuitBreaker.crlBreaker.RecordFailure()

				result.Errors = append(result.Errors, fmt.Sprintf("CRL update failed for %s: %v", dpURL, err))

				continue

			}

			s.circuitBreaker.crlBreaker.RecordSuccess()

		}



		// Check certificate against CRL.

		if s.isCertificateRevoked(cert, dp) {

			result.Status = RevocationStatusRevoked

			result.ResponseSource = dpURL

			// Get revocation details.

			if details := s.getRevocationDetails(cert, dp); details != nil {

				result.RevokedAt = details.RevokedAt

				result.RevocationReason = details.RevocationReason

			}

			break

		}



		// Certificate not in CRL - it's valid.

		result.Status = RevocationStatusValid

		result.ResponseSource = dpURL

		result.NextUpdate = &dp.NextUpdate

		break

	}



	result.CheckDuration = time.Since(result.CheckTime)

	return result

}



// performAsyncCheck performs asynchronous revocation checking.

func (s *EnhancedRevocationSystem) performAsyncCheck(ctx context.Context, cert *x509.Certificate) (*RevocationCheckResult, error) {

	request := &AsyncRevocationRequest{

		ID:          fmt.Sprintf("%s-%d", cert.SerialNumber.String(), time.Now().UnixNano()),

		Certificate: cert,

		Priority:    1,

		Timeout:     s.config.OCSPTimeout,

		ResultChan:  make(chan *RevocationCheckResult, 1),

		CreatedAt:   time.Now(),

	}



	// Submit request.

	select {

	case s.asyncValidator.workQueue <- request:

	case <-ctx.Done():

		return nil, ctx.Err()

	case <-time.After(1 * time.Second):

		return nil, fmt.Errorf("async validation queue full")

	}



	// Wait for result.

	select {

	case result := <-request.ResultChan:

		return result, nil

	case <-ctx.Done():

		return nil, ctx.Err()

	case <-time.After(request.Timeout):

		return nil, fmt.Errorf("async validation timeout")

	}

}



// performBatchCheck performs batch revocation checking.

func (s *EnhancedRevocationSystem) performBatchCheck(ctx context.Context, cert *x509.Certificate) (*RevocationCheckResult, error) {

	// Find or create batch.

	batch := s.batchProcessor.GetOrCreateBatch()



	// Add certificate to batch.

	batch.Certificates = append(batch.Certificates, cert)



	// Check if batch is ready.

	if len(batch.Certificates) >= s.config.BatchSize {

		// Process batch immediately.

		s.batchProcessor.ProcessBatch(batch)

	} else {

		// Schedule batch processing.

		s.batchProcessor.ScheduleBatch(batch)

	}



	// Wait for result.

	select {

	case <-batch.CompleteChan:

		if result, ok := batch.Results[cert.SerialNumber.String()]; ok {

			return result, nil

		}

		return nil, fmt.Errorf("certificate not found in batch results")

	case <-ctx.Done():

		return nil, ctx.Err()

	case <-time.After(s.config.BatchTimeout):

		return nil, fmt.Errorf("batch processing timeout")

	}

}



// Cache management.



func (s *EnhancedRevocationSystem) checkCache(cert *x509.Certificate) *RevocationCheckResult {

	if s.cacheManager == nil {

		return nil

	}



	key := cert.SerialNumber.String()



	// Check L1 cache.

	if entry := s.cacheManager.l1Cache.Get(key); entry != nil {

		atomic.AddUint32(&entry.HitCount, 1)

		entry.LastAccessed = time.Now()



		return &RevocationCheckResult{

			SerialNumber:   entry.SerialNumber,

			Status:         entry.Status,

			CheckTime:      entry.CheckTime,

			NextUpdate:     &entry.NextUpdate,

			ResponseSource: entry.Source,

			CacheHit:       true,

		}

	}



	// Check L2 cache if available.

	if s.cacheManager.l2Cache != nil {

		return s.cacheManager.l2Cache.Get(key)

	}



	return nil

}



func (s *EnhancedRevocationSystem) cacheResult(cert *x509.Certificate, result *RevocationCheckResult) {

	if s.cacheManager == nil {

		return

	}



	entry := &RevocationCacheEntry{

		SerialNumber: cert.SerialNumber.String(),

		Status:       result.Status,

		CheckTime:    result.CheckTime,

		Source:       result.ResponseSource,

	}



	if result.NextUpdate != nil {

		entry.NextUpdate = *result.NextUpdate

	}



	// Store in L1 cache.

	s.cacheManager.l1Cache.Put(cert.SerialNumber.String(), entry)



	// Store in L2 cache if available.

	if s.cacheManager.l2Cache != nil {

		s.cacheManager.l2Cache.Put(cert.SerialNumber.String(), entry)

	}

}



// CRL management.



func (s *EnhancedRevocationSystem) shouldUpdateCRL(dp *CRLDistributionPoint) bool {

	dp.mu.RLock()

	defer dp.mu.RUnlock()



	// Check if update is already in progress.

	if atomic.LoadInt32(&dp.UpdateInProgress) == 1 {

		return false

	}



	// Check if CRL exists.

	if dp.CRL == nil {

		return true

	}



	// Check if CRL has expired.

	if time.Now().After(dp.NextUpdate) {

		return true

	}



	// Check if it's time for scheduled update.

	updateWindow := dp.NextUpdate.Sub(dp.LastUpdate) / 4

	if time.Since(dp.LastUpdate) > updateWindow*3 {

		return true

	}



	return false

}



func (s *EnhancedRevocationSystem) updateCRL(ctx context.Context, dp *CRLDistributionPoint) error {

	// Mark update in progress.

	if !atomic.CompareAndSwapInt32(&dp.UpdateInProgress, 0, 1) {

		return nil // Update already in progress

	}

	defer atomic.StoreInt32(&dp.UpdateInProgress, 0)



	// Fetch CRL.

	req, err := http.NewRequestWithContext(ctx, "GET", dp.URL, http.NoBody)

	if err != nil {

		return fmt.Errorf("failed to create CRL request: %w", err)

	}



	// Add If-Modified-Since header if we have a previous CRL.

	dp.mu.RLock()

	if dp.CRL != nil && !dp.LastUpdate.IsZero() {

		req.Header.Set("If-Modified-Since", dp.LastUpdate.Format(http.TimeFormat))

	}

	dp.mu.RUnlock()



	resp, err := s.crlManager.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("CRL request failed: %w", err)

	}

	defer resp.Body.Close()



	// Handle 304 Not Modified.

	if resp.StatusCode == http.StatusNotModified {

		s.logger.Debug("CRL not modified", "url", dp.URL)

		return nil

	}



	if resp.StatusCode != http.StatusOK {

		return fmt.Errorf("CRL server returned status %d", resp.StatusCode)

	}



	// Check content length.

	if resp.ContentLength > s.config.CRLMaxSize {

		return fmt.Errorf("CRL size (%d) exceeds maximum (%d)", resp.ContentLength, s.config.CRLMaxSize)

	}



	// Read CRL data.

	crlData, err := io.ReadAll(io.LimitReader(resp.Body, s.config.CRLMaxSize))

	if err != nil {

		return fmt.Errorf("failed to read CRL data: %w", err)

	}



	// Parse CRL.

	crl, err := x509.ParseCRL(crlData)

	if err != nil {

		return fmt.Errorf("failed to parse CRL: %w", err)

	}



	// Update distribution point.

	dp.mu.Lock()

	dp.CRL = crl

	dp.LastUpdate = time.Now()

	// pkix.CertificateList doesn't have NextUpdate field - set a reasonable default.

	dp.NextUpdate = time.Now().Add(24 * time.Hour) // Default to 24 hours

	dp.mu.Unlock()



	// Process delta CRL if enabled.

	if s.config.CRLDeltaEnabled && s.crlManager.deltaProcessor != nil {

		s.crlManager.deltaProcessor.processDelta(dp.URL, crl)

	}



	s.logger.Info("CRL updated successfully",

		"url", dp.URL,

		"next_update", dp.NextUpdate,

		"revoked_count", len(crl.TBSCertList.RevokedCertificates))



	return nil

}



func (s *EnhancedRevocationSystem) isCertificateRevoked(cert *x509.Certificate, dp *CRLDistributionPoint) bool {

	dp.mu.RLock()

	defer dp.mu.RUnlock()



	if dp.CRL == nil {

		return false

	}



	// Check base CRL.

	for _, revoked := range dp.CRL.TBSCertList.RevokedCertificates {

		if revoked.SerialNumber.Cmp(cert.SerialNumber) == 0 {

			return true

		}

	}



	// Check delta CRL if available.

	if dp.DeltaCRL != nil {

		for _, revoked := range dp.DeltaCRL.TBSCertList.RevokedCertificates {

			if revoked.SerialNumber.Cmp(cert.SerialNumber) == 0 {

				return true

			}

		}

	}



	return false

}



func (s *EnhancedRevocationSystem) getRevocationDetails(cert *x509.Certificate, dp *CRLDistributionPoint) *RevocationDetails {

	dp.mu.RLock()

	defer dp.mu.RUnlock()



	if dp.CRL == nil {

		return nil

	}



	for _, revoked := range dp.CRL.TBSCertList.RevokedCertificates {

		if revoked.SerialNumber.Cmp(cert.SerialNumber) == 0 {

			details := &RevocationDetails{

				RevokedAt: &revoked.RevocationTime,

			}



			// Extract revocation reason.

			for _, ext := range revoked.Extensions {

				if ext.Id.Equal([]int{2, 5, 29, 21}) { // reasonCode extension OID

					if len(ext.Value) > 0 {

						reason := int(ext.Value[0])

						details.RevocationReason = &reason

					}

				}

			}



			return details

		}

	}



	return nil

}



// OCSP methods.



func (s *EnhancedRevocationSystem) createOCSPRequest(cert *x509.Certificate) ([]byte, error) {

	// Create OCSP request.

	// This is simplified - real implementation needs issuer certificate.

	issuerNameHash := sha256.Sum256(cert.RawIssuer)

	issuerKeyHash := sha256.Sum256(cert.AuthorityKeyId)



	template := ocsp.Request{

		HashAlgorithm:  crypto.SHA256, // Use crypto.SHA256 instead of ocsp.SHA256

		IssuerNameHash: issuerNameHash[:],

		IssuerKeyHash:  issuerKeyHash[:],

		SerialNumber:   cert.SerialNumber,

	}



	return template.Marshal()

}



func (s *EnhancedRevocationSystem) addNonceToRequest(req, nonce []byte) []byte {

	// Add nonce extension to OCSP request.

	// This is a simplified implementation.

	return req

}



func (s *EnhancedRevocationSystem) sendOCSPRequest(ctx context.Context, responder *OCSPResponder, reqData []byte, cert *x509.Certificate) (*ocsp.Response, error) {

	req, err := http.NewRequestWithContext(ctx, "POST", responder.URL, bytes.NewReader(reqData))

	if err != nil {

		return nil, fmt.Errorf("failed to create OCSP request: %w", err)

	}



	req.Header.Set("Content-Type", "application/ocsp-request")

	req.Header.Set("Accept", "application/ocsp-response")



	start := time.Now()

	resp, err := responder.Client.Do(req)

	if err != nil {

		return nil, fmt.Errorf("OCSP request failed: %w", err)

	}

	defer resp.Body.Close()



	responder.ResponseTime = time.Since(start)



	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("OCSP server returned status %d", resp.StatusCode)

	}



	respData, err := io.ReadAll(resp.Body)

	if err != nil {

		return nil, fmt.Errorf("failed to read OCSP response: %w", err)

	}



	ocspResp, err := ocsp.ParseResponse(respData, nil)

	if err != nil {

		return nil, fmt.Errorf("failed to parse OCSP response: %w", err)

	}



	// Cache stapled response if enabled.

	if s.config.OCSPStaplingEnabled && cert != nil {

		s.ocspManager.staplingCache.Put(cert.SerialNumber.String(), respData, ocspResp)

	}



	return ocspResp, nil

}



func (s *EnhancedRevocationSystem) parseOCSPResponse(resp *ocsp.Response, nonce []byte, result *RevocationCheckResult) *RevocationCheckResult {

	// Verify nonce if provided.

	if nonce != nil && !s.verifyOCSPNonce(resp, nonce) {

		result.Warnings = append(result.Warnings, "OCSP nonce verification failed")

	}



	switch resp.Status {

	case ocsp.Good:

		result.Status = RevocationStatusValid

	case ocsp.Revoked:

		result.Status = RevocationStatusRevoked

		result.RevokedAt = &resp.RevokedAt

		result.RevocationReason = &resp.RevocationReason

	default:

		result.Status = RevocationStatusUnknown

	}



	result.NextUpdate = &resp.NextUpdate

	result.CheckDuration = time.Since(result.CheckTime)



	// Add OCSP details using the existing OCSPInfo from revocation_checker.go.

	if result.Details == nil {

		result.Details = &RevocationCheckDetails{}

	}

	result.Details.OCSPInfo = &OCSPInfo{

		ResponderURL: "", // Will be filled by caller

		ProducedAt:   resp.ProducedAt,

		ThisUpdate:   resp.ThisUpdate,

		NextUpdate:   resp.NextUpdate,

		SignatureAlg: resp.SignatureAlgorithm.String(),

	}



	return result

}



func (s *EnhancedRevocationSystem) parseStapledResponse(stapled *StapledResponse, result *RevocationCheckResult) *RevocationCheckResult {

	resp, err := ocsp.ParseResponse(stapled.Response, nil)

	if err != nil {

		result.Errors = append(result.Errors, fmt.Sprintf("failed to parse stapled response: %v", err))

		return result

	}



	return s.parseOCSPResponse(resp, nil, result)

}



func (s *EnhancedRevocationSystem) verifyOCSPNonce(resp *ocsp.Response, expectedNonce []byte) bool {

	// Extract nonce from response and compare.

	// This is a simplified implementation.

	return true

}



// Automatic replacement.



func (s *EnhancedRevocationSystem) scheduleReplacement(cert *x509.Certificate, result *RevocationCheckResult) {

	if s.automaticReplacer == nil {

		return

	}



	request := &ReplacementRequest{

		SerialNumber:   cert.SerialNumber.String(),

		RevokedCert:    cert,

		RevocationTime: time.Now(),

		Priority:       1,

	}



	if result.RevokedAt != nil {

		request.RevocationTime = *result.RevokedAt

	}



	if result.RevocationReason != nil {

		request.Reason = s.getRevocationReasonString(*result.RevocationReason)

	}



	select {

	case s.automaticReplacer.replacementQueue <- request:

		s.logger.Info("scheduled automatic certificate replacement",

			"serial_number", cert.SerialNumber.String(),

			"reason", request.Reason)

	default:

		s.logger.Warn("replacement queue full, dropping request",

			"serial_number", cert.SerialNumber.String())

	}

}



func (s *EnhancedRevocationSystem) getRevocationReasonString(reason int) string {

	reasons := map[int]string{

		0:  "unspecified",

		1:  "keyCompromise",

		2:  "caCompromise",

		3:  "affiliationChanged",

		4:  "superseded",

		5:  "cessationOfOperation",

		6:  "certificateHold",

		8:  "removeFromCRL",

		9:  "privilegeWithdrawn",

		10: "aaCompromise",

	}



	if str, ok := reasons[reason]; ok {

		return str

	}

	return fmt.Sprintf("unknown (%d)", reason)

}



// Error handling.



func (s *EnhancedRevocationSystem) handleCheckError(err error) {

	s.logger.Error("revocation check failed", "error", err)

	s.metricsCollector.RecordError(err)



	// Update circuit breaker if needed.

	if s.circuitBreaker != nil {

		// Determine which circuit breaker to update based on error.

		// This is simplified - real implementation would parse error.

		s.circuitBreaker.crlBreaker.RecordFailure()

	}

}



// Start starts the enhanced revocation system.

func (s *EnhancedRevocationSystem) Start(ctx context.Context) error {

	s.logger.Info("starting enhanced revocation system")



	// Start CRL updater.

	if s.crlManager != nil {

		s.wg.Add(1)

		go s.runCRLUpdater()

	}



	// Start async validator workers.

	if s.asyncValidator != nil {

		for i := range s.asyncValidator.workers {

			s.wg.Add(1)

			go s.runAsyncWorker(i)

		}

	}



	// Start batch processor.

	if s.batchProcessor != nil {

		s.wg.Add(1)

		go s.runBatchProcessor()

	}



	// Start automatic replacer.

	if s.automaticReplacer != nil {

		s.wg.Add(1)

		go s.runAutomaticReplacer()

	}



	// Start cache cleanup.

	s.wg.Add(1)

	go s.runCacheCleanup()



	return nil

}



// Stop stops the enhanced revocation system.

func (s *EnhancedRevocationSystem) Stop() {

	s.logger.Info("stopping enhanced revocation system")

	s.cancel()

	s.wg.Wait()

}



// Worker routines.



func (s *EnhancedRevocationSystem) runCRLUpdater() {

	defer s.wg.Done()



	ticker := time.NewTicker(s.config.CRLUpdateInterval)

	defer ticker.Stop()



	for {

		select {

		case <-s.ctx.Done():

			return

		case <-ticker.C:

			s.updateAllCRLs()

		}

	}

}



func (s *EnhancedRevocationSystem) updateAllCRLs() {

	s.crlManager.mu.RLock()

	distributionPoints := make([]*CRLDistributionPoint, 0, len(s.crlManager.distributionPoints))

	for _, dp := range s.crlManager.distributionPoints {

		distributionPoints = append(distributionPoints, dp)

	}

	s.crlManager.mu.RUnlock()



	for _, dp := range distributionPoints {

		if s.shouldUpdateCRL(dp) {

			ctx, cancel := context.WithTimeout(s.ctx, 30*time.Second)

			if err := s.updateCRL(ctx, dp); err != nil {

				s.logger.Warn("CRL update failed",

					"url", dp.URL,

					"error", err)

			}

			cancel()

		}

	}

}



func (s *EnhancedRevocationSystem) runAsyncWorker(workerID int) {

	defer s.wg.Done()



	for {

		select {

		case <-s.ctx.Done():

			return

		case req := <-s.asyncValidator.workQueue:

			s.processAsyncRequest(workerID, req)

		}

	}

}



func (s *EnhancedRevocationSystem) processAsyncRequest(workerID int, req *AsyncRevocationRequest) {

	ctx, cancel := context.WithTimeout(s.ctx, req.Timeout)

	defer cancel()



	result, err := s.performStandardCheck(ctx, req.Certificate)

	if err != nil {

		result = &RevocationCheckResult{

			Status: RevocationStatusUnknown,

			Errors: []string{err.Error()},

		}

	}



	select {

	case req.ResultChan <- result:

	case <-time.After(1 * time.Second):

		s.logger.Warn("async result channel blocked",

			"worker_id", workerID,

			"request_id", req.ID)

	}

}



func (s *EnhancedRevocationSystem) runBatchProcessor() {

	defer s.wg.Done()



	for {

		select {

		case <-s.ctx.Done():

			return

		case batch := <-s.batchProcessor.batchQueue:

			s.processBatch(batch)

		}

	}

}



func (s *EnhancedRevocationSystem) processBatch(batch *RevocationBatch) {

	batch.Results = make(map[string]*RevocationCheckResult)



	// Process certificates in parallel.

	var wg sync.WaitGroup

	resultChan := make(chan struct {

		SerialNumber string

		Result       *RevocationCheckResult

	}, len(batch.Certificates))



	for _, cert := range batch.Certificates {

		wg.Add(1)

		go func(c *x509.Certificate) {

			defer wg.Done()



			ctx, cancel := context.WithTimeout(s.ctx, s.config.OCSPTimeout)

			defer cancel()



			result, _ := s.performStandardCheck(ctx, c)

			resultChan <- struct {

				SerialNumber string

				Result       *RevocationCheckResult

			}{

				SerialNumber: c.SerialNumber.String(),

				Result:       result,

			}

		}(cert)

	}



	// Wait for all checks to complete.

	go func() {

		wg.Wait()

		close(resultChan)

	}()



	// Collect results.

	for r := range resultChan {

		batch.Results[r.SerialNumber] = r.Result

	}



	// Signal completion.

	close(batch.CompleteChan)

}



func (s *EnhancedRevocationSystem) runAutomaticReplacer() {

	defer s.wg.Done()



	for {

		select {

		case <-s.ctx.Done():

			return

		case req := <-s.automaticReplacer.replacementQueue:

			s.processReplacementRequest(req)

		}

	}

}



func (s *EnhancedRevocationSystem) processReplacementRequest(req *ReplacementRequest) {

	s.logger.Info("processing certificate replacement request",

		"serial_number", req.SerialNumber,

		"reason", req.Reason)



	// Create new certificate request.

	certReq := &CertificateRequest{

		CommonName:       req.RevokedCert.Subject.CommonName,

		DNSNames:         req.RevokedCert.DNSNames,

		IPAddresses:      make([]string, len(req.RevokedCert.IPAddresses)),

		ValidityDuration: time.Until(req.RevokedCert.NotAfter),

		AutoRenew:        true,

		Metadata: map[string]string{

			"replaced_serial":    req.SerialNumber,

			"replacement_reason": req.Reason,

			"replacement_time":   time.Now().Format(time.RFC3339),

		},

	}



	for i, ip := range req.RevokedCert.IPAddresses {

		certReq.IPAddresses[i] = ip.String()

	}



	// Issue new certificate.

	ctx, cancel := context.WithTimeout(s.ctx, s.config.ReplacementTimeout)

	defer cancel()



	newCert, err := s.automaticReplacer.caManager.IssueCertificate(ctx, certReq)

	if err != nil {

		s.logger.Error("failed to issue replacement certificate",

			"serial_number", req.SerialNumber,

			"error", err)

		return

	}



	s.logger.Info("replacement certificate issued successfully",

		"old_serial", req.SerialNumber,

		"new_serial", newCert.SerialNumber)



	// Update in Kubernetes if applicable.

	// This would integrate with the Kubernetes client to update secrets.

}



func (s *EnhancedRevocationSystem) runCacheCleanup() {

	defer s.wg.Done()



	ticker := time.NewTicker(5 * time.Minute)

	defer ticker.Stop()



	for {

		select {

		case <-s.ctx.Done():

			return

		case <-ticker.C:

			s.cleanupCache()

		}

	}

}



func (s *EnhancedRevocationSystem) cleanupCache() {

	if s.cacheManager != nil && s.cacheManager.l1Cache != nil {

		s.cacheManager.l1Cache.Cleanup()

	}

}



// Helper types and methods.



// RevocationDetails represents a revocationdetails.

type RevocationDetails struct {

	RevokedAt        *time.Time

	RevocationReason *int

}



// RevocationProcessor represents a revocationprocessor.

type RevocationProcessor interface {

	ProcessRevocation(ctx context.Context, cert *x509.Certificate) (*RevocationCheckResult, error)

}



// ResultCollector represents a resultcollector.

type ResultCollector struct {

	results map[string]*RevocationCheckResult

	mu      sync.RWMutex

}



// CacheStatistics represents a cachestatistics.

type CacheStatistics struct {

	Hits      uint64 // atomic counter (replaced atomic.Uint64)

	Misses    uint64 // atomic counter (replaced atomic.Uint64)

	Evictions uint64 // atomic counter (replaced atomic.Uint64)

	Entries   uint32 // atomic counter (replaced atomic.Uint32)

}



// RevocationMetricsCollector represents a revocationmetricscollector.

type RevocationMetricsCollector struct {

	totalChecks    uint64 // atomic counter (replaced atomic.Uint64)

	cacheHits      uint64 // atomic counter (replaced atomic.Uint64)

	cacheMisses    uint64 // atomic counter (replaced atomic.Uint64)

	checkDurations []time.Duration

	mu             sync.RWMutex

}



// NewRevocationMetricsCollector performs newrevocationmetricscollector operation.

func NewRevocationMetricsCollector() *RevocationMetricsCollector {

	return &RevocationMetricsCollector{

		checkDurations: make([]time.Duration, 0, 1000),

	}

}



// RecordCacheHit performs recordcachehit operation.

func (m *RevocationMetricsCollector) RecordCacheHit() {

	atomic.AddUint64(&m.cacheHits, 1)

}



// RecordCacheMiss performs recordcachemiss operation.

func (m *RevocationMetricsCollector) RecordCacheMiss() {

	atomic.AddUint64(&m.cacheMisses, 1)

}



// RecordCheckDuration performs recordcheckduration operation.

func (m *RevocationMetricsCollector) RecordCheckDuration(d time.Duration) {

	m.mu.Lock()

	defer m.mu.Unlock()



	m.checkDurations = append(m.checkDurations, d)

	if len(m.checkDurations) > 1000 {

		m.checkDurations = m.checkDurations[1:]

	}

}



// RecordCheckResult performs recordcheckresult operation.

func (m *RevocationMetricsCollector) RecordCheckResult(status RevocationStatus) {

	atomic.AddUint64(&m.totalChecks, 1)

}



// RecordError performs recorderror operation.

func (m *RevocationMetricsCollector) RecordError(err error) {

	// Record error metrics.

}



// CRL Manager methods.



// GetOrCreateDistributionPoint performs getorcreatedistributionpoint operation.

func (m *CRLManager) GetOrCreateDistributionPoint(url string) *CRLDistributionPoint {

	m.mu.RLock()

	dp, exists := m.distributionPoints[url]

	m.mu.RUnlock()



	if exists {

		return dp

	}



	m.mu.Lock()

	defer m.mu.Unlock()



	// Double-check after acquiring write lock.

	if dp, exists := m.distributionPoints[url]; exists {

		return dp

	}



	dp = &CRLDistributionPoint{

		URL: url,

	}

	m.distributionPoints[url] = dp



	return dp

}



// OCSP Manager methods.



// GetBestResponder performs getbestresponder operation.

func (m *OCSPManager) GetBestResponder(urls []string) *OCSPResponder {

	if len(urls) == 0 {

		return nil

	}



	m.mu.RLock()

	defer m.mu.RUnlock()



	var bestResponder *OCSPResponder

	var bestScore float64



	for _, url := range urls {

		responder, exists := m.responders[url]

		if !exists {

			// Create new responder.

			responder = &OCSPResponder{

				URL: url,

				Client: &http.Client{

					Timeout: 10 * time.Second,

				},

			}

			m.mu.RUnlock()

			m.mu.Lock()

			m.responders[url] = responder

			m.mu.Unlock()

			m.mu.RLock()

		}



		// Calculate responder score based on success rate and response time.

		score := responder.SuccessRate

		if responder.ResponseTime > 0 {

			score = score / float64(responder.ResponseTime.Milliseconds())

		}



		if bestResponder == nil || score > bestScore {

			bestResponder = responder

			bestScore = score

		}

	}



	return bestResponder

}



// Stapling Cache methods.



// Get performs get operation.

func (c *StaplingCache) Get(serialNumber string) *StapledResponse {

	c.mu.RLock()

	defer c.mu.RUnlock()



	response, exists := c.responses[serialNumber]

	if !exists {

		return nil

	}



	// Check if response is still valid.

	if time.Now().After(response.NextUpdate) {

		return nil

	}



	return response

}



// Put performs put operation.

func (c *StaplingCache) Put(serialNumber string, responseData []byte, resp *ocsp.Response) {

	c.mu.Lock()

	defer c.mu.Unlock()



	// Check cache size.

	if len(c.responses) >= c.maxSize {

		c.evictOldest()

	}



	c.responses[serialNumber] = &StapledResponse{

		Response:   responseData,

		ProducedAt: resp.ProducedAt,

		NextUpdate: resp.NextUpdate,

		CachedAt:   time.Now(),

	}

}



func (c *StaplingCache) evictOldest() {

	var oldestKey string

	var oldestTime time.Time



	for key, response := range c.responses {

		if oldestKey == "" || response.CachedAt.Before(oldestTime) {

			oldestKey = key

			oldestTime = response.CachedAt

		}

	}



	if oldestKey != "" {

		delete(c.responses, oldestKey)

	}

}



// Nonce Generator methods.



// GenerateNonce performs generatenonce operation.

func (g *NonceGenerator) GenerateNonce() []byte {

	nonce := make([]byte, 32)

	// Generate random nonce.

	// crypto/rand.Read(nonce).



	// Store nonce.

	g.mu.Lock()

	g.nonces[hex.EncodeToString(nonce)] = time.Now()

	g.mu.Unlock()



	// Cleanup old nonces.

	g.cleanupOldNonces()



	return nonce

}



func (g *NonceGenerator) cleanupOldNonces() {

	g.mu.Lock()

	defer g.mu.Unlock()



	cutoff := time.Now().Add(-5 * time.Minute)

	for key, timestamp := range g.nonces {

		if timestamp.Before(cutoff) {

			delete(g.nonces, key)

		}

	}

}



// L1 Revocation Cache methods.



// Get performs get operation.

func (c *L1RevocationCache) Get(key string) *RevocationCacheEntry {

	c.mu.RLock()

	defer c.mu.RUnlock()



	entry, exists := c.entries[key]

	if !exists {

		return nil

	}



	// Check TTL.

	if time.Since(entry.CheckTime) > c.ttl {

		return nil

	}



	return entry

}



// Put performs put operation.

func (c *L1RevocationCache) Put(key string, entry *RevocationCacheEntry) {

	c.mu.Lock()

	defer c.mu.Unlock()



	// Check size limit.

	if len(c.entries) >= c.maxSize {

		c.evictLRU()

	}



	c.entries[key] = entry

}



func (c *L1RevocationCache) evictLRU() {

	var lruKey string

	var lruTime time.Time



	for key, entry := range c.entries {

		if lruKey == "" || entry.LastAccessed.Before(lruTime) {

			lruKey = key

			lruTime = entry.LastAccessed

		}

	}



	if lruKey != "" {

		delete(c.entries, lruKey)

	}

}



// Cleanup performs cleanup operation.

func (c *L1RevocationCache) Cleanup() {

	c.mu.Lock()

	defer c.mu.Unlock()



	now := time.Now()

	for key, entry := range c.entries {

		if now.Sub(entry.CheckTime) > c.ttl {

			delete(c.entries, key)

		}

	}

}



// Circuit Breaker methods.



// AllowCRLCheck performs allowcrlcheck operation.

func (cb *RevocationCircuitBreaker) AllowCRLCheck() bool {

	return cb.crlBreaker.AllowRequest()

}



// AllowOCSPCheck performs allowocspcheck operation.

func (cb *RevocationCircuitBreaker) AllowOCSPCheck() bool {

	return cb.ocspBreaker.AllowRequest()

}



// AllowRequest performs allowrequest operation.

func (cb *CircuitBreaker) AllowRequest() bool {

	state := cb.state.Load().(CircuitState)



	switch state {

	case CircuitClosed:

		return true

	case CircuitOpen:

		// Check if timeout has passed.

		lastFailure := cb.lastFailureTime.Load().(time.Time)

		if time.Since(lastFailure) > cb.config.Timeout {

			cb.state.Store(CircuitHalfOpen)

			return true

		}

		return false

	case CircuitHalfOpen:

		// Allow limited requests.

		return atomic.LoadUint32(&cb.successCount) < cb.config.HalfOpenRequests

	default:

		return false

	}

}



// RecordSuccess performs recordsuccess operation.

func (cb *CircuitBreaker) RecordSuccess() {

	atomic.AddUint32(&cb.successCount, 1)

	state := cb.state.Load().(CircuitState)



	if state == CircuitHalfOpen && atomic.LoadUint32(&cb.successCount) >= cb.config.SuccessThreshold {

		cb.state.Store(CircuitClosed)

		atomic.StoreUint32(&cb.failureCount, 0)

		atomic.StoreUint32(&cb.successCount, 0)

	}

}



// RecordFailure performs recordfailure operation.

func (cb *CircuitBreaker) RecordFailure() {

	atomic.AddUint32(&cb.failureCount, 1)

	cb.lastFailureTime.Store(time.Now())



	if atomic.LoadUint32(&cb.failureCount) >= cb.config.FailureThreshold {

		cb.state.Store(CircuitOpen)

	}

}



// Batch Processor methods.



// GetOrCreateBatch performs getorcreatebatch operation.

func (bp *BatchRevocationProcessor) GetOrCreateBatch() *RevocationBatch {

	// This is simplified - real implementation would manage batch lifecycle.

	return &RevocationBatch{

		ID:           fmt.Sprintf("batch-%d", time.Now().UnixNano()),

		Certificates: make([]*x509.Certificate, 0, bp.batchSize),

		CompleteChan: make(chan struct{}),

		CreatedAt:    time.Now(),

	}

}



// ProcessBatch performs processbatch operation.

func (bp *BatchRevocationProcessor) ProcessBatch(batch *RevocationBatch) {

	select {

	case bp.batchQueue <- batch:

	default:

		bp.logger.Warn("batch queue full")

	}

}



// ScheduleBatch performs schedulebatch operation.

func (bp *BatchRevocationProcessor) ScheduleBatch(batch *RevocationBatch) {

	// Schedule batch processing after timeout.

	time.AfterFunc(bp.batchTimeout, func() {

		bp.ProcessBatch(batch)

	})

}



// Placeholder types for undefined components.



// L2RevocationCache represents a l2revocationcache.

type L2RevocationCache struct{}



// Get performs get operation.

func (c *L2RevocationCache) Get(_ string) *RevocationCheckResult { return nil }



// Put performs put operation.

func (c *L2RevocationCache) Put(key string, entry *RevocationCacheEntry) {}



// PreloadRevocationCache represents a preloadrevocationcache.

type PreloadRevocationCache struct{}



// OCSPClient represents a ocspclient.

type OCSPClient struct {

	URL     string

	Timeout time.Duration

}



// NewOCSPClient performs newocspclient operation.

func NewOCSPClient(url string, timeout time.Duration) *OCSPClient {

	return &OCSPClient{URL: url, Timeout: timeout}

}



// CheckRevocation performs checkrevocation operation.

func (c *PooledConnection) CheckRevocation(ctx context.Context, cert *x509.Certificate) *RevocationCheckResult {

	// Simplified implementation.

	return &RevocationCheckResult{

		Status: RevocationStatusUnknown,

	}

}



// LRUEvictionPolicy represents a lruevictionpolicy.

type (

	LRUEvictionPolicy struct{}

	// WebhookClient represents a webhookclient.

	WebhookClient struct{}

)



// NewWebhookClient performs newwebhookclient operation.

func NewWebhookClient(logger *logging.StructuredLogger) *WebhookClient { return &WebhookClient{} }



// SendEvent performs sendevent operation.

func (c *WebhookClient) SendEvent(event *ValidationEvent) {}



// AuditLogger represents a auditlogger.

type AuditLogger struct{}



// NewAuditLogger performs newauditlogger operation.

func NewAuditLogger(logger *logging.StructuredLogger) *AuditLogger { return &AuditLogger{} }



// LogEmergencyBypass performs logemergencybypass operation.

func (a *AuditLogger) LogEmergencyBypass(authKey string, duration time.Duration, reason string) {}

