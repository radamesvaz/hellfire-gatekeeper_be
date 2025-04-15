package user

import (
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
			return user, errors.ErrUserNotFound
		} else {
			fmt.Printf("Could not get the user: %v", err)
			return user, errors.ErrCouldNotGetTheUser
		}
	}

	return user, nil
}
