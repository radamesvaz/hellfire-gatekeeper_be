package orders

import (
	"context"
	"database/sql"
	stdErrors "errors"
	"fmt"
	"time"

	"github.com/radamesvaz/bakery-app/internal/errors"
	ordersRepository "github.com/radamesvaz/bakery-app/internal/repository/orders"
	productRepo "github.com/radamesvaz/bakery-app/internal/repository/products"
	userRepo "github.com/radamesvaz/bakery-app/internal/repository/user"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	pModel "github.com/radamesvaz/bakery-app/model/products"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

// Interfaces for dependencies (to enable testing without DB)
type orderCreatorRepository interface {
	CreateOrderOrchestrator(ctx context.Context, order oModel.CreateFullOrder) (uint64, error)
	CreateOrderHistory(ctx context.Context, order oModel.OrderHistory) error
}

type productCreatorRepository interface {
	GetProductsByIDs(ctx context.Context, ids []uint64) ([]pModel.Product, error)
	UpdateProductStock(ctx context.Context, idProduct uint64, newStock uint64) error
}

type Creator struct {
	OrderRepo   orderCreatorRepository
	UserRepo    userRepo.Repository
	ProductRepo productCreatorRepository
}

func NewCreator(
	orderRepo ordersRepository.OrderRepository,
	userRepo userRepo.Repository,
	productRepo productRepo.ProductRepository,
) *Creator {
	return &Creator{
		UserRepo:    userRepo,
		ProductRepo: &productRepo,
		OrderRepo:   &orderRepo,
	}
}

// CreateOrder creates a costumer order
func (c *Creator) CreateOrder(ctx context.Context, payload oModel.CreateOrderPayload, deliveryDate time.Time) error {
	// Find user or create it if not found
	user, err := c.GetOrCreateUser(ctx, payload)
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

	// Validate the stock of said products
	productMap := make(map[uint64]pModel.Product)
	for _, p := range products {
		productMap[p.ID] = p
	}

	for _, item := range payload.Items {
		p := productMap[item.IdProduct]
		if p.Stock < item.Quantity {
			return fmt.Errorf("not enough product stock")
		}
	}

	// Calculate the total price
	var totalPrice float64

	for _, item := range payload.Items {
		product := productMap[item.IdProduct]
		totalPrice += product.Price * float64(item.Quantity)
	}

	// Create the order for the orchestrator
	order := oModel.CreateFullOrder{
		IdUser:       user.ID,
		DeliveryDate: deliveryDate,
		Note:         payload.Note,
		Price:        totalPrice,
		Status:       oModel.StatusPending,
		OrderItems:   mapItemsToInternalModel(payload.Items),
	}

	orderID, err := c.OrderRepo.CreateOrderOrchestrator(ctx, order)
	if err != nil {
		return fmt.Errorf("error creating order: %w", err)
	}

	// Create order history record
	orderHistory := oModel.OrderHistory{
		IDOrder: orderID,
		IdUser:  order.IdUser,
		Status:  order.Status,
		Price:   order.Price,
		Note:    order.Note,
		DeliveryDate: sql.NullTime{
			Time:  order.DeliveryDate,
			Valid: !order.DeliveryDate.IsZero(),
		},
		ModifiedBy: order.IdUser, // The user who created the order
		Action:     oModel.ActionCreate,
	}

	err = c.OrderRepo.CreateOrderHistory(ctx, orderHistory)
	if err != nil {
		// Log the error but don't fail the order creation
		fmt.Printf("Warning: failed to create order history: %v", err)
	}

	// Update product stock after successful order creation
	for _, item := range payload.Items {
		product := productMap[item.IdProduct]
		newStock := product.Stock - item.Quantity

		err := c.ProductRepo.UpdateProductStock(ctx, product.ID, newStock)
		if err != nil {
			// Log the error but don't fail the order creation
			fmt.Printf("Warning: failed to update product stock for product %d: %v", product.ID, err)
		}
	}

	return nil
}

func (c *Creator) GetOrCreateUser(ctx context.Context, payload oModel.CreateOrderPayload) (*uModel.User, error) {
	user, err := c.UserRepo.GetUserByEmail(payload.Email)
	if err == nil {
		return &user, nil
	}

	if stdErrors.Is(err, errors.ErrUserNotFound) {
		id, err := c.CreateUser(ctx, payload)
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

func (c *Creator) CreateUser(ctx context.Context, user oModel.CreateOrderPayload) (id uint64, err error) {
	createUserRequest := uModel.CreateUserRequest{
		IDRole: uModel.UserRoleClient,
		Name:   user.Name,
		Email:  user.Email,
		Phone:  user.Phone,
	}

	userID, err := c.UserRepo.CreateUser(ctx, createUserRequest)
	if err != nil {
		return 0, fmt.Errorf("Error creating the user: %w", err)
	}

	return userID, nil
}

func mapItemsToInternalModel(input []oModel.CreateOrderItemInput) []oModel.OrderItemRequest {
	items := make([]oModel.OrderItemRequest, len(input))
	for i, item := range input {
		items[i] = oModel.OrderItemRequest{
			IdProduct: item.IdProduct,
			Quantity:  item.Quantity,
		}
	}
	return items
}
