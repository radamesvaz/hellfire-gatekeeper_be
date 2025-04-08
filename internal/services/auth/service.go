package auth

import (
	"fmt"
	"os"
	"strconv"
	"time"

	jwt "github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct{}

func New() *AuthService {
	return &AuthService{}
}

// Comparing the hashed password to the simple password
func (s *AuthService) ComparePasswords(hashedPwd string, plainPwd string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(plainPwd))
}

// Generate a new JWT
func (s *AuthService) GenerateJWT(
	userID uint64,
	roleID uint64,
	email string,
) (string, error) {
	secret := os.Getenv("JWT_SECRET")
	expMinutes := os.Getenv("JWT_EXPIRATION_MINUTES")
	expiration, err := strconv.Atoi(expMinutes)
	if err != nil {
		fmt.Printf("Error getting the expiration")
		return "", err
	}

	claims := jwt.MapClaims{
		"user_id": userID,
		"role_id": roleID,
		"email":   email,
		"exp":     time.Now().Add(time.Minute * time.Duration(expiration)).Unix(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}
