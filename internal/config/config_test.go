package config

import (
	"testing"
	"time"
)

func TestConfigFields(t *testing.T) {
	cfg := Config{
		CooldownDuration: 5 * time.Minute,
		TargetResource:   "test-resource",
	}

	if cfg.CooldownDuration != 5*time.Minute {
		t.Errorf("expected CooldownDuration 5m, got %v", cfg.CooldownDuration)
	}
	if cfg.TargetResource != "test-resource" {
		t.Errorf("expected TargetResource 'test-resource', got %s", cfg.TargetResource)
	}
}

// The following test assumes NewConfigFromEnv is available in this package.
func TestConfigFromEnv(t *testing.T) {
	t.Setenv("AUTOSCALER_COOLDOWN", "120")
	t.Setenv("AUTOSCALER_TARGET_RESOURCE", "env-resource")

	cfg, err := NewConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.CooldownDuration != 120*time.Second {
		t.Errorf("expected CooldownDuration 120s, got %v", cfg.CooldownDuration)
	}
	if cfg.TargetResource != "env-resource" {
		t.Errorf("expected TargetResource 'env-resource', got %s", cfg.TargetResource)
	}
}
