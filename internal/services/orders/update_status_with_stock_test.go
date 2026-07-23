package orders

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockOrderStatusRepositoryWithStock for testing stock reversion
type MockOrderStatusRepositoryWithStock struct {
	mock.Mock
	DB *sql.DB
}

func (m *MockOrderStatusRepositoryWithStock) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if m.DB == nil {
		args := m.Called(ctx)
		if args.Get(0) == nil {
			return nil, args.Error(1)
		}
		return args.Get(0).(*sql.Tx), args.Error(1)
	}
	return m.DB.BeginTx(ctx, nil)
}

func (m *MockOrderStatusRepositoryWithStock) GetOrderByID(ctx context.Context, tenantID, id uint64) (oModel.OrderResponse, error) {
	args := m.Called(ctx, tenantID, id)
	return args.Get(0).(oModel.OrderResponse), args.Error(1)
}

func (m *MockOrderStatusRepositoryWithStock) UpdateOrderStatus(ctx context.Context, tenantID, orderID uint64, status oModel.OrderStatus, cancellationReason *string) error {
	args := m.Called(ctx, tenantID, orderID, status, cancellationReason)
	return args.Error(0)
}

func (m *MockOrderStatusRepositoryWithStock) UpdateOrderStatusTx(ctx context.Context, tx *sql.Tx, tenantID, orderID uint64, status oModel.OrderStatus, cancellationReason *string) error {
	args := m.Called(ctx, tx, tenantID, orderID, status, cancellationReason)
	return args.Error(0)
}

func (m *MockOrderStatusRepositoryWithStock) CreateOrderHistory(ctx context.Context, order oModel.OrderHistory) error {
	args := m.Called(ctx, order)
	return args.Error(0)
}

func (m *MockOrderStatusRepositoryWithStock) CreateOrderHistoryTx(ctx context.Context, tx *sql.Tx, order oModel.OrderHistory) error {
	args := m.Called(ctx, tx, order)
	return args.Error(0)
}

func (m *MockOrderStatusRepositoryWithStock) GetOrderItemsByOrderID(ctx context.Context, tenantID, orderID uint64) ([]oModel.OrderItems, error) {
	args := m.Called(ctx, tenantID, orderID)
	return args.Get(0).([]oModel.OrderItems), args.Error(1)
}

func (m *MockOrderStatusRepositoryWithStock) GetOrderItemsByOrderIDTx(ctx context.Context, tx *sql.Tx, tenantID, orderID uint64) ([]oModel.OrderItems, error) {
	args := m.Called(ctx, tx, tenantID, orderID)
	return args.Get(0).([]oModel.OrderItems), args.Error(1)
}

// MockProductRepositoryWithStock for testing stock reversion
type MockProductRepositoryWithStock struct {
	mock.Mock
}

func (m *MockProductRepositoryWithStock) RevertProductStock(ctx context.Context, tenantID, idProduct uint64, quantityToRevert uint64) error {
	args := m.Called(ctx, tenantID, idProduct, quantityToRevert)
	return args.Error(0)
}

func (m *MockProductRepositoryWithStock) RevertProductStockTx(ctx context.Context, tx *sql.Tx, tenantID, idProduct uint64, quantityToRevert uint64) error {
	args := m.Called(ctx, tx, tenantID, idProduct, quantityToRevert)
	return args.Error(0)
}

func TestValidateStatusTransition_RejectsFromCancelledOrExpired(t *testing.T) {
	s := &StatusUpdaterWithStock{}

	err := s.validateStatusTransition(oModel.StatusCancelled, oModel.StatusCancelled)
	assert.ErrorIs(t, err, errors.ErrOrderAlreadyCancelled)

	err = s.validateStatusTransition(oModel.StatusCancelled, oModel.StatusPreparing)
	assert.ErrorIs(t, err, errors.ErrInvalidStatusTransition)

	err = s.validateStatusTransition(oModel.StatusExpired, oModel.StatusPreparing)
	assert.ErrorIs(t, err, errors.ErrInvalidStatusTransition)

	err = s.validateStatusTransition(oModel.StatusPending, oModel.StatusPreparing)
	assert.NoError(t, err)
}

func TestUpdateOrderStatus_AdminCancelsOrder_RevertsStock(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlMock.ExpectBegin()
	sqlMock.ExpectCommit()

	mockOrderRepo := &MockOrderStatusRepositoryWithStock{DB: db}
	mockProductRepo := new(MockProductRepositoryWithStock)

	order := oModel.OrderResponse{
		ID:       1,
		TenantID: 1,
		IdUser:   1,
		Status:   oModel.StatusPending,
		Price:    15.0,
		OrderItems: []oModel.OrderItems{
			{ID: 1, IdOrder: 1, IdProduct: 1, Name: "Product 1", Quantity: 3},
			{ID: 2, IdOrder: 1, IdProduct: 2, Name: "Product 2", Quantity: 2},
		},
	}

	orderItems := []oModel.OrderItems{
		{ID: 1, IdOrder: 1, IdProduct: 1, Name: "Product 1", Quantity: 3},
		{ID: 2, IdOrder: 1, IdProduct: 2, Name: "Product 2", Quantity: 2},
	}

	const tenantID = uint64(1)
	mockOrderRepo.On("GetOrderByID", mock.Anything, tenantID, uint64(1)).Return(order, nil)
	mockOrderRepo.On("UpdateOrderStatusTx", mock.Anything, mock.Anything, tenantID, uint64(1), oModel.StatusCancelled, (*string)(nil)).Return(nil)
	mockOrderRepo.On("CreateOrderHistoryTx", mock.Anything, mock.Anything, mock.Anything).Return(nil)
	mockOrderRepo.On("GetOrderItemsByOrderIDTx", mock.Anything, mock.Anything, tenantID, uint64(1)).Return(orderItems, nil)

	mockProductRepo.On("RevertProductStockTx", mock.Anything, mock.Anything, tenantID, uint64(1), uint64(3)).Return(nil)
	mockProductRepo.On("RevertProductStockTx", mock.Anything, mock.Anything, tenantID, uint64(2), uint64(2)).Return(nil)

	statusUpdater := &StatusUpdaterWithStock{
		OrderRepo:   mockOrderRepo,
		ProductRepo: mockProductRepo,
	}

	err = statusUpdater.UpdateOrderStatusWithStockReversion(context.Background(), tenantID, 1, oModel.StatusCancelled, 1, true, nil, nil)

	assert.NoError(t, err)
	mockOrderRepo.AssertExpectations(t)
	mockProductRepo.AssertExpectations(t)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestUpdateOrderStatus_ClientCancelsOrder_NoStockRevert(t *testing.T) {
	mockOrderRepo := new(MockOrderStatusRepositoryWithStock)
	mockProductRepo := new(MockProductRepositoryWithStock)

	order := oModel.OrderResponse{
		ID:       1,
		TenantID: 1,
		IdUser:   2,
		Status:   oModel.StatusPending,
		Price:    15.0,
	}

	const tenantID = uint64(1)
	mockOrderRepo.On("GetOrderByID", mock.Anything, tenantID, uint64(1)).Return(order, nil)
	mockOrderRepo.On("UpdateOrderStatus", mock.Anything, tenantID, uint64(1), oModel.StatusCancelled, (*string)(nil)).Return(nil)
	mockOrderRepo.On("CreateOrderHistory", mock.Anything, mock.Anything).Return(nil)

	statusUpdater := &StatusUpdaterWithStock{
		OrderRepo:   mockOrderRepo,
		ProductRepo: mockProductRepo,
	}

	err := statusUpdater.UpdateOrderStatusWithStockReversion(context.Background(), tenantID, 1, oModel.StatusCancelled, 2, false, nil, nil)

	assert.NoError(t, err)
	mockOrderRepo.AssertExpectations(t)
	mockProductRepo.AssertNotCalled(t, "RevertProductStockTx")
	mockProductRepo.AssertNotCalled(t, "RevertProductStock")
}

func TestUpdateOrderStatus_NonCancelledStatus_NoStockRevert(t *testing.T) {
	mockOrderRepo := new(MockOrderStatusRepositoryWithStock)
	mockProductRepo := new(MockProductRepositoryWithStock)

	order := oModel.OrderResponse{
		ID:       1,
		TenantID: 1,
		IdUser:   1,
		Status:   oModel.StatusPending,
		Price:    15.0,
	}

	const tenantID = uint64(1)
	mockOrderRepo.On("GetOrderByID", mock.Anything, tenantID, uint64(1)).Return(order, nil)
	mockOrderRepo.On("UpdateOrderStatus", mock.Anything, tenantID, uint64(1), oModel.StatusPreparing, (*string)(nil)).Return(nil)
	mockOrderRepo.On("CreateOrderHistory", mock.Anything, mock.Anything).Return(nil)

	statusUpdater := &StatusUpdaterWithStock{
		OrderRepo:   mockOrderRepo,
		ProductRepo: mockProductRepo,
	}

	err := statusUpdater.UpdateOrderStatusWithStockReversion(context.Background(), tenantID, 1, oModel.StatusPreparing, 1, true, nil, nil)

	assert.NoError(t, err)
	mockOrderRepo.AssertExpectations(t)
	mockProductRepo.AssertNotCalled(t, "RevertProductStockTx")
}

func TestUpdateOrderStatus_StockRevertFails_ReturnsError(t *testing.T) {
	db, sqlMock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	sqlMock.ExpectBegin()
	sqlMock.ExpectRollback()

	mockOrderRepo := &MockOrderStatusRepositoryWithStock{DB: db}
	mockProductRepo := new(MockProductRepositoryWithStock)

	order := oModel.OrderResponse{
		ID:       1,
		TenantID: 1,
		IdUser:   1,
		Status:   oModel.StatusPending,
		Price:    15.0,
	}

	orderItems := []oModel.OrderItems{
		{ID: 1, IdOrder: 1, IdProduct: 1, Name: "Product 1", Quantity: 3},
	}

	const tenantID = uint64(1)
	mockOrderRepo.On("GetOrderByID", mock.Anything, tenantID, uint64(1)).Return(order, nil)
	mockOrderRepo.On("UpdateOrderStatusTx", mock.Anything, mock.Anything, tenantID, uint64(1), oModel.StatusCancelled, (*string)(nil)).Return(nil)
	mockOrderRepo.On("GetOrderItemsByOrderIDTx", mock.Anything, mock.Anything, tenantID, uint64(1)).Return(orderItems, nil)

	mockProductRepo.On("RevertProductStockTx", mock.Anything, mock.Anything, tenantID, uint64(1), uint64(3)).Return(errors.ErrDatabaseOperation)

	statusUpdater := &StatusUpdaterWithStock{
		OrderRepo:   mockOrderRepo,
		ProductRepo: mockProductRepo,
	}

	err = statusUpdater.UpdateOrderStatusWithStockReversion(context.Background(), tenantID, 1, oModel.StatusCancelled, 1, true, nil, nil)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "error reverting stock for cancelled order")
	mockOrderRepo.AssertExpectations(t)
	mockProductRepo.AssertExpectations(t)
	require.NoError(t, sqlMock.ExpectationsWereMet())
}

func TestUpdateOrderStatus_AlreadyCancelled_ReturnsError(t *testing.T) {
	mockOrderRepo := new(MockOrderStatusRepositoryWithStock)
	mockProductRepo := new(MockProductRepositoryWithStock)

	order := oModel.OrderResponse{
		ID:     1,
		Status: oModel.StatusCancelled,
	}

	const tenantID = uint64(1)
	mockOrderRepo.On("GetOrderByID", mock.Anything, tenantID, uint64(1)).Return(order, nil)

	statusUpdater := &StatusUpdaterWithStock{
		OrderRepo:   mockOrderRepo,
		ProductRepo: mockProductRepo,
	}

	err := statusUpdater.UpdateOrderStatusWithStockReversion(context.Background(), tenantID, 1, oModel.StatusCancelled, 1, true, nil, nil)
	assert.ErrorIs(t, err, errors.ErrOrderAlreadyCancelled)
	mockProductRepo.AssertNotCalled(t, "RevertProductStockTx")
}

func TestUpdateOrderStatus_PaidOverrideUsedInHistory(t *testing.T) {
	mockOrderRepo := new(MockOrderStatusRepositoryWithStock)
	mockProductRepo := new(MockProductRepositoryWithStock)

	order := oModel.OrderResponse{
		ID:     1,
		TenantID: 1,
		IdUser: 1,
		Status: oModel.StatusPending,
		Price:  15.0,
		Paid:   false,
	}

	const tenantID = uint64(1)
	paidTrue := true
	mockOrderRepo.On("GetOrderByID", mock.Anything, tenantID, uint64(1)).Return(order, nil)
	mockOrderRepo.On("UpdateOrderStatus", mock.Anything, tenantID, uint64(1), oModel.StatusPreparing, (*string)(nil)).Return(nil)
	mockOrderRepo.On("CreateOrderHistory", mock.Anything, mock.MatchedBy(func(h oModel.OrderHistory) bool {
		return h.Status == oModel.StatusPreparing && h.Paid == true && h.TenantID == tenantID
	})).Return(nil)

	statusUpdater := &StatusUpdaterWithStock{
		OrderRepo:   mockOrderRepo,
		ProductRepo: mockProductRepo,
	}

	err := statusUpdater.UpdateOrderStatusWithStockReversion(context.Background(), tenantID, 1, oModel.StatusPreparing, 1, true, nil, &paidTrue)
	require.NoError(t, err)
	mockOrderRepo.AssertExpectations(t)
}
