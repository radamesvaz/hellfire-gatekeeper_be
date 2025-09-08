package model

import (
	"database/sql"
)

type ProductAction string

const (
	ActionCreate ProductAction = "create"
	ActionUpdate ProductAction = "update"
	ActionDelete ProductAction = "delete"
)

type ProductHistory struct {
	ID          uint64        `json:"id_product_history" gorm:"primaryKey"`
	IDProduct   uint64        `json:"id_product" gorm:"primaryKey"`
	Name        string        `json:"name" gorm:"not null;unique"`
	Description string        `json:"description"`
	Price       float64       `json:"price" gorm:"not null;check:price >= 0"`
	Available   bool          `json:"available"`
	Stock       uint64        `json:"stock"`
	Status      ProductStatus `json:"status"`
	ImageURLs   []string      `json:"image_urls"`
	ModifiedOn  sql.NullTime  `json:"modified_on"`
	ModifiedBy  uint64        `json:"modified_by"`
	Action      ProductAction `json:"action"`
}
