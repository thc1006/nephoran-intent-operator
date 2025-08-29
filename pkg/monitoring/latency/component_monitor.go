
package latency



import (

	"context"

	"fmt"

	"sync"

	"sync/atomic"

	"time"



	"github.com/prometheus/client_golang/prometheus"

)



// ComponentMonitor provides detailed latency monitoring for individual system components.

type ComponentMonitor struct {

	mu sync.RWMutex



	// Component monitors.

	controllerMonitor *ControllerLatencyMonitor

	llmMonitor        *LLMProcessorMonitor

	ragMonitor        *RAGSystemMonitor

	gitopsMonitor     *GitOpsMonitor

	databaseMonitor   *DatabaseOperationMonitor

	cacheMonitor      *CachePerformanceMonitor

	queueMonitor      *QueueProcessingMonitor



	// Aggregated metrics.

	aggregator *ComponentAggregator



	// Performance tracking.

	performanceTracker *ComponentPerformanceTracker



	// Metrics.

	metrics *ComponentMetrics



	// Configuration.

	config *ComponentMonitorConfig

}



// ComponentMonitorConfig contains configuration for component monitoring.

type ComponentMonitorConfig struct {

	// Monitoring intervals.

	SamplingInterval    time.Duration `json:"sampling_interval"`

	AggregationInterval time.Duration `json:"aggregation_interval"`



	// Component-specific settings.

	ControllerSettings ControllerMonitorSettings `json:"controller_settings"`

	LLMSettings        LLMMonitorSettings        `json:"llm_settings"`

	RAGSettings        RAGMonitorSettings        `json:"rag_settings"`

	GitOpsSettings     GitOpsMonitorSettings     `json:"gitops_settings"`

	DatabaseSettings   DatabaseMonitorSettings   `json:"database_settings"`

	CacheSettings      CacheMonitorSettings      `json:"cache_settings"`

	QueueSettings      QueueMonitorSettings      `json:"queue_settings"`



	// Performance thresholds.

	PerformanceThresholds map[string]time.Duration `json:"performance_thresholds"`

}



// ControllerLatencyMonitor monitors Kubernetes controller latency.

type ControllerLatencyMonitor struct {

	mu sync.RWMutex



	// Reconciliation metrics.

	reconcileLatency *LatencyTracker

	reconcileQueue   *QueueMetrics



	// API server interactions.

	apiLatency   *LatencyTracker

	apiCallCount atomic.Int64



	// Resource operations.

	createLatency *LatencyTracker

	updateLatency *LatencyTracker

	deleteLatency *LatencyTracker



	// Watch operations.

	watchLatency    *LatencyTracker

	eventProcessing *LatencyTracker



	// Controller-specific metrics.

	workQueueDepth   atomic.Int64

	workQueueLatency *LatencyTracker

	retryCount       atomic.Int64



	// Settings.

	settings ControllerMonitorSettings

}



// LLMProcessorMonitor monitors LLM processing latency.

type LLMProcessorMonitor struct {

	mu sync.RWMutex



	// Token processing.

	tokenGeneration *LatencyTracker

	tokenPerSecond  *RateTracker



	// API interactions.

	apiCallLatency *LatencyTracker

	apiRetries     atomic.Int64



	// Context handling.

	contextBuilding  *LatencyTracker

	contextSwitching *LatencyTracker



	// Model-specific metrics.

	modelLatency map[string]*LatencyTracker

	modelUsage   map[string]int64



	// Streaming metrics.

	streamInitLatency  *LatencyTracker

	streamChunkLatency *LatencyTracker



	// Cost tracking.

	tokenCount    atomic.Int64

	estimatedCost atomic.Uint64 // In cents



	// Settings.

	settings LLMMonitorSettings

}



// RAGSystemMonitor monitors RAG system latency.

type RAGSystemMonitor struct {

	mu sync.RWMutex



	// Vector search.

	vectorSearchLatency *LatencyTracker

	embeddingLatency    *LatencyTracker



	// Document operations.

	documentRetrieval *LatencyTracker

	documentRanking   *LatencyTracker



	// Context building.

	contextGeneration *LatencyTracker

	contextSize       atomic.Int64



	// Cache performance.

	cacheHitLatency  *LatencyTracker

	cacheMissLatency *LatencyTracker

	cacheHitRate     *RateTracker



	// Index operations.

	indexUpdateLatency *LatencyTracker

	indexSearchLatency *LatencyTracker



	// Settings.

	settings RAGMonitorSettings

}



// GitOpsMonitor monitors GitOps pipeline latency.

type GitOpsMonitor struct {

	mu sync.RWMutex



	// Git operations.

	cloneLatency  *LatencyTracker

	fetchLatency  *LatencyTracker

	pushLatency   *LatencyTracker

	commitLatency *LatencyTracker



	// Package operations.

	packageGeneration *LatencyTracker

	packageValidation *LatencyTracker



	// Deployment operations.

	deploymentLatency *LatencyTracker

	syncLatency       *LatencyTracker



	// Pipeline stages.

	stageLatencies map[string]*LatencyTracker



	// Settings.

	settings GitOpsMonitorSettings

}



// DatabaseOperationMonitor monitors database operation latency.

type DatabaseOperationMonitor struct {

	mu sync.RWMutex



	// Query operations.

	queryLatency map[string]*LatencyTracker // By query type

	queryCount   atomic.Int64



	// Connection management.

	connectionWait *LatencyTracker

	connectionPool *PoolMetrics



	// Transaction metrics.

	transactionLatency *LatencyTracker

	lockWaitTime       *LatencyTracker



	// Slow query tracking.

	slowQueries *SlowQueryTracker



	// Settings.

	settings DatabaseMonitorSettings

}



// CachePerformanceMonitor monitors cache performance.

type CachePerformanceMonitor struct {

	mu sync.RWMutex



	// Cache operations.

	getLatency    *LatencyTracker

	setLatency    *LatencyTracker

	deleteLatency *LatencyTracker



	// Hit/Miss tracking.

	hitCount  atomic.Int64

	missCount atomic.Int64

	hitRate   *RateTracker



	// Eviction metrics.

	evictionCount   atomic.Int64

	evictionLatency *LatencyTracker



	// Size metrics.

	cacheSize   atomic.Int64

	memoryUsage atomic.Int64



	// Settings.

	settings CacheMonitorSettings

}



// QueueProcessingMonitor monitors message queue latency.

type QueueProcessingMonitor struct {

	mu sync.RWMutex



	// Queue metrics by queue name.

	queues map[string]*QueueMetrics



	// Processing latency.

	enqueueLatency    *LatencyTracker

	dequeueLatency    *LatencyTracker

	processingLatency *LatencyTracker



	// Queue depth.

	totalDepth atomic.Int64



	// Settings.

	settings QueueMonitorSettings

}



// Supporting types.



// LatencyTracker represents a latencytracker.

type LatencyTracker struct {

	mu           sync.RWMutex

	samples      []LatencySample

	maxSamples   int

	currentP50   time.Duration

	currentP95   time.Duration

	currentP99   time.Duration

	totalCount   int64

	totalLatency int64

}



// LatencySample represents a latencysample.

type LatencySample struct {

	Timestamp time.Time

	Duration  time.Duration

	Tags      map[string]string

}



// RateTracker represents a ratetracker.

type RateTracker struct {

	mu          sync.RWMutex

	window      time.Duration

	buckets     map[time.Time]int64

	currentRate float64

}



// QueueMetrics represents a queuemetrics.

type QueueMetrics struct {

	Depth       atomic.Int64

	EnqueueRate *RateTracker

	DequeueRate *RateTracker

	ProcessTime *LatencyTracker

	WaitTime    *LatencyTracker

}



// PoolMetrics represents a poolmetrics.

type PoolMetrics struct {

	ActiveConnections atomic.Int64

	IdleConnections   atomic.Int64

	TotalConnections  atomic.Int64

	WaitCount         atomic.Int64

	WaitTime          *LatencyTracker

}



// SlowQueryTracker represents a slowquerytracker.

type SlowQueryTracker struct {

	mu         sync.RWMutex

	queries    []SlowQuery

	threshold  time.Duration

	maxQueries int

}



// SlowQuery represents a slowquery.

type SlowQuery struct {

	Query      string

	Duration   time.Duration

	Timestamp  time.Time

	Parameters map[string]interface{}

}



// ComponentAggregator represents a componentaggregator.

type ComponentAggregator struct {

	mu              sync.RWMutex

	aggregates      map[string]*ComponentAggregate

	lastAggregation time.Time

}



// ComponentAggregate represents a componentaggregate.

type ComponentAggregate struct {

	Component    string

	TotalLatency time.Duration

	Count        int64

	P50          time.Duration

	P95          time.Duration

	P99          time.Duration

	MaxLatency   time.Duration

	MinLatency   time.Duration

}



// ComponentPerformanceTracker represents a componentperformancetracker.

type ComponentPerformanceTracker struct {

	mu          sync.RWMutex

	performance map[string]*PerformanceMetrics

	trends      map[string]*PerformanceTrend

}



// PerformanceMetrics represents a performancemetrics.

type PerformanceMetrics struct {

	Component       string

	CurrentLatency  time.Duration

	BaselineLatency time.Duration

	Deviation       float64

	TrendDirection  string // "improving", "degrading", "stable"

}



// PerformanceTrend represents a performancetrend.

type PerformanceTrend struct {

	Component  string

	DataPoints []TrendPoint

	Slope      float64

	Prediction time.Duration

}



// TrendPoint represents a trendpoint.

type TrendPoint struct {

	Timestamp time.Time

	Value     time.Duration

}



// ComponentMetrics represents a componentmetrics.

type ComponentMetrics struct {

	componentLatency    *prometheus.HistogramVec

	componentThroughput *prometheus.GaugeVec

	componentErrors     *prometheus.CounterVec

	queueDepth          *prometheus.GaugeVec

	cacheHitRate        prometheus.Gauge

}



// Monitor settings for each component.



// ControllerMonitorSettings represents a controllermonitorsettings.

type ControllerMonitorSettings struct {

	EnableReconcileTracking bool          `json:"enable_reconcile_tracking"`

	EnableAPITracking       bool          `json:"enable_api_tracking"`

	EnableQueueMetrics      bool          `json:"enable_queue_metrics"`

	MaxSamples              int           `json:"max_samples"`

	ReconcileTimeout        time.Duration `json:"reconcile_timeout"`

}



// LLMMonitorSettings represents a llmmonitorsettings.

type LLMMonitorSettings struct {

	EnableTokenTracking   bool          `json:"enable_token_tracking"`

	EnableCostTracking    bool          `json:"enable_cost_tracking"`

	EnableStreamMetrics   bool          `json:"enable_stream_metrics"`

	MaxSamples            int           `json:"max_samples"`

	SlowResponseThreshold time.Duration `json:"slow_response_threshold"`

}



// RAGMonitorSettings represents a ragmonitorsettings.

type RAGMonitorSettings struct {

	EnableVectorMetrics bool          `json:"enable_vector_metrics"`

	EnableCacheMetrics  bool          `json:"enable_cache_metrics"`

	EnableIndexMetrics  bool          `json:"enable_index_metrics"`

	MaxSamples          int           `json:"max_samples"`

	SlowSearchThreshold time.Duration `json:"slow_search_threshold"`

}



// GitOpsMonitorSettings represents a gitopsmonitorsettings.

type GitOpsMonitorSettings struct {

	EnableGitMetrics       bool          `json:"enable_git_metrics"`

	EnablePipelineMetrics  bool          `json:"enable_pipeline_metrics"`

	EnableDeployMetrics    bool          `json:"enable_deploy_metrics"`

	MaxSamples             int           `json:"max_samples"`

	SlowOperationThreshold time.Duration `json:"slow_operation_threshold"`

}



// DatabaseMonitorSettings represents a databasemonitorsettings.

type DatabaseMonitorSettings struct {

	EnableQueryTracking bool          `json:"enable_query_tracking"`

	EnablePoolMetrics   bool          `json:"enable_pool_metrics"`

	EnableSlowQueryLog  bool          `json:"enable_slow_query_log"`

	SlowQueryThreshold  time.Duration `json:"slow_query_threshold"`

	MaxSlowQueries      int           `json:"max_slow_queries"`

}



// CacheMonitorSettings represents a cachemonitorsettings.

type CacheMonitorSettings struct {

	EnableHitRateTracking bool `json:"enable_hit_rate_tracking"`

	EnableSizeMetrics     bool `json:"enable_size_metrics"`

	EnableEvictionMetrics bool `json:"enable_eviction_metrics"`

	MaxSamples            int  `json:"max_samples"`

}



// QueueMonitorSettings represents a queuemonitorsettings.

type QueueMonitorSettings struct {

	EnableDepthTracking   bool `json:"enable_depth_tracking"`

	EnableRateMetrics     bool `json:"enable_rate_metrics"`

	EnableLatencyTracking bool `json:"enable_latency_tracking"`

	MaxQueues             int  `json:"max_queues"`

}



// NewComponentMonitor creates a new component monitor.

func NewComponentMonitor(config *ComponentMonitorConfig) *ComponentMonitor {

	if config == nil {

		config = DefaultComponentMonitorConfig()

	}



	monitor := &ComponentMonitor{

		config:             config,

		aggregator:         NewComponentAggregator(),

		performanceTracker: NewComponentPerformanceTracker(),

	}



	// Initialize component monitors.

	monitor.controllerMonitor = NewControllerLatencyMonitor(config.ControllerSettings)

	monitor.llmMonitor = NewLLMProcessorMonitor(config.LLMSettings)

	monitor.ragMonitor = NewRAGSystemMonitor(config.RAGSettings)

	monitor.gitopsMonitor = NewGitOpsMonitor(config.GitOpsSettings)

	monitor.databaseMonitor = NewDatabaseOperationMonitor(config.DatabaseSettings)

	monitor.cacheMonitor = NewCachePerformanceMonitor(config.CacheSettings)

	monitor.queueMonitor = NewQueueProcessingMonitor(config.QueueSettings)



	// Initialize metrics.

	monitor.initMetrics()



	// Start background tasks.

	go monitor.runAggregation()



	return monitor

}



// RecordControllerLatency records controller operation latency.

func (m *ComponentMonitor) RecordControllerLatency(ctx context.Context, operation string, duration time.Duration) {

	m.controllerMonitor.RecordLatency(operation, duration)

	m.aggregator.Record("controller", operation, duration)

	m.performanceTracker.Update("controller", duration)

}



// RecordLLMLatency records LLM processing latency.

func (m *ComponentMonitor) RecordLLMLatency(ctx context.Context, model, operation string, duration time.Duration, tokens int) {

	m.llmMonitor.RecordLatency(model, operation, duration, tokens)

	m.aggregator.Record("llm_processor", operation, duration)

	m.performanceTracker.Update("llm_processor", duration)

}



// RecordRAGLatency records RAG system latency.

func (m *ComponentMonitor) RecordRAGLatency(ctx context.Context, operation string, duration time.Duration, cacheHit bool) {

	m.ragMonitor.RecordLatency(operation, duration, cacheHit)

	m.aggregator.Record("rag_system", operation, duration)

	m.performanceTracker.Update("rag_system", duration)

}



// RecordGitOpsLatency records GitOps operation latency.

func (m *ComponentMonitor) RecordGitOpsLatency(ctx context.Context, operation, stage string, duration time.Duration) {

	m.gitopsMonitor.RecordLatency(operation, stage, duration)

	m.aggregator.Record("gitops", operation, duration)

	m.performanceTracker.Update("gitops", duration)

}



// RecordDatabaseLatency records database operation latency.

func (m *ComponentMonitor) RecordDatabaseLatency(ctx context.Context, queryType string, duration time.Duration, rowsAffected int64) {

	m.databaseMonitor.RecordLatency(queryType, duration, rowsAffected)

	m.aggregator.Record("database", queryType, duration)

	m.performanceTracker.Update("database", duration)

}



// RecordCacheLatency records cache operation latency.

func (m *ComponentMonitor) RecordCacheLatency(ctx context.Context, operation string, duration time.Duration, hit bool) {

	m.cacheMonitor.RecordLatency(operation, duration, hit)

	m.aggregator.Record("cache", operation, duration)

	m.performanceTracker.Update("cache", duration)

}



// RecordQueueLatency records queue processing latency.

func (m *ComponentMonitor) RecordQueueLatency(ctx context.Context, queueName, operation string, duration time.Duration) {

	m.queueMonitor.RecordLatency(queueName, operation, duration)

	m.aggregator.Record(fmt.Sprintf("queue_%s", queueName), operation, duration)

	m.performanceTracker.Update(fmt.Sprintf("queue_%s", queueName), duration)

}



// GetComponentReport returns a comprehensive component latency report.

func (m *ComponentMonitor) GetComponentReport() *ComponentLatencyReport {

	report := &ComponentLatencyReport{

		Timestamp:  time.Now(),

		Components: make(map[string]*ComponentStats),

	}



	// Controller stats.

	report.Components["controller"] = m.controllerMonitor.GetStats()



	// LLM stats.

	report.Components["llm_processor"] = m.llmMonitor.GetStats()



	// RAG stats.

	report.Components["rag_system"] = m.ragMonitor.GetStats()



	// GitOps stats.

	report.Components["gitops"] = m.gitopsMonitor.GetStats()



	// Database stats.

	report.Components["database"] = m.databaseMonitor.GetStats()



	// Cache stats.

	report.Components["cache"] = m.cacheMonitor.GetStats()



	// Queue stats.

	report.Components["queue"] = m.queueMonitor.GetStats()



	// Add aggregated metrics.

	report.Aggregates = m.aggregator.GetAggregates()



	// Add performance trends.

	report.PerformanceTrends = m.performanceTracker.GetTrends()



	return report

}



// Component monitor implementations.



// NewControllerLatencyMonitor performs newcontrollerlatencymonitor operation.

func NewControllerLatencyMonitor(settings ControllerMonitorSettings) *ControllerLatencyMonitor {

	return &ControllerLatencyMonitor{

		reconcileLatency: NewLatencyTracker(settings.MaxSamples),

		reconcileQueue:   &QueueMetrics{},

		apiLatency:       NewLatencyTracker(settings.MaxSamples),

		createLatency:    NewLatencyTracker(settings.MaxSamples),

		updateLatency:    NewLatencyTracker(settings.MaxSamples),

		deleteLatency:    NewLatencyTracker(settings.MaxSamples),

		watchLatency:     NewLatencyTracker(settings.MaxSamples),

		eventProcessing:  NewLatencyTracker(settings.MaxSamples),

		workQueueLatency: NewLatencyTracker(settings.MaxSamples),

		settings:         settings,

	}

}



// RecordLatency performs recordlatency operation.

func (c *ControllerLatencyMonitor) RecordLatency(operation string, duration time.Duration) {

	switch operation {

	case "reconcile":

		c.reconcileLatency.Add(duration)

	case "api_call":

		c.apiLatency.Add(duration)

		c.apiCallCount.Add(1)

	case "create":

		c.createLatency.Add(duration)

	case "update":

		c.updateLatency.Add(duration)

	case "delete":

		c.deleteLatency.Add(duration)

	case "watch":

		c.watchLatency.Add(duration)

	case "event":

		c.eventProcessing.Add(duration)

	case "queue":

		c.workQueueLatency.Add(duration)

	}

}



// GetStats performs getstats operation.

func (c *ControllerLatencyMonitor) GetStats() *ComponentStats {

	return &ComponentStats{

		Component: "controller",

		Latencies: map[string]*LatencyStats{

			"reconcile":     c.reconcileLatency.GetStats(),

			"api_calls":     c.apiLatency.GetStats(),

			"create":        c.createLatency.GetStats(),

			"update":        c.updateLatency.GetStats(),

			"delete":        c.deleteLatency.GetStats(),

			"watch":         c.watchLatency.GetStats(),

			"event_process": c.eventProcessing.GetStats(),

			"work_queue":    c.workQueueLatency.GetStats(),

		},

		Counters: map[string]int64{

			"api_call_count":   c.apiCallCount.Load(),

			"work_queue_depth": c.workQueueDepth.Load(),

			"retry_count":      c.retryCount.Load(),

		},

	}

}



// NewLLMProcessorMonitor performs newllmprocessormonitor operation.

func NewLLMProcessorMonitor(settings LLMMonitorSettings) *LLMProcessorMonitor {

	return &LLMProcessorMonitor{

		tokenGeneration:    NewLatencyTracker(settings.MaxSamples),

		tokenPerSecond:     NewRateTracker(time.Minute),

		apiCallLatency:     NewLatencyTracker(settings.MaxSamples),

		contextBuilding:    NewLatencyTracker(settings.MaxSamples),

		contextSwitching:   NewLatencyTracker(settings.MaxSamples),

		modelLatency:       make(map[string]*LatencyTracker),

		modelUsage:         make(map[string]int64),

		streamInitLatency:  NewLatencyTracker(settings.MaxSamples),

		streamChunkLatency: NewLatencyTracker(settings.MaxSamples),

		settings:           settings,

	}

}



// RecordLatency performs recordlatency operation.

func (l *LLMProcessorMonitor) RecordLatency(model, operation string, duration time.Duration, tokens int) {

	l.mu.Lock()

	defer l.mu.Unlock()



	// Record model-specific latency.

	if _, exists := l.modelLatency[model]; !exists {

		l.modelLatency[model] = NewLatencyTracker(l.settings.MaxSamples)

	}

	l.modelLatency[model].Add(duration)

	l.modelUsage[model]++



	// Record operation-specific latency.

	switch operation {

	case "token_generation":

		l.tokenGeneration.Add(duration)

		if tokens > 0 && duration > 0 {

			tps := float64(tokens) / duration.Seconds()

			l.tokenPerSecond.Record(int64(tps))

		}

	case "api_call":

		l.apiCallLatency.Add(duration)

	case "context_build":

		l.contextBuilding.Add(duration)

	case "context_switch":

		l.contextSwitching.Add(duration)

	case "stream_init":

		l.streamInitLatency.Add(duration)

	case "stream_chunk":

		l.streamChunkLatency.Add(duration)

	}



	// Update token count and cost.

	if tokens > 0 {

		l.tokenCount.Add(int64(tokens))

		// Estimate cost (simplified - would use actual pricing).

		costCents := uint64(tokens) * 2 / 1000 // $0.02 per 1K tokens

		l.estimatedCost.Add(costCents)

	}

}



// GetStats performs getstats operation.

func (l *LLMProcessorMonitor) GetStats() *ComponentStats {

	l.mu.RLock()

	defer l.mu.RUnlock()



	stats := &ComponentStats{

		Component: "llm_processor",

		Latencies: map[string]*LatencyStats{

			"token_generation": l.tokenGeneration.GetStats(),

			"api_calls":        l.apiCallLatency.GetStats(),

			"context_build":    l.contextBuilding.GetStats(),

			"context_switch":   l.contextSwitching.GetStats(),

			"stream_init":      l.streamInitLatency.GetStats(),

			"stream_chunk":     l.streamChunkLatency.GetStats(),

		},

		Counters: map[string]int64{

			"total_tokens":         l.tokenCount.Load(),

			"api_retries":          l.apiRetries.Load(),

			"estimated_cost_cents": int64(l.estimatedCost.Load()),

		},

		ModelStats: make(map[string]*ModelStats),

	}



	// Add model-specific stats.

	for model, tracker := range l.modelLatency {

		stats.ModelStats[model] = &ModelStats{

			Model:   model,

			Usage:   l.modelUsage[model],

			Latency: tracker.GetStats(),

		}

	}



	return stats

}



// NewRAGSystemMonitor performs newragsystemmonitor operation.

func NewRAGSystemMonitor(settings RAGMonitorSettings) *RAGSystemMonitor {

	return &RAGSystemMonitor{

		vectorSearchLatency: NewLatencyTracker(settings.MaxSamples),

		embeddingLatency:    NewLatencyTracker(settings.MaxSamples),

		documentRetrieval:   NewLatencyTracker(settings.MaxSamples),

		documentRanking:     NewLatencyTracker(settings.MaxSamples),

		contextGeneration:   NewLatencyTracker(settings.MaxSamples),

		cacheHitLatency:     NewLatencyTracker(settings.MaxSamples),

		cacheMissLatency:    NewLatencyTracker(settings.MaxSamples),

		cacheHitRate:        NewRateTracker(time.Minute),

		indexUpdateLatency:  NewLatencyTracker(settings.MaxSamples),

		indexSearchLatency:  NewLatencyTracker(settings.MaxSamples),

		settings:            settings,

	}

}



// RecordLatency performs recordlatency operation.

func (r *RAGSystemMonitor) RecordLatency(operation string, duration time.Duration, cacheHit bool) {

	switch operation {

	case "vector_search":

		r.vectorSearchLatency.Add(duration)

	case "embedding":

		r.embeddingLatency.Add(duration)

	case "document_retrieval":

		r.documentRetrieval.Add(duration)

	case "document_ranking":

		r.documentRanking.Add(duration)

	case "context_generation":

		r.contextGeneration.Add(duration)

	case "index_update":

		r.indexUpdateLatency.Add(duration)

	case "index_search":

		r.indexSearchLatency.Add(duration)

	}



	// Record cache metrics.

	if cacheHit {

		r.cacheHitLatency.Add(duration)

		r.cacheHitRate.Record(1)

	} else {

		r.cacheMissLatency.Add(duration)

		r.cacheHitRate.Record(0)

	}

}



// GetStats performs getstats operation.

func (r *RAGSystemMonitor) GetStats() *ComponentStats {

	return &ComponentStats{

		Component: "rag_system",

		Latencies: map[string]*LatencyStats{

			"vector_search":      r.vectorSearchLatency.GetStats(),

			"embedding":          r.embeddingLatency.GetStats(),

			"document_retrieval": r.documentRetrieval.GetStats(),

			"document_ranking":   r.documentRanking.GetStats(),

			"context_generation": r.contextGeneration.GetStats(),

			"cache_hit":          r.cacheHitLatency.GetStats(),

			"cache_miss":         r.cacheMissLatency.GetStats(),

			"index_update":       r.indexUpdateLatency.GetStats(),

			"index_search":       r.indexSearchLatency.GetStats(),

		},

		Rates: map[string]float64{

			"cache_hit_rate": r.cacheHitRate.GetRate(),

		},

		Counters: map[string]int64{

			"context_size": r.contextSize.Load(),

		},

	}

}



// NewGitOpsMonitor performs newgitopsmonitor operation.

func NewGitOpsMonitor(settings GitOpsMonitorSettings) *GitOpsMonitor {

	return &GitOpsMonitor{

		cloneLatency:      NewLatencyTracker(settings.MaxSamples),

		fetchLatency:      NewLatencyTracker(settings.MaxSamples),

		pushLatency:       NewLatencyTracker(settings.MaxSamples),

		commitLatency:     NewLatencyTracker(settings.MaxSamples),

		packageGeneration: NewLatencyTracker(settings.MaxSamples),

		packageValidation: NewLatencyTracker(settings.MaxSamples),

		deploymentLatency: NewLatencyTracker(settings.MaxSamples),

		syncLatency:       NewLatencyTracker(settings.MaxSamples),

		stageLatencies:    make(map[string]*LatencyTracker),

		settings:          settings,

	}

}



// RecordLatency performs recordlatency operation.

func (g *GitOpsMonitor) RecordLatency(operation, stage string, duration time.Duration) {

	g.mu.Lock()

	defer g.mu.Unlock()



	switch operation {

	case "clone":

		g.cloneLatency.Add(duration)

	case "fetch":

		g.fetchLatency.Add(duration)

	case "push":

		g.pushLatency.Add(duration)

	case "commit":

		g.commitLatency.Add(duration)

	case "package_gen":

		g.packageGeneration.Add(duration)

	case "package_validate":

		g.packageValidation.Add(duration)

	case "deployment":

		g.deploymentLatency.Add(duration)

	case "sync":

		g.syncLatency.Add(duration)

	}



	// Record stage-specific latency.

	if stage != "" {

		if _, exists := g.stageLatencies[stage]; !exists {

			g.stageLatencies[stage] = NewLatencyTracker(g.settings.MaxSamples)

		}

		g.stageLatencies[stage].Add(duration)

	}

}



// GetStats performs getstats operation.

func (g *GitOpsMonitor) GetStats() *ComponentStats {

	g.mu.RLock()

	defer g.mu.RUnlock()



	stats := &ComponentStats{

		Component: "gitops",

		Latencies: map[string]*LatencyStats{

			"clone":              g.cloneLatency.GetStats(),

			"fetch":              g.fetchLatency.GetStats(),

			"push":               g.pushLatency.GetStats(),

			"commit":             g.commitLatency.GetStats(),

			"package_generation": g.packageGeneration.GetStats(),

			"package_validation": g.packageValidation.GetStats(),

			"deployment":         g.deploymentLatency.GetStats(),

			"sync":               g.syncLatency.GetStats(),

		},

		StageLatencies: make(map[string]*LatencyStats),

	}



	// Add stage-specific stats.

	for stage, tracker := range g.stageLatencies {

		stats.StageLatencies[stage] = tracker.GetStats()

	}



	return stats

}



// NewDatabaseOperationMonitor performs newdatabaseoperationmonitor operation.

func NewDatabaseOperationMonitor(settings DatabaseMonitorSettings) *DatabaseOperationMonitor {

	return &DatabaseOperationMonitor{

		queryLatency:       make(map[string]*LatencyTracker),

		connectionWait:     NewLatencyTracker(1000),

		connectionPool:     &PoolMetrics{},

		transactionLatency: NewLatencyTracker(settings.MaxSlowQueries),

		lockWaitTime:       NewLatencyTracker(settings.MaxSlowQueries),

		slowQueries:        NewSlowQueryTracker(settings.SlowQueryThreshold, settings.MaxSlowQueries),

		settings:           settings,

	}

}



// RecordLatency performs recordlatency operation.

func (d *DatabaseOperationMonitor) RecordLatency(queryType string, duration time.Duration, rowsAffected int64) {

	d.mu.Lock()

	defer d.mu.Unlock()



	// Record query-specific latency.

	if _, exists := d.queryLatency[queryType]; !exists {

		d.queryLatency[queryType] = NewLatencyTracker(1000)

	}

	d.queryLatency[queryType].Add(duration)

	d.queryCount.Add(1)



	// Check for slow query.

	if d.settings.EnableSlowQueryLog && duration > d.settings.SlowQueryThreshold {

		d.slowQueries.Add(SlowQuery{

			Query:     queryType,

			Duration:  duration,

			Timestamp: time.Now(),

		})

	}

}



// GetStats performs getstats operation.

func (d *DatabaseOperationMonitor) GetStats() *ComponentStats {

	d.mu.RLock()

	defer d.mu.RUnlock()



	stats := &ComponentStats{

		Component: "database",

		Latencies: make(map[string]*LatencyStats),

		Counters: map[string]int64{

			"query_count":           d.queryCount.Load(),

			"active_connections":    d.connectionPool.ActiveConnections.Load(),

			"idle_connections":      d.connectionPool.IdleConnections.Load(),

			"total_connections":     d.connectionPool.TotalConnections.Load(),

			"connection_wait_count": d.connectionPool.WaitCount.Load(),

		},

	}



	// Add query-specific stats.

	for queryType, tracker := range d.queryLatency {

		stats.Latencies[fmt.Sprintf("query_%s", queryType)] = tracker.GetStats()

	}



	stats.Latencies["connection_wait"] = d.connectionWait.GetStats()

	stats.Latencies["transaction"] = d.transactionLatency.GetStats()

	stats.Latencies["lock_wait"] = d.lockWaitTime.GetStats()



	// Add slow queries.

	stats.SlowQueries = d.slowQueries.GetQueries()



	return stats

}



// NewCachePerformanceMonitor performs newcacheperformancemonitor operation.

func NewCachePerformanceMonitor(settings CacheMonitorSettings) *CachePerformanceMonitor {

	return &CachePerformanceMonitor{

		getLatency:      NewLatencyTracker(settings.MaxSamples),

		setLatency:      NewLatencyTracker(settings.MaxSamples),

		deleteLatency:   NewLatencyTracker(settings.MaxSamples),

		hitRate:         NewRateTracker(time.Minute),

		evictionLatency: NewLatencyTracker(settings.MaxSamples),

		settings:        settings,

	}

}



// RecordLatency performs recordlatency operation.

func (c *CachePerformanceMonitor) RecordLatency(operation string, duration time.Duration, hit bool) {

	switch operation {

	case "get":

		c.getLatency.Add(duration)

		if hit {

			c.hitCount.Add(1)

			c.hitRate.Record(1)

		} else {

			c.missCount.Add(1)

			c.hitRate.Record(0)

		}

	case "set":

		c.setLatency.Add(duration)

	case "delete":

		c.deleteLatency.Add(duration)

	case "eviction":

		c.evictionLatency.Add(duration)

		c.evictionCount.Add(1)

	}

}



// GetStats performs getstats operation.

func (c *CachePerformanceMonitor) GetStats() *ComponentStats {

	hits := c.hitCount.Load()

	misses := c.missCount.Load()

	total := hits + misses



	hitRate := float64(0)

	if total > 0 {

		hitRate = float64(hits) / float64(total)

	}



	return &ComponentStats{

		Component: "cache",

		Latencies: map[string]*LatencyStats{

			"get":      c.getLatency.GetStats(),

			"set":      c.setLatency.GetStats(),

			"delete":   c.deleteLatency.GetStats(),

			"eviction": c.evictionLatency.GetStats(),

		},

		Counters: map[string]int64{

			"hit_count":      hits,

			"miss_count":     misses,

			"eviction_count": c.evictionCount.Load(),

			"cache_size":     c.cacheSize.Load(),

			"memory_usage":   c.memoryUsage.Load(),

		},

		Rates: map[string]float64{

			"hit_rate":          hitRate,

			"realtime_hit_rate": c.hitRate.GetRate(),

		},

	}

}



// NewQueueProcessingMonitor performs newqueueprocessingmonitor operation.

func NewQueueProcessingMonitor(settings QueueMonitorSettings) *QueueProcessingMonitor {

	return &QueueProcessingMonitor{

		queues:            make(map[string]*QueueMetrics),

		enqueueLatency:    NewLatencyTracker(1000),

		dequeueLatency:    NewLatencyTracker(1000),

		processingLatency: NewLatencyTracker(1000),

		settings:          settings,

	}

}



// RecordLatency performs recordlatency operation.

func (q *QueueProcessingMonitor) RecordLatency(queueName, operation string, duration time.Duration) {

	q.mu.Lock()

	defer q.mu.Unlock()



	// Get or create queue metrics.

	if _, exists := q.queues[queueName]; !exists {

		q.queues[queueName] = &QueueMetrics{

			EnqueueRate: NewRateTracker(time.Minute),

			DequeueRate: NewRateTracker(time.Minute),

			ProcessTime: NewLatencyTracker(1000),

			WaitTime:    NewLatencyTracker(1000),

		}

	}



	queue := q.queues[queueName]



	switch operation {

	case "enqueue":

		q.enqueueLatency.Add(duration)

		queue.EnqueueRate.Record(1)

	case "dequeue":

		q.dequeueLatency.Add(duration)

		queue.DequeueRate.Record(1)

	case "process":

		q.processingLatency.Add(duration)

		queue.ProcessTime.Add(duration)

	case "wait":

		queue.WaitTime.Add(duration)

	}

}



// GetStats performs getstats operation.

func (q *QueueProcessingMonitor) GetStats() *ComponentStats {

	q.mu.RLock()

	defer q.mu.RUnlock()



	stats := &ComponentStats{

		Component: "queue",

		Latencies: map[string]*LatencyStats{

			"enqueue":    q.enqueueLatency.GetStats(),

			"dequeue":    q.dequeueLatency.GetStats(),

			"processing": q.processingLatency.GetStats(),

		},

		QueueStats: make(map[string]*QueueStats),

		Counters: map[string]int64{

			"total_depth": q.totalDepth.Load(),

		},

	}



	// Add queue-specific stats.

	for name, queue := range q.queues {

		stats.QueueStats[name] = &QueueStats{

			Name:        name,

			Depth:       queue.Depth.Load(),

			EnqueueRate: queue.EnqueueRate.GetRate(),

			DequeueRate: queue.DequeueRate.GetRate(),

			ProcessTime: queue.ProcessTime.GetStats(),

			WaitTime:    queue.WaitTime.GetStats(),

		}

	}



	return stats

}



// Helper implementations.



// NewLatencyTracker performs newlatencytracker operation.

func NewLatencyTracker(maxSamples int) *LatencyTracker {

	if maxSamples <= 0 {

		maxSamples = 1000

	}

	return &LatencyTracker{

		samples:    make([]LatencySample, 0, maxSamples),

		maxSamples: maxSamples,

	}

}



// Add performs add operation.

func (l *LatencyTracker) Add(duration time.Duration) {

	l.mu.Lock()

	defer l.mu.Unlock()



	sample := LatencySample{

		Timestamp: time.Now(),

		Duration:  duration,

	}



	l.samples = append(l.samples, sample)

	if len(l.samples) > l.maxSamples {

		l.samples = l.samples[len(l.samples)-l.maxSamples:]

	}



	l.totalCount++

	l.totalLatency += int64(duration)



	// Update percentiles.

	l.updatePercentiles()

}



func (l *LatencyTracker) updatePercentiles() {

	if len(l.samples) == 0 {

		return

	}



	// Create sorted copy.

	durations := make([]time.Duration, len(l.samples))

	for i, s := range l.samples {

		durations[i] = s.Duration

	}



	// Simple bubble sort for small datasets.

	for i := 0; i < len(durations); i++ {

		for j := i + 1; j < len(durations); j++ {

			if durations[j] < durations[i] {

				durations[i], durations[j] = durations[j], durations[i]

			}

		}

	}



	// Calculate percentiles.

	p50Index := len(durations) * 50 / 100

	p95Index := len(durations) * 95 / 100

	p99Index := len(durations) * 99 / 100



	if p50Index < len(durations) {

		l.currentP50 = durations[p50Index]

	}

	if p95Index < len(durations) {

		l.currentP95 = durations[p95Index]

	}

	if p99Index < len(durations) {

		l.currentP99 = durations[p99Index]

	}

}



// GetStats performs getstats operation.

func (l *LatencyTracker) GetStats() *LatencyStats {

	l.mu.RLock()

	defer l.mu.RUnlock()



	if l.totalCount == 0 {

		return &LatencyStats{}

	}



	return &LatencyStats{

		Count:   l.totalCount,

		Mean:    time.Duration(l.totalLatency / l.totalCount),

		P50:     l.currentP50,

		P95:     l.currentP95,

		P99:     l.currentP99,

		Samples: len(l.samples),

	}

}



// NewRateTracker performs newratetracker operation.

func NewRateTracker(window time.Duration) *RateTracker {

	return &RateTracker{

		window:  window,

		buckets: make(map[time.Time]int64),

	}

}



// Record performs record operation.

func (r *RateTracker) Record(value int64) {

	r.mu.Lock()

	defer r.mu.Unlock()



	now := time.Now().Truncate(time.Second)

	r.buckets[now] += value



	// Clean old buckets.

	cutoff := now.Add(-r.window)

	for t := range r.buckets {

		if t.Before(cutoff) {

			delete(r.buckets, t)

		}

	}



	// Calculate rate.

	total := int64(0)

	for _, v := range r.buckets {

		total += v

	}



	r.currentRate = float64(total) / r.window.Seconds()

}



// GetRate performs getrate operation.

func (r *RateTracker) GetRate() float64 {

	r.mu.RLock()

	defer r.mu.RUnlock()

	return r.currentRate

}



// NewSlowQueryTracker performs newslowquerytracker operation.

func NewSlowQueryTracker(threshold time.Duration, maxQueries int) *SlowQueryTracker {

	return &SlowQueryTracker{

		queries:    make([]SlowQuery, 0, maxQueries),

		threshold:  threshold,

		maxQueries: maxQueries,

	}

}



// Add performs add operation.

func (s *SlowQueryTracker) Add(query SlowQuery) {

	s.mu.Lock()

	defer s.mu.Unlock()



	if query.Duration < s.threshold {

		return

	}



	s.queries = append(s.queries, query)

	if len(s.queries) > s.maxQueries {

		s.queries = s.queries[len(s.queries)-s.maxQueries:]

	}

}



// GetQueries performs getqueries operation.

func (s *SlowQueryTracker) GetQueries() []SlowQuery {

	s.mu.RLock()

	defer s.mu.RUnlock()



	result := make([]SlowQuery, len(s.queries))

	copy(result, s.queries)

	return result

}



// NewComponentAggregator performs newcomponentaggregator operation.

func NewComponentAggregator() *ComponentAggregator {

	return &ComponentAggregator{

		aggregates: make(map[string]*ComponentAggregate),

	}

}



// Record performs record operation.

func (a *ComponentAggregator) Record(component, operation string, duration time.Duration) {

	a.mu.Lock()

	defer a.mu.Unlock()



	key := fmt.Sprintf("%s_%s", component, operation)



	if _, exists := a.aggregates[key]; !exists {

		a.aggregates[key] = &ComponentAggregate{

			Component:  component,

			MinLatency: duration,

			MaxLatency: duration,

		}

	}



	agg := a.aggregates[key]

	agg.TotalLatency += duration

	agg.Count++



	if duration < agg.MinLatency {

		agg.MinLatency = duration

	}

	if duration > agg.MaxLatency {

		agg.MaxLatency = duration

	}

}



// GetAggregates performs getaggregates operation.

func (a *ComponentAggregator) GetAggregates() map[string]*ComponentAggregate {

	a.mu.RLock()

	defer a.mu.RUnlock()



	result := make(map[string]*ComponentAggregate)

	for k, v := range a.aggregates {

		result[k] = &ComponentAggregate{

			Component:    v.Component,

			TotalLatency: v.TotalLatency,

			Count:        v.Count,

			P50:          v.P50,

			P95:          v.P95,

			P99:          v.P99,

			MaxLatency:   v.MaxLatency,

			MinLatency:   v.MinLatency,

		}

	}



	return result

}



// NewComponentPerformanceTracker performs newcomponentperformancetracker operation.

func NewComponentPerformanceTracker() *ComponentPerformanceTracker {

	return &ComponentPerformanceTracker{

		performance: make(map[string]*PerformanceMetrics),

		trends:      make(map[string]*PerformanceTrend),

	}

}



// Update performs update operation.

func (p *ComponentPerformanceTracker) Update(component string, latency time.Duration) {

	p.mu.Lock()

	defer p.mu.Unlock()



	if _, exists := p.performance[component]; !exists {

		p.performance[component] = &PerformanceMetrics{

			Component:       component,

			BaselineLatency: latency,

		}

		p.trends[component] = &PerformanceTrend{

			Component:  component,

			DataPoints: []TrendPoint{},

		}

	}



	perf := p.performance[component]

	perf.CurrentLatency = latency



	// Calculate deviation from baseline.

	if perf.BaselineLatency > 0 {

		perf.Deviation = float64(latency-perf.BaselineLatency) / float64(perf.BaselineLatency)

	}



	// Determine trend direction.

	trend := p.trends[component]

	trend.DataPoints = append(trend.DataPoints, TrendPoint{

		Timestamp: time.Now(),

		Value:     latency,

	})



	// Keep only last 100 points.

	if len(trend.DataPoints) > 100 {

		trend.DataPoints = trend.DataPoints[len(trend.DataPoints)-100:]

	}



	// Calculate trend.

	if len(trend.DataPoints) >= 10 {

		p.calculateTrend(trend)



		if trend.Slope > 0.1 {

			perf.TrendDirection = "degrading"

		} else if trend.Slope < -0.1 {

			perf.TrendDirection = "improving"

		} else {

			perf.TrendDirection = "stable"

		}

	}

}



func (p *ComponentPerformanceTracker) calculateTrend(trend *PerformanceTrend) {

	// Simple linear regression.

	n := len(trend.DataPoints)

	if n < 2 {

		return

	}



	var sumX, sumY, sumXY, sumX2 float64



	for i, point := range trend.DataPoints {

		x := float64(i)

		y := float64(point.Value)



		sumX += x

		sumY += y

		sumXY += x * y

		sumX2 += x * x

	}



	// Calculate slope.

	denominator := float64(n)*sumX2 - sumX*sumX

	if denominator != 0 {

		trend.Slope = (float64(n)*sumXY - sumX*sumY) / denominator



		// Predict next value.

		intercept := (sumY - trend.Slope*sumX) / float64(n)

		nextX := float64(n)

		trend.Prediction = time.Duration(trend.Slope*nextX + intercept)

	}

}



// GetTrends performs gettrends operation.

func (p *ComponentPerformanceTracker) GetTrends() map[string]*PerformanceTrend {

	p.mu.RLock()

	defer p.mu.RUnlock()



	result := make(map[string]*PerformanceTrend)

	for k, v := range p.trends {

		result[k] = v

	}



	return result

}



func (m *ComponentMonitor) runAggregation() {

	ticker := time.NewTicker(m.config.AggregationInterval)

	defer ticker.Stop()



	for range ticker.C {

		m.aggregator.lastAggregation = time.Now()

		// Perform any periodic aggregation tasks.

	}

}



func (m *ComponentMonitor) initMetrics() {

	m.metrics = &ComponentMetrics{

		componentLatency: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name:    "nephoran_component_latency_seconds",

			Help:    "Component operation latency",

			Buckets: prometheus.DefBuckets,

		}, []string{"component", "operation"}),



		componentThroughput: prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_component_throughput",

			Help: "Component throughput",

		}, []string{"component"}),



		componentErrors: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "nephoran_component_errors_total",

			Help: "Component errors",

		}, []string{"component", "error_type"}),



		queueDepth: prometheus.NewGaugeVec(prometheus.GaugeOpts{

			Name: "nephoran_queue_depth",

			Help: "Queue depth",

		}, []string{"queue"}),



		cacheHitRate: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "nephoran_cache_hit_rate",

			Help: "Cache hit rate",

		}),

	}

}



// Report types.



// ComponentLatencyReport represents a componentlatencyreport.

type ComponentLatencyReport struct {

	Timestamp         time.Time                      `json:"timestamp"`

	Components        map[string]*ComponentStats     `json:"components"`

	Aggregates        map[string]*ComponentAggregate `json:"aggregates"`

	PerformanceTrends map[string]*PerformanceTrend   `json:"performance_trends"`

}



// ComponentStats represents a componentstats.

type ComponentStats struct {

	Component      string                   `json:"component"`

	Latencies      map[string]*LatencyStats `json:"latencies"`

	Counters       map[string]int64         `json:"counters"`

	Rates          map[string]float64       `json:"rates"`

	ModelStats     map[string]*ModelStats   `json:"model_stats,omitempty"`

	StageLatencies map[string]*LatencyStats `json:"stage_latencies,omitempty"`

	QueueStats     map[string]*QueueStats   `json:"queue_stats,omitempty"`

	SlowQueries    []SlowQuery              `json:"slow_queries,omitempty"`

}



// LatencyStats represents a latencystats.

type LatencyStats struct {

	Count   int64         `json:"count"`

	Mean    time.Duration `json:"mean"`

	P50     time.Duration `json:"p50"`

	P95     time.Duration `json:"p95"`

	P99     time.Duration `json:"p99"`

	Samples int           `json:"samples"`

}



// ModelStats represents a modelstats.

type ModelStats struct {

	Model   string        `json:"model"`

	Usage   int64         `json:"usage"`

	Latency *LatencyStats `json:"latency"`

}



// QueueStats represents a queuestats.

type QueueStats struct {

	Name        string        `json:"name"`

	Depth       int64         `json:"depth"`

	EnqueueRate float64       `json:"enqueue_rate"`

	DequeueRate float64       `json:"dequeue_rate"`

	ProcessTime *LatencyStats `json:"process_time"`

	WaitTime    *LatencyStats `json:"wait_time"`

}



// DefaultComponentMonitorConfig returns default configuration.

func DefaultComponentMonitorConfig() *ComponentMonitorConfig {

	return &ComponentMonitorConfig{

		SamplingInterval:    100 * time.Millisecond,

		AggregationInterval: 10 * time.Second,



		ControllerSettings: ControllerMonitorSettings{

			EnableReconcileTracking: true,

			EnableAPITracking:       true,

			EnableQueueMetrics:      true,

			MaxSamples:              1000,

			ReconcileTimeout:        30 * time.Second,

		},



		LLMSettings: LLMMonitorSettings{

			EnableTokenTracking:   true,

			EnableCostTracking:    true,

			EnableStreamMetrics:   true,

			MaxSamples:            1000,

			SlowResponseThreshold: 5 * time.Second,

		},



		RAGSettings: RAGMonitorSettings{

			EnableVectorMetrics: true,

			EnableCacheMetrics:  true,

			EnableIndexMetrics:  true,

			MaxSamples:          1000,

			SlowSearchThreshold: 500 * time.Millisecond,

		},



		GitOpsSettings: GitOpsMonitorSettings{

			EnableGitMetrics:       true,

			EnablePipelineMetrics:  true,

			EnableDeployMetrics:    true,

			MaxSamples:             1000,

			SlowOperationThreshold: 10 * time.Second,

		},



		DatabaseSettings: DatabaseMonitorSettings{

			EnableQueryTracking: true,

			EnablePoolMetrics:   true,

			EnableSlowQueryLog:  true,

			SlowQueryThreshold:  100 * time.Millisecond,

			MaxSlowQueries:      100,

		},



		CacheSettings: CacheMonitorSettings{

			EnableHitRateTracking: true,

			EnableSizeMetrics:     true,

			EnableEvictionMetrics: true,

			MaxSamples:            1000,

		},



		QueueSettings: QueueMonitorSettings{

			EnableDepthTracking:   true,

			EnableRateMetrics:     true,

			EnableLatencyTracking: true,

			MaxQueues:             50,

		},



		PerformanceThresholds: map[string]time.Duration{

			"controller_reconcile": 200 * time.Millisecond,

			"llm_processing":       800 * time.Millisecond,

			"rag_retrieval":        400 * time.Millisecond,

			"gitops_deployment":    400 * time.Millisecond,

			"database_query":       50 * time.Millisecond,

			"cache_access":         5 * time.Millisecond,

			"queue_processing":     100 * time.Millisecond,

		},

	}

}

