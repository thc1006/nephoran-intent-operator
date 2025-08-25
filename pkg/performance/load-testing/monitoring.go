// Package loadtesting provides monitoring and validation for load tests
package loadtesting

import (
	"context"
	"fmt"
	"math"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/v3/cpu"
	"github.com/shirou/gopsutil/v3/mem"
	"github.com/shirou/gopsutil/v3/net"
	"go.uber.org/zap"
)

// ComprehensiveMonitor provides comprehensive monitoring during load tests
type ComprehensiveMonitor struct {
	config       *LoadTestConfig
	logger       *zap.Logger
	collectors   []MetricCollector
	analyzers    []MetricAnalyzer
	alertManager *AlertManager
	metricsStore *MetricsStore
	stopChan     chan struct{}
	wg           sync.WaitGroup
	mu           sync.RWMutex
}

// MetricCollector interface for collecting different types of metrics
type MetricCollector interface {
	Collect(ctx context.Context) (map[string]float64, error)
	GetName() string
}

// MetricAnalyzer interface for analyzing collected metrics
type MetricAnalyzer interface {
	Analyze(metrics map[string]float64) AnalysisResult
	GetName() string
}

// AlertManager manages alerts during load testing
type AlertManager struct {
	logger     *zap.Logger
	thresholds AlertThresholds
	alertChan  chan Alert
	handlers   []AlertHandler
	mu         sync.RWMutex
}

// Alert represents an alert condition
type Alert struct {
	Timestamp   time.Time `json:"timestamp"`
	Level       string    `json:"level"` // info, warning, critical
	Type        string    `json:"type"`
	Metric      string    `json:"metric"`
	Value       float64   `json:"value"`
	Threshold   float64   `json:"threshold"`
	Description string    `json:"description"`
	Action      string    `json:"action"`
}

// AlertThresholds defines thresholds for alerts
type AlertThresholds struct {
	CPUPercent            float64 `json:"cpuPercent"`
	MemoryPercent         float64 `json:"memoryPercent"`
	NetworkMbps           float64 `json:"networkMbps"`
	LatencyP95Ms          float64 `json:"latencyP95Ms"`
	ErrorRatePercent      float64 `json:"errorRatePercent"`
	ThroughputDropPercent float64 `json:"throughputDropPercent"`
}

// AlertHandler interface for handling alerts
type AlertHandler interface {
	Handle(alert Alert) error
}

// MetricsStore stores historical metrics for analysis
type MetricsStore struct {
	data    []MetricSnapshot
	maxSize int
	mu      sync.RWMutex
}

// MetricSnapshot represents metrics at a point in time
type MetricSnapshot struct {
	Timestamp time.Time          `json:"timestamp"`
	Metrics   map[string]float64 `json:"metrics"`
	Labels    map[string]string  `json:"labels"`
}

// NewComprehensiveMonitor creates a new comprehensive monitor
func NewComprehensiveMonitor(config *LoadTestConfig, logger *zap.Logger) (*ComprehensiveMonitor, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required")
	}

	if logger == nil {
		logger = zap.NewNop()
	}

	monitor := &ComprehensiveMonitor{
		config:       config,
		logger:       logger,
		collectors:   make([]MetricCollector, 0),
		analyzers:    make([]MetricAnalyzer, 0),
		alertManager: newAlertManager(logger),
		metricsStore: newMetricsStore(10000),
		stopChan:     make(chan struct{}),
	}

	// Initialize collectors
	monitor.initializeCollectors()

	// Initialize analyzers
	monitor.initializeAnalyzers()

	return monitor, nil
}

// Start begins monitoring
func (m *ComprehensiveMonitor) Start(ctx context.Context) error {
	m.logger.Info("Starting comprehensive monitoring")

	m.wg.Add(1)
	go m.monitorLoop(ctx)

	// Start alert manager
	m.alertManager.Start(ctx)

	return nil
}

// Collect gathers current metrics
func (m *ComprehensiveMonitor) Collect() (MonitoringData, error) {
	data := MonitoringData{
		Timestamp:          time.Now(),
		ApplicationMetrics: make(map[string]float64),
		SystemMetrics:      make(map[string]float64),
		NetworkMetrics:     make(map[string]float64),
		CustomMetrics:      make(map[string]interface{}),
	}

	// Collect from all collectors
	for _, collector := range m.collectors {
		metrics, err := collector.Collect(context.Background())
		if err != nil {
			m.logger.Error("Failed to collect metrics",
				zap.String("collector", collector.GetName()),
				zap.Error(err))
			continue
		}

		// Categorize metrics
		for key, value := range metrics {
			switch collector.GetName() {
			case "system":
				data.SystemMetrics[key] = value
			case "network":
				data.NetworkMetrics[key] = value
			default:
				data.ApplicationMetrics[key] = value
			}
		}
	}

	// Store snapshot
	m.metricsStore.Add(MetricSnapshot{
		Timestamp: data.Timestamp,
		Metrics:   mergeMetrics(data.ApplicationMetrics, data.SystemMetrics, data.NetworkMetrics),
	})

	return data, nil
}

// Analyze performs real-time analysis
func (m *ComprehensiveMonitor) Analyze(data MonitoringData) AnalysisResult {
	result := AnalysisResult{
		Timestamp:       data.Timestamp,
		Anomalies:       make([]Anomaly, 0),
		Trends:          make(map[string]Trend),
		Predictions:     make(map[string]Prediction),
		Recommendations: make([]string, 0),
	}

	// Merge all metrics for analysis
	allMetrics := mergeMetrics(data.ApplicationMetrics, data.SystemMetrics, data.NetworkMetrics)

	// Run analyzers
	for _, analyzer := range m.analyzers {
		analysisResult := analyzer.Analyze(allMetrics)

		// Merge results
		result.Anomalies = append(result.Anomalies, analysisResult.Anomalies...)
		for k, v := range analysisResult.Trends {
			result.Trends[k] = v
		}
		for k, v := range analysisResult.Predictions {
			result.Predictions[k] = v
		}
		result.Recommendations = append(result.Recommendations, analysisResult.Recommendations...)
	}

	// Deduplicate recommendations
	result.Recommendations = deduplicateStrings(result.Recommendations)

	return result
}

// Alert triggers alerts for anomalies
func (m *ComprehensiveMonitor) Alert(result AnalysisResult) error {
	for _, anomaly := range result.Anomalies {
		alert := Alert{
			Timestamp:   result.Timestamp,
			Level:       anomaly.Severity,
			Type:        anomaly.Type,
			Metric:      anomaly.Metric,
			Value:       anomaly.Value,
			Threshold:   anomaly.Expected,
			Description: anomaly.Description,
			Action:      m.determineAction(anomaly),
		}

		if err := m.alertManager.Send(alert); err != nil {
			m.logger.Error("Failed to send alert", zap.Error(err))
		}
	}

	return nil
}

// Stop stops monitoring
func (m *ComprehensiveMonitor) Stop() error {
	m.logger.Info("Stopping monitoring")
	close(m.stopChan)
	m.wg.Wait()
	return m.alertManager.Stop()
}

// Private methods

func (m *ComprehensiveMonitor) initializeCollectors() {
	// System metrics collector
	m.collectors = append(m.collectors, NewSystemMetricsCollector(m.logger))

	// Network metrics collector
	m.collectors = append(m.collectors, NewNetworkMetricsCollector(m.logger))

	// Application metrics collector
	m.collectors = append(m.collectors, NewApplicationMetricsCollector(m.config, m.logger))

	// Kubernetes metrics collector (if in K8s environment)
	if isKubernetesEnvironment() {
		m.collectors = append(m.collectors, NewKubernetesMetricsCollector(m.logger))
	}
}

func (m *ComprehensiveMonitor) initializeAnalyzers() {
	// Anomaly detection analyzer
	m.analyzers = append(m.analyzers, NewAnomalyDetector(m.metricsStore, m.logger))

	// Trend analyzer
	m.analyzers = append(m.analyzers, NewTrendAnalyzer(m.metricsStore, m.logger))

	// Predictive analyzer
	m.analyzers = append(m.analyzers, NewPredictiveAnalyzer(m.metricsStore, m.logger))

	// Bottleneck analyzer
	m.analyzers = append(m.analyzers, NewBottleneckAnalyzer(m.logger))
}

func (m *ComprehensiveMonitor) monitorLoop(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(m.config.MetricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopChan:
			return
		case <-ticker.C:
			// Collect metrics
			data, err := m.Collect()
			if err != nil {
				m.logger.Error("Failed to collect metrics", zap.Error(err))
				continue
			}

			// Analyze metrics
			result := m.Analyze(data)

			// Send alerts if needed
			if len(result.Anomalies) > 0 {
				if err := m.Alert(result); err != nil {
					m.logger.Error("Failed to send alerts", zap.Error(err))
				}
			}
		}
	}
}

func (m *ComprehensiveMonitor) determineAction(anomaly Anomaly) string {
	switch anomaly.Type {
	case "cpu_high":
		return "Consider scaling horizontally or optimizing CPU-intensive operations"
	case "memory_high":
		return "Check for memory leaks or increase memory allocation"
	case "latency_spike":
		return "Investigate slow operations or network issues"
	case "error_rate_high":
		return "Check application logs for errors and failures"
	case "throughput_drop":
		return "Verify system capacity and check for bottlenecks"
	default:
		return "Monitor situation and investigate if condition persists"
	}
}

// SystemMetricsCollector collects system-level metrics
type SystemMetricsCollector struct {
	logger *zap.Logger
}

func NewSystemMetricsCollector(logger *zap.Logger) *SystemMetricsCollector {
	return &SystemMetricsCollector{logger: logger}
}

func (c *SystemMetricsCollector) Collect(ctx context.Context) (map[string]float64, error) {
	metrics := make(map[string]float64)

	// CPU metrics
	cpuPercent, err := cpu.Percent(time.Second, false)
	if err == nil && len(cpuPercent) > 0 {
		metrics["cpu_percent"] = cpuPercent[0]
	}

	// Memory metrics
	memStats, err := mem.VirtualMemory()
	if err == nil {
		metrics["memory_percent"] = memStats.UsedPercent
		metrics["memory_used_gb"] = float64(memStats.Used) / (1024 * 1024 * 1024)
		metrics["memory_available_gb"] = float64(memStats.Available) / (1024 * 1024 * 1024)
	}

	// Goroutine metrics
	metrics["goroutines"] = float64(runtime.NumGoroutine())

	// GC metrics
	var gcStats runtime.MemStats
	runtime.ReadMemStats(&gcStats)
	metrics["gc_pause_ms"] = float64(gcStats.PauseNs[(gcStats.NumGC+255)%256]) / 1e6
	metrics["gc_runs"] = float64(gcStats.NumGC)

	return metrics, nil
}

func (c *SystemMetricsCollector) GetName() string {
	return "system"
}

// NetworkMetricsCollector collects network-level metrics
type NetworkMetricsCollector struct {
	logger      *zap.Logger
	lastStats   *net.IOCountersStat
	lastCollect time.Time
	mu          sync.Mutex
}

func NewNetworkMetricsCollector(logger *zap.Logger) *NetworkMetricsCollector {
	return &NetworkMetricsCollector{
		logger: logger,
	}
}

func (c *NetworkMetricsCollector) Collect(ctx context.Context) (map[string]float64, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	metrics := make(map[string]float64)

	// Get network I/O stats
	stats, err := net.IOCounters(false)
	if err != nil || len(stats) == 0 {
		return metrics, err
	}

	currentStats := &stats[0]
	now := time.Now()

	// Calculate rates if we have previous stats
	if c.lastStats != nil && !c.lastCollect.IsZero() {
		duration := now.Sub(c.lastCollect).Seconds()
		if duration > 0 {
			// Calculate bandwidth in Mbps
			metrics["network_rx_mbps"] = float64(currentStats.BytesRecv-c.lastStats.BytesRecv) * 8 / (duration * 1e6)
			metrics["network_tx_mbps"] = float64(currentStats.BytesSent-c.lastStats.BytesSent) * 8 / (duration * 1e6)

			// Packet rates
			metrics["network_rx_pps"] = float64(currentStats.PacketsRecv-c.lastStats.PacketsRecv) / duration
			metrics["network_tx_pps"] = float64(currentStats.PacketsSent-c.lastStats.PacketsSent) / duration

			// Error rates
			metrics["network_rx_errors"] = float64(currentStats.Errin-c.lastStats.Errin) / duration
			metrics["network_tx_errors"] = float64(currentStats.Errout-c.lastStats.Errout) / duration

			// Drop rates
			metrics["network_rx_drops"] = float64(currentStats.Dropin-c.lastStats.Dropin) / duration
			metrics["network_tx_drops"] = float64(currentStats.Dropout-c.lastStats.Dropout) / duration
		}
	}

	// Update last stats
	c.lastStats = currentStats
	c.lastCollect = now

	return metrics, nil
}

func (c *NetworkMetricsCollector) GetName() string {
	return "network"
}

// ApplicationMetricsCollector collects application-specific metrics
type ApplicationMetricsCollector struct {
	config   *LoadTestConfig
	logger   *zap.Logger
	registry *prometheus.Registry
}

func NewApplicationMetricsCollector(config *LoadTestConfig, logger *zap.Logger) *ApplicationMetricsCollector {
	return &ApplicationMetricsCollector{
		config:   config,
		logger:   logger,
		registry: prometheus.DefaultRegisterer.(*prometheus.Registry),
	}
}

func (c *ApplicationMetricsCollector) Collect(ctx context.Context) (map[string]float64, error) {
	metrics := make(map[string]float64)

	// Collect from Prometheus metrics
	mfs, err := c.registry.Gather()
	if err != nil {
		return metrics, err
	}

	for _, mf := range mfs {
		// Extract relevant metrics
		switch mf.GetName() {
		case "loadtest_request_latency_seconds":
			if len(mf.GetMetric()) > 0 {
				// Get histogram quantiles
				for _, m := range mf.GetMetric() {
					if h := m.GetHistogram(); h != nil {
						metrics["latency_count"] = float64(h.GetSampleCount())
						metrics["latency_sum"] = h.GetSampleSum()
					}
				}
			}
		case "loadtest_throughput_rpm":
			if len(mf.GetMetric()) > 0 {
				for _, m := range mf.GetMetric() {
					if g := m.GetGauge(); g != nil {
						metrics["throughput_rpm"] = g.GetValue()
					}
				}
			}
		}
	}

	return metrics, nil
}

func (c *ApplicationMetricsCollector) GetName() string {
	return "application"
}

// KubernetesMetricsCollector collects Kubernetes-specific metrics
type KubernetesMetricsCollector struct {
	logger *zap.Logger
}

func NewKubernetesMetricsCollector(logger *zap.Logger) *KubernetesMetricsCollector {
	return &KubernetesMetricsCollector{logger: logger}
}

func (c *KubernetesMetricsCollector) Collect(ctx context.Context) (map[string]float64, error) {
	metrics := make(map[string]float64)

	// This would connect to Kubernetes metrics API
	// For now, return placeholder metrics

	return metrics, nil
}

func (c *KubernetesMetricsCollector) GetName() string {
	return "kubernetes"
}

// AnomalyDetector detects anomalies in metrics
type AnomalyDetector struct {
	store  *MetricsStore
	logger *zap.Logger
}

func NewAnomalyDetector(store *MetricsStore, logger *zap.Logger) *AnomalyDetector {
	return &AnomalyDetector{
		store:  store,
		logger: logger,
	}
}

func (d *AnomalyDetector) Analyze(metrics map[string]float64) AnalysisResult {
	result := AnalysisResult{
		Timestamp: time.Now(),
		Anomalies: make([]Anomaly, 0),
	}

	// Get historical data for comparison
	historical := d.store.GetRecent(100)
	if len(historical) < 10 {
		return result // Not enough data for anomaly detection
	}

	// Calculate statistics for each metric
	for metric, value := range metrics {
		stats := d.calculateStats(metric, historical)

		// Z-score anomaly detection
		if stats.stdDev > 0 {
			zScore := math.Abs((value - stats.mean) / stats.stdDev)

			if zScore > 3 {
				severity := "warning"
				if zScore > 4 {
					severity = "critical"
				}

				result.Anomalies = append(result.Anomalies, Anomaly{
					Type:        d.categorizeMetric(metric),
					Severity:    severity,
					Description: fmt.Sprintf("%s is %.2f standard deviations from mean", metric, zScore),
					Metric:      metric,
					Value:       value,
					Expected:    stats.mean,
					Deviation:   zScore,
					Timestamp:   time.Now(),
				})
			}
		}

		// Check for sudden changes
		if len(historical) > 0 {
			recent := historical[len(historical)-1].Metrics[metric]
			changePercent := math.Abs((value - recent) / recent * 100)

			if changePercent > 50 {
				result.Anomalies = append(result.Anomalies, Anomaly{
					Type:        "sudden_change",
					Severity:    "warning",
					Description: fmt.Sprintf("%s changed by %.1f%%", metric, changePercent),
					Metric:      metric,
					Value:       value,
					Expected:    recent,
					Deviation:   changePercent,
					Timestamp:   time.Now(),
				})
			}
		}
	}

	return result
}

func (d *AnomalyDetector) GetName() string {
	return "anomaly_detector"
}

func (d *AnomalyDetector) calculateStats(metric string, historical []MetricSnapshot) struct {
	mean   float64
	stdDev float64
} {
	if len(historical) == 0 {
		return struct {
			mean   float64
			stdDev float64
		}{}
	}

	// Calculate mean
	var sum float64
	var count int
	for _, snapshot := range historical {
		if val, ok := snapshot.Metrics[metric]; ok {
			sum += val
			count++
		}
	}

	if count == 0 {
		return struct {
			mean   float64
			stdDev float64
		}{}
	}

	mean := sum / float64(count)

	// Calculate standard deviation
	var varianceSum float64
	for _, snapshot := range historical {
		if val, ok := snapshot.Metrics[metric]; ok {
			diff := val - mean
			varianceSum += diff * diff
		}
	}

	variance := varianceSum / float64(count)
	stdDev := math.Sqrt(variance)

	return struct {
		mean   float64
		stdDev float64
	}{mean: mean, stdDev: stdDev}
}

func (d *AnomalyDetector) categorizeMetric(metric string) string {
	switch {
	case contains(metric, "cpu"):
		return "cpu_anomaly"
	case contains(metric, "memory"):
		return "memory_anomaly"
	case contains(metric, "network"):
		return "network_anomaly"
	case contains(metric, "latency"):
		return "latency_anomaly"
	case contains(metric, "error"):
		return "error_anomaly"
	case contains(metric, "throughput"):
		return "throughput_anomaly"
	default:
		return "metric_anomaly"
	}
}

// TrendAnalyzer analyzes trends in metrics
type TrendAnalyzer struct {
	store  *MetricsStore
	logger *zap.Logger
}

func NewTrendAnalyzer(store *MetricsStore, logger *zap.Logger) *TrendAnalyzer {
	return &TrendAnalyzer{
		store:  store,
		logger: logger,
	}
}

func (a *TrendAnalyzer) Analyze(metrics map[string]float64) AnalysisResult {
	result := AnalysisResult{
		Timestamp: time.Now(),
		Trends:    make(map[string]Trend),
	}

	// Get historical data for trend analysis
	historical := a.store.GetRecent(50)
	if len(historical) < 5 {
		return result // Not enough data for trend analysis
	}

	for metric := range metrics {
		// Extract time series for this metric
		timeSeries := make([]float64, 0, len(historical))
		for _, snapshot := range historical {
			if val, ok := snapshot.Metrics[metric]; ok {
				timeSeries = append(timeSeries, val)
			}
		}

		if len(timeSeries) < 5 {
			continue
		}

		// Calculate trend using linear regression
		slope, intercept := linearRegression(timeSeries)

		// Determine trend direction
		direction := "stable"
		if math.Abs(slope) > 0.01 {
			if slope > 0 {
				direction = "increasing"
			} else {
				direction = "decreasing"
			}
		}

		// Calculate R-squared for confidence
		rSquared := calculateRSquared(timeSeries, slope, intercept)

		// Predict next value
		nextValue := slope*float64(len(timeSeries)) + intercept

		result.Trends[metric] = Trend{
			Direction:  direction,
			Rate:       slope,
			Confidence: rSquared,
			Prediction: nextValue,
		}
	}

	return result
}

func (a *TrendAnalyzer) GetName() string {
	return "trend_analyzer"
}

// PredictiveAnalyzer performs predictive analysis
type PredictiveAnalyzer struct {
	store  *MetricsStore
	logger *zap.Logger
}

func NewPredictiveAnalyzer(store *MetricsStore, logger *zap.Logger) *PredictiveAnalyzer {
	return &PredictiveAnalyzer{
		store:  store,
		logger: logger,
	}
}

func (a *PredictiveAnalyzer) Analyze(metrics map[string]float64) AnalysisResult {
	result := AnalysisResult{
		Timestamp:       time.Now(),
		Predictions:     make(map[string]Prediction),
		Recommendations: make([]string, 0),
	}

	// Predict resource exhaustion
	if cpuPercent, ok := metrics["cpu_percent"]; ok {
		if cpuPercent > 80 {
			timeToExhaustion := a.predictTimeToExhaustion("cpu_percent", 95)
			result.Predictions["cpu_exhaustion"] = Prediction{
				Metric:      "cpu_percent",
				Value:       95,
				Confidence:  0.8,
				TimeHorizon: timeToExhaustion,
			}

			if timeToExhaustion < 10*time.Minute {
				result.Recommendations = append(result.Recommendations,
					"CPU exhaustion predicted within 10 minutes - consider scaling immediately")
			}
		}
	}

	if memPercent, ok := metrics["memory_percent"]; ok {
		if memPercent > 75 {
			timeToExhaustion := a.predictTimeToExhaustion("memory_percent", 95)
			result.Predictions["memory_exhaustion"] = Prediction{
				Metric:      "memory_percent",
				Value:       95,
				Confidence:  0.8,
				TimeHorizon: timeToExhaustion,
			}

			if timeToExhaustion < 10*time.Minute {
				result.Recommendations = append(result.Recommendations,
					"Memory exhaustion predicted within 10 minutes - investigate memory usage")
			}
		}
	}

	return result
}

func (a *PredictiveAnalyzer) GetName() string {
	return "predictive_analyzer"
}

func (a *PredictiveAnalyzer) predictTimeToExhaustion(metric string, threshold float64) time.Duration {
	historical := a.store.GetRecent(30)
	if len(historical) < 10 {
		return time.Hour // Default to 1 hour if insufficient data
	}

	// Extract time series
	timeSeries := make([]float64, 0, len(historical))
	for _, snapshot := range historical {
		if val, ok := snapshot.Metrics[metric]; ok {
			timeSeries = append(timeSeries, val)
		}
	}

	// Calculate rate of change
	slope, _ := linearRegression(timeSeries)

	if slope <= 0 {
		return time.Hour * 24 // No exhaustion predicted if decreasing
	}

	// Calculate time to reach threshold
	currentValue := timeSeries[len(timeSeries)-1]
	stepsToThreshold := (threshold - currentValue) / slope

	// Convert steps to duration (assuming metrics interval)
	return time.Duration(stepsToThreshold) * 10 * time.Second
}

// BottleneckAnalyzer identifies system bottlenecks
type BottleneckAnalyzer struct {
	logger *zap.Logger
}

func NewBottleneckAnalyzer(logger *zap.Logger) *BottleneckAnalyzer {
	return &BottleneckAnalyzer{logger: logger}
}

func (a *BottleneckAnalyzer) Analyze(metrics map[string]float64) AnalysisResult {
	result := AnalysisResult{
		Timestamp:       time.Now(),
		Recommendations: make([]string, 0),
	}

	// Check for CPU bottleneck
	if cpu, ok := metrics["cpu_percent"]; ok && cpu > 90 {
		result.Recommendations = append(result.Recommendations,
			"CPU bottleneck detected - consider optimizing CPU-intensive operations or scaling")
	}

	// Check for memory bottleneck
	if mem, ok := metrics["memory_percent"]; ok && mem > 85 {
		result.Recommendations = append(result.Recommendations,
			"Memory bottleneck detected - check for memory leaks or increase allocation")
	}

	// Check for network bottleneck
	if rxMbps, ok := metrics["network_rx_mbps"]; ok {
		if txMbps, ok := metrics["network_tx_mbps"]; ok {
			totalMbps := rxMbps + txMbps
			if totalMbps > 800 { // Assuming 1Gbps link
				result.Recommendations = append(result.Recommendations,
					"Network bottleneck detected - consider network optimization or scaling")
			}
		}
	}

	// Check for high error rates
	if errors, ok := metrics["network_rx_errors"]; ok && errors > 100 {
		result.Recommendations = append(result.Recommendations,
			"High network error rate detected - investigate network issues")
	}

	return result
}

func (a *BottleneckAnalyzer) GetName() string {
	return "bottleneck_analyzer"
}

// AlertManager implementation

func newAlertManager(logger *zap.Logger) *AlertManager {
	return &AlertManager{
		logger:    logger,
		alertChan: make(chan Alert, 100),
		handlers:  make([]AlertHandler, 0),
		thresholds: AlertThresholds{
			CPUPercent:            85,
			MemoryPercent:         80,
			NetworkMbps:           800,
			LatencyP95Ms:          2000,
			ErrorRatePercent:      5,
			ThroughputDropPercent: 20,
		},
	}
}

func (am *AlertManager) Start(ctx context.Context) {
	go am.processAlerts(ctx)
}

func (am *AlertManager) Send(alert Alert) error {
	select {
	case am.alertChan <- alert:
		return nil
	default:
		return fmt.Errorf("alert channel full")
	}
}

func (am *AlertManager) Stop() error {
	close(am.alertChan)
	return nil
}

func (am *AlertManager) processAlerts(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case alert, ok := <-am.alertChan:
			if !ok {
				return
			}

			am.logger.Warn("Alert triggered",
				zap.String("level", alert.Level),
				zap.String("type", alert.Type),
				zap.String("metric", alert.Metric),
				zap.Float64("value", alert.Value),
				zap.String("description", alert.Description))

			// Process through handlers
			for _, handler := range am.handlers {
				if err := handler.Handle(alert); err != nil {
					am.logger.Error("Failed to handle alert", zap.Error(err))
				}
			}
		}
	}
}

// MetricsStore implementation

func newMetricsStore(maxSize int) *MetricsStore {
	return &MetricsStore{
		data:    make([]MetricSnapshot, 0, maxSize),
		maxSize: maxSize,
	}
}

func (ms *MetricsStore) Add(snapshot MetricSnapshot) {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	ms.data = append(ms.data, snapshot)

	// Trim if exceeds max size
	if len(ms.data) > ms.maxSize {
		ms.data = ms.data[len(ms.data)-ms.maxSize:]
	}
}

func (ms *MetricsStore) GetRecent(count int) []MetricSnapshot {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	if count > len(ms.data) {
		count = len(ms.data)
	}

	if count == 0 {
		return []MetricSnapshot{}
	}

	return ms.data[len(ms.data)-count:]
}

// Helper functions

func mergeMetrics(maps ...map[string]float64) map[string]float64 {
	result := make(map[string]float64)
	for _, m := range maps {
		for k, v := range m {
			result[k] = v
		}
	}
	return result
}

func deduplicateStrings(strings []string) []string {
	seen := make(map[string]bool)
	result := make([]string, 0)

	for _, s := range strings {
		if !seen[s] {
			seen[s] = true
			result = append(result, s)
		}
	}

	return result
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[:len(substr)] == substr
}

func linearRegression(data []float64) (slope, intercept float64) {
	n := float64(len(data))
	if n == 0 {
		return 0, 0
	}

	var sumX, sumY, sumXY, sumX2 float64
	for i, y := range data {
		x := float64(i)
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}

	denominator := n*sumX2 - sumX*sumX
	if denominator == 0 {
		return 0, sumY / n
	}

	slope = (n*sumXY - sumX*sumY) / denominator
	intercept = (sumY - slope*sumX) / n

	return slope, intercept
}

func calculateRSquared(data []float64, slope, intercept float64) float64 {
	if len(data) == 0 {
		return 0
	}

	// Calculate mean
	var sum float64
	for _, y := range data {
		sum += y
	}
	mean := sum / float64(len(data))

	// Calculate total sum of squares
	var totalSS float64
	for _, y := range data {
		diff := y - mean
		totalSS += diff * diff
	}

	// Calculate residual sum of squares
	var residualSS float64
	for i, y := range data {
		predicted := slope*float64(i) + intercept
		diff := y - predicted
		residualSS += diff * diff
	}

	if totalSS == 0 {
		return 0
	}

	return 1 - (residualSS / totalSS)
}

func isKubernetesEnvironment() bool {
	// Check if running in Kubernetes by looking for service account
	_, err := os.Stat("/var/run/secrets/kubernetes.io/serviceaccount")
	return err == nil
}
