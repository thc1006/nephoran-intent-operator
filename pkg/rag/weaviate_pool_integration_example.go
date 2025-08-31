//go:build !disable_rag && !test

package rag

import (
	"github.com/thc1006/nephoran-intent-operator/pkg/monitoring"
)

// IntegrateWithMetricsCollector demonstrates how to integrate the Weaviate connection pool.

// with the main monitoring metrics collector.

//

// Example usage:.

//

//	metricsCollector := monitoring.NewMetricsCollector()

//	poolConfig := DefaultPoolConfig()

//	pool := IntegrateWeaviatePoolWithMetrics(poolConfig, metricsCollector)

//

//	if err := pool.Start(); err != nil {

//	    log.Fatal("Failed to start pool:", err)

//	}

func IntegrateWeaviatePoolWithMetrics(config *PoolConfig, metricsCollector *monitoring.MetricsCollector) *WeaviateConnectionPool {

	return NewWeaviateConnectionPoolWithMetrics(config, metricsCollector)

}

// SetupWeaviatePoolWithMonitoring creates a complete setup with monitoring integration.

func SetupWeaviatePoolWithMonitoring(config *PoolConfig) (*WeaviateConnectionPool, *monitoring.MetricsCollector) {

	metricsCollector := monitoring.NewMetricsCollector()

	pool := NewWeaviateConnectionPoolWithMetrics(config, metricsCollector)

	return pool, metricsCollector

}
