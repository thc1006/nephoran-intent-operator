// Package monitoring - Metrics collection implementation
package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/common/model"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// safeRegister safely registers a metric collector, avoiding panics from duplicate registrations
func safeRegister(registry prometheus.Registerer, collector prometheus.Collector) {
	if err := registry.Register(collector); err != nil {
		if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {
			// Only log if it's not a duplicate registration error
			log.Log.Error(err, "Failed to register metric collector")
		}
		// Ignore duplicate registration errors to prevent panics in tests
	}
}

// PrometheusMetricsCollector implements MetricsCollector using Prometheus
type PrometheusMetricsCollector struct {
	client     api.Client
	api        v1.API
	registry   prometheus.Registerer
	k8sClient  kubernetes.Interface
	collectors map[string]*ComponentCollector
	mu         sync.RWMutex

	// Custom metrics
	scrapeErrors     prometheus.Counter
	scrapeDuration   prometheus.Histogram
	metricsCollected prometheus.Counter
}

// ComponentCollector collects metrics for a specific component
type ComponentCollector struct {
	name       string
	namespace  string
	selector   map[string]string
	queries    []MetricQuery
	lastScrape time.Time
	registry   prometheus.Registerer
}

// MetricQuery defines a Prometheus query for collecting metrics
type MetricQuery struct {
	Name  string `json:"name"`
	Query string `json:"query"`
	Help  string `json:"help,omitempty"`
}

// NewPrometheusMetricsCollector creates a new Prometheus-based metrics collector
func NewPrometheusMetricsCollector(prometheusURL string, k8sClient kubernetes.Interface, registry prometheus.Registerer) (*PrometheusMetricsCollector, error) {
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	pmc := &PrometheusMetricsCollector{
		client:     client,
		api:        v1.NewAPI(client),
		registry:   registry,
		k8sClient:  k8sClient,
		collectors: make(map[string]*ComponentCollector),
	}

	pmc.initMetrics()
	return pmc, nil
}

// initMetrics initializes Prometheus metrics for the collector
func (pmc *PrometheusMetricsCollector) initMetrics() {
	pmc.scrapeErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "oran_metrics_scrape_errors_total",
		Help: "Total number of metrics scrape errors",
	})

	pmc.scrapeDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "oran_metrics_scrape_duration_seconds",
		Help:    "Duration of metrics scrapes in seconds",
		Buckets: prometheus.DefBuckets,
	})

	pmc.metricsCollected = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "oran_metrics_collected_total",
		Help: "Total number of metrics collected",
	})

	if pmc.registry != nil {
		safeRegister(pmc.registry, pmc.scrapeErrors)
		safeRegister(pmc.registry, pmc.scrapeDuration) 
		safeRegister(pmc.registry, pmc.metricsCollected)
	}
}

// RegisterMetrics registers the collector's metrics with Prometheus
func (pmc *PrometheusMetricsCollector) RegisterMetrics(registry prometheus.Registerer) error {
	if registry == nil {
		return fmt.Errorf("registry cannot be nil")
	}

	pmc.registry = registry
	pmc.initMetrics()

	return nil
}

// Start begins metrics collection - FIXED SIGNATURE
func (pmc *PrometheusMetricsCollector) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting Prometheus metrics collector")

	// Start collection loop for registered collectors
	go pmc.collectionLoop(ctx)

	return nil
}

// Stop halts metrics collection
func (pmc *PrometheusMetricsCollector) Stop() error {
	// Collection loop will be stopped by context cancellation
	return nil
}

// CollectMetrics gathers metrics from the specified source - FIXED SIGNATURE
func (pmc *PrometheusMetricsCollector) CollectMetrics(ctx context.Context, source string) (*MetricsData, error) {
	startTime := time.Now()
	defer func() {
		pmc.scrapeDuration.Observe(time.Since(startTime).Seconds())
	}()

	pmc.mu.RLock()
	collector, exists := pmc.collectors[source]
	pmc.mu.RUnlock()

	if !exists {
		pmc.scrapeErrors.Inc()
		return nil, fmt.Errorf("collector not found for source: %s", source)
	}

	logger := log.FromContext(ctx)
	logger.V(1).Info("Collecting metrics", "source", source)

	metricsData := &MetricsData{
		Timestamp: time.Now(),
		Source:    source,
		Namespace: collector.namespace,
		Metrics:   make(map[string]string),
		Labels:    make(map[string]string),
		Metadata:  json.RawMessage(`{}`),
	}

	// Execute queries
	for _, query := range collector.queries {
		result, warnings, err := pmc.api.Query(ctx, query.Query, time.Now())
		if err != nil {
			pmc.scrapeErrors.Inc()
			logger.Error(err, "Failed to execute query", "query", query.Query)
			continue
		}

		if len(warnings) > 0 {
			logger.V(1).Info("Query warnings", "warnings", warnings)
		}

		// Process query result
		if err := pmc.processQueryResult(query.Name, result, metricsData); err != nil {
			logger.Error(err, "Failed to process query result", "query", query.Name)
			continue
		}

		pmc.metricsCollected.Inc()
	}

	// Update last scrape time
	pmc.mu.Lock()
	collector.lastScrape = time.Now()
	pmc.mu.Unlock()

	return metricsData, nil
}

// Removed duplicate CollectMetrics method - use the one with context

// processQueryResult processes a Prometheus query result
func (pmc *PrometheusMetricsCollector) processQueryResult(name string, result model.Value, data *MetricsData) error {
	switch v := result.(type) {
	case model.Vector:
		// Handle vector results (instant vector)
		for _, sample := range v {
			metricName := fmt.Sprintf("%s_%s", name, string(sample.Metric[model.MetricNameLabel]))
			data.Metrics[metricName] = fmt.Sprintf("%.6f", float64(sample.Value))

			// Add labels from the metric
			for labelName, labelValue := range sample.Metric {
				if labelName != model.MetricNameLabel {
					data.Labels[string(labelName)] = string(labelValue)
				}
			}
		}
	case model.Matrix:
		// Handle matrix results (range vector)
		for _, sampleStream := range v {
			if len(sampleStream.Values) > 0 {
				// Use the latest value
				latest := sampleStream.Values[len(sampleStream.Values)-1]
				metricName := fmt.Sprintf("%s_%s", name, string(sampleStream.Metric[model.MetricNameLabel]))
				data.Metrics[metricName] = fmt.Sprintf("%.6f", float64(latest.Value))

				// Add labels from the metric
				for labelName, labelValue := range sampleStream.Metric {
					if labelName != model.MetricNameLabel {
						data.Labels[string(labelName)] = string(labelValue)
					}
				}
			}
		}
	case *model.Scalar:
		// Handle scalar results
		data.Metrics[name] = fmt.Sprintf("%.6f", float64(v.Value))
	case *model.String:
		// Handle string results (not common for metrics)
		// Create a map and marshal to JSON for metadata
		metadata := map[string]interface{}{name: string(v.Value)}
		if metadataJSON, err := json.Marshal(metadata); err == nil {
			data.Metadata = metadataJSON
		}
	default:
		return fmt.Errorf("unsupported result type: %T", result)
	}

	return nil
}

// AddCollector adds a new component collector
func (pmc *PrometheusMetricsCollector) AddCollector(name, namespace string, selector map[string]string, queries []MetricQuery) error {
	pmc.mu.Lock()
	defer pmc.mu.Unlock()

	collector := &ComponentCollector{
		name:      name,
		namespace: namespace,
		selector:  selector,
		queries:   queries,
		registry:  pmc.registry,
	}

	pmc.collectors[name] = collector
	return nil
}

// RemoveCollector removes a component collector
func (pmc *PrometheusMetricsCollector) RemoveCollector(name string) {
	pmc.mu.Lock()
	defer pmc.mu.Unlock()
	delete(pmc.collectors, name)
}

// GetCollectors returns all registered collectors
func (pmc *PrometheusMetricsCollector) GetCollectors() map[string]*ComponentCollector {
	pmc.mu.RLock()
	defer pmc.mu.RUnlock()

	collectors := make(map[string]*ComponentCollector)
	for k, v := range pmc.collectors {
		collectors[k] = v
	}
	return collectors
}

// collectionLoop runs the main metrics collection loop
func (pmc *PrometheusMetricsCollector) collectionLoop(ctx context.Context) {
	logger := log.FromContext(ctx)
	logger.Info("Starting metrics collection loop")

	ticker := time.NewTicker(30 * time.Second) // Collect every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Metrics collection loop stopping")
			return
		case <-ticker.C:
			pmc.collectAllMetrics(ctx)
		}
	}
}

// collectAllMetrics collects metrics from all registered collectors
func (pmc *PrometheusMetricsCollector) collectAllMetrics(ctx context.Context) {
	pmc.mu.RLock()
	collectors := make(map[string]*ComponentCollector)
	for k, v := range pmc.collectors {
		collectors[k] = v
	}
	pmc.mu.RUnlock()

	logger := log.FromContext(ctx)

	for name := range collectors {
		if _, err := pmc.CollectMetrics(ctx, name); err != nil {
			logger.Error(err, "Failed to collect metrics", "source", name)
		}
	}
}

// KubernetesMetricsCollector collects metrics directly from Kubernetes API
type KubernetesMetricsCollector struct {
	client   kubernetes.Interface
	registry prometheus.Registerer

	// Custom metrics
	k8sAPIErrors   prometheus.Counter
	k8sAPIDuration prometheus.Histogram
}

// NewKubernetesMetricsCollector creates a new Kubernetes-based metrics collector
func NewKubernetesMetricsCollector(client kubernetes.Interface, registry prometheus.Registerer) *KubernetesMetricsCollector {
	kmc := &KubernetesMetricsCollector{
		client:   client,
		registry: registry,
	}

	kmc.initMetrics()
	return kmc
}

// initMetrics initializes Prometheus metrics for the Kubernetes collector
func (kmc *KubernetesMetricsCollector) initMetrics() {
	kmc.k8sAPIErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "oran_k8s_api_errors_total",
		Help: "Total number of Kubernetes API errors",
	})

	kmc.k8sAPIDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "oran_k8s_api_duration_seconds",
		Help:    "Duration of Kubernetes API calls in seconds",
		Buckets: prometheus.DefBuckets,
	})

	if kmc.registry != nil {
		safeRegister(kmc.registry, kmc.k8sAPIErrors)
		safeRegister(kmc.registry, kmc.k8sAPIDuration)
	}
}

// RegisterMetrics registers the collector's metrics with Prometheus
func (kmc *KubernetesMetricsCollector) RegisterMetrics(registry prometheus.Registerer) error {
	if registry == nil {
		return fmt.Errorf("registry cannot be nil")
	}

	kmc.registry = registry
	kmc.initMetrics()

	return nil
}

// Start begins metrics collection - FIXED SIGNATURE
func (kmc *KubernetesMetricsCollector) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting Kubernetes metrics collector")

	// Start collection tasks
	go kmc.collectPodMetrics(ctx)
	go kmc.collectNodeMetrics(ctx)

	return nil
}

// Stop halts metrics collection
func (kmc *KubernetesMetricsCollector) Stop() error {
	// Collection will be stopped by context cancellation
	return nil
}

// CollectMetrics gathers metrics from Kubernetes API - FIXED SIGNATURE
func (kmc *KubernetesMetricsCollector) CollectMetrics(ctx context.Context, source string) (*MetricsData, error) {
	startTime := time.Now()
	defer func() {
		kmc.k8sAPIDuration.Observe(time.Since(startTime).Seconds())
	}()

	logger := log.FromContext(ctx)
	logger.V(1).Info("Collecting Kubernetes metrics", "source", source)

	// Collect different types of metrics based on source
	switch source {
	case "pods":
		return kmc.collectPodMetricsData(ctx)
	case "nodes":
		return kmc.collectNodeMetricsData(ctx)
	case "services":
		return kmc.collectServiceMetricsData(ctx)
	default:
		kmc.k8sAPIErrors.Inc()
		return nil, fmt.Errorf("unknown source: %s", source)
	}
}

// Removed duplicate CollectMetrics method - use the one with context

// collectPodMetrics collects pod-level metrics
func (kmc *KubernetesMetricsCollector) collectPodMetrics(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := kmc.collectPodMetricsData(ctx); err != nil {
				logger := log.FromContext(ctx)
				logger.Error(err, "Failed to collect pod metrics")
			}
		}
	}
}

// collectNodeMetrics collects node-level metrics
func (kmc *KubernetesMetricsCollector) collectNodeMetrics(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if _, err := kmc.collectNodeMetricsData(ctx); err != nil {
				logger := log.FromContext(ctx)
				logger.Error(err, "Failed to collect node metrics")
			}
		}
	}
}

// collectPodMetricsData collects actual pod metrics data
func (kmc *KubernetesMetricsCollector) collectPodMetricsData(ctx context.Context) (*MetricsData, error) {
	// TODO: Implement actual pod metrics collection
	// This would query the Kubernetes API for pod information and metrics

	return &MetricsData{
		Timestamp: time.Now(),
		Source:    "pods",
		Metrics:   make(map[string]string),
		Labels:    make(map[string]string),
		Metadata:  json.RawMessage(`{}`),
	}, nil
}

// collectNodeMetricsData collects actual node metrics data
func (kmc *KubernetesMetricsCollector) collectNodeMetricsData(ctx context.Context) (*MetricsData, error) {
	// TODO: Implement actual node metrics collection
	// This would query the Kubernetes API for node information and metrics

	return &MetricsData{
		Timestamp: time.Now(),
		Source:    "nodes",
		Metrics:   make(map[string]string),
		Labels:    make(map[string]string),
		Metadata:  json.RawMessage(`{}`),
	}, nil
}

// collectServiceMetricsData collects actual service metrics data
func (kmc *KubernetesMetricsCollector) collectServiceMetricsData(ctx context.Context) (*MetricsData, error) {
	// TODO: Implement actual service metrics collection
	// This would query the Kubernetes API for service information and metrics

	return &MetricsData{
		Timestamp: time.Now(),
		Source:    "services",
		Metrics:   make(map[string]string),
		Labels:    make(map[string]string),
		Metadata:  json.RawMessage(`{}`),
	}, nil
}

// MetricsAggregator aggregates metrics from multiple sources
type MetricsAggregator struct {
	collectors []MetricsCollector
	mu         sync.RWMutex
	cache      map[string]*MetricsData
	registry   prometheus.Registerer

	// Aggregation metrics
	aggregationErrors prometheus.Counter
	aggregationTime   prometheus.Histogram
}

// NewMetricsAggregator creates a new metrics aggregator
func NewMetricsAggregator(registry prometheus.Registerer) *MetricsAggregator {
	ma := &MetricsAggregator{
		collectors: make([]MetricsCollector, 0),
		cache:      make(map[string]*MetricsData),
		registry:   registry,
	}

	ma.initMetrics()
	return ma
}

// initMetrics initializes Prometheus metrics for the aggregator
func (ma *MetricsAggregator) initMetrics() {
	ma.aggregationErrors = prometheus.NewCounter(prometheus.CounterOpts{
		Name: "oran_metrics_aggregation_errors_total",
		Help: "Total number of metrics aggregation errors",
	})

	ma.aggregationTime = prometheus.NewHistogram(prometheus.HistogramOpts{
		Name:    "oran_metrics_aggregation_duration_seconds",
		Help:    "Duration of metrics aggregation in seconds",
		Buckets: prometheus.DefBuckets,
	})

	if ma.registry != nil {
		safeRegister(ma.registry, ma.aggregationErrors)
		safeRegister(ma.registry, ma.aggregationTime)
	}
}

// AddCollector adds a metrics collector to the aggregator
func (ma *MetricsAggregator) AddCollector(collector MetricsCollector) {
	ma.mu.Lock()
	defer ma.mu.Unlock()
	ma.collectors = append(ma.collectors, collector)
}

// RegisterMetrics registers the aggregator's metrics
func (ma *MetricsAggregator) RegisterMetrics(registry prometheus.Registerer) error {
	if registry == nil {
		return fmt.Errorf("registry cannot be nil")
	}

	ma.registry = registry
	ma.initMetrics()

	// Register all collector metrics
	ma.mu.RLock()
	collectors := make([]MetricsCollector, len(ma.collectors))
	copy(collectors, ma.collectors)
	ma.mu.RUnlock()

	for _, collector := range collectors {
		if err := collector.RegisterMetrics(registry); err != nil {
			return fmt.Errorf("failed to register collector metrics: %w", err)
		}
	}

	return nil
}

// Start starts all collectors and begins aggregation - FIXED SIGNATURE
func (ma *MetricsAggregator) Start(ctx context.Context) error {
	logger := log.FromContext(ctx)
	logger.Info("Starting metrics aggregator")

	// Start all collectors
	ma.mu.RLock()
	collectors := make([]MetricsCollector, len(ma.collectors))
	copy(collectors, ma.collectors)
	ma.mu.RUnlock()

	for _, collector := range collectors {
		if err := collector.Start(ctx); err != nil {
			return fmt.Errorf("failed to start collector: %w", err)
		}
	}

	// Start aggregation loop
	go ma.aggregationLoop(ctx)

	return nil
}

// Stop stops all collectors and aggregation
func (ma *MetricsAggregator) Stop() error {
	ma.mu.RLock()
	collectors := make([]MetricsCollector, len(ma.collectors))
	copy(collectors, ma.collectors)
	ma.mu.RUnlock()

	for _, collector := range collectors {
		if err := collector.Stop(); err != nil {
			return fmt.Errorf("failed to stop collector: %w", err)
		}
	}

	return nil
}

// CollectMetrics aggregates metrics from all collectors for the specified source - FIXED SIGNATURE
func (ma *MetricsAggregator) CollectMetrics(ctx context.Context, source string) (*MetricsData, error) {
	startTime := time.Now()
	defer func() {
		ma.aggregationTime.Observe(time.Since(startTime).Seconds())
	}()

	ma.mu.RLock()
	collectors := make([]MetricsCollector, len(ma.collectors))
	copy(collectors, ma.collectors)
	ma.mu.RUnlock()

	aggregated := &MetricsData{
		Timestamp: time.Now(),
		Source:    source,
		Metrics:   make(map[string]string),
		Labels:    make(map[string]string),
		Metadata:  json.RawMessage(`{}`),
	}

	for _, collector := range collectors {
		metricsData, err := collector.CollectMetrics(ctx, source)
		if err != nil {
			ma.aggregationErrors.Inc()
			continue
		}

		// Convert metrics map to aggregated data
		for name, value := range metricsData.Metrics {
			aggregated.Metrics[name] = value
		}
		for k, v := range metricsData.Labels {
			aggregated.Labels[k] = v
		}
	}

	// Cache the result
	ma.mu.Lock()
	ma.cache[source] = aggregated
	ma.mu.Unlock()

	return aggregated, nil
}

// Removed duplicate CollectMetrics method - use the one with context

// aggregationLoop runs the main aggregation loop
func (ma *MetricsAggregator) aggregationLoop(ctx context.Context) {
	logger := log.FromContext(ctx)
	logger.Info("Starting metrics aggregation loop")

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			logger.Info("Metrics aggregation loop stopping")
			return
		case <-ticker.C:
			ma.performAggregation(ctx)
		}
	}
}

// performAggregation performs periodic aggregation of all metrics
func (ma *MetricsAggregator) performAggregation(ctx context.Context) {
	logger := log.FromContext(ctx)
	logger.V(1).Info("Performing metrics aggregation")

	// Define sources to aggregate
	sources := []string{"pods", "nodes", "services", "oran-components"}

	for _, source := range sources {
		if _, err := ma.CollectMetrics(ctx, source); err != nil {
			logger.Error(err, "Failed to aggregate metrics", "source", source)
		}
	}
}

// GetCachedMetrics returns cached metrics data
func (ma *MetricsAggregator) GetCachedMetrics(source string) (*MetricsData, bool) {
	ma.mu.RLock()
	defer ma.mu.RUnlock()

	data, exists := ma.cache[source]
	return data, exists
}