package o1

import (
	"context"
	"strings"
	"time"
)

// Fault-specific types to avoid conflicts with common types

// FaultNotificationChannel extends NotificationChannel for fault-specific operations
type FaultNotificationChannel interface {
	NotificationChannel // Embed common interface
	SendAlarmNotification(ctx context.Context, alarm *EnhancedAlarm, template *NotificationTemplate) error
	GetChannelType() string
}

// FaultNotificationChannelImpl implements FaultNotificationChannel
type FaultNotificationChannelImpl struct {
	*DefaultNotificationChannel // Embed common implementation
}

func NewFaultNotificationChannel(id, chType string, config map[string]interface{}) *FaultNotificationChannelImpl {
	return &FaultNotificationChannelImpl{
		DefaultNotificationChannel: NewDefaultNotificationChannel(id, chType, config),
	}
}

func (f *FaultNotificationChannelImpl) SendAlarmNotification(ctx context.Context, alarm *EnhancedAlarm, template *NotificationTemplate) error {
	// Convert alarm to generic notification
	notification := &Notification{
		ID:        alarm.AlarmID,
		Type:      "ALARM",
		Severity:  alarm.PerceivedSeverity,
		Title:     template.Subject,
		Message:   f.formatMessage(alarm, template),
		Source:    "FAULT_MANAGER",
		Target:    alarm.ObjectInstance,
		Timestamp: func() time.Time {
			if alarm.AlarmRaisedTime != nil {
				return *alarm.AlarmRaisedTime
			}
			return time.Now()
		}(),
		AckRequired: true,
		Metadata: map[string]interface{}{
			"event_type":      alarm.EventType,
			"probable_cause":  alarm.ProbableCause,
			"managed_element": alarm.ObjectInstance,
			"additional_info": alarm.AdditionalText,
		},
	}

	return f.SendNotification(ctx, notification)
}

func (f *FaultNotificationChannelImpl) GetChannelType() string {
	return f.GetType()
}

func (f *FaultNotificationChannelImpl) formatMessage(alarm *EnhancedAlarm, template *NotificationTemplate) string {
	// Simple template formatting - in production would use proper templating
	message := template.Body

	// Replace common variables
	replacements := map[string]string{
		"{{.AlarmID}}":         alarm.AlarmID,
		"{{.Severity}}":        alarm.PerceivedSeverity,
		"{{.Type}}":            alarm.EventType,
		"{{.ProbableCause}}":   alarm.ProbableCause,
		"{{.SpecificProblem}}": alarm.AdditionalText,
		"{{.ManagedElement}}":  alarm.ObjectInstance,
		"{{.TimeRaised}}":      alarm.AlarmRaisedTime.Format(time.RFC3339),
		"{{.AdditionalInfo}}":  alarm.AdditionalText,
	}

	for placeholder, value := range replacements {
		message = strings.Replace(message, placeholder, value, -1)
	}

	return message
}
