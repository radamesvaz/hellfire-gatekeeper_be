package model

import (
	"database/sql"
)

type User struct {
	ID        uint64       `json:"id_user" gorm:"primaryKey"`
	IDRole    uint64       `json:"role"`
	Name      string       `json:"name" gorm:"not null"`
	Email     string       `json:"email" gorm:"not null;unique"`
	Phone     string       `json:"phone"`
	Password  string       `json:"password_hash"`
	CreatedOn sql.NullTime `json:"created_on"`
}
