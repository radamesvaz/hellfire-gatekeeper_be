package products

import (
	"context"
	"database/sql"
	"fmt"

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
	rows, err := r.DB.Query("SELECT id_product, name, description, price, available, stock, status, created_on FROM products")
	if err != nil {
		fmt.Printf("Error getting the products: %v", err)
		return nil, err
	}
	defer rows.Close()

	var products []pModel.Product
	for rows.Next() {
		var product pModel.Product
		if err := rows.Scan(
			&product.ID,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.Available,
			&product.Stock,
			&product.Status,
			&product.CreatedOn,
		); err != nil {
			fmt.Printf("Error mapping the products: %v", err)
			return nil, err
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

	err := r.DB.QueryRow(
		"SELECT id_product, name, description, price, available, stock, status, created_on FROM products WHERE id_product = ?",
		idProduct,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Available,
		&product.Stock,
		&product.Status,
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

	return product, nil
}

// Creating a product
func (r *ProductRepository) CreateProduct(_ context.Context, product pModel.Product) (pModel.Product, error) {
	fmt.Printf("Creating product: %v", product.Name)

	createdProduct := pModel.Product{}

	if product.Name == "" || product.Description == "" || product.Price == 0 {
		return createdProduct, errors.NewBadRequest(errors.ErrCreatingProduct)
	}

	result, err := r.DB.Exec(
		`INSERT INTO products 
		(name, description, price, available, stock, status) 
		VALUES (?, ?, ?, ?, ?, ?) `,
		product.Name, product.Description, product.Price, product.Available, product.Stock, product.Status)

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
