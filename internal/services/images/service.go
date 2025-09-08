package images

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type Service struct {
	UploadDir string
}

func New(uploadDir string) *Service {
	return &Service{
		UploadDir: uploadDir,
	}
}

// SaveProductImages saves uploaded images for a product
func (s *Service) SaveProductImages(productID uint64, files []*multipart.FileHeader) ([]string, error) {
	if len(files) == 0 {
		return []string{}, nil
	}

	// Create product directory
	productDir := filepath.Join(s.UploadDir, "products", fmt.Sprintf("%d", productID))
	if err := os.MkdirAll(productDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create product directory: %w", err)
	}

	var imageURLs []string

	for i, file := range files {
		// Validate file type
		if !s.isValidImageType(file) {
			return nil, fmt.Errorf("invalid file type: %s", file.Filename)
		}

		// Generate unique filename
		ext := filepath.Ext(file.Filename)
		filename := s.generateFilename(i, ext)
		filePath := filepath.Join(productDir, filename)

		// Save file
		if err := s.saveFile(file, filePath); err != nil {
			return nil, fmt.Errorf("failed to save file %s: %w", file.Filename, err)
		}

		// Add to URLs
		imageURL := fmt.Sprintf("/uploads/products/%d/%s", productID, filename)
		imageURLs = append(imageURLs, imageURL)
	}

	return imageURLs, nil
}

// isValidImageType checks if the file is a valid image type
func (s *Service) isValidImageType(file *multipart.FileHeader) bool {
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
	}

	contentType := file.Header.Get("Content-Type")
	return allowedTypes[contentType]
}

// generateFilename generates a unique filename for the image
func (s *Service) generateFilename(index int, ext string) string {
	timestamp := time.Now().UnixNano()
	if index == 0 {
		return fmt.Sprintf("main_%d%s", timestamp, ext)
	}
	return fmt.Sprintf("gallery_%d_%d%s", index, timestamp, ext)
}

// saveFile saves the uploaded file to disk
func (s *Service) saveFile(file *multipart.FileHeader, filePath string) error {
	src, err := file.Open()
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer dst.Close()

	if _, err = io.Copy(dst, src); err != nil {
		return err
	}

	return nil
}

// DeleteProductImages deletes all images for a product
func (s *Service) DeleteProductImages(productID uint64) error {
	productDir := filepath.Join(s.UploadDir, "products", fmt.Sprintf("%d", productID))

	// Check if directory exists
	if _, err := os.Stat(productDir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, nothing to delete
	}

	// Remove directory and all its contents
	if err := os.RemoveAll(productDir); err != nil {
		return fmt.Errorf("failed to delete product images: %w", err)
	}

	return nil
}

// GetImagePath returns the full path to an image
func (s *Service) GetImagePath(imageURL string) string {
	// Remove leading slash from URL
	cleanURL := strings.TrimPrefix(imageURL, "/")
	return filepath.Join(s.UploadDir, cleanURL)
}
