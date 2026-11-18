package provider

import (
	"context"
	"fmt"
	"sync"

	"github.com/aideveloper1976-arch/ollarouter/internal/config"
	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// Registry manages all configured provider instances.
type Registry struct {
	providers map[string]*types.ProviderState
	mu        sync.RWMutex
}

// NewRegistry creates a new provider registry.
func NewRegistry() *Registry {
	return &Registry{
		providers: make(map[string]*types.ProviderState),
	}
}

// Get returns a provider state by name.
func (r *Registry) Get(name string) (*types.ProviderState, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	ps, ok := r.providers[name]
	return ps, ok
}

// Add creates and adds a new provider.
func (r *Registry) Add(cfg config.ProviderConfig, p types.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.providers[cfg.Name]; exists {
		return fmt.Errorf("provider %q already exists", cfg.Name)
	}

	ps := &types.ProviderState{
		Provider: p,
		Config: types.ProviderConfig{
			Name:             cfg.Name,
			Type:             cfg.Type,
			BaseURL:          cfg.BaseURL,
			APIKey:           cfg.APIKey,
			OrganizationID:   cfg.OrganizationID,
			AnthropicVersion: cfg.AnthropicVersion,
			AutoDiscover:     cfg.AutoDiscover,
			PollingInterval:  cfg.PollingInterval,
			MaxRetries:       cfg.MaxRetries,
			TimeoutSec:       cfg.TimeoutSec,
			SessionCookie:    cfg.SessionCookie,
			UserAgent:        cfg.UserAgent,
			Auth:             mapAuthSettings(cfg.Auth),
			Guardrails:       mapGuardrailsConfig(cfg.Guardrails),
		},
		CircuitBreaker: &types.CircuitBreakerStatus{
			State: "closed",
		},
		Health: types.ProviderHealth{
			Status: "unknown",
		},
	}

	if cfg.Pricing != nil {
		ps.Config.Pricing = &types.PricingConfig{
			InputPer1K:  cfg.Pricing.InputPer1K,
			OutputPer1K: cfg.Pricing.OutputPer1K,
		}
	}

	modelNames := make(map[string]bool)
	if cfg.AutoDiscover {
		if models, err := p.ListModels(context.TODO()); err == nil {
			ps.ModelCache = models
			for _, m := range models {
				modelNames[m.Name] = true
			}
		}
	}

	for _, m := range cfg.Models {
		ps.Config.Models = append(ps.Config.Models, types.ModelConfig{
			Name:       m.Name,
			MaxTokens:  m.MaxTokens,
			ContextLen: m.ContextLen,
		})
		modelName := cfg.Name + "/" + m.Name
		if !modelNames[modelName] {
			ps.ModelCache = append(ps.ModelCache, types.ModelInfo{
				Name:         modelName,
				Size:         0,
				ProviderName: cfg.Name,
			})
			modelNames[modelName] = true
		}
	}

	r.providers[cfg.Name] = ps
	return nil
}

// Remove removes a provider by name.
func (r *Registry) Remove(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.providers, name)
}

// Update replaces a provider's configuration and instance.
// Optional fields from the new config take precedence; if not provided,
// values from the existing provider config are preserved.
func (r *Registry) Update(name string, cfg config.ProviderConfig, p types.Provider) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	existing, exists := r.providers[name]
	if !exists {
		return fmt.Errorf("provider %q not found", name)
	}

	// Merge optional fields: use cfg value if non-zero, otherwise preserve existing
	sessionCookie := cfg.SessionCookie
	if sessionCookie == "" {
		sessionCookie = existing.Config.SessionCookie
	}
	userAgent := cfg.UserAgent
	if userAgent == "" {
		userAgent = existing.Config.UserAgent
	}
	orgID := cfg.OrganizationID
	if orgID == "" {
		orgID = existing.Config.OrganizationID
	}
	anthropicVersion := cfg.AnthropicVersion
	if anthropicVersion == "" {
		anthropicVersion = existing.Config.AnthropicVersion
	}
	pollingInterval := cfg.PollingInterval
	if pollingInterval == 0 {
		pollingInterval = existing.Config.PollingInterval
	}
	maxRetries := cfg.MaxRetries
	if maxRetries == 0 {
		maxRetries = existing.Config.MaxRetries
	}
	timeoutSec := cfg.TimeoutSec
	if timeoutSec == 0 {
		timeoutSec = existing.Config.TimeoutSec
	}
	headers := cfg.Headers
	if headers == nil {
		headers = existing.Config.Headers
	}
	auth := cfg.Auth
	if auth == nil && existing.Config.Auth != nil {
		auth = &config.AuthSettings{
			Mode:            existing.Config.Auth.Mode,
			ClientID:        existing.Config.Auth.ClientID,
			ClientSecret:    existing.Config.Auth.ClientSecret,
			RedirectURI:     existing.Config.Auth.RedirectURI,
			CallbackPort:    existing.Config.Auth.CallbackPort,
			KeychainService: existing.Config.Auth.KeychainService,
			TokenEndpoint:   existing.Config.Auth.TokenEndpoint,
			AuthEndpoint:    existing.Config.Auth.AuthEndpoint,
			Scopes:          existing.Config.Auth.Scopes,
		}
	}
	pricing := cfg.Pricing
	if pricing == nil {
		if existing.Config.Pricing != nil {
			pricing = &config.PricingConfig{
				InputPer1K:  existing.Config.Pricing.InputPer1K,
				OutputPer1K: existing.Config.Pricing.OutputPer1K,
			}
		}
	}
	var pModels []types.ModelConfig
	if len(cfg.Models) > 0 {
		for _, m := range cfg.Models {
			pModels = append(pModels, types.ModelConfig{Name: m.Name, MaxTokens: m.MaxTokens, ContextLen: m.ContextLen})
		}
	} else {
		pModels = existing.Config.Models
	}

	var pAuth *types.AuthSettings
	if auth != nil {
		pAuth = mapAuthSettings(auth)
	}
	var pPricing *types.PricingConfig
	if pricing != nil {
		pPricing = &types.PricingConfig{
			InputPer1K:  pricing.InputPer1K,
			OutputPer1K: pricing.OutputPer1K,
		}
	}

	ps := &types.ProviderState{
		Provider: p,
		Config: types.ProviderConfig{
			Name:             cfg.Name,
			Type:             cfg.Type,
			BaseURL:          cfg.BaseURL,
			APIKey:           cfg.APIKey,
			OrganizationID:   orgID,
			AnthropicVersion: anthropicVersion,
			AutoDiscover:     cfg.AutoDiscover,
			PollingInterval:  pollingInterval,
			Models:           pModels,
			Pricing:          pPricing,
			MaxRetries:       maxRetries,
			TimeoutSec:       timeoutSec,
			Headers:          headers,
			SessionCookie:    sessionCookie,
			UserAgent:        userAgent,
			Auth:             pAuth,
			Guardrails:       mapGuardrailsConfig(cfg.Guardrails),
		},
		CircuitBreaker: &types.CircuitBreakerStatus{State: "closed"},
		Health:         types.ProviderHealth{Status: "unknown"},
	}

	modelNames := make(map[string]bool)
	if cfg.AutoDiscover {
		if models, err := p.ListModels(context.TODO()); err == nil {
			ps.ModelCache = models
			for _, m := range models {
				modelNames[m.Name] = true
			}
		}
	}

	for _, m := range cfg.Models {
		modelName := cfg.Name + "/" + m.Name
		if !modelNames[modelName] {
			ps.ModelCache = append(ps.ModelCache, types.ModelInfo{
				Name:         modelName,
				Size:         0,
				ProviderName: cfg.Name,
			})
			modelNames[modelName] = true
		}
	}

	r.providers[name] = ps
	return nil
}

// List returns all provider states.
func (r *Registry) List() []*types.ProviderState {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*types.ProviderState, 0, len(r.providers))
	for _, ps := range r.providers {
		result = append(result, ps)
	}
	return result
}

// AllProviders returns all providers (for model aggregation).
func (r *Registry) AllProviders() []types.Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]types.Provider, 0, len(r.providers))
	for _, ps := range r.providers {
		result = append(result, ps.Provider)
	}
	return result
}

// GetAllModels returns the aggregated model cache from all providers.
func (r *Registry) GetAllModels() []types.ModelInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var models []types.ModelInfo
	for _, ps := range r.providers {
		models = append(models, ps.ModelCache...)
	}
	return models
}

// RefreshModelCache refreshes the model cache for providers with auto-discovery.
func (r *Registry) RefreshModelCache() {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, ps := range r.providers {
		if ps.Config.AutoDiscover {
			models, err := ps.Provider.ListModels(context.TODO())
			if err == nil {
				ps.ModelCache = models
			}
		}
	}
}

// mapGuardrailsConfig converts config.GuardrailsConfig to types.GuardrailsConfig.
func mapGuardrailsConfig(cfg *config.GuardrailsConfig) *types.GuardrailsConfig {
	if cfg == nil {
		return nil
	}
	return &types.GuardrailsConfig{
		Enabled:            cfg.Enabled,
		ToolCallValidation: cfg.ToolCallValidation,
		RescueMalformed:    cfg.RescueMalformed,
		MixedOutputPolicy:  cfg.MixedOutputPolicy,
		MaxRetries:         cfg.MaxRetries,
	}
}

// mapAuthSettings converts config.AuthSettings to types.AuthSettings.
func mapAuthSettings(cfg *config.AuthSettings) *types.AuthSettings {
	if cfg == nil {
		return nil
	}
	return &types.AuthSettings{
		Mode:            cfg.Mode,
		ClientID:        cfg.ClientID,
		ClientSecret:    cfg.ClientSecret,
		RedirectURI:     cfg.RedirectURI,
		CallbackPort:    cfg.CallbackPort,
		KeychainService: cfg.KeychainService,
		TokenEndpoint:   cfg.TokenEndpoint,
		AuthEndpoint:    cfg.AuthEndpoint,
		Scopes:          cfg.Scopes,
	}
}

// UpdateHealth updates the health status for a provider.
func (r *Registry) UpdateHealth(name string, health types.ProviderHealth) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if ps, ok := r.providers[name]; ok {
		ps.Health = health
	}
}
