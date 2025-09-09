package ves

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCommonEventHeader_RequiredFields(t *testing.T) {
	header := CommonEventHeader{
		Domain:                  "heartbeat",
		EventID:                 "test-001",
		EventName:               "Test_Event",
		Priority:                "Normal",
		ReportingEntityName:     "test-entity",
		Sequence:                1,
		SourceName:              "test-source",
		StartEpochMicrosec:      time.Now().UnixNano() / 1000,
		LastEpochMicrosec:       time.Now().UnixNano() / 1000,
		Version:                 "4.0.1",
		VesEventListenerVersion: "7.0.1",
	}

	// Verify all required fields are present
	if header.Domain == "" {
		t.Error("Domain should not be empty")
	}
	if header.EventID == "" {
		t.Error("EventID should not be empty")
	}
	if header.EventName == "" {
		t.Error("EventName should not be empty")
	}
	if header.Priority == "" {
		t.Error("Priority should not be empty")
	}
	if header.SourceName == "" {
		t.Error("SourceName should not be empty")
	}
	if header.Version == "" {
		t.Error("Version should not be empty")
	}
	if header.VesEventListenerVersion == "" {
		t.Error("VesEventListenerVersion should not be empty")
	}
}

func TestNewHeartbeatEvent(t *testing.T) {
	sourceName := "test-o1-sim"
	interval := 30

	event := NewHeartbeatEvent(sourceName, interval)

	// Verify structure
	if event == nil {
		t.Fatal("Event should not be nil")
	}

	// Verify common header
	if event.Event.CommonEventHeader.Domain != "heartbeat" {
		t.Errorf("Expected domain 'heartbeat', got '%s'", event.Event.CommonEventHeader.Domain)
	}
	if event.Event.CommonEventHeader.SourceName != sourceName {
		t.Errorf("Expected source name '%s', got '%s'", sourceName, event.Event.CommonEventHeader.SourceName)
	}
	if event.Event.CommonEventHeader.Priority != "Normal" {
		t.Errorf("Expected priority 'Normal', got '%s'", event.Event.CommonEventHeader.Priority)
	}
	if event.Event.CommonEventHeader.EventType != "O-RAN" {
		t.Errorf("Expected event type 'O-RAN', got '%s'", event.Event.CommonEventHeader.EventType)
	}

	// Verify heartbeat fields
	if event.Event.HeartbeatFields == nil {
		t.Fatal("HeartbeatFields should not be nil for heartbeat event")
	}
	if event.Event.HeartbeatFields.HeartbeatInterval != interval {
		t.Errorf("Expected interval %d, got %d", interval, event.Event.HeartbeatFields.HeartbeatInterval)
	}
	if event.Event.HeartbeatFields.HeartbeatFieldsVersion != "3.0" {
		t.Errorf("Expected heartbeat fields version '3.0', got '%s'", event.Event.HeartbeatFields.HeartbeatFieldsVersion)
	}

	// Verify timestamps are reasonable
	if event.Event.CommonEventHeader.StartEpochMicrosec <= 0 {
		t.Error("StartEpochMicrosec should be positive")
	}
	if event.Event.CommonEventHeader.LastEpochMicrosec <= 0 {
		t.Error("LastEpochMicrosec should be positive")
	}
}

func TestEventJSONMarshaling(t *testing.T) {
	event := NewHeartbeatEvent("json-test", 60)

	jsonData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	// Verify JSON contains expected fields
	jsonStr := string(jsonData)
	expectedFields := []string{
		`"domain":"heartbeat"`,
		`"eventId"`,
		`"eventName":"Heartbeat_O1-VES-Sim"`,
		`"priority":"Normal"`,
		`"sourceName":"json-test"`,
		`"version":"4.0.1"`,
		`"vesEventListenerVersion":"7.0.1"`,
		`"heartbeatFieldsVersion":"3.0"`,
		`"heartbeatInterval":60`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON output missing expected field: %s", field)
		}
	}

	// Verify structure has nested "event" object
	if !strings.Contains(jsonStr, `"event":{`) {
		t.Error("JSON should have nested 'event' object")
	}

	// Unmarshal back and verify
	var decoded Event
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal event: %v", err)
	}

	if decoded.Event.CommonEventHeader.Domain != "heartbeat" {
		t.Error("Unmarshaled event has incorrect domain")
	}
}

func TestGenerateEventID(t *testing.T) {
	id1 := generateEventID()
	if id1 == "" {
		t.Error("EventID should not be empty")
	}

	// Verify format (YYYYMMDD-HHMMSS-suffix)
	if len(id1) < 18 {
		t.Errorf("EventID too short: %s", id1)
	}
	if !strings.Contains(id1, "-") {
		t.Error("EventID should contain separator")
	}

	// Generate multiple IDs to verify randomness
	id2 := generateEventID()
	id3 := generateEventID()

	// At minimum, the random suffixes should be different
	suffix1 := id1[len(id1)-8:]
	suffix2 := id2[len(id2)-8:]
	suffix3 := id3[len(id3)-8:]

	if suffix1 == suffix2 && suffix2 == suffix3 {
		t.Error("Random suffixes should be different")
	}
}

func TestHeartbeatFields_AdditionalFields(t *testing.T) {
	event := NewHeartbeatEvent("test", 30)
	event.Event.HeartbeatFields.AdditionalFields = json.RawMessage(`{"customField1":"value1","customField2":42}`)

	jsonData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event with additional fields: %v", err)
	}

	jsonStr := string(jsonData)
	if !strings.Contains(jsonStr, `"customField1":"value1"`) {
		t.Error("Additional field customField1 not found in JSON")
	}
	if !strings.Contains(jsonStr, `"customField2":42`) {
		t.Error("Additional field customField2 not found in JSON")
	}
}

func TestEvent_OmitEmptyFields(t *testing.T) {
	event := NewHeartbeatEvent("omit-test", 60)

	// Ensure fault and measurement fields are not set
	event.Event.FaultFields = nil
	event.Event.MeasurementFields = nil

	jsonData, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("Failed to marshal event: %v", err)
	}

	jsonStr := string(jsonData)

	// These fields should be omitted when nil
	if strings.Contains(jsonStr, "faultFields") {
		t.Error("faultFields should be omitted when nil")
	}
	if strings.Contains(jsonStr, "measurementFields") {
		t.Error("measurementFields should be omitted when nil")
	}
}

func TestPriorityValues(t *testing.T) {
	validPriorities := []string{"High", "Medium", "Normal", "Low"}

	for _, priority := range validPriorities {
		event := NewHeartbeatEvent("priority-test", 60)
		event.Event.CommonEventHeader.Priority = priority

		if event.Event.CommonEventHeader.Priority != priority {
			t.Errorf("Failed to set priority to %s", priority)
		}
	}
}

func TestNewFaultEvent(t *testing.T) {
	sourceName := "test-du"
	alarmCondition := "LinkDown"
	severity := "MAJOR"

	event := NewFaultEvent(sourceName, alarmCondition, severity)

	// Verify structure
	if event == nil {
		t.Fatal("Fault event should not be nil")
	}

	// Verify common header
	if event.Event.CommonEventHeader.Domain != "fault" {
		t.Errorf("Expected domain 'fault', got '%s'", event.Event.CommonEventHeader.Domain)
	}
	if event.Event.CommonEventHeader.SourceName != sourceName {
		t.Errorf("Expected source name '%s', got '%s'", sourceName, event.Event.CommonEventHeader.SourceName)
	}
	if event.Event.CommonEventHeader.Priority != "High" {
		t.Errorf("Expected priority 'High' for fault, got '%s'", event.Event.CommonEventHeader.Priority)
	}
	if !strings.Contains(event.Event.CommonEventHeader.EventName, alarmCondition) {
		t.Errorf("Event name should contain alarm condition '%s'", alarmCondition)
	}

	// Verify fault fields
	if event.Event.FaultFields == nil {
		t.Fatal("FaultFields should not be nil for fault event")
	}
	if event.Event.FaultFields.AlarmCondition != alarmCondition {
		t.Errorf("Expected alarm condition '%s', got '%s'", alarmCondition, event.Event.FaultFields.AlarmCondition)
	}
	if event.Event.FaultFields.EventSeverity != severity {
		t.Errorf("Expected severity '%s', got '%s'", severity, event.Event.FaultFields.EventSeverity)
	}
	if event.Event.FaultFields.EventSourceType != "O-RAN-DU" {
		t.Errorf("Expected source type 'O-RAN-DU', got '%s'", event.Event.FaultFields.EventSourceType)
	}
	if event.Event.FaultFields.FaultFieldsVersion != "4.0" {
		t.Errorf("Expected fault fields version '4.0', got '%s'", event.Event.FaultFields.FaultFieldsVersion)
	}

	// Verify heartbeat fields are not present
	if event.Event.HeartbeatFields != nil {
		t.Error("HeartbeatFields should be nil for fault event")
	}
}

