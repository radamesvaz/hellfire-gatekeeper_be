package auth

import jwt "github.com/golang-jwt/jwt/v5"

type Service interface {
	ComparePasswords(hashPwd string, plainPwd string) error
	GenerateJWT(userID uint64, roleID uint64, email string) (string, error)
	ValidateToken(token string) (*jwt.Token, error)
}
