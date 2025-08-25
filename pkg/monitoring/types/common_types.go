package types

import "time"

// BusinessImpact represents the business impact of system performance or issues
type BusinessImpact struct {
	// Financial impact
	RevenueImpact     float64 `json:"revenue_impact"`        // Total revenue impact in dollars
	CostPerIntent     float64 `json:"cost_per_intent"`       // Cost per intent processing in dollars
	SLAViolationCost  float64 `json:"sla_violation_cost"`    // Cost of SLA violations in dollars
	PerformanceROI    float64 `json:"performance_roi"`       // Performance return on investment as percentage
	
	// Customer impact
	CustomersAffected    int64   `json:"customers_affected"`     // Number of customers affected
	TransactionsLost     int64   `json:"transactions_lost"`      // Number of transactions lost
	CustomerSatisfaction float64 `json:"customer_satisfaction"`  // Customer satisfaction score (0-100)
	
	// Operational impact
	ReputationScore      float64 `json:"reputation_score"`       // Reputation impact score
	ServiceDegradation   float64 `json:"service_degradation"`    // Service degradation percentage
}

// Recommendation represents system recommendations
type Recommendation struct {
	// Basic recommendation information
	ID           string    `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Priority     string    `json:"priority"`     // low, medium, high, critical
	Category     string    `json:"category"`     // performance, security, cost, reliability
	Type         string    `json:"type"`         // scaling, configuration, infrastructure
	
	// Implementation details
	Action       string                 `json:"action"`       // Specific action to take
	Parameters   map[string]interface{} `json:"parameters"`   // Action parameters
	EstimatedROI float64                `json:"estimated_roi"` // Expected return on investment
	
	// Business impact
	BusinessImpact BusinessImpact `json:"business_impact"`
	
	// Metadata
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
	Source      string                 `json:"source"`      // Source of recommendation
	Confidence  float64                `json:"confidence"`  // Confidence level (0-1)
	Status      string                 `json:"status"`      // pending, approved, implemented, rejected
	Metadata    map[string]interface{} `json:"metadata"`
}