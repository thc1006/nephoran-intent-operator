package monitoring

import "time"

// MetricsPruningEngine manages pruning of old metrics
type MetricsPruningEngine struct {
	PruningInterval time.Duration
	RetentionRules  []RetentionRule
}

// RetentionRule defines metric retention rules  
type RetentionRule struct {
	Pattern    string
	Retention time.Duration
	Priority   int
}

// StateStore interface for storing state information
type StateStore interface {
	Store(key string, value interface{}) error
	Retrieve(key string) (interface{}, error)
	Delete(key string) error
	List() ([]string, error)
}

// ValidationRule defines validation rules for monitoring
type ValidationRule struct {
	Name      string  `json:"name"`
	Type      string  `json:"type"`
	Condition string  `json:"condition"`
	Weight    float64 `json:"weight"`
	Required  bool    `json:"required"`
}