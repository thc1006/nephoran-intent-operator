// Redis-integrated version of scaling xApp
// This file contains the modified reportPolicyStatus function
// that writes to Redis instead of POSTing to A1 Mediator

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Address  string // Redis server address (e.g., "a1-status-store.ricplt:6379")
	Password string // Redis password (empty for no auth)
	DB       int    // Redis database number (default 0)
}

// InitRedisClient initializes Redis client for policy status storage
func InitRedisClient(cfg RedisConfig) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Address,
		Password: cfg.Password,
		DB:       cfg.DB,
		// Connection settings
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("⚠️  Redis connection failed: %v (will retry)", err)
	} else {
		log.Printf("✅ Redis connected: %s", cfg.Address)
	}

	return client
}

// reportPolicyStatusRedis reports policy enforcement status to Redis
// MODIFIED: Writes to Redis instead of POSTing to A1 Mediator
func (x *ScalingXApp) reportPolicyStatusRedis(policyID string, enforced bool, reason string) {
	start := time.Now()
	defer func() {
		a1RequestDuration.WithLabelValues("REDIS_SET").Observe(time.Since(start).Seconds())
	}()

	// Prepare status payload using O-RAN spec field names
	status := PolicyStatus{
		EnforcementStatus: "NOT_ENFORCED",
		EnforcementReason: reason,
	}
	if enforced {
		status.EnforcementStatus = "ENFORCED"
	}

	statusJSON, err := json.Marshal(status)
	if err != nil {
		log.Printf("Failed to marshal policy status for %s: %v", policyID, err)
		policyStatusReports.WithLabelValues(status.EnforcementStatus, "marshal_error").Inc()
		return
	}

	// Write status to Redis with policy type ID prefix
	// Key format: policy:status:{policyTypeID}:{policyID}
	// Example: policy:status:100:scaling-policy-001
	redisKey := fmt.Sprintf("policy:status:100:%s", policyID)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = x.redisClient.Set(ctx, redisKey, string(statusJSON), 0).Err()
	if err != nil {
		log.Printf("❌ Failed to write policy status to Redis for %s: %v", policyID, err)
		policyStatusReports.WithLabelValues(status.EnforcementStatus, "redis_error").Inc()
		return
	}

	log.Printf("✅ Policy status written to Redis: %s → %s (key: %s)",
		policyID, status.EnforcementStatus, redisKey)
	policyStatusReports.WithLabelValues(status.EnforcementStatus, "success").Inc()

	// Optional: Set expiration for status (e.g., 24 hours)
	// This prevents indefinite accumulation of old policy statuses
	expireCtx, expireCancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer expireCancel()
	x.redisClient.Expire(expireCtx, redisKey, 24*time.Hour)
}

// Helper: Check Redis connection health
func (x *ScalingXApp) checkRedisHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return x.redisClient.Ping(ctx).Err()
}

// Helper: Format Redis key for policy status
func formatRedisKey(policyTypeID int, policyID string) string {
	return fmt.Sprintf("policy:status:%d:%s", policyTypeID, policyID)
}
