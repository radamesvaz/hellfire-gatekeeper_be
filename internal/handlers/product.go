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
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ProductHandler struct {
	Repo         *productsRepository.ProductRepository
	ImageService *imagesService.Service
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
	// Check if request is multipart/form-data
	contentType := r.Header.Get("Content-Type")
	if contentType == "application/json" {
		h.createProductJSON(w, r)
		return
	}

	// Handle multipart/form-data
	h.createProductMultipart(w, r)
}

// createProductJSON handles JSON requests (backward compatibility)
func (h *ProductHandler) createProductJSON(w http.ResponseWriter, r *http.Request) {
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

// createProductMultipart handles multipart/form-data requests with images
func (h *ProductHandler) createProductMultipart(w http.ResponseWriter, r *http.Request) {
	// Parse multipart form (max 10MB)
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Error parsing form", http.StatusBadRequest)
		return
	}

	// Extract product data from form
	req := pModel.CreateProductRequest{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		Price:       parseFloat64(r.FormValue("price")),
		Available:   r.FormValue("available") == "true",
		Stock:       parseUint64(r.FormValue("stock")),
		Status:      pModel.ProductStatus(r.FormValue("status")),
	}

	// Validate required fields
	if req.Name == "" || req.Description == "" || req.Price == 0 {
		http.Error(w, "Missing required fields", http.StatusBadRequest)
		return
	}

	product := pModel.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
		Available:   req.Available,
		Stock:       req.Stock,
		Status:      req.Status,
	}

	ctx := r.Context()

	// Handle image uploads
	var imageURLs []string
	if h.ImageService != nil {
		files := r.MultipartForm.File["images"]
		if len(files) > 0 {
			// Create product first to get ID
			newProduct, err := h.Repo.CreateProductWithImages(ctx, product, []string{})
			if err != nil {
				http.Error(w, "Failed to create product", http.StatusInternalServerError)
				return
			}

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

			product.ID = newProduct.ID
			product.ImageURLs = imageURLs
		} else {
			// No images, create product normally
			newProduct, err := h.Repo.CreateProductWithImages(ctx, product, []string{})
			if err != nil {
				http.Error(w, "Failed to create product", http.StatusInternalServerError)
				return
			}
			product.ID = newProduct.ID
		}
	} else {
		// No image service, create product normally
		newProduct, err := h.Repo.CreateProduct(ctx, product)
		if err != nil {
			http.Error(w, "Failed to create product", http.StatusInternalServerError)
			return
		}
		product.ID = newProduct.ID
	}

	// Update history
	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get id user from context", http.StatusInternalServerError)
		return
	}

	err = h.UpdateHistoryTable(ctx, &product, product.ID, idUser, pModel.ActionCreate)
	if err != nil {
		fmt.Printf("Error creating the history record for create product :%v", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"message":    "Product created successfully",
		"product_id": product.ID,
		"image_urls": imageURLs,
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

// Helper functions for parsing form values
func parseFloat64(s string) float64 {
	if s == "" {
		return 0
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

func parseUint64(s string) uint64 {
	if s == "" {
		return 0
	}
	val, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		return 0
	}
	return val
}
