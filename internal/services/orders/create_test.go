package orders

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	internalErrors "github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	pModel "github.com/radamesvaz/bakery-app/model/products"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/DATA-DOG/go-sqlmock"
)

// Use the Creator type from the orders package

type MockUserRepo struct {
	ShouldCreate   bool
	UserWasCreated bool
	CreateUserErr  error
}

func (m *MockUserRepo) GetUserByEmail(tenantID uint64, email string) (uModel.User, error) {
	if m.ShouldCreate {
		return uModel.User{}, internalErrors.ErrUserNotFound
	}
	return uModel.User{ID: 1, TenantID: tenantID, Email: email}, nil
}

func (m *MockUserRepo) CreateUser(ctx context.Context, input uModel.CreateUserRequest) (uint64, error) {
	if m.CreateUserErr != nil {
		return 0, m.CreateUserErr
	}
	m.UserWasCreated = true
	return 2, nil
}

func (m *MockUserRepo) EmailExists(tenantID uint64, email string) (bool, error) {
	return false, nil
}

type MockProductRepo2 struct {
	Products       map[uint64]pModel.Product
	StockUpdates   map[uint64]uint64 // final stock after decrements (for assertions)
	stockSnapshot  map[uint64]uint64 // mutable copy used by DecrementProductStockTx
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

func (m *MockProductRepo2) DecrementProductStockTx(ctx context.Context, tx *sql.Tx, idProduct uint64, quantity uint64) (int64, error) {
	if m.stockSnapshot == nil {
		m.stockSnapshot = make(map[uint64]uint64)
		for id, p := range m.Products {
			m.stockSnapshot[id] = p.Stock
		}
	}
	current := m.stockSnapshot[idProduct]
	if current < quantity {
		return 0, nil
	}
	m.stockSnapshot[idProduct] = current - quantity
	if m.StockUpdates != nil {
		m.StockUpdates[idProduct] = current - quantity
	}
	return 1, nil
}

type MockOrderRepo2 struct {
	DB             *sql.DB // set in tests to get a real *sql.Tx from sqlmock
	OrderCreated   bool
	OrderID        uint64
	HistoryCreated bool
}

func (m *MockOrderRepo2) BeginTx(ctx context.Context) (*sql.Tx, error) {
	if m.DB == nil {
		return nil, errors.New("no DB set for mock")
	}
	return m.DB.BeginTx(ctx, nil)
}

func (m *MockOrderRepo2) CreateOrder(ctx context.Context, tx *sql.Tx, order oModel.CreateOrderRequest) (uint64, error) {
	m.OrderCreated = true
	m.OrderID = 123
	return m.OrderID, nil
}

func (m *MockOrderRepo2) CreateOrderItems(ctx context.Context, tx *sql.Tx, tenantID uint64, items []oModel.OrderItemRequest) error {
	return nil
}

func (m *MockOrderRepo2) CreateOrderHistoryTx(ctx context.Context, tx *sql.Tx, history oModel.OrderHistory) error {
	m.HistoryCreated = true
	return nil
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

	user, err := service.GetOrCreateUser(ctx, 1, input)

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

	user, err := service.GetOrCreateUser(ctx, 1, input)

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

	user, err := service.GetOrCreateUser(ctx, 1, input)

	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "error creating user")
}

func TestCreateOrder_UpdatesProductStock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectCommit()

	mockUserRepo := &MockUserRepo{ShouldCreate: false}
	mockProductRepo := &MockProductRepo2{
		Products: map[uint64]pModel.Product{
			1: {ID: 1, Name: "Pan", Price: 2.50, Stock: 10},
			2: {ID: 2, Name: "Leche", Price: 1.80, Stock: 5},
		},
		StockUpdates: make(map[uint64]uint64),
	}
	mockOrderRepo := &MockOrderRepo2{DB: db}

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
	err = service.CreateOrder(ctx, 1, payload, deliveryDate)

	assert.NoError(t, err)
	assert.True(t, mockOrderRepo.OrderCreated)
	assert.True(t, mockOrderRepo.HistoryCreated)
	assert.Equal(t, uint64(7), mockProductRepo.StockUpdates[1])
	assert.Equal(t, uint64(3), mockProductRepo.StockUpdates[2])
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateOrder_RejectsOrderWhenInsufficientStock(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectRollback()

	mockUserRepo := &MockUserRepo{ShouldCreate: false}
	mockProductRepo := &MockProductRepo2{
		Products: map[uint64]pModel.Product{
			1: {ID: 1, Name: "Pan", Price: 2.50, Stock: 2}, // Solo hay 2 panes
		},
		StockUpdates: make(map[uint64]uint64),
	}
	mockOrderRepo := &MockOrderRepo2{DB: db}

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
	err = service.CreateOrder(ctx, 1, payload, deliveryDate)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enough product stock")
	assert.False(t, mockOrderRepo.OrderCreated)
	assert.False(t, mockOrderRepo.HistoryCreated)
	assert.Empty(t, mockProductRepo.StockUpdates)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateOrder_SecondOrderFailsAfterStockDepletion(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mockUserRepo := &MockUserRepo{ShouldCreate: false}
	mockProductRepo := &MockProductRepo2{
		Products: map[uint64]pModel.Product{
			1: {ID: 1, Name: "Pan", Price: 2.50, Stock: 3}, // Solo hay 3 panes
		},
		StockUpdates: make(map[uint64]uint64),
	}
	mockOrderRepo := &MockOrderRepo2{DB: db}

	service := Creator{
		UserRepo:    mockUserRepo,
		ProductRepo: mockProductRepo,
		OrderRepo:   mockOrderRepo,
	}

	ctx := context.Background()

	// First order: consumes all stock
	mock.ExpectBegin()
	mock.ExpectCommit()
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
	err = service.CreateOrder(ctx, 1, firstPayload, deliveryDate)
	assert.NoError(t, err)
	assert.True(t, mockOrderRepo.OrderCreated)
	assert.Equal(t, uint64(0), mockProductRepo.StockUpdates[1])

	// Second order: should fail due to insufficient stock
	mockOrderRepo.OrderCreated = false
	mockOrderRepo.HistoryCreated = false
	mock.ExpectBegin()
	mock.ExpectRollback()
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
	err2 := service.CreateOrder(ctx, 1, secondPayload, deliveryDate2)
	assert.Error(t, err2)
	assert.Contains(t, err2.Error(), "not enough product stock")
	assert.False(t, mockOrderRepo.OrderCreated)
	assert.False(t, mockOrderRepo.HistoryCreated)
	require.NoError(t, mock.ExpectationsWereMet())
}
