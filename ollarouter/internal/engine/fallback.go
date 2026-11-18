package engine

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/aideveloper1976-arch/ollarouter/internal/logger"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider"
	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// FallbackExecutor iterates through a fallback chain of providers.
type FallbackExecutor struct {
	registry      *provider.Registry
	retryEngine   *RetryEngine
	breakersMu    sync.Mutex
	breakers      map[string]*CircuitBreaker
	cbThreshold   int
	cbCooldown    time.Duration
	virtualModels map[string][]string // virtual model name -> fallback chain
	log           *logger.Logger
}

// NewFallbackExecutor creates a new fallback executor.
func NewFallbackExecutor(registry *provider.Registry, retryCfg RetryConfig, virtualModels map[string][]string, log *logger.Logger) *FallbackExecutor {
	if virtualModels == nil {
		virtualModels = make(map[string][]string)
	}
	return &FallbackExecutor{
		registry:      registry,
		retryEngine:   NewRetryEngine(retryCfg),
		breakers:      make(map[string]*CircuitBreaker),
		cbThreshold:   5,
		cbCooldown:    30 * time.Second,
		virtualModels: virtualModels,
		log:           log,
	}
}

// SetCircuitBreakerConfig sets the failure threshold and cooldown used for
// circuit breakers created after this call. Non-positive values keep defaults.
func (fe *FallbackExecutor) SetCircuitBreakerConfig(threshold int, cooldown time.Duration) {
	fe.breakersMu.Lock()
	defer fe.breakersMu.Unlock()
	if threshold > 0 {
		fe.cbThreshold = threshold
	}
	if cooldown > 0 {
		fe.cbCooldown = cooldown
	}
}

// BreakerStatus reports the live circuit breaker state for a provider, or
// false if no request has been routed to it through a fallback chain yet.
func (fe *FallbackExecutor) BreakerStatus(providerName string) (*types.CircuitBreakerStatus, bool) {
	fe.breakersMu.Lock()
	cb, ok := fe.breakers[providerName]
	fe.breakersMu.Unlock()
	if !ok {
		return nil, false
	}
	return cb.GetStatus(), true
}

// GetOrCreateBreaker gets or creates a circuit breaker for a provider.
// It is safe for concurrent use by request handlers.
func (fe *FallbackExecutor) GetOrCreateBreaker(providerName string) *CircuitBreaker {
	fe.breakersMu.Lock()
	defer fe.breakersMu.Unlock()
	if cb, ok := fe.breakers[providerName]; ok {
		return cb
	}
	cb := NewCircuitBreaker(fe.cbThreshold, fe.cbCooldown)
	fe.breakers[providerName] = cb
	return cb
}

// ExecuteWithFallback executes a function with fallback chain support.
// The fn receives the provider name and model name.
func (fe *FallbackExecutor) ExecuteWithFallback(
	ctx context.Context,
	chain []string,
	fn func(ctx context.Context, providerName, modelName string) error,
) (string, error) {
	seen := make(map[string]bool)

	for _, entry := range chain {
		// Parse "provider/model" format
		parts := strings.SplitN(entry, "/", 2)
		providerName := parts[0]
		modelName := ""
		if len(parts) > 1 {
			modelName = parts[1]
		}

		// Detect circular chains
		chainKey := providerName + ":" + modelName
		if seen[chainKey] {
			continue
		}
		seen[chainKey] = true

		ps, ok := fe.registry.Get(providerName)
		if !ok {
			continue
		}

		// Check circuit breaker
		cb := fe.GetOrCreateBreaker(providerName)
		if !cb.AllowRequest() {
			continue
		}

		actualModel := modelName
		if actualModel == "" || actualModel == "*" {
			// Use first available model from the provider
			models, _ := ps.Provider.ListModels(ctx)
			if len(models) > 0 {
				actualModel = models[0].Name
			} else {
				continue
			}
		}

		// Execute with retry
		err := fe.retryEngine.DoWithRetry(ctx, func(ctx context.Context) error {
			if fe.log != nil {
				fe.log.Info("routing request", "provider", providerName, "model", actualModel)
			}
			return fn(ctx, providerName, actualModel)
		})

		if err == nil {
			cb.RecordSuccess()
			return providerName, nil
		}

		cb.RecordFailure()

		if fe.log != nil {
			fe.log.Warn("provider request failed, attempting fallback", "provider", providerName, "error", err)
		}

		// If it's not a retryable error, continue to next fallback
		if !ShouldRetry(err) {
			continue
		}
	}

	return "", fmt.Errorf("%w: tried %v", types.ErrAllFallbacksExhausted, chain)
}

// BuildFallbackChain builds a fallback chain from virtual model config.
func (fe *FallbackExecutor) BuildFallbackChain(virtualModel string) []string {
	// Look up in virtual models config
	if chain, ok := fe.virtualModels[virtualModel]; ok {
		return chain
	}

	// Fall back to single-element chain for direct provider/model
	return []string{virtualModel}
}
