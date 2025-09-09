package images

import (
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_SaveProductImages(t *testing.T) {
	// Setup test directory
	testDir := "test_uploads"
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	service := New(testDir)

	tests := []struct {
		name          string
		productID     uint64
		files         []*multipart.FileHeader
		expectedURLs  []string
		expectedError bool
		errorMessage  string
	}{
		{
			name:      "HAPPY PATH: Save multiple images",
			productID: 1,
			files: []*multipart.FileHeader{
				createTestFileHeader("test1.jpg", "image/jpeg"),
				createTestFileHeader("test2.png", "image/png"),
			},
			expectedURLs: []string{
				"/uploads/products/1/main_*.jpg",
				"/uploads/products/1/gallery_1_*.png",
			},
			expectedError: false,
		},
		{
			name:          "HAPPY PATH: No images",
			productID:     1,
			files:         []*multipart.FileHeader{},
			expectedURLs:  []string{},
			expectedError: false,
		},
		{
			name:      "SAD PATH: Invalid file type",
			productID: 1,
			files: []*multipart.FileHeader{
				createTestFileHeader("test.txt", "text/plain"),
			},
			expectedError: true,
			errorMessage:  "invalid file type: test.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			imageURLs, err := service.SaveProductImages(tt.productID, tt.files)

			if tt.expectedError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMessage)
			} else {
				assert.NoError(t, err)
				assert.Len(t, imageURLs, len(tt.expectedURLs))

				// Check that files were created
				productDir := filepath.Join(testDir, "products", "1")
				if len(tt.files) > 0 {
					files, err := os.ReadDir(productDir)
					require.NoError(t, err)
					assert.Len(t, files, len(tt.files))
				}
			}
		})
	}
}

func TestService_DeleteProductImages(t *testing.T) {
	// Setup test directory
	testDir := "test_uploads"
	err := os.MkdirAll(testDir, 0755)
	require.NoError(t, err)
	defer os.RemoveAll(testDir)

	service := New(testDir)

	tests := []struct {
		name          string
		productID     uint64
		setupFiles    bool
		expectedError bool
	}{
		{
			name:          "HAPPY PATH: Delete existing images",
			productID:     1,
			setupFiles:    true,
			expectedError: false,
		},
		{
			name:          "HAPPY PATH: Delete non-existing images",
			productID:     999,
			setupFiles:    false,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup files if needed
			if tt.setupFiles {
				productDir := filepath.Join(testDir, "products", "1")
				err := os.MkdirAll(productDir, 0755)
				require.NoError(t, err)

				// Create a test file
				testFile := filepath.Join(productDir, "test.jpg")
				err = os.WriteFile(testFile, []byte("test content"), 0644)
				require.NoError(t, err)
			}

			err := service.DeleteProductImages(tt.productID)

			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Check that directory was deleted
				productDir := filepath.Join(testDir, "products", "1")
				_, err := os.Stat(productDir)
				assert.True(t, os.IsNotExist(err))
			}
		})
	}
}

func TestService_GetImagePath(t *testing.T) {
	service := New("/uploads")

	tests := []struct {
		name         string
		imageURL     string
		expectedPath string
	}{
		{
			name:         "HAPPY PATH: Valid image URL",
			imageURL:     "/uploads/products/1/main.jpg",
			expectedPath: filepath.Join("/uploads", "uploads", "products", "1", "main.jpg"),
		},
		{
			name:         "HAPPY PATH: URL without leading slash",
			imageURL:     "uploads/products/1/main.jpg",
			expectedPath: filepath.Join("/uploads", "uploads", "products", "1", "main.jpg"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := service.GetImagePath(tt.imageURL)
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

func TestService_isValidImageType(t *testing.T) {
	service := New("/uploads")

	tests := []struct {
		name        string
		filename    string
		contentType string
		expected    bool
	}{
		{
			name:        "Valid JPEG",
			filename:    "test.jpg",
			contentType: "image/jpeg",
			expected:    true,
		},
		{
			name:        "Valid PNG",
			filename:    "test.png",
			contentType: "image/png",
			expected:    true,
		},
		{
			name:        "Valid WebP",
			filename:    "test.webp",
			contentType: "image/webp",
			expected:    true,
		},
		{
			name:        "Invalid text file",
			filename:    "test.txt",
			contentType: "text/plain",
			expected:    false,
		},
		{
			name:        "Invalid PDF",
			filename:    "test.pdf",
			contentType: "application/pdf",
			expected:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fileHeader := createTestFileHeader(tt.filename, tt.contentType)
			result := service.isValidImageType(fileHeader)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// Helper function to create a test multipart.FileHeader
func createTestFileHeader(filename, contentType string) *multipart.FileHeader {
	// Create a temporary file
	file, err := os.CreateTemp("", "test_*")
	if err != nil {
		panic(err)
	}
	defer os.Remove(file.Name())

	// Write some content
	_, err = file.WriteString("fake image content")
	if err != nil {
		panic(err)
	}
	file.Close()

	// Create multipart form
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	fw, err := w.CreateFormFile("images", filename)
	if err != nil {
		panic(err)
	}

	// Read file content
	file, err = os.Open(file.Name())
	if err != nil {
		panic(err)
	}
	defer file.Close()

	_, err = io.Copy(fw, file)
	if err != nil {
		panic(err)
	}
	w.Close()

	// Parse the form
	r := multipart.NewReader(&b, w.Boundary())
	form, err := r.ReadForm(32 << 20) // 32 MB
	if err != nil {
		panic(err)
	}

	// Get the file header
	files := form.File["images"]
	if len(files) == 0 {
		panic("no files found")
	}

	// Set content type
	files[0].Header = make(map[string][]string)
	files[0].Header["Content-Type"] = []string{contentType}

	return files[0]
}
