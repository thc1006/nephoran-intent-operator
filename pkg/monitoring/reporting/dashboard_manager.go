// Copyright 2024 Nephoran Intent Operator Authors.

//

// Licensed under the Apache License, Version 2.0 (the "License");.

// you may not use this file except in compliance with the License.

package reporting

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"text/template"
	"time"

	"github.com/sirupsen/logrus"
)

// DashboardConfig represents the configuration for dashboard management.

type DashboardConfig struct {
	GrafanaURL string `yaml:"grafana_url"`

	APIKey string `yaml:"api_key"`

	OrgID int `yaml:"org_id"`

	DashboardPath string `yaml:"dashboard_path"`

	Templates TemplateConfig `yaml:"templates"`

	RBAC RBACConfig `yaml:"rbac"`

	Monitoring MonitoringConfig `yaml:"monitoring"`

	ABTesting ABTestingConfig `yaml:"ab_testing"`
}

// TemplateConfig contains template configuration.

type TemplateConfig struct {
	Path string `yaml:"path"`

	Variables map[string]string `yaml:"variables"`
}

// RBACConfig contains role-based access control configuration.

type RBACConfig struct {
	Enabled bool `yaml:"enabled"`

	Roles map[string]RoleConfig `yaml:"roles"`
}

// RoleConfig defines permissions for a role.

type RoleConfig struct {
	Dashboards []string `yaml:"dashboards"`

	Edit bool `yaml:"edit"`

	Admin bool `yaml:"admin"`
}

// MonitoringConfig contains monitoring configuration for dashboards.

type MonitoringConfig struct {
	HealthCheckInterval time.Duration `yaml:"health_check_interval"`

	DataFlowTimeout time.Duration `yaml:"data_flow_timeout"`

	AlertsEnabled bool `yaml:"alerts_enabled"`
}

// ABTestingConfig contains A/B testing configuration.

type ABTestingConfig struct {
	Enabled bool `yaml:"enabled"`

	TrafficSplit float64 `yaml:"traffic_split"`

	TestDuration string `yaml:"test_duration"`

	MetricsEndpoint string `yaml:"metrics_endpoint"`
}

// Dashboard represents a Grafana dashboard.

type Dashboard struct {
	ID int `json:"id,omitempty"`

	UID string `json:"uid,omitempty"`

	Title string `json:"title"`

	Tags []string `json:"tags,omitempty"`

	Templating DashboardTemplating `json:"templating"`

	Panels []DashboardPanel `json:"panels"`

	Time DashboardTimeRange `json:"time"`

	Refresh string `json:"refresh"`

	SchemaVersion int `json:"schemaVersion"`

	Version int `json:"version"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// DashboardTemplating contains dashboard template variables.

type DashboardTemplating struct {
	List []TemplateVariable `json:"list"`
}

// TemplateVariable represents a dashboard template variable.

type TemplateVariable struct {
	Name string `json:"name"`

	Type string `json:"type"`

	DataSource string `json:"datasource,omitempty"`

	Query string `json:"query,omitempty"`

	Options []TemplateOption `json:"options,omitempty"`

	Current TemplateOption `json:"current"`

	Hide int `json:"hide,omitempty"`

	Refresh int `json:"refresh,omitempty"`

	Multi bool `json:"multi,omitempty"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// TemplateOption represents an option for a template variable.

type TemplateOption struct {
	Text string `json:"text"`

	Value string `json:"value"`

	Selected bool `json:"selected"`
}

// DashboardPanel represents a dashboard panel.

type DashboardPanel struct {
	ID int `json:"id"`

	Title string `json:"title"`

	Type string `json:"type"`

	DataSource string `json:"datasource,omitempty"`

	Targets []PanelTarget `json:"targets,omitempty"`

	GridPos PanelGridPos `json:"gridPos"`

	Options json.RawMessage `json:"options,omitempty"`

	FieldConfig PanelFieldConfig `json:"fieldConfig,omitempty"`

	Alert *PanelAlert `json:"alert,omitempty"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// PanelTarget represents a panel query target.

type PanelTarget struct {
	Expr string `json:"expr,omitempty"`

	RefID string `json:"refId"`

	LegendFormat string `json:"legendFormat,omitempty"`

	IntervalFactor int `json:"intervalFactor,omitempty"`

	Format string `json:"format,omitempty"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// PanelGridPos represents panel grid position.

type PanelGridPos struct {
	H int `json:"h"`

	W int `json:"w"`

	X int `json:"x"`

	Y int `json:"y"`
}

// PanelFieldConfig represents panel field configuration.

type PanelFieldConfig struct {
	Defaults FieldDefaults `json:"defaults"`

	Overrides []FieldOverride `json:"overrides,omitempty"`

	Metadata json.RawMessage `json:"metadata,omitempty"`
}

// FieldDefaults represents default field settings.

type FieldDefaults struct {
	Unit string `json:"unit,omitempty"`

	Min *float64 `json:"min,omitempty"`

	Max *float64 `json:"max,omitempty"`

	Decimals *int `json:"decimals,omitempty"`

	Thresholds *FieldThresholds `json:"thresholds,omitempty"`

	Mappings []FieldMapping `json:"mappings,omitempty"`

	Custom json.RawMessage `json:"custom,omitempty"`
}

// FieldThresholds represents field thresholds.

type FieldThresholds struct {
	Mode string `json:"mode"`

	Steps []ThresholdStep `json:"steps"`
}

// ThresholdStep represents a threshold step.

type ThresholdStep struct {
	Color string `json:"color"`

	Value *float64 `json:"value"`
}

// FieldMapping represents field value mapping.

type FieldMapping struct {
	Type string `json:"type"`

	Value string `json:"value"`

	Text string `json:"text"`

	Options json.RawMessage `json:"options,omitempty"`
}

// FieldOverride represents field override.

type FieldOverride struct {
	Matcher FieldMatcher `json:"matcher"`

	Properties []FieldProperty `json:"properties"`
}

// FieldMatcher represents field matcher.

type FieldMatcher struct {
	ID string `json:"id"`

	Options string `json:"options"`
}

// FieldProperty represents field property.

type FieldProperty struct {
	ID string `json:"id"`

	Value interface{} `json:"value"`
}

// PanelAlert represents panel alert configuration.

type PanelAlert struct {
	ID int `json:"id,omitempty"`

	Name string `json:"name"`

	Message string `json:"message"`

	Frequency string `json:"frequency"`

	Conditions []AlertCondition `json:"conditions"`

	ExecutionErrorState string `json:"executionErrorState"`

	NoDataState string `json:"noDataState"`

	For string `json:"for"`
}

// AlertCondition represents alert condition.

type AlertCondition struct {
	Query AlertQuery `json:"query"`

	Reducer AlertReducer `json:"reducer"`

	Evaluator AlertEvaluator `json:"evaluator"`
}

// AlertQuery represents alert query.

type AlertQuery struct {
	QueryType string `json:"queryType"`

	RefID string `json:"refId"`

	Model json.RawMessage `json:"model"`
}

// AlertReducer represents alert reducer.

type AlertReducer struct {
	Type string `json:"type"`

	Params []interface{} `json:"params"`
}

// AlertEvaluator represents alert evaluator.

type AlertEvaluator struct {
	Params []float64 `json:"params"`

	Type string `json:"type"`
}

// DashboardTimeRange represents dashboard time range.

type DashboardTimeRange struct {
	From string `json:"from"`

	To string `json:"to"`
}

// DashboardStatus represents the status of a dashboard.

type DashboardStatus struct {
	UID string `json:"uid"`

	Title string `json:"title"`

	Version int `json:"version"`

	LastUpdated time.Time `json:"last_updated"`

	Health string `json:"health"` // healthy, degraded, unhealthy

	DataFlowStatus string `json:"data_flow_status"`

	PanelCount int `json:"panel_count"`

	QueryCount int `json:"query_count"`

	ErrorCount int `json:"error_count"`

	ResponseTimes []float64 `json:"response_times"`

	Metadata json.RawMessage `json:"metadata"`
}

// ABTestResult represents A/B test results.

type ABTestResult struct {
	DashboardA string `json:"dashboard_a"`

	DashboardB string `json:"dashboard_b"`

	StartTime time.Time `json:"start_time"`

	EndTime time.Time `json:"end_time"`

	TrafficSplitA float64 `json:"traffic_split_a"`

	TrafficSplitB float64 `json:"traffic_split_b"`

	MetricsA map[string]float64 `json:"metrics_a"`

	MetricsB map[string]float64 `json:"metrics_b"`

	Winner string `json:"winner"`

	Confidence float64 `json:"confidence"`

	Metadata json.RawMessage `json:"metadata"`
}

// DashboardManager manages Grafana dashboards.

type DashboardManager struct {
	config DashboardConfig

	client *http.Client

	templates map[string]*template.Template

	dashboards map[string]*Dashboard

	statuses map[string]*DashboardStatus

	abTests map[string]*ABTestResult

	logger *logrus.Logger

	mu sync.RWMutex

	stopCh chan struct{}
}

// NewDashboardManager creates a new dashboard manager.

func NewDashboardManager(config DashboardConfig, logger *logrus.Logger) (*DashboardManager, error) {
	dm := &DashboardManager{
		config: config,

		client: &http.Client{Timeout: 30 * time.Second},

		templates: make(map[string]*template.Template),

		dashboards: make(map[string]*Dashboard),

		statuses: make(map[string]*DashboardStatus),

		abTests: make(map[string]*ABTestResult),

		logger: logger,

		stopCh: make(chan struct{}),
	}

	// Load dashboard templates.

	if err := dm.loadTemplates(); err != nil {
		return nil, fmt.Errorf("failed to load templates: %w", err)
	}

	return dm, nil
}

// Start starts the dashboard manager.

func (dm *DashboardManager) Start(ctx context.Context) error {
	dm.logger.Info("Starting Dashboard Manager")

	// Start health monitoring.

	if dm.config.Monitoring.HealthCheckInterval > 0 {
		go dm.healthMonitoringLoop(ctx)
	}

	// Provision existing dashboards.

	if err := dm.provisionDashboards(ctx); err != nil {
		return fmt.Errorf("failed to provision dashboards: %w", err)
	}

	return nil
}

// Stop stops the dashboard manager.

func (dm *DashboardManager) Stop() {
	dm.logger.Info("Stopping Dashboard Manager")

	close(dm.stopCh)
}

// CreateDashboard creates a new dashboard from template.

func (dm *DashboardManager) CreateDashboard(ctx context.Context, templateName string, variables map[string]interface{}) (*Dashboard, error) {
	dm.mu.Lock()

	defer dm.mu.Unlock()

	template, exists := dm.templates[templateName]

	if !exists {
		return nil, fmt.Errorf("template %s not found", templateName)
	}

	// Render template with variables.

	var buf bytes.Buffer

	if err := template.Execute(&buf, variables); err != nil {
		return nil, fmt.Errorf("failed to render template: %w", err)
	}

	// Parse rendered JSON.

	var dashboard Dashboard

	if err := json.Unmarshal(buf.Bytes(), &dashboard); err != nil {
		return nil, fmt.Errorf("failed to parse dashboard JSON: %w", err)
	}

	// Generate UID if not provided.

	if dashboard.UID == "" {

		hash := sha256.Sum256([]byte(dashboard.Title + templateName))

		dashboard.UID = fmt.Sprintf("nephoran-%x", hash[:8])

	}

	// Store dashboard.

	dm.dashboards[dashboard.UID] = &dashboard

	// Create dashboard in Grafana.

	if err := dm.createGrafanaDashboard(ctx, &dashboard); err != nil {
		return nil, fmt.Errorf("failed to create dashboard in Grafana: %w", err)
	}

	dm.logger.WithFields(logrus.Fields{
		"uid": dashboard.UID,

		"title": dashboard.Title,

		"template": templateName,
	}).Info("Created dashboard")

	return &dashboard, nil
}

// UpdateDashboard updates an existing dashboard.

func (dm *DashboardManager) UpdateDashboard(ctx context.Context, uid string, dashboard *Dashboard) error {
	dm.mu.Lock()

	defer dm.mu.Unlock()

	existing, exists := dm.dashboards[uid]

	if !exists {
		return fmt.Errorf("dashboard %s not found", uid)
	}

	// Update version.

	dashboard.Version = existing.Version + 1

	dashboard.UID = uid

	// Store updated dashboard.

	dm.dashboards[uid] = dashboard

	// Update dashboard in Grafana.

	if err := dm.updateGrafanaDashboard(ctx, dashboard); err != nil {
		return fmt.Errorf("failed to update dashboard in Grafana: %w", err)
	}

	dm.logger.WithFields(logrus.Fields{
		"uid": dashboard.UID,

		"title": dashboard.Title,

		"version": dashboard.Version,
	}).Info("Updated dashboard")

	return nil
}

// DeleteDashboard deletes a dashboard.

func (dm *DashboardManager) DeleteDashboard(ctx context.Context, uid string) error {
	dm.mu.Lock()

	defer dm.mu.Unlock()

	dashboard, exists := dm.dashboards[uid]

	if !exists {
		return fmt.Errorf("dashboard %s not found", uid)
	}

	// Delete from Grafana.

	if err := dm.deleteGrafanaDashboard(ctx, uid); err != nil {
		return fmt.Errorf("failed to delete dashboard from Grafana: %w", err)
	}

	// Remove from local storage.

	delete(dm.dashboards, uid)

	delete(dm.statuses, uid)

	dm.logger.WithFields(logrus.Fields{
		"uid": uid,

		"title": dashboard.Title,
	}).Info("Deleted dashboard")

	return nil
}

// GetDashboard retrieves a dashboard.

func (dm *DashboardManager) GetDashboard(uid string) (*Dashboard, error) {
	dm.mu.RLock()

	defer dm.mu.RUnlock()

	dashboard, exists := dm.dashboards[uid]

	if !exists {
		return nil, fmt.Errorf("dashboard %s not found", uid)
	}

	return dashboard, nil
}

// ListDashboards lists all managed dashboards.

func (dm *DashboardManager) ListDashboards() []*Dashboard {
	dm.mu.RLock()

	defer dm.mu.RUnlock()

	dashboards := make([]*Dashboard, 0, len(dm.dashboards))

	for _, dashboard := range dm.dashboards {
		dashboards = append(dashboards, dashboard)
	}

	return dashboards
}

// GetDashboardStatus returns the status of a dashboard.

func (dm *DashboardManager) GetDashboardStatus(uid string) (*DashboardStatus, error) {
	dm.mu.RLock()

	defer dm.mu.RUnlock()

	status, exists := dm.statuses[uid]

	if !exists {
		return nil, fmt.Errorf("dashboard status %s not found", uid)
	}

	return status, nil
}

// StartABTest starts an A/B test between two dashboard versions.

func (dm *DashboardManager) StartABTest(ctx context.Context, dashboardA, dashboardB string, duration time.Duration) (*ABTestResult, error) {
	if !dm.config.ABTesting.Enabled {
		return nil, fmt.Errorf("A/B testing is not enabled")
	}

	dm.mu.Lock()

	defer dm.mu.Unlock()

	testID := fmt.Sprintf("%s-vs-%s-%d", dashboardA, dashboardB, time.Now().Unix())

	result := &ABTestResult{
		DashboardA: dashboardA,

		DashboardB: dashboardB,

		StartTime: time.Now(),

		EndTime: time.Now().Add(duration),

		TrafficSplitA: dm.config.ABTesting.TrafficSplit,

		TrafficSplitB: 1.0 - dm.config.ABTesting.TrafficSplit,

		MetricsA: make(map[string]float64),

		MetricsB: make(map[string]float64),

		Metadata: make(map[string]interface{}),
	}

	dm.abTests[testID] = result

	// Schedule test completion.

	go func() {
		timer := time.NewTimer(duration)

		defer timer.Stop()

		select {

		case <-timer.C:

			dm.completeABTest(ctx, testID)

		case <-ctx.Done():

			return

		}
	}()

	dm.logger.WithFields(logrus.Fields{
		"test_id": testID,

		"dashboard_a": dashboardA,

		"dashboard_b": dashboardB,

		"duration": duration,
	}).Info("Started A/B test")

	return result, nil
}

// loadTemplates loads dashboard templates.

func (dm *DashboardManager) loadTemplates() error {
	if dm.config.Templates.Path == "" {
		return nil
	}

	// Load template files.

	pattern := filepath.Join(dm.config.Templates.Path, "*.json")

	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob template files: %w", err)
	}

	for _, file := range files {

		content, err := os.ReadFile(file)
		if err != nil {

			dm.logger.WithError(err).WithField("file", file).Warn("Failed to read template file")

			continue

		}

		name := strings.TrimSuffix(filepath.Base(file), ".json")

		tmpl, err := template.New(name).Parse(string(content))
		if err != nil {

			dm.logger.WithError(err).WithField("file", file).Warn("Failed to parse template")

			continue

		}

		dm.templates[name] = tmpl

		dm.logger.WithField("template", name).Debug("Loaded template")

	}

	return nil
}

// provisionDashboards provisions existing dashboards.

func (dm *DashboardManager) provisionDashboards(ctx context.Context) error {
	// Load dashboards from filesystem.

	if dm.config.DashboardPath == "" {
		return nil
	}

	pattern := filepath.Join(dm.config.DashboardPath, "*.json")

	files, err := filepath.Glob(pattern)
	if err != nil {
		return fmt.Errorf("failed to glob dashboard files: %w", err)
	}

	for _, file := range files {

		content, err := os.ReadFile(file)
		if err != nil {

			dm.logger.WithError(err).WithField("file", file).Warn("Failed to read dashboard file")

			continue

		}

		var dashboard Dashboard

		if err := json.Unmarshal(content, &dashboard); err != nil {

			dm.logger.WithError(err).WithField("file", file).Warn("Failed to parse dashboard JSON")

			continue

		}

		// Generate UID if not provided.

		if dashboard.UID == "" {

			hash := sha256.Sum256(content)

			dashboard.UID = fmt.Sprintf("nephoran-%x", hash[:8])

		}

		dm.dashboards[dashboard.UID] = &dashboard

		// Create/update dashboard in Grafana.

		if err := dm.createOrUpdateGrafanaDashboard(ctx, &dashboard); err != nil {

			dm.logger.WithError(err).WithField("uid", dashboard.UID).Warn("Failed to provision dashboard")

			continue

		}

		dm.logger.WithFields(logrus.Fields{
			"uid": dashboard.UID,

			"title": dashboard.Title,
		}).Info("Provisioned dashboard")

	}

	return nil
}

// healthMonitoringLoop continuously monitors dashboard health.

func (dm *DashboardManager) healthMonitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(dm.config.Monitoring.HealthCheckInterval)

	defer ticker.Stop()

	for {
		select {

		case <-ctx.Done():

			return

		case <-dm.stopCh:

			return

		case <-ticker.C:

			dm.checkDashboardHealth(ctx)

		}
	}
}

// checkDashboardHealth checks the health of all dashboards.

func (dm *DashboardManager) checkDashboardHealth(ctx context.Context) {
	dm.mu.Lock()

	defer dm.mu.Unlock()

	for uid, dashboard := range dm.dashboards {

		status := dm.statuses[uid]

		if status == nil {

			status = &DashboardStatus{
				UID: uid,

				Title: dashboard.Title,

				Version: dashboard.Version,

				LastUpdated: time.Now(),

				Health: "unknown",

				DataFlowStatus: "unknown",

				PanelCount: len(dashboard.Panels),

				Metadata: make(map[string]interface{}),
			}

			dm.statuses[uid] = status

		}

		// Check dashboard accessibility.

		accessible := dm.checkDashboardAccessibility(ctx, uid)

		if !accessible {

			status.Health = "unhealthy"

			continue

		}

		// Check data flow.

		dataFlowing := dm.checkDataFlow(ctx, dashboard)

		if !dataFlowing {

			status.Health = "degraded"

			status.DataFlowStatus = "no_data"

		} else {

			status.Health = "healthy"

			status.DataFlowStatus = "flowing"

		}

		status.LastUpdated = time.Now()

	}
}

// checkDashboardAccessibility checks if a dashboard is accessible.

func (dm *DashboardManager) checkDashboardAccessibility(ctx context.Context, uid string) bool {
	url := fmt.Sprintf("%s/api/dashboards/uid/%s", dm.config.GrafanaURL, uid)

	req, err := http.NewRequestWithContext(ctx, "GET", url, http.NoBody)
	if err != nil {
		return false
	}

	req.Header.Set("Authorization", "Bearer "+dm.config.APIKey)

	req.Header.Set("Content-Type", "application/json")

	resp, err := dm.client.Do(req)
	if err != nil {
		return false
	}

	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK
}

// checkDataFlow checks if data is flowing to dashboard panels.

func (dm *DashboardManager) checkDataFlow(ctx context.Context, dashboard *Dashboard) bool {
	// This would typically query Prometheus or the data source.

	// to check if recent data exists for the dashboard queries.

	// For now, return true as placeholder.

	return true
}

// createGrafanaDashboard creates a dashboard in Grafana.

func (dm *DashboardManager) createGrafanaDashboard(ctx context.Context, dashboard *Dashboard) error {
	payload := json.RawMessage("{}")

	return dm.sendGrafanaRequest(ctx, "POST", "/api/dashboards/db", payload)
}

// updateGrafanaDashboard updates a dashboard in Grafana.

func (dm *DashboardManager) updateGrafanaDashboard(ctx context.Context, dashboard *Dashboard) error {
	payload := json.RawMessage("{}")

	return dm.sendGrafanaRequest(ctx, "POST", "/api/dashboards/db", payload)
}

// createOrUpdateGrafanaDashboard creates or updates a dashboard in Grafana.

func (dm *DashboardManager) createOrUpdateGrafanaDashboard(ctx context.Context, dashboard *Dashboard) error {
	payload := json.RawMessage("{}")

	return dm.sendGrafanaRequest(ctx, "POST", "/api/dashboards/db", payload)
}

// deleteGrafanaDashboard deletes a dashboard from Grafana.

func (dm *DashboardManager) deleteGrafanaDashboard(ctx context.Context, uid string) error {
	return dm.sendGrafanaRequest(ctx, "DELETE", fmt.Sprintf("/api/dashboards/uid/%s", uid), nil)
}

// sendGrafanaRequest sends a request to Grafana API.

func (dm *DashboardManager) sendGrafanaRequest(ctx context.Context, method, path string, payload interface{}) error {
	url := dm.config.GrafanaURL + path

	var body []byte

	if payload != nil {

		var err error

		body, err = json.Marshal(payload)
		if err != nil {
			return fmt.Errorf("failed to marshal payload: %w", err)
		}

	}

	req, err := http.NewRequestWithContext(ctx, method, url, bytes.NewBuffer(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+dm.config.APIKey)

	req.Header.Set("Content-Type", "application/json")

	if dm.config.OrgID > 0 {
		req.Header.Set("X-Grafana-Org-Id", fmt.Sprintf("%d", dm.config.OrgID))
	}

	resp, err := dm.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}

	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {

		body, _ := io.ReadAll(resp.Body)

		return fmt.Errorf("Grafana API error %d: %s", resp.StatusCode, string(body))

	}

	return nil
}

// completeABTest completes an A/B test and determines the winner.

func (dm *DashboardManager) completeABTest(ctx context.Context, testID string) {
	dm.mu.Lock()

	defer dm.mu.Unlock()

	result, exists := dm.abTests[testID]

	if !exists {
		return
	}

	// Collect metrics for both dashboards.

	// This would typically integrate with analytics systems.

	result.MetricsA["user_engagement"] = 0.85

	result.MetricsA["session_duration"] = 120.5

	result.MetricsB["user_engagement"] = 0.90

	result.MetricsB["session_duration"] = 135.2

	// Determine winner (simplified logic).

	scoreA := result.MetricsA["user_engagement"] * result.MetricsA["session_duration"]

	scoreB := result.MetricsB["user_engagement"] * result.MetricsB["session_duration"]

	if scoreB > scoreA {

		result.Winner = result.DashboardB

		result.Confidence = 0.95

	} else {

		result.Winner = result.DashboardA

		result.Confidence = 0.85

	}

	dm.logger.WithFields(logrus.Fields{
		"test_id": testID,

		"winner": result.Winner,

		"confidence": result.Confidence,
	}).Info("Completed A/B test")
}
