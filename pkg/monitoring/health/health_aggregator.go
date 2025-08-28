package health

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/thc1006/nephoran-intent-operator/pkg/health"
)

// HealthAggregator provides multi-dimensional health aggregation and analysis.
type HealthAggregator struct {
	// Core configuration.
	logger      *slog.Logger
	serviceName string

	// Data sources.
	enhancedChecker   *EnhancedHealthChecker
	dependencyTracker *DependencyHealthTracker

	// Aggregation configuration.
	aggregationConfig *AggregationConfig

	// Business impact weighting.
	businessWeights map[string]BusinessWeight
	featureWeights  map[string]FeatureWeight

	// User journey monitoring.
	userJourneys   map[string]*UserJourney
	journeyResults map[string]*JourneyHealthResult
	journeyMu      sync.RWMutex

	// Trend analysis.
	trendAnalyzer  *TrendAnalyzer
	historicalData map[string][]HealthDataPoint
	historyMu      sync.RWMutex

	// Health reporting.
	reportGenerator *ReportGenerator

	// Metrics.
	aggregatorMetrics *AggregatorMetrics
}

// AggregationConfig holds configuration for health aggregation.
type AggregationConfig struct {
	// Aggregation methods.
	DefaultMethod AggregationMethod                `json:"default_method"`
	TierMethods   map[HealthTier]AggregationMethod `json:"tier_methods"`

	// Weighting configuration.
	BusinessWeightEnabled    bool `json:"business_weight_enabled"`
	PerformanceWeightEnabled bool `json:"performance_weight_enabled"`

	// Trend analysis.
	TrendAnalysisEnabled bool          `json:"trend_analysis_enabled"`
	TrendWindow          time.Duration `json:"trend_window"`
	TrendMinDataPoints   int           `json:"trend_min_data_points"`

	// Forecasting.
	ForecastingEnabled bool          `json:"forecasting_enabled"`
	ForecastHorizon    time.Duration `json:"forecast_horizon"`

	// User journey tracking.
	UserJourneyEnabled bool          `json:"user_journey_enabled"`
	JourneyTimeout     time.Duration `json:"journey_timeout"`

	// Health reporting.
	ReportingEnabled bool          `json:"reporting_enabled"`
	ReportInterval   time.Duration `json:"report_interval"`
}

// AggregationMethod defines how health scores are aggregated.
type AggregationMethod string

const (
	// AggregationWeightedAverage holds aggregationweightedaverage value.
	AggregationWeightedAverage AggregationMethod = "weighted_average"
	// AggregationMinimum holds aggregationminimum value.
	AggregationMinimum AggregationMethod = "minimum"
	// AggregationMaximum holds aggregationmaximum value.
	AggregationMaximum AggregationMethod = "maximum"
	// AggregationMedian holds aggregationmedian value.
	AggregationMedian AggregationMethod = "median"
	// AggregationPercentile holds aggregationpercentile value.
	AggregationPercentile AggregationMethod = "percentile"
	// AggregationCustom holds aggregationcustom value.
	AggregationCustom AggregationMethod = "custom"
)

// BusinessWeight represents the business importance of a component.
type BusinessWeight struct {
	Component        string  `json:"component"`
	Weight           float64 `json:"weight"`
	RevenueImpact    float64 `json:"revenue_impact"`
	UserImpact       float64 `json:"user_impact"`
	ComplianceImpact float64 `json:"compliance_impact"`
}

// FeatureWeight represents the importance of specific features.
type FeatureWeight struct {
	Feature          string  `json:"feature"`
	Weight           float64 `json:"weight"`
	UserBase         int     `json:"user_base"`
	CriticalityLevel string  `json:"criticality_level"`
}

// UserJourney defines a user journey for health monitoring.
type UserJourney struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	Description    string              `json:"description"`
	Steps          []JourneyStep       `json:"steps"`
	Timeout        time.Duration       `json:"timeout"`
	CriticalPath   bool                `json:"critical_path"`
	BusinessImpact BusinessImpactLevel `json:"business_impact"`
	UserSegment    string              `json:"user_segment"`
}

// JourneyStep represents a step in a user journey.
type JourneyStep struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Dependencies []string      `json:"dependencies"`
	MaxLatency   time.Duration `json:"max_latency"`
	Required     bool          `json:"required"`
	Weight       float64       `json:"weight"`
	HealthChecks []string      `json:"health_checks"`
}

// JourneyHealthResult represents the health result of a user journey.
type JourneyHealthResult struct {
	JourneyID      string                 `json:"journey_id"`
	Status         health.Status          `json:"status"`
	OverallScore   float64                `json:"overall_score"`
	StepResults    map[string]*StepResult `json:"step_results"`
	TotalLatency   time.Duration          `json:"total_latency"`
	BottleneckStep string                 `json:"bottleneck_step,omitempty"`
	LastChecked    time.Time              `json:"last_checked"`
	Availability   float64                `json:"availability"`
	UserImpact     UserImpactAssessment   `json:"user_impact"`
}

// StepResult represents the result of a journey step.
type StepResult struct {
	StepID            string                   `json:"step_id"`
	Status            health.Status            `json:"status"`
	Score             float64                  `json:"score"`
	Latency           time.Duration            `json:"latency"`
	Error             string                   `json:"error,omitempty"`
	DependencyResults map[string]health.Status `json:"dependency_results"`
}

// UserImpactAssessment assesses the impact on users.
type UserImpactAssessment struct {
	AffectedUsers     int             `json:"affected_users"`
	ImpactLevel       UserImpactLevel `json:"impact_level"`
	Features          []string        `json:"features"`
	Workarounds       []string        `json:"workarounds"`
	EstimatedDuration time.Duration   `json:"estimated_duration"`
}

// TrendAnalyzer analyzes health trends and patterns.
type TrendAnalyzer struct {
	logger               *slog.Logger
	minDataPoints        int
	confidenceThreshold  float64
	seasonalityDetection bool
}

// HealthTrendResult represents the result of trend analysis.
type HealthTrendResult struct {
	Component       string             `json:"component"`
	Trend           TrendDirection     `json:"trend"`
	Confidence      float64            `json:"confidence"`
	Slope           float64            `json:"slope"`
	Volatility      float64            `json:"volatility"`
	Seasonality     *SeasonalPattern   `json:"seasonality,omitempty"`
	Predictions     []HealthPrediction `json:"predictions"`
	Recommendations []string           `json:"recommendations"`
}

// SeasonalPattern represents detected seasonal patterns in health data.
type SeasonalPattern struct {
	Period      time.Duration `json:"period"`
	Amplitude   float64       `json:"amplitude"`
	Phase       float64       `json:"phase"`
	Confidence  float64       `json:"confidence"`
	Description string        `json:"description"`
}

// ReportGenerator generates comprehensive health reports.
type ReportGenerator struct {
	logger          *slog.Logger
	templateEngine  *ReportTemplateEngine
	dashboardConfig *DashboardConfig
}

// ReportTemplateEngine handles report template processing.
type ReportTemplateEngine struct {
	templates map[string]*ReportTemplate
	mu        sync.RWMutex
}

// ReportTemplate defines a health report template.
type ReportTemplate struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Type        ReportType      `json:"type"`
	Stakeholder StakeholderType `json:"stakeholder"`
	Sections    []ReportSection `json:"sections"`
	Format      ReportFormat    `json:"format"`
	Schedule    ReportSchedule  `json:"schedule"`
}

// ReportType defines the type of health report.
type ReportType string

const (
	// ReportTypeExecutive holds reporttypeexecutive value.
	ReportTypeExecutive ReportType = "executive"
	// ReportTypeOperational holds reporttypeoperational value.
	ReportTypeOperational ReportType = "operational"
	// ReportTypeTechnical holds reporttypetechnical value.
	ReportTypeTechnical ReportType = "technical"
	// ReportTypeCompliance holds reporttypecompliance value.
	ReportTypeCompliance ReportType = "compliance"
	// ReportTypeIncident holds reporttypeincident value.
	ReportTypeIncident ReportType = "incident"
)

// StakeholderType defines the target stakeholder for reports.
type StakeholderType string

const (
	// StakeholderExecutive holds stakeholderexecutive value.
	StakeholderExecutive StakeholderType = "executive"
	// StakeholderOperations holds stakeholderoperations value.
	StakeholderOperations StakeholderType = "operations"
	// StakeholderDevelopment holds stakeholderdevelopment value.
	StakeholderDevelopment StakeholderType = "development"
	// StakeholderCompliance holds stakeholdercompliance value.
	StakeholderCompliance StakeholderType = "compliance"
	// StakeholderCustomer holds stakeholdercustomer value.
	StakeholderCustomer StakeholderType = "customer"
)

// ReportSection defines a section of a health report.
type ReportSection struct {
	ID            string                 `json:"id"`
	Title         string                 `json:"title"`
	Type          ReportSectionType      `json:"type"`
	DataSources   []string               `json:"data_sources"`
	Visualization VisualizationType      `json:"visualization"`
	Filters       map[string]interface{} `json:"filters,omitempty"`
}

// ReportSectionType defines the type of report section.
type ReportSectionType string

const (
	// SectionOverview holds sectionoverview value.
	SectionOverview ReportSectionType = "overview"
	// SectionTrends holds sectiontrends value.
	SectionTrends ReportSectionType = "trends"
	// SectionAlerts holds sectionalerts value.
	SectionAlerts ReportSectionType = "alerts"
	// SectionPerformance holds sectionperformance value.
	SectionPerformance ReportSectionType = "performance"
	// SectionDependencies holds sectiondependencies value.
	SectionDependencies ReportSectionType = "dependencies"
	// SectionUserJourneys holds sectionuserjourneys value.
	SectionUserJourneys ReportSectionType = "user_journeys"
	// SectionRecommendations holds sectionrecommendations value.
	SectionRecommendations ReportSectionType = "recommendations"
)

// VisualizationType defines visualization types for report sections.
type VisualizationType string

const (
	// VizTable holds viztable value.
	VizTable VisualizationType = "table"
	// VizChart holds vizchart value.
	VizChart VisualizationType = "chart"
	// VizGauge holds vizgauge value.
	VizGauge VisualizationType = "gauge"
	// VizHeatmap holds vizheatmap value.
	VizHeatmap VisualizationType = "heatmap"
	// VizTimeseries holds viztimeseries value.
	VizTimeseries VisualizationType = "timeseries"
	// VizDashboard holds vizdashboard value.
	VizDashboard VisualizationType = "dashboard"
)

// ReportFormat defines the output format of reports.
type ReportFormat string

const (
	// FormatHTML holds formathtml value.
	FormatHTML ReportFormat = "html"
	// FormatPDF holds formatpdf value.
	FormatPDF ReportFormat = "pdf"
	// FormatJSON holds formatjson value.
	FormatJSON ReportFormat = "json"
	// FormatCSV holds formatcsv value.
	FormatCSV ReportFormat = "csv"
	// FormatDashboard holds formatdashboard value.
	FormatDashboard ReportFormat = "dashboard"
)

// ReportSchedule defines when reports are generated.
type ReportSchedule struct {
	Frequency   ReportFrequency `json:"frequency"`
	Time        string          `json:"time"`          // HH:MM format
	DaysOfWeek  []int           `json:"days_of_week"`  // 0=Sunday, 1=Monday, etc.
	DaysOfMonth []int           `json:"days_of_month"` // 1-31
	Timezone    string          `json:"timezone"`
	Enabled     bool            `json:"enabled"`
}

// ReportFrequency defines how often reports are generated.
type ReportFrequency string

const (
	// FrequencyRealtime holds frequencyrealtime value.
	FrequencyRealtime ReportFrequency = "realtime"
	// FrequencyHourly holds frequencyhourly value.
	FrequencyHourly ReportFrequency = "hourly"
	// FrequencyDaily holds frequencydaily value.
	FrequencyDaily ReportFrequency = "daily"
	// FrequencyWeekly holds frequencyweekly value.
	FrequencyWeekly ReportFrequency = "weekly"
	// FrequencyMonthly holds frequencymonthly value.
	FrequencyMonthly ReportFrequency = "monthly"
	// FrequencyQuarterly holds frequencyquarterly value.
	FrequencyQuarterly ReportFrequency = "quarterly"
	// FrequencyOnDemand holds frequencyondemand value.
	FrequencyOnDemand ReportFrequency = "on_demand"
)

// DashboardConfig configures health dashboards.
type DashboardConfig struct {
	Enabled         bool                       `json:"enabled"`
	RefreshInterval time.Duration              `json:"refresh_interval"`
	Widgets         []DashboardWidget          `json:"widgets"`
	Layouts         map[string]DashboardLayout `json:"layouts"`
}

// DashboardWidget defines a dashboard widget.
type DashboardWidget struct {
	ID              string            `json:"id"`
	Type            WidgetType        `json:"type"`
	Title           string            `json:"title"`
	DataSource      string            `json:"data_source"`
	Query           string            `json:"query"`
	Visualization   VisualizationType `json:"visualization"`
	RefreshInterval time.Duration     `json:"refresh_interval"`
	Position        WidgetPosition    `json:"position"`
}

// WidgetType defines the type of dashboard widget.
type WidgetType string

const (
	// WidgetHealthScore holds widgethealthscore value.
	WidgetHealthScore WidgetType = "health_score"
	// WidgetTrendChart holds widgettrendchart value.
	WidgetTrendChart WidgetType = "trend_chart"
	// WidgetAlertList holds widgetalertlist value.
	WidgetAlertList WidgetType = "alert_list"
	// WidgetDependencyMap holds widgetdependencymap value.
	WidgetDependencyMap WidgetType = "dependency_map"
	// WidgetJourneyStatus holds widgetjourneystatus value.
	WidgetJourneyStatus WidgetType = "journey_status"
	// WidgetMetricsTable holds widgetmetricstable value.
	WidgetMetricsTable WidgetType = "metrics_table"
)

// WidgetPosition defines widget position and size.
type WidgetPosition struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Width  int `json:"width"`
	Height int `json:"height"`
}

// DashboardLayout defines dashboard layout configuration.
type DashboardLayout struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Widgets     []string `json:"widgets"`
	Responsive  bool     `json:"responsive"`
}

// AggregatorMetrics contains Prometheus metrics for the health aggregator.
type AggregatorMetrics struct {
	AggregationLatency    prometheus.Histogram
	AggregatedHealthScore *prometheus.GaugeVec
	JourneyHealthScore    *prometheus.GaugeVec
	TrendConfidence       *prometheus.GaugeVec
	BusinessImpactScore   *prometheus.GaugeVec
	ReportGenerationTime  prometheus.Histogram
}

// AggregatedHealthResult represents the result of health aggregation.
type AggregatedHealthResult struct {
	// Overall health information.
	Timestamp     time.Time     `json:"timestamp"`
	OverallStatus health.Status `json:"overall_status"`
	OverallScore  float64       `json:"overall_score"`

	// Multi-dimensional scores.
	BusinessScore       float64 `json:"business_score"`
	TechnicalScore      float64 `json:"technical_score"`
	UserExperienceScore float64 `json:"user_experience_score"`
	PerformanceScore    float64 `json:"performance_score"`

	// Tier-based aggregation.
	TierScores map[HealthTier]TierHealthResult `json:"tier_scores"`

	// Component-based aggregation.
	ComponentScores map[string]ComponentHealthResult `json:"component_scores"`

	// Journey-based aggregation.
	JourneyResults map[string]*JourneyHealthResult `json:"journey_results"`

	// Trend analysis.
	TrendAnalysis map[string]*HealthTrendResult `json:"trend_analysis,omitempty"`

	// Business impact assessment.
	BusinessImpact *BusinessImpactResult `json:"business_impact"`

	// Recommendations.
	Recommendations []HealthRecommendation `json:"recommendations"`

	// Execution metadata.
	AggregationMetadata AggregationMetadata `json:"aggregation_metadata"`
}

// TierHealthResult represents health results for a specific tier.
type TierHealthResult struct {
	Tier           HealthTier    `json:"tier"`
	Status         health.Status `json:"status"`
	Score          float64       `json:"score"`
	ComponentCount int           `json:"component_count"`
	HealthyCount   int           `json:"healthy_count"`
	UnhealthyCount int           `json:"unhealthy_count"`
	Weight         float64       `json:"weight"`
	CriticalIssues []string      `json:"critical_issues,omitempty"`
}

// ComponentHealthResult represents health results for a specific component.
type ComponentHealthResult struct {
	Name           string        `json:"name"`
	Type           string        `json:"type"`
	Status         health.Status `json:"status"`
	Score          float64       `json:"score"`
	BusinessWeight float64       `json:"business_weight"`
	Dependencies   []string      `json:"dependencies"`
	LastUpdated    time.Time     `json:"last_updated"`
	Issues         []string      `json:"issues,omitempty"`
}

// BusinessImpactResult represents the business impact of health status.
type BusinessImpactResult struct {
	Level                BusinessImpactLevel   `json:"level"`
	EstimatedRevenueLoss float64               `json:"estimated_revenue_loss"`
	AffectedCustomers    int                   `json:"affected_customers"`
	AffectedFeatures     []string              `json:"affected_features"`
	ComplianceRisks      []string              `json:"compliance_risks"`
	ReputationImpact     ReputationImpactLevel `json:"reputation_impact"`
}

// ReputationImpactLevel represents the level of reputation impact.
type ReputationImpactLevel string

const (
	// ReputationNone holds reputationnone value.
	ReputationNone ReputationImpactLevel = "none"
	// ReputationMinor holds reputationminor value.
	ReputationMinor ReputationImpactLevel = "minor"
	// ReputationModerate holds reputationmoderate value.
	ReputationModerate ReputationImpactLevel = "moderate"
	// ReputationMajor holds reputationmajor value.
	ReputationMajor ReputationImpactLevel = "major"
	// ReputationSevere holds reputationsevere value.
	ReputationSevere ReputationImpactLevel = "severe"
)

// HealthRecommendation represents a recommendation based on health analysis.
type HealthRecommendation struct {
	ID                 string                 `json:"id"`
	Type               RecommendationType     `json:"type"`
	Priority           RecommendationPriority `json:"priority"`
	Title              string                 `json:"title"`
	Description        string                 `json:"description"`
	AffectedComponents []string               `json:"affected_components"`
	Actions            []RecommendedAction    `json:"actions"`
	EstimatedImpact    RecommendationImpact   `json:"estimated_impact"`
	Confidence         float64                `json:"confidence"`
	AutomationLevel    AutomationLevel        `json:"automation_level"`
}

// RecommendationType defines the type of recommendation.
type RecommendationType string

const (
	// RecommendationScaling holds recommendationscaling value.
	RecommendationScaling RecommendationType = "scaling"
	// RecommendationOptimization holds recommendationoptimization value.
	RecommendationOptimization RecommendationType = "optimization"
	// RecommendationRecovery holds recommendationrecovery value.
	RecommendationRecovery RecommendationType = "recovery"
	// RecommendationMaintenance holds recommendationmaintenance value.
	RecommendationMaintenance RecommendationType = "maintenance"
	// RecommendationUpgrade holds recommendationupgrade value.
	RecommendationUpgrade RecommendationType = "upgrade"
	// RecommendationConfiguration holds recommendationconfiguration value.
	RecommendationConfiguration RecommendationType = "configuration"
)

// RecommendationPriority defines the priority of a recommendation.
type RecommendationPriority string

const (
	// PriorityEmergency holds priorityemergency value.
	PriorityEmergency RecommendationPriority = "emergency"
	// PriorityHigh holds priorityhigh value.
	PriorityHigh RecommendationPriority = "high"
	// PriorityMedium holds prioritymedium value.
	PriorityMedium RecommendationPriority = "medium"
	// PriorityLow holds prioritylow value.
	PriorityLow RecommendationPriority = "low"
	// PriorityInformational holds priorityinformational value.
	PriorityInformational RecommendationPriority = "informational"
)

// RecommendationImpact represents the estimated impact of a recommendation.
type RecommendationImpact struct {
	HealthImprovement  float64       `json:"health_improvement"`
	PerformanceGain    float64       `json:"performance_gain"`
	CostSavings        float64       `json:"cost_savings"`
	RiskReduction      float64       `json:"risk_reduction"`
	ImplementationTime time.Duration `json:"implementation_time"`
}

// AutomationLevel defines how automated a recommendation can be.
type AutomationLevel string

const (
	// AutomationFull holds automationfull value.
	AutomationFull AutomationLevel = "full"
	// AutomationPartial holds automationpartial value.
	AutomationPartial AutomationLevel = "partial"
	// AutomationManual holds automationmanual value.
	AutomationManual AutomationLevel = "manual"
)

// AggregationMetadata contains metadata about the aggregation process.
type AggregationMetadata struct {
	ExecutionTime        time.Duration     `json:"execution_time"`
	DataPointsProcessed  int               `json:"data_points_processed"`
	AggregationMethod    AggregationMethod `json:"aggregation_method"`
	WeightingEnabled     bool              `json:"weighting_enabled"`
	TrendAnalysisEnabled bool              `json:"trend_analysis_enabled"`
	DataSources          []string          `json:"data_sources"`
}

// NewHealthAggregator creates a new health aggregator.
func NewHealthAggregator(serviceName string, enhancedChecker *EnhancedHealthChecker, dependencyTracker *DependencyHealthTracker, logger *slog.Logger) *HealthAggregator {
	if logger == nil {
		logger = slog.Default()
	}

	aggregator := &HealthAggregator{
		logger:            logger.With("component", "health_aggregator"),
		serviceName:       serviceName,
		enhancedChecker:   enhancedChecker,
		dependencyTracker: dependencyTracker,
		aggregationConfig: defaultAggregationConfig(),
		businessWeights:   make(map[string]BusinessWeight),
		featureWeights:    make(map[string]FeatureWeight),
		userJourneys:      make(map[string]*UserJourney),
		journeyResults:    make(map[string]*JourneyHealthResult),
		historicalData:    make(map[string][]HealthDataPoint),
		aggregatorMetrics: initializeAggregatorMetrics(),
	}

	// Initialize components.
	aggregator.trendAnalyzer = &TrendAnalyzer{
		logger:               logger.With("component", "trend_analyzer"),
		minDataPoints:        10,
		confidenceThreshold:  0.7,
		seasonalityDetection: true,
	}

	aggregator.reportGenerator = &ReportGenerator{
		logger:         logger.With("component", "report_generator"),
		templateEngine: &ReportTemplateEngine{templates: make(map[string]*ReportTemplate)},
		dashboardConfig: &DashboardConfig{
			Enabled:         true,
			RefreshInterval: 30 * time.Second,
		},
	}

	// Configure default business weights.
	aggregator.configureDefaultWeights()

	// Configure default user journeys.
	aggregator.configureDefaultUserJourneys()

	return aggregator
}

// defaultAggregationConfig returns default aggregation configuration.
func defaultAggregationConfig() *AggregationConfig {
	return &AggregationConfig{
		DefaultMethod: AggregationWeightedAverage,
		TierMethods: map[HealthTier]AggregationMethod{
			TierSystem:     AggregationMinimum,
			TierService:    AggregationWeightedAverage,
			TierComponent:  AggregationWeightedAverage,
			TierDependency: AggregationWeightedAverage,
		},
		BusinessWeightEnabled:    true,
		PerformanceWeightEnabled: true,
		TrendAnalysisEnabled:     true,
		TrendWindow:              24 * time.Hour,
		TrendMinDataPoints:       10,
		ForecastingEnabled:       true,
		ForecastHorizon:          4 * time.Hour,
		UserJourneyEnabled:       true,
		JourneyTimeout:           30 * time.Second,
		ReportingEnabled:         true,
		ReportInterval:           time.Hour,
	}
}

// initializeAggregatorMetrics initializes Prometheus metrics for the aggregator.
func initializeAggregatorMetrics() *AggregatorMetrics {
	return &AggregatorMetrics{
		AggregationLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "health_aggregation_latency_seconds",
			Help:    "Latency of health aggregation operations",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 8),
		}),

		AggregatedHealthScore: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "aggregated_health_score",
			Help: "Aggregated health scores by dimension",
		}, []string{"dimension", "tier"}),

		JourneyHealthScore: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "user_journey_health_score",
			Help: "Health scores for user journeys",
		}, []string{"journey_id", "journey_name"}),

		TrendConfidence: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "health_trend_confidence",
			Help: "Confidence levels for health trends",
		}, []string{"component", "trend"}),

		BusinessImpactScore: prometheus.NewGaugeVec(prometheus.GaugeOpts{
			Name: "business_impact_score",
			Help: "Business impact scores by component",
		}, []string{"component", "impact_type"}),

		ReportGenerationTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "health_report_generation_time_seconds",
			Help:    "Time taken to generate health reports",
			Buckets: prometheus.ExponentialBuckets(0.1, 2, 8),
		}),
	}
}

// AggregateHealth performs comprehensive health aggregation.
func (ha *HealthAggregator) AggregateHealth(ctx context.Context) (*AggregatedHealthResult, error) {
	start := time.Now()

	// Create result structure.
	result := &AggregatedHealthResult{
		Timestamp:       start,
		TierScores:      make(map[HealthTier]TierHealthResult),
		ComponentScores: make(map[string]ComponentHealthResult),
		JourneyResults:  make(map[string]*JourneyHealthResult),
		AggregationMetadata: AggregationMetadata{
			AggregationMethod:    ha.aggregationConfig.DefaultMethod,
			WeightingEnabled:     ha.aggregationConfig.BusinessWeightEnabled,
			TrendAnalysisEnabled: ha.aggregationConfig.TrendAnalysisEnabled,
			DataSources:          []string{"enhanced_checker", "dependency_tracker"},
		},
	}

	// Collect health data from all sources.
	enhancedHealthData := ha.enhancedChecker.CheckEnhanced(ctx)
	dependencyHealthData := ha.dependencyTracker.GetAllDependencyHealth(ctx)

	// Update historical data.
	ha.updateHistoricalData(enhancedHealthData, dependencyHealthData)

	// Aggregate by tiers.
	ha.aggregateByTiers(enhancedHealthData, result)

	// Aggregate by components.
	ha.aggregateByComponents(enhancedHealthData, dependencyHealthData, result)

	// Calculate multi-dimensional scores.
	ha.calculateMultiDimensionalScores(result)

	// Evaluate user journeys if enabled.
	if ha.aggregationConfig.UserJourneyEnabled {
		ha.evaluateUserJourneys(ctx, result)
	}

	// Perform trend analysis if enabled.
	if ha.aggregationConfig.TrendAnalysisEnabled {
		result.TrendAnalysis = ha.performTrendAnalysis(ctx)
	}

	// Assess business impact.
	result.BusinessImpact = ha.assessBusinessImpact(result)

	// Generate recommendations.
	result.Recommendations = ha.generateRecommendations(result)

	// Determine overall status and score.
	ha.calculateOverallHealth(result)

	// Record execution metadata.
	result.AggregationMetadata.ExecutionTime = time.Since(start)
	result.AggregationMetadata.DataPointsProcessed = len(enhancedHealthData.Checks) + len(dependencyHealthData)

	// Record metrics.
	ha.recordAggregationMetrics(result)

	ha.logger.Debug("Health aggregation completed",
		"duration", result.AggregationMetadata.ExecutionTime,
		"overall_score", result.OverallScore,
		"overall_status", result.OverallStatus,
		"data_points", result.AggregationMetadata.DataPointsProcessed)

	return result, nil
}

// aggregateByTiers aggregates health data by tiers.
func (ha *HealthAggregator) aggregateByTiers(enhancedData *EnhancedHealthResponse, result *AggregatedHealthResult) {
	tierData := make(map[HealthTier][]float64)
	tierCounts := make(map[HealthTier]map[health.Status]int)

	// Initialize tier counts.
	for tier := TierSystem; tier <= TierDependency; tier++ {
		tierCounts[tier] = make(map[health.Status]int)
	}

	// Process enhanced check results.
	for _, check := range enhancedData.Checks {
		tierData[check.Tier] = append(tierData[check.Tier], check.Score)
		tierCounts[check.Tier][check.Status]++
	}

	// Calculate tier results.
	for tier, scores := range tierData {
		if len(scores) == 0 {
			continue
		}

		method := ha.aggregationConfig.TierMethods[tier]
		if method == "" {
			method = ha.aggregationConfig.DefaultMethod
		}

		aggregatedScore := ha.aggregateScores(scores, method)
		overallStatus := ha.determineStatusFromCounts(tierCounts[tier])

		result.TierScores[tier] = TierHealthResult{
			Tier:           tier,
			Status:         overallStatus,
			Score:          aggregatedScore,
			ComponentCount: len(scores),
			HealthyCount:   tierCounts[tier][health.StatusHealthy],
			UnhealthyCount: tierCounts[tier][health.StatusUnhealthy],
			Weight:         ha.getTierWeight(tier),
		}
	}
}

// aggregateByComponents aggregates health data by components.
func (ha *HealthAggregator) aggregateByComponents(enhancedData *EnhancedHealthResponse, dependencyData map[string]*DependencyHealth, result *AggregatedHealthResult) {
	// Process enhanced checks.
	for _, check := range enhancedData.Checks {
		businessWeight := ha.getBusinessWeight(check.Name)

		result.ComponentScores[check.Name] = ComponentHealthResult{
			Name:           check.Name,
			Type:           check.Tier.String(),
			Status:         check.Status,
			Score:          check.Score,
			BusinessWeight: businessWeight,
			Dependencies:   check.Dependencies,
			LastUpdated:    check.Timestamp,
		}
	}

	// Process dependency health.
	for name, dependency := range dependencyData {
		businessWeight := ha.getBusinessWeight(name)
		score := ha.dependencyHealthToScore(dependency.Status)

		result.ComponentScores[name] = ComponentHealthResult{
			Name:           name,
			Type:           string(dependency.Type),
			Status:         dependency.Status,
			Score:          score,
			BusinessWeight: businessWeight,
			LastUpdated:    dependency.LastChecked,
		}
	}
}

// calculateMultiDimensionalScores calculates scores for different dimensions.
func (ha *HealthAggregator) calculateMultiDimensionalScores(result *AggregatedHealthResult) {
	var businessScores []float64
	var technicalScores []float64
	var performanceScores []float64
	var businessWeights []float64
	var technicalWeights []float64
	var performanceWeights []float64

	for _, component := range result.ComponentScores {
		// Business score based on business weight.
		businessScore := component.Score * component.BusinessWeight
		businessScores = append(businessScores, businessScore)
		businessWeights = append(businessWeights, component.BusinessWeight)

		// Technical score (unweighted component score).
		technicalScores = append(technicalScores, component.Score)
		technicalWeights = append(technicalWeights, 1.0)

		// Performance score (could be enhanced with latency data).
		performanceScore := component.Score
		performanceScores = append(performanceScores, performanceScore)
		performanceWeights = append(performanceWeights, 1.0)
	}

	// Calculate weighted averages.
	result.BusinessScore = ha.calculateWeightedAverage(businessScores, businessWeights)
	result.TechnicalScore = ha.calculateWeightedAverage(technicalScores, technicalWeights)
	result.PerformanceScore = ha.calculateWeightedAverage(performanceScores, performanceWeights)

	// User experience score is a combination of business and performance.
	result.UserExperienceScore = (result.BusinessScore*0.6 + result.PerformanceScore*0.4)
}

// evaluateUserJourneys evaluates the health of user journeys.
func (ha *HealthAggregator) evaluateUserJourneys(ctx context.Context, result *AggregatedHealthResult) {
	ha.journeyMu.Lock()
	defer ha.journeyMu.Unlock()

	for journeyID, journey := range ha.userJourneys {
		journeyResult := ha.evaluateUserJourney(ctx, journey, result.ComponentScores)
		result.JourneyResults[journeyID] = journeyResult
		ha.journeyResults[journeyID] = journeyResult
	}
}

// evaluateUserJourney evaluates a single user journey.
func (ha *HealthAggregator) evaluateUserJourney(ctx context.Context, journey *UserJourney, componentScores map[string]ComponentHealthResult) *JourneyHealthResult {
	journeyResult := &JourneyHealthResult{
		JourneyID:   journey.ID,
		StepResults: make(map[string]*StepResult),
		LastChecked: time.Now(),
	}

	totalLatency := time.Duration(0)
	totalScore := 0.0
	totalWeight := 0.0
	allStepsHealthy := true
	bottleneckLatency := time.Duration(0)
	bottleneckStep := ""

	for _, step := range journey.Steps {
		stepResult := &StepResult{
			StepID:            step.ID,
			Status:            health.StatusHealthy,
			Score:             1.0,
			DependencyResults: make(map[string]health.Status),
		}

		stepHealthy := true
		stepScore := 1.0

		// Check dependencies for this step.
		for _, dependency := range step.Dependencies {
			if componentResult, exists := componentScores[dependency]; exists {
				stepResult.DependencyResults[dependency] = componentResult.Status
				if componentResult.Status != health.StatusHealthy {
					stepHealthy = false
					stepScore *= componentResult.Score
				}
			} else {
				stepResult.DependencyResults[dependency] = health.StatusUnknown
				stepHealthy = false
				stepScore *= 0.5 // Unknown dependencies get penalty
			}
		}

		// Simulate latency (in real implementation, this would come from actual measurements).
		stepLatency := time.Duration(len(step.Dependencies)) * 10 * time.Millisecond
		stepResult.Latency = stepLatency
		totalLatency += stepLatency

		// Check if step is bottleneck.
		if stepLatency > bottleneckLatency {
			bottleneckLatency = stepLatency
			bottleneckStep = step.ID
		}

		// Check latency threshold.
		if stepLatency > step.MaxLatency {
			stepHealthy = false
			stepScore *= 0.8 // Latency penalty
		}

		// Set step status based on health.
		if stepHealthy {
			stepResult.Status = health.StatusHealthy
		} else if stepScore > 0.5 {
			stepResult.Status = health.StatusDegraded
		} else {
			stepResult.Status = health.StatusUnhealthy
		}

		stepResult.Score = stepScore
		journeyResult.StepResults[step.ID] = stepResult

		// Update journey totals.
		totalScore += stepScore * step.Weight
		totalWeight += step.Weight

		if !stepHealthy && step.Required {
			allStepsHealthy = false
		}
	}

	// Calculate overall journey health.
	if totalWeight > 0 {
		journeyResult.OverallScore = totalScore / totalWeight
	} else {
		journeyResult.OverallScore = 0
	}

	if allStepsHealthy {
		journeyResult.Status = health.StatusHealthy
	} else if journeyResult.OverallScore > 0.7 {
		journeyResult.Status = health.StatusDegraded
	} else {
		journeyResult.Status = health.StatusUnhealthy
	}

	journeyResult.TotalLatency = totalLatency
	journeyResult.BottleneckStep = bottleneckStep

	// Calculate availability (simplified).
	journeyResult.Availability = journeyResult.OverallScore * 100

	// Assess user impact.
	journeyResult.UserImpact = UserImpactAssessment{
		AffectedUsers: ha.estimateAffectedUsers(journey, journeyResult.Status),
		ImpactLevel:   ha.assessUserImpactLevel(journey, journeyResult.Status),
		Features:      ha.getAffectedFeatures(journey),
	}

	return journeyResult
}

// performTrendAnalysis performs trend analysis on health data.
func (ha *HealthAggregator) performTrendAnalysis(ctx context.Context) map[string]*HealthTrendResult {
	ha.historyMu.RLock()
	defer ha.historyMu.RUnlock()

	results := make(map[string]*HealthTrendResult)

	for component, history := range ha.historicalData {
		if len(history) < ha.trendAnalyzer.minDataPoints {
			continue
		}

		trendResult := ha.trendAnalyzer.analyzeTrend(component, history)
		results[component] = trendResult
	}

	return results
}

// analyzeTrend analyzes the trend for a component's health data.
func (ta *TrendAnalyzer) analyzeTrend(component string, history []HealthDataPoint) *HealthTrendResult {
	if len(history) < ta.minDataPoints {
		return &HealthTrendResult{
			Component:  component,
			Trend:      TrendUnknown,
			Confidence: 0.0,
		}
	}

	// Calculate linear regression for trend.
	slope, confidence := ta.calculateLinearRegression(history)

	// Determine trend direction.
	trend := TrendStable
	if math.Abs(slope) > 0.01 { // Threshold for significant trend
		if slope > 0 {
			trend = TrendImproving
		} else {
			trend = TrendDegrading
		}
	}

	// Calculate volatility.
	volatility := ta.calculateVolatility(history)
	if volatility > 0.2 {
		trend = TrendVolatile
	}

	result := &HealthTrendResult{
		Component:  component,
		Trend:      trend,
		Confidence: confidence,
		Slope:      slope,
		Volatility: volatility,
	}

	// Generate predictions if confidence is high enough.
	if confidence >= ta.confidenceThreshold {
		result.Predictions = ta.generatePredictions(history, slope, 4) // 4 predictions
	}

	// Generate recommendations based on trend.
	result.Recommendations = ta.generateTrendRecommendations(result)

	return result
}

// calculateLinearRegression calculates linear regression slope and confidence.
func (ta *TrendAnalyzer) calculateLinearRegression(history []HealthDataPoint) (float64, float64) {
	n := float64(len(history))
	if n < 2 {
		return 0, 0
	}

	// Convert timestamps to numeric values (hours since first point).
	var x, y []float64
	firstTime := history[0].Timestamp

	for _, point := range history {
		hours := point.Timestamp.Sub(firstTime).Hours()
		x = append(x, hours)
		y = append(y, point.Score)
	}

	// Calculate means.
	var sumX, sumY float64
	for i := range x {
		sumX += x[i]
		sumY += y[i]
	}
	meanX := sumX / n
	meanY := sumY / n

	// Calculate slope and correlation.
	var numerator, denominatorX, denominatorY float64
	for i := range x {
		dx := x[i] - meanX
		dy := y[i] - meanY
		numerator += dx * dy
		denominatorX += dx * dx
		denominatorY += dy * dy
	}

	if denominatorX == 0 {
		return 0, 0
	}

	slope := numerator / denominatorX

	// Calculate correlation coefficient (confidence measure).
	correlation := numerator / math.Sqrt(denominatorX*denominatorY)
	confidence := math.Abs(correlation)

	return slope, confidence
}

// calculateVolatility calculates the volatility of health scores.
func (ta *TrendAnalyzer) calculateVolatility(history []HealthDataPoint) float64 {
	if len(history) < 2 {
		return 0
	}

	// Calculate standard deviation.
	var sum, sumSquares float64
	n := float64(len(history))

	for _, point := range history {
		sum += point.Score
		sumSquares += point.Score * point.Score
	}

	mean := sum / n
	variance := (sumSquares / n) - (mean * mean)

	return math.Sqrt(variance)
}

// generatePredictions generates future health predictions.
func (ta *TrendAnalyzer) generatePredictions(history []HealthDataPoint, slope float64, count int) []HealthPrediction {
	if len(history) == 0 {
		return nil
	}

	predictions := make([]HealthPrediction, count)
	lastPoint := history[len(history)-1]

	for i := 0; i < count; i++ {
		// Predict next hour.
		futureTime := lastPoint.Timestamp.Add(time.Duration(i+1) * time.Hour)
		predictedScore := lastPoint.Score + slope*float64(i+1)

		// Clamp score between 0 and 1.
		if predictedScore < 0 {
			predictedScore = 0
		} else if predictedScore > 1 {
			predictedScore = 1
		}

		// Convert score to status.
		var predictedStatus health.Status
		if predictedScore >= 0.9 {
			predictedStatus = health.StatusHealthy
		} else if predictedScore >= 0.6 {
			predictedStatus = health.StatusDegraded
		} else {
			predictedStatus = health.StatusUnhealthy
		}

		predictions[i] = HealthPrediction{
			PredictedTime:   futureTime,
			PredictedStatus: predictedStatus,
			PredictedScore:  predictedScore,
			Confidence:      math.Max(0.5-float64(i)*0.1, 0.1), // Decreasing confidence
		}
	}

	return predictions
}

// generateTrendRecommendations generates recommendations based on trend analysis.
func (ta *TrendAnalyzer) generateTrendRecommendations(result *HealthTrendResult) []string {
	var recommendations []string

	switch result.Trend {
	case TrendDegrading:
		recommendations = append(recommendations, fmt.Sprintf("Component %s is showing degrading trend - investigate root cause", result.Component))
		if result.Confidence > 0.8 {
			recommendations = append(recommendations, "High confidence degradation detected - immediate attention required")
		}
	case TrendVolatile:
		recommendations = append(recommendations, fmt.Sprintf("Component %s is showing volatile behavior - stabilization needed", result.Component))
	case TrendImproving:
		if result.Confidence > 0.7 {
			recommendations = append(recommendations, fmt.Sprintf("Component %s is improving - document recent changes for replication", result.Component))
		}
	}

	if result.Volatility > 0.3 {
		recommendations = append(recommendations, "High volatility detected - consider smoothing configurations")
	}

	return recommendations
}

// Helper methods.
func (ha *HealthAggregator) aggregateScores(scores []float64, method AggregationMethod) float64 {
	if len(scores) == 0 {
		return 0
	}

	switch method {
	case AggregationWeightedAverage:
		// For now, treat as simple average (weights would be applied separately).
		return ha.calculateAverage(scores)
	case AggregationMinimum:
		return ha.findMinimum(scores)
	case AggregationMaximum:
		return ha.findMaximum(scores)
	case AggregationMedian:
		return ha.calculateMedian(scores)
	default:
		return ha.calculateAverage(scores)
	}
}

func (ha *HealthAggregator) calculateAverage(scores []float64) float64 {
	sum := 0.0
	for _, score := range scores {
		sum += score
	}
	return sum / float64(len(scores))
}

func (ha *HealthAggregator) calculateWeightedAverage(scores, weights []float64) float64 {
	if len(scores) != len(weights) || len(scores) == 0 {
		return 0
	}

	var weightedSum, totalWeight float64
	for i, score := range scores {
		weightedSum += score * weights[i]
		totalWeight += weights[i]
	}

	if totalWeight == 0 {
		return 0
	}

	return weightedSum / totalWeight
}

func (ha *HealthAggregator) findMinimum(scores []float64) float64 {
	if len(scores) == 0 {
		return 0
	}

	min := scores[0]
	for _, score := range scores[1:] {
		if score < min {
			min = score
		}
	}
	return min
}

func (ha *HealthAggregator) findMaximum(scores []float64) float64 {
	if len(scores) == 0 {
		return 0
	}

	max := scores[0]
	for _, score := range scores[1:] {
		if score > max {
			max = score
		}
	}
	return max
}

func (ha *HealthAggregator) calculateMedian(scores []float64) float64 {
	if len(scores) == 0 {
		return 0
	}

	sorted := make([]float64, len(scores))
	copy(sorted, scores)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func (ha *HealthAggregator) determineStatusFromCounts(counts map[health.Status]int) health.Status {
	total := 0
	for _, count := range counts {
		total += count
	}

	if total == 0 {
		return health.StatusUnknown
	}

	// If more than 50% unhealthy, overall is unhealthy.
	if counts[health.StatusUnhealthy] > total/2 {
		return health.StatusUnhealthy
	}

	// If any unhealthy or more than 25% degraded, overall is degraded.
	if counts[health.StatusUnhealthy] > 0 || counts[health.StatusDegraded] > total/4 {
		return health.StatusDegraded
	}

	// If most are healthy, overall is healthy.
	if counts[health.StatusHealthy] > total*3/4 {
		return health.StatusHealthy
	}

	return health.StatusDegraded
}

func (ha *HealthAggregator) getTierWeight(tier HealthTier) float64 {
	switch tier {
	case TierSystem:
		return 1.0
	case TierService:
		return 0.8
	case TierComponent:
		return 0.6
	case TierDependency:
		return 0.4
	default:
		return 0.5
	}
}

func (ha *HealthAggregator) getBusinessWeight(componentName string) float64 {
	if weight, exists := ha.businessWeights[componentName]; exists {
		return weight.Weight
	}
	return 1.0 // Default weight
}

func (ha *HealthAggregator) dependencyHealthToScore(status health.Status) float64 {
	switch status {
	case health.StatusHealthy:
		return 1.0
	case health.StatusDegraded:
		return 0.6
	case health.StatusUnhealthy:
		return 0.0
	default:
		return 0.3
	}
}

// updateHistoricalData updates historical health data for trend analysis.
func (ha *HealthAggregator) updateHistoricalData(enhancedData *EnhancedHealthResponse, dependencyData map[string]*DependencyHealth) {
	ha.historyMu.Lock()
	defer ha.historyMu.Unlock()

	now := time.Now()

	// Update enhanced check history.
	for _, check := range enhancedData.Checks {
		dataPoint := HealthDataPoint{
			Timestamp: now,
			Status:    check.Status,
			Score:     check.Score,
			Duration:  check.Duration,
		}

		ha.historicalData[check.Name] = append(ha.historicalData[check.Name], dataPoint)

		// Keep only last 100 points.
		if len(ha.historicalData[check.Name]) > 100 {
			ha.historicalData[check.Name] = ha.historicalData[check.Name][len(ha.historicalData[check.Name])-100:]
		}
	}

	// Update dependency history.
	for name, dependency := range dependencyData {
		score := ha.dependencyHealthToScore(dependency.Status)
		dataPoint := HealthDataPoint{
			Timestamp: now,
			Status:    dependency.Status,
			Score:     score,
			Duration:  dependency.ResponseTime,
		}

		ha.historicalData[name] = append(ha.historicalData[name], dataPoint)

		// Keep only last 100 points.
		if len(ha.historicalData[name]) > 100 {
			ha.historicalData[name] = ha.historicalData[name][len(ha.historicalData[name])-100:]
		}
	}
}

// assessBusinessImpact assesses the business impact of current health status.
func (ha *HealthAggregator) assessBusinessImpact(result *AggregatedHealthResult) *BusinessImpactResult {
	impact := &BusinessImpactResult{
		Level:                BusinessImpactNone,
		EstimatedRevenueLoss: 0,
		AffectedCustomers:    0,
		AffectedFeatures:     []string{},
		ComplianceRisks:      []string{},
		ReputationImpact:     ReputationNone,
	}

	// Assess based on overall score.
	if result.OverallScore < 0.5 {
		impact.Level = BusinessImpactCritical
		impact.EstimatedRevenueLoss = 10000.0 // Example values
		impact.AffectedCustomers = 1000
		impact.ReputationImpact = ReputationSevere
	} else if result.OverallScore < 0.7 {
		impact.Level = BusinessImpactHigh
		impact.EstimatedRevenueLoss = 5000.0
		impact.AffectedCustomers = 500
		impact.ReputationImpact = ReputationMajor
	} else if result.OverallScore < 0.9 {
		impact.Level = BusinessImpactMedium
		impact.EstimatedRevenueLoss = 1000.0
		impact.AffectedCustomers = 100
		impact.ReputationImpact = ReputationModerate
	}

	// Identify affected features based on component health.
	for name, component := range result.ComponentScores {
		if component.Status != health.StatusHealthy {
			if feature, exists := ha.featureWeights[name]; exists {
				impact.AffectedFeatures = append(impact.AffectedFeatures, feature.Feature)
			}
		}
	}

	return impact
}

// generateRecommendations generates actionable recommendations based on health analysis.
func (ha *HealthAggregator) generateRecommendations(result *AggregatedHealthResult) []HealthRecommendation {
	var recommendations []HealthRecommendation

	// Generate recommendations based on unhealthy components.
	for name, component := range result.ComponentScores {
		if component.Status != health.StatusHealthy {
			recommendation := HealthRecommendation{
				ID:                 fmt.Sprintf("rec-%s-%d", name, time.Now().Unix()),
				Type:               RecommendationRecovery,
				Priority:           ha.determinePriority(component),
				Title:              fmt.Sprintf("Address issues with %s", name),
				Description:        fmt.Sprintf("Component %s is %s with score %.2f", name, component.Status, component.Score),
				AffectedComponents: []string{name},
				Actions: []RecommendedAction{
					{
						Action:      fmt.Sprintf("investigate_%s", name),
						Priority:    ActionPriorityUrgent,
						Description: fmt.Sprintf("Investigate root cause of %s issues", name),
						Automated:   false,
						ETA:         15 * time.Minute,
					},
				},
				EstimatedImpact: RecommendationImpact{
					HealthImprovement:  1.0 - component.Score,
					ImplementationTime: 30 * time.Minute,
				},
				Confidence:      0.8,
				AutomationLevel: AutomationPartial,
			}
			recommendations = append(recommendations, recommendation)
		}
	}

	// Generate recommendations based on business impact.
	if result.BusinessImpact.Level == BusinessImpactCritical {
		recommendation := HealthRecommendation{
			ID:          fmt.Sprintf("business-critical-%d", time.Now().Unix()),
			Type:        RecommendationRecovery,
			Priority:    PriorityEmergency,
			Title:       "Critical business impact detected",
			Description: "Multiple components are affecting business operations",
			Actions: []RecommendedAction{
				{
					Action:      "activate_incident_response",
					Priority:    ActionPriorityImmediate,
					Description: "Activate incident response procedures",
					Automated:   true,
					ETA:         5 * time.Minute,
				},
			},
			EstimatedImpact: RecommendationImpact{
				HealthImprovement:  0.5,
				CostSavings:        result.BusinessImpact.EstimatedRevenueLoss * 0.8,
				ImplementationTime: time.Hour,
			},
			Confidence:      0.9,
			AutomationLevel: AutomationFull,
		}
		recommendations = append(recommendations, recommendation)
	}

	return recommendations
}

// calculateOverallHealth calculates the overall health status and score.
func (ha *HealthAggregator) calculateOverallHealth(result *AggregatedHealthResult) {
	// Weighted combination of different dimensions.
	weights := map[string]float64{
		"business":        0.4,
		"technical":       0.3,
		"performance":     0.2,
		"user_experience": 0.1,
	}

	result.OverallScore = result.BusinessScore*weights["business"] +
		result.TechnicalScore*weights["technical"] +
		result.PerformanceScore*weights["performance"] +
		result.UserExperienceScore*weights["user_experience"]

	// Determine overall status.
	if result.OverallScore >= 0.95 {
		result.OverallStatus = health.StatusHealthy
	} else if result.OverallScore >= 0.7 {
		result.OverallStatus = health.StatusDegraded
	} else {
		result.OverallStatus = health.StatusUnhealthy
	}
}

// recordAggregationMetrics records metrics for aggregation results.
func (ha *HealthAggregator) recordAggregationMetrics(result *AggregatedHealthResult) {
	// Record aggregation latency.
	ha.aggregatorMetrics.AggregationLatency.Observe(result.AggregationMetadata.ExecutionTime.Seconds())

	// Record aggregated health scores.
	ha.aggregatorMetrics.AggregatedHealthScore.WithLabelValues("overall", "all").Set(result.OverallScore)
	ha.aggregatorMetrics.AggregatedHealthScore.WithLabelValues("business", "all").Set(result.BusinessScore)
	ha.aggregatorMetrics.AggregatedHealthScore.WithLabelValues("technical", "all").Set(result.TechnicalScore)
	ha.aggregatorMetrics.AggregatedHealthScore.WithLabelValues("performance", "all").Set(result.PerformanceScore)
	ha.aggregatorMetrics.AggregatedHealthScore.WithLabelValues("user_experience", "all").Set(result.UserExperienceScore)

	// Record tier scores.
	for tier, tierResult := range result.TierScores {
		ha.aggregatorMetrics.AggregatedHealthScore.WithLabelValues("tier", tier.String()).Set(tierResult.Score)
	}

	// Record journey scores.
	for journeyID, journey := range result.JourneyResults {
		ha.aggregatorMetrics.JourneyHealthScore.WithLabelValues(journeyID, journey.JourneyID).Set(journey.OverallScore)
	}

	// Record business impact.
	ha.aggregatorMetrics.BusinessImpactScore.WithLabelValues("overall", "revenue").Set(result.BusinessImpact.EstimatedRevenueLoss)
	ha.aggregatorMetrics.BusinessImpactScore.WithLabelValues("overall", "customers").Set(float64(result.BusinessImpact.AffectedCustomers))
}

// configureDefaultWeights configures default business and feature weights.
func (ha *HealthAggregator) configureDefaultWeights() {
	// Business weights for core components.
	ha.businessWeights["llm-processor"] = BusinessWeight{
		Component:        "llm-processor",
		Weight:           1.0,
		RevenueImpact:    0.8,
		UserImpact:       0.9,
		ComplianceImpact: 0.6,
	}

	ha.businessWeights["kubernetes-api"] = BusinessWeight{
		Component:        "kubernetes-api",
		Weight:           0.9,
		RevenueImpact:    0.7,
		UserImpact:       0.8,
		ComplianceImpact: 0.9,
	}

	ha.businessWeights["rag-api"] = BusinessWeight{
		Component:        "rag-api",
		Weight:           0.8,
		RevenueImpact:    0.6,
		UserImpact:       0.7,
		ComplianceImpact: 0.4,
	}

	// Feature weights.
	ha.featureWeights["intent_processing"] = FeatureWeight{
		Feature:          "intent_processing",
		Weight:           1.0,
		UserBase:         1000,
		CriticalityLevel: "critical",
	}

	ha.featureWeights["deployment"] = FeatureWeight{
		Feature:          "deployment",
		Weight:           0.9,
		UserBase:         500,
		CriticalityLevel: "high",
	}
}

// configureDefaultUserJourneys configures default user journeys for monitoring.
func (ha *HealthAggregator) configureDefaultUserJourneys() {
	// Intent processing journey.
	intentJourney := &UserJourney{
		ID:             "intent_processing",
		Name:           "Intent Processing Journey",
		Description:    "Complete flow from intent submission to deployment",
		Timeout:        5 * time.Minute,
		CriticalPath:   true,
		BusinessImpact: BusinessImpactHigh,
		UserSegment:    "all_users",
		Steps: []JourneyStep{
			{
				ID:           "intent_validation",
				Name:         "Intent Validation",
				Dependencies: []string{"kubernetes-api"},
				MaxLatency:   time.Second,
				Required:     true,
				Weight:       0.2,
			},
			{
				ID:           "llm_processing",
				Name:         "LLM Processing",
				Dependencies: []string{"llm-processor", "rag-api"},
				MaxLatency:   10 * time.Second,
				Required:     true,
				Weight:       0.5,
			},
			{
				ID:           "resource_deployment",
				Name:         "Resource Deployment",
				Dependencies: []string{"kubernetes-api"},
				MaxLatency:   30 * time.Second,
				Required:     true,
				Weight:       0.3,
			},
		},
	}

	ha.userJourneys["intent_processing"] = intentJourney
}

// Helper methods for user journey evaluation.
func (ha *HealthAggregator) estimateAffectedUsers(journey *UserJourney, status health.Status) int {
	baseUsers := 1000 // Default user base

	if feature, exists := ha.featureWeights[journey.ID]; exists {
		baseUsers = feature.UserBase
	}

	switch status {
	case health.StatusUnhealthy:
		return baseUsers // All users affected
	case health.StatusDegraded:
		return baseUsers / 2 // Half users affected
	default:
		return 0 // No users affected
	}
}

func (ha *HealthAggregator) assessUserImpactLevel(journey *UserJourney, status health.Status) UserImpactLevel {
	if !journey.CriticalPath {
		if status == health.StatusUnhealthy {
			return UserImpactMinor
		}
		return UserImpactNone
	}

	switch status {
	case health.StatusUnhealthy:
		return UserImpactSevere
	case health.StatusDegraded:
		return UserImpactModerate
	default:
		return UserImpactNone
	}
}

func (ha *HealthAggregator) getAffectedFeatures(journey *UserJourney) []string {
	features := []string{}

	// Map journey to features (simplified).
	switch journey.ID {
	case "intent_processing":
		features = []string{"intent_processing", "deployment", "llm_integration"}
	}

	return features
}

func (ha *HealthAggregator) determinePriority(component ComponentHealthResult) RecommendationPriority {
	if component.BusinessWeight > 0.8 {
		if component.Status == health.StatusUnhealthy {
			return PriorityEmergency
		}
		return PriorityHigh
	} else if component.BusinessWeight > 0.5 {
		return PriorityMedium
	}
	return PriorityLow
}

// GetCheckHistory returns historical health data for a component.
func (ha *HealthAggregator) GetCheckHistory(component string, limit int) []HealthDataPoint {
	ha.historyMu.RLock()
	defer ha.historyMu.RUnlock()

	history, exists := ha.historicalData[component]
	if !exists {
		return nil
	}

	if limit > 0 && len(history) > limit {
		return history[len(history)-limit:]
	}

	return history
}
