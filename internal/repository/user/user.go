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

func NewUserRepository(db *sql.DB) Repository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) GetUserByEmail(email string) (uModel.User, error) {
	fmt.Printf("Getting user by email")

	user := uModel.User{}

	err := r.DB.QueryRow(
		"SELECT id_user, id_role, name, email, password_hash, phone, created_on FROM users WHERE email = $1",
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
			return user, errors.ErrUserNotFound
		} else {
			fmt.Printf("Could not get the user: %v", err)
			return user, errors.ErrUserNotFound
		}
	}

	return user, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, user uModel.CreateUserRequest) (id uint64, err error) {
	fmt.Printf("Creating user")

	query := `INSERT INTO users (id_role, name, email, password_hash, phone) VALUES ($1, $2, $3, $4, $5) RETURNING id_user`

	var insertedID uint64
	err = r.DB.QueryRowContext(
		ctx,
		query,
		user.IDRole,
		user.Name,
		user.Email,
		user.Password,
		user.Phone,
	).Scan(&insertedID)

	if err != nil {
		fmt.Printf("Error creating the user: %v", err)
		return 0, errors.NewInternalServerError(errors.ErrCreatingUser)
	}

	return insertedID, nil
}

// EmailExists checks if an email already exists in the database
func (r *UserRepository) EmailExists(email string) (bool, error) {
	fmt.Printf("Checking if email exists: %s", email)

	var count int
	err := r.DB.QueryRow(
		"SELECT COUNT(*) FROM users WHERE email = $1",
		email,
	).Scan(&count)

	if err != nil {
		fmt.Printf("Error checking email existence: %v", err)
		return false, errors.NewInternalServerError(errors.ErrCouldNotGetTheUser)
	}

	return count > 0, nil
}
