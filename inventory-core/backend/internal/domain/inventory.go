package domain

import (
	"encoding/json"
	"fmt"
	"time"

	pb "github.com/DaDevFox/task-systems/inventory-core/backend/pkg/proto/inventory/v1"
	"google.golang.org/protobuf/encoding/protojson"
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

// Unit represents a measurement unit with conversion capabilities
type Unit struct {
	ID                   string            `json:"id"`
	Name                 string            `json:"name"`
	Symbol               string            `json:"symbol"`
	Description          string            `json:"description,omitempty"`
	BaseConversionFactor float64           `json:"base_conversion_factor"`
	Category             string            `json:"category,omitempty"`
	CreatedAt            time.Time         `json:"created_at"`
	UpdatedAt            time.Time         `json:"updated_at"`
	Metadata             map[string]string `json:"metadata,omitempty"`
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

// InventoryLevelSnapshot represents inventory level at a specific point in time
type InventoryLevelSnapshot struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     float64           `json:"level"`
	UnitID    string            `json:"unit_id"`
	Source    string            `json:"source"`   // e.g., "user_report", "system_update"
	Context   string            `json:"context"`  // optional context
	Metadata  map[string]string `json:"metadata"` // additional metadata
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

// UnitNotFoundError represents an error when a unit is not found
type UnitNotFoundError struct {
	ID string
}

func (e *UnitNotFoundError) Error() string {
	return fmt.Sprintf("unit with ID '%s' not found", e.ID)
}

// Custom JSON marshaling methods to handle protobuf oneof fields

// MarshalJSON implements custom JSON marshaling for InventoryItem
func (i *InventoryItem) MarshalJSON() ([]byte, error) {
	// Create a temporary struct for marshaling without the protobuf field
	type TempItem struct {
		ID                    string               `json:"id"`
		Name                  string               `json:"name"`
		Description           string               `json:"description,omitempty"`
		CurrentLevel          float64              `json:"current_level"`
		MaxCapacity           float64              `json:"max_capacity"`
		LowStockThreshold     float64              `json:"low_stock_threshold"`
		UnitID                string               `json:"unit_id"`
		AlternateUnitIDs      []string             `json:"alternate_unit_ids,omitempty"`
		ConsumptionBehavior   *ConsumptionBehavior `json:"consumption_behavior,omitempty"`
		CreatedAt             time.Time            `json:"created_at"`
		UpdatedAt             time.Time            `json:"updated_at"`
		Metadata              map[string]string    `json:"metadata,omitempty"`
		ActivePredictionModel json.RawMessage      `json:"active_prediction_model,omitempty"`
	}

	temp := TempItem{
		ID:                  i.ID,
		Name:                i.Name,
		Description:         i.Description,
		CurrentLevel:        i.CurrentLevel,
		MaxCapacity:         i.MaxCapacity,
		LowStockThreshold:   i.LowStockThreshold,
		UnitID:              i.UnitID,
		AlternateUnitIDs:    i.AlternateUnitIDs,
		ConsumptionBehavior: i.ConsumptionBehavior,
		CreatedAt:           i.CreatedAt,
		UpdatedAt:           i.UpdatedAt,
		Metadata:            i.Metadata,
	}

	// Handle the protobuf field with protojson
	if i.ActivePredictionModel != nil {
		protoJSON, err := protojson.Marshal(i.ActivePredictionModel)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal ActivePredictionModel: %w", err)
		}
		temp.ActivePredictionModel = json.RawMessage(protoJSON)
	}

	return json.Marshal(temp)
}

// UnmarshalJSON implements custom JSON unmarshaling for InventoryItem
func (i *InventoryItem) UnmarshalJSON(data []byte) error {
	// Create a temporary struct for unmarshaling without the protobuf field
	type TempItem struct {
		ID                    string               `json:"id"`
		Name                  string               `json:"name"`
		Description           string               `json:"description,omitempty"`
		CurrentLevel          float64              `json:"current_level"`
		MaxCapacity           float64              `json:"max_capacity"`
		LowStockThreshold     float64              `json:"low_stock_threshold"`
		UnitID                string               `json:"unit_id"`
		AlternateUnitIDs      []string             `json:"alternate_unit_ids,omitempty"`
		ConsumptionBehavior   *ConsumptionBehavior `json:"consumption_behavior,omitempty"`
		CreatedAt             time.Time            `json:"created_at"`
		UpdatedAt             time.Time            `json:"updated_at"`
		Metadata              map[string]string    `json:"metadata,omitempty"`
		ActivePredictionModel json.RawMessage      `json:"active_prediction_model,omitempty"`
	}

	var temp TempItem
	if err := json.Unmarshal(data, &temp); err != nil {
		return fmt.Errorf("failed to unmarshal InventoryItem: %w", err)
	}

	// Copy all the regular fields
	i.ID = temp.ID
	i.Name = temp.Name
	i.Description = temp.Description
	i.CurrentLevel = temp.CurrentLevel
	i.MaxCapacity = temp.MaxCapacity
	i.LowStockThreshold = temp.LowStockThreshold
	i.UnitID = temp.UnitID
	i.AlternateUnitIDs = temp.AlternateUnitIDs
	i.ConsumptionBehavior = temp.ConsumptionBehavior
	i.CreatedAt = temp.CreatedAt
	i.UpdatedAt = temp.UpdatedAt
	i.Metadata = temp.Metadata

	// Handle the protobuf field with protojson
	if len(temp.ActivePredictionModel) > 0 {
		i.ActivePredictionModel = &pb.PredictionModelConfig{}
		if err := protojson.Unmarshal(temp.ActivePredictionModel, i.ActivePredictionModel); err != nil {
			// If unmarshaling fails, log and continue without the prediction model
			// This provides backward compatibility
			i.ActivePredictionModel = nil
		}
	}

	return nil
}
