package kpm

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestNewGenerator(t *testing.T) {
	testDir := filepath.Join(os.TempDir(), "metrics")
	g, err := NewGenerator("test-node", testDir)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}
	if g.nodeID != "test-node" {
		t.Errorf("expected nodeID to be 'test-node', got %s", g.nodeID)
	}
	if g.outputDir != testDir {
		t.Errorf("expected outputDir to be '%s', got %s", testDir, g.outputDir)
	}

	// Test validation
	_, err = NewGenerator("", testDir)
	if err == nil {
		t.Error("expected error for empty nodeID")
	}

	_, err = NewGenerator("test-node", "")
	if err == nil {
		t.Error("expected error for empty outputDir")
	}
}

func TestGenerateMetric(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kpm-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	g, err := NewGenerator("test-node-001", tmpDir)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

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

func TestGeneratorConcurrency(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "kpm-concurrent-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Create multiple generators
	generators := make([]*Generator, 3)
	for i := 0; i < 3; i++ {
		g, err := NewGenerator(fmt.Sprintf("node-%d", i), tmpDir)
		if err != nil {
			t.Fatalf("Failed to create generator %d: %v", i, err)
		}
		generators[i] = g
	}

	// Generate metrics concurrently
	done := make(chan error, 3)
	for _, g := range generators {
		go func(gen *Generator) {
			done <- gen.GenerateMetric()
		}(g)
	}

	// Wait for all to complete
	for i := 0; i < 3; i++ {
		if err := <-done; err != nil {
			t.Errorf("Concurrent generation failed: %v", err)
		}
	}

	// Verify all files were created
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 3 {
		t.Errorf("Expected 3 files, got %d", len(files))
	}
}

func TestFilePermissions(t *testing.T) {
	// Skip on Windows as file permissions work differently
	if runtime.GOOS == "windows" {
		t.Skip("Skipping file permissions test on Windows")
	}

	tmpDir, err := os.MkdirTemp("", "kpm-perms-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	g, err := NewGenerator("perm-test", tmpDir)
	if err != nil {
		t.Fatalf("NewGenerator failed: %v", err)
	}

	if err := g.GenerateMetric(); err != nil {
		t.Fatalf("GenerateMetric failed: %v", err)
	}

	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatal(err)
	}

	if len(files) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(files))
	}

	info, err := files[0].Info()
	if err != nil {
		t.Fatal(err)
	}

	// Check file permissions (0600 = owner read/write only)
	mode := info.Mode().Perm()
	expectedMode := os.FileMode(0600)
	if mode != expectedMode {
		t.Errorf("Expected file permissions %o, got %o", expectedMode, mode)
	}
}
