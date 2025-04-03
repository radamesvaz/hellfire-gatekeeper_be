package model

import (
	"database/sql"
)

type Product struct {
	ID          int          `json:"id_product" gorm:"primaryKey"`
	Name        string       `json:"name" gorm:"not null;unique"`
	Description string       `json:"description"`
	Price       float64      `json:"price" gorm:"not null;check:price >= 0"`
	Available   bool         `json:"available"`
	CreatedOn   sql.NullTime `json:"created_on"`
}
