package orders

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	internalErrors "github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	pModel "github.com/radamesvaz/bakery-app/model/products"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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

func (m *MockUserRepo) GetUserByTenantAndEmail(tenantID uint64, email string) (uModel.User, error) {
	return m.GetUserByEmail(tenantID, email)
}

func (m *MockUserRepo) ReactivateUser(ctx context.Context, tenantID, userID uint64, req uModel.ReactivateUserRequest) error {
	return nil
}

type MockProductRepo2 struct {
	Products       map[uint64]pModel.Product
	StockUpdates   map[uint64]uint64 // final stock after decrements (for assertions)
	stockSnapshot  map[uint64]uint64 // mutable copy used by DecrementProductStockTx
	LastIDs        []uint64          // IDs passed to GetProductsByIDs
	DecrementCalls int
	onAssertActive func(id uint64) // optional hook before status check (simulates mid-tx deactivation)
}

func (m *MockProductRepo2) GetProductsByIDs(ctx context.Context, tenantID uint64, ids []uint64) ([]pModel.Product, error) {
	m.LastIDs = append([]uint64(nil), ids...)
	seen := make(map[uint64]struct{}, len(ids))
	var products []pModel.Product
	for _, id := range ids {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		if product, exists := m.Products[id]; exists {
			products = append(products, product)
		}
	}
	return products, nil
}

func (m *MockProductRepo2) AssertProductActiveTx(ctx context.Context, tx *sql.Tx, tenantID, idProduct uint64) (bool, error) {
	if m.onAssertActive != nil {
		m.onAssertActive(idProduct)
	}
	product, exists := m.Products[idProduct]
	if !exists {
		return false, internalErrors.ErrProductNotFound
	}
	if product.Status != pModel.StatusActive {
		return false, internalErrors.ErrProductNotPurchasable
	}
	return product.TrackInventory, nil
}

func (m *MockProductRepo2) DecrementProductStockTx(ctx context.Context, tx *sql.Tx, tenantID, idProduct uint64, quantity uint64) (int64, error) {
	m.DecrementCalls++
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
	LastItems      []oModel.OrderItemRequest
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
	m.LastItems = append([]oModel.OrderItemRequest(nil), items...)
	return nil
}

func (m *MockOrderRepo2) CreateOrderHistoryTx(ctx context.Context, tx *sql.Tx, history oModel.OrderHistory) error {
	m.HistoryCreated = true
	return nil
}

func activeProduct(id uint64, name string, price float64, stock uint64) pModel.Product {
	return pModel.Product{
		ID:             id,
		Name:           name,
		Price:          price,
		Stock:          stock,
		Status:         pModel.StatusActive,
		TrackInventory: true,
	}
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
			1: activeProduct(1, "Pan", 2.50, 10),
			2: activeProduct(2, "Leche", 1.80, 5),
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
		Name:              "Cliente Test",
		Email:             "test@example.com",
		Phone:             "12345678",
		DeliveryDirection: "https://maps.app.goo.gl/test-direction-1",
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

func TestCreateOrder_MergesDuplicateProductLines(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectCommit()

	mockUserRepo := &MockUserRepo{ShouldCreate: false}
	mockProductRepo := &MockProductRepo2{
		Products: map[uint64]pModel.Product{
			1: activeProduct(1, "Pan", 2.50, 10),
		},
		StockUpdates: make(map[uint64]uint64),
	}
	mockOrderRepo := &MockOrderRepo2{DB: db}

	service := Creator{
		UserRepo:    mockUserRepo,
		ProductRepo: mockProductRepo,
		OrderRepo:   mockOrderRepo,
	}

	payload := oModel.CreateOrderPayload{
		Name:              "Cliente Test",
		Email:             "test@example.com",
		Phone:             "12345678",
		DeliveryDirection: "https://maps.app.goo.gl/test-direction-merge",
		Items: []oModel.CreateOrderItemInput{
			{IdProduct: 1, Quantity: 2},
			{IdProduct: 1, Quantity: 3},
		},
		Note:         "merged lines",
		DeliveryDate: "2024-12-25",
	}
	deliveryDate, _ := time.Parse("2006-01-02", payload.DeliveryDate)
	err = service.CreateOrder(context.Background(), 1, payload, deliveryDate)

	require.NoError(t, err)
	assert.Equal(t, []uint64{1}, mockProductRepo.LastIDs)
	assert.Equal(t, 1, mockProductRepo.DecrementCalls)
	assert.Equal(t, uint64(5), mockProductRepo.StockUpdates[1]) // 10 - (2+3)
	require.Len(t, mockOrderRepo.LastItems, 1)
	assert.Equal(t, uint64(5), mockOrderRepo.LastItems[0].Quantity)
	assert.Equal(t, "Pan", mockOrderRepo.LastItems[0].ProductNameSnapshot)
	assert.Equal(t, 2.50, mockOrderRepo.LastItems[0].UnitPriceSnapshot)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateOrder_SkipsStockDecrementWhenNotTrackingInventory(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectCommit()

	mockUserRepo := &MockUserRepo{ShouldCreate: false}
	product := activeProduct(1, "Servicio", 10, 0)
	product.TrackInventory = false
	mockProductRepo := &MockProductRepo2{
		Products:     map[uint64]pModel.Product{1: product},
		StockUpdates: make(map[uint64]uint64),
	}
	mockOrderRepo := &MockOrderRepo2{DB: db}

	service := Creator{
		UserRepo:    mockUserRepo,
		ProductRepo: mockProductRepo,
		OrderRepo:   mockOrderRepo,
	}

	payload := oModel.CreateOrderPayload{
		Name:              "Cliente Test",
		Email:             "test@example.com",
		Phone:             "12345678",
		DeliveryDirection: "https://maps.app.goo.gl/test-direction-no-stock",
		Items: []oModel.CreateOrderItemInput{
			{IdProduct: 1, Quantity: 100},
		},
		DeliveryDate: "2024-12-25",
	}
	deliveryDate, _ := time.Parse("2006-01-02", payload.DeliveryDate)
	err = service.CreateOrder(context.Background(), 1, payload, deliveryDate)

	require.NoError(t, err)
	assert.Equal(t, 0, mockProductRepo.DecrementCalls)
	assert.Empty(t, mockProductRepo.StockUpdates)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateOrder_RejectsInactiveProduct(t *testing.T) {
	mockUserRepo := &MockUserRepo{ShouldCreate: false}
	product := activeProduct(1, "Pan", 2.50, 10)
	product.Status = pModel.StatusInactive
	mockProductRepo := &MockProductRepo2{
		Products: map[uint64]pModel.Product{1: product},
	}
	mockOrderRepo := &MockOrderRepo2{}

	service := Creator{
		UserRepo:    mockUserRepo,
		ProductRepo: mockProductRepo,
		OrderRepo:   mockOrderRepo,
	}

	payload := oModel.CreateOrderPayload{
		Name:              "Cliente Test",
		Email:             "test@example.com",
		Phone:             "12345678",
		DeliveryDirection: "https://maps.app.goo.gl/test-direction-inactive",
		Items: []oModel.CreateOrderItemInput{
			{IdProduct: 1, Quantity: 1},
		},
		DeliveryDate: "2024-12-25",
	}
	deliveryDate, _ := time.Parse("2006-01-02", payload.DeliveryDate)
	err := service.CreateOrder(context.Background(), 1, payload, deliveryDate)

	assert.ErrorIs(t, err, internalErrors.ErrProductNotPurchasable)
	assert.False(t, mockOrderRepo.OrderCreated)
}

func TestCreateOrder_RejectsWhenProductDeactivatedInsideTx(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectRollback()

	mockUserRepo := &MockUserRepo{ShouldCreate: false}
	product := activeProduct(1, "Pan", 2.50, 10)
	mockProductRepo := &MockProductRepo2{
		Products: map[uint64]pModel.Product{1: product},
	}
	mockProductRepo.onAssertActive = func(id uint64) {
		p := mockProductRepo.Products[id]
		p.Status = pModel.StatusInactive
		mockProductRepo.Products[id] = p
	}
	mockOrderRepo := &MockOrderRepo2{DB: db}

	service := Creator{
		UserRepo:    mockUserRepo,
		ProductRepo: mockProductRepo,
		OrderRepo:   mockOrderRepo,
	}

	payload := oModel.CreateOrderPayload{
		Name:              "Cliente Test",
		Email:             "test@example.com",
		Phone:             "12345678",
		DeliveryDirection: "https://maps.app.goo.gl/test-direction-race",
		Items: []oModel.CreateOrderItemInput{
			{IdProduct: 1, Quantity: 1},
		},
		DeliveryDate: "2024-12-25",
	}
	deliveryDate, _ := time.Parse("2006-01-02", payload.DeliveryDate)
	err = service.CreateOrder(context.Background(), 1, payload, deliveryDate)

	assert.ErrorIs(t, err, internalErrors.ErrProductNotPurchasable)
	assert.False(t, mockOrderRepo.OrderCreated)
	assert.Equal(t, 0, mockProductRepo.DecrementCalls)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestCreateOrder_UsesLockedTrackInventoryWhenEnabledMidCreate(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()
	mock.ExpectBegin()
	mock.ExpectCommit()

	mockUserRepo := &MockUserRepo{ShouldCreate: false}
	product := activeProduct(1, "Pan", 2.50, 10)
	product.TrackInventory = false // pre-tx snapshot: unlimited
	mockProductRepo := &MockProductRepo2{
		Products:     map[uint64]pModel.Product{1: product},
		StockUpdates: make(map[uint64]uint64),
	}
	// Admin enables inventory tracking after GetProductsByIDs, before/at FOR UPDATE.
	mockProductRepo.onAssertActive = func(id uint64) {
		p := mockProductRepo.Products[id]
		p.TrackInventory = true
		mockProductRepo.Products[id] = p
	}
	mockOrderRepo := &MockOrderRepo2{DB: db}

	service := Creator{
		UserRepo:    mockUserRepo,
		ProductRepo: mockProductRepo,
		OrderRepo:   mockOrderRepo,
	}

	payload := oModel.CreateOrderPayload{
		Name:              "Cliente Test",
		Email:             "test@example.com",
		Phone:             "12345678",
		DeliveryDirection: "https://maps.app.goo.gl/test-direction-track-race",
		Items: []oModel.CreateOrderItemInput{
			{IdProduct: 1, Quantity: 3},
		},
		DeliveryDate: "2024-12-25",
	}
	deliveryDate, _ := time.Parse("2006-01-02", payload.DeliveryDate)
	err = service.CreateOrder(context.Background(), 1, payload, deliveryDate)

	require.NoError(t, err)
	assert.Equal(t, 1, mockProductRepo.DecrementCalls, "must decrement using locked track_inventory, not pre-tx snapshot")
	assert.Equal(t, uint64(7), mockProductRepo.StockUpdates[1])
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
			1: activeProduct(1, "Pan", 2.50, 2), // Solo hay 2 panes
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
		Name:              "Cliente Test",
		Email:             "test@example.com",
		Phone:             "12345678",
		DeliveryDirection: "https://maps.app.goo.gl/test-direction-2",
		Items: []oModel.CreateOrderItemInput{
			{IdProduct: 1, Quantity: 5}, // Intenta comprar 5 panes pero solo hay 2
		},
		Note:         "Orden que excede stock",
		DeliveryDate: "2024-12-25",
	}

	deliveryDate, _ := time.Parse("2006-01-02", payload.DeliveryDate)
	err = service.CreateOrder(ctx, 1, payload, deliveryDate)

	assert.ErrorIs(t, err, internalErrors.ErrNotEnoughProductStock)
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
			1: activeProduct(1, "Pan", 2.50, 3), // Solo hay 3 panes
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
		Name:              "Cliente 1",
		Email:             "cliente1@example.com",
		Phone:             "12345678",
		DeliveryDirection: "https://maps.app.goo.gl/test-direction-3",
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
		Name:              "Cliente 2",
		Email:             "cliente2@example.com",
		Phone:             "87654321",
		DeliveryDirection: "https://maps.app.goo.gl/test-direction-4",
		Items: []oModel.CreateOrderItemInput{
			{IdProduct: 1, Quantity: 1}, // Intenta comprar 1 pan pero no hay stock
		},
		Note:         "Segunda orden",
		DeliveryDate: "2024-12-26",
	}
	deliveryDate2, _ := time.Parse("2006-01-02", secondPayload.DeliveryDate)
	err2 := service.CreateOrder(ctx, 1, secondPayload, deliveryDate2)
	assert.ErrorIs(t, err2, internalErrors.ErrNotEnoughProductStock)
	assert.False(t, mockOrderRepo.OrderCreated)
	assert.False(t, mockOrderRepo.HistoryCreated)
	require.NoError(t, mock.ExpectationsWereMet())
}
