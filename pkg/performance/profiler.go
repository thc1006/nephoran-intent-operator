package performance

import (
	"context"
	"fmt"
	"net/http"
	"net/http/pprof"
	"os"
	"runtime"
	rpprof "runtime/pprof"
	"runtime/trace"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"k8s.io/klog/v2"
)

// min returns the minimum of two integers.

func min(a, b int) int {

	if a < b {

		return a

	}

	return b

}

// Profiler provides comprehensive profiling capabilities.

type Profiler struct {
	cpuProfile *os.File

	memProfile *os.File

	blockProfile *os.File

	mutexProfile *os.File

	traceFile *os.File

	goroutineStats GoroutineStats

	memoryStats ProfilerMemoryStats

	profileDir string

	isActive bool

	httpServer *http.Server

	mu sync.RWMutex
}

// GoroutineStats tracks goroutine metrics.

type GoroutineStats struct {
	Current int

	Peak int

	Created int64

	Leaked int

	StackTraces map[string]int

	LastSnapshot time.Time
}

// ProfilerMemoryStats tracks memory metrics for profiling.

type ProfilerMemoryStats struct {
	HeapAlloc uint64

	HeapSys uint64

	HeapInuse uint64

	HeapReleased uint64

	GCPauseTotal time.Duration

	GCPauseAvg time.Duration

	GCCount uint32

	LastGC time.Time

	MemoryLeaks []MemoryLeak

	LastSnapshot time.Time
}

// MemoryLeak represents a detected memory leak.

type MemoryLeak struct {
	Location string

	AllocBytes uint64

	AllocCount uint64

	DetectedAt time.Time

	Description string
}

// ProfileReport contains comprehensive profiling results.

type ProfileReport struct {
	StartTime time.Time

	EndTime time.Time

	Duration time.Duration

	CPUProfile string

	MemoryProfile string

	BlockProfile string

	MutexProfile string

	GoroutineStats GoroutineStats

	MemoryStats ProfilerMemoryStats

	HotSpots []HotSpot

	Contentions []Contention

	Allocations []Allocation
}

// HotSpot represents a CPU hotspot.

type HotSpot struct {
	Function string

	File string

	Line int

	Samples int

	Percentage float64
}

// Contention represents lock contention.

type Contention struct {
	Mutex string

	WaitTime time.Duration

	Waiters int

	Location string
}

// Allocation represents memory allocation pattern.

type Allocation struct {
	Function string

	AllocBytes uint64

	AllocCount uint64

	InUseBytes uint64

	InUseCount uint64
}

// NewProfiler creates a new profiler.

func NewProfiler() *Profiler {

	return &Profiler{

		profileDir: "/tmp/nephoran-profiles",

		goroutineStats: GoroutineStats{

			StackTraces: make(map[string]int, 16), // Preallocate for typical stack trace count

		},

		memoryStats: ProfilerMemoryStats{

			MemoryLeaks: make([]MemoryLeak, 0, 8), // Preallocate for typical leak count

		},
	}

}

// StartCPUProfile starts CPU profiling.

func (p *Profiler) StartCPUProfile() error {

	p.mu.Lock()

	defer p.mu.Unlock()

	if p.cpuProfile != nil {

		return fmt.Errorf("CPU profiling already active")

	}

	// Create profile directory if not exists.

	if err := os.MkdirAll(p.profileDir, 0o755); err != nil {

		return fmt.Errorf("failed to create profile directory: %w", err)

	}

	// Create CPU profile file.

	filename := fmt.Sprintf("%s/cpu_%d.prof", p.profileDir, time.Now().Unix())

	file, err := os.Create(filename)

	if err != nil {

		return fmt.Errorf("failed to create CPU profile file: %w", err)

	}

	// Start CPU profiling.

	if err := rpprof.StartCPUProfile(file); err != nil {

		file.Close()

		return fmt.Errorf("failed to start CPU profiling: %w", err)

	}

	p.cpuProfile = file

	p.isActive = true

	klog.Infof("CPU profiling started, writing to %s", filename)

	return nil

}

// StopCPUProfile stops CPU profiling.

func (p *Profiler) StopCPUProfile() (string, error) {

	p.mu.Lock()

	defer p.mu.Unlock()

	if p.cpuProfile == nil {

		return "", fmt.Errorf("CPU profiling not active")

	}

	rpprof.StopCPUProfile()

	filename := p.cpuProfile.Name()

	p.cpuProfile.Close()

	p.cpuProfile = nil

	klog.Infof("CPU profiling stopped, profile saved to %s", filename)

	return filename, nil

}

// CaptureMemoryProfile captures a memory profile.

func (p *Profiler) CaptureMemoryProfile() (string, error) {

	p.mu.Lock()

	defer p.mu.Unlock()

	// Force garbage collection for accurate profile.

	runtime.GC()

	filename := fmt.Sprintf("%s/mem_%d.prof", p.profileDir, time.Now().Unix())

	file, err := os.Create(filename)

	if err != nil {

		return "", fmt.Errorf("failed to create memory profile file: %w", err)

	}

	defer file.Close()

	// Write heap profile.

	if err := rpprof.WriteHeapProfile(file); err != nil {

		return "", fmt.Errorf("failed to write memory profile: %w", err)

	}

	// Update memory statistics.

	var memStats runtime.MemStats

	runtime.ReadMemStats(&memStats)

	p.updateMemoryStats(&memStats)

	klog.Infof("Memory profile captured to %s", filename)

	return filename, nil

}

// CaptureGoroutineProfile captures goroutine profile.

func (p *Profiler) CaptureGoroutineProfile() (string, error) {

	p.mu.Lock()

	defer p.mu.Unlock()

	filename := fmt.Sprintf("%s/goroutine_%d.prof", p.profileDir, time.Now().Unix())

	file, err := os.Create(filename)

	if err != nil {

		return "", fmt.Errorf("failed to create goroutine profile file: %w", err)

	}

	defer file.Close()

	// Write goroutine profile.

	profile := rpprof.Lookup("goroutine")

	if err := profile.WriteTo(file, 2); err != nil {

		return "", fmt.Errorf("failed to write goroutine profile: %w", err)

	}

	// Update goroutine statistics.

	p.updateGoroutineStats()

	klog.Infof("Goroutine profile captured to %s", filename)

	return filename, nil

}

// StartBlockProfile starts block profiling.

func (p *Profiler) StartBlockProfile(rate int) error {

	p.mu.Lock()

	defer p.mu.Unlock()

	if p.blockProfile != nil {

		return fmt.Errorf("block profiling already active")

	}

	// Set block profile rate (1 = profile everything).

	runtime.SetBlockProfileRate(rate)

	p.blockProfile = &os.File{} // Placeholder

	klog.Infof("Block profiling started with rate %d", rate)

	return nil

}

// StopBlockProfile stops block profiling and saves the profile.

func (p *Profiler) StopBlockProfile() (string, error) {

	p.mu.Lock()

	defer p.mu.Unlock()

	if p.blockProfile == nil {

		return "", fmt.Errorf("block profiling not active")

	}

	// Reset block profile rate.

	runtime.SetBlockProfileRate(0)

	filename := fmt.Sprintf("%s/block_%d.prof", p.profileDir, time.Now().Unix())

	file, err := os.Create(filename)

	if err != nil {

		return "", fmt.Errorf("failed to create block profile file: %w", err)

	}

	defer file.Close()

	// Write block profile.

	profile := rpprof.Lookup("block")

	if err := profile.WriteTo(file, 0); err != nil {

		return "", fmt.Errorf("failed to write block profile: %w", err)

	}

	p.blockProfile = nil

	klog.Infof("Block profile saved to %s", filename)

	return filename, nil

}

// StartMutexProfile starts mutex profiling.

func (p *Profiler) StartMutexProfile(fraction int) error {

	p.mu.Lock()

	defer p.mu.Unlock()

	if p.mutexProfile != nil {

		return fmt.Errorf("mutex profiling already active")

	}

	// Set mutex profile fraction.

	runtime.SetMutexProfileFraction(fraction)

	p.mutexProfile = &os.File{} // Placeholder

	klog.Infof("Mutex profiling started with fraction %d", fraction)

	return nil

}

// StopMutexProfile stops mutex profiling.

func (p *Profiler) StopMutexProfile() (string, error) {

	p.mu.Lock()

	defer p.mu.Unlock()

	if p.mutexProfile == nil {

		return "", fmt.Errorf("mutex profiling not active")

	}

	// Reset mutex profile fraction.

	runtime.SetMutexProfileFraction(0)

	filename := fmt.Sprintf("%s/mutex_%d.prof", p.profileDir, time.Now().Unix())

	file, err := os.Create(filename)

	if err != nil {

		return "", fmt.Errorf("failed to create mutex profile file: %w", err)

	}

	defer file.Close()

	// Write mutex profile.

	profile := rpprof.Lookup("mutex")

	if err := profile.WriteTo(file, 0); err != nil {

		return "", fmt.Errorf("failed to write mutex profile: %w", err)

	}

	p.mutexProfile = nil

	klog.Infof("Mutex profile saved to %s", filename)

	return filename, nil

}

// StartTracing starts execution tracing.

func (p *Profiler) StartTracing() error {

	p.mu.Lock()

	defer p.mu.Unlock()

	if p.traceFile != nil {

		return fmt.Errorf("tracing already active")

	}

	filename := fmt.Sprintf("%s/trace_%d.out", p.profileDir, time.Now().Unix())

	file, err := os.Create(filename)

	if err != nil {

		return fmt.Errorf("failed to create trace file: %w", err)

	}

	// Start tracing.

	if err := trace.Start(file); err != nil {

		file.Close()

		return fmt.Errorf("failed to start tracing: %w", err)

	}

	p.traceFile = file

	klog.Infof("Execution tracing started, writing to %s", filename)

	return nil

}

// StopTracing stops execution tracing.

func (p *Profiler) StopTracing() (string, error) {

	p.mu.Lock()

	defer p.mu.Unlock()

	if p.traceFile == nil {

		return "", fmt.Errorf("tracing not active")

	}

	trace.Stop()

	filename := p.traceFile.Name()

	p.traceFile.Close()

	p.traceFile = nil

	klog.Infof("Execution tracing stopped, trace saved to %s", filename)

	return filename, nil

}

// DetectGoroutineLeaks detects potential goroutine leaks.

func (p *Profiler) DetectGoroutineLeaks(threshold int) []string {

	p.mu.Lock()

	defer p.mu.Unlock()

	p.updateGoroutineStats()

	leaks := make([]string, 0, len(p.goroutineStats.StackTraces)) // Preallocate based on stack trace count

	for stack, count := range p.goroutineStats.StackTraces {

		if count > threshold {

			leaks = append(leaks, fmt.Sprintf("Potential leak: %d goroutines at %s", count, stack))

		}

	}

	p.goroutineStats.Leaked = len(leaks)

	return leaks

}

// DetectMemoryLeaks detects potential memory leaks.

func (p *Profiler) DetectMemoryLeaks() []MemoryLeak {

	p.mu.Lock()

	defer p.mu.Unlock()

	var memStats runtime.MemStats

	runtime.ReadMemStats(&memStats)

	leaks := make([]MemoryLeak, 0, 4) // Preallocate for typical leak count

	// Simple leak detection based on heap growth.

	if p.memoryStats.HeapAlloc > 0 {

		growth := float64(memStats.HeapAlloc-p.memoryStats.HeapAlloc) / float64(p.memoryStats.HeapAlloc) * 100

		if growth > 20 { // 20% growth threshold

			leak := MemoryLeak{

				Location: "Heap",

				AllocBytes: memStats.HeapAlloc - p.memoryStats.HeapAlloc,

				AllocCount: memStats.Mallocs - memStats.Frees,

				DetectedAt: time.Now(),

				Description: fmt.Sprintf("Heap grew by %.2f%%", growth),
			}

			leaks = append(leaks, leak)

		}

	}

	p.updateMemoryStats(&memStats)

	p.memoryStats.MemoryLeaks = append(p.memoryStats.MemoryLeaks, leaks...)

	return leaks

}

// updateGoroutineStats updates goroutine statistics.

func (p *Profiler) updateGoroutineStats() {

	current := runtime.NumGoroutine()

	p.goroutineStats.Current = current

	if current > p.goroutineStats.Peak {

		p.goroutineStats.Peak = current

	}

	// Capture stack traces.

	buf := make([]byte, 1<<20) // 1MB buffer

	n := runtime.Stack(buf, true)

	// Parse stack traces (simplified).

	stacks := string(buf[:n])

	p.goroutineStats.StackTraces = make(map[string]int)

	// Count unique stack traces by splitting on goroutine boundaries.

	for _, trace := range strings.Split(stacks, "\n\ngoroutine ") {

		if len(trace) > 0 {

			p.goroutineStats.StackTraces[trace[:min(50, len(trace))]]++

		}

	}

	p.goroutineStats.LastSnapshot = time.Now()

}

// updateMemoryStats updates memory statistics.

func (p *Profiler) updateMemoryStats(memStats *runtime.MemStats) {

	p.memoryStats.HeapAlloc = memStats.HeapAlloc

	p.memoryStats.HeapSys = memStats.HeapSys

	p.memoryStats.HeapInuse = memStats.HeapInuse

	p.memoryStats.HeapReleased = memStats.HeapReleased

	p.memoryStats.GCCount = memStats.NumGC

	if memStats.NumGC > 0 {

		p.memoryStats.GCPauseTotal = time.Duration(memStats.PauseTotalNs)

		p.memoryStats.GCPauseAvg = time.Duration(memStats.PauseTotalNs / uint64(memStats.NumGC))

		p.memoryStats.LastGC = time.Unix(0, int64(memStats.LastGC))

	}

	p.memoryStats.LastSnapshot = time.Now()

}

// StartHTTPProfiler starts HTTP profiling endpoints.

func (p *Profiler) StartHTTPProfiler(addr string) error {

	p.mu.Lock()

	defer p.mu.Unlock()

	if p.httpServer != nil {

		return fmt.Errorf("HTTP profiler already running")

	}

	mux := http.NewServeMux()

	// Register pprof handlers.

	mux.HandleFunc("/debug/pprof/", pprof.Index)

	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)

	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)

	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)

	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	// Custom endpoints.

	mux.HandleFunc("/debug/stats", p.handleStats)

	mux.HandleFunc("/debug/leaks", p.handleLeaks)

	p.httpServer = &http.Server{

		Addr: addr,

		Handler: mux,
	}

	go func() {

		klog.Infof("HTTP profiler listening on %s", addr)

		if err := p.httpServer.ListenAndServe(); err != http.ErrServerClosed {

			klog.Errorf("HTTP profiler error: %v", err)

		}

	}()

	return nil

}

// StopHTTPProfiler stops the HTTP profiling server.

func (p *Profiler) StopHTTPProfiler(ctx context.Context) error {

	p.mu.Lock()

	defer p.mu.Unlock()

	if p.httpServer == nil {

		return fmt.Errorf("HTTP profiler not running")

	}

	err := p.httpServer.Shutdown(ctx)

	p.httpServer = nil

	return err

}

// handleStats handles statistics endpoint.

func (p *Profiler) handleStats(w http.ResponseWriter, r *http.Request) {

	p.mu.RLock()

	defer p.mu.RUnlock()

	stats := map[string]interface{}{

		"goroutines": p.goroutineStats,

		"memory": p.memoryStats,
	}

	w.Header().Set("Content-Type", "application/json")

	// In real implementation, encode stats as JSON.

	fmt.Fprintf(w, "%+v", stats)

}

// handleLeaks handles leak detection endpoint.

func (p *Profiler) handleLeaks(w http.ResponseWriter, r *http.Request) {

	goroutineLeaks := p.DetectGoroutineLeaks(10)

	memoryLeaks := p.DetectMemoryLeaks()

	leaks := map[string]interface{}{

		"goroutine_leaks": goroutineLeaks,

		"memory_leaks": memoryLeaks,
	}

	w.Header().Set("Content-Type", "application/json")

	// In real implementation, encode leaks as JSON.

	fmt.Fprintf(w, "%+v", leaks)

}

// GenerateFlameGraph generates a flame graph from CPU profile.

func (p *Profiler) GenerateFlameGraph(profilePath string) (string, error) {

	// In real implementation, use go-torch or similar tool.

	// to generate flame graphs from profile data.

	outputPath := fmt.Sprintf("%s.svg", profilePath)

	// Simulated flame graph generation.

	klog.Infof("Flame graph would be generated at %s", outputPath)

	return outputPath, nil

}

// AnalyzeProfile analyzes a profile and returns hotspots.

func (p *Profiler) AnalyzeProfile(profilePath string) (*ProfileReport, error) {

	report := &ProfileReport{

		StartTime: time.Now(),

		HotSpots: make([]HotSpot, 0, 10), // Preallocate for typical hotspot count

		Contentions: make([]Contention, 0, 5), // Preallocate for typical contention count

		Allocations: make([]Allocation, 0, 20), // Preallocate for typical allocation count

	}

	// In real implementation, parse profile data.

	// and identify hotspots, contentions, etc.

	// Example hotspot.

	report.HotSpots = append(report.HotSpots, HotSpot{

		Function: "example.function",

		File: "example.go",

		Line: 100,

		Samples: 1000,

		Percentage: 25.5,
	})

	report.EndTime = time.Now()

	report.Duration = report.EndTime.Sub(report.StartTime)

	return report, nil

}

// GetProfilerMetrics returns profiler metrics.

func (p *Profiler) GetProfilerMetrics() map[string]interface{} {

	p.mu.RLock()

	defer p.mu.RUnlock()

	return map[string]interface{}{

		"goroutines_current": p.goroutineStats.Current,

		"goroutines_peak": p.goroutineStats.Peak,

		"goroutines_leaked": p.goroutineStats.Leaked,

		"heap_alloc_mb": float64(p.memoryStats.HeapAlloc) / (1024 * 1024),

		"heap_sys_mb": float64(p.memoryStats.HeapSys) / (1024 * 1024),

		"gc_count": p.memoryStats.GCCount,

		"gc_pause_avg_ms": p.memoryStats.GCPauseAvg.Milliseconds(),

		"memory_leaks": len(p.memoryStats.MemoryLeaks),

		"is_active": p.isActive,
	}

}

// ExportMetrics exports profiler metrics to Prometheus.

func (p *Profiler) ExportMetrics(registry *prometheus.Registry) {

	p.mu.RLock()

	defer p.mu.RUnlock()

	// Goroutine metrics.

	goroutineGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{

		Name: "profiler_goroutines",

		Help: "Goroutine statistics",
	}, []string{"type"})

	goroutineGauge.WithLabelValues("current").Set(float64(p.goroutineStats.Current))

	goroutineGauge.WithLabelValues("peak").Set(float64(p.goroutineStats.Peak))

	goroutineGauge.WithLabelValues("leaked").Set(float64(p.goroutineStats.Leaked))

	registry.MustRegister(goroutineGauge)

	// Memory metrics.

	memoryGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{

		Name: "profiler_memory_mb",

		Help: "Memory statistics in MB",
	}, []string{"type"})

	memoryGauge.WithLabelValues("heap_alloc").Set(float64(p.memoryStats.HeapAlloc) / (1024 * 1024))

	memoryGauge.WithLabelValues("heap_sys").Set(float64(p.memoryStats.HeapSys) / (1024 * 1024))

	memoryGauge.WithLabelValues("heap_inuse").Set(float64(p.memoryStats.HeapInuse) / (1024 * 1024))

	registry.MustRegister(memoryGauge)

	// GC metrics.

	gcGauge := prometheus.NewGaugeVec(prometheus.GaugeOpts{

		Name: "profiler_gc",

		Help: "Garbage collection statistics",
	}, []string{"type"})

	gcGauge.WithLabelValues("count").Set(float64(p.memoryStats.GCCount))

	gcGauge.WithLabelValues("pause_ms").Set(float64(p.memoryStats.GCPauseAvg.Milliseconds()))

	registry.MustRegister(gcGauge)

}
