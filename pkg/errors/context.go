package errors

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"strings"
	"sync"
	"time"

	"k8s.io/client-go/rest"
)

// ContextExtractor defines how to extract context information from various sources.
type ContextExtractor interface {
	ExtractContext(source interface{}) map[string]interface{}
	GetContextKeys() []string
}

// HTTPContextExtractor extracts context from HTTP requests.
type HTTPContextExtractor struct{}

// ExtractContext performs extractcontext operation.
func (h *HTTPContextExtractor) ExtractContext(source interface{}) map[string]interface{} {
	ctx := make(map[string]interface{})

	if req, ok := source.(*http.Request); ok {
		ctx["http_method"] = req.Method
		ctx["http_path"] = req.URL.Path
		ctx["http_query"] = req.URL.RawQuery
		ctx["remote_addr"] = req.RemoteAddr
		ctx["user_agent"] = req.UserAgent()
		ctx["content_length"] = req.ContentLength
		ctx["host"] = req.Host

		// Extract common headers.
		if reqID := req.Header.Get("X-Request-ID"); reqID != "" {
			ctx["request_id"] = reqID
		}
		if corrID := req.Header.Get("X-Correlation-ID"); corrID != "" {
			ctx["correlation_id"] = corrID
		}
		if traceID := req.Header.Get("X-Trace-ID"); traceID != "" {
			ctx["trace_id"] = traceID
		}
		if spanID := req.Header.Get("X-Span-ID"); spanID != "" {
			ctx["span_id"] = spanID
		}
		if userID := req.Header.Get("X-User-ID"); userID != "" {
			ctx["user_id"] = userID
		}
		if sessionID := req.Header.Get("X-Session-ID"); sessionID != "" {
			ctx["session_id"] = sessionID
		}

		// Extract authorization info (without sensitive data).
		if auth := req.Header.Get("Authorization"); auth != "" {
			if len(auth) > 10 {
				ctx["auth_type"] = auth[:10] // e.g., "Bearer ", "Basic "
			}
		}

		// Extract content type.
		ctx["content_type"] = req.Header.Get("Content-Type")
		ctx["accept"] = req.Header.Get("Accept")
	}

	return ctx
}

// GetContextKeys performs getcontextkeys operation.
func (h *HTTPContextExtractor) GetContextKeys() []string {
	return []string{
		"http_method", "http_path", "http_query", "remote_addr", "user_agent",
		"content_length", "host", "request_id", "correlation_id", "trace_id",
		"span_id", "user_id", "session_id", "auth_type", "content_type", "accept",
	}
}

// GoContextExtractor extracts context from Go context.Context.
type GoContextExtractor struct{}

// ExtractContext performs extractcontext operation.
func (g *GoContextExtractor) ExtractContext(source interface{}) map[string]interface{} {
	ctx := make(map[string]interface{})

	if goCtx, ok := source.(context.Context); ok {
		// Extract deadline information.
		if deadline, hasDeadline := goCtx.Deadline(); hasDeadline {
			ctx["deadline"] = deadline.Format(time.RFC3339)
			ctx["time_until_deadline"] = time.Until(deadline).String()
		}

		// Extract common context values with type assertions.
		contextKeys := []string{
			"request_id", "correlation_id", "trace_id", "span_id", "user_id",
			"session_id", "tenant_id", "operation_id", "transaction_id",
			"component", "service", "version", "environment",
		}

		for _, key := range contextKeys {
			if value := goCtx.Value(key); value != nil {
				ctx[key] = fmt.Sprintf("%v", value)
			}
		}

		// Try to extract from structured context values.
		if ctxData := goCtx.Value("context_data"); ctxData != nil {
			if dataMap, ok := ctxData.(map[string]interface{}); ok {
				for k, v := range dataMap {
					ctx[k] = v
				}
			}
		}
	}

	return ctx
}

// GetContextKeys performs getcontextkeys operation.
func (g *GoContextExtractor) GetContextKeys() []string {
	return []string{
		"deadline", "time_until_deadline", "request_id", "correlation_id",
		"trace_id", "span_id", "user_id", "session_id", "tenant_id",
		"operation_id", "transaction_id", "component", "service", "version", "environment",
	}
}

// KubernetesContextExtractor extracts context from Kubernetes resources.
type KubernetesContextExtractor struct{}

// ExtractContext performs extractcontext operation.
func (k *KubernetesContextExtractor) ExtractContext(source interface{}) map[string]interface{} {
	ctx := make(map[string]interface{})

	// Extract from REST config.
	if config, ok := source.(*rest.Config); ok {
		ctx["k8s_host"] = config.Host
		ctx["k8s_api_version"] = config.APIPath
		if config.Username != "" {
			ctx["k8s_username"] = config.Username
		}
		if config.Impersonate.UserName != "" {
			ctx["k8s_impersonate_user"] = config.Impersonate.UserName
		}
	}

	// Extract namespace from various sources.
	if ns := os.Getenv("KUBERNETES_NAMESPACE"); ns != "" {
		ctx["k8s_namespace"] = ns
	}
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		ctx["k8s_namespace"] = ns
	}

	// Extract pod information.
	if podName := os.Getenv("POD_NAME"); podName != "" {
		ctx["k8s_pod_name"] = podName
	}
	if podIP := os.Getenv("POD_IP"); podIP != "" {
		ctx["k8s_pod_ip"] = podIP
	}
	if nodeName := os.Getenv("NODE_NAME"); nodeName != "" {
		ctx["k8s_node_name"] = nodeName
	}
	if serviceAccount := os.Getenv("SERVICE_ACCOUNT"); serviceAccount != "" {
		ctx["k8s_service_account"] = serviceAccount
	}

	// Extract cluster information.
	if clusterName := os.Getenv("CLUSTER_NAME"); clusterName != "" {
		ctx["k8s_cluster_name"] = clusterName
	}

	return ctx
}

// GetContextKeys performs getcontextkeys operation.
func (k *KubernetesContextExtractor) GetContextKeys() []string {
	return []string{
		"k8s_host", "k8s_api_version", "k8s_username", "k8s_impersonate_user",
		"k8s_namespace", "k8s_pod_name", "k8s_pod_ip", "k8s_node_name",
		"k8s_service_account", "k8s_cluster_name",
	}
}

// ORANContextExtractor extracts O-RAN specific context.
type ORANContextExtractor struct{}

// ExtractContext performs extractcontext operation.
func (o *ORANContextExtractor) ExtractContext(source interface{}) map[string]interface{} {
	ctx := make(map[string]interface{})

	// Extract O-RAN environment variables.
	if ricID := os.Getenv("RIC_ID"); ricID != "" {
		ctx["oran_ric_id"] = ricID
	}
	if ricInstance := os.Getenv("RIC_INSTANCE_ID"); ricInstance != "" {
		ctx["oran_ric_instance_id"] = ricInstance
	}
	if e2nodeID := os.Getenv("E2_NODE_ID"); e2nodeID != "" {
		ctx["oran_e2_node_id"] = e2nodeID
	}
	if plmnID := os.Getenv("PLMN_ID"); plmnID != "" {
		ctx["oran_plmn_id"] = plmnID
	}
	if cellID := os.Getenv("CELL_ID"); cellID != "" {
		ctx["oran_cell_id"] = cellID
	}
	if sliceID := os.Getenv("SLICE_ID"); sliceID != "" {
		ctx["oran_slice_id"] = sliceID
	}

	// Extract interface information.
	if a1Interface := os.Getenv("A1_INTERFACE_ENABLED"); a1Interface != "" {
		ctx["oran_a1_enabled"] = a1Interface
	}
	if o1Interface := os.Getenv("O1_INTERFACE_ENABLED"); o1Interface != "" {
		ctx["oran_o1_enabled"] = o1Interface
	}
	if o2Interface := os.Getenv("O2_INTERFACE_ENABLED"); o2Interface != "" {
		ctx["oran_o2_enabled"] = o2Interface
	}
	if e2Interface := os.Getenv("E2_INTERFACE_ENABLED"); e2Interface != "" {
		ctx["oran_e2_enabled"] = e2Interface
	}

	return ctx
}

// GetContextKeys performs getcontextkeys operation.
func (o *ORANContextExtractor) GetContextKeys() []string {
	return []string{
		"oran_ric_id", "oran_ric_instance_id", "oran_e2_node_id", "oran_plmn_id",
		"oran_cell_id", "oran_slice_id", "oran_a1_enabled", "oran_o1_enabled",
		"oran_o2_enabled", "oran_e2_enabled",
	}
}

// SystemContextExtractor extracts system-level context.
type SystemContextExtractor struct{}

// ExtractContext performs extractcontext operation.
func (s *SystemContextExtractor) ExtractContext(source interface{}) map[string]interface{} {
	ctx := make(map[string]interface{})

	// Runtime information.
	ctx["go_version"] = runtime.Version()
	ctx["go_os"] = runtime.GOOS
	ctx["go_arch"] = runtime.GOARCH
	ctx["num_cpu"] = runtime.NumCPU()
	ctx["num_goroutines"] = runtime.NumGoroutine()
	ctx["pid"] = os.Getpid()

	// Memory stats.
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	ctx["memory_alloc"] = memStats.Alloc
	ctx["memory_total_alloc"] = memStats.TotalAlloc
	ctx["memory_sys"] = memStats.Sys
	ctx["memory_num_gc"] = memStats.NumGC

	// Host information.
	if hostname, err := os.Hostname(); err == nil {
		ctx["hostname"] = hostname
	}

	// Environment information.
	if env := os.Getenv("ENVIRONMENT"); env != "" {
		ctx["environment"] = env
	}
	if region := os.Getenv("AWS_REGION"); region != "" {
		ctx["aws_region"] = region
	}
	if zone := os.Getenv("AWS_AVAILABILITY_ZONE"); zone != "" {
		ctx["aws_zone"] = zone
	}
	if instanceID := os.Getenv("AWS_INSTANCE_ID"); instanceID != "" {
		ctx["aws_instance_id"] = instanceID
	}

	// Application information.
	if appName := os.Getenv("APP_NAME"); appName != "" {
		ctx["app_name"] = appName
	}
	if appVersion := os.Getenv("APP_VERSION"); appVersion != "" {
		ctx["app_version"] = appVersion
	}
	if buildVersion := os.Getenv("BUILD_VERSION"); buildVersion != "" {
		ctx["build_version"] = buildVersion
	}
	if gitCommit := os.Getenv("GIT_COMMIT"); gitCommit != "" {
		ctx["git_commit"] = gitCommit
	}

	return ctx
}

// GetContextKeys performs getcontextkeys operation.
func (s *SystemContextExtractor) GetContextKeys() []string {
	return []string{
		"go_version", "go_os", "go_arch", "num_cpu", "num_goroutines", "pid",
		"memory_alloc", "memory_total_alloc", "memory_sys", "memory_num_gc",
		"hostname", "environment", "aws_region", "aws_zone", "aws_instance_id",
		"app_name", "app_version", "build_version", "git_commit",
	}
}

// ContextAwareErrorBuilder builds errors with rich contextual information.
type ContextAwareErrorBuilder struct {
	service    string
	operation  string
	component  string
	extractors []ContextExtractor
	mu         sync.RWMutex
	config     *ErrorConfiguration
}

// NewContextAwareErrorBuilder creates a new context-aware error builder.
func NewContextAwareErrorBuilder(service, operation, component string, config *ErrorConfiguration) *ContextAwareErrorBuilder {
	if config == nil {
		config = DefaultErrorConfiguration()
	}

	builder := &ContextAwareErrorBuilder{
		service:   service,
		operation: operation,
		component: component,
		config:    config,
	}

	// Register default extractors.
	builder.RegisterExtractor(&HTTPContextExtractor{})
	builder.RegisterExtractor(&GoContextExtractor{})
	builder.RegisterExtractor(&KubernetesContextExtractor{})
	builder.RegisterExtractor(&ORANContextExtractor{})
	builder.RegisterExtractor(&SystemContextExtractor{})

	return builder
}

// RegisterExtractor registers a new context extractor.
func (b *ContextAwareErrorBuilder) RegisterExtractor(extractor ContextExtractor) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.extractors = append(b.extractors, extractor)
}

// NewError creates a new context-aware error.
func (b *ContextAwareErrorBuilder) NewError(errType ErrorType, code, message string) *ServiceError {
	return b.NewErrorWithContext(context.Background(), errType, code, message)
}

// NewErrorWithContext creates a new context-aware error with Go context.
func (b *ContextAwareErrorBuilder) NewErrorWithContext(ctx context.Context, errType ErrorType, code, message string) *ServiceError {
	return b.NewErrorWithSources(ctx, errType, code, message, ctx)
}

// NewErrorWithHTTPRequest creates a new context-aware error with HTTP request context.
func (b *ContextAwareErrorBuilder) NewErrorWithHTTPRequest(req *http.Request, errType ErrorType, code, message string) *ServiceError {
	ctx := req.Context()
	return b.NewErrorWithSources(ctx, errType, code, message, req, ctx)
}

// NewErrorWithSources creates a new context-aware error with multiple context sources.
func (b *ContextAwareErrorBuilder) NewErrorWithSources(ctx context.Context, errType ErrorType, code, message string, sources ...interface{}) *ServiceError {
	err := &ServiceError{
		Type:       errType,
		Code:       code,
		Message:    message,
		Service:    b.service,
		Operation:  b.operation,
		Component:  b.component,
		Timestamp:  time.Now(),
		Category:   b.categorizeError(errType),
		Severity:   b.determineSeverity(errType),
		Impact:     b.determineImpact(errType),
		Retryable:  b.isRetryable(errType),
		Temporary:  b.isTemporary(errType),
		HTTPStatus: getHTTPStatusForErrorType(errType),
		Metadata:   make(map[string]interface{}),
	}

	// Extract context from all sources.
	b.extractContextFromSources(err, sources...)

	// Add stack trace if configured.
	if b.config.StackTraceEnabled {
		opts := DefaultStackTraceOptions()
		opts.MaxDepth = b.config.StackTraceDepth
		opts.IncludeSource = b.config.SourceCodeEnabled
		opts.SourceContext = b.config.SourceCodeLines
		err.StackTrace = CaptureStackTrace(opts)
	}

	// Set system context.
	b.setSystemContext(err)

	return err
}

// WrapErrorWithContext wraps an existing error with context.
func (b *ContextAwareErrorBuilder) WrapErrorWithContext(ctx context.Context, cause error, message string, sources ...interface{}) *ServiceError {
	errType := b.categorizeExternalError(cause)

	err := b.NewErrorWithSources(ctx, errType, "wrapped_error", message, sources...)
	err.Cause = cause

	// Build cause chain if the cause is also a ServiceError.
	if serviceErr, ok := cause.(*ServiceError); ok {
		err.CauseChain = []*ServiceError{serviceErr}
		if serviceErr.CauseChain != nil && len(serviceErr.CauseChain) < b.config.MaxCauseChainDepth {
			err.CauseChain = append(err.CauseChain, serviceErr.CauseChain...)
		}
	}

	return err
}

// extractContextFromSources extracts context information from various sources.
func (b *ContextAwareErrorBuilder) extractContextFromSources(err *ServiceError, sources ...interface{}) {
	b.mu.RLock()
	extractors := make([]ContextExtractor, len(b.extractors))
	copy(extractors, b.extractors)
	b.mu.RUnlock()

	for _, source := range sources {
		for _, extractor := range extractors {
			contextData := extractor.ExtractContext(source)
			for key, value := range contextData {
				if value != nil && value != "" {
					err.Metadata[key] = value

					// Map to specific error fields.
					switch key {
					case "request_id":
						if strVal, ok := value.(string); ok {
							err.RequestID = strVal
						}
					case "correlation_id":
						if strVal, ok := value.(string); ok {
							err.CorrelationID = strVal
						}
					case "user_id":
						if strVal, ok := value.(string); ok {
							err.UserID = strVal
						}
					case "session_id":
						if strVal, ok := value.(string); ok {
							err.SessionID = strVal
						}
					case "http_method":
						if strVal, ok := value.(string); ok {
							err.HTTPMethod = strVal
						}
					case "http_path":
						if strVal, ok := value.(string); ok {
							err.HTTPPath = strVal
						}
					case "remote_addr":
						if strVal, ok := value.(string); ok {
							err.RemoteAddr = strVal
						}
					case "user_agent":
						if strVal, ok := value.(string); ok {
							err.UserAgent = strVal
						}
					}
				}
			}
		}
	}
}

// setSystemContext sets system-level context information.
func (b *ContextAwareErrorBuilder) setSystemContext(err *ServiceError) {
	err.Hostname = getCurrentHostname()
	err.PID = getCurrentPID()
	err.GoroutineID = getCurrentGoroutineID()

	// Add debug information if this is an internal error.
	if err.Type == ErrorTypeInternal {
		debugInfo := make(map[string]interface{})
		debugInfo["caller"] = b.getCallerInfo()
		err.DebugInfo = debugInfo
	}
}

// getCallerInfo gets information about the function that created the error.
func (b *ContextAwareErrorBuilder) getCallerInfo() map[string]interface{} {
	callerInfo := make(map[string]interface{})

	if pc, file, line, ok := runtime.Caller(4); ok { // Skip 4 levels: getCallerInfo -> setSystemContext -> NewErrorWithSources -> caller
		callerInfo["file"] = file
		callerInfo["line"] = line

		fn := runtime.FuncForPC(pc)
		funcName := getSafeFunctionName(fn)
		if funcName != "" {
			callerInfo["function"] = funcName
		} else {
			callerInfo["function"] = "<unknown>"
		}
	}

	return callerInfo
}

// categorizeError categorizes an error type into a category.
func (b *ContextAwareErrorBuilder) categorizeError(errType ErrorType) ErrorCategory {
	switch errType {
	case ErrorTypeValidation, ErrorTypeRequired, ErrorTypeInvalid, ErrorTypeFormat, ErrorTypeRange:
		return CategoryValidation
	case ErrorTypeAuth, ErrorTypeUnauthorized, ErrorTypeForbidden, ErrorTypeExpired:
		return CategoryPermission
	case ErrorTypeNetwork, ErrorTypeTimeout, ErrorTypeExternal:
		return CategoryNetwork
	case ErrorTypeResource, ErrorTypeQuota, ErrorTypeDisk, ErrorTypeMemory, ErrorTypeCPU:
		return CategoryResource
	case ErrorTypeConfig:
		return CategoryConfig
	case ErrorTypeInternal:
		return CategoryInternal
	default:
		return CategorySystem
	}
}

// determineSeverity determines the severity level for an error type.
func (b *ContextAwareErrorBuilder) determineSeverity(errType ErrorType) ErrorSeverity {
	switch errType {
	case ErrorTypeValidation, ErrorTypeRequired, ErrorTypeInvalid, ErrorTypeFormat, ErrorTypeRange:
		return SeverityLow
	case ErrorTypeNotFound, ErrorTypeConflict, ErrorTypeDuplicate:
		return SeverityLow
	case ErrorTypeAuth, ErrorTypeUnauthorized, ErrorTypeForbidden:
		return SeverityMedium
	case ErrorTypeTimeout, ErrorTypeRateLimit, ErrorTypeQuota:
		return SeverityMedium
	case ErrorTypeNetwork, ErrorTypeExternal, ErrorTypeDatabase:
		return SeverityHigh
	case ErrorTypeInternal, ErrorTypeResource:
		return SeverityHigh
	case ErrorTypeDisk, ErrorTypeMemory, ErrorTypeCPU:
		return SeverityCritical
	default:
		return SeverityMedium
	}
}

// determineImpact determines the impact level for an error type.
func (b *ContextAwareErrorBuilder) determineImpact(errType ErrorType) ErrorImpact {
	switch errType {
	case ErrorTypeValidation, ErrorTypeRequired, ErrorTypeInvalid:
		return ImpactMinimal
	case ErrorTypeNotFound, ErrorTypeConflict:
		return ImpactMinimal
	case ErrorTypeAuth, ErrorTypeUnauthorized, ErrorTypeForbidden:
		return ImpactModerate
	case ErrorTypeTimeout, ErrorTypeRateLimit:
		return ImpactModerate
	case ErrorTypeNetwork, ErrorTypeExternal:
		return ImpactSevere
	case ErrorTypeDatabase, ErrorTypeInternal:
		return ImpactSevere
	case ErrorTypeResource, ErrorTypeDisk, ErrorTypeMemory:
		return ImpactCritical
	default:
		return ImpactModerate
	}
}

// isRetryable determines if an error type is retryable.
func (b *ContextAwareErrorBuilder) isRetryable(errType ErrorType) bool {
	for _, retryableType := range b.config.RetryableTypes {
		if errType == retryableType {
			return true
		}
	}
	return false
}

// isTemporary determines if an error type is temporary.
func (b *ContextAwareErrorBuilder) isTemporary(errType ErrorType) bool {
	for _, tempType := range b.config.TemporaryTypes {
		if errType == tempType {
			return true
		}
	}
	return false
}

// categorizeExternalError categorizes an external error.
func (b *ContextAwareErrorBuilder) categorizeExternalError(err error) ErrorType {
	if err == nil {
		return ErrorTypeInternal
	}

	errStr := err.Error()

	switch {
	case strings.Contains(errStr, "timeout"):
		return ErrorTypeTimeout
	case strings.Contains(errStr, "connection"):
		return ErrorTypeNetwork
	case strings.Contains(errStr, "not found"):
		return ErrorTypeNotFound
	case strings.Contains(errStr, "unauthorized") || strings.Contains(errStr, "forbidden"):
		return ErrorTypeAuth
	case strings.Contains(errStr, "invalid") || strings.Contains(errStr, "validation"):
		return ErrorTypeValidation
	default:
		return ErrorTypeInternal
	}
}

// ContextualError provides additional context-aware error methods.
type ContextualError struct {
	*ServiceError
	contextBuilder *ContextAwareErrorBuilder
}

// NewContextualError creates a new contextual error.
func NewContextualError(serviceErr *ServiceError, builder *ContextAwareErrorBuilder) *ContextualError {
	return &ContextualError{
		ServiceError:   serviceErr,
		contextBuilder: builder,
	}
}

// WithHTTPRequest adds HTTP request context to the error.
func (ce *ContextualError) WithHTTPRequest(req *http.Request) *ContextualError {
	ce.contextBuilder.extractContextFromSources(ce.ServiceError, req)
	return ce
}

// WithGoContext adds Go context to the error.
func (ce *ContextualError) WithGoContext(ctx context.Context) *ContextualError {
	ce.contextBuilder.extractContextFromSources(ce.ServiceError, ctx)
	return ce
}

// WithCustomContext adds custom context to the error.
func (ce *ContextualError) WithCustomContext(key string, value interface{}) *ContextualError {
	ce.ServiceError.mu.Lock()
	defer ce.ServiceError.mu.Unlock()

	if ce.ServiceError.Metadata == nil {
		ce.ServiceError.Metadata = make(map[string]interface{})
	}
	ce.ServiceError.Metadata[key] = value

	return ce
}

// WithLatency adds latency information to the error.
func (ce *ContextualError) WithLatency(latency time.Duration) *ContextualError {
	ce.ServiceError.mu.Lock()
	defer ce.ServiceError.mu.Unlock()

	ce.ServiceError.Latency = latency
	return ce
}

// WithResource adds resource information to the error.
func (ce *ContextualError) WithResource(resourceType, resourceID string) *ContextualError {
	ce.ServiceError.mu.Lock()
	defer ce.ServiceError.mu.Unlock()

	if ce.ServiceError.Resources == nil {
		ce.ServiceError.Resources = make(map[string]string)
	}
	ce.ServiceError.Resources[resourceType] = resourceID

	return ce
}

// ToServiceError returns the underlying ServiceError.
func (ce *ContextualError) ToServiceError() *ServiceError {
	return ce.ServiceError
}

// ContextKey is a type for context keys to avoid collisions.
type ContextKey string

const (
	// Standard context keys.
	RequestIDKey ContextKey = "request_id"
	// CorrelationIDKey holds correlationidkey value.
	CorrelationIDKey ContextKey = "correlation_id"
	// UserIDKey holds useridkey value.
	UserIDKey ContextKey = "user_id"
	// SessionIDKey holds sessionidkey value.
	SessionIDKey ContextKey = "session_id"
	// TraceIDKey holds traceidkey value.
	TraceIDKey ContextKey = "trace_id"
	// SpanIDKey holds spanidkey value.
	SpanIDKey ContextKey = "span_id"
	// TenantIDKey holds tenantidkey value.
	TenantIDKey ContextKey = "tenant_id"
	// ComponentKey holds componentkey value.
	ComponentKey ContextKey = "component"
	// OperationKey holds operationkey value.
	OperationKey ContextKey = "operation"
	// ServiceKey holds servicekey value.
	ServiceKey ContextKey = "service"
	// VersionKey holds versionkey value.
	VersionKey ContextKey = "version"
	// EnvironmentKey holds environmentkey value.
	EnvironmentKey ContextKey = "environment"
)

// WithRequestID adds request ID to context.
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID extracts request ID from context.
func GetRequestID(ctx context.Context) string {
	if id, ok := ctx.Value(RequestIDKey).(string); ok {
		return id
	}
	return ""
}

// WithCorrelationID adds correlation ID to context.
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// GetCorrelationID extracts correlation ID from context.
func GetCorrelationID(ctx context.Context) string {
	if id, ok := ctx.Value(CorrelationIDKey).(string); ok {
		return id
	}
	return ""
}

// WithUserID adds user ID to context.
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserID extracts user ID from context.
func GetUserID(ctx context.Context) string {
	if id, ok := ctx.Value(UserIDKey).(string); ok {
		return id
	}
	return ""
}

// WithOperationContext adds operation context information.
func WithOperationContext(ctx context.Context, service, component, operation string) context.Context {
	ctx = context.WithValue(ctx, ServiceKey, service)
	ctx = context.WithValue(ctx, ComponentKey, component)
	ctx = context.WithValue(ctx, OperationKey, operation)
	return ctx
}

// ExtractOperationContext extracts operation context from context.
func ExtractOperationContext(ctx context.Context) (service, component, operation string) {
	if s, ok := ctx.Value(ServiceKey).(string); ok {
		service = s
	}
	if c, ok := ctx.Value(ComponentKey).(string); ok {
		component = c
	}
	if o, ok := ctx.Value(OperationKey).(string); ok {
		operation = o
	}
	return service, component, operation
}
