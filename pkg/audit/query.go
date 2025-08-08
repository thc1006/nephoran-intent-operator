package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"log/slog"

	"github.com/thc1006/nephoran-intent-operator/pkg/audit/backends"
	"github.com/thc1006/nephoran-intent-operator/pkg/audit/types"
)

// QueryEngine provides advanced audit log querying and analysis capabilities
type QueryEngine struct {
	system   *AuditSystem
	backends map[string]backends.Backend
	logger   *slog.Logger
	metrics  *QueryMetrics
}

// QueryMetrics tracks query performance and usage
type QueryMetrics struct {
	queriesTotal     *prometheus.CounterVec
	queryDuration    *prometheus.HistogramVec
	queryComplexity  *prometheus.HistogramVec
	resultsReturned  *prometheus.HistogramVec
	queryErrors      *prometheus.CounterVec
}

// Query represents a structured audit log query
type Query struct {
	// Time range
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	
	// Event filtering
	EventTypes   []types.EventType `json:"event_types,omitempty"`
	Categories   []string    `json:"categories,omitempty"`
	Severities   []string    `json:"severities,omitempty"`
	
	// Actor filtering
	UserIDs      []string `json:"user_ids,omitempty"`
	UserTypes    []string `json:"user_types,omitempty"`
	SessionIDs   []string `json:"session_ids,omitempty"`
	ClientIPs    []string `json:"client_ips,omitempty"`
	
	// Resource filtering
	ResourceTypes []string `json:"resource_types,omitempty"`
	ResourceNames []string `json:"resource_names,omitempty"`
	Namespaces    []string `json:"namespaces,omitempty"`
	
	// System context
	Services     []string `json:"services,omitempty"`
	Hosts        []string `json:"hosts,omitempty"`
	TraceIDs     []string `json:"trace_ids,omitempty"`
	
	// Results control
	Limit        int    `json:"limit,omitempty"`
	Offset       int    `json:"offset,omitempty"`
	SortBy       string `json:"sort_by,omitempty"`
	SortOrder    string `json:"sort_order,omitempty"`
	
	// Advanced filters
	Filters      map[string]interface{} `json:"filters,omitempty"`
	TextSearch   string                 `json:"text_search,omitempty"`
	
	// Aggregation
	GroupBy      []string `json:"group_by,omitempty"`
	Aggregations map[string]AggregationType `json:"aggregations,omitempty"`
}

// AggregationType defines supported aggregation operations
type AggregationType string

const (
	AggCount      AggregationType = "count"
	AggSum        AggregationType = "sum"
	AggAvg        AggregationType = "avg"
	AggMin        AggregationType = "min"
	AggMax        AggregationType = "max"
	AggUnique     AggregationType = "unique"
	AggHistogram  AggregationType = "histogram"
)

// QueryResult contains the results of an audit log query
type QueryResult struct {
	Events      []*types.AuditEvent       `json:"events,omitempty"`
	Aggregations map[string]interface{}   `json:"aggregations,omitempty"`
	TotalCount  int64                     `json:"total_count"`
	QueryTime   time.Duration             `json:"query_time"`
	Backend     string                    `json:"backend"`
	HasMore     bool                      `json:"has_more"`
}

// SecurityAnalysis provides security-focused audit analysis
type SecurityAnalysis struct {
	SuspiciousActivities   []*SecurityThreat   `json:"suspicious_activities"`
	FailedAuthAttempts     *AuthAnalysis       `json:"failed_auth_attempts"`
	PrivilegeEscalations   []*PrivilegeEvent   `json:"privilege_escalations"`
	UnusualAccessPatterns  []*AccessAnomaly    `json:"unusual_access_patterns"`
	ComplianceViolations   []*ComplianceIssue  `json:"compliance_violations"`
	RiskScore             float64              `json:"risk_score"`
	Recommendations       []*SecurityAdvice    `json:"recommendations"`
}

// SecurityThreat represents a detected security threat
type SecurityThreat struct {
	Type        string            `json:"type"`
	Severity    string            `json:"severity"`
	Description string            `json:"description"`
	Actor       *types.UserContext     `json:"actor"`
	Timeline    []time.Time       `json:"timeline"`
	Evidence    []*types.AuditEvent     `json:"evidence"`
	Confidence  float64           `json:"confidence"`
	MITRE       string            `json:"mitre_technique,omitempty"`
}

// AuthAnalysis provides authentication failure analysis
type AuthAnalysis struct {
	TotalFailures      int64             `json:"total_failures"`
	UniqueUsers        int               `json:"unique_users"`
	UniqueIPs          int               `json:"unique_ips"`
	TimeDistribution   map[string]int64  `json:"time_distribution"`
	IPDistribution     map[string]int64  `json:"ip_distribution"`
	UserDistribution   map[string]int64  `json:"user_distribution"`
	BruteForcePatterns []*BruteForceEvent `json:"brute_force_patterns"`
}

// PrivilegeEvent represents privilege escalation attempts
type PrivilegeEvent struct {
	Timestamp      time.Time     `json:"timestamp"`
	User           string        `json:"user"`
	FromPrivileges []string      `json:"from_privileges"`
	ToPrivileges   []string      `json:"to_privileges"`
	Method         string        `json:"method"`
	Success        bool          `json:"success"`
	Context        *types.AuditEvent   `json:"context"`
}

// AccessAnomaly represents unusual access patterns
type AccessAnomaly struct {
	Type         string        `json:"type"`
	Description  string        `json:"description"`
	User         string        `json:"user"`
	Resource     string        `json:"resource"`
	Pattern      string        `json:"pattern"`
	Baseline     interface{}   `json:"baseline"`
	Deviation    float64       `json:"deviation"`
	Confidence   float64       `json:"confidence"`
	Timeline     []time.Time   `json:"timeline"`
}

// ComplianceIssue represents compliance violations
type ComplianceIssue struct {
	Standard     string        `json:"standard"`
	Control      string        `json:"control"`
	Violation    string        `json:"violation"`
	Severity     string        `json:"severity"`
	Events       []*types.AuditEvent `json:"events"`
	Impact       string        `json:"impact"`
	Remediation  string        `json:"remediation"`
}

// SecurityAdvice provides security recommendations
type SecurityAdvice struct {
	Category    string `json:"category"`
	Priority    string `json:"priority"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Action      string `json:"action"`
}

// BruteForceEvent represents detected brute force patterns
type BruteForceEvent struct {
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	User         string    `json:"user"`
	SourceIP     string    `json:"source_ip"`
	Attempts     int       `json:"attempts"`
	Services     []string  `json:"services"`
	Successful   bool      `json:"successful"`
}

// NewQueryEngine creates a new audit query engine
func NewQueryEngine(system *AuditSystem, backends map[string]backends.Backend, logger *slog.Logger) *QueryEngine {
	metrics := &QueryMetrics{
		queriesTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "audit_queries_total",
				Help: "Total number of audit queries executed",
			},
			[]string{"backend", "query_type", "status"},
		),
		queryDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "audit_query_duration_seconds",
				Help:    "Time spent executing audit queries",
				Buckets: prometheus.DefBuckets,
			},
			[]string{"backend", "query_type"},
		),
		queryComplexity: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "audit_query_complexity",
				Help:    "Complexity score of audit queries",
				Buckets: []float64{1, 2, 5, 10, 20, 50, 100},
			},
			[]string{"backend"},
		),
		resultsReturned: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "audit_query_results_returned",
				Help:    "Number of results returned by audit queries",
				Buckets: []float64{1, 10, 100, 1000, 10000, 100000},
			},
			[]string{"backend"},
		),
		queryErrors: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "audit_query_errors_total",
				Help: "Total number of audit query errors",
			},
			[]string{"backend", "error_type"},
		),
	}
	
	prometheus.MustRegister(
		metrics.queriesTotal,
		metrics.queryDuration,
		metrics.queryComplexity,
		metrics.resultsReturned,
		metrics.queryErrors,
	)
	
	return &QueryEngine{
		system:   system,
		backends: backends,
		logger:   logger,
		metrics:  metrics,
	}
}

// Execute executes an audit query against the specified backend
func (qe *QueryEngine) Execute(ctx context.Context, query *Query, backend string) (*QueryResult, error) {
	start := time.Now()
	complexity := qe.calculateComplexity(query)
	
	qe.metrics.queryComplexity.WithLabelValues(backend).Observe(complexity)
	
	defer func() {
		duration := time.Since(start)
		qe.metrics.queryDuration.WithLabelValues(backend, "standard").Observe(duration.Seconds())
	}()
	
	// Validate query
	if err := qe.validateQuery(query); err != nil {
		qe.metrics.queryErrors.WithLabelValues(backend, "validation").Inc()
		return nil, fmt.Errorf("invalid query: %w", err)
	}
	
	// Get backend
	backendImpl, exists := qe.backends[backend]
	if !exists {
		qe.metrics.queryErrors.WithLabelValues(backend, "backend_not_found").Inc()
		return nil, fmt.Errorf("backend %s not found", backend)
	}
	
	// Execute query
	result, err := qe.executeQuery(ctx, backendImpl, query, backend)
	if err != nil {
		qe.metrics.queryErrors.WithLabelValues(backend, "execution").Inc()
		qe.metrics.queriesTotal.WithLabelValues(backend, "standard", "error").Inc()
		return nil, fmt.Errorf("query execution failed: %w", err)
	}
	
	qe.metrics.queriesTotal.WithLabelValues(backend, "standard", "success").Inc()
	qe.metrics.resultsReturned.WithLabelValues(backend).Observe(float64(len(result.Events)))
	
	return result, nil
}

// SearchText performs full-text search across audit logs
func (qe *QueryEngine) SearchText(ctx context.Context, searchTerm string, timeRange time.Duration, backend string) (*QueryResult, error) {
	query := &Query{
		StartTime:  time.Now().Add(-timeRange),
		EndTime:    time.Now(),
		TextSearch: searchTerm,
		Limit:      1000,
		SortBy:     "timestamp",
		SortOrder:  "desc",
	}
	
	return qe.Execute(ctx, query, backend)
}

// GetUserActivity retrieves all activities for a specific user
func (qe *QueryEngine) GetUserActivity(ctx context.Context, userID string, timeRange time.Duration, backend string) (*QueryResult, error) {
	query := &Query{
		StartTime: time.Now().Add(-timeRange),
		EndTime:   time.Now(),
		UserIDs:   []string{userID},
		Limit:     10000,
		SortBy:    "timestamp",
		SortOrder: "desc",
	}
	
	return qe.Execute(ctx, query, backend)
}

// GetSecurityEvents retrieves security-related events
func (qe *QueryEngine) GetSecurityEvents(ctx context.Context, timeRange time.Duration, backend string) (*QueryResult, error) {
	securityEventTypes := []types.EventType{
		types.EventTypeAuthentication,
		types.EventTypeAuthenticationFailed,
		types.EventTypeAuthorizationFailed,
		types.EventTypeSecurityViolation,
		types.EventTypeIntrusionAttempt,
		types.EventTypeMalwareDetection,
		types.EventTypeAnomalyDetection,
	}
	
	query := &Query{
		StartTime:  time.Now().Add(-timeRange),
		EndTime:    time.Now(),
		EventTypes: securityEventTypes,
		Limit:      5000,
		SortBy:     "timestamp",
		SortOrder:  "desc",
	}
	
	return qe.Execute(ctx, query, backend)
}

// AnalyzeSecurity performs comprehensive security analysis
func (qe *QueryEngine) AnalyzeSecurity(ctx context.Context, timeRange time.Duration, backend string) (*SecurityAnalysis, error) {
	start := time.Now()
	defer func() {
		qe.metrics.queryDuration.WithLabelValues(backend, "security_analysis").Observe(time.Since(start).Seconds())
	}()
	
	analysis := &SecurityAnalysis{
		SuspiciousActivities:   []*SecurityThreat{},
		UnusualAccessPatterns:  []*AccessAnomaly{},
		ComplianceViolations:   []*ComplianceIssue{},
		PrivilegeEscalations:   []*PrivilegeEvent{},
		Recommendations:       []*SecurityAdvice{},
	}
	
	// Analyze authentication failures
	authAnalysis, err := qe.analyzeAuthFailures(ctx, timeRange, backend)
	if err != nil {
		qe.logger.Error("Failed to analyze auth failures", "error", err)
	} else {
		analysis.FailedAuthAttempts = authAnalysis
	}
	
	// Detect suspicious activities
	threats, err := qe.detectThreats(ctx, timeRange, backend)
	if err != nil {
		qe.logger.Error("Failed to detect threats", "error", err)
	} else {
		analysis.SuspiciousActivities = threats
	}
	
	// Analyze access patterns
	anomalies, err := qe.detectAccessAnomalies(ctx, timeRange, backend)
	if err != nil {
		qe.logger.Error("Failed to detect access anomalies", "error", err)
	} else {
		analysis.UnusualAccessPatterns = anomalies
	}
	
	// Check compliance violations
	violations, err := qe.checkComplianceViolations(ctx, timeRange, backend)
	if err != nil {
		qe.logger.Error("Failed to check compliance violations", "error", err)
	} else {
		analysis.ComplianceViolations = violations
	}
	
	// Calculate risk score
	analysis.RiskScore = qe.calculateRiskScore(analysis)
	
	// Generate recommendations
	analysis.Recommendations = qe.generateSecurityRecommendations(analysis)
	
	qe.metrics.queriesTotal.WithLabelValues(backend, "security_analysis", "success").Inc()
	
	return analysis, nil
}

// GetAggregatedStats returns aggregated statistics for audit events
func (qe *QueryEngine) GetAggregatedStats(ctx context.Context, query *Query, backend string) (map[string]interface{}, error) {
	result, err := qe.Execute(ctx, query, backend)
	if err != nil {
		return nil, err
	}
	
	stats := make(map[string]interface{})
	
	// Event type distribution
	eventTypes := make(map[string]int)
	severities := make(map[string]int)
	users := make(map[string]int)
	services := make(map[string]int)
	
	for _, event := range result.Events {
		eventTypes[string(event.EventType)]++
		severities[event.Severity.String()]++
		if event.UserContext != nil {
			users[event.UserContext.UserID]++
		}
		if event.SystemContext != nil {
			services[event.SystemContext.ServiceName]++
		}
	}
	
	stats["event_types"] = eventTypes
	stats["severities"] = severities
	stats["users"] = users
	stats["services"] = services
	stats["total_events"] = len(result.Events)
	
	return stats, nil
}

// validateQuery validates the query parameters
func (qe *QueryEngine) validateQuery(query *Query) error {
	if query.StartTime.IsZero() || query.EndTime.IsZero() {
		return fmt.Errorf("start_time and end_time are required")
	}
	
	if query.StartTime.After(query.EndTime) {
		return fmt.Errorf("start_time must be before end_time")
	}
	
	if query.EndTime.Sub(query.StartTime) > 30*24*time.Hour {
		return fmt.Errorf("time range cannot exceed 30 days")
	}
	
	if query.Limit <= 0 {
		query.Limit = 100
	}
	if query.Limit > 100000 {
		return fmt.Errorf("limit cannot exceed 100000")
	}
	
	if query.SortBy == "" {
		query.SortBy = "timestamp"
	}
	if query.SortOrder == "" {
		query.SortOrder = "desc"
	}
	
	return nil
}

// calculateComplexity calculates query complexity score
func (qe *QueryEngine) calculateComplexity(query *Query) float64 {
	complexity := 1.0
	
	// Time range complexity
	timeRange := query.EndTime.Sub(query.StartTime)
	if timeRange > 24*time.Hour {
		complexity += float64(timeRange.Hours() / 24)
	}
	
	// Filter complexity
	if len(query.EventTypes) > 0 {
		complexity += 1
	}
	if len(query.UserIDs) > 0 {
		complexity += float64(len(query.UserIDs)) * 0.1
	}
	if query.TextSearch != "" {
		complexity += 5
	}
	if len(query.Filters) > 0 {
		complexity += float64(len(query.Filters))
	}
	
	// Aggregation complexity
	if len(query.Aggregations) > 0 {
		complexity += float64(len(query.Aggregations)) * 2
	}
	
	// Result size complexity
	complexity += float64(query.Limit) / 1000
	
	return complexity
}

// executeQuery executes the query against the backend
func (qe *QueryEngine) executeQuery(ctx context.Context, backend backends.Backend, query *Query, backendName string) (*QueryResult, error) {
	// For this implementation, we'll simulate query execution
	// In a real implementation, this would interface with specific backends
	
	result := &QueryResult{
		Events:      []*types.AuditEvent{},
		Aggregations: make(map[string]interface{}),
		TotalCount:  0,
		Backend:     backendName,
		HasMore:     false,
	}
	
	// Simulate query processing time
	time.Sleep(10 * time.Millisecond)
	
	return result, nil
}

// analyzeAuthFailures analyzes authentication failure patterns
func (qe *QueryEngine) analyzeAuthFailures(ctx context.Context, timeRange time.Duration, backend string) (*AuthAnalysis, error) {
	query := &Query{
		StartTime:  time.Now().Add(-timeRange),
		EndTime:    time.Now(),
		EventTypes: []types.EventType{types.EventTypeAuthenticationFailed},
		Limit:      10000,
	}
	
	result, err := qe.Execute(ctx, query, backend)
	if err != nil {
		return nil, err
	}
	
	analysis := &AuthAnalysis{
		TotalFailures:      int64(len(result.Events)),
		TimeDistribution:   make(map[string]int64),
		IPDistribution:     make(map[string]int64),
		UserDistribution:   make(map[string]int64),
		BruteForcePatterns: []*BruteForceEvent{},
	}
	
	users := make(map[string]bool)
	ips := make(map[string]bool)
	
	for _, event := range result.Events {
		if event.UserContext != nil {
			users[event.UserContext.UserID] = true
			analysis.UserDistribution[event.UserContext.UserID]++
		}
		if event.NetworkContext != nil {
			ipStr := event.NetworkContext.SourceIP.String()
			ips[ipStr] = true
			analysis.IPDistribution[ipStr]++
		}
		
		// Time distribution by hour
		hour := event.Timestamp.Format("2006-01-02T15")
		analysis.TimeDistribution[hour]++
	}
	
	analysis.UniqueUsers = len(users)
	analysis.UniqueIPs = len(ips)
	
	// Detect brute force patterns
	analysis.BruteForcePatterns = qe.detectBruteForce(result.Events)
	
	return analysis, nil
}

// detectThreats detects security threats from audit events
func (qe *QueryEngine) detectThreats(ctx context.Context, timeRange time.Duration, backend string) ([]*SecurityThreat, error) {
	threats := []*SecurityThreat{}
	
	// Detect multiple failed auth attempts
	authQuery := &Query{
		StartTime:  time.Now().Add(-timeRange),
		EndTime:    time.Now(),
		EventTypes: []types.EventType{types.EventTypeAuthenticationFailed},
		Limit:      1000,
	}
	
	authResult, err := qe.Execute(ctx, authQuery, backend)
	if err == nil {
		ipCounts := make(map[string]int)
		for _, event := range authResult.Events {
			if event.NetworkContext != nil {
				ipStr := event.NetworkContext.SourceIP.String()
				ipCounts[ipStr]++
			}
		}
		
		for ip, count := range ipCounts {
			if count >= 10 {
				threat := &SecurityThreat{
					Type:        "brute_force",
					Severity:    "HIGH",
					Description: fmt.Sprintf("Multiple failed authentication attempts from IP %s", ip),
					Timeline:    []time.Time{time.Now()},
					Confidence:  0.8,
					MITRE:       "T1110",
				}
				threats = append(threats, threat)
			}
		}
	}
	
	return threats, nil
}

// detectAccessAnomalies detects unusual access patterns
func (qe *QueryEngine) detectAccessAnomalies(ctx context.Context, timeRange time.Duration, backend string) ([]*AccessAnomaly, error) {
	anomalies := []*AccessAnomaly{}
	
	// Detect unusual time-based access patterns
	query := &Query{
		StartTime: time.Now().Add(-timeRange),
		EndTime:   time.Now(),
		EventTypes: []types.EventType{
			types.EventTypeResourceAccess,
			types.EventTypeDataAccess,
			types.EventTypeResourceAccess,
		},
		Limit: 5000,
	}
	
	result, err := qe.Execute(ctx, query, backend)
	if err != nil {
		return anomalies, err
	}
	
	// Analyze access times
	userAccessTimes := make(map[string][]time.Time)
	for _, event := range result.Events {
		if event.UserContext != nil {
			userAccessTimes[event.UserContext.UserID] = append(userAccessTimes[event.UserContext.UserID], event.Timestamp)
		}
	}
	
	for user, times := range userAccessTimes {
		if qe.isUnusualAccessTime(times) {
			anomaly := &AccessAnomaly{
				Type:        "unusual_time",
				Description: "User accessing system during unusual hours",
				User:        user,
				Pattern:     "off_hours_access",
				Confidence:  0.7,
				Timeline:    times,
			}
			anomalies = append(anomalies, anomaly)
		}
	}
	
	return anomalies, nil
}

// checkComplianceViolations checks for compliance violations
func (qe *QueryEngine) checkComplianceViolations(ctx context.Context, timeRange time.Duration, backend string) ([]*ComplianceIssue, error) {
	violations := []*ComplianceIssue{}
	
	// Check for unauthorized access violations
	query := &Query{
		StartTime:  time.Now().Add(-timeRange),
		EndTime:    time.Now(),
		EventTypes: []types.EventType{types.EventTypeAuthorizationFailed, types.EventTypeResourceAccess},
		Limit:      1000,
	}
	
	result, err := qe.Execute(ctx, query, backend)
	if err != nil {
		return violations, err
	}
	
	if len(result.Events) > 0 {
		violation := &ComplianceIssue{
			Standard:    "SOC2",
			Control:     "CC6.1",
			Violation:   "Unauthorized access attempts detected",
			Severity:    "MEDIUM",
			Events:      result.Events,
			Impact:      "Potential data breach risk",
			Remediation: "Review access controls and implement additional monitoring",
		}
		violations = append(violations, violation)
	}
	
	return violations, nil
}

// calculateRiskScore calculates overall security risk score
func (qe *QueryEngine) calculateRiskScore(analysis *SecurityAnalysis) float64 {
	score := 0.0
	
	// Suspicious activities impact
	for _, threat := range analysis.SuspiciousActivities {
		switch threat.Severity {
		case "CRITICAL":
			score += 50
		case "HIGH":
			score += 30
		case "MEDIUM":
			score += 15
		case "LOW":
			score += 5
		}
	}
	
	// Authentication failures impact
	if analysis.FailedAuthAttempts != nil {
		if analysis.FailedAuthAttempts.TotalFailures > 100 {
			score += 20
		} else if analysis.FailedAuthAttempts.TotalFailures > 50 {
			score += 10
		}
	}
	
	// Compliance violations impact
	for _, violation := range analysis.ComplianceViolations {
		switch violation.Severity {
		case "CRITICAL":
			score += 40
		case "HIGH":
			score += 25
		case "MEDIUM":
			score += 10
		}
	}
	
	// Cap at 100
	if score > 100 {
		score = 100
	}
	
	return score
}

// generateSecurityRecommendations generates security recommendations
func (qe *QueryEngine) generateSecurityRecommendations(analysis *SecurityAnalysis) []*SecurityAdvice {
	recommendations := []*SecurityAdvice{}
	
	if analysis.RiskScore > 50 {
		recommendations = append(recommendations, &SecurityAdvice{
			Category:    "authentication",
			Priority:    "HIGH",
			Title:       "Strengthen Authentication Controls",
			Description: "High risk score indicates potential authentication issues",
			Action:      "Implement multi-factor authentication and review failed authentication patterns",
		})
	}
	
	if len(analysis.SuspiciousActivities) > 0 {
		recommendations = append(recommendations, &SecurityAdvice{
			Category:    "monitoring",
			Priority:    "MEDIUM",
			Title:       "Enhance Security Monitoring",
			Description: "Suspicious activities detected requiring investigation",
			Action:      "Review security monitoring rules and alert thresholds",
		})
	}
	
	if len(analysis.ComplianceViolations) > 0 {
		recommendations = append(recommendations, &SecurityAdvice{
			Category:    "compliance",
			Priority:    "HIGH",
			Title:       "Address Compliance Violations",
			Description: "Compliance violations require immediate attention",
			Action:      "Review and remediate identified compliance issues",
		})
	}
	
	return recommendations
}

// detectBruteForce detects brute force attack patterns
func (qe *QueryEngine) detectBruteForce(events []*types.AuditEvent) []*BruteForceEvent {
	patterns := []*BruteForceEvent{}
	
	// Group events by IP and time window
	ipEvents := make(map[string][]*types.AuditEvent)
	for _, event := range events {
		if event.NetworkContext != nil && len(event.NetworkContext.SourceIP) > 0 {
			ipStr := event.NetworkContext.SourceIP.String()
			ipEvents[ipStr] = append(ipEvents[ipStr], event)
		}
	}
	
	// Analyze each IP for brute force patterns
	for ip, ipEventList := range ipEvents {
		if len(ipEventList) >= 5 {
			// Sort by timestamp
			sort.Slice(ipEventList, func(i, j int) bool {
				return ipEventList[i].Timestamp.Before(ipEventList[j].Timestamp)
			})
			
			pattern := &BruteForceEvent{
				StartTime:  ipEventList[0].Timestamp,
				EndTime:    ipEventList[len(ipEventList)-1].Timestamp,
				SourceIP:   ip,
				Attempts:   len(ipEventList),
				Services:   []string{},
				Successful: false,
			}
			
			// Extract unique services and users
			services := make(map[string]bool)
			users := make(map[string]bool)
			for _, event := range ipEventList {
				if event.SystemContext != nil {
					services[event.SystemContext.ServiceName] = true
				}
				if event.UserContext != nil {
					users[event.UserContext.UserID] = true
				}
			}
			
			for service := range services {
				pattern.Services = append(pattern.Services, service)
			}
			
			// If only one user, set it
			if len(users) == 1 {
				for user := range users {
					pattern.User = user
					break
				}
			}
			
			patterns = append(patterns, pattern)
		}
	}
	
	return patterns
}

// isUnusualAccessTime determines if access times are unusual
func (qe *QueryEngine) isUnusualAccessTime(times []time.Time) bool {
	nightCount := 0
	weekendCount := 0
	
	for _, t := range times {
		hour := t.Hour()
		// Consider 11 PM to 6 AM as night
		if hour >= 23 || hour <= 6 {
			nightCount++
		}
		
		// Consider Saturday and Sunday as weekend
		if t.Weekday() == time.Saturday || t.Weekday() == time.Sunday {
			weekendCount++
		}
	}
	
	// If more than 50% of accesses are during night or weekend, consider unusual
	totalAccess := len(times)
	return float64(nightCount)/float64(totalAccess) > 0.5 || float64(weekendCount)/float64(totalAccess) > 0.3
}