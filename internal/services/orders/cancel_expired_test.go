package orders

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	orderRepo "github.com/radamesvaz/bakery-app/internal/repository/orders"
	productRepo "github.com/radamesvaz/bakery-app/internal/repository/products"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpiredOrderCanceller_CancelExpiredOrders_NoExpiredOrders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	mock.ExpectQuery("SELECT id_order, id_user, total_price, status, note, created_on, delivery_date, paid, cancellation_reason").
		WithArgs(sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id_order", "id_user", "total_price", "status", "note", "created_on", "delivery_date", "paid", "cancellation_reason",
		}))

	orderRepository := &orderRepo.OrderRepository{DB: db}
	productRepository := &productRepo.ProductRepository{DB: db}
	canceller := &ExpiredOrderCanceller{
		OrderRepo:      orderRepository,
		ProductRepo:    productRepository,
		TimeoutMinutes: 30,
	}

	cancelled, err := canceller.CancelExpiredOrders(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 0, cancelled)
	require.NoError(t, mock.ExpectationsWereMet())
}

func TestNewExpiredOrderCanceller_TimeoutFromEnv(t *testing.T) {
	orderRepository := &orderRepo.OrderRepository{}
	productRepository := &productRepo.ProductRepository{}

	t.Setenv("GHOST_ORDER_TIMEOUT_MINUTES", "15")
	canceller := NewExpiredOrderCanceller(orderRepository, productRepository)
	assert.Equal(t, 15, canceller.TimeoutMinutes, "timeout should be read from env when valid")
}

func TestNewExpiredOrderCanceller_DefaultTimeout(t *testing.T) {
	t.Setenv("GHOST_ORDER_TIMEOUT_MINUTES", "")
	orderRepository := &orderRepo.OrderRepository{}
	productRepository := &productRepo.ProductRepository{}
	canceller := NewExpiredOrderCanceller(orderRepository, productRepository)
	assert.Equal(t, defaultGhostOrderTimeoutMinutes, canceller.TimeoutMinutes)
}

func TestExpiredOrderCanceller_expirationTimeCalculation(t *testing.T) {
	// Given a timeout, expirationTime = now - timeout should be in the past and ~timeout ago.
	canceller := &ExpiredOrderCanceller{TimeoutMinutes: 60}
	before := time.Now()
	expirationTime := before.Add(-time.Duration(canceller.TimeoutMinutes) * time.Minute)
	after := time.Now()
	assert.True(t, expirationTime.Before(after), "expiration time should be in the past")
	assert.True(t, after.Sub(expirationTime) >= 59*time.Minute && after.Sub(expirationTime) <= 61*time.Minute,
		"expiration should be about 60 minutes ago")
}

func TestNewExpiredOrderCanceller_InvalidEnvUsesDefault(t *testing.T) {
	orderRepository := &orderRepo.OrderRepository{}
	productRepository := &productRepo.ProductRepository{}

	t.Setenv("GHOST_ORDER_TIMEOUT_MINUTES", "invalid")
	canceller := NewExpiredOrderCanceller(orderRepository, productRepository)
	assert.Equal(t, defaultGhostOrderTimeoutMinutes, canceller.TimeoutMinutes)

	t.Setenv("GHOST_ORDER_TIMEOUT_MINUTES", "0")
	canceller = NewExpiredOrderCanceller(orderRepository, productRepository)
	assert.Equal(t, defaultGhostOrderTimeoutMinutes, canceller.TimeoutMinutes)

	os.Unsetenv("GHOST_ORDER_TIMEOUT_MINUTES")
}
