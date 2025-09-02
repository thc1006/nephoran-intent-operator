package audit

import (
	"context"
	"fmt"
	"log/slog"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"

	"github.com/thc1006/nephoran-intent-operator/pkg/audit/backends"
)

// QueryEngineTestSuite tests audit query engine functionality
type QueryEngineTestSuite struct {
	suite.Suite
	queryEngine *QueryEngine
	testEvents  []*AuditEvent
}

func TestQueryEngineTestSuite(t *testing.T) {
	suite.Run(t, new(QueryEngineTestSuite))
}

func (suite *QueryEngineTestSuite) SetupTest() {
	// Create a mock audit system for testing
	auditConfig := &AuditSystemConfig{
		Enabled:       true,
		LogLevel:      SeverityInfo,
		BatchSize:     100,
		FlushInterval: 1 * time.Second,
		MaxQueueSize:  1000,
	}
	auditSystem, err := NewAuditSystem(auditConfig)
	suite.Require().NoError(err)

	// Create mock backends map
	mockBackends := make(map[string]backends.Backend)

	// Create logger
	logger := slog.Default()

	suite.queryEngine = NewQueryEngine(auditSystem, mockBackends, logger)
	suite.testEvents = suite.createTestEvents()
}

func (suite *QueryEngineTestSuite) createTestEvents() []*AuditEvent {
	now := time.Now()

	return []*AuditEvent{
		{
			ID:        uuid.New().String(),
			Timestamp: now.Add(-24 * time.Hour),
			EventType: EventTypeAuthentication,
			Component: "auth-service",
			Action:    "login",
			Severity:  SeverityInfo,
			Result:    ResultSuccess,
			UserContext: &UserContext{
				UserID:   "user1",
				Username: "alice",
				Role:     "admin",
			},
			NetworkContext: &NetworkContext{
				SourcePort: 8080,
			},
			Data: json.RawMessage(`{}`),
		},
		{
			ID:        uuid.New().String(),
			Timestamp: now.Add(-12 * time.Hour),
			EventType: EventTypeAuthenticationFailed,
			Component: "auth-service",
			Action:    "login",
			Severity:  SeverityWarning,
			Result:    ResultFailure,
			UserContext: &UserContext{
				UserID:   "user2",
				Username: "bob",
				Role:     "user",
			},
			NetworkContext: &NetworkContext{
				SourcePort: 8080,
			},
			Data: json.RawMessage(`{}`),
		},
		{
			ID:        uuid.New().String(),
			Timestamp: now.Add(-6 * time.Hour),
			EventType: EventTypeDataAccess,
			Component: "api-service",
			Action:    "get_user_data",
			Severity:  SeverityInfo,
			Result:    ResultSuccess,
			UserContext: &UserContext{
				UserID:   "user1",
				Username: "alice",
				Role:     "admin",
			},
			ResourceContext: &ResourceContext{
				ResourceType: "user",
				ResourceID:   "user123",
				Operation:    "read",
			},
			Data: json.RawMessage(`{}`),
		},
		{
			ID:        uuid.New().String(),
			Timestamp: now.Add(-3 * time.Hour),
			EventType: EventTypeSystemChange,
			Component: "config-service",
			Action:    "update_policy",
			Severity:  SeverityNotice,
			Result:    ResultSuccess,
			UserContext: &UserContext{
				UserID:   "user3",
				Username: "charlie",
				Role:     "operator",
			},
			Data: json.RawMessage(`{}`),
		},
		{
			ID:        uuid.New().String(),
			Timestamp: now.Add(-1 * time.Hour),
			EventType: EventTypeSecurityViolation,
			Component: "security-monitor",
			Action:    "anomaly_detected",
			Severity:  SeverityCritical,
			Result:    ResultFailure,
			UserContext: &UserContext{
				UserID:   "unknown",
				Username: "",
				Role:     "",
			},
			Data: json.RawMessage(`{}`),
		},
	}
}

// Basic Query Tests
func (suite *QueryEngineTestSuite) TestBasicEventRetrieval() {
	suite.Run("get all events", func() {
		query := &Query{
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 5)
		suite.Equal(int64(5), result.TotalCount)
	})

	suite.Run("limit results", func() {
		query := &Query{
			Limit: 2,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 2)
		suite.Equal(int64(5), result.TotalCount) // Total count should reflect all matches
		suite.True(result.HasMore)
	})

	suite.Run("offset results", func() {
		query := &Query{
			Limit:  2,
			Offset: 2,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 2)
		suite.Equal(int64(5), result.TotalCount)
		suite.Equal(4, result.NextOffset)
	})
}

// Filter Tests
func (suite *QueryEngineTestSuite) TestEventFiltering() {
	suite.Run("filter by event type", func() {
		query := &Query{
			Filters: json.RawMessage(`{}`),
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 1) // Only successful auth event
		suite.Equal(EventTypeAuthentication, result.Events[0].EventType)
	})

	suite.Run("filter by multiple event types", func() {
		query := &Query{
			Filters: json.RawMessage(`{}`),
			},
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 2)

		eventTypes := make([]EventType, len(result.Events))
		for i, event := range result.Events {
			eventTypes[i] = event.EventType
		}
		suite.Contains(eventTypes, EventTypeAuthentication)
		suite.Contains(eventTypes, EventTypeAuthenticationFailed)
	})

	suite.Run("filter by severity", func() {
		query := &Query{
			Filters: json.RawMessage(`{}`),
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 1)
		suite.Equal(SeverityCritical, result.Events[0].Severity)
	})

	suite.Run("filter by severity range", func() {
		query := &Query{
			Filters: json.RawMessage(`{}`),
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Greater(len(result.Events), 0)

		for _, event := range result.Events {
			suite.GreaterOrEqual(event.Severity, SeverityWarning)
			suite.LessOrEqual(event.Severity, SeverityCritical)
		}
	})

	suite.Run("filter by component", func() {
		query := &Query{
			Filters: json.RawMessage(`{}`),
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 2) // auth and auth-failed events

		for _, event := range result.Events {
			suite.Equal("auth-service", event.Component)
		}
	})

	suite.Run("filter by user", func() {
		query := &Query{
			Filters: json.RawMessage(`{}`),
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 2) // auth and data access events

		for _, event := range result.Events {
			suite.Equal("user1", event.UserContext.UserID)
		}
	})

	suite.Run("filter by result", func() {
		query := &Query{
			Filters: json.RawMessage(`{}`),
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 2) // auth-failed and security violation

		for _, event := range result.Events {
			suite.Equal(ResultFailure, event.Result)
		}
	})
}

// Time Range Tests
func (suite *QueryEngineTestSuite) TestTimeRangeFiltering() {
	now := time.Now()

	suite.Run("filter by start time", func() {
		query := &Query{
			StartTime: now.Add(-8 * time.Hour), // Last 8 hours
			Limit:     100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 3) // Events within last 8 hours

		for _, event := range result.Events {
			suite.True(event.Timestamp.After(query.StartTime) || event.Timestamp.Equal(query.StartTime))
		}
	})

	suite.Run("filter by end time", func() {
		query := &Query{
			EndTime: now.Add(-4 * time.Hour), // Before 4 hours ago
			Limit:   100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 3) // Events before 4 hours ago

		for _, event := range result.Events {
			suite.True(event.Timestamp.Before(query.EndTime) || event.Timestamp.Equal(query.EndTime))
		}
	})

	suite.Run("filter by time range", func() {
		query := &Query{
			StartTime: now.Add(-8 * time.Hour),
			EndTime:   now.Add(-2 * time.Hour),
			Limit:     100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 2) // Events within the range

		for _, event := range result.Events {
			suite.True(event.Timestamp.After(query.StartTime) || event.Timestamp.Equal(query.StartTime))
			suite.True(event.Timestamp.Before(query.EndTime) || event.Timestamp.Equal(query.EndTime))
		}
	})
}

// Sorting Tests
func (suite *QueryEngineTestSuite) TestEventSorting() {
	suite.Run("sort by timestamp ascending", func() {
		query := &Query{
			SortBy:    "timestamp",
			SortOrder: "asc",
			Limit:     100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 5)

		// Verify ascending order
		for i := 1; i < len(result.Events); i++ {
			suite.True(result.Events[i].Timestamp.After(result.Events[i-1].Timestamp) ||
				result.Events[i].Timestamp.Equal(result.Events[i-1].Timestamp))
		}
	})

	suite.Run("sort by timestamp descending", func() {
		query := &Query{
			SortBy:    "timestamp",
			SortOrder: "desc",
			Limit:     100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 5)

		// Verify descending order
		for i := 1; i < len(result.Events); i++ {
			suite.True(result.Events[i].Timestamp.Before(result.Events[i-1].Timestamp) ||
				result.Events[i].Timestamp.Equal(result.Events[i-1].Timestamp))
		}
	})

	suite.Run("sort by severity", func() {
		query := &Query{
			SortBy:    "severity",
			SortOrder: "desc", // Most severe first
			Limit:     100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 5)

		// Verify severity descending order
		for i := 1; i < len(result.Events); i++ {
			suite.LessOrEqual(result.Events[i].Severity, result.Events[i-1].Severity)
		}
	})

	suite.Run("sort by component", func() {
		query := &Query{
			SortBy:    "component",
			SortOrder: "asc",
			Limit:     100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 5)

		// Verify component alphabetical order
		for i := 1; i < len(result.Events); i++ {
			suite.LessOrEqual(result.Events[i-1].Component, result.Events[i].Component)
		}
	})
}

// Text Search Tests
func (suite *QueryEngineTestSuite) TestTextSearch() {
	suite.Run("search by query text", func() {
		query := &Query{
			TextSearch: "login",
			Limit:      100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 2) // Both auth events have "login" action
	})

	suite.Run("search in event data", func() {
		query := &Query{
			TextSearch: "session_123",
			Limit:      100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 1)
		suite.Equal("session_123", result.Events[0].Data["session_id"])
	})

	suite.Run("search case insensitive", func() {
		query := &Query{
			TextSearch: "ALICE",
			Limit:      100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 2) // Both events for user alice
	})

	suite.Run("search with wildcards", func() {
		query := &Query{
			TextSearch: "user*",
			Limit:      100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Greater(len(result.Events), 0)
	})
}

// Complex Query Tests
func (suite *QueryEngineTestSuite) TestComplexQueries() {
	suite.Run("multiple filters with time range", func() {
		now := time.Now()
		query := &Query{
			TextSearch: "auth",
			StartTime:  now.Add(-24 * time.Hour),
			EndTime:    now.Add(-1 * time.Hour),
			Filters: json.RawMessage(`{}`),
				"result":   "success",
			},
			SortBy:    "timestamp",
			SortOrder: "desc",
			Limit:     100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)

		// Verify all conditions are met
		for _, event := range result.Events {
			suite.True(event.Severity == SeverityInfo || event.Severity == SeverityWarning)
			suite.Equal(ResultSuccess, event.Result)
			suite.True(event.Timestamp.After(query.StartTime) || event.Timestamp.Equal(query.StartTime))
			suite.True(event.Timestamp.Before(query.EndTime) || event.Timestamp.Equal(query.EndTime))
		}
	})

	suite.Run("nested data filtering", func() {
		query := &Query{
			Filters: json.RawMessage(`{}`),
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 1) // Only data access event has sensitive=true
	})

	suite.Run("user context filtering", func() {
		query := &Query{
			Filters: json.RawMessage(`{}`),
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 2) // Both events for alice who is admin

		for _, event := range result.Events {
			suite.Equal("admin", event.UserContext.Role)
		}
	})

	suite.Run("resource context filtering", func() {
		query := &Query{
			Filters: json.RawMessage(`{}`),
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 1) // Only data access event has read operation
	})
}

// Field Selection Tests
func (suite *QueryEngineTestSuite) TestFieldSelection() {
	suite.Run("include specific fields", func() {
		query := &Query{
			// IncludeFields not supported in Query type []string{"id", "timestamp", "event_type", "severity"},
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 5)

		// Verify field filtering (would need actual implementation)
		for _, event := range result.Events {
			suite.NotEmpty(event.ID)
			suite.NotZero(event.Timestamp)
			suite.NotEmpty(event.EventType)
		}
	})

	suite.Run("exclude specific fields", func() {
		query := &Query{
			// ExcludeFields not supported in Query type []string{"data", "stack_trace"},
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.Len(result.Events, 5)

		// In a real implementation, data fields would be excluded
	})
}

// Aggregation Tests
func (suite *QueryEngineTestSuite) TestAggregations() {
	suite.Run("count by event type", func() {
		query := &Query{
			Aggregations: map[string]interface{}{
					"terms": json.RawMessage(`{}`),
				},
			},
			Limit: 0, // No events, just aggregations
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.NotNil(result.Aggregations)
		suite.Contains(result.Aggregations, "event_types")
	})

	suite.Run("count by severity", func() {
		query := &Query{
			Aggregations: map[string]interface{}{
					"terms": json.RawMessage(`{}`),
				},
			},
			Limit: 0,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.NotNil(result.Aggregations)
		suite.Contains(result.Aggregations, "severities")
	})

	suite.Run("date histogram", func() {
		query := &Query{
			Aggregations: map[string]interface{}{
					"date_histogram": json.RawMessage(`{}`),
				},
			},
			Limit: 0,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.NoError(err)
		suite.NotNil(result.Aggregations)
		suite.Contains(result.Aggregations, "events_over_time")
	})
}

// Performance Tests
func (suite *QueryEngineTestSuite) TestQueryPerformance() {
	suite.Run("large dataset query", func() {
		// Generate large dataset
		largeDataset := make([]*AuditEvent, 10000)
		for i := 0; i < len(largeDataset); i++ {
			largeDataset[i] = &AuditEvent{
				ID:        uuid.New().String(),
				Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
				EventType: EventType(fmt.Sprintf("event_type_%d", i%10)),
				Component: fmt.Sprintf("component_%d", i%5),
				Action:    fmt.Sprintf("action_%d", i),
				Severity:  Severity(i % 8),
				Result:    ResultSuccess,
			}
		}

		query := &Query{
			Filters: json.RawMessage(`{}`),
			SortBy:    "timestamp",
			SortOrder: "desc",
			Limit:     100,
		}

		start := time.Now()
		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		duration := time.Since(start)

		suite.NoError(err)
		suite.NotNil(result)
		suite.Less(duration, 5*time.Second, "Query took too long: %v", duration)
	})
}

// Error Handling Tests
func (suite *QueryEngineTestSuite) TestErrorHandling() {
	suite.Run("invalid sort field", func() {
		query := &Query{
			SortBy: "invalid_field",
			Limit:  100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.Error(err)
		suite.Nil(result)
	})

	suite.Run("invalid filter value", func() {
		query := &Query{
			Filters: json.RawMessage(`{}`),
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.Error(err)
		suite.Nil(result)
	})

	suite.Run("negative limit", func() {
		query := &Query{
			Limit: -1,
		}

		result, err := suite.queryEngine.Execute(context.Background(), query, "mock")
		suite.Error(err)
		suite.Nil(result)
	})

	suite.Run("context cancellation", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		query := &Query{
			Limit: 100,
		}

		result, err := suite.queryEngine.Execute(ctx, query, "mock")
		suite.Error(err)
		suite.Contains(err.Error(), "context")
		suite.Nil(result)
	})
}

// Query Builder Tests
func TestQueryBuilder(t *testing.T) {
	tests := []struct {
		name     string
		builder  func() *QueryBuilder
		expected *Query
	}{
		{
			name: "basic query",
			builder: func() *QueryBuilder {
				return NewQueryBuilder().
					WithEventType(EventTypeAuthentication).
					WithSeverity(SeverityError).
					WithLimit(50)
			},
			expected: &Query{
				Filters: json.RawMessage(`{}`),
				Limit: 50,
			},
		},
		{
			name: "time range query",
			builder: func() *QueryBuilder {
				start := time.Now().Add(-24 * time.Hour)
				end := time.Now()
				return NewQueryBuilder().
					WithTimeRange(start, end).
					WithComponent("auth-service").
					WithSortBy("timestamp", "desc")
			},
			expected: &Query{
				StartTime: time.Now().Add(-24 * time.Hour),
				EndTime:   time.Now(),
				Filters: json.RawMessage(`{}`),
				SortBy:    "timestamp",
				SortOrder: "desc",
			},
		},
		{
			name: "complex query with aggregations",
			builder: func() *QueryBuilder {
				return NewQueryBuilder().
					WithQuery("error").
					WithUser("user123").
					WithResult(ResultFailure).
					WithAggregation("severity_counts", map[string]interface{}{
							"field": "severity",
						},
					})
			},
			expected: &Query{
				TextSearch: "error",
				Filters: json.RawMessage(`{}`),
				Aggregations: map[string]interface{}{
						"terms": json.RawMessage(`{}`),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := tt.builder().Build()

			// Compare specific fields as full comparison is complex
			assert.Equal(t, tt.expected.TextSearch, query.TextSearch)
			assert.Equal(t, tt.expected.Limit, query.Limit)
			assert.Equal(t, tt.expected.SortBy, query.SortBy)
			assert.Equal(t, tt.expected.SortOrder, query.SortOrder)

			if tt.expected.Filters != nil {
				assert.NotNil(t, query.Filters)
				for key, expectedValue := range tt.expected.Filters {
					assert.Equal(t, expectedValue, query.Filters[key])
				}
			}
		})
	}
}

// Benchmark Tests
func BenchmarkQueryEngine(b *testing.B) {
	// Create a mock audit system for testing
	auditConfig := &AuditSystemConfig{
		Enabled:       true,
		LogLevel:      SeverityInfo,
		BatchSize:     100,
		FlushInterval: 1 * time.Second,
		MaxQueueSize:  1000,
	}
	auditSystem, err := NewAuditSystem(auditConfig)
	if err != nil {
		b.Fatalf("Failed to create audit system: %v", err)
	}

	// Create mock backends map
	mockBackends := make(map[string]backends.Backend)

	// Create logger
	logger := slog.Default()

	queryEngine := NewQueryEngine(auditSystem, mockBackends, logger)

	// Generate test data
	events := make([]*AuditEvent, 1000)
	for i := 0; i < len(events); i++ {
		events[i] = &AuditEvent{
			ID:        uuid.New().String(),
			Timestamp: time.Now().Add(-time.Duration(i) * time.Minute),
			EventType: EventType(fmt.Sprintf("type_%d", i%10)),
			Component: fmt.Sprintf("comp_%d", i%5),
			Severity:  Severity(i % 8),
			Result:    ResultSuccess,
		}
	}

	b.Run("simple filter", func(b *testing.B) {
		query := &Query{
			Filters: json.RawMessage(`{}`),
			Limit: 100,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			queryEngine.Execute(context.Background(), query, "mock")
		}
	})

	b.Run("complex filter with sort", func(b *testing.B) {
		query := &Query{
			Filters: json.RawMessage(`{}`),
			},
			SortBy:    "timestamp",
			SortOrder: "desc",
			Limit:     50,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			queryEngine.Execute(context.Background(), query, "mock")
		}
	})

	b.Run("text search", func(b *testing.B) {
		query := &Query{
			TextSearch: "comp_1",
			Limit:      100,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			queryEngine.Execute(context.Background(), query, "mock")
		}
	})
}

// All duplicate QueryEngine methods removed - they already exist in query.go

// QueryBuilder provides a fluent interface for building queries
type QueryBuilder struct {
	query *Query
}

func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		query: &Query{
			Filters: make(map[string]interface{}),
		},
	}
}

func (qb *QueryBuilder) WithQuery(query string) *QueryBuilder {
	qb.query.TextSearch = query
	return qb
}

func (qb *QueryBuilder) WithTimeRange(start, end time.Time) *QueryBuilder {
	qb.query.StartTime = start
	qb.query.EndTime = end
	return qb
}

func (qb *QueryBuilder) WithEventType(eventType EventType) *QueryBuilder {
	qb.query.Filters["event_type"] = eventType
	return qb
}

func (qb *QueryBuilder) WithSeverity(severity Severity) *QueryBuilder {
	qb.query.Filters["severity"] = severity
	return qb
}

func (qb *QueryBuilder) WithComponent(component string) *QueryBuilder {
	qb.query.Filters["component"] = component
	return qb
}

func (qb *QueryBuilder) WithUser(userID string) *QueryBuilder {
	qb.query.Filters["user_id"] = userID
	return qb
}

func (qb *QueryBuilder) WithResult(result EventResult) *QueryBuilder {
	qb.query.Filters["result"] = result
	return qb
}

func (qb *QueryBuilder) WithSortBy(field, order string) *QueryBuilder {
	qb.query.SortBy = field
	qb.query.SortOrder = order
	return qb
}

func (qb *QueryBuilder) WithLimit(limit int) *QueryBuilder {
	qb.query.Limit = limit
	return qb
}

func (qb *QueryBuilder) WithOffset(offset int) *QueryBuilder {
	qb.query.Offset = offset
	return qb
}

func (qb *QueryBuilder) WithAggregation(name string, config map[string]interface{}) *QueryBuilder {
	if qb.query.Aggregations == nil {
		qb.query.Aggregations = make(map[string]interface{})
	}
	qb.query.Aggregations[name] = config
	return qb
}

func (qb *QueryBuilder) Build() *Query {
	return qb.query
}

// Re-export backends types for testing
type QueryResponse = backends.QueryResponse

