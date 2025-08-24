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

// NotificationTemplate defines notification formatting for fault management
type NotificationTemplate struct {
	ID        string            `json:"id"`
	Name      string            `json:"name"`
	Subject   string            `json:"subject"`
	Body      string            `json:"body"`
	Format    string            `json:"format"` // HTML, TEXT, JSON
	Variables map[string]string `json:"variables"`
	Enabled   bool              `json:"enabled"`
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
		ID:        alarm.ID,
		Type:      "ALARM",
		Title:     template.Subject,
		Message:   f.formatMessage(alarm, template),
		Severity:  alarm.Severity,
		Source:    "FAULT_MANAGER",
		Timestamp: alarm.TimeRaised,
		Tags:      []string{alarm.Type, alarm.ProbableCause},
		Metadata: map[string]interface{}{
			"managed_element": alarm.ManagedElementID,
			"probable_cause":  alarm.ProbableCause,
			"additional_info": alarm.AdditionalInfo,
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
		"{{.AlarmID}}":          alarm.ID,
		"{{.Severity}}":         alarm.Severity,
		"{{.Type}}":             alarm.Type,
		"{{.ProbableCause}}":    alarm.ProbableCause,
		"{{.SpecificProblem}}":  alarm.SpecificProblem,
		"{{.ManagedElement}}":   alarm.ManagedElementID,
		"{{.TimeRaised}}":       alarm.TimeRaised.Format(time.RFC3339),
		"{{.AdditionalInfo}}":   alarm.AdditionalInfo,
	}
	
	for placeholder, value := range replacements {
		message = strings.Replace(message, placeholder, value, -1)
	}
	
	return message
}