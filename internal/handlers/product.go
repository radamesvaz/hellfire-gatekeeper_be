package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	"github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	productsRepository "github.com/radamesvaz/bakery-app/internal/repository/products"
	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ProductHandler struct {
	Repo *productsRepository.ProductRepository
}

// Get all products
func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	allProducts, err := h.Repo.GetAllProducts(ctx)
	if err != nil {
		http.Error(w, "Error getting products", http.StatusInternalServerError)
		return
	}

	response := []productsRepository.ProductResponse{}

	for _, product := range allProducts {
		entry := productsRepository.Marshal(&product)

		response = append(response, entry)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// GetProductByID retrieves a product by its ID
func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idProduct, err := strconv.ParseUint(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	product, err := h.Repo.GetProductByID(ctx, idProduct)
	if err != nil {
		if httpErr, ok := err.(*errors.HTTPError); ok {
			http.Error(w, httpErr.Error(), httpErr.StatusCode)
			return
		}
		http.Error(w, "internal server error", http.StatusInternalServerError)
		return
	}

	response := productsRepository.Marshal(&product)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// Create a product - Updates the table and history table
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req pModel.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	product := pModel.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Available:   req.Available,
		Status:      req.Status,
	}

	ctx := r.Context()
	newProduct, err := h.Repo.CreateProduct(ctx, product)
	if err != nil {
		http.Error(w, "Failed to create product", http.StatusInternalServerError)
		return
	}

	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get id user from context", http.StatusInternalServerError)
		return
	}

	err = h.UpdateHistoryTable(ctx, &product, newProduct.ID, idUser, pModel.ActionCreate)
	if err != nil {
		fmt.Printf("Error creating the history record for create product :%v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Product created successfully",
	})
}

// Update a product
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var req pModel.UpdateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	product := pModel.Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Available:   req.Available,
		Status:      req.Status,
	}

	ctx := r.Context()
	if err := h.Repo.UpdateProduct(ctx, product); err != nil {
		http.Error(w, "Failed to update product", http.StatusInternalServerError)
		return
	}

	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get id user from context", http.StatusInternalServerError)
		return
	}

	err = h.UpdateHistoryTable(ctx, &product, id, idUser, pModel.ActionUpdate)
	if err != nil {
		fmt.Printf("Error creating the history record for create product :%v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Product updated successfully",
	})
}

// Update a product status - soft delete
func (h *ProductHandler) UpdateProductStatus(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var req pModel.UpdateProductStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	if err := h.Repo.UpdateProductStatus(ctx, id, req.Status); err != nil {
		http.Error(w, "Failed to update product status", http.StatusInternalServerError)
		return
	}

	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed get user ID from context", http.StatusInternalServerError)
		return
	}

	product, err := h.Repo.GetProductByID(ctx, id)
	if err != nil {
		http.Error(w, errors.ErrCouldNotGetTheProduct.Error(), http.StatusInternalServerError)
	}

	err = h.UpdateHistoryTable(ctx, &product, id, idUser, pModel.ActionDelete)
	if err != nil {
		fmt.Printf("Error creating the history record for create product :%v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Product updated successfully",
	})
}

// Updates the product hsitory table
func (h *ProductHandler) UpdateHistoryTable(
	ctx context.Context,
	product *pModel.Product,
	idProduct uint64,
	idUser uint64,
	action pModel.ProductAction,
) error {
	productHistory := pModel.ProductHistory{
		IDProduct:   idProduct,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Available:   product.Available,
		Status:      product.Status,
		ModifiedBy:  idUser,
		Action:      action,
	}

	err := h.Repo.CreateProductHistory(ctx, productHistory)
	if err != nil {
		log.Printf("Warning: failed to store product history: %v", err)
		return err
	}
	return nil
}
