package router

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/aideveloper1976-arch/ollarouter/internal/types"
)

var startTime = time.Now()

func handleHealth(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providers := app.Registry.List()
		total := len(providers)
		online := 0
		degraded := 0
		offline := 0

		ctx := r.Context()
		healthCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		type healthResult struct {
			name   string
			health types.ProviderHealth
		}
		results := make(chan healthResult, len(providers))
		var wg sync.WaitGroup

		for _, ps := range providers {
			wg.Add(1)
			ps := ps
			go func() {
				defer wg.Done()
				// Per-provider timeout context
				pCtx, pCancel := context.WithTimeout(healthCtx, 15*time.Second)
				defer pCancel()
				h := ps.Provider.CheckHealth(pCtx)
				results <- healthResult{name: ps.Config.Name, health: h}
			}()
		}

		// Wait for all checks to complete
		wg.Wait()
		close(results)

		for res := range results {
			app.Registry.UpdateHealth(res.name, res.health)
			switch res.health.Status {
			case "online":
				online++
			case "degraded":
				degraded++
			default:
				offline++
			}
		}

		status := "ok"
		httpStatus := http.StatusOK
		if total == 0 || online == 0 {
			status = "degraded"
			httpStatus = http.StatusServiceUnavailable
		}

		writeJSON(w, httpStatus, map[string]interface{}{
			"status":             status,
			"version":            FallbackVersion,
			"uptime_seconds":     int(time.Since(startTime).Seconds()),
			"providers_total":    total,
			"providers_online":   online,
			"providers_degraded": degraded,
			"providers_offline":  offline,
		})
	}
}

func handleHealthProviders(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		providers := app.Registry.List()
		result := make([]map[string]interface{}, 0, len(providers))

		ctx := r.Context()
		healthCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
		defer cancel()

		type namedHealth struct {
			name   string
			health types.ProviderHealth
			cb     *types.CircuitBreakerStatus
			pType  string
			models int
		}

		results := make(chan namedHealth, len(providers))
		var wg sync.WaitGroup

		for _, ps := range providers {
			wg.Add(1)
			ps := ps
			go func() {
				defer wg.Done()
				pCtx, pCancel := context.WithTimeout(healthCtx, 15*time.Second)
				defer pCancel()
				h := ps.Provider.CheckHealth(pCtx)
				results <- namedHealth{
					name:   ps.Config.Name,
					health: h,
					cb:     ps.CircuitBreaker,
					pType:  ps.Config.Type,
					models: len(ps.ModelCache),
				}
			}()
		}

		wg.Wait()
		close(results)

		for res := range results {
			app.Registry.UpdateHealth(res.name, res.health)

			info := map[string]interface{}{
				"name":         res.name,
				"type":         res.pType,
				"status":       res.health.Status,
				"latency_ms":   res.health.LatencyMS,
				"last_check":   res.health.LastCheck.Format(time.RFC3339),
				"models_count": res.models,
			}

			if res.health.ErrorMessage != "" {
				info["error"] = res.health.ErrorMessage
			}

			if res.cb != nil {
				info["circuit_breaker"] = map[string]interface{}{
					"state":    res.cb.State,
					"failures": res.cb.Failures,
				}
			}

			result = append(result, info)
		}

		writeJSON(w, http.StatusOK, map[string]interface{}{
			"providers": result,
		})
	}
}
