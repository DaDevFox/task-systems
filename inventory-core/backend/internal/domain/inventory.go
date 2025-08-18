package domain

import (
	"fmt"
	"time"

	pb "github.com/DaDevFox/task-systems/inventory-core/pkg/proto/inventory/v1"
)

// InventoryItem represents the core business entity for inventory tracking
type InventoryItem struct {
	ID                    string                    `json:"id"`
	Name                  string                    `json:"name"`
	Description           string                    `json:"description,omitempty"`
	CurrentLevel          float64                   `json:"current_level"`
	MaxCapacity           float64                   `json:"max_capacity"`
	LowStockThreshold     float64                   `json:"low_stock_threshold"`
	UnitID                string                    `json:"unit_id"`
	AlternateUnitIDs      []string                  `json:"alternate_unit_ids,omitempty"`
	ConsumptionBehavior   *ConsumptionBehavior      `json:"consumption_behavior,omitempty"`
	ConsumptionHistory    []ConsumptionRecord       `json:"consumption_history,omitempty"`
	CreatedAt             time.Time                 `json:"created_at"`
	UpdatedAt             time.Time                 `json:"updated_at"`
	Metadata              map[string]string         `json:"metadata,omitempty"`
	ActivePredictionModel *pb.PredictionModelConfig `json:"active_prediction_model,omitempty"`
}

// ConsumptionBehavior defines how an item is consumed over time
type ConsumptionBehavior struct {
	Pattern           ConsumptionPattern `json:"pattern"`
	AverageRatePerDay float64            `json:"average_rate_per_day"`
	Variance          float64            `json:"variance"`
	SeasonalFactors   []float64          `json:"seasonal_factors,omitempty"` // 12 values for months
	LastUpdated       time.Time          `json:"last_updated"`
}

// ConsumptionPattern defines consumption behavior types
type ConsumptionPattern int

const (
	ConsumptionPatternUnspecified ConsumptionPattern = iota
	ConsumptionPatternLinear
	ConsumptionPatternSeasonal
	ConsumptionPatternBatch
	ConsumptionPatternRandom
)

// ConsumptionRecord tracks individual consumption events
type ConsumptionRecord struct {
	Timestamp      time.Time `json:"timestamp"`
	AmountConsumed float64   `json:"amount_consumed"`
	UnitID         string    `json:"unit_id"`
	Reason         string    `json:"reason,omitempty"`
}

// Unit represents a measurement unit with conversion capabilities
type Unit struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Symbol               string   `json:"symbol"`
	Type                 UnitType `json:"type"`
	BaseConversionFactor float64  `json:"base_conversion_factor"`
	BaseUnitID           string   `json:"base_unit_id"`
}

// UnitType defines the type of measurement
type UnitType int

const (
	UnitTypeUnspecified UnitType = iota
	UnitTypeWeight
	UnitTypeVolume
	UnitTypeCount
	UnitTypeLength
	UnitTypeArea
)

// ConsumptionPrediction provides forecasted usage for an item
type ConsumptionPrediction struct {
	ItemID                  string    `json:"item_id"`
	PredictedDaysRemaining  float64   `json:"predicted_days_remaining"`
	ConfidenceScore         float64   `json:"confidence_score"`
	PredictedEmptyDate      time.Time `json:"predicted_empty_date"`
	RecommendedRestockLevel float64   `json:"recommended_restock_level"`
	PredictionModel         string    `json:"prediction_model"`
}

// IsLowStock checks if the item is below its low stock threshold
func (i *InventoryItem) IsLowStock() bool {
	return i.CurrentLevel <= i.LowStockThreshold
}

// IsEmpty checks if the item has no stock
func (i *InventoryItem) IsEmpty() bool {
	return i.CurrentLevel <= 0
}

// GetCapacityUtilization returns the percentage of capacity being used
func (i *InventoryItem) GetCapacityUtilization() float64 {
	if i.MaxCapacity <= 0 {
		return 0
	}
	return (i.CurrentLevel / i.MaxCapacity) * 100
}

// AddConsumptionRecord adds a consumption event to the history
func (i *InventoryItem) AddConsumptionRecord(amount float64, unitID, reason string) {
	record := ConsumptionRecord{
		Timestamp:      time.Now(),
		AmountConsumed: amount,
		UnitID:         unitID,
		Reason:         reason,
	}

	i.ConsumptionHistory = append(i.ConsumptionHistory, record)
	i.UpdatedAt = time.Now()
}

// GetActivePredictionModel returns the active prediction model, creating a default if none exists
func (i *InventoryItem) GetActivePredictionModel() *pb.PredictionModelConfig {
	if i.ActivePredictionModel == nil {
		i.ActivePredictionModel = i.createDefaultPredictionModel()
	}
	return i.ActivePredictionModel
}

// SetActivePredictionModel updates the active prediction model
func (i *InventoryItem) SetActivePredictionModel(config *pb.PredictionModelConfig) {
	i.ActivePredictionModel = config
	i.UpdatedAt = time.Now()
}

// createDefaultPredictionModel creates a default parametric linear model
func (i *InventoryItem) createDefaultPredictionModel() *pb.PredictionModelConfig {
	// Default to linear consumption model with reasonable parameters
	baseConsumptionRate := -1.0 // 1 unit per day consumption

	// If we have consumption behavior, use that for better defaults
	if i.ConsumptionBehavior != nil && i.ConsumptionBehavior.AverageRatePerDay > 0 {
		baseConsumptionRate = -i.ConsumptionBehavior.AverageRatePerDay
	}

	return &pb.PredictionModelConfig{
		ModelConfig: &pb.PredictionModelConfig_Parametric{
			Parametric: &pb.ParametricModel{
				ModelType: &pb.ParametricModel_Linear{
					Linear: &pb.LinearEquationModel{
						Slope:         baseConsumptionRate,
						BaseLevel:     i.CurrentLevel,
						NoiseVariance: 0.5,
					},
				},
			},
		},
	}
}

// InventoryItemNotFoundError represents an error when an item is not found
type InventoryItemNotFoundError struct {
	ID string
}

func (e *InventoryItemNotFoundError) Error() string {
	return fmt.Sprintf("inventory item with ID '%s' not found", e.ID)
}
