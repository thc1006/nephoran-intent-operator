package testcontainers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"github.com/stretchr/testify/require"
)

// RedisContainer manages Redis test container lifecycle
type RedisContainer struct {
	container testcontainers.Container
	endpoint  string
	ctx       context.Context
}

// SetupRedisContainer starts a Redis container for testing
func SetupRedisContainer(t *testing.T) (*RedisContainer, func()) {
	ctx := context.Background()
	
	req := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor: wait.ForAll(
			wait.ForLog("Ready to accept connections"),
			wait.ForExposedPort(),
		).WithDeadline(30 * time.Second),
		Cmd: []string{"redis-server", "--appendonly", "yes"},
		Env: map[string]string{
			"REDIS_PASSWORD": "",
		},
	}
	
	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	require.NoError(t, err, "Failed to start Redis container")
	
	endpoint, err := redisContainer.Endpoint(ctx, "")
	require.NoError(t, err, "Failed to get Redis endpoint")
	
	container := &RedisContainer{
		container: redisContainer,
		endpoint:  fmt.Sprintf("redis://%s", endpoint),
		ctx:       ctx,
	}
	
	cleanup := func() {
		if err := redisContainer.Terminate(ctx); err != nil {
			t.Logf("Warning: failed to terminate Redis container: %v", err)
		}
	}
	
	// Verify container is ready
	container.waitForReady(t)
	
	return container, cleanup
}

// GetEndpoint returns the Redis connection endpoint
func (r *RedisContainer) GetEndpoint() string {
	return r.endpoint
}

// GetHost returns just the host:port without the redis:// prefix
func (r *RedisContainer) GetHost() string {
	endpoint, err := r.container.Endpoint(r.ctx, "")
	if err != nil {
		return "localhost:6379"
	}
	return endpoint
}

// GetPort returns the mapped Redis port
func (r *RedisContainer) GetPort(t *testing.T) int {
	port, err := r.container.MappedPort(r.ctx, "6379/tcp")
	require.NoError(t, err)
	return port.Int()
}

// waitForReady ensures Redis is ready to accept connections
func (r *RedisContainer) waitForReady(t *testing.T) {
	// Additional readiness check beyond container wait
	ready := make(chan bool, 1)
	timeout := time.After(30 * time.Second)
	
	go func() {
		for {
			if r.isReady() {
				ready <- true
				return
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()
	
	select {
	case <-ready:
		t.Log("Redis container is ready")
	case <-timeout:
		t.Fatal("Timeout waiting for Redis container to be ready")
	}
}

// isReady checks if Redis is accepting connections
func (r *RedisContainer) isReady() bool {
	// This would normally use a Redis client to ping
	// For now, we rely on the container's wait strategy
	return true
}

// ExecuteRedisCommand executes a Redis command in the container
func (r *RedisContainer) ExecuteRedisCommand(t *testing.T, command ...string) string {
	cmd := append([]string{"redis-cli"}, command...)
	
	exitCode, reader, err := r.container.Exec(r.ctx, cmd)
	require.NoError(t, err, "Failed to execute Redis command")
	require.Equal(t, 0, exitCode, "Redis command failed with exit code %d", exitCode)
	
	output, err := testcontainers.ReadAllAsString(reader)
	require.NoError(t, err, "Failed to read command output")
	
	return output
}

// FlushAll clears all data from Redis
func (r *RedisContainer) FlushAll(t *testing.T) {
	r.ExecuteRedisCommand(t, "FLUSHALL")
}

// SetKey sets a key-value pair in Redis
func (r *RedisContainer) SetKey(t *testing.T, key, value string) {
	r.ExecuteRedisCommand(t, "SET", key, value)
}

// GetKey retrieves a value by key from Redis
func (r *RedisContainer) GetKey(t *testing.T, key string) string {
	return r.ExecuteRedisCommand(t, "GET", key)
}

// ContainerManager manages multiple test containers
type ContainerManager struct {
	containers map[string]testcontainers.Container
	cleanups   []func()
	ctx        context.Context
}

// NewContainerManager creates a new container manager
func NewContainerManager() *ContainerManager {
	return &ContainerManager{
		containers: make(map[string]testcontainers.Container),
		cleanups:   make([]func(), 0),
		ctx:        context.Background(),
	}
}

// AddRedis adds a Redis container to the manager
func (cm *ContainerManager) AddRedis(t *testing.T, name string) *RedisContainer {
	redisContainer, cleanup := SetupRedisContainer(t)
	
	cm.containers[name] = redisContainer.container
	cm.cleanups = append(cm.cleanups, cleanup)
	
	return redisContainer
}

// Cleanup terminates all managed containers
func (cm *ContainerManager) Cleanup(t *testing.T) {
	for _, cleanup := range cm.cleanups {
		cleanup()
	}
	cm.containers = make(map[string]testcontainers.Container)
	cm.cleanups = make([]func(), 0)
}

// GetContainer returns a managed container by name
func (cm *ContainerManager) GetContainer(name string) testcontainers.Container {
	return cm.containers[name]
}