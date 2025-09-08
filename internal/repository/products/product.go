package products

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/radamesvaz/bakery-app/internal/errors"
	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ProductRepository struct {
	DB *sql.DB
}

// GetAllProducts gets all the products from the table
func (r *ProductRepository) GetAllProducts(_ context.Context) ([]pModel.Product, error) {
	fmt.Println(
		"Getting all products",
	)
	rows, err := r.DB.Query("SELECT id_product, name, description, price, available, stock, status, image_urls, created_on FROM products")
	if err != nil {
		fmt.Printf("Error getting the products: %v", err)
		return nil, err
	}
	defer rows.Close()

	var products []pModel.Product
	for rows.Next() {
		var product pModel.Product
		var imageURLsJSON sql.NullString
		if err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.Available,
			&product.Stock,
			&product.Status,
			&imageURLsJSON,
			&product.CreatedOn,
		); err != nil {
			fmt.Printf("Error mapping the products: %v", err)
			return nil, err
		}

		// Parse image URLs from JSON
		if imageURLsJSON.Valid && imageURLsJSON.String != "" {
			var imageURLs []string
			if err := json.Unmarshal([]byte(imageURLsJSON.String), &imageURLs); err != nil {
				fmt.Printf("Error parsing image URLs: %v", err)
				product.ImageURLs = []string{}
			} else {
				product.ImageURLs = imageURLs
			}
		} else {
			product.ImageURLs = []string{}
		}

		products = append(products, product)
	}
	return products, nil
}

// Getting a product by its ID
func (r *ProductRepository) GetProductByID(_ context.Context, idProduct uint64) (pModel.Product, error) {
	fmt.Printf(
		"Getting product by id = %v",
		idProduct,
	)

	product := pModel.Product{}
	var imageURLsJSON sql.NullString

	err := r.DB.QueryRow(
		"SELECT id_product, name, description, price, available, stock, status, image_urls, created_on FROM products WHERE id_product = ?",
		idProduct,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Available,
		&product.Stock,
		&product.Status,
		&imageURLsJSON,
		&product.CreatedOn,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Println("Product not found")
			return product, errors.NewNotFound(errors.ErrProductNotFound)
		}
		fmt.Printf("Error retrieving the product: %v", err)
		return product, errors.NewNotFound(errors.ErrCouldNotGetTheProduct)
	}

	// Parse image URLs from JSON
	if imageURLsJSON.Valid && imageURLsJSON.String != "" {
		var imageURLs []string
		if err := json.Unmarshal([]byte(imageURLsJSON.String), &imageURLs); err != nil {
			fmt.Printf("Error parsing image URLs: %v", err)
			product.ImageURLs = []string{}
		} else {
			product.ImageURLs = imageURLs
		}
	} else {
		product.ImageURLs = []string{}
	}

	return product, nil
}

// Getting multiple products product by their IDs
func (r *ProductRepository) GetProductsByIDs(ctx context.Context, ids []uint64) ([]pModel.Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Generate placeholders
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		placeholders[i] = "?"
		args[i] = id
	}

	query := fmt.Sprintf(`
		SELECT id_product, name, price, stock 
		FROM products 
		WHERE id_product IN (%s)`, strings.Join(placeholders, ","))

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying products: %w", err)
	}
	defer rows.Close()

	products := []pModel.Product{}
	for rows.Next() {
		var p pModel.Product
		if err := rows.Scan(&p.ID, &p.Name, &p.Price, &p.Stock); err != nil {
			return nil, fmt.Errorf("error scanning product: %w", err)
		}
		products = append(products, p)
	}

	return products, nil
}

// Creating a product
func (r *ProductRepository) CreateProduct(_ context.Context, product pModel.Product) (pModel.Product, error) {
	fmt.Printf("Creating product: %v", product.Name)

	createdProduct := pModel.Product{}

	if product.Name == "" || product.Description == "" || product.Price == 0 {
		return createdProduct, errors.NewBadRequest(errors.ErrCreatingProduct)
	}

	// Convert imageURLs to JSON
	imageURLsJSON, err := json.Marshal(product.ImageURLs)
	if err != nil {
		fmt.Printf("Error marshaling image URLs: %v", err)
		return createdProduct, errors.NewInternalServerError(errors.ErrCreatingProduct)
	}

	result, err := r.DB.Exec(
		`INSERT INTO products 
		(name, description, price, available, stock, status, image_urls) 
		VALUES (?, ?, ?, ?, ?, ?, ?) `,
		product.Name, product.Description, product.Price, product.Available, product.Stock, product.Status, string(imageURLsJSON))

	if err != nil {
		fmt.Printf("Error creating the product: %v", err)
		return createdProduct, errors.NewInternalServerError(errors.ErrCreatingProduct)
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		fmt.Printf("Error getting the last insert ID: %v", err)
		return createdProduct, errors.NewInternalServerError(errors.ErrCreatingProduct)
	}

	createdProduct.ID = uint64(insertedID)
	createdProduct.Name = product.Name
	createdProduct.Description = product.Description
	createdProduct.Price = product.Price
	createdProduct.Available = product.Available
	createdProduct.Stock = product.Stock
	createdProduct.Status = product.Status
	createdProduct.ImageURLs = product.ImageURLs

	return createdProduct, nil
}

// Updating a product status
func (r *ProductRepository) UpdateProductStatus(_ context.Context, idProduct uint64, status pModel.ProductStatus) error {
	fmt.Printf(
		"%s product status by id = %v",
		status,
		idProduct,
	)

	validStatus := IsValidStatus(status)
	if !validStatus {
		fmt.Printf("Invalid status: %v", status)
		return errors.NewBadRequest(errors.ErrInvalidStatus)
	}

	result, err := r.DB.Exec(
		"UPDATE products SET status = ? where id_product = ?",
		status,
		idProduct,
	)

	if err != nil {
		fmt.Printf("Error updating the product status: %v", err)
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Could not get the rows affected: %v", err)
	}

	if rows == 0 {
		return errors.NewNotFound(errors.ErrProductNotFound)
	}

	fmt.Printf("RESULT: %v", rows)

	return nil
}

// Updating product
func (r *ProductRepository) UpdateProduct(_ context.Context, product pModel.Product) error {
	fmt.Printf(
		"Updating product status by id = %v",
		product.ID,
	)

	validStatus := IsValidStatus(product.Status)
	if !validStatus {
		fmt.Printf("Invalid status: %v", product.Status)
		return errors.NewBadRequest(errors.ErrInvalidStatus)
	}

	result, err := r.DB.Exec(
		"UPDATE products SET name = ?, description = ?, price = ?, available = ?, stock = ?, status = ? where id_product = ?",
		product.Name,
		product.Description,
		product.Price,
		product.Available,
		product.Stock,
		product.Status,
		product.ID,
	)

	if err != nil {
		fmt.Printf("Error updating the product: %v", err)
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Could not get the rows affected: %v", err)
	}

	if rows == 0 {
		return errors.NewNotFound(errors.ErrProductNotFound)
	}

	fmt.Printf("RESULT: %v", rows)

	return nil
}

// Validates if the status is a valid one
func IsValidStatus(status pModel.ProductStatus) bool {
	switch pModel.ProductStatus(status) {
	case pModel.StatusActive, pModel.StatusInactive, pModel.StatusDeleted:
		return true
	default:
		return false
	}
}

// UpdateProductStock updates only the stock of a product
func (r *ProductRepository) UpdateProductStock(_ context.Context, idProduct uint64, newStock uint64) error {
	fmt.Printf("Updating product stock to %d for product id = %d", newStock, idProduct)

	result, err := r.DB.Exec(
		"UPDATE products SET stock = ? WHERE id_product = ?",
		newStock,
		idProduct,
	)

	if err != nil {
		fmt.Printf("Error updating product stock: %v", err)
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Could not get the rows affected: %v", err)
	}

	if rows == 0 {
		return errors.NewNotFound(errors.ErrProductNotFound)
	}

	fmt.Printf("Stock updated successfully. Rows affected: %v", rows)
	return nil
}

// UpdateProductImages updates the image URLs for a product
func (r *ProductRepository) UpdateProductImages(_ context.Context, idProduct uint64, imageURLs []string) error {
	fmt.Printf("Updating product images for product id = %d", idProduct)

	// Convert imageURLs to JSON
	imageURLsJSON, err := json.Marshal(imageURLs)
	if err != nil {
		fmt.Printf("Error marshaling image URLs: %v", err)
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	result, err := r.DB.Exec(
		"UPDATE products SET image_urls = ? WHERE id_product = ?",
		string(imageURLsJSON),
		idProduct,
	)

	if err != nil {
		fmt.Printf("Error updating product images: %v", err)
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		fmt.Printf("Could not get the rows affected: %v", err)
	}

	if rows == 0 {
		return errors.NewNotFound(errors.ErrProductNotFound)
	}

	fmt.Printf("Product images updated successfully. Rows affected: %v", rows)
	return nil
}
