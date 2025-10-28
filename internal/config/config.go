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
	Steps            uint
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

	// Get steps
	stepsStr := os.Getenv("AUTOSCALER_STEPS")
	if stepsStr == "" {
		stepsStr = "1" // Default 1 step
	}
	steps, err := strconv.Atoi(stepsStr)
	if err != nil {
		return nil, fmt.Errorf("invalid AUTOSCALER_STEPS value: %s", stepsStr)
	}

	return &Config{
		CooldownDuration: time.Duration(cooldownSeconds) * time.Second,
		TargetResource:   targetResource,
		Steps:            uint(steps),
	}, nil
}
