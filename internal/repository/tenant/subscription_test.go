package tenant

import (
	"context"
	"database/sql"
	"regexp"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRepository_GetBySlug_AllowsPendingSubscription(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT id, is_active FROM tenants
		 WHERE slug = $1
		   AND is_active = true
		   AND subscription_status IN ('active', 'pending')
		   AND (current_period_end IS NULL OR current_period_end > NOW())`)).
		WithArgs("acme").
		WillReturnRows(sqlmock.NewRows([]string{"id", "is_active"}).AddRow(5, true))

	id, isActive, err := repo.GetBySlug(context.Background(), "acme")
	require.NoError(t, err)
	assert.Equal(t, uint64(5), id)
	assert.True(t, isActive)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetSubscriptionSnapshot_ReturnsData(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	now := time.Now().UTC()
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT subscription_status, COALESCE(plan_code, ''), current_period_end
		 FROM tenants
		 WHERE id = $1`)).
		WithArgs(uint64(9)).
		WillReturnRows(sqlmock.NewRows([]string{"subscription_status", "plan_code", "current_period_end"}).AddRow("pending", "basic", now))

	snapshot, err := repo.GetSubscriptionSnapshot(context.Background(), 9)
	require.NoError(t, err)
	assert.Equal(t, "pending", snapshot.Status)
	assert.Equal(t, "basic", snapshot.PlanCode)
	assert.True(t, snapshot.CurrentPeriodEnd.Valid)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_MarkExpiredPendingAsCanceled_UsesGraceDays(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	now := time.Now().UTC()

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE tenants
		 SET subscription_status = 'canceled',
		     updated_on = NOW()
		 WHERE is_active = true
		   AND subscription_status = 'pending'
		   AND current_period_end IS NOT NULL
		   AND (current_period_end + make_interval(days => $2)) <= $1`)).
		WithArgs(now, 5).
		WillReturnResult(sqlmock.NewResult(0, 3))

	affected, err := repo.MarkExpiredPendingAsCanceled(context.Background(), now, 5)
	require.NoError(t, err)
	assert.Equal(t, int64(3), affected)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_UpdateTenantSubscription_UpdatesStatusAndPeriodEnd(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	periodEnd := time.Now().UTC().AddDate(0, 0, 30)

	mock.ExpectExec(regexp.QuoteMeta(`UPDATE tenants
		 SET subscription_status = $1,
		     current_period_end = CASE WHEN $4::boolean THEN $2 ELSE current_period_end END,
		     updated_on = NOW()
		 WHERE id = $3`)).
		WithArgs("active", sql.NullTime{Time: periodEnd, Valid: true}, uint64(3), true).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateTenantSubscription(context.Background(), 3, "active", sql.NullTime{Time: periodEnd, Valid: true}, true)
	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestRepository_GetSubscriptionSnapshot_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := &Repository{DB: db}
	mock.ExpectQuery(regexp.QuoteMeta(`SELECT subscription_status, COALESCE(plan_code, ''), current_period_end
		 FROM tenants
		 WHERE id = $1`)).
		WithArgs(uint64(88)).
		WillReturnError(sql.ErrNoRows)

	_, err = repo.GetSubscriptionSnapshot(context.Background(), 88)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "tenant not found")
	assert.NoError(t, mock.ExpectationsWereMet())
}
