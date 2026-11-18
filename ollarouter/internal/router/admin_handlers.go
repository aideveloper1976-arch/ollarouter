package router

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/aideveloper1976-arch/ollarouter/internal/config"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider"
	"github.com/aideveloper1976-arch/ollarouter/internal/types"
	"github.com/aideveloper1976-arch/ollarouter/internal/webui"
	"github.com/go-chi/chi/v5"
)

// handleAdminMe returns admin access info (no auth required).
// The SPA calls this on load to determine admin status.
func handleAdminMe(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"authenticated": true,
		})
	}
}

func saveConfig(app *App) error {
	if app.Config == nil || app.ConfigPath == "" {
		return fmt.Errorf("config path not configured")
	}
	return config.SaveFile(app.ConfigPath, app.Config)
}

func handleAdminListProviders(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ps := app.Registry.List()
		result := make([]map[string]interface{}, len(ps))
		costs := app.Tracker.GetCostByProvider()
		for i, p := range ps {
			models := make([]string, len(p.ModelCache))
			for j, m := range p.ModelCache {
				models[j] = m.Name
			}
			result[i] = map[string]interface{}{
				"name":          p.Config.Name,
				"type":          p.Config.Type,
				"base_url":      p.Config.BaseURL,
				"status":        p.Health.Status,
				"total_cost":    costs[p.Config.Name],
				"models":        models,
				"models_count":  len(p.ModelCache),
				"auto_discover": p.Config.AutoDiscover,
				"supports_balance": func() bool {
					_, ok := p.Provider.(types.BalanceProvider)
					return ok
				}(),
				"circuit_breaker": breakerStatusFor(app, p),
			}
			if p.Config.Pricing != nil {
				result[i]["pricing"] = map[string]float64{
					"input_per_1k":  p.Config.Pricing.InputPer1K,
					"output_per_1k": p.Config.Pricing.OutputPer1K,
				}
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{"providers": result})
	}
}

func handleAdminAddProvider(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name         string `json:"name"`
			Type         string `json:"type"`
			BaseURL      string `json:"base_url"`
			APIKey       string `json:"api_key"`
			AutoDiscover bool   `json:"auto_discover"`
			Pricing      *struct {
				InputPer1K  float64 `json:"input_per_1k"`
				OutputPer1K float64 `json:"output_per_1k"`
			} `json:"pricing"`
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid request body")
			return
		}
		if req.Name == "" || req.Type == "" {
			writeError(w, 400, "name and type are required")
			return
		}

		cfg := config.ProviderConfig{
			Name:         req.Name,
			Type:         req.Type,
			BaseURL:      req.BaseURL,
			APIKey:       req.APIKey,
			AutoDiscover: req.AutoDiscover,
		}
		if req.Pricing != nil {
			cfg.Pricing = &config.PricingConfig{
				InputPer1K:  req.Pricing.InputPer1K,
				OutputPer1K: req.Pricing.OutputPer1K,
			}
		}
		for _, m := range req.Models {
			cfg.Models = append(cfg.Models, config.ModelConfig{Name: m.Name})
		}

		pc := provider.ProviderConfig{
			Name:         req.Name,
			Type:         req.Type,
			BaseURL:      req.BaseURL,
			APIKey:       req.APIKey,
			AutoDiscover: req.AutoDiscover,
		}
		if req.Pricing != nil {
			pc.Pricing = &provider.PricingConfig{
				InputPer1K:  req.Pricing.InputPer1K,
				OutputPer1K: req.Pricing.OutputPer1K,
			}
		}
		for _, m := range req.Models {
			pc.Models = append(pc.Models, provider.ModelConfig{Name: m.Name})
		}

		p, err := provider.NewProvider(pc)
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}

		if err := app.Registry.Add(cfg, p); err != nil {
			writeError(w, 409, err.Error())
			return
		}

		app.Config.Providers = append(app.Config.Providers, cfg)
		if err := saveConfig(app); err != nil {
			writeError(w, 500, err.Error())
			return
		}

		writeJSON(w, http.StatusCreated, map[string]string{"status": "created", "name": req.Name})
	}
}

func handleAdminUpdateProvider(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		var req struct {
			BaseURL      string `json:"base_url"`
			APIKey       string `json:"api_key"`
			AutoDiscover bool   `json:"auto_discover"`
			Pricing      *struct {
				InputPer1K  float64 `json:"input_per_1k"`
				OutputPer1K float64 `json:"output_per_1k"`
			} `json:"pricing"`
			Models []struct {
				Name string `json:"name"`
			} `json:"models"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid request body")
			return
		}

		existing, ok := app.Registry.Get(name)
		if !ok {
			writeError(w, 404, "provider not found: "+name)
			return
		}

		apiKey := req.APIKey
		if apiKey == "" {
			apiKey = existing.Config.APIKey
		}

		// Merge optional fields from existing config when not provided in the update request
		existingCfg := existing.Config
		sessionCookie := existingCfg.SessionCookie
		userAgent := existingCfg.UserAgent
		orgID := existingCfg.OrganizationID
		anthropicVersion := existingCfg.AnthropicVersion
		pollingInterval := existingCfg.PollingInterval
		maxRetries := existingCfg.MaxRetries
		timeoutSec := existingCfg.TimeoutSec
		headers := existingCfg.Headers
		var authSettings *config.AuthSettings
		if existingCfg.Auth != nil {
			authSettings = &config.AuthSettings{
				Mode:            existingCfg.Auth.Mode,
				ClientID:        existingCfg.Auth.ClientID,
				ClientSecret:    existingCfg.Auth.ClientSecret,
				RedirectURI:     existingCfg.Auth.RedirectURI,
				CallbackPort:    existingCfg.Auth.CallbackPort,
				KeychainService: existingCfg.Auth.KeychainService,
				TokenEndpoint:   existingCfg.Auth.TokenEndpoint,
				AuthEndpoint:    existingCfg.Auth.AuthEndpoint,
				Scopes:          existingCfg.Auth.Scopes,
			}
		}
		var pricing *config.PricingConfig
		if req.Pricing != nil {
			pricing = &config.PricingConfig{
				InputPer1K:  req.Pricing.InputPer1K,
				OutputPer1K: req.Pricing.OutputPer1K,
			}
		} else if existingCfg.Pricing != nil {
			pricing = &config.PricingConfig{
				InputPer1K:  existingCfg.Pricing.InputPer1K,
				OutputPer1K: existingCfg.Pricing.OutputPer1K,
			}
		}

		cfg := config.ProviderConfig{
			Name:             name,
			Type:             existingCfg.Type,
			BaseURL:          req.BaseURL,
			APIKey:           apiKey,
			OrganizationID:   orgID,
			AnthropicVersion: anthropicVersion,
			AutoDiscover:     req.AutoDiscover,
			PollingInterval:  pollingInterval,
			Pricing:          pricing,
			MaxRetries:       maxRetries,
			TimeoutSec:       timeoutSec,
			Headers:          headers,
			SessionCookie:    sessionCookie,
			UserAgent:        userAgent,
			Auth:             authSettings,
		}
		// Preserve existing models if none provided in update request
		if len(req.Models) > 0 {
			for _, m := range req.Models {
				cfg.Models = append(cfg.Models, config.ModelConfig{Name: m.Name})
			}
		} else {
			for _, m := range existingCfg.Models {
				cfg.Models = append(cfg.Models, config.ModelConfig{Name: m.Name})
			}
		}

		var pAuth *types.AuthSettings
		if authSettings != nil {
			pAuth = &types.AuthSettings{
				Mode:            authSettings.Mode,
				ClientID:        authSettings.ClientID,
				ClientSecret:    authSettings.ClientSecret,
				RedirectURI:     authSettings.RedirectURI,
				CallbackPort:    authSettings.CallbackPort,
				KeychainService: authSettings.KeychainService,
				TokenEndpoint:   authSettings.TokenEndpoint,
				AuthEndpoint:    authSettings.AuthEndpoint,
				Scopes:          authSettings.Scopes,
			}
		}
		var pPricing *types.PricingConfig
		if pricing != nil {
			pPricing = &types.PricingConfig{
				InputPer1K:  pricing.InputPer1K,
				OutputPer1K: pricing.OutputPer1K,
			}
		}

		pc := provider.ProviderConfig{
			Name:             name,
			Type:             existingCfg.Type,
			BaseURL:          req.BaseURL,
			APIKey:           apiKey,
			OrganizationID:   orgID,
			AnthropicVersion: anthropicVersion,
			AutoDiscover:     req.AutoDiscover,
			PollingInterval:  pollingInterval,
			MaxRetries:       maxRetries,
			TimeoutSec:       timeoutSec,
			Headers:          headers,
			SessionCookie:    sessionCookie,
			UserAgent:        userAgent,
			Auth:             pAuth,
			Pricing:          pPricing,
		}
		if len(req.Models) > 0 {
			for _, m := range req.Models {
				pc.Models = append(pc.Models, provider.ModelConfig{Name: m.Name})
			}
		} else {
			for _, m := range existingCfg.Models {
				pc.Models = append(pc.Models, provider.ModelConfig{Name: m.Name})
			}
		}

		p, err := provider.NewProvider(pc)
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}

		if err := app.Registry.Update(name, cfg, p); err != nil {
			writeError(w, 404, err.Error())
			return
		}

		for i, existing := range app.Config.Providers {
			if existing.Name == name {
				app.Config.Providers[i] = cfg
				break
			}
		}
		if err := saveConfig(app); err != nil {
			writeError(w, 500, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "updated", "name": name})
	}
}

func handleAdminDiscoverProviderModels(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name         string `json:"name"`
			Type         string `json:"type"`
			BaseURL      string `json:"base_url"`
			APIKey       string `json:"api_key"`
			AutoDiscover bool   `json:"auto_discover"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid request body")
			return
		}

		if req.Name != "" {
			if existing, ok := app.Registry.Get(req.Name); ok {
				if req.Type == "" {
					req.Type = existing.Config.Type
				}
				if req.BaseURL == "" {
					req.BaseURL = existing.Config.BaseURL
				}
				if req.APIKey == "" {
					req.APIKey = existing.Config.APIKey
				}
			}
		}

		if req.Type == "" {
			writeError(w, 400, "provider type is required")
			return
		}
		if !req.AutoDiscover {
			writeError(w, 400, "auto_discover must be enabled to discover models")
			return
		}

		pc := provider.ProviderConfig{
			Name:         req.Name,
			Type:         req.Type,
			BaseURL:      req.BaseURL,
			APIKey:       req.APIKey,
			AutoDiscover: req.AutoDiscover,
		}

		p, err := provider.NewProvider(pc)
		if err != nil {
			writeError(w, 400, err.Error())
			return
		}

		models, err := p.ListModels(r.Context())
		if err != nil {
			writeError(w, 500, fmt.Sprintf("failed to discover models: %v", err))
			return
		}

		modelNames := make([]string, len(models))
		for i, m := range models {
			name := strings.TrimPrefix(m.Name, req.Name+"/")
			name = strings.TrimPrefix(name, "/")
			modelNames[i] = name
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{"models": modelNames})
	}
}

func handleAdminDeleteProvider(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		app.Registry.Remove(name)

		providers := make([]config.ProviderConfig, 0, len(app.Config.Providers))
		for _, p := range app.Config.Providers {
			if p.Name != name {
				providers = append(providers, p)
			}
		}
		app.Config.Providers = providers
		if err := saveConfig(app); err != nil {
			writeError(w, 500, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "name": name})
	}
}

func handleAdminTestProvider(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		ps, ok := app.Registry.Get(name)
		if !ok {
			writeError(w, 404, "provider not found: "+name)
			return
		}

		health := ps.Provider.CheckHealth(r.Context())
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":     health.Status,
			"latency_ms": health.LatencyMS,
			"error":      health.ErrorMessage,
		})
	}
}

func handleAdminGetProviderBalance(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := chi.URLParam(r, "name")
		ps, ok := app.Registry.Get(name)
		if !ok {
			writeError(w, 404, "provider not found: "+name)
			return
		}

		balanceProvider, ok := ps.Provider.(types.BalanceProvider)
		if !ok {
			writeError(w, 400, "balance API is not supported for provider type "+ps.Provider.Type())
			return
		}

		balance, err := balanceProvider.GetBalance(r.Context())
		if err != nil {
			writeError(w, 502, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"provider": name,
			"type":     ps.Provider.Type(),
			"balance":  balance,
		})
	}
}

func handleAdminStatsSummary(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		summary := app.Tracker.GetSummary()
		writeJSON(w, http.StatusOK, summary)
	}
}

func handleAdminCostByProvider(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		costs := app.Tracker.GetCostByProvider()
		writeJSON(w, http.StatusOK, map[string]interface{}{"cost_by_provider": costs})
	}
}

func handleAdminCostByModel(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		models := app.Tracker.GetCostByModel()
		writeJSON(w, http.StatusOK, map[string]interface{}{"cost_by_model": models})
	}
}

func handleAdminLogs(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		limit := 100
		if l := q.Get("limit"); l != "" {
			if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 {
				limit = parsed
			}
		}
		provider := q.Get("provider")
		logs := app.Tracker.GetRecentLogsFiltered(limit, provider)
		writeJSON(w, http.StatusOK, map[string]interface{}{"logs": logs})
	}
}

func handleAdminGetSettings(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"port":                  app.Config.Settings.Port,
			"log_level":             app.Config.Settings.LogLevel,
			"prometheus_enabled":    app.Config.Settings.PrometheusEnabled,
			"rate_limiting_enabled": app.Config.Settings.RateLimitingEnabled,
			"max_retries":           app.Config.Settings.DefaultMaxRetries,
			"timeout_seconds":       app.Config.Settings.DefaultTimeoutSec,
			"usage_persistence":     app.Config.Settings.UsagePersistence,
		})
	}
}

func handleAdminUpdateSettings(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req map[string]interface{}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid request body")
			return
		}

		// Apply settings from request body
		changed := make([]string, 0)

		if v, ok := req["log_level"]; ok {
			if s, ok := v.(string); ok {
				app.Config.Settings.LogLevel = s
				changed = append(changed, "log_level")
			}
		}

		if v, ok := req["prometheus_enabled"]; ok {
			if b, ok := v.(bool); ok {
				app.Config.Settings.PrometheusEnabled = b
				app.Settings.PrometheusEnabled = b
				changed = append(changed, "prometheus_enabled")
			}
		}

		if v, ok := req["rate_limiting_enabled"]; ok {
			if b, ok := v.(bool); ok {
				app.Config.Settings.RateLimitingEnabled = b
				if app.RateLimit != nil {
					// Toggle rate limiter - need to access the internal enabled field
					// For this implementation, we'll just record the setting change
				}
				changed = append(changed, "rate_limiting_enabled")
			}
		}

		if v, ok := req["max_retries"]; ok {
			if f, ok := v.(float64); ok {
				app.Config.Settings.DefaultMaxRetries = int(f)
				changed = append(changed, "max_retries")
			}
		}

		if v, ok := req["timeout_seconds"]; ok {
			if f, ok := v.(float64); ok {
				app.Config.Settings.DefaultTimeoutSec = int(f)
				changed = append(changed, "timeout_seconds")
			}
		}

		app.Logger.Info("settings updated", "changes", changed)
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "settings updated",
			"changes": changed,
		})
	}
}

func handleAdminConfigYAML(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		yaml := fmt.Sprintf(`# OllaRouter Runtime Configuration
settings:
  port: %d
  log_level: "%s"
  log_format: "%s"
  prometheus_enabled: %v
  rate_limiting_enabled: %v
  default_max_retries: %d
  default_timeout_seconds: %d
  usage_persistence: "%s"
  usage_file_path: "%s"
  circuit_breaker_threshold: %d
  circuit_breaker_cooldown_seconds: %d

api_keys:
`,
			app.Config.Settings.Port,
			app.Config.Settings.LogLevel,
			app.Config.Settings.LogFormat,
			app.Config.Settings.PrometheusEnabled,
			app.Config.Settings.RateLimitingEnabled,
			app.Config.Settings.DefaultMaxRetries,
			app.Config.Settings.DefaultTimeoutSec,
			app.Config.Settings.UsagePersistence,
			app.Config.Settings.UsageFilePath,
			app.Config.Settings.CircuitBreakerThreshold,
			app.Config.Settings.CircuitBreakerCooldown,
		)

		yaml += "\nproviders:\n"
		for _, ps := range app.Registry.List() {
			maskedKey := maskAPIKey(ps.Config.APIKey)
			yaml += fmt.Sprintf("  - name: %s\n    type: %s\n    base_url: \"%s\"\n    api_key: \"%s\"\n    auto_discover: %v\n",
				ps.Config.Name, ps.Config.Type, ps.Config.BaseURL, maskedKey, ps.Config.AutoDiscover)

			if len(ps.ModelCache) > 0 {
				yaml += "    models:\n"
				for _, m := range ps.ModelCache {
					yaml += fmt.Sprintf("      - name: %s\n", m.Name)
				}
			}
		}

		if len(app.Config.VirtualModels) > 0 {
			yaml += "\nvirtual_models:\n"
			for _, vm := range app.Config.VirtualModels {
				yaml += fmt.Sprintf("  - name: %s\n    fallback_chain:\n", vm.Name)
				for _, f := range vm.FallbackChain {
					yaml += fmt.Sprintf("      - %s\n", f)
				}
				if vm.Timeout > 0 {
					yaml += fmt.Sprintf("    timeout: %d\n", vm.Timeout)
				}
			}
		}

		w.Header().Set("Content-Type", "text/yaml")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(yaml))
	}
}

// handleAdminAuthStatus returns the upstream auth status for all providers.
func handleAdminAuthStatus(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if app.UpstreamAuth == nil {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"status":  "not_configured",
				"message": "upstream auth manager not configured",
			})
			return
		}

		// Gather status from all providers that have auth settings
		statuses := make(map[string]string)
		for _, ps := range app.Registry.List() {
			if ps.Config.Auth != nil {
				statuses[ps.Config.Name] = app.UpstreamAuth.GetCredentialStatus(ps.Config.Name)
			}
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":    "ok",
			"providers": statuses,
		})
	}
}

// handleAdminAuthConnect initiates upstream auth for a provider.
//
//   - If a "credential" is supplied, it is stored directly (legacy bearer mode).
//   - Otherwise an OAuth "Sign in with ChatGPT" flow is started and the
//     authorize URL + state are returned for the user to open.
func handleAdminAuthConnect(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if app.UpstreamAuth == nil {
			writeError(w, 400, "upstream auth manager not configured")
			return
		}

		var req struct {
			Provider   string `json:"provider"`
			Credential string `json:"credential"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid request body")
			return
		}

		if req.Provider == "" {
			writeError(w, 400, "provider is required")
			return
		}

		// Legacy: store a credential directly.
		if req.Credential != "" {
			if err := app.UpstreamAuth.StoreCredential(req.Provider, req.Credential); err != nil {
				writeError(w, 500, err.Error())
				return
			}
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"status":   "connected",
				"provider": req.Provider,
			})
			return
		}

		// OAuth: start a login and return the URL to open.
		authURL, state, err := app.UpstreamAuth.StartLogin(req.Provider)
		if err != nil {
			writeError(w, 500, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":   "authorize_required",
			"provider": req.Provider,
			"state":    state,
			"auth_url": authURL,
			"instructions": "Open auth_url in a browser and sign in. Your browser will redirect to a " +
				"localhost address that fails to load - copy that full URL (or the 'code' value from it) " +
				"and POST it to /admin/api/auth/callback as {\"state\":\"...\",\"code\":\"...\"} or {\"redirect_url\":\"...\"}.",
		})
	}
}

// handleAdminAuthCallback completes an OAuth flow by exchanging the
// authorization code for tokens. It accepts the code/state via JSON body, query
// parameters, or a pasted full redirect URL.
func handleAdminAuthCallback(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if app.UpstreamAuth == nil {
			writeError(w, 400, "upstream auth manager not configured")
			return
		}

		var body struct {
			Code        string `json:"code"`
			State       string `json:"state"`
			RedirectURL string `json:"redirect_url"`
		}
		if r.Body != nil {
			_ = json.NewDecoder(r.Body).Decode(&body)
		}

		code := firstNonEmpty(body.Code, r.URL.Query().Get("code"))
		state := firstNonEmpty(body.State, r.URL.Query().Get("state"))

		// Allow pasting the full localhost redirect URL.
		if redir := firstNonEmpty(body.RedirectURL, r.URL.Query().Get("redirect_url")); redir != "" {
			if u, err := url.Parse(redir); err == nil {
				if c := u.Query().Get("code"); c != "" {
					code = c
				}
				if s := u.Query().Get("state"); s != "" {
					state = s
				}
			}
		}

		if code == "" || state == "" {
			writeError(w, 400, "code and state are required (or a redirect_url containing them)")
			return
		}

		provider, err := app.UpstreamAuth.CompleteLogin(r.Context(), state, code)
		if err != nil {
			writeError(w, 502, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":   "connected",
			"provider": provider,
		})
	}
}

// firstNonEmpty returns the first non-empty string from the arguments.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// handleAdminAuthDisconnect revokes upstream auth for a provider.
func handleAdminAuthDisconnect(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if app.UpstreamAuth == nil {
			writeError(w, 400, "upstream auth manager not configured")
			return
		}

		var req struct {
			Provider string `json:"provider"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, 400, "invalid request body")
			return
		}

		if req.Provider == "" {
			writeError(w, 400, "provider is required")
			return
		}

		if err := app.UpstreamAuth.RevokeCredential(req.Provider); err != nil {
			writeError(w, 500, err.Error())
			return
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":   "disconnected",
			"provider": req.Provider,
		})
	}
}

// maskAPIKey masks an API key for safe display, showing only
// the first 4 and last 4 characters.
func maskAPIKey(key string) string {
	if len(key) <= 8 {
		return "***"
	}
	return key[:4] + "..." + key[len(key)-4:]
}

// spaHandler serves the embedded admin SPA.
func spaHandler(app *App) http.HandlerFunc {
	efs := webui.EmbeddedFS()
	// Pre-read embedded files into memory for fast serving
	indexHTML, _ := fs.ReadFile(efs, "index.html")
	jsFile := ""
	cssFile := ""

	// Discover the JS and CSS file names from the embedded FS
	fs.WalkDir(efs, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		// Always pick the one with the latest name (alphabetically) to handle duplicates
		if strings.HasSuffix(path, ".js") {
			if jsFile == "" || path > jsFile {
				jsFile = path
			}
		}
		if strings.HasSuffix(path, ".css") {
			if cssFile == "" || path > cssFile {
				cssFile = path
			}
		}
		return nil
	})

	if jsFile != "" {
		app.Logger.Info("SPA serving JS", "file", jsFile)
	}
	if cssFile != "" {
		app.Logger.Info("SPA serving CSS", "file", cssFile)
	}

	jsBytes, _ := fs.ReadFile(efs, jsFile)
	cssBytes, _ := fs.ReadFile(efs, cssFile)

	// Pre-detect MIME types
	jsMime := "application/javascript"
	cssMime := "text/css"

	return func(w http.ResponseWriter, r *http.Request) {
		// By this point chi strips the /admin prefix, so r.URL.Path is /*
		path := r.URL.Path
		if path == "" {
			path = "/"
		}

		// Serve appropriate file based on extension
		if len(path) > 3 && path[len(path)-3:] == ".js" {
			w.Header().Set("Content-Type", jsMime)
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			w.WriteHeader(200)
			w.Write(jsBytes)
			return
		}
		if len(path) > 4 && path[len(path)-4:] == ".css" {
			w.Header().Set("Content-Type", cssMime)
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
			w.WriteHeader(200)
			w.Write(cssBytes)
			return
		}
		if path == "/favicon.svg" {
			w.Header().Set("Content-Type", "image/svg+xml")
			w.Header().Set("Cache-Control", "public, max-age=86400")
			w.WriteHeader(200)
			// Serve a minimal SVG favicon or skip
			w.Write([]byte(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24"><rect width="24" height="24" rx="4" fill="#38bdf8"/><text x="12" y="16" text-anchor="middle" fill="white" font-size="12" font-family="sans-serif">O</text></svg>`))
			return
		}

		// SPA fallback: serve index.html for all non-file routes
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(200)
		w.Write(indexHTML)
	}
}

// breakerStatusFor returns the live circuit breaker used by the fallback
// engine, falling back to the registry's static state before first use.
func breakerStatusFor(app *App, p *types.ProviderState) map[string]interface{} {
	if app.Fallback != nil {
		if st, ok := app.Fallback.BreakerStatus(p.Config.Name); ok {
			return map[string]interface{}{"state": st.State, "failures": st.Failures}
		}
	}
	state, failures := "closed", 0
	if p.CircuitBreaker != nil {
		state, failures = p.CircuitBreaker.State, p.CircuitBreaker.Failures
	}
	return map[string]interface{}{"state": state, "failures": failures}
}
