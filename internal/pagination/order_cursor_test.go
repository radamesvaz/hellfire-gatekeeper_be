package pagination

import (
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeOrderCursor_RoundTrip(t *testing.T) {
	ts := time.Date(2025, 4, 14, 10, 0, 0, 123456789, time.UTC)
	s, err := EncodeOrderCursor(ts, 2)
	require.NoError(t, err)

	k, err := DecodeOrderCursor(s)
	require.NoError(t, err)
	assert.Equal(t, uint64(2), k.ID)
	assert.True(t, k.CreatedOn.Equal(ts.UTC()))
}

func TestDecodeOrderCursor_WrongVersion(t *testing.T) {
	s, err := EncodeIDCursor(5)
	require.NoError(t, err)
	_, err = DecodeOrderCursor(s)
	assert.Error(t, err)
}

func TestDecodeOrderCursor_RejectsLegacyV2(t *testing.T) {
	// Hand-built v2 payload (ASC era); must not decode under newest-first (v3).
	raw := []byte(`{"v":2,"id":2,"ts":"2025-04-14T10:00:00.123456789Z"}`)
	s := base64.RawURLEncoding.EncodeToString(raw)
	_, err := DecodeOrderCursor(s)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported order cursor version")
}
