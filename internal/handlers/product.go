package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/handlers/validators"
	"github.com/radamesvaz/bakery-app/internal/logger"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	"github.com/radamesvaz/bakery-app/internal/pagination"
	productsRepository "github.com/radamesvaz/bakery-app/internal/repository/products"
	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ProductHandler struct {
	Repo *productsRepository.ProductRepository
}

type productsListResponse struct {
	Items      []pModel.Product `json:"items"`
	NextCursor *string          `json:"next_cursor"`
}

// GetAllProducts lists products with cursor pagination (query: limit, cursor).
func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		tenantID = 1
	}

	limit, err := validators.ParseListLimit(r.URL.Query().Get("limit"))
	if err != nil {
		var he *appErrors.HTTPError
		if errors.As(err, &he) {
			http.Error(w, he.Error(), he.StatusCode)
		} else {
			http.Error(w, err.Error(), http.StatusBadRequest)
		}
		return
	}

	var afterID *uint64
	if c := r.URL.Query().Get("cursor"); c != "" {
		id, err := pagination.DecodeIDCursor(c)
		if err != nil {
			http.Error(w, "Invalid cursor", http.StatusBadRequest)
			return
		}
		afterID = &id
	}

	page, err := h.Repo.ListProductsPage(ctx, tenantID, limit, afterID)
	if err != nil {
		http.Error(w, "Failed to get products", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(productsListResponse{Items: page.Items, NextCursor: page.NextCursor})
}

// GetProductByID - Get a product by ID
func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		tenantID = 1
	}
	product, err := h.Repo.GetProductByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, appErrors.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get product", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

// CreateProduct - Create a product (JSON only)
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req pModel.CreateProductRequest

	// Parse JSON request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Create product
	product := pModel.Product{
		TenantID:    0, // will be overridden from context
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Available:   req.Available,
		Stock:       req.Stock,
		Status:      req.Status,
		ImageURLs:   []string{}, // Empty initially, images added via separate endpoint
	}

	ctx := r.Context()
	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		tenantID = 1
	}
	product.TenantID = tenantID

	// Create product
	newProduct, err := h.Repo.CreateProduct(ctx, tenantID, product)
	if err != nil {
		http.Error(w, "Failed to create product", http.StatusInternalServerError)
		return
	}

	// Update history
	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get id user from context", http.StatusInternalServerError)
		return
	}

	err = h.UpdateHistoryTable(ctx, &product, newProduct.ID, idUser, pModel.ActionCreate)
	if err != nil {
		logger.Warn().Err(err).
			Uint64("product_id", newProduct.ID).
			Uint64("user_id", idUser).
			Msg("Error creating the history record for create product")
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"message":    "Product created successfully",
		"product_id": newProduct.ID,
		"image_urls": newProduct.ImageURLs,
	}
	json.NewEncoder(w).Encode(response)
}

// UpdateProduct - Update a product (JSON only)
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var req pModel.UpdateProductRequest

	// Parse JSON request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ThumbnailURL != "" && !validators.ThumbnailURLInImageURLs(req.ThumbnailURL, req.ImageURLs) {
		http.Error(w, "thumbnail_url must exist in image_urls", http.StatusBadRequest)
		return
	}

	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		tenantID = 1
	}
	product := pModel.Product{
		ID:           id,
		TenantID:     tenantID,
		Name:         req.Name,
		Description:  req.Description,
		Price:        req.Price,
		Available:    req.Available,
		Stock:        req.Stock,
		Status:       req.Status,
		ImageURLs:    req.ImageURLs, // Images managed via separate endpoint
		ThumbnailURL: req.ThumbnailURL,
	}

	// Update product basic fields
	if err := h.Repo.UpdateProduct(ctx, tenantID, product); err != nil {
		http.Error(w, "Failed to update product", http.StatusInternalServerError)
		return
	}

	// Update history
	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get id user from context", http.StatusInternalServerError)
		return
	}

	err = h.UpdateHistoryTable(ctx, &product, id, idUser, pModel.ActionUpdate)
	if err != nil {
		logger.Warn().Err(err).
			Uint64("product_id", id).
			Uint64("user_id", idUser).
			Msg("Error creating the history record for update product")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Product updated successfully",
	})
}

// UpdateProductThumbnail - Update the thumbnail image for a product
func (h *ProductHandler) UpdateProductThumbnail(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var req struct {
		ThumbnailURL string `json:"thumbnail_url"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.ThumbnailURL == "" {
		http.Error(w, "thumbnail_url is required", http.StatusBadRequest)
		return
	}

	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		tenantID = 1
	}
	product, err := h.Repo.GetProductByID(ctx, tenantID, id)
	if err != nil {
		if errors.Is(err, appErrors.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get product", http.StatusInternalServerError)
		return
	}

	if !validators.ThumbnailURLInImageURLs(req.ThumbnailURL, product.ImageURLs) {
		http.Error(w, "thumbnail_url must exist in image_urls", http.StatusBadRequest)
		return
	}

	if err := h.Repo.UpdateProductThumbnail(ctx, tenantID, id, req.ThumbnailURL); err != nil {
		if errors.Is(err, appErrors.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update product thumbnail", http.StatusInternalServerError)
		return
	}

	product.ThumbnailURL = req.ThumbnailURL
	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get id user from context", http.StatusInternalServerError)
		return
	}

	err = h.UpdateHistoryTable(ctx, &product, id, idUser, pModel.ActionUpdate)
	if err != nil {
		logger.Warn().Err(err).
			Uint64("product_id", id).
			Uint64("user_id", idUser).
			Msg("Error creating the history record for update product thumbnail")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":       "Product thumbnail updated successfully",
		"product_id":    id,
		"thumbnail_url": req.ThumbnailURL,
	})
}

// UpdateProductStatus - Update product status (delete/inactive)
func (h *ProductHandler) UpdateProductStatus(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	tenantID, err := middleware.GetTenantIDFromContext(ctx)
	if err != nil {
		tenantID = 1
	}
	if err := h.Repo.UpdateProductStatus(ctx, tenantID, id, pModel.ProductStatus(req.Status)); err != nil {
		if errors.Is(err, appErrors.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to update product status", http.StatusInternalServerError)
		return
	}

	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get id user from context", http.StatusInternalServerError)
		return
	}

	product := pModel.Product{ID: id, Status: pModel.ProductStatus(req.Status)}
	err = h.UpdateHistoryTable(ctx, &product, id, idUser, pModel.ActionUpdate)
	if err != nil {
		logger.Warn().Err(err).
			Uint64("product_id", id).
			Uint64("user_id", idUser).
			Msg("Error creating the history record for update product status")
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Product status updated successfully",
	})
}

// UpdateHistoryTable - Update the history table
func (h *ProductHandler) UpdateHistoryTable(ctx context.Context, product *pModel.Product, idProduct uint64, idUser uint64, action pModel.ProductAction) error {
	history := pModel.ProductHistory{
		TenantID:     product.TenantID,
		IDProduct:    idProduct,
		Name:         product.Name,
		Description:  product.Description,
		Price:        product.Price,
		Available:    product.Available,
		Stock:        product.Stock,
		Status:       product.Status,
		ImageURLs:    product.ImageURLs,
		ThumbnailURL: product.ThumbnailURL,
		ModifiedBy:   idUser,
		Action:       action,
	}

	return h.Repo.CreateProductHistory(ctx, history.TenantID, history)
}
