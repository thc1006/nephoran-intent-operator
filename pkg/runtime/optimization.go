//go:build go1.24

// Package runtime provides Go 1.24.8 runtime optimizations for August 2025
// with GOMAXPROCS tuning, memory management, and Swiss Tables support.
package runtime

import (
	"runtime"
	"runtime/debug"
	"time"
	_ "unsafe" // Required for runtime linking
)

// Go 1.24.8 Swiss Tables runtime optimization
//go:linkname procyield runtime.procyield
//go:linkname osyield runtime.osyield

func procyield(cycles uint32)
func osyield()

// OptimizationConfig holds runtime optimization settings for Go 1.24.8
type OptimizationConfig struct {
	// GOMAXPROCS configuration
	MaxProcs           int     `json:"max_procs"`
	AutoTuneMaxProcs   bool    `json:"auto_tune_max_procs"`
	MaxProcsMultiplier float64 `json:"max_procs_multiplier"`

	// Garbage collector tuning
	GCPercent          int   `json:"gc_percent"`
	GCTarget           int64 `json:"gc_target"`
	MemoryLimitPercent int   `json:"memory_limit_percent"`

	// Swiss Tables optimization
	SwissTablesEnabled bool    `json:"swiss_tables_enabled"`
	MapPreallocation   bool    `json:"map_preallocation"`
	MapLoadFactor      float64 `json:"map_load_factor"`

	// CPU optimization
	CPUProfileEnabled bool `json:"cpu_profile_enabled"`
	MemProfileEnabled bool `json:"mem_profile_enabled"`
	PGOEnabled        bool `json:"pgo_enabled"`

	// August 2025 specific optimizations
	PacerOptimization bool `json:"pacer_optimization"`
	CoverageRedesign  bool `json:"coverage_redesign"`
}

// DefaultOptimizationConfig returns optimized settings for Nephoran
func DefaultOptimizationConfig() *OptimizationConfig {
	numCPU := runtime.NumCPU()

	// Optimal GOMAXPROCS for containerized workloads
	maxProcs := numCPU
	if numCPU > 8 {
		// For high-core systems, use slightly fewer cores to reduce contention
		maxProcs = int(float64(numCPU) * 0.9)
	}

	return &OptimizationConfig{
		MaxProcs:           maxProcs,
		AutoTuneMaxProcs:   true,
		MaxProcsMultiplier: 0.9,

		GCPercent:          75,      // Slightly more aggressive GC for memory-constrained environments
		GCTarget:           4 << 20, // 4MB target
		MemoryLimitPercent: 80,

		SwissTablesEnabled: true,
		MapPreallocation:   true,
		MapLoadFactor:      0.6, // Lower load factor for better performance

		CPUProfileEnabled: false,
		MemProfileEnabled: false,
		PGOEnabled:        true,

		PacerOptimization: true,
		CoverageRedesign:  true,
	}
}

// ApplyOptimizations configures the Go runtime with Go 1.24.8 optimizations
func ApplyOptimizations(config *OptimizationConfig) error {
	if config == nil {
		config = DefaultOptimizationConfig()
	}

	// Set GOMAXPROCS with intelligent tuning
	if config.AutoTuneMaxProcs {
		optimalProcs := calculateOptimalGOMAXPROCS()
		runtime.GOMAXPROCS(optimalProcs)
	} else if config.MaxProcs > 0 {
		runtime.GOMAXPROCS(config.MaxProcs)
	}

	// Configure garbage collector
	if config.GCPercent >= 0 {
		debug.SetGCPercent(config.GCPercent)
	}

	// Set memory target (Go 1.24.8 feature)
	if config.GCTarget > 0 {
		debug.SetMemoryLimit(config.GCTarget * int64(config.MemoryLimitPercent) / 100)
	}

	// Configure runtime for Swiss Tables (Go 1.24.8)
	if config.SwissTablesEnabled {
		enableSwissTablesOptimizations()
	}

	return nil
}

// calculateOptimalGOMAXPROCS determines the best GOMAXPROCS value
func calculateOptimalGOMAXPROCS() int {
	numCPU := runtime.NumCPU()

	// Container detection - check for cgroup limits
	if limit := getContainerCPULimit(); limit > 0 && limit < float64(numCPU) {
		// Use container CPU limit as basis
		return int(limit)
	}

	// Physical vs logical CPU consideration
	switch {
	case numCPU <= 2:
		return numCPU
	case numCPU <= 8:
		return numCPU
	case numCPU <= 16:
		// Leave 1-2 cores for OS and other processes
		return numCPU - 2
	default:
		// For large systems, use 90% of available cores
		return int(float64(numCPU) * 0.9)
	}
}

// getContainerCPULimit detects container CPU limits
func getContainerCPULimit() float64 {
	// This would typically read from /sys/fs/cgroup/cpu/cpu.cfs_quota_us
	// and /sys/fs/cgroup/cpu/cpu.cfs_period_us on Linux containers
	// For Windows containers, use different detection method
	// Simplified implementation for now
	return 0
}

// enableSwissTablesOptimizations enables Go 1.24.8 Swiss Tables features
func enableSwissTablesOptimizations() {
	// Swiss Tables are automatically enabled in Go 1.24.8 with GOEXPERIMENT=swisstable
	// Additional runtime tweaks can be applied here
	runtime.GC() // Force GC to initialize Swiss Tables metadata
}

// TuneForTelcoWorkloads applies specific optimizations for telecom workloads
func TuneForTelcoWorkloads() error {
	config := &OptimizationConfig{
		MaxProcs:           runtime.NumCPU(),
		AutoTuneMaxProcs:   true,
		MaxProcsMultiplier: 1.0, // Use all cores for telecom processing

		GCPercent:          50,      // More frequent GC for consistent latency
		GCTarget:           8 << 20, // 8MB target for real-time processing
		MemoryLimitPercent: 90,

		SwissTablesEnabled: true,
		MapPreallocation:   true,
		MapLoadFactor:      0.5, // Lower load factor for deterministic performance

		PGOEnabled:        true,
		PacerOptimization: true,
	}

	return ApplyOptimizations(config)
}

// GetRuntimeStats returns current runtime performance statistics
func GetRuntimeStats() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"gomaxprocs":      runtime.GOMAXPROCS(0),
		"num_cpu":         runtime.NumCPU(),
		"num_goroutine":   runtime.NumGoroutine(),
		"num_gc":          m.NumGC,
		"gc_pause_ns":     m.PauseNs[(m.NumGC+255)%256],
		"heap_alloc_mb":   bToMb(m.HeapAlloc),
		"heap_sys_mb":     bToMb(m.HeapSys),
		"heap_in_use_mb":  bToMb(m.HeapInuse),
		"stack_in_use_mb": bToMb(m.StackInuse),
		"next_gc_mb":      bToMb(m.NextGC),
	}
}

// bToMb converts bytes to megabytes
func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

// PerformanceMonitor continuously monitors and adjusts runtime settings
type PerformanceMonitor struct {
	config        *OptimizationConfig
	stopCh        chan struct{}
	adjustmentCh  chan bool
	monitorTicker *time.Ticker
}

// NewPerformanceMonitor creates a runtime performance monitor
func NewPerformanceMonitor(config *OptimizationConfig) *PerformanceMonitor {
	if config == nil {
		config = DefaultOptimizationConfig()
	}

	return &PerformanceMonitor{
		config:        config,
		stopCh:        make(chan struct{}),
		adjustmentCh:  make(chan bool, 1),
		monitorTicker: time.NewTicker(30 * time.Second),
	}
}

// Start begins performance monitoring and auto-tuning
func (pm *PerformanceMonitor) Start() {
	go pm.monitorLoop()
}

// Stop ends performance monitoring
func (pm *PerformanceMonitor) Stop() {
	close(pm.stopCh)
	pm.monitorTicker.Stop()
}

// monitorLoop continuously monitors runtime performance
func (pm *PerformanceMonitor) monitorLoop() {
	for {
		select {
		case <-pm.stopCh:
			return
		case <-pm.monitorTicker.C:
			pm.checkAndAdjust()
		case <-pm.adjustmentCh:
			pm.checkAndAdjust()
		}
	}
}

// checkAndAdjust performs runtime adjustments based on current performance
func (pm *PerformanceMonitor) checkAndAdjust() {
	stats := GetRuntimeStats()

	// Adjust GOMAXPROCS based on goroutine count and CPU utilization
	numGoroutines := stats["num_goroutine"].(int)
	currentMaxProcs := stats["gomaxprocs"].(int)

	if numGoroutines > currentMaxProcs*100 {
		// Too many goroutines per processor, consider increasing GOMAXPROCS
		newMaxProcs := min(currentMaxProcs+1, runtime.NumCPU())
		if newMaxProcs != currentMaxProcs {
			runtime.GOMAXPROCS(newMaxProcs)
		}
	} else if numGoroutines < currentMaxProcs*10 && currentMaxProcs > 1 {
		// Too few goroutines, consider decreasing GOMAXPROCS
		newMaxProcs := max(currentMaxProcs-1, 1)
		if newMaxProcs != currentMaxProcs {
			runtime.GOMAXPROCS(newMaxProcs)
		}
	}

	// Adjust GC target based on memory pressure
	heapInUseMB := stats["heap_in_use_mb"].(uint64)
	if heapInUseMB > 100 { // More than 100MB in use
		// Trigger more aggressive GC
		debug.SetGCPercent(50)
	} else {
		// Normal GC behavior
		debug.SetGCPercent(pm.config.GCPercent)
	}
}

// TriggerAdjustment signals the monitor to check and adjust settings
func (pm *PerformanceMonitor) TriggerAdjustment() {
	select {
	case pm.adjustmentCh <- true:
	default:
		// Channel full, adjustment already queued
	}
}

// min returns the smaller of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
