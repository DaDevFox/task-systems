package orchestration

import (
	"context"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

func TestOrchestrationServiceCreation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test creating orchestration service (should fail to connect but not panic)
	_, err := NewOrchestrationService("localhost:50051", "localhost:50052", logger)
	if err == nil {
		t.Error("Expected connection error since no services are running")
	}

	// The error should be a connection timeout, not a compilation error
	select {
	case <-ctx.Done():
		// Expected - connection should timeout
	default:
		// If we get here quickly, it means there was a different error
		t.Logf("Connection failed as expected: %v", err)
	}
}

func TestEventHandlerCreation(t *testing.T) {
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	// This should work even without a real orchestration service
	// since we're just testing the handler creation
	handler := NewEventHandler(nil, logger)
	if handler == nil {
		t.Error("Expected handler to be created")
	}

	if handler.logger != logger {
		t.Error("Expected logger to be set correctly")
	}
}
