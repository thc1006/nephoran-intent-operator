package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/logging"
)

// MTLSMonitoringClient provides mTLS-enabled monitoring client functionality.
type MTLSMonitoringClient struct {
	httpClient *http.Client
	logger     *logging.StructuredLogger
}

// MetricData represents a metric data point.
type MetricData struct {
	Name      string                 `json:"name"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Type      string                 `json:"type,omitempty"` // counter, gauge, histogram
	Unit      string                 `json:"unit,omitempty"`
	Help      string                 `json:"help,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AlertData represents alert information.
type AlertData struct {
	Name         string                 `json:"name"`
	State        string                 `json:"state"`    // firing, resolved, pending
	Severity     string                 `json:"severity"` // critical, warning, info
	Message      string                 `json:"message"`
	Labels       map[string]string      `json:"labels,omitempty"`
	Annotations  map[string]string      `json:"annotations,omitempty"`
	StartsAt     time.Time              `json:"starts_at"`
	EndsAt       *time.Time             `json:"ends_at,omitempty"`
	GeneratorURL string                 `json:"generator_url,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

// LogData represents log entry.
type LogData struct {
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Source    string                 `json:"source,omitempty"`
	Labels    map[string]string      `json:"labels,omitempty"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
}

// QueryRequest represents a monitoring query request.
type QueryRequest struct {
	Query     string            `json:"query"`
	StartTime *time.Time        `json:"start_time,omitempty"`
	EndTime   *time.Time        `json:"end_time,omitempty"`
	Step      string            `json:"step,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// QueryResponse represents a monitoring query response.
type QueryResponse struct {
	Status string      `json:"status"`
	Data   interface{} `json:"data"`
	Error  string      `json:"error,omitempty"`
}

// SendMetrics sends metrics to the monitoring system using mTLS.
func (c *MTLSMonitoringClient) SendMetrics(ctx context.Context, metrics []*MetricData, endpoint string) error {
	if len(metrics) == 0 {
		return fmt.Errorf("no metrics provided")
	}

	if endpoint == "" {
		endpoint = "/api/v1/metrics"
	}

	c.logger.Debug("sending metrics via mTLS monitoring client",
		"metric_count", len(metrics),
		"endpoint", endpoint)

	// Prepare metrics payload.
	payload := map[string]interface{}{
		"metrics":   metrics,
		"timestamp": time.Now(),
		"source":    "nephoran-intent-operator",
	}

	return c.sendPayload(ctx, "POST", endpoint, payload)
}

// SendAlert sends an alert to the monitoring system.
func (c *MTLSMonitoringClient) SendAlert(ctx context.Context, alert *AlertData, endpoint string) error {
	if alert == nil {
		return fmt.Errorf("alert data cannot be nil")
	}

	if endpoint == "" {
		endpoint = "/api/v1/alerts"
	}

	c.logger.Debug("sending alert via mTLS monitoring client",
		"alert_name", alert.Name,
		"severity", alert.Severity,
		"state", alert.State,
		"endpoint", endpoint)

	// Prepare alert payload.
	payload := map[string]interface{}{
		"alerts": []*AlertData{alert},
		"source": "nephoran-intent-operator",
	}

	return c.sendPayload(ctx, "POST", endpoint, payload)
}

// SendLogs sends logs to the monitoring system.
func (c *MTLSMonitoringClient) SendLogs(ctx context.Context, logs []*LogData, endpoint string) error {
	if len(logs) == 0 {
		return fmt.Errorf("no logs provided")
	}

	if endpoint == "" {
		endpoint = "/api/v1/logs"
	}

	c.logger.Debug("sending logs via mTLS monitoring client",
		"log_count", len(logs),
		"endpoint", endpoint)

	// Prepare logs payload.
	payload := map[string]interface{}{
		"logs":   logs,
		"source": "nephoran-intent-operator",
	}

	return c.sendPayload(ctx, "POST", endpoint, payload)
}

// QueryMetrics queries metrics from the monitoring system.
func (c *MTLSMonitoringClient) QueryMetrics(ctx context.Context, query *QueryRequest, endpoint string) (*QueryResponse, error) {
	if query == nil || query.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if endpoint == "" {
		endpoint = "/api/v1/query"
	}

	c.logger.Debug("querying metrics via mTLS monitoring client",
		"query", query.Query,
		"endpoint", endpoint)

	var response QueryResponse
	if err := c.makeRequest(ctx, "POST", endpoint, query, &response); err != nil {
		return nil, fmt.Errorf("failed to query metrics: %w", err)
	}

	return &response, nil
}

// QueryRangeMetrics queries range metrics from the monitoring system.
func (c *MTLSMonitoringClient) QueryRangeMetrics(ctx context.Context, query *QueryRequest, endpoint string) (*QueryResponse, error) {
	if query == nil || query.Query == "" {
		return nil, fmt.Errorf("query cannot be empty")
	}

	if endpoint == "" {
		endpoint = "/api/v1/query_range"
	}

	c.logger.Debug("querying range metrics via mTLS monitoring client",
		"query", query.Query,
		"start_time", query.StartTime,
		"end_time", query.EndTime,
		"endpoint", endpoint)

	var response QueryResponse
	if err := c.makeRequest(ctx, "POST", endpoint, query, &response); err != nil {
		return nil, fmt.Errorf("failed to query range metrics: %w", err)
	}

	return &response, nil
}

// GetAlerts retrieves active alerts from the monitoring system.
func (c *MTLSMonitoringClient) GetAlerts(ctx context.Context, labels map[string]string, endpoint string) ([]*AlertData, error) {
	if endpoint == "" {
		endpoint = "/api/v1/alerts"
	}

	c.logger.Debug("retrieving alerts via mTLS monitoring client",
		"labels", labels,
		"endpoint", endpoint)

	// Build query parameters.
	queryParams := make(map[string]interface{})
	if len(labels) > 0 {
		queryParams["labels"] = labels
	}

	var response struct {
		Alerts []*AlertData `json:"alerts"`
		Status string       `json:"status"`
		Error  string       `json:"error,omitempty"`
	}

	if err := c.makeRequest(ctx, "GET", endpoint, queryParams, &response); err != nil {
		return nil, fmt.Errorf("failed to get alerts: %w", err)
	}

	if response.Status != "success" {
		return nil, fmt.Errorf("alerts query failed: %s", response.Error)
	}

	return response.Alerts, nil
}

// CreateSilence creates an alert silence.
func (c *MTLSMonitoringClient) CreateSilence(ctx context.Context, silence *SilenceData, endpoint string) (*SilenceResponse, error) {
	if silence == nil {
		return nil, fmt.Errorf("silence data cannot be nil")
	}

	if endpoint == "" {
		endpoint = "/api/v1/silences"
	}

	c.logger.Debug("creating silence via mTLS monitoring client",
		"comment", silence.Comment,
		"starts_at", silence.StartsAt,
		"ends_at", silence.EndsAt,
		"endpoint", endpoint)

	var response SilenceResponse
	if err := c.makeRequest(ctx, "POST", endpoint, silence, &response); err != nil {
		return nil, fmt.Errorf("failed to create silence: %w", err)
	}

	c.logger.Info("silence created successfully", "silence_id", response.SilenceID)

	return &response, nil
}

// DeleteSilence deletes an alert silence.
func (c *MTLSMonitoringClient) DeleteSilence(ctx context.Context, silenceID string, endpoint string) error {
	if silenceID == "" {
		return fmt.Errorf("silence ID cannot be empty")
	}

	if endpoint == "" {
		endpoint = fmt.Sprintf("/api/v1/silences/%s", silenceID)
	}

	c.logger.Debug("deleting silence via mTLS monitoring client",
		"silence_id", silenceID,
		"endpoint", endpoint)

	if err := c.makeRequest(ctx, "DELETE", endpoint, nil, nil); err != nil {
		return fmt.Errorf("failed to delete silence: %w", err)
	}

	c.logger.Info("silence deleted successfully", "silence_id", silenceID)

	return nil
}

// sendPayload sends a payload to the specified endpoint.
func (c *MTLSMonitoringClient) sendPayload(ctx context.Context, method, endpoint string, payload interface{}) error {
	return c.makeRequest(ctx, method, endpoint, payload, nil)
}

// makeRequest makes an HTTP request to a monitoring endpoint.
func (c *MTLSMonitoringClient) makeRequest(ctx context.Context, method, endpoint string, requestBody interface{}, responseBody interface{}) error {
	var reqBody io.Reader

	if requestBody != nil {
		jsonBody, err := json.Marshal(requestBody)
		if err != nil {
			return fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewBuffer(jsonBody)
	}

	// Create HTTP request.
	httpReq, err := http.NewRequestWithContext(ctx, method, endpoint, reqBody)
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	if requestBody != nil {
		httpReq.Header.Set("Content-Type", "application/json")
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("User-Agent", "nephoran-intent-operator/mtls")

	c.logger.Debug("making monitoring request",
		"method", method,
		"endpoint", endpoint,
		"has_body", requestBody != nil)

	// Make request.
	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to make HTTP request: %w", err)
	}
	defer resp.Body.Close()

	c.logger.Debug("received monitoring response",
		"status_code", resp.StatusCode,
		"content_length", resp.ContentLength)

	// Check for success status codes.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("request failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Decode response if needed.
	if responseBody != nil && resp.ContentLength != 0 {
		if err := json.NewDecoder(resp.Body).Decode(responseBody); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}

// GetHealth returns the health status of the monitoring service.
func (c *MTLSMonitoringClient) GetHealth() (*HealthStatus, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var health HealthStatus
	if err := c.makeRequest(ctx, "GET", "/health", nil, &health); err != nil {
		return &HealthStatus{
			Status:    "unhealthy",
			Message:   fmt.Sprintf("failed to connect: %v", err),
			Timestamp: time.Now(),
		}, nil
	}

	return &health, nil
}

// Close closes the monitoring client and cleans up resources.
func (c *MTLSMonitoringClient) Close() error {
	c.logger.Debug("closing mTLS monitoring client")

	// Close idle connections in HTTP client.
	if transport, ok := c.httpClient.Transport.(*http.Transport); ok {
		transport.CloseIdleConnections()
	}

	return nil
}

// SilenceData represents alert silence data.
type SilenceData struct {
	Matchers  []SilenceMatcher `json:"matchers"`
	StartsAt  time.Time        `json:"startsAt"`
	EndsAt    time.Time        `json:"endsAt"`
	CreatedBy string           `json:"createdBy"`
	Comment   string           `json:"comment"`
}

// SilenceMatcher represents a silence matcher.
type SilenceMatcher struct {
	Name    string `json:"name"`
	Value   string `json:"value"`
	IsRegex bool   `json:"isRegex"`
}

// SilenceResponse represents a silence creation response.
type SilenceResponse struct {
	SilenceID string `json:"silenceID"`
}
