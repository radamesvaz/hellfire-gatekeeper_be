package validators

import (
	"net/http"
	"testing"

	"github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name         string
		password     string
		expectError  bool
		expectedCode int
	}{
		{
			name:        "HAPPY PATH: valid strong password",
			password:    "MyPassword123!",
			expectError: false,
		},
		{
			name:         "ERROR PATH: password too short",
			password:     "Pass1!",
			expectError:  true,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "ERROR PATH: no uppercase letter",
			password:     "mypassword123!",
			expectError:  true,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "ERROR PATH: no lowercase letter",
			password:     "MYPASSWORD123!",
			expectError:  true,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "ERROR PATH: no digit",
			password:     "MyPassword!",
			expectError:  true,
			expectedCode: http.StatusBadRequest,
		},
		{
			name:         "ERROR PATH: no special character",
			password:     "MyPassword123",
			expectError:  true,
			expectedCode: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)

			if tt.expectError {
				assert.Error(t, err)

				// Verify it's an HTTPError with correct status code
				httpErr, ok := err.(*errors.HTTPError)
				assert.True(t, ok, "Expected HTTPError type")
				assert.Equal(t, tt.expectedCode, httpErr.StatusCode)
				assert.Equal(t, errors.ErrWeakPassword, httpErr.Err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIsValidEmail(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		expected bool
	}{
		{
			name:     "HAPPY PATH: valid email",
			email:    "test@example.com",
			expected: true,
		},
		{
			name:     "ERROR PATH: invalid email format",
			email:    "invalid-email",
			expected: false,
		},
		{
			name:     "ERROR PATH: missing @ symbol",
			email:    "testexample.com",
			expected: false,
		},
		{
			name:     "ERROR PATH: missing domain",
			email:    "test@",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidEmail(tt.email)
			assert.Equal(t, tt.expected, result)
		})
	}
}
