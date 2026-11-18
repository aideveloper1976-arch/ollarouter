package guardrails

import (
	"context"
	"errors"
	"testing"
)

func TestRetryLoop_SuccessFirstAttempt(t *testing.T) {
	attempts := 0
	result, err := RetryLoop(context.Background(), 3, func(ctx context.Context) (string, error) {
		attempts++
		return "valid response", nil
	})
	if err != nil {
		t.Fatalf("RetryLoop() error = %v", err)
	}
	if result != "valid response" {
		t.Errorf("result = %q, want 'valid response'", result)
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1", attempts)
	}
}

func TestRetryLoop_RetriesOnError(t *testing.T) {
	attempts := 0
	result, err := RetryLoop(context.Background(), 3, func(ctx context.Context) (string, error) {
		attempts++
		if attempts < 3 {
			return "", errors.New("temporary error")
		}
		return "success", nil
	})
	if err != nil {
		t.Fatalf("RetryLoop() error = %v", err)
	}
	if result != "success" {
		t.Errorf("result = %q, want 'success'", result)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3", attempts)
	}
}

func TestRetryLoop_ExhaustsRetries(t *testing.T) {
	attempts := 0
	result, err := RetryLoop(context.Background(), 2, func(ctx context.Context) (string, error) {
		attempts++
		return "", errors.New("persistent error")
	})
	if err == nil {
		t.Fatal("RetryLoop() expected error after exhausting retries")
	}
	if result != "" {
		t.Errorf("result = %q, want empty", result)
	}
	if attempts != 3 {
		t.Errorf("attempts = %d, want 3 (maxRetries=2 means 1 original + 2 retries = 3 total)", attempts)
	}
}

func TestRetryLoop_ContextCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err := RetryLoop(ctx, 3, func(ctx context.Context) (string, error) {
		return "", errors.New("should not be called")
	})
	if err == nil {
		t.Fatal("RetryLoop() expected error for cancelled context")
	}
}

func TestRetryLoop_ZeroRetries(t *testing.T) {
	attempts := 0
	_, err := RetryLoop(context.Background(), 0, func(ctx context.Context) (string, error) {
		attempts++
		return "", errors.New("error")
	})
	if err == nil {
		t.Fatal("RetryLoop() expected error with 0 retries")
	}
	if attempts != 1 {
		t.Errorf("attempts = %d, want 1 (one attempt with 0 retries)", attempts)
	}
}

func TestIsRetryable_InvalidToolCall(t *testing.T) {
	if !IsRetryable(ErrInvalidToolCall) {
		t.Errorf("IsRetryable(ErrInvalidToolCall) = false, want true")
	}
}

func TestIsRetryable_UnretryableError(t *testing.T) {
	if IsRetryable(errors.New("fatal error")) {
		t.Errorf("IsRetryable(fatal) = true, want false")
	}
}
