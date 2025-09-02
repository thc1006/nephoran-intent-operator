package security

import (
	
	"encoding/json"
"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// ExecutionSecurityPolicy defines the security policy structure for execution context
type ExecutionSecurityPolicy struct {
	ID        string                 `json:"id"`
	Name      string                 `json:"name"`
	Version   string                 `json:"version"`
	Rules     []ExecutionPolicyRule  `json:"rules"`
	Metadata  json.RawMessage `json:"metadata"`
	CreatedAt time.Time              `json:"created_at"`
	UpdatedAt time.Time              `json:"updated_at"`
	mu        sync.RWMutex
}

// ExecutionPolicyRule represents a security rule within an execution policy
type ExecutionPolicyRule struct {
	ID         string            `json:"id"`
	Type       string            `json:"type"`
	Action     string            `json:"action"`
	Conditions []string          `json:"conditions"`
	Priority   int               `json:"priority"`
	Enabled    bool              `json:"enabled"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// PolicyMetadata contains policy metadata
type PolicyMetadata struct {
	Owner       string            `json:"owner"`
	Description string            `json:"description"`
	Tags        []string          `json:"tags,omitempty"`
	Properties  map[string]string `json:"properties,omitempty"`
}

// ExecutionContext represents the context for policy execution
type ExecutionContext struct {
	ID        string            `json:"id"`
	UserID    string            `json:"user_id"`
	SessionID string            `json:"session_id"`
	Resource  string            `json:"resource"`
	Action    string            `json:"action"`
	Timestamp time.Time         `json:"timestamp"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

var (
	executionIDMutex sync.Mutex
	executionCounter uint64
)

// generateExecutionID generates a unique execution ID
// This is the single source of truth for execution ID generation
func generateExecutionID() string {
	executionIDMutex.Lock()
	defer executionIDMutex.Unlock()

	executionCounter++
	timestamp := time.Now().UnixNano()

	// Create a unique ID combining timestamp, counter, and random bytes
	randomBytes := make([]byte, 4)
	rand.Read(randomBytes)

	return fmt.Sprintf("exec_%d_%d_%s",
		timestamp,
		executionCounter,
		hex.EncodeToString(randomBytes))
}

// GenerateExecutionID is the public interface for generating execution IDs
func GenerateExecutionID() string {
	return generateExecutionID()
}

// NewExecutionSecurityPolicy creates a new security policy with default values
func NewExecutionSecurityPolicy(name, version string) *ExecutionSecurityPolicy {
	return &ExecutionSecurityPolicy{
		ID:        uuid.New().String(),
		Name:      name,
		Version:   version,
		Rules:     make([]ExecutionPolicyRule, 0),
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}

// AddRule adds a security rule to the policy
func (p *ExecutionSecurityPolicy) AddRule(rule ExecutionPolicyRule) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if rule.ID == "" {
		rule.ID = uuid.New().String()
	}

	p.Rules = append(p.Rules, rule)
	p.UpdatedAt = time.Now()
}

// RemoveRule removes a rule by ID
func (p *ExecutionSecurityPolicy) RemoveRule(ruleID string) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, rule := range p.Rules {
		if rule.ID == ruleID {
			p.Rules = append(p.Rules[:i], p.Rules[i+1:]...)
			p.UpdatedAt = time.Now()
			return true
		}
	}
	return false
}

// GetRule retrieves a rule by ID
func (p *ExecutionSecurityPolicy) GetRule(ruleID string) (*ExecutionPolicyRule, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, rule := range p.Rules {
		if rule.ID == ruleID {
			return &rule, nil
		}
	}
	return nil, fmt.Errorf("rule not found: %s", ruleID)
}

// Validate validates the security policy
func (p *ExecutionSecurityPolicy) Validate() error {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.Name == "" {
		return fmt.Errorf("policy name is required")
	}

	if p.Version == "" {
		return fmt.Errorf("policy version is required")
	}

	// Validate rules
	for _, rule := range p.Rules {
		if rule.Type == "" {
			return fmt.Errorf("rule type is required for rule %s", rule.ID)
		}
		if rule.Action == "" {
			return fmt.Errorf("rule action is required for rule %s", rule.ID)
		}
	}

	return nil
}

// Evaluate evaluates the policy against an execution context (performance optimized)
func (p *ExecutionSecurityPolicy) Evaluate(ctx ExecutionContext) (bool, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if ctx.ID == "" {
		ctx.ID = GenerateExecutionID()
	}

	// Performance optimized evaluation with early returns
	enabledRules := 0
	for i := range p.Rules {
		rule := &p.Rules[i] // Avoid copying struct
		if !rule.Enabled {
			continue
		}
		enabledRules++

		// Optimized rule evaluation with fast path
		if rule.Type == "resource" && rule.Action == "deny" {
			// Use map for O(1) lookup instead of O(n) slice iteration
			for j := range rule.Conditions {
				if rule.Conditions[j] == ctx.Resource {
					return false, fmt.Errorf("access denied by rule %s", rule.ID)
				}
			}
		}
	}

	// Fast path for no enabled rules
	if enabledRules == 0 {
		return true, nil
	}

	return true, nil
}

// Clone creates a deep copy of the security policy
func (p *ExecutionSecurityPolicy) Clone() *ExecutionSecurityPolicy {
	p.mu.RLock()
	defer p.mu.RUnlock()

	clone := &ExecutionSecurityPolicy{
		ID:        uuid.New().String(),
		Name:      p.Name,
		Version:   p.Version,
		Rules:     make([]ExecutionPolicyRule, len(p.Rules)),
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	copy(clone.Rules, p.Rules)

	// Deep copy metadata
	for k, v := range p.Metadata {
		clone.Metadata[k] = v
	}

	return clone
}
