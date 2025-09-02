// FIXME: Adding package comment per revive linter.

// Package fcaps provides FCAPS (Fault, Configuration, Accounting, Performance, Security) event processing.

package fcaps

import (
	"fmt"
	"log"
	"time"

	ingest "github.com/thc1006/nephoran-intent-operator/internal/ingest"
)

// CommonEventHeader represents the common header for all VES events.

type CommonEventHeader struct {
	Version string `json:"version"`

	Domain string `json:"domain"`

	EventName string `json:"eventName"`

	EventID string `json:"eventId"`

	Sequence int `json:"sequence"`

	Priority string `json:"priority"`

	ReportingEntityName string `json:"reportingEntityName"`

	SourceName string `json:"sourceName"`

	NFVendorName string `json:"nfVendorName"`

	StartEpochMicrosec int64 `json:"startEpochMicrosec"`

	LastEpochMicrosec int64 `json:"lastEpochMicrosec"`

	TimeZoneOffset string `json:"timeZoneOffset"`
}

// FaultFields represents fault event specific fields.

type FaultFields struct {
	FaultFieldsVersion string `json:"faultFieldsVersion"`

	AlarmCondition string `json:"alarmCondition"`

	EventSeverity string `json:"eventSeverity"`

	SpecificProblem string `json:"specificProblem"`

	EventSourceType string `json:"eventSourceType"`

	VFStatus string `json:"vfStatus"`

	AlarmInterfaceA string `json:"alarmInterfaceA"`
}

// MeasurementsForVfScalingFields represents performance measurement fields.

type MeasurementsForVfScalingFields struct {
	MeasurementsForVfScalingVersion string `json:"measurementsForVfScalingVersion"`

	VNicUsageArray []VNicUsage `json:"vNicUsageArray,omitempty"`

	AdditionalFields map[string]interface{} `json:"additionalFields,omitempty"`
}

// VNicUsage represents network interface usage metrics.

type VNicUsage struct {
	VnfNetworkInterface string `json:"vnfNetworkInterface"`

	ReceivedOctetsDelta int64 `json:"receivedOctetsDelta"`

	TransmittedOctetsDelta int64 `json:"transmittedOctetsDelta"`
}

// HeartbeatFields represents heartbeat event fields.

type HeartbeatFields struct {
	HeartbeatFieldsVersion string `json:"heartbeatFieldsVersion"`

	HeartbeatInterval int `json:"heartbeatInterval"`
}

// FCAPSEvent represents a VES event with various field types.

type FCAPSEvent struct {
	Event struct {
		CommonEventHeader CommonEventHeader `json:"commonEventHeader"`

		FaultFields *FaultFields `json:"faultFields,omitempty"`

		MeasurementsForVfScalingFields *MeasurementsForVfScalingFields `json:"measurementsForVfScalingFields,omitempty"`

		HeartbeatFields *HeartbeatFields `json:"heartbeatFields,omitempty"`
	} `json:"event"`
}

// Processor handles FCAPS event processing and threshold detection.

type Processor struct {
	currentReplicas int

	maxReplicas int

	minReplicas int

	target string

	namespace string
}

// ScalingDecision represents a scaling decision made by the processor.

type ScalingDecision struct {
	ShouldScale bool

	NewReplicas int

	Reason string

	Severity string
}

// NewProcessor creates a new FCAPS processor.

func NewProcessor(target, namespace string) *Processor {
	return &Processor{
		currentReplicas: 1, // Start with 1 replica

		maxReplicas: 10,

		minReplicas: 1,

		target: target,

		namespace: namespace,
	}
}

// ProcessEvent processes a FCAPS event and returns scaling decision.

func (p *Processor) ProcessEvent(event FCAPSEvent) ScalingDecision {
	header := event.Event.CommonEventHeader

	log.Printf("Processing event: %s (domain: %s, source: %s)",

		header.EventName, header.Domain, header.SourceName)

	switch header.Domain {

	case "fault":

		return p.processFaultEvent(event)

	case "measurementsForVfScaling":

		return p.processPerformanceEvent(event)

	case "heartbeat":

		return p.processHeartbeatEvent(event)

	default:

		log.Printf("Unknown event domain: %s", header.Domain)

		return ScalingDecision{ShouldScale: false}

	}
}

// processFaultEvent handles fault events and determines scaling needs.

func (p *Processor) processFaultEvent(event FCAPSEvent) ScalingDecision {
	if event.Event.FaultFields == nil {
		return ScalingDecision{ShouldScale: false}
	}

	faultFields := event.Event.FaultFields

	log.Printf("Fault event: severity=%s, condition=%s, problem=%s",

		faultFields.EventSeverity, faultFields.AlarmCondition, faultFields.SpecificProblem)

	// For CRITICAL fault events, scale up by 2.

	if faultFields.EventSeverity == "CRITICAL" {

		newReplicas := p.currentReplicas + 2

		if newReplicas > p.maxReplicas {
			newReplicas = p.maxReplicas
		}

		if newReplicas > p.currentReplicas {

			reason := fmt.Sprintf("Critical fault detected: %s (%s)",

				faultFields.SpecificProblem, faultFields.AlarmCondition)

			return ScalingDecision{
				ShouldScale: true,

				NewReplicas: newReplicas,

				Reason: reason,

				Severity: "critical",
			}

		}

	}

	return ScalingDecision{ShouldScale: false}
}

// processPerformanceEvent handles performance measurement events.

func (p *Processor) processPerformanceEvent(event FCAPSEvent) ScalingDecision {
	if event.Event.MeasurementsForVfScalingFields == nil {
		return ScalingDecision{ShouldScale: false}
	}

	fields := event.Event.MeasurementsForVfScalingFields

	additionalFields := fields.AdditionalFields

	if additionalFields == nil {
		return ScalingDecision{ShouldScale: false}
	}

	// Check PRB utilization threshold (> 0.8).

	if prbUtil, exists := additionalFields["kpm.prb_utilization"]; exists {
		if utilization, ok := prbUtil.(float64); ok {

			log.Printf("PRB utilization: %.2f", utilization)

			if utilization > 0.8 {
				return p.createScaleUpDecision("PRB utilization exceeded threshold (%.2f > 0.8)", utilization)
			}

		}
	}

	// Check p95 latency threshold (> 100ms).

	if p95Latency, exists := additionalFields["kpm.p95_latency_ms"]; exists {
		if latency, ok := p95Latency.(float64); ok {

			log.Printf("P95 latency: %.2f ms", latency)

			if latency > 100.0 {
				return p.createScaleUpDecision("P95 latency exceeded threshold (%.2f ms > 100 ms)", latency)
			}

		}
	}

	return ScalingDecision{ShouldScale: false}
}

// processHeartbeatEvent handles heartbeat events (no scaling action needed).

func (p *Processor) processHeartbeatEvent(event FCAPSEvent) ScalingDecision {
	log.Printf("Heartbeat received from %s", event.Event.CommonEventHeader.SourceName)

	return ScalingDecision{ShouldScale: false}
}

// createScaleUpDecision creates a scale-up decision by 1 replica.

func (p *Processor) createScaleUpDecision(reasonFormat string, value float64) ScalingDecision {
	newReplicas := p.currentReplicas + 1

	if newReplicas > p.maxReplicas {
		newReplicas = p.maxReplicas
	}

	if newReplicas > p.currentReplicas {

		reason := fmt.Sprintf(reasonFormat, value)

		return ScalingDecision{
			ShouldScale: true,

			NewReplicas: newReplicas,

			Reason: reason,

			Severity: "performance",
		}

	}

	return ScalingDecision{ShouldScale: false}
}

// GenerateIntent creates a scaling intent from a scaling decision.

func (p *Processor) GenerateIntent(decision ScalingDecision) (*ingest.Intent, error) {
	if !decision.ShouldScale {
		return nil, fmt.Errorf("no scaling needed")
	}

	intent := &ingest.Intent{
		IntentType: "scaling",

		Target: p.target,

		Namespace: p.namespace,

		Replicas: decision.NewReplicas,

		Reason: decision.Reason,

		Source: "planner",

		CorrelationID: fmt.Sprintf("fcaps-%d", time.Now().Unix()),
	}

	// Update current replicas to new value.

	p.currentReplicas = decision.NewReplicas

	log.Printf("Updated current replicas to %d", p.currentReplicas)

	return intent, nil
}

// GetCurrentReplicas returns the current replica count.

func (p *Processor) GetCurrentReplicas() int {
	return p.currentReplicas
}

// SetCurrentReplicas sets the current replica count (for initialization).

func (p *Processor) SetCurrentReplicas(replicas int) {
	if replicas >= p.minReplicas && replicas <= p.maxReplicas {

		p.currentReplicas = replicas

		log.Printf("Set current replicas to %d", replicas)

	}
}
