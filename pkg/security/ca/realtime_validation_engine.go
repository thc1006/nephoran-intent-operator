package ca

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// RealtimeValidationEngine provides real-time certificate validation during TLS handshakes
type RealtimeValidationEngine struct {
	config            *RealtimeValidationConfig
	logger            *logging.StructuredLogger
	validator         *ValidationFramework
	revocationChecker *RevocationChecker
	policyEngine      *PolicyEngine
	cache             *ValidationCacheOptimized
	circuitBreaker    *ValidationCircuitBreaker
	connectionPool    *OCSPConnectionPool
	metricsRecorder   *ValidationMetricsRecorder
	webhookNotifier   *ValidationWebhookNotifier
	emergencyBypass   *EmergencyBypassController

	// Statistics
	stats *ValidationStatistics

	// Control
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	mu     sync.RWMutex
}

// RealtimeValidationConfig configures real-time validation
type RealtimeValidationConfig struct {
	// Core validation settings
	Enabled                  bool          `yaml:"enabled"`
	ValidationTimeout        time.Duration `yaml:"validation_timeout"`
	MaxConcurrentValidations int           `yaml:"max_concurrent_validations"`

	// Chain validation
	ChainValidationEnabled   bool   `yaml:"chain_validation_enabled"`
	IntermediateCAValidation bool   `yaml:"intermediate_ca_validation"`
	MaxChainDepth            int    `yaml:"max_chain_depth"`
	TrustedRootStore         string `yaml:"trusted_root_store"`

	// Certificate Transparency
	CTLogValidationEnabled bool          `yaml:"ct_log_validation_enabled"`
	CTLogEndpoints         []string      `yaml:"ct_log_endpoints"`
	CTLogTimeout           time.Duration `yaml:"ct_log_timeout"`
	RequireSCTValidation   bool          `yaml:"require_sct_validation"`

	// Revocation checking
	CRLValidationEnabled  bool          `yaml:"crl_validation_enabled"`
	OCSPValidationEnabled bool          `yaml:"ocsp_validation_enabled"`
	OCSPStaplingEnabled   bool          `yaml:"ocsp_stapling_enabled"`
	RevocationCacheSize   int           `yaml:"revocation_cache_size"`
	RevocationCacheTTL    time.Duration `yaml:"revocation_cache_ttl"`
	SoftFailRevocation    bool          `yaml:"soft_fail_revocation"`
	HardFailRevocation    bool          `yaml:"hard_fail_revocation"`

	// Performance optimization
	AsyncValidationEnabled bool          `yaml:"async_validation_enabled"`
	BatchValidationEnabled bool          `yaml:"batch_validation_enabled"`
	BatchSize              int           `yaml:"batch_size"`
	BatchTimeout           time.Duration `yaml:"batch_timeout"`
	ConnectionPoolSize     int           `yaml:"connection_pool_size"`
	CircuitBreakerEnabled  bool          `yaml:"circuit_breaker_enabled"`

	// Security policies
	PolicyValidationEnabled   bool     `yaml:"policy_validation_enabled"`
	CertificatePinningEnabled bool     `yaml:"certificate_pinning_enabled"`
	AlgorithmStrengthCheck    bool     `yaml:"algorithm_strength_check"`
	MinimumRSAKeySize         int      `yaml:"minimum_rsa_key_size"`
	AllowedECCurves           []string `yaml:"allowed_ec_curves"`
	RequireExtendedValidation bool     `yaml:"require_extended_validation"`

	// O-RAN compliance
	ORANComplianceEnabled     bool         `yaml:"oran_compliance_enabled"`
	ORANPolicyRules           []PolicyRule `yaml:"oran_policy_rules"`
	TelecomSpecificValidation bool         `yaml:"telecom_specific_validation"`

	// Monitoring and alerting
	MetricsEnabled       bool            `yaml:"metrics_enabled"`
	DetailedMetrics      bool            `yaml:"detailed_metrics"`
	WebhookNotifications bool            `yaml:"webhook_notifications"`
	WebhookEndpoints     []string        `yaml:"webhook_endpoints"`
	AlertThresholds      ValidationAlertThresholds `yaml:"alert_thresholds"`

	// Emergency procedures
	EmergencyBypassEnabled  bool          `yaml:"emergency_bypass_enabled"`
	BypassAuthorizationKeys []string      `yaml:"bypass_authorization_keys"`
	BypassDuration          time.Duration `yaml:"bypass_duration"`
	BypassAuditEnabled      bool          `yaml:"bypass_audit_enabled"`
}

// AlertThresholds defines thresholds for validation alerts
type ValidationAlertThresholds struct {
	ValidationFailureRate      float64       `yaml:"validation_failure_rate"`
	RevocationCheckFailureRate float64       `yaml:"revocation_check_failure_rate"`
	ResponseTimeP95            time.Duration `yaml:"response_time_p95"`
	CacheHitRateMin            float64       `yaml:"cache_hit_rate_min"`
}

// ValidationStatistics tracks validation statistics
type ValidationStatistics struct {
	TotalValidations      atomic.Uint64
	SuccessfulValidations atomic.Uint64
	FailedValidations     atomic.Uint64
	CacheHits             atomic.Uint64
	CacheMisses           atomic.Uint64
	RevocationChecks      atomic.Uint64
	CTLogChecks           atomic.Uint64
	PolicyViolations      atomic.Uint64
	EmergencyBypasses     atomic.Uint64

	// Performance metrics
	ValidationDurations []time.Duration
	mu                  sync.RWMutex
}

// ValidationCacheOptimized provides optimized caching for validation results
type ValidationCacheOptimized struct {
	l1Cache      *L1Cache                  // In-memory hot cache
	l2Cache      *L2Cache                  // Distributed cache
	preloadCache *PreProvisioningCache     // Pre-validated certificates
	stats        *CacheStatistics
	mu           sync.RWMutex
}

// L1ValidationCache is a fast in-memory cache
type L1ValidationCache struct {
	cache   map[string]*CachedValidationResult
	lru     *LRUEvictionPolicy
	maxSize int
	ttl     time.Duration
	mu      sync.RWMutex
}

// CachedValidationResult represents a cached validation result
type CachedValidationResult struct {
	Result           *ValidationResult
	RevocationStatus *RevocationCheckResult
	PolicyResult     *PolicyValidationResult
	CachedAt         time.Time
	TTL              time.Duration
	HitCount         atomic.Uint32
}

// ValidationCircuitBreaker provides circuit breaker pattern for validation
type ValidationCircuitBreaker struct {
	state           atomic.Value // CircuitState
	failureCount    atomic.Uint32
	successCount    atomic.Uint32
	lastFailureTime atomic.Value // time.Time
	config          *CircuitBreakerConfig
	mu              sync.RWMutex
}

// CircuitBreakerConfig configures the circuit breaker
type CircuitBreakerConfig struct {
	FailureThreshold uint32        `yaml:"failure_threshold"`
	SuccessThreshold uint32        `yaml:"success_threshold"`
	Timeout          time.Duration `yaml:"timeout"`
	HalfOpenRequests uint32        `yaml:"half_open_requests"`
}

// CircuitState represents circuit breaker states
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// OCSPConnectionPool manages OCSP responder connections
type OCSPConnectionPool struct {
	connections map[string]*PooledConnection
	maxSize     int
	timeout     time.Duration
	mu          sync.RWMutex
}

// PooledConnection represents a pooled OCSP connection
type PooledConnection struct {
	URL          string
	Client       *OCSPClient
	InUse        atomic.Bool
	LastUsed     time.Time
	RequestCount atomic.Uint64
	ErrorCount   atomic.Uint64
	ResponseTime time.Duration
}

// ValidationMetricsRecorder records validation metrics
type ValidationMetricsRecorder struct {
	validationTotal      *prometheus.CounterVec
	validationDuration   *prometheus.HistogramVec
	revocationCheckTotal *prometheus.CounterVec
	cacheHitRate         prometheus.Gauge
	policyViolations     *prometheus.CounterVec
	circuitBreakerState  *prometheus.GaugeVec
}

// ValidationWebhookNotifier sends webhook notifications for validation events
type ValidationWebhookNotifier struct {
	endpoints  []string
	client     *WebhookClient
	eventQueue chan *ValidationEvent
	workers    int
	mu         sync.RWMutex
}

// ValidationEvent represents a validation event for notification
type ValidationEvent struct {
	Type        string                 `json:"type"`
	Timestamp   time.Time              `json:"timestamp"`
	Certificate *CertificateInfo       `json:"certificate"`
	Result      *ValidationResult      `json:"result"`
	Severity    string                 `json:"severity"`
	Details     map[string]interface{} `json:"details"`
}

// EmergencyBypassController manages emergency bypass procedures
type EmergencyBypassController struct {
	enabled      atomic.Bool
	bypassUntil  atomic.Value // time.Time
	authorizedBy string
	reason       string
	auditLogger  *AuditLogger
	mu           sync.RWMutex
}

// NewRealtimeValidationEngine creates a new real-time validation engine
func NewRealtimeValidationEngine(
	config *RealtimeValidationConfig,
	logger *logging.StructuredLogger,
	validator *ValidationFramework,
	revocationChecker *RevocationChecker,
) (*RealtimeValidationEngine, error) {
	if config == nil {
		return nil, fmt.Errorf("validation config is required")
	}

	ctx, cancel := context.WithCancel(context.Background())

	engine := &RealtimeValidationEngine{
		config:            config,
		logger:            logger,
		validator:         validator,
		revocationChecker: revocationChecker,
		stats:             &ValidationStatistics{},
		ctx:               ctx,
		cancel:            cancel,
	}

	// Initialize policy engine
	if config.PolicyValidationEnabled {
		policyEngine, err := NewPolicyEngine(&PolicyConfig{
			Enabled:                   true,
			Rules:                     config.ORANPolicyRules,
			CertificatePinning:        config.CertificatePinningEnabled,
			AlgorithmStrengthCheck:    config.AlgorithmStrengthCheck,
			MinimumRSAKeySize:         config.MinimumRSAKeySize,
			AllowedECCurves:           config.AllowedECCurves,
			RequireExtendedValidation: config.RequireExtendedValidation,
		}, logger)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to initialize policy engine: %w", err)
		}
		engine.policyEngine = policyEngine
	}

	// Initialize optimized cache
	engine.cache = &ValidationCacheOptimized{
		l1Cache: &L1Cache{
			cache:   make(map[string]*CacheEntry),
			maxSize: config.RevocationCacheSize,
			ttl:     config.RevocationCacheTTL,
		},
		stats: &CacheStatistics{},
	}

	// Initialize circuit breaker
	if config.CircuitBreakerEnabled {
		engine.circuitBreaker = &ValidationCircuitBreaker{
			config: &CircuitBreakerConfig{
				FailureThreshold: 5,
				SuccessThreshold: 2,
				Timeout:          30 * time.Second,
				HalfOpenRequests: 3,
			},
		}
		engine.circuitBreaker.state.Store(CircuitClosed)
		engine.circuitBreaker.lastFailureTime.Store(time.Now())
	}

	// Initialize connection pool
	if config.OCSPValidationEnabled {
		engine.connectionPool = &OCSPConnectionPool{
			connections: make(map[string]*PooledConnection),
			maxSize:     config.ConnectionPoolSize,
			timeout:     config.ValidationTimeout,
		}
	}

	// Initialize metrics recorder
	if config.MetricsEnabled {
		engine.metricsRecorder = engine.initializeMetrics()
	}

	// Initialize webhook notifier
	if config.WebhookNotifications {
		engine.webhookNotifier = &ValidationWebhookNotifier{
			endpoints:  config.WebhookEndpoints,
			client:     NewWebhookClient(logger),
			eventQueue: make(chan *ValidationEvent, 1000),
			workers:    5,
		}
	}

	// Initialize emergency bypass controller
	if config.EmergencyBypassEnabled {
		engine.emergencyBypass = &EmergencyBypassController{
			auditLogger: NewAuditLogger(logger),
		}
		engine.emergencyBypass.enabled.Store(false)
	}

	return engine, nil
}

// ValidateCertificateRealtime performs real-time certificate validation
func (e *RealtimeValidationEngine) ValidateCertificateRealtime(ctx context.Context, cert *x509.Certificate, connState *tls.ConnectionState) (*ValidationResult, error) {
	start := time.Now()
	e.stats.TotalValidations.Add(1)

	// Check emergency bypass
	if e.isEmergencyBypassActive() {
		e.stats.EmergencyBypasses.Add(1)
		e.logger.Warn("certificate validation bypassed due to emergency override",
			"serial_number", cert.SerialNumber.String(),
			"subject", cert.Subject.String())
		return &ValidationResult{
			SerialNumber:   cert.SerialNumber.String(),
			Valid:          true,
			ValidationTime: start,
			Warnings:       []string{"validation bypassed due to emergency override"},
		}, nil
	}

	// Check circuit breaker
	if e.circuitBreaker != nil && !e.circuitBreaker.AllowRequest() {
		return nil, fmt.Errorf("validation circuit breaker is open")
	}

	// Check cache first
	cacheKey := e.generateCacheKey(cert)
	if cached := e.getFromCache(cacheKey); cached != nil {
		e.stats.CacheHits.Add(1)
		e.recordValidationMetrics(start, true, "cached")
		return cached.Result, nil
	}
	e.stats.CacheMisses.Add(1)

	// Create validation context with timeout
	validationCtx, cancel := context.WithTimeout(ctx, e.config.ValidationTimeout)
	defer cancel()

	// Initialize result
	result := &ValidationResult{
		SerialNumber:   cert.SerialNumber.String(),
		Valid:          true,
		ValidationTime: start,
		Errors:         []string{},
		Warnings:       []string{},
	}

	// Perform validations in parallel if configured
	if e.config.AsyncValidationEnabled {
		e.performAsyncValidation(validationCtx, cert, connState, result)
	} else {
		e.performSyncValidation(validationCtx, cert, connState, result)
	}

	// Cache the result
	e.cacheResult(cacheKey, result)

	// Record metrics
	duration := time.Since(start)
	e.recordValidationMetrics(start, result.Valid, "realtime")
	e.recordValidationDuration(duration)

	// Send webhook notification if needed
	if !result.Valid && e.webhookNotifier != nil {
		e.sendValidationEvent(cert, result, "validation_failed")
	}

	// Update circuit breaker
	if e.circuitBreaker != nil {
		if result.Valid {
			e.circuitBreaker.RecordSuccess()
		} else {
			e.circuitBreaker.RecordFailure()
		}
	}

	// Update statistics
	if result.Valid {
		e.stats.SuccessfulValidations.Add(1)
	} else {
		e.stats.FailedValidations.Add(1)
	}

	e.logger.Info("real-time certificate validation completed",
		"serial_number", cert.SerialNumber.String(),
		"valid", result.Valid,
		"duration", duration,
		"cached", false)

	return result, nil
}

// performAsyncValidation performs validations asynchronously
func (e *RealtimeValidationEngine) performAsyncValidation(ctx context.Context, cert *x509.Certificate, connState *tls.ConnectionState, result *ValidationResult) {
	var wg sync.WaitGroup
	resultChan := make(chan validationComponent, 5)

	// Chain validation
	if e.config.ChainValidationEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			chainResult := e.validateChain(ctx, cert, connState)
			resultChan <- validationComponent{Type: "chain", Result: chainResult}
		}()
	}

	// Revocation checking
	if e.config.CRLValidationEnabled || e.config.OCSPValidationEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			revocationResult := e.checkRevocation(ctx, cert)
			resultChan <- validationComponent{Type: "revocation", Result: revocationResult}
		}()
	}

	// CT log validation
	if e.config.CTLogValidationEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctResult := e.validateCTLog(ctx, cert)
			resultChan <- validationComponent{Type: "ct", Result: ctResult}
		}()
	}

	// Policy validation
	if e.policyEngine != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()
			policyResult := e.validatePolicy(ctx, cert)
			resultChan <- validationComponent{Type: "policy", Result: policyResult}
		}()
	}

	// O-RAN compliance validation
	if e.config.ORANComplianceEnabled {
		wg.Add(1)
		go func() {
			defer wg.Done()
			oranResult := e.validateORANCompliance(ctx, cert)
			resultChan <- validationComponent{Type: "oran", Result: oranResult}
		}()
	}

	// Wait for all validations to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Aggregate results
	for component := range resultChan {
		e.aggregateValidationResult(result, component)
	}
}

// performSyncValidation performs validations synchronously
func (e *RealtimeValidationEngine) performSyncValidation(ctx context.Context, cert *x509.Certificate, connState *tls.ConnectionState, result *ValidationResult) {
	// Chain validation
	if e.config.ChainValidationEnabled {
		chainResult := e.validateChain(ctx, cert, connState)
		e.aggregateValidationResult(result, validationComponent{Type: "chain", Result: chainResult})
	}

	// Revocation checking
	if e.config.CRLValidationEnabled || e.config.OCSPValidationEnabled {
		revocationResult := e.checkRevocation(ctx, cert)
		e.aggregateValidationResult(result, validationComponent{Type: "revocation", Result: revocationResult})
	}

	// CT log validation
	if e.config.CTLogValidationEnabled {
		ctResult := e.validateCTLog(ctx, cert)
		e.aggregateValidationResult(result, validationComponent{Type: "ct", Result: ctResult})
	}

	// Policy validation
	if e.policyEngine != nil {
		policyResult := e.validatePolicy(ctx, cert)
		e.aggregateValidationResult(result, validationComponent{Type: "policy", Result: policyResult})
	}

	// O-RAN compliance validation
	if e.config.ORANComplianceEnabled {
		oranResult := e.validateORANCompliance(ctx, cert)
		e.aggregateValidationResult(result, validationComponent{Type: "oran", Result: oranResult})
	}
}

// validateChain validates the certificate chain
func (e *RealtimeValidationEngine) validateChain(ctx context.Context, cert *x509.Certificate, connState *tls.ConnectionState) interface{} {
	if connState != nil && len(connState.PeerCertificates) > 0 {
		// Validate the full chain from the connection
		result, err := e.validator.validateCertificateChain(ctx, connState.PeerCertificates[0])
		if err != nil {
			return nil
		}
		return result
	}
	// Validate single certificate
	result, err := e.validator.ValidateCertificate(ctx, cert)
	if err != nil {
		return nil
	}
	return result
}

// checkRevocation performs revocation checking with optimization
func (e *RealtimeValidationEngine) checkRevocation(ctx context.Context, cert *x509.Certificate) interface{} {
	e.stats.RevocationChecks.Add(1)

	// Use connection pool for OCSP if available
	if e.connectionPool != nil && e.config.OCSPValidationEnabled {
		return e.checkRevocationWithPool(ctx, cert)
	}

	result, err := e.revocationChecker.CheckRevocationDetailed(ctx, cert)
	if err != nil {
		return nil
	}
	return result
}

// checkRevocationWithPool uses connection pooling for OCSP checks
func (e *RealtimeValidationEngine) checkRevocationWithPool(ctx context.Context, cert *x509.Certificate) *RevocationCheckResult {
	// Get OCSP responder URL
	if len(cert.OCSPServer) == 0 {
		return &RevocationCheckResult{
			Status: RevocationStatusUnknown,
			Errors: []string{"no OCSP responder URL in certificate"},
		}
	}

	ocspURL := cert.OCSPServer[0]
	conn := e.connectionPool.GetConnection(ocspURL)
	if conn == nil {
		// Create new connection
		conn = e.connectionPool.CreateConnection(ocspURL)
	}

	defer e.connectionPool.ReleaseConnection(conn)

	// Perform OCSP check using pooled connection
	return conn.CheckRevocation(ctx, cert)
}

// validateCTLog validates Certificate Transparency logs
func (e *RealtimeValidationEngine) validateCTLog(ctx context.Context, cert *x509.Certificate) interface{} {
	e.stats.CTLogChecks.Add(1)

	// Check for embedded SCTs if stapling is required
	if e.config.RequireSCTValidation {
		return e.validateSCTs(ctx, cert)
	}

	result, err := e.validator.validateCTLog(ctx, cert)
	if err != nil {
		return nil
	}
	return result
}

// validatePolicy validates against security policies
func (e *RealtimeValidationEngine) validatePolicy(ctx context.Context, cert *x509.Certificate) interface{} {
	result := e.policyEngine.ValidateCertificate(ctx, cert)

	if !result.Valid {
		e.stats.PolicyViolations.Add(1)
	}

	return result
}

// validateORANCompliance validates O-RAN compliance
func (e *RealtimeValidationEngine) validateORANCompliance(ctx context.Context, cert *x509.Certificate) interface{} {
	// O-RAN specific validation rules
	result := &ORANComplianceResult{
		Valid:  true,
		Checks: []ORANCheck{},
		Score:  100.0,
	}

	// Check for required O-RAN extensions
	e.checkORANExtensions(cert, result)

	// Validate key usage for O-RAN components
	e.checkORANKeyUsage(cert, result)

	// Validate subject naming conventions
	e.checkORANNaming(cert, result)

	// Check algorithm compliance
	e.checkORANAlgorithms(cert, result)

	return result
}

// Helper validation methods

func (e *RealtimeValidationEngine) checkORANExtensions(cert *x509.Certificate, result *ORANComplianceResult) {
	check := ORANCheck{
		Name:   "O-RAN Extensions",
		Passed: true,
	}

	// Check for O-RAN specific extensions
	requiredExtensions := []string{
		"1.3.6.1.4.1.53148.1.1", // O-RAN component identifier
		"1.3.6.1.4.1.53148.1.2", // O-RAN role identifier
	}

	for _, oid := range requiredExtensions {
		found := false
		for _, ext := range cert.Extensions {
			if ext.Id.String() == oid {
				found = true
				break
			}
		}
		if !found {
			check.Passed = false
			check.Message = fmt.Sprintf("missing required O-RAN extension: %s", oid)
			result.Score -= 10.0
		}
	}

	result.Checks = append(result.Checks, check)
	if !check.Passed {
		result.Valid = false
	}
}

func (e *RealtimeValidationEngine) checkORANKeyUsage(cert *x509.Certificate, result *ORANComplianceResult) {
	check := ORANCheck{
		Name:   "O-RAN Key Usage",
		Passed: true,
	}

	// Validate key usage based on O-RAN component type
	if cert.KeyUsage&x509.KeyUsageDigitalSignature == 0 {
		check.Passed = false
		check.Message = "digital signature key usage required for O-RAN components"
		result.Score -= 15.0
	}

	if cert.IsCA && cert.KeyUsage&x509.KeyUsageCertSign == 0 {
		check.Passed = false
		check.Message = "cert sign key usage required for O-RAN CA certificates"
		result.Score -= 20.0
	}

	result.Checks = append(result.Checks, check)
	if !check.Passed {
		result.Valid = false
	}
}

func (e *RealtimeValidationEngine) checkORANNaming(cert *x509.Certificate, result *ORANComplianceResult) {
	check := ORANCheck{
		Name:   "O-RAN Naming Convention",
		Passed: true,
	}

	// Check subject naming convention
	subject := cert.Subject.String()
	if !e.validateORANSubjectFormat(subject) {
		check.Passed = false
		check.Message = "subject does not follow O-RAN naming convention"
		result.Score -= 5.0
	}

	result.Checks = append(result.Checks, check)
}

func (e *RealtimeValidationEngine) checkORANAlgorithms(cert *x509.Certificate, result *ORANComplianceResult) {
	check := ORANCheck{
		Name:   "O-RAN Algorithm Compliance",
		Passed: true,
	}

	// Check signature algorithm
	allowedAlgorithms := []x509.SignatureAlgorithm{
		x509.SHA256WithRSA,
		x509.SHA384WithRSA,
		x509.SHA512WithRSA,
		x509.ECDSAWithSHA256,
		x509.ECDSAWithSHA384,
		x509.ECDSAWithSHA512,
	}

	allowed := false
	for _, alg := range allowedAlgorithms {
		if cert.SignatureAlgorithm == alg {
			allowed = true
			break
		}
	}

	if !allowed {
		check.Passed = false
		check.Message = fmt.Sprintf("signature algorithm %v not allowed for O-RAN", cert.SignatureAlgorithm)
		result.Score -= 25.0
		result.Valid = false
	}

	result.Checks = append(result.Checks, check)
}

func (e *RealtimeValidationEngine) validateORANSubjectFormat(subject string) bool {
	// Simplified O-RAN subject format validation
	// Real implementation would check against O-RAN specifications
	return true
}

// Cache management

func (e *RealtimeValidationEngine) generateCacheKey(cert *x509.Certificate) string {
	return fmt.Sprintf("%s-%x", cert.SerialNumber.String(), cert.AuthorityKeyId)
}

func (e *RealtimeValidationEngine) getFromCache(key string) *CachedValidationResult {
	if e.cache == nil {
		return nil
	}

	// Check L1 cache first
	e.cache.l1Cache.mu.RLock()
	if entry, exists := e.cache.l1Cache.cache[key]; exists && entry != nil {
		// Check if cache entry is still valid
		if time.Since(entry.CreatedAt) <= e.cache.l1Cache.ttl {
			entry.AccessCount++
			entry.AccessedAt = time.Now()
			if cached, ok := entry.Value.(*CachedValidationResult); ok {
				cached.HitCount.Add(1)
				e.cache.l1Cache.mu.RUnlock()
				return cached
			}
		}
	}
	e.cache.l1Cache.mu.RUnlock()

	// Check L2 cache if available
	if e.cache.l2Cache != nil {
		e.cache.l2Cache.mu.RLock()
		if entry, exists := e.cache.l2Cache.cache[key]; exists && entry != nil {
			// Check if cache entry is still valid
			if time.Since(entry.CreatedAt) <= e.cache.l2Cache.ttl {
				entry.AccessCount++
				entry.AccessedAt = time.Now()
				if cached, ok := entry.Value.(*CachedValidationResult); ok {
					e.cache.l2Cache.mu.RUnlock()
					return cached
				}
			}
		}
		e.cache.l2Cache.mu.RUnlock()
	}

	return nil
}

func (e *RealtimeValidationEngine) cacheResult(key string, result *ValidationResult) {
	if e.cache == nil {
		return
	}

	cached := &CachedValidationResult{
		Result:   result,
		CachedAt: time.Now(),
		TTL:      e.config.RevocationCacheTTL,
	}

	// Store in L1 cache
	e.cache.l1Cache.mu.Lock()
	// Remove oldest if at capacity
	if len(e.cache.l1Cache.cache) >= e.cache.l1Cache.maxSize && e.cache.l1Cache.order != nil {
		oldestKey := e.cache.l1Cache.order[0]
		delete(e.cache.l1Cache.cache, oldestKey)
		e.cache.l1Cache.order = e.cache.l1Cache.order[1:]
	}
	entry := &CacheEntry{
		Key:         key,
		Value:       cached,
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 1,
	}
	e.cache.l1Cache.cache[key] = entry
	e.cache.l1Cache.order = append(e.cache.l1Cache.order, key)
	e.cache.l1Cache.mu.Unlock()

	// Store in L2 cache if available
	if e.cache.l2Cache != nil {
		e.cache.l2Cache.mu.Lock()
		// Simple put without ordering for L2 cache
		entry := &CacheEntry{
			Key:         key,
			Value:       cached,
			CreatedAt:   time.Now(),
			AccessedAt:  time.Now(),
			AccessCount: 1,
		}
		e.cache.l2Cache.cache[key] = entry
		e.cache.l2Cache.mu.Unlock()
	}
}

// Circuit breaker methods

func (cb *ValidationCircuitBreaker) AllowRequest() bool {
	state := cb.state.Load().(CircuitState)

	switch state {
	case CircuitClosed:
		return true
	case CircuitOpen:
		// Check if timeout has passed
		lastFailure := cb.lastFailureTime.Load().(time.Time)
		if time.Since(lastFailure) > cb.config.Timeout {
			cb.state.Store(CircuitHalfOpen)
			return true
		}
		return false
	case CircuitHalfOpen:
		// Allow limited requests
		return cb.successCount.Load() < cb.config.HalfOpenRequests
	default:
		return false
	}
}

func (cb *ValidationCircuitBreaker) RecordSuccess() {
	cb.successCount.Add(1)
	state := cb.state.Load().(CircuitState)

	if state == CircuitHalfOpen && cb.successCount.Load() >= cb.config.SuccessThreshold {
		cb.state.Store(CircuitClosed)
		cb.failureCount.Store(0)
		cb.successCount.Store(0)
	}
}

func (cb *ValidationCircuitBreaker) RecordFailure() {
	cb.failureCount.Add(1)
	cb.lastFailureTime.Store(time.Now())

	if cb.failureCount.Load() >= cb.config.FailureThreshold {
		cb.state.Store(CircuitOpen)
	}
}

// Connection pool methods

func (pool *OCSPConnectionPool) GetConnection(url string) *PooledConnection {
	pool.mu.RLock()
	conn, exists := pool.connections[url]
	pool.mu.RUnlock()

	if exists && !conn.InUse.Load() {
		conn.InUse.Store(true)
		return conn
	}

	return nil
}

func (pool *OCSPConnectionPool) CreateConnection(url string) *PooledConnection {
	pool.mu.Lock()
	defer pool.mu.Unlock()

	if len(pool.connections) >= pool.maxSize {
		// Evict least recently used
		pool.evictLRU()
	}

	conn := &PooledConnection{
		URL:      url,
		Client:   NewOCSPClient(url, pool.timeout),
		LastUsed: time.Now(),
	}
	conn.InUse.Store(true)

	pool.connections[url] = conn
	return conn
}

func (pool *OCSPConnectionPool) ReleaseConnection(conn *PooledConnection) {
	if conn != nil {
		conn.InUse.Store(false)
		conn.LastUsed = time.Now()
	}
}

func (pool *OCSPConnectionPool) evictLRU() {
	var oldestURL string
	var oldestTime time.Time

	for url, conn := range pool.connections {
		if !conn.InUse.Load() && (oldestURL == "" || conn.LastUsed.Before(oldestTime)) {
			oldestURL = url
			oldestTime = conn.LastUsed
		}
	}

	if oldestURL != "" {
		delete(pool.connections, oldestURL)
	}
}

// Emergency bypass methods

func (e *RealtimeValidationEngine) isEmergencyBypassActive() bool {
	if e.emergencyBypass == nil || !e.emergencyBypass.enabled.Load() {
		return false
	}

	bypassUntil := e.emergencyBypass.bypassUntil.Load().(time.Time)
	if time.Now().After(bypassUntil) {
		e.emergencyBypass.enabled.Store(false)
		return false
	}

	return true
}

func (e *RealtimeValidationEngine) EnableEmergencyBypass(authKey string, duration time.Duration, reason string) error {
	if e.emergencyBypass == nil {
		return fmt.Errorf("emergency bypass not configured")
	}

	// Validate authorization key
	authorized := false
	for _, key := range e.config.BypassAuthorizationKeys {
		if key == authKey {
			authorized = true
			break
		}
	}

	if !authorized {
		return fmt.Errorf("invalid authorization key")
	}

	e.emergencyBypass.mu.Lock()
	defer e.emergencyBypass.mu.Unlock()

	e.emergencyBypass.enabled.Store(true)
	e.emergencyBypass.bypassUntil.Store(time.Now().Add(duration))
	e.emergencyBypass.authorizedBy = authKey
	e.emergencyBypass.reason = reason

	// Audit log
	if e.emergencyBypass.auditLogger != nil {
		e.emergencyBypass.auditLogger.LogEmergencyBypass(authKey, duration, reason)
	}

	e.logger.Warn("emergency bypass enabled",
		"authorized_by", authKey,
		"duration", duration,
		"reason", reason)

	return nil
}

// Metrics recording

func (e *RealtimeValidationEngine) recordValidationMetrics(start time.Time, success bool, source string) {
	if e.metricsRecorder == nil {
		return
	}

	status := "success"
	if !success {
		status = "failure"
	}

	e.metricsRecorder.validationTotal.WithLabelValues(status, source).Inc()
}

func (e *RealtimeValidationEngine) recordValidationDuration(duration time.Duration) {
	e.mu.Lock()
	e.stats.ValidationDurations = append(e.stats.ValidationDurations, duration)
	// Keep only last 1000 durations for statistics
	if len(e.stats.ValidationDurations) > 1000 {
		e.stats.ValidationDurations = e.stats.ValidationDurations[1:]
	}
	e.mu.Unlock()

	if e.metricsRecorder != nil {
		e.metricsRecorder.validationDuration.WithLabelValues("realtime").Observe(duration.Seconds())
	}
}

// Webhook notification

func (e *RealtimeValidationEngine) sendValidationEvent(cert *x509.Certificate, result *ValidationResult, eventType string) {
	if e.webhookNotifier == nil {
		return
	}

	event := &ValidationEvent{
		Type:      eventType,
		Timestamp: time.Now(),
		Certificate: &CertificateInfo{
			SerialNumber: cert.SerialNumber.String(),
			Subject:      cert.Subject.String(),
			Issuer:       cert.Issuer.String(),
			NotBefore:    cert.NotBefore,
			NotAfter:     cert.NotAfter,
		},
		Result:   result,
		Severity: "high",
		Details: map[string]interface{}{
			"errors":   result.Errors,
			"warnings": result.Warnings,
		},
	}

	select {
	case e.webhookNotifier.eventQueue <- event:
	default:
		e.logger.Warn("webhook event queue full, dropping event")
	}
}

// Initialize metrics
func (e *RealtimeValidationEngine) initializeMetrics() *ValidationMetricsRecorder {
	return &ValidationMetricsRecorder{
		validationTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cert_validation_total",
				Help: "Total number of certificate validations",
			},
			[]string{"status", "source"},
		),
		validationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "cert_validation_duration_seconds",
				Help:    "Certificate validation duration in seconds",
				Buckets: prometheus.ExponentialBuckets(0.001, 2, 10),
			},
			[]string{"type"},
		),
		revocationCheckTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cert_revocation_check_total",
				Help: "Total number of revocation checks",
			},
			[]string{"method", "status"},
		),
		cacheHitRate: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "cert_validation_cache_hit_rate",
				Help: "Certificate validation cache hit rate",
			},
		),
		policyViolations: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "cert_policy_violations_total",
				Help: "Total number of certificate policy violations",
			},
			[]string{"rule", "severity"},
		),
		circuitBreakerState: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "cert_validation_circuit_breaker_state",
				Help: "Certificate validation circuit breaker state (0=closed, 1=open, 2=half-open)",
			},
			[]string{"component"},
		),
	}
}

// GetStatistics returns validation statistics
func (e *RealtimeValidationEngine) GetStatistics() *ValidationStatistics {
	return e.stats
}

// GetCacheStatistics returns cache statistics
func (e *RealtimeValidationEngine) GetCacheStatistics() map[string]interface{} {
	stats := make(map[string]interface{})

	totalValidations := e.stats.TotalValidations.Load()
	cacheHits := e.stats.CacheHits.Load()

	if totalValidations > 0 {
		stats["cache_hit_rate"] = float64(cacheHits) / float64(totalValidations)
	}

	stats["total_validations"] = totalValidations
	stats["successful_validations"] = e.stats.SuccessfulValidations.Load()
	stats["failed_validations"] = e.stats.FailedValidations.Load()
	stats["cache_hits"] = cacheHits
	stats["cache_misses"] = e.stats.CacheMisses.Load()
	stats["revocation_checks"] = e.stats.RevocationChecks.Load()
	stats["ct_log_checks"] = e.stats.CTLogChecks.Load()
	stats["policy_violations"] = e.stats.PolicyViolations.Load()
	stats["emergency_bypasses"] = e.stats.EmergencyBypasses.Load()

	// Calculate P95 validation duration
	e.mu.RLock()
	if len(e.stats.ValidationDurations) > 0 {
		durations := make([]time.Duration, len(e.stats.ValidationDurations))
		copy(durations, e.stats.ValidationDurations)
		e.mu.RUnlock()

		// Sort and calculate P95
		p95Index := int(float64(len(durations)) * 0.95)
		if p95Index < len(durations) {
			stats["validation_duration_p95"] = durations[p95Index]
		}
	} else {
		e.mu.RUnlock()
	}

	return stats
}

// Start starts the validation engine
func (e *RealtimeValidationEngine) Start(ctx context.Context) error {
	e.logger.Info("starting real-time validation engine")

	// Start webhook notifier workers
	if e.webhookNotifier != nil {
		for i := 0; i < e.webhookNotifier.workers; i++ {
			e.wg.Add(1)
			go e.runWebhookWorker()
		}
	}

	// Start metrics updater
	if e.metricsRecorder != nil {
		e.wg.Add(1)
		go e.runMetricsUpdater()
	}

	// Start cache cleanup
	e.wg.Add(1)
	go e.runCacheCleanup()

	return nil
}

// Stop stops the validation engine
func (e *RealtimeValidationEngine) Stop() {
	e.logger.Info("stopping real-time validation engine")
	e.cancel()
	e.wg.Wait()
}

// Worker routines

func (e *RealtimeValidationEngine) runWebhookWorker() {
	defer e.wg.Done()

	for {
		select {
		case <-e.ctx.Done():
			return
		case event := <-e.webhookNotifier.eventQueue:
			if event != nil {
				e.webhookNotifier.client.SendEvent(event)
			}
		}
	}
}

func (e *RealtimeValidationEngine) runMetricsUpdater() {
	defer e.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.updateMetrics()
		}
	}
}

func (e *RealtimeValidationEngine) runCacheCleanup() {
	defer e.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.cleanupCache()
		}
	}
}

func (e *RealtimeValidationEngine) updateMetrics() {
	stats := e.GetCacheStatistics()

	if e.metricsRecorder != nil && stats["cache_hit_rate"] != nil {
		e.metricsRecorder.cacheHitRate.Set(stats["cache_hit_rate"].(float64))
	}

	// Update circuit breaker state
	if e.circuitBreaker != nil && e.metricsRecorder != nil {
		state := e.circuitBreaker.state.Load().(CircuitState)
		e.metricsRecorder.circuitBreakerState.WithLabelValues("main").Set(float64(state))
	}
}

func (e *RealtimeValidationEngine) cleanupCache() {
	if e.cache != nil && e.cache.l1Cache != nil {
		// Clean up expired entries from L1 cache
		e.cache.l1Cache.mu.Lock()
		now := time.Now()
		for key, entry := range e.cache.l1Cache.cache {
			if now.Sub(entry.CreatedAt) > e.cache.l1Cache.ttl {
				delete(e.cache.l1Cache.cache, key)
				// Remove from order slice
				for i, orderKey := range e.cache.l1Cache.order {
					if orderKey == key {
						e.cache.l1Cache.order = append(e.cache.l1Cache.order[:i], e.cache.l1Cache.order[i+1:]...)
						break
					}
				}
			}
		}
		e.cache.l1Cache.mu.Unlock()
	}
}

// Helper types

type validationComponent struct {
	Type   string
	Result interface{}
}

type ORANComplianceResult struct {
	Valid  bool        `json:"valid"`
	Checks []ORANCheck `json:"checks"`
	Score  float64     `json:"score"`
}

type ORANCheck struct {
	Name    string `json:"name"`
	Passed  bool   `json:"passed"`
	Message string `json:"message,omitempty"`
}

// L1 Cache methods

func (cache *L1ValidationCache) Get(key string) *CachedValidationResult {
	cache.mu.RLock()
	defer cache.mu.RUnlock()

	result, exists := cache.cache[key]
	if !exists {
		return nil
	}

	// Check TTL
	if time.Since(result.CachedAt) > cache.ttl {
		return nil
	}

	return result
}

func (cache *L1ValidationCache) Put(key string, result *CachedValidationResult) {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	// Check size limit
	if len(cache.cache) >= cache.maxSize {
		// Simple eviction - remove oldest
		cache.evictOldest()
	}

	cache.cache[key] = result
}

func (cache *L1ValidationCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, result := range cache.cache {
		if oldestKey == "" || result.CachedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = result.CachedAt
		}
	}

	if oldestKey != "" {
		delete(cache.cache, oldestKey)
	}
}

func (cache *L1ValidationCache) Cleanup() {
	cache.mu.Lock()
	defer cache.mu.Unlock()

	now := time.Now()
	for key, result := range cache.cache {
		if now.Sub(result.CachedAt) > cache.ttl {
			delete(cache.cache, key)
		}
	}
}

// aggregateValidationResult aggregates component validation results
func (e *RealtimeValidationEngine) aggregateValidationResult(result *ValidationResult, component validationComponent) {
	switch component.Type {
	case "chain":
		if chainResult, ok := component.Result.(*ChainValidationResult); ok {
			result.ChainValid = chainResult.Valid
			if !chainResult.Valid {
				result.Valid = false
				result.Errors = append(result.Errors, chainResult.Errors...)
			}
			result.Warnings = append(result.Warnings, chainResult.Warnings...)
		}
	case "revocation":
		if revResult, ok := component.Result.(*RevocationCheckResult); ok {
			result.RevocationStatus = revResult.Status
			if revResult.Status == RevocationStatusRevoked {
				result.Valid = false
				result.Errors = append(result.Errors, "certificate is revoked")
			}
			result.Errors = append(result.Errors, revResult.Errors...)
			result.Warnings = append(result.Warnings, revResult.Warnings...)
		}
	case "ct":
		if ctValid, ok := component.Result.(bool); ok {
			result.CTLogVerified = ctValid
			if !ctValid && e.config.RequireSCTValidation {
				result.Valid = false
				result.Errors = append(result.Errors, "CT log validation failed")
			}
		}
	case "policy":
		if policyResult, ok := component.Result.(*PolicyValidationResult); ok {
			if !policyResult.Valid {
				result.Valid = false
				for _, violation := range policyResult.Violations {
					result.Errors = append(result.Errors, violation.Description)
				}
			}
			for _, warning := range policyResult.Warnings {
				result.Warnings = append(result.Warnings, warning.Description)
			}
		}
	case "oran":
		if oranResult, ok := component.Result.(*ORANComplianceResult); ok {
			if !oranResult.Valid {
				result.Valid = false
				result.Errors = append(result.Errors, "O-RAN compliance validation failed")
			}
			for _, check := range oranResult.Checks {
				if !check.Passed && check.Message != "" {
					result.Warnings = append(result.Warnings, check.Message)
				}
			}
		}
	}
}

// validateSCTs validates Signed Certificate Timestamps
func (e *RealtimeValidationEngine) validateSCTs(ctx context.Context, cert *x509.Certificate) bool {
	// Check for embedded SCTs in certificate
	for _, ext := range cert.Extensions {
		// SCT List extension OID: 1.3.6.1.4.1.11129.2.4.2
		if ext.Id.Equal([]int{1, 3, 6, 1, 4, 1, 11129, 2, 4, 2}) {
			// Found SCT extension
			// Real implementation would parse and validate SCTs
			return true
		}
	}

	// No SCTs found
	return false
}
