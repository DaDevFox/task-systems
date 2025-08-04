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
	trainingStage TrainingStage
	minSamples    int
	parameters    map[string]float64
	lastUpdated   time.Time
}

type StateTransition struct {
	FromState string
	ToState   string
	Timestamp time.Time
	Level     float64
}

func NewMarkovPredictor(itemName string) *MarkovPredictor {
	return &MarkovPredictor{
		ItemName:      itemName,
		State:         "Unknown",
		Transitions:   make(map[string]map[string]float64),
		StateHistory:  make([]StateTransition, 0),
		trainingStage: TrainingStageCollecting,
		parameters:    make(map[string]float64),
		lastUpdated:   time.Now(),
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
	ItemName       string
	Alpha          float64
	MeanDemand     float64
	MeanInterval   float64
	LastReport     InventoryReport
	LastDemandTS   time.Time
	CountSinceLast float64
	DemandHistory  []DemandEvent

	// Training fields
	trainingStage TrainingStage
	minSamples    int
	parameters    map[string]float64
	lastUpdated   time.Time
}

type DemandEvent struct {
	Timestamp    time.Time
	DemandSize   float64
	IntervalDays float64
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
		error := math.Abs(forecast-actual) / math.Max(actual, 0.1)
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
	estimate := math.Max(0, c.LastReport.Level-forecast*days)

	// Calculate confidence based on variance in demand history
	variance := c.calculateDemandVariance()
	confidence := math.Max(0.4, math.Min(0.9, 1.0-variance/c.MeanDemand))

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
	ItemName       string
	DriftRate      float64 // units per day
	ImpulseSize    float64 // default restock impulse
	LastReport     InventoryReport
	DriftHistory   []DriftMeasurement
	ImpulseHistory []ImpulseEvent

	// Training fields
	trainingStage TrainingStage
	minSamples    int
	parameters    map[string]float64
	lastUpdated   time.Time
}

type DriftMeasurement struct {
	Timestamp time.Time
	Rate      float64
	Duration  float64 // days
}

type ImpulseEvent struct {
	Timestamp time.Time
	Size      float64
	Context   string
}

func NewDriftImpulsePredictor(itemName string) *DriftImpulsePredictor {
	return &DriftImpulsePredictor{
		ItemName:       itemName,
		DriftRate:      1.0,  // Default 1 unit per day
		ImpulseSize:    10.0, // Default restock size
		DriftHistory:   make([]DriftMeasurement, 0),
		ImpulseHistory: make([]ImpulseEvent, 0),
		trainingStage:  TrainingStageCollecting,
		parameters:     map[string]float64{"drift_smoothing": 0.5},
		lastUpdated:    time.Now(),
	}
}

func (d *DriftImpulsePredictor) StartTraining(minSamples int, parameters map[string]float64) error {
	d.minSamples = minSamples
	d.parameters = parameters
	d.trainingStage = TrainingStageCollecting
	d.lastUpdated = time.Now()
	return nil
}

func (d *DriftImpulsePredictor) GetTrainingStatus() TrainingStatus {
	totalSamples := len(d.DriftHistory) + len(d.ImpulseHistory)
	return TrainingStatus{
		Stage:            d.trainingStage,
		SamplesCollected: totalSamples,
		MinSamples:       d.minSamples,
		Accuracy:         d.calculateAccuracy(),
		LastUpdated:      d.lastUpdated,
		Parameters:       d.parameters,
	}
}

func (d *DriftImpulsePredictor) IsTrainingComplete() bool {
	totalSamples := len(d.DriftHistory) + len(d.ImpulseHistory)
	return d.trainingStage == TrainingStageTrained && totalSamples >= d.minSamples
}

func (d *DriftImpulsePredictor) GetModel() PredictionModel {
	return ModelDriftImpulse
}

func (d *DriftImpulsePredictor) SetParameters(params map[string]float64) error {
	d.parameters = params
	return nil
}

func (d *DriftImpulsePredictor) GetParameters() map[string]float64 {
	return d.parameters
}

func (d *DriftImpulsePredictor) Predict(t time.Time) InventoryEstimate {
	if !d.IsTrainingComplete() {
		return InventoryEstimate{
			ItemName:       d.ItemName,
			Estimate:       d.LastReport.Level,
			LowerBound:     d.LastReport.Level * 0.7,
			UpperBound:     d.LastReport.Level * 1.3,
			NextCheck:      t,
			Confidence:     0.4,
			Recommendation: "Learning consumption drift patterns",
			ModelUsed:      ModelDriftImpulse,
		}
	}

	days := t.Sub(d.LastReport.Timestamp).Hours() / 24.0
	estimate := math.Max(0, d.LastReport.Level-d.DriftRate*days)

	// Calculate confidence based on drift rate stability
	confidence := d.calculateDriftStability()
	variance := d.calculateDriftVariance()
	errorBound := variance * days

	return InventoryEstimate{
		ItemName:       d.ItemName,
		Estimate:       estimate,
		LowerBound:     math.Max(0, estimate-errorBound),
		UpperBound:     estimate + errorBound,
		NextCheck:      t,
		Confidence:     confidence,
		Recommendation: d.generateRecommendation(estimate, days),
		ModelUsed:      ModelDriftImpulse,
	}
}

func (d *DriftImpulsePredictor) calculateDriftStability() float64 {
	if len(d.DriftHistory) < 3 {
		return 0.5
	}

	// Calculate coefficient of variation for drift rate
	mean := 0.0
	for _, drift := range d.DriftHistory {
		mean += drift.Rate
	}
	mean /= float64(len(d.DriftHistory))

	variance := 0.0
	for _, drift := range d.DriftHistory {
		diff := drift.Rate - mean
		variance += diff * diff
	}
	variance /= float64(len(d.DriftHistory))

	if mean == 0 {
		return 0.5
	}

	cv := math.Sqrt(variance) / mean
	stability := math.Max(0.3, math.Min(0.9, 1.0-cv))
	return stability
}

func (d *DriftImpulsePredictor) calculateDriftVariance() float64 {
	if len(d.DriftHistory) < 2 {
		return 1.0
	}

	variance := 0.0
	mean := d.DriftRate
	for _, drift := range d.DriftHistory {
		diff := drift.Rate - mean
		variance += diff * diff
	}
	return variance / float64(len(d.DriftHistory))
}

func (d *DriftImpulsePredictor) generateRecommendation(estimate float64, daysAhead float64) string {
	if estimate <= 0 {
		return "Predicted depletion - immediate restocking needed"
	}

	daysToEmpty := estimate / d.DriftRate
	if daysToEmpty <= 3 {
		return "Low stock predicted within 3 days"
	}
	if daysToEmpty <= 7 {
		return "Consider restocking within a week"
	}
	return "Predicting steady consumption"
}

func (d *DriftImpulsePredictor) Update(report InventoryReport) {
	delta := report.Timestamp.Sub(d.LastReport.Timestamp).Hours() / 24.0
	deltaVal := report.Level - d.LastReport.Level

	if deltaVal > 0 {
		// Positive change - likely restock impulse
		impulse := ImpulseEvent{
			Timestamp: report.Timestamp,
			Size:      deltaVal,
			Context:   report.Context,
		}
		d.ImpulseHistory = append(d.ImpulseHistory, impulse)
		d.ImpulseSize = deltaVal // Update expected impulse size
	} else if deltaVal < 0 && delta > 0 {
		// Negative change - consumption drift
		usageRate := -deltaVal / delta

		drift := DriftMeasurement{
			Timestamp: report.Timestamp,
			Rate:      usageRate,
			Duration:  delta,
		}
		d.DriftHistory = append(d.DriftHistory, drift)

		// Update drift rate with smoothing
		smoothing := d.parameters["drift_smoothing"]
		if smoothing == 0 {
			smoothing = 0.5
		}
		d.DriftRate = smoothing*usageRate + (1-smoothing)*d.DriftRate
	}

	d.LastReport = report
	d.lastUpdated = time.Now()

	// Check if training should complete
	totalSamples := len(d.DriftHistory) + len(d.ImpulseHistory)
	if d.trainingStage == TrainingStageCollecting && totalSamples >= d.minSamples {
		d.trainingStage = TrainingStageLearning
		d.optimizeDriftParameters()
		d.trainingStage = TrainingStageTrained
	}
}

func (d *DriftImpulsePredictor) optimizeDriftParameters() {
	// Optimize smoothing parameter based on prediction accuracy
	if len(d.DriftHistory) < 3 {
		return
	}

	bestSmoothing := d.parameters["drift_smoothing"]
	bestAccuracy := d.calculateAccuracy()

	testSmoothings := []float64{0.1, 0.3, 0.5, 0.7, 0.9}
	for _, smoothing := range testSmoothings {
		d.parameters["drift_smoothing"] = smoothing
		accuracy := d.calculateAccuracy()
		if accuracy > bestAccuracy {
			bestAccuracy = accuracy
			bestSmoothing = smoothing
		}
	}

	d.parameters["drift_smoothing"] = bestSmoothing
}

func (d *DriftImpulsePredictor) calculateAccuracy() float64 {
	if len(d.DriftHistory) < 2 {
		return 0.0
	}

	// Calculate accuracy based on drift rate predictions
	totalError := 0.0
	predictions := 0

	for i := 1; i < len(d.DriftHistory); i++ {
		predicted := d.DriftRate
		actual := d.DriftHistory[i].Rate
		if actual > 0 {
			error := math.Abs(predicted-actual) / actual
			totalError += error
			predictions++
		}
	}

	if predictions == 0 {
		return 0.0
	}

	accuracy := 1.0 - (totalError / float64(predictions))
	return math.Max(0.0, math.Min(1.0, accuracy))
}

func (d *DriftImpulsePredictor) Name() string { return "DriftImpulse" }

// 4. BayesianPredictor
// Uses Bayesian inference to provide confidence intervals for predictions
type BayesianPredictor struct {
	ItemName      string
	PriorMean     float64   // Prior belief about consumption rate
	PriorVariance float64   // Prior uncertainty
	Observations  []float64 // Observed consumption rates
	LastReport    InventoryReport

	// Training fields
	trainingStage TrainingStage
	minSamples    int
	parameters    map[string]float64
	lastUpdated   time.Time
}

func NewBayesianPredictor(itemName string) *BayesianPredictor {
	return &BayesianPredictor{
		ItemName:      itemName,
		PriorMean:     1.0, // Default consumption rate
		PriorVariance: 1.0, // Default uncertainty
		Observations:  make([]float64, 0),
		trainingStage: TrainingStageCollecting,
		parameters:    map[string]float64{"prior_strength": 1.0},
		lastUpdated:   time.Now(),
	}
}

func (b *BayesianPredictor) StartTraining(minSamples int, parameters map[string]float64) error {
	b.minSamples = minSamples
	b.parameters = parameters
	if priorStrength, exists := parameters["prior_strength"]; exists && priorStrength > 0 {
		b.PriorVariance = 1.0 / priorStrength
	}
	b.trainingStage = TrainingStageCollecting
	b.lastUpdated = time.Now()
	return nil
}

func (b *BayesianPredictor) GetTrainingStatus() TrainingStatus {
	return TrainingStatus{
		Stage:            b.trainingStage,
		SamplesCollected: len(b.Observations),
		MinSamples:       b.minSamples,
		Accuracy:         b.calculateAccuracy(),
		LastUpdated:      b.lastUpdated,
		Parameters:       b.parameters,
	}
}

func (b *BayesianPredictor) IsTrainingComplete() bool {
	return b.trainingStage == TrainingStageTrained && len(b.Observations) >= b.minSamples
}

func (b *BayesianPredictor) GetModel() PredictionModel {
	return ModelBayesian
}

func (b *BayesianPredictor) SetParameters(params map[string]float64) error {
	b.parameters = params
	return nil
}

func (b *BayesianPredictor) GetParameters() map[string]float64 {
	return b.parameters
}

func (b *BayesianPredictor) calculateAccuracy() float64 {
	if len(b.Observations) < 2 {
		return 0.0
	}

	// Calculate posterior predictive accuracy
	posteriorMean := b.calculatePosteriorMean()
	totalError := 0.0

	for _, obs := range b.Observations {
		error := math.Abs(posteriorMean-obs) / math.Max(obs, 0.1)
		totalError += error
	}

	accuracy := 1.0 - (totalError / float64(len(b.Observations)))
	return math.Max(0.0, math.Min(1.0, accuracy))
}

func (b *BayesianPredictor) calculatePosteriorMean() float64 {
	if len(b.Observations) == 0 {
		return b.PriorMean
	}

	// Bayesian update: combine prior with observed data
	priorPrecision := 1.0 / b.PriorVariance
	n := float64(len(b.Observations))
	observationMean := 0.0

	for _, obs := range b.Observations {
		observationMean += obs
	}
	observationMean /= n

	// Assume observation variance of 1.0 for simplicity
	observationPrecision := n

	posteriorPrecision := priorPrecision + observationPrecision
	posteriorMean := (priorPrecision*b.PriorMean + observationPrecision*observationMean) / posteriorPrecision

	return posteriorMean
}

func (b *BayesianPredictor) calculatePosteriorVariance() float64 {
	priorPrecision := 1.0 / b.PriorVariance
	n := float64(len(b.Observations))
	observationPrecision := n // Assume unit variance observations

	posteriorPrecision := priorPrecision + observationPrecision
	return 1.0 / posteriorPrecision
}

func (b *BayesianPredictor) Predict(t time.Time) InventoryEstimate {
	if !b.IsTrainingComplete() {
		return InventoryEstimate{
			ItemName:       b.ItemName,
			Estimate:       b.LastReport.Level,
			LowerBound:     b.LastReport.Level * 0.6,
			UpperBound:     b.LastReport.Level * 1.4,
			NextCheck:      t,
			Confidence:     0.4,
			Recommendation: "Building Bayesian model confidence",
			ModelUsed:      ModelBayesian,
		}
	}

	days := t.Sub(b.LastReport.Timestamp).Hours() / 24.0
	posteriorMean := b.calculatePosteriorMean()
	posteriorVariance := b.calculatePosteriorVariance()

	estimate := math.Max(0, b.LastReport.Level-posteriorMean*days)

	// Calculate confidence intervals using posterior distribution
	stdDev := math.Sqrt(posteriorVariance * days)
	lowerBound := math.Max(0, estimate-1.96*stdDev) // 95% confidence interval
	upperBound := estimate + 1.96*stdDev

	// Confidence based on posterior certainty
	confidence := math.Max(0.4, math.Min(0.95, 1.0-posteriorVariance))

	return InventoryEstimate{
		ItemName:       b.ItemName,
		Estimate:       estimate,
		LowerBound:     lowerBound,
		UpperBound:     upperBound,
		NextCheck:      t,
		Confidence:     confidence,
		Recommendation: b.generateRecommendation(estimate, lowerBound),
		ModelUsed:      ModelBayesian,
	}
}

func (b *BayesianPredictor) generateRecommendation(estimate, lowerBound float64) string {
	if lowerBound <= 0 {
		return "High probability of stockout - restock immediately"
	}
	if estimate <= 2.0 {
		return "Low predicted inventory with uncertainty"
	}
	return "Bayesian prediction suggests adequate stock"
}

func (b *BayesianPredictor) Update(report InventoryReport) {
	delta := report.Timestamp.Sub(b.LastReport.Timestamp).Hours() / 24.0
	if delta > 0 {
		consumed := b.LastReport.Level - report.Level
		if consumed > 0 {
			consumptionRate := consumed / delta
			b.Observations = append(b.Observations, consumptionRate)
		}
	}

	b.LastReport = report
	b.lastUpdated = time.Now()

	// Check if training should complete
	if b.trainingStage == TrainingStageCollecting && len(b.Observations) >= b.minSamples {
		b.trainingStage = TrainingStageLearning
		b.trainingStage = TrainingStageTrained
	}
}

func (b *BayesianPredictor) Name() string { return "Bayesian" }

// 5. MemoryWindowPredictor
// Memory-augmented rolling windows with decay weighting
type MemoryWindowPredictor struct {
	ItemName          string
	WindowSize        int
	DecayFactor       float64
	ConsumptionEvents []ConsumptionEvent
	LastReport        InventoryReport

	// Training fields
	trainingStage TrainingStage
	minSamples    int
	parameters    map[string]float64
	lastUpdated   time.Time
}

type ConsumptionEvent struct {
	Timestamp    time.Time
	Amount       float64
	IntervalDays float64
	Context      string
	Weight       float64
}

func NewMemoryWindowPredictor(itemName string) *MemoryWindowPredictor {
	return &MemoryWindowPredictor{
		ItemName:          itemName,
		WindowSize:        20,   // Default window size
		DecayFactor:       0.05, // Default decay rate
		ConsumptionEvents: make([]ConsumptionEvent, 0),
		trainingStage:     TrainingStageCollecting,
		parameters:        map[string]float64{"decay_factor": 0.05, "window_size": 20},
		lastUpdated:       time.Now(),
	}
}

func (m *MemoryWindowPredictor) StartTraining(minSamples int, parameters map[string]float64) error {
	m.minSamples = minSamples
	m.parameters = parameters

	if decay, exists := parameters["decay_factor"]; exists {
		m.DecayFactor = decay
	}
	if windowSize, exists := parameters["window_size"]; exists {
		m.WindowSize = int(windowSize)
	}

	m.trainingStage = TrainingStageCollecting
	m.lastUpdated = time.Now()
	return nil
}

func (m *MemoryWindowPredictor) GetTrainingStatus() TrainingStatus {
	return TrainingStatus{
		Stage:            m.trainingStage,
		SamplesCollected: len(m.ConsumptionEvents),
		MinSamples:       m.minSamples,
		Accuracy:         m.calculateAccuracy(),
		LastUpdated:      m.lastUpdated,
		Parameters:       m.parameters,
	}
}

func (m *MemoryWindowPredictor) IsTrainingComplete() bool {
	return m.trainingStage == TrainingStageTrained && len(m.ConsumptionEvents) >= m.minSamples
}

func (m *MemoryWindowPredictor) GetModel() PredictionModel {
	return ModelMemoryWindow
}

func (m *MemoryWindowPredictor) SetParameters(params map[string]float64) error {
	m.parameters = params
	if decay, exists := params["decay_factor"]; exists {
		m.DecayFactor = decay
	}
	if windowSize, exists := params["window_size"]; exists {
		m.WindowSize = int(windowSize)
	}
	return nil
}

func (m *MemoryWindowPredictor) GetParameters() map[string]float64 {
	return m.parameters
}

func (m *MemoryWindowPredictor) calculateAccuracy() float64 {
	if len(m.ConsumptionEvents) < 3 {
		return 0.0
	}

	// Calculate weighted prediction accuracy
	totalError := 0.0
	totalWeight := 0.0
	now := time.Now()

	for i := 2; i < len(m.ConsumptionEvents); i++ {
		predicted := m.calculateWeightedRate(m.ConsumptionEvents[i].Timestamp)
		actual := m.ConsumptionEvents[i].Amount / math.Max(m.ConsumptionEvents[i].IntervalDays, 0.1)

		timeDiff := now.Sub(m.ConsumptionEvents[i].Timestamp).Hours() / 24.0
		weight := math.Exp(-m.DecayFactor * timeDiff)

		if actual > 0 {
			error := math.Abs(predicted-actual) / actual
			totalError += error * weight
			totalWeight += weight
		}
	}

	if totalWeight == 0 {
		return 0.0
	}

	accuracy := 1.0 - (totalError / totalWeight)
	return math.Max(0.0, math.Min(1.0, accuracy))
}

func (m *MemoryWindowPredictor) calculateWeightedRate(targetTime time.Time) float64 {
	if len(m.ConsumptionEvents) == 0 {
		return 1.0 // Default rate
	}

	totalWeightedRate := 0.0
	totalWeight := 0.0

	for _, event := range m.ConsumptionEvents {
		timeDiff := targetTime.Sub(event.Timestamp).Hours() / 24.0
		weight := math.Exp(-m.DecayFactor * math.Abs(timeDiff))

		rate := event.Amount / math.Max(event.IntervalDays, 0.1)
		totalWeightedRate += rate * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 1.0
	}

	return totalWeightedRate / totalWeight
}

func (m *MemoryWindowPredictor) Predict(t time.Time) InventoryEstimate {
	if !m.IsTrainingComplete() {
		return InventoryEstimate{
			ItemName:       m.ItemName,
			Estimate:       m.LastReport.Level,
			LowerBound:     m.LastReport.Level * 0.6,
			UpperBound:     m.LastReport.Level * 1.4,
			NextCheck:      t,
			Confidence:     0.4,
			Recommendation: "Learning memory-weighted consumption patterns",
			ModelUsed:      ModelMemoryWindow,
		}
	}

	days := t.Sub(m.LastReport.Timestamp).Hours() / 24.0
	weightedRate := m.calculateWeightedRate(t)
	estimate := math.Max(0, m.LastReport.Level-weightedRate*days)

	// Calculate confidence based on consistency of weighted predictions
	confidence := m.calculatePredictionConsistency()
	variance := m.calculateWeightedVariance(t)

	errorBound := math.Sqrt(variance) * days

	return InventoryEstimate{
		ItemName:       m.ItemName,
		Estimate:       estimate,
		LowerBound:     math.Max(0, estimate-errorBound),
		UpperBound:     estimate + errorBound,
		NextCheck:      t,
		Confidence:     confidence,
		Recommendation: m.generateRecommendation(estimate),
		ModelUsed:      ModelMemoryWindow,
	}
}

func (m *MemoryWindowPredictor) calculatePredictionConsistency() float64 {
	if len(m.ConsumptionEvents) < 3 {
		return 0.5
	}

	// Calculate how consistent recent predictions are
	recentRates := make([]float64, 0)
	now := time.Now()

	for _, event := range m.ConsumptionEvents {
		timeDiff := now.Sub(event.Timestamp).Hours() / 24.0
		if timeDiff <= 30 { // Last 30 days
			rate := event.Amount / math.Max(event.IntervalDays, 0.1)
			recentRates = append(recentRates, rate)
		}
	}

	if len(recentRates) < 2 {
		return 0.5
	}

	// Calculate coefficient of variation
	mean := 0.0
	for _, rate := range recentRates {
		mean += rate
	}
	mean /= float64(len(recentRates))

	variance := 0.0
	for _, rate := range recentRates {
		diff := rate - mean
		variance += diff * diff
	}
	variance /= float64(len(recentRates))

	if mean == 0 {
		return 0.5
	}

	cv := math.Sqrt(variance) / mean
	consistency := math.Max(0.3, math.Min(0.9, 1.0-cv))
	return consistency
}

func (m *MemoryWindowPredictor) calculateWeightedVariance(targetTime time.Time) float64 {
	if len(m.ConsumptionEvents) < 2 {
		return 1.0
	}

	weightedMean := m.calculateWeightedRate(targetTime)
	totalWeightedVariance := 0.0
	totalWeight := 0.0

	for _, event := range m.ConsumptionEvents {
		timeDiff := targetTime.Sub(event.Timestamp).Hours() / 24.0
		weight := math.Exp(-m.DecayFactor * math.Abs(timeDiff))

		rate := event.Amount / math.Max(event.IntervalDays, 0.1)
		diff := rate - weightedMean
		totalWeightedVariance += weight * diff * diff
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 1.0
	}

	return totalWeightedVariance / totalWeight
}

func (m *MemoryWindowPredictor) generateRecommendation(estimate float64) string {
	if estimate <= 1.0 {
		return "Memory pattern suggests low stock approaching"
	}
	if estimate <= 3.0 {
		return "Weighted analysis indicates moderate stock levels"
	}
	return "Memory-weighted prediction shows adequate inventory"
}

func (m *MemoryWindowPredictor) Update(report InventoryReport) {
	delta := report.Timestamp.Sub(m.LastReport.Timestamp).Hours() / 24.0

	if delta > 0 {
		consumed := m.LastReport.Level - report.Level
		if consumed > 0 {
			// Create consumption event
			event := ConsumptionEvent{
				Timestamp:    report.Timestamp,
				Amount:       consumed,
				IntervalDays: delta,
				Context:      report.Context,
				Weight:       1.0, // Will be calculated dynamically
			}

			m.ConsumptionEvents = append(m.ConsumptionEvents, event)

			// Maintain window size
			if len(m.ConsumptionEvents) > m.WindowSize {
				m.ConsumptionEvents = m.ConsumptionEvents[1:]
			}
		}
	}

	m.LastReport = report
	m.lastUpdated = time.Now()

	// Update weights for all events
	m.updateEventWeights()

	// Check if training should complete
	if m.trainingStage == TrainingStageCollecting && len(m.ConsumptionEvents) >= m.minSamples {
		m.trainingStage = TrainingStageLearning
		m.optimizeDecayParameter()
		m.trainingStage = TrainingStageTrained
	}
}

func (m *MemoryWindowPredictor) updateEventWeights() {
	now := time.Now()
	for i := range m.ConsumptionEvents {
		timeDiff := now.Sub(m.ConsumptionEvents[i].Timestamp).Hours() / 24.0
		m.ConsumptionEvents[i].Weight = math.Exp(-m.DecayFactor * timeDiff)
	}
}

func (m *MemoryWindowPredictor) optimizeDecayParameter() {
	if len(m.ConsumptionEvents) < 5 {
		return
	}

	bestDecay := m.DecayFactor
	bestAccuracy := m.calculateAccuracy()

	testDecays := []float64{0.01, 0.02, 0.05, 0.1, 0.15, 0.2}
	for _, decay := range testDecays {
		oldDecay := m.DecayFactor
		m.DecayFactor = decay
		accuracy := m.calculateAccuracy()
		if accuracy > bestAccuracy {
			bestAccuracy = accuracy
			bestDecay = decay
		}
		m.DecayFactor = oldDecay
	}

	m.DecayFactor = bestDecay
	m.parameters["decay_factor"] = bestDecay
}

func (m *MemoryWindowPredictor) Name() string { return "MemoryWindow" }
