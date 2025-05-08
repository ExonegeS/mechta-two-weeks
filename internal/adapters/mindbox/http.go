package mindbox

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"sync"
	"time"

	"github.com/ExonegeS/mechta-two-weeks/internal/core/domain"
)

type HTTPClient interface {
	Do(*http.Request) (*http.Response, error)
}

type APIClient struct {
	baseURL    *url.URL
	httpClient HTTPClient
	circuit    *CircuitBreaker
}

// region API Client

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

func NewAPIClient(baseURL string, opts *domain.OptionsSt, cb *CircuitBreaker) (*APIClient, error) {
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

// region Request Builder

type RequestBuilder struct {
	ctx      context.Context
	baseURL  *url.URL
	method   string
	endpoint string
	query    url.Values
	headers  http.Header
	body     io.Reader
}

func (c *APIClient) NewRequest(method, endpoint string) *RequestBuilder {
	return &RequestBuilder{
		ctx:      context.Background(),
		baseURL:  c.baseURL,
		method:   method,
		endpoint: endpoint,
		query:    make(url.Values),
		headers:  make(http.Header),
	}
}

func (b *RequestBuilder) WithHeader(key, value string) *RequestBuilder {
	b.headers.Add(key, value)
	return b
}

func (b *RequestBuilder) WithContext(ctx context.Context) *RequestBuilder {
	b.ctx = ctx
	return b
}

func (b *RequestBuilder) WithQueryParam(key, value string) *RequestBuilder {
	b.query.Add(key, value)
	return b
}

func (b *RequestBuilder) WithJSONBody(body any) *RequestBuilder {
	data, _ := json.Marshal(body)
	b.body = bytes.NewReader(data)
	b.headers.Set("Content-Type", "application/json")
	return b
}

func (b *RequestBuilder) Build() (*http.Request, error) {
	u := *b.baseURL
	u.Path = path.Join(u.Path, b.endpoint)
	u.RawQuery = b.query.Encode()

	req, err := http.NewRequestWithContext(b.ctx, b.method, u.String(), b.body)
	if err != nil {
		return nil, err
	}

	req.Header = b.headers
	return req, nil
}

// region Circuit breaker

type CircuitBreaker struct {
	mu           sync.Mutex
	failures     int
	maxFailures  int
	resetTimeout time.Duration
	lastFailure  time.Time
}

func NewCircuitBreaker(maxRetries int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxRetries,
		resetTimeout: resetTimeout,
	}
}

func (cb *CircuitBreaker) Execute(f func() error) error {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.failures > 0 && time.Since(cb.lastFailure) > cb.resetTimeout {
		cb.failures = 0
	}

	if cb.failures >= cb.maxFailures {
		return fmt.Errorf("circuit breaker open")
	}

	err := f()
	if err != nil {
		cb.failures++
		cb.lastFailure = time.Now()
		return err
	}
	return nil
}

// region Retry Decorator

type RetryDecorator struct {
	client     HTTPClient
	maxRetries int
	backoff    time.Duration
}

func NewRetryDecorator(client HTTPClient, maxRetries int, backoff time.Duration) *RetryDecorator {
	return &RetryDecorator{
		client:     client,
		maxRetries: maxRetries,
		backoff:    backoff,
	}
}

func (c *RetryDecorator) Do(req *http.Request) (resp *http.Response, err error) {
	getBody := req.GetBody
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 && getBody != nil {
			req.Body, _ = getBody()
		}

		resp, err = c.client.Do(req)
		if shouldRetry(err, resp) {
			time.Sleep(c.backoff)
			c.backoff *= 2
			continue
		}
		break
	}
	return
}

func shouldRetry(err error, resp *http.Response) bool {
	if err != nil {
		return true
	}
	return resp.StatusCode >= 500
}
