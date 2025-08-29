
package errors



import (

	"crypto/sha256"

	"encoding/hex"

	"fmt"

	"math"

	"sort"

	"strings"

	"sync"

	"time"

)



// ErrorAggregator collects and analyzes multiple errors.

type ErrorAggregator struct {

	mu                  sync.RWMutex

	errors              []*ServiceError

	patterns            map[string]*ErrorPattern

	deduplicationWindow time.Duration

	maxErrors           int

	analysisEnabled     bool

	correlations        map[string]*ErrorCorrelation

}



// ErrorPattern represents a pattern of similar errors.

type ErrorPattern struct {

	ID              string                 `json:"id"`

	Hash            string                 `json:"hash"`

	Count           int                    `json:"count"`

	FirstSeen       time.Time              `json:"first_seen"`

	LastSeen        time.Time              `json:"last_seen"`

	ErrorType       ErrorType              `json:"error_type"`

	Service         string                 `json:"service"`

	Operation       string                 `json:"operation"`

	Component       string                 `json:"component"`

	Message         string                 `json:"message"`

	Frequency       float64                `json:"frequency"`

	TrendDirection  string                 `json:"trend_direction"` // \"increasing\", \"decreasing\", \"stable\"

	Impact          ErrorImpact            `json:"impact"`

	Severity        ErrorSeverity          `json:"severity"`

	Examples        []*ServiceError        `json:"examples"`

	RelatedPatterns []string               `json:"related_patterns"`

	Metadata        map[string]interface{} `json:"metadata"`

}



// ErrorCorrelation represents correlations between errors.

type ErrorCorrelation struct {

	ID               string        `json:"id"`

	ErrorA           string        `json:"error_a"`

	ErrorB           string        `json:"error_b"`

	CorrelationScore float64       `json:"correlation_score"`

	TimeWindow       time.Duration `json:"time_window"`

	Count            int           `json:"count"`

	FirstSeen        time.Time     `json:"first_seen"`

	LastSeen         time.Time     `json:"last_seen"`

	CausationType    string        `json:"causation_type"` // "cascade", "common_cause", "coincident"

}



// ErrorAggregationConfig configures error aggregation behavior.

type ErrorAggregationConfig struct {

	MaxErrors            int           `json:"max_errors"`

	DeduplicationWindow  time.Duration `json:"deduplication_window"`

	PatternAnalysis      bool          `json:"pattern_analysis"`

	CorrelationAnalysis  bool          `json:"correlation_analysis"`

	RetentionPeriod      time.Duration `json:"retention_period"`

	MinPatternCount      int           `json:"min_pattern_count"`

	CorrelationThreshold float64       `json:"correlation_threshold"`

}



// DefaultErrorAggregationConfig returns sensible defaults.

func DefaultErrorAggregationConfig() *ErrorAggregationConfig {

	return &ErrorAggregationConfig{

		MaxErrors:            1000,

		DeduplicationWindow:  time.Minute * 5,

		PatternAnalysis:      true,

		CorrelationAnalysis:  true,

		RetentionPeriod:      time.Hour * 24,

		MinPatternCount:      3,

		CorrelationThreshold: 0.7,

	}

}



// NewErrorAggregator creates a new error aggregator.

func NewErrorAggregator(config *ErrorAggregationConfig) *ErrorAggregator {

	if config == nil {

		config = DefaultErrorAggregationConfig()

	}



	aggregator := &ErrorAggregator{

		errors:              make([]*ServiceError, 0),

		patterns:            make(map[string]*ErrorPattern),

		deduplicationWindow: config.DeduplicationWindow,

		maxErrors:           config.MaxErrors,

		analysisEnabled:     config.PatternAnalysis,

		correlations:        make(map[string]*ErrorCorrelation),

	}



	// Start background cleanup goroutine.

	go aggregator.cleanup(config.RetentionPeriod)



	return aggregator

}



// AddError adds an error to the aggregator.

func (ea *ErrorAggregator) AddError(err *ServiceError) {

	if err == nil {

		return

	}



	ea.mu.Lock()

	defer ea.mu.Unlock()



	// Add correlation ID if not present.

	if err.CorrelationID == "" {

		err.CorrelationID = ea.generateCorrelationID(err)

	}



	// Check for deduplication.

	if ea.isDuplicate(err) {

		ea.updateExistingError(err)

		return

	}



	// Add the error.

	ea.errors = append(ea.errors, err)



	// Enforce max errors limit.

	if len(ea.errors) > ea.maxErrors {

		ea.errors = ea.errors[len(ea.errors)-ea.maxErrors:]

	}



	// Update patterns.

	if ea.analysisEnabled {

		ea.updatePatterns(err)

		ea.updateCorrelations(err)

	}

}



// AddErrors adds multiple errors at once.

func (ea *ErrorAggregator) AddErrors(errors []*ServiceError) {

	for _, err := range errors {

		ea.AddError(err)

	}

}



// GetErrors returns all collected errors.

func (ea *ErrorAggregator) GetErrors() []*ServiceError {

	ea.mu.RLock()

	defer ea.mu.RUnlock()



	result := make([]*ServiceError, len(ea.errors))

	copy(result, ea.errors)

	return result

}



// GetErrorsInTimeRange returns errors within a specific time range.

func (ea *ErrorAggregator) GetErrorsInTimeRange(start, end time.Time) []*ServiceError {

	ea.mu.RLock()

	defer ea.mu.RUnlock()



	var result []*ServiceError

	for _, err := range ea.errors {

		if err.Timestamp.After(start) && err.Timestamp.Before(end) {

			result = append(result, err)

		}

	}



	return result

}



// GetErrorsByType returns errors of a specific type.

func (ea *ErrorAggregator) GetErrorsByType(errorType ErrorType) []*ServiceError {

	ea.mu.RLock()

	defer ea.mu.RUnlock()



	var result []*ServiceError

	for _, err := range ea.errors {

		if err.Type == errorType {

			result = append(result, err)

		}

	}



	return result

}



// GetErrorsByService returns errors from a specific service.

func (ea *ErrorAggregator) GetErrorsByService(service string) []*ServiceError {

	ea.mu.RLock()

	defer ea.mu.RUnlock()



	var result []*ServiceError

	for _, err := range ea.errors {

		if err.Service == service {

			result = append(result, err)

		}

	}



	return result

}



// GetPatterns returns all identified error patterns.

func (ea *ErrorAggregator) GetPatterns() []*ErrorPattern {

	ea.mu.RLock()

	defer ea.mu.RUnlock()



	patterns := make([]*ErrorPattern, 0, len(ea.patterns))

	for _, pattern := range ea.patterns {

		patterns = append(patterns, pattern)

	}



	// Sort by frequency (most frequent first).

	sort.Slice(patterns, func(i, j int) bool {

		return patterns[i].Frequency > patterns[j].Frequency

	})



	return patterns

}



// GetTopPatterns returns the top N error patterns by frequency.

func (ea *ErrorAggregator) GetTopPatterns(n int) []*ErrorPattern {

	patterns := ea.GetPatterns()

	if len(patterns) > n {

		patterns = patterns[:n]

	}

	return patterns

}



// GetCorrelations returns all error correlations.

func (ea *ErrorAggregator) GetCorrelations() []*ErrorCorrelation {

	ea.mu.RLock()

	defer ea.mu.RUnlock()



	correlations := make([]*ErrorCorrelation, 0, len(ea.correlations))

	for _, correlation := range ea.correlations {

		correlations = append(correlations, correlation)

	}



	// Sort by correlation score.

	sort.Slice(correlations, func(i, j int) bool {

		return correlations[i].CorrelationScore > correlations[j].CorrelationScore

	})



	return correlations

}



// GetStatistics returns aggregated error statistics.

func (ea *ErrorAggregator) GetStatistics() *ErrorStatistics {

	ea.mu.RLock()

	defer ea.mu.RUnlock()



	stats := &ErrorStatistics{

		TotalErrors:      len(ea.errors),

		ErrorsByType:     make(map[string]int),

		ErrorsByService:  make(map[string]int),

		ErrorsBySeverity: make(map[string]int),

		ErrorsByCategory: make(map[string]int),

		Patterns:         len(ea.patterns),

		Correlations:     len(ea.correlations),

	}



	if len(ea.errors) == 0 {

		return stats

	}



	// Calculate time range.

	stats.OldestError = ea.errors[0].Timestamp

	stats.NewestError = ea.errors[0].Timestamp



	// Aggregate statistics.

	for _, err := range ea.errors {

		if err.Timestamp.Before(stats.OldestError) {

			stats.OldestError = err.Timestamp

		}

		if err.Timestamp.After(stats.NewestError) {

			stats.NewestError = err.Timestamp

		}



		stats.ErrorsByType[string(err.Type)]++

		stats.ErrorsByService[err.Service]++

		stats.ErrorsBySeverity[err.Severity.String()]++

		stats.ErrorsByCategory[string(err.Category)]++

	}



	// Calculate error rate.

	duration := stats.NewestError.Sub(stats.OldestError)

	if duration > 0 {

		stats.ErrorsPerSecond = float64(stats.TotalErrors) / duration.Seconds()

		stats.ErrorsPerMinute = stats.ErrorsPerSecond * 60

		stats.ErrorsPerHour = stats.ErrorsPerMinute * 60

	}



	return stats

}



// GetTrendAnalysis analyzes error trends over time.

func (ea *ErrorAggregator) GetTrendAnalysis(timeWindow time.Duration) *TrendAnalysis {

	ea.mu.RLock()

	defer ea.mu.RUnlock()



	now := time.Now()

	cutoff := now.Add(-timeWindow)



	recentErrors := 0

	for _, err := range ea.errors {

		if err.Timestamp.After(cutoff) {

			recentErrors++

		}

	}



	// Calculate trend direction.

	halfWindow := timeWindow / 2

	middleCutoff := now.Add(-halfWindow)



	firstHalfErrors := 0

	secondHalfErrors := 0



	for _, err := range ea.errors {

		if err.Timestamp.After(cutoff) && err.Timestamp.Before(middleCutoff) {

			firstHalfErrors++

		} else if err.Timestamp.After(middleCutoff) {

			secondHalfErrors++

		}

	}



	trend := "stable"

	trendValue := 0.0



	if firstHalfErrors > 0 {

		change := float64(secondHalfErrors-firstHalfErrors) / float64(firstHalfErrors)

		trendValue = change * 100



		if change > 0.1 {

			trend = "increasing"

		} else if change < -0.1 {

			trend = "decreasing"

		}

	}



	return &TrendAnalysis{

		TimeWindow:       timeWindow,

		TotalErrors:      recentErrors,

		FirstHalfErrors:  firstHalfErrors,

		SecondHalfErrors: secondHalfErrors,

		Trend:            trend,

		TrendPercentage:  trendValue,

		AnalyzedAt:       now,

	}

}



// isDuplicate checks if an error is a duplicate within the deduplication window.

func (ea *ErrorAggregator) isDuplicate(err *ServiceError) bool {

	cutoff := time.Now().Add(-ea.deduplicationWindow)



	for i := len(ea.errors) - 1; i >= 0; i-- {

		existingErr := ea.errors[i]



		// Stop checking if we're outside the deduplication window.

		if existingErr.Timestamp.Before(cutoff) {

			break

		}



		// Check if it's a duplicate.

		if ea.errorsMatch(existingErr, err) {

			return true

		}

	}



	return false

}



// errorsMatch determines if two errors are considered duplicates.

func (ea *ErrorAggregator) errorsMatch(a, b *ServiceError) bool {

	return a.Type == b.Type &&

		a.Code == b.Code &&

		a.Service == b.Service &&

		a.Operation == b.Operation &&

		a.Component == b.Component &&

		a.Message == b.Message

}



// updateExistingError updates an existing error's information.

func (ea *ErrorAggregator) updateExistingError(err *ServiceError) {

	// For now, just update the timestamp of the most recent matching error.

	cutoff := time.Now().Add(-ea.deduplicationWindow)



	for i := len(ea.errors) - 1; i >= 0; i-- {

		existingErr := ea.errors[i]



		if existingErr.Timestamp.Before(cutoff) {

			break

		}



		if ea.errorsMatch(existingErr, err) {

			existingErr.Timestamp = err.Timestamp

			// Increment retry count or other metrics if applicable.

			if err.RetryCount > existingErr.RetryCount {

				existingErr.RetryCount = err.RetryCount

			}

			break

		}

	}

}



// generateCorrelationID generates a correlation ID for error grouping.

func (ea *ErrorAggregator) generateCorrelationID(err *ServiceError) string {

	// Create a hash based on error characteristics.

	hasher := sha256.New()

	fmt.Fprintf(hasher, "%s:%s:%s:%s:%s",

		err.Type, err.Code, err.Service, err.Operation, err.Component)



	hash := hasher.Sum(nil)

	return hex.EncodeToString(hash)[:12] // Use first 12 characters

}



// updatePatterns updates error patterns based on new errors.

func (ea *ErrorAggregator) updatePatterns(err *ServiceError) {

	patternHash := ea.calculatePatternHash(err)



	if pattern, exists := ea.patterns[patternHash]; exists {

		// Update existing pattern.

		pattern.Count++

		pattern.LastSeen = err.Timestamp



		// Add as example if we don't have many.

		if len(pattern.Examples) < 5 {

			pattern.Examples = append(pattern.Examples, err)

		}



		// Update frequency.

		duration := pattern.LastSeen.Sub(pattern.FirstSeen)

		if duration > 0 {

			pattern.Frequency = float64(pattern.Count) / duration.Seconds()

		}



		// Update trend.

		pattern.TrendDirection = ea.calculateTrendDirection(pattern)

	} else {

		// Create new pattern.

		ea.patterns[patternHash] = &ErrorPattern{

			ID:              fmt.Sprintf("pattern-%s", patternHash[:8]),

			Hash:            patternHash,

			Count:           1,

			FirstSeen:       err.Timestamp,

			LastSeen:        err.Timestamp,

			ErrorType:       err.Type,

			Service:         err.Service,

			Operation:       err.Operation,

			Component:       err.Component,

			Message:         err.Message,

			Frequency:       0,

			TrendDirection:  "new",

			Impact:          err.Impact,

			Severity:        err.Severity,

			Examples:        []*ServiceError{err},

			RelatedPatterns: []string{},

			Metadata:        make(map[string]interface{}),

		}

	}

}



// calculatePatternHash calculates a hash for pattern matching.

func (ea *ErrorAggregator) calculatePatternHash(err *ServiceError) string {

	hasher := sha256.New()



	// Normalize the message by removing variable parts.

	normalizedMessage := ea.normalizeMessage(err.Message)



	fmt.Fprintf(hasher, "%s:%s:%s:%s:%s",

		err.Type, err.Service, err.Operation, err.Component, normalizedMessage)



	hash := hasher.Sum(nil)

	return hex.EncodeToString(hash)

}



// normalizeMessage normalizes error messages by removing variable parts.

func (ea *ErrorAggregator) normalizeMessage(message string) string {

	// Simple normalization - replace numbers and UUIDs with placeholders.

	normalized := strings.ReplaceAll(message, "\\d+", "<number>")

	normalized = strings.ReplaceAll(normalized, "[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}", "<uuid>")

	return normalized

}



// calculateTrendDirection calculates the trend direction for a pattern.

func (ea *ErrorAggregator) calculateTrendDirection(pattern *ErrorPattern) string {

	// Simple trend calculation based on recent occurrences.

	now := time.Now()

	recentWindow := now.Add(-time.Hour)



	recentCount := 0

	for _, example := range pattern.Examples {

		if example.Timestamp.After(recentWindow) {

			recentCount++

		}

	}



	if pattern.Count < 5 {

		return "new"

	}



	recentRatio := float64(recentCount) / float64(pattern.Count)

	if recentRatio > 0.6 {

		return "increasing"

	} else if recentRatio < 0.2 {

		return "decreasing"

	}



	return "stable"

}



// updateCorrelations analyzes and updates error correlations.

func (ea *ErrorAggregator) updateCorrelations(err *ServiceError) {

	// Look for errors that occurred around the same time.

	timeWindow := time.Minute * 5

	correlationWindow := err.Timestamp.Add(-timeWindow)



	for _, otherErr := range ea.errors {

		if otherErr == err {

			continue

		}



		if otherErr.Timestamp.After(correlationWindow) && otherErr.Timestamp.Before(err.Timestamp.Add(timeWindow)) {

			ea.recordCorrelation(err, otherErr)

		}

	}

}



// recordCorrelation records a correlation between two errors.

func (ea *ErrorAggregator) recordCorrelation(errA, errB *ServiceError) {

	correlationID := ea.generateCorrelationHash(errA, errB)



	if correlation, exists := ea.correlations[correlationID]; exists {

		correlation.Count++

		correlation.LastSeen = time.Now()

	} else {

		score := ea.calculateCorrelationScore(errA, errB)

		if score > 0.5 { // Only record significant correlations

			ea.correlations[correlationID] = &ErrorCorrelation{

				ID:               correlationID,

				ErrorA:           ea.generateCorrelationID(errA),

				ErrorB:           ea.generateCorrelationID(errB),

				CorrelationScore: score,

				TimeWindow:       time.Minute * 5,

				Count:            1,

				FirstSeen:        time.Now(),

				LastSeen:         time.Now(),

				CausationType:    ea.determineCausationType(errA, errB),

			}

		}

	}

}



// generateCorrelationHash generates a hash for correlation identification.

func (ea *ErrorAggregator) generateCorrelationHash(errA, errB *ServiceError) string {

	// Ensure consistent ordering.

	var first, second *ServiceError

	if errA.Type < errB.Type || (errA.Type == errB.Type && errA.Service < errB.Service) {

		first, second = errA, errB

	} else {

		first, second = errB, errA

	}



	hasher := sha256.New()

	fmt.Fprintf(hasher, "%s:%s:%s:%s",

		first.Type, first.Service, second.Type, second.Service)



	hash := hasher.Sum(nil)

	return hex.EncodeToString(hash)[:16]

}



// calculateCorrelationScore calculates the correlation score between two errors.

func (ea *ErrorAggregator) calculateCorrelationScore(errA, errB *ServiceError) float64 {

	score := 0.0



	// Same service increases correlation.

	if errA.Service == errB.Service {

		score += 0.3

	}



	// Related operations increase correlation.

	if errA.Operation == errB.Operation {

		score += 0.2

	}



	// Same component increases correlation strongly.

	if errA.Component == errB.Component {

		score += 0.4

	}



	// Same user/session increases correlation.

	if errA.UserID != "" && errA.UserID == errB.UserID {

		score += 0.3

	}



	if errA.SessionID != "" && errA.SessionID == errB.SessionID {

		score += 0.2

	}



	// Same request ID means very high correlation.

	if errA.RequestID != "" && errA.RequestID == errB.RequestID {

		score += 0.6

	}



	return math.Min(score, 1.0)

}



// determineCausationType determines the type of causation between errors.

func (ea *ErrorAggregator) determineCausationType(errA, errB *ServiceError) string {

	// Simple heuristics for causation type.

	if errA.Service == errB.Service && errA.Component == errB.Component {

		return "common_cause"

	}



	if errA.RequestID != "" && errA.RequestID == errB.RequestID {

		return "cascade"

	}



	if math.Abs(errA.Timestamp.Sub(errB.Timestamp).Seconds()) < 1 {

		return "coincident"

	}



	return "cascade"

}



// cleanup removes old errors and patterns.

func (ea *ErrorAggregator) cleanup(retentionPeriod time.Duration) {

	ticker := time.NewTicker(time.Hour)

	defer ticker.Stop()



	for range ticker.C {

		ea.performCleanup(retentionPeriod)

	}

}



// performCleanup removes old data.

func (ea *ErrorAggregator) performCleanup(retentionPeriod time.Duration) {

	ea.mu.Lock()

	defer ea.mu.Unlock()



	cutoff := time.Now().Add(-retentionPeriod)



	// Clean old errors.

	i := 0

	for i < len(ea.errors) && ea.errors[i].Timestamp.Before(cutoff) {

		i++

	}

	if i > 0 {

		ea.errors = ea.errors[i:]

	}



	// Clean old patterns.

	for hash, pattern := range ea.patterns {

		if pattern.LastSeen.Before(cutoff) {

			delete(ea.patterns, hash)

		}

	}



	// Clean old correlations.

	for id, correlation := range ea.correlations {

		if correlation.LastSeen.Before(cutoff) {

			delete(ea.correlations, id)

		}

	}

}



// ErrorStatistics holds aggregated error statistics.

type ErrorStatistics struct {

	TotalErrors      int            `json:"total_errors"`

	ErrorsByType     map[string]int `json:"errors_by_type"`

	ErrorsByService  map[string]int `json:"errors_by_service"`

	ErrorsBySeverity map[string]int `json:"errors_by_severity"`

	ErrorsByCategory map[string]int `json:"errors_by_category"`

	Patterns         int            `json:"patterns"`

	Correlations     int            `json:"correlations"`

	OldestError      time.Time      `json:"oldest_error"`

	NewestError      time.Time      `json:"newest_error"`

	ErrorsPerSecond  float64        `json:"errors_per_second"`

	ErrorsPerMinute  float64        `json:"errors_per_minute"`

	ErrorsPerHour    float64        `json:"errors_per_hour"`

}



// TrendAnalysis holds error trend analysis results.

type TrendAnalysis struct {

	TimeWindow       time.Duration `json:"time_window"`

	TotalErrors      int           `json:"total_errors"`

	FirstHalfErrors  int           `json:"first_half_errors"`

	SecondHalfErrors int           `json:"second_half_errors"`

	Trend            string        `json:"trend"`

	TrendPercentage  float64       `json:"trend_percentage"`

	AnalyzedAt       time.Time     `json:"analyzed_at"`

}



// MultiError represents multiple errors that occurred together.

type MultiError struct {

	Errors []error `json:"errors"`

	mu     sync.RWMutex

}



// NewMultiError creates a new MultiError.

func NewMultiError() *MultiError {

	return &MultiError{

		Errors: make([]error, 0),

	}

}



// Add adds an error to the MultiError.

func (me *MultiError) Add(err error) {

	if err == nil {

		return

	}



	me.mu.Lock()

	defer me.mu.Unlock()

	me.Errors = append(me.Errors, err)

}



// AddMultiple adds multiple errors at once.

func (me *MultiError) AddMultiple(errors ...error) {

	me.mu.Lock()

	defer me.mu.Unlock()



	for _, err := range errors {

		if err != nil {

			me.Errors = append(me.Errors, err)

		}

	}

}



// Error implements the error interface.

func (me *MultiError) Error() string {

	me.mu.RLock()

	defer me.mu.RUnlock()



	if len(me.Errors) == 0 {

		return "no errors"

	}



	if len(me.Errors) == 1 {

		return me.Errors[0].Error()

	}



	var messages []string

	for i, err := range me.Errors {

		messages = append(messages, fmt.Sprintf("[%d] %s", i+1, err.Error()))

	}



	return fmt.Sprintf("multiple errors occurred: %s", strings.Join(messages, "; "))

}



// HasErrors returns true if there are any errors.

func (me *MultiError) HasErrors() bool {

	me.mu.RLock()

	defer me.mu.RUnlock()

	return len(me.Errors) > 0

}



// Count returns the number of errors.

func (me *MultiError) Count() int {

	me.mu.RLock()

	defer me.mu.RUnlock()

	return len(me.Errors)

}



// GetErrors returns a copy of all errors.

func (me *MultiError) GetErrors() []error {

	me.mu.RLock()

	defer me.mu.RUnlock()



	result := make([]error, len(me.Errors))

	copy(result, me.Errors)

	return result

}



// Clear removes all errors.

func (me *MultiError) Clear() {

	me.mu.Lock()

	defer me.mu.Unlock()

	me.Errors = me.Errors[:0]

}



// Unwrap returns the first error for error unwrapping.

func (me *MultiError) Unwrap() error {

	me.mu.RLock()

	defer me.mu.RUnlock()



	if len(me.Errors) == 0 {

		return nil

	}



	return me.Errors[0]

}



// ErrorCollector provides a convenient way to collect multiple errors.

type ErrorCollector struct {

	errors    *MultiError

	maxErrors int

	onError   func(error)

}



// NewErrorCollector creates a new error collector.

func NewErrorCollector(maxErrors int) *ErrorCollector {

	return &ErrorCollector{

		errors:    NewMultiError(),

		maxErrors: maxErrors,

	}

}



// SetErrorCallback sets a callback for when errors are added.

func (ec *ErrorCollector) SetErrorCallback(callback func(error)) {

	ec.onError = callback

}



// Collect adds an error to the collection.

func (ec *ErrorCollector) Collect(err error) {

	if err == nil {

		return

	}



	if ec.maxErrors > 0 && ec.errors.Count() >= ec.maxErrors {

		return // Skip if at limit

	}



	ec.errors.Add(err)



	if ec.onError != nil {

		ec.onError(err)

	}

}



// CollectMultiple collects multiple errors.

func (ec *ErrorCollector) CollectMultiple(errors ...error) {

	for _, err := range errors {

		ec.Collect(err)

	}

}



// Result returns the collected errors as a single error.

func (ec *ErrorCollector) Result() error {

	if !ec.errors.HasErrors() {

		return nil

	}



	return ec.errors

}



// HasErrors returns true if any errors were collected.

func (ec *ErrorCollector) HasErrors() bool {

	return ec.errors.HasErrors()

}



// Count returns the number of collected errors.

func (ec *ErrorCollector) Count() int {

	return ec.errors.Count()

}



// Clear clears all collected errors.

func (ec *ErrorCollector) Clear() {

	ec.errors.Clear()

}



// GetErrors returns all collected errors.

func (ec *ErrorCollector) GetErrors() []error {

	return ec.errors.GetErrors()

}

