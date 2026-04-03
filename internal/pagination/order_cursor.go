package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

const orderCursorVersion = 2

type orderCursorPayload struct {
	V  int    `json:"v"`
	ID uint64 `json:"id"`
	TS string `json:"ts"` // RFC3339Nano, UTC
}

// OrderKeyset marks a position in the orders list ordered by created_on ASC, id_order ASC.
type OrderKeyset struct {
	CreatedOn time.Time
	ID        uint64
}

// EncodeOrderCursor builds the opaque cursor for the last visible order on a page (creation order).
func EncodeOrderCursor(createdOn time.Time, id uint64) (string, error) {
	if id == 0 {
		return "", errors.New("pagination: invalid order cursor id")
	}
	t := createdOn.UTC()
	ts := t.Format(time.RFC3339Nano)
	b, err := json.Marshal(orderCursorPayload{V: orderCursorVersion, ID: id, TS: ts})
	if err != nil {
		return "", fmt.Errorf("pagination: encode order cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// DecodeOrderCursor parses a cursor from EncodeOrderCursor (version 2 only).
func DecodeOrderCursor(s string) (OrderKeyset, error) {
	var zero OrderKeyset
	if s == "" {
		return zero, errors.New("pagination: empty cursor")
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return zero, fmt.Errorf("pagination: invalid order cursor encoding: %w", err)
	}
	var p orderCursorPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return zero, fmt.Errorf("pagination: invalid order cursor payload: %w", err)
	}
	if p.V != orderCursorVersion {
		return zero, fmt.Errorf("pagination: unsupported order cursor version %d", p.V)
	}
	if p.ID == 0 {
		return zero, errors.New("pagination: invalid order cursor id")
	}
	createdOn, err := time.Parse(time.RFC3339Nano, p.TS)
	if err != nil {
		createdOn, err = time.Parse(time.RFC3339, p.TS)
	}
	if err != nil {
		return zero, fmt.Errorf("pagination: invalid order cursor time: %w", err)
	}
	return OrderKeyset{CreatedOn: createdOn.UTC(), ID: p.ID}, nil
}
