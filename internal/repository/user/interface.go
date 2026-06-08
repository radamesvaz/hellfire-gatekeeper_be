package user

import (
	"context"

	uModel "github.com/radamesvaz/bakery-app/model/users"
)

type Repository interface {
	GetUserByEmail(tenantID uint64, email string) (uModel.User, error)
	GetUserByTenantAndEmail(tenantID uint64, email string) (uModel.User, error)
	CreateUser(ctx context.Context, user uModel.CreateUserRequest) (id uint64, err error)
	ReactivateUser(ctx context.Context, tenantID, userID uint64, req uModel.ReactivateUserRequest) error
	EmailExists(tenantID uint64, email string) (bool, error)
}
