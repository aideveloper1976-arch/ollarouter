package router

import (
	"fmt"
	"net/http"
	"runtime"
	"time"
)

func handleMetrics(app *App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !app.Settings.PrometheusEnabled {
			writeError(w, 404, "metrics disabled")
			return
		}

		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		providers := app.Registry.List()
		var onlineCount, totalCount int
		totalCount = len(providers)
		for _, ps := range providers {
			if ps.Health.Status == "online" {
				onlineCount++
			}
		}

		metrics := fmt.Sprintf(`# HELP ollarouter_uptime_seconds Uptime in seconds
# TYPE ollarouter_uptime_seconds gauge
ollarouter_uptime_seconds %d
# HELP ollarouter_providers_total Total number of configured providers
# TYPE ollarouter_providers_total gauge
ollarouter_providers_total %d
# HELP ollarouter_providers_online Number of online providers
# TYPE ollarouter_providers_online gauge
ollarouter_providers_online %d
# HELP ollarouter_go_goroutines Number of goroutines
# TYPE ollarouter_go_goroutines gauge
ollarouter_go_goroutines %d
# HELP ollarouter_memory_alloc_bytes Memory allocated in bytes
# TYPE ollarouter_memory_alloc_bytes gauge
ollarouter_memory_alloc_bytes %d
# HELP ollarouter_version Build version info
# TYPE ollarouter_version gauge
ollarouter_version{version="%s"} 1
`,
			int(time.Since(startTime).Seconds()),
			totalCount,
			onlineCount,
			runtime.NumGoroutine(),
			memStats.Alloc,
			FallbackVersion,
		)

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(metrics))
	}
}
