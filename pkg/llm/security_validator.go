//go:build !disable_rag
// +build !disable_rag

package llm

import (
	"crypto/subtle"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"
)

// SecurityValidator handles input validation and prompt injection protection.

type SecurityValidator struct {
	config *SecurityConfig

	rateLimiter *RateLimiter

	promptDetector *PromptInjectionDetector

	contentFilter *ContentFilter

	logger *slog.Logger

	metrics *SecurityMetrics

	mutex sync.RWMutex
}

// SecurityConfig holds security validation configuration.

type SecurityConfig struct {

	// API Key validation.

	EnableAPIKeyAuth bool `json:"enable_api_key_auth"`

	ValidAPIKeys []string `json:"valid_api_keys"`

	APIKeyHeader string `json:"api_key_header"`

	// Rate limiting.

	EnableRateLimit bool `json:"enable_rate_limit"`

	RequestsPerMinute int `json:"requests_per_minute"`

	BurstLimit int `json:"burst_limit"`

	RateLimitWindow time.Duration `json:"rate_limit_window"`

	RateLimitByIP bool `json:"rate_limit_by_ip"`

	RateLimitByAPIKey bool `json:"rate_limit_by_api_key"`

	// Input validation.

	MaxInputLength int `json:"max_input_length"`

	MaxOutputLength int `json:"max_output_length"`

	ForbiddenPatterns []string `json:"forbidden_patterns"`

	RequiredHeaders []string `json:"required_headers"`

	// Prompt injection protection.

	EnablePromptInjectionDetection bool `json:"enable_prompt_injection_detection"`

	InjectionPatterns []string `json:"injection_patterns"`

	SuspiciousKeywords []string `json:"suspicious_keywords"`

	MaxPromptComplexity int `json:"max_prompt_complexity"`

	// Content filtering.

	EnableContentFilter bool `json:"enable_content_filter"`

	BlockedTopics []string `json:"blocked_topics"`

	RequiredTopics []string `json:"required_topics"`

	ProfanityFilter bool `json:"profanity_filter"`

	// IP filtering.

	EnableIPFiltering bool `json:"enable_ip_filtering"`

	AllowedIPs []string `json:"allowed_ips"`

	BlockedIPs []string `json:"blocked_ips"`

	AllowPrivateIPs bool `json:"allow_private_ips"`

	// Audit logging.

	EnableAuditLogging bool `json:"enable_audit_logging"`

	LogSuccessfulRequests bool `json:"log_successful_requests"`

	LogFailedRequests bool `json:"log_failed_requests"`

	AuditLogLevel string `json:"audit_log_level"`

	// Response sanitization.

	EnableResponseSanitization bool `json:"enable_response_sanitization"`

	SanitizationPatterns []string `json:"sanitization_patterns"`

	RemovePII bool `json:"remove_pii"`
}

// SecurityMetrics tracks security-related metrics.

type SecurityMetrics struct {
	TotalRequests int64 `json:"total_requests"`

	ValidatedRequests int64 `json:"validated_requests"`

	RejectedRequests int64 `json:"rejected_requests"`

	RateLimitedRequests int64 `json:"rate_limited_requests"`

	PromptInjectionAttempts int64 `json:"prompt_injection_attempts"`

	ContentFilterBlocks int64 `json:"content_filter_blocks"`

	IPFilterBlocks int64 `json:"ip_filter_blocks"`

	APIKeyFailures int64 `json:"api_key_failures"`

	ValidationErrors int64 `json:"validation_errors"`

	AverageValidationTime time.Duration `json:"average_validation_time"`

	LastUpdated time.Time `json:"last_updated"`

	mutex sync.RWMutex
}

// ValidationResult represents the result of security validation.

type ValidationResult struct {
	Valid bool `json:"valid"`

	Errors []string `json:"errors"`

	Warnings []string `json:"warnings"`

	RiskScore float64 `json:"risk_score"`

	ProcessingTime time.Duration `json:"processing_time"`

	AppliedFilters []string `json:"applied_filters"`

	DetectedThreats []string `json:"detected_threats"`

	SanitizedInput string `json:"sanitized_input"`

	Metadata map[string]interface{} `json:"metadata"`
}

// RateLimiter implements token bucket rate limiting.

type RateLimiter struct {
	limiters map[string]*TokenBucket

	config *RateLimitConfig

	mutex sync.RWMutex
}

// RateLimitConfig holds rate limiting configuration.

type RateLimitConfig struct {
	RequestsPerMinute int `json:"requests_per_minute"`

	BurstLimit int `json:"burst_limit"`

	WindowSize time.Duration `json:"window_size"`

	CleanupInterval time.Duration `json:"cleanup_interval"`
}

// TokenBucket implements token bucket algorithm.

type TokenBucket struct {
	tokens float64

	capacity float64

	refillRate float64

	lastRefill time.Time

	mutex sync.Mutex
}

// PromptInjectionDetector detects prompt injection attempts.

type PromptInjectionDetector struct {
	patterns []*regexp.Regexp

	suspiciousWords map[string]bool

	complexityThreshold int

	logger *slog.Logger
}

// ContentFilter filters content based on topics and profanity.

type ContentFilter struct {
	blockedTopics []*regexp.Regexp

	requiredTopics []*regexp.Regexp

	profanityWords map[string]bool

	logger *slog.Logger
}

// NewSecurityValidator creates a new security validator.

func NewSecurityValidator(config *SecurityConfig) *SecurityValidator {

	if config == nil {

		config = getDefaultSecurityConfig()

	}

	validator := &SecurityValidator{

		config: config,

		logger: slog.Default().With("component", "security-validator"),

		metrics: &SecurityMetrics{LastUpdated: time.Now()},
	}

	// Initialize rate limiter.

	if config.EnableRateLimit {

		validator.rateLimiter = NewRateLimiter(&RateLimitConfig{

			RequestsPerMinute: config.RequestsPerMinute,

			BurstLimit: config.BurstLimit,

			WindowSize: config.RateLimitWindow,

			CleanupInterval: 5 * time.Minute,
		})

	}

	// Initialize prompt injection detector.

	if config.EnablePromptInjectionDetection {

		validator.promptDetector = NewPromptInjectionDetector(

			config.InjectionPatterns,

			config.SuspiciousKeywords,

			config.MaxPromptComplexity,
		)

	}

	// Initialize content filter.

	if config.EnableContentFilter {

		validator.contentFilter = NewContentFilter(

			config.BlockedTopics,

			config.RequiredTopics,

			config.ProfanityFilter,
		)

	}

	return validator

}

// getDefaultSecurityConfig returns default security configuration.

func getDefaultSecurityConfig() *SecurityConfig {

	return &SecurityConfig{

		EnableAPIKeyAuth: false,

		APIKeyHeader: "X-API-Key",

		EnableRateLimit: true,

		RequestsPerMinute: 60,

		BurstLimit: 10,

		RateLimitWindow: time.Minute,

		RateLimitByIP: true,

		RateLimitByAPIKey: false,

		MaxInputLength: 10000,

		MaxOutputLength: 50000,

		EnablePromptInjectionDetection: true,

		InjectionPatterns: []string{

			`ignore.{0,20}previous.{0,20}instructions`,

			`forget.{0,20}everything.{0,20}above`,

			`system.{0,20}prompt`,

			`act.{0,20}as.{0,20}(admin|root|system)`,

			`override.{0,20}security`,
		},

		SuspiciousKeywords: []string{

			"ignore", "forget", "override", "bypass", "admin", "root", "system",

			"jailbreak", "prompt", "instruction", "command", "execute",
		},

		MaxPromptComplexity: 100,

		EnableContentFilter: true,

		ProfanityFilter: true,

		EnableIPFiltering: false,

		AllowPrivateIPs: true,

		EnableAuditLogging: true,

		LogSuccessfulRequests: false,

		LogFailedRequests: true,

		AuditLogLevel: "info",

		EnableResponseSanitization: true,

		RemovePII: true,
	}

}

// ValidateRequest validates an incoming HTTP request.

func (sv *SecurityValidator) ValidateRequest(r *http.Request, input string) (*ValidationResult, error) {

	startTime := time.Now()

	result := &ValidationResult{

		Valid: true,

		Errors: []string{},

		Warnings: []string{},

		RiskScore: 0.0,

		AppliedFilters: []string{},

		DetectedThreats: []string{},

		SanitizedInput: input,

		Metadata: make(map[string]interface{}),
	}

	// Update metrics.

	sv.updateMetrics(func(m *SecurityMetrics) {

		m.TotalRequests++

	})

	// API Key validation.

	if sv.config.EnableAPIKeyAuth {

		if err := sv.validateAPIKey(r); err != nil {

			result.Valid = false

			result.Errors = append(result.Errors, err.Error())

			result.RiskScore += 0.8

			result.DetectedThreats = append(result.DetectedThreats, "invalid_api_key")

			sv.updateMetrics(func(m *SecurityMetrics) {

				m.APIKeyFailures++

			})

		} else {

			result.AppliedFilters = append(result.AppliedFilters, "api_key_validation")

		}

	}

	// Rate limiting.

	if sv.config.EnableRateLimit && sv.rateLimiter != nil {

		identifier := sv.getRateLimitIdentifier(r)

		if !sv.rateLimiter.Allow(identifier) {

			result.Valid = false

			result.Errors = append(result.Errors, "rate limit exceeded")

			result.RiskScore += 0.6

			result.DetectedThreats = append(result.DetectedThreats, "rate_limit_exceeded")

			sv.updateMetrics(func(m *SecurityMetrics) {

				m.RateLimitedRequests++

			})

		} else {

			result.AppliedFilters = append(result.AppliedFilters, "rate_limiting")

		}

	}

	// IP filtering.

	if sv.config.EnableIPFiltering {

		if err := sv.validateIP(r); err != nil {

			result.Valid = false

			result.Errors = append(result.Errors, err.Error())

			result.RiskScore += 0.7

			result.DetectedThreats = append(result.DetectedThreats, "blocked_ip")

			sv.updateMetrics(func(m *SecurityMetrics) {

				m.IPFilterBlocks++

			})

		} else {

			result.AppliedFilters = append(result.AppliedFilters, "ip_filtering")

		}

	}

	// Input validation.

	if err := sv.validateInput(input); err != nil {

		result.Valid = false

		result.Errors = append(result.Errors, err.Error())

		result.RiskScore += 0.5

		sv.updateMetrics(func(m *SecurityMetrics) {

			m.ValidationErrors++

		})

	}

	// Prompt injection detection.

	if sv.config.EnablePromptInjectionDetection && sv.promptDetector != nil {

		threats, riskIncrease := sv.promptDetector.Detect(input)

		if len(threats) > 0 {

			result.Warnings = append(result.Warnings, fmt.Sprintf("potential prompt injection detected: %v", threats))

			result.RiskScore += riskIncrease

			result.DetectedThreats = append(result.DetectedThreats, threats...)

			result.AppliedFilters = append(result.AppliedFilters, "prompt_injection_detection")

			sv.updateMetrics(func(m *SecurityMetrics) {

				m.PromptInjectionAttempts++

			})

			// Block if risk is too high.

			if riskIncrease > 0.7 {

				result.Valid = false

				result.Errors = append(result.Errors, "high-risk prompt injection attempt blocked")

			}

		}

	}

	// Content filtering.

	if sv.config.EnableContentFilter && sv.contentFilter != nil {

		if blocked, reason := sv.contentFilter.Filter(input); blocked {

			result.Valid = false

			result.Errors = append(result.Errors, fmt.Sprintf("content blocked: %s", reason))

			result.RiskScore += 0.6

			result.DetectedThreats = append(result.DetectedThreats, "blocked_content")

			result.AppliedFilters = append(result.AppliedFilters, "content_filtering")

			sv.updateMetrics(func(m *SecurityMetrics) {

				m.ContentFilterBlocks++

			})

		}

	}

	// Sanitize input.

	result.SanitizedInput = sv.sanitizeInput(input)

	if result.SanitizedInput != input {

		result.AppliedFilters = append(result.AppliedFilters, "input_sanitization")

		result.Warnings = append(result.Warnings, "input was sanitized")

	}

	result.ProcessingTime = time.Since(startTime)

	// Update metrics based on result.

	if result.Valid {

		sv.updateMetrics(func(m *SecurityMetrics) {

			m.ValidatedRequests++

		})

	} else {

		sv.updateMetrics(func(m *SecurityMetrics) {

			m.RejectedRequests++

		})

	}

	sv.updateMetrics(func(m *SecurityMetrics) {

		m.AverageValidationTime = (m.AverageValidationTime*time.Duration(m.TotalRequests-1) + result.ProcessingTime) / time.Duration(m.TotalRequests)

		m.LastUpdated = time.Now()

	})

	// Audit logging.

	if sv.config.EnableAuditLogging {

		sv.auditLog(r, result)

	}

	return result, nil

}

// validateAPIKey validates the API key in the request.

func (sv *SecurityValidator) validateAPIKey(r *http.Request) error {

	apiKey := r.Header.Get(sv.config.APIKeyHeader)

	if apiKey == "" {

		return fmt.Errorf("missing API key in header %s", sv.config.APIKeyHeader)

	}

	// Check against valid API keys using constant-time comparison.

	for _, validKey := range sv.config.ValidAPIKeys {

		if subtle.ConstantTimeCompare([]byte(apiKey), []byte(validKey)) == 1 {

			return nil

		}

	}

	return fmt.Errorf("invalid API key")

}

// getRateLimitIdentifier gets the identifier for rate limiting.

func (sv *SecurityValidator) getRateLimitIdentifier(r *http.Request) string {

	if sv.config.RateLimitByAPIKey {

		if apiKey := r.Header.Get(sv.config.APIKeyHeader); apiKey != "" {

			return "api_key:" + apiKey

		}

	}

	if sv.config.RateLimitByIP {

		return "ip:" + sv.getClientIP(r)

	}

	return "global"

}

// getClientIP extracts the client IP from the request.

func (sv *SecurityValidator) getClientIP(r *http.Request) string {

	// Check X-Forwarded-For header first.

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {

		ips := strings.Split(xff, ",")

		if len(ips) > 0 {

			return strings.TrimSpace(ips[0])

		}

	}

	// Check X-Real-IP header.

	if xri := r.Header.Get("X-Real-IP"); xri != "" {

		return xri

	}

	// Fall back to RemoteAddr.

	host, _, err := net.SplitHostPort(r.RemoteAddr)

	if err != nil {

		return r.RemoteAddr

	}

	return host

}

// validateIP validates the client IP against allow/block lists.

func (sv *SecurityValidator) validateIP(r *http.Request) error {

	clientIP := sv.getClientIP(r)

	ip := net.ParseIP(clientIP)

	if ip == nil {

		return fmt.Errorf("invalid IP address: %s", clientIP)

	}

	// Check blocked IPs.

	for _, blockedIP := range sv.config.BlockedIPs {

		if blocked := net.ParseIP(blockedIP); blocked != nil && ip.Equal(blocked) {

			return fmt.Errorf("IP address is blocked: %s", clientIP)

		}

	}

	// Check allowed IPs (if specified).

	if len(sv.config.AllowedIPs) > 0 {

		allowed := false

		for _, allowedIP := range sv.config.AllowedIPs {

			if allow := net.ParseIP(allowedIP); allow != nil && ip.Equal(allow) {

				allowed = true

				break

			}

		}

		if !allowed {

			return fmt.Errorf("IP address not in allow list: %s", clientIP)

		}

	}

	// Check private IP restrictions.

	if !sv.config.AllowPrivateIPs && isPrivateIP(ip) {

		return fmt.Errorf("private IP addresses not allowed: %s", clientIP)

	}

	return nil

}

// isPrivateIP checks if an IP is private.

func isPrivateIP(ip net.IP) bool {

	privateRanges := []string{

		"10.0.0.0/8",

		"172.16.0.0/12",

		"192.168.0.0/16",
	}

	for _, rangeStr := range privateRanges {

		_, cidr, _ := net.ParseCIDR(rangeStr)

		if cidr.Contains(ip) {

			return true

		}

	}

	return false

}

// validateInput validates input content.

func (sv *SecurityValidator) validateInput(input string) error {

	if len(input) > sv.config.MaxInputLength {

		return fmt.Errorf("input too long: %d > %d", len(input), sv.config.MaxInputLength)

	}

	// Check forbidden patterns.

	for _, pattern := range sv.config.ForbiddenPatterns {

		if matched, _ := regexp.MatchString(pattern, input); matched {

			return fmt.Errorf("input contains forbidden pattern: %s", pattern)

		}

	}

	return nil

}

// sanitizeInput sanitizes input by removing/replacing dangerous content.

func (sv *SecurityValidator) sanitizeInput(input string) string {

	sanitized := input

	// Remove common injection attempts.

	injectionPatterns := []string{

		`<script[^>]*>.*?</script>`,

		`javascript:`,

		`vbscript:`,

		`onload\s*=`,

		`onerror\s*=`,
	}

	for _, pattern := range injectionPatterns {

		re := regexp.MustCompile(pattern)

		sanitized = re.ReplaceAllString(sanitized, "")

	}

	// Remove PII if enabled.

	if sv.config.RemovePII {

		sanitized = sv.removePII(sanitized)

	}

	return sanitized

}

// removePII removes personally identifiable information.

func (sv *SecurityValidator) removePII(input string) string {

	sanitized := input

	// Email addresses.

	emailRegex := regexp.MustCompile(`\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b`)

	sanitized = emailRegex.ReplaceAllString(sanitized, "[EMAIL]")

	// Phone numbers (basic patterns).

	phoneRegex := regexp.MustCompile(`\b\d{3}[-.]?\d{3}[-.]?\d{4}\b`)

	sanitized = phoneRegex.ReplaceAllString(sanitized, "[PHONE]")

	// Credit card numbers (basic patterns).

	ccRegex := regexp.MustCompile(`\b\d{4}[-\s]?\d{4}[-\s]?\d{4}[-\s]?\d{4}\b`)

	sanitized = ccRegex.ReplaceAllString(sanitized, "[CREDIT_CARD]")

	return sanitized

}

// auditLog logs security events.

func (sv *SecurityValidator) auditLog(r *http.Request, result *ValidationResult) {

	if !result.Valid || (result.Valid && sv.config.LogSuccessfulRequests) || (!result.Valid && sv.config.LogFailedRequests) {

		logData := map[string]interface{}{

			"client_ip": sv.getClientIP(r),

			"user_agent": r.UserAgent(),

			"method": r.Method,

			"path": r.URL.Path,

			"valid": result.Valid,

			"risk_score": result.RiskScore,

			"errors": result.Errors,

			"warnings": result.Warnings,

			"detected_threats": result.DetectedThreats,

			"applied_filters": result.AppliedFilters,

			"processing_time": result.ProcessingTime.String(),
		}

		if result.Valid {

			sv.logger.Info("Security validation passed", "audit", logData)

		} else {

			sv.logger.Warn("Security validation failed", "audit", logData)

		}

	}

}

// updateMetrics safely updates security metrics.

func (sv *SecurityValidator) updateMetrics(updater func(*SecurityMetrics)) {

	sv.metrics.mutex.Lock()

	defer sv.metrics.mutex.Unlock()

	updater(sv.metrics)

}

// GetMetrics returns current security metrics.

func (sv *SecurityValidator) GetMetrics() *SecurityMetrics {

	sv.metrics.mutex.RLock()

	defer sv.metrics.mutex.RUnlock()

	// Create a copy without the mutex.

	metrics := &SecurityMetrics{

		TotalRequests: sv.metrics.TotalRequests,

		ValidatedRequests: sv.metrics.ValidatedRequests,

		RejectedRequests: sv.metrics.RejectedRequests,

		RateLimitedRequests: sv.metrics.RateLimitedRequests,

		PromptInjectionAttempts: sv.metrics.PromptInjectionAttempts,

		ContentFilterBlocks: sv.metrics.ContentFilterBlocks,

		IPFilterBlocks: sv.metrics.IPFilterBlocks,

		APIKeyFailures: sv.metrics.APIKeyFailures,

		ValidationErrors: sv.metrics.ValidationErrors,

		AverageValidationTime: sv.metrics.AverageValidationTime,

		LastUpdated: sv.metrics.LastUpdated,
	}

	return metrics

}

// NewRateLimiter creates a new rate limiter.

func NewRateLimiter(config *RateLimitConfig) *RateLimiter {

	rl := &RateLimiter{

		limiters: make(map[string]*TokenBucket),

		config: config,
	}

	// Start cleanup routine.

	go rl.cleanupRoutine()

	return rl

}

// Allow checks if a request is allowed under rate limiting.

func (rl *RateLimiter) Allow(identifier string) bool {

	rl.mutex.Lock()

	defer rl.mutex.Unlock()

	bucket, exists := rl.limiters[identifier]

	if !exists {

		bucket = NewTokenBucket(float64(rl.config.BurstLimit), float64(rl.config.RequestsPerMinute)/60.0)

		rl.limiters[identifier] = bucket

	}

	return bucket.Allow()

}

// cleanupRoutine removes old rate limiters.

func (rl *RateLimiter) cleanupRoutine() {

	ticker := time.NewTicker(rl.config.CleanupInterval)

	defer ticker.Stop()

	for range ticker.C {

		rl.mutex.Lock()

		cutoff := time.Now().Add(-rl.config.WindowSize * 2)

		for identifier, bucket := range rl.limiters {

			bucket.mutex.Lock()

			if bucket.lastRefill.Before(cutoff) {

				delete(rl.limiters, identifier)

			}

			bucket.mutex.Unlock()

		}

		rl.mutex.Unlock()

	}

}

// NewTokenBucket creates a new token bucket.

func NewTokenBucket(capacity, refillRate float64) *TokenBucket {

	return &TokenBucket{

		tokens: capacity,

		capacity: capacity,

		refillRate: refillRate,

		lastRefill: time.Now(),
	}

}

// Allow checks if a token is available.

func (tb *TokenBucket) Allow() bool {

	tb.mutex.Lock()

	defer tb.mutex.Unlock()

	now := time.Now()

	elapsed := now.Sub(tb.lastRefill).Seconds()

	// Refill tokens.

	tb.tokens += elapsed * tb.refillRate

	if tb.tokens > tb.capacity {

		tb.tokens = tb.capacity

	}

	tb.lastRefill = now

	// Check if token is available.

	if tb.tokens >= 1 {

		tb.tokens--

		return true

	}

	return false

}

// NewPromptInjectionDetector creates a new prompt injection detector.

func NewPromptInjectionDetector(patterns, keywords []string, complexityThreshold int) *PromptInjectionDetector {

	detector := &PromptInjectionDetector{

		patterns: make([]*regexp.Regexp, 0),

		suspiciousWords: make(map[string]bool),

		complexityThreshold: complexityThreshold,

		logger: slog.Default().With("component", "prompt-injection-detector"),
	}

	// Compile patterns.

	for _, pattern := range patterns {

		if regex, err := regexp.Compile(`(?i)` + pattern); err == nil {

			detector.patterns = append(detector.patterns, regex)

		}

	}

	// Build suspicious words map.

	for _, word := range keywords {

		detector.suspiciousWords[strings.ToLower(word)] = true

	}

	return detector

}

// Detect detects potential prompt injection attempts.

func (pid *PromptInjectionDetector) Detect(input string) ([]string, float64) {

	var threats []string

	riskScore := 0.0

	inputLower := strings.ToLower(input)

	// Check against known injection patterns.

	for _, pattern := range pid.patterns {

		if pattern.MatchString(inputLower) {

			threats = append(threats, "injection_pattern")

			riskScore += 0.8

		}

	}

	// Check for suspicious keywords.

	words := strings.Fields(inputLower)

	suspiciousCount := 0

	for _, word := range words {

		if pid.suspiciousWords[word] {

			suspiciousCount++

		}

	}

	if suspiciousCount > 0 {

		threats = append(threats, "suspicious_keywords")

		riskScore += float64(suspiciousCount) * 0.1

	}

	// Check prompt complexity.

	complexity := pid.calculateComplexity(input)

	if complexity > pid.complexityThreshold {

		threats = append(threats, "high_complexity")

		riskScore += 0.3

	}

	// Normalize risk score to 0-1 range.

	if riskScore > 1.0 {

		riskScore = 1.0

	}

	return threats, riskScore

}

// calculateComplexity calculates prompt complexity.

func (pid *PromptInjectionDetector) calculateComplexity(input string) int {

	// Simple complexity calculation based on:.

	// - Character count.

	// - Special character density.

	// - Repeated patterns.

	complexity := len(input) / 10 // Base complexity from length

	// Count special characters.

	specialChars := 0

	for _, char := range input {

		if !((char >= 'a' && char <= 'z') || (char >= 'A' && char <= 'Z') || (char >= '0' && char <= '9') || char == ' ') {

			specialChars++

		}

	}

	if len(input) > 0 {

		complexity += int(float64(specialChars) / float64(len(input)) * 100)

	}

	return complexity

}

// NewContentFilter creates a new content filter.

func NewContentFilter(blockedTopics, requiredTopics []string, profanityFilter bool) *ContentFilter {

	filter := &ContentFilter{

		blockedTopics: make([]*regexp.Regexp, 0),

		requiredTopics: make([]*regexp.Regexp, 0),

		profanityWords: make(map[string]bool),

		logger: slog.Default().With("component", "content-filter"),
	}

	// Compile blocked topic patterns.

	for _, topic := range blockedTopics {

		if regex, err := regexp.Compile(`(?i)` + topic); err == nil {

			filter.blockedTopics = append(filter.blockedTopics, regex)

		}

	}

	// Compile required topic patterns.

	for _, topic := range requiredTopics {

		if regex, err := regexp.Compile(`(?i)` + topic); err == nil {

			filter.requiredTopics = append(filter.requiredTopics, regex)

		}

	}

	// Initialize profanity filter.

	if profanityFilter {

		filter.initializeProfanityFilter()

	}

	return filter

}

// Filter applies content filtering.

func (cf *ContentFilter) Filter(input string) (bool, string) {

	inputLower := strings.ToLower(input)

	// Check blocked topics.

	for _, pattern := range cf.blockedTopics {

		if pattern.MatchString(inputLower) {

			return true, "blocked topic detected"

		}

	}

	// Check required topics (if any).

	if len(cf.requiredTopics) > 0 {

		found := false

		for _, pattern := range cf.requiredTopics {

			if pattern.MatchString(inputLower) {

				found = true

				break

			}

		}

		if !found {

			return true, "required topic not found"

		}

	}

	// Check profanity.

	if len(cf.profanityWords) > 0 {

		words := strings.Fields(inputLower)

		for _, word := range words {

			if cf.profanityWords[word] {

				return true, "profanity detected"

			}

		}

	}

	return false, ""

}

// initializeProfanityFilter initializes the profanity filter.

func (cf *ContentFilter) initializeProfanityFilter() {

	// Basic profanity list (in production, use a comprehensive list).

	profanityList := []string{

		// Add actual profanity words here.

		"badword1", "badword2", // Placeholder

	}

	for _, word := range profanityList {

		cf.profanityWords[strings.ToLower(word)] = true

	}

}
