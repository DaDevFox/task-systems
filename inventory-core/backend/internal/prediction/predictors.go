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
// Intermittent demand forecaster with enhanced training capabilities
type CrostonPredictor struct {
	ItemName         string
	Alpha            float64
	MeanDemand       float64
	MeanInterval     float64
	LastReport       InventoryReport
	LastDemandTS     time.Time
	CountSinceLast   float64
	DemandHistory    []DemandEvent
	
	// Training fields
	trainingStage  TrainingStage
	minSamples     int
	parameters     map[string]float64
	lastUpdated    time.Time
}

type DemandEvent struct {
	Timestamp     time.Time
	DemandSize    float64
	IntervalDays  float64
}

func NewCrostonPredictor(itemName string) *CrostonPredictor {
	return &CrostonPredictor{
		ItemName:      itemName,
		Alpha:         0.1, // Default smoothing parameter
		MeanDemand:    1.0,
		MeanInterval:  7.0, // Default weekly consumption
		DemandHistory: make([]DemandEvent, 0),
		trainingStage: TrainingStageCollecting,
		parameters:    map[string]float64{"alpha": 0.1},
		lastUpdated:   time.Now(),
	}
}

func (c *CrostonPredictor) StartTraining(minSamples int, parameters map[string]float64) error {
	c.minSamples = minSamples
	if alpha, exists := parameters["alpha"]; exists {
		c.Alpha = alpha
		c.parameters["alpha"] = alpha
	}
	c.trainingStage = TrainingStageCollecting
	c.lastUpdated = time.Now()
	return nil
}

func (c *CrostonPredictor) GetTrainingStatus() TrainingStatus {
	return TrainingStatus{
		Stage:            c.trainingStage,
		SamplesCollected: len(c.DemandHistory),
		MinSamples:       c.minSamples,
		Accuracy:         c.calculateAccuracy(),
		LastUpdated:      c.lastUpdated,
		Parameters:       c.parameters,
	}
}

func (c *CrostonPredictor) IsTrainingComplete() bool {
	return c.trainingStage == TrainingStageTrained && len(c.DemandHistory) >= c.minSamples
}

func (c *CrostonPredictor) GetModel() PredictionModel {
	return ModelCroston
}

func (c *CrostonPredictor) SetParameters(params map[string]float64) error {
	c.parameters = params
	if alpha, exists := params["alpha"]; exists {
		c.Alpha = alpha
	}
	return nil
}

func (c *CrostonPredictor) GetParameters() map[string]float64 {
	return c.parameters
}

func (c *CrostonPredictor) calculateAccuracy() float64 {
	if len(c.DemandHistory) < 3 {
		return 0.0
	}
	// Calculate prediction accuracy for last few events
	totalError := 0.0
	predictions := 0
	for i := 2; i < len(c.DemandHistory); i++ {
		forecast := c.MeanDemand / c.MeanInterval
		actual := c.DemandHistory[i].DemandSize / c.DemandHistory[i].IntervalDays
		error := math.Abs(forecast - actual) / math.Max(actual, 0.1)
		totalError += error
		predictions++
	}
	if predictions == 0 {
		return 0.0
	}
	accuracy := 1.0 - (totalError / float64(predictions))
	return math.Max(0.0, math.Min(1.0, accuracy))
}

func (c *CrostonPredictor) Predict(t time.Time) InventoryEstimate {
	if !c.IsTrainingComplete() {
		return InventoryEstimate{
			ItemName:       c.ItemName,
			Estimate:       c.LastReport.Level,
			LowerBound:     c.LastReport.Level * 0.5,
			UpperBound:     c.LastReport.Level * 1.5,
			NextCheck:      t,
			Confidence:     0.3,
			Recommendation: "Collecting training data for intermittent demand",
			ModelUsed:      ModelCroston,
		}
	}
	
	forecast := c.MeanDemand / c.MeanInterval
	days := t.Sub(c.LastReport.Timestamp).Hours() / 24.0
	estimate := math.Max(0, c.LastReport.Level - forecast*days)
	
	// Calculate confidence based on variance in demand history
	variance := c.calculateDemandVariance()
	confidence := math.Max(0.4, math.Min(0.9, 1.0 - variance/c.MeanDemand))
	
	return InventoryEstimate{
		ItemName:       c.ItemName,
		Estimate:       estimate,
		LowerBound:     estimate * (1 - (1-confidence)*0.5),
		UpperBound:     estimate * (1 + (1-confidence)*0.5),
		NextCheck:      t,
		Confidence:     confidence,
		Recommendation: c.generateRecommendation(estimate),
		ModelUsed:      ModelCroston,
	}
}

func (c *CrostonPredictor) calculateDemandVariance() float64 {
	if len(c.DemandHistory) < 2 {
		return 0.0
	}
	
	mean := c.MeanDemand
	variance := 0.0
	for _, event := range c.DemandHistory {
		diff := event.DemandSize - mean
		variance += diff * diff
	}
	return variance / float64(len(c.DemandHistory))
}

func (c *CrostonPredictor) generateRecommendation(estimate float64) string {
	if estimate <= 1.0 {
		return "Low stock - consider intermittent restocking"
	}
	if estimate <= 3.0 {
		return "Moderate stock - monitor demand patterns"
	}
	return "Adequate stock for intermittent usage"
}

func (c *CrostonPredictor) Update(report InventoryReport) {
	delta := report.Timestamp.Sub(c.LastReport.Timestamp).Hours() / 24.0 // days
	consumed := c.LastReport.Level - report.Level
	
	if consumed > 0 {
		// Record demand event
		demandEvent := DemandEvent{
			Timestamp:    report.Timestamp,
			DemandSize:   consumed,
			IntervalDays: delta,
		}
		c.DemandHistory = append(c.DemandHistory, demandEvent)
		
		// Update exponentially weighted moving averages
		c.MeanDemand = c.Alpha*consumed + (1-c.Alpha)*c.MeanDemand
		c.MeanInterval = c.Alpha*delta + (1-c.Alpha)*c.MeanInterval
		c.LastDemandTS = report.Timestamp
	}
	
	c.LastReport = report
	c.lastUpdated = time.Now()
	
	// Check if training should complete
	if c.trainingStage == TrainingStageCollecting && len(c.DemandHistory) >= c.minSamples {
		c.trainingStage = TrainingStageLearning
		c.optimizeParameters()
		c.trainingStage = TrainingStageTrained
	}
}

func (c *CrostonPredictor) optimizeParameters() {
	// Simple parameter optimization based on historical accuracy
	bestAlpha := c.Alpha
	bestAccuracy := c.calculateAccuracy()
	
	// Test different alpha values
	testAlphas := []float64{0.05, 0.1, 0.15, 0.2, 0.25, 0.3}
	for _, alpha := range testAlphas {
		oldAlpha := c.Alpha
		c.Alpha = alpha
		accuracy := c.calculateAccuracy()
		if accuracy > bestAccuracy {
			bestAccuracy = accuracy
			bestAlpha = alpha
		}
		c.Alpha = oldAlpha
	}
	
	c.Alpha = bestAlpha
	c.parameters["alpha"] = bestAlpha
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
