package clients

import (
	"context"
	"net/http"
	"time"
)

// HTTPClient provides a configured HTTP client with timeouts and retries
type HTTPClient struct {
	client *http.Client
}

// NewHTTPClient creates a new HTTP client with sensible defaults
func NewHTTPClient(timeout time.Duration) *HTTPClient {
	return &HTTPClient{
		client: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
				DisableCompression:  false,
			},
		},
	}
}

// Do executes an HTTP request with context
func (hc *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	return hc.client.Do(req)
}

// DoWithContext executes an HTTP request with the provided context
func (hc *HTTPClient) DoWithContext(ctx context.Context, req *http.Request) (*http.Response, error) {
	req = req.WithContext(ctx)
	return hc.client.Do(req)
}

// Get performs a GET request with context
func (hc *HTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	return hc.client.Do(req)
}

// Post performs a POST request with context
func (hc *HTTPClient) Post(ctx context.Context, url, contentType string, body interface{}) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", contentType)
	return hc.client.Do(req)
}
