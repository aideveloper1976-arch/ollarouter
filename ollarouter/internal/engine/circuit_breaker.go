package engine

import (
	"sync"
	"time"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// CircuitBreakerState represents the state of a circuit breaker.
type CircuitBreakerState string

const (
	StateClosed   CircuitBreakerState = "closed"
	StateOpen     CircuitBreakerState = "open"
	StateHalfOpen CircuitBreakerState = "half-open"
)

// CircuitBreaker implements the circuit breaker pattern.
type CircuitBreaker struct {
	mu       sync.RWMutex
	state    CircuitBreakerState
	failures int
	lastOpen time.Time

	threshold int
	cooldown  time.Duration
}

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(threshold int, cooldown time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if cooldown <= 0 {
		cooldown = 30 * time.Second
	}
	return &CircuitBreaker{
		state:     StateClosed,
		threshold: threshold,
		cooldown:  cooldown,
	}
}

// State returns the current state.
func (cb *CircuitBreaker) State() CircuitBreakerState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// AllowRequest checks if a request should be allowed through.
func (cb *CircuitBreaker) AllowRequest() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case StateClosed:
		return true
	case StateOpen:
		if time.Since(cb.lastOpen) >= cb.cooldown {
			cb.state = StateHalfOpen
			return true
		}
		return false
	case StateHalfOpen:
		return true
	default:
		return true
	}
}

// RecordSuccess records a successful request.
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	if cb.state == StateHalfOpen {
		cb.state = StateClosed
	}
	cb.failures = 0
}

// RecordFailure records a failed request.
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	if cb.failures >= cb.threshold {
		cb.state = StateOpen
		cb.lastOpen = time.Now()
	}
}

// GetStatus returns the current circuit breaker status.
func (cb *CircuitBreaker) GetStatus() *types.CircuitBreakerStatus {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return &types.CircuitBreakerStatus{
		State:    string(cb.state),
		Failures: cb.failures,
		LastOpen: cb.lastOpen,
	}
}
