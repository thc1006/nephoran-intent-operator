// Package validation provides advanced load generation for performance testing
package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/montanaflynn/stats"
	"github.com/onsi/ginkgo/v2"
	vegeta "github.com/tsenart/vegeta/v12/lib"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	nephranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
)

// AdvancedLoadGenerator provides sophisticated load generation capabilities
type AdvancedLoadGenerator struct {
	k8sClient        client.Client
	httpClient       *http.Client
	baseURL          string
	metricsCollector *LoadMetricsCollector
	scenarios        map[string]LoadScenario
	mu               sync.RWMutex
}

// LoadScenario defines a load testing scenario
type LoadScenario interface {
	Generate(ctx context.Context) <-chan *LoadRequest
	GetName() string
	GetDescription() string
}

// LoadRequest represents a single load test request
type LoadRequest struct {
	Method      string
	URL         string
	Body        []byte
	Headers     map[string]string
	Intent      *nephranv1.NetworkIntent
	Timestamp   time.Time
	ScenarioTag string
}

// LoadMetricsCollector collects detailed load test metrics
type LoadMetricsCollector struct {
	requests      int64
	successes     int64
	failures      int64
	totalBytes    int64
	latencies     []time.Duration
	errorTypes    map[string]int64
	statusCodes   map[int]int64
	timestamps    []time.Time
	mu            sync.RWMutex
	startTime     time.Time
}

// LoadTestReport contains comprehensive load test results
type LoadTestReport struct {
	Scenario           string
	Duration           time.Duration
	TotalRequests      int64
	SuccessfulRequests int64
	FailedRequests     int64
	
	// Latency metrics
	MinLatency         time.Duration
	MaxLatency         time.Duration
	MeanLatency        time.Duration
	MedianLatency      time.Duration
	P50Latency         time.Duration
	P75Latency         time.Duration
	P90Latency         time.Duration
	P95Latency         time.Duration
	P99Latency         time.Duration
	StdDevLatency      time.Duration
	
	// Throughput metrics
	RequestsPerSecond  float64
	BytesPerSecond     float64
	
	// Error analysis
	ErrorRate          float64
	ErrorsByType       map[string]int64
	StatusCodeDist     map[int]int64
	
	// Advanced metrics
	ApdexScore         float64  // Application Performance Index
	TimeToFirstByte    time.Duration
	ConnectionErrors   int64
	TimeoutErrors      int64
	
	// Resource metrics
	CPUUsage           []float64
	MemoryUsage        []float64
	NetworkLatency     []time.Duration
}

// NewAdvancedLoadGenerator creates an advanced load generator
func NewAdvancedLoadGenerator(k8sClient client.Client, baseURL string) *AdvancedLoadGenerator {
	return &AdvancedLoadGenerator{
		k8sClient: k8sClient,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		baseURL:          baseURL,
		metricsCollector: NewLoadMetricsCollector(),
		scenarios:        make(map[string]LoadScenario),
	}
}

// NewLoadMetricsCollector creates a metrics collector
func NewLoadMetricsCollector() *LoadMetricsCollector {
	return &LoadMetricsCollector{
		latencies:   make([]time.Duration, 0, 100000),
		errorTypes:  make(map[string]int64),
		statusCodes: make(map[int]int64),
		timestamps:  make([]time.Time, 0, 100000),
		startTime:   time.Now(),
	}
}

// RegisterScenario registers a load testing scenario
func (alg *AdvancedLoadGenerator) RegisterScenario(scenario LoadScenario) {
	alg.mu.Lock()
	defer alg.mu.Unlock()
	alg.scenarios[scenario.GetName()] = scenario
}

// RunScenario executes a specific load testing scenario
func (alg *AdvancedLoadGenerator) RunScenario(ctx context.Context, scenarioName string, duration time.Duration) (*LoadTestReport, error) {
	alg.mu.RLock()
	scenario, exists := alg.scenarios[scenarioName]
	alg.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("scenario '%s' not found", scenarioName)
	}
	
	ginkgo.By(fmt.Sprintf("Running load scenario: %s", scenario.GetDescription()))
	
	// Reset metrics collector
	alg.metricsCollector = NewLoadMetricsCollector()
	
	// Create context with timeout
	testCtx, cancel := context.WithTimeout(ctx, duration)
	defer cancel()
	
	// Generate load
	requestChan := scenario.Generate(testCtx)
	
	// Process requests concurrently
	var wg sync.WaitGroup
	workers := 50 // Number of concurrent workers
	
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			alg.processRequests(testCtx, requestChan, workerID)
		}(i)
	}
	
	// Wait for completion
	wg.Wait()
	
	// Generate report
	return alg.generateReport(scenario.GetName(), duration), nil
}

// processRequests processes load test requests
func (alg *AdvancedLoadGenerator) processRequests(ctx context.Context, requests <-chan *LoadRequest, workerID int) {
	for {
		select {
		case <-ctx.Done():
			return
		case req, ok := <-requests:
			if !ok {
				return
			}
			alg.executeRequest(ctx, req)
		}
	}
}

// executeRequest executes a single load test request
func (alg *AdvancedLoadGenerator) executeRequest(ctx context.Context, req *LoadRequest) {
	startTime := time.Now()
	
	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, req.Method, req.URL, bytes.NewReader(req.Body))
	if err != nil {
		alg.metricsCollector.recordError("request_creation", err)
		return
	}
	
	// Set headers
	for key, value := range req.Headers {
		httpReq.Header.Set(key, value)
	}
	
	// Execute request
	resp, err := alg.httpClient.Do(httpReq)
	latency := time.Since(startTime)
	
	if err != nil {
		alg.metricsCollector.recordError("http_error", err)
		alg.metricsCollector.recordFailure(latency)
		return
	}
	defer resp.Body.Close()
	
	// Read response body
	body, _ := io.ReadAll(resp.Body)
	
	// Record metrics
	alg.metricsCollector.recordSuccess(latency, resp.StatusCode, int64(len(body)))
}

// RunVegetaTest runs a load test using Vegeta library
func (alg *AdvancedLoadGenerator) RunVegetaTest(ctx context.Context, rate uint64, duration time.Duration) (*LoadTestReport, error) {
	ginkgo.By(fmt.Sprintf("Running Vegeta load test: %d req/s for %v", rate, duration))
	
	// Create targeter
	targeter := vegeta.NewStaticTargeter(alg.generateVegetaTargets()...)
	
	// Create attacker
	attacker := vegeta.NewAttacker(
		vegeta.Workers(uint64(rate/10)),
		vegeta.MaxWorkers(100),
		vegeta.Timeout(10*time.Second),
	)
	
	// Metrics to collect results
	var metrics vegeta.Metrics
	
	// Run attack
	for res := range attacker.Attack(targeter, vegeta.Rate{Freq: int(rate), Per: time.Second}, duration, "Load Test") {
		metrics.Add(res)
	}
	
	metrics.Close()
	
	// Convert Vegeta metrics to our report format
	return alg.convertVegetaMetrics(&metrics), nil
}

// generateVegetaTargets generates targets for Vegeta
func (alg *AdvancedLoadGenerator) generateVegetaTargets() []vegeta.Target {
	targets := []vegeta.Target{}
	
	// Generate various intent types
	intents := []string{
		"Deploy AMF with high availability",
		"Configure SMF with QoS policies",
		"Setup UPF for edge deployment",
		"Create network slice for IoT",
		"Deploy Near-RT RIC",
	}
	
	for i, intent := range intents {
		body := map[string]interface{}{
			"apiVersion": "nephoran.io/v1",
			"kind":       "NetworkIntent",
			"metadata": map[string]interface{}{
				"name":      fmt.Sprintf("vegeta-test-%d", i),
				"namespace": "default",
			},
			"spec": map[string]interface{}{
				"intent": intent,
			},
		}
		
		bodyBytes, _ := json.Marshal(body)
		
		targets = append(targets, vegeta.Target{
			Method: "POST",
			URL:    fmt.Sprintf("%s/api/v1/namespaces/default/networkintents", alg.baseURL),
			Body:   bodyBytes,
			Header: http.Header{
				"Content-Type": []string{"application/json"},
			},
		})
	}
	
	return targets
}

// convertVegetaMetrics converts Vegeta metrics to our report format
func (alg *AdvancedLoadGenerator) convertVegetaMetrics(metrics *vegeta.Metrics) *LoadTestReport {
	return &LoadTestReport{
		Scenario:           "Vegeta Load Test",
		Duration:           metrics.Duration,
		TotalRequests:      int64(metrics.Requests),
		SuccessfulRequests: int64(float64(metrics.Requests) * metrics.Success),
		FailedRequests:     int64(float64(metrics.Requests) * (1 - metrics.Success)),
		
		MinLatency:        metrics.Latencies.Min,
		MaxLatency:        metrics.Latencies.Max,
		MeanLatency:       metrics.Latencies.Mean,
		P50Latency:        metrics.Latencies.P50,
		P90Latency:        metrics.Latencies.P90,
		P95Latency:        metrics.Latencies.P95,
		P99Latency:        metrics.Latencies.P99,
		
		RequestsPerSecond: metrics.Rate,
		BytesPerSecond:    metrics.Throughput,
		ErrorRate:         1 - metrics.Success,
		
		StatusCodeDist:    alg.convertStatusCodes(metrics.StatusCodes),
	}
}

// convertStatusCodes converts Vegeta status codes to our format
func (alg *AdvancedLoadGenerator) convertStatusCodes(codes map[string]int) map[int]int64 {
	result := make(map[int]int64)
	for code, count := range codes {
		var statusCode int
		fmt.Sscanf(code, "%d", &statusCode)
		result[statusCode] = int64(count)
	}
	return result
}

// generateReport generates a comprehensive load test report
func (alg *AdvancedLoadGenerator) generateReport(scenario string, duration time.Duration) *LoadTestReport {
	alg.metricsCollector.mu.RLock()
	defer alg.metricsCollector.mu.RUnlock()
	
	report := &LoadTestReport{
		Scenario:           scenario,
		Duration:           duration,
		TotalRequests:      atomic.LoadInt64(&alg.metricsCollector.requests),
		SuccessfulRequests: atomic.LoadInt64(&alg.metricsCollector.successes),
		FailedRequests:     atomic.LoadInt64(&alg.metricsCollector.failures),
		ErrorsByType:       alg.metricsCollector.errorTypes,
		StatusCodeDist:     alg.metricsCollector.statusCodes,
	}
	
	// Calculate latency statistics
	if len(alg.metricsCollector.latencies) > 0 {
		latencies := alg.metricsCollector.latencies
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})
		
		report.MinLatency = latencies[0]
		report.MaxLatency = latencies[len(latencies)-1]
		report.MedianLatency = latencies[len(latencies)/2]
		
		// Calculate percentiles
		report.P50Latency = alg.calculatePercentile(latencies, 50)
		report.P75Latency = alg.calculatePercentile(latencies, 75)
		report.P90Latency = alg.calculatePercentile(latencies, 90)
		report.P95Latency = alg.calculatePercentile(latencies, 95)
		report.P99Latency = alg.calculatePercentile(latencies, 99)
		
		// Calculate mean and standard deviation
		var sum time.Duration
		for _, l := range latencies {
			sum += l
		}
		report.MeanLatency = sum / time.Duration(len(latencies))
		
		// Convert to float64 for stats calculation
		floatLatencies := make([]float64, len(latencies))
		for i, l := range latencies {
			floatLatencies[i] = float64(l.Nanoseconds())
		}
		
		if stdDev, err := stats.StandardDeviation(floatLatencies); err == nil {
			report.StdDevLatency = time.Duration(stdDev)
		}
	}
	
	// Calculate throughput
	if duration > 0 {
		report.RequestsPerSecond = float64(report.TotalRequests) / duration.Seconds()
		report.BytesPerSecond = float64(atomic.LoadInt64(&alg.metricsCollector.totalBytes)) / duration.Seconds()
	}
	
	// Calculate error rate
	if report.TotalRequests > 0 {
		report.ErrorRate = float64(report.FailedRequests) / float64(report.TotalRequests)
	}
	
	// Calculate Apdex score (Application Performance Index)
	report.ApdexScore = alg.calculateApdex(alg.metricsCollector.latencies, 2*time.Second)
	
	return report
}

// calculatePercentile calculates the specified percentile
func (alg *AdvancedLoadGenerator) calculatePercentile(latencies []time.Duration, percentile int) time.Duration {
	if len(latencies) == 0 {
		return 0
	}
	
	index := int(math.Ceil(float64(percentile)/100.0*float64(len(latencies)))) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(latencies) {
		index = len(latencies) - 1
	}
	
	return latencies[index]
}

// calculateApdex calculates the Apdex score
func (alg *AdvancedLoadGenerator) calculateApdex(latencies []time.Duration, threshold time.Duration) float64 {
	if len(latencies) == 0 {
		return 0
	}
	
	satisfied := 0
	tolerating := 0
	
	for _, latency := range latencies {
		if latency <= threshold {
			satisfied++
		} else if latency <= 4*threshold {
			tolerating++
		}
	}
	
	return float64(satisfied+tolerating/2) / float64(len(latencies))
}

// LoadMetricsCollector methods

func (lmc *LoadMetricsCollector) recordSuccess(latency time.Duration, statusCode int, bytes int64) {
	atomic.AddInt64(&lmc.requests, 1)
	atomic.AddInt64(&lmc.successes, 1)
	atomic.AddInt64(&lmc.totalBytes, bytes)
	
	lmc.mu.Lock()
	defer lmc.mu.Unlock()
	
	lmc.latencies = append(lmc.latencies, latency)
	lmc.timestamps = append(lmc.timestamps, time.Now())
	
	if lmc.statusCodes == nil {
		lmc.statusCodes = make(map[int]int64)
	}
	lmc.statusCodes[statusCode]++
}

func (lmc *LoadMetricsCollector) recordFailure(latency time.Duration) {
	atomic.AddInt64(&lmc.requests, 1)
	atomic.AddInt64(&lmc.failures, 1)
	
	lmc.mu.Lock()
	defer lmc.mu.Unlock()
	
	lmc.latencies = append(lmc.latencies, latency)
	lmc.timestamps = append(lmc.timestamps, time.Now())
}

func (lmc *LoadMetricsCollector) recordError(errorType string, err error) {
	lmc.mu.Lock()
	defer lmc.mu.Unlock()
	
	if lmc.errorTypes == nil {
		lmc.errorTypes = make(map[string]int64)
	}
	lmc.errorTypes[errorType]++
}

// Predefined load scenarios

// ConstantRateScenario generates constant rate load
type ConstantRateScenario struct {
	name        string
	description string
	rate        int // requests per second
	duration    time.Duration
}

func NewConstantRateScenario(rate int, duration time.Duration) *ConstantRateScenario {
	return &ConstantRateScenario{
		name:        "constant_rate",
		description: fmt.Sprintf("Constant rate of %d req/s", rate),
		rate:        rate,
		duration:    duration,
	}
}

func (crs *ConstantRateScenario) GetName() string        { return crs.name }
func (crs *ConstantRateScenario) GetDescription() string { return crs.description }

func (crs *ConstantRateScenario) Generate(ctx context.Context) <-chan *LoadRequest {
	requests := make(chan *LoadRequest, 100)
	
	go func() {
		defer close(requests)
		
		ticker := time.NewTicker(time.Second / time.Duration(crs.rate))
		defer ticker.Stop()
		
		requestID := 0
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				req := &LoadRequest{
					Method:      "POST",
					URL:         "http://localhost:8080/api/v1/intents",
					Body:        crs.generateIntentBody(requestID),
					Headers:     map[string]string{"Content-Type": "application/json"},
					Timestamp:   time.Now(),
					ScenarioTag: crs.name,
				}
				
				select {
				case requests <- req:
					requestID++
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	
	return requests
}

func (crs *ConstantRateScenario) generateIntentBody(id int) []byte {
	intent := map[string]interface{}{
		"apiVersion": "nephoran.io/v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("constant-rate-%d", id),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"intent": fmt.Sprintf("Deploy test function %d", id),
		},
	}
	
	body, _ := json.Marshal(intent)
	return body
}

// RampUpScenario gradually increases load
type RampUpScenario struct {
	name        string
	description string
	startRate   int
	endRate     int
	rampTime    time.Duration
}

func NewRampUpScenario(startRate, endRate int, rampTime time.Duration) *RampUpScenario {
	return &RampUpScenario{
		name:        "ramp_up",
		description: fmt.Sprintf("Ramp from %d to %d req/s over %v", startRate, endRate, rampTime),
		startRate:   startRate,
		endRate:     endRate,
		rampTime:    rampTime,
	}
}

func (rus *RampUpScenario) GetName() string        { return rus.name }
func (rus *RampUpScenario) GetDescription() string { return rus.description }

func (rus *RampUpScenario) Generate(ctx context.Context) <-chan *LoadRequest {
	requests := make(chan *LoadRequest, 100)
	
	go func() {
		defer close(requests)
		
		startTime := time.Now()
		requestID := 0
		
		for {
			select {
			case <-ctx.Done():
				return
			default:
				elapsed := time.Since(startTime)
				progress := float64(elapsed) / float64(rus.rampTime)
				if progress > 1.0 {
					progress = 1.0
				}
				
				currentRate := rus.startRate + int(float64(rus.endRate-rus.startRate)*progress)
				sleepDuration := time.Second / time.Duration(currentRate)
				
				req := &LoadRequest{
					Method:      "POST",
					URL:         "http://localhost:8080/api/v1/intents",
					Body:        rus.generateIntentBody(requestID),
					Headers:     map[string]string{"Content-Type": "application/json"},
					Timestamp:   time.Now(),
					ScenarioTag: rus.name,
				}
				
				select {
				case requests <- req:
					requestID++
					time.Sleep(sleepDuration)
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	
	return requests
}

func (rus *RampUpScenario) generateIntentBody(id int) []byte {
	intent := map[string]interface{}{
		"apiVersion": "nephoran.io/v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("ramp-up-%d", id),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"intent": fmt.Sprintf("Deploy function during ramp %d", id),
		},
	}
	
	body, _ := json.Marshal(intent)
	return body
}

// SpikeScenario generates spike traffic patterns
type SpikeScenario struct {
	name         string
	description  string
	baseRate     int
	spikeRate    int
	spikeDuration time.Duration
	restDuration  time.Duration
}

func NewSpikeScenario(baseRate, spikeRate int, spikeDuration, restDuration time.Duration) *SpikeScenario {
	return &SpikeScenario{
		name:          "spike",
		description:   fmt.Sprintf("Spike from %d to %d req/s", baseRate, spikeRate),
		baseRate:      baseRate,
		spikeRate:     spikeRate,
		spikeDuration: spikeDuration,
		restDuration:  restDuration,
	}
}

func (ss *SpikeScenario) GetName() string        { return ss.name }
func (ss *SpikeScenario) GetDescription() string { return ss.description }

func (ss *SpikeScenario) Generate(ctx context.Context) <-chan *LoadRequest {
	requests := make(chan *LoadRequest, 100)
	
	go func() {
		defer close(requests)
		
		requestID := 0
		for {
			// Rest period
			restEnd := time.Now().Add(ss.restDuration)
			for time.Now().Before(restEnd) {
				select {
				case <-ctx.Done():
					return
				default:
					req := ss.generateRequest(requestID, ss.baseRate)
					select {
					case requests <- req:
						requestID++
						time.Sleep(time.Second / time.Duration(ss.baseRate))
					case <-ctx.Done():
						return
					}
				}
			}
			
			// Spike period
			spikeEnd := time.Now().Add(ss.spikeDuration)
			for time.Now().Before(spikeEnd) {
				select {
				case <-ctx.Done():
					return
				default:
					req := ss.generateRequest(requestID, ss.spikeRate)
					select {
					case requests <- req:
						requestID++
						time.Sleep(time.Second / time.Duration(ss.spikeRate))
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()
	
	return requests
}

func (ss *SpikeScenario) generateRequest(id int, rate int) *LoadRequest {
	intent := map[string]interface{}{
		"apiVersion": "nephoran.io/v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("spike-%d-rate-%d", id, rate),
			"namespace": "default",
		},
		"spec": map[string]interface{}{
			"intent": fmt.Sprintf("Deploy function during spike %d", id),
		},
	}
	
	body, _ := json.Marshal(intent)
	
	return &LoadRequest{
		Method:      "POST",
		URL:         "http://localhost:8080/api/v1/intents",
		Body:        body,
		Headers:     map[string]string{"Content-Type": "application/json"},
		Timestamp:   time.Now(),
		ScenarioTag: ss.name,
	}
}

// RealisticTelecomScenario simulates realistic telecom traffic patterns
type RealisticTelecomScenario struct {
	name        string
	description string
}

func NewRealisticTelecomScenario() *RealisticTelecomScenario {
	return &RealisticTelecomScenario{
		name:        "realistic_telecom",
		description: "Realistic telecom traffic with daily patterns",
	}
}

func (rts *RealisticTelecomScenario) GetName() string        { return rts.name }
func (rts *RealisticTelecomScenario) GetDescription() string { return rts.description }

func (rts *RealisticTelecomScenario) Generate(ctx context.Context) <-chan *LoadRequest {
	requests := make(chan *LoadRequest, 100)
	
	go func() {
		defer close(requests)
		
		requestID := 0
		startTime := time.Now()
		
		for {
			select {
			case <-ctx.Done():
				return
			default:
				// Simulate daily traffic pattern
				elapsed := time.Since(startTime)
				hour := int(elapsed.Hours()) % 24
				
				// Traffic varies by hour of day
				var rate int
				switch {
				case hour >= 0 && hour < 6:
					rate = 10 // Low traffic at night
				case hour >= 6 && hour < 9:
					rate = 40 // Morning ramp-up
				case hour >= 9 && hour < 12:
					rate = 60 // Morning peak
				case hour >= 12 && hour < 14:
					rate = 50 // Lunch dip
				case hour >= 14 && hour < 18:
					rate = 70 // Afternoon peak
				case hour >= 18 && hour < 22:
					rate = 45 // Evening
				default:
					rate = 20 // Late evening
				}
				
				// Add some randomness
				rate = rate + rand.Intn(10) - 5
				if rate < 5 {
					rate = 5
				}
				
				req := rts.generateRealisticRequest(requestID)
				select {
				case requests <- req:
					requestID++
					time.Sleep(time.Second / time.Duration(rate))
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	
	return requests
}

func (rts *RealisticTelecomScenario) generateRealisticRequest(id int) *LoadRequest {
	// Variety of realistic telecom intents
	intents := []string{
		"Deploy AMF for subscriber group A with high availability",
		"Configure SMF with premium QoS for enterprise customers",
		"Setup UPF at edge location for low latency services",
		"Create network slice for IoT devices with bandwidth limit 10Mbps",
		"Deploy Near-RT RIC with ML-based traffic optimization",
		"Configure O-DU for dense urban area with beamforming",
		"Update NSSF slice selection policy for emergency services",
		"Scale PCF instances to handle peak hour traffic",
		"Deploy UDM with geographical redundancy",
		"Configure AUSF with enhanced security for roaming users",
	}
	
	intent := intents[id%len(intents)]
	
	body := map[string]interface{}{
		"apiVersion": "nephoran.io/v1",
		"kind":       "NetworkIntent",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("realistic-%d", id),
			"namespace": "default",
			"labels": map[string]string{
				"scenario": "realistic",
				"priority": fmt.Sprintf("%d", rand.Intn(3)+1),
			},
		},
		"spec": map[string]interface{}{
			"intent": intent,
		},
	}
	
	bodyBytes, _ := json.Marshal(body)
	
	return &LoadRequest{
		Method:      "POST",
		URL:         "http://localhost:8080/api/v1/intents",
		Body:        bodyBytes,
		Headers:     map[string]string{"Content-Type": "application/json"},
		Timestamp:   time.Now(),
		ScenarioTag: rts.name,
	}
}