package chaos

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// PredefinedExperiments provides a catalog of pre-configured chaos experiments.
// specifically designed for telecommunications systems.
type PredefinedExperiments struct {
	logger *zap.Logger
	engine *ChaosEngine
}

// NewPredefinedExperiments creates a new catalog of predefined experiments.
func NewPredefinedExperiments(logger *zap.Logger, engine *ChaosEngine) *PredefinedExperiments {
	return &PredefinedExperiments{
		logger: logger,
		engine: engine,
	}
}

// GetNetworkLatencyExperiment creates a network latency injection experiment.
func (p *PredefinedExperiments) GetNetworkLatencyExperiment(targetNamespace string, latencyMS int) *Experiment {
	return &Experiment{
		Name:        "network-latency-injection",
		Description: "Inject network latency to test system tolerance",
		Type:        ExperimentTypeNetwork,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
			LabelSelector: map[string]string{
				"app.kubernetes.io/component": "network-function",
			},
		},
		Parameters: map[string]string{
			"latency":     fmt.Sprintf("%dms", latencyMS),
			"jitter":      fmt.Sprintf("%dms", latencyMS/10),
			"correlation": "25",
		},
		SafetyLevel: SafetyLevelMedium,
		BlastRadius: BlastRadius{
			Namespaces:     []string{targetNamespace},
			MaxPods:        3,
			TrafficPercent: 25,
			Duration:       5 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       2000,
			MinAvailability:    99.9,
			MinThroughput:      40,
			MaxErrorRate:       0.5,
			AutoRollbackEnable: true,
		},
		Duration:       5 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetPacketLossExperiment creates a packet loss experiment.
func (p *PredefinedExperiments) GetPacketLossExperiment(targetNamespace string, lossPercent float64) *Experiment {
	return &Experiment{
		Name:        "packet-loss-injection",
		Description: "Simulate packet loss in network communication",
		Type:        ExperimentTypeNetwork,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
			LabelSelector: map[string]string{
				"app.kubernetes.io/component": "network-function",
			},
		},
		Parameters: map[string]string{
			"loss":        fmt.Sprintf("%.2f", lossPercent),
			"correlation": "25",
		},
		SafetyLevel: SafetyLevelMedium,
		BlastRadius: BlastRadius{
			Namespaces:     []string{targetNamespace},
			MaxPods:        2,
			TrafficPercent: 20,
			Duration:       3 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       2500,
			MinAvailability:    99.5,
			MinThroughput:      35,
			MaxErrorRate:       1.0,
			AutoRollbackEnable: true,
		},
		Duration:       3 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetNetworkPartitionExperiment creates a network partition experiment.
func (p *PredefinedExperiments) GetNetworkPartitionExperiment(sourceNamespace, targetNamespace string) *Experiment {
	return &Experiment{
		Name:        "network-partition",
		Description: "Create network partition between components",
		Type:        ExperimentTypeNetwork,
		Target: ExperimentTarget{
			Namespace: sourceNamespace,
			LabelSelector: map[string]string{
				"app.kubernetes.io/component": "network-function",
			},
		},
		Parameters: map[string]string{
			"source_namespace": sourceNamespace,
			"target_namespace": targetNamespace,
			"direction":        "both",
			"protocol":         "tcp",
		},
		SafetyLevel: SafetyLevelHigh,
		BlastRadius: BlastRadius{
			Namespaces: []string{sourceNamespace, targetNamespace},
			MaxPods:    5,
			Duration:   2 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       3000,
			MinAvailability:    99.0,
			MinThroughput:      30,
			MaxErrorRate:       2.0,
			AutoRollbackEnable: true,
		},
		Duration:       2 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetPodChaosExperiment creates a pod failure experiment.
func (p *PredefinedExperiments) GetPodChaosExperiment(targetNamespace string, killMode string) *Experiment {
	return &Experiment{
		Name:        "pod-chaos",
		Description: "Randomly terminate pods to test resilience",
		Type:        ExperimentTypePod,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
			LabelSelector: map[string]string{
				"chaos": "enabled",
			},
		},
		Parameters: map[string]string{
			"mode":     killMode, // random, fixed, percentage
			"count":    "1",
			"interval": "30s",
			"grace":    "0",
		},
		SafetyLevel: SafetyLevelMedium,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			MaxPods:    3,
			Duration:   5 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       2000,
			MinAvailability:    99.5,
			MinThroughput:      40,
			MaxErrorRate:       0.5,
			AutoRollbackEnable: true,
		},
		Duration:       5 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetCPUStressExperiment creates a CPU stress experiment.
func (p *PredefinedExperiments) GetCPUStressExperiment(targetNamespace string, cpuPercent int) *Experiment {
	return &Experiment{
		Name:        "cpu-stress",
		Description: "Inject CPU stress to test performance under load",
		Type:        ExperimentTypeResource,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
			LabelSelector: map[string]string{
				"app.kubernetes.io/component": "compute-intensive",
			},
		},
		Parameters: map[string]string{
			"cpu_percent": fmt.Sprintf("%d", cpuPercent),
			"workers":     "2",
			"timeout":     "60s",
		},
		SafetyLevel: SafetyLevelMedium,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			MaxPods:    2,
			Duration:   3 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       3000,
			MinAvailability:    99.5,
			MinThroughput:      35,
			MaxErrorRate:       1.0,
			AutoRollbackEnable: true,
		},
		Duration:       3 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetMemoryStressExperiment creates a memory stress experiment.
func (p *PredefinedExperiments) GetMemoryStressExperiment(targetNamespace string, memoryMB int) *Experiment {
	return &Experiment{
		Name:        "memory-stress",
		Description: "Inject memory pressure to test OOM handling",
		Type:        ExperimentTypeResource,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
			LabelSelector: map[string]string{
				"app.kubernetes.io/component": "memory-intensive",
			},
		},
		Parameters: map[string]string{
			"memory_mb": fmt.Sprintf("%d", memoryMB),
			"workers":   "1",
			"timeout":   "60s",
			"vm_bytes":  fmt.Sprintf("%dM", memoryMB),
			"vm_hang":   "10",
		},
		SafetyLevel: SafetyLevelMedium,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			MaxPods:    2,
			Duration:   3 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       2500,
			MinAvailability:    99.5,
			MinThroughput:      35,
			MaxErrorRate:       1.0,
			AutoRollbackEnable: true,
		},
		Duration:       3 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetDiskIOStressExperiment creates a disk I/O stress experiment.
func (p *PredefinedExperiments) GetDiskIOStressExperiment(targetNamespace string, ioWorkers int) *Experiment {
	return &Experiment{
		Name:        "disk-io-stress",
		Description: "Stress disk I/O to test storage performance",
		Type:        ExperimentTypeResource,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
			LabelSelector: map[string]string{
				"app.kubernetes.io/component": "storage-intensive",
			},
		},
		Parameters: map[string]string{
			"io_workers": fmt.Sprintf("%d", ioWorkers),
			"io_ops":     "100",
			"io_size":    "1M",
			"io_type":    "mixed", // read, write, mixed
			"timeout":    "60s",
		},
		SafetyLevel: SafetyLevelMedium,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			MaxPods:    1,
			Duration:   3 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       4000,
			MinAvailability:    99.0,
			MinThroughput:      30,
			MaxErrorRate:       2.0,
			AutoRollbackEnable: true,
		},
		Duration:       3 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetDatabaseConnectionFailureExperiment creates a database connection failure experiment.
func (p *PredefinedExperiments) GetDatabaseConnectionFailureExperiment(targetNamespace string) *Experiment {
	return &Experiment{
		Name:        "database-connection-failure",
		Description: "Simulate database connection failures",
		Type:        ExperimentTypeDatabase,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
			LabelSelector: map[string]string{
				"app.kubernetes.io/component": "database-client",
			},
		},
		Parameters: map[string]string{
			"failure_type":  "connection_drop",
			"failure_rate":  "50",
			"database_type": "postgresql",
			"port":          "5432",
		},
		SafetyLevel: SafetyLevelHigh,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			MaxPods:    2,
			Duration:   2 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       3000,
			MinAvailability:    99.0,
			MinThroughput:      30,
			MaxErrorRate:       2.0,
			AutoRollbackEnable: true,
		},
		Duration:       2 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetDatabaseSlowQueryExperiment creates a slow database query experiment.
func (p *PredefinedExperiments) GetDatabaseSlowQueryExperiment(targetNamespace string, delayMS int) *Experiment {
	return &Experiment{
		Name:        "database-slow-query",
		Description: "Inject delays in database queries",
		Type:        ExperimentTypeDatabase,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
			LabelSelector: map[string]string{
				"app.kubernetes.io/component": "database",
			},
		},
		Parameters: map[string]string{
			"query_delay_ms":  fmt.Sprintf("%d", delayMS),
			"affected_tables": "all",
			"query_pattern":   "SELECT",
			"probability":     "0.5",
		},
		SafetyLevel: SafetyLevelMedium,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			MaxPods:    1,
			Duration:   3 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       5000,
			MinAvailability:    99.5,
			MinThroughput:      25,
			MaxErrorRate:       1.0,
			AutoRollbackEnable: true,
		},
		Duration:       3 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetLLMAPITimeoutExperiment creates an LLM API timeout experiment.
func (p *PredefinedExperiments) GetLLMAPITimeoutExperiment(targetNamespace string) *Experiment {
	return &Experiment{
		Name:        "llm-api-timeout",
		Description: "Simulate LLM API timeouts and delays",
		Type:        ExperimentTypeExternal,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
			LabelSelector: map[string]string{
				"app.kubernetes.io/name": "llm-processor",
			},
		},
		Parameters: map[string]string{
			"api_endpoint": "openai",
			"timeout_ms":   "5000",
			"failure_rate": "30",
			"failure_type": "timeout",
		},
		SafetyLevel: SafetyLevelHigh,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			MaxPods:    1,
			Duration:   2 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       8000,
			MinAvailability:    99.0,
			MinThroughput:      20,
			MaxErrorRate:       3.0,
			AutoRollbackEnable: true,
		},
		Duration:       2 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetRAGSystemFailureExperiment creates a RAG system failure experiment.
func (p *PredefinedExperiments) GetRAGSystemFailureExperiment(targetNamespace string) *Experiment {
	return &Experiment{
		Name:        "rag-system-failure",
		Description: "Simulate RAG vector database failures",
		Type:        ExperimentTypeExternal,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
			LabelSelector: map[string]string{
				"app.kubernetes.io/name": "rag-api",
			},
		},
		Parameters: map[string]string{
			"component":     "weaviate",
			"failure_type":  "connection_error",
			"failure_rate":  "40",
			"retry_enabled": "true",
		},
		SafetyLevel: SafetyLevelMedium,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			MaxPods:    2,
			Duration:   3 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       4000,
			MinAvailability:    99.0,
			MinThroughput:      30,
			MaxErrorRate:       2.0,
			AutoRollbackEnable: true,
		},
		Duration:       3 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetHighLoadExperiment creates a high load test experiment.
func (p *PredefinedExperiments) GetHighLoadExperiment(targetNamespace string, intentsPerMinute int) *Experiment {
	return &Experiment{
		Name:        "high-load-test",
		Description: "Test system under high intent processing load",
		Type:        ExperimentTypeLoad,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
		},
		Parameters: map[string]string{
			"intents_per_minute": fmt.Sprintf("%d", intentsPerMinute),
			"duration":           "300s",
			"ramp_up":            "30s",
			"intent_complexity":  "mixed",
		},
		SafetyLevel: SafetyLevelMedium,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			Duration:   5 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       3000,
			MinAvailability:    99.5,
			MinThroughput:      float64(intentsPerMinute * 80 / 100), // 80% of target
			MaxErrorRate:       1.0,
			AutoRollbackEnable: true,
		},
		Duration:       5 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetKubernetesAPIFailureExperiment creates a Kubernetes API failure experiment.
func (p *PredefinedExperiments) GetKubernetesAPIFailureExperiment(targetNamespace string) *Experiment {
	return &Experiment{
		Name:        "kubernetes-api-failure",
		Description: "Simulate Kubernetes API server issues",
		Type:        ExperimentTypeDependency,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
			LabelSelector: map[string]string{
				"app.kubernetes.io/component": "controller",
			},
		},
		Parameters: map[string]string{
			"api_operations": "get,list,watch",
			"failure_rate":   "20",
			"delay_ms":       "1000",
			"error_code":     "500",
		},
		SafetyLevel: SafetyLevelHigh,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			MaxPods:    2,
			Duration:   2 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       4000,
			MinAvailability:    99.0,
			MinThroughput:      25,
			MaxErrorRate:       2.0,
			AutoRollbackEnable: true,
		},
		Duration:       2 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetDNSFailureExperiment creates a DNS failure experiment.
func (p *PredefinedExperiments) GetDNSFailureExperiment(targetNamespace string) *Experiment {
	return &Experiment{
		Name:        "dns-failure",
		Description: "Simulate DNS resolution failures",
		Type:        ExperimentTypeDependency,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
		},
		Parameters: map[string]string{
			"domains":      "*.svc.cluster.local",
			"failure_type": "nxdomain",
			"failure_rate": "30",
			"cache_poison": "false",
		},
		SafetyLevel: SafetyLevelMedium,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			MaxPods:    5,
			Duration:   3 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       3000,
			MinAvailability:    99.0,
			MinThroughput:      30,
			MaxErrorRate:       2.0,
			AutoRollbackEnable: true,
		},
		Duration:       3 * time.Minute,
		RollbackOnFail: true,
	}
}

// Get5GCoreAMFFailureExperiment creates an AMF failure experiment.
func (p *PredefinedExperiments) Get5GCoreAMFFailureExperiment(targetNamespace string) *Experiment {
	return &Experiment{
		Name:        "5g-core-amf-failure",
		Description: "Simulate 5G Core AMF component failure",
		Type:        ExperimentTypePod,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
			LabelSelector: map[string]string{
				"app.kubernetes.io/name":      "amf",
				"app.kubernetes.io/component": "5g-core",
			},
		},
		Parameters: map[string]string{
			"failure_type":   "crash",
			"recovery_delay": "30s",
			"session_impact": "partial",
		},
		SafetyLevel: SafetyLevelHigh,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			MaxPods:    1,
			Duration:   2 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       2000,
			MinAvailability:    99.5,
			MinThroughput:      40,
			MaxErrorRate:       0.5,
			AutoRollbackEnable: true,
		},
		Duration:       2 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetORANRICFailureExperiment creates an O-RAN RIC failure experiment.
func (p *PredefinedExperiments) GetORANRICFailureExperiment(targetNamespace string) *Experiment {
	return &Experiment{
		Name:        "oran-ric-failure",
		Description: "Simulate O-RAN Near-RT RIC failure",
		Type:        ExperimentTypePod,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
			LabelSelector: map[string]string{
				"app.kubernetes.io/name":      "near-rt-ric",
				"app.kubernetes.io/component": "oran",
			},
		},
		Parameters: map[string]string{
			"failure_type":   "partial",
			"affected_xapps": "traffic-steering,qos-management",
			"e2_impact":      "degraded",
		},
		SafetyLevel: SafetyLevelHigh,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			MaxPods:    2,
			Duration:   3 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       3000,
			MinAvailability:    99.0,
			MinThroughput:      35,
			MaxErrorRate:       1.0,
			AutoRollbackEnable: true,
		},
		Duration:       3 * time.Minute,
		RollbackOnFail: true,
	}
}

// GetCompositeFailureExperiment creates a composite failure experiment.
func (p *PredefinedExperiments) GetCompositeFailureExperiment(targetNamespace string) *Experiment {
	return &Experiment{
		Name:        "composite-failure-scenario",
		Description: "Multiple simultaneous failures to test system resilience",
		Type:        ExperimentTypeComposite,
		Target: ExperimentTarget{
			Namespace: targetNamespace,
		},
		Parameters: map[string]string{
			"scenarios":     "network-latency,pod-failure,cpu-stress",
			"stagger_start": "true",
			"stagger_delay": "30s",
			"correlation":   "0.5",
		},
		SafetyLevel: SafetyLevelHigh,
		BlastRadius: BlastRadius{
			Namespaces: []string{targetNamespace},
			MaxPods:    5,
			Duration:   5 * time.Minute,
		},
		SLAThresholds: SLAThresholds{
			MaxLatencyMS:       4000,
			MinAvailability:    99.0,
			MinThroughput:      25,
			MaxErrorRate:       3.0,
			AutoRollbackEnable: true,
		},
		Duration:       5 * time.Minute,
		RollbackOnFail: true,
	}
}

// ExperimentSuite represents a collection of related experiments.
type ExperimentSuite struct {
	Name         string
	Description  string
	Experiments  []*Experiment
	RunSequence  RunSequence
	Dependencies []string
}

// RunSequence defines how experiments in a suite should run.
type RunSequence string

const (
	// RunSequenceSerial holds runsequenceserial value.
	RunSequenceSerial RunSequence = "serial"
	// RunSequenceParallel holds runsequenceparallel value.
	RunSequenceParallel RunSequence = "parallel"
	// RunSequenceStaggered holds runsequencestaggered value.
	RunSequenceStaggered RunSequence = "staggered"
)

// GetNetworkResilienceSuite returns a suite of network resilience experiments.
func (p *PredefinedExperiments) GetNetworkResilienceSuite(targetNamespace string) *ExperimentSuite {
	return &ExperimentSuite{
		Name:        "network-resilience-suite",
		Description: "Comprehensive network resilience testing",
		Experiments: []*Experiment{
			p.GetNetworkLatencyExperiment(targetNamespace, 100),
			p.GetPacketLossExperiment(targetNamespace, 5.0),
			p.GetNetworkPartitionExperiment(targetNamespace, "default"),
		},
		RunSequence: RunSequenceSerial,
	}
}

// GetResourceExhaustionSuite returns a suite of resource exhaustion experiments.
func (p *PredefinedExperiments) GetResourceExhaustionSuite(targetNamespace string) *ExperimentSuite {
	return &ExperimentSuite{
		Name:        "resource-exhaustion-suite",
		Description: "Test system behavior under resource constraints",
		Experiments: []*Experiment{
			p.GetCPUStressExperiment(targetNamespace, 80),
			p.GetMemoryStressExperiment(targetNamespace, 512),
			p.GetDiskIOStressExperiment(targetNamespace, 4),
		},
		RunSequence: RunSequenceStaggered,
	}
}

// GetDependencyFailureSuite returns a suite of dependency failure experiments.
func (p *PredefinedExperiments) GetDependencyFailureSuite(targetNamespace string) *ExperimentSuite {
	return &ExperimentSuite{
		Name:        "dependency-failure-suite",
		Description: "Test resilience to external dependency failures",
		Experiments: []*Experiment{
			p.GetDatabaseConnectionFailureExperiment(targetNamespace),
			p.GetLLMAPITimeoutExperiment(targetNamespace),
			p.GetRAGSystemFailureExperiment(targetNamespace),
			p.GetKubernetesAPIFailureExperiment(targetNamespace),
			p.GetDNSFailureExperiment(targetNamespace),
		},
		RunSequence: RunSequenceSerial,
	}
}

// GetTelecomSpecificSuite returns a suite of telecom-specific experiments.
func (p *PredefinedExperiments) GetTelecomSpecificSuite(targetNamespace string) *ExperimentSuite {
	return &ExperimentSuite{
		Name:        "telecom-specific-suite",
		Description: "Test telecommunications-specific components",
		Experiments: []*Experiment{
			p.Get5GCoreAMFFailureExperiment(targetNamespace),
			p.GetORANRICFailureExperiment(targetNamespace),
		},
		RunSequence: RunSequenceSerial,
	}
}

// GetSLAValidationSuite returns experiments specifically for SLA validation.
func (p *PredefinedExperiments) GetSLAValidationSuite(targetNamespace string) *ExperimentSuite {
	return &ExperimentSuite{
		Name:        "sla-validation-suite",
		Description: "Validate system maintains SLA under various conditions",
		Experiments: []*Experiment{
			// Test 99.95% availability.
			p.GetPodChaosExperiment(targetNamespace, "random"),
			// Test sub-2-second latency.
			p.GetNetworkLatencyExperiment(targetNamespace, 500),
			// Test 45 intents/minute throughput.
			p.GetHighLoadExperiment(targetNamespace, 60),
			// Combined stress test.
			p.GetCompositeFailureExperiment(targetNamespace),
		},
		RunSequence: RunSequenceSerial,
	}
}

// RunExperimentSuite executes a suite of experiments.
func (p *PredefinedExperiments) RunExperimentSuite(ctx context.Context, suite *ExperimentSuite) ([]*ExperimentResult, error) {
	p.logger.Info("Starting experiment suite",
		zap.String("suite", suite.Name),
		zap.Int("experiments", len(suite.Experiments)))

	_ = make([]*ExperimentResult, 0, len(suite.Experiments))

	switch suite.RunSequence {
	case RunSequenceSerial:
		return p.runSerialExperiments(ctx, suite.Experiments)
	case RunSequenceParallel:
		return p.runParallelExperiments(ctx, suite.Experiments)
	case RunSequenceStaggered:
		return p.runStaggeredExperiments(ctx, suite.Experiments)
	default:
		return nil, fmt.Errorf("unknown run sequence: %s", suite.RunSequence)
	}
}

// runSerialExperiments runs experiments one after another.
func (p *PredefinedExperiments) runSerialExperiments(ctx context.Context, experiments []*Experiment) ([]*ExperimentResult, error) {
	results := make([]*ExperimentResult, 0, len(experiments))

	for _, exp := range experiments {
		result, err := p.engine.RunExperiment(ctx, exp)
		if err != nil {
			p.logger.Error("Experiment failed",
				zap.String("experiment", exp.Name),
				zap.Error(err))
			// Continue with other experiments even if one fails.
		}
		results = append(results, result)

		// Add delay between experiments.
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		case <-time.After(30 * time.Second):
			// Recovery time between experiments.
		}
	}

	return results, nil
}

// runParallelExperiments runs all experiments simultaneously.
func (p *PredefinedExperiments) runParallelExperiments(ctx context.Context, experiments []*Experiment) ([]*ExperimentResult, error) {
	results := make([]*ExperimentResult, len(experiments))
	errChan := make(chan error, len(experiments))

	for i, exp := range experiments {
		go func(index int, experiment *Experiment) {
			result, err := p.engine.RunExperiment(ctx, experiment)
			results[index] = result
			if err != nil {
				errChan <- err
			} else {
				errChan <- nil
			}
		}(i, exp)
	}

	// Wait for all experiments to complete.
	var errors []error
	for i := 0; i < len(experiments); i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return results, fmt.Errorf("some experiments failed: %v", errors)
	}

	return results, nil
}

// runStaggeredExperiments runs experiments with a staggered start.
func (p *PredefinedExperiments) runStaggeredExperiments(ctx context.Context, experiments []*Experiment) ([]*ExperimentResult, error) {
	results := make([]*ExperimentResult, len(experiments))
	staggerDelay := 30 * time.Second

	for i, exp := range experiments {
		go func(index int, experiment *Experiment, delay time.Duration) {
			// Wait for stagger delay.
			select {
			case <-ctx.Done():
				return
			case <-time.After(delay):
			}

			result, err := p.engine.RunExperiment(ctx, experiment)
			results[index] = result
			if err != nil {
				p.logger.Error("Staggered experiment failed",
					zap.String("experiment", experiment.Name),
					zap.Error(err))
			}
		}(i, exp, time.Duration(i)*staggerDelay)
	}

	// Wait for all experiments to complete.
	maxDuration := time.Duration(len(experiments))*staggerDelay + 10*time.Minute
	select {
	case <-ctx.Done():
		return results, ctx.Err()
	case <-time.After(maxDuration):
		return results, nil
	}
}

// GameDayScenario represents a comprehensive failure simulation.
type GameDayScenario struct {
	Name        string
	Description string
	Severity    string // minor, major, critical
	Suites      []*ExperimentSuite
	Duration    time.Duration
	Objectives  []string
}

// GetGameDayScenario returns a predefined game day scenario.
func (p *PredefinedExperiments) GetGameDayScenario(scenarioType string, targetNamespace string) *GameDayScenario {
	switch scenarioType {
	case "datacenter-failure":
		return &GameDayScenario{
			Name:        "Datacenter Failure Simulation",
			Description: "Simulate partial datacenter failure",
			Severity:    "critical",
			Suites: []*ExperimentSuite{
				p.GetNetworkResilienceSuite(targetNamespace),
				p.GetResourceExhaustionSuite(targetNamespace),
				p.GetDependencyFailureSuite(targetNamespace),
			},
			Duration: 2 * time.Hour,
			Objectives: []string{
				"Validate multi-region failover",
				"Test data consistency during failures",
				"Verify SLA maintenance during major incident",
				"Assess incident response procedures",
			},
		}
	case "peak-load":
		return &GameDayScenario{
			Name:        "Peak Load Event",
			Description: "Simulate peak traffic conditions with failures",
			Severity:    "major",
			Suites: []*ExperimentSuite{
				p.GetSLAValidationSuite(targetNamespace),
			},
			Duration: 1 * time.Hour,
			Objectives: []string{
				"Validate auto-scaling under load",
				"Test circuit breaker effectiveness",
				"Verify graceful degradation",
				"Measure actual vs expected capacity",
			},
		}
	case "dependency-cascade":
		return &GameDayScenario{
			Name:        "Dependency Cascade Failure",
			Description: "Test cascading dependency failures",
			Severity:    "major",
			Suites: []*ExperimentSuite{
				p.GetDependencyFailureSuite(targetNamespace),
			},
			Duration: 90 * time.Minute,
			Objectives: []string{
				"Test isolation of failure domains",
				"Validate fallback mechanisms",
				"Assess recovery procedures",
				"Verify monitoring and alerting",
			},
		}
	default:
		return &GameDayScenario{
			Name:        "Basic Resilience Test",
			Description: "Basic system resilience validation",
			Severity:    "minor",
			Suites: []*ExperimentSuite{
				p.GetNetworkResilienceSuite(targetNamespace),
			},
			Duration: 30 * time.Minute,
			Objectives: []string{
				"Validate basic resilience mechanisms",
				"Test monitoring and alerting",
				"Verify recovery procedures",
			},
		}
	}
}
