package server

import (
	"fmt"
	"testing"
	"time"

	pb "github.com/DaDevFox/task-systems/shared/pkg/proto/events/v1"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestInMemoryEventStore_Save(t *testing.T) {
	store := NewInMemoryEventStore(0) // No TTL for testing

	event := &pb.Event{
		Id:            "test-event-1",
		SourceService: "test-service",
		Timestamp:     timestamppb.Now(),
	}

	err := store.Save(event)
	if err != nil {
		t.Fatalf("Failed to save event: %v", err)
	}

	// Verify event was saved
	retrieved, err := store.GetByID("test-event-1")
	if err != nil {
		t.Fatalf("Failed to retrieve event: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Event not found after saving")
	}

	if retrieved.Id != "test-event-1" {
		t.Errorf("Expected event ID 'test-event-1', got '%s'", retrieved.Id)
	}
}

func TestInMemoryEventStore_GetByID(t *testing.T) {
	store := NewInMemoryEventStore(0)

	// Test non-existent event
	event, err := store.GetByID("non-existent")
	if err != nil {
		t.Fatalf("Error retrieving non-existent event: %v", err)
	}
	if event != nil {
		t.Error("Expected nil for non-existent event")
	}

	// Save and retrieve event
	testEvent := &pb.Event{
		Id:            "test-event-2",
		SourceService: "test-service",
		Timestamp:     timestamppb.Now(),
	}

	err = store.Save(testEvent)
	if err != nil {
		t.Fatalf("Failed to save event: %v", err)
	}

	retrieved, err := store.GetByID("test-event-2")
	if err != nil {
		t.Fatalf("Failed to retrieve event: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Event not found")
	}

	if retrieved.Id != "test-event-2" {
		t.Errorf("Expected event ID 'test-event-2', got '%s'", retrieved.Id)
	}
}

func TestInMemoryEventStore_GetBySourceService(t *testing.T) {
	store := NewInMemoryEventStore(0)

	// Save events from different services
	event1 := &pb.Event{
		Id:            "event-1",
		SourceService: "service-a",
		Timestamp:     timestamppb.Now(),
	}

	event2 := &pb.Event{
		Id:            "event-2",
		SourceService: "service-a",
		Timestamp:     timestamppb.Now(),
	}

	event3 := &pb.Event{
		Id:            "event-3",
		SourceService: "service-b",
		Timestamp:     timestamppb.Now(),
	}

	store.Save(event1)
	store.Save(event2)
	store.Save(event3)

	// Get events from service-a
	events, err := store.GetBySourceService("service-a")
	if err != nil {
		t.Fatalf("Failed to get events by source service: %v", err)
	}

	if len(events) != 2 {
		t.Errorf("Expected 2 events from service-a, got %d", len(events))
	}

	// Get events from service-b
	events, err = store.GetBySourceService("service-b")
	if err != nil {
		t.Fatalf("Failed to get events by source service: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 event from service-b, got %d", len(events))
	}

	// Get events from non-existent service
	events, err = store.GetBySourceService("non-existent")
	if err != nil {
		t.Fatalf("Failed to get events by source service: %v", err)
	}

	if len(events) != 0 {
		t.Errorf("Expected 0 events from non-existent service, got %d", len(events))
	}
}

func TestInMemoryEventStore_GetByEventType(t *testing.T) {
	store := NewInMemoryEventStore(0)

	// Create events with different types
	event1 := &pb.Event{
		Id:            "event-1",
		SourceService: "test-service",
		Timestamp:     timestamppb.Now(),
		EventType: &pb.Event_TaskCreated{
			TaskCreated: &pb.TaskCreatedEvent{
				TaskId: "task-1",
			},
		},
	}

	event2 := &pb.Event{
		Id:            "event-2",
		SourceService: "test-service",
		Timestamp:     timestamppb.Now(),
		EventType: &pb.Event_TaskCompleted{
			TaskCompleted: &pb.TaskCompletedEvent{
				TaskId: "task-1",
			},
		},
	}

	store.Save(event1)
	store.Save(event2)

	// Get task created events
	events, err := store.GetByEventType(pb.EventType_TASK_CREATED)
	if err != nil {
		t.Fatalf("Failed to get events by type: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 TASK_CREATED event, got %d", len(events))
	}

	// Get task completed events
	events, err = store.GetByEventType(pb.EventType_TASK_COMPLETED)
	if err != nil {
		t.Fatalf("Failed to get events by type: %v", err)
	}

	if len(events) != 1 {
		t.Errorf("Expected 1 TASK_COMPLETED event, got %d", len(events))
	}
}

func TestInMemoryEventStore_GetAll(t *testing.T) {
	store := NewInMemoryEventStore(0)

	// Save multiple events
	for i := 0; i < 5; i++ {
		event := &pb.Event{
			Id:            fmt.Sprintf("event-%d", i),
			SourceService: "test-service",
			Timestamp:     timestamppb.Now(),
		}
		store.Save(event)
	}

	events, err := store.GetAll()
	if err != nil {
		t.Fatalf("Failed to get all events: %v", err)
	}

	if len(events) != 5 {
		t.Errorf("Expected 5 events, got %d", len(events))
	}
}

func TestInMemoryEventStore_TTL(t *testing.T) {
	// Use a very short TTL for testing
	store := NewInMemoryEventStore(100 * time.Millisecond)

	event := &pb.Event{
		Id:            "ttl-event",
		SourceService: "test-service",
		Timestamp:     timestamppb.Now(),
	}

	store.Save(event)

	// Event should exist immediately
	retrieved, err := store.GetByID("ttl-event")
	if err != nil {
		t.Fatalf("Failed to retrieve event: %v", err)
	}
	if retrieved == nil {
		t.Fatal("Event should exist immediately after saving")
	}

	// Wait for TTL to expire
	time.Sleep(200 * time.Millisecond)

	// Event should be cleaned up
	store.CleanupExpiredEvents()

	retrieved, err = store.GetByID("ttl-event")
	if err != nil {
		t.Fatalf("Failed to retrieve event: %v", err)
	}
	if retrieved != nil {
		t.Error("Event should have been cleaned up after TTL expiration")
	}
}

func TestInMemoryEventStore_GetEventCount(t *testing.T) {
	store := NewInMemoryEventStore(0)

	// Initially should be 0
	if count := store.GetEventCount(); count != 0 {
		t.Errorf("Expected initial count 0, got %d", count)
	}

	// Save some events
	for i := 0; i < 3; i++ {
		event := &pb.Event{
			Id:            fmt.Sprintf("count-event-%d", i),
			SourceService: "test-service",
			Timestamp:     timestamppb.Now(),
		}
		store.Save(event)
	}

	if count := store.GetEventCount(); count != 3 {
		t.Errorf("Expected count 3, got %d", count)
	}
}
