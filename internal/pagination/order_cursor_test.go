package pagination

import (
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
