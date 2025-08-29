package o1

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/prometheus/client_golang/api"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// EnhancedFaultManager provides comprehensive O-RAN fault management
// following O-RAN.WG10.O1-Interface.0-v07.00 specification
type EnhancedFaultManager struct {
	config            *FaultManagerConfig
	alarms            map[string]*EnhancedAlarm
	alarmHistory      []*AlarmHistoryEntry
	alarmsMux         sync.RWMutex
	correlationEngine *AlarmCorrelationEngine
	notificationMgr   *AlarmNotificationManager
	thresholdMgr      *AlarmThresholdManager
	maskingMgr        *AlarmMaskingManager
	rootCauseAnalyzer *RootCauseAnalyzer
	websocketUpgrader websocket.Upgrader
	subscribers       map[string]*AlarmSubscriber
	subscribersMux    sync.RWMutex
	prometheusClient  api.Client
	metrics           *FaultMetrics
	streamingEnabled  bool
}

// FaultManagerConfig holds fault manager configuration
type FaultManagerConfig struct {
	MaxAlarms           int
	MaxHistoryEntries   int
	CorrelationWindow   time.Duration
	NotificationTimeout time.Duration
	EnableWebSocket     bool
	PrometheusURL       string
	AlertManagerURL     string
	EnableRootCause     bool
	EnableMasking       bool
	EnableThresholds    bool
}

// EnhancedAlarm extends the basic Alarm with O-RAN specific features
type EnhancedAlarm struct {
	*Alarm
	AlarmIdentifier       uint32                 `json:"alarm_identifier"`
	AlarmRaisedTime       time.Time              `json:"alarm_raised_time"`
	AlarmChangedTime      time.Time              `json:"alarm_changed_time"`
	AlarmClearedTime      time.Time              `json:"alarm_cleared_time"`
	ProposedRepairActions string                 `json:"proposed_repair_actions"`
	AffectedObjects       []string               `json:"affected_objects"`
	RelatedAlarms         []uint32               `json:"related_alarms"`
	RootCauseIndicator    bool                   `json:"root_cause_indicator"`
	VendorSpecificData    map[string]interface{} `json:"vendor_specific_data"`
	CorrelationID         string                 `json:"correlation_id"`
	ProcessingState       string                 `json:"processing_state"`
	AcknowledgmentState   string                 `json:"acknowledgment_state"`
}

// AlarmHistoryEntry represents a historical alarm record
type AlarmHistoryEntry struct {
	AlarmID          string                 `json:"alarm_id"`
	Timestamp        time.Time              `json:"timestamp"`
	Action           string                 `json:"action"` // RAISED, CHANGED, CLEARED, ACKNOWLEDGED
	Severity         string                 `json:"severity"`
	Details          map[string]interface{} `json:"details"`
	OperatorComments string                 `json:"operator_comments,omitempty"`
}

// AlarmCorrelationEngine performs alarm correlation and root cause analysis
type AlarmCorrelationEngine struct {
	rules            map[string]*FaultCorrelationRule
	correlationGraph *AlarmGraph
	temporalWindow   time.Duration
	spatialRules     map[string]*SpatialCorrelationRule
	mutex            sync.RWMutex
}

// FaultCorrelationRule defines alarm correlation rules for fault management
type FaultCorrelationRule struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	TriggerPattern []AlarmPattern `json:"trigger_pattern"`
	ResultAction   string         `json:"result_action"` // SUPPRESS, CORRELATE, ROOT_CAUSE
	TimeWindow     time.Duration  `json:"time_window"`
	Enabled        bool           `json:"enabled"`
	Priority       int            `json:"priority"`
}

// AlarmPattern represents a pattern for alarm correlation
type AlarmPattern struct {
	AlarmType       string            `json:"alarm_type,omitempty"`
	Severity        string            `json:"severity,omitempty"`
	ObjectType      string            `json:"object_type,omitempty"`
	ObjectPattern   string            `json:"object_pattern,omitempty"` // regex pattern
	Conditions      map[string]string `json:"conditions,omitempty"`
	Count           int               `json:"count,omitempty"` // min occurrences
	TimeConstraint  time.Duration     `json:"time_constraint,omitempty"`
}

// SpatialCorrelationRule defines spatial correlation rules
type SpatialCorrelationRule struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Scope       string `json:"scope"` // CELL, SECTOR, SITE, CLUSTER
	Radius      int    `json:"radius"` // for geographic correlation
	Enabled     bool   `json:"enabled"`
	Priority    int    `json:"priority"`
}

// AlarmGraph represents the alarm correlation graph
type AlarmGraph struct {
	nodes map[string]*AlarmNode
	edges map[string][]*AlarmEdge
	mutex sync.RWMutex
}

// AlarmNode represents a node in the alarm graph
type AlarmNode struct {
	AlarmID     string      `json:"alarm_id"`
	ObjectID    string      `json:"object_id"`
	AlarmType   string      `json:"alarm_type"`
	Severity    string      `json:"severity"`
	Timestamp   time.Time   `json:"timestamp"`
	State       string      `json:"state"` // ACTIVE, CLEARED, CORRELATED
	Metadata    interface{} `json:"metadata"`
}

// AlarmEdge represents a correlation between alarms
type AlarmEdge struct {
	FromAlarmID    string  `json:"from_alarm_id"`
	ToAlarmID      string  `json:"to_alarm_id"`
	CorrelationType string  `json:"correlation_type"` // CAUSAL, TEMPORAL, SPATIAL
	Strength       float64 `json:"strength"` // 0.0 to 1.0
	Timestamp      time.Time `json:"timestamp"`
}

// AlarmNotificationManager handles alarm notifications
type AlarmNotificationManager struct {
	channels     map[string]FaultNotificationChannel
	templates    map[string]*NotificationTemplate
	rules        map[string]*NotificationRule
	rateLimiter  *RateLimiter
	batchManager *BatchNotificationManager
	mutex        sync.RWMutex
}

// NotificationRule defines when and how to send notifications
type NotificationRule struct {
	ID           string   `json:"id"`
	Name         string   `json:"name"`
	AlarmTypes   []string `json:"alarm_types,omitempty"`
	Severities   []string `json:"severities,omitempty"`
	ObjectTypes  []string `json:"object_types,omitempty"`
	ChannelIDs   []string `json:"channel_ids"`
	TemplateID   string   `json:"template_id"`
	Conditions   map[string]interface{} `json:"conditions,omitempty"`
	Enabled      bool     `json:"enabled"`
	Priority     int      `json:"priority"`
}

// RateLimiter controls notification rate limits
type RateLimiter struct {
	maxPerMinute int
	windows      map[string]*rateLimitWindow
	mutex        sync.RWMutex
}

type rateLimitWindow struct {
	count     int
	resetTime time.Time
}

// BatchNotificationManager handles batch notifications
type BatchNotificationManager struct {
	batchSize      int
	flushInterval  time.Duration
	pendingBatch   []*EnhancedAlarm
	mutex          sync.Mutex
	stopChan       chan bool
}

// AlarmThresholdManager manages alarm thresholds
type AlarmThresholdManager struct {
	thresholds map[string]*AlarmThreshold
	monitors   map[string]*ThresholdMonitor
	metrics    map[string]*MetricSource
	mutex      sync.RWMutex
}

// AlarmThreshold defines threshold conditions for alarm generation
type AlarmThreshold struct {
	ID             string              `json:"id"`
	Name           string              `json:"name"`
	MetricName     string              `json:"metric_name"`
	ObjectType     string              `json:"object_type"`
	Conditions     []*ThresholdCondition `json:"conditions"`
	AlarmTemplate  *AlarmTemplate      `json:"alarm_template"`
	Enabled        bool                `json:"enabled"`
	MonitoringInterval time.Duration   `json:"monitoring_interval"`
	ClearConditions []*ThresholdCondition `json:"clear_conditions,omitempty"`
}

// ThresholdCondition defines a threshold condition
type ThresholdCondition struct {
	Operator  string      `json:"operator"` // GT, LT, GTE, LTE, EQ, NE
	Value     interface{} `json:"value"`
	Duration  time.Duration `json:"duration,omitempty"` // condition must persist
	Severity  string      `json:"severity"`
}

// AlarmTemplate defines template for threshold-generated alarms
type AlarmTemplate struct {
	AlarmType       string            `json:"alarm_type"`
	ProbableCause   string            `json:"probable_cause"`
	SpecificProblem string            `json:"specific_problem"`
	AlarmTextTemplate string          `json:"alarm_text_template"`
	AdditionalInfoTemplate map[string]string `json:"additional_info_template,omitempty"`
}

// ThresholdMonitor monitors metrics against thresholds
type ThresholdMonitor struct {
	thresholdID string
	metricName  string
	objectID    string
	lastCheck   time.Time
	state       string // NORMAL, BREACHED, CLEARED
	history     []*ThresholdEvent
	mutex       sync.RWMutex
}

// ThresholdEvent represents a threshold event
type ThresholdEvent struct {
	Timestamp time.Time   `json:"timestamp"`
	Value     interface{} `json:"value"`
	State     string      `json:"state"`
	Condition *ThresholdCondition `json:"condition"`
}

// MetricSource represents a source of metrics for threshold monitoring
type MetricSource struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // PROMETHEUS, SNMP, JMX, etc.
	Endpoint string `json:"endpoint"`
	Query    string `json:"query"`
	Config   map[string]interface{} `json:"config"`
}

// AlarmMaskingManager handles alarm masking/suppression
type AlarmMaskingManager struct {
	masks      map[string]*AlarmMask
	conditions map[string]*MaskingCondition
	mutex      sync.RWMutex
}

// AlarmMask defines alarm masking rules
type AlarmMask struct {
	ID          string               `json:"id"`
	Name        string               `json:"name"`
	Conditions  []*MaskingCondition  `json:"conditions"`
	Action      string               `json:"action"` // SUPPRESS, DELAY, MODIFY
	Schedule    *MaskingSchedule     `json:"schedule,omitempty"`
	Enabled     bool                 `json:"enabled"`
	CreatedBy   string               `json:"created_by"`
	CreatedAt   time.Time            `json:"created_at"`
	ValidUntil  *time.Time           `json:"valid_until,omitempty"`
}

// MaskingCondition defines conditions for alarm masking
type MaskingCondition struct {
	Field    string      `json:"field"` // alarm_type, severity, object_type, etc.
	Operator string      `json:"operator"` // EQ, NE, CONTAINS, REGEX, etc.
	Value    interface{} `json:"value"`
}

// MaskingSchedule defines when masking is active
type MaskingSchedule struct {
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	Recurring  bool      `json:"recurring"`
	CronExpr   string    `json:"cron_expr,omitempty"`
	TimeZone   string    `json:"time_zone"`
}

// RootCauseAnalyzer performs root cause analysis
type RootCauseAnalyzer struct {
	knowledgeBase *KnowledgeBase
	analysisRules map[string]*AnalysisRule
	mlModels      map[string]*MLModel
	mutex         sync.RWMutex
}

// KnowledgeBase contains known fault patterns and their causes
type KnowledgeBase struct {
	faultPatterns map[string]*FaultPattern
	symptoms      map[string]*Symptom
	causes        map[string]*Cause
	relationships map[string][]*CauseRelationship
}

// FaultPattern represents a known fault pattern
type FaultPattern struct {
	ID          string      `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Symptoms    []string    `json:"symptoms"`
	Causes      []string    `json:"causes"`
	Confidence  float64     `json:"confidence"`
	Metadata    interface{} `json:"metadata"`
}

// Symptom represents observable symptoms
type Symptom struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // ALARM, METRIC, EVENT
	Indicators  map[string]interface{} `json:"indicators"`
	Weight      float64                `json:"weight"`
}

// Cause represents potential root causes
type Cause struct {
	ID           string                 `json:"id"`
	Name         string                 `json:"name"`
	Category     string                 `json:"category"` // HARDWARE, SOFTWARE, NETWORK, etc.
	Description  string                 `json:"description"`
	Remediation  []string               `json:"remediation"`
	Probability  float64                `json:"probability"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// CauseRelationship represents relationships between causes
type CauseRelationship struct {
	FromCause string  `json:"from_cause"`
	ToCause   string  `json:"to_cause"`
	Type      string  `json:"type"` // LEADS_TO, CAUSED_BY, CORRELATES_WITH
	Strength  float64 `json:"strength"`
}

// AnalysisRule defines rules for root cause analysis
type AnalysisRule struct {
	ID         string             `json:"id"`
	Name       string             `json:"name"`
	Conditions []*AnalysisCondition `json:"conditions"`
	Actions    []*AnalysisAction    `json:"actions"`
	Priority   int                `json:"priority"`
	Enabled    bool               `json:"enabled"`
}

// AnalysisCondition defines conditions for analysis rules
type AnalysisCondition struct {
	Type      string      `json:"type"` // ALARM_COUNT, METRIC_THRESHOLD, etc.
	Parameter string      `json:"parameter"`
	Operator  string      `json:"operator"`
	Value     interface{} `json:"value"`
	TimeWindow time.Duration `json:"time_window,omitempty"`
}

// AnalysisAction defines actions for analysis rules
type AnalysisAction struct {
	Type       string                 `json:"type"` // SET_ROOT_CAUSE, CREATE_TICKET, etc.
	Parameters map[string]interface{} `json:"parameters"`
}

// MLModel represents machine learning models for analysis
type MLModel struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"` // CLASSIFICATION, REGRESSION, etc.
	Version     string                 `json:"version"`
	ModelPath   string                 `json:"model_path"`
	Features    []string               `json:"features"`
	Config      map[string]interface{} `json:"config"`
	Accuracy    float64                `json:"accuracy"`
	LastTrained time.Time              `json:"last_trained"`
}

// AlarmSubscriber represents an alarm subscriber for streaming
type AlarmSubscriber struct {
	ID           string                 `json:"id"`
	Connection   *websocket.Conn        `json:"-"`
	Filters      *AlarmFilter           `json:"filters"`
	LastSeen     time.Time              `json:"last_seen"`
	BufferSize   int                    `json:"buffer_size"`
	MessageCount int64                  `json:"message_count"`
	Metadata     map[string]interface{} `json:"metadata"`
}

// AlarmFilter defines filtering criteria for alarm subscriptions
type AlarmFilter struct {
	AlarmTypes    []string  `json:"alarm_types,omitempty"`
	Severities    []string  `json:"severities,omitempty"`
	ObjectTypes   []string  `json:"object_types,omitempty"`
	ObjectIDs     []string  `json:"object_ids,omitempty"`
	StartTime     *time.Time `json:"start_time,omitempty"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	States        []string  `json:"states,omitempty"`
	Limit         int       `json:"limit,omitempty"`
	IncludeCleared bool     `json:"include_cleared"`
}

// FaultMetrics holds fault management metrics
type FaultMetrics struct {
	ActiveAlarms         prometheus.Gauge
	TotalAlarmsRaised    prometheus.Counter
	TotalAlarmsCleared   prometheus.Counter
	AlarmsByType         *prometheus.CounterVec
	AlarmsBySeverity     *prometheus.CounterVec
	AlarmProcessingTime  prometheus.Histogram
	CorrelationSuccess   prometheus.Counter
	CorrelationFailures  prometheus.Counter
	NotificationsSent    prometheus.Counter
	NotificationFailures prometheus.Counter
	ThresholdBreaches    prometheus.Counter
	MaskedAlarms         prometheus.Counter
}

// NewEnhancedFaultManager creates a new enhanced fault manager
func NewEnhancedFaultManager(config *FaultManagerConfig) (*EnhancedFaultManager, error) {
	if config == nil {
		config = &FaultManagerConfig{
			MaxAlarms:           10000,
			MaxHistoryEntries:   50000,
			CorrelationWindow:   5 * time.Minute,
			NotificationTimeout: 30 * time.Second,
			EnableWebSocket:     true,
			EnableRootCause:     true,
			EnableMasking:       true,
			EnableThresholds:    true,
		}
	}

	fm := &EnhancedFaultManager{
		config:       config,
		alarms:       make(map[string]*EnhancedAlarm),
		alarmHistory: make([]*AlarmHistoryEntry, 0, config.MaxHistoryEntries),
		subscribers:  make(map[string]*AlarmSubscriber),
		websocketUpgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool { return true },
		},
		streamingEnabled: config.EnableWebSocket,
	}

	// Initialize components
	if config.EnableRootCause {
		fm.correlationEngine = NewAlarmCorrelationEngine(config.CorrelationWindow)
		fm.rootCauseAnalyzer = NewRootCauseAnalyzer()
	}

	fm.notificationMgr = NewAlarmNotificationManager()

	if config.EnableThresholds {
		fm.thresholdMgr = NewAlarmThresholdManager()
	}

	if config.EnableMasking {
		fm.maskingMgr = NewAlarmMaskingManager()
	}

	// Initialize metrics
	fm.metrics = &FaultMetrics{
		ActiveAlarms: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "oran_o1_active_alarms_total",
			Help: "Total number of active alarms",
		}),
		TotalAlarmsRaised: promauto.NewCounter(prometheus.CounterOpts{
			Name: "oran_o1_alarms_raised_total",
			Help: "Total number of alarms raised",
		}),
		TotalAlarmsCleared: promauto.NewCounter(prometheus.CounterOpts{
			Name: "oran_o1_alarms_cleared_total",
			Help: "Total number of alarms cleared",
		}),
		AlarmsByType: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "oran_o1_alarms_by_type_total",
			Help: "Total alarms by type",
		}, []string{"alarm_type"}),
		AlarmsBySeverity: promauto.NewCounterVec(prometheus.CounterOpts{
			Name: "oran_o1_alarms_by_severity_total",
			Help: "Total alarms by severity",
		}, []string{"severity"}),
		AlarmProcessingTime: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "oran_o1_alarm_processing_duration_seconds",
			Help:    "Time spent processing alarms",
			Buckets: prometheus.DefBuckets,
		}),
	}

	return fm, nil
}

// RaiseAlarm raises a new alarm
func (fm *EnhancedFaultManager) RaiseAlarm(ctx context.Context, alarm *Alarm) (*EnhancedAlarm, error) {
	logger := log.FromContext(ctx)
	
	startTime := time.Now()
	defer func() {
		fm.metrics.AlarmProcessingTime.Observe(time.Since(startTime).Seconds())
	}()

	fm.alarmsMux.Lock()
	defer fm.alarmsMux.Unlock()

	// Check for duplicates
	if existingAlarm, exists := fm.alarms[alarm.AlarmID]; exists {
		if existingAlarm.ClearedTime.IsZero() {
			logger.Info("Alarm already exists and is active", "alarmID", alarm.AlarmID)
			return existingAlarm, nil
		}
	}

	// Check if alarm should be masked
	if fm.maskingMgr != nil && fm.maskingMgr.ShouldMaskAlarm(alarm) {
		fm.metrics.MaskedAlarms.Inc()
		logger.Info("Alarm masked", "alarmID", alarm.AlarmID)
		return nil, fmt.Errorf("alarm masked by policy")
	}

	// Create enhanced alarm
	enhancedAlarm := &EnhancedAlarm{
		Alarm:               alarm,
		AlarmIdentifier:     fm.generateAlarmIdentifier(),
		AlarmRaisedTime:     time.Now(),
		ProcessingState:     "NEW",
		AcknowledgmentState: "UNACKNOWLEDGED",
		VendorSpecificData:  make(map[string]interface{}),
	}

	// Store alarm
	fm.alarms[alarm.AlarmID] = enhancedAlarm

	// Add to history
	fm.addToHistory(&AlarmHistoryEntry{
		AlarmID:   alarm.AlarmID,
		Timestamp: enhancedAlarm.AlarmRaisedTime,
		Action:    "RAISED",
		Severity:  alarm.PerceivedSeverity,
		Details: map[string]interface{}{
			"alarm_type":     alarm.AlarmType,
			"object_type":    alarm.ObjectType,
			"object_instance": alarm.ObjectInstance,
			"probable_cause": alarm.ProbableCause,
		},
	})

	// Update metrics
	fm.metrics.ActiveAlarms.Inc()
	fm.metrics.TotalAlarmsRaised.Inc()
	fm.metrics.AlarmsByType.WithLabelValues(alarm.AlarmType).Inc()
	fm.metrics.AlarmsBySeverity.WithLabelValues(alarm.PerceivedSeverity).Inc()

	// Perform correlation analysis
	if fm.correlationEngine != nil {
		fm.correlationEngine.AnalyzeAlarm(enhancedAlarm)
	}

	// Send notifications
	if fm.notificationMgr != nil {
		go func() {
			if err := fm.notificationMgr.ProcessAlarmNotification(ctx, enhancedAlarm); err != nil {
				logger.Error(err, "Failed to process alarm notification", "alarmID", alarm.AlarmID)
			}
		}()
	}

	// Stream to subscribers
	if fm.streamingEnabled {
		fm.streamAlarmToSubscribers(enhancedAlarm, "RAISED")
	}

	logger.Info("Alarm raised successfully", 
		"alarmID", alarm.AlarmID,
		"severity", alarm.PerceivedSeverity,
		"managedObjectID", alarm.ManagedObjectID,
		"additionalText", alarm.AdditionalText)

	return enhancedAlarm, nil
}

// ClearAlarm clears an active alarm
func (fm *EnhancedFaultManager) ClearAlarm(ctx context.Context, alarmID string) error {
	logger := log.FromContext(ctx)
	
	fm.alarmsMux.Lock()
	defer fm.alarmsMux.Unlock()

	enhancedAlarm, exists := fm.alarms[alarmID]
	if !exists {
		return fmt.Errorf("alarm not found: %s", alarmID)
	}

	if !enhancedAlarm.ClearedTime.IsZero() {
		return fmt.Errorf("alarm already cleared: %s", alarmID)
	}

	// Clear the alarm
	now := time.Now()
	enhancedAlarm.AlarmClearedTime = now
	enhancedAlarm.ClearedTime = &now
	enhancedAlarm.ProcessingState = "CLEARED"

	// Add to history
	fm.addToHistory(&AlarmHistoryEntry{
		AlarmID:   alarmID,
		Timestamp: now,
		Action:    "CLEARED",
		Severity:  enhancedAlarm.PerceivedSeverity,
		Details: map[string]interface{}{
			"cleared_time": now,
			"managed_object_id": enhancedAlarm.ManagedObjectID,
			"additional_text": enhancedAlarm.AdditionalText,
		},
	})

	// Update metrics
	fm.metrics.ActiveAlarms.Dec()
	fm.metrics.TotalAlarmsCleared.Inc()

	// Stream to subscribers
	if fm.streamingEnabled {
		fm.streamAlarmToSubscribers(enhancedAlarm, "CLEARED")
	}

	// Send notifications
	if fm.notificationMgr != nil {
		go func() {
			if err := fm.notificationMgr.ProcessAlarmNotification(ctx, enhancedAlarm); err != nil {
				logger.Error(err, "Failed to process alarm clear notification", "alarmID", alarmID)
			}
		}()
	}

	logger.Info("Alarm cleared successfully", "alarmID", alarmID)
	return nil
}

// GetAlarm retrieves an alarm by ID
func (fm *EnhancedFaultManager) GetAlarm(alarmID string) (*EnhancedAlarm, error) {
	fm.alarmsMux.RLock()
	defer fm.alarmsMux.RUnlock()

	alarm, exists := fm.alarms[alarmID]
	if !exists {
		return nil, fmt.Errorf("alarm not found: %s", alarmID)
	}

	return alarm, nil
}

// ListAlarms returns a list of alarms matching the filter
func (fm *EnhancedFaultManager) ListAlarms(filter *AlarmFilter) ([]*EnhancedAlarm, error) {
	fm.alarmsMux.RLock()
	defer fm.alarmsMux.RUnlock()

	var result []*EnhancedAlarm
	for _, alarm := range fm.alarms {
		if fm.matchesFilter(alarm, filter) {
			result = append(result, alarm)
		}
	}

	// Sort by raised time (most recent first)
	sort.Slice(result, func(i, j int) bool {
		return result[i].AlarmRaisedTime.After(result[j].AlarmRaisedTime)
	})

	return result, nil
}

// AcknowledgeAlarm acknowledges an alarm
func (fm *EnhancedFaultManager) AcknowledgeAlarm(ctx context.Context, alarmID, operatorID, comment string) error {
	logger := log.FromContext(ctx)
	
	fm.alarmsMux.Lock()
	defer fm.alarmsMux.Unlock()

	enhancedAlarm, exists := fm.alarms[alarmID]
	if !exists {
		return fmt.Errorf("alarm not found: %s", alarmID)
	}

	enhancedAlarm.Acknowledged = true
	enhancedAlarm.AcknowledgmentState = "ACKNOWLEDGED"
	enhancedAlarm.AlarmChangedTime = time.Now()

	// Add to history
	fm.addToHistory(&AlarmHistoryEntry{
		AlarmID:          alarmID,
		Timestamp:        time.Now(),
		Action:           "ACKNOWLEDGED",
		Severity:         enhancedAlarm.PerceivedSeverity,
		OperatorComments: comment,
		Details: map[string]interface{}{
			"operator_id": operatorID,
			"managed_object_id": enhancedAlarm.ManagedObjectID,
		},
	})

	// Stream to subscribers
	if fm.streamingEnabled {
		fm.streamAlarmToSubscribers(enhancedAlarm, "ACKNOWLEDGED")
	}

	logger.Info("Alarm acknowledged", "alarmID", alarmID, "operatorID", operatorID)
	return nil
}

// generateAlarmIdentifier generates a unique alarm identifier
func (fm *EnhancedFaultManager) generateAlarmIdentifier() uint32 {
	// Simple implementation - in production would use proper ID generation
	return uint32(time.Now().Unix())
}

// addToHistory adds an entry to alarm history with cleanup
func (fm *EnhancedFaultManager) addToHistory(entry *AlarmHistoryEntry) {
	fm.alarmHistory = append(fm.alarmHistory, entry)
	
	// Cleanup old entries if we exceed the limit
	if len(fm.alarmHistory) > fm.config.MaxHistoryEntries {
		// Remove oldest 10% of entries
		removeCount := fm.config.MaxHistoryEntries / 10
		fm.alarmHistory = fm.alarmHistory[removeCount:]
	}
}

// matchesFilter checks if an alarm matches the given filter
func (fm *EnhancedFaultManager) matchesFilter(alarm *EnhancedAlarm, filter *AlarmFilter) bool {
	if filter == nil {
		return true
	}

	// Check alarm types
	if len(filter.AlarmTypes) > 0 {
		found := false
		for _, alarmType := range filter.AlarmTypes {
			if alarm.AlarmType == alarmType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check severities
	if len(filter.Severities) > 0 {
		found := false
		for _, severity := range filter.Severities {
			if alarm.PerceivedSeverity == severity {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check object types
	if len(filter.ObjectTypes) > 0 {
		found := false
		for _, objectType := range filter.ObjectTypes {
			if alarm.ObjectType == objectType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check object IDs
	if len(filter.ObjectIDs) > 0 {
		found := false
		for _, objectID := range filter.ObjectIDs {
			if alarm.ManagedObjectID == objectID {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check time range
	if filter.StartTime != nil && alarm.AlarmRaisedTime.Before(*filter.StartTime) {
		return false
	}
	if filter.EndTime != nil && alarm.AlarmRaisedTime.After(*filter.EndTime) {
		return false
	}

	// Check cleared status
	if !filter.IncludeCleared && !alarm.ClearedTime.IsZero() {
		return false
	}

	return true
}

// streamAlarmToSubscribers streams alarm updates to WebSocket subscribers
func (fm *EnhancedFaultManager) streamAlarmToSubscribers(alarm *EnhancedAlarm, action string) {
	fm.subscribersMux.RLock()
	defer fm.subscribersMux.RUnlock()

	message := map[string]interface{}{
		"action": action,
		"alarm":  alarm,
		"timestamp": time.Now(),
	}

	for _, subscriber := range fm.subscribers {
		if fm.matchesFilter(alarm, subscriber.Filters) {
			select {
			case <-time.After(1 * time.Second): // Non-blocking send with timeout
				// Skip slow subscribers
				continue
			default:
				if err := subscriber.Connection.WriteJSON(message); err != nil {
					// Remove failed connection
					delete(fm.subscribers, subscriber.ID)
				} else {
					subscriber.MessageCount++
					subscriber.LastSeen = time.Now()
				}
			}
		}
	}
}

// Helper functions for other managers (stubs for now)
func NewAlarmCorrelationEngine(window time.Duration) *AlarmCorrelationEngine {
	return &AlarmCorrelationEngine{
		rules:            make(map[string]*FaultCorrelationRule),
		temporalWindow:   window,
		spatialRules:     make(map[string]*SpatialCorrelationRule),
		correlationGraph: &AlarmGraph{
			nodes: make(map[string]*AlarmNode),
			edges: make(map[string][]*AlarmEdge),
		},
	}
}

func (ace *AlarmCorrelationEngine) AnalyzeAlarm(alarm *EnhancedAlarm) {
	// Placeholder for correlation analysis
}

func NewAlarmNotificationManager() *AlarmNotificationManager {
	return &AlarmNotificationManager{
		channels:  make(map[string]FaultNotificationChannel),
		templates: make(map[string]*NotificationTemplate),
		rules:     make(map[string]*NotificationRule),
	}
}

func (anm *AlarmNotificationManager) ProcessAlarmNotification(ctx context.Context, alarm *EnhancedAlarm) error {
	// Placeholder for notification processing
	return nil
}

func NewAlarmThresholdManager() *AlarmThresholdManager {
	return &AlarmThresholdManager{
		thresholds: make(map[string]*AlarmThreshold),
		monitors:   make(map[string]*ThresholdMonitor),
		metrics:    make(map[string]*MetricSource),
	}
}

func NewAlarmMaskingManager() *AlarmMaskingManager {
	return &AlarmMaskingManager{
		masks:      make(map[string]*AlarmMask),
		conditions: make(map[string]*MaskingCondition),
	}
}

func (amm *AlarmMaskingManager) ShouldMaskAlarm(alarm *Alarm) bool {
	// Placeholder for masking logic
	return false
}

func NewRootCauseAnalyzer() *RootCauseAnalyzer {
	return &RootCauseAnalyzer{
		knowledgeBase: &KnowledgeBase{
			faultPatterns: make(map[string]*FaultPattern),
			symptoms:      make(map[string]*Symptom),
			causes:        make(map[string]*Cause),
			relationships: make(map[string][]*CauseRelationship),
		},
		analysisRules: make(map[string]*AnalysisRule),
		mlModels:      make(map[string]*MLModel),
	}
}