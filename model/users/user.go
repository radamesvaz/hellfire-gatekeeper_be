package model

import (
	"database/sql"
)

type UserRole uint64

const (
	UserRoleAdmin  UserRole = 1
	UserRoleClient UserRole = 2
)

type User struct {
	ID        uint64       `json:"id_user" gorm:"primaryKey"`
	IDRole    UserRole     `json:"role"`
	Name      string       `json:"name" gorm:"not null"`
	Email     string       `json:"email" gorm:"not null;unique"`
	Phone     string       `json:"phone"`
	Password  string       `json:"password_hash"`
	CreatedOn sql.NullTime `json:"created_on"`
}

type CreateUserRequest struct {
	IDRole   UserRole `json:"role"`
	Name     string   `json:"name" gorm:"not null"`
	Email    string   `json:"email" gorm:"not null;unique"`
	Phone    string   `json:"phone"`
	Password string   `json:"password_hash"`
}
