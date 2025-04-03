package model

import (
	"database/sql"

	model "github.com/radamesvaz/bakery-app/model/roles"
)

type User struct {
	ID        int          `json:"id_product" gorm:"primaryKey"`
	IDRole    model.Roles  `gorm:"foreignKey:IDRole;references:ID" json:"role"`
	Name      string       `json:"name" gorm:"not null"`
	Email     string       `json:"email" gorm:"not null;unique"`
	Phone     string       `json:"phone"`
	Password  string       `json:"password_hash"`
	CreatedOn sql.NullTime `json:"created_on"`
}
