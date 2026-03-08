package orders

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/radamesvaz/bakery-app/internal/logger"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
	orderRepo "github.com/radamesvaz/bakery-app/internal/repository/orders"
	productRepo "github.com/radamesvaz/bakery-app/internal/repository/products"
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
	TimeoutMinutes int // From env today; in multi-tenant will come from DB per tenant (see NewExpiredOrderCanceller doc).
}

// NewExpiredOrderCanceller builds an ExpiredOrderCanceller reading GHOST_ORDER_TIMEOUT_MINUTES from env (default 30).
//
// FUTURE (multi-tenant): The expiration timeout will not come from .env but from the database,
// e.g. a per-tenant config (tenant_config or similar). CancelExpiredOrders will then need to
// resolve timeout per tenant and process expired orders per tenant. Keep this in mind when
// implementing multi-tenant support to avoid losing traceability of this design decision.
func NewExpiredOrderCanceller(orderRepo *orderRepo.OrderRepository, productRepo *productRepo.ProductRepository) *ExpiredOrderCanceller {
	timeout := defaultGhostOrderTimeoutMinutes
	if v := os.Getenv("GHOST_ORDER_TIMEOUT_MINUTES"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			timeout = n
		}
	}
	return &ExpiredOrderCanceller{
		OrderRepo:      orderRepo,
		ProductRepo:    productRepo,
		TimeoutMinutes: timeout,
	}
}

// CancelExpiredOrders finds orders that are pending, unpaid, and older than the configured timeout,
// then cancels each one in a transaction (update status, revert stock, insert history).
func (c *ExpiredOrderCanceller) CancelExpiredOrders(ctx context.Context) (cancelled int, err error) {
	expirationTime := time.Now().Add(-time.Duration(c.TimeoutMinutes) * time.Minute)

	orders, err := c.OrderRepo.GetExpiredPendingOrders(ctx, expirationTime)
	if err != nil {
		return 0, fmt.Errorf("get expired pending orders: %w", err)
	}

	logger.Info().
		Int("count", len(orders)).
		Time("expiration_before", expirationTime).
		Int("timeout_minutes", c.TimeoutMinutes).
		Msg("CancelExpiredOrders: found expired pending orders")

	for _, order := range orders {
		if cancelErr := c.cancelOneOrder(ctx, order); cancelErr != nil {
			logger.Err(cancelErr).
				Uint64("order_id", order.ID).
				Msg("CancelExpiredOrders: failed to cancel order")
			// Continue with other orders; do not return
			continue
		}
		cancelled++
	}

	logger.Info().
		Int("cancelled", cancelled).
		Int("found", len(orders)).
		Msg("CancelExpiredOrders: finished")
	return cancelled, nil
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

func (c *ExpiredOrderCanceller) cancelOneOrder(ctx context.Context, order oModel.Order) error {
	tx, err := c.OrderRepo.DB.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	reason := systemCancellationReason
	if err := c.OrderRepo.UpdateOrderStatusTx(ctx, tx, order.ID, oModel.StatusCancelled, &reason); err != nil {
		return fmt.Errorf("update order status: %w", err)
	}

	items, err := c.OrderRepo.GetOrderItemsByOrderIDTx(ctx, tx, order.ID)
	if err != nil {
		return fmt.Errorf("get order items: %w", err)
	}
	for _, item := range items {
		if err := c.ProductRepo.RevertProductStockTx(ctx, tx, item.IdProduct, item.Quantity); err != nil {
			return fmt.Errorf("revert stock product %d: %w", item.IdProduct, err)
		}
	}

	orderHistory := oModel.OrderHistory{
		IDOrder:            order.ID,
		IdUser:             nil,
		Status:             oModel.StatusCancelled,
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

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit tx: %w", err)
	}
	return nil
}
