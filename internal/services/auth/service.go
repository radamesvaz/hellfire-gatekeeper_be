package auth

import (
	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/radamesvaz/bakery-app/internal/services/tokens"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	sessionTokenManager tokens.SessionTokenManager
	oneTimeTokenManager tokens.OneTimeTokenManager
}

func New(secret string, expiration int) *AuthService {
	return NewWithManagers(
		tokens.NewJWTSessionTokenManager(secret, expiration),
		tokens.NewSHA256OneTimeTokenManager(32),
	)
}

func NewWithManagers(sessionManager tokens.SessionTokenManager, oneTimeManager tokens.OneTimeTokenManager) *AuthService {
	return &AuthService{
		sessionTokenManager: sessionManager,
		oneTimeTokenManager: oneTimeManager,
	}
}

// Hash a plain password
func (s *AuthService) HashPassword(plainPwd string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(plainPwd), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// Comparing the hashed password to the simple password
func (s *AuthService) ComparePasswords(hashedPwd string, plainPwd string) error {
	return bcrypt.CompareHashAndPassword([]byte(hashedPwd), []byte(plainPwd))
}

// Generate a new JWT
// tenantID is optional: nil means "no tenant" (e.g. superadmin/global user).
func (s *AuthService) GenerateJWT(userID uint64, roleID uModel.UserRole, email string, tenantID *uint64) (string, error) {
	return s.sessionTokenManager.Generate(tokens.SessionClaims{
		UserID:   userID,
		RoleID:   roleID,
		Email:    email,
		TenantID: tenantID,
	})
}

// Validates the token
func (s *AuthService) ValidateToken(tokenStr string) (*jwt.Token, error) {
	return s.sessionTokenManager.Validate(tokenStr)
}

// GenerateOneTimeToken returns a plain token and its hash.
func (s *AuthService) GenerateOneTimeToken() (plain string, hash string, err error) {
	return s.oneTimeTokenManager.Generate()
}

// HashOneTimeToken normalizes and hashes a one-time token.
func (s *AuthService) HashOneTimeToken(rawToken string) string {
	return s.oneTimeTokenManager.Hash(rawToken)
}
