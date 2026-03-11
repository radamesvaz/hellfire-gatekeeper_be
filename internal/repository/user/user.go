package user

import (
	"context"
	"database/sql"

	"github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/logger"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

type UserRepository struct {
	DB *sql.DB
}

func NewUserRepository(db *sql.DB) Repository {
	return &UserRepository{DB: db}
}

func (r *UserRepository) GetUserByEmail(email string) (uModel.User, error) {
	logger.Debug().Str("email", email).Msg("Getting user by email")

	user := uModel.User{}

	err := r.DB.QueryRow(
		"SELECT id_user, id_role, name, email, password_hash, phone, created_on, deleted_at FROM users WHERE email = $1 AND deleted_at IS NULL",
		email,
	).Scan(
		&user.ID,
		&user.IDRole,
		&user.Name,
		&user.Email,
		&user.Password,
		&user.Phone,
		&user.CreatedOn,
		&user.DeletedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			logger.Debug().Str("email", email).Msg("User not found")
			return user, errors.ErrUserNotFound
		} else {
			logger.Err(err).Str("email", email).Msg("Could not get the user")
			return user, errors.ErrUserNotFound
		}
	}

	logger.Debug().Uint64("user_id", user.ID).Str("email", email).Msg("User retrieved successfully")
	return user, nil
}

func (r *UserRepository) CreateUser(ctx context.Context, user uModel.CreateUserRequest) (id uint64, err error) {
	logger.Debug().
		Str("email", user.Email).
		Str("name", user.Name).
		Uint64("role_id", uint64(user.IDRole)).
		Msg("Creating user")

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
		logger.Err(err).
			Str("email", user.Email).
			Msg("Error creating the user")
		return 0, errors.NewInternalServerError(errors.ErrCreatingUser)
	}

	logger.Info().
		Uint64("user_id", insertedID).
		Str("email", user.Email).
		Msg("User created successfully")
	return insertedID, nil
}

// EmailExists checks if an email already exists in the database
func (r *UserRepository) EmailExists(email string) (bool, error) {
	logger.Debug().Str("email", email).Msg("Checking if email exists")

	var count int
	err := r.DB.QueryRow(
		"SELECT COUNT(*) FROM users WHERE email = $1 AND deleted_at IS NULL",
		email,
	).Scan(&count)

	if err != nil {
		logger.Err(err).Str("email", email).Msg("Error checking email existence")
		return false, errors.NewInternalServerError(errors.ErrCouldNotGetTheUser)
	}

	exists := count > 0
	logger.Debug().
		Str("email", email).
		Bool("exists", exists).
		Msg("Email existence checked")
	return exists, nil
}
