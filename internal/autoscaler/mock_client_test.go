package autoscaler

import (
	"context"

	"github.com/omnistrate-community/custom-auto-scaling-example/internal/omnistrate_api"
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
