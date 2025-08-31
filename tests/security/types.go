package security

import "time"

// ComplianceResult tracks compliance validation results
type ComplianceResult struct {
	Standard      string               `json:"standard"`
	Version       string               `json:"version"`
	Requirements  []*RequirementResult `json:"requirements"`
	OverallStatus string               `json:"overall_status"`
	Timestamp     time.Time            `json:"timestamp"`
	PassedCount   int                  `json:"passed_count"`
	FailedCount   int                  `json:"failed_count"`
}

// RequirementResult represents the result of a single compliance requirement
type RequirementResult struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Status      string    `json:"status"` // PASS, FAIL, SKIP
	Details     string    `json:"details"`
	Timestamp   time.Time `json:"timestamp"`
}