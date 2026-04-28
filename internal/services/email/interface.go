package email

import "context"

type PasswordResetPayload struct {
	ToEmail  string
	ResetURL string
}

type Sender interface {
	SendPasswordReset(ctx context.Context, payload PasswordResetPayload) error
}
