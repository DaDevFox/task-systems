package server

import (
	"sync"
	"time"

	pb "github.com/DaDevFox/task-systems/shared/pkg/proto/events/v1"
	"github.com/DaDevFox/task-systems/shared/util"
)

// EventStore defines the interface for storing and retrieving events
type EventStore interface {
	Save(event *pb.Event) error
	GetByID(id string) (*pb.Event, error)
	GetBySourceService(service string) ([]*pb.Event, error)
	GetByEventType(eventType pb.EventType) ([]*pb.Event, error)
	GetAll() ([]*pb.Event, error)
	CleanupExpiredEvents() error
}

// InMemoryEventStore implements EventStore using in-memory storage
type InMemoryEventStore struct {
	mu     sync.RWMutex
	events map[string]*pb.Event
	ttl    time.Duration
}

// NewInMemoryEventStore creates a new in-memory event store
func NewInMemoryEventStore(ttl time.Duration) *InMemoryEventStore {
	store := &InMemoryEventStore{
		events: make(map[string]*pb.Event),
		ttl:    ttl,
	}

	// Start cleanup goroutine if TTL is set
	if ttl > 0 {
		go store.cleanupRoutine()
	}

	return store
}

// Save stores an event in the store
func (s *InMemoryEventStore) Save(event *pb.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.events[event.Id] = event
	return nil
}

// GetByID retrieves an event by its ID
func (s *InMemoryEventStore) GetByID(id string) (*pb.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	event, exists := s.events[id]
	if !exists {
		return nil, nil // Not found
	}

	return event, nil
}

// GetBySourceService retrieves all events from a specific source service
func (s *InMemoryEventStore) GetBySourceService(service string) ([]*pb.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var events []*pb.Event
	for _, event := range s.events {
		if event.SourceService == service {
			events = append(events, event)
		}
	}

	return events, nil
}

// GetByEventType retrieves all events of a specific type
func (s *InMemoryEventStore) GetByEventType(eventType pb.EventType) ([]*pb.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var events []*pb.Event
	for _, event := range s.events {
		if util.GetEventType(event) == eventType {
			events = append(events, event)
		}
	}

	return events, nil
}

// GetAll retrieves all events
func (s *InMemoryEventStore) GetAll() ([]*pb.Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	events := make([]*pb.Event, 0, len(s.events))
	for _, event := range s.events {
		events = append(events, event)
	}

	return events, nil
}

// CleanupExpiredEvents removes events that have exceeded their TTL
func (s *InMemoryEventStore) CleanupExpiredEvents() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-s.ttl)

	for id, event := range s.events {
		if event.Timestamp.AsTime().Before(cutoff) {
			delete(s.events, id)
		}
	}

	return nil
}

// cleanupRoutine runs periodically to clean up expired events
func (s *InMemoryEventStore) cleanupRoutine() {
	ticker := time.NewTicker(s.ttl / 4) // Clean up every 1/4 of TTL
	defer ticker.Stop()

	for range ticker.C {
		if err := s.CleanupExpiredEvents(); err != nil {
			// In a real implementation, you might want to log this error
			continue
		}
	}
}

// GetEventCount returns the total number of events in the store
func (s *InMemoryEventStore) GetEventCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.events)
}
