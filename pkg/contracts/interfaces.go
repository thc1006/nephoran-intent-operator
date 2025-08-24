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

package contracts

import (
	"context"
	"time"
)

// ProcessingResult contains the outcome of a phase
type ProcessingResult struct {
	Success      bool                   `json:"success"`
	NextPhase    ProcessingPhase        `json:"nextPhase,omitempty"`
	Data         map[string]interface{} `json:"data,omitempty"`
	Metrics      map[string]float64     `json:"metrics,omitempty"`
	Events       []ProcessingEvent      `json:"events,omitempty"`
	RetryAfter   *time.Duration         `json:"retryAfter,omitempty"`
	ErrorMessage string                 `json:"errorMessage,omitempty"`
	Warnings     []string               `json:"warnings,omitempty"`
}

// ProcessingEvent captures significant events during processing
type ProcessingEvent struct {
	Timestamp   time.Time              `json:"timestamp"`
	Phase       ProcessingPhase        `json:"phase"`
	Event       string                 `json:"event"`
	Component   string                 `json:"component"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Level       string                 `json:"level"`
	Duration    time.Duration          `json:"duration,omitempty"`
	ResourceRef *ResourceReference     `json:"resourceRef,omitempty"`
}

// ResourceReference refers to a Kubernetes resource
type ResourceReference struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`
	Name       string `json:"name"`
	Namespace  string `json:"namespace,omitempty"`
	UID        string `json:"uid,omitempty"`
}

// StateManager interface for managing processing state
type StateManager interface {
	GetState(key string) (interface{}, bool)
	SetState(key string, value interface{}) error
	DeleteState(key string) error
	ListStates(prefix string) (map[string]interface{}, error)
}

// EventBus interface for event communication between controllers
type EventBus interface {
	Publish(ctx context.Context, topic string, event interface{}) error
	Subscribe(ctx context.Context, topic string, handler func(event interface{}) error) error
	Unsubscribe(topic string, handler func(event interface{}) error) error
}

// ComponentHealth represents the health status of a component
type ComponentHealth interface {
	IsHealthy() bool
	GetComponentType() ComponentType
}