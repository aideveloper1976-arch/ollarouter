package main

import (
	"context"
	"flag"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/aideveloper1976-arch/ollarouter/internal/auth"
	"github.com/aideveloper1976-arch/ollarouter/internal/auth/upstream"
	"github.com/aideveloper1976-arch/ollarouter/internal/config"
	"github.com/aideveloper1976-arch/ollarouter/internal/engine"
	"github.com/aideveloper1976-arch/ollarouter/internal/logger"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider/chatgpt"
	"github.com/aideveloper1976-arch/ollarouter/internal/router"
	"github.com/aideveloper1976-arch/ollarouter/internal/tracking"
)

func main() {
	healthcheckFlag := flag.Bool("healthcheck", false, "Run health check subcommand and exit")
	configPath := flag.String("config", "", "Path to providers.yaml (env: OLLAROUTER_CONFIG_PATH)")
	flag.Parse()

	if *healthcheckFlag {
		os.Exit(healthcheckCmd())
	}

	// Determine config file path
	cfgPath := *configPath
	if cfgPath == "" {
		cfgPath = os.Getenv("OLLAROUTER_CONFIG_PATH")
	}
	if cfgPath == "" {
		cfgPath = "providers.yaml"
	}

	// Load configuration
	cfg, err := config.ParseFile(cfgPath)
	if err != nil {
		// Try alternative paths
		altPaths := []string{
			"/app/config/providers.yaml",
			"/app/providers.yaml.example",
			"./providers.yaml",
			"./providers.yaml.example",
		}
		for _, p := range altPaths {
			if cfg2, err2 := config.ParseFile(p); err2 == nil {
				cfg = cfg2
				cfgPath = p
				err = nil
				break
			}
		}
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failed to load config: %v\n", err)
			os.Exit(1)
		}
	}

	// Initialize logger
	log := logger.New(cfg.Settings.LogLevel, cfg.Settings.LogFormat)
	log.Info("starting OllaRouter", "version", router.FallbackVersion, "config", cfgPath, "usage_persistence", cfg.Settings.UsagePersistence)

	// Initialize upstream auth manager (for providers with auth settings, e.g.
	// ChatGPT Plus "Sign in with ChatGPT" OAuth). Created before the registry so
	// the credential resolver can be wired into provider adapters.
	upstreamAuthManager, err := upstream.InitFromProviders(cfg.Providers, cfg.Settings.UpstreamAuthCredentialPath, cfg.Settings.UpstreamAuthEncryptionKey)
	if err != nil {
		log.Warn("failed to initialize upstream auth manager", "error", err)
	}
	if upstreamAuthManager != nil {
		authCount := 0
		for _, p := range cfg.Providers {
			if p.Auth != nil {
				authCount++
				// Apply per-provider OAuth overrides (endpoints/client id/etc).
				oc := upstream.DefaultCodexOAuthConfig()
				if p.Auth.ClientID != "" {
					oc.ClientID = p.Auth.ClientID
				}
				if p.Auth.AuthEndpoint != "" {
					oc.AuthEndpoint = p.Auth.AuthEndpoint
				}
				if p.Auth.TokenEndpoint != "" {
					oc.TokenEndpoint = p.Auth.TokenEndpoint
				}
				if p.Auth.RedirectURI != "" {
					oc.RedirectURI = p.Auth.RedirectURI
				}
				if p.Auth.Scopes != "" {
					oc.Scopes = p.Auth.Scopes
				}
				upstreamAuthManager.SetOAuthConfig(oc)
			}
		}
		log.Info("upstream auth manager initialized", "credential_path", cfg.Settings.UpstreamAuthCredentialPath, "providers_with_auth", authCount)
	}

	// Resolver passed to adapters that support upstream auth. Keep it nil (not a
	// typed-nil interface) when no manager exists, so adapters skip it cleanly.
	var credentialResolver chatgpt.CredentialResolver
	if upstreamAuthManager != nil {
		credentialResolver = upstreamAuthManager
	}

	// Initialize provider registry
	reg := provider.NewRegistry()

	for _, pc := range cfg.Providers {
		pCfg := provider.ProviderConfig{
			Name:             pc.Name,
			Type:             pc.Type,
			BaseURL:          pc.BaseURL,
			APIKey:           pc.APIKey,
			OrganizationID:   pc.OrganizationID,
			AnthropicVersion: pc.AnthropicVersion,
			AutoDiscover:     pc.AutoDiscover,
			PollingInterval:  pc.PollingInterval,
			MaxRetries:       pc.MaxRetries,
			TimeoutSec:       pc.TimeoutSec,
			SessionCookie:    pc.SessionCookie,
			UserAgent:        pc.UserAgent,
			Headers:          pc.Headers,
		}

		if pc.Auth != nil {
			pCfg.Auth = &provider.AuthSettings{
				Mode:          pc.Auth.Mode,
				ClientID:      pc.Auth.ClientID,
				ClientSecret:  pc.Auth.ClientSecret,
				RedirectURI:   pc.Auth.RedirectURI,
				CallbackPort:  pc.Auth.CallbackPort,
				TokenEndpoint: pc.Auth.TokenEndpoint,
				AuthEndpoint:  pc.Auth.AuthEndpoint,
				Scopes:        pc.Auth.Scopes,
			}
		}

		for _, m := range pc.Models {
			pCfg.Models = append(pCfg.Models, provider.ModelConfig{
				Name:       m.Name,
				MaxTokens:  m.MaxTokens,
				ContextLen: m.ContextLen,
			})
		}

		if pc.Pricing != nil {
			pCfg.Pricing = &provider.PricingConfig{
				InputPer1K:  pc.Pricing.InputPer1K,
				OutputPer1K: pc.Pricing.OutputPer1K,
			}
		}

		p, err := provider.NewProviderWithResolver(pCfg, credentialResolver)
		if err != nil {
			log.Warn("failed to initialize provider", "name", pc.Name, "error", err)
			continue
		}

		if err := reg.Add(pc, p); err != nil {
			log.Warn("failed to add provider", "name", pc.Name, "error", err)
			continue
		}

		log.Info("provider initialized", "name", pc.Name, "type", pc.Type)
	}

	log.Info("providers loaded", "count", len(reg.List()))

	// Refresh model cache for auto-discover providers (e.g. local-ollama)
	reg.RefreshModelCache()
	log.Info("model cache refreshed")

	// Initialize tracking
	tracker := tracking.NewTracker()

	if cfg.Settings.UsagePersistence == "file" {
		store := &tracking.UsageStore{Tracker: tracker}
		persistence := tracking.NewFilePersistence(store, cfg.Settings.UsageFilePath, 1*time.Minute)
		if err := persistence.Load(); err != nil {
			log.Warn("failed to load usage data", "error", err)
		}
		persistence.Start()
		defer persistence.Stop()
		log.Info("usage persistence enabled", "path", cfg.Settings.UsageFilePath)
	}

	// Initialize authenticator
	authn := auth.NewAuthenticator(cfg)
	if authn != nil {
		log.Info("authenticator initialized")
	}

	var rateLimiter *auth.RateLimiter

	if cfg.Settings.RateLimitingEnabled {
		rateLimiter = auth.NewRateLimiter(true)
		log.Info("rate limiting enabled")
	}

	// Build virtual models map
	virtualModels := make(map[string][]string)
	for _, vm := range cfg.VirtualModels {
		virtualModels[vm.Name] = vm.FallbackChain
	}
	if len(virtualModels) > 0 {
		log.Info("virtual models loaded", "count", len(virtualModels))
	}

	// Initialize engine
	modelResolver := engine.NewModelResolver(reg, virtualModels)
	retryCfg := engine.DefaultRetryConfig()
	// 0 is a valid value: fail over to the next provider without retrying.
	if cfg.Settings.DefaultMaxRetries >= 0 {
		retryCfg.MaxRetries = cfg.Settings.DefaultMaxRetries
	}
	fallbackExecutor := engine.NewFallbackExecutor(reg, retryCfg, virtualModels, log)
	fallbackExecutor.SetCircuitBreakerConfig(
		cfg.Settings.CircuitBreakerThreshold,
		time.Duration(cfg.Settings.CircuitBreakerCooldown)*time.Second,
	)

	// Create router
	app := &router.App{
		Logger:       log,
		Registry:     reg,
		Engine:       modelResolver,
		Auth:         authn,
		RateLimit:    rateLimiter,
		Tracker:      tracker,
		Fallback:     fallbackExecutor,
		UpstreamAuth: upstreamAuthManager,
		Settings: router.RouterSettings{
			PrometheusEnabled:    cfg.Settings.PrometheusEnabled,
			OpenAIEndpointPublic: cfg.Settings.OpenAIEndpointPublic,
		},
		Config:     cfg,
		ConfigPath: cfgPath,
	}

	handler := router.NewRouter(app)

	// Determine port. OLLAROUTER_PORT overrides the config file so the
	// container port, the -healthcheck probe and the server always agree.
	port := cfg.Settings.Port
	if envPort := os.Getenv("OLLAROUTER_PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 && p <= 65535 {
			port = p
		} else {
			log.Warn("ignoring invalid OLLAROUTER_PORT", "value", envPort)
		}
	}
	if port <= 0 {
		port = 11435
	}

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 300 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server
	go func() {
		log.Info("server listening", "port", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("server error", "error", err)
			os.Exit(1)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down gracefully...")
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Error("forced shutdown", "error", err)
	}
}

// healthcheckCmd runs a health check against the running server.
// Exit code 0 = healthy, 1 = unhealthy.
func healthcheckCmd() int {
	port := os.Getenv("OLLAROUTER_PORT")
	if port == "" {
		port = "11435"
	}

	resp, err := http.Get(fmt.Sprintf("http://localhost:%s/health", port))
	if err != nil {
		fmt.Fprintf(os.Stderr, "Health check failed: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		fmt.Fprintf(os.Stderr, "Health check returned HTTP %d\n", resp.StatusCode)
		return 1
	}

	return 0
}
