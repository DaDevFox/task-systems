package server

import (
	"context"
	"fmt"
	"sync"
	"time"

	pb "github.com/DaDevFox/task-systems/shared/pkg/proto/events/v1"
	"github.com/DaDevFox/task-systems/shared/util"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Subscription represents a client subscription to events
type Subscription struct {
	ID         string
	EventTypes []pb.EventType
	Filters    map[string]string
	Stream     grpc.ServerStreamingServer[pb.SubscribeToEventsResponse]
	done       chan struct{}
	mu         sync.Mutex
}

// NewSubscription creates a new subscription
func NewSubscription(eventTypes []pb.EventType, filters map[string]string, stream grpc.ServerStreamingServer[pb.SubscribeToEventsResponse]) *Subscription {
	return &Subscription{
		ID:         generateSubscriptionID(),
		EventTypes: eventTypes,
		Filters:    filters,
		Stream:     stream,
		done:       make(chan struct{}),
	}
}

// SendEvent sends an event to the subscription if it matches the filters
func (s *Subscription) SendEvent(ctx context.Context, event *pb.Event) error {
	if err := util.ValidateEvent(event); err != nil {
		return fmt.Errorf("invalid event: %w", err)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if subscription is still active
	select {
	case <-s.done:
		return status.Error(codes.Unavailable, "subscription closed")
	default:
	}

	// Check if event type matches
	if !s.matchesEventType(event) {
		return nil // Event type doesn't match, skip
	}

	// Check if filters match
	if !s.matchesFilters(event) {
		return nil // Filters don't match, skip
	}

	// Send event to client
	response := &pb.SubscribeToEventsResponse{
		Event: event,
	}

	if err := s.Stream.Send(response); err != nil {
		return err
	}

	return nil
}

// matchesEventType checks if the event type matches the subscription
func (s *Subscription) matchesEventType(event *pb.Event) bool {
	if len(s.EventTypes) == 0 {
		return true // No specific event types specified, match all
	}

	eventType := event.Type
	for _, subType := range s.EventTypes {
		if subType == eventType {
			return true
		}
	}

	return false
}

// matchesFilters checks if the event matches the subscription filters
func (s *Subscription) matchesFilters(event *pb.Event) bool {
	if len(s.Filters) == 0 {
		return true // No filters specified, match all
	}

	// Check source service filter
	if sourceFilter, exists := s.Filters["source_service"]; exists {
		if event.SourceService != sourceFilter {
			return false
		}
	}

	// Check metadata filters
	for key, expectedValue := range s.Filters {
		if key == "source_service" {
			continue // Already checked above
		}

		if actualValue, exists := event.Metadata[key]; !exists || actualValue != expectedValue {
			return false
		}
	}

	return true
}

// Close closes the subscription
func (s *Subscription) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	select {
	case <-s.done:
		// Already closed
		return
	default:
		close(s.done)
	}
}

// IsClosed returns true if the subscription is closed
func (s *Subscription) IsClosed() bool {
	select {
	case <-s.done:
		return true
	default:
		return false
	}
}

// SubscriptionManager manages all active subscriptions
type SubscriptionManager struct {
	subscriptions map[string]*Subscription
	mu            sync.RWMutex
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscriptions: make(map[string]*Subscription),
	}
}

// AddSubscription adds a new subscription
func (sm *SubscriptionManager) AddSubscription(sub *Subscription) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.subscriptions[sub.ID] = sub
}

// RemoveSubscription removes a subscription
func (sm *SubscriptionManager) RemoveSubscription(id string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	if sub, exists := sm.subscriptions[id]; exists {
		sub.Close()
		delete(sm.subscriptions, id)
	}
}

// BroadcastEvent sends an event to all matching subscriptions
func (sm *SubscriptionManager) BroadcastEvent(ctx context.Context, event *pb.Event) error {
	sm.mu.RLock()
	subs := make([]*Subscription, 0, len(sm.subscriptions))
	for _, sub := range sm.subscriptions {
		subs = append(subs, sub)
	}
	sm.mu.RUnlock()

	var lastErr error
	for _, sub := range subs {
		if err := sub.SendEvent(ctx, event); err != nil {
			lastErr = err
			// Remove failed subscription
			sm.RemoveSubscription(sub.ID)
		}
	}

	return lastErr
}

// GetSubscriptionCount returns the number of active subscriptions
func (sm *SubscriptionManager) GetSubscriptionCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()
	return len(sm.subscriptions)
}

// CleanupClosedSubscriptions removes all closed subscriptions
func (sm *SubscriptionManager) CleanupClosedSubscriptions() {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	for id, sub := range sm.subscriptions {
		if sub.IsClosed() {
			delete(sm.subscriptions, id)
		}
	}
}

// generateSubscriptionID generates a unique subscription ID
func generateSubscriptionID() string {
	// In a real implementation, you might use UUID or similar
	// For MVP, we'll use a simple counter
	return "sub_" + time.Now().Format("20060102150405") + "_" + fmt.Sprintf("%d", time.Now().UnixNano()%1000)
}
