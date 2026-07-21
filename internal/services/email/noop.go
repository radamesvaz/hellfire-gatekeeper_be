package email

import "context"

type NoopSender struct{}

func (NoopSender) SendPasswordReset(_ context.Context, _ PasswordResetPayload) error {
	return nil
}

func (NoopSender) SendTenantInvitation(_ context.Context, _ TenantInvitationPayload) error {
	return nil
}

func (NoopSender) SendTenantSignupCode(_ context.Context, _ TenantSignupCodePayload) error {
	return nil
}
