package email

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
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

	return s.send(ctx, reqBody)
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

	return s.send(ctx, reqBody)
}

func (s *BrevoSender) SendTenantSignupCode(ctx context.Context, payload TenantSignupCodePayload) error {
	expiresAt := strings.TrimSpace(payload.ExpiresAt)
	expiryHTML := ""
	expiryText := ""
	if expiresAt != "" {
		expiryHTML = fmt.Sprintf("<p>This link expires at %s (UTC).</p>", expiresAt)
		expiryText = "This link expires at " + expiresAt + " (UTC).\n"
	}

	reqBody := map[string]interface{}{
		"sender": map[string]string{
			"email": s.FromEmail,
			"name":  s.FromName,
		},
		"to": []map[string]string{
			{"email": payload.ToEmail},
		},
		"subject": "Complete your bakery registration",
		"htmlContent": fmt.Sprintf(
			"<p>You have been invited to register a new bakery on Hellfire Gatekeeper.</p><p><a href=\"%s\">Complete registration</a></p>%s<p>If you did not expect this email, ignore it.</p>",
			payload.RegisterURL,
			expiryHTML,
		),
		"textContent": fmt.Sprintf(
			"You have been invited to register a new bakery on Hellfire Gatekeeper.\nComplete registration: %s\n%s",
			payload.RegisterURL,
			expiryText,
		),
	}

	return s.send(ctx, reqBody)
}

func (s *BrevoSender) send(ctx context.Context, reqBody map[string]interface{}) error {
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
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		msg := strings.TrimSpace(string(body))
		if msg == "" {
			return fmt.Errorf("brevo returned unexpected status: %d", resp.StatusCode)
		}
		return fmt.Errorf("brevo returned unexpected status: %d: %s", resp.StatusCode, msg)
	}

	return nil
}
