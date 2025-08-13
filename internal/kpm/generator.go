// Package kpm provides E2 KPM (Key Performance Measurement) simulation capabilities
// for the Nephoran Intent Operator, generating metrics compatible with O-RAN E2SM-KPM.
package kpm

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

// KPMMetric represents a single KPM measurement from an E2 node.
// It follows the minimal structure defined in docs/contracts/e2.kpm.profile.md
// for planner consumption.
type KPMMetric struct {
	NodeID    string    `json:"node_id"`
	Timestamp time.Time `json:"timestamp"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
}

// Generator produces periodic KPM metrics for a specified E2 node.
// It simulates E2SM-KPM measurements by generating utilization metrics
// as JSON files for consumption by the planner component.
type Generator struct {
	nodeID    string
	outputDir string
}

// NewGenerator creates a new KPM metric generator for the specified node.
// It initializes the random number generator for realistic metric values.
// The nodeID identifies the E2 node and outputDir is where JSON files are written.
func NewGenerator(nodeID, outputDir string) *Generator {
	// Initialize random seed for realistic metric values
	rand.Seed(time.Now().UnixNano())
	return &Generator{
		nodeID:    nodeID,
		outputDir: outputDir,
	}
}

// GenerateMetric creates a new utilization metric and writes it to a timestamped JSON file.
// The metric value is a random float64 between 0 and 1 representing resource utilization.
// Returns an error if JSON marshaling or file writing fails.
func (g *Generator) GenerateMetric() error {
	metric := &KPMMetric{
		NodeID:    g.nodeID,
		Timestamp: time.Now().UTC(),
		Metric:    "utilization",
		Value:     rand.Float64(),
		Unit:      "ratio",
	}

	data, err := json.MarshalIndent(metric, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal metric: %w", err)
	}

	filename := fmt.Sprintf("%s_%s.json", 
		metric.Timestamp.Format("20060102T150405Z"),
		g.nodeID)
	metricPath := filepath.Join(g.outputDir, filename)

	if err := os.WriteFile(metricPath, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}