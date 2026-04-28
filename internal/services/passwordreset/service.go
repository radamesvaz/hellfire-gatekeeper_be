package passwordreset

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"

	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/logger"
	"github.com/radamesvaz/bakery-app/internal/repository/user"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	authTokensService "github.com/radamesvaz/bakery-app/internal/services/auth_tokens"
	"github.com/radamesvaz/bakery-app/internal/services/email"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

type PasswordResetService struct {
	Users        *user.UserRepository
	AuthService  authService.Service
	TokenService authTokensService.Service
	EmailSender  email.Sender
	AppBaseURL   string
}

func (s *PasswordResetService) ForgotPassword(ctx context.Context, tenantID uint64, tenantSlug string, emailAddr string) error {
	emailAddr = strings.TrimSpace(emailAddr)
	if emailAddr == "" {
		return appErrors.NewBadRequest(appErrors.ErrCouldNotGetTheUser)
	}

	u, err := s.Users.GetUserByEmail(tenantID, emailAddr)
	if err != nil {
		// Neutral response policy: unknown emails do not produce an outward error.
		if errors.Is(err, appErrors.ErrUserNotFound) {
			return nil
		}
		return err
	}

	tokenResp, err := s.TokenService.CreateToken(ctx, authModel.CreateActionTokenRequest{
		TenantID:      tenantID,
		Email:         emailAddr,
		Purpose:       authModel.ActionTokenPurposePasswordReset,
		SubjectUserID: &u.ID,
	})
	if err != nil {
		return err
	}

	resetURL, err := buildResetURL(s.AppBaseURL, tenantSlug, tokenResp.Token)
	if err != nil {
		return err
	}
	if s.EmailSender != nil {
		sendErr := s.EmailSender.SendPasswordReset(ctx, email.PasswordResetPayload{
			ToEmail:  emailAddr,
			ResetURL: resetURL,
		})
		if sendErr != nil {
			// Keep forgot response neutral. Do not leak if send failed.
			logger.Warn().Err(sendErr).Uint64("tenant_id", tenantID).Msg("password reset email delivery failed")
		}
	}
	return nil
}

func (s *PasswordResetService) ResetPassword(ctx context.Context, tenantID uint64, token string, newPassword string) error {
	rec, err := s.TokenService.ConsumeToken(ctx, tenantID, authModel.ActionTokenPurposePasswordReset, token)
	if err != nil {
		return err
	}
	if rec.SubjectUserID == nil || *rec.SubjectUserID == 0 {
		return appErrors.ErrInvalidToken
	}

	passwordHash, err := s.AuthService.HashPassword(newPassword)
	if err != nil {
		return err
	}
	return s.Users.UpdatePasswordHash(ctx, tenantID, *rec.SubjectUserID, passwordHash)
}

func buildResetURL(baseURL string, tenantSlug string, token string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", appErrors.NewInternalServerError(errors.New("APP_BASE_URL is required for password reset links"))
	}
	slug := strings.TrimSpace(tenantSlug)
	return fmt.Sprintf("%s/t/%s/reset-password?token=%s", base, url.PathEscape(slug), url.QueryEscape(strings.TrimSpace(token))), nil
}
