package tracking

import (
	"os"
	"testing"
	"time"
)

func TestNewTracker(t *testing.T) {
	tr := NewTracker()
	if tr == nil {
		t.Fatal("NewTracker() returned nil")
	}
}

func TestRecordAndGetSummary(t *testing.T) {
	tr := NewTracker()

	// Record some usage
	tr.Record(UsageRecord{
		Timestamp:        time.Now(),
		Provider:         "openai",
		Model:            "gpt-4",
		RequestType:      "chat",
		Status:           200,
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
		CostUSD:          0.002,
		APIKeyName:       "test-key",
	})

	summary := tr.GetSummary()
	if summary["total_requests"].(int) != 1 {
		t.Errorf("Expected total_requests = 1, got %d", summary["total_requests"])
	}
	if summary["total_cost"].(float64) != 0.002 {
		t.Errorf("Expected total_cost = 0.002, got %f", summary["total_cost"])
	}
	if summary["total_tokens"].(int) != 30 {
		t.Errorf("Expected total_tokens = 30, got %d", summary["total_tokens"])
	}
}

func TestRecordMultipleProviders(t *testing.T) {
	tr := NewTracker()

	tr.Record(UsageRecord{Provider: "openai", Model: "gpt-4", Status: 200, TotalTokens: 10, CostUSD: 0.001})
	tr.Record(UsageRecord{Provider: "anthropic", Model: "claude-3", Status: 200, TotalTokens: 20, CostUSD: 0.002})
	tr.Record(UsageRecord{Provider: "openai", Model: "gpt-3.5", Status: 200, TotalTokens: 5, CostUSD: 0.0005})

	summary := tr.GetSummary()
	if summary["total_requests"].(int) != 3 {
		t.Errorf("Expected total_requests = 3, got %d", summary["total_requests"])
	}
	if summary["total_tokens"].(int) != 35 {
		t.Errorf("Expected total_tokens = 35, got %d", summary["total_tokens"])
	}

	costs := tr.GetCostByProvider()
	if costs["openai"] != 0.0015 {
		t.Errorf("Expected openai cost = 0.0015, got %f", costs["openai"])
	}
	if costs["anthropic"] != 0.002 {
		t.Errorf("Expected anthropic cost = 0.002, got %f", costs["anthropic"])
	}
}

func TestRecordErrorCount(t *testing.T) {
	tr := NewTracker()

	tr.Record(UsageRecord{Provider: "openai", Status: 200})
	tr.Record(UsageRecord{Provider: "openai", Status: 500})
	tr.Record(UsageRecord{Provider: "openai", Status: 429})

	summary := tr.GetSummary()
	if summary["total_requests"].(int) != 3 {
		t.Errorf("Expected total_requests = 3, got %d", summary["total_requests"])
	}
}

func TestGetRecentLogs(t *testing.T) {
	tr := NewTracker()

	for i := 0; i < 10; i++ {
		tr.Record(UsageRecord{
			Provider:    "openai",
			Model:       "gpt-4",
			TotalTokens: i * 10,
		})
	}

	logs := tr.GetRecentLogs(5)
	if len(logs) != 5 {
		t.Errorf("Expected 5 recent logs, got %d", len(logs))
	}

	// Last log should have the highest token count
	if logs[len(logs)-1].TotalTokens != 90 {
		t.Errorf("Expected last log TotalTokens = 90, got %d", logs[len(logs)-1].TotalTokens)
	}
}

func TestCalculateCost(t *testing.T) {
	tests := []struct {
		name             string
		promptTokens     int
		completionTokens int
		inputPrice       float64
		outputPrice      float64
		want             float64
	}{
		{"zero tokens", 0, 0, 0.01, 0.03, 0.0},
		{"gpt-4 pricing", 100, 50, 0.03, 0.06, 0.003 + 0.003},
		{"gpt-3.5 pricing", 1000, 200, 0.001, 0.002, 0.001 + 0.0004},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalculateCost(tt.promptTokens, tt.completionTokens, tt.inputPrice, tt.outputPrice)
			if got != tt.want {
				t.Errorf("CalculateCost() = %f, want %f", got, tt.want)
			}
		})
	}
}

func TestNewUsageStore(t *testing.T) {
	store := NewUsageStore()
	if store == nil {
		t.Fatal("NewUsageStore() returned nil")
	}
}

func TestUsageStoreGetData(t *testing.T) {
	store := NewUsageStore()
	store.Record(UsageRecord{
		Provider: "openai",
		Model:    "gpt-4",
		Status:   200,
	})

	data := store.GetData()
	if data == nil {
		t.Fatal("GetData() returned nil")
	}
	if len(data.Records) != 1 {
		t.Errorf("Expected 1 record, got %d", len(data.Records))
	}
	if data.Providers["openai"] == nil {
		t.Error("Expected provider stats for openai")
	}
}

func TestFilePersistenceLoadNonExistent(t *testing.T) {
	store := NewUsageStore()
	fp := NewFilePersistence(store, "/tmp/nonexistent-usage-file.json", 5*time.Minute)

	err := fp.Load()
	if err != nil {
		t.Fatalf("Load() on non-existent file should succeed: %v", err)
	}
}

func TestFilePersistenceSaveAndLoad(t *testing.T) {
	tmpFile := os.TempDir() + "/ollarouter-test-usage.json"
	defer os.Remove(tmpFile)

	// Create store with data and save
	store1 := NewUsageStore()
	store1.Record(UsageRecord{
		Provider:         "openai",
		Model:            "gpt-4",
		RequestType:      "chat",
		Status:           200,
		PromptTokens:     10,
		CompletionTokens: 20,
		TotalTokens:      30,
		CostUSD:          0.002,
	})

	fp1 := NewFilePersistence(store1, tmpFile, 5*time.Minute)
	fp1.save() // direct call to save

	// Create new store and load
	store2 := NewUsageStore()
	fp2 := NewFilePersistence(store2, tmpFile, 5*time.Minute)
	err := fp2.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	summary := store2.GetSummary()
	if summary["total_requests"].(int) != 1 {
		t.Errorf("Expected 1 record after load, got %d", summary["total_requests"])
	}
}
