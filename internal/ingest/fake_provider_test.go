package ingest

import (
	"context"
)

// FakeIntentProvider is a test implementation of the IntentProvider interface
// that returns deterministic responses for testing purposes.
type FakeIntentProvider struct {
	// ParseResult allows customizing the return value of ParseIntent
	ParseResult map[string]interface{}
	// ParseError allows customizing the error returned by ParseIntent
	ParseError error
	// ProviderName allows customizing the name returned by Name()
	ProviderName string
}

// NewFakeIntentProvider creates a new FakeIntentProvider with default values
func NewFakeIntentProvider() *FakeIntentProvider {
	return &FakeIntentProvider{
		ParseResult: map[string]interface{}{},
		ParseError:   nil,
		ProviderName: "fake",
	}
}

// ParseIntent implements the IntentProvider interface
// Returns the configured ParseResult and ParseError
func (f *FakeIntentProvider) ParseIntent(ctx context.Context, text string) (map[string]interface{}, error) {
	if f.ParseError != nil {
		return nil, f.ParseError
	}

	// Return a copy to avoid accidental modification
	result := make(map[string]interface{})
	for k, v := range f.ParseResult {
		result[k] = v
	}

	return result, nil
}

// Name implements the IntentProvider interface
func (f *FakeIntentProvider) Name() string {
	if f.ProviderName == "" {
		return "fake"
	}
	return f.ProviderName
}

// WithParseResult allows chaining to set custom parse result
func (f *FakeIntentProvider) WithParseResult(result map[string]interface{}) *FakeIntentProvider {
	f.ParseResult = result
	return f
}

// WithParseError allows chaining to set custom parse error
func (f *FakeIntentProvider) WithParseError(err error) *FakeIntentProvider {
	f.ParseError = err
	return f
}

// WithName allows chaining to set custom provider name
func (f *FakeIntentProvider) WithName(name string) *FakeIntentProvider {
	f.ProviderName = name
	return f
}

