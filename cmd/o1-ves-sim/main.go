package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/nephio-project/nephoran-intent-operator/internal/ves"
)

func main() {

	outDir := flag.String("out", "events", "output directory for VES events")

	domain := flag.String("domain", "heartbeat", "event domain (heartbeat, fault, measurement)")

	sourceName := flag.String("source", "o1-ves-sim", "source name for the event")

	interval := flag.Int("interval", 60, "heartbeat interval in seconds")

	flag.Parse()

	// Create output directory if it doesn't exist.

	if err := os.MkdirAll(*outDir, 0o750); err != nil {

		log.Fatalf("Failed to create output directory: %v", err)

	}

	var event *ves.Event

	switch *domain {

	case "heartbeat":

		event = ves.NewHeartbeatEvent(*sourceName, *interval)

	case "fault":

		event = ves.NewFaultEvent(*sourceName, "LinkDown", "MAJOR")

	default:

		// For other domains, create a minimal event.

		now := time.Now().UTC()

		nowMicros := now.UnixNano() / 1000

		event = &ves.Event{

			Event: struct {
				CommonEventHeader ves.CommonEventHeader `json:"commonEventHeader"`

				HeartbeatFields *ves.HeartbeatFields `json:"heartbeatFields,omitempty"`

				FaultFields *ves.FaultFields `json:"faultFields,omitempty"`

				MeasurementFields map[string]interface{} `json:"measurementFields,omitempty"`
			}{

				CommonEventHeader: ves.CommonEventHeader{

					Domain: *domain,

					EventID: fmt.Sprintf("%s-%d", *domain, time.Now().Unix()),

					EventName: fmt.Sprintf("%s_Event", *domain),

					Priority: "Normal",

					ReportingEntityName: *sourceName,

					Sequence: 1,

					SourceName: *sourceName,

					StartEpochMicrosec: nowMicros,

					LastEpochMicrosec: nowMicros,

					Version: "4.0.1",

					VesEventListenerVersion: "7.0.1",
				},
			},
		}

	}

	// Marshal event to JSON with indentation.

	jsonData, err := json.MarshalIndent(event, "", "  ")

	if err != nil {

		log.Fatalf("Failed to marshal event: %v", err)

	}

	// Generate filename with timestamp.

	filename := time.Now().UTC().Format("20060102T150405Z") + ".json"

	filepath := filepath.Join(*outDir, filename)

	// Write to file.

	if err := os.WriteFile(filepath, jsonData, 0o640); err != nil {

		log.Fatalf("Failed to write event file: %v", err)

	}

	log.Printf("VES event written to: %s", filepath)

}
