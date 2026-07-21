package tenantsignup

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/radamesvaz/bakery-app/internal/logger"
	repo "github.com/radamesvaz/bakery-app/internal/repository/tenantsignup"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	"github.com/radamesvaz/bakery-app/internal/services/email"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

const defaultSignupCodeTTLMinutes = 120

var (
	ErrForbidden                = errors.New("forbidden")
	ErrInvalidEmail             = errors.New("invalid email")
	ErrEmailNotConfigured       = errors.New("email sender not configured")
	ErrEmailDeliveryFailed      = errors.New("email delivery failed")
	ErrAppBaseURLRequired       = errors.New("APP_BASE_URL is required for signup links")
	ErrInvalidOrUnavailableCode = repo.ErrInvalidOrUnavailableCode
	ErrTenantSlugExists         = repo.ErrTenantSlugExists
	ErrAdminEmailExists         = repo.ErrAdminEmailExists
)

type TenantSignupService struct {
	Repo        *repo.Repository
	AuthService authService.Service
	EmailSender email.Sender
	AppBaseURL  string
}

func (s *TenantSignupService) CreateSignupCode(
	ctx context.Context,
	roleID uint64,
	createdByUserID uint64,
	req authModel.CreateSignupCodeRequest,
) (authModel.CreateSignupCodeResponse, error) {
	if roleID != uint64(uModel.UserRoleSuperAdmin) {
		return authModel.CreateSignupCodeResponse{}, ErrForbidden
	}

	recipientEmail := strings.TrimSpace(strings.ToLower(req.Email))
	if recipientEmail == "" {
		return authModel.CreateSignupCodeResponse{}, ErrInvalidEmail
	}

	if s.EmailSender == nil {
		logger.Error().Msg("email sender not configured for tenant signup codes")
		return authModel.CreateSignupCodeResponse{}, ErrEmailNotConfigured
	}

	ttlMinutes := req.ExpiresInMinutes
	if ttlMinutes <= 0 {
		ttlMinutes = defaultSignupCodeTTLMinutes
	}
	expiresAt := time.Now().UTC().Add(time.Duration(ttlMinutes) * time.Minute)

	code, codeHash, err := s.AuthService.GenerateOneTimeToken()
	if err != nil {
		return authModel.CreateSignupCodeResponse{}, err
	}

	registerURL, err := buildTenantRegisterURL(s.AppBaseURL, code)
	if err != nil {
		return authModel.CreateSignupCodeResponse{}, err
	}

	// Send email before persisting so failed deliveries leave no orphan/revoked rows.
	if sendErr := s.EmailSender.SendTenantSignupCode(ctx, email.TenantSignupCodePayload{
		ToEmail:     recipientEmail,
		RegisterURL: registerURL,
		ExpiresAt:   expiresAt.Format(time.RFC3339),
	}); sendErr != nil {
		logger.Error().Err(sendErr).Str("recipient_email", recipientEmail).Msg("failed to send tenant signup code email")
		return authModel.CreateSignupCodeResponse{}, fmt.Errorf("%w: %v", ErrEmailDeliveryFailed, sendErr)
	}

	id, err := s.Repo.CreateSignupCode(ctx, codeHash, expiresAt, createdByUserID, recipientEmail, req.Notes)
	if err != nil {
		logger.Error().Err(err).Str("recipient_email", recipientEmail).Msg("signup email sent but failed to persist code")
		return authModel.CreateSignupCodeResponse{}, err
	}

	return authModel.CreateSignupCodeResponse{
		ID:        id,
		Code:      code,
		ExpiresAt: expiresAt,
		Email:     recipientEmail,
		Message:   "Signup code sent successfully",
	}, nil
}

func (s *TenantSignupService) RegisterTenantWithCode(ctx context.Context, req authModel.PublicTenantRegisterRequest) (authModel.PublicTenantRegisterResponse, error) {
	adminEmail := strings.TrimSpace(req.Email)
	hashedPassword, err := s.AuthService.HashPassword(req.Password)
	if err != nil {
		return authModel.PublicTenantRegisterResponse{}, err
	}

	result, err := s.Repo.RegisterTenantWithCode(ctx, repo.RegisterTenantWithCodeInput{
		CodeHash:     s.AuthService.HashOneTimeToken(req.OneTimeCode),
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

func buildTenantRegisterURL(baseURL string, code string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", ErrAppBaseURLRequired
	}
	return fmt.Sprintf("%s/tenant-register?code=%s", base, url.QueryEscape(strings.TrimSpace(code))), nil
}
