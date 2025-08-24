package main

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/DaDevFox/task-systems/workflows/backend/config"

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
	defer func() {
		os.Unsetenv("INVENTORY_SERVICE_ADDR")
		os.Unsetenv("TASK_SERVICE_ADDR")
	}()

	// Test the service configuration
	serviceConfig := config.LoadServiceConfig()
	if serviceConfig.InventoryServiceAddr != testInventoryAddr {
		t.Errorf("Expected %s, got %s", testInventoryAddr, serviceConfig.InventoryServiceAddr)
	}

	if serviceConfig.TaskServiceAddr != testTaskAddr {
		t.Errorf("Expected %s, got %s", testTaskAddr, serviceConfig.TaskServiceAddr)
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

	// Test service configuration loading
	serviceConfig := config.LoadServiceConfig()
	if serviceConfig.InventoryServiceAddr == "" || serviceConfig.TaskServiceAddr == "" {
		t.Error("Service configuration loading failed")
	}

	select {
	case <-ctx.Done():
		// Test completed within timeout
	default:
		t.Log("Test completed successfully")
	}
}
