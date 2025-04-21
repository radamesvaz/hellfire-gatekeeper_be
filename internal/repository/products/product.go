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
// TODO change name
func (r *ProductRepository) GetAllProducts() ([]pModel.Product, error) {
	fmt.Println(
		"Getting all products",
	)
	rows, err := r.DB.Query("SELECT id_product, name, description, price, available, status, created_on FROM products")
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
func (r *ProductRepository) GetProductByID(idProduct uint64) (pModel.Product, error) {
	fmt.Printf(
		"Getting product by id = %v",
		idProduct,
	)

	product := pModel.Product{}

	err := r.DB.QueryRow(
		"SELECT id_product, name, description, price, available, status, created_on FROM products WHERE id_product = ?",
		idProduct,
	).Scan(
		&product.ID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.Available,
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
func (r *ProductRepository) CreateProduct(product pModel.Product) (pModel.Product, error) {
	fmt.Printf("Creating product: %v", product.Name)

	createdProduct := pModel.Product{}

	if product.Name == "" || product.Description == "" || product.Price == 0 {
		return createdProduct, errors.NewBadRequest(errors.ErrCreatingProduct)
	}

	row := r.DB.QueryRow(
		`INSERT INTO products 
		(name, description, price, available, status) 
		VALUES (?, ?, ?, ?, ?) 
		RETURNING 
		id_product, 
		name, 
		description, 
		price,
		available,
		status, 
		created_on`,
		product.Name, product.Description, product.Price, product.Available, product.Status)
	err := row.Scan(
		&createdProduct.ID,
		&createdProduct.Name,
		&createdProduct.Description,
		&createdProduct.Price,
		&createdProduct.Available,
		&createdProduct.Status,
		&createdProduct.CreatedOn,
	)

	if err != nil {
		return createdProduct, errors.NewInternalServerError(errors.ErrCreatingProduct)
	}

	// TODO: change when we hace the id user on the handler
	// errHistory := r.CreateProductHistory(
	// 	createdProduct.ID,
	// 	createdProduct.Name,
	// 	createdProduct.Description,
	// 	createdProduct.Price,
	// 	createdProduct.Available,
	// 	createdProduct.Status,
	// 	1,
	// 	pModel.ActionCreate,
	// )

	// if errHistory != nil {
	// 	fmt.Printf("Error adding the product to the history table: %v", errHistory)
	// }

	return createdProduct, nil

}

// Updating a product status
func (r *ProductRepository) UpdateProductStatus(idProduct uint64, status pModel.ProductStatus) error {
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
		"UPDATE products SET name = ?, description = ?, price = ?, available = ?, status = ? where id_product = ?",
		product.Name,
		product.Description,
		product.Price,
		product.Available,
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
