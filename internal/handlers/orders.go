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
	"github.com/radamesvaz/bakery-app/internal/middleware"
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
		Paid:       order.Paid,
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

// UpdateOrderStatus updates the status of an order
func (h *OrderHandler) UpdateOrderStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idOrder, err := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	// Decode the JSON from the body
	var payload struct {
		Status oModel.OrderStatus `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate status
	if payload.Status == "" {
		http.Error(w, "Status is required", http.StatusBadRequest)
		return
	}

	// Validate status enum values
	validStatuses := []oModel.OrderStatus{
		oModel.StatusPreparing,
		oModel.StatusReady,
		oModel.StatusDelivered,
		oModel.StatusCancelled,
	}

	isValidStatus := false
	for _, validStatus := range validStatuses {
		if payload.Status == validStatus {
			isValidStatus = true
			break
		}
	}

	if !isValidStatus {
		http.Error(w, "Invalid status value", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get user ID from JWT token
	userID, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	// Create status updater service
	statusUpdater := orderService.NewStatusUpdater(h.Repo)

	// Update the order status
	err = statusUpdater.UpdateOrderStatus(ctx, idOrder, payload.Status, userID)
	if err != nil {
		if httpErr, ok := err.(*errors.HTTPError); ok {
			http.Error(w, httpErr.Error(), httpErr.StatusCode)
			return
		}
		http.Error(w, fmt.Sprintf("Error updating order status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Order status updated successfully",
	})
}

// UpdateOrder updates an order (status and/or paid status)
func (h *OrderHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idOrder, err := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	// Decode the JSON from the body
	var payload struct {
		Status *oModel.OrderStatus `json:"status,omitempty"`
		Paid   *bool               `json:"paid,omitempty"`
	}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate that at least one field is provided
	if payload.Status == nil && payload.Paid == nil {
		http.Error(w, "At least one field (status or paid) must be provided", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Get user ID from JWT token
	userID, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	// Get the current order to track changes
	currentOrder, err := h.Repo.GetOrderByID(ctx, idOrder)
	if err != nil {
		if httpErr, ok := err.(*errors.HTTPError); ok {
			http.Error(w, httpErr.Error(), httpErr.StatusCode)
			return
		}
		http.Error(w, "Error getting order", http.StatusInternalServerError)
		return
	}

	// Update status if provided
	if payload.Status != nil {
		// Validate status enum values
		validStatuses := []oModel.OrderStatus{
			oModel.StatusPreparing,
			oModel.StatusReady,
			oModel.StatusDelivered,
			oModel.StatusCancelled,
		}

		isValidStatus := false
		for _, validStatus := range validStatuses {
			if *payload.Status == validStatus {
				isValidStatus = true
				break
			}
		}

		if !isValidStatus {
			http.Error(w, "Invalid status value", http.StatusBadRequest)
			return
		}

		// Create status updater service
		statusUpdater := orderService.NewStatusUpdater(h.Repo)

		// Update the order status
		err = statusUpdater.UpdateOrderStatus(ctx, idOrder, *payload.Status, userID)
		if err != nil {
			if httpErr, ok := err.(*errors.HTTPError); ok {
				http.Error(w, httpErr.Error(), httpErr.StatusCode)
				return
			}
			http.Error(w, fmt.Sprintf("Error updating order status: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Update paid status if provided
	if payload.Paid != nil {
		err = h.Repo.UpdateOrderPaidStatus(ctx, idOrder, *payload.Paid)
		if err != nil {
			if httpErr, ok := err.(*errors.HTTPError); ok {
				http.Error(w, httpErr.Error(), httpErr.StatusCode)
				return
			}
			http.Error(w, fmt.Sprintf("Error updating order paid status: %v", err), http.StatusInternalServerError)
			return
		}
	}

	// Create order history record for the update
	orderModel := &oModel.Order{
		ID:           currentOrder.ID,
		IdUser:       currentOrder.IdUser,
		Status:       currentOrder.Status,
		Price:        currentOrder.Price,
		Note:         currentOrder.Note,
		DeliveryDate: currentOrder.DeliveryDate,
		Paid:         currentOrder.Paid,
	}

	// Update the model with new values
	if payload.Status != nil {
		orderModel.Status = *payload.Status
	}
	if payload.Paid != nil {
		orderModel.Paid = *payload.Paid
	}

	err = h.UpdateOrderHistoryTable(ctx, orderModel, idOrder, userID, oModel.ActionUpdate)
	if err != nil {
		log.Printf("Warning: failed to create order history: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Order updated successfully",
	})
}
