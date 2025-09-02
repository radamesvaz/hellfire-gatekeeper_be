package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/errors"
	v "github.com/radamesvaz/bakery-app/internal/handlers/validators"
	ordersRepository "github.com/radamesvaz/bakery-app/internal/repository/orders"
	productRepo "github.com/radamesvaz/bakery-app/internal/repository/products"
	userRepo "github.com/radamesvaz/bakery-app/internal/repository/user"
	orderService "github.com/radamesvaz/bakery-app/internal/services/orders"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

type OrderHandler struct {
	Repo        *ordersRepository.OrderRepository
	UserRepo    userRepo.Repository
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

	orderCreator := orderService.NewCreator(*h.Repo, h.UserRepo, *h.ProductRepo)
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

// UpdateOrderHistoryTable updates the order history table
func (h *OrderHandler) UpdateOrderHistoryTable(
	ctx context.Context,
	order *oModel.Order,
	idOrder uint64,
	idUser uint64,
	action oModel.OrderAction,
) error {
	orderHistory := oModel.OrderHistory{
		IDOrder: idOrder,
		IdUser:  order.IdUser,
		Status:  order.Status,
		Price:   order.Price,
		Note:    order.Note,
		DeliveryDate: sql.NullTime{
			Time:  order.DeliveryDate,
			Valid: !order.DeliveryDate.IsZero(),
		},
		ModifiedBy: idUser,
		Action:     action,
	}

	err := h.Repo.CreateOrderHistory(ctx, orderHistory)
	if err != nil {
		log.Printf("Warning: failed to store order history: %v", err)
		return err
	}
	return nil
}
