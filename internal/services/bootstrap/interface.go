package bootstrap

import (
	"context"

	authModel "github.com/radamesvaz/bakery-app/model/auth"
)

type Service interface {
	BootstrapTenant(ctx context.Context, req authModel.BootstrapTenantRequest) (authModel.BootstrapTenantResponse, error)
}
