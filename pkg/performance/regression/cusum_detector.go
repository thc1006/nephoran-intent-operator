//go:build go1.24

package regression

import (
	"fmt"
	"math"
	"sort"
	"time"

	"gonum.org/v1/gonum/stat"
	"k8s.io/klog/v2"
)

// median calculates the median of a float64 slice
func median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)

	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

// CUSUMDetector implements Cumulative Sum (CUSUM) algorithm for change point detection
// CUSUM is particularly effective for detecting small persistent changes in time series
type CUSUMDetector struct {
	// Algorithm parameters
	threshold      float64 // Decision threshold (h)
	driftFactor    float64 // Drift parameter (k)
	referenceValue float64 // Reference value (μ0)

	// State variables for online detection
	upperCUSUM float64 // Upper CUSUM statistic (S+)
	lowerCUSUM float64 // Lower CUSUM statistic (S-)

	// Configuration
	config *CUSUMConfig

	// Detection history
	detectedChanges []*CUSUMChangePoint
	lastDetection   time.Time
}

// CUSUMConfig configures the CUSUM detector
type CUSUMConfig struct {
	// Detection sensitivity
	ThresholdMultiplier  float64       `json:"thresholdMultiplier"`  // Multiplier for automatic threshold calculation
	DriftSensitivity     float64       `json:"driftSensitivity"`     // Sensitivity to drift (0-1)
	MinDetectionInterval time.Duration `json:"minDetectionInterval"` // Minimum time between detections

	// Statistical parameters
	UseRobustStatistics bool `json:"useRobustStatistics"` // Use median/MAD instead of mean/std
	AdaptiveThreshold   bool `json:"adaptiveThreshold"`   // Adapt threshold based on historical data
	SeasonalAdjustment  bool `json:"seasonalAdjustment"`  // Account for seasonal patterns

	// Performance optimization
	MaxHistorySize  int  `json:"maxHistorySize"`  // Maximum number of historical points to keep
	BatchProcessing bool `json:"batchProcessing"` // Process data in batches vs. online

	// Validation
	ValidationWindowSize   int  `json:"validationWindowSize"`   // Size of window for validating change points
	FalsePositiveReduction bool `json:"falsePositiveReduction"` // Enable false positive reduction techniques
}

// CUSUMChangePoint represents a detected change point with CUSUM-specific information
type CUSUMChangePoint struct {
	*ChangePoint // Embed base change point

	// CUSUM-specific properties
	UpperCUSUM     float64 `json:"upperCUSUM"`     // Upper CUSUM value at detection
	LowerCUSUM     float64 `json:"lowerCUSUM"`     // Lower CUSUM value at detection
	Direction      string  `json:"direction"`      // "increase" or "decrease"
	Magnitude      float64 `json:"magnitude"`      // Magnitude of the change
	DetectionDelay int     `json:"detectionDelay"` // Number of points after actual change

	// Statistical significance
	PValue             float64             `json:"pValue"`             // Statistical significance
	ConfidenceInterval *ConfidenceInterval `json:"confidenceInterval"` // Confidence interval for change magnitude

	// Context information
	PreChangeStatistics  *DescriptiveStatistics `json:"preChangeStatistics"`  // Statistics before change
	PostChangeStatistics *DescriptiveStatistics `json:"postChangeStatistics"` // Statistics after change

	// Validation results
	Validated       bool    `json:"validated"`       // Whether change point has been validated
	ValidationScore float64 `json:"validationScore"` // Validation confidence score
}

// DescriptiveStatistics provides statistical summary
type DescriptiveStatistics struct {
	Mean      float64    `json:"mean"`
	Median    float64    `json:"median"`
	StdDev    float64    `json:"stdDev"`
	MAD       float64    `json:"mad"` // Median Absolute Deviation
	Min       float64    `json:"min"`
	Max       float64    `json:"max"`
	Count     int        `json:"count"`
	Quartiles [3]float64 `json:"quartiles"` // Q1, Q2 (median), Q3
}

// NewCUSUMDetector creates a new CUSUM change point detector
func NewCUSUMDetector(threshold, driftFactor, referenceValue float64, config *CUSUMConfig) *CUSUMDetector {
	if config == nil {
		config = getDefaultCUSUMConfig()
	}

	return &CUSUMDetector{
		threshold:       threshold,
		driftFactor:     driftFactor,
		referenceValue:  referenceValue,
		config:          config,
		detectedChanges: make([]*CUSUMChangePoint, 0),
		upperCUSUM:      0.0,
		lowerCUSUM:      0.0,
	}
}

// DetectChangePoints implements the ChangePointAlgorithm interface
func (cd *CUSUMDetector) DetectChangePoints(data []float64, timestamps []time.Time) ([]*ChangePoint, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("empty data provided")
	}

	if len(data) != len(timestamps) {
		return nil, fmt.Errorf("data and timestamps length mismatch: %d vs %d", len(data), len(timestamps))
	}

	// Initialize detector if needed
	if err := cd.initializeFromData(data); err != nil {
		return nil, fmt.Errorf("failed to initialize detector: %w", err)
	}

	var changePoints []*ChangePoint

	if cd.config.BatchProcessing {
		cusumChangePoints, err := cd.detectChangePointsBatch(data, timestamps)
		if err != nil {
			return nil, err
		}

		// Convert CUSUM change points to generic change points
		for _, cp := range cusumChangePoints {
			changePoints = append(changePoints, cp.ChangePoint)
		}
	} else {
		cusumChangePoints, err := cd.detectChangePointsOnline(data, timestamps)
		if err != nil {
			return nil, err
		}

		// Convert CUSUM change points to generic change points
		for _, cp := range cusumChangePoints {
			changePoints = append(changePoints, cp.ChangePoint)
		}
	}

	// Apply false positive reduction if enabled
	if cd.config.FalsePositiveReduction {
		changePoints = cd.reduceFalsePositives(changePoints, data, timestamps)
	}

	klog.V(2).Infof("CUSUM detector found %d change points", len(changePoints))

	return changePoints, nil
}

// initializeFromData automatically initializes detector parameters from data
func (cd *CUSUMDetector) initializeFromData(data []float64) error {
	if cd.referenceValue == 0 {
		if cd.config.UseRobustStatistics {
			cd.referenceValue = median(data)
		} else {
			cd.referenceValue = stat.Mean(data, nil)
		}
	}

	if cd.threshold == 0 {
		var stdDev float64
		if cd.config.UseRobustStatistics {
			// Use Median Absolute Deviation (MAD)
			medianVal := median(data)
			deviations := make([]float64, len(data))
			for i, v := range data {
				deviations[i] = math.Abs(v - medianVal)
			}
			mad := median(deviations)
			stdDev = mad * 1.4826 // Convert MAD to approximate standard deviation
		} else {
			stdDev = stat.StdDev(data, nil)
		}

		cd.threshold = cd.config.ThresholdMultiplier * stdDev
	}

	if cd.driftFactor == 0 {
		// Set drift factor as fraction of standard deviation
		var stdDev float64
		if cd.config.UseRobustStatistics {
			medianVal := median(data)
			deviations := make([]float64, len(data))
			for i, v := range data {
				deviations[i] = math.Abs(v - medianVal)
			}
			mad := median(deviations)
			stdDev = mad * 1.4826
		} else {
			stdDev = stat.StdDev(data, nil)
		}

		cd.driftFactor = cd.config.DriftSensitivity * stdDev
	}

	klog.V(3).Infof("CUSUM initialized: threshold=%.4f, drift=%.4f, reference=%.4f",
		cd.threshold, cd.driftFactor, cd.referenceValue)

	return nil
}

// detectChangePointsBatch performs batch CUSUM analysis
func (cd *CUSUMDetector) detectChangePointsBatch(data []float64, timestamps []time.Time) ([]*CUSUMChangePoint, error) {
	n := len(data)
	upperCUSUM := make([]float64, n)
	lowerCUSUM := make([]float64, n)

	var changePoints []*CUSUMChangePoint

	// Calculate CUSUM statistics for all points
	for i := 0; i < n; i++ {
		deviation := data[i] - cd.referenceValue

		if i == 0 {
			upperCUSUM[i] = math.Max(0, deviation-cd.driftFactor)
			lowerCUSUM[i] = math.Min(0, deviation+cd.driftFactor)
		} else {
			upperCUSUM[i] = math.Max(0, upperCUSUM[i-1]+deviation-cd.driftFactor)
			lowerCUSUM[i] = math.Min(0, lowerCUSUM[i-1]+deviation+cd.driftFactor)
		}

		// Check for change point detection
		var changePoint *CUSUMChangePoint

		if upperCUSUM[i] > cd.threshold {
			changePoint = cd.createChangePoint(i, "increase", upperCUSUM[i], 0, data, timestamps)
		} else if math.Abs(lowerCUSUM[i]) > cd.threshold {
			changePoint = cd.createChangePoint(i, "decrease", 0, lowerCUSUM[i], data, timestamps)
		}

		if changePoint != nil {
			// Validate change point if validation is enabled
			if cd.config.ValidationWindowSize > 0 {
				cd.validateChangePoint(changePoint, data, timestamps, i)
			}

			changePoints = append(changePoints, changePoint)

			// Reset CUSUM statistics after detection
			cd.resetCUSUMStatistics(upperCUSUM, lowerCUSUM, i)
		}
	}

	return changePoints, nil
}

// detectChangePointsOnline performs online CUSUM analysis
func (cd *CUSUMDetector) detectChangePointsOnline(data []float64, timestamps []time.Time) ([]*CUSUMChangePoint, error) {
	var changePoints []*CUSUMChangePoint

	for i, value := range data {
		deviation := value - cd.referenceValue

		// Update CUSUM statistics
		cd.upperCUSUM = math.Max(0, cd.upperCUSUM+deviation-cd.driftFactor)
		cd.lowerCUSUM = math.Min(0, cd.lowerCUSUM+deviation+cd.driftFactor)

		// Check for change point detection
		var changePoint *CUSUMChangePoint

		if cd.upperCUSUM > cd.threshold {
			changePoint = cd.createChangePoint(i, "increase", cd.upperCUSUM, cd.lowerCUSUM, data, timestamps)
		} else if math.Abs(cd.lowerCUSUM) > cd.threshold {
			changePoint = cd.createChangePoint(i, "decrease", cd.upperCUSUM, cd.lowerCUSUM, data, timestamps)
		}

		if changePoint != nil {
			// Check minimum detection interval
			if cd.config.MinDetectionInterval > 0 {
				if time.Since(cd.lastDetection) < cd.config.MinDetectionInterval {
					continue // Skip this detection
				}
			}

			// Validate change point if validation is enabled
			if cd.config.ValidationWindowSize > 0 {
				cd.validateChangePoint(changePoint, data, timestamps, i)
			}

			changePoints = append(changePoints, changePoint)
			cd.detectedChanges = append(cd.detectedChanges, changePoint)
			cd.lastDetection = timestamps[i]

			// Reset CUSUM statistics
			cd.upperCUSUM = 0
			cd.lowerCUSUM = 0
		}
	}

	return changePoints, nil
}

// createChangePoint creates a CUSUM change point with detailed information
func (cd *CUSUMDetector) createChangePoint(index int, direction string, upperCUSUM, lowerCUSUM float64, data []float64, timestamps []time.Time) *CUSUMChangePoint {
	// Calculate change magnitude
	var magnitude float64
	if direction == "increase" {
		magnitude = upperCUSUM
	} else {
		magnitude = math.Abs(lowerCUSUM)
	}

	// Calculate before and after values
	windowSize := 10 // Fixed window size for before/after comparison
	var beforeValue, afterValue float64

	if index >= windowSize {
		beforeData := data[index-windowSize : index]
		beforeValue = stat.Mean(beforeData, nil)
	} else {
		beforeValue = cd.referenceValue
	}

	if index+windowSize < len(data) {
		afterData := data[index : index+windowSize]
		afterValue = stat.Mean(afterData, nil)
	} else {
		afterValue = data[index]
	}

	// Calculate confidence based on CUSUM value and threshold
	confidence := math.Min(magnitude/cd.threshold, 1.0) * 0.9 // Cap at 0.9

	// Create base change point
	baseChangePoint := &ChangePoint{
		Timestamp:   timestamps[index],
		Metric:      "cusum_detection",
		BeforeValue: beforeValue,
		AfterValue:  afterValue,
		ChangeType:  "step",
		Confidence:  confidence,
	}

	// Create CUSUM-specific change point
	cusumChangePoint := &CUSUMChangePoint{
		ChangePoint: baseChangePoint,
		UpperCUSUM:  upperCUSUM,
		LowerCUSUM:  lowerCUSUM,
		Direction:   direction,
		Magnitude:   magnitude,
	}

	// Calculate pre and post-change statistics
	cusumChangePoint.PreChangeStatistics = cd.calculateStatistics(data, 0, index)
	cusumChangePoint.PostChangeStatistics = cd.calculateStatistics(data, index, len(data))

	// Calculate p-value (simplified implementation)
	cusumChangePoint.PValue = cd.calculatePValue(magnitude)

	return cusumChangePoint
}

// validateChangePoint validates a detected change point using statistical tests
func (cd *CUSUMDetector) validateChangePoint(cp *CUSUMChangePoint, data []float64, timestamps []time.Time, index int) {
	windowSize := cd.config.ValidationWindowSize

	// Ensure we have enough data for validation
	if index < windowSize || index+windowSize >= len(data) {
		cp.Validated = false
		cp.ValidationScore = 0.0
		return
	}

	beforeData := data[index-windowSize : index]
	afterData := data[index : index+windowSize]

	// Perform Welch's t-test
	beforeMean := stat.Mean(beforeData, nil)
	afterMean := stat.Mean(afterData, nil)
	beforeVar := stat.Variance(beforeData, nil)
	afterVar := stat.Variance(afterData, nil)

	n1, n2 := float64(len(beforeData)), float64(len(afterData))

	// Calculate t-statistic
	pooledSE := math.Sqrt(beforeVar/n1 + afterVar/n2)
	if pooledSE == 0 {
		cp.Validated = false
		cp.ValidationScore = 0.0
		return
	}

	tStat := math.Abs(afterMean-beforeMean) / pooledSE

	// Degrees of freedom (Welch's formula)
	_ = math.Pow(beforeVar/n1+afterVar/n2, 2) /
		(math.Pow(beforeVar/n1, 2)/(n1-1) + math.Pow(afterVar/n2, 2)/(n2-1))

	// Simplified p-value calculation using normal approximation
	pValue := 2.0 * (1.0 - math.Abs(tStat)/(math.Abs(tStat)+1.0))

	// Validation criteria
	significanceLevel := 0.05
	effectSizeThreshold := 0.5

	effectSize := math.Abs(afterMean-beforeMean) / math.Sqrt((beforeVar+afterVar)/2)

	cp.Validated = pValue < significanceLevel && effectSize > effectSizeThreshold
	cp.ValidationScore = (1 - pValue) * (effectSize / (effectSize + 1)) // Combined score

	klog.V(3).Infof("Change point validation: validated=%t, score=%.4f, p-value=%.4f, effect-size=%.4f",
		cp.Validated, cp.ValidationScore, pValue, effectSize)
}

// calculateStatistics computes descriptive statistics for a data slice
func (cd *CUSUMDetector) calculateStatistics(data []float64, start, end int) *DescriptiveStatistics {
	if start >= end || start < 0 || end > len(data) {
		return &DescriptiveStatistics{}
	}

	slice := data[start:end]
	if len(slice) == 0 {
		return &DescriptiveStatistics{}
	}

	// Sort for percentile calculations
	sorted := make([]float64, len(slice))
	copy(sorted, slice)
	sort.Float64s(sorted)

	stats := &DescriptiveStatistics{
		Mean:   stat.Mean(slice, nil),
		Median: median(slice),
		StdDev: stat.StdDev(slice, nil),
		Min:    sorted[0],
		Max:    sorted[len(sorted)-1],
		Count:  len(slice),
	}

	// Calculate quartiles
	q1Index := int(float64(len(sorted)) * 0.25)
	q3Index := int(float64(len(sorted)) * 0.75)
	if q1Index < len(sorted) {
		stats.Quartiles[0] = sorted[q1Index]
	}
	stats.Quartiles[1] = stats.Median
	if q3Index < len(sorted) {
		stats.Quartiles[2] = sorted[q3Index]
	}

	// Calculate MAD (Median Absolute Deviation)
	deviations := make([]float64, len(slice))
	for i, v := range slice {
		deviations[i] = math.Abs(v - stats.Median)
	}
	stats.MAD = median(deviations)

	return stats
}

// calculatePValue calculates statistical significance of a change point
func (cd *CUSUMDetector) calculatePValue(magnitude float64) float64 {
	// Simplified p-value calculation based on magnitude relative to threshold
	// In practice, this would use proper statistical distributions

	ratio := magnitude / cd.threshold
	if ratio < 1.0 {
		return 0.5 // Not significant
	}

	// Exponential decay from 0.05 to 0.001 as magnitude increases
	pValue := 0.05 * math.Exp(-0.5*(ratio-1))
	return math.Max(pValue, 0.001)
}

// resetCUSUMStatistics resets CUSUM statistics after change point detection (batch mode)
func (cd *CUSUMDetector) resetCUSUMStatistics(upperCUSUM, lowerCUSUM []float64, fromIndex int) {
	for i := fromIndex; i < len(upperCUSUM); i++ {
		upperCUSUM[i] = 0
		lowerCUSUM[i] = 0
	}
}

// reduceFalsePositives applies techniques to reduce false positive detections
func (cd *CUSUMDetector) reduceFalsePositives(changePoints []*ChangePoint, data []float64, timestamps []time.Time) []*ChangePoint {
	if len(changePoints) <= 1 {
		return changePoints
	}

	filtered := make([]*ChangePoint, 0)

	for i, cp := range changePoints {
		// Skip changes that are too close to each other
		if i > 0 {
			prevCP := changePoints[i-1]
			timeDiff := cp.Timestamp.Sub(prevCP.Timestamp)
			if timeDiff < cd.config.MinDetectionInterval {
				continue
			}
		}

		// Skip changes with low confidence
		if cp.Confidence < 0.5 {
			continue
		}

		// Skip changes with small magnitude relative to noise
		changeMagnitude := math.Abs(cp.AfterValue - cp.BeforeValue)
		if changeMagnitude < cd.driftFactor {
			continue
		}

		filtered = append(filtered, cp)
	}

	klog.V(3).Infof("False positive reduction: %d -> %d change points", len(changePoints), len(filtered))

	return filtered
}

// GetName returns the algorithm name
func (cd *CUSUMDetector) GetName() string {
	return "CUSUM"
}

// GetParameters returns current algorithm parameters
func (cd *CUSUMDetector) GetParameters() map[string]interface{} {
	return map[string]interface{}{
		"threshold":       cd.threshold,
		"drift_factor":    cd.driftFactor,
		"reference_value": cd.referenceValue,
		"upper_cusum":     cd.upperCUSUM,
		"lower_cusum":     cd.lowerCUSUM,
	}
}

// UpdateParameters updates algorithm parameters
func (cd *CUSUMDetector) UpdateParameters(params map[string]interface{}) error {
	if threshold, ok := params["threshold"].(float64); ok {
		cd.threshold = threshold
	}

	if driftFactor, ok := params["drift_factor"].(float64); ok {
		cd.driftFactor = driftFactor
	}

	if referenceValue, ok := params["reference_value"].(float64); ok {
		cd.referenceValue = referenceValue
	}

	return nil
}

// AdaptThreshold adapts the detection threshold based on recent data
func (cd *CUSUMDetector) AdaptThreshold(recentData []float64) {
	if !cd.config.AdaptiveThreshold || len(recentData) < 10 {
		return
	}

	var stdDev float64
	if cd.config.UseRobustStatistics {
		medianValue := median(recentData)
		deviations := make([]float64, len(recentData))
		for i, v := range recentData {
			deviations[i] = math.Abs(v - medianValue)
		}
		mad := median(deviations)
		stdDev = mad * 1.4826
	} else {
		stdDev = stat.StdDev(recentData, nil)
	}

	newThreshold := cd.config.ThresholdMultiplier * stdDev

	// Smooth threshold adaptation
	alpha := 0.1 // Learning rate
	cd.threshold = alpha*newThreshold + (1-alpha)*cd.threshold

	klog.V(3).Infof("CUSUM threshold adapted: %.4f -> %.4f", cd.threshold, newThreshold)
}

// GetDetectionHistory returns the history of detected change points
func (cd *CUSUMDetector) GetDetectionHistory() []*CUSUMChangePoint {
	return cd.detectedChanges
}

// Reset resets the detector state
func (cd *CUSUMDetector) Reset() {
	cd.upperCUSUM = 0
	cd.lowerCUSUM = 0
	cd.detectedChanges = make([]*CUSUMChangePoint, 0)
	cd.lastDetection = time.Time{}
}

// GetDetectionStatistics returns statistics about detection performance
func (cd *CUSUMDetector) GetDetectionStatistics() map[string]interface{} {
	totalDetections := len(cd.detectedChanges)
	validatedDetections := 0

	for _, cp := range cd.detectedChanges {
		if cp.Validated {
			validatedDetections++
		}
	}

	var validationRate float64
	if totalDetections > 0 {
		validationRate = float64(validatedDetections) / float64(totalDetections)
	}

	return map[string]interface{}{
		"total_detections":     totalDetections,
		"validated_detections": validatedDetections,
		"validation_rate":      validationRate,
		"current_threshold":    cd.threshold,
		"current_upper_cusum":  cd.upperCUSUM,
		"current_lower_cusum":  cd.lowerCUSUM,
	}
}

// getDefaultCUSUMConfig returns default configuration for CUSUM detector
func getDefaultCUSUMConfig() *CUSUMConfig {
	return &CUSUMConfig{
		ThresholdMultiplier:    4.0,             // 4 standard deviations
		DriftSensitivity:       0.5,             // 50% of standard deviation
		MinDetectionInterval:   5 * time.Minute, // 5 minutes between detections
		UseRobustStatistics:    true,            // Use median and MAD
		AdaptiveThreshold:      true,            // Adapt threshold dynamically
		SeasonalAdjustment:     false,           // No seasonal adjustment by default
		MaxHistorySize:         1000,            // Keep last 1000 points
		BatchProcessing:        false,           // Use online processing
		ValidationWindowSize:   20,              // 20 points for validation
		FalsePositiveReduction: true,            // Enable false positive reduction
	}
}
