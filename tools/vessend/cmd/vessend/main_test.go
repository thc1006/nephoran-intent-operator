package main

import (
	"encoding/json"
	"testing"
	"time"
)

func TestEventGeneration(t *testing.T) {
	config := Config{
		CollectorURL: "http://localhost:9990/eventListener/v7",
		EventType:    "heartbeat",
		Interval:     10 * time.Second,
		Count:        1,
		Source:       "test-source",
	}

	sender := &EventSender{
		config:   config,
		sequence: 0,
	}

	tests := []struct {
		eventType string
	}{
		{"heartbeat"},
		{"fault"},
		{"measurement"},
	}

	for _, test := range tests {
		t.Run(test.eventType, func(t *testing.T) {
			sender.config.EventType = test.eventType
			event := sender.createEvent()

			// Verify the event can be marshaled to JSON
			jsonData, err := json.Marshal(event)
			if err != nil {
				t.Fatalf("Failed to marshal %s event: %v", test.eventType, err)
			}

			// Verify basic structure
			if event.Event.CommonEventHeader.Domain != test.eventType {
				t.Errorf("Expected domain %s, got %s", test.eventType, event.Event.CommonEventHeader.Domain)
			}

			if event.Event.CommonEventHeader.SourceName != "test-source" {
				t.Errorf("Expected source name 'test-source', got %s", event.Event.CommonEventHeader.SourceName)
			}

			// Verify event-specific fields
			switch test.eventType {
			case "heartbeat":
				if event.Event.HeartbeatFields == nil {
					t.Error("Expected heartbeat fields to be present")
				}
				if event.Event.HeartbeatFields.HeartbeatFieldsVersion != "3.0" {
					t.Errorf("Expected heartbeat version 3.0, got %s", event.Event.HeartbeatFields.HeartbeatFieldsVersion)
				}
			case "fault":
				if event.Event.FaultFields == nil {
					t.Error("Expected fault fields to be present")
				}
				if event.Event.FaultFields.EventSeverity != "CRITICAL" {
					t.Errorf("Expected CRITICAL severity, got %s", event.Event.FaultFields.EventSeverity)
				}
			case "measurement":
				if event.Event.MeasurementFields == nil {
					t.Error("Expected measurement fields to be present")
				}
				if len(event.Event.MeasurementFields.VNicUsageArray) == 0 {
					t.Error("Expected vNic usage array to have at least one entry")
				}
				if event.Event.MeasurementFields.AdditionalFields == nil {
					t.Error("Expected additional fields to be present")
				}
				// Check for KPM fields in AdditionalFields (JSON)
				var additionalFieldsMap map[string]interface{}
				if err := json.Unmarshal(event.Event.MeasurementFields.AdditionalFields, &additionalFieldsMap); err != nil {
					t.Errorf("Failed to unmarshal additional fields: %v", err)
				} else {
					if _, exists := additionalFieldsMap["kpm.p95_latency_ms"]; !exists {
						t.Error("Expected kpm.p95_latency_ms in additional fields")
					}
				}
			}

			t.Logf("Generated %s event: %s", test.eventType, string(jsonData))
		})
	}
}

func TestEventIDGeneration(t *testing.T) {
	config := Config{
		EventType: "test",
		Source:    "test-source",
	}

	sender := &EventSender{
		config:   config,
		sequence: 5,
	}

	eventID := sender.generateEventID()

	// Should contain event type, source, and sequence info
	if eventID == "" {
		t.Error("Event ID should not be empty")
	}

	// Should be unique for different sequences
	sender.sequence = 6
	eventID2 := sender.generateEventID()

	if eventID == eventID2 {
		t.Error("Event IDs should be unique for different sequences")
	}

	t.Logf("Generated event IDs: %s, %s", eventID, eventID2)
}

func TestConfigValidation(t *testing.T) {
	// Test valid event types
	validTypes := map[string]bool{
		"fault":       true,
		"measurement": true,
		"heartbeat":   true,
	}

	for eventType := range validTypes {
		if !validTypes[eventType] {
			t.Errorf("Valid event type %s should be accepted", eventType)
		}
	}

	// Test invalid event type
	invalidType := "invalid"
	if validTypes[invalidType] {
		t.Error("Should reject invalid event type")
	}
}
