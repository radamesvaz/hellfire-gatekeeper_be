package user

import (
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
				Email:     "Admin@test.com",
				Password:  string(hashedPassword),
				Phone:     "55-5555",
				CreatedOn: createdOn,
			},
			emailForLookup: "Admin@test.com",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.expectedError {
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM users WHERE email = ?"),
				).
					WithArgs(tt.emailForLookup).
					WillReturnError(sql.ErrNoRows)
			} else {
				mock.ExpectQuery(
					regexp.QuoteMeta("SELECT * FROM users WHERE email = ?"),
				).
					WithArgs(tt.emailForLookup).
					WillReturnRows(sqlmock.NewRows([]string{
						"id_user",
						"id_role",
						"name",
						"email",
						"password",
						"phone",
						"created_on",
					}).
						AddRow(
							tt.expected.ID,
							tt.expected.IDRole,
							tt.expected.Name,
							tt.expected.Email,
							tt.expected.Password,
							tt.expected.Phone,
							tt.expected.CreatedOn.Time,
						),
					)
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

// Validates the error to be of *HTTPError type, have the correct status and message
func assertHTTPError(t *testing.T, err error, expectedStatus int, expectedMessage string) {
	httpErr, ok := err.(*errors.HTTPError)

	if assert.True(t, ok, "The error is not HTTP type") {
		assert.Equal(t, expectedStatus, httpErr.StatusCode, "The code status is not as expected")
		assert.EqualError(t, httpErr.Err, expectedMessage, "Mismatch on error message")
	}
}
