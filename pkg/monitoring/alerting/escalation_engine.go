// Package alerting provides automated escalation policies and workflows.

// for rapid incident response and appropriate stakeholder engagement.

package alerting

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"github.com/nephio-project/nephoran-intent-operator/pkg/logging"
)

// EscalationEngine manages automated escalation policies and workflows.

// providing intelligent, time-based escalation with business context awareness.

type EscalationEngine struct {
	logger *logging.StructuredLogger

	config *EscalationConfig

	// Escalation management.

	escalationPolicies map[string]*EscalationPolicy

	activeEscalations map[string]*ActiveEscalation

	escalationHistory []*EscalationEvent

	// Stakeholder management.

	stakeholderRegistry *StakeholderRegistry

	oncallSchedule *OncallSchedule

	// Auto-resolution.

	autoResolver *AutoResolver

	resolutionDetector *ResolutionDetector

	// Workflow integration.

	workflowExecutor *WorkflowExecutor

	ticketingSystem *TicketingSystem

	// Performance tracking.

	metrics *EscalationMetrics

	escalationStats *EscalationStatistics

	// State management.

	started bool

	stopCh chan struct{}

	escalationQueue chan *EscalationRequest

	mu sync.RWMutex
}

// EscalationConfig holds configuration for the escalation engine.

type EscalationConfig struct {

	// Core escalation settings.

	DefaultEscalationDelay time.Duration `yaml:"default_escalation_delay"`

	MaxEscalationLevels int `yaml:"max_escalation_levels"`

	EscalationTimeout time.Duration `yaml:"escalation_timeout"`

	// Auto-resolution settings.

	AutoResolutionEnabled bool `yaml:"auto_resolution_enabled"`

	ResolutionCheckInterval time.Duration `yaml:"resolution_check_interval"`

	AutoResolutionTimeout time.Duration `yaml:"auto_resolution_timeout"`

	// Workflow settings.

	EnableWorkflowAutomation bool `yaml:"enable_workflow_automation"`

	WorkflowTimeout time.Duration `yaml:"workflow_timeout"`

	RetryFailedWorkflows bool `yaml:"retry_failed_workflows"`

	// Notification settings.

	EscalationNotifyDelay time.Duration `yaml:"escalation_notify_delay"`

	AcknowledgmentTimeout time.Duration `yaml:"acknowledgment_timeout"`

	// Business context.

	BusinessHoursEscalation bool `yaml:"business_hours_escalation"`

	WeekendEscalationDelay time.Duration `yaml:"weekend_escalation_delay"`

	// Performance settings.

	MaxConcurrentEscalations int `yaml:"max_concurrent_escalations"`

	EscalationQueueSize int `yaml:"escalation_queue_size"`
}

// EscalationPolicy defines how alerts should be escalated.

type EscalationPolicy struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Enabled bool `json:"enabled"`

	// Policy triggers.

	TriggerConditions []EscalationTrigger `json:"trigger_conditions"`

	// Escalation levels.

	Levels []EscalationLevel `json:"levels"`

	// Auto-resolution rules.

	AutoResolution *AutoResolutionPolicy `json:"auto_resolution,omitempty"`

	// Workflow integration.

	Workflows []WorkflowReference `json:"workflows,omitempty"`

	// Business context.

	BusinessImpact BusinessImpactRules `json:"business_impact"`

	Schedule *EscalationSchedule `json:"schedule,omitempty"`

	// Metadata.

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`

	CreatedBy string `json:"created_by"`

	Version int `json:"version"`
}

// EscalationTrigger defines conditions that trigger escalation.

type EscalationTrigger struct {
	Field string `json:"field"` // severity, sla_type, business_impact

	Operator string `json:"operator"` // equals, greater_than, contains

	Value string `json:"value"`

	TimeThreshold time.Duration `json:"time_threshold,omitempty"`
}

// EscalationLevel defines a single level in an escalation policy.

type EscalationLevel struct {
	Level int `json:"level"`

	Name string `json:"name"`

	Delay time.Duration `json:"delay"`

	Timeout time.Duration `json:"timeout"`

	// Stakeholders to notify.

	Stakeholders []StakeholderReference `json:"stakeholders"`

	// Actions to take.

	Actions []EscalationAction `json:"actions"`

	// Conditions for moving to next level.

	EscalationRules []EscalationRule `json:"escalation_rules"`

	// Schedule constraints.

	Schedule *LevelSchedule `json:"schedule,omitempty"`
}

// StakeholderReference references a stakeholder for notification.

type StakeholderReference struct {
	Type string `json:"type"` // individual, team, role, oncall

	Identifier string `json:"identifier"` // user_id, team_id, role_name

	NotificationMethod string `json:"notification_method"` // email, sms, phone, slack

	Parameters map[string]string `json:"parameters,omitempty"`
}

// EscalationAction defines an action to take during escalation.

type EscalationAction struct {
	Type string `json:"type"` // notify, create_ticket, run_workflow, auto_remediate

	Name string `json:"name"`

	Parameters map[string]string `json:"parameters"`

	Timeout time.Duration `json:"timeout,omitempty"`

	RetryCount int `json:"retry_count,omitempty"`

	OnFailure string `json:"on_failure,omitempty"` // continue, skip_level, stop

}

// EscalationRule defines when to escalate to the next level.

type EscalationRule struct {
	Type string `json:"type"` // time_based, acknowledgment_based, condition_based

	Condition string `json:"condition"` // unacknowledged, unresolved, condition_worsened

	Threshold time.Duration `json:"threshold,omitempty"`

	RequiredCount int `json:"required_count,omitempty"`
}

// AutoResolutionPolicy defines automatic resolution behavior.

type AutoResolutionPolicy struct {
	Enabled bool `json:"enabled"`

	ResolutionConditions []ResolutionCondition `json:"resolution_conditions"`

	ConfirmationWindow time.Duration `json:"confirmation_window"`

	NotifyOnResolution bool `json:"notify_on_resolution"`
}

// ResolutionCondition defines conditions for automatic resolution.

type ResolutionCondition struct {
	Type string `json:"type"` // metric_based, time_based, external_signal

	Condition string `json:"condition"` // metric query, duration, signal name

	Threshold interface{} `json:"threshold"`

	Duration time.Duration `json:"duration,omitempty"`
}

// WorkflowReference references an automated workflow.

type WorkflowReference struct {
	WorkflowID string `json:"workflow_id"`

	TriggerLevel int `json:"trigger_level"`

	Parameters map[string]string `json:"parameters,omitempty"`

	Async bool `json:"async"`
}

// BusinessImpactRules define business context for escalation.

type BusinessImpactRules struct {
	HighImpactEscalationDelay time.Duration `json:"high_impact_escalation_delay"`

	CustomerFacingPriority bool `json:"customer_facing_priority"`

	RevenueThresholds []RevenueThreshold `json:"revenue_thresholds"`
}

// RevenueThreshold defines revenue-based escalation rules.

type RevenueThreshold struct {
	MinRevenue float64 `json:"min_revenue"`

	EscalationDelay time.Duration `json:"escalation_delay"`

	RequiredApprovers []string `json:"required_approvers"`
}

// EscalationSchedule defines schedule constraints for escalation.

type EscalationSchedule struct {
	Timezone string `json:"timezone"`

	BusinessHours ScheduleWindow `json:"business_hours"`

	WeekendSchedule *WeekendSchedule `json:"weekend_schedule,omitempty"`

	HolidaySchedule *HolidaySchedule `json:"holiday_schedule,omitempty"`

	MaintenanceWindows []MaintenanceWindow `json:"maintenance_windows,omitempty"`
}

// ScheduleWindow defines a time window.

type ScheduleWindow struct {
	Start int `json:"start"` // Hour (0-23)

	End int `json:"end"` // Hour (0-23)

	Weekdays []int `json:"weekdays"` // Days (0=Sunday)

}

// WeekendSchedule defines weekend-specific escalation behavior.

type WeekendSchedule struct {
	Enabled bool `json:"enabled"`

	DelayMultiplier float64 `json:"delay_multiplier"`

	SkipLevels []int `json:"skip_levels,omitempty"`
}

// HolidaySchedule defines holiday-specific escalation behavior.

type HolidaySchedule struct {
	Enabled bool `json:"enabled"`

	Holidays []string `json:"holidays"`

	Behavior string `json:"behavior"` // skip, delay, emergency_only

}

// LevelSchedule defines schedule constraints for a specific level.

type LevelSchedule struct {
	OnlyDuringBusinessHours bool `json:"only_during_business_hours"`

	SkipWeekends bool `json:"skip_weekends"`

	MinimumStakeholders int `json:"minimum_stakeholders"`

	FallbackStakeholders []StakeholderReference `json:"fallback_stakeholders,omitempty"`
}

// ActiveEscalation represents an ongoing escalation.

type ActiveEscalation struct {
	ID string `json:"id"`

	AlertID string `json:"alert_id"`

	PolicyID string `json:"policy_id"`

	CurrentLevel int `json:"current_level"`

	State EscalationState `json:"state"`

	// Timing information.

	StartedAt time.Time `json:"started_at"`

	LastEscalated time.Time `json:"last_escalated"`

	NextEscalation *time.Time `json:"next_escalation,omitempty"`

	// Stakeholder interactions.

	Notifications []NotificationRecord `json:"notifications"`

	Acknowledgments []Acknowledgment `json:"acknowledgments"`

	// Workflow execution.

	ExecutedWorkflows []WorkflowExecution `json:"executed_workflows"`

	// Business context.

	BusinessImpact BusinessImpactScore `json:"business_impact"`

	Priority int `json:"priority"`

	// Resolution tracking.

	ResolutionAttempts []ResolutionAttempt `json:"resolution_attempts"`

	AutoResolved bool `json:"auto_resolved"`
}

// EscalationState represents the state of an escalation.

type EscalationState string

const (

	// EscalationStateActive holds escalationstateactive value.

	EscalationStateActive EscalationState = "active"

	// EscalationStateAcknowledged holds escalationstateacknowledged value.

	EscalationStateAcknowledged EscalationState = "acknowledged"

	// EscalationStateResolved holds escalationstateresolved value.

	EscalationStateResolved EscalationState = "resolved"

	// EscalationStateSuppressed holds escalationstatesuppressed value.

	EscalationStateSuppressed EscalationState = "suppressed"

	// EscalationStateTimedOut holds escalationstatetimedout value.

	EscalationStateTimedOut EscalationState = "timed_out"

	// EscalationStateFailed holds escalationstatefailed value.

	EscalationStateFailed EscalationState = "failed"
)

// NotificationRecord tracks sent notifications.

type NotificationRecord struct {
	ID string `json:"id"`

	Level int `json:"level"`

	Stakeholder StakeholderReference `json:"stakeholder"`

	Method string `json:"method"`

	SentAt time.Time `json:"sent_at"`

	DeliveredAt *time.Time `json:"delivered_at,omitempty"`

	Status string `json:"status"`

	Response string `json:"response,omitempty"`
}

// Acknowledgment tracks stakeholder acknowledgments.

type Acknowledgment struct {
	ID string `json:"id"`

	StakeholderID string `json:"stakeholder_id"`

	AcknowledgedAt time.Time `json:"acknowledged_at"`

	Message string `json:"message,omitempty"`

	Method string `json:"method"`
}

// WorkflowExecution tracks automated workflow executions.

type WorkflowExecution struct {
	ID string `json:"id"`

	WorkflowID string `json:"workflow_id"`

	Level int `json:"level"`

	StartedAt time.Time `json:"started_at"`

	CompletedAt *time.Time `json:"completed_at,omitempty"`

	Status string `json:"status"`

	Result map[string]interface{} `json:"result,omitempty"`

	Error string `json:"error,omitempty"`
}

// ResolutionAttempt tracks resolution attempts.

type ResolutionAttempt struct {
	ID string `json:"id"`

	Type string `json:"type"` // manual, automatic, workflow

	AttemptedAt time.Time `json:"attempted_at"`

	Success bool `json:"success"`

	Details string `json:"details,omitempty"`

	AttemptedBy string `json:"attempted_by,omitempty"`
}

// EscalationEvent represents a historical escalation event.

type EscalationEvent struct {
	ID string `json:"id"`

	EscalationID string `json:"escalation_id"`

	AlertID string `json:"alert_id"`

	EventType string `json:"event_type"` // started, escalated, acknowledged, resolved

	Level int `json:"level"`

	Timestamp time.Time `json:"timestamp"`

	Stakeholder *StakeholderReference `json:"stakeholder,omitempty"`

	Details map[string]interface{} `json:"details,omitempty"`
}

// EscalationRequest represents a request to start escalation.

type EscalationRequest struct {
	Alert *SLAAlert `json:"alert"`

	PolicyID string `json:"policy_id,omitempty"`

	Priority int `json:"priority"`
}

// StakeholderRegistry manages stakeholder information.

type StakeholderRegistry struct {
	logger *logging.StructuredLogger

	stakeholders map[string]*Stakeholder

	teams map[string]*Team

	roles map[string]*Role

	mu sync.RWMutex
}

// Stakeholder represents an individual stakeholder.

type Stakeholder struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Email string `json:"email"`

	Phone string `json:"phone,omitempty"`

	SlackUserID string `json:"slack_user_id,omitempty"`

	TimeZone string `json:"timezone"`

	Preferences map[string]string `json:"preferences"`

	Active bool `json:"active"`

	AvailabilityHours ScheduleWindow `json:"availability_hours"`
}

// Team represents a team of stakeholders.

type Team struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Members []string `json:"members"` // Stakeholder IDs

	Leads []string `json:"leads"` // Stakeholder IDs who are team leads

	EscalationOrder []string `json:"escalation_order"` // Order for team escalation

	Active bool `json:"active"`
}

// Role represents a functional role.

type Role struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Permissions []string `json:"permissions"`

	Stakeholders []string `json:"stakeholders"` // Current stakeholders in this role

}

// OncallSchedule manages on-call schedules.

type OncallSchedule struct {
	logger *logging.StructuredLogger

	schedules map[string]*Schedule

	mu sync.RWMutex
}

// Schedule represents an on-call schedule.

type Schedule struct {
	ID string `json:"id"`

	Name string `json:"name"`

	TimeZone string `json:"timezone"`

	Rotations []Rotation `json:"rotations"`

	Overrides []Override `json:"overrides"`

	Active bool `json:"active"`
}

// Rotation defines a rotation schedule.

type Rotation struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Duration time.Duration `json:"duration"`

	StartTime time.Time `json:"start_time"`

	Participants []string `json:"participants"` // Stakeholder IDs

	Current int `json:"current"` // Current participant index

}

// Override represents a schedule override.

type Override struct {
	ID string `json:"id"`

	StakeholderID string `json:"stakeholder_id"`

	StartTime time.Time `json:"start_time"`

	EndTime time.Time `json:"end_time"`

	Reason string `json:"reason"`
}

// AutoResolver handles automatic alert resolution.

type AutoResolver struct {
	logger *logging.StructuredLogger

	config *EscalationConfig

	detector *ResolutionDetector
}

// ResolutionDetector detects when alerts should be automatically resolved.

type ResolutionDetector struct {
	logger *logging.StructuredLogger

	config *EscalationConfig
}

// WorkflowExecutor executes automated workflows.

type WorkflowExecutor struct {
	logger *logging.StructuredLogger

	workflows map[string]*Workflow
}

// Workflow defines an automated workflow.

type Workflow struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Steps []WorkflowStep `json:"steps"`

	Timeout time.Duration `json:"timeout"`

	RetryPolicy *RetryPolicy `json:"retry_policy,omitempty"`
}

// WorkflowStep defines a single step in a workflow.

type WorkflowStep struct {
	ID string `json:"id"`

	Type string `json:"type"` // http, script, notification, condition

	Name string `json:"name"`

	Parameters map[string]string `json:"parameters"`

	Timeout time.Duration `json:"timeout"`

	OnSuccess string `json:"on_success,omitempty"` // next_step, complete

	OnFailure string `json:"on_failure,omitempty"` // retry, next_step, fail

}

// RetryPolicy defines retry behavior for workflows.

type RetryPolicy struct {
	MaxRetries int `json:"max_retries"`

	Backoff time.Duration `json:"backoff"`

	Multiplier float64 `json:"multiplier"`
}

// TicketingSystem integrates with external ticketing systems.

type TicketingSystem struct {
	logger *logging.StructuredLogger

	providers map[string]TicketingProvider
}

// TicketingProvider defines interface for ticketing systems.

type TicketingProvider interface {
	CreateTicket(ctx context.Context, alert *SLAAlert, escalation *ActiveEscalation) (*Ticket, error)

	UpdateTicket(ctx context.Context, ticketID string, update TicketUpdate) error

	GetTicket(ctx context.Context, ticketID string) (*Ticket, error)

	CloseTicket(ctx context.Context, ticketID, reason string) error
}

// Ticket represents a ticket in an external system.

type Ticket struct {
	ID string `json:"id"`

	ExternalID string `json:"external_id"`

	Title string `json:"title"`

	Description string `json:"description"`

	Status string `json:"status"`

	Priority string `json:"priority"`

	Assignee string `json:"assignee,omitempty"`

	CreatedAt time.Time `json:"created_at"`

	UpdatedAt time.Time `json:"updated_at"`

	URL string `json:"url,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TicketUpdate represents an update to a ticket.

type TicketUpdate struct {
	Status string `json:"status,omitempty"`

	Priority string `json:"priority,omitempty"`

	Assignee string `json:"assignee,omitempty"`

	Comment string `json:"comment,omitempty"`

	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// EscalationStatistics tracks escalation performance.

type EscalationStatistics struct {
	TotalEscalations int64 `json:"total_escalations"`

	EscalationsResolved int64 `json:"escalations_resolved"`

	EscalationsTimedOut int64 `json:"escalations_timed_out"`

	AutoResolutions int64 `json:"auto_resolutions"`

	AverageResolutionTime time.Duration `json:"average_resolution_time"`

	AverageAcknowledgmentTime time.Duration `json:"average_acknowledgment_time"`

	EscalationsByLevel map[int]int64 `json:"escalations_by_level"`

	PolicyEffectiveness map[string]float64 `json:"policy_effectiveness"`
}

// EscalationMetrics contains Prometheus metrics.

type EscalationMetrics struct {
	EscalationsStarted *prometheus.CounterVec

	EscalationsResolved *prometheus.CounterVec

	EscalationDuration *prometheus.HistogramVec

	AcknowledgmentTime *prometheus.HistogramVec

	NotificationsSent *prometheus.CounterVec

	WorkflowsExecuted *prometheus.CounterVec

	ActiveEscalations prometheus.Gauge
}

// DefaultEscalationConfig returns production-ready escalation configuration.

func DefaultEscalationConfig() *EscalationConfig {

	return &EscalationConfig{

		// Core escalation settings.

		DefaultEscalationDelay: 15 * time.Minute,

		MaxEscalationLevels: 4,

		EscalationTimeout: 4 * time.Hour,

		// Auto-resolution settings.

		AutoResolutionEnabled: true,

		ResolutionCheckInterval: 1 * time.Minute,

		AutoResolutionTimeout: 30 * time.Minute,

		// Workflow settings.

		EnableWorkflowAutomation: true,

		WorkflowTimeout: 10 * time.Minute,

		RetryFailedWorkflows: true,

		// Notification settings.

		EscalationNotifyDelay: 2 * time.Minute,

		AcknowledgmentTimeout: 10 * time.Minute,

		// Business context.

		BusinessHoursEscalation: true,

		WeekendEscalationDelay: 30 * time.Minute,

		// Performance settings.

		MaxConcurrentEscalations: 50,

		EscalationQueueSize: 100,
	}

}

// NewEscalationEngine creates a new escalation engine.

func NewEscalationEngine(config *EscalationConfig, logger *logging.StructuredLogger) (*EscalationEngine, error) {

	if config == nil {

		config = DefaultEscalationConfig()

	}

	if logger == nil {

		return nil, fmt.Errorf("logger is required")

	}

	// Initialize metrics.

	metrics := &EscalationMetrics{

		EscalationsStarted: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "escalation_engine_escalations_started_total",

			Help: "Total number of escalations started",
		}, []string{"policy", "severity", "sla_type"}),

		EscalationsResolved: prometheus.NewCounterVec(prometheus.CounterOpts{

			Name: "escalation_engine_escalations_resolved_total",

			Help: "Total number of escalations resolved",
		}, []string{"policy", "level", "method"}),

		EscalationDuration: prometheus.NewHistogramVec(prometheus.HistogramOpts{

			Name: "escalation_engine_duration_seconds",

			Help: "Duration of escalations in seconds",

			Buckets: prometheus.ExponentialBuckets(60, 2, 12), // 1 minute to ~68 hours

		}, []string{"policy", "result"}),

		ActiveEscalations: prometheus.NewGauge(prometheus.GaugeOpts{

			Name: "escalation_engine_active_escalations",

			Help: "Number of currently active escalations",
		}),
	}

	// Register metrics with duplicate handling.

	escalationMetrics := []prometheus.Collector{

		metrics.EscalationsStarted,

		metrics.EscalationsResolved,

		metrics.EscalationDuration,

		metrics.ActiveEscalations,
	}

	// Register each metric, ignoring duplicate registration errors.

	for _, metric := range escalationMetrics {

		if err := prometheus.Register(metric); err != nil {

			if _, ok := err.(prometheus.AlreadyRegisteredError); !ok {

				// Only propagate non-duplicate errors.

				logger.Error("Failed to register escalation metric", "error", err)

			}

		}

	}

	ee := &EscalationEngine{

		logger: logger.WithComponent("escalation-engine"),

		config: config,

		escalationPolicies: make(map[string]*EscalationPolicy),

		activeEscalations: make(map[string]*ActiveEscalation),

		escalationHistory: make([]*EscalationEvent, 0),

		metrics: metrics,

		escalationStats: &EscalationStatistics{

			EscalationsByLevel: make(map[int]int64),

			PolicyEffectiveness: make(map[string]float64),
		},

		stopCh: make(chan struct{}),

		escalationQueue: make(chan *EscalationRequest, config.EscalationQueueSize),
	}

	// Initialize sub-components.

	ee.stakeholderRegistry = &StakeholderRegistry{

		logger: logger.WithComponent("stakeholder-registry"),

		stakeholders: make(map[string]*Stakeholder),

		teams: make(map[string]*Team),

		roles: make(map[string]*Role),
	}

	ee.oncallSchedule = &OncallSchedule{

		logger: logger.WithComponent("oncall-schedule"),

		schedules: make(map[string]*Schedule),
	}

	ee.autoResolver = &AutoResolver{

		logger: logger.WithComponent("auto-resolver"),

		config: config,

		detector: &ResolutionDetector{

			logger: logger.WithComponent("resolution-detector"),

			config: config,
		},
	}

	ee.workflowExecutor = &WorkflowExecutor{

		logger: logger.WithComponent("workflow-executor"),

		workflows: make(map[string]*Workflow),
	}

	ee.ticketingSystem = &TicketingSystem{

		logger: logger.WithComponent("ticketing-system"),

		providers: make(map[string]TicketingProvider),
	}

	// Load default configurations.

	ee.loadDefaultEscalationPolicies()

	ee.loadDefaultStakeholders()

	ee.loadDefaultWorkflows()

	return ee, nil

}

// Start initializes the escalation engine.

func (ee *EscalationEngine) Start(ctx context.Context) error {

	ee.mu.Lock()

	defer ee.mu.Unlock()

	if ee.started {

		return fmt.Errorf("escalation engine already started")

	}

	ee.logger.InfoWithContext("Starting escalation engine",

		"max_escalation_levels", ee.config.MaxEscalationLevels,

		"default_delay", ee.config.DefaultEscalationDelay,

		"auto_resolution_enabled", ee.config.AutoResolutionEnabled,
	)

	// Start processing workers.

	for i := range 3 { // Start 3 escalation workers

		go ee.escalationWorker(ctx, i)

	}

	// Start background processes.

	go ee.escalationMonitor(ctx)

	if ee.config.AutoResolutionEnabled {

		go ee.autoResolutionMonitor(ctx)

	}

	go ee.metricsUpdateLoop(ctx)

	ee.started = true

	ee.logger.InfoWithContext("Escalation engine started successfully")

	return nil

}

// Stop shuts down the escalation engine.

func (ee *EscalationEngine) Stop(ctx context.Context) error {

	ee.mu.Lock()

	defer ee.mu.Unlock()

	if !ee.started {

		return nil

	}

	ee.logger.InfoWithContext("Stopping escalation engine")

	close(ee.stopCh)

	close(ee.escalationQueue)

	ee.started = false

	ee.logger.InfoWithContext("Escalation engine stopped")

	return nil

}

// StartEscalation begins escalation for an alert.

func (ee *EscalationEngine) StartEscalation(ctx context.Context, alert *SLAAlert) error {

	request := &EscalationRequest{

		Alert: alert,

		Priority: ee.calculateAlertPriority(alert),
	}

	// Find appropriate escalation policy.

	policy := ee.findEscalationPolicy(alert)

	if policy != nil {

		request.PolicyID = policy.ID

	}

	select {

	case ee.escalationQueue <- request:

		return nil

	case <-ctx.Done():

		return ctx.Err()

	default:

		return fmt.Errorf("escalation queue is full")

	}

}

// escalationWorker processes escalation requests.

func (ee *EscalationEngine) escalationWorker(ctx context.Context, workerID int) {

	ee.logger.DebugWithContext("Starting escalation worker",

		slog.Int("worker_id", workerID),
	)

	for {

		select {

		case <-ctx.Done():

			return

		case <-ee.stopCh:

			return

		case request, ok := <-ee.escalationQueue:

			if !ok {

				return

			}

			ee.processEscalationRequest(ctx, request, workerID)

		}

	}

}

// processEscalationRequest processes a single escalation request.

func (ee *EscalationEngine) processEscalationRequest(ctx context.Context, request *EscalationRequest, workerID int) {

	ee.logger.InfoWithContext("Starting escalation",

		slog.String("alert_id", request.Alert.ID),

		slog.String("policy_id", request.PolicyID),

		slog.Int("priority", request.Priority),

		slog.Int("worker_id", workerID),
	)

	// Create active escalation.

	escalation := &ActiveEscalation{

		ID: fmt.Sprintf("esc-%s-%d", request.Alert.ID, time.Now().Unix()),

		AlertID: request.Alert.ID,

		PolicyID: request.PolicyID,

		CurrentLevel: 0,

		State: EscalationStateActive,

		StartedAt: time.Now(),

		Priority: request.Priority,

		BusinessImpact: BusinessImpactScore{

			OverallScore: ee.calculateBusinessImpact(request.Alert),
		},

		Notifications: make([]NotificationRecord, 0),

		Acknowledgments: make([]Acknowledgment, 0),

		ExecutedWorkflows: make([]WorkflowExecution, 0),

		ResolutionAttempts: make([]ResolutionAttempt, 0),
	}

	// Store active escalation.

	ee.mu.Lock()

	ee.activeEscalations[escalation.ID] = escalation

	ee.mu.Unlock()

	// Update metrics.

	policy := ee.escalationPolicies[request.PolicyID]

	if policy != nil {

		ee.metrics.EscalationsStarted.WithLabelValues(

			policy.Name,

			string(request.Alert.Severity),

			string(request.Alert.SLAType),
		).Inc()

	}

	ee.metrics.ActiveEscalations.Inc()

	ee.escalationStats.TotalEscalations++

	// Record escalation event.

	ee.recordEscalationEvent(&EscalationEvent{

		ID: fmt.Sprintf("event-%d", time.Now().UnixNano()),

		EscalationID: escalation.ID,

		AlertID: escalation.AlertID,

		EventType: "started",

		Level: 0,

		Timestamp: time.Now(),

		Details: map[string]interface{}{

			"policy_id": request.PolicyID,

			"priority": request.Priority,
		},
	})

	// Start escalation process.

	ee.executeEscalationLevel(ctx, escalation, 0)

}

// executeEscalationLevel executes a specific escalation level.

func (ee *EscalationEngine) executeEscalationLevel(ctx context.Context, escalation *ActiveEscalation, level int) {

	policy, exists := ee.escalationPolicies[escalation.PolicyID]

	if !exists || level >= len(policy.Levels) {

		ee.logger.WarnWithContext("Invalid escalation level or policy",

			slog.String("escalation_id", escalation.ID),

			slog.String("policy_id", escalation.PolicyID),

			slog.Int("level", level),
		)

		return

	}

	levelConfig := policy.Levels[level]

	escalation.CurrentLevel = level

	ee.logger.InfoWithContext("Executing escalation level",

		slog.String("escalation_id", escalation.ID),

		slog.Int("level", level),

		slog.String("level_name", levelConfig.Name),
	)

	// Apply level delay if specified.

	if level > 0 && levelConfig.Delay > 0 {

		time.Sleep(levelConfig.Delay)

	}

	// Execute level actions.

	for _, action := range levelConfig.Actions {

		if err := ee.executeEscalationAction(ctx, escalation, action, level); err != nil {

			ee.logger.ErrorWithContext("Failed to execute escalation action", err,

				"action_type", action.Type, "alert_id", escalation.AlertID, "level", level)

		}

	}

	// Notify stakeholders.

	for _, stakeholder := range levelConfig.Stakeholders {

		if err := ee.notifyStakeholder(ctx, escalation, stakeholder, level); err != nil {

			ee.logger.ErrorWithContext("Failed to notify stakeholder", err,

				"stakeholder", stakeholder.Identifier, "alert_id", escalation.AlertID, "level", level)

		}

	}

	// Schedule next escalation level if conditions are met.

	ee.scheduleNextEscalation(ctx, escalation, levelConfig)

	// Update escalation statistics.

	ee.escalationStats.EscalationsByLevel[level]++

}

// Additional methods would include:.

// - executeEscalationAction: Execute specific escalation actions.

// - notifyStakeholder: Send notifications to stakeholders.

// - scheduleNextEscalation: Schedule next level based on rules.

// - escalationMonitor: Monitor active escalations for progression.

// - autoResolutionMonitor: Check for automatic resolution conditions.

// - Stakeholder management methods.

// - Workflow execution methods.

// - Ticketing system integration.

// - Policy management methods.

// - Statistics and metrics calculation.

// escalationMonitor monitors active escalations for progression.

func (ee *EscalationEngine) escalationMonitor(ctx context.Context) {

	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			// Check for escalations that need progression.

			ee.checkEscalationProgression(ctx)

		}

	}

}

// autoResolutionMonitor checks for automatic resolution conditions.

func (ee *EscalationEngine) autoResolutionMonitor(ctx context.Context) {

	ticker := time.NewTicker(60 * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			// Check for auto-resolution conditions.

			ee.checkAutoResolution(ctx)

		}

	}

}

// metricsUpdateLoop periodically updates escalation metrics.

func (ee *EscalationEngine) metricsUpdateLoop(ctx context.Context) {

	ticker := time.NewTicker(30 * time.Second)

	defer ticker.Stop()

	for {

		select {

		case <-ctx.Done():

			return

		case <-ticker.C:

			// Update escalation metrics.

			ee.updateEscalationMetrics(ctx)

		}

	}

}

// executeEscalationAction executes specific escalation actions.

func (ee *EscalationEngine) executeEscalationAction(ctx context.Context, escalation *ActiveEscalation, action EscalationAction, level int) error {

	ee.logger.InfoWithContext(fmt.Sprintf("Executing escalation action %s for alert %s at level %d",

		action.Type, escalation.AlertID, level))

	switch action.Type {

	case "notification":

		return ee.sendNotification(ctx, escalation, action.Parameters)

	case "ticket":

		return ee.createTicket(ctx, escalation, action.Parameters)

	case "webhook":

		return ee.callWebhook(ctx, escalation, action.Parameters)

	default:

		return fmt.Errorf("unknown escalation action type: %s", action.Type)

	}

}

// notifyStakeholder sends notifications to stakeholders.

func (ee *EscalationEngine) notifyStakeholder(ctx context.Context, escalation *ActiveEscalation, stakeholder StakeholderReference, level int) error {

	ee.logger.InfoWithContext(fmt.Sprintf("Notifying stakeholder %s for alert %s at level %d",

		stakeholder.Identifier, escalation.AlertID, level))

	// Implementation would send notifications via email, SMS, etc.

	return nil

}

// scheduleNextEscalation schedules the next escalation level.

func (ee *EscalationEngine) scheduleNextEscalation(ctx context.Context, escalation *ActiveEscalation, levelConfig EscalationLevel) {

	nextTime := time.Now().Add(levelConfig.Delay)

	ee.logger.InfoWithContext(fmt.Sprintf("Scheduling next escalation for alert %s at %v",

		escalation.AlertID, nextTime))

	// Implementation would schedule the next escalation.

}

// Helper methods for escalation monitoring.

func (ee *EscalationEngine) checkEscalationProgression(ctx context.Context) {

	// Check active escalations and progress them as needed.

}

func (ee *EscalationEngine) checkAutoResolution(ctx context.Context) {

	// Check if any active escalations can be auto-resolved.

}

func (ee *EscalationEngine) updateEscalationMetrics(ctx context.Context) {

	// Update metrics for escalation monitoring.

}

func (ee *EscalationEngine) sendNotification(ctx context.Context, escalation *ActiveEscalation, parameters map[string]string) error {

	// Send notification implementation.

	return nil

}

func (ee *EscalationEngine) createTicket(ctx context.Context, escalation *ActiveEscalation, parameters map[string]string) error {

	// Create ticket implementation.

	return nil

}

func (ee *EscalationEngine) callWebhook(ctx context.Context, escalation *ActiveEscalation, parameters map[string]string) error {

	// Call webhook implementation.

	return nil

}

// Helper methods for configuration loading and management.

func (ee *EscalationEngine) loadDefaultEscalationPolicies() {

	// Load default escalation policies for different SLA types and severities.

	// This would typically load from configuration files or database.

}

func (ee *EscalationEngine) loadDefaultStakeholders() {

	// Load stakeholder information from configuration or external systems.

}

func (ee *EscalationEngine) loadDefaultWorkflows() {

	// Load automated workflow definitions.

}

// Simplified implementations for key methods.

func (ee *EscalationEngine) calculateAlertPriority(alert *SLAAlert) int {

	// Calculate priority based on severity, business impact, and SLA type.

	var basePriority int

	switch alert.Severity {

	case AlertSeverityUrgent:

		basePriority = 5

	case AlertSeverityCritical:

		basePriority = 4

	case AlertSeverityMajor:

		basePriority = 3

	case AlertSeverityWarning:

		basePriority = 2

	default:

		basePriority = 1

	}

	// Adjust for business impact.

	if alert.BusinessImpact.CustomerFacing {

		basePriority++

	}

	return basePriority

}

func (ee *EscalationEngine) findEscalationPolicy(alert *SLAAlert) *EscalationPolicy {

	// Find the most appropriate escalation policy for the alert.

	// This would evaluate trigger conditions and return the best match.

	for _, policy := range ee.escalationPolicies {

		if ee.evaluatePolicyTriggers(alert, policy.TriggerConditions) {

			return policy

		}

	}

	// Return default policy if no specific match.

	if defaultPolicy, exists := ee.escalationPolicies["default"]; exists {

		return defaultPolicy

	}

	return nil

}

func (ee *EscalationEngine) evaluatePolicyTriggers(alert *SLAAlert, triggers []EscalationTrigger) bool {

	// Evaluate if the alert matches the policy triggers.

	// Simplified implementation - production would be more sophisticated.

	return true

}

func (ee *EscalationEngine) calculateBusinessImpact(alert *SLAAlert) float64 {

	// Calculate business impact score based on alert characteristics.

	impact := 0.0

	// Base impact by SLA type.

	switch alert.SLAType {

	case SLATypeAvailability:

		impact += 0.4

	case SLATypeLatency:

		impact += 0.3

	case SLAThroughput:

		impact += 0.2

	case SLAErrorRate:

		impact += 0.5

	}

	// Severity multiplier.

	switch alert.Severity {

	case AlertSeverityUrgent:

		impact *= 2.0

	case AlertSeverityCritical:

		impact *= 1.5

	case AlertSeverityMajor:

		impact *= 1.2

	}

	// Business context adjustments.

	if alert.BusinessImpact.CustomerFacing {

		impact *= 1.3

	}

	return math.Min(impact, 1.0)

}

func (ee *EscalationEngine) recordEscalationEvent(event *EscalationEvent) {

	ee.mu.Lock()

	defer ee.mu.Unlock()

	ee.escalationHistory = append(ee.escalationHistory, event)

	// Keep only recent history to prevent memory bloat.

	if len(ee.escalationHistory) > 10000 {

		ee.escalationHistory = ee.escalationHistory[1000:]

	}

}
