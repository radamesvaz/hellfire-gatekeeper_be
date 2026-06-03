package tenant

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionStatus_Valid(t *testing.T) {
	assert.True(t, SubscriptionStatusActive.Valid())
	assert.True(t, SubscriptionStatusPending.Valid())
	assert.True(t, SubscriptionStatusCanceled.Valid())
	assert.False(t, SubscriptionStatus("cancelled").Valid())
}

func TestParseSubscriptionStatus(t *testing.T) {
	st, ok := ParseSubscriptionStatus("  PENDING ")
	require.True(t, ok)
	assert.Equal(t, SubscriptionStatusPending, st)

	_, ok = ParseSubscriptionStatus("invalid")
	assert.False(t, ok)
}

func TestSubscriptionStatus_ScanAndValue(t *testing.T) {
	var st SubscriptionStatus
	require.NoError(t, st.Scan("active"))
	assert.Equal(t, SubscriptionStatusActive, st)

	v, err := SubscriptionStatusPending.Value()
	require.NoError(t, err)
	assert.Equal(t, "pending", v)
}
