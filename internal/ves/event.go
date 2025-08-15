package ves

import (
	"crypto/rand"
	"encoding/hex"
	"time"
)

// CommonEventHeader represents the common header for all VES events (VES 7.x spec)
type CommonEventHeader struct {
	Domain               string `json:"domain"`                     // heartbeat, fault, measurement, etc.
	EventID              string `json:"eventId"`                    // unique event identifier
	EventName            string `json:"eventName"`                  // unique event name
	EventType            string `json:"eventType,omitempty"`        // e.g., "applicationVnf"
	LastEpochMicrosec    int64  `json:"lastEpochMicrosec"`          // last epoch in microseconds
	Priority             string `json:"priority"`                   // High, Medium, Normal, Low
	ReportingEntityID    string `json:"reportingEntityId,omitempty"`
	ReportingEntityName  string `json:"reportingEntityName"`
	Sequence             int    `json:"sequence"`                   // ordering sequence
	SourceID             string `json:"sourceId,omitempty"`
	SourceName           string `json:"sourceName"`                 // name of entity experiencing event
	StartEpochMicrosec   int64  `json:"startEpochMicrosec"`         // start epoch in microseconds
	Version              string `json:"version"`                    // version of event header spec
	VesEventListenerVersion string `json:"vesEventListenerVersion"` // VES collector API version
}

// HeartbeatFields represents heartbeat domain specific fields
type HeartbeatFields struct {
	HeartbeatFieldsVersion string                 `json:"heartbeatFieldsVersion"`
	HeartbeatInterval      int                    `json:"heartbeatInterval"`
	AdditionalFields       map[string]interface{} `json:"additionalFields,omitempty"`
}

// FaultFields represents fault domain specific fields
type FaultFields struct {
	FaultFieldsVersion  string `json:"faultFieldsVersion"`
	AlarmCondition      string `json:"alarmCondition"`
	EventSeverity       string `json:"eventSeverity"`
	EventSourceType     string `json:"eventSourceType"`
	SpecificProblem     string `json:"specificProblem"`
	VfStatus            string `json:"vfStatus"`
	AlarmInterfaceA     string `json:"alarmInterfaceA,omitempty"`
}

// Event represents a VES Common Event Format event
type Event struct {
	Event struct {
		CommonEventHeader CommonEventHeader      `json:"commonEventHeader"`
		HeartbeatFields   *HeartbeatFields       `json:"heartbeatFields,omitempty"`
		FaultFields       *FaultFields           `json:"faultFields,omitempty"`
		MeasurementFields map[string]interface{} `json:"measurementFields,omitempty"`
	} `json:"event"`
}

// NewHeartbeatEvent creates a minimal heartbeat event
func NewHeartbeatEvent(sourceName string, interval int) *Event {
	now := time.Now().UTC()
	nowMicros := now.UnixNano() / 1000
	
	return &Event{
		Event: struct {
			CommonEventHeader CommonEventHeader      `json:"commonEventHeader"`
			HeartbeatFields   *HeartbeatFields       `json:"heartbeatFields,omitempty"`
			FaultFields       *FaultFields           `json:"faultFields,omitempty"`
			MeasurementFields map[string]interface{} `json:"measurementFields,omitempty"`
		}{
			CommonEventHeader: CommonEventHeader{
				Domain:                  "heartbeat",
				EventID:                 generateEventID(),
				EventName:               "Heartbeat_O1-VES-Sim",
				EventType:               "O-RAN",
				LastEpochMicrosec:       nowMicros,
				Priority:                "Normal",
				ReportingEntityName:     sourceName,
				Sequence:                0,
				SourceName:              sourceName,
				StartEpochMicrosec:      nowMicros,
				Version:                 "4.0.1",
				VesEventListenerVersion: "7.0.1",
			},
			HeartbeatFields: &HeartbeatFields{
				HeartbeatFieldsVersion: "3.0",
				HeartbeatInterval:      interval,
			},
		},
	}
}

// generateEventID creates a unique event ID
func generateEventID() string {
	return time.Now().UTC().Format("20060102-150405") + "-" + generateRandomSuffix()
}

func generateRandomSuffix() string {
	randomBytes := make([]byte, 4)
	_, _ = rand.Read(randomBytes)
	return hex.EncodeToString(randomBytes)
}

// NewFaultEvent creates a minimal fault event
func NewFaultEvent(sourceName, alarmCondition, severity string) *Event {
	now := time.Now().UTC()
	nowMicros := now.UnixNano() / 1000
	
	return &Event{
		Event: struct {
			CommonEventHeader CommonEventHeader      `json:"commonEventHeader"`
			HeartbeatFields   *HeartbeatFields       `json:"heartbeatFields,omitempty"`
			FaultFields       *FaultFields           `json:"faultFields,omitempty"`
			MeasurementFields map[string]interface{} `json:"measurementFields,omitempty"`
		}{
			CommonEventHeader: CommonEventHeader{
				Domain:                  "fault",
				EventID:                 generateEventID(),
				EventName:               "Fault_" + alarmCondition,
				EventType:               "O-RAN",
				LastEpochMicrosec:       nowMicros,
				Priority:                "High",
				ReportingEntityName:     sourceName,
				Sequence:                0,
				SourceName:              sourceName,
				StartEpochMicrosec:      nowMicros,
				Version:                 "4.0.1",
				VesEventListenerVersion: "7.0.1",
			},
			FaultFields: &FaultFields{
				FaultFieldsVersion:  "4.0",
				AlarmCondition:      alarmCondition,
				EventSeverity:       severity,
				EventSourceType:     "O-RAN-DU",
				SpecificProblem:     alarmCondition + " detected",
				VfStatus:            "Active",
			},
		},
	}
}