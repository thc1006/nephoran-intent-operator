package ingest

import (
	"context"
	"fmt"
	"math"
	"os"
	"regexp"
	"strconv"
	"strings"
)

// getEnv gets an environment variable with a default fallback.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// IntentProvider defines the interface for intent parsing providers.

type IntentProvider interface {
	// ParseIntent converts natural language text to structured intent.

	ParseIntent(ctx context.Context, text string) (map[string]interface{}, error)

	// Name returns the provider name.

	Name() string
}

// RulesProvider implements deterministic parsing using regex patterns.

type RulesProvider struct {
	patterns map[string]*regexp.Regexp
}

// NewRulesProvider creates a new rules-based provider.

func NewRulesProvider() *RulesProvider {
	return &RulesProvider{
		patterns: map[string]*regexp.Regexp{
			// Primary pattern: scale <target> to <N> in ns <namespace>.

			"scale_full": regexp.MustCompile(`(?i)scale\s+([a-z0-9\-]+)\s+to\s+(\d+)\s+in\s+ns\s+([a-z0-9\-]+)`),

			// Alternative: scale <target> to <N> (default namespace).

			"scale_simple": regexp.MustCompile(`(?i)scale\s+([a-z0-9\-]+)\s+to\s+(\d+)`),

			// Alternative: scale out/in <target> by <N> in ns <namespace>.

			"scale_out": regexp.MustCompile(`(?i)scale\s+out\s+([a-z0-9\-]+)\s+by\s+(\d+)(?:\s+in\s+ns\s+([a-z0-9\-]+))?`),

			"scale_in": regexp.MustCompile(`(?i)scale\s+in\s+([a-z0-9\-]+)\s+by\s+(\d+)(?:\s+in\s+ns\s+([a-z0-9\-]+))?`),
		},
	}
}

// Name returns the provider name.

func (p *RulesProvider) Name() string {
	return "rules"
}

// ParseIntent converts natural language to structured intent using rules.

func (p *RulesProvider) ParseIntent(ctx context.Context, text string) (map[string]interface{}, error) {
	text = strings.TrimSpace(text)

	// Try full scale pattern.

	if m := p.patterns["scale_full"].FindStringSubmatch(text); len(m) == 4 {

		replicas, err := strconv.Atoi(m[2])
		if err != nil {
			return nil, fmt.Errorf("invalid replica count: %s", m[2])
		}
		// Security fix (G115): Validate bounds for replica count
		if replicas < 0 || replicas > math.MaxInt32 {
			return nil, fmt.Errorf("replica count %d out of valid range (0-%d)", replicas, math.MaxInt32)
		}

		ns := "default"
		if len(m) > 3 && m[3] != "" {
			ns = m[3]
		}

		return map[string]interface{}{
			"intent_type":      "scaling",
			"target":           m[1],
			"replicas":         replicas,
			"namespace":        ns,
			"source":           "user",
			"status":           "pending",
			"target_resources": []string{fmt.Sprintf("deployment/%s", m[1])},
		}, nil

	}

	// Try simple scale pattern (default namespace).

	if m := p.patterns["scale_simple"].FindStringSubmatch(text); len(m) == 3 {

		replicas, err := strconv.Atoi(m[2])
		if err != nil {
			return nil, fmt.Errorf("invalid replica count: %s", m[2])
		}
		// Security fix (G115): Validate bounds for replica count
		if replicas < 0 || replicas > math.MaxInt32 {
			return nil, fmt.Errorf("replica count %d out of valid range (0-%d)", replicas, math.MaxInt32)
		}

		return map[string]interface{}{
			"intent_type":      "scaling",
			"target":           m[1],
			"replicas":         replicas,
			"namespace":        "default",
			"source":           "user",
			"status":           "pending",
			"target_resources": []string{fmt.Sprintf("deployment/%s", m[1])},
		}, nil

	}

	// Try scale out pattern.

	if m := p.patterns["scale_out"].FindStringSubmatch(text); len(m) >= 3 {

		delta, err := strconv.Atoi(m[2])
		if err != nil {
			return nil, fmt.Errorf("invalid delta count: %s", m[2])
		}
		// Security fix (G115): Validate bounds for delta count
		if delta < 0 || delta > math.MaxInt32 {
			return nil, fmt.Errorf("delta count %d out of valid range (0-%d)", delta, math.MaxInt32)
		}

		ns := "default"

		if len(m) > 3 && m[3] != "" {
			ns = m[3]
		}

		// For MVP, treat delta as absolute replica count.

		return map[string]interface{}{
			"intent_type":      "scaling",
			"target":           m[1],
			"replicas":         delta,
			"namespace":        ns,
			"source":           "user",
			"status":           "pending",
			"target_resources": []string{fmt.Sprintf("deployment/%s", m[1])},
		}, nil

	}

	// Try scale in pattern.

	if m := p.patterns["scale_in"].FindStringSubmatch(text); len(m) >= 3 {

		delta, err := strconv.Atoi(m[2])
		if err != nil {
			return nil, fmt.Errorf("invalid delta count: %s", m[2])
		}
		// Security fix (G115): Validate bounds for delta count
		if delta < 0 || delta > math.MaxInt32 {
			return nil, fmt.Errorf("delta count %d out of valid range (0-%d)", delta, math.MaxInt32)
		}

		ns := "default"

		if len(m) > 3 && m[3] != "" {
			ns = m[3]
		}

		// For MVP, scale in always sets minimum 1 replica.

		_ = delta

		return map[string]interface{}{
			"intent_type":      "scaling",
			"target":           m[1],
			"replicas":         1,
			"namespace":        ns,
			"source":           "user",
			"status":           "pending",
			"target_resources": []string{fmt.Sprintf("deployment/%s", m[1])},
		}, nil

	}

	return nil, fmt.Errorf("unable to parse intent from text: %s", text)
}

// MockLLMProvider simulates an LLM provider but returns the same results as rules.

type MockLLMProvider struct {
	rulesProvider *RulesProvider
}

// NewMockLLMProvider creates a new mock LLM provider.

func NewMockLLMProvider() *MockLLMProvider {
	return &MockLLMProvider{
		rulesProvider: NewRulesProvider(),
	}
}

// Name returns the provider name.

func (p *MockLLMProvider) Name() string {
	return "mock"
}

// ParseIntent simulates LLM processing but uses rules internally.

func (p *MockLLMProvider) ParseIntent(ctx context.Context, text string) (map[string]interface{}, error) {
	// Mock provider returns the same result as rules provider.

	// In a real implementation, this would call an actual LLM API.

	return p.rulesProvider.ParseIntent(ctx, text)
}

// ProviderFactory creates providers based on mode.

func NewProvider(mode, provider string) (IntentProvider, error) {
	// Default to rules mode if not specified.

	if mode == "" {
		mode = "rules"
	}

	switch mode {

	case "rules":

		return NewRulesProvider(), nil

	case "llm":

		// Default to mock provider if not specified.

		if provider == "" {
			provider = "mock"
		}

		switch provider {

		case "mock":

			return NewMockLLMProvider(), nil

		case "ollama":
			// Get Ollama configuration from environment or use defaults
			endpoint := getEnv("OLLAMA_ENDPOINT", "http://ollama-service.ollama.svc.cluster.local:11434")
			model := getEnv("OLLAMA_MODEL", "llama3.1")
			return NewOllamaProvider(endpoint, model), nil

		default:

			return nil, fmt.Errorf("unknown LLM provider: %s", provider)

		}

	default:

		return nil, fmt.Errorf("unknown mode: %s", mode)

	}
}
