package rules

import (
	"encoding/json"
	"log"
	"os"
	"sync"
	"time"
)

type KPMData struct {
	Timestamp       time.Time `json:"timestamp"`
	NodeID          string    `json:"node_id"`
	PRBUtilization  float64   `json:"prb_utilization"`
	P95Latency      float64   `json:"p95_latency"`
	ActiveUEs       int       `json:"active_ues"`
	CurrentReplicas int       `json:"current_replicas"`
}

type ScalingDecision struct {
	Action         string    `json:"action"`
	Target         string    `json:"target"`
	Namespace      string    `json:"namespace"`
	TargetReplicas int       `json:"target_replicas"`
	Reason         string    `json:"reason"`
	Timestamp      time.Time `json:"timestamp"`
}

type State struct {
	LastDecisionTime time.Time         `json:"last_decision_time"`
	CurrentReplicas  int               `json:"current_replicas"`
	MetricsHistory   []KPMData         `json:"metrics_history"`
	DecisionHistory  []ScalingDecision `json:"decision_history"`
}

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
}

type RuleEngine struct {
	config Config
	state  *State
	mu     sync.RWMutex
}

func NewRuleEngine(cfg Config) *RuleEngine {
	engine := &RuleEngine{
		config: cfg,
		state: &State{
			CurrentReplicas: 1,
			MetricsHistory:  make([]KPMData, 0),
			DecisionHistory: make([]ScalingDecision, 0),
		},
	}

	engine.loadState()
	return engine
}

func (e *RuleEngine) Evaluate(data KPMData) *ScalingDecision {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.state.MetricsHistory = append(e.state.MetricsHistory, data)
	e.pruneHistory()

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

func (e *RuleEngine) pruneHistory() {
	cutoff := time.Now().Add(-24 * time.Hour)
	newHistory := make([]KPMData, 0)

	for _, m := range e.state.MetricsHistory {
		if m.Timestamp.After(cutoff) {
			newHistory = append(newHistory, m)
		}
	}

	e.state.MetricsHistory = newHistory
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

func (e *RuleEngine) saveState() {
	if e.config.StateFile == "" {
		return
	}

	data, err := json.MarshalIndent(e.state, "", "  ")
	if err != nil {
		log.Printf("Error marshaling state: %v", err)
		return
	}

	if err := os.WriteFile(e.config.StateFile, data, 0644); err != nil {
		log.Printf("Error saving state: %v", err)
	}
}
