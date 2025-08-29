// Package kpm provides E2 KPM (Key Performance Measurement) simulation capabilities.

// for the Nephoran Intent Operator, generating metrics compatible with O-RAN E2SM-KPM.

package kpm

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"math/big"
	"os"
	"path/filepath"
	"time"
)

// KPMMetric represents a single KPM measurement from an E2 node.

// It follows the minimal structure defined in docs/contracts/e2.kpm.profile.md.

// for planner consumption.

type KPMMetric struct {
	NodeID string `json:"node_id"`

	Timestamp time.Time `json:"timestamp"`

	Metric string `json:"metric"`

	Value float64 `json:"value"`

	Unit string `json:"unit"`
}

// Generator produces periodic KPM metrics for a specified E2 node.

// It simulates E2SM-KPM measurements by generating utilization metrics.

// as JSON files for consumption by the planner component.

type Generator struct {
	nodeID string

	outputDir string

	// Removed rng field - using crypto/rand for secure random generation

}

// NewGenerator creates a new KPM metric generator for the specified node.

// It validates inputs and initializes an instance-specific random number generator.

// The nodeID identifies the E2 node and outputDir is where JSON files are written.

// Returns an error if inputs are invalid.

func NewGenerator(nodeID, outputDir string) (*Generator, error) {

	if nodeID == "" {

		return nil, fmt.Errorf("nodeID cannot be empty")

	}

	if outputDir == "" {

		return nil, fmt.Errorf("outputDir cannot be empty")

	}

	return &Generator{

		nodeID: nodeID,

		outputDir: outputDir,
	}, nil

}

// GenerateMetric creates a new utilization metric and writes it to a timestamped JSON file.

// The metric value is a random float64 between 0 and 1 representing resource utilization.

// Returns an error if JSON marshaling or file writing fails.

func (g *Generator) GenerateMetric() error {

	// Generate cryptographically secure random value between 0 and 1.

	value, err := g.secureRandFloat64()
	if err != nil {
		return fmt.Errorf("generate random value: %w", err)
	}

	if value < 0 {

		value = 0

	} else if value > 1 {

		value = 1

	}

	metric := &KPMMetric{

		NodeID: g.nodeID,

		Timestamp: time.Now().UTC(),

		Metric: "utilization",

		Value: value,

		Unit: "ratio",
	}

	data, err := json.MarshalIndent(metric, "", "  ")

	if err != nil {

		return fmt.Errorf("marshal metric: %w", err)

	}

	filename := fmt.Sprintf("%s_%s.json",

		metric.Timestamp.Format("20060102T150405Z"),

		g.nodeID)

	metricPath := filepath.Join(g.outputDir, filename)

	if err := os.WriteFile(metricPath, data, 0o640); err != nil {

		return fmt.Errorf("write file: %w", err)

	}

	return nil

}

// secureRandFloat64 generates a cryptographically secure random float64 between 0 and 1.
func (g *Generator) secureRandFloat64() (float64, error) {
	// Generate a random 64-bit integer
	max := big.NewInt(1 << 53) // Use 53 bits to avoid precision issues with float64
	n, err := rand.Int(rand.Reader, max)
	if err != nil {
		return 0, err
	}
	
	// Convert to float64 and normalize to [0, 1)
	return float64(n.Int64()) / float64(1<<53), nil
}
