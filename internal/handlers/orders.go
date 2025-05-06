package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/errors"
	ordersRepository "github.com/radamesvaz/bakery-app/internal/repository/orders"
)

type OrderHandler struct {
	Repo *ordersRepository.OrderRepository
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
