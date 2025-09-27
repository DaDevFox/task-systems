package events

import (
	"context"

	"github.com/sirupsen/logrus"
)

// EventBridge is a placeholder interface for bridging local events to shared event service
// TODO: Implement full bridge when shared event service dependencies are added to go.mod
type EventBridge interface {
	// Publish publishes to both local and shared event systems
	Publish(ctx context.Context, event Event)
	// Subscribe subscribes to local events (maintains existing interface)
	Subscribe(eventType EventType, handler Handler)
	// GetHandlerCount returns handler count from local PubSub
	GetHandlerCount(eventType EventType) int
	// Clear clears handlers from local PubSub
	Clear(eventType EventType)
}

// LocalOnlyBridge provides a local-only implementation for now
type LocalOnlyBridge struct {
	localPubSub *PubSub
	logger      *logrus.Logger
}

// NewLocalOnlyBridge creates a local-only bridge (fallback implementation)
func NewLocalOnlyBridge(localPubSub *PubSub, logger *logrus.Logger) *LocalOnlyBridge {
	return &LocalOnlyBridge{
		localPubSub: localPubSub,
		logger:      logger,
	}
}

// Publish publishes to local PubSub only
func (b *LocalOnlyBridge) Publish(ctx context.Context, event Event) {
	if b.localPubSub != nil {
		b.localPubSub.Publish(ctx, event)
	}
}

// Subscribe subscribes to local events
func (b *LocalOnlyBridge) Subscribe(eventType EventType, handler Handler) {
	if b.localPubSub != nil {
		b.localPubSub.Subscribe(eventType, handler)
	}
}

// GetHandlerCount returns handler count from local PubSub
func (b *LocalOnlyBridge) GetHandlerCount(eventType EventType) int {
	if b.localPubSub != nil {
		return b.localPubSub.GetHandlerCount(eventType)
	}
	return 0
}

// Clear clears handlers from local PubSub
func (b *LocalOnlyBridge) Clear(eventType EventType) {
	if b.localPubSub != nil {
		b.localPubSub.Clear(eventType)
	}
}
