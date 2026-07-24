package orders

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/radamesvaz/bakery-app/internal/errors"
	"github.com/radamesvaz/bakery-app/internal/logger"
	oModel "github.com/radamesvaz/bakery-app/model/orders"
)

// OrderStatusRepository defines the interface for order status operations
type OrderStatusRepository interface {
	BeginTx(ctx context.Context) (*sql.Tx, error)
	GetOrderByID(ctx context.Context, tenantID, id uint64) (oModel.OrderResponse, error)
	UpdateOrderStatus(ctx context.Context, tenantID, orderID uint64, status oModel.OrderStatus, cancellationReason *string) error
	UpdateOrderStatusTx(ctx context.Context, tx *sql.Tx, tenantID, orderID uint64, status oModel.OrderStatus, cancellationReason *string) error
	UpdateOrderPaidStatusTx(ctx context.Context, tx *sql.Tx, tenantID, orderID uint64, paid bool) error
	CreateOrderHistory(ctx context.Context, order oModel.OrderHistory) error
	CreateOrderHistoryTx(ctx context.Context, tx *sql.Tx, order oModel.OrderHistory) error
	GetOrderItemsByOrderID(ctx context.Context, tenantID, orderID uint64) ([]oModel.OrderItems, error)
	GetOrderItemsByOrderIDTx(ctx context.Context, tx *sql.Tx, tenantID, orderID uint64) ([]oModel.OrderItems, error)
}

// ProductStockRepository defines the interface for product stock operations
type ProductStockRepository interface {
	RevertProductStock(ctx context.Context, tenantID, idProduct uint64, quantityToRevert uint64) error
	RevertProductStockTx(ctx context.Context, tx *sql.Tx, tenantID, idProduct uint64, quantityToRevert uint64) error
}

type StatusUpdaterWithStock struct {
	OrderRepo   OrderStatusRepository
	ProductRepo ProductStockRepository
}

func NewStatusUpdaterWithStock(orderRepo OrderStatusRepository, productRepo ProductStockRepository) *StatusUpdaterWithStock {
	return &StatusUpdaterWithStock{
		OrderRepo:   orderRepo,
		ProductRepo: productRepo,
	}
}

func (s *StatusUpdaterWithStock) validateStatusTransition(currentStatus, newStatus oModel.OrderStatus) error {
	if currentStatus == oModel.StatusCancelled {
		if newStatus == oModel.StatusCancelled {
			return errors.ErrOrderAlreadyCancelled
		}
		return errors.ErrInvalidStatusTransition
	}
	if currentStatus == oModel.StatusExpired {
		return errors.ErrInvalidStatusTransition
	}
	return nil
}

// UpdateOrderStatusWithStockReversion updates order status and reverts stock if admin cancels order.
// cancellationReason is optional; only used when newStatus is cancelled (e.g. user-provided reason or nil).
// paidOverride, when non-nil, updates paid in the same transaction as status and is used for history
// (combined status+paid PATCH stays atomic).
func (s *StatusUpdaterWithStock) UpdateOrderStatusWithStockReversion(ctx context.Context, tenantID, orderID uint64, newStatus oModel.OrderStatus, userID uint64, isAdmin bool, cancellationReason *string, paidOverride *bool) error {
	// Get the current order
	order, err := s.OrderRepo.GetOrderByID(ctx, tenantID, orderID)
	if err != nil {
		return err
	}

	// Validate status transition
	if err := s.validateStatusTransition(order.Status, newStatus); err != nil {
		return err
	}

	var effectiveCancellationReason *string
	if newStatus == oModel.StatusCancelled {
		effectiveCancellationReason = cancellationReason
	}

	paidForHistory := order.Paid
	if paidOverride != nil {
		paidForHistory = *paidOverride
	}

	needsStockRevert := isAdmin && newStatus == oModel.StatusCancelled
	needsPaidUpdate := paidOverride != nil

	// Use a transaction when status must stay atomic with stock reversion and/or paid.
	if needsStockRevert || needsPaidUpdate {
		return s.updateStatusInTx(ctx, tenantID, orderID, order, newStatus, userID, effectiveCancellationReason, paidOverride, paidForHistory, needsStockRevert)
	}

	err = s.OrderRepo.UpdateOrderStatus(ctx, tenantID, orderID, newStatus, effectiveCancellationReason)
	if err != nil {
		return fmt.Errorf("error updating order status: %w", err)
	}

	s.writeOrderHistoryBestEffort(ctx, tenantID, orderID, order, newStatus, userID, effectiveCancellationReason, paidForHistory)
	return nil
}

func (s *StatusUpdaterWithStock) updateStatusInTx(
	ctx context.Context,
	tenantID, orderID uint64,
	order oModel.OrderResponse,
	newStatus oModel.OrderStatus,
	userID uint64,
	cancellationReason *string,
	paidOverride *bool,
	paidForHistory bool,
	needsStockRevert bool,
) error {
	tx, err := s.OrderRepo.BeginTx(ctx)
	if err != nil {
		return fmt.Errorf("error beginning transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if err := s.OrderRepo.UpdateOrderStatusTx(ctx, tx, tenantID, orderID, newStatus, cancellationReason); err != nil {
		return fmt.Errorf("error updating order status: %w", err)
	}

	if paidOverride != nil {
		if err := s.OrderRepo.UpdateOrderPaidStatusTx(ctx, tx, tenantID, orderID, *paidOverride); err != nil {
			return fmt.Errorf("error updating order paid status: %w", err)
		}
	}

	if needsStockRevert {
		if err := s.revertOrderStockTx(ctx, tx, tenantID, orderID); err != nil {
			logger.Warn().Err(err).
				Uint64("order_id", orderID).
				Msg("Failed to revert stock for cancelled order")
			return fmt.Errorf("error reverting stock for cancelled order: %w", err)
		}
	}

	var orderHistoryIdUser *uint64
	if order.IdUser != 0 {
		orderHistoryIdUser = &order.IdUser
	}
	orderHistory := oModel.OrderHistory{
		TenantID: tenantID,
		IDOrder:  orderID,
		IdUser:   orderHistoryIdUser,
		Status:   newStatus,
		Price:    order.Price,
		Note:     order.Note,
		DeliveryDate: sql.NullTime{
			Time:  order.DeliveryDate,
			Valid: !order.DeliveryDate.IsZero(),
		},
		Paid:               paidForHistory,
		CancellationReason: cancellationReason,
		ModifiedBy:         userID,
		Action:             oModel.ActionUpdate,
	}
	if err := s.OrderRepo.CreateOrderHistoryTx(ctx, tx, orderHistory); err != nil {
		logger.Warn().Err(err).
			Uint64("order_id", orderID).
			Str("new_status", string(newStatus)).
			Msg("Failed to create order history")
		// History is best-effort; still commit status/paid/stock
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing transaction: %w", err)
	}
	return nil
}

func (s *StatusUpdaterWithStock) writeOrderHistoryBestEffort(
	ctx context.Context,
	tenantID, orderID uint64,
	order oModel.OrderResponse,
	newStatus oModel.OrderStatus,
	userID uint64,
	cancellationReason *string,
	paidForHistory bool,
) {
	var orderHistoryIdUser *uint64
	if order.IdUser != 0 {
		orderHistoryIdUser = &order.IdUser
	}
	orderHistory := oModel.OrderHistory{
		TenantID: tenantID,
		IDOrder:  orderID,
		IdUser:   orderHistoryIdUser,
		Status:   newStatus,
		Price:    order.Price,
		Note:     order.Note,
		DeliveryDate: sql.NullTime{
			Time:  order.DeliveryDate,
			Valid: !order.DeliveryDate.IsZero(),
		},
		Paid:               paidForHistory,
		CancellationReason: cancellationReason,
		ModifiedBy:         userID,
		Action:             oModel.ActionUpdate,
	}

	if err := s.OrderRepo.CreateOrderHistory(ctx, orderHistory); err != nil {
		logger.Warn().Err(err).
			Uint64("order_id", orderID).
			Str("new_status", string(newStatus)).
			Msg("Failed to create order history")
	}
}

// revertOrderStockTx reverts the stock for all items in an order within a transaction.
// RevertProductStockTx is expected to no-op when track_inventory is false.
func (s *StatusUpdaterWithStock) revertOrderStockTx(ctx context.Context, tx *sql.Tx, tenantID, orderID uint64) error {
	items, err := s.OrderRepo.GetOrderItemsByOrderIDTx(ctx, tx, tenantID, orderID)
	if err != nil {
		return fmt.Errorf("error getting order items: %w", err)
	}

	for _, item := range items {
		err = s.ProductRepo.RevertProductStockTx(ctx, tx, tenantID, item.IdProduct, item.Quantity)
		if err != nil {
			return fmt.Errorf("error reverting stock for product %d: %w", item.IdProduct, err)
		}
	}

	return nil
}
