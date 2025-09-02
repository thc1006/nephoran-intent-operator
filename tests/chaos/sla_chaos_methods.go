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
