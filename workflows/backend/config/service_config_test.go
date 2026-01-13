package config

import (
	"os"
	"testing"
)

func TestLoadServiceConfig(t *testing.T) {
	// Test default values
	config := LoadServiceConfig()

	if config.InventoryServiceAddr != "localhost:50053" {
		t.Errorf("expected default inventory service addr to be localhost:50053, got %s", config.InventoryServiceAddr)
	}

	if config.TaskServiceAddr != "localhost:50054" {
		t.Errorf("expected default task service addr to be localhost:50054, got %s", config.TaskServiceAddr)
	}

	if config.MaxRetries != 3 {
		t.Errorf("expected default max retries to be 3, got %d", config.MaxRetries)
	}
}

func TestLoadServiceConfigWithEnvVars(t *testing.T) {
	// Set environment variables
	os.Setenv("INVENTORY_SERVICE_ADDR", "custom-inventory:9999")
	os.Setenv("TASK_SERVICE_ADDR", "custom-task:8888")
	os.Setenv("SERVICE_MAX_RETRIES", "5")
	defer func() {
		os.Unsetenv("INVENTORY_SERVICE_ADDR")
		os.Unsetenv("TASK_SERVICE_ADDR")
		os.Unsetenv("SERVICE_MAX_RETRIES")
	}()

	config := LoadServiceConfig()

	if config.InventoryServiceAddr != "custom-inventory:9999" {
		t.Errorf("expected inventory service addr to be custom-inventory:9999, got %s", config.InventoryServiceAddr)
	}

	if config.TaskServiceAddr != "custom-task:8888" {
		t.Errorf("expected task service addr to be custom-task:8888, got %s", config.TaskServiceAddr)
	}

	if config.MaxRetries != 5 {
		t.Errorf("expected max retries to be 5, got %d", config.MaxRetries)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	tests := []struct {
		key          string
		defaultValue string
		envValue     string
		expected     string
	}{
		{"TEST_KEY_1", "default", "", "default"},
		{"TEST_KEY_2", "default", "custom", "custom"},
	}

	for _, test := range tests {
		if test.envValue != "" {
			os.Setenv(test.key, test.envValue)
			defer os.Unsetenv(test.key)
		}

		result := getEnvOrDefault(test.key, test.defaultValue)
		if result != test.expected {
			t.Errorf("getEnvOrDefault(%s, %s) = %s; expected %s", test.key, test.defaultValue, result, test.expected)
		}
	}
}

func TestGetEnvIntOrDefault(t *testing.T) {
	tests := []struct {
		key          string
		defaultValue int
		envValue     string
		expected     int
	}{
		{"TEST_INT_1", 10, "", 10},
		{"TEST_INT_2", 10, "20", 20},
		{"TEST_INT_3", 10, "invalid", 10}, // Should fall back to default for invalid int
	}

	for _, test := range tests {
		if test.envValue != "" {
			os.Setenv(test.key, test.envValue)
			defer os.Unsetenv(test.key)
		}

		result := getEnvIntOrDefault(test.key, test.defaultValue)
		if result != test.expected {
			t.Errorf("getEnvIntOrDefault(%s, %d) = %d; expected %d", test.key, test.defaultValue, result, test.expected)
		}
	}
}

func TestServiceConfigValidate(t *testing.T) {
	config := LoadServiceConfig()
	err := config.Validate()
	if err != nil {
		t.Errorf("expected validation to pass for default config, got error: %v", err)
	}
}
