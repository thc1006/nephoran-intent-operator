package availability

import (
	
	"encoding/json"
"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

// TimeWindow represents different time aggregation windows.

type TimeWindow string

const (

	// Window1Minute holds window1minute value.

	Window1Minute TimeWindow = "1m"

	// Window5Minutes holds window5minutes value.

	Window5Minutes TimeWindow = "5m"

	// Window1Hour holds window1hour value.

	Window1Hour TimeWindow = "1h"

	// Window1Day holds window1day value.

	Window1Day TimeWindow = "1d"

	// Window1Week holds window1week value.

	Window1Week TimeWindow = "7d"

	// Window1Month holds window1month value.

	Window1Month TimeWindow = "30d"
)

// SLATarget represents SLA availability targets.

type SLATarget string

const (

	// SLA99_95 holds sla99_95 value.

	SLA99_95 SLATarget = "99.95" // 4.38 hours/year, 21.56 minutes/month

	// SLA99_9 holds sla99_9 value.

	SLA99_9 SLATarget = "99.9" // 8.77 hours/year, 43.83 minutes/month

	// SLA99_5 holds sla99_5 value.

	SLA99_5 SLATarget = "99.5" // 43.83 hours/year, 3.65 hours/month

	// SLA99 holds sla99 value.

	SLA99 SLATarget = "99" // 87.66 hours/year, 7.31 hours/month

	// SLA95 holds sla95 value.

	SLA95 SLATarget = "95" // 438.3 hours/year, 36.53 hours/month

)

// ErrorBudget represents error budget calculations.

type ErrorBudget struct {
	Target SLATarget `json:"target"`

	TotalTime time.Duration `json:"total_time"`

	AllowedDowntime time.Duration `json:"allowed_downtime"`

	ActualDowntime time.Duration `json:"actual_downtime"`

	RemainingDowntime time.Duration `json:"remaining_downtime"`

	BudgetUtilization float64 `json:"budget_utilization"` // 0.0 to 1.0

	BurnRate float64 `json:"burn_rate"` // Current consumption rate

	TimeToExhaustion time.Duration `json:"time_to_exhaustion"` // Time until budget exhausted

	IsExhausted bool `json:"is_exhausted"`

	AlertThresholds BudgetAlertThresholds `json:"alert_thresholds"`
}

// BudgetAlertThresholds defines when to alert on error budget consumption.

type BudgetAlertThresholds struct {
	Warning float64 `json:"warning"` // Alert at this utilization level (e.g., 0.5)

	Critical float64 `json:"critical"` // Alert at this utilization level (e.g., 0.8)

	Emergency float64 `json:"emergency"` // Alert at this utilization level (e.g., 0.95)
}

// AvailabilityCalculation represents a calculated availability metric.

type AvailabilityCalculation struct {
	EntityID string `json:"entity_id"`

	EntityType string `json:"entity_type"`

	Dimension AvailabilityDimension `json:"dimension"`

	TimeWindow TimeWindow `json:"time_window"`

	StartTime time.Time `json:"start_time"`

	EndTime time.Time `json:"end_time"`

	// Core metrics.

	Availability float64 `json:"availability"` // 0.0 to 1.0

	Uptime time.Duration `json:"uptime"`

	Downtime time.Duration `json:"downtime"`

	TotalTime time.Duration `json:"total_time"`

	// Performance metrics.

	MeanResponseTime time.Duration `json:"mean_response_time"`

	P50ResponseTime time.Duration `json:"p50_response_time"`

	P95ResponseTime time.Duration `json:"p95_response_time"`

	P99ResponseTime time.Duration `json:"p99_response_time"`

	// Error metrics.

	ErrorRate float64 `json:"error_rate"` // 0.0 to 1.0

	TotalRequests int64 `json:"total_requests"`

	SuccessfulRequests int64 `json:"successful_requests"`

	FailedRequests int64 `json:"failed_requests"`

	// Quality metrics.

	QualityScore float64 `json:"quality_score"` // Weighted score 0.0 to 1.0

	WeightedAvailability float64 `json:"weighted_availability"` // Business impact weighted

	// Error budget.

	ErrorBudget *ErrorBudget `json:"error_budget,omitempty"`

	// Incident data.

	IncidentCount int `json:"incident_count"`

	MTTR time.Duration `json:"mttr"` // Mean Time To Recovery

	MTBF time.Duration `json:"mtbf"` // Mean Time Between Failures

	// Metadata.

	Metadata json.RawMessage `json:"metadata"`
}

// BusinessHoursConfig defines business hours for weighted calculations.

type BusinessHoursConfig struct {
	Enabled bool `json:"enabled"`

	StartHour int `json:"start_hour"` // 0-23

	EndHour int `json:"end_hour"` // 0-23

	WeekDays []time.Weekday `json:"weekdays"`

	Timezone string `json:"timezone"`

	Weight float64 `json:"weight"` // Weight multiplier for business hours

	NonBusinessWeight float64 `json:"non_business_weight"` // Weight for non-business hours
}

// MaintenanceWindow represents planned maintenance periods.

type MaintenanceWindow struct {
	ID string `json:"id"`

	Name string `json:"name"`

	StartTime time.Time `json:"start_time"`

	EndTime time.Time `json:"end_time"`

	Recurring bool `json:"recurring"`

	RecurrenceRule string `json:"recurrence_rule,omitempty"` // RRULE format

	EntityFilter map[string]string `json:"entity_filter"` // Filter which entities this applies to

	ExcludeFromSLA bool `json:"exclude_from_sla"`
}

// AvailabilityCalculatorConfig holds configuration for the calculator.

type AvailabilityCalculatorConfig struct {
	DefaultSLATarget SLATarget `json:"default_sla_target"`

	BusinessHours BusinessHoursConfig `json:"business_hours"`

	MaintenanceWindows []MaintenanceWindow `json:"maintenance_windows"`

	CalculationInterval time.Duration `json:"calculation_interval"`

	RetentionPeriod time.Duration `json:"retention_period"`

	// Aggregation settings.

	EnabledWindows []TimeWindow `json:"enabled_windows"`

	MaxDataPoints int `json:"max_data_points"` // Max data points per calculation

	SamplingInterval time.Duration `json:"sampling_interval"` // How often to sample data

	// Quality scoring weights.

	AvailabilityWeight float64 `json:"availability_weight"` // Weight for availability in quality score

	PerformanceWeight float64 `json:"performance_weight"` // Weight for performance in quality score

	ErrorRateWeight float64 `json:"error_rate_weight"` // Weight for error rate in quality score

	// Business impact weighting.

	EnableBusinessWeighting bool `json:"enable_business_weighting"`

	BusinessImpactWeights map[BusinessImpact]float64 `json:"business_impact_weights"`

	// Error budget settings.

	ErrorBudgetAlertThresholds BudgetAlertThresholds `json:"error_budget_alert_thresholds"`
}

// AvailabilityCalculator performs sophisticated availability calculations.

type AvailabilityCalculator struct {
	config *AvailabilityCalculatorConfig

	// Storage for calculated metrics.

	calculations map[string]*AvailabilityCalculation

	calculationsMutex sync.RWMutex

	// Historical data.

	calculationHistory []AvailabilityCalculation

	historyMutex sync.RWMutex

	// Control.

	ctx context.Context

	cancel context.CancelFunc

	stopCh chan struct{}

	// Observability.

	tracer trace.Tracer

	// Data sources.

	tracker *MultiDimensionalTracker
}

// NewAvailabilityCalculator creates a new availability calculator.

func NewAvailabilityCalculator(
	config *AvailabilityCalculatorConfig,

	tracker *MultiDimensionalTracker,
) (*AvailabilityCalculator, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}

	if tracker == nil {
		return nil, fmt.Errorf("tracker cannot be nil")
	}

	// Set defaults.

	if len(config.EnabledWindows) == 0 {
		config.EnabledWindows = []TimeWindow{Window1Minute, Window5Minutes, Window1Hour, Window1Day, Window1Month}
	}

	if config.AvailabilityWeight == 0 {
		config.AvailabilityWeight = 0.6
	}

	if config.PerformanceWeight == 0 {
		config.PerformanceWeight = 0.3
	}

	if config.ErrorRateWeight == 0 {
		config.ErrorRateWeight = 0.1
	}

	if config.BusinessImpactWeights == nil {
		config.BusinessImpactWeights = map[BusinessImpact]float64{
			ImpactCritical: 1.0,

			ImpactHigh: 0.8,

			ImpactMedium: 0.6,

			ImpactLow: 0.4,

			ImpactMinimal: 0.2,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	calculator := &AvailabilityCalculator{
		config: config,

		calculations: make(map[string]*AvailabilityCalculation),

		calculationHistory: make([]AvailabilityCalculation, 0, 10000),

		ctx: ctx,

		cancel: cancel,

		stopCh: make(chan struct{}),

		tracer: otel.Tracer("availability-calculator"),

		tracker: tracker,
	}

	return calculator, nil
}

// Start begins availability calculations.

func (ac *AvailabilityCalculator) Start() error {
	ctx, span := ac.tracer.Start(ac.ctx, "availability-calculator-start")

	defer span.End()

	span.AddEvent("Starting availability calculator")

	// Start calculation routines for each time window.

	for _, window := range ac.config.EnabledWindows {
		go ac.runCalculations(ctx, window)
	}

	// Start cleanup routine.

	go ac.runCleanup(ctx)

	return nil
}

// Stop stops availability calculations.

func (ac *AvailabilityCalculator) Stop() error {
	ac.cancel()

	close(ac.stopCh)

	return nil
}

// runCalculations runs calculations for a specific time window.

func (ac *AvailabilityCalculator) runCalculations(ctx context.Context, window TimeWindow) {
	interval := ac.getCalculationInterval(window)

	ticker := time.NewTicker(interval)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			ac.performCalculations(ctx, window)

		}
	}
}

// getCalculationInterval returns the calculation interval for a time window.

func (ac *AvailabilityCalculator) getCalculationInterval(window TimeWindow) time.Duration {
	switch window {

	case Window1Minute:

		return time.Minute

	case Window5Minutes:

		return 5 * time.Minute

	case Window1Hour:

		return 30 * time.Minute // Calculate every 30 minutes for hourly window

	case Window1Day:

		return time.Hour // Calculate every hour for daily window

	case Window1Week:

		return 6 * time.Hour // Calculate every 6 hours for weekly window

	case Window1Month:

		return 24 * time.Hour // Calculate daily for monthly window

	default:

		return time.Hour

	}
}

// performCalculations performs availability calculations for a time window.

func (ac *AvailabilityCalculator) performCalculations(ctx context.Context, window TimeWindow) {
	ctx, span := ac.tracer.Start(ctx, "perform-calculations",

		trace.WithAttributes(attribute.String("window", string(window))))

	defer span.End()

	now := time.Now()

	windowDuration := ac.getWindowDuration(window)

	startTime := now.Add(-windowDuration)

	// Get current state from tracker.

	state := ac.tracker.GetCurrentState()

	if state == nil {

		span.AddEvent("No tracker state available")

		return

	}

	calculationCount := 0

	// Calculate for each entity.

	for key, metric := range state.CurrentMetrics {

		calculation, err := ac.calculateAvailabilityForEntity(ctx, metric, window, startTime, now)
		if err != nil {

			span.RecordError(err)

			continue

		}

		// Store calculation.

		calcKey := fmt.Sprintf("%s:%s", key, window)

		ac.calculationsMutex.Lock()

		ac.calculations[calcKey] = calculation

		ac.calculationsMutex.Unlock()

		// Add to history.

		ac.historyMutex.Lock()

		ac.calculationHistory = append(ac.calculationHistory, *calculation)

		ac.historyMutex.Unlock()

		calculationCount++

	}

	span.AddEvent("Calculations completed",

		trace.WithAttributes(attribute.Int("calculation_count", calculationCount)))
}

// calculateAvailabilityForEntity calculates availability for a specific entity.

func (ac *AvailabilityCalculator) calculateAvailabilityForEntity(
	ctx context.Context,

	metric *AvailabilityMetric,

	window TimeWindow,

	startTime time.Time,

	endTime time.Time,
) (*AvailabilityCalculation, error) {
	_, span := ac.tracer.Start(ctx, "calculate-availability-for-entity",

		trace.WithAttributes(

			attribute.String("entity_id", metric.EntityID),

			attribute.String("entity_type", metric.EntityType),

			attribute.String("window", string(window)),
		),
	)

	defer span.End()

	// Get historical metrics for this entity.

	historicalMetrics := ac.tracker.GetMetricsHistory(startTime, endTime)

	entityMetrics := ac.filterMetricsByEntity(historicalMetrics, metric.EntityID, metric.EntityType)

	if len(entityMetrics) == 0 {
		// Use current metric as single data point.

		entityMetrics = []AvailabilityMetric{*metric}
	}

	calculation := &AvailabilityCalculation{
		EntityID: metric.EntityID,

		EntityType: metric.EntityType,

		Dimension: metric.Dimension,

		TimeWindow: window,

		StartTime: startTime,

		EndTime: endTime,

		TotalTime: endTime.Sub(startTime),

		Metadata: make(map[string]interface{}),
	}

	// Calculate core availability metrics.

	ac.calculateCoreMetrics(calculation, entityMetrics)

	// Calculate performance metrics.

	ac.calculatePerformanceMetrics(calculation, entityMetrics)

	// Calculate error metrics.

	ac.calculateErrorMetrics(calculation, entityMetrics)

	// Calculate quality score.

	ac.calculateQualityScore(calculation, metric.BusinessImpact)

	// Calculate weighted availability with business impact.

	ac.calculateWeightedAvailability(calculation, metric.BusinessImpact)

	// Apply business hours weighting if enabled.

	if ac.config.BusinessHours.Enabled {
		ac.applyBusinessHoursWeighting(calculation, startTime, endTime)
	}

	// Exclude maintenance windows.

	ac.excludeMaintenanceWindows(calculation, startTime, endTime)

	// Calculate error budget.

	if ac.config.DefaultSLATarget != "" {
		ac.calculateErrorBudget(calculation, metric.BusinessImpact)
	}

	// Calculate incident metrics.

	ac.calculateIncidentMetrics(calculation, entityMetrics)

	// Add metadata.

	calculation.Metadata["business_impact"] = metric.BusinessImpact

	calculation.Metadata["sample_count"] = len(entityMetrics)

	calculation.Metadata["calculation_time"] = time.Now()

	span.AddEvent("Entity availability calculated",

		trace.WithAttributes(

			attribute.Float64("availability", calculation.Availability),

			attribute.Float64("quality_score", calculation.QualityScore),
		),
	)

	return calculation, nil
}

// filterMetricsByEntity filters metrics for a specific entity.

func (ac *AvailabilityCalculator) filterMetricsByEntity(
	metrics []AvailabilityMetric,

	entityID string,

	entityType string,
) []AvailabilityMetric {
	filtered := make([]AvailabilityMetric, 0)

	for _, metric := range metrics {
		if metric.EntityID == entityID && metric.EntityType == entityType {
			filtered = append(filtered, metric)
		}
	}

	return filtered
}

// calculateCoreMetrics calculates core availability metrics.

func (ac *AvailabilityCalculator) calculateCoreMetrics(
	calculation *AvailabilityCalculation,

	metrics []AvailabilityMetric,
) {
	if len(metrics) == 0 {

		calculation.Availability = 0.0

		return

	}

	var totalDuration time.Duration

	var uptimeDuration time.Duration

	// Sort metrics by timestamp.

	sortedMetrics := make([]AvailabilityMetric, len(metrics))

	copy(sortedMetrics, metrics)

	sort.Slice(sortedMetrics, func(i, j int) bool {
		return sortedMetrics[i].Timestamp.Before(sortedMetrics[j].Timestamp)
	})

	// Calculate uptime based on status.

	for i, metric := range sortedMetrics {

		var duration time.Duration

		if i < len(sortedMetrics)-1 {
			duration = sortedMetrics[i+1].Timestamp.Sub(metric.Timestamp)
		} else {
			// For the last metric, assume it's valid until the end of calculation window.

			duration = calculation.EndTime.Sub(metric.Timestamp)
		}

		totalDuration += duration

		if metric.Status == HealthHealthy {
			uptimeDuration += duration
		} else if metric.Status == HealthDegraded {
			// Count degraded as partial uptime (50%).

			uptimeDuration += duration / 2
		}

		// Unhealthy and Unknown count as downtime (0% uptime).

	}

	if totalDuration > 0 {
		calculation.Availability = float64(uptimeDuration) / float64(totalDuration)
	} else {
		calculation.Availability = 0.0
	}

	calculation.Uptime = uptimeDuration

	calculation.Downtime = totalDuration - uptimeDuration
}

// calculatePerformanceMetrics calculates performance-related metrics.

func (ac *AvailabilityCalculator) calculatePerformanceMetrics(
	calculation *AvailabilityCalculation,

	metrics []AvailabilityMetric,
) {
	if len(metrics) == 0 {
		return
	}

	responseTimes := make([]time.Duration, 0, len(metrics))

	var totalResponseTime time.Duration

	for _, metric := range metrics {
		if metric.ResponseTime > 0 {

			responseTimes = append(responseTimes, metric.ResponseTime)

			totalResponseTime += metric.ResponseTime

		}
	}

	if len(responseTimes) == 0 {
		return
	}

	// Calculate mean.

	calculation.MeanResponseTime = totalResponseTime / time.Duration(len(responseTimes))

	// Sort for percentile calculations.

	sort.Slice(responseTimes, func(i, j int) bool {
		return responseTimes[i] < responseTimes[j]
	})

	// Calculate percentiles.

	calculation.P50ResponseTime = ac.calculatePercentile(responseTimes, 0.5)

	calculation.P95ResponseTime = ac.calculatePercentile(responseTimes, 0.95)

	calculation.P99ResponseTime = ac.calculatePercentile(responseTimes, 0.99)
}

// calculatePercentile calculates a percentile from sorted response times.

func (ac *AvailabilityCalculator) calculatePercentile(sortedTimes []time.Duration, percentile float64) time.Duration {
	if len(sortedTimes) == 0 {
		return 0
	}

	index := percentile * float64(len(sortedTimes)-1)

	lowerIndex := int(math.Floor(index))

	upperIndex := int(math.Ceil(index))

	if lowerIndex == upperIndex {
		return sortedTimes[lowerIndex]
	}

	// Linear interpolation.

	weight := index - float64(lowerIndex)

	lower := sortedTimes[lowerIndex]

	upper := sortedTimes[upperIndex]

	return time.Duration(float64(lower)*(1-weight) + float64(upper)*weight)
}

// calculateErrorMetrics calculates error-related metrics.

func (ac *AvailabilityCalculator) calculateErrorMetrics(
	calculation *AvailabilityCalculation,

	metrics []AvailabilityMetric,
) {
	if len(metrics) == 0 {
		return
	}

	var totalErrorRate float64

	var totalRequests int64

	var totalErrors int64

	for _, metric := range metrics {

		totalErrorRate += metric.ErrorRate

		// Estimate requests based on error rate (this could be improved with actual request count data).

		estimatedRequests := int64(100) // Assume 100 requests per sample as default

		totalRequests += estimatedRequests

		totalErrors += int64(float64(estimatedRequests) * metric.ErrorRate)

	}

	if len(metrics) > 0 {
		calculation.ErrorRate = totalErrorRate / float64(len(metrics))
	}

	calculation.TotalRequests = totalRequests

	calculation.FailedRequests = totalErrors

	calculation.SuccessfulRequests = totalRequests - totalErrors
}

// calculateQualityScore calculates an overall quality score.

func (ac *AvailabilityCalculator) calculateQualityScore(
	calculation *AvailabilityCalculation,

	businessImpact BusinessImpact,
) {
	// Normalize performance score (inverse of response time, capped at reasonable limits).

	performanceScore := 1.0

	if calculation.P95ResponseTime > 0 {

		// Assume 1 second is perfect (1.0), 10 seconds is poor (0.1).

		maxAcceptable := 10 * time.Second

		normalized := float64(calculation.P95ResponseTime) / float64(maxAcceptable)

		performanceScore = math.Max(0.1, 1.0/normalized)

		performanceScore = math.Min(1.0, performanceScore)

	}

	// Normalize error rate score.

	errorScore := math.Max(0.0, 1.0-calculation.ErrorRate)

	// Calculate weighted quality score.

	calculation.QualityScore = (calculation.Availability * ac.config.AvailabilityWeight) +

		(performanceScore * ac.config.PerformanceWeight) +

		(errorScore * ac.config.ErrorRateWeight)

	calculation.QualityScore = math.Max(0.0, math.Min(1.0, calculation.QualityScore))
}

// calculateWeightedAvailability calculates business impact weighted availability.

func (ac *AvailabilityCalculator) calculateWeightedAvailability(
	calculation *AvailabilityCalculation,

	businessImpact BusinessImpact,
) {
	if !ac.config.EnableBusinessWeighting {

		calculation.WeightedAvailability = calculation.Availability

		return

	}

	weight, exists := ac.config.BusinessImpactWeights[businessImpact]

	if !exists {
		weight = 1.0
	}

	// Apply business impact weighting.

	calculation.WeightedAvailability = calculation.Availability * weight
}

// applyBusinessHoursWeighting applies business hours weighting to availability calculation.

func (ac *AvailabilityCalculator) applyBusinessHoursWeighting(
	calculation *AvailabilityCalculation,

	startTime time.Time,

	endTime time.Time,
) {
	if !ac.config.BusinessHours.Enabled {
		return
	}

	// Load timezone.

	loc, err := time.LoadLocation(ac.config.BusinessHours.Timezone)
	if err != nil {
		loc = time.UTC
	}

	// Calculate business hours vs non-business hours.

	totalTime := endTime.Sub(startTime)

	businessTime, nonBusinessTime := ac.calculateBusinessHours(startTime, endTime, loc)

	if totalTime == 0 {
		return
	}

	// Apply weighted calculation.

	businessWeight := float64(businessTime) / float64(totalTime) * ac.config.BusinessHours.Weight

	nonBusinessWeight := float64(nonBusinessTime) / float64(totalTime) * ac.config.BusinessHours.NonBusinessWeight

	totalWeight := businessWeight + nonBusinessWeight

	if totalWeight > 0 {
		calculation.WeightedAvailability = calculation.Availability * totalWeight
	}
}

// calculateBusinessHours calculates business and non-business time within a period.

func (ac *AvailabilityCalculator) calculateBusinessHours(
	startTime time.Time,

	endTime time.Time,

	loc *time.Location,
) (businessTime, nonBusinessTime time.Duration) {
	current := startTime.In(loc)

	end := endTime.In(loc)

	for current.Before(end) {

		nextHour := current.Truncate(time.Hour).Add(time.Hour)

		if nextHour.After(end) {
			nextHour = end
		}

		duration := nextHour.Sub(current)

		if ac.isBusinessHour(current) {
			businessTime += duration
		} else {
			nonBusinessTime += duration
		}

		current = nextHour

	}

	return
}

// isBusinessHour checks if a time falls within business hours.

func (ac *AvailabilityCalculator) isBusinessHour(t time.Time) bool {
	// Check if it's a business day.

	isBusinessDay := false

	for _, weekday := range ac.config.BusinessHours.WeekDays {
		if t.Weekday() == weekday {

			isBusinessDay = true

			break

		}
	}

	if !isBusinessDay {
		return false
	}

	// Check if it's within business hours.

	hour := t.Hour()

	return hour >= ac.config.BusinessHours.StartHour && hour < ac.config.BusinessHours.EndHour
}

// excludeMaintenanceWindows excludes planned maintenance from availability calculations.

func (ac *AvailabilityCalculator) excludeMaintenanceWindows(
	calculation *AvailabilityCalculation,

	startTime time.Time,

	endTime time.Time,
) {
	var maintenanceTime time.Duration

	for _, window := range ac.config.MaintenanceWindows {

		if !window.ExcludeFromSLA {
			continue
		}

		// Check if this entity is affected by this maintenance window.

		if !ac.entityMatchesFilter(calculation, window.EntityFilter) {
			continue
		}

		// Calculate overlap with our calculation window.

		overlap := ac.calculateTimeOverlap(

			startTime, endTime,

			window.StartTime, window.EndTime,
		)

		if overlap > 0 {
			maintenanceTime += overlap
		}

	}

	// Adjust availability calculation by removing maintenance time.

	if maintenanceTime > 0 {

		adjustedTotalTime := calculation.TotalTime - maintenanceTime

		if adjustedTotalTime > 0 {

			// Recalculate availability excluding maintenance time.

			adjustedDowntime := calculation.Downtime - maintenanceTime

			if adjustedDowntime < 0 {
				adjustedDowntime = 0
			}

			calculation.Availability = float64(adjustedTotalTime-adjustedDowntime) / float64(adjustedTotalTime)

			calculation.TotalTime = adjustedTotalTime

			calculation.Downtime = adjustedDowntime

			calculation.Uptime = adjustedTotalTime - adjustedDowntime

		}

	}
}

// entityMatchesFilter checks if an entity matches a maintenance window filter.

func (ac *AvailabilityCalculator) entityMatchesFilter(
	calculation *AvailabilityCalculation,

	filter map[string]string,
) bool {
	if len(filter) == 0 {
		return true // Empty filter matches all
	}

	for key, value := range filter {
		switch key {

		case "entity_id":

			if calculation.EntityID != value {
				return false
			}

		case "entity_type":

			if calculation.EntityType != value {
				return false
			}

		case "dimension":

			if string(calculation.Dimension) != value {
				return false
			}

		default:

			// Check in metadata.

			if metaValue, exists := calculation.Metadata[key]; exists {
				if fmt.Sprintf("%v", metaValue) != value {
					return false
				}
			} else {
				return false
			}

		}
	}

	return true
}

// calculateTimeOverlap calculates overlap between two time periods.

func (ac *AvailabilityCalculator) calculateTimeOverlap(
	start1, end1, start2, end2 time.Time,
) time.Duration {
	latest := start1

	if start2.After(start1) {
		latest = start2
	}

	earliest := end1

	if end2.Before(end1) {
		earliest = end2
	}

	if latest.Before(earliest) {
		return earliest.Sub(latest)
	}

	return 0
}

// calculateErrorBudget calculates error budget for the entity.

func (ac *AvailabilityCalculator) calculateErrorBudget(
	calculation *AvailabilityCalculation,

	businessImpact BusinessImpact,
) {
	slaTarget := ac.getSLATargetForBusinessImpact(businessImpact)

	targetAvailability := ac.getSLATargetValue(slaTarget)

	calculation.ErrorBudget = &ErrorBudget{
		Target: slaTarget,

		TotalTime: calculation.TotalTime,
	}

	// Calculate allowed downtime based on SLA target.

	calculation.ErrorBudget.AllowedDowntime = time.Duration(

		float64(calculation.TotalTime) * (1.0 - targetAvailability),
	)

	// Use actual downtime from calculation.

	calculation.ErrorBudget.ActualDowntime = calculation.Downtime

	// Calculate remaining downtime.

	calculation.ErrorBudget.RemainingDowntime = calculation.ErrorBudget.AllowedDowntime - calculation.ErrorBudget.ActualDowntime

	if calculation.ErrorBudget.RemainingDowntime < 0 {

		calculation.ErrorBudget.RemainingDowntime = 0

		calculation.ErrorBudget.IsExhausted = true

	}

	// Calculate budget utilization.

	if calculation.ErrorBudget.AllowedDowntime > 0 {
		calculation.ErrorBudget.BudgetUtilization = float64(calculation.ErrorBudget.ActualDowntime) /

			float64(calculation.ErrorBudget.AllowedDowntime)
	}

	// Calculate burn rate (how fast we're consuming the budget).

	windowHours := float64(calculation.TotalTime) / float64(time.Hour)

	if windowHours > 0 {

		actualDowntimeHours := float64(calculation.ErrorBudget.ActualDowntime) / float64(time.Hour)

		calculation.ErrorBudget.BurnRate = actualDowntimeHours / windowHours

	}

	// Calculate time to exhaustion at current burn rate.

	if calculation.ErrorBudget.BurnRate > 0 && !calculation.ErrorBudget.IsExhausted {

		remainingHours := float64(calculation.ErrorBudget.RemainingDowntime) / float64(time.Hour)

		hoursToExhaustion := remainingHours / calculation.ErrorBudget.BurnRate

		calculation.ErrorBudget.TimeToExhaustion = time.Duration(hoursToExhaustion * float64(time.Hour))

	}

	// Set alert thresholds.

	calculation.ErrorBudget.AlertThresholds = ac.config.ErrorBudgetAlertThresholds
}

// getSLATargetForBusinessImpact returns appropriate SLA target based on business impact.

func (ac *AvailabilityCalculator) getSLATargetForBusinessImpact(impact BusinessImpact) SLATarget {
	switch impact {

	case ImpactCritical:

		return SLA99_95

	case ImpactHigh:

		return SLA99_9

	case ImpactMedium:

		return SLA99_5

	case ImpactLow:

		return SLA99

	case ImpactMinimal:

		return SLA95

	default:

		return ac.config.DefaultSLATarget

	}
}

// getSLATargetValue converts SLA target to numeric value.

func (ac *AvailabilityCalculator) getSLATargetValue(target SLATarget) float64 {
	switch target {

	case SLA99_95:

		return 0.9995

	case SLA99_9:

		return 0.999

	case SLA99_5:

		return 0.995

	case SLA99:

		return 0.99

	case SLA95:

		return 0.95

	default:

		return 0.99

	}
}

// calculateIncidentMetrics calculates incident-related metrics.

func (ac *AvailabilityCalculator) calculateIncidentMetrics(
	calculation *AvailabilityCalculation,

	metrics []AvailabilityMetric,
) {
	if len(metrics) == 0 {
		return
	}

	// Identify incidents (periods of unhealthy status).

	incidents := ac.identifyIncidents(metrics)

	calculation.IncidentCount = len(incidents)

	if len(incidents) == 0 {
		return
	}

	// Calculate MTTR (Mean Time To Recovery).

	var totalRecoveryTime time.Duration

	for _, incident := range incidents {
		totalRecoveryTime += incident.Duration
	}

	calculation.MTTR = totalRecoveryTime / time.Duration(len(incidents))

	// Calculate MTBF (Mean Time Between Failures).

	if len(incidents) > 1 {

		totalTimeBetweenFailures := calculation.TotalTime - totalRecoveryTime

		calculation.MTBF = totalTimeBetweenFailures / time.Duration(len(incidents)-1)

	}
}

// Incident represents a service incident.

type Incident struct {
	StartTime time.Time

	EndTime time.Time

	Duration time.Duration

	Severity string
}

// identifyIncidents identifies incidents from metrics.

func (ac *AvailabilityCalculator) identifyIncidents(metrics []AvailabilityMetric) []Incident {
	// Sort metrics by timestamp.

	sortedMetrics := make([]AvailabilityMetric, len(metrics))

	copy(sortedMetrics, metrics)

	sort.Slice(sortedMetrics, func(i, j int) bool {
		return sortedMetrics[i].Timestamp.Before(sortedMetrics[j].Timestamp)
	})

	incidents := make([]Incident, 0)

	var currentIncident *Incident

	for _, metric := range sortedMetrics {
		if metric.Status == HealthUnhealthy {
			if currentIncident == nil {
				// Start of new incident.

				currentIncident = &Incident{
					StartTime: metric.Timestamp,

					Severity: "unhealthy",
				}
			}
		} else if currentIncident != nil {

			// End of current incident.

			currentIncident.EndTime = metric.Timestamp

			currentIncident.Duration = currentIncident.EndTime.Sub(currentIncident.StartTime)

			incidents = append(incidents, *currentIncident)

			currentIncident = nil

		}
	}

	// Handle ongoing incident.

	if currentIncident != nil {

		currentIncident.EndTime = time.Now()

		currentIncident.Duration = currentIncident.EndTime.Sub(currentIncident.StartTime)

		incidents = append(incidents, *currentIncident)

	}

	return incidents
}

// getWindowDuration converts TimeWindow to time.Duration.

func (ac *AvailabilityCalculator) getWindowDuration(window TimeWindow) time.Duration {
	switch window {

	case Window1Minute:

		return time.Minute

	case Window5Minutes:

		return 5 * time.Minute

	case Window1Hour:

		return time.Hour

	case Window1Day:

		return 24 * time.Hour

	case Window1Week:

		return 7 * 24 * time.Hour

	case Window1Month:

		return 30 * 24 * time.Hour

	default:

		return time.Hour

	}
}

// runCleanup runs cleanup of old calculations.

func (ac *AvailabilityCalculator) runCleanup(ctx context.Context) {
	ticker := time.NewTicker(time.Hour)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			ac.performCleanup(ctx)

		}
	}
}

// performCleanup cleans up old calculations.

func (ac *AvailabilityCalculator) performCleanup(ctx context.Context) {
	_, span := ac.tracer.Start(ctx, "perform-cleanup")

	defer span.End()

	cutoff := time.Now().Add(-ac.config.RetentionPeriod)

	// Clean calculation history.

	ac.historyMutex.Lock()

	validCalculations := make([]AvailabilityCalculation, 0)

	for _, calc := range ac.calculationHistory {
		if calc.EndTime.After(cutoff) {
			validCalculations = append(validCalculations, calc)
		}
	}

	removed := len(ac.calculationHistory) - len(validCalculations)

	ac.calculationHistory = validCalculations

	ac.historyMutex.Unlock()

	span.AddEvent("Cleanup completed",

		trace.WithAttributes(attribute.Int("calculations_removed", removed)))
}

// GetCalculation returns a specific calculation.

func (ac *AvailabilityCalculator) GetCalculation(entityID, entityType string, window TimeWindow) (*AvailabilityCalculation, bool) {
	ac.calculationsMutex.RLock()

	defer ac.calculationsMutex.RUnlock()

	key := fmt.Sprintf("%s:%s:%s:%s", DimensionService, entityType, entityID, window)

	calc, exists := ac.calculations[key]

	return calc, exists
}

// GetCalculationsForEntity returns all calculations for an entity.

func (ac *AvailabilityCalculator) GetCalculationsForEntity(entityID, entityType string) map[TimeWindow]*AvailabilityCalculation {
	ac.calculationsMutex.RLock()

	defer ac.calculationsMutex.RUnlock()

	result := make(map[TimeWindow]*AvailabilityCalculation)

	for _, window := range ac.config.EnabledWindows {

		key := fmt.Sprintf("%s:%s:%s:%s", DimensionService, entityType, entityID, window)

		if calc, exists := ac.calculations[key]; exists {
			result[window] = calc
		}

	}

	return result
}

// GetCalculationHistory returns historical calculations.

func (ac *AvailabilityCalculator) GetCalculationHistory(
	entityID, entityType string,

	window TimeWindow,

	since time.Time,

	until time.Time,
) []AvailabilityCalculation {
	ac.historyMutex.RLock()

	defer ac.historyMutex.RUnlock()

	result := make([]AvailabilityCalculation, 0)

	for _, calc := range ac.calculationHistory {
		if calc.EntityID == entityID &&

			calc.EntityType == entityType &&

			calc.TimeWindow == window &&

			calc.EndTime.After(since) &&

			calc.EndTime.Before(until) {

			result = append(result, calc)
		}
	}

	return result
}

// GetSLACompliance returns SLA compliance status for an entity.

func (ac *AvailabilityCalculator) GetSLACompliance(entityID, entityType string, window TimeWindow) (*SLAComplianceStatus, error) {
	calc, exists := ac.GetCalculation(entityID, entityType, window)

	if !exists {
		return nil, fmt.Errorf("no calculation found for entity %s:%s window %s", entityType, entityID, window)
	}

	if calc.ErrorBudget == nil {
		return nil, fmt.Errorf("no error budget calculated for entity %s:%s", entityType, entityID)
	}

	status := &SLAComplianceStatus{
		EntityID: entityID,

		EntityType: entityType,

		TimeWindow: window,

		SLATarget: calc.ErrorBudget.Target,

		CurrentAvailability: calc.Availability,

		RequiredAvailability: ac.getSLATargetValue(calc.ErrorBudget.Target),

		IsCompliant: calc.Availability >= ac.getSLATargetValue(calc.ErrorBudget.Target),

		ErrorBudget: *calc.ErrorBudget,

		LastUpdate: calc.EndTime,
	}

	// Determine compliance status.

	if status.IsCompliant {
		if calc.ErrorBudget.BudgetUtilization < calc.ErrorBudget.AlertThresholds.Warning {
			status.ComplianceLevel = "healthy"
		} else if calc.ErrorBudget.BudgetUtilization < calc.ErrorBudget.AlertThresholds.Critical {
			status.ComplianceLevel = "warning"
		} else {
			status.ComplianceLevel = "critical"
		}
	} else {
		status.ComplianceLevel = "breach"
	}

	return status, nil
}

// SLAComplianceStatus represents SLA compliance status.

type SLAComplianceStatus struct {
	EntityID string `json:"entity_id"`

	EntityType string `json:"entity_type"`

	TimeWindow TimeWindow `json:"time_window"`

	SLATarget SLATarget `json:"sla_target"`

	CurrentAvailability float64 `json:"current_availability"`

	RequiredAvailability float64 `json:"required_availability"`

	IsCompliant bool `json:"is_compliant"`

	ComplianceLevel string `json:"compliance_level"` // healthy, warning, critical, breach

	ErrorBudget ErrorBudget `json:"error_budget"`

	LastUpdate time.Time `json:"last_update"`
}
