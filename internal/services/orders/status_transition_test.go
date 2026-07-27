package orders

import (
	"testing"

	appErrors "github.com/radamesvaz/bakery-app/internal/errors"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	"github.com/stretchr/testify/assert"
)

func TestValidateStatusTransition_LinearForward(t *testing.T) {
	s := &StatusUpdaterWithStock{}

	assert.NoError(t, s.validateStatusTransition(oModel.StatusPending, oModel.StatusPreparing))
	assert.NoError(t, s.validateStatusTransition(oModel.StatusPreparing, oModel.StatusReady))
	assert.NoError(t, s.validateStatusTransition(oModel.StatusReady, oModel.StatusDelivered))
}

func TestValidateStatusTransition_RejectsSkipsAndBackwards(t *testing.T) {
	s := &StatusUpdaterWithStock{}

	assert.ErrorIs(t, s.validateStatusTransition(oModel.StatusPending, oModel.StatusReady), appErrors.ErrInvalidStatusTransition)
	assert.ErrorIs(t, s.validateStatusTransition(oModel.StatusPending, oModel.StatusDelivered), appErrors.ErrInvalidStatusTransition)
	assert.ErrorIs(t, s.validateStatusTransition(oModel.StatusPreparing, oModel.StatusDelivered), appErrors.ErrInvalidStatusTransition)
	assert.ErrorIs(t, s.validateStatusTransition(oModel.StatusReady, oModel.StatusPreparing), appErrors.ErrInvalidStatusTransition)
	assert.ErrorIs(t, s.validateStatusTransition(oModel.StatusDelivered, oModel.StatusPreparing), appErrors.ErrInvalidStatusTransition)
}

func TestValidateStatusTransition_CancelOnlyBeforeDelivered(t *testing.T) {
	s := &StatusUpdaterWithStock{}

	assert.NoError(t, s.validateStatusTransition(oModel.StatusPending, oModel.StatusCancelled))
	assert.NoError(t, s.validateStatusTransition(oModel.StatusPreparing, oModel.StatusCancelled))
	assert.NoError(t, s.validateStatusTransition(oModel.StatusReady, oModel.StatusCancelled))
	assert.ErrorIs(t, s.validateStatusTransition(oModel.StatusDelivered, oModel.StatusCancelled), appErrors.ErrOrderAlreadyDelivered)
}

func TestValidateStatusTransition_DeleteOnlyFromTerminal(t *testing.T) {
	s := &StatusUpdaterWithStock{}

	assert.ErrorIs(t, s.validateStatusTransition(oModel.StatusPending, oModel.StatusDeleted), appErrors.ErrInvalidStatusTransition)
	assert.ErrorIs(t, s.validateStatusTransition(oModel.StatusPreparing, oModel.StatusDeleted), appErrors.ErrInvalidStatusTransition)
	assert.ErrorIs(t, s.validateStatusTransition(oModel.StatusReady, oModel.StatusDeleted), appErrors.ErrInvalidStatusTransition)

	assert.NoError(t, s.validateStatusTransition(oModel.StatusCancelled, oModel.StatusDeleted))
	assert.NoError(t, s.validateStatusTransition(oModel.StatusExpired, oModel.StatusDeleted))
	assert.NoError(t, s.validateStatusTransition(oModel.StatusDelivered, oModel.StatusDeleted))
}

func TestValidateStatusTransition_TerminalLocks(t *testing.T) {
	s := &StatusUpdaterWithStock{}

	assert.ErrorIs(t, s.validateStatusTransition(oModel.StatusCancelled, oModel.StatusCancelled), appErrors.ErrOrderAlreadyCancelled)
	assert.ErrorIs(t, s.validateStatusTransition(oModel.StatusCancelled, oModel.StatusPreparing), appErrors.ErrInvalidStatusTransition)
	assert.ErrorIs(t, s.validateStatusTransition(oModel.StatusExpired, oModel.StatusPreparing), appErrors.ErrInvalidStatusTransition)
	assert.ErrorIs(t, s.validateStatusTransition(oModel.StatusDeleted, oModel.StatusPreparing), appErrors.ErrInvalidStatusTransition)
}
