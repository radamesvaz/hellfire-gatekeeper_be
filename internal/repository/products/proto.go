package products

import (
	"time"

	pModel "github.com/radamesvaz/bakery-app/model/products"
)

type ProductResponse struct {
	ID          int        `json:"id_product" gorm:"primaryKey"`
	Name        string     `json:"name" gorm:"not null;unique"`
	Description string     `json:"description"`
	Price       float64    `json:"price" gorm:"not null;check:price >= 0"`
	Available   bool       `json:"available"`
	CreatedOn   *time.Time `json:"created_on,omitempty"`
}

// Marshal the product to ProductResponse
func Marshal(product *pModel.Product) ProductResponse {
	response := ProductResponse{
		ID:          product.ID,
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		Available:   product.Available,
	}

	if product.CreatedOn.Valid {
		response.CreatedOn = &product.CreatedOn.Time
	}

	return response
}
