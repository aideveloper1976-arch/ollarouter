package engine

import (
	"fmt"
	"sync"
	"testing"
)

// TestGetOrCreateBreakerConcurrent guards against concurrent map writes when
// many requests hit providers for the first time at once. Run with -race.
func TestGetOrCreateBreakerConcurrent(t *testing.T) {
	fe := NewFallbackExecutor(nil, DefaultRetryConfig(), nil, nil)

	var wg sync.WaitGroup
	for i := 0; i < 64; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			name := fmt.Sprintf("provider-%d", i%8)
			if fe.GetOrCreateBreaker(name) == nil {
				t.Errorf("nil breaker for %s", name)
			}
		}(i)
	}
	wg.Wait()

	if a, b := fe.GetOrCreateBreaker("provider-1"), fe.GetOrCreateBreaker("provider-1"); a != b {
		t.Fatal("expected the same breaker instance for the same provider")
	}
}

func TestSetCircuitBreakerConfig(t *testing.T) {
	fe := NewFallbackExecutor(nil, DefaultRetryConfig(), nil, nil)
	fe.SetCircuitBreakerConfig(2, 0)

	cb := fe.GetOrCreateBreaker("p")
	cb.RecordFailure()
	if !cb.AllowRequest() {
		t.Fatal("breaker opened after 1 failure with threshold 2")
	}
	cb.RecordFailure()
	if cb.State() != StateOpen {
		t.Fatalf("expected open after 2 failures, got %s", cb.State())
	}
}
