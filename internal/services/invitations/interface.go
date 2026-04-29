package invitations

import (
	"context"

	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

type Service interface {
	CreateInvitation(ctx context.Context, tenantID uint64, tenantSlug string, roleID uint64, createdByUserID uint64, req authModel.CreateTenantInvitationRequest) (authModel.CreateTenantInvitationResponse, error)
	AcceptInvitation(ctx context.Context, tenantID uint64, req authModel.AcceptTenantInvitationRequest) (authModel.AcceptTenantInvitationResponse, error)
}
