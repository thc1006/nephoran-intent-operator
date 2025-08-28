package main

import (
	"bytes"
	"context"
	cryptorand "crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"math/big"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
)

// VES Event Structure based on docs/contracts/fcaps.ves.examples.json.
type VESEvent struct {
	Event EventPayload `json:"event"`
}

// EventPayload represents a eventpayload.
type EventPayload struct {
	CommonEventHeader CommonEventHeader  `json:"commonEventHeader"`
	FaultFields       *FaultFields       `json:"faultFields,omitempty"`
	MeasurementFields *MeasurementFields `json:"measurementsForVfScalingFields,omitempty"`
	HeartbeatFields   *HeartbeatFields   `json:"heartbeatFields,omitempty"`
}

// CommonEventHeader represents a commoneventheader.
type CommonEventHeader struct {
	Version             string `json:"version"`
	Domain              string `json:"domain"`
	EventName           string `json:"eventName"`
	EventID             string `json:"eventId"`
	Sequence            int    `json:"sequence"`
	Priority            string `json:"priority"`
	ReportingEntityName string `json:"reportingEntityName"`
	SourceName          string `json:"sourceName"`
	NFVendorName        string `json:"nfVendorName"`
	StartEpochMicrosec  int64  `json:"startEpochMicrosec"`
	LastEpochMicrosec   int64  `json:"lastEpochMicrosec"`
	TimeZoneOffset      string `json:"timeZoneOffset"`
}

// FaultFields represents a faultfields.
type FaultFields struct {
	FaultFieldsVersion string `json:"faultFieldsVersion"`
	AlarmCondition     string `json:"alarmCondition"`
	EventSeverity      string `json:"eventSeverity"`
	SpecificProblem    string `json:"specificProblem"`
	EventSourceType    string `json:"eventSourceType"`
	VFStatus           string `json:"vfStatus"`
	AlarmInterfaceA    string `json:"alarmInterfaceA"`
}

// MeasurementFields represents a measurementfields.
type MeasurementFields struct {
	MeasurementsVersion string                 `json:"measurementsForVfScalingVersion"`
	VNicUsageArray      []VNicUsage            `json:"vNicUsageArray"`
	AdditionalFields    map[string]interface{} `json:"additionalFields"`
}

// VNicUsage represents a vnicusage.
type VNicUsage struct {
	VNFNetworkInterface    string `json:"vnfNetworkInterface"`
	ReceivedOctetsDelta    int64  `json:"receivedOctetsDelta"`
	TransmittedOctetsDelta int64  `json:"transmittedOctetsDelta"`
}

// HeartbeatFields represents a heartbeatfields.
type HeartbeatFields struct {
	HeartbeatFieldsVersion string `json:"heartbeatFieldsVersion"`
	HeartbeatInterval      int    `json:"heartbeatInterval"`
}

// Config holds command-line configuration.
type Config struct {
	CollectorURL string
	EventType    string
	Interval     time.Duration
	Count        int
	Source       string
	Username     string
	Password     string
	InsecureTLS  bool
}

// EventSender handles VES event transmission.
type EventSender struct {
	config     Config
	httpClient *http.Client
	sequence   int
}

func main() {
	config := parseFlags()

	// Security warnings.
	if config.InsecureTLS {
		log.Println("⚠️  WARNING: TLS certificate verification is disabled. This is insecure and should only be used in development.")
		log.Println("⚠️  WARNING: Your connection is vulnerable to man-in-the-middle attacks.")
	}

	if config.Username != "" && config.Password != "" {
		validateCredentialStrength(config.Username, config.Password)
	}

	// Setup HTTP client with retry and TLS configuration.
	httpClient := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				InsecureSkipVerify: config.InsecureTLS,
				MinVersion:         tls.VersionTLS12, // Enforce minimum TLS 1.2
			},
		},
	}

	sender := &EventSender{
		config:     config,
		httpClient: httpClient,
		sequence:   0,
	}

	// Setup graceful shutdown.
	ctx, cancel := context.WithCancel(context.Background())
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal, stopping...")
		cancel()
	}()

	// Start sending events.
	if err := sender.Run(ctx); err != nil {
		log.Fatalf("Error running event sender: %v", err)
	}

	log.Println("Event sender stopped")
}

func parseFlags() Config {
	var config Config

	flag.StringVar(&config.CollectorURL, "collector", "http://localhost:9990/eventListener/v7", "VES collector URL")
	flag.StringVar(&config.EventType, "event-type", "heartbeat", "Event type to send (fault|measurement|heartbeat)")
	flag.DurationVar(&config.Interval, "interval", 10*time.Second, "Time between events")
	flag.IntVar(&config.Count, "count", 100, "Number of events to send (0 for unlimited)")
	flag.StringVar(&config.Source, "source", "vessend-tool", "Event source name")
	flag.StringVar(&config.Username, "username", "", "Basic auth username")
	flag.StringVar(&config.Password, "password", "", "Basic auth password")
	flag.BoolVar(&config.InsecureTLS, "insecure-tls", false, "Skip TLS certificate verification (INSECURE - development only)")

	flag.Parse()

	// Comprehensive input validation.
	if err := validateConfig(&config); err != nil {
		log.Fatalf("Configuration validation failed: %v", err)
	}

	return config
}

// validateConfig performs comprehensive input validation and sanitization.
func validateConfig(config *Config) error {
	// Validate and sanitize collector URL.
	parsedURL, err := url.Parse(config.CollectorURL)
	if err != nil {
		return fmt.Errorf("invalid collector URL: %w", err)
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("collector URL must use http or https scheme")
	}
	config.CollectorURL = parsedURL.String() // Use normalized URL

	// Validate event type.
	validTypes := map[string]bool{
		"fault":       true,
		"measurement": true,
		"heartbeat":   true,
	}

	if !validTypes[config.EventType] {
		return fmt.Errorf("invalid event type '%s'. Must be one of: fault, measurement, heartbeat", config.EventType)
	}

	// Validate interval bounds.
	if config.Interval < 100*time.Millisecond {
		return fmt.Errorf("interval must be at least 100ms")
	}
	if config.Interval > 24*time.Hour {
		return fmt.Errorf("interval cannot exceed 24 hours")
	}

	// Validate count bounds.
	if config.Count < 0 {
		return fmt.Errorf("count cannot be negative")
	}
	if config.Count > 1000000 {
		return fmt.Errorf("count cannot exceed 1000000 for safety")
	}

	// Sanitize source name (prevent injection attacks).
	config.Source = sanitizeString(config.Source, 50)

	// Sanitize username (but not password - it might have special chars).
	if config.Username != "" {
		config.Username = sanitizeString(config.Username, 100)
	}

	return nil
}

// sanitizeString removes potentially dangerous characters and limits length.
func sanitizeString(input string, maxLen int) string {
	// Remove control characters and non-printable characters.
	reg := regexp.MustCompile(`[\x00-\x1F\x7F]+`)
	input = reg.ReplaceAllString(input, "")

	// Remove potential injection characters.
	input = strings.ReplaceAll(input, "\n", "")
	input = strings.ReplaceAll(input, "\r", "")
	input = strings.ReplaceAll(input, "\t", "")

	// Limit length.
	if len(input) > maxLen {
		input = input[:maxLen]
	}

	return strings.TrimSpace(input)
}

// validateCredentialStrength checks for weak credentials and warns the user.
func validateCredentialStrength(username, password string) {
	weakPasswords := []string{"password", "admin", "123456", "12345678", "test", "demo"}

	for _, weak := range weakPasswords {
		if strings.ToLower(password) == weak {
			log.Println("⚠️  WARNING: Weak password detected. Please use a strong password in production.")
			break
		}
	}

	if username == password {
		log.Println("⚠️  WARNING: Username and password are identical. This is extremely insecure.")
	}

	if len(password) < 8 {
		log.Println("⚠️  WARNING: Password is shorter than 8 characters. Consider using a longer password.")
	}
}

// Run performs run operation.
func (s *EventSender) Run(ctx context.Context) error {
	log.Printf("Starting VES event sender...")
	log.Printf("Collector URL: %s", s.config.CollectorURL)
	log.Printf("Event Type: %s", s.config.EventType)
	log.Printf("Interval: %s", s.config.Interval)
	log.Printf("Count: %d", s.config.Count)
	log.Printf("Source: %s", s.config.Source)
	// Note: Never log credentials, even in debug mode.

	ticker := time.NewTicker(s.config.Interval)
	defer ticker.Stop()

	eventsSent := 0

	for {
		select {
		case <-ctx.Done():
			log.Printf("Context cancelled, sent %d events", eventsSent)
			return ctx.Err()
		case <-ticker.C:
			if s.config.Count > 0 && eventsSent >= s.config.Count {
				log.Printf("Reached target count of %d events", s.config.Count)
				return nil
			}

			event := s.createEvent()
			if err := s.sendEvent(event); err != nil {
				log.Printf("Failed to send event: %v", err)
				// Continue sending despite errors.
			} else {
				eventsSent++
				log.Printf("Sent event %d/%d (sequence: %d)", eventsSent, s.config.Count, s.sequence)
			}

			s.sequence++
		}
	}
}

func (s *EventSender) createEvent() VESEvent {
	now := time.Now()
	nowMicros := now.UnixMicro()

	header := CommonEventHeader{
		Version:             "4.1",
		Domain:              s.config.EventType,
		EventID:             s.generateEventID(),
		Sequence:            s.sequence,
		ReportingEntityName: s.config.Source,
		SourceName:          s.config.Source,
		NFVendorName:        "nephoran",
		StartEpochMicrosec:  nowMicros,
		LastEpochMicrosec:   nowMicros,
		TimeZoneOffset:      "+00:00",
	}

	payload := EventPayload{
		CommonEventHeader: header,
	}

	switch s.config.EventType {
	case "fault":
		header.EventName = "Fault_NFSim_LinkDown"
		header.Priority = "High"
		payload.FaultFields = &FaultFields{
			FaultFieldsVersion: "4.0",
			AlarmCondition:     "LINK_DOWN",
			EventSeverity:      "CRITICAL",
			SpecificProblem:    "eth0 loss of signal",
			EventSourceType:    "other",
			VFStatus:           "Active",
			AlarmInterfaceA:    "eth0",
		}

	case "measurement":
		header.EventName = "Perf_NFSim_Metrics"
		header.Priority = "Normal"
		payload.MeasurementFields = &MeasurementFields{
			MeasurementsVersion: "1.1",
			VNicUsageArray: []VNicUsage{
				{
					VNFNetworkInterface:    "eth0",
					ReceivedOctetsDelta:    1000000 + cryptoRandInt64(2000000),
					TransmittedOctetsDelta: 1500000 + cryptoRandInt64(2000000),
				},
			},
			AdditionalFields: map[string]interface{}{
				"kpm.p95_latency_ms":  85.3 + float64(cryptoRandInt64(20000))/1000.0,
				"kpm.prb_utilization": 0.3 + float64(cryptoRandInt64(600))/1000.0,
				"kpm.ue_count":        30 + int(cryptoRandInt64(40)),
			},
		}

	case "heartbeat":
		header.EventName = "Heartbeat_NFSim"
		header.Priority = "Low"
		payload.HeartbeatFields = &HeartbeatFields{
			HeartbeatFieldsVersion: "3.0",
			HeartbeatInterval:      int(s.config.Interval.Seconds()),
		}
	}

	payload.CommonEventHeader = header

	return VESEvent{Event: payload}
}

func (s *EventSender) generateEventID() string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%s-%d",
		strings.ToLower(s.config.EventType),
		s.config.Source,
		timestamp+int64(s.sequence))
}

func (s *EventSender) sendEvent(event VESEvent) error {
	jsonData, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Retry logic with exponential backoff and maximum delay cap.
	maxRetries := 3
	baseDelay := 1 * time.Second
	maxDelay := 30 * time.Second // Cap maximum delay to prevent excessive waiting

	for attempt := range maxRetries {
		err = s.sendEventAttempt(jsonData)
		if err == nil {
			return nil
		}

		if attempt < maxRetries-1 {
			// Calculate delay with exponential backoff.
			delay := time.Duration(1<<uint(attempt)) * baseDelay

			// Apply maximum delay cap.
			if delay > maxDelay {
				delay = maxDelay
			}

			// Add jitter to prevent thundering herd.
			jitter := time.Duration(cryptoRandInt64(int64(delay / 4)))
			delay = delay + jitter

			log.Printf("Attempt %d failed: %v. Retrying in %s...", attempt+1, err, delay)
			time.Sleep(delay)
		}
	}

	return fmt.Errorf("failed after %d attempts: %w", maxRetries, err)
}

// cryptoRandInt64 generates a cryptographically secure random int64 in [0, max).
func cryptoRandInt64(max int64) int64 {
	n, err := cryptorand.Int(cryptorand.Reader, big.NewInt(max))
	if err != nil {
		// Fallback to math/rand if crypto/rand fails.
		return rand.Int63n(max)
	}
	return n.Int64()
}

func (s *EventSender) sendEventAttempt(jsonData []byte) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", s.config.CollectorURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "vessend-tool/1.0")

	// Add basic auth if provided (credentials are never logged).
	if s.config.Username != "" || s.config.Password != "" {
		// Ensure both username and password are provided.
		if s.config.Username == "" || s.config.Password == "" {
			return fmt.Errorf("both username and password must be provided for basic auth")
		}
		auth := base64.StdEncoding.EncodeToString([]byte(s.config.Username + ":" + s.config.Password))
		req.Header.Set("Authorization", "Basic "+auth)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Check response status.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("HTTP error: %d %s", resp.StatusCode, resp.Status)
	}

	log.Printf("Event sent successfully (status: %d)", resp.StatusCode)
	return nil
}
