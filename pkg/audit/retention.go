package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

var (
	retentionOperationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "audit_retention_operations_total",

		Help: "Total number of retention operations performed",
	}, []string{"operation", "policy", "status"})

	retentionEventsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "audit_retention_events_processed_total",

		Help: "Total number of events processed by retention policies",
	}, []string{"policy", "action"})

	retentionStorageUsed = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "audit_retention_storage_bytes",

		Help: "Storage used by audit retention policies",
	}, []string{"policy", "storage_tier"})
)

// RetentionManager manages audit event retention policies.

type RetentionManager struct {
	mutex sync.RWMutex

	logger logr.Logger

	config *RetentionConfig

	policies map[string]*RetentionPolicy

	// Storage tracking.

	storageStats map[string]StorageStats

	// Background processing.

	running bool

	ticker *time.Ticker

	stopChannel chan bool
}

// RetentionConfig holds configuration for retention management.

type RetentionConfig struct {
	// Global settings.

	DefaultRetention time.Duration `json:"default_retention" yaml:"default_retention"`

	CheckInterval time.Duration `json:"check_interval" yaml:"check_interval"`

	BatchSize int `json:"batch_size" yaml:"batch_size"`

	ComplianceMode []ComplianceStandard `json:"compliance_mode" yaml:"compliance_mode"`

	// Policy-specific settings.

	Policies map[string]RetentionPolicyConfig `json:"policies" yaml:"policies"`

	// Storage tiers.

	StorageTiers []StorageTier `json:"storage_tiers" yaml:"storage_tiers"`

	// Archival settings.

	ArchivalEnabled bool `json:"archival_enabled" yaml:"archival_enabled"`

	ArchivalDestination string `json:"archival_destination" yaml:"archival_destination"`

	CompressionEnabled bool `json:"compression_enabled" yaml:"compression_enabled"`

	EncryptionEnabled bool `json:"encryption_enabled" yaml:"encryption_enabled"`
}

// RetentionPolicyConfig defines configuration for a specific retention policy.

type RetentionPolicyConfig struct {
	Name string `json:"name" yaml:"name"`

	Description string `json:"description" yaml:"description"`

	RetentionPeriod time.Duration `json:"retention_period" yaml:"retention_period"`

	EventTypes []EventType `json:"event_types" yaml:"event_types"`

	Severity Severity `json:"min_severity" yaml:"min_severity"`

	Components []string `json:"components" yaml:"components"`

	ComplianceStandard ComplianceStandard `json:"compliance_standard" yaml:"compliance_standard"`

	// Storage lifecycle.

	StorageTransitions []StorageTransition `json:"storage_transitions" yaml:"storage_transitions"`

	// Actions.

	ArchiveBeforeDelete bool `json:"archive_before_delete" yaml:"archive_before_delete"`

	NotifyOnDeletion bool `json:"notify_on_deletion" yaml:"notify_on_deletion"`

	RequireApproval bool `json:"require_approval" yaml:"require_approval"`

	// Legal hold.

	LegalHoldExempt bool `json:"legal_hold_exempt" yaml:"legal_hold_exempt"`
}

// RetentionPolicy represents an active retention policy.

type RetentionPolicy struct {
	config RetentionPolicyConfig

	lastExecution time.Time

	eventsRetained int64

	eventsArchived int64

	eventsDeleted int64

	storageUsage int64

	// Legal holds.

	legalHolds map[string]LegalHold

	legalHoldMutex sync.RWMutex
}

// StorageTier represents a storage tier with different characteristics.

type StorageTier struct {
	Name string `json:"name" yaml:"name"`

	Description string `json:"description" yaml:"description"`

	CostPerGB float64 `json:"cost_per_gb" yaml:"cost_per_gb"`

	AccessLatency time.Duration `json:"access_latency" yaml:"access_latency"`

	Durability string `json:"durability" yaml:"durability"`

	AvailabilityZones int `json:"availability_zones" yaml:"availability_zones"`
}

// StorageTransition defines how events move between storage tiers.

type StorageTransition struct {
	FromTier string `json:"from_tier" yaml:"from_tier"`

	ToTier string `json:"to_tier" yaml:"to_tier"`

	AfterDays int `json:"after_days" yaml:"after_days"`

	Conditions []string `json:"conditions" yaml:"conditions"`
}

// StorageStats tracks storage usage statistics.

type StorageStats struct {
	TotalEvents int64 `json:"total_events"`

	TotalSize int64 `json:"total_size_bytes"`

	TierDistribution map[string]StorageTierStat `json:"tier_distribution"`

	OldestEvent time.Time `json:"oldest_event"`

	NewestEvent time.Time `json:"newest_event"`

	LastUpdated time.Time `json:"last_updated"`
}

// StorageTierStat represents statistics for a specific storage tier.

type StorageTierStat struct {
	EventCount int64 `json:"event_count"`

	SizeBytes int64 `json:"size_bytes"`

	CostUSD float64 `json:"cost_usd"`
}

// LegalHold represents a legal hold on audit events.

type LegalHold struct {
	HoldID string `json:"hold_id"`

	Title string `json:"title"`

	Description string `json:"description"`

	StartDate time.Time `json:"start_date"`

	EndDate *time.Time `json:"end_date,omitempty"`

	Custodian string `json:"custodian"`

	Matter string `json:"matter"`

	// Scope.

	EventTypes []EventType `json:"event_types"`

	UserIDs []string `json:"user_ids"`

	DateRange *struct {
		StartTime time.Time `json:"start_time"`

		EndTime time.Time `json:"end_time"`
	} `json:"date_range,omitempty"`

	Status string `json:"status"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// RetentionAction represents an action taken by the retention policy.

type RetentionAction struct {
	ActionID string `json:"action_id"`

	PolicyName string `json:"policy_name"`

	ActionType string `json:"action_type"` // archive, delete, transition

	EventCount int64 `json:"event_count"`

	ExecutedAt time.Time `json:"executed_at"`

	ExecutedBy string `json:"executed_by"`

	Status string `json:"status"`

	Details json.RawMessage `json:"details,omitempty"`
}

// NewRetentionManager creates a new retention manager.

func NewRetentionManager(config *RetentionConfig) *RetentionManager {
	if config == nil {
		config = DefaultRetentionConfig()
	}

	rm := &RetentionManager{
		logger: log.Log.WithName("retention-manager"),

		config: config,

		policies: make(map[string]*RetentionPolicy),

		storageStats: make(map[string]StorageStats),

		stopChannel: make(chan bool),
	}

	// Initialize retention policies.

	for name, policyConfig := range config.Policies {

		policy := &RetentionPolicy{
			config: policyConfig,

			legalHolds: make(map[string]LegalHold),
		}

		rm.policies[name] = policy

	}

	return rm
}

// Start begins the retention manager background processing.

func (rm *RetentionManager) Start(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	rm.mutex.Lock()

	if rm.running {

		rm.mutex.Unlock()

		return

	}

	rm.running = true

	rm.mutex.Unlock()

	interval := rm.config.CheckInterval

	if interval == 0 {
		interval = 1 * time.Hour // Default check every hour
	}

	rm.ticker = time.NewTicker(interval)

	defer rm.ticker.Stop()

	rm.logger.Info("Starting retention manager", "check_interval", interval)

	for {
		select {

		case <-rm.ticker.C:

			rm.executeRetentionPolicies(ctx)

		case <-rm.stopChannel:

			rm.logger.Info("Retention manager stopping")

			rm.mutex.Lock()

			rm.running = false

			rm.mutex.Unlock()

			return

		case <-ctx.Done():

			rm.logger.Info("Retention manager context cancelled")

			rm.mutex.Lock()

			rm.running = false

			rm.mutex.Unlock()

			return

		}
	}
}

// Stop gracefully stops the retention manager.

func (rm *RetentionManager) Stop() {
	rm.mutex.RLock()

	running := rm.running

	rm.mutex.RUnlock()

	if running {
		close(rm.stopChannel)
	}
}

// AddLegalHold adds a legal hold to prevent deletion of events.

func (rm *RetentionManager) AddLegalHold(policyName string, hold LegalHold) error {
	rm.mutex.Lock()

	defer rm.mutex.Unlock()

	policy, exists := rm.policies[policyName]

	if !exists {
		return fmt.Errorf("retention policy %s not found", policyName)
	}

	policy.legalHoldMutex.Lock()

	defer policy.legalHoldMutex.Unlock()

	hold.Status = "active"

	if hold.StartDate.IsZero() {
		hold.StartDate = time.Now().UTC()
	}

	policy.legalHolds[hold.HoldID] = hold

	rm.logger.Info("Legal hold added",

		"policy", policyName,

		"hold_id", hold.HoldID,

		"title", hold.Title,

		"custodian", hold.Custodian)

	return nil
}

// RemoveLegalHold removes a legal hold.

func (rm *RetentionManager) RemoveLegalHold(policyName, holdID string) error {
	rm.mutex.Lock()

	defer rm.mutex.Unlock()

	policy, exists := rm.policies[policyName]

	if !exists {
		return fmt.Errorf("retention policy %s not found", policyName)
	}

	policy.legalHoldMutex.Lock()

	defer policy.legalHoldMutex.Unlock()

	hold, exists := policy.legalHolds[holdID]

	if !exists {
		return fmt.Errorf("legal hold %s not found", holdID)
	}

	// Mark as released rather than deleting immediately.

	now := time.Now().UTC()

	hold.EndDate = &now

	hold.Status = "released"

	policy.legalHolds[holdID] = hold

	rm.logger.Info("Legal hold removed",

		"policy", policyName,

		"hold_id", holdID,

		"title", hold.Title)

	return nil
}

// GetRetentionStatus returns the current status of retention policies.

func (rm *RetentionManager) GetRetentionStatus() map[string]interface{} {
	rm.mutex.RLock()

	defer rm.mutex.RUnlock()

	status := make(map[string]interface{})

	for name, policy := range rm.policies {

		policyStatusMap := map[string]interface{}{}

		// Add legal hold information.

		activeLegalHolds := 0

		policy.legalHoldMutex.RLock()

		for _, hold := range policy.legalHolds {
			if hold.Status == "active" {
				activeLegalHolds++
			}
		}

		policy.legalHoldMutex.RUnlock()

		policyStatusMap["active_legal_holds"] = activeLegalHolds

		policyStatusBytes, _ := json.Marshal(policyStatusMap)
		status[name] = json.RawMessage(policyStatusBytes)

	}

	status["storage_stats"] = rm.storageStats

	status["total_policies"] = len(rm.policies)

	status["running"] = rm.running

	return status
}

// GetStorageReport generates a detailed storage report.

func (rm *RetentionManager) GetStorageReport() StorageReport {
	rm.mutex.RLock()

	defer rm.mutex.RUnlock()

	report := StorageReport{
		GeneratedAt: time.Now().UTC(),

		Policies: make(map[string]PolicyStorageReport),
	}

	var totalEvents int64

	var totalSize int64

	var totalCost float64

	for name, stats := range rm.storageStats {

		policyReport := PolicyStorageReport{
			EventCount: stats.TotalEvents,

			StorageSize: stats.TotalSize,

			TierDistribution: stats.TierDistribution,

			OldestEvent: stats.OldestEvent,

			NewestEvent: stats.NewestEvent,
		}

		// Calculate total cost for this policy.

		var policyCost float64

		for tierName, tierStat := range stats.TierDistribution {

			tier := rm.getStorageTier(tierName)

			if tier != nil {

				cost := float64(tierStat.SizeBytes) / (1024 * 1024 * 1024) * tier.CostPerGB

				tierStat.CostUSD = cost

				policyCost += cost

			}

		}

		policyReport.EstimatedCost = policyCost

		report.Policies[name] = policyReport

		totalEvents += stats.TotalEvents

		totalSize += stats.TotalSize

		totalCost += policyCost

	}

	report.Summary = StorageReportSummary{
		TotalEvents: totalEvents,

		TotalStorageSize: totalSize,

		EstimatedCost: totalCost,

		PoliciesCount: len(rm.policies),
	}

	return report
}

// executeRetentionPolicies executes all retention policies.

func (rm *RetentionManager) executeRetentionPolicies(ctx context.Context) {
	rm.logger.Info("Executing retention policies")

	for name, policy := range rm.policies {
		if err := rm.executePolicyActions(ctx, name, policy); err != nil {

			rm.logger.Error(err, "Failed to execute retention policy", "policy", name)

			retentionOperationsTotal.WithLabelValues("execute_policy", name, "error").Inc()

		} else {
			retentionOperationsTotal.WithLabelValues("execute_policy", name, "success").Inc()
		}
	}
}

// executePolicyActions executes retention actions for a specific policy.

func (rm *RetentionManager) executePolicyActions(ctx context.Context, policyName string, policy *RetentionPolicy) error {
	now := time.Now().UTC()

	cutoffTime := now.Add(-policy.config.RetentionPeriod)

	// Check if there are any legal holds that prevent deletion.

	activeLegalHolds := rm.getActiveLegalHolds(policy)

	rm.logger.V(1).Info("Executing policy actions",

		"policy", policyName,

		"cutoff_time", cutoffTime,

		"legal_holds", len(activeLegalHolds))

	// This is a simplified implementation - in practice, you would:.

	// 1. Query audit events that match the policy criteria and are older than cutoff.

	// 2. Check each event against legal holds.

	// 3. Archive or delete events as appropriate.

	// 4. Update storage statistics.

	// Simulate processing events.

	eventsProcessed := rm.simulateEventProcessing(policyName, policy, cutoffTime, activeLegalHolds)

	policy.lastExecution = now

	retentionEventsProcessed.WithLabelValues(policyName, "processed").Add(float64(eventsProcessed))

	return nil
}

// simulateEventProcessing simulates processing events for retention.

// In a real implementation, this would interact with the audit storage backend.

func (rm *RetentionManager) simulateEventProcessing(policyName string, policy *RetentionPolicy, cutoffTime time.Time, legalHolds []LegalHold) int {
	// This is a placeholder implementation.

	// Real implementation would query actual audit events and process them.

	processedCount := 0

	// Simulate finding events that match the policy.

	// In reality, this would query the backend storage.

	for i := range 100 {

		// Simulate an event that's older than cutoff time.

		eventTime := cutoffTime.Add(-time.Duration(i) * time.Hour)

		// Check if event is protected by legal hold.

		protected := false

		for _, hold := range legalHolds {
			if rm.isEventProtectedByLegalHold(eventTime, hold) {

				protected = true

				break

			}
		}

		if !protected {
			// Archive or delete the event.

			if policy.config.ArchiveBeforeDelete {

				// Archive the event.

				policy.eventsArchived++

				retentionEventsProcessed.WithLabelValues(policyName, "archived").Inc()

			} else {

				// Delete the event.

				policy.eventsDeleted++

				retentionEventsProcessed.WithLabelValues(policyName, "deleted").Inc()

			}
		} else {

			// Event is protected, keep it.

			policy.eventsRetained++

			retentionEventsProcessed.WithLabelValues(policyName, "retained").Inc()

		}

		processedCount++

	}

	return processedCount
}

// getActiveLegalHolds returns active legal holds for a policy.

func (rm *RetentionManager) getActiveLegalHolds(policy *RetentionPolicy) []LegalHold {
	policy.legalHoldMutex.RLock()

	defer policy.legalHoldMutex.RUnlock()

	var activeLegalHolds []LegalHold

	for _, hold := range policy.legalHolds {
		if hold.Status == "active" && (hold.EndDate == nil || hold.EndDate.After(time.Now().UTC())) {
			activeLegalHolds = append(activeLegalHolds, hold)
		}
	}

	return activeLegalHolds
}

// isEventProtectedByLegalHold checks if an event is protected by a legal hold.

func (rm *RetentionManager) isEventProtectedByLegalHold(eventTime time.Time, hold LegalHold) bool {
	// Check if event falls within legal hold date range.

	if hold.DateRange != nil {
		if eventTime.Before(hold.DateRange.StartTime) || eventTime.After(hold.DateRange.EndTime) {
			return false
		}
	}

	// Check if event started after hold start date.

	if eventTime.Before(hold.StartDate) {
		return false
	}

	// Check if hold has ended.

	if hold.EndDate != nil && eventTime.After(*hold.EndDate) {
		return false
	}

	return true
}

// getStorageTier returns a storage tier by name.

func (rm *RetentionManager) getStorageTier(tierName string) *StorageTier {
	for _, tier := range rm.config.StorageTiers {
		if tier.Name == tierName {
			return &tier
		}
	}

	return nil
}

// StorageReport represents a comprehensive storage report.

type StorageReport struct {
	GeneratedAt time.Time `json:"generated_at"`

	Summary StorageReportSummary `json:"summary"`

	Policies map[string]PolicyStorageReport `json:"policies"`
}

// StorageReportSummary provides high-level storage metrics.

type StorageReportSummary struct {
	TotalEvents int64 `json:"total_events"`

	TotalStorageSize int64 `json:"total_storage_size_bytes"`

	EstimatedCost float64 `json:"estimated_cost_usd"`

	PoliciesCount int `json:"policies_count"`
}

// PolicyStorageReport provides storage metrics for a specific policy.

type PolicyStorageReport struct {
	EventCount int64 `json:"event_count"`

	StorageSize int64 `json:"storage_size_bytes"`

	EstimatedCost float64 `json:"estimated_cost_usd"`

	TierDistribution map[string]StorageTierStat `json:"tier_distribution"`

	OldestEvent time.Time `json:"oldest_event"`

	NewestEvent time.Time `json:"newest_event"`
}

// DefaultRetentionConfig returns a default retention configuration.

func DefaultRetentionConfig() *RetentionConfig {
	return &RetentionConfig{
		DefaultRetention: 365 * 24 * time.Hour, // 1 year default

		CheckInterval: 1 * time.Hour, // Check hourly

		BatchSize: 1000,

		ArchivalEnabled: true,

		CompressionEnabled: true,

		EncryptionEnabled: true,

		ComplianceMode: []ComplianceStandard{ComplianceSOC2, ComplianceISO27001},

		Policies: map[string]RetentionPolicyConfig{
			"default": {
				Name: "default",

				Description: "Default retention policy",

				RetentionPeriod: 365 * 24 * time.Hour,

				ArchiveBeforeDelete: true,

				NotifyOnDeletion: true,

				RequireApproval: false,

				LegalHoldExempt: false,
			},

			"security_events": {
				Name: "security_events",

				Description: "Security event retention policy",

				RetentionPeriod: 7 * 365 * 24 * time.Hour, // 7 years

				EventTypes: []EventType{EventTypeSecurityViolation, EventTypeIntrusionAttempt, EventTypeMalwareDetection},

				ArchiveBeforeDelete: true,

				NotifyOnDeletion: true,

				RequireApproval: true,

				LegalHoldExempt: false,
			},

			"authentication": {
				Name: "authentication",

				Description: "Authentication event retention policy",

				RetentionPeriod: 3 * 365 * 24 * time.Hour, // 3 years

				EventTypes: []EventType{EventTypeAuthentication, EventTypeAuthenticationFailed},

				ArchiveBeforeDelete: true,

				NotifyOnDeletion: false,

				RequireApproval: false,

				LegalHoldExempt: true,
			},
		},

		StorageTiers: []StorageTier{
			{
				Name: "hot",

				Description: "High-performance storage for recent events",

				CostPerGB: 0.023,

				AccessLatency: 1 * time.Millisecond,

				Durability: "99.999999999%",

				AvailabilityZones: 3,
			},

			{
				Name: "warm",

				Description: "Medium-performance storage for older events",

				CostPerGB: 0.012,

				AccessLatency: 10 * time.Millisecond,

				Durability: "99.999999999%",

				AvailabilityZones: 2,
			},

			{
				Name: "cold",

				Description: "Low-cost archival storage",

				CostPerGB: 0.004,

				AccessLatency: 1 * time.Second,

				Durability: "99.999999999%",

				AvailabilityZones: 1,
			},
		},
	}
}

