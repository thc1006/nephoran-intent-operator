package alerting

import (
	"context"
	"strings"
)

// Missing AlertRouter method implementations

func (ar *AlertRouter) convertPriorityToInt(priority string) int {
	switch priority {
	case "critical", "high":
		return 90
	case "medium":
		return 50
	case "low":
		return 10
	default:
		return 30
	}
}

func (ar *AlertRouter) isMoreSevere(severity1, severity2 string) bool {
	severityMap := map[string]int{
		"critical": 4,
		"high":     3,
		"medium":   2,
		"low":      1,
	}
	return severityMap[severity1] > severityMap[severity2]
}

func (ar *AlertRouter) evaluateRuleConditions(alert *SLAAlert, conditions []RoutingCondition) bool {
	for _, condition := range conditions {
		if !ar.evaluateSingleCondition(alert, condition) {
			return false
		}
	}
	return true
}

func (ar *AlertRouter) evaluateSingleCondition(alert *SLAAlert, condition RoutingCondition) bool {
	var fieldValue string
	switch condition.Field {
	case "severity":
		fieldValue = string(alert.Severity)
	case "sla_type":
		fieldValue = string(alert.SLAType)
	case "component":
		fieldValue = alert.Context.Component
	case "service":
		fieldValue = alert.Context.Service
	default:
		if value, exists := alert.Labels[condition.Field]; exists {
			fieldValue = value
		}
	}

	result := ar.matchesCondition(fieldValue, condition)
	if condition.Negate {
		return !result
	}
	return result
}

func (ar *AlertRouter) matchesCondition(value string, condition RoutingCondition) bool {
	switch condition.Operator {
	case "equals":
		for _, condValue := range condition.Values {
			if value == condValue {
				return true
			}
		}
	case "contains":
		for _, condValue := range condition.Values {
			if strings.Contains(value, condValue) {
				return true
			}
		}
	case "matches":
		// Regex matching would go here
		return false
	default:
		return false
	}
	return false
}

func (ar *AlertRouter) isChannelAvailable(channelName string, alert *SLAAlert) bool {
	channel, exists := ar.notificationChannels[channelName]
	if !exists || !channel.Enabled {
		return false
	}
	return true
}

func (ar *AlertRouter) passesChannelFilters(alert *EnrichedAlert, filters []AlertFilter) bool {
	if len(filters) == 0 {
		return true
	}
	
	for _, filter := range filters {
		if !ar.passesFilter(alert.SLAAlert, filter) {
			return false
		}
	}
	return true
}

func (ar *AlertRouter) passesFilter(alert *SLAAlert, filter AlertFilter) bool {
	var fieldValue string
	switch filter.Field {
	case "severity":
		fieldValue = string(alert.Severity)
	case "component":
		fieldValue = alert.Context.Component
	case "sla_type":
		fieldValue = string(alert.SLAType)
	default:
		if value, exists := alert.Labels[filter.Field]; exists {
			fieldValue = value
		}
	}

	result := ar.matchesFilterCondition(fieldValue, filter)
	if filter.Negate {
		return !result
	}
	return result
}

func (ar *AlertRouter) matchesFilterCondition(value string, filter AlertFilter) bool {
	switch filter.Operator {
	case "equals":
		return value == filter.Value
	case "contains":
		return strings.Contains(value, filter.Value)
	case "matches":
		// Regex matching would go here
		return false
	default:
		return false
	}
}

func (ar *AlertRouter) checkRateLimit(channelName string, rateLimit RateLimit) bool {
	// Implementation would track rate limiting per channel
	// For now, always return true (no rate limiting)
	return true
}

// Notification method stubs
func (ar *AlertRouter) sendSlackNotification(ctx context.Context, alert *EnrichedAlert, channel NotificationChannel) error {
	// TODO: Implement Slack notification
	return nil
}

func (ar *AlertRouter) sendEmailNotification(ctx context.Context, alert *EnrichedAlert, channel NotificationChannel) error {
	// TODO: Implement email notification
	return nil
}

func (ar *AlertRouter) sendWebhookNotification(ctx context.Context, alert *EnrichedAlert, channel NotificationChannel) error {
	// TODO: Implement webhook notification
	return nil
}

func (ar *AlertRouter) sendPagerDutyNotification(ctx context.Context, alert *EnrichedAlert, channel NotificationChannel) error {
	// TODO: Implement PagerDuty notification
	return nil
}

func (ar *AlertRouter) sendTeamsNotification(ctx context.Context, alert *EnrichedAlert, channel NotificationChannel) error {
	// TODO: Implement Teams notification
	return nil
}