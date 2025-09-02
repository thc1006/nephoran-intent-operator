package o1

import (
	"context"
	"encoding/json"
	"time"
)

// Shared types used across multiple O1 managers to avoid duplicates

// EventCallback is a generic callback function for events (used by multiple managers)
// EventCallback defined in common_types.go

// SecurityRule represents a security rule (used by multiple managers)
// SecurityRule defined in common_types.go

// SecurityPolicy represents security policy configuration (used by multiple managers)
// SecurityPolicy defined in common_types.go

// ReportGeneratorInterface is a generic report generator interface
type ReportGeneratorInterface interface {
	GenerateReport(ctx context.Context, template interface{}, data interface{}) (interface{}, error)
	GetSupportedFormats() []string
	GetGeneratorType() string
}

// BaseReportGenerator provides common report generation functionality
type BaseReportGenerator struct {
	name             string
	supportedFormats []string
	config           map[string]interface{}
}

func NewBaseReportGenerator(name string, formats []string, config map[string]interface{}) *BaseReportGenerator {
	return &BaseReportGenerator{
		name:             name,
		supportedFormats: formats,
		config:           config,
	}
}

func (brg *BaseReportGenerator) GetSupportedFormats() []string {
	return brg.supportedFormats
}

func (brg *BaseReportGenerator) GetGeneratorType() string {
	return brg.name
}

func (brg *BaseReportGenerator) GenerateReport(ctx context.Context, template interface{}, data interface{}) (interface{}, error) {
	// Base implementation - to be overridden by specific generators
	return json.RawMessage(`{}`), nil
}

