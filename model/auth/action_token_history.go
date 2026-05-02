package model

// ActionTokenHistoryAction is stored in auth_action_tokens_history.action (CHECK constraint).
type ActionTokenHistoryAction string

const (
	ActionTokenInviteCreated           ActionTokenHistoryAction = "invite_created"
	ActionTokenInviteRevoked           ActionTokenHistoryAction = "invite_revoked"
	ActionTokenInviteAccepted          ActionTokenHistoryAction = "invite_accepted"
	ActionTokenPasswordResetIssued    ActionTokenHistoryAction = "password_reset_issued"
	ActionTokenPasswordResetCompleted ActionTokenHistoryAction = "password_reset_completed"
)
