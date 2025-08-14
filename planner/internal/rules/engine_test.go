package rules

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRuleEngine_ScaleOut(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test-state.json")
	defer os.Remove(tmpFile)

	engine := NewRuleEngine(Config{
		StateFile:            tmpFile,
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
	})

	now := time.Now()

	highLoadData := []KPMData{
		{Timestamp: now.Add(-20 * time.Second), NodeID: "node1", PRBUtilization: 0.85, P95Latency: 120, CurrentReplicas: 2},
		{Timestamp: now.Add(-10 * time.Second), NodeID: "node1", PRBUtilization: 0.90, P95Latency: 130, CurrentReplicas: 2},
		{Timestamp: now, NodeID: "node1", PRBUtilization: 0.88, P95Latency: 125, CurrentReplicas: 2},
	}

	var decision *ScalingDecision
	for _, data := range highLoadData {
		decision = engine.Evaluate(data)
	}

	if decision == nil {
		t.Fatal("Expected scaling decision, got nil")
	}

	if decision.Action != "scale-out" {
		t.Errorf("Expected scale-out action, got %s", decision.Action)
	}

	if decision.TargetReplicas != 3 {
		t.Errorf("Expected target replicas to be 3, got %d", decision.TargetReplicas)
	}
}

func TestRuleEngine_ScaleIn(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test-state2.json")
	defer os.Remove(tmpFile)

	engine := NewRuleEngine(Config{
		StateFile:            tmpFile,
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
	})

	engine.state.CurrentReplicas = 5
	now := time.Now()

	lowLoadData := []KPMData{
		{Timestamp: now.Add(-20 * time.Second), NodeID: "node1", PRBUtilization: 0.2, P95Latency: 30, CurrentReplicas: 5},
		{Timestamp: now.Add(-10 * time.Second), NodeID: "node1", PRBUtilization: 0.25, P95Latency: 35, CurrentReplicas: 5},
		{Timestamp: now, NodeID: "node1", PRBUtilization: 0.22, P95Latency: 32, CurrentReplicas: 5},
	}

	var decision *ScalingDecision
	for _, data := range lowLoadData {
		decision = engine.Evaluate(data)
	}

	if decision == nil {
		t.Fatal("Expected scaling decision, got nil")
	}

	if decision.Action != "scale-in" {
		t.Errorf("Expected scale-in action, got %s", decision.Action)
	}

	if decision.TargetReplicas != 4 {
		t.Errorf("Expected target replicas to be 4, got %d", decision.TargetReplicas)
	}
}

func TestRuleEngine_Cooldown(t *testing.T) {
	tmpFile := filepath.Join(os.TempDir(), "test-state3.json")
	defer os.Remove(tmpFile)

	engine := NewRuleEngine(Config{
		StateFile:            tmpFile,
		CooldownDuration:     5 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
	})

	now := time.Now()

	data1 := []KPMData{
		{Timestamp: now.Add(-20 * time.Second), NodeID: "node1", PRBUtilization: 0.85, P95Latency: 120, CurrentReplicas: 2},
		{Timestamp: now.Add(-10 * time.Second), NodeID: "node1", PRBUtilization: 0.90, P95Latency: 130, CurrentReplicas: 2},
		{Timestamp: now, NodeID: "node1", PRBUtilization: 0.88, P95Latency: 125, CurrentReplicas: 2},
	}

	var decision1 *ScalingDecision
	for _, data := range data1 {
		decision1 = engine.Evaluate(data)
	}

	if decision1 == nil {
		t.Fatal("Expected first scaling decision")
	}

	data2 := KPMData{
		Timestamp:       now.Add(1 * time.Second),
		NodeID:          "node1",
		PRBUtilization:  0.95,
		P95Latency:      150,
		CurrentReplicas: 3,
	}
	decision2 := engine.Evaluate(data2)

	if decision2 != nil {
		t.Error("Expected no decision during cooldown period")
	}
}

func TestRuleEngine_MaxReplicas(t *testing.T) {
	engine := NewRuleEngine(Config{
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          3,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
	})

	engine.state.CurrentReplicas = 3
	now := time.Now()

	highLoadData := []KPMData{
		{Timestamp: now.Add(-20 * time.Second), NodeID: "node1", PRBUtilization: 0.95, P95Latency: 150, CurrentReplicas: 3},
		{Timestamp: now.Add(-10 * time.Second), NodeID: "node1", PRBUtilization: 0.96, P95Latency: 160, CurrentReplicas: 3},
		{Timestamp: now, NodeID: "node1", PRBUtilization: 0.97, P95Latency: 170, CurrentReplicas: 3},
	}

	var decision *ScalingDecision
	for _, data := range highLoadData {
		decision = engine.Evaluate(data)
	}

	if decision != nil {
		t.Error("Expected no decision when at max replicas")
	}
}

func TestRuleEngine_MinReplicas(t *testing.T) {
	engine := NewRuleEngine(Config{
		CooldownDuration:     1 * time.Second,
		MinReplicas:          1,
		MaxReplicas:          10,
		LatencyThresholdHigh: 100.0,
		LatencyThresholdLow:  50.0,
		PRBThresholdHigh:     0.8,
		PRBThresholdLow:      0.3,
		EvaluationWindow:     30 * time.Second,
	})

	engine.state.CurrentReplicas = 1
	now := time.Now()

	lowLoadData := []KPMData{
		{Timestamp: now.Add(-20 * time.Second), NodeID: "node1", PRBUtilization: 0.1, P95Latency: 20, CurrentReplicas: 1},
		{Timestamp: now.Add(-10 * time.Second), NodeID: "node1", PRBUtilization: 0.15, P95Latency: 25, CurrentReplicas: 1},
		{Timestamp: now, NodeID: "node1", PRBUtilization: 0.12, P95Latency: 22, CurrentReplicas: 1},
	}

	var decision *ScalingDecision
	for _, data := range lowLoadData {
		decision = engine.Evaluate(data)
	}

	if decision != nil {
		t.Error("Expected no decision when at min replicas")
	}
}
