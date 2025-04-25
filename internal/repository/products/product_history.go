package products

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/radamesvaz/bakery-app/internal/errors"
	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ProductHistoryRepository struct {
	DB *sql.DB
}

// Creating a product
func (r *ProductRepository) CreateProductHistory(_ context.Context, product pModel.ProductHistory) error {
	fmt.Printf("Creating product history: %v", product.IDProduct)

	validStatus := IsValidStatus(product.Status)
	if !validStatus {
		fmt.Printf("Invalid status: %v", product.Status)
		return errors.NewBadRequest(errors.ErrInvalidStatus)
	}

	result, err := r.DB.Exec(
		`INSERT INTO products_history (
		id_product, 
		name, 
		description, 
		price, 
		available, 
		status, 
		modified_by, 
		action
		) 
		VALUES (
		?,
		?, 
		?, 
		?, 
		?, 
		?, 
		?, 
		?)`,
		product.IDProduct,
		product.Name,
		product.Description,
		product.Price,
		product.Available,
		product.Status,
		product.ModifiedBy,
		product.Action,
	)

	if err != nil {
		fmt.Printf("Error creating the product in the history table: %v", err)
		return errors.NewInternalServerError(errors.ErrCreatingProductHistory)
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
