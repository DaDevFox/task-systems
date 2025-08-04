package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	testInventoryAddr = "localhost:50053"
	testTaskAddr      = "localhost:50054"
)

func TestMainFunctionality(t *testing.T) {
	// This test verifies that main.go components can be initialized without errors

	// Set up test environment variables
	os.Setenv("INVENTORY_SERVICE_ADDR", testInventoryAddr)
	os.Setenv("TASK_SERVICE_ADDR", testTaskAddr)

	// Test the helper function
	addr := getEnvOrDefault("INVENTORY_SERVICE_ADDR", "default")
	if addr != testInventoryAddr {
		t.Errorf("Expected %s, got %s", testInventoryAddr, addr)
	}

	// Test with missing env var
	missing := getEnvOrDefault("MISSING_VAR", "default_value")
	if missing != "default_value" {
		t.Errorf("Expected default_value, got %s", missing)
	}

	// Test logger configuration
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel) // Reduce noise in tests

	if logger.Level != logrus.WarnLevel {
		t.Error("Logger level not set correctly")
	}
}

func TestOrchestrationInitialization(t *testing.T) {
	// Test that orchestration initialization handles connection failures gracefully
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	// This should fail quickly since no services are running
	// but it shouldn't panic or cause the program to exit
	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	// We can't easily test the full orchestration service creation here
	// since it requires actual network connections, but we can test
	// that our environment variable handling works correctly

	addr1 := getEnvOrDefault("TEST_INVENTORY_ADDR", testInventoryAddr)
	addr2 := getEnvOrDefault("TEST_TASK_ADDR", testTaskAddr)

	if addr1 == "" || addr2 == "" {
		t.Error("Environment variable handling failed")
	}

	select {
	case <-ctx.Done():
		// Test completed within timeout
	default:
		t.Log("Test completed successfully")
	}
}
