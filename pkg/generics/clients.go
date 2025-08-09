//go:build go1.24

package generics

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// ClientInterface defines the contract for generic clients.
type ClientInterface[TRequest, TResponse any] interface {
	// Execute performs the client operation
	Execute(ctx context.Context, request TRequest) Result[TResponse, error]

	// ExecuteWithRetry performs the operation with retry logic
	ExecuteWithRetry(ctx context.Context, request TRequest, retries int) Result[TResponse, error]

	// IsHealthy performs a health check
	IsHealthy(ctx context.Context) Result[bool, error]

	// Close gracefully closes the client
	Close() error
}

// AsyncClientInterface defines the contract for asynchronous clients.
type AsyncClientInterface[TRequest, TResponse any] interface {
	ClientInterface[TRequest, TResponse]

	// ExecuteAsync performs the operation asynchronously
	ExecuteAsync(ctx context.Context, request TRequest) <-chan Result[TResponse, error]

	// ExecuteBatch processes multiple requests concurrently
	ExecuteBatch(ctx context.Context, requests []TRequest) <-chan Result[TResponse, error]
}

// HTTPClient provides a generic type-safe HTTP client wrapper.
type HTTPClient[TRequest, TResponse any] struct {
	client      *http.Client
	baseURL     string
	endpoint    string
	method      string
	headers     map[string]string
	transformer RequestTransformer[TRequest]
	decoder     ResponseDecoder[TResponse]
}

// HTTPClientConfig configures the HTTP client.
type HTTPClientConfig[TRequest, TResponse any] struct {
	BaseURL     string
	Endpoint    string
	Method      string
	Headers     map[string]string
	Timeout     time.Duration
	Transformer RequestTransformer[TRequest]
	Decoder     ResponseDecoder[TResponse]
}

// RequestTransformer converts a request object to HTTP request data.
type RequestTransformer[T any] interface {
	Transform(ctx context.Context, req T) Result[io.Reader, error]
}

// ResponseDecoder converts HTTP response to the desired type.
type ResponseDecoder[T any] interface {
	Decode(ctx context.Context, resp *http.Response) Result[T, error]
}

// JSONRequestTransformer implements RequestTransformer for JSON requests.
type JSONRequestTransformer[T any] struct{}

func (t JSONRequestTransformer[T]) Transform(ctx context.Context, req T) Result[io.Reader, error] {
	data, err := json.Marshal(req)
	if err != nil {
		return Err[io.Reader, error](fmt.Errorf("failed to marshal request: %w", err))
	}
	return Ok[io.Reader, error](strings.NewReader(string(data)))
}

// JSONResponseDecoder implements ResponseDecoder for JSON responses.
type JSONResponseDecoder[T any] struct{}

func (d JSONResponseDecoder[T]) Decode(ctx context.Context, resp *http.Response) Result[T, error] {
	var result T
	decoder := json.NewDecoder(resp.Body)

	if err := decoder.Decode(&result); err != nil {
		return Err[T, error](fmt.Errorf("failed to decode response: %w", err))
	}

	return Ok[T, error](result)
}

// NewHTTPClient creates a new generic HTTP client.
func NewHTTPClient[TRequest, TResponse any](config HTTPClientConfig[TRequest, TResponse]) *HTTPClient[TRequest, TResponse] {
	httpClient := &http.Client{
		Timeout: config.Timeout,
	}

	if config.Timeout == 0 {
		httpClient.Timeout = 30 * time.Second
	}

	// Set default transformer and decoder if not provided
	var transformer RequestTransformer[TRequest]
	var decoder ResponseDecoder[TResponse]

	if config.Transformer != nil {
		transformer = config.Transformer
	} else {
		transformer = JSONRequestTransformer[TRequest]{}
	}

	if config.Decoder != nil {
		decoder = config.Decoder
	} else {
		decoder = JSONResponseDecoder[TResponse]{}
	}

	return &HTTPClient[TRequest, TResponse]{
		client:      httpClient,
		baseURL:     config.BaseURL,
		endpoint:    config.Endpoint,
		method:      config.Method,
		headers:     config.Headers,
		transformer: transformer,
		decoder:     decoder,
	}
}

// Execute performs the HTTP request.
func (c *HTTPClient[TRequest, TResponse]) Execute(ctx context.Context, request TRequest) Result[TResponse, error] {
	// Transform request to HTTP body
	bodyResult := c.transformer.Transform(ctx, request)
	if bodyResult.IsErr() {
		var zero TResponse
		return Err[TResponse, error](bodyResult.Error())
	}

	// Create HTTP request
	url := c.baseURL + c.endpoint
	httpReq, err := http.NewRequestWithContext(ctx, c.method, url, bodyResult.Value())
	if err != nil {
		var zero TResponse
		return Err[TResponse, error](fmt.Errorf("failed to create HTTP request: %w", err))
	}

	// Set headers
	for key, value := range c.headers {
		httpReq.Header.Set(key, value)
	}

	// Set default content type for JSON
	if httpReq.Header.Get("Content-Type") == "" {
		httpReq.Header.Set("Content-Type", "application/json")
	}

	// Execute request
	resp, err := c.client.Do(httpReq)
	if err != nil {
		var zero TResponse
		return Err[TResponse, error](fmt.Errorf("HTTP request failed: %w", err))
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode >= 400 {
		var zero TResponse
		body, _ := io.ReadAll(resp.Body)
		return Err[TResponse, error](fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body)))
	}

	// Decode response
	return c.decoder.Decode(ctx, resp)
}

// ExecuteWithRetry performs the HTTP request with retry logic.
func (c *HTTPClient[TRequest, TResponse]) ExecuteWithRetry(ctx context.Context, request TRequest, retries int) Result[TResponse, error] {
	var lastErr error

	for attempt := 0; attempt <= retries; attempt++ {
		result := c.Execute(ctx, request)
		if result.IsOk() {
			return result
		}

		lastErr = result.Error()

		if attempt < retries {
			// Exponential backoff
			backoff := time.Duration(1<<uint(attempt)) * time.Second
			timer := time.NewTimer(backoff)
			select {
			case <-ctx.Done():
				timer.Stop()
				var zero TResponse
				return Err[TResponse, error](ctx.Err())
			case <-timer.C:
				// Continue to next attempt
			}
		}
	}

	var zero TResponse
	return Err[TResponse, error](fmt.Errorf("all retry attempts failed, last error: %w", lastErr))
}

// IsHealthy performs a health check.
func (c *HTTPClient[TRequest, TResponse]) IsHealthy(ctx context.Context) Result[bool, error] {
	healthURL := c.baseURL + "/health"
	req, err := http.NewRequestWithContext(ctx, "GET", healthURL, nil)
	if err != nil {
		return Err[bool, error](err)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return Err[bool, error](err)
	}
	defer resp.Body.Close()

	healthy := resp.StatusCode >= 200 && resp.StatusCode < 300
	return Ok[bool, error](healthy)
}

// Close gracefully closes the client.
func (c *HTTPClient[TRequest, TResponse]) Close() error {
	c.client.CloseIdleConnections()
	return nil
}

// ExecuteAsync performs the HTTP request asynchronously.
func (c *HTTPClient[TRequest, TResponse]) ExecuteAsync(ctx context.Context, request TRequest) <-chan Result[TResponse, error] {
	resultChan := make(chan Result[TResponse, error], 1)

	go func() {
		defer close(resultChan)
		result := c.Execute(ctx, request)
		resultChan <- result
	}()

	return resultChan
}

// ExecuteBatch processes multiple requests concurrently.
func (c *HTTPClient[TRequest, TResponse]) ExecuteBatch(ctx context.Context, requests []TRequest) <-chan Result[TResponse, error] {
	resultChan := make(chan Result[TResponse, error], len(requests))

	go func() {
		defer close(resultChan)

		for _, req := range requests {
			go func(request TRequest) {
				result := c.Execute(ctx, request)
				resultChan <- result
			}(req)
		}
	}()

	return resultChan
}

// KubernetesClient provides a generic type-safe Kubernetes client wrapper.
type KubernetesClient[T client.Object] struct {
	client    client.Client
	scheme    *runtime.Scheme
	gvk       schema.GroupVersionKind
	namespace string
}

// KubernetesClientConfig configures the Kubernetes client.
type KubernetesClientConfig struct {
	Config    *rest.Config
	Scheme    *runtime.Scheme
	Namespace string
}

// NewKubernetesClient creates a new generic Kubernetes client.
func NewKubernetesClient[T client.Object](config KubernetesClientConfig, obj T) (*KubernetesClient[T], error) {
	k8sClient, err := client.New(config.Config, client.Options{Scheme: config.Scheme})
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	gvk, err := config.Scheme.ObjectKinds(obj)
	if err != nil || len(gvk) == 0 {
		return nil, fmt.Errorf("failed to determine GVK for object type: %w", err)
	}

	return &KubernetesClient[T]{
		client:    k8sClient,
		scheme:    config.Scheme,
		gvk:       gvk[0],
		namespace: config.Namespace,
	}, nil
}

// Create creates a new Kubernetes object.
func (c *KubernetesClient[T]) Create(ctx context.Context, obj T) Result[T, error] {
	if err := c.client.Create(ctx, obj); err != nil {
		var zero T
		return Err[T, error](fmt.Errorf("failed to create object: %w", err))
	}
	return Ok[T, error](obj)
}

// Get retrieves a Kubernetes object by name.
func (c *KubernetesClient[T]) Get(ctx context.Context, name string) Result[T, error] {
	var obj T
	namespacedName := client.ObjectKey{
		Name:      name,
		Namespace: c.namespace,
	}

	if err := c.client.Get(ctx, namespacedName, obj); err != nil {
		var zero T
		return Err[T, error](fmt.Errorf("failed to get object: %w", err))
	}

	return Ok[T, error](obj)
}

// Update updates a Kubernetes object.
func (c *KubernetesClient[T]) Update(ctx context.Context, obj T) Result[T, error] {
	if err := c.client.Update(ctx, obj); err != nil {
		var zero T
		return Err[T, error](fmt.Errorf("failed to update object: %w", err))
	}
	return Ok[T, error](obj)
}

// Delete deletes a Kubernetes object.
func (c *KubernetesClient[T]) Delete(ctx context.Context, obj T) Result[bool, error] {
	if err := c.client.Delete(ctx, obj); err != nil {
		return Err[bool, error](fmt.Errorf("failed to delete object: %w", err))
	}
	return Ok[bool, error](true)
}

// List retrieves a list of Kubernetes objects.
func (c *KubernetesClient[T]) List(ctx context.Context, opts ...client.ListOption) Result[[]T, error] {
	var list T
	listObj := client.ObjectList(list)

	// Apply namespace option if set
	if c.namespace != "" {
		opts = append(opts, client.InNamespace(c.namespace))
	}

	if err := c.client.List(ctx, listObj, opts...); err != nil {
		return Err[[]T, error](fmt.Errorf("failed to list objects: %w", err))
	}

	// Extract items from the list (this is simplified for the example)
	var items []T
	return Ok[[]T, error](items)
}

// Patch applies a patch to a Kubernetes object.
func (c *KubernetesClient[T]) Patch(ctx context.Context, obj T, patch client.Patch) Result[T, error] {
	if err := c.client.Patch(ctx, obj, patch); err != nil {
		var zero T
		return Err[T, error](fmt.Errorf("failed to patch object: %w", err))
	}
	return Ok[T, error](obj)
}

// DatabaseClient provides a generic database client interface.
type DatabaseClient[TEntity, TKey Comparable] struct {
	connectionString string
	tableName        string
}

// DatabaseOperation represents database operations.
type DatabaseOperation int

const (
	OpCreate DatabaseOperation = iota
	OpRead
	OpUpdate
	OpDelete
	OpList
)

// DatabaseQuery represents a type-safe database query.
type DatabaseQuery[T any] struct {
	Operation DatabaseOperation
	Entity    T
	Key       any
	Filter    func(T) bool
	OrderBy   func(T, T) bool
	Limit     int
	Offset    int
}

// NewDatabaseQuery creates a new database query.
func NewDatabaseQuery[T any](op DatabaseOperation) *DatabaseQuery[T] {
	return &DatabaseQuery[T]{
		Operation: op,
	}
}

// WithEntity sets the entity for the query.
func (q *DatabaseQuery[T]) WithEntity(entity T) *DatabaseQuery[T] {
	q.Entity = entity
	return q
}

// WithKey sets the key for the query.
func (q *DatabaseQuery[T]) WithKey(key any) *DatabaseQuery[T] {
	q.Key = key
	return q
}

// WithFilter sets a filter predicate.
func (q *DatabaseQuery[T]) WithFilter(filter func(T) bool) *DatabaseQuery[T] {
	q.Filter = filter
	return q
}

// WithOrderBy sets an ordering function.
func (q *DatabaseQuery[T]) WithOrderBy(orderBy func(T, T) bool) *DatabaseQuery[T] {
	q.OrderBy = orderBy
	return q
}

// WithLimit sets the query limit.
func (q *DatabaseQuery[T]) WithLimit(limit int) *DatabaseQuery[T] {
	q.Limit = limit
	return q
}

// WithOffset sets the query offset.
func (q *DatabaseQuery[T]) WithOffset(offset int) *DatabaseQuery[T] {
	q.Offset = offset
	return q
}

// gRPC Client wrapper would follow similar patterns with protobuf support

// ClientPool manages a pool of clients for load balancing and failover.
type ClientPool[TRequest, TResponse any] struct {
	clients []ClientInterface[TRequest, TResponse]
	current int
}

// NewClientPool creates a new client pool.
func NewClientPool[TRequest, TResponse any](clients ...ClientInterface[TRequest, TResponse]) *ClientPool[TRequest, TResponse] {
	return &ClientPool[TRequest, TResponse]{
		clients: clients,
		current: 0,
	}
}

// Execute executes a request using round-robin client selection.
func (p *ClientPool[TRequest, TResponse]) Execute(ctx context.Context, request TRequest) Result[TResponse, error] {
	if len(p.clients) == 0 {
		var zero TResponse
		return Err[TResponse, error](fmt.Errorf("no clients available"))
	}

	client := p.clients[p.current]
	p.current = (p.current + 1) % len(p.clients)

	return client.Execute(ctx, request)
}

// ExecuteWithFailover executes a request with automatic failover.
func (p *ClientPool[TRequest, TResponse]) ExecuteWithFailover(ctx context.Context, request TRequest) Result[TResponse, error] {
	var lastErr error

	for _, client := range p.clients {
		result := client.Execute(ctx, request)
		if result.IsOk() {
			return result
		}
		lastErr = result.Error()
	}

	var zero TResponse
	return Err[TResponse, error](fmt.Errorf("all clients failed, last error: %w", lastErr))
}

// AddClient adds a client to the pool.
func (p *ClientPool[TRequest, TResponse]) AddClient(client ClientInterface[TRequest, TResponse]) {
	p.clients = append(p.clients, client)
}

// RemoveClient removes a client from the pool.
func (p *ClientPool[TRequest, TResponse]) RemoveClient(index int) {
	if index >= 0 && index < len(p.clients) {
		p.clients = append(p.clients[:index], p.clients[index+1:]...)
		if p.current >= len(p.clients) && len(p.clients) > 0 {
			p.current = 0
		}
	}
}

// HealthCheck performs health checks on all clients.
func (p *ClientPool[TRequest, TResponse]) HealthCheck(ctx context.Context) Result[[]bool, error] {
	results := make([]bool, len(p.clients))

	for i, client := range p.clients {
		health := client.IsHealthy(ctx)
		if health.IsOk() {
			results[i] = health.Value()
		} else {
			results[i] = false
		}
	}

	return Ok[[]bool, error](results)
}

// Close closes all clients in the pool.
func (p *ClientPool[TRequest, TResponse]) Close() error {
	var errs []error

	for _, client := range p.clients {
		if err := client.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing clients: %v", errs)
	}

	return nil
}
