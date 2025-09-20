package order

import (
	"context"
	"testing"

	"github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOrderRepository for testing
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) GetOrderItemsByOrderID(ctx context.Context, orderID uint64) ([]oModel.OrderItems, error) {
	args := m.Called(ctx, orderID)
	return args.Get(0).([]oModel.OrderItems), args.Error(1)
}

func TestGetOrderItemsByOrderID_Success(t *testing.T) {
	mockRepo := new(MockOrderRepository)

	expectedItems := []oModel.OrderItems{
		{
			ID:        1,
			IdOrder:   1,
			IdProduct: 1,
			Name:      "Product 1",
			Quantity:  3,
		},
		{
			ID:        2,
			IdOrder:   1,
			IdProduct: 2,
			Name:      "Product 2",
			Quantity:  2,
		},
	}

	// Setup expectations
	mockRepo.On("GetOrderItemsByOrderID", mock.Anything, uint64(1)).Return(expectedItems, nil)

	// Execute
	items, err := mockRepo.GetOrderItemsByOrderID(context.Background(), 1)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, expectedItems, items)
	mockRepo.AssertExpectations(t)
}

func TestGetOrderItemsByOrderID_OrderNotFound(t *testing.T) {
	mockRepo := new(MockOrderRepository)

	// Setup expectations
	mockRepo.On("GetOrderItemsByOrderID", mock.Anything, uint64(999)).Return([]oModel.OrderItems{}, errors.ErrOrderNotFound)

	// Execute
	items, err := mockRepo.GetOrderItemsByOrderID(context.Background(), 999)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, errors.ErrOrderNotFound, err)
	assert.Empty(t, items)
	mockRepo.AssertExpectations(t)
}

func TestGetOrderItemsByOrderID_EmptyOrder(t *testing.T) {
	mockRepo := new(MockOrderRepository)

	// Setup expectations
	mockRepo.On("GetOrderItemsByOrderID", mock.Anything, uint64(2)).Return([]oModel.OrderItems{}, nil)

	// Execute
	items, err := mockRepo.GetOrderItemsByOrderID(context.Background(), 2)

	// Assert
	assert.NoError(t, err)
	assert.Empty(t, items)
	mockRepo.AssertExpectations(t)
}

func TestGetOrderItemsByOrderID_SingleItem(t *testing.T) {
	mockRepo := new(MockOrderRepository)

	expectedItems := []oModel.OrderItems{
		{
			ID:        1,
			IdOrder:   3,
			IdProduct: 1,
			Name:      "Single Product",
			Quantity:  5,
		},
	}

	// Setup expectations
	mockRepo.On("GetOrderItemsByOrderID", mock.Anything, uint64(3)).Return(expectedItems, nil)

	// Execute
	items, err := mockRepo.GetOrderItemsByOrderID(context.Background(), 3)

	// Assert
	assert.NoError(t, err)
	assert.Len(t, items, 1)
	assert.Equal(t, expectedItems, items)
	mockRepo.AssertExpectations(t)
}
