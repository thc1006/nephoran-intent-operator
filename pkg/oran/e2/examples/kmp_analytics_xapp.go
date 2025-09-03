package examples

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	e2 "github.com/thc1006/nephoran-intent-operator/pkg/oran/e2"
)

// KMPAnalyticsXApp represents an xApp that subscribes to KPM measurements and performs analytics.

type KMPAnalyticsXApp struct {
	sdk *e2.XAppSDK

	config *e2.XAppConfig

	e2Manager *e2.E2Manager

	// Analytics state

	subscriptions map[string]*e2.XAppSubscription

	kpiData map[string][]KPIMeasurement

	alertRules []AlertRule

	// Context and cancellation

	ctx context.Context

	cancel context.CancelFunc

	mu sync.RWMutex
}

// KPIMeasurement represents a KPI measurement.

type KPIMeasurement struct {
	Timestamp time.Time `json:"timestamp"`

	NodeID string `json:"node_id"`

	CellID string `json:"cell_id"`

	Metrics map[string]float64 `json:"metrics"`
}

// AlertRule represents an alerting rule.

type AlertRule struct {
	Name string `json:"name"`

	Condition string `json:"condition"`

	Threshold float64 `json:"threshold"`

	Action string `json:"action"`
}

// NewKMPAnalyticsXApp creates a new KMP Analytics xApp.

func NewKMPAnalyticsXApp(config *e2.XAppConfig, e2Manager *e2.E2Manager) (*KMPAnalyticsXApp, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// Create SDK instance

	sdk, err := e2.NewXAppSDK(config, e2Manager)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to create xApp SDK: %w", err)
	}

	xapp := &KMPAnalyticsXApp{
		sdk:           sdk,
		config:        config,
		e2Manager:     e2Manager,
		subscriptions: make(map[string]*e2.XAppSubscription),
		kpiData:       make(map[string][]KPIMeasurement),
		ctx:           ctx,
		cancel:        cancel,
	}

	// Initialize default alert rules

	xapp.alertRules = []AlertRule{
		{
			Name:      "high_throughput",
			Condition: "throughput > threshold",
			Threshold: 100.0,
			Action:    "log",
		},
		{
			Name:      "high_latency",
			Condition: "latency > threshold",
			Threshold: 10.0,
			Action:    "alert",
		},
	}

	return xapp, nil
}

// Start starts the xApp and begins subscription to KMP measurements.

func (x *KMPAnalyticsXApp) Start() error {
	log.Println("Starting KMP Analytics xApp...")

	// Start analytics processing

	go x.processAnalytics()

	// Subscribe to KMP measurements

	if err := x.subscribeToKMPMeasurements(); err != nil {
		return fmt.Errorf("failed to subscribe to KMP measurements: %w", err)
	}

	log.Println("KMP Analytics xApp started successfully")

	return nil
}

// subscribeToKMPMeasurements subscribes to KMP measurements from E2 nodes.

func (x *KMPAnalyticsXApp) subscribeToKMPMeasurements() error {
	ctx := x.ctx

	// Define subscription conditions for KMP measurements

	conditions := map[string]interface{}{
		"measurement_type": "kmp",
		"granularity":      "1s",
		"reporting_period": 5000, // 5 seconds
		"metrics": []string{
			"DRB.UEThpDl",
			"DRB.UEThpUl",
			"RRU.PrbUsedDl",
			"RRU.PrbUsedUl",
		},
	}

	conditionsJSON, err := json.Marshal(conditions)
	if err != nil {
		return fmt.Errorf("failed to marshal subscription conditions: %w", err)
	}

	// Create E2SubscriptionRequest (not XAppSubscription)
	subscriptionReq := &e2.E2SubscriptionRequest{
		NodeID:         x.sdk.GetConfig().E2NodeID,
		SubscriptionID: fmt.Sprintf("kmp-analytics-%d", time.Now().Unix()),
		RequestorID:    x.config.XAppName,
		RanFunctionID:  1, // KMP service model
		EventTriggers: []e2.E2EventTrigger{
			{
				TriggerType: "PERIODIC",
				ReportingPeriod: 5 * time.Second,
				Conditions:  json.RawMessage(conditionsJSON),
			},
		},
		Actions: []e2.E2Action{
			{
				ActionID:   1,
				ActionType: "REPORT",
				ActionDefinition: json.RawMessage(`{
					"report_style": "detailed",
					"metrics": ["throughput", "latency", "resource_utilization", "handover_rate"]
				}`),
			},
		},
		ReportingPeriod: 5 * time.Second,
	}

	// Subscribe to KMP measurements using SDK method - capture both return values
	subscription, err := x.sdk.Subscribe(ctx, subscriptionReq)
	if err != nil {
		return fmt.Errorf("failed to subscribe to KMP measurements: %w", err)
	}

	// Store subscription

	x.mu.Lock()
	x.subscriptions[subscription.SubscriptionID] = subscription
	x.mu.Unlock()

	log.Printf("Subscribed to KMP measurements with ID: %s", subscription.SubscriptionID)

	return nil
}

// processAnalytics processes incoming KPI data and generates insights.

func (x *KMPAnalyticsXApp) processAnalytics() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			x.generateInsights()
		case <-x.ctx.Done():
			return
		}
	}
}

// generateInsights analyzes collected KPI data and generates insights.

func (x *KMPAnalyticsXApp) generateInsights() {
	x.mu.RLock()
	defer x.mu.RUnlock()

	for nodeID, measurements := range x.kpiData {
		if len(measurements) == 0 {
			continue
		}

		// Calculate average metrics for the node

		avgMetrics := make(map[string]float64)
		for _, measurement := range measurements {
			for metric, value := range measurement.Metrics {
				avgMetrics[metric] += value
			}
		}

		// Normalize averages

		for metric := range avgMetrics {
			avgMetrics[metric] /= float64(len(measurements))
		}

		// Check alert rules

		x.checkAlerts(nodeID, avgMetrics)

		log.Printf("Node %s average metrics: %+v", nodeID, avgMetrics)
	}
}

// checkAlerts checks if any alert rules are triggered.

func (x *KMPAnalyticsXApp) checkAlerts(nodeID string, metrics map[string]float64) {
	for _, rule := range x.alertRules {
		// Simplified alert logic

		if rule.Name == "high_throughput" {
			if throughput, exists := metrics["throughput"]; exists && throughput > rule.Threshold {
				x.triggerAlert(nodeID, rule, throughput)
			}
		}

		if rule.Name == "high_latency" {
			if latency, exists := metrics["latency"]; exists && latency > rule.Threshold {
				x.triggerAlert(nodeID, rule, latency)
			}
		}
	}
}

// triggerAlert triggers an alert based on the rule.

func (x *KMPAnalyticsXApp) triggerAlert(nodeID string, rule AlertRule, value float64) {
	switch rule.Action {
	case "log":
		log.Printf("ALERT [%s]: Node %s, %s = %.2f (threshold: %.2f)", 
			rule.Name, nodeID, rule.Condition, value, rule.Threshold)
	case "alert":
		log.Printf("CRITICAL ALERT [%s]: Node %s, %s = %.2f (threshold: %.2f)", 
			rule.Name, nodeID, rule.Condition, value, rule.Threshold)
		// In a real implementation, this would send alerts to external systems
	}
}

// Stop stops the xApp and cleans up resources.

func (x *KMPAnalyticsXApp) Stop() {
	log.Println("Stopping KMP Analytics xApp...")

	x.cancel()

	// Unsubscribe from all subscriptions using subscription ID strings
	x.mu.Lock()
	for id := range x.subscriptions {
		if err := x.sdk.Unsubscribe(context.Background(), id); err != nil {
			log.Printf("Error unsubscribing from %s: %v", id, err)
		}
	}
	x.subscriptions = make(map[string]*e2.XAppSubscription)
	x.mu.Unlock()

	log.Println("KMP Analytics xApp stopped")
}

// ProcessIndication handles incoming E2 indications.

func (x *KMPAnalyticsXApp) ProcessIndication(indication *e2.E2Indication) {
	// Parse KMP measurements from indication

	var measurement KPIMeasurement
	if err := json.Unmarshal(indication.IndicationMessage, &measurement); err != nil {
		log.Printf("Failed to parse KMP measurement: %v", err)
		return
	}

	// Store measurement - use subscription ID to derive node ID
	nodeID := indication.SubscriptionID // Using subscription ID as a proxy for node ID

	x.mu.Lock()
	x.kpiData[nodeID] = append(x.kpiData[nodeID], measurement)
	
	// Keep only recent measurements (last 100)
	if len(x.kpiData[nodeID]) > 100 {
		x.kpiData[nodeID] = x.kpiData[nodeID][1:]
	}
	x.mu.Unlock()

	log.Printf("Received KMP measurement from subscription %s: %+v", indication.SubscriptionID, measurement.Metrics)
}

// GetStatus returns the current status of the xApp.

func (x *KMPAnalyticsXApp) GetStatus() map[string]interface{} {
	x.mu.RLock()
	defer x.mu.RUnlock()

	status := map[string]interface{}{
		"xapp_name":           x.config.XAppName,
		"subscription_count":  len(x.subscriptions),
		"monitored_nodes":     len(x.kpiData),
		"alert_rules":         len(x.alertRules),
	}

	// Add subscription details

	subscriptions := make([]map[string]interface{}, 0, len(x.subscriptions))
	for id, sub := range x.subscriptions {
		subscriptions = append(subscriptions, map[string]interface{}{
			"id":     id,
			"status": sub.Status,
		})
	}
	status["subscriptions"] = subscriptions

	return status
}