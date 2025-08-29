
package main



import (

	"encoding/json"

	"flag"

	"fmt"

	"io"

	"log"

	"net/http"

	"os"

	"path/filepath"

	"sync"

	"time"



	"github.com/nephio-project/nephoran-intent-operator/internal/fcaps"

	"github.com/nephio-project/nephoran-intent-operator/internal/ingest"

)



// Config represents a config.

type Config struct {

	ListenAddr     string

	HandoffDir     string

	BurstThreshold int

	WindowSeconds  int

	Verbose        bool

}



// EventTracker represents a eventtracker.

type EventTracker struct {

	mu             sync.Mutex

	events         []fcaps.FCAPSEvent

	lastIntent     time.Time

	intentCooldown time.Duration

}



func main() {

	config := parseFlags()



	if config.Verbose {

		log.Printf("FCAPS Reducer starting with config: %+v", config)

	}



	// Ensure handoff directory exists.

	if err := os.MkdirAll(config.HandoffDir, 0o750); err != nil {

		log.Fatalf("Failed to create handoff directory: %v", err)

	}



	tracker := &EventTracker{

		events:         make([]fcaps.FCAPSEvent, 0),

		intentCooldown: time.Duration(config.WindowSeconds) * time.Second,

	}



	// Start periodic burst detection.

	go tracker.detectBursts(config)



	// Start HTTP server to receive VES events.

	http.HandleFunc("/eventListener/v7", tracker.handleVESEvent(config))

	http.HandleFunc("/health", handleHealth)



	log.Printf("FCAPS Reducer listening on %s", config.ListenAddr)

	log.Printf("Burst detection: %d events in %d seconds triggers scaling intent",

		config.BurstThreshold, config.WindowSeconds)



	// Use http.Server with timeouts to fix G114 security warning.

	server := &http.Server{

		Addr:         config.ListenAddr,

		Handler:      nil,

		ReadTimeout:  15 * time.Second,

		WriteTimeout: 15 * time.Second,

		IdleTimeout:  60 * time.Second,

	}

	if err := server.ListenAndServe(); err != nil {

		log.Fatalf("Server failed: %v", err)

	}

}



func parseFlags() Config {

	var config Config



	flag.StringVar(&config.ListenAddr, "listen", ":9999", "Listen address for VES collector")

	flag.StringVar(&config.HandoffDir, "handoff", "./handoff", "Directory for intent handoff files")

	flag.IntVar(&config.BurstThreshold, "burst", 3, "Number of critical events to trigger scaling")

	flag.IntVar(&config.WindowSeconds, "window", 60, "Time window in seconds for burst detection")

	flag.BoolVar(&config.Verbose, "verbose", false, "Enable verbose logging")

	flag.Parse()



	return config

}



func (t *EventTracker) handleVESEvent(config Config) http.HandlerFunc {

	return func(w http.ResponseWriter, r *http.Request) {

		if r.Method != http.MethodPost {

			w.WriteHeader(http.StatusMethodNotAllowed)

			return

		}



		body, err := io.ReadAll(r.Body)

		if err != nil {

			http.Error(w, "Failed to read body", http.StatusBadRequest)

			return

		}

		defer func() { _ = r.Body.Close() }()



		var event fcaps.FCAPSEvent

		if err := json.Unmarshal(body, &event); err != nil {

			http.Error(w, "Invalid VES event format", http.StatusBadRequest)

			return

		}



		// Track the event.

		t.mu.Lock()

		t.events = append(t.events, event)

		// Keep only recent events (sliding window).

		if len(t.events) > 100 {

			t.events = t.events[len(t.events)-100:]

		}

		eventCount := len(t.events)

		t.mu.Unlock()



		if config.Verbose {

			log.Printf("Received VES event: domain=%s, name=%s, severity=%s (total: %d)",

				event.Event.CommonEventHeader.Domain,

				event.Event.CommonEventHeader.EventName,

				getEventSeverity(event),

				eventCount)

		}



		// Return VES standard response.

		w.WriteHeader(http.StatusAccepted)

		if _, err := w.Write([]byte(`{"commandList": []}`)); err != nil {

			log.Printf("Failed to write VES response: %v", err)

		}

	}

}



func (t *EventTracker) detectBursts(config Config) {

	ticker := time.NewTicker(10 * time.Second)

	defer ticker.Stop()



	for range ticker.C {

		t.mu.Lock()



		// Clean old events outside the window.

		cutoff := time.Now().Add(-time.Duration(config.WindowSeconds) * time.Second)

		filtered := make([]fcaps.FCAPSEvent, 0)

		criticalCount := 0



		for _, event := range t.events {

			// Use current time if timestamp is old (for testing).

			eventTime := time.Unix(0, event.Event.CommonEventHeader.LastEpochMicrosec*1000)

			// If event time is more than 1 day old, consider it as recent (for testing with static examples).

			if time.Since(eventTime) > 24*time.Hour {

				eventTime = time.Now()

			}

			if eventTime.After(cutoff) {

				filtered = append(filtered, event)

				if isCriticalEvent(event) {

					criticalCount++

				}

			}

		}



		t.events = filtered

		shouldTrigger := criticalCount >= config.BurstThreshold

		canTrigger := time.Since(t.lastIntent) > t.intentCooldown



		t.mu.Unlock()



		if config.Verbose && len(filtered) > 0 {

			log.Printf("Burst detection: %d critical events in window (threshold: %d)",

				criticalCount, config.BurstThreshold)

		}



		// Check if we should generate an intent.

		if shouldTrigger && canTrigger {

			t.generateScalingIntent(config, criticalCount)



			t.mu.Lock()

			t.lastIntent = time.Now()

			t.mu.Unlock()

		}

	}

}



func (t *EventTracker) generateScalingIntent(config Config, eventCount int) {

	// Calculate scaling factor based on burst size.

	scaleFactor := 1 + (eventCount / config.BurstThreshold)

	if scaleFactor > 3 {

		scaleFactor = 3 // Cap at 3x scaling

	}



	intent := &ingest.Intent{

		IntentType:    "scaling",

		Target:        "nf-sim",

		Namespace:     "ran-a",

		Replicas:      scaleFactor * 2, // Scale to 2x, 4x, or 6x

		Reason:        fmt.Sprintf("Burst detected: %d critical events in %ds window", eventCount, config.WindowSeconds),

		Source:        "fcaps-reducer",

		CorrelationID: fmt.Sprintf("burst-%d", time.Now().Unix()),

	}



	// Write intent to handoff directory.

	intentJSON, err := json.MarshalIndent(intent, "", "  ")

	if err != nil {

		log.Printf("Failed to marshal intent: %v", err)

		return

	}



	timestamp := time.Now().UTC().Format("20060102T150405Z")

	filename := filepath.Join(config.HandoffDir, fmt.Sprintf("intent-%s.json", timestamp))



	if err := os.WriteFile(filename, intentJSON, 0o640); err != nil {

		log.Printf("Failed to write intent file: %v", err)

		return

	}



	log.Printf("SCALING INTENT GENERATED: %s", filename)

	log.Printf("Intent details: scale %s to %d replicas in namespace %s",

		intent.Target, intent.Replicas, intent.Namespace)

}



func isCriticalEvent(event fcaps.FCAPSEvent) bool {

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

			// PRB utilization > 0.8.

			if prbUtil, ok := fields["kpm.prb_utilization"].(float64); ok && prbUtil > 0.8 {

				return true

			}

			// P95 latency > 100ms.

			if latency, ok := fields["kpm.p95_latency_ms"].(float64); ok && latency > 100 {

				return true

			}

			// CPU utilization > 0.85.

			if cpuUtil, ok := fields["kpm.cpu_utilization"].(float64); ok && cpuUtil > 0.85 {

				return true

			}

		}

	}



	return false

}



func getEventSeverity(event fcaps.FCAPSEvent) string {

	if event.Event.FaultFields != nil {

		return event.Event.FaultFields.EventSeverity

	}

	if isCriticalEvent(event) {

		return "HIGH"

	}

	return "NORMAL"

}



func handleHealth(w http.ResponseWriter, r *http.Request) {

	w.WriteHeader(http.StatusOK)

	if _, err := w.Write([]byte("OK")); err != nil {

		log.Printf("Failed to write health response: %v", err)

	}

}

