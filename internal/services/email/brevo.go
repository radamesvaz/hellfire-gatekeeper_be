package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type BrevoSender struct {
	APIKey    string
	FromEmail string
	FromName  string
	Client    *http.Client
}

func NewBrevoSender(apiKey, fromEmail, fromName string) *BrevoSender {
	return &BrevoSender{
		APIKey:    apiKey,
		FromEmail: fromEmail,
		FromName:  fromName,
		Client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *BrevoSender) SendPasswordReset(ctx context.Context, payload PasswordResetPayload) error {
	reqBody := map[string]interface{}{
		"sender": map[string]string{
			"email": s.FromEmail,
			"name":  s.FromName,
		},
		"to": []map[string]string{
			{"email": payload.ToEmail},
		},
		"subject": "Password reset instructions",
		"htmlContent": fmt.Sprintf(
			"<p>You requested a password reset.</p><p><a href=\"%s\">Reset your password</a></p><p>If you did not request this, ignore this email.</p>",
			payload.ResetURL,
		),
		"textContent": "You requested a password reset. Open this link: " + payload.ResetURL,
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal brevo request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("create brevo request: %w", err)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("api-key", s.APIKey)

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("brevo http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("brevo returned unexpected status: %d", resp.StatusCode)
	}

	return nil
}

func (s *BrevoSender) SendTenantInvitation(ctx context.Context, payload TenantInvitationPayload) error {
	reqBody := map[string]interface{}{
		"sender": map[string]string{
			"email": s.FromEmail,
			"name":  s.FromName,
		},
		"to": []map[string]string{
			{"email": payload.ToEmail},
		},
		"subject": "You are invited to join a tenant",
		"htmlContent": fmt.Sprintf(
			"<p>You have been invited to join a tenant.</p><p><a href=\"%s\">Accept invitation</a></p>",
			payload.InviteURL,
		),
		"textContent": "You have been invited to join a tenant. Open this link: " + payload.InviteURL,
	}

	raw, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("marshal brevo request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.brevo.com/v3/smtp/email", bytes.NewReader(raw))
	if err != nil {
		return fmt.Errorf("create brevo request: %w", err)
	}
	req.Header.Set("accept", "application/json")
	req.Header.Set("content-type", "application/json")
	req.Header.Set("api-key", s.APIKey)

	resp, err := s.Client.Do(req)
	if err != nil {
		return fmt.Errorf("brevo http request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("brevo returned unexpected status: %d", resp.StatusCode)
	}

	return nil
}
