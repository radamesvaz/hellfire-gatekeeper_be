package orders

import (
	"context"
	"errors"
	"testing"
	"time"

	internalErrors "github.com/radamesvaz/bakery-app/internal/errors"
	ordersRepository "github.com/radamesvaz/bakery-app/internal/repository/orders"
	productRepo "github.com/radamesvaz/bakery-app/internal/repository/products"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	pModel "github.com/radamesvaz/bakery-app/model/products"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
)

// Use the Creator type from the orders package

type MockUserRepo struct {
	ShouldCreate   bool
	UserWasCreated bool
	CreateUserErr  error
}

func (m *MockUserRepo) GetUserByEmail(email string) (uModel.User, error) {
	if m.ShouldCreate {
		return uModel.User{}, internalErrors.ErrUserNotFound
	}
	return uModel.User{ID: 1, Email: email}, nil
}

func (m *MockUserRepo) CreateUser(ctx context.Context, input uModel.CreateUserRequest) (uint64, error) {
	if m.CreateUserErr != nil {
		return 0, m.CreateUserErr
	}
	m.UserWasCreated = true
	return 2, nil
}

func (m *MockUserRepo) EmailExists(email string) (bool, error) {
	return false, nil
}

type MockProductRepo2 struct {
	productRepo.ProductRepository
	Products     map[uint64]pModel.Product
	StockUpdates map[uint64]uint64
}

func (m *MockProductRepo2) GetProductsByIDs(ctx context.Context, ids []uint64) ([]pModel.Product, error) {
	var products []pModel.Product
	for _, id := range ids {
		if product, exists := m.Products[id]; exists {
			products = append(products, product)
		}
	}
	return products, nil
}

func (m *MockProductRepo2) UpdateProductStock(ctx context.Context, idProduct uint64, newStock uint64) error {
	m.StockUpdates[idProduct] = newStock
	if m.Products != nil {
		p, ok := m.Products[idProduct]
		if ok {
			p.Stock = newStock
			m.Products[idProduct] = p
		}
	}
	return nil
}

// Implement other required methods with empty implementations
func (m *MockProductRepo2) GetAllProducts(ctx context.Context) ([]pModel.Product, error) {
	return nil, nil
}
func (m *MockProductRepo2) GetProductByID(ctx context.Context, idProduct uint64) (pModel.Product, error) {
	return pModel.Product{}, nil
}
func (m *MockProductRepo2) CreateProduct(ctx context.Context, product pModel.Product) (pModel.Product, error) {
	return pModel.Product{}, nil
}
func (m *MockProductRepo2) UpdateProductStatus(ctx context.Context, idProduct uint64, status pModel.ProductStatus) error {
	return nil
}
func (m *MockProductRepo2) UpdateProduct(ctx context.Context, product pModel.Product) error {
	return nil
}

type MockOrderRepo2 struct {
	ordersRepository.OrderRepository
	OrderCreated   bool
	OrderID        uint64
	HistoryCreated bool
}

func (m *MockOrderRepo2) CreateOrderOrchestrator(ctx context.Context, order oModel.CreateFullOrder) (uint64, error) {
	m.OrderCreated = true
	m.OrderID = 123
	return m.OrderID, nil
}

func (m *MockOrderRepo2) CreateOrderHistory(ctx context.Context, history oModel.OrderHistory) error {
	m.HistoryCreated = true
	return nil
}

// Implement other required methods with empty implementations
func (m *MockOrderRepo2) GetOrders(ctx context.Context) ([]oModel.OrderResponse, error) {
	return nil, nil
}
func (m *MockOrderRepo2) GetOrderByID(ctx context.Context, idOrder uint64) (*oModel.Order, error) {
	return nil, nil
}
func (m *MockOrderRepo2) UpdateOrderStatus(ctx context.Context, orderID uint64, status oModel.OrderStatus) error {
	return nil
}
func (m *MockOrderRepo2) GetOrderHistoryByOrderID(ctx context.Context, orderID uint64) ([]oModel.OrderHistory, error) {
	return nil, nil
}

func TestFindOrCreateUser_CreatesUserIfNotExists(t *testing.T) {
	mockRepo := &MockUserRepo{ShouldCreate: true}
	service := Creator{UserRepo: mockRepo}

	ctx := context.Background()
	input := oModel.CreateOrderPayload{
		Name:  "Nuevo Cliente",
		Email: "nuevo@example.com",
		Phone: "12345678",
	}

	user, err := service.GetOrCreateUser(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, uint64(2), user.ID)
	assert.True(t, mockRepo.UserWasCreated)
}

func TestFindOrCreateUser_DoesNotCreateAnUser(t *testing.T) {
	mockRepo := &MockUserRepo{ShouldCreate: false}
	service := Creator{UserRepo: mockRepo}

	ctx := context.Background()
	input := oModel.CreateOrderPayload{
		Name:  "Existente Cliente",
		Email: "existente@example.com",
		Phone: "12345678",
	}

	user, err := service.GetOrCreateUser(ctx, input)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, uint64(1), user.ID)
	assert.False(t, mockRepo.UserWasCreated)
}

func TestFindOrCreateUser_CreateUserError(t *testing.T) {
	mockRepo := &MockUserRepo{
		ShouldCreate:  true,
		CreateUserErr: errors.New("database error"),
	}
	service := Creator{UserRepo: mockRepo}

	ctx := context.Background()
	input := oModel.CreateOrderPayload{
		Name:  "Error Cliente",
		Email: "error@example.com",
		Phone: "12345678",
	}

	user, err := service.GetOrCreateUser(ctx, input)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "error creating user")
}

func TestCreateOrder_UpdatesProductStock(t *testing.T) {
	// Setup mocks
	mockUserRepo := &MockUserRepo{ShouldCreate: false}
	mockProductRepo := &MockProductRepo2{
		Products: map[uint64]pModel.Product{
			1: {ID: 1, Name: "Pan", Price: 2.50, Stock: 10},
			2: {ID: 2, Name: "Leche", Price: 1.80, Stock: 5},
		},
		StockUpdates: make(map[uint64]uint64),
	}
	mockOrderRepo := &MockOrderRepo2{}

	service := Creator{
		UserRepo:    mockUserRepo,
		ProductRepo: mockProductRepo,
		OrderRepo:   mockOrderRepo,
	}

	ctx := context.Background()
	payload := oModel.CreateOrderPayload{
		Name:  "Cliente Test",
		Email: "test@example.com",
		Phone: "12345678",
		Items: []oModel.CreateOrderItemInput{
			{IdProduct: 1, Quantity: 3}, // Pan: 10 - 3 = 7
			{IdProduct: 2, Quantity: 2}, // Leche: 5 - 2 = 3
		},
		Note:         "Orden de prueba",
		DeliveryDate: "2024-12-25",
	}

	deliveryDate, _ := time.Parse("2006-01-02", payload.DeliveryDate)
	err := service.CreateOrder(ctx, payload, deliveryDate)

	// Assertions
	assert.NoError(t, err)
	assert.True(t, mockOrderRepo.OrderCreated)
	assert.True(t, mockOrderRepo.HistoryCreated)

	// Verify stock was updated correctly
	assert.Equal(t, uint64(7), mockProductRepo.StockUpdates[1]) // Pan: 10 - 3 = 7
	assert.Equal(t, uint64(3), mockProductRepo.StockUpdates[2]) // Leche: 5 - 2 = 3
}

func TestCreateOrder_RejectsOrderWhenInsufficientStock(t *testing.T) {
	// Setup mocks
	mockUserRepo := &MockUserRepo{ShouldCreate: false}
	mockProductRepo := &MockProductRepo2{
		Products: map[uint64]pModel.Product{
			1: {ID: 1, Name: "Pan", Price: 2.50, Stock: 2}, // Solo hay 2 panes
		},
		StockUpdates: make(map[uint64]uint64),
	}
	mockOrderRepo := &MockOrderRepo2{}

	service := Creator{
		UserRepo:    mockUserRepo,
		ProductRepo: mockProductRepo,
		OrderRepo:   mockOrderRepo,
	}

	ctx := context.Background()
	payload := oModel.CreateOrderPayload{
		Name:  "Cliente Test",
		Email: "test@example.com",
		Phone: "12345678",
		Items: []oModel.CreateOrderItemInput{
			{IdProduct: 1, Quantity: 5}, // Intenta comprar 5 panes pero solo hay 2
		},
		Note:         "Orden que excede stock",
		DeliveryDate: "2024-12-25",
	}

	deliveryDate, _ := time.Parse("2006-01-02", payload.DeliveryDate)
	err := service.CreateOrder(ctx, payload, deliveryDate)

	// Assertions
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enough product stock")

	// Verify no order was created
	assert.False(t, mockOrderRepo.OrderCreated)
	assert.False(t, mockOrderRepo.HistoryCreated)

	// Verify no stock was updated
	assert.Empty(t, mockProductRepo.StockUpdates)
}

func TestCreateOrder_SecondOrderFailsAfterStockDepletion(t *testing.T) {
	// Setup mocks
	mockUserRepo := &MockUserRepo{ShouldCreate: false}
	mockProductRepo := &MockProductRepo2{
		Products: map[uint64]pModel.Product{
			1: {ID: 1, Name: "Pan", Price: 2.50, Stock: 3}, // Solo hay 3 panes
		},
		StockUpdates: make(map[uint64]uint64),
	}
	mockOrderRepo := &MockOrderRepo2{}

	service := Creator{
		UserRepo:    mockUserRepo,
		ProductRepo: mockProductRepo,
		OrderRepo:   mockOrderRepo,
	}

	ctx := context.Background()

	// First order: consumes all stock
	firstPayload := oModel.CreateOrderPayload{
		Name:  "Cliente 1",
		Email: "cliente1@example.com",
		Phone: "12345678",
		Items: []oModel.CreateOrderItemInput{
			{IdProduct: 1, Quantity: 3}, // Compra todos los 3 panes
		},
		Note:         "Primera orden",
		DeliveryDate: "2024-12-25",
	}

	deliveryDate, _ := time.Parse("2006-01-02", firstPayload.DeliveryDate)
	err := service.CreateOrder(ctx, firstPayload, deliveryDate)

	// First order should succeed
	assert.NoError(t, err)
	assert.True(t, mockOrderRepo.OrderCreated)
	assert.Equal(t, uint64(0), mockProductRepo.StockUpdates[1]) // Stock: 3 - 3 = 0

	// Reset order creation flag for second order
	mockOrderRepo.OrderCreated = false
	mockOrderRepo.HistoryCreated = false

	// Second order: should fail due to insufficient stock
	secondPayload := oModel.CreateOrderPayload{
		Name:  "Cliente 2",
		Email: "cliente2@example.com",
		Phone: "87654321",
		Items: []oModel.CreateOrderItemInput{
			{IdProduct: 1, Quantity: 1}, // Intenta comprar 1 pan pero no hay stock
		},
		Note:         "Segunda orden",
		DeliveryDate: "2024-12-26",
	}

	deliveryDate2, _ := time.Parse("2006-01-02", secondPayload.DeliveryDate)
	err2 := service.CreateOrder(ctx, secondPayload, deliveryDate2)

	// Second order should fail
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "not enough product stock")
	assert.False(t, mockOrderRepo.OrderCreated)
	assert.False(t, mockOrderRepo.HistoryCreated)
}
