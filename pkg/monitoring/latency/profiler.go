package latency

import (
	"context"
	"fmt"
	"runtime"
	"runtime/pprof"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel/trace"
)

// ComponentLatency represents latency breakdown for a specific component.

type ComponentLatency struct {
	Component string `json:"component"`

	Duration time.Duration `json:"duration"`

	StartTime time.Time `json:"start_time"`

	EndTime time.Time `json:"end_time"`

	SubComponents []ComponentLatency `json:"sub_components,omitempty"`

	ResourceMetrics *ResourceMetrics `json:"resource_metrics,omitempty"`

	NetworkLatency *NetworkMetrics `json:"network_latency,omitempty"`

	DatabaseLatency *DatabaseMetrics `json:"database_latency,omitempty"`
}

// ResourceMetrics captures resource utilization during component execution.

type ResourceMetrics struct {
	CPUUsage float64 `json:"cpu_usage"`

	MemoryUsage int64 `json:"memory_usage_bytes"`

	GoroutineCount int `json:"goroutine_count"`

	IOWaitTime time.Duration `json:"io_wait_time"`

	ContextSwitches int64 `json:"context_switches"`

	GCPauseTime time.Duration `json:"gc_pause_time"`
}

// NetworkMetrics captures network-related latencies.

type NetworkMetrics struct {
	DNSLookupTime time.Duration `json:"dns_lookup_time"`

	ConnectionTime time.Duration `json:"connection_time"`

	TLSHandshakeTime time.Duration `json:"tls_handshake_time"`

	RequestTime time.Duration `json:"request_time"`

	ResponseTime time.Duration `json:"response_time"`

	TotalRoundTrip time.Duration `json:"total_round_trip"`

	BytesSent int64 `json:"bytes_sent"`

	BytesReceived int64 `json:"bytes_received"`
}

// DatabaseMetrics captures database operation latencies.

type DatabaseMetrics struct {
	QueryPrepTime time.Duration `json:"query_prep_time"`

	QueryExecTime time.Duration `json:"query_exec_time"`

	RowsFetchTime time.Duration `json:"rows_fetch_time"`

	ConnectionWait time.Duration `json:"connection_wait"`

	LockWaitTime time.Duration `json:"lock_wait_time"`

	TransactionTime time.Duration `json:"transaction_time"`

	RowsAffected int64 `json:"rows_affected"`
}

// LatencyProfiler provides deep instrumentation of the intent processing pipeline.

type LatencyProfiler struct {
	mu sync.RWMutex

	// Core profiling data.

	profiles map[string]*IntentProfile

	// Component-level timing.

	componentTimers map[string]*ComponentTimer

	// Resource monitoring.

	resourceMonitor *ResourceMonitor

	// Network profiling.

	networkProfiler *NetworkProfiler

	// Database profiling.

	databaseProfiler *DatabaseProfiler

	// CPU profiling integration.

	cpuProfiler *CPUProfiler

	// Flame graph generation.

	flameGraphGen *FlameGraphGenerator

	// Metrics.

	metrics *ProfilerMetrics

	// Configuration.

	config *ProfilerConfig

	// Active traces.

	activeTraces map[string]*ProfileTrace

	// Performance counters.

	counters *PerformanceCounters
}

// ProfilerConfig contains configuration for the latency profiler.

type ProfilerConfig struct {

	// Profiling configuration.

	EnableCPUProfiling bool `json:"enable_cpu_profiling"`

	EnableMemoryProfiling bool `json:"enable_memory_profiling"`

	EnableBlockProfiling bool `json:"enable_block_profiling"`

	EnableMutexProfiling bool `json:"enable_mutex_profiling"`

	ProfileSamplingRate int `json:"profile_sampling_rate"`

	// Instrumentation configuration.

	TraceEveryNthRequest int `json:"trace_every_nth_request"`

	DetailedProfilingRatio float64 `json:"detailed_profiling_ratio"`

	MaxProfileDuration time.Duration `json:"max_profile_duration"`

	// Resource monitoring.

	ResourceCheckInterval time.Duration `json:"resource_check_interval"`

	EnableResourceTracking bool `json:"enable_resource_tracking"`

	// Network profiling.

	EnableNetworkProfiling bool `json:"enable_network_profiling"`

	CapturePayloadSize bool `json:"capture_payload_size"`

	// Database profiling.

	EnableDatabaseProfiling bool `json:"enable_database_profiling"`

	SlowQueryThreshold time.Duration `json:"slow_query_threshold"`

	// Storage configuration.

	ProfileRetentionDays int `json:"profile_retention_days"`

	MaxProfilesStored int `json:"max_profiles_stored"`
}

// IntentProfile represents a complete profile of an intent processing request.

type IntentProfile struct {
	ID string `json:"id"`

	IntentID string `json:"intent_id"`

	StartTime time.Time `json:"start_time"`

	EndTime time.Time `json:"end_time"`

	TotalDuration time.Duration `json:"total_duration"`

	Components map[string]*ComponentLatency `json:"components"`

	CriticalPath []string `json:"critical_path"`

	ResourceProfile *ResourceProfile `json:"resource_profile"`

	NetworkProfile *NetworkProfile `json:"network_profile"`

	DatabaseProfile *DatabaseProfile `json:"database_profile"`

	FlameGraph []byte `json:"flame_graph,omitempty"`

	Bottlenecks []Bottleneck `json:"bottlenecks"`

	Anomalies []LatencyAnomaly `json:"anomalies"`
}

// ComponentTimer tracks timing for individual components.

type ComponentTimer struct {
	name string

	startTime time.Time

	spans []TimeSpan

	mu sync.Mutex
}

// TimeSpan represents a time interval with metadata.

type TimeSpan struct {
	Name string `json:"name"`

	Start time.Time `json:"start"`

	End time.Time `json:"end"`

	Duration time.Duration `json:"duration"`

	Tags map[string]string `json:"tags"`
}

// ResourceMonitor tracks resource utilization during profiling.

type ResourceMonitor struct {
	mu sync.RWMutex

	samples []ResourceSample

	startMemStats runtime.MemStats

	currentMemStats runtime.MemStats

	goroutineCount int32

	cpuPercent float64
}

// ResourceSample represents a point-in-time resource measurement.

type ResourceSample struct {
	Timestamp time.Time `json:"timestamp"`

	CPUPercent float64 `json:"cpu_percent"`

	MemoryAlloc uint64 `json:"memory_alloc"`

	MemoryHeap uint64 `json:"memory_heap"`

	GoroutineCount int `json:"goroutine_count"`

	GCPauses time.Duration `json:"gc_pauses"`

	ContextSwitches int64 `json:"context_switches"`
}

// ProfilerMetrics contains Prometheus metrics for the profiler.

type ProfilerMetrics struct {
	profilesCreated prometheus.Counter

	profileDuration prometheus.Histogram

	componentDuration *prometheus.HistogramVec

	resourceUtilization *prometheus.GaugeVec

	networkLatency *prometheus.HistogramVec

	databaseLatency *prometheus.HistogramVec

	bottlenecksDetected prometheus.Counter

	anomaliesDetected prometheus.Counter
}

// ProfileTrace represents an active profiling trace.

type ProfileTrace struct {
	TraceID string

	StartTime time.Time

	ComponentStack []string

	CurrentComponent string

	SpanContext trace.SpanContext

	ResourceBaseline *ResourceMetrics
}

// PerformanceCounters tracks various performance counters.

type PerformanceCounters struct {
	intentCount atomic.Int64

	totalLatency atomic.Int64

	componentLatencies sync.Map // map[string]*atomic.Int64

	networkCalls atomic.Int64

	databaseQueries atomic.Int64

	cacheHits atomic.Int64

	cacheMisses atomic.Int64
}

// NewLatencyProfiler creates a new latency profiler instance.

func NewLatencyProfiler(config *ProfilerConfig) *LatencyProfiler {

	if config == nil {

		config = DefaultProfilerConfig()

	}

	profiler := &LatencyProfiler{

		profiles: make(map[string]*IntentProfile),

		componentTimers: make(map[string]*ComponentTimer),

		activeTraces: make(map[string]*ProfileTrace),

		config: config,

		counters: &PerformanceCounters{},
	}

	// Initialize sub-profilers.

	profiler.resourceMonitor = NewResourceMonitor(config)

	profiler.networkProfiler = NewNetworkProfiler(config)

	profiler.databaseProfiler = NewDatabaseProfiler(config)

	profiler.cpuProfiler = NewCPUProfiler(config)

	profiler.flameGraphGen = NewFlameGraphGenerator()

	// Initialize metrics.

	profiler.initMetrics()

	// Start background monitoring if enabled.

	if config.EnableResourceTracking {

		go profiler.runResourceMonitoring()

	}

	// Configure runtime profiling.

	if config.EnableBlockProfiling {

		runtime.SetBlockProfileRate(config.ProfileSamplingRate)

	}

	if config.EnableMutexProfiling {

		runtime.SetMutexProfileFraction(config.ProfileSamplingRate)

	}

	return profiler

}

// StartIntentProfiling begins profiling an intent processing request.

func (p *LatencyProfiler) StartIntentProfiling(ctx context.Context, intentID string) context.Context {

	profileID := fmt.Sprintf("profile-%s-%d", intentID, time.Now().UnixNano())

	profile := &IntentProfile{

		ID: profileID,

		IntentID: intentID,

		StartTime: time.Now(),

		Components: make(map[string]*ComponentLatency),
	}

	// Store the profile.

	p.mu.Lock()

	p.profiles[profileID] = profile

	p.mu.Unlock()

	// Create trace.

	trace := &ProfileTrace{

		TraceID: profileID,

		StartTime: time.Now(),

		ComponentStack: []string{},

		ResourceBaseline: p.captureResourceMetrics(),
	}

	p.mu.Lock()

	p.activeTraces[profileID] = trace

	p.mu.Unlock()

	// Store profile ID in context.

	ctx = context.WithValue(ctx, "profile_id", profileID)

	// Start CPU profiling if enabled for this request.

	if p.shouldProfileRequest() && p.config.EnableCPUProfiling {

		p.cpuProfiler.Start(profileID)

	}

	// Increment counters.

	p.counters.intentCount.Add(1)

	return ctx

}

// EndIntentProfiling completes profiling for an intent.

func (p *LatencyProfiler) EndIntentProfiling(ctx context.Context) *IntentProfile {

	profileID, ok := ctx.Value("profile_id").(string)

	if !ok {

		return nil

	}

	p.mu.Lock()

	profile, exists := p.profiles[profileID]

	trace := p.activeTraces[profileID]

	delete(p.activeTraces, profileID)

	p.mu.Unlock()

	if !exists || profile == nil {

		return nil

	}

	// Finalize profile.

	profile.EndTime = time.Now()

	profile.TotalDuration = profile.EndTime.Sub(profile.StartTime)

	// Stop CPU profiling.

	if p.config.EnableCPUProfiling {

		cpuProfile := p.cpuProfiler.Stop(profileID)

		if cpuProfile != nil {

			profile.FlameGraph = p.flameGraphGen.Generate(cpuProfile)

		}

	}

	// Capture final resource metrics.

	if trace != nil && trace.ResourceBaseline != nil {

		currentMetrics := p.captureResourceMetrics()

		profile.ResourceProfile = p.calculateResourceDelta(trace.ResourceBaseline, currentMetrics)

	}

	// Analyze critical path.

	profile.CriticalPath = p.analyzeCriticalPath(profile)

	// Detect bottlenecks.

	profile.Bottlenecks = p.detectBottlenecks(profile)

	// Detect anomalies.

	profile.Anomalies = p.detectAnomalies(profile)

	// Update metrics.

	p.updateMetrics(profile)

	// Update counters.

	p.counters.totalLatency.Add(int64(profile.TotalDuration))

	return profile

}

// StartComponentTiming begins timing a specific component.

func (p *LatencyProfiler) StartComponentTiming(ctx context.Context, component string) context.Context {

	profileID, ok := ctx.Value("profile_id").(string)

	if !ok {

		return ctx

	}

	timer := &ComponentTimer{

		name: component,

		startTime: time.Now(),

		spans: []TimeSpan{},
	}

	p.mu.Lock()

	p.componentTimers[fmt.Sprintf("%s-%s", profileID, component)] = timer

	// Update component stack.

	if trace, exists := p.activeTraces[profileID]; exists {

		trace.ComponentStack = append(trace.ComponentStack, component)

		trace.CurrentComponent = component

	}

	p.mu.Unlock()

	// Store component in context.

	ctx = context.WithValue(ctx, "current_component", component)

	return ctx

}

// EndComponentTiming completes timing for a component.

func (p *LatencyProfiler) EndComponentTiming(ctx context.Context, component string) *ComponentLatency {

	profileID, ok := ctx.Value("profile_id").(string)

	if !ok {

		return nil

	}

	timerKey := fmt.Sprintf("%s-%s", profileID, component)

	p.mu.Lock()

	timer, exists := p.componentTimers[timerKey]

	delete(p.componentTimers, timerKey)

	// Update component stack.

	if trace, exists := p.activeTraces[profileID]; exists && len(trace.ComponentStack) > 0 {

		trace.ComponentStack = trace.ComponentStack[:len(trace.ComponentStack)-1]

		if len(trace.ComponentStack) > 0 {

			trace.CurrentComponent = trace.ComponentStack[len(trace.ComponentStack)-1]

		} else {

			trace.CurrentComponent = ""

		}

	}

	p.mu.Unlock()

	if !exists || timer == nil {

		return nil

	}

	endTime := time.Now()

	duration := endTime.Sub(timer.startTime)

	componentLatency := &ComponentLatency{

		Component: component,

		Duration: duration,

		StartTime: timer.startTime,

		EndTime: endTime,
	}

	// Capture resource metrics for this component.

	if p.config.EnableResourceTracking {

		componentLatency.ResourceMetrics = p.captureResourceMetrics()

	}

	// Store in profile.

	p.mu.Lock()

	if profile, exists := p.profiles[profileID]; exists {

		profile.Components[component] = componentLatency

	}

	p.mu.Unlock()

	// Update component counter.

	if counter, ok := p.counters.componentLatencies.LoadOrStore(component, &atomic.Int64{}); ok {

		counter.(*atomic.Int64).Add(int64(duration))

	}

	return componentLatency

}

// RecordNetworkLatency records network operation latency.

func (p *LatencyProfiler) RecordNetworkLatency(ctx context.Context, operation string, metrics *NetworkMetrics) {

	if !p.config.EnableNetworkProfiling {

		return

	}

	profileID, ok := ctx.Value("profile_id").(string)

	if !ok {

		return

	}

	component, _ := ctx.Value("current_component").(string)

	p.networkProfiler.Record(profileID, component, operation, metrics)

	p.counters.networkCalls.Add(1)

	// Update component latency with network metrics.

	p.mu.Lock()

	if profile, exists := p.profiles[profileID]; exists {

		if compLatency, exists := profile.Components[component]; exists {

			compLatency.NetworkLatency = metrics

		}

	}

	p.mu.Unlock()

}

// RecordDatabaseLatency records database operation latency.

func (p *LatencyProfiler) RecordDatabaseLatency(ctx context.Context, operation string, metrics *DatabaseMetrics) {

	if !p.config.EnableDatabaseProfiling {

		return

	}

	profileID, ok := ctx.Value("profile_id").(string)

	if !ok {

		return

	}

	component, _ := ctx.Value("current_component").(string)

	p.databaseProfiler.Record(profileID, component, operation, metrics)

	p.counters.databaseQueries.Add(1)

	// Update component latency with database metrics.

	p.mu.Lock()

	if profile, exists := p.profiles[profileID]; exists {

		if compLatency, exists := profile.Components[component]; exists {

			compLatency.DatabaseLatency = metrics

		}

	}

	p.mu.Unlock()

}

// RecordCacheAccess records cache hit/miss.

func (p *LatencyProfiler) RecordCacheAccess(ctx context.Context, hit bool, latency time.Duration) {

	if hit {

		p.counters.cacheHits.Add(1)

	} else {

		p.counters.cacheMisses.Add(1)

	}

	// Record in component timing if active.

	if component, ok := ctx.Value("current_component").(string); ok {

		p.recordSubComponentTiming(ctx, fmt.Sprintf("cache_%s", component), latency)

	}

}

// Helper methods.

func (p *LatencyProfiler) shouldProfileRequest() bool {

	// Implement sampling logic based on configuration.

	if p.config.TraceEveryNthRequest <= 0 {

		return false

	}

	count := p.counters.intentCount.Load()

	return count%int64(p.config.TraceEveryNthRequest) == 0

}

func (p *LatencyProfiler) captureResourceMetrics() *ResourceMetrics {

	var memStats runtime.MemStats

	runtime.ReadMemStats(&memStats)

	return &ResourceMetrics{

		CPUUsage: p.resourceMonitor.cpuPercent,

		MemoryUsage: int64(memStats.Alloc),

		GoroutineCount: runtime.NumGoroutine(),

		GCPauseTime: time.Duration(memStats.PauseTotalNs),
	}

}

func (p *LatencyProfiler) calculateResourceDelta(baseline, current *ResourceMetrics) *ResourceProfile {

	if baseline == nil || current == nil {

		return nil

	}

	return &ResourceProfile{

		CPUUsageDelta: current.CPUUsage - baseline.CPUUsage,

		MemoryUsageDelta: current.MemoryUsage - baseline.MemoryUsage,

		GoroutineDelta: current.GoroutineCount - baseline.GoroutineCount,

		GCPauseTimeDelta: current.GCPauseTime - baseline.GCPauseTime,
	}

}

func (p *LatencyProfiler) analyzeCriticalPath(profile *IntentProfile) []string {

	// Identify the longest sequence of dependent components.

	var criticalPath []string

	// Simple implementation: return components sorted by duration.

	// In production, this would use dependency graph analysis.

	type componentDuration struct {
		name string

		duration time.Duration
	}

	var components []componentDuration

	for name, latency := range profile.Components {

		components = append(components, componentDuration{

			name: name,

			duration: latency.Duration,
		})

	}

	// Sort by duration (descending).

	for i := 0; i < len(components); i++ {

		for j := i + 1; j < len(components); j++ {

			if components[j].duration > components[i].duration {

				components[i], components[j] = components[j], components[i]

			}

		}

	}

	// Return top components as critical path.

	for i := 0; i < len(components) && i < 5; i++ {

		criticalPath = append(criticalPath, components[i].name)

	}

	return criticalPath

}

func (p *LatencyProfiler) detectBottlenecks(profile *IntentProfile) []Bottleneck {

	var bottlenecks []Bottleneck

	// Check for components exceeding thresholds.

	for component, latency := range profile.Components {

		// Component taking more than 30% of total time.

		if latency.Duration > profile.TotalDuration*30/100 {

			bottlenecks = append(bottlenecks, Bottleneck{

				Component: component,

				Type: "DURATION",

				Impact: float64(latency.Duration) / float64(profile.TotalDuration),

				Description: fmt.Sprintf("Component %s takes %.2f%% of total time", component, float64(latency.Duration)/float64(profile.TotalDuration)*100),
			})

		}

		// High resource usage.

		if latency.ResourceMetrics != nil && latency.ResourceMetrics.CPUUsage > 80 {

			bottlenecks = append(bottlenecks, Bottleneck{

				Component: component,

				Type: "CPU",

				Impact: latency.ResourceMetrics.CPUUsage / 100,

				Description: fmt.Sprintf("High CPU usage: %.2f%%", latency.ResourceMetrics.CPUUsage),
			})

		}

		// Database bottlenecks.

		if latency.DatabaseLatency != nil && latency.DatabaseLatency.LockWaitTime > 100*time.Millisecond {

			bottlenecks = append(bottlenecks, Bottleneck{

				Component: component,

				Type: "DATABASE_LOCK",

				Impact: float64(latency.DatabaseLatency.LockWaitTime) / float64(latency.Duration),

				Description: fmt.Sprintf("Database lock wait: %v", latency.DatabaseLatency.LockWaitTime),
			})

		}

	}

	return bottlenecks

}

func (p *LatencyProfiler) detectAnomalies(profile *IntentProfile) []LatencyAnomaly {

	var anomalies []LatencyAnomaly

	// Compare against historical averages.

	for component, latency := range profile.Components {

		if avgLatency := p.getAverageLatency(component); avgLatency > 0 {

			deviation := float64(latency.Duration) / float64(avgLatency)

			if deviation > 2.0 { // More than 2x average

				anomalies = append(anomalies, LatencyAnomaly{

					Component: component,

					Type: "HIGH_LATENCY",

					Deviation: deviation,

					Message: fmt.Sprintf("Latency %.2fx higher than average", deviation),
				})

			}

		}

	}

	return anomalies

}

func (p *LatencyProfiler) getAverageLatency(component string) time.Duration {

	if counter, ok := p.counters.componentLatencies.Load(component); ok {

		totalLatency := counter.(*atomic.Int64).Load()

		intentCount := p.counters.intentCount.Load()

		if intentCount > 0 {

			return time.Duration(totalLatency / intentCount)

		}

	}

	return 0

}

func (p *LatencyProfiler) recordSubComponentTiming(ctx context.Context, subComponent string, duration time.Duration) {

	// Record timing for sub-components within a main component.

	profileID, _ := ctx.Value("profile_id").(string)

	component, _ := ctx.Value("current_component").(string)

	if profileID == "" || component == "" {

		return

	}

	p.mu.Lock()

	defer p.mu.Unlock()

	if profile, exists := p.profiles[profileID]; exists {

		if compLatency, exists := profile.Components[component]; exists {

			subComp := ComponentLatency{

				Component: subComponent,

				Duration: duration,

				StartTime: time.Now().Add(-duration),

				EndTime: time.Now(),
			}

			compLatency.SubComponents = append(compLatency.SubComponents, subComp)

		}

	}

}

func (p *LatencyProfiler) runResourceMonitoring() {

	ticker := time.NewTicker(p.config.ResourceCheckInterval)

	defer ticker.Stop()

	for range ticker.C {

		p.resourceMonitor.Sample()

	}

}

func (p *LatencyProfiler) initMetrics() {

	p.metrics = &ProfilerMetrics{

		profilesCreated: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "nephoran_profiler_profiles_created_total",

			Help: "Total number of latency profiles created",
		}),

		profileDuration: prometheus.NewHistogram(prometheus.HistogramOpts{

			Name: "nephoran_profiler_profile_duration_seconds",

			Help: "Duration of profiled intents",

			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 30.0},
		}),

		componentDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name: "nephoran_profiler_component_duration_seconds",

			Help: "Duration of individual components",

			Buckets: []float64{0.001, 0.01, 0.1, 0.5, 1.0, 2.0, 5.0},
		}, []string{"component"}),

		resourceUtilization: prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_profiler_resource_utilization",

			Help: "Resource utilization during profiling",
		}, []string{"resource_type"}),

		networkLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name: "nephoran_profiler_network_latency_seconds",

			Help: "Network operation latency",

			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0},
		}, []string{"operation"}),

		databaseLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name: "nephoran_profiler_database_latency_seconds",

			Help: "Database operation latency",

			Buckets: []float64{0.0001, 0.001, 0.01, 0.1, 0.5, 1.0},
		}, []string{"operation"}),

		bottlenecksDetected: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "nephoran_profiler_bottlenecks_detected_total",

			Help: "Total number of bottlenecks detected",
		}),

		anomaliesDetected: prometheus.NewCounter(prometheus.CounterOpts{

			Name: "nephoran_profiler_anomalies_detected_total",

			Help: "Total number of anomalies detected",
		}),
	}

}

func (p *LatencyProfiler) updateMetrics(profile *IntentProfile) {

	p.metrics.profilesCreated.Inc()

	p.metrics.profileDuration.Observe(profile.TotalDuration.Seconds())

	for component, latency := range profile.Components {

		p.metrics.componentDuration.WithLabelValues(component).Observe(latency.Duration.Seconds())

	}

	p.metrics.bottlenecksDetected.Add(float64(len(profile.Bottlenecks)))

	p.metrics.anomaliesDetected.Add(float64(len(profile.Anomalies)))

}

// GetProfile retrieves a specific profile by ID.

func (p *LatencyProfiler) GetProfile(profileID string) *IntentProfile {

	p.mu.RLock()

	defer p.mu.RUnlock()

	return p.profiles[profileID]

}

// GetRecentProfiles returns the most recent profiles.

func (p *LatencyProfiler) GetRecentProfiles(count int) []*IntentProfile {

	p.mu.RLock()

	defer p.mu.RUnlock()

	var profiles []*IntentProfile

	for _, profile := range p.profiles {

		profiles = append(profiles, profile)

		if len(profiles) >= count {

			break

		}

	}

	return profiles

}

// GetStatistics returns profiling statistics.

func (p *LatencyProfiler) GetStatistics() *ProfilerStatistics {

	return &ProfilerStatistics{

		TotalIntents: p.counters.intentCount.Load(),

		AverageLatency: time.Duration(p.counters.totalLatency.Load() / max(p.counters.intentCount.Load(), 1)),

		NetworkCalls: p.counters.networkCalls.Load(),

		DatabaseQueries: p.counters.databaseQueries.Load(),

		CacheHitRate: p.calculateCacheHitRate(),
	}

}

func (p *LatencyProfiler) calculateCacheHitRate() float64 {

	hits := p.counters.cacheHits.Load()

	misses := p.counters.cacheMisses.Load()

	total := hits + misses

	if total == 0 {

		return 0

	}

	return float64(hits) / float64(total)

}

// Supporting types.

// Bottleneck represents a bottleneck.

type Bottleneck struct {
	Component string `json:"component"`

	Type string `json:"type"`

	Impact float64 `json:"impact"`

	Description string `json:"description"`
}

// LatencyAnomaly represents a latencyanomaly.

type LatencyAnomaly struct {
	Component string `json:"component"`

	Type string `json:"type"`

	Deviation float64 `json:"deviation"`

	Message string `json:"message"`
}

// ResourceProfile represents a resourceprofile.

type ResourceProfile struct {
	CPUUsageDelta float64 `json:"cpu_usage_delta"`

	MemoryUsageDelta int64 `json:"memory_usage_delta"`

	GoroutineDelta int `json:"goroutine_delta"`

	GCPauseTimeDelta time.Duration `json:"gc_pause_time_delta"`
}

// NetworkProfile represents a networkprofile.

type NetworkProfile struct {
	TotalCalls int `json:"total_calls"`

	TotalLatency time.Duration `json:"total_latency"`

	AverageLatency time.Duration `json:"average_latency"`

	SlowCalls int `json:"slow_calls"`
}

// DatabaseProfile represents a databaseprofile.

type DatabaseProfile struct {
	TotalQueries int `json:"total_queries"`

	TotalLatency time.Duration `json:"total_latency"`

	SlowQueries int `json:"slow_queries"`

	LockWaitTime time.Duration `json:"lock_wait_time"`
}

// ProfilerStatistics represents a profilerstatistics.

type ProfilerStatistics struct {
	TotalIntents int64 `json:"total_intents"`

	AverageLatency time.Duration `json:"average_latency"`

	NetworkCalls int64 `json:"network_calls"`

	DatabaseQueries int64 `json:"database_queries"`

	CacheHitRate float64 `json:"cache_hit_rate"`
}

// Helper functions for sub-profilers.

// NewResourceMonitor performs newresourcemonitor operation.

func NewResourceMonitor(config *ProfilerConfig) *ResourceMonitor {

	return &ResourceMonitor{

		samples: make([]ResourceSample, 0, 1000),
	}

}

// Sample performs sample operation.

func (r *ResourceMonitor) Sample() {

	var memStats runtime.MemStats

	runtime.ReadMemStats(&memStats)

	sample := ResourceSample{

		Timestamp: time.Now(),

		MemoryAlloc: memStats.Alloc,

		MemoryHeap: memStats.HeapAlloc,

		GoroutineCount: runtime.NumGoroutine(),

		GCPauses: time.Duration(memStats.PauseTotalNs),
	}

	r.mu.Lock()

	r.samples = append(r.samples, sample)

	// Keep only last 1000 samples.

	if len(r.samples) > 1000 {

		r.samples = r.samples[len(r.samples)-1000:]

	}

	r.mu.Unlock()

}

// NewNetworkProfiler performs newnetworkprofiler operation.

func NewNetworkProfiler(config *ProfilerConfig) *NetworkProfiler {

	return &NetworkProfiler{

		profiles: make(map[string]*NetworkProfile),
	}

}

// NetworkProfiler represents a networkprofiler.

type NetworkProfiler struct {
	mu sync.RWMutex

	profiles map[string]*NetworkProfile
}

// Record performs record operation.

func (n *NetworkProfiler) Record(profileID, component, operation string, metrics *NetworkMetrics) {

	n.mu.Lock()

	defer n.mu.Unlock()

	if _, exists := n.profiles[profileID]; !exists {

		n.profiles[profileID] = &NetworkProfile{}

	}

	profile := n.profiles[profileID]

	profile.TotalCalls++

	profile.TotalLatency += metrics.TotalRoundTrip

	if metrics.TotalRoundTrip > 500*time.Millisecond {

		profile.SlowCalls++

	}

}

// NewDatabaseProfiler performs newdatabaseprofiler operation.

func NewDatabaseProfiler(config *ProfilerConfig) *DatabaseProfiler {

	return &DatabaseProfiler{

		profiles: make(map[string]*DatabaseProfile),

		slowQueryThreshold: config.SlowQueryThreshold,
	}

}

// DatabaseProfiler represents a databaseprofiler.

type DatabaseProfiler struct {
	mu sync.RWMutex

	profiles map[string]*DatabaseProfile

	slowQueryThreshold time.Duration
}

// Record performs record operation.

func (d *DatabaseProfiler) Record(profileID, component, operation string, metrics *DatabaseMetrics) {

	d.mu.Lock()

	defer d.mu.Unlock()

	if _, exists := d.profiles[profileID]; !exists {

		d.profiles[profileID] = &DatabaseProfile{}

	}

	profile := d.profiles[profileID]

	profile.TotalQueries++

	profile.TotalLatency += metrics.QueryExecTime

	profile.LockWaitTime += metrics.LockWaitTime

	if metrics.QueryExecTime > d.slowQueryThreshold {

		profile.SlowQueries++

	}

}

// NewCPUProfiler performs newcpuprofiler operation.

func NewCPUProfiler(config *ProfilerConfig) *CPUProfiler {

	return &CPUProfiler{

		profiles: make(map[string]*pprof.Profile),

		enabled: config.EnableCPUProfiling,
	}

}

// CPUProfiler represents a cpuprofiler.

type CPUProfiler struct {
	mu sync.RWMutex

	profiles map[string]*pprof.Profile

	enabled bool
}

// Start performs start operation.

func (c *CPUProfiler) Start(profileID string) {

	if !c.enabled {

		return

	}

	// CPU profiling implementation would go here.

	// Using runtime/pprof package.

}

// Stop performs stop operation.

func (c *CPUProfiler) Stop(profileID string) *pprof.Profile {

	if !c.enabled {

		return nil

	}

	c.mu.Lock()

	defer c.mu.Unlock()

	return c.profiles[profileID]

}

// NewFlameGraphGenerator performs newflamegraphgenerator operation.

func NewFlameGraphGenerator() *FlameGraphGenerator {

	return &FlameGraphGenerator{}

}

// FlameGraphGenerator represents a flamegraphgenerator.

type FlameGraphGenerator struct{}

// Generate performs generate operation.

func (f *FlameGraphGenerator) Generate(profile *pprof.Profile) []byte {

	// Flame graph generation implementation would go here.

	// This would convert the pprof data to flame graph format.

	return nil

}

// DefaultProfilerConfig returns default configuration.

func DefaultProfilerConfig() *ProfilerConfig {

	return &ProfilerConfig{

		EnableCPUProfiling: true,

		EnableMemoryProfiling: true,

		EnableBlockProfiling: false,

		EnableMutexProfiling: false,

		ProfileSamplingRate: 100,

		TraceEveryNthRequest: 10,

		DetailedProfilingRatio: 0.1,

		MaxProfileDuration: 5 * time.Minute,

		ResourceCheckInterval: 100 * time.Millisecond,

		EnableResourceTracking: true,

		EnableNetworkProfiling: true,

		CapturePayloadSize: true,

		EnableDatabaseProfiling: true,

		SlowQueryThreshold: 100 * time.Millisecond,

		ProfileRetentionDays: 7,

		MaxProfilesStored: 10000,
	}

}

func max(a, b int64) int64 {

	if a > b {

		return a

	}

	return b

}
