package availability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// AvailabilityDimension represents different dimensions of availability tracking
type AvailabilityDimension string

const (
	DimensionService    AvailabilityDimension = "service"
	DimensionComponent  AvailabilityDimension = "component"
	DimensionUserJourney AvailabilityDimension = "user_journey"
	DimensionBusiness   AvailabilityDimension = "business"
)

// HealthStatus represents the health state of a tracked entity
type HealthStatus string

const (
	HealthHealthy   HealthStatus = "healthy"
	HealthDegraded  HealthStatus = "degraded"
	HealthUnhealthy HealthStatus = "unhealthy"
	HealthUnknown   HealthStatus = "unknown"
)

// ServiceLayer represents different service layers in the architecture
type ServiceLayer string

const (
	LayerAPI        ServiceLayer = "api"
	LayerController ServiceLayer = "controller" 
	LayerProcessor  ServiceLayer = "processor"
	LayerStorage    ServiceLayer = "storage"
	LayerExternal   ServiceLayer = "external"
)

// BusinessImpact represents the criticality of a service or component
type BusinessImpact int

const (
	ImpactCritical BusinessImpact = 5 // Complete service unavailable
	ImpactHigh     BusinessImpact = 4 // Major functionality affected
	ImpactMedium   BusinessImpact = 3 // Some functionality affected
	ImpactLow      BusinessImpact = 2 // Minor functionality affected
	ImpactMinimal  BusinessImpact = 1 // No user-facing impact
)

// AvailabilityMetric represents a single availability measurement
type AvailabilityMetric struct {
	Timestamp       time.Time              `json:"timestamp"`
	Dimension       AvailabilityDimension  `json:"dimension"`
	EntityID        string                 `json:"entity_id"`
	EntityType      string                 `json:"entity_type"`
	Status          HealthStatus           `json:"status"`
	ResponseTime    time.Duration          `json:"response_time"`
	ErrorRate       float64                `json:"error_rate"`
	BusinessImpact  BusinessImpact         `json:"business_impact"`
	Layer          ServiceLayer           `json:"layer"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// ServiceEndpointConfig defines configuration for service endpoint monitoring
type ServiceEndpointConfig struct {
	Name           string         `json:"name"`
	URL            string         `json:"url"`
	Method         string         `json:"method"`
	ExpectedStatus int            `json:"expected_status"`
	Timeout        time.Duration  `json:"timeout"`
	BusinessImpact BusinessImpact `json:"business_impact"`
	Layer         ServiceLayer   `json:"layer"`
	SLAThreshold   time.Duration  `json:"sla_threshold"`
}

// ComponentConfig defines configuration for component health monitoring
type ComponentConfig struct {
	Name           string         `json:"name"`
	Namespace      string         `json:"namespace"`
	Selector       map[string]string `json:"selector"`
	ResourceType   string         `json:"resource_type"` // pod, deployment, service
	BusinessImpact BusinessImpact `json:"business_impact"`
	Layer         ServiceLayer   `json:"layer"`
}

// UserJourneyConfig defines configuration for user journey monitoring
type UserJourneyConfig struct {
	Name           string                 `json:"name"`
	Steps          []UserJourneyStep      `json:"steps"`
	BusinessImpact BusinessImpact         `json:"business_impact"`
	SLAThreshold   time.Duration          `json:"sla_threshold"`
	Metadata       map[string]interface{} `json:"metadata"`
}

// UserJourneyStep represents a single step in a user journey
type UserJourneyStep struct {
	Name         string        `json:"name"`
	Type         string        `json:"type"` // api_call, database_query, external_service
	Target       string        `json:"target"`
	Timeout      time.Duration `json:"timeout"`
	Required     bool          `json:"required"`
	Weight       float64       `json:"weight"` // Weight for calculating journey health
}

// TrackerConfig holds configuration for the availability tracker
type TrackerConfig struct {
	ServiceEndpoints []ServiceEndpointConfig `json:"service_endpoints"`
	Components       []ComponentConfig       `json:"components"`
	UserJourneys     []UserJourneyConfig     `json:"user_journeys"`
	
	// Thresholds and settings
	DegradedThreshold    time.Duration `json:"degraded_threshold"`     // Response time threshold for degraded state
	UnhealthyThreshold   time.Duration `json:"unhealthy_threshold"`    // Response time threshold for unhealthy state
	ErrorRateThreshold   float64       `json:"error_rate_threshold"`   // Error rate threshold for degraded state
	CollectionInterval   time.Duration `json:"collection_interval"`    // How often to collect metrics
	RetentionPeriod      time.Duration `json:"retention_period"`       // How long to retain metrics
	
	// Kubernetes configuration
	KubernetesNamespace string `json:"kubernetes_namespace"`
	
	// Prometheus configuration
	PrometheusURL string `json:"prometheus_url"`
}

// AvailabilityState represents the current state of availability tracking
type AvailabilityState struct {
	CurrentMetrics     map[string]*AvailabilityMetric `json:"current_metrics"`
	AggregatedStatus   HealthStatus                   `json:"aggregated_status"`
	LastUpdate         time.Time                      `json:"last_update"`
	BusinessImpactScore float64                       `json:"business_impact_score"`
}

// MultiDimensionalTracker tracks availability across multiple dimensions
type MultiDimensionalTracker struct {
	config *TrackerConfig
	
	// Clients
	kubeClient       client.Client
	kubeClientset    kubernetes.Interface
	promClient       v1.API
	cache           cache.Cache
	
	// State management
	state           *AvailabilityState
	stateMutex      sync.RWMutex
	metricsHistory  []AvailabilityMetric
	historyMutex    sync.RWMutex
	
	// Control
	ctx        context.Context
	cancel     context.CancelFunc
	stopCh     chan struct{}
	collectors []MetricCollector
	
	// Observability
	tracer trace.Tracer
}

// MetricCollector defines interface for metric collection
type MetricCollector interface {
	Collect(ctx context.Context) ([]*AvailabilityMetric, error)
	Name() string
	Dimension() AvailabilityDimension
}

// NewMultiDimensionalTracker creates a new availability tracker
func NewMultiDimensionalTracker(
	config *TrackerConfig,
	kubeClient client.Client,
	kubeClientset kubernetes.Interface,
	promClient api.Client,
	cache cache.Cache,
) (*MultiDimensionalTracker, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	var promAPI v1.API
	if promClient != nil {
		promAPI = v1.NewAPI(promClient)
	}
	
	tracker := &MultiDimensionalTracker{
		config:        config,
		kubeClient:    kubeClient,
		kubeClientset: kubeClientset,
		promClient:    promAPI,
		cache:        cache,
		ctx:          ctx,
		cancel:       cancel,
		stopCh:       make(chan struct{}),
		state: &AvailabilityState{
			CurrentMetrics:     make(map[string]*AvailabilityMetric),
			AggregatedStatus:   HealthUnknown,
			LastUpdate:         time.Now(),
			BusinessImpactScore: 0,
		},
		metricsHistory: make([]AvailabilityMetric, 0, 10000), // Pre-allocate for performance
		tracer:        otel.Tracer("availability-tracker"),
	}
	
	// Initialize collectors
	if err := tracker.initializeCollectors(); err != nil {
		return nil, fmt.Errorf("failed to initialize collectors: %w", err)
	}
	
	return tracker, nil
}

// initializeCollectors initializes all metric collectors
func (t *MultiDimensionalTracker) initializeCollectors() error {
	t.collectors = make([]MetricCollector, 0)
	
	// Service layer collector
	serviceCollector, err := NewServiceLayerCollector(t.config.ServiceEndpoints, t.promClient)
	if err != nil {
		return fmt.Errorf("failed to create service collector: %w", err)
	}
	t.collectors = append(t.collectors, serviceCollector)
	
	// Component health collector
	componentCollector, err := NewComponentHealthCollector(t.config.Components, t.kubeClient, t.kubeClientset)
	if err != nil {
		return fmt.Errorf("failed to create component collector: %w", err)
	}
	t.collectors = append(t.collectors, componentCollector)
	
	// User journey collector
	journeyCollector, err := NewUserJourneyCollector(t.config.UserJourneys, t.promClient)
	if err != nil {
		return fmt.Errorf("failed to create journey collector: %w", err)
	}
	t.collectors = append(t.collectors, journeyCollector)
	
	return nil
}

// Start begins availability tracking
func (t *MultiDimensionalTracker) Start() error {
	ctx, span := t.tracer.Start(t.ctx, "availability-tracker-start")
	defer span.End()
	
	span.AddEvent("Starting availability tracker")
	
	// Start collection goroutines
	for _, collector := range t.collectors {
		go t.runCollector(ctx, collector)
	}
	
	// Start aggregation routine
	go t.runAggregation(ctx)
	
	// Start cleanup routine
	go t.runCleanup(ctx)
	
	return nil
}

// Stop stops availability tracking
func (t *MultiDimensionalTracker) Stop() error {
	t.cancel()
	close(t.stopCh)
	return nil
}

// runCollector runs a metric collector continuously
func (t *MultiDimensionalTracker) runCollector(ctx context.Context, collector MetricCollector) {
	ticker := time.NewTicker(t.config.CollectionInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.collectMetrics(ctx, collector)
		}
	}
}

// collectMetrics collects metrics from a specific collector
func (t *MultiDimensionalTracker) collectMetrics(ctx context.Context, collector MetricCollector) {
	ctx, span := t.tracer.Start(ctx, "collect-metrics",
		trace.WithAttributes(
			attribute.String("collector", collector.Name()),
			attribute.String("dimension", string(collector.Dimension())),
		),
	)
	defer span.End()
	
	metrics, err := collector.Collect(ctx)
	if err != nil {
		span.RecordError(err)
		// Log error but continue operation
		return
	}
	
	// Store metrics
	t.historyMutex.Lock()
	t.metricsHistory = append(t.metricsHistory, convertMetricsSlice(metrics)...)
	t.historyMutex.Unlock()
	
	// Update current state
	t.updateCurrentState(metrics)
	
	span.AddEvent("Metrics collected", 
		trace.WithAttributes(attribute.Int("count", len(metrics))))
}

// updateCurrentState updates the current availability state
func (t *MultiDimensionalTracker) updateCurrentState(metrics []*AvailabilityMetric) {
	t.stateMutex.Lock()
	defer t.stateMutex.Unlock()
	
	// Update current metrics
	for _, metric := range metrics {
		key := fmt.Sprintf("%s:%s:%s", metric.Dimension, metric.EntityType, metric.EntityID)
		t.state.CurrentMetrics[key] = metric
	}
	
	// Recalculate aggregated status
	t.state.AggregatedStatus = t.calculateAggregatedStatus()
	t.state.BusinessImpactScore = t.calculateBusinessImpactScore()
	t.state.LastUpdate = time.Now()
}

// calculateAggregatedStatus calculates overall system availability status
func (t *MultiDimensionalTracker) calculateAggregatedStatus() HealthStatus {
	if len(t.state.CurrentMetrics) == 0 {
		return HealthUnknown
	}
	
	var totalWeight float64
	var weightedScore float64
	
	for _, metric := range t.state.CurrentMetrics {
		weight := float64(metric.BusinessImpact)
		totalWeight += weight
		
		var score float64
		switch metric.Status {
		case HealthHealthy:
			score = 1.0
		case HealthDegraded:
			score = 0.5
		case HealthUnhealthy:
			score = 0.0
		default:
			score = 0.0
		}
		
		weightedScore += score * weight
	}
	
	if totalWeight == 0 {
		return HealthUnknown
	}
	
	avgScore := weightedScore / totalWeight
	
	if avgScore >= 0.9 {
		return HealthHealthy
	} else if avgScore >= 0.5 {
		return HealthDegraded
	} else {
		return HealthUnhealthy
	}
}

// calculateBusinessImpactScore calculates current business impact score
func (t *MultiDimensionalTracker) calculateBusinessImpactScore() float64 {
	if len(t.state.CurrentMetrics) == 0 {
		return 0
	}
	
	var totalImpact float64
	var unhealthyImpact float64
	
	for _, metric := range t.state.CurrentMetrics {
		impact := float64(metric.BusinessImpact)
		totalImpact += impact
		
		if metric.Status == HealthUnhealthy {
			unhealthyImpact += impact
		} else if metric.Status == HealthDegraded {
			unhealthyImpact += impact * 0.5
		}
	}
	
	if totalImpact == 0 {
		return 0
	}
	
	return (unhealthyImpact / totalImpact) * 100
}

// runAggregation runs the aggregation process
func (t *MultiDimensionalTracker) runAggregation(ctx context.Context) {
	ticker := time.NewTicker(time.Minute) // Aggregate every minute
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.performAggregation(ctx)
		}
	}
}

// performAggregation performs metric aggregation
func (t *MultiDimensionalTracker) performAggregation(ctx context.Context) {
	ctx, span := t.tracer.Start(ctx, "perform-aggregation")
	defer span.End()
	
	t.stateMutex.RLock()
	currentState := *t.state
	t.stateMutex.RUnlock()
	
	// Perform any additional aggregation logic here
	// This could include calculating rolling averages, trend analysis, etc.
	
	span.AddEvent("Aggregation completed",
		trace.WithAttributes(
			attribute.String("status", string(currentState.AggregatedStatus)),
			attribute.Float64("business_impact", currentState.BusinessImpactScore),
		),
	)
}

// runCleanup runs the cleanup process for old metrics
func (t *MultiDimensionalTracker) runCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Hour) // Cleanup every hour
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			t.performCleanup(ctx)
		}
	}
}

// performCleanup removes old metrics based on retention policy
func (t *MultiDimensionalTracker) performCleanup(ctx context.Context) {
	ctx, span := t.tracer.Start(ctx, "perform-cleanup")
	defer span.End()
	
	t.historyMutex.Lock()
	defer t.historyMutex.Unlock()
	
	cutoff := time.Now().Add(-t.config.RetentionPeriod)
	originalCount := len(t.metricsHistory)
	
	// Remove metrics older than retention period
	validMetrics := make([]AvailabilityMetric, 0, len(t.metricsHistory))
	for _, metric := range t.metricsHistory {
		if metric.Timestamp.After(cutoff) {
			validMetrics = append(validMetrics, metric)
		}
	}
	
	t.metricsHistory = validMetrics
	
	span.AddEvent("Cleanup completed",
		trace.WithAttributes(
			attribute.Int("removed", originalCount-len(validMetrics)),
			attribute.Int("remaining", len(validMetrics)),
		),
	)
}

// GetCurrentState returns the current availability state
func (t *MultiDimensionalTracker) GetCurrentState() *AvailabilityState {
	t.stateMutex.RLock()
	defer t.stateMutex.RUnlock()
	
	// Return a copy to prevent external modifications
	state := *t.state
	currentMetrics := make(map[string]*AvailabilityMetric)
	for k, v := range t.state.CurrentMetrics {
		metric := *v
		currentMetrics[k] = &metric
	}
	state.CurrentMetrics = currentMetrics
	
	return &state
}

// GetMetricsHistory returns historical metrics within a time window
func (t *MultiDimensionalTracker) GetMetricsHistory(since time.Time, until time.Time) []AvailabilityMetric {
	t.historyMutex.RLock()
	defer t.historyMutex.RUnlock()
	
	result := make([]AvailabilityMetric, 0)
	for _, metric := range t.metricsHistory {
		if metric.Timestamp.After(since) && metric.Timestamp.Before(until) {
			result = append(result, metric)
		}
	}
	
	return result
}

// GetMetricsByDimension returns current metrics filtered by dimension
func (t *MultiDimensionalTracker) GetMetricsByDimension(dimension AvailabilityDimension) []*AvailabilityMetric {
	t.stateMutex.RLock()
	defer t.stateMutex.RUnlock()
	
	result := make([]*AvailabilityMetric, 0)
	for _, metric := range t.state.CurrentMetrics {
		if metric.Dimension == dimension {
			result = append(result, metric)
		}
	}
	
	return result
}

// GetMetricsByBusinessImpact returns current metrics filtered by business impact level
func (t *MultiDimensionalTracker) GetMetricsByBusinessImpact(impact BusinessImpact) []*AvailabilityMetric {
	t.stateMutex.RLock()
	defer t.stateMutex.RUnlock()
	
	result := make([]*AvailabilityMetric, 0)
	for _, metric := range t.state.CurrentMetrics {
		if metric.BusinessImpact >= impact {
			result = append(result, metric)
		}
	}
	
	return result
}

// convertMetricsSlice converts slice of metric pointers to slice of metrics
func convertMetricsSlice(metrics []*AvailabilityMetric) []AvailabilityMetric {
	result := make([]AvailabilityMetric, len(metrics))
	for i, m := range metrics {
		result[i] = *m
	}
	return result
}