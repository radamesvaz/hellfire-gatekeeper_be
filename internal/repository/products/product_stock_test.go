package products

import (
	"context"
	"testing"

	"github.com/radamesvaz/bakery-app/internal/errors"
	pModel "github.com/radamesvaz/bakery-app/model/products"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockProductRepository for testing
type MockProductRepository struct {
	mock.Mock
}

func (m *MockProductRepository) GetProductByID(ctx context.Context, idProduct uint64) (pModel.Product, error) {
	args := m.Called(ctx, idProduct)
	return args.Get(0).(pModel.Product), args.Error(1)
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

	// Setup expectations for RevertProductStock method
	mockRepo.On("RevertProductStock", mock.Anything, uint64(1), uint64(3)).Return(nil)

	// Execute
	err := mockRepo.RevertProductStock(context.Background(), 1, 3)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRevertProductStock_ProductNotFound(t *testing.T) {
	mockRepo := new(MockProductRepository)

	// Setup expectations
	mockRepo.On("RevertProductStock", mock.Anything, uint64(999), uint64(3)).Return(errors.ErrProductNotFound)

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
	mockRepo.On("RevertProductStock", mock.Anything, uint64(1), uint64(3)).Return(errors.ErrDatabaseOperation)

	// Execute
	err := mockRepo.RevertProductStock(context.Background(), 1, 3)

	// Assert
	assert.Error(t, err)
	assert.Equal(t, errors.ErrDatabaseOperation, err)
	mockRepo.AssertExpectations(t)
}

func TestRevertProductStock_ZeroQuantity(t *testing.T) {
	mockRepo := new(MockProductRepository)

	// Setup expectations for zero quantity (should return nil without calling any methods)
	mockRepo.On("RevertProductStock", mock.Anything, uint64(1), uint64(0)).Return(nil)

	// Execute
	err := mockRepo.RevertProductStock(context.Background(), 1, 0)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}

func TestRevertProductStock_LargeQuantity(t *testing.T) {
	mockRepo := new(MockProductRepository)

	// Setup expectations
	mockRepo.On("RevertProductStock", mock.Anything, uint64(1), uint64(1000)).Return(nil)

	// Execute
	err := mockRepo.RevertProductStock(context.Background(), 1, 1000)

	// Assert
	assert.NoError(t, err)
	mockRepo.AssertExpectations(t)
}
