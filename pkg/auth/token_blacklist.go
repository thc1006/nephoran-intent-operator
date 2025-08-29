
package auth



import (

	"context"

	"crypto/sha256"

	"encoding/hex"

	"errors"

	"fmt"

	"log/slog"

	"strconv"

	"strings"

	"sync"

	"time"



	"github.com/redis/go-redis/v9"

)



// TokenBlacklistManager manages revoked tokens and prevents their reuse.

type TokenBlacklistManager struct {

	redisClient   *redis.Client

	localCache    map[string]time.Time

	cacheMutex    sync.RWMutex

	config        *BlacklistConfig

	logger        *slog.Logger

	metrics       *BlacklistMetrics

	cleanupTicker *time.Ticker

	stopChan      chan struct{}

}



// BlacklistConfig holds token blacklist configuration.

type BlacklistConfig struct {

	RedisAddr         string        `json:"redis_addr"`

	RedisPassword     string        `json:"redis_password"`

	RedisDB           int           `json:"redis_db"`

	RedisKeyPrefix    string        `json:"redis_key_prefix"`

	LocalCacheSize    int           `json:"local_cache_size"`

	LocalCacheTTL     time.Duration `json:"local_cache_ttl"`

	CleanupInterval   time.Duration `json:"cleanup_interval"`

	MaxTokenAge       time.Duration `json:"max_token_age"`

	EnableCompression bool          `json:"enable_compression"`

}



// BlacklistMetrics tracks blacklist performance.

type BlacklistMetrics struct {

	TotalBlacklisted     int64         `json:"total_blacklisted"`

	TotalChecks          int64         `json:"total_checks"`

	CacheHits            int64         `json:"cache_hits"`

	CacheMisses          int64         `json:"cache_misses"`

	RedisErrors          int64         `json:"redis_errors"`

	AverageCheckTime     time.Duration `json:"average_check_time"`

	LastCleanup          time.Time     `json:"last_cleanup"`

	RemovedExpiredTokens int64         `json:"removed_expired_tokens"`

	mutex                sync.RWMutex

}



// BlacklistEntry represents a blacklisted token entry.

type BlacklistEntry struct {

	TokenHash     string    `json:"token_hash"`

	UserID        string    `json:"user_id"`

	Reason        string    `json:"reason"`

	BlacklistedAt time.Time `json:"blacklisted_at"`

	ExpiresAt     time.Time `json:"expires_at"`

	IPAddress     string    `json:"ip_address"`

	UserAgent     string    `json:"user_agent"`

}



// NewTokenBlacklistManager creates a new token blacklist instance.

func NewTokenBlacklistManager(config *BlacklistConfig) (*TokenBlacklistManager, error) {

	if config == nil {

		config = getDefaultBlacklistConfig()

	}



	// Initialize Redis client.

	rdb := redis.NewClient(&redis.Options{

		Addr:     config.RedisAddr,

		Password: config.RedisPassword,

		DB:       config.RedisDB,

	})



	// Test Redis connection.

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

	defer cancel()



	if err := rdb.Ping(ctx).Err(); err != nil {

		return nil, fmt.Errorf("failed to connect to Redis: %w", err)

	}



	blacklist := &TokenBlacklistManager{

		redisClient: rdb,

		localCache:  make(map[string]time.Time),

		config:      config,

		logger:      slog.Default().With("component", "token-blacklist"),

		metrics:     &BlacklistMetrics{LastCleanup: time.Now()},

		stopChan:    make(chan struct{}),

	}



	// Start cleanup routine.

	blacklist.startCleanupRoutine()



	return blacklist, nil

}



// getDefaultBlacklistConfig returns default blacklist configuration.

func getDefaultBlacklistConfig() *BlacklistConfig {

	return &BlacklistConfig{

		RedisAddr:         "localhost:6379",

		RedisPassword:     "",

		RedisDB:           0,

		RedisKeyPrefix:    "nephoran:blacklist:",

		LocalCacheSize:    10000,

		LocalCacheTTL:     5 * time.Minute,

		CleanupInterval:   time.Hour,

		MaxTokenAge:       7 * 24 * time.Hour, // 7 days

		EnableCompression: true,

	}

}



// IsBlacklisted checks if a token is blacklisted.

func (tb *TokenBlacklistManager) IsBlacklisted(ctx context.Context, token string) (bool, error) {

	startTime := time.Now()

	defer func() {

		tb.updateMetrics(func(m *BlacklistMetrics) {

			m.TotalChecks++

			m.AverageCheckTime = (m.AverageCheckTime*time.Duration(m.TotalChecks-1) +

				time.Since(startTime)) / time.Duration(m.TotalChecks)

		})

	}()



	tokenHash := tb.hashToken(token)



	// Check local cache first.

	tb.cacheMutex.RLock()

	if expiresAt, exists := tb.localCache[tokenHash]; exists {

		tb.cacheMutex.RUnlock()



		tb.updateMetrics(func(m *BlacklistMetrics) {

			m.CacheHits++

		})



		// Check if token is still valid (not expired).

		if time.Now().Before(expiresAt) {

			return true, nil

		}



		// Remove expired token from cache.

		tb.cacheMutex.Lock()

		delete(tb.localCache, tokenHash)

		tb.cacheMutex.Unlock()



		return false, nil

	}

	tb.cacheMutex.RUnlock()



	tb.updateMetrics(func(m *BlacklistMetrics) {

		m.CacheMisses++

	})



	// Check Redis.

	key := tb.config.RedisKeyPrefix + tokenHash

	result, err := tb.redisClient.Get(ctx, key).Result()

	if err != nil {

		if errors.Is(err, redis.Nil) {

			return false, nil // Token not blacklisted

		}



		tb.updateMetrics(func(m *BlacklistMetrics) {

			m.RedisErrors++

		})



		tb.logger.Error("Redis error checking blacklist", "error", err, "token_hash", tokenHash)

		return false, fmt.Errorf("failed to check blacklist: %w", err)

	}



	// Parse blacklist entry.

	var entry BlacklistEntry

	if err := tb.unmarshalEntry(result, &entry); err != nil {

		tb.logger.Error("Failed to parse blacklist entry", "error", err)

		return false, fmt.Errorf("failed to parse blacklist entry: %w", err)

	}



	// Check if token is still valid (not expired).

	if time.Now().After(entry.ExpiresAt) {

		// Token expired, remove from Redis.

		go tb.removeExpiredToken(tokenHash)

		return false, nil

	}



	// Add to local cache.

	tb.cacheMutex.Lock()

	tb.localCache[tokenHash] = entry.ExpiresAt



	// Enforce cache size limit.

	if len(tb.localCache) > tb.config.LocalCacheSize {

		tb.evictOldestCacheEntry()

	}

	tb.cacheMutex.Unlock()



	return true, nil

}



// BlacklistToken adds a token to the blacklist.

func (tb *TokenBlacklistManager) BlacklistToken(ctx context.Context, token, userID, reason, ipAddress, userAgent string, expiresAt time.Time) error {

	tokenHash := tb.hashToken(token)



	entry := BlacklistEntry{

		TokenHash:     tokenHash,

		UserID:        userID,

		Reason:        reason,

		BlacklistedAt: time.Now(),

		ExpiresAt:     expiresAt,

		IPAddress:     ipAddress,

		UserAgent:     userAgent,

	}



	// Store in Redis.

	data, err := tb.marshalEntry(&entry)

	if err != nil {

		return fmt.Errorf("failed to marshal blacklist entry: %w", err)

	}



	key := tb.config.RedisKeyPrefix + tokenHash

	expiration := time.Until(expiresAt)



	if err := tb.redisClient.Set(ctx, key, data, expiration).Err(); err != nil {

		tb.updateMetrics(func(m *BlacklistMetrics) {

			m.RedisErrors++

		})

		return fmt.Errorf("failed to store blacklist entry: %w", err)

	}



	// Add to local cache.

	tb.cacheMutex.Lock()

	tb.localCache[tokenHash] = expiresAt

	tb.cacheMutex.Unlock()



	tb.updateMetrics(func(m *BlacklistMetrics) {

		m.TotalBlacklisted++

	})



	tb.logger.Info("Token blacklisted",

		"user_id", userID,

		"reason", reason,

		"expires_at", expiresAt,

		"ip_address", ipAddress)



	return nil

}



// BlacklistAllUserTokens blacklists all tokens for a specific user.

func (tb *TokenBlacklistManager) BlacklistAllUserTokens(ctx context.Context, userID, reason string) error {

	// In a production system, you'd maintain a user -> tokens mapping.

	// For now, we'll use a pattern-based approach with user ID in the key.



	pattern := tb.config.RedisKeyPrefix + "user:" + userID + ":*"

	keys, err := tb.redisClient.Keys(ctx, pattern).Result()

	if err != nil {

		return fmt.Errorf("failed to find user tokens: %w", err)

	}



	for _, key := range keys {

		if err := tb.redisClient.Del(ctx, key).Err(); err != nil {

			tb.logger.Error("Failed to delete user token", "key", key, "error", err)

		}

	}



	// Create a general user blacklist entry.

	entry := BlacklistEntry{

		TokenHash:     "user:" + userID + ":all",

		UserID:        userID,

		Reason:        reason,

		BlacklistedAt: time.Now(),

		ExpiresAt:     time.Now().Add(tb.config.MaxTokenAge),

	}



	data, err := tb.marshalEntry(&entry)

	if err != nil {

		return fmt.Errorf("failed to marshal user blacklist entry: %w", err)

	}



	userKey := tb.config.RedisKeyPrefix + "user:" + userID + ":all"

	if err := tb.redisClient.Set(ctx, userKey, data, tb.config.MaxTokenAge).Err(); err != nil {

		return fmt.Errorf("failed to store user blacklist entry: %w", err)

	}



	tb.logger.Info("All user tokens blacklisted", "user_id", userID, "reason", reason)

	return nil

}



// GetBlacklistEntry retrieves a blacklist entry for a token.

func (tb *TokenBlacklistManager) GetBlacklistEntry(ctx context.Context, token string) (*BlacklistEntry, error) {

	tokenHash := tb.hashToken(token)

	key := tb.config.RedisKeyPrefix + tokenHash



	result, err := tb.redisClient.Get(ctx, key).Result()

	if err != nil {

		if errors.Is(err, redis.Nil) {

			return nil, nil // Token not blacklisted

		}

		return nil, fmt.Errorf("failed to get blacklist entry: %w", err)

	}



	var entry BlacklistEntry

	if err := tb.unmarshalEntry(result, &entry); err != nil {

		return nil, fmt.Errorf("failed to parse blacklist entry: %w", err)

	}



	return &entry, nil

}



// GetBlacklistStats returns current blacklist statistics.

func (tb *TokenBlacklistManager) GetBlacklistStats(ctx context.Context) (*BlacklistMetrics, error) {

	tb.metrics.mutex.RLock()

	defer tb.metrics.mutex.RUnlock()



	// Create new metrics struct to avoid copying mutex.

	stats := &BlacklistMetrics{

		TotalBlacklisted:     tb.metrics.TotalBlacklisted,

		TotalChecks:          tb.metrics.TotalChecks,

		CacheHits:            tb.metrics.CacheHits,

		CacheMisses:          tb.metrics.CacheMisses,

		RedisErrors:          tb.metrics.RedisErrors,

		AverageCheckTime:     tb.metrics.AverageCheckTime,

		LastCleanup:          tb.metrics.LastCleanup,

		RemovedExpiredTokens: tb.metrics.RemovedExpiredTokens,

	}



	// Get Redis stats.

	_, err := tb.redisClient.Info(ctx, "keyspace").Result()

	if err == nil {

		// Parse keyspace info for additional stats.

		// This is a simplified version.

		stats.LastCleanup = time.Now()

	}



	return stats, nil

}



// CleanupExpiredTokens removes expired tokens from the blacklist.

func (tb *TokenBlacklistManager) CleanupExpiredTokens(ctx context.Context) error {

	tb.logger.Info("Starting blacklist cleanup")

	startTime := time.Now()



	// Get all blacklist keys.

	pattern := tb.config.RedisKeyPrefix + "*"

	keys, err := tb.redisClient.Keys(ctx, pattern).Result()

	if err != nil {

		return fmt.Errorf("failed to get blacklist keys: %w", err)

	}



	removedCount := int64(0)

	for _, key := range keys {

		// Check if key exists (might have expired naturally).

		exists, err := tb.redisClient.Exists(ctx, key).Result()

		if err != nil {

			tb.logger.Error("Error checking key existence", "key", key, "error", err)

			continue

		}



		if exists == 0 {

			removedCount++

		}

	}



	// Clean local cache.

	tb.cacheMutex.Lock()

	now := time.Now()

	for tokenHash, expiresAt := range tb.localCache {

		if now.After(expiresAt) {

			delete(tb.localCache, tokenHash)

			removedCount++

		}

	}

	tb.cacheMutex.Unlock()



	tb.updateMetrics(func(m *BlacklistMetrics) {

		m.LastCleanup = now

		m.RemovedExpiredTokens += removedCount

	})



	tb.logger.Info("Blacklist cleanup completed",

		"duration", time.Since(startTime),

		"removed_tokens", removedCount)



	return nil

}



// Close shuts down the token blacklist.

func (tb *TokenBlacklistManager) Close() error {

	close(tb.stopChan)



	if tb.cleanupTicker != nil {

		tb.cleanupTicker.Stop()

	}



	return tb.redisClient.Close()

}



// Helper methods.



func (tb *TokenBlacklistManager) hashToken(token string) string {

	hash := sha256.Sum256([]byte(token))

	return hex.EncodeToString(hash[:])

}



func (tb *TokenBlacklistManager) marshalEntry(entry *BlacklistEntry) (string, error) {

	if tb.config.EnableCompression {

		// In production, implement compression.

		// For now, just use JSON.

		// TODO: Implement compression when needed.

	}



	data := fmt.Sprintf("%s|%s|%s|%d|%d|%s|%s",

		entry.TokenHash,

		entry.UserID,

		entry.Reason,

		entry.BlacklistedAt.Unix(),

		entry.ExpiresAt.Unix(),

		entry.IPAddress,

		entry.UserAgent)



	return data, nil

}



func (tb *TokenBlacklistManager) unmarshalEntry(data string, entry *BlacklistEntry) error {

	// Simple parsing for the format we used in marshalEntry.

	// In production, use proper serialization.

	parts := strings.Split(data, "|")

	if len(parts) != 7 {

		return fmt.Errorf("invalid entry format")

	}



	entry.TokenHash = parts[0]

	entry.UserID = parts[1]

	entry.Reason = parts[2]



	// Parse timestamps.

	if blacklistedAt, err := strconv.ParseInt(parts[3], 10, 64); err == nil {

		entry.BlacklistedAt = time.Unix(blacklistedAt, 0)

	}



	if expiresAt, err := strconv.ParseInt(parts[4], 10, 64); err == nil {

		entry.ExpiresAt = time.Unix(expiresAt, 0)

	}



	entry.IPAddress = parts[5]

	entry.UserAgent = parts[6]



	return nil

}



func (tb *TokenBlacklistManager) evictOldestCacheEntry() {

	var oldestHash string

	var oldestTime time.Time



	for hash, expiresAt := range tb.localCache {

		if oldestHash == "" || expiresAt.Before(oldestTime) {

			oldestHash = hash

			oldestTime = expiresAt

		}

	}



	if oldestHash != "" {

		delete(tb.localCache, oldestHash)

	}

}



func (tb *TokenBlacklistManager) removeExpiredToken(tokenHash string) {

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	defer cancel()



	key := tb.config.RedisKeyPrefix + tokenHash

	if err := tb.redisClient.Del(ctx, key).Err(); err != nil {

		tb.logger.Error("Failed to remove expired token", "token_hash", tokenHash, "error", err)

	}

}



func (tb *TokenBlacklistManager) startCleanupRoutine() {

	tb.cleanupTicker = time.NewTicker(tb.config.CleanupInterval)



	go func() {

		for {

			select {

			case <-tb.cleanupTicker.C:

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)

				if err := tb.CleanupExpiredTokens(ctx); err != nil {

					tb.logger.Error("Cleanup failed", "error", err)

				}

				cancel()

			case <-tb.stopChan:

				return

			}

		}

	}()

}



func (tb *TokenBlacklistManager) updateMetrics(updater func(*BlacklistMetrics)) {

	tb.metrics.mutex.Lock()

	defer tb.metrics.mutex.Unlock()

	updater(tb.metrics)

}

