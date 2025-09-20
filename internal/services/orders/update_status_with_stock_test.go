package orders

import (
	"context"
	"testing"

	"github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOrderStatusRepositoryWithStock for testing stock reversion
type MockOrderStatusRepositoryWithStock struct {
	mock.Mock
}

func (m *MockOrderStatusRepositoryWithStock) GetOrderByID(ctx context.Context, id uint64) (oModel.OrderResponse, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(oModel.OrderResponse), args.Error(1)
}

func (m *MockOrderStatusRepositoryWithStock) UpdateOrderStatus(ctx context.Context, orderID uint64, status oModel.OrderStatus) error {
	args := m.Called(ctx, orderID, status)
	return args.Error(0)
}

func (m *MockOrderStatusRepositoryWithStock) CreateOrderHistory(ctx context.Context, order oModel.OrderHistory) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderStatusRepositoryWithStock) GetOrderItemsByOrderID(ctx context.Context, orderID uint64) ([]oModel.OrderItems, error) {
	args := m.Called(ctx, orderID)
	return args.Get(0).([]oModel.OrderItems), args.Error(1)
}

// MockProductRepositoryWithStock for testing stock reversion
type MockProductRepositoryWithStock struct {
	mock.Mock
}

func (m *MockProductRepositoryWithStock) RevertProductStock(ctx context.Context, idProduct uint64, quantityToRevert uint64) error {
	args := m.Called(ctx, idProduct, quantityToRevert)
	return args.Error(0)
}

func TestUpdateOrderStatus_AdminCancelsOrder_RevertsStock(t *testing.T) {
	mockOrderRepo := new(MockOrderStatusRepositoryWithStock)
	mockProductRepo := new(MockProductRepositoryWithStock)

	// Setup order data
	order := oModel.OrderResponse{
		ID:     1,
		IdUser: 1,
		Status: oModel.StatusPending,
		Price:  15.0,
		OrderItems: []oModel.OrderItems{
			{ID: 1, IdOrder: 1, IdProduct: 1, Name: "Product 1", Quantity: 3},
			{ID: 2, IdOrder: 1, IdProduct: 2, Name: "Product 2", Quantity: 2},
		},
	}

	orderItems := []oModel.OrderItems{
		{ID: 1, IdOrder: 1, IdProduct: 1, Name: "Product 1", Quantity: 3},
		{ID: 2, IdOrder: 1, IdProduct: 2, Name: "Product 2", Quantity: 2},
	}

	// Setup expectations
	mockOrderRepo.On("GetOrderByID", mock.Anything, uint64(1)).Return(order, nil)
	mockOrderRepo.On("UpdateOrderStatus", mock.Anything, uint64(1), oModel.StatusCancelled).Return(nil)
	mockOrderRepo.On("CreateOrderHistory", mock.Anything, mock.Anything).Return(nil)
	mockOrderRepo.On("GetOrderItemsByOrderID", mock.Anything, uint64(1)).Return(orderItems, nil)

	// Expect stock reversion for both products
	mockProductRepo.On("RevertProductStock", mock.Anything, uint64(1), uint64(3)).Return(nil)
	mockProductRepo.On("RevertProductStock", mock.Anything, uint64(2), uint64(2)).Return(nil)

	// Create status updater with product repo
	statusUpdater := &StatusUpdaterWithStock{
		OrderRepo:   mockOrderRepo,
		ProductRepo: mockProductRepo,
	}

	// Execute
	err := statusUpdater.UpdateOrderStatusWithStockReversion(context.Background(), 1, oModel.StatusCancelled, 1, true)

	// Assert
	assert.NoError(t, err)
	mockOrderRepo.AssertExpectations(t)
	mockProductRepo.AssertExpectations(t)
}

func TestUpdateOrderStatus_ClientCancelsOrder_NoStockRevert(t *testing.T) {
	mockOrderRepo := new(MockOrderStatusRepositoryWithStock)
	mockProductRepo := new(MockProductRepositoryWithStock)

	// Setup order data
	order := oModel.OrderResponse{
		ID:     1,
		IdUser: 2, // Client user
		Status: oModel.StatusPending,
		Price:  15.0,
	}

	// Setup expectations
	mockOrderRepo.On("GetOrderByID", mock.Anything, uint64(1)).Return(order, nil)
	mockOrderRepo.On("UpdateOrderStatus", mock.Anything, uint64(1), oModel.StatusCancelled).Return(nil)
	mockOrderRepo.On("CreateOrderHistory", mock.Anything, mock.Anything).Return(nil)

	// Should NOT call stock reversion for client
	mockProductRepo.On("RevertProductStock", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create status updater with product repo
	statusUpdater := &StatusUpdaterWithStock{
		OrderRepo:   mockOrderRepo,
		ProductRepo: mockProductRepo,
	}

	// Execute
	err := statusUpdater.UpdateOrderStatusWithStockReversion(context.Background(), 1, oModel.StatusCancelled, 2, false) // Client role

	// Assert
	assert.NoError(t, err)
	mockOrderRepo.AssertExpectations(t)
	// Product repo should not be called for client cancellation
	mockProductRepo.AssertNotCalled(t, "RevertProductStock")
}

func TestUpdateOrderStatus_NonCancelledStatus_NoStockRevert(t *testing.T) {
	mockOrderRepo := new(MockOrderStatusRepositoryWithStock)
	mockProductRepo := new(MockProductRepositoryWithStock)

	// Setup order data
	order := oModel.OrderResponse{
		ID:     1,
		IdUser: 1,
		Status: oModel.StatusPending,
		Price:  15.0,
	}

	// Setup expectations
	mockOrderRepo.On("GetOrderByID", mock.Anything, uint64(1)).Return(order, nil)
	mockOrderRepo.On("UpdateOrderStatus", mock.Anything, uint64(1), oModel.StatusPreparing).Return(nil)
	mockOrderRepo.On("CreateOrderHistory", mock.Anything, mock.Anything).Return(nil)

	// Should NOT call stock reversion for non-cancelled status
	mockProductRepo.On("RevertProductStock", mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()

	// Create status updater with product repo
	statusUpdater := &StatusUpdaterWithStock{
		OrderRepo:   mockOrderRepo,
		ProductRepo: mockProductRepo,
	}

	// Execute
	err := statusUpdater.UpdateOrderStatusWithStockReversion(context.Background(), 1, oModel.StatusPreparing, 1, true) // Admin but not cancelled

	// Assert
	assert.NoError(t, err)
	mockOrderRepo.AssertExpectations(t)
	// Product repo should not be called for non-cancelled status
	mockProductRepo.AssertNotCalled(t, "RevertProductStock")
}

func TestUpdateOrderStatus_StockRevertFails_ReturnsError(t *testing.T) {
	mockOrderRepo := new(MockOrderStatusRepositoryWithStock)
	mockProductRepo := new(MockProductRepositoryWithStock)

	// Setup order data
	order := oModel.OrderResponse{
		ID:     1,
		IdUser: 1,
		Status: oModel.StatusPending,
		Price:  15.0,
	}

	orderItems := []oModel.OrderItems{
		{ID: 1, IdOrder: 1, IdProduct: 1, Name: "Product 1", Quantity: 3},
	}

	// Setup expectations
	mockOrderRepo.On("GetOrderByID", mock.Anything, uint64(1)).Return(order, nil)
	mockOrderRepo.On("UpdateOrderStatus", mock.Anything, uint64(1), oModel.StatusCancelled).Return(nil)
	mockOrderRepo.On("GetOrderItemsByOrderID", mock.Anything, uint64(1)).Return(orderItems, nil)
	// Don't expect CreateOrderHistory to be called when stock reversion fails

	// Stock reversion fails
	mockProductRepo.On("RevertProductStock", mock.Anything, uint64(1), uint64(3)).Return(errors.ErrDatabaseOperation)

	// Create status updater with product repo
	statusUpdater := &StatusUpdaterWithStock{
		OrderRepo:   mockOrderRepo,
		ProductRepo: mockProductRepo,
	}

	// Execute
	err := statusUpdater.UpdateOrderStatusWithStockReversion(context.Background(), 1, oModel.StatusCancelled, 1, true)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reverting stock for cancelled order")
	mockOrderRepo.AssertExpectations(t)
	mockProductRepo.AssertExpectations(t)
}
