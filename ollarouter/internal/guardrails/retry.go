package guardrails

import (
	"context"
	"errors"
	"fmt"
)

// RetryLoop executes fn up to maxRetries+1 times until it succeeds.
// A zero or negative maxRetries means one attempt with no retries.
// All errors are retried (the loop will not stop early for non-retryable errors).
func RetryLoop(ctx context.Context, maxRetries int, fn func(ctx context.Context) (string, error)) (string, error) {
	if maxRetries < 0 {
		maxRetries = 0
	}

	attempts := maxRetries + 1
	var lastErr error

	for i := 0; i < attempts; i++ {
		select {
		case <-ctx.Done():
			return "", fmt.Errorf("context cancelled: %w", ctx.Err())
		default:
		}

		result, err := fn(ctx)
		if err == nil {
			return result, nil
		}

		lastErr = err
	}

	return "", fmt.Errorf("all %d retries exhausted: %w", maxRetries, lastErr)
}

// IsRetryable returns true if the error is retryable (tool call validation error).
func IsRetryable(err error) bool {
	if errors.Is(err, ErrInvalidToolCall) {
		return true
	}
	return false
}
