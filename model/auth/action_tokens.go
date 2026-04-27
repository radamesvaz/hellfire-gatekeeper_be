package model

import (
	"encoding/json"
	"time"
)

type ActionTokenPurpose string

const (
	ActionTokenPurposeInvite        ActionTokenPurpose = "invite"
	ActionTokenPurposePasswordReset ActionTokenPurpose = "password_reset"
)

type CreateActionTokenRequest struct {
	TenantID         uint64
	Email            string
	Purpose          ActionTokenPurpose
	ExpiresInMinutes int
	SubjectUserID    *uint64
	CreatedByUserID  *uint64
	MetadataJSON     json.RawMessage
}

type CreateActionTokenResponse struct {
	ID        uint64    `json:"id"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
}

type ActionTokenRecord struct {
	ID            uint64
	TenantID      uint64
	Email         string
	Purpose       ActionTokenPurpose
	SubjectUserID *uint64
	ExpiresAt     time.Time
	UsedAt        *time.Time
	RevokedAt     *time.Time
}
