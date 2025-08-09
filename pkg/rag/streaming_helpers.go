//go:build !disable_rag && !test

package rag

import (
	"runtime"
	"sync/atomic"
	"time"
)

// MemoryMonitor and ProcessingPool definitions moved to document_loader.go to avoid duplicates

// NoOpRedisCache definition moved to redis_cache.go to avoid duplicates

// getDefaultChunkingConfig definition moved to config_defaults.go to avoid duplicates

// getDefaultEmbeddingConfig definition moved to config_defaults.go to avoid duplicates
