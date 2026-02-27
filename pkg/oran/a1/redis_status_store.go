package a1

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisStatusStore provides policy status storage using Redis
// This replaces the HTTP GET requests to A1 Mediator for status retrieval
type RedisStatusStore struct {
	client *redis.Client
}

// NewRedisStatusStore creates a new Redis-backed status store
func NewRedisStatusStore(redisAddr string) (*RedisStatusStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:         redisAddr,
		Password:     "", // No password for local Redis
		DB:           0,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     20,
		MinIdleConns: 5,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("redis connection failed: %w", err)
	}

	return &RedisStatusStore{
		client: client,
	}, nil
}

// GetPolicyStatus retrieves policy status from Redis
// Key format: policy:status:{policyTypeID}:{policyID}
func (r *RedisStatusStore) GetPolicyStatus(ctx context.Context, policyTypeID int, policyID string) (*A1PolicyStatus, error) {
	redisKey := fmt.Sprintf("policy:status:%d:%s", policyTypeID, policyID)

	// Get status from Redis
	statusJSON, err := r.client.Get(ctx, redisKey).Result()
	if err == redis.Nil {
		// Key doesn't exist - policy status not yet reported
		return &A1PolicyStatus{
			EnforcementStatus: "NOT_ENFORCED",
			EnforcementReason: "Policy status not yet reported by xApp",
			LastModified:      time.Now(),
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("redis get failed: %w", err)
	}

	// Parse JSON - use existing A1PolicyStatus type from a1_adaptor.go
	var status A1PolicyStatus

	if err := json.Unmarshal([]byte(statusJSON), &status); err != nil {
		return nil, fmt.Errorf("failed to unmarshal policy status: %w", err)
	}

	// Set LastModified to current time if not present in JSON
	if status.LastModified.IsZero() {
		status.LastModified = time.Now()
	}

	return &status, nil
}

// ListPolicyStatuses lists all policy statuses for a policy type
func (r *RedisStatusStore) ListPolicyStatuses(ctx context.Context, policyTypeID int) (map[string]*A1PolicyStatus, error) {
	pattern := fmt.Sprintf("policy:status:%d:*", policyTypeID)

	// Scan for matching keys
	var cursor uint64
	var keys []string

	for {
		var batch []string
		var err error
		batch, cursor, err = r.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan failed: %w", err)
		}

		keys = append(keys, batch...)

		if cursor == 0 {
			break
		}
	}

	// Get status for each key
	statuses := make(map[string]*A1PolicyStatus)
	for _, key := range keys {
		// Extract policy ID from key (policy:status:{typeID}:{policyID})
		policyID := key[len(fmt.Sprintf("policy:status:%d:", policyTypeID)):]

		status, err := r.GetPolicyStatus(ctx, policyTypeID, policyID)
		if err != nil {
			// Log error but continue
			continue
		}

		statuses[policyID] = status
	}

	return statuses, nil
}

// DeletePolicyStatus removes policy status from Redis
func (r *RedisStatusStore) DeletePolicyStatus(ctx context.Context, policyTypeID int, policyID string) error {
	redisKey := fmt.Sprintf("policy:status:%d:%s", policyTypeID, policyID)

	err := r.client.Del(ctx, redisKey).Err()
	if err != nil {
		return fmt.Errorf("redis delete failed: %w", err)
	}

	return nil
}

// Close closes the Redis connection
func (r *RedisStatusStore) Close() error {
	return r.client.Close()
}

// Health checks Redis connection health
func (r *RedisStatusStore) Health(ctx context.Context) error {
	return r.client.Ping(ctx).Err()
}
