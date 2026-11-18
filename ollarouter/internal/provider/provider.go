package provider

// This file re-exports types from internal/types for backward compatibility.
// New code should import directly from internal/types.

import "github.com/aideveloper1976-arch/ollarouter/internal/types"

// Provider is the interface all provider adapters must implement.
type Provider = types.Provider

// Sentinel errors
var (
	ErrProviderUnavailable   = types.ErrProviderUnavailable
	ErrProviderTimeout       = types.ErrProviderTimeout
	ErrProviderAuthFailed    = types.ErrProviderAuthFailed
	ErrProviderRateLimited   = types.ErrProviderRateLimited
	ErrProviderModelNotFound = types.ErrProviderModelNotFound
	ErrAllFallbacksExhausted = types.ErrAllFallbacksExhausted
)

// Type aliases for all data types
type (
	StreamHandler        = types.StreamHandler
	ChatRequest          = types.ChatRequest
	Message              = types.Message
	ToolCall             = types.ToolCall
	RequestOptions       = types.RequestOptions
	GenerateRequest      = types.GenerateRequest
	ChatResponse         = types.ChatResponse
	GenerateResponse     = types.GenerateResponse
	StreamChunk          = types.StreamChunk
	Metrics              = types.Metrics
	TokenUsage           = types.TokenUsage
	ModelInfo            = types.ModelInfo
	ModelDetails         = types.ModelDetails
	ProviderHealth       = types.ProviderHealth
	ProviderState        = types.ProviderState
	ProviderConfig       = types.ProviderConfig
	AuthSettings         = types.AuthSettings
	ModelConfig          = types.ModelConfig
	PricingConfig        = types.PricingConfig
	ProviderBalance      = types.ProviderBalance
	CircuitBreakerStatus = types.CircuitBreakerStatus
)

type BalanceProvider = types.BalanceProvider
