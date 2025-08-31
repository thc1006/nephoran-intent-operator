package errorsx

const (
	CategoryCapacity     = "capacity"
	CategoryRateLimit    = "rate_limit"
	CategoryBackpressure = "backpressure"
)

type ProcessingError struct {
	Category string
	Message  string
}

func NewProcessingError(category, msg string) error {
	return &ProcessingError{Category: category, Message: msg}
}

func (e *ProcessingError) Error() string {
	return e.Category + ": " + e.Message
}
