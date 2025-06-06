package auth

import (
	"fmt"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	secret     string
	expiration int
}

func New(secret string, expiration int) *AuthService {
	return &AuthService{
		secret:     secret,
		expiration: expiration,
	}
}

// Comparing the hashed password to the simple password
func (s *AuthService) ComparePasswords(hashedPwd string, plainPwd string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(plainPwd))
}

// Generate a new JWT
func (s *AuthService) GenerateJWT(userID uint64, roleID uint64, email string) (string, error) {
	claims := jwt.MapClaims{
		"user_id": userID,
		"role_id": roleID,
		"email":   email,
		"exp":     time.Now().Add(time.Minute * time.Duration(s.expiration)).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.secret))
}

// Validates the token
func (s *AuthService) ValidateToken(tokenStr string) (*jwt.Token, error) {
	return jwt.Parse(tokenStr, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.secret), nil
	})
}
