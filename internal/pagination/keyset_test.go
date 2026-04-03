package pagination

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncodeDecodeIDCursor_RoundTrip(t *testing.T) {
	s, err := EncodeIDCursor(42)
	require.NoError(t, err)
	assert.NotEmpty(t, s)

	id, err := DecodeIDCursor(s)
	require.NoError(t, err)
	assert.Equal(t, uint64(42), id)
}

func TestDecodeIDCursor_Errors(t *testing.T) {
	_, err := DecodeIDCursor("")
	assert.Error(t, err)

	_, err = DecodeIDCursor("not-base64!!!")
	assert.Error(t, err)

	_, err = DecodeIDCursor("e30") // {}
	assert.Error(t, err)
}
