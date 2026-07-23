package products

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/logger"
	"github.com/radamesvaz/bakery-app/internal/pagination"
	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ProductRepository struct {
	DB *sql.DB
}

// ListProductsPageResult is one page of products (cursor pagination on id_product DESC).
type ListProductsPageResult struct {
	Items      []pModel.Product
	NextCursor *string
}

// ListProductsPage returns up to limit products for the tenant, ordered by id_product descending.
// If afterID is non-nil, only rows with id_product < *afterID are considered (next page).
// If namePrefix is non-nil, only rows whose name contains the query text (case-insensitive) are included.
// If activeOnly is true, only products with status = 'active' are included.
// Fetches limit+1 rows internally to detect a following page.
func (r *ProductRepository) ListProductsPage(
	ctx context.Context,
	tenantID uint64,
	limit int,
	afterID *uint64,
	namePrefix *string,
	activeOnly bool,
) (ListProductsPageResult, error) {
	logger.Debug().Uint64("tenant_id", tenantID).Int("limit", limit).Msg("Listing products page")
	if limit < 1 {
		return ListProductsPageResult{}, fmt.Errorf("limit must be at least 1")
	}

	q := `SELECT id_product, tenant_id, name, description, price, track_inventory, stock, status, image_urls, thumbnail_url, created_on
FROM products WHERE tenant_id = $1`
	args := []interface{}{tenantID}
	argPos := 2
	if activeOnly {
		q += " AND status = 'active'"
	}
	if namePrefix != nil && *namePrefix != "" {
		q += fmt.Sprintf(" AND lower(name) LIKE $%d ESCAPE '\\'", argPos)
		args = append(args, *namePrefix)
		argPos++
	}
	if afterID != nil {
		q += fmt.Sprintf(" AND id_product < $%d", argPos)
		args = append(args, *afterID)
		argPos++
	}
	q += fmt.Sprintf(" ORDER BY id_product DESC LIMIT $%d", argPos)
	args = append(args, limit+1)

	rows, err := r.DB.QueryContext(ctx, q, args...)
	if err != nil {
		logger.Err(err).Msg("Error listing products page")
		return ListProductsPageResult{}, err
	}
	defer rows.Close()

	var products []pModel.Product
	for rows.Next() {
		var product pModel.Product
		var imageURLsJSON sql.NullString
		var thumbnailURL sql.NullString
		if err := rows.Scan(
			&product.ID,
			&product.TenantID,
			&product.Name,
			&product.Description,
			&product.Price,
			&product.TrackInventory,
			&product.Stock,
			&product.Status,
			&imageURLsJSON,
			&thumbnailURL,
			&product.CreatedOn,
		); err != nil {
			logger.Err(err).Msg("Error mapping the products")
			return ListProductsPageResult{}, err
		}

		if imageURLsJSON.Valid && imageURLsJSON.String != "" {
			var imageURLs []string
			if err := json.Unmarshal([]byte(imageURLsJSON.String), &imageURLs); err != nil {
				logger.Warn().Err(err).Uint64("product_id", product.ID).Msg("Error parsing image URLs")
				product.ImageURLs = []string{}
			} else {
				product.ImageURLs = imageURLs
			}
		} else {
			product.ImageURLs = []string{}
		}

		if thumbnailURL.Valid {
			product.ThumbnailURL = thumbnailURL.String
		}

		products = append(products, product)
	}
	if err := rows.Err(); err != nil {
		return ListProductsPageResult{}, err
	}

	if products == nil {
		products = []pModel.Product{}
	}

	hasNext := len(products) > limit
	if hasNext {
		products = products[:limit]
	}

	var next *string
	if hasNext && len(products) > 0 {
		last := products[len(products)-1].ID
		enc, err := pagination.EncodeIDCursor(last)
		if err != nil {
			return ListProductsPageResult{}, fmt.Errorf("encoding next cursor: %w", err)
		}
		next = &enc
	}

	logger.Debug().Int("count", len(products)).Bool("has_next", hasNext).Msg("Products page retrieved")
	return ListProductsPageResult{Items: products, NextCursor: next}, nil
}

// Getting a product by its ID for a tenant.
// If activeOnly is true, only returns the product when status = 'active'.
func (r *ProductRepository) GetProductByID(_ context.Context, tenantID, idProduct uint64, activeOnly bool) (pModel.Product, error) {
	logger.Debug().Uint64("tenant_id", tenantID).Uint64("product_id", idProduct).Msg("Getting product by id")

	product := pModel.Product{}
	var imageURLsJSON sql.NullString
	var thumbnailURL sql.NullString

	q := "SELECT id_product, tenant_id, name, description, price, track_inventory, stock, status, image_urls, thumbnail_url, created_on FROM products WHERE tenant_id = $1 AND id_product = $2"
	if activeOnly {
		q += " AND status = 'active'"
	}

	err := r.DB.QueryRow(
		q,
		tenantID,
		idProduct,
	).Scan(
		&product.ID,
		&product.TenantID,
		&product.Name,
		&product.Description,
		&product.Price,
		&product.TrackInventory,
		&product.Stock,
		&product.Status,
		&imageURLsJSON,
		&thumbnailURL,
		&product.CreatedOn,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			logger.Debug().Uint64("product_id", idProduct).Msg("Product not found")
			return product, errors.NewNotFound(errors.ErrProductNotFound)
		}
		logger.Err(err).Uint64("product_id", idProduct).Msg("Error retrieving the product")
		return product, errors.NewNotFound(errors.ErrCouldNotGetTheProduct)
	}

	// Parse image URLs from JSON
	if imageURLsJSON.Valid && imageURLsJSON.String != "" {
		var imageURLs []string
		if err := json.Unmarshal([]byte(imageURLsJSON.String), &imageURLs); err != nil {
			logger.Warn().Err(err).Uint64("product_id", idProduct).Msg("Error parsing image URLs")
			product.ImageURLs = []string{}
		} else {
			product.ImageURLs = imageURLs
		}
	} else {
		product.ImageURLs = []string{}
	}

	if thumbnailURL.Valid {
		product.ThumbnailURL = thumbnailURL.String
	}

	logger.Debug().Uint64("product_id", idProduct).Str("name", product.Name).Msg("Product retrieved successfully")
	return product, nil
}

// Getting multiple products by their IDs for a tenant
func (r *ProductRepository) GetProductsByIDs(ctx context.Context, tenantID uint64, ids []uint64) ([]pModel.Product, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	// Generate placeholders
	placeholders := make([]string, len(ids))
	args := make([]interface{}, 0, len(ids)+1)
	args = append(args, tenantID)
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args = append(args, id)
	}

	query := fmt.Sprintf(`
		SELECT id_product, tenant_id, name, price, stock, status, track_inventory 
		FROM products 
		WHERE tenant_id = $1 AND id_product IN (%s)`, strings.Join(placeholders, ","))

	rows, err := r.DB.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("error querying products: %w", err)
	}
	defer rows.Close()

	products := []pModel.Product{}
	for rows.Next() {
		var p pModel.Product
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Price, &p.Stock, &p.Status, &p.TrackInventory); err != nil {
			return nil, fmt.Errorf("error scanning product: %w", err)
		}
		products = append(products, p)
	}

	return products, nil
}

// Creating a product for a tenant
func (r *ProductRepository) CreateProduct(_ context.Context, tenantID uint64, product pModel.Product) (pModel.Product, error) {
	logger.Debug().
		Str("name", product.Name).
		Float64("price", product.Price).
		Msg("Creating product")

	createdProduct := pModel.Product{}

	if product.Name == "" || product.Description == "" {
		logger.Warn().
			Str("name", product.Name).
			Msg("Invalid product data for creation")
		return createdProduct, errors.NewBadRequest(errors.ErrCreatingProduct)
	}

	// Convert imageURLs to JSON
	imageURLsJSON, err := json.Marshal(product.ImageURLs)
	if err != nil {
		logger.Err(err).
			Str("name", product.Name).
			Msg("Error marshaling image URLs")
		return createdProduct, errors.NewInternalServerError(errors.ErrCreatingProduct)
	}

	var thumbnailValue interface{}
	if product.ThumbnailURL == "" {
		thumbnailValue = nil
	} else {
		thumbnailValue = product.ThumbnailURL
	}

	var insertedID uint64
	err = r.DB.QueryRow(
		`INSERT INTO products 
		(tenant_id, name, description, price, track_inventory, stock, status, image_urls, thumbnail_url) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) RETURNING id_product`,
		tenantID,
		product.Name, product.Description, product.Price, product.TrackInventory, product.Stock, product.Status, string(imageURLsJSON), thumbnailValue).Scan(&insertedID)

	if err != nil {
		logger.Err(err).
			Str("name", product.Name).
			Msg("Error getting the last insert ID")
		return createdProduct, errors.NewInternalServerError(errors.ErrCreatingProduct)
	}

	createdProduct.ID = insertedID
	createdProduct.TenantID = tenantID
	createdProduct.Name = product.Name
	createdProduct.Description = product.Description
	createdProduct.Price = product.Price
	createdProduct.TrackInventory = product.TrackInventory
	createdProduct.Stock = product.Stock
	createdProduct.Status = product.Status
	createdProduct.ImageURLs = product.ImageURLs
	createdProduct.ThumbnailURL = product.ThumbnailURL

	logger.Info().
		Uint64("product_id", insertedID).
		Str("name", product.Name).
		Msg("Product created successfully")
	return createdProduct, nil
}

// Updating a product status for a tenant
func (r *ProductRepository) UpdateProductStatus(_ context.Context, tenantID, idProduct uint64, status pModel.ProductStatus) error {
	logger.Debug().
		Uint64("product_id", idProduct).
		Str("status", string(status)).
		Msg("Updating product status")

	validStatus := IsValidStatus(status)
	if !validStatus {
		logger.Warn().
			Uint64("product_id", idProduct).
			Str("status", string(status)).
			Msg("Invalid status")
		return errors.NewBadRequest(errors.ErrInvalidStatus)
	}

	result, err := r.DB.Exec(
		"UPDATE products SET status = $1 WHERE tenant_id = $2 AND id_product = $3",
		status,
		tenantID,
		idProduct,
	)

	if err != nil {
		logger.Err(err).
			Uint64("product_id", idProduct).
			Str("status", string(status)).
			Msg("Error updating the product status")
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		logger.Warn().Err(err).
			Uint64("product_id", idProduct).
			Msg("Could not get the rows affected")
	}

	if rows == 0 {
		logger.Debug().
			Uint64("product_id", idProduct).
			Msg("Product not found for status update")
		return errors.NewNotFound(errors.ErrProductNotFound)
	}

	logger.Info().
		Uint64("product_id", idProduct).
		Str("status", string(status)).
		Int64("rows_affected", rows).
		Msg("Product status updated successfully")

	return nil
}

// Updating product for a tenant
func (r *ProductRepository) UpdateProduct(_ context.Context, tenantID uint64, product pModel.Product) error {
	logger.Debug().
		Uint64("product_id", product.ID).
		Str("name", product.Name).
		Msg("Updating product")

	validStatus := IsValidStatus(product.Status)
	if !validStatus {
		logger.Warn().
			Uint64("product_id", product.ID).
			Str("status", string(product.Status)).
			Msg("Invalid status")
		return errors.NewBadRequest(errors.ErrInvalidStatus)
	}

	var thumbnailValue interface{}
	if product.ThumbnailURL == "" {
		thumbnailValue = nil
	} else {
		thumbnailValue = product.ThumbnailURL
	}

	result, err := r.DB.Exec(
		"UPDATE products SET name = $1, description = $2, price = $3, stock = $4, status = $5, track_inventory = $6, thumbnail_url = $7 WHERE tenant_id = $8 AND id_product = $9",
		product.Name,
		product.Description,
		product.Price,
		product.Stock,
		product.Status,
		product.TrackInventory,
		thumbnailValue,
		tenantID,
		product.ID,
	)

	if err != nil {
		logger.Err(err).
			Uint64("product_id", product.ID).
			Msg("Error updating the product")
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		logger.Warn().Err(err).
			Uint64("product_id", product.ID).
			Msg("Could not get the rows affected")
	}

	if rows == 0 {
		logger.Debug().
			Uint64("product_id", product.ID).
			Msg("Product not found for update")
		return errors.NewNotFound(errors.ErrProductNotFound)
	}

	logger.Info().
		Uint64("product_id", product.ID).
		Int64("rows_affected", rows).
		Msg("Product updated successfully")

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

// UpdateProductStock updates only the stock of a product for a tenant
func (r *ProductRepository) UpdateProductStock(_ context.Context, tenantID, idProduct uint64, newStock uint64) error {
	logger.Debug().
		Uint64("product_id", idProduct).
		Uint64("new_stock", newStock).
		Msg("Updating product stock")

	result, err := r.DB.Exec(
		"UPDATE products SET stock = $1 WHERE tenant_id = $2 AND id_product = $3",
		newStock,
		tenantID,
		idProduct,
	)

	if err != nil {
		logger.Err(err).
			Uint64("product_id", idProduct).
			Uint64("new_stock", newStock).
			Msg("Error updating product stock")
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		logger.Warn().Err(err).
			Uint64("product_id", idProduct).
			Msg("Could not get the rows affected")
	}

	if rows == 0 {
		logger.Debug().
			Uint64("product_id", idProduct).
			Msg("Product not found for stock update")
		return errors.NewNotFound(errors.ErrProductNotFound)
	}

	logger.Info().
		Uint64("product_id", idProduct).
		Uint64("new_stock", newStock).
		Int64("rows_affected", rows).
		Msg("Stock updated successfully")
	return nil
}

// RevertProductStock adds stock back to a product (used when orders are cancelled)
func (r *ProductRepository) RevertProductStock(ctx context.Context, tenantID, idProduct uint64, quantityToRevert uint64) error {
	if quantityToRevert == 0 {
		return nil // Nothing to revert
	}

	// Get current product to verify it exists and get current stock
	product, err := r.GetProductByID(ctx, tenantID, idProduct, false)
	if err != nil {
		return fmt.Errorf("error getting product for stock revert: %w", err)
	}

	if !product.TrackInventory {
		return nil
	}

	// Calculate new stock (current + quantity to revert)
	newStock := product.Stock + quantityToRevert

	// Update the stock
	err = r.UpdateProductStock(ctx, tenantID, idProduct, newStock)
	if err != nil {
		return fmt.Errorf("error reverting product stock: %w", err)
	}

	logger.Info().
		Uint64("product_id", idProduct).
		Uint64("previous_stock", product.Stock).
		Uint64("quantity_reverted", quantityToRevert).
		Uint64("new_stock", newStock).
		Msg("Stock reverted successfully")

	return nil
}

// RevertProductStockTx adds stock back to a product within a transaction (atomic increment; for use in CancelExpiredOrders).
// Only increments when track_inventory is true. Unlimited products (track_inventory=false) succeed as a no-op.
func (r *ProductRepository) RevertProductStockTx(ctx context.Context, tx *sql.Tx, tenantID, idProduct uint64, quantityToRevert uint64) error {
	if quantityToRevert == 0 {
		return nil
	}
	result, err := tx.ExecContext(ctx,
		"UPDATE products SET stock = stock + $1 WHERE tenant_id = $2 AND id_product = $3 AND track_inventory = true",
		quantityToRevert, tenantID, idProduct,
	)
	if err != nil {
		return fmt.Errorf("error reverting product stock in tx: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("error getting rows affected: %w", err)
	}
	if rows == 0 {
		var trackInventory bool
		err := tx.QueryRowContext(ctx,
			"SELECT track_inventory FROM products WHERE tenant_id = $1 AND id_product = $2",
			tenantID, idProduct,
		).Scan(&trackInventory)
		if err == sql.ErrNoRows {
			return fmt.Errorf("product %d not found for stock revert", idProduct)
		}
		if err != nil {
			return fmt.Errorf("error checking product for stock revert: %w", err)
		}
		if !trackInventory {
			return nil
		}
		return fmt.Errorf("product %d not found for stock revert", idProduct)
	}
	return nil
}

// DecrementProductStockTx atomically decrements product stock within a transaction for a tenant.
// It updates only if the product is active and stock >= quantity (prevents overselling and
// racing with deactivation). Returns rows affected (1 = success, 0 = insufficient stock or not active).
// Callers should skip this when track_inventory is false; use AssertProductActiveTx instead.
func (r *ProductRepository) DecrementProductStockTx(ctx context.Context, tx *sql.Tx, tenantID, idProduct uint64, quantity uint64) (int64, error) {
	if quantity == 0 {
		return 1, nil
	}
	result, err := tx.ExecContext(ctx,
		"UPDATE products SET stock = stock - $1 WHERE tenant_id = $2 AND id_product = $3 AND stock >= $1 AND status = 'active'",
		quantity, tenantID, idProduct,
	)
	if err != nil {
		return 0, fmt.Errorf("error decrementing product stock in tx: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("error getting rows affected: %w", err)
	}
	return rows, nil
}

// AssertProductActiveTx locks the product row, ensures it is purchasable (status=active),
// and returns the current track_inventory flag under that lock (not a pre-tx snapshot).
func (r *ProductRepository) AssertProductActiveTx(ctx context.Context, tx *sql.Tx, tenantID, idProduct uint64) (bool, error) {
	var status pModel.ProductStatus
	var trackInventory bool
	err := tx.QueryRowContext(ctx,
		"SELECT status, track_inventory FROM products WHERE tenant_id = $1 AND id_product = $2 FOR UPDATE",
		tenantID, idProduct,
	).Scan(&status, &trackInventory)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, errors.ErrProductNotFound
		}
		return false, fmt.Errorf("error locking product for purchase: %w", err)
	}
	if status != pModel.StatusActive {
		return false, errors.ErrProductNotPurchasable
	}
	return trackInventory, nil
}

// UpdateProductImages replaces image_urls and thumbnail_url under a FOR UPDATE lock
// so concurrent append/delete/replace operations serialize on the product row.
func (r *ProductRepository) UpdateProductImages(
	ctx context.Context,
	tenantID, idProduct uint64,
	imageURLs []string,
	thumbnailURL string,
) error {
	logger.Debug().
		Uint64("product_id", idProduct).
		Int("image_count", len(imageURLs)).
		Msg("Updating product images")

	imageURLsJSON, err := json.Marshal(imageURLs)
	if err != nil {
		logger.Err(err).
			Uint64("product_id", idProduct).
			Msg("Error marshaling image URLs")
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	var thumbnailValue interface{}
	if thumbnailURL == "" {
		thumbnailValue = nil
	} else {
		thumbnailValue = thumbnailURL
	}

	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var existing string
	err = tx.QueryRowContext(
		ctx,
		"SELECT image_urls FROM products WHERE tenant_id = $1 AND id_product = $2 FOR UPDATE",
		tenantID,
		idProduct,
	).Scan(&existing)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.NewNotFound(errors.ErrProductNotFound)
		}
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	result, err := tx.ExecContext(
		ctx,
		"UPDATE products SET image_urls = $1, thumbnail_url = $2 WHERE tenant_id = $3 AND id_product = $4",
		string(imageURLsJSON),
		thumbnailValue,
		tenantID,
		idProduct,
	)
	if err != nil {
		logger.Err(err).
			Uint64("product_id", idProduct).
			Msg("Error updating product images")
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		logger.Warn().Err(err).
			Uint64("product_id", idProduct).
			Msg("Could not get the rows affected")
	}
	if rows == 0 {
		return errors.NewNotFound(errors.ErrProductNotFound)
	}

	if err := tx.Commit(); err != nil {
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}
	tx = nil

	logger.Info().
		Uint64("product_id", idProduct).
		Int("image_count", len(imageURLs)).
		Int64("rows_affected", rows).
		Msg("Product images updated successfully")
	return nil
}

// RemoveProductImage removes one image URL under FOR UPDATE and recomputes thumbnail.
// Concurrent appends wait on the lock, so newly added URLs are not dropped.
func (r *ProductRepository) RemoveProductImage(
	ctx context.Context,
	tenantID, idProduct uint64,
	imageURL string,
) ([]string, string, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var imageURLsJSON sql.NullString
	var thumbnailURL sql.NullString
	err = tx.QueryRowContext(
		ctx,
		"SELECT image_urls, thumbnail_url FROM products WHERE tenant_id = $1 AND id_product = $2 FOR UPDATE",
		tenantID,
		idProduct,
	).Scan(&imageURLsJSON, &thumbnailURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", errors.NewNotFound(errors.ErrProductNotFound)
		}
		return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	imageURLs := []string{}
	if imageURLsJSON.Valid && imageURLsJSON.String != "" {
		if err := json.Unmarshal([]byte(imageURLsJSON.String), &imageURLs); err != nil {
			return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
		}
	}

	imageIndex := -1
	for i, url := range imageURLs {
		if url == imageURL {
			imageIndex = i
			break
		}
	}
	if imageIndex == -1 {
		return nil, "", errors.NewNotFound(errors.ErrImageNotInProduct)
	}

	newImageURLs := make([]string, 0, len(imageURLs)-1)
	newImageURLs = append(newImageURLs, imageURLs[:imageIndex]...)
	newImageURLs = append(newImageURLs, imageURLs[imageIndex+1:]...)

	currentThumbnail := ""
	if thumbnailURL.Valid {
		currentThumbnail = thumbnailURL.String
	}
	newThumbnail := selectThumbnailURL(currentThumbnail, newImageURLs)

	updatedJSON, err := json.Marshal(newImageURLs)
	if err != nil {
		return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	var thumbnailValue interface{}
	if newThumbnail == "" {
		thumbnailValue = nil
	} else {
		thumbnailValue = newThumbnail
	}

	result, err := tx.ExecContext(
		ctx,
		"UPDATE products SET image_urls = $1, thumbnail_url = $2 WHERE tenant_id = $3 AND id_product = $4",
		string(updatedJSON),
		thumbnailValue,
		tenantID,
		idProduct,
	)
	if err != nil {
		return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}
	if rows == 0 {
		return nil, "", errors.NewNotFound(errors.ErrProductNotFound)
	}

	if err := tx.Commit(); err != nil {
		return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}
	tx = nil
	return newImageURLs, newThumbnail, nil
}

func selectThumbnailURL(current string, imageURLs []string) string {
	if len(imageURLs) == 0 {
		return ""
	}
	for _, url := range imageURLs {
		if url == current {
			return current
		}
	}
	return imageURLs[0]
}

// UpdateProductThumbnail updates only the thumbnail_url column for a product
func (r *ProductRepository) UpdateProductThumbnail(_ context.Context, tenantID, idProduct uint64, thumbnailURL string) error {
	logger.Debug().
		Uint64("product_id", idProduct).
		Str("thumbnail_url", thumbnailURL).
		Msg("Updating product thumbnail")

	var thumbnailValue interface{}
	if thumbnailURL == "" {
		thumbnailValue = nil
	} else {
		thumbnailValue = thumbnailURL
	}

	result, err := r.DB.Exec(
		"UPDATE products SET thumbnail_url = $1 WHERE tenant_id = $2 AND id_product = $3",
		thumbnailValue,
		tenantID,
		idProduct,
	)
	if err != nil {
		logger.Err(err).
			Uint64("product_id", idProduct).
			Msg("Error updating product thumbnail")
		return errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		logger.Warn().Err(err).
			Uint64("product_id", idProduct).
			Msg("Could not get the rows affected for thumbnail update")
	}

	if rows == 0 {
		logger.Debug().
			Uint64("product_id", idProduct).
			Msg("Product not found for thumbnail update")
		return errors.NewNotFound(errors.ErrProductNotFound)
	}

	logger.Info().
		Uint64("product_id", idProduct).
		Str("thumbnail_url", thumbnailURL).
		Int64("rows_affected", rows).
		Msg("Product thumbnail updated successfully")

	return nil
}

// PrependImageAndSetThumbnail prepends an image URL to image_urls and sets thumbnail_url.
func (r *ProductRepository) PrependImageAndSetThumbnail(ctx context.Context, tenantID, idProduct uint64, newURL string) ([]string, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var imageURLsJSON sql.NullString
	err = tx.QueryRowContext(
		ctx,
		"SELECT image_urls FROM products WHERE tenant_id = $1 AND id_product = $2 FOR UPDATE",
		tenantID,
		idProduct,
	).Scan(&imageURLsJSON)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, errors.NewNotFound(errors.ErrProductNotFound)
		}
		return nil, errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	imageURLs := []string{}
	if imageURLsJSON.Valid && imageURLsJSON.String != "" {
		if err := json.Unmarshal([]byte(imageURLsJSON.String), &imageURLs); err != nil {
			return nil, errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
		}
	}

	filtered := make([]string, 0, len(imageURLs))
	for _, url := range imageURLs {
		if url != newURL {
			filtered = append(filtered, url)
		}
	}
	updatedImageURLs := append([]string{newURL}, filtered...)

	updatedJSON, err := json.Marshal(updatedImageURLs)
	if err != nil {
		return nil, errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	result, err := tx.ExecContext(
		ctx,
		"UPDATE products SET image_urls = $1, thumbnail_url = $2 WHERE tenant_id = $3 AND id_product = $4",
		string(updatedJSON),
		newURL,
		tenantID,
		idProduct,
	)
	if err != nil {
		return nil, errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}
	if rows == 0 {
		return nil, errors.NewNotFound(errors.ErrProductNotFound)
	}

	if err := tx.Commit(); err != nil {
		return nil, errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}
	tx = nil
	return updatedImageURLs, nil
}

// AppendProductImages appends image URLs to image_urls under a FOR UPDATE lock.
// Returns the full image list and the current thumbnail_url after the update.
func (r *ProductRepository) AppendProductImages(ctx context.Context, tenantID, idProduct uint64, newURLs []string) ([]string, string, error) {
	tx, err := r.DB.BeginTx(ctx, nil)
	if err != nil {
		return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}
	defer func() {
		if tx != nil {
			_ = tx.Rollback()
		}
	}()

	var imageURLsJSON sql.NullString
	var thumbnailURL sql.NullString
	err = tx.QueryRowContext(
		ctx,
		"SELECT image_urls, thumbnail_url FROM products WHERE tenant_id = $1 AND id_product = $2 FOR UPDATE",
		tenantID,
		idProduct,
	).Scan(&imageURLsJSON, &thumbnailURL)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, "", errors.NewNotFound(errors.ErrProductNotFound)
		}
		return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	imageURLs := []string{}
	if imageURLsJSON.Valid && imageURLsJSON.String != "" {
		if err := json.Unmarshal([]byte(imageURLsJSON.String), &imageURLs); err != nil {
			return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
		}
	}

	updatedImageURLs := append(append([]string{}, imageURLs...), newURLs...)
	updatedJSON, err := json.Marshal(updatedImageURLs)
	if err != nil {
		return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	currentThumbnail := ""
	if thumbnailURL.Valid {
		currentThumbnail = thumbnailURL.String
	}

	var thumbnailValue interface{}
	if currentThumbnail == "" {
		thumbnailValue = nil
	} else {
		thumbnailValue = currentThumbnail
	}

	result, err := tx.ExecContext(
		ctx,
		"UPDATE products SET image_urls = $1, thumbnail_url = $2 WHERE tenant_id = $3 AND id_product = $4",
		string(updatedJSON),
		thumbnailValue,
		tenantID,
		idProduct,
	)
	if err != nil {
		return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}
	if rows == 0 {
		return nil, "", errors.NewNotFound(errors.ErrProductNotFound)
	}

	if err := tx.Commit(); err != nil {
		return nil, "", errors.NewInternalServerError(errors.ErrUpdatingProductStatus)
	}
	tx = nil
	return updatedImageURLs, currentThumbnail, nil
}
