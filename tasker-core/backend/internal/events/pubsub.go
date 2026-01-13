package events

import (
	"context"
	"sync"

	"github.com/sirupsen/logrus"
)

// EventType represents different types of events
type EventType string

const (
	EventTaskCreated   EventType = "task.created"
	EventTaskUpdated   EventType = "task.updated"
	EventTaskCompleted EventType = "task.completed"
	EventTaskDeleted   EventType = "task.deleted"
	EventCalendarSync  EventType = "calendar.sync"
	EventEmailSent     EventType = "email.sent"
	EventStageChanged  EventType = "task.stage_changed"
)

// Event represents a system event
type Event struct {
	Type      EventType              `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp int64                  `json:"timestamp"`
	UserID    string                 `json:"user_id,omitempty"`
}

// Handler is a function that handles events
type Handler func(ctx context.Context, event Event) error

// PubSub provides simple in-memory publish/subscribe functionality
type PubSub struct {
	mu       sync.RWMutex
	handlers map[EventType][]Handler
	logger   *logrus.Logger
}

// NewPubSub creates a new PubSub instance
func NewPubSub(logger *logrus.Logger) *PubSub {
	if logger == nil {
		logger = logrus.New()
	}

	return &PubSub{
		handlers: make(map[EventType][]Handler),
		logger:   logger,
	}
}

// Subscribe registers a handler for an event type
func (ps *PubSub) Subscribe(eventType EventType, handler Handler) {
	if handler == nil {
		ps.logger.WithField("event_type", eventType).Warn("attempted to subscribe nil handler")
		return
	}

	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.handlers[eventType] = append(ps.handlers[eventType], handler)

	ps.logger.WithFields(logrus.Fields{
		"event_type":    eventType,
		"handler_count": len(ps.handlers[eventType]),
	}).Debug("handler subscribed to event")
}

// Publish sends an event to all registered handlers
func (ps *PubSub) Publish(ctx context.Context, event Event) {
	if event.Type == "" {
		ps.logger.Warn("attempted to publish event with empty type")
		return
	}

	ps.mu.RLock()
	handlers, exists := ps.handlers[event.Type]
	ps.mu.RUnlock()

	if !exists || len(handlers) == 0 {
		ps.logger.WithField("event_type", event.Type).Debug("no handlers for event type")
		return
	}

	ps.logger.WithFields(logrus.Fields{
		"event_type":    event.Type,
		"handler_count": len(handlers),
		"user_id":       event.UserID,
	}).Debug("publishing event")

	// Execute handlers in goroutines to avoid blocking
	for _, handler := range handlers {
		go func(h Handler) {
			if err := h(ctx, event); err != nil {
				ps.logger.WithFields(logrus.Fields{
					"event_type": event.Type,
					"user_id":    event.UserID,
					"error":      err.Error(),
				}).Error("handler failed to process event")
			}
		}(handler)
	}
}

// GetHandlerCount returns the number of handlers for an event type
func (ps *PubSub) GetHandlerCount(eventType EventType) int {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	return len(ps.handlers[eventType])
}

// Clear removes all handlers for an event type, or all handlers if eventType is empty
func (ps *PubSub) Clear(eventType EventType) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	if eventType == "" {
		ps.handlers = make(map[EventType][]Handler)
		ps.logger.Info("cleared all event handlers")
	} else {
		delete(ps.handlers, eventType)
		ps.logger.WithField("event_type", eventType).Debug("cleared handlers for event type")
	}
}
