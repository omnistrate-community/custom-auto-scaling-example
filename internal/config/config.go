package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	CooldownDuration time.Duration
	TargetResource   string
}

// NewConfigFromEnv loads configuration from environment variables
func NewConfigFromEnv() (*Config, error) {
	// Get cooldown duration
	cooldownStr := os.Getenv("AUTOSCALER_COOLDOWN")
	if cooldownStr == "" {
		cooldownStr = "300" // Default 5 minutes
	}
	cooldownSeconds, err := strconv.Atoi(cooldownStr)
	if err != nil {
		return nil, fmt.Errorf("invalid AUTOSCALER_COOLDOWN value: %s", cooldownStr)
	}

	// Get target resource
	targetResource := os.Getenv("AUTOSCALER_TARGET_RESOURCE")
	if targetResource == "" {
		return nil, fmt.Errorf("AUTOSCALER_TARGET_RESOURCE environment variable is required")
	}
	return &Config{
		CooldownDuration: time.Duration(cooldownSeconds) * time.Second,
		TargetResource:   targetResource,
	}, nil
}
