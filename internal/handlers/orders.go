package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"time"

	stdErrors "errors"

	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/errors"
	v "github.com/radamesvaz/bakery-app/internal/handlers/validators"
	ordersRepository "github.com/radamesvaz/bakery-app/internal/repository/orders"
	productRepo "github.com/radamesvaz/bakery-app/internal/repository/products"
	userRepo "github.com/radamesvaz/bakery-app/internal/repository/user"
	orderService "github.com/radamesvaz/bakery-app/internal/services/orders"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

type OrderHandler struct {
	Repo        *ordersRepository.OrderRepository
	UserRepo    *userRepo.UserRepository
	ProductRepo *productRepo.ProductRepository
}

// Get all orders
func (h *OrderHandler) GetAllOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	orders, err := h.Repo.GetOrders(ctx)
	if err != nil {
		http.Error(w, "Error getting orders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(orders)
}

// GetOrderByID retrieves a product by its ID
func (h *OrderHandler) GetOrderByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idOrder, err := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	order, err := h.Repo.GetOrderByID(ctx, idOrder)
	if err != nil {
		if httpErr, ok := err.(*errors.HTTPError); ok {
			http.Error(w, httpErr.Error(), httpErr.StatusCode)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(order)
}

// Create order creates a costumer order
func (h *OrderHandler) CreateOrder(w http.ResponseWriter, r *http.Request) {
	fmt.Print("Creating order")
	ctx := r.Context()

	// Decode the JSON from the body
	payload := oModel.CreateOrderPayload{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate payload fields
	if err := v.ValidateCreateOrderPayload(payload); err != nil {
		http.Error(w, "Invalid payload", http.StatusBadRequest)
		return
	}

	// Date Validations
	deliveryDate, err := time.Parse("2006-01-02", payload.DeliveryDate)
	if err != nil {
		http.Error(w, "'delivery_date' must be in YYYY-MM-DD format", http.StatusBadRequest)
		return
	}

	if deliveryDate.Before(time.Now()) {
		http.Error(w, "'delivery_date' can't be before present date", http.StatusBadRequest)
		return
	}

	orderCreator := orderService.NewCreator(*h.Repo, *h.UserRepo, *h.ProductRepo)
	err = orderCreator.CreateOrder(ctx, payload, deliveryDate)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error creating the order: '%v'", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Order created successfully",
	})
}

func (h *OrderHandler) CreateUser(ctx context.Context, user oModel.CreateOrderPayload) (id uint64, err error) {
	createUserRequest := uModel.CreateUserRequest{
		IDRole: uModel.UserRoleClient,
		Name:   user.Name,
		Email:  user.Email,
		Phone:  user.Phone,
	}

	userID, err := h.UserRepo.CreateUser(ctx, createUserRequest)
	if err != nil {
		return 0, fmt.Errorf("Error creating the user: %w", err)
	}

	return userID, nil
}

func (h *OrderHandler) getOrCreateUser(ctx context.Context, payload oModel.CreateOrderPayload) (*uModel.User, error) {
	user, err := h.UserRepo.GetUserByEmail(payload.Email)
	if err == nil {
		return &user, nil
	}

	if stdErrors.Is(err, errors.ErrUserNotFound) {
		id, err := h.CreateUser(ctx, payload)
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
