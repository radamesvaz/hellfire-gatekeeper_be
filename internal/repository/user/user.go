package user

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/radamesvaz/bakery-app/internal/errors"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

type UserRepository struct {
	DB *sql.DB
}

func (r *UserRepository) GetUserByEmail(email string) (uModel.User, error) {
	fmt.Printf("Getting user by email")
	// Add email validation

	user := uModel.User{}

	err := r.DB.QueryRow(
		"SELECT id_user, id_role, name, email, password_hash, phone, created_on FROM users WHERE email = ?",
		email,
	).Scan(
		&user.ID,
		&user.IDRole,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Phone,
		&user.CreatedOn,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			fmt.Printf("User not found: %v", err)
			return user, errors.NewNotFound(errors.ErrUserNotFound)
		} else {
			fmt.Printf("Could not get the user: %v", err)
			return user, errors.NewNotFound(errors.ErrCouldNotGetTheUser)
		}
	}

	return user, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, user uModel.CreateUserRequest) (id uint64, err error) {
	fmt.Printf("Creating user")
	// Add email validation

	query := `INSERT INTO users (id_role, name, email, password_hash, phone) VALUES (?, ?, ?, ?, ?)`

	result, err := r.DB.ExecContext(
		ctx,
		query,
		user.IDRole,
		user.Name,
		user.Email,
		user.Password,
		user.Phone,
	)

	if err != nil {
		fmt.Printf("Error creating the user: %v", err)
		return 0, errors.NewInternalServerError(errors.ErrCreatingUser)
	}

	insertedID, err := result.LastInsertId()
	if err != nil {
		fmt.Printf("Error getting the last insert ID: %v", err)
		return 0, errors.NewInternalServerError(errors.ErrGettingTheUserID)
	}

	return uint64(insertedID), nil
}
