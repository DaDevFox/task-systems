package domain

import (
	"testing"
)

func TestInventoryItemIsLowStock(t *testing.T) {
	tests := []struct {
		name     string
		item     *InventoryItem
		expected bool
	}{
		{
			name: "item below threshold should be low stock",
			item: &InventoryItem{
				CurrentLevel:      5.0,
				LowStockThreshold: 10.0,
			},
			expected: true,
		},
		{
			name: "item at threshold should be low stock",
			item: &InventoryItem{
				CurrentLevel:      10.0,
				LowStockThreshold: 10.0,
			},
			expected: true,
		},
		{
			name: "item above threshold should not be low stock",
			item: &InventoryItem{
				CurrentLevel:      15.0,
				LowStockThreshold: 10.0,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.item.IsLowStock()
			if result != tt.expected {
				t.Errorf("IsLowStock() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestInventoryItemIsEmpty(t *testing.T) {
	tests := []struct {
		name     string
		level    float64
		expected bool
	}{
		{
			name:     "zero level should be empty",
			level:    0.0,
			expected: true,
		},
		{
			name:     "negative level should be empty",
			level:    -1.0,
			expected: true,
		},
		{
			name:     "positive level should not be empty",
			level:    5.0,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &InventoryItem{CurrentLevel: tt.level}
			result := item.IsEmpty()
			if result != tt.expected {
				t.Errorf("IsEmpty() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestInventoryItemGetCapacityUtilization(t *testing.T) {
	tests := []struct {
		name         string
		currentLevel float64
		maxCapacity  float64
		expected     float64
	}{
		{
			name:         "half capacity",
			currentLevel: 50.0,
			maxCapacity:  100.0,
			expected:     50.0,
		},
		{
			name:         "full capacity",
			currentLevel: 100.0,
			maxCapacity:  100.0,
			expected:     100.0,
		},
		{
			name:         "empty",
			currentLevel: 0.0,
			maxCapacity:  100.0,
			expected:     0.0,
		},
		{
			name:         "zero max capacity",
			currentLevel: 50.0,
			maxCapacity:  0.0,
			expected:     0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			item := &InventoryItem{
				CurrentLevel: tt.currentLevel,
				MaxCapacity:  tt.maxCapacity,
			}
			result := item.GetCapacityUtilization()
			if result != tt.expected {
				t.Errorf("GetCapacityUtilization() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
