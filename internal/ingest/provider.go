
package ingest



import (

	"context"

	"fmt"

	"regexp"

	"strconv"

	"strings"

)



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

			"scale_in":  regexp.MustCompile(`(?i)scale\s+in\s+([a-z0-9\-]+)\s+by\s+(\d+)(?:\s+in\s+ns\s+([a-z0-9\-]+))?`),

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

		return map[string]interface{}{

			"intent_type": "scaling",

			"target":      m[1],

			"replicas":    replicas,

			"namespace":   m[3],

			"source":      "user",

		}, nil

	}



	// Try simple scale pattern (default namespace).

	if m := p.patterns["scale_simple"].FindStringSubmatch(text); len(m) == 3 {

		replicas, err := strconv.Atoi(m[2])

		if err != nil {

			return nil, fmt.Errorf("invalid replica count: %s", m[2])

		}

		return map[string]interface{}{

			"intent_type": "scaling",

			"target":      m[1],

			"replicas":    replicas,

			"namespace":   "default",

			"source":      "user",

		}, nil

	}



	// Try scale out pattern.

	if m := p.patterns["scale_out"].FindStringSubmatch(text); len(m) >= 3 {

		delta, err := strconv.Atoi(m[2])

		if err != nil {

			return nil, fmt.Errorf("invalid delta count: %s", m[2])

		}

		ns := "default"

		if len(m) > 3 && m[3] != "" {

			ns = m[3]

		}

		// Note: scale out by N means increase by N, so we'd need current count.

		// For MVP, we'll just use the delta as the new replica count.

		return map[string]interface{}{

			"intent_type": "scaling",

			"target":      m[1],

			"replicas":    delta, // In production, this would be current + delta

			"namespace":   ns,

			"source":      "user",

			"reason":      fmt.Sprintf("scale out by %d", delta),

		}, nil

	}



	// Try scale in pattern.

	if m := p.patterns["scale_in"].FindStringSubmatch(text); len(m) >= 3 {

		delta, err := strconv.Atoi(m[2])

		if err != nil {

			return nil, fmt.Errorf("invalid delta count: %s", m[2])

		}

		ns := "default"

		if len(m) > 3 && m[3] != "" {

			ns = m[3]

		}

		// Note: scale in by N means decrease by N.

		// For MVP, we'll use 1 as minimum.

		return map[string]interface{}{

			"intent_type": "scaling",

			"target":      m[1],

			"replicas":    1, // In production, this would be max(1, current - delta)

			"namespace":   ns,

			"source":      "user",

			"reason":      fmt.Sprintf("scale in by %d", delta),

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

		default:

			return nil, fmt.Errorf("unknown LLM provider: %s", provider)

		}

	default:

		return nil, fmt.Errorf("unknown mode: %s", mode)

	}

}

