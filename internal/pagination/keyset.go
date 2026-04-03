package pagination

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
)

const idCursorVersion = 1

// idCursorPayload is the JSON shape inside an opaque list cursor (products, orders).
type idCursorPayload struct {
	V  int    `json:"v"`
	ID uint64 `json:"id"`
}

// EncodeIDCursor builds an opaque cursor from the last visible row id (keyset on id DESC).
func EncodeIDCursor(id uint64) (string, error) {
	b, err := json.Marshal(idCursorPayload{V: idCursorVersion, ID: id})
	if err != nil {
		return "", fmt.Errorf("pagination: encode cursor: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// DecodeIDCursor parses a cursor produced by EncodeIDCursor.
func DecodeIDCursor(s string) (uint64, error) {
	if s == "" {
		return 0, errors.New("pagination: empty cursor")
	}
	raw, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return 0, fmt.Errorf("pagination: invalid cursor encoding: %w", err)
	}
	var p idCursorPayload
	if err := json.Unmarshal(raw, &p); err != nil {
		return 0, fmt.Errorf("pagination: invalid cursor payload: %w", err)
	}
	if p.V != idCursorVersion {
		return 0, fmt.Errorf("pagination: unsupported cursor version %d", p.V)
	}
	if p.ID == 0 {
		return 0, errors.New("pagination: invalid cursor id")
	}
	return p.ID, nil
}
