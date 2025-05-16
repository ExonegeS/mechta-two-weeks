package httpclient

import (
	"fmt"
	"sync"
	"time"
)

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

	if cb.failures > cb.maxFailures {
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
