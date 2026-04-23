package tenantsignup

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	repo "github.com/radamesvaz/bakery-app/internal/repository/tenantsignup"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

const defaultSignupCodeTTLMinutes = 120

var (
	ErrForbidden                = errors.New("forbidden")
	ErrInvalidOrUnavailableCode = repo.ErrInvalidOrUnavailableCode
	ErrTenantSlugExists         = repo.ErrTenantSlugExists
	ErrAdminEmailExists         = repo.ErrAdminEmailExists
)

type TenantSignupService struct {
	Repo        *repo.Repository
	AuthService authService.Service
}

func (s *TenantSignupService) CreateSignupCode(
	ctx context.Context,
	roleID uint64,
	createdByUserID uint64,
	req authModel.CreateSignupCodeRequest,
) (authModel.CreateSignupCodeResponse, error) {
	if roleID != uint64(uModel.UserRoleAdmin) {
		return authModel.CreateSignupCodeResponse{}, ErrForbidden
	}

	ttlMinutes := req.ExpiresInMinutes
	if ttlMinutes <= 0 {
		ttlMinutes = defaultSignupCodeTTLMinutes
	}
	expiresAt := time.Now().UTC().Add(time.Duration(ttlMinutes) * time.Minute)

	code, err := generateOneTimeCode()
	if err != nil {
		return authModel.CreateSignupCodeResponse{}, err
	}

	id, err := s.Repo.CreateSignupCode(ctx, hashOneTimeCode(code), expiresAt, createdByUserID, req.Notes)
	if err != nil {
		return authModel.CreateSignupCodeResponse{}, err
	}

	return authModel.CreateSignupCodeResponse{
		ID:        id,
		Code:      code,
		ExpiresAt: expiresAt,
	}, nil
}

func (s *TenantSignupService) RegisterTenantWithCode(ctx context.Context, req authModel.PublicTenantRegisterRequest) (authModel.PublicTenantRegisterResponse, error) {
	adminEmail := strings.TrimSpace(req.Email)
	hashedPassword, err := s.AuthService.HashPassword(req.Password)
	if err != nil {
		return authModel.PublicTenantRegisterResponse{}, err
	}

	result, err := s.Repo.RegisterTenantWithCode(ctx, repo.RegisterTenantWithCodeInput{
		CodeHash:     hashOneTimeCode(req.OneTimeCode),
		TenantName:   req.TenantName,
		TenantSlug:   req.TenantSlug,
		AdminName:    strings.TrimSpace(req.AdminName),
		AdminEmail:   adminEmail,
		AdminPhone:   strings.TrimSpace(req.Phone),
		PasswordHash: hashedPassword,
	})
	if err != nil {
		return authModel.PublicTenantRegisterResponse{}, err
	}

	token, err := s.AuthService.GenerateJWT(result.AdminID, uModel.UserRoleAdmin, adminEmail, &result.TenantID)
	if err != nil {
		return authModel.PublicTenantRegisterResponse{}, err
	}

	return authModel.PublicTenantRegisterResponse{
		Message:    "Tenant registered successfully",
		Token:      token,
		TenantID:   result.TenantID,
		TenantSlug: req.TenantSlug,
		TenantName: req.TenantName,
		AdminID:    result.AdminID,
		AdminEmail: adminEmail,
	}, nil
}

func generateOneTimeCode() (string, error) {
	raw := make([]byte, 8)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	encoded := strings.ToUpper(hex.EncodeToString(raw))
	return encoded[:8] + "-" + encoded[8:], nil
}

func hashOneTimeCode(code string) string {
	sum := sha256.Sum256([]byte(strings.TrimSpace(strings.ToUpper(code))))
	return hex.EncodeToString(sum[:])
}
