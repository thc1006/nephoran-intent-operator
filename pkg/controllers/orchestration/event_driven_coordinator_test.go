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

package orchestration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	nephoranv1 "github.com/thc1006/nephoran-intent-operator/api/v1"
	"github.com/thc1006/nephoran-intent-operator/pkg/controllers/interfaces"
)

var _ = Describe("EventDrivenCoordinator", func() {
	var (
		ctx           context.Context
		coordinator   *EventDrivenCoordinator
		fakeClient    client.Client
		logger        logr.Logger
		scheme        *runtime.Scheme
		networkIntent *nephoranv1.NetworkIntent
	)

	BeforeEach(func() {
		ctx = context.Background()
		logger = zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true))

		// Create scheme and add types
		scheme = runtime.NewScheme()
		Expect(corev1.AddToScheme(scheme)).To(Succeed())
		Expect(nephoranv1.AddToScheme(scheme)).To(Succeed())

		// Create fake client
		fakeClient = fake.NewClientBuilder().
			WithScheme(scheme).
			Build()

		// Create test NetworkIntent
		networkIntent = &nephoranv1.NetworkIntent{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-intent",
				Namespace: "test-namespace",
				UID:       "test-uid-12345",
			},
			Spec: nephoranv1.NetworkIntentSpec{
				Intent: "Deploy AMF for 5G core",
				TargetComponents: []nephoranv1.NetworkTargetComponent{
					nephoranv1.NetworkTargetComponentAMF,
				},
			},
		}

		// Create coordinator
		coordinator = NewEventDrivenCoordinator(fakeClient, logger)
	})

	AfterEach(func() {
		if coordinator != nil {
			coordinator.Stop(ctx)
		}
	})

	Describe("Initialization and Basic Operations", func() {
		It("should create coordinator with proper configuration", func() {
			Expect(coordinator).NotTo(BeNil())
			Expect(coordinator.eventBus).NotTo(BeNil())
			Expect(coordinator.phaseTracker).NotTo(BeNil())
			Expect(coordinator.eventStore).NotTo(BeNil())
			Expect(coordinator.replayManager).NotTo(BeNil())
			Expect(coordinator.enablePersistence).To(BeTrue())
			Expect(coordinator.enableReplay).To(BeTrue())
		})

		It("should start and stop coordinator successfully", func() {
			err := coordinator.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = coordinator.Stop(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Event-Driven Coordination", func() {
		BeforeEach(func() {
			err := coordinator.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should coordinate intent with event publishing", func() {
			// Coordinate the intent
			err := coordinator.CoordinateIntentWithEvents(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())

			// Verify coordination context was created
			intentID := string(networkIntent.UID)
			coordCtx, exists := coordinator.GetCoordinationContext(intentID)
			Expect(exists).To(BeTrue())
			Expect(coordCtx.IntentID).To(Equal(intentID))
			Expect(coordCtx.CurrentPhase).To(Equal(interfaces.PhaseLLMProcessing))
		})

		It("should handle phase transitions", func() {
			intentID := string(networkIntent.UID)

			// Start coordination
			err := coordinator.CoordinateIntentWithEvents(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())

			// Create phase transition event
			event := ProcessingEvent{
				Type:     EventPhaseTransition,
				Source:   "test-controller",
				IntentID: intentID,
				Phase:    interfaces.PhaseResourcePlanning,
				Success:  true,
				Data:     json.RawMessage(`{}`),
			}

			// Handle the event
			err = coordinator.handlePhaseTransition(ctx, event)
			Expect(err).NotTo(HaveOccurred())

			// Verify phase was updated
			coordCtx, exists := coordinator.GetCoordinationContext(intentID)
			Expect(exists).To(BeTrue())
			Expect(coordCtx.CurrentPhase).To(Equal(interfaces.PhaseResourcePlanning))
			Expect(coordCtx.CompletedPhases).To(ContainElement(interfaces.PhaseLLMProcessing))
		})

		It("should handle conflict detection and resolution", func() {
			intentID := string(networkIntent.UID)

			// Start coordination
			err := coordinator.CoordinateIntentWithEvents(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())

			// Create conflict detection event
			conflictID := "test-conflict-123"
			conflictData := map[string]interface{}{
				"conflictId": conflictID,
				"type":       "resource-overlap",
			}
			conflictDataJSON, _ := json.Marshal(conflictData)
			conflictEvent := ProcessingEvent{
				Type:     EventConflictDetected,
				Source:   "conflict-detector",
				IntentID: intentID,
				Phase:    interfaces.PhaseResourcePlanning,
				Success:  true,
				Data:     json.RawMessage(conflictDataJSON),
			}

			// Handle conflict detection
			err = coordinator.handleConflictDetected(ctx, conflictEvent)
			Expect(err).NotTo(HaveOccurred())

			// Verify conflict was recorded
			coordCtx, exists := coordinator.GetCoordinationContext(intentID)
			Expect(exists).To(BeTrue())
			Expect(len(coordCtx.Conflicts)).To(Equal(1))
			Expect(coordCtx.Conflicts[0].ID).To(Equal(conflictID))

			// Create conflict resolution event
			resolutionEvent := ProcessingEvent{
				Type:     EventConflictResolved,
				Source:   "conflict-resolver",
				IntentID: intentID,
				Phase:    interfaces.PhaseResourcePlanning,
				Success:  true,
				Data:     json.RawMessage(`{}`),
			}

			// Handle conflict resolution
			err = coordinator.handleConflictResolved(ctx, resolutionEvent)
			Expect(err).NotTo(HaveOccurred())

			// Verify conflict was removed
			coordCtx, exists = coordinator.GetCoordinationContext(intentID)
			Expect(exists).To(BeTrue())
			Expect(len(coordCtx.Conflicts)).To(Equal(0))
		})

		It("should handle resource lock lifecycle", func() {
			intentID := string(networkIntent.UID)
			lockID := "test-lock-456"

			// Start coordination
			err := coordinator.CoordinateIntentWithEvents(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())

			// Create lock acquisition event
			lockData := map[string]interface{}{
				"lockId": lockID,
			}
			lockDataJSON, _ := json.Marshal(lockData)
			lockAcquiredEvent := ProcessingEvent{
				Type:     EventResourceLockAcquired,
				Source:   "lock-manager",
				IntentID: intentID,
				Phase:    interfaces.PhaseResourcePlanning,
				Success:  true,
				Data:     json.RawMessage(lockDataJSON),
			}

			// Handle lock acquisition
			err = coordinator.handleResourceLockAcquired(ctx, lockAcquiredEvent)
			Expect(err).NotTo(HaveOccurred())

			// Verify lock was recorded
			coordCtx, exists := coordinator.GetCoordinationContext(intentID)
			Expect(exists).To(BeTrue())
			Expect(coordCtx.Locks).To(ContainElement(lockID))

			// Create lock release event
			lockReleaseData := map[string]interface{}{
				"lockId": lockID,
			}
			lockReleaseDataJSON, _ := json.Marshal(lockReleaseData)
			lockReleasedEvent := ProcessingEvent{
				Type:     EventResourceLockReleased,
				Source:   "lock-manager",
				IntentID: intentID,
				Phase:    interfaces.PhaseResourcePlanning,
				Success:  true,
				Data:     json.RawMessage(lockReleaseDataJSON),
			}

			// Handle lock release
			err = coordinator.handleResourceLockReleased(ctx, lockReleasedEvent)
			Expect(err).NotTo(HaveOccurred())

			// Verify lock was removed
			coordCtx, exists = coordinator.GetCoordinationContext(intentID)
			Expect(exists).To(BeTrue())
			Expect(coordCtx.Locks).NotTo(ContainElement(lockID))
		})

		It("should handle recovery events", func() {
			intentID := string(networkIntent.UID)

			// Start coordination
			err := coordinator.CoordinateIntentWithEvents(ctx, networkIntent)
			Expect(err).NotTo(HaveOccurred())

			// Create recovery initiated event
			recoveryInitiatedEvent := ProcessingEvent{
				Type:     EventRecoveryInitiated,
				Source:   "recovery-manager",
				IntentID: intentID,
				Phase:    interfaces.PhaseLLMProcessing,
				Success:  true,
				Data:     json.RawMessage(`{}`),
			}

			// Handle recovery initiation
			err = coordinator.handleRecoveryInitiated(ctx, recoveryInitiatedEvent)
			Expect(err).NotTo(HaveOccurred())

			// Verify recovery was recorded
			coordCtx, exists := coordinator.GetCoordinationContext(intentID)
			Expect(exists).To(BeTrue())
			Expect(coordCtx.RetryCount).To(Equal(1))
			Expect(len(coordCtx.ErrorHistory)).To(Equal(1))
			Expect(coordCtx.ErrorHistory[0]).To(ContainSubstring("Recovery initiated"))

			// Create recovery completed event
			recoveryCompletedEvent := ProcessingEvent{
				Type:     EventRecoveryCompleted,
				Source:   "recovery-manager",
				IntentID: intentID,
				Phase:    interfaces.PhaseLLMProcessing,
				Success:  true,
				Data:     json.RawMessage(`{}`),
			}

			// Handle recovery completion
			err = coordinator.handleRecoveryCompleted(ctx, recoveryCompletedEvent)
			Expect(err).NotTo(HaveOccurred())

			// Verify recovery completion was recorded
			coordCtx, exists = coordinator.GetCoordinationContext(intentID)
			Expect(exists).To(BeTrue())
			Expect(len(coordCtx.ErrorHistory)).To(Equal(2))
			Expect(coordCtx.ErrorHistory[1]).To(ContainSubstring("Recovery completed"))
		})
	})

	Describe("Event Persistence", func() {
		BeforeEach(func() {
			err := coordinator.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should persist events to ConfigMaps", func() {
			intentID := string(networkIntent.UID)

			// Create test event
			event := ProcessingEvent{
				Type:          EventIntentReceived,
				Source:        "test",
				IntentID:      intentID,
				Phase:         interfaces.PhaseIntentReceived,
				Success:       true,
				Data:          json.RawMessage(`{}`),
				Timestamp:     time.Now(),
				CorrelationID: "test-correlation-123",
			}

			// Persist the event
			err := coordinator.eventStore.PersistEvent(ctx, event)
			Expect(err).NotTo(HaveOccurred())

			// Verify ConfigMap was created
			configMapName := "intent-events-" + intentID
			configMap := &corev1.ConfigMap{}

			err = fakeClient.Get(ctx, client.ObjectKey{
				Name:      configMapName,
				Namespace: "nephoran-system",
			}, configMap)
			Expect(err).NotTo(HaveOccurred())

			// Verify event data is in ConfigMap
			Expect(len(configMap.Data)).To(BeNumerically(">", 0))

			// Verify labels
			Expect(configMap.Labels).To(HaveKeyWithValue("nephoran.com/intent-id", intentID))
			Expect(configMap.Labels).To(HaveKeyWithValue("nephoran.com/event-store", "true"))
		})

		It("should retrieve events from persistent storage", func() {
			intentID := string(networkIntent.UID)

			// Create and persist multiple events
			events := []ProcessingEvent{
				{
					Type:          EventIntentReceived,
					Source:        "test",
					IntentID:      intentID,
					Phase:         interfaces.PhaseIntentReceived,
					Success:       true,
					Timestamp:     time.Now(),
					CorrelationID: "correlation-1",
				},
				{
					Type:          EventLLMProcessingStarted,
					Source:        "llm-processor",
					IntentID:      intentID,
					Phase:         interfaces.PhaseLLMProcessing,
					Success:       true,
					Timestamp:     time.Now().Add(1 * time.Minute),
					CorrelationID: "correlation-2",
				},
			}

			for _, event := range events {
				err := coordinator.eventStore.PersistEvent(ctx, event)
				Expect(err).NotTo(HaveOccurred())
			}

			// Retrieve events
			retrievedEvents, err := coordinator.eventStore.GetEventsForIntent(ctx, intentID)
			Expect(err).NotTo(HaveOccurred())
			Expect(len(retrievedEvents)).To(Equal(2))

			// Verify event content
			intentReceivedFound := false
			llmStartedFound := false

			for _, event := range retrievedEvents {
				switch event.Type {
				case EventIntentReceived:
					intentReceivedFound = true
					Expect(event.IntentID).To(Equal(intentID))
					Expect(event.Phase).To(Equal(interfaces.PhaseIntentReceived))
				case EventLLMProcessingStarted:
					llmStartedFound = true
					Expect(event.IntentID).To(Equal(intentID))
					Expect(event.Phase).To(Equal(interfaces.PhaseLLMProcessing))
				}
			}

			Expect(intentReceivedFound).To(BeTrue())
			Expect(llmStartedFound).To(BeTrue())
		})
	})

	Describe("Event Replay", func() {
		BeforeEach(func() {
			err := coordinator.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should replay events for intent recovery", func() {
			intentID := string(networkIntent.UID)

			// Create and persist events
			events := []ProcessingEvent{
				{
					Type:          EventIntentReceived,
					Source:        "test",
					IntentID:      intentID,
					Phase:         interfaces.PhaseIntentReceived,
					Success:       true,
					Timestamp:     time.Now(),
					CorrelationID: "replay-correlation-1",
				},
				{
					Type:          EventLLMProcessingCompleted,
					Source:        "llm-processor",
					IntentID:      intentID,
					Phase:         interfaces.PhaseLLMProcessing,
					Success:       true,
					Timestamp:     time.Now().Add(1 * time.Minute),
					CorrelationID: "replay-correlation-2",
				},
			}

			for _, event := range events {
				err := coordinator.eventStore.PersistEvent(ctx, event)
				Expect(err).NotTo(HaveOccurred())
			}

			// Track replayed events
			replayedEvents := make([]ProcessingEvent, 0)
			replayHandler := func(ctx context.Context, event ProcessingEvent) error {
				replayedEvents = append(replayedEvents, event)
				return nil
			}

			// Subscribe to all events to capture replays
			err := coordinator.eventBus.Subscribe("*", replayHandler)
			Expect(err).NotTo(HaveOccurred())

			// Wait for event bus to process subscriptions
			time.Sleep(100 * time.Millisecond)

			// Replay events
			err = coordinator.replayManager.ReplayEventsForIntent(ctx, intentID)
			Expect(err).NotTo(HaveOccurred())

			// Wait for replay to complete
			time.Sleep(500 * time.Millisecond)

			// Verify events were replayed
			Expect(len(replayedEvents)).To(BeNumerically(">=", 2))
		})

		It("should handle replay with no events gracefully", func() {
			intentID := "non-existent-intent"

			// Attempt replay for non-existent intent
			err := coordinator.replayManager.ReplayEventsForIntent(ctx, intentID)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("Recovery Monitoring", func() {
		BeforeEach(func() {
			err := coordinator.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should detect stuck intents and initiate recovery", func() {
			intentID := string(networkIntent.UID)

			// Create a coordination context with old timestamp
			oldTime := time.Now().Add(-15 * time.Minute)
			coordCtx := &CoordinationContext{
				IntentID:       intentID,
				CurrentPhase:   interfaces.PhaseLLMProcessing,
				StartTime:      oldTime,
				LastUpdateTime: oldTime,
				Metadata:       json.RawMessage(`{}`),
			}

			coordinator.mutex.Lock()
			coordinator.coordinationContexts[intentID] = coordCtx
			coordinator.mutex.Unlock()

			// Track recovery events
			recoveryEventReceived := false
			recoveryHandler := func(ctx context.Context, event ProcessingEvent) error {
				if event.Type == EventRecoveryInitiated && event.IntentID == intentID {
					recoveryEventReceived = true
				}
				return nil
			}

			err := coordinator.eventBus.Subscribe(EventRecoveryInitiated, recoveryHandler)
			Expect(err).NotTo(HaveOccurred())

			// Trigger recovery check
			coordinator.checkForRecoveryNeeded(ctx)

			// Wait for event processing
			time.Sleep(200 * time.Millisecond)

			// Verify recovery was initiated
			Expect(recoveryEventReceived).To(BeTrue())
		})
	})

	Describe("Concurrent Operations", func() {
		BeforeEach(func() {
			err := coordinator.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle multiple intents concurrently", func() {
			numIntents := 5
			intents := make([]*nephoranv1.NetworkIntent, numIntents)

			// Create multiple intents
			for i := 0; i < numIntents; i++ {
				intents[i] = &nephoranv1.NetworkIntent{
					ObjectMeta: metav1.ObjectMeta{
						Name:      fmt.Sprintf("test-intent-%d", i),
						Namespace: "test-namespace",
						UID:       types.UID(fmt.Sprintf("test-uid-%d", i)),
					},
					Spec: nephoranv1.NetworkIntentSpec{
						Intent: fmt.Sprintf("Deploy AMF for 5G core %d", i),
					},
				}
			}

			// Coordinate all intents concurrently
			done := make(chan bool, numIntents)
			for i := 0; i < numIntents; i++ {
				go func(intent *nephoranv1.NetworkIntent) {
					defer GinkgoRecover()
					err := coordinator.CoordinateIntentWithEvents(ctx, intent)
					Expect(err).NotTo(HaveOccurred())
					done <- true
				}(intents[i])
			}

			// Wait for all to complete
			for i := 0; i < numIntents; i++ {
				Eventually(done).Should(Receive())
			}

			// Verify all coordination contexts were created
			for i := 0; i < numIntents; i++ {
				intentID := string(intents[i].UID)
				coordCtx, exists := coordinator.GetCoordinationContext(intentID)
				Expect(exists).To(BeTrue())
				Expect(coordCtx.IntentID).To(Equal(intentID))
			}
		})
	})
})

func TestEventDrivenCoordinator(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "EventDrivenCoordinator Suite")
}
