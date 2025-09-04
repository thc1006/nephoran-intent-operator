/*

Copyright 2025.



Licensed under the Apache License, Version 2.0 (the "License");

you may not use this file except in compliance with the License.

You may obtain a copy of the License at



    http://www.apache.org/licenses/LICENSE-2.0



Unless required by applicable law or agreed to in writing, software

distributed under the License is distributed on an "AS IS" BASIS,

WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.

See the License for the specific language governing permissions and

limitations under the License.

*/

package shared

import (
	
	"encoding/json"
"context"
	"time"
)

// EventBus defines the interface for event communication.

type EventBus interface {
	// Core event operations.

	PublishStateChange(ctx context.Context, event StateChangeEvent) error

	Subscribe(eventType string, handler EventHandler) error

	Unsubscribe(eventType string) error

	// Event bus lifecycle.

	Start(ctx context.Context) error

	Stop(ctx context.Context) error

	// Event querying.

	GetEventHistory(ctx context.Context, intentID string) ([]ProcessingEvent, error)

	GetEventsByType(ctx context.Context, eventType string, limit int) ([]ProcessingEvent, error)
}

// EventHandler defines the function signature for event handlers.

type EventHandler func(ctx context.Context, event ProcessingEvent) error

// ProcessingEvent represents an event during intent processing.

type ProcessingEvent struct {
	Type string `json:"type"`

	Source string `json:"source"`

	IntentID string `json:"intentId"`

	Phase string `json:"phase"`

	Success bool `json:"success"`

	Data json.RawMessage `json:"data"`

	Timestamp int64 `json:"timestamp"` // Unix timestamp

	CorrelationID string `json:"correlationId"`

	Metadata map[string]string `json:"metadata,omitempty"`
}

// EventBusMetrics provides metrics for the event bus.

type EventBusMetrics interface {
	GetTotalEventsPublished() int64

	GetTotalEventsProcessed() int64

	GetFailedHandlers() int64

	GetAverageProcessingTime() int64 // in milliseconds

	GetBufferUtilization() float64

	// Recording methods.

	RecordEventPublished(eventType string)

	RecordEventProcessed(processingTime time.Duration)

	RecordHandlerFailure()

	SetBufferUtilization(utilization float64)

	SetPartitionCount(count int)
}
