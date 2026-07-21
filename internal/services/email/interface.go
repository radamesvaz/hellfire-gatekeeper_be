package email

import "context"

type PasswordResetPayload struct {
	ToEmail  string
	ResetURL string
}

type TenantInvitationPayload struct {
	ToEmail   string
	InviteURL string
}

type TenantSignupCodePayload struct {
	ToEmail     string
	RegisterURL string
	ExpiresAt   string
}

type Sender interface {
	SendPasswordReset(ctx context.Context, payload PasswordResetPayload) error
	SendTenantInvitation(ctx context.Context, payload TenantInvitationPayload) error
	SendTenantSignupCode(ctx context.Context, payload TenantSignupCodePayload) error
}
