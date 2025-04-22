package handlers

import (
	"encoding/json"
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

func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	allProducts, err := h.Repo.GetAllProducts()
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

	product, err := h.Repo.GetProductByID(idProduct)
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

// Update a product
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req pModel.CreateProductRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Preparar la estructura actualizada
	product := pModel.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Available:   req.Available,
		Status:      req.Status,
	}

	newProduct, err := h.Repo.CreateProduct(ctx, product)
	if err != nil {
		http.Error(w, "Failed to create product", http.StatusInternalServerError)
		return
	}

	userID, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get id user from context", http.StatusInternalServerError)
		return
	}

	productHistory := pModel.ProductHistory{
		IDProduct:   newProduct.ID,
		Name:        newProduct.Name,
		Description: newProduct.Description,
		Price:       newProduct.Price,
		Available:   newProduct.Available,
		Status:      newProduct.Status,
		ModifiedBy:  userID,
		Action:      pModel.ActionCreate,
	}

	err = h.Repo.CreateProductHistory(ctx, productHistory)
	if err != nil {
		log.Printf("Warning: failed to store product history: %v", err)
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

	ctx := r.Context()

	// Preparar la estructura actualizada
	updated := pModel.Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Available:   req.Available,
		Status:      req.Status,
	}

	if err := h.Repo.UpdateProduct(ctx, updated); err != nil {
		http.Error(w, "Failed to update product", http.StatusInternalServerError)
		return
	}

	userID, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get id user from context", http.StatusInternalServerError)
		return
	}

	productHistory := pModel.ProductHistory{
		IDProduct:   id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Available:   req.Available,
		Status:      req.Status,
		ModifiedBy:  userID,
		Action:      pModel.ActionUpdate,
	}

	err = h.Repo.CreateProductHistory(ctx, productHistory)
	if err != nil {
		log.Printf("Warning: failed to store product history: %v", err)
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

	userID, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed get user ID from context", http.StatusInternalServerError)
		return
	}

	product, err := h.Repo.GetProductByID(id)
	if err != nil {
		http.Error(w, errors.ErrCouldNotGetTheProduct.Error(), http.StatusInternalServerError)
	}

	productHistory := pModel.ProductHistory{
		IDProduct:   id,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Available:   product.Available,
		Status:      product.Status,
		ModifiedBy:  userID,
		Action:      pModel.ActionUpdate,
	}

	if req.Status == pModel.StatusDeleted {
		productHistory.Action = pModel.ActionDelete
	}

	err = h.Repo.CreateProductHistory(ctx, productHistory)
	if err != nil {
		log.Printf("Warning: failed to store product history: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Product updated successfully",
	})
}
