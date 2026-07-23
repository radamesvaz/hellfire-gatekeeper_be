package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/handlers/validators"
	"github.com/radamesvaz/bakery-app/internal/logger"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	"github.com/radamesvaz/bakery-app/internal/pagination"
	productsRepository "github.com/radamesvaz/bakery-app/internal/repository/products"
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ProductHandler struct {
	Repo         *productsRepository.ProductRepository
	ImageService *imagesService.Service
}

type productsListResponse struct {
	Items      []pModel.Product `json:"items"`
	NextCursor *string          `json:"next_cursor"`
}

func writeRepoError(w http.ResponseWriter, err error, fallbackMsg string) {
	var he *appErrors.HTTPError
	if errors.As(err, &he) {
		http.Error(w, he.Error(), he.StatusCode)
		return
	}
	http.Error(w, fallbackMsg, http.StatusInternalServerError)
}

func requireTenantID(w http.ResponseWriter, r *http.Request) (uint64, bool) {
	tenantID, err := middleware.GetTenantIDFromContext(r.Context())
	if err != nil {
		http.Error(w, "tenant context required", http.StatusBadRequest)
		return 0, false
	}
	return tenantID, true
}

// GetAllProducts lists active products only (public catalog / legacy GET /products).
// Query: limit, cursor, optional q (case-insensitive name contains; min 2 chars).
func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	h.listProducts(w, r, true)
}

// GetAllProductsAdmin lists all product statuses for admin management (GET /auth/products).
func (h *ProductHandler) GetAllProductsAdmin(w http.ResponseWriter, r *http.Request) {
	h.listProducts(w, r, false)
}

func (h *ProductHandler) listProducts(w http.ResponseWriter, r *http.Request, activeOnly bool) {
	ctx := r.Context()
	tenantID, ok := requireTenantID(w, r)
	if !ok {
		return
	}

	limit, err := validators.ParseListLimit(r.URL.Query().Get("limit"))
	if err != nil {
		writeRepoError(w, err, err.Error())
		return
	}

	namePrefixRaw, err := validators.ParseProductSearchQuery(r.URL.Query().Get("q"))
	if err != nil {
		writeRepoError(w, err, err.Error())
		return
	}
	var nameLikePattern *string
	if namePrefixRaw != nil {
		pat := validators.ProductNameContainsLikePattern(*namePrefixRaw)
		nameLikePattern = &pat
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

	page, err := h.Repo.ListProductsPage(ctx, tenantID, limit, afterID, nameLikePattern, activeOnly)
	if err != nil {
		http.Error(w, "Failed to get products", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(productsListResponse{Items: page.Items, NextCursor: page.NextCursor})
}

// GetProductByID returns an active product by ID (public catalog).
func (h *ProductHandler) GetProductByID(w http.ResponseWriter, r *http.Request) {
	h.getProductByID(w, r, true)
}

// GetProductByIDAdmin returns a product by ID regardless of status (GET /auth/products/{id}).
func (h *ProductHandler) GetProductByIDAdmin(w http.ResponseWriter, r *http.Request) {
	h.getProductByID(w, r, false)
}

func (h *ProductHandler) getProductByID(w http.ResponseWriter, r *http.Request, activeOnly bool) {
	vars := mux.Vars(r)
	idStr := vars["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	ctx := r.Context()
	tenantID, ok := requireTenantID(w, r)
	if !ok {
		return
	}
	product, err := h.Repo.GetProductByID(ctx, tenantID, id, activeOnly)
	if err != nil {
		if errors.Is(err, appErrors.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		writeRepoError(w, err, "Failed to get product")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(product)
}

// CreateProduct - Create a product (JSON only)
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	var req pModel.CreateProductRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if strings.TrimSpace(req.Description) == "" {
		http.Error(w, "Description is required", http.StatusBadRequest)
		return
	}
	if req.Price < 0 {
		http.Error(w, "Price must be greater than or equal to 0", http.StatusBadRequest)
		return
	}

	status := req.Status
	if status == "" {
		status = pModel.StatusActive
	}
	trackInventory := true
	if req.TrackInventory != nil {
		trackInventory = *req.TrackInventory
	}

	product := pModel.Product{
		TenantID:       0, // will be overridden from context
		Name:           req.Name,
		Description:    req.Description,
		Price:          req.Price,
		TrackInventory: trackInventory,
		Stock:          req.Stock,
		Status:         status,
		ImageURLs:      []string{}, // Empty initially, images added via separate endpoint
	}

	ctx := r.Context()
	tenantID, ok := requireTenantID(w, r)
	if !ok {
		return
	}
	product.TenantID = tenantID

	// TODO: persist product creator on products (e.g. created_by_user_id FK → users).
	// Today it is only recorded in products_history (modified_by + action=create), not on the product row or API.
	newProduct, err := h.Repo.CreateProduct(ctx, tenantID, product)
	if err != nil {
		writeRepoError(w, err, "Failed to create product")
		return
	}

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

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
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
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	tenantID, ok := requireTenantID(w, r)
	if !ok {
		return
	}

	existing, err := h.Repo.GetProductByID(ctx, tenantID, id, false)
	if err != nil {
		if errors.Is(err, appErrors.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		writeRepoError(w, err, "Failed to get product")
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}
	if req.Price < 0 {
		http.Error(w, "Price must be greater than or equal to 0", http.StatusBadRequest)
		return
	}

	trackInventory := existing.TrackInventory
	if req.TrackInventory != nil {
		trackInventory = *req.TrackInventory
	}
	status := req.Status
	if status == "" {
		status = existing.Status
	}

	product := pModel.Product{
		ID:             id,
		TenantID:       tenantID,
		Name:           req.Name,
		Description:    req.Description,
		Price:          req.Price,
		TrackInventory: trackInventory,
		Stock:          req.Stock,
		Status:         status,
		ImageURLs:      existing.ImageURLs,
		ThumbnailURL:   existing.ThumbnailURL,
	}

	if err := h.Repo.UpdateProduct(ctx, tenantID, product); err != nil {
		writeRepoError(w, err, "Failed to update product")
		return
	}

	h.cleanupImagesIfDeleted(id, status)

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

	tenantID, ok := requireTenantID(w, r)
	if !ok {
		return
	}
	product, err := h.Repo.GetProductByID(ctx, tenantID, id, false)
	if err != nil {
		if errors.Is(err, appErrors.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		writeRepoError(w, err, "Failed to get product")
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
		writeRepoError(w, err, "Failed to update product thumbnail")
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
	tenantID, ok := requireTenantID(w, r)
	if !ok {
		return
	}

	product, err := h.Repo.GetProductByID(ctx, tenantID, id, false)
	if err != nil {
		if errors.Is(err, appErrors.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		writeRepoError(w, err, "Failed to get product")
		return
	}

	newStatus := pModel.ProductStatus(req.Status)
	if err := h.Repo.UpdateProductStatus(ctx, tenantID, id, newStatus); err != nil {
		if errors.Is(err, appErrors.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		writeRepoError(w, err, "Failed to update product status")
		return
	}

	h.cleanupImagesIfDeleted(id, newStatus)

	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get id user from context", http.StatusInternalServerError)
		return
	}

	product.Status = newStatus
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

// cleanupImagesIfDeleted best-effort removes local product image files after soft-delete.
func (h *ProductHandler) cleanupImagesIfDeleted(productID uint64, status pModel.ProductStatus) {
	if status != pModel.StatusDeleted || h.ImageService == nil {
		return
	}
	if delErr := h.ImageService.DeleteProductImages(productID); delErr != nil {
		logger.Warn().Err(delErr).
			Uint64("product_id", productID).
			Msg("Best-effort DeleteProductImages after status=deleted failed")
	}
}

// UpdateHistoryTable - Update the history table
func (h *ProductHandler) UpdateHistoryTable(ctx context.Context, product *pModel.Product, idProduct uint64, idUser uint64, action pModel.ProductAction) error {
	history := pModel.ProductHistory{
		TenantID:       product.TenantID,
		IDProduct:      idProduct,
		Name:           product.Name,
		Description:    product.Description,
		Price:          product.Price,
		TrackInventory: product.TrackInventory,
		Stock:          product.Stock,
		Status:         product.Status,
		ImageURLs:      product.ImageURLs,
		ThumbnailURL:   product.ThumbnailURL,
		ModifiedBy:     idUser,
		Action:         action,
	}

	return h.Repo.CreateProductHistory(ctx, history.TenantID, history)
}
