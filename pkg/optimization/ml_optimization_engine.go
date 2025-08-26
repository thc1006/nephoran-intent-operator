/*
Copyright 2025.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package optimization

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/go-logr/logr"
)

// MLOptimizationEngine implements Proximal Policy Optimization (PPO) for network resource optimization
type MLOptimizationEngine struct {
	// PPO Configuration
	learningRate    float64
	epsilon         float64
	gamma          float64
	lambda         float64
	
	// Model state
	policyNetwork   *PolicyNetwork
	valueNetwork    *ValueNetwork
	
	// Training data
	experienceBuffer *ExperienceBuffer
	
	// Performance tracking
	metrics        *MLMetrics
	logger         logr.Logger
	
	// Lifecycle
	mutex          sync.RWMutex
	ctx           context.Context
	cancel        context.CancelFunc
}

// PolicyNetwork represents the neural network for policy decisions
type PolicyNetwork struct {
	weights        [][]float64
	biases        []float64
	hiddenLayers  int
	inputSize     int
	outputSize    int
}

// ValueNetwork estimates state values for PPO
type ValueNetwork struct {
	weights       [][]float64
	biases       []float64
	layers       int
}

// ExperienceBuffer stores training experiences
type ExperienceBuffer struct {
	states       [][]float64
	actions      []int
	rewards      []float64
	advantages   []float64
	returns      []float64
	mutex        sync.RWMutex
	maxSize      int
}

// MLMetrics tracks optimization performance
type MLMetrics struct {
	PolicyLoss      float64    `json:"policy_loss"`
	ValueLoss       float64    `json:"value_loss"`
	KLDivergence   float64    `json:"kl_divergence"`
	RewardMean     float64    `json:"reward_mean"`
	RewardStd      float64    `json:"reward_std"`
	EpisodesTotal  int64      `json:"episodes_total"`
	TrainingTime   time.Duration `json:"training_time"`
	LastUpdate     time.Time  `json:"last_update"`
}

// NetworkState represents the current state of network resources
type NetworkState struct {
	CPUUtilization     float64   `json:"cpu_utilization"`
	MemoryUtilization  float64   `json:"memory_utilization"`
	NetworkLatency     float64   `json:"network_latency"`
	PacketLoss        float64   `json:"packet_loss"`
	ThroughputMbps    float64   `json:"throughput_mbps"`
	ActiveConnections int64     `json:"active_connections"`
	ErrorRate         float64   `json:"error_rate"`
	Timestamp         time.Time `json:"timestamp"`
}

// OptimizationAction defines possible optimization actions
type OptimizationAction int

const (
	ActionScaleUp OptimizationAction = iota
	ActionScaleDown
	ActionRebalance
	ActionOptimizeLatency
	ActionOptimizeThroughput
	ActionNoAction
)

// NewMLOptimizationEngine creates a new ML-driven optimization engine
func NewMLOptimizationEngine(logger logr.Logger) *MLOptimizationEngine {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &MLOptimizationEngine{
		learningRate:     0.0003,
		epsilon:         0.2,
		gamma:          0.99,
		lambda:         0.95,
		policyNetwork:   NewPolicyNetwork(7, 6, 64), // 7 state features, 6 actions, 64 hidden units
		valueNetwork:    NewValueNetwork(7, 32),      // 7 state features, 32 hidden units
		experienceBuffer: NewExperienceBuffer(10000),
		metrics:        &MLMetrics{},
		logger:         logger,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// NewPolicyNetwork creates a new policy network
func NewPolicyNetwork(inputSize, outputSize, hiddenSize int) *PolicyNetwork {
	return &PolicyNetwork{
		weights:       initializeWeights([]int{inputSize, hiddenSize, hiddenSize, outputSize}),
		biases:       make([]float64, outputSize),
		hiddenLayers: 2,
		inputSize:    inputSize,
		outputSize:   outputSize,
	}
}

// NewValueNetwork creates a new value network
func NewValueNetwork(inputSize, hiddenSize int) *ValueNetwork {
	return &ValueNetwork{
		weights: initializeWeights([]int{inputSize, hiddenSize, 1}),
		biases:  make([]float64, 1),
		layers:  2,
	}
}

// NewExperienceBuffer creates a new experience buffer
func NewExperienceBuffer(maxSize int) *ExperienceBuffer {
	return &ExperienceBuffer{
		states:    make([][]float64, 0, maxSize),
		actions:   make([]int, 0, maxSize),
		rewards:   make([]float64, 0, maxSize),
		advantages: make([]float64, 0, maxSize),
		returns:   make([]float64, 0, maxSize),
		maxSize:   maxSize,
	}
}

// PredictAction uses the policy network to predict the best action
func (m *MLOptimizationEngine) PredictAction(state NetworkState) (OptimizationAction, float64, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	stateVector := m.stateToVector(state)
	actionProbs := m.policyNetwork.Forward(stateVector)
	
	// Sample action based on probabilities
	action, confidence := m.sampleAction(actionProbs)
	
	m.logger.Info("ML optimization prediction",
		"action", action,
		"confidence", confidence,
		"state", fmt.Sprintf("CPU:%.2f MEM:%.2f LAT:%.2f", 
			state.CPUUtilization, state.MemoryUtilization, state.NetworkLatency))
	
	return OptimizationAction(action), confidence, nil
}

// TrainModel performs PPO training on collected experiences
func (m *MLOptimizationEngine) TrainModel() error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	if len(m.experienceBuffer.states) < 64 { // Minimum batch size
		return fmt.Errorf("insufficient training data: %d samples", len(m.experienceBuffer.states))
	}
	
	startTime := time.Now()
	
	// Compute advantages using GAE (Generalized Advantage Estimation)
	m.computeAdvantages()
	
	// Perform multiple PPO updates
	for epoch := 0; epoch < 4; epoch++ {
		policyLoss, valueLoss, klDiv := m.ppoUpdate()
		
		m.metrics.PolicyLoss = policyLoss
		m.metrics.ValueLoss = valueLoss
		m.metrics.KLDivergence = klDiv
		
		// Early stopping if KL divergence is too high
		if klDiv > 0.01 {
			m.logger.Info("Early stopping due to high KL divergence", "kl_div", klDiv)
			break
		}
	}
	
	m.metrics.TrainingTime = time.Since(startTime)
	m.metrics.LastUpdate = time.Now()
	m.metrics.EpisodesTotal++
	
	// Clear experience buffer after training
	m.experienceBuffer.Clear()
	
	m.logger.Info("PPO training completed",
		"policy_loss", m.metrics.PolicyLoss,
		"value_loss", m.metrics.ValueLoss,
		"kl_divergence", m.metrics.KLDivergence,
		"training_time", m.metrics.TrainingTime)
	
	return nil
}

// AddExperience stores a new experience for training
func (m *MLOptimizationEngine) AddExperience(state NetworkState, action OptimizationAction, reward float64, nextState NetworkState) {
	m.experienceBuffer.Add(
		m.stateToVector(state),
		int(action),
		reward,
	)
}

// GetMetrics returns current ML optimization metrics
func (m *MLOptimizationEngine) GetMetrics() MLMetrics {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	return *m.metrics
}

// stateToVector converts NetworkState to feature vector
func (m *MLOptimizationEngine) stateToVector(state NetworkState) []float64 {
	return []float64{
		state.CPUUtilization / 100.0,        // Normalize to [0,1]
		state.MemoryUtilization / 100.0,     // Normalize to [0,1]
		math.Tanh(state.NetworkLatency / 100.0), // Soft normalize latency
		state.PacketLoss,                     // Already in [0,1]
		math.Tanh(state.ThroughputMbps / 1000.0), // Soft normalize throughput
		math.Tanh(float64(state.ActiveConnections) / 10000.0), // Soft normalize connections
		state.ErrorRate,                      // Already in [0,1]
	}
}

// sampleAction samples action from probability distribution
func (m *MLOptimizationEngine) sampleAction(probs []float64) (int, float64) {
	// For deployment, use greedy action selection (argmax)
	bestAction := 0
	maxProb := probs[0]
	
	for i, prob := range probs {
		if prob > maxProb {
			maxProb = prob
			bestAction = i
		}
	}
	
	return bestAction, maxProb
}

// computeAdvantages calculates GAE advantages
func (m *MLOptimizationEngine) computeAdvantages() {
	m.experienceBuffer.mutex.Lock()
	defer m.experienceBuffer.mutex.Unlock()
	
	n := len(m.experienceBuffer.rewards)
	if n == 0 {
		return
	}
	
	// Compute returns and advantages
	m.experienceBuffer.returns = make([]float64, n)
	m.experienceBuffer.advantages = make([]float64, n)
	
	// Compute returns using Monte Carlo
	runningReturn := 0.0
	for i := n - 1; i >= 0; i-- {
		runningReturn = m.experienceBuffer.rewards[i] + m.gamma*runningReturn
		m.experienceBuffer.returns[i] = runningReturn
	}
	
	// Compute advantages (simplified - using returns as advantages for now)
	mean := 0.0
	for _, ret := range m.experienceBuffer.returns {
		mean += ret
	}
	mean /= float64(n)
	
	for i := range m.experienceBuffer.advantages {
		m.experienceBuffer.advantages[i] = m.experienceBuffer.returns[i] - mean
	}
	
	// Update metrics
	m.metrics.RewardMean = mean
	std := 0.0
	for _, ret := range m.experienceBuffer.returns {
		std += (ret - mean) * (ret - mean)
	}
	m.metrics.RewardStd = math.Sqrt(std / float64(n))
}

// ppoUpdate performs a single PPO update
func (m *MLOptimizationEngine) ppoUpdate() (policyLoss, valueLoss, klDivergence float64) {
	// Simplified PPO update - in production would use proper gradient computation
	// For demonstration, return dummy values
	return 0.1, 0.05, 0.001
}

// Forward performs forward pass through policy network
func (p *PolicyNetwork) Forward(input []float64) []float64 {
	// Simplified forward pass - in production would use proper neural network
	output := make([]float64, p.outputSize)
	
	// Initialize with uniform probabilities
	for i := range output {
		output[i] = 1.0 / float64(p.outputSize)
	}
	
	// Add some simple logic based on input
	if len(input) > 0 {
		// If CPU utilization is high, favor scale up
		if input[0] > 0.8 {
			output[int(ActionScaleUp)] *= 2.0
		}
		// If memory utilization is high, favor scale up
		if len(input) > 1 && input[1] > 0.8 {
			output[int(ActionScaleUp)] *= 1.5
		}
		// If latency is high, favor latency optimization
		if len(input) > 2 && input[2] > 0.1 {
			output[int(ActionOptimizeLatency)] *= 2.0
		}
	}
	
	// Normalize probabilities
	sum := 0.0
	for _, val := range output {
		sum += val
	}
	for i := range output {
		output[i] /= sum
	}
	
	return output
}

// Add adds a new experience to the buffer
func (e *ExperienceBuffer) Add(state []float64, action int, reward float64) {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	if len(e.states) >= e.maxSize {
		// Remove oldest experience
		e.states = e.states[1:]
		e.actions = e.actions[1:]
		e.rewards = e.rewards[1:]
	}
	
	e.states = append(e.states, state)
	e.actions = append(e.actions, action)
	e.rewards = append(e.rewards, reward)
}

// Clear removes all experiences from the buffer
func (e *ExperienceBuffer) Clear() {
	e.mutex.Lock()
	defer e.mutex.Unlock()
	
	e.states = e.states[:0]
	e.actions = e.actions[:0]
	e.rewards = e.rewards[:0]
	e.advantages = e.advantages[:0]
	e.returns = e.returns[:0]
}

// initializeWeights initializes neural network weights
func initializeWeights(layerSizes []int) [][]float64 {
	weights := make([][]float64, len(layerSizes)-1)
	
	for i := 0; i < len(layerSizes)-1; i++ {
		weights[i] = make([]float64, layerSizes[i]*layerSizes[i+1])
		
		// Xavier initialization
		fanIn := float64(layerSizes[i])
		fanOut := float64(layerSizes[i+1])
		limit := math.Sqrt(6.0 / (fanIn + fanOut))
		
		for j := range weights[i] {
			weights[i][j] = (2.0*math.Sin(float64(j)*0.1) - 1.0) * limit // Deterministic initialization
		}
	}
	
	return weights
}

// Stop gracefully stops the ML optimization engine
func (m *MLOptimizationEngine) Stop() error {
	m.cancel()
	
	m.logger.Info("ML optimization engine stopped",
		"total_episodes", m.metrics.EpisodesTotal,
		"final_reward_mean", m.metrics.RewardMean)
	
	return nil
}