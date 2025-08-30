package security

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"

	"github.com/nephio-project/nephoran-intent-operator/pkg/logging"
)

// RateLimitConfig holds rate limiting configuration.

type RateLimitConfig struct {
	Enabled bool `json:"enabled"`

	Type RateLimitType `json:"type"`

	DefaultLimits *RateLimitPolicy `json:"default_limits"`

	PolicyLimits map[string]*RateLimitPolicy `json:"policy_limits"`

	UserLimits map[string]*RateLimitPolicy `json:"user_limits"`

	IPLimits map[string]*RateLimitPolicy `json:"ip_limits"`

	EndpointLimits map[string]*RateLimitPolicy `json:"endpoint_limits"`

	GlobalLimit *RateLimitPolicy `json:"global_limit"`

	BurstMultiplier float64 `json:"burst_multiplier"`

	WindowType WindowType `json:"window_type"`

	DistributedConfig *DistributedConfig `json:"distributed_config"`

	AdaptiveConfig *AdaptiveConfig `json:"adaptive_config"`

	WhitelistEnabled bool `json:"whitelist_enabled"`

	Whitelist []string `json:"whitelist"`

	BlacklistEnabled bool `json:"blacklist_enabled"`

	Blacklist []string `json:"blacklist"`

	BypassHeaders []string `json:"bypass_headers"`

	ResponseHeaders bool `json:"response_headers"`

	DDoSProtection *DDoSProtectionConfig `json:"ddos_protection"`
}

// RateLimitType represents the type of rate limiting.

type RateLimitType string

const (

	// RateLimitTypeLocal holds ratelimittypelocal value.

	RateLimitTypeLocal RateLimitType = "local"

	// RateLimitTypeDistributed holds ratelimittypedistributed value.

	RateLimitTypeDistributed RateLimitType = "distributed"

	// RateLimitTypeHybrid holds ratelimittypehybrid value.

	RateLimitTypeHybrid RateLimitType = "hybrid"
)

// WindowType represents the rate limit window type.

type WindowType string

const (

	// WindowTypeFixed holds windowtypefixed value.

	WindowTypeFixed WindowType = "fixed"

	// WindowTypeSliding holds windowtypesliding value.

	WindowTypeSliding WindowType = "sliding"

	// WindowTypeToken holds windowtypetoken value.

	WindowTypeToken WindowType = "token"

	// WindowTypeLeaky holds windowtypeleaky value.

	WindowTypeLeaky WindowType = "leaky"
)

// RateLimitPolicy defines a rate limit policy.

type RateLimitPolicy struct {
	Name string `json:"name"`

	RequestsPerMin int `json:"requests_per_min"`

	RequestsPerHour int `json:"requests_per_hour"`

	RequestsPerDay int `json:"requests_per_day"`

	BurstSize int `json:"burst_size"`

	WindowSize time.Duration `json:"window_size"`

	Priority int `json:"priority"`

	CostPerRequest int `json:"cost_per_request"`

	MaxConcurrent int `json:"max_concurrent"`

	QueueSize int `json:"queue_size"`

	QueueTimeout time.Duration `json:"queue_timeout"`

	BackoffFactor float64 `json:"backoff_factor"`

	BackoffMax time.Duration `json:"backoff_max"`

	ThrottleResponse *ThrottleResponse `json:"throttle_response"`
}

// DistributedConfig holds distributed rate limiting configuration.

type DistributedConfig struct {
	Backend DistributedBackend `json:"backend"`

	RedisConfig *RedisConfig `json:"redis_config"`

	SyncInterval time.Duration `json:"sync_interval"`

	FailoverMode FailoverMode `json:"failover_mode"`

	ConsistencyLevel ConsistencyLevel `json:"consistency_level"`

	Sharding bool `json:"sharding"`

	ShardCount int `json:"shard_count"`
}

// DistributedBackend represents the backend for distributed rate limiting.

type DistributedBackend string

const (

	// BackendRedis holds backendredis value.

	BackendRedis DistributedBackend = "redis"

	// BackendMemcached holds backendmemcached value.

	BackendMemcached DistributedBackend = "memcached"

	// BackendEtcd holds backendetcd value.

	BackendEtcd DistributedBackend = "etcd"

	// BackendConsul holds backendconsul value.

	BackendConsul DistributedBackend = "consul"
)

// RedisConfig holds Redis configuration.

type RedisConfig struct {
	Addresses []string `json:"addresses"`

	Password string `json:"password"`

	DB int `json:"db"`

	MaxRetries int `json:"max_retries"`

	PoolSize int `json:"pool_size"`

	ReadTimeout time.Duration `json:"read_timeout"`

	WriteTimeout time.Duration `json:"write_timeout"`

	ClusterMode bool `json:"cluster_mode"`

	SentinelMaster string `json:"sentinel_master"`
}

// FailoverMode represents the failover mode for distributed limiting.

type FailoverMode string

const (

	// FailoverLocal holds failoverlocal value.

	FailoverLocal FailoverMode = "local"

	// FailoverReject holds failoverreject value.

	FailoverReject FailoverMode = "reject"

	// FailoverDegrade holds failoverdegrade value.

	FailoverDegrade FailoverMode = "degrade"
)

// ConsistencyLevel represents the consistency level.

type ConsistencyLevel string

const (

	// ConsistencyStrong holds consistencystrong value.

	ConsistencyStrong ConsistencyLevel = "strong"

	// ConsistencyEventual holds consistencyeventual value.

	ConsistencyEventual ConsistencyLevel = "eventual"

	// ConsistencyWeak holds consistencyweak value.

	ConsistencyWeak ConsistencyLevel = "weak"
)

// AdaptiveConfig holds adaptive rate limiting configuration.

type AdaptiveConfig struct {
	Enabled bool `json:"enabled"`

	TargetLatency time.Duration `json:"target_latency"`

	TargetErrorRate float64 `json:"target_error_rate"`

	AdjustmentFactor float64 `json:"adjustment_factor"`

	MinLimit int `json:"min_limit"`

	MaxLimit int `json:"max_limit"`

	SampleWindow time.Duration `json:"sample_window"`

	AdjustInterval time.Duration `json:"adjust_interval"`
}

// DDoSProtectionConfig holds DDoS protection configuration.

type DDoSProtectionConfig struct {
	Enabled bool `json:"enabled"`

	ThresholdMultiplier float64 `json:"threshold_multiplier"`

	DetectionWindow time.Duration `json:"detection_window"`

	BlockDuration time.Duration `json:"block_duration"`

	SynFloodProtection bool `json:"syn_flood_protection"`

	UDPFloodProtection bool `json:"udp_flood_protection"`

	HTTPFloodProtection bool `json:"http_flood_protection"`

	ChallengeEnabled bool `json:"challenge_enabled"`

	ChallengeType ChallengeType `json:"challenge_type"`
}

// ChallengeType represents the type of challenge for DDoS protection.

type ChallengeType string

const (

	// ChallengeCaptcha holds challengecaptcha value.

	ChallengeCaptcha ChallengeType = "captcha"

	// ChallengeProofOfWork holds challengeproofofwork value.

	ChallengeProofOfWork ChallengeType = "proof_of_work"

	// ChallengeJavaScript holds challengejavascript value.

	ChallengeJavaScript ChallengeType = "javascript"
)

// ThrottleResponse defines how to respond when rate limited.

type ThrottleResponse struct {
	StatusCode int `json:"status_code"`

	Message string `json:"message"`

	Headers map[string]string `json:"headers"`

	RetryAfter bool `json:"retry_after"`

	IncludeDetails bool `json:"include_details"`
}

// RateLimiter manages rate limiting.

type RateLimiter struct {
	config *RateLimitConfig

	logger *logging.StructuredLogger

	localLimiters map[string]*rate.Limiter

	distributedStore DistributedStore

	adaptiveManager *AdaptiveManager

	ddosProtector *DDoSProtector

	mu sync.RWMutex

	stats *RateLimitStats

	whitelist map[string]bool

	blacklist map[string]bool
}

// DistributedStore interface for distributed rate limiting.

type DistributedStore interface {
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)

	Get(ctx context.Context, key string) (int64, error)

	Reset(ctx context.Context, key string) error

	SetWithExpiry(ctx context.Context, key string, value int64, expiry time.Duration) error

	Close() error
}

// AdaptiveManager manages adaptive rate limiting.

type AdaptiveManager struct {
	config *AdaptiveConfig

	logger *logging.StructuredLogger

	metrics *AdaptiveMetrics

	adjustTicker *time.Ticker

	mu sync.RWMutex
}

// AdaptiveMetrics tracks metrics for adaptive rate limiting.

type AdaptiveMetrics struct {
	mu sync.RWMutex

	RequestCount int64

	ErrorCount int64

	TotalLatency time.Duration

	CurrentLimit int

	LastAdjustment time.Time
}

// DDoSProtector provides DDoS protection.

type DDoSProtector struct {
	config *DDoSProtectionConfig

	logger *logging.StructuredLogger

	blockedIPs map[string]time.Time

	requestCounts map[string]*RequestCounter

	mu sync.RWMutex
}

// RequestCounter tracks request counts for DDoS detection.

type RequestCounter struct {
	Count int64

	FirstSeen time.Time

	LastSeen time.Time

	Suspicious bool
}

// RateLimitStats tracks rate limiting statistics.

type RateLimitStats struct {
	mu sync.RWMutex

	TotalRequests int64

	AllowedRequests int64

	ThrottledRequests int64

	BlockedRequests int64

	CurrentActive int64

	ThrottlesByKey map[string]int64

	LastReset time.Time
}

// RateLimitResult represents the result of rate limit check.

type RateLimitResult struct {
	Allowed bool `json:"allowed"`

	Reason string `json:"reason,omitempty"`

	Limit int `json:"limit"`

	Remaining int `json:"remaining"`

	Reset time.Time `json:"reset"`

	RetryAfter time.Duration `json:"retry_after,omitempty"`

	Key string `json:"key"`

	Cost int `json:"cost"`
}

// NewRateLimiter creates a new rate limiter.

func NewRateLimiter(config *RateLimitConfig, logger *logging.StructuredLogger) (*RateLimiter, error) {

	if config == nil {

		return nil, errors.New("rate limit config is required")

	}

	rl := &RateLimiter{

		config: config,

		logger: logger,

		localLimiters: make(map[string]*rate.Limiter),

		whitelist: make(map[string]bool),

		blacklist: make(map[string]bool),

		stats: &RateLimitStats{

			ThrottlesByKey: make(map[string]int64),

			LastReset: time.Now(),
		},
	}

	// Initialize distributed store if configured.

	if config.Type == RateLimitTypeDistributed || config.Type == RateLimitTypeHybrid {

		if err := rl.initializeDistributedStore(); err != nil {

			return nil, fmt.Errorf("failed to initialize distributed store: %w", err)

		}

	}

	// Initialize adaptive manager if configured.

	if config.AdaptiveConfig != nil && config.AdaptiveConfig.Enabled {

		rl.adaptiveManager = NewAdaptiveManager(config.AdaptiveConfig, logger)

		rl.adaptiveManager.Start()

	}

	// Initialize DDoS protector if configured.

	if config.DDoSProtection != nil && config.DDoSProtection.Enabled {

		rl.ddosProtector = NewDDoSProtector(config.DDoSProtection, logger)

		go rl.ddosProtector.Start()

	}

	// Load whitelist and blacklist.

	rl.loadLists()

	// Start statistics reset timer.

	go rl.periodicStatsReset()

	return rl, nil

}

// initializeDistributedStore initializes the distributed store.

func (rl *RateLimiter) initializeDistributedStore() error {

	if rl.config.DistributedConfig == nil {

		return errors.New("distributed config is required")

	}

	switch rl.config.DistributedConfig.Backend {

	case BackendRedis:

		store, err := NewRedisStore(rl.config.DistributedConfig.RedisConfig)

		if err != nil {

			return err

		}

		rl.distributedStore = store

	default:

		return fmt.Errorf("unsupported backend: %s", rl.config.DistributedConfig.Backend)

	}

	return nil

}

// loadLists loads whitelist and blacklist.

func (rl *RateLimiter) loadLists() {

	// Load whitelist.

	if rl.config.WhitelistEnabled {

		for _, entry := range rl.config.Whitelist {

			rl.whitelist[entry] = true

		}

	}

	// Load blacklist.

	if rl.config.BlacklistEnabled {

		for _, entry := range rl.config.Blacklist {

			rl.blacklist[entry] = true

		}

	}

}

// CheckLimit checks if a request should be rate limited.

func (rl *RateLimiter) CheckLimit(ctx context.Context, r *http.Request) (*RateLimitResult, error) {

	// Extract client identifier.

	clientID := rl.getClientID(r)

	// Check blacklist.

	if rl.isBlacklisted(clientID) {

		rl.stats.mu.Lock()

		rl.stats.BlockedRequests++

		rl.stats.mu.Unlock()

		return &RateLimitResult{

			Allowed: false,

			Reason: "blacklisted",

			Key: clientID,
		}, nil

	}

	// Check whitelist.

	if rl.isWhitelisted(clientID) {

		rl.stats.mu.Lock()

		rl.stats.AllowedRequests++

		rl.stats.mu.Unlock()

		return &RateLimitResult{

			Allowed: true,

			Key: clientID,
		}, nil

	}

	// Check bypass headers.

	if rl.hasBypassHeader(r) {

		return &RateLimitResult{

			Allowed: true,

			Key: clientID,
		}, nil

	}

	// Check DDoS protection.

	if rl.ddosProtector != nil {

		if blocked := rl.ddosProtector.IsBlocked(clientID); blocked {

			return &RateLimitResult{

				Allowed: false,

				Reason: "ddos_protection",

				Key: clientID,
			}, nil

		}

	}

	// Get appropriate policy.

	policy := rl.getPolicy(r, clientID)

	if policy == nil {

		policy = rl.config.DefaultLimits

	}

	// Check rate limit based on type.

	switch rl.config.Type {

	case RateLimitTypeLocal:

		return rl.checkLocalLimit(clientID, policy)

	case RateLimitTypeDistributed:

		return rl.checkDistributedLimit(ctx, clientID, policy)

	case RateLimitTypeHybrid:

		return rl.checkHybridLimit(ctx, clientID, policy)

	default:

		return rl.checkLocalLimit(clientID, policy)

	}

}

// checkLocalLimit checks rate limit using local limiter.

func (rl *RateLimiter) checkLocalLimit(key string, policy *RateLimitPolicy) (*RateLimitResult, error) {

	rl.mu.Lock()

	limiter, exists := rl.localLimiters[key]

	if !exists {

		// Create new limiter.

		rateLimit := rate.Limit(float64(policy.RequestsPerMin) / 60.0)

		burst := policy.BurstSize

		if burst == 0 {

			burst = int(float64(policy.RequestsPerMin) * rl.config.BurstMultiplier)

		}

		limiter = rate.NewLimiter(rateLimit, burst)

		rl.localLimiters[key] = limiter

	}

	rl.mu.Unlock()

	// Check if request is allowed.

	allowed := limiter.Allow()

	// Update statistics.

	rl.stats.mu.Lock()

	rl.stats.TotalRequests++

	if allowed {

		rl.stats.AllowedRequests++

	} else {

		rl.stats.ThrottledRequests++

		rl.stats.ThrottlesByKey[key]++

	}

	rl.stats.mu.Unlock()

	result := &RateLimitResult{

		Allowed: allowed,

		Key: key,

		Limit: policy.RequestsPerMin,

		Remaining: int(limiter.Tokens()),

		Cost: policy.CostPerRequest,
	}

	if !allowed {

		result.Reason = "rate_limit_exceeded"

		result.RetryAfter = time.Second / time.Duration(policy.RequestsPerMin/60)

	}

	return result, nil

}

// checkDistributedLimit checks rate limit using distributed store.

func (rl *RateLimiter) checkDistributedLimit(ctx context.Context, key string, policy *RateLimitPolicy) (*RateLimitResult, error) {

	if rl.distributedStore == nil {

		// Fallback to local if distributed store is unavailable.

		if rl.config.DistributedConfig.FailoverMode == FailoverLocal {

			return rl.checkLocalLimit(key, policy)

		}

		return &RateLimitResult{

			Allowed: false,

			Reason: "distributed_store_unavailable",

			Key: key,
		}, errors.New("distributed store unavailable")

	}

	// Increment counter in distributed store.

	count, err := rl.distributedStore.Increment(ctx, key, policy.WindowSize)

	if err != nil {

		rl.logger.Error("failed to increment distributed counter",

			slog.String("key", key),

			slog.String("error", err.Error()))

		// Handle failover.

		switch rl.config.DistributedConfig.FailoverMode {

		case FailoverLocal:

			return rl.checkLocalLimit(key, policy)

		case FailoverReject:

			return &RateLimitResult{

				Allowed: false,

				Reason: "distributed_store_error",

				Key: key,
			}, err

		default:

			// Degrade - allow with reduced limits.

			degradedPolicy := *policy

			degradedPolicy.RequestsPerMin = policy.RequestsPerMin / 2

			return rl.checkLocalLimit(key, &degradedPolicy)

		}

	}

	allowed := count <= int64(policy.RequestsPerMin)

	// Update statistics.

	rl.stats.mu.Lock()

	rl.stats.TotalRequests++

	if allowed {

		rl.stats.AllowedRequests++

	} else {

		rl.stats.ThrottledRequests++

		rl.stats.ThrottlesByKey[key]++

	}

	rl.stats.mu.Unlock()

	result := &RateLimitResult{

		Allowed: allowed,

		Key: key,

		Limit: policy.RequestsPerMin,

		Remaining: maxInt(0, policy.RequestsPerMin-int(count)),

		Cost: policy.CostPerRequest,
	}

	if !allowed {

		result.Reason = "rate_limit_exceeded"

		result.RetryAfter = policy.WindowSize

	}

	return result, nil

}

// checkHybridLimit checks rate limit using both local and distributed.

func (rl *RateLimiter) checkHybridLimit(ctx context.Context, key string, policy *RateLimitPolicy) (*RateLimitResult, error) {

	// Check local limit first (fast path).

	localResult, err := rl.checkLocalLimit(key, policy)

	if err != nil || !localResult.Allowed {

		return localResult, err

	}

	// Check distributed limit (authoritative).

	return rl.checkDistributedLimit(ctx, key, policy)

}

// getClientID extracts client identifier from request.

func (rl *RateLimiter) getClientID(r *http.Request) string {

	// Try to get authenticated user ID.

	if userID := r.Context().Value("user_id"); userID != nil {

		return fmt.Sprintf("user:%v", userID)

	}

	// Try to get API key.

	if apiKey := r.Header.Get("X-API-Key"); apiKey != "" {

		return fmt.Sprintf("api:%s", apiKey)

	}

	// Fall back to IP address.

	ip := rl.getClientIP(r)

	return fmt.Sprintf("ip:%s", ip)

}

// getClientIP extracts client IP from request.

func (rl *RateLimiter) getClientIP(r *http.Request) string {

	// Check X-Forwarded-For header.

	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {

		parts := strings.Split(xff, ",")

		if len(parts) > 0 {

			return strings.TrimSpace(parts[0])

		}

	}

	// Check X-Real-IP header.

	if xri := r.Header.Get("X-Real-IP"); xri != "" {

		return xri

	}

	// Fall back to RemoteAddr.

	ip, _, _ := net.SplitHostPort(r.RemoteAddr)

	return ip

}

// getPolicy gets the appropriate rate limit policy.

func (rl *RateLimiter) getPolicy(r *http.Request, clientID string) *RateLimitPolicy {

	// Check endpoint-specific limits.

	endpoint := fmt.Sprintf("%s %s", r.Method, r.URL.Path)

	if policy, ok := rl.config.EndpointLimits[endpoint]; ok {

		return policy

	}

	// Check user-specific limits.

	if strings.HasPrefix(clientID, "user:") {

		userID := strings.TrimPrefix(clientID, "user:")

		if policy, ok := rl.config.UserLimits[userID]; ok {

			return policy

		}

	}

	// Check IP-specific limits.

	if strings.HasPrefix(clientID, "ip:") {

		ip := strings.TrimPrefix(clientID, "ip:")

		if policy, ok := rl.config.IPLimits[ip]; ok {

			return policy

		}

	}

	// Check policy-specific limits.

	if policyID := r.Header.Get("X-Policy-ID"); policyID != "" {

		if policy, ok := rl.config.PolicyLimits[policyID]; ok {

			return policy

		}

	}

	return rl.config.DefaultLimits

}

// isWhitelisted checks if client is whitelisted.

func (rl *RateLimiter) isWhitelisted(clientID string) bool {

	rl.mu.RLock()

	defer rl.mu.RUnlock()

	// Check exact match.

	if rl.whitelist[clientID] {

		return true

	}

	// Check IP range if applicable.

	if strings.HasPrefix(clientID, "ip:") {

		ip := strings.TrimPrefix(clientID, "ip:")

		for entry := range rl.whitelist {

			if strings.Contains(entry, "/") {

				// CIDR notation.

				_, ipnet, err := net.ParseCIDR(entry)

				if err == nil && ipnet.Contains(net.ParseIP(ip)) {

					return true

				}

			}

		}

	}

	return false

}

// isBlacklisted checks if client is blacklisted.

func (rl *RateLimiter) isBlacklisted(clientID string) bool {

	rl.mu.RLock()

	defer rl.mu.RUnlock()

	// Check exact match.

	if rl.blacklist[clientID] {

		return true

	}

	// Check IP range if applicable.

	if strings.HasPrefix(clientID, "ip:") {

		ip := strings.TrimPrefix(clientID, "ip:")

		for entry := range rl.blacklist {

			if strings.Contains(entry, "/") {

				// CIDR notation.

				_, ipnet, err := net.ParseCIDR(entry)

				if err == nil && ipnet.Contains(net.ParseIP(ip)) {

					return true

				}

			}

		}

	}

	return false

}

// hasBypassHeader checks if request has bypass header.

func (rl *RateLimiter) hasBypassHeader(r *http.Request) bool {

	for _, header := range rl.config.BypassHeaders {

		if r.Header.Get(header) != "" {

			return true

		}

	}

	return false

}

// SetResponseHeaders sets rate limit headers in response.

func (rl *RateLimiter) SetResponseHeaders(w http.ResponseWriter, result *RateLimitResult) {

	if !rl.config.ResponseHeaders {

		return

	}

	w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", result.Limit))

	w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", result.Remaining))

	w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", result.Reset.Unix()))

	if !result.Allowed && result.RetryAfter > 0 {

		w.Header().Set("Retry-After", fmt.Sprintf("%d", int(result.RetryAfter.Seconds())))

	}

}

// periodicStatsReset periodically resets statistics.

func (rl *RateLimiter) periodicStatsReset() {

	ticker := time.NewTicker(24 * time.Hour)

	defer ticker.Stop()

	for range ticker.C {

		rl.stats.mu.Lock()

		rl.stats.TotalRequests = 0

		rl.stats.AllowedRequests = 0

		rl.stats.ThrottledRequests = 0

		rl.stats.BlockedRequests = 0

		rl.stats.ThrottlesByKey = make(map[string]int64)

		rl.stats.LastReset = time.Now()

		rl.stats.mu.Unlock()

	}

}

// GetStats returns rate limiting statistics.

func (rl *RateLimiter) GetStats() *RateLimitStats {

	rl.stats.mu.RLock()

	defer rl.stats.mu.RUnlock()

	// Return a copy of stats.

	return &RateLimitStats{

		TotalRequests: rl.stats.TotalRequests,

		AllowedRequests: rl.stats.AllowedRequests,

		ThrottledRequests: rl.stats.ThrottledRequests,

		BlockedRequests: rl.stats.BlockedRequests,

		CurrentActive: rl.stats.CurrentActive,

		ThrottlesByKey: rl.stats.ThrottlesByKey,

		LastReset: rl.stats.LastReset,
	}

}

// Close closes the rate limiter.

func (rl *RateLimiter) Close() error {

	if rl.distributedStore != nil {

		return rl.distributedStore.Close()

	}

	return nil

}

// Helper implementations.

// NewAdaptiveManager performs newadaptivemanager operation.

func NewAdaptiveManager(config *AdaptiveConfig, logger *logging.StructuredLogger) *AdaptiveManager {

	return &AdaptiveManager{

		config: config,

		logger: logger,

		metrics: &AdaptiveMetrics{

			CurrentLimit: config.MaxLimit,
		},
	}

}

// Start performs start operation.

func (am *AdaptiveManager) Start() {

	am.adjustTicker = time.NewTicker(am.config.AdjustInterval)

	go am.adjustLimits()

}

func (am *AdaptiveManager) adjustLimits() {

	for range am.adjustTicker.C {

		// Adjust limits based on metrics.

		am.mu.Lock()

		// Implementation would adjust limits based on latency and error rate.

		am.mu.Unlock()

	}

}

// NewDDoSProtector performs newddosprotector operation.

func NewDDoSProtector(config *DDoSProtectionConfig, logger *logging.StructuredLogger) *DDoSProtector {

	return &DDoSProtector{

		config: config,

		logger: logger,

		blockedIPs: make(map[string]time.Time),

		requestCounts: make(map[string]*RequestCounter),
	}

}

// Start performs start operation.

func (dp *DDoSProtector) Start() {

	ticker := time.NewTicker(dp.config.DetectionWindow)

	defer ticker.Stop()

	for range ticker.C {

		dp.detectAndBlock()

	}

}

// IsBlocked performs isblocked operation.

func (dp *DDoSProtector) IsBlocked(clientID string) bool {

	dp.mu.RLock()

	defer dp.mu.RUnlock()

	if expiry, ok := dp.blockedIPs[clientID]; ok {

		if time.Now().Before(expiry) {

			return true

		}

		// Remove expired block.

		delete(dp.blockedIPs, clientID)

	}

	return false

}

func (dp *DDoSProtector) detectAndBlock() {

	dp.mu.Lock()

	defer dp.mu.Unlock()

	// Implementation would detect DDoS patterns and block IPs.

}

// Redis store implementation.

// RedisStore represents a redisstore.

type RedisStore struct {
	client *redis.Client
}

// NewRedisStore performs newredisstore operation.

func NewRedisStore(config *RedisConfig) (*RedisStore, error) {

	client := redis.NewClient(&redis.Options{

		Addr: config.Addresses[0],

		Password: config.Password,

		DB: config.DB,

		PoolSize: config.PoolSize,

		MaxRetries: config.MaxRetries,

		ReadTimeout: config.ReadTimeout,

		WriteTimeout: config.WriteTimeout,
	})

	// Test connection.

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {

		return nil, fmt.Errorf("failed to connect to Redis: %w", err)

	}

	return &RedisStore{client: client}, nil

}

// Increment performs increment operation.

func (rs *RedisStore) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {

	pipe := rs.client.Pipeline()

	incr := pipe.Incr(ctx, key)

	pipe.Expire(ctx, key, window)

	if _, err := pipe.Exec(ctx); err != nil {

		return 0, err

	}

	return incr.Val(), nil

}

// Get performs get operation.

func (rs *RedisStore) Get(ctx context.Context, key string) (int64, error) {

	val, err := rs.client.Get(ctx, key).Int64()

	if errors.Is(err, redis.Nil) {

		return 0, nil

	}

	return val, err

}

// Reset performs reset operation.

func (rs *RedisStore) Reset(ctx context.Context, key string) error {

	return rs.client.Del(ctx, key).Err()

}

// SetWithExpiry performs setwithexpiry operation.

func (rs *RedisStore) SetWithExpiry(ctx context.Context, key string, value int64, expiry time.Duration) error {

	return rs.client.Set(ctx, key, value, expiry).Err()

}

// Close performs close operation.

func (rs *RedisStore) Close() error {

	return rs.client.Close()

}

// Helper function.

func maxInt(a, b int) int {

	if a > b {

		return a

	}

	return b

}
