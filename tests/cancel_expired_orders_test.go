// Integration tests for CancelExpiredOrders (ghost orders).
// Rollback: when RevertProductStockTx or CreateOrderHistoryTx fails, cancelOneOrder uses defer tx.Rollback(),
// so the order is not left half-updated. That path is not asserted here; it is ensured by the implementation.
package tests

import (
	"context"
	"testing"
	"time"

	ordersRepo "github.com/radamesvaz/bakery-app/internal/repository/orders"
	"github.com/radamesvaz/bakery-app/internal/repository/products"
	orderService "github.com/radamesvaz/bakery-app/internal/services/orders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelExpiredOrders_Integration(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	ctx := context.Background()

	// Remove seed pending/unpaid orders so only our test orders exist
	_, err := db.ExecContext(ctx, `DELETE FROM order_items WHERE id_order IN (SELECT id_order FROM orders WHERE status = 'pending' AND paid = false)`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `DELETE FROM orders WHERE status = 'pending' AND paid = false`)
	require.NoError(t, err)

	ghostCreated := time.Now().Add(-2 * time.Hour)
	recentCreated := time.Now().Add(-5 * time.Minute)
	deliveryDate := time.Now().AddDate(0, 0, 7)

	_, err = db.ExecContext(ctx, `INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, paid)
		VALUES (1, 2, 'pending', 20.0, 'ghost order', $1, $2, false)`, ghostCreated, deliveryDate)
	require.NoError(t, err)

	var ghostOrderID int
	err = db.QueryRowContext(ctx, `SELECT id_order FROM orders WHERE note = 'ghost order'`).Scan(&ghostOrderID)
	require.NoError(t, err)

	// Set expires_at explicitly for the ghost order so that the cron (which filters by expires_at < now())
	// will pick it up in this integration test. In production, expires_at is set when creating the order.
	_, err = db.ExecContext(ctx, `UPDATE orders SET expires_at = $1 WHERE id_order = $2`, time.Now().Add(-time.Minute), ghostOrderID)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO order_items (tenant_id, id_order, id_product, quantity) VALUES (1, $1, 1, 2), (1, $1, 2, 1)`, ghostOrderID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE products SET stock = stock - 2 WHERE id_product = 1`)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE products SET stock = stock - 1 WHERE id_product = 2`)
	require.NoError(t, err)

	_, err = db.ExecContext(ctx, `INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, paid)
		VALUES (1, 2, 'pending', 10.0, 'recent order', $1, $2, false)`, recentCreated, deliveryDate)
	require.NoError(t, err)

	var recentOrderID int
	err = db.QueryRowContext(ctx, `SELECT id_order FROM orders WHERE note = 'recent order'`).Scan(&recentOrderID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `INSERT INTO order_items (tenant_id, id_order, id_product, quantity) VALUES (1, $1, 1, 1)`, recentOrderID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE products SET stock = stock - 1 WHERE id_product = 1`)
	require.NoError(t, err)

	orderRepo := &ordersRepo.OrderRepository{DB: db}
	productRepo := &products.ProductRepository{DB: db}
	canceller := &orderService.ExpiredOrderCanceller{
		OrderRepo:      orderRepo,
		ProductRepo:    productRepo,
		TimeoutMinutes: 30,
	}

	cancelled, err := canceller.CancelExpiredOrders(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, cancelled, "should cancel exactly one ghost order")

	var ghostStatus, ghostReason string
	err = db.QueryRowContext(ctx, `SELECT status, COALESCE(cancellation_reason, '') FROM orders WHERE id_order = $1`, ghostOrderID).Scan(&ghostStatus, &ghostReason)
	require.NoError(t, err)
	assert.Equal(t, "expired", ghostStatus)
	assert.Contains(t, ghostReason, "Cancelación automática", "cancellation_reason should contain system message")

	var recentStatus string
	err = db.QueryRowContext(ctx, `SELECT status FROM orders WHERE id_order = $1`, recentOrderID).Scan(&recentStatus)
	require.NoError(t, err)
	assert.Equal(t, "pending", recentStatus)

	var stock1, stock2 int
	err = db.QueryRowContext(ctx, `SELECT stock FROM products WHERE id_product = 1`).Scan(&stock1)
	require.NoError(t, err)
	err = db.QueryRowContext(ctx, `SELECT stock FROM products WHERE id_product = 2`).Scan(&stock2)
	require.NoError(t, err)
	assert.Equal(t, 5, stock1, "product 1 stock after reverting ghost (2 units)")
	assert.Equal(t, 2, stock2, "product 2 stock reverted to 2")

	var count int
	err = db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM orders_history WHERE id_order = $1 AND modified_by = 0 AND cancellation_reason LIKE '%Cancelación automática%'`,
		ghostOrderID).Scan(&count)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, count, 1, "orders_history should have record for cancelled order with system modified_by and reason")
}

func TestCancelExpiredOrders_Integration_PaidOrderNotCancelled(t *testing.T) {
	_, db, terminate, dsn := setupPostgreSQLContainer(t)
	defer terminate()

	runMigrations(t, dsn)

	ctx := context.Background()
	// Remove seed pending/unpaid so only our "old but paid" order exists
	_, _ = db.ExecContext(ctx, `DELETE FROM order_items WHERE id_order IN (SELECT id_order FROM orders WHERE status = 'pending' AND paid = false)`)
	_, _ = db.ExecContext(ctx, `DELETE FROM orders WHERE status = 'pending' AND paid = false`)

	oldCreated := time.Now().Add(-2 * time.Hour)
	deliveryDate := time.Now().AddDate(0, 0, 7)

	_, err := db.ExecContext(ctx, `INSERT INTO orders (tenant_id, id_user, status, total_price, note, created_on, delivery_date, paid)
		VALUES (1, 2, 'pending', 15.0, 'old but paid', $1, $2, true)`, oldCreated, deliveryDate)
	require.NoError(t, err)

	var orderID int
	err = db.QueryRowContext(ctx, `SELECT id_order FROM orders WHERE note = 'old but paid'`).Scan(&orderID)
	require.NoError(t, err)

	orderRepo := &ordersRepo.OrderRepository{DB: db}
	productRepo := &products.ProductRepository{DB: db}
	canceller := &orderService.ExpiredOrderCanceller{
		OrderRepo:      orderRepo,
		ProductRepo:    productRepo,
		TimeoutMinutes: 30,
	}

	cancelled, err := canceller.CancelExpiredOrders(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, cancelled)

	var status string
	err = db.QueryRowContext(ctx, `SELECT status FROM orders WHERE id_order = $1`, orderID).Scan(&status)
	require.NoError(t, err)
	assert.Equal(t, "pending", status)
}
