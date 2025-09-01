package auth

import (
	jwt "github.com/golang-jwt/jwt/v5"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

type Service interface {
	HashPassword(plainPwd string) (string, error)
	ComparePasswords(hashPwd string, plainPwd string) error
	GenerateJWT(userID uint64, roleID uModel.UserRole, email string) (string, error)
	ValidateToken(token string) (*jwt.Token, error)
}
