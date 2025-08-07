package compliance

import (
	"sync"
	"time"

	"github.com/thc1006/nephoran-intent-operator/pkg/audit"
)

// SOC2Tracker tracks SOC 2 compliance requirements
type SOC2Tracker struct {
	mutex                sync.RWMutex
	authenticationEvents int64
	authorizationEvents  int64
	dataAccessEvents     int64
	systemChanges        int64
	securityViolations   int64
	lastActivity         time.Time
	
	// Trust Service Categories tracking
	securityControls          map[string]ControlStatus
	availabilityControls      map[string]ControlStatus
	processingIntegrityControls map[string]ControlStatus
	confidentialityControls   map[string]ControlStatus
	privacyControls           map[string]ControlStatus
}

// ISO27001Tracker tracks ISO 27001 compliance requirements
type ISO27001Tracker struct {
	mutex               sync.RWMutex
	informationSecurity int64
	accessManagement    int64
	incidentManagement  int64
	businessContinuity  int64
	lastActivity        time.Time
	
	// Annex A controls tracking
	annexAControls map[string]ControlStatus
}

// PCIDSSTracker tracks PCI DSS compliance requirements
type PCIDSSTracker struct {
	mutex                    sync.RWMutex
	cardholderDataAccess     int64
	authenticationAttempts   int64
	networkSecurityEvents   int64
	vulnerabilityEvents     int64
	lastActivity            time.Time
	
	// PCI DSS requirements tracking
	requirements map[string]ControlStatus
}

// ControlStatus represents the status of a compliance control
type ControlStatus struct {
	ControlID       string                 `json:"control_id"`
	Status          string                 `json:"status"`          // compliant, non-compliant, not-applicable
	LastAssessment  time.Time              `json:"last_assessment"`
	EvidenceCount   int                    `json:"evidence_count"`
	ViolationCount  int                    `json:"violation_count"`
	RiskLevel       string                 `json:"risk_level"`
	NextReview      time.Time              `json:"next_review"`
	Remediation     []RemediationAction    `json:"remediation"`
	Metadata        map[string]interface{} `json:"metadata"`
}

// RemediationAction represents an action to remediate compliance issues
type RemediationAction struct {
	ActionID    string    `json:"action_id"`
	Description string    `json:"description"`
	Assignee    string    `json:"assignee"`
	DueDate     time.Time `json:"due_date"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
}

// NewSOC2Tracker creates a new SOC 2 tracker
func NewSOC2Tracker() *SOC2Tracker {
	tracker := &SOC2Tracker{
		securityControls:            make(map[string]ControlStatus),
		availabilityControls:        make(map[string]ControlStatus),
		processingIntegrityControls: make(map[string]ControlStatus),
		confidentialityControls:     make(map[string]ControlStatus),
		privacyControls:             make(map[string]ControlStatus),
	}
	
	// Initialize SOC 2 controls
	tracker.initializeSOC2Controls()
	return tracker
}

// NewISO27001Tracker creates a new ISO 27001 tracker
func NewISO27001Tracker() *ISO27001Tracker {
	tracker := &ISO27001Tracker{
		annexAControls: make(map[string]ControlStatus),
	}
	
	// Initialize ISO 27001 Annex A controls
	tracker.initializeISO27001Controls()
	return tracker
}

// NewPCIDSSTracker creates a new PCI DSS tracker
func NewPCIDSSTracker() *PCIDSSTracker {
	tracker := &PCIDSSTracker{
		requirements: make(map[string]ControlStatus),
	}
	
	// Initialize PCI DSS requirements
	tracker.initializePCIDSSRequirements()
	return tracker
}

// SOC2Tracker methods

func (st *SOC2Tracker) ProcessEvent(event *audit.AuditEvent) {
	st.mutex.Lock()
	defer st.mutex.Unlock()
	
	st.lastActivity = event.Timestamp
	
	switch event.EventType {
	case audit.EventTypeAuthentication, audit.EventTypeAuthenticationFailed, audit.EventTypeAuthenticationSuccess:
		st.authenticationEvents++
		st.updateControlStatus("CC6.1", event)
		
	case audit.EventTypeAuthorization, audit.EventTypeAuthorizationFailed, audit.EventTypeAuthorizationSuccess:
		st.authorizationEvents++
		st.updateControlStatus("CC6.2", event)
		
	case audit.EventTypeDataAccess, audit.EventTypeDataCreate, audit.EventTypeDataRead, 
		 audit.EventTypeDataUpdate, audit.EventTypeDataDelete:
		st.dataAccessEvents++
		st.updateControlStatus("CC6.7", event)
		
	case audit.EventTypeSystemChange, audit.EventTypeConfigChange:
		st.systemChanges++
		st.updateControlStatus("CC8.1", event)
		
	case audit.EventTypeSecurityViolation, audit.EventTypeIntrusionAttempt:
		st.securityViolations++
		st.updateControlStatus("CC7.1", event)
	}
}

func (st *SOC2Tracker) GetStatus() map[string]interface{} {
	st.mutex.RLock()
	defer st.mutex.RUnlock()
	
	return map[string]interface{}{
		"standard":               "SOC2",
		"authentication_events":  st.authenticationEvents,
		"authorization_events":   st.authorizationEvents,
		"data_access_events":     st.dataAccessEvents,
		"system_changes":         st.systemChanges,
		"security_violations":    st.securityViolations,
		"last_activity":          st.lastActivity,
		"security_controls":      st.securityControls,
		"availability_controls":  st.availabilityControls,
		"processing_integrity":   st.processingIntegrityControls,
		"confidentiality":        st.confidentialityControls,
		"privacy":                st.privacyControls,
	}
}

func (st *SOC2Tracker) initializeSOC2Controls() {
	// Common Criteria (CC) - Security
	st.securityControls["CC6.1"] = ControlStatus{
		ControlID:      "CC6.1",
		Status:         "compliant",
		LastAssessment: time.Now().UTC(),
		RiskLevel:      "low",
		NextReview:     time.Now().AddDate(0, 3, 0), // Review quarterly
	}
	
	st.securityControls["CC6.2"] = ControlStatus{
		ControlID:      "CC6.2",
		Status:         "compliant",
		LastAssessment: time.Now().UTC(),
		RiskLevel:      "low",
		NextReview:     time.Now().AddDate(0, 3, 0),
	}
	
	st.securityControls["CC6.7"] = ControlStatus{
		ControlID:      "CC6.7",
		Status:         "compliant",
		LastAssessment: time.Now().UTC(),
		RiskLevel:      "low",
		NextReview:     time.Now().AddDate(0, 3, 0),
	}
	
	// Additional controls for Availability, Processing Integrity, etc.
	st.availabilityControls["A1.1"] = ControlStatus{
		ControlID:      "A1.1",
		Status:         "compliant",
		LastAssessment: time.Now().UTC(),
		RiskLevel:      "low",
		NextReview:     time.Now().AddDate(0, 3, 0),
	}
}

func (st *SOC2Tracker) updateControlStatus(controlID string, event *audit.AuditEvent) {
	// Update control status based on event
	if control, exists := st.securityControls[controlID]; exists {
		control.EvidenceCount++
		if event.Result == audit.ResultFailure {
			control.ViolationCount++
			if control.ViolationCount > 5 {
				control.Status = "non-compliant"
				control.RiskLevel = "high"
			}
		}
		control.LastAssessment = event.Timestamp
		st.securityControls[controlID] = control
	}
}

// ISO27001Tracker methods

func (it *ISO27001Tracker) ProcessEvent(event *audit.AuditEvent) {
	it.mutex.Lock()
	defer it.mutex.Unlock()
	
	it.lastActivity = event.Timestamp
	
	switch event.EventType {
	case audit.EventTypeAuthentication, audit.EventTypeAuthenticationFailed:
		it.accessManagement++
		it.updateControlStatus("A.9.2.1", event)
		
	case audit.EventTypeAuthorization, audit.EventTypeAuthorizationFailed:
		it.accessManagement++
		it.updateControlStatus("A.9.2.2", event)
		
	case audit.EventTypeDataAccess:
		it.informationSecurity++
		it.updateControlStatus("A.12.4.1", event)
		
	case audit.EventTypeSecurityViolation, audit.EventTypeIncidentResponse:
		it.incidentManagement++
		it.updateControlStatus("A.16.1.1", event)
		
	case audit.EventTypeSystemChange:
		it.informationSecurity++
		it.updateControlStatus("A.12.1.2", event)
	}
}

func (it *ISO27001Tracker) GetStatus() map[string]interface{} {
	it.mutex.RLock()
	defer it.mutex.RUnlock()
	
	return map[string]interface{}{
		"standard":             "ISO27001",
		"information_security": it.informationSecurity,
		"access_management":    it.accessManagement,
		"incident_management":  it.incidentManagement,
		"business_continuity":  it.businessContinuity,
		"last_activity":        it.lastActivity,
		"annex_a_controls":     it.annexAControls,
	}
}

func (it *ISO27001Tracker) initializeISO27001Controls() {
	// A.9 - Access Control
	it.annexAControls["A.9.2.1"] = ControlStatus{
		ControlID:      "A.9.2.1",
		Status:         "compliant",
		LastAssessment: time.Now().UTC(),
		RiskLevel:      "low",
		NextReview:     time.Now().AddDate(1, 0, 0), // Annual review
	}
	
	it.annexAControls["A.9.2.2"] = ControlStatus{
		ControlID:      "A.9.2.2",
		Status:         "compliant",
		LastAssessment: time.Now().UTC(),
		RiskLevel:      "low",
		NextReview:     time.Now().AddDate(1, 0, 0),
	}
	
	// A.12 - Operations Security
	it.annexAControls["A.12.4.1"] = ControlStatus{
		ControlID:      "A.12.4.1",
		Status:         "compliant",
		LastAssessment: time.Now().UTC(),
		RiskLevel:      "low",
		NextReview:     time.Now().AddDate(1, 0, 0),
	}
	
	// A.16 - Information Security Incident Management
	it.annexAControls["A.16.1.1"] = ControlStatus{
		ControlID:      "A.16.1.1",
		Status:         "compliant",
		LastAssessment: time.Now().UTC(),
		RiskLevel:      "low",
		NextReview:     time.Now().AddDate(1, 0, 0),
	}
}

func (it *ISO27001Tracker) updateControlStatus(controlID string, event *audit.AuditEvent) {
	if control, exists := it.annexAControls[controlID]; exists {
		control.EvidenceCount++
		if event.Result == audit.ResultFailure {
			control.ViolationCount++
			if control.ViolationCount > 3 {
				control.Status = "non-compliant"
				control.RiskLevel = "high"
			}
		}
		control.LastAssessment = event.Timestamp
		it.annexAControls[controlID] = control
	}
}

// PCIDSSTracker methods

func (pt *PCIDSSTracker) ProcessEvent(event *audit.AuditEvent) {
	pt.mutex.Lock()
	defer pt.mutex.Unlock()
	
	pt.lastActivity = event.Timestamp
	
	switch event.EventType {
	case audit.EventTypeAuthentication, audit.EventTypeAuthenticationFailed:
		pt.authenticationAttempts++
		pt.updateControlStatus("8.1.1", event)
		
	case audit.EventTypeDataAccess:
		pt.cardholderDataAccess++
		pt.updateControlStatus("7.1.1", event)
		pt.updateControlStatus("10.2.1", event)
		
	case audit.EventTypeNetworkAccess, audit.EventTypeFirewallRule:
		pt.networkSecurityEvents++
		pt.updateControlStatus("1.1.1", event)
		
	case audit.EventTypeVulnerability, audit.EventTypeSecurityViolation:
		pt.vulnerabilityEvents++
		pt.updateControlStatus("6.1.1", event)
	}
}

func (pt *PCIDSSTracker) GetStatus() map[string]interface{} {
	pt.mutex.RLock()
	defer pt.mutex.RUnlock()
	
	return map[string]interface{}{
		"standard":                  "PCI_DSS",
		"cardholder_data_access":    pt.cardholderDataAccess,
		"authentication_attempts":   pt.authenticationAttempts,
		"network_security_events":   pt.networkSecurityEvents,
		"vulnerability_events":      pt.vulnerabilityEvents,
		"last_activity":             pt.lastActivity,
		"requirements":              pt.requirements,
	}
}

func (pt *PCIDSSTracker) initializePCIDSSRequirements() {
	// Requirement 1: Install and maintain a firewall configuration
	pt.requirements["1.1.1"] = ControlStatus{
		ControlID:      "1.1.1",
		Status:         "compliant",
		LastAssessment: time.Now().UTC(),
		RiskLevel:      "low",
		NextReview:     time.Now().AddDate(0, 3, 0), // Quarterly review
	}
	
	// Requirement 6: Develop and maintain secure systems and applications
	pt.requirements["6.1.1"] = ControlStatus{
		ControlID:      "6.1.1",
		Status:         "compliant",
		LastAssessment: time.Now().UTC(),
		RiskLevel:      "low",
		NextReview:     time.Now().AddDate(0, 1, 0), // Monthly review
	}
	
	// Requirement 7: Restrict access to cardholder data by business need-to-know
	pt.requirements["7.1.1"] = ControlStatus{
		ControlID:      "7.1.1",
		Status:         "compliant",
		LastAssessment: time.Now().UTC(),
		RiskLevel:      "low",
		NextReview:     time.Now().AddDate(0, 3, 0),
	}
	
	// Requirement 8: Identify and authenticate access to system components
	pt.requirements["8.1.1"] = ControlStatus{
		ControlID:      "8.1.1",
		Status:         "compliant",
		LastAssessment: time.Now().UTC(),
		RiskLevel:      "low",
		NextReview:     time.Now().AddDate(0, 3, 0),
	}
	
	// Requirement 10: Track and monitor all access to network resources and cardholder data
	pt.requirements["10.2.1"] = ControlStatus{
		ControlID:      "10.2.1",
		Status:         "compliant",
		LastAssessment: time.Now().UTC(),
		RiskLevel:      "low",
		NextReview:     time.Now().AddDate(0, 3, 0),
	}
}

func (pt *PCIDSSTracker) updateControlStatus(controlID string, event *audit.AuditEvent) {
	if control, exists := pt.requirements[controlID]; exists {
		control.EvidenceCount++
		
		// PCI DSS has stricter violation thresholds
		if event.Result == audit.ResultFailure {
			control.ViolationCount++
			if control.ViolationCount > 1 { // Even single failures are significant for PCI DSS
				control.Status = "non-compliant"
				control.RiskLevel = "critical"
			}
		}
		
		control.LastAssessment = event.Timestamp
		pt.requirements[controlID] = control
	}
}

// Utility functions for all trackers

func (cs *ControlStatus) AddRemediation(action RemediationAction) {
	if cs.Remediation == nil {
		cs.Remediation = make([]RemediationAction, 0)
	}
	cs.Remediation = append(cs.Remediation, action)
}

func (cs *ControlStatus) UpdateRiskLevel() {
	if cs.ViolationCount == 0 {
		cs.RiskLevel = "low"
	} else if cs.ViolationCount <= 3 {
		cs.RiskLevel = "medium"
	} else if cs.ViolationCount <= 10 {
		cs.RiskLevel = "high"
	} else {
		cs.RiskLevel = "critical"
	}
}

func (cs *ControlStatus) IsCompliant() bool {
	return cs.Status == "compliant"
}

func (cs *ControlStatus) NeedsReview() bool {
	return time.Now().After(cs.NextReview)
}