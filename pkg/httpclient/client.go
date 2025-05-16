package httpclient

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type APIClient struct {
	baseURL    *url.URL
	httpClient HTTPClient
	circuit    *CircuitBreaker
}

type APIError struct {
	StatusCode int
	Message    error
	Body       []byte
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API request failed with status %d", e.StatusCode)
}

func NewAPIError(resp *http.Response) error {
	var (
		code int
		body []byte
	)
	if resp != nil {
		code = resp.StatusCode
		body, _ = io.ReadAll(resp.Body)
	}
	return &APIError{
		StatusCode: code,
		Body:       body,
	}
}

type OptionsSt struct {
	Timeout time.Duration
	Params  url.Values
	Headers http.Header

	RetryCount         int
	RetryInterval      time.Duration
	InsecureSkipVerify bool
}

func (o *OptionsSt) Normalize() {
	if o.Timeout == 0 {
		o.Timeout = 15 * time.Second
	}
	if o.RetryCount < 0 {
		o.RetryCount = 0
	}
	if o.RetryInterval == 0 {
		o.RetryInterval = 1 * time.Second
	}
}

func NewAPIClient(baseURL string, opts *OptionsSt, cb *CircuitBreaker) (*APIClient, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: opts.InsecureSkipVerify,
		},
	}

	client := &http.Client{
		Timeout:   opts.Timeout,
		Transport: transport,
	}

	return &APIClient{
		baseURL:    u,
		httpClient: NewRetryDecorator(client, opts.RetryCount, opts.RetryInterval),
		circuit:    cb,
	}, nil
}

func (c *APIClient) Execute(ctx context.Context, req *http.Request, v any) error {
	return c.circuit.Execute(func() error {
		resp, err := c.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode < 200 || resp.StatusCode > 300 {
			return NewAPIError(resp)
		}

		if v != nil {
			if err := json.NewDecoder(resp.Body).Decode(v); err != nil {
				return fmt.Errorf("failed to decode response: %w", err)
			}
		}
		return nil
	})
}
