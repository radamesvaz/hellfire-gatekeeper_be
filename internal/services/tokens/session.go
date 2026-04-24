package tokens

import (
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

// SessionClaims represents the claims required by the current auth flow.
type SessionClaims struct {
	UserID   uint64
	RoleID   uModel.UserRole
	Email    string
	TenantID *uint64
}

// SessionTokenManager abstracts session token behavior (JWT today).
type SessionTokenManager interface {
	Generate(claims SessionClaims) (string, error)
	Validate(token string) (*jwt.Token, error)
}

// JWTSessionTokenManager provides HS256 JWT generation/validation.
type JWTSessionTokenManager struct {
	secret     string
	expiration time.Duration
}

func NewJWTSessionTokenManager(secret string, expirationMinutes int) *JWTSessionTokenManager {
	return &JWTSessionTokenManager{
		secret:     secret,
		expiration: time.Minute * time.Duration(expirationMinutes),
	}
}

func (m *JWTSessionTokenManager) Generate(claims SessionClaims) (string, error) {
	jwtClaims := jwt.MapClaims{
		"user_id": claims.UserID,
		"role_id": claims.RoleID,
		"email":   claims.Email,
		"exp":     time.Now().Add(m.expiration).Unix(),
	}

	if claims.TenantID != nil {
		jwtClaims["tenant_id"] = *claims.TenantID
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwtClaims)
	return token.SignedString([]byte(m.secret))
}

func (m *JWTSessionTokenManager) Validate(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(m.secret), nil
	})
}
