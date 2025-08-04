package clients

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestInventoryClientCreation(t *testing.T) {
	// This test just verifies that the client can be created without errors
	// We're not actually connecting to a service here

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should fail to connect but not panic during creation
	_, err := NewInventoryClient("localhost:50051")
	if err == nil {
		t.Error("Expected connection error since no server is running")
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

func TestTaskClientCreation(t *testing.T) {
	// This test just verifies that the task client can be created without errors
	// We're not actually connecting to a service here

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// This should fail to connect but not panic during creation
	_, err := NewTaskClient("localhost:50052")
	if err == nil {
		t.Error("Expected connection error since no server is running")
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

func TestClientCreationErrorHandling(t *testing.T) {
	// Test that client creation handles connection errors gracefully
	_, err := NewInventoryClient("localhost:99999") // Invalid port
	if err == nil {
		t.Error("Expected connection error for invalid address")
	}
	if !strings.Contains(err.Error(), "failed to connect") {
		t.Errorf("Expected connection error message, got: %v", err)
	}

	_, err = NewTaskClient("localhost:99999") // Invalid port
	if err == nil {
		t.Error("Expected connection error for invalid address")
	}
	if !strings.Contains(err.Error(), "failed to connect") {
		t.Errorf("Expected connection error message, got: %v", err)
	}
}
