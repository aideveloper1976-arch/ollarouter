package config

import "testing"

// TestExampleConfigParses keeps providers.yaml.example loadable.
func TestExampleConfigParses(t *testing.T) {
	cfg, err := ParseFile("../../providers.yaml.example")
	if err != nil {
		t.Fatalf("providers.yaml.example does not parse: %v", err)
	}
	if len(cfg.Providers) == 0 || len(cfg.VirtualModels) == 0 {
		t.Fatalf("example should define providers and virtual models, got %d and %d", len(cfg.Providers), len(cfg.VirtualModels))
	}
	if cfg.Settings.CircuitBreakerThreshold != 5 {
		t.Fatalf("unexpected circuit_breaker_threshold %d", cfg.Settings.CircuitBreakerThreshold)
	}
}
