package kpm

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewGenerator(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "metrics")
	g := NewGenerator("test-node", testDir)
	if g.nodeID != "test-node" {
		t.Errorf("expected nodeID to be 'test-node', got %s", g.nodeID)
	}
	if g.outputDir != testDir {
		t.Errorf("expected outputDir to be '%s', got %s", testDir, g.outputDir)
	}
}

func TestGenerateMetric(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kpm-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	g := NewGenerator("test-node-001", tmpDir)
	
	if err := g.GenerateMetric(); err != nil {
		t.Fatalf("GenerateMetric failed: %v", err)
	}

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	data, err := os.ReadFile(filepath.Join(tmpDir, files[0].Name()))
	if err != nil {
		t.Fatal(err)
	}

	var metric KPMMetric
	if err := json.Unmarshal(data, &metric); err != nil {
		t.Fatalf("failed to unmarshal metric: %v", err)
	}

	if metric.NodeID != "test-node-001" {
		t.Errorf("expected NodeID to be 'test-node-001', got %s", metric.NodeID)
	}

	if metric.Metric != "utilization" {
		t.Errorf("expected Metric to be 'utilization', got %s", metric.Metric)
	}

	if metric.Value < 0 || metric.Value > 1 {
		t.Errorf("expected Value to be between 0 and 1, got %f", metric.Value)
	}

	if metric.Unit != "ratio" {
		t.Errorf("expected Unit to be 'ratio', got %s", metric.Unit)
	}

	if time.Since(metric.Timestamp) > 5*time.Second {
		t.Errorf("timestamp is too old: %v", metric.Timestamp)
	}
}

// TestSample is in builder_test.go since Sample is defined in builder.go