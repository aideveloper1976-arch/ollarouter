package engine

import (
	"fmt"
	"strings"

	"github.com/aideveloper1976-arch/ollarouter/internal/provider"
	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

// ModelResolver resolves model names to provider+model pairs.
type ModelResolver struct {
	registry      *provider.Registry
	virtualModels map[string][]string // virtual model name -> fallback chain
}

// NewModelResolver creates a new model resolver.
func NewModelResolver(registry *provider.Registry, virtualModels map[string][]string) *ModelResolver {
	if virtualModels == nil {
		virtualModels = make(map[string][]string)
	}
	return &ModelResolver{registry: registry, virtualModels: virtualModels}
}

// ResolveResult holds the resolved provider and model.
type ResolveResult struct {
	ProviderName string
	ModelName    string
	Provider     types.Provider
	State        *types.ProviderState
}

// ResolveModel resolves a model name to a provider and model.
func (r *ModelResolver) ResolveModel(modelName string) (*ResolveResult, error) {
	for _, ps := range r.registry.List() {
		for _, m := range ps.ModelCache {
			if m.Name == modelName {
				return &ResolveResult{
					ProviderName: ps.Config.Name,
					ModelName:    modelName,
					Provider:     ps.Provider,
					State:        ps,
				}, nil
			}
		}
	}

	if strings.Contains(modelName, "/") {
		parts := strings.SplitN(modelName, "/", 2)
		providerName := parts[0]
		modelShort := parts[1]

		ps, ok := r.registry.Get(providerName)
		if !ok {
			return nil, fmt.Errorf("%w: provider %q not found", types.ErrProviderModelNotFound, providerName)
		}

		if modelShort != "" && modelShort != "*" {
			return &ResolveResult{
				ProviderName: providerName,
				ModelName:    modelShort,
				Provider:     ps.Provider,
				State:        ps,
			}, nil
		}
	}

	for _, ps := range r.registry.List() {
		for _, m := range ps.ModelCache {
			if m.Name == modelName || strings.HasSuffix(m.Name, "/"+modelName) {
				return &ResolveResult{
					ProviderName: ps.Config.Name,
					ModelName:    modelName,
					Provider:     ps.Provider,
					State:        ps,
				}, nil
			}
		}
	}

	return nil, fmt.Errorf("%w: model %q not found", types.ErrProviderModelNotFound, modelName)
}

// ResolveVirtualModel resolves a virtual model name to its fallback chain.
// Returns (chain, nil) ONLY for actual virtual models. Returns (nil, error)
// for real provider models so callers can fall through to non-virtual resolution.
func (r *ModelResolver) ResolveVirtualModel(virtualName string) ([]string, error) {
	if chain, ok := r.virtualModels[virtualName]; ok {
		if len(chain) == 0 {
			return nil, fmt.Errorf("%w: virtual model %q has empty fallback chain", types.ErrProviderModelNotFound, virtualName)
		}
		return chain, nil
	}
	return nil, fmt.Errorf("%w: model %q not found as virtual model", types.ErrProviderModelNotFound, virtualName)
}
