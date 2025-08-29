// Package loadtesting provides telecom-specific load generators.


package loadtesting



import (

	"context"

	"fmt"

	"math"

	"math/rand"

	"strings"

	"sync"

	"sync/atomic"

	"time"



	"go.uber.org/zap"

	"golang.org/x/time/rate"

)



// RealisticTelecomGenerator simulates realistic telecommunications workloads.

type RealisticTelecomGenerator struct {

	id              int

	config          *LoadTestConfig

	logger          *zap.Logger

	limiter         *rate.Limiter

	metrics         *GeneratorMetrics

	workloadGen     *TelecomWorkloadGenerator

	intentGenerator *IntentGenerator

	stopChan        chan struct{}

	wg              sync.WaitGroup

	mu              sync.RWMutex



	// Metrics tracking.

	requestsSent      atomic.Int64

	requestsSucceeded atomic.Int64

	requestsFailed    atomic.Int64

	totalLatency      atomic.Int64

	latencyCount      atomic.Int64

}



// TelecomWorkloadGenerator generates telecom-specific workloads.

type TelecomWorkloadGenerator struct {

	config          *LoadTestConfig

	workloadWeights []WorkloadWeight

	complexityDist  []ComplexityWeight

	random          *rand.Rand

	mu              sync.Mutex

}



// WorkloadWeight represents weighted workload selection.

type WorkloadWeight struct {

	Type       string

	Weight     float64

	Cumulative float64

}



// ComplexityWeight represents weighted complexity selection.

type ComplexityWeight struct {

	Level      string

	Weight     float64

	Cumulative float64

}



// IntentGenerator creates realistic network intents.

type IntentGenerator struct {

	templates map[string][]IntentTemplate

	random    *rand.Rand

	mu        sync.Mutex

}



// IntentTemplate defines intent structure.

type IntentTemplate struct {

	Type         string

	Complexity   string

	Template     string

	Parameters   map[string]ParameterSpec

	Dependencies []string

	Validation   ValidationSpec

}



// ParameterSpec defines parameter generation rules.

type ParameterSpec struct {

	Type      string        // string, int, float, bool, enum

	Range     []interface{} // min/max for numbers, options for enum

	Pattern   string        // regex pattern for strings

	Generator string        // custom generator function name

}



// ValidationSpec defines expected validation behavior.

type ValidationSpec struct {

	ExpectedLatency   time.Duration

	ExpectedResources ResourceRequirements

	SuccessRate       float64

}



// ResourceRequirements defines resource needs.

type ResourceRequirements struct {

	CPU         float64 // cores

	MemoryGB    float64

	NetworkMbps float64

	StorageGB   float64

}



// NetworkIntent represents a generated intent.

type NetworkIntent struct {

	ID         string                 `json:"id"`

	Type       string                 `json:"type"`

	Complexity string                 `json:"complexity"`

	Intent     string                 `json:"intent"`

	Parameters map[string]interface{} `json:"parameters"`

	Metadata   IntentMetadata         `json:"metadata"`

	Timestamp  time.Time              `json:"timestamp"`

}



// IntentMetadata contains intent metadata.

type IntentMetadata struct {

	Source       string   `json:"source"`

	Priority     int      `json:"priority"`

	Tags         []string `json:"tags"`

	Dependencies []string `json:"dependencies"`

	Region       string   `json:"region"`

	Tenant       string   `json:"tenant"`

}



// NewRealisticTelecomGenerator creates a new realistic telecom load generator.

func NewRealisticTelecomGenerator(id int, config *LoadTestConfig, logger *zap.Logger) (*RealisticTelecomGenerator, error) {

	if config == nil {

		return nil, fmt.Errorf("config is required")

	}



	if logger == nil {

		logger = zap.NewNop()

	}



	// Calculate rate limit based on target throughput.

	rateLimit := config.TargetThroughput / 60.0 // Convert per minute to per second

	if rateLimit <= 0 {

		rateLimit = 1.0 // Default to 1 intent per second

	}



	gen := &RealisticTelecomGenerator{

		id:       id,

		config:   config,

		logger:   logger.With(zap.Int("generatorId", id)),

		limiter:  rate.NewLimiter(rate.Limit(rateLimit), int(rateLimit*2)),

		metrics:  &GeneratorMetrics{},

		stopChan: make(chan struct{}),

	}



	// Initialize workload generator.

	gen.workloadGen = newTelecomWorkloadGenerator(config)



	// Initialize intent generator.

	gen.intentGenerator = newIntentGenerator()



	return gen, nil

}



// Initialize prepares the generator.

func (g *RealisticTelecomGenerator) Initialize(config *LoadTestConfig) error {

	g.mu.Lock()

	defer g.mu.Unlock()



	g.config = config



	// Recalculate rate limit.

	rateLimit := config.TargetThroughput / 60.0

	g.limiter = rate.NewLimiter(rate.Limit(rateLimit), int(rateLimit*2))



	g.logger.Info("Generator initialized",

		zap.Float64("rateLimit", rateLimit),

		zap.String("pattern", string(config.LoadPattern)))



	return nil

}



// Generate starts generating load.

func (g *RealisticTelecomGenerator) Generate(ctx context.Context) error {

	g.logger.Info("Starting load generation")



	g.wg.Add(1)

	defer g.wg.Done()



	// Generate workload based on pattern.

	switch g.config.LoadPattern {

	case LoadPatternConstant:

		return g.generateConstantLoad(ctx)

	case LoadPatternBurst:

		return g.generateBurstLoad(ctx)

	case LoadPatternWave:

		return g.generateWaveLoad(ctx)

	case LoadPatternRealistic:

		return g.generateRealisticLoad(ctx)

	default:

		return g.generateConstantLoad(ctx)

	}

}



// GetMetrics returns current metrics.

func (g *RealisticTelecomGenerator) GetMetrics() GeneratorMetrics {

	sent := g.requestsSent.Load()

	succeeded := g.requestsSucceeded.Load()

	failed := g.requestsFailed.Load()



	metrics := GeneratorMetrics{

		RequestsSent:      sent,

		RequestsSucceeded: succeeded,

		RequestsFailed:    failed,

	}



	// Calculate average latency.

	latencySum := g.totalLatency.Load()

	latencyCount := g.latencyCount.Load()

	if latencyCount > 0 {

		metrics.AverageLatency = time.Duration(latencySum / latencyCount)

	}



	// Calculate error rate.

	if sent > 0 {

		metrics.ErrorRate = float64(failed) / float64(sent)

	}



	// Calculate current rate (requests per second).

	// This is a simplified calculation - in production, use a sliding window.

	metrics.CurrentRate = float64(sent) / time.Since(time.Now()).Seconds()



	return metrics

}



// Adjust modifies generation parameters.

func (g *RealisticTelecomGenerator) Adjust(params map[string]interface{}) error {

	g.mu.Lock()

	defer g.mu.Unlock()



	if multiplier, ok := params["rate_multiplier"].(float64); ok {

		newRate := (g.config.TargetThroughput / 60.0) * multiplier

		g.limiter.SetLimit(rate.Limit(newRate))

		g.logger.Debug("Adjusted rate limit",

			zap.Float64("multiplier", multiplier),

			zap.Float64("newRate", newRate))

	}



	return nil

}



// Stop gracefully stops generation.

func (g *RealisticTelecomGenerator) Stop() error {

	g.logger.Info("Stopping generator")

	close(g.stopChan)

	g.wg.Wait()

	return nil

}



// Private methods for different load patterns.



func (g *RealisticTelecomGenerator) generateConstantLoad(ctx context.Context) error {

	for {

		select {

		case <-ctx.Done():

			return ctx.Err()

		case <-g.stopChan:

			return nil

		default:

			// Wait for rate limiter.

			if err := g.limiter.Wait(ctx); err != nil {

				return err

			}



			// Generate and send intent.

			g.sendIntent(ctx)

		}

	}

}



func (g *RealisticTelecomGenerator) generateBurstLoad(ctx context.Context) error {

	burstSize := 10

	burstInterval := 5 * time.Second

	ticker := time.NewTicker(burstInterval)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return ctx.Err()

		case <-g.stopChan:

			return nil

		case <-ticker.C:

			// Send burst of requests.

			for range burstSize {

				g.sendIntent(ctx)

			}

		}

	}

}



func (g *RealisticTelecomGenerator) generateWaveLoad(ctx context.Context) error {

	ticker := time.NewTicker(100 * time.Millisecond)

	defer ticker.Stop()



	var t float64

	for {

		select {

		case <-ctx.Done():

			return ctx.Err()

		case <-g.stopChan:

			return nil

		case <-ticker.C:

			// Generate wave pattern (sine wave).

			t += 0.1

			amplitude := g.config.TargetThroughput / 60.0

			waveRate := amplitude * (1 + math.Sin(t))



			// Adjust rate limiter.

			g.limiter.SetLimit(rate.Limit(waveRate))



			// Send intent if allowed.

			if g.limiter.Allow() {

				g.sendIntent(ctx)

			}

		}

	}

}



func (g *RealisticTelecomGenerator) generateRealisticLoad(ctx context.Context) error {

	// Simulate realistic telecom traffic patterns.

	// - Morning ramp-up (6am-9am).

	// - Steady daytime load (9am-5pm).

	// - Evening peak (5pm-10pm).

	// - Night time low (10pm-6am).



	ticker := time.NewTicker(1 * time.Second)

	defer ticker.Stop()



	for {

		select {

		case <-ctx.Done():

			return ctx.Err()

		case <-g.stopChan:

			return nil

		case <-ticker.C:

			// Get current hour (simplified - in production use proper timezone).

			hour := time.Now().Hour()



			// Determine load multiplier based on time.

			var multiplier float64

			switch {

			case hour >= 6 && hour < 9:

				// Morning ramp-up.

				multiplier = 0.5 + 0.5*float64(hour-6)/3.0

			case hour >= 9 && hour < 17:

				// Daytime steady.

				multiplier = 1.0

			case hour >= 17 && hour < 22:

				// Evening peak.

				multiplier = 1.2

			default:

				// Night time low.

				multiplier = 0.3

			}



			// Add some randomness.

			multiplier *= (0.9 + rand.Float64()*0.2)



			// Adjust rate.

			newRate := (g.config.TargetThroughput / 60.0) * multiplier

			g.limiter.SetLimit(rate.Limit(newRate))



			// Send intent if allowed.

			if g.limiter.Allow() {

				g.sendIntent(ctx)

			}

		}

	}

}



func (g *RealisticTelecomGenerator) sendIntent(ctx context.Context) {

	start := time.Now()



	// Generate intent.

	intent := g.generateIntent()



	// Simulate sending intent (in real implementation, send to actual system).

	success, latency := g.simulateSendIntent(ctx, intent)



	// Update metrics.

	g.requestsSent.Add(1)

	if success {

		g.requestsSucceeded.Add(1)

	} else {

		g.requestsFailed.Add(1)

	}



	g.totalLatency.Add(int64(latency))

	g.latencyCount.Add(1)



	g.logger.Debug("Intent sent",

		zap.String("intentId", intent.ID),

		zap.String("type", intent.Type),

		zap.String("complexity", intent.Complexity),

		zap.Bool("success", success),

		zap.Duration("latency", latency),

		zap.Duration("elapsed", time.Since(start)))

}



func (g *RealisticTelecomGenerator) generateIntent() *NetworkIntent {

	// Select workload type.

	workloadType := g.workloadGen.SelectWorkloadType()



	// Select complexity.

	complexity := g.workloadGen.SelectComplexity()



	// Generate intent based on type and complexity.

	intent := g.intentGenerator.Generate(workloadType, complexity)



	// Add metadata.

	intent.Metadata = IntentMetadata{

		Source:   fmt.Sprintf("generator-%d", g.id),

		Priority: rand.Intn(5) + 1,

		Tags:     []string{workloadType, complexity},

		Region:   g.selectRegion(),

		Tenant:   fmt.Sprintf("tenant-%d", rand.Intn(10)+1),

	}



	return intent

}



func (g *RealisticTelecomGenerator) selectRegion() string {

	if len(g.config.Regions) > 0 {

		return g.config.Regions[rand.Intn(len(g.config.Regions))]

	}

	return "default"

}



func (g *RealisticTelecomGenerator) simulateSendIntent(ctx context.Context, intent *NetworkIntent) (bool, time.Duration) {

	// Simulate network latency and processing time.

	baseLatency := 50 * time.Millisecond



	// Add complexity-based latency.

	switch intent.Complexity {

	case "simple":

		baseLatency += time.Duration(rand.Intn(200)) * time.Millisecond

	case "moderate":

		baseLatency += time.Duration(rand.Intn(800)) * time.Millisecond

	case "complex":

		baseLatency += time.Duration(rand.Intn(1500)) * time.Millisecond

	}



	// Simulate processing.

	select {

	case <-ctx.Done():

		return false, baseLatency

	case <-time.After(baseLatency):

		// Simulate success rate based on complexity.

		successRate := 0.95

		switch intent.Complexity {

		case "moderate":

			successRate = 0.90

		case "complex":

			successRate = 0.85

		}



		success := rand.Float64() < successRate

		return success, baseLatency

	}

}



// TelecomWorkloadGenerator implementation.



func newTelecomWorkloadGenerator(config *LoadTestConfig) *TelecomWorkloadGenerator {

	gen := &TelecomWorkloadGenerator{

		config: config,

		random: rand.New(rand.NewSource(time.Now().UnixNano())),

	}



	// Build weighted workload selection.

	gen.buildWorkloadWeights()



	// Build complexity distribution.

	gen.buildComplexityWeights()



	return gen

}



func (g *TelecomWorkloadGenerator) buildWorkloadWeights() {

	mix := g.config.WorkloadMix

	cumulative := 0.0



	g.workloadWeights = []WorkloadWeight{

		{Type: "core_network", Weight: mix.CoreNetworkDeployments, Cumulative: cumulative + mix.CoreNetworkDeployments},

		{Type: "oran_config", Weight: mix.ORANConfigurations, Cumulative: cumulative + mix.CoreNetworkDeployments + mix.ORANConfigurations},

		{Type: "network_slice", Weight: mix.NetworkSlicing, Cumulative: cumulative + mix.CoreNetworkDeployments + mix.ORANConfigurations + mix.NetworkSlicing},

		{Type: "multi_vendor", Weight: mix.MultiVendorOrchestration, Cumulative: cumulative + mix.CoreNetworkDeployments + mix.ORANConfigurations + mix.NetworkSlicing + mix.MultiVendorOrchestration},

		{Type: "policy_mgmt", Weight: mix.PolicyManagement, Cumulative: cumulative + mix.CoreNetworkDeployments + mix.ORANConfigurations + mix.NetworkSlicing + mix.MultiVendorOrchestration + mix.PolicyManagement},

		{Type: "perf_opt", Weight: mix.PerformanceOptimization, Cumulative: cumulative + mix.CoreNetworkDeployments + mix.ORANConfigurations + mix.NetworkSlicing + mix.MultiVendorOrchestration + mix.PolicyManagement + mix.PerformanceOptimization},

		{Type: "fault_mgmt", Weight: mix.FaultManagement, Cumulative: cumulative + mix.CoreNetworkDeployments + mix.ORANConfigurations + mix.NetworkSlicing + mix.MultiVendorOrchestration + mix.PolicyManagement + mix.PerformanceOptimization + mix.FaultManagement},

		{Type: "scaling", Weight: mix.ScalingOperations, Cumulative: 1.0},

	}

}



func (g *TelecomWorkloadGenerator) buildComplexityWeights() {

	profile := g.config.IntentComplexity



	g.complexityDist = []ComplexityWeight{

		{Level: "simple", Weight: profile.Simple, Cumulative: profile.Simple},

		{Level: "moderate", Weight: profile.Moderate, Cumulative: profile.Simple + profile.Moderate},

		{Level: "complex", Weight: profile.Complex, Cumulative: 1.0},

	}

}



// SelectWorkloadType performs selectworkloadtype operation.

func (g *TelecomWorkloadGenerator) SelectWorkloadType() string {

	g.mu.Lock()

	defer g.mu.Unlock()



	r := g.random.Float64()



	for _, w := range g.workloadWeights {

		if r <= w.Cumulative {

			return w.Type

		}

	}



	return "core_network" // Default

}



// SelectComplexity performs selectcomplexity operation.

func (g *TelecomWorkloadGenerator) SelectComplexity() string {

	g.mu.Lock()

	defer g.mu.Unlock()



	r := g.random.Float64()



	for _, c := range g.complexityDist {

		if r <= c.Cumulative {

			return c.Level

		}

	}



	return "simple" // Default

}



// IntentGenerator implementation.



func newIntentGenerator() *IntentGenerator {

	gen := &IntentGenerator{

		templates: make(map[string][]IntentTemplate),

		random:    rand.New(rand.NewSource(time.Now().UnixNano())),

	}



	// Initialize templates.

	gen.initializeTemplates()



	return gen

}



func (g *IntentGenerator) initializeTemplates() {

	// 5G Core Network templates.

	g.templates["core_network"] = []IntentTemplate{

		{

			Type:       "core_network",

			Complexity: "simple",

			Template:   "Deploy AMF instance {{.name}} in region {{.region}}",

			Parameters: map[string]ParameterSpec{

				"name":   {Type: "string", Pattern: "amf-[a-z0-9]{8}"},

				"region": {Type: "enum", Range: []interface{}{"us-east", "us-west", "eu-central", "ap-south"}},

			},

			Validation: ValidationSpec{

				ExpectedLatency: 500 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 2.0, MemoryGB: 4.0, NetworkMbps: 100,

				},

				SuccessRate: 0.95,

			},

		},

		{

			Type:       "core_network",

			Complexity: "moderate",

			Template:   "Deploy SMF {{.name}} with {{.sessions}} max sessions and connect to UPF {{.upf}}",

			Parameters: map[string]ParameterSpec{

				"name":     {Type: "string", Pattern: "smf-[a-z0-9]{8}"},

				"sessions": {Type: "int", Range: []interface{}{1000, 100000}},

				"upf":      {Type: "string", Pattern: "upf-[a-z0-9]{8}"},

			},

			Dependencies: []string{"upf"},

			Validation: ValidationSpec{

				ExpectedLatency: 1000 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 4.0, MemoryGB: 8.0, NetworkMbps: 500,

				},

				SuccessRate: 0.90,

			},

		},

		{

			Type:       "core_network",

			Complexity: "complex",

			Template:   "Deploy complete 5G Core with AMF, SMF, UPF, and configure network slice {{.slice}} with QoS {{.qos}}",

			Parameters: map[string]ParameterSpec{

				"slice": {Type: "string", Pattern: "slice-[a-z0-9]{8}"},

				"qos":   {Type: "enum", Range: []interface{}{"guaranteed", "best-effort", "premium"}},

			},

			Validation: ValidationSpec{

				ExpectedLatency: 2000 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 16.0, MemoryGB: 32.0, NetworkMbps: 1000,

				},

				SuccessRate: 0.85,

			},

		},

	}



	// O-RAN configurations templates.

	g.templates["oran_config"] = []IntentTemplate{

		{

			Type:       "oran_config",

			Complexity: "simple",

			Template:   "Deploy Near-RT RIC {{.name}} with {{.xapps}} xApps",

			Parameters: map[string]ParameterSpec{

				"name":  {Type: "string", Pattern: "ric-[a-z0-9]{8}"},

				"xapps": {Type: "int", Range: []interface{}{1, 10}},

			},

			Validation: ValidationSpec{

				ExpectedLatency: 600 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 4.0, MemoryGB: 8.0, NetworkMbps: 200,

				},

				SuccessRate: 0.93,

			},

		},

		{

			Type:       "oran_config",

			Complexity: "moderate",

			Template:   "Configure O-DU {{.odu}} with {{.cells}} cells and connect to O-CU {{.ocu}}",

			Parameters: map[string]ParameterSpec{

				"odu":   {Type: "string", Pattern: "odu-[a-z0-9]{8}"},

				"cells": {Type: "int", Range: []interface{}{1, 32}},

				"ocu":   {Type: "string", Pattern: "ocu-[a-z0-9]{8}"},

			},

			Dependencies: []string{"ocu"},

			Validation: ValidationSpec{

				ExpectedLatency: 1200 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 8.0, MemoryGB: 16.0, NetworkMbps: 500,

				},

				SuccessRate: 0.88,

			},

		},

		{

			Type:       "oran_config",

			Complexity: "complex",

			Template:   "Deploy complete O-RAN setup with RIC, {{.dus}} DUs, {{.cus}} CUs, and A1 policy {{.policy}}",

			Parameters: map[string]ParameterSpec{

				"dus":    {Type: "int", Range: []interface{}{2, 8}},

				"cus":    {Type: "int", Range: []interface{}{1, 4}},

				"policy": {Type: "enum", Range: []interface{}{"traffic-steering", "qos-optimization", "energy-saving"}},

			},

			Validation: ValidationSpec{

				ExpectedLatency: 3000 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 32.0, MemoryGB: 64.0, NetworkMbps: 2000,

				},

				SuccessRate: 0.80,

			},

		},

	}



	// Network slicing templates.

	g.templates["network_slice"] = []IntentTemplate{

		{

			Type:       "network_slice",

			Complexity: "simple",

			Template:   "Create network slice {{.name}} for {{.service}} service",

			Parameters: map[string]ParameterSpec{

				"name":    {Type: "string", Pattern: "slice-[a-z0-9]{8}"},

				"service": {Type: "enum", Range: []interface{}{"eMBB", "URLLC", "mMTC"}},

			},

			Validation: ValidationSpec{

				ExpectedLatency: 800 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 2.0, MemoryGB: 4.0, NetworkMbps: 100,

				},

				SuccessRate: 0.92,

			},

		},

		{

			Type:       "network_slice",

			Complexity: "moderate",

			Template:   "Configure slice {{.name}} with bandwidth {{.bandwidth}}Mbps and latency SLA {{.latency}}ms",

			Parameters: map[string]ParameterSpec{

				"name":      {Type: "string", Pattern: "slice-[a-z0-9]{8}"},

				"bandwidth": {Type: "int", Range: []interface{}{100, 10000}},

				"latency":   {Type: "int", Range: []interface{}{1, 100}},

			},

			Validation: ValidationSpec{

				ExpectedLatency: 1500 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 4.0, MemoryGB: 8.0, NetworkMbps: 500,

				},

				SuccessRate: 0.87,

			},

		},

		{

			Type:       "network_slice",

			Complexity: "complex",

			Template:   "Deploy multi-domain slice {{.name}} across {{.domains}} domains with E2E orchestration",

			Parameters: map[string]ParameterSpec{

				"name":    {Type: "string", Pattern: "slice-[a-z0-9]{8}"},

				"domains": {Type: "int", Range: []interface{}{2, 5}},

			},

			Validation: ValidationSpec{

				ExpectedLatency: 4000 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 8.0, MemoryGB: 16.0, NetworkMbps: 1000,

				},

				SuccessRate: 0.75,

			},

		},

	}



	// Policy management templates.

	g.templates["policy_mgmt"] = []IntentTemplate{

		{

			Type:       "policy_mgmt",

			Complexity: "simple",

			Template:   "Apply QoS policy {{.policy}} to slice {{.slice}}",

			Parameters: map[string]ParameterSpec{

				"policy": {Type: "string", Pattern: "policy-[a-z0-9]{8}"},

				"slice":  {Type: "string", Pattern: "slice-[a-z0-9]{8}"},

			},

			Validation: ValidationSpec{

				ExpectedLatency: 200 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 0.5, MemoryGB: 1.0, NetworkMbps: 10,

				},

				SuccessRate: 0.98,

			},

		},

		{

			Type:       "policy_mgmt",

			Complexity: "moderate",

			Template:   "Configure A1 policy for traffic steering with threshold {{.threshold}}% and target {{.target}}",

			Parameters: map[string]ParameterSpec{

				"threshold": {Type: "int", Range: []interface{}{50, 90}},

				"target":    {Type: "string", Pattern: "cell-[a-z0-9]{8}"},

			},

			Validation: ValidationSpec{

				ExpectedLatency: 500 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 1.0, MemoryGB: 2.0, NetworkMbps: 50,

				},

				SuccessRate: 0.95,

			},

		},

	}



	// Performance optimization templates.

	g.templates["perf_opt"] = []IntentTemplate{

		{

			Type:       "perf_opt",

			Complexity: "simple",

			Template:   "Optimize {{.metric}} for cell {{.cell}}",

			Parameters: map[string]ParameterSpec{

				"metric": {Type: "enum", Range: []interface{}{"throughput", "latency", "coverage"}},

				"cell":   {Type: "string", Pattern: "cell-[a-z0-9]{8}"},

			},

			Validation: ValidationSpec{

				ExpectedLatency: 300 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 1.0, MemoryGB: 2.0, NetworkMbps: 20,

				},

				SuccessRate: 0.94,

			},

		},

		{

			Type:       "perf_opt",

			Complexity: "moderate",

			Template:   "Enable ML-based optimization for {{.scope}} with target KPI improvement {{.target}}%",

			Parameters: map[string]ParameterSpec{

				"scope":  {Type: "enum", Range: []interface{}{"cell", "cluster", "network"}},

				"target": {Type: "int", Range: []interface{}{5, 30}},

			},

			Validation: ValidationSpec{

				ExpectedLatency: 1000 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 4.0, MemoryGB: 8.0, NetworkMbps: 100,

				},

				SuccessRate: 0.89,

			},

		},

	}



	// Scaling operations templates.

	g.templates["scaling"] = []IntentTemplate{

		{

			Type:       "scaling",

			Complexity: "simple",

			Template:   "Scale {{.function}} to {{.replicas}} replicas",

			Parameters: map[string]ParameterSpec{

				"function": {Type: "enum", Range: []interface{}{"AMF", "SMF", "UPF"}},

				"replicas": {Type: "int", Range: []interface{}{1, 10}},

			},

			Validation: ValidationSpec{

				ExpectedLatency: 400 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 2.0, MemoryGB: 4.0, NetworkMbps: 50,

				},

				SuccessRate: 0.96,

			},

		},

		{

			Type:       "scaling",

			Complexity: "moderate",

			Template:   "Configure auto-scaling for {{.function}} with min {{.min}} max {{.max}} based on {{.metric}}",

			Parameters: map[string]ParameterSpec{

				"function": {Type: "string", Pattern: "[a-z]+-[a-z0-9]{8}"},

				"min":      {Type: "int", Range: []interface{}{1, 5}},

				"max":      {Type: "int", Range: []interface{}{5, 20}},

				"metric":   {Type: "enum", Range: []interface{}{"cpu", "memory", "requests", "custom"}},

			},

			Validation: ValidationSpec{

				ExpectedLatency: 800 * time.Millisecond,

				ExpectedResources: ResourceRequirements{

					CPU: 1.0, MemoryGB: 2.0, NetworkMbps: 20,

				},

				SuccessRate: 0.91,

			},

		},

	}

}



// Generate performs generate operation.

func (g *IntentGenerator) Generate(workloadType, complexity string) *NetworkIntent {

	g.mu.Lock()

	defer g.mu.Unlock()



	// Get templates for workload type.

	templates, ok := g.templates[workloadType]

	if !ok {

		// Fallback to core network.

		templates = g.templates["core_network"]

	}



	// Find template with matching complexity.

	var template *IntentTemplate

	for _, t := range templates {

		if t.Complexity == complexity {

			template = &t

			break

		}

	}



	// Fallback to first template if no match.

	if template == nil && len(templates) > 0 {

		template = &templates[0]

	}



	// Generate intent from template.

	intent := &NetworkIntent{

		ID:         fmt.Sprintf("intent-%d-%d", time.Now().UnixNano(), g.random.Intn(10000)),

		Type:       workloadType,

		Complexity: complexity,

		Timestamp:  time.Now(),

		Parameters: make(map[string]interface{}),

	}



	// Generate parameters.

	for name, spec := range template.Parameters {

		intent.Parameters[name] = g.generateParameter(spec)

	}



	// Generate intent text from template.

	intent.Intent = g.fillTemplate(template.Template, intent.Parameters)



	return intent

}



func (g *IntentGenerator) generateParameter(spec ParameterSpec) interface{} {

	switch spec.Type {

	case "string":

		if spec.Pattern != "" {

			return g.generateStringFromPattern(spec.Pattern)

		}

		return fmt.Sprintf("value-%d", g.random.Intn(10000))



	case "int":

		if len(spec.Range) == 2 {

			minVal := spec.Range[0].(int)

			maxVal := spec.Range[1].(int)

			return minVal + g.random.Intn(maxVal-minVal+1)

		}

		return g.random.Intn(100)



	case "float":

		if len(spec.Range) == 2 {

			minVal := spec.Range[0].(float64)

			maxVal := spec.Range[1].(float64)

			return minVal + g.random.Float64()*(maxVal-minVal)

		}

		return g.random.Float64() * 100



	case "bool":

		return g.random.Float64() < 0.5



	case "enum":

		if len(spec.Range) > 0 {

			return spec.Range[g.random.Intn(len(spec.Range))]

		}

		return "default"



	default:

		return nil

	}

}



func (g *IntentGenerator) generateStringFromPattern(pattern string) string {

	// Simplified pattern generation.

	// In production, use a proper regex-based generator.



	if pattern == "amf-[a-z0-9]{8}" {

		return fmt.Sprintf("amf-%08x", g.random.Uint32())

	}

	if pattern == "smf-[a-z0-9]{8}" {

		return fmt.Sprintf("smf-%08x", g.random.Uint32())

	}

	if pattern == "upf-[a-z0-9]{8}" {

		return fmt.Sprintf("upf-%08x", g.random.Uint32())

	}

	if pattern == "slice-[a-z0-9]{8}" {

		return fmt.Sprintf("slice-%08x", g.random.Uint32())

	}

	if pattern == "ric-[a-z0-9]{8}" {

		return fmt.Sprintf("ric-%08x", g.random.Uint32())

	}

	if pattern == "odu-[a-z0-9]{8}" {

		return fmt.Sprintf("odu-%08x", g.random.Uint32())

	}

	if pattern == "ocu-[a-z0-9]{8}" {

		return fmt.Sprintf("ocu-%08x", g.random.Uint32())

	}

	if pattern == "cell-[a-z0-9]{8}" {

		return fmt.Sprintf("cell-%08x", g.random.Uint32())

	}

	if pattern == "policy-[a-z0-9]{8}" {

		return fmt.Sprintf("policy-%08x", g.random.Uint32())

	}

	if pattern == "[a-z]+-[a-z0-9]{8}" {

		funcs := []string{"amf", "smf", "upf", "ausf", "nrf"}

		return fmt.Sprintf("%s-%08x", funcs[g.random.Intn(len(funcs))], g.random.Uint32())

	}



	return fmt.Sprintf("gen-%08x", g.random.Uint32())

}



func (g *IntentGenerator) fillTemplate(template string, params map[string]interface{}) string {

	result := template



	// Simple template filling.

	// In production, use text/template package.

	for key, value := range params {

		placeholder := fmt.Sprintf("{{.%s}}", key)

		result = strings.ReplaceAll(result, placeholder, fmt.Sprintf("%v", value))

	}



	return result

}



// Additional generator types.



// BurstLoadGenerator generates burst traffic patterns.

type BurstLoadGenerator struct {

	*RealisticTelecomGenerator

	burstSize     int

	burstInterval time.Duration

}



// NewBurstLoadGenerator creates a burst load generator.

func NewBurstLoadGenerator(id int, config *LoadTestConfig, logger *zap.Logger) (*BurstLoadGenerator, error) {

	base, err := NewRealisticTelecomGenerator(id, config, logger)

	if err != nil {

		return nil, err

	}



	return &BurstLoadGenerator{

		RealisticTelecomGenerator: base,

		burstSize:                 20,

		burstInterval:             10 * time.Second,

	}, nil

}



// ChaosLoadGenerator generates chaotic traffic patterns for stress testing.

type ChaosLoadGenerator struct {

	*RealisticTelecomGenerator

	chaosLevel float64 // 0.0 to 1.0

}



// NewChaosLoadGenerator creates a chaos load generator.

func NewChaosLoadGenerator(id int, config *LoadTestConfig, logger *zap.Logger) (*ChaosLoadGenerator, error) {

	base, err := NewRealisticTelecomGenerator(id, config, logger)

	if err != nil {

		return nil, err

	}



	return &ChaosLoadGenerator{

		RealisticTelecomGenerator: base,

		chaosLevel:                0.3, // 30% chaos

	}, nil

}



// StandardLoadGenerator provides basic load generation.

type StandardLoadGenerator struct {

	*RealisticTelecomGenerator

}



// NewStandardLoadGenerator creates a standard load generator.

func NewStandardLoadGenerator(id int, config *LoadTestConfig, logger *zap.Logger) (*StandardLoadGenerator, error) {

	base, err := NewRealisticTelecomGenerator(id, config, logger)

	if err != nil {

		return nil, err

	}



	return &StandardLoadGenerator{

		RealisticTelecomGenerator: base,

	}, nil

}

