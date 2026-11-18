package router

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/aideveloper1976-arch/ollarouter/internal/auth"
	"github.com/aideveloper1976-arch/ollarouter/internal/auth/upstream"
	"github.com/aideveloper1976-arch/ollarouter/internal/config"
	"github.com/aideveloper1976-arch/ollarouter/internal/engine"
	"github.com/aideveloper1976-arch/ollarouter/internal/logger"
	"github.com/aideveloper1976-arch/ollarouter/internal/provider"
	"github.com/aideveloper1976-arch/ollarouter/internal/tracking"
	"github.com/aideveloper1976-arch/ollarouter/internal/types"
	"github.com/go-chi/chi/v5"
	chimw "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
)

// App holds all application dependencies.
type App struct {
	Config       *config.Config
	ConfigPath   string
	Logger       *logger.Logger
	Registry     *provider.Registry
	Engine       *engine.ModelResolver
	Tracker      *tracking.Tracker
	Fallback     *engine.FallbackExecutor
	Settings     RouterSettings
	Auth         *auth.Authenticator
	RateLimit    *auth.RateLimiter
	UpstreamAuth *upstream.UpstreamAuthManager
}

// recordUsage helper to centralize tracking and cost calculation.
func (app *App) recordUsage(providerName, modelName string, status int, usage *types.TokenUsage, latencyMs int64) {
	if app.Tracker == nil {
		return
	}

	cost := 0.0
	if usage != nil {
		if ps, ok := app.Registry.Get(providerName); ok && ps.Config.Pricing != nil {
			cost = (float64(usage.PromptTokens) / 1000.0 * ps.Config.Pricing.InputPer1K) +
				(float64(usage.CompletionTokens) / 1000.0 * ps.Config.Pricing.OutputPer1K)
		}
	}

	record := tracking.UsageRecord{
		Timestamp: time.Now(),
		Provider:  providerName,
		Model:     modelName,
		Status:    status,
		LatencyMs: latencyMs,
		CostUSD:   cost,
	}

	if usage != nil {
		record.PromptTokens = usage.PromptTokens
		record.CompletionTokens = usage.CompletionTokens
		record.TotalTokens = usage.TotalTokens
	}

	app.Logger.Info("usage recorded",
		"provider", providerName,
		"model", modelName,
		"cost", cost,
		"tokens", record.TotalTokens,
	)

	app.Tracker.Record(record)
}

// RouterSettings holds router configuration.
type RouterSettings struct {
	PrometheusEnabled bool
	AdminUIPath       string
	// OpenAIEndpointPublic skips API-key auth on the OpenAI-compatible
	// chat endpoint. Off by default.
	OpenAIEndpointPublic bool
}

// NewRouter creates the chi router with all middleware and routes.
func NewRouter(app *App) http.Handler {
	r := chi.NewRouter()

	// Global middleware
	r.Use(chimw.RequestID)
	r.Use(chimw.RealIP)
	r.Use(requestLoggerMiddleware(app.Logger))
	r.Use(chimw.Recoverer)

	// Root redirect - /
	r.Get("/", handleRootRedirect)
	r.Get("/index.html", handleRootRedirect)

	// All /api routes in one sub-router (avoids chi conflicts between root-level routes and sub-router mounts)
	r.Route("/api", func(r chi.Router) {
		// Group 1: Public routes - no auth/rate-limiting required (model discovery for tools like Open WebUI)
		r.Group(func(r chi.Router) {
			r.Get("/tags", handleTags(app))
			r.Get("/tags/", handleTags(app))
			r.Get("/models", handleTags(app))
			r.Get("/models/", handleTags(app))
			r.Get("/ps", handlePS(app))
			r.Get("/ps/", handlePS(app))
			r.Post("/show", handleShow(app))
			r.Get("/show", handleShow(app))
			r.Post("/show/", handleShow(app))
			r.Get("/show/", handleShow(app))
			r.Get("/version", handleVersion(app))
		})

		// Group 2: Protected routes with auth + rate limiting (Ollama-compatible)
		r.Group(func(r chi.Router) {
			if app.Auth != nil {
				r.Use(app.Auth.Middleware)
			}
			if app.RateLimit != nil {
				r.Use(app.RateLimit.Middleware)
			}
			r.Post("/chat", handleChat(app))
			r.Post("/generate", handleGenerate(app))
		})
	})

	// OpenAI-compatible chat endpoint. It spends upstream provider credit, so
	// it gets the same API-key auth and rate limiting as /api/chat unless the
	// operator sets settings.openai_endpoint_public for a client that cannot
	// send a key.
	r.Group(func(r chi.Router) {
		if !app.Settings.OpenAIEndpointPublic {
			if app.Auth != nil {
				r.Use(app.Auth.Middleware)
			}
			if app.RateLimit != nil {
				r.Use(app.RateLimit.Middleware)
			}
		}
		r.Post("/v1/chat/completions", handleV1ChatCompletions(app))
		// Some VS Code versions call /chat/completions without the /v1 prefix.
		r.Post("/chat/completions", handleV1ChatCompletions(app))
	})

	// Health routes (always public)
	r.Get("/health", handleHealth(app))
	r.Get("/health/providers", handleHealthProviders(app))

	// Metrics (optional, public)
	if app.Settings.PrometheusEnabled {
		r.Get("/metrics", handleMetrics(app))
	}

	// Admin API + UI - no auth required for admin
	r.Route("/admin", func(r chi.Router) {
		// CORS only for admin (not for Ollama API routes, which must match Ollama's exact header format)
		r.Use(cors.Handler(cors.Options{
			AllowedOrigins:   []string{"*"},
			AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
			AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
			ExposedHeaders:   []string{"Link"},
			AllowCredentials: false,
			MaxAge:           300,
		}))
		// Admin API routes
		r.Route("/api", func(r chi.Router) {
			// /me must be unauthenticated - the SPA uses it to detect auth state
			r.Get("/me", handleAdminMe(app))

			// All other admin API routes require auth when API keys are configured
			r.Group(func(r chi.Router) {
				if app.Auth != nil {
					r.Use(app.Auth.Middleware)
				}
				r.Get("/providers", handleAdminListProviders(app))
				r.Post("/providers", handleAdminAddProvider(app))
				r.Put("/providers/{name}", handleAdminUpdateProvider(app))
				r.Delete("/providers/{name}", handleAdminDeleteProvider(app))
				r.Get("/providers/{name}/test", handleAdminTestProvider(app))
				r.Get("/providers/{name}/balance", handleAdminGetProviderBalance(app))
				r.Post("/providers/discover", handleAdminDiscoverProviderModels(app))
				r.Get("/stats/summary", handleAdminStatsSummary(app))
				r.Get("/stats/cost-by-provider", handleAdminCostByProvider(app))
				r.Get("/stats/cost-by-model", handleAdminCostByModel(app))

				r.Get("/logs", handleAdminLogs(app))

				r.Get("/settings", handleAdminGetSettings(app))
				r.Put("/settings", handleAdminUpdateSettings(app))
				r.Get("/config/yaml", handleAdminConfigYAML(app))

				// Upstream auth lifecycle endpoints
				r.Get("/auth/status", handleAdminAuthStatus(app))
				r.Post("/auth/connect", handleAdminAuthConnect(app))
				r.Get("/auth/callback", handleAdminAuthCallback(app))
				r.Post("/auth/callback", handleAdminAuthCallback(app))
				r.Post("/auth/disconnect", handleAdminAuthDisconnect(app))
			})
		})

		// Admin SPA - serve embedded UI
		// DESIGN DECISION: The SPA is served without auth middleware because the login
		// overlay must load from the SPA itself. Once the user enters their API key via
		// the login overlay, subsequent /admin/api/* calls include the Authorization header.
		// All /admin/api/* routes above are protected by auth middleware.
		// The SPA uses GET /admin/api/me to verify auth state on load.
		adminSpaHandler := spaHandler(app)
		r.Get("/*", adminSpaHandler)
	})

	return r
}

// requestLoggerMiddleware creates a chi-compatible request logger.
func requestLoggerMiddleware(log *logger.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ww := chimw.NewWrapResponseWriter(w, r.ProtoMajor)
			log.InfoContext(r.Context(), "request",
				"method", r.Method,
				"path", r.URL.Path,
				"remote", r.RemoteAddr,
			)
			next.ServeHTTP(ww, r)
			log.InfoContext(r.Context(), "response",
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
			)
		})
	}
}

// handleRootRedirect redirects / and /index.html to /admin/.
func handleRootRedirect(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/admin/", http.StatusFound)
}

// writeError writes an Ollama-compatible JSON error response.
func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	w.Write([]byte(`{"error":"` + escapeJSON(message) + `"}`))
}

// writeJSON writes a JSON response.
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// escapeJSON escapes a string for safe JSON inclusion.
func escapeJSON(s string) string {
	result := make([]byte, 0, len(s))
	for _, c := range []byte(s) {
		switch c {
		case '"':
			result = append(result, '\\', '"')
		case '\\':
			result = append(result, '\\', '\\')
		case '\n':
			result = append(result, '\\', 'n')
		case '\r':
			result = append(result, '\\', 'r')
		case '\t':
			result = append(result, '\\', 't')
		default:
			if c < 0x20 {
				result = append(result, '\\', 'u', '0', '0', hexChar(c>>4), hexChar(c&0xF))
			} else {
				result = append(result, c)
			}
		}
	}
	return string(result)
}

func hexChar(c byte) byte {
	if c < 10 {
		return '0' + c
	}
	return 'a' + c - 10
}
