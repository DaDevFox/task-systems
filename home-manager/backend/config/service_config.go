package config

import (
	"os"
	"strconv"
)

// ServiceConfig holds configuration for external services
type ServiceConfig struct {
	InventoryServiceAddr string
	TaskServiceAddr      string
	GRPCPort             string
	HTTPPort             string
	DashboardPort        string
	MaxRetries           int
	TimeoutSeconds       int
}

// LoadServiceConfig loads service configuration from environment variables with defaults
func LoadServiceConfig() *ServiceConfig {
	return &ServiceConfig{
		InventoryServiceAddr: getEnvOrDefault("INVENTORY_SERVICE_ADDR", "localhost:50053"),
		TaskServiceAddr:      getEnvOrDefault("TASK_SERVICE_ADDR", "localhost:50054"),
		GRPCPort:             getEnvOrDefault("GRPC_PORT", "50051"),
		HTTPPort:             getEnvOrDefault("HTTP_PORT", "8080"),
		DashboardPort:        getEnvOrDefault("DASHBOARD_PORT", "8082"),
		MaxRetries:           getEnvIntOrDefault("SERVICE_MAX_RETRIES", 3),
		TimeoutSeconds:       getEnvIntOrDefault("SERVICE_TIMEOUT_SECONDS", 5),
	}
}

// Validate checks if the configuration is valid
func (c *ServiceConfig) Validate() error {
	// Add validation logic here if needed
	return nil
}

// getEnvOrDefault returns the environment variable value or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getEnvIntOrDefault returns the environment variable value as int or a default value
func getEnvIntOrDefault(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}
