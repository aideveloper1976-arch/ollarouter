package provider

import (
	"fmt"

	"github.com/aideveloper1976-arch/ollarouter/internal/provider/anthropic"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider/chatgpt"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider/google"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider/ollama"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider/openai"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider/openaicompat"
)

// NewProvider creates a provider adapter from the given config.
func NewProvider(cfg ProviderConfig) (Provider, error) {
	return NewProviderWithResolver(cfg, nil)
}

// NewProviderWithResolver creates a provider adapter, passing a credential
// resolver to provider types that support upstream auth (currently chatgpt-plus).
// A nil resolver is treated as "no resolver".
func NewProviderWithResolver(cfg ProviderConfig, resolver chatgpt.CredentialResolver) (Provider, error) {
	switch cfg.Type {
	case "openai":
		return openai.NewAdapter(cfg)
	case "anthropic":
		return anthropic.NewAdapter(cfg)
	case "google":
		return google.NewAdapter(cfg)
	case "openrouter":
		return openaicompat.NewAdapter(cfg)
	case "deepseek":
		return openaicompat.NewAdapter(cfg)
	case "openai-compatible":
		return openaicompat.NewAdapter(cfg)
	case "ollama":
		return ollama.NewAdapter(cfg)
	case "chatgpt-plus":
		if resolver != nil {
			return chatgpt.NewAdapterWithResolver(cfg, resolver)
		}
		return chatgpt.NewAdapter(cfg)
	default:
		return nil, fmt.Errorf("unknown provider type: %s", cfg.Type)
	}
}
