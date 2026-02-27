package a1

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TDD Red Phase: All tests should FAIL initially

func TestNewRedisStatusStore(t *testing.T) {
	tests := []struct {
		name      string
		redisAddr string
		wantErr   bool
	}{
		{
			name:      "valid connection",
			redisAddr: "localhost:6379",
			wantErr:   false,
		},
		{
			name:      "invalid address",
			redisAddr: "invalid:9999",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store, err := NewRedisStatusStore(tt.redisAddr)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, store)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, store)
				defer store.Close()
			}
		})
	}
}

func TestRedisStatusStore_GetPolicyStatus(t *testing.T) {
	// Setup
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1, // Use DB 1 for testing
	})
	defer client.Close()

	ctx := context.Background()
	client.FlushDB(ctx)

	store := &RedisStatusStore{client: client}

	tests := []struct {
		name         string
		setupKey     string
		setupValue   string
		policyTypeID int
		policyID     string
		wantStatus   *A1PolicyStatus
		wantErr      bool
	}{
		{
			name:         "existing enforced status",
			setupKey:     "policy:status:100:test-001",
			setupValue:   `{"enforcement_status":"ENFORCED","enforcement_reason":"Scaled to 5"}`,
			policyTypeID: 100,
			policyID:     "test-001",
			wantStatus: &A1PolicyStatus{
				EnforcementStatus:  "ENFORCED",
				EnforcementReason:  "Scaled to 5",
				LastModified: time.Time{}, // Skip time comparison
			},
			wantErr: false,
		},
		{
			name:         "existing not enforced status",
			setupKey:     "policy:status:100:test-002",
			setupValue:   `{"enforcement_status":"NOT_ENFORCED","enforcement_reason":"Target not found"}`,
			policyTypeID: 100,
			policyID:     "test-002",
			wantStatus: &A1PolicyStatus{
				EnforcementStatus:  "NOT_ENFORCED",
				EnforcementReason:  "Target not found",
				LastModified: time.Time{}, // Skip time comparison
			},
			wantErr: false,
		},
		{
			name:         "missing key returns default",
			policyTypeID: 100,
			policyID:     "nonexistent",
			wantStatus: &A1PolicyStatus{
				EnforcementStatus:  "NOT_ENFORCED",
				EnforcementReason:  "Policy status not yet reported by xApp",
				LastModified: time.Time{}, // Will be set by implementation
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test data
			if tt.setupKey != "" {
				err := client.Set(ctx, tt.setupKey, tt.setupValue, 0).Err()
				require.NoError(t, err)
			}

			// Call method
			status, err := store.GetPolicyStatus(ctx, tt.policyTypeID, tt.policyID)

			// Verify
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantStatus.EnforcementStatus, status.EnforcementStatus)
				assert.Equal(t, tt.wantStatus.EnforcementReason, status.EnforcementReason)
				// Only check LastModified is not zero when we expect data
				if tt.setupKey != "" {
					assert.False(t, status.LastModified.IsZero(), "LastModified should be set")
				}
			}
		})
	}
}

func TestRedisStatusStore_GetPolicyStatus_InvalidJSON(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer client.Close()

	ctx := context.Background()
	client.FlushDB(ctx)

	store := &RedisStatusStore{client: client}

	// Write invalid JSON
	key := "policy:status:100:invalid-json"
	client.Set(ctx, key, "not-a-json", 0)

	// Should return error
	_, err := store.GetPolicyStatus(ctx, 100, "invalid-json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unmarshal")
}

func TestRedisStatusStore_ListPolicyStatuses(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer client.Close()

	ctx := context.Background()
	client.FlushDB(ctx)

	store := &RedisStatusStore{client: client}

	// Setup multiple policies
	testData := map[string]string{
		"policy:status:100:policy-001": `{"enforce_status":"ENFORCED","enforce_reason":"OK"}`,
		"policy:status:100:policy-002": `{"enforce_status":"NOT_ENFORCED","enforce_reason":"Failed"}`,
		"policy:status:100:policy-003": `{"enforce_status":"ENFORCED","enforce_reason":"Success"}`,
		"policy:status:200:other-001":  `{"enforce_status":"ENFORCED","enforce_reason":"Different type"}`,
	}

	for key, value := range testData {
		err := client.Set(ctx, key, value, 0).Err()
		require.NoError(t, err)
	}

	// List policies for type 100
	statuses, err := store.ListPolicyStatuses(ctx, 100)
	require.NoError(t, err)

	// Should only return 3 policies of type 100
	assert.Len(t, statuses, 3)
	assert.Contains(t, statuses, "policy-001")
	assert.Contains(t, statuses, "policy-002")
	assert.Contains(t, statuses, "policy-003")
	assert.NotContains(t, statuses, "other-001")
}

func TestRedisStatusStore_DeletePolicyStatus(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer client.Close()

	ctx := context.Background()
	client.FlushDB(ctx)

	store := &RedisStatusStore{client: client}

	// Setup
	key := "policy:status:100:delete-test"
	client.Set(ctx, key, `{"enforce_status":"ENFORCED"}`, 0)

	// Verify exists
	val, err := client.Get(ctx, key).Result()
	require.NoError(t, err)
	assert.NotEmpty(t, val)

	// Delete
	err = store.DeletePolicyStatus(ctx, 100, "delete-test")
	assert.NoError(t, err)

	// Verify deleted
	_, err = client.Get(ctx, key).Result()
	assert.Equal(t, redis.Nil, err)
}

func TestRedisStatusStore_Health(t *testing.T) {
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

			store := &RedisStatusStore{client: client}

			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			err := store.Health(ctx)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestRedisStatusStore_Close(t *testing.T) {
	store, err := NewRedisStatusStore("localhost:6379")
	require.NoError(t, err)

	err = store.Close()
	assert.NoError(t, err)

	// After close, health should fail
	ctx := context.Background()
	err = store.Health(ctx)
	assert.Error(t, err)
}

func TestRedisKeyPatternMatching(t *testing.T) {
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})
	defer client.Close()

	ctx := context.Background()
	client.FlushDB(ctx)

	// Setup keys with different patterns
	keys := []string{
		"policy:status:100:test-001",
		"policy:status:100:test-002",
		"policy:status:200:test-003",
		"other:key:format",
	}

	for _, key := range keys {
		client.Set(ctx, key, "value", 0)
	}

	// Scan for pattern
	pattern := "policy:status:100:*"
	var matchedKeys []string
	var cursor uint64

	for {
		var batch []string
		var err error
		batch, cursor, err = client.Scan(ctx, cursor, pattern, 100).Result()
		require.NoError(t, err)

		matchedKeys = append(matchedKeys, batch...)
		if cursor == 0 {
			break
		}
	}

	// Should match only type 100 policies
	assert.Len(t, matchedKeys, 2)
}
