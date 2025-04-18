package products

import (
	"database/sql"
	"fmt"

	"github.com/radamesvaz/bakery-app/internal/errors"
	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ProductHistoryRepository struct {
	DB *sql.DB
}

// Creating a product
func (r *ProductRepository) CreateProductHistory(
	id int,
	productName string,
	productDescription string,
	productPrice float64,
	available bool,
	status string,
	modifiedBy int,
	action pModel.ProductAction,
) error {
	fmt.Printf("Creating product history: %v", id)

	if productName == "" || productDescription == "" || productPrice == 0 {
		return errors.NewBadRequest(errors.ErrCreatingProductHistory)
	}

	result, err := r.DB.Exec(
		`INSERT INTO products_history (
		id, 
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
		id,
		productName,
		productDescription,
		productPrice,
		available,
		status,
		modifiedBy,
		action,
	)

	if err != nil {
		return errors.NewInternalServerError(errors.ErrCreatingProduct)
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

// // Validates if the status is a valid one
// func IsValidStatus(status pModel.ProductStatus) bool {
// 	switch pModel.ProductStatus(status) {
// 	case pModel.StatusActive, pModel.StatusInactive, pModel.StatusDeleted:
// 		return true
// 	default:
// 		return false
// 	}
// }
