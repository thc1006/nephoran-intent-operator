package kpm

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"time"
)

type KPMMetric struct {
	NodeID    string    `json:"node_id"`
	Timestamp time.Time `json:"timestamp"`
	Metric    string    `json:"metric"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
}

type Generator struct {
	nodeID    string
	outputDir string
}

func NewGenerator(nodeID, outputDir string) *Generator {
	return &Generator{
		nodeID:    nodeID,
		outputDir: outputDir,
	}
}

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
	filepath := filepath.Join(g.outputDir, filename)

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}