// Package performance provides performance optimization configurations and utilities
package performance

import (
	"context"
	"fmt"
	"net/http"
	_ "net/http/pprof" // Import pprof for profiling endpoints
	"runtime"
	"runtime/debug"
	"time"

	"github.com/bytedance/sonic"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

// Config holds performance optimization settings
type Config struct {
	// JSON processing configuration
	UseSONIC bool `json:"use_sonic" yaml:"use_sonic"`

	// HTTP2 configuration
	EnableHTTP2          bool   `json:"enable_http2" yaml:"enable_http2"`
	HTTP2MaxConcurrent   uint32 `json:"http2_max_concurrent" yaml:"http2_max_concurrent"`
	HTTP2MaxUploadBuffer uint32 `json:"http2_max_upload_buffer" yaml:"http2_max_upload_buffer"`

	// Profiling configuration
	EnableProfiling bool   `json:"enable_profiling" yaml:"enable_profiling"`
	ProfilingAddr   string `json:"profiling_addr" yaml:"profiling_addr"`

	// Metrics configuration
	EnableMetrics bool   `json:"enable_metrics" yaml:"enable_metrics"`
	MetricsAddr   string `json:"metrics_addr" yaml:"metrics_addr"`

	// Runtime optimization
	GOMAXPROCS int `json:"gomaxprocs" yaml:"gomaxprocs"`
	GOGC       int `json:"gogc" yaml:"gogc"`

	// Connection pool settings
	MaxIdleConns          int           `json:"max_idle_conns" yaml:"max_idle_conns"`
	MaxConnsPerHost       int           `json:"max_conns_per_host" yaml:"max_conns_per_host"`
	IdleConnTimeout       time.Duration `json:"idle_conn_timeout" yaml:"idle_conn_timeout"`
	DisableKeepAlives     bool          `json:"disable_keep_alives" yaml:"disable_keep_alives"`
	DisableCompression    bool          `json:"disable_compression" yaml:"disable_compression"`
	ResponseHeaderTimeout time.Duration `json:"response_header_timeout" yaml:"response_header_timeout"`
}

// DefaultConfig returns optimized default performance configuration
func DefaultConfig() *Config {
	return &Config{
		UseSONIC:              true,
		EnableHTTP2:           true,
		HTTP2MaxConcurrent:    1000,
		HTTP2MaxUploadBuffer:  1 << 20, // 1MB
		EnableProfiling:       true,
		ProfilingAddr:         ":6060",
		EnableMetrics:         true,
		MetricsAddr:           ":9090",
		GOMAXPROCS:            runtime.NumCPU(),
		GOGC:                  100, // Default GC percentage
		MaxIdleConns:          100,
		MaxConnsPerHost:       10,
		IdleConnTimeout:       90 * time.Second,
		DisableKeepAlives:     false,
		DisableCompression:    false,
		ResponseHeaderTimeout: 10 * time.Second,
	}
}

// JSONMarshal uses bytedance/sonic for high-performance JSON marshaling
func JSONMarshal(v interface{}) ([]byte, error) {
	return sonic.Marshal(v)
}

// JSONUnmarshal uses bytedance/sonic for high-performance JSON unmarshaling
func JSONUnmarshal(data []byte, v interface{}) error {
	return sonic.Unmarshal(data, v)
}

// SetupHTTP2Server configures an HTTP server with HTTP/2 support
func SetupHTTP2Server(handler http.Handler, config *Config) *http.Server {
	h2s := &http2.Server{
		MaxConcurrentStreams:         config.HTTP2MaxConcurrent,
		MaxUploadBufferPerStream:     int32(config.HTTP2MaxUploadBuffer),
		MaxUploadBufferPerConnection: int32(config.HTTP2MaxUploadBuffer * 10),
	}

	server := &http.Server{
		Handler:      h2c.NewHandler(handler, h2s),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	return server
}

// SetupOptimizedTransport creates an optimized HTTP transport with HTTP/2 support
func SetupOptimizedTransport(config *Config) *http.Transport {
	transport := &http.Transport{
		MaxIdleConns:          config.MaxIdleConns,
		MaxIdleConnsPerHost:   config.MaxConnsPerHost,
		MaxConnsPerHost:       config.MaxConnsPerHost,
		IdleConnTimeout:       config.IdleConnTimeout,
		DisableKeepAlives:     config.DisableKeepAlives,
		DisableCompression:    config.DisableCompression,
		ResponseHeaderTimeout: config.ResponseHeaderTimeout,
		ForceAttemptHTTP2:     config.EnableHTTP2,
	}

	if config.EnableHTTP2 {
		if err := http2.ConfigureTransport(transport); err != nil {
			// Log error but continue with HTTP/1.1
			fmt.Printf("Failed to configure HTTP/2 transport: %v\n", err)
		}
	}

	return transport
}

// StartProfilingServer starts the pprof profiling server
func StartProfilingServer(config *Config, logger *zap.Logger) error {
	if !config.EnableProfiling {
		return nil
	}

	go func() {
		logger.Info("Starting profiling server", zap.String("addr", config.ProfilingAddr))
		if err := http.ListenAndServe(config.ProfilingAddr, nil); err != nil {
			logger.Error("Profiling server failed", zap.Error(err))
		}
	}()

	return nil
}

// StartMetricsServer starts the Prometheus metrics server
func StartMetricsServer(config *Config, registry *prometheus.Registry, logger *zap.Logger) error {
	if !config.EnableMetrics {
		return nil
	}

	go func() {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{
			EnableOpenMetrics: true,
		}))

		server := &http.Server{
			Addr:         config.MetricsAddr,
			Handler:      mux,
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		}

		logger.Info("Starting metrics server", zap.String("addr", config.MetricsAddr))
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Metrics server failed", zap.Error(err))
		}
	}()

	return nil
}

// ApplyRuntimeOptimizations applies runtime performance optimizations
func ApplyRuntimeOptimizations(config *Config, logger *zap.Logger) {
	// Set GOMAXPROCS
	if config.GOMAXPROCS > 0 {
		runtime.GOMAXPROCS(config.GOMAXPROCS)
		logger.Info("Set GOMAXPROCS", zap.Int("value", config.GOMAXPROCS))
	}

	// Set GC percentage
	if config.GOGC > 0 {
		debug.SetGCPercent(config.GOGC)
		logger.Info("Set GOGC", zap.Int("value", config.GOGC))
	}
}

// Manager handles performance monitoring and optimization
type Manager struct {
	config  *Config
	logger  *zap.Logger
	metrics *Metrics
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewManager creates a new performance manager
func NewManager(config *Config, logger *zap.Logger, registry *prometheus.Registry) *Manager {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &Manager{
		config:  config,
		logger:  logger,
		metrics: NewMetrics(registry),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Apply runtime optimizations
	ApplyRuntimeOptimizations(config, logger)

	// Start profiling server
	if err := StartProfilingServer(config, logger); err != nil {
		logger.Error("Failed to start profiling server", zap.Error(err))
	}

	// Start metrics server
	if err := StartMetricsServer(config, registry, logger); err != nil {
		logger.Error("Failed to start metrics server", zap.Error(err))
	}

	return manager
}

// Stop gracefully stops the performance manager
func (m *Manager) Stop() {
	m.cancel()
}

// GetConfig returns the current performance configuration
func (m *Manager) GetConfig() *Config {
	return m.config
}
