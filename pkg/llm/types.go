package llm

// Priority represents request priority levels
type Priority int

const (
	LowPriority Priority = iota
	NormalPriority
	HighPriority
	CriticalPriority
)

// String returns the string representation of the priority
func (p Priority) String() string {
	switch p {
	case LowPriority:
		return "low"
	case NormalPriority:
		return "normal"
	case HighPriority:
		return "high"
	case CriticalPriority:
		return "critical"
	default:
		return "unknown"
	}
}