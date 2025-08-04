package prediction

import (
	"math"
	"time"
)

// TrainingStage represents the current training phase
type TrainingStage int

const (
	TrainingStageCollecting TrainingStage = iota
	TrainingStageLearning
	TrainingStageTrained
	TrainingStageRetraining
)

// PredictionModel represents available prediction algorithms
type PredictionModel string

const (
	ModelMarkov       PredictionModel = "Markov"
	ModelCroston      PredictionModel = "Croston"
	ModelDriftImpulse PredictionModel = "DriftImpulse"
	ModelBayesian     PredictionModel = "Bayesian"
	ModelMemoryWindow PredictionModel = "MemoryWindow"
	ModelEventTrigger PredictionModel = "EventTrigger"
)

// InventoryEstimate represents a prediction with uncertainty bounds and optional metadata.
type InventoryEstimate struct {
	ItemName       string
	Estimate       float64
	LowerBound     float64
	UpperBound     float64
	NextCheck      time.Time
	Confidence     float64 // 0 to 1
	Recommendation string
	ModelUsed      PredictionModel
}

// TrainingStatus tracks the training progress for a prediction model
type TrainingStatus struct {
	Stage            TrainingStage
	SamplesCollected int
	MinSamples       int
	Accuracy         float64
	LastUpdated      time.Time
	Parameters       map[string]float64
}

// Predictor defines the interface that all prediction strategies must implement.
type Predictor interface {
	Predict(nextReportTime time.Time) InventoryEstimate
	Update(report InventoryReport)
	Name() string
	
	// Training capabilities
	StartTraining(minSamples int, parameters map[string]float64) error
	GetTrainingStatus() TrainingStatus
	IsTrainingComplete() bool
	
	// Model management
	GetModel() PredictionModel
	SetParameters(params map[string]float64) error
	GetParameters() map[string]float64
}

// InventoryReport represents an observation of the inventory at a time.
type InventoryReport struct {
	ItemName  string
	Timestamp time.Time
	Level     float64
	Context   string // Optional context like "after dinner", "weekend"
	Metadata  map[string]string
}

// 1. MarkovPredictor
// Discrete state tracking with transition probabilities.
type MarkovPredictor struct {
	ItemName     string
	State        string                        // e.g. "Stocked", "Low", etc.
	Transitions  map[string]map[string]float64 // from -> to -> prob
	LastReport   InventoryReport
	StateHistory []StateTransition // Training data
	
	// Training fields
	trainingStage  TrainingStage
	minSamples     int
	parameters     map[string]float64
	lastUpdated    time.Time
}

type StateTransition struct {
	FromState string
	ToState   string
	Timestamp time.Time
	Level     float64
}

func NewMarkovPredictor(itemName string) *MarkovPredictor {
	return &MarkovPredictor{
		ItemName:     itemName,
		State:        "Unknown",
		Transitions:  make(map[string]map[string]float64),
		StateHistory: make([]StateTransition, 0),
		trainingStage: TrainingStageCollecting,
		parameters:   make(map[string]float64),
		lastUpdated:  time.Now(),
	}
}

func (m *MarkovPredictor) StartTraining(minSamples int, parameters map[string]float64) error {
	m.minSamples = minSamples
	m.parameters = parameters
	m.trainingStage = TrainingStageCollecting
	m.lastUpdated = time.Now()
	return nil
}

func (m *MarkovPredictor) GetTrainingStatus() TrainingStatus {
	return TrainingStatus{
		Stage:            m.trainingStage,
		SamplesCollected: len(m.StateHistory),
		MinSamples:       m.minSamples,
		Accuracy:         m.calculateAccuracy(),
		LastUpdated:      m.lastUpdated,
		Parameters:       m.parameters,
	}
}

func (m *MarkovPredictor) IsTrainingComplete() bool {
	return m.trainingStage == TrainingStageTrained && len(m.StateHistory) >= m.minSamples
}

func (m *MarkovPredictor) GetModel() PredictionModel {
	return ModelMarkov
}

func (m *MarkovPredictor) SetParameters(params map[string]float64) error {
	m.parameters = params
	return nil
}

func (m *MarkovPredictor) GetParameters() map[string]float64 {
	return m.parameters
}

func (m *MarkovPredictor) calculateAccuracy() float64 {
	if len(m.StateHistory) < 2 {
		return 0.0
	}
	// Simple accuracy calculation based on transition predictions
	correct := 0
	total := len(m.StateHistory) - 1
	for i := 0; i < total; i++ {
		predicted := m.mostLikelyNextState(m.StateHistory[i].FromState)
		if predicted == m.StateHistory[i+1].ToState {
			correct++
		}
	}
	return float64(correct) / float64(total)
}

func (m *MarkovPredictor) Predict(t time.Time) InventoryEstimate {
	if !m.IsTrainingComplete() {
		return InventoryEstimate{
			ItemName:       m.ItemName,
			Estimate:       m.LastReport.Level,
			LowerBound:     m.LastReport.Level * 0.5,
			UpperBound:     m.LastReport.Level * 1.5,
			NextCheck:      t,
			Confidence:     0.3,
			Recommendation: "Insufficient training data",
			ModelUsed:      ModelMarkov,
		}
	}
	
	// Enhanced logic: predict based on current state and transition probabilities
	nextState := m.mostLikelyNextState(m.State)
	predictedLevel := m.levelForState(nextState)
	confidence := m.getStateConfidence(m.State, nextState)
	
	return InventoryEstimate{
		ItemName:       m.ItemName,
		Estimate:       predictedLevel,
		LowerBound:     predictedLevel * (1 - (1-confidence)*0.4),
		UpperBound:     predictedLevel * (1 + (1-confidence)*0.4),
		NextCheck:      t,
		Confidence:     confidence,
		Recommendation: m.generateRecommendation(nextState, predictedLevel),
		ModelUsed:      ModelMarkov,
	}
}

func (m *MarkovPredictor) getStateConfidence(fromState, toState string) float64 {
	if transitions, exists := m.Transitions[fromState]; exists {
		if prob, exists := transitions[toState]; exists {
			return prob
		}
	}
	return 0.5 // Default uncertainty
}

func (m *MarkovPredictor) generateRecommendation(state string, level float64) string {
	switch state {
	case "Depleted":
		return "Immediate restocking required"
	case "Low":
		return "Consider restocking soon"
	case "Stocked":
		return "Inventory levels stable"
	default:
		return "Monitor state transitions"
	}
}

func (m *MarkovPredictor) determineState(level float64) string {
	// Configurable thresholds through parameters
	lowThreshold := m.parameters["low_threshold"]
	if lowThreshold == 0 {
		lowThreshold = 3.0 // default
	}
	
	depletedThreshold := m.parameters["depleted_threshold"] 
	if depletedThreshold == 0 {
		depletedThreshold = 0.5 // default
	}
	
	if level <= depletedThreshold {
		return "Depleted"
	}
	if level <= lowThreshold {
		return "Low"
	}
	return "Stocked"
}

func (m *MarkovPredictor) mostLikelyNextState(current string) string {
	maxProb := -1.0
	best := current
	for state, prob := range m.Transitions[current] {
		if prob > maxProb {
			maxProb = prob
			best = state
		}
	}
	return best
}

func (m *MarkovPredictor) levelForState(state string) float64 {
	switch state {
	case "Stocked":
		return 10
	case "Low":
		return 3
	case "Depleted":
		return 0
	default:
		return 5
	}
}

func (m *MarkovPredictor) Update(report InventoryReport) {
	newState := m.determineState(report.Level)
	
	// Record state transition if we have a previous state
	if m.State != "Unknown" && m.State != newState {
		transition := StateTransition{
			FromState: m.State,
			ToState:   newState,
			Timestamp: report.Timestamp,
			Level:     report.Level,
		}
		m.StateHistory = append(m.StateHistory, transition)
		
		// Update transition probabilities
		m.updateTransitionProbabilities(m.State, newState)
	}
	
	m.State = newState
	m.LastReport = report
	m.lastUpdated = time.Now()
	
	// Check if training should complete
	if m.trainingStage == TrainingStageCollecting && len(m.StateHistory) >= m.minSamples {
		m.trainingStage = TrainingStageLearning
		m.processTrainingData()
		m.trainingStage = TrainingStageTrained
	}
}

func (m *MarkovPredictor) updateTransitionProbabilities(fromState, toState string) {
	if m.Transitions[fromState] == nil {
		m.Transitions[fromState] = make(map[string]float64)
	}
	
	// Count transitions from this state
	totalFromState := 0
	for _, transition := range m.StateHistory {
		if transition.FromState == fromState {
			totalFromState++
		}
	}
	
	// Count transitions to the specific target state
	countToState := 0
	for _, transition := range m.StateHistory {
		if transition.FromState == fromState && transition.ToState == toState {
			countToState++
		}
	}
	
	// Calculate probability
	if totalFromState > 0 {
		m.Transitions[fromState][toState] = float64(countToState) / float64(totalFromState)
	}
}

func (m *MarkovPredictor) processTrainingData() {
	// Recalculate all transition probabilities from scratch
	m.Transitions = make(map[string]map[string]float64)
	
	// Count all transitions
	transitionCounts := make(map[string]map[string]int)
	stateCounts := make(map[string]int)
	
	for _, transition := range m.StateHistory {
		if transitionCounts[transition.FromState] == nil {
			transitionCounts[transition.FromState] = make(map[string]int)
		}
		transitionCounts[transition.FromState][transition.ToState]++
		stateCounts[transition.FromState]++
	}
	
	// Calculate probabilities
	for fromState, toCounts := range transitionCounts {
		if m.Transitions[fromState] == nil {
			m.Transitions[fromState] = make(map[string]float64)
		}
		total := stateCounts[fromState]
		for toState, count := range toCounts {
			m.Transitions[fromState][toState] = float64(count) / float64(total)
		}
	}
}

func (m *MarkovPredictor) Name() string { return "Markov" }

// 2. CrostonPredictor
// Intermittent demand forecaster

type CrostonPredictor struct {
	ItemName       string
	Alpha          float64
	MeanDemand     float64
	MeanInterval   float64
	LastReport     InventoryReport
	LastDemandTS   time.Time
	CountSinceLast float64
}

func (c *CrostonPredictor) Predict(t time.Time) InventoryEstimate {
	forecast := c.MeanDemand / c.MeanInterval
	return InventoryEstimate{
		ItemName:       c.ItemName,
		Estimate:       forecast,
		LowerBound:     forecast * 0.75,
		UpperBound:     forecast * 1.25,
		NextCheck:      t,
		Confidence:     0.6,
		Recommendation: "Refill if below threshold",
	}
}

func (c *CrostonPredictor) Update(report InventoryReport) {
	delta := report.Timestamp.Sub(c.LastReport.Timestamp).Hours() / 24.0 // days
	consumed := c.LastReport.Level - report.Level
	if consumed > 0 {
		c.MeanDemand = c.Alpha*consumed + (1-c.Alpha)*c.MeanDemand
		c.MeanInterval = c.Alpha*delta + (1-c.Alpha)*c.MeanInterval
		c.LastDemandTS = report.Timestamp
	}
	c.LastReport = report
}

func (c *CrostonPredictor) Name() string { return "Croston" }

// 3. DriftImpulsePredictor
// Inventory as a physical system with drift (gradual usage) and impulses (events)
type DriftImpulsePredictor struct {
	ItemName    string
	DriftRate   float64 // units per day
	ImpulseSize float64 // default restock impulse
	LastReport  InventoryReport
}

func (d *DriftImpulsePredictor) Predict(t time.Time) InventoryEstimate {
	days := t.Sub(d.LastReport.Timestamp).Hours() / 24.0
	estimate := math.Max(0, d.LastReport.Level-d.DriftRate*days)
	return InventoryEstimate{
		ItemName:       d.ItemName,
		Estimate:       estimate,
		LowerBound:     estimate * 0.8,
		UpperBound:     estimate * 1.1,
		NextCheck:      t,
		Confidence:     0.7,
		Recommendation: "Predicting steady consumption",
	}
}

func (d *DriftImpulsePredictor) Update(report InventoryReport) {
	delta := report.Timestamp.Sub(d.LastReport.Timestamp).Hours() / 24.0
	deltaVal := report.Level - d.LastReport.Level
	if deltaVal > 0 {
		d.ImpulseSize = deltaVal // restock
	} else {
		usageRate := -deltaVal / delta
		d.DriftRate = 0.5*usageRate + 0.5*d.DriftRate
	}
	d.LastReport = report
}

func (d *DriftImpulsePredictor) Name() string { return "DriftImpulse" }
