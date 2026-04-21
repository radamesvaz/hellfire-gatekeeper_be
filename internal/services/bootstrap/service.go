package bootstrap

import (
	"context"
	"errors"
	"strings"

	repo "github.com/radamesvaz/bakery-app/internal/repository/bootstrap"
	authService "github.com/radamesvaz/bakery-app/internal/services/auth"
	authModel "github.com/radamesvaz/bakery-app/model/auth"
	uModel "github.com/radamesvaz/bakery-app/model/users"
)

var (
	ErrTenantSlugExists = repo.ErrTenantSlugExists
	ErrAdminEmailExists = repo.ErrAdminEmailExists
)

type BootstrapService struct {
	Repo        *repo.Repository
	AuthService authService.Service
}

func (s *BootstrapService) BootstrapTenant(ctx context.Context, req authModel.BootstrapTenantRequest) (authModel.BootstrapTenantResponse, error) {
	adminEmail := strings.TrimSpace(req.Email)
	passwordHash, err := s.AuthService.HashPassword(req.Password)
	if err != nil {
		return authModel.BootstrapTenantResponse{}, err
	}

	result, err := s.Repo.BootstrapTenant(ctx, repo.BootstrapTenantInput{
		TenantName:   req.TenantName,
		TenantSlug:   req.TenantSlug,
		AdminName:    strings.TrimSpace(req.AdminName),
		AdminEmail:   adminEmail,
		AdminPhone:   strings.TrimSpace(req.Phone),
		PasswordHash: passwordHash,
	})
	if err != nil {
		return authModel.BootstrapTenantResponse{}, err
	}

	token, err := s.AuthService.GenerateJWT(result.AdminID, uModel.UserRoleAdmin, adminEmail, &result.TenantID)
	if err != nil {
		return authModel.BootstrapTenantResponse{}, err
	}

	return authModel.BootstrapTenantResponse{
		Message:    "Tenant bootstrap completed successfully",
		Token:      token,
		TenantID:   result.TenantID,
		TenantSlug: req.TenantSlug,
		TenantName: req.TenantName,
		AdminID:    result.AdminID,
		AdminEmail: adminEmail,
	}, nil
}

func IsKnownConflict(err error) bool {
	return errors.Is(err, ErrTenantSlugExists) || errors.Is(err, ErrAdminEmailExists)
}
