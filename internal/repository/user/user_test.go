package user

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/radamesvaz/bakery-app/internal/errors"
	userModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"golang.org/x/crypto/bcrypt"
)

func TestUserRepository_GetUserByEmail(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &UserRepository{DB: db}

	createdOn := sql.NullTime{
		Time:  time.Now(),
		Valid: true,
	}

	password := "adminpass"
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		panic(err)
	}

	tests := []struct {
		name           string
		mockRows       *sqlmock.Rows
		mockError      error
		expected       userModel.User
		expectedError  bool
		emailForLookup string
		errorStatus    int
	}{
		{
			name: "HAPPY PATH: finding an user by its email",
			mockRows: sqlmock.NewRows([]string{
				"id_user",
				"id_role",
				"name",
				"email",
				"password",
				"phone",
				"created_on",
			}).AddRow(
				"1",
				"1",
				"Admin",
				"admin@test.com",
				hashedPassword,
				"55-5555",
				createdOn,
			).AddRow(
				"2",
				"2",
				"client",
				"client@test.com",
				nil,
				"55-5555",
				createdOn,
			),
			mockError:     nil,
			expectedError: false,
			expected: userModel.User{
				ID:        1,
				IDRole:    1,
				Name:      "Admin",
				Email:     "admin@test.com",
				Password:  string(hashedPassword),
				Phone:     "55-5555",
				CreatedOn: createdOn,
			},
			emailForLookup: "admin@test.com",
		},
		{
			name:          "SAD PATH: user not found",
			expectedError: true,
			mockRows: sqlmock.NewRows([]string{
				"id_user",
				"id_role",
				"name",
				"email",
				"password",
				"phone",
				"created_on",
			}).AddRow(
				"1",
				"1",
				"Admin",
				"admin@test.com",
				hashedPassword,
				"55-5555",
				createdOn,
			).AddRow(
				"2",
				"2",
				"client",
				"client@test.com",
				nil,
				"55-5555",
				createdOn,
			),
			mockError: errors.ErrUserNotFound,
			expected: userModel.User{
				ID:        1,
				IDRole:    1,
				Name:      "Admin",
				Email:     "admin@test.com",
				Password:  string(hashedPassword),
				Phone:     "55-5555",
				CreatedOn: createdOn,
			},
			emailForLookup: "nonexistent@test.com",
			errorStatus:    404,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT id_user, id_role, name, email, password_hash, phone, created_on FROM users WHERE email = ?"),
				).
					WithArgs(tt.emailForLookup).
					WillReturnError(sql.ErrNoRows)
			} else {
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT id_user, id_role, name, email, password_hash, phone, created_on FROM users WHERE email = ?"),
				).
					WithArgs(tt.emailForLookup).
					WillReturnRows(tt.mockRows)
			}

			user, err := repo.GetUserByEmail(tt.emailForLookup)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, user)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

func TestUserRepository_CreateUser(t *testing.T) {
	// Setting up mock
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("Error setting up the mock: %v", err)
	}

	defer db.Close()

	repo := &UserRepository{DB: db}

	password := "adminpass"

	if err != nil {
		panic(err)
	}

	tests := []struct {
		name          string
		payload       userModel.CreateUserRequest
		mockError     error
		expected      uint64
		expectedError bool
		errorStatus   int
	}{
		{
			name: "HAPPY PATH: Create a user of ADMIN role",
			payload: userModel.CreateUserRequest{
				IDRole:   userModel.UserRoleAdmin,
				Name:     "Test admin 1",
				Email:    "adminemail@email.com",
				Phone:    "55-88888",
				Password: password,
			},
			mockError:     nil,
			expectedError: false,
			expected:      1,
		},
		{
			name: "HAPPY PATH: Create a user of CLIENT role",
			payload: userModel.CreateUserRequest{
				IDRole: userModel.UserRoleClient,
				Name:   "Test client 1",
				Email:  "client@email.com",
				Phone:  "55-88888",
			},
			mockError:     nil,
			expectedError: false,
			expected:      1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO users (id_role, name, email, password_hash, phone) VALUES (?, ?, ?, ?, ?)"),
				).
					WithArgs(tt.payload.IDRole, tt.payload.Name, tt.payload.Email, tt.payload.Password, tt.payload.Phone).
					WillReturnResult(sqlmock.NewResult(int64(tt.expected), 1))
			} else {
				mock.ExpectExec(
					regexp.QuoteMeta("INSERT INTO users (id_role, name, email, password_hash, phone) VALUES (?, ?, ?, ?, ?)"),
				).
					WithArgs(tt.payload.IDRole, tt.payload.Name, tt.payload.Email, tt.payload.Password, tt.payload.Phone).
					WillReturnResult(sqlmock.NewResult(int64(tt.expected), 1))
			}

			userID, err := repo.CreateUser(context.Background(), tt.payload)
			if tt.expectedError {
				assertHTTPError(t, err, tt.errorStatus, tt.mockError.Error())
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, userID)
			}

			assert.NoError(t, mock.ExpectationsWereMet())
		})
	}
}

// Validates the error to be of *HTTPError type, have the correct status and message
func assertHTTPError(t *testing.T, err error, expectedStatus int, expectedMessage string) {
	httpErr, ok := err.(*errors.HTTPError)

	if assert.True(t, ok, "The error is not HTTP type") {
		assert.Equal(t, expectedStatus, httpErr.StatusCode, "The code status is not as expected")
		assert.EqualError(t, httpErr.Err, expectedMessage, "Mismatch on error message")
	}
}
