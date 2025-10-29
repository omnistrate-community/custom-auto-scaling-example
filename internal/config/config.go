package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	CooldownDuration           time.Duration
	TargetResource             string
	Steps                      uint
	DryRun                     bool
	WaitForActiveTimeout       time.Duration
	WaitForActiveCheckInterval time.Duration
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

	// Get dry run flag
	dryRunStr := os.Getenv("AUTOSCALER_DRY_RUN")
	dryRun := false // Default to false
	if dryRunStr != "" {
		dryRun, err = strconv.ParseBool(dryRunStr)
		if err != nil {
			return nil, fmt.Errorf("invalid AUTOSCALER_DRY_RUN value: %s", dryRunStr)
		}
	}

	// Get wait for active timeout
	waitForActiveTimeoutStr := os.Getenv("AUTOSCALER_WAIT_FOR_ACTIVE_TIMEOUT")
	if waitForActiveTimeoutStr == "" {
		waitForActiveTimeoutStr = "900" // Default 15 minutes
	}
	waitForActiveTimeoutSeconds, err := strconv.Atoi(waitForActiveTimeoutStr)
	if err != nil {
		return nil, fmt.Errorf("invalid AUTOSCALER_WAIT_FOR_ACTIVE_TIMEOUT value: %s", waitForActiveTimeoutStr)
	}

	// Get wait for active check interval
	waitForActiveCheckIntervalStr := os.Getenv("AUTOSCALER_WAIT_FOR_ACTIVE_CHECK_INTERVAL")
	if waitForActiveCheckIntervalStr == "" {
		waitForActiveCheckIntervalStr = "30" // Default 30 seconds
	}
	waitForActiveCheckIntervalSeconds, err := strconv.Atoi(waitForActiveCheckIntervalStr)
	if err != nil {
		return nil, fmt.Errorf("invalid AUTOSCALER_WAIT_FOR_ACTIVE_CHECK_INTERVAL value: %s", waitForActiveCheckIntervalStr)
	}

	return &Config{
		CooldownDuration:           time.Duration(cooldownSeconds) * time.Second,
		TargetResource:             targetResource,
		Steps:                      uint(steps),
		DryRun:                     dryRun,
		WaitForActiveTimeout:       time.Duration(waitForActiveTimeoutSeconds) * time.Second,
		WaitForActiveCheckInterval: time.Duration(waitForActiveCheckIntervalSeconds) * time.Second,
	}, nil
}
