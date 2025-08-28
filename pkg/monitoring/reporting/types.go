// Copyright 2024 Nephoran Intent Operator Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");.
// you may not use this file except in compliance with the License.

// Package reporting provides common types for monitoring and reporting functionality.
package reporting

// BusinessImpact represents the business impact of performance issues or SLA violations.
type BusinessImpact struct {
	// Cost impact.
	CostPerIntent    float64 `json:"costPerIntent,omitempty"`    // dollars
	RevenueImpact    float64 `json:"revenueImpact"`              // dollars per hour
	SLAViolationCost float64 `json:"slaViolationCost,omitempty"` // dollars
	PerformanceROI   float64 `json:"performanceROI,omitempty"`   // percentage

	// Customer impact.
	CustomerSatisfaction float64 `json:"customerSatisfaction,omitempty"` // 0-100 score
	CustomersAffected    int64   `json:"customersAffected,omitempty"`
	TransactionsLost     int64   `json:"transactionsLost,omitempty"`

	// Reputation and competitive.
	ReputationScore     float64 `json:"reputationScore,omitempty"`     // 0-100 score
	CompetitivePosition string  `json:"competitivePosition,omitempty"` // Leading, Competitive, Lagging
}

// Recommendation provides actionable performance improvement suggestions.
type Recommendation struct {
	ID           string   `json:"id,omitempty"`
	Category     string   `json:"category,omitempty"` // Performance, Cost, Reliability, etc.
	Type         string   `json:"type,omitempty"`     // performance, capacity, reliability
	Priority     string   `json:"priority"`           // Critical, High, Medium, Low
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Action       string   `json:"action,omitempty"`
	Impact       string   `json:"impact"`             // Expected improvement
	Effort       string   `json:"effort,omitempty"`   // Implementation effort
	Timeline     string   `json:"timeline,omitempty"` // Implementation timeline
	Dependencies []string `json:"dependencies,omitempty"`
}
