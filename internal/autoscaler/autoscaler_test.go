package autoscaler

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/omnistrate-community/custom-auto-scaling-example/internal/config"
	"github.com/omnistrate-community/custom-auto-scaling-example/internal/omnistrate_api"
	"github.com/stretchr/testify/assert"
)

// Helper function to create a test autoscaler with mocked client

func createTestAutoscaler(client omnistrate_api.Client) *Autoscaler {
	// Set required env vars for config
	os.Setenv("AUTOSCALER_COOLDOWN", "0")
	os.Setenv("AUTOSCALER_TARGET_RESOURCE", "test-resource")
	os.Setenv("AUTOSCALER_STEPS", "1")
	os.Setenv("DRY_RUN", "true")
	config, err := config.NewConfigFromEnv()
	if err != nil {
		panic(err)
	}
	return &Autoscaler{
		config: config,
		client: client,
	}
}

func TestScaleToTarget_AlreadyAtTarget(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock the GetCurrentCapacity call to return capacity matching target
	expectedCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 3,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(expectedCapacity, nil).Once()

	// Call ScaleToTarget with the same capacity
	err := autoscaler.ScaleToTarget(ctx, 3)

	// Assertions
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestScaleToTarget_ScaleUp(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock the initial GetCurrentCapacity call to return lower capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 2,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock first waitForActiveState call before first scaling (already ACTIVE)
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the first AddCapacity call
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("AddCapacity", ctx, "test-resource", uint(1)).Return(expectedInstance, nil).Once()

	// Mock waitForActiveState at start of second loop iteration - capacity now 3
	intermediateCapacity := currentCapacity
	intermediateCapacity.CurrentCapacity = 3
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(intermediateCapacity, nil).Once()

	// Mock the second AddCapacity call
	mockClient.On("AddCapacity", ctx, "test-resource", uint(1)).Return(expectedInstance, nil).Once()

	// Mock waitForActiveState at start of third loop iteration - capacity now 4 (target reached, loop exits)
	finalCapacity := currentCapacity
	finalCapacity.CurrentCapacity = 4
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Mock final waitForActiveState call after loop exits
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Call ScaleToTarget
	err := autoscaler.ScaleToTarget(ctx, 4)

	// Assertions
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestScaleToTarget_ScaleDown(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock the initial GetCurrentCapacity call to return higher capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 5,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock first waitForActiveState call before first scaling (already ACTIVE)
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the first RemoveCapacity call
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("RemoveCapacity", ctx, "test-resource", uint(1)).Return(expectedInstance, nil).Once()

	// Mock waitForActiveState at start of second loop iteration - capacity now 4
	intermediateCapacity := currentCapacity
	intermediateCapacity.CurrentCapacity = 4
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(intermediateCapacity, nil).Once()

	// Mock the second RemoveCapacity call
	mockClient.On("RemoveCapacity", ctx, "test-resource", uint(1)).Return(expectedInstance, nil).Once()

	// Mock waitForActiveState at start of third loop iteration - capacity now 3 (target reached, loop exits)
	finalCapacity := currentCapacity
	finalCapacity.CurrentCapacity = 3
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Mock final waitForActiveState call after loop exits
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Call ScaleToTarget
	err := autoscaler.ScaleToTarget(ctx, 3)

	// Assertions
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestScaleToTarget_GetCurrentCapacityError(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock the GetCurrentCapacity call to return an error
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(omnistrate_api.ResourceInstanceCapacity{}, errors.New("API error"))

	// Call ScaleToTarget
	err := autoscaler.ScaleToTarget(ctx, 3)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get current capacity")
	mockClient.AssertExpectations(t)
}

func TestWaitForActiveState_InstanceFailed(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock the initial GetCurrentCapacity call to return lower capacity with STARTING status
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.STARTING,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 2,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the waitForActiveState call that will return FAILED state after polling
	failedCapacity := currentCapacity
	failedCapacity.Status = omnistrate_api.FAILED
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(failedCapacity, nil).Once()

	// Call ScaleToTarget
	err := autoscaler.ScaleToTarget(ctx, 4)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "instance is in FAILED state")
	mockClient.AssertExpectations(t)
}

func TestScaleToTarget_AddCapacityError(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock the initial GetCurrentCapacity call to return lower capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 2,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the waitForActiveState call before scaling
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the AddCapacity call to return an error
	mockClient.On("AddCapacity", ctx, "test-resource", uint(1)).Return(omnistrate_api.ResourceInstance{}, errors.New("Add capacity failed"))

	// Call ScaleToTarget
	err := autoscaler.ScaleToTarget(ctx, 3)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to scale")
	assert.Contains(t, err.Error(), "Add capacity failed")
	mockClient.AssertExpectations(t)
}

func TestScaleToTarget_RemoveCapacityError(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock the initial GetCurrentCapacity call to return higher capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 4,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the waitForActiveState call before scaling
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the RemoveCapacity call to return an error
	mockClient.On("RemoveCapacity", ctx, "test-resource", uint(1)).Return(omnistrate_api.ResourceInstance{}, errors.New("Remove capacity failed"))

	// Call ScaleToTarget
	err := autoscaler.ScaleToTarget(ctx, 3)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to scale")
	assert.Contains(t, err.Error(), "Remove capacity failed")
	mockClient.AssertExpectations(t)
}

func TestScaleToTarget_CooldownPeriod(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Set a very short cooldown for testing
	autoscaler.config.CooldownDuration = 10 * time.Millisecond
	autoscaler.lastActionTime = time.Now() // Set last action time to now

	// Mock the initial GetCurrentCapacity call to return lower capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 2,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the waitForActiveState call before scaling
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the AddCapacity call
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("AddCapacity", ctx, "test-resource", uint(1)).Return(expectedInstance, nil).Once()

	// Mock the waitForActiveState call at start of next loop iteration - capacity is now 3 (target reached)
	finalCapacity := currentCapacity
	finalCapacity.CurrentCapacity = 3
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Mock final waitForActiveState call after loop exits
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Record start time
	startTime := time.Now()

	// Call ScaleToTarget
	err := autoscaler.ScaleToTarget(ctx, 3)

	// Record end time
	endTime := time.Now()

	// Assertions
	assert.NoError(t, err)
	// Should have waited at least the cooldown duration
	assert.True(t, endTime.Sub(startTime) >= autoscaler.config.CooldownDuration)
	mockClient.AssertExpectations(t)
}

func TestGetCurrentCapacity(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock the GetCurrentCapacity call
	expectedCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 3,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(expectedCapacity, nil)

	// Call GetCurrentCapacity
	capacity, err := autoscaler.GetCurrentCapacity(ctx)

	// Assertions
	assert.NoError(t, err)
	assert.NotNil(t, capacity)
	assert.Equal(t, expectedCapacity.InstanceID, capacity.InstanceID)
	assert.Equal(t, expectedCapacity.Status, capacity.Status)
	assert.Equal(t, expectedCapacity.CurrentCapacity, capacity.CurrentCapacity)
	mockClient.AssertExpectations(t)
}

func TestGetCurrentCapacity_Error(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock the GetCurrentCapacity call to return an error
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(omnistrate_api.ResourceInstanceCapacity{}, errors.New("API error"))

	// Call GetCurrentCapacity
	capacity, err := autoscaler.GetCurrentCapacity(ctx)

	// Assertions
	assert.Error(t, err)
	assert.Nil(t, capacity)
	assert.Contains(t, err.Error(), "API error")
	mockClient.AssertExpectations(t)
}

func TestGetConfig(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)

	// Call GetConfig
	config := autoscaler.GetConfig()

	// Assertions
	assert.NotNil(t, config)
	assert.Equal(t, "test-resource", config.TargetResource)
	assert.Equal(t, uint(1), config.Steps)
	assert.Equal(t, 0*time.Second, config.CooldownDuration) // Set to 0 via env var
	assert.True(t, config.DryRun)
}

func TestScaleUp_MultipleSteps(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	autoscaler.config.Steps = 2 // Set steps to 2
	ctx := context.Background()

	// Mock the initial GetCurrentCapacity call to return lower capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 1,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock first waitForActiveState call before scaling
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the AddCapacity call with steps=2 (should take us from 1 to 3)
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("AddCapacity", ctx, "test-resource", uint(2)).Return(expectedInstance, nil).Once()

	// Mock waitForActiveState at start of next loop iteration - capacity is now 3 (target reached, loop exits)
	finalCapacity := currentCapacity
	finalCapacity.CurrentCapacity = 3
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Mock final waitForActiveState call after the loop exits
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Call ScaleToTarget (need to scale up from 1 to 3)
	err := autoscaler.ScaleToTarget(ctx, 3)

	// Assertions
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestScaleDown_MultipleSteps(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	autoscaler.config.Steps = 2 // Set steps to 2
	ctx := context.Background()

	// Mock the initial GetCurrentCapacity call to return higher capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 5,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock first waitForActiveState call before scaling
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the first RemoveCapacity call with steps=2
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("RemoveCapacity", ctx, "test-resource", uint(2)).Return(expectedInstance, nil).Once()

	// Mock waitForActiveState at start of next loop iteration - capacity is now 3 (target reached, loop exits)
	finalCapacity := currentCapacity
	finalCapacity.CurrentCapacity = 3
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Mock final waitForActiveState call after loop exits
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Call ScaleToTarget (need to scale down by 2)
	err := autoscaler.ScaleToTarget(ctx, 3)

	// Assertions
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestScaleDown_LimitedByCurrentCapacity(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	autoscaler.config.Steps = 3 // Set steps to 3, but current capacity is only 2
	ctx := context.Background()

	// Mock the initial GetCurrentCapacity call to return capacity of 2
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 2,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock first waitForActiveState call before scaling
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the RemoveCapacity call - should only remove 2 (current capacity), not 3 (steps)
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("RemoveCapacity", ctx, "test-resource", uint(2)).Return(expectedInstance, nil).Once()

	// Mock the waitForActiveState at start of next loop iteration - capacity is now 0 (target reached)
	finalCapacity := currentCapacity
	finalCapacity.CurrentCapacity = 0
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Mock final waitForActiveState call after loop exits (target reached)
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Call ScaleToTarget to scale down to 0
	err := autoscaler.ScaleToTarget(ctx, 0)

	// Assertions
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestWaitForActiveState_Success(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock instance with STARTING status first
	startingCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.STARTING,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 2,
	}

	activeCapacity := startingCapacity
	activeCapacity.Status = omnistrate_api.ACTIVE

	// Mock the initial call returning STARTING status
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(startingCapacity, nil).Once()

	// Mock the first waitForActiveState call that polls and eventually returns ACTIVE status (capacity still 2)
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(activeCapacity, nil).Once()

	// Mock the AddCapacity call (scaling from 2 to 3)
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("AddCapacity", ctx, "test-resource", uint(1)).Return(expectedInstance, nil).Once()

	// Mock the waitForActiveState at start of next loop iteration - now shows updated capacity of 3 (target reached, loop exits)
	finalCapacity := activeCapacity
	finalCapacity.CurrentCapacity = 3
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Mock final waitForActiveState call after loop exits
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Call ScaleToTarget to trigger waitForActiveState behavior
	err := autoscaler.ScaleToTarget(ctx, 3)

	// Assertions
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestWaitForActiveState_Failed(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock instance with FAILED status
	failedCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.FAILED,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 3,
	}

	// Mock the initial call
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(failedCapacity, nil).Once()
	// Mock the waitForActiveState call that returns FAILED
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(failedCapacity, nil).Once()

	// Call ScaleToTarget which will trigger waitForActiveState
	err := autoscaler.ScaleToTarget(ctx, 2) // Different target to trigger scaling

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "instance is in FAILED state")
	mockClient.AssertExpectations(t)
}

func TestScaleToTarget_ScaleDownBeyondMinimum(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock the initial GetCurrentCapacity call to return capacity of 1
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 1,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the first waitForActiveState call before scaling (returns capacity 1)
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Mock the first RemoveCapacity call - should only remove 1 (current capacity), not steps
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("RemoveCapacity", ctx, "test-resource", uint(1)).Return(expectedInstance, nil).Once()

	// Mock the waitForActiveState at start of next loop iteration - capacity is now 0 (target reached)
	finalCapacity := currentCapacity
	finalCapacity.CurrentCapacity = 0
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Mock final waitForActiveState call after loop exits (target reached)
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(finalCapacity, nil).Once()

	// Call ScaleToTarget to scale down to 0
	err := autoscaler.ScaleToTarget(ctx, 0)

	// Assertions
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}
