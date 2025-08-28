package rules

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// KPMData represents a kpmdata.
type KPMData struct {
	Timestamp       time.Time `json:"timestamp"`
	NodeID          string    `json:"node_id"`
	PRBUtilization  float64   `json:"prb_utilization"`
	P95Latency      float64   `json:"p95_latency"`
	ActiveUEs       int       `json:"active_ues"`
	CurrentReplicas int       `json:"current_replicas"`
}

// ScalingDecision represents a scalingdecision.
type ScalingDecision struct {
	Action         string    `json:"action"`
	Target         string    `json:"target"`
	Namespace      string    `json:"namespace"`
	TargetReplicas int       `json:"target_replicas"`
	Reason         string    `json:"reason"`
	Timestamp      time.Time `json:"timestamp"`
}

// State represents a state.
type State struct {
	LastDecisionTime time.Time         `json:"last_decision_time"`
	CurrentReplicas  int               `json:"current_replicas"`
	MetricsHistory   []KPMData         `json:"metrics_history"`
	DecisionHistory  []ScalingDecision `json:"decision_history"`
}

// Config represents a config.
type Config struct {
	StateFile            string
	CooldownDuration     time.Duration
	MinReplicas          int
	MaxReplicas          int
	LatencyThresholdHigh float64
	LatencyThresholdLow  float64
	PRBThresholdHigh     float64
	PRBThresholdLow      float64
	EvaluationWindow     time.Duration
	// Performance optimization settings.
	MaxHistorySize int           // Maximum metrics history size (default: 300)
	PruneInterval  time.Duration // How often to prune history (default: 30s)
}

// Validator interface for KMP data validation.
type Validator interface {
	ValidateKMPData(data KPMData) error
	SanitizeForLogging(value string) string
}

// RuleEngine represents a ruleengine.
type RuleEngine struct {
	config         Config
	state          *State
	mu             sync.RWMutex
	lastPruneTime  time.Time
	pruneThreshold time.Duration // Cached 24h threshold for pruning
	validator      Validator     // Security validator for KMP data
}

// NewRuleEngine performs newruleengine operation.
func NewRuleEngine(cfg Config) *RuleEngine {
	// Set default performance parameters if not configured.
	if cfg.MaxHistorySize <= 0 {
		cfg.MaxHistorySize = 300 // ~5 hours at 1 metric/minute
	}
	if cfg.PruneInterval <= 0 {
		cfg.PruneInterval = 30 * time.Second
	}

	engine := &RuleEngine{
		config:         cfg,
		lastPruneTime:  time.Now(),
		pruneThreshold: 24 * time.Hour,
		state: &State{
			CurrentReplicas: 1,
			// Pre-allocate with reasonable capacity to reduce early reallocations.
			MetricsHistory:  make([]KPMData, 0, min(cfg.MaxHistorySize/4, 50)),
			DecisionHistory: make([]ScalingDecision, 0, 20),
		},
	}

	engine.loadState()
	return engine
}

// SetValidator sets the security validator for KMP data validation.
func (e *RuleEngine) SetValidator(validator Validator) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.validator = validator
}

// Evaluate performs evaluate operation.
func (e *RuleEngine) Evaluate(data KPMData) *ScalingDecision {
	e.mu.Lock()
	defer e.mu.Unlock()

	// SECURITY: Validate KMP data if validator is available.
	if e.validator != nil {
		if err := e.validator.ValidateKMPData(data); err != nil {
			log.Printf("KMP data validation failed: %v", err)
			return nil
		}
	}

	e.addMetric(data)
	e.conditionalPrune()

	if !e.canMakeDecision() {
		log.Printf("Cooldown active, skipping evaluation")
		return nil
	}

	if data.CurrentReplicas > 0 {
		e.state.CurrentReplicas = data.CurrentReplicas
	}

	avgMetrics := e.calculateAverageMetrics()

	var decision *ScalingDecision

	if e.shouldScaleOut(avgMetrics) {
		decision = &ScalingDecision{
			Action:         "scale-out",
			Target:         data.NodeID,
			Namespace:      "default",
			TargetReplicas: e.state.CurrentReplicas + 1,
			Reason:         e.getScaleOutReason(avgMetrics),
			Timestamp:      time.Now(),
		}
	} else if e.shouldScaleIn(avgMetrics) {
		decision = &ScalingDecision{
			Action:         "scale-in",
			Target:         data.NodeID,
			Namespace:      "default",
			TargetReplicas: e.state.CurrentReplicas - 1,
			Reason:         e.getScaleInReason(avgMetrics),
			Timestamp:      time.Now(),
		}
	}

	if decision != nil {
		e.applyDecision(decision)
		e.saveState()
	}

	return decision
}

func (e *RuleEngine) shouldScaleOut(metrics AverageMetrics) bool {
	if e.state.CurrentReplicas >= e.config.MaxReplicas {
		return false
	}

	return (metrics.AvgPRB > e.config.PRBThresholdHigh ||
		metrics.AvgLatency > e.config.LatencyThresholdHigh) &&
		metrics.DataPoints >= 3
}

func (e *RuleEngine) shouldScaleIn(metrics AverageMetrics) bool {
	if e.state.CurrentReplicas <= e.config.MinReplicas {
		return false
	}

	return metrics.AvgPRB < e.config.PRBThresholdLow &&
		metrics.AvgLatency < e.config.LatencyThresholdLow &&
		metrics.DataPoints >= 3
}

func (e *RuleEngine) getScaleOutReason(metrics AverageMetrics) string {
	reasons := []string{}
	if metrics.AvgPRB > e.config.PRBThresholdHigh {
		reasons = append(reasons, "High PRB utilization")
	}
	if metrics.AvgLatency > e.config.LatencyThresholdHigh {
		reasons = append(reasons, "High P95 latency")
	}

	if len(reasons) > 0 {
		return reasons[0]
	}
	return "Performance degradation detected"
}

func (e *RuleEngine) getScaleInReason(metrics AverageMetrics) string {
	return "Low resource utilization"
}

func (e *RuleEngine) canMakeDecision() bool {
	if e.state.LastDecisionTime.IsZero() {
		return true
	}

	return time.Since(e.state.LastDecisionTime) >= e.config.CooldownDuration
}

func (e *RuleEngine) applyDecision(decision *ScalingDecision) {
	e.state.LastDecisionTime = decision.Timestamp
	e.state.CurrentReplicas = decision.TargetReplicas
	e.state.DecisionHistory = append(e.state.DecisionHistory, *decision)

	if len(e.state.DecisionHistory) > 100 {
		e.state.DecisionHistory = e.state.DecisionHistory[len(e.state.DecisionHistory)-100:]
	}
}

// AverageMetrics represents a averagemetrics.
type AverageMetrics struct {
	AvgPRB     float64
	AvgLatency float64
	DataPoints int
}

func (e *RuleEngine) calculateAverageMetrics() AverageMetrics {
	if len(e.state.MetricsHistory) == 0 {
		return AverageMetrics{}
	}

	cutoff := time.Now().Add(-e.config.EvaluationWindow)
	var sumPRB, sumLatency float64
	var count int

	for _, m := range e.state.MetricsHistory {
		if m.Timestamp.After(cutoff) {
			sumPRB += m.PRBUtilization
			sumLatency += m.P95Latency
			count++
		}
	}

	if count == 0 {
		return AverageMetrics{}
	}

	return AverageMetrics{
		AvgPRB:     sumPRB / float64(count),
		AvgLatency: sumLatency / float64(count),
		DataPoints: count,
	}
}

// addMetric adds a new metric with capacity management.
func (e *RuleEngine) addMetric(data KPMData) {
	// Check if we need to enforce capacity limit immediately.
	if len(e.state.MetricsHistory) >= e.config.MaxHistorySize {
		// Emergency capacity management: remove oldest 25% to avoid frequent resizing.
		removeCount := e.config.MaxHistorySize / 4
		if removeCount < 1 {
			removeCount = 1
		}
		copy(e.state.MetricsHistory, e.state.MetricsHistory[removeCount:])
		e.state.MetricsHistory = e.state.MetricsHistory[:len(e.state.MetricsHistory)-removeCount]
	}

	e.state.MetricsHistory = append(e.state.MetricsHistory, data)
}

// conditionalPrune performs pruning only when needed based on time interval.
func (e *RuleEngine) conditionalPrune() {
	now := time.Now()
	if now.Sub(e.lastPruneTime) < e.config.PruneInterval {
		return
	}

	e.lastPruneTime = now
	e.pruneHistoryInPlace()
}

// pruneHistoryInPlace performs in-place pruning to avoid slice reallocation.
func (e *RuleEngine) pruneHistoryInPlace() {
	cutoff := time.Now().Add(-e.pruneThreshold)

	// Find first valid entry (binary search could be used for large datasets).
	writeIndex := 0
	for readIndex, metric := range e.state.MetricsHistory {
		if metric.Timestamp.After(cutoff) {
			if writeIndex != readIndex {
				e.state.MetricsHistory[writeIndex] = metric
			}
			writeIndex++
		}
	}

	// Truncate slice to new length without reallocation.
	if writeIndex < len(e.state.MetricsHistory) {
		// Zero out unused elements to help GC.
		for i := writeIndex; i < len(e.state.MetricsHistory); i++ {
			e.state.MetricsHistory[i] = KPMData{}
		}
		e.state.MetricsHistory = e.state.MetricsHistory[:writeIndex]
	}
}

func (e *RuleEngine) loadState() {
	if e.config.StateFile == "" {
		return
	}

	data, err := os.ReadFile(e.config.StateFile)
	if err != nil {
		if !os.IsNotExist(err) {
			log.Printf("Error loading state: %v", err)
		}
		return
	}

	if err := json.Unmarshal(data, e.state); err != nil {
		log.Printf("Error parsing state: %v", err)
	}
}

// saveState securely persists the rule engine state with O-RAN compliant permissions.
// State files contain sensitive operational data including metrics history and scaling.
// decisions that must be protected from unauthorized access per O-RAN WG11 requirements.
func (e *RuleEngine) saveState() {
	if e.config.StateFile == "" {
		return
	}

	data, err := json.MarshalIndent(e.state, "", "  ")
	if err != nil {
		log.Printf("Error marshaling state: %v", err)
		return
	}

	// SECURITY: Use 0600 permissions to ensure only the owner can read/write state files.
	// This prevents unauthorized access to sensitive O-RAN operational data including:.
	// - Metrics history (PRB utilization, latency measurements).
	// - Scaling decision history.
	// - Network function performance data.
	// Complies with O-RAN WG11 security specifications for operational data protection.
	if err := os.WriteFile(e.config.StateFile, data, 0o600); err != nil {
		log.Printf("Error saving state with secure permissions: %v", err)
		return
	}

	log.Printf("State saved securely to %s (permissions: 0600)", e.config.StateFile)
}
