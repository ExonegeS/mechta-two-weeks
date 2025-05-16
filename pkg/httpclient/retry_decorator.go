package httpclient

import (
	"net/http"
	"time"
)

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
