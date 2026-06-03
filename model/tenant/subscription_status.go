package tenant

import (
	"database/sql/driver"
	"fmt"
	"strings"
)

// SubscriptionStatus mirrors PostgreSQL enum subscription_status on tenants.
type SubscriptionStatus string

const (
	SubscriptionStatusActive   SubscriptionStatus = "active"
	SubscriptionStatusPending  SubscriptionStatus = "pending"
	SubscriptionStatusCanceled SubscriptionStatus = "canceled"
)

func (s SubscriptionStatus) String() string {
	return string(s)
}

// Valid reports whether s is one of the allowed enum values.
func (s SubscriptionStatus) Valid() bool {
	switch s {
	case SubscriptionStatusActive, SubscriptionStatusPending, SubscriptionStatusCanceled:
		return true
	default:
		return false
	}
}

// ParseSubscriptionStatus normalizes input and validates against allowed values.
func ParseSubscriptionStatus(raw string) (SubscriptionStatus, bool) {
	st := SubscriptionStatus(strings.TrimSpace(strings.ToLower(raw)))
	return st, st.Valid()
}

// OperableSubscriptionStatuses are tenants that may use public routes and /auth/* (MVP).
func OperableSubscriptionStatuses() []SubscriptionStatus {
	return []SubscriptionStatus{SubscriptionStatusActive, SubscriptionStatusPending}
}

// Scan implements sql.Scanner for reading the PG enum into Go.
func (s *SubscriptionStatus) Scan(value interface{}) error {
	if value == nil {
		*s = ""
		return nil
	}
	var raw string
	switch v := value.(type) {
	case string:
		raw = v
	case []byte:
		raw = string(v)
	default:
		return fmt.Errorf("unsupported subscription_status type %T", value)
	}
	parsed, ok := ParseSubscriptionStatus(raw)
	if !ok {
		return fmt.Errorf("invalid subscription_status: %q", raw)
	}
	*s = parsed
	return nil
}

// Value implements driver.Valuer for writes to the PG enum column.
func (s SubscriptionStatus) Value() (driver.Value, error) {
	if !s.Valid() {
		return nil, fmt.Errorf("invalid subscription status: %q", s)
	}
	return string(s), nil
}
