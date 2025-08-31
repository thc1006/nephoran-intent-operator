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
		NotificationID:   alarm.AlarmID,
		NotificationType: "ALARM",
		EventTime:        alarm.AlarmRaisedTime,
		SystemDN:         alarm.ManagedObjectID,
		NotificationData: map[string]interface{}{
			"title":           template.Subject,
			"message":         f.formatMessage(alarm, template),
			"severity":        alarm.PerceivedSeverity,
			"source":          "FAULT_MANAGER",
			"alarm_type":      alarm.AlarmType,
			"probable_cause":  alarm.ProbableCause,
			"managed_element": alarm.ManagedObjectID,
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
		"{{.Type}}":            alarm.AlarmType,
		"{{.ProbableCause}}":   alarm.ProbableCause,
		"{{.SpecificProblem}}": alarm.AdditionalText,
		"{{.ManagedElement}}":  alarm.ManagedObjectID,
		"{{.TimeRaised}}":      alarm.AlarmRaisedTime.Format(time.RFC3339),
		"{{.AdditionalInfo}}":  alarm.AdditionalText,
	}

	for placeholder, value := range replacements {
		message = strings.Replace(message, placeholder, value, -1)
	}

	return message
}
