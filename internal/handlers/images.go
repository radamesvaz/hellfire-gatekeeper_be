package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	productsRepository "github.com/radamesvaz/bakery-app/internal/repository/products"
	imagesService "github.com/radamesvaz/bakery-app/internal/services/images"
	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ImageHandler struct {
	Repo         *productsRepository.ProductRepository
	ImageService *imagesService.Service
}

// AddProductImages - Add images to a product
func (h *ImageHandler) AddProductImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get product ID from URL
	idStr := mux.Vars(r)["id"]
	productID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	// Check if product exists
	_, err = h.Repo.GetProductByID(ctx, productID)
	if err != nil {
		if errors.Is(err, appErrors.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get product", http.StatusInternalServerError)
		return
	}

	// Parse multipart form
	err = r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Get files from the parsed multipart form
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		http.Error(w, "No files provided", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "No images provided", http.StatusBadRequest)
		return
	}

	// Validate image files
	for _, file := range files {
		if !h.ImageService.IsValidImageType(file) {
			http.Error(w, fmt.Sprintf("Invalid image type: %s", file.Filename), http.StatusBadRequest)
			return
		}
	}

	// Get existing product to preserve current images
	existingProduct, err := h.Repo.GetProductByID(ctx, productID)
	if err != nil {
		http.Error(w, "Failed to get existing product", http.StatusInternalServerError)
		return
	}

	// Save new images
	newImageURLs, err := h.ImageService.SaveProductImages(productID, files)
	if err != nil {
		http.Error(w, "Failed to save images", http.StatusInternalServerError)
		return
	}

	// Combine existing and new image URLs
	allImageURLs := append(existingProduct.ImageURLs, newImageURLs...)

	// Update product with all image URLs (existing + new)
	err = h.Repo.UpdateProductImages(ctx, productID, allImageURLs)
	if err != nil {
		http.Error(w, "Failed to update product images", http.StatusInternalServerError)
		return
	}

	// Update history
	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}

	product := pModel.Product{ID: productID, ImageURLs: allImageURLs}
	err = h.UpdateHistoryTable(ctx, &product, productID, idUser, pModel.ActionUpdate)
	if err != nil {
		fmt.Printf("Error creating the history record for add images: %v", err)
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"message":     "Images added successfully",
		"product_id":  productID,
		"new_images":  newImageURLs,
		"all_images":  allImageURLs,
		"total_count": len(allImageURLs),
	}
	json.NewEncoder(w).Encode(response)
}

// DeleteProductImage - Delete a specific image from a product
func (h *ImageHandler) DeleteProductImage(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get product ID and image URL from URL
	vars := mux.Vars(r)
	productIDStr := vars["id"]
	imageURL := vars["imageUrl"]

	productID, err := strconv.ParseUint(productIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	// Check if product exists
	product, err := h.Repo.GetProductByID(ctx, productID)
	if err != nil {
		if errors.Is(err, appErrors.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get product", http.StatusInternalServerError)
		return
	}

	// Check if image exists in product
	imageIndex := -1
	for i, url := range product.ImageURLs {
		if url == imageURL {
			imageIndex = i
			break
		}
	}

	if imageIndex == -1 {
		http.Error(w, "Image not found in product", http.StatusNotFound)
		return
	}

	// Remove image from slice
	newImageURLs := make([]string, 0, len(product.ImageURLs)-1)
	newImageURLs = append(newImageURLs, product.ImageURLs[:imageIndex]...)
	newImageURLs = append(newImageURLs, product.ImageURLs[imageIndex+1:]...)

	// Update product with new image URLs
	err = h.Repo.UpdateProductImages(ctx, productID, newImageURLs)
	if err != nil {
		http.Error(w, "Failed to update product images", http.StatusInternalServerError)
		return
	}

	// Delete image file from filesystem
	err = h.ImageService.DeleteImage(imageURL)
	if err != nil {
		fmt.Printf("Warning: Failed to delete image file %s: %v", imageURL, err)
		// Don't fail the request if file deletion fails
	}

	// Update history
	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}

	updatedProduct := pModel.Product{ID: productID, ImageURLs: newImageURLs}
	err = h.UpdateHistoryTable(ctx, &updatedProduct, productID, idUser, pModel.ActionUpdate)
	if err != nil {
		fmt.Printf("Error creating the history record for delete image: %v", err)
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"message":          "Image deleted successfully",
		"product_id":       productID,
		"deleted_url":      imageURL,
		"remaining_images": newImageURLs,
		"total_count":      len(newImageURLs),
	}
	json.NewEncoder(w).Encode(response)
}

// ReplaceProductImages - Replace all images for a product
func (h *ImageHandler) ReplaceProductImages(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Get product ID from URL
	idStr := mux.Vars(r)["id"]
	productID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid product ID", http.StatusBadRequest)
		return
	}

	// Check if product exists
	existingProduct, err := h.Repo.GetProductByID(ctx, productID)
	if err != nil {
		if errors.Is(err, appErrors.ErrProductNotFound) {
			http.Error(w, "Product not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get product", http.StatusInternalServerError)
		return
	}

	// Parse multipart form
	err = r.ParseMultipartForm(32 << 20) // 32 MB max
	if err != nil {
		http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
		return
	}

	// Get files from the parsed multipart form
	if r.MultipartForm == nil || r.MultipartForm.File == nil {
		http.Error(w, "No files provided", http.StatusBadRequest)
		return
	}

	files := r.MultipartForm.File["images"]
	if len(files) == 0 {
		http.Error(w, "No images provided", http.StatusBadRequest)
		return
	}

	// Validate image files
	for _, file := range files {
		if !h.ImageService.IsValidImageType(file) {
			http.Error(w, fmt.Sprintf("Invalid image type: %s", file.Filename), http.StatusBadRequest)
			return
		}
	}

	// Delete existing images from filesystem
	for _, imageURL := range existingProduct.ImageURLs {
		err = h.ImageService.DeleteImage(imageURL)
		if err != nil {
			fmt.Printf("Warning: Failed to delete existing image file %s: %v", imageURL, err)
		}
	}

	// Save new images
	newImageURLs, err := h.ImageService.SaveProductImages(productID, files)
	if err != nil {
		http.Error(w, "Failed to save images", http.StatusInternalServerError)
		return
	}

	// Update product with new image URLs (replace all)
	err = h.Repo.UpdateProductImages(ctx, productID, newImageURLs)
	if err != nil {
		http.Error(w, "Failed to update product images", http.StatusInternalServerError)
		return
	}

	// Update history
	idUser, err := middleware.GetUserIDFromContext(ctx)
	if err != nil {
		http.Error(w, "Failed to get user ID from context", http.StatusInternalServerError)
		return
	}

	product := pModel.Product{ID: productID, ImageURLs: newImageURLs}
	err = h.UpdateHistoryTable(ctx, &product, productID, idUser, pModel.ActionUpdate)
	if err != nil {
		fmt.Printf("Error creating the history record for replace images: %v", err)
	}

	// Return response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	response := map[string]interface{}{
		"message":     "Images replaced successfully",
		"product_id":  productID,
		"new_images":  newImageURLs,
		"total_count": len(newImageURLs),
	}
	json.NewEncoder(w).Encode(response)
}

// UpdateHistoryTable - Update the history table (shared with ProductHandler)
func (h *ImageHandler) UpdateHistoryTable(ctx context.Context, product *pModel.Product, idProduct uint64, idUser uint64, action pModel.ProductAction) error {
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
