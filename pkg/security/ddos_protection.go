// Package security implements DDoS protection and advanced rate limiting
// for Nephoran Intent Operator with O-RAN WG11 compliance
package security

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"golang.org/x/time/rate"
)

var (
	// DDoS Protection Metrics
	ddosAttacksDetected = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_ddos_attacks_detected_total",
			Help: "Total number of DDoS attacks detected",
		},
		[]string{"type", "source"},
	)

	ddosBlockedRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_ddos_blocked_requests_total",
			Help: "Total number of requests blocked by DDoS protection",
		},
		[]string{"reason", "source_ip"},
	)

	rateLimitViolations = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_rate_limit_violations_total",
			Help: "Total number of rate limit violations",
		},
		[]string{"tier", "source_ip"},
	)

	activeConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nephoran_active_connections",
			Help: "Number of active connections",
		},
		[]string{"source_type"},
	)
)

// DDoSProtectionConfig contains DDoS protection configuration
type DDoSProtectionConfig struct {
	// Rate limiting tiers
	GlobalRateLimit     int           `json:"global_rate_limit"`     // Requests per second globally
	PerIPRateLimit      int           `json:"per_ip_rate_limit"`     // Requests per second per IP
	BurstSize           int           `json:"burst_size"`            // Burst capacity
	WindowSize          time.Duration `json:"window_size"`           // Time window for rate limiting

	// Connection limits
	MaxConcurrentConns  int `json:"max_concurrent_conns"`  // Maximum concurrent connections
	MaxConnsPerIP       int `json:"max_conns_per_ip"`      // Maximum connections per IP
	ConnectionTimeout   time.Duration `json:"connection_timeout"`   // Connection timeout

	// Detection thresholds
	SuspiciousThreshold int           `json:"suspicious_threshold"`  // Requests to trigger suspicious behavior
	AttackThreshold     int           `json:"attack_threshold"`      // Requests to trigger attack detection
	DetectionWindow     time.Duration `json:"detection_window"`      // Time window for attack detection

	// Blocking and mitigation
	BlockDuration       time.Duration `json:"block_duration"`        // How long to block attacking IPs
	TempBanDuration     time.Duration `json:"temp_ban_duration"`     // Temporary ban duration
	MaxBlockedIPs       int           `json:"max_blocked_ips"`       // Maximum number of IPs to block

	// Whitelist and blacklist
	WhitelistIPs        []string      `json:"whitelist_ips"`         // Always allowed IPs
	WhitelistCIDRs      []string      `json:"whitelist_cidrs"`       // Always allowed CIDR ranges
	BlacklistIPs        []string      `json:"blacklist_ips"`         // Always blocked IPs
	BlacklistCIDRs      []string      `json:"blacklist_cidrs"`       // Always blocked CIDR ranges

	// Geolocation filtering
	EnableGeoFiltering  bool          `json:"enable_geo_filtering"`  // Enable geolocation-based filtering
	AllowedCountries    []string      `json:"allowed_countries"`     // Allowed country codes
	BlockedCountries    []string      `json:"blocked_countries"`     // Blocked country codes

	// Challenge mechanisms
	EnableCaptcha       bool          `json:"enable_captcha"`        // Enable CAPTCHA challenges
	EnableRateProof     bool          `json:"enable_rate_proof"`     // Enable proof-of-work challenges
	ChallengeThreshold  int           `json:"challenge_threshold"`   // Requests to trigger challenge

	// Monitoring and alerting
	EnableAlerts        bool          `json:"enable_alerts"`         // Enable security alerts
	AlertWebhook        string        `json:"alert_webhook"`         // Webhook URL for alerts
	MetricsRetention    time.Duration `json:"metrics_retention"`     // How long to keep metrics
}

// DDoSProtector implements comprehensive DDoS protection
type DDoSProtector struct {
	config         *DDoSProtectionConfig
	logger         *slog.Logger

	// Rate limiting
	globalLimiter  *rate.Limiter
	ipLimiters     sync.Map // map[string]*IPLimiter
	
	// Connection tracking
	activeConns    int64
	connsByIP      sync.Map // map[string]int64
	
	// Attack detection
	requestCounts  sync.Map // map[string]*RequestCounter
	blockedIPs     sync.Map // map[string]*BlockedIP
	suspiciousIPs  sync.Map // map[string]*SuspiciousActivity
	
	// Network lists
	whitelist      *NetworkList
	blacklist      *NetworkList
	
	// Geolocation filter
	geoFilter      *GeolocationFilter
	
	// Challenge system
	challenges     sync.Map // map[string]*Challenge
	
	// Metrics and stats
	stats          *DDoSStats
	
	// Background tasks
	cleanup        *time.Ticker
	shutdown       chan struct{}
	wg             sync.WaitGroup
}

// IPLimiter tracks rate limiting for a specific IP
type IPLimiter struct {
	limiter      *rate.Limiter
	lastAccess   time.Time
	violations   int64
	totalRequests int64
}

// RequestCounter tracks request counts for attack detection
type RequestCounter struct {
	count       int64
	firstSeen   time.Time
	lastSeen    time.Time
	windowStart time.Time
}

// BlockedIP represents a blocked IP address
type BlockedIP struct {
	IP           string
	BlockedAt    time.Time
	ExpiresAt    time.Time
	Reason       string
	AttackType   string
	RequestCount int64
}

// SuspiciousActivity tracks suspicious behavior patterns
type SuspiciousActivity struct {
	IP               string
	FirstDetected    time.Time
	LastSeen         time.Time
	SuspiciousEvents []SuspiciousEvent
	ThreatScore      int
}

// SuspiciousEvent represents a suspicious event
type SuspiciousEvent struct {
	Timestamp   time.Time
	EventType   string
	Details     string
	Severity    int
}

// NetworkList manages IP whitelists and blacklists
type NetworkList struct {
	ips     map[string]bool
	cidrs   []*net.IPNet
	mu      sync.RWMutex
}

// GeolocationFilter provides geolocation-based filtering
type GeolocationFilter struct {
	enabled          bool
	allowedCountries map[string]bool
	blockedCountries map[string]bool
	// In production, integrate with MaxMind GeoIP or similar
}

// Challenge represents a security challenge
type Challenge struct {
	ID          string
	IP          string
	Type        string
	CreatedAt   time.Time
	ExpiresAt   time.Time
	Attempts    int
	Solved      bool
	Token       string
}

// DDoSStats tracks DDoS protection statistics
type DDoSStats struct {
	TotalRequests      int64     `json:"total_requests"`
	BlockedRequests    int64     `json:"blocked_requests"`
	SuspiciousRequests int64     `json:"suspicious_requests"`
	AttacksDetected    int64     `json:"attacks_detected"`
	IPsBlocked         int64     `json:"ips_blocked"`
	ActiveBlocks       int64     `json:"active_blocks"`
	ChallengesSent     int64     `json:"challenges_sent"`
	ChallengesSolved   int64     `json:"challenges_solved"`
	LastAttack         time.Time `json:"last_attack"`
	LastBlock          time.Time `json:"last_block"`
}

// NewDDoSProtector creates a new DDoS protector
func NewDDoSProtector(config *DDoSProtectionConfig, logger *slog.Logger) (*DDoSProtector, error) {
	if config == nil {
		config = DefaultDDoSProtectionConfig()
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid DDoS protection config: %w", err)
	}

	ddp := &DDoSProtector{
		config:    config,
		logger:    logger.With(slog.String("component", "ddos_protector")),
		stats:     &DDoSStats{},
		cleanup:   time.NewTicker(1 * time.Minute),
		shutdown:  make(chan struct{}),
	}

	// Initialize global rate limiter
	ddp.globalLimiter = rate.NewLimiter(rate.Limit(config.GlobalRateLimit), config.BurstSize)

	// Initialize network lists
	if err := ddp.initializeNetworkLists(); err != nil {
		return nil, fmt.Errorf("failed to initialize network lists: %w", err)
	}

	// Initialize geolocation filter
	if config.EnableGeoFiltering {
		ddp.geoFilter = &GeolocationFilter{
			enabled:          true,
			allowedCountries: make(map[string]bool),
			blockedCountries: make(map[string]bool),
		}
		
		for _, country := range config.AllowedCountries {
			ddp.geoFilter.allowedCountries[strings.ToUpper(country)] = true
		}
		for _, country := range config.BlockedCountries {
			ddp.geoFilter.blockedCountries[strings.ToUpper(country)] = true
		}
	}

	// Start background tasks
	ddp.wg.Add(1)
	go ddp.backgroundCleanup()

	logger.Info("DDoS protector initialized",
		slog.Int("global_rate_limit", config.GlobalRateLimit),
		slog.Int("per_ip_rate_limit", config.PerIPRateLimit),
		slog.Int("max_concurrent_conns", config.MaxConcurrentConns),
		slog.Bool("geo_filtering", config.EnableGeoFiltering))

	return ddp, nil
}

// initializeNetworkLists initializes whitelist and blacklist
func (ddp *DDoSProtector) initializeNetworkLists() error {
	// Initialize whitelist
	ddp.whitelist = &NetworkList{
		ips:   make(map[string]bool),
		cidrs: make([]*net.IPNet, 0),
	}

	for _, ip := range ddp.config.WhitelistIPs {
		ddp.whitelist.ips[ip] = true
	}

	for _, cidr := range ddp.config.WhitelistCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid whitelist CIDR %s: %w", cidr, err)
		}
		ddp.whitelist.cidrs = append(ddp.whitelist.cidrs, ipNet)
	}

	// Initialize blacklist
	ddp.blacklist = &NetworkList{
		ips:   make(map[string]bool),
		cidrs: make([]*net.IPNet, 0),
	}

	for _, ip := range ddp.config.BlacklistIPs {
		ddp.blacklist.ips[ip] = true
	}

	for _, cidr := range ddp.config.BlacklistCIDRs {
		_, ipNet, err := net.ParseCIDR(cidr)
		if err != nil {
			return fmt.Errorf("invalid blacklist CIDR %s: %w", cidr, err)
		}
		ddp.blacklist.cidrs = append(ddp.blacklist.cidrs, ipNet)
	}

	return nil
}

// ProcessRequest processes an incoming request through DDoS protection
func (ddp *DDoSProtector) ProcessRequest(r *http.Request) (bool, string, error) {
	clientIP := getClientIP(r)
	if clientIP == "" {
		return false, "unable to determine client IP", fmt.Errorf("no client IP")
	}

	atomic.AddInt64(&ddp.stats.TotalRequests, 1)

	// Check blacklist first
	if ddp.isBlacklisted(clientIP) {
		atomic.AddInt64(&ddp.stats.BlockedRequests, 1)
		ddosBlockedRequests.WithLabelValues("blacklisted", clientIP).Inc()
		return false, "IP is blacklisted", nil
	}

	// Check if IP is currently blocked
	if blocked, reason := ddp.isBlocked(clientIP); blocked {
		atomic.AddInt64(&ddp.stats.BlockedRequests, 1)
		ddosBlockedRequests.WithLabelValues("blocked", clientIP).Inc()
		return false, reason, nil
	}

	// Check whitelist (bypass other checks)
	if ddp.isWhitelisted(clientIP) {
		return true, "whitelisted", nil
	}

	// Check geolocation filter
	if ddp.config.EnableGeoFiltering && !ddp.isGeoAllowed(clientIP) {
		atomic.AddInt64(&ddp.stats.BlockedRequests, 1)
		ddosBlockedRequests.WithLabelValues("geo_blocked", clientIP).Inc()
		return false, "geolocation not allowed", nil
	}

	// Check global rate limit
	if !ddp.globalLimiter.Allow() {
		atomic.AddInt64(&ddp.stats.BlockedRequests, 1)
		ddosBlockedRequests.WithLabelValues("global_rate_limit", clientIP).Inc()
		return false, "global rate limit exceeded", nil
	}

	// Check per-IP rate limit
	if !ddp.checkIPRateLimit(clientIP) {
		atomic.AddInt64(&ddp.stats.BlockedRequests, 1)
		rateLimitViolations.WithLabelValues("per_ip", clientIP).Inc()
		return false, "IP rate limit exceeded", nil
	}

	// Check connection limits
	if !ddp.checkConnectionLimits(clientIP) {
		atomic.AddInt64(&ddp.stats.BlockedRequests, 1)
		ddosBlockedRequests.WithLabelValues("connection_limit", clientIP).Inc()
		return false, "connection limit exceeded", nil
	}

	// Update attack detection counters
	ddp.updateAttackDetection(clientIP)

	// Check for suspicious activity
	if ddp.detectSuspiciousActivity(clientIP, r) {
		atomic.AddInt64(&ddp.stats.SuspiciousRequests, 1)
		
		// Send challenge if enabled
		if ddp.config.EnableCaptcha || ddp.config.EnableRateProof {
			if ddp.shouldChallenge(clientIP) {
				return false, "security challenge required", nil
			}
		}
	}

	// Check for DDoS attack patterns
	if ddp.detectDDoSAttack(clientIP) {
		ddp.blockIP(clientIP, "DDoS attack detected", "volumetric")
		atomic.AddInt64(&ddp.stats.AttacksDetected, 1)
		atomic.AddInt64(&ddp.stats.BlockedRequests, 1)
		ddosAttacksDetected.WithLabelValues("volumetric", clientIP).Inc()
		ddp.stats.LastAttack = time.Now()
		
		// Send alert if enabled
		if ddp.config.EnableAlerts {
			go ddp.sendSecurityAlert("DDoS Attack Detected", clientIP, "volumetric")
		}
		
		return false, "DDoS attack detected, IP blocked", nil
	}

	return true, "allowed", nil
}

// isWhitelisted checks if an IP is whitelisted
func (ddp *DDoSProtector) isWhitelisted(ip string) bool {
	ddp.whitelist.mu.RLock()
	defer ddp.whitelist.mu.RUnlock()

	// Check exact IP match
	if ddp.whitelist.ips[ip] {
		return true
	}

	// Check CIDR ranges
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range ddp.whitelist.cidrs {
		if cidr.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// isBlacklisted checks if an IP is blacklisted
func (ddp *DDoSProtector) isBlacklisted(ip string) bool {
	ddp.blacklist.mu.RLock()
	defer ddp.blacklist.mu.RUnlock()

	// Check exact IP match
	if ddp.blacklist.ips[ip] {
		return true
	}

	// Check CIDR ranges
	parsedIP := net.ParseIP(ip)
	if parsedIP == nil {
		return false
	}

	for _, cidr := range ddp.blacklist.cidrs {
		if cidr.Contains(parsedIP) {
			return true
		}
	}

	return false
}

// isBlocked checks if an IP is currently blocked
func (ddp *DDoSProtector) isBlocked(ip string) (bool, string) {
	if blockedInfo, exists := ddp.blockedIPs.Load(ip); exists {
		blocked := blockedInfo.(*BlockedIP)
		if time.Now().Before(blocked.ExpiresAt) {
			return true, fmt.Sprintf("IP blocked until %v: %s", blocked.ExpiresAt, blocked.Reason)
		}
		// Block expired, remove it
		ddp.blockedIPs.Delete(ip)
		atomic.AddInt64(&ddp.stats.ActiveBlocks, -1)
	}
	return false, ""
}

// isGeoAllowed checks if an IP's geolocation is allowed
func (ddp *DDoSProtector) isGeoAllowed(ip string) bool {
	if ddp.geoFilter == nil || !ddp.geoFilter.enabled {
		return true
	}

	// In production, integrate with GeoIP database
	// For now, return true (allow all)
	return true
}

// checkIPRateLimit checks per-IP rate limiting
func (ddp *DDoSProtector) checkIPRateLimit(ip string) bool {
	now := time.Now()

	// Get or create IP limiter
	limiterInfo, _ := ddp.ipLimiters.LoadOrStore(ip, &IPLimiter{
		limiter:    rate.NewLimiter(rate.Limit(ddp.config.PerIPRateLimit), ddp.config.BurstSize),
		lastAccess: now,
	})

	ipLimiter := limiterInfo.(*IPLimiter)
	ipLimiter.lastAccess = now
	atomic.AddInt64(&ipLimiter.totalRequests, 1)

	if !ipLimiter.limiter.Allow() {
		atomic.AddInt64(&ipLimiter.violations, 1)
		return false
	}

	return true
}

// checkConnectionLimits checks connection limits
func (ddp *DDoSProtector) checkConnectionLimits(ip string) bool {
	// Check global connection limit
	currentConns := atomic.LoadInt64(&ddp.activeConns)
	if currentConns >= int64(ddp.config.MaxConcurrentConns) {
		return false
	}

	// Check per-IP connection limit
	if connCount, exists := ddp.connsByIP.Load(ip); exists {
		if connCount.(int64) >= int64(ddp.config.MaxConnsPerIP) {
			return false
		}
	}

	return true
}

// updateAttackDetection updates attack detection counters
func (ddp *DDoSProtector) updateAttackDetection(ip string) {
	now := time.Now()
	
	counterInfo, _ := ddp.requestCounts.LoadOrStore(ip, &RequestCounter{
		firstSeen:   now,
		windowStart: now,
	})
	
	counter := counterInfo.(*RequestCounter)
	counter.lastSeen = now
	
	// Reset counter if window has passed
	if now.Sub(counter.windowStart) > ddp.config.DetectionWindow {
		counter.count = 0
		counter.windowStart = now
	}
	
	atomic.AddInt64(&counter.count, 1)
}

// detectSuspiciousActivity detects suspicious behavior patterns
func (ddp *DDoSProtector) detectSuspiciousActivity(ip string, r *http.Request) bool {
	// Check for suspicious patterns
	suspicious := false
	
	// Check user agent
	userAgent := r.UserAgent()
	if userAgent == "" || strings.Contains(strings.ToLower(userAgent), "bot") {
		suspicious = true
	}
	
	// Check for suspicious headers
	if r.Header.Get("X-Forwarded-For") != "" && len(strings.Split(r.Header.Get("X-Forwarded-For"), ",")) > 3 {
		suspicious = true
	}
	
	// Check request patterns
	if strings.Contains(r.URL.Path, "..") || strings.Contains(r.URL.Path, "admin") {
		suspicious = true
	}
	
	if suspicious {
		now := time.Now()
		suspiciousInfo, _ := ddp.suspiciousIPs.LoadOrStore(ip, &SuspiciousActivity{
			IP:            ip,
			FirstDetected: now,
			ThreatScore:   0,
		})
		
		activity := suspiciousInfo.(*SuspiciousActivity)
		activity.LastSeen = now
		activity.ThreatScore++
		activity.SuspiciousEvents = append(activity.SuspiciousEvents, SuspiciousEvent{
			Timestamp: now,
			EventType: "suspicious_request",
			Details:   fmt.Sprintf("Path: %s, UA: %s", r.URL.Path, userAgent),
			Severity:  1,
		})
	}
	
	return suspicious
}

// detectDDoSAttack detects DDoS attack patterns
func (ddp *DDoSProtector) detectDDoSAttack(ip string) bool {
	if counterInfo, exists := ddp.requestCounts.Load(ip); exists {
		counter := counterInfo.(*RequestCounter)
		currentCount := atomic.LoadInt64(&counter.count)
		
		// Check if request count exceeds attack threshold
		if currentCount > int64(ddp.config.AttackThreshold) {
			return true
		}
	}
	
	return false
}

// shouldChallenge determines if a challenge should be sent
func (ddp *DDoSProtector) shouldChallenge(ip string) bool {
	if counterInfo, exists := ddp.requestCounts.Load(ip); exists {
		counter := counterInfo.(*RequestCounter)
		currentCount := atomic.LoadInt64(&counter.count)
		
		return currentCount > int64(ddp.config.ChallengeThreshold)
	}
	
	return false
}

// blockIP blocks an IP address
func (ddp *DDoSProtector) blockIP(ip, reason, attackType string) {
	now := time.Now()
	expiresAt := now.Add(ddp.config.BlockDuration)
	
	blocked := &BlockedIP{
		IP:         ip,
		BlockedAt:  now,
		ExpiresAt:  expiresAt,
		Reason:     reason,
		AttackType: attackType,
	}
	
	if counterInfo, exists := ddp.requestCounts.Load(ip); exists {
		counter := counterInfo.(*RequestCounter)
		blocked.RequestCount = atomic.LoadInt64(&counter.count)
	}
	
	ddp.blockedIPs.Store(ip, blocked)
	atomic.AddInt64(&ddp.stats.IPsBlocked, 1)
	atomic.AddInt64(&ddp.stats.ActiveBlocks, 1)
	ddp.stats.LastBlock = now
	
	ddp.logger.Warn("IP blocked",
		slog.String("ip", ip),
		slog.String("reason", reason),
		slog.String("attack_type", attackType),
		slog.Time("expires_at", expiresAt))
}

// sendSecurityAlert sends a security alert
func (ddp *DDoSProtector) sendSecurityAlert(alertType, ip, details string) {
	if ddp.config.AlertWebhook == "" {
		return
	}
	
	// In production, send alert to webhook
	ddp.logger.Error("Security alert",
		slog.String("type", alertType),
		slog.String("ip", ip),
		slog.String("details", details))
}

// CreateHTTPMiddleware creates HTTP middleware for DDoS protection
func (ddp *DDoSProtector) CreateHTTPMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Track connection
			atomic.AddInt64(&ddp.activeConns, 1)
			clientIP := getClientIP(r)
			if clientIP != "" {
				connCount, _ := ddp.connsByIP.LoadOrStore(clientIP, int64(0))
				atomic.AddInt64(connCount.(*int64), 1)
				activeConnections.WithLabelValues("http").Inc()
			}
			
			defer func() {
				atomic.AddInt64(&ddp.activeConns, -1)
				if clientIP != "" {
					if connCount, exists := ddp.connsByIP.Load(clientIP); exists {
						atomic.AddInt64(connCount.(*int64), -1)
					}
					activeConnections.WithLabelValues("http").Dec()
				}
			}()
			
			// Process request through DDoS protection
			allowed, reason, err := ddp.ProcessRequest(r)
			if err != nil {
				ddp.logger.Error("DDoS protection error", slog.String("error", err.Error()))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}
			
			if !allowed {
				ddp.logger.Warn("Request blocked by DDoS protection",
					slog.String("client_ip", clientIP),
					slog.String("reason", reason),
					slog.String("path", r.URL.Path),
					slog.String("user_agent", r.UserAgent()))
				
				// Add security headers
				w.Header().Set("X-Content-Type-Options", "nosniff")
				w.Header().Set("X-Frame-Options", "DENY")
				w.Header().Set("Retry-After", "60")
				
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				return
			}
			
			next.ServeHTTP(w, r)
		})
	}
}

// backgroundCleanup performs cleanup tasks
func (ddp *DDoSProtector) backgroundCleanup() {
	defer ddp.wg.Done()
	
	for {
		select {
		case <-ddp.cleanup.C:
			ddp.cleanupExpiredEntries()
		case <-ddp.shutdown:
			return
		}
	}
}

// cleanupExpiredEntries cleans up expired entries
func (ddp *DDoSProtector) cleanupExpiredEntries() {
	now := time.Now()
	cleanupThreshold := now.Add(-1 * time.Hour)
	
	// Clean up IP limiters
	ddp.ipLimiters.Range(func(key, value interface{}) bool {
		limiter := value.(*IPLimiter)
		if limiter.lastAccess.Before(cleanupThreshold) {
			ddp.ipLimiters.Delete(key)
		}
		return true
	})
	
	// Clean up request counters
	ddp.requestCounts.Range(func(key, value interface{}) bool {
		counter := value.(*RequestCounter)
		if counter.lastSeen.Before(cleanupThreshold) {
			ddp.requestCounts.Delete(key)
		}
		return true
	})
	
	// Clean up blocked IPs
	ddp.blockedIPs.Range(func(key, value interface{}) bool {
		blocked := value.(*BlockedIP)
		if now.After(blocked.ExpiresAt) {
			ddp.blockedIPs.Delete(key)
			atomic.AddInt64(&ddp.stats.ActiveBlocks, -1)
		}
		return true
	})
	
	// Clean up suspicious IPs
	ddp.suspiciousIPs.Range(func(key, value interface{}) bool {
		activity := value.(*SuspiciousActivity)
		if activity.LastSeen.Before(cleanupThreshold) {
			ddp.suspiciousIPs.Delete(key)
		}
		return true
	})
}

// GetStats returns DDoS protection statistics
func (ddp *DDoSProtector) GetStats() *DDoSStats {
	return ddp.stats
}

// Close shuts down the DDoS protector
func (ddp *DDoSProtector) Close() error {
	ddp.cleanup.Stop()
	close(ddp.shutdown)
	ddp.wg.Wait()
	
	ddp.logger.Info("DDoS protector shut down")
	return nil
}

// DefaultDDoSProtectionConfig returns default DDoS protection configuration
func DefaultDDoSProtectionConfig() *DDoSProtectionConfig {
	return &DDoSProtectionConfig{
		GlobalRateLimit:     1000,
		PerIPRateLimit:      50,
		BurstSize:          100,
		WindowSize:         1 * time.Minute,
		MaxConcurrentConns: 10000,
		MaxConnsPerIP:      100,
		ConnectionTimeout:  30 * time.Second,
		SuspiciousThreshold: 100,
		AttackThreshold:    200,
		DetectionWindow:    1 * time.Minute,
		BlockDuration:      15 * time.Minute,
		TempBanDuration:    1 * time.Hour,
		MaxBlockedIPs:      10000,
		EnableGeoFiltering: false,
		EnableCaptcha:      false,
		EnableRateProof:    false,
		ChallengeThreshold: 50,
		EnableAlerts:       true,
		MetricsRetention:   24 * time.Hour,
	}
}

// Validate validates the DDoS protection configuration
func (config *DDoSProtectionConfig) Validate() error {
	if config.GlobalRateLimit <= 0 {
		return fmt.Errorf("global rate limit must be positive")
	}
	if config.PerIPRateLimit <= 0 {
		return fmt.Errorf("per-IP rate limit must be positive")
	}
	if config.BurstSize <= 0 {
		return fmt.Errorf("burst size must be positive")
	}
	if config.MaxConcurrentConns <= 0 {
		return fmt.Errorf("max concurrent connections must be positive")
	}
	return nil
}

// Helper function to get client IP (reuse from existing implementation)
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (most common)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		ip := strings.TrimSpace(ips[0])
		if ip != "" && net.ParseIP(ip) != nil {
			return ip
		}
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		ip := strings.TrimSpace(xri)
		if ip != "" && net.ParseIP(ip) != nil {
			return ip
		}
	}

	// Check CF-Connecting-IP header (Cloudflare)
	if cfip := r.Header.Get("CF-Connecting-IP"); cfip != "" {
		ip := strings.TrimSpace(cfip)
		if ip != "" && net.ParseIP(ip) != nil {
			return ip
		}
	}

	// Fall back to RemoteAddr
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		if net.ParseIP(r.RemoteAddr) != nil {
			return r.RemoteAddr
		}
		return ""
	}

	if net.ParseIP(ip) != nil {
		return ip
	}

	return ""
}