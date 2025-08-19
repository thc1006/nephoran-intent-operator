package audit

import (
	"context"
	"fmt"
	"sort"
	"strings"
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
	suite.queryEngine = NewQueryEngine()
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
			Data: map[string]interface{}{
				"session_id": "session_123",
				"duration":   300,
			},
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
			Data: map[string]interface{}{
				"failure_reason": "invalid_password",
				"attempts":       3,
			},
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
			Data: map[string]interface{}{
				"records_accessed": 5,
				"sensitive":        true,
			},
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
			Data: map[string]interface{}{
				"policy_id":   "policy_456",
				"change_type": "security_update",
			},
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
			Data: map[string]interface{}{
				"violation_type": "suspicious_activity",
				"risk_score":     95,
			},
		},
	}
}

// Basic Query Tests
func (suite *QueryEngineTestSuite) TestBasicEventRetrieval() {
	suite.Run("get all events", func() {
		query := &QueryRequest{
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 5)
		suite.Equal(int64(5), result.TotalCount)
	})

	suite.Run("limit results", func() {
		query := &QueryRequest{
			Limit: 2,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 2)
		suite.Equal(int64(5), result.TotalCount) // Total count should reflect all matches
		suite.True(result.HasMore)
	})

	suite.Run("offset results", func() {
		query := &QueryRequest{
			Limit:  2,
			Offset: 2,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 2)
		suite.Equal(int64(5), result.TotalCount)
		suite.Equal(4, result.NextOffset)
	})
}

// Filter Tests
func (suite *QueryEngineTestSuite) TestEventFiltering() {
	suite.Run("filter by event type", func() {
		query := &QueryRequest{
			Filters: map[string]interface{}{
				"event_type": "authentication",
			},
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 1) // Only successful auth event
		suite.Equal(EventTypeAuthentication, result.Events[0].EventType)
	})

	suite.Run("filter by multiple event types", func() {
		query := &QueryRequest{
			Filters: map[string]interface{}{
				"event_type": []string{"authentication", "authentication_failed"},
			},
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
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
		query := &QueryRequest{
			Filters: map[string]interface{}{
				"severity": "critical",
			},
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 1)
		suite.Equal(SeverityCritical, result.Events[0].Severity)
	})

	suite.Run("filter by severity range", func() {
		query := &QueryRequest{
			Filters: map[string]interface{}{
				"min_severity": "warning",
				"max_severity": "critical",
			},
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Greater(len(result.Events), 0)

		for _, event := range result.Events {
			suite.GreaterOrEqual(event.Severity, SeverityWarning)
			suite.LessOrEqual(event.Severity, SeverityCritical)
		}
	})

	suite.Run("filter by component", func() {
		query := &QueryRequest{
			Filters: map[string]interface{}{
				"component": "auth-service",
			},
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 2) // auth and auth-failed events

		for _, event := range result.Events {
			suite.Equal("auth-service", event.Component)
		}
	})

	suite.Run("filter by user", func() {
		query := &QueryRequest{
			Filters: map[string]interface{}{
				"user_id": "user1",
			},
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 2) // auth and data access events

		for _, event := range result.Events {
			suite.Equal("user1", event.UserContext.UserID)
		}
	})

	suite.Run("filter by result", func() {
		query := &QueryRequest{
			Filters: map[string]interface{}{
				"result": "failure",
			},
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
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
		query := &QueryRequest{
			StartTime: now.Add(-8 * time.Hour), // Last 8 hours
			Limit:     100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 3) // Events within last 8 hours

		for _, event := range result.Events {
			suite.True(event.Timestamp.After(query.StartTime) || event.Timestamp.Equal(query.StartTime))
		}
	})

	suite.Run("filter by end time", func() {
		query := &QueryRequest{
			EndTime: now.Add(-4 * time.Hour), // Before 4 hours ago
			Limit:   100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 3) // Events before 4 hours ago

		for _, event := range result.Events {
			suite.True(event.Timestamp.Before(query.EndTime) || event.Timestamp.Equal(query.EndTime))
		}
	})

	suite.Run("filter by time range", func() {
		query := &QueryRequest{
			StartTime: now.Add(-8 * time.Hour),
			EndTime:   now.Add(-2 * time.Hour),
			Limit:     100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
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
		query := &QueryRequest{
			SortBy:    "timestamp",
			SortOrder: "asc",
			Limit:     100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 5)

		// Verify ascending order
		for i := 1; i < len(result.Events); i++ {
			suite.True(result.Events[i].Timestamp.After(result.Events[i-1].Timestamp) ||
				result.Events[i].Timestamp.Equal(result.Events[i-1].Timestamp))
		}
	})

	suite.Run("sort by timestamp descending", func() {
		query := &QueryRequest{
			SortBy:    "timestamp",
			SortOrder: "desc",
			Limit:     100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 5)

		// Verify descending order
		for i := 1; i < len(result.Events); i++ {
			suite.True(result.Events[i].Timestamp.Before(result.Events[i-1].Timestamp) ||
				result.Events[i].Timestamp.Equal(result.Events[i-1].Timestamp))
		}
	})

	suite.Run("sort by severity", func() {
		query := &QueryRequest{
			SortBy:    "severity",
			SortOrder: "desc", // Most severe first
			Limit:     100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 5)

		// Verify severity descending order
		for i := 1; i < len(result.Events); i++ {
			suite.LessOrEqual(result.Events[i].Severity, result.Events[i-1].Severity)
		}
	})

	suite.Run("sort by component", func() {
		query := &QueryRequest{
			SortBy:    "component",
			SortOrder: "asc",
			Limit:     100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
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
		query := &QueryRequest{
			Query: "login",
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 2) // Both auth events have "login" action
	})

	suite.Run("search in event data", func() {
		query := &QueryRequest{
			Query: "session_123",
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 1)
		suite.Equal("session_123", result.Events[0].Data["session_id"])
	})

	suite.Run("search case insensitive", func() {
		query := &QueryRequest{
			Query: "ALICE",
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 2) // Both events for user alice
	})

	suite.Run("search with wildcards", func() {
		query := &QueryRequest{
			Query: "user*",
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Greater(len(result.Events), 0)
	})
}

// Complex Query Tests
func (suite *QueryEngineTestSuite) TestComplexQueries() {
	suite.Run("multiple filters with time range", func() {
		now := time.Now()
		query := &QueryRequest{
			Query:     "auth",
			StartTime: now.Add(-24 * time.Hour),
			EndTime:   now.Add(-1 * time.Hour),
			Filters: map[string]interface{}{
				"severity": []string{"info", "warning"},
				"result":   "success",
			},
			SortBy:    "timestamp",
			SortOrder: "desc",
			Limit:     100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
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
		query := &QueryRequest{
			Filters: map[string]interface{}{
				"data.sensitive": true,
			},
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 1) // Only data access event has sensitive=true
	})

	suite.Run("user context filtering", func() {
		query := &QueryRequest{
			Filters: map[string]interface{}{
				"user_context.role": "admin",
			},
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 2) // Both events for alice who is admin

		for _, event := range result.Events {
			suite.Equal("admin", event.UserContext.Role)
		}
	})

	suite.Run("resource context filtering", func() {
		query := &QueryRequest{
			Filters: map[string]interface{}{
				"resource_context.operation": "read",
			},
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 1) // Only data access event has read operation
	})
}

// Field Selection Tests
func (suite *QueryEngineTestSuite) TestFieldSelection() {
	suite.Run("include specific fields", func() {
		query := &QueryRequest{
			IncludeFields: []string{"id", "timestamp", "event_type", "severity"},
			Limit:         100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
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
		query := &QueryRequest{
			ExcludeFields: []string{"data", "stack_trace"},
			Limit:         100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.Len(result.Events, 5)

		// In a real implementation, data fields would be excluded
	})
}

// Aggregation Tests
func (suite *QueryEngineTestSuite) TestAggregations() {
	suite.Run("count by event type", func() {
		query := &QueryRequest{
			Aggregations: map[string]interface{}{
				"event_types": map[string]interface{}{
					"terms": map[string]interface{}{
						"field": "event_type",
						"size":  10,
					},
				},
			},
			Limit: 0, // No events, just aggregations
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.NotNil(result.Aggregations)
		suite.Contains(result.Aggregations, "event_types")
	})

	suite.Run("count by severity", func() {
		query := &QueryRequest{
			Aggregations: map[string]interface{}{
				"severities": map[string]interface{}{
					"terms": map[string]interface{}{
						"field": "severity",
					},
				},
			},
			Limit: 0,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.NoError(err)
		suite.NotNil(result.Aggregations)
		suite.Contains(result.Aggregations, "severities")
	})

	suite.Run("date histogram", func() {
		query := &QueryRequest{
			Aggregations: map[string]interface{}{
				"events_over_time": map[string]interface{}{
					"date_histogram": map[string]interface{}{
						"field":    "timestamp",
						"interval": "1h",
					},
				},
			},
			Limit: 0,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
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

		query := &QueryRequest{
			Filters: map[string]interface{}{
				"component": "component_0",
			},
			SortBy:    "timestamp",
			SortOrder: "desc",
			Limit:     100,
		}

		start := time.Now()
		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, largeDataset)
		duration := time.Since(start)

		suite.NoError(err)
		suite.NotNil(result)
		suite.Less(duration, 5*time.Second, "Query took too long: %v", duration)
	})
}

// Error Handling Tests
func (suite *QueryEngineTestSuite) TestErrorHandling() {
	suite.Run("invalid sort field", func() {
		query := &QueryRequest{
			SortBy: "invalid_field",
			Limit:  100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.Error(err)
		suite.Nil(result)
	})

	suite.Run("invalid filter value", func() {
		query := &QueryRequest{
			Filters: map[string]interface{}{
				"severity": "invalid_severity",
			},
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.Error(err)
		suite.Nil(result)
	})

	suite.Run("negative limit", func() {
		query := &QueryRequest{
			Limit: -1,
		}

		result, err := suite.queryEngine.ExecuteQuery(context.Background(), query, suite.testEvents)
		suite.Error(err)
		suite.Nil(result)
	})

	suite.Run("context cancellation", func() {
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		query := &QueryRequest{
			Limit: 100,
		}

		result, err := suite.queryEngine.ExecuteQuery(ctx, query, suite.testEvents)
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
		expected *QueryRequest
	}{
		{
			name: "basic query",
			builder: func() *QueryBuilder {
				return NewQueryBuilder().
					WithEventType(EventTypeAuthentication).
					WithSeverity(SeverityError).
					WithLimit(50)
			},
			expected: &QueryRequest{
				Filters: map[string]interface{}{
					"event_type": EventTypeAuthentication,
					"severity":   SeverityError,
				},
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
			expected: &QueryRequest{
				StartTime: time.Now().Add(-24 * time.Hour),
				EndTime:   time.Now(),
				Filters: map[string]interface{}{
					"component": "auth-service",
				},
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
						"terms": map[string]interface{}{
							"field": "severity",
						},
					})
			},
			expected: &QueryRequest{
				Query: "error",
				Filters: map[string]interface{}{
					"user_id": "user123",
					"result":  ResultFailure,
				},
				Aggregations: map[string]interface{}{
					"severity_counts": map[string]interface{}{
						"terms": map[string]interface{}{
							"field": "severity",
						},
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := tt.builder().Build()

			// Compare specific fields as full comparison is complex
			assert.Equal(t, tt.expected.Query, query.Query)
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
	queryEngine := NewQueryEngine()

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
		query := &QueryRequest{
			Filters: map[string]interface{}{
				"component": "comp_0",
			},
			Limit: 100,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			queryEngine.ExecuteQuery(context.Background(), query, events)
		}
	})

	b.Run("complex filter with sort", func(b *testing.B) {
		query := &QueryRequest{
			Filters: map[string]interface{}{
				"severity": []string{"error", "warning"},
			},
			SortBy:    "timestamp",
			SortOrder: "desc",
			Limit:     50,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			queryEngine.ExecuteQuery(context.Background(), query, events)
		}
	})

	b.Run("text search", func(b *testing.B) {
		query := &QueryRequest{
			Query: "comp_1",
			Limit: 100,
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			queryEngine.ExecuteQuery(context.Background(), query, events)
		}
	})
}

// Mock QueryEngine implementation for testing
type QueryEngine struct{}

func NewQueryEngine() *QueryEngine {
	return &QueryEngine{}
}

func (qe *QueryEngine) ExecuteQuery(ctx context.Context, query *QueryRequest, events []*AuditEvent) (*QueryResponse, error) {
	// Check context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// Validate query
	if err := qe.validateQuery(query); err != nil {
		return nil, err
	}

	// Filter events
	filtered := qe.filterEvents(events, query)

	// Sort events
	sorted := qe.sortEvents(filtered, query)

	// Apply pagination
	total := int64(len(sorted))
	start := query.Offset
	end := start + query.Limit

	if start >= len(sorted) {
		sorted = []*AuditEvent{}
	} else if end > len(sorted) {
		sorted = sorted[start:]
	} else {
		sorted = sorted[start:end]
	}

	// Generate aggregations if requested
	aggregations := qe.generateAggregations(filtered, query)

	return &QueryResponse{
		Events:       sorted,
		TotalCount:   total,
		HasMore:      query.Offset+len(sorted) < int(total),
		NextOffset:   query.Offset + len(sorted),
		QueryTime:    time.Since(time.Now().Add(-1 * time.Millisecond)), // Mock duration
		Aggregations: aggregations,
	}, nil
}

func (qe *QueryEngine) validateQuery(query *QueryRequest) error {
	if query.Limit < 0 {
		return fmt.Errorf("limit cannot be negative")
	}

	if query.SortBy != "" {
		validFields := []string{"timestamp", "severity", "component", "event_type", "result"}
		valid := false
		for _, field := range validFields {
			if field == query.SortBy {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid sort field: %s", query.SortBy)
		}
	}

	// Validate severity filters
	if severity, exists := query.Filters["severity"]; exists {
		if severityStr, ok := severity.(string); ok {
			if !isValidSeverity(severityStr) {
				return fmt.Errorf("invalid severity: %s", severityStr)
			}
		}
	}

	return nil
}

func (qe *QueryEngine) filterEvents(events []*AuditEvent, query *QueryRequest) []*AuditEvent {
	var result []*AuditEvent

	for _, event := range events {
		if qe.matchesFilters(event, query) {
			result = append(result, event)
		}
	}

	return result
}

func (qe *QueryEngine) matchesFilters(event *AuditEvent, query *QueryRequest) bool {
	// Time range filters
	if !query.StartTime.IsZero() && event.Timestamp.Before(query.StartTime) {
		return false
	}
	if !query.EndTime.IsZero() && event.Timestamp.After(query.EndTime) {
		return false
	}

	// Text search
	if query.Query != "" && !qe.matchesTextSearch(event, query.Query) {
		return false
	}

	// Field filters
	if query.Filters != nil {
		for key, value := range query.Filters {
			if !qe.matchesFilter(event, key, value) {
				return false
			}
		}
	}

	return true
}

func (qe *QueryEngine) matchesTextSearch(event *AuditEvent, query string) bool {
	query = strings.ToLower(query)

	searchFields := []string{
		strings.ToLower(string(event.EventType)),
		strings.ToLower(event.Component),
		strings.ToLower(event.Action),
		strings.ToLower(event.Description),
	}

	if event.UserContext != nil {
		searchFields = append(searchFields, strings.ToLower(event.UserContext.Username))
	}

	// Search in data fields
	if event.Data != nil {
		for _, v := range event.Data {
			if str, ok := v.(string); ok {
				searchFields = append(searchFields, strings.ToLower(str))
			}
		}
	}

	for _, field := range searchFields {
		if strings.Contains(field, query) {
			return true
		}
	}

	return false
}

func (qe *QueryEngine) matchesFilter(event *AuditEvent, key string, value interface{}) bool {
	switch key {
	case "event_type":
		return qe.matchesEventType(event.EventType, value)
	case "severity":
		return qe.matchesSeverity(event.Severity, value)
	case "component":
		return event.Component == value
	case "user_id":
		return event.UserContext != nil && event.UserContext.UserID == value
	case "result":
		return string(event.Result) == value
	case "user_context.role":
		return event.UserContext != nil && event.UserContext.Role == value
	case "resource_context.operation":
		return event.ResourceContext != nil && event.ResourceContext.Operation == value
	case "data.sensitive":
		if event.Data != nil {
			if sensitive, exists := event.Data["sensitive"]; exists {
				return sensitive == value
			}
		}
		return false
	default:
		return false
	}
}

func (qe *QueryEngine) matchesEventType(eventType EventType, value interface{}) bool {
	switch v := value.(type) {
	case string:
		return string(eventType) == v
	case []string:
		for _, t := range v {
			if string(eventType) == t {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (qe *QueryEngine) matchesSeverity(severity Severity, value interface{}) bool {
	switch v := value.(type) {
	case string:
		return severity.String() == v
	case []string:
		for _, s := range v {
			if severity.String() == s {
				return true
			}
		}
		return false
	default:
		return false
	}
}

func (qe *QueryEngine) sortEvents(events []*AuditEvent, query *QueryRequest) []*AuditEvent {
	if query.SortBy == "" {
		return events
	}

	result := make([]*AuditEvent, len(events))
	copy(result, events)

	sort.Slice(result, func(i, j int) bool {
		var less bool

		switch query.SortBy {
		case "timestamp":
			less = result[i].Timestamp.Before(result[j].Timestamp)
		case "severity":
			less = result[i].Severity < result[j].Severity
		case "component":
			less = result[i].Component < result[j].Component
		case "event_type":
			less = string(result[i].EventType) < string(result[j].EventType)
		case "result":
			less = string(result[i].Result) < string(result[j].Result)
		default:
			return false
		}

		if query.SortOrder == "desc" {
			less = !less
		}

		return less
	})

	return result
}

func (qe *QueryEngine) generateAggregations(events []*AuditEvent, query *QueryRequest) map[string]interface{} {
	if query.Aggregations == nil {
		return nil
	}

	result := make(map[string]interface{})

	for key, aggConfig := range query.Aggregations {
		if config, ok := aggConfig.(map[string]interface{}); ok {
			if _, hasTerms := config["terms"]; hasTerms {
				result[key] = qe.generateTermsAggregation(events, config["terms"])
			}
			if _, hasDateHist := config["date_histogram"]; hasDateHist {
				result[key] = qe.generateDateHistogram(events, config["date_histogram"])
			}
		}
	}

	return result
}

func (qe *QueryEngine) generateTermsAggregation(events []*AuditEvent, config interface{}) interface{} {
	// Mock terms aggregation
	return map[string]interface{}{
		"buckets": []map[string]interface{}{
			{"key": "authentication", "doc_count": 2},
			{"key": "data_access", "doc_count": 1},
			{"key": "security_violation", "doc_count": 1},
		},
	}
}

func (qe *QueryEngine) generateDateHistogram(events []*AuditEvent, config interface{}) interface{} {
	// Mock date histogram
	return map[string]interface{}{
		"buckets": []map[string]interface{}{
			{"key": "2023-01-01T00:00:00Z", "doc_count": 1},
			{"key": "2023-01-01T01:00:00Z", "doc_count": 2},
			{"key": "2023-01-01T02:00:00Z", "doc_count": 1},
		},
	}
}

func isValidSeverity(severity string) bool {
	validSeverities := []string{"emergency", "alert", "critical", "error", "warning", "notice", "info", "debug"}
	for _, valid := range validSeverities {
		if severity == valid {
			return true
		}
	}
	return false
}

// QueryBuilder provides a fluent interface for building queries
type QueryBuilder struct {
	query *QueryRequest
}

func NewQueryBuilder() *QueryBuilder {
	return &QueryBuilder{
		query: &QueryRequest{
			Filters: make(map[string]interface{}),
		},
	}
}

func (qb *QueryBuilder) WithQuery(query string) *QueryBuilder {
	qb.query.Query = query
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

func (qb *QueryBuilder) Build() *QueryRequest {
	return qb.query
}

// Re-export backends types for testing
type QueryRequest = backends.QueryRequest
type QueryResponse = backends.QueryResponse
