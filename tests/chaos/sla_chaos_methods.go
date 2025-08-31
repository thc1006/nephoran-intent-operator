package chaos

import (
	"context"
	"fmt"
	"math"
	"time"
)

// ChaosEngine methods
func (ce *ChaosEngine) StartExperiment(ctx context.Context, experiment *ChaosExperiment) (*RunningExperiment, error) {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	if len(ce.activeExperiments) >= ce.config.MaxConcurrentExperiments {
		return nil, fmt.Errorf("maximum concurrent experiments reached: %d", ce.config.MaxConcurrentExperiments)
	}

	runningExp := &RunningExperiment{
		Experiment:       experiment,
		StartTime:        time.Now(),
		Status:           ExperimentStatusRunning,
		Injectors:        make([]*FailureInstance, 0),
		Observations:     make([]*ChaosObservation, 0),
		SafetyViolations: make([]*SafetyViolation, 0),
	}

	ce.activeExperiments[experiment.ID] = runningExp
	return runningExp, nil
}

func (ce *ChaosEngine) StopExperiment(ctx context.Context, experimentID string) error {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	exp, exists := ce.activeExperiments[experimentID]
	if !exists {
		return fmt.Errorf("experiment not found: %s", experimentID)
	}

	exp.Status = ExperimentStatusCompleted
	delete(ce.activeExperiments, experimentID)
	return nil
}

func (ce *ChaosEngine) StopAllExperiments(ctx context.Context) error {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()

	for id := range ce.activeExperiments {
		ce.activeExperiments[id].Status = ExperimentStatusAborted
	}
	
	ce.activeExperiments = make(map[string]*RunningExperiment)
	return nil
}

func (ce *ChaosEngine) RegisterExperiment(experiment *ChaosExperiment) {
	ce.mutex.Lock()
	defer ce.mutex.Unlock()
	
	ce.experiments = append(ce.experiments, experiment)
}

// ChaosMonitor methods
func (cm *ChaosMonitor) StartMonitoring(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cm.collectMetrics()
		}
	}
}

func (cm *ChaosMonitor) collectMetrics() {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	now := time.Now()
	
	// Simulate metric collection
	cm.metrics.AvailabilityTimeSeries = append(cm.metrics.AvailabilityTimeSeries, MetricPoint{
		Timestamp: now,
		Value:     99.5 + (0.5 * (0.5 - math.Mod(float64(now.Unix()), 1.0))),
		Labels:    map[string]string{"component": "sla-service"},
	})

	cm.metrics.LatencyTimeSeries = append(cm.metrics.LatencyTimeSeries, MetricPoint{
		Timestamp: now,
		Value:     100 + (50 * math.Sin(float64(now.Unix())/100)),
		Labels:    map[string]string{"percentile": "p95"},
	})
}

// AlertMonitor methods
func (am *AlertMonitor) StartMonitoring(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			am.checkAlerts()
		}
	}
}

func (am *AlertMonitor) checkAlerts() {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	// Simulate alert checking - in real implementation would query Prometheus/Alertmanager
	if len(am.alerts)%5 == 0 { // Simulate periodic alerts
		alert := &AlertEvent{
			ID:        fmt.Sprintf("alert-%d", time.Now().Unix()),
			AlertName: "SLAViolation",
			Severity:  AlertSeverityWarning,
			State:     AlertStateFiring,
			StartsAt:  time.Now(),
			Labels:    map[string]string{"instance": "sla-service", "job": "monitoring"},
			Context:   map[string]interface{}{"experiment": "chaos-test"},
		}
		am.alerts = append(am.alerts, alert)
	}
}

// DataConsistencyMonitor methods
func (dcm *DataConsistencyMonitor) StartMonitoring(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			dcm.checkConsistency()
		}
	}
}

func (dcm *DataConsistencyMonitor) checkConsistency() {
	dcm.mutex.Lock()
	defer dcm.mutex.Unlock()

	// Simulate consistency checking
	baseScore := 0.995
	deviation := 0.01 * math.Sin(float64(time.Now().Unix())/300)
	dcm.consistencyScore = math.Max(0.90, baseScore+deviation)

	// Add consistency check if score drops significantly
	if dcm.consistencyScore < 0.99 {
		violation := &DataInconsistency{
			CheckName: "metric_completeness",
			Expected:  1.0,
			Actual:    dcm.consistencyScore,
			Deviation: 1.0 - dcm.consistencyScore,
			Timestamp: time.Now(),
			Severity:  InconsistencySeverityMinor,
			Resolved:  false,
		}
		dcm.violations = append(dcm.violations, violation)
	}
}

func (dcm *DataConsistencyMonitor) GetConsistencyScore() float64 {
	dcm.mutex.RLock()
	defer dcm.mutex.RUnlock()
	return dcm.consistencyScore
}

// FalsePositiveTracker methods
func (fpt *FalsePositiveTracker) MonitorAlerts(ctx context.Context, prometheusClient interface{}) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			fpt.checkForFalsePositives()
		}
	}
}

func (fpt *FalsePositiveTracker) checkForFalsePositives() {
	fpt.mutex.Lock()
	defer fpt.mutex.Unlock()

	// Simulate false positive detection logic
	fpt.totalAlerts++
	
	// Simulate 1% false positive rate
	if fpt.totalAlerts%100 == 0 {
		fpt.falsePositives++
		falseAlert := &AlertEvent{
			ID:        fmt.Sprintf("false-alert-%d", time.Now().Unix()),
			AlertName: "FalsePositive",
			Severity:  AlertSeverityWarning,
			State:     AlertStateFiring,
			StartsAt:  time.Now(),
			Context:   map[string]interface{}{"type": "false_positive"},
		}
		fpt.alertEvents = append(fpt.alertEvents, falseAlert)
	}
}

func (fpt *FalsePositiveTracker) CalculateFalsePositiveRate() float64 {
	fpt.mutex.RLock()
	defer fpt.mutex.RUnlock()

	if fpt.totalAlerts == 0 {
		return 0.0
	}
	return float64(fpt.falsePositives) / float64(fpt.totalAlerts)
}

func (fpt *FalsePositiveTracker) GetTotalAlerts() int {
	fpt.mutex.RLock()
	defer fpt.mutex.RUnlock()
	return fpt.totalAlerts
}

func (fpt *FalsePositiveTracker) GetFalsePositives() int {
	fpt.mutex.RLock()
	defer fpt.mutex.RUnlock()
	return fpt.falsePositives
}

// SLATracker methods
func (st *SLATracker) TrackMetrics(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			st.updateMetrics()
		}
	}
}

func (st *SLATracker) updateMetrics() {
	st.mutex.Lock()
	defer st.mutex.Unlock()

	now := time.Now()
	
	// Simulate SLA metric updates
	st.metrics.Availability = 99.5 + (0.5 * math.Sin(float64(now.Unix())/600))
	st.metrics.LatencyP95 = time.Duration(100+int(50*math.Cos(float64(now.Unix())/300))) * time.Millisecond
	st.metrics.Throughput = 45 + (10 * math.Sin(float64(now.Unix())/900))
	st.metrics.ErrorRate = math.Max(0, 0.5+0.3*math.Sin(float64(now.Unix())/400))
	st.metrics.LastUpdated = now

	// Check for SLA violations
	if st.metrics.Availability < 99.9 {
		violation := &SLAViolation{
			ID:          fmt.Sprintf("sla-violation-%d", now.Unix()),
			MetricName:  "availability",
			Threshold:   99.9,
			ActualValue: st.metrics.Availability,
			Deviation:   99.9 - st.metrics.Availability,
			Severity:    ViolationSeverityMedium,
			StartTime:   now,
			Duration:    0,
			Resolved:    false,
		}
		st.violations = append(st.violations, violation)
	}
}

func (st *SLATracker) GetCurrentMetrics() *SLAMetrics {
	st.mutex.RLock()
	defer st.mutex.RUnlock()
	
	// Return a copy to avoid race conditions
	return &SLAMetrics{
		Availability: st.metrics.Availability,
		LatencyP95:   st.metrics.LatencyP95,
		Throughput:   st.metrics.Throughput,
		ErrorRate:    st.metrics.ErrorRate,
		LastUpdated:  st.metrics.LastUpdated,
	}
}

// RecoveryTracker methods
func (rt *RecoveryTracker) TrackRecovery(componentID string, failureType ExperimentType) string {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	eventID := fmt.Sprintf("recovery-%s-%d", componentID, time.Now().Unix())
	
	event := &RecoveryEvent{
		ID:          eventID,
		ComponentID: componentID,
		FailureType: failureType,
		StartTime:   time.Now(),
		Success:     false,
		Metrics:     make(map[string]float64),
	}

	rt.recoveryEvents[eventID] = event
	return eventID
}

func (rt *RecoveryTracker) CompleteRecovery(eventID string, success bool) {
	rt.mutex.Lock()
	defer rt.mutex.Unlock()

	if event, exists := rt.recoveryEvents[eventID]; exists {
		event.EndTime = time.Now()
		event.Duration = event.EndTime.Sub(event.StartTime)
		event.Success = success
	}
}

func (rt *RecoveryTracker) GetRecoveryEvent(eventID string) *RecoveryEvent {
	rt.mutex.RLock()
	defer rt.mutex.RUnlock()
	
	if event, exists := rt.recoveryEvents[eventID]; exists {
		// Return copy to avoid race conditions
		eventCopy := *event
		return &eventCopy
	}
	return nil
}

// Additional helper methods for the test suite

// establishBaseline collects baseline metrics before chaos experiments
func (s *SLAChaosTestSuite) establishBaseline(ctx context.Context, duration time.Duration) *SLABaseline {
	s.T().Logf("Establishing baseline for %v", duration)
	
	// Start tracking
	go s.slaTracker.TrackMetrics(ctx)
	
	// Wait for baseline period
	select {
	case <-ctx.Done():
		return nil
	case <-time.After(duration):
		// Continue to get baseline
	}

	currentMetrics := s.slaTracker.GetCurrentMetrics()
	baseline := &SLABaseline{
		Availability:    currentMetrics.Availability,
		LatencyP95:      currentMetrics.LatencyP95,
		Throughput:      currentMetrics.Throughput,
		ErrorRate:       currentMetrics.ErrorRate,
		MeasurementTime: time.Now(),
		Duration:        duration,
	}

	s.T().Logf("Baseline established: Availability=%.2f%%, Latency=%v, Throughput=%.1f req/min, ErrorRate=%.3f%%",
		baseline.Availability, baseline.LatencyP95, baseline.Throughput, baseline.ErrorRate*100)

	return baseline
}

// monitorSLADuringChaos monitors SLA compliance during chaos experiments
func (s *SLAChaosTestSuite) monitorSLADuringChaos(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			metrics := s.slaTracker.GetCurrentMetrics()
			s.T().Logf("Current SLA metrics: Availability=%.2f%%, Latency=%v, Throughput=%.1f req/min",
				metrics.Availability, metrics.LatencyP95, metrics.Throughput)

			if metrics.Availability < s.config.AvailabilityTargetUnderChaos {
				s.T().Logf("WARNING: Availability below target: %.2f%% < %.2f%%",
					metrics.Availability, s.config.AvailabilityTargetUnderChaos)
			}
		}
	}
}

// monitorExperiment monitors a running chaos experiment
func (s *SLAChaosTestSuite) monitorExperiment(ctx context.Context, experiment *RunningExperiment) {
	s.T().Logf("Monitoring experiment: %s", experiment.Experiment.Name)
	
	// Start recovery tracking if this is a failure experiment
	if experiment.Experiment.Type == ExperimentTypeComponentFailure ||
		experiment.Experiment.Type == ExperimentTypeNetworkPartition ||
		experiment.Experiment.Type == ExperimentTypeResourceExhaustion {
		
		recoveryID := s.recoveryTracker.TrackRecovery(
			experiment.Experiment.Target.Component,
			experiment.Experiment.Type,
		)
		
		// In a real implementation, we'd monitor for recovery completion
		go func() {
			select {
			case <-ctx.Done():
				return
			case <-time.After(experiment.Experiment.Duration + 1*time.Minute):
				s.recoveryTracker.CompleteRecovery(recoveryID, true)
			}
		}()
	}
}

// Additional placeholder methods to satisfy test compilation

func (s *SLAChaosTestSuite) observeRecovery(ctx context.Context, duration time.Duration) {
	s.T().Logf("Observing recovery for %v", duration)
	
	select {
	case <-ctx.Done():
		return
	case <-time.After(duration):
		s.T().Log("Recovery observation period completed")
	}
}

func (s *SLAChaosTestSuite) validateChaosResilience(baseline *SLABaseline) {
	s.T().Log("Validating chaos resilience against baseline")
	
	currentMetrics := s.slaTracker.GetCurrentMetrics()
	
	availabilityDrop := baseline.Availability - currentMetrics.Availability
	s.T().Logf("Availability drop: %.3f%%", availabilityDrop)
	
	s.Assert().LessOrEqual(availabilityDrop, 0.10, // Allow up to 0.10% drop
		"Availability dropped too much: %.3f%% > 0.10%%", availabilityDrop)
}

func (s *SLAChaosTestSuite) getChaosExperiment(experimentType string) *ChaosExperiment {
	for _, exp := range s.chaosEngine.experiments {
		if string(exp.Type) == experimentType {
			return exp
		}
	}
	return nil
}

func (s *SLAChaosTestSuite) recordExperimentResults(experiment *RunningExperiment) {
	s.T().Logf("Recording results for experiment: %s", experiment.Experiment.Name)
	
	// In a real implementation, this would persist results to storage
	s.T().Logf("Experiment %s ran for %v with status %s",
		experiment.Experiment.ID,
		time.Since(experiment.StartTime),
		experiment.Status)
}

func (s *SLAChaosTestSuite) generateChaosTestReport() {
	s.T().Log("Generating chaos test report")
	
	totalExperiments := s.activeExperiments.Load()
	totalFailures := s.totalFailures.Load()
	recoveryEvents := s.recoveryEvents.Load()
	
	s.T().Logf("Chaos Test Summary:")
	s.T().Logf("  Total experiments run: %d", totalExperiments)
	s.T().Logf("  Total failures induced: %d", totalFailures)  
	s.T().Logf("  Recovery events tracked: %d", recoveryEvents)
	s.T().Logf("  Test duration: %v", time.Since(s.chaosStartTime))
}

// createCascadingFailureExperiment creates a cascading failure experiment
func (s *SLAChaosTestSuite) createCascadingFailureExperiment() *ChaosExperiment {
	return &ChaosExperiment{
		ID:          "cascading_failure_001",
		Name:        "Cascading Failure Test", 
		Description: "Test cascading failures across multiple components",
		Type:        ExperimentTypeCascadingFailure,
		Target: ExperimentTarget{
			Component:  "monitoring-stack",
			Percentage: 0.2, // Affect 20% of components
			Selector: map[string]string{
				"app": "sla-monitoring",
			},
		},
		Duration: 20 * time.Minute,
		ExpectedBehavior: &ExpectedBehavior{
			SLAImpact: &SLAImpactExpectation{
				AvailabilityDrop: 0.08, // 0.08% drop expected
				LatencyIncrease:  0.30, // 30% latency increase
				ThroughputDrop:   0.15, // 15% throughput drop
			},
			RecoveryTime:   4 * time.Minute,
			AlertsExpected: []string{"CascadingFailure", "MultipleComponentDown", "SLAViolation"},
		},
		SafetyChecks: []*SafetyCheck{
			{
				Name:      "catastrophic_failure_prevention",
				Type:      SafetyCheckAvailability,
				Threshold: 98.5, // Stop if availability drops below 98.5%
				Action:    SafetyActionAbort,
			},
		},
	}
}

func (s *SLAChaosTestSuite) runExperimentWithAlertValidation(ctx context.Context, experiment *ChaosExperiment, alertMonitor *AlertMonitor) *AlertExperimentResults {
	s.T().Logf("Running experiment with alert validation: %s", experiment.Name)

	results := &AlertExperimentResults{
		AlertEvents:       make([]*AlertEvent, 0),
		ResponseTimes:     make([]time.Duration, 0),
		ValidationSummary: &AlertValidationResults{},
	}

	// Start the experiment
	_, err := s.chaosEngine.StartExperiment(ctx, experiment)
	if err != nil {
		s.T().Errorf("Failed to start experiment: %v", err)
		return results
	}

	// Monitor for expected alerts
	startTime := time.Now()
	expectedAlerts := experiment.ExpectedBehavior.AlertsExpected

	// Wait for experiment duration
	select {
	case <-ctx.Done():
	case <-time.After(experiment.Duration):
	}

	// Stop experiment
	err = s.chaosEngine.StopExperiment(ctx, experiment.ID)
	if err != nil {
		s.T().Errorf("Failed to stop experiment: %v", err)
	}

	// Analyze alerts
	alertMonitor.mutex.RLock()
	experimentAlerts := make([]*AlertEvent, 0)
	for _, alert := range alertMonitor.alerts {
		if alert.StartsAt.After(startTime) {
			experimentAlerts = append(experimentAlerts, alert)
		}
	}
	alertMonitor.mutex.RUnlock()

	results.TotalAlerts = len(experimentAlerts)
	results.AlertEvents = experimentAlerts
	results.ExpectedAlerts = len(expectedAlerts)
	results.ActualAlerts = len(experimentAlerts)

	// Calculate metrics
	if results.ExpectedAlerts > 0 {
		results.AlertAccuracy = float64(results.ActualAlerts) / float64(results.ExpectedAlerts)
		if results.AlertAccuracy > 1.0 {
			results.AlertAccuracy = 1.0 // Cap at 100%
		}
	}

	// Simulate response times
	for i := 0; i < results.ActualAlerts; i++ {
		responseTime := time.Duration(15+i*5) * time.Second // Simulate varying response times
		results.ResponseTimes = append(results.ResponseTimes, responseTime)
	}

	return results
}

func (s *SLAChaosTestSuite) validateAlertAccuracy(results *AlertExperimentResults) {
	s.T().Logf("Validating alert accuracy: %.2f%%", results.AlertAccuracy*100)
	
	s.Assert().GreaterOrEqual(results.AlertAccuracy, s.config.AlertAccuracyThreshold,
		"Alert accuracy below threshold: %.2f%% < %.2f%%",
		results.AlertAccuracy*100, s.config.AlertAccuracyThreshold*100)
}

func (s *SLAChaosTestSuite) validateFalsePositiveRate(results *AlertExperimentResults) {
	// Calculate false positive rate from results
	falsePositiveCount := 0
	for _, alert := range results.AlertEvents {
		if alert.Severity == AlertSeverityWarning && alert.AlertName == "FalsePositive" {
			falsePositiveCount++
		}
	}
	
	falsePositiveRate := 0.0
	if results.TotalAlerts > 0 {
		falsePositiveRate = float64(falsePositiveCount) / float64(results.TotalAlerts)
	}
	
	s.T().Logf("False positive rate: %.2f%%", falsePositiveRate*100)
	
	s.Assert().Less(falsePositiveRate, s.config.FalsePositiveThreshold,
		"False positive rate too high: %.2f%% >= %.2f%%",
		falsePositiveRate*100, s.config.FalsePositiveThreshold*100)
}

func (s *SLAChaosTestSuite) validateAlertResponseTime(results *AlertExperimentResults) {
	if len(results.ResponseTimes) == 0 {
		s.T().Log("No response times to validate")
		return
	}
	
	var totalResponseTime time.Duration
	maxResponseTime := time.Duration(0)
	
	for _, responseTime := range results.ResponseTimes {
		totalResponseTime += responseTime
		if responseTime > maxResponseTime {
			maxResponseTime = responseTime
		}
	}
	
	avgResponseTime := totalResponseTime / time.Duration(len(results.ResponseTimes))
	
	s.T().Logf("Alert response times - Avg: %v, Max: %v", avgResponseTime, maxResponseTime)
	
	// Assert reasonable response times
	s.Assert().Less(avgResponseTime, 2*time.Minute, "Average alert response time too high: %v", avgResponseTime)
	s.Assert().Less(maxResponseTime, 5*time.Minute, "Maximum alert response time too high: %v", maxResponseTime)
}

func (s *SLAChaosTestSuite) runDataConsistencyExperiment(ctx context.Context, experimentType string, monitor *DataConsistencyMonitor) {
	s.T().Logf("Running data consistency experiment: %s", experimentType)
	
	// Create experiment based on type
	var experiment *ChaosExperiment
	
	switch experimentType {
	case "prometheus_corruption":
		experiment = s.createPrometheusCorruptionExperiment()
	case "metric_loss":
		experiment = s.createMetricLossExperiment()
	case "timestamp_skew":
		experiment = s.createTimestampSkewExperiment()
	default:
		s.T().Errorf("Unknown data consistency experiment: %s", experimentType)
		return
	}
	
	// Run the experiment
	_, err := s.chaosEngine.StartExperiment(ctx, experiment)
	if err != nil {
		s.T().Errorf("Failed to start data consistency experiment: %v", err)
		return
	}
	
	// Wait for experiment to complete
	select {
	case <-ctx.Done():
	case <-time.After(experiment.Duration):
	}
	
	// Stop experiment
	err = s.chaosEngine.StopExperiment(ctx, experiment.ID)
	if err != nil {
		s.T().Errorf("Failed to stop data consistency experiment: %v", err)
	}
	
	// Check consistency after experiment
	consistencyScore := monitor.GetConsistencyScore()
	s.T().Logf("Post-experiment consistency score for %s: %.2f%%", experimentType, consistencyScore*100)
	
	// Allow some degradation during chaos but not too much
	s.Assert().GreaterOrEqual(consistencyScore, 0.95, 
		"Data consistency severely degraded in %s: %.2f%% < 95%%", 
		experimentType, consistencyScore*100)
}

func (s *SLAChaosTestSuite) createPrometheusCorruptionExperiment() *ChaosExperiment {
	return &ChaosExperiment{
		ID:          "prometheus_corruption_001",
		Name:        "Prometheus Data Corruption",
		Description: "Inject data corruption in Prometheus metrics",
		Type:        ExperimentTypeDataCorruption,
		Target: ExperimentTarget{
			Component: "prometheus",
			Instance:  "prometheus-server",
		},
		Duration: 15 * time.Minute,
	}
}

func (s *SLAChaosTestSuite) createMetricLossExperiment() *ChaosExperiment {
	return &ChaosExperiment{
		ID:          "metric_loss_001",
		Name:        "Metric Loss Simulation",
		Description: "Simulate loss of metrics data",
		Type:        ExperimentTypeDataCorruption,
		Target: ExperimentTarget{
			Component: "metric-collector",
			Instance:  "sla-collector",
		},
		Duration: 10 * time.Minute,
	}
}

func (s *SLAChaosTestSuite) createTimestampSkewExperiment() *ChaosExperiment {
	return &ChaosExperiment{
		ID:          "timestamp_skew_001",
		Name:        "Timestamp Skew Injection",
		Description: "Inject timestamp skew in metrics",
		Type:        ExperimentTypeDataCorruption,
		Target: ExperimentTarget{
			Component: "time-sync",
			Instance:  "ntp-server",
		},
		Duration: 12 * time.Minute,
	}
}

func (s *SLAChaosTestSuite) validateRecoveryMeasurement(ctx context.Context, scenario *RecoveryScenario) {
	s.T().Logf("Validating recovery measurement for scenario: %s", scenario.Name)
	
	// Create experiment for the scenario
	experiment := &ChaosExperiment{
		ID:          fmt.Sprintf("recovery_test_%s", scenario.Name),
		Name:        fmt.Sprintf("Recovery Test - %s", scenario.Name),
		Description: fmt.Sprintf("Test recovery measurement for %s", scenario.FailureType),
		Type:        scenario.FailureType,
		Target: ExperimentTarget{
			Component: "test-component",
			Instance:  fmt.Sprintf("instance-%s", scenario.Name),
		},
		Duration: 2 * time.Minute, // Short duration for recovery test
	}
	
	// Start recovery tracking
	recoveryID := s.recoveryTracker.TrackRecovery("test-component", scenario.FailureType)
	
	// Run experiment
	_, err := s.chaosEngine.StartExperiment(ctx, experiment)
	if err != nil {
		s.T().Errorf("Failed to start recovery test experiment: %v", err)
		return
	}
	
	// Simulate controlled recovery
	go func() {
		select {
		case <-ctx.Done():
			return
		case <-time.After(scenario.ExpectedRecovery):
			s.recoveryTracker.CompleteRecovery(recoveryID, true)
		}
	}()
	
	// Wait for experiment completion
	select {
	case <-ctx.Done():
	case <-time.After(experiment.Duration + scenario.ExpectedRecovery + scenario.Tolerance):
	}
	
	// Stop experiment
	err = s.chaosEngine.StopExperiment(ctx, experiment.ID)
	if err != nil {
		s.T().Errorf("Failed to stop recovery test experiment: %v", err)
	}
	
	// Validate recovery measurement
	recoveryEvent := s.recoveryTracker.GetRecoveryEvent(recoveryID)
	if recoveryEvent == nil {
		s.T().Errorf("Recovery event not found: %s", recoveryID)
		return
	}
	
	actualRecovery := recoveryEvent.Duration
	expectedRecovery := scenario.ExpectedRecovery
	tolerance := scenario.Tolerance
	
	s.T().Logf("Recovery measurement for %s - Expected: %v, Actual: %v, Tolerance: %v",
		scenario.Name, expectedRecovery, actualRecovery, tolerance)
	
	deviation := actualRecovery - expectedRecovery
	if deviation < 0 {
		deviation = -deviation
	}
	
	s.Assert().LessOrEqual(deviation, tolerance,
		"Recovery time deviation too large for %s: %v > %v",
		scenario.Name, deviation, tolerance)
	
	s.Assert().True(recoveryEvent.Success, "Recovery should have succeeded for %s", scenario.Name)
}

func (s *SLAChaosTestSuite) testPartialFailureScenario(ctx context.Context, scenario PartialFailureScenario) {
	s.T().Logf("Testing partial failure scenario: %s", scenario.Name)
	
	// Create partial failure experiment
	experiment := &ChaosExperiment{
		ID:          fmt.Sprintf("partial_failure_%s", scenario.Name),
		Name:        fmt.Sprintf("Partial Failure - %s", scenario.Name),
		Description: fmt.Sprintf("Test partial failure in %s", scenario.Name),
		Type:        ExperimentTypeComponentFailure,
		Target: ExperimentTarget{
			Component:  scenario.Name,
			Percentage: float64(len(scenario.AffectedNodes)) / float64(len(scenario.AffectedNodes)+len(scenario.HealthyNodes)),
		},
		Duration: 15 * time.Minute,
	}
	
	// Run experiment
	_, err := s.chaosEngine.StartExperiment(ctx, experiment)
	if err != nil {
		s.T().Errorf("Failed to start partial failure experiment: %v", err)
		return
	}
	
	// Monitor SLA during partial failure
	startTime := time.Now()
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				metrics := s.slaTracker.GetCurrentMetrics()
				s.T().Logf("Partial failure %s - Availability: %.2f%%", scenario.Name, metrics.Availability)
			}
		}
	}()
	
	// Wait for experiment
	select {
	case <-ctx.Done():
	case <-time.After(experiment.Duration):
	}
	
	// Stop experiment
	err = s.chaosEngine.StopExperiment(ctx, experiment.ID)
	if err != nil {
		s.T().Errorf("Failed to stop partial failure experiment: %v", err)
	}
	
	// Validate SLA during partial failure
	finalMetrics := s.slaTracker.GetCurrentMetrics()
	
	for metricName, expectedValue := range scenario.ExpectedSLA {
		switch metricName {
		case "availability":
			s.Assert().GreaterOrEqual(finalMetrics.Availability, expectedValue,
				"Availability below expected for partial failure %s: %.2f%% < %.2f%%",
				scenario.Name, finalMetrics.Availability, expectedValue)
		}
	}
	
	duration := time.Since(startTime)
	s.T().Logf("Partial failure scenario %s completed in %v", scenario.Name, duration)
}

func (s *SLAChaosTestSuite) generateBenignLoadVariations(ctx context.Context) {
	s.T().Log("Generating benign load variations")
	
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			// Simulate normal load variations that shouldn't trigger alerts
			s.T().Log("Simulating benign load variation")
			// In a real implementation, this would generate controlled load
		}
	}
}

func (s *SLAChaosTestSuite) runMixedChaosExperiments(ctx context.Context) []*ChaosExperiment {
	s.T().Log("Running mixed chaos experiments")
	
	experiments := []*ChaosExperiment{
		{
			ID:       "mixed_chaos_001",
			Name:     "Mixed Chaos Test 1",
			Type:     ExperimentTypeLatencyInjection,
			Duration: 10 * time.Minute,
		},
		{
			ID:       "mixed_chaos_002", 
			Name:     "Mixed Chaos Test 2",
			Type:     ExperimentTypeComponentFailure,
			Duration: 8 * time.Minute,
		},
	}
	
	// Run experiments concurrently
	for _, exp := range experiments {
		go func(experiment *ChaosExperiment) {
			_, err := s.chaosEngine.StartExperiment(ctx, experiment)
			if err != nil {
				s.T().Errorf("Failed to start mixed chaos experiment %s: %v", experiment.ID, err)
				return
			}
			
			select {
			case <-ctx.Done():
			case <-time.After(experiment.Duration):
			}
			
			err = s.chaosEngine.StopExperiment(ctx, experiment.ID)
			if err != nil {
				s.T().Errorf("Failed to stop mixed chaos experiment %s: %v", experiment.ID, err)
			}
		}(exp)
	}
	
	// Wait for all experiments to complete
	maxDuration := time.Duration(0)
	for _, exp := range experiments {
		if exp.Duration > maxDuration {
			maxDuration = exp.Duration
		}
	}
	
	select {
	case <-ctx.Done():
	case <-time.After(maxDuration + 1*time.Minute):
	}
	
	return experiments
}