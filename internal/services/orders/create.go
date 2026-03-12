package orders

import (
	"context"
	"database/sql"
	stdErrors "errors"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/logger"
	userRepo "github.com/radamesvaz/bakery-app/internal/repository/user"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	pModel "github.com/radamesvaz/bakery-app/model/products"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

// Interfaces for dependencies (to enable testing without DB)
type orderCreatorRepository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	CreateOrder(ctx context.Context, tx *sql.Tx, order oModel.CreateOrderRequest) (uint64, error)
	CreateOrderItems(ctx context.Context, tx *sql.Tx, items []oModel.OrderItemRequest) error
	CreateOrderHistoryTx(ctx context.Context, tx *sql.Tx, order oModel.OrderHistory) error
}

type productCreatorRepository interface {
	GetProductsByIDs(ctx context.Context, ids []uint64) ([]pModel.Product, error)
	DecrementProductStockTx(ctx context.Context, tx *sql.Tx, idProduct uint64, quantity uint64) (int64, error)
}

type Creator struct {
	OrderRepo   orderCreatorRepository
	UserRepo    userRepo.Repository
	ProductRepo productCreatorRepository
}

// TODO multi-tenant: when tenant-specific config exists, this timeout should come from the
// tenant configuration instead of a global environment variable. For now we mirror the
// logic in NewExpiredOrderCanceller so that CreateOrder and the ghost-order cron stay in sync.
func getGhostOrderTimeoutMinutes() int {
	const defaultGhostOrderTimeoutMinutes = 30
	if v := os.Getenv("GHOST_ORDER_TIMEOUT_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return defaultGhostOrderTimeoutMinutes
}

func NewCreator(
	orderRepo orderCreatorRepository,
	userRepo userRepo.Repository,
	productRepo productCreatorRepository,
) *Creator {
	return &Creator{
		UserRepo:    userRepo,
		ProductRepo: productRepo,
		OrderRepo:   orderRepo,
	}
}

// CreateOrder creates a costumer order
func (c *Creator) CreateOrder(ctx context.Context, tenantID uint64, payload oModel.CreateOrderPayload, deliveryDate time.Time) error {
	// Find user or create it if not found (scoped to tenant)
	user, err := c.GetOrCreateUser(ctx, tenantID, payload)
	if err != nil {
		return fmt.Errorf("error getting or creating user: %w", err)
	}

	// Get all of the products by their ID
	productIDs := make([]uint64, len(payload.Items))
	for i, item := range payload.Items {
		productIDs[i] = item.IdProduct
	}

	products, err := c.ProductRepo.GetProductsByIDs(ctx, productIDs)
	if err != nil {
		return fmt.Errorf("error getting products: %w", err)
	}

	if len(products) != len(productIDs) {
		return errors.ErrProductNotFound
	}

	productMap := make(map[uint64]pModel.Product)
	for _, p := range products {
		productMap[p.ID] = p
	}

	// Calculate the total price (stock is validated atomically in the tx below)
	var totalPrice float64
	for _, item := range payload.Items {
		product := productMap[item.IdProduct]
		totalPrice += product.Price * float64(item.Quantity)
	}

	// Compute per-order expiration snapshot using the current global timeout.
	// TODO multi-tenant: when tenant-specific timeout is implemented, this should use the
	// tenant's configuration instead of the global env value.
	timeoutMinutes := getGhostOrderTimeoutMinutes()
	expiresAt := time.Now().Add(time.Duration(timeoutMinutes) * time.Minute)

	tx, err := c.OrderRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Atomic stock decrement: only proceeds if stock >= quantity for each item
	for _, item := range payload.Items {
		rows, err := c.ProductRepo.DecrementProductStockTx(ctx, tx, item.IdProduct, item.Quantity)
		if err != nil {
			return fmt.Errorf("error reserving stock: %w", err)
		}
		if rows == 0 {
			return fmt.Errorf("not enough product stock")
		}
	}

	orderRequest := oModel.CreateOrderRequest{
		IdUser:       user.ID,
		DeliveryDate: deliveryDate,
		Note:         payload.Note,
		Price:        totalPrice,
		Status:       oModel.StatusPending,
		Paid:         false,
		ExpiresAt:    expiresAt,
	}

	orderID, err := c.OrderRepo.CreateOrder(ctx, tx, orderRequest)
	if err != nil {
		return fmt.Errorf("error creating order: %w", err)
	}

	orderItems := make([]oModel.OrderItemRequest, len(payload.Items))
	for i, item := range payload.Items {
		product := productMap[item.IdProduct]
		orderItems[i] = oModel.OrderItemRequest{
			IdOrder:             orderID,
			IdProduct:           item.IdProduct,
			ProductNameSnapshot: product.Name,
			UnitPriceSnapshot:   product.Price,
			Quantity:            item.Quantity,
		}
	}
	if err := c.OrderRepo.CreateOrderItems(ctx, tx, orderItems); err != nil {
		return fmt.Errorf("error creating order items: %w", err)
	}

	idUser := user.ID
	orderHistory := oModel.OrderHistory{
		IDOrder: orderID,
		IdUser:  &idUser,
		Status:  orderRequest.Status,
		Price:   orderRequest.Price,
		Note:    orderRequest.Note,
		DeliveryDate: sql.NullTime{
			Time:  deliveryDate,
			Valid: !deliveryDate.IsZero(),
		},
		Paid:       orderRequest.Paid,
		ModifiedBy: user.ID,
		Action:     oModel.ActionCreate,
	}
	if err := c.OrderRepo.CreateOrderHistoryTx(ctx, tx, orderHistory); err != nil {
		logger.Warn().Err(err).Uint64("order_id", orderID).Msg("Failed to create order history")
		// Continue and commit order+items; history is best-effort for new orders
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}
	return nil
}

func (c *Creator) GetOrCreateUser(ctx context.Context, tenantID uint64, payload oModel.CreateOrderPayload) (*uModel.User, error) {
	user, err := c.UserRepo.GetUserByEmail(tenantID, payload.Email)
	if err == nil {
		return &user, nil
	}

	if stdErrors.Is(err, errors.ErrUserNotFound) {
		id, err := c.CreateUser(ctx, tenantID, payload)
		if err != nil {
			return nil, fmt.Errorf("error creating user: %w", err)
		}
		return &uModel.User{
			ID:    id,
			Email: payload.Email,
			Name:  payload.Name,
			Phone: payload.Phone,
		}, nil
	}

	return nil, fmt.Errorf("error retrieving user: %w", err)
}

// CreateUser creates a user in the order flow: always as client (customer), scoped to the given tenant.
func (c *Creator) CreateUser(ctx context.Context, tenantID uint64, user oModel.CreateOrderPayload) (id uint64, err error) {
	createUserRequest := uModel.CreateUserRequest{
		TenantID: tenantID,
		IDRole:   uModel.UserRoleClient, // customer placing the order, not admin/staff
		Name:     user.Name,
		Email:    user.Email,
		Phone:    user.Phone,
	}

	userID, err := c.UserRepo.CreateUser(ctx, createUserRequest)
	if err != nil {
		return 0, fmt.Errorf("Error creating the user: %w", err)
	}

	return userID, nil
}
