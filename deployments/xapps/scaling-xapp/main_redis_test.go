package main

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TDD Red Phase: All tests should FAIL initially

func TestInitRedisClient(t *testing.T) {
	tests := []struct {
		name    string
		config  RedisConfig
		wantErr bool
	}{
		{
			name: "valid connection",
			config: RedisConfig{
				Address:  "localhost:6379",
				Password: "",
				DB:       0,
			},
			wantErr: false,
		},
		{
			name: "invalid address",
			config: RedisConfig{
				Address:  "invalid:9999",
				Password: "",
				DB:       0,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := InitRedisClient(tt.config)
			require.NotNil(t, client)

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := client.Ping(ctx).Err()
			if tt.wantErr {
				assert.Error(t, err, "Expected connection to fail")
			} else {
				assert.NoError(t, err, "Expected connection to succeed")
			}
		})
	}
}

func TestScalingXApp_reportPolicyStatusRedis(t *testing.T) {
	// Setup Redis client (assumes Redis running on localhost:6379)
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use DB 1 for testing
	})
	defer client.Close()

	// Clear test data
	ctx := context.Background()
	client.FlushDB(ctx)

	xapp := &ScalingXApp{
		redisClient: client,
	}

	tests := []struct {
		name     string
		policyID string
		enforced bool
		reason   string
		wantKey  string
	}{
		{
			name:     "write enforced status",
			policyID: "test-policy-001",
			enforced: true,
			reason:   "Scaled to 5 replicas",
			wantKey:  "policy:status:100:test-policy-001",
		},
		{
			name:     "write not enforced status",
			policyID: "test-policy-002",
			enforced: false,
			reason:   "Target deployment not found",
			wantKey:  "policy:status:100:test-policy-002",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Call the method
			xapp.reportPolicyStatusRedis(tt.policyID, tt.enforced, tt.reason)

			// Verify Redis write
			statusJSON, err := client.Get(ctx, tt.wantKey).Result()
			require.NoError(t, err, "Expected status to be written to Redis")

			// Parse and verify
			var status PolicyStatus
			err = json.Unmarshal([]byte(statusJSON), &status)
			require.NoError(t, err, "Expected valid JSON")

			if tt.enforced {
				assert.Equal(t, "ENFORCED", status.EnforcementStatus)
			} else {
				assert.Equal(t, "NOT_ENFORCED", status.EnforcementStatus)
			}
			assert.Equal(t, tt.reason, status.EnforcementReason)
		})
	}
}

func TestScalingXApp_reportPolicyStatusRedis_ConnectionError(t *testing.T) {
	// Create client with invalid address
	client := redis.NewClient(&redis.Options{
		Addr: "invalid:9999",
		DB:   0,
	})
	defer client.Close()

	xapp := &ScalingXApp{
		redisClient: client,
	}

	// This should not panic, but log error
	xapp.reportPolicyStatusRedis("test-policy", true, "test reason")

	// Test passes if no panic occurred
}

func TestScalingXApp_checkRedisHealth(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{
			name:    "healthy connection",
			addr:    "localhost:6379",
			wantErr: false,
		},
		{
			name:    "unhealthy connection",
			addr:    "invalid:9999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := redis.NewClient(&redis.Options{
				Addr: tt.addr,
				DB:   0,
			})
			defer client.Close()

			xapp := &ScalingXApp{
				redisClient: client,
			}

			err := xapp.checkRedisHealth()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisKeyFormat(t *testing.T) {
	tests := []struct {
		name         string
		policyTypeID int
		policyID     string
		wantKey      string
	}{
		{
			name:         "standard policy",
			policyTypeID: 100,
			policyID:     "scaling-001",
			wantKey:      "policy:status:100:scaling-001",
		},
		{
			name:         "policy with hyphens",
			policyTypeID: 100,
			policyID:     "test-policy-with-many-hyphens",
			wantKey:      "policy:status:100:test-policy-with-many-hyphens",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			key := formatRedisKey(tt.policyTypeID, tt.policyID)
			assert.Equal(t, tt.wantKey, key)
		})
	}
}

// Helper function that should exist in main code
func formatRedisKey(policyTypeID int, policyID string) string {
	return fmt.Sprintf("policy:status:%d:%s", policyTypeID, policyID)
}

func TestRedisStatusExpiration(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer client.Close()

	ctx := context.Background()
	client.FlushDB(ctx)

	// Write status with expiration
	key := "policy:status:100:expiry-test"
	statusJSON := `{"enforcement_status":"ENFORCED","enforcement_reason":"test"}`

	err := client.Set(ctx, key, statusJSON, 1*time.Second).Err()
	require.NoError(t, err)

	// Verify exists
	val, err := client.Get(ctx, key).Result()
	require.NoError(t, err)
	assert.Equal(t, statusJSON, val)

	// Wait for expiration
	time.Sleep(2 * time.Second)

	// Verify expired
	_, err = client.Get(ctx, key).Result()
	assert.Equal(t, redis.Nil, err, "Expected key to be expired")
}
