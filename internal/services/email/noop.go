package email

import "context"

type NoopSender struct{}

func (NoopSender) SendPasswordReset(_ context.Context, _ PasswordResetPayload) error {
	return nil
}
