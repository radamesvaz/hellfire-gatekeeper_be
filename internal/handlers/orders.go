package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/errors"
	v "github.com/radamesvaz/bakery-app/internal/handlers/validators"
	ordersRepository "github.com/radamesvaz/bakery-app/internal/repository/orders"
	userRepo "github.com/radamesvaz/bakery-app/internal/repository/user"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

type OrderHandler struct {
	Repo     *ordersRepository.OrderRepository
	UserRepo *userRepo.UserRepository
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

	// Find user or create it if not found
	user, err := h.UserRepo.GetUserByEmail(payload.Email)
	if err != nil {
		if err == errors.NewNotFound(errors.ErrUserNotFound) {
			idUser, err := h.CreateUser(ctx, payload)
			if err != nil {
				http.Error(w, "Error creating the user", http.StatusInternalServerError)
				return
			}
			user.ID = idUser //find a cleaner way
		} else {
			http.Error(w, "Error getting the user", http.StatusInternalServerError)
			return
		}
	}

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
