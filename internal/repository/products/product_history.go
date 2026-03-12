package products

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/logger"
	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ProductHistoryRepository struct {
	DB *sql.DB
}

// Creating a product history row for a tenant
func (r *ProductRepository) CreateProductHistory(_ context.Context, tenantID uint64, product pModel.ProductHistory) error {
	logger.Debug().
		Uint64("product_id", product.IDProduct).
		Str("action", string(product.Action)).
		Msg("Creating product history")

	validStatus := IsValidStatus(product.Status)
	if !validStatus {
		logger.Warn().
			Uint64("product_id", product.IDProduct).
			Str("status", string(product.Status)).
			Msg("Invalid status")
		return errors.NewBadRequest(errors.ErrInvalidStatus)
	}

	imageURLsJSON, err := json.Marshal(product.ImageURLs)
	if err != nil {
		logger.Err(err).
			Uint64("product_id", product.IDProduct).
			Msg("Error marshaling image URLs for product history")
		return errors.NewInternalServerError(errors.ErrCreatingProductHistory)
	}

	var thumbnailValue interface{}
	if product.ThumbnailURL == "" {
		thumbnailValue = nil
	} else {
		thumbnailValue = product.ThumbnailURL
	}

	result, err := r.DB.Exec(
		`INSERT INTO products_history (
		tenant_id,
		id_product, 
		name, 
		description, 
		price, 
		available, 
		stock,
		status, 
		image_urls,
		thumbnail_url,
		modified_by, 
		action
		) 
		VALUES (
		$1,
		$2,
		$3, 
		$4, 
		$5, 
		$6, 
		$7, 
		$8,
		$9,
		$10, 
		$11,
		$12)`,
		tenantID,
		product.IDProduct,
		product.Name,
		product.Description,
		product.Price,
		product.Available,
		product.Stock,
		product.Status,
		string(imageURLsJSON),
		thumbnailValue,
		product.ModifiedBy,
		product.Action,
	)

	if err != nil {
		logger.Err(err).
			Uint64("product_id", product.IDProduct).
			Msg("Error creating the product in the history table")
		return errors.NewInternalServerError(errors.ErrCreatingProductHistory)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		logger.Warn().Err(err).
			Uint64("product_id", product.IDProduct).
			Msg("Could not get the rows affected")
	}

	if rows == 0 {
		logger.Debug().
			Uint64("product_id", product.IDProduct).
			Msg("No rows affected when creating product history")
		return errors.NewNotFound(errors.ErrProductNotFound)
	}

	logger.Debug().
		Uint64("product_id", product.IDProduct).
		Int64("rows_affected", rows).
		Msg("Product history created successfully")

	return nil

}
