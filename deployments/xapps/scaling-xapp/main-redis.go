package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig holds Redis connection configuration.
type RedisConfig struct {
	Address  string
	Password string
	DB       int
}

// InitRedisClient initializes Redis client for policy status storage.
func InitRedisClient(cfg RedisConfig) *redis.Client {
	client := redis.NewClient(&redis.Options{
		Addr:         cfg.Address,
		Password:     cfg.Password,
		DB:           cfg.DB,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
		PoolSize:     10,
		MinIdleConns: 5,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		log.Printf("Redis connection failed: %v (will retry)", err)
	} else {
		log.Printf("Redis connected: %s", cfg.Address)
	}

	return client
}

func (x *ScalingXApp) checkRedisHealth() error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	return x.redisClient.Ping(ctx).Err()
}

func formatRedisKey(policyTypeID int, policyID string) string {
	return fmt.Sprintf("policy:status:%d:%s", policyTypeID, policyID)
}
