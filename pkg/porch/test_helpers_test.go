package porch

import (
	"net"
	"net/http"
	"time"
)

// newTestClient creates a Porch client for testing purposes, bypassing SSRF
// validation since httptest.Server uses localhost URLs. This function must
// only be used in test code.
func newTestClient(baseURL string, dryRun bool) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 50,
				IdleConnTimeout:     90 * time.Second,
				DialContext: (&net.Dialer{
					Timeout:   10 * time.Second,
					KeepAlive: 30 * time.Second,
				}).DialContext,
			},
		},
		dryRun: dryRun,
	}
}

// newTestClientWithAuth creates an authenticated Porch client for testing,
// bypassing SSRF validation since httptest.Server uses localhost URLs.
func newTestClientWithAuth(baseURL, token string, dryRun bool) *Client {
	baseTransport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 50,
		IdleConnTimeout:     90 * time.Second,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	client := &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		dryRun: dryRun,
	}

	if token != "" {
		client.httpClient.Transport = &authTransport{
			token: token,
			base:  baseTransport,
		}
	} else {
		client.httpClient.Transport = baseTransport
	}

	return client
}
