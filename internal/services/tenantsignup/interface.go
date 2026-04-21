package tenantsignup

import (
	"context"

	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

type Service interface {
	CreateSignupCode(ctx context.Context, roleID uint64, createdByUserID uint64, req authModel.CreateSignupCodeRequest) (authModel.CreateSignupCodeResponse, error)
	RegisterTenantWithCode(ctx context.Context, req authModel.PublicTenantRegisterRequest) (authModel.PublicTenantRegisterResponse, error)
}
