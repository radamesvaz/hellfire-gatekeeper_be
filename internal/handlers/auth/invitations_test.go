package auth

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	jwt "github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/mux"
	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/middleware"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type mockInvitationService struct {
	revokeFn func(ctx context.Context, tenantID uint64, roleID uint64, invitationID uint64) error
}

func (m *mockInvitationService) CreateInvitation(_ context.Context, _ uint64, _ string, _ uint64, _ uint64, _ authModel.CreateTenantInvitationRequest) (authModel.CreateTenantInvitationResponse, error) {
	return authModel.CreateTenantInvitationResponse{}, nil
}

func (m *mockInvitationService) AcceptInvitation(_ context.Context, _ uint64, _ authModel.AcceptTenantInvitationRequest) (authModel.AcceptTenantInvitationResponse, error) {
	return authModel.AcceptTenantInvitationResponse{}, nil
}

func (m *mockInvitationService) RevokeInvitation(ctx context.Context, tenantID uint64, roleID uint64, invitationID uint64) error {
	if m.revokeFn == nil {
		return nil
	}
	return m.revokeFn(ctx, tenantID, roleID, invitationID)
}

func TestInvitationHandler_RevokeInvitation_Success(t *testing.T) {
	var gotTenantID, gotRoleID, gotInvitationID uint64
	handler := &InvitationHandler{
		Service: &mockInvitationService{
			revokeFn: func(_ context.Context, tenantID uint64, roleID uint64, invitationID uint64) error {
				gotTenantID = tenantID
				gotRoleID = roleID
				gotInvitationID = invitationID
				return nil
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/invitations/17/revoke", nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, uint64(9))
	ctx = context.WithValue(ctx, middleware.UserClaimsKey, jwt.MapClaims{
		"user_id": float64(33),
		"role_id": float64(1),
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/auth/invitations/{id}/revoke", handler.RevokeInvitation).Methods(http.MethodPost)
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusOK, rr.Code, rr.Body.String())
	assert.Equal(t, uint64(9), gotTenantID)
	assert.Equal(t, uint64(1), gotRoleID)
	assert.Equal(t, uint64(17), gotInvitationID)
	assert.Contains(t, rr.Body.String(), "Invitation revoked successfully")
}

func TestInvitationHandler_RevokeInvitation_Forbidden(t *testing.T) {
	handler := &InvitationHandler{
		Service: &mockInvitationService{
			revokeFn: func(_ context.Context, _ uint64, _ uint64, _ uint64) error {
				return appErrors.ErrForbidden
			},
		},
	}

	req := httptest.NewRequest(http.MethodPost, "/auth/invitations/22/revoke", nil)
	ctx := context.WithValue(req.Context(), middleware.TenantIDKey, uint64(3))
	ctx = context.WithValue(ctx, middleware.UserClaimsKey, jwt.MapClaims{
		"user_id": float64(88),
		"role_id": float64(2),
	})
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	router := mux.NewRouter()
	router.HandleFunc("/auth/invitations/{id}/revoke", handler.RevokeInvitation).Methods(http.MethodPost)
	router.ServeHTTP(rr, req)

	require.Equal(t, http.StatusForbidden, rr.Code, rr.Body.String())
	assert.Contains(t, rr.Body.String(), "Forbidden")
}
