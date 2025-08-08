//go:build go1.24

package generics

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Event represents a generic event with type safety.
type Event[T any] struct {
	ID        string
	Type      string
	Timestamp time.Time
	Data      T
	Metadata  map[string]any
}

// NewEvent creates a new typed event.
func NewEvent[T any](id, eventType string, data T) Event[T] {
	return Event[T]{
		ID:        id,
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
		Metadata:  make(map[string]any),
	}
}

// WithMetadata adds metadata to the event.
func (e Event[T]) WithMetadata(key string, value any) Event[T] {
	e.Metadata[key] = value
	return e
}

// EventHandler defines a function that handles events of type T.
type EventHandler[T any] func(context.Context, Event[T]) Result[bool, error]

// AsyncEventHandler defines an asynchronous event handler.
type AsyncEventHandler[T any] func(context.Context, Event[T]) <-chan Result[bool, error]

// EventFilter defines a function that filters events.
type EventFilter[T any] func(Event[T]) bool

// EventTransformer defines a function that transforms events.
type EventTransformer[T, U any] func(Event[T]) Result[Event[U], error]

// EventBus provides a type-safe event bus implementation.
type EventBus[T any] struct {
	handlers    []EventHandler[T]
	filters     []EventFilter[T]
	middlewares []EventMiddleware[T]
	mu          sync.RWMutex
	buffer      chan Event[T]
	ctx         context.Context
	cancel      context.CancelFunc
	wg          sync.WaitGroup
}

// EventBusConfig configures the event bus.
type EventBusConfig struct {
	BufferSize  int
	WorkerCount int
	Timeout     time.Duration
}

// NewEventBus creates a new type-safe event bus.
func NewEventBus[T any](config EventBusConfig) *EventBus[T] {
	if config.BufferSize == 0 {
		config.BufferSize = 100
	}
	if config.WorkerCount == 0 {
		config.WorkerCount = 3
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	bus := &EventBus[T]{
		buffer: make(chan Event[T], config.BufferSize),
		ctx:    ctx,
		cancel: cancel,
	}
	
	// Start worker goroutines
	for i := 0; i < config.WorkerCount; i++ {
		bus.wg.Add(1)
		go bus.worker(config.Timeout)
	}
	
	return bus
}

// Subscribe adds an event handler.
func (eb *EventBus[T]) Subscribe(handler EventHandler[T]) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.handlers = append(eb.handlers, handler)
}

// SubscribeWithFilter adds an event handler with a filter.
func (eb *EventBus[T]) SubscribeWithFilter(handler EventHandler[T], filter EventFilter[T]) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	
	// Wrap handler with filter
	wrappedHandler := func(ctx context.Context, event Event[T]) Result[bool, error] {
		if !filter(event) {
			return Ok[bool, error](true) // Skip this event
		}
		return handler(ctx, event)
	}
	
	eb.handlers = append(eb.handlers, wrappedHandler)
}

// AddFilter adds a global event filter.
func (eb *EventBus[T]) AddFilter(filter EventFilter[T]) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.filters = append(eb.filters, filter)
}

// AddMiddleware adds middleware to the event processing pipeline.
func (eb *EventBus[T]) AddMiddleware(middleware EventMiddleware[T]) {
	eb.mu.Lock()
	defer eb.mu.Unlock()
	eb.middlewares = append(eb.middlewares, middleware)
}

// Publish publishes an event to all subscribers.
func (eb *EventBus[T]) Publish(event Event[T]) Result[bool, error] {
	select {
	case eb.buffer <- event:
		return Ok[bool, error](true)
	case <-eb.ctx.Done():
		return Err[bool, error](fmt.Errorf("event bus is closed"))
	default:
		return Err[bool, error](fmt.Errorf("event buffer is full"))
	}
}

// PublishSync publishes an event synchronously to all subscribers.
func (eb *EventBus[T]) PublishSync(ctx context.Context, event Event[T]) Result[[]bool, error] {
	// Apply global filters
	eb.mu.RLock()
	for _, filter := range eb.filters {
		if !filter(event) {
			eb.mu.RUnlock()
			return Ok[[]bool, error]([]bool{}) // Event filtered out
		}
	}
	
	handlers := make([]EventHandler[T], len(eb.handlers))
	copy(handlers, eb.handlers)
	middlewares := make([]EventMiddleware[T], len(eb.middlewares))
	copy(middlewares, eb.middlewares)
	eb.mu.RUnlock()
	
	// Apply middlewares
	processedEvent := event
	for _, middleware := range middlewares {
		result := middleware.Process(ctx, processedEvent)
		if result.IsErr() {
			return Err[[]bool, error](result.Error())
		}
		processedEvent = result.Value()
	}
	
	// Execute handlers
	results := make([]bool, len(handlers))
	for i, handler := range handlers {
		result := handler(ctx, processedEvent)
		if result.IsErr() {
			return Err[[]bool, error](result.Error())
		}
		results[i] = result.Value()
	}
	
	return Ok[[]bool, error](results)
}

// worker processes events from the buffer.
func (eb *EventBus[T]) worker(timeout time.Duration) {
	defer eb.wg.Done()
	
	for {
		select {
		case event := <-eb.buffer:
			ctx, cancel := context.WithTimeout(eb.ctx, timeout)
			_ = eb.PublishSync(ctx, event) // Process event
			cancel()
		case <-eb.ctx.Done():
			return
		}
	}
}

// Close gracefully shuts down the event bus.
func (eb *EventBus[T]) Close() error {
	eb.cancel()
	eb.wg.Wait()
	close(eb.buffer)
	return nil
}

// EventMiddleware defines middleware for event processing.
type EventMiddleware[T any] interface {
	Process(ctx context.Context, event Event[T]) Result[Event[T], error]
}

// LoggingMiddleware logs events as they pass through.
type LoggingMiddleware[T any] struct {
	Logger func(Event[T])
}

func (m LoggingMiddleware[T]) Process(ctx context.Context, event Event[T]) Result[Event[T], error] {
	m.Logger(event)
	return Ok[Event[T], error](event)
}

// MetricsMiddleware collects metrics on events.
type MetricsMiddleware[T any] struct {
	Counter  func(string) // Count events by type
	Latency  func(string, time.Duration) // Track processing latency
	recorder func(Event[T])
}

func (m MetricsMiddleware[T]) Process(ctx context.Context, event Event[T]) Result[Event[T], error] {
	start := time.Now()
	m.Counter(event.Type)
	
	defer func() {
		m.Latency(event.Type, time.Since(start))
	}()
	
	if m.recorder != nil {
		m.recorder(event)
	}
	
	return Ok[Event[T], error](event)
}

// EventRouter routes events to different buses based on criteria.
type EventRouter[T any] struct {
	routes map[string]*EventBus[T]
	router func(Event[T]) string
	mu     sync.RWMutex
}

// NewEventRouter creates a new event router.
func NewEventRouter[T any](routerFunc func(Event[T]) string) *EventRouter[T] {
	return &EventRouter[T]{
		routes: make(map[string]*EventBus[T]),
		router: routerFunc,
	}
}

// AddRoute adds a route to a specific event bus.
func (er *EventRouter[T]) AddRoute(name string, bus *EventBus[T]) {
	er.mu.Lock()
	defer er.mu.Unlock()
	er.routes[name] = bus
}

// Route routes an event to the appropriate bus.
func (er *EventRouter[T]) Route(event Event[T]) Result[bool, error] {
	routeName := er.router(event)
	
	er.mu.RLock()
	bus, exists := er.routes[routeName]
	er.mu.RUnlock()
	
	if !exists {
		return Err[bool, error](fmt.Errorf("no route found for: %s", routeName))
	}
	
	return bus.Publish(event)
}

// EventAggregator aggregates events and publishes summary events.
type EventAggregator[TInput, TOutput any] struct {
	inputBus    *EventBus[TInput]
	outputBus   *EventBus[TOutput]
	aggregator  func([]Event[TInput]) Result[Event[TOutput], error]
	window      time.Duration
	buffer      []Event[TInput]
	mu          sync.Mutex
	ticker      *time.Ticker
	ctx         context.Context
	cancel      context.CancelFunc
}

// NewEventAggregator creates a new event aggregator.
func NewEventAggregator[TInput, TOutput any](
	inputBus *EventBus[TInput],
	outputBus *EventBus[TOutput],
	aggregator func([]Event[TInput]) Result[Event[TOutput], error],
	window time.Duration,
) *EventAggregator[TInput, TOutput] {
	ctx, cancel := context.WithCancel(context.Background())
	
	ea := &EventAggregator[TInput, TOutput]{
		inputBus:   inputBus,
		outputBus:  outputBus,
		aggregator: aggregator,
		window:     window,
		buffer:     make([]Event[TInput], 0),
		ticker:     time.NewTicker(window),
		ctx:        ctx,
		cancel:     cancel,
	}
	
	// Subscribe to input events
	inputBus.Subscribe(ea.handleInputEvent)
	
	// Start aggregation worker
	go ea.aggregateWorker()
	
	return ea
}

// handleInputEvent collects input events.
func (ea *EventAggregator[TInput, TOutput]) handleInputEvent(ctx context.Context, event Event[TInput]) Result[bool, error] {
	ea.mu.Lock()
	defer ea.mu.Unlock()
	ea.buffer = append(ea.buffer, event)
	return Ok[bool, error](true)
}

// aggregateWorker periodically aggregates buffered events.
func (ea *EventAggregator[TInput, TOutput]) aggregateWorker() {
	for {
		select {
		case <-ea.ticker.C:
			ea.processBuffer()
		case <-ea.ctx.Done():
			ea.ticker.Stop()
			return
		}
	}
}

// processBuffer processes the current buffer of events.
func (ea *EventAggregator[TInput, TOutput]) processBuffer() {
	ea.mu.Lock()
	if len(ea.buffer) == 0 {
		ea.mu.Unlock()
		return
	}
	
	events := make([]Event[TInput], len(ea.buffer))
	copy(events, ea.buffer)
	ea.buffer = ea.buffer[:0] // Clear buffer
	ea.mu.Unlock()
	
	// Aggregate events
	result := ea.aggregator(events)
	if result.IsOk() {
		_ = ea.outputBus.Publish(result.Value())
	}
}

// Close shuts down the aggregator.
func (ea *EventAggregator[TInput, TOutput]) Close() error {
	ea.cancel()
	return nil
}

// EventStore provides event sourcing capabilities.
type EventStore[T any] interface {
	Store(ctx context.Context, event Event[T]) Result[bool, error]
	LoadByID(ctx context.Context, id string) Result[[]Event[T], error]
	LoadByType(ctx context.Context, eventType string) Result[[]Event[T], error]
	LoadSince(ctx context.Context, timestamp time.Time) Result[[]Event[T], error]
}

// MemoryEventStore provides an in-memory event store implementation.
type MemoryEventStore[T any] struct {
	events []Event[T]
	mu     sync.RWMutex
}

// NewMemoryEventStore creates a new in-memory event store.
func NewMemoryEventStore[T any]() *MemoryEventStore[T] {
	return &MemoryEventStore[T]{
		events: make([]Event[T], 0),
	}
}

// Store stores an event.
func (mes *MemoryEventStore[T]) Store(ctx context.Context, event Event[T]) Result[bool, error] {
	mes.mu.Lock()
	defer mes.mu.Unlock()
	mes.events = append(mes.events, event)
	return Ok[bool, error](true)
}

// LoadByID loads events by ID.
func (mes *MemoryEventStore[T]) LoadByID(ctx context.Context, id string) Result[[]Event[T], error] {
	mes.mu.RLock()
	defer mes.mu.RUnlock()
	
	var matching []Event[T]
	for _, event := range mes.events {
		if event.ID == id {
			matching = append(matching, event)
		}
	}
	
	return Ok[[]Event[T], error](matching)
}

// LoadByType loads events by type.
func (mes *MemoryEventStore[T]) LoadByType(ctx context.Context, eventType string) Result[[]Event[T], error] {
	mes.mu.RLock()
	defer mes.mu.RUnlock()
	
	var matching []Event[T]
	for _, event := range mes.events {
		if event.Type == eventType {
			matching = append(matching, event)
		}
	}
	
	return Ok[[]Event[T], error](matching)
}

// LoadSince loads events since a timestamp.
func (mes *MemoryEventStore[T]) LoadSince(ctx context.Context, timestamp time.Time) Result[[]Event[T], error] {
	mes.mu.RLock()
	defer mes.mu.RUnlock()
	
	var matching []Event[T]
	for _, event := range mes.events {
		if event.Timestamp.After(timestamp) {
			matching = append(matching, event)
		}
	}
	
	return Ok[[]Event[T], error](matching)
}

// EventStream provides streaming event capabilities.
type EventStream[T any] struct {
	bus        *EventBus[T]
	subscriber chan Event[T]
	filters    []EventFilter[T]
	ctx        context.Context
	cancel     context.CancelFunc
}

// NewEventStream creates a new event stream.
func NewEventStream[T any](bus *EventBus[T], bufferSize int) *EventStream[T] {
	ctx, cancel := context.WithCancel(context.Background())
	
	stream := &EventStream[T]{
		bus:        bus,
		subscriber: make(chan Event[T], bufferSize),
		ctx:        ctx,
		cancel:     cancel,
	}
	
	// Subscribe to bus events
	bus.Subscribe(stream.handleEvent)
	
	return stream
}

// AddFilter adds a filter to the stream.
func (es *EventStream[T]) AddFilter(filter EventFilter[T]) {
	es.filters = append(es.filters, filter)
}

// handleEvent handles events from the bus.
func (es *EventStream[T]) handleEvent(ctx context.Context, event Event[T]) Result[bool, error] {
	// Apply filters
	for _, filter := range es.filters {
		if !filter(event) {
			return Ok[bool, error](true) // Skip filtered events
		}
	}
	
	select {
	case es.subscriber <- event:
		return Ok[bool, error](true)
	case <-es.ctx.Done():
		return Err[bool, error](fmt.Errorf("stream is closed"))
	default:
		return Err[bool, error](fmt.Errorf("stream buffer is full"))
	}
}

// Subscribe returns a channel for receiving events.
func (es *EventStream[T]) Subscribe() <-chan Event[T] {
	return es.subscriber
}

// Close closes the event stream.
func (es *EventStream[T]) Close() error {
	es.cancel()
	close(es.subscriber)
	return nil
}

// Predefined event filters

// TypeFilter creates a filter for specific event types.
func TypeFilter[T any](eventTypes ...string) EventFilter[T] {
	typeSet := NewSet(eventTypes...)
	return func(event Event[T]) bool {
		return typeSet.Contains(event.Type)
	}
}

// TimeRangeFilter creates a filter for events within a time range.
func TimeRangeFilter[T any](start, end time.Time) EventFilter[T] {
	return func(event Event[T]) bool {
		return event.Timestamp.After(start) && event.Timestamp.Before(end)
	}
}

// MetadataFilter creates a filter based on metadata.
func MetadataFilter[T any](key string, value any) EventFilter[T] {
	return func(event Event[T]) bool {
		if v, exists := event.Metadata[key]; exists {
			return v == value
		}
		return false
	}
}