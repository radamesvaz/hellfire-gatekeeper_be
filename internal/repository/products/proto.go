package products

import (
	"time"

	pModel "github.com/radamesvaz/bakery-app/model/products"
)

// ProductResponse struct
type ProductResponse struct {
	ID          uint64               `json:"id_product" gorm:"primaryKey"`
	Name        string               `json:"name" gorm:"not null;unique"`
	Description string               `json:"description"`
	Price       float64              `json:"price" gorm:"not null;check:price >= 0"`
	Available   bool                 `json:"available"`
	Status      pModel.ProductStatus `json:"status"`
	CreatedOn   *time.Time           `json:"created_on,omitempty"`
}

// Marshal the product to ProductResponse
func Marshal(product *pModel.Product) ProductResponse {
	response := ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Status:      product.Status,
		Available:   product.Available,
	}

	if product.CreatedOn.Valid {
		response.CreatedOn = &product.CreatedOn.Time
	}

	return response
}
