package ingest

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// IntentNLP defines the interface for natural language to intent conversion
type IntentNLP interface {
	// ParseIntent converts natural language text to structured Intent JSON
	ParseIntent(text string) (map[string]interface{}, error)
}

// RuleBasedIntentParser implements IntentNLP with deterministic rule-based parsing
type RuleBasedIntentParser struct {
	// Patterns for different intent types
	patterns map[string]*regexp.Regexp
}

// NewRuleBasedIntentParser creates a new rule-based intent parser
func NewRuleBasedIntentParser() *RuleBasedIntentParser {
	return &RuleBasedIntentParser{
		patterns: map[string]*regexp.Regexp{
			"scaling": regexp.MustCompile(`(?i)scale\s+(\S+)\s+to\s+(\d+)(?:\s+in\s+ns\s+(\S+))?`),
			"deploy":  regexp.MustCompile(`(?i)deploy\s+(\S+)(?:\s+in\s+ns\s+(\S+))?`),
			"delete":  regexp.MustCompile(`(?i)delete\s+(\S+)(?:\s+from\s+ns\s+(\S+))?`),
			"update":  regexp.MustCompile(`(?i)update\s+(\S+)\s+set\s+(\S+)\s*=\s*(\S+)(?:\s+in\s+ns\s+(\S+))?`),
		},
	}
}

// ParseIntent implements the IntentNLP interface
func (p *RuleBasedIntentParser) ParseIntent(text string) (map[string]interface{}, error) {
	// Trim and normalize the input
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, fmt.Errorf("empty input text")
	}

	// Try to match scaling pattern: "scale <target> to <N> in ns <namespace>"
	if matches := p.patterns["scaling"].FindStringSubmatch(text); matches != nil {
		replicas, err := strconv.Atoi(matches[2])
		if err != nil {
			return nil, fmt.Errorf("invalid replica count: %s", matches[2])
		}

		intent := map[string]interface{}{
			"intent_type": "scaling",
			"target":      matches[1],
			"replicas":    replicas,
		}

		// Add namespace if provided
		if len(matches) > 3 && matches[3] != "" {
			intent["namespace"] = matches[3]
		} else {
			intent["namespace"] = "default"
		}

		return intent, nil
	}

	// Try to match deploy pattern: "deploy <target> in ns <namespace>"
	if matches := p.patterns["deploy"].FindStringSubmatch(text); matches != nil {
		intent := map[string]interface{}{
			"intent_type": "deployment",
			"target":      matches[1],
		}

		// Add namespace if provided
		if len(matches) > 2 && matches[2] != "" {
			intent["namespace"] = matches[2]
		} else {
			intent["namespace"] = "default"
		}

		return intent, nil
	}

	// Try to match delete pattern: "delete <target> from ns <namespace>"
	if matches := p.patterns["delete"].FindStringSubmatch(text); matches != nil {
		intent := map[string]interface{}{
			"intent_type": "deletion",
			"target":      matches[1],
		}

		// Add namespace if provided
		if len(matches) > 2 && matches[2] != "" {
			intent["namespace"] = matches[2]
		} else {
			intent["namespace"] = "default"
		}

		return intent, nil
	}

	// Try to match update pattern: "update <target> set <key>=<value> in ns <namespace>"
	if matches := p.patterns["update"].FindStringSubmatch(text); matches != nil {
		intent := map[string]interface{}{
			"intent_type": "configuration",
			"target":      matches[1],
			"config": map[string]interface{}{
				matches[2]: matches[3],
			},
		}

		// Add namespace if provided
		if len(matches) > 4 && matches[4] != "" {
			intent["namespace"] = matches[4]
		} else {
			intent["namespace"] = "default"
		}

		return intent, nil
	}

	// No pattern matched
	return nil, fmt.Errorf("unable to parse intent from text: %s", text)
}

// ValidateIntent validates the intent against expected structure
func ValidateIntent(intent map[string]interface{}) error {
	// Check required fields
	intentType, ok := intent["intent_type"].(string)
	if !ok || intentType == "" {
		return fmt.Errorf("missing or invalid intent_type")
	}

	target, ok := intent["target"].(string)
	if !ok || target == "" {
		return fmt.Errorf("missing or invalid target")
	}

	// Validate based on intent type
	switch intentType {
	case "scaling":
		replicas, ok := intent["replicas"].(int)
		if !ok || replicas < 0 {
			return fmt.Errorf("invalid replicas value for scaling intent")
		}
	case "deployment", "deletion":
		// These just need target and namespace
	case "configuration":
		config, ok := intent["config"].(map[string]interface{})
		if !ok || len(config) == 0 {
			return fmt.Errorf("invalid or empty config for configuration intent")
		}
	default:
		return fmt.Errorf("unknown intent_type: %s", intentType)
	}

	// Validate namespace
	namespace, ok := intent["namespace"].(string)
	if !ok || namespace == "" {
		return fmt.Errorf("missing or invalid namespace")
	}

	return nil
}

// IntentToJSON converts an intent map to JSON string
func IntentToJSON(intent map[string]interface{}) (string, error) {
	data, err := json.Marshal(intent)
	if err != nil {
		return "", fmt.Errorf("failed to marshal intent: %w", err)
	}
	return string(data), nil
}
