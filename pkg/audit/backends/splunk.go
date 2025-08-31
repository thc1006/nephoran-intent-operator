// Package backends provides audit logging backend implementations

// for various storage and forwarding systems.

package backends

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/thc1006/nephoran-intent-operator/pkg/audit/types"
)

var (
	splunkRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{

		Name: "splunk_audit_requests_total",

		Help: "Total number of requests to Splunk HEC",
	}, []string{"instance", "status"})

	splunkRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{

		Name: "splunk_audit_request_duration_seconds",

		Help: "Duration of Splunk HEC requests",

		Buckets: prometheus.DefBuckets,
	}, []string{"instance"})
)

// SplunkBackend implements audit logging to Splunk via HTTP Event Collector (HEC).

type SplunkBackend struct {
	config BackendConfig

	httpClient *http.Client

	logger logr.Logger

	hecURL string

	token string

	index string

	source string

	sourceType string

	host string

	// Metrics.

	eventsWritten int64

	eventsFailed int64
}

// SplunkConfig represents Splunk-specific configuration.

type SplunkConfig struct {
	HecURL string `json:"hec_url" yaml:"hec_url"`

	Token string `json:"token" yaml:"token"`

	Index string `json:"index" yaml:"index"`

	Source string `json:"source" yaml:"source"`

	SourceType string `json:"source_type" yaml:"source_type"`

	Host string `json:"host" yaml:"host"`

	// HEC settings.

	Channel string `json:"channel" yaml:"channel"`

	UseCompression bool `json:"use_compression" yaml:"use_compression"`

	MaxBatchSize int `json:"max_batch_size" yaml:"max_batch_size"`

	RequestTimeout time.Duration `json:"request_timeout" yaml:"request_timeout"`

	// Security.

	VerifyTLS bool `json:"verify_tls" yaml:"verify_tls"`

	ClientCert string `json:"client_cert" yaml:"client_cert"`

	ClientKey string `json:"client_key" yaml:"client_key"`

	CACert string `json:"ca_cert" yaml:"ca_cert"`

	// Acknowledgement.

	UseAck bool `json:"use_ack" yaml:"use_ack"`

	AckTimeout time.Duration `json:"ack_timeout" yaml:"ack_timeout"`
}

// SplunkEvent represents a Splunk HEC event.

type SplunkEvent struct {
	Time *float64 `json:"time,omitempty"`

	Host string `json:"host"`

	Source string `json:"source"`

	SourceType string `json:"sourcetype"`

	Index string `json:"index"`

	Event map[string]interface{} `json:"event"`

	Fields map[string]interface{} `json:"fields,omitempty"`
}

// SplunkResponse represents a Splunk HEC response.

type SplunkResponse struct {
	Text string `json:"text"`

	Code int `json:"code"`

	AckID *int64 `json:"ackId,omitempty"`

	InvalidLines []struct {
		Index int `json:"index"`

		Reason string `json:"reason"`
	} `json:"invalid-event-number,omitempty"`
}

// NewSplunkBackend creates a new Splunk backend.

func NewSplunkBackend(config BackendConfig) (*SplunkBackend, error) {

	splunkConfig, err := parseSplunkConfig(config.Settings)

	if err != nil {

		return nil, fmt.Errorf("invalid Splunk configuration: %w", err)

	}

	backend := &SplunkBackend{

		config: config,

		logger: log.Log.WithName("splunk-backend").WithValues("instance", config.Name),

		hecURL: splunkConfig.HecURL,

		token: splunkConfig.Token,

		index: splunkConfig.Index,

		source: splunkConfig.Source,

		sourceType: splunkConfig.SourceType,

		host: splunkConfig.Host,
	}

	// Set defaults.

	if backend.index == "" {

		backend.index = "nephoran_audit"

	}

	if backend.source == "" {

		backend.source = "nephoran-intent-operator"

	}

	if backend.sourceType == "" {

		backend.sourceType = "nephoran:audit"

	}

	if backend.host == "" {

		backend.host = "nephoran-operator"

	}

	// Validate required fields.

	if backend.hecURL == "" {

		return nil, fmt.Errorf("HEC URL is required")

	}

	if backend.token == "" {

		return nil, fmt.Errorf("HEC token is required")

	}

	// Create HTTP client.

	tlsConfig := &tls.Config{

		InsecureSkipVerify: !splunkConfig.VerifyTLS,
	}

	timeout := splunkConfig.RequestTimeout

	if timeout == 0 {

		timeout = 30 * time.Second

	}

	backend.httpClient = &http.Client{

		Timeout: timeout,

		Transport: &http.Transport{

			TLSClientConfig: tlsConfig,
		},
	}

	return backend, nil

}

// Initialize sets up the Splunk backend.

func (sb *SplunkBackend) Initialize(config BackendConfig) error {

	sb.logger.Info("Initializing Splunk backend")

	// Test connectivity by sending a health check event.

	testEvent := &types.AuditEvent{

		ID: "health-check",

		Version: "1.0",

		Timestamp: time.Now().UTC(),

		EventType: types.EventTypeHealthCheck,

		Severity: types.SeverityInfo,

		Component: "splunk-backend",

		Action: "health_check",

		Description: "Splunk backend health check",

		Result: types.ResultSuccess,
	}

	if err := sb.WriteEvent(context.Background(), testEvent); err != nil {

		return fmt.Errorf("failed to send health check event: %w", err)

	}

	sb.logger.Info("Splunk backend initialized successfully")

	return nil

}

// Type returns the backend type.

func (sb *SplunkBackend) Type() string {

	return string(BackendTypeSplunk)

}

// WriteEvent writes a single audit event to Splunk.

func (sb *SplunkBackend) WriteEvent(ctx context.Context, event *types.AuditEvent) error {

	return sb.WriteEvents(ctx, []*types.AuditEvent{event})

}

// WriteEvents writes multiple audit events to Splunk HEC.

func (sb *SplunkBackend) WriteEvents(ctx context.Context, events []*types.AuditEvent) error {

	if len(events) == 0 {

		return nil

	}

	start := time.Now()

	defer func() {

		duration := time.Since(start)

		splunkRequestDuration.WithLabelValues(sb.config.Name).Observe(duration.Seconds())

	}()

	// Convert events to Splunk format.
	// Pre-allocate slice with capacity of input events to avoid dynamic growth
	splunkEvents := make([]SplunkEvent, 0, len(events))

	for _, event := range events {

		// Filter event if necessary.

		if !sb.config.Filter.ShouldProcessEvent(event) {

			continue

		}

		filteredEvent := sb.config.Filter.ApplyFieldFilters(event)

		splunkEvent := sb.convertToSplunkEvent(filteredEvent)

		splunkEvents = append(splunkEvents, splunkEvent)

	}

	if len(splunkEvents) == 0 {

		return nil // No events to write after filtering

	}

	// Build request payload.

	var payload bytes.Buffer

	for _, splunkEvent := range splunkEvents {

		eventBytes, err := json.Marshal(splunkEvent)

		if err != nil {

			sb.logger.Error(err, "Failed to marshal Splunk event")

			atomic.AddInt64(&sb.eventsFailed, 1)

			continue

		}

		payload.Write(eventBytes)

		payload.WriteByte('\n')

	}

	// Send request.

	url := sb.hecURL + "/services/collector"

	req, err := http.NewRequestWithContext(ctx, "POST", url, &payload)

	if err != nil {

		return fmt.Errorf("failed to create HEC request: %w", err)

	}

	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", sb.token))

	req.Header.Set("Content-Type", "application/json")

	resp, err := sb.httpClient.Do(req)

	if err != nil {

		splunkRequestsTotal.WithLabelValues(sb.config.Name, "error").Inc()

		atomic.AddInt64(&sb.eventsFailed, int64(len(events)))

		return fmt.Errorf("HEC request failed: %w", err)

	}

	defer resp.Body.Close()

	// Check response.

	body, err := io.ReadAll(resp.Body)

	if err != nil {

		sb.logger.Error(err, "Failed to read HEC response")

	}

	if resp.StatusCode >= 400 {

		splunkRequestsTotal.WithLabelValues(sb.config.Name, fmt.Sprintf("%d", resp.StatusCode)).Inc()

		atomic.AddInt64(&sb.eventsFailed, int64(len(events)))

		// Try to parse error response.

		var splunkResp SplunkResponse

		if err := json.Unmarshal(body, &splunkResp); err == nil {

			return fmt.Errorf("HEC request failed: %s (code: %d)", splunkResp.Text, splunkResp.Code)

		}

		return fmt.Errorf("HEC request failed with status %d: %s", resp.StatusCode, string(body))

	}

	// Parse successful response.

	var splunkResp SplunkResponse

	if err := json.Unmarshal(body, &splunkResp); err != nil {

		sb.logger.Error(err, "Failed to parse HEC response")

	} else if len(splunkResp.InvalidLines) > 0 {

		sb.logger.Error(fmt.Errorf("some events were invalid"), "Invalid events in batch",

			"count", len(splunkResp.InvalidLines))

		atomic.AddInt64(&sb.eventsFailed, int64(len(splunkResp.InvalidLines)))

	}

	splunkRequestsTotal.WithLabelValues(sb.config.Name, "success").Inc()

	atomic.AddInt64(&sb.eventsWritten, int64(len(splunkEvents)))

	return nil

}

// Query searches for audit events in Splunk (using REST API).

func (sb *SplunkBackend) Query(ctx context.Context, query *QueryRequest) (*QueryResponse, error) {

	// Splunk search requires different authentication and API endpoints.

	// This is a simplified implementation for demonstration.

	searchQuery := sb.buildSplunkSearch(query)

	// Create search job.

	searchURL := fmt.Sprintf("%s/servicesNS/-/-/search/jobs", sb.getSplunkManagementURL())

	form := fmt.Sprintf("search=%s&earliest_time=%s&latest_time=%s&count=%d&offset=%d",

		searchQuery,

		query.StartTime.Format(time.RFC3339),

		query.EndTime.Format(time.RFC3339),

		query.Limit,

		query.Offset)

	req, err := http.NewRequestWithContext(ctx, "POST", searchURL, bytes.NewBufferString(form))

	if err != nil {

		return nil, fmt.Errorf("failed to create search request: %w", err)

	}

	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", sb.token))

	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := sb.httpClient.Do(req)

	if err != nil {

		return nil, fmt.Errorf("search request failed: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {

		body, _ := io.ReadAll(resp.Body)

		return nil, fmt.Errorf("search request failed with status %d: %s", resp.StatusCode, string(body))

	}

	// This is a simplified implementation - full Splunk integration would require.

	// parsing the search job ID, polling for completion, and retrieving results.

	return &QueryResponse{

		Events: []*types.AuditEvent{},

		TotalCount: 0,

		HasMore: false,
	}, nil

}

// Health checks the Splunk backend health.

func (sb *SplunkBackend) Health(ctx context.Context) error {

	// Check HEC health endpoint.

	url := sb.hecURL + "/services/collector/health"

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)

	if err != nil {

		return fmt.Errorf("failed to create health check request: %w", err)

	}

	req.Header.Set("Authorization", fmt.Sprintf("Splunk %s", sb.token))

	resp, err := sb.httpClient.Do(req)

	if err != nil {

		return fmt.Errorf("health check failed: %w", err)

	}

	defer resp.Body.Close()

	if resp.StatusCode >= 400 {

		return fmt.Errorf("health check failed with status %d", resp.StatusCode)

	}

	return nil

}

// Close closes the Splunk backend.

func (sb *SplunkBackend) Close() error {

	sb.logger.Info("Closing Splunk backend")

	return nil

}

// Helper methods.

func (sb *SplunkBackend) convertToSplunkEvent(event *types.AuditEvent) SplunkEvent {

	// Convert timestamp to Unix epoch with milliseconds.

	timestamp := float64(event.Timestamp.UnixNano()) / 1e9

	// Create the main event data.

	eventData := map[string]interface{}{

		"id": event.ID,

		"version": event.Version,

		"timestamp": event.Timestamp.Format(time.RFC3339Nano),

		"event_type": event.EventType,

		"category": event.Category,

		"severity": event.Severity.String(),

		"result": event.Result,

		"component": event.Component,

		"action": event.Action,

		"description": event.Description,

		"message": event.Message,
	}

	// Add context information.

	if event.UserContext != nil {

		eventData["user_context"] = event.UserContext

	}

	if event.NetworkContext != nil {

		eventData["network_context"] = event.NetworkContext

	}

	if event.SystemContext != nil {

		eventData["system_context"] = event.SystemContext

	}

	if event.ResourceContext != nil {

		eventData["resource_context"] = event.ResourceContext

	}

	// Add additional data.

	if event.Data != nil {

		eventData["data"] = event.Data

	}

	// Add error information.

	if event.Error != "" {

		eventData["error"] = event.Error

	}

	if event.ErrorCode != "" {

		eventData["error_code"] = event.ErrorCode

	}

	// Create fields for indexing.

	fields := map[string]interface{}{

		"event_type": string(event.EventType),

		"severity": event.Severity.String(),

		"component": event.Component,

		"result": string(event.Result),
	}

	if event.UserContext != nil {

		fields["user_id"] = event.UserContext.UserID

		fields["username"] = event.UserContext.Username

		fields["user_role"] = event.UserContext.Role

	}

	if event.NetworkContext != nil && event.NetworkContext.SourceIP != nil {

		fields["source_ip"] = event.NetworkContext.SourceIP.String()

	}

	// Add compliance metadata as fields.

	if event.ComplianceMetadata != nil {

		for key, value := range event.ComplianceMetadata {

			fields["compliance_"+key] = value

		}

	}

	return SplunkEvent{

		Time: &timestamp,

		Host: sb.host,

		Source: sb.source,

		SourceType: sb.sourceType,

		Index: sb.index,

		Event: eventData,

		Fields: fields,
	}

}

func (sb *SplunkBackend) buildSplunkSearch(query *QueryRequest) string {

	search := fmt.Sprintf("search index=%s", sb.index)

	// Add time range.

	search += fmt.Sprintf(" earliest=%d latest=%d",

		query.StartTime.Unix(),

		query.EndTime.Unix())

	// Add query string if provided.

	if query.Query != "" {

		search += fmt.Sprintf(" %s", query.Query)

	}

	// Add filters.

	for field, value := range query.Filters {

		search += fmt.Sprintf(" %s=\"%v\"", field, value)

	}

	// Add sorting.

	if query.SortBy != "" {

		order := "desc"

		if query.SortOrder == "asc" {

			order = "asc"

		}

		search += fmt.Sprintf(" | sort %s %s", order, query.SortBy)

	}

	return search

}

func (sb *SplunkBackend) getSplunkManagementURL() string {

	// Extract management URL from HEC URL.

	// This is a simplified approach - in practice you might have separate endpoints.

	return strings.Replace(sb.hecURL, ":8088", ":8089", 1)

}

func parseSplunkConfig(settings map[string]interface{}) (*SplunkConfig, error) {

	configBytes, err := json.Marshal(settings)

	if err != nil {

		return nil, fmt.Errorf("failed to marshal settings: %w", err)

	}

	var config SplunkConfig

	if err := json.Unmarshal(configBytes, &config); err != nil {

		return nil, fmt.Errorf("failed to unmarshal configuration: %w", err)

	}

	// Set defaults.

	if config.RequestTimeout == 0 {

		config.RequestTimeout = 30 * time.Second

	}

	if config.MaxBatchSize == 0 {

		config.MaxBatchSize = 100

	}

	if config.AckTimeout == 0 {

		config.AckTimeout = 60 * time.Second

	}

	return &config, nil

}
