package security

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"
)

// IncidentResponse manages security incident detection and response.

type IncidentResponse struct {
	config *IncidentConfig

	logger *slog.Logger

	incidents map[string]*SecurityIncident

	playbooks map[string]*ResponsePlaybook

	escalation *EscalationEngine

	forensics *ForensicsCollector

	metrics *IncidentMetrics

	mutex sync.RWMutex

	stopChan chan struct{}
}

// IncidentConfig holds incident response configuration.

type IncidentConfig struct {
	EnableAutoResponse bool `json:"enable_auto_response"`

	AutoResponseThreshold string `json:"auto_response_threshold"` // Critical, High, Medium

	MaxAutoActions int `json:"max_auto_actions"`

	IncidentRetention time.Duration `json:"incident_retention"`

	EscalationTimeout time.Duration `json:"escalation_timeout"`

	ForensicsEnabled bool `json:"forensics_enabled"`

	NotificationConfig *NotificationConfig `json:"notification_config"`

	IntegrationConfig *IRIntegrationConfig `json:"integration_config"`

	WebhookSecret string `json:"webhook_secret"`
}

// NotificationConfig holds notification settings.

type NotificationConfig struct {
	EnableEmail bool `json:"enable_email"`

	EnableSlack bool `json:"enable_slack"`

	EnableSMS bool `json:"enable_sms"`

	EnablePagerDuty bool `json:"enable_pagerduty"`

	Recipients []string `json:"recipients"`

	EscalationList []string `json:"escalation_list"`
}

// IRIntegrationConfig holds integration settings for incident response.

type IRIntegrationConfig struct {
	SIEM *SIEMConfig `json:"siem,omitempty"`

	SOAR *SOARConfig `json:"soar,omitempty"`

	Ticketing *TicketingConfig `json:"ticketing,omitempty"`
}

// SIEMConfig holds SIEM integration configuration.

type SIEMConfig struct {
	Type string `json:"type"` // splunk, elk, sentinel

	Endpoint string `json:"endpoint"`

	APIKey string `json:"api_key"`

	Index string `json:"index"`
}

// SOARConfig holds SOAR platform integration.

type SOARConfig struct {
	Platform string `json:"platform"` // phantom, demisto, etc.

	Endpoint string `json:"endpoint"`

	APIKey string `json:"api_key"`
}

// TicketingConfig holds ticketing system integration.

type TicketingConfig struct {
	System string `json:"system"` // jira, servicenow, etc.

	Endpoint string `json:"endpoint"`

	APIKey string `json:"api_key"`

	Project string `json:"project"`
}

// SecurityIncident represents a security incident.

type SecurityIncident struct {
	ID string `json:"id"`

	Title string `json:"title"`

	Description string `json:"description"`

	Severity string `json:"severity"`

	Status string `json:"status"`

	Category string `json:"category"`

	Source string `json:"source"`

	DetectedAt time.Time `json:"detected_at"`

	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`

	ResolvedAt *time.Time `json:"resolved_at,omitempty"`

	Assignee string `json:"assignee"`

	Tags []string `json:"tags"`

	Evidence []*Evidence `json:"evidence"`

	Timeline []*TimelineEvent `json:"timeline"`

	Actions []*ResponseAction `json:"actions"`

	Artifacts json.RawMessage `json:"artifacts"`

	MITRE *MITREMapping `json:"mitre,omitempty"`

	Impact *ImpactAssessment `json:"impact"`

	Remediation *RemediationPlan `json:"remediation"`
}

// Evidence represents incident evidence.

type Evidence struct {
	ID string `json:"id"`

	Type string `json:"type"` // log, file, network, memory

	Source string `json:"source"`

	Timestamp time.Time `json:"timestamp"`

	Description string `json:"description"`

	Data json.RawMessage `json:"data"`

	Hash string `json:"hash"`

	Collected bool `json:"collected"`
}

// TimelineEvent represents an event in the incident timeline.

type TimelineEvent struct {
	Timestamp time.Time `json:"timestamp"`

	Type string `json:"type"`

	Description string `json:"description"`

	Actor string `json:"actor"`

	Automated bool `json:"automated"`
}

// ResponseAction represents an automated response action.

type ResponseAction struct {
	ID string `json:"id"`

	Type string `json:"type"`

	Description string `json:"description"`

	Status string `json:"status"`

	ExecutedAt *time.Time `json:"executed_at,omitempty"`

	CompletedAt *time.Time `json:"completed_at,omitempty"`

	Result string `json:"result"`

	Parameters json.RawMessage `json:"parameters"`

	Automated bool `json:"automated"`
}

// MITREMapping represents MITRE ATT&CK framework mapping.

type MITREMapping struct {
	Tactics []string `json:"tactics"`

	Techniques []string `json:"techniques"`

	SubTechniques []string `json:"sub_techniques"`

	Confidence float64 `json:"confidence"`
}

// ImpactAssessment represents the impact assessment of an incident.

type ImpactAssessment struct {
	Confidentiality string `json:"confidentiality"` // None, Low, Medium, High

	Integrity string `json:"integrity"`

	Availability string `json:"availability"`

	BusinessImpact string `json:"business_impact"`

	AffectedSystems []string `json:"affected_systems"`

	AffectedUsers int `json:"affected_users"`

	EstimatedCost float64 `json:"estimated_cost"`
}

// RemediationPlan represents the remediation plan.

type RemediationPlan struct {
	ShortTermActions []string `json:"short_term_actions"`

	LongTermActions []string `json:"long_term_actions"`

	PreventiveActions []string `json:"preventive_actions"`

	Timeline time.Duration `json:"timeline"`

	AssignedTo string `json:"assigned_to"`

	Status string `json:"status"`
}

// ResponsePlaybook represents an automated response playbook.

type ResponsePlaybook struct {
	ID string `json:"id"`

	Name string `json:"name"`

	Description string `json:"description"`

	Triggers []*PlaybookTrigger `json:"triggers"`

	Actions []*PlaybookAction `json:"actions"`

	Enabled bool `json:"enabled"`

	Priority int `json:"priority"`

	LastExecuted *time.Time `json:"last_executed,omitempty"`
}

// PlaybookTrigger represents a playbook trigger condition.

type PlaybookTrigger struct {
	Type string `json:"type"`

	Conditions json.RawMessage `json:"conditions"`
}

// PlaybookAction represents a playbook action.

type PlaybookAction struct {
	ID string `json:"id"`

	Type string `json:"type"`

	Description string `json:"description"`

	Parameters json.RawMessage `json:"parameters"`

	Timeout time.Duration `json:"timeout"`

	RetryCount int `json:"retry_count"`

	OnFailure string `json:"on_failure"` // continue, abort, escalate
}

// EscalationEngine handles incident escalation.

type EscalationEngine struct {
	config *IncidentConfig

	logger *slog.Logger

	rules []*EscalationRule

	mutex sync.RWMutex
}

// EscalationRule represents an escalation rule.

type EscalationRule struct {
	ID string `json:"id"`

	Conditions []*Condition `json:"conditions"`

	Actions []*Action `json:"actions"`

	Timeout time.Duration `json:"timeout"`

	Enabled bool `json:"enabled"`
}

// Condition represents an escalation condition.

type Condition struct {
	Field string `json:"field"`

	Operator string `json:"operator"`

	Value interface{} `json:"value"`
}

// Action represents an escalation action.

type Action struct {
	Type string `json:"type"`

	Parameters json.RawMessage `json:"parameters"`
}

// ForensicsCollector handles evidence collection.

type ForensicsCollector struct {
	config *IncidentConfig

	logger *slog.Logger

	storage *EvidenceStorage
}

// EvidenceStorage manages evidence storage.

type EvidenceStorage struct {
	artifacts map[string]*Evidence

	mutex sync.RWMutex
}

// IncidentMetrics tracks incident response metrics.

type IncidentMetrics struct {
	TotalIncidents int64 `json:"total_incidents"`

	OpenIncidents int64 `json:"open_incidents"`

	ResolvedIncidents int64 `json:"resolved_incidents"`

	IncidentsBySeverity map[string]int64 `json:"incidents_by_severity"`

	IncidentsByCategory map[string]int64 `json:"incidents_by_category"`

	MTTR time.Duration `json:"mttr"` // Mean Time To Resolution

	MTTA time.Duration `json:"mtta"` // Mean Time To Acknowledgment

	AutomatedActions int64 `json:"automated_actions"`

	EscalatedIncidents int64 `json:"escalated_incidents"`

	LastIncidentTime time.Time `json:"last_incident_time"`

	mutex sync.RWMutex
}

// NewIncidentResponse creates a new incident response system.

func NewIncidentResponse(config *IncidentConfig) (*IncidentResponse, error) {
	if config == nil {
		config = getDefaultIncidentConfig()
	}

	ir := &IncidentResponse{
		config: config,

		logger: slog.Default().With("component", "incident-response"),

		incidents: make(map[string]*SecurityIncident),

		playbooks: make(map[string]*ResponsePlaybook),

		metrics: &IncidentMetrics{
			IncidentsBySeverity: make(map[string]int64),

			IncidentsByCategory: make(map[string]int64),
		},

		stopChan: make(chan struct{}),
	}

	// Initialize components.

	ir.escalation = NewEscalationEngine(config)

	ir.forensics = NewForensicsCollector(config)

	// Load default playbooks.

	ir.loadDefaultPlaybooks()

	// Start background processes.

	go ir.startIncidentMonitoring()

	go ir.startEscalationMonitoring()

	return ir, nil
}

// getDefaultIncidentConfig returns default incident response configuration.

func getDefaultIncidentConfig() *IncidentConfig {
	return &IncidentConfig{
		EnableAutoResponse: true,

		AutoResponseThreshold: "High",

		MaxAutoActions: 10,

		IncidentRetention: 90 * 24 * time.Hour, // 90 days

		EscalationTimeout: 30 * time.Minute,

		ForensicsEnabled: true,

		NotificationConfig: &NotificationConfig{
			EnableEmail: true,

			EnableSlack: true,

			Recipients: []string{"security@company.com"},
		},
	}
}

// CreateIncident creates a new security incident.

func (ir *IncidentResponse) CreateIncident(ctx context.Context, request *CreateIncidentRequest) (*SecurityIncident, error) {
	incident := &SecurityIncident{
		ID: generateIncidentID(),

		Title: request.Title,

		Description: request.Description,

		Severity: request.Severity,

		Status: "Open",

		Category: request.Category,

		Source: request.Source,

		DetectedAt: time.Now(),

		Tags: request.Tags,

		Evidence: make([]*Evidence, 0),

		Timeline: make([]*TimelineEvent, 0),

		Actions: make([]*ResponseAction, 0),

		Artifacts: json.RawMessage(`{}`),

		Impact: request.Impact,
	}

	// Add initial timeline event.

	incident.Timeline = append(incident.Timeline, &TimelineEvent{
		Timestamp: time.Now(),

		Type: "created",

		Description: "Incident created",

		Actor: "system",

		Automated: true,
	})

	// Store incident.

	ir.mutex.Lock()

	ir.incidents[incident.ID] = incident

	ir.mutex.Unlock()

	// Update metrics.

	ir.updateMetrics(incident, "created")

	// Trigger automated response if enabled.

	if ir.config.EnableAutoResponse {
		go ir.triggerAutomatedResponse(ctx, incident)
	}

	// Send notifications.

	go ir.sendIncidentNotification(ctx, incident, "created")

	// Start evidence collection.

	if ir.config.ForensicsEnabled {
		go ir.forensics.CollectEvidence(ctx, incident)
	}

	ir.logger.Info("Security incident created",

		"incident_id", incident.ID,

		"severity", incident.Severity,

		"category", incident.Category)

	return incident, nil
}

// CreateIncidentRequest represents an incident creation request.

type CreateIncidentRequest struct {
	Title string `json:"title"`

	Description string `json:"description"`

	Severity string `json:"severity"`

	Category string `json:"category"`

	Source string `json:"source"`

	Tags []string `json:"tags"`

	Impact *ImpactAssessment `json:"impact"`

	Evidence []*Evidence `json:"evidence,omitempty"`
}

// UpdateIncident updates an existing incident.

func (ir *IncidentResponse) UpdateIncident(ctx context.Context, incidentID string, updates *IncidentUpdate) error {
	ir.mutex.Lock()

	incident, exists := ir.incidents[incidentID]

	if !exists {

		ir.mutex.Unlock()

		return fmt.Errorf("incident not found: %s", incidentID)

	}

	// Apply updates.

	if updates.Status != "" && updates.Status != incident.Status {

		incident.Status = updates.Status

		incident.Timeline = append(incident.Timeline, &TimelineEvent{
			Timestamp: time.Now(),

			Type: "status_changed",

			Description: fmt.Sprintf("Status changed to %s", updates.Status),

			Actor: updates.UpdatedBy,

			Automated: false,
		})

		if updates.Status == "Acknowledged" && incident.AcknowledgedAt == nil {

			now := time.Now()

			incident.AcknowledgedAt = &now

		} else if updates.Status == "Resolved" && incident.ResolvedAt == nil {

			now := time.Now()

			incident.ResolvedAt = &now

		}

	}

	if updates.Assignee != "" {

		incident.Assignee = updates.Assignee

		incident.Timeline = append(incident.Timeline, &TimelineEvent{
			Timestamp: time.Now(),

			Type: "assigned",

			Description: fmt.Sprintf("Assigned to %s", updates.Assignee),

			Actor: updates.UpdatedBy,

			Automated: false,
		})

	}

	if updates.Severity != "" && updates.Severity != incident.Severity {

		oldSeverity := incident.Severity

		incident.Severity = updates.Severity

		incident.Timeline = append(incident.Timeline, &TimelineEvent{
			Timestamp: time.Now(),

			Type: "severity_changed",

			Description: fmt.Sprintf("Severity changed from %s to %s", oldSeverity, updates.Severity),

			Actor: updates.UpdatedBy,

			Automated: false,
		})

	}

	if len(updates.Tags) > 0 {
		incident.Tags = append(incident.Tags, updates.Tags...)
	}

	ir.mutex.Unlock()

	// Update metrics.

	ir.updateMetrics(incident, "updated")

	// Send notification if significant change.

	if updates.Status != "" || updates.Severity != "" {
		go ir.sendIncidentNotification(ctx, incident, "updated")
	}

	return nil
}

// IncidentUpdate represents incident update information.

type IncidentUpdate struct {
	Status string `json:"status,omitempty"`

	Assignee string `json:"assignee,omitempty"`

	Severity string `json:"severity,omitempty"`

	Tags []string `json:"tags,omitempty"`

	UpdatedBy string `json:"updated_by"`
}

// GetIncident retrieves an incident by ID.

func (ir *IncidentResponse) GetIncident(incidentID string) (*SecurityIncident, error) {
	ir.mutex.RLock()

	defer ir.mutex.RUnlock()

	incident, exists := ir.incidents[incidentID]

	if !exists {
		return nil, fmt.Errorf("incident not found: %s", incidentID)
	}

	return incident, nil
}

// ListIncidents lists incidents with optional filtering.

func (ir *IncidentResponse) ListIncidents(filter *IncidentFilter) ([]*SecurityIncident, error) {
	ir.mutex.RLock()

	defer ir.mutex.RUnlock()

	var incidents []*SecurityIncident

	for _, incident := range ir.incidents {
		if filter == nil || ir.matchesFilter(incident, filter) {
			incidents = append(incidents, incident)
		}
	}

	// Sort by detection time (newest first).

	sort.Slice(incidents, func(i, j int) bool {
		return incidents[i].DetectedAt.After(incidents[j].DetectedAt)
	})

	// Apply limit if specified.

	if filter != nil && filter.Limit > 0 && len(incidents) > filter.Limit {
		incidents = incidents[:filter.Limit]
	}

	return incidents, nil
}

// IncidentFilter represents incident filtering criteria.

type IncidentFilter struct {
	Severity string `json:"severity,omitempty"`

	Status string `json:"status,omitempty"`

	Category string `json:"category,omitempty"`

	Assignee string `json:"assignee,omitempty"`

	Source string `json:"source,omitempty"`

	FromDate time.Time `json:"from_date,omitempty"`

	ToDate time.Time `json:"to_date,omitempty"`

	Tags []string `json:"tags,omitempty"`

	Limit int `json:"limit,omitempty"`
}

// AddEvidence adds evidence to an incident.

func (ir *IncidentResponse) AddEvidence(incidentID string, evidence *Evidence) error {
	ir.mutex.Lock()

	defer ir.mutex.Unlock()

	incident, exists := ir.incidents[incidentID]

	if !exists {
		return fmt.Errorf("incident not found: %s", incidentID)
	}

	evidence.ID = generateEvidenceID()

	evidence.Hash = ir.calculateEvidenceHash(evidence)

	incident.Evidence = append(incident.Evidence, evidence)

	incident.Timeline = append(incident.Timeline, &TimelineEvent{
		Timestamp: time.Now(),

		Type: "evidence_added",

		Description: fmt.Sprintf("Evidence added: %s", evidence.Type),

		Actor: "system",

		Automated: true,
	})

	return nil
}

// ExecutePlaybook executes a response playbook.

func (ir *IncidentResponse) ExecutePlaybook(ctx context.Context, incidentID, playbookID string) error {
	incident, err := ir.GetIncident(incidentID)
	if err != nil {
		return err
	}

	playbook, exists := ir.playbooks[playbookID]

	if !exists {
		return fmt.Errorf("playbook not found: %s", playbookID)
	}

	ir.logger.Info("Executing playbook",

		"incident_id", incidentID,

		"playbook_id", playbookID,

		"playbook_name", playbook.Name)

	// Execute playbook actions.

	for _, action := range playbook.Actions {

		responseAction := &ResponseAction{
			ID: generateActionID(),

			Type: action.Type,

			Description: action.Description,

			Status: "executing",

			Parameters: action.Parameters,

			Automated: true,
		}

		now := time.Now()

		responseAction.ExecutedAt = &now

		// Add to incident.

		ir.mutex.Lock()

		incident.Actions = append(incident.Actions, responseAction)

		ir.mutex.Unlock()

		// Execute the action.

		err := ir.executeAction(ctx, action, incident)

		if err != nil {

			responseAction.Status = "failed"

			responseAction.Result = err.Error()

			ir.logger.Error("Playbook action failed", "error", err, "action", action.Type)

			if action.OnFailure == "abort" {
				break
			}

		} else {

			responseAction.Status = "completed"

			responseAction.Result = "success"

		}

		now = time.Now()

		responseAction.CompletedAt = &now

	}

	// Update playbook last executed time.

	now := time.Now()

	playbook.LastExecuted = &now

	return nil
}

// triggerAutomatedResponse triggers automated response based on incident.

func (ir *IncidentResponse) triggerAutomatedResponse(ctx context.Context, incident *SecurityIncident) {
	if !ir.shouldAutoRespond(incident) {
		return
	}

	ir.logger.Info("Triggering automated response", "incident_id", incident.ID)

	// Find matching playbooks.

	for _, playbook := range ir.playbooks {

		if !playbook.Enabled {
			continue
		}

		if ir.playbookMatches(incident, playbook) {
			if err := ir.ExecutePlaybook(ctx, incident.ID, playbook.ID); err != nil {
				ir.logger.Error("Failed to execute playbook", "error", err, "playbook", playbook.Name)
			}
		}

	}
}

// Helper methods.

func (ir *IncidentResponse) shouldAutoRespond(incident *SecurityIncident) bool {
	if !ir.config.EnableAutoResponse {
		return false
	}

	severityWeight := map[string]int{
		"Critical": 4,

		"High": 3,

		"Medium": 2,

		"Low": 1,
	}

	thresholdWeight := severityWeight[ir.config.AutoResponseThreshold]

	incidentWeight := severityWeight[incident.Severity]

	return incidentWeight >= thresholdWeight
}

func (ir *IncidentResponse) playbookMatches(incident *SecurityIncident, playbook *ResponsePlaybook) bool {
	for _, trigger := range playbook.Triggers {
		if ir.evaluateTrigger(incident, trigger) {
			return true
		}
	}

	return false
}

func (ir *IncidentResponse) evaluateTrigger(incident *SecurityIncident, trigger *PlaybookTrigger) bool {
	// Unmarshal conditions from JSON
	var conditions map[string]interface{}
	if len(trigger.Conditions) > 0 {
		if err := json.Unmarshal(trigger.Conditions, &conditions); err != nil {
			return false
		}
	}

	switch trigger.Type {

	case "severity":
		if severity, ok := conditions["severity"].(string); ok {
			return incident.Severity == severity
		}

	case "category":

		if category, ok := conditions["category"].(string); ok {
			return incident.Category == category
		}

	case "source":

		if source, ok := conditions["source"].(string); ok {
			return incident.Source == source
		}

	}

	return false
}

func (ir *IncidentResponse) executeAction(ctx context.Context, action *PlaybookAction, incident *SecurityIncident) error {
	// Unmarshal parameters from JSON
	var params map[string]interface{}
	if len(action.Parameters) > 0 {
		if err := json.Unmarshal(action.Parameters, &params); err != nil {
			return fmt.Errorf("failed to unmarshal action parameters: %w", err)
		}
	}

	switch action.Type {

	case "isolate_system":
		return ir.isolateSystem(ctx, params, incident)

	case "block_ip":
		return ir.blockIP(ctx, params, incident)

	case "disable_user":
		return ir.disableUser(ctx, params, incident)

	case "create_ticket":
		return ir.createTicket(ctx, params, incident)

	case "send_notification":
		return ir.sendNotification(ctx, params, incident)

	default:

		return fmt.Errorf("unknown action type: %s", action.Type)

	}
}

func (ir *IncidentResponse) isolateSystem(ctx context.Context, params map[string]interface{}, incident *SecurityIncident) error {
	// Implementation would isolate affected systems.

	ir.logger.Info("Isolating system", "incident_id", incident.ID)

	return nil
}

func (ir *IncidentResponse) blockIP(ctx context.Context, params map[string]interface{}, incident *SecurityIncident) error {
	// Implementation would block malicious IPs.

	ir.logger.Info("Blocking IP", "incident_id", incident.ID)

	return nil
}

func (ir *IncidentResponse) disableUser(ctx context.Context, params map[string]interface{}, incident *SecurityIncident) error {
	// Implementation would disable compromised user accounts.

	ir.logger.Info("Disabling user", "incident_id", incident.ID)

	return nil
}

func (ir *IncidentResponse) createTicket(ctx context.Context, params map[string]interface{}, incident *SecurityIncident) error {
	// Implementation would create support ticket.

	ir.logger.Info("Creating ticket", "incident_id", incident.ID)

	return nil
}

func (ir *IncidentResponse) sendNotification(ctx context.Context, params map[string]interface{}, incident *SecurityIncident) error {
	// Implementation would send notifications.

	ir.logger.Info("Sending notification", "incident_id", incident.ID)

	return nil
}

func (ir *IncidentResponse) matchesFilter(incident *SecurityIncident, filter *IncidentFilter) bool {
	if filter.Severity != "" && incident.Severity != filter.Severity {
		return false
	}

	if filter.Status != "" && incident.Status != filter.Status {
		return false
	}

	if filter.Category != "" && incident.Category != filter.Category {
		return false
	}

	if filter.Assignee != "" && incident.Assignee != filter.Assignee {
		return false
	}

	if filter.Source != "" && incident.Source != filter.Source {
		return false
	}

	if !filter.FromDate.IsZero() && incident.DetectedAt.Before(filter.FromDate) {
		return false
	}

	if !filter.ToDate.IsZero() && incident.DetectedAt.After(filter.ToDate) {
		return false
	}

	if len(filter.Tags) > 0 {

		tagMap := make(map[string]bool)

		for _, tag := range incident.Tags {
			tagMap[tag] = true
		}

		for _, filterTag := range filter.Tags {
			if !tagMap[filterTag] {
				return false
			}
		}

	}

	return true
}

func (ir *IncidentResponse) updateMetrics(incident *SecurityIncident, action string) {
	ir.metrics.mutex.Lock()

	defer ir.metrics.mutex.Unlock()

	switch action {

	case "created":

		ir.metrics.TotalIncidents++

		ir.metrics.OpenIncidents++

		ir.metrics.IncidentsBySeverity[incident.Severity]++

		ir.metrics.IncidentsByCategory[incident.Category]++

		ir.metrics.LastIncidentTime = incident.DetectedAt

	case "resolved":

		ir.metrics.OpenIncidents--

		ir.metrics.ResolvedIncidents++

		// Calculate MTTR.

		if incident.ResolvedAt != nil {

			resolution_time := incident.ResolvedAt.Sub(incident.DetectedAt)

			ir.metrics.MTTR = (ir.metrics.MTTR*time.Duration(ir.metrics.ResolvedIncidents-1) + resolution_time) / time.Duration(ir.metrics.ResolvedIncidents)

		}

		// Calculate MTTA.

		if incident.AcknowledgedAt != nil {

			ack_time := incident.AcknowledgedAt.Sub(incident.DetectedAt)

			ir.metrics.MTTA = (ir.metrics.MTTA*time.Duration(ir.metrics.ResolvedIncidents-1) + ack_time) / time.Duration(ir.metrics.ResolvedIncidents)

		}

	}
}

func (ir *IncidentResponse) sendIncidentNotification(ctx context.Context, incident *SecurityIncident, action string) {
	if ir.config.NotificationConfig == nil {
		return
	}

	message := fmt.Sprintf("Security Incident %s: %s\nSeverity: %s\nCategory: %s\nStatus: %s",

		action, incident.Title, incident.Severity, incident.Category, incident.Status)

	// Send email notification.

	if ir.config.NotificationConfig.EnableEmail {
		ir.sendEmailNotification(ctx, incident, message)
	}

	// Send Slack notification.

	if ir.config.NotificationConfig.EnableSlack {
		ir.sendSlackNotification(ctx, incident, message)
	}

	// Send SMS notification (for critical incidents).

	if ir.config.NotificationConfig.EnableSMS && incident.Severity == "Critical" {
		ir.sendSMSNotification(ctx, incident, message)
	}

	// Send PagerDuty alert.

	if ir.config.NotificationConfig.EnablePagerDuty && (incident.Severity == "Critical" || incident.Severity == "High") {
		ir.sendPagerDutyAlert(ctx, incident, message)
	}
}

func (ir *IncidentResponse) sendEmailNotification(ctx context.Context, incident *SecurityIncident, message string) {
	// Implementation would send email.

	ir.logger.Info("Email notification sent", "incident_id", incident.ID)
}

func (ir *IncidentResponse) sendSlackNotification(ctx context.Context, incident *SecurityIncident, message string) {
	// Implementation would send Slack message.

	ir.logger.Info("Slack notification sent", "incident_id", incident.ID)
}

func (ir *IncidentResponse) sendSMSNotification(ctx context.Context, incident *SecurityIncident, message string) {
	// Implementation would send SMS.

	ir.logger.Info("SMS notification sent", "incident_id", incident.ID)
}

func (ir *IncidentResponse) sendPagerDutyAlert(ctx context.Context, incident *SecurityIncident, message string) {
	// Implementation would send PagerDuty alert.

	ir.logger.Info("PagerDuty alert sent", "incident_id", incident.ID)
}

func (ir *IncidentResponse) loadDefaultPlaybooks() {
	// Load default response playbooks.

	playbooks := []*ResponsePlaybook{
		{
			ID: "malware_detected",

			Name: "Malware Detection Response",

			Description: "Automated response to malware detection",

			Triggers: []*PlaybookTrigger{
				{
					Type: "category",

					Conditions: json.RawMessage(`{}`),
				},
			},

			Actions: []*PlaybookAction{
				{
					ID: "isolate_host",

					Type: "isolate_system",

					Description: "Isolate infected host",

					Parameters: json.RawMessage(`{}`),

					Timeout: 5 * time.Minute,
				},

				{
					ID: "collect_evidence",

					Type: "collect_evidence",

					Description: "Collect forensic evidence",

					Parameters: json.RawMessage(`{}`),

					Timeout: 30 * time.Minute,
				},
			},

			Enabled: true,

			Priority: 1,
		},
	}

	for _, playbook := range playbooks {
		ir.playbooks[playbook.ID] = playbook
	}
}

func (ir *IncidentResponse) startIncidentMonitoring() {
	ticker := time.NewTicker(5 * time.Minute)

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			ir.monitorIncidents()

		case <-ir.stopChan:

			return

		}
	}
}

func (ir *IncidentResponse) startEscalationMonitoring() {
	ticker := time.NewTicker(1 * time.Minute)

	defer ticker.Stop()

	for {
		select {

		case <-ticker.C:

			ir.checkEscalations()

		case <-ir.stopChan:

			return

		}
	}
}

func (ir *IncidentResponse) monitorIncidents() {
	// Monitor incidents for SLA violations, stale incidents, etc.
}

func (ir *IncidentResponse) checkEscalations() {
	// Check for incidents that need escalation.
}

func (ir *IncidentResponse) calculateEvidenceHash(evidence *Evidence) string {
	data, _ := json.Marshal(evidence.Data)

	hash := sha256.Sum256(data)

	return hex.EncodeToString(hash[:])
}

// Utility functions.

func generateIncidentID() string {
	return fmt.Sprintf("INC-%d", time.Now().UnixNano())
}

func generateEvidenceID() string {
	return fmt.Sprintf("EVD-%d", time.Now().UnixNano())
}

func generateActionID() string {
	return fmt.Sprintf("ACT-%d", time.Now().UnixNano())
}

// GetMetrics returns current incident response metrics.

func (ir *IncidentResponse) GetMetrics() *IncidentMetrics {
	ir.metrics.mutex.RLock()

	defer ir.metrics.mutex.RUnlock()

	return &IncidentMetrics{
		TotalIncidents: ir.metrics.TotalIncidents,

		OpenIncidents: ir.metrics.OpenIncidents,

		ResolvedIncidents: ir.metrics.ResolvedIncidents,

		IncidentsBySeverity: copyStringInt64Map(ir.metrics.IncidentsBySeverity),

		IncidentsByCategory: copyStringInt64Map(ir.metrics.IncidentsByCategory),

		MTTR: ir.metrics.MTTR,

		MTTA: ir.metrics.MTTA,

		AutomatedActions: ir.metrics.AutomatedActions,

		EscalatedIncidents: ir.metrics.EscalatedIncidents,

		LastIncidentTime: ir.metrics.LastIncidentTime,
	}
}

// Close shuts down the incident response system.

func (ir *IncidentResponse) Close() error {
	close(ir.stopChan)

	return nil
}

// NewEscalationEngine creates a new escalation engine.

func NewEscalationEngine(config *IncidentConfig) *EscalationEngine {
	return &EscalationEngine{
		config: config,

		logger: slog.Default().With("component", "escalation-engine"),

		rules: make([]*EscalationRule, 0),
	}
}

// NewForensicsCollector creates a new forensics collector.

func NewForensicsCollector(config *IncidentConfig) *ForensicsCollector {
	return &ForensicsCollector{
		config: config,

		logger: slog.Default().With("component", "forensics-collector"),

		storage: &EvidenceStorage{
			artifacts: make(map[string]*Evidence),
		},
	}
}

// CollectEvidence collects evidence for an incident.

func (fc *ForensicsCollector) CollectEvidence(ctx context.Context, incident *SecurityIncident) error {
	fc.logger.Info("Starting evidence collection", "incident_id", incident.ID)

	// Collect different types of evidence.

	evidenceTypes := []string{"logs", "network", "system", "memory"}

	for _, evidenceType := range evidenceTypes {

		evidence, err := fc.collectEvidenceType(ctx, incident, evidenceType)
		if err != nil {

			fc.logger.Error("Failed to collect evidence", "type", evidenceType, "error", err)

			continue

		}

		if evidence != nil {

			fc.storage.mutex.Lock()

			fc.storage.artifacts[evidence.ID] = evidence

			fc.storage.mutex.Unlock()

		}

	}

	return nil
}

func (fc *ForensicsCollector) collectEvidenceType(ctx context.Context, incident *SecurityIncident, evidenceType string) (*Evidence, error) {
	evidence := &Evidence{
		ID:          generateEvidenceID(),
		Type:        evidenceType,
		Source:      "forensics-collector",
		Timestamp:   time.Now(),
		Description: fmt.Sprintf("Collected %s evidence for incident %s", evidenceType, incident.ID),
		Collected:   true,
	}

	// Build data map
	dataMap := make(map[string]interface{})

	switch evidenceType {
	case "logs":
		// Collect relevant logs.
		dataMap["log_entries"] = []string{"Sample log entry 1", "Sample log entry 2"}

	case "network":
		// Collect network data.
		dataMap["connections"] = []string{"192.168.1.100:443", "10.0.0.50:80"}

	case "system":
		// Collect system information.
		dataMap["processes"] = []string{"process1", "process2"}

	case "memory":
		// Collect memory dumps (simulated).
		dataMap["memory_regions"] = []string{"region1", "region2"}

	}

	// Marshal data map to JSON
	if dataBytes, err := json.Marshal(dataMap); err == nil {
		evidence.Data = dataBytes
	} else {
		evidence.Data = json.RawMessage(`{}`)
	}

	return evidence, nil
}

// HandleWebhook handles incoming webhook events with HMAC signature verification.

func (ir *IncidentResponse) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	// Read the request body.

	body, err := io.ReadAll(r.Body)
	if err != nil {

		ir.logger.Error("Failed to read webhook body", "error", err)

		http.Error(w, "Failed to read request body", http.StatusBadRequest)

		return

	}

	defer r.Body.Close()

	// Get the signature from headers.

	signatureHeader := r.Header.Get("X-Hub-Signature-256")

	if signatureHeader == "" {

		ir.logger.Warn("Missing X-Hub-Signature-256 header")

		http.Error(w, "Missing signature header", http.StatusUnauthorized)

		return

	}

	// Verify the signature.

	if !ir.verifyWebhookSignature(body, signatureHeader) {

		ir.logger.Warn("Invalid webhook signature", "signature", signatureHeader)

		http.Error(w, "Invalid signature", http.StatusUnauthorized)

		return

	}

	// Process the webhook payload.

	if err := ir.processWebhookPayload(r.Context(), body); err != nil {

		ir.logger.Error("Failed to process webhook payload", "error", err)

		http.Error(w, "Failed to process webhook", http.StatusInternalServerError)

		return

	}

	// Return success response.

	w.WriteHeader(http.StatusOK)

	w.Write([]byte("Webhook processed successfully"))

	ir.logger.Info("Webhook processed successfully")
}

// verifyWebhookSignature verifies the HMAC signature of the webhook payload.

func (ir *IncidentResponse) verifyWebhookSignature(payload []byte, signatureHeader string) bool {
	if ir.config.WebhookSecret == "" {

		ir.logger.Warn("Webhook secret not configured")

		return false

	}

	// Parse the signature header (format: "sha256=<hex_encoded_signature>").

	if !strings.HasPrefix(signatureHeader, "sha256=") {

		ir.logger.Warn("Invalid signature header format", "header", signatureHeader)

		return false

	}

	receivedSignature := signatureHeader[7:] // Remove "sha256=" prefix

	// Compute HMAC-SHA256 of the payload.

	mac := hmac.New(sha256.New, []byte(ir.config.WebhookSecret))

	mac.Write(payload)

	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Use constant time comparison to prevent timing attacks.

	return hmac.Equal([]byte(receivedSignature), []byte(expectedSignature))
}

// processWebhookPayload processes the validated webhook payload.

func (ir *IncidentResponse) processWebhookPayload(ctx context.Context, payload []byte) error {
	// Parse the JSON payload.

	var webhookData map[string]interface{}

	if err := json.Unmarshal(payload, &webhookData); err != nil {

		ir.logger.Error("Failed to parse webhook JSON", "error", err)

		return fmt.Errorf("invalid JSON payload: %w", err)

	}

	// Log webhook receipt.

	ir.logger.Info("Processing webhook payload",

		"content_length", len(payload),

		"webhook_type", webhookData["type"])

	// Process based on webhook type.

	webhookType, ok := webhookData["type"].(string)

	if !ok {

		ir.logger.Warn("Missing or invalid webhook type")

		return fmt.Errorf("missing or invalid webhook type")

	}

	switch webhookType {

	case "security_alert":

		return ir.processSecurityAlert(ctx, webhookData)

	case "incident_update":

		return ir.processIncidentUpdate(ctx, webhookData)

	case "threat_intelligence":

		return ir.processThreatIntelligence(ctx, webhookData)

	default:

		ir.logger.Info("Unknown webhook type, processing as generic event", "type", webhookType)

		return ir.processGenericWebhook(ctx, webhookData)

	}
}

// processSecurityAlert processes security alert webhooks.

func (ir *IncidentResponse) processSecurityAlert(ctx context.Context, data map[string]interface{}) error {
	alert, ok := data["alert"].(map[string]interface{})

	if !ok {
		return fmt.Errorf("invalid security alert format")
	}

	// Create incident from security alert.

	request := &CreateIncidentRequest{
		Title: getString(alert, "title", "Security Alert"),

		Description: getString(alert, "description", "Security alert received via webhook"),

		Severity: getString(alert, "severity", "Medium"),

		Category: getString(alert, "category", "alert"),

		Source: "webhook",

		Tags: []string{"webhook", "external"},
	}

	if _, err := ir.CreateIncident(ctx, request); err != nil {

		ir.logger.Error("Failed to create incident from security alert", "error", err)

		return err

	}

	ir.logger.Info("Created incident from security alert webhook")

	return nil
}

// processIncidentUpdate processes incident update webhooks.

func (ir *IncidentResponse) processIncidentUpdate(ctx context.Context, data map[string]interface{}) error {
	incidentID := getString(data, "incident_id", "")

	if incidentID == "" {
		return fmt.Errorf("missing incident_id in webhook payload")
	}

	updates := &IncidentUpdate{
		Status: getString(data, "status", ""),

		Assignee: getString(data, "assignee", ""),

		Severity: getString(data, "severity", ""),

		UpdatedBy: "webhook",
	}

	if err := ir.UpdateIncident(ctx, incidentID, updates); err != nil {

		ir.logger.Error("Failed to update incident from webhook", "incident_id", incidentID, "error", err)

		return err

	}

	ir.logger.Info("Updated incident from webhook", "incident_id", incidentID)

	return nil
}

// processThreatIntelligence processes threat intelligence webhooks.

func (ir *IncidentResponse) processThreatIntelligence(ctx context.Context, data map[string]interface{}) error {
	threatType := getString(data, "threat_type", "unknown")

	indicators := getStringSlice(data, "indicators")

	ir.logger.Info("Processing threat intelligence",

		"threat_type", threatType,

		"indicator_count", len(indicators))

	// Here you would typically:.

	// 1. Update threat intelligence database.

	// 2. Check for matches against current incidents.

	// 3. Create new incidents if critical threats detected.

	return nil
}

// processGenericWebhook processes unknown webhook types.

func (ir *IncidentResponse) processGenericWebhook(ctx context.Context, data map[string]interface{}) error {
	ir.logger.Info("Processing generic webhook", "keys", getMapKeys(data))

	// Store the webhook data for later analysis or create a low-priority incident.

	return nil
}

// Helper functions for webhook processing.

func getString(data map[string]interface{}, key, defaultValue string) string {
	if val, ok := data[key].(string); ok {
		return val
	}

	return defaultValue
}

func getStringSlice(data map[string]interface{}, key string) []string {
	if val, ok := data[key].([]interface{}); ok {

		result := make([]string, 0, len(val))

		for _, item := range val {
			if str, ok := item.(string); ok {
				result = append(result, str)
			}
		}

		return result

	}

	return []string{}
}

func getMapKeys(data map[string]interface{}) []string {
	keys := make([]string, 0, len(data))

	for key := range data {
		keys = append(keys, key)
	}

	return keys
}

// copyStringInt64Map creates a deep copy of a map[string]int64.

func copyStringInt64Map(original map[string]int64) map[string]int64 {
	if original == nil {
		return nil
	}

	result := make(map[string]int64, len(original))

	for k, v := range original {
		result[k] = v
	}

	return result
}

