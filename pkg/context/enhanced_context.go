package context

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

// EnhancedContext extends Go 1.24+ context capabilities with additional features.

// for distributed tracing, request correlation, and performance monitoring.

type EnhancedContext struct {
	context.Context

	// Request correlation.

	requestID string

	traceID string

	spanID string

	parentSpanID string

	// User and session context.

	userID string

	sessionID string

	tenantID string

	// Component context.

	serviceName string

	componentName string

	operationName string

	// Performance tracking.

	startTime time.Time

	deadlineTime time.Time

	timeoutDur time.Duration

	// Resource tracking - removed unused fields per staticcheck

	// Metadata and tags.

	metadata map[string]interface{}

	tags map[string]string

	// Cancellation enhancements.

	cancelFunc context.CancelFunc

	cancelReason string

	cancelledAt time.Time

	// Error context.

	errors []error

	// Performance metrics.

	metrics *ContextMetrics

	// Thread safety.

	mu sync.RWMutex
}

// ContextMetrics tracks performance metrics for context operations.

type ContextMetrics struct {
	CreatedAt time.Time

	OperationCount atomic.Int64

	ErrorCount atomic.Int64

	CancellationTime time.Duration

	MaxDepth atomic.Int32

	ChildContexts atomic.Int32
}

// ContextBuilder provides a fluent interface for creating enhanced contexts.

type ContextBuilder struct {
	parent context.Context

	requestID string

	traceID string

	spanID string

	userID string

	sessionID string

	tenantID string

	serviceName string

	componentName string

	operationName string

	timeout time.Duration

	deadline time.Time

	metadata map[string]interface{}

	tags map[string]string
}

// NewContextBuilder creates a new context builder.

func NewContextBuilder(parent context.Context) *ContextBuilder {
	if parent == nil {
		parent = context.Background()
	}

	return &ContextBuilder{
		parent: parent,

		metadata: make(map[string]interface{}),

		tags: make(map[string]string),
	}
}

// WithRequestID sets the request ID.

func (b *ContextBuilder) WithRequestID(requestID string) *ContextBuilder {
	if requestID == "" {
		requestID = uuid.New().String()
	}

	b.requestID = requestID

	return b
}

// WithTrace sets trace information.

func (b *ContextBuilder) WithTrace(traceID, spanID string) *ContextBuilder {
	b.traceID = traceID

	b.spanID = spanID

	return b
}

// WithUser sets user context information.

func (b *ContextBuilder) WithUser(userID, sessionID, tenantID string) *ContextBuilder {
	b.userID = userID

	b.sessionID = sessionID

	b.tenantID = tenantID

	return b
}

// WithComponent sets component information.

func (b *ContextBuilder) WithComponent(service, component, operation string) *ContextBuilder {
	b.serviceName = service

	b.componentName = component

	b.operationName = operation

	return b
}

// WithTimeout sets a timeout for the context.

func (b *ContextBuilder) WithTimeout(timeout time.Duration) *ContextBuilder {
	b.timeout = timeout

	return b
}

// WithDeadline sets a deadline for the context.

func (b *ContextBuilder) WithDeadline(deadline time.Time) *ContextBuilder {
	b.deadline = deadline

	return b
}

// WithMetadata adds metadata key-value pairs.

func (b *ContextBuilder) WithMetadata(key string, value interface{}) *ContextBuilder {
	b.metadata[key] = value

	return b
}

// WithTag adds a tag key-value pair.

func (b *ContextBuilder) WithTag(key, value string) *ContextBuilder {
	b.tags[key] = value

	return b
}

// Build creates the enhanced context.

func (b *ContextBuilder) Build() *EnhancedContext {
	var ctx context.Context

	var cancel context.CancelFunc

	// Apply timeout or deadline.

	if !b.deadline.IsZero() {
		ctx, cancel = context.WithDeadline(b.parent, b.deadline)
	} else if b.timeout > 0 {
		ctx, cancel = context.WithTimeout(b.parent, b.timeout)
	} else {
		ctx, cancel = context.WithCancel(b.parent)
	}

	// Generate IDs if not provided.

	if b.requestID == "" {
		b.requestID = uuid.New().String()
	}

	if b.traceID == "" {
		b.traceID = uuid.New().String()
	}

	if b.spanID == "" {
		b.spanID = uuid.New().String()
	}

	enhanced := &EnhancedContext{
		Context: ctx,

		requestID: b.requestID,

		traceID: b.traceID,

		spanID: b.spanID,

		userID: b.userID,

		sessionID: b.sessionID,

		tenantID: b.tenantID,

		serviceName: b.serviceName,

		componentName: b.componentName,

		operationName: b.operationName,

		startTime: time.Now(),

		deadlineTime: b.deadline,

		timeoutDur: b.timeout,

		metadata: make(map[string]interface{}),

		tags: make(map[string]string),

		cancelFunc: cancel,

		metrics: &ContextMetrics{
			CreatedAt: time.Now(),
		},
	}

	// Copy metadata and tags.

	for k, v := range b.metadata {
		enhanced.metadata[k] = v
	}

	for k, v := range b.tags {
		enhanced.tags[k] = v
	}

	return enhanced
}

// Request correlation methods.

// RequestID returns the request ID.

func (c *EnhancedContext) RequestID() string {
	return c.requestID
}

// TraceID returns the trace ID.

func (c *EnhancedContext) TraceID() string {
	return c.traceID
}

// SpanID returns the span ID.

func (c *EnhancedContext) SpanID() string {
	return c.spanID
}

// User context methods.

// UserID returns the user ID.

func (c *EnhancedContext) UserID() string {
	return c.userID
}

// SessionID returns the session ID.

func (c *EnhancedContext) SessionID() string {
	return c.sessionID
}

// TenantID returns the tenant ID.

func (c *EnhancedContext) TenantID() string {
	return c.tenantID
}

// Component context methods.

// ServiceName returns the service name.

func (c *EnhancedContext) ServiceName() string {
	return c.serviceName
}

// ComponentName returns the component name.

func (c *EnhancedContext) ComponentName() string {
	return c.componentName
}

// OperationName returns the operation name.

func (c *EnhancedContext) OperationName() string {
	return c.operationName
}

// Performance tracking methods.

// StartTime returns when the context was created.

func (c *EnhancedContext) StartTime() time.Time {
	return c.startTime
}

// Duration returns how long the context has been active.

func (c *EnhancedContext) Duration() time.Duration {
	return time.Since(c.startTime)
}

// RemainingTime returns time until deadline.

func (c *EnhancedContext) RemainingTime() time.Duration {
	if c.deadlineTime.IsZero() {
		return 0
	}

	remaining := time.Until(c.deadlineTime)

	if remaining < 0 {
		return 0
	}

	return remaining
}

// IsExpired checks if the context has exceeded its deadline.

func (c *EnhancedContext) IsExpired() bool {
	if c.deadlineTime.IsZero() {
		return false
	}

	return time.Now().After(c.deadlineTime)
}

// Metadata methods.

// GetMetadata retrieves metadata by key.

func (c *EnhancedContext) GetMetadata(key string) (interface{}, bool) {
	c.mu.RLock()

	defer c.mu.RUnlock()

	val, exists := c.metadata[key]

	return val, exists
}

// SetMetadata sets metadata key-value pair.

func (c *EnhancedContext) SetMetadata(key string, value interface{}) {
	c.mu.Lock()

	defer c.mu.Unlock()

	c.metadata[key] = value
}

// GetAllMetadata returns all metadata.

func (c *EnhancedContext) GetAllMetadata() map[string]interface{} {
	c.mu.RLock()

	defer c.mu.RUnlock()

	result := make(map[string]interface{})

	for k, v := range c.metadata {
		result[k] = v
	}

	return result
}

// Tag methods.

// GetTag retrieves tag by key.

func (c *EnhancedContext) GetTag(key string) (string, bool) {
	c.mu.RLock()

	defer c.mu.RUnlock()

	val, exists := c.tags[key]

	return val, exists
}

// SetTag sets tag key-value pair.

func (c *EnhancedContext) SetTag(key, value string) {
	c.mu.Lock()

	defer c.mu.Unlock()

	c.tags[key] = value
}

// GetAllTags returns all tags.

func (c *EnhancedContext) GetAllTags() map[string]string {
	c.mu.RLock()

	defer c.mu.RUnlock()

	result := make(map[string]string)

	for k, v := range c.tags {
		result[k] = v
	}

	return result
}

// Enhanced cancellation methods.

// CancelWithReason cancels the context with a reason.

func (c *EnhancedContext) CancelWithReason(reason string) {
	c.mu.Lock()

	c.cancelReason = reason

	c.cancelledAt = time.Now()

	c.mu.Unlock()

	if c.cancelFunc != nil {
		c.cancelFunc()
	}
}

// CancelReason returns the cancellation reason.

func (c *EnhancedContext) CancelReason() string {
	c.mu.RLock()

	defer c.mu.RUnlock()

	return c.cancelReason
}

// CancelledAt returns when the context was cancelled.

func (c *EnhancedContext) CancelledAt() time.Time {
	c.mu.RLock()

	defer c.mu.RUnlock()

	return c.cancelledAt
}

// Error handling methods.

// AddError adds an error to the context.

func (c *EnhancedContext) AddError(err error) {
	if err == nil {
		return
	}

	c.mu.Lock()

	defer c.mu.Unlock()

	c.errors = append(c.errors, err)

	c.metrics.ErrorCount.Add(1)
}

// GetErrors returns all errors associated with the context.

func (c *EnhancedContext) GetErrors() []error {
	c.mu.RLock()

	defer c.mu.RUnlock()

	result := make([]error, len(c.errors))

	copy(result, c.errors)

	return result
}

// HasErrors returns true if there are any errors.

func (c *EnhancedContext) HasErrors() bool {
	c.mu.RLock()

	defer c.mu.RUnlock()

	return len(c.errors) > 0
}

// Child context methods.

// WithChildSpan creates a child context with a new span.

func (c *EnhancedContext) WithChildSpan(operationName string) *EnhancedContext {
	builder := NewContextBuilder(c.Context)

	// Inherit parent information.

	builder.WithRequestID(c.requestID)

	builder.WithTrace(c.traceID, uuid.New().String())

	builder.WithUser(c.userID, c.sessionID, c.tenantID)

	builder.WithComponent(c.serviceName, c.componentName, operationName)

	child := builder.Build()

	child.parentSpanID = c.spanID

	c.metrics.ChildContexts.Add(1)

	return child
}

// WithChildOperation creates a child context for a specific operation.

func (c *EnhancedContext) WithChildOperation(componentName, operationName string) *EnhancedContext {
	builder := NewContextBuilder(c.Context)

	// Inherit parent information.

	builder.WithRequestID(c.requestID)

	builder.WithTrace(c.traceID, uuid.New().String())

	builder.WithUser(c.userID, c.sessionID, c.tenantID)

	builder.WithComponent(c.serviceName, componentName, operationName)

	child := builder.Build()

	child.parentSpanID = c.spanID

	// Inherit metadata and tags.

	c.mu.RLock()

	for k, v := range c.metadata {
		child.metadata[k] = v
	}

	for k, v := range c.tags {
		child.tags[k] = v
	}

	c.mu.RUnlock()

	c.metrics.ChildContexts.Add(1)

	return child
}

// Metrics methods.

// GetMetrics returns context performance metrics.

func (c *EnhancedContext) GetMetrics() *ContextMetrics {
	return c.metrics
}

// IncrementOperationCount increments the operation counter.

func (c *EnhancedContext) IncrementOperationCount() {
	c.metrics.OperationCount.Add(1)
}

// Utility methods.

// String returns a string representation of the context.

func (c *EnhancedContext) String() string {
	return fmt.Sprintf(

		"EnhancedContext{requestID=%s, traceID=%s, spanID=%s, user=%s, service=%s, component=%s, operation=%s}",

		c.requestID, c.traceID, c.spanID, c.userID, c.serviceName, c.componentName, c.operationName,
	)
}

// ToMap returns context information as a map for logging.

func (c *EnhancedContext) ToMap() map[string]interface{} {
	c.mu.RLock()

	defer c.mu.RUnlock()

	result := make(map[string]interface{})

	// Add metadata.

	for k, v := range c.metadata {
		result["meta_"+k] = v
	}

	// Add tags.

	for k, v := range c.tags {
		result["tag_"+k] = v
	}

	return result
}

// Context propagation utilities.

// FromContext extracts an EnhancedContext from a standard context.

func FromContext(ctx context.Context) (*EnhancedContext, bool) {
	enhanced, ok := ctx.(*EnhancedContext)

	return enhanced, ok
}

// MustFromContext extracts an EnhancedContext or creates a new one.

func MustFromContext(ctx context.Context) *EnhancedContext {
	if enhanced, ok := FromContext(ctx); ok {
		return enhanced
	}

	return NewContextBuilder(ctx).Build()
}

// WithEnhancedContext creates an enhanced context from a standard context.

func WithEnhancedContext(ctx context.Context) *EnhancedContext {
	return NewContextBuilder(ctx).
		WithRequestID("").
		Build()
}

// Middleware for HTTP handlers to create enhanced contexts.

type ContextMiddleware struct {
	serviceName string

	componentName string
}

// NewContextMiddleware creates a new context middleware.

func NewContextMiddleware(serviceName, componentName string) *ContextMiddleware {
	return &ContextMiddleware{
		serviceName: serviceName,

		componentName: componentName,
	}
}

// Handler wraps an HTTP handler with enhanced context.

func (m *ContextMiddleware) Handler(next func(*EnhancedContext)) func(context.Context) {
	return func(ctx context.Context) {
		enhanced := NewContextBuilder(ctx).
			WithComponent(m.serviceName, m.componentName, "http_request").
			WithRequestID("").
			Build()

		next(enhanced)
	}
}

// Background creates a new enhanced background context.

func Background() *EnhancedContext {
	return NewContextBuilder(context.Background()).Build()
}

// TODO creates a new enhanced TODO context.

func TODO() *EnhancedContext {
	return NewContextBuilder(context.TODO()).Build()
}
