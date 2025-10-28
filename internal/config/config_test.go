package config

import (
	"testing"
	"time"
)

func TestConfigFields(t *testing.T) {
	// Create a Config struct with all fields
	cfg := Config{
		CooldownDuration: 5 * time.Minute,
		TargetResource:   "test-resource",
		Steps:            3,
		DryRun:           true,
	}

	// Verify CooldownDuration field
	if cfg.CooldownDuration != 5*time.Minute {
		t.Errorf("expected CooldownDuration 5m, got %v", cfg.CooldownDuration)
	}

	// Verify TargetResource field
	if cfg.TargetResource != "test-resource" {
		t.Errorf("expected TargetResource 'test-resource', got %s", cfg.TargetResource)
	}

	// Verify Steps field
	if cfg.Steps != 3 {
		t.Errorf("expected Steps 3, got %d", cfg.Steps)
	}

	// Verify DryRun field
	if cfg.DryRun != true {
		t.Errorf("expected DryRun true, got %t", cfg.DryRun)
	}
}

// The following test assumes NewConfigFromEnv is available in this package.
func TestConfigFromEnv(t *testing.T) {
	// Set up environment variables for testing
	t.Setenv("AUTOSCALER_COOLDOWN", "120")
	t.Setenv("AUTOSCALER_TARGET_RESOURCE", "env-resource")
	t.Setenv("AUTOSCALER_STEPS", "5")
	t.Setenv("DRY_RUN", "true")

	// Call NewConfigFromEnv to load configuration
	cfg, err := NewConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify CooldownDuration is correctly parsed from environment
	if cfg.CooldownDuration != 120*time.Second {
		t.Errorf("expected CooldownDuration 120s, got %v", cfg.CooldownDuration)
	}

	// Verify TargetResource is correctly loaded from environment
	if cfg.TargetResource != "env-resource" {
		t.Errorf("expected TargetResource 'env-resource', got %s", cfg.TargetResource)
	}

	// Verify Steps is correctly parsed from environment
	if cfg.Steps != 5 {
		t.Errorf("expected Steps 5, got %d", cfg.Steps)
	}

	// Verify DryRun is correctly parsed from environment
	if cfg.DryRun != true {
		t.Errorf("expected DryRun true, got %t", cfg.DryRun)
	}
}

func TestConfigFromEnv_Defaults(t *testing.T) {
	// Set up minimal environment (only required AUTOSCALER_TARGET_RESOURCE)
	t.Setenv("AUTOSCALER_TARGET_RESOURCE", "default-resource")
	// Explicitly unset optional environment variables to test defaults
	t.Setenv("AUTOSCALER_COOLDOWN", "")
	t.Setenv("AUTOSCALER_STEPS", "")
	t.Setenv("DRY_RUN", "")

	// Call NewConfigFromEnv with minimal configuration
	cfg, err := NewConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify default CooldownDuration (5 minutes)
	expectedCooldown := 300 * time.Second // 5 minutes default
	if cfg.CooldownDuration != expectedCooldown {
		t.Errorf("expected default CooldownDuration %v, got %v", expectedCooldown, cfg.CooldownDuration)
	}

	// Verify TargetResource is set from environment
	if cfg.TargetResource != "default-resource" {
		t.Errorf("expected TargetResource 'default-resource', got %s", cfg.TargetResource)
	}

	// Verify default Steps (1)
	if cfg.Steps != 1 {
		t.Errorf("expected default Steps 1, got %d", cfg.Steps)
	}

	// Verify default DryRun (false)
	if cfg.DryRun != false {
		t.Errorf("expected default DryRun false, got %t", cfg.DryRun)
	}
}

func TestConfigFromEnv_MissingTargetResource(t *testing.T) {
	// Unset AUTOSCALER_TARGET_RESOURCE to test error handling
	t.Setenv("AUTOSCALER_TARGET_RESOURCE", "")

	// Call NewConfigFromEnv and expect an error
	_, err := NewConfigFromEnv()

	// Verify that an error is returned for missing required field
	if err == nil {
		t.Error("expected error for missing AUTOSCALER_TARGET_RESOURCE, got nil")
	}
}

func TestConfigFromEnv_InvalidCooldown(t *testing.T) {
	// Set up environment with invalid cooldown value
	t.Setenv("AUTOSCALER_COOLDOWN", "invalid")
	t.Setenv("AUTOSCALER_TARGET_RESOURCE", "test-resource")

	// Call NewConfigFromEnv and expect an error
	_, err := NewConfigFromEnv()

	// Verify that an error is returned for invalid cooldown
	if err == nil {
		t.Error("expected error for invalid AUTOSCALER_COOLDOWN, got nil")
	}
}

func TestConfigFromEnv_InvalidSteps(t *testing.T) {
	// Set up environment with invalid steps value
	t.Setenv("AUTOSCALER_COOLDOWN", "120")
	t.Setenv("AUTOSCALER_TARGET_RESOURCE", "test-resource")
	t.Setenv("AUTOSCALER_STEPS", "invalid")

	// Call NewConfigFromEnv and expect an error
	_, err := NewConfigFromEnv()

	// Verify that an error is returned for invalid steps
	if err == nil {
		t.Error("expected error for invalid AUTOSCALER_STEPS, got nil")
	}
}

func TestConfigFromEnv_DryRunTrue(t *testing.T) {
	// Set up environment with DRY_RUN=true
	t.Setenv("AUTOSCALER_TARGET_RESOURCE", "test-resource")
	t.Setenv("DRY_RUN", "true")

	// Call NewConfigFromEnv to load configuration
	cfg, err := NewConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify DryRun is set to true
	if cfg.DryRun != true {
		t.Errorf("expected DryRun true, got %t", cfg.DryRun)
	}
}

func TestConfigFromEnv_DryRunFalse(t *testing.T) {
	// Set up environment with DRY_RUN=false
	t.Setenv("AUTOSCALER_TARGET_RESOURCE", "test-resource")
	t.Setenv("DRY_RUN", "false")

	// Call NewConfigFromEnv to load configuration
	cfg, err := NewConfigFromEnv()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify DryRun is set to false
	if cfg.DryRun != false {
		t.Errorf("expected DryRun false, got %t", cfg.DryRun)
	}
}

func TestConfigFromEnv_DryRunVariations(t *testing.T) {
	testCases := []struct {
		name     string
		value    string
		expected bool
	}{
		{"1", "1", true},
		{"0", "0", false},
		{"yes", "yes", true},
		{"no", "no", false},
		{"TRUE", "TRUE", true},
		{"FALSE", "FALSE", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Set up environment
			t.Setenv("AUTOSCALER_TARGET_RESOURCE", "test-resource")
			t.Setenv("DRY_RUN", tc.value)

			// Call NewConfigFromEnv to load configuration
			cfg, err := NewConfigFromEnv()
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Verify DryRun is correctly parsed
			if cfg.DryRun != tc.expected {
				t.Errorf("expected DryRun %t for value '%s', got %t", tc.expected, tc.value, cfg.DryRun)
			}
		})
	}
}

func TestConfigFromEnv_InvalidDryRun(t *testing.T) {
	// Set up environment with invalid DRY_RUN value
	t.Setenv("AUTOSCALER_TARGET_RESOURCE", "test-resource")
	t.Setenv("DRY_RUN", "invalid")

	// Call NewConfigFromEnv and expect an error
	_, err := NewConfigFromEnv()

	// Verify that an error is returned for invalid DRY_RUN
	if err == nil {
		t.Error("expected error for invalid DRY_RUN, got nil")
	}
}
