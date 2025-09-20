package products

import (
	"context"
	"testing"

	"github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProductRepository for testing
type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) GetProductByID(ctx context.Context, idProduct uint64) (Product, error) {
	args := m.Called(ctx, idProduct)
	return args.Get(0).(Product), args.Error(1)
}

func (m *MockProductRepository) UpdateProductStock(ctx context.Context, idProduct uint64, newStock uint64) error {
	args := m.Called(ctx, idProduct, newStock)
	return args.Error(0)
}

func (m *MockProductRepository) RevertProductStock(ctx context.Context, idProduct uint64, quantityToRevert uint64) error {
	args := m.Called(ctx, idProduct, quantityToRevert)
	return args.Error(0)
}

func TestRevertProductStock_Success(t *testing.T) {
	mockRepo := new(MockProductRepository)

	// Setup expectations
	mockRepo.On("GetProductByID", mock.Anything, uint64(1)).Return(Product{
		ID:    1,
		Name:  "Test Product",
		Stock: 5,
	}, nil)

	mockRepo.On("UpdateProductStock", mock.Anything, uint64(1), uint64(8)).Return(nil)

	// Execute
	err := mockRepo.RevertProductStock(context.Background(), 1, 3)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRevertProductStock_ProductNotFound(t *testing.T) {
	mockRepo := new(MockProductRepository)

	// Setup expectations
	mockRepo.On("GetProductByID", mock.Anything, uint64(999)).Return(Product{}, errors.ErrProductNotFound)

	// Execute
	err := mockRepo.RevertProductStock(context.Background(), 999, 3)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, errors.ErrProductNotFound, err)
	mockRepo.AssertExpectations(t)
}

func TestRevertProductStock_UpdateStockFails(t *testing.T) {
	mockRepo := new(MockProductRepository)

	// Setup expectations
	mockRepo.On("GetProductByID", mock.Anything, uint64(1)).Return(Product{
		ID:    1,
		Name:  "Test Product",
		Stock: 5,
	}, nil)

	mockRepo.On("UpdateProductStock", mock.Anything, uint64(1), uint64(8)).Return(errors.ErrDatabaseOperation)

	// Execute
	err := mockRepo.RevertProductStock(context.Background(), 1, 3)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, errors.ErrDatabaseOperation, err)
	mockRepo.AssertExpectations(t)
}

func TestRevertProductStock_ZeroQuantity(t *testing.T) {
	mockRepo := new(MockProductRepository)

	// Execute
	err := mockRepo.RevertProductStock(context.Background(), 1, 0)

	// Assert
	assert.NoError(t, err)
	// Should not call any methods for zero quantity
	mockRepo.AssertExpectations(t)
}

func TestRevertProductStock_LargeQuantity(t *testing.T) {
	mockRepo := new(MockProductRepository)

	// Setup expectations
	mockRepo.On("GetProductByID", mock.Anything, uint64(1)).Return(Product{
		ID:    1,
		Name:  "Test Product",
		Stock: 5,
	}, nil)

	mockRepo.On("UpdateProductStock", mock.Anything, uint64(1), uint64(1005)).Return(nil)

	// Execute
	err := mockRepo.RevertProductStock(context.Background(), 1, 1000)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
