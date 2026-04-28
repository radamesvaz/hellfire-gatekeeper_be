package passwordreset

import "context"

type Service interface {
	ForgotPassword(ctx context.Context, tenantID uint64, tenantSlug string, email string) error
	ResetPassword(ctx context.Context, tenantID uint64, token string, newPassword string) error
}
