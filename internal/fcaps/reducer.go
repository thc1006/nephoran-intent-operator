package fcaps

import (
	"fmt"
	"sync"
	"time"
)

// Reducer tracks events and detects bursts for scaling decisions.
type Reducer struct {
	mu             sync.Mutex
	events         []timestampedEvent
	burstThreshold int
	windowDuration time.Duration
	outHandoff     string
	lastIntentTime time.Time
	criticalCount  int
}

type timestampedEvent struct {
	event     FCAPSEvent
	timestamp time.Time
}

// ScalingIntent represents the reducer's scaling intent output.
type ScalingIntent struct {
	IntentType    string `json:"intent_type"`
	Target        string `json:"target"`
	Namespace     string `json:"namespace"`
	Replicas      int    `json:"replicas"`
	Reason        string `json:"reason"`
	Source        string `json:"source"`
	CorrelationID string `json:"correlation_id"`
	Timestamp     string `json:"timestamp"`
}

// NewReducer creates a new event reducer.
func NewReducer(burstThreshold int, outHandoff string) *Reducer {
	return &Reducer{
		events:         make([]timestampedEvent, 0),
		burstThreshold: burstThreshold,
		windowDuration: 60 * time.Second,
		outHandoff:     outHandoff,
	}
}

// ProcessEvent processes an event and returns a scaling intent if burst detected.
func (r *Reducer) ProcessEvent(event FCAPSEvent) *ScalingIntent {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	r.events = append(r.events, timestampedEvent{
		event:     event,
		timestamp: now,
	})

	// Clean old events outside window.
	cutoff := now.Add(-r.windowDuration)
	filtered := make([]timestampedEvent, 0)
	criticalCount := 0

	for _, te := range r.events {
		if te.timestamp.After(cutoff) {
			filtered = append(filtered, te)
			if isCriticalEvent(te.event) {
				criticalCount++
			}
		}
	}

	r.events = filtered
	r.criticalCount = criticalCount

	// Check burst condition with 30s cooldown.
	if criticalCount >= r.burstThreshold {
		if time.Since(r.lastIntentTime) > 30*time.Second {
			r.lastIntentTime = now

			// Calculate scaling factor.
			replicas := 2 + (criticalCount / r.burstThreshold)
			if replicas > 10 {
				replicas = 10
			}

			sourceName := event.Event.CommonEventHeader.SourceName
			if sourceName == "" {
				sourceName = "nf-sim"
			}

			return &ScalingIntent{
				IntentType:    "scaling",
				Target:        sourceName,
				Namespace:     "ran-a",
				Replicas:      replicas,
				Reason:        fmt.Sprintf("Burst detected: %d critical events in %v window", criticalCount, r.windowDuration),
				Source:        "fcaps-reducer",
				CorrelationID: fmt.Sprintf("burst-%d", now.Unix()),
				Timestamp:     now.UTC().Format(time.RFC3339),
			}
		}
	}

	return nil
}

func isCriticalEvent(event FCAPSEvent) bool {
	// Check fault events.
	if event.Event.FaultFields != nil {
		severity := event.Event.FaultFields.EventSeverity
		if severity == "CRITICAL" || severity == "MAJOR" {
			return true
		}
	}

	// Check performance thresholds.
	if event.Event.MeasurementsForVfScalingFields != nil {
		fields := event.Event.MeasurementsForVfScalingFields.AdditionalFields
		if fields != nil {
			// PRB utilization > 80%.
			if prb, ok := fields["kpm.prb_utilization"].(float64); ok && prb > 0.8 {
				return true
			}
			// P95 latency > 100ms.
			if latency, ok := fields["kpm.p95_latency_ms"].(float64); ok && latency > 100 {
				return true
			}
			// CPU utilization > 85%.
			if cpu, ok := fields["kpm.cpu_utilization"].(float64); ok && cpu > 0.85 {
				return true
			}
		}
	}

	return false
}

// GetStats returns current event statistics.
func (r *Reducer) GetStats() (total, critical int) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.events), r.criticalCount
}
