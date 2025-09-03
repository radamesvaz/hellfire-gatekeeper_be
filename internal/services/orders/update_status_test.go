package orders

import (
	"context"
	"testing"
	"time"

	"github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOrderRepository is a mock implementation of the OrderStatusRepository interface
type MockOrderRepository struct {
	mock.Mock
}

func (m *MockOrderRepository) GetOrderByID(ctx context.Context, id uint64) (oModel.OrderResponse, error) {
	args := m.Called(ctx, id)
	return args.Get(0).(oModel.OrderResponse), args.Error(1)
}

func (m *MockOrderRepository) UpdateOrderStatus(ctx context.Context, orderID uint64, status oModel.OrderStatus) error {
	args := m.Called(ctx, orderID, status)
	return args.Error(0)
}

func (m *MockOrderRepository) CreateOrderHistory(ctx context.Context, order oModel.OrderHistory) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func TestUpdateOrderStatus_Success(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	statusUpdater := NewStatusUpdater(mockRepo)
	ctx := context.Background()
	orderID := uint64(1)
	newStatus := oModel.StatusPreparing

	// Mock order retrieval (before update)
	existingOrder := oModel.OrderResponse{
		ID:           orderID,
		IdUser:       1,
		Status:       oModel.StatusPending,
		Price:        50.0,
		Note:         "Test order",
		DeliveryDate: time.Now().AddDate(0, 0, 7),
		CreatedOn:    time.Now(),
		User:         "Test User",
		OrderItems:   []oModel.OrderItems{},
	}
	mockRepo.On("GetOrderByID", ctx, orderID).Return(existingOrder, nil).Once()

	// Mock status update
	mockRepo.On("UpdateOrderStatus", ctx, orderID, newStatus).Return(nil)

	// Mock history creation
	mockRepo.On("CreateOrderHistory", ctx, mock.AnythingOfType("model.OrderHistory")).Return(nil)

	// Mock order retrieval (after update) - should return updated status
	updatedOrder := existingOrder
	updatedOrder.Status = newStatus
	mockRepo.On("GetOrderByID", ctx, orderID).Return(updatedOrder, nil).Once()

	// Act
	err := statusUpdater.UpdateOrderStatus(ctx, orderID, newStatus, 1)

	// Assert
	assert.NoError(t, err)

	// Verify the order was actually updated with the new status
	actualOrder, err := statusUpdater.OrderRepo.GetOrderByID(ctx, orderID)
	assert.NoError(t, err)
	assert.Equal(t, newStatus, actualOrder.Status)

	mockRepo.AssertExpectations(t)
}

func TestUpdateOrderStatus_OrderNotFound(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	statusUpdater := NewStatusUpdater(mockRepo)
	ctx := context.Background()
	orderID := uint64(999)
	newStatus := oModel.StatusPreparing

	// Mock order not found
	mockRepo.On("GetOrderByID", ctx, orderID).Return(oModel.OrderResponse{}, errors.ErrOrderNotFound)

	// Act
	err := statusUpdater.UpdateOrderStatus(ctx, orderID, newStatus, 1)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, errors.ErrOrderNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateOrderStatus_InvalidTransition(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	statusUpdater := NewStatusUpdater(mockRepo)
	ctx := context.Background()
	orderID := uint64(1)
	newStatus := oModel.OrderStatus("invalid_status") // This status doesn't exist in our model

	// Mock order retrieval - order is pending
	existingOrder := oModel.OrderResponse{
		ID:           orderID,
		IdUser:       1,
		Status:       oModel.StatusPending,
		Price:        50.0,
		Note:         "Test order",
		DeliveryDate: time.Now().AddDate(0, 0, 7),
		CreatedOn:    time.Now(),
		User:         "Test User",
		OrderItems:   []oModel.OrderItems{},
	}
	mockRepo.On("GetOrderByID", ctx, orderID).Return(existingOrder, nil)

	// Act
	err := statusUpdater.UpdateOrderStatus(ctx, orderID, newStatus, 1)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, errors.ErrInvalidStatusTransition, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateOrderStatus_OrderAlreadyCancelled(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	statusUpdater := NewStatusUpdater(mockRepo)
	ctx := context.Background()
	orderID := uint64(1)
	newStatus := oModel.StatusPreparing

	// Mock order retrieval - order is already cancelled
	existingOrder := oModel.OrderResponse{
		ID:           orderID,
		IdUser:       1,
		Status:       oModel.StatusCancelled,
		Price:        50.0,
		Note:         "Test order",
		DeliveryDate: time.Now().AddDate(0, 0, 7),
		CreatedOn:    time.Now(),
		User:         "Test User",
		OrderItems:   []oModel.OrderItems{},
	}
	mockRepo.On("GetOrderByID", ctx, orderID).Return(existingOrder, nil)

	// Act
	err := statusUpdater.UpdateOrderStatus(ctx, orderID, newStatus, 1)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, errors.ErrOrderAlreadyCancelled, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateOrderStatus_OrderAlreadyDelivered(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	statusUpdater := NewStatusUpdater(mockRepo)
	ctx := context.Background()
	orderID := uint64(1)
	newStatus := oModel.StatusPreparing

	// Mock order retrieval - order is already delivered
	existingOrder := oModel.OrderResponse{
		ID:           orderID,
		IdUser:       1,
		Status:       oModel.StatusDelivered,
		Price:        50.0,
		Note:         "Test order",
		DeliveryDate: time.Now().AddDate(0, 0, 7),
		CreatedOn:    time.Now(),
		User:         "Test User",
		OrderItems:   []oModel.OrderItems{},
	}
	mockRepo.On("GetOrderByID", ctx, orderID).Return(existingOrder, nil)

	// Act
	err := statusUpdater.UpdateOrderStatus(ctx, orderID, newStatus, 1)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, errors.ErrOrderAlreadyDelivered, err)
	mockRepo.AssertExpectations(t)
}

func TestUpdateOrderStatus_ValidTransitions(t *testing.T) {
	testCases := []struct {
		name        string
		current     oModel.OrderStatus
		new         oModel.OrderStatus
		shouldError bool
	}{
		{"Pending to Preparing", oModel.StatusPending, oModel.StatusPreparing, false},
		{"Pending to Cancelled", oModel.StatusPending, oModel.StatusCancelled, false},
		{"Preparing to Ready", oModel.StatusPreparing, oModel.StatusReady, false},
		{"Preparing to Cancelled", oModel.StatusPreparing, oModel.StatusCancelled, false},
		{"Ready to Delivered", oModel.StatusReady, oModel.StatusDelivered, false},
		{"Ready to Cancelled", oModel.StatusReady, oModel.StatusCancelled, false},
		{"Pending to Delivered", oModel.StatusPending, oModel.StatusDelivered, true},
		{"Preparing to Delivered", oModel.StatusPreparing, oModel.StatusDelivered, true},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Arrange - create fresh mock for each test case
			mockRepo := new(MockOrderRepository)
			statusUpdater := NewStatusUpdater(mockRepo)
			ctx := context.Background()
			orderID := uint64(1)

			// Mock order retrieval
			existingOrder := oModel.OrderResponse{
				ID:           orderID,
				IdUser:       1,
				Status:       tc.current,
				Price:        50.0,
				Note:         "Test order",
				DeliveryDate: time.Now().AddDate(0, 0, 7),
				CreatedOn:    time.Now(),
				User:         "Test User",
				OrderItems:   []oModel.OrderItems{},
			}
			mockRepo.On("GetOrderByID", ctx, orderID).Return(existingOrder, nil)

			if !tc.shouldError {
				// Mock successful operations
				mockRepo.On("UpdateOrderStatus", ctx, orderID, tc.new).Return(nil)
				mockRepo.On("CreateOrderHistory", ctx, mock.AnythingOfType("model.OrderHistory")).Return(nil)
			}

			// Act
			err := statusUpdater.UpdateOrderStatus(ctx, orderID, tc.new, 1)

			// Assert
			if tc.shouldError {
				assert.Error(t, err)
				assert.Equal(t, errors.ErrInvalidStatusTransition, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify mock expectations
			mockRepo.AssertExpectations(t)
		})
	}
}

func TestUpdateOrderStatus_UpdateStatusFails(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	statusUpdater := NewStatusUpdater(mockRepo)
	ctx := context.Background()
	orderID := uint64(1)
	newStatus := oModel.StatusPreparing

	// Mock order retrieval
	existingOrder := oModel.OrderResponse{
		ID:           orderID,
		IdUser:       1,
		Status:       oModel.StatusPending,
		Price:        50.0,
		Note:         "Test order",
		DeliveryDate: time.Now().AddDate(0, 0, 7),
		CreatedOn:    time.Now(),
		User:         "Test User",
		OrderItems:   []oModel.OrderItems{},
	}
	mockRepo.On("GetOrderByID", ctx, orderID).Return(existingOrder, nil)

	// Mock status update failure
	mockRepo.On("UpdateOrderStatus", ctx, orderID, newStatus).Return(errors.ErrDatabaseOperation)

	// Act
	err := statusUpdater.UpdateOrderStatus(ctx, orderID, newStatus, 1)

	// Assert
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error updating order status")
	mockRepo.AssertExpectations(t)
}

func TestUpdateOrderStatus_HistoryCreationFails(t *testing.T) {
	// Arrange
	mockRepo := new(MockOrderRepository)
	statusUpdater := NewStatusUpdater(mockRepo)
	ctx := context.Background()
	orderID := uint64(1)
	newStatus := oModel.StatusPreparing

	// Mock order retrieval
	existingOrder := oModel.OrderResponse{
		ID:           orderID,
		IdUser:       1,
		Status:       oModel.StatusPending,
		Price:        50.0,
		Note:         "Test order",
		DeliveryDate: time.Now().AddDate(0, 0, 7),
		CreatedOn:    time.Now(),
		User:         "Test User",
		OrderItems:   []oModel.OrderItems{},
	}
	mockRepo.On("GetOrderByID", ctx, orderID).Return(existingOrder, nil)

	// Mock successful status update
	mockRepo.On("UpdateOrderStatus", ctx, orderID, newStatus).Return(nil)

	// Mock history creation failure
	mockRepo.On("CreateOrderHistory", ctx, mock.AnythingOfType("model.OrderHistory")).Return(errors.ErrCreatingOrderHistory)

	// Act
	err := statusUpdater.UpdateOrderStatus(ctx, orderID, newStatus, 1)

	// Assert
	// History creation failure should not fail the status update
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
