package o1

import (
	"context"
	"encoding/xml"
	"fmt"
	"strconv"
	"strings"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

// parseAlarmData parses NETCONF XML alarm data into Alarm structs
func (a *O1Adaptor) parseAlarmData(xmlData string, managedElementID string) ([]*Alarm, error) {
	if xmlData == "" {
		return []*Alarm{}, nil
	}

	// Define struct for parsing O-RAN alarm XML
	type ORANAlarm struct {
		FaultID     uint16    `xml:"fault-id"`
		FaultSource string    `xml:"fault-source"`
		Severity    string    `xml:"fault-severity"`
		IsCleared   bool      `xml:"is-cleared"`
		FaultText   string    `xml:"fault-text"`
		EventTime   time.Time `xml:"event-time"`
	}

	type ActiveAlarmList struct {
		Alarms []ORANAlarm `xml:"active-alarms"`
	}

	type AlarmData struct {
		XMLName         xml.Name        `xml:"data"`
		ActiveAlarmList ActiveAlarmList `xml:"active-alarm-list"`
	}

	var alarmData AlarmData
	if err := xml.Unmarshal([]byte(xmlData), &alarmData); err != nil {
		// If parsing as O-RAN format fails, try generic parsing
		return a.parseGenericAlarmData(xmlData, managedElementID)
	}

	// Convert O-RAN alarms to our Alarm format
	alarms := make([]*Alarm, 0, len(alarmData.ActiveAlarmList.Alarms))
	for _, oranAlarm := range alarmData.ActiveAlarmList.Alarms {
		if oranAlarm.IsCleared {
			continue // Skip cleared alarms
		}

		alarm := &Alarm{
			ID:               fmt.Sprintf("%d-%s", oranAlarm.FaultID, oranAlarm.FaultSource),
			ManagedElementID: managedElementID,
			Severity:         strings.ToUpper(oranAlarm.Severity),
			Type:             "EQUIPMENT", // Default type for O-RAN alarms
			ProbableCause:    oranAlarm.FaultSource,
			SpecificProblem:  oranAlarm.FaultText,
			TimeRaised:       oranAlarm.EventTime,
		}

		// Map O-RAN severity to standard alarm severity
		switch strings.ToLower(oranAlarm.Severity) {
		case "critical":
			alarm.Severity = "CRITICAL"
		case "major":
			alarm.Severity = "MAJOR"
		case "minor":
			alarm.Severity = "MINOR"
		case "warning":
			alarm.Severity = "WARNING"
		default:
			alarm.Severity = "MINOR"
		}

		alarms = append(alarms, alarm)
	}

	return alarms, nil
}

// parseGenericAlarmData parses generic XML alarm data
func (a *O1Adaptor) parseGenericAlarmData(xmlData string, managedElementID string) ([]*Alarm, error) {
	// Simple generic parsing for demonstration
	// In production, this would handle various vendor-specific formats

	alarms := []*Alarm{}

	// Check if data contains alarm indicators
	if strings.Contains(xmlData, "alarm") || strings.Contains(xmlData, "fault") {
		// Create a generic alarm entry
		alarm := &Alarm{
			ID:               fmt.Sprintf("generic-%d", time.Now().Unix()),
			ManagedElementID: managedElementID,
			Severity:         "MINOR",
			Type:             "COMMUNICATIONS",
			ProbableCause:    "UNKNOWN",
			SpecificProblem:  "Generic alarm parsed from NETCONF response",
			TimeRaised:       time.Now(),
			AdditionalInfo:   "Parsed from XML: " + xmlData[:min(100, len(xmlData))],
		}
		alarms = append(alarms, alarm)
	}

	return alarms, nil
}

// convertEventToAlarm converts a NETCONF event to an Alarm
func (a *O1Adaptor) convertEventToAlarm(event *NetconfEvent, managedElementID string) *Alarm {
	if event == nil {
		return nil
	}

	// Extract alarm information from event data
	alarm := &Alarm{
		ID:               fmt.Sprintf("event-%d", time.Now().UnixNano()),
		ManagedElementID: managedElementID,
		Severity:         "MINOR",
		Type:             "COMMUNICATIONS",
		ProbableCause:    "EVENT_NOTIFICATION",
		SpecificProblem:  "Alarm notification received",
		TimeRaised:       event.Timestamp,
	}

	// Extract more specific information from event data
	if eventType, exists := event.Data["event_type"]; exists {
		if eventTypeStr, ok := eventType.(string); ok {
			alarm.Type = strings.ToUpper(eventTypeStr)
		}
	}

	if severity, exists := event.Data["severity"]; exists {
		if severityStr, ok := severity.(string); ok {
			alarm.Severity = strings.ToUpper(severityStr)
		}
	}

	if description, exists := event.Data["description"]; exists {
		if descStr, ok := description.(string); ok {
			alarm.SpecificProblem = descStr
		}
	}

	// Parse XML content for more details if available
	if event.XML != "" {
		if parsedAlarms, err := a.parseAlarmData(event.XML, managedElementID); err == nil && len(parsedAlarms) > 0 {
			// Use the first parsed alarm if available
			return parsedAlarms[0]
		}
	}

	return alarm
}

// Enhanced metric collection with real NETCONF integration
func (a *O1Adaptor) collectMetricsFromDevice(ctx context.Context, clientID string, metricNames []string) (map[string]interface{}, error) {
	logger := log.FromContext(ctx)

	a.clientsMux.RLock()
	client, exists := a.clients[clientID]
	a.clientsMux.RUnlock()

	if !exists || !client.IsConnected() {
		return nil, fmt.Errorf("no active client found or client not connected")
	}

	metrics := make(map[string]interface{})

	// Query different metric types using appropriate NETCONF filters
	for _, metricName := range metricNames {
		var filter string
		var defaultValue interface{}

		switch metricName {
		case "cpu_usage":
			filter = "/o-ran-hardware:hardware/component[class='cpu']/state/cpu-usage"
			defaultValue = 0.0
		case "memory_usage":
			filter = "/o-ran-hardware:hardware/component[class='memory']/state/memory-usage"
			defaultValue = 0.0
		case "temperature":
			filter = "/o-ran-hardware:hardware/component/state/temperature"
			defaultValue = 25.0
		case "throughput_mbps":
			filter = "/ietf-interfaces:interfaces/interface/statistics/out-octets"
			defaultValue = 0.0
		case "packet_loss_rate":
			filter = "/ietf-interfaces:interfaces/interface/statistics/in-errors"
			defaultValue = 0.0
		case "power_consumption":
			filter = "/o-ran-hardware:hardware/component[class='power']/state/power-consumption"
			defaultValue = 0.0
		default:
			// Unknown metric, use default
			metrics[metricName] = 0
			continue
		}

		// Query the metric using NETCONF
		configData, err := client.GetConfig(filter)
		if err != nil {
			logger.Info("failed to query metric via NETCONF, using default",
				"metric", metricName, "error", err)
			metrics[metricName] = defaultValue
			continue
		}

		// Parse the response and extract metric value
		value, err := a.parseMetricValue(configData.XMLData, metricName)
		if err != nil {
			logger.Info("failed to parse metric value, using default",
				"metric", metricName, "error", err)
			metrics[metricName] = defaultValue
		} else {
			metrics[metricName] = value
		}
	}

	return metrics, nil
}

// parseMetricValue parses a metric value from NETCONF XML response
func (a *O1Adaptor) parseMetricValue(xmlData string, metricName string) (interface{}, error) {
	if xmlData == "" {
		return 0, fmt.Errorf("empty XML data")
	}

	// Simple XML parsing for metric values
	// In production, this would use proper XML parsing with schemas

	// Look for numeric values in the XML
	lines := strings.Split(xmlData, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.Contains(line, ">") && strings.Contains(line, "<") {
			// Extract content between XML tags
			start := strings.Index(line, ">")
			end := strings.LastIndex(line, "<")
			if start >= 0 && end > start {
				content := line[start+1 : end]

				// Try to parse as float
				if value, err := strconv.ParseFloat(content, 64); err == nil {
					return value, nil
				}

				// Try to parse as int
				if value, err := strconv.ParseInt(content, 10, 64); err == nil {
					return value, nil
				}

				// Return as string if not numeric
				if content != "" {
					return content, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("no value found in XML data")
}

// startPeriodicMetricCollection starts a goroutine for periodic metric collection
func (a *O1Adaptor) startPeriodicMetricCollection(ctx context.Context, collector *MetricCollector) {
	logger := log.FromContext(ctx)
	logger.Info("starting periodic metric collection",
		"collectorID", collector.ID,
		"element", collector.ManagedElement,
		"period", collector.CollectionPeriod)

	// Create a cancellable context for this collection
	collectionCtx, cancel := context.WithCancel(ctx)
	collector.cancel = cancel

	go func() {
		defer cancel()

		ticker := time.NewTicker(collector.CollectionPeriod)
		defer ticker.Stop()

		for {
			select {
			case <-collectionCtx.Done():
				logger.Info("stopping metric collection", "collectorID", collector.ID)
				return
			case <-ticker.C:
				// Collect metrics
				clientID := collector.ManagedElement // Assuming clientID format
				metrics, err := a.collectMetricsFromDevice(collectionCtx, clientID, collector.MetricNames)
				if err != nil {
					logger.Error(err, "failed to collect metrics", "collectorID", collector.ID)
					continue
				}

				// Update last collection time
				a.metricsMux.Lock()
				if c, exists := a.metricCollectors[collector.ID]; exists {
					c.LastCollection = time.Now()
				}
				a.metricsMux.Unlock()

				// Log collected metrics (in production, you might want to export to monitoring system)
				logger.Info("collected metrics",
					"collectorID", collector.ID,
					"metrics", metrics,
					"timestamp", time.Now())
			}
		}
	}()
}

// validateNetworkElement checks if the network element specification is valid
func (a *O1Adaptor) validateNetworkElement(me interface{}) error {
	// This would typically validate against the ManagedElement CRD schema
	// For now, perform basic validation

	if me == nil {
		return fmt.Errorf("managed element cannot be nil")
	}

	// Additional validation logic would go here
	return nil
}

// buildSecurityConfiguration builds security configuration XML
func (a *O1Adaptor) buildSecurityConfiguration(policy *SecurityPolicy) string {
	if policy == nil {
		return ""
	}

	var xmlBuilder strings.Builder
	xmlBuilder.WriteString("<security-configuration>\n")
	xmlBuilder.WriteString(fmt.Sprintf("  <policy-id>%s</policy-id>\n", policy.PolicyID))
	xmlBuilder.WriteString(fmt.Sprintf("  <policy-type>%s</policy-type>\n", policy.PolicyType))
	xmlBuilder.WriteString(fmt.Sprintf("  <enforcement>%s</enforcement>\n", policy.Enforcement))

	if len(policy.Rules) > 0 {
		xmlBuilder.WriteString("  <rules>\n")
		for _, rule := range policy.Rules {
			xmlBuilder.WriteString("    <rule>\n")
			xmlBuilder.WriteString(fmt.Sprintf("      <rule-id>%s</rule-id>\n", rule.RuleID))
			xmlBuilder.WriteString(fmt.Sprintf("      <action>%s</action>\n", rule.Action))
			xmlBuilder.WriteString("    </rule>\n")
		}
		xmlBuilder.WriteString("  </rules>\n")
	}

	xmlBuilder.WriteString("</security-configuration>\n")
	return xmlBuilder.String()
}

// Helper function to find minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
