package autoscaler

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/omnistrate-community/custom-auto-scaling-example/internal/config"
	"github.com/omnistrate-community/custom-auto-scaling-example/internal/omnistrate_api"
)

type Autoscaler struct {
	config         *config.Config
	client         omnistrate_api.Client
	lastActionTime time.Time
}

// NewAutoscaler creates a new autoscaler instance with configuration from environment variables
func NewAutoscaler(ctx context.Context) (*Autoscaler, error) {
	config, err := config.NewConfigFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	client := omnistrate_api.NewClient(config)

	return &Autoscaler{
		config: config,
		client: client,
	}, nil
}

// ScaleToTarget scales the resource to match the target capacity
func (a *Autoscaler) ScaleToTarget(ctx context.Context, targetCapacity int) error {
	log.Printf("Scaling to target capacity: %d", targetCapacity)

	// Check if we're within cooldown period
	if !a.lastActionTime.IsZero() && time.Since(a.lastActionTime) < a.config.CooldownDuration {
		waitTime := a.config.CooldownDuration - time.Since(a.lastActionTime)
		log.Printf("Within cooldown period, waiting %v before scaling", waitTime)
		time.Sleep(waitTime)
	}

	// Get current capacity
	currentCapacity, err := a.getCurrentCapacity(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current capacity: %w", err)
	}

	log.Printf("Current capacity: %d, Target capacity: %d", currentCapacity.CurrentCapacity, targetCapacity)

	// Check if scaling is needed
	if currentCapacity.CurrentCapacity == targetCapacity {
		log.Printf("Already at target capacity: %d", targetCapacity)
		return nil
	}

	// Wait for instance to be in ACTIVE state
	err = a.waitForActiveState(ctx)
	if err != nil {
		return fmt.Errorf("failed to wait for active state: %w", err)
	}

	// Perform scaling operation
	if currentCapacity.CurrentCapacity < targetCapacity {
		err = a.scaleUp(ctx, targetCapacity-currentCapacity.CurrentCapacity)
	} else {
		err = a.scaleDown(ctx, currentCapacity.CurrentCapacity-targetCapacity)
	}

	if err != nil {
		return fmt.Errorf("failed to scale: %w", err)
	}

	// Update last action time
	a.lastActionTime = time.Now()
	log.Printf("Scaling operation completed successfully")

	return nil
}

// getCurrentCapacity gets the current capacity of the resource
func (a *Autoscaler) getCurrentCapacity(ctx context.Context) (*omnistrate_api.ResourceInstanceCapacity, error) {
	capacity, err := a.client.GetCurrentCapacity(ctx, a.config.TargetResource)
	if err != nil {
		return nil, err
	}
	return &capacity, nil
}

// waitForActiveState waits for the instance to be in ACTIVE state
func (a *Autoscaler) waitForActiveState(ctx context.Context) error {
	log.Printf("Waiting for instance to be in ACTIVE state")

	maxWaitTime := 10 * time.Minute
	checkInterval := 30 * time.Second
	timeout := time.After(maxWaitTime)
	ticker := time.NewTicker(checkInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout:
			return fmt.Errorf("timeout waiting for instance to become ACTIVE")
		case <-ticker.C:
			capacity, err := a.getCurrentCapacity(ctx)
			if err != nil {
				log.Printf("Error checking instance status: %v", err)
				continue
			}

			log.Printf("Current instance status: %s", capacity.Status)
			if capacity.Status == omnistrate_api.ACTIVE {
				log.Printf("Instance is now ACTIVE")
				return nil
			}

			if capacity.Status == omnistrate_api.FAILED {
				return fmt.Errorf("instance is in FAILED state")
			}

			log.Printf("Instance status is %s, waiting...", capacity.Status)
		}
	}
}

// scaleUp adds capacity to the resource
func (a *Autoscaler) scaleUp(ctx context.Context, increaseBy int) error {
	log.Printf("Scaling up by %d instances", increaseBy)

	for i := 0; i < increaseBy; i++ {
		_, err := a.client.AddCapacity(ctx, a.config.TargetResource, a.config.Steps)
		if err != nil {
			return fmt.Errorf("failed to add capacity (iteration %d): %w", i+1, err)
		}
		log.Printf("Added capacity: %d/%d", i+1, increaseBy)

		// Small delay between operations to avoid overwhelming the API
		if i < increaseBy-1 {
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

// scaleDown removes capacity from the resource
func (a *Autoscaler) scaleDown(ctx context.Context, decreaseBy int) error {
	log.Printf("Scaling down by %d instances", decreaseBy)

	for i := 0; i < decreaseBy; i++ {
		_, err := a.client.RemoveCapacity(ctx, a.config.TargetResource, a.config.Steps)
		if err != nil {
			return fmt.Errorf("failed to remove capacity (iteration %d): %w", i+1, err)
		}
		log.Printf("Removed capacity: %d/%d", i+1, decreaseBy)

		// Small delay between operations to avoid overwhelming the API
		if i < decreaseBy-1 {
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

// GetCurrentCapacity returns the current capacity of the resource (public method for external use)
func (a *Autoscaler) GetCurrentCapacity(ctx context.Context) (*omnistrate_api.ResourceInstanceCapacity, error) {
	return a.getCurrentCapacity(ctx)
}

// GetConfig returns the current configuration
func (a *Autoscaler) GetConfig() *config.Config {
	return a.config
}
