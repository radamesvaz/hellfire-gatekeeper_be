package images

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
)

type Service struct {
	UploadDir     string
	Cloudinary    *cloudinary.Cloudinary
	UseCloudinary bool
}

func New(uploadDir string) *Service {
	// Try to initialize Cloudinary from environment variables
	cloudName := os.Getenv("CLOUDINARY_CLOUD_NAME")
	apiKey := os.Getenv("CLOUDINARY_API_KEY")
	apiSecret := os.Getenv("CLOUDINARY_API_SECRET")

	var cld *cloudinary.Cloudinary
	useCloudinary := false

	if cloudName != "" && apiKey != "" && apiSecret != "" {
		var err error
		cld, err = cloudinary.NewFromParams(cloudName, apiKey, apiSecret)
		if err == nil {
			useCloudinary = true
			fmt.Println("✅ Cloudinary initialized successfully")
		} else {
			fmt.Printf("⚠️ Failed to initialize Cloudinary: %v, falling back to local storage\n", err)
		}
	} else {
		fmt.Println("⚠️ Cloudinary credentials not found, using local storage")
	}

	return &Service{
		UploadDir:     uploadDir,
		Cloudinary:    cld,
		UseCloudinary: useCloudinary,
	}
}

// SaveProductImages saves uploaded images for a product
func (s *Service) SaveProductImages(productID uint64, files []*multipart.FileHeader) ([]string, error) {
	if len(files) == 0 {
		return []string{}, nil
	}

	var imageURLs []string

	for i, file := range files {
		// Validate file type
		if !s.IsValidImageType(file) {
			return nil, fmt.Errorf("invalid file type: %s", file.Filename)
		}

		var imageURL string
		var err error

		if s.UseCloudinary {
			// Upload to Cloudinary
			imageURL, err = s.uploadToCloudinary(productID, file, i)
			if err != nil {
				return nil, fmt.Errorf("failed to upload file %s to Cloudinary: %w", file.Filename, err)
			}
		} else {
			// Fallback to local storage
			imageURL, err = s.saveToLocal(productID, file, i)
			if err != nil {
				return nil, fmt.Errorf("failed to save file %s locally: %w", file.Filename, err)
			}
		}

		imageURLs = append(imageURLs, imageURL)
	}

	return imageURLs, nil
}

// IsValidImageType checks if the file is a valid image type
func (s *Service) IsValidImageType(file *multipart.FileHeader) bool {
	allowedTypes := map[string]bool{
		"image/jpeg": true,
		"image/jpg":  true,
		"image/png":  true,
		"image/webp": true,
	}

	// Check by Content-Type first
	contentType := file.Header.Get("Content-Type")
	if allowedTypes[contentType] {
		return true
	}

	// Fallback: check by file extension
	ext := strings.ToLower(filepath.Ext(file.Filename))
	allowedExtensions := map[string]bool{
		".jpg":  true,
		".jpeg": true,
		".png":  true,
		".webp": true,
	}

	return allowedExtensions[ext]
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

// DeleteImage deletes a specific image file
func (s *Service) DeleteImage(imageURL string) error {
	if s.UseCloudinary {
		return s.deleteFromCloudinary(imageURL)
	}
	return s.deleteFromLocal(imageURL)
}

// deleteFromCloudinary deletes an image from Cloudinary
func (s *Service) deleteFromCloudinary(imageURL string) error {
	// Extract public ID from Cloudinary URL
	publicID := s.extractPublicIDFromURL(imageURL)
	if publicID == "" {
		return fmt.Errorf("could not extract public ID from URL: %s", imageURL)
	}

	ctx := context.Background()
	_, err := s.Cloudinary.Upload.Destroy(ctx, uploader.DestroyParams{
		PublicID: publicID,
	})

	if err != nil {
		return fmt.Errorf("failed to delete image from Cloudinary: %w", err)
	}

	return nil
}

// deleteFromLocal deletes an image from local filesystem
func (s *Service) deleteFromLocal(imageURL string) error {
	// Get the full path to the image
	imagePath := s.GetImagePath(imageURL)

	// Check if file exists
	if _, err := os.Stat(imagePath); os.IsNotExist(err) {
		return fmt.Errorf("image file not found: %s", imagePath)
	}

	// Delete the file
	if err := os.Remove(imagePath); err != nil {
		return fmt.Errorf("failed to delete image file %s: %w", imagePath, err)
	}

	return nil
}

// extractPublicIDFromURL extracts the public ID from a Cloudinary URL
func (s *Service) extractPublicIDFromURL(imageURL string) string {
	// Cloudinary URLs look like: https://res.cloudinary.com/cloud_name/image/upload/v1234567890/folder/public_id.jpg
	// We need to extract the part after the last slash and before the file extension

	// Remove the base URL and get the path
	parts := strings.Split(imageURL, "/")
	if len(parts) < 2 {
		return ""
	}

	// Find the "upload" part and get everything after it
	uploadIndex := -1
	for i, part := range parts {
		if part == "upload" {
			uploadIndex = i
			break
		}
	}

	if uploadIndex == -1 || uploadIndex >= len(parts)-1 {
		return ""
	}

	// Get the public ID (everything after upload, skipping version if present)
	publicIDParts := parts[uploadIndex+1:]

	// Skip version if it starts with 'v' followed by numbers
	if len(publicIDParts) > 0 && strings.HasPrefix(publicIDParts[0], "v") {
		publicIDParts = publicIDParts[1:]
	}

	if len(publicIDParts) == 0 {
		return ""
	}

	// Join the remaining parts and remove file extension
	publicID := strings.Join(publicIDParts, "/")
	ext := filepath.Ext(publicID)
	if ext != "" {
		publicID = strings.TrimSuffix(publicID, ext)
	}

	return publicID
}

// GetImagePath returns the full path to an image
func (s *Service) GetImagePath(imageURL string) string {
	// Remove leading slash from URL
	cleanURL := strings.TrimPrefix(imageURL, "/")
	// Remove uploads/ prefix if present
	cleanURL = strings.TrimPrefix(cleanURL, "uploads/")
	return filepath.Join(s.UploadDir, cleanURL)
}

// uploadToCloudinary uploads an image to Cloudinary
func (s *Service) uploadToCloudinary(productID uint64, file *multipart.FileHeader, index int) (string, error) {
	// Open the file
	src, err := file.Open()
	if err != nil {
		return "", err
	}
	defer src.Close()

	// Generate public ID
	ext := filepath.Ext(file.Filename)
	publicID := s.generateCloudinaryPublicID(productID, index, ext)

	// Upload to Cloudinary
	ctx := context.Background()
	result, err := s.Cloudinary.Upload.Upload(ctx, src, uploader.UploadParams{
		PublicID: publicID,
		Folder:   "bakery/products",
		Tags:     []string{"bakery", "product", fmt.Sprintf("product_%d", productID)},
	})

	if err != nil {
		return "", err
	}

	return result.SecureURL, nil
}

// saveToLocal saves an image to local filesystem (fallback)
func (s *Service) saveToLocal(productID uint64, file *multipart.FileHeader, index int) (string, error) {
	// Create product directory
	productDir := filepath.Join(s.UploadDir, "products", fmt.Sprintf("%d", productID))
	if err := os.MkdirAll(productDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create product directory: %w", err)
	}

	// Generate unique filename
	ext := filepath.Ext(file.Filename)
	filename := s.generateFilename(index, ext)
	filePath := filepath.Join(productDir, filename)

	// Save file
	if err := s.saveFile(file, filePath); err != nil {
		return "", err
	}

	// Return local URL
	return fmt.Sprintf("/uploads/products/%d/%s", productID, filename), nil
}

// generateCloudinaryPublicID generates a unique public ID for Cloudinary
func (s *Service) generateCloudinaryPublicID(productID uint64, index int, ext string) string {
	timestamp := time.Now().UnixNano()
	if index == 0 {
		return fmt.Sprintf("product_%d_main_%d", productID, timestamp)
	}
	return fmt.Sprintf("product_%d_gallery_%d_%d", productID, index, timestamp)
}
