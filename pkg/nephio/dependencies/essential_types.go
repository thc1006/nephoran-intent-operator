/*

Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

*/

package dependencies

import (
	"context"
	"fmt"
	"time"

	"github.com/go-logr/logr"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// ValidationWorkerPool manages worker pool for validation operations.
// This is needed by validator.go which is always compiled.
type ValidationWorkerPool struct {
	workerCount int
	queueSize   int
	logger      logr.Logger
	workers     chan struct{}
	workQueue   chan func()
	closed      bool
}

// NewValidationWorkerPool creates a new validation worker pool.
func NewValidationWorkerPool(workerCount, queueSize int) *ValidationWorkerPool {
	pool := &ValidationWorkerPool{
		workerCount: workerCount,
		queueSize:   queueSize,
		logger:      log.Log.WithName("validation-worker-pool"),
		workers:     make(chan struct{}, workerCount),
		workQueue:   make(chan func(), queueSize),
	}

	// Initialize worker pool.
	for range workerCount {
		go pool.worker()
	}

	return pool
}

func (p *ValidationWorkerPool) worker() {
	for {
		select {
		case work := <-p.workQueue:
			if work == nil {
				return
			}
			work()
		}
	}
}

// Submit performs submit operation.
func (p *ValidationWorkerPool) Submit(work func()) {
	if !p.closed {
		select {
		case p.workQueue <- work:
		default:
			// Queue is full, execute synchronously.
			work()
		}
	}
}

// Close performs close operation.
func (p *ValidationWorkerPool) Close() error {
	p.closed = true
	close(p.workQueue)
	return nil
}

// AnalysisWorkerPool manages worker pool for analysis operations.
// This is needed by analyzer.go which is always compiled.
type AnalysisWorkerPool struct {
	workerCount int
	queueSize   int
	logger      logr.Logger
	workers     chan struct{}
	workQueue   chan func()
	closed      bool
}

// NewAnalysisWorkerPool creates a new analysis worker pool.
func NewAnalysisWorkerPool(workerCount, queueSize int) *AnalysisWorkerPool {
	pool := &AnalysisWorkerPool{
		workerCount: workerCount,
		queueSize:   queueSize,
		logger:      log.Log.WithName("analysis-worker-pool"),
		workers:     make(chan struct{}, workerCount),
		workQueue:   make(chan func(), queueSize),
	}

	// Initialize worker pool.
	for range workerCount {
		go pool.worker()
	}

	return pool
}

func (p *AnalysisWorkerPool) worker() {
	for {
		select {
		case work := <-p.workQueue:
			if work == nil {
				return
			}
			work()
		}
	}
}

// Submit performs submit operation.
func (p *AnalysisWorkerPool) Submit(work func()) {
	if !p.closed {
		select {
		case p.workQueue <- work:
		default:
			// Queue is full, execute synchronously.
			work()
		}
	}
}

// Close performs close operation.
func (p *AnalysisWorkerPool) Close() error {
	p.closed = true
	close(p.workQueue)
	return nil
}

// AnalysisCache provides caching for analysis results.
// This is needed by analyzer.go which is always compiled.
type AnalysisCache struct {
	cache  map[string]interface{}
	ttl    time.Duration
	logger logr.Logger
}

// NewAnalysisCache creates a new analysis cache.
func NewAnalysisCache(config *AnalysisCacheConfig) *AnalysisCache {
	return &AnalysisCache{
		cache:  make(map[string]interface{}),
		ttl:    config.TTL,
		logger: log.Log.WithName("analysis-cache"),
	}
}

// Get retrieves an analysis result from cache.
func (c *AnalysisCache) Get(ctx context.Context, key string) (interface{}, error) {
	if result, exists := c.cache[key]; exists {
		return result, nil
	}
	return nil, fmt.Errorf("cache miss for key: %s", key)
}

// Set stores an analysis result in cache.
func (c *AnalysisCache) Set(ctx context.Context, key string, result interface{}) error {
	c.cache[key] = result
	return nil
}

// Close closes the cache.
func (c *AnalysisCache) Close() error {
	c.cache = nil
	return nil
}

// AnalysisDataStore provides persistent storage for analysis data.
// This is needed by analyzer.go which is always compiled.
type AnalysisDataStore struct {
	logger logr.Logger
}

// NewAnalysisDataStore creates a new analysis data store.
func NewAnalysisDataStore(config *DataStoreConfig) (*AnalysisDataStore, error) {
	return &AnalysisDataStore{
		logger: log.Log.WithName("analysis-data-store"),
	}, nil
}

// Close performs close operation.
func (s *AnalysisDataStore) Close() error {
	return nil
}

// Basic types needed by non-stub files
type PackageReference struct {
	Repository        string                 `json:"repository"`
	Name              string                 `json:"name"`
	Version           string                 `json:"version,omitempty"`
	VersionConstraint *VersionConstraint     `json:"versionConstraint,omitempty"`
	Source            PackageSource          `json:"source,omitempty"`
	Metadata          map[string]interface{} `json:"metadata,omitempty"`
}

type VersionConstraint struct {
	Constraint string `json:"constraint"`
	Type       string `json:"type,omitempty"`
}

type DependencyScope string

type PackageSource string

type ResolutionStrategy string
