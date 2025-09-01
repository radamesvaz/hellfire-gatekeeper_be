package user

import (
	"context"

	uModel "github.com/radamesvaz/bakery-app/model/users"
)

type Repository interface {
	GetUserByEmail(email string) (uModel.User, error)
	CreateUser(ctx context.Context, user uModel.CreateUserRequest) (id uint64, err error)
	EmailExists(email string) (bool, error)
}
