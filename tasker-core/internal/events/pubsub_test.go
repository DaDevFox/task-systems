package events

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/DaDevFox/task-systems/task-core/internal/domain"
	"github.com/sirupsen/logrus"
)

const (
	testTaskID   = "test-id"
	testTaskName = "Test Task"
)

func TestPubSub(t *testing.T) {
	tests := []struct {
		name string
		test func(t *testing.T)
	}{
		{"NewPubSub", testNewPubSub},
		{"Subscribe", testSubscribe},
		{"Publish", testPublish},
		{"MultipleSubscribers", testMultipleSubscribers},
		{"GetHandlerCount", testGetHandlerCount},
		{"Clear", testClear},
		{"PublishConcurrent", testPublishConcurrent},
		{"PublishWithNilHandler", testPublishWithNilHandler},
		{"PublishWithEmptyType", testPublishWithEmptyType},
	}

	for _, tt := range tests {
		t.Run(tt.name, tt.test)
	}
}

func testNewPubSub(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel) // Reduce noise in tests

	pubsub := NewPubSub(logger)
	if pubsub == nil {
		t.Fatal("NewPubSub() returned nil")
	}

	if pubsub.GetHandlerCount(EventTaskCreated) != 0 {
		t.Errorf("Expected 0 handlers, got %d", pubsub.GetHandlerCount(EventTaskCreated))
	}

	// Test with nil logger
	pubsub2 := NewPubSub(nil)
	if pubsub2 == nil {
		t.Fatal("NewPubSub(nil) returned nil")
	}
}

func testSubscribe(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	pubsub := NewPubSub(logger)

	called := false
	handler := func(ctx context.Context, event Event) error {
		called = true
		return nil
	}

	pubsub.Subscribe(EventTaskCreated, handler)

	if pubsub.GetHandlerCount(EventTaskCreated) != 1 {
		t.Errorf("Expected 1 handler, got %d", pubsub.GetHandlerCount(EventTaskCreated))
	}

	// Test publishing to verify handler is called
	ctx := context.Background()
	task := &domain.Task{ID: testTaskID, Name: testTaskName}
	pubsub.Publish(ctx, Event{
		Type: EventTaskCreated,
		Data: map[string]interface{}{
			"task": task,
		},
	})

	// Give some time for the goroutine to execute
	time.Sleep(10 * time.Millisecond)

	if !called {
		t.Error("Handler was not called")
	}

	// Test subscribing nil handler
	pubsub.Subscribe(EventTaskCreated, nil)
	// Should still have only 1 handler
	if pubsub.GetHandlerCount(EventTaskCreated) != 1 {
		t.Errorf("Expected 1 handler after nil subscription, got %d", pubsub.GetHandlerCount(EventTaskCreated))
	}
}

func testPublish(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	pubsub := NewPubSub(logger)

	var receivedEvent Event
	var wg sync.WaitGroup
	wg.Add(1)

	handler := func(ctx context.Context, event Event) error {
		receivedEvent = event
		wg.Done()
		return nil
	}

	pubsub.Subscribe(EventTaskCreated, handler)

	task := &domain.Task{ID: testTaskID, Name: testTaskName}
	testEvent := Event{
		Type: EventTaskCreated,
		Data: map[string]interface{}{
			"task": task,
		},
		UserID: "user123",
	}

	ctx := context.Background()
	pubsub.Publish(ctx, testEvent)

	// Wait for the handler to be called
	wg.Wait()

	if receivedEvent.Type != EventTaskCreated {
		t.Errorf("Expected event type %s, got %s", EventTaskCreated, receivedEvent.Type)
	}

	if receivedEvent.Data["task"] != task {
		t.Error("Event data did not match expected task")
	}

	if receivedEvent.UserID != "user123" {
		t.Errorf("Expected UserID user123, got %s", receivedEvent.UserID)
	}
}

func testMultipleSubscribers(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	pubsub := NewPubSub(logger)

	var callCount int
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Subscribe multiple handlers
	for i := 0; i < 3; i++ {
		wg.Add(1)
		handler := func(ctx context.Context, event Event) error {
			mu.Lock()
			callCount++
			mu.Unlock()
			wg.Done()
			return nil
		}

		pubsub.Subscribe(EventTaskCreated, handler)
	}

	if pubsub.GetHandlerCount(EventTaskCreated) != 3 {
		t.Errorf("Expected 3 handlers, got %d", pubsub.GetHandlerCount(EventTaskCreated))
	}

	// Publish one event
	ctx := context.Background()
	task := &domain.Task{ID: testTaskID, Name: testTaskName}
	pubsub.Publish(ctx, Event{
		Type: EventTaskCreated,
		Data: map[string]interface{}{
			"task": task,
		},
	})

	// Wait for all handlers to be called
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if callCount != 3 {
		t.Errorf("Expected 3 handler calls, got %d", callCount)
	}
}

func testGetHandlerCount(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	pubsub := NewPubSub(logger)

	// Initially should be 0
	if pubsub.GetHandlerCount(EventTaskCreated) != 0 {
		t.Errorf("Expected 0 handlers, got %d", pubsub.GetHandlerCount(EventTaskCreated))
	}

	// Add some handlers
	emptyHandler := func(ctx context.Context, event Event) error { return nil }
	pubsub.Subscribe(EventTaskCreated, emptyHandler)
	pubsub.Subscribe(EventTaskCreated, emptyHandler)
	pubsub.Subscribe(EventTaskUpdated, emptyHandler)

	if pubsub.GetHandlerCount(EventTaskCreated) != 2 {
		t.Errorf("Expected 2 handlers for EventTaskCreated, got %d", pubsub.GetHandlerCount(EventTaskCreated))
	}

	if pubsub.GetHandlerCount(EventTaskUpdated) != 1 {
		t.Errorf("Expected 1 handler for EventTaskUpdated, got %d", pubsub.GetHandlerCount(EventTaskUpdated))
	}

	if pubsub.GetHandlerCount(EventTaskCompleted) != 0 {
		t.Errorf("Expected 0 handlers for EventTaskCompleted, got %d", pubsub.GetHandlerCount(EventTaskCompleted))
	}
}

func testClear(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	pubsub := NewPubSub(logger)

	// Add some handlers
	emptyHandler := func(ctx context.Context, event Event) error { return nil }
	pubsub.Subscribe(EventTaskCreated, emptyHandler)
	pubsub.Subscribe(EventTaskUpdated, emptyHandler)
	pubsub.Subscribe(EventTaskCompleted, emptyHandler)

	// Verify handlers exist
	if pubsub.GetHandlerCount(EventTaskCreated) == 0 {
		t.Error("Expected handlers to be added")
	}

	// Clear specific event type
	pubsub.Clear(EventTaskCreated)

	if pubsub.GetHandlerCount(EventTaskCreated) != 0 {
		t.Errorf("Expected 0 handlers for EventTaskCreated after clear, got %d", pubsub.GetHandlerCount(EventTaskCreated))
	}

	// Other handlers should still exist
	if pubsub.GetHandlerCount(EventTaskUpdated) != 1 {
		t.Errorf("Expected 1 handler for EventTaskUpdated after specific clear, got %d", pubsub.GetHandlerCount(EventTaskUpdated))
	}

	// Clear all handlers
	pubsub.Clear("")

	if pubsub.GetHandlerCount(EventTaskUpdated) != 0 {
		t.Errorf("Expected 0 handlers for EventTaskUpdated after clear all, got %d", pubsub.GetHandlerCount(EventTaskUpdated))
	}

	if pubsub.GetHandlerCount(EventTaskCompleted) != 0 {
		t.Errorf("Expected 0 handlers for EventTaskCompleted after clear all, got %d", pubsub.GetHandlerCount(EventTaskCompleted))
	}
}

func testPublishConcurrent(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	pubsub := NewPubSub(logger)

	var callCount int
	var mu sync.Mutex
	var wg sync.WaitGroup

	// Subscribe a handler that counts calls
	pubsub.Subscribe(EventTaskCreated, func(ctx context.Context, event Event) error {
		mu.Lock()
		callCount++
		mu.Unlock()
		wg.Done()
		return nil
	})

	// Publish multiple events concurrently
	numEvents := 10
	wg.Add(numEvents)

	ctx := context.Background()
	for i := 0; i < numEvents; i++ {
		go func(id int) {
			task := &domain.Task{ID: testTaskID, Name: testTaskName}
			pubsub.Publish(ctx, Event{
				Type: EventTaskCreated,
				Data: map[string]interface{}{
					"task": task,
					"id":   id,
				},
			})
		}(i)
	}

	// Wait for all events to be processed
	wg.Wait()

	mu.Lock()
	defer mu.Unlock()
	if callCount != numEvents {
		t.Errorf("Expected %d handler calls, got %d", numEvents, callCount)
	}
}

func testPublishWithNilHandler(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	pubsub := NewPubSub(logger)

	// Publish to event type with no handlers
	ctx := context.Background()
	task := &domain.Task{ID: testTaskID, Name: testTaskName}

	// This should not panic or cause issues
	pubsub.Publish(ctx, Event{
		Type: EventTaskCreated,
		Data: map[string]interface{}{
			"task": task,
		},
	})

	// Give time for any potential goroutines
	time.Sleep(5 * time.Millisecond)
}

func testPublishWithEmptyType(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	pubsub := NewPubSub(logger)

	called := false
	handler := func(ctx context.Context, event Event) error {
		called = true
		return nil
	}

	pubsub.Subscribe(EventTaskCreated, handler)

	// Publish event with empty type
	ctx := context.Background()
	pubsub.Publish(ctx, Event{
		Type: "",
		Data: map[string]interface{}{
			"test": "data",
		},
	})

	// Give time for any potential goroutines
	time.Sleep(5 * time.Millisecond)

	if called {
		t.Error("Handler was called for event with empty type")
	}
}

func TestEventTypes(t *testing.T) {
	// Test that all event types are properly defined
	eventTypes := []EventType{
		EventTaskCreated,
		EventTaskUpdated,
		EventTaskCompleted,
		EventTaskDeleted,
		EventCalendarSync,
		EventEmailSent,
		EventStageChanged,
	}

	for _, eventType := range eventTypes {
		if string(eventType) == "" {
			t.Errorf("Event type is empty")
		}
	}

	// Test uniqueness
	seen := make(map[EventType]bool)
	for _, eventType := range eventTypes {
		if seen[eventType] {
			t.Errorf("Duplicate event type: %s", eventType)
		}
		seen[eventType] = true
	}
}

func TestHandlerError(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.ErrorLevel)
	pubsub := NewPubSub(logger)

	var wg sync.WaitGroup
	wg.Add(1)

	// Handler that returns an error
	errorHandler := func(ctx context.Context, event Event) error {
		wg.Done()
		return errors.New("test error") // Return an error to test error handling
	}

	pubsub.Subscribe(EventTaskCreated, errorHandler)

	ctx := context.Background()
	task := &domain.Task{ID: testTaskID, Name: testTaskName}
	pubsub.Publish(ctx, Event{
		Type: EventTaskCreated,
		Data: map[string]interface{}{
			"task": task,
		},
	})

	// Wait for the handler to be called
	wg.Wait()

	// The error should be logged but not propagated (since handlers run in goroutines)
	// This test mainly ensures error handling doesn't panic
}
