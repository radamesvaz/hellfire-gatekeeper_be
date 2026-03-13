package orders

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	orderRepo "github.com/radamesvaz/bakery-app/internal/repository/orders"
	productRepo "github.com/radamesvaz/bakery-app/internal/repository/products"
	tenantRepo "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpiredOrderCanceller_CancelExpiredOrders_NoExpiredOrders(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	// Single tenant, no expired orders: Begin, Claim (UPDATE ... RETURNING returns 0 rows), Commit
	const tenantID = uint64(1)
	tenantRepository := &tenantRepo.Repository{DB: db}

	mock.ExpectQuery("SELECT id FROM tenants").
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(tenantID))
	mock.ExpectBegin()
	mock.ExpectQuery("UPDATE orders").
		WithArgs("expired", "Cancelación automática: tiempo de espera de pago agotado", tenantID, sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{
			"id_order", "tenant_id", "id_user", "total_price", "status", "note", "created_on", "delivery_date", "paid", "cancellation_reason",
		}))
	mock.ExpectCommit()

	orderRepository := &orderRepo.OrderRepository{DB: db}
	productRepository := &productRepo.ProductRepository{DB: db}
	canceller := &ExpiredOrderCanceller{
		OrderRepo:      orderRepository,
		ProductRepo:    productRepository,
		TenantRepo:     tenantRepository,
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
	tenantRepository := &tenantRepo.Repository{}

	t.Setenv("GHOST_ORDER_TIMEOUT_MINUTES", "15")
	canceller := NewExpiredOrderCanceller(orderRepository, productRepository, tenantRepository)
	assert.Equal(t, 15, canceller.TimeoutMinutes, "timeout should be read from env when valid")
}

func TestNewExpiredOrderCanceller_DefaultTimeout(t *testing.T) {
	t.Setenv("GHOST_ORDER_TIMEOUT_MINUTES", "")
	orderRepository := &orderRepo.OrderRepository{}
	productRepository := &productRepo.ProductRepository{}
	tenantRepository := &tenantRepo.Repository{}
	canceller := NewExpiredOrderCanceller(orderRepository, productRepository, tenantRepository)
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
	tenantRepository := &tenantRepo.Repository{}

	t.Setenv("GHOST_ORDER_TIMEOUT_MINUTES", "invalid")
	canceller := NewExpiredOrderCanceller(orderRepository, productRepository, tenantRepository)
	assert.Equal(t, defaultGhostOrderTimeoutMinutes, canceller.TimeoutMinutes)

	t.Setenv("GHOST_ORDER_TIMEOUT_MINUTES", "0")
	canceller = NewExpiredOrderCanceller(orderRepository, productRepository, tenantRepository)
	assert.Equal(t, defaultGhostOrderTimeoutMinutes, canceller.TimeoutMinutes)

	os.Unsetenv("GHOST_ORDER_TIMEOUT_MINUTES")
}
