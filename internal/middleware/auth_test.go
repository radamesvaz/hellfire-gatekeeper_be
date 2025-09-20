package middleware

import (
	"context"
	"testing"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

func TestGetUserRoleFromContext(t *testing.T) {
	tests := []struct {
		name         string
		claims       jwt.MapClaims
		expectedRole uint64
		expectError  bool
	}{
		{
			name: "Valid admin role",
			claims: jwt.MapClaims{
				"user_id": float64(1),
				"role_id": float64(1), // Admin role
				"email":   "admin@example.com",
			},
			expectedRole: 1,
			expectError:  false,
		},
		{
			name: "Valid client role",
			claims: jwt.MapClaims{
				"user_id": float64(2),
				"role_id": float64(2), // Client role
				"email":   "client@example.com",
			},
			expectedRole: 2,
			expectError:  false,
		},
		{
			name: "Missing role_id in claims",
			claims: jwt.MapClaims{
				"user_id": float64(1),
				"email":   "admin@example.com",
			},
			expectedRole: 0,
			expectError:  true,
		},
		{
			name: "Invalid role_id type",
			claims: jwt.MapClaims{
				"user_id": float64(1),
				"role_id": "invalid", // Should be float64
				"email":   "admin@example.com",
			},
			expectedRole: 0,
			expectError:  true,
		},
		{
			name:         "No claims in context",
			claims:       nil,
			expectedRole: 0,
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.claims != nil {
				ctx = context.WithValue(ctx, UserClaimsKey, tt.claims)
			}

			role, err := GetUserRoleFromContext(ctx)

			if tt.expectError {
				assert.Error(t, err)
				assert.Equal(t, uint64(0), role)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedRole, role)
			}
		})
	}
}

func TestGetUserRoleFromContext_AdminRole(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserClaimsKey, jwt.MapClaims{
		"user_id": float64(1),
		"role_id": float64(1), // Admin role
		"email":   "admin@example.com",
	})

	role, err := GetUserRoleFromContext(ctx)

	assert.NoError(t, err)
	assert.Equal(t, uint64(1), role)
}

func TestGetUserRoleFromContext_ClientRole(t *testing.T) {
	ctx := context.WithValue(context.Background(), UserClaimsKey, jwt.MapClaims{
		"user_id": float64(2),
		"role_id": float64(2), // Client role
		"email":   "client@example.com",
	})

	role, err := GetUserRoleFromContext(ctx)

	assert.NoError(t, err)
	assert.Equal(t, uint64(2), role)
}
