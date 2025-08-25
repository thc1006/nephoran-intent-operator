package o1

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/oran"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestNewO1Adaptor(t *testing.T) {
	tests := []struct {
		name   string
		config *oran.O1Config
		want   *O1Adaptor
	}{
		{
			name:   "with nil config",
			config: nil,
			want: &O1Adaptor{
				clients:          make(map[string]*NetconfClient),
				yangRegistry:     NewYANGModelRegistry(),
				subscriptions:    make(map[string][]EventCallback),
				metricCollectors: make(map[string]*MetricCollector),
			},
		},
		{
			name: "with custom config",
			config: &oran.O1Config{
				Endpoint:      "localhost:830",
				Timeout:       30 * time.Second,
				RetryAttempts: 5,
			},
			want: &O1Adaptor{
				clients:          make(map[string]*NetconfClient),
				yangRegistry:     NewYANGModelRegistry(),
				subscriptions:    make(map[string][]EventCallback),
				metricCollectors: make(map[string]*MetricCollector),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewO1Adaptor(tt.config, nil)
			assert.NotNil(t, got)
			assert.NotNil(t, got.clients)
			assert.NotNil(t, got.yangRegistry)
			assert.NotNil(t, got.subscriptions)
			assert.NotNil(t, got.metricCollectors)

			if tt.config != nil {
				assert.Equal(t, tt.config, got.config)
			} else {
				// Check default config values
				assert.Equal(t, "localhost:830", got.config.Endpoint)
				assert.Equal(t, 30*time.Second, got.config.Timeout)
				assert.Equal(t, 3, got.config.RetryAttempts)
			}
		})
	}
}

func TestO1Adaptor_IsConnected(t *testing.T) {
	adaptor := NewO1Adaptor(nil, nil)

	me := &nephoranv1.ManagedElement{
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-element",
		},
		Spec: nephoranv1.ManagedElementSpec{
			Host: "192.168.1.100",
			Port: 830,
		},
	}

	// Test when not connected
	connected := adaptor.IsConnected(me)
	assert.False(t, connected)
}

func TestO1Adaptor_ValidateConfiguration(t *testing.T) {
	adaptor := NewO1Adaptor(nil, nil)
	ctx := context.Background()

	tests := []struct {
		name    string
		config  string
		wantErr bool
	}{
		{
			name:    "empty config",
			config:  "",
			wantErr: true,
		},
		{
			name: "valid XML config",
			config: `<hardware>
				<component>
					<name>test</name>
					<class>cpu</class>
				</component>
			</hardware>`,
			wantErr: false,
		},
		{
			name:    "valid JSON config",
			config:  `{"hardware": {"component": {"name": "test", "class": "cpu"}}}`,
			wantErr: false,
		},
		{
			name:    "invalid config",
			config:  "invalid xml/json",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := adaptor.ValidateConfiguration(ctx, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestO1Adaptor_parseAlarmData(t *testing.T) {
	adaptor := NewO1Adaptor(nil, nil)

	tests := []struct {
		name               string
		xmlData            string
		managedElementID   string
		expectedAlarmCount int
		wantErr            bool
	}{
		{
			name:               "empty XML data",
			xmlData:            "",
			managedElementID:   "test-element",
			expectedAlarmCount: 0,
			wantErr:            false,
		},
		{
			name: "valid O-RAN alarm data",
			xmlData: `<data>
				<active-alarm-list>
					<active-alarms>
						<fault-id>123</fault-id>
						<fault-source>power-supply-1</fault-source>
						<fault-severity>major</fault-severity>
						<is-cleared>false</is-cleared>
						<fault-text>Power supply failure</fault-text>
						<event-time>2024-01-15T10:30:00Z</event-time>
					</active-alarms>
				</active-alarm-list>
			</data>`,
			managedElementID:   "test-element",
			expectedAlarmCount: 1,
			wantErr:            false,
		},
		{
			name: "cleared alarm should be filtered",
			xmlData: `<data>
				<active-alarm-list>
					<active-alarms>
						<fault-id>123</fault-id>
						<fault-source>power-supply-1</fault-source>
						<fault-severity>major</fault-severity>
						<is-cleared>true</is-cleared>
						<fault-text>Power supply failure</fault-text>
						<event-time>2024-01-15T10:30:00Z</event-time>
					</active-alarms>
				</active-alarm-list>
			</data>`,
			managedElementID:   "test-element",
			expectedAlarmCount: 0,
			wantErr:            false,
		},
		{
			name:               "generic alarm data",
			xmlData:            "<alarm><type>fault</type><description>Generic alarm</description></alarm>",
			managedElementID:   "test-element",
			expectedAlarmCount: 1,
			wantErr:            false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alarms, err := adaptor.parseAlarmData(tt.xmlData, tt.managedElementID)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Len(t, alarms, tt.expectedAlarmCount)

				// Verify alarm properties if alarms exist
				for _, alarm := range alarms {
					assert.Equal(t, tt.managedElementID, alarm.ManagedElementID)
					assert.NotEmpty(t, alarm.ID)
					assert.NotEmpty(t, alarm.Severity)
				}
			}
		})
	}
}

func TestO1Adaptor_convertEventToAlarm(t *testing.T) {
	adaptor := NewO1Adaptor(nil, nil)

	tests := []struct {
		name             string
		event            *NetconfEvent
		managedElementID string
		expectedAlarm    bool
	}{
		{
			name:             "nil event",
			event:            nil,
			managedElementID: "test-element",
			expectedAlarm:    false,
		},
		{
			name: "valid event",
			event: &NetconfEvent{
				Type:      "notification",
				Timestamp: time.Now(),
				Source:    "test-source",
				Data: map[string]interface{}{
					"event_type":  "alarm",
					"severity":    "major",
					"description": "Test alarm event",
				},
			},
			managedElementID: "test-element",
			expectedAlarm:    true,
		},
		{
			name: "event with XML data",
			event: &NetconfEvent{
				Type:      "notification",
				Timestamp: time.Now(),
				Source:    "test-source",
				XML:       "<alarm><severity>critical</severity></alarm>",
				Data:      make(map[string]interface{}),
			},
			managedElementID: "test-element",
			expectedAlarm:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			alarm := adaptor.convertEventToAlarm(tt.event, tt.managedElementID)

			if tt.expectedAlarm {
				assert.NotNil(t, alarm)
				assert.Equal(t, tt.managedElementID, alarm.ManagedElementID)
				assert.NotEmpty(t, alarm.ID)
			} else {
				assert.Nil(t, alarm)
			}
		})
	}
}

func TestO1Adaptor_parseMetricValue(t *testing.T) {
	adaptor := NewO1Adaptor(nil, nil)

	tests := []struct {
		name       string
		xmlData    string
		metricName string
		expected   interface{}
		wantErr    bool
	}{
		{
			name:       "empty XML data",
			xmlData:    "",
			metricName: "cpu_usage",
			expected:   0,
			wantErr:    true,
		},
		{
			name:       "XML with float value",
			xmlData:    "<cpu-usage>45.6</cpu-usage>",
			metricName: "cpu_usage",
			expected:   45.6,
			wantErr:    false,
		},
		{
			name:       "XML with integer value",
			xmlData:    "<packet-count>1024</packet-count>",
			metricName: "packet_count",
			expected:   int64(1024),
			wantErr:    false,
		},
		{
			name:       "XML with string value",
			xmlData:    "<status>active</status>",
			metricName: "status",
			expected:   "active",
			wantErr:    false,
		},
		{
			name:       "XML without value",
			xmlData:    "<root><empty></empty></root>",
			metricName: "test",
			expected:   0,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := adaptor.parseMetricValue(tt.xmlData, tt.metricName)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, value)
			}
		})
	}
}

func TestO1Adaptor_buildSecurityConfiguration(t *testing.T) {
	adaptor := NewO1Adaptor(nil, nil)

	tests := []struct {
		name     string
		policy   *SecurityPolicy
		expected string
	}{
		{
			name:     "nil policy",
			policy:   nil,
			expected: "",
		},
		{
			name: "policy with rules",
			policy: &SecurityPolicy{
				PolicyID:    "policy-123",
				PolicyType:  "access-control",
				Enforcement: "strict",
				Rules: []SecurityRule{
					{
						RuleID: "rule-001",
						Action: "ALLOW",
					},
				},
			},
			expected: "<security-configuration>",
		},
		{
			name: "policy without rules",
			policy: &SecurityPolicy{
				PolicyID:    "policy-456",
				PolicyType:  "firewall",
				Enforcement: "permissive",
				Rules:       []SecurityRule{},
			},
			expected: "<security-configuration>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := adaptor.buildSecurityConfiguration(tt.policy)

			if tt.policy == nil {
				assert.Empty(t, result)
			} else {
				assert.Contains(t, result, tt.expected)
				assert.Contains(t, result, tt.policy.PolicyID)
				assert.Contains(t, result, tt.policy.PolicyType)
				assert.Contains(t, result, tt.policy.Enforcement)
			}
		})
	}
}

func TestMetricCollector(t *testing.T) {
	collector := &MetricCollector{
		ID:               "test-collector",
		ManagedElement:   "test-element",
		MetricNames:      []string{"cpu_usage", "memory_usage"},
		CollectionPeriod: 30 * time.Second,
		ReportingPeriod:  60 * time.Second,
		Active:           true,
		LastCollection:   time.Now(),
	}

	assert.Equal(t, "test-collector", collector.ID)
	assert.Equal(t, "test-element", collector.ManagedElement)
	assert.Len(t, collector.MetricNames, 2)
	assert.True(t, collector.Active)
}

func TestAlarmStruct(t *testing.T) {
	alarm := &Alarm{
		ID:               "alarm-123",
		ManagedElementID: "test-element",
		Severity:         "MAJOR",
		Type:             "EQUIPMENT",
		ProbableCause:    "POWER_SUPPLY_FAILURE",
		SpecificProblem:  "Power supply redundancy lost",
		AdditionalInfo:   "Check power supply unit 2",
		TimeRaised:       time.Now(),
	}

	assert.Equal(t, "alarm-123", alarm.ID)
	assert.Equal(t, "test-element", alarm.ManagedElementID)
	assert.Equal(t, "MAJOR", alarm.Severity)
	assert.Equal(t, "EQUIPMENT", alarm.Type)
	assert.NotZero(t, alarm.TimeRaised)
}

// Benchmark tests for performance verification
func BenchmarkO1Adaptor_parseAlarmData(b *testing.B) {
	adaptor := NewO1Adaptor(nil, nil)
	xmlData := `<data>
		<active-alarm-list>
			<active-alarms>
				<fault-id>123</fault-id>
				<fault-source>power-supply-1</fault-source>
				<fault-severity>major</fault-severity>
				<is-cleared>false</is-cleared>
				<fault-text>Power supply failure</fault-text>
				<event-time>2024-01-15T10:30:00Z</event-time>
			</active-alarms>
		</active-alarm-list>
	</data>`

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = adaptor.parseAlarmData(xmlData, "test-element")
	}
}

func BenchmarkO1Adaptor_parseMetricValue(b *testing.B) {
	adaptor := NewO1Adaptor(nil, nil)
	xmlData := "<cpu-usage>45.6</cpu-usage>"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = adaptor.parseMetricValue(xmlData, "cpu_usage")
	}
}
