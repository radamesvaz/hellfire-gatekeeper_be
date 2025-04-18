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
	ID          int           `json:"id_product_history" gorm:"primaryKey"`
	IDProduct   int           `json:"id_product" gorm:"primaryKey"`
	Name        string        `json:"name" gorm:"not null;unique"`
	Description string        `json:"description"`
	Price       float64       `json:"price" gorm:"not null;check:price >= 0"`
	Available   bool          `json:"available"`
	Status      string        `json:"status"`
	ModifiedOn  sql.NullTime  `json:"modified_on"`
	ModifiedBy  int           `json:"modified_by"`
	Action      ProductAction `json:"action"`
}
