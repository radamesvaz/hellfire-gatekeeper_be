package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/mux"
	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	productsRepository "github.com/radamesvaz/bakery-app/internal/repository/products"
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ProductHandler struct {
	Repo         *productsRepository.ProductRepository
	ImageService *imagesService.Service
}

// GetAllProducts - Get all products
func (h *ProductHandler) GetAllProducts(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	products, err := h.Repo.GetAllProducts(ctx)
	if err != nil {
		http.Error(w, "Failed to get products", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(products)
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
	product, err := h.Repo.GetProductByID(ctx, id)
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

// CreateProduct - Create a product (handles both JSON and multipart/form-data)
func (h *ProductHandler) CreateProduct(w http.ResponseWriter, r *http.Request) {
	contentType := r.Header.Get("Content-Type")

	var req pModel.CreateProductRequest
	var imageURLs []string

	// Parse request based on content type
	if strings.Contains(contentType, "multipart/form-data") {
		// Handle multipart/form-data (with potential images)
		name, description, price, available, stock, status, err := h.parseMultipartForm(r)
		if err != nil {
			http.Error(w, "Failed to parse multipart request: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Create request struct from parsed values
		req = pModel.CreateProductRequest{
			Name:        name,
			Description: description,
			Price:       price,
			Available:   available,
			Stock:       stock,
			Status:      status,
		}
		imageURLs = []string{} // Will be populated later if images are present
	} else {
		// Handle JSON request (backward compatibility)
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		imageURLs = []string{} // No images for JSON requests
	}

	// Validate required fields
	if req.Name == "" {
		http.Error(w, "Name is required", http.StatusBadRequest)
		return
	}

	// Create product
	product := pModel.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Available:   req.Available,
		Stock:       req.Stock,
		Status:      req.Status,
		ImageURLs:   imageURLs,
	}

	ctx := r.Context()

	// Create product first
	newProduct, err := h.Repo.CreateProduct(ctx, product)
	if err != nil {
		http.Error(w, "Failed to create product", http.StatusInternalServerError)
		return
	}

	// Handle images if this was a multipart request
	if strings.Contains(contentType, "multipart/form-data") && h.ImageService != nil {
		// Get files from the parsed multipart form
		if r.MultipartForm != nil && r.MultipartForm.File != nil {
			files := r.MultipartForm.File["images"]
			if len(files) > 0 {
				// Save images
				imageURLs, err = h.ImageService.SaveProductImages(newProduct.ID, files)
				if err != nil {
					http.Error(w, "Failed to save images", http.StatusInternalServerError)
					return
				}

				// Update product with image URLs
				err = h.Repo.UpdateProductImages(ctx, newProduct.ID, imageURLs)
				if err != nil {
					http.Error(w, "Failed to update product images", http.StatusInternalServerError)
					return
				}

				// Update the product object with image URLs
				newProduct.ImageURLs = imageURLs
			}
		}
	}

	// Update history
	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get id user from context", http.StatusInternalServerError)
		return
	}

	err = h.UpdateHistoryTable(ctx, &product, newProduct.ID, idUser, pModel.ActionCreate)
	if err != nil {
		fmt.Printf("Error creating the history record for create product: %v", err)
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

// parseMultipartForm parses multipart/form-data requests and returns common product fields
func (h *ProductHandler) parseMultipartForm(r *http.Request) (name, description string, price float64, available bool, stock uint64, status pModel.ProductStatus, err error) {
	// Parse multipart form
	err = r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		return "", "", 0, false, 0, "", fmt.Errorf("failed to parse multipart form: %w", err)
	}

	// Extract form values
	name = r.FormValue("name")
	description = r.FormValue("description")
	price = parseFloat64(r.FormValue("price"))
	available = r.FormValue("available") == "true"
	stock = parseUint64(r.FormValue("stock"))
	status = pModel.ProductStatus(r.FormValue("status"))

	return name, description, price, available, stock, status, nil
}

// UpdateProduct - Update a product (handles both JSON and multipart/form-data)
func (h *ProductHandler) UpdateProduct(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	idStr := mux.Vars(r)["id"]
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	contentType := r.Header.Get("Content-Type")
	var req pModel.UpdateProductRequest
	var imageURLs []string

	// Parse request based on content type
	if strings.Contains(contentType, "multipart/form-data") {
		// Handle multipart/form-data (with potential images)
		name, description, price, available, stock, status, err := h.parseMultipartForm(r)
		if err != nil {
			http.Error(w, "Failed to parse multipart request: "+err.Error(), http.StatusBadRequest)
			return
		}

		// Create request struct from parsed values
		req = pModel.UpdateProductRequest{
			Name:        name,
			Description: description,
			Price:       price,
			Available:   available,
			Stock:       stock,
			Status:      status,
			ImageURLs:   []string{}, // Will be populated later if images are present
		}
		imageURLs = []string{} // Will be populated later if images are present
	} else {
		// Handle JSON request
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		imageURLs = req.ImageURLs // Use image URLs from JSON request
	}

	product := pModel.Product{
		ID:          id,
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Available:   req.Available,
		Stock:       req.Stock,
		Status:      req.Status,
		ImageURLs:   imageURLs,
	}

	ctx = r.Context()

	// Update product basic fields
	if err := h.Repo.UpdateProduct(ctx, product); err != nil {
		http.Error(w, "Failed to update product", http.StatusInternalServerError)
		return
	}

	// Handle images if this was a multipart request
	if strings.Contains(contentType, "multipart/form-data") && h.ImageService != nil {
		// Get files from the parsed multipart form
		if r.MultipartForm != nil && r.MultipartForm.File != nil {
			files := r.MultipartForm.File["images"]
			if len(files) > 0 {
				// Get existing product to preserve current images
				existingProduct, err := h.Repo.GetProductByID(ctx, id)
				if err != nil {
					fmt.Printf("Warning: Failed to get existing product: %v\n", err)
					// Continue with empty existing images
					existingProduct = pModel.Product{ImageURLs: []string{}}
				}

				// Save new images
				newImageURLs, err := h.ImageService.SaveProductImages(id, files)
				if err != nil {
					http.Error(w, "Failed to save images", http.StatusInternalServerError)
					return
				}

				// Combine existing and new image URLs
				allImageURLs := append(existingProduct.ImageURLs, newImageURLs...)

				// Update product with all image URLs (existing + new)
				fmt.Printf("DEBUG: About to update product images for ID %d with URLs: %v\n", id, allImageURLs)
				err = h.Repo.UpdateProductImages(ctx, id, allImageURLs)
				if err != nil {
					fmt.Printf("DEBUG: Error updating product images: %v\n", err)
					http.Error(w, "Failed to update product images", http.StatusInternalServerError)
					return
				}
				fmt.Printf("DEBUG: Successfully updated product images\n")

				// Update the product object with all image URLs
				product.ImageURLs = allImageURLs
			}
		}
	} else if len(imageURLs) > 0 {
		// Update image URLs from JSON request
		err = h.Repo.UpdateProductImages(ctx, id, imageURLs)
		if err != nil {
			http.Error(w, "Failed to update product images", http.StatusInternalServerError)
			return
		}
	}

	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get id user from context", http.StatusInternalServerError)
		return
	}

	err = h.UpdateHistoryTable(ctx, &product, id, idUser, pModel.ActionUpdate)
	if err != nil {
		fmt.Printf("Error creating the history record for update product: %v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"message": "Product updated successfully",
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
	if err := h.Repo.UpdateProductStatus(ctx, id, pModel.ProductStatus(req.Status)); err != nil {
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
		fmt.Printf("Error creating the history record for update product status: %v", err)
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
		IDProduct:   idProduct,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Available:   product.Available,
		Stock:       product.Stock,
		Status:      product.Status,
		ImageURLs:   product.ImageURLs,
		ModifiedBy:  idUser,
		Action:      action,
	}

	return h.Repo.CreateProductHistory(ctx, history)
}

// Helper functions
func parseFloat64(s string) float64 {
	if s == "" {
		return 0
	}
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return f
}

func parseUint64(s string) uint64 {
	if s == "" {
		return 0
	}
	u, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return u
}
