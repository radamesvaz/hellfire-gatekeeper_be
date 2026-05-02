package model

import "time"

type CreateTenantInvitationRequest struct {
	Email string `json:"email"`
}

type CreateTenantInvitationResponse struct {
	ID        uint64    `json:"id"`
	Email     string    `json:"email"`
	ExpiresAt time.Time `json:"expires_at"`
	Message   string    `json:"message"`
}

type AcceptTenantInvitationRequest struct {
	Token    string `json:"token"`
	Name     string `json:"name"`
	Phone    string `json:"phone"`
	Password string `json:"password"`
}

type AcceptTenantInvitationResponse struct {
	Message string `json:"message"`
	Token   string `json:"token"`
	Email   string `json:"email"`
	UserID  uint64 `json:"user_id"`
}
