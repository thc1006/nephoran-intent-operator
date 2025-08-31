//go:build go1.24

package performance

import "time"

// DefaultProfilerConfig returns a default ProfilerConfig with sensible defaults
func DefaultProfilerConfig() *ProfilerConfig {
	return &ProfilerConfig{
		OutputDirectory:        "/tmp/nephoran-profiles",
		ProfilePrefix:          "nephoran",
		CompressionEnabled:     true,
		RetentionDays:          7,
		CPUProfileRate:         100,           // 100 samples/sec
		MemProfileRate:         1024 * 1024,   // 1MB
		BlockProfileRate:       1,
		MutexProfileRate:       1,
		GoroutineThreshold:     1000,
		ContinuousInterval:     5 * time.Minute,
		AutoAnalysisEnabled:    true,
		OptimizationHints:      true,
		ExecutionTracing:       true,
		GCProfileEnabled:       true,
		AllocProfileEnabled:    true,
		ThreadCreationProfile:  true,
		HTTPEnabled:            true,
		HTTPAddress:            ":6060",
		HTTPAuth:               false,
		PrometheusIntegration:  true,
		AlertingEnabled:        true,
		SlackIntegration:       false,
		EnableCPUProfiling:     true,
		EnableMemoryProfiling:  true,
		EnableGoroutineMonitoring: true,
	}
}