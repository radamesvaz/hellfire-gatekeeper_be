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
	TenantID  uint64       `json:"tenant_id,omitempty"`
	IDRole    UserRole     `json:"role"`
	Name      string       `json:"name" gorm:"not null"`
	Email     string       `json:"email" gorm:"not null;unique"`
	Phone     string       `json:"phone"`
	Password  string       `json:"password_hash"`
	CreatedOn sql.NullTime `json:"created_on"`
	DeletedAt sql.NullTime `json:"deleted_at,omitempty"`
}

type CreateUserRequest struct {
	TenantID uint64   `json:"tenant_id"`
	IDRole   UserRole `json:"role"`
	Name     string   `json:"name" gorm:"not null"`
	Email    string   `json:"email" gorm:"not null;unique"`
	Phone    string   `json:"phone"`
	Password string   `json:"password_hash"`
}
