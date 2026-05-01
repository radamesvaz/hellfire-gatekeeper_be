package invitations

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
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

type InvitationService struct {
	Users        *user.UserRepository
	AuthService  authService.Service
	TokenService authTokensService.Service
	EmailSender  email.Sender
	AppBaseURL   string
}

func (s *InvitationService) CreateInvitation(
	ctx context.Context,
	tenantID uint64,
	tenantSlug string,
	roleID uint64,
	createdByUserID uint64,
	req authModel.CreateTenantInvitationRequest,
) (authModel.CreateTenantInvitationResponse, error) {
	if roleID != uint64(uModel.UserRoleAdmin) {
		return authModel.CreateTenantInvitationResponse{}, appErrors.ErrForbidden
	}

	emailAddr := strings.TrimSpace(strings.ToLower(req.Email))
	exists, err := s.Users.EmailExists(tenantID, emailAddr)
	if err != nil {
		return authModel.CreateTenantInvitationResponse{}, err
	}
	if exists {
		return authModel.CreateTenantInvitationResponse{}, appErrors.ErrEmailAlreadyExists
	}

	tokenResp, err := s.TokenService.CreateToken(ctx, authModel.CreateActionTokenRequest{
		TenantID:        tenantID,
		Email:           emailAddr,
		Purpose:         authModel.ActionTokenPurposeInvite,
		CreatedByUserID: &createdByUserID,
	})
	if err != nil {
		return authModel.CreateTenantInvitationResponse{}, err
	}

	inviteURL, err := buildInviteURL(s.AppBaseURL, tenantSlug, tokenResp.Token)
	if err != nil {
		return authModel.CreateTenantInvitationResponse{}, err
	}

	if s.EmailSender == nil {
		logger.Error().
			Uint64("tenant_id", tenantID).
			Msg("Brevo sender is not running/configured for invitations")
		return authModel.CreateTenantInvitationResponse{}, appErrors.NewInternalServerError(errors.New("email sender not configured"))
	}
	if sendErr := s.EmailSender.SendTenantInvitation(ctx, email.TenantInvitationPayload{
		ToEmail:   emailAddr,
		InviteURL: inviteURL,
	}); sendErr != nil {
		return authModel.CreateTenantInvitationResponse{}, sendErr
	}

	return authModel.CreateTenantInvitationResponse{
		ID:        tokenResp.ID,
		Email:     emailAddr,
		ExpiresAt: tokenResp.ExpiresAt,
		Message:   "Invitation sent successfully",
	}, nil
}

func (s *InvitationService) AcceptInvitation(ctx context.Context, tenantID uint64, req authModel.AcceptTenantInvitationRequest) (authModel.AcceptTenantInvitationResponse, error) {
	rec, err := s.TokenService.ConsumeToken(ctx, tenantID, authModel.ActionTokenPurposeInvite, req.Token)
	if err != nil {
		return authModel.AcceptTenantInvitationResponse{}, err
	}

	exists, err := s.Users.EmailExists(tenantID, rec.Email)
	if err != nil {
		return authModel.AcceptTenantInvitationResponse{}, err
	}
	if exists {
		return authModel.AcceptTenantInvitationResponse{}, appErrors.ErrEmailAlreadyExists
	}

	name := strings.TrimSpace(req.Name)
	phone := strings.TrimSpace(req.Phone)
	if name == "" {
		return authModel.AcceptTenantInvitationResponse{}, appErrors.NewBadRequest(errors.New("name is required"))
	}

	passwordHash, err := s.AuthService.HashPassword(req.Password)
	if err != nil {
		return authModel.AcceptTenantInvitationResponse{}, err
	}

	createdID, err := s.Users.CreateUser(ctx, uModel.CreateUserRequest{
		TenantID: tenantID,
		IDRole:   uModel.UserRoleClient,
		Name:     name,
		Email:    rec.Email,
		Phone:    phone,
		Password: passwordHash,
	})
	if err != nil {
		return authModel.AcceptTenantInvitationResponse{}, err
	}

	tenantIDPtr := tenantID
	jwtToken, err := s.AuthService.GenerateJWT(createdID, uModel.UserRoleClient, rec.Email, &tenantIDPtr)
	if err != nil {
		return authModel.AcceptTenantInvitationResponse{}, err
	}

	return authModel.AcceptTenantInvitationResponse{
		Message: "Invitation accepted successfully",
		Token:   jwtToken,
		Email:   rec.Email,
		UserID:  createdID,
	}, nil
}

func (s *InvitationService) RevokeInvitation(ctx context.Context, tenantID uint64, roleID uint64, invitationID uint64) error {
	if roleID != uint64(uModel.UserRoleAdmin) {
		return appErrors.ErrForbidden
	}
	return s.TokenService.RevokeTokenScoped(ctx, tenantID, authModel.ActionTokenPurposeInvite, invitationID)
}

func (s *InvitationService) ResendInvitation(
	ctx context.Context,
	tenantID uint64,
	tenantSlug string,
	roleID uint64,
	createdByUserID uint64,
	invitationID uint64,
) (authModel.CreateTenantInvitationResponse, error) {
	if roleID != uint64(uModel.UserRoleAdmin) {
		return authModel.CreateTenantInvitationResponse{}, appErrors.ErrForbidden
	}

	current, err := s.TokenService.GetTokenByIDScoped(ctx, tenantID, authModel.ActionTokenPurposeInvite, invitationID)
	if err != nil {
		return authModel.CreateTenantInvitationResponse{}, err
	}
	if current.UsedAt != nil {
		return authModel.CreateTenantInvitationResponse{}, appErrors.ErrTokenAlreadyConsumed
	}
	if current.RevokedAt != nil {
		return authModel.CreateTenantInvitationResponse{}, appErrors.ErrTokenRevoked
	}

	resp, err := s.CreateInvitation(ctx, tenantID, tenantSlug, roleID, createdByUserID, authModel.CreateTenantInvitationRequest{
		Email: current.Email,
	})
	if err != nil {
		return authModel.CreateTenantInvitationResponse{}, err
	}
	if err := s.TokenService.RevokeTokenScoped(ctx, tenantID, authModel.ActionTokenPurposeInvite, invitationID); err != nil {
		return authModel.CreateTenantInvitationResponse{}, err
	}
	return resp, nil
}

func buildInviteURL(baseURL string, tenantSlug string, token string) (string, error) {
	base := strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if base == "" {
		return "", appErrors.NewInternalServerError(errors.New("APP_BASE_URL is required for invitation links"))
	}
	return fmt.Sprintf("%s/t/%s/invite/accept?token=%s", base, url.PathEscape(strings.TrimSpace(tenantSlug)), url.QueryEscape(strings.TrimSpace(token))), nil
}
