package tracking

import (
	"strings"
	"sync"
	"time"
)

// UsageRecord holds usage data for a single request.
type UsageRecord struct {
	Timestamp        time.Time `json:"timestamp"`
	Provider         string    `json:"provider"`
	Model            string    `json:"model"`
	RequestType      string    `json:"request_type"` // "chat" or "generate"
	Status           int       `json:"status"`
	LatencyMs        int64     `json:"latency_ms"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	CostUSD          float64   `json:"cost_usd"`
	APIKeyName       string    `json:"api_key_name"`
}

// Tracker records and aggregates usage data.
type Tracker struct {
	mu        sync.RWMutex
	records   []UsageRecord
	providers map[string]*providerStats
	models    map[string]*modelStats
}

type providerStats struct {
	Requests    int     `json:"requests"`
	TotalTokens int     `json:"total_tokens"`
	TotalCost   float64 `json:"total_cost"`
	Errors      int     `json:"errors"`
}

type modelStats struct {
	Requests    int     `json:"requests"`
	TotalTokens int     `json:"total_tokens"`
	TotalCost   float64 `json:"total_cost"`
}

// NewTracker creates a new usage tracker.
func NewTracker() *Tracker {
	return &Tracker{
		providers: make(map[string]*providerStats),
		models:    make(map[string]*modelStats),
	}
}

// Record records a usage record.
func (t *Tracker) Record(record UsageRecord) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.records = append(t.records, record)

	// Update provider stats
	if _, ok := t.providers[record.Provider]; !ok {
		t.providers[record.Provider] = &providerStats{}
	}
	ps := t.providers[record.Provider]
	ps.Requests++
	ps.TotalTokens += record.TotalTokens
	ps.TotalCost += record.CostUSD
	if record.Status >= 400 {
		ps.Errors++
	}

	// Update model stats
	modelKey := record.Provider + "/" + record.Model
	if _, ok := t.models[modelKey]; !ok {
		t.models[modelKey] = &modelStats{}
	}
	ms := t.models[modelKey]
	ms.Requests++
	ms.TotalTokens += record.TotalTokens
	ms.TotalCost += record.CostUSD
}

// GetSummary returns a summary of all usage.
func (t *Tracker) GetSummary() map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	totalRequests := len(t.records)
	totalCost := 0.0
	totalTokens := 0

	for _, ps := range t.providers {
		totalCost += ps.TotalCost
	}
	for _, r := range t.records {
		totalTokens += r.TotalTokens
	}

	return map[string]interface{}{
		"total_requests":   totalRequests,
		"total_cost":       totalCost,
		"total_tokens":     totalTokens,
		"providers_active": len(t.providers),
	}
}

// GetCostByProvider returns cost breakdown by provider.
func (t *Tracker) GetCostByProvider() map[string]float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string]float64)
	for name, ps := range t.providers {
		result[name] = ps.TotalCost
	}
	return result
}

// GetRecentLogs returns the most recent log entries.
func (t *Tracker) GetRecentLogs(limit int) []UsageRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if len(t.records) <= limit {
		return t.records
	}
	return t.records[len(t.records)-limit:]
}

// GetRecentLogsFiltered returns recent logs filtered by provider name.
func (t *Tracker) GetRecentLogsFiltered(limit int, provider string) []UsageRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if provider == "" {
		return t.GetRecentLogs(limit)
	}

	var filtered []UsageRecord
	for i := len(t.records) - 1; i >= 0; i-- {
		r := t.records[i]
		if r.Provider == provider {
			filtered = append(filtered, r)
			if len(filtered) >= limit {
				break
			}
		}
	}

	// reverse to chronological order
	reverse := make([]UsageRecord, len(filtered))
	for i, r := range filtered {
		reverse[len(filtered)-1-i] = r
	}
	return reverse
}

// CalculateCost calculates the cost of a request based on token usage and pricing.
func CalculateCost(promptTokens, completionTokens int, inputPricePer1K, outputPricePer1K float64) float64 {
	inputCost := float64(promptTokens) / 1000.0 * inputPricePer1K
	outputCost := float64(completionTokens) / 1000.0 * outputPricePer1K
	return inputCost + outputCost
}

// GetCostByModel returns cost breakdown by model, grouped by provider.
func (t *Tracker) GetCostByModel() map[string][]map[string]interface{} {
	t.mu.RLock()
	defer t.mu.RUnlock()

	result := make(map[string][]map[string]interface{})
	for key, ms := range t.models {
		parts := strings.SplitN(key, "/", 2)
		provider := parts[0]
		model := parts[1]
		result[provider] = append(result[provider], map[string]interface{}{
			"model":        model,
			"requests":     ms.Requests,
			"total_tokens": ms.TotalTokens,
			"total_cost":   ms.TotalCost,
		})
	}
	return result
}
