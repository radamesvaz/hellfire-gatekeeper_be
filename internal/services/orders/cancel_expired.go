package orders

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/radamesvaz/bakery-app/internal/logger"
	orderRepo "github.com/radamesvaz/bakery-app/internal/repository/orders"
	productRepo "github.com/radamesvaz/bakery-app/internal/repository/products"
	tenantRepo "github.com/radamesvaz/bakery-app/internal/repository/tenant"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

const (
	defaultGhostOrderTimeoutMinutes = 30
	systemModifiedByID              = 0
	systemCancellationReason        = "Cancelación automática: tiempo de espera de pago agotado"
)

// ExpiredOrderCanceller cancels expired pending (unpaid) orders and reverts their stock.
type ExpiredOrderCanceller struct {
	OrderRepo      *orderRepo.OrderRepository
	ProductRepo    *productRepo.ProductRepository
	TenantRepo     *tenantRepo.Repository
	TimeoutMinutes int // From env today; in multi-tenant will come from DB per tenant (see NewExpiredOrderCanceller doc).
}

// NewExpiredOrderCanceller builds an ExpiredOrderCanceller reading GHOST_ORDER_TIMEOUT_MINUTES from env (default 30).
//
// FUTURE (multi-tenant): The expiration timeout will not come from .env but from the database,
// e.g. a per-tenant config (tenant_config or similar). CancelExpiredOrders will then need to
// resolve timeout per tenant and process expired orders per tenant. Keep this in mind when
// implementing multi-tenant support to avoid losing traceability of this design decision.
func NewExpiredOrderCanceller(orderRepo *orderRepo.OrderRepository, productRepo *productRepo.ProductRepository, tenantRepo *tenantRepo.Repository) *ExpiredOrderCanceller {
	timeout := defaultGhostOrderTimeoutMinutes
	if v := os.Getenv("GHOST_ORDER_TIMEOUT_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeout = n
		}
	}
	return &ExpiredOrderCanceller{
		OrderRepo:      orderRepo,
		ProductRepo:    productRepo,
		TenantRepo:     tenantRepo,
		TimeoutMinutes: timeout,
	}
}

// CancelExpiredOrders atomically claims expired pending orders (single UPDATE ... RETURNING), then
// in the same transaction reverts stock and inserts history for each. Safe for overlapping runs
// and multiple workers: only one can claim a given order.
func (c *ExpiredOrderCanceller) CancelExpiredOrders(ctx context.Context) (cancelled int, err error) {
	// NOTE: with expires_at persisted per order, the cron filters by expires_at < now().
	// The TimeoutMinutes here is still relevant as documentation/telemetry, but the
	// actual expiration is defined by the stored expires_at snapshot on each order.
	now := time.Now()
	reason := systemCancellationReason

	tenantIDs, err := c.TenantRepo.ListActiveTenantIDs(ctx)
	if err != nil {
		return 0, fmt.Errorf("list active tenants: %w", err)
	}
	if len(tenantIDs) == 0 {
		logger.Info().Msg("CancelExpiredOrders: no active tenants found, nothing to process")
		return 0, nil
	}

	totalCancelled := 0

	for _, tenantID := range tenantIDs {
		tx, beginErr := c.OrderRepo.DB.BeginTx(ctx, nil)
		if beginErr != nil {
			return totalCancelled, fmt.Errorf("begin tx for tenant %d: %w", tenantID, beginErr)
		}

		claimed, claimErr := c.OrderRepo.ClaimExpiredPendingOrdersTx(ctx, tx, tenantID, now, oModel.StatusExpired, &reason)
		if claimErr != nil {
			_ = tx.Rollback()
			return totalCancelled, fmt.Errorf("claim expired pending orders for tenant %d: %w", tenantID, claimErr)
		}

		logger.Info().
			Uint64("tenant_id", tenantID).
			Int("count", len(claimed)).
			Time("now", now).
			Int("timeout_minutes", c.TimeoutMinutes).
			Msg("CancelExpiredOrders: claimed expired pending orders (using expires_at < now())")

		for _, order := range claimed {
			if processErr := c.revertStockAndRecordHistoryTx(ctx, tx, order); processErr != nil {
				_ = tx.Rollback()
				return totalCancelled, fmt.Errorf("process claimed order %d for tenant %d: %w", order.ID, tenantID, processErr)
			}
		}

		if commitErr := tx.Commit(); commitErr != nil {
			return totalCancelled, fmt.Errorf("commit tx for tenant %d: %w", tenantID, commitErr)
		}

		totalCancelled += len(claimed)
	}

	logger.Info().
		Int("expired", totalCancelled).
		Int("found", totalCancelled).
		Msg("CancelExpiredOrders: finished (marked as expired) across tenants")
	return totalCancelled, nil
}

// RunGhostOrderWorker runs CancelExpiredOrders every intervalMinutes until ctx is cancelled.
// Logs at the start of each run; CancelExpiredOrders logs found/cancelled at the end.
func RunGhostOrderWorker(ctx context.Context, c *ExpiredOrderCanceller, intervalMinutes int) {
	if intervalMinutes <= 0 {
		intervalMinutes = 5
	}
	ticker := time.NewTicker(time.Duration(intervalMinutes) * time.Minute)
	defer ticker.Stop()

	logger.Info().
		Int("interval_minutes", intervalMinutes).
		Msg("Ghost order worker: started")

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("Ghost order worker: stopping")
			return
		case <-ticker.C:
			logger.Info().Msg("Ghost order job: starting run")
			cancelled, err := c.CancelExpiredOrders(ctx)
			if err != nil {
				logger.Err(err).Msg("Ghost order job: run failed")
			} else {
				logger.Info().Int("cancelled", cancelled).Msg("Ghost order job: run finished")
			}
		}
	}
}

// revertStockAndRecordHistoryTx reverts product stock for all items of the order and inserts
// one row into orders_history. The order must already be updated to cancelled (e.g. by ClaimExpiredPendingOrdersTx).
// Must be called within an existing transaction.
func (c *ExpiredOrderCanceller) revertStockAndRecordHistoryTx(ctx context.Context, tx *sql.Tx, order oModel.Order) error {
	items, err := c.OrderRepo.GetOrderItemsByOrderIDTx(ctx, tx, order.TenantID, order.ID)
	if err != nil {
		return fmt.Errorf("get order items: %w", err)
	}
	for _, item := range items {
		if err := c.ProductRepo.RevertProductStockTx(ctx, tx, order.TenantID, item.IdProduct, item.Quantity); err != nil {
			return fmt.Errorf("revert stock product %d: %w", item.IdProduct, err)
		}
	}

	reason := systemCancellationReason
	orderHistory := oModel.OrderHistory{
		TenantID:           order.TenantID,
		IDOrder:            order.ID,
		IdUser:             nil,
		Status:             oModel.StatusExpired,
		Price:              order.Price,
		Note:               order.Note,
		Paid:               order.Paid,
		CancellationReason: &reason,
		ModifiedBy:         systemModifiedByID,
		Action:             oModel.ActionUpdate,
	}
	if order.IdUser != 0 {
		orderHistory.IdUser = &order.IdUser
	}
	orderHistory.DeliveryDate = sql.NullTime{Time: order.DeliveryDate, Valid: !order.DeliveryDate.IsZero()}

	if err := c.OrderRepo.CreateOrderHistoryTx(ctx, tx, orderHistory); err != nil {
		return fmt.Errorf("create order history: %w", err)
	}
	return nil
}
