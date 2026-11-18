package engine

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// RetryConfig holds retry configuration.
type RetryConfig struct {
	MaxRetries int
	BaseDelay  time.Duration
	MaxDelay   time.Duration
	UseJitter  bool
}

// DefaultRetryConfig returns sensible defaults.
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxRetries: 3,
		BaseDelay:  1 * time.Second,
		MaxDelay:   30 * time.Second,
		UseJitter:  true,
	}
}

// RetryEngine handles retry logic with exponential backoff.
type RetryEngine struct {
	config RetryConfig
}

// NewRetryEngine creates a new retry engine.
func NewRetryEngine(config RetryConfig) *RetryEngine {
	return &RetryEngine{config: config}
}

// ShouldRetry checks if the error is retryable.
func ShouldRetry(err error) bool {
	if err == nil {
		return false
	}
	switch {
	case isError(err, types.ErrProviderUnavailable):
		return true
	case isError(err, types.ErrProviderTimeout):
		return true
	case isError(err, types.ErrProviderRateLimited):
		return true
	default:
		return false
	}
}

// isError checks if err is or wraps the target error.
func isError(err, target error) bool {
	for err != nil {
		if err == target {
			return true
		}
		unwrapped, ok := err.(interface{ Unwrap() error })
		if !ok {
			return false
		}
		err = unwrapped.Unwrap()
	}
	return false
}

// DoWithRetry executes a function with retry logic.
func (e *RetryEngine) DoWithRetry(ctx context.Context, fn func(context.Context) error) error {
	var lastErr error

	for i := 0; i <= e.config.MaxRetries; i++ {
		if i > 0 {
			if ctx.Err() != nil {
				return ctx.Err()
			}

			delay := e.getDelay(i)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
		}

		err := fn(ctx)
		if err == nil {
			return nil
		}

		lastErr = err
		if !ShouldRetry(err) {
			return err
		}
	}

	return lastErr
}

func (e *RetryEngine) getDelay(attempt int) time.Duration {
	delay := float64(e.config.BaseDelay) * math.Pow(2, float64(attempt-1))
	if delay > float64(e.config.MaxDelay) {
		delay = float64(e.config.MaxDelay)
	}

	if e.config.UseJitter {
		jitter := delay * (0.75 + 0.5*rand.Float64())
		return time.Duration(jitter)
	}

	return time.Duration(delay)
}
