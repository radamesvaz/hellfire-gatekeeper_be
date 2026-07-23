package handlers

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	v "github.com/radamesvaz/bakery-app/internal/handlers/validators"
	"github.com/radamesvaz/bakery-app/internal/logger"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	"github.com/radamesvaz/bakery-app/internal/pagination"
	ordersRepository "github.com/radamesvaz/bakery-app/internal/repository/orders"
	productRepo "github.com/radamesvaz/bakery-app/internal/repository/products"
	tenantRepository "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	userRepo "github.com/radamesvaz/bakery-app/internal/repository/user"
	orderService "github.com/radamesvaz/bakery-app/internal/services/orders"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

type OrderHandler struct {
	Repo        *ordersRepository.OrderRepository
	UserRepo    userRepo.Repository
	ProductRepo *productRepo.ProductRepository
	TenantRepo  *tenantRepository.Repository
}

type ordersListResponse struct {
	Items      []oModel.OrderResponse `json:"items"`
	NextCursor *string                `json:"next_cursor"`
}

// GetAllOrders lists orders with cursor pagination (query: limit, cursor, optional id_user) and filters ignore_status, status.
// id_user: positive integer filters orders for that user within the tenant; omit for all users.
func (h *OrderHandler) GetAllOrders(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		http.Error(w, "tenant context required", http.StatusBadRequest)
		return
	}

	ignoreStatus := r.URL.Query().Get("ignore_status") == "true"
	statusFilter := r.URL.Query().Get("status")
	searchQuery, err := v.NormalizeAndValidateOrderSearchQuery(r.URL.Query().Get("q"))
	if err != nil {
		var he *appErrors.HTTPError
		if errors.As(err, &he) {
			http.Error(w, he.Error(), he.StatusCode)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	if err := v.ValidateOrderListStatusFilter(statusFilter); err != nil {
		var he *appErrors.HTTPError
		if errors.As(err, &he) {
			http.Error(w, he.Error(), he.StatusCode)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	var statusFilterPtr *string
	if statusFilter != "" {
		statusFilterPtr = &statusFilter
	}

	limit, err := v.ParseListLimit(r.URL.Query().Get("limit"))
	if err != nil {
		var he *appErrors.HTTPError
		if errors.As(err, &he) {
			http.Error(w, he.Error(), he.StatusCode)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	var after *pagination.OrderKeyset
	if c := r.URL.Query().Get("cursor"); c != "" {
		k, err := pagination.DecodeOrderCursor(c)
		if err != nil {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
			return
		}
		after = &k
	}

	var filterUserID *uint64
	if s := r.URL.Query().Get("id_user"); s != "" {
		uid, err := strconv.ParseUint(s, 10, 64)
		if err != nil || uid == 0 {
			http.Error(w, "Invalid id_user", http.StatusBadRequest)
			return
		}
		filterUserID = &uid
	}

	var searchQueryPtr *string
	if searchQuery != "" {
		searchQueryPtr = &searchQuery
	}

	page, err := h.Repo.ListOrdersWithFiltersPage(ctx, tenantID, ignoreStatus, statusFilterPtr, limit, after, filterUserID, searchQueryPtr)
	if err != nil {
		http.Error(w, "Error getting orders", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ordersListResponse{Items: page.Items, NextCursor: page.NextCursor})
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
	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		http.Error(w, "tenant context required", http.StatusBadRequest)
		return
	}
	order, err := h.Repo.GetOrderByID(ctx, tenantID, idOrder)
	if err != nil {
		if httpErr, ok := err.(*appErrors.HTTPError); ok {
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
	logger.Debug().Msg("Creating order")
	ctx := r.Context()

	// Decode the JSON from the body
	payload := oModel.CreateOrderPayload{}
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate payload fields
	if err := v.ValidateCreateOrderPayload(payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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

	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		http.Error(w, "tenant context required", http.StatusBadRequest)
		return
	}
	var tenantCfgRepo orderService.TenantConfigRepository = nil
	if h.TenantRepo != nil {
		tenantCfgRepo = h.TenantRepo
	}
	orderCreator := orderService.NewCreator(h.Repo, h.UserRepo, h.ProductRepo, tenantCfgRepo)
	err = orderCreator.CreateOrder(ctx, tenantID, payload, deliveryDate)
	if err != nil {
		switch {
		case errors.Is(err, appErrors.ErrProductNotFound):
			http.Error(w, err.Error(), http.StatusNotFound)
		case errors.Is(err, appErrors.ErrProductNotPurchasable):
			http.Error(w, err.Error(), http.StatusConflict)
		case errors.Is(err, appErrors.ErrNotEnoughProductStock),
			strings.Contains(err.Error(), "not enough product stock"):
			http.Error(w, err.Error(), http.StatusConflict)
		default:
			http.Error(w, fmt.Sprintf("Error creating the order: '%v'", err), http.StatusInternalServerError)
		}
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
	var orderHistoryIdUser *uint64
	if order.IdUser != 0 {
		orderHistoryIdUser = &order.IdUser
	}
	orderHistory := oModel.OrderHistory{
		TenantID:          order.TenantID,
		IDOrder:           idOrder,
		IdUser:            orderHistoryIdUser,
		Status:            order.Status,
		Price:             order.Price,
		Note:              order.Note,
		DeliveryDirection: order.DeliveryDirection,
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
		logger.Warn().Err(err).
			Uint64("order_id", idOrder).
			Msg("Failed to store order history")
		return err
	}
	return nil
}

// UpdateOrder updates an order (status and/or paid status) - UNIFIED FUNCTION
func (h *OrderHandler) UpdateOrder(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idOrder, err := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid order ID", http.StatusBadRequest)
		return
	}

	// Decode the JSON from the body
	var payload struct {
		Status             *oModel.OrderStatus `json:"status,omitempty"`
		Paid               *bool               `json:"paid,omitempty"`
		CancellationReason *string             `json:"cancellation_reason,omitempty"`
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

	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		http.Error(w, "tenant context required", http.StatusBadRequest)
		return
	}

	// Get user ID from JWT token
	userID, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Unauthorized: invalid token", http.StatusUnauthorized)
		return
	}

	// Get the current order to track changes
	currentOrder, err := h.Repo.GetOrderByID(ctx, tenantID, idOrder)
	if err != nil {
		if httpErr, ok := err.(*appErrors.HTTPError); ok {
			http.Error(w, httpErr.Error(), httpErr.StatusCode)
			return
		}
		http.Error(w, "Error getting order", http.StatusInternalServerError)
		return
	}

	// Apply paid first so a combined status+paid PATCH can record history with the final paid flag.
	if payload.Paid != nil {
		err = h.Repo.UpdateOrderPaidStatus(ctx, tenantID, idOrder, *payload.Paid)
		if err != nil {
			if httpErr, ok := err.(*appErrors.HTTPError); ok {
				http.Error(w, httpErr.Error(), httpErr.StatusCode)
				return
			}
			http.Error(w, fmt.Sprintf("Error updating order paid status: %v", err), http.StatusInternalServerError)
			return
		}
	}

	statusUpdated := false
	if payload.Status != nil {
		// Validate status enum values
		validStatuses := []oModel.OrderStatus{
			oModel.StatusPreparing,
			oModel.StatusReady,
			oModel.StatusDelivered,
			oModel.StatusCancelled,
			oModel.StatusDeleted,
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

		// Get user role from context
		userRole, err := middleware.GetUserRoleFromContext(ctx)
		if err != nil {
			http.Error(w, "Unauthorized: invalid token role", http.StatusUnauthorized)
			return
		}

		// Admin and superadmin cancel restore inventory; clients do not.
		isAdmin := middleware.IsAdminRole(userRole)

		// Create status updater service with stock reversion capability
		statusUpdater := orderService.NewStatusUpdaterWithStock(h.Repo, h.ProductRepo)

		// Status updater persists history (with paidOverride when paid was also patched).
		err = statusUpdater.UpdateOrderStatusWithStockReversion(ctx, tenantID, idOrder, *payload.Status, userID, isAdmin, payload.CancellationReason, payload.Paid)
		if err != nil {
			if httpErr, ok := err.(*appErrors.HTTPError); ok {
				http.Error(w, httpErr.Error(), httpErr.StatusCode)
				return
			}
			http.Error(w, fmt.Sprintf("Error updating order status: %v", err), http.StatusInternalServerError)
			return
		}
		statusUpdated = true
	}

	// Paid-only updates: write history here (status path already recorded history).
	if !statusUpdated {
		orderModel := &oModel.Order{
			ID:                currentOrder.ID,
			TenantID:          tenantID,
			IdUser:            currentOrder.IdUser,
			Status:            currentOrder.Status,
			Price:             currentOrder.Price,
			Note:              currentOrder.Note,
			DeliveryDirection: currentOrder.DeliveryDirection,
			DeliveryDate:      currentOrder.DeliveryDate,
			Paid:              currentOrder.Paid,
		}
		if payload.Paid != nil {
			orderModel.Paid = *payload.Paid
		}

		err = h.UpdateOrderHistoryTable(ctx, orderModel, idOrder, userID, oModel.ActionUpdate)
		if err != nil {
			logger.Warn().Err(err).
				Uint64("order_id", idOrder).
				Msg("Failed to create order history")
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Order updated successfully",
	})
}
