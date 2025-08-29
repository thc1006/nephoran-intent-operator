
package ca



import (

	"context"

	"crypto/x509"

	"crypto/x509/pkix"

	"fmt"

	"io"

	"net/http"

	"sync"

	"time"



	"golang.org/x/crypto/ocsp"



	"github.com/nephio-project/nephoran-intent-operator/pkg/logging"

)



// RevocationChecker provides comprehensive certificate revocation checking.

type RevocationChecker struct {

	config     *RevocationConfig

	logger     *logging.StructuredLogger

	crlCache   *CRLCache

	ocspCache  *OCSPCache

	httpClient *http.Client

	mu         sync.RWMutex

}



// RevocationConfig configures revocation checking.

type RevocationConfig struct {

	CRLCheckEnabled  bool          `yaml:"crl_check_enabled"`

	OCSPCheckEnabled bool          `yaml:"ocsp_check_enabled"`

	CacheSize        int           `yaml:"cache_size"`

	CacheTTL         time.Duration `yaml:"cache_ttl"`

	Timeout          time.Duration `yaml:"timeout"`

	MaxCRLSize       int64         `yaml:"max_crl_size"`

	OCSPFallback     bool          `yaml:"ocsp_fallback"`

	CRLFallback      bool          `yaml:"crl_fallback"`

	ParallelCheck    bool          `yaml:"parallel_check"`

	RetryAttempts    int           `yaml:"retry_attempts"`

	RetryDelay       time.Duration `yaml:"retry_delay"`

}



// CRLCache caches Certificate Revocation Lists.

type CRLCache struct {

	cache   map[string]*CRLEntry

	ttl     time.Duration

	maxSize int

	mu      sync.RWMutex

}



// CRLEntry represents a cached CRL entry.

type CRLEntry struct {

	URL          string                `json:"url"`

	CRL          *pkix.CertificateList `json:"-"`

	LastModified time.Time             `json:"last_modified"`

	NextUpdate   time.Time             `json:"next_update"`

	CachedAt     time.Time             `json:"cached_at"`

	Size         int64                 `json:"size"`

	EntryCount   int                   `json:"entry_count"`

}



// OCSPCache caches OCSP responses.

type OCSPCache struct {

	cache   map[string]*OCSPEntry

	ttl     time.Duration

	maxSize int

	mu      sync.RWMutex

}



// OCSPEntry represents a cached OCSP entry.

type OCSPEntry struct {

	SerialNumber string         `json:"serial_number"`

	IssuerHash   string         `json:"issuer_hash"`

	Status       int            `json:"status"`

	Response     *ocsp.Response `json:"-"`

	NextUpdate   time.Time      `json:"next_update"`

	CachedAt     time.Time      `json:"cached_at"`

}



// RevocationCheckResult represents the result of a revocation check.

type RevocationCheckResult struct {

	SerialNumber     string                  `json:"serial_number"`

	Status           RevocationStatus        `json:"status"`

	Method           RevocationMethod        `json:"method"`

	CheckTime        time.Time               `json:"check_time"`

	RevokedAt        *time.Time              `json:"revoked_at,omitempty"`

	RevocationReason *int                    `json:"revocation_reason,omitempty"`

	NextUpdate       *time.Time              `json:"next_update,omitempty"`

	ResponseSource   string                  `json:"response_source"`

	CacheHit         bool                    `json:"cache_hit"`

	CheckDuration    time.Duration           `json:"check_duration"`

	Details          *RevocationCheckDetails `json:"details,omitempty"`

	Errors           []string                `json:"errors,omitempty"`

	Warnings         []string                `json:"warnings,omitempty"`

}



// RevocationMethod represents the method used for revocation checking.

type RevocationMethod string



const (

	// RevocationMethodCRL holds revocationmethodcrl value.

	RevocationMethodCRL RevocationMethod = "crl"

	// RevocationMethodOCSP holds revocationmethodocsp value.

	RevocationMethodOCSP RevocationMethod = "ocsp"

	// RevocationMethodCached holds revocationmethodcached value.

	RevocationMethodCached RevocationMethod = "cached"

	// RevocationMethodHybrid holds revocationmethodhybrid value.

	RevocationMethodHybrid RevocationMethod = "hybrid"

)



// RevocationCheckDetails provides detailed information about the revocation check.

type RevocationCheckDetails struct {

	CRLDistributionPoints []string        `json:"crl_distribution_points,omitempty"`

	OCSPServers           []string        `json:"ocsp_servers,omitempty"`

	IssuerInfo            *IssuerInfo     `json:"issuer_info,omitempty"`

	CRLInfo               *CRLInfo        `json:"crl_info,omitempty"`

	OCSPInfo              *OCSPInfo       `json:"ocsp_info,omitempty"`

	MethodsFailed         []MethodFailure `json:"methods_failed,omitempty"`

}



// IssuerInfo provides information about the certificate issuer.

type IssuerInfo struct {

	Subject string `json:"subject"`

	KeyID   string `json:"key_id,omitempty"`

	Hash    string `json:"hash"`

}



// CRLInfo provides information about CRL checking.

type CRLInfo struct {

	URL          string    `json:"url"`

	LastModified time.Time `json:"last_modified"`

	NextUpdate   time.Time `json:"next_update"`

	EntryCount   int       `json:"entry_count"`

	Size         int64     `json:"size"`

	Version      int       `json:"version"`

}



// OCSPInfo provides information about OCSP checking.

type OCSPInfo struct {

	ResponderURL string    `json:"responder_url"`

	ProducedAt   time.Time `json:"produced_at"`

	ThisUpdate   time.Time `json:"this_update"`

	NextUpdate   time.Time `json:"next_update"`

	ResponderID  string    `json:"responder_id"`

	SignatureAlg string    `json:"signature_algorithm"`

}



// MethodFailure represents a failed revocation check method.

type MethodFailure struct {

	Method RevocationMethod `json:"method"`

	URL    string           `json:"url"`

	Error  string           `json:"error"`

}



// NewRevocationChecker creates a new revocation checker.

func NewRevocationChecker(config *RevocationConfig, logger *logging.StructuredLogger) (*RevocationChecker, error) {

	if config == nil {

		return nil, fmt.Errorf("revocation config is required")

	}



	// Set defaults.

	if config.Timeout == 0 {

		config.Timeout = 30 * time.Second

	}

	if config.CacheTTL == 0 {

		config.CacheTTL = 1 * time.Hour

	}

	if config.CacheSize == 0 {

		config.CacheSize = 1000

	}

	if config.MaxCRLSize == 0 {

		config.MaxCRLSize = 50 * 1024 * 1024 // 50MB

	}

	if config.RetryAttempts == 0 {

		config.RetryAttempts = 3

	}

	if config.RetryDelay == 0 {

		config.RetryDelay = 1 * time.Second

	}



	checker := &RevocationChecker{

		config: config,

		logger: logger,

		httpClient: &http.Client{

			Timeout: config.Timeout,

		},

	}



	// Initialize CRL cache.

	if config.CRLCheckEnabled {

		checker.crlCache = &CRLCache{

			cache:   make(map[string]*CRLEntry),

			ttl:     config.CacheTTL,

			maxSize: config.CacheSize,

		}

	}



	// Initialize OCSP cache.

	if config.OCSPCheckEnabled {

		checker.ocspCache = &OCSPCache{

			cache:   make(map[string]*OCSPEntry),

			ttl:     config.CacheTTL,

			maxSize: config.CacheSize,

		}

	}



	return checker, nil

}



// CheckRevocation performs comprehensive revocation checking.

func (rc *RevocationChecker) CheckRevocation(ctx context.Context, cert *x509.Certificate) (RevocationStatus, error) {

	result, err := rc.CheckRevocationDetailed(ctx, cert)

	if err != nil {

		return RevocationStatusUnknown, err

	}

	return result.Status, nil

}



// CheckRevocationDetailed performs detailed revocation checking.

func (rc *RevocationChecker) CheckRevocationDetailed(ctx context.Context, cert *x509.Certificate) (*RevocationCheckResult, error) {

	start := time.Now()

	serialNumber := cert.SerialNumber.String()



	result := &RevocationCheckResult{

		SerialNumber: serialNumber,

		Status:       RevocationStatusUnknown,

		CheckTime:    start,

		CacheHit:     false,

		Details: &RevocationCheckDetails{

			MethodsFailed: []MethodFailure{},

		},

	}



	rc.logger.Debug("starting revocation check",

		"serial_number", serialNumber,

		"subject", cert.Subject.String())



	// Extract revocation endpoints from certificate.

	rc.extractRevocationEndpoints(cert, result.Details)



	// Determine check strategy.

	if rc.config.ParallelCheck && rc.config.CRLCheckEnabled && rc.config.OCSPCheckEnabled {

		return rc.performParallelCheck(ctx, cert, result)

	} else {

		return rc.performSequentialCheck(ctx, cert, result)

	}

}



// Extract revocation endpoints from certificate.

func (rc *RevocationChecker) extractRevocationEndpoints(cert *x509.Certificate, details *RevocationCheckDetails) {

	// Extract CRL Distribution Points.

	details.CRLDistributionPoints = cert.CRLDistributionPoints



	// Extract OCSP servers.

	details.OCSPServers = cert.OCSPServer



	// Extract issuer information.

	details.IssuerInfo = &IssuerInfo{

		Subject: cert.Issuer.String(),

		Hash:    fmt.Sprintf("%x", cert.AuthorityKeyId),

	}

}



// Perform parallel revocation checking.

func (rc *RevocationChecker) performParallelCheck(ctx context.Context, cert *x509.Certificate, result *RevocationCheckResult) (*RevocationCheckResult, error) {

	var wg sync.WaitGroup

	resultsChan := make(chan *RevocationCheckResult, 2)



	// Launch CRL check.

	if rc.config.CRLCheckEnabled && len(result.Details.CRLDistributionPoints) > 0 {

		wg.Add(1)

		go func() {

			defer wg.Done()

			crlResult := rc.checkCRL(ctx, cert)

			crlResult.Method = RevocationMethodCRL

			resultsChan <- crlResult

		}()

	}



	// Launch OCSP check.

	if rc.config.OCSPCheckEnabled && len(result.Details.OCSPServers) > 0 {

		wg.Add(1)

		go func() {

			defer wg.Done()

			ocspResult := rc.checkOCSP(ctx, cert)

			ocspResult.Method = RevocationMethodOCSP

			resultsChan <- ocspResult

		}()

	}



	// Wait for results.

	go func() {

		wg.Wait()

		close(resultsChan)

	}()



	// Process results.

	var bestResult *RevocationCheckResult

	for checkResult := range resultsChan {

		if checkResult.Status == RevocationStatusRevoked {

			// If revoked, return immediately.

			result.Status = RevocationStatusRevoked

			result.Method = checkResult.Method

			result.RevokedAt = checkResult.RevokedAt

			result.RevocationReason = checkResult.RevocationReason

			result.ResponseSource = checkResult.ResponseSource

			result.CacheHit = checkResult.CacheHit

			break

		} else if checkResult.Status == RevocationStatusGood {

			// Keep the good result.

			bestResult = checkResult

		}



		// Merge errors and warnings.

		result.Errors = append(result.Errors, checkResult.Errors...)

		result.Warnings = append(result.Warnings, checkResult.Warnings...)

	}



	if result.Status == RevocationStatusUnknown && bestResult != nil {

		result.Status = bestResult.Status

		result.Method = RevocationMethodHybrid

		result.ResponseSource = bestResult.ResponseSource

		result.CacheHit = bestResult.CacheHit

		result.NextUpdate = bestResult.NextUpdate

	}



	result.CheckDuration = time.Since(result.CheckTime)

	return result, nil

}



// Perform sequential revocation checking.

func (rc *RevocationChecker) performSequentialCheck(ctx context.Context, cert *x509.Certificate, result *RevocationCheckResult) (*RevocationCheckResult, error) {

	// Try OCSP first (faster).

	if rc.config.OCSPCheckEnabled && len(result.Details.OCSPServers) > 0 {

		ocspResult := rc.checkOCSP(ctx, cert)

		if ocspResult.Status != RevocationStatusUnknown {

			result.Status = ocspResult.Status

			result.Method = RevocationMethodOCSP

			result.RevokedAt = ocspResult.RevokedAt

			result.RevocationReason = ocspResult.RevocationReason

			result.ResponseSource = ocspResult.ResponseSource

			result.CacheHit = ocspResult.CacheHit

			result.NextUpdate = ocspResult.NextUpdate

			result.CheckDuration = time.Since(result.CheckTime)

			return result, nil

		}



		result.Errors = append(result.Errors, ocspResult.Errors...)

		result.Warnings = append(result.Warnings, ocspResult.Warnings...)



		// Add to failed methods if CRL fallback is disabled.

		if !rc.config.CRLFallback {

			result.Details.MethodsFailed = append(result.Details.MethodsFailed, MethodFailure{

				Method: RevocationMethodOCSP,

				Error:  "OCSP check failed",

			})

		}

	}



	// Try CRL if OCSP failed or is not available.

	if rc.config.CRLCheckEnabled && len(result.Details.CRLDistributionPoints) > 0 {

		crlResult := rc.checkCRL(ctx, cert)

		if crlResult.Status != RevocationStatusUnknown {

			result.Status = crlResult.Status

			result.Method = RevocationMethodCRL

			result.RevokedAt = crlResult.RevokedAt

			result.RevocationReason = crlResult.RevocationReason

			result.ResponseSource = crlResult.ResponseSource

			result.CacheHit = crlResult.CacheHit

			result.NextUpdate = crlResult.NextUpdate

			result.CheckDuration = time.Since(result.CheckTime)

			return result, nil

		}



		result.Errors = append(result.Errors, crlResult.Errors...)

		result.Warnings = append(result.Warnings, crlResult.Warnings...)



		result.Details.MethodsFailed = append(result.Details.MethodsFailed, MethodFailure{

			Method: RevocationMethodCRL,

			Error:  "CRL check failed",

		})

	}



	result.CheckDuration = time.Since(result.CheckTime)

	return result, nil

}



// Check certificate revocation via CRL.

func (rc *RevocationChecker) checkCRL(ctx context.Context, cert *x509.Certificate) *RevocationCheckResult {

	result := &RevocationCheckResult{

		SerialNumber: cert.SerialNumber.String(),

		Status:       RevocationStatusUnknown,

		CheckTime:    time.Now(),

		Method:       RevocationMethodCRL,

		Errors:       []string{},

		Warnings:     []string{},

	}



	for _, crlURL := range cert.CRLDistributionPoints {

		crlEntry, err := rc.getCRL(ctx, crlURL)

		if err != nil {

			result.Errors = append(result.Errors, fmt.Sprintf("CRL fetch failed for %s: %v", crlURL, err))

			continue

		}



		// Check if certificate is revoked.

		for _, revokedCert := range crlEntry.CRL.TBSCertList.RevokedCertificates {

			if revokedCert.SerialNumber.Cmp(cert.SerialNumber) == 0 {

				result.Status = RevocationStatusRevoked

				result.RevokedAt = &revokedCert.RevocationTime



				// Extract revocation reason if present.

				for _, ext := range revokedCert.Extensions {

					if ext.Id.Equal([]int{2, 5, 29, 21}) { // reasonCode extension

						if len(ext.Value) > 0 {

							reason := int(ext.Value[0])

							result.RevocationReason = &reason

						}

					}

				}



				result.ResponseSource = crlURL

				result.NextUpdate = &crlEntry.NextUpdate



				// Add CRL info to details.

				if result.Details == nil {

					result.Details = &RevocationCheckDetails{}

				}

				result.Details.CRLInfo = &CRLInfo{

					URL:          crlURL,

					LastModified: crlEntry.LastModified,

					NextUpdate:   crlEntry.NextUpdate,

					EntryCount:   len(crlEntry.CRL.TBSCertList.RevokedCertificates),

					Size:         crlEntry.Size,

					Version:      crlEntry.CRL.TBSCertList.Version,

				}



				return result

			}

		}



		// Certificate not found in CRL - it's good.

		result.Status = RevocationStatusGood

		result.ResponseSource = crlURL

		result.NextUpdate = &crlEntry.NextUpdate



		// Add CRL info to details.

		if result.Details == nil {

			result.Details = &RevocationCheckDetails{}

		}

		result.Details.CRLInfo = &CRLInfo{

			URL:          crlURL,

			LastModified: crlEntry.LastModified,

			NextUpdate:   crlEntry.NextUpdate,

			EntryCount:   len(crlEntry.CRL.TBSCertList.RevokedCertificates),

			Size:         crlEntry.Size,

			Version:      crlEntry.CRL.TBSCertList.Version,

		}



		break // Use first successful CRL

	}



	return result

}



// Check certificate revocation via OCSP.

func (rc *RevocationChecker) checkOCSP(ctx context.Context, cert *x509.Certificate) *RevocationCheckResult {

	result := &RevocationCheckResult{

		SerialNumber: cert.SerialNumber.String(),

		Status:       RevocationStatusUnknown,

		CheckTime:    time.Now(),

		Method:       RevocationMethodOCSP,

		Errors:       []string{},

		Warnings:     []string{},

	}



	// Need issuer certificate for OCSP request.

	// This is a simplified implementation - real implementation would need the issuer certificate.

	if len(cert.OCSPServer) == 0 {

		result.Errors = append(result.Errors, "no OCSP servers found in certificate")

		return result

	}



	for _, ocspURL := range cert.OCSPServer {

		// Check cache first.

		cacheKey := fmt.Sprintf("%s-%x", cert.SerialNumber.String(), cert.AuthorityKeyId)

		if cached := rc.getOCSPCache(cacheKey); cached != nil {

			// Convert integer OCSP status to RevocationStatus.

			switch cached.Status {

			case ocsp.Good:

				result.Status = RevocationStatusGood

			case ocsp.Revoked:

				result.Status = RevocationStatusRevoked

			default:

				result.Status = RevocationStatusUnknown

			}

			result.ResponseSource = ocspURL

			result.CacheHit = true

			result.NextUpdate = &cached.NextUpdate



			if cached.Status == ocsp.Revoked && cached.Response != nil {

				result.RevokedAt = &cached.Response.RevokedAt

				result.RevocationReason = &cached.Response.RevocationReason

			}



			return result

		}



		// Perform OCSP request.

		// This is a placeholder - real implementation would construct and send OCSP request.

		ocspResp, err := rc.performOCSPRequest(ctx, ocspURL, cert)

		if err != nil {

			result.Errors = append(result.Errors, fmt.Sprintf("OCSP request failed for %s: %v", ocspURL, err))

			continue

		}



		// Process OCSP response.

		switch ocspResp.Status {

		case ocsp.Good:

			result.Status = RevocationStatusGood

		case ocsp.Revoked:

			result.Status = RevocationStatusRevoked

			result.RevokedAt = &ocspResp.RevokedAt

			result.RevocationReason = &ocspResp.RevocationReason

		default:

			result.Status = RevocationStatusUnknown

		}



		result.ResponseSource = ocspURL

		result.NextUpdate = &ocspResp.NextUpdate



		// Cache the response - map the status to OCSP int value.

		var statusInt int

		switch result.Status {

		case RevocationStatusValid, RevocationStatusGood:

			statusInt = ocsp.Good

		case RevocationStatusRevoked:

			statusInt = ocsp.Revoked

		default:

			statusInt = ocsp.Unknown

		}

		rc.cacheOCSPResponse(cacheKey, &OCSPEntry{

			SerialNumber: cert.SerialNumber.String(),

			Status:       statusInt,

			Response:     ocspResp,

			NextUpdate:   ocspResp.NextUpdate,

			CachedAt:     time.Now(),

		})



		// Add OCSP info to details.

		if result.Details == nil {

			result.Details = &RevocationCheckDetails{}

		}

		result.Details.OCSPInfo = &OCSPInfo{

			ResponderURL: ocspURL,

			ProducedAt:   ocspResp.ProducedAt,

			ThisUpdate:   ocspResp.ThisUpdate,

			NextUpdate:   ocspResp.NextUpdate,

			SignatureAlg: ocspResp.SignatureAlgorithm.String(),

		}



		break // Use first successful OCSP response

	}



	return result

}



// Get CRL from cache or fetch from URL.

func (rc *RevocationChecker) getCRL(ctx context.Context, crlURL string) (*CRLEntry, error) {

	// Check cache first.

	if cached := rc.getCRLCache(crlURL); cached != nil {

		// Check if CRL is still valid.

		if time.Now().Before(cached.NextUpdate) {

			return cached, nil

		}

	}



	// Fetch CRL from URL.

	return rc.fetchCRL(ctx, crlURL)

}



// Fetch CRL from URL.

func (rc *RevocationChecker) fetchCRL(ctx context.Context, crlURL string) (*CRLEntry, error) {

	req, err := http.NewRequestWithContext(ctx, "GET", crlURL, http.NoBody)

	if err != nil {

		return nil, fmt.Errorf("failed to create CRL request: %w", err)

	}



	resp, err := rc.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("CRL request failed: %w", err)

	}

	defer resp.Body.Close()



	if resp.StatusCode != http.StatusOK {

		return nil, fmt.Errorf("CRL server returned status %d", resp.StatusCode)

	}



	// Check content length.

	if resp.ContentLength > rc.config.MaxCRLSize {

		return nil, fmt.Errorf("CRL size (%d bytes) exceeds maximum (%d bytes)", resp.ContentLength, rc.config.MaxCRLSize)

	}



	// Read CRL data.

	crlData, err := io.ReadAll(resp.Body)

	if err != nil {

		return nil, fmt.Errorf("failed to read CRL data: %w", err)

	}



	// Parse CRL.

	crl, err := x509.ParseCRL(crlData)

	if err != nil {

		return nil, fmt.Errorf("failed to parse CRL: %w", err)

	}



	entry := &CRLEntry{

		URL:          crlURL,

		CRL:          crl,

		LastModified: time.Now(), // Would use Last-Modified header if present

		NextUpdate:   crl.TBSCertList.NextUpdate,

		CachedAt:     time.Now(),

		Size:         int64(len(crlData)),

		EntryCount:   len(crl.TBSCertList.RevokedCertificates),

	}



	// Cache the CRL.

	rc.cacheCRL(crlURL, entry)



	return entry, nil

}



// Perform OCSP request.

func (rc *RevocationChecker) performOCSPRequest(ctx context.Context, ocspURL string, cert *x509.Certificate) (*ocsp.Response, error) {

	// This is a placeholder - real implementation would:.

	// 1. Get the issuer certificate.

	// 2. Create OCSP request.

	// 3. Send HTTP POST to OCSP responder.

	// 4. Parse OCSP response.



	return nil, fmt.Errorf("OCSP request not implemented")

}



// Cache management methods.



func (rc *RevocationChecker) getCRLCache(url string) *CRLEntry {

	if rc.crlCache == nil {

		return nil

	}



	rc.crlCache.mu.RLock()

	defer rc.crlCache.mu.RUnlock()



	entry, exists := rc.crlCache.cache[url]

	if !exists || time.Since(entry.CachedAt) > rc.crlCache.ttl {

		return nil

	}



	return entry

}



func (rc *RevocationChecker) cacheCRL(url string, entry *CRLEntry) {

	if rc.crlCache == nil {

		return

	}



	rc.crlCache.mu.Lock()

	defer rc.crlCache.mu.Unlock()



	// Check cache size and evict if necessary.

	if len(rc.crlCache.cache) >= rc.crlCache.maxSize {

		rc.evictOldestCRL()

	}



	rc.crlCache.cache[url] = entry

}



func (rc *RevocationChecker) evictOldestCRL() {

	var oldestURL string

	var oldestTime time.Time



	for url, entry := range rc.crlCache.cache {

		if oldestURL == "" || entry.CachedAt.Before(oldestTime) {

			oldestURL = url

			oldestTime = entry.CachedAt

		}

	}



	if oldestURL != "" {

		delete(rc.crlCache.cache, oldestURL)

	}

}



func (rc *RevocationChecker) getOCSPCache(key string) *OCSPEntry {

	if rc.ocspCache == nil {

		return nil

	}



	rc.ocspCache.mu.RLock()

	defer rc.ocspCache.mu.RUnlock()



	entry, exists := rc.ocspCache.cache[key]

	if !exists || time.Since(entry.CachedAt) > rc.ocspCache.ttl {

		return nil

	}



	return entry

}



func (rc *RevocationChecker) cacheOCSPResponse(key string, entry *OCSPEntry) {

	if rc.ocspCache == nil {

		return

	}



	rc.ocspCache.mu.Lock()

	defer rc.ocspCache.mu.Unlock()



	// Check cache size and evict if necessary.

	if len(rc.ocspCache.cache) >= rc.ocspCache.maxSize {

		rc.evictOldestOCSP()

	}



	rc.ocspCache.cache[key] = entry

}



func (rc *RevocationChecker) evictOldestOCSP() {

	var oldestKey string

	var oldestTime time.Time



	for key, entry := range rc.ocspCache.cache {

		if oldestKey == "" || entry.CachedAt.Before(oldestTime) {

			oldestKey = key

			oldestTime = entry.CachedAt

		}

	}



	if oldestKey != "" {

		delete(rc.ocspCache.cache, oldestKey)

	}

}



// GetCacheStats returns cache statistics.

func (rc *RevocationChecker) GetCacheStats() map[string]interface{} {

	stats := make(map[string]interface{})



	if rc.crlCache != nil {

		rc.crlCache.mu.RLock()

		stats["crl_cache"] = map[string]interface{}{

			"entries":  len(rc.crlCache.cache),

			"max_size": rc.crlCache.maxSize,

			"ttl":      rc.crlCache.ttl.String(),

		}

		rc.crlCache.mu.RUnlock()

	}



	if rc.ocspCache != nil {

		rc.ocspCache.mu.RLock()

		stats["ocsp_cache"] = map[string]interface{}{

			"entries":  len(rc.ocspCache.cache),

			"max_size": rc.ocspCache.maxSize,

			"ttl":      rc.ocspCache.ttl.String(),

		}

		rc.ocspCache.mu.RUnlock()

	}



	return stats

}



// ClearCache clears all caches.

func (rc *RevocationChecker) ClearCache() {

	if rc.crlCache != nil {

		rc.crlCache.mu.Lock()

		rc.crlCache.cache = make(map[string]*CRLEntry)

		rc.crlCache.mu.Unlock()

	}



	if rc.ocspCache != nil {

		rc.ocspCache.mu.Lock()

		rc.ocspCache.cache = make(map[string]*OCSPEntry)

		rc.ocspCache.mu.Unlock()

	}



	rc.logger.Info("revocation caches cleared")

}

