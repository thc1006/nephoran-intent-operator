package llm

import (
	"context"
	"net/http"
)

// Client is a client for the LLM processor.
type Client struct {
	httpClient *http.Client
	url        string
}

// NewClient creates a new LLM client.
func NewClient(url string) *Client {
	return &Client{
		httpClient: &http.Client{},
		url:        url,
	}
}

func (c *Client) ProcessIntent(ctx context.Context, intent string) (string, error) {
	// TODO: Implement the actual call to the LLM processor
	return "{}", nil
}
