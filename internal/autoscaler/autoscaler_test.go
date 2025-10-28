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
	"github.com/stretchr/testify/mock"
)

// MockClient is a mock implementation of the omnistrate_api.Client interface
type MockClient struct {
	mock.Mock
}

func (m *MockClient) GetCurrentCapacity(ctx context.Context, resourceAlias string) (omnistrate_api.ResourceInstanceCapacity, error) {
	args := m.Called(ctx, resourceAlias)
	return args.Get(0).(omnistrate_api.ResourceInstanceCapacity), args.Error(1)
}

func (m *MockClient) AddCapacity(ctx context.Context, resourceAlias string, capacityToBeAdded uint) (omnistrate_api.ResourceInstance, error) {
	args := m.Called(ctx, resourceAlias, capacityToBeAdded)
	return args.Get(0).(omnistrate_api.ResourceInstance), args.Error(1)
}

func (m *MockClient) RemoveCapacity(ctx context.Context, resourceAlias string, capacityToBeRemoved uint) (omnistrate_api.ResourceInstance, error) {
	args := m.Called(ctx, resourceAlias, capacityToBeRemoved)
	return args.Get(0).(omnistrate_api.ResourceInstance), args.Error(1)
}

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
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(expectedCapacity, nil)

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

	// Mock the GetCurrentCapacity call to return lower capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 2,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil)

	// Mock the AddCapacity calls (need to scale up by 2)
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("AddCapacity", ctx, "test-resource", uint(1)).Return(expectedInstance, nil).Times(2)

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

	// Mock the GetCurrentCapacity call to return higher capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 5,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil)

	// Mock the RemoveCapacity calls (need to scale down by 2)
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("RemoveCapacity", ctx, "test-resource", uint(1)).Return(expectedInstance, nil).Times(2)

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

func TestScaleToTarget_WaitForActiveState_InstanceNotActive(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock the GetCurrentCapacity call to return lower capacity with ACTIVE status
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE, // Start with ACTIVE to skip wait
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 2,
	}

	// First call for getting current capacity
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil)

	// Mock the AddCapacity calls
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("AddCapacity", ctx, "test-resource", uint(1)).Return(expectedInstance, nil).Times(2)

	// Call ScaleToTarget
	err := autoscaler.ScaleToTarget(ctx, 4)

	// Assertions
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}

func TestScaleToTarget_WaitForActiveState_InstanceFailed(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	ctx := context.Background()

	// Mock the GetCurrentCapacity call to return lower capacity with FAILED status
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.STARTING,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 2,
	}

	// First call for getting current capacity
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil).Once()

	// Subsequent call for waiting - return FAILED state immediately
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

	// Mock the GetCurrentCapacity call to return lower capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 2,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil)

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

	// Mock the GetCurrentCapacity call to return higher capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 4,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil)

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

	// Mock the GetCurrentCapacity call to return lower capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 2,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil)

	// Mock the AddCapacity call
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("AddCapacity", ctx, "test-resource", uint(1)).Return(expectedInstance, nil)

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
	assert.False(t, config.DryRun)
}

func TestScaleUp_MultipleSteps(t *testing.T) {
	t.Parallel()

	mockClient := new(MockClient)
	autoscaler := createTestAutoscaler(mockClient)
	autoscaler.config.Steps = 2 // Set steps to 2
	ctx := context.Background()

	// Mock the GetCurrentCapacity call to return lower capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 1,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil)

	// Mock the AddCapacity calls with steps=2
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("AddCapacity", ctx, "test-resource", uint(2)).Return(expectedInstance, nil).Times(2)

	// Call ScaleToTarget (need to scale up by 2)
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

	// Mock the GetCurrentCapacity call to return higher capacity
	currentCapacity := omnistrate_api.ResourceInstanceCapacity{
		InstanceID:      "test-instance",
		Status:          omnistrate_api.ACTIVE,
		ResourceID:      "test-resource-id",
		ResourceAlias:   "test-resource",
		CurrentCapacity: 5,
	}
	mockClient.On("GetCurrentCapacity", ctx, "test-resource").Return(currentCapacity, nil)

	// Mock the RemoveCapacity calls with steps=2
	expectedInstance := omnistrate_api.ResourceInstance{
		InstanceID:    "test-instance",
		ResourceID:    "test-resource-id",
		ResourceAlias: "test-resource",
	}
	mockClient.On("RemoveCapacity", ctx, "test-resource", uint(2)).Return(expectedInstance, nil).Times(2)

	// Call ScaleToTarget (need to scale down by 2)
	err := autoscaler.ScaleToTarget(ctx, 3)

	// Assertions
	assert.NoError(t, err)
	mockClient.AssertExpectations(t)
}
