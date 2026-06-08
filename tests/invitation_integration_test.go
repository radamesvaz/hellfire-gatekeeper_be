package tests

import (
	"context"
	"testing"

	jwt "github.com/golang-jwt/jwt/v5"
	authActionTokensRepo "github.com/radamesvaz/bakery-app/internal/repository/auth_action_tokens"
	"github.com/radamesvaz/bakery-app/internal/repository/user"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	authActionTokensService "github.com/radamesvaz/bakery-app/internal/services/auth_action_tokens"
	"github.com/radamesvaz/bakery-app/internal/services/email"
	invitationService "github.com/radamesvaz/bakery-app/internal/services/invitations"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	uModel "github.com/radamesvaz/bakery-app/model/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInvitationService_AcceptInvitation_AssignsAdminRole(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()
	runMigrations(t, dsn)

	ctx := context.Background()
	const tenantID = uint64(1)
	const inviterUserID = uint64(1)
	inviteEmail := "invited-admin@example.com"

	authSvc := authService.New("testingsecret", 60)
	userRepo := user.UserRepository{DB: db}
	tokenSvc := &authActionTokensService.ActionTokenService{
		DB:          db,
		Repo:        &authActionTokensRepo.SQLRepository{DB: db},
		AuthService: authSvc,
	}
	svc := &invitationService.InvitationService{
		Users:        &userRepo,
		AuthService:  authSvc,
		TokenService: tokenSvc,
		EmailSender:  email.NoopSender{},
		AppBaseURL:   "http://localhost:5173",
	}

	tokenResp, err := tokenSvc.CreateToken(ctx, authModel.CreateActionTokenRequest{
		TenantID:        tenantID,
		Email:           inviteEmail,
		Purpose:         authModel.ActionTokenPurposeInvite,
		CreatedByUserID: ptrUint64(inviterUserID),
	})
	require.NoError(t, err)
	require.NotEmpty(t, tokenResp.Token)

	resp, err := svc.AcceptInvitation(ctx, tenantID, authModel.AcceptTenantInvitationRequest{
		Token:    tokenResp.Token,
		Name:     "Invited Admin",
		Phone:    "555-0000",
		Password: "MyPassword123!",
	})
	require.NoError(t, err)
	require.NotZero(t, resp.UserID)
	assert.Equal(t, inviteEmail, resp.Email)
	assert.NotEmpty(t, resp.Token)

	created, err := userRepo.GetUserByEmail(tenantID, inviteEmail)
	require.NoError(t, err)
	assert.Equal(t, resp.UserID, created.ID)
	assert.Equal(t, tenantID, created.TenantID)
	assert.Equal(t, uModel.UserRoleAdmin, created.IDRole)

	parsed, err := authSvc.ValidateToken(resp.Token)
	require.NoError(t, err)
	claims, ok := parsed.Claims.(jwt.MapClaims)
	require.True(t, ok)
	assert.Equal(t, float64(uModel.UserRoleAdmin), claims["role_id"])
	assert.Equal(t, float64(tenantID), claims["tenant_id"])
}

func ptrUint64(v uint64) *uint64 {
	return &v
}
