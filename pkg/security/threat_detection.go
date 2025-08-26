// Package security implements real-time security monitoring and threat detection
// for Nephoran Intent Operator with O-RAN WG11 compliance
package security

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Threat Detection Metrics
	threatsDetectedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_threats_detected_total",
			Help: "Total number of threats detected",
		},
		[]string{"threat_type", "severity", "source"},
	)

	securityEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nephoran_security_events_total",
			Help: "Total number of security events",
		},
		[]string{"event_type", "action_taken"},
	)

	threatScoreGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nephoran_threat_score",
			Help: "Current threat score for IP addresses",
		},
		[]string{"ip_address"},
	)

	incidentResponseTime = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "nephoran_incident_response_time_seconds",
			Help: "Time taken to respond to security incidents",
		},
		[]string{"incident_type", "response_action"},
	)
)

// ThreatDetectionConfig contains threat detection configuration
type ThreatDetectionConfig struct {
	// Detection rules
	EnableBehavioralAnalysis  bool              `json:"enable_behavioral_analysis"`
	EnableSignatureDetection  bool              `json:"enable_signature_detection"`
	EnableAnomalyDetection    bool              `json:"enable_anomaly_detection"`
	EnableMLDetection         bool              `json:"enable_ml_detection"`
	
	// Thresholds
	HighThreatThreshold       int               `json:"high_threat_threshold"`     // Score threshold for high threats
	MediumThreatThreshold     int               `json:"medium_threat_threshold"`   // Score threshold for medium threats
	AnomalyThreshold          float64           `json:"anomaly_threshold"`         // Anomaly detection threshold
	
	// Time windows
	AnalysisWindow            time.Duration     `json:"analysis_window"`           // Time window for analysis
	BaselinePeriod            time.Duration     `json:"baseline_period"`           // Baseline establishment period
	AlertCooldown             time.Duration     `json:"alert_cooldown"`            // Cooldown between alerts
	
	// Response actions
	AutoBlockThreats          bool              `json:"auto_block_threats"`        // Automatically block high threats
	AutoQuarantineThreats     bool              `json:"auto_quarantine_threats"`   // Quarantine suspicious activities
	SendAlerts                bool              `json:"send_alerts"`               // Send security alerts
	
	// Integration settings
	SIEMIntegration           bool              `json:"siem_integration"`          // Enable SIEM integration
	SIEMEndpoint              string            `json:"siem_endpoint"`             // SIEM endpoint URL
	ThreatIntelFeeds          []string          `json:"threat_intel_feeds"`        // Threat intelligence feeds
	
	// Monitoring settings
	MonitoringInterval        time.Duration     `json:"monitoring_interval"`       // How often to run analysis
	RetentionPeriod           time.Duration     `json:"retention_period"`          // How long to keep data
	MaxEvents                 int               `json:"max_events"`                // Maximum events to keep in memory
}

// ThreatDetector implements comprehensive threat detection and monitoring
type ThreatDetector struct {
	config              *ThreatDetectionConfig
	logger              *slog.Logger
	
	// Detection engines
	behavioralEngine    *BehavioralAnalysisEngine
	signatureEngine     *SignatureDetectionEngine
	anomalyEngine       *AnomalyDetectionEngine
	mlEngine            *MLDetectionEngine
	
	// Event tracking
	securityEvents      []SecurityEvent
	threatScores        sync.Map // map[string]*ThreatScore
	incidentHistory     []SecurityIncident
	
	// Baseline and patterns
	trafficBaseline     *TrafficBaseline
	userBehaviorPatterns map[string]*UserBehaviorPattern
	
	// Active monitoring
	activeThreats       sync.Map // map[string]*ActiveThreat
	quarantinedIPs      sync.Map // map[string]*QuarantineInfo
	
	// Statistics
	stats               *ThreatDetectionStats
	
	// Background processing
	analysisWorkers     int
	eventQueue          chan SecurityEvent
	shutdown            chan struct{}
	wg                  sync.WaitGroup
	mu                  sync.RWMutex
}

// SecurityEvent represents a security-related event
type SecurityEvent struct {
	ID              string                 `json:"id"`
	Timestamp       time.Time              `json:"timestamp"`
	EventType       string                 `json:"event_type"`
	Severity        string                 `json:"severity"`
	Source          string                 `json:"source"`
	SourceIP        string                 `json:"source_ip"`
	Target          string                 `json:"target"`
	Description     string                 `json:"description"`
	RawData         map[string]interface{} `json:"raw_data"`
	ThreatScore     int                    `json:"threat_score"`
	Tags            []string               `json:"tags"`
	Context         map[string]string      `json:"context"`
	ResponseActions []string               `json:"response_actions"`
}

// ThreatScore tracks threat scoring for IP addresses
type ThreatScore struct {
	IP                string    `json:"ip"`
	Score             int       `json:"score"`
	LastUpdated       time.Time `json:"last_updated"`
	Events            []string  `json:"events"`
	Category          string    `json:"category"`
	Confidence        float64   `json:"confidence"`
	DecayRate         float64   `json:"decay_rate"`
}

// SecurityIncident represents a security incident
type SecurityIncident struct {
	ID              string                 `json:"id"`
	Title           string                 `json:"title"`
	Description     string                 `json:"description"`
	Severity        string                 `json:"severity"`
	Status          string                 `json:"status"`
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
	ResolvedAt      *time.Time             `json:"resolved_at,omitempty"`
	AssignedTo      string                 `json:"assigned_to"`
	Events          []string               `json:"events"`
	ResponseActions []IncidentResponse     `json:"response_actions"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// IncidentResponse represents an incident response action
type IncidentResponse struct {
	Action        string                 `json:"action"`
	Timestamp     time.Time              `json:"timestamp"`
	Result        string                 `json:"result"`
	Details       map[string]interface{} `json:"details"`
	AutomatedBy   string                 `json:"automated_by,omitempty"`
}

// ActiveThreat represents an active threat being monitored
type ActiveThreat struct {
	ID            string                 `json:"id"`
	Type          string                 `json:"type"`
	Source        string                 `json:"source"`
	FirstSeen     time.Time              `json:"first_seen"`
	LastSeen      time.Time              `json:"last_seen"`
	EventCount    int64                  `json:"event_count"`
	ThreatScore   int                    `json:"threat_score"`
	Active        bool                   `json:"active"`
	Mitigated     bool                   `json:"mitigated"`
	Indicators    []ThreatIndicator      `json:"indicators"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// ThreatIndicator represents an indicator of compromise (IoC)
type ThreatIndicator struct {
	Type        string    `json:"type"`        // ip, domain, hash, pattern
	Value       string    `json:"value"`
	Confidence  float64   `json:"confidence"`
	Source      string    `json:"source"`
	FirstSeen   time.Time `json:"first_seen"`
	LastSeen    time.Time `json:"last_seen"`
	Description string    `json:"description"`
}

// QuarantineInfo represents quarantine information
type QuarantineInfo struct {
	IP            string    `json:"ip"`
	QuarantinedAt time.Time `json:"quarantined_at"`
	ExpiresAt     time.Time `json:"expires_at"`
	Reason        string    `json:"reason"`
	ThreatScore   int       `json:"threat_score"`
	AutoQuarantine bool     `json:"auto_quarantine"`
}

// TrafficBaseline represents normal traffic patterns
type TrafficBaseline struct {
	RequestsPerMinute    float64            `json:"requests_per_minute"`
	AverageResponseTime  float64            `json:"average_response_time"`
	ErrorRate           float64            `json:"error_rate"`
	TopUserAgents       map[string]int     `json:"top_user_agents"`
	TopPaths            map[string]int     `json:"top_paths"`
	GeographicDistribution map[string]int  `json:"geographic_distribution"`
	TimeOfDayPatterns   map[int]float64    `json:"time_of_day_patterns"`
	EstablishedAt       time.Time          `json:"established_at"`
	LastUpdated         time.Time          `json:"last_updated"`
}

// UserBehaviorPattern represents user behavior patterns
type UserBehaviorPattern struct {
	UserID              string            `json:"user_id"`
	NormalAccessTimes   []time.Duration   `json:"normal_access_times"`
	TypicalPaths        map[string]int    `json:"typical_paths"`
	AverageSessionTime  time.Duration     `json:"average_session_time"`
	DeviceFingerprints  []string          `json:"device_fingerprints"`
	IPRanges            []string          `json:"ip_ranges"`
	LastUpdated         time.Time         `json:"last_updated"`
}

// ThreatDetectionStats tracks detection statistics
type ThreatDetectionStats struct {
	TotalEvents         int64     `json:"total_events"`
	ThreatsDetected     int64     `json:"threats_detected"`
	IncidentsCreated    int64     `json:"incidents_created"`
	AutoBlocks          int64     `json:"auto_blocks"`
	FalsePositives      int64     `json:"false_positives"`
	AverageResponseTime time.Duration `json:"average_response_time"`
	LastAnalysis        time.Time `json:"last_analysis"`
	SystemHealth        string    `json:"system_health"`
}

// Detection engines (simplified interfaces)
type BehavioralAnalysisEngine struct {
	patterns map[string]*BehaviorPattern
	mu       sync.RWMutex
}

type SignatureDetectionEngine struct {
	signatures []ThreatSignature
	mu         sync.RWMutex
}

type AnomalyDetectionEngine struct {
	baseline *TrafficBaseline
	mu       sync.RWMutex
}

type MLDetectionEngine struct {
	model interface{} // Placeholder for ML model
	mu    sync.RWMutex
}

// BehaviorPattern represents a behavioral pattern
type BehaviorPattern struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Indicators  []string          `json:"indicators"`
	Threshold   float64           `json:"threshold"`
	Metadata    map[string]string `json:"metadata"`
}

// ThreatSignature represents a threat signature
type ThreatSignature struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Pattern     *regexp.Regexp `json:"-"`
	PatternStr  string         `json:"pattern"`
	Severity    string         `json:"severity"`
	Category    string         `json:"category"`
	Enabled     bool           `json:"enabled"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
}

// NewThreatDetector creates a new threat detector
func NewThreatDetector(config *ThreatDetectionConfig, logger *slog.Logger) (*ThreatDetector, error) {
	if config == nil {
		config = DefaultThreatDetectionConfig()
	}

	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid threat detection config: %w", err)
	}

	td := &ThreatDetector{
		config:               config,
		logger:               logger.With(slog.String("component", "threat_detector")),
		securityEvents:       make([]SecurityEvent, 0),
		incidentHistory:      make([]SecurityIncident, 0),
		userBehaviorPatterns: make(map[string]*UserBehaviorPattern),
		stats:                &ThreatDetectionStats{SystemHealth: "healthy"},
		analysisWorkers:      4,
		eventQueue:          make(chan SecurityEvent, 1000),
		shutdown:            make(chan struct{}),
	}

	// Initialize detection engines
	if err := td.initializeEngines(); err != nil {
		return nil, fmt.Errorf("failed to initialize detection engines: %w", err)
	}

	// Start background workers
	td.startAnalysisWorkers()
	td.startPeriodicTasks()

	logger.Info("Threat detector initialized",
		slog.Bool("behavioral_analysis", config.EnableBehavioralAnalysis),
		slog.Bool("signature_detection", config.EnableSignatureDetection),
		slog.Bool("anomaly_detection", config.EnableAnomalyDetection),
		slog.Bool("ml_detection", config.EnableMLDetection))

	return td, nil
}

// initializeEngines initializes the detection engines
func (td *ThreatDetector) initializeEngines() error {
	// Initialize behavioral analysis engine
	if td.config.EnableBehavioralAnalysis {
		td.behavioralEngine = &BehavioralAnalysisEngine{
			patterns: make(map[string]*BehaviorPattern),
		}
		td.loadBehaviorPatterns()
	}

	// Initialize signature detection engine
	if td.config.EnableSignatureDetection {
		td.signatureEngine = &SignatureDetectionEngine{
			signatures: make([]ThreatSignature, 0),
		}
		td.loadThreatSignatures()
	}

	// Initialize anomaly detection engine
	if td.config.EnableAnomalyDetection {
		td.anomalyEngine = &AnomalyDetectionEngine{}
		td.establishBaseline()
	}

	// Initialize ML detection engine
	if td.config.EnableMLDetection {
		td.mlEngine = &MLDetectionEngine{}
		// Load ML model (placeholder)
	}

	return nil
}

// ProcessRequest processes an HTTP request for threat detection
func (td *ThreatDetector) ProcessRequest(r *http.Request) *SecurityEvent {
	clientIP := getClientIP(r)
	
	// Create security event
	event := SecurityEvent{
		ID:          generateEventID(),
		Timestamp:   time.Now(),
		EventType:   "http_request",
		Source:      clientIP,
		SourceIP:    clientIP,
		Target:      r.Host,
		Description: fmt.Sprintf("%s %s from %s", r.Method, r.URL.Path, clientIP),
		RawData: map[string]interface{}{
			"method":     r.Method,
			"path":       r.URL.Path,
			"user_agent": r.UserAgent(),
			"referer":    r.Referer(),
			"headers":    r.Header,
		},
		Tags:            []string{"http", "inbound"},
		Context:         make(map[string]string),
		ResponseActions: make([]string, 0),
	}

	// Analyze the request
	td.analyzeEvent(&event)
	
	// Queue event for processing
	select {
	case td.eventQueue <- event:
	default:
		td.logger.Warn("Event queue full, dropping event", slog.String("event_id", event.ID))
	}

	atomic.AddInt64(&td.stats.TotalEvents, 1)

	return &event
}

// analyzeEvent performs real-time analysis on a security event
func (td *ThreatDetector) analyzeEvent(event *SecurityEvent) {
	threatScore := 0
	
	// Signature-based detection
	if td.config.EnableSignatureDetection {
		score := td.runSignatureDetection(event)
		threatScore += score
	}

	// Behavioral analysis
	if td.config.EnableBehavioralAnalysis {
		score := td.runBehavioralAnalysis(event)
		threatScore += score
	}

	// Anomaly detection
	if td.config.EnableAnomalyDetection {
		score := td.runAnomalyDetection(event)
		threatScore += score
	}

	// ML-based detection
	if td.config.EnableMLDetection {
		score := td.runMLDetection(event)
		threatScore += score
	}

	event.ThreatScore = threatScore
	
	// Update threat score for source IP
	td.updateThreatScore(event.SourceIP, threatScore, event.EventType)

	// Determine severity based on threat score
	if threatScore >= td.config.HighThreatThreshold {
		event.Severity = "high"
		td.handleHighThreat(event)
	} else if threatScore >= td.config.MediumThreatThreshold {
		event.Severity = "medium"
		td.handleMediumThreat(event)
	} else {
		event.Severity = "low"
	}

	// Record threat if above threshold
	if threatScore > 0 {
		atomic.AddInt64(&td.stats.ThreatsDetected, 1)
		threatsDetectedTotal.WithLabelValues(event.EventType, event.Severity, event.Source).Inc()
	}
}

// runSignatureDetection runs signature-based threat detection
func (td *ThreatDetector) runSignatureDetection(event *SecurityEvent) int {
	if td.signatureEngine == nil {
		return 0
	}

	td.signatureEngine.mu.RLock()
	defer td.signatureEngine.mu.RUnlock()

	score := 0
	for _, signature := range td.signatureEngine.signatures {
		if !signature.Enabled {
			continue
		}

		// Check different fields for pattern match
		fields := []string{
			event.Description,
			fmt.Sprintf("%v", event.RawData["path"]),
			fmt.Sprintf("%v", event.RawData["user_agent"]),
		}

		for _, field := range fields {
			if signature.Pattern.MatchString(field) {
				switch signature.Severity {
				case "critical":
					score += 50
				case "high":
					score += 30
				case "medium":
					score += 15
				case "low":
					score += 5
				}
				
				event.Tags = append(event.Tags, "signature_match:"+signature.Name)
				td.logger.Warn("Threat signature matched",
					slog.String("signature", signature.Name),
					slog.String("event_id", event.ID),
					slog.String("source_ip", event.SourceIP))
				break
			}
		}
	}

	return score
}

// runBehavioralAnalysis runs behavioral analysis
func (td *ThreatDetector) runBehavioralAnalysis(event *SecurityEvent) int {
	if td.behavioralEngine == nil {
		return 0
	}

	score := 0
	
	// Check for rapid requests from same IP
	if td.isRapidRequests(event.SourceIP) {
		score += 20
		event.Tags = append(event.Tags, "rapid_requests")
	}

	// Check for suspicious user agents
	if userAgent, ok := event.RawData["user_agent"].(string); ok {
		if td.isSuspiciousUserAgent(userAgent) {
			score += 15
			event.Tags = append(event.Tags, "suspicious_user_agent")
		}
	}

	// Check for suspicious paths
	if path, ok := event.RawData["path"].(string); ok {
		if td.isSuspiciousPath(path) {
			score += 25
			event.Tags = append(event.Tags, "suspicious_path")
		}
	}

	return score
}

// runAnomalyDetection runs anomaly detection
func (td *ThreatDetector) runAnomalyDetection(event *SecurityEvent) int {
	if td.anomalyEngine == nil || td.trafficBaseline == nil {
		return 0
	}

	score := 0
	
	// Check for unusual request patterns
	if td.isAnomalousTraffic(event) {
		score += 20
		event.Tags = append(event.Tags, "anomalous_traffic")
	}

	// Check for geographic anomalies
	if td.isGeographicAnomaly(event.SourceIP) {
		score += 15
		event.Tags = append(event.Tags, "geographic_anomaly")
	}

	return score
}

// runMLDetection runs ML-based detection (placeholder)
func (td *ThreatDetector) runMLDetection(event *SecurityEvent) int {
	if td.mlEngine == nil {
		return 0
	}

	// Placeholder for ML-based detection
	// In production, this would use trained models
	return 0
}

// updateThreatScore updates the threat score for an IP address
func (td *ThreatDetector) updateThreatScore(ip string, scoreIncrement int, eventType string) {
	now := time.Now()
	
	scoreInfo, _ := td.threatScores.LoadOrStore(ip, &ThreatScore{
		IP:          ip,
		Score:       0,
		LastUpdated: now,
		Events:      make([]string, 0),
		DecayRate:   0.1, // 10% decay per hour
	})
	
	score := scoreInfo.(*ThreatScore)
	
	// Apply decay based on time since last update
	timeDiff := now.Sub(score.LastUpdated)
	decay := score.DecayRate * timeDiff.Hours()
	score.Score = int(float64(score.Score) * (1.0 - decay))
	
	// Add new score
	score.Score += scoreIncrement
	score.LastUpdated = now
	score.Events = append(score.Events, eventType)
	
	// Limit event history
	if len(score.Events) > 100 {
		score.Events = score.Events[len(score.Events)-100:]
	}

	// Update metrics
	threatScoreGauge.WithLabelValues(ip).Set(float64(score.Score))
}

// handleHighThreat handles high-severity threats
func (td *ThreatDetector) handleHighThreat(event *SecurityEvent) {
	td.logger.Error("High threat detected",
		slog.String("event_id", event.ID),
		slog.String("source_ip", event.SourceIP),
		slog.Int("threat_score", event.ThreatScore),
		slog.String("description", event.Description))

	// Auto-block if configured
	if td.config.AutoBlockThreats {
		td.blockIP(event.SourceIP, "high_threat", event.ThreatScore)
		event.ResponseActions = append(event.ResponseActions, "auto_blocked")
		atomic.AddInt64(&td.stats.AutoBlocks, 1)
	}

	// Create security incident
	td.createSecurityIncident(event)

	// Send alert
	if td.config.SendAlerts {
		td.sendSecurityAlert(event)
	}

	securityEventsTotal.WithLabelValues("high_threat", "blocked").Inc()
}

// handleMediumThreat handles medium-severity threats
func (td *ThreatDetector) handleMediumThreat(event *SecurityEvent) {
	td.logger.Warn("Medium threat detected",
		slog.String("event_id", event.ID),
		slog.String("source_ip", event.SourceIP),
		slog.Int("threat_score", event.ThreatScore))

	// Quarantine if configured
	if td.config.AutoQuarantineThreats {
		td.quarantineIP(event.SourceIP, "medium_threat", event.ThreatScore)
		event.ResponseActions = append(event.ResponseActions, "quarantined")
	}

	securityEventsTotal.WithLabelValues("medium_threat", "quarantined").Inc()
}

// blockIP blocks an IP address
func (td *ThreatDetector) blockIP(ip, reason string, threatScore int) {
	// Implementation would integrate with firewall/network policies
	td.logger.Warn("IP blocked",
		slog.String("ip", ip),
		slog.String("reason", reason),
		slog.Int("threat_score", threatScore))
}

// quarantineIP quarantines an IP address
func (td *ThreatDetector) quarantineIP(ip, reason string, threatScore int) {
	expiresAt := time.Now().Add(1 * time.Hour)
	
	quarantineInfo := &QuarantineInfo{
		IP:            ip,
		QuarantinedAt: time.Now(),
		ExpiresAt:     expiresAt,
		Reason:        reason,
		ThreatScore:   threatScore,
		AutoQuarantine: true,
	}
	
	td.quarantinedIPs.Store(ip, quarantineInfo)
	
	td.logger.Info("IP quarantined",
		slog.String("ip", ip),
		slog.String("reason", reason),
		slog.Time("expires_at", expiresAt))
}

// createSecurityIncident creates a security incident
func (td *ThreatDetector) createSecurityIncident(event *SecurityEvent) {
	incident := SecurityIncident{
		ID:          generateIncidentID(),
		Title:       fmt.Sprintf("High threat detected from %s", event.SourceIP),
		Description: event.Description,
		Severity:    event.Severity,
		Status:      "open",
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		Events:      []string{event.ID},
		ResponseActions: make([]IncidentResponse, 0),
		Metadata: map[string]interface{}{
			"threat_score":    event.ThreatScore,
			"source_ip":       event.SourceIP,
			"event_type":      event.EventType,
			"auto_created":    true,
		},
	}

	td.mu.Lock()
	td.incidentHistory = append(td.incidentHistory, incident)
	td.mu.Unlock()

	atomic.AddInt64(&td.stats.IncidentsCreated, 1)

	td.logger.Info("Security incident created",
		slog.String("incident_id", incident.ID),
		slog.String("severity", incident.Severity))
}

// sendSecurityAlert sends a security alert
func (td *ThreatDetector) sendSecurityAlert(event *SecurityEvent) {
	// Implementation would send to SIEM, email, Slack, etc.
	td.logger.Error("Security alert",
		slog.String("event_id", event.ID),
		slog.String("threat_type", event.EventType),
		slog.String("severity", event.Severity),
		slog.Int("threat_score", event.ThreatScore))
}

// startAnalysisWorkers starts background analysis workers
func (td *ThreatDetector) startAnalysisWorkers() {
	for i := 0; i < td.analysisWorkers; i++ {
		td.wg.Add(1)
		go td.analysisWorker()
	}
}

// analysisWorker processes events from the queue
func (td *ThreatDetector) analysisWorker() {
	defer td.wg.Done()
	
	for {
		select {
		case event := <-td.eventQueue:
			start := time.Now()
			
			// Store event
			td.mu.Lock()
			td.securityEvents = append(td.securityEvents, event)
			// Limit event history
			if len(td.securityEvents) > td.config.MaxEvents {
				td.securityEvents = td.securityEvents[len(td.securityEvents)-td.config.MaxEvents:]
			}
			td.mu.Unlock()
			
			// Record response time
			responseTime := time.Since(start)
			incidentResponseTime.WithLabelValues(event.EventType, "analysis").Observe(responseTime.Seconds())
			
		case <-td.shutdown:
			return
		}
	}
}

// startPeriodicTasks starts periodic maintenance tasks
func (td *ThreatDetector) startPeriodicTasks() {
	ticker := time.NewTicker(td.config.MonitoringInterval)
	
	td.wg.Add(1)
	go func() {
		defer td.wg.Done()
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				td.periodicMaintenance()
			case <-td.shutdown:
				return
			}
		}
	}()
}

// periodicMaintenance performs periodic maintenance tasks
func (td *ThreatDetector) periodicMaintenance() {
	// Update baseline
	if td.config.EnableAnomalyDetection {
		td.updateBaseline()
	}
	
	// Clean up old quarantines
	td.cleanupQuarantines()
	
	// Decay threat scores
	td.decayThreatScores()
	
	// Update statistics
	td.updateStats()
	
	td.stats.LastAnalysis = time.Now()
}

// Helper methods (simplified implementations)

func (td *ThreatDetector) loadBehaviorPatterns() {
	// Load behavioral patterns from configuration or database
}

func (td *ThreatDetector) loadThreatSignatures() {
	// Common threat signatures
	signatures := []ThreatSignature{
		{
			ID:          "sql_injection",
			Name:        "SQL Injection",
			Description: "Detects SQL injection attempts",
			PatternStr:  `(?i)(union|select|insert|delete|update|drop|create|alter|exec|script|javascript)`,
			Severity:    "high",
			Category:    "injection",
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "xss_attempt",
			Name:        "Cross-Site Scripting",
			Description: "Detects XSS attempts",
			PatternStr:  `(?i)(<script|javascript:|vbscript:|onload=|onerror=|onclick=)`,
			Severity:    "high",
			Category:    "injection",
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
		{
			ID:          "directory_traversal",
			Name:        "Directory Traversal",
			Description: "Detects directory traversal attempts",
			PatternStr:  `(\.\.\/|\.\.\\|%2e%2e%2f|%2e%2e%5c)`,
			Severity:    "medium",
			Category:    "traversal",
			Enabled:     true,
			CreatedAt:   time.Now(),
		},
	}

	for i := range signatures {
		pattern, err := regexp.Compile(signatures[i].PatternStr)
		if err != nil {
			td.logger.Warn("Invalid threat signature pattern",
				slog.String("signature", signatures[i].Name),
				slog.String("error", err.Error()))
			continue
		}
		signatures[i].Pattern = pattern
	}

	td.signatureEngine.signatures = signatures
}

func (td *ThreatDetector) establishBaseline() {
	// Establish traffic baseline
	td.trafficBaseline = &TrafficBaseline{
		EstablishedAt: time.Now(),
		LastUpdated:   time.Now(),
	}
}

func (td *ThreatDetector) updateBaseline() {
	// Update traffic baseline with recent data
}

func (td *ThreatDetector) isRapidRequests(ip string) bool {
	// Check if IP is making rapid requests
	return false // Placeholder
}

func (td *ThreatDetector) isSuspiciousUserAgent(userAgent string) bool {
	suspicious := []string{"bot", "crawler", "scanner", "test", "curl", "wget"}
	ua := strings.ToLower(userAgent)
	for _, pattern := range suspicious {
		if strings.Contains(ua, pattern) {
			return true
		}
	}
	return false
}

func (td *ThreatDetector) isSuspiciousPath(path string) bool {
	suspicious := []string{"admin", "config", "backup", ".env", "wp-admin", "phpmyadmin"}
	pathLower := strings.ToLower(path)
	for _, pattern := range suspicious {
		if strings.Contains(pathLower, pattern) {
			return true
		}
	}
	return false
}

func (td *ThreatDetector) isAnomalousTraffic(event *SecurityEvent) bool {
	// Check for anomalous traffic patterns
	return false // Placeholder
}

func (td *ThreatDetector) isGeographicAnomaly(ip string) bool {
	// Check for geographic anomalies
	return false // Placeholder
}

func (td *ThreatDetector) cleanupQuarantines() {
	now := time.Now()
	td.quarantinedIPs.Range(func(key, value interface{}) bool {
		quarantine := value.(*QuarantineInfo)
		if now.After(quarantine.ExpiresAt) {
			td.quarantinedIPs.Delete(key)
		}
		return true
	})
}

func (td *ThreatDetector) decayThreatScores() {
	now := time.Now()
	td.threatScores.Range(func(key, value interface{}) bool {
		score := value.(*ThreatScore)
		timeDiff := now.Sub(score.LastUpdated)
		decay := score.DecayRate * timeDiff.Hours()
		score.Score = int(float64(score.Score) * (1.0 - decay))
		
		if score.Score <= 0 {
			td.threatScores.Delete(key)
			threatScoreGauge.DeleteLabelValues(score.IP)
		} else {
			threatScoreGauge.WithLabelValues(score.IP).Set(float64(score.Score))
		}
		return true
	})
}

func (td *ThreatDetector) updateStats() {
	td.stats.SystemHealth = "healthy" // Simplified
}

// GetStats returns threat detection statistics
func (td *ThreatDetector) GetStats() *ThreatDetectionStats {
	return td.stats
}

// Close shuts down the threat detector
func (td *ThreatDetector) Close() error {
	close(td.shutdown)
	td.wg.Wait()
	
	td.logger.Info("Threat detector shut down")
	return nil
}

// Helper functions

func generateEventID() string {
	return fmt.Sprintf("evt-%d", time.Now().UnixNano())
}

func generateIncidentID() string {
	return fmt.Sprintf("inc-%d", time.Now().UnixNano())
}

// DefaultThreatDetectionConfig returns default configuration
func DefaultThreatDetectionConfig() *ThreatDetectionConfig {
	return &ThreatDetectionConfig{
		EnableBehavioralAnalysis: true,
		EnableSignatureDetection: true,
		EnableAnomalyDetection:   true,
		EnableMLDetection:        false,
		HighThreatThreshold:      80,
		MediumThreatThreshold:    40,
		AnomalyThreshold:         0.95,
		AnalysisWindow:           5 * time.Minute,
		BaselinePeriod:           24 * time.Hour,
		AlertCooldown:            15 * time.Minute,
		AutoBlockThreats:         true,
		AutoQuarantineThreats:    true,
		SendAlerts:               true,
		SIEMIntegration:          false,
		MonitoringInterval:       1 * time.Minute,
		RetentionPeriod:          7 * 24 * time.Hour,
		MaxEvents:                10000,
	}
}

// Validate validates the threat detection configuration
func (config *ThreatDetectionConfig) Validate() error {
	if config.HighThreatThreshold <= config.MediumThreatThreshold {
		return fmt.Errorf("high threat threshold must be greater than medium threshold")
	}
	if config.MaxEvents <= 0 {
		return fmt.Errorf("max events must be positive")
	}
	if config.MonitoringInterval <= 0 {
		return fmt.Errorf("monitoring interval must be positive")
	}
	return nil
}